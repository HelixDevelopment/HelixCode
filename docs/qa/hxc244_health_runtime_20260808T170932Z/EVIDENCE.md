# HXC-244 — health endpoint closed at the RUNTIME layer

Date: 2026-08-08T17:09:32Z

## Before (binary mtime 2026-08-08 13:52, predating the fix)
```
go tool nm <running binary> | grep -c registerHealthChecks  ->  0
{"status":"healthy","components":[],...}
guard RED_MODE=0 -> GUARD FAILED: EMPTY: status='healthy' but zero components checked
```

## Artifact verified BEFORE install (§11.4.108 layer 2)
```
registerHealthChecks: 6   Ping method-values: 4
md5 0b9ee49bca3c  (previous 8a1bab4645a5, backed up per §9.2)
```

## After restart — RUNTIME layer
```json
{"status":"healthy","components":[{"name":"llm_providers","status":"healthy","duration":387166,"last_checked":"2026-08-08T22:09:32.279348454+05:00"},{"name":"kv_cache_redis","status":"healthy","duration":182766,"last_checked":"2026-08-08T22:09:32.279144425+05:00"},{"name":"vector_store_qdrant","status":"healthy","duration":983960,"last_checked":"2026-08-08T22:09:32.279988063+05:00"},{"name":"llms_verifier","status":"healthy","duration":432635,"last_checked":"2026-08-08T22:09:32.279417038+05:00"}],"timestamp":"2026-08-08T22:09:32.279990456+05:00"}```

## Guard polarity (proves the guard is not blind)
- RED_MODE=0 -> GREEN: names 4 components (llm_providers, kv_cache_redis, vector_store_qdrant, llms_verifier)
- RED_MODE=1 -> exit 1, correctly refuses a fixed build

## No regression
- completions 200, models 200
- gateway NRestarts=0
- coder container untouched

Each component carries a measured `duration` — evidence of a real round-trip,
not a hardcoded list. That is the difference between this and the empty array.
