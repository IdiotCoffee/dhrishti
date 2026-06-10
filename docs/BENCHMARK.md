# Benchmarking dhrishti vs k6 (Option B)

Dhrishti and k6 answer **different questions**. This guide uses **Option B**: compare metrics only where the layers align, and treat everything else as informational.

## Quick reference: what to compare

| Question | Use | Pass/fail? |
|----------|-----|------------|
| Is throughput right? | k6 `req/s` vs dhrishti `client → api-gateway` RPS | Yes |
| Is the mesh visible? | Graph edges under load (15+ services) | Yes |
| Where are clients? | Client IPs sidebar | Yes |
| Is the gateway slow for users? | k6 HTTP `avg` / `p(95)` | Yes (k6 only) |
| Which hop is hot? | dhrishti internal edge TCP heat | Yes (dhrishti only) |
| Does client TCP p95 match k6 HTTP p95? | — | **No — different metrics** |

## Prerequisites

```bash
# 1 — Flash-sale mock stack (15 services)
cd mock_services && docker compose up --build --remove-orphans

# 2 — Dhrishti (repo root)
sudo go run main.go

# 3 — k6 (Arch AUR)
yay -S k6-bin
```

Entry services in `dhrishti.json`:

```json
{ "entry_services": ["api-gateway"] }
```

## Run a benchmark

```bash
# 1 — Mock stack
cd mock_services && docker compose up -d

# 2 — Baseline (Dhrishti OFF)
cd benchmark && ./baseline.sh

# 3 — Start Dhrishti (repo root)
sudo go run main.go

# 4 — Benchmark (Dhrishti ON, compares vs baseline automatically)
cd benchmark && sudo ./benchmark.sh
```

Optional: `sudo CLIENTS=500 ./benchmark.sh` to scale load (default 500).

Dhrishti CPU when off = **0**. The delta is printed at the end of `benchmark.sh`.

The script prints:

1. **k6 (HTTP)** — user-facing latency and throughput
2. **Comparable** — k6 req/s vs dhrishti client→api-gateway RPS
3. **Informational** — TCP session duration (do not compare to k6 HTTP latency)
4. **Service mesh** — top edges by RPS under flash-sale load

## Flash-sale scenario

`baseline.sh` and `benchmark.sh` both run the same ~4 minute timeline automatically: browse/search → flash rush → checkout. See [benchmark/README.md](../benchmark/README.md).

## What each system measures

| | k6 | dhrishti |
|---|---|---|
| Layer | HTTP | TCP (kernel socket lifecycle) |
| Client latency | Request → response | Connect → close (session) |
| Throughput | HTTP req/s | Edge RPS + eBPF events/s |
| Scope | Gateway endpoints | Full 15-service dependency mesh |

### Why client TCP latency ≠ k6 HTTP latency

k6 times each HTTP request. dhrishti times how long the **TCP socket stays open**. With HTTP keep-alive, one socket serves many requests, so dhrishti session duration is often **much longer** than k6 HTTP time.

**Compare RPS.** Do not fail a benchmark because TCP p95 ≠ HTTP p95.

## Test matrix

| Test | Command | Verify |
|------|---------|--------|
| Default | `./baseline.sh` then `sudo ./benchmark.sh` | Full flash sale ~4 min |
| Heavier | `sudo CLIENTS=1000 ./benchmark.sh` | More k6 VUs + more client IPs |
| Engine cost | `sudo CLIENTS=1000 ./benchmark.sh` | heap stable, cpu acceptable |

## Manual checks

```bash
curl -s http://localhost:8090/graph | python3 -m json.tool
curl -s http://localhost:8090/metrics | python3 -m json.tool
curl -s http://localhost:8080/api/v1/flash-sale | python3 -m json.tool
```

## Cheat sheet

- **RPS match, TCP latency differs** → Normal (Option B)
- **No client edge** → Check `dhrishti.json` entry_services = `api-gateway`
- **User SLA** → k6 HTTP only
- **Hot path under flash sale** → `flash-sale → inventory`, `order → payment`
