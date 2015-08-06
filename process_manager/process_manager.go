package process_manager

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Healthchecker func() error
type TeardownCallback func()

type ProcessManager struct {
	tdcb      TeardownCallback
	teardown  chan chan interface{}
	pid       int
	subscribe chan chan int
}

func NewProcessManager(tdcb TeardownCallback, executablePath string, args []string, healthcheck Healthchecker) (*ProcessManager, error) {
	retFuture := make(chan *ProcessManager)
	go startProcessManager(tdcb, executablePath, args, healthcheck, retFuture)
	retVal := <-retFuture
	log.Info("Retval: ", retVal)
	if retVal == nil {
		err := fmt.Errorf("Unknown Error")
		return retVal, err
	} else {
		return retVal, nil
	}
}

func (pm *ProcessManager) Listen() chan int {
	ret := make(chan int, 1)
	pm.subscribe <- ret
	return ret
}
func (pm *ProcessManager) TearDown() {
	replyChan := make(chan interface{})
	pm.teardown <- replyChan
	<-replyChan
	return
}
func startProcessManager(tdcb TeardownCallback, executablePath string, args []string, healthcheck Healthchecker, retChan chan *ProcessManager) {
	defer close(retChan)
	pm := &ProcessManager{
		teardown:  make(chan chan interface{}, 10),
		tdcb:      tdcb,
		subscribe: make(chan chan int, 10),
	}
	defer close(pm.teardown)
	defer close(pm.subscribe)
	signals := make(chan os.Signal, 3)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
	defer signal.Stop(signals)
	defer close(signals)

	// Hopefully we don't get more than 1000 SIGCHLD in the before we come up
	sigchlds := make(chan os.Signal, 1000)
	signal.Notify(sigchlds, syscall.SIGCHLD)

	pm.start(executablePath, args)
	waitChan := subscribe(pm.pid)
	defer unsubscribe(pm.pid)
	defer close(waitChan)
	subscriptions := []chan int{}

	// Wait 60 seconds for the process to start, and pass its healthcheck.
	for i := 0; i < 60; i++ {
		select {
		case subscribe := <-pm.subscribe:
			{
				subscriptions = append(subscriptions, subscribe)
			}
		case <-signals:
			{
				log.Info("Tearing down at signal")
				pm.notify(-1, subscriptions)
				pm.killProcess(waitChan)
				pm.tdcb()
				return
			}
		case status := <-waitChan:
			{
				pm.notify(status.wstatus.ExitStatus(), subscriptions)
				pm.tdcb()
				pm.killProcess(waitChan)
				return
			}
		case tearDownChan := <-pm.teardown:
			{
				log.Info("Tearing down")
				pm.notify(-1, subscriptions)
				pm.killProcess(waitChan)
				pm.tdcb()
				tearDownChan <- nil
				return
			}
		case <-time.After(1000 * time.Millisecond):
			{
				// Try pinging Riak Explorer
				err := healthcheck()
				if err == nil {
					log.Info("Process status: ", err)
					retChan <- pm
					// re.background() should never return
					pm.background(waitChan, subscriptions, signals)
					return
				}
			}
		}
	}
	log.Info("Process manager failed to start process in time")
	pm.killProcess(waitChan)
}

func (pm *ProcessManager) notify(status int, subscriptions []chan int) {
	log.Info("Notify being called")
	for _, sub := range subscriptions {
		select {
		case sub <- status:
		default:
		}
	}
}
func (pm *ProcessManager) background(waitChan chan pidChangeNotification, subscriptions []chan int, signals chan os.Signal) {
	log.Info("Going into background mode")
	for {
		select {
		case status := <-waitChan:
			{
				pm.notify(status.wstatus.ExitStatus(), subscriptions)
				pm.tdcb()
				pm.killProcess(waitChan)
				return
			}
		case <-signals:
			{
				log.Info("Tearing down at signal")
				pm.notify(-1, subscriptions)
				pm.tdcb()
				pm.killProcess(waitChan)
				return
			}
		case tearDownChan := <-pm.teardown:
			{
				log.Info("Tearing down")
				pm.notify(-1, subscriptions)
				pm.killProcess(waitChan)
				pm.tdcb()
				tearDownChan <- nil
				return
			}
		case subscribe := <-pm.subscribe:
			{
				subscriptions = append(subscriptions, subscribe)
			}
		}
	}
}

func (pm *ProcessManager) killProcess(waitChan chan pidChangeNotification) {
	log.Info("Killing process")
	// Is it alive?
	if syscall.Kill(pm.pid, syscall.Signal(0)) != nil {
		return
	}

	// TODO: Work around some potential races here

	syscall.Kill(pm.pid, syscall.SIGTERM)
	select {
	case <-waitChan:
		{
			return
		}
	case <-time.After(time.Second * 5):
		{
			syscall.Kill(pm.pid, syscall.SIGKILL)
		}
	}
	<-waitChan
}

func (pm *ProcessManager) start(executablePath string, args []string) {

	sysprocattr := &syscall.SysProcAttr{
		Setpgid: true,
	}
	env := os.Environ()

	procattr := &syscall.ProcAttr{
		Sys:   sysprocattr,
		Env:   env,
		Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
	}

	if os.Getenv("HOME") == "" {
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal("Could not get current working directory")
		}
		procattr.Dir = wd
		homevar := fmt.Sprintf("HOME=%s", wd)
		procattr.Env = append(os.Environ(), homevar)
	}
	var err error
	realArgs := append([]string{executablePath}, args...)
	pm.pid, err = syscall.ForkExec(executablePath, realArgs, procattr)
	if err != nil {
		log.Panic("Error starting process")
	} else {
		log.Infof("Process Manager started to manage %v at PID: %v", executablePath, pm.pid)
	}

}
