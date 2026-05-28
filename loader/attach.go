package loader

import (
	"fmt"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
)

type Probe struct {
	Collection *ebpf.Collection
	Program *ebpf.Program
	Link    link.Link
	Reader  *ringbuf.Reader
}

type ProbeSet struct {
	TCPConnect *Probe
	TCPClose   *Probe
	TCPAccept  *Probe
}

func AttachKprobe(
	objPath string,
	progName string,
	kernelFunc string,
) (*Probe, error) {

	spec, err := ebpf.LoadCollectionSpec(objPath)

	if err != nil {
		return nil, fmt.Errorf(
			"load spec: %w",
			err,
		)
	}

	coll, err := ebpf.NewCollection(spec)

	if err != nil {
		return nil, fmt.Errorf(
			"new collection: %w",
			err,
		)
	}

	prog := coll.Programs[progName]

	if prog == nil {
		return nil, fmt.Errorf(
			"program not found: %s",
			progName,
		)
	}

	lnk, err := link.Kprobe(
		kernelFunc,
		prog,
		nil,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"kprobe attach: %w",
			err,
		)
	}

	eventsMap := coll.Maps["events"]

	if eventsMap == nil {
		return nil, fmt.Errorf(
			"events map missing",
		)
	}

	rd, err := ringbuf.NewReader(eventsMap)

	if err != nil {
		return nil, fmt.Errorf(
			"ringbuf reader: %w",
			err,
		)
	}

	return &Probe{
		Collection: coll,
		Program: prog,
		Link:    lnk,
		Reader:  rd,
	}, nil
}

func AttachKretprobe(
	objPath string,
	progName string,
	kernelFunc string,
) (*Probe, error) {

	spec, err := ebpf.LoadCollectionSpec(objPath)

	if err != nil {
		return nil, fmt.Errorf(
			"load spec: %w",
			err,
		)
	}

	coll, err := ebpf.NewCollection(spec)

	if err != nil {
		return nil, fmt.Errorf(
			"new collection: %w",
			err,
		)
	}

	prog := coll.Programs[progName]

	if prog == nil {
		return nil, fmt.Errorf(
			"program not found: %s",
			progName,
		)
	}

	lnk, err := link.Kretprobe(
		kernelFunc,
		prog,
		nil,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"kretprobe attach: %w",
			err,
		)
	}

	eventsMap := coll.Maps["events"]

	if eventsMap == nil {
		return nil, fmt.Errorf(
			"events map missing",
		)
	}

	rd, err := ringbuf.NewReader(eventsMap)

	if err != nil {
		return nil, fmt.Errorf(
			"ringbuf reader: %w",
			err,
		)
	}

	return &Probe{
		Collection: coll,
		Program: prog,
		Link:    lnk,
		Reader:  rd,
	}, nil
}
