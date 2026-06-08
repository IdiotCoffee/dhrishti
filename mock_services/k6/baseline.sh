#!/usr/bin/env bash
set -euo pipefail

# Baseline load test — same k6 script and env as benchmark.sh, but Dhrishti OFF.
#
# Usage (match env vars with benchmark.sh):
#   ./baseline.sh
#   VUS=10 DURATION=2m ./baseline.sh
#
# Then start Dhrishti and run:  VUS=10 DURATION=2m ./benchmark.sh
# Then compare:                 ./compare.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DHRISHTI_URL="${DHRISHTI_URL:-http://localhost:8090}"
BASELINE_K6="${BASELINE_K6:-/tmp/dhrishti-baseline-k6.json}"
BASELINE_ENV="${BASELINE_ENV:-/tmp/dhrishti-benchmark-env.txt}"

if ! command -v k6 >/dev/null 2>&1; then
  echo "k6 not found.  Arch: yay -S k6-bin"
  exit 1s
fi

if curl -sf "${DHRISHTI_URL}/metrics" >/dev/null 2>&1; then
  echo "WARNING: Dhrishti is running at ${DHRISHTI_URL}"
  echo "Stop it first for a clean baseline:  Ctrl+C on 'sudo go run main.go'"
  echo ""
  read -r -p "Continue anyway? [y/N] " ans
  [[ "${ans,,}" == "y" ]] || exit 1
fi

# Record env so compare.sh can verify benchmark used the same profile.
cat > "${BASELINE_ENV}" <<EOF
VUS=${VUS:-50000}
DURATION=${DURATION:-2m}
RAMP_UP=${RAMP_UP:-50s}
RAMP_DOWN=${RAMP_DOWN:-10s}
EXECUTOR=${EXECUTOR:-ramping}
RATE=${RATE:-5}
SLEEP_MIN=${SLEEP_MIN:-0.5}
SLEEP_MAX=${SLEEP_MAX:-1.5}
BASE_URL=${BASE_URL:-http://localhost:8080}
EOF

echo "=== Baseline run (Dhrishti OFF) ==="
echo "Saved env → ${BASELINE_ENV}"
cat "${BASELINE_ENV}"
echo ""

echo "=== Running k6 (same load.js as benchmark.sh) ==="
k6 run --summary-export "${BASELINE_K6}" "${SCRIPT_DIR}/load.js"

echo ""
echo "=== Baseline k6 results ==="
python3 "${SCRIPT_DIR}/k6_parse.py" "${BASELINE_K6}"

echo ""
echo "Summary saved → ${BASELINE_K6}"
echo ""
echo "Next steps:"
echo "  1. Start Dhrishti:  sudo go run main.go   (from repo root)"
echo "  2. Same env:        $(grep -v BASE_URL "${BASELINE_ENV}" | tr '\n' ' ') ./benchmark.sh"
echo "  3. Compare:         ./compare.sh"
