package common

import (
	"encoding/json"
)

type TaskData struct {
	FullyQualifiedNodeName string
	Zookeepers             []string
	NodeID                 string
	FrameworkName          string
	ClusterName            string
	URI                    string
	UseSuperChroot         bool
	HTTPPort               int64
	PBPort                 int64
	HandoffPort            int64
	DisterlPort            int64
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
	FrameworkName string
	ClusterName   string
	NodeName      string
	DisterlPort   int
	PBPort        int
	HTTPPort      int
	Hostname      string
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
	NodeName    string
	DisterlPort int
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

type TaskStatusData struct {
	RexPort int64
}

func (s *TaskStatusData) Serialize() ([]byte, error) {
	b, err := json.Marshal(s)
	return b, err
}

func DeserializeTaskStatusData(data []byte) (TaskStatusData, error) {
	t := TaskStatusData{}
	err := json.Unmarshal(data, &t)
	return t, err
}
