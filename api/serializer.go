package api

import (
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
) GraphResponse {

	/*
		Read lock because we are only
		serializing graph state.

		No mutation occurs here.
	*/
	graph.Mu.RLock()
	defer graph.Mu.RUnlock()

	response := GraphResponse{
		Nodes: []NodeResponse{},
		Edges: []EdgeResponse{},
	}

	/*
		Track unique nodes.

		Edges reference services,
		but frontend graph rendering
		also requires explicit node lists.
	*/
	nodeSet := make(map[string]bool)

	for _, edge := range graph.Edges {

		/*
			Register source node.
		*/
		nodeSet[edge.Source] = true

		/*
			Register destination node.
		*/
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

	return response
}
