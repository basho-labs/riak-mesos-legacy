package scheduler

import (
	log "github.com/Sirupsen/logrus"
	"github.com/basho-labs/riak-mesos/cepmd/cepm"
	"github.com/basho-labs/riak-mesos/common"
	metamgr "github.com/basho-labs/riak-mesos/metadata_manager"
	"github.com/golang/protobuf/proto"
	auth "github.com/mesos/mesos-go/auth"
	sasl "github.com/mesos/mesos-go/auth/sasl"
	mesos "github.com/mesos/mesos-go/mesosproto"
	sched "github.com/mesos/mesos-go/scheduler"
	"golang.org/x/net/context"
	"io/ioutil"
	"os"
	"sync"
)

const (
	OFFER_INTERVAL float64 = 5
)

type SchedulerCore struct {
	lock                *sync.Mutex
	schedulerHTTPServer *SchedulerHTTPServer
	mgr                 *metamgr.MetadataManager
	schedulerIPAddr     string
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

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	config := sched.DriverConfig{
		Scheduler:        sc,
		Framework:        fwinfo,
		Master:           mesosMaster,
		Credential:       cred,
		HostnameOverride: hostname,
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

func (sc *SchedulerCore) createOperationsForOffers(offers []*mesos.Offer) map[string][]*mesos.Offer_Operation {
	operations := make(map[string][]*mesos.Offer_Operation)

	// Populate operations
	for _, offer := range offers {
		needsReconciliation := false
		offerHelper := common.NewOfferHelper(offer)
		log.Infof("Got offer with these resources: %s", offerHelper.String())

		for _, cluster := range sc.schedulerState.Clusters {
			if cluster.ApplyOffer(offerHelper, sc) {
				needsReconciliation = true
			}
		}

		if !needsReconciliation {
			offerHelper.MaybeUnreserve()
		}

		operations[*offer.Id.Value] = offerHelper.Operations()
	}

	return operations
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

	log.Info("Received status updates: ", status)
	foundNode := false

	for _, cluster := range sc.schedulerState.Clusters {
		if cluster.HasNode(status.TaskId.GetValue()) {
			foundNode = true
			cluster.HandleNodeStatusUpdate(status)
			break
		}
	}

	if foundNode {
		sc.schedulerState.Persist()
	}

	if !foundNode {
		for _, cluster := range sc.schedulerState.Graveyard {
			if cluster.HasNode(status.TaskId.GetValue()) {
				foundNode = true
				log.Warn("Received status update for node in killed cluster: ", status)
				break
			}
		}
	}

	if !foundNode {
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
