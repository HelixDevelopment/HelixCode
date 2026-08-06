# HXC-218 — candidate-site classification (§11.4.102)

Every `%s:%s`-shaped composition in `helix_code/` and `submodules/containers/`
was classified **before** any edit, because this ticket family has already been
bitten by applying the join to a value that was not a bare host. Non-test files
only; `_test.go` excluded.

Raw sweep output: `7_treewide_sweep.log`.

## The measurement that determined the fix boundary

Recorded in `0_parse_forensics.log` (go1.26.4). The two library layers do **not**
agree, and that disagreement is the whole shape of the defect:

| expression | result |
|---|---|
| `net.ResolveTCPAddr("tcp", "::1:8080")` | **ERROR** `too many colons in address` |
| `net.ResolveTCPAddr("tcp", "[::1]:8080")` | resolves |
| `url.Parse("http://::1:8080")` | **parses** — `Host="::1:8080"` `Hostname="::1"` `Port="8080"` |
| `url.Parse("http://::1:8080/api/health")` | **parses** — same |
| `url.Parse("http://2001:db8::1:9000/x")` | **parses** — `Hostname="2001:db8::1"` `Port="9000"` |
| `url.Parse("http://fe80::abcd/x")` | **ERROR** `invalid port ":abcd" after host` |

`url.Parse` is **lenient**: it splits the authority at the **last** colon, so an
unbracketed IPv6 authority still yields the correct `Hostname`/`Port` whenever a
numeric port is present, and `net/http` then dials the bracketed form itself.
The resolver is **not** lenient.

Consequence: **the dialled sites were hard-broken; the URL-composing sites were
not.** Both were repaired, but only the dialled ones could be shown to fail a
reachability assertion — see "correction to the brief" below.

## Classified sites

### Repaired (this commit)

| # | file:line | value shape | class | verdict |
|---|---|---|---|---|
| 1 | `containers/pkg/health/types.go:53` | bare host + port | dialled by `CheckTCP`/`CheckGRPC` | **hard failure** — a REACHABLE IPv6 socket reported unhealthy. Fixed. |
| 2 | `containers/pkg/endpoint/resolver.go:29` | bare host + port | `ResolveHostPort`, exported for consumers to dial | **hard failure** at the public surface. Fixed. |
| 3 | `containers/pkg/health/http.go:26` | bare host + port → URL | parsed by `net/http` | probe **succeeded** via leniency, but the URL published in `Details["url"]` carries an authority the resolver rejects, and fails `url.Parse` outright once the port is absent/non-numeric. Fixed (canonicality). |
| 4 | `containers/pkg/endpoint/endpoint.go:85` | bare host + port → URL | `resolveURL` | same as #3, **plus** a no-port branch where the final IPv6 group is misread as a port. Fixed (both branches). |

### Already correct — left untouched

| file:line | why |
|---|---|
| `containers/pkg/health/helix_infra.go:106` | already `net.JoinHostPort`. The internal inconsistency that made this defect visible. |
| `containers/pkg/discovery/tcp.go:37` | already `net.JoinHostPort`. |
| `containers/pkg/serviceregistry/registry.go:250` | already `net.JoinHostPort`. |
| `containers/pkg/egress/egress.go:98` | already `net.JoinHostPort`. |
| `containers/pkg/remote/bootstrap.go:66` | already `net.JoinHostPort`. |
| `containers/pkg/brokertest/{postgres,redis,etcd,brokertest}.go` | already `net.JoinHostPort`, or a hardcoded `127.0.0.1` literal. |
| `helix_code/tests/testinfra/testinfra.go:138` | the `%s:%s` is **user:password**, not host:port; the authority already goes through `netutil.JoinHostPortStr` (HXC-185). |

### NOT a host authority — must NOT be changed (trap #1)

Applying the join to any of these would corrupt working code.

| file:line | actual meaning |
|---|---|
| `containers/pkg/emulator/containerized.go:467` | `container:path` — a `podman cp` source. |
| `containers/pkg/remote/runtime.go:42` | `remote:<name>:<runtime>` — a display identifier. |
| `helix_code/internal/tools/lsp_client.go:260` | `<server>:<file>:<id>` — a request key. |
| `helix_code/internal/worker/isolation.go:146` | `chown user:group` — a POSIX owner spec. |
| `helix_code/internal/workflow/snapshots/comparison.go:302` | `git show <ref>:<path>` — a git revision spec. |
| `helix_code/internal/cognee/service.go:846`, `internal/memory/providers/character_ai_provider.go:{1748,1773}`, `internal/llm/{integrated_model_manager.go:336,351, cross_provider_registry.go:228, model_download_manager.go:300}` | cache/map composite keys. |

### Open sibling — adjacent class, NOT fixed here (§11.4.6 honest boundary)

Two sites compose an **SSH/scp** target, not a `host:port` authority. IPv6 needs
bracketing there too (`user@[::1]:/path`, `ssh -L port:[::1]:port`), but the
syntax, the escaping rules and the RED reproduction are all different — an
SSH-forward spec is not a URL authority. Fixing them inside this ticket would be
an unreproduced, untested change to remote execution, which is exactly the
trap this ticket family keeps falling into.

| file:line | shape |
|---|---|
| `containers/pkg/remote/ssh_executor.go:237` | `user@host:remoteParent` — scp destination |
| `containers/pkg/remote/ssh_executor.go:265` | `user@host:remoteDir` — scp destination |
| `containers/pkg/network/tunnel.go:222` | `-R remotePort:host:localPort` — ssh reverse forward |
| `containers/pkg/network/tunnel.go:226` | `-L localPort:host:remotePort` — ssh local forward |

**Recommend a follow-up ticket.** Both are reachable with an operator-supplied
IPv6 address via `CONTAINERS_REMOTE_HOST_N_ADDRESS`.

## Why `net.JoinHostPort` could not be used directly

It brackets on seeing **any** colon, so it corrupts two shapes these call sites
legitimately receive:

- an already-bracketed host: `"[::1]"` → `"[[::1]]:9000"` — rejected as hard as
  the unbracketed form;
- a host field holding a full URL: `"http://localhost:8080"` →
  `"[http://localhost:8080]:9000"` — this is trap #1, and `resolveURL` really
  does accept a host that already carries a scheme (it checks
  `strings.HasPrefix(base, "http://")` immediately afterwards).

`internal/netaddr.BracketHost` therefore brackets only what `net.ParseIP`
confirms is an IPv6 literal, and covers two forms a naive rule misses:

- `::ffff:127.0.0.1` — `net.IP.To4()` reports it as IPv4, so a `To4`-based test
  would wrongly exempt it, yet its colons break an authority all the same;
- `fe80::1%eth0` — `net.ParseIP` rejects a zone, so the zone must be stripped
  before the parse and preserved in the output.

## Why the fix is at the JOIN sites and never on a stored host (trap #2)

`containers/pkg/endpoint/resolver.go:52` compares the **raw** host against
`"::1"` to decide locality. Bracketing `ServiceEndpoint.Host` at the source
would have silently stopped the loopback being recognised as local, flipping
every local-vs-remote decision downstream — trading a loud defect for a quiet
one. Two guards pin this:

- `TestHXC218_IsLocalEndpoint_UnaffectedByBracketing` — `"::1"` is still local;
- `TestHXC218_ResolveHostPort_DoesNotMutateHost` — three resolve calls in
  sequence leave `Host` untouched and locality intact.

## Correction to the brief this ticket was dispatched with

The dispatch stated as settled fact that `url.Parse` **hard-errors** on
unbracketed IPv6, citing HXC-202's recorded evidence, and instructed that this
be treated as settled. **The measurement above shows the opposite** for the
shapes that actually occur here: `url.Parse` accepts `http://::1:8080` and
recovers the correct host and port. The predecessor agent's instinct — recorded
in its final message and overridden — was correct.

This does **not** invalidate HXC-202's *fix* (setting `URL` explicitly is still
right, and it makes the reported and probed addresses byte-identical). It
invalidates HXC-202's stated *rationale* for the HTTP path, and it means
`cmd/security_scan/hxc202_ipv6_addr_test.go` contains a **blind RED assertion**:
its `RED_MODE=1` branch requires `url.Parse` to reject
`http://::1:9000/api/system/status`, which the measurement shows it does not.
That guard would therefore fail to reproduce against its own pre-fix artifact.

**Recommend a follow-up** to re-baseline HXC-202's RED assertion against the
measured behaviour. Not changed here: it is another ticket's guard, its GREEN
polarity is correct and passing, and silently rewriting another ticket's RED
would destroy the audit trail (§11.4.120 — reconcile visibly, never quietly).

The same flaw was present in this ticket's inherited draft guard and **was**
corrected here, because it is this ticket's own guard: see `1_red_prefix.log`,
where the original HTTP RED assertion failed to reproduce (the probe returned
200 against a real IPv6 server), and `3_red_prefix_corrected.log`, where the
re-baselined assertion reproduces against pristine `HEAD`.
