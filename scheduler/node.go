package scheduler

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/basho-labs/riak-mesos/common"
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
	// Zeroing out the executor resources because it complicates the logic to reconcile reserved resources
	// when task dies but not the executor. Also, there's a rounding error in mesos when adding cpus under 1.0
	CPUS_PER_EXECUTOR = 0.1
	MEM_PER_EXECUTOR  = 32
	PORTS_PER_TASK    = 10
	CONTAINER_PATH    = "root"
)

type FrameworkRiakNode struct {
	// This is super hacky, we're relying on the following to be NOT serialized, and defaults. FIX THIS. Somehow..
	reconciled           bool      `json:"-"`
	lastAskedToReconcile time.Time `json:"-"`

	SimpleId          int
	DestinationState  process_state.ProcessState
	CurrentState      process_state.ProcessState
	TaskStatus        *mesos.TaskStatus
	Generation        int
	SlaveID           *mesos.SlaveID
	Hostname          string
	TaskData          common.TaskData
	FrameworkName     string
	ClusterName       string
	Role              *string
	Principal         *string
	Cpus              float64
	Mem               float64
	Disk              float64
	Ports             int
	UUID              string
	ContainerPath     string
	RestartGeneration int64
}

func NewFrameworkRiakNode(sc *SchedulerCore, clusterName string, restartGeneration int64, simpleId int) *FrameworkRiakNode {
	nodeCpusFloat, err := strconv.ParseFloat(sc.nodeCpus, 64)
	if err != nil {
		log.Panicf("Unable to determine node_cpus: %+v", err)
	}
	nodeMemFloat, err := strconv.ParseFloat(sc.nodeMem, 64)
	if err != nil {
		log.Panicf("Unable to determine node_mem: %+v", err)
	}
	nodeDiskFloat, err := strconv.ParseFloat(sc.nodeDisk, 64)
	if err != nil {
		log.Panicf("Unable to determine node_disk: %+v", err)
	}

	return &FrameworkRiakNode{
		DestinationState:  process_state.Started,
		CurrentState:      process_state.Unknown,
		Generation:        0,
		reconciled:        false,
		FrameworkName:     sc.frameworkName,
		Role:              &sc.frameworkRole,
		Principal:         &sc.mesosAuthPrincipal,
		SimpleId:          simpleId,
		ClusterName:       clusterName,
		Cpus:              nodeCpusFloat,
		Mem:               nodeMemFloat,
		Disk:              nodeDiskFloat,
		Ports:             PORTS_PER_TASK,
		UUID:              uuid.NewV4().String(),
		ContainerPath:     CONTAINER_PATH,
		RestartGeneration: restartGeneration,
	}
}

// --- Values ---

func (frn *FrameworkRiakNode) CurrentID() string {
	return fmt.Sprintf("%s-%s-%d", frn.FrameworkName, frn.ClusterName, frn.SimpleId)
}

func (frn *FrameworkRiakNode) Name() string {
	return fmt.Sprintf("%s-%s", frn.FrameworkName, frn.ClusterName)
}

func (frn *FrameworkRiakNode) ExecutorID() string {
	return fmt.Sprintf("%s-%s-%s-%d", frn.FrameworkName, frn.ClusterName, frn.UUID, frn.Generation)
}

func (frn *FrameworkRiakNode) PersistenceID() string {
	return frn.UUID
}

func (frn *FrameworkRiakNode) CreateTaskID() *mesos.TaskID {
	return &mesos.TaskID{Value: proto.String(frn.CurrentID())}
}

func (frn *FrameworkRiakNode) CreateExecutorID() *mesos.ExecutorID {
	return util.NewExecutorID(frn.ExecutorID())
}

// --- Resources ---

func (frn *FrameworkRiakNode) ApplyUnreservedOffer(offerHelper *common.OfferHelper) bool {
	if !offerHelper.CanFitUnreserved(frn.Cpus+CPUS_PER_EXECUTOR, frn.Mem+MEM_PER_EXECUTOR, frn.Disk, frn.Ports) {
		return false
	}

	log.Infof("Found a new offer for a node. OfferID: %+v, NodeID: %+v", offerHelper.OfferIDStr, frn.CurrentID())

	// Remove the ports and executor requirements from offerHelper, but don't reserve
	_ = offerHelper.ApplyUnreserved(CPUS_PER_EXECUTOR, MEM_PER_EXECUTOR, 0, frn.Ports)

	// Create reservation + volumes, add to offerHelper
	offerHelper.MakeReservation(frn.Cpus, frn.Mem, frn.Disk, 0, *frn.Principal, *frn.Role)
	offerHelper.MakeVolume(frn.Disk, *frn.Principal, *frn.Role, frn.PersistenceID(), frn.ContainerPath)

	// Update state
	frn.SlaveID = offerHelper.MesosOffer.SlaveId
	frn.Hostname = offerHelper.MesosOffer.GetHostname()
	frn.CurrentState = process_state.Reserved
	return true
}

func (frn *FrameworkRiakNode) ApplyReservedOffer(offerHelper *common.OfferHelper, sc *SchedulerCore) bool {
	taskAsk := []*mesos.Resource{}
	execAsk := []*mesos.Resource{}
	if sc.compatibilityMode {
		if !offerHelper.CanFitUnreserved(frn.Cpus+CPUS_PER_EXECUTOR, frn.Mem+MEM_PER_EXECUTOR, frn.Disk, frn.Ports) {
			return false
		}
		taskAsk = offerHelper.ApplyUnreserved(frn.Cpus, frn.Mem, frn.Disk, frn.Ports)
		execAsk = offerHelper.ApplyUnreserved(CPUS_PER_EXECUTOR, MEM_PER_EXECUTOR, 0, 0)
	} else {
		if !offerHelper.CanFitReserved(frn.Cpus, frn.Mem, frn.Disk, 0) ||
			!offerHelper.CanFitUnreserved(CPUS_PER_EXECUTOR, MEM_PER_EXECUTOR, 0, frn.Ports) {
			return false
		}
		taskAsk = offerHelper.ApplyReserved(frn.Cpus, frn.Mem, frn.Disk, 0, *frn.Principal, *frn.Role, frn.PersistenceID(), frn.ContainerPath)
		taskAsk = append(taskAsk, offerHelper.ApplyUnreserved(0, 0, 0, frn.Ports)...)
		execAsk = offerHelper.ApplyUnreserved(CPUS_PER_EXECUTOR, MEM_PER_EXECUTOR, 0, 0)
	}

	log.Infof("Found an offer for a launchable node. OfferID: %+v, NodeID: %+v", offerHelper.OfferIDStr, frn.CurrentID())

	frn.SlaveID = offerHelper.MesosOffer.SlaveId
	frn.Hostname = offerHelper.MesosOffer.GetHostname()
	frn.Generation = frn.Generation + 1
	frn.TaskStatus = nil
	frn.CurrentState = process_state.Starting

	taskId := frn.CreateTaskID()
	nodename := frn.CurrentID() + "@" + frn.Hostname
	if !strings.Contains(frn.Hostname, ".") {
		nodename = nodename + "."
	}
	ports := common.PortIterator(taskAsk)

	taskData := common.TaskData{
		FullyQualifiedNodeName: nodename,
		Host:           frn.Hostname,
		Zookeepers:     sc.zookeepers,
		FrameworkName:  sc.frameworkName,
		URI:            sc.schedulerHTTPServer.GetURI(),
		ClusterName:    frn.ClusterName,
		UseSuperChroot: os.Getenv("USE_SUPER_CHROOT") != "false",
		HTTPPort:       <-ports,
		PBPort:         <-ports,
		DisterlPort:    <-ports,
	}
	frn.TaskData = taskData

	binTaskData, err := taskData.Serialize()
	if err != nil {
		log.Panic(err)
	}

	execName := fmt.Sprintf("%s Executor", frn.CurrentID())
	taskInfo := &mesos.TaskInfo{
		Name:    proto.String(frn.Name()),
		TaskId:  taskId,
		SlaveId: frn.SlaveID,
		Executor: &mesos.ExecutorInfo{
			ExecutorId: frn.CreateExecutorID(),
			Name:       proto.String(execName),
			Source:     proto.String(frn.FrameworkName),
			Command: &mesos.CommandInfo{
				Value: proto.String(ExecutorValue()),
				Uris: []*mesos.CommandInfo_URI{
					&mesos.CommandInfo_URI{
						Value:      &(sc.schedulerHTTPServer.hostURI),
						Executable: proto.Bool(false),
					},
					&mesos.CommandInfo_URI{
						Value:      &(sc.schedulerHTTPServer.riakURI),
						Executable: proto.Bool(false),
					},
					&mesos.CommandInfo_URI{
						Value:      &(sc.schedulerHTTPServer.cepmdURI),
						Executable: proto.Bool(true),
					},
				},
				Shell:     proto.Bool(ExecutorShell()),
				Arguments: ExecutorArgs(frn.CurrentID()),
			},
			Resources: execAsk,
		},
		Resources: taskAsk,
		Data:      binTaskData,
	}

	offerHelper.TasksToLaunch = append(offerHelper.TasksToLaunch, taskInfo)

	return true
}

func (frn *FrameworkRiakNode) GetTaskStatus() *mesos.TaskStatus {
	if frn.TaskStatus != nil {
		// SlaveIDs can change during failure, don't make that a part of the reconcilliation
		return &mesos.TaskStatus{
			TaskId:  frn.TaskStatus.TaskId,
			State:   frn.TaskStatus.State,
			SlaveId: &mesos.SlaveID{Value: proto.String("")},
		}
	}

	if frn.CurrentState <= process_state.Reserved {
		return nil
	}

	ts := mesos.TaskState_TASK_ERROR
	return &mesos.TaskStatus{
		TaskId:  &mesos.TaskID{Value: proto.String(frn.CurrentID())},
		State:   &ts,
		SlaveId: &mesos.SlaveID{Value: proto.String("")}, // Slave ID isn't required
	}
}

// --- State ---

// Getters
func (frn *FrameworkRiakNode) HasRequestedReservation() bool {
	return frn.CurrentState >= process_state.Reserved
}

func (frn *FrameworkRiakNode) NeedsToBeReconciled() bool {
	return !frn.reconciled &&
		frn.CurrentState != process_state.Unknown &&
		frn.CurrentState != process_state.Reserved
}

func (frn *FrameworkRiakNode) CanBeScheduled() bool {
	if frn.NeedsToBeReconciled() {
		return false
	}

	switch frn.DestinationState {
	case process_state.Restarting:
		{
			switch frn.CurrentState {
			case process_state.Started:
				return false
			case process_state.Unknown:
				return true
			case process_state.Reserved:
				return true
			case process_state.Starting:
				return false
			case process_state.Shutdown:
				return true
			case process_state.Failed:
				return true
			}
		}
	case process_state.Started:
		{
			switch frn.CurrentState {
			case process_state.Started:
				return false
			case process_state.Unknown:
				return true
			case process_state.Reserved:
				return true
			case process_state.Starting:
				return false
			case process_state.Shutdown:
				return false
			case process_state.Failed:
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
		frn.CurrentState != process_state.Started
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
func (frn *FrameworkRiakNode) HasRestarted(generation int64) bool {
	return frn.DestinationState == process_state.Started &&
		frn.CurrentState == process_state.Started &&
		frn.RestartGeneration >= generation
}
func (frn *FrameworkRiakNode) IsRestarting(generation int64) bool {
	return (frn.DestinationState == process_state.Started || frn.DestinationState == process_state.Restarting) &&
		frn.CurrentState != process_state.Started &&
		frn.RestartGeneration >= generation
}

// Setters
func (frn *FrameworkRiakNode) Restart(generation int64) {
	frn.RestartGeneration = generation
	frn.DestinationState = process_state.Restarting
}

func (frn *FrameworkRiakNode) Unreserve() {
	frn.CurrentState = process_state.Unknown
	frn.SlaveID = nil
}
func (frn *FrameworkRiakNode) KillNext() {
	frn.DestinationState = process_state.Shutdown
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
	frn.DestinationState = process_state.Started
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
	frn.CurrentState = process_state.Failed
}
