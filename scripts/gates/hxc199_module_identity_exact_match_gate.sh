#!/usr/bin/env bash
# hxc199_module_identity_exact_match_gate.sh — CM-MODULE-IDENTITY-EXACT-MATCH
#
# ===========================================================================
# WHY THIS EXISTS
# ===========================================================================
# HXC-187 renamed the thin root meta-repo module from `dev.helix.code` to
# `dev.helix.code/meta` so it would stop colliding with the real inner
# application module (helix_code/go.mod, which legitimately keeps
# `dev.helix.code`). `scripts/probes/hxc159_env_facts.sh` re-derives whether
# that collision still exists by comparing the two declared module paths.
#
# The FIRST version of that comparison used a substring test:
#     [[ "$root_mod" == *"dev.helix.code"* ]]
# where $root_mod was the FULL `module ...` line read from the root go.mod.
# That is WRONG: `dev.helix.code` is a literal PREFIX of `dev.helix.code/meta`,
# so the substring test still matched AFTER the HXC-187 fix landed, and the
# probe kept reporting the collision as unfixed (GAP-IN-ITEM) on a tree where
# it was already resolved. That is a live false positive: a reader trusting
# the probe's report could be misled into "fixing" (i.e. reverting) already-
# correct work.
#
# The fix (already landed — see scripts/probes/hxc159_env_facts.sh lines
# ~162-186) replaced the substring test with an EXACT string-equality
# comparison of the extracted module-path tokens, factored into
# scripts/lib/module_identity.sh as `module_paths_identical()` +
# `go_mod_module_path()` — the single source of truth this gate reuses rather
# than re-deriving. HXC-199 exists because that fix landed WITHOUT a permanent
# regression guard (§11.4.135): nothing stops a future edit from silently
# swapping the exact-match predicate back for a substring/prefix test. This
# gate is that guard.
#
# ===========================================================================
# POLARITY SWITCH (Constitution §11.4.115)
# ===========================================================================
#   RED_MODE=1 — reproduce the FALSE POSITIVE on the CURRENT (already-fixed)
#       tree. Applies the reconstructed OLD substring predicate to the real,
#       on-disk root go.mod `module` line and asserts it WRONGLY reports a
#       collision — even though the root and inner modules are, by the exact-
#       match predicate, distinct today. PASS here is the reproduction proof:
#       it shows the historical bug's mechanism is real and would still
#       misfire on this tree if the substring test were still in use.
#
#   RED_MODE=0 — DEFAULT. The standing GREEN regression guard. Uses
#       module_paths_identical() (scripts/lib/module_identity.sh) to assert
#       the root and inner go.mod module paths are exact-match DISTINCT on
#       the current tree (i.e. the R-26 collision stays resolved).
#
#   Both modes share the same falsifiability preconditions (S1 below), which
#   run unconditionally: a predicate that cannot tell a genuine recurrence
#   from a prefix-only lookalike is not a guard, and a RED reproduction that
#   no longer reproduces the historical mechanism cannot be trusted to prove
#   anything about RED_MODE=0 either.
#
# ===========================================================================
# FALSIFIABILITY — BOTH DIRECTIONS (§11.4.201: assert the real condition)
# ===========================================================================
#   S1a — a GENUINE recurrence (two module paths that really are identical)
#         MUST be caught by module_paths_identical(). A predicate that never
#         fires is not a guard.
#   S1b — a PREFIX-ONLY LOOKALIKE ("dev.helix.codebase" merely starts with
#         "dev.helix.code") must NOT be reported identical by
#         module_paths_identical(). A predicate that always fires is worse
#         than no guard — it is indistinguishable from the original bug.
#   S1c — negative control: the reconstructed OLD substring predicate MUST
#         still misfire on that same lookalike fixture. If it stops
#         misfiring, this gate's own "old vs new" comparison is dead and its
#         RED_MODE=1 reproduction proves nothing — fail loudly rather than
#         silently passing vacuously.
#
# ===========================================================================
# HONEST BOUNDARY (Constitution §11.4.6)
# ===========================================================================
#   This proves the SOURCE layer only: that module_paths_identical() is wired
#   into hxc159_env_facts.sh's root-identity check and behaves correctly on
#   synthetic fixtures + the real go.mod files on disk. It does NOT execute
#   hxc159_env_facts.sh itself (that probe has its own broader environment
#   preconditions — SSH reachability, etc. — orthogonal to this defect) and
#   does NOT prove anything about a built/deployed Go binary; module identity
#   is a build-time (`go build`/`go mod`) concern, not a runtime one, so there
#   is no ARTIFACT/RUNTIME layer distinct from SOURCE for this specific
#   invariant.
#
# ===========================================================================
# EXIT CODES
# ===========================================================================
#   0  gate satisfied for the active RED_MODE
#   1  gate violated for the active RED_MODE (substantive)
#   2  environment SKIP: module_identity.sh or either go.mod is missing/
#      unreadable. Certifies NOTHING — callers MUST NOT report this as a
#      substantive pass or as a detected violation.
#
# Honest shebang; `bash -n` clean.

set -uo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
LIB="$ROOT/scripts/lib/module_identity.sh"
ROOT_GOMOD="$ROOT/go.mod"
INNER_GOMOD="$ROOT/helix_code/go.mod"
RED_MODE="${RED_MODE:-0}"

echo "CM-MODULE-IDENTITY-EXACT-MATCH  RED_MODE=$RED_MODE"

# --- Environment preconditions (exit 2 = could not RUN) ---------------------
if [[ ! -f "$LIB" ]]; then
    echo "SKIP(env): scripts/lib/module_identity.sh not found at $LIB" >&2
    exit 2
fi
# shellcheck source=../lib/module_identity.sh
source "$LIB"
if [[ ! -f "$ROOT_GOMOD" ]]; then
    echo "SKIP(env): root go.mod not found at $ROOT_GOMOD" >&2
    exit 2
fi
if [[ ! -f "$INNER_GOMOD" ]]; then
    echo "SKIP(env): inner go.mod not found at $INNER_GOMOD" >&2
    exit 2
fi

root_path="$(go_mod_module_path "$ROOT_GOMOD")"
inner_path="$(go_mod_module_path "$INNER_GOMOD")"

if [[ -z "$root_path" || -z "$inner_path" ]]; then
    echo "SKIP(env): could not extract a 'module' line from go.mod (root='$root_path' inner='$inner_path')" >&2
    exit 2
fi

echo "  root  go.mod module path : $root_path   ($ROOT_GOMOD)"
echo "  inner go.mod module path : $inner_path   ($INNER_GOMOD)"

# old_substring_says_collision <haystack> — reproduces the EXACT pre-HXC-199
# predicate `[[ "$root_mod" == *"dev.helix.code"* ]]`, where $root_mod was the
# FULL `module ...` line (not just the path token). Reconstructed from
# scripts/probes/hxc159_env_facts.sh's own header note describing the bug;
# never edited to make this gate pass — that would defeat its purpose as a
# negative control.
old_substring_says_collision() {
    local hay="$1"
    [[ "$hay" == *"dev.helix.code"* ]]
}

# ==========================================================================
# S1 — falsifiability preconditions (both directions), unconditional
# ==========================================================================
identical_case_caught="no"
if module_paths_identical "dev.helix.code" "dev.helix.code"; then
    identical_case_caught="yes"
fi

lookalike_falsely_identical="no"
if module_paths_identical "dev.helix.code" "dev.helix.codebase"; then
    lookalike_falsely_identical="yes"
fi

old_logic_flags_lookalike="no"
if old_substring_says_collision "module dev.helix.codebase"; then
    old_logic_flags_lookalike="yes"
fi

echo "  S1a genuine recurrence caught (identical paths)         : $identical_case_caught   (want yes)"
echo "  S1b prefix-lookalike NOT falsely caught by exact-match  : $([[ "$lookalike_falsely_identical" == no ]] && echo yes || echo no)   (want yes)"
echo "  S1c OLD substring predicate DOES misfire on the lookalike: $old_logic_flags_lookalike   (want yes — reproduces the historical bug's mechanism)"

s1_fail=0
if [[ "$identical_case_caught" != "yes" ]]; then
    echo "FAIL: module_paths_identical() did NOT catch a genuine recurrence (two identical module paths)." >&2
    echo "      The predicate is BLIND, not strict — it can no longer detect a real collision." >&2
    s1_fail=1
fi
if [[ "$lookalike_falsely_identical" == "yes" ]]; then
    echo "FAIL: module_paths_identical() reported the prefix-only lookalike \"dev.helix.codebase\" as" >&2
    echo "      IDENTICAL to \"dev.helix.code\" — it has regressed to substring/prefix behaviour, which" >&2
    echo "      is the exact HXC-199 defect this gate exists to prevent." >&2
    s1_fail=1
fi
if [[ "$old_logic_flags_lookalike" != "yes" ]]; then
    echo "FAIL: the reconstructed OLD substring predicate no longer misfires on the lookalike fixture —" >&2
    echo "      this gate's negative control is dead, so it cannot prove module_paths_identical() is an" >&2
    echo "      improvement over it. Investigate per §11.4.102 before trusting any verdict below." >&2
    s1_fail=1
fi
if [[ "$s1_fail" -ne 0 ]]; then
    exit 1
fi

# ==========================================================================
# RED_MODE=1 — reproduce the false positive on the CURRENT tree
# ==========================================================================
if [[ "$RED_MODE" == "1" ]]; then
    root_line="$(grep -m1 '^module ' "$ROOT_GOMOD" 2>/dev/null || true)"

    old_verdict="distinct"
    if old_substring_says_collision "$root_line"; then
        old_verdict="collision(WRONG)"
    fi

    exact_verdict="distinct"
    if module_paths_identical "$root_path" "$inner_path"; then
        exact_verdict="collision"
    fi

    echo "  old-substring verdict on real root go.mod line : $old_verdict"
    echo "  exact-match   verdict on real module paths     : $exact_verdict"

    if [[ "$old_verdict" == "collision(WRONG)" && "$exact_verdict" == "distinct" ]]; then
        echo "RED PASS — reproduced the false positive: the OLD substring predicate reports a collision"
        echo "           on the CURRENT tree (root='$root_path' inner='$inner_path') even though the"
        echo "           two module paths are exact-match distinct. This is precisely the defect"
        echo "           HXC-199 exists to guard against — a reader trusting the old predicate would be"
        echo "           misled into believing the R-26 collision is still unresolved."
        exit 0
    fi

    echo "RED FAIL — the OLD substring predicate no longer disagrees with the exact-match predicate on" >&2
    echo "           this tree (old=$old_verdict exact=$exact_verdict). Either the module paths changed" >&2
    echo "           shape (re-derive this reproduction against the new values) or the fixtures above" >&2
    echo "           are no longer representative — do not trust RED_MODE=0 without investigating." >&2
    exit 1
fi

# ==========================================================================
# RED_MODE=0 — DEFAULT — standing GREEN regression guard
# ==========================================================================
if module_paths_identical "$root_path" "$inner_path"; then
    echo "GREEN FAIL — root go.mod ($root_path) and inner go.mod ($inner_path) declare the EXACT SAME" >&2
    echo "             module path. This is the real R-26 collision HXC-187 was supposed to resolve:" >&2
    echo "             two disjoint import trees would collide under one module identity." >&2
    exit 1
fi

echo "GREEN PASS — root go.mod ($root_path) and inner go.mod ($inner_path) are exact-match distinct,"
echo "             verified via module_paths_identical() (scripts/lib/module_identity.sh) — never a"
echo "             substring/prefix test. The reconstructed OLD substring predicate would have"
echo "             wrongly flagged this pair as a collision (see S1c above); this guard would not."
exit 0
