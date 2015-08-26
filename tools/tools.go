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
	clusterName   string
	frameworkName string
	cmd           string
	client        *SchedulerHTTPClient
)

func init() {
	flag.StringVar(&zookeeperAddr, "zk", "33.33.33.2:2181", "Zookeeper")
	flag.IntVar(&nodes, "nodes", 1, "Nodes in new cluster")
	flag.StringVar(&clusterName, "cluster-name", "", "Name of new cluster")
	flag.StringVar(&frameworkName, "name", "riakMesosFramework", "Framework Instance ID")
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
	case "delete-framework":
		respond(deleteFramework(), nil)
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

func deleteFramework() string {
	mgr := metadata_manager.NewMetadataManager(frameworkName, []string{zookeeperAddr})
	mgr.GetRootNode().Delete()

	return "ok"
}

func requireClusterName() {
	if clusterName == "" {
		fmt.Println("Please specify value for cluster name")
		os.Exit(1)
	}
}
