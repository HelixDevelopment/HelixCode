# HXC-211 — bounded waits in the shell executor (§11.4.83 evidence)

| | |
|---|---|
| Revision | 1 |
| Created | 2026-07-31 |
| Last modified | 2026-07-31 |
| Status | active |
| Item | HXC-211 |
| Branch | `main` |
| Base commit | `4590c638` |

## Table of contents

- [What HXC-211 is](#what-hxc-211-is)
- [Per-site status](#per-site-status)
- [Site 3 — why a contract guard and not a behavioural change](#site-3--why-a-contract-guard-and-not-a-behavioural-change)
- [Evidence index](#evidence-index)
- [RED/GREEN polarity matrix](#redgreen-polarity-matrix)
- [Paired §1.1 mutation](#paired-11-mutation)
- [Non-regression of d99ce58c](#non-regression-of-d99ce58c)
- [Twin sweep](#twin-sweep)
- [Honest gaps](#honest-gaps)

## What HXC-211 is

HXC-184 (`ExecuteStream`) and HXC-198 (`ExecuteWithProgress`) each closed one
instance of *"this call waits for something that may never happen"*. HXC-211 is
the package sweep for the survivors of that same shape.

Two defects were found in `DefaultExecutor.Execute`, and they are independent —
either one alone hangs the call.

**Defect A — the timeout's SIGKILL never landed.** `SignalHandler.Send` signals
the process *group* (`kill(-PGID)`) whenever the registered PGID is positive, and
both `Execute` and `ExecuteStream` registered a PGID that was *always* positive,
via an `if` whose branches were identical:

```go
pgid := pid
if execCmd.SysProcAttr != nil && execCmd.SysProcAttr.Setpgid {
    pgid = pid          // ← same value either way
}
```

A group whose ID equals the child's PID exists only if the child called
`setpgid`, which `os/exec` does only when `SysProcAttr.Setpgid` is set — and that
is set only by `Sandbox.applyResourceLimits`, which never runs when sandboxing is
disabled (`PermissiveConfig`). Without it the child stays in the *parent's* group,
so `kill(-childpid)` fails with `ESRCH`. The error was discarded, and the `<-done`
that follows the kill then had nothing to bound it. The timeout — the very
mechanism meant to bound the call — silently did nothing.

**Defect B — `Cmd.Wait` parked on the I/O goroutines a grandchild held open.**
`Execute` assigns a non-`*os.File` writer to `Cmd.Stdout/Stderr`, so `os/exec`
makes its own pipes and copies from them on goroutines `Wait` waits for. A pipe
reaches EOF only once *every* write end is closed, including the copies a
grandchild inherited. `Cmd.WaitDelay` was unset, so that wait had no bound.

Defect B needs no unusual configuration — only that the call cannot be cancelled,
which is the case whenever `Timeout` is 0 and the context is not cancellable.
That is reachable through the *documented* usage: `doc.go` shows
`NewDefaultExecutor(DefaultConfig())` with `context.Background()`, and
`NewDefaultExecutor` (`executor.go:229`) — unlike `NewShellExecutor` — never runs
`applyDefaults`, so a zero `Timeout` is never substituted.

## Per-site status

The original sweep listed four sites plus a no-op `if`. Status of each:

| # | Site | Status |
|---|---|---|
| 1 | `executor.go` `Execute` — `<-done` after SIGKILL | **Fixed** — `processGroupID()` reports 0 when no group exists, so `Send` falls back to signalling the PID and the kill lands |
| 2 | `executor.go` `Execute` select — unbounded when `timeout <= 0` | **Fixed** — `execCmd.WaitDelay = executeDrainGrace` (2s) bounds the post-exit drain |
| — | `executor.go` no-op `if` (both branches assign `pgid = pid`) | **Fixed** — replaced by `processGroupID()` at both call sites. It was the *cause* of site 1, not a cosmetic defect |
| 3 | `output.go` — `wg.Wait()` in `Start`, `scanner.Scan()` loop in `streamOutput` | **Contract documented + standing guard added.** No behavioural change — see next section |
| 4 | `workflow/background.go:303` — synchronous `exec(...)` | **Resolved at root by sites 1–2**; no independent defect. See [Honest gaps](#honest-gaps) |

A **third instance of defect A** that the original sweep did not list was also
found and fixed: `ExecuteStream`'s cancel path has the identical
`Send(SIGKILL)` → `<-waitDone` shape and read the *same* vacuously-registered
PGID. HXC-184's drain grace does not rescue it — that grace bounds the
*scanners*, and this wait sits before it, on the *process*. It carries its own
guard (`TestExecuteStreamTimeoutKillsTheChildWhenSandboxingIsDisabled`).

## Site 3 — why a contract guard and not a behavioural change

`output.go` holds the same unbounded shape twice, and neither instance is fixable
*there*:

1. `OutputStreamer` is handed plain `io.Reader`s, which carry no `Close`. It
   cannot release a scanner parked in `Read` without closing a descriptor it does
   not own.
2. The one production caller that *does* own them (`ExecuteStream`, via
   `streamPipes`) already closes them deliberately as part of an ordered
   teardown. A second closer would be a double close.
3. A read *timeout* inside `streamOutput` was already tried and rejected in this
   package's history. Commit `fa1b52b3` records that a total budget truncated a
   healthy but slow-draining consumer — 490/5000 lines delivered where the
   pre-fix code delivered 5000/5000 — and replaced it with a progress-aware
   no-progress window. Re-adding a per-read deadline would reintroduce exactly
   that regression class.

So the defect is not *in* `OutputStreamer`; it is an **undocumented contract**.
The bound legitimately lives with the reader owner, and `ExecuteStream`
discharges it in a specific order:

```go
streamer.Stop()
pipes.closeReadEnds()   // ← load-bearing
<-streamer.Done()       // ← unbounded, safe ONLY because of the line above
```

Drop or reorder that middle line and the wait below it is the HXC-198 hang
verbatim. The treatment is therefore to make the contract explicit and
mechanically enforced, not to change behaviour:

- **Docs** on `OutputStreamer`, `Stop`, and `Done` stating that the streamer never
  closes its readers, that `Stop` cannot reach a scanner parked in `Read`, and
  that `Done` is bounded only after the owner closes the readers.
- **Guard** `TestOutputStreamerDoneRequiresTheReaderOwnerToCloseTheReaders`,
  which fails the moment `Stop` alone becomes sufficient (the streamer took
  ownership it should not have) or closing the readers stops being sufficient
  (the release path broke).

That guard deliberately has **no RED polarity**. There is no defect here to
reproduce — it asserts an invariant that holds identically before and after
HXC-211, so a `RED_MODE` branch would have nothing to assert. It is registered as
a §11.4.135 standing guard on the strength of the invariant, not of a fixed
defect. The paired mutation below is what proves the invariant is load-bearing.

## Evidence index

| File | What it shows |
|---|---|
| `hxc211_half1_prefix_RED1.txt` | `RED_MODE=1` on the **pre-fix** artifact — all four defect tests PASS (defects reproduced) |
| `hxc211_half2_fixed_RED1.txt` | `RED_MODE=1` on the **fixed** artifact — all four FAIL (defects gone; polarity proof) |
| `hxc211_green_full.txt` | `RED_MODE=0` full package, `-count=1` |
| `hxc211_green_count3.txt` | `RED_MODE=0` full package, `-count=3` (flake check) |
| `hxc211_d99ce58c_count3.txt` | HXC-198 / `d99ce58c` guards at `-count=3` with release timings |
| `hxc211_site3_green.txt` | The new site-3 contract guard |
| `hxc211_mut_before.txt` | Paired mutation — baseline, guard PASSes |
| `hxc211_mut_after.txt` | Paired mutation — `closeReadEnds()` stripped, guard FAILs |
| `hxc211_vet.txt` | `gofmt` + `go vet` |
| `predecessor_baseline_shell.txt` | Baseline captured by a predecessor agent before it stalled |

The pre-fix artifact was reconstructed with `git archive HEAD:helix_code` into
`/tmp` rather than a worktree, so no `.git` locks were contended and the live tree
was never checked out over.

## RED/GREEN polarity matrix

`RED_MODE=1` asserts *the defect is present*; `RED_MODE=0` is the standing guard.
So a correct fix makes the RED tests **fail** — that inversion is the proof.

| Test | pre-fix, `RED=1` | fixed, `RED=1` | Polarity |
|---|---|---|---|
| `TestExecuteTimeoutKillsTheChildWhenSandboxingIsDisabled` | PASS (8.00s) | FAIL (1.01s) | holds |
| `TestKillReachesAnUnsandboxedChild` | PASS (0.02s) | FAIL (0.02s) | holds |
| `TestExecuteStreamTimeoutKillsTheChildWhenSandboxingIsDisabled` | PASS (8.06s) | FAIL (1.01s) | holds |
| `TestExecuteIsBoundedWhenAGrandchildHoldsThePipe` | PASS (6.03s) | FAIL (2.02s) | holds |

Pre-fix `RED confirmed:` lines, verbatim:

```
RED confirmed: Execute still parked after 8.000150687s despite a 1s timeout (the kill went to a process group that does not exist)
RED confirmed: Kill returned no such process (signalled a group that was never created)
RED confirmed: ExecuteStream still parked after 8.034961026s despite a 1s timeout
RED confirmed: Execute still parked after 6.029774225s although the direct child exited immediately (Cmd.Wait held by the grandchild's pipe copies)
```

**Not timing coin-flips (§11.4.201).** Every verdict clears its decision deadline
by a wide margin: 1.00s vs an 8s deadline, 0.98s vs 8s, 2.02s vs 6s. The 2.02s
figure matches the 2s `executeDrainGrace` exactly, which is the mechanism
asserting itself rather than a scheduling artefact. `TestKillReachesAnUnsandboxedChild`
is binary (`ESRCH` vs success), not timed at all. Host load during the runs was
2.27 on 64 cores — near idle — so contention is not a candidate explanation.

## Paired §1.1 mutation

The site-3 doc and guard both rest on the claim that `pipes.closeReadEnds()` is
load-bearing. That claim is proven by removing it:

| | Result |
|---|---|
| Unmutated | `TestExecuteStreamReturnsWhenGrandchildHoldsPipes` **PASS** (4.49s) |
| `pipes.closeReadEnds()` stripped from the drain teardown | **FAIL** — *"ExecuteStream must return within the drain grace period even though a grandchild still holds the pipe write ends open; it was still blocked after 6.098359406s"* |

The mutation was applied to an **isolated `/tmp` copy** of the fixed tree
(`git archive HEAD:helix_code` + the three modified files copied over), never to
the live tree, so no residue could survive it. Post-run verification on the live
tree: a grep for the four §11.4.84 residue markers (as enumerated by the
`pre-commit` hook's own pattern) returns empty, `pipes.closeReadEnds()` is
present at its call site, and the `Stop()` → `closeReadEnds()` → `<-Done()`
ordering is intact. The isolated copy was then deleted.

The markers are referred to here by description rather than quoted verbatim,
because the `pre-commit` hook scans *staged content* for those literals and
would — correctly — refuse a commit that embedded them, including one that
embedded them only as documentation.

## Non-regression of d99ce58c

`d99ce58c` (HXC-198, `ExecuteWithProgress`) and its predecessor `fa1b52b3` are
untouched by this change. `fa1b52b3` specifically replaced a *total budget* with a
*progress-aware no-progress window*, and that rejected design is not
reintroduced here: `executeDrainGrace` is a total budget, but it applies to a
different consumer and the difference is deliberate and documented at its
declaration. `ExecuteStream` hands lines to a caller that may legitimately read
slowly and that holds `Cancel`; `Execute` collects into an in-memory
`OutputCollector` that cannot be slow, and its caller is blocked inside the
function with no `Cancel` handle. Rearming for a still-producing descendant there
would let a daemon hold the call open forever — the exact hang the bound exists to
prevent.

Evidence: `hxc211_d99ce58c_count3.txt`.

## Twin sweep

**The vacuous-PGID pattern (defect A) has no twins in the module.** The two other
places that set `Setpgid` — `internal/hooks/shell_runner_unix.go:14` and
`internal/mcp/transport_stdio_unix.go:12` — set it *unconditionally*, so their
`killProcessGroup` always has a real group to signal. Only the shell package had
the conditional-sandbox path where the flag may or may not be applied.

**Every remaining blocking wait in the shell package is accounted for:** bounded
by a select on a timer or context, bounded by `WaitDelay`, or unbounded-by-design
with the bound correctly placed at the reader owner and now documented.

**The broader "unbounded wait on a caller-supplied closure" pattern does have one
genuine structural twin outside this package**, found while analysing site 4:
`internal/hooks`. `hook.go:171-179` applies `context.WithTimeout` only when
`h.Timeout > 0`, then calls the externally-supplied `h.Handler(ctx, event)`;
`executor.go:161-189` runs it under `e.wg.Add(1)` / `defer e.wg.Done()`, and
`executor.go:141-143` `Wait()` blocks on that WaitGroup with **no** bound. It is
better than site 4 in offering an optional per-hook timeout and worse in having
no escape equivalent to `Close()`'s 5s. Out of scope for HXC-211 — recorded here
so it is not rediscovered from scratch.

Checked and rejected as false positives: `registry.go:1443` (bounded semaphore,
internal callee), `worker_pool.go:370`, `base_agent.go:1379`, `mcp/registry.go`,
`notification/queue.go`, `context_manager.go:253`, `verifier/poller.go:55` — all
wait on internal concrete functions. `event/bus.go:267` is a partial match (it
does invoke an external `EventHandler`) but its waiter is the caller's own
synchronous call rather than a lifecycle drain, so the slot-exhaustion
consequence does not arise.

## Honest gaps

1. **Site 4 (`workflow/background.go:303`) received no code change.** The sweep
   described it as having "no timeout, no abandonment path". The second half of
   that is wrong: `BackgroundManager.Close()` (`background.go:378-404`) *does*
   cancel every task's context and *does* bound its own wait at 5s
   (`:398-402`), so the caller is not hung. The call chain is
   `bm.run` → `exec(task.ctx, …)` → `adaptToolForBackground` → `tool.Execute(ctx, …)`,
   which passes the context straight through — so site 4 *amplified* the shell
   executor's unboundedness rather than generating its own, and cancelling
   `task.ctx` now genuinely kills the child.

   Traced residual for a hypothetical ctx-ignoring `BackgroundExecutor`, worst
   consequence first: the in-flight slot is never freed (`countInFlightLocked`,
   `:407-415`), so after `MaxConcurrent` (default 64) such tasks `StartTask`
   returns `ErrTooManyTasks` permanently — the manager is bricked for new work;
   then unbounded map growth (`sweep` skips non-terminal tasks, `:438-440`); then
   the leaked goroutine; then `Close()` and `StopTask` both returning `nil` for a
   stop that did not stop.

   A select-on-ctx wrapper at `:303` is nonetheless the wrong remedy, and not
   only because Go cannot reclaim the parked goroutine. The abandoned goroutine
   still holds `task.AppendOutput` as its sink, so letting `run` return would let
   it write into the ring of a task the manager has marked terminal and may
   already have swept — a write-after-terminal race the current shape does not
   have. Making that safe needs a sink kill-switch, i.e. a design change.
   Decisively, this codebase has already ruled on the *same* shape with the roles
   swapped: `shell/background.go:32-44` documents that a `sink` which never
   returns parks `ExecuteWithProgress`, and resolves it as a documented contract
   for future callers rather than a wrapper. Applying a different remedy to the
   same shape in the same change would be inconsistent.

   Existing coverage, for the record: three tests use a ctx-*respecting* blocking
   closure (`background_test.go:113-128`, `:244-260`,
   `workflow_chaos_test.go:106-159`) and two use a ctx-*ignoring* one
   (`background_test.go:182-198`, `:215-233`). None covers the *intersection* —
   a ctx-ignoring closure still parked when `Close()` runs — so `Close()`'s 5s
   branch is untested (`"drain timeout"` appears only at `background.go:401`).

2. **`UNCONFIRMED:` one audit could flip site 4 from theoretical to live.**
   Background dispatch is not gated by tool — any tool can be sent
   `run_in_background: true` — and every registered tool that is *not*
   `BackgroundAware` takes the fallback at `registry.go:1225-1231`, which calls
   `tool.Execute(ctx, args)` directly. Whether any in-tree non-`BackgroundAware`
   tool's `Execute` blocks unboundedly *while ignoring ctx* was not determined.
   Settled by enumerating registered `Tool` implementations lacking
   `BackgroundAware` and checking each `Execute`. If one exists, site 4 stops
   being latent and warrants code. Also unresolved: whether
   `planmode.PlanModeWorkflow.ExecuteWithProgress`
   (`internal/workflow/planmode/planmode.go:215`) is a registered tool.

3. **`docs/workable_items.db` was not updated in this commit.** The DB is the
   §11.4.93/§11.4.95 single source of truth for item status, but it was already
   modified in the working tree by a concurrent agent working a different item.
   Staging it would have swept that agent's changes into this commit, which
   §11.4.84 forbids. The HXC-211 row therefore still needs closing separately.

4. **`OutputIncomplete` is "not detected", not "proven complete".** `os/exec`
   reports a `WaitDelay` expiry only when the command had no other error, so a
   command that *both* exits non-zero *and* has its drain cut short reports the
   exit status alone and leaves the flag false. `Execute` cannot distinguish that
   case because `os/exec` owns those pipes and does not expose whether its copy
   goroutines finished. This is documented at the field rather than papered over.

5. **`shell_test.go` is not `gofmt`-clean.** It is a pre-existing file untouched
   by this change and is left alone deliberately; reformatting it would have put
   unrelated churn in this commit.
