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
	clusterName   string
	nodes         int
	frameworkName string
	cmd           string
	client        *SchedulerHTTPClient
)

func init() {
	flag.StringVar(&zookeeperAddr, "zk", "33.33.33.2:2181", "Zookeeper")
	flag.StringVar(&clusterName, "cluster-name", "", "Name of new cluster")
	flag.IntVar(&nodes, "nodes", 1, "Nodes in new cluster")
	flag.StringVar(&frameworkName, "name", "riak-mesos-go3", "Framework Instance Name")
	flag.StringVar(&cmd, "command", "get-url", "get-url, get-clusters, get-cluster, create-cluster, delete-cluster, get-nodes, add-node, add-nodes")
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
	case "get-clusters":
		respond(client.GetClusters())
	case "get-cluster":
		requireClusterName()
		respond(client.GetCluster(clusterName))
	case "create-cluster":
		requireClusterName()
		respond(client.CreateCluster(clusterName))
	case "delete-cluster":
		requireClusterName()
		respond(client.DeleteCluster(clusterName))
	case "get-nodes":
		requireClusterName()
		respond(client.GetNodes(clusterName))
	case "add-node":
		requireClusterName()
		respond(client.AddNode(clusterName))
	case "add-nodes":
		requireClusterName()
		for i := 1; i <= nodes; i++ {
			respond(client.AddNode(clusterName))
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
	mgr := metadata_manager.NewMetadataManager(frameworkName, []string{zookeeperAddr})
	zkNode, err := mgr.GetRootNode().GetChild("uri")
	if err != nil {
		log.Panic(err)
	}
	return string(zkNode.GetData())
}

func requireClusterName() {
	if clusterName == "" {
		fmt.Println("Please specify value for cluster name")
		os.Exit(1)
	}
}
