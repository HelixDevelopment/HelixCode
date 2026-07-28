# HXC-174 — QA dashboard read a live QA session while the orchestrator wrote it

| | |
|---|---|
| Revision | 1 |
| Created | 2026-07-29 |
| Last modified | 2026-07-29 |
| Status | active |
| Item | HXC-174 (Bug / High) |
| Fix commit | `2775493` |
| Pre-fix artifact | `a7d8dbb5` (parent of the fix) |
| Status summary | Fixed — RED reproduced on the broken artifact, GREEN after the fix, harness self-validated |

## Table of contents

- [What was wrong](#what-was-wrong)
- [The fix](#the-fix)
- [Evidence](#evidence)
- [Honest boundaries](#honest-boundaries)

## What was wrong

`applications/terminal_ui/main.go` → `showQA()` rendered the QA dashboard straight off
the **live** `*SessionState` pointers returned by `helixqa.Engine.ListSessions()`,
reading `Status` / `Phase` / `PhaseProgress` / `EndTime` / `StartTime` / `Platforms` /
`Banks` with no lock held — while the orchestrator goroutine spawned by
`Engine.StartSession` writes exactly those fields under `state.Mu.Lock()`.

A lock held on one side alone establishes no happens-before edge with an
unsynchronized access on the other, so this was a genuine **data race**, not merely a
race condition. The sharpest instance was the duration cell:

```go
if s.EndTime != nil { s.EndTime.Sub(s.StartTime) }
```

a check-then-use against a pointer the writer publishes, so the reader could observe a
non-nil pointer to a `time.Time` whose bytes were not yet fully published.

Every other consumer already went through the lock — `internal/server/qa_handlers.go`
serialises sessions with `json.Marshal`, which `SessionState.MarshalJSON` guards with
`s.Mu.RLock`. The TUI was the sole unguarded reader in the tree. Pre-existing; not
introduced by recent work.

## The fix

- `internal/helixqa/wrapper.go`: added `Engine.ListSessionSnapshots()`, built on the
  existing HXC-154 `creationSnapshot` point-in-time detach (copies under `s.Mu.RLock`,
  own zero mutex, nil `CancelFunc`, `Platforms`/`Banks` copied rather than aliased).
  `ListSessions` keeps returning live pointers for identity-preserving callers, and its
  contract is now documented so the next caller does not repeat the mistake.
- `applications/terminal_ui/main.go`: both the session table and the stats tally now
  render from `ListSessionSnapshots()`.

Guard: `applications/terminal_ui/qa_dashboard_race_test.go` drives the **real production
`showQA`** against live sessions. An overlap oracle brackets each batch of renders with a
lock-guarded status vector, so a clean `-race` result cannot come from a window in which
the writer never ran. Sampling is per *batch*, not per render, to keep the test's own
`s.Mu` acquisitions rare enough not to mask the race under test. An unmet overlap floor
`SKIP`s with a reason per §11.4.3 rather than failing.

## Evidence

All runs need `-tags=ci` (Fyne's non-GL driver; this host has no X11/GL headers). Host
load is stated with every measurement per §11.4.201.

| Capture | File | Outcome | Exit | Load |
|---|---|---|---|---|
| RED on the pre-fix artifact | [`evidence/red_prefix_artifact.txt`](evidence/red_prefix_artifact.txt) | `WARNING: DATA RACE` ×7, `--- FAIL` | 1 | 17.58 |
| GREEN post-fix, default polarity | [`evidence/green_default_polarity.txt`](evidence/green_default_polarity.txt) | `--- PASS`, overlap proven | 0 | see file |
| Golden-bad harness self-validation | [`evidence/goldenbad_red_mode.txt`](evidence/goldenbad_red_mode.txt) | `DATA RACE` in `readSessionsUnguarded`, `--- FAIL` | 1 | see file |
| Regression, touched packages | [`evidence/regression_touched_packages.txt`](evidence/regression_touched_packages.txt) | all `ok` | 0 | see file |
| §1.1 paired mutation — skip branch | [`evidence/paired_mutation_skip_branch.txt`](evidence/paired_mutation_skip_branch.txt) | `--- SKIP` with reason (not FAIL), then `--- PASS` after byte-identical restore | 0 | see file |

The RED run's racy read sites — 2745, 2756, 2757, 2763, 2766, 2767, 2787 — are exactly
the line set cited on the item, confirmed **independently by the race detector** rather
than by re-reading the citation.

The `RED_MODE=1` golden-bad fixture (§11.4.107(10)) replays the legacy unguarded read
shape and still reports a data race **after** the fix. That is what proves the green
result above means something: the guard is not blind to this defect class.

The default-polarity guard was additionally stable at `-count=5` (exit 0).

A §1.1 paired mutation raises the overlap floor out of reach (1 → 1000000, deadline 30s → 3s)
and confirms the inconclusive branch really `SKIP`s with a reason and exit 0 rather than
failing — HXC-173's defect applied to this guard itself. The guard is then restored
byte-identically (`git diff` empty against the committed file) and PASSes again, so no
mutation residue persists per §11.4.84.

## Honest boundaries

- `main.go:2765` (`s.ID`), `:2769` (`s.Platforms`), `:2770` (`s.Banks`) were cited on the
  item and were unguarded reads, but the detector did **not** report them racing, and it
  is correct not to: those fields are written once at session creation and never mutated
  by the orchestrator goroutine, so no concurrent write exists to race with. The fix
  snapshots them anyway, for a consistent per-row view.
- Snapshots are consistent **per session**, not across the set — sessions are snapshotted
  one after another, so two rows in one render may reflect slightly different instants.
  That is the right granularity for a list; no caller needs a globally atomic view.
- `ListSessionSnapshots` inherits `ListSessions`' unspecified (Go map) ordering, so
  dashboard rows can still reorder between refreshes. That is a pre-existing UX wart,
  untouched here because changing it is a behaviour change outside this item's scope.
- This closes the race at the reader. It does not prove the QA dashboard is correct in
  any other respect.
