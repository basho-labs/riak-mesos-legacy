package scheduler

import (
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	"github.com/basho-labs/riak-mesos/scheduler/process_state"
)

// Getters
func (frn *FrameworkRiakNode) HasRequestedReservation() bool {
	if frn.LastOfferUsed == nil {
		return false
	}

	for _, resource := range util.FilterResources(frn.LastOfferUsed.Resources, func(res *mesos.Resource) bool { return res.GetName() == "disk" && res.Reservation != nil }) {
		if resource.Disk != nil && *resource.Disk.Persistence.Id == frn.PersistenceID() {
			return true
		}
	}

	if frn.CurrentState == process_state.ReservationRequested {
		return true
	}

	return false
}

func (frn *FrameworkRiakNode) CanBeScheduled() bool {
	switch frn.DestinationState {
	case process_state.Started:
		{
			switch frn.CurrentState {
			case process_state.Started:
				return false
			case process_state.Unknown:
				return true
			case process_state.ReservationRequested:
				return true
			case process_state.Starting:
				return false
			case process_state.Shutdown:
				return false
			case process_state.Failed:
				return true
			case process_state.ReservationFailed:
				return true
			}
		}
	}
	return false
}
func (frn *FrameworkRiakNode) CanBeKilled() bool {
	return frn.DestinationState == process_state.Shutdown &&
		(frn.CurrentState == process_state.Starting ||
			frn.CurrentState == process_state.Started)
}
func (frn *FrameworkRiakNode) CanBeRemoved() bool {
	return frn.DestinationState == process_state.Shutdown &&
		(frn.CurrentState == process_state.Shutdown ||
			frn.CurrentState == process_state.Failed)
}
func (frn *FrameworkRiakNode) CanJoinCluster() bool {
	return frn.CurrentState == process_state.Starting &&
		frn.DestinationState == process_state.Started
}
func (frn *FrameworkRiakNode) CanBeJoined() bool {
	return frn.CurrentState == process_state.Started &&
		frn.DestinationState == process_state.Started
}
func (frn *FrameworkRiakNode) CanBeLeft() bool {
	return frn.CanBeJoined()
}

// Setters
func (frn *FrameworkRiakNode) KillNext() {
	frn.DestinationState = process_state.Shutdown
}

func (frn *FrameworkRiakNode) Unreserve() {
	frn.CurrentState = process_state.Unknown
}
func (frn *FrameworkRiakNode) Stage() {
	frn.Start()
}
func (frn *FrameworkRiakNode) Start() {
	frn.CurrentState = process_state.Starting
	frn.DestinationState = process_state.Started
}
func (frn *FrameworkRiakNode) Run() {
	frn.CurrentState = process_state.Started
}
func (frn *FrameworkRiakNode) Finish() {
	frn.CurrentState = process_state.Shutdown
}
func (frn *FrameworkRiakNode) Kill() {
	frn.CurrentState = process_state.Shutdown
}
func (frn *FrameworkRiakNode) Fail() {
	frn.Error()
}
func (frn *FrameworkRiakNode) Lost() {
	frn.Error()
}
func (frn *FrameworkRiakNode) Error() {
	if frn.CurrentState == process_state.ReservationRequested {
		frn.CurrentState = process_state.ReservationFailed
	} else {
		frn.CurrentState = process_state.Failed
	}
}
