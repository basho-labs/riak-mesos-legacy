package scheduler

import (
//	log "github.com/Sirupsen/logrus"
//	metamgr "github.com/basho-labs/riak-mesos/metadata_manager"
//	"github.com/satori/go.uuid"
)

type FrameworkRiakCluster struct {
	Name  string
	Nodes map[string]*FrameworkRiakNode
}

func (frc *FrameworkRiakCluster) GetNodes() map[string]*FrameworkRiakNode {
	return frc.Nodes
}

func (frc *FrameworkRiakCluster) Trigger() {
	// Go through all of the Riak Nodes
	// See if they are running

}

/*
func (frc *FrameworkRiakCluster) NewNode() metamgr.MetadataManagerNode {
	nodes, err := frc.zkNode.GetChild("nodes")
	if err != nil {
		log.Panic(err)
	}
	myUUID := uuid.NewV4()
	zkNode := nodes.MakeEmptyChild(myUUID.String())
	frn := NewFrameworkRiakNode()
	frn.frc = frc
	frn.zkNode = zkNode
	frn.UUID = myUUID

	return frn
}


// This is an add cluster callback from the metadata manager
func (frc *FrameworkRiakCluster) AddNode(zkNode *metamgr.ZkNode) metamgr.MetadataManagerNode {
	frc.sc.lock.Lock()
	defer frc.sc.lock.Unlock()
	frn := NewFrameworkRiakNode()
	frn.frc = frc
	frn.zkNode = zkNode
	err := json.Unmarshal(zkNode.GetData(), &frn)
	if err != nil {
		log.Panic("Error getting node: ", err)
	}
	// This should happen naturally because of Go "initializers"
	frn.reconciled = false
	frc.sc.frnDict[frn.GetTaskStatus().TaskId.GetValue()] = frn
	frc.nodes[frn.UUID.String()] = frn
	frc.sc.rServer.nodesToReconcile <- frn
	return frn
}
*/
func NewFrameworkRiakCluster(name string) *FrameworkRiakCluster {

	return &FrameworkRiakCluster{
		Nodes: make(map[string]*FrameworkRiakNode),
		Name:  name,
	}
}
