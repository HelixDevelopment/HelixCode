# Concurrency measurement — full facts

Commit under test: `c9bad26aa7821832cc6297d7d2193731ba8ac733`

Historical claim (the commit's OWN recorded measurement, NOT re-measured
here — this run exercises the wire path only, and RegisterTool is
unreachable from outside the process, see header note):
```
Verified: RED 11.8s -> GREEN 3.5s (-race); go test ./internal/mcp -run 'Stress|Chaos' -race
-count=3 ok, 0 deadlock, 0 race; conductor re-verified -race -count=2 ok 9.1s; full package -race
ok; vet clean. Independent §11.4.142 review: GO, no blocking findings.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>

```

## This run's measurements (N=32 independent MCP clients each
doing a real WebSocket upgrade + JSON-RPC initialize + tools/list)

```
-- sequential (one connection at a time) --
{
  "n": 32,
  "wall_clock_ms": 34.088383999915095,
  "ok": 32,
  "err": 0,
  "min_ms": 0.04470699786907062,
  "mean_ms": 0.057470874935461325,
  "p50_ms": 0.05399200017563999,
  "p90_ms": 0.07128799916245043,
  "p95_ms": 0.08082199929049239,
  "max_ms": 0.11305099906167015
}

-- concurrent (all fired together) --
{
  "n": 32,
  "wall_clock_ms": 23.63165800124989,
  "ok": 32,
  "err": 0,
  "min_ms": 0.6940489984117448,
  "mean_ms": 0.7223841563472888,
  "p50_ms": 0.7152709986257832,
  "p90_ms": 0.756803998228861,
  "p95_ms": 0.7601690012961626,
  "max_ms": 0.7619809985044412
}
```

## Calibration method (§11.4.6 — no invented thresholds)

- Throughput invariant: `concurrent_wall_clock_ms < sequential_wall_clock_ms`
  — both numbers measured in THIS run, on the SAME server, moments apart.
- Latency-bound invariant: `concurrent_p95_ms <= 3 * max(sequential_max_ms, 50ms)`
  — the multiplier and floor are explained in the script header; the
  reference value (sequential max) is measured in THIS run, not assumed.

## Honest boundary (§11.4.6)

This run proves the CURRENT build's tools/list path stays correct and
fast under N=32-way concurrent client load. It does NOT
re-run the pre-fix artifact and does NOT reproduce the concurrent
register-vs-list contention the fix's commit message describes, because
RegisterTool has no client-reachable entry point in the shipped server
(grep-verified, see grounding.txt) — that half of the fix is covered by
the Go stress test cited in the commit message, not re-derived here.
