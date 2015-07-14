package scheduler

import (
	"encoding/json"
	"sync"

	log "github.com/Sirupsen/logrus"
	metamgr "github.com/basho-labs/riak-mesos/metadata_manager"
	"github.com/golang/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	sched "github.com/mesos/mesos-go/scheduler"

	//"github.com/basho-labs/riak-mesos/common"
	"time"
)

const (
	OFFER_INTERVAL float64 = 5
)

func newReconciliationServer(driver sched.SchedulerDriver) *ReconcilationServer {
	rs := &ReconcilationServer{
		tasksToReconcile: make(chan *mesos.TaskStatus, 10),
		lock:             &sync.Mutex{},
		enabled:          false,
		driver:           driver,
	}
	go rs.loop()
	return rs
}

type ReconcilationServer struct {
	tasksToReconcile chan *mesos.TaskStatus
	driver           sched.SchedulerDriver
	lock             *sync.Mutex
	enabled          bool
}

func (rServer *ReconcilationServer) enable() {
	log.Info("Reconcilation process enabled")
	rServer.lock.Lock()
	defer rServer.lock.Unlock()
	rServer.enabled = true
}

func (rServer *ReconcilationServer) disable() {
	log.Info("Reconcilation process disabled")
	rServer.lock.Lock()
	defer rServer.lock.Unlock()
	rServer.enabled = true
}
func (rServer *ReconcilationServer) loop() {
	tasksToReconcile := []*mesos.TaskStatus{}
	ticker := time.Tick(time.Millisecond * 100)
	for {
		select {
		case task := <-rServer.tasksToReconcile:
			{
				tasksToReconcile = append(tasksToReconcile, task)
			}
		case <-ticker:
			{
				rServer.lock.Lock()
				if rServer.enabled {
					rServer.lock.Unlock()
					if len(tasksToReconcile) > 0 {
						log.Info("Reconciling tasks: ", tasksToReconcile)
						rServer.driver.ReconcileTasks(tasksToReconcile)
						tasksToReconcile = []*mesos.TaskStatus{}
					}
				} else {
					rServer.lock.Unlock()
				}
			}
		}
	}
}

type SchedulerCore struct {
	lock                *sync.Mutex
	frameworkName       string
	clusters            map[string]*FrameworkRiakCluster
	schedulerHTTPServer *SchedulerHTTPServer
	mgr                 *metamgr.MetadataManager
	schedulerIpAddr     string
	frnDict             map[string]*FrameworkRiakNode
	rServer             *ReconcilationServer
	user                string
	zookeepers			[]string
}

func NewSchedulerCore(schedulerHostname string, frameworkName string, zookeepers []string, schedulerIpAddr string, user string) *SchedulerCore {
	mgr := metamgr.NewMetadataManager(frameworkName, zookeepers)
	scheduler := &SchedulerCore{
		lock:            &sync.Mutex{},
		frameworkName:   frameworkName,
		schedulerIpAddr: schedulerIpAddr,
		clusters:        make(map[string]*FrameworkRiakCluster),
		mgr:             mgr,
		frnDict:         make(map[string]*FrameworkRiakNode),
		user:            user,
		zookeepers:      zookeepers,
	}
	scheduler.schedulerHTTPServer = ServeExecutorArtifact(scheduler, schedulerHostname)
	return scheduler
}

// This is an add cluster callback from the metadata manager
func (sc *SchedulerCore) AddCluster(zkNode *metamgr.ZkNode) metamgr.MetadataManagerCluster {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	frc := NewFrameworkRiakCluster()
	frc.sc = sc
	frc.zkNode = zkNode
	err := json.Unmarshal(zkNode.GetData(), &frc)
	if err != nil {
		log.Panic("Error getting node: ", err)
	}
	sc.clusters[frc.Name] = frc
	return frc
}
func (sc *SchedulerCore) GetCluster(name string) metamgr.MetadataManagerCluster {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	return sc.clusters[name]
}

// Should basically just be a callback - DO NOT change state
func (sc SchedulerCore) NewCluster(zkNode *metamgr.ZkNode, name string) metamgr.MetadataManagerCluster {
	frc := &FrameworkRiakCluster{
		zkNode: zkNode,
		nodes:  make(map[string]*FrameworkRiakNode),
		Name:   name,
	}
	return frc
}

func (sc *SchedulerCore) setupMetadataManager() {
	sc.mgr.SetupFramework(sc.schedulerHTTPServer.URI, sc)
}
func (sc *SchedulerCore) Run(mesosMaster string) {
	frameworkId := &mesos.FrameworkID{
		Value: proto.String(sc.frameworkName),
	}
	// TODO: Get "Real" credentials here
	var frameworkUser *string
	if sc.user != "" {
		frameworkUser = proto.String(sc.user)
	}
	cred := (*mesos.Credential)(nil)
	bindingAddress := parseIP(sc.schedulerIpAddr)
	fwinfo := &mesos.FrameworkInfo{
		User:            frameworkUser,
		Name:            proto.String("Riak Framework"),
		Id:              frameworkId,
		FailoverTimeout: proto.Float64(86400),
	}

	log.Info("Running scheduler with FrameworkInfo: ", fwinfo)

	config := sched.DriverConfig{
		Scheduler:      sc,
		Framework:      fwinfo,
		Master:         mesosMaster,
		Credential:     cred,
		BindingAddress: bindingAddress,
		//	WithAuthContext: func(ctx context.Context) context.Context {
		//		ctx = auth.WithLoginProvider(ctx, *authProvider)
		//		ctx = sasl.WithBindingAddress(ctx, bindingAddress)
		//		return ctx
		//	},
	}
	driver, err := sched.NewMesosSchedulerDriver(config)
	if err != nil {
		log.Error("Unable to create a SchedulerDriver ", err.Error())
	}
	sc.rServer = newReconciliationServer(driver)

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
	launchTasks := []*mesos.TaskInfo{}
	toBeScheduled := []*FrameworkRiakNode{}
	for _, cluster := range sc.clusters {
		for _, riakNode := range cluster.nodes {
			if riakNode.NeedsToBeScheduled() {
				log.Infof("Adding Riak node for scheduling: %+v", riakNode)
				// We need to schedule this task I guess?
				toBeScheduled = append(toBeScheduled, riakNode)
			}
		}
	}

	// Issue https://github.com/basho-labs/riak-mesos/issues/11
	// TODO: This currently fills in each Mesos node as much as possible
	// Simply switching the outer and inner loops would result in spreading
	// tasks across as many nodes as possible
	// We either need to make this pluggalbe, or something? I don't know.

	for _, offer := range offers {
		resources := offer.Resources
		for _, riakNode := range toBeScheduled {
			var success bool
			var ask []*mesos.Resource
			resources, ask, success = riakNode.GetCombinedAsk()(resources)
			if success {
				taskInfo := riakNode.PrepareForLaunchAndGetNewTaskInfo(offer, ask)
				sc.frnDict[riakNode.CurrentID()] = riakNode
				launchTasks = append(launchTasks, taskInfo)
				riakNode.Persist()
			} else {
				log.Error("Not enough resources to schedule RiakNode")
			}

		}
	}

	offerIDs := make([]*mesos.OfferID, len(offers))
	for idx, offer := range offers {
		offerIDs[idx] = offer.Id
	}
	log.Info("Launching Tasks: ", launchTasks)
	status, err := driver.LaunchTasks(offerIDs, launchTasks, &mesos.Filters{RefuseSeconds: proto.Float64(OFFER_INTERVAL)})
	if status != mesos.Status_DRIVER_RUNNING {
		log.Fatal("Driver not running, while trying to launch tasks")
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
		riak_node.handleStatusUpdate(status)
		riak_node.Persist()
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
