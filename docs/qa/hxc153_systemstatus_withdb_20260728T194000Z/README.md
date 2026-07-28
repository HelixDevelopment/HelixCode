# HXC-153 — evidence: `TestGuard_GetSystemStatus_WithDB_StillReports` now exercises the DB-present path

**Item:** HXC-153 (Bug / Medium) — test named for a database-present case it never exercised.

## What the test actually covered vs what it claimed

| | Claimed by the name | Actually covered (pre-fix) |
|---|---|---|
| Server under test | non-nil `db` wired | `&Server{}` — **no database at all** |
| Assertion | db-present path still reports a real verdict | `w.Code == 200` only |
| Net effect | db-present branch guarded | re-ran the nil-db case its sibling already covers |

The `s.db != nil` branch of `getSystemStatus` (handlers.go:964) had **no coverage**
from this test. The name is what a reader trusts when deciding whether a
behaviour is guarded — same documentation-vs-reality class as HXC-152.

## Fix: a real seam, not a rename

`&database.Database{Pool: nil}` is a genuinely **non-nil** `*Database` whose
`HealthCheck()` returns an error (database.go:307 guards a nil `Pool`). It drives
the `s.db != nil` branch end-to-end with **zero infrastructure**, so the test now
earns its existing name rather than trading a misleading name for a silent hole.

Load-bearing assertion: the response must NOT be `"disabled"` (a value the nil-db
branch can never escape) and must be the real verdict `"unhealthy"`.

Honest boundary (§11.4.6): this covers db-present + HealthCheck-**fails**. The
db-present + HealthCheck-**succeeds** (`"healthy"`) arm needs a live pool and is
left to the integration suite — deliberately not claimed here.

## §1.1 paired mutation

Mutation applied to the guarded behaviour in `internal/server/handlers.go`:

```go
-	if s.db != nil {
+	if false && s.db != nil {   // nil-guard swallows the real health check
```

| Run | Result | Log |
|---|---|---|
| Baseline package (pre-change) | exit 0 | `00_baseline_pkg.txt` |
| GREEN — both guards, fixed artifact | exit 0, both PASS | `01_green_guard.txt` |
| **RED — under mutation** | **exit 1, `WithDB_StillReports` FAILS** | `02_mutation_red.txt` |
| Full package (mutation restored) | exit 0 | `03_full_pkg.txt` |
| `-race` full package | exit 0 | `04_race.txt` |

**Proof the OLD test was blind:** under the same mutation the failure is raised on
the *body* assertion, which runs only *after* the `w.Code != http.StatusOK` check
has already passed — i.e. the handler still returned **200**. The pre-fix test body
asserted nothing but `w.Code == 200`, so it would have **PASSED** under this
mutation. `TestGuard_GetSystemStatus_NilDB` also still PASSES under the mutation
(nil-db legitimately yields `"disabled"`), confirming the sibling does not cover
this branch either.

Mutation restored before commit; `git diff` on `handlers.go` is empty (§11.4.84).
