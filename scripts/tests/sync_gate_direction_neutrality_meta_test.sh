#!/usr/bin/env bash
# scripts/tests/sync_gate_direction_neutrality_meta_test.sh — §1.1 paired-mutation
# meta-test for the DIRECTION-NEUTRALITY of scripts/gates/workable_items_sync_gate.sh
# (HXC-252).
#
# WHY THIS EXISTS
# ---------------
# The gate detects md↔db drift with a SYMMETRIC `diff -q`. A symmetric diff
# reports THAT two sides differ; it cannot compute WHICH side is stale — that
# fact is simply not in the comparison. The gate nonetheless asserted a
# direction ("committed DB is STALE vs Issues.md") and prescribed the remedy
# that follows from it (`sync md-to-db`).
#
# In the DB-newer case — the one actually hit on 2026-08-09 — that diagnosis is
# FALSE and the prescription is DESTRUCTIVE: md-to-db replaces the §11.4.95 SSoT
# with the derived document. Measured at the time: a 36-item stale doc would
# have overwritten a 46-item DB, deleting 10 items and resurrecting 8 closed
# ones. The trap fires precisely when a maintainer is already worried about
# consistency and is therefore most inclined to obey the tool (§9.2).
#
# WHAT THIS ASSERTS
# -----------------
# That the gate's drift messages do not assert a direction the gate never
# computed, that both remedies are enumerated with the condition under which
# each applies — so the reader determines direction from evidence rather than
# from an assertion (§11.4.6) — and that each remedy is BOUND to the right
# condition, with the data-loss warning on the destructive one.
#
# The binding half was added after an independent review (round 5) showed the
# presence half is not sufficient: see the "(4) ASSOCIATION" block below.
#
# It asserts on the MESSAGE TEXT deliberately. The message IS the defect: the
# gate's detection logic was never wrong, only its advice, and advice is the
# artifact a maintainer acts on. There is no runtime signal for "the operator
# read a sentence and destroyed the SSoT", so the message is the observable.
#
# POLARITY (§11.4.115): RED_MODE=1 asserts the PRE-FIX shape (a bare directional
# staleness claim + an unconditional md-to-db prescription). It MUST fail on a
# fixed artifact. RED_MODE=0 (default) is the standing GREEN regression guard
# (§11.4.135) asserting the defect is ABSENT.
#
# Exit: 0 assertions hold | 1 an assertion failed | 2 could not run (§11.4.3)
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# Overridable so RED stays re-runnable against a PRE-FIX revision forever
# (§11.4.115 requires the RED to reproduce on the broken artifact, and the
# broken artifact only exists in history once the fix lands):
#   git show be5d56be:scripts/gates/workable_items_sync_gate.sh > /tmp/pre.sh
#   SYNC_GATE_PATH=/tmp/pre.sh RED_MODE=1 bash scripts/tests/sync_gate_direction_neutrality_meta_test.sh
GATE="${SYNC_GATE_PATH:-$ROOT/scripts/gates/workable_items_sync_gate.sh}"
RED_MODE="${RED_MODE:-0}"
PASSES=0; FAILS=0

[ -r "$GATE" ] || { echo "SKIP(env): gate not readable: $GATE" >&2; exit 2; }

ck() { # ck <desc> <condition-rc>
    if [ "$2" -eq 0 ]; then PASSES=$((PASSES+1)); printf '  ok    %s\n' "$1"
    else FAILS=$((FAILS+1)); printf '  NOT OK %s\n' "$1"; fi
}

# The drift branches are the two `fail` calls that compare the DB projection
# against the live docs — i.e. the ones whose message a maintainer acts on.
# Extracted by their stable marker rather than by line number, so the test does
# not silently stop testing anything when the file is edited (§11.4.201: a test
# that matches nothing reports "clean" for the same reason a passing one does).
# Anchored on the section number alone. An earlier revision anchored on the
# full prose of the heading and stopped matching the moment the fix reworded it
# — the test SKIPped instead of asserting, which is honest but useless. The
# section number is the stable part.
DRIFT_REGION="$(sed -n '/^# (3) /,$p' "$GATE")"
if [ -z "$DRIFT_REGION" ]; then
    echo "SKIP(env): could not locate the drift-branch block in $GATE — the \
marker moved. Refusing to report on a region that was never extracted." >&2
    exit 2
fi

# ONLY the REMEDY-ADVICE messages: the fail() calls inside the `if ! diff -q`
# drift branches. Two earlier revisions of this extraction were wrong in the
# same direction and are worth recording, because both would have certified the
# UNFIXED gate as fixed (§11.4.201):
#   a) scanning the whole region matched line 78's `"$BIN" sync db-to-md ...` —
#      the gate's own INVOCATION of the renderer, not advice to anybody;
#   b) scanning every fail() message matched `fail "db-to-md on committed DB
#      failed: ..."` — a report that a TOOL broke, not a remedy for drift.
# Neither string is advice a maintainer can act on, yet both satisfied a naive
# `grep db-to-md`. The drift branches are structurally identifiable — they are
# exactly the `if ! diff -q ...; then fail ...; fi` blocks — so anchor there and
# nowhere else.
#
# Two artifact shapes must both be readable, because RED runs against the
# pre-fix revision and GREEN against the fixed one:
#   PRE-FIX : the advice is inline in each `if ! diff -q ...; then fail "..."`
#   FIXED   : the advice is factored into a drift_remedy() helper the branches call
# When the helper exists it IS the message by construction, so it is the only
# thing scanned — which also keeps the `"$BIN" sync db-to-md` invocation on line
# 78 (outside the helper) from ever being mistaken for advice again.
if printf '%s\n' "$DRIFT_REGION" | grep -q '^drift_remedy()'; then
    DRIFT_MSGS="$(printf '%s\n' "$DRIFT_REGION" | awk '/^drift_remedy\(\)/{inf=1} inf{print} inf && /^}$/{exit}')"
    # WIRING (§11.4.226): a perfectly-worded helper nobody calls is not advice.
    # Both drift branches must actually emit it, or the message a maintainer
    # sees is still whatever the branch does instead.
    WIRED="$(printf '%s\n' "$DRIFT_REGION" | awk '
        /if ! diff -q/ { inblk=1 }
        inblk && /drift_remedy/ { n++ ; inblk=0 }
        END { print n+0 }')"
    [ "${WIRED:-0}" -ge 2 ] || {
        echo "GUARD FAILED (HXC-252): drift_remedy() exists but only ${WIRED:-0} \
of the 2 drift branches call it — the other still emits its own message, which \
is the text this test cannot see and the maintainer will act on." >&2; exit 1; }
else
    DRIFT_MSGS="$(printf '%s\n' "$DRIFT_REGION" | awk '
        /if ! diff -q/ { inblk=1 }
        inblk && /fail "/ { inmsg=1 }
        inmsg { print }
        inmsg && !/\\$/ { inmsg=0 }
        /^fi$|^\}$/ { inblk=0 }
    ')"
fi
if [ -z "$(printf '%s' "$DRIFT_MSGS" | tr -d '[:space:]')" ]; then
    echo "SKIP(env): no drift advice found in $GATE — neither a drift_remedy() \
helper nor a fail() inside a diff-q branch. A test that matched nothing must \
not report clean." >&2
    exit 2
fi

# --- the three properties the fix must establish ---------------------------

# (1) No bare directional staleness ASSERTION. "X is STALE vs Y" states as fact
#     something a symmetric diff cannot know.
echo "$DRIFT_MSGS" | grep -qE 'is STALE vs'; HAS_STALE_CLAIM=$?

# (2) BOTH remedies enumerated — a reader cannot choose a direction from a
#     message that names only one.
echo "$DRIFT_MSGS" | grep -q 'db-to-md'; HAS_DBTOMD=$?
echo "$DRIFT_MSGS" | grep -q 'md-to-db'; HAS_MDTODB=$?

# (3) The destructive remedy carries its data-loss warning. Enumerating both
#     without saying which one can delete the SSoT just moves the trap.
echo "$DRIFT_MSGS" | grep -qiE 'overwrit|destroy|data.loss|OVERWRITES'; HAS_WARNING=$?

if [ "$RED_MODE" = "1" ]; then
    # RED reproduces the defect on the CURRENT artifact: a directional claim
    # present, and the safe remedy absent.
    if [ "$HAS_STALE_CLAIM" -eq 0 ] && [ "$HAS_DBTOMD" -ne 0 ]; then
        echo "RED confirmed: the gate ASSERTS a direction ('is STALE vs') that a \
symmetric diff cannot compute, and never names the db-to-md remedy — so the only \
advice it gives is the one that overwrites the §11.4.95 SSoT."
        exit 0
    fi
    echo "RED baseline did NOT reproduce: the gate no longer carries the bare \
directional prescription (stale-claim=$([ $HAS_STALE_CLAIM -eq 0 ] && echo present || echo absent), \
db-to-md=$([ $HAS_DBTOMD -eq 0 ] && echo present || echo absent)). This artifact \
already carries the fix — run RED against a pre-fix revision." >&2
    exit 1
fi

# (4) ASSOCIATION, not merely PRESENCE  (independent review round 5, F1)
# ----------------------------------------------------------------------
# Checks (1)-(3) are TOKEN-PRESENCE checks, and presence is not advice. A review
# mutation SWAPPED the two remedies inside drift_remedy() — leaving "(a) if the
# DB is the newer side ... run 'sync md-to-db'", which is the DESTRUCTIVE
# prescription bound to the exact condition under which it deletes the SSoT,
# with the WARNING now pointing at the safe remedy — and this test reported
# 4 ok / 0 not ok. Every token it looked for was still present; only the
# BINDINGS had inverted. The pairing IS the advice, and the pairing is what the
# original HIGH was about, so presence checks alone guard the pre-fix STRING
# SHAPE rather than the defect class.
#
# Bind them mechanically instead. The message is flattened to the sentence a
# maintainer actually reads (source line-wrapping is not part of the advice),
# then each anchor is resolved to the remedy it governs the way the PROSE reads,
# not by raw distance:
#   selects(cond)   = the first remedy AFTER the condition ("if X ... run R"),
#                     falling back to the nearest preceding one if a message
#                     puts the remedy first ("run R if X").
#   attaches(warn)  = the last remedy BEFORE the warning, since a warning
#                     back-refers ("... run R. WARNING: R overwrites ..."),
#                     falling back to the first following one.
# Then:
#   A  the overwrite WARNING must attach to md-to-db  (the destructive remedy)
#   B  the DB-is-newer condition must select db-to-md (the safe one)
#   C  the doc-is-newer condition must select md-to-db
# Order-independent by construction, so re-ordering or re-lettering the clauses
# does not fail a message that is still correct — but swapping either the
# REMEDIES or the CONDITIONS inverts at least two of the three.
#
# Raw nearest-neighbour was tried first and is weaker: a condition sitting at
# the START of its clause can be closer to the PREVIOUS clause's remedy token
# than to its own, so a condition-swap scored one check as satisfied purely on
# distance. Reading forward from the condition is both more faithful to the
# sentence and strictly more sensitive (§11.4.201: measured, not assumed).
FLAT="$(printf '%s\n' "$DRIFT_MSGS" | sed 's/\\[[:space:]]*$//' | tr '\n' ' ' | tr -s '[:space:]' ' ')"
_ASSOC="$(printf '%s\n' "$FLAT" | awk '
function selects(p) {
    if (o_safe > p && o_dest > p) return (o_safe < o_dest) ? "safe" : "dest"
    if (o_safe > p) return "safe"
    if (o_dest > p) return "dest"
    return (o_safe > o_dest) ? "safe" : "dest"
}
function attaches(p) {
    if (o_safe < p && o_dest < p) return (o_safe > o_dest) ? "safe" : "dest"
    if (o_safe < p) return "safe"
    if (o_dest < p) return "dest"
    return (o_safe < o_dest) ? "safe" : "dest"
}
{
    t = tolower($0)
    o_safe = index(t, "db-to-md")
    o_dest = index(t, "md-to-db")
    o_warn = match(t, /overwrit|destroy|data.loss/)
    o_dbn  = match(t, /db is (the )?newer/)
    o_docn = match(t, /carries newer|docs? (is|are) (the )?newer/)
    miss = ""
    if (!o_safe) miss = miss " db-to-md"
    if (!o_dest) miss = miss " md-to-db"
    if (!o_warn) miss = miss " overwrite-warning"
    if (!o_dbn)  miss = miss " DB-is-newer-condition"
    if (!o_docn) miss = miss " doc-is-newer-condition"
    if (miss != "") { print "MISSING" miss; exit }
    printf "%d %d %d\n", (attaches(o_warn) == "dest"), \
                         (selects(o_dbn)   == "safe"), \
                         (selects(o_docn)  == "dest")
}')"
# A missing anchor means the association is UNVERIFIABLE, which must be loud.
# Scoring it as a silent pass is the §11.4.201 shape this whole file argues
# against: a check that matched nothing would report clean for the same reason a
# satisfied one does.
if [ "${_ASSOC%% *}" = "MISSING" ]; then
    echo "GUARD FAILED (HXC-252): cannot verify remedy/condition BINDING — the \
drift message no longer contains these anchors:${_ASSOC#MISSING}. Either the \
message dropped a remedy or a condition, or it was reworded past this test's \
locators. Refusing to certify a binding that was never located." >&2
    exit 1
fi
read -r ASSOC_WARN ASSOC_DBNEW ASSOC_DOCNEW <<<"$_ASSOC"

echo "=== HXC-252 direction-neutrality guard (RED_MODE=0: defect must be ABSENT) ==="
ck "no bare 'is STALE vs' directional assertion"        "$([ "$HAS_STALE_CLAIM" -ne 0 ] && echo 0 || echo 1)"
ck "names the db-to-md remedy (DB newer -> regen docs)" "$HAS_DBTOMD"
ck "names the md-to-db remedy (docs newer)"             "$HAS_MDTODB"
ck "warns that one remedy overwrites the SSoT"          "$HAS_WARNING"
ck "the overwrite warning binds to md-to-db, not db-to-md"   "$([ "$ASSOC_WARN"   = 1 ] && echo 0 || echo 1)"
ck "'DB is newer' selects db-to-md (the SAFE remedy)"        "$([ "$ASSOC_DBNEW"  = 1 ] && echo 0 || echo 1)"
ck "'doc is newer' selects md-to-db (the DESTRUCTIVE one)"   "$([ "$ASSOC_DOCNEW" = 1 ] && echo 0 || echo 1)"

echo "--- $PASSES ok, $FAILS not ok"
[ "$FAILS" -eq 0 ] || { echo "GUARD FAILED (HXC-252): the sync gate prescribes a \
direction it never computed — following it can delete the SSoT (§9.2)." >&2; exit 1; }
echo "GREEN (HXC-252): the drift message states what was detected, enumerates both \
remedies, BINDS each to the condition under which it is correct, and puts the \
data-loss warning on the destructive one."
