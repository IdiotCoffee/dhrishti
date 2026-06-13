import http from "k6/http";
import { check, sleep } from "k6";

/*
  Configurable load simulation for Dhrishti.

  Env:
    VIRTUAL_USERS / CLIENTS  peak k6 virtual users (default 200)
    DURATION                 test length, e.g. 5s, 10m, 1h (default 4m)
    BASE_URL                 api-gateway URL
*/

const baseUrl = __ENV.BASE_URL || "http://localhost:8080";
const clients = parseInt(__ENV.VIRTUAL_USERS || __ENV.CLIENTS || "200", 10);
const duration = __ENV.DURATION || "4m";
const targetVus = Math.min(Math.max(clients, 10), 1000);

const flashSkus = ["sku-1001", "sku-1004", "sku-1005"];
const searchTerms = ["laptop", "shoes", "deal", "watch", "keyboard", "earbuds"];
const params = { headers: { Connection: "close" } };

function think(min = 0.3, max = 1.2) {
  return min + Math.random() * (max - min);
}

function rampStages(totalDuration, peakVus) {
  const match = String(totalDuration).match(/^(\d+)(s|m|h)$/);
  if (!match) {
    return [
      { duration: "30s", target: Math.max(10, Math.floor(peakVus * 0.2)) },
      { duration: "3m", target: peakVus },
      { duration: "30s", target: 0 },
    ];
  }

  const amount = parseInt(match[1], 10);
  const unit = match[2];
  const totalSec = unit === "s" ? amount : unit === "m" ? amount * 60 : amount * 3600;
  const rampSec = Math.min(120, Math.max(30, Math.floor(totalSec * 0.15)));
  const downSec = Math.min(60, Math.max(15, Math.floor(totalSec * 0.05)));
  const holdSec = Math.max(totalSec - rampSec - downSec, 30);

  const sec = (n) => `${n}s`;
  const warm = Math.max(10, Math.floor(peakVus * 0.25));

  return [
    { duration: sec(rampSec), target: warm },
    { duration: sec(Math.max(15, rampSec)), target: peakVus },
    { duration: sec(holdSec), target: peakVus },
    { duration: sec(downSec), target: 0 },
  ];
}

export const options = {
  scenarios: {
    mixed_load: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: rampStages(duration, targetVus),
      gracefulRampDown: "30s",
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.50"],
  },
};

export function setup() {
  console.log(
    `simulation virtual_users=${clients} peak_vus=${targetVus} duration=${duration}`,
  );
  if (targetVus > 250) {
    console.warn(
      "peak_vus > 250 — mock Flask stack may time out; try VIRTUAL_USERS=150 for long runs",
    );
  }
}

export default function () {
  const roll = Math.random();
  if (roll < 0.35) {
    const res = http.get(`${baseUrl}/api/v1/products`, { timeout: "45s", ...params });
    check(res, { "browse ok": (r) => r.status === 200 });
  } else if (roll < 0.55) {
    const q = searchTerms[Math.floor(Math.random() * searchTerms.length)];
    const res = http.get(`${baseUrl}/api/v1/search?q=${q}`, { timeout: "45s", ...params });
    check(res, { "search ok": (r) => r.status === 200 });
  } else if (roll < 0.80) {
    const res = http.get(`${baseUrl}/api/v1/flash-sale`, { timeout: "45s", ...params });
    check(res, { "flash sale ok": (r) => r.status === 200 });
    if (Math.random() < 0.35) {
      const sku = flashSkus[Math.floor(Math.random() * flashSkus.length)];
      http.post(
        `${baseUrl}/api/v1/flash-sale/${sku}/reserve`,
        JSON.stringify({ user_id: `user-${__VU}-${__ITER}` }),
        { headers: { "Content-Type": "application/json", Connection: "close" }, timeout: "45s" },
      );
    }
  } else {
    http.get(`${baseUrl}/api/v1/cart`, { timeout: "45s", ...params });
    http.post(
      `${baseUrl}/api/v1/orders`,
      JSON.stringify({ product_id: flashSkus[Math.floor(Math.random() * flashSkus.length)], qty: 1 }),
      { headers: { "Content-Type": "application/json", Connection: "close" }, timeout: "90s" },
    );
  }
  sleep(think());
}
