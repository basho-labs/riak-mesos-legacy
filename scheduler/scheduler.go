package scheduler

import (
	"io/ioutil"
	"sync"

	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
	"github.com/basho-labs/riak-mesos/cepmd/cepm"
	"github.com/basho-labs/riak-mesos/common"
	metamgr "github.com/basho-labs/riak-mesos/metadata_manager"
	//rex "github.com/basho-labs/riak-mesos/riak_explorer"
	"github.com/golang/protobuf/proto"
	auth "github.com/mesos/mesos-go/auth"
	sasl "github.com/mesos/mesos-go/auth/sasl"
	mesos "github.com/mesos/mesos-go/mesosproto"
	sched "github.com/mesos/mesos-go/scheduler"
	"time"
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
	cepm                *cepm.CEPM
	frameworkName       string
	frameworkRole       string
	NodeCpus            string
	NodeMem             string
	NodeDisk            string
	schedulerState      *SchedulerState
	authProvider        string
	mesosAuthPrincipal  string
	mesosAuthSecretFile string
	mesosHttpClient     *MesosClient
}

func NewSchedulerCore(
	schedulerHostname string,
	frameworkName string,
	frameworkRole string,
	zookeepers []string,
	schedulerIPAddr string,
	user string,
	nodeCpus string,
	nodeMem string,
	nodeDisk string,
	authProvider string,
	mesosAuthPrincipal string,
	mesosAuthSecretFile string) *SchedulerCore {

	mgr := metamgr.NewMetadataManager(frameworkName, zookeepers)
	ss := GetSchedulerState(mgr)

	c := cepm.NewCPMd(0, mgr)
	c.Background()

	mesosHttpClient := NewMesosClient(ss.MesosMaster)

	scheduler := &SchedulerCore{
		lock:            &sync.Mutex{},
		schedulerIPAddr: schedulerIPAddr,
		mgr:             mgr,
		frnDict:         make(map[string]*FrameworkRiakNode),
		user:            user,
		zookeepers:      zookeepers,
		cepm:            c,
		frameworkName:   frameworkName,
		frameworkRole:   frameworkRole,
		NodeCpus:        nodeCpus,
		NodeMem:         nodeMem,
		NodeDisk:        nodeDisk,
		schedulerState:  ss,
	}
	scheduler.schedulerHTTPServer = ServeExecutorArtifact(scheduler, schedulerHostname)
	return scheduler
}

func (sc *SchedulerCore) Run(mesosMaster string) {
	var frameworkId *mesos.FrameworkID
	sc.schedulerState.MesosMaster = mesosMaster
	if sc.schedulerState.FrameworkID == nil {
		frameworkId = nil
	} else {
		frameworkId = &mesos.FrameworkID{
			Value: sc.schedulerState.FrameworkID,
		}
	}

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

	cred := (*mesos.Credential)(nil)
	if sc.mesosAuthPrincipal != "" {
		fwinfo.Principal = proto.String(sc.mesosAuthPrincipal)
		secret, err := ioutil.ReadFile(sc.mesosAuthSecretFile)
		if err != nil {
			log.Fatal(err)
		}
		cred = &mesos.Credential{
			Principal: proto.String(sc.mesosAuthPrincipal),
			Secret:    secret,
		}
	}

	config := sched.DriverConfig{
		Scheduler:  sc,
		Framework:  fwinfo,
		Master:     mesosMaster,
		Credential: cred,
	}

	if sc.schedulerIPAddr != "" {
		config.BindingAddress = parseIP(sc.schedulerIPAddr)
	}

	if sc.mesosAuthPrincipal != "" {
		config.WithAuthContext = func(ctx context.Context) context.Context {
			ctx = auth.WithLoginProvider(ctx, sc.authProvider)
			if sc.schedulerIPAddr != "" {
				ctx = sasl.WithBindingAddress(ctx, parseIP(sc.schedulerIPAddr))
			}
			return ctx
		}
	}

	log.Infof("Running scheduler with FrameworkInfo: %v and DriverConfig: %v", fwinfo, config)

	driver, err := sched.NewMesosSchedulerDriver(config)
	if err != nil {
		log.Error("Unable to create a SchedulerDriver ", err.Error())
	}
	sc.rServer = newReconciliationServer(driver, sc)

	sc.mgr.SetupFramework(sc.schedulerHTTPServer.URI)

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

func (sc *SchedulerCore) spreadNodesAcrossOffers(allOffers []*mesos.Offer, allResources [][]*mesos.Resource, allNodes []*FrameworkRiakNode, currentOfferIndex int, currentRiakNodeIndex int, acceptedOffers []*AcceptOfferInfo, launchTasks map[string][]*mesos.TaskInfo) ([]*AcceptOfferInfo, []*mesos.Offer, map[string][]*mesos.TaskInfo, error) {
	log.Infof("spreadNodesAcrossOffers: currentOfferIndex: %+v, currentRiakNodeIndex: %+v", currentOfferIndex, currentRiakNodeIndex)
	log.Infof("spreadNodesAcrossOffers: allNodes: %+v, allResources: %+v, allOffers: %+v, launchTasks: %+v", len(allNodes), len(allResources), len(allOffers), len(launchTasks))

	// Nothing to launch
	if len(allNodes) == 0 || len(allResources) == 0 || len(allOffers) == 0 {
		return acceptedOffers, launchTasks, allOffers, nil
	}

	// No more nodes to schedule
	if currentRiakNodeIndex >= len(allNodes) {
		return acceptedOffers, launchTasks, allOffers, nil
	}

	// No more offers, just get out now, more offers will come
	if currentOfferIndex >= len(allOffers) {
		return acceptedOffers, launchTasks, allOffers, nil
	}

	offer := allOffers[currentOfferIndex]
	riakNode := allNodes[currentRiakNodeIndex]

	var success bool
	var executorAsk, taskAsk []*mesos.Resource
	allResources[currentOfferIndex], executorAsk, taskAsk, success = riakNode.GetCombinedAsk(sc)(allResources[currentOfferIndex])

	if success && !riakNode.hasReservation {
		acceptInfo := riakNode.PrepareForReservation(sc, offer, executorAsk, taskAsk)
		acceptedOffers = append(acceptedOffers, acceptInfo)
		//Remove from the pool of offers so we don't confuse mesos-go
		allOffers = append(allOffers[:currentOfferIndex], allOffers[currentOfferIndex+1:]...)
		return sc.spreadNodesAcrossOffers(allOffers, allResources, allNodes, currentOfferIndex, currentRiakNodeIndex+1, acceptedOffers, launchTasks)
	} else if success && riakNode.hasReservation {
		taskInfo := riakNode.PrepareForLaunchAndGetNewTaskInfo(sc, offer, executorAsk, taskAsk)
		sc.frnDict[riakNode.CurrentID()] = riakNode

		if launchTasks[*offer.Id.Value] == nil {
			launchTasks[*offer.Id.Value] = []*mesos.TaskInfo{}
		}

		log.Infof("spreadNodesAcrossOffers: Using offerId: %+v, for riakNode.CurrentID(): %+v", *offer.Id.Value, riakNode.CurrentID())

		launchTasks[*offer.Id.Value] = append(launchTasks[*offer.Id.Value], taskInfo)

		log.Infof("spreadNodesAcrossOffers: New LaunchTasks: %+v", launchTasks)

		sc.schedulerState.Persist()

		// Everything went well, add to the launch tasks
		return sc.spreadNodesAcrossOffers(allOffers, allResources, allNodes, currentOfferIndex+1, currentRiakNodeIndex+1, acceptedOffers, launchTasks)
	}

	return sc.spreadNodesAcrossOffers(allOffers, allResources, allNodes, currentOfferIndex+1, currentRiakNodeIndex, acceptedOffers, launchTasks)
}

func (sc *SchedulerCore) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	log.Info("Received resource offers: ", offers)
	launchTasks := make(map[string][]*mesos.TaskInfo)
	acceptedOffers := make([]acceptedOffers)
	toBeScheduled := []*FrameworkRiakNode{}
	for _, cluster := range sc.schedulerState.Clusters {
		for _, riakNode := range cluster.Nodes {
			if riakNode.NeedsToBeScheduled() {
				log.Infof("Adding Riak node for scheduling: %+v", riakNode)
				// We need to schedule this task I guess?
				toBeScheduled = append(toBeScheduled, riakNode)
			}
		}
	}

	for _, offer := range offers {
		for _, riakNode := range toBeScheduled {
			// The state of the nodes will be modified inside the loop, so ignore starting nodes
			if !riakNode.NeedsToBeScheduled() {
				continue
			}

			// Someone has claimed this offer, was it me?
			if common.ResourcesHaveReservations(offer.Resources) {
				if riakNode.OfferCompatible(offer) {
					// Assign the updated offer and make sure we still fit
					applySuccess, offer = riakNode.ApplyOffer(offer)
					if applySuccess {
						// Launch

					}
				}
				continue
			}

			// Fresh node, apply offer
			if !riakNode.HasReservation() {
				applySuccess, offer = riakNode.ApplyOffer(offer)
				if applySuccess {
					// Reserve
					sc.mesosHttpClient.ReserveResourceAndCreateVolume(riakNode)
				}
			}
		}
	}

	// Populate a mutable slice of offer resources
	allResources := [][]*mesos.Resource{}
	copiedResources := *[]mesos.Resource{}
	for _, offer := range offers {
		copy(copiedResources, offer.Resources)
		allResources = append(allResources, copiedResources)
	}

	acceptedOffers, launchOffers, launchTasks, err := sc.spreadNodesAcrossOffers(offers, allResources, toBeScheduled, 0, 0, acceptedOffers, launchTasks)
	if err != nil {
		log.Error(err)
	}

	mesosClient = NewMesosClient(sc.schedulerState.MesosMaster)

	for _, acceptInfo := range acceptedOffers {
		oid := &mesos.OfferID{
			Value: acceptInfo.offerID,
		}
		driver.CleanOffers([]*mesos.OfferID{oid})
		mesosClient.ReserveResourceAndCreateVolume(acceptInfo)
	}

	for _, offer := range launchOffers {
		tasks := launchTasks[*offer.Id.Value]

		if tasks == nil {
			tasks = []*mesos.TaskInfo{}
		}

		log.Infof("Resource Offers: In launch loop, currently on offerId: %+v, tasks for offer: %+v", *offer.Id.Value, tasks)

		// This is somewhat of a hack, to avoid synchronously calling the mesos-go SDK
		// to avoid a deadlock situation.
		// TODO: Fix and make actually queues around driver interactions
		// This is a massive hack
		// -Sargun Dhillon 2015-10-01
		go func(innerOffer *mesos.Offer, innerTasks []*mesos.TaskInfo) {
			log.Infof("Resource Offers: In launch loop inner go func, currently on offerId: %+v, tasks for offer: %+v", *innerOffer.Id.Value, innerTasks)
			innerStatus, innerErr := driver.LaunchTasks([]*mesos.OfferID{innerOffer.Id}, innerTasks, &mesos.Filters{RefuseSeconds: proto.Float64(OFFER_INTERVAL)})

			if innerStatus != mesos.Status_DRIVER_RUNNING {
				log.Fatal("Driver not running, while trying to launch tasks")
			}
			if innerErr != nil {
				log.Panic("Failed to launch tasks: ", innerErr)
			}
		}(offer, tasks)
	}
}
func (sc *SchedulerCore) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	riak_node, assigned := sc.frnDict[status.TaskId.GetValue()]
	if assigned {
		log.Info("Received status updates: ", status)
		log.Info("Riak Node: ", riak_node)
		riak_node.handleStatusUpdate(sc, sc.schedulerState.Clusters[riak_node.ClusterName], status)
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

// Callback from reconciliation server
// This is a massive hack that was because I didn't want to make the scheduler async
func (sc *SchedulerCore) GetTasksToReconcile() []*mesos.TaskStatus {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	tasksToReconcile := []*mesos.TaskStatus{}
	for _, cluster := range sc.schedulerState.Clusters {
		for _, node := range cluster.Nodes {
			if node.reconciled == false && time.Since(node.lastAskedToReconcile).Seconds() > 5 {
				if _, assigned := sc.frnDict[node.GetTaskStatus().TaskId.GetValue()]; !assigned {
					sc.frnDict[node.GetTaskStatus().TaskId.GetValue()] = node
				}
				node.lastAskedToReconcile = time.Now()
				tasksToReconcile = append(tasksToReconcile, node.GetTaskStatus())
			}
		}
	}
	return tasksToReconcile
}
