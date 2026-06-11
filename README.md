# Dhrishti

*A runtime observability engine for dynamically reconstructing distributed system architecture using eBPF.*

---

## TL;DR

**What it is:** Dhrishti watches live TCP traffic in the Linux kernel and rebuilds a real-time dependency graph of your microservices — no app instrumentation required.

**Probes:** `tcp_connect` (outbound intent), `inet_csk_accept` (server accept), `tcp_close` (teardown / duration). Events flow through eBPF ring buffers into a Go correlation engine that resolves Docker container identities and computes per-edge RPS, latency, and failure rates.

**Stack:** eBPF (C) → Go engine → WebSocket/HTTP API → React + Cytoscape.js. Historical metrics are written to **local Redis** (TimeSeries) and served by a **FastAPI** history API.

**Prerequisite:** Redis running locally on `localhost:6379` (`redis-cli ping` → `PONG`). Redis Stack is required for TimeSeries (`TS.ADD` / `TS.RANGE`). Copy `.env.example` → `.env` to configure Redis URL and history settings (loaded by `make dhrishti`).

**Run it (3 commands):**

```bash
# 1. Mock microservices
cd mock_services && docker compose up --build

# 2. Dhrishti (Go engine + History API + frontend)
make dhrishti

# 3. Simulate traffic (tunable)
make run-simulation DURATION=4m VIRTUAL_USERS=500 CONNECTING_IPS=10
```

Open **http://localhost:5173** for the live graph. Go API: `:8090`. History API: `:8000`.

**Benchmarks:** `benchmark/` runs k6 flash-sale load against a 15-service Docker stack. Use `make run-simulation` for a quick configurable run, or `benchmark/benchmark.sh` for the full A/B comparison vs baseline. See `docs/BENCHMARK.md`.

---

# Overview

Dhrishti is a Linux-based observability system that infers service dependencies by tracing live TCP activity directly from the kernel using eBPF.

Instead of relying on:

* manually maintained architecture diagrams,
* application-level tracing instrumentation,
* or predefined service topology,

Dhrishti observes actual runtime behavior and reconstructs communication relationships dynamically.

The system captures kernel-level TCP lifecycle events, correlates them into runtime flows, enriches them with container metadata, and exposes a live dependency graph of the running system.

---

# Architecture

```text
Docker Workloads
        ↓
eBPF Kernel Probes
        ↓
Ring Buffer Telemetry
        ↓
Go Runtime Collector
        ↓
Flow Correlation Engine
        ↓
Runtime Graph State
        ↓
WebSocket Streaming
        ↓
Cytoscape.js Visualization
```

---

# Runtime Model

The Linux kernel only exposes low-level primitives such as:

```text
processes
sockets
IP addresses
ports
TCP state
```

Dhrishti converts these primitives into higher-level architectural relationships such as:

```text
api-gateway → auth-service
api-gateway → flash-sale-service → inventory-service
order-service → payment-service
```

This allows the system to reconstruct distributed service topology dynamically from runtime execution itself.

---

# eBPF Probes

Dhrishti currently uses multiple eBPF probes to observe different stages of the TCP lifecycle.

## `tcp_connect`

Tracks outbound TCP connection attempts.

This forms the foundation of dependency inference by identifying:

* which process initiated communication,
* destination IPs,
* and destination ports.

---

## `inet_csk_accept`

Tracks server-side socket acceptance.

This provides the server perspective of a connection and enables:

* connection validation,
* client/server correlation,
* and failed connection inference.

---

## `tcp_close`

Tracks TCP connection teardown events.

This allows Dhrishti to compute:

* connection duration,
* active flow counts,
* retry churn,
* and short-lived connection behavior.

---

## `tcp_state`

Tracks TCP state transitions such as:

```text
ESTABLISHED
FIN_WAIT
TIME_WAIT
```

This adds deeper visibility into runtime connection behavior.

---

# Flow Correlation

Raw kernel telemetry is stateless.

Individual events only indicate:

```text
connect happened
accept happened
close happened
```

Dhrishti correlates these events into logical runtime flows.

This enables the system to infer:

* successful connections,
* failed handshakes,
* flow duration,
* rolling latency metrics,
* and dependency behavior over time.

---

# Current Features

* Live TCP dependency inference
* Docker-aware service resolution
* Runtime flow tracking
* Connection lifecycle reconstruction
* Rolling latency calculations
* P95 latency tracking
* WebSocket-based graph streaming
* Live Cytoscape visualization

---

# Example Runtime Graph


![Runtime Graph](./assets/graph.png)


---

# Example Test Architecture

Dhrishti is tested against a **flash-sale e-commerce** mock stack (`mock_services/`) with **15 microservices**:

```text
k6 (host) → api-gateway
                ├→ auth-service, user-service, analytics-service
                ├→ product-catalog ← search-service, recommendation-service
                ├→ flash-sale-service → inventory-service, pricing-service
                ├→ cart-service → inventory-service
                └→ order-service → payment, shipping, notification
```

The benchmark suite (`benchmark/`) drives realistic traffic with k6 — configurable **50k–100k virtual users** simulating a flash sale (browse, search, reserve, checkout). Dhrishti reconstructs the full dependency mesh under load.

| Component | Location | Purpose |
|-----------|----------|---------|
| Mock microservices | `mock_services/` | 15-service Docker Compose stack |
| Load testing | `benchmark/` | k6 flash-sale scenarios + A/B scripts |
| Benchmark guide | `docs/BENCHMARK.md` | Comparison methodology |

---

# Tech Stack

| Layer                  | Technology        |
| ---------------------- | ----------------- |
| Kernel Instrumentation | eBPF              |
| Probe Language         | C                 |
| Telemetry Engine       | Go                |
| Runtime Metadata       | Docker Engine API |
| Frontend               | React             |
| Visualization          | Cytoscape.js      |
| Streaming              | WebSockets        |

---

# Running Dhrishti

Clone the repository:

```bash
git clone <repo-url>
cd dhrishti
```

Then use the three commands from the TL;DR above. `make dhrishti` builds eBPF probes if needed, starts the Go engine (requires sudo for eBPF), the FastAPI history API on `:8000`, and the Vite frontend on `:5173`.

For full A/B benchmark comparison:

```bash
cd benchmark/
./baseline.sh              # Dhrishti OFF (run first)
sudo ./benchmark.sh        # Dhrishti ON (comparison vs baseline)
```

See `make help` for simulation tunables (`DURATION`, `VIRTUAL_USERS`, `CONNECTING_IPS`).
