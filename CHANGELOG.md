# Changelog

All notable changes to HelixCode are recorded here. Format loosely follows
[Keep a Changelog](https://keepachangelog.com/). Release tags are
**project-prefixed** per §11.4.151, resolved from `.env` `HELIX_RELEASE_PREFIX`
(currently `helix-code`): `helix-code-X.Y.Z-dev-N.N.N`. Two legacy unprefixed
`helixcode-vX.Y.Z` tags predate that scheme; this document's *recommendation*
is to retain them as historical artifacts rather than an active naming
convention, but that disposition is not settled and awaits an explicit
operator decision — see `docs/changelogs/helix-code-1.2.0-dev-0.0.1.md`
§"§11.4.151 release-tag prefix compliance" for the full finding and the
recommendation's stated conditions.

## [helix-code-1.2.0-dev-0.0.1] — DRAFT, not yet tagged (documentation prepared 2026-08-10/11)

**No git tag exists for this entry yet.** Full per-commit citations, SHA-level
detail, tracker-item status cross-checked against `docs/workable_items.db`,
and an honest "Known gaps" ledger live in
`docs/changelogs/helix-code-1.2.0-dev-0.0.1.md` — this entry is a summary of
that document, not a replacement for it. The full document covers **256
commits** in the main repo since the last tag (`helix-code-1.1.0-dev-0.0.3`)
plus the entirety of `submodules/helix_agent`'s unpushed history (24 commits);
this summary highlights only the user-visible headline items.

### Added

- **The whole platform installs and boots as systemd units** via `./setup.sh`
  — HelixCode server, HelixAgent, HelixLLM gateway + coder, infra
  (Postgres/Redis/Weaviate/Qdrant/ChromaDB/Cognee/Ollama), and LLMsVerifier —
  with real HTTP health checks and a verified stop/start cycle.
- **LLMsVerifier is wired to real provider data**, publishing genuine
  verification scores instead of an empty `{}`; the gateway now routes on
  measured scores instead of always falling back to its static table.
- **Hyper provider discovery, registry, and verifier types** land in
  `submodules/helix_agent`.

### Fixed

- **Model identity is honest end-to-end.** A client requesting an alias
  (e.g. `"model":"default"`) now gets back the model that actually served the
  request, on both the OpenAI- and Anthropic-compatible wires, streaming and
  non-streaming — not the alias echoed back.
- **The gateway's primary completion path, which was returning HTTP 500 on
  every request with no cloud key configured, is restored** — the local
  provider was pointed at a port matching no running service. Live-reverified
  during this documentation pass: a real completion answered through the
  public endpoint. **The corresponding tracker item (HXC-229 for the related
  release-mode fix, HXC-233 for this one) remains `Queued` in
  `docs/workable_items.db`** — see the per-release changelog's dedicated
  reconciliation section for exactly why and what would close it.
- **The health endpoint no longer reports "healthy" without checking
  anything** — it now names four real dependencies with measured round-trip
  durations, and the fix required a redeploy of a stale-by-15-days binary,
  not just a source change (§11.4.108).
- **A large, misleading class of HelixAgent integration-test failures is
  fixed at the root.** Roughly a hundred failing assertions across several
  packages traced to test guards checking mere port-reachability instead of
  service *identity* — a different local service (LLMsVerifier) or a
  container (Weaviate) squatting on a well-known port was being blamed for
  HelixAgent regressions it had nothing to do with. Every affected guard now
  verifies identity, not reachability.
- **HelixAgent's gRPC server can start alongside HelixAgent's own
  infrastructure** — it previously hardcoded the same port
  (`:50051`) this project's own Weaviate container publishes and could not
  bind. Re-derived the server, its startup banner, its docs, its Kubernetes
  manifests, and a protocol-compliance challenge script from one port-registry
  source. **Landed in source; not yet verified against a rebuilt, redeployed
  artifact — see the per-release changelog's Known gaps.**
- **Generated CLI-agent config files pointed 46 of 48 downstream assistants
  at the wrong port.** Fixed via the same single-resolver pattern.
  **Landed in source for every currently-wired call site; one additional,
  currently-unwired call site inside `submodules/helix_agent` was found
  during this documentation pass and is flagged, not fixed, in the per-release
  changelog — treat this fix as source-complete, not verification-complete.**
- **265 data races eliminated** in the desktop application, root-caused to a
  code comment falsely claiming a widget setter was goroutine-safe.
- **A guaranteed streaming-with-tools deadlock is closed**, along with a
  family of provider goroutine/HTTP-body leaks on client disconnect.
- **265 data races aside, three separate "unprioritized `select`" races** are
  closed across persistence, LLM provider, and database cleanup code.
- **`go.mod`/`go.sum` tidied** — 30 stale dependency hashes and 2
  structurally incomplete entries, undetected by `go build`/`go vet` but a
  real `go mod tidy -diff` failure.
- **The project's own release-validation scripts could report "all tests
  passed" without running a single test** — fixed to discover the real
  server address by identity and treat zero-tests-executed as a hard failure.
- **Two credential/security exposures were found, traced, and handled** (one
  published on all four mirrors for 48 days); rotation remains outstanding
  and is tracked, not silently closed.

### Changed

- **The project's own workable-items tracker was re-derived from its
  database of record** after `docs/Issues.md` drifted 18 open items and
  several days stale — surfacing and fixing a hidden governance-gate
  violation in the process.
- **Three standing regression guards, written earlier and never wired into
  any suite, are now part of the release-gate sweep** (32 gates, up from 29),
  after three independent review rounds found and closed further
  false-positive/false-negative failure modes in the guards themselves.
- Root module renamed `dev.helix.code` → `dev.helix.code/meta` (HXC-187/D-7)
  so the meta-repo and inner Go module stop sharing an identity.

### Known gaps (NOT closed by this documentation pass)

- **HXC-229 (gateway release mode) and HXC-235 (semantic-embeddings signal)
  each have a specific, tracked, previously-recorded blocking reason that has
  since been addressed by later work, without a follow-up review or tracker
  closure; HXC-233 (this completion-path fix) has genuine tracked evidence
  and a passing live-reverified guard but was never routed through a closure
  review at all.** None of the three is "Fixed" in `docs/workable_items.db`
  and this document does not change that — see the per-release changelog's
  "HXC-229/233/235 tracker-vs-runtime-state reconciliation" section for the
  full evidence trail (guard scripts, `docs/qa/` paths, and the exact
  independent-review commit each item's prior `Queued` status traces back
  to).
- **§11.4.185 manual QA-team confirmation has not been given** for any of the
  above — automated-green is necessary but not sufficient per this project's
  own governance.
- **The gRPC and CLI-agent-config port fan-outs are landed in source, not
  confirmed against a rebuilt/redeployed artifact** (see "Fixed" above and
  the per-release changelog's Known gaps 19–20).
- **A 32-commit stretch of `submodules/helix_agent` history
  (`a345c551..66a3c1c6`), already published on all four of its remotes, was
  found never itemized in any prior changelog revision.** Out of scope for
  this pass (it predates the unpushed work this task documents); flagged as a
  follow-up documentation task.
- **This version has not been tagged, pushed, or built for this entry.**
  Documentation and version metadata only — see the per-release changelog for
  the full, itemized "Known gaps" ledger.
- Mobile clients and GUI desktop builds continue to require simulator/device/
  display access not available in this documentation pass (carried over,
  unchanged, from the `helixcode-v1.0.0` entry below).

## [helix-code-1.1.0-dev-0.0.3] — 2026-07-24

### Added

- **HelixLLM mode switch.** The shared HelixLLM engine now accepts a
  runtime mode parameter (`coder` / `claude`) so the same provider
  backend serves both conversational and code-agent workloads with
  mode-appropriate system prompts and token budgets without
  reconfiguration.

### Changed

- **HelixAgent submodule pointer bumped.** Updated to the latest
  revision bringing Zen provider as the default and a generic
  provider fallback path.

### Fixed

- **`.gitignore` covers build derivatives.** Added `scratchpad/` and
  `.git-backups/` patterns to prevent 86 GB+ of build artifacts and
  backup copies from polluting `git status` and accidentally being
  staged.

## [helixcode-v1.0.0] — 2026-06-15

First release tag.

### Release-gate evidence (all green)

- `go build ./...` → exit 0 (whole inner module `dev.helix.code`).
- Unit suite `go test -short ./...` → **188 packages, 0 failures**.
- Anti-bluff smoke (CONST-035) → **0 real production bluffs**.
- Integration e2e against a **live LLM provider** (real round-trips, no mocks):
  `POST /api/v1/llm/generate`, `POST /api/v1/llm/stream` (token-by-token SSE +
  `[DONE]`), `browser → server → provider → browser` (chromedp, real DOM), and
  `POST /api/v1/specify` (real 2-agent speckit debate) — all PASS.
- Durable cross-session memory (sqlite `DiskStore`): persist → restart → recall proven.

Runtime evidence: `docs/qa/web-llm-e2e-20260615/`.

### Added

- Honest TUI context-window USED-% indicator — real per-session token accounting,
  omits when the model window is unknown (CONST-035) (HXC-077).
- Overlapping-skill precedence guard — deterministic lexicographic resolution
  coverage (HXC-078).
- Internal-package i18n wiring on **all** entry paths (`cmd/server`, TUI, desktop,
  aurora_os, harmony_os) so user-facing strings resolve for real users while the
  loud raw-key default is preserved (HXC-099).
- Runtime e2e suites for the web LLM endpoints — `/generate`, `/stream`, browser,
  `/specify` (HXC-103, HXC-105).
- helix_agent durable-fallback path-resolver test coverage — persist→restart→recall
  through the production-chosen disk store (HXC-106).

### Fixed

- `streamLLM` production hang: `chunkChan` was never closed, so `[DONE]` was never
  emitted and **every** `/api/v1/llm/stream` request hung until the 120s deadline
  (HXC-104).
- `security` TLS test: removed a live external-network dependency and a nil-deref
  panic that crashed the whole `security` test binary (HXC-101).
- Out-of-box config: a `config.json` omitting `version`/`server.port` no longer
  fails validation — the JSON load path now merges viper defaults (HXC-098).
- harmony_os REPL `Goodbye!`/`Error` strings routed through i18n (HXC-102).
- `/specify` + `/debate` min-agents wiring and model-tag parsing (earlier in cycle).

### Docs / hygiene

- `docs/CONTINUATION.md` de-bloated (line-1 header 54,856 → 2,931 chars) and
  resynced; CONST-064 metadata table + ToC restored (HXC-100).
- SQLite-backed workable-items tracker kept in sync; every closure carries captured
  RED→GREEN evidence.

### Known gaps (NOT headlessly validated in this release gate)

- Mobile clients (iOS / Android / Aurora OS / Harmony OS) and GUI desktop feature
  recordings require simulator / device / display access.
