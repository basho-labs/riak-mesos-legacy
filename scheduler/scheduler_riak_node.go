package scheduler

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/basho-labs/riak-mesos/common"
	metamgr "github.com/basho-labs/riak-mesos/metadata_manager"
	"github.com/basho-labs/riak-mesos/scheduler/process_state"
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
	DestinationState process_state.ProcessState
	CurrentState     process_state.ProcessState
	TaskStatus       *mesos.TaskStatus
	Generation       int
	LastTaskInfo     *mesos.TaskInfo
	LastOfferUsed    *mesos.Offer
	TaskData         common.TaskData
}

func NewFrameworkRiakNode() *FrameworkRiakNode {
	return &FrameworkRiakNode{
		// We can assume this for now? I think
		DestinationState: process_state.Started,
		CurrentState:     process_state.Unknown,
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

	log.Infof("Checking if node needs to be scheduled, Node: (%v), Current State: (%v), Destination State: (%v)", frn.UUID.String(), frn.CurrentState, frn.DestinationState)

	switch frn.DestinationState {
	case process_state.Started:
		{
			switch frn.CurrentState {
			case process_state.Started:
				return false
			case process_state.Unknown:
				return false
			case process_state.Starting:
				return false
			case process_state.Shutdown:
				return true
			case process_state.Failed:
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

func (frn *FrameworkRiakNode) ExecutorID() string {
	return frn.CurrentID()
}

func (frn *FrameworkRiakNode) NodeName() string {
	return fmt.Sprintf("%s-%s-%d", frn.frc.Name, frn.UUID.String(), frn.Generation)
}

func (frn *FrameworkRiakNode) GetZkNode() *metamgr.ZkNode {
	return frn.zkNode
}

func (frn *FrameworkRiakNode) handleRunningToFailedTransition() {
	rexc := frn.frc.sc.rex.NewRiakExplorerClient()
	for riakNodeName := range frn.frc.nodes {
		riakNode := frn.frc.nodes[riakNodeName]
		if riakNode.CurrentState == process_state.Started {
			// We should try to join against this node
			leaveReply, leaveErr := rexc.ForceRemove(riakNode.TaskData.FullyQualifiedNodeName, frn.TaskData.FullyQualifiedNodeName)
			log.Infof("Triggered leave: %+v, %+v", leaveReply, leaveErr)
			if leaveErr == nil {
				break // We're done here
			}
		}
	}
}
func (frn *FrameworkRiakNode) handleStartingToRunningTransition() {
	rexc := frn.frc.sc.rex.NewRiakExplorerClient()
	for riakNodeName := range frn.frc.nodes {
		riakNode := frn.frc.nodes[riakNodeName]
		if riakNode.CurrentState == process_state.Started {
			// We should try to join against this node
			joinReply, joinErr := rexc.Join(frn.TaskData.FullyQualifiedNodeName, riakNode.TaskData.FullyQualifiedNodeName)
			log.Infof("Triggered join: %+v, %+v", joinReply, joinErr)
			if joinErr == nil {
				break // We're done here
			}
		}
	}
}
func (frn *FrameworkRiakNode) handleStatusUpdate(statusUpdate *mesos.TaskStatus) {
	// TODO: Check the task ID in the TaskStatus to make sure it matches our current task

	// Poor man's FSM event handler
	frn.TaskStatus = statusUpdate
	switch *statusUpdate.State.Enum() {
	case mesos.TaskState_TASK_STAGING:
		frn.CurrentState = process_state.Starting
	case mesos.TaskState_TASK_STARTING:
		{
			frn.CurrentState = process_state.Starting
		}
	case mesos.TaskState_TASK_RUNNING:
		{
			frn.frc.Trigger()
			if frn.CurrentState == process_state.Starting {
				frn.handleStartingToRunningTransition()
			}
			frn.CurrentState = process_state.Started
		}
	case mesos.TaskState_TASK_FINISHED:
		{
			log.Info("We should never get to this state")
			frn.CurrentState = process_state.Shutdown
		}
	case mesos.TaskState_TASK_FAILED:
		{
			if frn.CurrentState == process_state.Started {
				frn.handleRunningToFailedTransition()
			}
			frn.CurrentState = process_state.Failed
		}

	// Maybe? -- Not entirely sure.
	case mesos.TaskState_TASK_KILLED:
		frn.CurrentState = process_state.Shutdown

	// These two could actually appear if the task is running -- we should better handle
	// status updates in these two scenarios
	case mesos.TaskState_TASK_LOST:
		frn.CurrentState = process_state.Failed
	case mesos.TaskState_TASK_ERROR:
		frn.CurrentState = process_state.Failed
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
	// 10 for good measure
	// Ports:
	// -Protocol Buffers
	// -HTTP
	// -Riak Explorer (rex)
	// 4-10 -- unknown, so far
	// Potential:
	// EPM

	return []common.ResourceAsker{common.AskForCPU(0.3), common.AskForPorts(10), common.AskForMemory(320)}
}
func (frn *FrameworkRiakNode) GetCombinedAsk() common.CombinedResourceAsker {
	log.Infof("Before combined ask, Node: (%v), Current State: (%v), Destination State: (%v)", frn.UUID.String(), frn.CurrentState, frn.DestinationState)
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
	log.Infof("After combined ask, Node: (%v), Current State: (%v), Destination State: (%v)", frn.UUID.String(), frn.CurrentState, frn.DestinationState)
	return ret
}

func (frn *FrameworkRiakNode) PrepareForLaunchAndGetNewTaskInfo(offer *mesos.Offer, resources []*mesos.Resource) *mesos.TaskInfo {
	// THIS IS A MUTATING CALL

	log.Infof("Preparing for launch, Node: (%v), Current State: (%v), Destination State: (%v)", frn.UUID.String(), frn.CurrentState, frn.DestinationState)

	if frn.CurrentState != process_state.Shutdown && frn.CurrentState != process_state.Failed && frn.CurrentState != process_state.Unknown {
		log.Panicf("Trying to generate Task Info while node is up. ZK FRN State: %v", frn.CurrentState)
	}
	frn.Generation = frn.Generation + 1
	frn.TaskStatus = nil
	frn.CurrentState = process_state.Starting
	frn.LastOfferUsed = offer

	executorUris := []*mesos.CommandInfo_URI{
		&mesos.CommandInfo_URI{
			Value:      &(frn.frc.sc.schedulerHTTPServer.hostURI),
			Executable: proto.Bool(true),
		},
		&mesos.CommandInfo_URI{
			Value:      &(frn.frc.sc.schedulerHTTPServer.riakURI),
			Executable: proto.Bool(false),
			Extract:    proto.Bool(true),
		},
	}
	//executorUris = append(executorUris,
	//	&mesos.CommandInfo_URI{Value: &(frn.frc.sc.schedulerHTTPServer.hostURI), Executable: proto.Bool(true)})

	exec := &mesos.ExecutorInfo{
		//No idea is this is the "right" way to do it, but I think so?
		ExecutorId: util.NewExecutorID(frn.ExecutorID()),
		Name:       proto.String("Test Executor (Go)"),
		Source:     proto.String("Riak Mesos Framework (Go)"),
		Command: &mesos.CommandInfo{
			Value:     proto.String(frn.frc.sc.schedulerHTTPServer.executorName),
			Uris:      executorUris,
			Shell:     proto.Bool(false),
			Arguments: []string{frn.frc.sc.schedulerHTTPServer.executorName, "-logtostderr=true", "-taskinfo", frn.CurrentID()},
		},
		Resources: []*mesos.Resource{
			util.NewScalarResource("cpus", 0.01),
			util.NewScalarResource("mem", 32),
		},
	}
	taskId := &mesos.TaskID{
		Value: proto.String(frn.CurrentID()),
	}

	nodename := frn.NodeName() + "@" + offer.GetHostname()

	if !strings.Contains(offer.GetHostname(), ".") {
		nodename = nodename + "."
	}

	taskData := common.TaskData{
		FullyQualifiedNodeName:    nodename,
		RexFullyQualifiedNodeName: "rex-" + nodename,
		Zookeepers:                frn.frc.sc.zookeepers,
		ClusterName:               frn.frc.Name,
		NodeID:                    frn.UUID.String(),
	}
	frn.TaskData = taskData

	binTaskData, err := taskData.Serialize()

	if err != nil {
		log.Panic(err)
	}

	taskInfo := &mesos.TaskInfo{
		Name:      proto.String(frn.CurrentID()),
		TaskId:    taskId,
		SlaveId:   offer.SlaveId,
		Executor:  exec,
		Resources: resources,
		Data:      binTaskData,
	}
	frn.LastTaskInfo = taskInfo

	return taskInfo
}
