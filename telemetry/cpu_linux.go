//go:build linux

package telemetry

import (
	"os"
	"strconv"
	"strings"
)

// readProcessCPUSeconds returns cumulative user+system CPU time for this process
// from /proc/self/stat (field utime + stime, in clock ticks).
func readProcessCPUSeconds() float64 {
	data, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return 0
	}

	fields := strings.Fields(string(data))
	if len(fields) < 15 {
		return 0
	}

	utime, err1 := strconv.ParseUint(fields[13], 10, 64)
	stime, err2 := strconv.ParseUint(fields[14], 10, 64)
	if err1 != nil || err2 != nil {
		return 0
	}

	// Linux USER_HZ is typically 100 on x86_64.
	const clockTicks = 100.0
	return float64(utime+stime) / clockTicks
}
