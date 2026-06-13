import React, { useMemo, useRef, useState } from "react";

import { FONT, chartColors, panel } from "./styles";

const W = 720;
const H = 200;
const PAD = { top: 16, right: 16, bottom: 28, left: 48 };

function formatTime(ms) {
  return new Date(ms).toLocaleString([], {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function formatShortTime(ms) {
  return new Date(ms).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function formatValue(metric, value) {
  if (metric === "failure_rate") return `${(value * 100).toFixed(1)}%`;
  if (metric.includes("latency")) return `${value.toFixed(0)} ms`;
  if (Number.isInteger(value)) return String(value);
  return value < 10 ? value.toFixed(2) : value.toFixed(1);
}

function nearestPoint(series, plot, svgX) {
  const targetMs = plot.minX + ((svgX - PAD.left) / plot.innerW) * (plot.maxX - plot.minX);
  let best = null;

  series.forEach((s, seriesIdx) => {
    s.points.forEach((p) => {
      const dist = Math.abs(p[0] - targetMs);
      if (!best || dist < best.dist) {
        best = {
          dist,
          service: s.service,
          value: p[1],
          ms: p[0],
          color: chartColors[seriesIdx % chartColors.length],
          x: plot.xScale(p[0]),
          y: plot.yScale(p[1]),
        };
      }
    });
  });

  return best;
}

export default function TimeSeriesChart({
  series,
  metric,
  metricLabel,
  scrubMs,
  onScrub,
  onHoverMs,
}) {
  const svgRef = useRef(null);
  const [hover, setHover] = useState(null);

  const plot = useMemo(() => {
    const allPoints = series.flatMap((s) => s.points);
    if (!allPoints.length) return null;

    const xs = allPoints.map((p) => p[0]);
    const ys = allPoints.map((p) => p[1]);
    const minX = Math.min(...xs);
    const maxX = Math.max(...xs);
    const maxY = Math.max(...ys, 0.001);

    const innerW = W - PAD.left - PAD.right;
    const innerH = H - PAD.top - PAD.bottom;

    const xScale = (x) => PAD.left + ((x - minX) / (maxX - minX || 1)) * innerW;
    const yScale = (y) => PAD.top + innerH - (y / maxY) * innerH;

    const paths = series.map((s, i) => {
      if (!s.points.length) return null;
      const d = s.points
        .map((p, idx) => `${idx === 0 ? "M" : "L"} ${xScale(p[0]).toFixed(1)} ${yScale(p[1]).toFixed(1)}`)
        .join(" ");
      return { service: s.service, d, color: chartColors[i % chartColors.length], points: s.points };
    }).filter(Boolean);

    return { paths, minX, maxX, maxY, xScale, yScale, innerW, innerH };
  }, [series]);

  const handleMouseMove = (evt) => {
    if (!plot || !svgRef.current) return;
    const rect = svgRef.current.getBoundingClientRect();
    const svgX = ((evt.clientX - rect.left) / rect.width) * W;
    if (svgX < PAD.left || svgX > PAD.left + plot.innerW) {
      setHover(null);
      return;
    }

    const point = nearestPoint(series, plot, svgX);
    if (!point) return;

    setHover(point);
    if (onHoverMs) onHoverMs(point.ms);
  };

  const handleMouseLeave = () => {
    setHover(null);
  };

  if (!plot) {
    return (
      <div
        style={{
          ...panel,
          padding: 24,
          fontFamily: FONT,
          fontSize: 12,
          color: "#94a3b8",
          textAlign: "center",
        }}
      >
        No data in this range — run a simulation while Dhrishti is up
      </div>
    );
  }

  const scrubX = scrubMs != null ? plot.xScale(scrubMs) : null;

  return (
    <div style={{ ...panel, padding: "12px 14px", fontFamily: FONT, position: "relative" }}>
      <div style={{ fontSize: 11, color: "#64748b", marginBottom: 8 }}>
        {metricLabel || metric}
      </div>
      <svg
        ref={svgRef}
        viewBox={`0 0 ${W} ${H}`}
        style={{ width: "100%", height: "auto", display: "block", cursor: "crosshair" }}
        role="img"
        aria-label="Time series chart"
        onMouseMove={handleMouseMove}
        onMouseLeave={handleMouseLeave}
      >
        <line
          x1={PAD.left}
          y1={PAD.top + plot.innerH}
          x2={PAD.left + plot.innerW}
          y2={PAD.top + plot.innerH}
          stroke="#e2e8f0"
        />
        <line
          x1={PAD.left}
          y1={PAD.top}
          x2={PAD.left}
          y2={PAD.top + plot.innerH}
          stroke="#e2e8f0"
        />
        <text x={PAD.left} y={H - 6} fontSize={10} fill="#94a3b8">
          {formatShortTime(plot.minX)}
        </text>
        <text x={PAD.left + plot.innerW} y={H - 6} fontSize={10} fill="#94a3b8" textAnchor="end">
          {formatShortTime(plot.maxX)}
        </text>
        <text x={PAD.left - 6} y={PAD.top + 4} fontSize={10} fill="#94a3b8" textAnchor="end">
          {formatValue(metric, plot.maxY)}
        </text>
        {plot.paths.map((p) => (
          <path
            key={p.service}
            d={p.d}
            fill="none"
            stroke={p.color}
            strokeWidth={2}
            strokeLinejoin="round"
          />
        ))}
        {hover && (
          <>
            <line
              x1={hover.x}
              y1={PAD.top}
              x2={hover.x}
              y2={PAD.top + plot.innerH}
              stroke="#94a3b8"
              strokeWidth={1}
              strokeDasharray="3 3"
            />
            <circle cx={hover.x} cy={hover.y} r={4} fill={hover.color} stroke="#fff" strokeWidth={1.5} />
          </>
        )}
        {scrubX != null && scrubX >= PAD.left && scrubX <= PAD.left + plot.innerW && (
          <line
            x1={scrubX}
            y1={PAD.top}
            x2={scrubX}
            y2={PAD.top + plot.innerH}
            stroke="#1e293b"
            strokeWidth={1.5}
            opacity={0.6}
          />
        )}
      </svg>

      {hover && (
        <div
          style={{
            position: "absolute",
            left: 14,
            top: 36,
            fontSize: 11,
            lineHeight: 1.5,
            color: "#334155",
            background: "rgba(255, 255, 255, 0.97)",
            border: "1px solid #e2e8f0",
            borderRadius: 6,
            padding: "8px 10px",
            boxShadow: "0 4px 12px rgba(15, 23, 42, 0.08)",
            pointerEvents: "none",
          }}
        >
          <div style={{ fontWeight: 600, color: hover.color }}>{hover.service}</div>
          <div>{metricLabel || metric}: <strong>{formatValue(metric, hover.value)}</strong></div>
          <div style={{ color: "#64748b" }}>{formatTime(hover.ms)}</div>
        </div>
      )}

      <div style={{ display: "flex", flexWrap: "wrap", gap: 12, marginTop: 8 }}>
        {plot.paths.map((p) => (
          <span key={p.service} style={{ fontSize: 11, color: "#475569", display: "flex", alignItems: "center", gap: 6 }}>
            <span style={{ width: 10, height: 2, background: p.color, borderRadius: 1 }} />
            {p.service}
          </span>
        ))}
      </div>
      {onScrub && (
        <input
          type="range"
          min={plot.minX}
          max={plot.maxX}
          value={scrubMs ?? plot.minX}
          onChange={(e) => onScrub(Number(e.target.value))}
          style={{ width: "100%", marginTop: 10, accentColor: "#3b82f6" }}
          aria-label="Timeline position"
        />
      )}
    </div>
  );
}
