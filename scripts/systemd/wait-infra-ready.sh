#!/usr/bin/env bash
#
# wait-infra-ready.sh — block until the HelixCode infrastructure stack is
# actually SERVING, not merely created.
#
# WHY THIS EXISTS (HXC-228)
# ------------------------
# helixcode-infra.service is Type=oneshot running `podman compose up -d`, which
# returns as soon as the containers are CREATED — long before they serve. So
# `After=helixcode-infra.service` in a consumer unit orders it after container
# *creation*, not after readiness. Captured on a real cold boot
# (docs/qa/live_systemd_boot_20260806T145857Z/11_helixagent_coldboot_race.txt):
# helixagent started, probed Cognee, and exited fatally all inside the SAME
# second the unit started, because Cognee — whose compose entry waits on
# postgres+redis being healthy and is therefore always among the last to
# serve — was still coming up. Fifteen seconds later the identical probe passed.
#
# Running this as ExecStartPost makes the unit's "started" signal mean "the
# stack is serving", so the systemd ordering relationship consumers already
# declare becomes truthful.
#
# SAFETY
# ------
# * Read-only. Only `podman inspect` / `podman ps` are used. Nothing is started,
#   stopped, killed or otherwise mutated. It cannot damage the stack.
# * Scoped to containers named `helixcode-infra-*` — never another project's
#   workload (§11.4.174 process/resource ownership).
# * Rootless podman only; no sudo, no docker (§11.4.161).
# * Bounded. An unbounded wait would convert a slow boot into a hang.
#
# EXIT CONTRACT (deliberate, see the note at the timeout branch)
# --------------------------------------------------------------
#   0  every container is running, and every container that declares a
#      healthcheck reported healthy  — OR — some are running but still
#      un-healthy at the deadline (WARNED, not fatal).
#   1  at least one container is NOT RUNNING at the deadline, or podman is
#      unusable. That is unambiguous infrastructure failure.
#
set -uo pipefail

readonly NAME_PREFIX='helixcode-infra-'
readonly TIMEOUT="${HELIX_INFRA_READY_TIMEOUT:-180}"
readonly POLL_INTERVAL="${HELIX_INFRA_READY_POLL:-2}"

log() { printf '[wait-infra-ready] %s\n' "$*"; }

if ! command -v podman >/dev/null 2>&1; then
    log "FATAL: podman not found on PATH"
    exit 1
fi

# Enumerate OUR containers dynamically, so this never drifts out of sync with
# compose.helixcode-infra.yml and never reaches outside our own stack.
mapfile -t containers < <(podman ps -a --format '{{.Names}}' 2>/dev/null | grep "^${NAME_PREFIX}" | sort)

if [ "${#containers[@]}" -eq 0 ]; then
    log "FATAL: no ${NAME_PREFIX}* containers exist — compose up did not create the stack"
    exit 1
fi

log "waiting up to ${TIMEOUT}s for ${#containers[@]} container(s) to become ready"

# container_state <name> -> "<status>|<health>"; health is "none" when the
# container declares no healthcheck (running is then the readiness condition).
container_state() {
    podman inspect "$1" \
        --format '{{.State.Status}}|{{if .State.Healthcheck}}{{.State.Health.Status}}{{else}}none{{end}}' \
        2>/dev/null || echo "missing|none"
}

deadline=$(( $(date +%s) + TIMEOUT ))
declare -a not_running=() not_healthy=()

while :; do
    not_running=()
    not_healthy=()

    for c in "${containers[@]}"; do
        state="$(container_state "$c")"
        status="${state%%|*}"
        health="${state##*|}"

        if [ "$status" != "running" ]; then
            not_running+=("${c} (${status})")
        elif [ "$health" != "none" ] && [ "$health" != "healthy" ]; then
            not_healthy+=("${c} (${health})")
        fi
    done

    if [ "${#not_running[@]}" -eq 0 ] && [ "${#not_healthy[@]}" -eq 0 ]; then
        log "READY: all ${#containers[@]} container(s) running and healthy"
        exit 0
    fi

    now=$(date +%s)
    if [ "$now" -ge "$deadline" ]; then
        break
    fi

    log "still waiting (${#not_running[@]} not running, ${#not_healthy[@]} not healthy) — $(( deadline - now ))s left"
    sleep "$POLL_INTERVAL"
done

# Timeout reached.
#
# A container that is NOT RUNNING is unambiguous infra failure -> fail the unit
# so the operator sees it.
#
# A container that IS running but has not gone healthy is deliberately NOT
# fatal here: failing this unit would take down every consumer via their
# Requires=, and a slow-but-alive dependency is exactly the case the consumer's
# own bounded readiness wait already handles (HXC-228, helixagent
# verifyAllMandatoryDependencies). Warning loudly beats amplifying one slow
# container into a platform-wide outage — but it is never silent.
if [ "${#not_running[@]}" -gt 0 ]; then
    log "FAILED after ${TIMEOUT}s — container(s) NOT RUNNING: ${not_running[*]}"
    [ "${#not_healthy[@]}" -gt 0 ] && log "  also un-healthy: ${not_healthy[*]}"
    exit 1
fi

log "WARNING after ${TIMEOUT}s — running but not yet healthy: ${not_healthy[*]}"
log "proceeding; consumers apply their own bounded readiness wait"
exit 0
