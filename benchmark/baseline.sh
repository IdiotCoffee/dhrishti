#!/usr/bin/env bash
set -euo pipefail

# Full flash-sale load test — Dhrishti must be OFF.
#
# Usage:
#   ./baseline.sh
#   sudo CLIENTS=500 ./baseline.sh    # sudo = multi-IP clients in sidebar

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=_lib.sh
source "${SCRIPT_DIR}/_lib.sh"

ensure_root "$@"

BASELINE_K6="${BASELINE_K6:-/tmp/dhrishti-baseline-k6.json}"
RUN_PROFILE="${RUN_PROFILE:-/tmp/dhrishti-benchmark-env.txt}"

check_deps
check_stack

if curl -sf "${DHRISHTI_URL}/metrics" >/dev/null 2>&1; then
  echo "ERROR: Dhrishti is running at ${DHRISHTI_URL}"
  echo "Stop it first:  Ctrl+C on 'sudo go run main.go'"
  exit 1
fi

trap stop_fleet EXIT

echo "=============================================="
echo "  BASELINE — flash sale (Dhrishti OFF)"
echo "=============================================="
echo "  clients (load scale): ${CLIENTS}"
echo "  client IPs (fleet):   $(fleet_client_ip_list)"
echo "  gateway:              ${BASE_URL}"
echo ""
echo "  Timeline: browse/search → flash rush → checkout (~4 min)"
echo "=============================================="
echo ""

save_run_profile "${RUN_PROFILE}"
start_fleet
run_k6 "${BASELINE_K6}"

echo ""
echo "=== Baseline results ==="
print_k6_summary "${BASELINE_K6}"
echo ""
echo "Saved → ${BASELINE_K6}"
echo ""
echo "Next: start Dhrishti (sudo go run main.go) then run  ./benchmark.sh"
