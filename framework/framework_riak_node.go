package framework

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	metamgr "github.com/basho/bletchley/metadata_manager"
	"github.com/satori/go.uuid"
)

// Next Status

type FrameworkRiakNode struct {
	frc    *FrameworkRiakCluster `json:"-"`
	zkNode *metamgr.ZkNode       `json:"-"`
	UUID   uuid.UUID
}

func NewFrameworkRiakNode() *FrameworkRiakNode {
	return &FrameworkRiakNode{}
}

func (frn *FrameworkRiakNode) Persist() {
	data, err := json.Marshal(frn)
	if err != nil {
		log.Panic("error:", err)
	}
	frn.zkNode.SetData(data)
}

func (frn *FrameworkRiakNode) GetZkNode() *metamgr.ZkNode {
	return frn.zkNode
}
