import React from "react";

import { FONT } from "./styles";

const MIN_ZOOM = 0.2;
const MAX_ZOOM = 2.5;

export function sliderToZoom(value) {
  return MIN_ZOOM + (value / 100) * (MAX_ZOOM - MIN_ZOOM);
}

export function zoomToSlider(level) {
  const clamped = Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, level));
  return ((clamped - MIN_ZOOM) / (MAX_ZOOM - MIN_ZOOM)) * 100;
}

export default function ZoomSlider({ value, onChange }) {
  return (
    <div
      style={{
        position: "absolute",
        right: 14,
        top: "50%",
        transform: "translateY(-50%)",
        zIndex: 10,
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        gap: 8,
        fontFamily: FONT,
        userSelect: "none",
      }}
    >
      <span style={{ fontSize: 14, color: "#64748b", lineHeight: 1 }} title="Zoom in">
        +
      </span>
      <div
        style={{
          height: 140,
          width: 28,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          background: "rgba(255, 255, 255, 0.94)",
          border: "1px solid #e2e8f0",
          borderRadius: 8,
          boxShadow: "0 2px 8px rgba(15, 23, 42, 0.06)",
          padding: "6px 0",
        }}
      >
        <input
          type="range"
          min={0}
          max={100}
          value={value}
          onChange={(e) => onChange(Number(e.target.value))}
          orient="vertical"
          aria-label="Graph zoom"
          style={{
            width: 120,
            height: 20,
            transform: "rotate(-90deg)",
            cursor: "pointer",
            accentColor: "#3b82f6",
          }}
        />
      </div>
      <span style={{ fontSize: 14, color: "#64748b", lineHeight: 1 }} title="Zoom out">
        −
      </span>
    </div>
  );
}
