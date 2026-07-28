# HelixCode Release Changelog — helix-code-1.2.0-dev-0.0.1

**Revision:** 1
**Last modified:** 2026-07-28T13:11:20Z
**Maintainer:** HelixCode release engineering (documentation draft, AI-agent authored)

---

> **DRAFT — pinned to a stated HEAD, this is not a cut tag.** No git tag has been
> created by this document. Everything below was derived by reading real git
> history (`git log`, `git show`) on 2026-07-28 at approximately 13:11 UTC.
> Multiple background work streams were actively landing commits in this repo
> and its submodules while this draft was written (constitution §11.4.103 /
> §11.4.192 continuous multi-track work), so HEAD will have moved by the time
> this is read. **Re-derive the commit ranges below before treating any of this
> as final** — do not copy the SHAs or counts forward without checking.
>
> | Repo | Stated HEAD (this draft) | Relative to its configured remotes |
> |---|---|---|
> | main (this repo) | `c29e1dcc9fad029f64a8836148d8d5ddfc32f0fb` | 8 commits ahead, 0 behind — **not pushed** |
> | `submodules/helix_agent` | `69a5ab8dc2469f73ddf3afa0bced01103df8b656` | 2 commits ahead of its remotes; working tree dirty (14 modified, 4 untracked) |
> | `submodules/tool_schema` | `8cec90baadad281eb26a5671b792921f03eb7386` | 1 commit ahead of its remotes; working tree clean |
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

## Per-repo commit inventory

### main (this repo) — 18 commits since `helix-code-1.1.0-dev-0.0.3` (`5ab97d8c`)

```
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

## Sources verified

All SHAs, commit subjects, commit bodies, tag names/dates, gitlink pins,
container names, and the `go build` error text above were read directly from
this repository, `submodules/helix_agent`, and `submodules/tool_schema` via
`git log`, `git show`, `git tag`, `git ls-tree`, `podman ps`, and a direct
`go build` invocation on 2026-07-28. No claim in the Fixed / Added / Changed
tables above is unsourced from a real commit; anything that could not be
traced to a commit was left out rather than inferred.
