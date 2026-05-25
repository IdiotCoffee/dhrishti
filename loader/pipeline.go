package loader

import (
	"log"

	"dhrishti/resolver"

	"github.com/cilium/ebpf/ringbuf"
)

func StartTCPConnectPipeline(
	rd *ringbuf.Reader,

	dockerResolver *resolver.DockerResolver,
	ipMap map[string]string,

	handler func(*resolver.EnrichedRuntimeEvent),
) {
	go ReadTCPConnectEvents(
		rd,
		func(raw TCPConnectEvent) {

			event := NormalizeTCPConnectEvent(raw)

			enriched, err := dockerResolver.EnrichEvent(
				event,
				ipMap,
			)

			if err != nil {
				log.Printf(
					"tcp_connect enrichment error: %v",
					err,
				)
				return
			}

			handler(enriched)
		},
	)
}

func StartTCPClosePipeline(
	rd *ringbuf.Reader,

	dockerResolver *resolver.DockerResolver,
	ipMap map[string]string,

	handler func(*resolver.EnrichedRuntimeEvent),
) {
	go ReadTCPCloseEvents(
		rd,
		func(raw TCPCloseEvent) {

			event := NormalizeTCPCloseEvent(raw)

			enriched, err := dockerResolver.EnrichEvent(
				event,
				ipMap,
			)

			if err != nil {
				log.Printf(
					"tcp_close enrichment error: %v",
					err,
				)
				return
			}

			handler(enriched)
		},
	)
}

func StartTCPAcceptPipeline(
	rd *ringbuf.Reader,

	dockerResolver *resolver.DockerResolver,
	ipMap map[string]string,

	handler func(*resolver.EnrichedRuntimeEvent),
) {
	go ReadTCPAcceptEvents(
		rd,
		func(raw TCPAcceptEvent) {

			event := NormalizeTCPAcceptEvent(raw)

			enriched, err := dockerResolver.EnrichEvent(
				event,
				ipMap,
			)

			if err != nil {
				log.Printf(
					"tcp_accept enrichment error: %v",
					err,
				)
				return
			}

			handler(enriched)
		},
	)
}

func PrintEnrichedRuntimeEvent(
	event *resolver.EnrichedRuntimeEvent,
) {
	log.Printf(
		"[%s] %s -> %s (%s:%d -> %s:%d)",
		event.Type,

		event.SourceService,
		event.DestinationService,

		event.SourceIP,
		event.SourcePort,

		event.DestinationIP,
		event.DestinationPort,
	)
}
