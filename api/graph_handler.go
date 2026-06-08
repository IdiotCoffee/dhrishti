package api

import (
	"encoding/json"
	"net/http"

	"dhrishti/config"
	"dhrishti/types"
)

/*
GraphHandler exposes live runtime graph state
through HTTP.

This becomes the FIRST external platform boundary.

Clients can now consume:
- topology
- edge metrics
- behavioral state

without direct access to runtime internals.
*/
type GraphHandler struct {
	Graph   *types.Graph
	Unknown *types.UnknownIPRegistry
	Config  config.Config
}

/*
ServeHTTP handles:

	GET /graph

and returns serialized graph state as JSON.
*/
func (h *GraphHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {

	/*
		Only allow GET requests.

		This endpoint is read-only.
	*/
	if r.Method != http.MethodGet {

		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)

		return
	}
	/*
		Allow frontend dev server
		to access observability API.

		Frontend runs on:
		localhost:5173

		Go API runs on:
		localhost:8090

		Browsers enforce same-origin policy,
		so we must explicitly allow cross-origin requests.
	*/
	w.Header().Set(
		"Access-Control-Allow-Origin",
		"*",
	)

	/*
		Convert runtime graph
		into API-safe response structure.
	*/
	response := BuildGraphResponse(h.Graph, h.Unknown, h.Config)

	/*
		Response content type matters heavily
		for frontend/API interoperability.
	*/
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	/*
		Serialize graph response.

		Encoder streams directly into response writer.
	*/
	json.NewEncoder(w).Encode(response)
}
