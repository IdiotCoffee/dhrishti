package loader

import (
	"log"
	"time"

	"dhrishti/resolver"
	"dhrishti/types"
)

/*
BuildFlowKey converts runtime events
into canonical flow identity.
*/
func BuildFlowKey(
	event *resolver.EnrichedRuntimeEvent,
) types.FlowKey {

	return types.FlowKey{
		SourceIP:   event.SourceIP,
		SourcePort: event.SourcePort,

		DestinationIP:   event.DestinationIP,
		DestinationPort: event.DestinationPort,
	}
}

/*
HandleConnect creates runtime flow state.

This represents:
client-side connection initiation.
*/
func HandleConnect(
	tracker *types.FlowTracker,
	event *resolver.EnrichedRuntimeEvent,
) {

	key := BuildFlowKey(event)

	tracker.Mu.Lock()
	defer tracker.Mu.Unlock()

	tracker.Flows[key] = &types.ConnectionState{
		Key: key,

		SourceService:      event.SourceService,
		DestinationService: event.DestinationService,

		ConnectTime: time.Now(),
	}
}

/*
HandleAccept marks server-side acceptance.

IMPORTANT:
accept events arrive reversed.

So:
we reverse flow identity first.
*/
func HandleAccept(
	tracker *types.FlowTracker,
	event *resolver.EnrichedRuntimeEvent,
) {

	key := BuildFlowKey(event).Reverse()

	tracker.Mu.Lock()
	defer tracker.Mu.Unlock()

	flow, exists := tracker.Flows[key]
	if !exists {
		return
	}

	flow.Accepted = true
	flow.AcceptTime = time.Now()
}

/*
HandleClose finalizes runtime connection state.

Computes:
- duration
- closure semantics
*/
func HandleClose(
	tracker *types.FlowTracker,
	event *resolver.EnrichedRuntimeEvent,
) {

	key := BuildFlowKey(event)

	tracker.Mu.Lock()
	defer tracker.Mu.Unlock()

	flow, exists := tracker.Flows[key]
	if !exists {
		return
	}

	flow.Closed = true
	flow.CloseTime = time.Now()

	flow.Duration =
		flow.CloseTime.Sub(flow.ConnectTime)
}

/*
PrintFlows renders live runtime flow state.
*/
func PrintFlows(
	tracker *types.FlowTracker,
) {

	tracker.Mu.RLock()
	defer tracker.Mu.RUnlock()

	log.Println("\n=== Runtime Flows ===")

	for _, flow := range tracker.Flows {

		log.Printf(
			"%s -> %s | accepted=%v | closed=%v | duration=%s",

			flow.SourceService,
			flow.DestinationService,

			flow.Accepted,
			flow.Closed,

			flow.Duration,
		)
	}
}
