#!/usr/bin/env bash
# scripts/gates/qa_transcript_redaction_gate.sh
#
# CM-QA-TRANSCRIPT-REDACTION-FAIL-CLOSED — permanent regression guard for
# HXC-167 (§11.4.135 standing regression-guard suite).
#
# ---------------------------------------------------------------------------
# WHAT DEFECT THIS GUARDS
# ---------------------------------------------------------------------------
# scripts/qa/lib/sec_capture_lib.sh scrubs secrets out of the QA transcripts
# committed under docs/qa/. Before the HXC-167 fix that scrubbing had two
# weaknesses, and the second one is a §11.4.201 false-null living inside the
# control that protects every other capture:
#
#   HALF 1 — allowlist-only.
#       Only secrets the capture script itself registered via qa_redact were
#       removed. Anything the SERVER sent back — Set-Cookie, a refresh or
#       session token, a pre-signed URL — was unknown to the scrubber and
#       written verbatim. Observable proof already in the tree: expired
#       Cloudflare `__cf_bm` cookies committed verbatim in
#       docs/qa/feat_cerebras_models_f8c38181_20260728T120731Z/transcripts/.
#
#   HALF 2 — fail-open, silently.  (the serious one)
#       The redaction was:
#           sed -i "s|${p//|/\\|}|<EPHEMERAL_KEY_REDACTED>|g" "$f" 2>/dev/null || true
#       The secret was interpolated INTO A REGEX and only `|` was escaped. A
#       secret containing any other metacharacter — canonically an unbalanced
#       `[` — made sed abort ("unterminated `s' command"), and
#       `2>/dev/null || true` discarded the message AND the exit status. The
#       secret survived in the transcript with no signal whatsoever.
#       (Escaping `|` as `\|` was independently wrong: under GNU BRE `\|` is
#       the ALTERNATION operator, not an escaped literal.)
#
# ---------------------------------------------------------------------------
# POLARITY (§11.4.115 — one source, two roles)
# ---------------------------------------------------------------------------
#   RED_MODE=1  Reproduce the defect on the PRE-FIX artifact and assert it is
#               PRESENT. The pre-fix implementation is not hand-copied: it is
#               recovered from git history (the parent of the commit that
#               removed the vulnerable line). If history is unavailable the
#               gate SKIPs with reason (§11.4.3) — it never silently passes.
#
#   RED_MODE=0  (DEFAULT) Standing GREEN regression guard: run the CURRENT
#               library and assert the defect is ABSENT — the metacharacter
#               secret is removed, failure is loud, and a server-originated
#               secret the script never registered is caught by the post-write
#               scan.
#
# All fixture secrets are SYNTHETIC — obviously fake, structurally realistic.
# No real credential is present in this file or produced by it.
#
# Usage:
#   scripts/gates/qa_transcript_redaction_gate.sh              # GREEN guard
#   RED_MODE=1 scripts/gates/qa_transcript_redaction_gate.sh   # reproduce defect
#
# Exit codes:
#   0  gate satisfied for the active RED_MODE
#   1  gate violated for the active RED_MODE
#   2  environment/SKIP — certifies nothing (§11.4.3)
#
# Cross-references: §11.4.1 / §11.4.10 / §11.4.74 / §11.4.107(10) / §11.4.115 /
#   §11.4.135 / §11.4.201; scripts/qa/lib/sec_capture_lib.sh;
#   scripts/secret_scan.sh (reused as the shared key-shape layer).

set -uo pipefail

RED_MODE="${RED_MODE:-0}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
LIB="$ROOT/scripts/qa/lib/sec_capture_lib.sh"

# The vulnerable line's fingerprint — used both to locate the pre-fix commit
# and to assert the current source no longer contains it.
VULN_FINGERPRINT='sed -i "s|${p//'

# --- Synthetic fixtures (§11.4.10: never a real value) ----------------------
# Deliberately hyphenated so the literal in THIS file does not match
# scripts/secret_scan.sh's `sk-[A-Za-z0-9]{20,}` provider shape. That scanner
# runs in the pre-commit hook and was correct to flag an earlier, un-hyphenated
# version of this fixture — the right response is to stop the test data looking
# like a live key, never to weaken or allowlist the scanner. The one fixture
# that genuinely MUST match a provider shape (the Layer 3 check) is assembled
# at runtime below, so no key-shaped literal exists in this source at all.
# A key with NO regex metacharacter — the case the old code handled.
FIXTURE_PLAIN='sk-FAKE-hxc167-plain-AAAA-BBBB-CCCC-DDDD'
# THE DECISIVE CASE: a key containing an unbalanced '[' — a legal character in
# many providers' key alphabets, and the one that silently broke sed.
FIXTURE_META='sk-FAKEHXC167meta[unbalancedBBBBBBBBBBBBBB'
# A SERVER-ORIGINATED secret the capture script can never have registered.
FIXTURE_SERVER_COOKIE='__cf_bm=FAKEhxc167ServerSentCookieValueCCCCCCCCCCCCCCCC'
# The marker the library replaces secrets with (kept in sync with
# QA_REDACT_TOKEN in scripts/qa/lib/sec_capture_lib.sh; asserted below).
QA_REDACT_TOKEN_EXPECTED='<EPHEMERAL_KEY_REDACTED>'

TMP="$(mktemp -d)"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

fail() { echo "GATE FAIL — $*" >&2; exit 1; }
skip() { echo "GATE SKIP (§11.4.3) — $*" >&2; exit 2; }

# _contains_vuln <file>
# True when <file> carries the vulnerable construct in EXECUTABLE (non-comment)
# form. Comments are stripped first because the fix DOCUMENTS the construct it
# replaced, so a raw match would trip on the fix's own forensic record.
#
# Deliberately NOT `sed ... | grep -qF`. That pipeline silently returns a FALSE
# NEGATIVE here, and it took a measurement to see it: `grep -q` exits the moment
# it matches, which SIGPIPEs the upstream `sed`, and under `set -o pipefail` the
# pipeline then reports sed's 141 instead of grep's 0 — so a REAL match reads as
# "no match". Measured directly on the pre-fix blob: `grep -qF` -> 141 while
# `grep -cF` -> 0 with count 1. It hid because the interactive shell this was
# first tested in resolves `grep` to a ugrep shim that drains its input, while
# the gate runs as a script where `grep` is GNU grep and does not. That is the
# same defect class this whole gate exists to catch — a check that reports
# success while checking nothing — so it is closed with a construct that has no
# pipeline, no external tool, and no exit-status subtlety at all.
_contains_vuln() {
    local stripped
    stripped="$(sed -E 's/^[[:space:]]*#.*$//' "$1" 2>/dev/null)" || return 1
    case "$stripped" in
        *"$VULN_FINGERPRINT"*) return 0 ;;
    esac
    return 1
}

# _qa_token_is_inert
# True when QA_REDACT_TOKEN cannot be matched as a credential VALUE by any row
# of the library's own scan table — the invariant the scrub loop's termination
# depends on. Sources the library to read the real table rather than restating
# it, so the two cannot drift.
_qa_token_is_inert() {
    ( set +u
      # shellcheck source=/dev/null
      source "$LIB" >/dev/null 2>&1 || exit 1
      probe="< set-cookie: sid=${QA_REDACT_TOKEN}; Path=/"
      probe2="{\"access_token\":\"${QA_REDACT_TOKEN}\"}"
      while IFS=$'\t' read -r _label _ctx valre _mode; do
          [ -n "${_label:-}" ] || continue
          for line in "$probe" "$probe2"; do
              while IFS= read -r m; do
                  [ -n "$m" ] || continue
                  v="$(_qa_strip_value "$_mode" "$m")"
                  case "$v" in
                      "$QA_REDACT_TOKEN"|*"$QA_REDACT_TOKEN"*) continue ;;
                  esac
                  [ -n "$v" ] && exit 1
              done < <(printf '%s\n' "$line" | grep -oE -e "$valre" 2>/dev/null || true)
          done
      done < <(_qa_scan_patterns)
      exit 0 )
}

[ -f "$LIB" ] || fail "library not found: $LIB"

# ===========================================================================
# RED_MODE=1 — reproduce the defect on the genuine PRE-FIX artifact
# ===========================================================================
if [ "$RED_MODE" = "1" ]; then
    echo "CM-QA-TRANSCRIPT-REDACTION-FAIL-CLOSED  RED_MODE=1 (reproduce on pre-fix artifact)"

    command -v git >/dev/null 2>&1 || skip "git not available; cannot recover the pre-fix artifact"
    git -C "$ROOT" rev-parse --git-dir >/dev/null 2>&1 || skip "not a git repository"

    # Find the NEWEST revision of the library whose EXECUTABLE (non-comment)
    # source still contains the vulnerable construct — that is the genuine
    # pre-fix artifact.
    #
    # `git log -S` is deliberately NOT used to locate it. The fix documents the
    # construct it replaced, so the string's occurrence COUNT is unchanged
    # across the fix commit (one executable line became one comment line) and
    # -S — which reports only count changes — does not see the fix at all. An
    # earlier version of this gate relied on -S and consequently printed
    # "fix not yet landed" AFTER the fix had landed: a true RED result under a
    # false caption. Stripping comments before matching is what makes the
    # search agree with reality.
    prefix_lib=""
    prefix_commit=""
    while IFS= read -r rev; do
        [ -n "$rev" ] || continue
        git -C "$ROOT" show "${rev}:scripts/qa/lib/sec_capture_lib.sh" \
            > "$TMP/cand.sh" 2>/dev/null || continue
        if _contains_vuln "$TMP/cand.sh"; then
            cp "$TMP/cand.sh" "$TMP/prefix_lib.sh"
            prefix_lib="$TMP/prefix_lib.sh"
            prefix_commit="$rev"
            break
        fi
    done < <(git -C "$ROOT" log --format=%H -- scripts/qa/lib/sec_capture_lib.sh 2>/dev/null || true)

    [ -n "$prefix_lib" ] || skip "no revision of the library carries the vulnerable construct in executable form"

    head_rev="$(git -C "$ROOT" rev-parse HEAD 2>/dev/null || echo '')"
    if [ "$prefix_commit" = "$head_rev" ]; then
        echo "  pre-fix artifact: ${prefix_commit} (== HEAD — the fix has NOT landed yet)"
    else
        echo "  pre-fix artifact: ${prefix_commit} (fix landed in a later commit)"
    fi

    # Extract ONLY the pre-fix redaction pair from that artifact and drive it.
    # Sourcing the whole pre-fix library would execute unrelated setup; we want
    # exactly the defective code path, taken verbatim from the artifact.
    awk '/^qa_redact\(\)/,/^}/' "$prefix_lib"          >  "$TMP/prefix_impl.sh"
    awk '/^_qa_apply_redaction\(\)/,/^}/' "$prefix_lib" >> "$TMP/prefix_impl.sh"
    grep -qF -- "$VULN_FINGERPRINT" "$TMP/prefix_impl.sh" \
        || skip "extracted pre-fix functions do not contain the vulnerable line"

    # shellcheck disable=SC1090
    ( set +u
      QA_REDACT_PATTERNS=()
      source "$TMP/prefix_impl.sh"

      printf 'Authorization: Bearer %s\n' "$FIXTURE_PLAIN" > "$TMP/red_plain.txt"
      printf 'Authorization: Bearer %s\n' "$FIXTURE_META"  > "$TMP/red_meta.txt"
      printf '< set-cookie: %s; Path=/\n' "$FIXTURE_SERVER_COOKIE" > "$TMP/red_server.txt"

      QA_REDACT_PATTERNS=(); qa_redact "$FIXTURE_PLAIN";  _qa_apply_redaction "$TMP/red_plain.txt"
      QA_REDACT_PATTERNS=(); qa_redact "$FIXTURE_META";   _qa_apply_redaction "$TMP/red_meta.txt"
      # Server-originated: nothing to register — that IS half 1 of the defect.
      QA_REDACT_PATTERNS=();                              _qa_apply_redaction "$TMP/red_server.txt"
    ) >/dev/null 2>&1

    rc_defects=0

    # HALF 2 (decisive): the metacharacter secret must SURVIVE on the pre-fix
    # artifact, and the control must have stayed silent about it.
    if grep -qF -- "$FIXTURE_META" "$TMP/red_meta.txt"; then
        echo "  [reproduced] metacharacter secret SURVIVED redaction (fail-open, no signal)"
        rc_defects=$((rc_defects+1))
    else
        echo "  [NOT reproduced] metacharacter secret was removed"
    fi

    # Control: the plain secret WAS handled — proves the fixture drives a
    # working code path and the survival above is the metacharacter, not a
    # broken harness (§11.4.107(10) negative control).
    if grep -qF -- "$FIXTURE_PLAIN" "$TMP/red_plain.txt"; then
        echo "  [control BROKEN] plain secret also survived — harness is not exercising redaction"
        fail "RED negative control failed: the pre-fix path did not redact even a plain secret"
    fi
    echo "  [control OK] plain secret WAS redacted by the same pre-fix path"

    # HALF 1: the server-originated secret must be written verbatim.
    if grep -qF -- "$FIXTURE_SERVER_COOKIE" "$TMP/red_server.txt"; then
        echo "  [reproduced] server-originated Set-Cookie written VERBATIM (allowlist-only)"
        rc_defects=$((rc_defects+1))
    else
        echo "  [NOT reproduced] server-originated Set-Cookie was removed"
    fi

    if [ "$rc_defects" -eq 2 ]; then
        echo
        echo "RED PASS — both halves of HXC-167 reproduced on the pre-fix artifact."
        echo "           Flip to RED_MODE=0 (the default) for the standing GREEN guard."
        exit 0
    fi
    fail "RED expected to reproduce 2 defect halves, reproduced ${rc_defects}"
fi

# ===========================================================================
# RED_MODE=0 — standing GREEN regression guard against the CURRENT library
# ===========================================================================
echo "CM-QA-TRANSCRIPT-REDACTION-FAIL-CLOSED  RED_MODE=0 (standing GREEN guard)"
green_fail_early=0

# (0) The vulnerable construct must be gone from the EXECUTABLE source.
# Comment lines are excluded deliberately: the fix documents the construct it
# replaced (that documentation is the forensic record and must not be what the
# gate trips on). Only a live, non-comment occurrence is a regression.
if _contains_vuln "$LIB"; then
    fail "the fail-open sed redaction is STILL PRESENT (non-comment) in $LIB"
fi
echo "  [ok] fail-open sed construct absent from the library's executable source"

# (0b) The marker this gate asserts on must be the marker the library uses.
# Without this the gate could silently drift and test a token nothing produces.
# shellcheck disable=SC1090  # $LIB is resolved from this script's location at runtime
lib_token="$( set +u; source "$LIB" >/dev/null 2>&1; printf '%s' "${QA_REDACT_TOKEN:-}" )"
if [ "$lib_token" != "$QA_REDACT_TOKEN_EXPECTED" ]; then
    fail "redaction marker drift: library uses '${lib_token}', gate asserts '${QA_REDACT_TOKEN_EXPECTED}'"
fi
echo "  [ok] gate and library agree on the redaction marker"

# (0c) PAIRED MUTATION for check (0) — §1.1 / §11.4.107(10).
# Check (0) is itself a gate, so it needs its own falsifiability proof. Without
# this, check (0) could silently never fire and nobody would notice — which is
# exactly what happened: while it was written as
#   sed -E 's/#.*//' "$LIB" | grep -qF -- "$FINGERPRINT"
# `grep -q` exited on first match, SIGPIPEd `sed`, and under `set -o pipefail`
# the pipeline returned sed's 141 — so a PRESENT construct read as absent and
# check (0) printed "[ok] absent" against a fully reverted library. It was
# caught only because the BEHAVIOURAL checks failed. A check with no mutation
# proving it can fail is decorative, so here is that mutation: plant the
# construct in EXECUTABLE form and require the detector to see it.
mutant="$TMP/mutant_lib.sh"
{
    echo '#!/usr/bin/env bash'
    echo '# a comment quoting the construct must NOT trip the detector:'
    echo "#     sed -i \"s|\${p//|/\\\\|}|<EPHEMERAL_KEY_REDACTED>|g\" \"\$f\" 2>/dev/null || true"
    echo '_qa_apply_redaction() {'
    echo '    local f="$1" p'
    echo '    for p in "${QA_REDACT_PATTERNS[@]:-}"; do'
    echo "        sed -i \"s|\${p//|/\\\\|}|<EPHEMERAL_KEY_REDACTED>|g\" \"\$f\" 2>/dev/null || true"
    echo '    done'
    echo '}'
} > "$mutant"
if _contains_vuln "$mutant"; then
    echo "  [ok] check (0) detector FIRES on a planted executable construct (paired mutation)"
else
    echo "  [FAIL] check (0) detector is BLIND — a reverted library would pass unnoticed"
    green_fail_early=1
fi
# Negative control: a file where the construct appears ONLY in a comment must
# NOT trip it, or check (0) would fail on the fix's own forensic record.
comment_only="$TMP/comment_only.sh"
{
    echo '#!/usr/bin/env bash'
    echo "#     sed -i \"s|\${p//|/\\\\|}|<EPHEMERAL_KEY_REDACTED>|g\" \"\$f\" 2>/dev/null || true"
    echo 'true'
} > "$comment_only"
if _contains_vuln "$comment_only"; then
    echo "  [FAIL] check (0) detector trips on a COMMENT-only occurrence (false positive)"
    green_fail_early=1
else
    echo "  [ok] check (0) detector ignores comment-only occurrences (negative control)"
fi

# (0d) TERMINATION INVARIANT — the redaction marker must match NO scan pattern.
# The auto-redact/re-scan loop terminates only because a redacted value can
# never re-match. That rests entirely on QA_REDACT_TOKEN starting with '<',
# which no value character class admits. It is load-bearing and was untested:
# changing the marker to a bare word makes an ordinary Set-Cookie capture loop
# and then abort every time. Asserted here so the dependency cannot be broken
# silently by an innocent-looking rename.
if _qa_token_is_inert; then
    echo "  [ok] redaction marker matches no scan pattern (loop termination safe)"
else
    echo "  [FAIL] redaction marker itself matches a scan pattern — scrub loop cannot converge"
    green_fail_early=1
fi

# Drive the REAL library. qa_init is not called (it would create a docs/qa run
# directory); the scrub helpers are self-contained and operate on a given file.
run_scrub() {
    # $1 = target file, remaining args = secrets to register.
    local target="$1"; shift
    local s
    (
        set +u
        # SC1090: $LIB is resolved from this script's own location at runtime, so
        # it cannot be a literal path here — following it statically is exactly
        # what shellcheck cannot do and is not a defect.
        # SC2034: QA_REPO_ROOT and QA_REDACT_PATTERNS look unused to shellcheck
        # because their only consumers live in the dynamically-sourced library
        # ($LIB) that SC1090 just told us it cannot follow. Both are genuinely
        # read there: QA_REPO_ROOT by _qa_apply_redaction's Layer 3, and
        # QA_REDACT_PATTERNS by qa_redact/_qa_apply_redaction's Layer 1.
        # shellcheck disable=SC1090
        source "$LIB"
        # shellcheck disable=SC2034
        QA_REPO_ROOT="$ROOT"
        # shellcheck disable=SC2034
        QA_REDACT_PATTERNS=()
        for s in "$@"; do qa_redact "$s"; done
        _qa_apply_redaction "$target"
    )
}

green_fail=${green_fail_early:-0}

# --- (1) THE DECISIVE CASE: metacharacter secret must be removed ------------
printf 'Authorization: Bearer %s\n' "$FIXTURE_META" > "$TMP/g_meta.txt"
run_scrub "$TMP/g_meta.txt" "$FIXTURE_META" > "$TMP/g_meta.log" 2>&1
rc=$?
if [ ! -f "$TMP/g_meta.txt" ]; then
    echo "  [ok] metacharacter secret: file quarantined and run aborted (rc=$rc) — fail-CLOSED"
elif grep -qF -- "$FIXTURE_META" "$TMP/g_meta.txt"; then
    echo "  [FAIL] metacharacter secret SURVIVED redaction — HXC-167 has regressed"
    green_fail=1
else
    echo "  [ok] metacharacter secret removed (literal replacement, no regex)"
fi

# --- (1b) METACHARACTER BATTERY (§11.4.146 STEP 3 — extend to all cases) ---
# The single `[` case above proves the defect class; this battery pins the
# whole class so a future "optimisation" back to a pattern-based substitution
# cannot pass by handling only the one character the ticket happened to name.
# Every entry must be REMOVED (rc=0), or ABORT loudly — never survive at rc=0.
meta_battery=(
    'sk-FAKEbatch]closeBBBBBBBBBBBBBBBBBBBB'
    'sk-FAKEbatch.*+?meta^$CCCCCCCCCCCCCCCC'
    'sk-FAKEbatch\backDDDDDDDDDDDDDDDDDDDDD'
    'sk-FAKEbatch\\dblEEEEEEEEEEEEEEEEEEEEE'
    'sk-FAKEbatch&ampFFFFFFFFFFFFFFFFFFFFFF'
    'sk-FAKEbatch|pipeGGGGGGGGGGGGGGGGGGGGG'
    'sk-FAKEbatch/slash/HHHHHHHHHHHHHHHHHHH'
    'sk-FAKEbatch(){}braceIIIIIIIIIIIIIIIII'
)
battery_bad=0
for secret in "${meta_battery[@]}"; do
    bf="$TMP/g_batt.txt"
    printf 'Authorization: Bearer %s\n' "$secret" > "$bf"
    run_scrub "$bf" "$secret" >/dev/null 2>&1
    brc=$?
    if [ -f "$bf" ] && grep -qF -- "$secret" "$bf" && [ "$brc" -eq 0 ]; then
        echo "  [FAIL] metacharacter battery: a secret survived at rc=0 (value withheld)"
        battery_bad=1
    fi
done
if [ "$battery_bad" -eq 0 ]; then
    echo "  [ok] metacharacter battery: all ${#meta_battery[@]} variants removed or aborted loudly"
else
    green_fail=1
fi

# --- (1c) degenerate no-op must NOT abort (anti cry-wolf, §11.4.201) -------
# Registering the redaction marker itself is meaningless but harmless; it must
# not take down a healthy run.
printf 'x: %s\n' "$QA_REDACT_TOKEN_EXPECTED" > "$TMP/g_degen.txt"
run_scrub "$TMP/g_degen.txt" "$QA_REDACT_TOKEN_EXPECTED" >/dev/null 2>&1
if [ "$?" -ne 0 ]; then
    echo "  [FAIL] registering the redaction marker aborted a healthy run (cry-wolf)"
    green_fail=1
else
    echo "  [ok] degenerate no-op (marker registered as secret) does not abort"
fi

# --- (2) plain secret still removed (no regression in the ordinary case) ----
printf 'Authorization: Bearer %s\n' "$FIXTURE_PLAIN" > "$TMP/g_plain.txt"
run_scrub "$TMP/g_plain.txt" "$FIXTURE_PLAIN" >/dev/null 2>&1
if [ -f "$TMP/g_plain.txt" ] && grep -qF -- "$FIXTURE_PLAIN" "$TMP/g_plain.txt"; then
    echo "  [FAIL] plain secret survived redaction"
    green_fail=1
else
    echo "  [ok] plain secret removed"
fi

# --- (3) HALF 1: server-originated secret caught with NOTHING registered ----
printf '< set-cookie: %s; Path=/; Domain=api.example.test\n' "$FIXTURE_SERVER_COOKIE" \
    > "$TMP/g_server.txt"
run_scrub "$TMP/g_server.txt" >/dev/null 2>&1
if [ ! -f "$TMP/g_server.txt" ]; then
    echo "  [ok] server-originated Set-Cookie: file quarantined — fail-CLOSED"
elif grep -qF -- "$FIXTURE_SERVER_COOKIE" "$TMP/g_server.txt"; then
    echo "  [FAIL] server-originated Set-Cookie written VERBATIM — post-write scan did not catch it"
    green_fail=1
else
    echo "  [ok] server-originated Set-Cookie neutralised by the post-write scan"
fi

# --- (3b) LAYER 3: a real provider KEY SHAPE the script never registered ---
# Assembled at runtime from fragments so this file contains no key-shaped
# literal for scripts/secret_scan.sh to flag (see the fixture note at the top).
# Nothing is registered via qa_redact, so Layer 1 cannot help and Layer 2 has
# no credential-bearing header to anchor on: only the shared key-shape scanner
# can catch this, which is exactly what the check proves is wired.
l3_secret="sk-$(printf 'FAKEl3hxc167')$(printf 'K%.0s' $(seq 1 24))"
printf '< HTTP/1.1 200 OK\n' > "$TMP/g_l3.txt"
printf -- '--- RESPONSE BODY (48 bytes) ---\n{"note":"%s"}\n' "$l3_secret" >> "$TMP/g_l3.txt"
printf -- '--- curl exit code: 0 ---\n' >> "$TMP/g_l3.txt"
run_scrub "$TMP/g_l3.txt" > "$TMP/g_l3.log" 2>&1
l3_rc=$?
if [ ! -f "$TMP/g_l3.txt" ]; then
    echo "  [ok] unregistered provider-shaped key: file quarantined (rc=$l3_rc) — Layer 3 wired"
elif grep -qF -- "$l3_secret" "$TMP/g_l3.txt" && [ "$l3_rc" -eq 0 ]; then
    echo "  [FAIL] unregistered provider-shaped key survived at rc=0 — Layer 3 not wired"
    green_fail=1
else
    echo "  [ok] unregistered provider-shaped key neutralised (rc=$l3_rc)"
fi
if grep -qF -- "$l3_secret" "$TMP/g_l3.log" 2>/dev/null; then
    echo "  [FAIL] Layer 3 failure output printed the secret VALUE (§11.4.10)"
    green_fail=1
fi

# --- (3c) TWO credentials on ONE line — both must go ------------------------
# Regression guard for a fail-open found during development of this very fix:
# the extract expressions lead with a greedy `.*`, so extracting per-LINE
# returned only the LAST credential; and because redacting it put the marker on
# that line, a line-level allowlist then hid the survivor and the run exited 0.
# Two independent ways to leak, both reported as success. Pinned here so no
# future "simplification" can reintroduce either.
cat > "$TMP/g_multi.txt" <<'MULTI'
< HTTP/1.1 200 OK
--- RESPONSE BODY (220 bytes) ---
{"access_token":"FAKEmultiVALUEaaaaaaaaaaaaaaaa","refresh_token":"FAKEmultiVALUEbbbbbbbbbbbbbbbb"}
{"url":"https://s3.test/o?X-Amz-Signature=FAKEmultiVALUEcccccccccccccccc&X-Amz-Credential=FAKEmultiVALUEdddddddddddddddd"}
--- curl exit code: 0 ---
MULTI
run_scrub "$TMP/g_multi.txt" >/dev/null 2>&1
multi_rc=$?
if [ ! -f "$TMP/g_multi.txt" ]; then
    echo "  [ok] two-per-line credentials: file quarantined (rc=$multi_rc) — fail-CLOSED"
else
    multi_left="$(grep -oE 'FAKEmultiVALUE[a-z]*' "$TMP/g_multi.txt" 2>/dev/null | sort -u | wc -l)"
    if [ "$multi_left" -gt 0 ] && [ "$multi_rc" -eq 0 ]; then
        echo "  [FAIL] ${multi_left} of 4 same-line credentials survived at rc=0 — FAIL-OPEN"
        green_fail=1
    else
        echo "  [ok] all 4 same-line credentials removed (greedy-extract + allowlist-masking closed)"
    fi
fi

# --- (3d) the shared allowlist must NOT exempt transcripts ------------------
# Layer 3 delegates to scripts/secret_scan.sh, which honours the repo-root
# .scan-secrets-allow path globs. If a future entry there ever matched
# docs/qa/**, *.http, or a transcripts/ directory, Layer 3 would silently skip
# the very files this control exists to protect — the scan would still "pass"
# on every run while checking nothing. That is a decorative-gate failure, so it
# is asserted rather than assumed. Ground truth, not a reimplementation of the
# glob logic: plant a synthetic key at a REAL docs/qa transcript path and
# require the scanner to catch it.
allow_probe_dir="$ROOT/docs/qa/_hxc167_allowlist_probe_$$/transcripts"
cleanup_probe() { rm -rf "$ROOT/docs/qa/_hxc167_allowlist_probe_$$"; }
trap 'cleanup_probe; cleanup' EXIT
if mkdir -p "$allow_probe_dir" 2>/dev/null; then
    probe_key="sk-$(printf 'FAKEallowprobe')$(printf 'Z%.0s' $(seq 1 24))"
    printf '{"k":"%s"}\n' "$probe_key" > "$allow_probe_dir/probe.http"
    if "$ROOT/scripts/secret_scan.sh" "$allow_probe_dir/probe.http" >/dev/null 2>&1; then
        echo "  [FAIL] .scan-secrets-allow EXEMPTS docs/qa transcripts — Layer 3 is decorative"
        green_fail=1
    else
        echo "  [ok] .scan-secrets-allow does not exempt docs/qa transcripts — Layer 3 real"
    fi
    cleanup_probe
else
    echo "  [warn] could not create allowlist probe dir; Layer 3 exemption NOT verified"
fi

# --- (4) fail LOUD: an unscrubbable secret must abort, not continue ---------
# A genuinely unscrubbable secret is produced by making the target unwritable,
# so the literal rewrite cannot land. The run must exit non-zero and must not
# leave an unscrubbed transcript behind.
printf 'Authorization: Bearer %s\n' "$FIXTURE_PLAIN" > "$TMP/g_loud.txt"
chmod a-w "$TMP/g_loud.txt" 2>/dev/null || true
mkdir -p "$TMP/loud_ro" && mv "$TMP/g_loud.txt" "$TMP/loud_ro/g_loud.txt" 2>/dev/null || true
chmod a-w "$TMP/loud_ro" 2>/dev/null || true
run_scrub "$TMP/loud_ro/g_loud.txt" "$FIXTURE_PLAIN" > "$TMP/g_loud.log" 2>&1
loud_rc=$?
chmod u+w "$TMP/loud_ro" 2>/dev/null || true
chmod u+w "$TMP/loud_ro/g_loud.txt" 2>/dev/null || true
if [ "$loud_rc" -eq 0 ] && grep -qF -- "$FIXTURE_PLAIN" "$TMP/loud_ro/g_loud.txt" 2>/dev/null; then
    echo "  [FAIL] unscrubbable secret: exited 0 while the secret remained — FAIL-OPEN"
    green_fail=1
elif [ "$loud_rc" -ne 0 ]; then
    echo "  [ok] unscrubbable secret: aborted loudly (rc=$loud_rc)"
    if grep -q 'TRANSCRIPT SCRUBBING FAILED' "$TMP/g_loud.log" 2>/dev/null; then
        echo "  [ok] abort message present on stderr (not swallowed)"
    fi
else
    echo "  [ok] unscrubbable secret: removed despite restricted permissions"
fi

# --- (4b) the abort must terminate the CAPTURE SCRIPT, not just a subshell -
# The library is SOURCED and callers run `set -uo pipefail` WITHOUT errexit, so
# "we returned non-zero" is not enough: if _qa_scrub_fail's exit only killed a
# subshell, the capture would sail past a failed scrub and go on to record a
# PASS — a fail-open at the control-flow layer rather than the string layer.
cat > "$TMP/fake_capture.sh" <<'CAP'
set -uo pipefail
source "$LIBPATH"
QA_REPO_ROOT="$REPOROOT"
QA_REDACT_PATTERNS=()
f="$WORKDIR/prop.http"
key="sk-$(printf 'FAKEabort')$(printf 'M%.0s' $(seq 1 24))"
printf '< HTTP/1.1 200 OK\n--- RESPONSE BODY (9 bytes) ---\n{"k":"%s"}\n--- curl exit code: 0 ---\n' "$key" > "$f"
_qa_apply_redaction "$f"
echo "REACHED_LINE_AFTER_SCRUB"
CAP
prop_out="$(LIBPATH="$LIB" REPOROOT="$ROOT" WORKDIR="$TMP" bash "$TMP/fake_capture.sh" 2>&1)"
prop_rc=$?
if case "$prop_out" in *REACHED_LINE_AFTER_SCRUB*) true ;; *) false ;; esac; then
    echo "  [FAIL] abort did NOT propagate — a capture would continue past a failed scrub"
    green_fail=1
elif [ "$prop_rc" -eq 0 ]; then
    echo "  [FAIL] capture script exited 0 despite a failed scrub"
    green_fail=1
else
    echo "  [ok] abort terminates the whole capture script (rc=$prop_rc), not just a subshell"
fi
if [ -f "$TMP/prop.http" ]; then
    echo "  [FAIL] the unscrubbable transcript was left on disk"
    green_fail=1
else
    echo "  [ok] unscrubbable transcript quarantined, not left for a later commit"
fi

# --- (5) the abort path must never print the secret value (§11.4.10) -------
for logf in "$TMP/g_meta.log" "$TMP/g_loud.log"; do
    [ -f "$logf" ] || continue
    if grep -qF -- "$FIXTURE_META" "$logf" || grep -qF -- "$FIXTURE_PLAIN" "$logf"; then
        echo "  [FAIL] a secret VALUE was printed in the failure output ($logf)"
        green_fail=1
    fi
done
echo "  [ok] no secret value appears in any failure output"

# --- (6) FALSE-POSITIVE control: ordinary transcript content must survive ---
# Sec-WebSocket-Accept is a publicly computable SHA-1 digest, not a credential;
# trace ids and response ids are not credentials either. If the scan mangles
# these it is crying wolf (§11.4.201) and will get switched off.
cat > "$TMP/g_benign.txt" <<'BENIGN'
< HTTP/1.1 101 Switching Protocols
< Upgrade: websocket
< Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=
< x-debug-trace-id: 7f3a9c21e8b44d6fa0c5e7d92b13a4f8
< Content-Type: application/json
--- RESPONSE BODY (64 bytes) ---
{"id":"chatcmpl-9xKq2mTvB7nHsLpQwErTyUiOpAsDfGhJ","object":"chat.completion"}
BENIGN
cp "$TMP/g_benign.txt" "$TMP/g_benign.orig"
run_scrub "$TMP/g_benign.txt" >/dev/null 2>&1
if [ ! -f "$TMP/g_benign.txt" ]; then
    echo "  [FAIL] benign transcript was QUARANTINED — the scan is crying wolf (§11.4.201)"
    green_fail=1
elif ! cmp -s "$TMP/g_benign.txt" "$TMP/g_benign.orig"; then
    echo "  [FAIL] benign transcript was MODIFIED — false positive (§11.4.201)"
    diff "$TMP/g_benign.orig" "$TMP/g_benign.txt" | head -10
    green_fail=1
else
    echo "  [ok] benign transcript untouched (no false positive on WS digest / trace id / response id)"
fi

echo
if [ "$green_fail" -eq 0 ]; then
    echo "GREEN PASS — HXC-167 fix holds: literal redaction, loud failure, post-write scan"
    echo "             catches server-originated secrets, and benign content is untouched."
    exit 0
fi
fail "one or more HXC-167 invariants regressed (see [FAIL] lines above)"
