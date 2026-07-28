# What this run certifies

`0c2f4940` is a docs-and-tests follow-up to `fa1b52b3` with no behaviour change,
and it exists because two things in that commit were wrong or unguarded.

The overstatement: `fa1b52b3` claimed an abandoned consumer "fills its 100-slot
channel ... and the teardown fires within two windows". That is false. A send
completes — and therefore counts as progress — either because a consumer
received the line OR because there was still buffer headroom to absorb it, so
progress does NOT imply an active consumer. The real bound is roughly
(headroom + 1) x grace. The conclusion survived (still finite, the semaphore
slot still returns, HXC-184 does not recur); the stated mechanism did not.

The coverage gap: M2 (a genuine stderr failure must outrank the stdout teardown
sentinel) and M3 (`OutputIncomplete` must be true when a cancel drops in-flight
lines) shipped in `fa1b52b3` with no standing guard — reverting M2 left the
whole in-tree suite green, which is exactly the §11.4.135 silent-recurrence
vector.

Captured here: the three corrected statements are present in the tracked source
that ships today, and all five guards RAN and passed twice under `-race`
(asserted per test on their `--- PASS:` lines) — the corrected bound pinned by
a real measurement rather than by prose, both M2 guards, and both M3 guards
including the negative case that must stay FALSE.

Not certified: the specific timings quoted in the commit message. The bound
guard asserts the property (exceeds one window, still terminates), which is
what makes the two-window claim unable to return.
