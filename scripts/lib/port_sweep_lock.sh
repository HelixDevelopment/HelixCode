#!/usr/bin/env bash
# shellcheck shell=bash
#
# scripts/lib/port_sweep_lock.sh
# ==============================================================================
# HOST-GLOBAL MUTEX FOR PORT-BINDING TEST SWEEPS
#
# WHY THIS EXISTS — DO NOT "OPTIMISE" IT AWAY BY MAKING SWEEPS PARALLEL AGAIN.
# ------------------------------------------------------------------------------
# Constitutional anchor: §11.4.119 (single-resource-owner partitioning for
# parallel hardware testing). The host ephemeral port range is an EXCLUSIVE,
# SHARED resource. Exactly ONE port-binding sweep may own it at a time.
# Parallelism is partitioned by RESOURCE, never by repository/module.
#
# CAPTURED FORENSIC EVIDENCE (FACT, 2026-07-27) — the defect this prevents:
#
#   Host ephemeral range (/proc/sys/net/ipv4/ip_local_port_range): 32768 60999
#
#   Two sweeps were launched ~2 SECONDS apart and contended for that one range:
#     qa-results/full_retest/helix_agent_20260727T133218Z.log
#     qa-results/full_retest/helix_code_inner_20260727T133220Z.log
#
#   Measured consequences:
#     * helix_agent log: 112 occurrences of
#         "dial tcp: connect: cannot assign requested address"
#       (`grep -c 'cannot assign requested address' <log>` -> 112)
#     * helix_code inner log: dev.helix.code/internal/discovery FAILED (10.728s)
#       with 7 top-level failures, including:
#         port_allocator_test.go:375: no ports available in configured range
#         failed to allocate ephemeral port: listen tcp :0: bind: address already in use
#         TestAllocatePort_RangeExhausted_WithEphemeral / TestConcurrentAllocations
#         TestConcurrentReleases / TestCheckHTTPHealth / TestCheckHTTPHealth_CustomEndpoint
#         TestCheckTCPHealth / TestPerformHealthChecks_Integration
#
#   Counter-evidence that the CODE is fine and the HOST was the defect: on a
#   quiet host the same package passes —
#     ok  dev.helix.code/internal/discovery  (captured in qa-results)
#
#   i.e. concurrent port-binding sweeps manufacture PHANTOM code defects.
#   A red test produced under port contention is NOT evidence of a code bug
#   (§11.4.119: "a PASS under contention is an evidence-integrity bluff" — the
#   converse FAIL is equally worthless).
#
# WHY THE LOCK IS HOST-GLOBAL AND *NOT* TRACK-QUALIFIED
# ------------------------------------------------------------------------------
# §11.4.178 mandates track-qualified identity for resources with one claimant
# PER TRACK. This resource is the opposite: ONE kernel port range shared by
# EVERY checkout, track, worktree and clone on the host. A per-track or
# per-checkout lock path would hand each track its own private mutex and the
# tracks would collide exactly as they did on 2026-07-27. The shared key is
# deliberate. Do not add "$TRACK"/"$PWD"/branch to HELIX_PORTLOCK_FILE.
#
# WHY flock AND NOT A PID FILE
# ------------------------------------------------------------------------------
# §11.4.180 requires stale-lock reaping. `flock(2)` locks are held by the open
# file description and released by the KERNEL when the holder dies (including
# SIGKILL), so a dead holder can never wedge the queue — §11.4.180 is satisfied
# BY CONSTRUCTION, with no reaping heuristic to get wrong. The mkdir fallback
# (hosts without flock(1), §11.4.81 cross-platform parity) is existence-based
# and therefore DOES reap, but only holders PROVEN dead via `kill -0`
# (§11.4.180: never remove a lock whose holder is alive).
#
# LOCK LIFETIME COVERS THE WHOLE PROCESS **TREE** (measured, intentional)
# ------------------------------------------------------------------------------
# flock(2) is held by the OPEN FILE DESCRIPTION, which fork() SHARES. Every
# descendant inherits the lock FD, so the lock stays held until the acquiring
# process AND all descendants holding that FD have exited.
#
# This is the CORRECT safety property, not a leak: the descendants are the
# `go test` processes that actually bind the ports. If the wrapper shell is
# killed but `go test` keeps running, a second sweep MUST still be blocked —
# releasing early would reintroduce the 2026-07-27 defect.
#
# Measured proof (FACT, this host, Linux, util-linux flock 2.39.2):
#   child `sleep` inherited the lock FD:
#     /proc/<child>/fd/10 -> /tmp/.../portlock_diag.lock
#   kernel view: /proc/locks -> "FLOCK ADVISORY WRITE <pid> 00:22:8211 0 EOF"
#   SIGKILL parent only  -> orphan child alive -> new acquire rc=75 (still held)
#   kill orphan child    -> new acquire rc=0   (kernel released it; NO reaping)
#
# OPERATIONAL CONSEQUENCE: a wedged/orphaned `go test` will block later sweeps
# until it exits. That surfaces as an honest rc=75 timeout naming the holder —
# NOT as a silent overlap. Do not "fix" this by releasing the lock early.
#
# HONEST BOUNDARY (§11.4.6)
# ------------------------------------------------------------------------------
# This mutex serialises sweeps that COOPERATE by calling it. It cannot stop an
# uncooperative process (an ad-hoc `go test ./...`, another project's stack, a
# stray container) from consuming the same range. It guarantees OUR sweeps do
# not collide with each other; it does not guarantee a globally quiet host.
# Verify quiescence separately before trusting a red result (§11.4.174).
#
# USAGE
# ------------------------------------------------------------------------------
#   source "<repo>/scripts/lib/port_sweep_lock.sh"
#   helix_portlock_acquire "run-all-tests.sh unit"   # blocks until owned
#   ...run port-binding tests...
#   helix_portlock_release                            # or just exit (EXIT trap)
#
# ENV KNOBS (there is deliberately NO disable flag — §11.4 no escape hatch)
#   HELIX_PORTLOCK_FILE     lock path (default ${TMPDIR:-/tmp}/helix_port_bound_sweep.lock)
#   HELIX_PORTLOCK_TIMEOUT  max seconds to wait for the lock (default 10800 = 3h)
#   HELIX_PORTLOCK_QUIET    1 = suppress the informational wait banner
#
# EXIT CODES
#   75 (EX_TEMPFAIL)  timed out waiting for another sweep to finish
#   70 (EX_SOFTWARE)  could not establish a lock at all (unusable environment)
# ==============================================================================

# Idempotent source guard: sourcing twice must not redefine state.
if [ -n "${_HELIX_PORTLOCK_SOURCED:-}" ]; then
    return 0 2>/dev/null || true
fi
_HELIX_PORTLOCK_SOURCED=1

HELIX_PORTLOCK_FILE="${HELIX_PORTLOCK_FILE:-${TMPDIR:-/tmp}/helix_port_bound_sweep.lock}"
HELIX_PORTLOCK_TIMEOUT="${HELIX_PORTLOCK_TIMEOUT:-10800}"
HELIX_PORTLOCK_QUIET="${HELIX_PORTLOCK_QUIET:-0}"

_HELIX_PORTLOCK_HELD=0
_HELIX_PORTLOCK_FD=""
_HELIX_PORTLOCK_BACKEND=""
_HELIX_PORTLOCK_DIR="${HELIX_PORTLOCK_FILE}.d"

_helix_portlock_say() {
    [ "$HELIX_PORTLOCK_QUIET" = "1" ] && return 0
    echo "[port-lock] $*" >&2
}

# Never fatal: metadata is diagnostic only.
_helix_portlock_meta_path() {
    if [ "$_HELIX_PORTLOCK_BACKEND" = "mkdir" ]; then
        echo "${_HELIX_PORTLOCK_DIR}/holder"
    else
        echo "$HELIX_PORTLOCK_FILE"
    fi
}

_helix_portlock_write_meta() {
    local label="$1" meta
    meta="$(_helix_portlock_meta_path)"
    # Only the holder writes. Truncating the file does NOT release a flock(2)
    # lock (the lock lives on the open file description, not the content).
    {
        echo "pid=$$"
        echo "host=$(uname -n 2>/dev/null || echo unknown)"
        echo "user=${USER:-$(id -un 2>/dev/null || echo unknown)}"
        echo "cwd=$PWD"
        echo "label=${label:-unlabelled}"
        echo "acquired_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    } >"$meta" 2>/dev/null || true
}

_helix_portlock_report_holder() {
    local meta
    meta="$(_helix_portlock_meta_path)"
    if [ -s "$meta" ]; then
        _helix_portlock_say "current holder:"
        while IFS= read -r line; do
            _helix_portlock_say "    $line"
        done <"$meta"
    else
        _helix_portlock_say "current holder: <no metadata recorded>"
    fi
}

# --- mkdir fallback: reap ONLY provably-dead holders (§11.4.180) --------------
_helix_portlock_reap_if_dead() {
    local meta pid host self_host
    meta="${_HELIX_PORTLOCK_DIR}/holder"
    [ -d "$_HELIX_PORTLOCK_DIR" ] || return 1
    [ -s "$meta" ] || return 1

    pid="$(sed -n 's/^pid=//p' "$meta" 2>/dev/null | head -1)"
    host="$(sed -n 's/^host=//p' "$meta" 2>/dev/null | head -1)"
    self_host="$(uname -n 2>/dev/null || echo unknown)"

    # Cannot prove liveness of a PID on a different host -> never reap.
    if [ -z "$pid" ] || [ -z "$host" ] || [ "$host" != "$self_host" ]; then
        _helix_portlock_say "KEPT lock (holder liveness unprovable: pid='${pid}' host='${host}')"
        return 1
    fi
    if kill -0 "$pid" 2>/dev/null; then
        return 1   # holder ALIVE -> removing it would corrupt a live sweep
    fi
    _helix_portlock_say "REAPED lock: holder pid=$pid on $host is dead (kill -0 failed)"
    rm -rf "$_HELIX_PORTLOCK_DIR" 2>/dev/null || true
    return 0
}

_helix_portlock_acquire_mkdir() {
    local label="$1" deadline waited=0 announced=0
    deadline=$(( $(date +%s) + HELIX_PORTLOCK_TIMEOUT ))
    while ! mkdir "$_HELIX_PORTLOCK_DIR" 2>/dev/null; do
        if [ "$announced" -eq 0 ]; then
            _helix_portlock_say "another port-binding sweep owns the host ephemeral range; waiting (§11.4.119)."
            _helix_portlock_report_holder
            announced=1
        fi
        _helix_portlock_reap_if_dead && continue
        if [ "$(date +%s)" -ge "$deadline" ]; then
            _helix_portlock_say "TIMEOUT after ${HELIX_PORTLOCK_TIMEOUT}s waiting for $HELIX_PORTLOCK_FILE"
            _helix_portlock_say "NOTE: the recorded pid may already be dead while a DESCENDANT"
            _helix_portlock_say "      (e.g. an orphaned 'go test') still holds the inherited lock FD."
            _helix_portlock_say "      Inspect:  grep FLOCK /proc/locks   /  ls -l /proc/<pid>/fd"
            _helix_portlock_say "      Do NOT start a second sweep anyway (§11.4.119)."
            return 75
        fi
        sleep 5
        waited=$((waited + 5))
    done
    _HELIX_PORTLOCK_HELD=1
    _helix_portlock_write_meta "$label"
    return 0
}

_helix_portlock_acquire_flock() {
    local label="$1"
    # Dynamic FD (bash >= 4.1) avoids colliding with a caller's fixed FD.
    if ! exec {_HELIX_PORTLOCK_FD}>"$HELIX_PORTLOCK_FILE"; then
        _helix_portlock_say "FATAL: cannot open lock file $HELIX_PORTLOCK_FILE"
        return 70
    fi
    if ! flock -n "$_HELIX_PORTLOCK_FD"; then
        _helix_portlock_say "another port-binding sweep owns the host ephemeral range; waiting (§11.4.119)."
        _helix_portlock_report_holder
        _helix_portlock_say "waiting up to ${HELIX_PORTLOCK_TIMEOUT}s ..."
        if ! flock -w "$HELIX_PORTLOCK_TIMEOUT" "$_HELIX_PORTLOCK_FD"; then
            _helix_portlock_say "TIMEOUT after ${HELIX_PORTLOCK_TIMEOUT}s waiting for $HELIX_PORTLOCK_FILE"
            _helix_portlock_say "NOTE: the recorded pid may already be dead while a DESCENDANT"
            _helix_portlock_say "      (e.g. an orphaned 'go test') still holds the inherited lock FD."
            _helix_portlock_say "      Inspect:  grep FLOCK /proc/locks   /  ls -l /proc/<pid>/fd"
            _helix_portlock_say "      Do NOT start a second sweep anyway (§11.4.119)."
            eval "exec ${_HELIX_PORTLOCK_FD}>&-" 2>/dev/null || true
            _HELIX_PORTLOCK_FD=""
            return 75
        fi
    fi
    _HELIX_PORTLOCK_HELD=1
    _helix_portlock_write_meta "$label"
    return 0
}

# helix_portlock_acquire [label]
# Blocks until this process exclusively owns the host ephemeral port range.
helix_portlock_acquire() {
    local label="${1:-}" rc

    if [ "$_HELIX_PORTLOCK_HELD" = "1" ]; then
        return 0   # re-entrant: already ours
    fi

    if command -v flock >/dev/null 2>&1; then
        _HELIX_PORTLOCK_BACKEND="flock"
    else
        # §11.4.81 cross-platform parity: macOS/BSD ship no flock(1).
        _HELIX_PORTLOCK_BACKEND="mkdir"
        _helix_portlock_say "flock(1) absent ($(uname -s)); using mkdir mutex + kill -0 reaping."
    fi

    mkdir -p "$(dirname "$HELIX_PORTLOCK_FILE")" 2>/dev/null || true

    if [ "$_HELIX_PORTLOCK_BACKEND" = "flock" ]; then
        _helix_portlock_acquire_flock "$label"; rc=$?
    else
        _helix_portlock_acquire_mkdir "$label"; rc=$?
    fi
    [ "$rc" -eq 0 ] || return "$rc"

    # Release on ANY exit path. Chain onto whatever the caller already trapped.
    _HELIX_PORTLOCK_PREV_EXIT_TRAP="$(trap -p EXIT | sed -n "s/^trap -- '\(.*\)' EXIT$/\1/p")"
    trap 'helix_portlock_release; eval "${_HELIX_PORTLOCK_PREV_EXIT_TRAP:-:}"' EXIT

    _helix_portlock_say "ACQUIRED $HELIX_PORTLOCK_FILE (backend=$_HELIX_PORTLOCK_BACKEND, pid=$$, label=${label:-unlabelled})"
    return 0
}

# helix_portlock_release — idempotent; safe to call when not held.
helix_portlock_release() {
    [ "$_HELIX_PORTLOCK_HELD" = "1" ] || return 0
    _HELIX_PORTLOCK_HELD=0
    if [ "$_HELIX_PORTLOCK_BACKEND" = "mkdir" ]; then
        rm -rf "$_HELIX_PORTLOCK_DIR" 2>/dev/null || true
    else
        : >"$HELIX_PORTLOCK_FILE" 2>/dev/null || true
        if [ -n "$_HELIX_PORTLOCK_FD" ]; then
            eval "exec ${_HELIX_PORTLOCK_FD}>&-" 2>/dev/null || true
            _HELIX_PORTLOCK_FD=""
        fi
    fi
    _helix_portlock_say "RELEASED $HELIX_PORTLOCK_FILE (pid=$$)"
    return 0
}

# helix_portlock_held — exit 0 when this process owns the lock.
helix_portlock_held() {
    [ "$_HELIX_PORTLOCK_HELD" = "1" ]
}
