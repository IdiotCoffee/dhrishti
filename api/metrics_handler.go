package api

import (
	"encoding/json"
	"net/http"

	"dhrishti/telemetry"
	"dhrishti/types"
)

type MetricsHandler struct {
	Counter *telemetry.Counter
	Graph   *types.Graph
}

type MetricsResponse struct {
	Runtime telemetry.Snapshot     `json:"runtime"`
	Edges   []EdgeLatencySummary   `json:"edges"`
}

type EdgeLatencySummary struct {
	Source           string  `json:"source"`
	Target           string  `json:"target"`
	P95LatencyMs     int64   `json:"p95_latency_ms"`
	RecentAverageMs  int64   `json:"recent_average_latency_ms"`
	RequestsPerSecond float64 `json:"requests_per_second"`
	FailureRate      float64 `json:"failure_rate"`
}

func (h *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := MetricsResponse{
		Runtime: h.Counter.Snapshot(),
		Edges:   make([]EdgeLatencySummary, 0),
	}

	h.Graph.Mu.RLock()
	for _, edge := range h.Graph.Edges {
		resp.Edges = append(resp.Edges, EdgeLatencySummary{
			Source:            edge.Source,
			Target:            edge.Destination,
			P95LatencyMs:      edge.P95Latency.Milliseconds(),
			RecentAverageMs:   edge.RecentAverageLatency.Milliseconds(),
			RequestsPerSecond: edge.RequestsPerSecond,
			FailureRate:       edge.FailureRate,
		})
	}
	h.Graph.Mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
