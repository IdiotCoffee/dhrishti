//go:build !linux

package telemetry

import "runtime/metrics"

func readProcessCPUSeconds() float64 {
	samples := []metrics.Sample{{Name: "/cpu/classes/user:cpu-seconds"}}
	metrics.Read(samples)
	return samples[0].Value.Float64()
}
