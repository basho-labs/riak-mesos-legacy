package main

//go:generate go-bindata -o bindata_generated.go data/

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	log "github.com/Sirupsen/logrus"
	exec "github.com/mesos/mesos-go/executor"
	mesos "github.com/mesos/mesos-go/mesosproto"
)

const (
	kill int = iota
)

type ExecutorCore struct {
	lock      *sync.Mutex
	riakNode  *RiakNode
	Driver    exec.ExecutorDriver
	execInfo  *mesos.ExecutorInfo
	slaveInfo *mesos.SlaveInfo
	fwInfo    *mesos.FrameworkInfo
}

func newExecutor() *ExecutorCore {
	return &ExecutorCore{
		lock: &sync.Mutex{},
	}
}

func (exec *ExecutorCore) Registered(driver exec.ExecutorDriver, execInfo *mesos.ExecutorInfo, fwinfo *mesos.FrameworkInfo, slaveInfo *mesos.SlaveInfo) {
	exec.lock.Lock()
	defer exec.lock.Unlock()
	fmt.Println("Registered Executor on slave ", slaveInfo.GetHostname())
	log.Info("Executor Info: ", execInfo)
	log.Info("Slave Info: ", slaveInfo)
	log.Info("Framework Info: ", fwinfo)
	exec.slaveInfo = slaveInfo
	exec.execInfo = execInfo
	exec.fwInfo = fwinfo
}

func (exec *ExecutorCore) Reregistered(driver exec.ExecutorDriver, slaveInfo *mesos.SlaveInfo) {
	exec.lock.Lock()
	defer exec.lock.Unlock()
	exec.Driver = driver
	fmt.Println("Re-registered Executor on slave ", slaveInfo.GetHostname())
	exec.slaveInfo = slaveInfo
}

func (exec *ExecutorCore) Disconnected(exec.ExecutorDriver) {
	exec.lock.Lock()
	defer exec.lock.Unlock()
	fmt.Println("Executor disconnected.")
}

func (exec *ExecutorCore) LaunchTask(driver exec.ExecutorDriver, taskInfo *mesos.TaskInfo) {
	exec.lock.Lock()
	defer exec.lock.Unlock()
	fmt.Println("Launching task", taskInfo.GetName(), "with command", taskInfo.Command.GetValue())
	os.Args[0] = fmt.Sprintf("executor - %s", taskInfo.TaskId.GetValue())

	//fmt.Println("Other hilarious facts: ", taskInfo)

	//
	// this is where one would perform the requested task
	//
	fmt.Println("Starting task")

	runStatus := &mesos.TaskStatus{
		TaskId: taskInfo.TaskId,
		State:  mesos.TaskState_TASK_STARTING.Enum(),
	}
	_, err := driver.SendStatusUpdate(runStatus)

	if err != nil {
		log.Panic("Got error", err)
	}

	if exec.riakNode != nil {
		log.Fatalf("Task being started, twice, existing task: %+v, new task: %+v", exec.riakNode)
	}
	exec.riakNode = NewRiakNode(taskInfo, exec)
	exec.riakNode.Run()

}

func (exec *ExecutorCore) KillTask(driver exec.ExecutorDriver, taskId *mesos.TaskID) {
	exec.lock.Lock()
	defer exec.lock.Unlock()
	fmt.Println("Kill task")
	runStatus := &mesos.TaskStatus{
		TaskId: exec.riakNode.taskInfo.GetTaskId(),
		State:  mesos.TaskState_TASK_KILLED.Enum(),
	}
	_, err := driver.SendStatusUpdate(runStatus)

	if err != nil {
		log.Panic("Got error", err)
	}
}

func (exec *ExecutorCore) FrameworkMessage(driver exec.ExecutorDriver, msg string) {
	exec.lock.Lock()
	defer exec.lock.Unlock()
	fmt.Println("Got framework message: ", msg)
}

func (exec *ExecutorCore) Shutdown(driver exec.ExecutorDriver) {
	fmt.Println("Shutting down the executor")
	driver.Stop()
	os.Exit(0)
}

func (exec *ExecutorCore) Error(driver exec.ExecutorDriver, err string) {
	exec.lock.Lock()
	defer exec.lock.Unlock()
	fmt.Println("Got error message:", err)
}

func main() {
	log.SetLevel(log.DebugLevel)
	fmt.Println("Starting Riak Executor")
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGCHLD)

	data, _ := Asset("data/stuff")
	s := string(data)
	fmt.Printf("data=%v\n", s)

	executor := newExecutor()
	dconfig := exec.DriverConfig{
		Executor: executor,
	}
	driver, err := exec.NewMesosExecutorDriver(dconfig)

	if err != nil {
		fmt.Println("Unable to create a ExecutorDriver ", err.Error())
	}

	_, err = driver.Start()
	if err != nil {
		fmt.Println("Got error:", err)
		return
	}
	go signalWatcher(signals, executor)
	executor.Driver = driver
	fmt.Println("Executor process has started and running.")
	driver.Join()
}

func signalWatcher(signals chan os.Signal, exec *ExecutorCore) {
	for signal := range signals {
		switch signal {
		case syscall.SIGUSR1:
			{
				log.Info("Marking task as finished")
				exec.riakNode.finish()
			}
		case syscall.SIGUSR2:
			{
				log.Info("Marking task as finished")
				exec.riakNode.next()
			}
		case syscall.SIGCHLD:
			{
				log.Info("Got SIGCHLD")
				for {
					var status syscall.WaitStatus
					var rusage syscall.Rusage
					pid, _ := syscall.Wait4(-1, &status, syscall.WNOHANG, &rusage)
					if pid <= 0 { break }
					log.Infof("Handled SIGCHLD for PID: %d, Waitstatus: %+v, Rusage: %+v", pid, status, rusage)

				}
			}
		}
	}
}
