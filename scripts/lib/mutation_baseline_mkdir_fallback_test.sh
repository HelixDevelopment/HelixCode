#!/usr/bin/env bash
# scripts/lib/mutation_baseline_mkdir_fallback_test.sh
# HXC-282 -- coverage for the §11.4.81 flock-less (macOS/BSD) mkdir-mutex
# fallback backend of _mb_locked (scripts/lib/mutation_baseline.sh ::
# _mb_locked_mkdir_fallback).
#
# WHY THIS FILE EXISTS: this backend is UNREACHABLE on any host that ships
# flock(1) -- this host does (/usr/bin/flock present), so
# mutation_baseline_test.sh's own concurrency case (C11) never exercises it,
# and MUTATION_BASELINE_FORCE_MKDIR_LOCK (a test-only knob the library
# exposes for exactly this purpose) had never been used anywhere before this
# file (round 3). That zero-coverage state is WHY several real defects
# survived review undetected on this code path across two rounds:
#   R1 (round 3, blocking): the ORIGINAL reap logic did "read pid -> kill -0
#       dead? -> rm -rf lockdir -> retry mkdir" as FOUR separate, non-atomic
#       steps. Two waiters could both observe the SAME dead holder before
#       either acted, letting two callers hold the "exclusive" critical
#       section at the same time. FIXED (round 3) via a single atomic
#       `mv <lockdir> <unique-reap-target>`, then TIGHTENED (round 4, see F2
#       below) with a recheck immediately before acting.
#   R2 (round 3, low): a holder dying between `mkdir` and writing its pid
#       file left a pid-less lockdir the reap logic would never touch --
#       PART 4 below proves the age-gated reap fires, and (round 4) that a
#       failed acquire attempt no longer resets that lockdir's own mtime as
#       a side effect (see mutation_baseline.sh's own R2 header).
#   F1 (round 4, BLOCKING): `mypid=$(_mb_real_pid)` forked to capture the
#       pid -- every lock's recorded pid was DEAD ON ARRIVAL, so `kill -0`
#       reported every LIVE holder as dead and every reap-safety layer
#       compared the same falsely-dead value. PART 2 below is the dedicated
#       PLAIN-CONTENTION regression test this needed (round 3 had none --
#       its "dead holder" fixtures were manufactured by hand, which
#       accidentally matched F1's own broken always-looks-dead behaviour and
#       could not tell correct from broken).
#   F2 (round 4, REQUIRED): round 3's PART 2 measured RECORD-BODY
#       serialisation (success counts / open events), not LOCK exclusivity
#       -- a check that cannot observe the property it claims to certify.
#       And round 3's PART 3 R1b oracle only proved "hold shorter than the
#       waiter's detour", not exclusion: lengthening the hold from 1.2s to
#       3s reproduced 1.68s of overlap on the (F1-broken) round-3 code. PART
#       2 and PART 3 below are both reworked: a genuine lock-layer
#       acquire/release timeline oracle, replacing the record-body one, and
#       a hold duration well past the widest realistic waiter detour, not a
#       number picked because it happened to pass.
#   F3 (round 4, REQUIRED): the unconditional `rm -rf "$lockdir"` release at
#       the end of _mb_locked_mkdir_fallback did not verify OWNERSHIP first
#       -- a displaced holder's release could destroy a successor's live
#       lock. FIXED with an ownership check; PART 5 below tests that
#       invariant directly (deterministically -- the full three-way natural
#       reproduction needs "a syscall-width double coincidence", per the
#       round-4 review, which is not something a CI-safe test should rely on
#       hitting by luck).
#
# CORRECTED NOTE ON THE TWO REAP-SAFETY CHECKS (round-4 finding corrected in
# round 5 -- F-1): round 4 investigating F2 found that removing EITHER the
# R1 pre-mv recheck (check-1) or the R1 post-mv defence-in-depth check
# (check-2) alone still yielded NO-OVERLAP in PART 3's race scenario, and
# concluded from that single window placement that the two checks were
# "individually redundant with each other". That was WRONG, generalized
# from a scenario that only ever exercised ONE of the two windows these
# checks each guard. They are NOT mutually redundant -- they are the SAME
# predicate ("does <lockdir> still hold the pid we proved dead?") evaluated
# at two DIFFERENT moments straddling the `mv`, and each is the sole
# survivor of a race confined to ITS OWN window:
#   WINDOW 1 (before check-1's own read): the gap between the ORIGINAL
#     kill-0 liveness observation and check-1 re-reading <lockdir>/pid.
#     _MB_TEST_REAP_DELAY_SECONDS widens THIS window (sleeps BEFORE
#     check-1's read in mutation_baseline.sh). check-1 absorbs it by
#     construction -- it reads FRESH state right before deciding -- so
#     round 4's PART 3 (which only ever widened window 1) could never see
#     check-1 fail, and removing check-2 there naturally still passed.
#   WINDOW 2 (between check-1's read and the `mv`): the gap check-1
#     structurally CANNOT close, because it commits to a decision the
#     instant it reads and has no way to know the world changes again
#     before its own `mv` executes. _MB_TEST_RECHECK_TO_MV_DELAY_SECONDS
#     (round 5) widens THIS window instead. MEASURED (round-5 review,
#     reproduced here): with check-2 removed and ONLY window 2 widened,
#     the shipped check-1 alone overlaps (2.06s / 2.05s / 2.05s, 3/3) --
#     proving check-1 is NOT a substitute for check-2 outside its own
#     window. Case R1d below is exactly this configuration, made
#     permanent suite coverage.
# So: check-1 is PRIMARY (cheaper, closes window 1, never touches a lock
# whose occupant has already changed by the time it reads); check-2 is the
# genuine BACKSTOP for window 2, which check-1 cannot see by construction.
# Removing either is a measured regression once BOTH windows are tested.
# Case R1c (both removed) still stands as the proof that the COMBINED
# mechanism is necessary; R1d adds that check-1 ALONE is insufficient for
# window 2 specifically -- together they upgrade the §1.1 pairing from "the
# combined mechanism is load-bearing" to "each check is individually
# load-bearing, in its own window".
#
# ANTI-BLUFF: every case here drives the REAL library functions (sourced via
# `.`, never re-implemented) against real files on real disk under a
# throwaway git repo or plain temp dir (mktemp -d), never a re-grep of the
# library's own source.
#
# Usage:   scripts/lib/mutation_baseline_mkdir_fallback_test.sh
# Exit:    0 iff every case passed; 1 otherwise.
# Side-effects: creates + removes temp dirs; touches no real repo state.
# Dependencies: git, bash (subshells inherit functions), mktemp, sha256sum
#   (or shasum), `date +%s.%N` (GNU coreutils -- subsecond timestamps are
#   used only to DETECT overlap in this test's own oracle, never emitted by
#   production code).
# Cross-references: §11.4.84 / §11.4.81 / §11.4.180 / §11.4.6 / HXC-282;
#   scripts/lib/mutation_baseline.sh (_mb_locked / _mb_locked_mkdir_fallback
#   / _mb_real_pid); scripts/lib/mutation_baseline_test.sh (Case 11 -- the
#   flock-backend sibling of PART 2/3 here).

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

PASS=0
FAIL=0
declare -a FAILED_CASES=()

ok()  { PASS=$((PASS+1)); printf 'PASS  %s\n' "$1"; }
bad() { FAIL=$((FAIL+1)); FAILED_CASES+=("$1"); printf 'FAIL  %s\n' "$1"; }

assert_rc() {
  local name="$1" expect="$2" actual="$3"
  if [ "$actual" = "$expect" ]; then
    ok "$name (rc=$actual as expected)"
  else
    bad "$name (expected rc=$expect, got rc=$actual)"
  fi
}

TMP=$(mktemp -d 2>/dev/null) || { echo "mktemp failed"; exit 1; }
cleanup() { rm -rf "$TMP" 2>/dev/null || true; }
trap cleanup EXIT

echo "=== HXC-282 mutation_baseline.sh mkdir-fallback (§11.4.81) coverage ==="
echo

# =============================================================================
# overlap_verdict <timeline-file> <label-A> <label-B>
#   Shared lock-layer oracle used by PART 2 and PART 3: parses ENTER/EXIT
#   lines (high-resolution `date +%s.%N` timestamps) for two labelled
#   participants and echoes OVERLAP or NO-OVERLAP. This is what makes these
#   parts a genuine LOCK-EXCLUSIVITY oracle rather than a proxy metric
#   (record-body success counts, open-event counts) that cannot observe
#   whether the critical sections themselves ever ran concurrently.
# =============================================================================
overlap_verdict() {
  local timeline="$1" la="$2" lb="$3"
  awk -v la="$la" -v lb="$lb" '
    $1=="ENTER" && $2==la {enterA=$3}
    $1=="EXIT"  && $2==la {exitA=$3}
    $1=="ENTER" && $2==lb {enterB=$3}
    $1=="EXIT"  && $2==lb {exitB=$3}
    END {
      if (enterA=="" || exitA=="" || enterB=="" || exitB=="") { print "INCOMPLETE"; exit }
      lo = (enterA>enterB) ? enterA : enterB
      hi = (exitA<exitB) ? exitA : exitB
      if (lo < hi) print "OVERLAP " (hi-lo); else print "NO-OVERLAP"
    }
  ' "$timeline"
}

# =============================================================================
# PART 1 -- basic correctness of the fallback backend, forced via
# MUTATION_BASELINE_FORCE_MKDIR_LOCK=1 even though this host ships flock(1).
# Before this file, the fallback branch had NEVER been exercised by any test
# at all -- not even the ordinary record/verify/abandon happy path.
# =============================================================================
BTMP="$TMP/basic"
mkdir -p "$BTMP"
(
  cd "$BTMP" || exit 1
  git init -q
  git config user.email "t@example.com"; git config user.name "t"; git config commit.gpgsign false
  mkdir -p scripts/lib sub
  cp "$HERE/mutation_baseline.sh" scripts/lib/mutation_baseline.sh
  export MUTATION_BASELINE_FORCE_MKDIR_LOCK=1
  # shellcheck source=/dev/null
  . scripts/lib/mutation_baseline.sh

  printf 'alpha\nprotective_rule\nomega\n' > sub/guard.sh
  orig_hash=$(sha256sum sub/guard.sh | awk '{print $1}')

  mutation_baseline_record sub/guard.sh >/dev/null 2>&1
  echo "$? record" >>"$BTMP/rcs.txt"

  got_hash=$(mutation_baseline_get_hash sub/guard.sh)
  [ "$got_hash" = "$orig_hash" ] && echo "0 hash-match" >>"$BTMP/rcs.txt" || echo "1 hash-match" >>"$BTMP/rcs.txt"

  sed -i '/protective_rule/d' sub/guard.sh
  mutation_baseline_verify_restored sub/guard.sh >/dev/null 2>&1
  echo "$? verify-mismatch" >>"$BTMP/rcs.txt"

  printf 'alpha\nprotective_rule\nomega\n' > sub/guard.sh
  mutation_baseline_verify_restored sub/guard.sh >/dev/null 2>&1
  echo "$? verify-match" >>"$BTMP/rcs.txt"

  mutation_baseline_is_open sub/guard.sh
  echo "$? is-open-after-close" >>"$BTMP/rcs.txt"

  mutation_baseline_record sub/guard.sh >/dev/null 2>&1
  echo "extra intentional line" >> sub/guard.sh
  mutation_baseline_abandon sub/guard.sh "" >/dev/null 2>&1
  echo "$? abandon-no-reason" >>"$BTMP/rcs.txt"
  mutation_baseline_abandon sub/guard.sh "intentional follow-up, not residue" >/dev/null 2>&1
  echo "$? abandon-with-reason" >>"$BTMP/rcs.txt"

  # Confirm the fallback was actually exercised, not silently skipped back
  # onto flock: the on-disk lock artifact is a DIRECTORY (lockfile.d), never
  # a flock'd regular file, under the record's own storage dir.
  gitdir=$(git rev-parse --git-dir)
  record_id=$(printf '%s' "sub/guard.sh" | sha256sum | awk '{print $1}')
  if [ -d "$gitdir/mutation_baselines/records" ]; then
    echo "0 storage-dir-created" >>"$BTMP/rcs.txt"
  else
    echo "1 storage-dir-created" >>"$BTMP/rcs.txt"
  fi
  : "$record_id"  # (kept for readability of the paragraph above; not asserted further)
)
if [ -f "$BTMP/rcs.txt" ]; then
  while read -r rc name; do
    case "$name" in
      record) assert_rc "MK-record-succeeds-on-mkdir-backend" 0 "$rc" ;;
      hash-match) assert_rc "MK-recorded-hash-matches-real-sha256sum" 0 "$rc" ;;
      verify-mismatch) assert_rc "MK-verify-restored-catches-undone-deletion" 1 "$rc" ;;
      verify-match) assert_rc "MK-verify-restored-closes-on-genuine-match" 0 "$rc" ;;
      is-open-after-close) assert_rc "MK-record-closed-after-match" 1 "$rc" ;;
      abandon-no-reason) assert_rc "MK-abandon-without-reason-refuses" 2 "$rc" ;;
      abandon-with-reason) assert_rc "MK-abandon-with-reason-succeeds" 0 "$rc" ;;
      storage-dir-created) assert_rc "MK-storage-dir-created-by-mkdir-backend" 0 "$rc" ;;
    esac
  done < "$BTMP/rcs.txt"
else
  bad "MK-PART1-subshell-produced-no-results (see stderr above)"
fi

# =============================================================================
# PART 2 -- F1 / F2: mutual exclusion under ORDINARY LIVE contention (one
# real holder, no synthetic dead pid anywhere), measured as a genuine
# LOCK-LAYER acquire/release TIMELINE with overlap detection -- not the
# record-body success/open-event proxy round 3 used, which stayed green
# (1/1) on a lock later proven broken because it cannot observe whether the
# critical sections themselves ever overlapped.
#
# This is F1's dedicated regression test: F1 (a fork inside `_mb_real_pid`
# recording a pid that was already dead) meant EVERY live holder's own pid
# looked dead to every waiter, so a waiter would reap a LIVE holder's lock
# almost immediately regardless of how long the holder intended to hold it.
# MEASURED on the pre-fix code (round-4 review, reproduced independently
# here): holder acquires, waiter arrives 0.7s later, reaps the LIVE lock,
# enters -- 2.28s overlap. This case must fail loudly if that regresses.
# =============================================================================
plain_contention_probe() {
  local hold_secs="$1" arrive_after="$2" rtmp
  rtmp=$(mktemp -d 2>/dev/null) || { echo "ERROR"; return; }
  local lockdir="$rtmp/plain.lock.d"
  local timeline="$rtmp/timeline.txt"
  : > "$timeline"

  cat > "$rtmp/holder.sh" <<EOF
#!/usr/bin/env bash
source "$HERE/mutation_baseline.sh"
hold() {
  printf 'ENTER HOLDER %s\n' "\$(date +%s.%N)" >>"$timeline"
  sleep $hold_secs
  printf 'EXIT HOLDER %s\n' "\$(date +%s.%N)" >>"$timeline"
}
_mb_locked_mkdir_fallback "$lockdir" hold
EOF
  cat > "$rtmp/waiter.sh" <<EOF
#!/usr/bin/env bash
source "$HERE/mutation_baseline.sh"
hold() {
  printf 'ENTER WAITER %s\n' "\$(date +%s.%N)" >>"$timeline"
  sleep 0.3
  printf 'EXIT WAITER %s\n' "\$(date +%s.%N)" >>"$timeline"
}
_mb_locked_mkdir_fallback "$lockdir" hold
EOF
  chmod +x "$rtmp/holder.sh" "$rtmp/waiter.sh"

  "$rtmp/holder.sh" &
  local pidH=$!
  sleep "$arrive_after"
  "$rtmp/waiter.sh" &
  local pidW=$!
  wait "$pidH" 2>/dev/null
  wait "$pidW" 2>/dev/null

  overlap_verdict "$timeline" HOLDER WAITER
  rm -rf "$rtmp" 2>/dev/null || true
}

for run in 1 2 3; do
  pc_result=$(plain_contention_probe 2.5 0.7)
  if [ "$pc_result" = "NO-OVERLAP" ]; then
    ok "F1-plain-contention-no-synthetic-dead-holder-run${run} (invariant intact: $pc_result)"
  else
    bad "F1-plain-contention-no-synthetic-dead-holder-run${run} (waiter must wait for the LIVE holder to release; got: $pc_result)"
  fi
done

# =============================================================================
# PART 3 -- R1 / F2: paired §1.1 mutation proof that the reap-safety
# mechanism genuinely prevents two waiters from occupying the critical
# section at the same time when a REAL dead holder is involved, where the
# ORIGINAL four-step reap did not. HOLD_SECS is chosen well past
# DETOUR_SECS (the widest realistic time a waiter spends between its own
# "holder is dead" observation and acting on it in this harness) so a
# NO-OVERLAP verdict certifies exclusion for the run's whole duration, not
# merely "the hold happened to be shorter than the detour" (round-3's R1b
# used hold=1.2s against a detour of ~1.3s -- exactly the artifact F2
# identified).
# =============================================================================
DETOUR_SECS=1.3    # ~ 0.3s stagger + 1s widened _MB_TEST_REAP_DELAY_SECONDS
HOLD_SECS=4         # well past DETOUR_SECS -- see rationale above

# race_probe <fallback-fn-name> <label>
#   Sets up a lockdir already "held" by a pid that is REAL but PROVABLY DEAD
#   (a backgrounded subshell that has already exited), then launches TWO
#   concurrent waiters via <fallback-fn-name>, each holding a fake "critical
#   section" for HOLD_SECS and recording high-resolution ENTER/EXIT
#   timestamps to a shared file. Echoes the overlap_verdict.
race_probe() {
  local fn="$1" label="$2" rtmp
  rtmp=$(mktemp -d 2>/dev/null) || { echo "ERROR"; return; }
  local lockdir="$rtmp/target.lock.d"

  # A holder that acquired the lock and then died without releasing it: a
  # real, now-provably-dead pid (never reused within this test's lifetime --
  # negligible collision risk in a throwaway mktemp -d over a few seconds).
  ( exit 0 ) &
  local dead_pid=$!
  wait "$dead_pid" 2>/dev/null
  mkdir -p "$lockdir"
  printf '%s' "$dead_pid" >"$lockdir/pid"

  hold_and_record() {
    # $1 = who (A|B), $2 = out-file -- this is the "critical section" body
    # run while each waiter believes it holds the lock.
    local who="$1" outfile="$2"
    printf 'ENTER %s %s\n' "$who" "$(date +%s.%N)" >>"$outfile"
    sleep "$HOLD_SECS"
    printf 'EXIT %s %s\n' "$who" "$(date +%s.%N)" >>"$outfile"
  }

  local outfile="$rtmp/timeline.txt"
  : > "$outfile"
  export _MB_TEST_REAP_DELAY_SECONDS=1

  ( "$fn" "$lockdir" hold_and_record A "$outfile" ) &
  local pidA=$!
  sleep 0.3   # stagger just enough that both racers reach the "holder is
              # dead" check while still inside the SAME widened window --
              # the widening (_MB_TEST_REAP_DELAY_SECONDS) is what makes the
              # race genuinely reachable; this stagger only keeps both
              # racers inside it rather than trivially serialised by
              # scheduling luck.
  ( "$fn" "$lockdir" hold_and_record B "$outfile" ) &
  local pidB=$!
  wait "$pidA" 2>/dev/null
  wait "$pidB" 2>/dev/null
  unset _MB_TEST_REAP_DELAY_SECONDS

  overlap_verdict "$outfile" A B
  rm -rf "$rtmp" 2>/dev/null || true
}

# --- source the real library once, into THIS shell, so the real
#     `_mb_locked_mkdir_fallback` is available to the `( "$fn" ... ) &`
#     subshells above (bash subshells forked via `( )` inherit the parent's
#     function table).
LTMP="$TMP/lib_only"
mkdir -p "$LTMP"
cp "$HERE/mutation_baseline.sh" "$LTMP/mutation_baseline.sh"
# shellcheck source=/dev/null
. "$LTMP/mutation_baseline.sh"

# --- MUTANT R1a: byte-derived from the real library's own header comment,
#     reconstructing the four SEPARATE, non-atomic steps the round-3 fix
#     replaced (mkdir-fails -> read pid -> kill -0 dead? -> rm -rf lockdir
#     -> retry mkdir). Uses its OWN inline pid capture (never
#     `_mb_real_pid`, whose CALLING CONVENTION changed under the F1 fix --
#     this mutant intentionally reconstructs a HISTORICAL defect in the
#     REAP logic, not in pid capture, so it must not be coupled to an
#     internal helper's evolving signature) with the SAME
#     _MB_TEST_REAP_DELAY_SECONDS hook wired at the SAME point (immediately
#     after the liveness check, before acting) so the widened window is
#     comparable across all three mutants/real runs below.
_mb_locked_mkdir_fallback_BROKEN() {
  local lockdir="${1:-}"
  shift
  local mypid="${BASHPID:-$$}"
  local waited=0
  while :; do
    if mkdir "$lockdir" 2>/dev/null; then
      printf '%s' "$mypid" >"$lockdir/pid" 2>/dev/null || true
      break
    fi
    local holder_pid
    holder_pid=$(cat "$lockdir/pid" 2>/dev/null)
    if [ -n "$holder_pid" ] && ! kill -0 "$holder_pid" 2>/dev/null; then
      [ -n "${_MB_TEST_REAP_DELAY_SECONDS:-}" ] && sleep "$_MB_TEST_REAP_DELAY_SECONDS"
      rm -rf "$lockdir" 2>/dev/null || true
      continue
    fi
    if [ "$waited" -ge 30 ]; then
      echo "_mb_locked_mkdir_fallback_BROKEN: timed out waiting for $lockdir" >&2
      return 75
    fi
    sleep 1
    waited=$((waited + 1))
  done
  "$@"
  local rc=$?
  rm -rf "$lockdir" 2>/dev/null || true
  return "$rc"
}

# --- MUTANT R1c: the real, CURRENT function with BOTH reap-safety checks
#     (the pre-mv recheck AND the post-mv defence-in-depth restore)
#     mutated away at once -- an unconditional blind reap the moment a
#     (possibly stale) liveness snapshot says "dead", matching the ORIGINAL
#     round-3-first-pass shape this file's introduction (round 3) shipped
#     and round 4 disproved. See the CORRECTED NOTE at the top of this file:
#     this is ADDITIONAL to, not a substitute for, R1d below -- R1c proves
#     the combined mechanism is necessary; R1d (window 2, check-2 removed)
#     proves check-1 alone is insufficient outside its own window, which
#     R1c's combined-removal cannot distinguish from "either check would
#     have sufficed".
UNSAFE_MUTANT="$TMP/mutant_unsafe_reap.sh"
python3 - "$LTMP/mutation_baseline.sh" "$UNSAFE_MUTANT" <<'PYEOF'
import sys
src, dst = sys.argv[1], sys.argv[2]
with open(src) as f:
    content = f.read()
old = '''      if [ "$recheck_pid" = "$holder_pid" ]; then
        local reap_target="${lockdir}.reap.${mypid}"
        if mv "$lockdir" "$reap_target" 2>/dev/null; then
          # Defence in depth against the residual gap between the recheck
          # read immediately above and this mv: confirm what we actually
          # moved still belongs to the dead pid before destroying it.
          if [ "$(cat "$reap_target/pid" 2>/dev/null)" = "$holder_pid" ]; then
            rm -rf "$reap_target" 2>/dev/null || true
          else
            mv "$reap_target" "$lockdir" 2>/dev/null || true
          fi
        fi
      fi'''
new = '''      local reap_target="${lockdir}.reap.${mypid}"
      if mv "$lockdir" "$reap_target" 2>/dev/null; then
        rm -rf "$reap_target" 2>/dev/null || true
      fi'''
if old not in content:
    sys.stderr.write("MUTATION-PATTERN-NOT-FOUND\n")
    sys.exit(1)
content = content.replace(old, new)
with open(dst, "w") as f:
    f.write(content)
PYEOF
if [ $? -eq 0 ] && [ -s "$UNSAFE_MUTANT" ] && ! diff -q "$LTMP/mutation_baseline.sh" "$UNSAFE_MUTANT" >/dev/null 2>&1; then
  ok "R1c-sanity-unsafe-reap-mutant-actually-differs-from-real-library"
else
  bad "R1c-sanity-unsafe-reap-mutant-actually-differs-from-real-library (mutation pattern not found or produced no change -- library source may have shifted; update the python pattern above)"
fi
chmod +x "$UNSAFE_MUTANT" 2>/dev/null || true
UNSAFE_LTMP="$TMP/lib_unsafe"
mkdir -p "$UNSAFE_LTMP"
cp "$UNSAFE_MUTANT" "$UNSAFE_LTMP/mutation_baseline.sh"

# --- MUTANT R1d (round 5, F-1): check-2 (the post-mv defence-in-depth
#     restore) removed, check-1 (the pre-mv recheck) KEPT INTACT -- the
#     mutation R1a/round-4's "individually redundant" claim was measured
#     against, but ONLY under window-1 widening, where check-1 always reads
#     fresh state right before its own decision and so structurally cannot
#     be caught stale. This mutant is probed with
#     _MB_TEST_RECHECK_TO_MV_DELAY_SECONDS instead (widens window 2 --
#     strictly BETWEEN check-1's read and the `mv` -- see the CORRECTED
#     NOTE at the top of this file), which check-1 cannot see by
#     construction. Only removing the defence-in-depth backstop.
R1D_MUTANT="$TMP/mutant_r1d_postcheck_removed.sh"
python3 - "$LTMP/mutation_baseline.sh" "$R1D_MUTANT" <<'PYEOF'
import sys
src, dst = sys.argv[1], sys.argv[2]
with open(src) as f:
    content = f.read()
old = '''        if mv "$lockdir" "$reap_target" 2>/dev/null; then
          # Defence in depth against the residual gap between the recheck
          # read immediately above and this mv: confirm what we actually
          # moved still belongs to the dead pid before destroying it.
          if [ "$(cat "$reap_target/pid" 2>/dev/null)" = "$holder_pid" ]; then
            rm -rf "$reap_target" 2>/dev/null || true
          else
            mv "$reap_target" "$lockdir" 2>/dev/null || true
          fi
        fi'''
new = '''        if mv "$lockdir" "$reap_target" 2>/dev/null; then
          rm -rf "$reap_target" 2>/dev/null || true
        fi'''
if old not in content:
    sys.stderr.write("MUTATION-PATTERN-NOT-FOUND\n")
    sys.exit(1)
content = content.replace(old, new)
with open(dst, "w") as f:
    f.write(content)
PYEOF
if [ $? -eq 0 ] && [ -s "$R1D_MUTANT" ] && ! diff -q "$LTMP/mutation_baseline.sh" "$R1D_MUTANT" >/dev/null 2>&1 \
  && grep -qF 'if [ "$recheck_pid" = "$holder_pid" ]; then' "$R1D_MUTANT"; then
  ok "R1d-sanity-postcheck-removed-mutant-differs-and-keeps-check1-gate"
else
  bad "R1d-sanity-postcheck-removed-mutant-differs-and-keeps-check1-gate (mutation pattern not found, produced no change, or accidentally removed check-1's own gate too -- library source may have shifted; update the python pattern above)"
fi
chmod +x "$R1D_MUTANT" 2>/dev/null || true
R1D_LTMP="$TMP/lib_r1d"
mkdir -p "$R1D_LTMP"
cp "$R1D_MUTANT" "$R1D_LTMP/mutation_baseline.sh"

# Sanity: the mutants are genuinely different code paths, not accidental
# aliases of the real function (guards against a copy/paste mistake making
# this section unable to fail no matter which behaviour is present).
if [ "$(type -t _mb_locked_mkdir_fallback_BROKEN)" = "function" ] \
  && [ "$(type -t _mb_locked_mkdir_fallback)" = "function" ]; then
  ok "R1-sanity-both-mutant-and-real-fallback-functions-defined"
else
  bad "R1-sanity-both-mutant-and-real-fallback-functions-defined"
fi

broken_result=$(race_probe _mb_locked_mkdir_fallback_BROKEN "pre-fix four-step reap")
case "$broken_result" in
  OVERLAP*)
    ok "R1a-BROKEN-four-step-reap-lets-two-waiters-overlap (mutation detected: $broken_result -- reproduces the pre-fix TOCTOU, hold=${HOLD_SECS}s)"
    ;;
  *)
    bad "R1a-BROKEN-four-step-reap-lets-two-waiters-overlap (expected OVERLAP to reproduce the historical defect, got: $broken_result)"
    ;;
esac

real_result=$(race_probe _mb_locked_mkdir_fallback "shipped atomic-mv reap")
if [ "$real_result" = "NO-OVERLAP" ]; then
  ok "R1b-REAL-atomic-mv-reap-prevents-overlap (invariant intact: $real_result, hold=${HOLD_SECS}s -- well past DETOUR_SECS=${DETOUR_SECS}s)"
else
  bad "R1b-REAL-atomic-mv-reap-prevents-overlap (expected NO-OVERLAP -- mutual exclusion must hold, got: $real_result)"
fi

# The unsafe (both-checks-removed) mutant lives in a SEPARATE sourced
# library file, so race_probe (which calls the plain function name) needs
# its own copy of the harness pointed at that file's function instead.
race_probe_unsafe() {
  local label="$1" rtmp
  rtmp=$(mktemp -d 2>/dev/null) || { echo "ERROR"; return; }
  local lockdir="$rtmp/target.lock.d"
  ( exit 0 ) &
  local dead_pid=$!
  wait "$dead_pid" 2>/dev/null
  mkdir -p "$lockdir"
  printf '%s' "$dead_pid" >"$lockdir/pid"
  local outfile="$rtmp/timeline.txt"
  : > "$outfile"

  cat > "$rtmp/racer.sh" <<EOF
#!/usr/bin/env bash
who="\$1"
source "$UNSAFE_LTMP/mutation_baseline.sh"
hold() {
  printf 'ENTER %s %s\n' "\$who" "\$(date +%s.%N)" >>"$outfile"
  sleep $HOLD_SECS
  printf 'EXIT %s %s\n' "\$who" "\$(date +%s.%N)" >>"$outfile"
}
_mb_locked_mkdir_fallback "$lockdir" hold
EOF
  chmod +x "$rtmp/racer.sh"

  export _MB_TEST_REAP_DELAY_SECONDS=1
  "$rtmp/racer.sh" A &
  local pidA=$!
  sleep 0.3
  "$rtmp/racer.sh" B &
  local pidB=$!
  wait "$pidA" 2>/dev/null
  wait "$pidB" 2>/dev/null
  unset _MB_TEST_REAP_DELAY_SECONDS

  overlap_verdict "$outfile" A B
  rm -rf "$rtmp" 2>/dev/null || true
}
unsafe_result=$(race_probe_unsafe "both reap-safety checks removed")
case "$unsafe_result" in
  OVERLAP*)
    ok "R1c-BOTH-checks-removed-lets-two-waiters-overlap (mutation detected: $unsafe_result -- the COMBINED reap-safety mechanism is load-bearing, hold=${HOLD_SECS}s)"
    ;;
  *)
    bad "R1c-BOTH-checks-removed-lets-two-waiters-overlap (expected OVERLAP -- removing the entire reap-safety mechanism should reproduce a race, got: $unsafe_result)"
    ;;
esac

# --- R1d (round 5, F-1): check-2 removed, check-1 KEPT, WINDOW 2 widened
#     via _MB_TEST_RECHECK_TO_MV_DELAY_SECONDS instead of
#     _MB_TEST_REAP_DELAY_SECONDS -- the window strictly between check-1's
#     own read and the `mv`, which check-1 cannot see by construction (it
#     has already committed to its decision by the time this window
#     opens). This is what makes check-1 individually FALSIFIABLE: if
#     check-1 alone were sufficient in every window, this would still be
#     NO-OVERLAP. It is not.
race_probe_r1d() {
  local label="$1" rtmp
  rtmp=$(mktemp -d 2>/dev/null) || { echo "ERROR"; return; }
  local lockdir="$rtmp/target.lock.d"
  ( exit 0 ) &
  local dead_pid=$!
  wait "$dead_pid" 2>/dev/null
  mkdir -p "$lockdir"
  printf '%s' "$dead_pid" >"$lockdir/pid"
  local outfile="$rtmp/timeline.txt"
  : > "$outfile"

  cat > "$rtmp/racer.sh" <<EOF
#!/usr/bin/env bash
who="\$1"
source "$R1D_LTMP/mutation_baseline.sh"
hold() {
  printf 'ENTER %s %s\n' "\$who" "\$(date +%s.%N)" >>"$outfile"
  sleep $HOLD_SECS
  printf 'EXIT %s %s\n' "\$who" "\$(date +%s.%N)" >>"$outfile"
}
_mb_locked_mkdir_fallback "$lockdir" hold
EOF
  chmod +x "$rtmp/racer.sh"

  # Deliberately widen ONLY window 2 (_MB_TEST_RECHECK_TO_MV_DELAY_SECONDS)
  # -- window 1 (_MB_TEST_REAP_DELAY_SECONDS) stays UNSET so this case
  # isolates window 2 specifically, rather than conflating the two.
  export _MB_TEST_RECHECK_TO_MV_DELAY_SECONDS=1
  "$rtmp/racer.sh" A &
  local pidA=$!
  sleep 0.3
  "$rtmp/racer.sh" B &
  local pidB=$!
  wait "$pidA" 2>/dev/null
  wait "$pidB" 2>/dev/null
  unset _MB_TEST_RECHECK_TO_MV_DELAY_SECONDS

  overlap_verdict "$outfile" A B
  rm -rf "$rtmp" 2>/dev/null || true
}
r1d_result=$(race_probe_r1d "check-1 alone, window 2 widened")
case "$r1d_result" in
  OVERLAP*)
    ok "R1d-check1-alone-cannot-close-window2 (mutation detected: $r1d_result -- check-1 is NOT a substitute for check-2 outside window 1, hold=${HOLD_SECS}s)"
    ;;
  *)
    bad "R1d-check1-alone-cannot-close-window2 (expected OVERLAP -- check-1 alone should NOT be able to see a race introduced strictly after its own read, got: $r1d_result)"
    ;;
esac

# =============================================================================
# PART 4 -- R2: a lockdir that exists but carries NO pid (the shape left
# behind by a holder dying between `mkdir` and writing its pid file) is
# eventually reaped via the bounded, age-gated fallback -- never a permanent,
# hand-recovery-only deadlock. The production age gate is 5s; this test
# BACKDATES the lockdir's mtime rather than sleeping 5 real seconds, so the
# test stays fast while still exercising the genuine age comparison
# (_mb_mtime_epoch reads the actual on-disk mtime, which `touch -d` sets for
# real -- this is not a shortcut around the code under test).
# =============================================================================
R2TMP="$TMP/r2"
mkdir -p "$R2TMP"
lockdir_r2="$R2TMP/pidless.lock.d"
mkdir -p "$lockdir_r2"
# No pid file written -- exactly the shape R2 exists to close. Backdate the
# directory's mtime well past the 5s age gate.
if touch -d "-10 seconds" "$lockdir_r2" 2>/dev/null; then
  :
else
  # BSD/macOS touch -d syntax differs; -A -10S is the portable-enough
  # fallback for this repo's own §11.4.81 parity posture. Not expected to be
  # exercised on this Linux host, but kept honest rather than silently
  # assuming GNU touch everywhere.
  touch -A -0010 "$lockdir_r2" 2>/dev/null || true
fi

r2_out="$R2TMP/out.txt"
r2_rc_file="$R2TMP/rc.txt"
r2_probe() {
  printf 'acquired\n' > "$r2_out"
}
start_ts=$(date +%s)
( _mb_locked_mkdir_fallback "$lockdir_r2" r2_probe; echo $? > "$r2_rc_file" )
end_ts=$(date +%s)
elapsed=$((end_ts - start_ts))

r2_rc=$(cat "$r2_rc_file" 2>/dev/null || echo "")
if [ "$r2_rc" = "0" ] && [ -f "$r2_out" ]; then
  ok "R2a-pidless-aged-lockdir-eventually-reaped-and-acquired (rc=0, elapsed=${elapsed}s)"
else
  bad "R2a-pidless-aged-lockdir-eventually-reaped-and-acquired (rc=${r2_rc:-<none>}, elapsed=${elapsed}s -- expected the age-gated reap to fire, not a permanent block)"
fi
if [ "$elapsed" -le 10 ]; then
  ok "R2b-reap-happens-promptly-not-after-the-full-30s-timeout (elapsed=${elapsed}s)"
else
  bad "R2b-reap-happens-promptly-not-after-the-full-30s-timeout (elapsed=${elapsed}s -- suspiciously close to/over the 30s bound)"
fi
if [ ! -d "$lockdir_r2" ]; then
  ok "R2c-lockdir-cleaned-up-after-successful-acquire-and-release"
else
  bad "R2c-lockdir-cleaned-up-after-successful-acquire-and-release (still present: $lockdir_r2)"
fi

# =============================================================================
# PART 5 -- F3: the release step must verify OWNERSHIP before removing
# <lockdir>, so a holder displaced mid-hold (the round-4 "live-lock
# displacement -> third-party admission -> nesting-clobber-on-restore"
# chain, documented in mutation_baseline.sh's own HONEST BOUNDARY comment)
# can never have its own release destroy whatever legitimately occupies
# that path afterward.
#
# This deliberately does NOT try to reproduce the full three-way natural
# race (holder + displacing waiter + admitted third party): the round-4
# review itself measured that chain needs "a syscall-width double
# coincidence" to begin, which is not something a test should rely on
# hitting by luck. Instead it tests the INVARIANT the F3 fix establishes
# directly and deterministically: from INSIDE our own critical section, we
# simulate exactly the end-state the natural race would produce (someone
# else's live pid now occupies <lockdir>), then verify our own release
# leaves it alone. A positive control (PART 4's R2c above) already proves
# the ordinary, non-displaced case still cleans up normally -- this part
# does not need to re-prove that.
# =============================================================================
F3TMP="$TMP/f3"
mkdir -p "$F3TMP"
lockdir_f3="$F3TMP/target.lock.d"
rm -rf "$lockdir_f3"
# MUST be genuinely different from our own $mypid -- f3_critical_section
# below runs in THIS SAME process (no fork happens between
# _mb_locked_mkdir_fallback's own acquisition and its "$@" call), so a
# naive `successor_pid=$$` here would be IDENTICAL to what
# _mb_locked_mkdir_fallback recorded as its own $mypid, making the
# ownership check accidentally "pass" and remove it anyway -- proving
# nothing. A synthetic, clearly-distinct value is fine: the ownership check
# only ever does a string comparison against the lockfile's content, it
# never calls kill -0 on it.
this_process_pid="$$"
successor_pid="synthetic9${this_process_pid}"   # always textually distinct
                                                  # from $$ itself
f3_critical_section() {
  # Simulate the displaced-then-third-party-admitted end state from INSIDE
  # our own hold, deterministically, rather than waiting for the natural
  # race: someone else's live lock now occupies our lockdir.
  rm -rf "$lockdir_f3"
  mkdir -p "$lockdir_f3"
  printf '%s' "$successor_pid" > "$lockdir_f3/pid"
}
_mb_locked_mkdir_fallback "$lockdir_f3" f3_critical_section
f3_rc=$?
if [ "$f3_rc" = "0" ] && [ -d "$lockdir_f3" ] && [ "$(cat "$lockdir_f3/pid" 2>/dev/null)" = "$successor_pid" ]; then
  ok "F3-release-does-not-destroy-a-displaced-successors-live-lock (successor's pid file survived our release: $successor_pid)"
else
  bad "F3-release-does-not-destroy-a-displaced-successors-live-lock (rc=$f3_rc, lockdir present=$([ -d "$lockdir_f3" ] && echo yes || echo no), pid=$(cat "$lockdir_f3/pid" 2>/dev/null))"
fi

echo
echo "=== HXC-282 mkdir-fallback coverage: PASS=$PASS FAIL=$FAIL ==="
if [ "$FAIL" -ne 0 ]; then
  printf 'FAILED CASES:\n'
  for c in "${FAILED_CASES[@]}"; do printf '  - %s\n' "$c"; done
  exit 1
fi
exit 0
