package types

import (
	"fmt"
	"sync"
	"time"
)

/*
FlowSample represents ONE completed
connection lifecycle sample.

This becomes the foundation for:
- rolling latency windows
- requests/sec
- p95 calculations
- temporal observability

IMPORTANT:

This is intentionally lightweight.

We do NOT store:
- payloads
- packets
- full traces

Only operationally useful metrics.
*/
type FlowSample struct {

	// When this flow completed.
	Timestamp time.Time

	// Total observed connection lifetime.
	Duration time.Duration

	// Whether this lifecycle failed.
	Failed bool
}

/*
Graph represents the live runtime topology state
of the system.

This is NOT just visualization data anymore.

It is now:
- runtime memory,
- behavioral state,
- live dependency tracking.

The graph owns all discovered service relationships.
*/
type Graph struct {

	/*
		RWMutex protects concurrent access.

		Why needed?

		Because multiple goroutines already exist:
		- tcp_connect pipeline
		- tcp_close pipeline
		- tcp_accept pipeline
		- periodic print loop

		All of them can touch graph state concurrently.

		Without synchronization:
		Go maps will panic with:
		"concurrent map read/write"

		RWMutex allows:
		- many readers simultaneously
		- only one writer at a time
	*/
	Mu sync.RWMutex

	/*
		Edges stores all discovered service relationships.

		Key:
			stable directional dependency identity

		Value:
			runtime edge metadata
	*/
	Edges map[EdgeKey]*Edge
}

/*
EdgeKey uniquely identifies
a service dependency edge.
*/
type EdgeKey struct {
	Source      string
	Destination string
}

/*
Edge models a runtime dependency relationship
between two services.

This is no longer just:

	client -> server

It now stores:
- historical traffic volume
- current live activity
- temporal metadata
- behavioral metrics

This is the beginning of:
behavioral observability.
*/
type Edge struct {

	// Human-readable service names
	Source      string
	Destination string

	// Transport metadata
	Port uint16

	/*
		Total historical connections observed.

		This continuously increases forever.

		Useful for:
		- edge strength
		- traffic volume
		- dependency importance
	*/
	ConnectionCount int

	/*
		Current active live connections.

		CONNECT:
			active++

		CLOSE:
			active--

		This models:
		real live runtime state.
	*/
	ActiveConnections int

	/*
		Temporal metadata.

		Useful later for:
		- stale edge cleanup
		- anomaly detection
		- edge aging
		- topology evolution
	*/
	FirstSeen time.Time
	LastSeen  time.Time

	// Total cumulative duration of all completed flows.
	TotalDuration time.Duration

	// Average observed connection duration.
	AverageDuration time.Duration

	// Number of failed connection attempts.
	FailedConnections int

	// Number of extremely short-lived flows.
	ShortLivedConnections int

	// Number of fully completed flows.
	//
	// Important:
	// average durations should ONLY use
	// finalized connection lifecycles.
	CompletedConnections int
	/*
		Recent completed flow samples.

		This powers:
		- rolling latency metrics
		- RPS
		- failure rate
		- p95 latency

		IMPORTANT:

		We intentionally keep this in-memory.

		This is:
		operational observability state,
		NOT historical persistence.
	*/
	RecentFlows []FlowSample

	/*
		Rolling operational metrics.

		These represent:
		recent runtime behavior.

		Unlike cumulative metrics,
		these continuously evolve over time.
	*/
	RequestsPerSecond float64

	RecentAverageLatency time.Duration

	P95Latency time.Duration

	FailureRate float64
}

/*
NewGraph initializes empty graph state.

This becomes:
the central runtime topology model.
*/
func NewGraph() *Graph {
	return &Graph{
		Edges: make(map[EdgeKey]*Edge),
	}
}

/*
RecordConnect handles CONNECT lifecycle events.

Semantic meaning:

	a service initiated communication.

This:
- creates edges if missing
- increments total historical traffic
- increments live active connections

This is now:
runtime state reconstruction.
*/
func (g *Graph) RecordConnect(
	source string,
	destination string,
	port uint16,
) {

	/*
		Lock for writing.

		We are mutating shared state.
	*/
	g.Mu.Lock()

	/*
		Always unlock even if function exits early.

		defer is extremely important in systems code
		to avoid deadlocks/resource leaks.
	*/
	defer g.Mu.Unlock()

	/*
		Stable edge identity.

		Typed identity is MUCH safer than:
			"source->destination"

		This becomes extremely important
		as runtime semantics evolve.
	*/
	key := EdgeKey{
		Source:      source,
		Destination: destination,
	}

	now := time.Now()

	/*
		If edge already exists:
		update runtime state.
	*/
	if edge, exists := g.Edges[key]; exists {

		/*
			Historical traffic volume.
		*/
		edge.ConnectionCount++

		/*
			Live concurrent connections.
		*/
		edge.ActiveConnections++

		/*
			Refresh temporal activity.
		*/
		edge.LastSeen = now

		return
	}

	/*
		Otherwise:
		create entirely new dependency edge.

		This represents:
		discovery of a new architecture relationship.
	*/
	g.Edges[key] = &Edge{
		Source:      source,
		Destination: destination,

		Port: port,

		ConnectionCount:   1,
		ActiveConnections: 1,

		FirstSeen: now,
		LastSeen:  now,
	}
}

/*
RecordClose handles CLOSE lifecycle events.

Semantic meaning:

	a live runtime connection terminated.

This does NOT:
- remove edge
- decrement historical count

Because:
the relationship still exists historically.

It ONLY updates:
live runtime activity.
*/
func (g *Graph) RecordClose(
	source string,
	destination string,
) {

	/*
		Write lock because we mutate edge state.
	*/
	g.Mu.Lock()
	defer g.Mu.Unlock()

	key := EdgeKey{
		Source:      source,
		Destination: destination,
	}

	/*
		If edge doesn't exist:
		safely ignore.

		This can happen because:
		- missed events
		- startup ordering
		- partial telemetry
		- race conditions

		Real observability systems must tolerate
		incomplete telemetry gracefully.
	*/
	edge, exists := g.Edges[key]
	if !exists {
		return
	}

	/*
		Never allow negative active connections.

		Defensive programming matters heavily
		in infrastructure systems.
	*/
	if edge.ActiveConnections > 0 {
		edge.ActiveConnections--
	}

	/*
		Update temporal metadata.
	*/
	edge.LastSeen = time.Now()
}

/*
Print renders current runtime topology state.

This is currently:
terminal visualization.

Later:
- HTTP API
- WebSocket streaming
- Cytoscape frontend
- GraphQL
- persistence

may all consume the same graph state.
*/
func (g *Graph) Print() {

	/*
		Read lock:
		multiple readers allowed concurrently.

		No mutation occurs here.
	*/
	g.Mu.RLock()
	defer g.Mu.RUnlock()

	fmt.Println("\n=== Live Service Graph ===")

	/*
		Iterate through all known edges.
	*/
	for _, edge := range g.Edges {

		fmt.Printf(
			"%s ─────▶ %s | total=%d | active=%d | avg=%s | failed=%d | short_lived=%d | last_seen=%s\n",

			edge.Source,
			edge.Destination,

			edge.ConnectionCount,
			edge.ActiveConnections,

			edge.AverageDuration,

			edge.FailedConnections,
			edge.ShortLivedConnections,

			edge.LastSeen.Format(time.RFC3339),
		)
	}
}
