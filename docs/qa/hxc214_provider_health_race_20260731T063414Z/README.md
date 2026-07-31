# HXC-214 — provider `GetHealth` data race + live-pointer aliasing

| | |
|---|---|
| Revision | 1 |
| Created | 2026-07-31 |
| Last modified | 2026-07-31 |
| Status | active |
| Run ID | `hxc214_provider_health_race_20260731T063414Z` |
| Branch | `main` |
| Baseline HEAD | `58984d27` |

## Table of contents

- [1. The defect](#1-the-defect)
- [2. Site confirmation](#2-site-confirmation)
- [3. Caller table](#3-caller-table)
- [4. The fix](#4-the-fix)
- [5. Evidence — RED then GREEN](#5-evidence--red-then-green)
- [6. Gate reconciliation](#6-gate-reconciliation)
- [7. Honest limits](#7-honest-limits)
- [8. Sibling sweep — reported, not fixed](#8-sibling-sweep--reported-not-fixed)

## 1. The defect

Six providers in `helix_code/internal/llm/` implement `GetHealth` — a method NAMED
as a query — that in practice MUTATES the provider's stored health record and then
returns a pointer to that very record. Two faults follow from the one line:

1. **Unguarded concurrent write.** Four of the six DECLARE a mutex (`catalogMu`),
   but it guards only the model catalogue, never `lastHealth`; the other two
   declare no mutex at all. Because the method reads as a query, callers invoke it
   from concurrent paths, so verdicts interleave field-by-field. A provider can be
   reported healthy when the probe found it failing, or the reverse. The error
   counter was worse than interleaved: `updateHealth(…, lastHealth.ErrorCount+1)`
   read the counter at the CALL SITE with no lock held, so two concurrent failures
   both read N and both stored N+1 — one error silently vanished.
2. **Aliasing.** The returned pointer IS the provider's record, so anything a
   caller does to the value it was handed mutates provider state. This half bites
   with or without the race detector.

## 2. Site confirmation

All seven reported sites confirmed — **not merely by reading, but by the race
detector naming them** in `red_baseline_race_prefix.log`:

| Reported | Confirmed | What is there |
|---|---|---|
| `anthropic:991` | YES | `ap.lastHealth = health` (then `return health`) |
| `openai:222` | YES | `op.lastHealth.ModelCount = …` |
| `openai:498` | YES | `updateHealth` body, first field write |
| `deepseek:194` | YES | `dp.lastHealth.ModelCount = …` |
| `deepseek:471` | YES | `updateHealth` body, first field write |
| `mistral:180` | YES | `mp.lastHealth.ModelCount = …` |
| `mistral:448` | YES | `updateHealth` body, first field write |

**Two sites were MISSING from the report** — same package, same method name,
byte-identical defect shape. The reported sweep keyed on the presence of a
`catalogMu` field; these two types do not declare one, so the anchor missed them:

| Additional | What is there |
|---|---|
| `openrouter_provider.go:174` / `:561` | identical `lastHealth` mutate-and-return-live-pointer |
| `openai_compatible_provider.go:343` / `:725` | identical; also reached in production via `XiaomiProvider.GetHealth`, which delegates straight to it |

Fixing four of six identical files in one package is exactly the sibling-miss
pattern the ticket warns about, so all six are fixed here. This is the one
deliberate scope expansion beyond the ticket text.

## 3. Caller table

Every call site that can dispatch to one of the six, with what it does with the
returned pointer:

| # | file:line | Enclosing | Classification |
|---|---|---|---|
| 1 | `internal/llm/openai_provider.go:179` | `OpenAIProvider.IsAvailable` | READ-ONLY (`health.Status == "healthy"`) |
| 2 | `internal/llm/deepseek_provider.go:157` | `DeepSeekProvider.IsAvailable` | READ-ONLY |
| 3 | `internal/llm/mistral_provider.go:143` | `MistralProvider.IsAvailable` | READ-ONLY |
| 4 | `internal/llm/openrouter_provider.go:131` | `OpenRouterProvider.IsAvailable` | READ-ONLY |
| 5 | `internal/llm/openai_compatible_provider.go:287` | `OpenAICompatibleProvider.IsAvailable` | READ-ONLY |
| 6 | `internal/llm/xiaomi_provider.go:237` | `XiaomiProvider.GetHealth` | READ-ONLY passthrough; statically resolves to `*OpenAICompatibleProvider` |
| 7 | `internal/llm/tool_provider.go:467` | `ToolCallingProvider.GetHealth` | READ-ONLY passthrough via the `Provider` interface; constructed only with test doubles, never with a target type |
| 8 | `internal/llm/ensemble_provider.go:923` | `EnsembleProvider.GetHealth` | READ-ONLY (`h.Status`); genuinely dispatches to all six in production via `internal/clientcore/providers.go` |
| 9 | `internal/llm/model_manager.go:186` | `ModelManager.HealthCheck` | **STORES** — puts the pointer in a map that escapes to the caller |

**Does any caller write through the returned pointer? NO.** Every call site only
reads fields. This is the load-bearing question, because HXC-205's sibling had a
caller that depended on the aliased pointer to write health verdicts back into
manager state, and deep-copying alone there would have turned health monitoring
into a silent no-op that still looked green. That trap does **not** apply here.

The one STORES site (#9) was traced through to all of its own consumers —
7 application call sites in `applications/{desktop,aurora_os,harmony_os,terminal_ui}`
plus tests — and every one only reads `.Status`. Post-fix that map holds
independent copies, which is strictly safer; nothing depended on the aliasing.

A second load-bearing fact: `ProviderHealth` (`missing_types.go:114-121`) is a
FLAT value struct — every field is a value type. So `*lastHealth` is a COMPLETE
copy, unlike HXC-205's record which carried pointer and map fields and whose
value copy was therefore only a shallow copy. That property is now pinned by
`TestProviderHealth_HXC214_ProviderHealthIsFlat`, which FAILS if a pointer,
slice, map, chan or func field is ever added — so the aliasing hole cannot
silently re-open through a new field.

## 4. The fix

New shared file `internal/llm/provider_health.go` gives all six ONE verdict
shape, so the fix cannot drift between them:

- `errCountAdjust` (`errCountKeep` / `errCountIncrement` / `errCountReset`) —
  the error count is now an INTENT applied inside the critical section, instead
  of an absolute computed at the call site with no lock held.
- `modelCountUnchanged` — the failure and degraded paths keep the last good
  model count, which is the behaviour the pre-fix code had by omission.
- `applyProviderHealth(h, …)` — writes the verdict and returns `*h` as a copy.
  Caller must hold the lock; the whole verdict lands in ONE critical section so
  no reader observes it half-applied.

Each of the five shape-A providers gains a `healthMu sync.Mutex` and a
self-locking `recordHealth(status, latency, adj, modelCount) *ProviderHealth`.
`updateHealth` is gone from all five (it had no callers outside `GetHealth`).

`AnthropicProvider` keeps its distinct shape — it builds a fresh record per probe
and issues a real `Generate` call — but takes the same two invariants via
`storeHealth`: the write to `lastHealth` is guarded, and the caller never
receives the pointer the provider kept. No network call is made with a lock held
in any of the six.

**Additional race found and fixed in the same method:** `anthropic_provider.go`
read `len(ap.models)` for `ModelCount` with no lock, while `refreshCatalogOnce`
writes `ap.models` under `catalogMu.Lock()`. That read is now under
`catalogMu.RLock()`. This was not in the reported site list.

## 5. Evidence — RED then GREEN

Exit codes captured directly from `go test`, never after a pipe (§11.4.6).
The RED baseline was taken against the genuine unmodified pre-fix artifact
BEFORE any source edit — `git status --porcelain` at that moment showed only the
new test file as untracked, with zero modified production files.

| Run | Log | Exit | Result |
|---|---|---|---|
| RED — aliasing, pre-fix | `red_baseline_aliasing_prefix.log` | **1** | 6/6 providers FAIL: caller's writes reached provider state |
| RED — race, pre-fix, `-race` | `red_baseline_race_prefix.log` | **1** | **46 DATA RACE** reports; every reported site named by the detector |
| RED_MODE=1 harness self-validation | `red_mode1_harness_selfvalidation.log` | **1** | 6 DATA RACE — the golden-bad fixture is seen, so GREEN means "no race" not "blind test" |
| GREEN — guard post-fix, `-race -count=3` | `green_guard_postfix.log` | **0** | 0 races, all subtests pass |
| Paired mutation — copy reverted to live pointer | `paired_mutation_copy_to_live_pointer.log` | **1** | 6/6 aliasing asserts FAIL — the guard breaks the NEW invariant |
| Compile gate — `go build -tags=nogui ./...` | `compile_gate_nogui_and_ci.log` | **0** | whole tree compiles |
| Compile gate — `go vet -tags=ci ./internal/llm/` | `compile_gate_nogui_and_ci.log` | **0** | clean |
| Full package `./internal/llm -race` | `full_package_race_postfix.log` | 1 | **0 data races**; 1 unrelated wall-clock stress-test timeout — see below |
| Contention isolation of that timeout | `contention_isolation_stress_test.log` | **0** | passes in 4.3 s when run alone |
| Full package `./internal/llm -race`, clean re-run | `full_package_race_postfix_rerun.log` | **0** | 0 data races, 0 failures, 352 s — the stress test passes |

### The one full-package failure was host contention, not a defect (§11.4.201)

The first whole-package run reported **zero data races** and exactly one failure:
`TestModelManager_Stress_ConcurrentRegisterAndRead` tripped its 25 s deadlock
guard. Investigated rather than reported as a defect:

- **The HXC-214 change is not in that test's code path.** It builds its registry
  from `newStressTestProvider(...)` — a purpose-built stub in
  `manager_stress_test.go` — not from any of the six providers this change
  touches, and its goroutines call `GetAvailableModels` /
  `GetModelsByCapability` / `SelectOptimalModel`, never `GetHealth`.
- **The host was heavily oversubscribed.** Load average was 107 on a 64-core box
  during that run (other agents are live in this checkout — `internal/deployment`
  and `internal/tools/shell` were both modified by concurrent work), and this
  session was running the `-tags=nogui` whole-tree compile gate at the same time.
  The failing assertion is a 25 s wall-clock budget for 16 goroutines under
  `-race`, which is precisely the assertion class that starvation breaks.
- **Isolated, it passes in 4.3 s** (`contention_isolation_stress_test.log`,
  exit 0) — even with load average still at 171, because run alone it is not
  competing with the rest of the package.

No lock-order inversion is introduced by this change: `ModelManager.HealthCheck`
takes `m.mu.RLock()` then a provider's `healthMu`, and nothing anywhere takes
`healthMu` before `m.mu` — `recordHealth` calls only `applyProviderHealth`, which
touches no other lock.

The paired mutation is the §1.1 pair against the NEW mechanism (§11.4.120):
`applyProviderHealth` and `AnthropicProvider.storeHealth` were changed to return
the live pointer, the guard FAILED for all six, and both files were restored and
verified byte-identical by `sha256sum -c` (exit 0). A mutation-marker residue
scan over `helix_code/internal/` returned only one hit — a pre-existing,
unrelated comment in `internal/memory/helixmemory_provider_test.go:184`, a file
this change never touched, confirmed by `git status --porcelain`.

## 6. Gate reconciliation

Two pre-existing tests drove `OpenAICompatibleProvider.updateHealth`, which this
change removes:

- `internal/llm/local_providers_test.go` — `TestHealthStatusUpdate`
- `internal/llm/openai_compatible_provider_test.go` — `TestOpenAICompatibleProvider_UpdateHealth`

Investigated per §11.4.120 before touching either. The break is a COMPILE failure
from a renamed private method, not a behavioural regression: the invariant they
guard — a verdict writes Status, Latency and ErrorCount onto the stored record —
is preserved by `recordHealth`, and both tests still assert exactly that. They
were RECONCILED to the new mechanism, not fake-passed and not reverted, and were
STRENGTHENED: they now also pin the per-intent ErrorCount semantics
(reset / increment / keep), the model-count-preserved-on-failure behaviour, and
(in `TestHealthStatusUpdate`) that the returned record is a copy.

The discriminator that this is reconciliation rather than bluffing: the guard and
its mutation still form a valid §1.1 pair — mutating the new invariant makes the
gate FAIL, as `paired_mutation_copy_to_live_pointer.log` shows.

The other `updateHealth` references in the package (`copilot_provider_test.go`,
`koboldai_provider_test.go`, `local_provider_test.go`) belong to different
provider types that this change does not touch; their `updateHealth` methods are
untouched and those tests are unmodified.

## 7. Honest limits

- **The anthropic `ap.models` race is fixed by inspection, not RED-captured.**
  Provoking it needs a concurrent `refreshCatalogOnce`, which performs a live
  catalogue fetch — driving that deterministically would have made the guard
  network-dependent and flaky. The `lastHealth` write-write race for anthropic
  IS captured (`anthropic_provider.go:991` appears in the RED detector output).
- **Aliasing for anthropic was latent, not active.** `ap.lastHealth` is written
  but never read anywhere in the package, so a caller corrupting the returned
  record could not change any observable judgement today. The returned pointer
  was nonetheless identical to the stored one, and the unguarded pointer write
  was a genuine race. The field is kept, not removed: §11.4.124 requires captured
  proof before removing seemingly-dead state, which this change did not gather.
- **`OpenAICompatibleProvider.isRunning` remains unguarded** — written by `Stop()`
  at `:350`, read at `:225 :255 :283 :293` including inside `GetHealth`. It is a
  different field on a path that spans five call sites outside the health record;
  bringing it under a lock belongs to its own change rather than being smuggled
  into this one. Documented on the `healthMu` field and reported here.
- **`openai`'s first error path changed behaviour deliberately.** It used a
  literal `updateHealth("unhealthy", 0, 1)` — forcing ErrorCount to 1 regardless
  of history — where its five siblings used `prev+1`. It now increments like the
  others. This is a normalisation, disclosed rather than silent.
- The error paths of `GetHealth` were not driven by the guard against a failing
  endpoint; the harness serves well-formed responses so the SUCCESS path (the one
  that writes and returns the live pointer) is exercised. The error paths'
  locking is the same `recordHealth` call and rests on review, not execution.

## 8. Sibling sweep — reported, not fixed

Full machine-generated listing: `sibling_sweep_live_aggregate_returns.txt`
(66 sites across `helix_code/internal/**`). Nothing in this section is fixed here.

Highest risk — type declares a mutex AND the returned aggregate is genuinely
mutated after construction. The `GetModels()` group returns the live `models`
slice; in the `catalogOnce`/`catalogMu` providers `refreshCatalogOnce` REPLACES
`models` after construction, and a caller holding the returned slice can write
through it into the provider's catalogue:

| file:line | Method |
|---|---|
| `internal/llm/anthropic_provider.go:397` | `AnthropicProvider.GetModels` |
| `internal/llm/openai_provider.go:107` | `OpenAIProvider.GetModels` |
| `internal/llm/deepseek_provider.go:101` | `DeepSeekProvider.GetModels` |
| `internal/llm/mistral_provider.go:85` | `MistralProvider.GetModels` |
| `internal/llm/openrouter_provider.go:81` | `OpenRouterProvider.GetModels` |
| `internal/llm/openai_compatible_provider.go:215` | `OpenAICompatibleProvider.GetModels` |
| `internal/llm/azure_provider.go:467` | `AzureProvider.GetModels` |
| `internal/llm/groq_provider.go:191` | `GroqProvider.GetModels` |
| `internal/llm/vertexai_provider.go:452` | `VertexAIProvider.GetModels` |

Same `GetModels` shape, no mutex declared: `bedrock_provider.go:422`,
`copilot_provider.go:192`, `gemini_provider.go:300`, `koboldai_provider.go:119`,
`local_provider.go:67`, `qwen_provider.go:118`, `xai_provider.go:73`,
`providers/cerebras/cerebras.go:121`.

The `config` cluster — `return X.config` / `GetConfig` / `GetConfiguration`
handing out a live `*Config` or `map[string]interface{}`, with a mutex declared:
`internal/config/config.go:907`, `internal/mcp/registry.go:42`,
`internal/llm/compression/compressor.go:375`,
`internal/memory/cognee_integration.go:541`, and the ten
`internal/memory/providers/*_provider.go` `GetConfiguration` implementations
(`anima:188`, `baseai:167`, `character_ai:1101`, `chromadb:153`, `faiss:870`,
`memonto:210`, `pinecone:177`, `qdrant:152`, `weaviate:217`, `zep:107`);
plus `internal/memory/providers/config.go:425` and `mem0_provider.go:91` with no
mutex.

Also notable, mutex-declaring types handing out live internal pointers:
`internal/agent/base_agent.go:883 :948 :964 :982`,
`internal/tools/registry.go:1542 :1547 :1552 :1557 :1567 :1572`,
`internal/providers/ai_integration.go:495 :500 :505 :510 :1191`,
`internal/workflow/executor.go:108`, `internal/workflow/autonomy/controller.go:287`,
`internal/workflow/autonomy/executor.go:282`, `internal/tools/web/cache.go:170`,
`internal/notification/ratelimit.go:155`, `internal/cognee/service.go:948 :953`,
`internal/context/context_manager.go:215`, `internal/llm/vision/switcher.go:275`.
