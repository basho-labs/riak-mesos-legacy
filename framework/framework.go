package main

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	"github.com/basho-labs/riak-mesos/scheduler"
)

var (
	mesosMaster       string
	zookeeperAddr     string
	schedulerHostname string
	schedulerIpAddr   string
	FrameworkName     string
	user              string
)

func init() {
	flag.StringVar(&mesosMaster, "master", "zk://33.33.33.2:2181/mesos", "Mesos master")
	flag.StringVar(&zookeeperAddr, "zk", "33.33.33.2:2181", "Zookeeper")
	flag.StringVar(&schedulerHostname, "hostname", "", "Framework hostname")
	flag.StringVar(&schedulerIpAddr, "ip", "33.33.33.1", "Framework ip")
	flag.StringVar(&FrameworkName, "name", "riak-mesos-go3", "Framework Instance Name")
	flag.StringVar(&user, "user", "", "Framework Username")
	flag.Parse()
}

func main() {
	log.SetLevel(log.DebugLevel)

	sched := scheduler.NewSchedulerCore(schedulerHostname, FrameworkName, []string{zookeeperAddr}, schedulerIpAddr, user)
	//go framework.NewTargetTask("golang-riak-task-a", sched, mgr).Loop()
	//	go framework.NewTargetTask("golang-riak-task-b", sched, mgr).Loop()
	//	go framework.NewTargetTask("golang-riak-task-c", sched, mgr).Loop()
	sched.Run(mesosMaster)

}
