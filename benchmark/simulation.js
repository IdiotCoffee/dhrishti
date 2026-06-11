import http from "k6/http";
import { check, sleep } from "k6";

/*
  Configurable load simulation for Dhrishti.

  Env:
    CLIENTS / VIRTUAL_USERS  k6 virtual users (default 500)
    DURATION                 test length, e.g. 5s, 10m, 1h (default 4m)
    BASE_URL                 api-gateway URL
*/

const baseUrl = __ENV.BASE_URL || "http://localhost:8080";
const clients = parseInt(__ENV.CLIENTS || __ENV.VIRTUAL_USERS || "500", 10);
const duration = __ENV.DURATION || "4m";
const pool = Math.min(Math.max(clients, 10), 2000);

const flashSkus = ["sku-1001", "sku-1004", "sku-1005"];
const searchTerms = ["laptop", "shoes", "deal", "watch", "keyboard", "earbuds"];
const params = { headers: { Connection: "close" } };

function think(min = 0.2, max = 1.0) {
  return min + Math.random() * (max - min);
}

export const options = {
  scenarios: {
    mixed_load: {
      executor: "constant-vus",
      vus: pool,
      duration,
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.40"],
  },
};

export function setup() {
  console.log(`simulation clients=${clients} vus=${pool} duration=${duration}`);
}

export default function () {
  const roll = Math.random();
  if (roll < 0.35) {
    const res = http.get(`${baseUrl}/api/v1/products`, { timeout: "25s", ...params });
    check(res, { "browse ok": (r) => r.status === 200 });
  } else if (roll < 0.55) {
    const q = searchTerms[Math.floor(Math.random() * searchTerms.length)];
    const res = http.get(`${baseUrl}/api/v1/search?q=${q}`, { timeout: "20s", ...params });
    check(res, { "search ok": (r) => r.status === 200 });
  } else if (roll < 0.80) {
    const res = http.get(`${baseUrl}/api/v1/flash-sale`, { timeout: "30s", ...params });
    check(res, { "flash sale ok": (r) => r.status === 200 });
    if (Math.random() < 0.35) {
      const sku = flashSkus[Math.floor(Math.random() * flashSkus.length)];
      http.post(
        `${baseUrl}/api/v1/flash-sale/${sku}/reserve`,
        JSON.stringify({ user_id: `user-${__VU}-${__ITER}` }),
        { headers: { "Content-Type": "application/json", Connection: "close" }, timeout: "35s" },
      );
    }
  } else {
    http.get(`${baseUrl}/api/v1/cart`, { timeout: "20s", ...params });
    http.post(
      `${baseUrl}/api/v1/orders`,
      JSON.stringify({ product_id: flashSkus[Math.floor(Math.random() * flashSkus.length)], qty: 1 }),
      { headers: { "Content-Type": "application/json", Connection: "close" }, timeout: "60s" },
    );
  }
  sleep(think(0.1, 0.8));
}
