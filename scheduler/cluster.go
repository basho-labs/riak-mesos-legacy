package scheduler

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	rexclient "github.com/basho-labs/riak-mesos/riak_explorer/client"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"time"
)

type FrameworkRiakCluster struct {
	Name           string
	Nodes          map[string]*FrameworkRiakNode
	Graveyard      map[string]*FrameworkRiakNode
	RiakConfig     string
	AdvancedConfig string
	IsKilled       bool
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
		Graveyard:      make(map[string]*FrameworkRiakNode),
		Name:           name,
		AdvancedConfig: string(advancedConfig),
		RiakConfig:     string(riakConfig),
		IsKilled:       false,
	}
}

func (frc *FrameworkRiakCluster) KillNext() {
	frc.IsKilled = true
}

func (frc *FrameworkRiakCluster) CanBeRemoved() bool {
	return frc.IsKilled && len(frc.Nodes) == 0
}

func (frc *FrameworkRiakCluster) GetNodes() map[string]*FrameworkRiakNode {
	return frc.Nodes
}

func (frc *FrameworkRiakCluster) GetNextSimpleId() int {
	return len(frc.Nodes) + len(frc.Graveyard) + 1
}

func (frc *FrameworkRiakCluster) CreateNode(sc *SchedulerCore) *FrameworkRiakNode {
	simpleId := frc.GetNextSimpleId()
	riakNode := NewFrameworkRiakNode(sc, frc.Name, simpleId)
	frc.Nodes[riakNode.CurrentID()] = riakNode
	return riakNode
}

func (frc *FrameworkRiakCluster) HasNode(riakNodeID string) bool {
	_, isAlive := frc.Nodes[riakNodeID]
	_, isDead := frc.Graveyard[riakNodeID]
	return isAlive || isDead
}

func (frc *FrameworkRiakCluster) RemoveNode(riakNode *FrameworkRiakNode) {
	log.Infof("Removing node: %+v", riakNode.CurrentID())
	frc.Graveyard[riakNode.CurrentID()] = riakNode
	delete(frc.Nodes, riakNode.CurrentID())
}

func (frc *FrameworkRiakCluster) GetNodesToKillOrRemove() (map[string]*FrameworkRiakNode, map[string]*FrameworkRiakNode) {
	nodesToKill := make(map[string]*FrameworkRiakNode)
	nodesToRemove := make(map[string]*FrameworkRiakNode)

	for _, riakNode := range frc.Nodes {
		if riakNode.CanBeKilled() {
			nodesToKill[riakNode.CurrentID()] = riakNode
		}
		if riakNode.CanBeRemoved() {
			nodesToRemove[riakNode.CurrentID()] = riakNode
		}
	}

	return nodesToKill, nodesToRemove
}

func (frc *FrameworkRiakCluster) GetNodeTasksToReconcile() []*mesos.TaskStatus {
	tasksToReconcile := []*mesos.TaskStatus{}

	for _, riakNode := range frc.Nodes {
		if riakNode.GetTaskStatus() != nil {
			if riakNode.reconciled == false && time.Since(riakNode.lastAskedToReconcile).Seconds() > 5 {
				riakNode.lastAskedToReconcile = time.Now()
				tasksToReconcile = append(tasksToReconcile, riakNode.GetTaskStatus())
			}
		}
	}

	return tasksToReconcile
}

func (frc *FrameworkRiakCluster) HandleNodeStatusUpdate(status *mesos.TaskStatus) {
	deadNode, updateForDeadNode := frc.Graveyard[status.TaskId.GetValue()]

	if updateForDeadNode {
		log.Warnf("Status update is for a node that's already been killed, ignoring. Node: ", deadNode)
		return
	}

	riakNode, _ := frc.Nodes[status.TaskId.GetValue()]
	riakNode.reconciled = true
	riakNode.TaskStatus = status

	switch *status.State.Enum() {
	case mesos.TaskState_TASK_STAGING:
		riakNode.Stage()
	case mesos.TaskState_TASK_STARTING:
		riakNode.Start()
	case mesos.TaskState_TASK_RUNNING:
		frc.Join(riakNode)
	case mesos.TaskState_TASK_FINISHED:
		frc.Leave(riakNode)
		riakNode.Finish()
	case mesos.TaskState_TASK_FAILED:
		// frc.Leave(riakNode)
		riakNode.Fail()
	case mesos.TaskState_TASK_KILLED:
		frc.Leave(riakNode)
		riakNode.Kill()
	case mesos.TaskState_TASK_LOST:
		// frc.Leave(riakNode)
		riakNode.Lost()
	case mesos.TaskState_TASK_ERROR:
		// frc.Leave(riakNode)
		riakNode.Error()
	default:
		log.Warn("Received unknown status update: %+v", status)
	}
}

func doJoin(oldNode *FrameworkRiakNode, newNode *FrameworkRiakNode, retry int, maxRetry int) bool {
	if retry > maxRetry {
		log.Infof("Attempted joining %+v to %+v %+v times and failed.", newNode.TaskData.FullyQualifiedNodeName, oldNode.TaskData.FullyQualifiedNodeName, maxRetry)
		return false
	}

	rexHostname := fmt.Sprintf("%s:%d", oldNode.LastOfferUsed.GetHostname(), oldNode.TaskData.HTTPPort)
	rexc := rexclient.NewRiakExplorerClient(rexHostname)
	// We should try to join against this node
	log.Infof("Joining %+v to %+v", newNode.TaskData.FullyQualifiedNodeName, oldNode.TaskData.FullyQualifiedNodeName)
	joinReply, joinErr := rexc.Join(newNode.TaskData.FullyQualifiedNodeName, oldNode.TaskData.FullyQualifiedNodeName)
	log.Infof("Triggered join: %+v, %+v", joinReply, joinErr)
	if joinReply.Join.Success == "ok" {
		log.Info("Join successful")
		return true
	}
	if joinReply.Join.Error == "not_single_node" {
		log.Info("Node already joined")
		return true
	}

	time.Sleep(5 * time.Second)
	return doJoin(oldNode, newNode, retry+1, maxRetry)
}

func (frc *FrameworkRiakCluster) Join(newNode *FrameworkRiakNode) {
	if !newNode.CanJoinCluster() {
		// The node doesn't want to be part of a cluster?
		log.Infof("Node is now running, but doesn't need to join a cluster right now: %+v", newNode)
		newNode.Run()
		return
	}

	if len(frc.Nodes) == 1 {
		// Cluster of one
		newNode.Run()
		return
	}

	joinSuccess := false
	for _, oldNode := range frc.Nodes {
		if oldNode.CanBeJoined() {
			joinSuccess = doJoin(oldNode, newNode, 0, 5)
			if joinSuccess {
				break
			}
		}
	}

	newNode.Run()

	if !joinSuccess {
		// We're running now, but we can't join the cluster for some reason
		log.Info("Node is now running, but cannot find a node to join.")
	}
}

func doLeave(stayingNode *FrameworkRiakNode, leavingNode *FrameworkRiakNode, retry int, maxRetry int) bool {
	if retry > maxRetry {
		log.Infof("Attempted removing %+v to %+v's cluster %+v times and failed.", leavingNode.TaskData.FullyQualifiedNodeName, stayingNode.TaskData.FullyQualifiedNodeName, maxRetry)
		return false
	}

	rexHostname := fmt.Sprintf("%s:%d", stayingNode.LastOfferUsed.GetHostname(), stayingNode.TaskData.HTTPPort)
	rexc := rexclient.NewRiakExplorerClient(rexHostname)
	// We should try to join against this node
	log.Infof("Removing %+v from %+v's cluster", leavingNode.TaskData.FullyQualifiedNodeName, stayingNode.TaskData.FullyQualifiedNodeName)
	leaveReply, leaveErr := rexc.ForceRemove(stayingNode.TaskData.FullyQualifiedNodeName, leavingNode.TaskData.FullyQualifiedNodeName)
	log.Infof("Triggered force remove: %+v, %+v", leaveReply, leaveErr)
	if leaveReply.ForceRemove.Success == "ok" {
		log.Info("Leave successful")
		return true
	}
	if leaveReply.ForceRemove.Error == "not_member" {
		log.Info("Node already removed")
		return true
	}

	time.Sleep(5 * time.Second)
	return doLeave(stayingNode, leavingNode, retry+1, maxRetry)
}

func (frc *FrameworkRiakCluster) Leave(leavingNode *FrameworkRiakNode) {
	// Cluster of one
	if len(frc.Nodes) == 1 {
		return
	}

	leaveSuccess := false
	for _, stayingNode := range frc.Nodes {
		if stayingNode.CanBeLeft() && stayingNode != leavingNode {
			leaveSuccess = doLeave(stayingNode, leavingNode, 0, 5)
			if leaveSuccess {
				break
			}
		}
	}

	if !leaveSuccess {
		// We're running now, but we can't join the cluster for some reason
		log.Warnf("Attempted to remove node from cluster, but was unable to. Cluster Nodes: %+v", frc.Nodes)
	}
}
