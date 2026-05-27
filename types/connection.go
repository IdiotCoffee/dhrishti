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

	// Last time any event updated this flow.
	// Used for cleanup and liveness tracking.
	LastUpdated time.Time

	// Whether this flow represents a failed connection lifecycle.
	Failed bool

	// Human-readable reason explaining why the flow failed.
	FailureReason string

	// Whether the connection was extremely short-lived.
	// Useful for retry storm detection later.
	ShortLived bool
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

/*
Duration computes total runtime lifetime
of the flow dynamically.

IMPORTANT:

We intentionally DO NOT store duration
as a separate field.

Why?

Because duration is derived state:

	close_time - connect_time

Storing both timestamps AND duration
creates redundant state and risks
inconsistency bugs.

Instead:
timestamps remain canonical truth,
and duration is derived when needed.
*/
func (c *ConnectionState) Duration() time.Duration {

	// Flow still active.
	//
	// Duration grows dynamically.
	if c.CloseTime.IsZero() {
		return time.Since(c.ConnectTime)
	}

	// Closed flow.
	return c.CloseTime.Sub(c.ConnectTime)
}
