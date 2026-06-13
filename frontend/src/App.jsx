import React, { useEffect, useState } from "react";

import GraphView from "./GraphView";
import ReplayView from "./ReplayView";
import TimelineView from "./TimelineView";
import UnknownIPTable from "./UnknownIPTable";

import { fetchGraph } from "./api";
import { FONT } from "./styles";

const TABS = [
  { id: "live", label: "Live graph" },
  { id: "timeline", label: "Timeline" },
  { id: "replay", label: "Replay" },
];

export default function App() {
  const [tab, setTab] = useState("live");
  const [graph, setGraph] = useState({
    nodes: [],
    edges: [],
    unknown_ips: [],
    entry_services: [],
  });

  useEffect(() => {
    if (tab !== "live") return undefined;

    const wsUrl = "ws://localhost:8090/ws";
    const pollingMs = 2000;
    let ws = null;
    let pollInterval = null;
    const stoppedRef = { stopped: false };

    async function loadGraph() {
      try {
        const data = await fetchGraph();
        setGraph(data);
      } catch (err) {
        console.error("fetching graph:", err);
      }
    }

    const startPolling = () => {
      if (pollInterval != null) return;
      loadGraph();
      pollInterval = setInterval(loadGraph, pollingMs);
    };

    try {
      ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        console.info("[dhrishti] graph WebSocket connected");
      };

      ws.onmessage = (evt) => {
        try {
          const data = JSON.parse(evt.data);
          setGraph(data);
        } catch {
          startPolling();
        }
      };

      ws.onerror = () => startPolling();
      ws.onclose = () => {
        if (!stoppedRef.stopped) startPolling();
      };
    } catch {
      startPolling();
    }

    return () => {
      stoppedRef.stopped = true;
      if (ws) ws.close();
      if (pollInterval) clearInterval(pollInterval);
    };
  }, [tab]);

  return (
    <div
      style={{
        width: "100vw",
        height: "100vh",
        display: "flex",
        flexDirection: "column",
        background: "#f1f5f9",
        fontFamily: FONT,
      }}
    >
      <header
        style={{
          display: "flex",
          alignItems: "center",
          gap: 4,
          padding: "10px 14px",
          background: "#ffffff",
          borderBottom: "1px solid #e2e8f0",
          flexShrink: 0,
        }}
      >
        <span
          style={{
            fontSize: 13,
            fontWeight: 600,
            color: "#1e293b",
            marginRight: 12,
          }}
        >
          Dhrishti
        </span>
        {TABS.map((t) => (
          <button
            key={t.id}
            type="button"
            onClick={() => setTab(t.id)}
            style={{
              fontFamily: FONT,
              fontSize: 12,
              fontWeight: tab === t.id ? 600 : 400,
              padding: "6px 12px",
              border: "1px solid",
              borderColor: tab === t.id ? "#cbd5e1" : "transparent",
              borderRadius: 6,
              background: tab === t.id ? "#f8fafc" : "transparent",
              color: tab === t.id ? "#1e293b" : "#64748b",
              cursor: "pointer",
            }}
          >
            {t.label}
          </button>
        ))}
      </header>

      <div style={{ flex: 1, display: "flex", minHeight: 0 }}>
        {tab === "live" ? (
          <>
            <div style={{ flex: 1, minWidth: 0 }}>
              <GraphView graph={graph} />
            </div>
            <UnknownIPTable
              entries={graph.unknown_ips}
              entryServices={graph.entry_services}
            />
          </>
        ) : tab === "timeline" ? (
          <TimelineView />
        ) : (
          <ReplayView />
        )}
      </div>
    </div>
  );
}
