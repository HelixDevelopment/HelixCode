#!/usr/bin/env bash
#
# validate-helix-platform.sh — Post-boot health and readiness verification.
#
# Verifies every systemd user unit is active and every exposed service/port is
# reachable. Writes a timestamped report under qa-results/helix-platform-validation/.
#
# Exit code:
#   0  all mandatory probes passed
#   1  at least one mandatory probe failed
#
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TS="$(date -u +%Y%m%dT%H%M%SZ)"
EVIDENCE_DIR="${REPO_ROOT}/qa-results/helix-platform-validation/${TS}"
mkdir -p "$EVIDENCE_DIR"

REPORT_TXT="${EVIDENCE_DIR}/report.txt"
REPORT_JSON="${EVIDENCE_DIR}/report.json"

declare -A MANDATORY_UNITS
declare -A MANDATORY_PORTS
declare -A OPTIONAL_PORTS

MANDATORY_UNITS=(
  [helix.target]=target
  [helixcode-infra.service]=service
  [llmsverifier.service]=service
  [helixllm-gateway.service]=service
  [helixagent.service]=service
  [helixcode-server.service]=service
  [helixllm-coder.service]=service
)

# service -> "host:port|path|probe_type"
# probe_type: http, tcp, skip
MANDATORY_PORTS=(
  [helixcode-server]=":8081|/health|http"
  [helixagent]=":7061||tcp"
  [llmsverifier]=":8100|/api/scores|http"
  [helixllm-gateway]=":8443|/internal/health|https"
  [helixllm-coder]=":18434|/v1/models|http"
)

OPTIONAL_PORTS=(
  [postgres]=":5433||tcp"
  [redis]=":6380||tcp"
  [ollama]=":11434|/api/tags|http"
  [weaviate]=":8083|/v1/.well-known/ready|http"
  [chromadb]=":8082|/api/v2/heartbeat|http"
  [qdrant]=":6333|/readyz|http"
  [cognee]=":8000|/health|http"
  [memcached]=":11211||tcp"
  [selenium]=":4444|/status|http"
  [chromedp]=":9222|/json/version|http"
)

log()    { printf '[validate] %s\n' "$*"; }
ok()     { printf '  \033[32m✓\033[0m %s\n' "$*"; }
fail()   { printf '  \033[31m✗\033[0m %s\n' "$*"; }
warn()   { printf '  \033[33m!\033[0m %s\n' "$*"; }

check_unit() {
  local unit="$1" kind="$2"
  local state
  state="$(systemctl --user show -p ActiveState --value "$unit" 2>/dev/null || echo unknown)"
  printf '{"unit":"%s","kind":"%s","state":"%s"}\n' "$unit" "$kind" "$state" >> "$REPORT_JSON.unit-stream"
  if [ "$state" = "active" ] || { [ "$kind" = "service" ] && [ "$state" = "exited" ]; }; then
    ok "$unit: $state"
    return 0
  else
    fail "$unit: $state (expected active/exited)"
    return 1
  fi
}

probe_port() {
  local name="$1" spec="$2"
  local host port path probe_type
  host="127.0.0.1"
  port="${spec%%|*}"
  port="${port##*:}"
  path="${spec#*|}"
  path="${path%%|*}"
  probe_type="${spec##*|}"

  if [ "$probe_type" = "skip" ]; then
    warn "$name: probe skipped by configuration"
    return 0
  fi

  if [ "$probe_type" = "tcp" ]; then
    if timeout 5 bash -c "exec 3<>/dev/tcp/${host}/${port}" 2>/dev/null; then
      ok "$name: tcp ${host}:${port} reachable"
      return 0
    else
      fail "$name: tcp ${host}:${port} unreachable"
      return 1
    fi
  fi

  local scheme="http"
  local curl_opts=(-s -o /dev/null -w '%{http_code}' --max-time 8)
  if [ "$probe_type" = "https" ]; then
    scheme="https"
    curl_opts+=(-k)
  fi

  local url="${scheme}://${host}:${port}${path}"
  local http_code
  http_code="$(curl "${curl_opts[@]}" "$url" 2>/dev/null || echo 000)"
  if [ "$http_code" = "200" ] || [ "$http_code" = "401" ] || [ "$http_code" = "403" ]; then
    ok "$name: ${url} -> HTTP ${http_code}"
    return 0
  else
    fail "$name: ${url} -> HTTP ${http_code}"
    return 1
  fi
}

main() {
  log "Helix platform validation starting at ${TS}"
  log "Evidence directory: ${EVIDENCE_DIR}"
  echo

  : > "$REPORT_JSON.unit-stream"

  local overall_rc=0
  local failed=0

  log "=== systemd user units ==="
  for unit in "${!MANDATORY_UNITS[@]}"; do
    if ! check_unit "$unit" "${MANDATORY_UNITS[$unit]}"; then
      failed=1
    fi
  done
  echo

  log "=== mandatory service ports ==="
  for svc in "${!MANDATORY_PORTS[@]}"; do
    if ! probe_port "$svc" "${MANDATORY_PORTS[$svc]}"; then
      failed=1
    fi
  done
  echo

  log "=== optional infrastructure ports ==="
  for svc in "${!OPTIONAL_PORTS[@]}"; do
    if ! probe_port "$svc" "${OPTIONAL_PORTS[$svc]}"; then
      warn "$svc optional probe failed (not counted as fatal)"
    fi
  done
  echo

  # Build final JSON report
  {
    echo '{'
    echo "  \"timestamp\": \"${TS}\","
    echo "  \"repo_root\": \"${REPO_ROOT}\","
    echo '  "units": ['
    if [ -s "$REPORT_JSON.unit-stream" ]; then
      sed 's/$/,/' "$REPORT_JSON.unit-stream" | sed '$ s/,$//'
    fi
    echo '  ],'
    echo "  \"overall_pass\": $([ "$failed" -eq 0 ] && echo true || echo false)"
    echo '}'
  } > "$REPORT_JSON"
  rm -f "$REPORT_JSON.unit-stream"

  # Build human-readable report
  {
    echo "Helix Platform Validation Report"
    echo "Generated: ${TS}"
    echo "Repo: ${REPO_ROOT}"
    echo ""
    echo "Overall: $([ "$failed" -eq 0 ] && echo PASS || echo FAIL)"
    echo "Report JSON: ${REPORT_JSON}"
  } > "$REPORT_TXT"

  if [ "$failed" -ne 0 ]; then
    log "VALIDATION FAILED — see ${REPORT_TXT} and ${REPORT_JSON}"
    exit 1
  fi

  log "VALIDATION PASSED — see ${REPORT_TXT} and ${REPORT_JSON}"
  exit 0
}

main "$@"
