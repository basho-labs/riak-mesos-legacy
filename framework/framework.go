package framework

import (
	log "github.com/Sirupsen/logrus"
	metamgr "github.com/basho/bletchley/metadata_manager"
	"github.com/golang/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	sched "github.com/mesos/mesos-go/scheduler"
	"sync"
	"encoding/json"
	"github.com/satori/go.uuid"
)

const (
	OFFER_INTERVAL float64 = 5
)

type SchedulerCore struct {
	// New:
	lock         	*sync.Mutex
	frameworkName	string
	clusters		map[string]*FrameworkRiakCluster
	schedulerHTTPServer *SchedulerHTTPServer
	// Old:
	//driver              sched.SchedulerDriver
	internalTaskStates  map[string]*mesos.TaskStatus

	//targetTasksSubs     map[string]*TargetTask
	//driverConfig        *sched.DriverConfig
	mgr                 *metamgr.MetadataManager
	schedulerIpAddr     string

}

func NewSchedulerCore(schedulerHostname string, frameworkName string,  mgr *metamgr.MetadataManager, schedulerIpAddr string) *SchedulerCore {
	scheduler := &SchedulerCore{
		lock: 			&sync.Mutex{},
		frameworkName:  frameworkName,
		schedulerIpAddr:     schedulerIpAddr,
		clusters:			make(map[string]*FrameworkRiakCluster),

		// Old:
		internalTaskStates:  make(map[string]*mesos.TaskStatus),
		//driverConfig:        nil,
		mgr:                 mgr,
	}
	scheduler.schedulerHTTPServer = ServeExecutorArtifact(scheduler, schedulerHostname)
	//scheduler.driver = driver
	//scheduler.driverConfig = &config
	return scheduler
}

/*

func (sched *SchedulerCore) handleStatusUpdate(msg statusUpdateCast) {
	targetTask, assigned := sched.targetTasksSubs[msg.status.TaskId.GetValue()]
	if assigned {
		targetTask.UpdateStatus(msg.status)
	}
	// We should probably garbage collect the internal task state dictionary
	// But, for now just collect them all -- memory is cheap!
	sched.internalTaskStates[msg.status.TaskId.GetValue()] = msg.status
}
*/

/*
func (sched *SchedulerCore) handleResourceOffers(mesosOffers []*mesos.Offer) {
log.Debugf("Received resource offers: %v", mesosOffers)
launchPlan := make(map[string][]scheduleTask)
outstandingOffers := make(map[string]*mesos.Offer)
for _, offer := range mesosOffers {
	outstandingOffers[offer.Id.GetValue()] = offer
	launchPlan[offer.Id.GetValue()] = []scheduleTask{}
}

for {
	select {
	case request := <-sched.resourceOffersRescinded:
		{
			delete(outstandingOffers, request.offerId.GetValue())
			for _, scheduledTask := range launchPlan[request.offerId.GetValue()] {
				scheduledTask.replyChannel <- false
			}
			delete(launchPlan, request.offerId.GetValue())
		}
	case request := <-sched.outstandingTasks:
		{
			// This actually works, surprisingly enough
			// In order to add multi-task constraints, we need to know what tasks are related to one another
			// and then bucket them appropriately

			// Right now, it fills up individual hosts
			// This is "good enough" (IMHO) for the  demo
			log.Infof("Got asked to schedule outstanding task: %v\n", request)
			for key, offer := range outstandingOffers {
				tmpResources := offer.Resources
				var resourceAsk *mesos.Resource
				var success bool
				asks := []*mesos.Resource{}
				for _, filter := range request.Filters {
					tmpResources, resourceAsk, success = filter(tmpResources)
					if !success {
						break
					}
					asks = append(asks, resourceAsk)
				}
				if success {
					// The new reduced version of the resources
					outstandingOffers[key].Resources = tmpResources
					request.TaskInfo.SlaveId = outstandingOffers[key].SlaveId
					request.TaskInfo.Resources = asks
					launchPlan[key] = append(launchPlan[key], request)
				} else {
					request.replyChannel <- false
				}

			}

		}
	default:
		{
			offerIDs := []*mesos.OfferID{}
			tasks := []*mesos.TaskInfo{}
			for offerID, launchPlanTasks := range launchPlan {
				offerIDs = append(offerIDs, outstandingOffers[offerID].Id)
				for _, task := range launchPlanTasks {
					sched.subscriptionLock.Lock()
					sched.targetTasksSubs[task.TaskInfo.TaskId.GetValue()] = task.TargetTask
					sched.subscriptionLock.Unlock()
					task.replyChannel <- true
					tasks = append(tasks, task.TaskInfo)
				}
			}
			log.Infof("Launching %v task(s) using offerID(s): %v\n", len(tasks), offerIDs)
			sched.driver.LaunchTasks(offerIDs, tasks, &mesos.Filters{RefuseSeconds: proto.Float64(OFFER_INTERVAL)})
			log.Info("No outstanding tasks")
			return
		}
	}
}
}
*/
/*
func (sched *SchedulerCore) SchedulingLoop() {
for {
	select {
	case offers := <-sched.resourceOffers:
		sched.handleResourceOffers(offers.offers)
	// This number is chosen
	case <-time.After(time.Duration(3*OFFER_INTERVAL) * time.Second):
		{
			log.Info("No resource offers received")
			select {
			case request := <-sched.outstandingTasks:
				{
					log.Info("received outstanding tasks during no offer period: ", request)
					request.replyChannel <- false
				}
			default:
				log.Info("Received no outstanding tasks during no offer period")
			}
		}

	}
}
*/
/*
func (sched *SchedulerCore) handleSubChange(subChange taskStateSubscribe) {
	log.Info("Changing subscription: ", subChange)
	sched.subscriptionLock.Lock()
	defer sched.subscriptionLock.Unlock()
	switch subChange.subscriptionChangeType {
	case subscribe:
		{
			// This should trigger a reconcilation
			_, assigned := sched.targetTasksSubs[subChange.taskID]
			if assigned {
				panic("Only one task be assigned to a task ID at a time")
			} else {
				sched.targetTasksSubs[subChange.taskID] = subChange.targetTask
				sched.TriggerReconcilation(subChange.taskID)
			}
		}
	case unsubscribe:
		{
			delete(sched.targetTasksSubs, subChange.taskID)
		}
	}
}
*/
/*
func (sched *SchedulerCore) MesosLoop() {
	initialRegistration := <-sched.registered
	log.Info("Scheduler routine registered: ", initialRegistration.frameworkId, initialRegistration.masterInfo)

	go sched.SchedulingLoop()
	for {
		select {
		case msg := <-sched.reregistered:
			{
				log.Info("Scheduler routine reregistered: ", msg.masterInfo)
			}
		case msg := <-sched.statusUpdate:
			{
				sched.handleStatusUpdate(msg)
			}
		case subChange := <-sched.taskStateSubscribe:
			{
				sched.handleSubChange(subChange)

			}
		}
	}
}
*/

/*
func (sched *SchedulerCore) TriggerReconcilation(taskID string) {
	ts := mesos.TaskState_TASK_ERROR
	task := &mesos.TaskStatus{
		TaskId:  &mesos.TaskID{Value: proto.String(taskID)},
		State:   &ts,
		SlaveId: &mesos.SlaveID{Value: proto.String("")}, // Slave ID isn't required
	}
	taskStatuses := []*mesos.TaskStatus{task}
	sched.driver.ReconcileTasks(taskStatuses)
}
*/



type FrameworkRiakNode struct {
	frc				*FrameworkRiakCluster `json:"-"`
	zkNode			*metamgr.ZkNode `json:"-"`
	UUID			uuid.UUID
}


func (frn *FrameworkRiakNode) Persist() {
	data, err := json.Marshal(frn)
	if err != nil {
		log.Panic("error:", err)
	}
	frn.zkNode.SetData(data)
}

type FrameworkRiakCluster struct {
	sc 				*SchedulerCore
	zkNode			*metamgr.ZkNode `json:"-"`
	nodes			map[string]*FrameworkRiakNode `json:"-"`
	// Do not use direct access to properties!
	Name			string
}

func (frc *FrameworkRiakCluster) GetNodes() map[string]*FrameworkRiakNode {
	return frc.nodes
}

func (frc *FrameworkRiakCluster) Persist() {
	data, err := json.Marshal(frc)
	if err != nil {
		log.Panic("error:", err)
	}
	frc.zkNode.SetData(data)
}

func (frc *FrameworkRiakCluster) GetZkNode() *metamgr.ZkNode {
	return frc.zkNode
}

func (frn *FrameworkRiakNode) GetZkNode() *metamgr.ZkNode {
	return frn.zkNode
}

func (frc *FrameworkRiakCluster) NewNode() metamgr.MetadataManagerNode {
	nodes := frc.zkNode.GetChild("nodes")
	myUUID := uuid.NewV4()
	zkNode := nodes.MakeEmptyChild(myUUID.String())
	frn := NewFrameworkRiakNode()
	frn.frc = frc
	frn.zkNode = zkNode
	frn.UUID = myUUID

	return frn
}
// This is an add cluster callback from the metadata manager
func (frc *FrameworkRiakCluster) AddNode(zkNode *metamgr.ZkNode) metamgr.MetadataManagerNode {
	frc.sc.lock.Lock()
	defer frc.sc.lock.Unlock()
	log.Debug("Adding node: ", zkNode)
	frn := NewFrameworkRiakNode()
	frn.frc = frc
	frn.zkNode = zkNode
	err := json.Unmarshal(zkNode.GetData(), &frn)
	if err != nil {
		log.Panic("Error getting node: ", err)
	}
	frc.nodes[frn.UUID.String()] = frn
	return frn
}
func NewFrameworkRiakCluster() *FrameworkRiakCluster {
	return &FrameworkRiakCluster{
		nodes:  make(map[string]*FrameworkRiakNode),
	}
}

func NewFrameworkRiakNode() *FrameworkRiakNode {
	 return &FrameworkRiakNode{

	 }
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
	cred := (*mesos.Credential)(nil)
	bindingAddress := parseIP(sc.schedulerIpAddr)
	fwinfo := &mesos.FrameworkInfo{
		User:            proto.String("sargun"), // Mesos-go will fill in user.
		Name:            proto.String("Test Framework (Go)"),
		Id:              frameworkId,
		FailoverTimeout: proto.Float64(86400),
	}
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
	sc.setupMetadataManager()
	if stat, err := driver.Run(); err != nil {
		log.Infof("Framework stopped with status %s and error: %s\n", stat.String(), err.Error())
	}
}
func (sc *SchedulerCore) Registered(driver sched.SchedulerDriver, frameworkId *mesos.FrameworkID, masterInfo *mesos.MasterInfo) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	log.Info("Framework registered")

}

func (sc *SchedulerCore) Reregistered(driver sched.SchedulerDriver, masterInfo *mesos.MasterInfo) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	//go NewTargetTask(*sched).Loop()
	// We don't actually handle this correctly
	log.Error("Framework reregistered")

}
func (sc *SchedulerCore) Disconnected(sched.SchedulerDriver) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	log.Error("Framework disconnected")
}

func (sc *SchedulerCore) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	log.Info("Received resource offers")
}
func (sc *SchedulerCore) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	log.Info("Received status updates")
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


// Old:

/*

// Private "internal" structs
type registeredCast struct {
	frameworkId *mesos.FrameworkID
	masterInfo  *mesos.MasterInfo
}
type reregisteredCast struct {
	masterInfo *mesos.MasterInfo
}
type statusUpdateCast struct {
	status *mesos.TaskStatus
}

type SubscriptionChangeType int

const (
	subscribe   SubscriptionChangeType = iota
	unsubscribe                        = iota
)

type taskStateSubscribe struct {
	targetTask             *TargetTask
	taskID                 string
	subscriptionChangeType SubscriptionChangeType
}

type resourceOffers struct {
	offers []*mesos.Offer
}

type resourceOffersRescinded struct {
	offerId *mesos.OfferID
}

type scheduleTask struct {
	TaskInfo     *mesos.TaskInfo
	TargetTask   *TargetTask
	Filters      []common.ResourceAsker
	replyChannel chan bool
}
*/