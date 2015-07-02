package main

//go:generate go-bindata -o bindata_generated.go data/

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	exec "github.com/mesos/mesos-go/executor"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"os"
)

const (
	kill int = iota
)

type ExecutorCore struct {
	riakNode *RiakNode
	Driver   exec.ExecutorDriver
}

func newExecutor() *ExecutorCore {
	return &ExecutorCore{}
}

func (exec *ExecutorCore) Registered(driver exec.ExecutorDriver, execInfo *mesos.ExecutorInfo, fwinfo *mesos.FrameworkInfo, slaveInfo *mesos.SlaveInfo) {
	exec.Driver = driver
	fmt.Println("Registered Executor on slave ", slaveInfo.GetHostname())
}

func (exec *ExecutorCore) Reregistered(driver exec.ExecutorDriver, slaveInfo *mesos.SlaveInfo) {
	exec.Driver = driver
	fmt.Println("Re-registered Executor on slave ", slaveInfo.GetHostname())
}

func (exec *ExecutorCore) Disconnected(exec.ExecutorDriver) {
	fmt.Println("Executor disconnected.")
}

func (exec *ExecutorCore) LaunchTask(driver exec.ExecutorDriver, taskInfo *mesos.TaskInfo) {
	fmt.Println("Launching task", taskInfo.GetName(), "with command", taskInfo.Command.GetValue())
	os.Args[0] = fmt.Sprintf("executor - %s", taskInfo.TaskId.GetValue())

	//fmt.Println("Other hilarious facts: ", taskInfo)

	//
	// this is where one would perform the requested task
	//
	fmt.Println("Starting task")

	exec.riakNode = NewRiakNode(taskInfo, exec)
	go exec.riakNode.Loop()
	/*	select {
				case <-time.After(time.Second * 120):
					{
						fmt.Println("Finishing task", taskInfo.GetName())
						finStatus := &mesos.TaskStatus{
							TaskId: taskInfo.GetTaskId(),
							State:  mesos.TaskState_TASK_FINISHED.Enum(),
						}
						_, err = driver.SendStatusUpdate(finStatus)
						if err != nil {
							fmt.Println("Got error", err)
						}
						delete(exec.tasks, *taskInfo.TaskId.Value)
						fmt.Println("Task finished", taskInfo.GetName())
					}
				case <-ch:
					{
						fmt.Println("Killing task", taskInfo.GetName())
						finStatus := &mesos.TaskStatus{
							TaskId: taskInfo.GetTaskId(),
							State:  mesos.TaskState_TASK_KILLED.Enum(),
						}
						_, err = driver.SendStatusUpdate(finStatus)
						if err != nil {
							fmt.Println("Got error", err)
						}
						delete(exec.tasks, *taskInfo.TaskId.Value)
						fmt.Println("Killed task", taskInfo.GetName())
					}
			}
			time.Sleep(time.Second * 10)
			driver.Stop()
			time.Sleep(time.Second * 10)
			os.Exit(0)
		}
		go lol()
		fmt.Println("Scheduler continuing")        */
}

func (exec *ExecutorCore) KillTask(driver exec.ExecutorDriver, taskId *mesos.TaskID) {
	fmt.Println("Kill task")
}

func (exec *ExecutorCore) FrameworkMessage(driver exec.ExecutorDriver, msg string) {
	fmt.Println("Got framework message: ", msg)
}

func (exec *ExecutorCore) Shutdown(driver exec.ExecutorDriver) {
	fmt.Println("Shutting down the executor")
	driver.Stop()
	os.Exit(0)
}

func (exec *ExecutorCore) Error(driver exec.ExecutorDriver, err string) {
	fmt.Println("Got error message:", err)
}

func main() {

	log.SetLevel(log.DebugLevel)
	fmt.Println("Starting Example Executor (Go)")
	fmt.Println("Args: ", os.Args)
	data, _ := Asset("data/stuff")
	s := string(data)
	fmt.Printf("data=%v\n", s)

	dconfig := exec.DriverConfig{
		Executor: newExecutor(),
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
	fmt.Println("Executor process has started and running.")
	driver.Join()
}
