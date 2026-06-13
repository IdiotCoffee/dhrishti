import { FONT } from "./styles";

export function toLocalInputValue(date) {
  const d = new Date(date);
  d.setMinutes(d.getMinutes() - d.getTimezoneOffset());
  return d.toISOString().slice(0, 16);
}

export function toISO(localValue) {
  return new Date(localValue).toISOString();
}

export function defaultRange() {
  const end = new Date();
  const start = new Date(end.getTime() - 10 * 60 * 1000);
  return { start, end };
}

export const labelStyle = {
  fontSize: 11,
  color: "#64748b",
  marginBottom: 4,
  display: "block",
};

export const inputStyle = {
  fontFamily: FONT,
  fontSize: 12,
  padding: "6px 8px",
  border: "1px solid #e2e8f0",
  borderRadius: 6,
  color: "#1e293b",
  background: "#fff",
};

export const btnStyle = {
  fontFamily: FONT,
  fontSize: 12,
  fontWeight: 500,
  padding: "7px 14px",
  border: "1px solid #cbd5e1",
  borderRadius: 6,
  background: "#f8fafc",
  color: "#1e293b",
  cursor: "pointer",
};
