#!/usr/bin/env bash
# Falsification battery for the three live-service standing guards wired as
# G30/G31/G32 in scripts/verify-all-constitution-rules.sh.
#
# WHY THIS EXISTS
# ---------------
# The sweep's G30-G32 block originally carried a prose claim — "Verified
# 2026-08-09 across 9 defect shapes ... 9 FAIL, 0 SKIP" — with no cited evidence
# artifact and nothing re-runnable behind it. An independent review named that
# for what it is: the uncited-claim class the block's own rationale (§11.4.226)
# argues against, and a §1.1 / CONST-055 gap, since a gate whose falsification
# lives only in a commit message cannot be re-falsified by the next maintainer.
# This script replaces the claim with a mechanism.
#
# WHAT IT ASSERTS
# ---------------
# The three-way exit contract, on subjects it constructs itself:
#     PASS(0)  live and correct
#     FAIL(1)  REACHABLE / RUNNING but defective   <- the falsification
#     SKIP(2)  provably absent                     <- must not swallow a defect
#
# Every FAIL case is served over HTTP 200 or by a live process, because a guard
# that only notices a closed port has not been falsified — it has been unplugged.
#
# It also carries a REGRESSION case for each hole an independent review found in
# the first revision of these guards, so none can silently return:
#     R1  proxy env vars must not turn a LIVE subject into SKIP  (curl honours
#         https_proxy, so exit 7 could describe the proxy, not the target)
#     R2  a DEPLOYED-and-CRASHED unit (ActiveState=failed) must FAIL, not SKIP
#     R3  a crashed ANALYZER must FAIL in both polarities, never "RED confirmed"
#     R4  a CRASH-LOOPING unit under the production Restart policy must FAIL —
#         it never reaches `failed`, so R2 alone does not cover it
#
# Exit: 0 all assertions hold | 1 an assertion failed | 2 could not run (§11.4.3)
#
# Usage: bash scripts/testing/guard_live_service_falsification.sh
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
G229="$ROOT/scripts/testing/guard_hxc229_gateway_release_mode.sh"
G233="$ROOT/scripts/testing/guard_hxc233_completion_path_live.sh"
G244="$ROOT/scripts/testing/guard_hxc244_health_components_registered.sh"
PASSES=0; FAILS=0
TMP="$(mktemp -d)"; STUB_PIDS=(); UNITS=()

# SINGLE-OWNER PER INVOCATION (§11.4.119). This battery binds fixed-purpose
# stub ports and creates named transient units, so two concurrent runs collide:
# the second run's `systemd-run --unit=<name>` and port binds lose to the first,
# and BOTH runs then measure each other's subjects. Observed for real — a
# reviewer ran this battery on the same host at the same time as its author and
# the run reported 2 spurious "not ok" assertions that reproduce nowhere in
# isolation. A harness whose verdict depends on who else is running is not an
# instrument (§11.4.201: a false FAIL is as damaging as a false PASS).
#
# Two independent defences, because either alone is thin:
#   1. an advisory flock, so concurrent invocations SERIALISE rather than race
#   2. PID-suffixed unit names and a per-run port base, so even if the lock is
#      unavailable (no flock binary) the two runs do not address the same objects
# The lock path is FIXED, not TMPDIR-derived (review W1): callers routinely run
# with private TMPDIRs, and two invocations under different TMPDIRs would take
# two different locks and never serialise — leaving PID-scoping to carry alone.
LOCK_FILE="/tmp/.guard_live_service_falsification.$(id -u).lock"
if command -v flock >/dev/null 2>&1; then
    # An open FAILURE and a lock TIMEOUT are different facts and must not share a
    # message (review W2): reporting "another invocation held the lock" when the
    # lock file could not even be opened is a false diagnosis (§11.4.201).
    # The braces are load-bearing. `exec 9>FILE 2>/dev/null` applies BOTH
    # redirections to the shell itself and they PERSIST — so on the success path
    # (the normal case) stderr stays pointed at /dev/null for the rest of the
    # run, silently swallowing the self-test refusal, the flock-timeout SKIP
    # reason, unit()'s abort, stub-never-came-up SKIPs, and the final
    # "BATTERY FAILED" line. Exit codes stayed correct while every diagnostic
    # vanished: a silent refusal is the §11.4.3 loudness violation this battery
    # exists to police. Wrapping the exec in a group scopes 2>/dev/null to the
    # group, so it suppresses only bash's open-error message.
    if { exec 9>"$LOCK_FILE"; } 2>/dev/null; then
        if ! flock -w 600 9 2>/dev/null; then
            echo "SKIP(env): another invocation of this battery held \
$LOCK_FILE for >600s. Concurrent runs are serialised on purpose; retry when it \
finishes." >&2
            exit 2
        fi
    else
        echo "NOTE: could not open $LOCK_FILE for locking (permissions? read-only \
/tmp?). Proceeding WITHOUT the lock — PID-scoped unit names and ports still keep \
concurrent runs disjoint. This is not a lock timeout." >&2
    fi
fi
RUN="$$"
# Port base stays inside an unprivileged, rarely-used band and is derived from
# the PID so parallel runs occupy disjoint ranges.
#
# THE STRIDE MUST EXCEED THE NUMBER OF PORTS USED. At stride 10 with indices
# 1..9 the range was exactly full, so adding a tenth subject would have handed
# index 10 to the NEXT pid's index 0 — reintroducing, silently, the very
# cross-run collision the PID scoping exists to prevent (§11.4.119). The stride
# is 16 for 11 indices, leaving headroom for the next subject; the whole band
# (19000-25399) stays below the default ip_local_port_range floor of 32768, so
# it cannot collide with an ephemeral port either.
PORT_BASE=$(( 19000 + (RUN % 400) * 16 ))
P_H_EMPTY=$((PORT_BASE+1)); P_H_UNNAMED=$((PORT_BASE+2))
P_C_ERR=$((PORT_BASE+3));   P_C_NOCH=$((PORT_BASE+4))
P_C_WRONG=$((PORT_BASE+5)); P_C_NOMODEL=$((PORT_BASE+6))
P_WEDGE=$((PORT_BASE+7));   P_EMPTY=$((PORT_BASE+8))
P_CLOSED=$((PORT_BASE+9))   # deliberately never bound — the absence subject
P_RESET=$((PORT_BASE+10))

cleanup() {  # §11.4.14: reap every child, always, even on early exit
    # Stubs are reaped BY PID, never by `pkill -f <pattern>`: a cmdline pattern
    # also matches the reaping shell's own command line, so the helper kills
    # itself (§11.4.174 / §12.12). Measured during this battery's development —
    # the symptom is an inexplicable SIGTERM (exit 143) mid-script.
    for p in "${STUB_PIDS[@]:-}"; do [ -n "$p" ] && kill "$p" 2>/dev/null; done
    # `stop` alone leaves a deliberately-crashed unit in ActiveState=failed,
    # which lingers in `systemctl --user list-units` and pollutes the next run.
    for u in "${UNITS[@]:-}"; do
        systemctl --user stop "$u" 2>/dev/null
        systemctl --user reset-failed "$u" 2>/dev/null
    done
    rm -rf "$TMP"
}
trap cleanup EXIT

for tool in curl python3 systemd-run systemctl; do
    command -v "$tool" >/dev/null 2>&1 || { echo "SKIP(env): $tool not on PATH" >&2; exit 2; }
done

# Reap PROVABLY-stale units from earlier runs (§11.4.180 applied to transient
# units). A SIGKILLed battery cannot run its EXIT trap, so its gfals-<pid>-*
# units survive in ActiveState=failed and clutter `systemctl --user list-units`
# for every later reader. Because unit names carry the owning battery's pid, a
# stale one is decidable rather than guessed: the owner is stale IFF /proc/<pid>
# no longer exists. A unit whose owner is ALIVE is never touched — that would be
# this script sabotaging a concurrent run of itself (§11.4.174 / §9.2).
for _u in $(systemctl --user list-units --all --no-legend 'gfals-*' 2>/dev/null \
            | sed 's/^[^a-zA-Z]*//' | awk '{print $1}'); do
    _owner="$(printf '%s' "$_u" | sed -nE 's/^gfals-([0-9]+)-.*$/\1/p')"
    [ -n "$_owner" ] || continue
    [ -d "/proc/$_owner" ] && continue          # owner alive: leave it alone
    systemctl --user stop "$_u" 2>/dev/null
    systemctl --user reset-failed "$_u" 2>/dev/null
    echo "reaped stale unit $_u (owning battery pid $_owner is gone)"
done
for g in "$G229" "$G233" "$G244"; do
    [ -r "$g" ] || { echo "SKIP(env): guard not readable: $g" >&2; exit 2; }
done

# assert <expected-rc> <label> <command...>
assert() {
    local want="$1" label="$2"; shift 2
    "$@" >"$TMP/out" 2>&1; local got=$?
    local n=("PASS(0)" "FAIL(1)" "SKIP(2)")
    if [ "$got" = "$want" ]; then
        PASSES=$((PASSES + 1)); printf '  ok    %-52s %s\n' "$label" "${n[$want]}"
    else
        FAILS=$((FAILS + 1))
        printf '  NOT OK %-51s expected %s, got exit %s\n' "$label" "${n[$want]}" "$got"
        sed 's/^/          | /' "$TMP/out" | head -3
    fi
}

# SELF-VALIDATION OF THE INSTRUMENT (§11.4.107(10)) — runs before any real
# assertion. This battery is itself an analyzer, and an analyzer nobody has
# falsified is exactly the unvalidated gate it exists to prevent. If assert()
# were broken — comparing the wrong variable, swallowing an exit code, treating
# any completion as agreement — every line below would print "ok" while proving
# nothing. So: three expectations that MATCH reality must register ok, and
# three that DIVERGE must register not-ok, INCLUDING a SKIP where FAIL was
# expected, which is the precise fail-open shape (§11.4.69) the whole exercise
# is about. Counters are restored afterwards so the self-test does not pollute
# the real tally.
# It drives the REAL assert(), not a replica. An earlier revision re-implemented
# the comparison in a private _selftest() helper — which validated a copy while
# leaving the actual instrument untested, so an assert() mutated to always-ok
# still printed "instrument self-test: ... ok" and sailed on. That is precisely
# the "every line prints ok while proving nothing" failure this block claims to
# prevent, reproduced inside the prevention itself. The real counters are driven
# and then restored, so the self-test does not pollute the tally.
_sv_out=$(
    PASSES=0; FAILS=0
    assert 0 "selftest agreement PASS" bash -c 'exit 0'   >/dev/null
    assert 1 "selftest agreement FAIL" bash -c 'exit 1'   >/dev/null
    assert 2 "selftest agreement SKIP" bash -c 'exit 2'   >/dev/null
    assert 1 "selftest divergence a"   bash -c 'exit 0'   >/dev/null
    assert 2 "selftest divergence b"   bash -c 'exit 0'   >/dev/null
    assert 1 "selftest divergence c"   bash -c 'exit 2'   >/dev/null   # SKIP-where-FAIL: the fail-open shape
    printf '%s %s' "$PASSES" "$FAILS"
)
read -r _sv_pass _sv_fail <<<"$_sv_out"
if [ "${_sv_pass:-x}" != "3" ] || [ "${_sv_fail:-x}" != "3" ]; then
    echo "SELF-TEST FAILED: assert() — the battery's real instrument — is broken \
(agreements=${_sv_pass:-?}, divergences=${_sv_fail:-?}; expected 3 and 3). \
Refusing to report on the guards with a broken instrument: a harness that cannot \
tell exit codes apart would print 'ok' for everything." >&2
    exit 1
fi
echo "instrument self-test: real assert() scored 3 agreements ok, 3 divergences caught (incl. SKIP-where-FAIL-expected)"

cat > "$TMP/stub.py" <<'PY'
import sys, json
from http.server import BaseHTTPRequestHandler, HTTPServer
SHAPE = sys.argv[1]; PORT = int(sys.argv[2])
B = {
  "health_empty":   {"status": "healthy", "components": []},
  "health_unnamed": {"status": "healthy", "components": [{"status": "healthy"}]},
  "chat_error":     {"error": {"message": "all providers exhausted"}},
  "chat_nochoices": {"model": "qwen", "choices": []},
  "chat_wrong":     {"model": "qwen", "choices": [{"message": {"content": "banana"}}]},
  "chat_nomodel":   {"choices": [{"message": {"content": "4"}}]},
}
class H(BaseHTTPRequestHandler):
    def _s(self):
        b = json.dumps(B[SHAPE]).encode()
        self.send_response(200)          # 200 deliberately: status is NOT the oracle
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b))); self.end_headers(); self.wfile.write(b)
    do_GET = do_POST = lambda s: s._s()
    def log_message(self, *a): pass
HTTPServer(("127.0.0.1", PORT), H).serve_forever()
PY

stub() {  # stub <shape> <port>
    # fd 9 (the flock) is CLOSED in the child (review W3). A backgrounded
    # serve_forever inherits it otherwise, so if this battery is SIGKILLed —
    # EXIT trap never runs — the orphaned stubs keep holding the lock and every
    # later invocation burns the full 600s timeout and then SKIPs. The lock
    # would outlive the run that took it.
    # `exec python3` is load-bearing: it REPLACES the subshell rather than
    # forking under it. Without the exec, `$!` records the SUBSHELL's pid, so
    # cleanup kills the wrapper while python3 survives as an orphan still
    # holding its port — and, before fd 9 was closed here, still holding the
    # flock, which made the NEXT run block the full 600s timeout. Measured: 12
    # orphaned stubs accumulated across two runs before this was corrected.
    { exec 9>&-; exec python3 "$TMP/stub.py" "$1" "$2"; } & STUB_PIDS+=("$!")
    for _ in $(seq 1 50); do curl -s --noproxy '*' -m 1 "http://127.0.0.1:$2" >/dev/null 2>&1 && return 0; sleep 0.2; done
    echo "SKIP(env): stub '$1' never came up on $2 (port taken, or python3 failed \
to bind). Refusing to assert against a subject that was never constructed." >&2
    exit 2
}

# unit <name> <systemd-run args...> — creates a transient unit and REGISTERS it
# for cleanup. A creation failure aborts (review N4): running an assert against a
# unit that was never created surfaces as a confusing NOT-OK about the guard,
# when the real fault is the harness's own setup.
unit() {
    local name="$1"; shift
    if ! systemd-run --user --unit="$name" --quiet "$@"; then
        echo "SKIP(env): could not create transient unit '$name' — harness setup \
failed, so no verdict about the guards is possible." >&2
        exit 2
    fi
    UNITS+=("$name")
}

echo "=== REACHABLE BUT DEFECTIVE — every one served over HTTP 200 (must FAIL) ==="
stub health_empty   "$P_H_EMPTY"; stub health_unnamed "$P_H_UNNAMED"
stub chat_error     "$P_C_ERR";   stub chat_nochoices "$P_C_NOCH"
stub chat_wrong     "$P_C_WRONG"; stub chat_nomodel   "$P_C_NOMODEL"
assert 1 "244: components:[] (the pre-fix defect shape)" env RED_MODE=0 HXC244_URL=http://127.0.0.1:$P_H_EMPTY/h bash "$G244"
assert 1 "244: components present but unnamed"           env RED_MODE=0 HXC244_URL=http://127.0.0.1:$P_H_UNNAMED/h bash "$G244"
assert 1 "233: 200 carrying an error envelope"           env RED_MODE=0 HXC233_URL=http://127.0.0.1:$P_C_ERR/v1 bash "$G233"
assert 1 "233: 200 with an empty choices array"          env RED_MODE=0 HXC233_URL=http://127.0.0.1:$P_C_NOCH/v1 bash "$G233"
assert 1 "233: 200, fluent content, wrong answer"        env RED_MODE=0 HXC233_URL=http://127.0.0.1:$P_C_WRONG/v1 bash "$G233"
assert 1 "233: 200, right answer, no model id"           env RED_MODE=0 HXC233_URL=http://127.0.0.1:$P_C_NOMODEL/v1 bash "$G233"

echo "=== RUNNING BUT DEFECTIVE — live processes (must FAIL) ==="
unit "gfals-$RUN-nogin" bash -c 'echo "Starting fake"; sleep 120'
unit "gfals-$RUN-debug" -p Environment=GIN_MODE=release \
    bash -c 'echo "Starting fake"; echo "[GIN-debug] GET /v1/models --> h"; sleep 120'
unit "gfals-$RUN-vacuous" -p Environment=GIN_MODE=release \
    bash -c 'echo "no startup marker here"; sleep 120'
# The same unprovable window, but WITH the defect present. This is the
# fail-open probe for the PASS-with-caveat path below: if that path ever stops
# scanning for GIN-debug before deciding, this case silently flips to a pass.
unit "gfals-$RUN-vacuous-debug" -p Environment=GIN_MODE=release \
    bash -c 'echo "no startup marker here"; echo "[GIN-debug] GET /v1/models --> h"; sleep 120'
# A process with NO GIN_MODE at all AND an unprovable window: check 1 must FAIL
# first, so the caveat path can never rescue a genuine violation.
unit "gfals-$RUN-vacuous-nogin" bash -c 'echo "no startup marker here"; sleep 120'
sleep 2
assert 1 "229: live process lacks GIN_MODE=release"      env RED_MODE=0 HXC229_UNIT=gfals-$RUN-nogin   bash "$G229"
assert 1 "229: process emits GIN-debug route dumps"      env RED_MODE=0 HXC229_UNIT=gfals-$RUN-debug   bash "$G229"
# RECONCILED, NOT RELAXED (§11.4.120). This asserted SKIP(2) until 2026-08-10,
# when a review showed the SKIP was discarding a check that HAD run and passed:
# check 1 reads GIN_MODE from the live process's /proc and is the primary
# invariant, so "certified nothing" was false, and a long-lived gateway whose
# journal had rotated sat in permanent SKIP. The guard now reports
# PASS-with-caveat, and this assertion tracks the NEW mechanism rather than
# being edited until it agreed. The two probes beneath it are what keep the
# change from being a relaxation: the defect must still be caught on exactly
# this window shape, from both directions.
assert 0 "229: unprovable window -> PASS-with-caveat"    env RED_MODE=0 HXC229_UNIT=gfals-$RUN-vacuous bash "$G229"
assert 1 "229: unprovable window + GIN-debug still FAILs" env RED_MODE=0 HXC229_UNIT=gfals-$RUN-vacuous-debug bash "$G229"
assert 1 "229: unprovable window + no GIN_MODE still FAILs" env RED_MODE=0 HXC229_UNIT=gfals-$RUN-vacuous-nogin bash "$G229"

echo "=== R2 REGRESSION — deployed AND CRASHED must FAIL, never SKIP ==="
unit "gfals-$RUN-crashed" /bin/false
sleep 2
assert 1 "229: unit loaded but ActiveState=failed"       env RED_MODE=0 HXC229_UNIT=gfals-$RUN-crashed bash "$G229"

# R4 — the state the DEPLOYED unit actually presents when it crash-loops.
# The real helixllm-gateway ships Restart=on-failure with RestartUSec=10s,
# StartLimitBurst=5 and StartLimitIntervalUSec=10s (re-measured 2026-08-10 on
# the live unit): at most one start per limit interval, so the burst is
# structurally unexceedable and ActiveState=failed is UNREACHABLE. A gateway
# crashing every 10s therefore sits in activating/auto-restart forever. A
# revision of this guard that FAILed only on `failed` still SKIPped that — a
# continuously-crashing service filed as "stopped by choice", nothing serving,
# and a green sweep.
#
# THE TIMING IS THE TEST (review R4-b, 2026-08-10). This case used RestartSec=1
# while claiming to reproduce production "exactly", and the difference was not
# cosmetic — it changed WHICH detection path ran. The guard has two live
# signals: SubState=auto-restart, and NRestarts RISING across its 2.5s sample.
# Measured at each timing:
#   RestartSec=1   -> restarts inside the sample window, NRestarts RISES
#   RestartSec=10  -> ActiveState=activating SubState=auto-restart, NRestarts=0
# So the fast fixture was carried by the rising-counter path, and the
# auto-restart path — the ONLY one that fires under the production policy — was
# never exercised by it. Deleting the auto-restart signal would have left this
# battery green while every real crash-loop went undetected.
# Both timings are asserted now: the production one because it is what deploys,
# the fast one because the rising-counter path is real code that also needs a
# subject. Neither is a substitute for the other.
echo "=== R4 REGRESSION — crash-LOOPING under the production Restart policy ==="
unit "gfals-$RUN-loop" -p Restart=on-failure -p RestartSec=10 \
    -p StartLimitBurst=5 -p StartLimitIntervalSec=10 /bin/false
unit "gfals-$RUN-loopfast" -p Restart=on-failure -p RestartSec=1 \
    -p StartLimitBurst=5 -p StartLimitIntervalSec=10 /bin/false
sleep 3
_st=$(systemctl --user show "gfals-$RUN-loop" -p ActiveState --value 2>/dev/null)
_ss=$(systemctl --user show "gfals-$RUN-loop" -p SubState --value 2>/dev/null)
_nr=$(systemctl --user show "gfals-$RUN-loop" -p NRestarts --value 2>/dev/null)
echo "    (production timing: ActiveState=$_st SubState=$_ss NRestarts=$_nr — never-'failed', counter static)"
assert 1 "229: crash-loop @ production RestartSec=10 (auto-restart path)" env RED_MODE=0 HXC229_UNIT=gfals-$RUN-loop     bash "$G229"
assert 1 "229: crash-loop @ RestartSec=1 (rising-counter path)"           env RED_MODE=0 HXC229_UNIT=gfals-$RUN-loopfast bash "$G229"

# --- THE SKIP CONTRACT'S NEGATIVE SPACE (independent review, round 4) --------
#
# Both HTTP guards' headers assert a CLOSED set: curl rc 6 and 7 are SKIP,
# every other rc stays FAIL — 28 (timeout), 35/60 (TLS), 52 (empty reply), 56
# (reset). The positive half of that contract was asserted below; the negative
# half never was. A review MEASURED the behaviour as correct today and still
# filed it, correctly: an unasserted invariant is one edit from being untrue,
# and this one fails OPEN. Broadening the SKIP set by a single rc — the exact
# shape of review mutation N1 — converts a live, wedged, or TLS-broken gateway
# into "not deployed here" and the sweep goes green over a dead service. That is
# the §11.4.69 CM-NO-FAIL-OPEN-SKIP hole the guards' own headers argue against.
#
# Each subject below is a REAL listener that accepts the connection and then
# misbehaves, so it is reachable by construction and can never legitimately
# reach the SKIP branch (a closed port is already covered further down).
#
# THE VERDICT ASSERTION CANNOT PIN THE rc  (independent review round 5, F2)
# ------------------------------------------------------------------------
# `assert 1 ...` says the guard FAILed. It does NOT say which rc produced that
# FAIL, because the guards FAIL identically on 28, 35, 52, 56 and 60 — so a
# comment naming the wrong one survives forever with every assertion green.
# It did: the `empty` stub closed WITHOUT reading the request, which discards
# pending data and obliges the kernel to send RST, so it yielded rc 56 while the
# stub comment, the commit message and the evidence all said 52. The declared
# member that was actually being exercised (56) was already covered by the rc35
# case's sibling, and 52 — the one the batch reported as closed — remained
# fail-open: broadening the SKIP set to include ONLY rc 52 left the battery
# fully green, which is the N1 shape on the very rc the fix claimed.
#
# So each subject's rc is now MEASURED and pinned by assert_rc() before its
# verdict is asserted. The label and the wire can no longer drift apart
# silently: if a stub stops producing what it claims, the pin fails by name
# (§11.4.201 — a claim nothing measures is not an assertion).
assert_rc() {  # assert_rc <expected-curl-rc> <label> <url> [timeout]
    local want="$1" label="$2" url="$3" tmo="${4:-5}"
    # Same flags the guards use, so this measures THEIR view of the subject.
    curl -sk --noproxy '*' -m "$tmo" "$url" >/dev/null 2>&1; local got=$?
    if [ "$got" = "$want" ]; then
        PASSES=$((PASSES + 1)); printf '  ok    %-52s curl rc=%s\n' "$label" "$got"
    else
        FAILS=$((FAILS + 1))
        printf '  NOT OK %-51s expected curl rc=%s, got rc=%s\n' "$label" "$want" "$got"
    fi
}
cat > "$TMP/tcpstub.py" <<'PY'
import select, socket, sys, time
SHAPE = sys.argv[1]; PORT = int(sys.argv[2])
s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
s.bind(("127.0.0.1", PORT)); s.listen(8)
while True:
    c, _ = s.accept()
    if SHAPE == "wedge":
        # Accept and never answer: curl gives rc 28 (operation timed out).
        # A closed port refuses INSTANTLY on loopback, so a timeout here proves
        # a listener took the connection and then hung — a live defect.
        time.sleep(3600)
    elif SHAPE == "reset":
        # WAIT for the request to arrive, then close WITHOUT reading it.
        # Discarding data that is already sitting in the receive queue is what
        # obliges the kernel to send RST instead of FIN, so curl reports rc 56
        # (recv failure: connection reset by peer).
        #
        # The select() is load-bearing and was added after a measured flake: a
        # bare close() races the client's send, and when the server wins there
        # is nothing unread, the close is a clean FIN, and curl reports 52 —
        # which is the exact 52/56 confusion this shape exists to distinguish.
        # Observed once in ~40 runs under load before the wait was added.
        select.select([c], [], [], 5.0)
        c.close()
    elif SHAPE == "empty":
        # Drain the request FIRST, then close: nothing is left unread, so the
        # close is a clean FIN and curl reports rc 52 (empty reply from server).
        # The recv is what distinguishes this shape from `reset` above; without
        # it this stub silently produces 56 while claiming 52 (review round 5).
        try:
            c.settimeout(2.0)
            c.recv(65535)
        except OSError:
            pass
        c.close()
PY

tcpstub() {  # tcpstub <shape> <port>
    { exec 9>&-; exec python3 "$TMP/tcpstub.py" "$1" "$2"; } & STUB_PIDS+=("$!")
    # Readiness is a successful CONNECT, not a successful request — these
    # subjects deliberately never complete one. Polling with a normal curl
    # request would time out against `wedge` and be mistaken for "never came up".
    for _ in $(seq 1 50); do
        python3 - "$2" <<'PY' 2>/dev/null && return 0
import socket, sys
s = socket.socket(); s.settimeout(0.5)
sys.exit(0 if s.connect_ex(("127.0.0.1", int(sys.argv[1]))) == 0 else 1)
PY
        sleep 0.2
    done
    echo "SKIP(env): tcp stub '$1' never bound $2. Refusing to assert against a \
subject that was never constructed." >&2
    exit 2
}

echo "=== SKIP-CONTRACT NEGATIVE SPACE — reachable-but-broken must FAIL, never SKIP ==="
tcpstub wedge "$P_WEDGE"; tcpstub empty "$P_EMPTY"; tcpstub reset "$P_RESET"

echo "--- each subject's rc is MEASURED before its verdict is asserted ---"
assert_rc 28 "wedge stub really yields rc28 (timeout)"        "http://127.0.0.1:$P_WEDGE/h" 3
assert_rc 52 "empty stub really yields rc52 (empty reply)"    "http://127.0.0.1:$P_EMPTY/h"
assert_rc 56 "reset stub really yields rc56 (peer reset)"     "http://127.0.0.1:$P_RESET/h"
assert_rc 35 "plaintext listener really yields rc35 (TLS)"    "https://127.0.0.1:$P_H_EMPTY/h"

# rc 28 — accepted then hung. The timeout is short so the assertion is quick;
# the guards' own default (90s) would make this case dominate the battery.
assert 1 "233: rc28 timeout (listener accepted, then wedged)" env RED_MODE=0 HXC233_URL=http://127.0.0.1:$P_WEDGE/v1 HXC233_TIMEOUT=3 bash "$G233"
assert 1 "244: rc28 timeout (listener accepted, then wedged)" env RED_MODE=0 HXC244_URL=http://127.0.0.1:$P_WEDGE/h  HXC244_TIMEOUT=3 bash "$G244"
# rc 52 — request READ, then closed with no reply (clean FIN).
assert 1 "233: rc52 empty reply (read, then zero bytes)"      env RED_MODE=0 HXC233_URL=http://127.0.0.1:$P_EMPTY/v1 HXC233_TIMEOUT=5 bash "$G233"
assert 1 "244: rc52 empty reply (read, then zero bytes)"      env RED_MODE=0 HXC244_URL=http://127.0.0.1:$P_EMPTY/h  HXC244_TIMEOUT=5 bash "$G244"
# rc 56 — closed with the request unread, so the peer resets the connection.
assert 1 "233: rc56 peer reset (closed with request unread)"  env RED_MODE=0 HXC233_URL=http://127.0.0.1:$P_RESET/v1 HXC233_TIMEOUT=5 bash "$G233"
assert 1 "244: rc56 peer reset (closed with request unread)"  env RED_MODE=0 HXC244_URL=http://127.0.0.1:$P_RESET/h  HXC244_TIMEOUT=5 bash "$G244"
# rc 35 — TLS handshake against a plaintext listener: a server IS present and
# misconfigured, which is the misconfigured-gateway shape, not an absent one.
# Note both guards pass -k, so this is a genuine handshake failure and not a
# certificate-trust complaint that -k would have waived.
assert 1 "233: rc35 TLS handshake vs plaintext listener"      env RED_MODE=0 HXC233_URL=https://127.0.0.1:$P_C_ERR/v1 HXC233_TIMEOUT=5 bash "$G233"
assert 1 "244: rc35 TLS handshake vs plaintext listener"      env RED_MODE=0 HXC244_URL=https://127.0.0.1:$P_H_EMPTY/h HXC244_TIMEOUT=5 bash "$G244"

# rc 60 — THE LAST DECLARED MEMBER, AND IT HAS NO NETWORK SUBJECT.
# ---------------------------------------------------------------
# rc 60 is "peer certificate cannot be authenticated". Both guards pass -k,
# which waives peer verification outright, so no TLS listener can make THEM see
# 60. Measured against a self-signed listener: with -k curl returns 0, without
# -k it returns 60 — the guards' own flags put this rc out of reach, which is a
# §11.4.112 structural fact, not a missing fixture. Constructing an elaborate
# TLS subject and asserting FAIL would prove nothing about 60; it would silently
# re-measure some other rc, which is exactly the mislabel this section fixes.
#
# What IS testable — and is what the closed set actually claims — is the
# CLASSIFIER: "SKIP iff rc is 6 or 7, every other rc FAILs". So drive the
# classifier directly with a curl stub that exits a chosen code. Same technique
# the R3 case below already uses for the analyzer.
#
# THE INJECTION IS SELF-VALIDATING, which matters more than the injection.
# Each case is aimed at a subject whose REAL rc yields the OPPOSITE verdict:
#   rc60 is asserted against a CLOSED port  — real curl gives 7 -> SKIP(2),
#        so observing FAIL(1) proves the stub, not curl, was executed;
#   rc6  is asserted against a LIVE stub    — real curl gives 52 -> FAIL(1),
#        so observing SKIP(2) proves the same in the other direction.
# A PATH injection that silently failed to take would flip both to the real
# verdict and both assertions would go NOT OK (§11.4.201).
mkdir -p "$TMP/curlbin"
printf '#!/bin/sh\nexit ${FAKE_CURL_RC:-0}\n' > "$TMP/curlbin/curl"; chmod +x "$TMP/curlbin/curl"
assert 1 "233: rc60 TLS cert -> FAIL (vs a CLOSED port)"      env PATH="$TMP/curlbin:$PATH" FAKE_CURL_RC=60 RED_MODE=0 HXC233_URL=http://127.0.0.1:$P_CLOSED/v1 bash "$G233"
assert 1 "244: rc60 TLS cert -> FAIL (vs a CLOSED port)"      env PATH="$TMP/curlbin:$PATH" FAKE_CURL_RC=60 RED_MODE=0 HXC244_URL=http://127.0.0.1:$P_CLOSED/h  bash "$G244"
assert 2 "233: rc6 unresolvable -> SKIP (vs a LIVE stub)"     env PATH="$TMP/curlbin:$PATH" FAKE_CURL_RC=6  RED_MODE=0 HXC233_URL=http://127.0.0.1:$P_EMPTY/v1  bash "$G233"
assert 2 "244: rc6 unresolvable -> SKIP (vs a LIVE stub)"     env PATH="$TMP/curlbin:$PATH" FAKE_CURL_RC=6  RED_MODE=0 HXC244_URL=http://127.0.0.1:$P_EMPTY/h   bash "$G244"

echo "=== PROVABLY ABSENT (must SKIP — and must not swallow anything above) ==="
assert 2 "229: unit never installed (not-found)"         env RED_MODE=0 HXC229_UNIT=gfals-$RUN-absent-unit bash "$G229"
assert 2 "233: connection refused"                       env RED_MODE=0 HXC233_URL=http://127.0.0.1:$P_CLOSED/v1 HXC233_TIMEOUT=5 bash "$G233"
assert 2 "244: connection refused"                       env RED_MODE=0 HXC244_URL=http://127.0.0.1:$P_CLOSED/h  HXC244_TIMEOUT=5 bash "$G244"
assert 2 "233: absence under RED_MODE=1 too"             env RED_MODE=1 HXC233_URL=http://127.0.0.1:$P_CLOSED/v1 HXC233_TIMEOUT=5 bash "$G233"

echo "=== R3 REGRESSION — a crashed ANALYZER must FAIL, never 'RED confirmed' ==="
mkdir -p "$TMP/badbin"; printf '#!/bin/sh\nexit 9\n' > "$TMP/badbin/python3"; chmod +x "$TMP/badbin/python3"
assert 1 "233: analyzer exits 9 (RED polarity)" env PATH="$TMP/badbin:$PATH" RED_MODE=1 HXC233_URL=http://127.0.0.1:$P_C_NOMODEL/v1 bash "$G233"
assert 1 "244: analyzer exits 9 (RED polarity)" env PATH="$TMP/badbin:$PATH" RED_MODE=1 HXC244_URL=http://127.0.0.1:$P_H_EMPTY/h  bash "$G244"

# The live-stack cases below need the real gateway. Absent it, they are an
# honest §11.4.3 SKIP — never a silent pass, and never a FAIL of this battery.
echo "=== LIVE STACK — golden-good controls + R1 proxy regression ==="
if curl -sk --noproxy '*' -m 5 https://localhost:8443/internal/health >/dev/null 2>&1; then
    assert 0 "244: live gateway (must PASS, never SKIP)"  env RED_MODE=0 bash "$G244"
    assert 0 "233: live gateway (must PASS, never SKIP)"  env RED_MODE=0 bash "$G233"
    # R1: a dead proxy in the environment must NOT convert a live subject to SKIP.
    assert 0 "244: PASSes with a dead https_proxy set"    env RED_MODE=0 https_proxy=http://127.0.0.1:59999 HTTPS_PROXY=http://127.0.0.1:59999 bash "$G244"
    assert 0 "233: PASSes with a dead https_proxy set"    env RED_MODE=0 https_proxy=http://127.0.0.1:59999 HTTPS_PROXY=http://127.0.0.1:59999 bash "$G233"
    # RED on a fixed artifact must not "reproduce" (§11.4.115).
    assert 1 "244: RED_MODE=1 does not reproduce on fixed" env RED_MODE=1 bash "$G244"
    assert 1 "233: RED_MODE=1 does not reproduce on fixed" env RED_MODE=1 bash "$G233"
else
    echo "  SKIP  live gateway not reachable — golden-good + R1 not asserted. SKIP-OK: §11.4.3"
fi

echo
echo "=== falsification battery: $PASSES ok, $FAILS not ok ==="
[ "$FAILS" -eq 0 ] || { echo "BATTERY FAILED — a guard did not behave as its exit contract claims" >&2; exit 1; }
echo "every guard FAILs on a reachable/running defect, SKIPs only on provable absence,"
echo "and no reviewed regression (proxy-SKIP, crashed-unit-SKIP, analyzer-crash-PASS) returned."
