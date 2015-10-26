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
	sc.schedulerState.MesosMaster = mesosMaster
	var frameworkId *mesos.FrameworkID
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

func FindNodeForOffer(offer *mesos.Offer, nodes []*FrameworkRiakNode) (*FrameworkRiakNode, []*FrameworkRiakNode) {
	for _, riakNode := range nodes {
		// Fresh node
		if !riakNode.HasRequestedReservation() {
			log.Infof("Found a new offer for a node. Offer: %+v, Node: %+v", offer, riakNode)
			return riakNode, nodes
		}
	}

	// log.Infof("Couldn't find any nodes suitable for an unreserved offer. Offer: %+v, Nodes: %+v", offer, nodes)
	return nil, nodes
}

func FindNodeForReservation(offerWithReservation *mesos.Offer, nodes []*FrameworkRiakNode) (*FrameworkRiakNode, []*FrameworkRiakNode) {
	for _, riakNode := range nodes {
		if riakNode.HasRequestedReservation() && riakNode.OfferCompatible(offerWithReservation) {
			log.Infof("Found an offer with a reservation for a node. Offer: %+v, Node: %+v", offerWithReservation, riakNode)
			return riakNode, nodes
		}
	}

	log.Warnf("Found an offer with a reservation, but no nodes were compatible with it. Offer: %+v, Nodes: %+v", offerWithReservation, nodes)
	return nil, nodes
}

func (sc *SchedulerCore) AddNodeToLaunchTasks(riakNode *FrameworkRiakNode, launchTasks map[string][]*mesos.TaskInfo) map[string][]*mesos.TaskInfo {
	taskInfo := riakNode.PrepareForLaunchAndGetNewTaskInfo(sc)
	sc.frnDict[riakNode.CurrentID()] = riakNode

	if launchTasks[*riakNode.LastOfferUsed.Id.Value] == nil {
		launchTasks[*riakNode.LastOfferUsed.Id.Value] = []*mesos.TaskInfo{}
	}

	log.Infof("Using offerId: %+v, for riakNode.CurrentID(): %+v", *riakNode.LastOfferUsed.Id.Value, riakNode.CurrentID())

	launchTasks[*riakNode.LastOfferUsed.Id.Value] = append(launchTasks[*riakNode.LastOfferUsed.Id.Value], taskInfo)
	return launchTasks
}

func (sc *SchedulerCore) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	log.Info("Received resource offers: ", offers)
	launchTasks := make(map[string][]*mesos.TaskInfo)
	nodesWithAcceptedOffers := make(map[string][]*FrameworkRiakNode)
	toBeScheduled := []*FrameworkRiakNode{}

	// Find nodes which need to be scheduled
	for _, cluster := range sc.schedulerState.Clusters {
		for _, riakNode := range cluster.Nodes {
			if riakNode.NeedsToBeScheduled() {
				log.Infof("Adding Riak node for scheduling: %+v", riakNode)
				// We need to schedule this task I guess?
				toBeScheduled = append(toBeScheduled, riakNode)
			}
		}
	}

	// Populate launchTasks and nodesWithAcceptedOffers
	for _, offer := range offers {
		// Someone has claimed this offer
		if common.ResourcesHaveReservations(offer.Resources) {
			var riakNode *FrameworkRiakNode
			riakNode, toBeScheduled = FindNodeForReservation(offer, toBeScheduled)
			if riakNode != nil {
				// Assign the updated offer and make sure we still fit
				var applySuccess bool
				applySuccess, offer = riakNode.ApplyOffer(offer)
				if applySuccess && riakNode.HasReservation() {
					// Can Launch
					launchTasks = sc.AddNodeToLaunchTasks(riakNode, launchTasks)
					sc.schedulerState.Persist()
				}
			}
		} else {
			// Resource does not have any reservations at this point, need to see if there are any nodes that still need reservations
			var riakNode *FrameworkRiakNode
			riakNode, toBeScheduled = FindNodeForOffer(offer, toBeScheduled)
			if riakNode != nil {
				var applySuccess bool
				applySuccess, offer = riakNode.ApplyOffer(offer)
				if applySuccess {
					// Can Reserve
					nodesWithAcceptedOffers[*offer.Id.Value] = append(nodesWithAcceptedOffers[*offer.Id.Value], riakNode)
					sc.schedulerState.Persist()
				}
			}
		}
	}

	// Attempt to reserve resources and/or launch nodes
	for _, offer := range offers {
		tasks := launchTasks[*offer.Id.Value]
		nodesToReserve := nodesWithAcceptedOffers[*offer.Id.Value]

		if nodesToReserve != nil {
			log.Infof("Resource Offers: In reserve loop, currently on offerId: %+v, tasks for offer: %+v", *offer.Id.Value, tasks)
			// Note, there should only ever be one node or one task per offer id
			driver.CleanOffers([]*mesos.OfferID{offer.Id})
			for _, riakNode := range nodesToReserve {
				mesosHttpClient := NewMesosClient(sc.schedulerState.MesosMaster, sc.schedulerState.FrameworkID, OFFER_INTERVAL)
				reserveSuccess, err := mesosHttpClient.ReserveResourceAndCreateVolume(riakNode)
				if !reserveSuccess || err != nil {
					log.Warnf("Failed to reserve resources / create volumes. Error: %+v, Node: %+v, Offer: %+v", err, riakNode, offer)
				}
			}
			continue
		}

		if tasks == nil {
			tasks = []*mesos.TaskInfo{}
		}

		// This is somewhat of a hack, to avoid synchronously calling the mesos-go SDK
		// to avoid a deadlock situation.
		// TODO: Fix and make actually queues around driver interactions
		// This is a massive hack
		// -Sargun Dhillon 2015-10-01
		go func(innerOffer *mesos.Offer, innerTasks []*mesos.TaskInfo) {
			log.Infof("Launching tasks on offerId: %+v, tasks for offer: %+v", *innerOffer.Id.Value, innerTasks)
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
