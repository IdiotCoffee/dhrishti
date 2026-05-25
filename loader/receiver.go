package loader

import (
	"bytes"
	"encoding/binary"
	"log"

	"github.com/cilium/ebpf/ringbuf"
)

func ReadTCPConnectEvents(
	rd *ringbuf.Reader,
	handler func(TCPConnectEvent),
) {
	for {
		record, err := rd.Read()

		if err != nil {
			log.Printf(
				"tcp_connect ringbuf read error: %v",
				err,
			)
			continue
		}

		var event TCPConnectEvent

		err = binary.Read(
			bytes.NewBuffer(record.RawSample),
			binary.LittleEndian,
			&event,
		)

		if err != nil {
			log.Printf(
				"tcp_connect decode error: %v",
				err,
			)
			continue
		}

		handler(event)
	}
}

func ReadTCPCloseEvents(
	rd *ringbuf.Reader,
	handler func(TCPCloseEvent),
) {
	for {
		record, err := rd.Read()

		if err != nil {
			log.Printf(
				"tcp_close ringbuf read error: %v",
				err,
			)
			continue
		}

		var event TCPCloseEvent

		err = binary.Read(
			bytes.NewBuffer(record.RawSample),
			binary.LittleEndian,
			&event,
		)

		if err != nil {
			log.Printf(
				"tcp_close decode error: %v",
				err,
			)
			continue
		}

		handler(event)
	}
}

func ReadTCPAcceptEvents(
	rd *ringbuf.Reader,
	handler func(TCPAcceptEvent),
) {
	for {
		record, err := rd.Read()

		if err != nil {
			log.Printf(
				"tcp_accept ringbuf read error: %v",
				err,
			)
			continue
		}

		var event TCPAcceptEvent

		err = binary.Read(
			bytes.NewBuffer(record.RawSample),
			binary.LittleEndian,
			&event,
		)

		if err != nil {
			log.Printf(
				"tcp_accept decode error: %v",
				err,
			)
			continue
		}

		handler(event)
	}
}
