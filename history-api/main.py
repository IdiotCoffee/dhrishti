"""Dhrishti history API — reads metrics and graph snapshots from Redis."""

from __future__ import annotations

import json
import os
from datetime import datetime, timedelta, timezone
from typing import Any
from urllib.parse import urlparse

import redis
from dateutil import parser as date_parser
from fastapi import FastAPI, HTTPException, Query
from fastapi.middleware.cors import CORSMiddleware

SERVICES_KEY = "dhrishti:services:known"
SNAPSHOTS_INDEX_KEY = "dhrishti:snapshots:index"

SERVICE_METRICS = {
    "outbound_rps": "Outbound requests per second",
    "inbound_rps": "Inbound requests per second",
    "failure_rate": "Failure rate (0–1)",
    "p95_latency_ms": "P95 latency (ms)",
    "active_connections": "Active connections",
}

EDGE_METRICS = {
    "rps": "Requests per second",
    "failure_rate": "Failure rate (0–1)",
    "p95_latency_ms": "P95 latency (ms)",
    "avg_latency_ms": "Average latency (ms)",
    "active_connections": "Active connections",
}

DEFAULT_REDIS_URL = os.getenv("DHRISHTI_REDIS_URL", "redis://localhost:6379")
HISTORY_API_PORT = int(os.getenv("DHRISHTI_HISTORY_API_PORT", "8000"))
RETENTION_HOURS = int(os.getenv("DHRISHTI_HISTORY_RETENTION_HOURS", "24"))


def redis_client() -> redis.Redis:
    parsed = urlparse(DEFAULT_REDIS_URL)
    db = 0
    if parsed.path and parsed.path != "/":
        db = int(parsed.path.lstrip("/"))
    return redis.Redis(
        host=parsed.hostname or "localhost",
        port=parsed.port or 6379,
        password=parsed.password,
        db=db,
        decode_responses=True,
    )


rdb = redis_client()

app = FastAPI(title="Dhrishti History API", version="0.1.0")
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)


def parse_time(value: str) -> datetime:
    dt = date_parser.isoparse(value)
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=timezone.utc)
    return dt.astimezone(timezone.utc)


def clamp_history_window(start: datetime, end: datetime) -> tuple[datetime, datetime]:
    now = datetime.now(timezone.utc)
    earliest = now - timedelta(hours=RETENTION_HOURS)
    if end > now:
        end = now
    if start < earliest:
        start = earliest
    if start >= end:
        raise HTTPException(status_code=400, detail="start must be before end")
    return start, end


def to_ms(dt: datetime) -> int:
    return int(dt.timestamp() * 1000)


def sanitize_label(value: str) -> str:
    return value.replace(":", "_").replace("|", "_").replace(" ", "_")


def service_series_key(service: str, metric: str) -> str:
    return f"dhrishti:ts:svc:{sanitize_label(service)}:{metric}"


@app.get("/health")
def health() -> dict[str, str]:
    try:
        rdb.ping()
    except redis.RedisError as exc:
        raise HTTPException(status_code=503, detail=f"redis unavailable: {exc}") from exc
    return {"status": "ok"}


@app.get("/api/v1/services")
def list_services() -> dict[str, list[str]]:
    services = sorted(rdb.smembers(SERVICES_KEY))
    return {"services": services}


@app.get("/api/v1/metrics")
def list_metrics() -> dict[str, Any]:
    return {
        "service_metrics": [
            {"id": key, "label": label} for key, label in SERVICE_METRICS.items()
        ],
        "edge_metrics": [
            {"id": key, "label": label} for key, label in EDGE_METRICS.items()
        ],
    }


@app.get("/api/v1/timeseries")
def timeseries(
    services: str = Query(..., description="Comma-separated service names"),
    metric: str = Query("outbound_rps", description="Metric id"),
    start: str = Query(..., description="ISO-8601 start time"),
    end: str = Query(..., description="ISO-8601 end time"),
) -> dict[str, Any]:
    if metric not in SERVICE_METRICS:
        raise HTTPException(status_code=400, detail=f"unsupported metric: {metric}")

    start_dt, end_dt = clamp_history_window(parse_time(start), parse_time(end))
    start_ms = to_ms(start_dt)
    end_ms = to_ms(end_dt)

    selected = [s.strip() for s in services.split(",") if s.strip()]
    if not selected:
        raise HTTPException(status_code=400, detail="at least one service is required")

    series: list[dict[str, Any]] = []
    for service in selected:
        key = service_series_key(service, metric)
        try:
            raw = rdb.execute_command("TS.RANGE", key, start_ms, end_ms)
        except redis.ResponseError as exc:
            if "key does not exist" in str(exc).lower():
                series.append({"service": service, "points": []})
                continue
            raise HTTPException(status_code=500, detail=str(exc)) from exc

        points = [[int(ts), float(val)] for ts, val in raw]
        series.append({"service": service, "points": points})

    return {
        "metric": metric,
        "start": start_dt.isoformat(),
        "end": end_dt.isoformat(),
        "series": series,
    }


@app.get("/api/v1/graph/snapshot")
def graph_snapshot(
    at: str = Query(..., description="ISO-8601 timestamp"),
) -> dict[str, Any]:
    target_ms = to_ms(parse_time(at))
    now_ms = to_ms(datetime.now(timezone.utc))
    earliest_ms = now_ms - int(RETENTION_HOURS * 3600 * 1000)

    if target_ms < earliest_ms or target_ms > now_ms:
        raise HTTPException(
            status_code=400,
            detail=f"timestamp must be within the last {RETENTION_HOURS} hours",
        )

    candidates = rdb.zrangebyscore(
        SNAPSHOTS_INDEX_KEY,
        target_ms - 60_000,
        target_ms + 60_000,
    )
    if not candidates:
        raise HTTPException(status_code=404, detail="no snapshot found for that time")

    nearest = min(candidates, key=lambda member: abs(int(member) - target_ms))
    payload = rdb.get(f"dhrishti:snapshot:{nearest}")
    if not payload:
        raise HTTPException(status_code=404, detail="snapshot payload missing")

    data = json.loads(payload)
    return {
        "requested_at": parse_time(at).isoformat(),
        "actual_timestamp": data.get("timestamp"),
        "snapshot_ms": int(nearest),
        "graph": data.get("graph", {}),
    }


@app.get("/api/v1/graph/range")
def graph_range(
    start: str = Query(...),
    end: str = Query(...),
) -> dict[str, Any]:
    start_dt, end_dt = clamp_history_window(parse_time(start), parse_time(end))
    members = rdb.zrangebyscore(
        SNAPSHOTS_INDEX_KEY,
        to_ms(start_dt),
        to_ms(end_dt),
    )
    timestamps = [
        datetime.fromtimestamp(int(member) / 1000, tz=timezone.utc).isoformat()
        for member in members
    ]
    return {
        "start": start_dt.isoformat(),
        "end": end_dt.isoformat(),
        "timestamps": timestamps,
        "count": len(timestamps),
    }


MAX_PLAYBACK_FRAMES = 500


@app.get("/api/v1/graph/playback")
def graph_playback(
    start: str = Query(...),
    end: str = Query(...),
) -> dict[str, Any]:
    """Return ordered graph snapshots in [start, end] for timeline replay."""
    start_dt, end_dt = clamp_history_window(parse_time(start), parse_time(end))
    members = rdb.zrangebyscore(
        SNAPSHOTS_INDEX_KEY,
        to_ms(start_dt),
        to_ms(end_dt),
    )

    if len(members) > MAX_PLAYBACK_FRAMES:
        step = max(1, len(members) // MAX_PLAYBACK_FRAMES)
        members = members[::step]

    frames: list[dict[str, Any]] = []
    for member in members:
        ms = int(member)
        payload = rdb.get(f"dhrishti:snapshot:{member}")
        if not payload:
            continue
        data = json.loads(payload)
        frames.append({
            "snapshot_ms": ms,
            "timestamp": data.get("timestamp"),
            "graph": data.get("graph", {}),
        })

    return {
        "start": start_dt.isoformat(),
        "end": end_dt.isoformat(),
        "count": len(frames),
        "frames": frames,
    }
