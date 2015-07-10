package riak_node_states

//go:generate jsonenums -type=ProcessState
//go:generate stringer -type=ProcessState

type ProcessState int

const (
	ProcessStateUnknown ProcessState = 0
	ProcessStateStarting = 1
	ProcessStateStarted = 2
	ProcessStateShuttingDown = 3
	ProcessStateShutdown = 4

	Unknown = 0
	Starting = 1
	Started = 2
	ShuttingDown = 3
	Shutdown = 4

)

type RiakClusterState int

const (
	RiakClusterStateEmpty RiakClusterState = iota
)
