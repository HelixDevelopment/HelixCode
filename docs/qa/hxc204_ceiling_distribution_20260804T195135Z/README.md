# HXC-204 — sizing the ceiling from a measured distribution

**Status: PARTIAL — root cause measured and fixed, acceptance criterion NOT yet met.**
The ≥5-runs-under-load verification of the changed tests has **not been run in this
session**. HXC-204 stays OPEN. What follows is what was actually measured.

## What the previous pass got wrong, and how

The prior sizing of `maxExtensionFactor = 4` rested on ONE data point — *"34.6s under
3x CPU oversubscription (192 busy workers on 64 CPUs)"* — and concluded 100s covered
it with ~3x headroom. Re-measured with a distribution instead of a point, the same
16x120 matrix takes **108.5–121.6s** under load. The 100s ceiling sat **below the
entire distribution**, which is precisely why the verdict alternated PASS/SKIP: the
verdict was reporting host load, not code.

Two facts the original sizing missed:

1. **The cgroup CPU quota, not the host CPU count, is binding (§11.4.225).** The
   project session slice has `cpu.max = 860000 100000` — **8.6 CPUs, not the host's
   64** — and was measurably throttling: `nr_throttled 54991` of `nr_periods 192813`
   (28.5% of periods), `throttled_usec 86494719809`. So "192 workers on 64 CPUs" was
   never 3x oversubscription; it was ~20x oversubscription *of an 8.6-CPU quota*.
   This also explains the previously-puzzling observation that pinning with `taskset`
   to 2 CPUs made a run **faster**: fewer runnable threads burning one shared quota
   means fewer throttle stalls.
2. **Allocation-heavy race-instrumented work inflates ~29x under that pressure**
   (3.88s quiet -> 114s loaded), far more than any CPU-share ratio predicts, because
   GC assist and race shadow-memory traffic starve hardest.

## The real cost driver: the workload is QUADRATIC

Every iteration creates one more conversation and never removes it, while `GetAll` /
`Search` / `GetStatistics` each scan — and `GetAll`/`Search` *clone* — the entire
store. Per-call cost therefore grows linearly with the iteration index and total cost
as the square. Measured under `-race` (`scaling_measure_test.go.txt`):

| iters | calls | duration | per call | ratio vs prev |
|------:|------:|---------:|---------:|--------------:|
| 15  | 240  | 79 ms   | 333 us  | —     |
| 30  | 480  | 254 ms  | 531 us  | 3.22x |
| 60  | 960  | 1019 ms | 1062 us | 4.01x |
| 120 | 1920 | 3880 ms | 2021 us | 3.81x |

Doubling iterations multiplies cost by ~4x, and per-call cost doubles each step —
textbook O(n^2). The extra cost is memcpy under an RLock, **not** additional
lock-contention coverage.

## Measured distribution under load (the deliverable)

Uncapped (the real `RunConcurrent` ceiling truncates observation and cannot be used
to measure the thing it bounds), `-race`, 160 competing workers, **n=8 per matrix**.
Raw log: `red/uncapped_duration_distribution.log`.

| matrix | calls | min | median | mean | max | spread |
|--------|------:|----:|-------:|-----:|----:|-------:|
| 16x120 | 1920 | 108,506 | 114,747 | 114,138 | 121,609 ms | 1.12x |
| 16x60  | 960  | 19,112  | 20,997  | 21,275  | 25,207 ms  | 1.32x |
| 16x40  | 640  | 10,402  | 11,109  | 11,337  | 12,990 ms  | 1.25x |

All 24 samples verbatim in the log. The distributions are tight (spread ≤1.32x),
which is what makes them usable for sizing.

## The fix, and why this shape

`IterationsPerGoroutine` 120 -> **40** in `TestManager_Stress_ConcurrentReadWrite`.

At 16x40 the whole loaded distribution (10.4–13.0s) fits inside the **25s base
budget** with ~1.9x margin and never reaches the extension at all — versus 16x120,
where every loaded run overran even the 100s ceiling. Raising the ceiling instead was
rejected: `maxExtensionFactor` is global to all 73 `RunConcurrent` call sites, no
measurements of the other 72 exist, and a ceiling is a fixed point on an unbounded
axis — load can always grow past it, whereas a run that fits its base budget is
stable under any load that budget already tolerates.

`maxExtensionFactor` therefore **stays at 4**, with its comment corrected: it had
asserted the discredited 34.6s figure and claimed "3x headroom" that did not exist.
Inventing a larger value from one caller's data would repeat the original error.

**Coverage is preserved, not traded away.** 640 real concurrent calls still clears the
§11.4.85(A) sustained floor (N>=100); parallelism stays 16, clearing the contention
floor (N>=10); and the sustained-load axis has its own dedicated coverage in
`TestManager_Stress_SustainedCreateAddGet`.

## Honest limits (§11.4.6)

- **The acceptance criterion is NOT met.** No verification runs of the changed tests
  were executed this session — neither the 5 loaded runs of
  `TestManager_Stress_ConcurrentReadWrite` at 16x40, nor the 5 loaded runs of
  `TestConsensus_Chaos_DropVoteFraction` (whose fix remains proven-correct but
  never verified stable under load). The 16x40 numbers above are the *raw workload*,
  which excludes `RunConcurrent` overhead (GC settle, goroutine census, report
  write) — measured at roughly +3.8s quiet at 16x120, and itself load-inflated.
  Predicting a PASS from raw-workload timing is inference, not evidence.
- **No mutation proof was captured this session** that the changed assertions still
  bite. The pre-existing paired mutation
  (`TestMeta_RunConcurrent_DetectsDeadlock`) was not re-run against these edits.
- **Ambient-load attribution is uncertain.** An orphaned runaway process was reaped
  by the operator around the start of this session (host load 31 -> 2.8). This
  session's own first reading was 18.86, then 6.08 shortly after, so I **cannot
  determine** which side of that reaping the quiet-host scaling curve (79/254/1019/
  3880 ms) fell on, and I am not going to guess. Two mitigations apply: the
  *quadratic* conclusion rests on the RATIO between those points (~4x per doubling),
  which is invariant to a constant ambient load factor; and every loaded number was
  taken under this session's own controlled 160-worker load, which dominates ambient
  variation — corroborated by the ≤1.32x spreads. The figure most exposed to this
  uncertainty is the absolute quiet-host baseline (3.88s), and hence the derived
  "~29x inflation", which should be read as an order-of-magnitude finding.

## Reproducing

```bash
./measure_dist.sh docs/qa/hxc204_ceiling_distribution_20260804T195135Z 160 8
# scaling_measure_test.go.txt is the instrument: drop it into
# helix_code/internal/memory/ as a _test.go, then
#   HXC204_SCALING=1 HXC204_ITERS=120 HXC204_REPEATS=8 \
#     go test -tags=nogui -race -count=1 -timeout 3600s \
#     -run TestHXC204_ScalingCurve ./internal/memory/ -v
# It is kept OUT of the package deliberately: it is a measurement instrument,
# not a test, and it must not ship in a normal `go test ./...`.
```

`loadgen.sh` is the ownership-tagged load generator carried over unchanged
(§11.4.174: every worker carries `HXC204LOADMARK`; `stop` kills only marked
processes; `start` refuses to spawn when the cgroup `pids.max` budget is short).

## What remains

1. Run `TestManager_Stress_ConcurrentReadWrite` >=5x under load at 16x40; require a
   stable verdict (expected PASS, unverified).
2. Run `TestConsensus_Chaos_DropVoteFraction` >=5x under load.
3. Re-run the paired mutation to prove both assertions still bite.
4. Optional, still untouched: `consensus_stress_chaos_test.go` `require.Positive(
   ft.stats()["votes_dropped"])` is an anti-vacuity floor that should report
   inconclusive rather than FAIL if it ever fires. It has never fired.
