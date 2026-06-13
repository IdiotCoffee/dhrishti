#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT_DIR="${ROOT}/benchmark"

DURATION="${DURATION:-4m}"
VIRTUAL_USERS="${VIRTUAL_USERS:-200}"
CONNECTING_IPS="${CONNECTING_IPS:-}"

# k6 reads VIRTUAL_USERS; do not inherit a stale CLIENTS from the shell.
unset CLIENTS
export DURATION VIRTUAL_USERS CONNECTING_IPS

# shellcheck source=../benchmark/_lib.sh
source "${SCRIPT_DIR}/_lib.sh"

ensure_root "$@"

K6_SUMMARY="${K6_SUMMARY:-/tmp/dhrishti-simulation-summary.json}"
K6_SCRIPT="${SCRIPT_DIR}/simulation.js"

check_deps
check_stack

if ! curl -sf "${DHRISHTI_URL}/metrics" >/dev/null 2>&1; then
  echo "ERROR: Dhrishti not reachable at ${DHRISHTI_URL}"
  echo "Start it: make dhrishti"
  exit 1
fi

trap stop_fleet EXIT

duration_sec() {
  local raw="${1}"
  if [[ "${raw}" =~ ^([0-9]+)s$ ]]; then
    echo "${BASH_REMATCH[1]}"
    return
  fi
  if [[ "${raw}" =~ ^([0-9]+)m$ ]]; then
    echo $((BASH_REMATCH[1] * 60))
    return
  fi
  if [[ "${raw}" =~ ^([0-9]+)h$ ]]; then
    echo $((BASH_REMATCH[1] * 3600))
    return
  fi
  echo 240
}

FLEET_DURATION="$(duration_sec "${DURATION}")"
export FLEET_DURATION

echo "=============================================="
echo "  Dhrishti traffic simulation"
echo "=============================================="
echo "  duration:        ${DURATION} (${FLEET_DURATION}s)"
echo "  virtual_users:   ${VIRTUAL_USERS}"
echo "  connecting_ips:  ${CONNECTING_IPS:-auto ($(fleet_ips))}"
echo "  gateway:         ${BASE_URL}"
echo "=============================================="
echo ""

start_fleet
export DURATION
run_k6 "${K6_SUMMARY}" "${K6_SCRIPT}"

echo ""
echo "=== Post-simulation dhrishti /metrics ==="
curl -sf "${DHRISHTI_URL}/metrics" | python3 -m json.tool

echo ""
echo "=== History API health ==="
if curl -sf "http://localhost:8000/health" >/dev/null 2>&1; then
  curl -sf "http://localhost:8000/health" | python3 -m json.tool
  echo ""
  echo "Services in history:"
  curl -sf "http://localhost:8000/api/v1/services" | python3 -m json.tool
else
  echo "(History API not running — start with: make dhrishti)"
fi

echo ""
echo "Saved k6 summary → ${K6_SUMMARY}"
