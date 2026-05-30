package resolver

import (
	"dhrishti/types"
)

type EnrichedRuntimeEvent struct {
	types.RuntimeEvent

	SourceContainerID string
	SourceService     string

	DestinationService string
}

func (d *DockerResolver) EnrichEvent(
	event types.RuntimeEvent,
) (*EnrichedRuntimeEvent, error) {

	// containerID, err := ResolveContainerID(event.PID)
	// if err != nil {
	// 	return nil, err
	// }
	containerID := "unknown"

	resolvedContainerID, err := ResolveContainerID(event.PID)
	if err == nil {
		containerID = resolvedContainerID
	}

	// sourceService, err := d.ResolveServiceName(containerID)
	// if err != nil {
	// 	return nil, err
	// }
	sourceService := "unknown"

	if containerID != "unknown" {

		resolvedService, err := d.ResolveServiceName(containerID)

		if err == nil {
			sourceService = resolvedService
		}
	}

	// PID->container lookup can fail on some cgroup layouts; use source IP as a
	// fallback identity hint. Do not overwrite with "external" because that
	// hides unresolved telemetry as false certainty.
	if sourceService == "unknown" {
		resolvedSource := d.ResolveIP(event.SourceIP)
		if resolvedSource != "external" {
			sourceService = resolvedSource
		}
	}

	lookupIP := event.DestinationIP

	// ACCEPT events are observed from the
	// server kernel perspective.
	//
	// That means:
	//
	// source = server
	// destination = client
	//
	// But semantically we want:
	// client -> server
	//
	// So for ACCEPT events,
	// the service identity lives on SourceIP.
	if event.Type == types.EventAccept {
		lookupIP = event.SourceIP
	}

	/*
		Resolve destination identity
		from live runtime cache.

		IMPORTANT:

		Identity resolution is now:
		dynamic runtime state.

		NOT static startup metadata.
	*/
	dstService := d.ResolveIP(lookupIP)

	// if !exists {
	// 	dstService = "external"
	// }

	return &EnrichedRuntimeEvent{
		RuntimeEvent: event,

		SourceContainerID:  containerID,
		SourceService:      sourceService,
		DestinationService: dstService,
	}, nil
}
