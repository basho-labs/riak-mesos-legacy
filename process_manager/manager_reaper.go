package process_manager

import (
	log "github.com/Sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	subscribeNotifications = make(chan subscriptionRequest, 1)
	unsubscribeNotifications = make(chan unsubscriptionRequest, 1)
	go processManagerLoop()
}

var subscribeNotifications chan subscriptionRequest
var unsubscribeNotifications chan unsubscriptionRequest

type subscriptionRequest struct {
	pid       int
	replyChan chan pidChangeNotification
}

type unsubscriptionRequest struct {
	pid int
}
type pidChangeNotification struct {
	pid     int
	wstatus syscall.WaitStatus
	rusage  syscall.Rusage
}

func subscribe(pid int) chan pidChangeNotification {
	// reply must be non-blocking
	replyChan := make(chan pidChangeNotification, 1)
	req := subscriptionRequest{pid: pid, replyChan: replyChan}
	subscribeNotifications <- req
	return replyChan
}

func unsubscribe(pid int) {
	req := unsubscriptionRequest{pid: pid}
	unsubscribeNotifications <- req
}

func processManagerLoop() {
	defer close(subscribeNotifications)
	defer close(unsubscribeNotifications)

	sigchlds := make(chan os.Signal, 1000)
	signal.Notify(sigchlds, syscall.SIGCHLD)
	defer signal.Stop(sigchlds)
	defer close(sigchlds)
	subscriptions := make(map[int]chan pidChangeNotification)

	for {
		select {
		case subscribe := <-subscribeNotifications:
			{
				_, assigned := subscriptions[subscribe.pid]
				if !assigned {
					subscriptions[subscribe.pid] = subscribe.replyChan
				} else {
					panic("Duplicate subscription for a PID")
				}
			}
		case unsubscribe := <-unsubscribeNotifications:
			{
				delete(subscriptions, unsubscribe.pid)
			}
		case <-sigchlds:
			{
				var wstatus syscall.WaitStatus
				var rusage syscall.Rusage
				waitPid, err := syscall.Wait4(-1, &wstatus, 0, &rusage)
				if err == nil {
					subscription, assigned := subscriptions[waitPid]
					if assigned {
						notification := pidChangeNotification{
							pid:     waitPid,
							wstatus: wstatus,
							rusage:  rusage,
						}
						select {
						case subscription <- notification:
							{
							}
						default:
							{
								log.Info("PID Change subscription channel full, deleting entry")
								delete(subscriptions, waitPid)
							}
						}
					} else {
						log.Info("PID notification received without actor to send it to: ", waitPid)
					}
				} else {
					log.Error("Error waiting for PID: ", err)
				}
			}
		}
	}
}
