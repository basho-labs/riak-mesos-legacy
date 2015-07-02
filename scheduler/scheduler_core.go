package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/golang/protobuf/proto"
	"github.com/basho/bletchley/common"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"github.com/basho/bletchley/metadata_manager"
	sched "github.com/mesos/mesos-go/scheduler"
	"sync"
	"time"
	"flag"
)

var (
	mesosMaster	string
)

const (
	OFFER_INTERVAL float64 = 5
)

func init() {
	flag.StringVar(&mesosMaster, "master", "33.33.33.2", "mesos master")
	flag.Parse()
}

type SchedulerCore struct {
	subscribtionLock        *sync.Mutex
	driver                  sched.SchedulerDriver
	registered              chan registeredCast
	reregistered            chan reregisteredCast
	resourceOffers          chan resourceOffers
	resourceOffersRescinded chan resourceOffersRescinded
	statusUpdate            chan statusUpdateCast
	internalTaskStates      map[string]*mesos.TaskStatus
	targetTasksSubs         map[string]*TargetTask
	schedulerHTTPServer     SchedulerHTTPServer
	driverConfig            *sched.DriverConfig
	outstandingTasks        chan scheduleTask
	taskStateSubscribe      chan taskStateSubscribe
	mgr                     *metadata_manager.MetadataManager
}

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
	TaskInfo   		*mesos.TaskInfo
	TargetTask 		*TargetTask
	Filters			[]common.ResourceAsker
	replyChannel    chan bool
}

func newSchedulerCore(frameworkName string, schedulerHTTPServer *SchedulerHTTPServer, mgr *metadata_manager.MetadataManager) *SchedulerCore {
	scheduler := &SchedulerCore{
		subscribtionLock:        &sync.Mutex{},
		driver:                  nil,
		registered:              make(chan registeredCast, 1),
		reregistered:            make(chan reregisteredCast, 1),
		resourceOffers:          make(chan resourceOffers, 1),
		resourceOffersRescinded: make(chan resourceOffersRescinded, 1),
		statusUpdate:            make(chan statusUpdateCast, 100),
		taskStateSubscribe:      make(chan taskStateSubscribe, 1),
		internalTaskStates:      make(map[string]*mesos.TaskStatus),
		targetTasksSubs:         make(map[string]*TargetTask),
		schedulerHTTPServer:     *schedulerHTTPServer,
		driverConfig:            nil,
		outstandingTasks:        make(chan scheduleTask, 10),
		mgr:                     mgr,
	}
	frameworkId := &mesos.FrameworkID{
		Value: proto.String(frameworkName),
	}
	// TODO: Get "Real" credentials here
	cred := (*mesos.Credential)(nil)
	// TODO: Take flag for
	bindingAddress := parseIP("33.33.33.1")
	fwinfo := &mesos.FrameworkInfo{
		User:            proto.String("sargun"), // Mesos-go will fill in user.
		Name:            proto.String("Test Framework (Go)"),
		Id:              frameworkId,
		FailoverTimeout: proto.Float64(86400),
	}
	config := sched.DriverConfig{
		Scheduler:      scheduler,
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
	scheduler.driver = driver
	scheduler.driverConfig = &config
	return scheduler
}

func (sched *SchedulerCore) Subscribe(taskID string, targetTask *TargetTask) {
	sched.taskStateSubscribe <- taskStateSubscribe{targetTask: targetTask, taskID: taskID, subscriptionChangeType: subscribe}
}
func (sched *SchedulerCore) Unsubscribe(taskID string, targetTask *TargetTask) {
	sched.taskStateSubscribe <- taskStateSubscribe{targetTask: targetTask, taskID: taskID, subscriptionChangeType: unsubscribe}
}
func (sched *SchedulerCore) ScheduleTask(TaskInfo *mesos.TaskInfo, TargetTask *TargetTask, askers []common.ResourceAsker) bool {
	log.Infof("Scheduler called!")
	sc := scheduleTask{TaskInfo: TaskInfo, TargetTask: TargetTask, replyChannel:make(chan bool), Filters: askers}
	sched.outstandingTasks <- sc
	return <-sc.replyChannel
}

func (sched *SchedulerCore) handleStatusUpdate(msg statusUpdateCast) {
	targetTask, assigned := sched.targetTasksSubs[msg.status.TaskId.GetValue()]
	if assigned {
		targetTask.UpdateStatus(msg.status)
	}
	// We should probably garbage collect the internal task state dictionary
	// But, for now just collect them all -- memory is cheap!
	sched.internalTaskStates[msg.status.TaskId.GetValue()] = msg.status
}
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
						if !success { break }
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
						sched.subscribtionLock.Lock()
						sched.targetTasksSubs[task.TaskInfo.TaskId.GetValue()] = task.TargetTask
						sched.subscribtionLock.Unlock()
						task.replyChannel <- true
						tasks = append(tasks, task.TaskInfo)
					}
				}
				sched.driver.LaunchTasks(offerIDs, tasks, &mesos.Filters{RefuseSeconds: proto.Float64(OFFER_INTERVAL)})
				log.Info("No outstanding tasks")
				return
			}
		}
	}
}
func (sched *SchedulerCore) SchedulingLoop() {
	for {
		select {
		case offers := <-sched.resourceOffers: sched.handleResourceOffers(offers.offers)
		// This number is chosen
		case <- time.After(time.Duration(3 * OFFER_INTERVAL) * time.Second):
			{
				log.Info("No resource offers received")
				select {
				case request := <-sched.outstandingTasks: {
					log.Info("received outstanding tasks during no offer period: ", request)
					request.replyChannel <- false
				}
				default: log.Info("Received no outstanding tasks during no offer period")
				}
			}

		}
	}
}
func (sched *SchedulerCore) handleSubChange(subChange taskStateSubscribe) {
	log.Info("Changing subscription: ", subChange)
	sched.subscribtionLock.Lock()
	defer sched.subscribtionLock.Unlock()
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
func (sched *SchedulerCore) MesosLoop() {
	select {
	case msg := <-sched.registered:
		log.Info("Scheduler routine registered: ", msg.frameworkId, msg.masterInfo)
	case msg := <-sched.reregistered:
		log.Info("Scheduler routine reregistered: ", msg.masterInfo)
	}
	go sched.SchedulingLoop()
	for {
		select {
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
func (sched *SchedulerCore) Run() {
	go sched.MesosLoop()
	if stat, err := sched.driver.Run(); err != nil {
		log.Infof("Framework stopped with status %s and error: %s\n", stat.String(), err.Error())
	}
}
func (sched *SchedulerCore) Registered(driver sched.SchedulerDriver, frameworkId *mesos.FrameworkID, masterInfo *mesos.MasterInfo) {
	log.Info("Framework registered")
	sched.registered <- registeredCast{frameworkId, masterInfo}

}

func (sched *SchedulerCore) Reregistered(driver sched.SchedulerDriver, masterInfo *mesos.MasterInfo) {
	//go NewTargetTask(*sched).Loop()
	log.Info("Framework reregistered")
	sched.reregistered <- reregisteredCast{masterInfo}

}
func (sched *SchedulerCore) Disconnected(sched.SchedulerDriver) {}

func (sched *SchedulerCore) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	sched.resourceOffers <- resourceOffers{offers}
}
func (sched *SchedulerCore) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	sched.statusUpdate <- statusUpdateCast{status}
}

func (sched *SchedulerCore) OfferRescinded(driver sched.SchedulerDriver, offerID *mesos.OfferID) {
	sched.resourceOffersRescinded <- resourceOffersRescinded{offerID}
}

func (sched *SchedulerCore) FrameworkMessage(driver sched.SchedulerDriver, executorID *mesos.ExecutorID, slaveID *mesos.SlaveID, message string) {
	log.Info("Got unknown framework message %v")
}
// TODO: Write handler
func (sched *SchedulerCore) SlaveLost(sched.SchedulerDriver, *mesos.SlaveID) {
	log.Info("Slave Lost")
}
// TODO: Write handler
func (sched *SchedulerCore) ExecutorLost(sched.SchedulerDriver, *mesos.ExecutorID, *mesos.SlaveID, int) {
	log.Info("Executor Lost")
}

func (sched *SchedulerCore) Error(driver sched.SchedulerDriver, err string) {
	log.Info("Scheduler received error:", err)
}
