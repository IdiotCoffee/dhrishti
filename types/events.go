package types

import "time"

type EventType string

const (
	EventConnect EventType = "CONNECT"
	EventClose   EventType = "CLOSE"
	EventAccept  EventType = "ACCEPT"
	EventState   EventType = "STATE"
)

type RuntimeEvent struct {
	Type EventType

	Timestamp time.Time

	SourceService      string
	DestinationService string

	SourceIP      string
	DestinationIP string

	SourcePort      uint16
	DestinationPort uint16

	PID uint32

	Metadata map[string]string

	EventType string
}
