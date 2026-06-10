package api

import (
	"bytes"
	"net"
	"sort"
	"strings"

	"dhrishti/config"
	"dhrishti/types"
	"time"
)

/*
BuildGraphResponse converts internal runtime graph state
into frontend/API-safe JSON structures.

IMPORTANT:

This function creates a clean serialization boundary
between:

	internal runtime ownership
	            ↓
	external API consumers

This is a VERY important architectural separation.
*/
func BuildGraphResponse(
	graph *types.Graph,
	unknown *types.UnknownIPRegistry,
	cfg config.Config,
) GraphResponse {

	/*
		Read lock because we are only
		serializing graph state.

		No mutation occurs here.
	*/
	graph.Mu.RLock()
	defer graph.Mu.RUnlock()

	response := GraphResponse{
		Nodes:         []NodeResponse{},
		Edges:         []EdgeResponse{},
		UnknownIPs:    []UnknownIPResponse{},
		EntryServices: cfg.EntryServices,
	}

	/*
		Track unique nodes.

		Edges reference services,
		but frontend graph rendering
		also requires explicit node lists.
	*/
	nodeSet := make(map[string]bool)

	for _, edge := range graph.Edges {

		if !shouldShowEdge(edge.Source, edge.Destination) {
			continue
		}

		nodeSet[edge.Source] = true
		nodeSet[edge.Destination] = true

		/*
			Convert runtime edge
			into API-safe response model.
		*/
		response.Edges = append(
			response.Edges,
			EdgeResponse{
				Source: edge.Source,
				Target: edge.Destination,

				Port: edge.Port,

				ConnectionCount: edge.ConnectionCount,

				ActiveConnections: edge.ActiveConnections,

				FailedConnections: edge.FailedConnections,

				ShortLivedConnections: edge.ShortLivedConnections,

				/*
					Frontend/UI should NOT deal
					with Go time.Duration directly.

					Milliseconds are much easier
					for visualization systems.
				*/
				AverageDurationMs: edge.AverageDuration.Milliseconds(),
				/*
					Live operational metrics.

					These represent rolling
					recent behavior.
				*/
				RequestsPerSecond: edge.RequestsPerSecond,

				RecentAverageLatencyMs: edge.RecentAverageLatency.Milliseconds(),

				P95LatencyMs: edge.P95Latency.Milliseconds(),

				FailureRate: edge.FailureRate,

				/*
					RFC3339 is stable and frontend-safe.
				*/
				LastSeen: edge.LastSeen.Format(time.RFC3339),
			},
		)
	}

	/*
		Convert unique node set
		into API node list.
	*/
	for node := range nodeSet {

		response.Nodes = append(
			response.Nodes,
			NodeResponse{
				ID: node,
			},
		)
	}

	if unknown != nil {
		unknown.Mu.RLock()
		for _, entry := range unknown.Entries {
			dests := make(map[string]int, len(entry.Destinations))
			for k, v := range entry.Destinations {
				dests[k] = v
			}
			response.UnknownIPs = append(response.UnknownIPs, UnknownIPResponse{
				IP:                entry.IP,
				ConnectionCount:   entry.ConnectionCount,
				ActiveConnections: entry.ActiveConnections,
				Destinations:      dests,
				LastSeen:          entry.LastSeen.Format(time.RFC3339),
			})
		}
		unknown.Mu.RUnlock()

		sort.Slice(response.UnknownIPs, func(i, j int) bool {
			return compareClientIPs(response.UnknownIPs[i].IP, response.UnknownIPs[j].IP) < 0
		})
	}

	return response
}

func compareClientIPs(a, b string) int {
	ipA := net.ParseIP(a)
	ipB := net.ParseIP(b)
	if ipA != nil && ipB != nil {
		return bytes.Compare(ipA.To16(), ipB.To16())
	}
	return strings.Compare(a, b)
}
