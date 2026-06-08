#!/usr/bin/env bash
set -euo pipefail

# Compare baseline (Dhrishti OFF) vs with-Dhrishti benchmark runs.
#
# Requires:
#   /tmp/dhrishti-baseline-k6.json     from ./baseline.sh
#   /tmp/dhrishti-k6-summary.json      from ./benchmark.sh
#   /tmp/dhrishti-with-metrics.json    from ./benchmark.sh (auto-saved)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASELINE_K6="${BASELINE_K6:-/tmp/dhrishti-baseline-k6.json}"
WITH_K6="${WITH_K6:-/tmp/dhrishti-k6-summary.json}"
WITH_METRICS="${WITH_METRICS:-/tmp/dhrishti-with-metrics.json}"
BASELINE_ENV="${BASELINE_ENV:-/tmp/dhrishti-benchmark-env.txt}"
BENCH_ENV="${BENCH_ENV:-/tmp/dhrishti-benchmark-env.txt}"

for f in "${BASELINE_K6}" "${WITH_K6}" "${WITH_METRICS}"; do
  if [[ ! -f "${f}" ]]; then
    echo "Missing ${f}"
    echo "Run:  ./baseline.sh  then  ./benchmark.sh  then  ./compare.sh"
    exit 1
  fi
done

python3 - "${SCRIPT_DIR}" "${BASELINE_K6}" "${WITH_K6}" "${WITH_METRICS}" "${BASELINE_ENV}" "${BENCH_ENV}" <<'PY'
import json, os, sys

sys.path.insert(0, sys.argv[1])
from k6_parse import extract, print_k6

baseline_path, with_k6_path, metrics_path, base_env, bench_env = sys.argv[2:7]

base = extract(json.load(open(baseline_path)))
with_k6 = extract(json.load(open(with_k6_path)))
metrics = json.load(open(metrics_path))
runtime = metrics["runtime"]
num_cpu = runtime.get("num_cpu") or os.cpu_count() or 1
cores = runtime.get("cpu_cores_avg", 0)
pct = runtime.get("cpu_percent_of_machine", 0)
heap_mib = runtime["heap_inuse_bytes"] / 1024 / 1024

print("=== Baseline vs With-Dhrishti ===")
print()

if os.path.exists(base_env) and os.path.exists(bench_env):
    if open(base_env).read().strip() != open(bench_env).read().strip():
        print("WARNING: env profiles differ — use the same VUS/DURATION for both runs")
        print(f"  baseline: {base_env}")
        print(f"  with:     {bench_env}")
        print()

print_k6("Baseline k6 (Dhrishti OFF)", base)
print()
print_k6("With-Dhrishti k6", with_k6)
print()

def delta(a, b, unit="", pct_base=None):
    d = b - a
    if pct_base and pct_base > 0:
        return f"{d:+.0f}{unit} ({d/pct_base*100:+.1f}%)"
    return f"{d:+.0f}{unit}"

print("── Application impact (k6 HTTP — expect ~0 delta) ──")
print(f"  avg latency:  {base['avg_ms']:.0f} ms → {with_k6['avg_ms']:.0f} ms  {delta(base['avg_ms'], with_k6['avg_ms'], ' ms', base['avg_ms'])}")
print(f"  p95 latency:  {base['p95_ms']:.0f} ms → {with_k6['p95_ms']:.0f} ms  {delta(base['p95_ms'], with_k6['p95_ms'], ' ms', base['p95_ms'])}")
print(f"  request rate: {base['rps']:.2f} → {with_k6['rps']:.2f} req/s  {delta(base['rps'], with_k6['rps'], ' req/s', base['rps'])}")
print()

print("── Dhrishti overhead (baseline = 0, process not running) ──")
print(f"  CPU added:     {cores:.3f} cores ({pct:.2f}% of {num_cpu} cores)")
print(f"  Memory:        {heap_mib:.1f} MiB heap in-use")
print(f"  Throughput:    {runtime['throughput_events_per_second']:.1f} eBPF events/s")
print()
PY
