package main
import (
	log "github.com/Sirupsen/logrus"
	"github.com/basho/bletchley/metadata_manager"
)

func main() {
	log.SetLevel(log.DebugLevel)
	SchedulerHTTPServer := serveExecutorArtifact()
	scheduler_name := "riak-mesos-go3"
	mgr := metadata_manager.NewMetadataManager(scheduler_name)
	sched := newSchedulerCore(scheduler_name, SchedulerHTTPServer, mgr)
	go NewTargetTask("golang-riak-task-a", sched, mgr).Loop()
	go NewTargetTask("golang-riak-task-b", sched, mgr).Loop()
	go NewTargetTask("golang-riak-task-c", sched, mgr).Loop()


	sched.Run()


}