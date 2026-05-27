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
	ipMap map[string]string,
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
	if event.EventType == "accept" {
		lookupIP = event.SourceIP
	}

	dstService, exists := ipMap[lookupIP]

	if !exists {
		dstService = "external"
	}

	return &EnrichedRuntimeEvent{
		RuntimeEvent: event,

		SourceContainerID:  containerID,
		SourceService:      sourceService,
		DestinationService: dstService,
	}, nil
}
