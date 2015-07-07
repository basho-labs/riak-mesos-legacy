package framework

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/basho/bletchley/common"
	"github.com/basho/bletchley/framework/riak_node_states"
	metamgr "github.com/basho/bletchley/metadata_manager"
	"github.com/golang/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	"github.com/satori/go.uuid"
)

// Next Status

type FrameworkRiakNode struct {
	frc              *FrameworkRiakCluster `json:"-"`
	zkNode           *metamgr.ZkNode       `json:"-"`
	UUID             uuid.UUID
	DestinationState riak_node_states.State
	CurrentState     riak_node_states.State
	TaskStatus       *mesos.TaskStatus
	Generation       int
	LastTaskInfo	 *mesos.TaskInfo
	LastOfferUsed	 *mesos.Offer
}

func NewFrameworkRiakNode() *FrameworkRiakNode {
	return &FrameworkRiakNode{
		// We can assume this for now? I think
		DestinationState: riak_node_states.Started,
		CurrentState:     riak_node_states.Unknown,
		Generation:       0,
	}
}

func (frn *FrameworkRiakNode) Persist() {
	data, err := json.Marshal(frn)
	if err != nil {
		log.Panic("error:", err)
	}
	frn.zkNode.SetData(data)
}
func (frn *FrameworkRiakNode) NeedsToBeScheduled() bool {
	// Poor man's FSM:
	// TODO: Fill out rest of possible states
	switch frn.DestinationState {
	case riak_node_states.Started:
		{
			switch frn.CurrentState {
			case riak_node_states.Started:
				return false
			case riak_node_states.Unknown:
				return false
			case riak_node_states.Starting:
				return false
			case riak_node_states.Shutdown:
				return true
			}
		}
	}
	log.Panicf("Hit unknown, Current State: (%v), Destination State: (%v)", frn.CurrentState, frn.DestinationState)
	return false
}
func (frn *FrameworkRiakNode) CurrentID() string {
	return fmt.Sprintf("%s-%s-%d", frn.frc.Name, frn.UUID.String(), frn.Generation)
}

func (frn *FrameworkRiakNode) GetZkNode() *metamgr.ZkNode {
	return frn.zkNode
}

func (frn *FrameworkRiakNode) handleStatusUpdate(statusUpdate *mesos.TaskStatus) {
	// Poor man's FSM event handler
	frn.TaskStatus = statusUpdate
	switch *statusUpdate.State.Enum() {
	case mesos.TaskState_TASK_STAGING:
		frn.CurrentState = riak_node_states.Starting
	case mesos.TaskState_TASK_STARTING:
		frn.CurrentState = riak_node_states.Starting
	case mesos.TaskState_TASK_RUNNING:
		frn.CurrentState = riak_node_states.Started
	case mesos.TaskState_TASK_FINISHED:
		frn.CurrentState = riak_node_states.Shutdown
	case mesos.TaskState_TASK_FAILED:
		frn.CurrentState = riak_node_states.Shutdown
	case mesos.TaskState_TASK_KILLED:
		frn.CurrentState = riak_node_states.Shutdown

	// These two could actually appear if the task is running -- we should better handle
	// status updates in these two scenarios
	case mesos.TaskState_TASK_LOST:
		frn.CurrentState = riak_node_states.Shutdown
	case mesos.TaskState_TASK_ERROR:
		frn.CurrentState = riak_node_states.Shutdown
	default:
		log.Fatal("Received unknown status update")
	}
}
func (frn *FrameworkRiakNode) GetTaskStatus() *mesos.TaskStatus {
	if frn.TaskStatus != nil {
		return frn.TaskStatus
	} else {
		ts := mesos.TaskState_TASK_ERROR
		return &mesos.TaskStatus{
			TaskId:  &mesos.TaskID{Value: proto.String(frn.CurrentID())},
			State:   &ts,
			SlaveId: &mesos.SlaveID{Value: proto.String("")}, // Slave ID isn't required
		}
	}
}
func (frn *FrameworkRiakNode) GetAsks() []common.ResourceAsker {
	return []common.ResourceAsker{common.AskForCPU(0.1), common.AskForPorts(3), common.AskForMemory(128)}
}
func (frn *FrameworkRiakNode) GetCombinedAsk() common.CombinedResourceAsker {
	ret := func(offer []*mesos.Resource) ([]*mesos.Resource, []*mesos.Resource, bool) {
		asks := []*mesos.Resource{}
		success := true
		remaining := offer
		for _, fun := range frn.GetAsks() {
			var newAsk *mesos.Resource
			remaining, newAsk, success = fun(remaining)
			asks = append(asks, newAsk)
			if !success {
				return offer, []*mesos.Resource{}, false
			}
		}
		return remaining, asks, success
	}
	return ret
}

func (frn *FrameworkRiakNode) PrepareForLaunchAndGetNewTaskInfo(offer *mesos.Offer, resources []*mesos.Resource) *mesos.TaskInfo {
	// THIS IS A MUTATING CALL

	if frn.CurrentState != riak_node_states.Shutdown {
		panic("Generate Task Info while node is up")
	}
	frn.Generation = frn.Generation + 1
	frn.TaskStatus = nil
	frn.CurrentState = riak_node_states.Starting
	frn.LastOfferUsed = offer

	executorUris := []*mesos.CommandInfo_URI{}
	executorUris = append(executorUris,
		&mesos.CommandInfo_URI{Value: &(frn.frc.sc.schedulerHTTPServer.hostURI), Executable: proto.Bool(true)})

	exec := &mesos.ExecutorInfo{
		//No idea is this is the "right" way to do it, but I think so?
		ExecutorId: util.NewExecutorID(frn.CurrentID()),
		Name:       proto.String("Test Executor (Go)"),
		Source:     proto.String("Riak Mesos Framework (Go)"),
		Command: &mesos.CommandInfo{
			Value:     proto.String(frn.frc.sc.schedulerHTTPServer.executorName),
			Uris:      executorUris,
			Shell:     proto.Bool(false),
			Arguments: []string{frn.frc.sc.schedulerHTTPServer.executorName, "-taskid", frn.CurrentID()},
		},
	}
	taskId := &mesos.TaskID{
		Value: proto.String(frn.CurrentID()),
	}
	taskInfo := &mesos.TaskInfo{
		Name:      proto.String(frn.CurrentID()),
		TaskId:    taskId,
		SlaveId:   offer.SlaveId,
		Executor:  exec,
		Resources: resources,
		Data:      []byte{'h', 'e', 'l', 'l', 'o'},
	}
	frn.LastTaskInfo = taskInfo

	return taskInfo
}
