package telemetry

import (
	"runtime"
	"sync/atomic"
	"time"
)

// Counter tracks eBPF events processed by dhrishti.
type Counter struct {
	startTime time.Time

	connect atomic.Uint64
	accept  atomic.Uint64
	close   atomic.Uint64
}

func NewCounter() *Counter {
	return &Counter{startTime: time.Now()}
}

func (c *Counter) IncConnect() { c.connect.Add(1) }
func (c *Counter) IncAccept()  { c.accept.Add(1) }
func (c *Counter) IncClose()   { c.close.Add(1) }

type Snapshot struct {
	UptimeSeconds           float64 `json:"uptime_seconds"`
	EventsConnect           uint64  `json:"events_connect"`
	EventsAccept            uint64  `json:"events_accept"`
	EventsClose             uint64  `json:"events_close"`
	EventsTotal             uint64  `json:"events_total"`
	ThroughputEventsPerSec  float64 `json:"throughput_events_per_second"`
	Goroutines              int     `json:"goroutines"`
	HeapAllocBytes          uint64  `json:"heap_alloc_bytes"`
	HeapInuseBytes          uint64  `json:"heap_inuse_bytes"`
	SysBytes                uint64  `json:"sys_bytes"`
	ProcessCPUSseconds      float64 `json:"process_cpu_seconds"`
	NumCPU                  int     `json:"num_cpu"`
	CPUCoresAvg             float64 `json:"cpu_cores_avg"`
	CPUPercentOfMachine     float64 `json:"cpu_percent_of_machine"`
}

func (c *Counter) Snapshot() Snapshot {
	connect := c.connect.Load()
	accept := c.accept.Load()
	closeEv := c.close.Load()
	total := connect + accept + closeEv

	uptime := time.Since(c.startTime).Seconds()
	var eps float64
	if uptime > 0 {
		eps = float64(total) / uptime
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	cpuTotal := readProcessCPUSeconds()
	numCPU := runtime.NumCPU()
	var coresAvg, pctMachine float64
	if uptime > 0 {
		coresAvg = cpuTotal / uptime
		if numCPU > 0 {
			pctMachine = (coresAvg / float64(numCPU)) * 100
		}
	}

	return Snapshot{
		UptimeSeconds:          uptime,
		EventsConnect:          connect,
		EventsAccept:           accept,
		EventsClose:            closeEv,
		EventsTotal:            total,
		ThroughputEventsPerSec: eps,
		Goroutines:             runtime.NumGoroutine(),
		HeapAllocBytes:         mem.HeapAlloc,
		HeapInuseBytes:         mem.HeapInuse,
		SysBytes:               mem.Sys,
		ProcessCPUSseconds:     cpuTotal,
		NumCPU:                 numCPU,
		CPUCoresAvg:            coresAvg,
		CPUPercentOfMachine:    pctMachine,
	}
}
