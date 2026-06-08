import React from "react";

const FONT = '"IBM Plex Sans", system-ui, sans-serif';

function formatDestinations(dests) {
  if (!dests || Object.keys(dests).length === 0) return "—";
  return Object.entries(dests)
    .sort((a, b) => b[1] - a[1])
    .map(([name, count]) => `${name} (${count})`)
    .join(", ");
}

export default function UnknownIPTable({ entries, entryServices }) {
  const rows = entries ?? [];
  const entriesLabel = (entryServices ?? []).join(", ") || "entry services";

  return (
    <div
      style={{
        fontFamily: FONT,
        background: "#ffffff",
        borderLeft: "1px solid #e2e8f0",
        display: "flex",
        flexDirection: "column",
        height: "100%",
        minWidth: 280,
        maxWidth: 360,
      }}
    >
      <div
        style={{
          padding: "14px 16px 10px",
          borderBottom: "1px solid #e2e8f0",
        }}
      >
        <div style={{ fontSize: 13, fontWeight: 600, color: "#1e293b" }}>
          Client IPs
        </div>
        <div style={{ fontSize: 11, color: "#64748b", marginTop: 4, lineHeight: 1.4 }}>
          Unresolved hosts hitting entry point{entryServices?.length !== 1 ? "s" : ""}:{" "}
          <code style={{ color: "#0f766e" }}>{entriesLabel}</code>
        </div>
      </div>

      <div style={{ flex: 1, overflow: "auto" }}>
        {rows.length === 0 ? (
          <div style={{ padding: 16, fontSize: 12, color: "#94a3b8" }}>
            No client IPs yet — traffic to an entry service will appear here
          </div>
        ) : (
          <table
            style={{
              width: "100%",
              borderCollapse: "collapse",
              fontSize: 11,
            }}
          >
            <thead>
              <tr style={{ color: "#64748b", textAlign: "left" }}>
                <th style={thStyle}>IP</th>
                <th style={thStyle}>Conn</th>
                <th style={thStyle}>Active</th>
                <th style={thStyle}>Targets</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.ip} style={{ borderTop: "1px solid #f1f5f9" }}>
                  <td style={tdStyle}>
                    <code style={{ color: "#0f766e", fontSize: 11 }}>{row.ip}</code>
                  </td>
                  <td style={tdStyle}>{row.connection_count}</td>
                  <td style={tdStyle}>
                    {row.active_connections > 0 ? (
                      <span style={{ color: "#d97706", fontWeight: 500 }}>
                        {row.active_connections}
                      </span>
                    ) : (
                      0
                    )}
                  </td>
                  <td style={{ ...tdStyle, color: "#475569", lineHeight: 1.4 }}>
                    {formatDestinations(row.destinations)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}

const thStyle = {
  padding: "8px 12px",
  fontWeight: 500,
  fontSize: 10,
  textTransform: "uppercase",
  letterSpacing: "0.04em",
};

const tdStyle = {
  padding: "10px 12px",
  verticalAlign: "top",
};
