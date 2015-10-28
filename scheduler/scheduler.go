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
	auth "github.com/basho-labs/mesos-go/auth"
	sasl "github.com/basho-labs/mesos-go/auth/sasl"
	mesos "github.com/basho-labs/mesos-go/mesosproto"
	sched "github.com/basho-labs/mesos-go/scheduler"
	"github.com/golang/protobuf/proto"
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
	compatibilityMode   bool
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
	mesosAuthSecretFile string,
	useReservations bool) *SchedulerCore {

	mgr := metamgr.NewMetadataManager(frameworkName, zookeepers)
	ss := GetSchedulerState(mgr)

	c := cepm.NewCPMd(0, mgr)
	c.Background()

	scheduler := &SchedulerCore{
		lock:                &sync.Mutex{},
		schedulerIPAddr:     schedulerIPAddr,
		mgr:                 mgr,
		frnDict:             make(map[string]*FrameworkRiakNode),
		user:                user,
		zookeepers:          zookeepers,
		cepm:                c,
		frameworkName:       frameworkName,
		frameworkRole:       frameworkRole,
		NodeCpus:            nodeCpus,
		NodeMem:             nodeMem,
		NodeDisk:            nodeDisk,
		schedulerState:      ss,
		authProvider:        authProvider,
		mesosAuthPrincipal:  mesosAuthPrincipal,
		mesosAuthSecretFile: mesosAuthSecretFile,
		compatibilityMode:   !useReservations,
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
	}
	if sc.mesosAuthSecretFile != "" && sc.mesosAuthPrincipal != "" {
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

	if sc.authProvider != "" && sc.mesosAuthPrincipal != "" {
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

func FindNodeForOffer(offer *mesos.Offer, nodes []*FrameworkRiakNode) *FrameworkRiakNode {
	for _, riakNode := range nodes {
		// Fresh node
		if !riakNode.HasRequestedReservation() {
			var applySuccess bool
			applySuccess, offer = riakNode.ApplyOffer(offer)
			if !applySuccess {
				return nil
			}
			log.Infof("Found a new offer for a node. Offer: %+v, Node: %+v", *offer.Id.Value, riakNode.CurrentID())
			return riakNode
		}
	}

	return nil
}

func FindNodeForReservation(offerWithReservation *mesos.Offer, nodes []*FrameworkRiakNode) *FrameworkRiakNode {
	for _, riakNode := range nodes {
		if riakNode.HasRequestedReservation() && riakNode.OfferCompatible(offerWithReservation) {
			var applySuccess bool
			applySuccess, offerWithReservation = riakNode.ApplyOffer(offerWithReservation)
			if !applySuccess {
				return nil
			}
			log.Infof("Found an offer with a reservation for a node. Offer: %+v, Node: %+v", *offerWithReservation.Id.Value, riakNode.CurrentID())
			return riakNode
		}
	}

	return nil
}

func FindNodeForOfferCompatibilityMode(offer *mesos.Offer, nodes []*FrameworkRiakNode) *FrameworkRiakNode {
	for _, riakNode := range nodes {
		if riakNode.NeedsToBeScheduled() {
			var applySuccess bool
			applySuccess, offer = riakNode.ApplyOffer(offer)
			if !applySuccess {
				return nil
			}
			log.Infof("Found an offer for a node (Compatibility Mode). OfferID: %+v, Node: %+v", *offer.Id.Value, riakNode.CurrentID())
			return riakNode
		}
	}

	log.Warnf("Found an offer (Compatibility Mode), but no nodes were compatible with it. OfferID: %+v", *offer.Id.Value)
	return nil
}

func UnreserveOffer(offer *mesos.Offer, operations map[string][]*mesos.Offer_Operation) map[string][]*mesos.Offer_Operation {
	//Make copies of resources so it isn't stupid
	destroy := &mesos.Offer_Operation_Destroy{
		Volumes: offer.Resources,
	}
	destroyType := mesos.Offer_Operation_DESTROY
	destroyOperation := &mesos.Offer_Operation{
		Type:    &destroyType,
		Destroy: destroy,
	}
	unreserve := &mesos.Offer_Operation_Unreserve{
		Resources: offer.Resources,
	}
	unreserveType := mesos.Offer_Operation_UNRESERVE
	unreserveOperation := &mesos.Offer_Operation{
		Type:      &unreserveType,
		Unreserve: unreserve,
	}

	if operations[*offer.Id.Value] == nil {
		operations[*offer.Id.Value] = []*mesos.Offer_Operation{}
	}

	operations[*offer.Id.Value] = append(operations[*offer.Id.Value], destroyOperation)
	operations[*offer.Id.Value] = append(operations[*offer.Id.Value], unreserveOperation)

	return operations
}

func ReserveOfferForNode(riakNode *FrameworkRiakNode, operations map[string][]*mesos.Offer_Operation) map[string][]*mesos.Offer_Operation {
	reserveResources := riakNode.GetResourcesToReserve()
	createResources := riakNode.GetResourcesToCreate()

	reserve := &mesos.Offer_Operation_Reserve{
		Resources: reserveResources,
	}
	reserveType := mesos.Offer_Operation_RESERVE
	reserveOperation := &mesos.Offer_Operation{
		Type:    &reserveType,
		Reserve: reserve,
	}
	create := &mesos.Offer_Operation_Create{
		Volumes: createResources,
	}
	createType := mesos.Offer_Operation_CREATE
	createOperation := &mesos.Offer_Operation{
		Type:   &createType,
		Create: create,
	}

	if operations[*riakNode.LastOfferUsed.Id.Value] == nil {
		operations[*riakNode.LastOfferUsed.Id.Value] = []*mesos.Offer_Operation{}
	}

	operations[*riakNode.LastOfferUsed.Id.Value] = append(operations[*riakNode.LastOfferUsed.Id.Value], reserveOperation)
	operations[*riakNode.LastOfferUsed.Id.Value] = append(operations[*riakNode.LastOfferUsed.Id.Value], createOperation)

	return operations
}

func (sc *SchedulerCore) AddNodeToLaunchTasks(riakNode *FrameworkRiakNode, operations map[string][]*mesos.Offer_Operation) map[string][]*mesos.Offer_Operation {
	log.Infof("Using offerId: %+v, for riakNode.CurrentID(): %+v", *riakNode.LastOfferUsed.Id.Value, riakNode.CurrentID())

	taskInfos := []*mesos.TaskInfo{}
	taskInfo := riakNode.PrepareForLaunchAndGetNewTaskInfo(sc)
	taskInfos = append(taskInfos, taskInfo)
	sc.frnDict[riakNode.CurrentID()] = riakNode

	launch := &mesos.Offer_Operation_Launch{
		TaskInfos: taskInfos,
	}
	operationType := mesos.Offer_Operation_LAUNCH
	launchOperation := &mesos.Offer_Operation{
		Type:   &operationType,
		Launch: launch,
	}

	if operations[*riakNode.LastOfferUsed.Id.Value] == nil {
		operations[*riakNode.LastOfferUsed.Id.Value] = []*mesos.Offer_Operation{}
	}
	operations[*riakNode.LastOfferUsed.Id.Value] = append(operations[*riakNode.LastOfferUsed.Id.Value], launchOperation)

	return operations
}

func (sc *SchedulerCore) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	log.Info("Received resource offers: ", offers)
	operations := make(map[string][]*mesos.Offer_Operation)
	toBeScheduled := []*FrameworkRiakNode{}

	// Find nodes which need to be scheduled
	for _, cluster := range sc.schedulerState.Clusters {
		for _, riakNode := range cluster.Nodes {
			if riakNode.NeedsToBeScheduled() {
				log.Infof("Adding Riak node for scheduling: %+v", riakNode.CurrentID())
				toBeScheduled = append(toBeScheduled, riakNode)
			}
		}
	}

	// Populate launchTasks and nodesWithAcceptedOffers
	for _, offer := range offers {
		if sc.compatibilityMode {
			riakNode := FindNodeForOfferCompatibilityMode(offer, toBeScheduled)
			if riakNode != nil {
				operations = sc.AddNodeToLaunchTasks(riakNode, operations)
				sc.schedulerState.Persist()
			}
		} else if common.ResourcesHaveReservations(offer.Resources) {
			riakNode := FindNodeForReservation(offer, toBeScheduled)
			if riakNode == nil {
				log.Warnf("Found an offer with a reservation, but no nodes were compatible with it. Unreserving. Offer: %+v", *offer.Id.Value)
				operations = UnreserveOffer(offer, operations)
			} else {
				operations = sc.AddNodeToLaunchTasks(riakNode, operations)
				sc.schedulerState.Persist()
			}
		} else {
			riakNode := FindNodeForOffer(offer, toBeScheduled)
			if riakNode != nil {
				operations = ReserveOfferForNode(riakNode, operations)
				sc.schedulerState.Persist()
			}
		}
	}

	// Attempt to reserve resources and/or launch nodes
	for _, offer := range offers {
		offerOperations := operations[*offer.Id.Value]

		if offerOperations == nil {
			offerOperations = []*mesos.Offer_Operation{}
		}

		go func(innerOffer *mesos.Offer, innerOperations []*mesos.Offer_Operation) {
			log.Infof("Accepting offerID: %+v, operations for offer: %+v", *innerOffer.Id.Value, innerOperations)

			var innerStatus mesos.Status
			var innerErr error

			if sc.compatibilityMode {
				innerTasks := []*mesos.TaskInfo{}
				for _, innerOperation := range innerOperations {
					if *innerOperation.Type == mesos.Offer_Operation_LAUNCH {
						innerTasks = innerOperation.Launch.TaskInfos
					}
				}
				innerStatus, innerErr = driver.LaunchTasks([]*mesos.OfferID{innerOffer.Id}, innerTasks, &mesos.Filters{RefuseSeconds: proto.Float64(OFFER_INTERVAL)})
			} else {
				innerStatus, innerErr = driver.AcceptOffers([]*mesos.OfferID{innerOffer.Id}, innerOperations, &mesos.Filters{RefuseSeconds: proto.Float64(OFFER_INTERVAL)})
			}

			if innerStatus != mesos.Status_DRIVER_RUNNING {
				log.Fatal("Driver not running, while trying to accept offers")
			}
			if innerErr != nil {
				if innerErr.Error() == "404 Not Found" {
					log.Warnf("Attempted to call an endpoint that does not exist on the mesos master. Moving to compatibility mode.")
					sc.compatibilityMode = true
				} else {
					log.Panic("Failed to launch tasks: ", innerErr)
				}
			}
		}(offer, offerOperations)
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
