# HXC-194 — mock services accepted requests from any origin

**Commit:** `92840ade` · **Gate:** `CM-MOCK-SERVICES-CORS-ALLOWLIST` — `gate_run.txt`, exit 0.

A literal `*`, not origin reflection, so exploitation did not even require the
attacker's origin to be reachable by name — and it was in TWO services, not one.
The Slack mock's routes return every captured message and webhook body, with
DELETE to destroy them: a read-and-drive hole, not a header nit.

**The trap:** the upstream advisory describes the same weakness in the library's own
CORS toolkit, but neither module depends on it (`gin-contrib/cors` has zero
occurrences in either `go.sum`). Upgrading would have closed the advisory, cleared
the scanner, and left both holes open. The gate asserts OUR behaviour, never a
dependency version.

Mutation-tested by the orchestrator: wildcard reintroduced → exit 1 naming the
assertion (`wildcard emitted: ... method=OPTIONS`); reverted → exit 0.

Allowlist is config-sourced, default empty = deny all, and a literal `*` in the
allowlist is rejected so the defect cannot return via configuration.
