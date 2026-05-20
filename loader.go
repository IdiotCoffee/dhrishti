package main

import (
	"bytes"
	"dhrishti/resolver"
	"encoding/binary"
	"fmt"
	"log"
	"net"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
)

// this struct must match the one used in tcp_connect.bpf.c
// as this is a binary contract - if it fails, nothing else works.

func init() {

}

type Event struct {
	Pid   uint32
	Comm  [16]byte
	Daddr uint32
	Dport uint16
}

func main() {
	ipMap, err := resolver.BuildIPServiceMap()
	if err != nil {
		log.Fatal(err)
	}
	// load the eBPF collection spec from object file
	spec, err := ebpf.LoadCollectionSpec("ebpf/tcp_connect.bpf.o")
	if err != nil {
		log.Fatalf("loading the spec: %v", err)
	}
	collection, err := ebpf.NewCollection(spec)
	if err != nil {
		log.Fatalf("creating collection: %v", err)
	}
	// you want to release these resources when the program exits
	defer collection.Close()

	prog := collection.Programs["trace_tcp_connect"]

	// connect to the probe! Attach my eBPF program to the tcp_connect function in the kernel.
	kp, err := link.Kprobe("tcp_connect", prog, nil)
	if err != nil {
		log.Fatalf("attaching kprobe: %v", err)
	}
	defer kp.Close()
	fmt.Println("eBPF tcp_connect tracer running...")
	eventsMap := collection.Maps["events"]
	// my probe emits events to a ring buffer, so read from there
	reader, err := ringbuf.NewReader(eventsMap)
	if err != nil {
		log.Fatalf("creating ringbuf reader: %v", err)
	}
	defer reader.Close()

	seenEdges := make(map[string]bool)

	for {
		record, err := reader.Read()
		if err != nil {
			log.Printf("reading ringbuf: %v", err)
			continue
		}

		var event Event
		// reading binary data from the buffer
		err = binary.Read(
			bytes.NewBuffer(record.RawSample),
			binary.LittleEndian,
			&event,
		)
		if err != nil {
			log.Printf("parsing event: %v", err)
			continue
		}
		ip := make(net.IP, 4)
		binary.LittleEndian.PutUint32(ip, event.Daddr)
		// show me the discoveries!
		// fmt.Printf(
		// 	"PID=%d COMM=%s DST=%s:%d\n",
		// 	event.Pid,
		// 	bytes.TrimRight(event.Comm[:], "\x00"),
		// 	ip,
		// 	event.Dport,
		// )
		resolvedContainer, err := resolver.ResolveContainerID(event.Pid)
		if err != nil {
			fmt.Println("error in resolver!", err)
			return
		}
		// fmt.Println(resolvedContainer)
		sourceService, err := resolver.ResolveServiceName(resolvedContainer)
		if err != nil {
			log.Println(err)
			continue
		}
		dstIP := ip.String()
		dstService := ipMap[dstIP]

		edgeKey := sourceService + "->" + dstService
		if !seenEdges[edgeKey] {
			seenEdges[edgeKey] = true

			fmt.Printf(
				"%s ─────▶ %s\n",
				sourceService,
				dstService,
			)
		}
	}

}
