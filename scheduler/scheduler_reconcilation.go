package scheduler

import (
	log "github.com/Sirupsen/logrus"
	mesos "github.com/mesos/mesos-go/mesosproto"
	sched "github.com/mesos/mesos-go/scheduler"
	"sync"
	"time"
)

type ReconcilationServerState int

const (
	waiting ReconcilationServerState = iota
	reconcilining
)

func newReconciliationServer(driver sched.SchedulerDriver) *ReconcilationServer {
	rs := &ReconcilationServer{
		nodesToReconcile: make(chan *FrameworkRiakNode, 10),
		lock:             &sync.Mutex{},
		enabled:          false,
		driver:           driver,
		status:           waiting,
	}
	go rs.loop()
	return rs
}

type ReconcilationServer struct {
	nodesToReconcile chan *FrameworkRiakNode
	driver           sched.SchedulerDriver
	lock             *sync.Mutex
	enabled          bool
	status           ReconcilationServerState
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
func (rServer *ReconcilationServer) reconcile(nodesToReconcile map[string]*FrameworkRiakNode) {
	rServer.lock.Lock()
	defer rServer.lock.Unlock()
	if rServer.enabled {
		tasksToReconcile := []*mesos.TaskStatus{}
		for key, node := range nodesToReconcile {
			if node.reconciled {
				delete(nodesToReconcile, key)
			} else {
				tasksToReconcile = append(tasksToReconcile, node.GetTaskStatus())
				rServer.driver.ReconcileTasks(tasksToReconcile)
			}
		}
	}
}
func (rServer *ReconcilationServer) loop() {
	nodesToReconcile := make(map[string]*FrameworkRiakNode)

	// TODO: Add exponential backoff
	ticker := time.Tick(time.Second * 5)
	for {
		select {
		case node := <-rServer.nodesToReconcile:
			{
				nodesToReconcile[node.UUID.String()] = node
				rServer.reconcile(nodesToReconcile)
			}
		case <-ticker:
			{
				rServer.reconcile(nodesToReconcile)
			}
		}
	}
}
