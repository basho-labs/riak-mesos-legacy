package scheduler

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/basho-labs/riak-mesos/metadata_manager"
	// Unfortunately, we're leaking abstractions, but sometimes things like this need to be done
	// -Sargun Dhillon
	// TODO: Make metadata manager "better"
	"bytes"
	"compress/zlib"
	"github.com/samuel/go-zookeeper/zk"
)

type SchedulerState struct {
	zkNode      *metadata_manager.ZkNode
	FrameworkID *string
	Clusters    map[string]*FrameworkRiakCluster
}

func emptySchedulerState() *SchedulerState {
	return &SchedulerState{
		Clusters: make(map[string]*FrameworkRiakCluster),
	}
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
	var returnBuffer bytes.Buffer
	w := zlib.NewWriter(&returnBuffer)
	encoder := json.NewEncoder(w)
	encoder.Encode(ss)
	err := w.Close()
	if err != nil {
		log.Panic(err)
	}
	return returnBuffer.Bytes()
}
func (ss *SchedulerState) Persist() error {
	b := ss.serialize()
	err := ss.zkNode.SetData(b)
	return err
}

func deserializeSchedulerState(data []byte) (*SchedulerState, error) {
	r, err := zlib.NewReader(bytes.NewBuffer(data))
	if err != nil {
		log.Panic(err)
	}
	decoder := json.NewDecoder(r)
	t := &SchedulerState{}
	err = decoder.Decode(&t)
	if err != nil {
		log.Panic(err)
	}
	return t, err
}
