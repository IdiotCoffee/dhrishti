import React, { useEffect, useMemo, useRef, useState } from "react";

import CytoscapeComponent from "react-cytoscapejs";

const INITIAL_LAYOUT = {
  name: "cose",
  animate: false,
  padding: 60,
  nodeRepulsion: 12000,
  idealEdgeLength: 140,
  nodeOverlap: 24,
};

function classifyNode(id) {
  const lower = id.toLowerCase();
  if (lower === "client" || lower.endsWith("-client")) return "client";
  if (lower === "gateway" || lower.includes("gateway")) return "gateway";
  return "service";
}

/*
Rolling operational metrics (RPS, p95, failure_rate) describe *recent* edge health.
Lifetime connection counters stay in tooltips — labels show what operators act on now.
*/
function formatRps(rps) {
  if (rps >= 10) return rps.toFixed(1);
  if (rps >= 1) return rps.toFixed(1);
  return rps.toFixed(2);
}

function formatFailRate(rate) {
  return `${(rate * 100).toFixed(rate < 0.01 && rate > 0 ? 1 : 0)}%`;
}

/*
Dashed "live" animation only when a socket is open right now.
Longer mock request handlers keep active_connections > 0 long enough to see.
*/
function isEdgeHot(edge) {
  return (edge.active_connections ?? 0) > 0;
}

/*
Edge color priority: unhealthy (fail rate) > tail latency (p95) > live traffic > idle.
*/
function edgeLineColor(edge) {
  if (edge.failure_rate > 0.5) return "#ef4444";
  if (edge.p95_latency_ms > 1000) return "#f97316";
  if ((edge.active_connections ?? 0) > 0) return "#22c55e";
  if ((edge.requests_per_second ?? 0) > 0) return "#84cc16";
  return "#6b7280";
}

/*
Thickness encodes load (requests_per_second) so hot dependencies stand out
without a separate chart — width is proportional to sqrt(RPS) for readability.
*/
function edgeWidth(rps) {
  const min = 2;
  const max = 10;
  return min + Math.min(max - min, Math.sqrt(Math.max(0, rps)) * 2.5);
}

function edgeLabel(edge) {
  const rps = edge.requests_per_second ?? 0;
  const avg = edge.recent_average_latency_ms ?? 0;
  const p95 = edge.p95_latency_ms ?? 0;
  const fail = edge.failure_rate ?? 0;

  return [
    `RPS: ${formatRps(rps)}`,
    `avg: ${avg}ms`,
    `p95: ${p95}ms`,
    `fail: ${formatFailRate(fail)}`,
  ].join("\n");
}

/*
Tooltip carries lifetime + rolling detail — too much for edge labels,
but needed when inspecting a single dependency under load.
*/
function edgeTooltip(edge) {
  const rps = edge.requests_per_second ?? 0;
  const avg = edge.recent_average_latency_ms ?? 0;
  const p95 = edge.p95_latency_ms ?? 0;
  const fail = edge.failure_rate ?? 0;

  return [
    `${edge.source} → ${edge.target}:${edge.port}`,
    `Connections: ${edge.connection_count ?? 0}`,
    `Active: ${edge.active_connections ?? 0}`,
    `RPS: ${formatRps(rps)}`,
    `Recent avg latency: ${avg}ms`,
    `p95 latency: ${p95}ms`,
    `Failure rate: ${formatFailRate(fail)}`,
  ].join("\n");
}

function stableEdgeId(edge) {
  return `${edge.source}->${edge.target}:${edge.port}`;
}

function graphToElements(graph) {
  const elements = [];

  graph.nodes.forEach((node) => {
    elements.push({
      data: { id: node.id, label: node.id },
      classes: classifyNode(node.id),
    });
  });

  graph.edges.forEach((edge) => {
    const hot = isEdgeHot(edge);
    const rps = edge.requests_per_second ?? 0;

    elements.push({
      data: {
        id: stableEdgeId(edge),
        source: edge.source,
        target: edge.target,
        label: edgeLabel(edge),
        tooltip: edgeTooltip(edge),
        edgeColor: edgeLineColor(edge),
        edgeWidth: edgeWidth(rps),
        hot: hot ? "yes" : "no",
      },
      classes: hot ? "active-edge" : "idle-edge",
    });
  });

  return elements;
}

const STYLESHEET = [
  {
    selector: "core",
    style: { "background-color": "#0f1117" },
  },
  {
    selector: "node",
    style: {
      label: "data(label)",
      shape: "round-rectangle",
      width: "label",
      height: 36,
      padding: "10px",
      "font-size": 11,
      color: "#e5e7eb",
      "text-valign": "center",
      "text-halign": "center",
      "border-width": 1,
      "border-color": "#374151",
    },
  },
  {
    selector: "node.client",
    style: { "background-color": "#7c3aed" },
  },
  {
    selector: "node.gateway",
    style: { "background-color": "#0284c7" },
  },
  {
    selector: "node.service",
    style: { "background-color": "#475569" },
  },
  {
    selector: "edge",
    style: {
      width: "data(edgeWidth)",
      label: "data(label)",
      "curve-style": "bezier",
      "target-arrow-shape": "triangle",
      "arrow-scale": 0.8,
      "line-color": "data(edgeColor)",
      "target-arrow-color": "data(edgeColor)",
      color: "#9ca3af",
      "font-size": 8,
      "text-wrap": "wrap",
      "text-max-width": 120,
      "text-background-color": "#0f1117",
      "text-background-opacity": 0.9,
      "text-background-padding": 3,
    },
  },
  {
    selector: "edge.active-edge",
    style: {
      "line-style": "dashed",
      "line-dash-pattern": [8, 4],
    },
  },
];

function Legend() {
  const row = { display: "flex", alignItems: "center", gap: 8, marginBottom: 4 };
  const swatch = (color, dashed = false) => ({
    width: 24,
    height: 0,
    borderTop: `2px ${dashed ? "dashed" : "solid"} ${color}`,
  });

  return (
    <div
      style={{
        position: "absolute",
        top: 12,
        left: 12,
        zIndex: 10,
        pointerEvents: "none",
        fontSize: 11,
        color: "#9ca3af",
        background: "rgba(15, 17, 23, 0.9)",
        border: "1px solid #374151",
        borderRadius: 6,
        padding: "10px 12px",
        lineHeight: 1.4,
      }}
    >
      <div style={{ color: "#e5e7eb", marginBottom: 6, fontSize: 12 }}>
        Dependency graph
      </div>
      <div style={{ marginBottom: 6, color: "#6b7280", fontSize: 10 }}>
        Edges (rolling metrics)
      </div>
      <div style={row}>
        <span style={swatch("#ef4444")} />
        <span>High failure rate (&gt;50%)</span>
      </div>
      <div style={row}>
        <span style={swatch("#f97316")} />
        <span>Slow p95 (&gt;1s)</span>
      </div>
      <div style={row}>
        <span style={swatch("#22c55e", true)} />
        <span>Open connection (active)</span>
      </div>
      <div style={row}>
        <span style={swatch("#84cc16")} />
        <span>Recent traffic (RPS &gt; 0)</span>
      </div>
      <div style={row}>
        <span style={swatch("#6b7280")} />
        <span>Idle / baseline</span>
      </div>
      <div style={{ marginTop: 4, fontSize: 10, color: "#6b7280" }}>
        Line width ∝ √RPS · hover for detail
      </div>
      <div style={{ margin: "8px 0 6px", color: "#6b7280", fontSize: 10 }}>
        Nodes
      </div>
      <div style={row}>
        <span
          style={{
            width: 10,
            height: 10,
            borderRadius: 2,
            background: "#7c3aed",
          }}
        />
        <span>Client</span>
      </div>
      <div style={row}>
        <span
          style={{
            width: 10,
            height: 10,
            borderRadius: 2,
            background: "#0284c7",
          }}
        />
        <span>Gateway</span>
      </div>
      <div style={row}>
        <span
          style={{
            width: 10,
            height: 10,
            borderRadius: 2,
            background: "#475569",
          }}
        />
        <span>Internal service</span>
      </div>
    </div>
  );
}

function EdgeTooltip({ tooltip }) {
  if (!tooltip) return null;

  return (
    <div
      style={{
        position: "absolute",
        left: tooltip.x + 14,
        top: tooltip.y + 14,
        zIndex: 20,
        pointerEvents: "none",
        fontSize: 11,
        lineHeight: 1.5,
        color: "#e5e7eb",
        background: "rgba(15, 17, 23, 0.95)",
        border: "1px solid #4b5563",
        borderRadius: 6,
        padding: "8px 10px",
        whiteSpace: "pre-line",
        maxWidth: 280,
      }}
    >
      {tooltip.text}
    </div>
  );
}

export default function GraphView({ graph }) {
  const cyRef = useRef(null);
  const knownNodesRef = useRef(new Set());
  const animFrameRef = useRef(null);
  const [tooltip, setTooltip] = useState(null);

  const elements = useMemo(() => graphToElements(graph), [graph]);

  /*
  react-cytoscapejs can leave stale edge classes between polls.
  Sync color, width, and hot/idle class directly on each graph update.
  */
  useEffect(() => {
    const cy = cyRef.current;
    if (!cy) return;

    graph.edges.forEach((edge) => {
      const ele = cy.getElementById(stableEdgeId(edge));
      if (ele.empty()) return;

      const hot = isEdgeHot(edge);
      const rps = edge.requests_per_second ?? 0;

      ele.data({
        label: edgeLabel(edge),
        tooltip: edgeTooltip(edge),
        edgeColor: edgeLineColor(edge),
        edgeWidth: edgeWidth(rps),
        hot: hot ? "yes" : "no",
      });

      if (hot) {
        ele.addClass("active-edge");
        ele.removeClass("idle-edge");
      } else {
        ele.removeClass("active-edge");
        ele.addClass("idle-edge");
      }
    });
  }, [graph]);

  useEffect(() => {
    const cy = cyRef.current;
    if (!cy || graph.nodes.length === 0) return;

    const currentIds = new Set(graph.nodes.map((n) => n.id));
    const newIds = [...currentIds].filter((id) => !knownNodesRef.current.has(id));

    if (knownNodesRef.current.size === 0) {
      cy.layout(INITIAL_LAYOUT).run();
      knownNodesRef.current = new Set(currentIds);
      return;
    }

    if (newIds.length === 0) return;

    const saved = {};
    cy.nodes().forEach((n) => {
      if (!newIds.includes(n.id())) {
        saved[n.id()] = n.position();
      }
    });

    cy.layout(INITIAL_LAYOUT).run();
    cy.one("layoutstop", () => {
      Object.entries(saved).forEach(([id, pos]) => {
        cy.getElementById(id).position(pos);
      });
    });

    newIds.forEach((id) => knownNodesRef.current.add(id));
  }, [graph]);

  const handleCy = (cy) => {
    cyRef.current = cy;

    if (!cy.scratch("_edgeHandlers")) {
      cy.scratch("_edgeHandlers", true);

      cy.on("mouseover", "edge", (evt) => {
        const pos = evt.renderedPosition;
        setTooltip({
          x: pos.x,
          y: pos.y,
          text: evt.target.data("tooltip"),
        });
      });

      cy.on("mouseout", "edge", () => setTooltip(null));
    }

    if (animFrameRef.current != null) return;

    let dashOffset = 0;
    const tick = () => {
      if (cyRef.current) {
        dashOffset = (dashOffset + 0.6) % 24;
        cyRef.current
          .edges(".active-edge")
          .style("line-dash-offset", dashOffset);
      }
      animFrameRef.current = requestAnimationFrame(tick);
    };
    animFrameRef.current = requestAnimationFrame(tick);
  };

  useEffect(() => {
    return () => {
      if (animFrameRef.current != null) {
        cancelAnimationFrame(animFrameRef.current);
        animFrameRef.current = null;
      }
    };
  }, []);

  return (
    <div style={{ width: "100%", height: "100%", position: "relative" }}>
      <Legend />
      <EdgeTooltip tooltip={tooltip} />
      <CytoscapeComponent
        cy={handleCy}
        elements={elements}
        stylesheet={STYLESHEET}
        style={{ width: "100%", height: "100%" }}
        wheelSensitivity={0.2}
      />
    </div>
  );
}
