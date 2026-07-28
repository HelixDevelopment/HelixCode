# What this run certifies

`27754934` closed HXC-174. `showQA` rendered straight off the LIVE
`*SessionState` pointers returned by `Engine.ListSessions()`, reading Status /
Phase / PhaseProgress / EndTime / StartTime / Platforms / Banks with no lock
held, while the orchestrator goroutine spawned by `StartSession` writes exactly
those fields under `state.Mu`. A lock held on one side alone establishes no
happens-before edge, so this was a genuine data race rather than merely a race
condition — its sharpest instance being the duration cell's check-then-use on
`EndTime`, which could observe a non-nil pointer to a not-yet-published value.
The TUI was the only unguarded reader in the tree; every other consumer already
went through the lock.

Captured here (real command execution; `transcripts/` holds every byte):

* the detached accessor and the rerouted render path are present in the tracked
  source that ships today, and the live-pointer read is provably GONE from the
  dashboard (`git grep` finds no surviving occurrence);
* the commit's own guard, which drives the REAL production `showQA` against
  live sessions, ran under `-race` with `-tags=ci`;
* RED_MODE=1 re-drives the legacy unguarded pattern on this same fixed artifact
  and is asserted BOTH to fail AND to carry a `DATA RACE` detector report —
  that second assertion is what makes it a golden-bad self-validation
  (§11.4.107(10)) rather than a test agreeing with itself: it proves the
  harness still SEES this defect shape and is not blind;
* both affected packages are race-clean.

Honest boundary (§11.4.3 / §11.4.201): this guard is load-sensitive by
construction. It brackets each batch of renders with a lock-guarded status
vector so that a clean `-race` result cannot come from a window the writer never
touched, and if no orchestrator status transition lands inside a rendering batch
it SKIPs with a reason rather than claiming a pass over an empty window. If the
verdict table below records that SKIP, this run is INCOMPLETE, not green — the
host load that produced it is in `transcripts/host_context.txt`.
