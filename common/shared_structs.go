package common

import (
	"encoding/json"
)

type TaskData struct {
	FullyQualifiedNodeName    string
	RexFullyQualifiedNodeName string
	Zookeepers                []string
	ClusterName               string
	NodeID                    string
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

type CoordinatedData struct {
	NodeName string
	DisterlPort	int
}

func (s *CoordinatedData) Serialize() ([]byte, error) {
	b, err := json.Marshal(s)
	return b, err
}

func DeserializeCoordinatedData(data []byte) (CoordinatedData, error) {
	t := CoordinatedData{}
	err := json.Unmarshal(data, &t)
	return t, err
}


type DisterlData struct {
	NodeName string
	DisterlPort	int
}

func (s *DisterlData) Serialize() ([]byte, error) {
	b, err := json.Marshal(s)
	return b, err
}

func DeserializeDisterlData(data []byte) (DisterlData, error) {
	t := DisterlData{}
	err := json.Unmarshal(data, &t)
	return t, err
}
