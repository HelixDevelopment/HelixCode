#!/usr/bin/env bash
# cgroup_cpu_quota.sh — resolve the CPU limit that ACTUALLY binds this process.
#
# WHY THIS EXISTS (measured 2026-08-05, not theorised):
#
#   nproc                      -> 64
#   runtime.NumCPU()           -> 64
#   GOMAXPROCS (Go 1.26)       -> 64
#   the quota that really binds -> 8.6      (cpu.max = 860000 100000)
#
# Go 1.25+ IS container-aware, but it reads the process's own (leaf) cgroup.
# Here the leaf is a per-command `run-<id>.scope` carrying NO cpu.max, while the
# quota sits on the parent `tmxw-helix-code-<n>.slice`. Finding nothing at the
# leaf, Go falls back to NumCPU() and every build runs 64-way against 8.6 CPUs
# — 7.4x oversubscription per process, ~30x with four agents.
#
# Consequences measured on this host:
#   50.3% of scheduling periods throttled  (337970 / 672426)
#   75.5 hours of wall-clock spent frozen
#   0.62 s frozen per 1 s of CPU actually obtained
#   GOMAXPROCS=8 beat GOMAXPROCS=64 on a trivial build: 3.52s vs 4.25s
#
# That throttling is why long agent commands overran the 600s stream watchdog
# and got reported as "stalled". They were not stuck; they were frozen for
# roughly 38% of every wall-clock second they tried to use.
#
# THE RULE, which is the general lesson: in a cgroup hierarchy the binding limit
# is the MINIMUM across every level, never the value at the level you happen to
# read. The same single-level-read mistake produced two other wrong numbers in
# this project on the same day — `ulimit -u` reporting 262144 against a real
# 4096 pids.max, and a repo-root search reporting 0 tracked node_modules while a
# submodule held 8,907.

# cgroup_cpu_quota — echo the binding CPU count, or the host CPU count if
# genuinely unlimited. Always echoes a positive integer; never fails the caller.
cgroup_cpu_quota() {
    local cg acc seg raw quota period cpus best=""

    cg=$(cut -d: -f3 </proc/self/cgroup 2>/dev/null | head -1) || cg=""
    if [ -n "$cg" ]; then
        acc=""
        # Walk root -> leaf, keeping the smallest concrete limit found.
        local IFS='/'
        # shellcheck disable=SC2086
        for seg in $cg; do
            [ -z "$seg" ] && continue
            acc="$acc/$seg"
            raw=$(cat "/sys/fs/cgroup${acc}/cpu.max" 2>/dev/null) || continue
            [ -z "$raw" ] && continue
            quota=${raw%% *}
            period=${raw##* }
            [ "$quota" = "max" ] && continue          # this level is unlimited
            [ -z "$period" ] || [ "$period" -le 0 ] 2>/dev/null && continue
            # Round UP: a 8.6-CPU quota should permit 9, not 8.
            cpus=$(( (quota + period - 1) / period ))
            [ "$cpus" -lt 1 ] && cpus=1
            if [ -z "$best" ] || [ "$cpus" -lt "$best" ]; then best=$cpus; fi
        done
    fi

    if [ -n "$best" ]; then
        echo "$best"
    else
        nproc 2>/dev/null || echo 1
    fi
}

# cgroup_export_gomaxprocs — set GOMAXPROCS to the binding limit unless the
# caller has already chosen one. Deliberately does NOT override an explicit
# operator value: someone who set it meant it.
cgroup_export_gomaxprocs() {
    if [ -n "${GOMAXPROCS:-}" ]; then
        return 0
    fi
    GOMAXPROCS=$(cgroup_cpu_quota)
    export GOMAXPROCS
}

# Executed directly (not sourced) -> print what we resolved, for diagnosis.
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    printf 'nproc                : %s\n' "$(nproc 2>/dev/null || echo '?')"
    printf 'binding cgroup quota : %s\n' "$(cgroup_cpu_quota)"
    printf 'GOMAXPROCS now       : %s\n' "${GOMAXPROCS:-<unset, Go would use NumCPU>}"
fi
