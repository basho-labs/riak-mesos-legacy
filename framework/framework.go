package main

import (
	"flag"
	"os"
	"strconv"

	"regexp"

	log "github.com/Sirupsen/logrus"
	"github.com/basho-labs/riak-mesos/scheduler"
)

// Must start with a-z, or A-Z
// Can contain any of the following, a-z, A-Z, 0-9,  -, _
var frameNameRegex *regexp.Regexp = regexp.MustCompile("[a-zA-Z][a-zA-Z0-9-_]*")

var (
	mesosMaster       string
	zookeeperAddr     string
	schedulerHostname string
	schedulerIPAddr   string
	user              string
	logFile           string
	frameworkName     string
	frameworkRole     string
)

func init() {
	flag.StringVar(&mesosMaster, "master", "zk://33.33.33.2:2181/mesos", "Mesos master")
	flag.StringVar(&zookeeperAddr, "zk", "33.33.33.2:2181", "Zookeeper")
	flag.StringVar(&schedulerHostname, "hostname", "", "Framework hostname")
	flag.StringVar(&schedulerIPAddr, "ip", "", "Framework ip")
	flag.StringVar(&user, "user", "", "Framework Username")
	flag.StringVar(&logFile, "log", "", "Log File Location")
	flag.StringVar(&frameworkName, "name", "riakMesosFramework", "Framework Instance Name")
	flag.StringVar(&frameworkRole, "role", "*", "Framework Role Name")

	flag.Parse()
}

func main() {
	log.SetLevel(log.DebugLevel)

	if logFile != "" {
		fo, logErr := os.Create(logFile)
		if logErr != nil {
			panic(logErr)
		}
		log.SetOutput(fo)
	}

	if frameNameRegex.FindString(frameworkName) != frameworkName {
		log.Fatal("Error, framework name not valid")
	}
	// When starting scheduler from Marathon, PORT0-N env vars will be set
	rexPortStr := os.Getenv("PORT1")

	// If PORT1 isn't set, fallback to a hardcoded one for now
	// TODO: Sargun fix me
	if rexPortStr == "" {
		rexPortStr = "9090"
	}

	rexPort, portErr := strconv.Atoi(rexPortStr)
	if portErr != nil {
		log.Fatal(portErr)
	}

	sched := scheduler.NewSchedulerCore(schedulerHostname, frameworkName, frameworkRole, []string{zookeeperAddr}, schedulerIPAddr, user, rexPort)
	sched.Run(mesosMaster)
}
