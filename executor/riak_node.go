package main

import (
	log "github.com/Sirupsen/logrus"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"os"
	"text/template"
	util "github.com/mesos/mesos-go/mesosutil"
	"github.com/basho-labs/riak-mesos/common"
)

type RiakNode struct {
	executor *ExecutorCore
	taskInfo *mesos.TaskInfo
}

type templateData struct {
	HTTPPort int64
	PBPort   int64
	FullyQualifiedNodeName string
}

func NewRiakNode(taskInfo *mesos.TaskInfo, executor *ExecutorCore) *RiakNode {
	return &RiakNode{
		executor: executor,
		taskInfo: taskInfo,
	}
}
func (riakNode *RiakNode) Run() {

	var err error
	log.Info("Other hilarious facts: ", riakNode.taskInfo)

	taskData, err := common.DeserializeTaskData(riakNode.taskInfo.Data)
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
	vars.FullyQualifiedNodeName = taskData.FullyQualifiedNodeName




	ports := make(chan int64)
	go func() {
		defer close(ports)
		for _, resource := range util.FilterResources(riakNode.taskInfo.Resources, func(res *mesos.Resource) bool { return res.GetName() == "ports" }) {
			for _, port := range common.RangesToArray(resource.GetRanges().GetRange()) {
				ports <- port
			}
		}
	}()

	vars.HTTPPort = <-ports
	vars.PBPort = <-ports

	file, err := os.OpenFile("riak/etc/riak.conf", os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0)

	if err != nil {
		log.Panic("Unable to open file: ", err)
	}

	err = tmpl.Execute(file, vars)

	if err != nil {
		log.Panic("Got error", err)
	}


	runStatus := &mesos.TaskStatus{
		TaskId: riakNode.taskInfo.GetTaskId(),
		State:  mesos.TaskState_TASK_RUNNING.Enum(),
	}
	_, err = riakNode.executor.Driver.SendStatusUpdate(runStatus)

	if err != nil {
		log.Panic("Got error", err)
	}
}

func (riakNode *RiakNode) finish() {
	riakNode.executor.lock.Lock()
	defer riakNode.executor.lock.Unlock()
	runStatus := &mesos.TaskStatus{
		TaskId: riakNode.taskInfo.GetTaskId(),
		State:  mesos.TaskState_TASK_FINISHED.Enum(),
	}
	_, err := riakNode.executor.Driver.SendStatusUpdate(runStatus)
	if err != nil {
		log.Panic("Got error", err)
	}
	riakNode.executor.riakNode = nil
}
