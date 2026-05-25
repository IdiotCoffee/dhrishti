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

	// dstService := ipMap[event.DestinationIP]
	dstService, exists := ipMap[event.DestinationIP]

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
