# HXC-298 — F1 ParseIP guard: paired §1.1 mutation evidence

**Captured** 2026-08-13, main-stream (T1/main - claude1), host quiescent
(all four background streams had terminated; no concurrent build or sweep).

## What was being proven

The prior independent review found that the `net.ParseIP(xri)` validation
guard on the `X-Real-IP` header had **no test asserting it anywhere**.
Deleting it left **all four** packages' suites green, and from a trusted
peer `X-Real-IP: '; DROP TABLE audit; --` was returned as the resolved
identity — which feeds both the audit trail and the rate-limit key.

A guard that no test defends is decoration. The remediation's claim is that
the guard is now asserted. This mutation tests that claim in the only way
that can settle it: **delete the guard and see whether anything notices.**

## Method

Subject: `clientip/clientip.go`, pre-mutation sha256
`1bf1e1fae14cd7a7f9e3a1c4e151787385030fdf994de0e398a05ef363ff1db5`.

Mutation — the guard removed so a trusted peer's `X-Real-IP` is returned
unvalidated (exactly the reviewer's original exploit shape):

```go
-			if net.ParseIP(xri) != nil {
-				return xri
-			}
+			return xri
```

Mutant compiled cleanly (`go build ./clientip/` rc=0) — required, since a
mutant that does not build proves nothing.

## Result

| package | rc | failing tests |
|---|---|---|
| `./clientip/` | 1 | 1 |
| `./security/` | 1 | 1 |
| `./api/` | 1 | 1 |
| `./enhanced/enterprise/` | 1 | 1 |

Before remediation this mutation produced **rc=0 in all four**. It now
produces **rc=1 in all four**. That inversion is the finding.

## Restoration

Restored and verified byte-identical by sha256:
`1bf1e1fae14cd7a7f9e3a1c4e151787385030fdf994de0e398a05ef363ff1db5`
— matches pre-mutation exactly. No mutation markers left on disk (§11.4.84).

## Honest boundary (§11.4.6)

This proves **one** guard is now defended in all four consumers. It does
NOT prove the other guards in this package are defended, that the
extraction preserved the three consumers' prior behaviour, or that the
package's own comments describe what it does. Those are the independent
review's scope and remain open at time of capture.
