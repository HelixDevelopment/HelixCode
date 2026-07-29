# HXC-203 — LocalLLMManager provider-status data race: RED → GREEN evidence

| Field | Value |
|---|---|
| Revision | 1 |
| Created | 2026-07-29 |
| Last modified | 2026-07-29 |
| Status | active |
| Item | HXC-203 (Bug, Critical) — release blocker for `helix-code-1.2.0-dev-0.0.1` |
| Track | (T1/main - claude2 - opus - xhigh) |
| Subject | `helix_code/internal/llm/local_llm_manager.go` |
| Guard | `helix_code/tests/unit/local_llm_manager_hxc203_race_test.go` |

## Table of contents

- [The defect](#the-defect)
- [Captures](#captures)
- [What each capture proves](#what-each-capture-proves)
- [Unrelated change carried in this diff](#unrelated-change-carried-in-this-diff)
- [Independent review](#independent-review)
- [Host conditions](#host-conditions)
- [Not verified](#not-verified)

## The defect

Three compounding parts, all confirmed on the pre-fix artifact rather than assumed:

1. `GetProviderStatus` was named as a query but **mutated** shared state — it overwrote
   each provider's `Status` and stamped `LastCheck` on every call.
2. `LocalLLMManager` carried **no mutex of any kind** (verified by grep for
   `sync.Mutex` / `sync.RWMutex` / `mu ` returning nothing at `HEAD`), so two
   concurrent callers wrote over each other.
3. It returned `m.providers` — the **live internal map** — so every caller received
   pointers into manager state that it could also mutate. The race the detector
   caught was therefore only the instance that happened to be exercised.

`isProviderHealthy` performs network I/O, so no lock may be held across it: a hung
provider would otherwise block every reader for the full HTTP timeout. That
constraint is what forced the three-phase refresh rather than wrapping the loop.

## Captures

Every exit code below was captured directly from the command, never through a pipe.

| # | File | Command | Exit |
|---|---|---|---|
| 1 | `red_baseline_prefix.log` | `go test -race -count=1 -run TestLocalLLMManager_ConcurrentAccess ./tests/unit/` on the **pre-fix** artifact | **1** (DATA RACE) |
| 2 | `green_after_fix.log` | same command, post-fix | 0 |
| 3 | `full_tests_unit.log` | `go test -race -count=1 ./tests/unit/` | 0 |
| 4 | `full_internal_llm.log` | `go test -race -count=1 ./internal/llm/` | 0 |
| 5 | `guard_green_default.log` | `go test -race -count=3 -run TestLocalLLMManager_HXC203 ./tests/unit/` | 0 |
| 6 | `guard_redmode1_selfvalidation.log` | `RED_MODE=1 go test -race -count=1 -run ..._HarnessSelfValidation ./tests/unit/` | **1** (2× DATA RACE) |
| 7 | `guard_red_on_prefix_artifact.log` | new guard run against the **pre-fix** source (paired mutation) | **1** |
| 8 | `green_after_gofmt.log` | `go test -race -count=2 -run TestLocalLLMManager ./tests/unit/` | 0 |

## What each capture proves

**(1) The reported race is real and reproduced verbatim.** The pre-fix run reproduces
the filed signature exactly — write/write on the same address, both arms inside
`GetProviderStatus` at the `LastCheck` line, one of them reached through
`GetRunningProviders`. (The address differs from the filed report only by heap
layout, which is expected.)

**(7) The new guard is RED on the broken artifact (§11.4.115).** This is the
load-bearing capture: the guard was run against the pre-fix source restored from
`HEAD`, and it failed — so it reproduces the defect rather than merely agreeing with
the fix. Both halves fired, and the second is independent of the race detector:

- `..._ConcurrentStatusIsRaceFree` → `WARNING: DATA RACE` (defects 1 + 2).
- `..._StatusIsNotLiveInternalState` → the caller's `delete()` on the returned map
  **actually deregistered `gpt4all` from the manager**, proving defect 3 concretely.

The mutation was applied and reverted inside a single command with a restore trap;
the working tree was verified byte-identical afterwards
(`md5 79f3ee7f50f3621677b258d1544fe904` before and after).

**(6) The guard is not blind (§11.4.107(10) golden-bad fixture).** A data race has no
in-process assertion API, so a GREEN run only means something if the harness can be
shown to see a race at all. Under `RED_MODE=1` the guard performs the pre-fix
unsynchronized `provider.LastCheck = time.Now()` write concurrently on a shared
record; the detector reports it and the run exits non-zero. If that run ever goes
clean, every GREEN result from this guard is worthless.

**(2, 3, 4, 5, 8) The fix holds and nothing regressed.** Both suites that exercise
this type pass under `-race`, including the pre-existing
`TestLocalLLMManager_ConcurrentAccess` that filed the bug.

## Unrelated change carried in this diff

The diff also contains a **whitespace-only `gofmt` realignment of the
`NewLocalLLMManager` struct literal**. It is not part of the fix. The file was
already `gofmt`-dirty at `HEAD` (verified by extracting `HEAD`'s copy to a temp
path and running `gofmt -l` on it, which listed it), in a region this change
does not otherwise touch. It was normalised so that a file this change does
touch is not left failing `make fmt`. No behaviour, no logic, no semantics —
but it is disclosed here rather than left for a reader to discover, because
"only X changed" is a claim like any other.

## Independent review

Reviewed per §11.4.142 / §11.4.125 / §11.4.194 by an independent agent on the
§11.4.209 substrate (Fable, xhigh effort), adversarially, across seven angles:
deadlock, lock-across-blocking-work, remaining unguarded shared state,
compare-and-set correctness, caller-visible behaviour change, whether the guard
test can actually fail, and comment honesty.

**Verdict: GO, zero blocking findings.** The reviewer re-ran the build, vet,
gofmt and both `-race` suites itself, and confirmed the cited RED evidence file
exists and contains a real DATA RACE report rather than taking the claim on
trust.

Non-blocking findings and their disposition:

| # | Finding | Disposition |
|---|---|---|
| F3 | `skipProviderInstall` was an unguarded mutable field on this same type | **Fixed** — the only finding judged to be genuinely the same defect on the same type |
| F1 | ABA window in the phase-3 compare-and-set: a stop+restart inside one probe window can apply a stale `unhealthy` verdict to the new instance | Follow-up. The dangerous direction (resurrecting a stopped provider) is already blocked and the wrong verdict self-corrects on the next refresh, so it is a refinement of the new code rather than the filed defect, and it deserves its own RED test instead of being bundled here. **It is not an expensive fix** — see the correction note below |
| F2 | `Initialize` check-then-act TOCTOU | Follow-up. Pre-existing, identical shape pre-fix, logical race not a data race |
| F4 | White-box tests in `internal/llm/local_llm_manager_test.go` bypass the accessors and touch private state directly: a write at `:196-197` (`providers["vllm"].Status`) and reads at `:25` (`isInitialized`) and `:46-48` (`skipProviderInstall`, read immediately after the now-locking `SetSkipProviderInstall`) | Follow-up, all three sites as one item. Sequential today, so nothing races; deletion or rework is governed by §11.4.124 investigate-before-remove |
| F5 | The GREEN guard's write traffic depends on phase 3 stamping `LastCheck` on all providers; a compound future regression could evade it | Disclosed below under "Not verified" |
| F6 | **The same defect class is live in the sibling type `AutoLLMManager`** | Follow-up ticket — see below |
| F7/F8 | `.go` source carries mode 100755; `HealthURL` is write-only | Nits, both pre-existing |

**Correction to F1's deferral rationale (round 2).** An earlier revision of this
document justified deferring F1 partly on the grounds that it would change an
exported, JSON-serialised API. That was wrong and is retracted: an *unexported*
`generation` field is invisible to `encoding/json`, and a manager-side
`map[string]uint64` keyed by provider name needs no struct change at all. The
deferral still stands on its remaining grounds, but the follow-up ticket must be
written up as a **cheap** fix so a future reader does not price it as an
API-surface change.

**F6 is the one worth acting on next.** `internal/llm/auto_llm_manager.go`
*owns* a `sync.RWMutex` (`:84`, used at `:101` and `:253`) yet writes
`Status` / `Process` / `LastHealthCheck` outside it at `:474-475`, `:643-645`
and `:814-816`. Verified directly, not relayed on trust. That is the same
defect class as HXC-203 on a different type — the "applied in one place, not to
its sibling doing the same thing nearby" pattern that HXC-202 records as having
already occurred seven times in this cycle. It is out of HXC-203's declared
scope (different type, untouched by this diff) and should be filed separately
per §11.4.146 STEP 3 so the class is actually emptied rather than this one
instance.

## Host conditions

Captured on a quiet host: 64 CPUs, load average 2.04, no competing build and no
other project's test stack running (checked per §11.4.174 before trusting any
verdict). The race detector does not report false positives, and quiescence rules
out contention-induced noise in the timing-sensitive runs.

## Not verified

- No provider was ever in the `running` state during these runs, because the test
  manager runs with `SetSkipProviderInstall(true)` and starts nothing. The probe
  path in `refreshProviderHealth` (phases 2–3, including the compare-and-set that
  reconciles a verdict with a concurrent `StopProvider`) is therefore **covered by
  reasoning and review, not by executed evidence**. Exercising it needs a real or
  stubbed local provider endpoint; that gap is stated here rather than implied
  closed. This is also the root of review finding F5: the GREEN guard's write
  traffic comes from phase 3 stamping `LastCheck` on every provider, so a
  hypothetical future regression that dropped the locking *and* simultaneously
  narrowed stamping to running-only providers would produce no writes in this
  topology and stay green. The `RED_MODE=1` fixture and the live-map assertion
  both remain effective against it, but closing the gap properly needs a
  white-box companion that drives a provider through the probe arm.
- Review finding F1 (the ABA window in the compare-and-set) is interleaving
  analysis, not a live reproduction — reproducing it needs a real health endpoint
  with a scripted stop/restart inside the probe window.
- `go build ./...` fails on the GUI packages (`go-gl`, `glfw`) for missing
  `X11/Xlib.h` and `gl.pc` on this host. Pre-existing and unrelated:
  `go build -tags=ci ./applications/...` exits 0, as do
  `go build ./internal/... ./cmd/...`.
