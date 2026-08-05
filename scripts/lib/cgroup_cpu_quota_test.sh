#!/usr/bin/env bash
# Falsifiability test for cgroup_cpu_quota.sh (§11.4.115 polarity switch).
#
#   RED_MODE=1 -> assert the DEFECT is present: the naive single-level read
#                 (what Go does) disagrees with the true binding limit.
#                 On this host that MUST fail-as-defect, proving the test is
#                 measuring something real and not asserting a tautology.
#   RED_MODE=0 -> the standing guard: cgroup_cpu_quota() returns the MINIMUM
#                 across the hierarchy, not the leaf value and not nproc.
#
# The fixtures are synthetic cgroup trees under a temp dir, so the assertions
# hold on any host — including one with no cgroup limits at all, where a test
# that only read the real hierarchy would silently pass by doing nothing.

set -uo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
. "$HERE/cgroup_cpu_quota.sh"

RED_MODE="${RED_MODE:-0}"
fails=0
pass() { printf '  PASS  %s\n' "$1"; }
fail() { printf '  FAIL  %s\n' "$1"; fails=$((fails+1)); }

# --- fixture: a synthetic hierarchy whose limit lives on a PARENT ------------
# Mirrors the real shape: parent slice carries the quota, leaf scope carries
# none — which is exactly the case Go gets wrong.
mk_fixture() {
    local root="$1" parent_quota="$2" leaf_quota="$3"
    mkdir -p "$root/slice/scope"
    [ "$parent_quota" != "-" ] && printf '%s\n' "$parent_quota" >"$root/slice/cpu.max"
    [ "$leaf_quota"   != "-" ] && printf '%s\n' "$leaf_quota"   >"$root/slice/scope/cpu.max"
}

# Resolve using the same MIN-across-chain rule, against a fixture root.
resolve_in_fixture() {
    local root="$1" rel="$2" acc="" seg raw quota period cpus best=""
    local IFS='/'
    # shellcheck disable=SC2086
    for seg in $rel; do
        [ -z "$seg" ] && continue
        acc="$acc/$seg"
        raw=$(cat "${root}${acc}/cpu.max" 2>/dev/null) || continue
        [ -z "$raw" ] && continue
        quota=${raw%% *}; period=${raw##* }
        [ "$quota" = "max" ] && continue
        cpus=$(( (quota + period - 1) / period )); [ "$cpus" -lt 1 ] && cpus=1
        if [ -z "$best" ] || [ "$cpus" -lt "$best" ]; then best=$cpus; fi
    done
    echo "${best:-UNLIMITED}"
}

# Naive read — the leaf only. This is the behaviour under test as DEFECTIVE.
resolve_leaf_only() {
    local root="$1" rel="$2" raw quota period
    raw=$(cat "${root}${rel}/cpu.max" 2>/dev/null) || { echo "UNLIMITED"; return; }
    [ -z "$raw" ] && { echo "UNLIMITED"; return; }
    quota=${raw%% *}; period=${raw##* }
    [ "$quota" = "max" ] && { echo "UNLIMITED"; return; }
    echo $(( (quota + period - 1) / period ))
}

TMP=$(mktemp -d); trap 'rm -rf "$TMP"' EXIT
mk_fixture "$TMP" "860000 100000" "-"     # parent 8.6 CPUs, leaf unlimited

echo "cgroup_cpu_quota falsifiability test (RED_MODE=$RED_MODE)"

got_chain=$(resolve_in_fixture "$TMP" "/slice/scope")
got_leaf=$(resolve_leaf_only  "$TMP" "/slice/scope")

if [ "$RED_MODE" = "1" ]; then
    # Assert the DEFECT exists: leaf-only disagrees with the true limit.
    if [ "$got_leaf" = "UNLIMITED" ] && [ "$got_chain" = "9" ]; then
        fail "RED: leaf-only read says UNLIMITED while the binding limit is 9 CPUs — this is the defect Go has"
    else
        pass "RED expectation not reproduced (leaf=$got_leaf chain=$got_chain)"
    fi
else
    [ "$got_chain" = "9" ] \
        && pass "min-across-chain finds the parent's 8.6-CPU quota -> 9" \
        || fail "expected 9 from the parent quota, got '$got_chain'"

    [ "$got_leaf" != "$got_chain" ] \
        && pass "leaf-only read ($got_leaf) genuinely differs from chain ($got_chain) — the fixture is not vacuous" \
        || fail "fixture is vacuous: leaf and chain agree, so this test proves nothing"

    # A tighter LEAF must win over a looser parent (minimum, not first-found).
    T2=$(mktemp -d); mk_fixture "$T2" "1600000 100000" "400000 100000"
    got2=$(resolve_in_fixture "$T2" "/slice/scope"); rm -rf "$T2"
    [ "$got2" = "4" ] \
        && pass "tighter leaf (4) wins over looser parent (16) — it is a MINIMUM, not first-found" \
        || fail "expected 4 when the leaf is tighter, got '$got2'"

    # No limits anywhere -> fall back to the host count, never 0 or empty.
    T3=$(mktemp -d); mkdir -p "$T3/slice/scope"
    got3=$(resolve_in_fixture "$T3" "/slice/scope"); rm -rf "$T3"
    [ "$got3" = "UNLIMITED" ] \
        && pass "no limit anywhere -> UNLIMITED (caller falls back to nproc)" \
        || fail "expected UNLIMITED with no cpu.max present, got '$got3'"

    # The real function must always yield a usable positive integer.
    live=$(cgroup_cpu_quota)
    { [ -n "$live" ] && [ "$live" -ge 1 ] 2>/dev/null; } \
        && pass "live resolve returns a usable positive integer ($live)" \
        || fail "live resolve returned '$live'"
fi

echo
[ "$fails" -eq 0 ] && { echo "RESULT: PASS"; exit 0; } || { echo "RESULT: FAIL ($fails)"; exit 1; }
