#!/usr/bin/env bash
# fyneui_uithread_dispatch_meta_test.sh — §1.1 paired-mutation meta-test for the
# CM-FYNEUI-UITHREAD-DISPATCH gate (scripts/gates/fyneui_uithread_dispatch_gate.sh).
#
# A gate is only worth having if it can FAIL. This meta-test proves both
# directions, and — equally load-bearing per §11.4.201 — proves the gate does
# NOT cry wolf on the real tree. A false-positive refusal is exactly as damaging
# as a false-negative pass: it gets the gate switched off, which is strictly
# worse than never having written it.
#
# Assertions:
#   (1) MUTATED fixture: a main-goroutine fyneui.DoAndWait  -> gate FAILs (1).
#   (2) FIXED fixture:   the same call inside a `go func`    -> gate PASSes (0).
#   (3) TRANSITIVE:      a named helper reached from a go func -> PASSes (0).
#         This is the shape of the LIVE tree's desktop/main.go streamDesktopChat,
#         which a naive "must be lexically inside go func" rule would reject.
#   (4) REAL TREE:       helix_code/applications                -> PASSes (0).
#         Guards the zero-false-positive property against future drift.
#   (5) REAL TREE + PLANTED VIOLATION (on a COPY, never the live tree):
#         -> gate FAILs (1). This is the assertion that matters most: it proves
#         the gate still discriminates in the presence of the real tree's large
#         goroutine-reachable set, rather than passing everything vacuously.
#   (6) The gate's own --self-test                              -> exits 0.
#
# The live repository is never mutated: fixtures live in a temp dir, and (5)
# operates on a `cp -r` copy. No mutation residue can survive (§11.4.84).
#
# Honest shebang, `bash -n` clean (CONST-068).

set -uo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
GATE="$ROOT/scripts/gates/fyneui_uithread_dispatch_gate.sh"
REAL_TREE="$ROOT/helix_code/applications"

if [ ! -x "$GATE" ]; then
  echo "META-TEST FAIL: gate not found or not executable: $GATE"
  exit 1
fi

FIXTURE_DIR="$(mktemp -d "${TMPDIR:-/tmp}/fyneui_dispatch_meta.XXXXXX")"
cleanup() { rm -rf "$FIXTURE_DIR"; }
trap cleanup EXIT INT TERM

fail_meta=0
assert() {
  # assert <desc> <expected-exit> <actual-exit>
  if [ "$2" -eq "$3" ]; then
    echo "  ASSERT PASS — $1 (exit=$3)"
  else
    echo "  ASSERT FAIL — $1 (expected exit=$2, got exit=$3)"
    fail_meta=1
  fi
}

echo "=== meta-test: CM-FYNEUI-UITHREAD-DISPATCH paired mutation ==="
echo "Gate under test: $GATE"
echo "Fixture dir:     $FIXTURE_DIR"
echo

SRC="$FIXTURE_DIR/src"
mkdir -p "$SRC"

# --- (1) MUTATED: main-goroutine call site ----------------------------------
cat > "$SRC/app.go" <<'GOEOF'
package app

// No goroutine anywhere: this call runs on the main/UI goroutine. Under glfw it
// queues onto the FIFO the main goroutine itself drains, then blocks on it.
func buildTab() {
	fyneui.DoAndWait(func() { label.SetText("frozen UI") })
}
GOEOF
"$GATE" "$SRC" > "$FIXTURE_DIR/out1.txt" 2>&1
rc1=$?
echo "--- gate output on MUTATED fixture ---"
sed 's/^/    /' "$FIXTURE_DIR/out1.txt"
assert "gate FAILs on a main-goroutine fyneui.DoAndWait call site" 1 "$rc1"
echo

# --- (2) FIXED: same call, inside a go func ---------------------------------
cat > "$SRC/app.go" <<'GOEOF'
package app

func buildTab() {
	go func() {
		fyneui.DoAndWait(func() { label.SetText("dispatched from a worker") })
	}()
}
GOEOF
"$GATE" "$SRC" > "$FIXTURE_DIR/out2.txt" 2>&1
rc2=$?
echo "--- gate output on FIXED fixture ---"
sed 's/^/    /' "$FIXTURE_DIR/out2.txt"
assert "gate PASSes once the call is inside a go func body" 0 "$rc2"
echo

# --- (3) TRANSITIVE: named helper reached from a go func --------------------
cat > "$SRC/app.go" <<'GOEOF'
package app

// Shape of the live tree's desktop/main.go streamDesktopChat: Do is called at
// the top level of a NAMED function, which is only ever entered from a worker.
func streamChat() {
	fyneui.Do(func() { label.SetText("token") })
}

func buildTab() {
	go func() {
		streamChat()
	}()
}
GOEOF
"$GATE" "$SRC" > "$FIXTURE_DIR/out3.txt" 2>&1
rc3=$?
echo "--- gate output on TRANSITIVE fixture ---"
sed 's/^/    /' "$FIXTURE_DIR/out3.txt"
assert "gate PASSes for a helper reached transitively from a go func" 0 "$rc3"
echo

# --- (4) REAL TREE: zero false positives ------------------------------------
if [ -d "$REAL_TREE" ]; then
  "$GATE" "$REAL_TREE" > "$FIXTURE_DIR/out4.txt" 2>&1
  rc4=$?
  echo "--- gate output on the REAL applications tree ---"
  sed 's/^/    /' "$FIXTURE_DIR/out4.txt"
  assert "gate PASSes on the real tree (no false positives, §11.4.201)" 0 "$rc4"
else
  echo "  SKIP §11.4.3 — real tree absent: $REAL_TREE"
fi
echo

# --- (5) REAL TREE + planted violation, on a COPY ---------------------------
# The decisive assertion: the gate must still discriminate against the real
# tree's full goroutine-reachable set, not pass everything vacuously.
if [ -d "$REAL_TREE" ]; then
  COPY="$FIXTURE_DIR/realcopy"
  mkdir -p "$COPY"
  cp -r "$REAL_TREE" "$COPY/applications"
  TARGET="$COPY/applications/desktop/main.go"
  if [ -f "$TARGET" ]; then
    cat >> "$TARGET" <<'GOEOF'

// PLANTED BY META-TEST (§1.1) — main-goroutine misuse, never reached from a
// goroutine. Exists only inside a throwaway copy under $TMPDIR.
func plantedMainGoroutineMisuse(label *widget.Entry) {
	fyneui.DoAndWait(func() { label.SetText("self-deadlock under glfw") })
}
GOEOF
    "$GATE" "$COPY/applications" > "$FIXTURE_DIR/out5.txt" 2>&1
    rc5=$?
    echo "--- gate output on REAL-TREE COPY with a planted violation ---"
    sed 's/^/    /' "$FIXTURE_DIR/out5.txt"
    assert "gate FAILs on a violation planted into a copy of the real tree" 1 "$rc5"
    if grep -q 'plantedMainGoroutineMisuse' "$FIXTURE_DIR/out5.txt"; then
      echo "  ASSERT PASS — gate names the offending function in its report"
    else
      echo "  ASSERT FAIL — gate failed but did not identify the planted function"
      fail_meta=1
    fi
  else
    echo "  SKIP §11.4.3 — expected file absent: $TARGET"
  fi
else
  echo "  SKIP §11.4.3 — real tree absent, cannot run the discrimination assertion"
fi
echo

# --- (6) the gate's own embedded self-test ----------------------------------
"$GATE" --self-test > "$FIXTURE_DIR/out6.txt" 2>&1
rc6=$?
echo "--- gate --self-test ---"
sed 's/^/    /' "$FIXTURE_DIR/out6.txt"
assert "gate's own --self-test passes" 0 "$rc6"
echo

# --- residue check (§11.4.84) ----------------------------------------------
# Nothing this meta-test wrote may exist outside the fixture dir.
if [ -d "$REAL_TREE" ] && grep -rqs 'plantedMainGoroutineMisuse' "$REAL_TREE"; then
  echo "META-TEST FAIL: mutation residue found in the LIVE tree — this must never happen"
  exit 1
fi
echo "  ASSERT PASS — no mutation residue in the live tree (§11.4.84)"
echo

if [ "$fail_meta" -ne 0 ]; then
  echo "META-TEST FAIL: paired-mutation assertions did not hold"
  exit 1
fi
echo "META-TEST PASS: gate FAILs on main-goroutine misuse (fixture AND real-tree copy),"
echo "PASSes on both correct dispatch shapes, and does not false-positive on the real tree."
exit 0
