package scheduler

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	metamgr "github.com/basho-labs/riak-mesos/metadata_manager"
	"github.com/satori/go.uuid"
)

type FrameworkRiakCluster struct {
	sc     *SchedulerCore
	zkNode *metamgr.ZkNode               `json:"-"`
	nodes  map[string]*FrameworkRiakNode `json:"-"`
	// Do not use direct access to properties!
	Name string
}

func (frc *FrameworkRiakCluster) GetNodes() map[string]*FrameworkRiakNode {
	return frc.nodes
}

func (frc *FrameworkRiakCluster) Trigger() {
	// Go through all of the Riak Nodes
	// See if they are running

}
func (frc *FrameworkRiakCluster) Persist() {
	data, err := json.Marshal(frc)
	if err != nil {
		log.Panic("error:", err)
	}
	frc.zkNode.SetData(data)
}

func (frc *FrameworkRiakCluster) GetZkNode() *metamgr.ZkNode {
	return frc.zkNode
}

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
	frc.sc.frnDict[frn.GetTaskStatus().TaskId.GetValue()] = frn
	frc.nodes[frn.UUID.String()] = frn
	frc.sc.rServer.tasksToReconcile <- frn.GetTaskStatus()
	return frn
}
func NewFrameworkRiakCluster() *FrameworkRiakCluster {

	return &FrameworkRiakCluster{
		nodes: make(map[string]*FrameworkRiakNode),
	}
}
