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

func NewFrameworkRiakCluster(name string) *FrameworkRiakCluster {

	return &FrameworkRiakCluster{
		Nodes: make(map[string]*FrameworkRiakNode),
		Name:  name,
	}
}
