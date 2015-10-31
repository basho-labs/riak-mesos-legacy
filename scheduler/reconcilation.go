package scheduler

import (
	log "github.com/Sirupsen/logrus"
	mesos "github.com/basho-labs/mesos-go/mesosproto"
	sched "github.com/basho-labs/mesos-go/scheduler"
	"sync/atomic"
	"time"
)

func newReconciliationServer(driver sched.SchedulerDriver, sc *SchedulerCore) *ReconcilationServer {
	rs := &ReconcilationServer{
		enabled: atomic.Value{},
		driver:  driver,
		sc:      sc,
	}
	rs.enabled.Store(false)
	go rs.loop()
	return rs
}

type ReconcilationServer struct {
	driver  sched.SchedulerDriver
	enabled atomic.Value
	sc      *SchedulerCore
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
	// Get Tasks to Reconcile
	if rServer.enabled.Load().(bool) == true {
		rServer.sc.lock.Lock()
		defer rServer.sc.lock.Unlock()
		rServer.reconcileTasks()
		rServer.killTasks()
	}
}
func (rServer *ReconcilationServer) loop() {
	rServer.reconcile()
	ticker := time.Tick(time.Second * 5)
	for range ticker {
		rServer.reconcile()
	}
}

func (rServer *ReconcilationServer) killTasks() {
	// Get Tasks to Kill
	for _, cluster := range rServer.sc.schedulerState.Clusters {
		nodesToKill, nodesToRemove := cluster.GetNodesToKillOrRemove()
		for _, riakNode := range nodesToKill {
			log.Infof("Killing node: %+v", riakNode.CurrentID())
			status, err := rServer.driver.KillTask(riakNode.GetTaskStatus().TaskId)
			if status != mesos.Status_DRIVER_RUNNING {
				log.Fatal("Driver not running, while trying to kill tasks")
			}
			if err != nil {
				log.Warnf("Failed to kill tasks: ", err)
			}
		}

		if len(nodesToRemove) > 0 {
			for _, riakNode := range nodesToRemove {
				cluster.RemoveNode(riakNode)
			}
			rServer.sc.schedulerState.Persist()
		}
	}
}

func (rServer *ReconcilationServer) reconcileTasks() {
	for _, cluster := range rServer.sc.schedulerState.Clusters {
		tasksToReconcile := cluster.GetNodeTasksToReconcile()
		if len(tasksToReconcile) != 0 {
			log.Debug("Reconciling tasks: ", tasksToReconcile)
			rServer.driver.ReconcileTasks(tasksToReconcile)
		}
	}
}
