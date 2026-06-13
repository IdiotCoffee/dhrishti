import axios from "axios";

const GO_URL = "http://localhost:8090";
const HISTORY_URL = "http://localhost:8000";

export async function fetchGraph() {
  const response = await axios.get(`${GO_URL}/graph`);
  return response.data;
}

export async function fetchHistoryServices() {
  const response = await axios.get(`${HISTORY_URL}/api/v1/services`);
  return response.data.services ?? [];
}

export async function fetchHistoryMetrics() {
  const response = await axios.get(`${HISTORY_URL}/api/v1/metrics`);
  return response.data;
}

export async function fetchTimeseries({ services, metric, start, end }) {
  const response = await axios.get(`${HISTORY_URL}/api/v1/timeseries`, {
    params: {
      services: services.join(","),
      metric,
      start,
      end,
    },
  });
  return response.data;
}

export async function fetchGraphSnapshot(at) {
  const response = await axios.get(`${HISTORY_URL}/api/v1/graph/snapshot`, {
    params: { at },
  });
  return response.data;
}

export async function fetchGraphPlayback({ start, end }) {
  const response = await axios.get(`${HISTORY_URL}/api/v1/graph/playback`, {
    params: { start, end },
  });
  return response.data;
}
