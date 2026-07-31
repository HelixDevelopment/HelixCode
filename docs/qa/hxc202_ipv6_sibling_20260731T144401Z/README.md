# HXC-202 — IPv6 authority bracketing: the two HXC-185 sibling misses

Evidence for HXC-202, the seventh sibling-miss of this cycle. HXC-185
(`9d082a7b`) centralised IPv6 authority bracketing in `internal/netutil` and
applied it where the SERVER binds. Two places that compose the same class of
address were outside that commit's examined set.

Neither site is newly broken; both predate HXC-185. Nothing was made worse.

## 1. Classified site table (§11.4.102)

Every candidate was classified BEFORE any edit. The value class — bare host vs
already-bracketed vs full URL — decides whether the helper applies. Applying it
uniformly corrupts working code.

| # | `file:line` (pre-fix) | What the value actually is | Needs helper? | Action |
|---|---|---|---|---|
| 1 | `helix_code/applications/harmony_os/main.go:348` | `cfg.Server.Address` — **bare host**. `internal/server/server.go:196` binds this very field via `netutil.JoinHostPort`. | YES | **FIXED** |
| 2 | `helix_code/cmd/security_scan/main.go:193` (`endpointDesc`) | `sonarqubeHost()` — **bare host** from `HELIX_SONARQUBE_HOST`, default `localhost`. Used as the audit string AND now as the probed URL. | YES | **FIXED** |
| 3 | `helix_code/cmd/security_scan/main.go:179` (ready message) | same bare host | YES | **FIXED** |

### Confirmation that site 1 is the described defect

`cfg.Server.Address` is a bare host: `internal/server/server.go:196` binds it
with `netutil.JoinHostPort`. With an IPv6 address configured the server binds
`[::1]:8080` correctly while the HarmonyOS client built `http://::1:8080`, which
`url.Parse` rejects — the server is up, the client cannot reach it, and the
symptom reads as "server is down". Captured in `red/harmony_red_mode1.txt`.

`applications/harmony_os` is the ONLY platform app that composes its API base
URL from `Server.Address` + `Server.Port`. `applications/desktop` and
`applications/aurora_os` set a literal `"http://localhost:8080"` into a UI entry
field — a full URL, no host:port join, nothing to fix, and applying the helper
there would corrupt it.

## 2. Full sweep — sites deliberately NOT changed

| `file:line` | Value class | Why NOT changed |
|---|---|---|
| `internal/llm/llamacpp_provider.go:141,323` | **full URL** — `baseURL` starts as `p.config.ServerHost`, fed from `cfg.Endpoint` (`provider_factory.go:230`), default `"http://localhost:8080"` | `JoinHostPort` would yield `[http://localhost:8080]:9000` and **BREAK working code**. Independently re-verified; HXC-185's original reclassification is correct. The genuine latent defect there (a port appended to a URL that may already carry one) is a different item. |
| `applications/harmony_os/main.go:1055` | worker `Host`/`Port` rendered into a Fyne label | Display only — never dialled. Cosmetic. |
| `applications/desktop/main_nogui.go:571`, `applications/harmony_os/main_nogui.go:650` | worker `Host`/`Port` in a `fmt.Printf` list | Display only — never dialled. Cosmetic; would require opening two more files under a second build tag for no functional gain. |
| `applications/desktop/main.go:1576`, `applications/aurora_os/main.go:1867` | literal `"http://localhost:8080"` | Already a full URL. Not a join. |
| `internal/deployment/production_deployer.go:1049` | `net.JoinHostPort(server, "22")` | Already correct. |
| `tests/e2e/challenges/{functional_validator,helixcode_server_client}.go` | `net.JoinHostPort(...)` | Already correct (fixed under HXC-185). |
| `internal/{cognee,config,discovery,memory,notification,redis,server,worker}`, `tests/testinfra` | bare hosts | Already routed through `netutil` by HXC-185. |
| cache keys / diagnostics: `llm/integrated_model_manager.go:336,351`, `llm/cross_provider_registry.go:228`, `llm/model_download_manager.go:300`, `memory/providers/character_ai_provider.go:1748,1773`, `repomap/repomap.go:544`, `tools/lsp_client.go:260`, `tools/web/search.go:144`, `cognee/service.go:846` | `a:b` composite keys | Not addresses. |
| `file:line` diagnostics: `context/mentions/problems_mention.go:79`, `tools/browser/console.go:309,393`, `tools/lsp.go:262`, `tools/mapping/definitions.go:112`, `tests/e2e/challenges/validator.go:508` | source locations | Not addresses. |
| `workflow/snapshots/comparison.go:302` (`git show ref:path`), `worker/isolation.go:146` (`chown user:group`), `deployment/production_deployer.go:933` (`host:path`) | other `a:b` syntaxes | Not addresses. |
| `cli_agents/**`, `constitution/**`, `submodules/**` | vendored / other projects | Out of scope. |

### Out-of-scope defect FOUND but NOT fixed (another project)

`submodules/containers` (`digital.vasic.containers`) carries the same defect
class, and the same internal asymmetry HXC-185 found in `health_monitor.go`:

- `pkg/health/http.go:26` — `fmt.Sprintf("%s://%s:%s%s", scheme, Host, Port, path)`, unbracketed.
- `pkg/health/types.go:53` — `ResolvedAddress()` → `fmt.Sprintf("%s:%s", Host, Port)`, unbracketed; used by the TCP dial.
- `pkg/health/helix_infra.go:106` — correctly uses `net.JoinHostPort`.

Not touched: separate project, explicitly out of bounds for this item.

**Consequence for the `-action=start` path:** the SonarQube and PostgreSQL
endpoints go through `endpoint.Builder` → `BootManager` → the unbracketed
composition above. Bracketing `ep.Host` to work around it is **UNSAFE**:
`pkg/endpoint/resolver.go:50` `IsLocalEndpoint` compares `h == "::1"`, so a
bracketed host would silently stop being recognised as local. The `status` path
is fixed instead, via the documented `HealthTarget.URL` precedence.

## 3. RED / GREEN (§11.4.115)

One test source per site, two roles via the `RED_MODE` polarity switch.
`RED_MODE=1` = reproduce-and-assert-defect-PRESENT; `RED_MODE=0` (default) =
standing regression guard asserting defect-ABSENT. Both roles drive the REAL
production functions — nothing re-implements the join, so neither is a replica.

All exit codes taken directly from the command, never through a pipe (§11.4.6).

| Artifact | Polarity | Expected | Exit | Evidence |
|---|---|---|---|---|
| pre-fix | `RED_MODE=1` | PASS (defect reproduced) | **0** | `red/harmony_red_mode1.txt`, `red/security_scan_red_mode1.txt` |
| pre-fix | `RED_MODE=0` | FAIL (guard catches it) | **1** | `red/harmony_red_mode0_prefix.txt`, `red/security_scan_red_mode0_prefix.txt` |
| fixed | `RED_MODE=0` | PASS (defect absent) | **0** | `green/harmony_green_mode0.txt`, `green/security_scan_green_mode0.txt` |
| fixed | `RED_MODE=1` | FAIL (no longer reproducible) | **1** | `green/harmony_red_mode1_postfix.txt`, `green/security_scan_red_mode1_postfix.txt` |

Reproduced defect, verbatim from `red/harmony_red_mode1.txt`:

```
RED reproduced on the pre-fix artifact: apiServerURL -> "http://::1:8080" (url.Parse rejects it)
```

and from `red/security_scan_red_mode1.txt`:

```
RED reproduced on the pre-fix artifact: sonarqubeHealthURL -> "http://::1:9000/api/system/status" (url.Parse rejects it)
RED reproduced: status target carries no URL, so CheckHTTP composes the unbracketed form
```

### Negative cases in the guards

A guard testing only IPv6 would not have caught the classification mistake that
motivated this item. Each guard asserts, byte-for-byte:

- IPv4 literal (`127.0.0.1`, `0.0.0.0`) — **unchanged**
- hostname (`helix.internal`, `sonar.internal`, `localhost`) — **unchanged**
- already-bracketed `[::1]` — bracketed **exactly once**, never `[[::1]]`
- full URL (`defaultAPIServerURL`) — returned **verbatim**, explicitly asserting
  it is not turned into `[http://localhost:8080]:...`

## 4. Full-suite regression

`go test -tags=nogui -count=1` over the two changed packages plus the shared
helper and the server package that pins the same invariant — `green/full_package_suites.txt`:

```
ok  dev.helix.code/applications/harmony_os  2.203s
ok  dev.helix.code/cmd/security_scan        0.498s
ok  dev.helix.code/internal/netutil         0.102s
ok  dev.helix.code/internal/server         10.295s
```

exit **0** — no pre-existing test broken.

## 5. Honest gaps (§11.4.6)

1. **`applications/harmony_os/main.go` is not compile-verified on this host.**
   It is `//go:build !nogui` and needs X11/GL, which this host lacks — captured:
   `fatal error: X11/Xlib.h: No such file or directory`, exit 1. This is why the
   composition was extracted into the UNTAGGED `server_url.go`, which compiles
   and is tested under `-tags=nogui`. What remains unverified locally is the
   single call site `NewAPIClient(apiServerURL(cfg))`; `config.Load()` returns
   `*config.Config` (`internal/config/config.go:346`), matching the signature,
   and both files are `gofmt -e` clean. A host with X11/GL headers should run
   `go build ./applications/harmony_os/` to close this.
2. **There is no `-tags=ci` in this Go tree.** The repo's convention is
   `-tags=nogui` (`Makefile:340` `go build -tags=nogui ./...`). All evidence here
   uses `-tags=nogui`.
3. **The `security_scan -action=start` path remains IPv6-broken** via the
   containers submodule, as described in §2. Only `-action=status` is repaired.
4. The guards prove address composition, not that a real SonarQube or HelixCode
   server answers; no such service was running here.

## 6. Is the class closed tree-wide?

**Within `helix_code/` (this project): yes**, for host+port joins that are
actually dialled or reported. The sweep in §2 enumerates every remaining
`%s:%d` / `%s:%s` / `":"+` occurrence and classifies each.

**Tree-wide including submodules: no** — `submodules/containers` still carries
the defect (§2), and it is another project.

## 7. Commit attribution — HXC-202 landed inside `a0dc39a5`, not its own commit

Recorded per the precedent already set in this repo by `9fc97eb5` ("HXC-186 —
record that the fix landed inside 790d097c, not its own commit").

While this item was being verified, two concurrent sweeps by other agents staged
and committed the whole HXC-202 change set before it could be committed under its
own message:

- `790d097c` — swept 6 of the RED/GREEN evidence captures.
- `a0dc39a5` ("chore(session): land in-flight agent work + bump helix_agent for
  HXC-172") — swept the remaining evidence plus **all five source files**:
  `applications/harmony_os/{main.go,server_url.go,hxc202_ipv6_addr_test.go}` and
  `cmd/security_scan/{main.go,hxc202_ipv6_addr_test.go}`.

This is the §11.4.84 working-tree-quiescence hazard in reverse: a foreign
`git add` swept files belonging to another agent's in-flight unit of work. No
history is rewritten to correct it (§11.4.113 — force-push is absolutely
forbidden, and the content is correct where it sits).

**Verified after the fact — the swept content is the FINAL, FIXED state, not a
mid-edit RED snapshot** (`git show HEAD:<path>`):

- `server_url.go:53` → `return "http://" + netutil.JoinHostPort(cfg.Server.Address, cfg.Server.Port)`
- `harmony_os/main.go:349` → `app.apiClient = NewAPIClient(apiServerURL(cfg))`
- `security_scan/main.go:96` → `return "http://" + netutil.JoinHostPortStr(sonarqubeHost(), sonarqubePort())`
- `security_scan/main.go:120` → `URL:     sonarqubeHealthURL(),`

`git diff HEAD` over every HXC-202 path is empty: the working tree and HEAD
agree, and the RED/GREEN evidence in this directory was captured against exactly
this content.

**Consequence for a cold session:** searching `git log` for an "HXC-202" commit
subject finds nothing. The change is in `a0dc39a5`; this document is the
attribution record.
