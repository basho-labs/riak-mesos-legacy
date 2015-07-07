package framework

import (
	"encoding/json"
	"github.com/golang/protobuf/proto"
	log "github.com/Sirupsen/logrus"
	"github.com/basho/bletchley/framework/riak_node_states"
	metamgr "github.com/basho/bletchley/metadata_manager"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"github.com/satori/go.uuid"
	"fmt"
)

// Next Status

type FrameworkRiakNode struct {
	frc              *FrameworkRiakCluster `json:"-"`
	zkNode           *metamgr.ZkNode       `json:"-"`
	UUID             uuid.UUID
	DestinationState riak_node_states.State
	CurrentState     riak_node_states.State
	TaskStatus       *mesos.TaskStatus
	generation       int
}

func NewFrameworkRiakNode() *FrameworkRiakNode {
	return &FrameworkRiakNode{
		// We can assume this for now? I think
		DestinationState: riak_node_states.Started,
		CurrentState:     riak_node_states.Unknown,
		generation:		  0,
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

	switch (frn.DestinationState) {
		case riak_node_states.Started: {
			switch frn.CurrentState {
				case riak_node_states.Started: return false
				case riak_node_states.Unknown: return false
				case riak_node_states.Starting: return false
				case riak_node_states.Shutdown: return true
			}
		}
	}
	log.Panicf("Hit unknown, Current State: (%v), Destination State: (%v)", frn.CurrentState, frn.DestinationState)
	return false
}
func (frn *FrameworkRiakNode) NewID() string {
	return fmt.Sprintf("%s-%s-%d", frn.frc.Name, frn.UUID.String(), frn.generation)
}

func (frn *FrameworkRiakNode) GetZkNode() *metamgr.ZkNode {
	return frn.zkNode
}

func (frn *FrameworkRiakNode) handleStatusUpdate(statusUpdate *mesos.TaskStatus) {
	frn.TaskStatus = statusUpdate
	switch *statusUpdate.State.Enum() {
		case mesos.TaskState_TASK_RUNNING: frn.CurrentState = riak_node_states.Started
		case mesos.TaskState_TASK_FINISHED: frn.CurrentState = riak_node_states.Shutdown
		case mesos.TaskState_TASK_LOST: frn.CurrentState = riak_node_states.Shutdown
		default: log.Fatal("Received unknown status update")
	}
}
func (frn *FrameworkRiakNode) GetTaskStatus() *mesos.TaskStatus {
	if frn.TaskStatus != nil {
		return frn.TaskStatus
	} else {
		ts := mesos.TaskState_TASK_ERROR
		return &mesos.TaskStatus{
			TaskId:  &mesos.TaskID{Value: proto.String(frn.NewID())},
			State:   &ts,
			SlaveId: &mesos.SlaveID{Value: proto.String("")}, // Slave ID isn't required
		}
	}

}