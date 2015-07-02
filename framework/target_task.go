package framework

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/basho/bletchley/common"
	"github.com/basho/bletchley/metadata_manager"
	"github.com/golang/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	"time"
)

type targetTaskStateType int

const (
	unknownTargetTaskState targetTaskStateType = iota
)

type internalStatus int

const (
	targetTaskUnknown        internalStatus = iota
	targetTaskStarting                      = iota
	targetTaskStartingFailed                = iota
)

type TargetTask struct {
	schedulerCore          *SchedulerCore
	TaskName               string
	uuid                   string
	mesosTaskStatusUpdates chan *mesos.TaskStatus
	currentTaskStatus      *mesos.TaskStatus
	targetTaskState        targetTaskStateType
	mgr                    *metadata_manager.MetadataManager
	status                 internalStatus
}

func NewTargetTask(taskName string, schedulerCore *SchedulerCore, mgr *metadata_manager.MetadataManager) *TargetTask {
	uuid := mgr.GetTaskUUID(taskName)
	return &TargetTask{schedulerCore: schedulerCore,
		TaskName:               taskName,
		mesosTaskStatusUpdates: make(chan *mesos.TaskStatus),
		// TODO: Add lower generation marker
		uuid:   uuid,
		mgr:    mgr,
		status: targetTaskUnknown,
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
		case <-time.After(1 * time.Minute):
			{
				// Refresh task
				if task.currentTaskStatus != nil {
					switch task.status {
					case targetTaskStartingFailed:
						task.reviveTask()
					}
				}
			}
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
			Value:     proto.String(task.schedulerCore.schedulerHTTPServer.executorName),
			Uris:      executorUris,
			Shell:     proto.Bool(false),
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
	if !task.schedulerCore.ScheduleTask(taskInfo, task, []common.ResourceAsker{common.AskForPorts(2), common.AskForMemory(128)}) {
		log.Error("Failed to schedule task")
		task.status = targetTaskStartingFailed
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
		default:
			panic("Historical task may have come back alive")
		}
	} else {
		task.currentTaskStatus = statusUpdate
		task.status = targetTaskStarting
		switch *statusUpdate.State {
		case mesos.TaskState_TASK_LOST:
			task.reviveTask()
		case mesos.TaskState_TASK_FAILED:
			task.reviveTask()
		case mesos.TaskState_TASK_FINISHED:
			{
				task.reviveTask()
			}
		case mesos.TaskState_TASK_ERROR:
			task.reviveTask()
		default:
			log.Info("Current task status update: ", statusUpdate)
		}
	}
}
