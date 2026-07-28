# What this run certifies — and what it explicitly does NOT

`ccce0b77` had two halves.

FIRST HALF (certified at runtime here): `monitorSystem` looped on
`for app.systemMonitor.monitoring`, a plain cross-goroutine bool read, while
`Cleanup()` wrote that bool from another goroutine — a data race whose store may
never be observed, so the monitor could outlive the teardown that told it to
stop. The commit's own RED baseline was `TestCleanup` failing with
`WARNING: DATA RACE`. That test is executed here under `-race -count=2` and is
asserted to have genuinely run (`--- PASS: TestCleanup`), and the whole package
is race-clean.

SECOND HALF (NOT certified — recorded as a SKIP, §11.4.3): the five metric fields
shared between the monitor goroutine and the system tab were also unsynchronised
and are now mutex-guarded. No test in this package constructs the system tab, so
nothing exercises the reader side; the detector cannot see a pair it never runs.
This run therefore certifies the accessors EXIST in the shipped source and
nothing more about that half. This run is INCOMPLETE (exit 2) for exactly that
reason — a SKIP is never counted as a pass.
