package loader

import (
	"log"

	"dhrishti/resolver"

	"github.com/cilium/ebpf/ringbuf"
)

/*
StartTCPConnectPipeline handles:

	kernel tcp_connect events
		↓
	runtime normalization
		↓
	live identity enrichment
		↓
	semantic graph updates

VERY IMPORTANT:

Identity enrichment is now:
dynamic runtime state.

NOT startup-time snapshot state.
*/
func StartTCPConnectPipeline(
	rd *ringbuf.Reader,

	dockerResolver *resolver.DockerResolver,

	handler func(*resolver.EnrichedRuntimeEvent),
) {

	go ReadTCPConnectEvents(
		rd,

		func(raw TCPConnectEvent) {

			/*
				Normalize raw kernel telemetry
				into stable runtime event structure.
			*/
			event :=
				NormalizeTCPConnectEvent(raw)

			/*
				Enrich runtime event using:
				live runtime identity cache.

				This allows topology evolution
				as containers:
				- restart
				- appear
				- disappear
			*/
			enriched, err :=
				dockerResolver.EnrichEvent(
					event,
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

/*
StartTCPClosePipeline handles:

	connection termination lifecycle.

This enables:
- duration computation
- churn detection
- lifecycle reconstruction
- rolling observability metrics
*/
func StartTCPClosePipeline(
	rd *ringbuf.Reader,

	dockerResolver *resolver.DockerResolver,

	handler func(*resolver.EnrichedRuntimeEvent),
) {

	go ReadTCPCloseEvents(
		rd,

		func(raw TCPCloseEvent) {

			event :=
				NormalizeTCPCloseEvent(raw)

			/*
				Runtime identity is now:
				dynamically reconciled.
			*/
			enriched, err :=
				dockerResolver.EnrichEvent(
					event,
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

/*
StartTCPAcceptPipeline handles:

	server-side socket acceptance.

VERY IMPORTANT:

CONNECT and ACCEPT represent:
different runtime perspectives.

This pipeline enables:
client/server lifecycle reconciliation.
*/
func StartTCPAcceptPipeline(
	rd *ringbuf.Reader,

	dockerResolver *resolver.DockerResolver,

	handler func(*resolver.EnrichedRuntimeEvent),
) {

	go ReadTCPAcceptEvents(
		rd,

		func(raw TCPAcceptEvent) {

			event :=
				NormalizeTCPAcceptEvent(raw)

			/*
				Resolve service identity from:
				live runtime cache.
			*/
			enriched, err :=
				dockerResolver.EnrichEvent(
					event,
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

/*
PrintEnrichedRuntimeEvent renders:

	raw kernel telemetry
		↓
	semantic runtime relationship
*/
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
