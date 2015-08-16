package main

import (
	"flag"
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/basho-labs/riak-mesos/metadata_manager"
)

var (
	zookeeperAddr string
	nodes         int
	frameworkName string
	cmd           string
	client        *SchedulerHTTPClient
)

func init() {
	flag.StringVar(&zookeeperAddr, "zk", "33.33.33.2:2181", "Zookeeper")
	flag.IntVar(&nodes, "nodes", 1, "Nodes in new cluster")
	flag.StringVar(&frameworkName, "name", "riakMesosFramework", "Framework Instance ID")
	flag.StringVar(&cmd, "command", "get-url", "get-url, get-nodes, add-node, add-nodes")
	flag.Parse()

	if cmd == "" {
		fmt.Println("Please specify command")
		os.Exit(1)
	}
	log.SetLevel(log.DebugLevel)
}

func main() {

	if cmd == "get-url" {
		fmt.Println(getURL())
		os.Exit(1)
	}

	client = NewSchedulerHTTPClient(getURL())

	switch cmd {
	case "get-nodes":
		respond(client.GetNodes())
	case "add-node":
		respond(client.AddNode())
	case "add-nodes":
		for i := 1; i <= nodes; i++ {
			respond(client.AddNode())
		}
	default:
		log.Fatal("Unknown command")
	}
}

func respond(val string, err error) {
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(val)
}

func getURL() string {
	overrideURL := os.Getenv("RM_API")

	if overrideURL != "" {
		return overrideURL
	}

	mgr := metadata_manager.NewMetadataManager(frameworkName, []string{zookeeperAddr})
	zkNode, err := mgr.GetRootNode().GetChild("uri")
	if err != nil {
		log.Panic(err)
	}
	return string(zkNode.GetData())
}