# HXC-253 — captured verification: three standing guards existed and nothing executed them

Captured 2026-08-10T18:47:22Z. Main repo HEAD `be5d56be`.

## 1. The wiring commit

```
be5d56be gates: execute the three standing guards nothing was running (G30-G32)
```

## 2. The three guards are now invoked by the sweep (G30-G32)

```
1160:        bash "$ROOT/scripts/testing/guard_hxc229_gateway_release_mode.sh" \
1185:        bash "$ROOT/scripts/testing/guard_hxc233_completion_path_live.sh" \
1203:        bash "$ROOT/scripts/testing/guard_hxc244_health_components_registered.sh" \
```

## 3. Guard files present and executable

```
-rwxr-xr-x 1 milos milos 16036 Aug  9 19:37 scripts/testing/guard_hxc229_gateway_release_mode.sh
-rwxr-xr-x 1 milos milos  8942 Aug  9 18:15 scripts/testing/guard_hxc233_completion_path_live.sh
-rwxr-xr-x 1 milos milos  8119 Aug  9 18:15 scripts/testing/guard_hxc244_health_components_registered.sh
```

## 4. Honest boundary (§11.4.6)

The sweep that now names these guards is operator-invoked (`make ci-validate-all` /
`scripts/verify-all-constitution-rules.sh`). Wiring makes them REACHABLE and executed
when the sweep runs; it does not establish unattended enforcement. That residual gap is
stated in the item, not claimed closed.
