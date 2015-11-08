package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"

	log "github.com/Sirupsen/logrus"
	"github.com/basho-labs/riak-mesos/scheduler"
	"github.com/mesos/mesos-go/auth/sasl"
	"github.com/mesos/mesos-go/auth/sasl/mech"
	"runtime"
)

// Must start with a-z, or A-Z
// Can contain any of the following, a-z, A-Z, 0-9,  -, _
var frameNameRegex *regexp.Regexp = regexp.MustCompile("[a-zA-Z][a-zA-Z0-9-_]*")

var (
	mesosMaster         string
	zookeeperAddr       string
	schedulerHostname   string
	schedulerIPAddr     string
	user                string
	logFile             string
	frameworkName       string
	frameworkRole       string
	nodeCpus            string
	nodeMem             string
	nodeDisk            string
	authProvider        string
	mesosAuthPrincipal  string
	mesosAuthSecretFile string
	useReservations     bool
)

func init() {
	runtime.GOMAXPROCS(1)
	flag.BoolVar(&useReservations, "use_reservations", false, "Set this to true if the Mesos cluster supports Dynamic Reservations and Persistent Volumes")
	flag.StringVar(&mesosMaster, "master", "zk://33.33.33.2:2181/mesos", "Mesos master")
	flag.StringVar(&zookeeperAddr, "zk", "33.33.33.2:2181", "Zookeeper")
	flag.StringVar(&schedulerHostname, "hostname", "", "Framework hostname")
	flag.StringVar(&schedulerIPAddr, "ip", "", "Framework ip")
	flag.StringVar(&user, "user", "", "Framework Username")
	flag.StringVar(&logFile, "log", "", "Log File Location")
	flag.StringVar(&frameworkName, "name", "riak", "Framework Instance Name")
	flag.StringVar(&frameworkRole, "role", "*", "Framework Role Name")
	flag.StringVar(&nodeCpus, "node_cpus", "1.0", "Per Node CPUs")
	flag.StringVar(&nodeMem, "node_mem", "16000", "Per Node Mem")
	flag.StringVar(&nodeDisk, "node_disk", "20000", "Per Node Disk")
	flag.StringVar(&authProvider, "mesos_authentication_provider", sasl.ProviderName,
		fmt.Sprintf("Authentication provider to use, default is SASL that supports mechanisms: %+v", mech.ListSupported()))
	flag.StringVar(&mesosAuthPrincipal, "mesos_authentication_principal", "", "Mesos authentication principal.")
	flag.StringVar(&mesosAuthSecretFile, "mesos_authentication_secret_file", "", "Mesos authentication secret file.")

	flag.Parse()
}

func main() {
	runtime.GOMAXPROCS(1)
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

	sched := scheduler.NewSchedulerCore(
		schedulerHostname,
		frameworkName,
		frameworkRole,
		[]string{zookeeperAddr},
		schedulerIPAddr,
		user,
		nodeCpus,
		nodeMem,
		nodeDisk,
		authProvider,
		mesosAuthPrincipal,
		mesosAuthSecretFile,
		useReservations)
	sched.Run(mesosMaster)
}
