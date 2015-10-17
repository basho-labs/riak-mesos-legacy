package scheduler

import (
	log "github.com/Sirupsen/logrus"
	//	metamgr "github.com/basho-labs/riak-mesos/metadata_manager"
	//	"github.com/satori/go.uuid"
)

type FrameworkRiakCluster struct {
	Name           string
	Nodes          map[string]*FrameworkRiakNode
	RiakConfig     string
	AdvancedConfig string
}

func (frc *FrameworkRiakCluster) GetNodes() map[string]*FrameworkRiakNode {
	return frc.Nodes
}

func NewFrameworkRiakCluster(name string) *FrameworkRiakCluster {

	advancedConfig, err := Asset("advanced.config")
	if err != nil {
		log.Error("Unable to open up advanced.config: ", err)
	}
	riakConfig, err := Asset("riak.conf")
	if err != nil {
		log.Error("Unable to open up riak.conf: ", err)
	}

	return &FrameworkRiakCluster{
		Nodes:          make(map[string]*FrameworkRiakNode),
		Name:           name,
		AdvancedConfig: string(advancedConfig),
		RiakConfig:     string(riakConfig),
	}
}
