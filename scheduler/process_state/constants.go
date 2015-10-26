package process_state

//go:generate jsonenums -type=ProcessState
//go:generate stringer -type=ProcessState

type ProcessState int

// Although we use JsonEnums / Stringer to prevent serializing values with just ints
// It's wise not to rely on that, and only add enum values _at the end_ --
// Also, never retire an enum, otherwise legacy JSON may fail to deserialize properly
const (
	Unknown ProcessState = iota
	ReservationRequested
	Starting
	Started
	ShuttingDown
	Shutdown
	Failed
)
