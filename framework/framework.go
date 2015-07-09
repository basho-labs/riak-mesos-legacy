package main

//go:generate go-bindata -ignore=Makefile -o bindata_generated.go -tags rel data/

import (
	"flag"

	log "github.com/Sirupsen/logrus"
	"github.com/basho-labs/riak-mesos/metadata_manager"
	"github.com/basho-labs/riak-mesos/scheduler"
)

var (
	mesosMaster       string
	zookeeperAddr     string
	schedulerHostname string
	schedulerIpAddr   string
	FrameworkName     string
)

func init() {
	flag.StringVar(&mesosMaster, "master", "zk://33.33.33.2:2181/mesos", "Mesos master")
	flag.StringVar(&zookeeperAddr, "zk", "33.33.33.2:2181", "Zookeeper")
	flag.StringVar(&schedulerHostname, "hostname", "", "Framework hostname")
	flag.StringVar(&schedulerIpAddr, "ip", "33.33.33.1", "Framework ip")
	flag.StringVar(&FrameworkName, "name", "riak-mesos-go3", "Framework Instance Name")
	flag.Parse()
}

func main() {
	log.SetLevel(log.DebugLevel)
	mgr := metadata_manager.NewMetadataManager(FrameworkName, zookeeperAddr)
	sched := framework.NewSchedulerCore(schedulerHostname, FrameworkName, mgr, schedulerIpAddr)
	//go framework.NewTargetTask("golang-riak-task-a", sched, mgr).Loop()
	//	go framework.NewTargetTask("golang-riak-task-b", sched, mgr).Loop()
	//	go framework.NewTargetTask("golang-riak-task-c", sched, mgr).Loop()
	sched.Run(mesosMaster)

}
