package types

import (
	"sync"
	"time"
)

/*
ConnectionState models the lifecycle
of ONE runtime TCP connection.

This is:
stateful observability.

NOT just telemetry anymore.
*/
type ConnectionState struct {

	// stable identity
	Key FlowKey

	// semantic services
	SourceService      string
	DestinationService string

	/*
		Lifecycle timestamps.
	*/
	ConnectTime time.Time
	AcceptTime  time.Time
	CloseTime   time.Time

	/*
		Runtime state flags.
	*/
	Accepted bool
	Closed   bool

	/*
		Computed duration.

		Computed during CLOSE.
	*/
	Duration time.Duration
}

/*
FlowTracker owns all active runtime flows.

This becomes:
live runtime memory.
*/
type FlowTracker struct {

	/*
		Multiple pipelines mutate shared state.

		Need synchronization.
	*/
	Mu sync.RWMutex

	/*
		All active/known flows.
	*/
	Flows map[FlowKey]*ConnectionState
}

func NewFlowTracker() *FlowTracker {
	return &FlowTracker{
		Flows: make(map[FlowKey]*ConnectionState),
	}
}
