package loader

type TCPConnectEvent struct {
	PID uint32

	Comm [16]byte

	SAddr uint32
	DAddr uint32

	SPort uint16
	DPort uint16

	TimestampNS uint64
}

type TCPCloseEvent struct {
	PID uint32

	Comm [16]byte

	SAddr uint32
	DAddr uint32

	SPort uint16
	DPort uint16

	TimestampNS uint64
}

type TCPAcceptEvent struct {
	PID uint32

	Comm [16]byte

	SAddr uint32
	DAddr uint32

	SPort uint16
	DPort uint16

	TimestampNS uint64
}

type TCPStateEvent struct {
	PID uint32

	Comm [16]byte

	SAddr uint32
	DAddr uint32

	SPort uint16
	DPort uint16

	OldState uint32
	NewState uint32

	TimestampNS uint64
}
