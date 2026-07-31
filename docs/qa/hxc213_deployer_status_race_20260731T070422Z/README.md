# HXC-213 — ProductionDeployer status data race + status aliasing

| Field | Value |
|---|---|
| Revision | 1 |
| Created | 2026-07-31 |
| Last modified | 2026-07-31 |
| Status | active |
| Run ID | `hxc213_deployer_status_race_20260731T070422Z` |
| Branch | `main` |
| Pre-fix baseline | `ac9e9e4d186a0f12989842651b26013acaea00d5` |
| Fixed source sha256 | `77a64071b87366554d326ee217221961c6053b0581f0c49fb5b393435145d1bb` |

## Table of contents

- [Defect](#defect)
- [Files](#files)
- [RED baseline on the broken artifact](#red-baseline-on-the-broken-artifact)
- [GREEN on the fixed artifact](#green-on-the-fixed-artifact)
- [Paired mutations](#paired-mutations)
- [Aliasing blast-radius check](#aliasing-blast-radius-check)
- [Sibling sweep](#sibling-sweep)
- [Transcript filtering](#transcript-filtering)
- [Honest gaps](#honest-gaps)

## Defect

`ProductionDeployer` declared a `sync.RWMutex` at `production_deployer.go:28` and held it
at exactly **one** site (`addNotification`) against 92 `pd.status.*` accesses across the
type. That single site is the residue of the **HXC-014** fix, which guarded only the two
lines the race detector happened to point at — the report is recorded verbatim in the
comment at `production_deployer.go:1179-1186` (pre-fix `:1026-1027`) — and left every
other access untouched.

This is worse than carrying no lock at all: the file *reads* as synchronized, and the
package's own concurrent tests (`production_deployer_chaos_test.go:132-135`) already
follow the advertised protocol (take `pd.mutex.RLock()` to read `pd.status`), a discipline
that is only sound if **every** writer takes the lock. It did not.

Two independent halves:

- **(A) Lock coverage** — a reader honouring the type's own `RLock` protocol races with
  any concurrent unguarded writer.
- **(B) Aliasing** — `StartProductionDeployment` returned `pd.status`, the **live**
  internal pointer, and is the type's only exported method, so every caller received a
  handle into deployer state that a subsequent deployment keeps writing.

The fix converts the type to an accessor-based discipline (documented at
`production_deployer.go:32-45`): no field of `status` is touched outside a critical
section, the lock is never held across blocking I/O or `log.Printf` (three-phase
snapshot → work → single-write-back), and the exported method hands back a deep copy
via `copyStatusLocked` / `snapshotStatus`.

## Files

| Path | Role |
|---|---|
| `helix_code/internal/deployment/production_deployer.go` | the fix (+371 / −83) |
| `helix_code/internal/deployment/production_deployer_hxc213_race_test.go` | §11.4.115 polarity guards + §11.4.107(10) golden-bad fixture |

## RED baseline on the broken artifact

§11.4.115 requires the RED test to reproduce the defect on the **actual pre-fix
artifact**, not a synthetic failure. The artifact was reconstructed with
`git archive HEAD helix_code` (chosen over `git worktree` to avoid `.git` lock
contention with concurrently-running agents, §11.4.179/§11.4.180) and confirmed to carry
**2** `pd.mutex.` sites — the HXC-014 partial fix — with only the new test file overlaid.

| Evidence | Result |
|---|---|
| `red01_lock_coverage_PREFIX.txt` | exit 1, **23 × `WARNING: DATA RACE`**; write `failDeployment` @ `production_deployer.go:1096` vs read under `RLock` @ test `:122` |
| `red02_aliasing_PREFIX.txt` | exit 0 — `RED_MODE` assertion held: returned status **is** the live internal pointer |
| `red03_goldenbad_analyzer_selfvalidation.txt` | exit 1, **2 × `DATA RACE`** |

`red03` is the §11.4.107(10) analyzer self-validation. Its purpose is to make a green run
of the other guards mean *"no race"* rather than *"blind harness"*: it performs the pre-fix
write shape and the detector **must** report it. It trips, so the harness demonstrably sees
races. It is `SKIP`-ped in the standing suite (`RED_MODE=0`) so the guard set stays green.

## GREEN on the fixed artifact

| Evidence | Result |
|---|---|
| `green_full_package_race.txt` | `ok dev.helix.code/internal/deployment 15.726s` — whole package, `-race`, exit 0 (no sibling test regressed) |
| `green_hxc213_count3.txt` | exit 0, `-count=3`, **0 races**, 6 × PASS + 3 × SKIP (golden-bad), deterministic per §11.4.50 |

## Paired mutations

Both mutations were applied to an **isolated** base (`git archive HEAD` + exactly the two
fixed files, verified byte-identical to the live tree by sha256), never to the live working
tree. Two agents left live mutation residue in this checkout earlier today by mutating
in-place; isolation makes that class of §11.4.84 accident structurally impossible here.
Restoration is a re-copy from the live tree — the source of truth — re-verified by
checksum, not from a stale file backup.

| # | Axis | Mutation | Guard result |
|---|---|---|---|
| 1 | locking | stripped `pd.mutex.Lock()/defer Unlock()` from `markFailed` | **FAIL** (exit 1) — `mut01_lock_removed_MUSTFAIL.txt`, **37 × DATA RACE**, writer `markFailed` |
| 2 | aliasing | `copyStatusLocked` hands out `pd.status.Metrics` instead of cloning it | **FAIL** (exit 1) — `mut02_shallow_metrics_MUSTFAIL.txt` |

Mutation 2 deliberately breaks the deep-copy **depth** rather than swapping in the whole
live pointer. A weaker guard that only asserted `status != pd.status` would still pass a
shallow copy; this one names the exact regression:

> `returned status shares the *DeploymentMetrics pointer with the deployer; a shallow copy is not enough — Metrics must be deep-copied`

Restore proofs: `mut01_restored_GREEN.txt`, `mut02_restored_GREEN.txt` — both exit 0, both
with the restored file at sha256 `77a640…5d1bb`. `mut00_isolated_base_GREEN.txt` proves the
isolated base was GREEN before any mutation, so each FAIL is attributable to the mutation.

## Aliasing blast-radius check

The **HXC-205** lesson applies directly: deep-copying alone can turn a feature into a
silent no-op that still looks green, if a caller depends on writing back through the
aliased pointer. Checked, and it does not apply here — the only non-test caller of
`StartProductionDeployment` is the read-only usage example in `doc.go:24`, which binds the
result and never writes through it. In-package tests reach `pd.status` directly and are
unaffected; the whole-package `-race` run confirms none regressed.

## Sibling sweep

Enumerated scope: every non-test `*.go` under `helix_code/internal/deployment/` including
the `i18n/` subdirectory. The package declares exactly **two** mutexes:

| Site | Verdict |
|---|---|
| `production_deployer.go:46` | the defect — fixed here (2 → 25 lock acquisitions) |
| `translator.go:35` | **not a defect.** Package-level `translatorMu` guarding the `translator` var; every read and write is covered (`:44-45`, `:59-61`, `:64-69`). An earlier regex flagged it only because it is bare rather than receiver-prefixed. |

**No sibling hits.** Per §11.4.118 this is an enumerated negative, not an absence of
looking — but it is scoped to `internal/deployment/` as tasked, and says nothing about
the same shape elsewhere in the module.

## Transcript filtering

The raw transcripts totalled 3.9 MB — 40% of the size of this repo's entire existing
400-file `docs/qa` corpus — because the deployer emits ~13 `log.Printf` lines per
`failDeployment` call and the lock-coverage guard makes 900 of those per run (2703
byte-identical repetitions per file).

Each `.txt` carries a provenance header stating its original line/byte count. Only lines
matching the Go log prefix `^YYYY/MM/DD HH:MM:SS ` were elided, with the first 25 retained
as a sample. **Every race report and every test-framework verdict line is preserved
verbatim** — the filter aborts if a race count changes, and the post-filter counts were
re-verified against the originals (37 / 23 / 2 races and all PASS/FAIL/SKIP/ok lines
identical). Result: 340 KB.

## Honest gaps

- The race detector is a **sampling** oracle: a green `-race` run is strong evidence, not
  proof of absence. Half (A) is therefore backed by a second, detector-independent
  assertion — a lost-update check on `FailedPhases` (`want writers*iters` entries) that
  catches a torn read-modify-write even in a run where the detector missed the window.
  Half (B) is fully detector-independent by construction (it proves ownership by mutation).
- Runs were captured on a host at load average ~100 across 64 cores (concurrent agents).
  The guards assert on the detector and on count consistency, never on wall-clock timing,
  so contention cannot produce a false verdict in either direction (§11.4.201). Timings in
  the logs are correspondingly inflated and are not a performance claim.
- This covers `internal/deployment/` only. No claim is made about unguarded-mutex shapes
  in other packages.
