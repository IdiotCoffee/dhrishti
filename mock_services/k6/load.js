import http from "k6/http";
import { check, sleep } from "k6";

/*
  Configurable via environment variables:

  BASE_URL     — gateway target (default http://localhost:8080)
  VUS          — peak virtual users (default 10)
  DURATION     — steady-state duration (default 2m)
  RAMP_UP      — ramp-up duration (default 30s)
  RAMP_DOWN    — ramp-down duration (default 10s)
  SLEEP_MIN    — think time lower bound in seconds (default 0.5)
  SLEEP_MAX    — think time upper bound in seconds (default 1.5)
  EXECUTOR     — "ramping" (default) or "constant-arrival-rate"
  RATE         — target RPS when EXECUTOR=constant-arrival-rate (default 5)
*/

const baseUrl = __ENV.BASE_URL || "http://localhost:8080";
const vus = parseInt(__ENV.VUS || "10", 10);
const duration = __ENV.DURATION || "2m";
const rampUp = __ENV.RAMP_UP || "30s";
const rampDown = __ENV.RAMP_DOWN || "10s";
const sleepMin = parseFloat(__ENV.SLEEP_MIN || "0.5");
const sleepMax = parseFloat(__ENV.SLEEP_MAX || "1.5");
const executor = __ENV.EXECUTOR || "ramping";
const rate = parseInt(__ENV.RATE || "5", 10);

const rampingScenario = {
  executor: "ramping-vus",
  startVUs: 0,
  stages: [
    { duration: rampUp, target: vus },
    { duration: duration, target: vus },
    { duration: rampDown, target: 0 },
  ],
};

const constantScenario = {
  executor: "constant-arrival-rate",
  rate: rate,
  timeUnit: "1s",
  duration: duration,
  preAllocatedVUs: Math.max(vus, rate),
  maxVUs: Math.max(vus * 2, rate * 2),
};

export const options = {
  scenarios: {
    load: executor === "constant-arrival-rate" ? constantScenario : rampingScenario,
  },
  thresholds: {
    http_req_duration: ["p(95)<8000"],
    http_req_failed: ["rate<0.20"],
  },
};

export default function () {
  const res = http.get(`${baseUrl}/`, { timeout: "15s" });

  check(res, {
    "status is 200": (r) => r.status === 200,
  });

  const span = sleepMax - sleepMin;
  sleep(sleepMin + Math.random() * span);
}
