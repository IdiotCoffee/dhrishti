import React, { useMemo, useRef, useState } from "react";

import { FONT, chartColors, panel } from "./styles";

const W = 900;
const H = 300;
const PAD = { top: 20, right: 20, bottom: 34, left: 58 };

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

function nearestLinePoint(series, plot, svgX, svgY) {
  let best = null;

  series.forEach((item, seriesIndex) => {
    const color = chartColors[seriesIndex % chartColors.length];
    const points = item.points ?? [];
    const segments = points.length === 1 ? [[points[0], points[0]]] : points.slice(1).map((point, index) => [points[index], point]);

    segments.forEach(([a, b]) => {
      const ax = plot.xScale(a[0]);
      const ay = plot.yScale(a[1]);
      const bx = plot.xScale(b[0]);
      const by = plot.yScale(b[1]);
      const dx = bx - ax;
      const dy = by - ay;
      const lengthSquared = dx * dx + dy * dy;
      const t = lengthSquared === 0
        ? 0
        : Math.max(0, Math.min(1, ((svgX - ax) * dx + (svgY - ay) * dy) / lengthSquared));
      const x = ax + t * dx;
      const y = ay + t * dy;
      const distance = Math.hypot(svgX - x, svgY - y);

      if (!best || distance < best.distance) {
        best = {
          distance,
          service: item.service,
          value: a[1] + t * (b[1] - a[1]),
          ms: a[0] + t * (b[0] - a[0]),
          color,
          x,
          y,
        };
      }
    });
  });

  return best;
}

export default function TimeSeriesChart({ series, metric, metricLabel }) {
  const svgRef = useRef(null);
  const [hover, setHover] = useState(null);

  const plot = useMemo(() => {
    const allPoints = series.flatMap((item) => item.points ?? []);
    if (!allPoints.length) return null;

    const minX = Math.min(...allPoints.map((point) => point[0]));
    const maxX = Math.max(...allPoints.map((point) => point[0]));
    const maxY = Math.max(...allPoints.map((point) => point[1]), 0.001);
    const innerW = W - PAD.left - PAD.right;
    const innerH = H - PAD.top - PAD.bottom;
    const baseline = PAD.top + innerH;
    const xScale = (x) => PAD.left + ((x - minX) / (maxX - minX || 1)) * innerW;
    const yScale = (y) => PAD.top + innerH - (y / maxY) * innerH;

    const paths = series.map((item, index) => {
      const points = item.points ?? [];
      if (!points.length) return null;
      const linePath = points
        .map((point, pointIndex) => `${pointIndex === 0 ? "M" : "L"} ${xScale(point[0]).toFixed(2)} ${yScale(point[1]).toFixed(2)}`)
        .join(" ");
      const first = points[0];
      const last = points[points.length - 1];
      return {
        service: item.service,
        points,
        color: chartColors[index % chartColors.length],
        gradientId: `series-area-${index}`,
        linePath,
        areaPath: `${linePath} L ${xScale(last[0]).toFixed(2)} ${baseline} L ${xScale(first[0]).toFixed(2)} ${baseline} Z`,
      };
    }).filter(Boolean);

    return { paths, minX, maxX, maxY, xScale, yScale, innerW, innerH, baseline };
  }, [series]);

  const handleMouseMove = (event) => {
    if (!plot || !svgRef.current) return;
    const rect = svgRef.current.getBoundingClientRect();
    const svgX = ((event.clientX - rect.left) / rect.width) * W;
    const svgY = ((event.clientY - rect.top) / rect.height) * H;
    if (svgX < PAD.left || svgX > PAD.left + plot.innerW || svgY < PAD.top || svgY > plot.baseline) {
      setHover(null);
      return;
    }

    const point = nearestLinePoint(series, plot, svgX, svgY);
    if (!point) return;
    setHover(point);
  };

  if (!plot) {
    return (
      <div style={{ ...panel, padding: 24, fontFamily: FONT, fontSize: 12, color: "#94a3b8", textAlign: "center" }}>
        No data in this range — run a simulation while Dhrishti is up
      </div>
    );
  }

  return (
    <div style={{ ...panel, padding: "14px 16px", fontFamily: FONT, position: "relative" }}>
      <div style={{ fontSize: 12, color: "#475569", marginBottom: 8, fontWeight: 500 }}>
        {metricLabel || metric}
      </div>
      <svg
        ref={svgRef}
        viewBox={`0 0 ${W} ${H}`}
        style={{ width: "100%", height: "auto", display: "block", cursor: "crosshair", overflow: "visible" }}
        role="img"
        aria-label="Time series chart"
        onMouseMove={handleMouseMove}
        onMouseLeave={() => setHover(null)}
      >
        <defs>
          {plot.paths.map((path) => (
            <linearGradient key={path.gradientId} id={path.gradientId} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={path.color} stopOpacity="0.16" />
              <stop offset="100%" stopColor={path.color} stopOpacity="0.005" />
            </linearGradient>
          ))}
        </defs>

        {[0, 0.25, 0.5, 0.75, 1].map((fraction) => {
          const y = PAD.top + plot.innerH * fraction;
          const value = plot.maxY * (1 - fraction);
          return (
            <g key={fraction}>
              <line x1={PAD.left} y1={y} x2={PAD.left + plot.innerW} y2={y} stroke="#eef2f7" strokeWidth="1" />
              <text x={PAD.left - 10} y={y + 3} fontSize="10" fill="#94a3b8" textAnchor="end">
                {formatValue(metric, value)}
              </text>
            </g>
          );
        })}

        {plot.paths.map((path) => (
          <path key={`${path.service}-area`} d={path.areaPath} fill={`url(#${path.gradientId})`} stroke="none" />
        ))}
        {plot.paths.map((path) => (
          <path
            key={path.service}
            d={path.linePath}
            fill="none"
            stroke={path.color}
            strokeWidth="1.7"
            strokeLinejoin="round"
            strokeLinecap="round"
            vectorEffect="non-scaling-stroke"
          />
        ))}

        <text x={PAD.left} y={H - 8} fontSize="10" fill="#94a3b8">{formatShortTime(plot.minX)}</text>
        <text x={PAD.left + plot.innerW} y={H - 8} fontSize="10" fill="#94a3b8" textAnchor="end">
          {formatShortTime(plot.maxX)}
        </text>

        {hover && (
          <>
            <line x1={hover.x} y1={PAD.top} x2={hover.x} y2={plot.baseline} stroke="#94a3b8" strokeWidth="1" strokeDasharray="3 4" />
            <circle cx={hover.x} cy={hover.y} r="4" fill={hover.color} stroke="#fff" strokeWidth="2" />
          </>
        )}
      </svg>

      {hover && (
        <div
          style={{
            position: "absolute",
            left: 18,
            top: 40,
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

      <div style={{ display: "flex", flexWrap: "wrap", gap: 14, marginTop: 10 }}>
        {plot.paths.map((path) => (
          <span key={path.service} style={{ fontSize: 11, color: "#475569", display: "flex", alignItems: "center", gap: 6 }}>
            <span style={{ width: 14, height: 2, background: path.color, borderRadius: 1 }} />
            {path.service}
          </span>
        ))}
      </div>
    </div>
  );
}
