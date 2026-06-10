"""Shared k6 summary-export parser for baseline/benchmark/compare scripts."""

import json
import sys


def load(path):
    with open(path) as f:
        return json.load(f)


def find_metric(k6_data, name):
    metrics = k6_data.get("metrics")
    if isinstance(metrics, dict):
        if name in metrics:
            return metrics[name]
        for key, val in metrics.items():
            if key == name or key.startswith(name + "{"):
                return val
    results = k6_data.get("results", {})
    if isinstance(results, dict):
        for m in results.get("metrics", []) or []:
            if m.get("name") == name:
                return m
    return {}


def metric_values(metric):
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


def extract(k6_data):
    http = find_metric(k6_data, "http_req_duration")
    http_trend = trend_ms(http)
    http_reqs = find_metric(k6_data, "http_reqs")
    http_fail = find_metric(k6_data, "http_req_failed")
    fail = rate_value(http_fail)
    return {
        "avg_ms": http_trend.get("avg", 0),
        "p95_ms": http_trend.get("p(95)", 0),
        "p99_ms": http_trend.get("p(99)", 0),
        "rps": counter_rate(http_reqs, k6_data) or rate_value(http_reqs) or 0,
        "fail_rate": fail if fail is not None else 0.0,
    }


def print_k6(label, stats):
    print(f"── {label} ──")
    if stats["avg_ms"]:
        print(f"  avg latency:     {stats['avg_ms']:.0f} ms")
        print(f"  p95 latency:     {stats['p95_ms']:.0f} ms")
        if stats["p99_ms"]:
            print(f"  p99 latency:     {stats['p99_ms']:.0f} ms")
    else:
        print("  (could not parse k6 summary)")
    print(f"  request rate:    {stats['rps']:.2f} req/s")
    print(f"  failure rate:    {stats['fail_rate'] * 100:.1f}%")


if __name__ == "__main__":
    print_k6("k6", extract(load(sys.argv[1])))
