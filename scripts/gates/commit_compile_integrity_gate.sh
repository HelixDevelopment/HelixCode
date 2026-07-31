#!/usr/bin/env bash
# scripts/gates/commit_compile_integrity_gate.sh — §11.4.108 SOURCE→ARTIFACT gate.
#
# WHAT THIS CLOSES
# ----------------
# helix_code/Makefile already ships the right *checker*:
#
#   verify-compile        -> go build -tags=nogui ./...
#   verify-compile-tests  -> go test  -tags=nogui -run='^$' -count=1 ./...
#
# `verify-compile-tests` compiles *_test.go (plain `go build` never does), so it
# is capable of catching a test file that references a symbol which does not
# exist. But it is only ever run against the WORKING TREE. That leaves the
# §11.4.108 SOURCE→ARTIFACT gap wide open: an author whose working tree carries
# an uncommitted symbol can commit a test that references it, watch the gate go
# green locally, and push a commit that does not compile for anyone else.
#
# FORENSIC ANCHOR (FACT, 2026-07-28). Commit 3fd55a4d added
# internal/server/llm_generate_native_model_regression_test.go, whose composite
# literal set a field `respModel` on `modelRecordingProvider`. That field lived
# only in the author's working tree; it was not committed until 34e264e1. So
# `internal/server` did not compile at 3fd55a4d, 3c8197cf or 905a0b0a, while the
# author's pasted "ok 0.018s" evidence truthfully described the working tree.
# Three commits shipped broken and every working-tree gate stayed green.
#
# THE FIX: bind the EXISTING canonical checker to each COMMIT rather than to the
# working tree. This gate checks out each commit in a range into a throwaway
# detached worktree and runs the repo's own compile gate there.
#
# HONEST BOUNDARY (§11.4.6). A PASS proves the inner Go module type-checks at
# that commit under -tags=nogui. It does NOT prove the code is correct, that
# tests pass, or that `!nogui`-tagged GUI sources compile — see COVERAGE GAP.
#
# COVERAGE GAP (§11.4.3, stated not hidden). -tags=nogui matches the repo's own
# canonical gate and dodges the go-gl/glfw -> X11/Xlib.h C dependency that a
# headless host cannot satisfy. Consequently the `!nogui` GUI sources of
# applications/{desktop,aurora_os,harmony_os} are NOT compiled by this gate.
# Covering them requires a host with GL/X11 development headers; run this gate
# with --tags='' there to close the gap.
#
# MODES
#   (default)            check the unpushed delta @{u}..HEAD; exit 1 on any
#                        non-compiling commit.
#   --range <A..B>       check an explicit commit range.
#   --last <N>           check the last N commits.
#   --advisory           report but always exit 0.
#   --tags <list>        override build tags (default: nogui; '' = default tags).
#   --self-test          §1.1 paired mutation: assert the gate FAILS on the
#                        known-bad commit and PASSES on the known-good one.
#                        Proves the gate can actually fail. Also runs the
#                        classifier self-validation below.
#   --classifier-self-test
#                        §11.4.107(10) analyzer self-validation only — feeds
#                        golden-good/golden-bad log fixtures to the failure
#                        classifier. Needs no toolchain, runs in milliseconds.
#
# EXIT CODES
#   0  every compile-relevant commit type-checks (or nothing to check).
#   1  a commit genuinely does not compile — a real compiler diagnostic exists.
#   2  usage error.
#   3  INCONCLUSIVE: host/toolchain failure or an unrecognised build failure.
#      Still blocks (non-zero), but callers MUST NOT report it as "a commit does
#      not compile" — nothing was proven about the source.
#
# WHAT A FAILURE VERDICT MEANS (HXC-215). `go test` exiting non-zero is NOT by
# itself evidence that a commit is broken, so the exit is classified:
#   DOES-NOT-COMPILE           a file.go:line:col compiler diagnostic exists ->
#                              the commit really is broken. check_commit returns
#                              1; the script exits 1 and NAMES the commit.
#   INFRA-FAIL                 a known host/toolchain signature (OOM-kill, disk
#                              full, I/O error) and no diagnostic -> the host
#                              failed, not the source. check_commit returns 3;
#                              the script exits 3 and names NOBODY.
#   BUILD-FAILED-UNCLASSIFIED  neither -> inconclusive. check_commit returns 4;
#                              the script exits 3 and names NOBODY.
# Only the first names a commit. Every non-PASS build preserves its full log
# under $CCI_FAIL_LOG_DIR (default $TMPDIR/cci-gate-failures) and prints the
# path; the gate used to delete that log, destroying the only evidence that
# could tell these three apart.
#
# HONEST GAP (§11.4.6 / §11.4.3). The COMPILE-FAIL test is the presence of a
# POSITIONAL diagnostic, so a genuine source defect that emits none — an import
# cycle, an unsatisfied module requirement — classifies UNCLASSIFIED. That still
# BLOCKS (exit 3): the gate under-NAMES rather than mis-names, which is the only
# safe direction here. Widening the COMPILE-FAIL test requires a captured log of
# that shape to calibrate against, not a guess (§11.4.6) — the previous version
# of this gate guessed, which is precisely what HXC-215 is.
set -uo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
# Fail loudly rather than continue somewhere else: every git call below is
# relative to this root, and a silent cd failure would run the gate against
# whatever repo the caller happened to be in — a verdict about the wrong tree
# is the same class of defect as a verdict about the wrong commit (§11.4.201).
cd "$ROOT" || { echo "FATAL: cannot cd to repo root '$ROOT'" >&2; exit 2; }

# §1.1 self-test pins. Immutable history; do not repoint without re-verifying.
KNOWN_BAD="3fd55a4d"   # references respModel before it was committed
KNOWN_GOOD="34e264e1"  # the commit that landed respModel

RANGE=""; LAST=""; ADVISORY=0; SELFTEST=0; CLASSIFIER_ONLY=0; TAGS="nogui"
while [ $# -gt 0 ]; do
  case "$1" in
    --range)     RANGE="${2:-}"; shift 2 ;;
    --last)      LAST="${2:-}"; shift 2 ;;
    --tags)      TAGS="${2:-}"; shift 2 ;;
    --advisory)  ADVISORY=1; shift ;;
    --self-test) SELFTEST=1; shift ;;
    --classifier-self-test) CLASSIFIER_ONLY=1; shift ;;
    # Print the whole header, bounded by the first line of code rather than a
    # magic line number — a hardcoded count silently truncates the help text the
    # moment the header grows, which is how the EXIT CODES block came to be cut
    # off mid-sentence while claiming to document exactly those codes.
    -h|--help)   awk '/^set -uo pipefail$/{exit} {print}' "${BASH_SOURCE[0]}"; exit 0 ;;
    *)           echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

# The classifier needs no toolchain, so its self-test must stay runnable (and
# therefore falsifiable) on a host without go — otherwise the analyzer's own
# §1.1 pairing would silently SKIP into a green result.
if [ "$CLASSIFIER_ONLY" != 1 ]; then
  command -v go >/dev/null 2>&1 || { echo "SKIP §11.4.3: go toolchain not on PATH"; exit 0; }
fi

WORKDIR="$(mktemp -d "${TMPDIR:-/tmp}/cci-gate.XXXXXX")"
cleanup() {
  # Never leave worktrees registered against the live repo.
  local d
  for d in "$WORKDIR"/*; do
    [ -d "$d" ] || continue
    git -C "$ROOT" worktree remove --force "$d" >/dev/null 2>&1
  done
  rm -rf "$WORKDIR"
  git -C "$ROOT" worktree prune >/dev/null 2>&1
}
# HXC-215 (second defect, found while reproducing the first). `trap cleanup EXIT
# INT TERM` was wrong: bash runs the handler for a trapped signal and then
# RESUMES the script.
#
# REPRODUCED (FACT, 2026-07-31). A minimal script with `trap cleanup EXIT INT
# TERM` that SIGTERMs itself printed: "CLEANUP RAN" / "PHASE-2-STILL-RUNNING" /
# "CLEANUP RAN" and **exited 0**. Three consequences, all bad, and the third is
# the worst: (a) cleanup ran mid-flight, `rm -rf`-ing WORKDIR out from under the
# still-running loop; (b) cleanup then ran a second time at exit; (c) a gate
# killed by a signal reported SUCCESS. (c) is a PASS-bluff (§11.4 / §11.4.1) —
# strictly worse than the false accusation this ticket started from, because a
# false PASS is silent. The same script with the handlers below exits 143 and
# runs cleanup exactly once.
#
# Interrupting the gate must STOP it, not let it invent verdicts or claim one it
# never reached. cleanup stays on EXIT so it still runs exactly once on the way
# out; INT/TERM now terminate with the conventional 128+signo.
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# --------------------------------------------------------------- classification
# FORENSIC ANCHOR (FACT, HXC-215, 2026-07-31). This gate twice reported
# DOES-NOT-COMPILE naming a DIFFERENT innocent commit each time (8490a5d9 in
# /tmp/g19-cci.out, 92840ade in /tmp/g19b.out); both commits compile cleanly on
# re-check, and the packages blamed were unrelated to anything either commit
# touched. Both reports carried `FAIL <pkg> [build failed]` with ZERO
# `file.go:line:col:` diagnostics — a build that produced no compiler
# diagnostic did not demonstrate a source defect.
#
# The two reports CONTRADICT EACH OTHER, which is what makes the falseness a
# FACT rather than an opinion: /tmp/g19b.out blames 92840ade while listing
# `8490a5d9  COMPILES` — the very commit /tmp/g19-cci.out had blamed one run
# earlier, same tags, unchanged history. At least one of the two verdicts is
# provably wrong, and re-checking both commits shows both compile.
#
# One accusation is CAUSALLY IMPOSSIBLE, which settles it without any compile:
# 92840ade touches ONLY helix_code/tests/e2e/mocks/** , yet was blamed for
# internal/rules and internal/telemetry/i18n. No package under internal/ imports
# the e2e mocks (verified: the sole grep hit is a comment), so that commit could
# not have broken those packages by any mechanism.
#
# ROOT CAUSE (FACT): the gate asserted the WRONG CONDITION. It read `go test`
# exiting non-zero as "this commit does not compile" — literally
# `if [ $rc -eq 0 ]; then COMPILES; else DOES-NOT-COMPILE; fi`. But `go test
# ./...` also exits non-zero when the toolchain itself dies, printing
# `[build failed]` for whichever packages happened to be in flight — which is
# why the accused package set differed between runs and matched nothing either
# commit touched. §11.4.201: a gate MUST assert the real condition; a
# false-positive refusal is a FAIL-bluff that teaches everyone to dismiss the
# gate.
#
# WHICH TRANSIENT (FACT, 2026-07-31). Running the OLD logic 8 times over a PINNED
# range on an unchanged tree, keeping every build log, produced 4 PASS / 4 FAIL
# and accused two FURTHER innocent commits (11861996, d99ce58c) — four distinct
# innocent commits across all runs. The kept logs name the two real causes, and
# neither is a source defect:
#
#   (1) fork/exec .../compile: resource temporarily unavailable
#       EAGAIN on fork(2) — the kernel refused to create the process, so the Go
#       driver could not spawn its compile/link/vet workers. No commit was ever
#       read. The §12.12 process-ceiling class; ~7 agents were live on this
#       64-core host. Caught by the "resource temporarily unavailable" signature.
#       WHICH ceiling (measured 2026-07-31, stated with its confidence): the
#       binding one is the cgroup pids controller on the per-project session
#       slice — .../tmxw-helix\x2dcode\x2d6339.slice/pids.max = 4096 — NOT the
#       per-user RLIMIT_NPROC, which is 262144 here, nor the system-wide
#       threads-max (2056078). `go test ./...` on a 64-CPU host fans out to many
#       concurrent compile/link/vet processes, so a 4096-PID slice shared by
#       several agents is reachable. Honest boundary (§11.4.6): the EAGAIN is
#       FACT from the log and the 4096 ceiling is FACT from /sys/fs/cgroup, but
#       pids.current was NOT sampled at the instant of failure — that the 4096
#       slice limit is the one that bound is a well-supported INFERENCE from the
#       other ceilings being three to four orders of magnitude larger.
#
#   (2) chdir .../cci-gate.XXXXXX/<sha>/helix_code: no such file or directory
#       The gate's OWN worktree vanished mid-build — the trap defect above,
#       observed in the wild: a signal ran cleanup, which rm -rf'd WORKDIR out
#       from under the still-running `go test`, and every remaining package then
#       failed on a directory that no longer existed. The two defects HXC-215
#       fixes are therefore not independent: the trap bug manufactured the
#       evidence the classification bug then misread as a broken commit.
#
# Note the kernel log holds no OOM-kill record — the OOM/ENOSPC/EIO entries below
# remain a calibrated starting set rather than a diagnosis of these runs; (1) and
# (2) are the ones actually observed.
#
# The fix is a classifier, NOT a retry and NOT a downgrade-to-warning. It is
# deliberately FAIL-SAFE: only a POSITIVELY IDENTIFIED transient signature is
# exonerated, and anything unrecognised still blocks (exit non-zero) — it is
# merely reported as UNCLASSIFIED rather than blamed on the commit. A genuinely
# non-compiling commit is still caught, because a real compile error emits a
# diagnostic and diagnostics always classify as COMPILE-FAIL.
# Kept deliberately NARROW. Every entry is a message the OS or the Go toolchain
# emits when the machine failed, none is a phrase a compiler emits about source.
# A bare "Killed" was considered and REJECTED as too broad — it could appear in
# arbitrary captured output and would then excuse a real defect, the one
# direction of error this gate must never make.
# The last entry is anchored to the gate's OWN scratch-dir name (cci-gate.XXXXXX)
# so it can only ever match the gate's workdir disappearing underneath a build —
# cause (2) above. A bare "no such file or directory" was REJECTED as far too
# broad: a commit that references a file it forgot to commit produces one too,
# and excusing that would be the one direction of error this gate must never make.
INFRA_SIGNATURES='signal: killed|signal: terminated|signal: bus error|no space left on device|cannot allocate memory|out of memory|too many open files|input/output error|resource temporarily unavailable|bad file descriptor|stale file handle|broken pipe|fork/exec .*: cannot allocate|text file busy|chdir [^:]*cci-gate\.[^:]*: no such file or directory'

# Preserve the evidence. The gate used to `rm -rf "$WORKDIR"` at exit, which
# destroyed the per-commit log — the ONE artefact that distinguishes a compile
# error from a killed toolchain. Every non-PASS build now leaves its full log
# behind and the path is printed.
FAILDIR="${CCI_FAIL_LOG_DIR:-${TMPDIR:-/tmp}/cci-gate-failures}"

# preserve_log <short-sha> <logfile> -> echoes the preserved path, or a marker.
# EVERY non-PASS path must route through this before printing a log location.
# WORKDIR is rm -rf'd by cleanup at exit, so a path under it is already dangling
# by the time a human reads the output — an evidence pointer that resolves to
# nothing is not evidence (§11.4.5).
preserve_log() {
  local short="$1" log="$2" kept
  mkdir -p "$FAILDIR" 2>/dev/null
  kept="$FAILDIR/${short}-$(date +%Y%m%d-%H%M%S-%N).log"
  if cp "$log" "$kept" 2>/dev/null; then
    printf '%s\n' "$kept"
  else
    printf '%s\n' "(could not preserve log)"
  fi
}

# classify_build_failure <logfile> -> COMPILE-FAIL | INFRA | UNCLASSIFIED
# A Go compile/type/vet diagnostic (`<path>.go:<line>:<col>: `) is the only
# positive proof the SOURCE is broken; it is checked FIRST so a build that
# emitted a real diagnostic can never be excused as infrastructure noise even
# if the log also contains a transient signature.
classify_build_failure() {
  local log="$1"
  [ -f "$log" ] || { echo UNCLASSIFIED; return; }
  if grep -qE '[^[:space:]]+\.go:[0-9]+:[0-9]+: ' "$log"; then echo COMPILE-FAIL; return; fi
  if grep -qE "$INFRA_SIGNATURES" "$log"; then echo INFRA; return; fi
  echo UNCLASSIFIED
}

# §11.4.107(10) analyzer self-validation. The classifier is now the thing that
# decides whether a commit gets blamed, so the classifier itself MUST be proven
# unable to bluff: golden-good fixtures must classify one way, golden-bad the
# other. Runs in milliseconds and needs no toolchain, so it is wired into
# --self-test unconditionally.
classifier_self_test() {
  local tmp rc=0 got
  tmp="$(mktemp -d "${TMPDIR:-/tmp}/cci-classify.XXXXXX")"

  # golden-bad for the "excuse it" path: a REAL compile error must never be
  # excused, not even when the log also carries a transient host signature.
  printf '# dev.helix.code/internal/server\ninternal/server/x_test.go:31:4: unknown field respModel in struct literal\nFAIL\tdev.helix.code/internal/server [build failed]\n' > "$tmp/diag.log"
  printf '# dev.helix.code/internal/project\n/usr/lib/golang/pkg/tool/linux_amd64/compile: signal: killed\nFAIL\tdev.helix.code/internal/project [build failed]\n' > "$tmp/killed.log"
  printf '# dev.helix.code/internal/rules\nwrite /tmp/go-build/b001/_pkg_.a: no space left on device\nFAIL\tdev.helix.code/internal/rules [build failed]\n' > "$tmp/enospc.log"
  printf 'FAIL\tdev.helix.code/internal/telemetry/i18n [build failed]\n' > "$tmp/bare.log"
  printf '# dev.helix.code/internal/server\ninternal/server/x_test.go:9:2: undefined: Foo\n/usr/lib/golang/pkg/tool/linux_amd64/link: signal: killed\nFAIL\tdev.helix.code/internal/server [build failed]\n' > "$tmp/both.log"
  # golden-bad for the OPPOSITE direction — the classifier must not manufacture
  # an accusation either. A stack trace names .go files with a LINE but no
  # COLUMN; only a `file.go:line:col: ` diagnostic is a compiler complaint about
  # source. Mistaking `proc.go:267 +0x1e` for one would resurrect HXC-215 with
  # extra steps: an innocent commit blamed on the strength of a crash trace.
  printf 'panic: runtime error: invalid memory address\n\ngoroutine 1 [running]:\ndev.helix.code/internal/server.run(0x0)\n\t/home/u/helix_code/internal/server/run.go:88 +0x24\nruntime.goexit()\n\t/usr/lib/golang/src/runtime/asm_amd64.s:1700 +0x1\nFAIL\tdev.helix.code/internal/server [build failed]\n' > "$tmp/stack.log"
  # The two transients ACTUALLY observed in the HXC-215 flake, verbatim from the
  # preserved logs of the 8-run reproduction (runs 6 and 8).
  printf 'dev.helix.code/pkg/i18n: /usr/lib/golang/pkg/tool/linux_amd64/compile: fork/exec /usr/lib/golang/pkg/tool/linux_amd64/compile: resource temporarily unavailable\nFAIL\tdev.helix.code/internal/project [build failed]\n' > "$tmp/eagain.log"
  printf 'dev.helix.code/internal/rules: /usr/lib/golang/pkg/tool/linux_amd64/compile: chdir /tmp/.private/milos/cci-gate.7GVMhX/d99ce58c/helix_code: no such file or directory\nFAIL\tdev.helix.code/internal/rules [build failed]\n' > "$tmp/vanished.log"
  # golden-bad guarding the narrowness of that last signature: a MISSING SOURCE
  # FILE is a real commit defect and must NOT be excused just because the words
  # "no such file or directory" appear.
  printf '# dev.helix.code/internal/foo\ninternal/foo/bar.go:3:8: open internal/foo/missing.go: no such file or directory\nFAIL\tdev.helix.code/internal/foo [build failed]\n' > "$tmp/missingsrc.log"

  _expect() { # <file> <expected>
    got="$(classify_build_failure "$tmp/$1")"
    if [ "$got" = "$2" ]; then
      echo "  OK   $1 -> $got"
    else
      echo "  BAD  $1 -> $got (expected $2)"; rc=1
    fi
  }
  echo "classifier self-validation (golden-good / golden-bad fixtures):"
  _expect diag.log   COMPILE-FAIL   # real diagnostic => blame the commit
  _expect killed.log INFRA          # OOM-killed toolchain => do NOT blame it
  _expect enospc.log INFRA          # disk full => do NOT blame it
  _expect bare.log   UNCLASSIFIED   # the observed HXC-215 shape => blame nobody
  _expect both.log   COMPILE-FAIL   # diagnostic WINS over a host signature
  _expect stack.log  UNCLASSIFIED   # a stack trace is NOT a compiler diagnostic
  _expect eagain.log    INFRA        # real HXC-215 cause (1): RLIMIT_NPROC exhaustion
  _expect vanished.log  INFRA        # real HXC-215 cause (2): gate workdir deleted mid-build
  _expect missingsrc.log COMPILE-FAIL # a genuinely missing source file IS the commit's fault
  rm -rf "$tmp"
  return $rc
}

if [ "$CLASSIFIER_ONLY" = 1 ]; then
  if classifier_self_test; then
    echo "CLASSIFIER SELF-TEST PASS: every golden fixture classified correctly"; exit 0
  fi
  echo "CLASSIFIER SELF-TEST FAIL"; exit 1
fi

# Compile-check ONE commit in an isolated detached worktree.
# Returns 0 = compiles, 1 = does not compile, 3 = infrastructure failure,
#         4 = build failed for an unrecognised reason (blocks, blames nobody).
check_commit() {
  local sha="$1" short dir log rc start end verdict kept
  short="$(git rev-parse --short=8 "$sha")"
  dir="$WORKDIR/$short"
  log="$WORKDIR/$short.log"

  if ! git -C "$ROOT" worktree add --detach "$dir" "$sha" >"$log" 2>&1; then
    # Setting up the checkout failed, so the compiler never ran and the SOURCE
    # was never examined. That is an INFRA verdict by construction — it must
    # never be reported as, or counted toward, a non-compiling commit.
    kept="$(preserve_log "$short" "$log")"
    echo "  $short  INFRA-FAIL  (worktree add failed — the build never ran; NOT a source defect)"
    echo "      full log: $kept"
    return 3
  fi

  # The inner module `replace`s to ../submodules/*. A detached worktree carries
  # no submodule content, so bind the live submodule trees in. This holds
  # submodule state constant across commits, isolating the main-repo delta as
  # the single variable under test.
  rm -rf "$dir/submodules"
  ln -s "$ROOT/submodules" "$dir/submodules"

  start=$(date +%s)
  if [ -n "$TAGS" ]; then
    ( cd "$dir/helix_code" && go test -tags="$TAGS" -run='^$' -count=1 ./... ) >>"$log" 2>&1
  else
    ( cd "$dir/helix_code" && go test -run='^$' -count=1 ./... ) >>"$log" 2>&1
  fi
  rc=$?
  end=$(date +%s)

  git -C "$ROOT" worktree remove --force "$dir" >/dev/null 2>&1

  if [ $rc -eq 0 ]; then
    echo "  $short  COMPILES     $((end-start))s"
    return 0
  fi

  # Build failed. WHY it failed decides whether the COMMIT is at fault.
  verdict="$(classify_build_failure "$log")"
  kept="$(preserve_log "$short" "$log")"

  case "$verdict" in
    COMPILE-FAIL)
      echo "  $short  DOES-NOT-COMPILE  $((end-start))s"
      grep -E '[^[:space:]]+\.go:[0-9]+:[0-9]+: |\[build failed\]' "$log" | head -8 | sed 's/^/      /'
      echo "      full log: $kept"
      return 1 ;;
    INFRA)
      echo "  $short  INFRA-FAIL  $((end-start))s  (toolchain/host failure — NOT a source defect)"
      grep -oE "$INFRA_SIGNATURES" "$log" | sort -u | head -4 | sed 's/^/      signature: /'
      echo "      full log: $kept"
      return 3 ;;
    *)
      # Unrecognised. Do NOT claim the commit is broken and do NOT claim PASS.
      echo "  $short  BUILD-FAILED-UNCLASSIFIED  $((end-start))s  (no compiler diagnostic, no known host signature)"
      grep -E '\[build failed\]|^#' "$log" | head -6 | sed 's/^/      /'
      echo "      full log: $kept  <- inspect this; extend INFRA_SIGNATURES or fix the source"
      return 4 ;;
  esac
}

# ---------------------------------------------------------------- self-test
if [ "$SELFTEST" = 1 ]; then
  echo "§1.1 paired-mutation self-test (tags='${TAGS}')"
  st_rc=0

  # The classifier decides who gets blamed, so validate it first — it is free.
  classifier_self_test || { echo "  => SELF-TEST FAILED: classifier mis-classifies a golden fixture"; st_rc=1; }

  echo "expect DOES-NOT-COMPILE on known-bad $KNOWN_BAD:"
  check_commit "$KNOWN_BAD"; bad_rc=$?
  echo "expect COMPILES on known-good $KNOWN_GOOD:"
  check_commit "$KNOWN_GOOD"; good_rc=$?

  # HXC-215. Distinguish "the gate is a bluff" from "the fixtures could not be
  # built here" — collapsing those two is the same false-accusation defect this
  # ticket fixes, aimed at the gate itself.
  #
  # The pins are 130+ commits old, and check_commit symlinks the LIVE submodule
  # trees over each checkout to hold submodule state constant. Once the inner
  # go.mod/go.sum at a pin no longer resolves against today's submodules, `go
  # test` dies with `go: updates to go.mod needed` before compiling a single
  # line — so NEITHER pin can be built and neither verdict means anything.
  # (Observed 2026-07-31: both pins UNCLASSIFIED in 0s, go.mod 9c9b5912 at the
  # pins vs 4960895d at HEAD.)
  #
  # NOTE the old gate could not see this at all: it returned 1 for ANY non-zero
  # exit, so the known-bad half "passed" while the compiler never ran — a
  # falsifiability proof that proved nothing.
  if [ $bad_rc -ge 3 ] || [ $good_rc -ge 3 ]; then
    echo "SELF-TEST INCONCLUSIVE (§11.4.3): the pinned fixtures could not be BUILT in"
    echo "this environment (known-bad rc=$bad_rc, known-good rc=$good_rc), so neither"
    echo "half exercised the gate. This is NOT evidence the gate is a bluff and NOT a"
    echo "pass. Read the preserved log path printed above for the reason."
    echo "To restore a real §1.1 pairing here, either repin KNOWN_BAD/KNOWN_GOOD to a"
    echo "recent pair whose go.mod resolves against current submodules, or exercise the"
    echo "gate against a synthetic broken commit built from HEAD."
    exit 3
  fi

  if [ $bad_rc -eq 1 ]; then
    echo "  => PASS: gate correctly FAILED the known-bad commit"
  else
    echo "  => SELF-TEST FAILED: gate did not fail known-bad (rc=$bad_rc) — gate is a bluff"
    st_rc=1
  fi

  if [ $good_rc -eq 0 ]; then
    echo "  => PASS: gate correctly PASSED the known-good commit"
  else
    echo "  => SELF-TEST FAILED: gate false-positives on known-good (rc=$good_rc)"
    st_rc=1
  fi

  [ $st_rc -eq 0 ] && echo "SELF-TEST PASS: gate can both fail and pass — §1.1 pairing intact"
  exit $st_rc
fi

# ---------------------------------------------------------------- range mode
if [ -z "$RANGE" ]; then
  if [ -n "$LAST" ]; then
    RANGE="HEAD~${LAST}..HEAD"
  elif up="$(git rev-parse --abbrev-ref '@{u}' 2>/dev/null)"; then
    RANGE="${up}..HEAD"
  else
    echo "SKIP §11.4.3: no upstream configured and no --range/--last given"
    exit 0
  fi
fi

mapfile -t COMMITS < <(git rev-list --reverse "$RANGE" 2>/dev/null)
if [ "${#COMMITS[@]}" -eq 0 ]; then
  echo "CM-COMMIT-COMPILE-INTEGRITY: no commits in range '$RANGE' — nothing to check"
  exit 0
fi

echo "CM-COMMIT-COMPILE-INTEGRITY (§11.4.108) range='$RANGE' tags='${TAGS}'"
checked=0; skipped=0; failed=0; infra=0; unclassified=0
declare -a BROKEN=()
declare -a UNCLEAR=()

for sha in "${COMMITS[@]}"; do
  short="$(git rev-parse --short=8 "$sha")"
  # Only commits that touch the inner Go module can break its compilation.
  if ! git show --name-only --format='' "$sha" | grep -qE '^helix_code/.*\.go$'; then
    skipped=$((skipped+1))
    continue
  fi
  checked=$((checked+1))
  check_commit "$sha"; rc=$?
  case $rc in
    1) failed=$((failed+1)); BROKEN+=("$short") ;;
    3) infra=$((infra+1)) ;;
    4) unclassified=$((unclassified+1)); UNCLEAR+=("$short") ;;
  esac
done

echo "---"
echo "commits in range: ${#COMMITS[@]} | compile-checked: $checked | skipped (no inner Go): $skipped"
echo "non-compiling: $failed | infrastructure failures: $infra | unclassified build failures: $unclassified"

if [ "$failed" -gt 0 ]; then
  echo "NON-COMPILING COMMITS: ${BROKEN[*]}"
  echo "FAIL §11.4.108: a commit that does not compile shipped. The working-tree"
  echo "gate cannot see this class — a symbol referenced by a committed file must"
  echo "itself be committed."
  echo "(Each blamed commit above carries at least one file.go:line:col compiler"
  echo "diagnostic — that diagnostic IS the evidence the source is broken.)"
  [ "$ADVISORY" = 1 ] && { echo "(--advisory: exiting 0)"; exit 0; }
  exit 1
fi

if [ "$unclassified" -gt 0 ]; then
  echo "UNCLASSIFIED BUILD FAILURES: ${UNCLEAR[*]}"
  echo "INCONCLUSIVE: the build failed with no compiler diagnostic and no known"
  echo "host-failure signature. This is NOT a claim that those commits are broken"
  echo "(§11.4.6 — the evidence does not establish it) and NOT a PASS. Read the"
  echo "preserved log printed above before acting."
  [ "$ADVISORY" = 1 ] && exit 0
  exit 3
fi

if [ "$infra" -gt 0 ]; then
  echo "WARN: $infra commit(s) could not be checked (host/toolchain failure, not a"
  echo "source defect) — not a PASS for those, and not an accusation against them."
  [ "$ADVISORY" = 1 ] && exit 0
  exit 3
fi

echo "PASS: every compile-relevant commit in range type-checks at its own tree."
exit 0
