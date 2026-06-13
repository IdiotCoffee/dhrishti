import React, { useCallback, useEffect, useState } from "react";

import {
  fetchHistoryMetrics,
  fetchHistoryServices,
  fetchTimeseries,
} from "./api";
import TimeSeriesChart from "./TimeSeriesChart";
import {
  btnStyle,
  defaultRange,
  inputStyle,
  labelStyle,
  toISO,
  toLocalInputValue,
} from "./historyUi";
import { panel } from "./styles";

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
  const [scrubMs, setScrubMs] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const loadData = useCallback(async () => {
    if (!selectedServices.length) {
      setError("Select at least one service");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const start = toISO(startInput);
      const end = toISO(endInput);
      const tsData = await fetchTimeseries({
        services: selectedServices,
        metric,
        start,
        end,
      });

      setSeries(tsData.series ?? []);
      const meta = metrics.find((item) => item.id === metric);
      setMetricLabel(meta?.label ?? metric);

      const firstPoint = tsData.series?.flatMap((s) => s.points ?? [])[0];
      setScrubMs(firstPoint ? firstPoint[0] : null);
    } catch (err) {
      setError("Failed to load timeline data");
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, [startInput, endInput, selectedServices, metric, metrics]);

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
        const tsData = await fetchTimeseries({
          services: selected,
          metric: "outbound_rps",
          start,
          end,
        });
        setSeries(tsData.series ?? []);
        const meta = metricList.find((item) => item.id === "outbound_rps");
        setMetricLabel(meta?.label ?? "outbound_rps");
        const firstPoint = tsData.series?.flatMap((s) => s.points ?? [])[0];
        setScrubMs(firstPoint ? firstPoint[0] : null);
        setLoading(false);
      } catch (err) {
        setError("History API unavailable — is make dhrishti running?");
        setLoading(false);
        console.error(err);
      }
    }
    init();
  }, []);

  const toggleService = (name) => {
    setSelectedServices((prev) =>
      prev.includes(name) ? prev.filter((s) => s !== name) : [...prev, name],
    );
  };

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
        scrubMs={scrubMs}
        onScrub={setScrubMs}
        onHoverMs={setScrubMs}
      />
    </div>
  );
}
