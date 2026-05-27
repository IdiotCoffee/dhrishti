package api

import (
	"log"
	"net/http"

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

	/*
		Register HTTP routes.
	*/
	http.Handle(
		"/graph",
		graphHandler,
	)

	log.Println(
		"observability API listening on :8080",
	)

	/*
		Start HTTP server.

		This blocks forever,
		so we usually launch this
		inside a goroutine.
	*/
	err := http.ListenAndServe(
		":8090",
		nil,
	)

	if err != nil {
		log.Fatalf(
			"starting API server: %v",
			err,
		)
	}
}
