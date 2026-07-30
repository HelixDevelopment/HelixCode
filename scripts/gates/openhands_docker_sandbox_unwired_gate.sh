#!/usr/bin/env bash
# scripts/gates/openhands_docker_sandbox_unwired_gate.sh
#   — CM-OPENHANDS-DOCKER-SANDBOX-UNWIRED
#
# HXC-169 standing regression guard (§11.4.135 / §11.4.115 / §11.4.124).
#
# ---------------------------------------------------------------------------
# WHAT THIS ASSERTS, AND WHY IT EXISTS
# ---------------------------------------------------------------------------
# HXC-169 recorded five publicly-documented advisories against
# `github.com/docker/docker` (pinned `v28.5.2+incompatible` in
# `submodules/helix_agent/go.mod`), three of which have NO fixed version on the
# `docker/docker` (pre-v2) module line. The affected `docker cp` API surface is
# genuinely called by our own code — `govulncheck` traces
# `openhands.Sandbox.CopyTo -> client.Client.CopyToContainer` at
# `internal/clis/openhands/sandbox.go:298` and
# `openhands.Sandbox.CopyFrom -> client.Client.CopyFromContainer` at `:312`.
#
# The HXC-169 resolution (docs/research/hxc169_container_sandbox_reachability_
# 20260731/ANALYSIS.md) is UNREACHABLE-IN-OUR-CONFIGURATION, resting on ONE
# structural fact:
#
#     NOTHING IMPORTS `dev.helix.agent/internal/clis/openhands`.
#
# The package is a Docker-backed code-execution sandbox that was ported in
# commit a5f2857d and never wired to any caller — `go list -deps ./cmd/...`
# across 1097 transitive dependencies of every helix_agent binary contains
# neither the package nor `github.com/docker/docker`, and `govulncheck
# ./cmd/...` reports "No vulnerabilities found" while `govulncheck
# ./internal/clis/openhands/...` reports three.
#
# That is a property of the CURRENT import graph, not a law of nature. The
# moment ANY package imports it, the vulnerable `docker cp` client surface is
# linked into a shipped binary and the unreachability claim silently becomes
# false — with no test failing, because the sandbox's own unit tests pass either
# way. An unreachability claim with no guard rots the moment someone wires it
# up; this gate is that guard.
#
# ---------------------------------------------------------------------------
# THE TWO ASSERTIONS
# ---------------------------------------------------------------------------
# (A) IMPORTER SCAN (primary; build-tag-independent).
#     No .go file anywhere in submodules/helix_agent — OUTSIDE the package's own
#     directory — may contain the import path literal
#     `dev.helix.agent/internal/clis/openhands"`.
#
#     A TEXT scan is used deliberately in preference to `go list`: `go list`
#     honours build tags and therefore cannot see an importer that a
#     non-default build tag excludes, while a text scan sees every file on
#     disk regardless of tags, GOOS/GOARCH, or `//go:build` lines. The trailing
#     double-quote is load-bearing: it is what distinguishes the UNWIRED Docker
#     sandbox package `.../clis/openhands` from the DIFFERENT, genuinely-wired
#     CLI-integration package `.../clis/agents/openhands` (imported by
#     internal/clis/agents/master/master.go), which shells out via os/exec and
#     never touches the Docker client library.
#
# (B) HELIX_CODE MODULE SCAN (always runnable, no submodule required).
#     `github.com/docker/docker` must not appear in helix_code's own module
#     files. helix_code (module `dev.helix.code`) cannot import
#     `dev.helix.agent/internal/...` at all — Go's `internal/` visibility rule
#     is compiler-enforced across module boundaries — so the Docker client can
#     only reach helix_code by becoming a direct dependency of its own module.
#
# Neither assertion claims the advisories are fixed. They are not, and three of
# them cannot be: the guard asserts the vulnerable surface stays OUT of what we
# build and ship. If this gate FAILs, HXC-169 must be REOPENED (§11.4.34) and
# re-decided against the containment options in the ANALYSIS document — the
# correct response is never to weaken this assertion (§11.4.120).
#
# ---------------------------------------------------------------------------
# PAIRED §1.1 MUTATION
# ---------------------------------------------------------------------------
#   ./scripts/gates/openhands_docker_sandbox_unwired_gate.sh --self-test
#
# Runs the SAME scanner used in anger against two hermetic fixture trees under
# a temp dir: a clean tree (must report 0 hits) and a mutated tree carrying a
# planted importer (must report >=1 hit). The self-test never writes inside any
# repository, so it is safe to run while other agents hold the working tree
# (§11.4.84). A scanner that passes its golden-bad fixture is a bluff gate
# (§11.4.107(10)).
#
# Exit codes: 0 = PASS or honest SKIP, 1 = violation, 2 = gate could not run.
# ---------------------------------------------------------------------------

set -uo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
GATE="CM-OPENHANDS-DOCKER-SANDBOX-UNWIRED"

# The unwired Docker-sandbox package. The trailing quote is what keeps
# `.../clis/agents/openhands` (a different, wired, docker-free package) out.
readonly SANDBOX_IMPORT='dev.helix.agent/internal/clis/openhands"'
readonly SANDBOX_PKG_DIR='internal/clis/openhands'
readonly SUBMODULE='submodules/helix_agent'
readonly DOCKER_MODULE='github.com/docker/docker'

# scan_importers <tree-root> <package-dir-relative-to-tree-root>
#
# Prints one `path:line:text` record per importer found, and nothing at all when
# the tree is clean. The package's own directory is excluded because the package
# naturally contains its own name in its files.
scan_importers() {
    local tree="$1" pkg_dir="$2"
    [[ -d "$tree" ]] || return 0
    local f
    while IFS= read -r -d '' f; do
        case "$f" in
            "$tree/$pkg_dir"/*) continue ;;
        esac
        grep -Hn -F -e "$SANDBOX_IMPORT" -- "$f" 2>/dev/null
    done < <(find "$tree" -type f -name '*.go' -print0 2>/dev/null)
    return 0
}

# --- paired §1.1 mutation -------------------------------------------------
if [[ "${1:-}" == "--self-test" ]]; then
    tmp="$(mktemp -d)" || { echo "FAIL $GATE  cannot create temp dir" >&2; exit 2; }
    trap 'rm -rf "$tmp"' EXIT

    # golden-good: a tree importing the SIMILARLY-NAMED but different wired
    # package, plus the sandbox package's own file. Must report ZERO.
    mkdir -p "$tmp/good/$SANDBOX_PKG_DIR" "$tmp/good/internal/clis/agents/master"
    printf 'package openhands\nimport "github.com/docker/docker/client"\n' \
        > "$tmp/good/$SANDBOX_PKG_DIR/sandbox.go"
    printf 'package master\nimport "dev.helix.agent/internal/clis/agents/openhands"\n' \
        > "$tmp/good/internal/clis/agents/master/master.go"

    # golden-bad: same tree PLUS a planted importer of the sandbox package.
    cp -r "$tmp/good" "$tmp/bad"
    mkdir -p "$tmp/bad/internal/wired"
    printf 'package wired\nimport "dev.helix.agent/internal/clis/openhands"\n' \
        > "$tmp/bad/internal/wired/wired.go"

    good_hits="$(scan_importers "$tmp/good" "$SANDBOX_PKG_DIR" | wc -l)"
    bad_hits="$(scan_importers "$tmp/bad" "$SANDBOX_PKG_DIR" | wc -l)"
    st_fail=0

    if [[ "$good_hits" -ne 0 ]]; then
        echo "FAIL $GATE  self-test: golden-good fixture reported $good_hits hit(s), expected 0 (false positive — the scanner is matching the wired agents/openhands package)" >&2
        st_fail=1
    fi
    if [[ "$bad_hits" -lt 1 ]]; then
        echo "FAIL $GATE  self-test: golden-bad fixture reported $bad_hits hit(s), expected >=1 (the scanner cannot detect a planted importer — this gate would be a bluff)" >&2
        st_fail=1
    fi

    if [[ "$st_fail" -ne 0 ]]; then
        exit 1
    fi
    echo "$GATE: self-test PASS (golden-good=0 hits, golden-bad=$bad_hits hit(s) — scanner detects a planted importer and does not false-positive on the wired sibling package)"
    exit 0
fi

cd "$ROOT" || exit 2
violations=0
checks=0

# --- (A) importer scan ----------------------------------------------------
if [[ ! -d "$ROOT/$SUBMODULE/$SANDBOX_PKG_DIR" ]]; then
    # §11.4.3 honest SKIP: the submodule is not checked out in this clone, so
    # the assertion is genuinely unevaluable here — never a silent pass.
    echo "$GATE: SKIP assertion (A) — reason=submodule_not_checked_out path=$SUBMODULE/$SANDBOX_PKG_DIR (run 'git submodule update --init $SUBMODULE' to evaluate)"
else
    checks=$((checks + 1))
    importers="$(scan_importers "$ROOT/$SUBMODULE" "$SANDBOX_PKG_DIR")"
    if [[ -n "$importers" ]]; then
        echo "FAIL $GATE  (A) '$SANDBOX_IMPORT' now has importer(s) — the HXC-169 unreachability finding is BROKEN." >&2
        echo "     The Docker-backed sandbox reaches CopyToContainer/CopyFromContainer (sandbox.go:298/:312)," >&2
        echo "     whose advisories GHSA-rg2x-37c3-w2rh / GHSA-x86f-5xw2-fm2r / GHSA-vp62-88p7-qqf5 have NO fix." >&2
        echo "     REOPEN HXC-169 (§11.4.34) and re-decide against docs/research/hxc169_container_sandbox_reachability_20260731/ANALYSIS.md." >&2
        echo "     Do NOT weaken this gate to make it pass (§11.4.120)." >&2
        while IFS= read -r line; do
            [[ -n "$line" ]] && echo "     importer: $line" >&2
        done <<< "$importers"
        violations=$((violations + 1))
    fi
fi

# --- (B) helix_code module scan -------------------------------------------
for mf in go.mod go.sum helix_code/go.mod helix_code/go.sum; do
    [[ -f "$ROOT/$mf" ]] || continue
    checks=$((checks + 1))
    if grep -q -F -e "$DOCKER_MODULE" -- "$ROOT/$mf" 2>/dev/null; then
        echo "FAIL $GATE  (B) '$DOCKER_MODULE' appeared in $mf — the vulnerable Docker client is now a helix_code dependency." >&2
        echo "     helix_code carried ZERO docker references when HXC-169 was decided; three of its advisories have no fix." >&2
        echo "     REOPEN HXC-169 (§11.4.34) before proceeding." >&2
        violations=$((violations + 1))
    fi
done

echo "$GATE: ${checks} assertion(s) evaluated, ${violations} violation(s)"
[[ "$violations" -gt 0 ]] && exit 1
exit 0
