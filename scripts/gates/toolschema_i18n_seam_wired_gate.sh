#!/usr/bin/env bash
# toolschema_i18n_seam_wired_gate.sh — CM-TOOLSCHEMA-I18N-DEFAULT-RESOLVES
#
# FILENAME IS HISTORICAL. This file was authored as a source-layer guard for a
# defect whose fix landed by a DIFFERENT mechanism; it has been reconciled per
# §11.4.120 (see RECONCILIATION below) and now asserts a different, stronger
# invariant. The path is kept because the file is referenced by that name in the
# sweep registration; the gate ID inside was changed to describe what it now
# actually asserts, because a gate ID that names the old contract is itself a
# small bluff.
#
# ===========================================================================
# RECONCILIATION RECORD (Constitution §11.4.120)
# ===========================================================================
# WHAT THE OLD GATE ASSERTED (now-removed behaviour)
#   The `digital.vasic.toolschema` package default was:
#       i18n.go:  activeTr Translator = NoopTranslator{}
#   `NoopTranslator.T` returns the message ID VERBATIM, so until some consumer
#   called `SetTranslator`, all 45 `tr()` call sites (9 error-helper wrappers in
#   i18n.go + 36 in handler.go) echoed RAW MESSAGE IDs to users — including
#   user-visible `ToolResult.Error` payloads. The old gate therefore:
#     * hard-required the literal `activeTr Translator = NoopTranslator{}` as a
#       "seam anchor" (exit 2 if absent), and
#     * at RED_MODE=0 required "at least one NON-TEST caller of SetTranslator
#       exists" — i.e. it demanded a COMPOSITION ROOT do the wiring.
#
# WHY IT BROKE (and why this is a reconcile, NOT the gate catching a regression)
#   Commit 8cec90b in submodules/tool_schema
#     ("fix(i18n): embed translation bundles so lookups work from any directory")
#   fixed the SAME user-facing defect by a better mechanism: it added
#   i18n_bundle.go (a `go:embed`-compiled English bundle translator) and changed
#   the default to:
#       i18n.go:47  activeTr Translator = defaultTranslator()
#   Verified from git: `git log -L 40,50:i18n.go` shows exactly one change to
#   that line, `NoopTranslator{}` -> `defaultTranslator()`, in 8cec90b.
#   Consequence: the old RED anchor no longer exists (gate exited 2), AND the old
#   GREEN contract became WRONG — no composition root is needed any more, so
#   requiring one would demand wiring that the fix deliberately made redundant.
#   That is the §11.4.120 signature: the gate asserted OLD-CORRECT-NOW-REMOVED
#   behaviour. It is NOT catching a regression — the new behaviour is strictly
#   better for the end user (real text with ZERO wiring, independent of the
#   caller's working directory).
#
# WHAT THIS GATE ASSERTS NOW (the invariant that actually protects users)
#   (1) the package DEFAULT resolves REAL bundle text with ZERO wiring, and
#   (2) `NoopTranslator` STILL echoes verbatim — so the seam remains overridable
#       AND this guard remains falsifiable (see NEGATIVE CONTROL below).
#
# ===========================================================================
# RUNTIME EVIDENCE (Constitution §11.4.5 / §11.4.108) — the upgrade
# ===========================================================================
# The old gate's own header conceded its runtime half was "DELIBERATELY NOT
# IMPLEMENTED ... because it has no observable subject yet". That is no longer
# true: `tr()` now renders a resolvable string, so an in-package Go test CAN
# observe it. This gate therefore runs a REAL RUNTIME PROBE — it executes
# `tr()` in a fresh process and asserts on the rendered output.
#
# The probe is injected with `go test -overlay=...`, which maps a file path
# INTO the package's build WITHOUT writing anything into the module tree. The
# probe source lives in a mktemp dir outside the repository and is removed on
# exit. This is stricter than writing a temp `_test.go` into the submodule and
# deleting it afterwards: there is no window in which the submodule tree is
# dirty at all. The gate still verifies `git status --porcelain` in the
# submodule is EMPTY after the probe and FAILs loudly if it is not.
#
# GROUND TRUTH IS NOT HARDCODED. The probe reads the expected strings from the
# on-disk bundle (i18n/bundles/active.en.yaml) and compares the RUNTIME output
# of `tr()` (which resolves against the go:embed-compiled COPY of that file)
# against them. So the gate asserts "runtime resolves to what the bundle says",
# which survives legitimate bundle-text edits instead of false-FAILing on them
# (§11.4.201 — assert the real condition, not a brittle proxy).
#
# NEGATIVE CONTROL (why this gate cannot silently become unfalsifiable):
#   The probe also drives `SetTranslator(NoopTranslator{})` and asserts the SAME
#   message ID then comes back as the RAW ID. That reproduces the historical
#   defect's exact symptom IN-PROCESS, and proves the probe can distinguish
#   resolved text from a raw echo. If that control ever stops echoing, this gate
#   FAILs in BOTH polarities rather than passing vacuously.
#
# ===========================================================================
# POLARITY SWITCH (Constitution §11.4.115) — re-pointed at the NEW invariant
# ===========================================================================
#   RED_MODE=1  — reproduce the ABSENCE of the new guarantee.
#       PASS iff the zero-wired default does NOT resolve real bundle text (i.e.
#       the pre-fix artifact, where the default echoed raw message IDs).
#       On the CURRENT (fixed) tree this MUST FAIL — that failure is the proof
#       the fix is in.
#
#   RED_MODE=0  — DEFAULT. The standing GREEN regression guard.
#       PASS iff the zero-wired default resolves real bundle text, placeholder
#       interpolation works, `SetTranslator(nil)` restores that default, and
#       `NoopTranslator` still echoes verbatim.
#
#   The default flipped 1 -> 0 relative to the pre-reconcile file because the
#   defect this now guards is ALREADY FIXED; per §11.4.115 the post-fix role of
#   the same source is the standing GREEN guard.
#
# ===========================================================================
# HONEST BOUNDARY (Constitution §11.4.6)
# ===========================================================================
#   * This proves the SOURCE + RUNTIME-in-`go test` layers (§11.4.108 layers 1
#     and a test-process form of 3). It does NOT prove behaviour inside a
#     SHIPPED consumer binary on a clean deployment — `go test` rebuilds from
#     source, so a stale deployed artifact would not be caught here.
#   * The probe exercises the `tr()` seam and one handler-independent path; it
#     does not enumerate all 45 call sites.
#   * The submodule already ships its own permanent regression test
#     (i18n_cwd_regression_test.go). This gate is deliberately INDEPENDENT of
#     it: `go test -run <name>` exits 0 when NO test matches, so a gate that
#     merely invoked that test by name would pass vacuously the moment the test
#     were renamed or deleted. The injected probe cannot be removed by editing
#     the submodule.
#
# ===========================================================================
# EXIT CODES
# ===========================================================================
#   0  gate satisfied for the active RED_MODE
#   1  gate violated for the active RED_MODE (SUBSTANTIVE — includes a seam
#      anchor having changed shape, which demands reconciliation, not silence)
#   2  environment SKIP: the gate could not RUN (module tree or Go toolchain
#      absent, probe failed to execute). Certifies NOTHING — callers MUST NOT
#      report this as a substantive pass or as a detected violation.
#
# Honest shebang; `bash -n` clean (CONST-068).

set -uo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
MODULE_DIR="$ROOT/submodules/tool_schema"
MODULE_PATH="digital.vasic.toolschema"
BUNDLE_REL="i18n/bundles/active.en.yaml"
RED_MODE="${RED_MODE:-0}"

# Message IDs the probe exercises: one plain, one with a {{.Placeholder}}.
PLAIN_ID="toolschema_git_desc_commit"
TMPL_ID="toolschema_err_invalid_file_path"

echo "CM-TOOLSCHEMA-I18N-DEFAULT-RESOLVES  RED_MODE=$RED_MODE"
echo "  module : $MODULE_PATH ($MODULE_DIR)"

# --- Environment preconditions (exit 2 = could not RUN) ----------------------
if [[ ! -f "$MODULE_DIR/i18n.go" ]]; then
    echo "SKIP(env): toolschema module not found at $MODULE_DIR (expected i18n.go)" >&2
    exit 2
fi
if [[ ! -f "$MODULE_DIR/$BUNDLE_REL" ]]; then
    echo "SKIP(env): bundle not found at $MODULE_DIR/$BUNDLE_REL" >&2
    exit 2
fi
if ! command -v go >/dev/null 2>&1; then
    echo "SKIP(env): no Go toolchain on PATH — the runtime probe cannot run" >&2
    exit 2
fi

# --- S1: seam integrity (substantive; contract must still have this shape) ---
# If the seam itself is gone the gate is meaningless and MUST NOT silently
# pass — that is the §11.4.120 fake-pass failure mode. -F because the `(` in
# these needles is literal, not a regex group.
seam_missing=0
for needle in 'func SetTranslator(' 'func tr(' 'func (NoopTranslator) T('; do
    if ! grep -qF "$needle" "$MODULE_DIR/i18n.go"; then
        echo "FAIL: seam anchor missing from $MODULE_DIR/i18n.go: $needle" >&2
        seam_missing=1
    fi
done
if [[ "$seam_missing" -ne 0 ]]; then
    echo "      The guarded contract changed shape — reconcile this gate per §11.4.120," >&2
    echo "      do NOT weaken it to pass." >&2
    exit 1
fi

# --- S2: the historical regression's literal shape (source layer) -----------
# Secondary to the runtime probe below, which is the authority. Kept because it
# names the exact regression this gate exists to stop and gives a precise
# diagnostic. Deliberately NOT an equality check against
# `defaultTranslator()`: any correct resolving default must pass, so pinning
# one spelling would false-FAIL a legitimate replacement (§11.4.201).
src_default_is_noop=0
if grep -qE 'activeTr[[:space:]]+Translator[[:space:]]*=[[:space:]]*NoopTranslator\{\}' "$MODULE_DIR/i18n.go"; then
    src_default_is_noop=1
fi
echo "  source : package default is NoopTranslator{} ? $([[ $src_default_is_noop -eq 1 ]] && echo yes || echo no)"

# --- RUNTIME PROBE ----------------------------------------------------------
PROBE_DIR="$(mktemp -d "${TMPDIR:-/tmp}/toolschema-i18n-probe.XXXXXX")" || {
    echo "SKIP(env): could not create a temp dir for the probe" >&2
    exit 2
}
cleanup() { rm -rf "$PROBE_DIR"; }
trap cleanup EXIT

cat > "$PROBE_DIR/probe_test.go" <<'GOEOF'
package tools

// Injected by scripts/gates/toolschema_i18n_seam_wired_gate.sh via
// `go test -overlay`. NEVER written into the module tree.
//
// This probe is an ORACLE, not a verdict: it reports FACTS about what the
// runtime actually renders and always exits 0 when it could run. The gate
// applies the RED_MODE polarity to those facts. That split is deliberate —
// under RED_MODE=1 the "default does not resolve" outcome is the EXPECTED
// one, so a probe that failed the test on it could not express both polarities
// from one source.

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type gateProbeMsg struct {
	Other string `yaml:"other"`
}

func TestGateProbeToolschemaI18nDefault(t *testing.T) {
	bundlePath := os.Getenv("GATE_PROBE_BUNDLE")
	plainID := os.Getenv("GATE_PROBE_PLAIN_ID")
	tmplID := os.Getenv("GATE_PROBE_TMPL_ID")

	raw, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("GATEPROBE FATAL cannot read bundle %q: %v", bundlePath, err)
	}
	var bundle map[string]gateProbeMsg
	if err := yaml.Unmarshal(raw, &bundle); err != nil {
		t.Fatalf("GATEPROBE FATAL cannot parse bundle %q: %v", bundlePath, err)
	}

	// Ground truth from the on-disk bundle. If the ID is absent or its text
	// equals the ID itself, every downstream comparison would be vacuous, so
	// refuse to render a verdict at all.
	plainWant := bundle[plainID].Other
	if plainWant == "" || plainWant == plainID {
		t.Fatalf("GATEPROBE FATAL bundle ground truth for %q is empty or self-echoing (%q) — "+
			"the probe would pass vacuously", plainID, plainWant)
	}
	tmplRaw := bundle[tmplID].Other
	if !strings.Contains(tmplRaw, "{{") {
		t.Fatalf("GATEPROBE FATAL bundle entry for %q (%q) has no {{.Placeholder}} — "+
			"the interpolation check would be vacuous", tmplID, tmplRaw)
	}

	// R1 — ZERO WIRING. This is the first line in this process that touches
	// the i18n seam, and `-run` selects only this test, so it observes the
	// UNTOUCHED package default exactly as an unwired consumer would.
	gotDefault := tr(plainID, nil)
	defaultResolves := gotDefault == plainWant
	fmt.Printf("GATEPROBE r1_default id=%s got=%q want=%q\n", plainID, gotDefault, plainWant)

	// R2 — placeholder interpolation through that same default.
	const probeArg = "/gate/probe/path"
	gotTmpl := tr(tmplID, map[string]any{"Path": probeArg})
	interpOK := gotTmpl != tmplID && gotTmpl != tmplRaw && strings.Contains(gotTmpl, probeArg)
	fmt.Printf("GATEPROBE r2_interp id=%s got=%q template=%q\n", tmplID, gotTmpl, tmplRaw)

	// R3 — NEGATIVE CONTROL. Explicitly opting into NoopTranslator must still
	// yield the RAW message ID. This reproduces the historical defect's symptom
	// in-process and proves this probe can tell resolved text from a raw echo.
	SetTranslator(NoopTranslator{})
	gotNoop := tr(plainID, nil)
	noopEchoes := gotNoop == plainID
	fmt.Printf("GATEPROBE r3_noop id=%s got=%q\n", plainID, gotNoop)

	// R4 — the documented restore path must bring the resolving default back.
	SetTranslator(nil)
	gotRestored := tr(plainID, nil)
	restoreOK := gotRestored == plainWant
	fmt.Printf("GATEPROBE r4_restore id=%s got=%q want=%q\n", plainID, gotRestored, plainWant)

	yn := func(b bool) string {
		if b {
			return "yes"
		}
		return "no"
	}
	fmt.Printf("GATEPROBE RESULT default_resolves=%s interpolation_ok=%s noop_echoes=%s restore_ok=%s\n",
		yn(defaultResolves), yn(interpOK), yn(noopEchoes), yn(restoreOK))
}
GOEOF

# Overlay maps a path that does NOT exist in the module onto our temp source,
# so the package builds WITH the probe while the tree stays byte-identical.
OVERLAY_TARGET="$MODULE_DIR/zz_gate_probe_injected_test.go"
printf '{"Replace":{"%s":"%s"}}\n' "$OVERLAY_TARGET" "$PROBE_DIR/probe_test.go" > "$PROBE_DIR/overlay.json"

# Snapshot the tree BEFORE the probe so the post-check can attribute dirt to
# the probe rather than to whatever the working tree already had. Comparing
# against "empty" instead of against this baseline was a real defect in the
# first draft of this gate: it made the gate refuse whenever ANY edit to the
# guarded source was in flight — including its own §1.1 paired mutation, which
# short-circuited before the verdict and left the gate unprovable. A guard must
# assert the REAL condition it claims to (§11.4.201); a pre-existing dirty tree
# is legitimate developer state, not probe residue.
TREE_BEFORE="$(cd "$MODULE_DIR" && git status --porcelain 2>/dev/null)"

PROBE_OUT="$PROBE_DIR/probe.out"
(
    cd "$MODULE_DIR" || exit 97
    GATE_PROBE_BUNDLE="$MODULE_DIR/$BUNDLE_REL" \
    GATE_PROBE_PLAIN_ID="$PLAIN_ID" \
    GATE_PROBE_TMPL_ID="$TMPL_ID" \
    go test -count=1 -overlay="$PROBE_DIR/overlay.json" \
        -run '^TestGateProbeToolschemaI18nDefault$' -v .
) > "$PROBE_OUT" 2>&1
probe_rc=$?

# Tree-delta assertion: the overlay must not have ADDED anything to the
# submodule tree. Attribution is by before/after delta, never by absolute
# cleanliness (see the TREE_BEFORE comment above).
TREE_AFTER="$(cd "$MODULE_DIR" && git status --porcelain 2>/dev/null)"
if [[ "$TREE_AFTER" != "$TREE_BEFORE" ]]; then
    echo "FAIL: the probe CHANGED the submodule working tree — §11.4.84 residue risk." >&2
    echo "      before:" >&2; printf '%s\n' "${TREE_BEFORE:-<clean>}" | sed 's/^/        /' >&2
    echo "      after :" >&2; printf '%s\n' "${TREE_AFTER:-<clean>}"  | sed 's/^/        /' >&2
    exit 1
fi
if [[ -z "$TREE_AFTER" ]]; then
    echo "  tree   : submodules/tool_schema byte-clean, unchanged by the probe (porcelain empty)"
else
    echo "  tree   : submodules/tool_schema had pre-existing local edits; the probe added none"
    printf '%s\n' "$TREE_AFTER" | sed 's/^/           /'
fi

if [[ "$probe_rc" -ne 0 ]]; then
    echo "SKIP(env): the runtime probe did not execute cleanly (go test rc=$probe_rc)." >&2
    echo "           This certifies NOTHING about the invariant. Probe output:" >&2
    tail -20 "$PROBE_OUT" >&2
    exit 2
fi

# `go test` exits 0 when NO test matches the -run filter, so the RESULT line is
# the proof the probe actually EXECUTED. Its absence is an environment failure,
# never a pass.
result_line="$(grep -m1 '^GATEPROBE RESULT ' "$PROBE_OUT")"
if [[ -z "$result_line" ]]; then
    echo "SKIP(env): probe produced no GATEPROBE RESULT line — it did not run." >&2
    tail -20 "$PROBE_OUT" >&2
    exit 2
fi

grep '^GATEPROBE ' "$PROBE_OUT" | sed 's/^/    /'

get_fact() { printf '%s\n' "$result_line" | grep -oE "$1=[a-z]+" | cut -d= -f2; }
default_resolves="$(get_fact default_resolves)"
interpolation_ok="$(get_fact interpolation_ok)"
noop_echoes="$(get_fact noop_echoes)"
restore_ok="$(get_fact restore_ok)"

# --- Falsifiability precondition (both polarities) --------------------------
# If NoopTranslator stops echoing, this guard can no longer distinguish
# resolved text from a raw ID, and every verdict it renders is worthless.
if [[ "$noop_echoes" != "yes" ]]; then
    echo "FAIL: the NEGATIVE CONTROL is dead — SetTranslator(NoopTranslator{}) no longer echoes the" >&2
    echo "      raw message ID, so this guard cannot distinguish resolved text from a raw echo." >&2
    echo "      An unfalsifiable guard is a §11.4 bluff gate. Fix the seam or reconcile per §11.4.120." >&2
    exit 1
fi

# --- Verdict ----------------------------------------------------------------
if [[ "$RED_MODE" == "1" ]]; then
    if [[ "$default_resolves" == "no" ]]; then
        echo "RED PASS — defect reproduced on the current artifact: the zero-wired package default"
        echo "           does NOT resolve real bundle text, so every tr() site echoes the raw"
        echo "           message ID into user-visible output."
        exit 0
    fi
    echo "RED FAIL — the zero-wired default DOES resolve real bundle text, so the defect is no" >&2
    echo "           longer reproducible: the fix is in. This is the EXPECTED result on a fixed" >&2
    echo "           tree. Run with RED_MODE=0 (the default) for the standing GREEN guard." >&2
    exit 1
fi

green_fail=0
if [[ "$default_resolves" != "yes" ]]; then
    echo "GREEN FAIL — the zero-wired package default does NOT resolve real bundle text. Every" >&2
    echo "             tr() site (incl. user-visible ToolResult.Error payloads) emits the raw" >&2
    echo "             message ID to end users." >&2
    if [[ "$src_default_is_noop" -eq 1 ]]; then
        echo "             Source confirms the exact historical regression: i18n.go sets" >&2
        echo "             activeTr Translator = NoopTranslator{}." >&2
    fi
    green_fail=1
fi
if [[ "$interpolation_ok" != "yes" ]]; then
    echo "GREEN FAIL — placeholder interpolation through the default translator is broken:" >&2
    echo "             a {{.Placeholder}} message did not render its argument." >&2
    green_fail=1
fi
if [[ "$restore_ok" != "yes" ]]; then
    echo "GREEN FAIL — SetTranslator(nil) no longer restores the resolving default, so a consumer" >&2
    echo "             that temporarily overrides the seam cannot get real text back." >&2
    green_fail=1
fi
if [[ "$src_default_is_noop" -eq 1 && "$green_fail" -eq 0 ]]; then
    echo "GREEN FAIL — source and runtime DISAGREE: i18n.go declares the default as" >&2
    echo "             NoopTranslator{} yet the runtime resolved real text. One of the two" >&2
    echo "             signals is lying; investigate per §11.4.102 before trusting either." >&2
    green_fail=1
fi
if [[ "$green_fail" -ne 0 ]]; then
    exit 1
fi

echo "GREEN PASS (RUNTIME EVIDENCE) — the zero-wired package default resolves real bundle text"
echo "           matching $BUNDLE_REL, placeholder interpolation renders, SetTranslator(nil)"
echo "           restores that default, and NoopTranslator still echoes verbatim (negative"
echo "           control alive). Probed in a fresh process; submodule tree left byte-clean."
exit 0
