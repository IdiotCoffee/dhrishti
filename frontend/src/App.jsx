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
  Poll runtime graph every 2 seconds.

  Very intentionally simple.

  We do NOT need WebSockets yet.
  */
  useEffect(() => {
    async function loadGraph() {
      try {
        const data = await fetchGraph();

        setGraph(data);
      } catch (err) {
        console.error("fetching graph:", err);
      }
    }

    /*
    Initial load.
    */
    loadGraph();

    /*
    Periodic refresh.
    */
    const interval = setInterval(loadGraph, 2000);

    return () => clearInterval(interval);
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
