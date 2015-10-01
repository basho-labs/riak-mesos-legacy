package riak_explorer

import (
	json "encoding/json"
	"errors"
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

// PingReply is the expected result struct of a ping request
type PingReply struct {
	Ping struct {
		ID      string `json:"id"`
		Message string `json:"message"`
	} `json:"ping"`
	Links Links `json:"links"`
}

// Ping provides general health information about Riak Explorer
func (client *RiakExplorerClient) Ping() (PingReply, error) {
	var m PingReply
	v, err := client.doGet("explore/ping")
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// JoinReply is the expected result struct of a join request
type JoinReply struct {
	Join struct {
		Success string `json:"success"`
		Error   string `json:"error"`
	} `json:"join"`
	Links Links `json:"links"`
}

// Join instructs fromNode to join toNode's cluster immediately
func (client *RiakExplorerClient) Join(fromNode string, toNode string) (JoinReply, error) {
	var m JoinReply
	commandURI := fmt.Sprintf("control/nodes/%s/join/%s", fromNode, toNode)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// StagedJoinReply is the expected result struct of a join request
type StagedJoinReply struct {
	StagedJoin struct {
		Success string `json:"success"`
		Error   string `json:"error"`
	} `json:"staged-join"`
	Links Links `json:"links"`
}

// StagedJoin instructs fromNode to stage a join toNode's cluster
func (client *RiakExplorerClient) StagedJoin(fromNode string, toNode string) (StagedJoinReply, error) {
	var m StagedJoinReply
	commandURI := fmt.Sprintf("control/nodes/%s/staged-join/%s", fromNode, toNode)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// LeaveReply is the expected result struct of a leave request
type LeaveReply struct {
	Leave struct {
		Success string `json:"success"`
		Error   string `json:"error"`
	} `json:"leave"`
	Links Links `json:"links"`
}

// Leave instructs leavingNode to leave stayingNode's cluster immediately
func (client *RiakExplorerClient) Leave(stayingNode string, leavingNode string) (LeaveReply, error) {
	var m LeaveReply
	commandURI := fmt.Sprintf("control/nodes/%s/leave/%s", stayingNode, leavingNode)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// StagedLeaveReply is the expected result struct of a leave request
type StagedLeaveReply struct {
	StagedLeave struct {
		Success string `json:"success"`
		Error   string `json:"error"`
	} `json:"staged-leave"`
	Links Links `json:"links"`
}

// StagedLeave instructs node to stage a leave from its current cluster
func (client *RiakExplorerClient) StagedLeave(node string) (StagedLeaveReply, error) {
	var m StagedLeaveReply
	commandURI := fmt.Sprintf("control/nodes/%s/staged-leave", node)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// StagedLeaveTarget instructs leavingNode to stage a leave stayingNode's cluster
func (client *RiakExplorerClient) StagedLeaveTarget(stayingNode string, leavingNode string) (StagedLeaveReply, error) {
	var m StagedLeaveReply
	commandURI := fmt.Sprintf("control/nodes/%s/staged-leave/%s", stayingNode, leavingNode)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// ForceRemoveReply is the expected result struct of a force remove request
type ForceRemoveReply struct {
	ForceRemove struct {
		Success string `json:"success"`
		Error   string `json:"error"`
	} `json:"force-remove"`
	Links Links `json:"links"`
}

// ForceRemove instructs leavingNode to leave stayingNode's cluster immediately
func (client *RiakExplorerClient) ForceRemove(stayingNode string, leavingNode string) (ForceRemoveReply, error) {
	var m ForceRemoveReply
	commandURI := fmt.Sprintf("control/nodes/%s/force-remove/%s", stayingNode, leavingNode)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// ReplaceReply is the expected result struct of a replace request
type ReplaceReply struct {
	Replace struct {
		Success string `json:"success"`
		Error   string `json:"error"`
	} `json:"replace"`
	Links Links `json:"links"`
}

// Replace instructs fromNode to replace oldNode with newNode immediately
func (client *RiakExplorerClient) Replace(fromNode string, oldNode string, newNode string) (ReplaceReply, error) {
	var m ReplaceReply
	commandURI := fmt.Sprintf("control/nodes/%s/replace/%s/%s", fromNode, oldNode, newNode)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// StagedReplaceReply is the expected result struct of a replace request
type StagedReplaceReply struct {
	StagedReplace struct {
		Success string `json:"success"`
		Error   string `json:"error"`
	} `json:"staged-replace"`
	Links Links `json:"links"`
}

// StagedReplace instructs fromNode to stage a replace for oldNode with newNode
func (client *RiakExplorerClient) StagedReplace(fromNode string, oldNode string, newNode string) (StagedReplaceReply, error) {
	var m StagedReplaceReply
	commandURI := fmt.Sprintf("control/nodes/%s/staged-replace/%s/%s", fromNode, oldNode, newNode)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// ForceReplaceReply is the expected result struct of a replace request
type ForceReplaceReply struct {
	ForceReplace struct {
		Success string `json:"success"`
		Error   string `json:"error"`
	} `json:"force-replace"`
	Links Links `json:"links"`
}

// ForceReplace instructs fromNode to replace oldNode with newNode
func (client *RiakExplorerClient) ForceReplace(fromNode string, oldNode string, newNode string) (ForceReplaceReply, error) {
	var m ForceReplaceReply
	commandURI := fmt.Sprintf("control/nodes/%s/force-replace/%s/%s", fromNode, oldNode, newNode)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// NodeChangeReply is the expected result struct of a plan request
type NodeChangeReply struct {
	Node   string `json:"node"`
	Action string `json:"action"`
	Target string `json:"target"`
}

// PlanReply is the expected result struct of a plan request
type PlanReply struct {
	Plan struct {
		Changes []NodeChangeReply `json:"changes"`
		Error   string            `json:"error"`
	} `json:"plan"`
	Links Links `json:"links"`
}

// Plan instructs node to perform a cluster plan changes
func (client *RiakExplorerClient) Plan(node string) (PlanReply, error) {
	var m PlanReply
	commandURI := fmt.Sprintf("control/nodes/%s/plan", node)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// RepairReply is the expected result struct of a repair request
type RepairReply struct {
	Repair struct {
		Success int `json:"success"`
		Error   int `json:"failure"`
	} `json:"repair"`
	Links Links `json:"links"`
}

// Repair instructs node to repair its partitions
func (client *RiakExplorerClient) Repair(node string) (RepairReply, error) {
	var m RepairReply
	commandURI := fmt.Sprintf("control/nodes/%s/repair", node)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// CommitReply is the expected result struct of a commit request
type CommitReply struct {
	Commit struct {
		Success string `json:"success"`
		Error   string `json:"error"`
	} `json:"commit"`
	Links Links `json:"links"`
}

// Commit instructs node to commit its current planned cluster changes
func (client *RiakExplorerClient) Commit(node string) (CommitReply, error) {
	var m CommitReply
	commandURI := fmt.Sprintf("control/nodes/%s/commit", node)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// ClearReply is the expected result struct of a clear request
type ClearReply struct {
	Clear struct {
		Success string `json:"success"`
		Error   string `json:"error"`
	} `json:"clear"`
	Links Links `json:"links"`
}

// Clear instructs node to clear its current planned cluster changes
func (client *RiakExplorerClient) Clear(node string) (ClearReply, error) {
	var m ClearReply
	commandURI := fmt.Sprintf("control/nodes/%s/clear", node)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// NodeStatusReply contains data about each node in a status request
type NodeStatusReply struct {
	ID                string  `json:"id"`
	Status            string  `json:"status"`
	RingPercentage    float32 `json:"ring_percentage"`
	PendingPercentage float32 `json:"pending_percentage"`
}

// StatusReply is the expected result struct of a status request
type StatusReply struct {
	Status struct {
		Nodes   []NodeStatusReply `json:"nodes"`
		Valid   int               `json:"valid"`
		Leaving int               `json:"leaving"`
		Exiting int               `json:"exiting"`
		Joining int               `json:"joining"`
		Down    int               `json:"down"`
	} `json:"status"`
	Links Links `json:"links"`
}

// Status instructs node to status its current cluster
func (client *RiakExplorerClient) Status(node string) (StatusReply, error) {
	var m StatusReply
	commandURI := fmt.Sprintf("control/nodes/%s/status", node)
	v, err := client.doGet(commandURI)
	if err != nil {
		return m, err
	}
	json.Unmarshal(v, &m)
	return m, nil
}

// RingReadyReply is the expected result struct of a ringReady request
type RingReadyReply struct {
	RingReady struct {
		Ready bool     `json:"ready"`
		Nodes []string `json:"nodes"`
		Error string   `json:"error"`
	} `json:"ringready"`
	Links Links `json:"links"`
}

// RingReady instructs node to ringReady its current cluster
func (client *RiakExplorerClient) RingReady(node string) (RingReadyReply, error) {
	var m RingReadyReply
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
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return body, nil
	} else {
		return body, errors.New(fmt.Sprintf("Unknown HTTP Status: %d", resp.StatusCode))
	}
	return body, nil
}
