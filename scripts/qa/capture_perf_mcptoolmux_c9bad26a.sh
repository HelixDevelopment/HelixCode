#!/usr/bin/env bash
# capture_perf_mcptoolmux_c9bad26a.sh
#
# §11.4.83 retrospective end-user QA capture for:
#   c9bad26a  fix(mcp): move blocking I/O out of toolMux critical section
#             (lock-hold O(N^2) -> snapshot-then-act)
#
# WHAT THE FIX CHANGED (helix_code/internal/mcp/server.go)
#   BEFORE: handleListTools held toolMux.RLock() (via `defer RUnlock()`) across
#           the ENTIRE tool-map copy AND the sendMessage -> conn.WriteJSON call
#           (which JSON-marshals the returned tools array and performs the
#           actual socket write). RegisterTool likewise held toolMux.Lock()
#           across the map insert AND a log.Print call (which serialises on
#           Go's process-global log mutex + a stderr write).
#           Per the commit message, Go's writer-preferring sync.RWMutex means
#           a long RLock hold blocks a QUEUED writer, and once a writer is
#           queued it blocks every SUBSEQUENT reader — so concurrent
#           register+list traffic collapses onto a fully-serial critical
#           section whose length grows with tool count (measured O(N^2) as
#           the registered-tool count grew 1->1920 in the Go stress test
#           TestMCPServer_Stress_ConcurrentRegisterListCall, which crossed its
#           25s wall-clock guard before the fix).
#   AFTER:  handleListTools builds the tools snapshot under RLock, then calls
#           RUnlock() BEFORE building the response / calling sendMessage.
#           RegisterTool mutates the map under Lock, calls Unlock() BEFORE
#           log.Print. Copy-under-lock-then-act — the discipline
#           handleCallTool already used for its own toolMux.RLock().
#
# WHAT THIS CAPTURE PROVES, AND WHAT IT HONESTLY DOES NOT (§11.4.6)
#   internal/mcp.MCPServer.RegisterTool is called from exactly one place in
#   the whole tree outside test files: a doc-comment example
#   (internal/mcp/doc.go). No production wiring in internal/server calls
#   RegisterTool at boot or at runtime (grep verified, see grounding.txt) —
#   so the concurrent WRITER (RegisterTool) side of the historical defect
#   (concurrent register-while-list, the exact shape
#   TestMCPServer_Stress_ConcurrentRegisterListCall drives in-process) is NOT
#   reachable from outside the process over the wire. That half of the fix is
#   already covered by the Go stress test cited in the commit message
#   (RED 11.8s -> GREEN 3.5s, -race, 0 deadlock) — this capture does not
#   re-derive that number; it is cited below as the commit's OWN recorded
#   measurement, not something re-measured here.
#
#   What IS reachable, end-to-end, the way a real MCP client reaches it: many
#   independent MCP clients opening independent /ws sessions and concurrently
#   calling tools/list against the SAME running server process, through the
#   exact wire path (WebSocket upgrade -> JSON-RPC "initialize" ->
#   JSON-RPC "tools/list") a real client uses. This capture drives that path
#   for real, with N=32 independent concurrent WebSocket sessions (§11.4.85
#   stress requires N>=10 for concurrent contention; 32 is comfortably above
#   that floor and above the commit's own cited 16-goroutine scenario, while
#   staying light enough to run reliably on a shared development host,
#   §11.4.174), and MEASURES:
#     (a) correctness under concurrent load — every one of the N concurrent
#         callers gets back a well-formed, correctly-correlated
#         "tools/list" response (§11.4.107: a fast wrong answer is not a pass);
#     (b) throughput — wall-clock time to complete N calls fired CONCURRENTLY
#         vs. the SAME N calls run SEQUENTIALLY, one connection at a time, on
#         the SAME running server, moments apart. If the handler still
#         serialised internally (the pre-fix critical-section shape), N
#         concurrent callers would gain little or nothing over N sequential
#         callers, because the "concurrent" work would collapse onto the same
#         one-at-a-time critical section anyway (plus queuing/lock overhead,
#         which would make concurrent execution SLOWER, not faster, than
#         sequential). Genuine parallel handling makes concurrent execution
#         markedly FASTER than sequential execution of the same N calls. This
#         PASS/FAIL bound is derived entirely from the two wall-clock numbers
#         measured in THIS run (concurrent_wall_clock_ms < sequential_wall_
#         clock_ms) — not an externally invented millisecond figure.
#     (c) a per-call latency ceiling under concurrency, calibrated against
#         THIS run's OWN sequential baseline: the 95th-percentile latency of
#         the N concurrent calls must not exceed 3x the SLOWEST sequential
#         call observed in this run (floored at 50ms to absorb ordinary
#         OS-scheduler / asyncio-loop dispatch jitter on a shared host rather
#         than manufacturing a razor-thin bound out of a sub-millisecond
#         loopback baseline). The 3x multiplier is a conservative allowance
#         for the scheduling/contention overhead inherent to N-way concurrent
#         dispatch relative to an uncontended sequential call, anchored to a
#         number this run itself measured, not to an externally chosen
#         constant.
#   The full raw per-call latency distribution (min/p50/p90/p95/max/mean) for
#   both runs is captured as FACTS (not gated behind pass/fail) in
#   CONCURRENCY_ANALYSIS.md alongside the raw JSON.
#
# WHY AN EPHEMERAL INSTANCE IS NEEDED (§11.4.6 honest scope, same pattern as
# capture_sec_ws_9c876819.sh / capture_feat_wirefacade_51c058b1.sh)
#   The already-running server (localhost:8081) has HELIX_WIRE_FACADE_API_KEYS
#   UNSET, so /ws is fail-closed (wsAuthMiddleware rejects with 401 before the
#   WebSocket upgrade). Phase 1 below uses that fail-closed 401 itself as a
#   real, live "the route is registered and gated, not missing" proof (same
#   idiom as the other captures' 401-not-404 checks). The actual tools/list
#   traffic can only be driven on an instance WITH a key configured, which
#   Phase 2 boots on a free loopback port via qa_boot_ephemeral_server(). This
#   test needs no LLM backend and no database — tools/list touches neither —
#   so, unlike capture_feat_wirefacade_51c058b1.sh, there is no CODER_UP
#   precondition here.
#
# TOOLING: the WebSocket JSON-RPC exchange is driven by a short Python
#   (asyncio + the `websockets` package, already present on this host) driver
#   fed to `python3 -` over stdin — no new tracked file, no dependency beyond
#   what is already installed. curl cannot complete an MCP JSON-RPC exchange
#   over an upgraded WebSocket connection, so it is used only for the
#   HTTP-level Phase 1 checks (route registered / gated), exactly as the
#   existing capture_sec_ws_9c876819.sh does for the same /ws route.
#
# EXIT CODES: 0 all PASS | 1 an assertion FAILED (perf/concurrency regressed)
#             | 2 INCOMPLETE (a case SKIPped with reason, §11.4.3)

set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/sec_capture_lib.sh
source "${SCRIPT_DIR}/lib/sec_capture_lib.sh"
QA_SCRIPT_NAME="capture_perf_mcptoolmux_c9bad26a.sh"

COMMIT="c9bad26aa7821832cc6297d7d2193731ba8ac733"
QA_BASE_URL="${QA_BASE_URL:-http://localhost:8081}"
CONCURRENCY_N="${QA_MCP_CONCURRENCY_N:-32}"

# A WebSocket handshake needs these four headers (RFC 6455 example nonce; not
# a credential).
WS_HDRS=(
    -H "Connection: Upgrade"
    -H "Upgrade: websocket"
    -H "Sec-WebSocket-Version: 13"
    -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ=="
)

# ---------------------------------------------------------------------------
# Custom oracle: evaluates a {sequential.json, concurrent.json} pair produced
# by the Python driver below and records PASS/FAIL/SKIP verdicts via the
# lib's _qa_record (sourced, not edited). Kept as a function so --self-test
# can run it against synthetic golden-good/golden-bad fixtures
# (§11.4.107(10) — an oracle that cannot fail is a bluff gate).
# ---------------------------------------------------------------------------
mcp_eval_concurrency() {
    local seq_json="$1" conc_json="$2" transcripts_dir="$3"

    if [ ! -s "$seq_json" ] || [ ! -s "$conc_json" ]; then
        _qa_record FAIL "eph_concurrent_all_ok" "every concurrent call returns a well-formed, correlated tools/list response" "0 errors" "NO-EVIDENCE (measurement file missing/empty)"
        _qa_record FAIL "eph_concurrent_faster_than_sequential" "concurrent_wall_clock_ms < sequential_wall_clock_ms" "concurrent faster" "NO-EVIDENCE"
        _qa_record FAIL "eph_concurrent_latency_bound" "concurrent p95 latency <= 3x sequential max (floor 50ms)" "bounded" "NO-EVIDENCE"
        return 1
    fi

    local seq_wall conc_wall seq_ok conc_ok seq_err conc_err seq_max conc_p95 bound
    seq_wall="$(jq -r '.wall_clock_ms' "$seq_json")"
    conc_wall="$(jq -r '.wall_clock_ms' "$conc_json")"
    seq_ok="$(jq -r '[.results[]|select(.ok==true)]|length' "$seq_json")"
    conc_ok="$(jq -r '[.results[]|select(.ok==true)]|length' "$conc_json")"
    seq_err="$(jq -r '[.results[]|select(.ok==false)]|length' "$seq_json")"
    conc_err="$(jq -r '[.results[]|select(.ok==false)]|length' "$conc_json")"
    seq_n="$(jq -r '.results|length' "$seq_json")"
    conc_n="$(jq -r '.results|length' "$conc_json")"
    seq_max="$(jq -r '[.results[]|select(.ok==true)|.latency_ms] | if length==0 then 0 else max end' "$seq_json")"
    conc_p95="$(jq -r '[.results[]|select(.ok==true)|.latency_ms] | sort | if length==0 then 0 else .[(length*0.95|floor) as $i | (if $i>=length then length-1 else $i end)] end' "$conc_json")"

    {
        echo "### case: eph_concurrent_all_ok / eph_concurrent_faster_than_sequential / eph_concurrent_latency_bound"
        echo "### time: $(qa_iso)"
        echo "### sequential run  : n=${seq_n} ok=${seq_ok} err=${seq_err} wall_clock_ms=${seq_wall} max_latency_ms=${seq_max}"
        echo "### concurrent run  : n=${conc_n} ok=${conc_ok} err=${conc_err} wall_clock_ms=${conc_wall} p95_latency_ms=${conc_p95}"
        echo "### raw files       : sequential.json, concurrent.json (this run directory)"
    } > "${transcripts_dir}/eph_concurrent_all_ok.http"
    cp "${transcripts_dir}/eph_concurrent_all_ok.http" "${transcripts_dir}/eph_concurrent_faster_than_sequential.http"
    cp "${transcripts_dir}/eph_concurrent_all_ok.http" "${transcripts_dir}/eph_concurrent_latency_bound.http"

    # (a) correctness under concurrent load
    if [ "$conc_err" = "0" ] && [ "$conc_n" -gt 0 ]; then
        _qa_record PASS "eph_concurrent_all_ok" "every concurrent call returns a well-formed, correlated tools/list response" "0 errors" "0 errors / ${conc_n} calls"
    else
        _qa_record FAIL "eph_concurrent_all_ok" "every concurrent call returns a well-formed, correlated tools/list response" "0 errors" "${conc_err} errors / ${conc_n} calls"
    fi

    # (b) throughput: concurrency must genuinely win over serial execution
    if awk -v a="$conc_wall" -v b="$seq_wall" 'BEGIN{exit !(a<b)}'; then
        _qa_record PASS "eph_concurrent_faster_than_sequential" "concurrent_wall_clock_ms < sequential_wall_clock_ms" "concurrent < ${seq_wall}ms" "${conc_wall}ms"
    else
        _qa_record FAIL "eph_concurrent_faster_than_sequential" "concurrent_wall_clock_ms < sequential_wall_clock_ms" "concurrent < ${seq_wall}ms" "${conc_wall}ms"
    fi

    # (c) bounded per-call latency under concurrency, calibrated on THIS run's
    #     own sequential ceiling (floor 50ms; 3x multiplier — see header).
    bound="$(awk -v m="$seq_max" 'BEGIN{f=(m>50?m:50); printf "%.3f", f*3}')"
    if awk -v p="$conc_p95" -v b="$bound" 'BEGIN{exit !(p<=b)}'; then
        _qa_record PASS "eph_concurrent_latency_bound" "concurrent p95 latency <= 3x sequential max (floor 50ms)" "<= ${bound}ms" "${conc_p95}ms"
    else
        _qa_record FAIL "eph_concurrent_latency_bound" "concurrent p95 latency <= 3x sequential max (floor 50ms)" "<= ${bound}ms" "${conc_p95}ms"
    fi
}

# ---------------------------------------------------------------------------
# --self-test: proves (1) the generic assertion engine can fail (lib's own
# qa_self_test) and (2) THIS script's concurrency oracle can fail, against
# synthetic golden-good / golden-bad measurement fixtures — never against a
# live server. §11.4.107(10): an analyzer that cannot fail is a bluff gate.
# ---------------------------------------------------------------------------
if [ "${1:-}" = "--self-test" ]; then
    qa_self_test
    generic_rc=$?

    tmp="$(mktemp -d)"
    mkdir -p "$tmp/transcripts"

    # golden-GOOD: concurrency genuinely wins, zero errors, tight p95.
    cat > "$tmp/good_seq.json" <<'EOF'
{"mode":"sequential","n":4,"wall_clock_ms":400.0,"results":[
 {"idx":0,"ok":true,"latency_ms":95.0},{"idx":1,"ok":true,"latency_ms":98.0},
 {"idx":2,"ok":true,"latency_ms":97.0},{"idx":3,"ok":true,"latency_ms":100.0}]}
EOF
    cat > "$tmp/good_conc.json" <<'EOF'
{"mode":"concurrent","n":4,"wall_clock_ms":110.0,"results":[
 {"idx":0,"ok":true,"latency_ms":100.0},{"idx":1,"ok":true,"latency_ms":102.0},
 {"idx":2,"ok":true,"latency_ms":101.0},{"idx":3,"ok":true,"latency_ms":103.0}]}
EOF
    # golden-BAD: the pre-fix shape — concurrency does NOT win (lock
    # serialisation), plus errors from timed-out queued callers.
    cat > "$tmp/bad_seq.json" <<'EOF'
{"mode":"sequential","n":4,"wall_clock_ms":400.0,"results":[
 {"idx":0,"ok":true,"latency_ms":95.0},{"idx":1,"ok":true,"latency_ms":98.0},
 {"idx":2,"ok":true,"latency_ms":97.0},{"idx":3,"ok":true,"latency_ms":100.0}]}
EOF
    cat > "$tmp/bad_conc.json" <<'EOF'
{"mode":"concurrent","n":4,"wall_clock_ms":9500.0,"results":[
 {"idx":0,"ok":true,"latency_ms":9000.0},{"idx":1,"ok":false,"error":"TimeoutError"},
 {"idx":2,"ok":true,"latency_ms":9100.0},{"idx":3,"ok":false,"error":"TimeoutError"}]}
EOF

    echo "== self-test: concurrency oracle, golden-GOOD must PASS all 3"
    QA_RUN_DIR="$tmp"; QA_VERDICTS="$tmp/verdicts.tsv"; : > "$QA_VERDICTS"
    QA_PASS=0; QA_FAIL=0
    mcp_eval_concurrency "$tmp/good_seq.json" "$tmp/good_conc.json" "$tmp/transcripts"
    good_pass=$QA_PASS; good_fail=$QA_FAIL

    echo "== self-test: concurrency oracle, golden-BAD must FAIL (anti-bluff proof)"
    QA_PASS=0; QA_FAIL=0
    mcp_eval_concurrency "$tmp/bad_seq.json" "$tmp/bad_conc.json" "$tmp/transcripts"
    bad_pass=$QA_PASS; bad_fail=$QA_FAIL

    echo "== self-test: missing measurement files must FAIL (cannot assert on nothing)"
    QA_PASS=0; QA_FAIL=0
    mcp_eval_concurrency "$tmp/does-not-exist.json" "$tmp/also-missing.json" "$tmp/transcripts"
    empty_fail=$QA_FAIL

    rm -rf "$tmp"

    echo
    echo "  generic engine (lib qa_self_test) : rc=${generic_rc} (want 0)"
    echo "  oracle golden-GOOD : PASS=${good_pass} FAIL=${good_fail}  (want FAIL=0)"
    echo "  oracle golden-BAD  : PASS=${bad_pass} FAIL=${bad_fail}   (want FAIL=3)"
    echo "  oracle NO-EVIDENCE : FAIL=${empty_fail}                  (want FAIL=3)"
    if [ "$generic_rc" -eq 0 ] && [ "$good_fail" -eq 0 ] && [ "$bad_fail" -eq 3 ] && [ "$empty_fail" -eq 3 ]; then
        echo "SELF-TEST PASS — both the generic engine and this script's concurrency oracle provably detect their golden-bad fixtures."
        exit 0
    fi
    echo "SELF-TEST FAIL — the harness cannot be trusted; do not run the captures."
    exit 1
fi

qa_init "perf_mcptoolmux" "$COMMIT" \
    "MCP tools/list stays correct and fast under concurrent client load (toolMux lock-hold fix)"
trap qa_stop_ephemeral_server EXIT

echo
echo "### PHASE 1 — /ws is REGISTERED and auth-gated on the LIVE server (${QA_BASE_URL})"
echo "### (fail-closed: HELIX_WIRE_FACADE_API_KEYS unset -> 401 before any MCP traffic)"
echo

qa_http "live_ws_route_gated" GET "${QA_BASE_URL}/ws" "${WS_HDRS[@]}" --max-time 10
assert_status            live_ws_route_gated 401
assert_header_absent     live_ws_route_gated "Sec-WebSocket-Accept"
assert_body_not_contains live_ws_route_gated "page not found"

# CONTROL: an unregistered sibling path must 404 — proves the harness can
# distinguish "registered but gated" (401 above) from "not registered" (404).
qa_http "live_ws_unregistered_control" GET "${QA_BASE_URL}/ws-qa-probe-no-such-route" "${WS_HDRS[@]}" --max-time 10
assert_status        live_ws_unregistered_control 404
assert_body_contains live_ws_unregistered_control "page not found"

echo
echo "### PHASE 2 — real concurrent MCP client load (ephemeral instance, N=${CONCURRENCY_N})"
echo

EPH_KEY="qa-ephemeral-$(head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n')"
qa_redact "$EPH_KEY"

if qa_boot_ephemeral_server \
        HELIX_WIRE_FACADE_API_KEYS="$EPH_KEY" \
        HELIX_AUTH_JWT_SECRET="${HELIX_AUTH_JWT_SECRET:-}" \
        HELIX_DATABASE_PASSWORD="${HELIX_DATABASE_PASSWORD:-}"
then
    EPH_WS="ws://127.0.0.1:${QA_EPH_PORT}/ws"

    # ---- Python asyncio+websockets driver -----------------------------
    # Reads (ws_url, api_key, mode, n, out_json_path) from argv; performs a
    # real WebSocket upgrade + "initialize" + "tools/list" JSON-RPC exchange
    # per call, on its OWN independent connection (models N independent MCP
    # clients, not one client issuing N calls on one session — the shape the
    # fix's concurrency defect actually depends on). Writes raw per-call
    # timings + correctness verdicts to out_json_path. No new tracked file:
    # the interpreter reads its program from stdin.
    mcp_probe() {
        local mode="$1" n="$2" out="$3"
        python3 - "$EPH_WS" "$EPH_KEY" "$mode" "$n" "$out" <<'PY'
import asyncio, json, sys, time

import websockets

ws_url, api_key, mode, n, out_path = sys.argv[1], sys.argv[2], sys.argv[3], int(sys.argv[4]), sys.argv[5]


async def one_call(idx: int) -> dict:
    t0 = time.perf_counter()
    try:
        async with websockets.connect(
            ws_url,
            additional_headers={"Authorization": f"Bearer {api_key}"},
            open_timeout=15,
            close_timeout=5,
        ) as ws:
            init_req = {
                "jsonrpc": "2.0",
                "id": f"init-{idx}",
                "method": "initialize",
                "params": {
                    "protocolVersion": "2024-11-05",
                    "capabilities": {},
                    "clientInfo": {"name": "qa-mcp-concurrency-capture", "version": "1.0.0"},
                },
            }
            await ws.send(json.dumps(init_req))
            await asyncio.wait_for(ws.recv(), timeout=20)

            list_id = f"list-{idx}"
            t_list0 = time.perf_counter()
            await ws.send(json.dumps({"jsonrpc": "2.0", "id": list_id, "method": "tools/list"}))
            raw = await asyncio.wait_for(ws.recv(), timeout=20)
            t_list1 = time.perf_counter()

            resp = json.loads(raw)
            ok = (
                resp.get("id") == list_id
                and resp.get("error") is None
                and isinstance(resp.get("result"), dict)
                and "tools" in resp.get("result", {})
                and isinstance(resp["result"]["tools"], list)
            )
            return {
                "idx": idx,
                "ok": bool(ok),
                "latency_ms": (t_list1 - t_list0) * 1000.0,
                "total_ms": (t_list1 - t0) * 1000.0,
                "response_snippet": raw[:300],
                "error": None,
            }
    except Exception as exc:  # noqa: BLE001 - deliberately broad: any failure is a real, reportable finding
        t1 = time.perf_counter()
        return {
            "idx": idx,
            "ok": False,
            "latency_ms": (t1 - t0) * 1000.0,
            "total_ms": (t1 - t0) * 1000.0,
            "response_snippet": None,
            "error": f"{type(exc).__name__}: {exc}",
        }


async def run_sequential(n: int):
    results = []
    wall0 = time.perf_counter()
    for i in range(n):
        results.append(await one_call(i))
    wall1 = time.perf_counter()
    return results, (wall1 - wall0) * 1000.0


async def run_concurrent(n: int):
    wall0 = time.perf_counter()
    results = await asyncio.gather(*(one_call(i) for i in range(n)))
    wall1 = time.perf_counter()
    return list(results), (wall1 - wall0) * 1000.0


async def main():
    if mode == "sequential":
        results, wall_ms = await run_sequential(n)
    else:
        results, wall_ms = await run_concurrent(n)
    with open(out_path, "w") as f:
        json.dump({"mode": mode, "n": n, "wall_clock_ms": wall_ms, "results": results}, f, indent=2)
    ok_n = sum(1 for r in results if r["ok"])
    err_n = n - ok_n
    print(f"-- mcp_probe mode={mode} n={n} wall_clock_ms={wall_ms:.1f} ok={ok_n} err={err_n}")


asyncio.run(main())
PY
    }

    # ---- functional smoke: one call, proves basic correctness first -------
    FUNC_JSON="${QA_RUN_DIR}/functional.json"
    mcp_probe "sequential" 1 "$FUNC_JSON"
    {
        echo "### case: eph_tools_list_functional"
        echo "### time: $(qa_iso)"
        echo "### single call over a real WS+JSON-RPC exchange (initialize -> tools/list)"
        echo "### raw file: functional.json"
        jq -r '.results[0] | "### ok=\(.ok) latency_ms=\(.latency_ms) response_snippet=\(.response_snippet // "<none>") error=\(.error // "<none>")"' "$FUNC_JSON" 2>/dev/null
    } > "${QA_RUN_DIR}/transcripts/eph_tools_list_functional.http"

    if jq -e '.results[0].ok == true' "$FUNC_JSON" >/dev/null 2>&1; then
        _qa_record PASS "eph_tools_list_functional" "single tools/list call returns a well-formed, correlated response" "ok=true" "ok=true"
    else
        obs="$(jq -r '.results[0].error // "ok=false"' "$FUNC_JSON" 2>/dev/null || echo "NO-EVIDENCE")"
        _qa_record FAIL "eph_tools_list_functional" "single tools/list call returns a well-formed, correlated response" "ok=true" "$obs"
    fi

    # ---- sequential baseline, then concurrent load, same N -----------------
    SEQ_JSON="${QA_RUN_DIR}/sequential.json"
    CONC_JSON="${QA_RUN_DIR}/concurrent.json"
    mcp_probe "sequential" "$CONCURRENCY_N" "$SEQ_JSON"
    mcp_probe "concurrent" "$CONCURRENCY_N" "$CONC_JSON"

    mcp_eval_concurrency "$SEQ_JSON" "$CONC_JSON" "${QA_RUN_DIR}/transcripts"

    # ---- full facts, not gated behind pass/fail (§11.4.6) ------------------
    {
        echo "# Concurrency measurement — full facts"
        echo
        echo "Commit under test: \`${COMMIT}\`"
        echo
        echo "Historical claim (the commit's OWN recorded measurement, NOT re-measured"
        echo "here — this run exercises the wire path only, and RegisterTool is"
        echo "unreachable from outside the process, see header note):"
        echo '```'
        git -C "$QA_REPO_ROOT" log -1 --format='%B' "$COMMIT" | sed -n '/^Verified:/,/^Independent/p'
        echo '```'
        echo
        echo "## This run's measurements (N=${CONCURRENCY_N} independent MCP clients each"
        echo "doing a real WebSocket upgrade + JSON-RPC initialize + tools/list)"
        echo
        echo '```'
        echo "-- sequential (one connection at a time) --"
        jq '{n, wall_clock_ms, ok: ([.results[]|select(.ok)]|length), err: ([.results[]|select(.ok|not)]|length),
             min_ms: ([.results[]|select(.ok)|.latency_ms]|if length==0 then null else min end),
             mean_ms: ([.results[]|select(.ok)|.latency_ms] as $l | if ($l|length)==0 then null else ($l|add)/($l|length) end),
             p50_ms: ([.results[]|select(.ok)|.latency_ms]|sort|if length==0 then null else .[(length*0.50|floor) as $i|(if $i>=length then length-1 else $i end)] end),
             p90_ms: ([.results[]|select(.ok)|.latency_ms]|sort|if length==0 then null else .[(length*0.90|floor) as $i|(if $i>=length then length-1 else $i end)] end),
             p95_ms: ([.results[]|select(.ok)|.latency_ms]|sort|if length==0 then null else .[(length*0.95|floor) as $i|(if $i>=length then length-1 else $i end)] end),
             max_ms: ([.results[]|select(.ok)|.latency_ms]|if length==0 then null else max end)}' "$SEQ_JSON"
        echo
        echo "-- concurrent (all fired together) --"
        jq '{n, wall_clock_ms, ok: ([.results[]|select(.ok)]|length), err: ([.results[]|select(.ok|not)]|length),
             min_ms: ([.results[]|select(.ok)|.latency_ms]|if length==0 then null else min end),
             mean_ms: ([.results[]|select(.ok)|.latency_ms] as $l | if ($l|length)==0 then null else ($l|add)/($l|length) end),
             p50_ms: ([.results[]|select(.ok)|.latency_ms]|sort|if length==0 then null else .[(length*0.50|floor) as $i|(if $i>=length then length-1 else $i end)] end),
             p90_ms: ([.results[]|select(.ok)|.latency_ms]|sort|if length==0 then null else .[(length*0.90|floor) as $i|(if $i>=length then length-1 else $i end)] end),
             p95_ms: ([.results[]|select(.ok)|.latency_ms]|sort|if length==0 then null else .[(length*0.95|floor) as $i|(if $i>=length then length-1 else $i end)] end),
             max_ms: ([.results[]|select(.ok)|.latency_ms]|if length==0 then null else max end)}' "$CONC_JSON"
        echo '```'
        echo
        echo "## Calibration method (§11.4.6 — no invented thresholds)"
        echo
        echo "- Throughput invariant: \`concurrent_wall_clock_ms < sequential_wall_clock_ms\`"
        echo "  — both numbers measured in THIS run, on the SAME server, moments apart."
        echo "- Latency-bound invariant: \`concurrent_p95_ms <= 3 * max(sequential_max_ms, 50ms)\`"
        echo "  — the multiplier and floor are explained in the script header; the"
        echo "  reference value (sequential max) is measured in THIS run, not assumed."
        echo
        echo "## Honest boundary (§11.4.6)"
        echo
        echo "This run proves the CURRENT build's tools/list path stays correct and"
        echo "fast under N=${CONCURRENCY_N}-way concurrent client load. It does NOT"
        echo "re-run the pre-fix artifact and does NOT reproduce the concurrent"
        echo "register-vs-list contention the fix's commit message describes, because"
        echo "RegisterTool has no client-reachable entry point in the shipped server"
        echo "(grep-verified, see grounding.txt) — that half of the fix is covered by"
        echo "the Go stress test cited in the commit message, not re-derived here."
    } > "${QA_RUN_DIR}/CONCURRENCY_ANALYSIS.md"
    qa_redact "$EPH_KEY"
    _qa_apply_redaction "${QA_RUN_DIR}/CONCURRENCY_ANALYSIS.md"

    qa_stop_ephemeral_server
else
    SKIP_WHY="${QA_EPH_SKIP_REASON:-ephemeral configured server unavailable}"
    qa_skip "eph_tools_list_functional" \
        "single tools/list call returns a well-formed, correlated response" "$SKIP_WHY"
    qa_skip "eph_concurrent_all_ok" \
        "every concurrent call returns a well-formed, correlated tools/list response" "$SKIP_WHY"
    qa_skip "eph_concurrent_faster_than_sequential" \
        "concurrent_wall_clock_ms < sequential_wall_clock_ms" "$SKIP_WHY"
    qa_skip "eph_concurrent_latency_bound" \
        "concurrent p95 latency <= 3x sequential max (floor 50ms)" "$SKIP_WHY"
fi

qa_finish
