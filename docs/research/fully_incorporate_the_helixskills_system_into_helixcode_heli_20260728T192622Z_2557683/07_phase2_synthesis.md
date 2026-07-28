# HXC-159 Phase 2 — Specification + implementation-plan synthesis

| Field | Value |
|---|---|
| **Document** | `07_phase2_synthesis.md` |
| **Work item** | HXC-159 (Task, In-progress, High) |
| **Phase** | 2 — SpecKit specification + phase/task/subtask implementation plan |
| **Revision** | 1 |
| **Last modified** | 2026-07-29 |
| **Author** | `(T1/main - claude1 - opus - xhigh)` |
| **Status summary** | Specification and plan authored at `specs/001-helixskills-incorporation/`. **63 tasks** across **13 phases** in 2 concurrent tracks. **29 register risks → 29 mitigations → task coverage**, plus **3 new risks** found during planning. **9 `UNCONFIRMED:` items** carried, 2 resolved during this phase. Nothing implemented. |
| **Anchors** | §11.4.6 · §11.4.8 · §11.4.35 · §11.4.74 · §11.4.108 · §11.4.118 · §11.4.122 · §11.4.169 · §11.4.201 · §11.4.224 · CONST-051(C) · CONST-059 |

**Phase 2 does not re-derive Phase 1.** Its value is synthesis: turning the six
Phase-1 documents into a specification with per-feature dispositions, a sequenced plan,
and a risk→mitigation→task map. Where Phase 2 *corrected* a Phase-1 or brief-level
claim, the correction is recorded in §2 rather than silently absorbed.

## Table of contents

1. [Deliverables](#1-deliverables)
2. [Corrections to inherited framings](#2-corrections-to-inherited-framings)
3. [The six open decisions, resolved](#3-the-six-open-decisions-resolved)
4. [Sequencing rationale](#4-sequencing-rationale)
5. [Risk coverage](#5-risk-coverage)
6. [What was written vs planned](#6-what-was-written-vs-planned)
7. [Honest boundary](#7-honest-boundary)

---

## 1. Deliverables

| Artefact | Path | Contents |
|---|---|---|
| **Specification** | `specs/001-helixskills-incorporation/spec.md` | 5 prioritised user stories · **50 functional + 4 non-functional requirements** · all **57** upstream capability rows dispositioned · all **23** constitution-derived extensions dispositioned · all **7** Bridge layers dispositioned · 14 success criteria · 9 `UNCONFIRMED:` items |
| **Implementation plan** | `specs/001-helixskills-incorporation/plan.md` | Technical context · constitution check · structure decision · **13-phase / 2-track** architecture with per-phase rationale · finding-by-finding handling · risk→mitigation→gate→mutation table · test strategy · **22-entry runtime-signature registry** · complexity tracking · doc deliverables |
| **Task breakdown** | `specs/001-helixskills-incorporation/tasks.md` | **63 tasks**, each with scope / §11.4.169 test types / §11.4.108 runtime signature / acceptance evidence / risks, decomposed into **~230 subtasks** · dependency graph · parallel-execution guidance |
| **SpecKit constitution pointer** | `.specify/memory/constitution.md` | R-23 mitigation — **written, not merely planned** (§6) |

SpecKit conventions were followed for the `specs/<###-feature>/{spec,plan,tasks}.md`
layout and the template section structure. **`create-new-feature.sh` was deliberately
not run**: it creates a git branch, and branch creation in a shared tree with eleven
live agents is out of this task's remit. The directory was created at the path the
script would have chosen.

---

## 2. Corrections to inherited framings

Three claims arrived with the Phase-2 brief. Each was checked against the artifact
(§11.4.6), and two needed correcting. Recorded here because a plan built on a
mis-framing inherits the mis-framing.

### 2.1 "The upstream repo is nearly empty of implementation" — **PARTIALLY WRONG**

**Measured** (`02` §1, §4, §11): 237 Go files · 1 418 top-level declarations ·
**755 exported symbols** · 26 packages · 13 extension-point interfaces · 12 migrations ·
13 MCP tools · 27 CLI subcommands · **0 parse errors**. That is a production-shaped
service, not an empty repository.

What *is* true is the narrower and sharper statement, and it is the one the plan uses:

- **0 `.go` files outside `docs/`** — every one is under
  `docs/research/mvp/Agent_AI_Skill_Tree_Development/project/`;
- **no root `go.mod`**, so `go get` against the repository URL cannot resolve it;
- the declared module path `github.com/helixdevelopment/skill-system` **disagrees with
  the repository URL** `github.com/HelixDevelopment/skills` in *both name and case*;
- **no release tags**, so any pin must be a raw SHA.

**Why the distinction changes the plan.** "Nearly empty" implies *write it*. The
measurement says *relocate and reconcile it* — a smaller, safer operation that preserves
755 exported symbols instead of reinventing them. The brief's operative conclusions
survive intact (**extend, not reuse**; **raw-SHA pin**); only the size of the work
changes, and it shrinks.

`06`'s R-01 heading ("ships nothing to incorporate") is consistent with this once read
precisely: nothing *consumable at a resolvable module path*. Phase 2 states it that way
to remove the ambiguity.

### 2.2 "2 of 7 Bridge layers exist" — **SUPERSEDED mid-phase**

A dedicated layer-health probe landed during this phase with a materially different and
better-resolved result:

**`WIRED: L1, L2, L6` · `PARTIAL: L7` · `DESIGNED-ONLY: L3, L4` · `NONEXISTENT: L5`.**

The load-bearing change is that **the remedy is mixed, not uniform** — three absences
need three different responses, and a uniform installer would necessarily contain a step
that can never succeed:

| Class | Layers | Correct response |
|---|---|---|
| **ABSENT-BUT-OBTAINABLE** | L3, L4 | An installer genuinely fixes these — but **must use an explicit `--from` release URL**, because the bare `specify extension add` fails on a default install (the first-party catalogue holds 4 entries, all `spec-kit-core`; the bridges live in the community catalogue) |
| **NONEXISTENT** | L5 | **Delete the tier.** No installer can help — there is nothing to install |
| **ABSENT-BUT-BUILDABLE** | L7 | A **build flag** (`-DGGML_RPC=ON`; the submodule is already in-tree and already builds `llama-server`), not a procurement item |

**L6 was a false negative**, and the mechanism matters more than the correction:
`systemctl list-unit-files | grep -i helix` at **system** scope returns nothing because
all seven Helix units are **user**-scope. A scope-limited probe produced a *confident*
wrong answer — the §11.4.201 false-null class. FR-046 therefore requires the probe to
enumerate both scopes, so the trap is **encoded rather than remembered**. This is logged
as new risk **R-30**.

### 2.3 The work item's own environment facts are **stale**

HXC-159's `PRE-VERIFIED ENVIRONMENT FACTS` block states SpecKit *"is NOT yet initialized
in this repo (no `.specify/` and no `specs/` directory present)"* and that
*"Initialization is therefore an explicit early task."* `.specify/` **exists** with full
scaffolding and ten registered `speckit-*` skills; the item's own later addendum says so.
The tracker contradicts itself and **the addendum is correct**.

Consequence: **no L2 initialisation task is planned** (FR-045). Logged as **R-31**, with
the general mitigation *re-derive environment facts from the artifact at phase start*
(task T-P0.01) — because a plan that re-plans a completed step is working against a
description instead of against the artifact.

---

## 3. The six open decisions, resolved

`03` §8 left six decisions to Phase 2. All six are taken in `spec.md` §2; summarised
here with the evidence that forced each.

| # | Decision | Forced by |
|---|---|---|
| **D-1** Canonical schema = **Agent Skills open standard**; the two proprietary dialects become profiled extensions with round-trip converters | It is now an open cross-vendor standard with ~45 implementations. Picking either dialect strands the other; picking the standard strands neither and yields §11.4.228 portability for free |
| **D-2** All three upstream layers in scope, ordered L1 → L3 → L2, with **L2 never linked into a consumer binary** | The `internal/` visibility wall + R-25 (`replace` ignored outside the main module) |
| **D-3** Contract crosses the module boundary **over the wire (MCP) plus a pure-data `pkg/` loader** | `03` §2/§6: `pkg/` and the wire are the *only two proven* cross-module seams. `03b` §5 calls the MCP-config path "the cleanest cross-consumer integration path in the entire analysis" |
| **D-4** **HelixAgent does not depend on HelixSkills** — it becomes a client | Upstream's own `helix-deps.yaml` declares HelixAgent a dependency *of* HelixSkills. Inverting it risks a cycle CONST-051(C) forbids. Enforced mechanically by T-P6.05's module-graph gate |
| **D-5** Fix the two HelixCode structural defects **first** | No exported `Skill` constructor blocks the loader seam; the discarded CLI dispatcher kills auto-trigger on the exact surface the runtime signature is taken from |
| **D-6** The 1 174-file corpus **does not migrate** in this feature | Relocating it removes a tree from HelixAgent — a §11.4.122 event needing an operator *yes* before, not a report after. It stays `vendored`-tier and **unactivated** (which R-19 requires anyway) |

---

## 4. Sequencing rationale

Ten Track-A phases plus three Track-B phases, running concurrently and joining at P7.

The ordering is forced by measurement in four places, and each is easy to get wrong:

1. **P2 (shadowing benchmark) precedes every activation policy.** The published effect
   is established; its magnitude *on this corpus* is not. Setting a ceiling before the
   curve exists would be exactly the §11.4.6 guess this plan exists to avoid — so the
   benchmark is a **deliverable**, not an input. This is the plan's most
   schedule-surprising decision and the one the evidence forces most directly.
2. **P1 (hygiene) precedes P4 (loader).** The loader is fail-closed by design, so it
   will error immediately against today's corpus — `multitrack` has a `register.sh` and
   no `SKILL.md`. Normalising first means P4's first red is a real defect, not a known one.
3. **P3 (upstream extension) precedes P4.** R-01's mitigation is explicit: *"an
   incorporation phase MUST NOT be scheduled before Phase C exists."*
4. **P5 and P6 run in parallel, never sequential-copy.** Parity proven by two
   independent integrations is evidence; parity proven by copying one integration twice
   is a tautology.

Track B starts immediately because **no phase may declare a layer dependency before the
real layer state is measured** — and because B3 (deleting L5) is a design correction
that blocks nothing and unblocks the honest statement of what the Bridge is.

---

## 5. Risk coverage

**All 29 register rows are dispositioned; none is left as a note.** 26 open rows map to
task groups with a named gate and a paired §1.1 mutation; 3 were already resolved in the
register and are carried as design inputs (R-15 → pure-data skills; R-16 → R-23;
R-17 → R-25). The full table is `plan.md` §7.

**Three risks were found during planning** and are new to the register:

| ID | Risk | Sev | Mitigation | Task |
|---|---|---|---|---|
| **R-29** | Two independent installers provisioning into one `.claude/skills/` tree. Claude Code **neither errors nor warns** on a skill-name collision — it resolves silently — and the upstream namespacing request was **closed as not planned**. The collision happens in the *filesystem*, before Claude Code reads the directory, so no upstream mechanism observes it | HIGH | Install-time collision detection shipped by this feature: pre-write enumeration, abort-never-overwrite, reserved prefixes, manifest-driven regeneration. **§11.4.8: `NO external solution found — original work`** | T-P1.04 |
| **R-30** | Scope-limited probes yield *confident* false negatives (the L6 case, §2.2) | MED | Every presence probe enumerates all applicable scopes and records which it checked — encoded in the probe, not remembered | T-B1.01 |
| **R-31** | The work item's own environment facts are stale (§2.3) | MED | Re-derive environment facts from the artifact at phase start; record contradictions rather than reconciling them silently | T-P0.01 |

The risk register itself (`06`) is **not** edited by this document — it is a Phase-1b
artefact at Revision 2, and amending a sibling agent's committed deliverable in a shared
tree is not this task's remit. R-29/R-30/R-31 live in `plan.md` §7 and are proposed for
a future Revision 3.

---

## 6. What was written vs planned

**Planned only** (no production code touched): everything in `specs/`.

**Written**: `.specify/memory/constitution.md` — replaced SpecKit's stock 2 346-byte
placeholder with a CONST-059 inheritance pointer to `constitution/Constitution.md`.

Rationale for acting rather than scheduling: the file is a **governance pointer, not
production code**, and the placeholder was **live** — read by every `/speckit-plan`
Constitution Check run, including this feature's own. Leaving it would have allowed a
generic template to shadow the real constitution during the very workflow this
specification mandates. The unmodified SpecKit template remains untouched at
`.specify/templates/constitution-template.md`, and the gate that makes the fix durable
(`CM-CANONICAL-ROOT-CLARITY` extension) is scheduled as T-P1.06.4 rather than claimed.

No other file outside `specs/` and this research tree was modified. No submodule pointer
was touched. No branch was created. Nothing was pushed. `docs/Issues.md`,
`docs/Fixed.md` and the summaries were **not** regenerated, per the coordinator's
standing deferral.

---

## 7. Honest boundary

**Verified**: every factual claim traces to a cited section of the Phase-1 research tree
or to a probe captured in this session. Two runtime observables were verified directly
so the signature registry names real commands rather than plausible ones:
`helix_code/cmd/cli/main.go:3306` (the `skills` subcommand dispatcher) and
`submodules/helix_agent/internal/router/router.go:1363-1368`
(`GET /v1/skills`, `/categories`, `/:category`, `POST /match`).

**Not verified — nothing was built, run or wired.** This is a planning phase. Under
§11.4.108 every claim here is SOURCE-layer at best; no runtime signature in the registry
has been observed, no gate exists, and no mitigation has been proven. None of them is
"done" until verified on a clean target.

**Not read**: the six Bridge documents Phase 1 also left unread (NANO_TASK_ENGINE,
EXTENSION_DEVELOPMENT, TDD_INTEGRATION, CONSTITUTION_INTEGRATION, SECURITY, APPENDIX).
Obligations stated only there are not reflected in FR-044…FR-050 — task T-B2.04 closes
this and **must complete before the installer is considered final** (U-7).

**Nine `UNCONFIRMED:` items** are carried in `spec.md` §12, each with what would settle
it. Two were resolved during this phase (U-1 → L6 is WIRED; U-2 → no longer load-bearing
because L5 is deleted rather than substituted). Seven remain open: the local shadowing
magnitude (**U-3 — blocking; no threshold may be chosen before it**), the GitLab group
enumeration (U-4), HelixAgent's unsurveyed `Toolkit/` and `MCP/` trees (U-5), CONST-040
capability sourcing (U-6), the unread Bridge documents (U-7), the four stale upstream
branch diffs (U-8), and the upstream module's true maturity (U-9).

**Where the plan is weakest**: P2's cost is genuinely unestimable before its task set
exists; U-5 could reshape P6's attachment point; and assumption A-2 (upstream is
writable by this team) is an assumption, not a verified fact — if it fails, P3 becomes a
mirror and the plan's shape changes at its root.

---

## Sources

Phase-1 tree, this directory: `01_speckit_init.md` · `02_upstream_feature_inventory.md` ·
`03_consumer_integration_surfaces.md` · `03b_extension_integration.md` ·
`04_deep_research.md` · `05_catalogue_survey.md` · `06_risk_register.md`.

Local verification, 2026-07-29: `sqlite3 docs/workable_items.db` (HXC-159 description) ·
`.specify/{templates,memory,scripts}` inspection · `helix_code/cmd/cli/main.go` ·
`submodules/helix_agent/internal/router/router.go`.

Bridge layer-health probe, 2026-07-29 (sibling stream): per-layer WIRED/PARTIAL/
DESIGNED-ONLY/NONEXISTENT verdicts, supply-chain measurements for SuperSpec / SuperB /
`speckit-community` / `erophames/superpowers-mcp`, and the spec-kit catalogue-scope
finding.
