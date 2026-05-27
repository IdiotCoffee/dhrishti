import React, { useEffect, useState } from "react";

import GraphView from "./GraphView";

import { fetchGraph } from "./api";

/*
Main observability UI.
*/
export default function App() {
  const [graph, setGraph] = useState({
    nodes: [],
    edges: [],
  });

  /*
  Prefer WebSocket snapshots for snappy edge state.

  Observability is about reaction time:
  when a connection closes/fails/recover, the UI should reflect immediately.

  We keep polling as a fallback for environments where WS is unavailable.
  */
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
        } catch (e) {
          // If messages are malformed, fall back to polling.
          startPolling();
        }
      };

      ws.onerror = () => {
        startPolling();
      };

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
        background: "#0f1117",
      }}
    >
      <GraphView graph={graph} />
    </div>
  );
}
