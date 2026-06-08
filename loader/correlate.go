package loader

import (
	"dhrishti/config"
	"dhrishti/resolver"
	"dhrishti/types"
	"log"
	"time"
)

/*
BuildFlowKey converts runtime events
into canonical flow identity.
*/
func BuildFlowKey(
	event *resolver.EnrichedRuntimeEvent,
) types.FlowKey {
	key := types.FlowKey{
		SourceIP:   event.SourceIP,
		SourcePort: event.SourcePort,

		DestinationIP:   event.DestinationIP,
		DestinationPort: event.DestinationPort,
	}

	// Canonicalize tuple orientation so CONNECT/ACCEPT/CLOSE from either side
	// map to the same flow key.
	if key.SourceIP > key.DestinationIP ||
		(key.SourceIP == key.DestinationIP && key.SourcePort > key.DestinationPort) {
		return key.Reverse()
	}

	return key
}

/*
HandleConnect creates runtime flow state.

This represents:
client-side connection initiation.
*/
func HandleConnect(
	tracker *types.FlowTracker,
	event *resolver.EnrichedRuntimeEvent,
) {

	key := BuildFlowKey(event)

	tracker.Mu.Lock()
	defer tracker.Mu.Unlock()

	tracker.Flows[key] = &types.ConnectionState{
		Key: key,

		SourceService:      event.SourceService,
		DestinationService: event.DestinationService,

		ConnectTime: time.Now(),

		// Tracks latest lifecycle activity.
		// Used for flow expiration cleanup.
		LastUpdated: time.Now(),
	}
}

/*
RecordNewConnection updates flow and graph state for a connect event.

Never hold tracker.Mu and graph.Mu at the same time — nested locks
deadlock with InferStaleFlows and stall all telemetry pipelines.
*/
func RecordNewConnection(
	tracker *types.FlowTracker,
	graph *types.Graph,
	event *resolver.EnrichedRuntimeEvent,
) {
	HandleConnect(tracker, event)

	graph.RecordConnect(
		event.SourceService,
		event.DestinationService,
		event.DestinationPort,
	)
}

/*
HandleAccept marks server-side acceptance.

IMPORTANT:
accept events arrive reversed.

So:
we reverse flow identity first.
*/
func HandleAccept(
	tracker *types.FlowTracker,
	event *resolver.EnrichedRuntimeEvent,
) {
	key := BuildFlowKey(event)

	tracker.Mu.Lock()
	defer tracker.Mu.Unlock()

	flow, exists := tracker.Flows[key]
	if !exists {
		return
	}

	flow.Accepted = true
	flow.AcceptTime = time.Now()

	// Refresh runtime liveness timestamp.
	flow.LastUpdated = time.Now()
}

/*
HandleClose finalizes runtime connection state.

Computes:
- duration
- failure semantics
- behavioral edge metrics
*/
func HandleClose(
	tracker *types.FlowTracker,
	graph *types.Graph,
	cfg config.Config,
	unknown *types.UnknownIPRegistry,
	event *resolver.EnrichedRuntimeEvent,
) {

	// Close events may arrive from either side:
	//
	// client -> server
	// OR
	// server -> client
	//
	// So we first try direct lookup,
	// then reversed lookup.

	key := BuildFlowKey(event)

	tracker.Mu.Lock()

	flow, exists := tracker.Flows[key]

	// If direct lookup fails,
	// try reversed flow identity.
	if !exists {
		key = key.Reverse()

		flow, exists = tracker.Flows[key]
		if !exists {
			tracker.Mu.Unlock()
			return
		}
	}

	/*
		CLOSE is observed from both client and server kernels.
		Process each flow once — otherwise active counts and
		rolling metrics double-count the same lifecycle.
	*/
	if flow.Closed {
		tracker.Mu.Unlock()
		return
	}

	// Mark runtime flow as closed.
	flow.Closed = true
	flow.CloseTime = time.Now()

	// Refresh liveness timestamp.
	//
	// Important for cleanup expiration logic.
	flow.LastUpdated = time.Now()

	// Derive behavioral semantics.
	//
	// Example:
	// - failed handshake
	// - short-lived flow
	InferFailure(flow)

	edgeKey := types.EdgeKey{
		Source:      flow.SourceService,
		Destination: flow.DestinationService,
	}
	flowSnapshot := *flow

	TrackClientIPClose(&flowSnapshot, cfg, unknown)

	tracker.Mu.Unlock()

	graph.Mu.Lock()

	edge, exists := graph.Edges[edgeKey]
	if exists {
		/*
			Decrement using canonical flow direction (client -> server),
			not the reversed CLOSE event's source/destination labels.
		*/
		if edge.ActiveConnections > 0 {
			edge.ActiveConnections--
		}

		UpdateEdgeMetrics(edge, &flowSnapshot)
	}

	graph.Mu.Unlock()
}

/*
PrintFlows renders live runtime flow state.
*/
func PrintFlows(
	tracker *types.FlowTracker,
) {

	tracker.Mu.RLock()

	snapshots := make([]types.ConnectionState, 0, len(tracker.Flows))
	for _, flow := range tracker.Flows {
		snapshots = append(snapshots, *flow)
	}
	tracker.Mu.RUnlock()

	log.Println("\n=== Runtime Flows ===")

	for _, flow := range snapshots {
		if flow.SourceService == "unknown" && flow.DestinationService == "external" {
			continue
		}

		log.Printf(
			"%s -> %s | accepted=%v | closed=%v | duration=%s",

			flow.SourceService,
			flow.DestinationService,

			flow.Accepted,
			flow.Closed,

			flow.Duration(),
		)
	}
}
