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
	schedulerIPAddr   string
	frameworkName     string
	user              string
)

func init() {
	flag.StringVar(&mesosMaster, "master", "zk://33.33.33.2:2181/mesos", "Mesos master")
	flag.StringVar(&zookeeperAddr, "zk", "33.33.33.2:2181", "Zookeeper")
	flag.StringVar(&schedulerHostname, "hostname", "", "Framework hostname")
	flag.StringVar(&schedulerIPAddr, "ip", "33.33.33.1", "Framework ip")
	flag.StringVar(&frameworkName, "name", "riak-mesos-go3", "Framework Instance Name")
	flag.StringVar(&user, "user", "", "Framework Username")
	flag.Parse()
}

func main() {
	log.SetLevel(log.DebugLevel)

	sched := scheduler.NewSchedulerCore(schedulerHostname, frameworkName, []string{zookeeperAddr}, schedulerIPAddr, user)
	sched.Run(mesosMaster)
}
