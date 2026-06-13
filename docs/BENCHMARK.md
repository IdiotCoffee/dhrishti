# Benchmarking & Load Testing

Dhrishti and k6 answer **different questions**. This guide covers three ways to generate traffic, how to compare results, and what to expect from each metric layer.

---

## Three ways to run load

| Mode | Command | When to use |
|------|---------|-------------|
| **Quick simulation** | `make run-simulation DURATION=15m VIRTUAL_USERS=200 CONNECTING_IPS=10` | Day-to-day testing, timeline replay, live graph under load |
| **Baseline (Dhrishti OFF)** | `cd benchmark && ./baseline.sh` | Capture k6 metrics without eBPF overhead |
| **Full benchmark (Dhrishti ON)** | `cd benchmark && sudo ./benchmark.sh` | A/B comparison vs baseline + mesh report |

All modes require the mock stack running (`cd mock_services && docker compose up --build`).

---

## Prerequisites

| Tool | Purpose | Install |
|------|---------|---------|
| Docker + Compose | Mock microservices | [docker.com](https://docs.docker.com/get-docker/) |
| k6 | HTTP load generator | [k6.io/docs](https://grafana.com/docs/k6/latest/set-up/install-k6/) |
| Go 1.21+ | Dhrishti engine | [go.dev](https://go.dev/doc/install) |
| Redis Stack | TimeSeries history | Local `redis-cli ping` → `PONG`; needs `TS.ADD` / `TS.RANGE` |
| sudo | eBPF probes + multi-IP client fleet | Required for engine and benchmark scripts |

k6 on Arch: `yay -S k6-bin` (or use the official k6 install guide above).

---

## Full A/B benchmark workflow

Run these in order from a fresh terminal session:

```bash
# 1 — Mock stack (terminal 1)
cd mock_services && docker compose up --build

# 2 — Baseline, Dhrishti OFF (terminal 2)
cd benchmark && ./baseline.sh

# 3 — Start Dhrishti (terminal 3)
cd /path/to/dhrishti
cp .env.example .env    # first time only
make dhrishti

# 4 — Benchmark, Dhrishti ON + comparison report (terminal 2)
cd benchmark && sudo ./benchmark.sh
```

`baseline.sh` saves k6 results to `/tmp/dhrishti-baseline-k6.json`. `benchmark.sh` runs the same flash-sale timeline with Dhrishti running and prints a comparison at the end.

Optional load scaling:

```bash
sudo CLIENTS=1000 ./benchmark.sh
```

`CLIENTS` scales k6 virtual users (default `500`). Client IP count is derived automatically (`CLIENTS / 50`, capped at 20).

---

## Quick simulation workflow

Use this when you want traffic for the live graph or timeline replay without running the full A/B scripts:

```bash
# Terminal 1 — mock stack
cd mock_services && docker compose up --build

# Terminal 2 — Dhrishti (engine + history API + frontend)
make dhrishti

# Terminal 3 — simulation
make run-simulation DURATION=15m VIRTUAL_USERS=200 CONNECTING_IPS=10
```

Tunables (override on the command line):

| Variable | Default | Example |
|----------|---------|---------|
| `DURATION` | `4m` | `15m`, `1h`, `300s` |
| `VIRTUAL_USERS` | `200` | `400` (keep ≤ 250 unless you know the stack can handle it) |
| `CONNECTING_IPS` | auto | `10` distinct client IPs on the docker bridge |

Run `make help` for a summary.

---

## Flash-sale timeline (~4 minutes)

Both `baseline.sh` and `benchmark.sh` run the same scenario automatically:

| Phase | Time | Traffic |
|-------|------|---------|
| Normal shopping | 0:00–0:40 | Browse catalog, search |
| Flash sale live | 0:40–2:30 | Flash rush, reserves, product views |
| Checkout | 0:50+ | Cart + orders |
| Ramp down | ~2:30 | Wind down |

A background **client fleet** (5–20 distinct IPs) runs when scripts execute with sudo. This binds multiple addresses on the docker bridge so Dhrishti sees realistic client IPs instead of one collapsed localhost proxy address.

---

## What to compare (Option B)

| Question | Use | Pass/fail? |
|----------|-----|------------|
| Is throughput right? | k6 `req/s` vs Dhrishti `client → api-gateway` RPS | Yes |
| Is the mesh visible? | Graph edges under load (15+ services) | Yes |
| Where are clients? | Client IPs sidebar | Yes |
| Is the gateway slow for users? | k6 HTTP `avg` / `p(95)` | Yes (k6 only) |
| Which hop is hot? | Dhrishti internal edge TCP heat | Yes (Dhrishti only) |
| Does client TCP p95 match k6 HTTP p95? | — | **No — different metrics** |

### What each system measures

| | k6 | Dhrishti |
|---|---|---|
| Layer | HTTP | TCP (kernel socket lifecycle) |
| Client latency | Request → response | Connect → close (session) |
| Throughput | HTTP req/s | Edge RPS + eBPF events/s |
| Scope | Gateway endpoints | Full 15-service dependency mesh |

### Why client TCP latency ≠ k6 HTTP latency

k6 times each HTTP request. Dhrishti times how long the **TCP socket stays open**. With HTTP keep-alive, one socket serves many requests, so Dhrishti session duration is often **much longer** than k6 HTTP time.

**Compare RPS.** Do not fail a benchmark because TCP p95 ≠ HTTP p95.

Dhrishti CPU when off = **0**. The delta is printed at the end of `benchmark.sh`.

---

## Benchmark script output

`benchmark.sh` prints:

1. **k6 (HTTP)** — user-facing latency and throughput
2. **Comparable** — k6 req/s vs Dhrishti client→api-gateway RPS
3. **Informational** — TCP session duration (do not compare to k6 HTTP latency)
4. **Service mesh** — top edges by RPS under flash-sale load

---

## Manual sanity checks

```bash
curl -s http://localhost:8090/graph | python3 -m json.tool
curl -s http://localhost:8090/metrics | python3 -m json.tool
curl -s http://localhost:8080/api/v1/flash-sale | python3 -m json.tool
curl -s http://localhost:8000/health | python3 -m json.tool
```

---

## Benchmark directory reference

| File | Purpose |
|------|---------|
| `baseline.sh` | Flash sale, Dhrishti off |
| `benchmark.sh` | Flash sale, Dhrishti on + comparison report |
| `simulation.js` | k6 scenario used by `make run-simulation` |
| `flash_sale.js` | k6 scenario used by baseline/benchmark scripts |
| `client_fleet.py` | Multi-IP background clients (auto-started with sudo) |
| `_lib.sh` | Shared dependency checks and fleet logic |

---

## Test matrix

| Test | Command | Verify |
|------|---------|--------|
| Default | `./baseline.sh` then `sudo ./benchmark.sh` | Full flash sale ~4 min |
| Heavier | `sudo CLIENTS=1000 ./benchmark.sh` | More k6 VUs + more client IPs |
| Engine cost | `sudo CLIENTS=1000 ./benchmark.sh` | Heap stable, CPU acceptable |
| Timeline replay | `make run-simulation DURATION=15m` + Timeline tab | Graph playback over the window |

---

## Cheat sheet

- **RPS match, TCP latency differs** → Normal (Option B)
- **No client edge** → Check `dhrishti.json` entry_services = `api-gateway`
- **User SLA** → k6 HTTP only
- **Hot path under flash sale** → `flash-sale → inventory`, `order → payment`
- **Simulation timeouts** → Lower `VIRTUAL_USERS` (start at 200) or increase mock stack resources
