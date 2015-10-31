package scheduler

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	mesos "github.com/basho-labs/mesos-go/mesosproto"
	util "github.com/basho-labs/mesos-go/mesosutil"
	"github.com/basho-labs/riak-mesos/common"
	"github.com/basho-labs/riak-mesos/scheduler/process_state"
	"github.com/golang/protobuf/proto"
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
	CONTAINER_PATH    = "root"
)

type FrameworkRiakNode struct {
	// This is super hacky, we're relying on the following to be NOT serialized, and defaults. FIX THIS. Somehow..
	reconciled           bool      `json:"-"`
	lastAskedToReconcile time.Time `json:"-"`

	SimpleId         int
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
	UUID             string
	ContainerPath    string
}

func NewFrameworkRiakNode(sc *SchedulerCore, clusterName string, simpleId int) *FrameworkRiakNode {
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
		DestinationState: process_state.Started,
		CurrentState:     process_state.Unknown,
		Generation:       0,
		reconciled:       false,
		FrameworkName:    sc.frameworkName,
		Role:             &sc.frameworkRole,
		Principal:        &sc.mesosAuthPrincipal,
		SimpleId:         simpleId,
		ClusterName:      clusterName,
		Cpus:             nodeCpusFloat,
		Mem:              nodeMemFloat,
		Disk:             nodeDiskFloat,
		Ports:            PORTS_PER_TASK,
		ExecCpus:         CPUS_PER_EXECUTOR,
		ExecMem:          MEM_PER_EXECUTOR,
		UUID:             uuid.NewV4().String(),
		ContainerPath:    CONTAINER_PATH,
	}
}

func (frn *FrameworkRiakNode) GetExecutorResources() []*mesos.Resource {
	return frn.GetReservedResources(frn.ExecCpus, frn.ExecMem, 0, false, 0)
}

func (frn *FrameworkRiakNode) GetTaskResources() []*mesos.Resource {
	return frn.GetReservedResources(frn.Cpus, frn.Mem, frn.Disk, true, frn.Ports)
}

func (frn *FrameworkRiakNode) GetResourcesToReserve() []*mesos.Resource {
	return frn.GetReservedResources(frn.Cpus+frn.ExecCpus, frn.Mem+frn.ExecMem, frn.Disk, false, 0)
}

func (frn *FrameworkRiakNode) GetResourcesToCreate() []*mesos.Resource {
	return frn.GetReservedResources(0, 0, frn.Disk, true, 0)
}

func (frn *FrameworkRiakNode) GetReservedResources(cpusValue float64, memValue float64, diskValue float64, includeVolume bool, portsValue int) []*mesos.Resource {
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
		persistenceId := frn.PersistenceID()
		persistence := &mesos.Resource_DiskInfo_Persistence{
			Id: &persistenceId,
		}
		info := &mesos.Resource_DiskInfo{
			Persistence: persistence,
			Volume:      volume,
		}
		disk := util.NewScalarResource("disk", diskValue)
		disk.Role = frn.Role
		disk.Reservation = reservation
		if includeVolume {
			disk.Disk = info
		}
		resources = append(resources, disk)
	}

	if portsValue > 0 {
		var ports *mesos.Resource
		frn.LastOfferUsed.Resources, ports = common.ApplyRangesResource(frn.LastOfferUsed.Resources, portsValue)
		resources = append(resources, ports)
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
	resources := []*mesos.Resource{}
	if frn.HasRequestedReservation() {
		resources = common.FilterReservedResources(mutableOffer.Resources)
	} else {
		resources = common.FilterUnreservedResources(mutableOffer.Resources)
	}

	if !common.ScalarResourcesWillFit(resources, frn.Cpus+frn.ExecCpus, frn.Mem+frn.ExecMem, frn.Disk) ||
		!common.PortResourceWillFit(mutableOffer.Resources, frn.Ports) {
		log.Info("Attempted to apply offer but offer does not have enough capacity")
		return false, mutableOffer
	}

	mutableOffer.Resources = common.ApplyScalarResources(mutableOffer.Resources, frn.Cpus+frn.ExecCpus, frn.Mem+frn.ExecMem, frn.Disk)
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

func (frn *FrameworkRiakNode) OfferCompatible(immutableOffer *mesos.Offer) bool {
	log.Infof("Checking if offer is compatible for reservation. Offer's SlaveId: %+v, Node's last SlaveId: %+v", immutableOffer.SlaveId.GetValue(), frn.LastOfferUsed.SlaveId.GetValue())
	if !immutableOffer.SlaveId.Equal(frn.LastOfferUsed.SlaveId) {
		return false
	}

	for _, resource := range util.FilterResources(immutableOffer.Resources, func(res *mesos.Resource) bool { return res.GetName() == "disk" && res.Reservation != nil }) {
		if resource.Disk != nil && *resource.Disk.Persistence.Id == frn.PersistenceID() {
			return true
		}
	}

	return false
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

	execName := fmt.Sprintf("%s Executor", frn.CurrentID())
	exec := &mesos.ExecutorInfo{
		ExecutorId: frn.CreateExecutorID(),
		Name:       proto.String(execName),
		Source:     proto.String(frn.FrameworkName),
		Command: &mesos.CommandInfo{
			Value:     proto.String(sc.schedulerHTTPServer.executorName),
			Uris:      executorUris,
			Shell:     proto.Bool(false),
			Arguments: []string{sc.schedulerHTTPServer.executorName, "-logtostderr=true", "-taskinfo", frn.CurrentID()},
		},
		Resources: executorAsk,
	}
	taskId := frn.CreateTaskID()
	nodename := frn.CurrentID() + "@" + offer.GetHostname()
	if !strings.Contains(offer.GetHostname(), ".") {
		nodename = nodename + "."
	}
	ports := portIter(taskAsk)

	taskData := common.TaskData{
		FullyQualifiedNodeName: nodename,
		Host:           offer.GetHostname(),
		Zookeepers:     sc.zookeepers,
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
	return fmt.Sprintf("%s-%s-%d", frn.FrameworkName, frn.ClusterName, frn.SimpleId)
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

func (frn *FrameworkRiakNode) GetTaskStatus() *mesos.TaskStatus {
	if frn.TaskStatus != nil {
		return frn.TaskStatus
	}

	// A nil task status doesn't mean there's an error, it could mean the node hasn't started yet.
	return nil
	// ts := mesos.TaskState_TASK_ERROR
	// return &mesos.TaskStatus{
	// 	TaskId:  &mesos.TaskID{Value: proto.String(frn.CurrentID())},
	// 	State:   &ts,
	// 	SlaveId: &mesos.SlaveID{Value: proto.String("")}, // Slave ID isn't required
	// }
}
