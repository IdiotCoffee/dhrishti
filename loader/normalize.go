package loader

import (
	"net"
	"time"

	"dhrishti/types"
)

func NormalizeTCPConnectEvent(
	event TCPConnectEvent,
) types.RuntimeEvent {

	return types.RuntimeEvent{
		Type: types.EventConnect,

		Timestamp: time.Unix(
			0,
			int64(event.TimestampNS),
		),

		PID: event.PID,

		SourceIP:      uint32ToIP(event.SAddr),
		DestinationIP: uint32ToIP(event.DAddr),

		SourcePort:      event.SPort,
		DestinationPort: event.DPort,
	}
}

func NormalizeTCPCloseEvent(
	event TCPCloseEvent,
) types.RuntimeEvent {

	return types.RuntimeEvent{
		Type: types.EventClose,

		Timestamp: time.Unix(
			0,
			int64(event.TimestampNS),
		),

		PID: event.PID,

		SourceIP:      uint32ToIP(event.SAddr),
		DestinationIP: uint32ToIP(event.DAddr),

		SourcePort:      event.SPort,
		DestinationPort: event.DPort,
	}
}

func NormalizeTCPAcceptEvent(
	event TCPAcceptEvent,
) types.RuntimeEvent {

	return types.RuntimeEvent{
		Type: types.EventAccept,

		Timestamp: time.Unix(
			0,
			int64(event.TimestampNS),
		),

		PID: event.PID,

		SourceIP:      uint32ToIP(event.SAddr),
		DestinationIP: uint32ToIP(event.DAddr),

		SourcePort:      event.SPort,
		DestinationPort: event.DPort,
	}
}

func uint32ToIP(ip uint32) string {

	bytes := make([]byte, 4)

	bytes[0] = byte(ip)
	bytes[1] = byte(ip >> 8)
	bytes[2] = byte(ip >> 16)
	bytes[3] = byte(ip >> 24)

	return net.IP(bytes).String()
}
