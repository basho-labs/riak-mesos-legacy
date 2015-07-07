package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"os"
	"time"
)

type RiakNode struct {
	executor *ExecutorCore
	taskInfo *mesos.TaskInfo
}

func NewRiakNode(taskInfo *mesos.TaskInfo, executor *ExecutorCore) *RiakNode {
	return &RiakNode{
		executor: executor,
		taskInfo: taskInfo,
	}
}
func (riakNode *RiakNode) Loop() {
	log.Info("Other hilarious facts: ", riakNode.taskInfo)
	runStatus := &mesos.TaskStatus{
		TaskId: riakNode.taskInfo.GetTaskId(),
		State:  mesos.TaskState_TASK_RUNNING.Enum(),
	}
	_, err := riakNode.executor.Driver.SendStatusUpdate(runStatus)
	fmt.Println("Sent starting status update: ", err)

	if err != nil {
		log.Panic("Got error", err)
	}
	time.Sleep(1000 * time.Second)
	finStatus := &mesos.TaskStatus{
		TaskId: riakNode.taskInfo.GetTaskId(),
		State:  mesos.TaskState_TASK_FINISHED.Enum(),
	}
	_, err = riakNode.executor.Driver.SendStatusUpdate(finStatus)
	fmt.Println("Sent starting finish update: ", err)

	if err != nil {
		log.Panic("Got error", err)

	}
	time.Sleep(10 * time.Second)
	riakNode.executor.Driver.Stop()
	time.Sleep(10 * time.Second)

	os.Exit(0)

}
