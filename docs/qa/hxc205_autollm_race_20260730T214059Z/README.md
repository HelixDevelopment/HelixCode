# HXC-205 — AutoLLMManager provider-state data race

Captured evidence for the fix to the `AutoLLMManager` synchronization defect.
Every exit code below was captured directly from `$?`, never after a pipe.

## The defect

`AutoLLMManager` **declares** a `sync.RWMutex` (`auto_llm_manager.go:84`) and
uses it in seven methods, then writes provider health, process handles and
last-checked timestamps from **nine** other methods that take no lock at all.
That is worse than carrying no lock: a reader sees synchronization and
reasonably assumes it applies throughout.

The tracker predicted "three other places". The real count is nine — see the
shared-state table below. The prediction was treated as a hypothesis and
verified, not trusted.

Two writers reach the **same** `*HealthStatus` concurrently in production: with
`AutoMonitor` enabled, `Initialize` starts the "health" background task
(`autoHealthCheck`) and `Start` launches the `HealthMonitor` goroutine
(`performHealthChecks`). Both stamp `Health.LastCheck / ResponseTime /
IsHealthy / Status / Error` on the same record while `GetStatus` value-copies
it for the CLI and the load balancer.

A third, compounding part: `GetStatus` returned `providerCopy := *v`, a
**shallow** copy. `Health`, `Metrics` and `Config` are pointer/map fields, so
every caller received live references into manager state.

## Shared-state table (§11.4.102)

Guarded state: the `providers` and `backgroundTasks` maps; per-provider
`Status`, `Process`, `Config`, `Health.*`, `Metrics.*`, `LastHealthCheck`,
`RetryCount`; per-task `IsRunning`, `LastRun`; plus `isInitialized`,
`isRunning`, `metricsRecorder`, `healthMonitor`, `loadBalancer`.

Line numbers are pre-fix (`fcb8833d`).

| Site | Field(s) | Under lock pre-fix? |
|---|---|---|
| `SetMetricsRecorder` :101 | metricsRecorder | YES (Lock) |
| `Initialize` :253 | isInitialized, providers, backgroundTasks | YES (Lock) |
| `Start` :296 | isRunning, healthMonitor, loadBalancer | YES (Lock) |
| `updatePerformanceMetrics` :883-885 | metricsRecorder (read only) | PARTIAL — read guarded, writes not |
| `GetStatus` :991 | providers (read) | YES (RLock) but **shallow copy** |
| `GetRunningEndpoints` :1007 | providers, Status, Health.IsHealthy | YES (RLock) |
| `Stop` :1022 | isRunning, backgroundTasks, Process | YES (Lock), but Kill held the lock |
| `initializeProviders` :426 | providers | YES — called under `Initialize`'s Lock |
| `startBackgroundTasks` :689/:702/:715 | backgroundTasks | YES — called under `Initialize`'s Lock |
| **`autoInstallAllProviders` :441 :474 :475** | providers iter, Status, LastHealthCheck | **NO** |
| **`autoConfigureProvider` :566** | Config | **NO** |
| **`autoStartProvider` :643 :644 :645 :651 :655 :656** | Process, Status, LastHealthCheck | **NO** |
| **`autoStartAllProviders` :576** | providers iter, Status | **NO** |
| **`autoHealthCheck` :753 :762-:773 :776 :782** | Health.*, RetryCount, LastHealthCheck | **NO** |
| **`autoRecoverProvider` :814-:816** | Process | **NO** |
| **`autoPerformanceOptimization` :835 :848 :852** | providers iter, Status, Metrics.* | **NO** |
| **`updatePerformanceMetrics` :880 :897-:903** | Metrics.* | **NO** |
| **`autoUpdateCheck` :911** | providers iter | **NO** |
| **`autoUpdateProvider` :963-:965** | Process | **NO** |
| **`runBackgroundTask` :728 :735 :739 :746** | task.IsRunning, task.LastRun | **NO** |
| **`HealthMonitor.updateProviderHealth` health_monitor.go:107-:121** | Health.*, RetryCount | **NO** |
| **`HealthMonitor.handleUnhealthyProvider` health_monitor.go:138** | RetryCount (read) | **NO** |
| **`HealthMonitor.triggerAutoRecovery` health_monitor.go:154** | RetryCount (read) | **NO** |

## Where the twin's (HXC-203) shape did and did not fit

It fits, with one deliberate difference that the code forced.

* **Three-phase refresh — fits, applied per-provider.** `autoHealthCheck` now
  snapshots under the lock, probes with no lock held, and writes the verdict
  back in one critical section. The batch form the twin used was unnecessary
  because the loop already iterates a snapshot.
* **Never hold the lock across blocking work — fits.** No lock is held across a
  health probe, `exec`, `git fetch`, `Kill` or `time.Sleep`.
* **Do not return live internal state — fits, and was the sharper bite here.**
  `GetStatus` now returns a deep copy.
* **The difference:** `HealthMonitor.performHealthChecks` **depended** on the
  shallow copy. It wrote verdicts into manager state only by accident, through
  the aliased `Health` pointer. Deep-copying `GetStatus` without giving the
  monitor a real write path would have turned health monitoring into a silent
  no-op that still looked green — and would have broken the pre-existing
  `TestHealthMonitor_PerformHealthChecks_WithRunningProvider`. That test failing
  would have been the gate correctly catching a regression, not a stale gate
  (§11.4.120). `performHealthChecks` therefore iterates the manager's **live**
  records and writes back through the manager's accessors.

## Evidence

| Run | File | Exit |
|---|---|---|
| New guard vs **pre-fix** source (§11.4.115 RED, paired mutation) | `red_baseline_prefix.log` | **1**, 22 DATA RACE reports, 3/3 guards FAIL |
| New guard post-fix, `-count=3` | `green_postfix.log` | **0**, 0 races |
| `RED_MODE=1` harness self-validation (§11.4.107(10) golden-bad) | `red_mode1_self_validation.log` | **1**, 6 DATA RACE reports |
| Targeted §1.1 mutation: deep copy reverted to shallow | `paired_mutation_shallow_copy.log` | **1**, aliasing assertions FAIL |
| All pre-existing AutoLLM/HealthMonitor tests, `-race` | `regression_existing_tests.log` | **0** |
| Whole `./internal/llm` package, `-race` | `full_pkg_internal_llm.log` | **0** (142s) |
| `./tests/unit`, `-race` | `tests_unit.log` | **0** |

The **RED baseline is the load-bearing capture**: it was taken against the
genuine unmodified pre-fix artifact before any source edit, so the guard
reproduces the defect rather than merely agreeing with the fix. It also proved
a consequence that was predicted and then confirmed: `RetryCount` came back
`0` where `2` was required, because pre-fix it was written to the throwaway
copy `GetStatus` returned and lost on every cycle.

The targeted mutation was applied to a `cp` of the file and restored from that
copy; both sides verified byte-identical by `sha256sum`
(`7db3d534817d5f3115aa243e125ff623bc6668973f8dc2c55477f366d662debf`). A
mutation-residue scan over all three touched files returned clean (§11.4.84).

## Honest limits (§11.4.6)

1. **No provider was ever genuinely `running` as an OS process in these tests.**
   The seeded providers carry `Status: "running"` and a live `httptest` health
   endpoint, which exercises the health/metrics/status paths for real, but
   `autoStartProvider`, `autoRecoverProvider`, `autoUpdateProvider`, `Stop`'s
   kill loop and `takeProcess` were **not** executed against a live child
   process. Their correctness rests on reasoning and review, not on executed
   evidence. This is the same gap the HXC-203 twin disclosed, and it is
   unchanged here.
2. **`config` is treated as immutable after `NewAutoLLMManager` returns** and is
   read unguarded (`autoHealthCheck`, `handleUnhealthyProvider`). Nothing in
   production writes it post-construction; the existing tests do, but only
   before starting any goroutine. This is documented on the `mutex` field
   rather than locked, matching the twin's treatment of immutable fields. If a
   runtime config-reload is ever added, it must be brought under the lock.
3. **No compare-and-set on the health verdict.** The twin needed one because its
   refresh probed many providers in one batch. Here the probe→writeback window
   is per-provider and the writeback is a single critical section, so a verdict
   cannot be applied in torn form. A verdict formed microseconds before a
   concurrent stop can still be applied; that is a pre-existing behaviour, not
   introduced here, and it is not what the ticket describes.
4. **The sibling sweep is static.** Tier-1/2 findings below were verified by
   reading the code; tier-3 are candidates that still need triage. No sibling
   was executed under `-race`.

## Sibling sweep (§11.4.1 — reported, NOT fixed here)

Method: a static scan for the exact HXC-205 signature — a type that **declares**
a mutex, and a method on that type that writes a receiver field while the
method body contains no `Lock()`/`RLock()` anywhere. 51 raw candidates;
`*Locked`-convention helpers and construction-time initialisers were filtered
out by reading them. Also scanned: query-named methods that mutate, and methods
returning a live internal map/slice.

### Tier 1 — VERIFIED, same class, highest priority

**`internal/deployment/production_deployer.go`** — `mutex sync.RWMutex` declared
at `:28`, used at **exactly one** call site (`:1187`), against **78** accesses
to `pd.status.*`. Unguarded writes:

- `:242` `StartProductionDeployment` → `pd.status.CurrentPhase`
- `:358` `executeSecurityCheck` → `pd.status.SecurityGateStatus.Status`
- `:415` `executePerformanceCheck` → `pd.status.PerformanceGate.Status`
- `:517` `executeDeployment` → `pd.status.EndTime`
- `:570` `executeProductionDeploy` → `append(pd.status.ServersDeployed, ...)`
- `:734` `executeHealthCheck` → `pd.status.HealthStatus`
- `:1078` `completeDeployment` → `pd.status.EndTime`
- `:1096` `failDeployment` → `pd.status.EndTime`
- `:1119` `triggerRollback` → `pd.status.RollbackTriggered`

This one is already **proven**, not merely suspected: the comment at
`:1180-1186` records a real race-detector report at `:1026-1027` and states the
struct "already declar[es] pd.mutex for exactly this purpose". A previous fix
guarded that **single** site and left the rest of the class live — the same
partial-remediation pattern HXC-205 is.

### Tier 2 — VERIFIED, both bites present (query-mutates + returns-live-state)

Four LLM providers each declare `catalogMu sync.RWMutex` (which guards only the
model catalogue), then mutate `lastHealth` **and hand back the live
`*ProviderHealth` pointer** with no lock:

| File | Unguarded write | Returns live pointer |
|---|---|---|
| `internal/llm/anthropic_provider.go` | `:991` `GetHealth` → `ap.lastHealth = health` | `:991` region |
| `internal/llm/openai_provider.go` | `:222` `GetHealth`, `:498` `updateHealth` | `:189` `:200` `:206` `:217` |
| `internal/llm/deepseek_provider.go` | `:194` `GetHealth`, `:471` `updateHealth` | `:165` `:174` `:180` |
| `internal/llm/mistral_provider.go` | `:180` `GetHealth`, `:448` `updateHealth` | `:151` `:160` `:166` |

`GetHealth` is query-named and mutates shared state — the same naming lie
HXC-203 carried in `GetProviderStatus`.

### Tier 3 — CANDIDATES, need triage (not yet verified reachable-concurrently)

- `internal/notification/metrics.go:151` `updateResponseTime` → `m.TotalResponseTime += duration`
- `internal/notification/ratelimit.go:67` `refill` → `r.tokens = r.maxRequests`
- `internal/llm/model_manager.go:359` `calculateHardwareCompatibility` → `m.hardwareDetectErr`
- `internal/memory/providers/character_ai_provider.go:1170` `loadCharacters` → `p.stats.TotalCollections`
- `internal/memory/providers/pinecone_provider.go:94` `testConnection` → `p.config.Host`
- `internal/tools/voice/recorder.go:343` `LevelMonitor.calculateLevels` → `l.currentPeak`
- `internal/tools/interactive_tools.go:99` `TaskTrackerTool.Execute` → `t.tasks = make(...)`
- `internal/verifier/poller.go:74` `Poller.loop` → `p.ticker`
- `internal/mcp/server.go:177` `SetAllowedOrigins` → `s.upgrader.CheckOrigin`
- `internal/llm/cross_provider_registry.go:357` `loadRegistry` → `r.compatibility`

### Tier 3b — methods returning a live internal map/slice

`return X.models` on ~15 provider types (`openai_compatible_provider.go:200`,
`openrouter_provider.go:73`, `local_provider.go:67`, `gemini_provider.go:300`,
`mistral_provider.go:78`, `openai_provider.go:100`, `xai_provider.go:73`,
`groq_provider.go:191`, `deepseek_provider.go:94`, `copilot_provider.go:192`,
`vertexai_provider.go:452`, `qwen_provider.go:118`, `anthropic_provider.go:390`,
`bedrock_provider.go:422`, `koboldai_provider.go:119`,
`providers/cerebras/cerebras.go:121`, `azure_provider.go:467`);
`return p.config` on four memory providers (`qdrant:152`, `chromadb:153`,
`pinecone:177`, `weaviate:217`); `internal/config/platform_ui_adapters.go:65`
and `:70`; `internal/llm/compression/retention.go:120`.

These are aliasing exposures rather than proven races — most are written once at
construction — but they are the same "caller receives live internal state"
shape, and worth a pass.
