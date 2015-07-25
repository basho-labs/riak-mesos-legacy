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
	pid		  int
	subscribe chan chan int
}

func NewProcessManager(tdcb TeardownCallback, executablePath string, args []string, healthcheck Healthchecker) (*ProcessManager, error) {
	retFuture := make(chan *ProcessManager)
	go StartProcessManager(tdcb, executablePath, args, healthcheck, retFuture)
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
func StartProcessManager(tdcb TeardownCallback, executablePath string, args []string, healthcheck Healthchecker, retChan chan *ProcessManager) {
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
	waitChan := startSignalHandler(pm.pid, sigchlds)
	subscriptions := []chan int{}


	for i := 0; i < 30; i++ {
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
		case <-time.After(100 * time.Millisecond):
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
func (pm *ProcessManager) background(waitChan chan handleSignal, subscriptions []chan int, signals chan os.Signal) {
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

func (pm *ProcessManager) killProcess(waitChan chan handleSignal) {
	log.Info("Killing process")
	// Is it alive?
	if syscall.Kill(pm.pid, syscall.Signal(0)) != nil {
		return
	}


	syscall.Kill(pm.pid, syscall.SIGTERM)
	select {
		case <- waitChan: {
			return
		}
		case <-time.After(time.Second * 5): {
			syscall.Kill(pm.pid, syscall.SIGKILL)
		}
	}
	<- waitChan
}

func (pm *ProcessManager) start(executablePath string, args []string) {

	sysprocattr := &syscall.SysProcAttr{
		Setpgid:true,
	}
	procattr := &syscall.ProcAttr{
		Sys:sysprocattr,
		Env:os.Environ(),
		Files:[]uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
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
	}
}

type handleSignal struct {
	wstatus syscall.WaitStatus
	rusage syscall.Rusage
	pid int
}
func startSignalHandler(pid int, chlds chan os.Signal) chan handleSignal {
	ch := make(chan handleSignal, 1)
	go signalHandler(pid, chlds, ch)
	return ch
}
func signalHandler(pid int, chlds chan os.Signal, waitChan chan handleSignal) {
	var wstatus syscall.WaitStatus
	var rusage syscall.Rusage

	defer signal.Stop(chlds)
	defer close(chlds)
	for _ =  range chlds {
		waitPid, err := syscall.Wait4(-1 * pid, &wstatus, 0, &rusage)
		if err == nil && waitPid == pid {
			hs := handleSignal {
				wstatus:wstatus,
				rusage:rusage,
				pid:waitPid,
			}
			waitChan<-hs
		}
	}
}
