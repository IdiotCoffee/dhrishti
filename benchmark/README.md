# Dhrishti Benchmark

Two scripts. One flash-sale test (~4 minutes). No other config needed.

## Prerequisites

```bash
cd mock_services && docker compose up -d    # mock stack
yay -S k6-bin                               # load generator (Arch)
```

## Run

```bash
cd benchmark

# 1 — Dhrishti OFF (auto re-execs with sudo for multi-IP clients)
./baseline.sh

# 2 — Start Dhrishti (repo root, other terminal)
sudo go run main.go

# 3 — Dhrishti ON (prints comparison vs baseline automatically)
./benchmark.sh
```

Scripts re-exec with `sudo` automatically so a background **client fleet** can bind 5–20 distinct IPs on the docker bridge and hit the api-gateway container directly (bypassing Docker’s localhost port proxy, which collapses all traffic into one bridge IP).

## What runs automatically

Each script runs the full flash-sale timeline in one shot:

| Phase | Time | Traffic |
|-------|------|---------|
| Normal shopping | 0:00–0:40 | Browse catalog, search |
| Flash sale live | 0:40–2:30 | Flash rush, reserves, product views |
| Checkout | 0:50+ | Cart + orders |
| Ramp down | ~2:30 | Wind down |

Plus a background **client fleet** (5–20 distinct IPs) when run with sudo.

## Optional override

```bash
sudo CLIENTS=1000 ./benchmark.sh
```

`CLIENTS` scales k6 load (default `500`). Client IP count is derived automatically (`CLIENTS / 50`, capped at 20).

## Files

| File | Purpose |
|------|---------|
| `baseline.sh` | Flash sale, Dhrishti off |
| `benchmark.sh` | Flash sale, Dhrishti on + report |
| `flash_sale.js` | k6 scenario (called automatically) |
| `client_fleet.py` | Multi-IP clients (called automatically) |
| `_lib.sh` | Internal shared logic |

Methodology: [docs/BENCHMARK.md](../docs/BENCHMARK.md)
