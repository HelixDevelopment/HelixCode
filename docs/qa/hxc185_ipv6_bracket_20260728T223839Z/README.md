# HXC-185 — unbracketed IPv6 authorities: captured RED/GREEN evidence

**Item:** HXC-185 (Bug, High)
**Fix commit:** `82313504`
**Shared helper:** `helix_code/internal/netutil/netutil.go`

All exit codes below were taken directly from the invoking shell. (The first agent on
this item died mid-run and explicitly flagged its own scan verdicts as false alarms
caused by `cmd | head` masking exit status; every verdict here was re-established with
`${PIPESTATUS[0]}` or an unpiped command.)

## The defect, captured on go1.26.4

```
net.Dial("tcp", "::1:6379")   -> dial tcp: address ::1:6379: too many colons in address
net.Listen("tcp", "::1:0")    -> listen tcp: address ::1:0: too many colons in address
net.Dial("tcp", "[::1]:6379") -> connect: connection refused        (address ACCEPTED)
net.Dial("tcp", "[[::1]]:25") -> address [[::1]]:25: missing port   (double-bracket also fatal)
```

The sharpest case: `internal/discovery/health_monitor.go` hand-bracketed
`ServiceInfo.Host` in `checkTCP` (comment: "IPv6 address needs brackets") while the
HTTP check twelve lines away did not — same file, same field, one path working and one
failing for the same service.

## Citation triage (§11.4.6 — verified, not trusted)

| Cited site | Verdict |
|---|---|
| `discovery/health_monitor.go:331` | CONFIRMED — fixed (plus `checkTCP` folded onto the shared helper) |
| `discovery/registry.go:45,:427,:459` | CONFIRMED — fixed (Address, health URL, 2x dial) |
| `redis/redis.go:37` | CONFIRMED — fixed |
| `server/server.go:192` | CONFIRMED — fixed |
| `worker/ssh_pool.go:605` | CONFIRMED — fixed |
| `memory/memory_manager.go:549,:868` | CONFIRMED — fixed (redis + memcached) |
| `notification/engine.go:480` | CONFIRMED — fixed (SMTP) |
| `cognee/client.go:64` | CONFIRMED — fixed |
| `tests/testinfra/testinfra.go:147,152,157,162,378` | CONFIRMED — and UNDER-cited: 11 affected sites, not 5. Swept whole. |
| `config/config.go:1260` | RECLASSIFIED — cosmetic, not a defect. `ConfigInfo.ServerAddress` is a JSON-reported diagnostic, never dialled. Changed for consistency with what `server.go` binds; the one changed site with no dedicated guard. |
| `llm/llamacpp_provider.go:141,:323` | **RECLASSIFIED — deliberately NOT changed.** |

### Why llamacpp was left alone

`baseURL` at those lines is a **full URL**, not a host:

```go
baseURL := p.config.ServerHost          // fed from cfg.Endpoint (provider_factory.go:230)
if baseURL == "" { baseURL = "http://localhost:8080" }
if p.config.ServerPort != 0 { baseURL = fmt.Sprintf("%s:%d", baseURL, p.config.ServerPort) }
```

`netutil.JoinHostPort("http://localhost:8080", 9000)` would see colons in the "host",
bracket the entire URL, and yield `[http://localhost:8080]:9000` — turning a working
call site into a broken one. The genuine latent defect there is different (a port
appended to a URL that may already carry one; no scheme when `ServerHost` is a bare
host) and is out of scope for this item.

## The double-bracket trap

`net.JoinHostPort` brackets unconditionally when the host contains a colon, including
an already-bracketed host. Hosts arrive in BOTH shapes in this codebase
(`net.SplitHostPort` strips brackets; a URL authority carries them), so
`netutil.JoinHostPort` normalises via `UnbracketHost` first and is idempotent for either.

## §11.4.115 RED — guards FAIL on the pre-fix artifact

Production fixes reverse-patched out (`git apply -R`), guards left in place:

```
--- FAIL: TestNewClient_IPv6Host_ReachesRealListener            (internal/redis)
--- FAIL: TestServiceInfo_Address_BracketsIPv6                  (internal/discovery)
--- FAIL: TestHealthMonitor_checkTCP_IPv6                       (internal/discovery)
--- FAIL: TestHealthMonitor_checkHTTP_IPv6                      (internal/discovery)
--- FAIL: TestServiceRegistry_checkTCPHealth_IPv6               (internal/discovery)
--- FAIL: TestServiceRegistry_checkHTTPHealth_IPv6              (internal/discovery)
--- FAIL: TestNewClient_IPv6Host_BaseURLReachesServer           (internal/cognee)
--- FAIL: TestNewRedisMemoryProvider_IPv6_ReachesListener       (internal/memory)
--- FAIL: TestNewMemcachedMemoryProvider_IPv6_ReachesListener   (internal/memory)
--- FAIL: TestEmailChannel_Send_IPv6_ReachesListener            (internal/notification)
--- FAIL: TestServerAddr_IPv6_Listens                           (internal/server)
--- FAIL: TestCreateSSHClient_IPv6_ReachesListener              (internal/worker)
--- FAIL: TestConfigURLBuilders_IPv6                            (tests/testinfra)
--- FAIL: TestConfigDialURLs_IPv6                               (tests/testinfra)
RED0_PREFIX_EXIT=1   RED0_REST_EXIT=1
```

Patch re-applied afterwards; restored tree verified **md5-identical** to the pre-revert
diff (`ffac4cfbbcc17ba13d0a922605e54087`), so the revert left no residue (§11.4.84).

## GREEN — guards on the fixed artifact

```
ok internal/redis  ok internal/discovery  ok internal/cognee  ok internal/memory
ok internal/notification  ok internal/server  ok internal/worker  ok tests/testinfra
GUARDS_GREEN_EXIT=0
ok dev.helix.code/internal/netutil   NETUTIL_EXIT=0
```

## Full regression sweep — every affected package, complete suites

```
ok internal/netutil 0.104s   ok internal/redis 13.599s    ok internal/discovery 22.500s
ok internal/cognee 34.204s   ok internal/memory 24.067s   ok internal/notification 29.697s
ok internal/server 10.508s   ok internal/worker 49.979s   ok internal/config 21.305s
ok tests/testinfra 0.405s
FULL_SUITES_EXIT=0
```

No pre-existing test was broken.

## Anti-bluff notes

Guards assert **positive sink-side evidence** (§11.4.69): a real TCP listener bound to
IPv6 loopback records whether a connection was genuinely accepted. They drive REAL
exported constructors (`redis.NewClient`, the cognee client, `ssh.Dial`, SMTP send, the
server's listen path). **No guard re-implements the join**, so none is a replica-RED.

## Honest gaps (§11.4.6)

- **RFC 6874 zone IDs are NOT handled.** `fe80::1%eth0` needs `%25` percent-encoding
  before `url.Parse` accepts it. No current producer supplies zones, so this is noted
  and deliberately unfixed — it is a separate latent defect.
- `config.go GetConfigInfo` has no dedicated guard (cosmetic site; see triage above).
- Hosts without IPv6 loopback SKIP these guards honestly (§11.4.3) rather than pass.
- `llm/llamacpp_provider.go`'s real latent defect (port appended to a URL; missing
  scheme for a bare host) is identified but NOT fixed here.
