#!/usr/bin/env bash
# shellcheck shell=bash
#
# scripts/lib/module_identity.sh
# ==============================================================================
# WHOLE-NAME Go MODULE-PATH IDENTITY COMPARISON (HXC-199 / §11.4.111 / §11.4.201)
#
# WHY THIS EXISTS
# ------------------------------------------------------------------------------
# HXC-187 renamed the thin root meta-repo module from `dev.helix.code` to
# `dev.helix.code/meta` so it would stop colliding with the real inner
# application module at helix_code/go.mod, which also declares `dev.helix.code`
# (the R-26 duplicate: `dev.helix.code/internal/theme` used to name two
# disjoint packages depending on which module directory a build started from).
#
# `scripts/probes/hxc159_env_facts.sh` re-derives whether that collision still
# exists by comparing the root module's declared path against the inner
# module's. The FIRST version of that comparison used a substring/prefix test
# (`[[ "$root_mod" == *"dev.helix.code"* ]]`). That is WRONG: `dev.helix.code`
# is a literal PREFIX of `dev.helix.code/meta`, so the substring test still
# matched after the fix landed and the probe kept reporting the collision as
# present (GAP-IN-ITEM) on a tree where it was already resolved — the exact
# false-positive HXC-199 exists to close. Anyone trusting that report could be
# misled into "fixing" (i.e. reverting) already-correct work.
#
# Two module paths name the SAME package identity only when they are EXACTLY
# equal — that is how Go's own module resolution works, and it is the only
# comparison that cannot be fooled by one legitimate path being a prefix of
# another. This file is the SINGLE SOURCE OF TRUTH for that comparison so the
# probe and its standing regression guard (scripts/gates/
# hxc199_module_identity_exact_match_gate.sh) can never drift apart
# (§11.4.201 — assert the real condition, never a brittle proxy).
#
# Usage:
#   source "$(dirname "${BASH_SOURCE[0]}")/../lib/module_identity.sh"
#   if module_paths_identical "dev.helix.code" "dev.helix.code"; then ...      # true
#   if module_paths_identical "dev.helix.code/meta" "dev.helix.code"; then ... # false
# ==============================================================================

# module_paths_identical <path_a> <path_b>
#   Shell-success (0) iff path_a and path_b are the EXACT SAME non-empty
#   string. Shell-failure (1) otherwise — including when either argument is
#   empty: an unreadable/missing go.mod must never be silently treated as "a
#   duplicate of" anything.
module_paths_identical() {
  local a="${1:-}" b="${2:-}"
  [[ -n "$a" && -n "$b" && "$a" == "$b" ]]
}

# go_mod_module_path <path-to-go.mod>
#   Extracts the bare module path token from a go.mod file's `module` line
#   (e.g. "dev.helix.code/meta" from "module dev.helix.code/meta"). Echoes
#   nothing (empty string) if the file is missing or has no `module` line —
#   callers MUST treat an empty result as "unknown", never as a match.
go_mod_module_path() {
  local go_mod_file="${1:-}"
  [[ -n "$go_mod_file" && -f "$go_mod_file" ]] || return 0
  grep -m1 '^module ' "$go_mod_file" 2>/dev/null | awk '{print $2}'
}
