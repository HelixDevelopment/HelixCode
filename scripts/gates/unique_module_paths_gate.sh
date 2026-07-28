#!/usr/bin/env bash
# CM-UNIQUE-MODULE-PATHS — no two LIVE Go modules may declare the same module path.
#
# Authored in T-P0.03.5. Deliberately NOT wired into any pre-build sweep yet:
# it currently FAILS (by design — it detects the real R-26 collision), and six
# agents are working concurrently in this checkout. Wiring it is a later-phase
# step, to be done in the same change that lands the D-7 rename. Run it by hand:
#
#   scripts/gates/unique_module_paths_gate.sh          # gate
#   scripts/gates/unique_module_paths_gate.sh --list   # show the live module set
#
# WHY THE SCOPING IS NOT OPTIONAL (measured, not assumed):
# a naive "collect every `module` line and fail on duplicates" gate fails on this
# repo for reasons that are not defects — archived source snapshots under docs/,
# per-run QA harness copies that legitimately reuse a module path, third-party
# vendored fixtures, and generated test-result trees. A gate that cries wolf gets
# disabled in its first week, so it scopes to LIVE modules: those reachable from
# a build, excluding documentation, captured evidence, test output and vendored
# third-party trees.
#
# Exit 0 = no duplicate live module paths. Exit 1 = duplicates found.
set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT" || exit 1

# Non-live trees. Each entry is justified; do not extend without one.
#   docs/            — archived source snapshots + captured QA harnesses
#   qa-results/      — captured evidence (§11.4.83)
#   test-results/    — generated challenge output
#   cli_agents/      — third-party reference CLI agents (vendored, not ours)
#   cli_agents_resources/ — third-party reference resources (vendored)
#   node_modules/ vendor/ .git/ — tool-mandated / vendored
is_live() {
  case "$1" in
    docs/*|*/docs/*) return 1 ;;
    qa-results/*|*/qa-results/*) return 1 ;;
    test-results/*|*/test-results/*) return 1 ;;
    cli_agents/*|cli_agents_resources/*) return 1 ;;
    */node_modules/*|*/vendor/*|*/.git/*) return 1 ;;
  esac
  return 0
}

declare -a live_paths=() live_files=()
while IFS= read -r f; do
  rel="${f#./}"
  is_live "$rel" || continue
  m="$(grep -m1 '^module ' "$f" 2>/dev/null | awk '{print $2}')"
  [[ -n "$m" ]] || continue
  live_paths+=("$m"); live_files+=("$rel")
done < <(find . -name go.mod -not -path '*/.git/*' 2>/dev/null | sort)

if [[ "${1:-}" == "--list" ]]; then
  printf '%-46s %s\n' "MODULE PATH" "go.mod"
  for i in "${!live_paths[@]}"; do printf '%-46s %s\n' "${live_paths[$i]}" "${live_files[$i]}"; done
  echo "(${#live_paths[@]} live modules)"
  exit 0
fi

dupes="$(printf '%s\n' "${live_paths[@]}" | sort | uniq -d)"

if [[ -z "$dupes" ]]; then
  echo "PASS CM-UNIQUE-MODULE-PATHS — ${#live_paths[@]} live modules, all paths distinct"
  exit 0
fi

echo "FAIL CM-UNIQUE-MODULE-PATHS — duplicate module path(s) among ${#live_paths[@]} live modules:"
while IFS= read -r d; do
  [[ -n "$d" ]] || continue
  echo "  '$d' declared by:"
  for i in "${!live_paths[@]}"; do
    [[ "${live_paths[$i]}" == "$d" ]] && echo "    - ${live_files[$i]}"
  done
  # Name the concrete hazard: import paths that resolve to different packages.
  for i in "${!live_paths[@]}"; do
    [[ "${live_paths[$i]}" == "$d" ]] || continue
    base="$(dirname "${live_files[$i]}")"; [[ "$base" == "." ]] && base=""
    for sub in $(find "${base:-.}/internal" -maxdepth 1 -mindepth 1 -type d 2>/dev/null); do
      echo "      colliding import candidate: $d/internal/$(basename "$sub")  <- ${base:-<root>}"
    done
  done
done <<< "$dupes"
echo
echo "Fix per spec.md D-7: rename the thin root module; bind nothing new to it."
exit 1
