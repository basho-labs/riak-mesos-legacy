package common

import (
	"encoding/json"
)

type TaskData struct {
	FullyQualifiedNodeName string
	Zookeepers             []string
	ClusterName            string
	NodeID                 string
}

func (s *TaskData) Serialize() ([]byte, error) {
	b, err := json.Marshal(s)
	return b, err
}

func DeserializeTaskData(data []byte) (TaskData, error) {
	t := TaskData{}
	err := json.Unmarshal(data, &t)
	return t, err
}
