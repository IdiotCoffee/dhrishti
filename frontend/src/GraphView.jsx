import React, { useEffect, useMemo, useRef } from "react";

import CytoscapeComponent from "react-cytoscapejs";

/*
Force-directed layout config — applied only when topology grows,
not on every 2s poll. Re-layout on each refresh would jitter nodes
and make runtime dependency structure harder to read.
*/
const INITIAL_LAYOUT = {
  name: "cose",
  animate: false,
  padding: 60,
  nodeRepulsion: 12000,
  idealEdgeLength: 140,
  nodeOverlap: 24,
};

/*
Classify nodes by service id (Docker / resolver names).
Observability readers scan topology by role: who calls whom.
*/
function classifyNode(id) {
  const lower = id.toLowerCase();
  if (lower === "client" || lower.endsWith("-client")) return "client";
  if (lower === "gateway" || lower.includes("gateway")) return "gateway";
  return "service";
}

/*
Edge color maps eBPF connection semantics to visual priority:
failed (red) > active (green) > slow avg latency (orange) > idle (gray).
*/
function edgeLineColor(edge) {
  if (edge.failed_connections > 0) return "#ef4444";
  if (edge.active_connections > 0) return "#22c55e";
  if (edge.average_duration_ms > 1000) return "#f97316";
  return "#6b7280";
}

function edgeLabel(edge) {
  return `${edge.average_duration_ms}ms | fail=${edge.failed_connections}`;
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
    const active = edge.active_connections > 0;
    elements.push({
      data: {
        id: stableEdgeId(edge),
        source: edge.source,
        target: edge.target,
        label: edgeLabel(edge),
        edgeColor: edgeLineColor(edge),
      },
      classes: active ? "active-edge" : "",
    });
  });

  return elements;
}

/*
Stylesheet encodes observability semantics — not decoration.
Colors answer: what failed, what is live, what is slow, what role is this node?
*/
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
      width: 2.5,
      label: "data(label)",
      "curve-style": "bezier",
      "target-arrow-shape": "triangle",
      "arrow-scale": 0.8,
      "line-color": "data(edgeColor)",
      "target-arrow-color": "data(edgeColor)",
      color: "#9ca3af",
      "font-size": 9,
      "text-background-color": "#0f1117",
      "text-background-opacity": 0.85,
      "text-background-padding": 2,
    },
  },
  {
    /*
    Dashed offset animation on active edges signals live traffic
    without charts — motion draws the eye along hot paths.
    */
    selector: "edge.active-edge",
    style: {
      width: 3,
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
        Edges
      </div>
      <div style={row}>
        <span style={swatch("#ef4444")} />
        <span>Failed connections</span>
      </div>
      <div style={row}>
        <span style={swatch("#22c55e", true)} />
        <span>Active (live traffic)</span>
      </div>
      <div style={row}>
        <span style={swatch("#f97316")} />
        <span>Slow (&gt;1s avg)</span>
      </div>
      <div style={row}>
        <span style={swatch("#6b7280")} />
        <span>Idle / baseline</span>
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

/*
GraphView renders live service dependency topology from eBPF-derived graph state.
*/
export default function GraphView({ graph }) {
  const cyRef = useRef(null);
  const knownNodesRef = useRef(new Set());
  const animFrameRef = useRef(null);

  const elements = useMemo(() => graphToElements(graph), [graph]);

  /*
  Preserve positions between polls: layout runs only when new node ids appear.
  Existing nodes keep their coordinates so the graph stays readable over time.
  */
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
      <CytoscapeComponent
        cy={handleCy}
        elements={elements}
        stylesheet={STYLESHEET}
        style={{ width: "100%", height: "100%" }}
        /*
        Do not pass `layout` — a new object each render re-ran cose every poll
        and caused jitter. Layout is triggered manually in useEffect above.
        */
        wheelSensitivity={0.2}
      />
    </div>
  );
}
