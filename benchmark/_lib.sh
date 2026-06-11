# Shared logic for baseline.sh and benchmark.sh — do not run directly.

: "${SCRIPT_DIR:=$(cd "$(dirname "${BASH_SOURCE[1]:-${BASH_SOURCE[0]}}")" && pwd)}"

CLIENTS="${CLIENTS:-500}"
BASE_URL="${BASE_URL:-http://localhost:8080}"
GATEWAY_HEALTH="${BASE_URL}/health"
DHRISHTI_URL="${DHRISHTI_URL:-http://localhost:8090}"
K6_SCRIPT="${SCRIPT_DIR}/flash_sale.js"
FLEET_SCRIPT="${SCRIPT_DIR}/client_fleet.py"
FLEET_PID=""
GATEWAY_CONTAINER_IP=""
DOCKER_BRIDGE_IF=""
CLIENT_IP_OCTETS=""

# Re-exec with sudo so client fleet can bind distinct IPs on the docker bridge.
ensure_root() {
  if [[ "${EUID}" -ne 0 ]]; then
    if [[ "${DHRISHTI_BENCHMARK_REEXEC:-}" == "1" ]]; then
      echo "ERROR: root required for multi-client IP simulation."
      echo "Run:  sudo $0"
      exit 1
    fi
    export DHRISHTI_BENCHMARK_REEXEC=1
    exec sudo -E env DHRISHTI_BENCHMARK_REEXEC=1 "$0" "$@"
  fi
}

# Distinct client IPs for Dhrishti sidebar (capped; k6 uses CLIENTS for volume).
fleet_ips() {
  if [[ -n "${CONNECTING_IPS:-}" ]]; then
    echo "${CONNECTING_IPS}"
    return
  fi
  local n=$(( CLIENTS / 50 ))
  if [[ "${n}" -lt 5 ]]; then n=5; fi
  if [[ "${n}" -gt 20 ]]; then n=20; fi
  echo "${n}"
}

fleet_client_ip_list() {
  local n start i prefix ips
  n="$(fleet_ips)"
  prefix="${CLIENT_IP_OCTETS:-10.254.0}"
  start="${CLIENT_IP_START:-200}"
  ips=""
  for ((i = 0; i < n; i++)); do
    ips+="${prefix}.$((start + i)),"
  done
  echo "${ips%,}"
}

# Hit api-gateway by container IP (not localhost) so eBPF sees real client source IPs.
resolve_stack_urls() {
  local cid gw_ip net_id

  cid="$(docker ps -q -f 'label=com.docker.compose.service=api-gateway' 2>/dev/null | head -1)"
  if [[ -z "${cid}" ]]; then
    echo "Note: api-gateway container not found — using ${BASE_URL}"
    return 0
  fi

  gw_ip="$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "${cid}" 2>/dev/null | head -1)"
  if [[ -z "${gw_ip}" ]]; then
    return 0
  fi

  GATEWAY_CONTAINER_IP="${gw_ip}"
  BASE_URL="http://${gw_ip}:8080"
  GATEWAY_HEALTH="${BASE_URL}/health"
  CLIENT_IP_OCTETS="$(echo "${gw_ip}" | cut -d. -f1-3)"

  net_id="$(docker inspect -f '{{range $k,$v := .NetworkSettings.Networks}}{{$v.NetworkID}}{{end}}' "${cid}" 2>/dev/null | head -1)"
  if [[ -n "${net_id}" ]]; then
    DOCKER_BRIDGE_IF="br-${net_id:0:12}"
  fi
}

check_deps() {
  if ! command -v k6 >/dev/null 2>&1; then
    echo "k6 not found. Install:  yay -S k6-bin"
    exit 1
  fi
  if ! command -v python3 >/dev/null 2>&1; then
    echo "python3 required"
    exit 1
  fi
  if ! command -v docker >/dev/null 2>&1; then
    echo "docker required (to resolve api-gateway container IP)"
    exit 1
  fi
}

check_stack() {
  resolve_stack_urls
  if ! curl -sf "${GATEWAY_HEALTH}" >/dev/null 2>&1; then
    echo "Mock stack not reachable at ${BASE_URL}"
    echo "Start it:  cd mock_services && docker compose up -d"
    exit 1
  fi
}

stop_fleet() {
  if [[ -n "${FLEET_PID}" ]] && kill -0 "${FLEET_PID}" 2>/dev/null; then
    kill "${FLEET_PID}" 2>/dev/null || true
    wait "${FLEET_PID}" 2>/dev/null || true
  fi
  FLEET_PID=""
}

start_fleet() {
  local n ips rps bridge_if
  n="$(fleet_ips)"
  ips="$(fleet_client_ip_list)"
  bridge_if="${DOCKER_BRIDGE_IF:-lo}"
  rps=$(( CLIENTS / 100 ))
  if [[ "${rps}" -lt 2 ]]; then rps=2; fi
  if [[ "${rps}" -gt 8 ]]; then rps=8; fi

  if [[ ! -f "${FLEET_SCRIPT}" ]]; then
    return 0
  fi

  echo "Starting ${n} simulated client IPs on ${bridge_if} → ${BASE_URL}"
  echo "  IPs: ${ips}"
  fleet_duration="${FLEET_DURATION:-280}"
  NUM_CLIENTS="${n}" \
    CLIENT_IPS="${ips}" \
    BRIDGE_IF="${bridge_if}" \
    RPS_PER_CLIENT="${rps}" \
    DURATION="${fleet_duration}" \
    BASE_URL="${BASE_URL}" \
    python3 "${FLEET_SCRIPT}" &
  FLEET_PID=$!
}

run_k6() {
  local summary="$1" script="${2:-${K6_SCRIPT}}" code
  export CLIENTS BASE_URL DURATION
  k6 run --summary-export "${summary}" "${script}" || {
    code=$?
    # k6 exit 99 = thresholds crossed; run still produced a valid summary.
    if [[ "${code}" -eq 99 ]]; then
      echo ""
      echo "Note: k6 latency threshold crossed (stack was saturated — summary still saved)"
      return 0
    fi
    return "${code}"
  }
}

print_k6_summary() {
  python3 "${SCRIPT_DIR}/k6_parse.py" "$1"
}

save_run_profile() {
  local path="$1"
  cat > "${path}" <<EOF
CLIENTS=${CLIENTS}
BASE_URL=${BASE_URL}
GATEWAY_CONTAINER_IP=${GATEWAY_CONTAINER_IP}
FLEET_IPS=$(fleet_ips)
FLEET_CLIENT_IPS=$(fleet_client_ip_list)
EOF
}
