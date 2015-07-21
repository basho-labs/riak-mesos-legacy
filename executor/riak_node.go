package main

import (
	"encoding/binary"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/basho-labs/riak-mesos/common"
	metamgr "github.com/basho-labs/riak-mesos/metadata_manager"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
	"time"
	rex "github.com/basho-labs/riak-mesos/riak_explorer"
)

type RiakNode struct {
	executor        *ExecutorCore
	taskInfo        *mesos.TaskInfo
	generation      uint64
	finishChan      chan interface{}
	waitChan        chan interface{}
	running         bool
	metadataManager *metamgr.MetadataManager
	taskData        common.TaskData
	rex				*rex.RiakExplorer
}

type templateData struct {
	HTTPPort               int64
	PBPort                 int64
	HandoffPort            int64
	FullyQualifiedNodeName string
}

func NewRiakNode(taskInfo *mesos.TaskInfo, executor *ExecutorCore) *RiakNode {
	taskData, err := common.DeserializeTaskData(taskInfo.Data)
	if err != nil {
		log.Panic("Got error", err)
	}

	log.Infof("Deserialized task data: %+v", taskData)
	mgr := metamgr.NewMetadataManager(executor.fwInfo.GetId().GetValue(), taskData.Zookeepers)
	return &RiakNode{
		executor:        executor,
		taskInfo:        taskInfo,
		finishChan:      make(chan interface{}, 1),
		waitChan:        make(chan interface{}, 1),
		running:         false,
		metadataManager: mgr,
		taskData:        taskData,
	}
}

func portIter(resources []*mesos.Resource) chan int64 {
	ports := make(chan int64)
	go func() {
		defer close(ports)
		for _, resource := range util.FilterResources(resources, func(res *mesos.Resource) bool { return res.GetName() == "ports" }) {
			for _, port := range common.RangesToArray(resource.GetRanges().GetRange()) {
				ports <- port
			}
		}
	}()
	return ports
}
func (riakNode *RiakNode) shutdownRiak() error {
	process := exec.Command("riak/bin/riak", "stop")
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("Could not get current working directory")
	}
	home := filepath.Join(wd, "riak/data")
	homevar := fmt.Sprintf("HOME=%s", home)
	process.Env = append(os.Environ(), homevar)
	return process.Run()
}
func (riakNode *RiakNode) waitLoop() {
	// I guess I can send an unneccessary ping, but eh
	for riakNode.running == true {
		process := exec.Command("riak/bin/riak", "ping")
		process.Stdout = os.Stdout
		process.Stderr = os.Stderr
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal("Could not get current working directory")
		}
		home := filepath.Join(wd, "riak/data")
		homevar := fmt.Sprintf("HOME=%s", home)
		process.Env = append(os.Environ(), homevar)
		err = process.Run()
		if err != nil {
			log.Info("Error pinging Riak: ", err)
			riakNode.waitChan <- nil
			break
		}
		<-time.After(10 * time.Second)
	}
}
func (riakNode *RiakNode) runLoop() {

	runStatus := &mesos.TaskStatus{
		TaskId: riakNode.taskInfo.GetTaskId(),
		State:  mesos.TaskState_TASK_RUNNING.Enum(),
	}
	_, err := riakNode.executor.Driver.SendStatusUpdate(runStatus)
	if err != nil {
		log.Panic("Got error", err)
	}

	select {
	case <-riakNode.waitChan:
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
			riakNode.shutdownRiak()
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
	time.Sleep(15 * time.Minute)
	log.Info("Shutting down")
	riakNode.executor.Driver.Stop()

}
func (riakNode *RiakNode) configureRiak(ports chan int64) {

	data, err := Asset("data/riak.conf")
	if err != nil {
		log.Panic("Got error", err)
	}
	tmpl, err := template.New("test").Parse(string(data))

	if err != nil {
		log.Panic(err)
	}

	// Populate template data from the MesosTask
	vars := templateData{}
	vars.FullyQualifiedNodeName = riakNode.taskData.FullyQualifiedNodeName


	vars.HTTPPort = <-ports
	vars.PBPort = <-ports
	vars.HandoffPort = <-ports

	file, err := os.OpenFile("riak/etc/riak.conf", os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0)

	if err != nil {
		log.Panic("Unable to open file: ", err)
	}

	err = tmpl.Execute(file, vars)

	if err != nil {
		log.Panic("Got error", err)
	}
}
func (riakNode *RiakNode) Run() {

	var err error
	log.Info("Other hilarious facts: ", riakNode.taskInfo)

	ports := portIter(riakNode.taskInfo.Resources)
	riakNode.configureRiak(ports)
	rexPort := <-ports
	log.Info("RexPort (before sleeping): ", rexPort)

	riakNode.rex, err = rex.NewRiakExplorer(rexPort, riakNode.taskData.RexFullyQualifiedNodeName)

	if err != nil {
		log.Fatal("Could not start up Riak Explorer")
	}

	process := exec.Command("riak/bin/riak", "start")
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("Could not get current working directory")
	}
	home := filepath.Join(wd, "riak/data")
	homevar := fmt.Sprintf("HOME=%s", home)
	process.Env = append(os.Environ(), homevar)
	err = process.Run()

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
		rootNode := riakNode.metadataManager.GetRootNode()

		clustersNode := rootNode.GetChild("clusters")
		clusterNode := clustersNode.GetChild(riakNode.taskData.ClusterName)

		clusterNode.CreateChildIfNotExists("coordinator")
		coordinator := clusterNode.GetChild("coordinator")
		coordinator.CreateChildIfNotExists("coordinatedNodes")
		coordinatedNodes := coordinator.GetChild("coordinatedNodes")

		lock := coordinator.GetLock()
		lock.Lock()
		// Do cluster joiny stuff
		children := coordinatedNodes.GetChildren()
		log.Info("Coordinator Children: ", children)
		for _, child := range children {
			cData, err := common.DeserializeCoordinatedData(child.GetData())
			if err != nil {
				log.Panicf("Unable to deserialize Coordinated data for child %v: %v", child, err)
			}
			reply, _ := riakNode.rex.NewRiakExplorerClient().Join(riakNode.taskData.FullyQualifiedNodeName, cData.NodeName)
			log.Info("Join: ", reply)
		}
		child := coordinatedNodes.MakeChild(riakNode.taskInfo.GetTaskId().GetValue(), true)
		coordinatedData := common.CoordinatedData{NodeName:riakNode.taskData.FullyQualifiedNodeName}
		cdBytes, err := coordinatedData.Serialize()
		if err != nil {
			log.Panic("Could not serialize coordinated data	", err)
		}
		child.SetData(cdBytes)
		lock.Unlock()
		runStatus := &mesos.TaskStatus{
			TaskId: riakNode.taskInfo.GetTaskId(),
			State:  mesos.TaskState_TASK_RUNNING.Enum(),
		}
		_, err = riakNode.executor.Driver.SendStatusUpdate(runStatus)
		if err != nil {
			log.Panic("Got error", err)
		}
		riakNode.running = true
		go riakNode.runLoop()
		go riakNode.waitLoop()

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
