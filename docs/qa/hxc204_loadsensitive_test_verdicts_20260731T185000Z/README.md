# HXC-204 — two endurance tests reported failure whenever the machine was busy

**Status: PARTIAL — landed, not finished.** Both root causes are FACT and both
fixes are in. One is verified stable; the other is verified *correct* but NOT
verified *stable*, and that gap is stated below rather than papered over.

## The two mechanisms — different, as suspected

They do not fail for the same reason.

### 1. `TestManager_Stress_ConcurrentReadWrite` — a wall-clock deadline

- Site: `helix_code/tests/stresschaos/stresschaos.go:428` (pre-fix) —
  `case <-time.After(timeout): deadlock = true`, then `:462`
  `t.Fatalf("... DEADLOCK — %d goroutines did not finish within %s")`.
- The harness had exactly one signal — *did the run finish inside the budget?* —
  and reported the overrun as `DEADLOCK`. A deadlock and a busy host produce the
  identical signal, so the verdict was about the host, not the code
  (§11.4.201: a guard must assert the condition it names).
- Captured RED (`red/red_memory_concurrent_readwrite_under_load.log`), 192 busy
  workers on 64 CPUs: `FAIL ... DEADLOCK — 16 goroutines did not finish within 25s`.
  Its own evidence file contradicts the accusation:
  `{"deadlock": true, "error_count": 0, "duration_ms": 25001}` — 1920 calls were
  proceeding without a single error; the run simply had not finished.
- Isolation (`red/isolation_memory_race_baseline.log`): **7.68s** against a 25s
  budget under `-race`. Whether a fixed budget is cleared is a property of the
  HOST, not of the code under test.

### 2. `TestConsensus_Chaos_DropVoteFraction` — an instantaneous sample of a steady-state property

- Site: `helix_code/internal/worker/consensus_stress_chaos_test.go:171` (pre-fix),
  in `noNodeStuckCandidate`, called from `:374`.
- It read each node's state **once** and failed if that single sample said
  `Candidate`. But `Candidate` is a legitimate, healthy, transient Raft state.
  Livelock is a STEADY-STATE property; one sample cannot tell "wedged" from
  "mid-election".
- Captured RED (`red/red_consensus_dropvote_under_load.log`) is unambiguous —
  the test declared node-0 livelocked and node-0 won the very next term:

```
Node node-0 starting election for term 2 (peers=2)
    Error: Should not be: 1
    Messages: node node-0 stuck as Candidate (livelock)
Node node-0 won election for term 2 (2/3 votes, quorum=2)
Node node-0 became leader for term 2
```

  Note it failed in **0.20s** — this one is not a slow-machine problem at all.
  It is a pure race; load merely widens the window in which the sample lands
  mid-election.

## What was done to each

**#2 made deterministic.** `noNodeStuckCandidate` now samples the property it
names: a node is stuck only if observed `Candidate` on *every* sample across a
3s settle window, never once seen Leader or Follower. Robust to the continuous
election churn a 1/3 vote-drop creates (at any instant *some* node is
legitimately Candidate). The window is sized from the fixture's own 8ms
heartbeat / ~232ms election re-arm, not from taste. **Polarity unchanged and it
still bites: a node that never leaves Candidate fails on any host, however fast.**
Fixes all four call sites (`:339 :374 :413 :490`), not just the named one.

**#1 made honestly-inconclusive — and deadlock detection made sharper.** The
budget no longer detects deadlock. Forward progress does:
`concurrentWorker` counts completed iterations, and the classifier reads that
plus a goroutine-state census (`workerStateCensus`) taken from a real stack dump:

| verdict | condition | outcome |
|---|---|---|
| `completed` | every iteration finished | PASS — **no wall-clock condition at all** |
| `deadlock` | progress stalled AND every live worker parked in a sync primitive | FAIL, naming the observed condition |
| `inconclusive` | did not finish, evidence contradicts or is silent on deadlock | SKIP-with-reason (§11.4.3) |

Because deadlock is caught on the *stall* rather than the ceiling, it fires
sooner than the old timeout, and the ceiling can safely stretch
(`maxExtensionFactor = 4`) while work is demonstrably completing — a hung run
cannot reach the extension. This is the HXC-215 classifier shape: a real defect
FAILs and names itself; an inconclusive run names nobody. Applies to all 73
`RunConcurrent` call sites.

## Proof the assertions still bite (§1.1)

`green/meta_tests_deadlock_still_detected.log` — the pre-existing paired mutation
plants a genuinely deadlocking fn (`<-block`, never closed) and asserts the
harness catches it. Run **under load (252 runnable tasks)**:

```
--- PASS: TestMeta_RunConcurrent_DetectsDeadlock (0.50s)
--- PASS: TestMeta_RunConcurrent_DetectsGoroutineLeak (0.80s)
ok  dev.helix.code/tests/stresschaos  1.409s
```

A real deadlock is still detected, and now in 0.50s. The fix did not buy hung
code any room to hide.

## What is NOT done — honest gap (§11.4.6)

**The memory test's verdict is correct but NOT stable, so HXC-204's acceptance
criterion is not met and the item must stay open.** Five loaded runs
(160 busy workers) gave **1 PASS, 3 SKIP(inconclusive)**, and the 5th was cut off
by my own 10-minute command timeout:

```
RUN1 SKIP  (100.20s)  INCONCLUSIVE — run still progressing at the 1m40s ceiling
RUN2 PASS  (96.01s)
RUN3 SKIP  (100.20s)  INCONCLUSIVE — run still progressing at the 1m40s ceiling
RUN4 SKIP  (100.01s)  INCONCLUSIVE — run still progressing at the 1m40s ceiling
RUN5 (not captured — command timeout)
```

Every verdict is honest — **zero false DEADLOCK accusations across all runs**,
which is the defect this ticket names, and that much IS fixed. But PASS/SKIP
alternating is not a stable verdict. `maxExtensionFactor = 4` was sized on a
measured 3x-oversubscription figure (34.6s); this load is heavier and the run
needs >100s. The remaining work is to size the ceiling from a real measured
distribution rather than a single point, or to drop the per-run work so the
16x120 matrix is not this expensive under contention.

Not attempted, and worth a look next: `consensus_stress_chaos_test.go:384`
`require.Positive(t, ft.stats()["votes_dropped"])` is an anti-vacuity floor of
the same class as the HXC-173 renderer-overlap floor — if it ever fires it
should be inconclusive, not FAIL. It did not fire in any run here, so it is a
theoretical concern, not an observed one.

## Reproducing

`loadgen.sh` is the ownership-tagged load generator (§11.4.174: every worker
carries `HXC204LOADMARK` and `stop` kills only marked processes; it refuses to
spawn when the cgroup `pids.max` budget is short, which it did — correctly —
when other agents had the host at 3652/4096).

```
./loadgen.sh start 160
go test -race -count=1 -run TestManager_Stress_ConcurrentReadWrite ./internal/memory/ -v
go test -race -count=1 -run TestConsensus_Chaos_DropVoteFraction ./internal/worker/ -v
./loadgen.sh stop
```

Note the binding process ceiling on this host is the cgroup `pids.max` = 4096 at
the project session slice, **not** `ulimit -u` (262144).
