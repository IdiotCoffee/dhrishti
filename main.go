package main

import (
	"log"
	"time"

	"dhrishti/loader"
	"dhrishti/resolver"
	"dhrishti/types"
)

func main() {

	log.Println("starting dhrishti telemetry engine")

	/*
		Initialize Docker metadata resolver.

		This layer translates:
		PID/IP/port
			↓
		container/service identity

		Without this:
		we only have raw kernel telemetry.
	*/
	dockerResolver, err := resolver.NewDockerResolver()
	if err != nil {
		log.Fatalf(
			"creating docker resolver: %v",
			err,
		)
	}

	/*
		Build runtime IP -> service mapping.

		Example:
			172.19.0.4 -> payment-service
	*/
	ipMap, err := dockerResolver.BuildIPServiceMap()
	if err != nil {
		log.Fatalf(
			"building IP service map: %v",
			err,
		)
	}

	/*
		Live topology graph.

		This stores:
		service dependency structure.
	*/
	graph := types.NewGraph()

	/*
		Runtime flow tracker.

		This stores:
		live TCP lifecycle state.

		This is the BIG architectural leap:
			event stream
				↓
			state reconstruction
	*/
	tracker := types.NewFlowTracker()

	/*
		Periodic graph printer.

		This visualizes:
		service-to-service topology.
	*/
	// Start background cleanup subsystem.
	//
	// Closed flows older than 30 seconds
	// will be removed automatically.
	loader.StartFlowCleanupLoop(
		tracker,
		30*time.Second,
	)
	go func() {

		for {

			graph.Print()

			time.Sleep(5 * time.Second)
		}
	}()

	/*
		Background semantic inference loop.

		This periodically scans runtime flows
		for incomplete lifecycle state.

		Example:

				connect observed
				but no accept/close ever arrives

		This often indicates:
		- timeout
		- dropped telemetry
		- failed handshake
		- partial lifecycle observation
	*/
	go func() {

		for {

			loader.InferStaleFlows(
				tracker,
				graph,
				5*time.Second,
			)

			time.Sleep(5 * time.Second)
		}
	}()

	/*
		Periodic runtime flow printer.

		This visualizes:
		connection lifecycle state.
	*/
	go func() {

		for {

			loader.PrintFlows(tracker)

			time.Sleep(5 * time.Second)
		}
	}()

	/*
		Attach tcp_connect probe.

		Client-side outbound connection intent.
	*/
	connectProbe, err := loader.AttachKprobe(
		"ebpf/tcp_connect.bpf.o",
		"trace_tcp_connect",
		"tcp_connect",
	)

	if err != nil {
		log.Fatalf(
			"attach tcp_connect: %v",
			err,
		)
	}

	/*
		Attach tcp_close probe.

		Connection termination lifecycle.
	*/
	closeProbe, err := loader.AttachKprobe(
		"ebpf/tcp_close.bpf.o",
		"trace_tcp_close",
		"tcp_close",
	)

	if err != nil {
		log.Fatalf(
			"attach tcp_close: %v",
			err,
		)
	}

	/*
		Attach tcp_accept probe.

		Server-side inbound acceptance.

		Very important:
		this gives server visibility.
	*/
	acceptProbe, err := loader.AttachKretprobe(
		"ebpf/tcp_accept.bpf.o",
		"trace_inet_csk_accept",
		"inet_csk_accept",
	)

	if err != nil {
		log.Fatalf(
			"attach tcp_accept: %v",
			err,
		)
	}
	/*
		START CONNECT PIPELINE

		Runtime meaning:
			client initiated connection
	*/
	loader.StartTCPConnectPipeline(
		connectProbe.Reader,

		dockerResolver,
		ipMap,

		func(event *resolver.EnrichedRuntimeEvent) {

			/*
				Print enriched semantic event.
			*/
			loader.PrintEnrichedRuntimeEvent(event)

			/*
				Update topology graph state.

				This tracks:
				service dependency emergence.
			*/
			graph.RecordConnect(
				event.SourceService,
				event.DestinationService,
				event.DestinationPort,
			)

			/*
				Create runtime flow state.

				This begins:
				connection lifecycle tracking.
			*/
			loader.HandleConnect(
				tracker,
				event,
			)
		},
	)

	/*
		START CLOSE PIPELINE

		Runtime meaning:
			connection terminated
	*/
	loader.StartTCPClosePipeline(
		closeProbe.Reader,

		dockerResolver,
		ipMap,

		func(event *resolver.EnrichedRuntimeEvent) {

			/*
				Print enriched lifecycle event.
			*/
			loader.PrintEnrichedRuntimeEvent(event)

			/*
				Update graph active connection state.
			*/
			graph.RecordClose(
				event.SourceService,
				event.DestinationService,
			)

			/*
				Finalize runtime flow.

				Computes:
				- duration
				- closed state
			*/
			loader.HandleClose(
				tracker,
				graph,
				event,
			)
		},
	)

	/*
		START ACCEPT PIPELINE

		Runtime meaning:
			server accepted connection
	*/
	loader.StartTCPAcceptPipeline(
		acceptProbe.Reader,

		dockerResolver,
		ipMap,

		func(event *resolver.EnrichedRuntimeEvent) {

			/*
				Print server-side lifecycle event.
			*/
			loader.PrintEnrichedRuntimeEvent(event)

			/*
				Mark runtime flow as accepted.

				Very important:
				CONNECT and ACCEPT are observed
				from opposite perspectives.

				The correlation engine handles:
				client/server identity reconciliation.
			*/
			loader.HandleAccept(
				tracker,
				event,
			)
		},
	)

	log.Println("telemetry pipelines running")

	/*
		Keep runtime alive forever.

		All telemetry processing now happens
		inside concurrent goroutines.
	*/
	select {}
}
