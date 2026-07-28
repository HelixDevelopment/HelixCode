# What this run certifies

`dd3c0c3b` replaced an unprioritised two-case `select` in the persistence
auto-save loop with an explicit stop-first probe, extracted into
`(*Store).autoSaveTick` so the ordering is directly testable. Go's `select`
picks uniformly at random among ready cases, so with both a pending tick and a
closed stop channel the loop could perform another save AFTER being told to
stop.

Captured here (real command execution, transcripts/ holds every byte):
* the extracted function is present in the tracked source that ships today;
* the commit's own guard `TestAutoSaveTick_StopAlwaysWinsOverPendingTick`
  actually RAN (asserted on `--- PASS:` lines, not merely on exit 0) and passed
  twice under `-race`;
* the whole package is race-clean.

Not certified: nothing about the wider auto-save schedule under real workload —
this is a select-ordering guarantee, not a durability claim.
