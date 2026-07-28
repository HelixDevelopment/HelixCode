#!/bin/bash
set -e

# HelixCode Comprehensive Test Runner
# Runs all tests across the entire codebase

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# The Go application lives in the inner module dir, NOT at the meta-repo root
# (CLAUDE.md §3.2.1). Directory name is lowercase snake_case per CONST-052.
INNER_MODULE_DIR="$PROJECT_ROOT/helix_code"

# ==============================================================================
# §11.4.119 — SINGLE-RESOURCE-OWNER: PORT-BINDING SWEEPS ARE **NOT** PARALLELISABLE
# ------------------------------------------------------------------------------
# DO NOT make module sweeps run concurrently. Partition parallelism by RESOURCE,
# not by repository. The host ephemeral port range is ONE exclusive resource
# shared by every checkout/track on this host.
#
# CAPTURED FORENSIC EVIDENCE (FACT, 2026-07-27):
#   /proc/sys/net/ipv4/ip_local_port_range = 32768 60999   (one shared range)
#   Two sweeps launched ~2 SECONDS apart contended for it:
#     qa-results/full_retest/helix_agent_20260727T133218Z.log
#     qa-results/full_retest/helix_code_inner_20260727T133220Z.log
#   Measured fallout:
#     - 112x "dial tcp: connect: cannot assign requested address" (helix_agent log)
#     - FAIL dev.helix.code/internal/discovery (10.728s), 7 top-level failures:
#         "no ports available in configured range" (port_allocator_test.go:375)
#         "listen tcp :0: bind: address already in use"
#         TestAllocatePort_RangeExhausted_WithEphemeral, TestConcurrentAllocations,
#         TestConcurrentReleases, TestCheckHTTPHealth,
#         TestCheckHTTPHealth_CustomEndpoint, TestCheckTCPHealth,
#         TestPerformHealthChecks_Integration
#   Counter-evidence the CODE is fine: the same package passes on a quiet host
#   ("ok  dev.helix.code/internal/discovery", captured in qa-results).
#
# => Concurrent port-binding sweeps MANUFACTURE PHANTOM CODE DEFECTS. A red
#    result produced under port contention is evidence of nothing (§11.4.119).
#
# The mutex below is an flock held for the whole process lifetime, so even two
# INDEPENDENT invocations of this script (different shells, different checkouts,
# different tracks) cannot overlap. Lint/build phases do not bind ports and are
# deliberately left OUTSIDE the lock — that is the resource partition, not an
# oversight. See scripts/lib/port_sweep_lock.sh for the full rationale.
# ==============================================================================
# shellcheck source=lib/port_sweep_lock.sh
. "$SCRIPT_DIR/lib/port_sweep_lock.sh"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

# NOTE: arithmetic MUST NOT use ((VAR++)). Under `set -e` (line 2) the
# post-increment of a zero-valued counter evaluates to 0, which bash reports as
# exit status 1, killing the script on the FIRST error()/success() call.
# Captured repro:
#   $ bash -c 'set -e; N=0; ((N++)); echo "reached: N=$N"'; echo "exit=$?"
#   exit=1          # "reached" never printed
# $((VAR + 1)) assignment always returns 0.
error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
    FAILED_TESTS=$((FAILED_TESTS + 1))
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
    PASSED_TESTS=$((PASSED_TESTS + 1))
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Check if we're in the right directory
check_environment() {
    if [ ! -f "${PROJECT_ROOT}/helix_code/go.mod" ]; then
        error "Not in HelixCode project root directory"
        exit 1
    fi
}

# Run tests for a specific package
run_package_tests() {
    local package_path="$1"
    local package_name="$2"
    # $3 — extra `go test` flags, space-separated (e.g. "-tags=ci -race").
    # $4 — timeout override; the 60s default is too short for -race runs.
    local extra_flags_raw="${3:-}"
    local pkg_timeout="${4:-60s}"

    local -a extra_flags=()
    if [ -n "$extra_flags_raw" ]; then
        read -r -a extra_flags <<< "$extra_flags_raw"
    fi

    log "Running tests for $package_name..."

    # Package paths are relative to the INNER Go module (CLAUDE.md §3.2.1), and
    # the existence check must be absolute — a relative check resolves against
    # whatever cwd the previous package left behind.
    local abs_package_path="$INNER_MODULE_DIR/$package_path"

    if [ -d "$abs_package_path" ]; then
        cd "$abs_package_path"

        if go test -v ${extra_flags[@]+"${extra_flags[@]}"} ./... -timeout "$pkg_timeout"; then
            success "$package_name tests passed"
        else
            error "$package_name tests failed"
        fi
    else
        warning "Package directory $package_path not found, skipping"
    fi
}

# Run all unit tests
run_unit_tests() {
    log "Running all unit tests..."

    cd "$INNER_MODULE_DIR"

    # Run tests for all packages
    run_package_tests "." "main"
    run_package_tests "internal/auth" "auth"
    run_package_tests "internal/config" "config"
    run_package_tests "internal/database" "database"
    run_package_tests "internal/hardware" "hardware"
    run_package_tests "internal/llm" "llm"
    run_package_tests "internal/mcp" "mcp"
    run_package_tests "internal/notification" "notification"
    run_package_tests "internal/project" "project"
    run_package_tests "internal/redis" "redis"
    run_package_tests "internal/server" "server"
    run_package_tests "internal/session" "session"
    run_package_tests "internal/task" "task"
    run_package_tests "internal/worker" "worker"
    run_package_tests "shared/mobile-core" "mobile-core"

    # Run tests for applications
    run_package_tests "applications/terminal_ui" "terminal-ui"

    # Fyne GUI applications — MUST carry -tags=ci and -race.
    #
    # -tags=ci selects Fyne's non-GL driver. Without it these packages do not
    # COMPILE on a host lacking X11/GL dev headers (`fatal error: X11/Xlib.h`),
    # so `desktop` and `aurora-os` — listed here since 2025-11 — have never
    # actually had their tests executed by this suite. The entries looked like
    # coverage while producing none.
    #
    # -race is not optional either: applications/{harmony_os,aurora_os}/
    # gui_thread_race_test.go and applications/desktop/main_racefix_test.go
    # exist to detect data races from off-main-goroutine Fyne widget mutation.
    # Without the detector they still exercise the code but cannot observe the
    # thing they were written to catch — a green run would mean nothing. These
    # guards measured 415 / 495 races on the pre-fix artifact and 0 on HEAD.
    #
    # 600s because a -race run of these packages takes ~60s each (~160s at
    # -count=3); the 60s default would time out and report a false failure
    # (§11.4.201: a false-positive failure is as damaging as a false pass).
    run_package_tests "applications/desktop"    "desktop"    "-tags=ci -race" "600s"
    run_package_tests "applications/aurora_os"  "aurora-os"  "-tags=ci -race" "600s"
    # harmony_os: symphony-os was RENAMED to harmony_os in 866dec8f
    # (2025-11-07 — the same commit deleted symphony-os/main.go and added
    # harmony_os/main.go). This suite kept naming the old path, so for nine
    # months it silently skipped ("directory not found") while harmony_os was
    # covered by nothing. Restoring the reference, not deleting it (§11.4.124).
    run_package_tests "applications/harmony_os" "harmony-os" "-tags=ci -race" "600s"
}

# Run integration tests
run_integration_tests() {
    log "Running integration tests..."

    cd "$INNER_MODULE_DIR"

    # Run integration tests (marked with Integration in test name)
    if go test -v ./... -run Integration -timeout 120s; then
        success "Integration tests passed"
    else
        error "Integration tests failed"
    fi
}

# Run end-to-end tests
run_e2e_tests() {
    log "Running end-to-end tests..."

    cd "$INNER_MODULE_DIR"

    if [ -d "test/e2e" ]; then
        if go test -v ./test/e2e/... -timeout 300s; then
            success "E2E tests passed"
        else
            error "E2E tests failed"
        fi
    else
        warning "E2E test directory not found, skipping"
    fi
}

# Generate coverage report
run_coverage() {
    log "Generating comprehensive coverage report..."

    cd "$INNER_MODULE_DIR"

    # Run coverage for all packages
    go test -coverprofile="$PROJECT_ROOT/coverage.out" -covermode=atomic ./...

    # Run coverage for task package specifically
    go test -coverprofile="$PROJECT_ROOT/task-coverage.out" -covermode=atomic ./internal/task/...

    # Display coverage summary
    if [ -f "$PROJECT_ROOT/coverage.out" ]; then
        log "Overall coverage summary:"
        go tool cover -func="$PROJECT_ROOT/coverage.out" | tail -1
    fi

    if [ -f "$PROJECT_ROOT/task-coverage.out" ]; then
        log "Task package coverage summary:"
        go tool cover -func="$PROJECT_ROOT/task-coverage.out" | tail -1
    fi

    success "Coverage report generated"
}

# Run linting
run_linting() {
    log "Running linting checks..."

    cd "$INNER_MODULE_DIR"

    if command -v golangci-lint &> /dev/null; then
        if golangci-lint run ./...; then
            success "Linting passed"
        else
            error "Linting failed"
        fi
    else
        warning "golangci-lint not found, skipping linting"
    fi
}

# Run build checks
run_build_checks() {
    log "Running build checks..."

    cd "$INNER_MODULE_DIR"

    # Try to build all applications
    applications=("terminal-ui" "desktop" "aurora-os" "symphony-os" "server")

    for app in "${applications[@]}"; do
        log "Building $app..."
        if go build -o "/tmp/$app" "./cmd/$app" 2>/dev/null; then
            success "$app build successful"
        else
            error "$app build failed"
        fi
    done
}

# Main execution
main() {
    local test_type="${1:-all}"

    log "Starting HelixCode comprehensive test suite"
    log "Test type: $test_type"

    check_environment

    # --- §11.4.119 resource partition -----------------------------------------
    # Phases that BIND PORTS must own the host ephemeral range exclusively.
    # Phases that do not bind ports (lint, build) stay outside the mutex so
    # genuine parallelism is preserved where it is actually safe.
    # The lock is released automatically on ANY exit path (EXIT trap installed
    # by helix_portlock_acquire; flock is additionally kernel-released on death).
    case "$test_type" in
        unit|integration|e2e|coverage|all)
            helix_portlock_acquire "run-all-tests.sh $test_type"
            ;;
    esac

    case "$test_type" in
        "unit")
            run_unit_tests
            ;;
        "integration")
            run_integration_tests
            ;;
        "e2e")
            run_e2e_tests
            ;;
        "coverage")
            run_coverage
            ;;
        "lint")
            run_linting
            ;;
        "build")
            run_build_checks
            ;;
        "all")
            run_linting
            run_build_checks
            run_unit_tests
            run_integration_tests
            run_e2e_tests
            run_coverage
            ;;
        *)
            error "Unknown test type: $test_type"
            echo "Available test types: unit, integration, e2e, coverage, lint, build, all"
            exit 1
            ;;
    esac

    # Print summary
    log "Test Summary:"
    log "  Total: $((PASSED_TESTS + FAILED_TESTS))"
    log "  Passed: $PASSED_TESTS"
    log "  Failed: $FAILED_TESTS"

    if [ $FAILED_TESTS -eq 0 ]; then
        success "All tests completed successfully!"
        exit 0
    else
        error "Some tests failed. Check the output above for details."
        exit 1
    fi
}

# Run main function
main "$@"