package scheduler

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/basho-labs/riak-mesos/common"
	rexclient "github.com/basho-labs/riak-mesos/riak_explorer/client"
	"github.com/basho-labs/riak-mesos/scheduler/process_state"
	"github.com/golang/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	"github.com/satori/go.uuid"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	CPUS_PER_EXECUTOR = 0
	MEM_PER_EXECUTOR  = 32
	PORTS_PER_TASK    = 10
	CONTAINER_PATH    = "data"
)

type FrameworkRiakNode struct {
	// This is super hacky, we're relying on the following to be NOT serialized, and defaults. FIX THIS. Somehow..
	reconciled           bool      `json:"-"`
	lastAskedToReconcile time.Time `json:"-"`

	UUID             uuid.UUID
	DestinationState process_state.ProcessState
	CurrentState     process_state.ProcessState
	TaskStatus       *mesos.TaskStatus
	Generation       int
	LastOfferUsed    *mesos.Offer
	LastPortsUsed    *mesos.Resource
	TaskData         common.TaskData
	FrameworkName    string
	ClusterName      string
	Role             *string
	Principal        *string
	Cpus             float64
	Mem              float64
	Disk             float64
	Ports            int
	ExecCpus         float64
	ExecMem          float64
	PersistenceID    string
	ContainerPath    string
}

func NewFrameworkRiakNode(sc *SchedulerCore, clusterName string) *FrameworkRiakNode {
	nodeCpusFloat, err := strconv.ParseFloat(sc.NodeCpus, 64)
	if err != nil {
		log.Panicf("Unable to determine node_cpus: %+v", err)
	}
	nodeMemFloat, err := strconv.ParseFloat(sc.NodeMem, 64)
	if err != nil {
		log.Panicf("Unable to determine node_mem: %+v", err)
	}
	nodeDiskFloat, err := strconv.ParseFloat(sc.NodeDisk, 64)
	if err != nil {
		log.Panicf("Unable to determine node_disk: %+v", err)
	}

	return &FrameworkRiakNode{
		DestinationState: process_state.Started,
		CurrentState:     process_state.Unknown,
		Generation:       0,
		reconciled:       false,
		FrameworkName:    sc.frameworkName,
		Role:             &sc.frameworkRole,
		Principal:        &sc.mesosAuthPrincipal,
		UUID:             uuid.NewV4(),
		ClusterName:      clusterName,
		Cpus:             nodeCpusFloat,
		Mem:              nodeMemFloat,
		Disk:             nodeDiskFloat,
		Ports:            PORTS_PER_TASK,
		ExecCpus:         CPUS_PER_EXECUTOR,
		ExecMem:          MEM_PER_EXECUTOR,
		PersistenceID:    uuid.NewV4().String(),
		ContainerPath:    CONTAINER_PATH,
	}
}

func (frn *FrameworkRiakNode) NeedsToBeScheduled() bool {
	// Poor man's FSM:
	// TODO: Fill out rest of possible states

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
				return true
			case process_state.Failed:
				return true
			case process_state.ReservationFailed:
				return true
			}
		}
	}
	log.Panicf("Hit unknown, Current State: (%v), Destination State: (%v)", frn.CurrentState, frn.DestinationState)
	return false
}

func (frn *FrameworkRiakNode) GetExecutorResources() []*mesos.Resource {
	return frn.GetReservedResources(frn.ExecCpus, frn.ExecMem, 0, 0)
}

func (frn *FrameworkRiakNode) GetTaskResources() []*mesos.Resource {
	return frn.GetReservedResources(frn.Cpus, frn.Mem, frn.Disk, frn.Ports)
}

func (frn *FrameworkRiakNode) GetCombinedResources() []*mesos.Resource {
	return frn.GetReservedResources(frn.Cpus+frn.ExecCpus, frn.Mem+frn.ExecMem, frn.Disk, frn.Ports)
}

func (frn *FrameworkRiakNode) GetReservedResources(cpusValue float64, memValue float64, diskValue float64, portsValue int) []*mesos.Resource {
	resources := []*mesos.Resource{}

	reservation := &mesos.Resource_ReservationInfo{}
	if frn.Principal != nil {
		reservation.Principal = frn.Principal
	}

	if cpusValue > 0 {
		cpus := util.NewScalarResource("cpus", cpusValue)
		cpus.Role = frn.Role
		cpus.Reservation = reservation
		resources = append(resources, cpus)
	}

	if memValue > 0 {
		mem := util.NewScalarResource("mem", memValue)
		mem.Role = frn.Role
		mem.Reservation = reservation
		resources = append(resources, mem)
	}

	if diskValue > 0 {
		mode := mesos.Volume_RW
		volume := &mesos.Volume{
			ContainerPath: &frn.ContainerPath,
			Mode:          &mode,
		}
		persistence := &mesos.Resource_DiskInfo_Persistence{
			Id: &frn.PersistenceID,
		}
		info := &mesos.Resource_DiskInfo{
			Persistence: persistence,
			Volume:      volume,
		}
		disk := util.NewScalarResource("disk", diskValue)
		disk.Role = frn.Role
		disk.Reservation = reservation
		disk.Disk = info
		resources = append(resources, disk)
	}

	if portsValue > 0 {
		// log.Infof("Offer before grabbing ports... %+v", frn.LastOfferUsed)
		_, ports := common.CreatePortsResourceFromResources(frn.LastOfferUsed.Resources, portsValue)
		// ports.Role = frn.Role
		// ports.Reservation = reservation
		// log.Infof("Ports after creating from offer... %+v", ports)
		// log.Infof("Offer after grabbing ports... %+v", frn.LastOfferUsed)
		resources = append(resources, ports)
		// resources = append(resources, leftoverPorts)
	}

	return resources
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

func (frn *FrameworkRiakNode) ApplyOffer(mutableOffer *mesos.Offer) (bool, *mesos.Offer) {
	//Add logic here to reuse LastPortsUsed if it exists. Use same logic when creating resources later for reservations / launches

	if !common.ScalarResourcesWillFit(mutableOffer.Resources, frn.Cpus+frn.ExecCpus, frn.Mem+frn.ExecMem, frn.Disk) ||
		!common.PortResourceWillFit(mutableOffer.Resources, frn.Ports) {
		log.Info("Attempted to apply offer but offer does not have enough capacity")
		return false, mutableOffer
	}

	mutableOffer.Resources = common.ApplyScalarResources(mutableOffer.Resources, frn.Cpus+frn.ExecCpus, frn.Mem+frn.ExecMem, frn.Disk)
	mutableOffer.Resources, frn.LastPortsUsed = ApplyRangesResource(mutableResources, ports)
	frn.LastOfferUsed = mutableOffer
	frn.CurrentState = process_state.ReservationRequested
	return true, mutableOffer
}

func (frn *FrameworkRiakNode) HasRequestedReservation() bool {
	if frn.LastOfferUsed == nil {
		return false
	}

	if frn.CurrentState == process_state.ReservationRequested {
		return true
	}

	return false
}

func (frn *FrameworkRiakNode) HasReservation() bool {
	if frn.LastOfferUsed == nil {
		return false
	}

	if common.ResourcesHaveReservations(frn.LastOfferUsed.Resources) {
		return true
	}

	return false
}

func (frn *FrameworkRiakNode) OfferCompatible(immutableOffer *mesos.Offer) bool {
	log.Infof("Checking if offer is compatible for reservation. Offer's SlaveId: %+v, Node's last SlaveId: %+v", immutableOffer.SlaveId.GetValue(), frn.LastOfferUsed.SlaveId.GetValue())
	if immutableOffer.SlaveId.GetValue() != frn.LastOfferUsed.SlaveId.GetValue() {
		return false
	}
	return true
}

func (frn *FrameworkRiakNode) PrepareForLaunchAndGetNewTaskInfo(sc *SchedulerCore) *mesos.TaskInfo {
	// THIS IS A MUTATING CALL
	if frn.CurrentState != process_state.Shutdown &&
		frn.CurrentState != process_state.Failed &&
		frn.CurrentState != process_state.Unknown &&
		frn.CurrentState != process_state.ReservationRequested &&
		frn.CurrentState != process_state.ReservationFailed {
		log.Panicf("Trying to generate Task Info while node is up. ZK FRN State: %v", frn.CurrentState)
	}
	frn.Generation = frn.Generation + 1
	frn.TaskStatus = nil
	frn.CurrentState = process_state.Starting

	offer := frn.LastOfferUsed
	executorAsk := frn.GetExecutorResources()
	taskAsk := frn.GetTaskResources()

	if sc.compatibilityMode {
		executorAsk = common.RemoveReservations(executorAsk)
		taskAsk = common.RemoveReservations(taskAsk)
	}

	executorUris := []*mesos.CommandInfo_URI{
		&mesos.CommandInfo_URI{
			Value:      &(sc.schedulerHTTPServer.hostURI),
			Executable: proto.Bool(true),
		},
	}

	superChrootValue := true
	if os.Getenv("USE_SUPER_CHROOT") == "false" {
		superChrootValue = false
	}

	exec := &mesos.ExecutorInfo{
		//No idea is this is the "right" way to do it, but I think so?
		ExecutorId: util.NewExecutorID(frn.ExecutorID()),
		Name:       proto.String("Executor (Go)"),
		Source:     proto.String("Riak Mesos Framework (Go)"),
		Command: &mesos.CommandInfo{
			Value:     proto.String(sc.schedulerHTTPServer.executorName),
			Uris:      executorUris,
			Shell:     proto.Bool(false),
			Arguments: []string{sc.schedulerHTTPServer.executorName, "-logtostderr=true", "-taskinfo", frn.CurrentID()},
		},
		Resources: executorAsk,
	}
	taskId := &mesos.TaskID{
		Value: proto.String(frn.CurrentID()),
	}

	nodename := frn.CurrentID() + "@" + offer.GetHostname()

	if !strings.Contains(offer.GetHostname(), ".") {
		nodename = nodename + "."
	}
	ports := portIter(taskAsk)

	taskData := common.TaskData{
		FullyQualifiedNodeName: nodename,
		Host:           offer.GetHostname(),
		Zookeepers:     sc.zookeepers,
		NodeID:         frn.UUID.String(),
		FrameworkName:  sc.frameworkName,
		URI:            sc.schedulerHTTPServer.GetURI(),
		ClusterName:    frn.ClusterName,
		UseSuperChroot: superChrootValue,
		HTTPPort:       <-ports,
		PBPort:         <-ports,
		DisterlPort:    <-ports,
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
		Resources: taskAsk,
		Data:      binTaskData,
	}

	return taskInfo
}

func (frn *FrameworkRiakNode) CurrentID() string {
	return fmt.Sprintf("%s-%s-%s-%d", frn.FrameworkName, frn.ClusterName, frn.UUID.String(), frn.Generation)
}

func (frn *FrameworkRiakNode) ExecutorID() string {
	return frn.CurrentID()
}

func (frn *FrameworkRiakNode) handleUpToDownTransition(sc *SchedulerCore, frc *FrameworkRiakCluster) {
	for _, riakNode := range sc.schedulerState.Clusters[frc.Name].Nodes {
		if riakNode.CurrentState == process_state.Started && riakNode != frn {

			// rexc := rexclient.NewRiakExplorerClient(fmt.Sprintf("%s:%d", riakNode.LastOfferUsed.GetHostname(), riakNode.TaskData.RexPort))
			rexc := rexclient.NewRiakExplorerClient(fmt.Sprintf("%s:%d", riakNode.LastOfferUsed.GetHostname(), riakNode.TaskData.HTTPPort))

			// We should try to join against this node
			log.Infof("Making leave: %+v to %+v", frn.TaskData.FullyQualifiedNodeName, riakNode.TaskData.FullyQualifiedNodeName)
			leaveReply, leaveErr := rexc.ForceRemove(riakNode.TaskData.FullyQualifiedNodeName, frn.TaskData.FullyQualifiedNodeName)
			log.Infof("Triggered leave: %+v, %+v", leaveReply, leaveErr)
			if leaveErr == nil {
				log.Info("Leave successful")
				break // We're done here
			}
		}
	}
}

func (frn *FrameworkRiakNode) attemptJoin(riakNode *FrameworkRiakNode, retry int, maxRetry int) bool {
	if retry > maxRetry {
		log.Infof("Attempted joining %+v to %+v %+v times and failed.", frn.TaskData.FullyQualifiedNodeName, riakNode.TaskData.FullyQualifiedNodeName, maxRetry)
		return false
	}

	rexHostname := fmt.Sprintf("%s:%d", riakNode.LastOfferUsed.GetHostname(), riakNode.TaskData.HTTPPort)
	rexc := rexclient.NewRiakExplorerClient(rexHostname)
	// We should try to join against this node
	log.Infof("Joining %+v to %+v", frn.TaskData.FullyQualifiedNodeName, riakNode.TaskData.FullyQualifiedNodeName)
	joinReply, joinErr := rexc.Join(frn.TaskData.FullyQualifiedNodeName, riakNode.TaskData.FullyQualifiedNodeName)
	log.Infof("Triggered join: %+v, %+v", joinReply, joinErr)
	if joinReply.Join.Success == "ok" {
		return true
	}

	time.Sleep(5 * time.Second)
	return frn.attemptJoin(riakNode, retry+1, maxRetry)
}

func (frn *FrameworkRiakNode) handleStartingToRunningTransition(sc *SchedulerCore, frc *FrameworkRiakCluster) {
	for _, riakNode := range sc.schedulerState.Clusters[frc.Name].Nodes {
		if riakNode.CurrentState == process_state.Started {

			joinSuccess := frn.attemptJoin(riakNode, 0, 5)

			if joinSuccess {
				break // We're done here
			}
		}
	}
}
func (frn *FrameworkRiakNode) handleStatusUpdate(sc *SchedulerCore, frc *FrameworkRiakCluster, statusUpdate *mesos.TaskStatus) {
	frn.reconciled = true

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
			if frn.CurrentState == process_state.Starting {
				frn.handleStartingToRunningTransition(sc, frc)
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
				frn.handleUpToDownTransition(sc, frc)
			}
			if frn.CurrentState == process_state.ReservationRequested {
				frn.CurrentState = process_state.ReservationFailed
			} else {
				frn.CurrentState = process_state.Failed
			}
		}

	// Maybe? -- Not entirely sure.
	case mesos.TaskState_TASK_KILLED:
		frn.CurrentState = process_state.Shutdown

	// These two could actually appear if the task is running -- we should better handle
	// status updates in these two scenarios
	case mesos.TaskState_TASK_LOST:
		{
			if frn.CurrentState == process_state.Started {
				frn.handleUpToDownTransition(sc, frc)
			}
			if frn.CurrentState == process_state.ReservationRequested {
				frn.CurrentState = process_state.ReservationFailed
			} else {
				frn.CurrentState = process_state.Failed
			}
		}
	case mesos.TaskState_TASK_ERROR:
		if frn.CurrentState == process_state.ReservationRequested {
			frn.CurrentState = process_state.ReservationFailed
		} else {
			frn.CurrentState = process_state.Failed
		}
	default:
		log.Fatal("Received unknown status update")
	}
}

func (frn *FrameworkRiakNode) GetTaskStatus() *mesos.TaskStatus {
	if frn.TaskStatus != nil {
		return frn.TaskStatus
	}

	ts := mesos.TaskState_TASK_ERROR
	return &mesos.TaskStatus{
		TaskId:  &mesos.TaskID{Value: proto.String(frn.CurrentID())},
		State:   &ts,
		SlaveId: &mesos.SlaveID{Value: proto.String("")}, // Slave ID isn't required
	}
}
