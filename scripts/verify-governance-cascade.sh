#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ANCHOR="Article XI.*11.9"
CONST047_ANCHOR="CONST-047"
CONST048_ANCHOR="CONST-048"
CONST049_ANCHOR="CONST-049"
CONST050_ANCHOR="CONST-050"
CONST051_ANCHOR="CONST-051"
CONST052_ANCHOR="CONST-052"
CONST053_ANCHOR="CONST-053"
CONST054_ANCHOR="CONST-054"
CONST055_ANCHOR="CONST-055"
CONST056_ANCHOR="CONST-056"
CONST057_ANCHOR="CONST-057"
CONST058_ANCHOR="CONST-058"
CONST059_ANCHOR="CONST-059"

# ---------------------------------------------------------------------------
# Covenant-114 propagation anchors — DERIVED DYNAMICALLY from the canonical
# constitution (§11.4.32 / CONST-055 / §11.4.157 five-carrier lockstep).
#
# WHY DYNAMIC (2026-07-27 — forensic FACT, captured):
#   This list used to be HARDCODED and topped out at §11.4.165/§11.4.166.
#   The canonical constitution had meanwhile advanced to §11.4.234, so 46
#   genuinely-missing anchors were STRUCTURALLY UNDETECTABLE: the verifier
#   reported "277 checks / 0 failures / PASS" while every consumer carrier
#   lagged the canon by a contiguous block of 42 anchors (§11.4.193..§11.4.234)
#   plus 6 older stragglers (§11.4.18/.22/.39/.63/.64/.100). A hardcoded
#   ceiling re-creates exactly that blind spot on every future canonical
#   addition — so the anchor set is now DISCOVERED AT RUN TIME from the canon
#   and there is no ceiling left to out-grow.
#
# DISCOVERY SOURCE: constitution/Constitution.md — every line that OPENS an
# anchor block. The canon renders openers in four equivalent shapes:
#     "### §11.4.N — …"   "## §11.4.N — …"   "**§11.4.N — …"   "§11.4.N — …"
# all matched by ANCHOR_OPENER_RE below (line-anchored + em-dash guarded, so a
# mid-prose cross-reference such as "composes §11.4.107 / §11.4.112" is NOT
# mistaken for a definition).
#
# TWO TIERS — NEITHER WEAKER THAN THE CHECK THIS REPLACES:
#   TIER-TOKEN  : every canonical anchor's literal token (e.g. `11.4.193`) MUST
#                 appear in every consumer carrier. This is precisely what the
#                 anchors themselves specify — "Propagation gate
#                 CM-COVENANT-114-NNN-PROPAGATION (literal `11.4.NNN` across the
#                 consumer fleet)". Digit-boundary guarded so `11.4.19` can
#                 never be satisfied by a hit inside `11.4.190`.
#   TIER-OPENER : every canonical anchor at or above COVENANT114_BLOCK_FLOOR
#                 MUST **additionally** appear as a genuine BLOCK OPENER, so a
#                 passing cross-reference buried in another anchor's prose
#                 cannot masquerade as propagation. This tier is STRICTLY
#                 STRONGER than the previous implementation, which used an
#                 unanchored fixed-string substring match (`grep -F`) that would
#                 have accepted the literal anywhere on any line.
#
# COVENANT114_BLOCK_FLOOR is a FLOOR, NEVER a ceiling. Anchors below it are the
# parent covenant's internal sub-clauses (§11.4.1..§11.4.67); they are inherited
# BY REFERENCE per CONST-059 / CONST-051(B) and have never been cascaded to
# consumers as standalone blocks — demanding block form for them would be
# over-reach, not rigour. Critically, a FLOOR cannot hide a newly-added
# canonical anchor the way the old CEILING did: every future anchor is numbered
# above it and is therefore enforced automatically, with no edit to this script.
COVENANT114_BLOCK_FLOOR=68
CANON_FILE="$ROOT/constitution/Constitution.md"
ANCHOR_OPENER_RE='^[[:space:]]*(>[[:space:]]*)?(#{1,6}[[:space:]]*)?(\*\*)?§11\.4\.[0-9]+[[:space:]]*(\*\*)?[[:space:]]*—'

if [ ! -f "$CANON_FILE" ]; then
  echo "FAIL: canonical constitution not found at $CANON_FILE — anchor discovery impossible"
  echo "=== Result: 1 failures ==="
  echo "FAIL"
  exit 1
fi

# Numeric, de-duplicated, ascending list of every anchor the canon DEFINES.
mapfile -t COVENANT114_ANCHOR_NUMS < <(
  grep -oE "$ANCHOR_OPENER_RE" "$CANON_FILE" \
    | grep -oE '§11\.4\.[0-9]+' | sed 's/§11\.4\.//' | sort -n -u
)

# Fail-closed (§11.4 anti-bluff): a discovery that comes back empty or absurdly
# small means the canon moved or the regex broke. That is a BROKEN VERIFIER and
# MUST report FAIL — never a vacuous PASS over an empty anchor set.
if [ "${#COVENANT114_ANCHOR_NUMS[@]}" -lt 100 ]; then
  echo "FAIL: anchor discovery returned only ${#COVENANT114_ANCHOR_NUMS[@]} anchors from $CANON_FILE (expected >= 100)"
  echo "      Discovery is broken — refusing to report PASS over an unverified anchor set."
  echo "=== Result: 1 failures ==="
  echo "FAIL"
  exit 1
fi

COVENANT114_TOTAL="${#COVENANT114_ANCHOR_NUMS[@]}"
COVENANT114_MAX="${COVENANT114_ANCHOR_NUMS[$((COVENANT114_TOTAL-1))]}"

# Append every MISSING covenant-114 anchor for one file to $missing_anchors.
check_covenant114_anchors() {
  local f="$1" n
  for n in "${COVENANT114_ANCHOR_NUMS[@]}"; do
    # TIER-TOKEN — literal anchor token, digit-boundary guarded.
    if ! grep -qE "11\.4\.${n}([^0-9]|\$)" "$f" 2>/dev/null; then
      missing_anchors+=" CM-COVENANT-114-${n}-PROPAGATION(token 11.4.${n})"
      continue
    fi
    # TIER-OPENER — genuine block opener required at/above the floor.
    if [ "$n" -ge "$COVENANT114_BLOCK_FLOOR" ] &&
       ! grep -qE "^[[:space:]]*(>[[:space:]]*)?(#{1,6}[[:space:]]*)?(\*\*)?§11\.4\.${n}[[:space:]]*(\*\*)?[[:space:]]*—" "$f" 2>/dev/null; then
      missing_anchors+=" CM-COVENANT-114-${n}-PROPAGATION(opener §11.4.${n} —)"
    fi
  done
}

FAILURES=0
OWNED_FILE="$ROOT/docs/improvements/submodule_owned.txt"
THIRD_PARTY_FILE="$ROOT/docs/improvements/submodule_third_party.txt"

echo "=== Governance Cascade Verification ==="
echo "Repo: $ROOT"
echo ""

# 1. Root governance files
echo "--- Root governance ---"
for f in CONSTITUTION.md AGENTS.md; do
  if grep -q "$ANCHOR" "$ROOT/$f" 2>/dev/null; then
    echo "PASS: root/$f"
  else
    echo "FAIL: root/$f"; FAILURES=$((FAILURES+1))
  fi
done

# 1b. Root govfiles — covenant-114 propagation (§11.4.69, §11.4.75..97).
#     All 5 consumer-extension govfiles must carry every cascaded anchor.
echo ""
echo "--- Root govfiles — covenant-114 propagation (dynamically discovered from canon) ---"
echo "    canon: $CANON_FILE"
echo "    discovered $COVENANT114_TOTAL anchors, highest §11.4.$COVENANT114_MAX; block-opener floor §11.4.$COVENANT114_BLOCK_FLOOR"
for f in CLAUDE.md AGENTS.md QWEN.md GEMINI.md CRUSH.md CONSTITUTION.md; do
  if [ ! -f "$ROOT/$f" ]; then
    echo "FAIL: root/$f — file missing (covenant-114 scope)"; FAILURES=$((FAILURES+1))
    continue
  fi
  missing_anchors=""
  check_covenant114_anchors "$ROOT/$f"
  if [ -z "$missing_anchors" ]; then
    echo "PASS: root/$f (all $COVENANT114_TOTAL covenant-114 anchors present, up to §11.4.$COVENANT114_MAX)"
  else
    echo "FAIL: root/$f — missing:$missing_anchors"; FAILURES=$((FAILURES+1))
  fi
done

# 2. Owned-by-us submodules (require CONSTITUTION.md, CLAUDE.md, AGENTS.md with anchor)
#
# Canonical-path convention (CONST-052 snake_case + CONST-051(C) dependencies layout):
#  - Owned-by-us submodule paths in docs/improvements/submodule_owned.txt MUST be
#    the canonical on-disk paths matching the current submodule layout.
#  - Top-level submodules: lowercase snake_case (`challenges`, `containers`,
#    `github_pages_website`, `helix_agent`, `helix_qa`, `panoptic`, `security`).
#  - Nested-dependency submodules: `dependencies/<org>/<name>` per CONST-051(C).
#    Own-org `<name>` segment is lowercase snake_case per CONST-052 (§11.4.29);
#    the path column tracks the on-disk dir, the URL column keeps the (unchanged)
#    remote repo name. Only genuine third-party submodules keep upstream casing.
#  - Anti-regression: if a listed path does NOT exist on disk, the verifier now
#    FAILS loudly (was previously a silent SKIP, which masked the round-56
#    blemish where 7 stale capitalized entries hid behind "not initialized").
#    A genuinely-uninitialized submodule MUST be initialized BEFORE the cascade
#    can be verified — there is no honest middle state.
echo ""
echo "--- Owned-by-us submodules ---"
if [ -f "$OWNED_FILE" ]; then
  while IFS=' |' read -r sm rest; do
    [ -z "$sm" ] && continue
    if [ ! -d "$ROOT/$sm" ]; then
      echo "FAIL: $sm — path does not exist on disk (verifier path list out of sync; see submodule_owned.txt and CONST-052 / CONST-051(C))"
      FAILURES=$((FAILURES+1))
      continue
    fi
    for f in CONSTITUTION.md CLAUDE.md AGENTS.md; do
      if [ ! -f "$ROOT/$sm/$f" ]; then
        echo "FAIL: $sm/$f — file missing"; FAILURES=$((FAILURES+1))
        continue
      fi
      # Thin-inheritance model (operator decision 2026-06-23; SUPERSEDES the
      # earlier G1 inline-anchor-band requirement). Per CONST-059 + CONST-051(B)
      # / §11.4.28, owned-submodule governance carriers MUST be project-agnostic
      # THIN-INHERITANCE stubs that POINT to the canonical constitution (via a
      # `## INHERITED FROM` heading + the `find_constitution.sh` resolver, never
      # a hardcoded path), NOT inline restatements of the universal anchors.
      # The universal §11.9 / CONST-047..059 / covenant-114 (§11.4.69..165)
      # anchors live in the constitution submodule + the meta-root carriers
      # (verified in section 1 above); a decoupled submodule inherits them by
      # reference, so re-checking the inline literals here would re-impose the
      # very project-coupling CONST-051(B) forbids. The gate therefore asserts
      # the inheritance pointer is present.
      if grep -qiE 'INHERITED FROM|find_constitution|@constitution' "$ROOT/$sm/$f" 2>/dev/null; then
        echo "PASS: $sm/$f (thin-inheritance pointer present — CONST-059 / CONST-051(B) / §11.4.28 decoupled)"
      else
        echo "FAIL: $sm/$f — missing thin-inheritance pointer (## INHERITED FROM / find_constitution.sh); see CONST-059 / CONST-051(B) / §11.4.28"
        FAILURES=$((FAILURES+1))
      fi
    done
  done < "$OWNED_FILE"
else
  echo "SKIP: $OWNED_FILE not found (run P1-T01 first)"
fi

# 3. Third-party submodules — acknowledgement is presence in
#    docs/improvements/submodule_third_party.txt (meta-repo-tracked,
#    manually curated). An optional in-submodule `.helix-governance`
#    file is still accepted as a stronger per-submodule ACK.
#
# Earlier revisions required the per-submodule marker file unconditionally,
# but that file cannot be committed to a third-party submodule's own tree
# without polluting upstream, and a meta-repo cannot track files inside a
# submodule path either — so the marker was unreachable in practice. The
# curated third-party list IS the deliberate acknowledgement.
echo ""
echo "--- Third-party submodules ---"
if [ -f "$THIRD_PARTY_FILE" ]; then
  while IFS=' |' read -r sm rest; do
    [ -z "$sm" ] && continue
    [ ! -d "$ROOT/$sm" ] && echo "SKIP: $sm (not initialized)" && continue
    if [ -f "$ROOT/$sm/.helix-governance" ]; then
      echo "PASS: $sm (in-submodule .helix-governance marker)"
    else
      echo "PASS: $sm (listed in submodule_third_party.txt)"
    fi
  done < "$THIRD_PARTY_FILE"
else
  echo "SKIP: $THIRD_PARTY_FILE not found (run P1-T01 first)"
fi

echo ""
echo "=== Result: $FAILURES failures ==="
if [ "$FAILURES" -eq 0 ]; then
  echo "PASS"
  exit 0
else
  echo "FAIL"
  exit 1
fi
