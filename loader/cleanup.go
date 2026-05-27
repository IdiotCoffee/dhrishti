package loader

import (
	"log"
	"time"

	"dhrishti/types"
)

// StartFlowCleanupLoop periodically removes expired flows.
//
// Why this matters:
//
// Without cleanup:
// - memory usage grows forever
// - stale runtime state accumulates
// - active flow counts become incorrect
//
// Real observability systems MUST continuously manage lifecycle state.
func StartFlowCleanupLoop(
	tracker *types.FlowTracker,
	expiration time.Duration,
) {
	ticker := time.NewTicker(15 * time.Second)

	go func() {
		for range ticker.C {
			cleanupExpiredFlows(tracker, expiration)
		}
	}()
}

// cleanupExpiredFlows removes stale closed flows.
func cleanupExpiredFlows(
	tracker *types.FlowTracker,
	expiration time.Duration,
) {
	tracker.Mu.Lock()
	defer tracker.Mu.Unlock()

	now := time.Now()

	removed := 0

	for key, flow := range tracker.Flows {
		// We ONLY cleanup flows that:
		//
		// 1. are already closed
		// 2. have been inactive long enough
		if flow.Closed &&
			now.Sub(flow.LastUpdated) > expiration {

			delete(tracker.Flows, key)
			removed++
		}
	}

	if removed > 0 {
		log.Printf(
			"[cleanup] removed %d expired flows",
			removed,
		)
	}
}
