# §11.4.150 deep multi-angle research pass — client-IP trust family

**Covers:** HXC-292, HXC-298, HXC-299 (and bears on HXC-283, HXC-310, HXC-318).
**Conducted:** 2026-08-13, main stream (T1/main - claude1), in parallel with four
background review streams per §11.4.150(F).
**Sources verified:** 2026-08-13 (URLs and access date below, per §11.4.99).

---

## Why this family needed a pass

These items all touch one question: *when a request arrives carrying a header
claiming a different origin address, do we believe it?* §11.4.150 makes a
documented multi-angle pass a precondition for closing any of them, and
requires the pass to do two things — find the best solution, **and** confirm we
are not sitting on a bigger problem we have not noticed.

The second half is the one that paid.

---

## Angle 1 — attack precedent: is this a real, current, exploited class?

Yes, and it is actively being assigned CVEs in 2026. The pattern is uniform
across every case: an application reads a caller-supplied origin header with no
trusted-proxy validation, and uses the result as an identity.

- **CVE-2026-55501** (9router): the login rate limiter reads the origin header
  directly from the request with no trusted-proxy validation, so the value is
  fully attacker-controlled and login throttling is bypassed by rotating it.
- **GHSA-hm36-ffrh-c77c** (Litestar): same shape — header spoofing bypasses
  rate limiting.
- **LibreTranslate #986**: trusts the header to determine client IP with no
  validation or trusted-proxy configuration; rate limit and flood ban bypassed.
- **MDN** states the rule plainly: security-related uses of this header —
  rate limiting, access control — must use **only** addresses added by a
  *trusted* proxy; otherwise the consequences include rate-limiter avoidance,
  access-control bypass and memory exhaustion.

The correct selection rule, per MDN, is to search the list **from the
rightmost** by proxy-count minus one — with one reverse proxy, take the
rightmost entry. The leftmost entry is the one an attacker controls outright.

**Bearing on our items:** HXC-292's right-to-left walk is the correct direction,
matching the documented rule rather than the intuitive-but-wrong leftmost read.

## Angle 2 — framework semantics: what does our stack do by default?

This is the angle that changes the risk assessment, because the unsafe posture
is **the default**, not a misconfiguration.

Gin's trusted-proxy feature is enabled by default **and trusts all proxies by
default** — `0.0.0.0/0` and `::/0`. Its own documentation is explicit that this
is not safe, and lists exactly the three impacts: bypass of IP-based access
controls, **poisoning of logs and audit trails**, and evasion of rate limiting.
The remediation is to call `SetTrustedProxies` with specific CIDRs, or
`SetTrustedProxies(nil)` to ignore the headers entirely and use the peer address.

Independently corroborated inside this repository: a prior agent verified the
`0.0.0.0/0`, `::/0` default against `gin.go:225` at `gin-gonic/gin@v1.12.0` and
recorded it in `submodules/helix_agent/tests/pentest/rate_limit_bypass_test.go`.
Two sources, one external and one measured against our own pinned version.

**Consequence:** in a Gin application, *no code change is required to be
vulnerable*. Silence is the vulnerability. That inverts the usual review
question from "did someone write something unsafe?" to "did anyone write the
one line that makes it safe?"

## Angle 3 — confirm-no-bigger-problem (§11.4.150(C))

Asked directly: does our fix for these items mask a larger instance elsewhere?

Swept every Gin engine construction and every `SetTrustedProxies` call across
all repositories:

- **Production Gin engines: 6** — `helix_code/internal/server/server.go:65`,
  `submodules/helix_llm/cmd/a2a-server/main.go:67`,
  `submodules/helix_llm/internal/server/server.go:51`,
  `submodules/helix_agent/cmd/api/main.go:174`,
  `submodules/helix_agent/internal/router/router.go:160`,
  `constitution/scripts/scheduled-work-engine/internal/server/rest.go:26`.
- **`SetTrustedProxies` calls in production code: ZERO.** The only call in the
  tree is inside a pentest test, constructing a deliberately hardened engine
  for comparison.

So every production Gin engine we ship inherits trust-everyone.

That alone is a shape, not a finding — six items this session were filed on a
defect's shape and turned out to sit in code that never runs. So the next
question was reachability, and the chain was traced end to end:

| step | evidence |
|---|---|
| shipped binary constructs the engine | `helix_code/cmd/server/main.go:156` → `server.New` |
| engine never sets trusted proxies | `internal/server/server.go:65`, `gin.New()` |
| an **unauthenticated** route is registered on it | `server.go:238` `auth.POST("/login", s.login)` |
| the handler consults the spoofable value | `handlers.go:218` `c.ClientIP()` |
| the value is parsed and stored on the session | `auth.go:221-230` → `Session.IPAddress` |
| the session is persisted | `auth_db.go:196-197` |

Reachable, unauthenticated, and persisted. **Filed as HXC-321 (High).**

**Precision matters here.** `net.ParseIP` at `auth.go:222` means a non-IP string
is rejected, so this is *not* injection. The impact is narrower and quieter:
the stored address is well-formed and entirely attacker-chosen. Sign-in history
does not go missing — it actively misleads, which is the "poisoning of logs and
audit trails" impact Gin's own documentation names.

---

## What this changes about the family

1. **HXC-292's right-to-left walk is confirmed correct** against the documented
   selection rule, not merely internally consistent.
2. **HXC-298's `ParseIP` guard matters more than its severity suggested.** The
   audit trail is the *primary* consumer of this value in our system, so
   validation failures land in exactly the record used to investigate an
   incident. Verified closed by paired mutation on 2026-08-13 — see
   `docs/qa/hxc298-parseip-guard-20260813/mutation-evidence.md`.
3. **The family was scoped one component too narrowly.** Fixing how we parse a
   claimed origin address is necessary but not sufficient while the framework
   beneath accepts the claim from anyone. HXC-321 closes that gap.
4. **Reopen-breaking (§11.4.150(E)):** the reason this class kept recurring
   component by component is that each fix addressed *parsing* while the
   *trust boundary* stayed at the default. Any future fix in this family must
   state which of the two it addresses.

## Honest boundary (§11.4.6)

This pass establishes the defect class is real, current, and present in our
tree, and it found one new reachable instance. It does **not** prove the other
five production engines are exploitable — reachability was traced only for
`helix_code`'s login path. The remaining five are an enumerated, un-exercised
gap, stated here rather than silently implied clean (§11.4.118). It also does
not substitute for §11.4.108 runtime-signature verification or the §11.4.40
full-suite retest.

## Sources verified 2026-08-13

- https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/X-Forwarded-For
- https://gin-gonic.com/en/docs/server-config/trusted-proxies/
- https://pkg.go.dev/github.com/gin-gonic/gin
- https://securelayer7.net/lab/cve-2026-55501-9router-x-forwarded-for-rate-limit-bypass
- https://github.com/litestar-org/litestar/security/advisories/GHSA-hm36-ffrh-c77c
- https://github.com/LibreTranslate/LibreTranslate/issues/986
- https://github.com/gin-gonic/gin/blob/master/gin.go
