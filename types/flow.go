package types

/*
FlowKey uniquely identifies one TCP flow.

This becomes:
the canonical runtime connection identity.

Very important:
multiple probes must agree on this identity.

The same:
CONNECT
ACCEPT
CLOSE

must map to:
ONE shared flow state.
*/
type FlowKey struct {

	// client endpoint
	SourceIP   string
	SourcePort uint16

	// server endpoint
	DestinationIP   string
	DestinationPort uint16
}

/*
Reverse returns the opposite flow direction.

Why needed?

Because:
ACCEPT events are observed
from server perspective.

Meaning:

CONNECT:

	client -> server

ACCEPT:

	server -> client

But both belong to:
the SAME logical connection.
*/
func (f FlowKey) Reverse() FlowKey {
	return FlowKey{
		SourceIP:   f.DestinationIP,
		SourcePort: f.DestinationPort,

		DestinationIP:   f.SourceIP,
		DestinationPort: f.SourcePort,
	}
}
