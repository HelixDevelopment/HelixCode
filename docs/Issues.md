# HelixCode — Open Issues Tracker

> Per Constitution §11.4.15 (Item-status tracking) + §11.4.16 (Item-type tracking) + §11.4.19 (Fixed-document column-alignment) + CONST-057 (Type-aware closure vocabulary) + CONST-058 (Reopened-source attribution).
>
> **Authoritative resumption ledger**: `docs/CONTINUATION.md` (CONST-044). This file complements it with item-level granularity for currently-open work.
>
> **Status vocabulary** (closed set): `Queued` | `In progress` | `Ready for testing` | `In testing` | `Reopened` | `Fixed/Implemented/Completed (→ Fixed.md)`
>
> **Type vocabulary** (closed set): `Bug` | `Feature` | `Task`

---

## Prefix convention

Round 189 (2026-05-19) introduced per-project / per-submodule ID prefixes replacing the legacy `ISSUE-NNN` flat namespace. New items MUST use the prefix matching their scope; cross-cutting items affecting two or more submodules (or any meta-repo concern) live under `HXC`. Numeric portion is per-prefix (each prefix starts at `001`). Legacy `ISSUE-NNN` IDs renamed forward-only per CONST-043; historical close-out narratives in `docs/CONTINUATION.md` preserve original IDs verbatim, and `docs/Fixed.md` annotates each closure with `(ex-ISSUE-NNN)` for git-history traceability.

| Prefix | Scope | Source |
|--------|-------|--------|
| HXC | HelixCode root project (project-wide, multi-submodule, governance, infrastructure) | this repo |
| HXA | HelixAgent submodule | `submodules/helix_agent` (when present) / `helix_agent/` tree |
| HXM | HelixMemory submodule | `submodules/helix_memory` |
| HXL | HelixLLM submodule | `submodules/helix_llm` |
| HXQ | HelixQA submodule | `submodules/helix_qa` (`helix_qa/`) |
| HXS | HelixSpecifier submodule | `submodules/helix_specifier` |
| HXO | HelixOrchestrator (= LLMOrchestrator) submodule | `submodules/llm_orchestrator` |
| HXV | HelixVerifier (= LLMsVerifier) submodule | `submodules/llms_verifier` |
| HXD | HelixDocProcessor (= DocProcessor) submodule | `submodules/doc_processor` |
| HXI | HelixI18n (when added) | tba |
| PLN | Planning submodule (vasic-digital) | `submodules/planning` |
| VEN | VisionEngine submodule (HelixDevelopment) | `submodules/vision_engine` |
| SLF | SelfImprove submodule (vasic-digital) | `submodules/self_improve` |
| STO | Storage submodule (vasic-digital) | `submodules/storage` |
| OPS | LLMOps submodule (vasic-digital) | `submodules/llm_ops` |
| VDB | VectorDB submodule (vasic-digital) | `submodules/vector_db` |
| OBS | Observability submodule (vasic-digital) | `submodules/observability` |
| MCP | MCP_Module submodule (vasic-digital) | `submodules/mcp_module` |
| MSG | Messaging submodule (vasic-digital) | `submodules/messaging` |
| MDW | Middleware submodule (vasic-digital) | `submodules/middleware` |
| PLG | Plugins submodule (vasic-digital) | `submodules/plugins` |
| STR | Streaming submodule (vasic-digital) | `submodules/streaming` |
| WAT | Watcher submodule (vasic-digital) | `submodules/watcher` |
| CNV | conversation submodule (vasic-digital) | `submodules/conversation` |
| AUT | Auth submodule (vasic-digital) | `submodules/auth` |
| LZY | Lazy submodule (vasic-digital) | `submodules/lazy` |
| ATP | AutoTemp submodule (vasic-digital) | `submodules/auto_temp` |
| PLI | PliniusCommon submodule (vasic-digital) | `submodules/plinius_common` |
| CHL | challenges submodule (vasic-digital) | `challenges/` |
| CNT | containers submodule | `containers/` |
| SEC | security submodule | `security/` |
| PAN | panoptic submodule | `panoptic/` |

For submodules not listed above, default to the first 3 letters of the submodule name, uppercase (e.g. `Watcher` → `WAT`). Document the new prefix in this table on first use.

### Legacy → new mapping (round 189)

| Old ID | New ID | Scope rationale |
|--------|--------|-----------------|
| ISSUE-001 | VEN-001 | VisionEngine `helix-gitlab` URL |
| ISSUE-002 | VEN-002 | VisionEngine `vasic-digital-github` fork divergent |
| ISSUE-003 | HXL-001 | HelixLLM `analysis_test.go` hardcoded path |
| ISSUE-004 | HXL-002 | HelixLLM TOON `WriteTOON` 500 |
| ISSUE-005 | HXC-001 | CONST-052 rename programme (project-wide) |
| ISSUE-006 | HXC-002 | Round-74 residual LOGIC FAILs (multi-submodule cross-cutting) |
| ISSUE-007 | HXC-003 | CONST-046 migration backlog (project-wide) |
| ISSUE-008 | HXQ-001 | helix_qa `TestPerformance` flake |
| ISSUE-009 | HXA-001 | helix_agent handler tests (4) |
| ISSUE-010 | HXA-002 | helix_agent debate API drift |
| ISSUE-011 | HXA-003 | venice CONST-037 (in helix_agent) |

---

## HXC-153 — TestGuard_GetSystemStatus_WithDB_StillReports is named for a database-present case it never exercises

**Status:** Queued
**Type:** Bug
**Severity:** Medium
**Created-By:** Claude

A regression test in the server package is named as though it proves the system-status endpoint behaves correctly when a real database IS attached, but the test actually builds a server with no database at all and then only checks that the response code is 200. That means it silently re-runs the same no-database case a neighbouring test already covers, and the database-present path it advertises has no coverage from it whatsoever. This matters because the name is what a reader trusts when deciding whether a behaviour is guarded: anyone scanning the suite would reasonably conclude the with-database path is protected when it is not. It is the same documentation-versus-reality defect class as HXC-152, where a file's comment disagreed with its code, and it was surfaced by the independent reviewer of that batch. The test's own comment does honestly defer coverage to another file, so nothing is being falsely asserted about the product — the defect is confined to the misleading name. Done means the test either is renamed to describe what it genuinely checks, or is given a fake-database seam so it earns the name it already has, with the choice backed by a captured run.

## HXC-154 — TestStartQASession_Success asserts on a session field a background goroutine is concurrently changing

**Status:** Queued
**Type:** Bug
**Severity:** Medium
**Created-By:** Claude

A QA-session test checks that a freshly started session reports the status 'pending', but the value it reads is one that a background worker is racing to change to 'running' at that very moment. The server hands the JSON encoder the same live session object the worker mutates, so whether the response says 'pending' or 'running' depends purely on which goroutine wins, and under load or with the race detector enabled the worker can win. The result is a test that usually passes and occasionally fails for reasons unrelated to any real product defect, which erodes trust in the suite and trains readers to dismiss red runs. It was observed failing once during unrelated work, then passed eleven consecutive full-package race runs afterwards, so it is genuinely rare rather than broken outright. The underlying product behaviour is not necessarily wrong — a caller may legitimately see either state — so the likely correct outcome is to make the assertion accept either value or to observe the status through a synchronised read instead. Done means the test no longer depends on goroutine scheduling, proven by repeated runs under the race detector, with the reproduction captured first.

## HXC-155 — Standardise the divergent polarity-switch environment variable convention across the server test package

**Status:** Queued
**Type:** Task
**Severity:** Low
**Created-By:** Claude

The server package's regression guards each support a switch that flips them between reproducing an old defect and guarding against its return, but the switch is spelled four different ways across the files and is read through three different helper styles. Having several names is genuinely useful, because each guard corresponds to a different historical fix and testers need to flip them one at a time rather than all at once, so the goal is not to collapse them into one variable. The problem is that the inconsistency has already caused real defects twice: one file shipped with its default set the wrong way round so its verification checks never ran, and another shipped with a comment describing the opposite of what its code did. Both were fixed, but the drift that produced them remains. This work is to agree one naming pattern and one helper shape, apply it to every guard in the package, and record the convention in a single place so the next guard author copies something correct. Done means every switch follows the documented pattern, every default is the safe standing-guard direction, and a check exists that would catch a future file breaking the convention.

## HXC-158 — No test builds the harmony_os or aurora_os screens, so their widget-threading fixes rest on code review rather than captured runtime proof

**Status:** Queued
**Type:** Bug
**Severity:** Medium
**Created-By:** Claude

The Harmony OS and Aurora OS front-ends were just corrected so that background workers no longer modify on-screen elements directly. That correction is, however, only partly backed by evidence. No test in either package ever constructs the application's screens, so the background workers that paint them are never started while the tests run, and the race detector therefore never gets the opportunity to observe the very code paths that were changed. Concretely, running the Aurora OS package under the race detector reported zero races both before and after the fix — not because the defect was absent, but because nothing under test reaches it; the corresponding Harmony OS failure that was reproduced and fixed came from a different component, the background system monitor, which tests do start. The practical risk is that a future edit could silently reintroduce a direct on-screen write from a background worker in either file and every test would stay green, exactly the situation the anti-bluff policy exists to prevent, since a green suite would then be certifying behaviour nobody has actually exercised. Closing this requires a test that builds the tabs and drives those workers — the chat worker, the provider-health poller and the dashboard and resource timers — with the race detector on, so the fix is proven by observation rather than by inspection, together with a deliberately-broken variant proving the new test genuinely fails when the direct write is put back. Acceptance: a test in each package starts the previously-unexercised background workers against real widgets, passes with zero races over repeated runs, and demonstrably fails when the dispatch helper is removed from any one of those sites. Until that exists, the fixes for those specific sites should be described as review-justified, not runtime-proven.

## HXC-159 — Fully incorporate the HelixSkills System into HelixCode + HelixAgent (all power-features, SpecKit-driven)

**Status:** Queued
**Type:** Task
**Severity:** High
**Created-By:** Operator

**Reported-Via:** §11.4.202 reporting directive `task` on 2026-07-28T19:26:22Z
**Reported-By:** Operator

**What (the report, verbatim):**
Feature description (verbatim):
Fully incorporate the HelixSkills System (git@github.com:HelixDevelopment/skills.git) into BOTH HelixCode and HelixAgent, covering ALL of its power-features with nothing left out.

VERBATIM OPERATOR REQUIREMENTS (2026-07-29):
"Fully incorporate into HelixCode and HelixAgent the HelixSkills System for all work with Skills: git@github.com:HelixDevelopment/skills.git , we MUST fully incorporate all HelixSkills power-features fully without nothing leftout! Full exhaustive implementation plan divided into phases, tasks and subtasks with exhaustive amount of data with all details MUST BE prepared! Full coverage with ll supported test types, challenges and HelixQA test banks (suites) is mandatory! Extend and update all existing documentation, user manuals, guides, FAQs, create stunning illustrations, graphs and diagrams and incorporate them in all materials! Any gaps, shortcomings, weak spots, danger zones, issues, or any inconsistencies MUST BE detected in advance during the planning and implementation process and properly tackled with comprehensive rock-solid solutions implementations, risk-free and safe! Track all progress through the constinuation docs, memory, knowledge and remebering mechanisms we have! Use GitHub SpecKit for all phases of incorporating with bridge to Superpowers used for the implementation and subagents driven development mechanism!"

PRE-VERIFIED ENVIRONMENT FACTS (established 2026-07-29, not assumed — §11.4.6):
- GitHub SpecKit IS installed at /home/milos/.local/bin/specify. It is NOT yet initialized in this repo (no .specify/ and no specs/ directory present). Initialization is therefore an explicit early task, not an assumption.
- git@github.com:HelixDevelopment/skills.git IS reachable via SSH. `git ls-remote --heads` returns branches including feature/catalog-docs, feature/deep-research, feature/testing-infra. The default branch and full branch/tag inventory MUST be re-derived at research time rather than inferred from this partial listing.
- The repository is NOT currently a submodule of this project. `.gitmodules` contains no HelixDevelopment/skills entry; the only 'skills'-matching entry is cli_agents/codex-skills -> vasic-digital/caf-codex-skills.git, which is a DIFFERENT repository and must not be confused with it.
- Both consumers exist in-tree: the inner Go module helix_code/ (module dev.helix.code) and the submodule submodules/helix_agent (module dev.helix.agent).

MANDATORY DELIVERY CONSTRAINTS:
- SpecKit-first: use GitHub SpecKit for ALL phases of the incorporation, bridged to Superpowers for implementation, executed subagent-driven (§11.4.70 / §11.4.20).
- Exhaustive plan structured as phases -> tasks -> subtasks, with an exhaustive level of detail (down to lines-of-code and micro-POCs where the design is load-bearing).
- Complete test coverage per §11.4.169: unit, integration, e2e, full-automation, Challenges (vasic-digital/challenges), HelixQA banks (HelixDevelopment/helix_qa) with autonomous sessions, DDoS, security, stress + chaos, concurrency/atomicity, race/deadlock, memory, benchmarking. Every PASS carries captured evidence (§11.4.5 / §11.4.69); every guard ships a paired §1.1 mutation.
- Documentation: extend AND update every existing doc, user manual, guide and FAQ — not only new ones. Produce illustrations, graphs and diagrams and incorporate them into all materials. Four-format export discipline per §11.4.65 / §11.4.153. Every doc reachable from the main README per §11.4.212.
- Risk work is a PLANNING deliverable, not a post-hoc note: every gap, shortcoming, weak spot, danger zone, issue and inconsistency MUST be detected DURING planning and each one paired with a comprehensive, rock-solid, risk-free and safe solution (§11.4.102 systematic-debugging over all gathered data).
- Reuse before rewrite (§11.4.74 / §11.4.28): survey the vasic-digital and HelixDevelopment catalogues on GitHub AND GitLab first; extend owned submodules in place rather than duplicating, keeping them project-agnostic and decoupled. HelixSkills itself must be incorporated as a properly-laid-out dependency per §11.4.28(C) / CONST-051(C) — root-level or submodules/<name>, never a nested own-org chain.
- Cross-consumer parity: HelixCode and HelixAgent must BOTH receive the full feature set; a capability landing in one and not the other is an incompleteness to be tracked, not silently accepted.
- Progress tracked through the continuation docs, the memory/knowledge/remembering mechanisms, and the workable-items SSoT (§12.10 / §11.4.131 / §11.4.93 / §11.4.95).

ACCEPTANCE CRITERIA:
1. A SpecKit specification exists and is initialized in-repo, covering every HelixSkills power-feature, with an explicit feature inventory proven complete against the upstream repository (an enumerated list, not a sample — §11.4.118).
2. A phase/task/subtask implementation plan exists at the stated depth, with each risk item paired to a concrete mitigation.
3. Every planned capability is wired and end-user-reachable in BOTH consumers, proven by §11.4.108 runtime signatures on a clean target — not merely compiled.
4. The full §11.4.169 test-type matrix is green with captured evidence, Challenges and HelixQA banks included.
5. All documentation is updated and exported, with the produced diagrams/illustrations incorporated, and reachable from README.
6. Independent code review reaches a zero-finding GO per §11.4.125 / §11.4.134 / §11.4.142.
7. No capability is claimed without runtime evidence — the anti-bluff covenant governs every closure (§11.4 / §11.4.1 / §11.4.123).

§11.4.213 FEATURE research-and-planning WORK PROGRAM (scheduled, not yet executed):

(a) Deep web-research series (articles / guides / scientific papers / open-source projects+codebases / real-world examples) on the best way to incorporate this feature into the project (§11.4.8 / §11.4.99 / §11.4.150).
(b) Systematic-debug (§11.4.102) of ALL data obtained to enumerate EVERY weak spot / gap / danger-zone / inconsistency / imperfection, and design a risk-free, rock-solid solution for EACH one found.
(c) The design MUST ALWAYS be enterprise-grade, bleeding-edge, and innovative -- never less.
(d) MANDATORY exhaustive documentation: many technical documents, an implementation plan down to lines-of-code + micro-proof-of-concepts, diagrams / schemes / graphs, illustrations, SQL definitions, templates, and every other relevant material (§11.4.65 / §11.4.73).
(e) Investigate + plan REUSE of existing vasic-digital + HelixDevelopment submodules/components BEFORE proposing a rewrite; extend them freely where genuinely needed while keeping them fully decoupled / project-not-aware / reusable (§11.4.28 / §11.4.74 / §11.4.177).
(f) Plan from the TESTING point of view from day one: every supported test type, the Challenges submodule, and full HelixQA bank coverage (§11.4.27 / §11.4.169).
(g) Divide the work into phases / tasks / subtasks, fine-grained and nano-detailed enough to drive integration / implementation / wiring / testing / scaling directly.
(h) Plan explicitly for enterprise scalability and maximal performance.
(i) When the feature has a UI/UX surface, produce full wireframes / diagrams / design files (Figma / PSD / PDF, or whatever the project mandates) authored via OpenDesign (§11.4.162 / §11.4.190).
(j) Plan full CodeGraph integration with regular / real-time index synchronisation (§11.4.78 / §11.4.79 / §11.4.80).
(k) Create fully-detailed follow-on workable items (every detail + reference + attachment) synced in REAL TIME to the SQLite single-source-of-truth (§11.4.93 / §11.4.95) + every derived workable-items document + every related project doc/component + every connected external work-tracking system (§11.4.148 D5) -- honestly SKIPPING any absent tracker with a machine-readable reason (credentials_absent / tracker_client_absent, §11.4.10) rather than EVER faking a push.

Research-doc destination (to be populated when this item is worked): docs/research/fully_incorporate_the_helixskills_system_into_helixcode_heli_20260728T192622Z_2557683/

Execution model (§11.4.213 clauses 1-2): this item is SCHEDULED, not yet executed. The multi-day research/planning/documentation work above is performed LATER by the project's standing autonomous loop (§11.4.87 / §11.4.94 / §11.4.97 / §11.4.103 / §11.4.126) when it claims this item, and MUST be driven to a genuinely COMPLETED-and-wired or explicitly evidence-backed CLOSED terminal state under §11.4.197 -- this item MUST NOT be left sitting un-wired in the backlog.

**Affected scope / file-scope manifest:**
Feature research/planning -- destination: docs/research/fully_incorporate_the_helixskills_system_into_helixcode_heli_20260728T192622Z_2557683/

**Reproduction / context:**
UNKNOWN: not stated in the report — a reproduction MUST be established before any fix (§11.4.102 / §11.4.146 / §11.4.199)

**Acceptance criteria:**
The full research-doc tree exists under docs/research/fully_incorporate_the_helixskills_system_into_helixcode_heli_20260728T192622Z_2557683/, every enumerated weak-spot/gap/danger-zone carries a designed risk-free solution, an implementation plan down to lines-of-code exists, the planned follow-on workable items are created, and the whole effort is validated per §11.4.197 (never left un-wired).

