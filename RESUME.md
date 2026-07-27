# RESUME — HelixCode Session Handoff

**Revision:** 4
**Created:** 2026-07-08
**Last modified:** 2026-07-27
**Status:** active
**Maintainer:** CLI-agent main work stream
**Authority:** §11.4.131 Session-resumption file — point a fresh agent here. Composes §11.4.127 / §12.10 / §11.4.65 / §11.4.44.

---

## SHORT variant (one paste, first sentence)

> Read `RESUME.md` then `docs/CONTINUATION.md`, run `git fetch --all --prune --tags`, and continue the test-suite remediation on `main` at HEAD `cf26a173`. **Everything is COMMITTED AND PUSHED** to all upstreams as of 2026-07-28 (main + constitution + challenges + helix_qa + helix_agent; every push fast-forward, no force §11.4.113). Working tree is clean at every submodule depth, zero unpushed commits. The full sweep is **NOT green** — the tag is blocked. **The whole Helix platform is STOPPED and systemd-DISABLED as of 2026-07-27 evening — nothing auto-starts; you must boot infra before any infra-dependent test** (see "Session 2026-07-27 evening" below). Do NOT trust any "package ok" without checking the ambient-contention caveat in "Traps". Next action: settle the competing-orchestrator decision (task #9) — until it is settled, postgres:15432 cannot stay up and the precondition gate cannot pass. Force-push is absolutely forbidden (§11.4.113).

---

## FULL variant (paste-ready block)

> You are resuming the HelixCode test-suite remediation cycle. The prior session landed real fixes with captured RED→GREEN evidence, and also made two attribution errors that are corrected below — read the corrections before trusting any earlier summary.
>
> **Read first:** `RESUME.md` (this file), `docs/CONTINUATION.md`. Then `git fetch --all --prune --tags` and `git log --oneline HEAD..@{u}`.
>
> **State:** meta-repo `main` @ `66d6fb29`; helix_agent @ `36597718`; helix_qa @ `a729b51`; constitution checkout @ `b00ab1c`. NOTHING committed, nothing tagged, nothing pushed.
>
> **Terminal goal:** green full sweep → tag `helix-code-1.2.0-dev-0.0.1` on the main repo AND every owned submodule carrying changes, identical `helix-code-` prefix (§11.4.151), fast-forward-only to all upstreams. The operator authorised the tag **conditional on the sweep being green**. It is not green. Do not tag.
>
> **Anti-bluff is binding (§11.4):** every closure needs pasted runtime output; every gate touched needs a paired §1.1 mutation; no assertion may be weakened and no bare `t.Skip()` added to force green.

---

## Live-state anchors

| Key | Value |
|-----|-------|
| **Meta-repo HEAD** | `cf26a173` (branch `main`) — committed + pushed to github/gitlab/origin/upstream, all tips verified equal |
| **helix_agent** | `0165ab1d` — pushed to 4 remotes, all in sync |
| **helix_qa** | `88ef0579` — pushed to 5 remotes, all in sync |
| **challenges** | `072724af` — pushed to 5 remotes, all in sync |
| **constitution (checked out)** | `731bf1d3` — pushed to all 8 remotes; upstream merge brought **§11.4.235** (speed-first build-and-deploy) |
| **Release prefix** | `helix-code` (from `HELIX_RELEASE_PREFIX` in `.env`, mode 0600) |
| **Latest existing tag** | `helix-code-1.1.0-dev-0.0.3` → next is `helix-code-1.2.0-dev-0.0.1` |
| **Working tree** | ~470 modified files, ALL uncommitted |
| **Sweep logs (this cycle)** | `qa-results/full_retest/{helix_code_inner,helix_agent,helix_qa,verify_rules,verify_cascade}_20260727T13*.log` |
| **Pre-gofmt tree backup** | `<scratchpad>/helix_agent_pre_gofmt.patch` (3322 lines) |

---

## Operator decisions recorded 2026-07-27 (binding)

1. **Ports** — do NOT hand-assign. Use the **containers submodule's dynamic-port + service-discovery** capability; extend that submodule for anything missing (§11.4.74 extend-don't-reimplement) and cover the extension with all supported test types. All systems/services/infrastructure choose an available port at runtime, bind it, and expose it through discovery; services discover each other. This supersedes the earlier "which service moves off 8100" framing.
2. **Constitution pin** — advance to the checked-out HEAD **in this release**.
3. **Moving baselines** — fix the tests so they stop writing to tracked files (root-cause fix, not gitignore).
4. **Compose coupling** — **relocate `mcp_servers`** under helix_agent so no upward path traversal is needed.

---

## Landed this session, with captured evidence

- **gofmt across our Go code**: 507 → 0 unformatted. Vendored/archival trees deliberately EXCLUDED and reverted (see Correction 2). 431 format-only files.
- **Production nil-deref crash FIXED** — `internal/llm/llamacpp_provider.go` `GenerateStream`. `req, _ := http.NewRequestWithContext(...)` discarded the error, so the next line dereferenced a nil `*http.Request`. Fault address `0x38` == `offsetof(http.Request.Header)`, confirming it exactly. Guarded by `internal/llm/llamacpp_stream_malformed_url_test.go` (§11.4.115 `RED_MODE` polarity test). **Proven:** `RED_MODE=1` PASS pre-fix / FAIL post-fix; `RED_MODE=0` FAIL pre-fix / PASS post-fix; whole package `ok` 69.789s.
- **G8 gate reconciled** (§11.4.120) — was a FALSE POSITIVE, see Correction 3. Now PASS, with a 7-assertion paired mutation.
- **G13 now 82/82** — four honest Sources-verified footers, zero invented URLs, two recording explicit non-verification gaps.

---

## Session 2026-07-27 evening — additions, corrections, and shutdown state

**PLATFORM IS DOWN AND DISABLED.** All six user units stopped **and `systemctl --user disable`d**: `helix.target`, `helixagent.service`, `helixcode-server.service`, `helixcode-infra.service`, `helixllm-gateway.service`, `helixllm-coder.service` (+ the generated `helix-{pg,redis}-data-volume.service`). Verified `active=inactive enabled=disabled` on all six; ports 7061 / 8081 / 8100 / 15432 / 16379 / 18081 / 5433 / 6380 / 11434 / 18434 / 8001 all confirmed unbound. **Nothing auto-starts on next login** — re-enable deliberately before expecting any service.

Untouched by design (other projects on the shared host, §11.4.174): `helixterm-*` + `77c45f5c104d-infra`, `penpot-*`, `helix_sonarqube*`, and **`atmosphere-aosp-build`** (was running at shutdown — do not assume the host is idle).

### Landed — PORT_CONFLICT ownership fix (RED→GREEN captured)

`tests/precondition/containers_boot_test.go` + new `tests/precondition/port_conflict_ownership_test.go`. The gate held **two opposite §11.4.201 defects**:

- **Fail-closed:** probed `{port, port+10000}` and declared `PORT_CONFLICT` if both answered, with zero ownership check. Our stack publishes postgres on `15432 = 5432+10000`, so *any* foreign postgres on the standard port (here `helixterm`) produced a false failure.
- **Fail-open:** `findContainerInstances` ran `docker compose ps --filter name=…` (because `detectContainerRuntime` returns args `["compose"]`) — not a container-listing command. It returned nothing, so the duplicate-instance check **silently never fired**.

Replaced with runtime-driven ownership: `listRunningContainers` → `ourServiceInstances` (filters the `helixagent-` prefix) → `portConflict`. Evidence: `RED_MODE=1` reproduces (*"legacy blind probe flags a foreign listener as our own duplicate"*), `RED_MODE=0` GREEN, 3/3 both polarities, `ok dev.helix.agent/tests/precondition 0.005s`. The real-duplicate case is still caught (§11.4.120 — not weakened to a tautology).

### Correction 4 — gate status in the 1334Z log is STALE

Fresh sweep `qa-results/full_retest/verify_rules_20260727T184002Z.log`: **G8 PASS, G12 PASS, G13 PASS (82/82 footered)**. Only **G7** fails, and at **18** violations, not 24. Re-read that log before acting on any earlier gate claim.

### Correction 5 — the discovery "failures" were never code defects

`internal/discovery` `TestAllocatePort_RangeExhausted_WithEphemeral` / `TestConcurrentAllocations` / `TestConcurrentReleases` pass on a quiet host: `ok dev.helix.code/internal/discovery 0.013s`. The 1332Z run launched the helix_code (`133220Z`) and helix_agent (`133218Z`) sweeps **2 seconds apart**; helix_agent's stress/chaos saturated the ephemeral range `32768-60999`, so the tests' preferred ports `50000-50049` were all unavailable (forcing fallback into the 19-port `api` range → exactly the 16 unique ports logged) and `net.Listen(":0")` returned `EADDRINUSE`. Independent confirmation of Correction 1.

### NEW blocker — competing container orchestrators (task #9, architectural)

`helixagent.service`'s container adapter recreates `helixagent-postgres` from **`docker-compose.yml`, where postgres has NO `ports:` section** — so `15432` is never published. Manual `docker-compose.test.yml` runs DO publish it. With both in play postgres crash-loops: `could not bind IPv4 address "0.0.0.0": Address in use` (host `5432` is held by the unrelated `helixterm` stack). **While the service runs, the precondition gate cannot see 15432.** Compounding it, `scripts/ensure-infrastructure.sh` health-checks `${DB_PORT:-15432}` / `${REDIS_PORT:-16379}` but boots the default compose — it cannot satisfy its own health check (task #7).

Recommended (evidence-determined, reversible, NOT yet applied — needs the operator's call because it changes which stack owns the live platform's containers): point the infra boot at `docker-compose.test.yml` for postgres/redis, restoring the script's own declared port contract.

---

## Sweep results 2026-07-27 — NOT GREEN

| Stream | Verdict | Detail |
|---|---|---|
| helix_qa | GREEN | 142 ok / 0 FAIL / 23 no-test-files |
| governance cascade | GREEN | 277 PASS / 0 FAIL; 227 anchors discovered, all 6 carriers to §11.4.234 |
| helix_code inner | NOT-GREEN | 192 ok / 3 FAIL / 0 build-failed (285 pkgs, 0 cached) |
| helix_agent | NOT-GREEN | 355 ok / 23 FAIL (506 pkgs); `tests/integration` killed by 10-min timeout → PARTIAL |
| constitution rules | NOT-GREEN | 16 gates, 3 FAIL (G7/G8/G13), G14 SKIP (docs_chain engine absent) |

G8 + G13 are now fixed. **G7 remains failing by design** (see below).

---

## CORRECTIONS — read these before trusting earlier summaries

**Correction 1 — three concurrent sweeps contaminated the results (§11.4.119).** The prior session ran three full `go test ./...` sweeps simultaneously on one host. Captured proof of contamination: 112× `cannot assign requested address` and 8× `bind: address already in use`, with sibling sweep PIDs verified by cwd/argv. Test sweeps that bind ports are NOT parallelisable — partition by RESOURCE, not by repo. **Several reported failures are ambient-suspect, not proven code defects.** A serial, host-quiet re-run is REQUIRED before any of them are called real.

**Correction 2 — `docs/research/go-elder-plinius-v3` is NOT vendored.** It has no nested `.git` and no LICENSE; the nested repos under `docs/research/` belong to SIBLING directories (`Gandalf-Solutions`, `CL4R1T4S`, `LEAKHUB`, `AutoRedTeam`, `CLAUDE-CODE-SYSTEM-PROMPT`). Reverting gofmt there was still CORRECT, but for a different reason: `go list ./...` returns **0** packages from it — it is archival research intake outside the root module graph that no build/vet/test gate can validate.

**Correction 3 — G8 was a false-positive gate, not a documentation defect.** Both items already carried complete, truthful `Obsolete-Details`. The gate asserted `Triple-check:`; canonical (`constitution/Constitution.md:7687`) and the generator (`obsolete.go:187`) both say `Triple-check evidence:`. That markdown is machine-generated from the SQLite SSoT — hand-editing it would have broken **G11** (byte-identical md→db→md round-trip) and been reverted at next regeneration.

---

## G7 (§11.4.83) — 24 violations split THREE ways

- **Class C (6) — FALSE violations.** Evidence exists; the gate cannot match it. It requires the COMMIT SUBJECT to contain the evidence directory basename, but run-ids are timestamped slugs (`hxc135_20260712T130921Z`). Affected: `3e67fa1a`, `54a76c3c`, `ca76e14b`, `7210f373`. Two more (`02e9e505`, `1254e0a6`) committed evidence as FLAT `docs/qa/*.md` files, invisible because the gate enumerates only directories.
- **Class B (3)** — genuine non-features tripping the heuristic: `d6c05f76` (README only), `f9dcf6a6` (doc-comment only), `b058c7c2` (`const`→`var` test-robustness, prod default unchanged).
- **Class A (15)** — genuine features shipped with NO evidence anywhere, **including 3 security fixes**: `4727a9d0` (CORS wildcard+credentials), `9c876819` (CSWSH `/ws` auth), `2ff55c31` (wire-facade 401). Full list: `fbfffd7d 0e3bb747 6efadd15 225cdf77 aa6b20b4 eb233785 f8c38181 67c9a9bc 4727a9d0 9c876819 2ff55c31 a21ad7ca 51c058b1 66f9c21e c9bad26a`.

**HARD CONSTRAINT:** the `[no-qa-evidence]` opt-out lives in the COMMIT MESSAGE and all 24 are already pushed. Retroactive application requires history rewrite — **forbidden absolutely** (§11.4.113 / CONST-043). The opt-out is available only going forward.

**Class A is an OPERATOR DECISION, not yet made:** retrospective verification runs against real infra (legitimate — the transcript records a real run performed now; security fixes first) VS a documented baseline bump like the gate header records for 118 commits on 2026-06-22. Fix Class C FIRST — bumping before that hides a real matching bug and overstates the debt by 6.

---

## Open defects — confirmed, NOT yet fixed

- **IPv6 bracket-unawareness — TWO independent bugs, one class, neither a regression** (no shared helper; both non-bracket-aware since creation):
  - `helix_code/internal/llm/response_err_round53_test.go:869` `mustSplitHostPort` re-joins the output of `net.SplitHostPort` (which strips IPv6 brackets by design) with plain concatenation → `http://::1:PORT`.
  - `helix_agent/internal/vectordb/qdrant/client_mock_test.go:31-35` uses `strings.Split(url, ":")` on a bracketed authority → `host == "["`, and a discarded `Sscanf` error silently defaults the port to 80 → `http://[:80`.
  - Trigger for both: `httptest.NewServer`'s IPv6 fallback when IPv4 loopback bind fails. **These tests are environment-flaky by construction.**
- **i18n key leak — LATENT, not live.** `tool_schema` ships a consumer-injected translator seam; **nothing anywhere calls `toolschema.SetTranslator`**, so `activeTr` stays `NoopTranslator` and `tr()` echoes the key. Bundle entry EXISTS and is correct (`active.en.yaml:50` → `"Create git commit"`). No production path currently reaches it, so no user sees it — but it arms on first use, and all 36 `tr()` sites include user-visible `ToolResult.Error` payloads. **Do NOT "fix" the test to expect the raw key** — that cements the unwired state and disarms the only cross-module check.
- **HXC-131 evidence path is dead** — `docs/qa/followup_fixes_20260712T085616Z/HXC131_evidence.md` never existed (that commit added only `HXC133_evidence.md`). Genuine evidence IS at tracked `scratch/discovery/fixes/HXC131_evidence.md`. Correct repair is a DB mutation on the SSoT (`workable-items obsolete-details --evidence …` + re-render), not a doc edit.
- **`sdk/web/node_modules/flatted/golang/pkg/flatted/flatted.go`** — genuinely third-party vendored code sitting INSIDE the gofmt gate. Clean today; an npm bump reintroduces the tension.
- **`go vet ./...` covers only the root module** — 36 nested modules (ours included) are never vetted. Real coverage hole, own item.

---

## In flight at handoff (§11.4.147 — a crashed agent is NOT complete)

Four background agents dispatched; results NOT yet collected or verified:
1. Independent review (§11.4.142) of the llamacpp fix + its RED test.
2. G7 Class-C matching-bug reconciliation + paired mutation.
3. Extend the discarded-error guard to 4 sibling sites (`internal/verifier/embedded_server.go:84`, `together/client.go:72`, `replicate/client.go:87,129`), TDD RED-first.
4. Apply the prepared carrier elision patch (48 entries, 6 carriers) + verify §11.4.157 lockstep + regenerate exports.

If any did not land, re-dispatch — do not assume completion.

---

## Traps that already cost time — do not re-learn these

- **A "package ok" may be ambient, not repair.** `internal/llm` reported `ok` after the crash fix — including 3 round-53 tests that had failed. Those pass because `httptest` bound IPv4 that run, NOT because the IPv6 defect was fixed. It is unfixed. Green here is environment.
- **The legacy summary generators DESTROY data.** `scripts/generate_{issues,fixed}_summary.sh` read the TEXT trackers; committed summaries are SQLite-derived (§11.4.93/.95). Running them rewrites 344 items down to 188 — 156 items of tracked state destroyed.
- **`make test` can never pass on a headless host** — runs `go test ./...` with no `-tags=nogui`.
- **A "completed / exit 0" notification describes the WRAPPER, not the work.** Verify the artefact — mtimes, `pgrep`, pasted output.
- **`replace_all` is unsafe when a NEW helper's body contains the pattern being replaced** — rewrites the helper into infinite self-recursion.
- **PDF export silently drops wide tables' rightmost column** (§11.4.168) while reporting `rendered=2 failed=0`. Keep load-bearing evidence out of wide tables.
- **`git ls-files '*.go' | xargs gofmt -w` reaches vendored AND archival trees.** Always exclude `docs/research/` (and check `node_modules`) before formatting.
- **`podman pod rm -f pod_helix_agent` KILLS `helixagent.service`.** Its container adapter sees the pod vanish and shuts down gracefully (`"Shutting down container adapter..."`, `status=0/SUCCESS`) — it looks like a clean exit, not a casualty. Cost this session: `chaos/api` + `chaos/auth` regressed `ok` → `connection refused` on `:7061`, and a whole 30-min sweep ran against a dead server. Stop the *service* first, or don't touch its pod.
- **`pgrep -f 'go test'` matches YOUR OWN shell** — the loop's command line contains the literal string. It killed my own bash mid-command (exit 144). Always exclude `$$`/`$PPID` and verify by `cwd` (§11.4.174 / §11.4.201 carrier footgun; §12.12 names this exact class).
- **`:0` failing with `EADDRINUSE` is the ephemeral-exhaustion tell.** Binding port 0 asks the kernel for *any* free port; it can only fail when the range is genuinely exhausted. That single line is the fastest way to distinguish host contention from a real allocator bug.
- **The test stack and the platform stack use DIFFERENT ports.** Precondition demands `15432`/`16379`/`18081` (`docker-compose.test.yml`); the live platform uses `5433`/`6380` (`helixcode-infra-*`). Seeing `5432`/`6379` listening proves nothing about test readiness — those belong to another project.

---

## Sweep commands (exact) — RUN SERIALLY, ONE AT A TIME

```bash
# Host must be quiet. Three of these in parallel exhausts IPv4 ephemeral
# ports, forces httptest onto its IPv6 fallback, and produces failures that
# are contention artefacts rather than code defects (§11.4.119).
cd helix_code            && go test -tags=nogui -count=1 ./...
cd submodules/helix_agent && go test -count=1 ./...
cd submodules/helix_qa    && go test -count=1 ./...
./scripts/verify-all-constitution-rules.sh
./scripts/verify-governance-cascade.sh
```

`-tags=nogui` is REQUIRED for the inner module (headless host, no X11/GL dev headers).
`-count=1` is REQUIRED — an earlier run reported 250 `ok` of which 249 were `(cached)`.

---

## Binding constraints (restated — §11.4.131(C))

- **Anti-bluff §11.4** — every PASS needs captured runtime evidence; metadata-only / config-only / absence-of-error / grep-without-runtime PASS are critical defects.
- **§11.4.113 — force-push is absolutely forbidden**, no exception, no operator override. Integrate by merging onto latest `main`, push fast-forward only.
- **§11.4.151** — tags carry the `helix-code-` prefix, identical across main repo and every owned submodule in one release.
- **§11.4.142** — every change needs INDEPENDENT review before commit/build; author self-verification (§11.4.92) precedes but never satisfies it.
- **§11.4.84** — account for every modified file before `git add`; unaccounted entries → ABORT. Never `git add -A` here.
- **§11.4.119** — one owner per exclusive resource; partition parallel work by RESOURCE, not by repository.
- **§11.4.185** — manual QA-team confirmation is the final sufficiency gate; not self-certifiable.
