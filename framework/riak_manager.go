package framework

	/*
	frc.zkNode.CreateChildIfNotExists("nodes")
	nodes := frc.zkNode.GetChild("nodes")
	nodeUUID := uuid.NewV4()
	zkNode := nodes.MakeChild(nodeUUID.String())
	frn := &FrameworkRiakNode{
		rc: frc,
		zkNode: zkNode,
		UUID: nodeUUID,
	}
	frn.Persist()
}
*/

/*
func FrameworkRiakClusterFromZKNode(clusterZkNode *metamgr.ZkNode) *FrameworkRiakCluster {
	frc := &FrameworkRiakCluster{
		zkNode: clusterZkNode,
		nodes:	make(map[string]*FrameworkRiakNode),
	}
	err := json.Unmarshal(clusterZkNode.GetData(), &frc)
	if err != nil {
		log.Panic("Error getting cluster: ", err)
	}

	clusterChildren, watchChan := clusterZkNode.GetChildrenW()
	for _, child := range clusterChildren {
		log.Infof("Saw child: %v", child)
	}

	clusterZkNode.CreateChildIfNotExists("nodes")
	nodes := clusterZkNode.GetChild("nodes")
	children := nodes.GetChildren()

	for _, value := range children {
		frn := &FrameworkRiakNode{rc: frc, zkNode: value}
		err := json.Unmarshal(value.GetData(), &frn)
		if err != nil {
			log.Panic("Error getting node: ", err)
		}
		frc.nodes[frn.UUID.String()] = frn
	}
	go func() {
		for msg := range watchChan  {
			log.Infof("Received event: %v for NS: %v", msg, clusterZkNode)
		}
	}()
	return frc
}

func NewFrameworkRiakCluster(root_node *metamgr.ZkNode, name string) *FrameworkRiakCluster {
	root_node.CreateChildIfNotExists("clusters")
	clusters := root_node.GetChild("clusters")

	// Some error detection to ensure we don't recreate a cluster
	cluster_node := clusters.MakeChild(name)
	frc := &FrameworkRiakCluster{
		zkNode: cluster_node,
		nodes:  make(map[string]*FrameworkRiakNode),
		Name:	name,
	}
	frc.Persist()
	return frc
}
*/
