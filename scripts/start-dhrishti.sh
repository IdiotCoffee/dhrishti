#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PID_DIR="${ROOT}/.dhrishti"
LOG_DIR="${PID_DIR}/logs"
mkdir -p "${PID_DIR}" "${LOG_DIR}"

# Load repo .env (optional). Does not override vars already set in the shell.
if [[ -f "${ROOT}/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "${ROOT}/.env"
  set +a
fi

GO_PID=""
API_PID=""
FE_PID=""

cleanup() {
  echo ""
  echo "Stopping Dhrishti..."
  [[ -n "${FE_PID}" ]] && kill "${FE_PID}" 2>/dev/null || true
  [[ -n "${API_PID}" ]] && kill "${API_PID}" 2>/dev/null || true
  if [[ -n "${GO_PID}" ]]; then
    if [[ "${EUID}" -eq 0 ]]; then
      kill "${GO_PID}" 2>/dev/null || true
    else
      sudo kill "${GO_PID}" 2>/dev/null || true
    fi
  fi
  wait 2>/dev/null || true
  echo "Done."
}

trap cleanup EXIT INT TERM

if [[ ! -f "${ROOT}/ebpf/tcp_connect.bpf.o" ]]; then
  echo "eBPF probes not built. Run: make ebpf"
  exit 1
fi

if ! command -v redis-cli >/dev/null 2>&1 || ! redis-cli -h localhost ping >/dev/null 2>&1; then
  echo "WARNING: Redis not reachable on localhost:6379"
  echo "Start your local Redis server, then re-run: make dhrishti"
fi

# History API venv (--no-cache-dir avoids stale global pip wheel cache warnings)
if [[ ! -d "${ROOT}/history-api/.venv" ]]; then
  echo "Creating history-api virtualenv..."
  python3 -m venv "${ROOT}/history-api/.venv"
  PIP_DISABLE_PIP_VERSION_CHECK=1 \
    "${ROOT}/history-api/.venv/bin/pip" install --no-cache-dir -q \
    -r "${ROOT}/history-api/requirements.txt"
fi

if [[ ! -d "${ROOT}/frontend/node_modules" ]]; then
  echo "Installing frontend dependencies..."
  (cd "${ROOT}/frontend" && npm install)
fi

echo "=============================================="
echo "  Dhrishti — starting all services"
echo "=============================================="
echo "  Go engine:    http://localhost:8090"
echo "  History API:  http://localhost:8000"
echo "  Frontend:     http://localhost:5173"
echo "  Press Ctrl+C to stop"
echo "=============================================="
echo ""

cd "${ROOT}"

export DHRISHTI_REDIS_URL="${DHRISHTI_REDIS_URL:-redis://localhost:6379}"
export DHRISHTI_HISTORY_ENABLED="${DHRISHTI_HISTORY_ENABLED:-true}"
export DHRISHTI_HISTORY_INTERVAL="${DHRISHTI_HISTORY_INTERVAL:-10s}"
export DHRISHTI_HISTORY_RETENTION="${DHRISHTI_HISTORY_RETENTION:-24h}"
export DHRISHTI_HISTORY_API_PORT="${DHRISHTI_HISTORY_API_PORT:-8000}"
export DHRISHTI_HISTORY_RETENTION_HOURS="${DHRISHTI_HISTORY_RETENTION_HOURS:-24}"

if [[ "${EUID}" -eq 0 ]]; then
  go run main.go >"${LOG_DIR}/engine.log" 2>&1 &
else
  sudo -E go run main.go >"${LOG_DIR}/engine.log" 2>&1 &
fi
GO_PID=$!

sleep 1

DHRISHTI_HISTORY_API_PORT="${DHRISHTI_HISTORY_API_PORT}" \
  "${ROOT}/history-api/.venv/bin/uvicorn" main:app \
  --app-dir "${ROOT}/history-api" \
  --host 0.0.0.0 \
  --port "${DHRISHTI_HISTORY_API_PORT}" \
  >"${LOG_DIR}/history-api.log" 2>&1 &
API_PID=$!

(cd "${ROOT}/frontend" && npm run dev) >"${LOG_DIR}/frontend.log" 2>&1 &
FE_PID=$!

for _ in $(seq 1 30); do
  if curl -sf "http://localhost:8090/metrics" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

echo "Logs: ${LOG_DIR}/"
wait "${GO_PID}" "${API_PID}" "${FE_PID}" 2>/dev/null || wait
