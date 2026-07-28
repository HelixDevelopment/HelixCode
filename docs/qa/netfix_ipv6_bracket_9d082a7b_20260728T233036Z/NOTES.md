# What this run certifies

`9d082a7b` closed HXC-185: an IPv6 literal contains colons, so joining it to a
port with `fmt.Sprintf("%s:%d", host, port)` produces an authority that is both
unparseable per RFC 3986 §3.2.2 and rejected outright by the Go resolver. Any
host arriving from configuration, service discovery, or an environment variable
can legitimately be a bare IPv6 literal, so every such join had to bracket. The
fix routes all of them through one shared helper, which also closes the
double-bracket trap: `net.JoinHostPort` brackets unconditionally when the host
contains a colon, including a host that is ALREADY bracketed, and `[[::1]]:80`
is rejected just as hard as the unbracketed form. Hosts arrive in both shapes
here (`net.SplitHostPort` strips brackets; a URL authority carries them), so the
helper normalises before joining and is idempotent for either.

Captured here (real command execution; `transcripts/` holds every byte):

* both helper entry points and the rerouted production call sites are present
  in the tracked source that ships today;
* every guard listed in the verdict table RAN and passed twice under `-race`
  (asserted per test on its `--- PASS:` line, never on exit 0 alone);
* the guards are SINK-SIDE, not string comparisons: each stands up a real
  listener on `::1`, drives the production code path at it, and asserts the
  listener actually accepted the connection or served the request. The
  discovery and memory transcripts are additionally asserted to contain the
  `positive sink-side evidence:` line those tests emit with the observed
  accept/serve count, so this run cannot pass on a test that merely compared
  two strings (§11.4.69);
* RED_MODE=1 on this same fixed artifact FAILS the discovery guard, proving it
  still detects the defect shape (§11.4.115).

Honest boundary (§11.4.6) — the polarity switch is NOT uniform across these
guards, and this run does not pretend it is. `internal/netutil`'s own RED_MODE
branch characterises stdlib behaviour (that a naive join is rejected and a
double-bracketed one too), so it passes on ANY artifact by design and is
therefore useless as a falsifiability proof; the RED polarity case above
deliberately targets `internal/discovery`, whose RED_MODE branch asserts the
DEFECT is present and so can only pass on a pre-fix artifact.

Also not certified: whole-package suites for cognee / discovery / memory /
notification / redis / server / worker. Those packages carry integration tests
that need real infrastructure, and running them here would report the
infrastructure's state rather than this commit's. The call sites this commit
changed are certified by their NAMED sink-side guards; `internal/netutil`, the
package this commit introduced, IS covered by its full suite.
