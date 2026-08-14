#!/usr/bin/env bash
# shellcheck shell=bash
#
# scripts/lib/mutation_baseline.sh
# ==============================================================================
# BASELINE-COMPARISON RECORD/VERIFY PRIMITIVES FOR §11.4.84 MUTATION EXPERIMENTS
# (HXC-282)
#
# WHY THIS EXISTS
# ------------------------------------------------------------------------------
# The §11.4.84 pre-commit residue scan (scripts/git_hooks/pre-commit) used to
# rely ENTIRELY on a phrase grep for tell-tale markers a mutation experiment is
# supposed to leave behind -- a confession such as `MUTATED` sitting next to
# `for paired`, or `//` sitting next to `always pass`, … (deliberately NOT
# quoted here as one contiguous phrase: this file is production code, staged
# on every ordinary commit, and reproducing the exact trigger byte-for-byte
# would make THIS FILE'S OWN documentation trip the very sweep it describes --
# see HXC-282 round 6/Blocker 2, and scripts/lib/mutation_baseline_test.sh's
# equivalent fixture-check for the same discipline applied to executable
# code, not just prose). A grep
# for a confession cannot see an experiment whose whole method was to DELETE
# something: a deletion leaves no phrase to find. FORENSIC CASE (HXC-282): a
# mutation that removed a credential-masking rule from a live source file was
# left un-restored on disk; the marker sweep reported clean because nothing was
# ever inserted, only removed; it was one `git add` away from reinstating the
# exact leak that file had just been repaired to prevent.
#
# This library replaces "search the file for a phrase" with "compare the file
# against a recorded known-good reference" for any file an experiment has
# explicitly registered. A byte-for-byte content-hash comparison surfaces an
# unintended difference of ANY shape — added, changed, or removed — because it
# does not care what the difference LOOKS like, only whether it exists.
#
# THE DISCRIMINATOR (why this does not fire on ordinary edits)
# ------------------------------------------------------------------------------
# This is deliberately NOT "diff this staged file against git HEAD" — that
# would fire on every single ordinary commit (every staged file differs from
# HEAD by definition) and would get the check disabled within a day. The
# trigger is narrower and OPT-IN: a file is only ever examined by this
# mechanism if some experiment explicitly called `mutation_baseline_record` on
# it BEFORE mutating, and has not yet explicitly closed that record (via
# `mutation_baseline_verify_restored` on a genuine content match, or
# `mutation_baseline_abandon` with a stated reason). A file with no OPEN
# record is invisible to this mechanism no matter how large its diff is — that
# is what keeps an ordinary edit, and a legitimate further edit made AFTER a
# prior mutation on the same file was properly closed, unblocked.
#
# LIFECYCLE
# ------------------------------------------------------------------------------
#   record()            -> status=open,     baseline_sha256 + preserved copy
#                           captured. Refuses if a record is ALREADY open for
#                           the same path (§11.4.84 serialisation: one
#                           mutation in flight per file at a time).
#   verify_restored()    -> compares CURRENT content's sha256 against the
#                           recorded baseline_sha256.
#                             match    -> status=closed (reason=verified_match)
#                             mismatch -> stays status=open (this IS the
#                                         residue condition — of any shape)
#                           Either outcome is APPENDED to the events log
#                           (docs/qa-style captured evidence, never merely an
#                           in-memory assertion) — §11.4.84's "the
#                           verification is captured, not asserted".
#   abandon(<reason>)    -> status=abandoned, explicit escape hatch. Does NOT
#                           require a hash match. Requires a reason (never
#                           silent, always auditable). This exists so a
#                           genuinely-stale open record can never trap the
#                           fleet forever (see "FAILS SAFE" note in
#                           scripts/git_hooks/pre-commit) without ever letting
#                           an UNEXAMINED divergence through silently: closing
#                           via abandon is a deliberate, logged human/agent
#                           act, not a timeout.
#
# STORAGE
# ------------------------------------------------------------------------------
# Everything lives under `$(git rev-parse --git-dir)/mutation_baselines/` —
# INSIDE .git, never inside the working tree beside the real file. Two
# consequences: (1) records are never accidentally staged/committed (nothing
# under .git/ is trackable), and (2) an agent's own scratch/backup copies of a
# file living NEXT TO it in the working tree (e.g. `foo.sh.orig`) are simply
# ordinary untracked files to this mechanism — they are not consulted and do
# not need to be "cleaned up" for this check to behave correctly.
#
#   <gitdir>/mutation_baselines/
#     records/<sha256(repo-relative path)>/
#       meta            key=value lines: path, baseline_sha256, recorded_at,
#                       recorded_by_pid, status, closed_at, closed_reason.
#       baseline_copy   preserved verbatim copy of the file at record() time
#                       (absent if the file did not exist yet — see ABSENT).
#     events.jsonl      append-only, one JSON object per line, NEVER rewritten
#                       or truncated by this library. The captured-evidence
#                       trail: open / verified_restored (match|mismatch) /
#                       abandoned.
#
# The record id is `sha256(repo-relative-path)`, not the path itself, so
# lookups never need directory creation for slash-containing paths and never
# collide with filesystem path-length/escaping limits.
#
# THE "ABSENT" SENTINEL
# ------------------------------------------------------------------------------
# A mutation experiment may create a brand-new file that did not exist before.
# `record()` on a not-yet-existing path stores baseline_sha256=ABSENT (no
# preserved copy). `verify_restored()` then requires the file to be absent
# again (the "restored" state for a file that never existed) to count as a
# match — deleting the file the experiment created is exactly as valid a
# restoration as reverting an in-place edit.
#
# HONEST BOUNDARY (§11.4.6)
# ------------------------------------------------------------------------------
# This is keyed by REPO-RELATIVE PATH. A rename performed mid-experiment (the
# mutated file staged under a NEW name) is not followed — the OLD path's
# record would stay open with nothing to compare against once nothing is
# staged under that name, and the NEW path would have no record of its own.
# This is a narrow, deliberately-undisguised gap, not a claim of full
# coverage; the forensic case this library exists to close is an in-place
# content mutation, not a rename-to-evade.
#
# Records are never auto-expired by age (no TTL). A stale-looking open record
# could still be genuinely un-restored residue, so silence is never the safe
# choice — see `abandon()` above for the explicit, audited way through one.
#
# QUERIES DO NOT WRITE. `mutation_baseline_is_open` / `_get_hash` /
# `_get_meta_field` never create the storage directory or the events log
# when neither already exists (see `_mb_root_path` vs `_mb_root` below) —
# merely CHECKING whether a file has a baseline is not itself a write. This
# matters in practice: the pre-commit hook calls `mutation_baseline_is_open`
# for every staged file on every commit that reaches its baseline-comparison
# layer, including the overwhelming majority of ordinary commits that never
# register a baseline at all. Without this split, that alone would leave an
# empty `.git/mutation_baselines/{records/,events.jsonl}` behind on every
# single commit — harmless (nothing under `.git/` is trackable) but
# needless, and a query silently writing state is a design smell regardless.
#
# USAGE
# ------------------------------------------------------------------------------
#   source "<repo>/scripts/lib/mutation_baseline.sh"
#   mutation_baseline_record "path/to/file"          # before mutating
#   ...mutate...
#   ...restore...
#   mutation_baseline_verify_restored "path/to/file"  # compares + closes/keeps-open
#   # or, if the divergence is intentional and not residue:
#   mutation_baseline_abandon "path/to/file" "explanation"
#
#   mutation_baseline_is_open "path/to/file"          # 0 = an open record exists
#   mutation_baseline_get_hash "path/to/file"         # prints recorded baseline sha256
#   mutation_baseline_get_meta_field "path/to/file" recorded_at
#
# ENV KNOBS
#   MUTATION_BASELINE_DIR   override the storage root (default: derived from
#                            `git rev-parse --git-dir`). Primarily for tests
#                            that want an isolated store without a full repo.
#
# Dependencies: git, sha256sum (or shasum -a 256 — §11.4.81 macOS/BSD parity),
#   awk, POSIX sh utilities.
# Cross-references: §11.4.84 / §11.4.115 / §11.4.6 / §11.4.180 / §11.4.201 /
#   HXC-282; scripts/git_hooks/pre-commit;
#   scripts/lib/mutation_baseline_test.sh;
#   scripts/git_hooks/test_mutation_baseline_deletion_detection.sh.
# ==============================================================================

# Idempotent source guard: sourcing twice must not redefine state oddly.
if [ -n "${_MUTATION_BASELINE_SOURCED:-}" ]; then
  return 0 2>/dev/null || true
fi
_MUTATION_BASELINE_SOURCED=1

# --- internal: sha256 of a FILE's bytes, or the ABSENT sentinel -------------
_mb_sha256_file() {
  local p="${1:-}"
  if [ -z "$p" ] || [ ! -e "$p" ]; then
    printf '%s' "ABSENT"
    return 0
  fi
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum -- "$p" 2>/dev/null | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 -- "$p" 2>/dev/null | awk '{print $1}'
  else
    return 1
  fi
}

# --- internal: sha256 of a STRING (used to derive the record id) ------------
_mb_sha256_string() {
  local s="${1:-}"
  if command -v sha256sum >/dev/null 2>&1; then
    printf '%s' "$s" | sha256sum 2>/dev/null | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    printf '%s' "$s" | shasum -a 256 2>/dev/null | awk '{print $1}'
  else
    return 1
  fi
}

# --- internal: minimal JSON string escaping (backslash + double-quote) ------
_mb_json_escape() {
  local s="${1:-}"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  printf '"%s"' "$s"
}

# --- internal: resolve $1 to a path relative to the repo toplevel -----------
_mb_repo_relpath() {
  local p="${1:-}" toplevel dir base abspath
  [ -n "$p" ] || return 1
  toplevel=$(git rev-parse --show-toplevel 2>/dev/null) || return 1
  if [ -e "$p" ]; then
    dir=$(dirname -- "$p")
    base=$(basename -- "$p")
    abspath="$(cd "$dir" 2>/dev/null && pwd)/$base" || return 1
  else
    # Not-yet-existing path (ABSENT baseline, or verifying a restored
    # deletion): the file need not exist, but its directory should.
    dir=$(dirname -- "$p")
    base=$(basename -- "$p")
    if abspath="$(cd "$dir" 2>/dev/null && pwd)/$base"; then
      :
    else
      abspath="$p"
    fi
  fi
  case "$abspath" in
    "$toplevel"/*) printf '%s' "${abspath#"$toplevel"/}" ;;
    "$toplevel") printf '%s' "." ;;
    *) printf '%s' "$abspath" ;;   # outside the repo entirely: return as-is
  esac
}

# --- internal: storage root, PURE path computation, NO side effects ---------
# Used by every read-only query (is_open / get_hash / get_meta_field) so
# merely CHECKING whether a file has a baseline never itself creates
# storage state under .git/ -- a query is not a write. (This distinction is
# real, not cosmetic: the pre-commit hook calls mutation_baseline_is_open
# for every staged file on every commit that reaches layer 3b, including
# ordinary commits with zero registered baselines. Without this split, that
# alone would create an empty .git/mutation_baselines/{records/,
# events.jsonl} on every such commit.)
_mb_root_path() {
  local gitdir root
  if [ -n "${MUTATION_BASELINE_DIR:-}" ]; then
    root="$MUTATION_BASELINE_DIR"
  else
    gitdir=$(git rev-parse --git-dir 2>/dev/null) || return 1
    case "$gitdir" in
      /*) : ;;
      *) gitdir="$(pwd)/$gitdir" ;;
    esac
    root="$gitdir/mutation_baselines"
  fi
  printf '%s' "$root"
}

# --- internal: storage root, CREATES it on demand ----------------------------
# Only WRITE paths (record / verify_restored / abandon / append_event) call
# this. It ensures the directory + append-only log exist before writing.
_mb_root() {
  local root
  root=$(_mb_root_path) || return 1
  mkdir -p "$root/records" 2>/dev/null || true
  : >>"$root/events.jsonl" 2>/dev/null || true
  printf '%s' "$root"
}

# _mb_record_dir <relpath> [readonly]
#   Computes the record directory for <relpath>. With the "readonly" mode
#   argument, this NEVER creates storage state (used by is_open / get_hash /
#   get_meta_field); without it, storage is created on demand (used by the
#   write operations).
_mb_record_dir() {
  local relpath="${1:-}" mode="${2:-}" root record_id
  if [ "$mode" = "readonly" ]; then
    root=$(_mb_root_path) || return 1
  else
    root=$(_mb_root) || return 1
  fi
  record_id=$(_mb_sha256_string "$relpath") || return 1
  printf '%s' "$root/records/$record_id"
}

_mb_append_event() {
  local root
  root=$(_mb_root) || return 1
  printf '%s\n' "$1" >>"$root/events.jsonl"
}

# --- internal: key=value meta file get/set (atomic upsert via temp+mv) ------
_mb_meta_get() {
  local f="${1:-}" k="${2:-}"
  [ -f "$f" ] || return 1
  awk -F'=' -v k="$k" '$1==k { sub(/^[^=]*=/, ""); print; exit }' "$f"
}

_mb_meta_set() {
  local f="${1:-}" k="${2:-}" v="${3:-}" tmp
  tmp="${f}.tmp.$$"
  {
    [ -f "$f" ] && awk -F'=' -v k="$k" '$1!=k' "$f"
    printf '%s=%s\n' "$k" "$v"
  } >"$tmp"
  mv -f "$tmp" "$f"
}

# --- internal: the REAL pid of the currently-executing (sub)shell -----------
# `$$` famously does NOT update inside a `( ) &` subshell in bash (a
# historical Bourne-shell-heritage quirk POSIX preserves): it keeps
# reporting the TOP-LEVEL invoking shell's pid, IDENTICAL across every
# concurrent subshell forked from the same parent. Discovered while testing
# an earlier version of this fix (2026-08-12): 20 racers backgrounded as
# `( ... ) &` from the same parent script all shared one $$, so a candidate-
# path-uniqueness / pid-recording scheme keyed on $$ collides across every
# racer. $BASHPID (bash >= 4.0) is the correct, per-process replacement.
#
# ROUND-3 FIRST-PASS DESIGN (2026-08-12, since found CATASTROPHICALLY WRONG
# -- see the tightening below): this function used to `printf` the pid for
# the caller to capture via `mypid=$(_mb_real_pid)`, on the (explicitly
# stated, and WRONG) claim that the pre-4.0 `sh -c 'echo $PPID'` fallback was
# "verified empirically to match $BASHPID in every case tested". Command
# substitution (`$(...)`) ALWAYS forks a subshell to run its command list --
# including a call to a shell FUNCTION, which cannot be exec-replaced the
# way a single external binary can. So `mypid=$(_mb_real_pid)` read
# `$BASHPID` INSIDE that forked, transient subshell: the CORRECT pid of a
# process that was ALREADY EXITING by the time the assignment completed, not
# the pid of the actual caller who would go on to hold the lock.
#
# MEASURED (round-4 review + independently reproduced here, 2026-08-12): a
# lock holder's OWN recorded pid was DEAD the instant it was written -- every
# `kill -0 <recorded pid>` by any waiter therefore reported "dead" for every
# LIVE holder, unconditionally, defeating the dead-holder liveness check
# (and everything layered on top of it: the R1 recheck, the R1 defence-in-
# depth restore -- all three compared the SAME falsely-dead value). Measured
# overlap on the real shipped code, zero synthetic dead-holder setup: a
# holder acquires, a waiter arrives 0.7s later, reaps the LIVE lock, and
# enters -- 2.28s of concurrent occupancy. The round-3 mkdir-fallback test
# coverage never caught this because it manufactured its OWN "dead holder"
# fixture by hand (a genuinely-exited backgrounded pid, written directly to
# the lockfile) rather than letting the code under test record its own pid
# via its own (broken) path for the HOLDER side of the scenario -- so the
# test's fixture and the code's own defect were, by coincidence, the exact
# same shape, and the test could not tell correct behaviour from broken.
#
# FIX: this function no longer prints for the caller to capture. It writes
# the real pid DIRECTLY into a caller-supplied variable name (a portable
# eval-based "return by reference" -- `local -n` namerefs need bash >= 4.3
# and are NOT assumed here, because this whole code path exists for
# §11.4.81 macOS/BSD parity, where the *default* system bash is 3.2). The
# fast path (`$BASHPID` set) is now a bare `eval "$outvar=\$BASHPID"` in the
# CALLER's own process -- no subshell, no fork, nothing to go stale.
#
# The pre-4.0 fallback still needs `sh -c 'echo $PPID'` wrapped in `$(...)`
# (there is no fork-free way to capture an external command's stdout), but
# THIS use is safe: MEASURED (2026-08-12, at top level, inside one nested
# function, and inside two nested functions, each 3-5x) that
# `cap=$(sh -c 'echo $PPID')` correctly reports the CALLING process's real
# pid, not a transient forked child's -- because that substitution's entire
# command list is exactly ONE external command with no redirection attached
# to it, which bash's exec-optimization replaces the forked child's own
# process image with directly (no additional fork), so `sh`'s own $PPID is
# the ORIGINAL caller's pid. This is DIFFERENT from a `$(...)` around a
# shell FUNCTION (which always genuinely forks, as this whole finding
# demonstrates) -- the two shapes are not interchangeable, and this comment
# exists so a future edit does not conflate them again.
#
# SHARP EDGE, MEASURED (2026-08-12): attaching `2>/dev/null` DIRECTLY to the
# `sh -c '...'` command inside the substitution (`$(sh -c '...' 2>/dev/null)`
# -- the ORIGINAL round-3 code's exact form) DEFEATS the exec-optimization
# above: 8/8 repeated runs then mismatched the calling process's real pid by
# +1 or more (a genuine extra fork occurring). Suppressing `sh`'s stderr
# (in case it is missing or errors on some ancient system) MUST instead wrap
# the WHOLE assignment in a brace group -- `{ cap=$(sh -c '...'); } 2>/dev/
# null` -- which was verified (3x) to preserve the optimization and still
# suppress stderr. This is not a stylistic preference; the two forms are
# NOT equivalent, and the wrong one silently reintroduces this entire
# finding on the very system (pre-4.0 bash) this fallback exists to serve.
_mb_real_pid() {
  # $1 = name of the caller's variable to receive the real pid. Never call
  # this via `$(...)` -- see the header above for exactly why that forks and
  # defeats the whole point.
  #
  # F-6 (round-5 review finding): the eval-based return-by-reference below
  # is an injection surface if $1 is not validated first -- e.g.
  # `_mb_real_pid 'zz; touch /tmp/x; y'` executes arbitrary code (MEASURED).
  # Passing this function's OWN internal local variable name (e.g.
  # `_mb_real_pid __mb_real_pid_outvar`) silently swallows the write
  # instead: the eval assigns to the closest `local` with that name, which
  # is this function's own, never propagating to any caller (MEASURED: the
  # caller's own variable is left untouched with no error at all). Both are
  # unreachable today -- the single call site in this file passes a
  # literal, never anything attacker- or caller-controlled -- but the guard
  # is cheap and this function's whole contract (a name-based return by
  # reference) makes it the kind of thing a future caller could get wrong.
  local __mb_real_pid_outvar="$1"
  case "$__mb_real_pid_outvar" in
    '' | *[!A-Za-z0-9_]* | [0-9]*)
      echo "_mb_real_pid: refusing unsafe/empty variable name: '$__mb_real_pid_outvar'" >&2
      return 2
      ;;
  esac
  case "$__mb_real_pid_outvar" in
    __mb_real_pid_outvar | __mb_real_pid_fallback)
      echo "_mb_real_pid: refusing to target this function's own internal local variable name: '$__mb_real_pid_outvar' (the write would be silently swallowed, never reaching the caller)" >&2
      return 2
      ;;
  esac
  if [ -n "${BASHPID:-}" ]; then
    eval "$__mb_real_pid_outvar=\$BASHPID"
  else
    local __mb_real_pid_fallback
    { __mb_real_pid_fallback=$(sh -c 'echo $PPID'); } 2>/dev/null
    eval "$__mb_real_pid_outvar=\$__mb_real_pid_fallback"
  fi
}

# --- internal: portable directory mtime, epoch seconds (GNU + BSD/macOS) ----
_mb_mtime_epoch() {
  local p="${1:-}" v
  [ -e "$p" ] || return 1
  v=$(stat -c %Y "$p" 2>/dev/null) && [ -n "$v" ] && { printf '%s' "$v"; return 0; }
  v=$(stat -f %m "$p" 2>/dev/null) && [ -n "$v" ] && { printf '%s' "$v"; return 0; }
  return 1
}

# --- internal: per-record-id lock (F4, HXC-282 follow-up) -------------------
# WHY THIS EXISTS: concurrent streams are this repo's normal operating state
# (§11.4.103/§11.4.176), and record()'s own check-then-write (read status ==
# open? -> mkdir + write meta) was NOT atomic against a second concurrent
# record() on the SAME path: both could observe "not open" before either
# wrote, and the later `mv` would silently clobber the earlier baseline with
# no error to either caller (measured, 2026-08-12: 20 concurrent rounds on
# one path -> 20/20 rc=0, 40 "open" events logged for 20 record()
# invocations, last mv wins). A clobbered baseline is exactly the class of
# silent, undetected state this library exists to prevent -- an experiment
# whose recorded reference was quietly replaced by a DIFFERENT experiment's
# reference can no longer tell genuine restoration from residue.
#
# _mb_locked <record_id> <function-name> [args...]
#   Runs <function-name> with the remaining args while holding an EXCLUSIVE
#   lock scoped to this specific record_id (never host-global, never a
#   whole-repo lock -- concurrent experiments on DIFFERENT files must not
#   contend with each other). Repo precedent: scripts/lib/port_sweep_lock.sh
#   (§11.4.180 stale-lock handling; flock(2) locks are released by the
#   KERNEL on holder death, so a crashed caller can never wedge a later one
#   waiting on the same record). §11.4.81 cross-platform parity: a bounded,
#   dead-holder-reaping mkdir-mutex fallback on hosts without flock(1)
#   (macOS/BSD) -- see _mb_locked_mkdir_fallback below.
#   Output (stdout/stderr) of <function-name> propagates normally; its exit
#   code becomes _mb_locked's own return code. 30s bound: a mutation-
#   baseline critical section is a handful of filesystem ops (stat/mkdir/cp/
#   mv), never a long-running sweep, so a 30s wait indicates real contention
#   or a wedged holder, not normal work.
#   MUTATION_BASELINE_FORCE_MKDIR_LOCK=1 forces the mkdir-fallback backend
#   even when flock(1) is present -- test-only, so the fallback (otherwise
#   unreachable on any host that ships flock) has real coverage (R3).
_mb_locked() {
  local record_id="${1:-}" root lockfile rc
  shift
  root=$(_mb_root) || return 1
  lockfile="$root/records/${record_id}.lock"
  if [ "${MUTATION_BASELINE_FORCE_MKDIR_LOCK:-0}" != "1" ] && command -v flock >/dev/null 2>&1; then
    ( exec {_mb_lock_fd}>"$lockfile" 2>/dev/null || exit 70
      flock -w 30 "$_mb_lock_fd" || exit 75
      "$@"
    )
    rc=$?
  else
    _mb_locked_mkdir_fallback "${lockfile}.d" "$@"
    rc=$?
  fi
  return "$rc"
}

# _mb_locked_mkdir_fallback <lockdir> <function-name> [args...]
#   §11.4.81 flock-less (macOS/BSD) backend for _mb_locked. Runs
#   <function-name> with the remaining args while holding an exclusive
#   mkdir-based mutex at <lockdir>.
#
# R1 (review finding, blocking): the ORIGINAL version of this fallback did
# "read pid -> kill -0 dead? -> rm -rf lockdir -> retry mkdir" as FOUR
# separate steps. Two waiters could both observe the SAME dead holder before
# either acted; whichever one's `rm -rf` ran SECOND could delete a lock the
# FIRST one had, by then, already freshly re-acquired and legitimately held
# -- a TOCTOU that reopened exactly the silent-clobber class F4 exists to
# close, contradicting this code's own "never remove a lock whose holder is
# alive" invariant. Demonstrated (review, 2026-08-12) with an artificially
# widened window: two waiters occupied the critical section concurrently for
# 1.4s.
#
# ROUND-3 FIRST-PASS FIX (2026-08-12, since DISPROVEN -- see the tightening
# below): reaping was made a single atomic step, `mv <lockdir> <unique-per-
# pid-reap-target>`, reasoning that the kernel's serialisation of concurrent
# renames of the SAME source path would make the winner's removal "provably
# the exact directory it just verified as dead". That reasoning has a hole:
# atomicity of the RENAME says nothing about whether the CONTENT being
# renamed is still what was checked. A waiter that reads $holder_pid, then
# spends any wall-clock time before acting (the loop body itself in
# production; the same _MB_TEST_REAP_DELAY_SECONDS hook described below,
# deliberately widened, in a test), can reach its `mv` AFTER a different,
# faster racer has already reaped that SAME dead holder and legitimately
# RE-ACQUIRED the identical path with a brand-new, live pid. The slower
# racer's `mv` does not care that the directory it is about to remove now
# belongs to someone alive -- it removes whatever currently sits at that
# path. DEMONSTRATED (coverage added this round,
# scripts/lib/mutation_baseline_mkdir_fallback_test.sh PART 3, case R1b):
# with the identical widened window, the SHIPPED "round-3 first-pass" fix
# still let two waiters occupy the critical section concurrently -- the
# fix closed the narrow "two SIMULTANEOUS movers of the same still-dead
# content" case the original finding demonstrated, but left the broader
# "stale liveness snapshot acted on after a delay" case wide open, which is
# the SAME root defect (kill-0-then-act-later, non-atomically) one layer up.
#
# FIX (this tightening): re-verify $lockdir/pid IMMEDIATELY before acting --
# i.e. adjacent to the `mv`, after any delay, never before it -- and refuse
# to act at all unless it STILL equals the pid just proved dead. If a
# different pid (or nothing, mid-flux) is there now, this waiter correctly
# recognises "not my dead holder any more" and backs off to retry from the
# top of the loop instead of blindly removing whatever it finds. As DEFENCE
# IN DEPTH against the residual (now microscopic) gap between that recheck
# read and the `mv` itself, the pid captured INSIDE the moved-away directory
# is checked AGAIN after the move: if it does not match the dead pid either
# (a live holder's directory got renamed into place in that final sliver of
# time), the moved directory is put BACK rather than destroyed.
#
# HONEST BOUNDARY (§11.4.6, corrected 2026-08-12 round-4 review -- F3): this
# narrows the window to two syscalls with no sleep between them (a `cat`
# immediately followed by a `mv`), the same order of magnitude as any
# ordinary non-atomic check-then-act filesystem operation. It is NOT a claim
# that the gap is mathematically zero -- POSIX filesystems offer no atomic
# compare-and-swap on directory content, only atomic rename of a whole path
# -- and an EARLIER version of this comment overstated what closing that gap
# means: it claimed the residual failure mode was merely "extremely rarely
# fails to reap a lock that was in fact still dead a syscall earlier (safe:
# the waiter just loops)". MEASURED (review, 2026-08-12) that this
# understates it. In the syscall-width window this leaves open, THREE things
# can still happen, chained:
#   (1) LIVE-LOCK DISPLACEMENT: a waiter's recheck and defence-in-depth can
#       both pass incorrectly against a genuinely-live holder in a narrow
#       double-coincidence window, momentarily moving that holder's lock
#       away to a reap_target.
#   (2) THIRD-PARTY ADMISSION: while the lock is momentarily absent from
#       <lockdir> because of (1), a completely unrelated waiter can
#       legitimately acquire that now-vacant path via the ordinary
#       `[ ! -e "$lockdir" ]` acquire path above -- measured: C-D overlap of
#       3.01s, both running unmodified shipped code.
#   (3) RESTORE-TIME NESTING CLOBBER: the displacing waiter's defence-in-
#       depth then discovers the mismatch and tries to put the displaced
#       holder's directory BACK -- but by now <lockdir> is occupied by the
#       THIRD party from (2), so the "put back" `mv` hits the SAME ambiguous
#       move-into-an-existing-directory shape documented in the R2 section
#       below: it NESTS the original holder's directory inside the third
#       party's live lockdir rather than replacing it, leaving the original
#       holder's own release (see the ownership check at the end of this
#       function, also added for this finding) unable to find or restore its
#       own state -- it simply, correctly, no longer touches a path it no
#       longer owns, but its own hold is silently invalidated without it
#       ever being told.
# This chain needs a syscall-width double coincidence at step (1) to begin
# at all, so it is microscopic in production, and step (2)+the release-time
# ownership check together guarantee it can never manifest as "a live lock
# gets silently destroyed" -- but it is a real, demonstrated residual, not a
# rounding error, and this comment states it rather than the earlier,
# narrower claim.
#
# R2 (review finding, low): the ORIGINAL version also `mkdir`'d <lockdir>
# directly at its well-known, contended path and wrote the pid file into it
# as a SEPARATE, later step -- so a holder dying in between left a lockdir
# that exists but carries no pid, which the reap logic explicitly (and
# correctly, per R1's own fix) refuses to touch without a provable-dead pid:
# every later acquire attempt would then wait the full 30s and return 75,
# forever, until a human removed it by hand.
#
# ROUND-3 FIRST-PASS FIX (2026-08-12, since found INERT under contention --
# see the tightening below): acquisition was made STAGE-THEN-PUBLISH-THEN-
# VERIFY -- the pid written into a per-PID-unique CANDIDATE directory before
# any attempt to publish it at the contended path, so <lockdir> could no
# longer come into existence without already carrying a pid via this code
# path -- with publication as an UNCONDITIONAL `mv <candidate> <lockdir>` on
# every single loop iteration, reasoning that a portable "fail if
# destination exists" flag (`-T`/`--no-target-directory`) is GNU-only and
# unavailable on the macOS/BSD `mv` this fallback exists for, so ownership
# would instead be VERIFIED by reading back "$lockdir/pid". That reasoning
# is correct about ownership verification but missed a SIDE EFFECT of the
# unconditional `mv`: when <lockdir> already exists -- EMPTY or not, holder
# alive or already dead -- `mv` does not fail, it silently NESTS the
# candidate INSIDE <lockdir> (verified empirically, 2026-08-12: true for
# both an empty and a non-empty destination directory, not merely a
# non-empty one as originally assumed), and that nesting mutates <lockdir>'s
# OWN mtime as a side effect of adding, and moments later removing, a
# directory entry inside it. On EVERY single failed attempt. Forever, for as
# long as anyone keeps polling. DEMONSTRATED (coverage added this round,
# scripts/lib/mutation_baseline_mkdir_fallback_test.sh PART 4): a pid-less
# lockdir's mtime was measured to advance by exactly 1s on every ~1s retry
# iteration -- `age` was 0 on 30+ consecutive attempts against a lockdir
# whose mtime had been deliberately backdated 10s before the loop even
# started -- so the "short, bounded, age-gated reap" described below never
# actually fires while anyone is polling: the polling itself is what starves
# the staleness signal it depends on.
#
# FIX (this tightening): check `[ ! -e "$lockdir" ]` OURSELVES, in the
# shell, BEFORE ever attempting the publish `mv`. When <lockdir> already
# exists, the mv is SKIPPED entirely for this attempt -- we already know it
# cannot succeed, so there is nothing to gain from letting it nest a
# candidate inside <lockdir> and every reason not to (that nesting is
# exactly the mtime-refresh side effect above). The mv is issued only when
# our own existence check says <lockdir> is currently absent, which is the
# one case where `mv` performs a plain, unambiguous rename regardless of
# platform (destination genuinely does not exist -- no `-T` needed). A short,
# bounded, age-gated reap (5s -- far longer than any legitimate stage-then-
# publish sequence, far shorter than the 30s overall wait budget) remains as
# a defensive fallback for a pid-less lockdir from any cause this design did
# not anticipate, so that state is a bounded delay rather than a permanent,
# hand-recovery-only deadlock -- and, with this tightening, an actually
# reachable one under real contention, not merely a theoretical one.
#
# HONEST BOUNDARY (§11.4.6): a race remains between our existence check and
# our `mv` -- if a concurrent acquirer's own publish lands in that narrow
# gap, our `mv` can still nest into it once. That is now a rare, one-off
# occurrence bounded by two adjacent syscalls' worth of time, not a
# guaranteed-every-iteration certainty, so it cannot by itself starve the 5s
# age gate the way the unconditional mv did.
#
# TEST HOOKS (both production no-ops unless set -- TWO INDEPENDENT windows,
# see the F-1 comment at check-1's read site below for why one hook cannot
# stand in for the other):
#   _MB_TEST_REAP_DELAY_SECONDS: sleeps immediately after the dead-holder
#     (or pid-less) liveness/age check, BEFORE check-1 re-reads <lockdir>/
#     pid -- widens the window check-1 itself absorbs by construction.
#   _MB_TEST_RECHECK_TO_MV_DELAY_SECONDS (round 5): sleeps immediately
#     after check-1's own read, BEFORE the atomic reap `mv` -- widens the
#     window check-1 structurally CANNOT see, which only the post-mv
#     defence-in-depth check (check-2) can close.
# Either lets a test deterministically WIDEN a specific race window rather
# than relying on scheduling luck. Both unset (the only values production
# code ever sees) is a no-op.
_mb_locked_mkdir_fallback() {
  local lockdir="${1:-}"
  shift
  # Resolve ONCE per invocation and reuse -- not just for cost, but for
  # SELF-CONSISTENCY: every candidate/reap/pid-file use below must agree on
  # the exact same value for this call, and re-deriving it from $$ or even
  # re-invoking _mb_real_pid at each use site would still be internally
  # consistent, but caching makes that guarantee obvious by construction.
  # F1 (round-4 review finding): called PLAINLY, never via `$(...)` -- see
  # _mb_real_pid's own header for exactly why that distinction is the whole
  # fix. This is the process that will go on to hold the lock; $mypid MUST
  # be ITS real pid, not a transient helper's.
  local mypid
  _mb_real_pid mypid
  local candidate="${lockdir}.candidate.${mypid}"
  local waited=0

  while :; do
    rm -rf "$candidate" 2>/dev/null || true
    if ! mkdir "$candidate" 2>/dev/null; then
      echo "_mb_locked_mkdir_fallback: cannot create $candidate" >&2
      return 70
    fi
    printf '%s' "$mypid" >"$candidate/pid" 2>/dev/null || true

    # R2 tightening: only ATTEMPT the publish when <lockdir> does not
    # currently exist -- see the header comment above for why an
    # unconditional `mv` here (the round-3 first-pass design) silently
    # starved the age-gated pid-less reap below on every failed attempt.
    if [ ! -e "$lockdir" ]; then
      mv "$candidate" "$lockdir" 2>/dev/null || true   # exit status not trusted -- see header
      if [ "$(cat "$lockdir/pid" 2>/dev/null)" = "$mypid" ]; then
        rm -rf "$candidate" 2>/dev/null || true
        break   # verified, genuinely ours
      fi
      # Lost a genuine race to a concurrent acquirer in the narrow window
      # between our existence check and our mv (or our candidate got
      # nested into a lockdir that sprang into existence in that same
      # window) -- clean up our own attempt without ever touching a
      # <lockdir> we do not own.
      local candidate_base
      candidate_base=$(basename -- "$candidate")
      rm -rf "$candidate" 2>/dev/null || true
      [ -n "$candidate_base" ] && rm -rf "${lockdir:?}/$candidate_base" 2>/dev/null || true
    else
      rm -rf "$candidate" 2>/dev/null || true
    fi

    local holder_pid
    holder_pid=$(cat "$lockdir/pid" 2>/dev/null)
    if [ -n "$holder_pid" ] && ! kill -0 "$holder_pid" 2>/dev/null; then
      [ -n "${_MB_TEST_REAP_DELAY_SECONDS:-}" ] && sleep "$_MB_TEST_REAP_DELAY_SECONDS"
      # R1 tightening: re-verify IMMEDIATELY before acting -- never rely on
      # the $holder_pid snapshot taken before the delay above, which may now
      # be stale (see the header comment above for the demonstrated failure
      # mode this closes).
      local recheck_pid
      recheck_pid=$(cat "$lockdir/pid" 2>/dev/null)
      # F-1 (round-5 review finding): a SECOND, INDEPENDENT test-only delay
      # hook, positioned strictly BETWEEN check-1's own read (immediately
      # above) and the `mv` below -- this is a DIFFERENT window than
      # _MB_TEST_REAP_DELAY_SECONDS above (which widens the gap BEFORE
      # check-1's read, a window check-1 absorbs by construction since it
      # re-reads fresh state right before deciding). check-1 CANNOT see
      # anything that changes strictly AFTER this read -- it has already
      # committed to its decision -- so THIS window is exactly what the
      # post-mv defence-in-depth check below exists to close, and a test
      # needs its own knob to widen it independently. Production no-op
      # unless set.
      [ -n "${_MB_TEST_RECHECK_TO_MV_DELAY_SECONDS:-}" ] && sleep "$_MB_TEST_RECHECK_TO_MV_DELAY_SECONDS"
      if [ "$recheck_pid" = "$holder_pid" ]; then
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
      fi
      continue   # retry immediately: we reaped it, someone else did, this
                 # waiter correctly backed off a live re-acquisition, or the
                 # defence-in-depth check put a live lock back
    fi

    if [ -z "$holder_pid" ]; then
      local dir_mtime now age
      dir_mtime=$(_mb_mtime_epoch "$lockdir")
      if [ -n "$dir_mtime" ]; then
        now=$(date -u +%s)
        age=$((now - dir_mtime))
        if [ "$age" -ge 5 ]; then
          [ -n "${_MB_TEST_REAP_DELAY_SECONDS:-}" ] && sleep "$_MB_TEST_REAP_DELAY_SECONDS"
          local reap_target="${lockdir}.reap.${mypid}"
          if mv "$lockdir" "$reap_target" 2>/dev/null; then
            rm -rf "$reap_target" 2>/dev/null || true
          fi
          continue
        fi
      fi
    fi

    if [ "$waited" -ge 30 ]; then
      rm -rf "$candidate" 2>/dev/null || true
      echo "_mb_locked_mkdir_fallback: timed out waiting for $lockdir" >&2
      return 75
    fi
    sleep 1
    waited=$((waited + 1))
  done

  "$@"
  local rc=$?
  # F3 (round-4 review finding, REQUIRED): verify OWNERSHIP before
  # releasing. An unconditional `rm -rf "$lockdir"` here assumed nothing
  # could have changed ownership of <lockdir> between our acquisition and
  # this point -- but the residual gap in the dead-holder reap above (a
  # syscall-width race between the recheck read and the mv, and again
  # between the mv and the defence-in-depth read -- documented there) means
  # a DIFFERENT waiter can, in a rare double coincidence, momentarily
  # displace THIS holder's still-live lock, discover the mismatch via its
  # own defence-in-depth check, and try to put it back -- by which time a
  # THIRD waiter may have legitimately acquired the momentarily-vacant path
  # with its own live pid, so the "put back" nests the displaced holder's
  # directory INSIDE the third waiter's (the same ambiguous-mv-into-an-
  # existing-directory shape documented in the R2 section above). Without
  # this check, THIS holder's own release would then unconditionally
  # destroy the THIRD waiter's live lock -- measured (review, 2026-08-12):
  # a displaced holder's release destroyed its successor's live lock, C-D
  # overlap 3.01s, both running unmodified shipped code. Releasing only
  # when <lockdir> still verifiably belongs to US closes that: if we have
  # been displaced, we simply leave whatever is there now alone.
  if [ "$(cat "$lockdir/pid" 2>/dev/null)" = "$mypid" ]; then
    rm -rf "$lockdir" 2>/dev/null || true
  fi
  return "$rc"
}

# ==============================================================================
# PUBLIC API
# ==============================================================================

# _mb_record_body <relpath> <toplevel>
#   The actual check-then-write logic of mutation_baseline_record(), run
#   ONLY while _mb_locked holds this record's exclusive lock (F4). Not part
#   of the public API.
_mb_record_body() {
  local relpath="${1:-}" toplevel="${2:-}" dir meta hash ts
  dir=$(_mb_record_dir "$relpath") || return 2
  meta="$dir/meta"
  if [ -f "$meta" ]; then
    local status
    status=$(_mb_meta_get "$meta" status)
    if [ "$status" = "open" ]; then
      {
        echo "mutation_baseline_record: REFUSED — an OPEN baseline already exists for '$relpath'"
        echo "  recorded_at=$(_mb_meta_get "$meta" recorded_at)"
        echo "  Close it first: mutation_baseline_verify_restored, or mutation_baseline_abandon"
        echo "  with a reason (§11.4.84 serialisation: one mutation in flight per file)."
      } >&2
      return 1
    fi
  fi
  mkdir -p "$dir" 2>/dev/null || { echo "mutation_baseline_record: cannot create $dir" >&2; return 2; }
  hash=$(_mb_sha256_file "$toplevel/$relpath") || {
    echo "mutation_baseline_record: no sha256 tool available (sha256sum / shasum)" >&2
    return 2
  }
  ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  if [ -e "$toplevel/$relpath" ]; then
    cp -p -- "$toplevel/$relpath" "$dir/baseline_copy" 2>/dev/null \
      || cp -- "$toplevel/$relpath" "$dir/baseline_copy" 2>/dev/null || true
  else
    rm -f "$dir/baseline_copy" 2>/dev/null || true
  fi
  {
    printf 'path=%s\n' "$relpath"
    printf 'baseline_sha256=%s\n' "$hash"
    printf 'recorded_at=%s\n' "$ts"
    printf 'recorded_by_pid=%s\n' "$$"
    printf 'status=%s\n' "open"
  } >"$meta.tmp.$$"
  mv -f "$meta.tmp.$$" "$meta"
  _mb_append_event "{\"event\":\"open\",\"path\":$(_mb_json_escape "$relpath"),\"sha256\":\"$hash\",\"ts\":\"$ts\",\"pid\":$$}"
  echo "mutation_baseline_record: recorded baseline for '$relpath' ($hash)" >&2
  return 0
}

# mutation_baseline_record <path>
#   Records the CURRENT content of <path> (or the ABSENT sentinel if it does
#   not yet exist) as the known-good reference an experiment must restore to.
#   Refuses (rc=1) if a record for this path is ALREADY open — close it
#   first. Refuses (rc=2, F6) if <path> EXISTS but is not a REGULAR file
#   (directory, submodule gitlink, or other special file) — this mechanism
#   tracks file CONTENT; a directory has none to hash, and recording one
#   (e.g. a submodule gitlink, which is a directory on disk) would store an
#   empty hash and then fail-closed-block every subsequent legitimate
#   pointer bump until explicitly abandoned. The whole check-then-write is
#   serialised per-path via _mb_locked (F4) so two concurrent record() calls
#   on the SAME path can never race each other into a silently-clobbered
#   baseline.
mutation_baseline_record() {
  local input="${1:-}" relpath toplevel record_id
  [ -n "$input" ] || { echo "mutation_baseline_record: path required" >&2; return 2; }
  toplevel=$(git rev-parse --show-toplevel 2>/dev/null) || {
    echo "mutation_baseline_record: not inside a git repository" >&2
    return 2
  }
  relpath=$(_mb_repo_relpath "$input") || {
    echo "mutation_baseline_record: cannot resolve path '$input'" >&2
    return 2
  }
  if [ -e "$toplevel/$relpath" ] && [ ! -f "$toplevel/$relpath" ]; then
    echo "mutation_baseline_record: REFUSED — '$relpath' is not a regular file (directory / submodule gitlink / special file). This mechanism tracks file CONTENT; register the specific file(s) inside it instead." >&2
    return 2
  fi
  record_id=$(_mb_sha256_string "$relpath") || {
    echo "mutation_baseline_record: no sha256 tool available (sha256sum / shasum)" >&2
    return 2
  }
  _mb_locked "$record_id" _mb_record_body "$relpath" "$toplevel"
}

# _mb_verify_restored_body <relpath> <toplevel>
#   The actual check-then-write logic of mutation_baseline_verify_restored(),
#   run ONLY while _mb_locked holds this record's exclusive lock (F4 — this
#   serialises against a concurrent record()/abandon() on the SAME path too,
#   not merely against another verify_restored()). Not part of the public API.
_mb_verify_restored_body() {
  local relpath="${1:-}" toplevel="${2:-}" dir meta status rec_hash cur_hash ts
  dir=$(_mb_record_dir "$relpath") || return 2
  meta="$dir/meta"
  if [ ! -f "$meta" ]; then
    echo "mutation_baseline_verify_restored: no baseline record found for '$relpath'" >&2
    return 2
  fi
  status=$(_mb_meta_get "$meta" status)
  if [ "$status" != "open" ]; then
    echo "mutation_baseline_verify_restored: baseline record for '$relpath' is not open (status=${status:-unknown})" >&2
    return 2
  fi
  rec_hash=$(_mb_meta_get "$meta" baseline_sha256)
  cur_hash=$(_mb_sha256_file "$toplevel/$relpath") || {
    echo "mutation_baseline_verify_restored: no sha256 tool available" >&2
    return 2
  }
  ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  if [ "$cur_hash" = "$rec_hash" ]; then
    _mb_meta_set "$meta" status closed
    _mb_meta_set "$meta" closed_at "$ts"
    _mb_meta_set "$meta" closed_reason verified_match
    _mb_append_event "{\"event\":\"verified_restored\",\"path\":$(_mb_json_escape "$relpath"),\"result\":\"match\",\"sha256\":\"$cur_hash\",\"ts\":\"$ts\",\"pid\":$$}"
    echo "mutation_baseline_verify_restored: MATCH — '$relpath' equals its recorded baseline ($cur_hash); record closed." >&2
    return 0
  fi
  _mb_append_event "{\"event\":\"verified_restored\",\"path\":$(_mb_json_escape "$relpath"),\"result\":\"mismatch\",\"baseline_sha256\":\"$rec_hash\",\"current_sha256\":\"$cur_hash\",\"ts\":\"$ts\",\"pid\":$$}"
  {
    echo "mutation_baseline_verify_restored: MISMATCH — '$relpath' does NOT match its recorded baseline."
    echo "  baseline sha256: $rec_hash"
    echo "  current  sha256: $cur_hash"
    echo "  Record stays OPEN (this is a residue condition of unknown shape:"
    echo "  addition, edit, or deletion — comparison does not distinguish)."
  } >&2
  return 1
}

# mutation_baseline_verify_restored <path>
#   Compares CURRENT content's sha256 against the recorded baseline. On a
#   match, closes the record (status=closed) and CAPTURES the verification
#   (never merely asserts it). On a mismatch, the record STAYS OPEN — this is
#   the residue condition, and it is caught by comparison, not by pattern
#   matching, so it does not care whether the mismatch is an insertion, an
#   edit, or a deletion. Serialised per-path via _mb_locked (F4).
#   Returns: 0 = verified match (closed). 1 = mismatch (residue, still open).
#            2 = usage error (no open record found for this path).
mutation_baseline_verify_restored() {
  local input="${1:-}" relpath toplevel record_id
  [ -n "$input" ] || { echo "mutation_baseline_verify_restored: path required" >&2; return 2; }
  toplevel=$(git rev-parse --show-toplevel 2>/dev/null) || {
    echo "mutation_baseline_verify_restored: not inside a git repository" >&2
    return 2
  }
  relpath=$(_mb_repo_relpath "$input") || {
    echo "mutation_baseline_verify_restored: cannot resolve path '$input'" >&2
    return 2
  }
  record_id=$(_mb_sha256_string "$relpath") || {
    echo "mutation_baseline_verify_restored: no sha256 tool available (sha256sum / shasum)" >&2
    return 2
  }
  _mb_locked "$record_id" _mb_verify_restored_body "$relpath" "$toplevel"
}

# _mb_abandon_body <relpath> <reason>
#   The actual check-then-write logic of mutation_baseline_abandon(), run
#   ONLY while _mb_locked holds this record's exclusive lock (F4). Not part
#   of the public API.
_mb_abandon_body() {
  local relpath="${1:-}" reason="${2:-}" dir meta status ts
  dir=$(_mb_record_dir "$relpath") || return 2
  meta="$dir/meta"
  if [ ! -f "$meta" ]; then
    echo "mutation_baseline_abandon: no baseline record found for '$relpath'" >&2
    return 2
  fi
  status=$(_mb_meta_get "$meta" status)
  if [ "$status" != "open" ]; then
    echo "mutation_baseline_abandon: baseline record for '$relpath' is not open (status=${status:-unknown})" >&2
    return 2
  fi
  ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  _mb_meta_set "$meta" status abandoned
  _mb_meta_set "$meta" closed_at "$ts"
  _mb_meta_set "$meta" closed_reason "abandoned"
  _mb_append_event "{\"event\":\"abandoned\",\"path\":$(_mb_json_escape "$relpath"),\"reason\":$(_mb_json_escape "$reason"),\"ts\":\"$ts\",\"pid\":$$}"
  echo "mutation_baseline_abandon: closed (abandoned) the baseline for '$relpath': $reason" >&2
  return 0
}

# mutation_baseline_abandon <path> <reason>
#   Explicit, audited escape hatch: closes an OPEN record WITHOUT requiring a
#   hash match. Requires a non-empty reason (never a silent close). Use this
#   when a staged divergence from a recorded baseline is INTENTIONAL new work,
#   not leftover mutation residue. Serialised per-path via _mb_locked (F4).
mutation_baseline_abandon() {
  local input="${1:-}" reason="${2:-}" relpath record_id
  [ -n "$input" ] || { echo "mutation_baseline_abandon: path required" >&2; return 2; }
  if [ -z "$reason" ]; then
    echo "mutation_baseline_abandon: a reason is required (§11.4.6 no-guessing / auditability)" >&2
    return 2
  fi
  relpath=$(_mb_repo_relpath "$input") || {
    echo "mutation_baseline_abandon: cannot resolve path '$input'" >&2
    return 2
  }
  record_id=$(_mb_sha256_string "$relpath") || {
    echo "mutation_baseline_abandon: no sha256 tool available (sha256sum / shasum)" >&2
    return 2
  }
  _mb_locked "$record_id" _mb_abandon_body "$relpath" "$reason"
}

# mutation_baseline_is_open <path>
#   Shell-success (0) iff an OPEN baseline record exists for <path>.
mutation_baseline_is_open() {
  local relpath dir meta status
  relpath=$(_mb_repo_relpath "${1:-}") || return 1
  dir=$(_mb_record_dir "$relpath" readonly) || return 1
  meta="$dir/meta"
  [ -f "$meta" ] || return 1
  status=$(_mb_meta_get "$meta" status)
  [ "$status" = "open" ]
}

# mutation_baseline_get_hash <path>
#   Prints the recorded baseline sha256 for the OPEN record on <path>.
#   Returns 1 (nothing printed) if there is no open record.
mutation_baseline_get_hash() {
  local relpath dir meta status
  relpath=$(_mb_repo_relpath "${1:-}") || return 1
  dir=$(_mb_record_dir "$relpath" readonly) || return 1
  meta="$dir/meta"
  [ -f "$meta" ] || return 1
  status=$(_mb_meta_get "$meta" status)
  [ "$status" = "open" ] || return 1
  _mb_meta_get "$meta" baseline_sha256
}

# mutation_baseline_get_meta_field <path> <field>
#   Prints an arbitrary meta field (recorded_at, status, closed_reason, …)
#   for whatever record currently exists (open OR closed) for <path>.
mutation_baseline_get_meta_field() {
  local relpath dir meta
  relpath=$(_mb_repo_relpath "${1:-}") || return 1
  dir=$(_mb_record_dir "$relpath" readonly) || return 1
  meta="$dir/meta"
  [ -f "$meta" ] || return 1
  _mb_meta_get "$meta" "${2:-}"
}

# mutation_baseline_preserved_copy_path <path>
#   Prints the filesystem path to the preserved verbatim copy captured at
#   record() time, for whatever record currently exists (open OR closed)
#   for <path>. Returns 1 (nothing printed) if there is no record at all, OR
#   if the record's baseline was the ABSENT sentinel (no preserved copy
#   exists for a file that did not exist when recorded). Read-only: never
#   creates storage state (see _mb_record_dir's "readonly" mode).
#   A caller blocked by the pre-commit hook's baseline-comparison check can
#   use this to inspect (`diff <path> "$(mutation_baseline_preserved_copy_path
#   <path>)"`) exactly what changed before deciding whether to restore or
#   abandon -- the block message does not embed the file's own content, only
#   this path.
mutation_baseline_preserved_copy_path() {
  local relpath dir meta copy
  relpath=$(_mb_repo_relpath "${1:-}") || return 1
  dir=$(_mb_record_dir "$relpath" readonly) || return 1
  meta="$dir/meta"
  [ -f "$meta" ] || return 1
  copy="$dir/baseline_copy"
  [ -f "$copy" ] || return 1
  printf '%s' "$copy"
}
