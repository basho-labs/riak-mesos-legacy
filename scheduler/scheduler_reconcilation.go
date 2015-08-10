package scheduler

import (
	"sync"
	"time"
	log "github.com/Sirupsen/logrus"
	mesos "github.com/mesos/mesos-go/mesosproto"
	sched "github.com/mesos/mesos-go/scheduler"
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
