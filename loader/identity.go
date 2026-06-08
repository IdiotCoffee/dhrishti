package loader

import (
	"dhrishti/config"
	"dhrishti/resolver"
	"dhrishti/types"
)

// ApplyEntryPointRules promotes unresolved external traffic hitting a configured
// entry service to "client", while keeping the source IP in the registry.
func ApplyEntryPointRules(
	event *resolver.EnrichedRuntimeEvent,
	cfg config.Config,
	unknown *types.UnknownIPRegistry,
) {
	if !cfg.IsEntryService(event.DestinationService) {
		return
	}

	if event.SourceService != "unknown" && event.SourceService != "external" {
		return
	}

	if !types.IsLoopbackIP(event.SourceIP) {
		unknown.RecordConnect(event.SourceIP, event.DestinationService)
	}

	event.SourceService = "client"
}

func TrackClientIPClose(
	flow *types.ConnectionState,
	cfg config.Config,
	unknown *types.UnknownIPRegistry,
) {
	if flow.SourceService != "client" {
		return
	}
	if !cfg.IsEntryService(flow.DestinationService) {
		return
	}
	unknown.RecordClose(flow.Key.SourceIP, flow.DestinationService)
}
