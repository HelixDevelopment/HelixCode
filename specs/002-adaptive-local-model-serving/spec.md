# Feature Specification: Adaptive Local Model Serving

**Feature Branch**: `002-adaptive-local-model-serving`
**Created**: 2026-09-02
**Status**: Draft
**Input**: User description: "Extend HelixLLM (helix_llm) so our LLM sub-system can dynamically bring up proper local model based on fully checking hosts hardware and system resources capabilities (host on which models will be running). Exhaustive research about all possible types of models we could run and then create variety of options for particular host — LLMs, Vision, Audio generating or recognition, Text to speech, Speech to text, Generative — Images, Design and vector graphics, and all others. All model configuration we could pick MUST BE passed through HelixAgent further to upper layers and final consumers of the System. Create proper configurations and make them fully available for all combinations users can pick. Make sure we can access them fully through all possible providers -> models configurations added (exportable fully) to OpenCode, HelixCode, Claude Code (the Claude Toolkit providers aliases). Extend Claude Toolkit to recognize HelixAgent as provider and all HelixLLM models and variations (proper namings so users can recognize HelixLLM model configs and models running locally). Claude Toolkit MUST BE able to recognize HelixAgent and HelixLLM on current host, in local network (all running instances) and from the cloud (proper configuration ways using .env or environment variables). Chosen models to run locally MUST be properly selected so no performance glitches, gaps or bottlenecks happen. Run them using llama.cpp and colibri. Colibri is not yet supported, however it MUST BE fully incorporated and decisions on which one to use MUST BE fully dynamic and based on environment we have on current host. Everything MUST BE exhaustively planned with multiple rounds of full systematic deep web research, incorporating new cutting edge opensource codebase solutions, all fully documented, all documentation extended, exhaustive user guides, manuals and FAQs for everything created and everything covered with full schemes, diagrams and graphs."

## User Scenarios & Testing *(mandatory)*

<!--
  SCOPE NOTE (resolved 2026-09-02): all five stories are IN the first release. The P1..P5 priorities
  below are BUILD ORDER within that release — the sequence in which slices become independently
  demonstrable — not a scope-cutting ladder. Per FR-031..029 none of them may ship partially.
-->

### User Story 1 - Get a model that actually runs well on this machine (Priority: P1)

A developer sits down at an unfamiliar machine and wants local AI capability. They do not know what
model sizes, quantisations or memory footprints their hardware can carry. They ask the system for a
local model, and it inspects the machine it will actually serve from, then offers a short list of
options it can confidently run — each labelled with what the user gets and what it costs in
resources. They pick one, it starts, and it performs without stalling the machine.

**Why this priority**: This is the whole point of the feature and the only story that delivers value
on its own. Without it, nothing else has anything to expose, export, or discover. Every other story
is a distribution or breadth concern layered on top of this one.

**Independent Test**: On a machine with no prior configuration, request a local model and confirm the
system returns options sized to that machine, starts the chosen one, and serves a request — with no
model offered that the machine cannot actually hold.

**Acceptance Scenarios**:

1. **Given** a machine with limited memory and no discrete GPU, **When** the user asks for local
   model options, **Then** only options that fit within that machine's measured headroom are offered,
   and each states its resource cost in terms the user can compare.
2. **Given** a machine with NO accelerator but ample system memory, **When** the user asks for
   options, **Then** a real, non-empty set of CPU-served options is offered — the absence of a GPU
   reduces what is offered but never reduces it to nothing.
3. **Given** a machine with a large discrete GPU, **When** the user asks for the same thing, **Then**
   materially more capable options appear, sized to that accelerator's usable memory.
4. **Given** a user selects an offered option, **When** the model starts, **Then** it serves requests
   and the host remains responsive — no memory exhaustion and no swap thrashing.
5. **Given** hardware capacity changes (another workload claims the GPU), **When** the user next asks
   for options, **Then** the offered set reflects the currently available headroom, not the headroom
   measured at install time.

---

### User Story 2 - Use those local models from the tools I already work in (Priority: P2)

A developer already works in Claude Code, OpenCode or HelixCode. They want the local models to appear
there as selectable providers, under names that make it obvious which are local HelixLLM models and
which are remote. They configure it once and their existing tool picks the models up.

**Why this priority**: A local model nobody can reach from their daily tool delivers little. This is
what converts US1 from a capability into something people use. It depends on US1 but is otherwise
independent.

**Independent Test**: With a model running from US1, generate the provider configuration, apply it to
one supported tool, and confirm the model is selectable and answers a prompt from inside that tool.

**Acceptance Scenarios**:

1. **Given** local models are running, **When** the user requests provider configuration, **Then**
   they receive configuration for each supported tool that works without hand-editing.
2. **Given** that configuration is applied, **When** the user lists available models in their tool,
   **Then** HelixLLM-served models appear under names that distinguish them from cloud providers and
   identify which host serves them.
3. **Given** a model is stopped, **When** the user's tool next lists models, **Then** the unavailable
   model is not presented as usable.

---

### User Story 3 - Find capacity beyond this machine (Priority: P3)

A developer's laptop cannot run the model they need, but a workstation on the same network can. They
want their tool to discover HelixLLM instances reachable to them — on this host, elsewhere on the
local network, or hosted remotely — and use those models as if local.

**Why this priority**: Extends reach substantially, but only has value once US1 and US2 work. Carries
the feature's main security surface (FR-018, FR-019 — pre-shared-secret authentication), so it is
built after the basics are proven, and it also carries the multi-host placement in FR-027..FR-030.

**Independent Test**: With an instance running on a second machine, confirm a client on the first
machine discovers it, lists its models distinctly from local ones, and can send a request to it.

**Acceptance Scenarios**:

1. **Given** an instance is running elsewhere on the network, **When** the user's tool performs
   discovery, **Then** that instance's models are offered and clearly labelled with their host.
2. **Given** discovery is not wanted, **When** the user disables it by configuration, **Then** no
   discovery traffic is emitted and only explicitly configured endpoints are used.
3. **Given** a discovered instance becomes unreachable, **When** the user next lists models, **Then**
   its models are marked unavailable rather than failing silently at request time.

---

### User Story 4 - Run a model this machine could not otherwise hold (Priority: P4)

A user wants a very large mixture-of-experts model that cannot fit in their machine's memory at all.
Rather than being told "not possible", the system offers it on a slower footing — streaming the parts
it needs from disk — and states plainly that the trade is capability for speed.

**Why this priority**: Unlocks a genuinely new capability, but serves a narrower case than US1-US3 and
depends on the selection machinery from US1 already working.

**Independent Test**: On a machine that cannot hold a given large model in memory, confirm the system
still offers it via the streaming path, labels the speed trade-off, and serves a request.

**Acceptance Scenarios**:

1. **Given** a model too large for available memory but of a supported architecture, **When** the user
   asks for options, **Then** it is offered as a distinct slower-but-possible choice, not silently
   omitted and not offered as if it were fast.
2. **Given** a model that fits comfortably in memory, **When** options are produced, **Then** the
   in-memory path is preferred and the streaming path is not proposed for it.
3. **Given** a model too large AND of an architecture the streaming path cannot serve, **When** the
   user asks, **Then** the system says so explicitly rather than offering something that will fail.

---

### User Story 5 - Capability beyond text (Priority: P5)

A user wants more than chat: transcribing audio, generating speech, describing images, or generating
images. They want the same hardware-aware selection and the same tool integration for those.

**Why this priority**: Broadens the feature considerably, and each family multiplies research,
packaging and testing. It is built last because it reuses the selection, export and discovery
machinery from the earlier stories — but it is IN the first release, not deferred, and must be
complete for every family before release (FR-031..029).

**Independent Test**: For one non-text family, confirm hardware-aware options are offered, one starts,
and it produces correct output for a known input.

**Acceptance Scenarios**:

1. **Given** a machine meets a non-text family's requirements, **When** the user asks for options in
   that family, **Then** suitable options are offered with their resource cost.
2. **Given** a machine cannot support any option in a family, **When** the user asks, **Then** the
   system says the family is unavailable on this host and why, rather than offering an unusable option.

---

### Edge Cases

- A model is selected, then another process claims the memory before it finishes loading. The system
  must fail clearly and leave the host healthy rather than triggering an out-of-memory kill.
- Hardware reports capacity the machine cannot actually deliver (shared or virtualised GPUs,
  containers with limits below host totals). Measured headroom must reflect what is genuinely usable.
- Two model families are requested at once and each fits alone but not together.
- A discovered remote instance advertises models it can no longer serve.
- The machine has capable hardware but no supported driver or runtime present.
- A user's configuration names a model that no longer exists in the catalogue.
- Discovery finds an instance that is reachable but of an incompatible version.
- Disk fills during the streaming path, mid-request.
- Thermal or power throttling degrades a host that profiled as capable when cold.

#### Brainstorm Prompts

- **Boundary conditions**: What is the smallest machine that gets any usable option? What is offered
  when a host is exactly at the threshold?
- **Error scenarios**: Runtime absent, driver mismatch, model download interrupted, remote instance
  disappears mid-session.
- **Scale**: Many concurrent users against one host; many discovered instances; a very large catalogue.
- **Security**: What can a hostile instance on the network claim to be? What does discovery leak about
  a host's capabilities? Can an exported configuration carry a secret?
- **User confusion**: Will users understand why a model is missing on one machine and present on
  another? Will they know which models are local versus remote?
- **Data integrity**: Concurrent starts competing for the same memory; stale advertised capability.
- **Backwards compatibility**: Existing configurations and already-running local inference must keep
  working unchanged.

## Open Questions

| # | Question | Status | Resolution |
|---|----------|--------|------------|
| Q1 | Which modality families are in scope for the first release? | **Resolved** 2026-09-02 | **All families in v1**, each complete end-to-end: text, vision, audio generation, audio recognition, text-to-speech, speech-to-text, image generation, and design/vector graphics. Combined with FR-031..029 this makes the first release large — nothing may ship advertised-but-unserved, so every family must be finished before release. See "Release size" in Assumptions. |
| Q2 | What trust model governs discovery of instances beyond the current host? | **Resolved** 2026-09-02 | **Pre-shared secret supplied via environment variables / `.env`.** Instances and clients authenticate with it; an instance that cannot present it is never trusted as a model source. See FR-018, FR-019. |
| Q3 | Does hardware-aware selection cover remote serving hosts, or only the local host? | **Resolved** 2026-09-02 | **Any host, including remote.** Selection profiles and places models across every reachable serving host, not only the machine the user is on. See FR-027..FR-030. |

## Requirements *(mandatory)*

### Functional Requirements

**Hardware-aware selection**

- **FR-001**: System MUST measure the capabilities and currently available resource headroom of the
  host that will actually serve the model, not the host that requested it.
- **FR-002**: System MUST always determine accelerator (GPU) presence as part of that measurement,
  and when one is present MUST determine its usable memory capacity — not its nominal capacity — as a
  first-class input to every selection decision. Selection MUST NOT proceed on system memory alone
  while an accelerator is present and usable.
- **FR-003**: System MUST treat "no usable accelerator" as a fully supported configuration, not a
  failure or a degraded fallback: on such a host it MUST select against available system memory and
  CPU capability and still offer every option that genuinely runs there. A host with no accelerator
  but sufficient system memory MUST receive real options, never an empty set.
- **FR-004**: System MUST offer only model options that the measured host can run within its
  available headroom, and MUST NOT offer options it cannot support.
- **FR-005**: System MUST express each option's resource cost and expected capability in terms a
  non-expert can compare, without requiring knowledge of model internals.
- **FR-006**: System MUST re-evaluate available headroom at selection time, so that offers reflect
  current conditions rather than a one-time measurement.
- **FR-007**: System MUST refuse a selection that would exhaust host resources, and MUST state why.
- **FR-008**: System MUST leave the host responsive while a selected model serves requests, per the
  thresholds in Success Criteria.

**Model catalogue and configuration**

- **FR-009**: System MUST maintain a catalogue of runnable model options recording, for each, its
  capability family, its resource requirements, and which execution paths can serve it.
- **FR-010**: System MUST allow users to select among offered combinations, including running more
  than one model where the host has headroom for all of them.
- **FR-011**: System MUST give every offered option a name that identifies it as HelixLLM-served,
  names the serving host, and distinguishes it from remote provider models.
- **FR-012**: System MUST make the active model configuration available to consuming layers, so that
  tools above it see the same set of models the serving layer actually has running.

**Consumer integration**

- **FR-013**: System MUST produce provider configuration for each supported consuming tool that works
  without hand-editing.
- **FR-014**: Users MUST be able to obtain that configuration on demand and apply it themselves;
  the system MUST NOT silently modify another tool's configuration files.
- **FR-015**: System MUST let a consuming tool distinguish available from unavailable models, so a
  stopped model is not presented as usable.
- **FR-016**: System MUST support configuration through environment variables and environment files
  for endpoint selection and credentials.

**Discovery**

- **FR-017**: System MUST support locating instances on the current host, elsewhere on the local
  network, and at explicitly configured remote endpoints.
- **FR-018**: Users MUST be able to disable each discovery mode independently; when disabled, the
  system MUST emit no discovery traffic for that mode.
- **FR-019**: System MUST label every discovered model with the host serving it.
- **FR-020**: System MUST NOT accept model definitions from an unverified network source without the
  trust check established by Q2.

**Execution paths**

- **FR-021**: System MUST select the execution path per model based on the measured host and the
  model's requirements, preferring the in-memory path whenever the model fits.
- **FR-022**: System MUST offer the disk-streaming path only for models it can actually serve that
  way, and MUST label the speed trade-off when doing so.
- **FR-023**: System MUST state plainly when a requested model cannot be served by any available path
  on that host, rather than offering an option that will fail.
- **FR-024**: System MUST continue to serve existing local inference unchanged for users who do not
  adopt the new selection flow.

**Documentation**

- **FR-025**: System MUST ship user-facing guidance covering selection, configuration, consumer
  integration, and discovery, including a troubleshooting section for the failure modes in Edge Cases.
- **FR-026**: Documentation MUST include diagrams of the selection flow, the path between serving
  layer and consuming tools, and the discovery topology.

**Multi-host selection and placement**

- **FR-027**: System MUST profile the capabilities and available headroom of every reachable,
  authenticated serving host — not only the host the user is working on.
- **FR-028**: System MUST be able to place a selected model on the serving host best able to run it,
  and MUST tell the user which host was chosen and why in terms they can act on.
- **FR-029**: System MUST account for capacity across the fleet, so that placing a model on a host
  reflects that host's currently free resources and does not exceed them.
- **FR-030**: System MUST degrade to local-only selection when no remote host is reachable or
  authenticated, offering whatever the local machine genuinely supports rather than failing.

**Completeness (no partial delivery)**

- **FR-031**: System MUST NOT present any capability, model option, family or provider entry as
  available unless it is fully implemented and actually served end-to-end. A listed-but-unserved
  entry is a defect of the same severity as a wrong answer, not a cosmetic gap.
- **FR-032**: Every capability included in the first release MUST be complete along its whole path —
  selectable, startable, reachable from the consuming tools it claims to support, and documented.
  A capability that works in the serving layer but is not reachable from the tools it is advertised
  in is NOT complete.
- **FR-033**: Where a capability is deliberately not in the first release, the system MUST omit it
  entirely rather than expose it in a partial or non-functional state. Absence is acceptable;
  advertised-but-broken is not.

### Key Entities

- **Host Capability Profile**: What a candidate serving machine can currently support — total and
  available memory, accelerator presence and capacity, storage headroom, and which execution paths
  are usable there. Measured, with a freshness point in time.
- **Model Option**: A runnable choice — its capability family, human-meaningful description, resource
  requirements, which execution paths can serve it, and expected performance characteristics.
- **Model Selection**: A user's chosen set of options, the host each runs on, and current running
  state. Bounds what the consuming layers are told is available.
- **Serving Instance**: A reachable provider of models — on this host, on the local network, or
  remote — with its identity, reachability state, trust status, and the models it currently serves.
- **Consumer Configuration**: The provider definition handed to a consuming tool: endpoint, model
  names, and credential references, in that tool's expected shape.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user with no prior configuration obtains a working local model on an unfamiliar
  machine in under 10 minutes, without needing to know model names, sizes, or quantisation formats.
- **SC-002**: Zero offered options fail to start for lack of resources on the host that offered them,
  across the supported hardware range.
- **SC-003**: While a selected model serves a typical request, the host retains at least 15% of its
  memory free and performs no sustained swapping.
- **SC-004**: A user integrates local models into their existing tool with a single configuration
  step and no manual editing.
- **SC-005**: In a listing of available models, a user can tell local from remote, and identify the
  serving host, without consulting documentation.
- **SC-006**: A model that cannot fit in memory but is otherwise supportable is still offered on at
  least one supported host class, with its speed trade-off stated before selection.
- **SC-007**: Disabling a discovery mode results in no discovery traffic for that mode, verifiable by
  observation.
- **SC-008**: Every failure mode listed in Edge Cases produces a message stating what happened and
  what the user can do, rather than a silent failure or a generic error.
- **SC-009**: Users who do not adopt the new flow see no change in existing local inference behaviour.
- **SC-011**: Every capability listed as available in the first release can be exercised end-to-end
  by a user — selected, started, and used from at least one supported consuming tool — with zero
  entries that appear in a list but cannot be used.
- **SC-010**: On a host with a usable accelerator, offered options reflect its usable memory; on an
  otherwise-identical host with the accelerator removed, options are offered against system memory
  and the set is non-empty wherever memory genuinely permits. Neither host is offered an option the
  other's hardware would be required to run.

## Assumptions

**Verified against the codebase (2026-09-02)** — this feature extends existing components rather than
building new ones, per the reuse-before-rewrite rule:

- HelixLLM already performs hardware profiling and memory budgeting, and already runs a control-plane
  prober and scheduler across multiple hosts. Selection extends these rather than replacing them.
- HelixLLM already serves local inference through the in-memory execution path across CUDA, Metal and
  ROCm, and already exposes OpenAI- and Anthropic-compatible interfaces that consuming tools can use.
- HelixLLM already has partial coverage of vision, image and embedding families. Audio, speech-to-text
  and text-to-speech have no current implementation and are genuinely new.
- The disk-streaming execution path has no current implementation and is genuinely new.

**Verified externally (2026-09-02)** — this materially shapes FR-021 through FR-023:

- The disk-streaming runtime under consideration is **not** a general-purpose replacement for the
  existing in-memory runtime. It is specialised: it streams mixture-of-experts weights from disk to
  run models that would not otherwise fit in memory, trading speed for feasibility, and it does not
  serve arbitrary models. The choice between paths is therefore **not** a symmetric preference
  between interchangeable engines — it is "in-memory when the model fits, streaming when it otherwise
  could not run at all." Any plan that treats the two as interchangeable is proceeding on a false
  premise. See Sources below.

**Release completeness** (operator direction, 2026-09-02):

- A live Workshop deployment currently shows capabilities that are listed but not served. That is the
  failure this feature must not repeat: for the first release, everything included must be complete
  and usable end-to-end, with nothing advertised in a partial state (FR-031 through FR-033).
- This raises the stakes on Q1 rather than removing it. "Everything complete" and "everything
  included" are different statements: the narrower the first release's scope line, the sooner it can
  be genuinely complete. Q1 sets that line.

**Release size** (scope resolved 2026-09-02):

- All modality families are in the first release, and FR-031..029 forbid shipping any of them
  partially. Stated plainly: this is a large first release. Audio generation, audio recognition,
  text-to-speech and speech-to-text have no implementation today, and design/vector graphics
  generation is a distinct problem from raster image generation. Each family carries its own model
  research, packaging, hardware profiling and test surface, and each must be complete and reachable
  from the consuming tools before release. Planning should size these per family rather than assume
  the text path's effort generalises.
- Selection spans the whole reachable fleet, not just the local machine, which adds capacity
  accounting and placement policy on top of per-host profiling.

**Scope and environment**:

- "No performance glitches or bottlenecks" is interpreted as SC-003's measurable headroom and
  no-sustained-swap thresholds. Absolute latency targets are hardware-dependent and are therefore not
  fixed here.
- Users are developers running on their own machines or on machines they administer. This feature
  assumes no multi-tenant isolation guarantees between users of one host.
- Consuming tools retain their own configuration formats; the system produces configuration for them
  and never writes into their files unasked.
- The exhaustive model research the request calls for is a **planning input**, not a specification
  deliverable. This document states what the system must do; which specific models populate the
  catalogue is determined during planning and will change over time as the field moves.

## Sources

Verified during specification, 2026-09-02:

- Colibri project repository — <https://github.com/JustVugg/colibri>
- Colibri capability analysis — <https://wavect.io/blog/colibri-glm-5-2-consumer-hardware/>
- Colibri model/memory characteristics — <https://pasqualepillitteri.it/en/news/7923/colibri-glm-5-2-744b-25gb-ram-en>

## Brainstorm Log

<!-- Maintained by /speckit-superspec-brainstorm — no sessions recorded yet. -->
