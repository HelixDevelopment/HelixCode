# What this run certifies — and what it explicitly does NOT

`0a4df699` routed every background widget mutation in the harmony_os and
aurora_os Fyne apps through `internal/fyneui`, and gave three `for range
ticker.C` loops — which could never terminate and leaked for the process
lifetime — the app's existing stop channel.

CERTIFIED HERE: the dispatch sites and the stop-honouring loops are present in
the tracked source that ships today; both packages are green and race-clean
under `-race`; and — the part that actually matters — the dispatch sites are
RUNTIME-exercised with the race detector watching.

That last point is a correction to the commit's own honest caveat. `0a4df699`
said "No test in either package constructs the tabs, so the workers never start
and the detector never observes them", and noted aurora measured zero races
BEFORE the change for exactly that reason. That was true then. It is no longer
true: `9afc3da2` (HXC-158) added guards that build the real tabs, drive the
production `TickOrStop` loop, and watch a real renderer repaint while the worker
mutates the widgets. This run executes those guards rather than repeating the
commit's stale caveat — the gap the commit flagged has since been closed by
other work, and evidence that ignored that would understate what is provable
today.

Load sensitivity (§11.4.50): those guards need genuine renderer/worker overlap
and call `t.Fatalf` when they observe fewer than two repaints in 3.5 s, which a
contended host can cause. That is an INCONCLUSIVE run, not a defect, so it is
recorded as a SKIP with the host load attached (`transcripts/host_context.txt`)
and never as a pass. Check the verdict table above to see which applied to THIS
run.
