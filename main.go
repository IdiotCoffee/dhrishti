package main

import (
	"dhrishti/loader"
	"dhrishti/resolver"
	"dhrishti/types"
	"log"
	"time"
)

func main() {

	log.Println("starting dhrishti telemetry engine")

	// initialize docker metadata resolver
	dockerResolver, err := resolver.NewDockerResolver()
	if err != nil {
		log.Fatalf(
			"creating docker resolver: %v",
			err,
		)
	}

	// build IP -> service mapping
	ipMap, err := dockerResolver.BuildIPServiceMap()
	if err != nil {
		log.Fatalf(
			"building IP service map: %v",
			err,
		)
	}
	graph := types.NewGraph()
	go func() {

		for {

			graph.Print()

			time.Sleep(5 * time.Second)
		}
	}()

	// attach tcp_connect probe
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

	// attach tcp_close probe
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

	// attach tcp_accept probe
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

	// start tcp_connect event pipeline
	loader.StartTCPConnectPipeline(
		connectProbe.Reader,

		dockerResolver,
		ipMap,

		func(event *resolver.EnrichedRuntimeEvent) {

			// print event
			loader.PrintEnrichedRuntimeEvent(event)

			// mutate graph state
			graph.RecordConnect(
				event.SourceService,
				event.DestinationService,
				event.DestinationPort,
			)
		},
	)

	// start tcp_close event pipeline
	loader.StartTCPClosePipeline(
		closeProbe.Reader,

		dockerResolver,
		ipMap,

		func(event *resolver.EnrichedRuntimeEvent) {

			// print event
			loader.PrintEnrichedRuntimeEvent(event)

			// update live connection state
			graph.RecordClose(
				event.SourceService,
				event.DestinationService,
			)
		},
	)

	// start tcp_accept event pipeline
	loader.StartTCPAcceptPipeline(
		acceptProbe.Reader,

		dockerResolver,
		ipMap,

		func(event *resolver.EnrichedRuntimeEvent) {

			// currently just print accept lifecycle events
			loader.PrintEnrichedRuntimeEvent(event)
		},
	)

	log.Println("telemetry pipelines running")

	// keep process alive forever
	select {}
}
