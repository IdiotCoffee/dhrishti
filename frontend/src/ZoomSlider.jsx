import React, { useCallback, useRef } from "react";

import { FONT } from "./styles";

export const MIN_ZOOM = 0.2;
export const MAX_ZOOM = 2.5;

export function sliderToZoom(value) {
  return MIN_ZOOM + (value / 100) * (MAX_ZOOM - MIN_ZOOM);
}

export function zoomToSlider(level) {
  const clamped = Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, level));
  return ((clamped - MIN_ZOOM) / (MAX_ZOOM - MIN_ZOOM)) * 100;
}

export default function ZoomSlider({ value, onChange }) {
  const trackRef = useRef(null);

  const valueFromClientY = useCallback((clientY) => {
    const track = trackRef.current;
    if (!track) return value;
    const rect = track.getBoundingClientRect();
    const ratio = 1 - (clientY - rect.top) / rect.height;
    return Math.round(Math.max(0, Math.min(1, ratio)) * 100);
  }, [value]);

  const handlePointer = (evt) => {
    onChange(valueFromClientY(evt.clientY));
  };

  const nudge = (delta) => {
    onChange(Math.max(0, Math.min(100, value + delta)));
  };

  const thumbBottom = `${value}%`;

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
        gap: 6,
        fontFamily: FONT,
        userSelect: "none",
      }}
    >
      <button
        type="button"
        onClick={() => nudge(8)}
        aria-label="Zoom in"
        style={{
          width: 28,
          height: 28,
          border: "1px solid #e2e8f0",
          borderRadius: 6,
          background: "rgba(255, 255, 255, 0.94)",
          color: "#64748b",
          cursor: "pointer",
          fontSize: 16,
          lineHeight: 1,
        }}
      >
        +
      </button>
      <div
        ref={trackRef}
        role="slider"
        aria-label="Graph zoom"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={value}
        tabIndex={0}
        onPointerDown={(evt) => {
          evt.currentTarget.setPointerCapture(evt.pointerId);
          handlePointer(evt);
        }}
        onPointerMove={(evt) => {
          if (evt.buttons !== 1) return;
          handlePointer(evt);
        }}
        onKeyDown={(evt) => {
          if (evt.key === "ArrowUp") {
            evt.preventDefault();
            nudge(4);
          } else if (evt.key === "ArrowDown") {
            evt.preventDefault();
            nudge(-4);
          }
        }}
        style={{
          position: "relative",
          height: 140,
          width: 28,
          background: "rgba(255, 255, 255, 0.94)",
          border: "1px solid #e2e8f0",
          borderRadius: 8,
          boxShadow: "0 2px 8px rgba(15, 23, 42, 0.06)",
          cursor: "pointer",
          touchAction: "none",
        }}
      >
        <div
          style={{
            position: "absolute",
            left: "50%",
            top: 8,
            bottom: 8,
            width: 4,
            marginLeft: -2,
            background: "#e2e8f0",
            borderRadius: 2,
          }}
        />
        <div
          style={{
            position: "absolute",
            left: "50%",
            bottom: thumbBottom,
            width: 16,
            height: 16,
            marginLeft: -8,
            marginBottom: -8,
            borderRadius: "50%",
            background: "#3b82f6",
            border: "2px solid #fff",
            boxShadow: "0 1px 4px rgba(15, 23, 42, 0.2)",
            pointerEvents: "none",
          }}
        />
      </div>
      <button
        type="button"
        onClick={() => nudge(-8)}
        aria-label="Zoom out"
        style={{
          width: 28,
          height: 28,
          border: "1px solid #e2e8f0",
          borderRadius: 6,
          background: "rgba(255, 255, 255, 0.94)",
          color: "#64748b",
          cursor: "pointer",
          fontSize: 16,
          lineHeight: 1,
        }}
      >
        −
      </button>
    </div>
  );
}
