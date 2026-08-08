#!/usr/bin/env bash
# HXC-229 standing regression guard (§11.4.135) — helixllm-gateway must run in
# Gin RELEASE mode, proven on the RUNNING PROCESS, not on the unit file.
#
# WHY THIS GUARD EXISTS
# --------------------
# HXC-229 was fixed by adding `Environment=GIN_MODE=release` to the unit, and
# verified live once. An independent review refused to close it: verified-once
# is not guarded, and the recurrence vector here is concrete rather than
# theoretical. The unit is a rendered TEMPLATE (@HELIX_ROOT@, @HELIXLLM_BIN@),
# and it DELIBERATELY places Environment= before EnvironmentFile= so an operator
# .env can override GIN_MODE back to debug. A re-render, a stale template, or an
# operator override each silently restores the defect. That is the §11.4.135
# forensic-anchor pattern: a source-side fix that no test mirrors or re-runs.
#
# WHY IT ASSERTS THE PROCESS, NOT THE UNIT FILE
# ---------------------------------------------
# Grepping the unit for GIN_MODE is a §11.4 config-only assertion: it proves
# what was WRITTEN, never what is RUNNING. The unit could be correct while the
# live process inherited debug from .env, from a stale rendered copy, or from a
# daemon-reload that never happened. /proc/<pid>/environ is the running truth.
#
# THE SCOPING TRAP THIS GUARD IS BUILT AROUND  (§11.4.6)
# -----------------------------------------------------
# A journal scan by LINE COUNT reaches back past the fix. Measured 2026-08-08:
# `journalctl -n 120` showed 5 GIN-debug lines — all from PID 4140522 at
# 2026-08-07T12:52:13, a PRE-fix run — while the live process (started
# 23:40:15) had 0 out of 89. A line-count window would false-FAIL forever on
# history that can never change. This guard scopes to ActiveEnterTimestamp so
# it reads ONLY the current process's output.
#
# ANTI-VACUITY  (§11.4.1)
# -----------------------
# "No debug lines" is trivially true of an empty window. The guard therefore
# REFUSES to pass unless the window also contains a real startup — the moment a
# debug-mode gateway would have dumped its route table. Absence of evidence is
# not evidence of absence.
#
# POLARITY (§11.4.115): RED_MODE=1 asserts the pre-fix shape (debug mode
# present). It MUST fail on a fixed artifact.
set -uo pipefail

UNIT="${HXC229_UNIT:-helixllm-gateway}"
RED_MODE="${RED_MODE:-0}"
fail() { echo "GUARD FAILED (HXC-229): $*" >&2; exit 1; }

# --- locate the running process -------------------------------------------
PID="$(systemctl --user show "$UNIT" -p MainPID --value 2>/dev/null)"
[ -n "$PID" ] && [ "$PID" != "0" ] || fail \
  "unit '$UNIT' has no running MainPID. This guard asserts the RUNNING process; \
a stopped unit cannot be proven release-mode. Start it, or SKIP with reason."

ACTIVE_SINCE="$(systemctl --user show "$UNIT" -p ActiveEnterTimestamp --value 2>/dev/null | sed 's/^[A-Za-z]* //')"
[ -n "$ACTIVE_SINCE" ] || fail "could not read ActiveEnterTimestamp for '$UNIT'"

# --- check 1: the RUNNING process carries GIN_MODE=release -----------------
if ! tr '\0' '\n' < "/proc/$PID/environ" 2>/dev/null | grep -qx 'GIN_MODE=release'; then
    actual="$(tr '\0' '\n' < "/proc/$PID/environ" 2>/dev/null | grep '^GIN_MODE=' || echo '<unset>')"
    [ "$RED_MODE" = "1" ] && { echo "RED confirmed: live process $PID has $actual"; exit 0; }
    fail "live process $PID does not carry GIN_MODE=release (found: $actual). \
The unit file may still be correct — this is why the guard reads /proc, not the unit."
fi

# --- check 2 (anti-vacuity): the window contains a real startup ------------
WINDOW="$(journalctl --user -u "$UNIT" --since "$ACTIVE_SINCE" --no-pager 2>/dev/null)"
LINES="$(printf '%s\n' "$WINDOW" | grep -c . || true)"
STARTUPS="$(printf '%s\n' "$WINDOW" | grep -cE 'Starting|Listening|listen' || true)"
[ "$STARTUPS" -ge 1 ] || fail \
  "window since '$ACTIVE_SINCE' holds $LINES lines but NO startup marker. \
A clean scan of a window with no startup proves nothing — a debug-mode gateway \
dumps its routes only at startup, so this window could not have caught it."

# --- check 3: zero GIN-debug in the CURRENT process's own output -----------
DEBUG="$(printf '%s\n' "$WINDOW" | grep -c 'GIN-debug' || true)"

if [ "$RED_MODE" = "1" ]; then
    [ "$DEBUG" -gt 0 ] || fail \
      "RED baseline did NOT reproduce: 0 GIN-debug lines in the live window. \
This artifact already carries the fix — run RED against a pre-fix deployment."
    echo "RED confirmed: $DEBUG GIN-debug lines from the live process"
    exit 0
fi

[ "$DEBUG" -eq 0 ] || fail \
  "$DEBUG GIN-debug line(s) from the LIVE process (pid $PID, since '$ACTIVE_SINCE'). \
The gateway is serving in debug mode."

echo "GREEN (HXC-229): pid=$PID GIN_MODE=release; ${DEBUG} debug lines in ${LINES} \
live lines since '${ACTIVE_SINCE}'; ${STARTUPS} startup marker(s) present (non-vacuous)."
