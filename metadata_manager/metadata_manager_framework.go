package metadata_manager

import (
	log "github.com/Sirupsen/logrus"
	"github.com/samuel/go-zookeeper/zk"
)

type MetadataManagerFramework interface {
	AddCluster(*ZkNode)		MetadataManagerCluster
	GetCluster(string)		MetadataManagerCluster
	NewCluster(*ZkNode, string)	MetadataManagerCluster
}
type MetadataManagerCluster interface {
	GetZkNode()			*ZkNode
	AddNode(*ZkNode)	MetadataManagerNode
	NewNode()           MetadataManagerNode
	Persist()
}
type MetadataManagerNode interface {
	GetZkNode()			*ZkNode
	Persist()
}


func (mgr *MetadataManager) CreateNode(cluster MetadataManagerCluster) MetadataManagerNode {
	nodesNS := makeSubSpace(cluster.GetZkNode().ns, "nodes")
	mgr.CreateNSIfNotExists(nodesNS)
	node := cluster.NewNode()
	node.Persist()
	cluster.AddNode(node.GetZkNode())
	return node
}
func (mgr *MetadataManager) CreateCluster(name string) MetadataManagerCluster {
	clustersNS := makeSubSpace(mgr.namespace, "clusters")
	mgr.CreateNSIfNotExists(clustersNS)
	newClusterNS := makeSubSpace(clustersNS, name)
	node := &ZkNode{mgr: mgr, ns: newClusterNS}
	cluster := mgr.framework.NewCluster(node, name)
	cluster.Persist()
	mgr.framework.AddCluster(node)
	return cluster
}
func (mgr *MetadataManager) SetupFramework(URI string, mmf MetadataManagerFramework) {
	mgr.framework = mmf
	err := mgr.zkLock.Lock()
	if err != nil {
		log.Panic("Unable to get framework lock: ", err)
	}
	URIPath := makeSubSpace(mgr.namespace, "uri")
	_, err = mgr.zkConn.Create(URIPath.GetZKPath(), []byte(URI), zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	if err != nil {
		log.Panic(err)
	}


	clustersPath := makeSubSpace(mgr.namespace, "clusters")
	mgr.CreateNSIfNotExists(clustersPath)
	clusters, clustersEventChannel := mgr.getChildrenW(clustersPath)
	go func() {
		for clusterEvent := range clustersEventChannel {
			switch (clusterEvent.Type) {
                case zk.EventNodeChildrenChanged: log.Debugf("Cluster event received, and not yet implemented: %+v at path: %+v, state: %+v", clusterEvent, clusterEvent.Path, clusterEvent.State)
				default: log.Debugf("Cluster event received, and not yet implemented: %+v", clusterEvent)
			}
		}
	}()
	for _, clusterZKNode := range clusters {
		cluster := mmf.AddCluster(clusterZKNode)
		nodesPath := makeSubSpace(clusterZKNode.ns, "nodes")
		mgr.CreateNSIfNotExists(nodesPath)
		nodes, nodeEventChannel := mgr.getChildrenW(nodesPath)
		go func() {
			for nodeEvent := range nodeEventChannel {
				log.Debugf("Node event received for cluster: %v, and not yet implemented: %v", cluster, nodeEvent)
			}
		}()
		for _, nodeZKNode := range nodes {
			cluster.AddNode(nodeZKNode)
		}
	}
}