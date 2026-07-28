#!/usr/bin/env bash
# scripts/gates/fyneui_uithread_dispatch_gate.sh — CM-FYNEUI-UITHREAD-DISPATCH
# (§11.4.108 SOURCE-layer gate; §11.4.110 diff-catchable clash class).
#
# WHAT THIS CLOSES
# ----------------
# internal/fyneui documents, in prose, that Do / DoAndWait MUST be called ONLY
# from a goroutine the caller spawned. Prose is not enforcement. The failure the
# rule guards is severe and SILENT at compile time:
#
#   glfw (production). fyne's runOnMainWithWait executes fn directly only when
#   the app is NOT yet running and the caller is the main goroutine, or when the
#   queue has already drained at shutdown (internal/driver/glfw/loop.go:40-57).
#   Once the app IS running, a DoAndWait issued FROM the main goroutine appends
#   fn onto the very FIFO that the main goroutine itself drains, then blocks on
#   `done`. Nothing can drain it. That is a SELF-DEADLOCK: a permanently frozen
#   UI, with no panic, no log line, and no test failure.
#
#   test driver. Main-goroutine misuse is not silent but is still wrong: fyne
#   logs a thread error and re-dispatches onto a NEW goroutine, manufacturing
#   the exact race internal/fyneui exists to remove.
#
# WHY A STATIC GATE AND NOT A RUNTIME ASSERTION (§11.4.6 — honest boundary).
# A cheap runtime "am I on the main goroutine?" check is genuinely unavailable:
# Go deliberately hides goroutine identity, and fyne's own IsMainGoroutine lives
# in an internal/ package that is not importable from here. Fyne's public API
# ships the identical doc-only contract. So this gate does NOT close the class —
# it raises the cost of the most common way to enter it (writing the call in
# plainly main-goroutine code). Read a PASS as "no textually-detectable
# main-goroutine call site", never as "main-goroutine misuse is impossible".
#
# THE RULE
# --------
# A call to fyneui.Do / fyneui.DoAndWait is ACCEPTED when either holds:
#   (a) DIRECT   — it is lexically inside a `go func` body; or
#   (b) INDIRECT — its enclosing function is goroutine-reachable: the function
#                  is invoked from inside a `go func` body, or from another
#                  goroutine-reachable function (computed to a fixpoint).
# Anything else is a VIOLATION.
#
# Clause (b) is NOT optional generosity — it is required for correctness. The
# live tree already contains a legitimate call site that clause (a) alone would
# reject: applications/desktop/main.go streamDesktopChat() calls fyneui.Do at
# top level of a NAMED function, and is invoked from the `go func` at main.go
# ~1375. A gate that flagged it would be a §11.4.201 FAIL-bluff (a false-positive
# refusal is exactly as damaging as a false-negative pass — it gets the gate
# switched off, which is worse than having no gate).
#
# WHAT IT CANNOT CATCH (§11.4.3 — stated, not hidden)
# ---------------------------------------------------
# This is a TEXTUAL analyser. It parses no types and resolves no symbols.
#   1. INDIRECTION defeats it. A call reached through a stored func value, an
#      interface method, a callback registered elsewhere, reflection, or a
#      function-typed struct field is invisible: the gate sees no textual call
#      edge, so such a helper is NOT marked goroutine-reachable (a false
#      POSITIVE risk), and conversely a main-goroutine path that reaches Do
#      through a func value is NOT flagged (a false NEGATIVE).
#   2. NAME-ONLY reachability. Call edges match on the bare identifier, so two
#      different types with a same-named method are conflated. This is
#      deliberately biased toward OVER-marking a function as goroutine-reachable
#      — the conservative direction, which trades a possible missed violation
#      for never manufacturing a false refusal.
#   3. Reachability is computed WITHIN the scanned root only. A helper defined
#      in another package and called from a goroutine there is not seen.
#   4. It proves nothing about RUNTIME. A function that is goroutine-reachable
#      may ALSO be called from the main goroutine on another path; the gate
#      cannot distinguish. That residual is what item (2) above trades for.
#   5. _test.go files are OUT of scope by design: tests legitimately drive these
#      entry points from the test goroutine under the headless driver.
#
# MODES
#   (default)      scan helix_code/applications (production, non-test *.go).
#   <dir>          scan an explicit root instead (used by the meta-test to
#                  point at a fixture tree, so the gate never rewrites the repo).
#   --advisory     report findings but always exit 0.
#   --self-test    §1.1 paired mutation, self-contained in a temp fixture:
#                  plant a main-goroutine call site -> assert the gate FAILS;
#                  move it into a `go func` -> assert the gate PASSES.
#                  Proves the gate is capable of failing.
#
# EXIT CODES: 0 = PASS (or advisory), 1 = FAIL (violations found), 2 = usage.
#
# Honest shebang; `bash -n` clean (CONST-068).

set -uo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"

SCAN_ROOT=""
ADVISORY=0
SELFTEST=0

while [ $# -gt 0 ]; do
  case "$1" in
    --advisory)  ADVISORY=1; shift ;;
    --self-test) SELFTEST=1; shift ;;
    -h|--help)   sed -n '1,80p' "${BASH_SOURCE[0]}"; exit 0 ;;
    -*)          echo "unknown arg: $1" >&2; exit 2 ;;
    *)           SCAN_ROOT="$1"; shift ;;
  esac
done

# ---------------------------------------------------------------------------
# Stage 1 — sanitiser. Blanks out comments, string/rune/raw literals so that
# brace counting cannot be thrown off by a `{` in a string, and so that a
# comment MENTIONING fyneui.Do (there are several in this tree) is never
# mistaken for a call. Line numbering is preserved 1:1.
# ---------------------------------------------------------------------------
SANITIZE_AWK='
{
  line = $0; out = ""; i = 1; n = length(line)
  while (i <= n) {
    c = substr(line, i, 1)
    if (inblock) {
      if (c == "*" && substr(line, i+1, 1) == "/") { inblock = 0; i += 2; out = out "  "; continue }
      i++; out = out " "; continue
    }
    if (inraw) {
      if (c == "`") { inraw = 0 }
      i++; out = out " "; continue
    }
    if (c == "/" && substr(line, i+1, 1) == "/") { break }
    if (c == "/" && substr(line, i+1, 1) == "*") { inblock = 1; i += 2; out = out "  "; continue }
    if (c == "`") { inraw = 1; i++; out = out " "; continue }
    if (c == "\"") {
      i++; out = out " "
      while (i <= n) {
        ch = substr(line, i, 1)
        if (ch == "\\") { i += 2; out = out "  "; continue }
        out = out " "; i++
        if (ch == "\"") { break }
      }
      continue
    }
    if (c == "'"'"'") {
      i++; out = out " "
      while (i <= n) {
        ch = substr(line, i, 1)
        if (ch == "\\") { i += 2; out = out "  "; continue }
        out = out " "; i++
        if (ch == "'"'"'") { break }
      }
      continue
    }
    out = out c; i++
  }
  print out
}
'

# ---------------------------------------------------------------------------
# Stage 2 — analyser. Emits two record kinds on the sanitised stream:
#   GOFN <ident>                      ident is called from goroutine context
#   CALL <file> <line> <fn> <verdict> a fyneui.Do/DoAndWait call site
# `known` (goroutine-reachable function names so far) is read from $GOFNS,
# which the driver grows to a fixpoint.
# ---------------------------------------------------------------------------
ANALYZE_AWK='
BEGIN {
  while ((getline ln < GOFNS) > 0) { if (ln != "") known[ln] = 1 }
  close(GOFNS)
}
FNR == 1 { depth = 0; gotop = 0; fname = ""; armed = 0; armgate = 0 }
{
  line = $0
  n = length(line)

  # `go func` — arm the next `{` as a goroutine-body opener.
  go_at = 0
  if (match(line, /(^|[^A-Za-z0-9_.])go[ \t]+func/)) { go_at = RSTART; armed = 1; armgate = RSTART }

  # `go someFn(` / `go x.someFn(` — that callee runs on a new goroutine.
  if (go_at == 0 && match(line, /(^|[^A-Za-z0-9_.])go[ \t]+[A-Za-z_][A-Za-z0-9_.]*[ \t]*\(/)) {
    seg = substr(line, RSTART, RLENGTH)
    sub(/[ \t]*\($/, "", seg)
    cnt = split(seg, p, /[^A-Za-z0-9_]+/)
    if (cnt >= 2 && p[cnt] != "func") { print "GOFN " p[cnt] }
  }

  # Top-level func declaration: remember its name for enclosing-scope attribution.
  if (depth == 0 && match(line, /^func[ \t]/)) {
    if (match(line, /^func[ \t]*\([^)]*\)[ \t]*[A-Za-z_][A-Za-z0-9_]*/)) {
      seg = substr(line, RSTART, RLENGTH)
    } else if (match(line, /^func[ \t]+[A-Za-z_][A-Za-z0-9_]*/)) {
      seg = substr(line, RSTART, RLENGTH)
    } else { seg = "" }
    if (seg != "") { cnt = split(seg, p, /[^A-Za-z0-9_]+/); fname = p[cnt] }
  }

  sawcontext = (gotop > 0) || (fname != "" && (fname in known))

  i = 1
  while (i <= n) {
    c = substr(line, i, 1)

    if (c == "{") {
      depth++
      if (armed == 1 && i > armgate) { gotop++; gostack[gotop] = depth; armed = 0; armgate = 0 }
      if (gotop > 0) { sawcontext = 1 }
      i++; continue
    }
    if (c == "}") {
      if (gotop > 0 && depth == gostack[gotop]) { gotop-- }
      depth--
      if (depth <= 0) { depth = 0; fname = "" }
      i++; continue
    }
    if (c == "f" && substr(line, i, 7) == "fyneui.") {
      rest = substr(line, i)
      if (match(rest, /^fyneui\.(DoAndWait|Do)[ \t]*\(/)) {
        ok = (gotop > 0) || (fname != "" && (fname in known))
        print "CALL " FILENAME " " FNR " " (fname == "" ? "<file-scope>" : fname) " " (ok ? "OK" : "VIOLATION")
      }
    }
    i++
  }

  # Harvest call edges out of goroutine context so the driver can close the
  # transitive set. Over-marking here is the safe direction (see header note 2).
  if (sawcontext) {
    tmp = line
    while (match(tmp, /[A-Za-z_][A-Za-z0-9_]*[ \t]*\(/)) {
      seg = substr(tmp, RSTART, RLENGTH)
      sub(/[ \t]*\($/, "", seg)
      gsub(/[ \t]/, "", seg)
      if (seg != "func" && seg != "if" && seg != "for" && seg != "switch" && seg != "return") {
        print "GOFN " seg
      }
      tmp = substr(tmp, RSTART + RLENGTH)
    }
  }
}
'

# ---------------------------------------------------------------------------
# run_scan <dir> -> prints CALL records for the fixpoint-closed known set.
# ---------------------------------------------------------------------------
run_scan() {
  local dir="$1"
  local work gofns prev files
  work="$(mktemp -d "${TMPDIR:-/tmp}/fyneui-gate.XXXXXX")"
  gofns="$work/gofns"
  prev="$work/gofns.prev"
  : > "$gofns"

  # Sanitise every in-scope file once, keeping the original path as FILENAME by
  # mirroring the tree under $work/src.
  files="$work/files"
  find "$dir" -type f -name '*.go' ! -name '*_test.go' 2>/dev/null | sort > "$files"
  if [ ! -s "$files" ]; then
    echo "__NOFILES__"
    rm -rf "$work"
    return 0
  fi

  local f san
  while IFS= read -r f; do
    san="$work/src/${f#/}"
    mkdir -p "$(dirname "$san")"
    awk "$SANITIZE_AWK" "$f" > "$san"
  done < "$files"

  # Fixpoint: grow the goroutine-reachable set until it stops changing.
  local iter=0 out="$work/out"
  while [ "$iter" -lt 12 ]; do
    cp "$gofns" "$prev"
    ( cd "$work/src" && find . -type f -name '*.go' | sort | xargs awk -v GOFNS="$gofns" "$ANALYZE_AWK" ) > "$out" 2>/dev/null
    grep '^GOFN ' "$out" | awk '{print $2}' | sort -u > "$gofns.new"
    cat "$prev" "$gofns.new" | sort -u > "$gofns"
    if cmp -s "$prev" "$gofns"; then break; fi
    iter=$((iter+1))
  done

  grep '^CALL ' "$out" | sed 's#^CALL \./#CALL #'
  rm -rf "$work"
}

# ---------------------------------------------------------------------------
# §1.1 paired-mutation self-test — fully self-contained in a temp fixture.
# ---------------------------------------------------------------------------
if [ "$SELFTEST" = 1 ]; then
  FIX="$(mktemp -d "${TMPDIR:-/tmp}/fyneui-selftest.XXXXXX")"
  trap 'rm -rf "$FIX"' EXIT INT TERM
  st_rc=0

  # (1) MUTATED: a main-goroutine call site (no `go func` anywhere).
  cat > "$FIX/app.go" <<'GOEOF'
package app

func buildTab() {
	fyneui.DoAndWait(func() { label.SetText("on the main goroutine — self-deadlock under glfw") })
}
GOEOF
  echo "§1.1 paired-mutation self-test"
  echo "expect VIOLATION on a main-goroutine call site:"
  out1="$(run_scan "$FIX")"
  echo "$out1" | sed 's/^/    /'
  if echo "$out1" | grep -q 'VIOLATION'; then
    echo "  => PASS: gate correctly FLAGGED the planted main-goroutine call"
  else
    echo "  => SELF-TEST FAILED: planted violation not detected — gate is a bluff"
    st_rc=1
  fi

  # (2) FIXED: identical call, moved inside a `go func` body.
  cat > "$FIX/app.go" <<'GOEOF'
package app

func buildTab() {
	go func() {
		fyneui.DoAndWait(func() { label.SetText("dispatched from a worker goroutine") })
	}()
}
GOEOF
  echo "expect OK once the same call moves inside a go func:"
  out2="$(run_scan "$FIX")"
  echo "$out2" | sed 's/^/    /'
  if echo "$out2" | grep -q 'VIOLATION'; then
    echo "  => SELF-TEST FAILED: gate still flags a correctly-dispatched call (false positive)"
    st_rc=1
  else
    echo "  => PASS: gate correctly ACCEPTED the goroutine-dispatched call"
  fi

  # (3) INDIRECT: the live tree's streamDesktopChat shape — a named helper that
  #     calls Do at its top level, invoked from a `go func`. Must be ACCEPTED.
  cat > "$FIX/app.go" <<'GOEOF'
package app

func streamChat() {
	fyneui.Do(func() { label.SetText("helper, reached only from a worker") })
}

func buildTab() {
	go func() {
		streamChat()
	}()
}
GOEOF
  echo "expect OK for a helper reached transitively from a go func:"
  out3="$(run_scan "$FIX")"
  echo "$out3" | sed 's/^/    /'
  if echo "$out3" | grep -q 'VIOLATION'; then
    echo "  => SELF-TEST FAILED: transitive reachability not honoured — §11.4.201 false positive"
    st_rc=1
  else
    echo "  => PASS: gate correctly ACCEPTED the transitively-reached helper"
  fi

  if [ "$st_rc" -eq 0 ]; then
    echo "SELF-TEST PASS: gate fails on the bluff and passes both correct shapes — §1.1 pairing intact"
  fi
  exit $st_rc
fi

# ---------------------------------------------------------------------------
# Normal run.
# ---------------------------------------------------------------------------
if [ -z "$SCAN_ROOT" ]; then
  SCAN_ROOT="$ROOT/helix_code/applications"
fi

if [ ! -d "$SCAN_ROOT" ]; then
  echo "SKIP §11.4.3: scan root not present: $SCAN_ROOT"
  exit 0
fi

echo "CM-FYNEUI-UITHREAD-DISPATCH (§11.4.108) root='$SCAN_ROOT'"

RESULT="$(run_scan "$SCAN_ROOT")"

if [ "$RESULT" = "__NOFILES__" ]; then
  echo "SKIP §11.4.3: no non-test *.go files under $SCAN_ROOT"
  exit 0
fi

total=$(printf '%s\n' "$RESULT" | grep -c '^CALL ' || true)
viol=$(printf '%s\n' "$RESULT" | grep -c 'VIOLATION$' || true)
okc=$((total - viol))

printf '%s\n' "$RESULT" | grep 'VIOLATION$' | while read -r _ f l fn _; do
  echo "  VIOLATION  $f:$l  in $fn()"
done

echo "---"
echo "fyneui.Do/DoAndWait call sites: $total | goroutine-dispatched: $okc | violations: $viol"

if [ "$viol" -gt 0 ]; then
  echo "FAIL: a fyneui.Do/DoAndWait call site is neither inside a 'go func' body nor"
  echo "in a function reachable from one. Under glfw a DoAndWait issued from the main"
  echo "goroutine queues onto the FIFO that goroutine itself drains, then blocks on it:"
  echo "a silent, permanent UI freeze. Move the call onto a worker goroutine, or (if it"
  echo "is main-goroutine code that must be mutually exclusive with workers) use"
  echo "fyneui.Sync, which runs on the caller and is safe from the main goroutine."
  [ "$ADVISORY" = 1 ] && { echo "(--advisory: exiting 0)"; exit 0; }
  exit 1
fi

echo "PASS: every fyneui.Do/DoAndWait call site is goroutine-dispatched."
echo "NOTE (§11.4.6): textual analysis — see this script's header for the four"
echo "classes it provably cannot catch. A PASS is not proof of absence."
exit 0
