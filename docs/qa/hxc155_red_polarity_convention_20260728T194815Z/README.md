# HXC-155 — evidence: standardised RED-polarity switch convention (`internal/server`)

**Item:** HXC-155 (Task / Low) — divergent polarity-switch env-var convention.

## 1. Complete enumeration (§11.4.118 — enumerated, not "none found")

Re-derived independently by scanning **every** env-var read in the package
(`os.Getenv` / `os.LookupEnv` / helper shims), **not** by grepping `RED_MODE` —
which under-enumerates and is precisely how the drift went unnoticed.

**7 distinct switch names across 8 read sites:**

| # | Env var | File(s) | Read style (before) | Default |
|---|---|---|---|---|
| 1 | `RED_MODE` | `llm_generate_regression_test.go` (`redMode` helper), used by `llm_default_model_regression_test.go`, `llm_generate_native_model_regression_test.go`, `llm_rag_test.go`, `llm_working_funnel_test.go`, `wire_facade_model_identity_test.go` | shared `redMode(t)` | GREEN |
| 2 | `RED_MODE` | `server_nil_redis_test.go` (×2 sites) | bare `os.Getenv` | GREEN |
| 3 | `HELIX_RED_MODE` | `handlers_systemstatus_guard_test.go` | bare `os.Getenv` | GREEN |
| 4 | `HELIX_QASTATUS_RED_MODE` | `handlers_qastatus_deadlock_guard_test.go` | bare `os.Getenv` | GREEN |
| 5 | `RED_LLM_AUTH` | `llm_auth_guard_test.go` | `getenvDefault` shim | GREEN |
| 6 | `RED_WS_AUTH` | `ws_auth_test.go` | `getenvDefault` shim | GREEN |
| 7 | `RED_WIRE_FACADE_AUTH` | `wire_facade_auth_test.go` | `getenvDefault` shim | GREEN |
| 8 | `RED_PROJECT_IDOR` | `handlers_project_idor_test.go` | `getenvDefault` shim | GREEN |

Three distinct read styles confirmed: shared `redMode(t)`, bare `os.Getenv`, and
the `getenvDefault` shim.

**Not a switch (checked, excluded with reason):**
- `cors_security_test.go` — `TestCORSMiddleware_RED_WildcardOriginWithCredentials_Forbidden`
  carries `RED` in its *name* only; it is an always-on guard with no env switch.
- `HELIX_LLM_PROVIDER`, `qaStatusProbeEnv`, `streamCrashProbeEnv`, `PROBE_N`,
  `PROBE_PRESSURE` — env vars, but not polarity switches.

## 2. Inverted-default audit (the HXC-152 class)

**No switch has an inverted default.** All 8 read sites default GREEN. This is
now asserted *behaviourally*, per switch, by
`TestRedPolarityConvention_DefaultsAreGreen` — a check, not a comment.

## 3. Decision and justification

The item's authoritative description states per-guard names are **genuinely
useful** ("testers need to flip them one at a time rather than all at once, so
the goal is **not** to collapse them into one variable"). So the switches were
**not** unified. Instead:

1. **Per-guard names kept** — each still flips exactly its own guard.
2. **`RED_MODE` promoted to an umbrella** — it now turns *every* migrated guard
   RED at once. This closes the concrete operator-facing defect: before this
   change, `RED_MODE=1` silently did **not** flip the divergently-named guards,
   so a RED sweep reported success while those guards never reproduced their
   defects — a false impression of coverage.
3. **One helper shape** — `redModeFor(t, ENV)`; the bespoke `getenvDefault` shim
   and every bare `os.Getenv` polarity read are gone.
4. **One documented place** — `red_polarity_convention_test.go` holds the
   convention, the registry, the helper, and the enforcement.

**Backward compatibility: zero names dropped.** Every historical name still
works, because reproduction commands cite them verbatim — e.g. `docs/Fixed.md:993`
documents `RED_WIRE_FACADE_AUTH=1 go test -run TestWireFacadeEndpoints_NoKeyRejected`.
Renaming would have orphaned that documented command. The truthy vocabulary was
*widened* (`1`/`true`/`yes`), a superset of both prior behaviours, so no existing
invocation changes meaning.

## 4. Proof the sweep gap is closed (end-to-end, not just unit)

| Command | Exit | Meaning |
|---|---|---|
| `go test -run TestWireFacadeEndpoints_NoKeyRejected` | 0 | GREEN guard passes |
| `RED_MODE=1 go test -run TestWireFacadeEndpoints_NoKeyRejected` | **1** | umbrella now **reaches** the `RED_WIRE_FACADE_AUTH` guard; its RED branch asserts the defect is present and correctly fails on the fixed artifact (§11.4.115) |

Before this change the second command exited **0** — the guard ignored `RED_MODE`
and silently ran GREEN.

## 5. Honest residual gap (§11.4.6)

`HELIX_QASTATUS_RED_MODE` in `handlers_qastatus_deadlock_guard_test.go` was
**deliberately not migrated**: that file was being edited concurrently under
HXC-154, and racing it would violate §11.4.84 working-tree quiescence. It is
registered in the enumeration and defaults GREEN, but is not yet umbrella-aware.
Tracked in `redPolarityPendingMigration`; the conformance test emits an explicit
`SKIP-OK` naming the reason rather than passing silently.

The gap is demonstrated live, which also shows the exact pre-fix failure mode:

| Command | Exit | Meaning |
|---|---|---|
| `RED_MODE=1 go test -run TestGuard_GetQASessionStatus_RecursiveRLockDeadlock` | 0 | umbrella **silently ignored** — the pre-HXC-155 failure mode |
| `HELIX_QASTATUS_RED_MODE=1 go test -run …` | 1 | its own name still flips it — per-guard independence intact |

## 6. §1.1 paired mutations

| Mutation | Expected | Result | Log |
|---|---|---|---|
| A: a new guard reads unregistered `RED_SOME_NEW_GUARD` | conformance FAILs | **exit 1**, `unregistered RED polarity switch "RED_SOME_NEW_GUARD"` | `02_mutationA_unregistered_switch.txt` |
| B: `redEnvTruthy` returns true when unset (HXC-152 inverted default) | defaults check FAILs | **exit 1**, FAILs for every registered switch | `03_mutationB_inverted_default.txt` |

Both mutations reverted; residue scan clean (§11.4.84).

## 7. Verification (exit codes captured directly, never through a pipe)

| Command | Exit |
|---|---|
| `go vet ./internal/server/...` | 0 |
| `go test -count=1 ./internal/server/` | 0 |
| `go test -race -count=1 ./internal/server/` | 0 |
| `go build ./internal/... ./cmd/...` | 0 |
| `gofmt -l <8 touched files>` | empty |

`go build ./...` is not used: it fails at pristine HEAD on missing X11/GLFW
headers — a host gap (§11.4.201), not a code defect.

**Pre-existing finding (not introduced here, not fixed here):**
`internal/server/cors_security_test.go` is gofmt-unformatted **at HEAD**. It was
not touched by this change, so it was left alone rather than adding unrelated
churn. `server_nil_redis_test.go` was also unformatted at HEAD and *was*
formatted, since this change touches it.
