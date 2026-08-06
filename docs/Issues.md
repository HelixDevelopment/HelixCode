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

=== ADDENDUM — BINDING OPERATOR REQUIREMENT (2026-07-29, added after scheduling) ===

VERBATIM: "Make sure that SpecKit and Superpowers are used with Bridge extension and with all extensons we have created derrived from constitution Submodule!"

This is not optional colour on the SpecKit mandate — it names specific existing assets that this work MUST compose with rather than duplicate (§11.4.74 extend-don't-reimplement).

THE BRIDGE EXTENSION (verified to exist, 2026-07-29):
constitution/docs/research/extensions/speckit_superpowers/implementation/ — the
"Helix Constitution-Powered SpecKit-Superpowers Bridge". Eight documents, each in
.md/.html/.pdf/.docx: README, IMPLEMENTATION_PLAN, NANO_TASK_ENGINE,
EXTENSION_DEVELOPMENT, TDD_INTEGRATION, CONSTITUTION_INTEGRATION, SECURITY, APPENDIX.

It defines a SEVEN-LAYER architecture:
  1 Developer Workstation
  2 Spec-Kit Core                (governance)
  3 SuperSpec bridge             (orchestration — github.com/WangX0111/superspec)
  4 SuperB extension             (discipline enforcement — speckit-community.github.io/extensions/superb)
  5 SuperBridge MCP              (execution)
  6 Helix LLM                    (inference — github.com/HelixDevelopment/helix_llm)
  7 Distributed host cluster     (llama.cpp RPC)

CONSTITUTION-DERIVED EXTENSIONS THAT MUST BE IN SCOPE (inherited BY REFERENCE per
§11.4.228 — never copied locally, a copy diverges silently):
  constitution/skills/   (7) action-prefix-system, media-validator, multitrack,
                             reporting-workable-items, scheduled-work-queue,
                             session-sync, workable-item-lifecycle
  constitution/mcp/      (2) media-validator-mcp.json, scheduled-work-mcp.json
  constitution/plugins/  (2) helix, scheduled-work
  constitution/actions/      registry.yaml (§11.4.140 action system), subagent_tiering.yaml
  constitution/scripts/hooks/ (7 live) action_prefix_expand.sh, credential_scan_lib.sh,
                             guard-branch-consistency.sh, guard-forbidden-commands.sh,
                             guard-track-branch-label.sh, guard-work-track-binding.sh,
                             post-merge

ALREADY PRESENT LOCALLY — do not re-create:
  .claude/skills/  10 registered speckit-* skills (analyze, checklist, clarify,
                   constitution, converge, implement, plan, specify, tasks, taskstoissues)
  .specify/        workflows/speckit/workflow.yml, integrations/{claude,speckit}.manifest.json,
                   scripts/bash/, templates/, memory/

ADDED ACCEPTANCE CRITERIA:
  8. Every constitution-derived extension above has an explicit disposition — reused,
     extended, or orthogonal-with-reason. Silence is not an answer (§11.4.118).
  9. Each of the 7 Bridge layers is classified WIRED vs DESIGNED-ONLY with a captured
     probe. A design document existing is NOT evidence a layer is installed — that is
     precisely the §11.4.108 SOURCE-vs-RUNTIME gap, and reporting a documented layer as
     an available one would be a §11.4 bluff.
 10. The skill-namespace collision risk is addressed with a designed mitigation:
     HelixSkills is itself a skills system, while §11.4.164's post_update_hook.sh already
     registers constitution skills into .claude/skills/. Two systems registering into one
     namespace needs defined precedence and clash handling, not discovery at runtime.
 11. SuperSpec and SuperB are THIRD-PARTY, non-own-org dependencies (§11.4.74 vendor
     path) — their health, maintenance status and supply-chain exposure must be assessed,
     not assumed from the design doc referencing them.

## HXC-164 — Infrastructure startup script drives containers directly instead of through the shared containers component

**Status:** Queued
**Type:** Task
**Created-By:** Claude

The script that brings up the supporting services for testing talks to the container tooling directly, rather than going through the shared containers component the organisation maintains for exactly this purpose. That component exists so every project starts services the same way, gets the same health checking, and benefits from fixes once rather than repeatedly. Bypassing it means this project carries its own copy of that logic, which drifts and has already caused one outage of its own. A related leftover is that several optional service startups still hide their errors, so a failure to start looks identical to a success. This was flagged honestly in a commit message, but a note in a commit message is not an obligation anyone will act on, which is why it is being tracked here. Closing this means routing the startup through the shared component and removing the remaining error-hiding so a failed start is visible.

## HXC-166 — The agent component has 204 known security advisories against its dependencies, including 5 critical

**Status:** Queued
**Type:** Bug
**Created-By:** Claude

When publishing the agent component, the hosting provider reported 204 outstanding security advisories affecting the libraries it depends on: 5 rated critical, 80 high, 100 moderate and 19 low. These are known, publicly documented weaknesses in third-party code the product ships and runs, so anyone who knows about them knows about them for our deployment too. Nothing here was introduced by recent work; the count has simply been accumulating and was surfaced by a routine publish. The risk is real but unquantified for us specifically, because a published advisory only matters if the affected code path is actually reachable in how we use the library. The right next step is to pull the full advisory list, separate the ones we genuinely reach from the ones we do not, and upgrade or replace the dependencies behind the critical and high findings first. Until that triage is done we cannot honestly claim the component is free of known vulnerabilities.

## HXC-168 — A database password is written directly into published setup files and has never been changed

**Status:** Queued
**Type:** Bug
**Severity:** High
**Created-By:** Claude

The password used to reach the project database is written in plain text inside several setup and container files, and those files are published on all four public code-hosting accounts. Anyone who reads the project can read the password. This is not new and was not introduced by recent work; the project's own earlier security review already recorded it as something that must be replaced, and that has still not happened. The practical risk depends on whether the same password is used anywhere reachable from outside, which has not been established either way and should not be assumed harmless. Because the value is already public, changing the files alone is not enough — the password itself has to be changed everywhere it is used, and the setup files must then read it from a private configuration source rather than containing it. Until both halves are done the project cannot honestly claim its credentials are protected.

## HXC-171 — A copied-in third-party web app carries most of our reported security warnings but cannot even be built

**Status:** Queued
**Type:** Bug
**Severity:** Medium

A complete copy of somebody else's web application sits inside the agent's source tree, and it alone accounts for one hundred and thirty-nine of the two hundred and four reported dependency security warnings, including three of the five most severe. It cannot currently be built or run: the startup configuration points at a build recipe file that does not exist in that folder, and its supporting packages have never been downloaded here. This matters in two opposite ways at once. Because it is not built, those warnings do not describe anything we actually run today, which is why they rank low. But because a startup entry still references it, the setup is broken and misleading, and if anyone ever supplies the missing recipe file, all one hundred and thirty-nine warnings become live at once with no warning to whoever does it. Anyone reading our security reports benefits from removing this distortion, since it currently buries the small number of warnings that genuinely matter. The expected outcome is an explicit decision, recorded in writing: either keep the copy and properly maintain and build it, or remove it and its broken startup entry, or replace it with a reference to the upstream hosted service. Because deleting shipped components requires approval, the decision must be put to the operator rather than taken unilaterally.

## HXC-172 — Three security warnings in a small plugin service we wrote and actually deploy, with no reachability evidence either way

**Status:** Operator-blocked
**Type:** Task
**Severity:** High

A small companion service we wrote ourselves is packaged into a container and started as part of the standard deployment, and it carries three known security warnings in its supporting packages, two of them rated high. Unlike the great bulk of the reported warnings, these have no mitigating evidence in either direction: we cannot say the code is unshipped, because it demonstrably ships, and no analysis has been run to establish whether the affected functionality is ever exercised. This matters because it is the one place in the whole review where something we genuinely run carries unassessed risk, which makes it far more deserving of attention than its small count suggests. One of the three is not really an upgrade problem at all: it concerns a protective setting that is switched off unless explicitly enabled, so updating the package without also turning the protection on would leave the exposure in place while appearing to resolve it. Users of any deployment running this service benefit. The expected outcome is a reachability assessment of the three warnings, the two straightforward package updates applied, the protective setting explicitly verified as enabled in our configuration rather than assumed, and the service rebuilt and confirmed working.

## HXC-175 — An unreachable safety fallback in the model cache could silently serve stale data forever if edited

**Status:** Queued
**Type:** Bug
**Severity:** Low
**Created-By:** Claude

The cache that remembers model information has a small safety fallback for the case where its clock was never configured. That fallback can never actually run today, because the only way to build the cache always configures the clock — so it is untested and unexercised. On its own that is harmless. The reason it is worth recording is what happens if someone later edits it: a plausible-looking change to return an empty time instead of the current time would make every cached entry appear to have been fetched at a point so far in the past that the freshness check reads as a negative age, which never exceeds the expiry limit. Every entry would then be considered fresh forever, and the product would keep serving outdated model information indefinitely with no error and no failing test. The choice is to either add a small test that pins the fallback's behaviour, or remove the branch entirely and document that there is one supported way to construct the cache.

## HXC-178 — The API key check compares secrets in a way that can leak them one character at a time

**Status:** Queued
**Type:** Bug
**Severity:** Low
**Created-By:** Claude

When a caller presents an API key, the server compares it to the configured key using an ordinary text comparison. That kind of comparison stops as soon as it finds a character that does not match, so a key sharing the first few characters takes measurably longer to reject than one that differs immediately. An attacker who can measure those tiny differences across many attempts can recover the key one character at a time without ever guessing it outright. Network timing noise makes this difficult in practice, which is why it is low rather than urgent, but it is a genuine weakness and the remedy is standard and cheap: use the comparison function designed for secrets, which always takes the same amount of time regardless of where the difference falls. This is pre-existing and was not introduced by recent work.

## HXC-179 — A browser-origin check safely allows requests with no origin, but only because a separate login check is wired in

**Status:** Queued
**Type:** Task
**Severity:** Medium
**Created-By:** Claude

The check that decides which websites may open a live connection to our server deliberately allows requests that carry no origin information at all. That is correct behaviour, because only browsers send that information and non-browser clients legitimately omit it. It is safe today only because a separate login check runs first and turns away anyone without valid credentials. The two protections are therefore coupled, but nothing records or enforces that coupling: if someone later removed or reordered the login check while working on something unrelated, every client that omits origin information would become completely ungated, and no existing test would notice. The work here is to add a check that specifically confirms the login step is still wired ahead of the connection upgrade, so the dependency is protected rather than merely true by luck.

## HXC-180 — Three historical commits cannot be built, and this must be documented because it can never be repaired

**Status:** Queued
**Type:** Task
**Severity:** Low
**Created-By:** Claude

Three commits in the current release range do not build. All three share one cause: a test referred to a piece of data that was not added until a later commit, so anyone checking out those three points gets a build error. The tip of the branch is fine and the problem was corrected shortly afterwards. This cannot be repaired, because repairing it would mean rewriting published history, which the project forbids outright. The practical consequence is for anyone hunting a regression by stepping backwards through history: they will hit three points that fail to build for a reason unrelated to whatever they are hunting, and may wrongly conclude the fault lies there. The work here is simply to record the three affected points and their shared cause somewhere a person doing that search will find it, so the failure is recognised immediately as known and irrelevant rather than investigated from scratch.

## HXC-181 — Amend the constitution canon's superseded summary-generator naming and re-cascade to the 129 owned-submodule copies

**Status:** Queued
**Type:** Task
**Severity:** Medium
**Created-By:** Claude
**Assigned-To:** Claude

HXC-160 corrected the 21 stale references to scripts/generate_issues_summary.sh and scripts/generate_fixed_summary.sh across the 12 parent-repo governance carriers, and added the CM-SUPERSEDED-GENERATOR-NAMING gate that guards them. Two reference sets were deliberately left untouched and are the subject of this item. First, constitution/Constitution.md still carries 14 references across several different anchors, including universal rules that projects other than HelixCode consume and historical migration-plan records such as the Phase 5 Go-binary-shim plan. Amending canonical constitutional text is a governance amendment that CONST-049 requires be committed and pushed to every configured upstream, then re-cascaded and pointer-bumped. Pushing was forbidden in the session that closed HXC-160, and a canon amended locally but never propagated is worse drift than the stale wording, so the canon was left alone. Second, 129 files across roughly 45 owned submodules carry cascaded restatements of the same two anchor sentences. The correction applied in the parent repo is deliberately scoped with the words in HelixCode, because the fix names a HelixCode-specific mechanism. Copying that HelixCode-scoped wording into decoupled, project-agnostic submodules would inject project context into them and violate CONST-051(B). The correct sequence is therefore to first agree project-neutral canonical wording that names the workable-items DB exporter generically, land and push it in the constitution submodule per CONST-049, and only then re-cascade to the fleet and bump pointers. Who benefits: every consuming project whose manuals inherit the anchor text, and any agent reading a submodule manual to learn how tracker summaries are produced. Acceptance: canon names the DB exporter in project-neutral wording, the change is pushed to every constitution upstream, all 129 submodule copies are re-cascaded, submodule pointers are bumped, and the cascade verifier plus CM-SUPERSEDED-GENERATOR-NAMING both pass with scope widened to the fleet.

## HXC-188 — The project handbook describes a folder layout the project no longer has

**Status:** Queued
**Type:** Task
**Severity:** Low
**Created-By:** Claude

The main handbook that tells contributors and automated helpers how this project is organised lists several folders at the top level that do not exist. It names four internal areas when only one is present, and refers to a top-level commands folder that is absent entirely. Anyone following it — a new contributor, or an automated helper reading it for orientation — will look for things that are not there, and may conclude the project is broken or that they have checked out the wrong thing. It also undermines trust in the rest of the document, which is otherwise the authoritative description of how to work here. The correction is small and purely descriptive: measure the actual layout and update the section to match, then keep the two in step as the layout changes.

## HXC-189 — A dependency is pinned to an old project name that only works because of a redirect

**Status:** Queued
**Type:** Task
**Severity:** Low
**Created-By:** Claude

One of the external projects this codebase depends on has been renamed. Our configuration still refers to it by its previous name, which currently works because the hosting service silently forwards requests from the old name to the new one. That forwarding is a courtesy, not a guarantee: it disappears if anyone ever creates a new project under the old name, and it is not honoured by every tool that might fetch the dependency. If it stops working, fetching the project fails with an error that points at a name nobody recognises, which is unusually confusing to diagnose. The fix is to record the project's current canonical name so the reference stands on its own rather than depending on a redirect continuing to exist.

## HXC-190 — Commands stopped by a timeout are reported as though they were not

**Status:** Queued
**Type:** Bug
**Severity:** Low
**Created-By:** Claude

When a running command is cut short because it took too long, the result that comes back says it was not a timeout. The check that sets that flag looks at the wrong clock: it inspects the caller's own deadline, which usually has none at all, rather than the timer that actually stopped the command. Investigation showed the obvious alternative would not work either, because the internal signal it would consult can only ever report a plain cancellation and never a timeout, and the timer that does the stopping is a separate mechanism entirely. The effect is that anyone diagnosing a slow command sees a generic failure and no indication that a time limit was the cause, which sends them looking in the wrong place. A sibling code path that runs commands the ordinary way sets the flag correctly by doing it inside the timer's own callback, so a working pattern already exists to copy.

## HXC-191 — Cancelling a command cannot actually stop it when the protective sandbox is switched off

**Status:** Queued
**Type:** Bug
**Severity:** Low
**Created-By:** Claude

When a command is cancelled or times out, the system tries to stop not just that command but anything it started, by signalling the whole group of processes together. That only works if the command was placed into its own group when it began. With the protective sandbox enabled — the normal configuration — it is, and cancellation works. With the sandbox switched off, the grouping step is skipped but the stop attempt still asks the operating system to signal a group that does not exist, which fails silently because the error is discarded. The result is a cancelled command that keeps running. Today this is dormant, because the only configuration that disables the sandbox is used by a single test, so no shipped path reaches it. It is worth fixing because the failure is completely silent and would appear only in an unusual configuration, which is the hardest kind of problem to diagnose later.

## HXC-192 — A single over-long line of output permanently stops reading, which can freeze the command producing it

**Status:** Queued
**Type:** Bug
**Severity:** Low
**Created-By:** Claude

Output from a running command is read line by line, with a limit on how long any single line may be. When a line exceeds that limit the reader stops — not just for that line, but permanently for that stream, because the buffer is fixed at the limit and can never grow past it. Nothing else takes over the reading. The command on the other end keeps writing until the operating system's own small buffer fills, and then it blocks forever waiting for someone to read. So one unusually long line, which programs emitting compact machine-readable output produce quite naturally, can wedge the command that produced it. The fix is to keep draining and discarding the remainder of an over-long line rather than abandoning the stream, so the producing command is never starved of a reader.

## HXC-196 — The project handbook names a component version the code no longer uses

**Status:** Queued
**Type:** Task
**Severity:** Medium
**Created-By:** Claude

The handbook that describes this project's technology stack names a specific version of one of its networking libraries. Both parts of the codebase now use a newer version, because a security fix required moving off the older one. The upgrade was correct and deliberate; it is the handbook that is now wrong. This matters because that section is what a new contributor or an automated helper reads to learn what the project runs on, so it will confidently state a version that has not been used for some time — and because the same document has already been found describing folders that no longer exist, each additional inaccuracy makes the whole document less trustworthy. The fix is to update the recorded version to what is actually in use, and to note that the security upgrade is the reason, so nobody later reverts it believing the handbook was correct.

## HXC-197 — A directory tree contains a stale copy of itself nested several levels deep

**Status:** Queued
**Type:** Bug
**Severity:** Low
**Created-By:** Claude

One test directory contains a duplicate of its own path nested inside itself, so the same ten files appear twice in the project at two different depths. It was created by an earlier renaming exercise that moved a tree while leaving a copy behind at the old location inside the new one. Nothing reads the nested copy, so it causes no failure today, but it means anyone searching the project finds two results for every one of those files and cannot tell which is real, and any tool that scans for duplicate declarations reports it forever. It also blocks a useful check from being switched on: a guard that detects accidentally-nested duplicates cannot be enabled while this one exists, because it would report a failure on every run. Removing it requires first confirming through the project history that the nested copy is genuinely the leftover and not the original.

## HXC-199 — A check for a just-fixed problem would still report it as broken, because it matches on a fragment

**Status:** Queued
**Type:** Bug
**Severity:** Medium
**Created-By:** Claude

A recent change gave one part of the project a distinct name so that two different pieces of code could stop claiming the same identity. A diagnostic script that checks whether that problem still exists searches for the old name as a fragment of text, and the new name contains the old one as a prefix — so the check still matches and would report the problem as unfixed even though it is fixed. Anyone re-running it would be told to redo work that is already done, and might undo the fix believing it had never worked. The project handbook also still describes the old arrangement in three places. The fix is to make the check match the whole name rather than a fragment, and to update the handbook so both agree with what the code actually does now.

## HXC-200 — A leak detector recognises only one of the two shapes the leak can take

**Status:** Queued
**Type:** Bug
**Severity:** Low
**Created-By:** Claude

A detector was written to catch a specific kind of stall where a background worker gets stuck trying to hand off a result nobody is collecting. It identifies that stall by looking for one particular description of what the worker is waiting on. There is a second, equally real form of the same stall — where the worker waits on a choice between several possibilities rather than a single handoff — and the detector does not recognise it. This was demonstrated rather than theorised: a deliberately introduced leak of the second form went completely undetected while the detector reported everything healthy. The shape it does catch is the historical one, so the guard is not useless, but a future change that keeps the newer structure while breaking its escape route would slip past. Widening it to recognise both descriptions is a small change and restores the guard to covering the whole defect rather than half of it.

## HXC-204 — Two more endurance tests report failure whenever the machine is busy

**Status:** Queued
**Type:** Bug
**Severity:** Medium
**Created-By:** Claude

Two long-running tests — one exercising heavy simultaneous access to the memory store, one exercising leader election among worker nodes under deliberately induced faults — report failure when the machine is under load, and pass comfortably when run on their own. Measured: during a full test run both failed, while in isolation they passed three times consecutively, one finishing in roughly a fifth of its allowed time and the other in about a twentieth. Their verdicts are therefore reporting how busy the machine was, not whether the product works. This matters most at exactly the wrong moment: the complete test run performed before a release is by definition the busiest the machine ever gets, so these two will mark a healthy release as broken and train everyone to wave the result through. A third test with the same problem was already corrected by making an inconclusive run report as skipped rather than failed, while keeping its real checks strict; the same treatment fits here. Both should wait for the condition they care about rather than assuming a fixed time is enough.

## HXC-206 — A provider stopped and restarted during a health probe can have a stale verdict applied

**Status:** Queued
**Type:** Bug
**Severity:** Low
**Created-By:** Claude

The newly-fixed health refresh deliberately releases its lock while it contacts each provider over the network, so that a slow or hanging provider cannot block everyone else. It then re-acquires the lock and only applies its verdict if the provider's state has not changed underneath it. There is a narrow remaining window: if a provider is stopped and started again entirely within one probe, the check sees the same state it started with and applies a verdict formed against the previous instance. The result would be a provider briefly reported unhealthy when it is fine, which the next refresh corrects, and the dangerous direction — reporting a stopped provider as healthy — is already prevented. Worth recording that this is cheap to close rather than expensive: a hidden counter incremented on each state change is invisible to the outside world and requires no change to any published structure. An earlier note describing it as a published-interface change was wrong and has been retracted.

## HXC-207 — Every model manager shares one settings map by reference, so a future write would corrupt all of them

**Status:** Queued
**Type:** Bug
**Severity:** Low
**Created-By:** Claude

When a model provider record is created, its environment settings are taken from a shared template by reference rather than copied. Every record in every manager therefore points at the same underlying settings, including across separate manager instances. Nothing writes to those settings today, and the values handed out to callers are copied, so no corruption occurs at present — this was verified rather than assumed. The reason to record it is that the protection now added to these managers is per-instance: a lock in one manager cannot protect a structure shared with another. So the day someone adds a legitimate, properly-locked write to a provider's settings, it will silently alter every other manager's view as well, and the locking will look correct while failing. Copying the settings when the record is created removes the hazard permanently and costs nothing.

## HXC-208 — Two mock services are wired into e2e compose stacks that cannot build them

**Status:** Queued
**Type:** Bug
**Created-By:** Claude

Four compose entries across two end-to-end test stacks declare a build context pointing at the Slack mock and the LLM-provider mock, but neither directory contains a Dockerfile. Any attempt to build those stacks therefore fails at the build step rather than at run time, which means the containerised form of these two services has almost certainly never been exercised. This was surfaced while fixing an unrelated permissive-origin defect in the same two services: the fix could be proven at the source and test-process level but not against a running container, precisely because no container of either service can be produced. That gap matters beyond these two mocks, because a fix validated only in-process leaves the deployed-artifact layer unproven, which is exactly the class of silent failure the four-layer verification discipline exists to catch. The work is to add the missing Dockerfiles, or to remove the build entries if these services are genuinely never meant to run containerised, and then to prove the choice by building the stack.

## HXC-216 — Thirty-five historical evidence folders hold runs that were never saved to the repository

**Status:** Queued
**Type:** Task
**Created-By:** Claude

A blanket rule excluding log files from version control was also excluding captured test evidence, because evidence transcripts are written as log files. The rule has now been corrected so this cannot happen again, and the three folders backing recently closed items have been recovered and committed. Thirty-five older folders remain in the same state: the runs exist only on the machine that produced them, so anyone cloning this project sees an evidence folder that appears empty, which is indistinguishable from a test that was never run. Recovering them is not automatic, because two of the files alone account for seventy-eight megabytes and committing those into permanent history is a poor trade that deserves a deliberate decision rather than a reflex. The work is to review the thirty-five, keep what genuinely substantiates a closed item, trim or summarise the two very large ones rather than storing them whole, and record explicitly which ones are being let go and why, so the gap is visible rather than silent.

## HXC-217 — Fifteen closure records describe their proof in prose instead of pointing at it

**Status:** Queued
**Type:** Bug
**Created-By:** Claude

When an item is closed, the record carries a field meant to hold the location of the captured proof, so that anyone can go and look at it and so that a machine can confirm the proof genuinely exists. For fifteen closures that field instead contains a paragraph describing what was proved. The descriptions are substantive and in every case checked the underlying evidence does exist elsewhere on disk, so nothing is actually missing and no closure is unfounded. The cost is that those fifteen cannot be verified mechanically: a check that asks whether each closure points at something real will report them as pointing at nothing, which is indistinguishable from a closure with no proof behind it at all. That ambiguity is the defect, and it matters because a sweep that cannot tell a well-documented closure from an empty one will eventually be believed about the wrong one. The work is to move the narrative into the record's description where it belongs, put the actual location in the location field, and add a check that refuses a closure whose stated location does not resolve.

## HXC-218 — Container health checks still fail for IPv6 addresses, so starting the scan service breaks

**Status:** Queued
**Type:** Bug
**Severity:** High

When the security scanning tool is asked to START its supporting services, it checks whether each service is healthy by building a network address from a host and a port. For ordinary addresses this works, but modern IPv6 addresses contain colons of their own, so they must be wrapped in square brackets before a port is attached. The shared container code does not do that wrapping in two places, so the address it builds is malformed and the health check can never succeed on an IPv6 machine. A companion fix already repaired the STATUS command, but START goes through a different path in the shared container library and was not covered, which means operators on IPv6 hosts still see startup fail for what looks like no reason. The obvious shortcut of pre-wrapping the address is unsafe, because another part of the same library compares the raw unwrapped form when deciding whether a service is local, and wrapping it would silently break that check. The benefit of fixing this properly is that operators running on IPv6 networks can start and monitor the scanner normally instead of hitting a confusing dead end. Success means the start command completes its health checks on an IPv6 host exactly as it does on an IPv4 one.

## HXC-219 — Evidence-gate handbook never explained the new citation rule, and its exported copies are stale

**Status:** Queued
**Type:** Task
**Severity:** Medium

A recent repair changed how the release gate decides whether a piece of recorded test evidence really belongs to the change it claims to document. Evidence must now name its commit on a labelled line rather than merely happening to contain the right code somewhere in the text. That new requirement was never written down for the people who have to follow it: the guide that authors read before recording evidence says nothing about the new labelled field, so a contributor following the current instructions can produce evidence the gate will reject without understanding why. Separately, the reference page describing the gate itself has printable and web copies that were generated well before the page was last edited, so all three versions disagree with each other and with the tool they describe. Fixing this means updating the author-facing guide to describe the labelled citation field and regenerating the exported copies so every version matches. The people who benefit are contributors recording evidence and reviewers reading it, who currently have no accurate written description of the rule being enforced. Success means someone can follow the written guide and produce evidence that passes the gate on the first attempt.

## HXC-220 — The module-name check has no permanent test, so the fixed false alarm could return unnoticed

**Status:** Queued
**Type:** Task
**Severity:** High

A diagnostic script used to report that two parts of the project shared the same module name, when in fact they no longer did. The cause was that it looked for the shorter name inside the longer one instead of comparing the two names as wholes, so a rename that deliberately made them different was still reported as a clash. The comparison itself has been corrected and behaves properly today, but no permanent automated test was ever added to keep it correct. That matters because this exact kind of check is easy to rewrite carelessly during future cleanups, and if the loose comparison came back nobody would notice until the script again told an engineer that finished work was unfinished, risking someone undoing a correct change. The project requires every fixed defect to leave behind a standing test that fails if the defect returns, and that test is the only piece still missing here. The people who benefit are engineers relying on this diagnostic to tell them the true state of the project. Success means a registered test exists that passes on the current corrected code and fails if the loose name matching is ever reintroduced.

## HXC-221 — When secure randomness fails, API key generation quietly falls back to a guessable value

**Status:** Queued
**Type:** Bug
**Created-By:** Claude

The agent component issues API keys for callers of its protocol service. It builds each key from sixteen bytes of cryptographic randomness, which is correct. But if the system's randomness source ever fails to answer, the code does not stop — it silently substitutes the current clock reading and hands back a key derived from the time of day. A key made that way is guessable by anyone who can estimate when it was issued, and the caller receiving it is told nothing: it looks like an ordinary key and is accepted as one. The function is already able to report a failure to its caller, so the ability to refuse safely exists and is simply not used. The likelihood of the randomness source failing is genuinely low, which is why this has gone unnoticed, but the consequence if it does is that the service starts issuing predictable credentials with no signal that anything is wrong. Failing loudly is the correct behaviour here: a caller that receives an error can retry or abort, whereas a caller that receives a weak key has no way to know it must. The fix is to return the error instead of the clock reading, and to add a test that forces the randomness source to fail and confirms no key is produced.

## HXC-222 — Downloaded third-party libraries are committed into the agent repository, 104 megabytes of them

**Status:** Queued
**Type:** Bug
**Created-By:** Claude

The agent component has two folders of downloaded third-party libraries committed into version control: 7,870 files under its web toolkit and 1,037 under its plugin service, together about 104 megabytes. These are not our code. They are fetched automatically from a package registry using the manifest files that sit beside them, which means every one of them can be recreated on demand and none of them needs to be stored. Committing them makes every clone of the repository permanently larger and makes it hard to see our own changes among theirs, because any routine dependency update rewrites thousands of files at once. The project already has a rule against versioning anything that a documented mechanism can regenerate, and the ignore file does not currently list these folders, so nothing prevented them from being added. The fix is to stop tracking both folders, add them to the ignore file, and confirm the documented install step still reproduces them exactly on a fresh checkout — the last part matters, because removing them is only safe if the recreation path is proven to work.

## HXC-223 — The safety check that stops half-finished experiments also blocks the proof they were finished

**Status:** Queued
**Type:** Bug
**Created-By:** Claude

Before every commit, a check scans the files being committed for markers left behind by a deliberate sabotage experiment — the kind where a working safeguard is temporarily broken to prove it really notices. Leaving such a marker in real code would be dangerous, so blocking it is correct. But the same experiment is required to record what it did, and that record necessarily quotes the sabotaged lines verbatim as its proof. The check cannot tell the difference between code that is still broken and a written account of code that was broken and then repaired, so it refuses the account. The two rules therefore contradict each other: one demands the proof be captured, the other forbids it from being stored. This is not hypothetical — it happened today and the proof file had to be left out of its own commit, which is exactly the situation where evidence quietly goes missing. The fix is to teach the check where it is looking: markers inside a recorded transcript under the evidence folder are a description of past work, while the same markers in live source are a real hazard. The check must keep refusing the second while permitting the first, and must be tested against both so it cannot drift back to refusing everything or accepting everything.

## HXC-225 — One vendored helper server accounts for two thirds of all our reported vulnerabilities

**Status:** Queued
**Type:** Task
**Created-By:** Claude

A single bundled helper component inside the agent project is responsible for 140 of the roughly 208 security advisories reported against the whole project — about two thirds of the total — and it entered the codebase without ever being discussed. Nobody has established what it does, whether we wrote it or copied it in, whether anything we ship actually uses it, or whether removing it would cost us anything. Until those questions are answered the headline advisory count is misleading: it reads as though the agent project carries enormous risk, when in reality the great bulk of it sits in one component that may not even be part of what we deliver. The work is to determine the component's origin and purpose, whether it is reachable from anything we build or run, and then to make a decision about it — keep and maintain, upgrade its dependencies, or remove it — and record which was chosen and why. The decision matters more than the individual advisories, because upgrading dependencies inside a component we do not need would be effort spent on nothing.

## HXC-187 — Two different pieces of code claim the same identity, so which one gets used depends on where the build starts

**Status:** Reopened
**Reopened-Details:** By: AI; On: 2026-08-05; Reason: captured-evidence-contradicts; Evidence: docs/qa/hxc217_evidence_path_resolve_20260805T082033Z — HXC-217 closure-evidence resolvability audit. This item's closure record cited commit 09a086a6 (go.mod, 1 file, +1/-1) and unlogged build/vet results, with no captured artefact. A repo-wide search returns ZERO tracked paths and ZERO on-disk directories matching 'hxc187'. The commits are real and the rename may well be correct — but a commit reference is not captured runtime evidence (§11.4.5 / §11.4.123), so the closure's warrant cannot be produced on demand. Reopened to capture a real runtime signature for the module-identity invariant (§11.4.108), not to redo the rename.
**Type:** Bug
**Evidence:** commit 09a086a6, go.mod only (1 file, +1/-1). Root module renamed dev.helix.code -> dev.helix.code/meta; ZERO import updates were needed because the root module genuinely had no importers. TWO-METHOD verification, and method 1 alone would have produced two FALSE POSITIVES: a git ls-files sweep hit an ASCII-art string literal ('dev.helix.code v1.0.0') and prose in doc comments, neither an import. Method 2 (filesystem, sees submodules + untracked) found all 947 quoted imports under helix_code/ and zero elsewhere; sweep validity proven rather than assumed (9944 .go files under submodules/, 1909 under cli_agents/ — it had real content to see). Reversal conditions each cleared: no go.mod requires/replaces the root; scripts/audit_const046 is an independent module not a consumer; and D-7's 'no go.work exists' was imprecise — two do exist but neither names dev.helix.code. SEMANTIC PROOF the collision is gone: dev.helix.code/internal/theme now resolves ONLY from the inner module; from the root it reports 'no required module provides package'. Build/vet for the root module's own 5 packages: 0/0. The agent deliberately did NOT delete the root stub (removal is its own 11.4.124 decision) and did NOT touch skill_registry/U-12 (it cannot be fixed in isolation — reconciling it repairs helix_agent's replace but BREAKS helix_llm's currently-correct one). Gate deliberately left UNREGISTERED and unweakened: its premise 'exactly one duplicate' proved stale — 118 live modules, 8 duplicate groups, 7 surviving the rename (1 ours: a self-nested tests/e2e/orchestrator copy from refactor cc339fc0; 6 in another repo's nested submodule content). Registering a failing gate would break the sweep for all live agents, and narrowing its exclusions would destroy its ability to detect an accidentally-nested duplicate in our own tree (assertion-weakening under 11.4.120). Both refusals correct.
**Severity:** High
**Created-By:** Claude

The project declares the same module identity in two places: once at the top level and once for the real application inside it. Because of that, one particular internal name refers to two completely different pieces of code — a five-line placeholder at the top level, and the real fifty-kilobyte subsystem inside. They share no files at all. Which one any given piece of code actually receives depends entirely on which directory the build was started from, which is not something a developer would ever expect or notice. Nothing is broken today only because the top-level placeholder has no users, but that is luck rather than design, and the first time someone adds one the behaviour will differ between two builds of the same source with no error to explain it. The fix is to give the top-level module a distinct identity so the collision cannot occur. Removing the placeholder is a separate decision and is deliberately not bundled with this.

## HXC-224 — Closure-evidence pointers are unchecked free text, so proof of completed work silently becomes unreachable

**Status:** Queued
**Type:** Bug
**Severity:** High
**Created-By:** Claude
**Evidence:** docs/qa/hxc217_evidence_path_resolve_20260805T082033Z

When we mark a piece of work finished, we record a pointer to the proof that it really works. Nobody ever checked that those pointers actually lead anywhere. As a result three different kinds of rot accumulated silently and none of them were visible to any test or report. First, some pointers hold a whole paragraph of explanation instead of a location, so there is no way for a machine to follow them. Second, some point at a folder that was never created because the person doing the work saved the proof somewhere else, which means the proof exists but nobody can find it from the record. Third, some are attached to ordinary progress updates rather than to the actual completion, which makes routine notes look like formal proof. The effect is that a completed item can look fully evidence-backed in every summary and dashboard while the evidence is unreachable, and the only way anyone finds out is by manually hunting for it long afterwards. The work here is to make the completion pointer a checked field: it must point at something that genuinely exists, it must be attached only to real completions, and anything that fails those rules must be refused at the moment someone tries to record it rather than discovered months later. This protects everyone who relies on our finished-work reports being true, because a claim of proof that cannot be produced on demand is worse than no claim at all.

### Forensic anchor (FACT, HXC-217 audit 2026-08-05)

Of 127 closure records carrying an evidence pointer, **16 (across 14 tickets) did
not resolve**. Systematically resolving every one:

- **4 rows** (HXC-107, HXC-108 x2, HXC-112) were `event_type='Updated'`,
  `reason='operator-blocked'` **progress notes, not closures**. Their real
  closure events resolve cleanly. These were false positives of the audit's own
  population query — a guard that refuses a valid state (§11.4.201(1)).
- **4 rows** (HXC-124, HXC-131 x2, HXC-132) were **path drift**: substantive
  git-tracked evidence exists at `scratch/discovery/fixes/*_evidence.md`; the DB
  recorded a `docs/qa/...` path that was never populated. An earlier pass
  concluded these files "never existed" because it searched the recorded (wrong)
  full path instead of the basename across full history.
- **7 rows** (HXC-158, 161, 165, 167, 182, 184, 203) held **narrative in the
  path field**; every referent proved producible and has been repointed.
- **1 row** (HXC-187) had **no producible artefact at all** — commit references
  only. Reopened separately under §11.4.34.

### Closure criteria

1. `workable-items validate` refuses a closure whose `evidence_path` does not
   resolve, scoped to genuine closure events so it cannot false-positive on
   progress notes.
2. A paired §1.1 mutation proves the guard is not a tautology.
3. The population query distinguishes closure events from historical updates.
## HXC-226 — The copy of the agent project we actually pull from reports no vulnerabilities at all

**Status:** Queued
**Type:** Bug
**Created-By:** Claude

The agent project is mirrored to two hosting accounts, and automatic vulnerability alerting was only ever switched on for one of them. Queried today, one mirror reports two hundred and ten open alerts while the other reports none. The one reporting none is the one the main project is configured to pull from, so anyone who checks the security posture of the component we actually consume is shown a clean result while more than two hundred findings sit unseen on the sibling copy. Nothing is wrong with either set of code; the two mirrors hold the same commits. What differs is that only one of them is being watched. This is dangerous precisely because the answer looks reassuring: a report of zero findings is indistinguishable from a report of zero findings because nobody is looking, and the second case is what we have. The work is to switch alerting on for the mirror we pull from, confirm both then report the same figures, and add a periodic check that compares the two so a future divergence is noticed rather than trusted. Until that is done, any statement about this component's vulnerability status must name which mirror it came from.

