# Changelog

All notable changes to HelixCode are recorded here. Format loosely follows
[Keep a Changelog](https://keepachangelog.com/). Release tags are
**helixcode-prefixed semver**: `helixcode-vX.Y.Z`.

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
