import React, { useCallback, useEffect, useState } from "react";

import {
  fetchGraphPlayback,
  fetchHistoryMetrics,
  fetchHistoryServices,
  fetchTimeseries,
} from "./api";
import GraphView from "./GraphView";
import TimeSeriesChart from "./TimeSeriesChart";
import { FONT, panel } from "./styles";

const FRAME_MS = 600;

function toLocalInputValue(date) {
  const d = new Date(date);
  d.setMinutes(d.getMinutes() - d.getTimezoneOffset());
  return d.toISOString().slice(0, 16);
}

function toISO(localValue) {
  return new Date(localValue).toISOString();
}

function defaultRange() {
  const end = new Date();
  const start = new Date(end.getTime() - 10 * 60 * 1000);
  return { start, end };
}

const labelStyle = {
  fontSize: 11,
  color: "#64748b",
  marginBottom: 4,
  display: "block",
};

const inputStyle = {
  fontFamily: FONT,
  fontSize: 12,
  padding: "6px 8px",
  border: "1px solid #e2e8f0",
  borderRadius: 6,
  color: "#1e293b",
  background: "#fff",
};

const btnStyle = {
  fontFamily: FONT,
  fontSize: 12,
  fontWeight: 500,
  padding: "7px 14px",
  border: "1px solid #cbd5e1",
  borderRadius: 6,
  background: "#f8fafc",
  color: "#1e293b",
  cursor: "pointer",
};

export default function TimelineView() {
  const initial = defaultRange();
  const [startInput, setStartInput] = useState(toLocalInputValue(initial.start));
  const [endInput, setEndInput] = useState(toLocalInputValue(initial.end));
  const [services, setServices] = useState([]);
  const [metrics, setMetrics] = useState([]);
  const [selectedServices, setSelectedServices] = useState([]);
  const [metric, setMetric] = useState("outbound_rps");
  const [series, setSeries] = useState([]);
  const [metricLabel, setMetricLabel] = useState("Outbound RPS");
  const [frames, setFrames] = useState([]);
  const [playIndex, setPlayIndex] = useState(0);
  const [playing, setPlaying] = useState(false);
  const [playbackSpeed, setPlaybackSpeed] = useState(1);
  const [scrubMs, setScrubMs] = useState(null);
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
    setScrubMs(frame.snapshot_ms);
  }, [frames]);

  const loadData = useCallback(async () => {
    if (!selectedServices.length) {
      setError("Select at least one service");
      return;
    }
    setLoading(true);
    setPlaying(false);
    setError(null);
    try {
      const start = toISO(startInput);
      const end = toISO(endInput);
      const [tsData, playback] = await Promise.all([
        fetchTimeseries({ services: selectedServices, metric, start, end }),
        fetchGraphPlayback({ start, end }),
      ]);

      setSeries(tsData.series ?? []);
      const meta = metrics.find((item) => item.id === metric);
      setMetricLabel(meta?.label ?? metric);

      const loadedFrames = (playback.frames ?? []).map((f, index) => ({ ...f, index }));
      setFrames(loadedFrames);

      if (loadedFrames.length > 0) {
        showFrame(0);
      } else {
        setHistoricalGraph({ nodes: [], edges: [], unknown_ips: [], entry_services: [] });
        setSnapshotTime(null);
        setScrubMs(null);
        setError("No graph snapshots in this range — ensure Dhrishti was running during that period");
      }
    } catch (err) {
      setError("Failed to load timeline data");
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, [startInput, endInput, selectedServices, metric, metrics, showFrame]);

  useEffect(() => {
    async function init() {
      try {
        const [svcList, metricData] = await Promise.all([
          fetchHistoryServices(),
          fetchHistoryMetrics(),
        ]);
        const metricList = metricData.service_metrics ?? [];
        setServices(svcList);
        setMetrics(metricList);

        const defaults = svcList.filter((s) =>
          ["api-gateway", "flash-sale-service", "inventory-service"].includes(s),
        );
        const selected = defaults.length ? defaults : svcList.slice(0, 3);
        setSelectedServices(selected);

        if (!selected.length) return;

        setLoading(true);
        const start = toISO(toLocalInputValue(defaultRange().start));
        const end = toISO(toLocalInputValue(new Date()));
        const [tsData, playback] = await Promise.all([
          fetchTimeseries({ services: selected, metric: "outbound_rps", start, end }),
          fetchGraphPlayback({ start, end }),
        ]);
        setSeries(tsData.series ?? []);
        const meta = metricList.find((item) => item.id === "outbound_rps");
        setMetricLabel(meta?.label ?? "outbound_rps");
        const loadedFrames = (playback.frames ?? []).map((f, index) => ({ ...f, index }));
        setFrames(loadedFrames);
        if (loadedFrames.length > 0) {
          const f = loadedFrames[0];
          setPlayIndex(0);
          setHistoricalGraph(f.graph ?? { nodes: [], edges: [], unknown_ips: [], entry_services: [] });
          setSnapshotTime(f.timestamp ?? new Date(f.snapshot_ms).toISOString());
          setScrubMs(f.snapshot_ms);
        }
        setLoading(false);
      } catch (err) {
        setError("History API unavailable — is make dhrishti running?");
        setLoading(false);
        console.error(err);
      }
    }
    init();
  }, []);

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
        setScrubMs(frame.snapshot_ms);
        return next;
      });
    }, FRAME_MS / playbackSpeed);

    return () => clearInterval(tick);
  }, [playing, frames, playbackSpeed]);

  const toggleService = (name) => {
    setSelectedServices((prev) =>
      prev.includes(name) ? prev.filter((s) => s !== name) : [...prev, name],
    );
  };

  const handleScrub = (ms) => {
    setPlaying(false);
    if (!frames.length) {
      setScrubMs(ms);
      return;
    }
    let best = 0;
    let bestDist = Infinity;
    frames.forEach((f, i) => {
      const dist = Math.abs(f.snapshot_ms - ms);
      if (dist < bestDist) {
        bestDist = dist;
        best = i;
      }
    });
    showFrame(best);
  };

  const handleChartHover = (ms) => {
    if (playing || !frames.length) return;
    let best = 0;
    let bestDist = Infinity;
    frames.forEach((f, i) => {
      const dist = Math.abs(f.snapshot_ms - ms);
      if (dist < bestDist) {
        bestDist = dist;
        best = i;
      }
    });
    if (best !== playIndex) {
      showFrame(best);
    }
  };

  const nearestFrameMs = frames.length
    ? frames[Math.min(playIndex, frames.length - 1)].snapshot_ms
    : scrubMs;

  return (
    <div
      style={{
        flex: 1,
        display: "flex",
        flexDirection: "column",
        minHeight: 0,
        overflow: "auto",
        padding: 16,
        gap: 14,
      }}
    >
      <div style={{ ...panel, padding: 14 }}>
        <div
          style={{
            display: "flex",
            flexWrap: "wrap",
            gap: 12,
            alignItems: "flex-end",
            marginBottom: 14,
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
          <button type="button" onClick={loadData} disabled={loading} style={{ ...btnStyle, cursor: loading ? "wait" : "pointer" }}>
            {loading ? "Loading…" : "Load"}
          </button>
          {error && (
            <span style={{ fontSize: 11, color: "#dc2626" }}>{error}</span>
          )}
        </div>

        <div style={{ marginBottom: 12 }}>
          <span style={labelStyle}>Metric</span>
          <div style={{ display: "flex", flexWrap: "wrap", gap: 10 }}>
            {metrics.map((m) => (
              <label
                key={m.id}
                style={{ fontSize: 12, color: "#475569", display: "flex", alignItems: "center", gap: 5, cursor: "pointer" }}
              >
                <input
                  type="radio"
                  name="metric"
                  checked={metric === m.id}
                  onChange={() => setMetric(m.id)}
                />
                {m.label}
              </label>
            ))}
          </div>
        </div>

        <div>
          <span style={labelStyle}>Services (compare)</span>
          <div
            style={{
              display: "flex",
              flexWrap: "wrap",
              gap: 8,
              maxHeight: 72,
              overflow: "auto",
            }}
          >
            {services.length === 0 ? (
              <span style={{ fontSize: 11, color: "#94a3b8" }}>No services in history yet</span>
            ) : (
              services.map((name) => (
                <label
                  key={name}
                  style={{
                    fontSize: 11,
                    color: selectedServices.includes(name) ? "#1e293b" : "#94a3b8",
                    display: "flex",
                    alignItems: "center",
                    gap: 4,
                    cursor: "pointer",
                    padding: "3px 8px",
                    border: "1px solid #e2e8f0",
                    borderRadius: 6,
                    background: selectedServices.includes(name) ? "#f8fafc" : "#fff",
                  }}
                >
                  <input
                    type="checkbox"
                    checked={selectedServices.includes(name)}
                    onChange={() => toggleService(name)}
                  />
                  {name}
                </label>
              ))
            )}
          </div>
        </div>
      </div>

      <TimeSeriesChart
        series={series}
        metric={metric}
        metricLabel={metricLabel}
        scrubMs={nearestFrameMs}
        onScrub={handleScrub}
        onHoverMs={handleChartHover}
      />

      <div style={{ ...panel, flex: 1, minHeight: 360, display: "flex", flexDirection: "column" }}>
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
          }}
        >
          <span>
            Replay{" "}
            <span style={{ color: "#1e293b", fontWeight: 500 }}>
              {snapshotTime ? new Date(snapshotTime).toLocaleString() : "—"}
            </span>
          </span>
          {frames.length > 0 && (
            <span style={{ color: "#94a3b8" }}>
              frame {playIndex + 1} / {frames.length}
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
            style={{ width: "calc(100% - 28px)", margin: "8px 14px 0", accentColor: "#3b82f6" }}
            aria-label="Replay position"
          />
        )}
        <div style={{ flex: 1, minHeight: 320, position: "relative" }}>
          <GraphView graph={historicalGraph} showLegend={false} />
        </div>
      </div>
    </div>
  );
}
