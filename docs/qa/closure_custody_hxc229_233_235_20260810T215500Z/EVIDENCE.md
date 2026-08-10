# Closure-custody verification — HXC-229 / HXC-233 / HXC-235

Captured 2026-08-10T21:55Z. Main repo HEAD `4b597c16`. Verifier: independent agent,
not the author of any of the three fixes.

All three items read `Queued` in the SSoT (`docs/workable_items.db`) despite landed
work. This run establishes, per item and as FACT, whether the §11.4.146(D3) /
§11.4.226 closure-custody chain is complete:

    registry row keyed by the exact item id
      -> executable REGISTERED (wired) guard
        -> §11.4.115 RED + GREEN verdict pair
          -> class-matched captured evidence (committed, not merely captured)

"Class-matched" is load-bearing: all three defects are user-visible/runtime-layer,
so source-layer evidence (grep transcripts, diffs, build logs) cannot satisfy them.

## What "registry" means in THIS project (§11.4.35 instantiation)

There is no separate §11.4.108 runtime-signature registry artifact in this repo —
the anchor is referenced only in research/plan docs. The project's concrete
registry is the sweep's own gate registration, keyed by item id, in
`scripts/verify-all-constitution-rules.sh`. The sweep states the property that
makes this the registry (committed blob, G30-G32 block):

> There is no auto-discovery glob here — a guard is executed if and only if some
> caller names it — so all three had been written, reviewed, and left unreferenced
> by anything.

So: a guard file that exists but is named by nothing has NO registry row and is not
a standing guard. That distinction decides HXC-235 below.

## Live subject

Gateway `helixllm-gateway.service` active, MainPID **107268**
(`/home/milos/.local/bin/helixllm`), `https://localhost:8443/internal/health` = 200.
This is a DIFFERENT process from the one the 2026-08-08 independent review
interrogated (PID 2101934) — the service restarted 2026-08-08 22:08:59 — so the
invariants below are re-proven against the currently-running artifact, not
inherited from the review's session.

§11.4.174 note: a bare `pgrep -af helixllm` also matches the checking shell's own
command line. PID 107268 was confirmed by binary path, not by name match.

## Per-item results

### HXC-229 — gateway must serve in Gin RELEASE mode — CHAIN COMPLETE

| link | verdict | citation |
|---|---|---|
| registry row | PRESENT | `git show HEAD:scripts/verify-all-constitution-rules.sh` L1220 `if want_gate G30`, L64 names `HXC-229` |
| wired guard | PRESENT | L1224 invokes `scripts/testing/guard_hxc229_gateway_release_mode.sh`; file `-rwxr-xr-x`, committed, working tree clean |
| RED (broken artifact) | PRESENT | `docs/qa/hxc229_gin_debug_mode_20260807T000000Z/12_red_journal.txt` — PID 4140522 at 2026-08-07T12:52:13 logs `[GIN-debug] [WARNING] Running in "debug" mode` + the route dump |
| GREEN (fixed artifact) | PRESENT | `.../30_green.txt` (0 GIN-debug lines with a real startup marker) + this run's re-execution |
| evidence class | MATCHED (runtime) | `/proc/<pid>/environ` + journal of the live process; NOT a unit-file grep |

Re-run this session (`hxc229_guard_green.txt`, `hxc229_guard_red.txt`):

- GREEN polarity: **EXIT=0**, `PASS-with-caveat`, pid=107268 carries `GIN_MODE=release`
  read from `/proc/107268/environ`.
- RED polarity (`RED_MODE=1`): **EXIT=1** — "RED baseline did NOT reproduce ... This
  artifact already carries the fix." A RED that fails on a fixed artifact is the
  §11.4.115 polarity switch behaving correctly.

On the caveat (§11.4.6 honest boundary): the guard could not run its corroborating
route-dump cross-check because the journal window since the 2026-08-08 restart holds
437 lines but no startup marker (rotated). The guard explicitly declines to claim
that scan as evidence. The closure does not rest on it: the primary invariant is a
POSITIVE presence claim read directly from the running process's environment, which
needs no anti-vacuity backstop (anti-vacuity exists to stop an empty window
satisfying a negative absence claim). The caveat narrows corroboration, not the
runtime signature.

Blocking reason at the 2026-08-08 review was, verbatim in the diary, "no §11.4.135
standing regression guard was ever registered". That guard was subsequently written
AND wired as G30 in `be5d56be`. The blocking reason is therefore STALE, verified
against the committed blob rather than the working tree (the sweep is concurrently
being edited by another agent).

### HXC-233 — completion path must return a REAL generation — CHAIN COMPLETE

| link | verdict | citation |
|---|---|---|
| registry row | PRESENT | committed sweep L1244 `if want_gate G31`, L65 names `HXC-233` |
| wired guard | PRESENT | L1249 invokes `scripts/testing/guard_hxc233_completion_path_live.sh`; `-rwxr-xr-x`, committed, tree clean |
| RED / GREEN | PRESENT | see below |
| evidence class | MATCHED (runtime) | live HTTP completion returning real model output, content-asserted |

Re-run this session (`hxc233_guard_green.txt`, `hxc233_guard_red.txt`):

- GREEN: **EXIT=0** — `live completion answered "4" from model
  "/models/Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf" — real generation through
  https://localhost:8443/v1/chat/completions, not a status code.`
- RED (`RED_MODE=1`): **EXIT=1** — "RED baseline did NOT reproduce: the endpoint
  returned a real answer ... This deployment already carries the fix."

The guard asserts answer CONTENT from a named local GGUF model, so it cannot pass on
a bare HTTP 200, an empty body, or an error envelope. That is the correct runtime
signature for the reported defect (local model serving absent for the life of the
service).

Standing gap that this closure does NOT paper over: HXC-233 never went through an
independent closure review, and had zero diary rows. This verification is that
independent pass — performed by an agent that authored none of the fix — and the
diary row written from it is the record.

Supporting proof captured this session, `falsification_battery.txt`: the whole
live-service falsification battery run at HEAD `ca740516` reports **42 ok, 0 not ok**
(exit 0). It builds live stub servers and asserts the HXC-233 guard exits 1 on an
error envelope, an empty choices array, fluent-but-wrong content, a missing model
id, and five transport failure shapes — while SKIPping only on provable absence.
This is the §1.1 paired-mutation proof, executed rather than merely written.

On the RED link, stated precisely (§11.4.6). The captured RED
`docs/qa/hxc233_llama_server_absent_20260807T183000Z/02_pre_fix_live_curl.txt` is
runtime-class and committed, and shows the guarded invariant violated on the live
pre-fix deployment:

```
$ curl -sk https://localhost:8443/v1/models
{"object":"list","data":null}
... "brain error: all providers exhausted, last error: llamacpp: ..." HTTP_500
```

That is an error envelope with no models — vacuity shape #1, which the battery
proves the guard rejects with exit 1. So RED-on-broken → GREEN-on-fixed IS
demonstrated for the guarded invariant.

The honest residue, which this closure does NOT claim away: the root cause moved
between 2026-08-07 (llamacpp TLS/x509 against :8080) and the final routing fix
`15f00801` (`dial tcp 127.0.0.1:50052: connection refused`), and that later
connection-refused state was never captured as a raw artifact — it exists only as
prose in `hxc233_routing_fix_20260808T140500Z/FINDING.md`. The RED that IS captured
is an earlier failure mode of the same dead capability, not the exact state
immediately preceding the fix. The guard was also authored after the fix, so it was
never run in RED against a broken deployment; the falsification battery is what
supplies that proof instead, and it is the stronger instrument because it is
re-runnable on demand rather than a one-off historical capture.

Also recorded, not fixed here: `FINDING.md:73` ("No standing guard yet asserts the
completion path stays alive") and `evidence.md:155-162` ("not yet live") are both
stale relative to G31 and the deploy. And the ticket's literal text is still true —
`command -v llama-server` still exits 1 on this host; the capability was restored by
ROUTING to the separate `helixllm-coder` backend, not by installing the missing
binary. The tracker still proposes the superseded remedy. That description-integrity
defect is filed separately rather than silently corrected (see HXC-264 below).

### HXC-235 — semantic-embeddings signal — CHAIN INCOMPLETE, LEFT Queued

| link | verdict | citation |
|---|---|---|
| registry row | **MISSING** | no G-gate for HXC-235 anywhere in the sweep; G30/G31/G32 are the only guard gates |
| wired guard | **MISSING** | no `guard_hxc235_*.sh` exists (`ls scripts/testing/guard_*` = 229, 233, 244, falsification only); `grep -rn "HXC-235\|semantic_embeddings" scripts/` is EMPTY — instrument-validated, the same grep for 229/233/244 returns 30+ hits |
| RED (runtime) | PRESENT | `docs/qa/hxc235_deploy_20260808T135200Z/01_pre_deploy_response.json` — top-level keys `['object','data','model','usage']`, field ABSENT |
| GREEN (runtime) | PRESENT | `.../02_post_deploy_response.json` — keys include `semantic_embeddings`, value `false` |
| evidence class | MATCHED (runtime), committed | both probes tracked via `git ls-files` |
| live today | PRESENT | endpoint on PID 107268 returns `"semantic_embeddings": false` |

What DOES exist: Go guards at
`submodules/helix_llm/internal/gateway/hxc235_embedding_semantic_signal_test.go` and
`internal/knowledge/hxc235_semantic_signal_test.go`, with a real `RED_MODE` switch,
positive controls asserting `true` for a non-hash embedder, and a genuine paired
mutation (`hxc235_MUTATION_always_false_guard_FAILS.log`) proving they are
falsifiable. The fix is live and survived a later rebuild: the running binary
(mtime 2026-08-08 22:08:59) is NOT the one the deploy FINDING.md installed, and it
still emits the field.

**Why it is still not closeable.** The decisive point is not paperwork, it is
§11.4.226 evidence-class-at-closure. Those Go guards are `httptest.NewRecorder()`
source-class tests. The defect that actually blocked closure was a STALE DEPLOYED
BINARY — source correct, runtime wrong — and a source-class test would have reported
GREEN through that entire window. It cannot defend the layer where this defect
lived. On top of that, nothing schedules them: the sweep runs no `go test` against
`helix_llm`, and the runner that would (`scripts/release-gate-test.sh`) is itself
invoked by no Makefile target and no sweep — its only mentions are comments.

Per the sweep's own rule — "a guard is executed if and only if some caller names it"
— HXC-235 has no registry row and no wired standing guard. Two of the four chain
links are absent, so the status stays `Queued`.

To close it later, the missing link is specific and small: a
`scripts/testing/guard_hxc235_semantic_signal_live.sh` that probes the deployed
`/v1/embeddings` for the field (the runtime layer, not the source layer), registered
as a G-gate beside G30-G32, with a paired assertion added to the falsification
battery. That work was deliberately NOT done in this session: it edits
`scripts/verify-all-constitution-rules.sh` and `guard_live_service_falsification.sh`,
both of which another agent was concurrently remediating (landed as `ca740516`).
Writing a G33 into that file would have been a §11.4.84 collision. It is reported,
not attempted.

## Instrument validation (§11.4.201)

Every null in this run was validated before it was believed:

- `git grep` from the repo root does NOT traverse submodules. HXC-235 first appeared
  to have zero code references for exactly this reason; the gateway source lives in
  `submodules/helix_llm/`, and searching inside the submodule returns the real hits.
  A root-level "no references" result here is an artifact of the tool's scope, not a
  fact about the codebase.
- `pgrep -af helixllm` matches the checking shell itself (§11.4.174); process
  identity was established by binary path.
- The diary tooling's PASS-requires-evidence rule was positively probed on a
  throwaway DB copy, not assumed: a PASS with no `--evidence` is refused
  ("a PASS without captured evidence is a bluff"), and a PASS citing a nonexistent
  path is refused ("evidence must be real captured proof"). Enforcement is genuine
  at two layers — the CLI check and the `test_diary` schema CHECK constraint. No
  finding.

## Additional findings filed this session (coordinator scope)

Two findings were recorded in `ca740516`'s evidence files rather than filed, because
the workable-items DB was being regenerated concurrently and new rows would have
collided. Both were re-derived here rather than taken on trust, and neither is a
§11.4.214 recurrence of an existing id (checked: no open or closed item covers
either; the nearest matches, HXC-057 and HXC-261, are unrelated).

### F1 — TOCTOU inside the falsification guard (filed as HXC-262)

Source: `docs/qa/r5_review_remediation_20260811T024500Z/EVIDENCE.md:394-416`.
`ACTIVE_STATE` and `MainPID` are read by two separate `systemctl show` invocations,
so under load the two reads can straddle a `RestartSec=1` unit's very short `active`
window, yielding `active` with the pid already reaped — which falls through to the
"active but exposes no MainPID" SKIP. Observed once in 13 battery runs; 6/6 correct
in isolation. It did not reproduce in this session's run (`falsification_battery.txt`,
42 ok / 0 not ok), which is consistent with a ~1-in-13 intermittent and is NOT
evidence of absence.

Severity assessed as Medium, from measured blast radius rather than by default: it
is a §11.4.201(1) false-POSITIVE refusal (blocks on a condition that is not present)
rather than a missed defect, and it degrades to SKIP rather than to a false PASS — so
it cannot pass a broken artifact. What raises it above Low is the location: it sits
inside the battery whose entire purpose is falsification, and an assertion that
intermittently cries wolf trains readers to discount it.

### F2 — systemically unwired meta-tests (filed as HXC-263)

Re-derived at HEAD `ca740516`, since `ca740516` itself changed the figures the
coordinator quoted (3/14 and 1/14, measured before it landed).

Of the **14** files in `scripts/tests/`, exactly **2** have a real executable
invocation site — `sync_gate_direction_neutrality_meta_test.sh` (sweep line 513, the
one `ca740516` wired into G11) and `qa_evidence_citation_coincidence_meta_test.sh`
(sweep line 1076). Both are in the sweep. The other **12 are executed by nothing**.

A naive reference count returns 5, and that is the trap: three of those five are not
invocations at all — `cross_platform_parity_meta_test.sh` appears in a `continue`
skip-list inside its own gate, `obsolete_details_meta_test.sh` appears in a comment,
and `verify_qa_evidence_meta_test.sh` appears only in RESUME prose. A count is a
lead; the matched lines are the finding. Instrument validated: the same regex finds
20 `scripts/gates/` invocations in the sweep, and `helix_code/scripts/tests/...` at
sweep line 633 is a different directory, not one of the 14.

This is the HXC-253 class exactly — an assertion that exists but is unreachable
certifies nothing, and its presence is actively misleading because a reader counting
test files sees coverage that is never exercised.

### F3 — HXC-233's own tracker row carries three different defects (filed as HXC-264)

Discovered during this verification, not reported by anyone. Inside the single SSoT
row for HXC-233: `items.title` says the program "is not installed on the machine at
all"; `items.body_md` — rendered verbatim to `docs/Issues.md:497` — says it "is not
on its search path"; `items.description` says "no file called llama-server exists
anywhere". These are three materially different defects (absent binary vs PATH
scoping), and the proposed remedy in the body ("give the service an explicit path")
was superseded by the routing fix that actually shipped. This is not doc↔DB drift —
`Issues.md` faithfully renders `body_md` — the inconsistency is internal to the row,
which is §11.4.148 D2 territory.

Filed rather than silently corrected: rewriting a closed item's description to match
the fix that shipped would erase the record of what was originally reported.
