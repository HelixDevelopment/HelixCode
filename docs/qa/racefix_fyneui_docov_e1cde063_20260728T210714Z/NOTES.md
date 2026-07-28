# What this run certifies

`e1cde063` closed three Minor findings from the independent review of `e879702c`:
a test comment that stated the opposite of the truth, a coverage gap where `Do`'s
lock path was pinned by no assertion, and a test that closed `stopUpdate`
directly, sidestepping the `sync.Once` that `Close()` uses (one edit away from a
"close of closed channel" panic).

Captured here: the new `TestDoSerializesAgainstSync` guard exists and RUNS (both
serialisation guards asserted on their `--- PASS:` lines under `-race -count=2`),
the `sync.Once` bypass is provably gone from the desktop test (`git grep` finds
no surviving occurrence), and both affected packages are race-clean.

Honest note carried forward from the commit: `-race`, and only `-race`, is what
pins these locks. The count assertion alone still passes with the locking
stripped, because every increment runs on a single worker goroutine under the
Fyne test driver. That is stated in the test itself rather than papered over.
