package api

import (
	"log"
	"net/http"
	"time"

	"dhrishti/types"
)

/*
StartServer bootstraps the observability API layer.

This exposes runtime graph state externally.

Current endpoints:

	GET /graph

Future endpoints may include:

	/flows
	/metrics
	/ws
*/
func StartServer(
	graph *types.Graph,
) {

	/*
		Create graph handler.

		Handler owns ONLY HTTP transport logic.
	*/
	graphHandler := &GraphHandler{
		Graph: graph,
	}

	wsGraphHandler := &WSGraphHandler{
		Graph:    graph,
		Interval: 100 * time.Millisecond,
	}

	/*
		Register HTTP routes.
	*/
	mux := http.NewServeMux()
	mux.Handle("/graph", graphHandler)
	mux.Handle("/ws", wsGraphHandler)

	log.Println(
		"observability API listening on :8090",
	)

	/*
		Start HTTP server.

		This blocks forever,
		so we usually launch this
		inside a goroutine.
	*/
	err := http.ListenAndServe(
		":8090",
		mux,
	)

	if err != nil {
		log.Fatalf(
			"starting API server: %v",
			err,
		)
	}
}
