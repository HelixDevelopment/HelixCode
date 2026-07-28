# What this run certifies

`ed0cdcb8` landed three fixes found by an independent review. Two are covered
here by their own named guards: the auto-save residual window (a stop that
arrives while the loop is already parked in the blocking select) and the
server's response `model` field, which reported the REQUESTED model rather than
the one that actually served the request.

Scope note (§11.4.6): the full `internal/server` suite is NOT run here. That
package was under concurrent edit by other agents during this capture, so a
whole-package verdict would report their in-flight state, not this commit's.
The server half is therefore certified by its two NAMED guards, executed under
`-race -count=2` with `--- PASS:` asserted per test; the persistence half is
certified by both its named guard and the full package suite.
