package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/basho/bletchley/metadata_manager"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"github.com/golang/protobuf/proto"
	util "github.com/mesos/mesos-go/mesosutil"
	"time"
//	"github.com/satori/go.uuid"

)

type targetTaskStateType int

const (
	unknownTargetTaskState targetTaskStateType = iota
)

type TargetTask struct {
	schedulerCore          *SchedulerCore
	TaskName               string
	uuid             	   string
	mesosTaskStatusUpdates chan *mesos.TaskStatus
	targetTaskState        targetTaskStateType
	mgr                    *metadata_manager.MetadataManager
}

func NewTargetTask(taskName string, schedulerCore *SchedulerCore, mgr *metadata_manager.MetadataManager) *TargetTask {
	uuid := mgr.GetTaskUUID(taskName)
	return &TargetTask{schedulerCore: schedulerCore,
		TaskName:               taskName,
		mesosTaskStatusUpdates: make(chan *mesos.TaskStatus),
		// TODO: Add lower generation marker
		uuid:        uuid,
		mgr:               mgr,
	}
}

func (task *TargetTask) currentTaskName() string {
	return fmt.Sprintf("%s-%s", task.TaskName, task.uuid)
}
func (task *TargetTask) UpdateStatus(status *mesos.TaskStatus) {
	task.mesosTaskStatusUpdates <- status
}
func (task *TargetTask) subscribe() {
	mesosTaskName := task.currentTaskName()
	task.schedulerCore.Subscribe(mesosTaskName, task)
}

func (task *TargetTask) Loop() {
	defer close(task.mesosTaskStatusUpdates)
	log.Info("Starting task: ", task.TaskName)
	task.subscribe()
	for {
		select {
		case statusUpdate := <-task.mesosTaskStatusUpdates:
			task.handleStatusUpdate(statusUpdate)
		}
	}

}

func (task *TargetTask) reviveTask() {
	log.Info("Bringing her back")
	// Time to bring 'er back.
	task.uuid = task.mgr.SetTaskUUID(task.TaskName, task.uuid)
	executorUris := []*mesos.CommandInfo_URI{}
	executorUris = append(executorUris,
		&mesos.CommandInfo_URI{Value: &(task.schedulerCore.schedulerHTTPServer.hostURI), Executable: proto.Bool(true)})
	//	executorUris = append(executorUris, &mesos.CommandInfo_URI{Value: &(task.schedulerCore.schedulerHTTPServer.riakURI),
	//		Executable: proto.Bool(false), Extract: proto.Bool(true)})

	 exec := &mesos.ExecutorInfo{
		//No idea is this is the "right" way to do it, but I think so?
		ExecutorId: util.NewExecutorID(task.currentTaskName()),
		Name:       proto.String("Test Executor (Go)"),
		// Dynamically populate this based
		Source: proto.String(task.schedulerCore.driverConfig.Framework.Id.GetValue()),
		Command: &mesos.CommandInfo{
			Value: proto.String(task.schedulerCore.schedulerHTTPServer.executorName),
			Uris:  executorUris,
			Shell: proto.Bool(false),
			Arguments: []string{"-taskid", task.currentTaskName()},
		},
	}
	taskId := &mesos.TaskID{
		Value: proto.String(task.currentTaskName()),
	}
	taskInfo := &mesos.TaskInfo{
		Name:     proto.String(task.currentTaskName()),
		TaskId:   taskId,
		SlaveId:  nil,
		Executor: exec,
		Resources: []*mesos.Resource{
			util.NewScalarResource("mem", 1),
		},
		Data: []byte{'h', 'e', 'l', 'l', 'o'},
	}
	if !task.schedulerCore.ScheduleTask(taskInfo, task, []ResourceAsker{AskForPorts(2), AskForMemory(128)}) {
		log.Info("Failed to schedule task")
		time.AfterFunc(15 * time.Second, func() { task.schedulerCore.TriggerReconcilation(task.currentTaskName()) })
	}

}
func (task *TargetTask) handleStatusUpdate(statusUpdate *mesos.TaskStatus) {
	log.Info("Got status update: ", *statusUpdate)
	if statusUpdate.TaskId.GetValue() != task.currentTaskName() {
		switch *statusUpdate.State {
		case mesos.TaskState_TASK_LOST:
		case mesos.TaskState_TASK_FAILED:
		case mesos.TaskState_TASK_FINISHED:
		case mesos.TaskState_TASK_ERROR:
		default: panic("Historical task may have come back alive")
		}
	} else {
		switch *statusUpdate.State {
		case mesos.TaskState_TASK_LOST: task.reviveTask()
		case mesos.TaskState_TASK_FAILED: task.reviveTask()
		case mesos.TaskState_TASK_FINISHED: task.reviveTask()
		case mesos.TaskState_TASK_ERROR: task.reviveTask()
		default: log.Info("Curren task status update: ", statusUpdate)
		}
	}
}
