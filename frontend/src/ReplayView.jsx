import React, { useCallback, useEffect, useState } from "react";

import { fetchGraphPlayback } from "./api";
import GraphView from "./GraphView";
import {
  btnStyle,
  defaultRange,
  inputStyle,
  labelStyle,
  toISO,
  toLocalInputValue,
} from "./historyUi";
import { panel } from "./styles";

const FRAME_MS = 600;

export default function ReplayView() {
  const initial = defaultRange();
  const [startInput, setStartInput] = useState(toLocalInputValue(initial.start));
  const [endInput, setEndInput] = useState(toLocalInputValue(initial.end));
  const [frames, setFrames] = useState([]);
  const [playIndex, setPlayIndex] = useState(0);
  const [playing, setPlaying] = useState(false);
  const [playbackSpeed, setPlaybackSpeed] = useState(1);
  const [historicalGraph, setHistoricalGraph] = useState({
    nodes: [],
    edges: [],
    unknown_ips: [],
    entry_services: [],
  });
  const [snapshotTime, setSnapshotTime] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const showFrame = useCallback((index) => {
    const frame = frames[index];
    if (!frame) return;
    setPlayIndex(index);
    setHistoricalGraph(frame.graph ?? { nodes: [], edges: [], unknown_ips: [], entry_services: [] });
    setSnapshotTime(frame.timestamp ?? new Date(frame.snapshot_ms).toISOString());
  }, [frames]);

  const loadData = useCallback(async () => {
    setLoading(true);
    setPlaying(false);
    setError(null);
    try {
      const start = toISO(startInput);
      const end = toISO(endInput);
      const playback = await fetchGraphPlayback({ start, end });
      const loadedFrames = (playback.frames ?? []).map((f, index) => ({ ...f, index }));
      setFrames(loadedFrames);

      if (loadedFrames.length > 0) {
        showFrame(0);
      } else {
        setHistoricalGraph({ nodes: [], edges: [], unknown_ips: [], entry_services: [] });
        setSnapshotTime(null);
        setError("No graph snapshots in this range — ensure Dhrishti was running during that period");
      }
    } catch (err) {
      setError("Failed to load replay data — is make dhrishti running?");
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, [startInput, endInput, showFrame]);

  useEffect(() => {
    loadData();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (!playing || frames.length === 0) return undefined;

    const tick = setInterval(() => {
      setPlayIndex((idx) => {
        const next = idx + 1;
        if (next >= frames.length) {
          setPlaying(false);
          return idx;
        }
        const frame = frames[next];
        setHistoricalGraph(frame.graph ?? { nodes: [], edges: [], unknown_ips: [], entry_services: [] });
        setSnapshotTime(frame.timestamp ?? new Date(frame.snapshot_ms).toISOString());
        return next;
      });
    }, FRAME_MS / playbackSpeed);

    return () => clearInterval(tick);
  }, [playing, frames, playbackSpeed]);

  return (
    <div
      style={{
        flex: 1,
        display: "flex",
        flexDirection: "column",
        minHeight: 0,
        overflow: "hidden",
        padding: 16,
        gap: 14,
      }}
    >
      <div style={{ ...panel, padding: 14, flexShrink: 0 }}>
        <div
          style={{
            display: "flex",
            flexWrap: "wrap",
            gap: 12,
            alignItems: "flex-end",
          }}
        >
          <label>
            <span style={labelStyle}>Start</span>
            <input
              type="datetime-local"
              value={startInput}
              onChange={(e) => setStartInput(e.target.value)}
              style={inputStyle}
            />
          </label>
          <label>
            <span style={labelStyle}>End</span>
            <input
              type="datetime-local"
              value={endInput}
              onChange={(e) => setEndInput(e.target.value)}
              style={inputStyle}
            />
          </label>
          <button
            type="button"
            onClick={loadData}
            disabled={loading}
            style={{ ...btnStyle, cursor: loading ? "wait" : "pointer" }}
          >
            {loading ? "Loading…" : "Load"}
          </button>
          {error && (
            <span style={{ fontSize: 11, color: "#dc2626" }}>{error}</span>
          )}
        </div>
      </div>

      <div style={{ ...panel, flex: 1, minHeight: 0, display: "flex", flexDirection: "column" }}>
        <div
          style={{
            padding: "10px 14px",
            borderBottom: "1px solid #e2e8f0",
            display: "flex",
            flexWrap: "wrap",
            alignItems: "center",
            gap: 10,
            fontSize: 12,
            color: "#64748b",
            flexShrink: 0,
          }}
        >
          <span>
            Frame{" "}
            <span style={{ color: "#1e293b", fontWeight: 500 }}>
              {snapshotTime ? new Date(snapshotTime).toLocaleString() : "—"}
            </span>
          </span>
          {frames.length > 0 && (
            <span style={{ color: "#94a3b8" }}>
              {playIndex + 1} / {frames.length}
            </span>
          )}
          <div style={{ marginLeft: "auto", display: "flex", gap: 8, alignItems: "center" }}>
            <button
              type="button"
              disabled={!frames.length}
              onClick={() => setPlaying((p) => !p)}
              style={{ ...btnStyle, opacity: frames.length ? 1 : 0.5 }}
            >
              {playing ? "Pause" : "Play"}
            </button>
            {[1, 2, 4].map((speed) => (
              <button
                key={speed}
                type="button"
                onClick={() => setPlaybackSpeed(speed)}
                style={{
                  ...btnStyle,
                  fontWeight: playbackSpeed === speed ? 600 : 400,
                  background: playbackSpeed === speed ? "#eff6ff" : "#f8fafc",
                  borderColor: playbackSpeed === speed ? "#93c5fd" : "#cbd5e1",
                }}
              >
                {speed}x
              </button>
            ))}
          </div>
        </div>
        {frames.length > 1 && (
          <input
            type="range"
            min={0}
            max={frames.length - 1}
            value={playIndex}
            onChange={(e) => {
              setPlaying(false);
              showFrame(Number(e.target.value));
            }}
            style={{ width: "calc(100% - 28px)", margin: "8px 14px 0", accentColor: "#3b82f6", flexShrink: 0 }}
            aria-label="Replay position"
          />
        )}
        <div style={{ flex: 1, minHeight: 0, position: "relative" }}>
          <GraphView graph={historicalGraph} showLegend={false} />
        </div>
      </div>
    </div>
  );
}
