package scheduler

import (
	"io/ioutil"
	"sync"

	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
	auth "github.com/basho-labs/mesos-go/auth"
	sasl "github.com/basho-labs/mesos-go/auth/sasl"
	mesos "github.com/basho-labs/mesos-go/mesosproto"
	sched "github.com/basho-labs/mesos-go/scheduler"
	"github.com/basho-labs/riak-mesos/cepmd/cepm"
	metamgr "github.com/basho-labs/riak-mesos/metadata_manager"
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
	nodeCpus            string
	nodeMem             string
	nodeDisk            string
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
		nodeCpus:            nodeCpus,
		nodeMem:             nodeMem,
		nodeDisk:            nodeDisk,
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

func (sc *SchedulerCore) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	log.Info("Received resource offers: ", offers)

	operations := sc.createOperationsForOffers(offers)

	// Attempt to reserve resources and/or launch nodes
	for _, offer := range offers {
		offerOperations := operations[*offer.Id.Value]

		if offerOperations == nil {
			offerOperations = []*mesos.Offer_Operation{}
		}

		go sc.acceptOffer(driver, offer, offerOperations)
	}
}

func (sc *SchedulerCore) acceptOffer(driver sched.SchedulerDriver, offer *mesos.Offer, operations []*mesos.Offer_Operation) {
	log.Infof("Accepting OfferID: %+v, Operations: %+v", *offer.Id.Value, operations)

	var status mesos.Status
	var err error

	if sc.compatibilityMode {
		tasks := []*mesos.TaskInfo{}
		for _, operation := range operations {
			if *operation.Type == mesos.Offer_Operation_LAUNCH {
				tasks = operation.Launch.TaskInfos
			}
		}
		status, err = driver.LaunchTasks([]*mesos.OfferID{offer.Id}, tasks, &mesos.Filters{RefuseSeconds: proto.Float64(OFFER_INTERVAL)})
	} else {
		status, err = driver.AcceptOffers([]*mesos.OfferID{offer.Id}, operations, &mesos.Filters{RefuseSeconds: proto.Float64(OFFER_INTERVAL)})
	}

	if status != mesos.Status_DRIVER_RUNNING {
		log.Fatal("Driver not running, while trying to accept offers")
	}
	if err != nil {
		log.Panic("Failed to launch tasks: ", err)
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
