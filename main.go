package main

import (
	"log"
	"runtime"
	"time"

	"dhrishti/api"
	"dhrishti/config"
	"dhrishti/history"
	"dhrishti/loader"
	"dhrishti/resolver"
	"dhrishti/telemetry"
	"dhrishti/types"
)

func main() {

	log.Println("starting dhrishti telemetry engine")

	cfg := config.Load()

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
		Continuously refresh runtime identity.

		This allows:
		- dynamic container discovery
		- runtime topology evolution
		- container restart reconciliation

		VERY IMPORTANT:

		Distributed systems are dynamic runtime environments.

		Identity must evolve continuously.
	*/
	dockerResolver.StartRefreshLoop(
		5 * time.Second,
	)

	/*
		Live topology graph.

		This stores:
		service dependency structure.
	*/
	graph := types.NewGraph()
	unknownIPs := types.NewUnknownIPRegistry()
	counter := telemetry.NewCounter()
	/*
		Start observability API server.

		This exposes runtime graph state externally.

		Current endpoint:

			GET /graph

		Future:
		- /flows
		- /metrics
		- /ws
	*/
	go api.StartServer(graph, unknownIPs, counter, cfg)

	historyCfg := history.LoadConfig()
	history.StartWriter(graph, unknownIPs, cfg, historyCfg)
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
				15*time.Second,
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

		func(event *resolver.EnrichedRuntimeEvent) {
			loader.ApplyEntryPointRules(event, cfg, unknownIPs)
			counter.IncConnect()
			loader.PrintEnrichedRuntimeEvent(event)
			loader.RecordNewConnection(
				tracker,
				graph,
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
		func(event *resolver.EnrichedRuntimeEvent) {
			counter.IncClose()
			loader.PrintEnrichedRuntimeEvent(event)
			loader.HandleClose(
				tracker,
				graph,
				cfg,
				unknownIPs,
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

		func(event *resolver.EnrichedRuntimeEvent) {
			counter.IncAccept()
			loader.PrintEnrichedRuntimeEvent(event)
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

	/*
		Pin probes for the lifetime of the process.

		The pipeline goroutines only hold a reference to Probe.Reader, not to the
		Probe struct itself. After the StartTCP*Pipeline calls above, the local
		variables connectProbe/closeProbe/acceptProbe may be marked dead by the
		compiler's liveness maps (no subsequent instruction in main reads them).

		When the GC traces main's stack at the select{} safe-point and sees those
		variables are dead, it can collect the Probe structs. Collecting a Probe
		lets its embedded link.Link finalizer fire, which silently detaches the
		eBPF kprobe from the kernel. rd.Read() then blocks forever waiting for
		events that will never arrive — no error, no crash, just silence.

		Fix: pass the probes as ARGUMENTS to a goroutine that runs forever.
		Function arguments are evaluated at the call site (while main's frame is
		still fully live), copied into the goroutine's own stack frame, and remain
		reachable GC roots for as long as that goroutine exists.
	*/
	go func(probes ...*loader.Probe) {
		for {
			time.Sleep(time.Hour)
			runtime.KeepAlive(probes)
		}
	}(connectProbe, closeProbe, acceptProbe)

	select {}
}
