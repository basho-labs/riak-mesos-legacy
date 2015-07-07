package riak_node_states

//go:generate jsonenums -type=State
//go:generate stringer -type=State

type State int

const (
	Unknown      State = iota
	Starting
	Started
	ShuttingDown
	Shutdown
)
