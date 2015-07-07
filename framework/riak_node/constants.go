package riak_node

type DestinationState int

const (
	Starting     DestinationState = iota
	Started                       = iota
	ShuttingDown                  = iota
	Down                          = iota
)
