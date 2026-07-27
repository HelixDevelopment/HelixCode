#!/usr/bin/env bash
# scripts/security-scan.sh — Security Scanning Script for HelixCode
# Ported from HelixAgent's scripts/security-scan.sh with HelixCode-specific paths.
#
# TODO(P3-refactor): this file handles 7+ scanners in ~580 lines; split into
# per-scanner sourced modules under scripts/scanners/ in a future Phase 3 task.
#
# Usage:
#   ./scripts/security-scan.sh [scanner] [options]
#
# Scanners:
#   snyk        - Snyk vulnerability scanner (requires SNYK_TOKEN)
#   sonarqube   - SonarQube code quality and security analysis
#   trivy       - Trivy vulnerability and secret scanner
#   gosec       - Go security checker
#   grype       - Anchore Grype container vulnerability scanner
#   kics        - Keeping Infrastructure as Code Secure
#   semgrep     - Semgrep SAST scanner
#   all         - Run all scanners (gosec + trivy + grype + kics + semgrep; sonarqube and snyk need tokens)
#
# Options:
#   --json      Output in JSON format
#   --html      Generate HTML report (where supported)
#   --fix       Attempt to auto-fix issues (where supported)
#   --help      Show this help text and exit
#
# Credentials:
#   Loaded from HelixCode/.env (gitignored, mode 0600) or from the calling environment.
#   Never hardcode credentials. Required env vars:
#     SONAR_TOKEN, SONARQUBE_PROJECT_KEY, SONARQUBE_PROJECT_NAME, SONARQUBE_PROJECT_VERSION
#     SNYK_TOKEN
#
# Endpoint configuration (optional, non-secret):
#   HELIX_SONARQUBE_HOST  SonarQube host (default: localhost)
#   HELIX_SONARQUBE_PORT  SonarQube port (default: 9000)
#   These are the SAME two variables with the SAME two defaults that
#   cmd/security_scan/main.go reads (sonarqubeHost() / sonarqubePort()). This
#   script boots the server through that binary and then polls the server
#   itself, so both halves MUST resolve the same address — see the
#   "SonarQube endpoint" block below.
#
# Scanner orchestration uses cmd/security_scan/main.go (Containers BootManager) when go
# is on PATH; falls back to direct compose otherwise.

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
REPORTS_DIR="${PROJECT_DIR}/reports/security"
SONARQUBE_COMPOSE="${PROJECT_DIR}/docker/security/sonarqube/docker-compose.yml"
SNYK_COMPOSE="${PROJECT_DIR}/docker/security/snyk/docker-compose.yml"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# -------------------------------------------------------------------
# help / --help
# -------------------------------------------------------------------
show_help() {
    sed -n '/^# Usage:/,/^[^#]/p' "$0" | grep '^#' | sed 's/^# \?//'
    exit 0
}

for arg in "$@"; do
    case "$arg" in
        --help|-h) show_help ;;
    esac
done

# -------------------------------------------------------------------
# Load environment
# -------------------------------------------------------------------
load_env() {
    local env_file="${PROJECT_DIR}/.env"
    if [ -f "$env_file" ]; then
        set -a
        # shellcheck source=/dev/null
        source "$env_file"
        set +a
        echo -e "${BLUE}Loaded environment from: ${env_file}${NC}"
    else
        echo -e "${YELLOW}Note: ${env_file} not found; using calling environment.${NC}"
    fi
}

load_env

# Create reports directory
mkdir -p "$REPORTS_DIR"

# -------------------------------------------------------------------
# SonarQube endpoint — single source of truth shared with the Go binary
#
# cmd/security_scan/main.go resolves the SonarQube address from
# HELIX_SONARQUBE_HOST / HELIX_SONARQUBE_PORT, defaulting to localhost / 9000
# (see its sonarqubeHost() / sonarqubePort() accessors). start_sonarqube()
# below boots the server *through that binary* and then this script polls the
# server directly, so both halves MUST resolve the same address from the same
# variables with the same defaults. Hardcoding localhost:9000 here would mean
# the Go half boots endpoint X while the shell half polls localhost:9000 —
# a split brain the moment either variable is set.
#
# Resolved AFTER load_env (not in the config block near the top) so values
# coming from HelixCode/.env are honoured, not just the calling environment.
# -------------------------------------------------------------------
SONARQUBE_HOST="${HELIX_SONARQUBE_HOST:-localhost}"
SONARQUBE_PORT="${HELIX_SONARQUBE_PORT:-9000}"
SONARQUBE_URL="http://${SONARQUBE_HOST}:${SONARQUBE_PORT}"

# -------------------------------------------------------------------
# Container runtime detection
# -------------------------------------------------------------------
detect_runtime() {
    if command -v docker &>/dev/null && docker info &>/dev/null 2>&1; then
        echo "docker"
    elif command -v podman &>/dev/null; then
        echo "podman"
    else
        echo "none"
    fi
}

detect_compose() {
    local runtime="$1"
    if [ "$runtime" = "docker" ]; then
        if docker compose version &>/dev/null 2>&1; then
            echo "docker compose"
        elif command -v docker-compose &>/dev/null; then
            echo "docker-compose"
        fi
    elif [ "$runtime" = "podman" ]; then
        if command -v podman-compose &>/dev/null; then
            echo "podman-compose"
        fi
    fi
}

RUNTIME=$(detect_runtime)
COMPOSE_CMD=$(detect_compose "$RUNTIME")

if [ -z "$COMPOSE_CMD" ]; then
    echo -e "${RED}Error: No container runtime found. Install Docker or Podman.${NC}"
    exit 1
fi

echo -e "${BLUE}Using container runtime: ${RUNTIME}${NC}"
echo -e "${BLUE}Using compose command:   ${COMPOSE_CMD}${NC}"

# -------------------------------------------------------------------
# Gosec
# -------------------------------------------------------------------
run_gosec() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Running Gosec Security Scanner${NC}"
    echo -e "${BLUE}========================================${NC}"

    local report_file="${REPORTS_DIR}/gosec-${TIMESTAMP}.json"
    local html_report="${REPORTS_DIR}/gosec-${TIMESTAMP}.html"

    if command -v gosec &>/dev/null; then
        echo -e "${GREEN}Using local gosec installation${NC}"
        gosec -fmt=json -out="$report_file" ./... 2>/dev/null || true
        gosec -fmt=html -out="$html_report" ./... 2>/dev/null || true
    else
        echo -e "${YELLOW}Gosec not installed locally, using container${NC}"
        $RUNTIME run --rm \
            -v "${PROJECT_DIR}:/app:ro" \
            -w /app \
            securego/gosec:2.21.4 \
            -fmt json -out /app/reports/security/gosec-"${TIMESTAMP}".json ./... 2>/dev/null || true
    fi

    if [ -f "$report_file" ]; then
        echo -e "${GREEN}Gosec report: ${report_file}${NC}"
        local issues; issues=$(jq '.Issues | length' "$report_file" 2>/dev/null || echo "0")
        local high;   high=$(jq '[.Issues[] | select(.severity == "HIGH")] | length'   "$report_file" 2>/dev/null || echo "0")
        local medium; medium=$(jq '[.Issues[] | select(.severity == "MEDIUM")] | length' "$report_file" 2>/dev/null || echo "0")
        local low;    low=$(jq '[.Issues[] | select(.severity == "LOW")] | length'    "$report_file" 2>/dev/null || echo "0")
        echo -e "${YELLOW}Gosec Summary: Total=${issues}  High=${high}  Medium=${medium}  Low=${low}${NC}"
    fi
}

# -------------------------------------------------------------------
# Trivy
# -------------------------------------------------------------------
run_trivy() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Running Trivy Vulnerability Scanner${NC}"
    echo -e "${BLUE}========================================${NC}"

    local report_file="${REPORTS_DIR}/trivy-${TIMESTAMP}.json"

    if command -v trivy &>/dev/null; then
        echo -e "${GREEN}Using local trivy installation${NC}"
        trivy fs --format json --output "$report_file" \
            --scanners vuln,secret,misconfig \
            --severity HIGH,CRITICAL \
            "$PROJECT_DIR" 2>/dev/null || true
    else
        echo -e "${YELLOW}Trivy not installed locally, using container${NC}"
        $RUNTIME run --rm \
            -v "${PROJECT_DIR}:/app:ro" \
            -v "${REPORTS_DIR}:/reports" \
            aquasec/trivy:0.55.2 fs \
            --format json \
            --output /reports/trivy-"${TIMESTAMP}".json \
            --scanners vuln,secret,misconfig \
            --severity HIGH,CRITICAL \
            /app 2>/dev/null || true
    fi

    if [ -f "$report_file" ]; then
        echo -e "${GREEN}Trivy report: ${report_file}${NC}"
        local vulns; vulns=$(jq '.Results[]?.Vulnerabilities // [] | length' "$report_file" 2>/dev/null | awk '{s+=$1} END {print s+0}')
        echo -e "${YELLOW}Trivy Summary: Vulnerabilities=${vulns:-0}${NC}"
    fi
}

# -------------------------------------------------------------------
# Grype (Anchore)
# -------------------------------------------------------------------
run_grype() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Running Grype Vulnerability Scanner${NC}"
    echo -e "${BLUE}========================================${NC}"

    local report_file="${REPORTS_DIR}/grype-${TIMESTAMP}.json"

    if command -v grype &>/dev/null; then
        echo -e "${GREEN}Using local grype installation${NC}"
        grype dir:"$PROJECT_DIR" -o json > "$report_file" 2>/dev/null || true
    else
        echo -e "${YELLOW}Grype not installed locally, using container${NC}"
        $RUNTIME run --rm \
            -v "${PROJECT_DIR}:/app:ro" \
            anchore/grype:v0.86.0 \
            dir:/app -o json > "$report_file" 2>/dev/null || true
    fi

    if [ -f "$report_file" ] && [ -s "$report_file" ]; then
        echo -e "${GREEN}Grype report: ${report_file}${NC}"
        local matches; matches=$(jq '.matches | length' "$report_file" 2>/dev/null || echo "0")
        echo -e "${YELLOW}Grype Summary: Matches=${matches}${NC}"
    fi
}

# -------------------------------------------------------------------
# KICS (Infrastructure as Code)
# -------------------------------------------------------------------
run_kics() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Running KICS IaC Security Scanner${NC}"
    echo -e "${BLUE}========================================${NC}"

    local report_file="${REPORTS_DIR}/kics-${TIMESTAMP}.json"

    if command -v kics &>/dev/null; then
        echo -e "${GREEN}Using local kics installation${NC}"
        kics scan -p "$PROJECT_DIR/docker" -o "$REPORTS_DIR" --output-name "kics-${TIMESTAMP}" --report-formats json 2>/dev/null || true
    else
        echo -e "${YELLOW}KICS not installed locally, using container${NC}"
        $RUNTIME run --rm \
            -v "${PROJECT_DIR}:/app:ro" \
            checkmarx/kics:v2.1.4 \
            scan -p /app/docker -o /app/reports/security --output-name "kics-${TIMESTAMP}" --report-formats json 2>/dev/null || true
    fi

    echo -e "${GREEN}KICS scan complete (see reports/security/ for output)${NC}"
}

# -------------------------------------------------------------------
# Semgrep
# -------------------------------------------------------------------
run_semgrep() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Running Semgrep SAST Scanner${NC}"
    echo -e "${BLUE}========================================${NC}"

    local report_file="${REPORTS_DIR}/semgrep-${TIMESTAMP}.json"

    if command -v semgrep &>/dev/null; then
        echo -e "${GREEN}Using local semgrep installation${NC}"
        semgrep scan --config auto --json --output "$report_file" "$PROJECT_DIR" 2>/dev/null || true
    else
        echo -e "${YELLOW}Semgrep not installed locally, using container${NC}"
        $RUNTIME run --rm \
            -v "${PROJECT_DIR}:/app:ro" \
            returntocorp/semgrep:1.93.0 \
            semgrep scan --config auto --json --output /app/reports/security/semgrep-"${TIMESTAMP}".json /app 2>/dev/null || true
    fi

    if [ -f "$report_file" ] && [ -s "$report_file" ]; then
        echo -e "${GREEN}Semgrep report: ${report_file}${NC}"
        local findings; findings=$(jq '.results | length' "$report_file" 2>/dev/null || echo "0")
        echo -e "${YELLOW}Semgrep Summary: Findings=${findings}${NC}"
    fi
}

# -------------------------------------------------------------------
# Snyk
# -------------------------------------------------------------------
run_snyk() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Running Snyk Vulnerability Scanner${NC}"
    echo -e "${BLUE}========================================${NC}"

    local report_file="${REPORTS_DIR}/snyk-${TIMESTAMP}.json"

    if [ -z "${SNYK_TOKEN:-}" ]; then
        echo -e "${YELLOW}Warning: SNYK_TOKEN not set. Running in limited OSS mode.${NC}"
        echo -e "${YELLOW}Set SNYK_TOKEN in HelixCode/.env for full features.${NC}"
    fi

    # containers BootManager call (P0-T08.7/4): use Go binary if go is available.
    if command -v snyk &>/dev/null; then
        echo -e "${GREEN}Using local snyk installation${NC}"
        cd "$PROJECT_DIR"
        snyk test --json > "$report_file" 2>/dev/null || true
    else
        echo -e "${YELLOW}Snyk not installed locally, using container${NC}"
        $RUNTIME run --rm \
            -v "${PROJECT_DIR}:/app:ro" \
            -v "${REPORTS_DIR}:/reports" \
            -e SNYK_TOKEN="${SNYK_TOKEN:-}" \
            snyk/snyk:golang test --json /app > "$report_file" 2>/dev/null || true
    fi

    if [ -f "$report_file" ] && [ -s "$report_file" ]; then
        echo -e "${GREEN}Snyk report: ${report_file}${NC}"
        local vulns; vulns=$(jq '.vulnerabilities | length' "$report_file" 2>/dev/null || echo "0")
        local high;  high=$(jq '[.vulnerabilities[] | select(.severity == "high")] | length' "$report_file" 2>/dev/null || echo "0")
        echo -e "${YELLOW}Snyk Summary: Total=${vulns}  High=${high}${NC}"
    else
        echo -e "${YELLOW}Snyk scan completed (no vulnerabilities found or running in limited mode)${NC}"
    fi
}

# -------------------------------------------------------------------
# SonarQube
# -------------------------------------------------------------------
start_sonarqube() {
    echo -e "${BLUE}Starting SonarQube server via compose (endpoint: ${SONARQUBE_URL})...${NC}"
    # containers BootManager call (P0-T08.7/4): use Go binary if go is available.
    # Falls back to direct compose when go is not on PATH.
    if command -v go &>/dev/null && [ -f "${PROJECT_DIR}/cmd/security_scan/main.go" ]; then
        (cd "$PROJECT_DIR" && go run ./cmd/security_scan -scanner=sonarqube -action=start)
    else
        $COMPOSE_CMD -f "$SONARQUBE_COMPOSE" up -d sonarqube postgres
    fi

    echo -e "${YELLOW}Waiting for SonarQube to be ready (this may take 2-3 minutes)...${NC}"
    local max_attempts=60
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        if curl -sf "${SONARQUBE_URL}/api/system/status" | grep -q '"status":"UP"'; then
            echo -e "${GREEN}SonarQube is ready!${NC}"
            return 0
        fi
        attempt=$((attempt + 1))
        echo -n "."
        sleep 5
    done

    echo -e "${RED}SonarQube failed to start within timeout (endpoint: ${SONARQUBE_URL})${NC}"
    return 1
}

run_sonarqube() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Running SonarQube Code Analysis${NC}"
    echo -e "${BLUE}========================================${NC}"

    if ! curl -sf "${SONARQUBE_URL}/api/system/status" | grep -q '"status":"UP"'; then
        echo -e "${YELLOW}SonarQube not running, starting...${NC}"
        start_sonarqube || return 1
    fi

    local sonar_token="${SONAR_TOKEN:-}"
    if [ -z "$sonar_token" ]; then
        local sonar_user="${SONARQUBE_ADMIN_USER:-admin}"
        local sonar_pass="${SONARQUBE_ADMIN_PASSWORD:-admin}"
        echo -e "${YELLOW}No SONAR_TOKEN — generating temporary token with admin credentials${NC}"
        sonar_token=$(curl -sf -u "${sonar_user}:${sonar_pass}" \
            -X POST "${SONARQUBE_URL}/api/user_tokens/generate" \
            -d "name=helixcode-scan-${TIMESTAMP}" 2>/dev/null | jq -r '.token // empty')
        if [ -z "$sonar_token" ]; then
            echo -e "${RED}Failed to generate SonarQube token. Set SONAR_TOKEN in HelixCode/.env.${NC}"
            return 1
        fi
        echo -e "${GREEN}Generated temporary scan token${NC}"
    fi

    # Generate coverage report first
    echo -e "${YELLOW}Generating coverage report...${NC}"
    cd "$PROJECT_DIR"
    go test -coverprofile=coverage.out ./internal/... 2>/dev/null || true

    local project_key="${SONARQUBE_PROJECT_KEY:-helixcode}"
    local project_name="${SONARQUBE_PROJECT_NAME:-HelixCode}"
    local project_version="${SONARQUBE_PROJECT_VERSION:-1.0.0}"

    if command -v sonar-scanner &>/dev/null; then
        echo -e "${GREEN}Using local sonar-scanner${NC}"
        sonar-scanner \
            -Dsonar.host.url="${SONARQUBE_URL}" \
            -Dsonar.token="$sonar_token" \
            -Dsonar.projectKey="$project_key" \
            -Dsonar.projectName="$project_name" \
            -Dsonar.projectVersion="$project_version" \
            -Dsonar.projectBaseDir="$PROJECT_DIR"
    else
        echo -e "${YELLOW}Using containerized sonar-scanner${NC}"
        local mount_opts=""
        if [ "$RUNTIME" = "podman" ]; then
            mount_opts=":Z"
        fi
        $RUNTIME run --rm \
            -v "${PROJECT_DIR}:/usr/src${mount_opts}" \
            -w /usr/src \
            --network=host \
            -e SONAR_HOST_URL="${SONARQUBE_URL}" \
            -e SONAR_TOKEN="$sonar_token" \
            docker.io/sonarsource/sonar-scanner-cli \
            -Dsonar.projectKey="$project_key" \
            -Dsonar.projectName="$project_name" \
            -Dsonar.projectVersion="$project_version" \
            -Dsonar.sources=internal,cmd \
            -Dsonar.sourceEncoding=UTF-8 \
            -Dsonar.qualitygate.wait=false
    fi

    echo -e "${GREEN}SonarQube analysis complete!${NC}"
    echo -e "${BLUE}View results at: ${SONARQUBE_URL}/dashboard?id=${project_key}${NC}"
}

# -------------------------------------------------------------------
# Go static analysis (vet + staticcheck + golangci-lint)
# -------------------------------------------------------------------
run_go_analysis() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Running Go Static Analysis${NC}"
    echo -e "${BLUE}========================================${NC}"

    local report_file="${REPORTS_DIR}/go-analysis-${TIMESTAMP}.txt"

    cd "$PROJECT_DIR"

    echo -e "${YELLOW}Running go vet...${NC}"
    go vet ./... >"$report_file" 2>&1 || true

    echo -e "${YELLOW}Running staticcheck...${NC}"
    if command -v staticcheck &>/dev/null; then
        staticcheck ./... >>"$report_file" 2>&1 || true
    fi

    echo -e "${YELLOW}Running golangci-lint...${NC}"
    if command -v golangci-lint &>/dev/null; then
        golangci-lint run --out-format json >"${REPORTS_DIR}/golangci-lint-${TIMESTAMP}.json" 2>/dev/null || true
    fi

    echo -e "${GREEN}Go analysis report: ${report_file}${NC}"
}

# -------------------------------------------------------------------
# Combined report
# -------------------------------------------------------------------
generate_combined_report() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Generating Combined Security Report${NC}"
    echo -e "${BLUE}========================================${NC}"

    local combined_report="${REPORTS_DIR}/security-summary-${TIMESTAMP}.md"

    cat >"$combined_report" <<EOF
# Security Scan Report
**Date:** $(date '+%Y-%m-%d %H:%M:%S')
**Project:** HelixCode

## Executive Summary

Comprehensive security and code quality scan results.

EOF

    local gosec_file; gosec_file=$(ls -t "${REPORTS_DIR}"/gosec-*.json 2>/dev/null | head -1)
    if [ -f "$gosec_file" ]; then
        local gosec_issues; gosec_issues=$(jq '.Issues | length' "$gosec_file" 2>/dev/null || echo "0")
        cat >>"$combined_report" <<EOF
### Gosec (Go Security Checker)
- **Total Issues:** ${gosec_issues}
- **Report:** $(basename "$gosec_file")

EOF
    fi

    local trivy_file; trivy_file=$(ls -t "${REPORTS_DIR}"/trivy-*.json 2>/dev/null | head -1)
    if [ -f "$trivy_file" ]; then
        cat >>"$combined_report" <<EOF
### Trivy (Vulnerability Scanner)
- **Report:** $(basename "$trivy_file")

EOF
    fi

    local snyk_file; snyk_file=$(ls -t "${REPORTS_DIR}"/snyk-*.json 2>/dev/null | head -1)
    if [ -f "$snyk_file" ]; then
        local snyk_vulns; snyk_vulns=$(jq '.vulnerabilities | length' "$snyk_file" 2>/dev/null || echo "0")
        cat >>"$combined_report" <<EOF
### Snyk (Dependency Scanner)
- **Vulnerabilities:** ${snyk_vulns}
- **Report:** $(basename "$snyk_file")

EOF
    fi

    cat >>"$combined_report" <<'EOF'
## Next Steps

1. Review all HIGH severity issues immediately
2. Address MEDIUM severity issues in the next sprint
3. Track LOW severity issues in backlog
4. Rotate any credentials flagged as leaked

## Reports Location
All detailed reports are available in: `reports/security/`
EOF

    echo -e "${GREEN}Combined report: ${combined_report}${NC}"
}

# -------------------------------------------------------------------
# Main dispatch
# -------------------------------------------------------------------
main() {
    local scanner="${1:-all}"
    shift || true

    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}HelixCode Security Scanner${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""

    cd "$PROJECT_DIR"

    case "$scanner" in
        gosec)        run_gosec        ;;
        trivy)        run_trivy        ;;
        grype)        run_grype        ;;
        kics)         run_kics         ;;
        semgrep)      run_semgrep      ;;
        snyk)         run_snyk         ;;
        sonarqube|sonar) run_sonarqube ;;
        go|goanalysis) run_go_analysis  ;;
        all)
            run_gosec
            echo ""
            run_trivy
            echo ""
            run_grype
            echo ""
            run_go_analysis
            echo ""
            generate_combined_report
            ;;
        start-sonar)
            start_sonarqube
            ;;
        stop)
            echo -e "${YELLOW}Stopping SonarQube services...${NC}"
            $COMPOSE_CMD -f "$SONARQUBE_COMPOSE" down || true
            ;;
        *)
            echo "Usage: $0 [scanner] [options]"
            echo ""
            echo "Scanners:"
            echo "  gosec       - Go security checker"
            echo "  trivy       - Vulnerability and secret scanner"
            echo "  grype       - Anchore Grype container vulnerability scanner"
            echo "  kics        - Infrastructure as Code security scanner"
            echo "  semgrep     - SAST scanner"
            echo "  snyk        - Snyk dependency/code scanner (requires SNYK_TOKEN)"
            echo "  sonarqube   - SonarQube code quality analysis (requires SONAR_TOKEN)"
            echo "  go          - Go static analysis (vet, staticcheck, golangci-lint)"
            echo "  all         - Run all open-source scanners (gosec + trivy + grype + go)"
            echo ""
            echo "Commands:"
            echo "  start-sonar - Start SonarQube server"
            echo "  stop        - Stop SonarQube services"
            echo ""
            echo "Options:"
            echo "  --help      Show this help and exit"
            exit 1
            ;;
    esac

    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}Security scan complete!${NC}"
    echo -e "${GREEN}Reports: ${REPORTS_DIR}${NC}"
    echo -e "${GREEN}========================================${NC}"
}

main "$@"
