import http from "k6/http";
import { check, sleep } from "k6";

/*
  Full flash-sale timeline (one command, no config needed).

  Optional env:
    CLIENTS   Symbolic shopper count — scales k6 VUs (default 500)
    BASE_URL  Gateway (default http://localhost:8080)

  Timeline (~4 min):
    0:00–0:30   Normal — browse + search ramp up
    0:40–2:30   Flash sale goes live — rush, reserves, product views
    0:50+       Checkout wave
    2:30–3:00   Ramp down
*/

const baseUrl = __ENV.BASE_URL || "http://localhost:8080";
const clients = parseInt(__ENV.CLIENTS || "500", 10);
const pool = Math.min(Math.max(clients, 50), 800);

const flashSkus = ["sku-1001", "sku-1004", "sku-1005"];
const searchTerms = ["laptop", "shoes", "deal", "watch", "keyboard", "earbuds"];

const params = { headers: { Connection: "close" } };

function share(pct) {
  return Math.max(2, Math.floor(pool * pct));
}

function think(min = 0.2, max = 1.0) {
  return min + Math.random() * (max - min);
}

export const options = {
  scenarios: {
    browsing: {
      executor: "ramping-vus",
      exec: "browse",
      startVUs: 0,
      stages: [
        { duration: "30s", target: share(0.12) },
        { duration: "2m", target: share(0.12) },
        { duration: "30s", target: 0 },
      ],
    },
    search: {
      executor: "ramping-vus",
      exec: "searchCatalog",
      startVUs: 0,
      stages: [
        { duration: "30s", target: share(0.08) },
        { duration: "2m", target: share(0.08) },
        { duration: "30s", target: 0 },
      ],
    },
    flash_rush: {
      executor: "ramping-vus",
      exec: "flashSale",
      startTime: "40s",
      startVUs: 0,
      stages: [
        { duration: "45s", target: share(0.45) },
        { duration: "1m45s", target: share(0.45) },
        { duration: "30s", target: 0 },
      ],
    },
    product_detail: {
      executor: "ramping-vus",
      exec: "productDetail",
      startTime: "40s",
      startVUs: 0,
      stages: [
        { duration: "30s", target: share(0.15) },
        { duration: "2m", target: share(0.15) },
        { duration: "30s", target: 0 },
      ],
    },
    checkout: {
      executor: "ramping-vus",
      exec: "checkout",
      startTime: "50s",
      startVUs: 0,
      stages: [
        { duration: "40s", target: share(0.10) },
        { duration: "1m50s", target: share(0.10) },
        { duration: "30s", target: 0 },
      ],
    },
  },
  thresholds: {
    // Soft limits — saturated stack may breach latency; script still completes.
    http_req_failed: ["rate<0.35"],
  },
};

export function setup() {
  console.log(`flash_sale clients=${clients} vu_pool=${pool}`);
}

export function browse() {
  const res = http.get(`${baseUrl}/api/v1/products`, { timeout: "25s", ...params });
  check(res, { "browse ok": (r) => r.status === 200 });
  sleep(think());
}

export function flashSale() {
  const res = http.get(`${baseUrl}/api/v1/flash-sale`, { timeout: "30s", ...params });
  check(res, { "flash sale ok": (r) => r.status === 200 });
  if (Math.random() < 0.4) {
    const sku = flashSkus[Math.floor(Math.random() * flashSkus.length)];
    const reserve = http.post(
      `${baseUrl}/api/v1/flash-sale/${sku}/reserve`,
      JSON.stringify({ user_id: `user-${__VU}-${__ITER}` }),
      { headers: { "Content-Type": "application/json", Connection: "close" }, timeout: "35s" },
    );
    check(reserve, { "reserve ok": (r) => r.status === 200 || r.status === 409 });
  }
  sleep(think(0.15, 0.7));
}

export function checkout() {
  http.get(`${baseUrl}/api/v1/cart`, { timeout: "20s", ...params });
  const order = http.post(
    `${baseUrl}/api/v1/orders`,
    JSON.stringify({ product_id: flashSkus[Math.floor(Math.random() * flashSkus.length)], qty: 1 }),
    { headers: { "Content-Type": "application/json", Connection: "close" }, timeout: "60s" },
  );
  check(order, { "order placed": (r) => r.status === 200 });
  sleep(think(0.6, 2.0));
}

export function searchCatalog() {
  const q = searchTerms[Math.floor(Math.random() * searchTerms.length)];
  const res = http.get(`${baseUrl}/api/v1/search?q=${q}`, { timeout: "20s", ...params });
  check(res, { "search ok": (r) => r.status === 200 });
  sleep(think());
}

export function productDetail() {
  const sku = flashSkus[Math.floor(Math.random() * flashSkus.length)];
  const res = http.get(`${baseUrl}/api/v1/products/${sku}`, { timeout: "25s", ...params });
  check(res, { "product ok": (r) => r.status === 200 });
  sleep(think(0.2, 0.9));
}
