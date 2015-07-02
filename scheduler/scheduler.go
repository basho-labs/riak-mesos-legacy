package main
import (
	log "github.com/Sirupsen/logrus"
	"github.com/basho/bletchley/metadata_manager"
	"flag"
)

var (
	mesosMaster string
	zookeeperAddr string
	schedulerHostname string
	schedulerIpAddr string
)

func init() {
	flag.StringVar(&mesosMaster, "master", "33.33.33.2:5050", "Mesos master address")
	flag.StringVar(&zookeeperAddr, "zk", "33.33.33.2:2181", "Zookeeper address")
	flag.StringVar(&schedulerHostname, "hostname", "", "Framework hostname")
	flag.StringVar(&schedulerIpAddr, "ip", "33.33.33.1", "Framework ip address")
	flag.Parse()
}

func main() {
	log.SetLevel(log.DebugLevel)
	SchedulerHTTPServer := serveExecutorArtifact(schedulerHostname)
	scheduler_name := "riak-mesos-go3"
	mgr := metadata_manager.NewMetadataManager(scheduler_name, zookeeperAddr)
	sched := newSchedulerCore(scheduler_name, SchedulerHTTPServer, mgr, mesosMaster, schedulerIpAddr)
	go NewTargetTask("golang-riak-task-a", sched, mgr).Loop()
	go NewTargetTask("golang-riak-task-b", sched, mgr).Loop()
	go NewTargetTask("golang-riak-task-c", sched, mgr).Loop()


	sched.Run()


}
