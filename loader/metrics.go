package loader

import (
	"time"

	"dhrishti/types"
)

// InferFailure derives behavioral meaning from a flow lifecycle.
//
// This is the beginning of semantic observability logic.
//
// Example:
//
// connect=true
// accept=false
// closed=true
//
// may indicate:
// - connection refusal
// - timeout
// - failed handshake
func InferFailure(flow *types.ConnectionState) {
	// If the connection closed without ever being accepted,
	// we consider it suspicious/failed.
	if flow.Closed && !flow.Accepted {
		flow.Failed = true
		flow.FailureReason = "closed_without_accept"
	}

	// Extremely short-lived connections are often signals of:
	// - retries
	// - refused connections
	// - unstable services
	if flow.Duration() < 10*time.Millisecond {
		flow.ShortLived = true
	}
}

// UpdateEdgeMetrics aggregates completed flow behavior into graph edges.
func UpdateEdgeMetrics(edge *types.Edge, flow *types.ConnectionState) {
	edge.LastSeen = time.Now()

	// Aggregate total observed duration.
	edge.TotalDuration += flow.Duration()

	// Recompute rolling average duration.
	// Only completed flows contribute
	// to duration metrics.
	edge.CompletedConnections++

	if edge.CompletedConnections > 0 {
		edge.AverageDuration =
			edge.TotalDuration /
				time.Duration(edge.CompletedConnections)
	}

	// Aggregate failed flow count.
	if flow.Failed {
		edge.FailedConnections++
	}

	// Aggregate short-lived flow count.
	if flow.ShortLived {
		edge.ShortLivedConnections++
	}
}

/*
InferStaleFlows scans active runtime flows and derives
failure semantics for incomplete lifecycle state.

Why this matters:

Real telemetry is imperfect.

Sometimes:
- ACCEPT events never arrive
- CLOSE events are missed
- connections timeout silently

Observability systems must infer:
probable runtime truth.

This is the beginning of:
semantic timeout inference.
*/
func InferStaleFlows(
	tracker *types.FlowTracker,
	graph *types.Graph,
	timeout time.Duration,
) {

	tracker.Mu.Lock()
	defer tracker.Mu.Unlock()

	now := time.Now()

	for _, flow := range tracker.Flows {

		/*
			Ignore flows already finalized.
		*/
		if flow.Closed {
			continue
		}

		/*
			How long has this flow existed?
		*/
		age := now.Sub(flow.ConnectTime)

		/*
			If a flow remains incomplete
			for too long:

				connect observed
				but no accept/close

			then infer probable timeout/failure.
		*/
		if !flow.Accepted && age > timeout {

			flow.Failed = true

			flow.FailureReason =
				"stale_incomplete_flow"

			/*
				Mark as closed semantically.

				Even though kernel close
				was never observed.
			*/
			flow.Closed = true

			flow.CloseTime = now

			flow.LastUpdated = now

			/*
				Update graph edge state because
				we are semantically finalizing
				this lifecycle.
			*/
			edgeKey := types.EdgeKey{
				Source:      flow.SourceService,
				Destination: flow.DestinationService,
			}

			graph.Mu.Lock()

			edge, exists := graph.Edges[edgeKey]
			if exists {

				/*
					This flow is now considered
					semantically closed.

					So decrement active count.
				*/
				if edge.ActiveConnections > 0 {
					edge.ActiveConnections--
				}

				/*
					Aggregate behavioral failure metrics.
				*/
				UpdateEdgeMetrics(edge, flow)
			}

			graph.Mu.Unlock()
		}
	}
}
