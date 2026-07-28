#!/usr/bin/env bash
# scripts/gates/sources_verified_gate.sh — §11.4.99 "Sources verified" footer gate.
#
# §11.4.99 requires every operator-facing instruction/guide/manual/setup/
# troubleshooting doc to cross-reference its steps against the LATEST official
# online source and carry a `## Sources verified <date>: <urls>` footer. This
# gate scans the operator-facing doc set and reports footer coverage.
#
# Modes:
#   (default / advisory)  print coverage + the docs still missing a footer;
#                         ALWAYS exit 0 — non-blocking while the HXC-030 sweep
#                         is in progress.
#   --enforce             exit 1 if ANY in-scope doc lacks the footer (flip this
#                         on once the sweep is complete, per HXC-030 closure).
#   --check-stale         additionally WARN on footers whose date is > the
#                         §11.4.99 staleness window (180 days; 90 for risk docs)
#                         — advisory only.
#
# Scope = operator-facing docs. EXCLUDES trackers/governance/internal-planning/
# generated trees (they are not operator instructions).
set -uo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

ENFORCE=0; CHECK_STALE=0
for a in "$@"; do
  case "$a" in
    --enforce) ENFORCE=1 ;;
    --check-stale) CHECK_STALE=1 ;;
  esac
done

# Operator-facing doc set: top-level docs/*.md guides + manual/guide subtrees +
# README, MINUS the tracker/governance/internal/generated set.
mapfile -t CANDIDATES < <(
  {
    printf '%s\n' README.md
    find docs -type f -name '*.md' 2>/dev/null
  } | grep -vE '(^|/)(Issues|Issues_Summary|Fixed|Fixed_Summary|CONTINUATION)\.md$' \
    | grep -vE '^docs/(audits|qa|qa_evidence|helix_qa|research|improvements|bluff_proofing|superpowers|codegraph|llms_verifier|adr|materials|architecture|testdata|coverage|exports|scripts|distribution|requests)/' \
    | grep -vE '(^|/)(Status|Status_Summary)\.md$' \
    | sort -u
)

total=0; have=0; missing=0
declare -a MISSING=()
for f in "${CANDIDATES[@]}"; do
  [ -f "$f" ] || continue
  total=$((total+1))
  if grep -qE '^## +Sources verified' "$f" || grep -qiE 'Sources verified [0-9]{4}-[0-9]{2}-[0-9]{2}' "$f"; then
    have=$((have+1))
    if [ "$CHECK_STALE" = 1 ]; then
      d=$(grep -aoE 'Sources verified ([0-9]{4}-[0-9]{2}-[0-9]{2})' "$f" | grep -aoE '[0-9]{4}-[0-9]{2}-[0-9]{2}' | sort | tail -1)
      if [ -n "$d" ]; then
        age=$(( ( $(date +%s) - $(date -d "$d" +%s 2>/dev/null || echo "$(date +%s)") ) / 86400 ))
        [ "$age" -gt 180 ] && echo "WARN §11.4.99 stale ($age d): $f (verified $d)"
      fi
    fi
  else
    missing=$((missing+1)); MISSING+=("$f")
  fi
done

pct=0; [ "$total" -gt 0 ] && pct=$(( have * 100 / total ))
echo "§11.4.99 Sources-verified gate: $have/$total operator-facing docs footered (${pct}%)"
if [ "$missing" -gt 0 ]; then
  echo "  missing footer ($missing):"
  printf '    - %s\n' "${MISSING[@]}"
fi

if [ "$ENFORCE" = 1 ] && [ "$missing" -gt 0 ]; then
  echo "CM-SOURCES-VERIFIED: FAIL (--enforce) — $missing operator doc(s) lack a §11.4.99 footer" >&2
  exit 1
fi
echo "CM-SOURCES-VERIFIED: ADVISORY-PASS ($missing missing; run with --enforce to block once HXC-030 sweep completes)"
exit 0
