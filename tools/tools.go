package main

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	"github.com/basho/bletchley/metadata_manager"
	//"github.com/basho/bletchley/framework"
	"fmt"
	"os"
)

var (
	zookeeperAddr string
	clusterName   string
	nodes         int
	FrameworkName string
	Cmd           string
)

func init() {
	flag.StringVar(&zookeeperAddr, "zk", "33.33.33.2:2181", "Zookeeper")
	flag.StringVar(&clusterName, "cluster-name", "", "Name of new cluster")
	flag.IntVar(&nodes, "nodes", 1, "Nodes in new cluster")
	flag.StringVar(&FrameworkName, "name", "riak-mesos-go3", "Framework Instance Name")
	flag.StringVar(&Cmd, "command", "", "Command")
	flag.Parse()

	if Cmd == "" {
		fmt.Println("Please specify command")
		os.Exit(1)
	}
	log.SetLevel(log.DebugLevel)
}

func main() {

	switch Cmd {
	case "get-url":
		fmt.Println(get_url())
	}
}
func get_url() string {
	mgr := metadata_manager.NewMetadataManager(FrameworkName, zookeeperAddr)
	return string(mgr.GetRootNode().GetChild("uri").GetData())
}

/*
if clusterName == "" {
		fmt.Println("Please specify value for cluster name")
		os.Exit(1)
	}
*/
/*		case "add-cluster": add_cluster()
		case "add-node": add_node()
		case "dump-cluster": dump_cluster()
	}
}

func add_cluster() {
	mgr := metadata_manager.NewMetadataManager(FrameworkName, zookeeperAddr)
	root_node := mgr.GetRootNode()
	framework.NewFrameworkRiakCluster(root_node, clusterName)
}

func dump_cluster() {
	mgr := metadata_manager.NewMetadataManager(FrameworkName, zookeeperAddr)
	root_node := mgr.GetRootNode()
	clusters := root_node.GetChild("clusters")
	cluster := clusters.GetChild(clusterName)
	frc := framework.FrameworkRiakClusterFromZKNode(cluster)
	fmt.Printf("Cluster: %s\n", frc.Name)
	for key, node := range frc.GetNodes() {
		fmt.Printf("\tNode: %s -> %+v\n", key, node)
	}

}
func add_node() {
	mgr := metadata_manager.NewMetadataManager(FrameworkName, zookeeperAddr)
	root_node := mgr.GetRootNode()
	clusters := root_node.GetChild("clusters")
	cluster := clusters.GetChild(clusterName)
	framework.FrameworkRiakClusterFromZKNode(cluster).AddNode()
}
*/
