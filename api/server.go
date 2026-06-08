package api

import (
	"log"
	"net/http"
	"time"

	"dhrishti/config"
	"dhrishti/telemetry"
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
	unknown *types.UnknownIPRegistry,
	counter *telemetry.Counter,
	cfg config.Config,
) {
	graphHandler := &GraphHandler{
		Graph:   graph,
		Unknown: unknown,
		Config:  cfg,
	}

	wsGraphHandler := &WSGraphHandler{
		Graph:    graph,
		Unknown:  unknown,
		Config:   cfg,
		Interval: 100 * time.Millisecond,
	}

	/*
		Register HTTP routes.
	*/
	metricsHandler := &MetricsHandler{
		Counter: counter,
		Graph:   graph,
	}

	mux := http.NewServeMux()
	mux.Handle("/graph", graphHandler)
	mux.Handle("/ws", wsGraphHandler)
	mux.Handle("/metrics", metricsHandler)

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
