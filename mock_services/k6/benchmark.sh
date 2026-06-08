#!/usr/bin/env bash
set -euo pipefail

# Run k6 and compare HTTP metrics against dhrishti eBPF edge metrics.
#
# Usage:
#   ./benchmark.sh
#   VUS=10 DURATION=1m ./benchmark.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DHRISHTI_URL="${DHRISHTI_URL:-http://localhost:8090}"
K6_SUMMARY="${K6_SUMMARY:-/tmp/dhrishti-k6-summary.json}"
WITH_METRICS="${WITH_METRICS:-/tmp/dhrishti-with-metrics.json}"
BENCH_ENV="${BENCH_ENV:-/tmp/dhrishti-benchmark-env.txt}"
BASELINE_ENV="${BASELINE_ENV:-/tmp/dhrishti-benchmark-env.txt}"

if ! command -v k6 >/dev/null 2>&1; then
  echo "k6 not found."
  echo "  Arch: yay -S k6-bin"
  echo "  https://grafana.com/docs/k6/latest/set-up/install-k6/"
  exit 1
fi

# Save env profile for compare.sh (should match baseline.sh).
cat > "${BENCH_ENV}" <<EOF
VUS=${VUS:-50000}
DURATION=${DURATION:-2m}
RAMP_UP=${RAMP_UP:-50s}
RAMP_DOWN=${RAMP_DOWN:-10s}
EXECUTOR=${EXECUTOR:-ramping}
RATE=${RATE:-5}
SLEEP_MIN=${SLEEP_MIN:-0.5}
SLEEP_MAX=${SLEEP_MAX:-1.5}
BASE_URL=${BASE_URL:-http://localhost:8080}
EOF

if [[ -f "${BASELINE_ENV}" ]] && ! diff -q "${BASELINE_ENV}" "${BENCH_ENV}" >/dev/null 2>&1; then
  echo "WARNING: k6 env differs from baseline run — compare.sh delta may be misleading"
  echo "  baseline env: ${BASELINE_ENV}"
  echo "  this run:     ${BENCH_ENV}"
  echo ""
fi

echo "=== Pre-test dhrishti /metrics ==="
curl -sf "${DHRISHTI_URL}/metrics" | python3 -m json.tool || {
  echo "dhrishti not reachable at ${DHRISHTI_URL}"
  echo "Start with: sudo go run main.go  (from repo root, with dhrishti.json present)"
  exit 1
}

echo ""
echo "=== Running k6 load test ==="
k6 run --summary-export "${K6_SUMMARY}" "${SCRIPT_DIR}/load.js"

echo ""
echo "=== Post-test dhrishti /metrics ==="
DHRISHTI_METRICS="$(curl -sf "${DHRISHTI_URL}/metrics")"
echo "${DHRISHTI_METRICS}" > "${WITH_METRICS}"
echo "${DHRISHTI_METRICS}" | python3 -m json.tool

GRAPH_JSON="$(curl -sf "${DHRISHTI_URL}/graph")"

echo ""
echo "=== Benchmark comparison ==="
python3 - "${K6_SUMMARY}" "${DHRISHTI_METRICS}" "${GRAPH_JSON}" <<'PY'
import json, os, sys

k6_path, dhrishti_json, graph_json = sys.argv[1], sys.argv[2], sys.argv[3]

with open(k6_path) as f:
    k6 = json.load(f)


def find_metric(k6_data, name):
    """Support legacy dict, flat dict, tagged names, and summary schema v1."""
    metrics = k6_data.get("metrics")
    if isinstance(metrics, dict):
        if name in metrics:
            return metrics[name]
        for key, val in metrics.items():
            if key == name or key.startswith(name + "{"):
                return val
    results = k6_data.get("results", {})
    if isinstance(results, dict):
        arr = results.get("metrics", [])
        if isinstance(arr, list):
            for m in arr:
                if m.get("name") == name:
                    return m
    return {}


def metric_values(metric):
    """k6 exports stats flat on the metric, or nested under 'values'."""
    if not metric:
        return {}
    nested = metric.get("values")
    if isinstance(nested, dict) and nested:
        return nested
    return metric


def trend_ms(metric):
    vals = metric_values(metric)
    out = {}
    for key in ("avg", "med", "min", "max", "p(90)", "p(95)", "p(99)"):
        if key in vals and vals[key] is not None:
            out[key] = float(vals[key])
    return out


def rate_value(metric):
    vals = metric_values(metric)
    # k6 rate metrics expose the rate directly as "value" (0.0–1.0)
    if "value" in vals and isinstance(vals["value"], (int, float)):
        return float(vals["value"])
    if "rate" in vals:
        return float(vals["rate"])
    return None


def counter_rate(metric, k6_data):
    vals = metric_values(metric)
    if "rate" in vals:
        return float(vals["rate"])
    count = vals.get("count") or vals.get("value")
    if count is None:
        return None
    duration = (
        k6_data.get("config", {}).get("duration")
        or k6_data.get("state", {}).get("testRunDurationMs", 0) / 1000.0
    )
    if duration and duration > 0:
        return float(count) / float(duration)
    return None


http = find_metric(k6, "http_req_duration")
http_trend = trend_ms(http)
http_reqs = find_metric(k6, "http_reqs")
http_fail = find_metric(k6, "http_req_failed")

k6_avg = http_trend.get("avg", 0)
k6_p95 = http_trend.get("p(95)", 0)
k6_p99 = http_trend.get("p(99)", 0)
k6_rps = counter_rate(http_reqs, k6) or rate_value(http_reqs) or 0
k6_fail = rate_value(http_fail)
if k6_fail is None:
    k6_fail = 0.0

dhrishti = json.loads(dhrishti_json)
graph = json.loads(graph_json)
runtime = dhrishti["runtime"]
entry_services = graph.get("entry_services") or ["gateway"]
num_cpu = os.cpu_count() or 1

print("── k6 (HTTP) ──")
if http_trend:
    print(f"  avg latency:     {k6_avg:.0f} ms")
    print(f"  p95 latency:     {k6_p95:.0f} ms")
    if k6_p99:
        print(f"  p99 latency:     {k6_p99:.0f} ms")
else:
    print("  (could not parse k6 summary — check k6 version / summary export format)")
print(f"  request rate:    {k6_rps:.2f} req/s")
print(f"  failure rate:    {k6_fail * 100:.1f}%")

print()
print("── dhrishti engine ──")
print(f"  events total:    {runtime['events_total']}")
print(f"  event throughput:{runtime['throughput_events_per_second']:.1f} events/s")
print(f"  heap in-use:     {runtime['heap_inuse_bytes'] / 1024 / 1024:.1f} MiB")
num_cpu = runtime.get("num_cpu") or num_cpu
cores_avg = runtime.get("cpu_cores_avg")
pct_machine = runtime.get("cpu_percent_of_machine")
if cores_avg is not None and pct_machine is not None:
    print(f"  CPU:             {cores_avg:.2f} cores avg ({pct_machine:.1f}% of {num_cpu} cores)")
else:
    cpu = runtime.get("process_cpu_seconds", runtime.get("cpu_total_seconds", 0))
    uptime = runtime.get("uptime_seconds", 0)
    if cpu and uptime:
        cores_used = cpu / uptime
        pct_of_machine = (cores_used / num_cpu) * 100
        print(f"  CPU:             {cores_used:.2f} cores avg ({pct_of_machine:.1f}% of {num_cpu} cores)")

client_edge = None
for edge in dhrishti["edges"]:
    if edge["source"] == "client" and edge["target"] in entry_services:
        if client_edge is None or edge["requests_per_second"] > client_edge["requests_per_second"]:
            client_edge = edge

print()
if client_edge:
    d_avg = client_edge["recent_average_latency_ms"]
    d_p95 = client_edge["p95_latency_ms"]
    print(f"── Comparable: throughput (client → {client_edge['target']}) ──")
    print(f"  k6 req/s:        {k6_rps:.2f}")
    print(f"  dhrishti RPS:    {client_edge['requests_per_second']:.2f}")
    if k6_rps > 0:
        print(f"  delta:           {client_edge['requests_per_second'] - k6_rps:+.2f} req/s")
    print()
    print(f"── Informational: TCP session duration (not HTTP latency) ──")
    print(f"  avg duration:    {d_avg} ms")
    print(f"  p95 duration:    {d_p95} ms")
    print(f"  failure rate:    {client_edge['failure_rate'] * 100:.1f}%")
    print("  (TCP connect→close; often higher than k6 HTTP time due to keep-alive)")
else:
    print(f"No client → {entry_services} edge found.")
    print("Ensure dhrishti.json lists your entry service and restart dhrishti.")

print()
print(f"── Client IP breakdown ({len(graph.get('unknown_ips', []))} hosts) ──")
for row in graph.get("unknown_ips", []):
    dests = ", ".join(f"{k}({v})" for k, v in sorted(row.get("destinations", {}).items(), key=lambda x: -x[1]))
    print(f"  {row['ip']}: {row['connection_count']} conn, active={row['active_connections']}, → {dests}")
PY

echo ""
echo "Metrics saved → ${WITH_METRICS}"
echo "Compare with baseline:  ./compare.sh"
