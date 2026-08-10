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
#
# EXIT CONTRACT — ABSENT IS NOT DEFECTIVE  (§11.4.201 / §11.4.3)
# --------------------------------------------------------------
#   0  GREEN — a LIVE process was interrogated and carries the invariant
#   1  FAIL  — a LIVE process was interrogated and VIOLATES the invariant
#   2  SKIP  — there was no live process to interrogate; the guard certified
#              nothing and says so (`SKIP(env):` on stderr, matching the
#              scripts/gates/ convention)
#
# The earlier revision exited 1 when the unit was merely stopped or not
# installed. That is the §11.4.201 false-positive refusal: it reports "the
# gateway is serving in debug mode" on a host where the gateway is not serving
# at all, which is exactly as false as passing a broken one. A sweep wired to
# that guard goes red on every machine that does not happen to be running the
# stack, and a gate that cries wolf gets muted — losing the real signal.
#
# The SKIP path is deliberately NARROW so it cannot fail open (§11.4.69
# CM-NO-FAIL-OPEN-SKIP). It fires ONLY when there is provably no process to
# read: no systemd user manager, unit never installed (LoadState=not-found),
# unit installed but stopped-by-choice (inactive/activating), no MainPID, or an
# unreadable /proc/<pid>/environ. Once a live process exists and is readable,
# the only outcomes are PASS and FAIL.
#
# A JOURNAL WINDOW WITH NO STARTUP MARKER IS NO LONGER A SKIP (review R4,
# 2026-08-10). It was, and that was wrong in the opposite direction: check 1 had
# already interrogated the live process and passed, so the SKIP was discarding
# earned evidence and calling it "certified nothing". On a gateway up long
# enough for journald to rotate its startup line — the production case, measured
# on the very first real sweep — that produced a PERMANENT SKIP. A release-wired
# gate that can never go red is muted just as effectively as one nobody runs.
# The unprovable window now yields PASS-with-caveat: it reports the check that
# ran and passed, and names the cross-check that could not run and why. It
# cannot fail open, because check 1 FAILs before reaching it and the GIN-debug
# scan FAILs on any hit irrespective of provability.
#
# CRUCIALLY, ActiveState=failed is NOT in that set — it FAILs. An independent
# review demonstrated the original hole: a deployed gateway that crash-looped
# into `failed` was read as "absent" and SKIPped, and because the sibling HTTP
# guards SKIP on connection-refused, a completely dead gateway produced three
# SKIPs and a green sweep. Deployed-and-crashed is the loudest defect there is;
# it must never be filed under "not deployed".
set -uo pipefail

UNIT="${HXC229_UNIT:-helixllm-gateway}"
RED_MODE="${RED_MODE:-0}"
fail() { echo "GUARD FAILED (HXC-229): $*" >&2; exit 1; }
skip_env() { echo "SKIP(env): HXC-229 — $*" >&2; exit 2; }

# --- locate the running process -------------------------------------------
# Absence checks run BEFORE any assertion, in both polarities: a defect cannot
# be reproduced (RED) nor proven absent (GREEN) on a subject that is not there.
command -v systemctl >/dev/null 2>&1 || skip_env \
  "systemctl is not on PATH — there is no systemd user manager to interrogate, \
so no running gateway can be located. Not a regression."

systemctl --user show "$UNIT" -p LoadState --value >/dev/null 2>&1 || skip_env \
  "cannot query the systemd user manager for '$UNIT' (no user session bus?). \
The guard could not look; it is not reporting what it saw."

LOAD_STATE="$(systemctl --user show "$UNIT" -p LoadState --value 2>/dev/null)"
[ "$LOAD_STATE" != "not-found" ] || skip_env \
  "unit '$UNIT' is not installed on this host (LoadState=not-found). The \
gateway is absent, which is not the same as the gateway being in debug mode."

ACTIVE_STATE="$(systemctl --user show "$UNIT" -p ActiveState --value 2>/dev/null)"
SUB_STATE="$(systemctl --user show "$UNIT" -p SubState --value 2>/dev/null)"
NRESTARTS="$(systemctl --user show "$UNIT" -p NRestarts --value 2>/dev/null)"
# DEPLOYED-AND-CRASHED IS A DEFECT, NOT AN ABSENCE — and "crashed" is not only
# ActiveState=failed (independent review, F2 then B1).
#
# The first revision SKIPped on any non-active state, so a crash-looping gateway
# read as "not deployed"; combined with the HTTP guards SKIPping on
# connection-refused, a completely dead gateway produced three SKIPs and a green
# sweep (§11.4.69). The second revision FAILed on `failed` — correct but
# insufficient, and the review proved why by reading the REAL unit's config:
#
#   Restart=on-failure  RestartUSec=10s  StartLimitBurst=5  StartLimitIntervalUSec=10s
#
# At most one start can occur per 10s limit interval, so the burst of 5 is
# structurally unexceedable and `failed` is UNREACHABLE for this unit. A gateway
# crashing every 10s forever sits in ActiveState=activating / SubState=auto-restart
# indefinitely. Measured over three restart cycles with that exact config: every
# sample read activating/auto-restart, and the guard SKIPped each time, calling a
# continuously-crashing service "stopped by choice". That is the round-1 hole
# surviving in the one state the deployed unit actually presents.
#
#   LoadState=not-found                       -> never installed     -> SKIP
#   inactive                                  -> stopped by choice   -> SKIP
#   activating + start/start-pre, NRestarts static -> genuinely starting -> SKIP
#   activating + auto-restart (either read), or NRestarts RISING between
#   two reads 2.5s apart                      -> CRASH-LOOPING       -> FAIL
#   failed                                    -> crashed, gave up    -> FAIL
#
# Note the crash-loop test is on a LIVE signal, never the cumulative NRestarts
# alone — that counter is stale after recovery and would false-FAIL a healthy
# unit sampled mid-restart (see the branch itself for the measurement).
case "$ACTIVE_STATE" in
    active) ;;
    failed)
        fail "unit '$UNIT' is installed (LoadState=$LOAD_STATE) but ActiveState=failed \
— the gateway is DEPLOYED AND CRASHED, which is a defect, not an absence. \
$(systemctl --user show "$UNIT" -p Result -p NRestarts --value 2>/dev/null | tr '\n' ' ')"
        ;;
    activating|deactivating|reloading)
        # Distinguishing a crash LOOP from a genuine transition needs a LIVE
        # signal, not a cumulative one. NRestarts is cumulative and STALE:
        # measured on a probe unit that failed once and then recovered, it reads
        # NRestarts=1 while ActiveState=active/SubState=running — forever. So
        # "NRestarts > 0" would FAIL a perfectly healthy gateway that an operator
        # restarted by hand, if the sweep happened to sample it mid-restart,
        # after any historical blip. That is the §11.4.201 false positive this
        # whole change exists to remove, reintroduced through the back door.
        #
        # Two unambiguous live signals instead:
        #   SubState=auto-restart      — systemd is restarting it BECAUSE it failed
        #   NRestarts RISING right now — it is failing again while we watch
        # A genuine `systemctl restart` shows SubState=start/start-pre and a
        # static NRestarts, so it SKIPs as a transition, which is correct.
        # NO ${:-0} default here. Defaulting an ABSENT first read to 0 makes the
        # [ -n ] guard below dead code for that read, so an unreadable first read
        # paired with a stale nonzero second read "proves" a rise that was never
        # observed — a fabricated FAIL (§11.4.6). Keep it empty and let the guard
        # decline to compare.
        _nr_before="$NRESTARTS"
        [ "$SUB_STATE" = "auto-restart" ] && _looping=1 || _looping=0
        if [ "$_looping" -eq 0 ]; then
            sleep 2.5    # spans RestartSec=1..2; the real unit uses RestartUSec=10s
            _nr_after="$(systemctl --user show "$UNIT" -p NRestarts --value 2>/dev/null)"
            _sub_after="$(systemctl --user show "$UNIT" -p SubState --value 2>/dev/null)"
            # BOTH reads must be present AND numeric before any comparison.
            # An empty string does not match *[!0-9]* — it would slip through and
            # be rescued only by the ${:-0} defaults, which is a coincidence, not
            # a guard. And an unreadable first read paired with a nonzero second
            # would otherwise "prove" a rise that was never observed (§11.4.6).
            if [ -n "$_nr_before" ] && [ -n "$_nr_after" ]; then
                case "$_nr_before$_nr_after" in
                    *[!0-9]*) : ;;   # non-numeric on either read: no claim
                    *) [ "$_nr_after" -gt "$_nr_before" ] && _looping=1 ;;
                esac
            fi
            [ "$_sub_after" = "auto-restart" ] && _looping=1
        fi
        if [ "$_looping" -eq 1 ]; then
            fail "unit '$UNIT' is CRASH-LOOPING (ActiveState=$ACTIVE_STATE \
SubState=$SUB_STATE NRestarts=${NRESTARTS:-0}) — systemd is restarting it on a \
timer, so it never reaches ActiveState=failed. A service that cannot stay up is \
a defect, not an absence, and nothing is serving while it flaps."
        fi
        skip_env "unit '$UNIT' is mid-transition (ActiveState=$ACTIVE_STATE \
SubState=$SUB_STATE, NRestarts=${NRESTARTS:-0} and not rising) — a genuine \
start/stop in progress, not a crash loop. Uncertifiable at this instant, not \
defective."
        ;;
    *)  skip_env \
          "unit '$UNIT' is installed but not running (ActiveState=$ACTIVE_STATE \
SubState=$SUB_STATE). A stopped-by-choice unit is uncertifiable, not defective \
— note neither a 'failed' state nor a crash loop reaches this branch; both FAIL \
above."
        ;;
esac

PID="$(systemctl --user show "$UNIT" -p MainPID --value 2>/dev/null)"
[ -n "$PID" ] && [ "$PID" != "0" ] || skip_env \
  "unit '$UNIT' is active but exposes no MainPID — there is no process whose \
environment could be read."

[ -r "/proc/$PID/environ" ] || skip_env \
  "/proc/$PID/environ is not readable (process gone, or a permission boundary). \
An unreadable environment is an absent measurement, never a failed one — \
reporting GIN_MODE 'unset' from a file we could not open would be a fabricated \
finding (§11.4.6)."

ACTIVE_SINCE="$(systemctl --user show "$UNIT" -p ActiveEnterTimestamp --value 2>/dev/null | sed 's/^[A-Za-z]* //')"
[ -n "$ACTIVE_SINCE" ] || skip_env "could not read ActiveEnterTimestamp for '$UNIT' — the journal window cannot be scoped, so the scan would be unbounded (see the scoping trap above)"

# --- check 1: the RUNNING process carries GIN_MODE=release -----------------
if ! tr '\0' '\n' < "/proc/$PID/environ" 2>/dev/null | grep -qx 'GIN_MODE=release'; then
    actual="$(tr '\0' '\n' < "/proc/$PID/environ" 2>/dev/null | grep '^GIN_MODE=' || echo '<unset>')"
    [ "$RED_MODE" = "1" ] && { echo "RED confirmed: live process $PID has $actual"; exit 0; }
    fail "live process $PID does not carry GIN_MODE=release (found: $actual). \
The unit file may still be correct — this is why the guard reads /proc, not the unit."
fi

# --- check 2 (anti-vacuity): the window contains a real startup ------------
WINDOW="$(journalctl --user -u "$UNIT" --since "$ACTIVE_SINCE" --no-pager 2>/dev/null)"
# journalctl prints a "-- No entries --" banner (and "-- Boot ... --" separators)
# for an empty selection. Counting those as content makes the SKIP message below
# report "holds 1 line(s)" for a window that is genuinely empty (review N2), so
# they are excluded from the content count.
LINES="$(printf '%s\n' "$WINDOW" | grep -cvE '^\s*(--.*--\s*)?$' || true)"
STARTUPS="$(printf '%s\n' "$WINDOW" | grep -cE 'Starting|Listening|listen' || true)"

# AN UNPROVABLE WINDOW IS UNCERTIFIABLE, NOT DEFECTIVE (independent review, F7).
#
# The original anti-vacuity rule was right that a window with no startup in it
# proves nothing — a debug-mode gateway dumps its routes only at startup, so a
# clean scan of such a window is worthless. It drew the wrong conclusion from
# that, though: it FAILed. But "this window could not have caught the defect"
# is precisely the definition of no-evidence, and no-evidence is §11.4.3 SKIP,
# never a detected regression. journald rotation and vacuuming retire old
# output on their own schedule, so a healthy gateway running for weeks can lose
# its startup line and start reporting "serving in debug mode" — the §11.4.201
# false-positive refusal, arriving on a timer.
#
# A first attempt narrowed this to the window going fully EMPTY, but that case
# proved unreachable in practice: systemd's own unit records survive even with
# StandardOutput=null and LogLevelMax=emerg (measured — the window never drops
# below 1 line), so the mitigation would have been dead code while the real
# partial-vacuum case still false-FAILed. Treating the whole unprovable class
# as SKIP is both reachable and correct.
#
# This does NOT reintroduce the vacuous pass the original rule guarded against:
# an unprovable window yields SKIP, never PASS. Check 1 above — GIN_MODE=release
# read from the live process's own /proc/<pid>/environ — has already run, so the
# guard is not blind here; it simply declines to certify on a cross-check it
# cannot perform. And a window that DOES contain GIN-debug lines still FAILs
# below regardless, because that is positive evidence of the defect.
# --- check 3: zero GIN-debug in the CURRENT process's own output -----------
# Computed BEFORE the window-provability branch, deliberately. GIN-debug lines
# are POSITIVE evidence of the defect and are conclusive wherever they appear —
# an unprovable window can fail to CONTAIN them, but if it does contain them the
# gateway is demonstrably serving in debug mode. Skipping the scan because the
# window lacks a startup marker would discard a real finding (§11.4.201).
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

# AN UNPROVABLE CROSS-CHECK IS A CAVEAT, NOT A VERDICT OF NOTHING (review R4).
#
# The previous revision SKIPped here, discarding check 1 — which HAD run, on the
# live process's own /proc/<pid>/environ, and HAD passed. "Certified nothing" was
# therefore false: it certified the primary invariant and then threw the result
# away. Measured consequence on the production host 2026-08-08: the gateway has
# been up since before its journal window rotated, so the startup marker is gone
# and G30 SKIPped on its very first real sweep — and would keep SKIPping for as
# long as the process stays up. A release-wired gate stuck in permanent SKIP is
# the muted-gate dynamic approached from the other side: it never goes red, so
# nobody looks, and the fact that it is enforcing nothing is invisible.
#
# THIS CANNOT FAIL OPEN, and the ordering is what guarantees it:
#   - check 1 (GIN_MODE=release, read from the live process) runs FIRST and
#     FAILs outright on violation; reaching this line means it passed.
#   - check 3 (GIN-debug scan) runs ABOVE this branch and FAILs on any hit,
#     regardless of whether the window is provable.
# So the only thing this branch changes is the verdict when both real checks
# passed and only the corroboration was unavailable: PASS-with-caveat instead of
# discarding the evidence. It is a WEAKER pass than the full GREEN below, and it
# says so rather than presenting itself as equivalent (§11.4.6).
if [ "$STARTUPS" -lt 1 ]; then
    echo "PASS-with-caveat (HXC-229): pid=$PID carries GIN_MODE=release, read \
from the live process's own /proc/$PID/environ — the primary invariant is \
ENFORCED and holds. CAVEAT: the corroborating route-dump cross-check could NOT \
run — the journal window since '$ACTIVE_SINCE' holds $LINES line(s) but no \
startup marker (rotated or vacuumed; expected on a long-lived process). A \
debug-mode gateway dumps its routes only at startup, so a clean scan of this \
window proves nothing on its own and is NOT claimed as evidence here. What is \
claimed: the running process's environment, plus $DEBUG GIN-debug line(s) in \
the window (a hit would have FAILed above regardless of provability). To restore \
the full cross-check, restart the unit and re-run."
    exit 0
fi

echo "GREEN (HXC-229): pid=$PID GIN_MODE=release; ${DEBUG} debug lines in ${LINES} \
live lines since '${ACTIVE_SINCE}'; ${STARTUPS} startup marker(s) present (non-vacuous)."
