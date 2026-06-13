import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";

import cytoscape from "cytoscape";

import ZoomSlider, { MAX_ZOOM, MIN_ZOOM, sliderToZoom, zoomToSlider } from "./ZoomSlider";
import { FONT } from "./styles";

const INITIAL_LAYOUT = {
  name: "cose",
  animate: false,
  padding: 60,
  nodeRepulsion: 12000,
  idealEdgeLength: 140,
  nodeOverlap: 24,
};

const HEAT = {
  normal: { color: "#22c55e", label: "Normal" },
  elevated: { color: "#f59e0b", label: "Elevated" },
  hot: { color: "#ef4444", label: "Hot" },
  idle: { color: "#cbd5e1", label: "Idle" },
};

function classifyNode(id) {
  const lower = id.toLowerCase();
  if (lower === "client") return "client";
  if (lower.includes("gateway")) return "gateway";
  return "service";
}

function formatRps(rps) {
  if (rps >= 10) return rps.toFixed(1);
  if (rps >= 1) return rps.toFixed(1);
  return rps.toFixed(2);
}

function formatFailRate(rate) {
  return `${(rate * 100).toFixed(rate < 0.01 && rate > 0 ? 1 : 0)}%`;
}

function isEdgeHot(edge) {
  return (edge.active_connections ?? 0) > 0;
}

function sortedValues(edges, getter) {
  return edges.map(getter).filter((v) => v > 0).sort((a, b) => a - b);
}

function percentileRank(sorted, value) {
  if (!sorted.length) return 0;
  const below = sorted.filter((v) => v < value).length;
  return below / sorted.length;
}

function buildColorContext(edges) {
  return {
    p95: sortedValues(edges, (e) => e.p95_latency_ms ?? 0),
    fail: sortedValues(edges, (e) => e.failure_rate ?? 0),
    rps: sortedValues(edges, (e) => e.requests_per_second ?? 0),
  };
}

function edgeHeatPercentile(edge, ctx) {
  const scores = [];
  const p95 = edge.p95_latency_ms ?? 0;
  const fail = edge.failure_rate ?? 0;
  const rps = edge.requests_per_second ?? 0;

  if (ctx.p95.length && p95 > 0) scores.push(percentileRank(ctx.p95, p95));
  if (ctx.fail.length && fail > 0) scores.push(percentileRank(ctx.fail, fail));
  if (ctx.rps.length && rps > 0) scores.push(percentileRank(ctx.rps, rps));

  return scores.length ? Math.max(...scores) : 0;
}

function edgeHeatLevel(edge, ctx) {
  const heat = edgeHeatPercentile(edge, ctx);
  const hasTraffic =
    (edge.requests_per_second ?? 0) > 0 ||
    (edge.p95_latency_ms ?? 0) > 0 ||
    (edge.failure_rate ?? 0) > 0;

  if (!hasTraffic) return "idle";
  if (heat >= 0.75) return "hot";
  if (heat >= 0.5) return "elevated";
  return "normal";
}

function edgeLineColor(edge, ctx) {
  return HEAT[edgeHeatLevel(edge, ctx)].color;
}

function edgeWidth(rps, ctx) {
  const min = 2;
  const max = 9;
  if (!ctx.rps.length || rps <= 0) return min;
  const rank = percentileRank(ctx.rps, rps);
  return min + rank * (max - min);
}

function edgeLabel(edge) {
  const rps = edge.requests_per_second ?? 0;
  const avg = edge.recent_average_latency_ms ?? 0;
  const p95 = edge.p95_latency_ms ?? 0;
  const fail = edge.failure_rate ?? 0;

  return [
    `RPS: ${formatRps(rps)}`,
    `TCP avg: ${avg}ms`,
    `TCP p95: ${p95}ms`,
    `fail: ${formatFailRate(fail)}`,
  ].join("\n");
}

function edgeTooltip(edge, ctx) {
  const level = edgeHeatLevel(edge, ctx);
  return [
    `${edge.source} → ${edge.target}:${edge.port}`,
    `Heat: ${HEAT[level].label}`,
    `Connections: ${edge.connection_count ?? 0}`,
    `Active: ${edge.active_connections ?? 0}`,
    `RPS: ${formatRps(edge.requests_per_second ?? 0)}`,
    `TCP avg: ${edge.recent_average_latency_ms ?? 0}ms`,
    `TCP p95: ${edge.p95_latency_ms ?? 0}ms`,
    `Failure: ${formatFailRate(edge.failure_rate ?? 0)}`,
  ].join("\n");
}

function stableEdgeId(edge) {
  return `${edge.source}->${edge.target}:${edge.port}`;
}

function buildEdgeElement(edge, ctx) {
  const hot = isEdgeHot(edge);
  const rps = edge.requests_per_second ?? 0;

  return {
    group: "edges",
    data: {
      id: stableEdgeId(edge),
      source: edge.source,
      target: edge.target,
      label: edgeLabel(edge),
      tooltip: edgeTooltip(edge, ctx),
      edgeColor: edgeLineColor(edge, ctx),
      edgeWidth: edgeWidth(rps, ctx),
    },
    classes: hot ? "active-edge" : "idle-edge",
  };
}

function graphToElements(graph, ctx) {
  const elements = [];

  graph.nodes.forEach((node) => {
    elements.push({
      group: "nodes",
      data: { id: node.id, label: node.id },
      classes: classifyNode(node.id),
    });
  });

  graph.edges.forEach((edge) => {
    elements.push(buildEdgeElement(edge, ctx));
  });

  return elements;
}

function graphStructureKey(graph) {
  const nodes = graph.nodes.map((n) => n.id).sort().join(",");
  const edges = graph.edges.map(stableEdgeId).sort().join(",");
  return `${nodes}|${edges}`;
}

function updateEdgeMetrics(cy, edges, ctx) {
  edges.forEach((edge) => {
    const ele = cy.getElementById(stableEdgeId(edge));
    if (ele.empty()) return;

    const hot = isEdgeHot(edge);
    const rps = edge.requests_per_second ?? 0;

    ele.data({
      label: edgeLabel(edge),
      tooltip: edgeTooltip(edge, ctx),
      edgeColor: edgeLineColor(edge, ctx),
      edgeWidth: edgeWidth(rps, ctx),
    });

    if (hot) {
      ele.addClass("active-edge");
      ele.removeClass("idle-edge");
    } else {
      ele.removeClass("active-edge");
      ele.addClass("idle-edge");
    }
  });
}

const STYLESHEET = [
  {
    selector: "core",
    style: { "background-color": "#faf8f5" },
  },
  {
    selector: "node",
    style: {
      label: "data(label)",
      shape: "round-rectangle",
      width: "label",
      height: 34,
      padding: "10px",
      "font-family": FONT,
      "font-size": 12,
      "font-weight": 500,
      color: "#1e293b",
      "text-valign": "center",
      "text-halign": "center",
      "background-color": "#ffffff",
      "border-width": 2,
    },
  },
  {
    selector: "node.client",
    style: { "border-color": "#8b5cf6", "background-color": "#f5f3ff" },
  },
  {
    selector: "node.gateway",
    style: { "border-color": "#3b82f6", "background-color": "#eff6ff" },
  },
  {
    selector: "node.service",
    style: { "border-color": "#94a3b8", "background-color": "#f8fafc" },
  },
  {
    selector: "edge",
    style: {
      width: "data(edgeWidth)",
      label: "data(label)",
      "curve-style": "bezier",
      "target-arrow-shape": "triangle",
      "arrow-scale": 0.85,
      "line-color": "data(edgeColor)",
      "target-arrow-color": "data(edgeColor)",
      "font-family": FONT,
      color: "#64748b",
      "font-size": 9,
      "text-wrap": "wrap",
      "text-max-width": 120,
      "text-background-color": "#faf8f5",
      "text-background-opacity": 0.95,
      "text-background-padding": 3,
    },
  },
  {
    selector: "edge.active-edge",
    style: {
      "line-style": "dashed",
      "line-dash-pattern": [8, 5],
    },
  },
];

function Legend() {
  const row = { display: "flex", alignItems: "center", gap: 8, marginBottom: 5 };
  const swatch = (color, dashed = false) => ({
    width: 28,
    height: 0,
    borderTop: `2.5px ${dashed ? "dashed" : "solid"} ${color}`,
  });

  return (
    <div
      style={{
        position: "absolute",
        top: 14,
        left: 14,
        zIndex: 10,
        pointerEvents: "none",
        fontFamily: FONT,
        fontSize: 11,
        color: "#475569",
        background: "rgba(255, 255, 255, 0.94)",
        border: "1px solid #e2e8f0",
        borderRadius: 8,
        padding: "12px 14px",
        lineHeight: 1.4,
        boxShadow: "0 2px 8px rgba(15, 23, 42, 0.06)",
      }}
    >
      <div style={{ fontWeight: 600, color: "#1e293b", marginBottom: 8 }}>
        Service graph
      </div>
      <div style={{ fontSize: 10, color: "#94a3b8", marginBottom: 8 }}>
        Edge color = heat · dashed = live
      </div>
      {["normal", "elevated", "hot"].map((level) => (
        <div style={row} key={level}>
          <span style={swatch(HEAT[level].color)} />
          <span style={swatch(HEAT[level].color, true)} />
          <span>{HEAT[level].label}</span>
        </div>
      ))}
      <div style={{ ...row, marginTop: 2 }}>
        <span style={swatch(HEAT.idle.color)} />
        <span>No recent traffic</span>
      </div>
      <div style={{ margin: "8px 0 4px", fontSize: 10, color: "#94a3b8" }}>Nodes</div>
      <div style={row}>
        <span style={{ width: 10, height: 10, borderRadius: 2, background: "#f5f3ff", border: "2px solid #8b5cf6" }} />
        <span>Client (aggregated)</span>
      </div>
      <div style={row}>
        <span style={{ width: 10, height: 10, borderRadius: 2, background: "#eff6ff", border: "2px solid #3b82f6" }} />
        <span>Entry / gateway</span>
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
        fontFamily: FONT,
        fontSize: 11,
        lineHeight: 1.5,
        color: "#334155",
        background: "rgba(255, 255, 255, 0.97)",
        border: "1px solid #e2e8f0",
        borderRadius: 6,
        padding: "8px 10px",
        whiteSpace: "pre-line",
        maxWidth: 280,
        boxShadow: "0 4px 12px rgba(15, 23, 42, 0.08)",
      }}
    >
      {tooltip.text}
    </div>
  );
}

export default function GraphView({ graph, showLegend = true, showZoom = true }) {
  const containerRef = useRef(null);
  const cyRef = useRef(null);
  const structureKeyRef = useRef("");
  const animFrameRef = useRef(null);
  const zoomFromSliderRef = useRef(false);
  const zoomSliderRef = useRef(42);
  const [tooltip, setTooltip] = useState(null);
  const [zoomSlider, setZoomSlider] = useState(42);

  zoomSliderRef.current = zoomSlider;

  const colorCtx = useMemo(
    () => buildColorContext(graph.edges),
    [graph.edges],
  );

  const structureKey = useMemo(
    () => graphStructureKey(graph),
    [graph.nodes, graph.edges],
  );

  const applyZoom = useCallback((sliderValue, cy = cyRef.current) => {
    if (!cy) return;
    cy.resize();
    if (cy.width() === 0 || cy.height() === 0) return;
    zoomFromSliderRef.current = true;
    const level = sliderToZoom(sliderValue);
    cy.zoom({
      level,
      renderedPosition: { x: cy.width() / 2, y: cy.height() / 2 },
    });
    requestAnimationFrame(() => {
      zoomFromSliderRef.current = false;
    });
  }, []);

  const handleZoomChange = (value) => {
    setZoomSlider(value);
    applyZoom(value);
  };

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return undefined;

    const cy = cytoscape({
      container,
      elements: [],
      style: STYLESHEET,
      wheelSensitivity: 0.2,
      minZoom: MIN_ZOOM,
      maxZoom: MAX_ZOOM,
    });
    cyRef.current = cy;

    applyZoom(zoomSliderRef.current, cy);

    cy.on("zoom", () => {
      if (zoomFromSliderRef.current) return;
      setZoomSlider(zoomToSlider(cy.zoom()));
    });

    cy.on("mouseover", "edge", (evt) => {
      const pos = evt.renderedPosition;
      setTooltip({
        x: pos.x,
        y: pos.y,
        text: evt.target.data("tooltip"),
      });
    });

    cy.on("mouseout", "edge", () => setTooltip(null));

    let dashOffset = 0;
    const tick = () => {
      if (cyRef.current) {
        dashOffset = (dashOffset + 0.5) % 26;
        cyRef.current
          .edges(".active-edge")
          .style("line-dash-offset", dashOffset);
      }
      animFrameRef.current = requestAnimationFrame(tick);
    };
    animFrameRef.current = requestAnimationFrame(tick);

    const resizeObserver = new ResizeObserver(() => {
      const instance = cyRef.current;
      if (!instance) return;
      instance.resize();
      applyZoom(zoomSliderRef.current, instance);
    });
    resizeObserver.observe(container);

    return () => {
      resizeObserver.disconnect();
      if (animFrameRef.current != null) {
        cancelAnimationFrame(animFrameRef.current);
        animFrameRef.current = null;
      }
      cy.destroy();
      cyRef.current = null;
      structureKeyRef.current = "";
    };
  }, [applyZoom]);

  useEffect(() => {
    const cy = cyRef.current;
    if (!cy) return;

    if (graph.nodes.length === 0) {
      cy.elements().remove();
      structureKeyRef.current = "";
      return;
    }

    if (structureKey !== structureKeyRef.current) {
      cy.batch(() => {
        cy.elements().remove();
        cy.add(graphToElements(graph, colorCtx));
      });
      structureKeyRef.current = structureKey;
      if (graph.edges.length > 0) {
        const layout = cy.layout(INITIAL_LAYOUT);
        layout.one("layoutstop", () => {
          applyZoom(zoomSliderRef.current, cy);
        });
        layout.run();
      } else {
        applyZoom(zoomSliderRef.current, cy);
      }
      return;
    }

    updateEdgeMetrics(cy, graph.edges, colorCtx);
  }, [graph, colorCtx, structureKey, applyZoom]);

  return (
    <div style={{ width: "100%", height: "100%", position: "relative" }}>
      {showLegend && <Legend />}
      {showZoom && (
        <ZoomSlider value={zoomSlider} onChange={handleZoomChange} />
      )}
      <EdgeTooltip tooltip={tooltip} />
      <div
        ref={containerRef}
        style={{ width: "100%", height: "100%" }}
      />
    </div>
  );
}
