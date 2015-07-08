package main

import (
	log "github.com/Sirupsen/logrus"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"os"
	"text/template"
)

type RiakNode struct {
	executor *ExecutorCore
	taskInfo *mesos.TaskInfo
}

type templateData struct {
	HTTPPort int
	PBPort   int
	NodeName string
	HostName string
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
	_ = os.Stdout
	_ = vars
	_ = tmpl
	//err = tmpl.Execute(os.Stdout, vars)

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
