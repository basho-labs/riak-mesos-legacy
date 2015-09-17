package main

import (
	"flag"
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/basho-labs/riak-mesos/metadata_manager"
	"github.com/basho-labs/riak-mesos/scheduler"
	"github.com/kr/pretty"
	"github.com/samuel/go-zookeeper/zk"
	"time"
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
	flag.StringVar(&cmd, "command", "get-url", "get-url, get-clusters, get-cluster, create-cluster, delete-cluster, get-nodes, add-node, add-nodes, get-state")
	flag.Parse()

	if cmd == "" {
		fmt.Println("Please specify command")
		os.Exit(1)
	}
	log.SetLevel(log.DebugLevel)
}

func main() {
	switch cmd {
	case "get-url":
		fmt.Println(getURL())
	case "get-state":
		getState()
	case "delete-framework":
		respond(deleteFramework(), nil)
	case "zk-list-children":
		respondList(zkListChildren(), nil)
	case "zk-get-data":
		respond(zkGetData(), nil)
	case "zk-delete":
		respond("ok", zkDelete())
	case "get-clusters":
		client = NewSchedulerHTTPClient(getURL())
		respond(client.GetClusters())
	case "get-cluster":
		client = NewSchedulerHTTPClient(getURL())
		requireClusterName()
		respond(client.GetCluster(clusterName))
	case "create-cluster":
		client = NewSchedulerHTTPClient(getURL())
		requireClusterName()
		respond(client.CreateCluster(clusterName))
	case "delete-cluster":
		client = NewSchedulerHTTPClient(getURL())
		requireClusterName()
		respond(client.DeleteCluster(clusterName))
	case "get-nodes":
		client = NewSchedulerHTTPClient(getURL())
		requireClusterName()
		respond(client.GetNodes(clusterName))
	case "add-node":
		client = NewSchedulerHTTPClient(getURL())
		requireClusterName()
		respond(client.AddNode(clusterName))
	case "add-nodes":
		client = NewSchedulerHTTPClient(getURL())
		requireClusterName()
		for i := 1; i <= nodes; i++ {
			respond(client.AddNode(clusterName))
		}
	default:
		log.Fatal("Unknown command")
	}
}

func respondList(val []string, err error) {
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(val)
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
	frameworkName = "/riak/frameworks/" + frameworkName
	zkDelete()
	return "ok"
}

func zkListChildren() []string {
	conn, _, err := zk.Connect([]string{zookeeperAddr}, time.Second)
	if err != nil {
		log.Panic(err)
	}
	children, _, err := conn.Children(frameworkName)

	if err != nil {
		log.Panic(err)
	}
	return children
}

func zkGetData() string {
	conn, _, err := zk.Connect([]string{zookeeperAddr}, time.Second)
	if err != nil {
		log.Panic(err)
	}
	data, _, err := conn.Get(frameworkName)

	if err != nil {
		log.Panic(err)
	}
	return string(data)
}

func zkDelete() error {
	conn, _, err := zk.Connect([]string{zookeeperAddr}, time.Second)
	if err != nil {
		log.Panic(err)
	}

	zkDeleteChildren(conn, frameworkName)

	return nil
}

func requireClusterName() {
	if clusterName == "" {
		fmt.Println("Please specify value for cluster name")
		os.Exit(1)
	}
}

func zkDeleteChildren(conn *zk.Conn, path string) {
	children, _, _ := conn.Children(path)

	// Leaf
	if len(children) == 0 {
		fmt.Println("Deleting ", path)
		err := conn.Delete(path, -1)
		if err != nil {
			log.Panic(err)
		}
		return
	}

	// Branches
	for _, name := range children {
		zkDeleteChildren(conn, path+"/"+name)
	}

	return
}

func getState() {
	mm := metadata_manager.NewMetadataManager(frameworkName, []string{zookeeperAddr})
	zkNode, err := mm.GetRootNode().GetChild("SchedulerState")
	if err != zk.ErrNoNode {
		ss, err := scheduler.DeserializeSchedulerState(zkNode.GetData())
		if err != nil {
			log.Panic(err)
		}
		fmt.Printf("%# v", pretty.Formatter(ss))

	}
}
