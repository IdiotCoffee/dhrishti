# Mock Services Architecture

This folder contains a small microservice-style demo system used to generate realistic service-to-service traffic, latency, and occasional failures for observability demos.

## High-level topology

- `client` continuously sends requests to `gateway`.
- `gateway` calls:
  - `auth-service` (always)
  - `product-service` (always)
  - `order-service` (about 70% of requests)
- `product-service` calls `inventory-service`.
- `order-service` calls `inventory-service` and `payment-service` (with retry).

In short, traffic fans out from gateway to downstream services, creating edge activity and concurrent open connections that are visible in graphs.

## Services and behavior

### 1) `client`

File: `client/client.py`

- Generates burst-style traffic:
  - idles for a random interval
  - sends a burst of 1-2 requests
- Current idle interval is tuned for demos: **5-7 seconds**
- Adds a short 1-2 second gap between requests inside the same burst

### 2) `gateway`

File: `gateway/app.py`

- Entry point: `GET /`
- Sequentially fetches downstream data and returns an aggregated JSON payload
- Uses per-service timeouts:
  - auth: 5s
  - product: 8s
  - order: 15s
- Wraps timeout/network errors into structured responses

### 3) `auth-service`

File: `auth-service/app.py`

- Endpoint: `GET /auth`
- Simulates work by sleeping ~1.0-2.0s
- Returns authenticated user data

### 4) `product-service`

File: `product-service/app.py`

- Endpoint: `GET /product`
- Adds short local delay (~0.3-0.6s)
- Calls `inventory-service` and embeds inventory response in product payload
- Returns 502 on request failures to inventory

### 5) `inventory-service`

File: `inventory-service/app.py`

- Endpoint: `GET /inventory`
- Simulates slower backend latency (~1.0-2.5s)
- Returns random stock value
- Injects failures sometimes (~8% chance, HTTP 500)

### 6) `order-service`

File: `order-service/app.py`

- Endpoint: `GET /order`
- Calls `inventory-service`
- Calls `payment-service` with one retry loop (up to 2 attempts total)
- Waits 0.2s before retrying payment when needed
- Returns combined inventory + payment result

### 7) `payment-service`

File: `payment-service/app.py`

- Endpoint: `GET /pay`
- Simulates external provider latency (~1.0-2.0s)
- Sometimes fails (~10% chance, HTTP 500)

## Why this setup is useful for demos

- **Visible latency:** deliberate sleeps keep requests open long enough to observe active edges.
- **Fan-out patterns:** gateway and order create multi-hop request graphs.
- **Error signals:** random failures emulate real-world instability.
- **Retry behavior:** order -> payment retry demonstrates repeated downstream attempts.

## Running the stack

From `mock_services/`:

```bash
docker compose up --build
```

This starts all services defined in `mock_services/docker-compose.yml`.

Gateway is exposed on:

- `http://localhost:8080/`

The client runs inside the compose network and continuously drives traffic.

## Tuning traffic for demos

If you want to adjust pacing later, edit `client/client.py`:

- Request interval between bursts: `idle_seconds = random.uniform(<min>, <max>)`
- Requests per burst: `burst_size = random.randint(<min>, <max>)`
- Intra-burst delay: `time.sleep(random.uniform(<min>, <max>))`

For faster demos:

- lower idle bounds
- increase burst size

For quieter demos:

- increase idle bounds
- reduce burst size to 1
