package loader

import (
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

	return types.FlowKey{
		SourceIP:   event.SourceIP,
		SourcePort: event.SourcePort,

		DestinationIP:   event.DestinationIP,
		DestinationPort: event.DestinationPort,
	}
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

	key := BuildFlowKey(event).Reverse()

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
	defer tracker.Mu.Unlock()

	flow, exists := tracker.Flows[key]

	// If direct lookup fails,
	// try reversed flow identity.
	if !exists {
		key = key.Reverse()

		flow, exists = tracker.Flows[key]
		if !exists {
			return
		}
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

	// Lookup graph edge representing:
	//
	// source_service -> destination_service
	edgeKey := types.EdgeKey{
		Source:      flow.SourceService,
		Destination: flow.DestinationService,
	}

	// IMPORTANT:
	// graph state is a separate ownership domain.
	graph.Mu.Lock()

	edge, exists := graph.Edges[edgeKey]
	if exists {
		// Aggregate behavioral metrics
		// into dependency edge state.
		UpdateEdgeMetrics(edge, flow)
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
	defer tracker.Mu.RUnlock()

	log.Println("\n=== Runtime Flows ===")

	for _, flow := range tracker.Flows {

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
