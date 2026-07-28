# HXC-154 — QA-session creation response was non-deterministic

| | |
|---|---|
| Revision | 1 |
| Created | 2026-07-29 |
| Last modified | 2026-07-29 |
| Status | active |
| Item | HXC-154 (Bug / Medium) → `Fixed (→ Fixed.md)` |
| Scope | `helix_code/internal/helixqa`, `helix_code/internal/server` |
| Host | 64 CPU, Linux 6.12.41; Go race detector |

## Table of contents

- [1. Reported symptom and what it actually was](#1-reported-symptom-and-what-it-actually-was)
- [2. Reproduction, measured](#2-reproduction-measured)
- [3. Confirmed root cause](#3-confirmed-root-cause)
- [4. The fix](#4-the-fix)
- [5. RED → GREEN evidence](#5-red--green-evidence)
- [6. Paired §1.1 mutation](#6-paired-11-mutation)
- [7. Knock-on findings and their reconciliation](#7-knock-on-findings-and-their-reconciliation)
- [8. Verification](#8-verification)
- [9. Honest boundaries](#9-honest-boundaries)

## 1. Reported symptom and what it actually was

`TestStartQASession_Success` asserted the 201 body reported `status: "pending"` and
occasionally saw `"running"`.

The item reported the mechanism as: *the handler serialises a struct while another
goroutine writes it.* **That framing is refuted as a data race** (§11.4.6):
`(*SessionState).MarshalJSON` takes `state.Mu.RLock()` and the orchestrator takes
`state.Mu.Lock()`, so every access is synchronised. Across **128 parallel `-race` runs**
on the pre-fix artifact there were **3 failures and ZERO `WARNING: DATA RACE` reports** —
all 3 were plain assertion failures (`expected: "pending" / actual: "running"`).

It is a **race condition** (unordered happens-before), not a data race. The user-visible
consequence is larger than a flaky test: the endpoint answered `201 Created` with a
non-deterministic `status`, and under contention **2 of 512 responses reported
`"completed"`** — a 201 Created announcing an already-finished resource.

## 2. Reproduction, measured

| Condition | Result |
|---|---|
| 60 serial `-race` runs of the original test | 60 pass / 0 fail — too rare to be an oracle |
| 128 parallel `-race` runs (32 × 4) | 125 pass / **3 fail** (2.3%) |
| 9000 requests in-process, `-race` | 238 non-`pending` (**2.64%**) |
| 9000 requests in-process, no `-race` | 78 non-`pending` (**0.87%**) |
| 512 requests, 64 concurrent clients (pre-fix) | **106 non-`pending` (20.7%)**, incl. 2 × `completed` |

Trial counts in the guards are derived from these numbers, not guessed: at the
conservative 0.87%, 2000 trials miss a regression with probability
`(1-0.0087)^2000 ≈ 3e-8`.

## 3. Confirmed root cause

`Engine.StartSession` built the session with `Status: "pending"`, registered it, spawned a
goroutine whose **first act** is `state.Mu.Lock(); state.Status = "running"`, and then
returned **that same pointer**. `startQASession` passed the live pointer to `c.JSON`.
Whether the marshal or the goroutine went first was pure scheduling.

The handler was never the wrong layer — it faithfully rendered what the engine handed it.

## 4. The fix

`StartSession` now captures a **detached creation snapshot before the goroutine exists**
and returns that. The live handle stays reachable via `GetSession(id)`, which is the
documented way to observe progress. `qa_handlers.go` is unchanged.

This also removes the footgun that caused two earlier defensive workarounds in the test
suite (see §7).

## 5. RED → GREEN evidence

Both guards carry the §11.4.115 `RED_MODE` polarity switch and use the package-shared
`redMode` helper (per the `internal/server` convention: a guard with no guard-specific
switch uses the umbrella).

| Artifact | `RED_MODE=1` | `RED_MODE=0` (standing guard) |
|---|---|---|
| Pre-fix | **PASS** (defect present) | **FAIL** (guard falsifiable) |
| Fixed | **FAIL** (polarity flips) | **PASS** |

- `logs/engine_red1.log`, `logs/engine_red0.log`, `logs/server_red1.log`, `logs/server_red0.log` — pre-fix
- `logs/engine_green.log`, `logs/server_green.log`, `logs/concurrent_green.log` — fixed

The engine-level guard is **deterministic in both polarities**: `Shutdown()` blocks on the
engine WaitGroup and every exit path of the orchestrator commits a terminal status, so the
happens-after edge is exact. Pre-fix it caught the caller's handle holding
`Status:"cancelled"` where the caller had been handed `"pending"`.

The HTTP guard is necessarily statistical — once gin writes the bytes the body is frozen,
so no test can force the losing interleaving after the fact. Its trial count is calibrated
from §2.

**After the fix:** the original 32-parallel harness that produced 3/128 failures ran
**256/256 clean** (double the sample), and the new guard ran 20× = **40,000 responses, all
`pending`**.

## 6. Paired §1.1 mutation

Reverting only the fix (`return created` → `return state`, keeping the snapshot call so it
still compiles — a *behavioural*, not a compile, mutation):

| Guard | Mutated (fix reverted) |
|---|---|
| engine, GREEN polarity | **FAIL** (`logs/mut_engine.log`) |
| server serial, GREEN | **FAIL** — `pending=1947 running=53` (`logs/mut_server.log`) |
| server concurrent, GREEN | **FAIL** — `pending=406 running=104 completed=2` (`logs/mut_concurrent.log`) |
| both, RED polarity | **PASS** (defect reproduces again) |

A first mutation attempt was discarded because it broke compilation (unused variable) — a
compile break is not a mutation proof.

## 7. Knock-on findings and their reconciliation

The fix surfaced two **pre-existing latent** test defects. Neither is a product regression;
both were reconciled per §11.4.120, never fake-passed and never worked around by reverting.

### 7a. `TestGetQASessionReport_SessionNotCompleted` — caused by this fix, test premise unsound

Failure rate rose from ~10% to ~56%. Isolated to this fix on an otherwise identical tree
(fix reverted: 3/25 fail; fix applied: 14/25 fail).

**Mechanism, proven not assumed:** pre-fix the handler marshalled the *live* session, so
its `MarshalJSON` RLock contended with the orchestrator's `state.Mu` writes and
**accidentally throttled it**. Re-introducing that contention on the fixed build halved
the failure rate again (21/25 → 13/25), confirming the mechanism.

The test assumed a freshly-created session would still be un-completed when its report was
requested — pure scheduling luck. It now starts the session with an **invalid platform**,
so `ParsePlatforms` fails and the session settles on `"failed"`; every state it can be
observed in is `!= "completed"`, so the 409 branch is exercised with probability 1.
**40/40 deterministic**, and still falsifiable: disabling the handler's 409 branch makes it
FAIL (`logs/mut_409.log`).

### 7b. `TestEngine_ListSessions` — NOT caused by this fix

Started two real sessions against a `t.TempDir()` and never called `Shutdown()`, so its
orchestrator goroutines outlived the test and raced temp-dir cleanup
(`unlinkat ...: directory not empty`) — the exact leak `wrapper.go`'s `activeWG` comment
documents. Observed once under heavy self-inflicted host load (load avg ~21).

Measured pristine-vs-fixed, interleaved: **identical** (0/30 vs 0/30 isolated; 0/12 vs 0/12
at full-package `-count=10`). Fixed anyway with the file's own established pattern
(`t.Cleanup(engine.Shutdown)`).

### 7c. Assertion strengthened where the defect had forced a weakening

`TestEngine_StartSession` previously could not pin `Status` at all — it accepted any of the
five states and read even that under `state.Mu`, with a comment stating that pinning
`"pending"` "is a true data race". With the fix that is no longer true, so the assertion
now says what it always meant (`require.Equal(t, "pending", state.Status)`) and the stale
comment is corrected. Falsifiable: reverting the fix makes it FAIL
(`logs/mut_startsession.log`).

## 8. Verification

All on the final tree, exit codes captured directly (never through a pipe):

| Command | Exit |
|---|---|
| `gofmt -l` on all five touched files | 0 (nothing listed) |
| `go vet ./internal/server/... ./internal/helixqa/...` | 0 |
| `go build ./internal/... ./cmd/...` | 0 |
| `go build ./applications/terminal_ui/` (other `StartSession` caller) | 0 |
| `go test -count=1 ./internal/server/` | 0 |
| `go test -count=1 ./internal/helixqa/` | 0 |
| `go test -race -count=10 -run QASession ./internal/server/` | 0 |
| `go test -race -count=10 ./internal/helixqa/` (×3) | 0 |
| `go test -race -count=1 ./internal/server/` | 0 |

`go build ./...` is not used: it fails at pristine HEAD on missing X11/GLFW headers — a
host gap (§11.4.201), not a defect.

## 9. Honest boundaries

- The HTTP-layer guard is statistical, not deterministic. Its power is quantified
  (~3e-8 miss) but it is not a proof. The deterministic proof is the engine-level guard.
- The measured rates are specific to this host and load; they calibrate the trial count,
  they are not a claim about production frequency.
- `GET /qa/session/:id/status` and `GET /qa/sessions` still render live state, deliberately
  and correctly — reporting current status is their job. Only the `201 Created` body is
  pinned.
- `internal/server/qa_handlers.go` is byte-identical to HEAD; the temporary probes applied
  to it during investigation were reverted and verified.
