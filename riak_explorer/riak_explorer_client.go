package riak_explorer

//go:generate go-bindata -ignore=Makefile -o bindata_generated.go data/

import (
	json "encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Links are location metadata about a Riak Explorer resource
type Links struct {
	Self    string `json:"self"`
	Related string `json:"related"`
}

// RiakExplorerClient contains information common to all Riak Explorer requests
type RiakExplorerClient struct {
	Host string
}

// NewRiakExplorerClient creates a client struct to be used for future calls
func NewRiakExplorerClient(host string) *RiakExplorerClient {
	c := &RiakExplorerClient{
		Host: host,
	}

	return c
}

// PingType is the expected result struct of a ping request
type PingType struct {
	Ping struct {
		ID      string `json:"id"`
		Message string `json:"message"`
	} `json:"ping"`
	Links Links `json:"links"`
}

// Ping provides general health information about Riak Explorer
func (client *RiakExplorerClient) Ping() (PingType, error) {
	var m PingType
	v, err := client.doGet("explore/ping")
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// JoinType is the expected result struct of a join request
type JoinType struct {
	Join struct {
		Success string `json:"success"`
		Error   string `json:"error"`
	} `json:"join"`
	Links Links `json:"links"`
}

// Join instructs fromNode to join toNode's cluster
func (client *RiakExplorerClient) Join(fromNode string, toNode string) (JoinType, error) {
	var m JoinType
	commandURI := fmt.Sprintf("control/nodes/%s/join/%s", fromNode, toNode)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// LeaveType is the expected result struct of a leave request
type LeaveType struct {
	Leave struct {
		Success string `json:"success"`
		Error   string `json:"error"`
	} `json:"leave"`
	Links Links `json:"links"`
}

// Leave instructs node to leave its current cluster
func (client *RiakExplorerClient) Leave(node string) (LeaveType, error) {
	var m LeaveType
	commandURI := fmt.Sprintf("control/nodes/%s/leave", node)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// LeaveTarget instructs leavingNode to leave stayingNode's cluster
func (client *RiakExplorerClient) LeaveTarget(stayingNode string, leavingNode string) (LeaveType, error) {
	var m LeaveType
	commandURI := fmt.Sprintf("control/nodes/%s/leave/%s", stayingNode, leavingNode)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// ForceRemoveType is the expected result struct of a force remove request
type ForceRemoveType struct {
	ForceRemove struct {
		Success string `json:"success"`
		Error   string `json:"error"`
	} `json:"force-remove"`
	Links Links `json:"links"`
}

// ForceRemove instructs leavingNode to leave stayingNode's cluster
func (client *RiakExplorerClient) ForceRemove(stayingNode string, leavingNode string) (ForceRemoveType, error) {
	var m ForceRemoveType
	commandURI := fmt.Sprintf("control/nodes/%s/force-remove/%s", stayingNode, leavingNode)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// ReplaceType is the expected result struct of a replace request
type ReplaceType struct {
	Replace struct {
		Success string `json:"success"`
		Error   string `json:"error"`
	} `json:"replace"`
	Links Links `json:"links"`
}

// Replace instructs fromNode to replace oldNode with newNode
func (client *RiakExplorerClient) Replace(fromNode string, oldNode string, newNode string) (ReplaceType, error) {
	var m ReplaceType
	commandURI := fmt.Sprintf("control/nodes/%s/replace/%s/%s", fromNode, oldNode, newNode)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// ForceReplaceType is the expected result struct of a replace request
type ForceReplaceType struct {
	ForceReplace struct {
		Success string `json:"success"`
		Error   string `json:"error"`
	} `json:"force-replace"`
	Links Links `json:"links"`
}

// ForceReplace instructs fromNode to replace oldNode with newNode
func (client *RiakExplorerClient) ForceReplace(fromNode string, oldNode string, newNode string) (ForceReplaceType, error) {
	var m ForceReplaceType
	commandURI := fmt.Sprintf("control/nodes/%s/force-replace/%s/%s", fromNode, oldNode, newNode)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// NodeChangeType is the expected result struct of a plan request
type NodeChangeType struct {
	Node   string `json:"node"`
	Action string `json:"action"`
	Target string `json:"target"`
}

// PlanType is the expected result struct of a plan request
type PlanType struct {
	Plan struct {
		Changes []NodeChangeType `json:"changes"`
		Error   string           `json:"error"`
	} `json:"plan"`
	Links Links `json:"links"`
}

// Plan instructs node to perform a cluster plan changes
func (client *RiakExplorerClient) Plan(node string) (PlanType, error) {
	var m PlanType
	commandURI := fmt.Sprintf("control/nodes/%s/plan", node)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// CommitType is the expected result struct of a commit request
type CommitType struct {
	Commit struct {
		Success string `json:"success"`
		Error   string `json:"error"`
	} `json:"commit"`
	Links Links `json:"links"`
}

// Commit instructs node to commit its current planned cluster changes
func (client *RiakExplorerClient) Commit(node string) (CommitType, error) {
	var m CommitType
	commandURI := fmt.Sprintf("control/nodes/%s/commit", node)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// ClearType is the expected result struct of a clear request
type ClearType struct {
	Clear struct {
		Success string `json:"success"`
		Error   string `json:"error"`
	} `json:"clear"`
	Links Links `json:"links"`
}

// Clear instructs node to clear its current planned cluster changes
func (client *RiakExplorerClient) Clear(node string) (ClearType, error) {
	var m ClearType
	commandURI := fmt.Sprintf("control/nodes/%s/clear", node)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// NodeStatusType contains data about each node in a status request
type NodeStatusType struct {
	ID                string  `json:"id"`
	Status            string  `json:"status"`
	RingPercentage    float32 `json:"ring_percentage"`
	PendingPercentage float32 `json:"pending_percentage"`
}

// StatusType is the expected result struct of a status request
type StatusType struct {
	Status struct {
		Nodes   []NodeStatusType `json:"nodes"`
		Valid   int              `json:"valid"`
		Leaving int              `json:"leaving"`
		Exiting int              `json:"exiting"`
		Joining int              `json:"joining"`
		Down    int              `json:"down"`
	} `json:"status"`
	Links Links `json:"links"`
}

// Status instructs node to status its current cluster
func (client *RiakExplorerClient) Status(node string) (StatusType, error) {
	var m StatusType
	commandURI := fmt.Sprintf("control/nodes/%s/status", node)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// RingReadyType is the expected result struct of a ringReady request
type RingReadyType struct {
	RingReady struct {
		Ready bool     `json:"ready"`
		Nodes []string `json:"nodes"`
		Error string   `json:"error"`
	} `json:"ringready"`
	Links Links `json:"links"`
}

// RingReady instructs node to ringReady its current cluster
func (client *RiakExplorerClient) RingReady(node string) (RingReadyType, error) {
	var m RingReadyType
	commandURI := fmt.Sprintf("control/nodes/%s/ringready", node)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

func (client *RiakExplorerClient) doGet(path string) ([]byte, error) {
	commandURL := fmt.Sprintf("http://%s/%s", client.Host, path)
	resp, err := http.Get(commandURL)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	return body, nil
}
