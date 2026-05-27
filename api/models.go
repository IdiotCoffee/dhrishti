package api

/*
GraphResponse is the serialized
frontend-safe graph representation.

IMPORTANT:

This is intentionally separate from
internal runtime graph structures.

Why?

Internal runtime structs contain:
- mutexes
- internal state
- implementation details

API consumers should ONLY receive:
stable semantic state.
*/
type GraphResponse struct {
	Nodes []NodeResponse `json:"nodes"`
	Edges []EdgeResponse `json:"edges"`
}

/*
NodeResponse represents a service node
in the runtime dependency graph.
*/
type NodeResponse struct {
	ID string `json:"id"`
}

/*
EdgeResponse represents a directional
service dependency relationship.
*/
type EdgeResponse struct {
	Source string `json:"source"`
	Target string `json:"target"`

	Port uint16 `json:"port"`

	ConnectionCount int `json:"connection_count"`

	ActiveConnections int `json:"active_connections"`

	FailedConnections int `json:"failed_connections"`

	ShortLivedConnections int `json:"short_lived_connections"`

	AverageDurationMs int64 `json:"average_duration_ms"`

	LastSeen string `json:"last_seen"`
}
