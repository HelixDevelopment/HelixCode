# HelixCode Release Changelog — helix-code-1.2.0-dev-0.0.1

**Revision:** 3
**Last modified:** 2026-08-09T12:38:49Z
**Maintainer:** HelixCode release engineering (documentation draft, AI-agent authored; independently reviewed per §11.4.142/§11.4.194 — see "Independent review" section below)

---

> **DRAFT — pinned to a stated HEAD, this is not a cut tag.** No git tag has been
> created by this document. Everything below was derived by reading real git
> history (`git log`, `git show`) — revision 1–2 content on 2026-07-28, the
> revision 3 content on 2026-08-09 at 12:38 UTC.
> Multiple background work streams were actively landing commits in this repo
> and its submodules while this draft was written (constitution §11.4.103 /
> §11.4.192 continuous multi-track work), so HEAD will have moved by the time
> this is read. **Re-derive the commit ranges below before treating any of this
> as final** — do not copy the SHAs or counts forward without checking.
>
> | Repo | Stated HEAD (revision 3) | Relative to its configured remotes |
> |---|---|---|
> | main (this repo) | `5725ee068c965dabf2720ff6063acf3753a6c232` | 3 ahead, 0 behind on **all four** remotes (github/gitlab/origin/upstream) — measured *after* a real `git fetch --all --prune`, not from stale tracking refs (§11.4.6) |
> | `constitution` (pin) | `177f2b05545f2fe73fbb432ad0de3a6bed7edcf4` | clean; carries §11.4.236/§11.4.237/§11.4.238 |
> | `submodules/helix_llm` (pin) | `3d789cb888a09cfecdd2f0007a08a96d91caf5e9` | clean; **verified to contain `8260cf8`**, the HXC-244 health fix, so a fresh clone gets it |
> | `submodules/helix_agent` (pin) | `66a3c1c6133a0ebdfeaa2ca4f1231588467cd145` | working checkout is at `d6856cc9` and **differs from the recorded pin** (`git submodule status` shows `+`); three other agents were actively editing this submodule and it was deliberately not touched by revision 3 |
> | `submodules/tool_schema` (pin) | `8cec90baadad281eb26a5671b792921f03eb7386` | clean |
>
> **Revision 3 scope.** Revisions 1–2 documented `5ab97d8c..c29e1dcc`. Revision 3
> re-derives from `c29e1dcc..5725ee06` — a further **231 commits** spanning
> 2026-07-28 → 2026-08-09, of which exactly **2** are the literal `Auto-commit`
> message and 229 are substantive; **0 merges** anywhere in the release range.
> Total since the last tag `helix-code-1.1.0-dev-0.0.3` (`5ab97d8c`): **249
> commits** (18 + 231, and `git rev-list --count 5ab97d8c..HEAD` independently
> returns 249). The revision 1–2 content below is **preserved, not rewritten** —
> it was independently reviewed and its claims re-verified; revision 3 adds the
> new range, corrects one counting error revision 2 introduced (see "Independent
> review"), and refreshes every pin and remote position.
>
> **Format note:** `docs/changelogs/` did not exist in this repository before this
> draft — it was created now. The repository's existing root `CHANGELOG.md` uses
> a `## [tag] — date` per-release section with Added / Changed / Fixed /
> Known-gaps groups; this draft follows that convention but adds explicit
> short-SHA citations per entry and a per-repo commit inventory, so every claim
> traces back to a real, readable commit (constitution §11.4.6 — no invented
> history). Per the task scope, nothing here was committed with `git add -A` or
> similar — only this one file is staged, by explicit path.

## Version

**`helix-code-1.2.0-dev-0.0.1`** — draft, not yet tagged.

Prefix resolves to `helix-code` from `.env` `HELIX_RELEASE_PREFIX` (§11.4.151).

Prior tags in this lineage:

| Tag | Date |
|---|---|
| `helix-code-1.0.0-dev-0.0.1` | 2026-07-08 |
| `helix-code-1.1.0-dev-0.0.1` | 2026-07-09 |
| `helix-code-1.1.0-dev-0.0.2` | 2026-07-13 |
| `helix-code-1.1.0-dev-0.0.3` | 2026-07-24 |

**On the version jump:** `helix-code-1.2.0-dev-0.0.1` is a **1.1.x → 1.2.0 minor
bump**, not the next `-1.1.0-dev-0.0.4` patch in the existing sequence. This was
named explicitly by the operator as this release's target tag; per instruction
for this task, that is recorded here as an **operator decision**, not
re-litigated or treated as unexplained drift.

## Summary — what this release actually delivers for a user

- **Model identity is now honest end-to-end.** A client asking for an alias
  (e.g. `"model":"default"`) previously got that same alias echoed back in the
  response, on both the OpenAI-compatible and Anthropic-compatible wires,
  streaming and non-streaming. The response now names the model that actually
  served the request (e.g. the real `.gguf` path), matching CONST-036/037.
  Live SSE evidence was captured against a real running coder backend for both
  wires. One honest exception remains: Anthropic's streaming `message_start`
  event fires before any chunk exists, so it still carries the requested model
  — documented in-source, not silently left as a gap.
- **`tool_schema` translations now work regardless of where you run from.**
  Previously, calling into this library's i18n from any working directory
  other than its own module root silently returned raw translation keys
  instead of translated text (e.g. `toolschema_git_desc_commit` instead of
  "Create git commit"). Bundles are now compiled into the binary
  (`go:embed`), so this can no longer happen.
- **A large batch of false test failures in `helix_agent` is fixed.** A
  liveness check used to accept *any* HTTP response under 500 as "the server
  is up" — so when a different local service (LLMsVerifier) held the same
  default port, its 404 response was read as a healthy HelixAgent, and roughly
  a dozen test packages failed against the wrong service with nothing
  actually wrong in the product.
- **The whole platform can now be installed and booted as systemd units**
  (`./setup.sh`) — HelixCode server, HelixAgent, HelixLLM gateway + coder,
  infra (Postgres/Redis/Weaviate/Qdrant/ChromaDB/Cognee/Ollama), and now
  LLMsVerifier — with real HTTP health checks and a verified stop/start
  cycle, rather than requiring manual, host-specific setup.
- **LLMsVerifier is wired to real provider data and publishing real scores**
  instead of an empty `{}` — the gateway now routes on measured scores
  instead of always falling back to its static table.
- **`go test ./...` no longer breaks on generated challenge output** — a
  gitignored-but-still-scanned results directory could fail three otherwise
  unrelated packages depending on what an LLM happened to answer during a
  challenge run.
- **`helix_agent` boot no longer requires ChromaDB/Cognee on their compiled-in
  ports**, and no longer hangs indefinitely trying to start a vector store
  that is already running under a different owner (committed in `helix_agent`
  but see the pinning note below).

## Fixed

| SHA | Repo | Summary |
|---|---|---|
| `e330149f` | main | LLM response now reports the model that actually served the request (OpenAI + Anthropic wires, streaming + non-streaming), not the requested alias. Four layers: `llm.LLMResponse.Model` field + JSON round-trip, `OpenAICompatibleProvider` populating it from both non-stream and per-chunk stream responses, and both wire facades preferring the served model. Guards at `internal/server/wire_facade_model_identity_test.go` and `internal/llm/openai_compatible_stream_model_test.go`, each proven to fail when its fix is reverted. |
| `66d6fb29` | main | `go test ./...` no longer descends into `tests/e2e/challenges/test-results/` (renamed to `_test-results/`, the Go toolchain's own "not a package" convention) and tries to compile an LLM's prose answer as Go source. Unblocked 3 packages. |
| `3268fd73` | helix_agent *(pinned)* | `testutil.ServerAvailable` now requires a real HelixAgent-shaped health payload (HTTP 200 + identity check), not just "any status under 500". Fixes a false-failure cluster of roughly a dozen packages caused by a different local service (LLMsVerifier) answering on the same default port. Three additional call sites that bypassed the shared helper entirely (a raw TCP dial, an error-only skip check) were also fixed. |
| `0165ab1d` | helix_agent *(pinned)* | Precondition gate's container-ownership check fixed in both directions: it no longer flags an unrelated third-party Postgres as "our" duplicate instance (false positive from `{port, port+10000}` guessing), and it no longer silently no-ops because it ran the wrong runtime command (false negative). Ownership is now established from the actual container runtime, filtered to this project's own containers. |
| `0dfa69e1` | helix_agent *(pinned)* | HelixAgent boot no longer hardcodes ChromaDB (`:8001`) / Cognee (`:8000`) endpoints — both resolve via `HELIXAGENT_DEP_HOST` / `HELIXAGENT_PORT_CHROMADB` / `HELIXAGENT_PORT_COGNEE` with the old literals as fallback. Vector-store startup now checks reachability first instead of unconditionally trying to become a second owner of an already-running store, which had been blocking the HTTP listener bind indefinitely. |
| `8cec90b` | tool_schema *(pinned)* | Translation bundles are `go:embed`-ed, so `tr()` lookups no longer depend on the process's current working directory. Previously any consumer calling in from outside this module's own root got raw key strings back instead of translations. |
| `69a5ab8d` | helix_agent **(committed, NOT yet pinned — see Known gaps)** | Fixes a goroutine leak in the Ollama provider's `CompleteStream`: 4 of 6 channel-send sites had no `ctx.Done()` escape hatch, so a caller that cancelled its context and stopped draining left the background goroutine parked forever. RED/GREEN regression guard added, driven via a real stuck-HTTP-response scenario and a live goroutine-stack inspection. |
| `a345c551` | helix_agent **(committed, NOT yet pinned — see Known gaps)** | Two `tests/unit` test defects fixed (product behavior was correct in both cases): `TestCompletionHandler_Models` now wires a real model source instead of asserting on an intentionally-empty list; `TestCompletionHandler_Stream` now distinguishes an honest empty response (client-gone, request-timeout path) from an actual failure, instead of treating the httptest recorder's default `200` as an unconditional pass signal. |
| `dd3c0c3b` | main | Closes the third confirmed instance of an "unprioritized `select`" race this session: `autoSaveLoop`'s per-tick `select` between `<-ticker.C` and `<-stop` had no priority ordering, so a coincidentally-pending tick could still fire one more `SaveAll()` after `DisableAutoSave()` had already closed `stop` — the cause of `TestAutoSaveLifecycle_ReEnableTicks`'s "disabled auto-save must not tick" flake. Fixed by extracting the per-iteration select into `autoSaveTick(tickerC, stop)` with a non-blocking priority pre-check on `stop`. Full RED/GREEN/pre-fix/post-fix polarity evidence captured (§11.4.115): 2488/5000 forced-race hits pre-fix with the priority check disabled, 0/5000 post-fix. **Found missing from this changelog's original draft during independent review — added in revision 2; see "Independent review" section below.** |

## Added

| SHA | Repo | Summary |
|---|---|---|
| `c4115c73` | main | `./setup.sh` now installs and boots the entire platform as systemd units (`helix.target` + 5 services) — HelixCode server, HelixAgent, HelixLLM gateway, HelixLLM coder, and infra — with a documented port map, boot-persistence via `Linger=yes`, and a verified full stop/start cycle. Fixes a `KillMode=control-group` bug that had been silently killing all ten infra containers' port forwarders whenever the infra unit's own health check failed. |
| `d3e2c6e7` | main | Adds LLMsVerifier as a 7th systemd unit, wired to the gateway. Implements the `/api/scores` endpoint upstream in the submodule (project-agnostic; consumer supplies the port via the unit). Gateway's log changed from "verifier unreachable" to actually reaching the verifier and correctly falling back on an empty score map. |
| `98db7d79` | main | LLMsVerifier is now seeded with a real, reachable provider (`llamacpp` → the local llama.cpp coder server) via a consumer-owned `config/llmsverifier/config.yaml`, instead of the submodule's own empty `llms: []` default. First real verification score published (0.96) from genuine inference, replacing the previously-empty `/api/scores`. |
| `410afda5` | helix_agent *(pinned)* | Hyper provider discovery, registry, and verifier types. |
| `1296d328` | main | Adds `scripts/setup.sh` scaffolding (157 lines) and two systemd unit files (`helixcode-infra.service`, `helixllm-gateway.service`) — precursor work later built out by `c4115c73` / `d3e2c6e7`. Commit message is the literal `Auto-commit`; see Known gaps. |

## Changed

| SHA | Repo | Summary |
|---|---|---|
| `925aa859` | main | Backfills the constitution's §11.4.182 TRUNK RULE (work on trunk is always labelled Track 1) across this project's five root governance carriers, and bumps the `helix_agent` and `tool_schema` submodule pins to the commits described above. |
| `f14a3f89` | main | Bumps the `helix_agent` submodule pointer (Hyper provider). |
| `565506f6` | main | Bumps HelixLLM to `helixllm-0.2.0` (CPU inference). |
| `c20215bd` | helix_agent *(pinned)* | Chaos/compliance/automation test hardening carried in from prior sessions and committed so it is not lost — chaos/api, chaos/auth, module-compliance, config-generation automation, plus new `helix_endpoint` contract tests. `go build ./...` verified exit 0; the commit itself is explicit that the full suite was not independently reviewed or proven green at the time. |
| `4b5f2867` | helix_agent *(pinned)* | `gofmt` alignment and comment-spacing pass across the module. |
| `36597718` | helix_agent *(pinned)* | Search test reconciled to a reordered `UnknownType` check; adds coverage for the reachability short-circuit. |

## Infrastructure / Governance / QA evidence

| SHA | Repo | Summary |
|---|---|---|
| `578de68d` | main | Lands §11.4.83 end-user QA evidence for 7 feature commits, captured against a freshly rebuilt binary (111 assertions, 0 FAIL, 0 SKIP). Explicitly documents that an earlier evidence pass had validated a 15-day-stale binary and was therefore invalid and re-run. |
| `1c9fdcf2` | main | §11.4.83 evidence against live paid providers (Cerebras, Cohere) and a real MCP wire for 3 further commits; drops the qa-evidence-gate violation count from 6 to 3. |
| `e952f4d1` | main | §11.4.83 evidence for the last 3 non-feature commits (CORS allowlist, Fyne desktop build docs, cognee determinism), each carrying an explicit "honest scope statement" about limited/no end-user surface. **Drives the qa-evidence gate (G7) from 13 to 0 violations across 28 feature-shipping commits.** |
| `d8c44618` | main | §11.4.83 evidence specifically for the `e330149f` model-identity fix (8/8 assertions PASS, mutation table proving each guard fails when deleted). |
| `a5462e35` | main | Live SSE proof, against the real coder backend, that `stream:true` reports the served model on every chunk (13 frames captured, 1 distinct model id, 0 frames carrying the alias). |
| `c29e1dcc` | main | Live proof for the Anthropic wire (`POST /v1/messages` returns the real served-model path) and for the `tool_schema` i18n fix (a compiled consumer binary resolves translations from unrelated working directories). |
| `cf26a173` | main | Bumps 4 owned submodule gitlinks (constitution, challenges, helix_qa, helix_agent) after pushing them to their upstreams. Commit itself states the full test sweep was **not** green at the time and no tag was cut. |
| `0a4eb8d0` | main | Corrects stale live-state anchors in `RESUME.md` after the above push — the exact "stale handoff doc" trap the project's own governance warns about, caught and fixed in the same session. |
| `56f7edf7` | main | Message is the literal `Auto-commit` (no descriptive body). Diffstat shows real, substantive content: nil-request guards for the Replicate and Together LLM providers, a malformed-URL guard for the llamacpp provider, a `security_scan` tool rewrite, and an overhaul of `verify_qa_evidence.sh` / `verify-governance-cascade.sh` / `obsolete_details_gate.sh` / `summary_sync_gate.sh`, plus a `helix_agent` pointer bump. See Known gaps. |

## Revision 3 — the `c29e1dcc..5725ee06` range (231 commits, 2026-07-28 → 2026-08-09)

Everything from here to "Per-repo commit inventory" was derived on 2026-08-09.
Commit-type histogram over the 229 substantive commits: 87 `docs`, 74 `fix`,
27 `chore`, 15 `test`, 9 `feat`, 7 `items`, 3 `track`, 2 `governance`, 1 `probe`,
1 `docs+test`. 72 distinct HXC ids are named in commit subjects (HXC-151…HXC-248).

### Summary — what changed for a user in this range

- **The gateway's primary capability was dead and now works.** Every completion
  returned HTTP 500 "all providers exhausted": with no cloud keys configured the
  local provider was the only completion path, and it was pointed at port 50052
  — a default matching no service in the deployment, while the local model is
  served on 18434. Completions now return 200 with real model output.
- **The health endpoint no longer reports a verdict it never earned.** It
  returned `{"status":"healthy","components":[]}` and could return nothing else.
  It now names four checked dependencies, each with a measured round-trip
  duration.
- **You can now tell semantic search from a hash fallback.** The embeddings
  response previously carried no signal at all about which pipeline served it.
- **The desktop application no longer corrupts its own UI state under load** —
  265 data races eliminated, root-caused to a *false* code comment claiming a
  widget setter was goroutine-safe when the toolkit's own source says the
  opposite.
- **Streaming-with-tools no longer deadlocks** — a guaranteed (not racy) hang,
  plus a family of provider send-leaks that stranded a goroutine and an open
  HTTP body whenever a client disconnected.
- **Infrastructure "started" now means "serving", not "created"** — the boot
  path previously reported success once containers existed, before anything
  answered.
- **Two credential exposures were found and handled**, one of which had been
  published on all four mirrors for 48 days.
- **Test suites that could not fail were made falsifiable** — 16 HelixQA banks
  covering 72 steps declared no assertions at all and scored PASS against a
  dead target; 238 steps now carry explicit expectations.

### Fixed (revision 3 range)

| SHA | Repo | Summary |
|---|---|---|
| `15f00801` | main | **HXC-233** — completions were returning HTTP 500 "all providers exhausted" on every request. `LocalRPCPort` defaults to `50052` and the live process had the env var unset, so the gateway dialled a port no service listens on while `helixllm-coder.service` served the model on `18434`. Fixed with two `Environment=` lines placed *before* `EnvironmentFile=` so an operator `.env` can still override. Verified: `:50052` connection refused → after fix, HTTPS 200 naming the real `.gguf`. Findable only because HXC-235's deploy an hour earlier replaced a misleading TLS error with an honest one. |
| `fee7e1bc` | main | **HXC-233 guard** — `scripts/testing/guard_hxc233_completion_path_live.sh`, asserting a *real* assistant message and a model id, not a status code. Its load-bearing proof is Mutation B: a stub returning HTTP 200 with an error envelope makes the guard exit 1, which a status-only assertion would have passed. |
| `70cb8821` | main | **HXC-244 RED baseline** — establishes as fact that `cmd/helixllm/main.go:148` builds `health.NewChecker()` and registers nothing; of the 41 `.Register(` call sites in the tree, zero are on the health checker. The guard deliberately asserts the **component list**, not `status=="healthy"` — that assertion passed on the broken build for the wrong reason. Two defects in the guard itself were caught by its own golden-good/golden-bad fixtures (§11.4.107(10)): an invalid f-string made the analyzer crash on every input, and the crash's exit 1 read as "defect found", so revision 1 reported RED-confirmed against a *healthy* fixture. Fixed structurally — analyzer findings start at exit 2, and any exit outside {0} ∪ {2..6} is a broken instrument and hard-fails in both polarities. |
| `262a44ad` | main | **HXC-244 closed at the RUNTIME layer.** The source fix (`helix_llm 8260cf8`, 969 insertions across 9 files with tests) had merged but never reached production: the running binary's mtime predated it and `go tool nm \| grep -c registerHealthChecks` returned **0**. Closed by rebuild + redeploy of the gateway only — §9.2 backup first, artifact verified *before* install (`registerHealthChecks` 6, `Ping` values 4), `NRestarts` stayed 0. After: `llm_providers`, `kv_cache_redis`, `vector_store_qdrant`, `llms_verifier`, each carrying a measured `duration` — the evidence of a real round-trip that distinguishes this from a hardcoded list. |
| `f2e12570` | main | **HXC-235 deployed** — the deployed binary was **15 days stale**, containing `semantic_embeddings` zero times, so four `helix_llm` commits (HXC-233's fix among them) were not live either. The process-start time had suggested a 13-hour gap; `strings` on `/proc/<pid>/exe` gave the real answer in one command. Post-deploy the key appears with value `false`, which is *correct* (`HELIX_EMBEDDING_PROVIDER` is unset, so the pipeline genuinely is the HashEmbedder) — the defect was that a caller could not tell. |
| `724d2bb0` | main | **HXC-229** — the gateway ran in its web framework's debug mode under systemd. `Environment=GIN_MODE=release` added to the unit; verified on the running process via `/proc/<pid>/environ`, with 0 debug lines in a window provably containing a real startup. |
| `e879702c` | main | **265 data races** eliminated in `applications/desktop`. Root cause: a code comment asserting `(*widget.Entry).SetText` is goroutine-safe, which the toolkit's own `thread.go` has contradicted verbatim since v2.6. RED 265 `WARNING: DATA RACE` exit 1 → GREEN 0 races exit 0 at `-count=3`. |
| `905a0b0a` | main | A **guaranteed** `StreamWithTools` deadlock (not a race): `GenerateStream` was inlined at both call sites writing into a 100-buffered channel nobody drained. Plus six provider files with unguarded sends leaking a goroutine and an open HTTP body on client disconnect. |
| `dd3c0c3b` | main | Third confirmed "unprioritized `select`" race — `autoSaveLoop` could fire one more `SaveAll()` after `DisableAutoSave()` had closed `stop`. 2488/5000 forced-race hits pre-fix, 0/5000 post-fix. *(Documented in revision 2 but filed under the wrong range — see "Independent review".)* |
| `0bcd2e98` + `6d8a4920` | main | **HXC-228** — infra "started" now means "serving" rather than "created"; verified by a full-target cold boot with zero restarts. |
| `11861996` | main | **HXC-168 / CONST-042** — a database password carried as a literal in shipped setup and container files published to all four mirrors. Reachability was *established, not assumed*: two compose files published Postgres on all interfaces, and the autoboot one boots by default. Explicitly does **not** close the item — rotation is outstanding. |
| `be599c10` + `ce5ecc04` | main | **HXC-167** — transcript redaction was fail-open and allowlist-only; three further fail-opens found, one of them *inside the guard itself*. |
| `e6118f99` | main | Dropped inverted `docs_chain` derive edges that would have destroyed **156 tracked items**. |

### Added / Changed (revision 3 range)

| SHA | Repo | Summary |
|---|---|---|
| `ae218ee3` | main | Cascades **§11.4.236** (QA-deploy-readiness gate — no manual-QA hand-off until the mandated validation produced a candidate-fingerprinted PASS verdict; a blocker must bind to a seam, never prose) and **§11.4.237** (mandatory exhaustive context-and-spirit-aware translation review — accuracy necessary, never sufficient) into all six consumer carriers, closing a measured §11.4.157 lockstep gap (canon at .237, consumers at .235). Verified byte-identical across all six (md5 `df2cb867668b82e66e2d25235cd1dec5`). |
| `74f24528` | main | Cascades **§11.4.238** (automated QA must be the DISCOVERER: manual QA finds nothing new, and anything it finds is a coverage escape) into all six carriers **and** bumps the constitution pin to `177f2b0` in the same commit per CONST-049 step 7, after verifying the head published on all 8 constitution remotes. §11.4.238's forensic anchor is *this project*: every defect closed that day — the health endpoint, the dead completion port, the two unfetchable pins, both hardware-profiler defects — was found by an agent reading source or probing by hand, while HelixQA reported green throughout. |
| `f0dd7069` | main | Bumps four submodule pointers so a **fresh clone actually gets the fixes** — every fix lived in a submodule while the parent still pinned a commit predating it. Every old pointer verified an ancestor of its new head (FF-safe ×4, no force), every head verified published. Notable §11.4.6 datum: `git rev-list --count HEAD..@{u}` reported behind=0 *before* the fetch; after `fetch --all`, behind=3 — remote state is not knowable without fetching. |
| `204bc7b4` | main | Two third-party pins recorded on main that **no clone can fetch**. The dispatching premise was wrong and the correction is the finding: 7 of 9 suspected pins were fine. The real defect is that `signoz` and `skyvern` were explicitly listed as deliberately-not-recorded, then recorded anyway by an `Auto-commit`. Two independent oracles agree neither exists upstream (GitHub REST 422; live `upload-pack` "not our ref"). |
| `33ee0670` | main | **HXC-243** — of 129 HelixQA banks / 243 http steps, **16 banks / 72 steps declared no assertions at all** and the executor compares a field only when declared, so they scored PASS on any response. Baseline measured: 16/16 banks PASS against a dead target, exit 0. Now 238 steps carry an explicit expectation, 5 an explicit skip+reason, 0 assertion-free. Key finding: `expect_status` alone would **not** have fixed it — the wrong port answers 200 with a JSON-RPC error envelope. |
| `d8e46001` | main | A HelixQA agent **proved a PASS-bluff in our own test banks by construction** — it pointed a bank at the wrong service (a completion path already proven dead with HTTP 503) and wrote its thesis before running. Both cases reported PASS. |
| `9f8fdd27` | main | **HXC-227** — a live provider key published in a tracked design doc, committed 2026-06-19 and present on all four mirrors for **48 days**, under a heading reading "already configured". The discovering agent's handling is recorded as correct: it recognised that exporting the doc would copy the key into two *new* tracked files, removed the artifacts and quarantined the source rather than attempting a scrub. Only rotation withdraws it. |
| `09a086a6` | main | **HXC-187 / D-7** — renames the thin root module to `dev.helix.code/meta` so the root and inner modules stop claiming the same identity. |
| `1dab3e06` + `40c8d5ce` | main | Tune test parallelism from the **binding cgroup quota**, not `nproc`. Paired with `918f969c`, which explicitly corrects its own earlier root cause ("GOMAXPROCS does not explain the agent stalls"). |
| `732320c8` | main | Untracks **104 MB** of vendored `node_modules` from the agent repository. |
| `ee39ea30`, `9f68ed04`, `cf4eb22a`, `e9a1ecb6`, `69772e56` | main | §11.4.65 sibling-export backfill (101 root docs, 73 inner docs, 62 under `docs/**`, script companions, 4 missing `.pdf`). These account for the bulk of the range's 458k inserted lines. |

### Tracker / workable-items (revision 3 range)

| SHA | Repo | Summary |
|---|---|---|
| `ac9e9e4d`, `38d8f641` | main | Closed verified items in two passes; High severity 8 → 4, Critical 0. `38d8f641` also caught that the shipped `workable-items` binary was **ten days older than the invariant its own source defines**, so `validate` reported OK on records the source rejects. |
| `8214bd93`, `e84e5b4c` | main | **HXC-217** — 15 closure records described their proof in prose instead of pointing at it, which is mechanically indistinguishable from having no proof. Narratives preserved verbatim in a new `item_history.note` column; a resolvability validator added that refuses a closure whose stated location does not resolve. |
| `679e2e2e` + `825e1c3c` | main | **HXC-245** filed, then its own filed root cause **retracted as wrong, refuted by the code** — the correction is the artifact worth reading. |
| `79939789` | main | **HXC-246** — the agent test suite reports 74 failures, filed with classification *deliberately open*. Three signatures, none yet proven a product defect and none yet cleared: 22 test files hardcode a cache port that is closed on this host (the container publishes it elsewhere); a cluster failing in 0.00s while the service under test is provably healthy; wall-clock assertions overshooting ~26% under measured contention. The commit withdraws one of its own guesses explicitly. |
| `b48d235e` | main | **HXC-247** — two owned, individually-correct components both claim port 8100. The agent's port allocator assigns it 8100 while its shipped config still pins 7061, so the verifier takes 8100 first; **82 test files** address 8100 expecting the agent, receive the verifier's 404, and report the agent broken while it is healthy. Root cause of HXC-246's 0.00s cluster. Contains a self-correction retracting an earlier "8090" claim with measurements. |
| `d706f8cc` | main | **HXC-248** — integration-test cleanup decides whether to tear down the stack by checking only that *something* answers on port 8100. That something is the verifier, not the agent, so teardown is skipped and **the live platform survives by coincidence**; stop or move the verifier and the same check would tear down a running deployment. The log line it prints is also false. |
| `1f177495` | main | **HXC-244 filed** — the defect was found, fixed, deployed and verified, but no tracker entry was ever created; HXC-243 and HXC-245 existed and 244 was a genuine gap. Filed directly as `Fixed (→ Fixed.md)` against the runtime evidence above. |
| `7bd80ff6` | main | Swept all 48 non-terminal items against `git log --all -i --grep=<ID>`; closed **HXC-199** and **HXC-217** on re-verified evidence and deliberately left 46 open. Records a systemic finding: `verify-all-constitution-rules.sh` invokes 21 `scripts/gates/` entries and **zero** of the three `scripts/testing/guard_hxc*.sh` guards written this cycle — a guard nothing runs cannot fail (§11.4.135). |
| `5725ee06` | main | Normalises 16 severity values that held entire triage paragraphs where a closed-set token belongs. 10 were safe to reduce because the rationale already rendered from `body_md`; **6 were not** — the column was their only copy, so the rationale was first written into `body_md` and only then reduced. Prose 16 → 0, valid 230 → 246, with all 16 rationales proven to survive a full regeneration. |

## Per-repo commit inventory

### main (this repo) — 18 commits in `5ab97d8c..c29e1dcc` (revisions 1–2 range)

*(**Revision 3 correction.** Revision 2 changed this figure from 18 to 19 on the
grounds that `dd3c0c3b` had been omitted. The omission was real; the count change
was not. `git rev-list --count 5ab97d8c..c29e1dcc` returns **18**, and
`git merge-base --is-ancestor c29e1dcc dd3c0c3b` succeeds — `dd3c0c3b` is dated
2026-07-28 18:13 and lies **after** `c29e1dcc`, i.e. in the revision 3 range, not
this one. Revision 2 correctly added the missing commit to the Fixed table but
attributed it to the wrong range and inflated this count to match. Restored to
the measured 18; `dd3c0c3b` is carried in the revision 3 range above. The two
figures now reconcile: 18 + 231 = 249 = `git rev-list --count 5ab97d8c..HEAD`.)*

```
dd3c0c3b fix(persistence): close unprioritized-select race in autoSaveLoop (3rd instance)
c29e1dcc docs(qa): LIVE proof for the Anthropic wire and the i18n cwd fix
a5462e35 docs(qa): LIVE proof that stream:true reports the served model
e952f4d1 docs(qa): §11.4.83 evidence for the three non-feature commits — G7 reaches 0
1c9fdcf2 docs(qa): §11.4.83 evidence for the Cerebras, Cohere and MCP-concurrency commits
d8c44618 docs(qa): §11.4.83 evidence for e330149f model-identity fix
925aa859 docs(governance): backfill the §11.4.182 TRUNK RULE + bump two submodule pins
e330149f fix(llm/server): report the model that ACTUALLY served the request
578de68d docs(qa): land §11.4.83 end-user QA evidence for 7 feature commits
0a4eb8d0 docs(resume): correct live-state anchors after the end-of-session push
cf26a173 chore(submodules): bump gitlinks after end-of-session sync
56f7edf7 Auto-commit
66d6fb29 fix(e2e): stop go test compiling generated challenge output (3 packages unblocked)
98db7d79 feat(llmsverifier): seed real providers via consumer-owned config; first real scores published
d3e2c6e7 feat(systemd): add LLMsVerifier as the seventh unit, wired to the gateway
c4115c73 feat(systemd): install and boot the whole platform — HelixCode + HelixAgent + HelixLLM + infra
1296d328 Auto-commit
565506f6 chore: bump HelixLLM to helixllm-0.2.0 (CPU inference)
f14a3f89 chore: bump HelixAgent (Hyper provider)
```

### submodules/helix_agent — 9 commits since the previously-pinned `55ad06c6`

**Pinned into main's current gitlink (`3268fd73`) — 7 commits:**

```
3268fd73 fix(testutil/tests): verify the server is HelixAgent, not just that a port answers
0165ab1d fix(precondition): establish container ownership from the runtime, not port math
c20215bd test(helix_agent): chaos/compliance/automation hardening + live-endpoint guards
4b5f2867 style: gofmt alignment and comment spacing across helix_agent
36597718 test(search): reconcile UnknownType assertion to the reordered type check; cover the reachability short-circuit
0dfa69e1 fix(boot): resolve dependency endpoints from env; don't block startup on an already-reachable vector store
410afda5 feat: Hyper provider discovery + registry + verifier types
```

**Committed in `helix_agent`'s own history but NOT yet reflected in main's gitlink — 2 commits (see Known gaps):**

```
69a5ab8d fix(ollama): close goroutine-leak in CompleteStream on ctx cancellation
a345c551 fix(tests/unit): wire the model source; stop reading recorder-default 200 as success
```

### submodules/tool_schema — 1 commit since the previously-pinned `cb7bf675`, fully pinned

```
8cec90b fix(i18n): embed translation bundles so lookups work from any directory
```

## Known gaps / not yet verified

This section exists so this document does not read as more finished than the
underlying work is. None of these were papered over.

1. **§11.4.185 manual QA-team confirmation has not been given.** No record of a
   manual QA sign-off for this scope exists anywhere in this repository. Per
   this project's own governance, automated green is necessary but not
   sufficient — a manual QA pass is required before this scope can be called
   fully complete, and it has not happened.

2. **Desktop GUI packages do not build on this host.** Directly attempted
   `go build ./applications/...` in `helix_code/` on this host and it fails
   with two real, host-environment errors, not code defects:
   - `github.com/go-gl/gl/v2.1/gl`: `pkg-config` cannot find `gl.pc`.
   - `github.com/go-gl/glfw/v3.3/glfw`: `fatal error: X11/Xlib.h: No such file
     or directory`.
   This is not a new discovery — a prior commit in this cycle (`d6c05f76`,
   already merged before this release's commit range) documented this same
   gap and verified the README's claimed error text is byte-identical to the
   real one on this host. Re-confirmed here, not re-litigated. Any package
   depending on the Fyne desktop GUI stack is affected.

3. **The zero-skip full integration test stack was not booted for the last
   sweep.** Checked directly (`podman ps`): the containers currently running
   on this host (`helixcode-infra-postgres`, `-redis`, `-weaviate`, `-qdrant`,
   `-chromadb`, `-cognee`, `-ollama`, etc., all systemd-managed via
   `compose.helixcode-infra.yml`) are the **production-like platform infra**,
   not the `docker-compose.full-test.yml` zero-skip test stack (which would
   run containers named `helixcode-postgres-full`, `helixcode-redis-full`,
   etc. — none of those are present). Any integration-test result gathered
   without first running `make test-infra-up` for that specific stack is
   environmental, not a confirmed pass or fail, until it is booted and the
   sweep re-run against it.

4. **The full `go test` suite has not concluded cleanly on a quiet host for
   this cycle.** `docs/CONTINUATION.md`'s own rev-18 audit (not independently
   re-run as part of this documentation-only task, to avoid adding more load
   to a host other agents are actively building/testing on) records one
   completed `helix_agent` full-suite run on a contended host reporting 18
   `FAIL` packages, and attributes this to ambient host contention
   (constitution §11.4.119) consistent with this project's own documented
   pattern — but that specific 18-package list has not itself been re-run on
   a quiet host and confirmed clean. Treat as **unconfirmed**, not cleared.

5. **`helix_agent`'s goroutine-leak fix (`69a5ab8d`) and test fix (`a345c551`)
   are real, committed work but are NOT yet part of this release's pinned
   artifact.** Main's current gitlink for `submodules/helix_agent` points at
   `3268fd73`; the submodule's own `HEAD` is two commits further, at
   `69a5ab8d`. Until the gitlink is bumped and re-verified, this release does
   not actually ship the Ollama goroutine-leak fix even though the fix exists
   and is committed.

6. **`helix_agent`'s working tree is dirty** (modified tracked files and
   untracked new test files observed at read time) and **2 commits ahead of
   its own configured remotes** — i.e. not yet pushed anywhere outside this
   host. This draft does not push anything.

7. **main is 8 commits ahead of every configured remote** (`0` behind), i.e.
   also not yet pushed. This document commits only itself; per this task's
   constraints, nothing is pushed and no tag is created.

8. **Two commits carry the literal message `Auto-commit`** (`56f7edf7`,
   `1296d328`) with no descriptive body. Real, substantive content was
   recovered from their diffstats (see the Infrastructure / Added tables
   above) rather than invented, but the commit messages themselves do not
   describe intent, verification, or evidence — a documentation-clarity gap
   at the commit-message layer, flagged rather than silently backfilled.

9. **No claim in this document was accepted from a stale project artifact
   without independent re-derivation of the specific git facts it depends on**
   (HEAD SHAs, gitlink pins, tag list, container names) — those were each
   re-checked directly against the live repositories at draft time. The one
   exception is item 4 above, which is explicitly attributed to
   `docs/CONTINUATION.md` rather than re-run, and is labeled unconfirmed for
   exactly that reason.

10. **`helix_agent` has moved further since this draft's stated pin
    (`69a5ab8d`), adding a third sibling to the same defect class this
    document already tracks.** Commit `e2f42b12`
    (`fix(llm,database): prioritize cancellation in racy select statements`)
    fixes exactly the two "sibling" locations `dd3c0c3b`'s commit message
    names as instances of the same unprioritized-`select` pattern —
    `internal/llm/lazy_provider.go`'s `createProviderWithContext` and
    `internal/database/debate_log_repository.go`'s `StartCleanupWorker` — with
    the same RED/GREEN polarity-evidence discipline (§11.4.115). It is real,
    committed work, but as of this revision it is **not yet part of this
    release's pinned artifact**: main's gitlink for `submodules/helix_agent`
    still points at `3268fd73`, four commits behind `helix_agent`'s own
    current `HEAD` (`3268fd73` → `a345c551` → `69a5ab8d` → `e2f42b12`). This
    compounds Known gap 5 rather than replacing it — do not treat `69a5ab8d`
    as the current unpinned tip; re-derive `submodules/helix_agent`'s actual
    `HEAD` before citing it, per this document's own opening warning.

### Known gaps added or re-measured in revision 3

Gaps 1–10 above were written against the revision 1–2 range. Re-checked on
2026-08-09; the following supersede or extend them.

11. **Three standing guards written this cycle are registered in no runner.**
    Measured: `scripts/verify-all-constitution-rules.sh` invokes **21**
    `scripts/gates/` entries and **0** `scripts/testing/guard_hxc*.sh`. There is
    no auto-discovery glob. So `guard_hxc229_gateway_release_mode.sh`,
    `guard_hxc233_completion_path_live.sh` and
    `guard_hxc244_health_components_registered.sh` all exist, are individually
    falsifiable, and **never execute**. §11.4.135 asks for a standing suite;
    these are standing files. This is the direct reason HXC-229 was left open
    rather than closed in `7bd80ff6`.

12. **`db-to-md` silently drops two items.** `workable-items validate` reports
    HXC-247 and HXC-248 have no `doc_segments` row, and a regeneration to a
    scratch path confirms it empirically: both appear **0** times in the
    regenerated `Issues.md`, while every other item appears. 2 of 442 items.
    Pre-existing — reproduced identically against a pre-change backup — and
    deliberately **not** silently repaired, because synthesising the missing
    segments is adjacent to HXC-245's failure mode and deserves its own item.

13. **13 open items carry no severity, so §11.4.132 risk-ordering cannot rank
    them.** 196 rows have an empty severity, but 183 of those are already
    closed, where severity ranks nothing. The actionable set is the 13 open
    ones: HXC-227, 229, 231, 232, 233, 234, 235, 236, 239, 240, 241, 242, 243.
    No rule exists in this repository to derive a severity — the schema itself
    says *"informational only; not closed-set"*, with no CHECK and no default —
    so these were left alone rather than invented (§11.4.6).

14. **`submodules/helix_agent`'s recorded pin and its working checkout
    disagree.** `git submodule status` shows a `+` prefix: the pin is
    `66a3c1c6`, the checkout is `d6856cc9`. Three other agents were actively
    editing that submodule while revision 3 was written, so it was not entered
    and the delta is **not characterised here**. HXC-234 could not be resolved
    past ambiguous for the same reason — its fix lives inside that submodule.

15. **The release position is 3 ahead / 0 behind on all four remotes**, measured
    *after* a real `git fetch --all --prune`. Supersedes gap 7's "8 ahead". The
    three local commits are revision 3's own tracker work (`1f177495`,
    `7bd80ff6`, `5725ee06`). Nothing was pushed and no tag was cut.

16. **Two further `Auto-commit` commits** landed in the revision 3 range
    (`c209bff2`, `29696107`), extending gap 8. Additionally the pinned
    `helix_llm` head `3d789cb` itself carries the literal message `Auto-commit`
    — it was verified by content (it contains `8260cf8`), not by its subject.

17. **§11.4.185 manual QA-team confirmation is still not given** — unchanged
    from gap 1, and now load-bearing for §11.4.238, which this range cascaded:
    manual QA is a confirmation step that must find nothing new, and it has not
    been run at all for this scope.

## Independent review

Per constitution §11.4.142 (every change gets an independent review, no
exception) and §11.4.194 (exhaustive all-scenario review), this document
received a full independent review after its original commit (`8bdac5b0`)
landed. The authoring agent had been killed by an API session limit
immediately after committing and never reported, so this changelog had zero
review until now.

**Method:** every cited short SHA in the Fixed / Added / Changed /
Infrastructure tables and the per-repo commit inventories was checked to
exist and to actually do what its entry claims, via `git show --stat` and
full `git show` against this repository, `submodules/helix_agent`, and
`submodules/tool_schema` — not accepted on the strength of the subject line
alone. Quantitative claims (assertion counts, PASS/FAIL tallies, live scores,
frame counts) were spot-checked against the cited commits' own bodies.
`git log --oneline 8bdac5b0..HEAD` was run in this repo (empty — main has not
moved) and, using the changelog's own stated submodule pins as the base,
in `submodules/helix_agent` (one further commit, `e2f42b12` — see Known gap
10) and `submodules/tool_schema` (none). The `.env` `HELIX_RELEASE_PREFIX`
resolution, the §11.4.44 revision header, and the "3 desktop packages fail to
build" claim (`applications/desktop`, `applications/aurora_os`,
`applications/harmony_os` — confirmed all three import the Fyne stack, so
all three are genuinely covered by `go build ./applications/...` and by the
stated X11/OpenGL failure) were all independently re-verified rather than
taken on trust.

**Finding (blocking, now fixed in revision 2):** commit `dd3c0c3b` — this
changelog's own direct parent commit, committed roughly 90 seconds before
`8bdac5b0` — was completely absent from every section of the original draft
despite being real, substantive, in-scope work covered by the document's own
stated data-collection window. It has been added to the Fixed table, the
per-repo commit inventory, and cross-referenced against its `helix_agent`
sibling fix (Known gap 10).

**Verdict:** no invented history was found anywhere in this document — every
other cited SHA, quantitative claim, and Known-gaps disclosure checked out
exactly as stated. With the one finding above corrected, this document is
independently confirmed accurate as of that revision (2026-07-28T15:21:59Z).

### Revision 3 (2026-08-09)

**Why it was needed:** the document was pinned to a HEAD 11 days and 231 commits
stale, and covered none of the work in that range.

**Method:** the range was re-derived from `git log` / `git show`, never from
memory or from any prior project artifact. Every short SHA newly cited above was
checked to exist and to carry the subject attributed to it — **51 main-repo SHAs
verified present via `git cat-file -e`, 0 missing**, plus `helix_llm` `3d789cb`
and `8260cf8` and `constitution` `177f2b0` in their own repositories. The two
`helix_agent` values (`66a3c1c6`, `d6856cc9`) are quoted from main-repo data
only — `git ls-tree HEAD` and `git submodule status` — precisely because that
submodule was not entered. Counts are pasted command output, not estimates: 231 commits in
`c29e1dcc..HEAD`, 2 of them the literal `Auto-commit`, 0 merges, 249 since the
tag. Remote positions were measured **after** an actual `git fetch --all --prune`
rather than from tracking refs, because this range contains a commit
(`f0dd7069`) whose own message documents a tracking ref reporting behind=0 before
a fetch and behind=3 after.

**Finding 1 (corrected here): revision 2 introduced a counting error while
fixing a real omission.** It raised "commits since the last tag" from 18 to 19 to
account for `dd3c0c3b`. `git rev-list --count 5ab97d8c..c29e1dcc` returns 18, and
`dd3c0c3b` is provably *after* `c29e1dcc` — it belongs to the revision 3 range.
The commit did need adding to the Fixed table; the count did not need changing.
Both are now correct and reconcile: 18 + 231 = 249.

**Finding 2: a claim in the revision 1–2 pin table could not have been true as
written and is superseded.** It described `submodules/helix_agent` as pinned at
`69a5ab8d`. The pin recorded in `HEAD`'s tree today is `66a3c1c6`, and the
working checkout is a third value, `d6856cc9`. Rather than reconcile a stale
figure, revision 3 states all three positions and marks the delta uncharacterised
(Known gap 14), because the submodule was deliberately not entered.

**Verified rather than assumed:** that the pinned `helix_llm` (`3d789cb`)
actually contains `8260cf8`, the HXC-244 health fix — `git merge-base
--is-ancestor` confirms it, so a fresh clone gets the fix. This mattered because
this same range contains two separate incidents (`f0dd7069`, `204bc7b4`) of
pins that did not carry the fixes attributed to them.

**Honest boundary (§11.4.6):** revision 3 has itself had **no** independent
review — it is authored by the same agent that produced the tracker commits it
describes. Its factual claims are individually re-derivable from the commands
cited, but the §11.4.142 independent-reviewer seam has not been crossed for this
revision. Treat the revision 3 sections as verified-by-construction, not
peer-reviewed.

## Sources verified

All SHAs, commit subjects, commit bodies, tag names/dates, gitlink pins,
container names, and the `go build` error text above were read directly from
this repository, `submodules/helix_agent`, and `submodules/tool_schema` via
`git log`, `git show`, `git tag`, `git ls-tree`, `podman ps`, and a direct
`go build` invocation on 2026-07-28. No claim in the Fixed / Added / Changed
tables above is unsourced from a real commit; anything that could not be
traced to a commit was left out rather than inferred.

**Revision 3 (2026-08-09).** The `c29e1dcc..5725ee06` range was derived with
`git rev-list --count`, `git log --oneline/--format`, `git show --stat`,
`git log -1 --format=%B` on every commit cited, `git merge-base --is-ancestor`
for the ancestry claims, `git ls-tree HEAD <path>` and `git submodule status`
for the pins, `git fetch --all --prune` followed by
`git rev-list --count refs/remotes/<r>/main..HEAD` for the remote positions,
`git ls-files` to confirm cited evidence directories are actually **tracked**
rather than merely present on disk, and `sqlite3 -readonly` plus
`workable-items validate` / `sync db-to-md` for every tracker figure. The
constitution anchors were read from `constitution/Constitution.md` at the pinned
commit and confirmed to be genuine `### ` block-openers, not incidental
mentions. File sizes were taken with `wc -c`, not `stat`, which misreports on
this host. Nothing in the revision 3 sections is carried over from a prior
document without re-derivation.
