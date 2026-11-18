import React, { useCallback, useEffect, useRef, useState } from "react";

import {
  fetchHistoryMetrics,
  fetchHistoryServices,
  fetchTimeseries,
} from "./api";
import TimeSeriesChart from "./TimeSeriesChart";
import {
  defaultRange,
  inputStyle,
  labelStyle,
  toLocalInputValue,
} from "./historyUi";
import { panel } from "./styles";

const REFRESH_MS = 5000;

export default function TimelineView() {
  const initial = defaultRange();
  const [startInput, setStartInput] = useState(toLocalInputValue(initial.start));
  const [endInput, setEndInput] = useState(toLocalInputValue(initial.end));
  const [followLive, setFollowLive] = useState(true);
  const [services, setServices] = useState([]);
  const [metrics, setMetrics] = useState([]);
  const [selectedServices, setSelectedServices] = useState([]);
  const [metric, setMetric] = useState("outbound_rps");
  const [series, setSeries] = useState([]);
  const [metricLabel, setMetricLabel] = useState("Outbound RPS");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [lastUpdated, setLastUpdated] = useState(null);
  const [ready, setReady] = useState(false);
  const requestSequence = useRef(0);

  const loadData = useCallback(async (advanceLiveEnd = false) => {
    const requestId = ++requestSequence.current;
    if (!selectedServices.length || !startInput || (!followLive && !endInput)) {
      setLoading(false);
      setSeries([]);
      setError(selectedServices.length ? "Choose a valid time range" : null);
      return;
    }

    const startDate = new Date(startInput);
    const endDate = followLive && advanceLiveEnd ? new Date() : new Date(endInput);
    if (Number.isNaN(startDate.getTime()) || Number.isNaN(endDate.getTime()) || startDate >= endDate) {
      setLoading(false);
      setError("Choose a valid time range");
      return;
    }

    setLoading(true);
    setError(null);
    try {
      const tsData = await fetchTimeseries({
        services: selectedServices,
        metric,
        start: startDate.toISOString(),
        end: endDate.toISOString(),
      });

      if (requestId !== requestSequence.current) return;
      setSeries(tsData.series ?? []);
      const meta = metrics.find((item) => item.id === metric);
      setMetricLabel(meta?.label ?? metric);
      setLastUpdated(new Date());

      if (followLive && advanceLiveEnd) {
        const currentInput = toLocalInputValue(new Date());
        setEndInput((previous) => previous === currentInput ? previous : currentInput);
      }
    } catch (err) {
      if (requestId === requestSequence.current) {
        setError("Could not load timeline data. Check that Dhrishti and Redis are running.");
      }
      console.error(err);
    } finally {
      if (requestId === requestSequence.current) setLoading(false);
    }
  }, [endInput, followLive, metric, metrics, selectedServices, startInput]);

  useEffect(() => {
    let cancelled = false;

    async function init() {
      try {
        const [svcList, metricData] = await Promise.all([
          fetchHistoryServices(),
          fetchHistoryMetrics(),
        ]);
        if (cancelled) return;

        const metricList = metricData.service_metrics ?? [];
        setServices(svcList);
        setMetrics(metricList);

        const defaults = svcList.filter((name) =>
          ["api-gateway", "flash-sale-service", "inventory-service"].includes(name),
        );
        setSelectedServices(defaults.length ? defaults : svcList.slice(0, 3));
        setReady(true);
      } catch (err) {
        if (!cancelled) {
          setError("History API unavailable — is make dhrishti running?");
          console.error(err);
        }
      }
    }

    init();
    return () => { cancelled = true; };
  }, []);

  // Fetch whenever a filter changes; debounce checkbox clicks so multi-select
  // updates do not fire a request for every rapid click.
  useEffect(() => {
    if (!ready) return undefined;
    const timeout = setTimeout(() => loadData(), 180);
    return () => clearTimeout(timeout);
  }, [ready, startInput, endInput, selectedServices, metric, loadData]);

  // Keep the chart current while following the live end of the time range.
  useEffect(() => {
    if (!ready || !followLive) return undefined;
    const interval = setInterval(() => loadData(true), REFRESH_MS);
    return () => clearInterval(interval);
  }, [ready, followLive, loadData]);

  const toggleService = (name) => {
    setSelectedServices((prev) =>
      prev.includes(name) ? prev.filter((service) => service !== name) : [...prev, name],
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
        <div style={{ display: "flex", flexWrap: "wrap", gap: 12, alignItems: "flex-end", marginBottom: 14 }}>
          <label>
            <span style={labelStyle}>Start</span>
            <input
              type="datetime-local"
              value={startInput}
              onChange={(event) => setStartInput(event.target.value)}
              style={inputStyle}
            />
          </label>
          <label>
            <span style={labelStyle}>End</span>
            <input
              type="datetime-local"
              value={endInput}
              onChange={(event) => {
                setEndInput(event.target.value);
                setFollowLive(false);
              }}
              style={inputStyle}
            />
          </label>
          {!followLive && (
            <button
              type="button"
              onClick={() => {
                setEndInput(toLocalInputValue(new Date()));
                setFollowLive(true);
              }}
              style={{ ...inputStyle, cursor: "pointer", fontWeight: 500 }}
            >
              Return to live
            </button>
          )}
          <span style={{ fontSize: 11, color: followLive ? "#0f766e" : "#64748b", paddingBottom: 7 }}>
            {followLive ? "Live · refreshes every 5 sec" : "Auto-loads when filters change"}
            {loading ? " · Updating…" : lastUpdated ? ` · Updated ${lastUpdated.toLocaleTimeString()}` : ""}
          </span>
          {error && <span style={{ fontSize: 11, color: "#dc2626", paddingBottom: 7 }}>{error}</span>}
        </div>

        <div style={{ marginBottom: 12 }}>
          <span style={labelStyle}>Metric</span>
          <div style={{ display: "flex", flexWrap: "wrap", gap: 10 }}>
            {metrics.map((item) => (
              <label
                key={item.id}
                style={{ fontSize: 12, color: "#475569", display: "flex", alignItems: "center", gap: 5, cursor: "pointer" }}
              >
                <input
                  type="radio"
                  name="metric"
                  checked={metric === item.id}
                  onChange={() => setMetric(item.id)}
                />
                {item.label}
              </label>
            ))}
          </div>
        </div>

        <div>
          <span style={labelStyle}>Services (compare)</span>
          <div style={{ display: "flex", flexWrap: "wrap", gap: 8, maxHeight: 72, overflow: "auto" }}>
            {services.length === 0 ? (
              <span style={{ fontSize: 11, color: "#94a3b8" }}>No services in history yet</span>
            ) : services.map((name) => (
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
                <input type="checkbox" checked={selectedServices.includes(name)} onChange={() => toggleService(name)} />
                {name}
              </label>
            ))}
          </div>
        </div>
      </div>

      <TimeSeriesChart series={series} metric={metric} metricLabel={metricLabel} />
    </div>
  );
}
