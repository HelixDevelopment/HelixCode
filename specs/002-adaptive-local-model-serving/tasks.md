---
feature: Adaptive Local Model Serving
branch: 002-adaptive-local-model-serving
date: 2026-09-02
---

# Tasks: Adaptive Local Model Serving

**Spec**: [spec.md](./spec.md) · **Plan**: [plan.md](./plan.md) · **Data model**: [data-model.md](./data-model.md) · **Contracts**: [contracts/](./contracts/) · **Validation**: [quickstart.md](./quickstart.md)

## Task Format

```
[ID] [markers] [Story] Description with file path
```

**Markers**: `[P]` parallelizable · `[TDD]` RED-GREEN-REFACTOR · `[REVIEW]` review gate · `[SUBAGENT]` delegable

**TDD is not optional here.** §11.4.224 mandates test-first for all work, and §11.4.115 requires the
RED to reproduce on the *pre-change* artifact with a polarity switch so the same test source becomes
the standing regression guard. Nearly every implementation task is therefore `[TDD]`.

## Path Conventions

Paths are repository-root-relative. The inner Go module is `submodules/helix_llm/`
(module `github.com/HelixDevelopment/HelixLLM`), propagation is `submodules/helix_agent/`, and the
consumer exporter is `claude_toolkit/scripts/`.

---

## Phase 1: Setup (Shared Infrastructure)

- [ ] T001 Create package skeletons `capability/`, `catalogue/`, `selection/`, `runtime/` under `submodules/helix_llm/internal/` with doc.go stating each package's single responsibility
- [ ] T002 [P] Add fixture-host test helpers in `submodules/helix_llm/internal/capability/testdata/` covering: no-accelerator, single-accelerator, dual-accelerator, low-storage, and unmeasurable hosts
- [ ] T003 [P] Add catalogue test fixtures in `submodules/helix_llm/internal/catalogue/testdata/` covering: commercial-safe, non-commercial, streaming-roster-member, and streaming-ineligible-but-MoE entries
- [ ] T004 Verify baseline is green before any change: `cd submodules/helix_llm && make test-unit`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Blocks every user story.** Nothing below Phase 2 can be built until measurement and the catalogue
exist, because selection is defined as a pure function of the two.

### Host Capability Profile

- [ ] T005 [TDD] Define `HostCapabilityProfile` in `submodules/helix_llm/internal/capability/profile.go` per data-model.md, including `measurement_complete` and separate `memory_available` / `storage_available`
- [ ] T006 [TDD] Implement CPU and system-memory measurement in `submodules/helix_llm/internal/capability/measure_system.go`
- [ ] T007 [TDD] Implement free-storage measurement in `submodules/helix_llm/internal/capability/measure_storage.go` — a distinct axis from memory (research.md D2)
- [ ] T008 [TDD] [REVIEW] Implement accelerator detection in `submodules/helix_llm/internal/capability/measure_accelerator.go` binding by **stable device identity, never an enumeration index** (§11.4.111). RED must include a two-device fixture whose enumeration order is reversed; the binding must still resolve to the same physical device
- [ ] T009 [TDD] Implement the zero-accelerator path in `submodules/helix_llm/internal/capability/measure_accelerator.go` as a valid first-class state, distinct from "accelerator state unknown"
- [ ] T010 [TDD] Implement measurement-failure handling in `submodules/helix_llm/internal/capability/measure.go`: partial or failed measurement sets `measurement_complete=false` and MUST NOT be silently completed with defaults (FR-056)
- [ ] T011 [P] [TDD] [SUBAGENT] Add macOS measurement paths in `submodules/helix_llm/internal/capability/measure_darwin.go` with per-OS dispatch against T006-T009 (§11.4.81 cross-platform parity); honest SKIP-with-reason where a platform cannot supply a figure

### Catalogue

- [ ] T012 [TDD] Define the catalogue entry type in `submodules/helix_llm/internal/catalogue/entry.go` per data-model.md § Model Option, including `usage_terms`, `storage_required`, `streaming_eligible`, and `integrity_expectation`
- [ ] T013 [TDD] Implement catalogue loading in `submodules/helix_llm/internal/catalogue/load.go` — roster membership is **data**, so adding a streaming-supported family is a data change not a code change (research.md D1)
- [ ] T014 [P] [TDD] [SUBAGENT] Populate text-LLM catalogue entries in `submodules/helix_llm/internal/catalogue/data/text.yaml` from `research/01-text-llm.md`, carrying every `UNVERIFIED:` marker forward rather than resolving it silently
- [ ] T015 [P] [TDD] [SUBAGENT] Populate vision + image-generation entries in `submodules/helix_llm/internal/catalogue/data/vision_image.yaml` from `research/02-vision-image.md`
- [ ] T016 [P] [TDD] [SUBAGENT] Populate speech + audio entries in `submodules/helix_llm/internal/catalogue/data/speech_audio.yaml` from `research/03-speech-audio.md`, including the non-commercial terms that make several strong candidates unofferable for commercial use
- [ ] T017 [TDD] Implement integrity verification in `submodules/helix_llm/internal/catalogue/integrity.go`: no weight file loads unverified, no source outside the allowlist (SC-011)

**Checkpoint**: measurement and catalogue exist and are independently tested. `make test-unit` green.

---

## Phase 3: User Story 1 — Get a model that actually runs well on this machine (P1) — MVP

**Goal**: The user asks for a local model and receives options that genuinely run well on *their*
machine, or an honest refusal naming what is missing.

**Independent test**: On a fixture no-accelerator host, every processor-servable family returns a
non-empty option set; generative image/audio return a stated reason; no offered option requires an
accelerator. Runnable with zero hardware (quickstart Scenario 1).

### Tests for User Story 1

- [ ] T018 [P] [TDD] [US1] Write the selection-contract test suite in `submodules/helix_llm/internal/selection/selection_test.go` from [contracts/selection.md](./contracts/selection.md) — RED before any implementation
- [ ] T019 [P] [TDD] [US1] Write refusal-path tests in `submodules/helix_llm/internal/selection/refusal_test.go`: one per reason — `insufficient_resources`, `unsupported_configuration`, `excluded_by_usage_terms` — asserting they are **distinct**, not one generic unavailability (FR-055)
- [ ] T020 [P] [TDD] [US1] Write the static-selection regression guard in `submodules/helix_llm/internal/selection/no_static_selection_test.go`: with a model-naming environment variable set, selection still measures and does not honour it (FR-056). RED must reproduce on the current artifact, where it *is* honoured

### Implementation for User Story 1

- [ ] T021 [TDD] [US1] Implement the option set type and `withheld` reason enum in `submodules/helix_llm/internal/selection/option.go`
- [ ] T022 [TDD] [US1] Implement the core join in `submodules/helix_llm/internal/selection/select.go` as a pure function of (profile, catalogue, declared usage) — one-directional, never writing back into measurement
- [ ] T023 [TDD] [US1] Implement dual-axis fit checking in `submodules/helix_llm/internal/selection/fit.go`: memory **and** storage evaluated separately (D2)
- [ ] T024 [TDD] [US1] Implement per-family non-empty guarantee in `submodules/helix_llm/internal/selection/family.go`: a family that cannot be served returns a stated reason naming the missing requirement, never an empty list and never a below-threshold offer (D5)
- [ ] T025 [TDD] [US1] Implement usage-terms filtering in `submodules/helix_llm/internal/selection/terms.go` (FR-054): declared commercial use withholds every non-commercial model with its restricting term named
- [ ] T026 [TDD] [US1] Implement pin handling in `submodules/helix_llm/internal/selection/pin.go`: a pin constrains selection and is refused with the insufficient resource named — never a measurement bypass (FR-056)
- [ ] T027 [TDD] [US1] Compose refusal and description text from measurement + catalogue data in `submodules/helix_llm/internal/selection/describe.go` — no fixed English strings (CONST-046)
- [ ] T028 [TDD] [US1] Implement runtime choice in `submodules/helix_llm/internal/runtime/choose.go`: in-memory first, streaming only when it does not fit **and** is roster-supported **and** meets that runtime's minimums (D6)
- [ ] T029 [TDD] [US1] Wire selection into the existing `vrambroker` lease flow in `submodules/helix_llm/internal/runtime/lease.go` — extend the existing arbitration, do not replace it (§11.4.74)
- [ ] T030 [TDD] [US1] Migrate `submodules/helix_llm/cmd/visiongen-boot/` off `VISIONGEN_MODEL_GGUF` / `VISIONGEN_MODEL_DIR` selection onto measured selection; the variables may still say where files live, never which model runs
- [ ] T031 [TDD] [US1] Migrate `submodules/helix_llm/cmd/imagegen-boot/` off static selection onto measured selection
- [ ] T032 [REVIEW] [US1] Independent review of the US1 diff on the Fable substrate at xhigh effort (§11.4.209, §11.4.125/§11.4.142), iterated to a zero-finding GO (§11.4.134). Must specifically catch: index-based device binding, any path where configuration names the model, and any offer whose terms exclude the declared usage

**Checkpoint**: quickstart Scenarios 1-4 pass with no hardware. MVP is deliverable here.

---

## Phase 4: User Story 2 — Use those local models from the tools I already work in (P2)

**Goal**: Selected models appear in the user's daily tools under recognisable names.

**Independent test**: A served option appears in the model listing with its
`helixllm/<host>/<model>[:<variant>]` identity in the value field and a derived, charset-safe
identifier the consumer accepts unmodified.

### Tests for User Story 2

- [ ] T033 [P] [TDD] [US2] Write the model-listing contract tests in `submodules/helix_llm/internal/brain/models_test.go` from [contracts/model-listing.md](./contracts/model-listing.md)
- [ ] T034 [P] [TDD] [US2] Write the identifier-derivation test suite in `submodules/helix_llm/internal/naming/derive_test.go`, asserting output satisfies **both** consumer validators as they stand: `^[a-zA-Z][a-zA-Z0-9_-]*$` and `[A-Za-z0-9._-]` only
- [ ] T035 [P] [TDD] [US2] Write the guard-integrity test in `claude_toolkit/tests/provider_validation_test.sh` asserting the provider-id charset check is **unchanged** — this test exists so a future "just allow `/`" change fails loudly (FR-014a)

### Implementation for User Story 2

- [ ] T036 [TDD] [US2] Implement identity construction in `submodules/helix_llm/internal/naming/identity.go` producing `helixllm/<host>/<model>[:<variant>]` as a value
- [ ] T037 [TDD] [US2] Implement per-consumer identifier derivation in `submodules/helix_llm/internal/naming/derive.go`, recording the `identity ↔ identifier` mapping so the two cannot drift
- [ ] T038 [TDD] [US2] Correct `Brain.Models()` in `submodules/helix_llm/internal/brain/brain.go` to publish both the derived identifier and the host-prefixed identity — the single emission point reaching all consumers (D7)
- [ ] T039 [TDD] [US2] Ensure unavailable options carry their withheld reason through the listing in `submodules/helix_llm/internal/brain/brain.go`; a model not actually being served is never listed as available
- [ ] T040 [TDD] [US2] Propagate the option set through HelixAgent in `submodules/helix_agent/` to upper layers and final consumers
- [ ] T041 [TDD] [US2] Extend the Claude Toolkit detector in `claude_toolkit/scripts/claude-providers.sh` from one-model-per-host to **one entry per model per host** (D8) — extension, not replacement
- [ ] T042 [TDD] [US2] Make toolkit export idempotent in `claude_toolkit/scripts/claude-providers.sh`: re-running updates rather than duplicates
- [ ] T043 [TDD] [US2] Implement HelixCode consumer export in `submodules/helix_llm/internal/naming/export_helixcode.go` honouring HelixCode's own identifier rules
- [ ] T044 [TDD] [US2] [SUBAGENT] Design and implement OpenCode provider export — **net-new**: the toolkit has no OpenCode provider-config integration today, only Skills/MCP/instructions sync (D8)
- [ ] T045 [REVIEW] [US2] Independent review of the US2 diff, with explicit attention to whether any consumer validation was relaxed to fit a name (FR-014a)

**Checkpoint**: quickstart Scenario 6 passes end-to-end with a real served response.

---

## Phase 5: User Story 3 — Find capacity beyond this machine (P3)

**Goal**: Reach serving instances on the local network and remotely.

**Independent test**: An instance that cannot present the pre-shared secret is never trusted as a
model source and receives no request content.

- [ ] T046 [P] [TDD] [US3] Write discovery + trust tests in `submodules/helix_llm/internal/discovery/discovery_test.go`, including the negative case: unauthenticated instance receives no request content
- [ ] T047 [TDD] [US3] Implement instance discovery in `submodules/helix_llm/internal/discovery/discover.go` across local-host / local-network / remote reachability
- [ ] T048 [TDD] [US3] [REVIEW] Implement pre-shared-secret authentication in `submodules/helix_llm/internal/discovery/trust.go` (FR-024, FR-025) — the feature's main security surface. Secret from environment or `.env`, never logged, never committed (§11.4.10)
- [ ] T049 [TDD] [US3] Implement health tracking in `submodules/helix_llm/internal/discovery/health.go` so an unreachable instance is not exported as available
- [ ] T050 [TDD] [US3] Implement multi-host placement in `submodules/helix_llm/internal/selection/placement.go` (FR-040..FR-043) — each option's host recorded in the selection
- [ ] T051 [TDD] [US3] Ensure the discovery secret never reaches an exported consumer configuration in `claude_toolkit/scripts/claude-providers.sh`
- [ ] T052 [REVIEW] [US3] Independent security-focused review of the US3 diff — credential handling, trust boundary, and what an untrusted instance can observe

---

## Phase 6: User Story 4 — Run a model this machine could not otherwise hold (P4)

**Goal**: Incorporate the disk-streaming runtime so oversized models become feasible.

**Independent test**: A roster-supported model too large for memory is offered on the streaming path
with its speed trade-off labelled; an architecturally-similar but roster-absent model is not.

- [ ] T053 [P] [TDD] [US4] Write streaming-eligibility tests in `submodules/helix_llm/internal/runtime/streaming_test.go`, including the negative case that a roster-absent MoE model is **not** offered (D1)
- [ ] T054 [REVIEW] [US4] **Operator checkpoint** — confirm adopting the streaming runtime as a dependency in the serving path before integration proceeds (plan.md § Human Checkpoints)
- [ ] T055 [TDD] [US4] Implement roster-membership eligibility in `submodules/helix_llm/internal/runtime/streaming.go` as a catalogue lookup, never an architecture predicate
- [ ] T056 [TDD] [US4] Implement streaming-runtime process lifecycle in `submodules/helix_llm/internal/runtime/streaming_launch.go` under the same lease and health discipline as the in-memory path
- [ ] T057 [TDD] [US4] Enforce the runtime's own per-model memory **and** disk minimums in `submodules/helix_llm/internal/runtime/streaming.go`
- [ ] T058 [TDD] [US4] Label the speed trade-off on every streaming option in `submodules/helix_llm/internal/selection/describe.go`
- [ ] T059 [TDD] [US4] Ensure "unsupported configuration" and "insufficient resources" remain distinct reasons on the streaming path in `submodules/helix_llm/internal/runtime/streaming.go` (FR-055, D6)

---

## Phase 7: User Story 5 — Capability beyond text (P5)

**Goal**: The same measured selection governs every capability family, not only text.

**Independent test**: Each family's options are selected from measurement on the same code path as
text, with no family-specific static selection remaining.

- [ ] T060 [P] [TDD] [US5] Bring vision under measured selection in `submodules/helix_llm/internal/selection/` using the existing measured on-GPU boot path as the reference implementation
- [ ] T061 [P] [TDD] [US5] Bring image generation under measured selection in `submodules/helix_llm/services/imagegen/imagegen_server.py` + `submodules/helix_llm/cmd/imagegen-boot/`; the existing generation path has never been run on hardware — this task includes its first real run with captured evidence
- [ ] T062 [P] [TDD] [US5] [SUBAGENT] Wire the **existing** `submodules/helix_llm/container/whisper_stt_server.py` into the catalogue and measured selection — speech-to-text is not greenfield, only unwired
- [ ] T063 [P] [TDD] [US5] [SUBAGENT] Implement text-to-speech serving in `submodules/helix_llm/services/tts/` — genuinely new; prefer commercially-safe licensed models as the default per T016's catalogue
- [ ] T064 [P] [TDD] [US5] [SUBAGENT] Implement audio classification serving in `submodules/helix_llm/services/audio/` (processor-viable on every host tier)
- [ ] T065 [TDD] [US5] Mark audio **generation** accelerator-required in `submodules/helix_llm/internal/catalogue/data/speech_audio.yaml` — it has no processor-viable option, so no-GPU hosts receive a stated reason, not an offer (D5)
- [ ] T066 [P] [TDD] [US5] Bring vector graphics and embeddings under measured selection via `submodules/helix_llm/services/vectorize/`
- [ ] T067 [REVIEW] [US5] Independent review of the US5 diff — specifically that no family retained a private static-selection path

---

## Phase 8: Polish & Cross-Cutting Concerns

- [ ] T068 [P] [SUBAGENT] Write the user guide at `submodules/helix_llm/docs/guides/adaptive_model_serving.md` covering selection, refusals, pins, and usage terms
- [ ] T069 [P] [SUBAGENT] Write the FAQ at `submodules/helix_llm/docs/guides/adaptive_model_serving_faq.md` — lead with "why was this model not offered to me?", the question every refusal reason will generate
- [ ] T070 [P] [SUBAGENT] Produce architecture and selection-flow diagrams in `submodules/helix_llm/docs/diagrams/` (mermaid), including the one-directional measurement→option join
- [ ] T071 [P] [SUBAGENT] Document consumer setup per tool in `submodules/helix_llm/docs/guides/consumer_setup.md` for Claude Toolkit, HelixCode and OpenCode
- [ ] T072 [P] Run and record the full mandated test-type sweep (§11.4.169): `cd submodules/helix_llm && make test-all`, plus `test-stress test-chaos test-race test-benchmark`
- [ ] T073 [P] Capture runtime evidence per §11.4.5/§11.4.69 under `docs/qa/<run-id>/` — a real measurement and a real served response per family, never a configuration read
- [ ] T074 [P] Register each closed defect's regression guard into the standing suite (§11.4.135) with its RED preserved at `RED_MODE=1`
- [ ] T075 Verify cross-platform parity (§11.4.81) across `submodules/helix_llm/internal/capability/` and `internal/runtime/`: every platform-dependent path has a per-OS branch or an honest documented gap
- [ ] T076 [REVIEW] Investigate the dead `helix-llm`/`helix-debate` provider block at `helix_code/config/config.yaml:155-166` via git history (§11.4.124), then **ask the operator** whether to wire or remove — its own commit, never bundled (§11.4.122)
- [ ] T077 [REVIEW] Final independent review across all phases, iterated to a zero-finding GO (§11.4.134)

---

## Dependencies & Execution Order

### Phase Dependencies

```text
Phase 1 (Setup)
   └─> Phase 2 (Foundational: measurement + catalogue)   ← BLOCKS everything
          └─> Phase 3 (US1, P1)  ← MVP, deliverable alone
                 ├─> Phase 4 (US2, P2)   needs US1's option set
                 │      └─> Phase 5 (US3, P3)   needs US1+US2
                 ├─> Phase 6 (US4, P4)   needs US1's selection machinery
                 └─> Phase 7 (US5, P5)   needs US1's selection machinery
                        └─> Phase 8 (Polish)
```

US3 depends on US2; US4 and US5 depend only on US1 and are independent of each other and of US2.

### Within Each User Story

Tests (RED) → types → core logic → integration → review gate.

### Parallel Opportunities

| Where | Tasks | Why they don't contend |
|---|---|---|
| Catalogue population | T014, T015, T016 | Separate data files, separate research sources |
| Measurement platforms | T011 alongside T006-T009 | Per-OS dispatch, separate files |
| US1 test authoring | T018, T019, T020 | Separate test files, all RED before implementation |
| US2 test authoring | T033, T034, T035 | Separate files, two repos |
| Capability families | T060-T064, T066 | One family each, separate services |
| Documentation | T068-T071 | Separate documents |
| After Phase 3 | Phases 4, 6, 7 | Three independent streams off the same MVP |

Hardware-touching validation is **single-owner per device** (§11.4.119) — parallel streams may probe
read-only, but only one drives a given accelerator at a time, or the evidence is cross-contaminated.

---

## Implementation Strategy

**MVP = Phase 1 + Phase 2 + Phase 3 (US1).** That increment delivers the feature's actual point: a
user asks for a local model and gets options that genuinely run on their machine, or an honest
refusal. Everything after it is distribution (US2, US3) or breadth (US4, US5).

Deliver incrementally, stopping at each checkpoint. The first three phases need **no hardware** to
validate — selection is a pure function, so its whole surface including every refusal path is
reachable from fixture hosts. Hardware enters at Scenario 5 and above.

## Notes

- Every `[TDD]` task RED-reproduces on the pre-change artifact (§11.4.115), with the polarity switch
  turning that same test into the standing regression guard.
- `[REVIEW]` gates are Fable at xhigh effort (§11.4.209), iterated to zero findings **and zero
  warnings** (§11.4.134).
- Commit per task or logical group; stage deliberately, never `git add -A` (§11.4.30).
- A refusal is a first-class outcome. Most of the US1 test surface asserts the system saying *no*
  correctly — a selection layer that cannot refuse is not selecting.
