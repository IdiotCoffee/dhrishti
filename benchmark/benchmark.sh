#!/usr/bin/env bash
set -euo pipefail

# Full flash-sale load test with Dhrishti ON + comparison report.
#
# Usage:
#   ./benchmark.sh
#   sudo CLIENTS=500 ./benchmark.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=_lib.sh
source "${SCRIPT_DIR}/_lib.sh"

ensure_root "$@"

K6_SUMMARY="${K6_SUMMARY:-/tmp/dhrishti-k6-summary.json}"
WITH_METRICS="${WITH_METRICS:-/tmp/dhrishti-with-metrics.json}"
BASELINE_K6="${BASELINE_K6:-/tmp/dhrishti-baseline-k6.json}"
RUN_PROFILE="${RUN_PROFILE:-/tmp/dhrishti-benchmark-env.txt}"

check_deps
check_stack

if ! curl -sf "${DHRISHTI_URL}/metrics" >/dev/null 2>&1; then
  echo "ERROR: Dhrishti not reachable at ${DHRISHTI_URL}"
  echo "Start it:  sudo go run main.go   (from repo root)"
  exit 1
fi

trap stop_fleet EXIT

echo "=============================================="
echo "  BENCHMARK — flash sale (Dhrishti ON)"
echo "=============================================="
echo "  clients (load scale): ${CLIENTS}"
echo "  client IPs (fleet):   $(fleet_client_ip_list)"
echo "  gateway:              ${BASE_URL}"
echo ""
echo "  Timeline: browse/search → flash rush → checkout (~4 min)"
echo "=============================================="
echo ""

save_run_profile "${RUN_PROFILE}"

echo "=== Pre-test dhrishti /metrics ==="
curl -sf "${DHRISHTI_URL}/metrics" | python3 -m json.tool
echo ""

start_fleet
run_k6 "${K6_SUMMARY}"

echo ""
echo "=== Post-test dhrishti /metrics ==="
DHRISHTI_METRICS="$(curl -sf "${DHRISHTI_URL}/metrics")"
echo "${DHRISHTI_METRICS}" > "${WITH_METRICS}"
echo "${DHRISHTI_METRICS}" | python3 -m json.tool

GRAPH_JSON="$(curl -sf "${DHRISHTI_URL}/graph")"

echo ""
echo "=== Results ==="
python3 - "${K6_SUMMARY}" "${DHRISHTI_METRICS}" "${GRAPH_JSON}" "${BASELINE_K6}" "${SCRIPT_DIR}" <<'PY'
import json, os, sys

k6_path, dhrishti_json, graph_json, baseline_path, script_dir = sys.argv[1:6]
sys.path.insert(0, script_dir)
from k6_parse import extract, print_k6

with open(k6_path) as f:
    k6 = json.load(f)


def find_metric(k6_data, name):
    metrics = k6_data.get("metrics")
    if isinstance(metrics, dict):
        if name in metrics:
            return metrics[name]
        for key, val in metrics.items():
            if key == name or key.startswith(name + "{"):
                return val
    return {}


def metric_values(metric):
    if not metric:
        return {}
    nested = metric.get("values")
    return nested if isinstance(nested, dict) and nested else metric


def trend_ms(metric):
    vals = metric_values(metric)
    return {k: float(vals[k]) for k in ("avg", "p(95)", "p(99)") if k in vals and vals[k] is not None}


def counter_rate(metric, k6_data):
    vals = metric_values(metric)
    if "rate" in vals:
        return float(vals["rate"])
    count = vals.get("count") or vals.get("value")
    dur = k6_data.get("state", {}).get("testRunDurationMs", 0) / 1000.0
    return float(count) / dur if count and dur else 0


def rate_value(metric):
    vals = metric_values(metric)
    if "value" in vals:
        return float(vals["value"])
    if "rate" in vals:
        return float(vals["rate"])
    return 0.0


http = find_metric(k6, "http_req_duration")
http_trend = trend_ms(http)
http_reqs = find_metric(k6, "http_reqs")
http_fail = find_metric(k6, "http_req_failed")
k6_rps = counter_rate(http_reqs, k6) or 0
k6_fail = rate_value(http_fail)

dhrishti = json.loads(dhrishti_json)
graph = json.loads(graph_json)
runtime = dhrishti["runtime"]
entry_services = graph.get("entry_services") or ["api-gateway"]
num_cpu = runtime.get("num_cpu") or os.cpu_count() or 1

print_k6("k6 HTTP (this run)", extract(k6))
print()

print("── dhrishti engine ──")
cores = runtime.get("cpu_cores_avg", 0)
pct = runtime.get("cpu_percent_of_machine", 0)
print(f"  events/s:        {runtime['throughput_events_per_second']:.1f}")
print(f"  heap:            {runtime['heap_inuse_bytes'] / 1024 / 1024:.1f} MiB")
print(f"  CPU:             {cores:.3f} cores ({pct:.2f}% of {num_cpu})")

client_edge = None
for edge in dhrishti["edges"]:
    if edge["source"] == "client" and edge["target"] in entry_services:
        if client_edge is None or edge["requests_per_second"] > client_edge["requests_per_second"]:
            client_edge = edge

print()
if client_edge:
    print(f"── Throughput (client → {client_edge['target']}) ──")
    print(f"  k6 req/s:        {k6_rps:.2f}")
    print(f"  dhrishti RPS:    {client_edge['requests_per_second']:.2f}")
else:
    print("── No client → gateway edge (check dhrishti.json entry_services) ──")

print()
print(f"── Service mesh ({len(dhrishti['edges'])} edges, top 8) ──")
for edge in sorted(dhrishti["edges"], key=lambda e: -e.get("requests_per_second", 0))[:8]:
    print(f"  {edge['source']} → {edge['target']}: {edge['requests_per_second']:.2f} rps")

print()
print(f"── Client IPs ({len(graph.get('unknown_ips', []))} hosts) ──")
for row in graph.get("unknown_ips", [])[:15]:
    print(f"  {row['ip']}: {row['connection_count']} conn, active={row['active_connections']}")

if os.path.isfile(baseline_path):
    base = extract(json.load(open(baseline_path)))
    cur = extract(k6)
    print()
    print("── vs baseline (Dhrishti was OFF) ──")
    print(f"  avg latency:  {base['avg_ms']:.0f} → {cur['avg_ms']:.0f} ms")
    print(f"  p95 latency:  {base['p95_ms']:.0f} → {cur['p95_ms']:.0f} ms")
    print(f"  request rate: {base['rps']:.2f} → {cur['rps']:.2f} req/s")
    print(f"  dhrishti CPU: 0 → {cores:.3f} cores")
else:
    print()
    print("(No baseline yet — run ./baseline.sh first for A/B comparison)")
PY

echo ""
echo "Saved → ${K6_SUMMARY}  ${WITH_METRICS}"
