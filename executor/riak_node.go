package main

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"bytes"
	"errors"
	log "github.com/Sirupsen/logrus"
	mesos "github.com/basho-labs/mesos-go/mesosproto"
	"github.com/basho-labs/riak-mesos/cepmd/cepm"
	"github.com/basho-labs/riak-mesos/common"
	metamgr "github.com/basho-labs/riak-mesos/metadata_manager"
	"github.com/basho-labs/riak-mesos/process_manager"
	"net/http"
)

type RiakNode struct {
	executor        *ExecutorCore
	taskInfo        *mesos.TaskInfo
	generation      uint64
	finishChan      chan interface{}
	running         bool
	metadataManager *metamgr.MetadataManager
	taskData        common.TaskData
	pm              *process_manager.ProcessManager
}

type templateData struct {
	HTTPPort               int64
	PBPort                 int64
	HandoffPort            int64
	FullyQualifiedNodeName string
	DisterlPort            int64
}

type advancedTemplateData struct {
	CEPMDPort int
}

func NewRiakNode(taskInfo *mesos.TaskInfo, executor *ExecutorCore) *RiakNode {
	taskData, err := common.DeserializeTaskData(taskInfo.Data)
	if err != nil {
		log.Panic("Got error", err)
	}

	log.Infof("Deserialized task data: %+v", taskData)
	mgr := metamgr.NewMetadataManager(taskData.FrameworkName, taskData.Zookeepers)

	return &RiakNode{

		executor:        executor,
		taskInfo:        taskInfo,
		running:         false,
		metadataManager: mgr,
		taskData:        taskData,
	}
}

func (riakNode *RiakNode) runLoop(child *metamgr.ZkNode) {

	var runStatus *mesos.TaskStatus
	var err error

	// runStatus := &mesos.TaskStatus{
	// 	TaskId: riakNode.taskInfo.GetTaskId(),
	// 	State:  mesos.TaskState_TASK_RUNNING.Enum(),
	// }
	// _, err := riakNode.executor.Driver.SendStatusUpdate(runStatus)
	// if err != nil {
	// 	log.Panic("Got error", err)
	// }

	waitChan := riakNode.pm.Listen()
	select {
	case <-waitChan:
		{
			log.Info("Riak Died, failing")
			// Just in case, cleanup
			// This means the node died :(
			runStatus = &mesos.TaskStatus{
				TaskId: riakNode.taskInfo.GetTaskId(),
				State:  mesos.TaskState_TASK_FAILED.Enum(),
			}
			_, err = riakNode.executor.Driver.SendStatusUpdate(runStatus)
			if err != nil {
				log.Panic("Got error", err)
			}
		}
	case <-riakNode.finishChan:
		{
			log.Info("Finish channel says to shut down Riak")
			riakNode.pm.TearDown()
			runStatus = &mesos.TaskStatus{
				TaskId: riakNode.taskInfo.GetTaskId(),
				State:  mesos.TaskState_TASK_FINISHED.Enum(),
			}
			_, err = riakNode.executor.Driver.SendStatusUpdate(runStatus)
			if err != nil {
				log.Panic("Got error", err)
			}
		}
	}
	child.Delete()
	time.Sleep(15 * time.Second)
	log.Info("Shutting down")
	riakNode.executor.Driver.Stop()

}
func (riakNode *RiakNode) configureRiak(taskData common.TaskData) templateData {

	fetchURI := fmt.Sprintf("%s/api/v1/clusters/%s/config", riakNode.taskData.URI, riakNode.taskData.ClusterName)
	resp, err := http.Get(fetchURI)
	if err != nil {
		log.Error("Unable to fetch config: ", err)
	}
	config, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Error("Unable to fetch config: ", err)
	}

	tmpl, err := template.New("config").Parse(string(config))

	if err != nil {
		log.Panic(err)
	}

	// Populate template data from the MesosTask
	vars := templateData{}
	vars.FullyQualifiedNodeName = riakNode.taskData.FullyQualifiedNodeName

	vars.HTTPPort = taskData.HTTPPort
	vars.PBPort = taskData.PBPort
	vars.HandoffPort = taskData.HandoffPort
	vars.DisterlPort = taskData.DisterlPort

	file, err := os.OpenFile("root/riak/etc/riak.conf", os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0664)

	defer file.Close()
	if err != nil {
		log.Panic("Unable to open file: ", err)
	}

	err = tmpl.Execute(file, vars)

	if err != nil {
		log.Panic("Got error", err)
	}
	return vars
}
func (riakNode *RiakNode) configureAdvanced(cepmdPort int) {

	fetchURI := fmt.Sprintf("%s/api/v1/clusters/%s/advancedConfig", riakNode.taskData.URI, riakNode.taskData.ClusterName)
	resp, err := http.Get(fetchURI)
	if err != nil {
		log.Error("Unable to fetch advanced config: ", err)
	}
	advancedConfig, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Error("Unable to fetch advanced config: ", err)
	}

	tmpl, err := template.New("advanced").Parse(string(advancedConfig))

	if err != nil {
		log.Panic(err)
	}

	// Populate template data from the MesosTask
	vars := advancedTemplateData{}
	vars.CEPMDPort = cepmdPort
	file, err := os.OpenFile("root/riak/etc/advanced.config", os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0664)

	defer file.Close()
	if err != nil {
		log.Panic("Unable to open file: ", err)
	}

	err = tmpl.Execute(file, vars)

	if err != nil {
		log.Panic("Got error", err)
	}
}

func (riakNode *RiakNode) setCoordinatedData(config templateData) *metamgr.ZkNode {
	rootNode := riakNode.metadataManager.GetRootNode()

	rootNode.CreateChildIfNotExists("coordinator")
	coordinator, err := rootNode.GetChild("coordinator")
	if err != nil {
		log.Panic(err)
	}
	coordinator.CreateChildIfNotExists("coordinatedNodes")
	coordinatedNodes, err := coordinator.GetChild("coordinatedNodes")
	if err != nil {
		log.Panic(err)
	}

	child, err := coordinatedNodes.MakeChild(riakNode.taskInfo.GetTaskId().GetValue(), true)
	if err != nil {
		log.Panic(err)
	}
	coordinatedData := common.CoordinatedData{
		NodeName:      riakNode.taskData.FullyQualifiedNodeName,
		DisterlPort:   int(config.DisterlPort),
		PBPort:        int(config.PBPort),
		HTTPPort:      int(config.HTTPPort),
		Hostname:      riakNode.taskData.Host,
		ClusterName:   riakNode.taskData.ClusterName,
		FrameworkName: riakNode.taskData.FrameworkName,
	}
	cdBytes, err := coordinatedData.Serialize()
	if err != nil {
		log.Panic("Could not serialize coordinated data	", err)
	}
	child.SetData(cdBytes)
	return child
}

func (riakNode *RiakNode) Run() {
	var err error
	var fetchURI string
	var resp *http.Response

	os.Mkdir("root", 0777)

	riakNode.decompress()

	fetchURI = fmt.Sprintf("%s/static2/riak-bin.tar.gz", riakNode.taskData.URI)
	log.Info("Preparing to fetch riak from: ", fetchURI)
	resp, err = http.Get(fetchURI)
	if err != nil {
		log.Panic("Unable to fetch riak root: ", err)
	}
	err = common.ExtractGZ("root", resp.Body)
	if err != nil {
		log.Panic("Unable to extract riak root: ", err)
	}

	config := riakNode.configureRiak(riakNode.taskData)

	c := cepm.NewCPMd(0, riakNode.metadataManager)
	c.Background()
	riakNode.configureAdvanced(c.GetPort())

	args := []string{"console", "-noinput"}

	kernelDirs, err := filepath.Glob("root/riak/lib/kernel*")
	if err != nil {
		log.Fatal("Could not find kernel directory")
	}

	log.Infof("Found kernel dirs: %v", kernelDirs)

	err = cepm.InstallInto(fmt.Sprint(kernelDirs[0], "/ebin"))
	if err != nil {
		log.Panic(err)
	}
	if err := common.KillEPMD("root/riak"); err != nil {
		log.Fatal("Could not kill EPMd: ", err)
	}
	args = append(args, "-no_epmd")
	os.MkdirAll(fmt.Sprint(kernelDirs[0], "/priv"), 0777)
	ioutil.WriteFile(fmt.Sprint(kernelDirs[0], "/priv/cepmd_port"), []byte(fmt.Sprintf("%d.", c.GetPort())), 0777)

	HealthCheckFun := func() error {
		log.Info("Checking is Riak is started")
		data, err := ioutil.ReadFile("root/riak/log/console.log")
		if err != nil {
			if bytes.Contains(data, []byte("Wait complete for service riak_kv")) {
				log.Info("Riak started, waiting 10 seconds to avoid race conditions (HACK)")
				time.Sleep(10 * time.Second)
				return nil
			}
			return errors.New("Riak KV not yet started")
		}
		return err
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Panic("Could not get wd: ", err)
	}
	chroot := filepath.Join(wd, "root")
	riakNode.pm, err = process_manager.NewProcessManager(func() { return }, "/riak/bin/riak", args, HealthCheckFun, &chroot, riakNode.taskData.UseSuperChroot)

	if err != nil {
		log.Error("Could not start Riak: ", err)

		runStatus := &mesos.TaskStatus{
			TaskId: riakNode.taskInfo.GetTaskId(),
			State:  mesos.TaskState_TASK_FAILED.Enum(),
		}
		_, err = riakNode.executor.Driver.SendStatusUpdate(runStatus)
		if err != nil {
			log.Panic("Got error", err)
		}
		// Shutdown:
		time.Sleep(15 * time.Minute)
		log.Info("Shutting down due to GC, after failing to bring up Riak node")
		riakNode.executor.Driver.Stop()
	} else {
		child := riakNode.setCoordinatedData(config)

		rexPort := riakNode.taskData.HTTPPort
		tsd := common.TaskStatusData{
			RexPort: rexPort,
		}
		tsdBytes, err := tsd.Serialize()

		if err != nil {
			log.Panic("Could not serialize Riak Explorer data", err)
		}

		runStatus := &mesos.TaskStatus{
			TaskId: riakNode.taskInfo.GetTaskId(),
			State:  mesos.TaskState_TASK_RUNNING.Enum(),
			Data:   tsdBytes,
		}
		_, err = riakNode.executor.Driver.SendStatusUpdate(runStatus)
		if err != nil {
			log.Panic("Got error", err)
		}
		riakNode.running = true
		go riakNode.runLoop(child)
	}
}

func (riakNode *RiakNode) next() {
	riakNode.executor.lock.Lock()
	defer riakNode.executor.lock.Unlock()
	bin := make([]byte, 4)
	binary.PutUvarint(bin, riakNode.generation)
	runStatus := &mesos.TaskStatus{
		TaskId: riakNode.taskInfo.GetTaskId(),
		State:  mesos.TaskState_TASK_RUNNING.Enum(),
		Data:   bin,
	}
	_, err := riakNode.executor.Driver.SendStatusUpdate(runStatus)
	if err != nil {
		log.Panic("Got error", err)
	}
	riakNode.generation = riakNode.generation + 1
}

func (riakNode *RiakNode) finish() {
	riakNode.finishChan <- nil
}

func (riakNode *RiakNode) ForceFinish() {

}
