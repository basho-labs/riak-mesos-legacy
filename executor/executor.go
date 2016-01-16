package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	exec "github.com/mesos/mesos-go/executor"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
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

	exec.riakNode.finish()
}

func (exec *ExecutorCore) FrameworkMessage(driver exec.ExecutorDriver, msg string) {
	exec.lock.Lock()
	defer exec.lock.Unlock()
	fmt.Println("Got framework message: ", msg)
	switch msg {
	case "finish":
		{
			log.Info("Force finishing riak node")
			exec.riakNode.ForceFinish()
		}
	}
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
	runtime.GOMAXPROCS(1)
	log.SetLevel(log.DebugLevel)
	fmt.Println("Starting Riak Executor")
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGUSR1, syscall.SIGUSR2)

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
		}
	}
}
