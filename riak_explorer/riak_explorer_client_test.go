package riak_explorer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJoin(t *testing.T) {
	if !riakExplorerAlive() {
		return
	}
	ensureJoined(t, "dev1@127.0.0.1", "dev2@127.0.0.1")
}

// func TestLeave(t *testing.T) {
// 	if !riakExplorerAlive() {
// 		return
// 	}
// 	ensureJoined(t, "dev1@127.0.0.1", "dev2@127.0.0.1")
// 	assert := assert.New(t)
// 	client := NewRiakExplorerClient("localhost:9000")
// 	resp, err := client.Leave("dev1@127.0.0.1", "dev2@127.0.0.1")
//
// 	assert.Equal(nil, err)
// 	assert.Equal("", resp.Leave.Error)
// 	assert.Equal("ok", resp.Leave.Success)
//
// 	var nodeStatuses []NodeStatusReply
// 	nodeStatuses = append(nodeStatuses, NodeStatusReply{ID: "dev1@127.0.0.1", Status: "valid", RingPercentage: 100, PendingPercentage: 0})
// 	assertStatusNodes(t, nodeStatuses)
// 	assertStatusCounts(t, 0, 0, 0, 0, 1)
//
// 	assertRingReady(t)
// }

func TestReplace(t *testing.T) {
	if !riakExplorerAlive() {
		return
	}
	ensureJoined(t, "dev1@127.0.0.1", "dev2@127.0.0.1")
	assert := assert.New(t)
	client := NewRiakExplorerClient("localhost:9000")

	resp, err := client.StagedJoin("dev3@127.0.0.1", "dev1@127.0.0.1")

	assert.Equal(nil, err)
	assert.Equal("", resp.StagedJoin.Error)
	assert.Equal("ok", resp.StagedJoin.Success)

	time.Sleep(1000 * time.Millisecond)

	replaceResp, err := client.Replace("dev1@127.0.0.1", "dev2@127.0.0.1", "dev3@127.0.0.1")

	assert.Equal(nil, err)
	assert.Equal("", replaceResp.Replace.Error)
	assert.Equal("ok", replaceResp.Replace.Success)

	var nodeStatuses []NodeStatusReply
	nodeStatuses = append(nodeStatuses, NodeStatusReply{ID: "dev1@127.0.0.1", Status: "valid", RingPercentage: 50, PendingPercentage: 0})
	nodeStatuses = append(nodeStatuses, NodeStatusReply{ID: "dev3@127.0.0.1", Status: "valid", RingPercentage: 50, PendingPercentage: 0})
	assertStatusNodes(t, nodeStatuses)
	assertStatusCounts(t, 0, 0, 0, 0, 2)

	// assertRingReady(t) // Needs a delay before this passes
}

func ensureJoined(t *testing.T, node1 string, node2 string) {
	assert := assert.New(t)
	client := NewRiakExplorerClient("localhost:9000")

	if isJoined(node1, node2) {
		return
	}

	resp, err := client.Join(node2, node1)

	assert.Equal(nil, err)
	assert.Equal("", resp.Join.Error)
	assert.Equal("ok", resp.Join.Success)

	time.Sleep(1000 * time.Millisecond)

	waitForJoined(node1, node2)

	var nodeStatuses []NodeStatusReply
	nodeStatuses = append(nodeStatuses, NodeStatusReply{ID: node1, Status: "valid", RingPercentage: 50, PendingPercentage: 0})
	nodeStatuses = append(nodeStatuses, NodeStatusReply{ID: node2, Status: "valid", RingPercentage: 50, PendingPercentage: 0})
	assertStatusNodes(t, nodeStatuses)
	assertStatusCounts(t, 0, 0, 0, 0, 2)

	assertRingReady(t)
}

// var commitChanges = false
//
// func maybeCommit(t *testing.T) {
// 	if commitChanges {
// 		assertCommit(t)
// 		time.Sleep(3000 * time.Millisecond)
// 	} else {
// 		assertClear(t)
// 	}
// }
//
// func TestPing(t *testing.T) {
// 	if !riakExplorerAlive() {
// 		return
// 	}
// 	assert := assert.New(t)
// 	client := NewRiakExplorerClient("localhost:9000")
// 	resp, err := client.Ping()
// 	assert.Equal(nil, err)
// 	assert.Equal("pong", resp.Ping.Message)
// }
//
// func TestStagedJoin(t *testing.T) {
// 	if !riakExplorerAlive() {
// 		return
// 	}
// 	ensureStagedJoined(t, "dev1@127.0.0.1", "dev2@127.0.0.1")
// }
//
// func TestStagedLeave(t *testing.T) {
// 	if !riakExplorerAlive() {
// 		return
// 	}
// 	ensureStagedJoined(t, "dev1@127.0.0.1", "dev2@127.0.0.1")
// 	assert := assert.New(t)
// 	client := NewRiakExplorerClient("localhost:9000")
// 	resp, err := client.StagedLeave("dev2@127.0.0.1")
//
// 	assert.Equal(nil, err)
// 	assert.Equal("", resp.StagedLeave.Error)
// 	assert.Equal("ok", resp.StagedLeave.Success)
//
// 	var nodeChanges []NodeChangeReply
// 	nodeChanges = append(nodeChanges, NodeChangeReply{Node: "dev2@127.0.0.1", Action: "leave", Target: ""})
// 	assertPlan(t, nodeChanges)
//
// 	maybeCommit(t)
//
// 	// Status doesn't change until after commit when performing leave
// 	if commitChanges {
// 		var nodeStatuses []NodeStatusReply
// 		nodeStatuses = append(nodeStatuses, NodeStatusReply{ID: "dev2@127.0.0.1", Status: "leaving", RingPercentage: 50, PendingPercentage: 0})
// 		nodeStatuses = append(nodeStatuses, NodeStatusReply{ID: "dev1@127.0.0.1", Status: "valid", RingPercentage: 50, PendingPercentage: 100})
// 		assertStatusNodes(t, nodeStatuses)
// 		assertStatusCounts(t, 0, 0, 0, 1, 1)
// 	}
//
// 	assertRingReady(t)
// }
//
// func TestStagedLeaveTarget(t *testing.T) {
// 	if !riakExplorerAlive() {
// 		return
// 	}
// 	ensureStagedJoined(t, "dev1@127.0.0.1", "dev2@127.0.0.1")
// 	assert := assert.New(t)
// 	client := NewRiakExplorerClient("localhost:9000")
// 	resp, err := client.StagedLeaveTarget("dev1@127.0.0.1", "dev2@127.0.0.1")
//
// 	assert.Equal(nil, err)
// 	assert.Equal("", resp.StagedLeave.Error)
// 	assert.Equal("ok", resp.StagedLeave.Success)
//
// 	var nodeChanges []NodeChangeReply
// 	nodeChanges = append(nodeChanges, NodeChangeReply{Node: "dev2@127.0.0.1", Action: "leave", Target: ""})
// 	assertPlan(t, nodeChanges)
//
// 	maybeCommit(t)
//
// 	// Status doesn't change until after commit when performing leave
// 	if commitChanges {
// 		var nodeStatuses []NodeStatusReply
// 		nodeStatuses = append(nodeStatuses, NodeStatusReply{ID: "dev2@127.0.0.1", Status: "leaving", RingPercentage: 50, PendingPercentage: 0})
// 		nodeStatuses = append(nodeStatuses, NodeStatusReply{ID: "dev1@127.0.0.1", Status: "valid", RingPercentage: 50, PendingPercentage: 100})
// 		assertStatusNodes(t, nodeStatuses)
// 		assertStatusCounts(t, 0, 0, 0, 1, 1)
// 	}
//
// 	assertRingReady(t)
// }
//
// func TestForceRemove(t *testing.T) {
// 	if !riakExplorerAlive() {
// 		return
// 	}
// 	ensureStagedJoined(t, "dev1@127.0.0.1", "dev2@127.0.0.1")
// 	assert := assert.New(t)
// 	client := NewRiakExplorerClient("localhost:9000")
// 	resp, err := client.ForceRemove("dev1@127.0.0.1", "dev2@127.0.0.1")
//
// 	assert.Equal(nil, err)
// 	assert.Equal("", resp.ForceRemove.Error)
// 	assert.Equal("ok", resp.ForceRemove.Success)
//
// 	var nodeChanges []NodeChangeReply
// 	nodeChanges = append(nodeChanges, NodeChangeReply{Node: "dev2@127.0.0.1", Action: "remove", Target: ""})
// 	assertPlan(t, nodeChanges)
//
// 	maybeCommit(t)
//
// 	if commitChanges {
// 		var nodeStatuses []NodeStatusReply
// 		nodeStatuses = append(nodeStatuses, NodeStatusReply{ID: "dev1@127.0.0.1", Status: "valid", RingPercentage: 100, PendingPercentage: 0})
// 		assertStatusNodes(t, nodeStatuses)
// 		assertStatusCounts(t, 0, 0, 0, 0, 1)
// 	}
//
// 	assertRingReady(t)
// }
//
// func TestClear(t *testing.T) {
// 	if !riakExplorerAlive() {
// 		return
// 	}
// 	ensureStagedJoined(t, "dev1@127.0.0.1", "dev2@127.0.0.1")
// 	assert := assert.New(t)
// 	client := NewRiakExplorerClient("localhost:9000")
// 	resp, err := client.StagedLeave("dev2@127.0.0.1")
//
// 	assert.Equal(nil, err)
// 	assert.Equal("", resp.StagedLeave.Error)
// 	assert.Equal("ok", resp.StagedLeave.Success)
//
// 	var nodeChanges []NodeChangeReply
// 	nodeChanges = append(nodeChanges, NodeChangeReply{Node: "dev2@127.0.0.1", Action: "leave", Target: ""})
// 	assertPlan(t, nodeChanges)
//
// 	assertClear(t)
//
// 	var nodeStatuses []NodeStatusReply
// 	nodeStatuses = append(nodeStatuses, NodeStatusReply{ID: "dev1@127.0.0.1", Status: "valid", RingPercentage: 50, PendingPercentage: 0})
// 	nodeStatuses = append(nodeStatuses, NodeStatusReply{ID: "dev2@127.0.0.1", Status: "valid", RingPercentage: 50, PendingPercentage: 0})
// 	assertStatusNodes(t, nodeStatuses)
// 	assertStatusCounts(t, 0, 0, 0, 0, 2)
//
// 	assertRingReady(t)
// }
//
// func TestStagedReplace(t *testing.T) {
// 	if !riakExplorerAlive() {
// 		return
// 	}
// 	ensureStagedJoined(t, "dev1@127.0.0.1", "dev2@127.0.0.1")
// 	assert := assert.New(t)
// 	client := NewRiakExplorerClient("localhost:9000")
//
// 	resp, err := client.StagedJoin("dev3@127.0.0.1", "dev1@127.0.0.1")
//
// 	assert.Equal(nil, err)
// 	assert.Equal("", resp.StagedJoin.Error)
// 	assert.Equal("ok", resp.StagedJoin.Success)
//
// 	time.Sleep(1000 * time.Millisecond)
//
// 	replaceResp, err := client.StagedReplace("dev1@127.0.0.1", "dev2@127.0.0.1", "dev3@127.0.0.1")
//
// 	assert.Equal(nil, err)
// 	assert.Equal("", replaceResp.StagedReplace.Error)
// 	assert.Equal("ok", replaceResp.StagedReplace.Success)
//
// 	var nodeChanges []NodeChangeReply
// 	nodeChanges = append(nodeChanges, NodeChangeReply{Node: "dev2@127.0.0.1", Action: "replace", Target: "dev3@127.0.0.1"})
// 	nodeChanges = append(nodeChanges, NodeChangeReply{Node: "dev3@127.0.0.1", Action: "join", Target: ""})
// 	assertPlan(t, nodeChanges)
//
// 	maybeCommit(t)
//
// 	if commitChanges {
// 		var nodeStatuses []NodeStatusReply
// 		nodeStatuses = append(nodeStatuses, NodeStatusReply{ID: "dev2@127.0.0.1", Status: "leaving", RingPercentage: 50, PendingPercentage: 0})
// 		nodeStatuses = append(nodeStatuses, NodeStatusReply{ID: "dev1@127.0.0.1", Status: "valid", RingPercentage: 50, PendingPercentage: 50})
// 		nodeStatuses = append(nodeStatuses, NodeStatusReply{ID: "dev3@127.0.0.1", Status: "valid", RingPercentage: 0, PendingPercentage: 50})
// 		assertStatusNodes(t, nodeStatuses)
// 		assertStatusCounts(t, 0, 0, 0, 1, 2)
// 	}
//
// 	assertRingReady(t)
// }
//
// func TestForceReplace(t *testing.T) {
// 	if !riakExplorerAlive() {
// 		return
// 	}
// 	ensureStagedJoined(t, "dev1@127.0.0.1", "dev2@127.0.0.1")
// 	assert := assert.New(t)
// 	client := NewRiakExplorerClient("localhost:9000")
//
// 	client.StagedJoin("dev4@127.0.0.1", "dev1@127.0.0.1")
//
// 	time.Sleep(1000 * time.Millisecond)
//
// 	replaceResp, err := client.ForceReplace("dev1@127.0.0.1", "dev2@127.0.0.1", "dev4@127.0.0.1")
//
// 	assert.Equal(nil, err)
// 	assert.Equal("", replaceResp.ForceReplace.Error)
// 	assert.Equal("ok", replaceResp.ForceReplace.Success)
//
// 	var nodeChanges []NodeChangeReply
// 	nodeChanges = append(nodeChanges, NodeChangeReply{Node: "dev2@127.0.0.1", Action: "force_replace", Target: "dev4@127.0.0.1"})
// 	nodeChanges = append(nodeChanges, NodeChangeReply{Node: "dev4@127.0.0.1", Action: "join", Target: ""})
// 	assertPlan(t, nodeChanges)
//
// 	maybeCommit(t)
//
// 	if commitChanges {
// 		var nodeStatuses []NodeStatusReply
// 		nodeStatuses = append(nodeStatuses, NodeStatusReply{ID: "dev1@127.0.0.1", Status: "valid", RingPercentage: 50, PendingPercentage: 0})
// 		nodeStatuses = append(nodeStatuses, NodeStatusReply{ID: "dev4@127.0.0.1", Status: "valid", RingPercentage: 50, PendingPercentage: 0})
// 		assertStatusNodes(t, nodeStatuses)
// 		assertStatusCounts(t, 0, 0, 0, 0, 2)
// 	}
//
// 	assertRingReady(t)
// }
//
// func assertPlan(t *testing.T, changes []NodeChangeReply) {
// 	assert := assert.New(t)
// 	client := NewRiakExplorerClient("localhost:9000")
// 	resp, err := client.Plan("dev1@127.0.0.1")
//
// 	assert.Equal(nil, err)
// 	assert.Equal("", resp.Plan.Error)
// 	assert.Equal(changes, resp.Plan.Changes)
// }
//
// func assertCommit(t *testing.T) {
// 	assert := assert.New(t)
// 	client := NewRiakExplorerClient("localhost:9000")
// 	resp, err := client.Commit("dev1@127.0.0.1")
//
// 	assert.Equal(nil, err)
// 	assert.Equal("", resp.Commit.Error)
// 	assert.Equal("ok", resp.Commit.Success)
// }
//
// func assertClear(t *testing.T) {
// 	assert := assert.New(t)
// 	client := NewRiakExplorerClient("localhost:9000")
// 	resp, err := client.Clear("dev1@127.0.0.1")
//
// 	assert.Equal(nil, err)
// 	assert.Equal("", resp.Clear.Error)
// 	assert.Equal("ok", resp.Clear.Success)
// }
//
// func ensureStagedJoined(t *testing.T, node1 string, node2 string) {
// 	assert := assert.New(t)
// 	client := NewRiakExplorerClient("localhost:9000")
//
// 	if isJoined(node1, node2) {
// 		return
// 	}
//
// 	resp, err := client.StagedJoin(node2, node1)
//
// 	assert.Equal(nil, err)
// 	assert.Equal("", resp.StagedJoin.Error)
// 	assert.Equal("ok", resp.StagedJoin.Success)
//
// 	time.Sleep(1000 * time.Millisecond)
//
// 	var nodeStatuses []NodeStatusReply
// 	nodeStatuses = append(nodeStatuses, NodeStatusReply{ID: node2, Status: "joining", RingPercentage: 0, PendingPercentage: 0})
// 	nodeStatuses = append(nodeStatuses, NodeStatusReply{ID: node1, Status: "valid", RingPercentage: 100, PendingPercentage: 0})
// 	assertStatusNodes(t, nodeStatuses)
// 	assertStatusCounts(t, 0, 0, 1, 0, 1)
//
// 	var nodeChanges []NodeChangeReply
// 	nodeChanges = append(nodeChanges, NodeChangeReply{Node: node2, Action: "join", Target: ""})
// 	assertPlan(t, nodeChanges)
//
// 	assertCommit(t)
//
// 	waitForJoined(node1, node2)
//
// 	assertRingReady(t)
// }

func assertStatusCounts(t *testing.T, down int, exiting int, joining int, leaving int, valid int) {
	assert := assert.New(t)
	client := NewRiakExplorerClient("localhost:9000")
	resp, err := client.Status("dev1@127.0.0.1")

	assert.Equal(nil, err)
	assert.Equal(down, resp.Status.Down)
	assert.Equal(exiting, resp.Status.Exiting)
	assert.Equal(joining, resp.Status.Joining)
	assert.Equal(leaving, resp.Status.Leaving)
	assert.Equal(valid, resp.Status.Valid)
}

func assertStatusNodes(t *testing.T, nodes []NodeStatusReply) {
	assert := assert.New(t)
	client := NewRiakExplorerClient("localhost:9000")
	resp, err := client.Status("dev1@127.0.0.1")

	assert.Equal(nil, err)
	assert.Equal(nodes, resp.Status.Nodes)
}

func assertRingReady(t *testing.T) {
	assert := assert.New(t)
	client := NewRiakExplorerClient("localhost:9000")
	resp, err := client.RingReady("dev1@127.0.0.1")

	assert.Equal(nil, err)
	assert.Equal(true, resp.RingReady.Ready)
}

func isJoined(node1 string, node2 string) bool {
	client := NewRiakExplorerClient("localhost:9000")

	statusResp, _ := client.Status(node1)
	var alreadyJoined []NodeStatusReply
	alreadyJoined = append(alreadyJoined, NodeStatusReply{ID: node1, Status: "valid", RingPercentage: 50, PendingPercentage: 0})
	alreadyJoined = append(alreadyJoined, NodeStatusReply{ID: node2, Status: "valid", RingPercentage: 50, PendingPercentage: 0})
	return testStatusEq(statusResp.Status.Nodes, alreadyJoined)
}

func waitForJoined(node1 string, node2 string) {
	if isJoined(node1, node2) {
		return
	}

	time.Sleep(1000 * time.Millisecond)
	waitForJoined(node1, node2)
}

func riakExplorerAlive() bool {
	client := NewRiakExplorerClient("localhost:9000")
	resp, _ := client.Ping()
	return resp.Ping.Message == "pong"
}

func testStatusEq(a, b []NodeStatusReply) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
