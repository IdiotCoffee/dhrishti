import React, { useEffect, useState } from "react";

import GraphView from "./GraphView";
import UnknownIPTable from "./UnknownIPTable";

import { fetchGraph } from "./api";

export default function App() {
  const [graph, setGraph] = useState({
    nodes: [],
    edges: [],
    unknown_ips: [],
    entry_services: [],
  });

  useEffect(() => {
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
  }, []);

  return (
    <div
      style={{
        width: "100vw",
        height: "100vh",
        display: "flex",
        background: "#f1f5f9",
      }}
    >
      <div style={{ flex: 1, minWidth: 0 }}>
        <GraphView graph={graph} />
      </div>
      <UnknownIPTable entries={graph.unknown_ips} entryServices={graph.entry_services} />
    </div>
  );
}
