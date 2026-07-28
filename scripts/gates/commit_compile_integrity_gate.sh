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
#                        Proves the gate can actually fail.
set -uo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

# §1.1 self-test pins. Immutable history; do not repoint without re-verifying.
KNOWN_BAD="3fd55a4d"   # references respModel before it was committed
KNOWN_GOOD="34e264e1"  # the commit that landed respModel

RANGE=""; LAST=""; ADVISORY=0; SELFTEST=0; TAGS="nogui"
while [ $# -gt 0 ]; do
  case "$1" in
    --range)     RANGE="${2:-}"; shift 2 ;;
    --last)      LAST="${2:-}"; shift 2 ;;
    --tags)      TAGS="${2:-}"; shift 2 ;;
    --advisory)  ADVISORY=1; shift ;;
    --self-test) SELFTEST=1; shift ;;
    -h|--help)   sed -n '1,60p' "${BASH_SOURCE[0]}"; exit 0 ;;
    *)           echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

command -v go >/dev/null 2>&1 || { echo "SKIP §11.4.3: go toolchain not on PATH"; exit 0; }

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
trap cleanup EXIT INT TERM

# Compile-check ONE commit in an isolated detached worktree.
# Returns 0 = compiles, 1 = does not compile, 3 = infrastructure failure.
check_commit() {
  local sha="$1" short dir log rc start end
  short="$(git rev-parse --short=8 "$sha")"
  dir="$WORKDIR/$short"
  log="$WORKDIR/$short.log"

  if ! git -C "$ROOT" worktree add --detach "$dir" "$sha" >"$log" 2>&1; then
    echo "  $short  INFRA-FAIL  (worktree add failed; see $log)"
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
  echo "  $short  DOES-NOT-COMPILE  $((end-start))s"
  grep -E '^[^ ]+\.go:[0-9]+:[0-9]+:|\[build failed\]' "$log" | head -8 | sed 's/^/      /'
  return 1
}

# ---------------------------------------------------------------- self-test
if [ "$SELFTEST" = 1 ]; then
  echo "§1.1 paired-mutation self-test (tags='${TAGS}')"
  st_rc=0

  echo "expect DOES-NOT-COMPILE on known-bad $KNOWN_BAD:"
  check_commit "$KNOWN_BAD"; bad_rc=$?
  if [ $bad_rc -eq 1 ]; then
    echo "  => PASS: gate correctly FAILED the known-bad commit"
  else
    echo "  => SELF-TEST FAILED: gate did not fail known-bad (rc=$bad_rc) — gate is a bluff"
    st_rc=1
  fi

  echo "expect COMPILES on known-good $KNOWN_GOOD:"
  check_commit "$KNOWN_GOOD"; good_rc=$?
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
checked=0; skipped=0; failed=0; infra=0
declare -a BROKEN=()

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
  esac
done

echo "---"
echo "commits in range: ${#COMMITS[@]} | compile-checked: $checked | skipped (no inner Go): $skipped"
echo "non-compiling: $failed | infrastructure failures: $infra"

if [ "$failed" -gt 0 ]; then
  echo "NON-COMPILING COMMITS: ${BROKEN[*]}"
  echo "FAIL §11.4.108: a commit that does not compile shipped. The working-tree"
  echo "gate cannot see this class — a symbol referenced by a committed file must"
  echo "itself be committed."
  [ "$ADVISORY" = 1 ] && { echo "(--advisory: exiting 0)"; exit 0; }
  exit 1
fi

if [ "$infra" -gt 0 ]; then
  echo "WARN: $infra commit(s) could not be checked (infrastructure) — not a PASS for those."
  [ "$ADVISORY" = 1 ] && exit 0
  exit 1
fi

echo "PASS: every compile-relevant commit in range type-checks at its own tree."
exit 0
