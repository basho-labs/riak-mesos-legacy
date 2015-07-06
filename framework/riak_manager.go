package framework
import (
	"encoding/json"
	"github.com/satori/go.uuid"
	metamgr "github.com/basho/bletchley/metadata_manager"
	log "github.com/Sirupsen/logrus"

)

type FrameworkRiakNode struct {
	rc				*FrameworkRiakCluster `json:"-"`
	zkNode			*metamgr.ZkNode `json:"-"`
	UUID			uuid.UUID
}


func (frn *FrameworkRiakNode) Persist() {
	data, err := json.Marshal(frn)
	if err != nil {
		log.Panic("error:", err)
	}
	frn.zkNode.SetData(data)
}

type FrameworkRiakCluster struct {
	zkNode			*metamgr.ZkNode `json:"-"`
	nodes			map[string]*FrameworkRiakNode `json:"-"`
	// Do not use direct access to properties!
	Name			string
}

func (frc *FrameworkRiakCluster) GetNodes() map[string]*FrameworkRiakNode {
	return frc.nodes
}

func (frc *FrameworkRiakCluster) Persist() {
	data, err := json.Marshal(frc)
	if err != nil {
		log.Panic("error:", err)
	}
	frc.zkNode.SetData(data)
}
func (frc *FrameworkRiakCluster) AddNode() {
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

func FrameworkRiakClusterFromZKNode(node *metamgr.ZkNode) *FrameworkRiakCluster {
	frc := &FrameworkRiakCluster{
		zkNode: node,
		nodes:	make(map[string]*FrameworkRiakNode),
	}
	err := json.Unmarshal(node.GetData(), &frc)
	if err != nil {
		log.Panic("Error getting cluster: ", err)
	}

	node.CreateChildIfNotExists("nodes")
	nodes := node.GetChild("nodes")
	children := nodes.GetChildren()
	for _, value := range children {
		frn := &FrameworkRiakNode{rc: frc, zkNode: value}
		err := json.Unmarshal(value.GetData(), &frn)
		if err != nil {
			log.Panic("Error getting node: ", err)
		}
		frc.nodes[frn.UUID.String()] = frn
	}
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