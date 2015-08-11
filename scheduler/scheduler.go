package scheduler

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/basho-labs/riak-mesos/cepmd/cepm"
	metamgr "github.com/basho-labs/riak-mesos/metadata_manager"
	rex "github.com/basho-labs/riak-mesos/riak_explorer"
	"github.com/golang/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	sched "github.com/mesos/mesos-go/scheduler"
	"github.com/satori/go.uuid"
)

const (
	OFFER_INTERVAL float64 = 5
)

type SchedulerCore struct {
	lock                *sync.Mutex
	schedulerHTTPServer *SchedulerHTTPServer
	mgr                 *metamgr.MetadataManager
	schedulerIPAddr     string
	frnDict             map[string]*FrameworkRiakNode
	rServer             *ReconcilationServer
	user                string
	zookeepers          []string
	rex                 *rex.RiakExplorer
	rexPort             int
	cepm                *cepm.CEPM
	frameworkName       string
	frameworkRole       string
	schedulerState      *SchedulerState
}

func NewSchedulerCore(schedulerHostname string, frameworkName string, frameworkRole string, zookeepers []string, schedulerIPAddr string, user string, rexPort int) *SchedulerCore {
	mgr := metamgr.NewMetadataManager(frameworkName, zookeepers)
	ss := GetSchedulerState(mgr)
	hostname, err := os.Hostname()
	if err != nil {
		log.Panic("Could not get hostname")
	}
	nodename := fmt.Sprintf("rex-%s@%s", uuid.NewV4().String(), hostname)

	if !strings.Contains(nodename, ".") {
		nodename = nodename + "."
	}

	c := cepm.NewCPMd(0, mgr)
	c.Background()

	myRex, err := rex.NewRiakExplorer(int64(rexPort), nodename, c)
	if err != nil {
		log.Fatal("Could not start up Riak Explorer in scheduler")
	}
	scheduler := &SchedulerCore{
		lock:            &sync.Mutex{},
		schedulerIPAddr: schedulerIPAddr,
		mgr:             mgr,
		frnDict:         make(map[string]*FrameworkRiakNode),
		user:            user,
		zookeepers:      zookeepers,
		rex:             myRex,
		rexPort:         rexPort,
		cepm:            c,
		frameworkName:   frameworkName,
		frameworkRole:   frameworkRole,
		schedulerState:  ss,
	}
	scheduler.schedulerHTTPServer = ServeExecutorArtifact(scheduler, schedulerHostname)
	return scheduler
}


func (sc *SchedulerCore) setupMetadataManager() {
	sc.mgr.SetupFramework(sc.schedulerHTTPServer.URI)
}
func (sc *SchedulerCore) Run(mesosMaster string) {
	var frameworkId *mesos.FrameworkID
	if sc.schedulerState.FrameworkID == nil {
		frameworkId = nil
	} else {
		frameworkId = &mesos.FrameworkID{
			Value: sc.schedulerState.FrameworkID,
		}
	}

	// TODO: Get "Real" credentials here

	cred := (*mesos.Credential)(nil)
	fwinfo := &mesos.FrameworkInfo{
		Name:            proto.String(sc.frameworkName),
		Id:              frameworkId,
		FailoverTimeout: proto.Float64(86400),
		WebuiUrl:        proto.String(sc.schedulerHTTPServer.GetURI()),
		Checkpoint:      proto.Bool(true),
		Role:            proto.String(sc.frameworkRole),
	}

	if sc.user != "" {
		fwinfo.User = proto.String(sc.user)
	} else {
		guestUser := "guest"
		fwinfo.User = &guestUser
	}

	log.Info("Running scheduler with FrameworkInfo: ", fwinfo)

	config := sched.DriverConfig{
		Scheduler:  sc,
		Framework:  fwinfo,
		Master:     mesosMaster,
		Credential: cred,
		// BindingAddress: bindingAddress,
		//	WithAuthContext: func(ctx context.Context) context.Context {
		//		ctx = auth.WithLoginProvider(ctx, *authProvider)
		//		ctx = sasl.WithBindingAddress(ctx, bindingAddress)
		//		return ctx
		//	},
	}

	if sc.schedulerIPAddr != "" {
		config.BindingAddress = parseIP(sc.schedulerIPAddr)
	}

	driver, err := sched.NewMesosSchedulerDriver(config)
	if err != nil {
		log.Error("Unable to create a SchedulerDriver ", err.Error())
	}
	sc.rServer = newReconciliationServer(driver, sc)

	sc.setupMetadataManager()

	if stat, err := driver.Run(); err != nil {
		log.Infof("Framework stopped with status %s and error: %s\n", stat.String(), err.Error())
	}
}

func (sc *SchedulerCore) Registered(driver sched.SchedulerDriver, frameworkId *mesos.FrameworkID, masterInfo *mesos.MasterInfo) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	log.Info("Framework registered")
	log.Info("Framework ID: ", frameworkId)
	log.Info("Master Info: ", masterInfo)
	sc.schedulerState.FrameworkID = frameworkId.Value
	if err := sc.schedulerState.Persist(); err != nil {
		log.Error("Unable to persist framework ID after startup")
	}
	sc.rServer.enable()
}

func (sc *SchedulerCore) Reregistered(driver sched.SchedulerDriver, masterInfo *mesos.MasterInfo) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	//go NewTargetTask(*sched).Loop()
	// We don't actually handle this correctly
	log.Error("Framework reregistered")
	log.Info("Master Info: ", masterInfo)
	sc.rServer.enable()
}
func (sc *SchedulerCore) Disconnected(sched.SchedulerDriver) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	log.Error("Framework disconnected")
}

func (sc *SchedulerCore) spreadNodesAcrossOffers(allOffers []*mesos.Offer, allResources [][]*mesos.Resource, allNodes []*FrameworkRiakNode, currentOfferIndex int, currentRiakNodeIndex int, launchTasks map[string][]*mesos.TaskInfo) (map[string][]*mesos.TaskInfo, error) {
	if len(allNodes) == 0 || len(allResources) == 0 {
		return launchTasks, nil
	}

	// No more nodes to schedule
	if currentRiakNodeIndex >= len(allNodes) {
		return launchTasks, nil
	}

	// No more offers, start from the beginning (round robin)
	if currentOfferIndex >= len(allResources) {
		return sc.spreadNodesAcrossOffers(allOffers, allResources, allNodes, 0, currentRiakNodeIndex, launchTasks)
	}

	offer := allOffers[currentOfferIndex]
	riakNode := allNodes[currentRiakNodeIndex]

	var success bool
	var executorAsk, taskAsk []*mesos.Resource
	allResources[currentOfferIndex], executorAsk, taskAsk, success = riakNode.GetCombinedAsk()(allResources[currentOfferIndex])

	if success {
		taskInfo := riakNode.PrepareForLaunchAndGetNewTaskInfo(sc, offer, executorAsk, taskAsk)
		sc.frnDict[riakNode.CurrentID()] = riakNode

		if launchTasks[*offer.Id.Value] == nil {
			launchTasks[*offer.Id.Value] = []*mesos.TaskInfo{}
		}

		launchTasks[*offer.Id.Value] = append(launchTasks[*offer.Id.Value], taskInfo)
		sc.schedulerState.Persist()

		// Everything went well, add to the launch tasks
		allNodes = append(allNodes[:currentRiakNodeIndex], allNodes[currentRiakNodeIndex+1:]...)
		return sc.spreadNodesAcrossOffers(allOffers, allResources, allNodes, currentOfferIndex+1, currentRiakNodeIndex+1, launchTasks)
	}

	// There are no more offers with sufficient resources for a single node
	if len(allResources) <= 1 {
		return launchTasks, errors.New("Not enough resources to schedule RiakNode")
	}

	// This offer no longer has sufficient resources available, remove it from the pool
	allOffers = append(allOffers[:currentOfferIndex], allOffers[currentOfferIndex+1:]...)
	allResources = append(allResources[:currentOfferIndex], allResources[currentOfferIndex+1:]...)
	return sc.spreadNodesAcrossOffers(allOffers, allResources, allNodes, currentOfferIndex+1, currentRiakNodeIndex, launchTasks)
}

func (sc *SchedulerCore) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	log.Info("Received resource offers: ", offers)
	launchTasks := make(map[string][]*mesos.TaskInfo)
	toBeScheduled := []*FrameworkRiakNode{}

	for _, riakNode := range sc.schedulerState.Nodes {
		if riakNode.NeedsToBeScheduled() {
			log.Infof("Adding Riak node for scheduling: %+v", riakNode)
			// We need to schedule this task I guess?
			toBeScheduled = append(toBeScheduled, riakNode)
		}
	}

	// Populate a mutable slice of offer resources
	allResources := [][]*mesos.Resource{}
	for _, offer := range offers {
		allResources = append(allResources, offer.Resources)
	}

	launchTasks, err := sc.spreadNodesAcrossOffers(offers, allResources, toBeScheduled, 0, 0, launchTasks)
	if err != nil {
		log.Error(err)
	}

	for _, offer := range offers {
		tasks := launchTasks[*offer.Id.Value]

		if tasks == nil {
			tasks = []*mesos.TaskInfo{}
		}

		log.Infof("Launching Tasks: %v for offer %v", tasks, *offer.Id.Value)
		status, err := driver.LaunchTasks([]*mesos.OfferID{offer.Id}, tasks, &mesos.Filters{RefuseSeconds: proto.Float64(OFFER_INTERVAL)})

		if status != mesos.Status_DRIVER_RUNNING {
			log.Fatal("Driver not running, while trying to launch tasks")
		}
		if err != nil {
			log.Panic("Failed to launch tasks: ", err)
		}
	}
}
func (sc *SchedulerCore) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	riak_node, assigned := sc.frnDict[status.TaskId.GetValue()]
	if assigned {
		log.Info("Received status updates: ", status)
		log.Info("Riak Node: ", riak_node)
		riak_node.handleStatusUpdate(sc, status)
		sc.schedulerState.Persist()
	} else {
		log.Error("Received status update for unknown job: ", status)
	}

}

func (sc *SchedulerCore) OfferRescinded(driver sched.SchedulerDriver, offerID *mesos.OfferID) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	log.Info("Offer rescinded from Mesos")
}

func (sc *SchedulerCore) FrameworkMessage(driver sched.SchedulerDriver, executorID *mesos.ExecutorID, slaveID *mesos.SlaveID, message string) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	log.Info("Got unknown framework message %v")
}

// TODO: Write handler
func (sc *SchedulerCore) SlaveLost(sched.SchedulerDriver, *mesos.SlaveID) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	log.Info("Slave Lost")
}

// TODO: Write handler
func (sc *SchedulerCore) ExecutorLost(sched.SchedulerDriver, *mesos.ExecutorID, *mesos.SlaveID, int) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	log.Info("Executor Lost")
}

func (sc *SchedulerCore) Error(driver sched.SchedulerDriver, err string) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	log.Info("Scheduler received error:", err)
}
