package main
import (
	log "github.com/Sirupsen/logrus"
	"github.com/basho/bletchley/metadata_manager"
	"github.com/basho/bletchley/framework"
	"flag"
)

var (
	mesosMaster string
	zookeeperAddr string
	schedulerHostname string
	schedulerIpAddr string
)

func init() {
	flag.StringVar(&mesosMaster, "master", "zk://33.33.33.2:2181/mesos", "Mesos master")
	flag.StringVar(&zookeeperAddr, "zk", "33.33.33.2:2181", "Zookeeper")
	flag.StringVar(&schedulerHostname, "hostname", "", "Framework hostname")
	flag.StringVar(&schedulerIpAddr, "ip", "33.33.33.1", "Framework ip")
	flag.Parse()
}

func main() {
	log.SetLevel(log.DebugLevel)
	SchedulerHTTPServer := framework.ServeExecutorArtifact(schedulerHostname)
	scheduler_name := "riak-mesos-go3"
	mgr := metadata_manager.NewMetadataManager(scheduler_name, zookeeperAddr)
	sched := framework.NewSchedulerCore(scheduler_name, SchedulerHTTPServer, mgr, mesosMaster, schedulerIpAddr)
	go framework.NewTargetTask("golang-riak-task-a", sched, mgr).Loop()
//	go framework.NewTargetTask("golang-riak-task-b", sched, mgr).Loop()
//	go framework.NewTargetTask("golang-riak-task-c", sched, mgr).Loop()
	sched.Run()


}
