# Flash-Sale E-Commerce Mock Stack

A 15-service microservices testbed that simulates a flash-sale e-commerce platform for Dhrishti integration and stress testing.

Load is generated externally via k6 — see [../benchmark/README.md](../benchmark/README.md).

## Architecture

```text
                    ┌─────────────────────────────────────────┐
                    │              api-gateway :8080           │
                    │         (only published host port)       │
                    └───────────────┬─────────────────────────┘
                                    │
     ┌──────────────┬───────────────┼───────────────┬──────────────┐
     ▼              ▼               ▼               ▼              ▼
 auth-service  product-catalog  flash-sale    search-service  cart-service
 user-service       │           -service          │              │
                    │               │              │              │
                    ▼               ▼              ▼              ▼
            recommendation    inventory +     product-catalog  inventory
               -service       pricing              │           -service
                    │               │              │              │
                    └───────────────┴──────────────┴──────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
              order-service   analytics-service  (checkout path)
                    │
        ┌───────────┼───────────┬──────────────┐
        ▼           ▼           ▼              ▼
  inventory    payment     shipping     notification
  -service     -service    -service      -service
```

## Services (15)

| Service | Role | Key dependencies |
|---------|------|------------------|
| **api-gateway** | HTTP entry point, routes all storefront APIs | Most services |
| **auth-service** | Token validation | — |
| **user-service** | User profiles | — |
| **product-catalog** | Product listings and featured items | — |
| **inventory-service** | Stock levels, flash-sale reservations | — |
| **pricing-service** | Dynamic / flash-sale pricing | — |
| **flash-sale-service** | Sale orchestration, reservation flow | inventory, pricing |
| **cart-service** | Shopping cart | inventory |
| **order-service** | Checkout orchestration | inventory, payment, shipping, notification |
| **payment-service** | Payment processing | — |
| **shipping-service** | Shipping quotes | — |
| **notification-service** | Order/event notifications | — |
| **search-service** | Product search | product-catalog |
| **recommendation-service** | Personalized picks | product-catalog |
| **analytics-service** | Fire-and-forget event tracking | — |

## API routes (gateway)

| Method | Path | Fan-out |
|--------|------|---------|
| GET | `/api/v1/products` | catalog + recommendations |
| GET | `/api/v1/products/:id` | catalog + inventory + pricing |
| GET | `/api/v1/search?q=` | search → catalog |
| GET | `/api/v1/flash-sale` | flash-sale → inventory + pricing |
| POST | `/api/v1/flash-sale/:id/reserve` | auth + flash-sale |
| GET/POST | `/api/v1/cart` | auth + cart |
| POST | `/api/v1/orders` | auth + user + order chain |
| GET | `/health` | health check |

## Production patterns

- **Gunicorn** with multiple workers/threads (not Flask dev server)
- **Health checks** on every service (`/health`)
- **Docker Compose** health-gated `depends_on`
- **Shared `common/`** module for latency simulation and HTTP helpers
- **Bimodal latency** and probabilistic failures on hot paths (inventory, payment, flash-sale)
- **Resource limits** on gateway, inventory, and flash-sale services

## Quick start

```bash
docker compose up --build --remove-orphans
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/flash-sale
```

Configure Dhrishti entry service in `dhrishti.json`:

```json
{ "entry_services": ["api-gateway"] }
```

## Benchmarking

```bash
cd ../benchmark
./baseline.sh              # Dhrishti OFF
sudo ./benchmark.sh        # Dhrishti ON
```

See [../benchmark/README.md](../benchmark/README.md) and [../docs/BENCHMARK.md](../docs/BENCHMARK.md).
