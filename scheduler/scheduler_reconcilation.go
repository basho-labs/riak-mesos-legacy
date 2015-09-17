package scheduler

import (
	log "github.com/Sirupsen/logrus"
	mesos "github.com/mesos/mesos-go/mesosproto"
	sched "github.com/mesos/mesos-go/scheduler"
	"sync"
	"time"
)

func newReconciliationServer(driver sched.SchedulerDriver, sc *SchedulerCore) *ReconcilationServer {
	rs := &ReconcilationServer{
		nodesToReconcile: make(chan *FrameworkRiakNode, 10),
		lock:             &sync.Mutex{},
		enabled:          false,
		driver:           driver,
		wakeup:           make(chan bool, 1),
		sc:               sc,
	}
	go rs.loop()
	return rs
}

type ReconcilationServer struct {
	nodesToReconcile chan *FrameworkRiakNode
	driver           sched.SchedulerDriver
	lock             *sync.Mutex
	enabled          bool
	wakeup           chan bool
	sc               *SchedulerCore
}

func (rServer *ReconcilationServer) enable() {
	log.Info("Reconcilation process enabled")
	rServer.lock.Lock()
	defer rServer.lock.Unlock()
	rServer.enabled = true
	select {
	case rServer.wakeup <- true:
	default:
	}
}

func (rServer *ReconcilationServer) disable() {
	log.Info("Reconcilation process disabled")
	rServer.lock.Lock()
	defer rServer.lock.Unlock()
	rServer.enabled = true
}
func (rServer *ReconcilationServer) reconcile() {
	rServer.lock.Lock()
	defer rServer.lock.Unlock()
	if rServer.enabled {
		tasksToReconcile := []*mesos.TaskStatus{}
		for _, cluster := range rServer.sc.schedulerState.Clusters {
			for _, node := range cluster.Nodes {
				if !node.reconciled {
					if _, assigned := rServer.sc.frnDict[node.GetTaskStatus().TaskId.GetValue()]; !assigned {
						rServer.sc.frnDict[node.GetTaskStatus().TaskId.GetValue()] = node
					}
					tasksToReconcile = append(tasksToReconcile, node.GetTaskStatus())
					rServer.driver.ReconcileTasks(tasksToReconcile)
				}
			}
		}
	}
}
func (rServer *ReconcilationServer) loop() {
	// TODO: Add exponential backoff
	ticker := time.Tick(time.Second * 5)
	for range ticker {
		rServer.reconcile()
	}
}
