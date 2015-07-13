package cluster_state

//go:generate jsonenums -type=ClusterState
//go:generate stringer -type=ClusterState

type ClusterState int

// Although we use JsonEnums / Stringer to prevent serializing values with just ints
// It's wise not to rely on that, and only add enum values _at the end_ --
// Also, never retire an enum, otherwise legacy JSON may fail to deserialize properly

const (
	Unknown ClusterState = iota
	Alone
	Looking
	Joined
	Committing
	Committed
	Leaving
	Left
	Failed
)

// A node goes Alone -> Looking -> Joined -> Leaving -> Left
// We can technically transition a node from Left -> Alone, but I don't think that's ever neccessary


// When a node is added, it _should_ start in the alone phase until the cluster is activated
// At this point, we transition all Alone members to Looking

// If all members are looking we then do riak-admin cross-joins, and move to the joined state
// If any member is committed, we join to it, and move to the joined state

// Once all the members have left the looking state, we run riak-admin cluster commit
// This moves all of the members (including those previously in committed) to committing

// Once the handoffs are completed, we then move all committing members of the cluster to committed

// How leaving / shrinking the cluster works is yet to be determined.

// If a node without persistent storage dies
// Its process state goes to failed
// ProcessState / ClusterState
// Failed / Failed ->
// Starting / Failed
// Started / Failed
// At this point, failed looks like looking
// We follow the same rules -- do nothing if any of the nodes are in committing

// If all of the nodes are in looking, or failed, then proceed
// riak-admin force-replace the failed nodes, and join to a committed node. If there are no committed nodes, then cross-join to all nodes
// Once we do this successfully, we move to the joined state
// and proceed as normal in the FSM.

// Transitions happen as so:
// NewUnknown -> NewEmpty
// NewEmpty -> NewJoined
// NewJoined -> NewCommitting
// NewCommitting -> Running
// Running -> OldEmpty
// OldEmpty -> OldReplaced
// OldReplaced -> OldCommitting
// OldCommitting -> OldRepairing
// OldRepairing -> Running
// Running -> OldLeaving
// OldLeaving -> OldLeft

// New / Old can be persisted in zookeeper
