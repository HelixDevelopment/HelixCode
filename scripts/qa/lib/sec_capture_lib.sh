#!/usr/bin/env bash
# sec_capture_lib.sh — shared capture/assert library for the §11.4.83 retrospective
# security QA runs (docs/qa/<run-id>/).
#
# WHAT THIS IS
#   A real HTTP capture + assertion harness. Every assertion is evaluated against
#   bytes actually observed on the wire during THIS run, written verbatim into a
#   per-case transcript under docs/qa/<run-id>/transcripts/. There is no code path
#   that emits a PASS without a non-empty captured transcript for that case.
#
# ANTI-BLUFF CONTRACT (§11.4 / §11.4.1 / §11.4.5 / §11.4.6)
#   * assert_* helpers FAIL (not skip, not pass) when the transcript for the case
#     is missing or empty — you cannot assert on nothing.
#   * SKIP is a distinct terminal state. A SKIP is NEVER counted as a PASS and
#     forces a non-zero exit (2). Every SKIP carries a mandatory reason string
#     (§11.4.3 SKIP-with-reason).
#   * Exit codes:  0 = every assertion PASSed
#                  1 = at least one assertion FAILed  (security fix broken/regressed)
#                  2 = no FAIL, but at least one SKIP (evidence INCOMPLETE — the
#                      run certifies nothing about the skipped case)
#   * --self-test runs the assertion engine against known-good and known-bad
#     fixtures and proves the engine reports the bad one as FAIL
#     (§11.4.107(10) self-validated analyzer). A harness that cannot fail is a
#     bluff gate.
#
# USAGE (sourced, not executed):
#   source "$(dirname "$0")/lib/sec_capture_lib.sh"
#   qa_init "<slug>" "<full-commit-sha>" "<human title>"
#   qa_http <case-id> <method> <url> [curl args...]
#   assert_status <case-id> <expected>
#   qa_finish

set -uo pipefail

# ---------------------------------------------------------------------------
# Globals
# ---------------------------------------------------------------------------
QA_REPO_ROOT="${QA_REPO_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)}"
QA_RUN_DIR=""
QA_RUN_ID=""
QA_COMMIT_SHA=""
QA_TITLE=""
QA_SLUG=""
QA_VERDICTS=""
QA_PASS=0
QA_FAIL=0
QA_SKIP=0
QA_REDACT_PATTERNS=()

# Ephemeral-server state (set by qa_boot_ephemeral_server)
QA_EPH_PID=""
QA_EPH_PORT=""
QA_EPH_CONFIG=""
QA_EPH_LOG=""
QA_EPH_KEY=""

qa_ts() { date -u +%Y%m%dT%H%M%SZ; }
qa_iso() { date -u +%Y-%m-%dT%H:%M:%SZ; }

# ---------------------------------------------------------------------------
# qa_init <slug> <full-sha> <title>
# ---------------------------------------------------------------------------
qa_init() {
    QA_SLUG="$1"
    QA_COMMIT_SHA="$2"
    QA_TITLE="$3"

    local sha8="${QA_COMMIT_SHA:0:8}"
    QA_RUN_ID="${QA_SLUG}_${sha8}_$(qa_ts)"
    QA_RUN_DIR="${QA_REPO_ROOT}/docs/qa/${QA_RUN_ID}"

    mkdir -p "${QA_RUN_DIR}/transcripts"
    QA_VERDICTS="${QA_RUN_DIR}/verdicts.tsv"
    printf 'result\tcase_id\tassertion\texpected\tobserved\tevidence\n' > "$QA_VERDICTS"

    echo "== QA run: ${QA_RUN_ID}"
    echo "== dir:    ${QA_RUN_DIR}"

    qa_capture_grounding
}

# Redact a literal secret from every transcript written after this call.
qa_redact() { [ -n "${1:-}" ] && QA_REDACT_PATTERNS+=("$1"); }

_qa_apply_redaction() {
    local f="$1" p
    for p in "${QA_REDACT_PATTERNS[@]:-}"; do
        [ -n "$p" ] || continue
        # shellcheck disable=SC2001
        sed -i "s|${p//|/\\|}|<EPHEMERAL_KEY_REDACTED>|g" "$f" 2>/dev/null || true
    done
}

# ---------------------------------------------------------------------------
# Grounding facts (§11.4.108 SOURCE -> ARTIFACT -> RUNTIME chain).
# Captured, never asserted-by-assumption.
# ---------------------------------------------------------------------------
qa_capture_grounding() {
    local out="${QA_RUN_DIR}/grounding.txt"
    {
        echo "# Grounding facts captured $(qa_iso)"
        echo "# §11.4.108: the running artifact must be the one built from the fixed source."
        echo
        echo "## repo HEAD"
        git -C "$QA_REPO_ROOT" log --oneline -1 2>/dev/null
        echo
        echo "## fix commit under test"
        git -C "$QA_REPO_ROOT" log --oneline -1 "$QA_COMMIT_SHA" 2>/dev/null
        echo
        echo "## fix commit is an ancestor of HEAD?"
        if git -C "$QA_REPO_ROOT" merge-base --is-ancestor "$QA_COMMIT_SHA" HEAD 2>/dev/null; then
            echo "YES — ${QA_COMMIT_SHA} <= HEAD"
        else
            echo "NO — ${QA_COMMIT_SHA} is NOT in HEAD's history"
        fi
        echo
        echo "## live server under test"
        echo "base url: ${QA_BASE_URL:-<unset>}"
        local lp
        lp="$(ss -lntp 2>/dev/null | grep -oP 'pid=\K[0-9]+(?=.*helixcode)' | head -1)"
        if [ -n "$lp" ]; then
            echo "listening pid: $lp"
            echo "exe: $(readlink -f "/proc/$lp/exe" 2>/dev/null)"
            echo "exe md5: $(md5sum "/proc/$lp/exe" 2>/dev/null | awk '{print $1}')"
            echo "cwd: $(readlink -f "/proc/$lp/cwd" 2>/dev/null)"
            echo "--- security-relevant env (presence only, values never printed, §11.4.10) ---"
            local v
            for v in HELIX_WIRE_FACADE_API_KEYS HELIX_CORS_ALLOWED_ORIGINS HELIX_WS_ALLOWED_ORIGINS; do
                if tr '\0' '\n' < "/proc/$lp/environ" 2>/dev/null | grep -q "^${v}="; then
                    echo "${v}=<SET>"
                else
                    echo "${v}=<UNSET — fail-closed / default-deny>"
                fi
            done
        else
            echo "(no listening helixcode process found via ss; grounding limited)"
        fi
    } > "$out" 2>&1
    echo "-- grounding captured: $out"
}

# ---------------------------------------------------------------------------
# qa_http <case-id> <method> <url> [extra curl args...]
#
# Performs ONE real HTTP request and writes the full bidirectional wire trace
# (curl -v: '> ' request lines, '< ' response lines) plus the response body to
# docs/qa/<run-id>/transcripts/<case-id>.http
#
# Sets: QA_LAST_CASE, QA_LAST_TRANSCRIPT, QA_LAST_STATUS, QA_LAST_CURL_RC
# ---------------------------------------------------------------------------
qa_http() {
    local case_id="$1"; shift
    local method="$1"; shift
    local url="$1"; shift

    local t="${QA_RUN_DIR}/transcripts/${case_id}.http"
    local body_f; body_f="$(mktemp)"
    local trace_f; trace_f="$(mktemp)"

    {
        echo "### case: ${case_id}"
        echo "### time: $(qa_iso)"
        echo "### request: ${method} ${url}"
        echo "### curl args: $*"
        echo
    } > "$t"

    # curl's non-zero rc is DATA here (it is recorded and asserted on), not a
    # failure — so errexit is suspended across the call. It MUST be restored to
    # whatever it was before, never switched ON: this library and every capture
    # script run under `set -uo pipefail` WITHOUT errexit, so a bare `set -e`
    # here silently enabled errexit for the remainder of the run. The first
    # command legitimately returning non-zero then killed the script — and in
    # assert_header_absent that command is the grep whose EMPTY result is the
    # PASSING condition, so a run was destroyed at the exact moment a security
    # assertion succeeded (a §11.4.1 FAIL-bluff: script-internal death reported
    # as a failed capture while every evaluated assertion had passed).
    local _qa_errexit_was_on=0
    case $- in *e*) _qa_errexit_was_on=1 ;; esac
    set +e
    curl -sS -v -X "$method" "$url" "$@" -o "$body_f" > /dev/null 2> "$trace_f"
    QA_LAST_CURL_RC=$?
    if [ "$_qa_errexit_was_on" = 1 ]; then set -e; fi

    {
        echo "--- WIRE TRACE (curl -v; '>' = sent by client, '<' = received from server) ---"
        cat "$trace_f"
        echo
        echo "--- RESPONSE BODY ($(wc -c < "$body_f") bytes) ---"
        cat "$body_f"
        echo
        echo "--- curl exit code: ${QA_LAST_CURL_RC} ---"
    } >> "$t"

    # Status line is read from the RECEIVED half of the trace only.
    QA_LAST_STATUS="$(grep -m1 '^< HTTP/' "$trace_f" | awk '{print $3}')"
    QA_LAST_TRANSCRIPT="$t"
    QA_LAST_CASE="$case_id"
    QA_LAST_BODY="$(cat "$body_f")"

    rm -f "$body_f" "$trace_f"
    _qa_apply_redaction "$t"

    echo "-- ${case_id}: HTTP ${QA_LAST_STATUS:-<none>} (curl rc=${QA_LAST_CURL_RC})"
}

# ---------------------------------------------------------------------------
# Verdict recording
# ---------------------------------------------------------------------------
_qa_record() {
    local result="$1" case_id="$2" assertion="$3" expected="$4" observed="$5"
    local ev="docs/qa/${QA_RUN_ID}/transcripts/${case_id}.http"
    printf '%s\t%s\t%s\t%s\t%s\t%s\n' \
        "$result" "$case_id" "$assertion" "$expected" "$observed" "$ev" >> "$QA_VERDICTS"
    case "$result" in
        PASS) QA_PASS=$((QA_PASS+1)); echo "   PASS  ${case_id}: ${assertion}" ;;
        FAIL) QA_FAIL=$((QA_FAIL+1)); echo "   FAIL  ${case_id}: ${assertion} | expected=${expected} observed=${observed}" ;;
        SKIP) QA_SKIP=$((QA_SKIP+1)); echo "   SKIP  ${case_id}: ${assertion} | reason=${observed}" ;;
    esac
}

# Guard: every assertion requires a real, non-empty transcript for the case.
_qa_have_evidence() {
    local case_id="$1"
    local t="${QA_RUN_DIR}/transcripts/${case_id}.http"
    [ -s "$t" ] || return 1
    grep -q '^< HTTP/' "$t" || return 1
    return 0
}

# assert_status <case-id> <expected-status>
assert_status() {
    local case_id="$1" want="$2"
    if ! _qa_have_evidence "$case_id"; then
        _qa_record FAIL "$case_id" "status == ${want}" "$want" "NO-EVIDENCE (no response captured)"
        return 1
    fi
    local got; got="$(grep -m1 '^< HTTP/' "${QA_RUN_DIR}/transcripts/${case_id}.http" | awk '{print $3}')"
    if [ "$got" = "$want" ]; then
        _qa_record PASS "$case_id" "status == ${want}" "$want" "$got"
    else
        _qa_record FAIL "$case_id" "status == ${want}" "$want" "${got:-<none>}"
    fi
}

# assert_status_not <case-id> <forbidden-status> — for positive auth cases where
# the exact downstream status is not the security property under test.
assert_status_not() {
    local case_id="$1" bad="$2"
    if ! _qa_have_evidence "$case_id"; then
        _qa_record FAIL "$case_id" "status != ${bad}" "not ${bad}" "NO-EVIDENCE (no response captured)"
        return 1
    fi
    local got; got="$(grep -m1 '^< HTTP/' "${QA_RUN_DIR}/transcripts/${case_id}.http" | awk '{print $3}')"
    if [ -n "$got" ] && [ "$got" != "$bad" ]; then
        _qa_record PASS "$case_id" "status != ${bad}" "not ${bad}" "$got"
    else
        _qa_record FAIL "$case_id" "status != ${bad}" "not ${bad}" "${got:-<none>}"
    fi
}

# assert_header_absent <case-id> <header-name>
# Asserts the RESPONSE (received '<' half) carries no such header.
assert_header_absent() {
    local case_id="$1" hdr="$2"
    if ! _qa_have_evidence "$case_id"; then
        _qa_record FAIL "$case_id" "response header '${hdr}' absent" "absent" "NO-EVIDENCE"
        return 1
    fi
    local got
    got="$(grep -i "^< ${hdr}:" "${QA_RUN_DIR}/transcripts/${case_id}.http" | head -3 | tr '\n' ';')"
    if [ -z "$got" ]; then
        _qa_record PASS "$case_id" "response header '${hdr}' absent" "absent" "absent"
    else
        _qa_record FAIL "$case_id" "response header '${hdr}' absent" "absent" "$got"
    fi
}

# assert_header_equals <case-id> <header-name> <expected-value>
assert_header_equals() {
    local case_id="$1" hdr="$2" want="$3"
    if ! _qa_have_evidence "$case_id"; then
        _qa_record FAIL "$case_id" "header ${hdr} == ${want}" "$want" "NO-EVIDENCE"
        return 1
    fi
    local got
    got="$(grep -i "^< ${hdr}:" "${QA_RUN_DIR}/transcripts/${case_id}.http" | head -1 | sed -E "s/^< [^:]+:[[:space:]]*//" | tr -d '\r')"
    if [ "$got" = "$want" ]; then
        _qa_record PASS "$case_id" "header ${hdr} == ${want}" "$want" "$got"
    else
        _qa_record FAIL "$case_id" "header ${hdr} == ${want}" "$want" "${got:-<absent>}"
    fi
}

# assert_header_present <case-id> <header-name> [expected-substring]
assert_header_present() {
    local case_id="$1" hdr="$2" want="${3:-}"
    if ! _qa_have_evidence "$case_id"; then
        _qa_record FAIL "$case_id" "header ${hdr} present" "present" "NO-EVIDENCE"
        return 1
    fi
    local got
    got="$(grep -i "^< ${hdr}:" "${QA_RUN_DIR}/transcripts/${case_id}.http" | head -3 | sed -E "s/^< [^:]+:[[:space:]]*//" | tr -d '\r' | tr '\n' ';')"
    if [ -z "$got" ]; then
        _qa_record FAIL "$case_id" "header ${hdr} present${want:+ containing '${want}'}" "present" "<absent>"
    elif [ -n "$want" ] && ! printf '%s' "$got" | grep -qi -- "$want"; then
        _qa_record FAIL "$case_id" "header ${hdr} contains '${want}'" "$want" "$got"
    else
        _qa_record PASS "$case_id" "header ${hdr} present${want:+ containing '${want}'}" "present" "$got"
    fi
}

# assert_body_not_contains <case-id> <needle> — proves a rejected request did NOT
# reach the protected backend (e.g. no provider completion leaked back).
assert_body_not_contains() {
    local case_id="$1" needle="$2"
    if ! _qa_have_evidence "$case_id"; then
        _qa_record FAIL "$case_id" "body excludes '${needle}'" "absent" "NO-EVIDENCE"
        return 1
    fi
    local bodysec
    bodysec="$(sed -n '/^--- RESPONSE BODY/,/^--- curl exit code/p' "${QA_RUN_DIR}/transcripts/${case_id}.http")"
    if printf '%s' "$bodysec" | grep -q -- "$needle"; then
        _qa_record FAIL "$case_id" "body excludes '${needle}'" "absent" "PRESENT — protected backend was reached"
    else
        _qa_record PASS "$case_id" "body excludes '${needle}'" "absent" "absent"
    fi
}

# assert_body_contains <case-id> <needle>
assert_body_contains() {
    local case_id="$1" needle="$2"
    if ! _qa_have_evidence "$case_id"; then
        _qa_record FAIL "$case_id" "body contains '${needle}'" "$needle" "NO-EVIDENCE"
        return 1
    fi
    local bodysec
    bodysec="$(sed -n '/^--- RESPONSE BODY/,/^--- curl exit code/p' "${QA_RUN_DIR}/transcripts/${case_id}.http")"
    if printf '%s' "$bodysec" | grep -q -- "$needle"; then
        _qa_record PASS "$case_id" "body contains '${needle}'" "$needle" "present"
    else
        _qa_record FAIL "$case_id" "body contains '${needle}'" "$needle" "<absent>"
    fi
}

# THE CORS invariant: never wildcard-origin together with credentials.
# Evaluated against EVERY transcript captured in this run.
assert_no_wildcard_with_credentials() {
    local t case_id acao acac bad=0
    for t in "${QA_RUN_DIR}"/transcripts/*.http; do
        [ -e "$t" ] || continue
        case_id="$(basename "$t" .http)"
        acao="$(grep -i '^< Access-Control-Allow-Origin:' "$t" | head -1 | sed -E 's/^< [^:]+:[[:space:]]*//' | tr -d '\r')"
        acac="$(grep -i '^< Access-Control-Allow-Credentials:' "$t" | head -1 | sed -E 's/^< [^:]+:[[:space:]]*//' | tr -d '\r')"
        if [ "$acao" = "*" ] && [ -n "$acac" ]; then
            _qa_record FAIL "$case_id" "NOT (ACAO:* AND ACAC present) [CORS spec invariant]" \
                "no wildcard+credentials" "ACAO=* ACAC=${acac}"
            bad=1
        fi
    done
    [ $bad -eq 0 ] && _qa_record PASS "ALL-CASES" \
        "NOT (ACAO:* AND ACAC present) across every captured response" \
        "no wildcard+credentials" "invariant held on $(ls -1 "${QA_RUN_DIR}"/transcripts/*.http 2>/dev/null | wc -l) captured responses"
    return 0
}

# qa_skip <case-id> <assertion> <reason>  — §11.4.3 SKIP-with-reason. Never a PASS.
qa_skip() { _qa_record SKIP "$1" "$2" "n/a" "$3"; }

# ---------------------------------------------------------------------------
# Ephemeral configured server (needed for POSITIVE cases: the already-running
# instance is deliberately fail-closed with NO keys / NO allowlist configured,
# so an allow-path can only be exercised on a purpose-configured instance).
#
# qa_boot_ephemeral_server <env-assignments...>
# Returns 0 on healthy boot, 1 otherwise (caller must qa_skip on 1).
# ---------------------------------------------------------------------------
qa_free_port() {
    python3 - <<'PY' 2>/dev/null || echo ""
import socket
s = socket.socket()
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
}

qa_boot_ephemeral_server() {
    local bin="${QA_REPO_ROOT}/helix_code/bin/helixcode"
    local src_cfg="${QA_SRC_CONFIG:-${QA_REPO_ROOT}/helix_code/config/replica-8081.yaml}"

    if [ ! -x "$bin" ]; then
        QA_EPH_SKIP_REASON="server binary not present at ${bin} (build it: cd helix_code && make build)"
        return 1
    fi
    if [ ! -f "$src_cfg" ]; then
        QA_EPH_SKIP_REASON="template config not found at ${src_cfg}"
        return 1
    fi

    QA_EPH_PORT="$(qa_free_port)"
    if [ -z "$QA_EPH_PORT" ]; then
        QA_EPH_SKIP_REASON="could not allocate a free TCP port (python3 unavailable?)"
        return 1
    fi

    QA_EPH_CONFIG="${QA_RUN_DIR}/ephemeral-config.yaml"
    # Rewrite ONLY the port inside the top-level `server:` block, and bind to
    # loopback so the ephemeral instance is never externally reachable.
    awk -v port="$QA_EPH_PORT" '
        /^server:/ { inserver=1; print; next }
        /^[a-z_]+:/ && !/^server:/ { inserver=0 }
        inserver && /^[[:space:]]+port:/ { sub(/port:.*/, "port: " port); print; next }
        inserver && /^[[:space:]]+address:/ { sub(/address:.*/, "address: \"127.0.0.1\""); print; next }
        { print }
    ' "$src_cfg" > "$QA_EPH_CONFIG"

    if ! grep -qE "^[[:space:]]+port:[[:space:]]*${QA_EPH_PORT}\b" "$QA_EPH_CONFIG"; then
        QA_EPH_SKIP_REASON="failed to rewrite server.port into ephemeral config"
        return 1
    fi

    QA_EPH_LOG="${QA_RUN_DIR}/ephemeral-server.log"

    # Inherit the live server's required secrets ONLY if already exported by the
    # caller's environment (§11.4.10 — never read a secret out of a config file
    # into the transcript, never hardcode one here).
    (
        cd "${QA_REPO_ROOT}/helix_code" || exit 1
        env HELIX_CONFIG="$QA_EPH_CONFIG" \
            HELIX_AUTOBOOT_INFRA=false \
            "$@" \
            "$bin" server
    ) > "$QA_EPH_LOG" 2>&1 &
    QA_EPH_PID=$!

    local i
    for i in $(seq 1 45); do
        if ! kill -0 "$QA_EPH_PID" 2>/dev/null; then
            QA_EPH_SKIP_REASON="ephemeral server exited during boot; see $(basename "$QA_EPH_LOG"): $(tail -3 "$QA_EPH_LOG" 2>/dev/null | tr '\n' ' ')"
            QA_EPH_PID=""
            return 1
        fi
        if curl -sS -m 2 -o /dev/null "http://127.0.0.1:${QA_EPH_PORT}/health" 2>/dev/null; then
            echo "-- ephemeral server healthy on 127.0.0.1:${QA_EPH_PORT} (pid ${QA_EPH_PID})"
            return 0
        fi
        sleep 1
    done

    QA_EPH_SKIP_REASON="ephemeral server did not become healthy within 45s; see $(basename "$QA_EPH_LOG")"
    qa_stop_ephemeral_server
    return 1
}

# §11.4.174: only ever signal the PID WE launched. Never a pattern-kill.
qa_stop_ephemeral_server() {
    if [ -n "$QA_EPH_PID" ] && kill -0 "$QA_EPH_PID" 2>/dev/null; then
        kill "$QA_EPH_PID" 2>/dev/null
        local i
        for i in 1 2 3 4 5; do
            kill -0 "$QA_EPH_PID" 2>/dev/null || break
            sleep 1
        done
        kill -9 "$QA_EPH_PID" 2>/dev/null
        echo "-- ephemeral server (pid ${QA_EPH_PID}) stopped"
    fi
    QA_EPH_PID=""
}

# ---------------------------------------------------------------------------
# qa_finish — write EVIDENCE.md and exit with the three-state code.
# ---------------------------------------------------------------------------
qa_finish() {
    local ev="${QA_RUN_DIR}/EVIDENCE.md"
    local verdict rc
    if [ "$QA_FAIL" -gt 0 ]; then
        verdict="FAIL"; rc=1
    elif [ "$QA_SKIP" -gt 0 ]; then
        verdict="INCOMPLETE (PASS with SKIPs — certifies nothing about the skipped cases)"; rc=2
    else
        verdict="PASS"; rc=0
    fi

    {
        echo "# ${QA_TITLE} — QA evidence (§11.4.83)"
        echo
        echo "**Run ID:** \`${QA_RUN_ID}\`"
        echo "**Source commit:** \`${QA_COMMIT_SHA}\`"
        echo "**Captured (UTC):** $(qa_iso)"
        echo "**Verdict:** ${verdict}"
        echo "**Counts:** PASS=${QA_PASS} FAIL=${QA_FAIL} SKIP=${QA_SKIP}"
        echo
        echo "## Commit under test"
        echo '```'
        git -C "$QA_REPO_ROOT" log -1 --format='%H%n%an%n%cI%n%s' "$QA_COMMIT_SHA" 2>/dev/null
        echo '```'
        echo
        echo "## How this evidence was produced"
        echo
        echo "Generated by \`scripts/qa/${QA_SCRIPT_NAME:-<script>}\` — a real HTTP capture harness."
        echo "Every row below was evaluated against bytes observed on the wire during this run;"
        echo "the full bidirectional trace for each case (request lines sent, response headers and"
        echo "body received) is in \`transcripts/<case-id>.http\`. The harness FAILs any assertion"
        echo "whose transcript is missing or contains no response — it cannot emit a PASS without"
        echo "captured evidence (§11.4 / §11.4.1 / §11.4.5)."
        echo
        echo "Exit codes: \`0\`=all PASS, \`1\`=a security assertion FAILED, \`2\`=INCOMPLETE (a case was"
        echo "SKIPped with reason, §11.4.3 — never counted as a pass)."
        echo
        echo "## Per-assertion verdicts"
        echo
        echo "| Result | Case | Assertion | Expected | Observed |"
        echo "|--------|------|-----------|----------|----------|"
        tail -n +2 "$QA_VERDICTS" | while IFS=$'\t' read -r r c a e o _; do
            printf '| %s | `%s` | %s | `%s` | `%s` |\n' "$r" "$c" "$a" "$e" "$o"
        done
        echo
        if [ "$QA_SKIP" -gt 0 ]; then
            echo "## SKIPped cases (§11.4.3) — NOT certified by this run"
            echo
            grep -P '^SKIP\t' "$QA_VERDICTS" | while IFS=$'\t' read -r _ c a _ o _; do
                echo "- \`${c}\` — ${a} — **reason:** ${o}"
            done
            echo
        fi
        echo "## Grounding (§11.4.108)"
        echo
        echo '```'
        cat "${QA_RUN_DIR}/grounding.txt" 2>/dev/null
        echo '```'
        echo
        echo "## Files"
        echo
        echo "- \`verdicts.tsv\` — machine-readable per-assertion results"
        echo "- \`grounding.txt\` — artifact/runtime identity facts"
        echo "- \`transcripts/*.http\` — full bidirectional wire traces"
        [ -f "$QA_EPH_LOG" ] && echo "- \`ephemeral-server.log\` — boot log of the purpose-configured instance"
        [ -f "$QA_EPH_CONFIG" ] && echo "- \`ephemeral-config.yaml\` — generated config for that instance"
    } > "$ev"

    _qa_apply_redaction "$ev"

    echo
    echo "==============================================================="
    echo " ${QA_TITLE}"
    echo " verdict : ${verdict}"
    echo " counts  : PASS=${QA_PASS} FAIL=${QA_FAIL} SKIP=${QA_SKIP}"
    echo " evidence: docs/qa/${QA_RUN_ID}/"
    echo "==============================================================="
    exit "$rc"
}

# ---------------------------------------------------------------------------
# --self-test: prove the assertion engine can actually FAIL (§11.4.107(10)).
# Runs the helpers against a golden-good and a golden-bad synthetic transcript.
# ---------------------------------------------------------------------------
qa_self_test() {
    local tmp; tmp="$(mktemp -d)"
    QA_RUN_DIR="$tmp"; QA_RUN_ID="selftest"
    mkdir -p "$tmp/transcripts"
    QA_VERDICTS="$tmp/verdicts.tsv"; : > "$QA_VERDICTS"

    # golden-GOOD: fixed behaviour (specific origin, Vary, 401 on unauth)
    cat > "$tmp/transcripts/good.http" <<'EOF'
--- WIRE TRACE ---
< HTTP/1.1 401 Unauthorized
< Vary: Origin
< Access-Control-Allow-Origin: https://app.allowed.example
< Access-Control-Allow-Credentials: true
--- RESPONSE BODY (10 bytes) ---
{"error":1}
--- curl exit code: 0 ---
EOF
    # golden-BAD: the pre-fix vulnerability (wildcard + credentials, unauth 200)
    cat > "$tmp/transcripts/bad.http" <<'EOF'
--- WIRE TRACE ---
< HTTP/1.1 200 OK
< Access-Control-Allow-Origin: *
< Access-Control-Allow-Credentials: true
--- RESPONSE BODY (30 bytes) ---
{"choices":[{"text":"hello"}]}
--- curl exit code: 0 ---
EOF
    # golden-EMPTY: no response captured at all
    : > "$tmp/transcripts/empty.http"

    local p0=$QA_PASS f0=$QA_FAIL
    echo "== self-test: golden-GOOD must PASS"
    QA_PASS=0; QA_FAIL=0
    assert_status good 401
    assert_header_present good "Vary" "Origin"
    assert_header_equals good "Access-Control-Allow-Origin" "https://app.allowed.example"
    assert_body_not_contains good '"choices"'
    local good_pass=$QA_PASS good_fail=$QA_FAIL

    echo "== self-test: golden-BAD must FAIL (this is the anti-bluff proof)"
    QA_PASS=0; QA_FAIL=0
    assert_status bad 401
    assert_header_absent bad "Access-Control-Allow-Origin"
    assert_body_not_contains bad '"choices"'
    assert_no_wildcard_with_credentials
    local bad_pass=$QA_PASS bad_fail=$QA_FAIL

    echo "== self-test: EMPTY transcript must FAIL (cannot assert on nothing)"
    QA_PASS=0; QA_FAIL=0
    assert_status empty 401
    local empty_fail=$QA_FAIL

    rm -rf "$tmp"
    QA_PASS=$p0; QA_FAIL=$f0

    echo
    echo "  golden-GOOD : PASS=${good_pass} FAIL=${good_fail}   (want FAIL=0)"
    echo "  golden-BAD  : PASS=${bad_pass} FAIL=${bad_fail}   (want FAIL>=4)"
    echo "  EMPTY       : FAIL=${empty_fail}                   (want FAIL=1)"
    if [ "$good_fail" -eq 0 ] && [ "$bad_fail" -ge 4 ] && [ "$empty_fail" -eq 1 ]; then
        echo "SELF-TEST PASS — the assertion engine provably detects the pre-fix vulnerability."
        return 0
    fi
    echo "SELF-TEST FAIL — the harness cannot be trusted; do not run the captures."
    return 1
}
