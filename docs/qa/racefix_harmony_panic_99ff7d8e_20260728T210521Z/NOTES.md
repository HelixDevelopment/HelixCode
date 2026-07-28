# What this run certifies

`99ff7d8e` repaired guards that panicked on their FIRST execution — a test that
cannot run is not a guard. The production half added a nil-client guard on
harmony's API refresh path so it degrades to local data rather than panicking.

Captured here: the nil-guard's log line in the shipped source, the repaired
aurora guard executed under `-race -count=2` with `--- PASS:` asserted, and both
application packages green under `-race` — which is the direct falsification of
"panics on first execution".
