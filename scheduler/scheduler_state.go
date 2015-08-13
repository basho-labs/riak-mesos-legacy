package scheduler

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/basho-labs/riak-mesos/metadata_manager"
	// Unfortunately, we're leaking abstractions, but sometimes things like this need to be done
	// -Sargun Dhillon
	// TODO: Make metadata manager "better"
	"github.com/samuel/go-zookeeper/zk"
)

type SchedulerState struct {
	zkNode      *metadata_manager.ZkNode
	FrameworkID *string
}

func emptySchedulerState() *SchedulerState {
	return &SchedulerState{}
}
func GetSchedulerState(mm *metadata_manager.MetadataManager) *SchedulerState {
	var zkNode *metadata_manager.ZkNode
	var err error
	mm.GetRootNode().GetChild("SchedulerState")
	zkNode, err = mm.GetRootNode().GetChild("SchedulerState")
	if err == zk.ErrNoNode {
		ess := emptySchedulerState()
		zkNode, err = mm.GetRootNode().MakeChildWithData("SchedulerState", ess.serialize(), false)
		if err != nil {
			log.Panic(err)
		}
		ess.zkNode = zkNode
		return ess
	} else {
		ss, err := deserializeSchedulerState(zkNode.GetData())
		if err != nil {
			log.Panic(err)
		}
		ss.zkNode = zkNode
		return ss
	}
}
func (ss *SchedulerState) serialize() []byte {
	b, err := json.Marshal(ss)
	if err != nil {
		log.Panic(err)
	}
	return b
}
func (ss *SchedulerState) Persist() error {
	b := ss.serialize()
	err := ss.zkNode.SetData(b)
	return err
}

func deserializeSchedulerState(data []byte) (*SchedulerState, error) {
	t := &SchedulerState{}
	err := json.Unmarshal(data, t)
	return t, err
}
