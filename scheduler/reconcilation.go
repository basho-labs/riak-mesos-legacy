package scheduler

import (
	log "github.com/Sirupsen/logrus"
	sched "github.com/mesos/mesos-go/scheduler"
	"time"
	"sync/atomic"
)

func newReconciliationServer(driver sched.SchedulerDriver, sc *SchedulerCore) *ReconcilationServer {
	rs := &ReconcilationServer{
		nodesToReconcile: make(chan *FrameworkRiakNode, 10),
		enabled:          atomic.Value{},
		driver:           driver,
		sc:               sc,
	}
	rs.enabled.Store(false)
	go rs.loop()
	return rs
}

type ReconcilationServer struct {
	nodesToReconcile chan *FrameworkRiakNode
	driver           sched.SchedulerDriver
	enabled          atomic.Value
	sc               *SchedulerCore
}

func (rServer *ReconcilationServer) enable() {
	log.Info("Reconcilation process enabled")
	rServer.enabled.Store(true)
}

func (rServer *ReconcilationServer) disable() {
	log.Info("Reconcilation process disabled")
	rServer.enabled.Store(false)
}
func (rServer *ReconcilationServer) reconcile() {
	//rServer.sc.lock.Lock()
	//defer rServer.sc.lock.Unlock()
	if rServer.enabled.Load().(bool) == true {
		rServer.driver.ReconcileTasks(rServer.sc.GetTasksToReconcile())
	}
}
func (rServer *ReconcilationServer) loop() {
	// TODO: Add exponential backoff
	ticker := time.Tick(time.Second * 5)
	for range ticker {
		rServer.reconcile()
	}
}
