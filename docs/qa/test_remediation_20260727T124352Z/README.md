# QA Evidence — Test-Suite Remediation Run

| Field | Value |
|---|---|
| Revision | 1 |
| Created | 2026-07-27 |
| Last modified | 2026-07-27 |
| Status | active |
| Run ID | `test_remediation_20260727T124352Z` |
| Base HEAD | `66d6fb29` (meta-repo, branch `main`) |
| Scope | helix_code inner module · submodules/helix_agent · submodules/helix_qa |
| Evidence class | §11.4.83 end-user QA transcript; §11.4.5 / §11.4.69 captured runtime evidence |

## Table of contents

- [Purpose](#purpose)
- [Method](#method)
- [Closures with captured evidence](#closures-with-captured-evidence)
- [Production defects found](#production-defects-found)
- [Systemic pattern: host-state-dependent tests](#systemic-pattern-host-state-dependent-tests)
- [Paired mutations](#paired-mutations)
- [Open findings not closed in this run](#open-findings-not-closed-in-this-run)
- [Honest coverage boundary](#honest-coverage-boundary)

## Purpose

Remediate the failing-package set blocking a `helix-code-1.2.0-dev-0.0.1` tag, under
the §11.4 anti-bluff covenant: every closure carries pasted runtime output from a real
run, and every gate touched carries a paired §1.1 mutation proving it still bites.

## Method

The inherited failure list (`qa-results/full_retest/*_20260727T115828Z.log`) was
**re-derived rather than trusted** — it predated HEAD `66d6fb29`. Re-deriving mattered:
two entries were already green, one package (`internal/verifier`) had newly regressed,
and `internal/search` had recovered. The set churns; it is not a countdown.

Work ran as one main stream plus parallel subagent streams on disjoint file scopes
(§11.4.58 / §11.4.103), each stream forbidden from weakening assertions, adding
`t.Skip()`, or stubbing behaviour to force green.

## Closures with captured evidence

> **Render note (§11.4.168):** the closures below are a list, not a table.
> A table form of this same content was rendered twice — at four columns and at
> three — and the exported PDF silently dropped the rightmost *Captured proof*
> column both times, while the exporter reported `rendered=2 failed=0`. See
> [Open findings](#open-findings-not-closed-in-this-run) item 8.

**`applications/desktop` + `aurora_os` + `harmony_os`** (3 pkgs)
: *Root cause* — 5 test files referenced GUI-only symbols (`parseHexColor`, `CustomTheme`, `NewThemeManager`, `NewDesktopApp`, `NewAuroraApp`, `NewHarmonyApp`) declared in `!nogui`-tagged sources, but carried no build constraint themselves, so they were compiled under `-tags=nogui` where those symbols do not exist.
: *Proof* — `go test -tags=nogui ./applications/...` exits 0, all 10 pkgs `ok`.

**`cmd/security_scan`**
: *Root cause* — the unhealthy-path test asserted "unhealthy implies exit 1" while assuming nothing listens on :9000. SonarQube **is** running, so the health check succeeded and the assertion inverted.
: *Proof* — passes at `-count=5`; paired mutation FAILs correctly.

**`tests/compliance`**
: *Root cause* — `getRoot()` returned the *submodule's* root; the 20 extracted modules live at the *consuming project* root (CONST-051(C)) in snake_case (CONST-052), not CamelCase.
: *Proof* — `Found 20/20 extracted modules`, previously `0/20`.

**`internal/tools/gittools`**
: *Root cause* — `git init` inherits the host's `init.defaultBranch` (unset yields `master`) while assertions hardcode `main`.
: *Proof* — `ok 0.075s`, stable at `-count=3`.

**`internal/adapters/containers`**
: *Root cause* — compose build contexts off by one directory since the grouped `submodules/helix_agent` layout move.
: *Proof* — `ok 0.700s`, including `-race`.

**`internal/mcp/servers`**
: *Root cause* — same `init.defaultBranch` class, independent fixture.
: *Proof* — `ok 0.241s`, including `-race`.

**`internal/handlers`**
: *Root cause* — `/v1/debates` is deliberately JWT-protected; tests sent no `Authorization` header after commit `186f3c9c` repointed them at the live server.
: *Proof* — `ok 2.315s`, 9 consecutive green runs.

**`internal/clis/agents/master`**
: *Root cause* — `ListAvailable()` returned `nil` rather than an empty slice.
: *Proof* — `ok 0.092s`.

**`internal/ensemble/multi_instance`**
: *Root cause* — `CreateSession` gates on a real `exec.LookPath`; neither `aider` nor `claude` is installed here.
: *Proof* — `ok 0.009s`.

**`helix_qa pkg/vision`**
: *Root cause* — Ollama reachable but zero models pulled; `GetAvailableModels` correctly returned an empty list while the test asserted non-empty, a property of the host rather than the code.
: *Proof* — `ok 0.529s` plus 16 sibling packages.

**`internal/testing/acp` + `embeddings` + `vision` + `integration`** (4 pkgs)
: *Root cause* — port-registry collision; tests probed `:8100`, now owned by LLMsVerifier, which 404s HelixAgent routes.
: *Proof* — all four `ok`, 3 consecutive `-count=1` runs identical.

## Production defects found

Two defects would have shipped broken — neither was a test artifact:

1. **`internal/clis/agents/master`** — `ListAvailable()` passed the registry's
   `var infos []AgentInfo` straight through. With no CLI agents installed it returned
   `nil`, serialising as JSON `null` instead of `[]`. Fixed by nil→empty normalisation.

2. **`docker/mcp/docker-compose.mcp-servers.yml`** — build contexts resolved to
   `<meta-repo>/submodules/mcp_servers`, which does not exist; the submodule is at
   `<meta-repo>/mcp_servers`. Commit `e19da5fd` computed "3 levels up", correct when
   helix_agent sat at `<meta-repo>/helix_agent`, stale after the grouped-layout move.
   **All 7 core MCP services would fail `compose build`.**

## Systemic pattern: host-state-dependent tests

Six of sixteen failures were one class: **the test's verdict depended on ambient host
state rather than on the code under test** (§11.4.50).

| Package | Host property silently required |
|---|---|
| `cmd/security_scan` | SonarQube **not** running on :9000 |
| `internal/tools/gittools` | `init.defaultBranch == main` |
| `internal/mcp/servers` | `init.defaultBranch == main` |
| `internal/ensemble/multi_instance` | `aider` + `claude` on `PATH` |
| `helix_qa pkg/vision` | ≥1 Ollama model pulled |
| `internal/testing/*` | HelixAgent owning port 8100 |

Each passed on the machine that authored it and failed here. **Bringing the platform
live is what exposed them** — the suite had been green partly because the
infrastructure was down.

A related bluff was found in the shared harness: `testutil.checkHTTP` treats **any
status < 500 as "available"**, so its availability guard green-lit a 404-answering
neighbouring service instead of skipping. The four testing packages now verify service
**identity**, not mere reachability. The `checkHTTP` weakness itself remains open
(see below).

One test **could never have passed**: `ListProviders` filtered on `p.Status ==
"available"`, but `/v1/providers` has never emitted a `status` field (git pickaxe:
zero hits) — the filter discarded every provider unconditionally.

## Paired mutations

Every gate touched was mutation-verified (§1.1) — mutate → assert FAIL → restore →
assert PASS → residue-scan clean.

| Gate | Mutation | Result |
|---|---|---|
| `cmd/security_scan` unhealthy path | `os.Exit(1)` → `os.Exit(0)` | FAIL, showing the real round trip: `dial tcp 127.0.0.1:39695: connect: connection refused` |
| `tests/compliance` count | `HELIX_PROJECT_ROOT` → empty dir | `Found 0/20` → FAIL |
| `adapters/containers` build context | restored the broken 3-up path | FAIL: `not a directory on disk` |
| `ensemble/multi_instance` | disabled binary injection | FAIL: `agent type aider is not available` |
| `pkg/vision` | dropped last model during mapping | FAIL on exact ordered list |
| `internal/testing/*` | pointed at impostor `:8100` | still PASS — identity check rejects and falls through |
| `master.ListAvailable` | reverted nil→empty normalisation | FAIL: `Expected value not to be nil` |

## Open findings not closed in this run

Stated explicitly rather than left implied (§11.4.6):

1. **Port-registry collision** — `internal/ports` + registry doc assign 8100 to
   HelixAgent; `configs/development.yaml` still says 7061; LLMsVerifier now occupies
   8100 (`d3e2c6e7`). Needs an owner decision; live-service blast radius.
2. **`testutil.checkHTTP` `< 500 == available`** — affects every package using the
   shared harness, not only the four repaired here.
3. **Tracked files rewritten by tests** (CONST-053) —
   `reports/latency/p99-baseline-2026-03-16.txt` and
   `releases/.version-data/helixagent.last-hash` are regenerated on every run. A
   "baseline" that moves each run can never detect a regression.
4. **`/v1/auth/register` broken on the live server** —
   `column "username" does not exist (SQLSTATE 42703)`; a real schema defect.
5. **No working LLM providers on this host** — every debate fails
   `insufficient agents (have 0, need 3)`, `circuit breaker is open`. HTTP contracts
   pass; end-to-end debate execution is genuinely non-functional here.
6. **42 constitutional anchors (§11.4.193–§11.4.234) never cascaded**; the cascade
   verifier's ceiling is §11.4.166, so it cannot fail on them.
7. **§11.4.164 post-update hook never wired** — no `last_update.log`; this is the
   mechanism whose absence allowed the drift above.
8. **PDF export silently drops the rightmost column of WIDE tables**
   (§11.4.168). Discovered while validating this very document. `pandoc` →
   `weasyprint` overflows A4 and clips, while
   `scripts/testing/sync_all_markdown_exports.sh` reports `rendered=2 failed=0`.
   Measured on this doc's own earlier revisions. The trigger is total rendered
   WIDTH, not column count: the closures table lost its rightmost column at BOTH
   four and three columns, while the three-column mutations table (shorter cells)
   survives. An initial column-count hypothesis was falsified by re-testing after
   the narrowing changed nothing.

   Verified as genuine loss, not an extraction artifact: the strings are present
   in the HTML, absent from both `pdftotext` and `pdftotext -layout`, and the PDF
   is not truncated (all 8 headings and the closing paragraph render). Affects
   every governed doc in the repo containing a wide table. Worked around here by
   narrowing the tables to three columns; the exporter itself is unfixed.
9. **`.docs_chain/contexts/{issues,fixed}.yaml` declare the legacy summary
   generators as live transforms** deriving from Markdown rather than the DB. Inert
   today only because the docs_chain engine is absent (G14 SKIPs). If it ever runs,
   it would reproduce exactly the corruption the reconciled G12 gate now blocks.

## Honest coverage boundary

This run repaired sixteen packages and captured the evidence above. It does **not**
establish that the full suite is green: a full-suite sweep was deliberately deferred
because parallel test streams were saturating the host (~4,288 TIME_WAIT sockets, with
one observed ephemeral-port `bind: address already in use`), and evidence gathered
under that contention would be cross-contaminated (§11.4.119).

No release tag was cut. The claim earned by this run is precisely: *these sixteen
packages pass, for these reasons, with this evidence* — not "the suite is green."
