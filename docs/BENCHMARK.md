# Benchmarking dhrishti vs k6 (Option B)

Dhrishti and k6 answer **different questions**. This guide uses **Option B**: compare metrics only where the layers align, and treat everything else as informational.

## Quick reference: what to compare

| Question | Use | Pass/fail? |
|----------|-----|------------|
| Is throughput right? | k6 `req/s` vs dhrishti `client → gateway` RPS | Yes |
| Is the mesh visible? | Graph edges under load | Yes |
| Where are clients? | Client IPs sidebar | Yes |
| Is the gateway slow for users? | k6 HTTP `avg` / `p(95)` | Yes (k6 only) |
| Which hop is hot? | dhrishti internal edge TCP heat | Yes (dhrishti only) |
| Does client TCP p95 match k6 HTTP p95? | — | **No — different metrics** |

## Prerequisites

```bash
# 1 — Mock services
cd mock_services && docker compose up --remove-orphans

# 2 — Dhrishti (repo root)
sudo go run main.go

# 3 — k6 (Arch AUR)
yay -S k6-bin
```

Entry services in `dhrishti.json`:

```json
{ "entry_services": ["gateway"] }
```

## Run a benchmark

### Full A/B (baseline + with-Dhrishti) — for overhead delta / resume numbers

Use the **same env vars** for both runs.

```bash
# 1 — Mock stack (no Dhrishti)
cd mock_services && docker compose up --remove-orphans

# 2 — Baseline: k6 only, Dhrishti OFF
cd mock_services/k6
VUS=10 DURATION=2m ./baseline.sh

# 3 — Start Dhrishti (repo root, separate terminal)
sudo go run main.go

# 4 — Same load with Dhrishti
VUS=10 DURATION=2m ./benchmark.sh

# 5 — Deltas + resume one-liner
./compare.sh
```

Dhrishti CPU when off = **0**. The delta comes from `/metrics` on the with-Dhrishti run. k6 HTTP delta shows whether eBPF affects the app (expect ~0).

### With-Dhrishti only

```bash
cd mock_services/k6
./benchmark.sh
VUS=10 DURATION=1m ./benchmark.sh
```

The script prints three sections:

1. **k6 (HTTP)** — user-facing latency and throughput
2. **Comparable** — k6 req/s vs dhrishti client→gateway RPS
3. **Informational** — TCP session duration (do not compare to k6 HTTP latency)

## What each system measures

| | k6 | dhrishti |
|---|---|---|
| Layer | HTTP | TCP (kernel socket lifecycle) |
| Client latency | Request → response | Connect → close (session) |
| Throughput | HTTP req/s | Edge RPS + eBPF events/s |
| Scope | One endpoint | Full dependency mesh |

### Why client TCP latency ≠ k6 HTTP latency

k6 times each HTTP request. dhrishti times how long the **TCP socket stays open**. With HTTP keep-alive, one socket serves many requests, so dhrishti session duration is often **much longer** than k6 HTTP time. That is expected, not a bug.

**Compare RPS.** Do not fail a benchmark because TCP p95 ≠ HTTP p95.

### Internal edges

`gateway → auth-service`, `gateway → product-service`, etc. measure **per-hop TCP** on short container-to-container flows. These are closer to segment-level timing and useful for finding which dependency is slow.

## CPU: what the number means

Dhrishti reports **process CPU from `/proc/self/stat`** (actual user+system time for the dhrishti binary only).

```
cpu_cores_avg = process_cpu_seconds / uptime
```

On a **16-core** machine, `0.05 cores avg (0.3% of 16 cores)` is typical at light load.

**Do not use** Go's `/cpu/classes/total:cpu-seconds` for this — that metric is `GOMAXPROCS × wall time` (CPU *budget*, not usage). On a 16-core machine it always averages ~16 "cores" regardless of actual load. That caused the bogus ~98% reading.

## Reading benchmark output

### Comparable (pass/fail)

```
── Comparable: throughput (client → gateway) ──
  k6 req/s:        2.22
  dhrishti RPS:    2.43
  delta:           +0.21 req/s
```

Within ~20% is healthy. Large gaps may mean rolling-window lag or connections still open.

### Informational only

```
── Informational: TCP session duration (not HTTP latency) ──
  avg duration:    17598 ms
  p95 duration:    19641 ms
```

Compare these to k6 **only** to understand the gap, not as an accuracy score.

### k6 HTTP (user SLA)

```
── k6 (HTTP) ──
  avg latency:     2900 ms
  p95 latency:     5590 ms
```

This is the number that answers “how slow is the gateway for users?”

## Manual checks

```bash
curl -s http://localhost:8090/graph | python3 -m json.tool
curl -s http://localhost:8090/metrics | python3 -m json.tool
VUS=5 DURATION=30s k6 run load.js
```

## Test matrix

| Test | Command | Verify |
|------|---------|--------|
| Smoke | `VUS=2 DURATION=30s ./benchmark.sh` | client→gateway, client IPs |
| Throughput | `VUS=10 DURATION=2m ./benchmark.sh` | k6 req/s ≈ client edge RPS |
| Hop latency | Same run | internal edges sensible vs mock sleeps |
| Engine cost | Soak 10m | heap stable, cpu_percent_of_machine acceptable |

## Cheat sheet

- **RPS match, TCP latency differs** → Normal (Option B)
- **No client edge** → Check `dhrishti.json` entry_services
- **CPU > 100% in old output** → Wrong label; use `cpu_cores_avg` / `cpu_percent_of_machine`
- **User SLA** → k6 HTTP only
