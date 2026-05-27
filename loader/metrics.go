package loader

import (
	"sort"
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

/*
UpdateEdgeMetrics aggregates completed
flow behavior into dependency edge state.

This function is now becoming:
a temporal observability engine.

VERY IMPORTANT:

We maintain BOTH:
- cumulative historical metrics
- rolling operational metrics

These solve different observability problems.
*/
func UpdateEdgeMetrics(
	edge *types.Edge,
	flow *types.ConnectionState,
) {

	now := time.Now()

	edge.LastSeen = now

	/*
		Aggregate cumulative lifetime metrics.
	*/

	edge.TotalDuration += flow.Duration()

	edge.CompletedConnections++

	if edge.CompletedConnections > 0 {

		edge.AverageDuration =
			edge.TotalDuration /
				time.Duration(edge.CompletedConnections)
	}

	if flow.Failed {
		edge.FailedConnections++
	}

	if flow.ShortLived {
		edge.ShortLivedConnections++
	}

	/*
		Convert completed flow into
		a rolling temporal sample.
	*/
	sample := types.FlowSample{
		Timestamp: now,
		Duration:  flow.Duration(),
		Failed:    flow.Failed,
	}

	edge.RecentFlows =
		append(edge.RecentFlows, sample)

	/*
		Trim rolling window.

		We intentionally keep ONLY
		recent operational behavior.

		This prevents:
		- unbounded memory growth
		- stale observability state
		- historical dilution

		Current window:
			last 30 seconds
	*/
	windowStart :=
		now.Add(-30 * time.Second)

	filtered :=
		make([]types.FlowSample, 0)

	for _, sample := range edge.RecentFlows {

		if sample.Timestamp.After(windowStart) {

			filtered =
				append(filtered, sample)
		}
	}

	edge.RecentFlows = filtered

	/*
		Derive live operational metrics.

		These metrics now answer:

			"What is happening NOW?"

		rather than:

			"What happened historically?"
	*/

	recentCount :=
		len(edge.RecentFlows)

	if recentCount == 0 {
		return
	}

	/*
		Requests/sec over rolling window.
	*/
	edge.RequestsPerSecond =
		float64(recentCount) / 30.0

	/*
		Rolling latency metrics.
	*/
	var totalLatency time.Duration

	failedCount := 0

	latencies :=
		make([]time.Duration, 0)

	for _, sample := range edge.RecentFlows {

		totalLatency += sample.Duration

		latencies =
			append(latencies, sample.Duration)

		if sample.Failed {
			failedCount++
		}
	}

	/*
		Recent rolling average latency.
	*/
	edge.RecentAverageLatency =
		totalLatency / time.Duration(recentCount)

	/*
		Rolling failure rate.
	*/
	edge.FailureRate =
		float64(failedCount) /
			float64(recentCount)

	/*
		Compute p95 latency.

		P95 reveals:
		tail latency behavior.

		Extremely important in:
		distributed systems observability.
	*/
	sort.Slice(
		latencies,

		func(i, j int) bool {
			return latencies[i] < latencies[j]
		},
	)

	p95Index :=
		int(float64(len(latencies)-1) * 0.95)

	edge.P95Latency =
		latencies[p95Index]
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
