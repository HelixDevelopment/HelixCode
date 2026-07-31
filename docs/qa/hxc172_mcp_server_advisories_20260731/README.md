# HXC-172 — mcp-server dependency advisories: evidence (2026-07-31)

Scope: `submodules/helix_agent/plugins/mcp-server`. Fix commit in that submodule:
`d0a53e0b`. Prior related commits: `6c2c6091` (ws bump), `e27581da` (HXC-212 CORS),
`30c81925` (HXC-193/195 env config).

## Advisory state — verified, not inherited

| Advisory | Package | Sev | Affected | Ours | State |
|---|---|---|---|---|---|
| GHSA-96hv-2xvq-fx4p / CVE-2026-48779 | `ws` | high | `<8.21.0` | 8.21.1 | **CLOSED** by `6c2c6091` |
| GHSA-58qx-3vcg-4xpx / CVE-2026-45736 | `ws` | medium | `<8.20.1` | 8.21.1 | **CLOSED** by `6c2c6091` |
| GHSA-w48q-cv73-mx4w / CVE-2025-66414 | `@modelcontextprotocol/sdk` | high (CVSS4.0 7.6) | `<1.24.0` | 0.5.0 | **OPEN as a version fact; mitigated in our own code** |
| GHSA-73jw-fp74-p77x / CVE-2026-62389 | `ws` | — | `<8.21.1` | 8.21.1 | CLOSED (published 2026-07-15, after the triage; no action needed) |

The HXC-166 triage doc cites `GHSA-345p-7cg4-v4c7` for this package. That is a
real advisory but a **different** bug (cross-client data leak, affected
`>=1.10.0 <=1.25.3`) which does **not** affect 0.5.0. The doc needs correcting.

## Why the SDK advisory is not closed by upgrading

1. `enableDnsRebindingProtection` first appears in SDK **1.14.0**; 0.5.0 contains
   no occurrence of it. The setting was never available to enable.
2. We never construct an SDK transport — `runSSE()` is a hand-rolled
   `http.createServer`, and the SDK is a declared dependency that is **never
   imported** in `src/` or built `dist/`. It ships in the image and is never loaded.
3. Upgrading to the advisory's stated fix (1.24.0) lands inside the affected range
   of two other SDK advisories that do **not** touch 0.5.0 today
   (GHSA-8r9q-7v3j-jr4g `<1.25.2`; GHSA-345p-7cg4-v4c7 `<=1.25.3`). Clean floor is
   `>=1.26.0` — a breaking major adding ~79 packages to patch never-loaded code.

## Reachability, re-derived after `30c81925` (the service now really listens)

`MCP_TRANSPORT=sse` + published port `9100` mean the HTTP edge genuinely ships.
The SDK/ws *packages* are unreachable (no import path). The *defect class* the
advisory names — no Host-header validation on a locally-bound MCP HTTP server — was
**reachable and present in our own code**, and no bump could have reached it.

## Files

| File | What it shows |
|---|---|
| `red_host_rebinding_prefix.log` | RED on the **pre-fix** artifact, exit 0 — `Host: evil.example` and a hostile `Origin` both **executed** (200, real `serverInfo`) |
| `green_host_rebinding_fixed.log` | GREEN on the **fixed** artifact, exit 0 — both refused 403 before execution; loopback / configured / IP-literal hosts, no-Origin CLI client, allowlisted origin, `/health` all still work |
| `red_on_fixed_artifact_must_fail.log` | RED on the fixed artifact, exit **1** — the guard is not blind |
| `green_cors_regression_after_fix.log` | Pre-existing HXC-212 gate still passes (§11.4.120 reconciled, not fake-passed) |

All runs spawn the real built `dist/index.js` as a child process and drive it over
real sockets — artifact layer, not source (§11.4.108).

## Honest gaps

- The paired §1.1 mutation (disable the host check → assert GREEN FAILs) was **not
  run**. `red_on_fixed_artifact_must_fail.log` is real falsification evidence but is
  not the mutation gate.
- `MCP_ALLOWED_HOSTS` is not yet documented in the Dockerfile / compose file.
- `npm audit` still reports GHSA-w48q-cv73-mx4w and will keep doing so while the
  pin stays at 0.5.0 — that is a version fact, and the decision to accept it is
  recorded in `d0a53e0b`, not silently suppressed.
