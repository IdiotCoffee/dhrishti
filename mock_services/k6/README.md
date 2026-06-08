# k6 Load Testing

[k6](https://grafana.com/docs/k6/latest/) is a modern load testing tool. Scripts are JavaScript, results include p95/p99 latency, and configuration is driven by environment variables — no code changes needed to tune a run.

## Prerequisites

```bash
# Install k6 — https://grafana.com/docs/k6/latest/set-up/install-k6/
  # Arch Linux — k6 is in AUR, not official repos:
  # yay -S k6-bin   OR   paru -S k6-bin
  # Or download binary: https://github.com/grafana/k6/releases
# macOS: brew install k6
```

Start the mock stack (without the old Python client):

```bash
cd mock_services
docker compose up --build
```

Start dhrishti on the host:

```bash
sudo go run main.go
```

## Quick run

```bash
cd mock_services/k6
k6 run load.js
```

## Configuration (environment variables)

| Variable | Default | Description |
|----------|---------|-------------|
| `BASE_URL` | `http://localhost:8080` | Gateway target |
| `VUS` | `10` | Peak virtual users (concurrent) |
| `DURATION` | `2m` | Steady-state duration |
| `RAMP_UP` | `30s` | Ramp-up duration |
| `RAMP_DOWN` | `10s` | Ramp-down duration |
| `SLEEP_MIN` | `0.5` | Think time lower bound (seconds) |
| `SLEEP_MAX` | `1.5` | Think time upper bound (seconds) |
| `EXECUTOR` | `ramping` | `ramping` or `constant-arrival-rate` |
| `RATE` | `5` | Target req/s when using constant-arrival-rate |

### Examples

```bash
# 20 concurrent users for 1 minute
VUS=20 DURATION=1m k6 run load.js

# Fixed 10 req/s for 3 minutes
EXECUTOR=constant-arrival-rate RATE=10 DURATION=3m k6 run load.js

# Aggressive soak test
VUS=50 DURATION=10m RAMP_UP=1m k6 run load.js
```

## Important stats to test dhrishti with

| k6 metric | What it tells you |
|-----------|-------------------|
| `http_req_duration` (p95, p99) | End-to-end latency — compare against dhrishti edge p95 |
| `http_reqs` (rate) | Throughput — compare against dhrishti edge RPS |
| `http_req_failed` (rate) | Error rate — compare against dhrishti edge failure_rate |
| `vus` / `vus_max` | Concurrency level during the test |
| `iteration_duration` | Full loop time including think-time |
| `data_received` / `data_sent` | Network volume (less relevant for dhrishti, useful for infra sizing) |

**Dhrishti-specific metrics** (via `GET http://localhost:8090/metrics`):

| Field | What it tells you |
|-------|-------------------|
| `events_total` / `throughput_events_per_second` | eBPF event processing throughput |
| `heap_inuse_bytes` | Memory overhead of the observability engine |
| `goroutines` | Concurrency overhead |
| `edges[].p95_latency_ms` | Per-dependency tail latency (TCP connection lifetime) |

Note: dhrishti measures **TCP connection duration**, not HTTP response time. Under load, client→gateway p95 should correlate with k6 p95 but won't match exactly (gateway fan-out adds downstream time that eBPF sees as separate edges).

## Benchmark script

Runs k6 and compares results against dhrishti `/metrics` and `/graph`:

```bash
cd mock_services/k6
./benchmark.sh

# Custom load
VUS=15 DURATION=1m ./benchmark.sh
```

Full guide: [docs/BENCHMARK.md](../../docs/BENCHMARK.md)

**Before running:** ensure `dhrishti.json` in the repo root lists your entry service (default: `gateway`). Unresolved host traffic hitting an entry service appears as `client → gateway` on the graph, with real IPs in the sidebar table.
