# Address-composition family — §11.4.150 deep multi-angle research pass

**Date:** 2026-08-12
**Scope:** HXC-268 (landed, not closed) / HXC-280 (open) / HXC-283 (open) / HXC-284 (open)
**Repository state:** `submodules/llms_verifier` at `3951010c` ("fix(HXC-268): compose
dial addresses via a bracketing helper at fifteen sites")
**Nature:** research and documentation only — no source, test, or tracker file modified.

One pass covers the family because the failure modes interact: HXC-280 exists
*precisely because* HXC-268's fix would make its seven sites worse. Separate
passes would each have seen one facet.

---

## 0. Ground truth established by reading the code

### 0.1 The shared helper

`submodules/llms_verifier/llm-verifier/pkg/helixendpoint/helixendpoint.go` (527–724):

- `unbracket(host)` strips one `[...]` layer.
- `DialAddress(host, port)` = `net.JoinHostPort(unbracket(host), strconv.Itoa(port))`
  — no validation, no placeholder, no RFC 6874 encoding, by design.
- `normalizeHost(host)` (588–672) does the interesting part:

```go
bare, zone := host, ""
if i := strings.Index(host, "%"); i >= 0 {          // FIRST '%'
    bare, zone = host[:i], host[i+1:]
    if rest := strings.TrimPrefix(zone, "25"); rest != "" { zone = rest }
}
if strings.Contains(bare, ":") {                     // checked against `bare`
    if net.ParseIP(bare) == nil { return "", false }
    if zone != "" { return bare + "%25" + zone, true }
    return bare, true
}
if zone != "" { return "", false }
return bare, true                                     // returns `bare`
```

- `BaseURL(host, port)` calls `normalizeHost`; on failure substitutes
  `DefaultHost` ("localhost") **and keeps the caller's port**.

All three HXC-284 killer inputs were traced directly against this code and check
out exactly as filed:

| input | behaviour | mutant behaviour |
|---|---|---|
| `fe80::1%25e%25h` | accepted (splits at first `%`) | rejected by `ParseIP` (splits at last) |
| `1.2.3.4%eth0:5` | rejected (`Contains(bare,":")` false) | **wrongly accepted** if checking `host` |
| `abc%` | trailing `%` silently stripped via the `zone == ""` fallthrough | `return host` keeps the dangling `%` |

### 0.2 HXC-283's exact code

`llm-verifier/api/middleware.go:347-370` (`getClientIP`): for a direct
(no-proxy) IPv6 connection, `r.RemoteAddr` is `[2001:db8::1]:54321` (Go always
brackets). `strings.LastIndex(r.RemoteAddr, ":")` **correctly** finds the port
separator — this is not a mis-split — but the slice **retains the brackets**:
`[2001:db8::1]`. The `X-Forwarded-For` / `X-Real-IP` paths return the header
value verbatim, typically unbracketed (`2001:db8::1`).

Same client, two different bucket keys depending on whether a proxy header is
present. The divergence is **systematic**, not occasional.

### 0.3 HXC-280's hidden seventh site — located

`submodules/llms_verifier/internal/messaging/rabbitmq/config.go:104-128`
(root module, **not** `llm-verifier/`), `Config.AMQPURI()`:

```go
uri += c.Username; if c.Password != "" { uri += ":" + c.Password }   // unescaped
uri += c.Host                                                        // unbracketed
if c.Port != 0 { uri += ":" + intToString(c.Port) }                  // hand-rolled
```

Matches HXC-280's description exactly: userinfo + unbracketed host + unescaped
password + trailing vhost segment. The hand-rolled `intToString` is why grep
sweeps for `fmt.Sprintf` missed it.

The root `go.mod` carries `replace digital.vasic.llmsverifier => ./llm-verifier`,
confirming `helixendpoint` is mechanically reachable from here even though
nothing in this module imports it yet.

### 0.4 A fresh count-drift, on the item that already documents one

- HXC-268's ticket and commit say **fifteen** sites, scoped only to
  `capabilities/config_generator.go`. Its own scope note defers "several
  per-agent config writers and two unrelated subsystems" as deliberately
  out of scope.
- `helixendpoint.go`'s package doc (same commit) says the census is **31**
  total — "24 routed through this package by HXC-268, 7 tracked under HXC-280."
- A direct grep for `helixendpoint\.(DialAddress|BaseURL)\(` today returns **30**
  call sites: 15 in `capabilities/config_generator.go`, 12 in `pkg/cliagents/*.go`,
  and 3 single `DialAddress` sites in `auth/ldap.go`, `events/email_notifier.go`,
  `scoring/alert_manager.go`.
- `git show --stat 3951010c` confirms the commit touched all of those — i.e. the
  landed fix **already covers the files the ticket calls out-of-scope**. Verified
  concretely for `kilocode.go`, converted in that same commit from
  `fmt.Sprintf("http://%s:%d/v1", …)` to `helixendpoint.BaseURL(…)`.
- `pkg/cliagents/generator.go`'s 6 sites were not touched by this commit at all —
  they already called `BaseURL` from HXC-250 and inherited the fix for free.

**15 / 24 / 30 / 31 do not agree.** Reconcile with a fresh recount before
HXC-268 closes; its own acceptance criteria require the census command re-run.
This is the second instance of a failure mode the item's own text already warns
about once.

---

## 1. Standards — all fetched directly, none from memory

- **RFC 3986 §3.2.2**: `IP-literal = "[" ( IPv6address / IPvFuture ) "]"`.
  Brackets exist solely to disambiguate address-internal colons from the port
  separator. RFC 3986 explicitly states its grammar "does not support IPv6
  scoped addressing zone identifiers" at all. §2.1: `pct-encoded = "%" HEXDIG
  HEXDIG` (strict); §2.4 forbids double-decoding.
- **RFC 6874** (URI form): `IPv6addrz = IPv6address "%25" ZoneID`,
  `ZoneID = 1*(unreserved / pct-encoded)`. Rationale: `%` is always a URI escape
  character, so the raw zone delimiter must be spelled `%25`. Example: raw
  `fe80::a%en1` → URI `[fe80::a%25en1]`. **No guidance on empty zones, doubled
  delimiters, or other malformation** — a genuine documented gap in the standard.
- **RFC 4007 §11** (raw/socket form): `<address>%<zone_id>`. The character set
  for `zone_id` is loosely specified ("must not conflict with the delimiter").
  **The RFC does not state whether `zone_id` may be empty or contain `%`,** and
  specifies no behaviour for malformed representations beyond calling a zone on
  a global address "meaningless."
- **Go stdlib**: `net.SplitHostPort` supports `host%zone:port` and
  `[host%zone]:port`, strips brackets, and returns hard errors — never a partial
  or repaired string. `net.JoinHostPort` does **not validate** its host argument;
  it brackets on colon-presence and nothing more.

### The load-bearing new finding

Verified against literal Go source (`golang/go` master, `src/net/ipsock.go`):

```go
func splitHostZone(s string) (host, zone string) {
    // The IPv6 scoped addressing zone identifier starts after the
    // last percent sign.
    if i := bytealg.LastIndexByteString(s, '%'); i > 0 {
        host, zone = s[:i], s[i+1:]
    } else { host = s }
    return
}
```

**Go's own zone-splitting convention splits at the LAST `%`, not the first.**
`normalizeHost` splits at the FIRST (`strings.Index`). For a doubled-`%` input
the two conventions disagree about which substring is the zone — so the guard's
currently-green reading of `fe80::1%25e%25h` is not self-consistent with the
stdlib its own module depends on. This is grounding HXC-284 did not have: the
rule is not merely unpinned, its chosen convention is arguably the wrong one.

---

## 2. Real-world precedent

- **CVE-2026-39361** (OpenObserve, GHSA-gcwf-3p7h-wm79): an SSRF filter compared
  `url.Parse(...).host_str()` — `[::1]`, bracketed — against a bare `::1`
  blocklist entry. Bypass, because brackets were not stripped before comparison.
  Fix: normalise before comparing. **This is HXC-283's exact defect shape,** as a
  security check rather than a rate-limit key.
- **CVE-2026-25679** (Go `net/url`, fixed 1.26.1 / 1.25.8, CVSS 7.5):
  `net/url.Parse` used to tolerate garbage before an IPv6 IP-literal instead of
  rejecting it. The 2026 fix direction was reject-outright. Direct, dated,
  same-language precedent for reject-don't-repair in this exact domain.

---

## 3. The four questions

### Q1 — Zone identifiers

RFC 4007 (raw, one `%`, no encoding) and RFC 6874 (URI, `%25` + restricted zone)
are the two conventions in play; neither says what a *second* `%` means.
`normalizeHost`'s `TrimPrefix(zone, "25")` is an attempt to accept **both**
conventions through one field.

Original analysis (§11.4.8 — no external source addresses this directly): that
dual-convention leniency is *why* the doubled-delimiter case is inherently
ambiguous rather than merely untested. Accepting either convention removes the
external signal that would disambiguate a second `%`. Separately and more
concretely: the first-`%` choice disagrees with Go's own `splitHostZone`.

### Q2 — Reject versus repair

The guard rejects `host:port`, scheme-ful URLs, and `/?#@`-bearing hosts
outright, but silently strips a trailing empty zone marker (`abc%` → `abc`) —
the inconsistency HXC-284 flags.

Two independent angles point the same way: the IETF-hosted critique of Postel's
robustness principle (leniency hides errors until a less-tolerant downstream
system meets them), and Go's own 2026 `net/url` tightening from tolerant to
reject. Neither states a universal law that reject always beats repair; both
support it specifically for security- and identity-relevant parses, which host
resolution and rate-limit identity both are.

### Q3 — The default-substitution asymmetry

**No external source names this pattern** — substituting a default host while
keeping the caller's port, correct for services you own and wrong for ones you
don't. Closest fits, each checked and found imperfect:

- **CWE-636** (Not Failing Securely) concerns reverting to a globally weaker
  *mode*, not substituting one field while preserving another.
- **Saltzer & Schroeder fail-safe-defaults** (1975, via secondary
  characterisation) is the closest general precedent but is generic, not
  this-shaped.
- The **OWASP SSRF Prevention Cheat Sheet**, fetched directly, does not address
  own-service-versus-third-party validation asymmetry or fail-open/closed
  defaulting at all — a genuine negative finding.

**Per §11.4.8: NO external solution found for the asymmetry as HXC-280 states
it — the item's own framing is original work.**

### Q4 — The malformed-input contract for rate limiting

`net.SplitHostPort` treats malformed input as a hard error, never a partial
repair.

**OWASP API4:2019**, fetched directly, gives zero guidance on IP normalisation
or malformed-address handling — negative finding.

**A frequently-cited "OWASP ASVS 1.14.8" control could not be verified.** Direct
fetches of ASVS 5.0's actual chapter files found no such control at that number,
and the 5.0 numbering scheme has no reachable "1.14" section. Flagged as an
unconfirmed, likely-inaccurate secondary-search claim — **not reported as fact.**

Best-supported inference (original, not a cited mandate): a malformed caller
address should map to its own explicit, distinguishable rejection — never the
same shared bucket as every other malformed input (bypass shape), and never
silently repaired into someone else's identity (the collision half of HXC-283's
own bug). **CWE-180** (Validate Before Canonicalize) is the relevant weakness
class for `normalizeHost`'s canonicalise→validate→canonicalise→validate
interleaving, though no concrete exploit of it was found in this file.

---

## 4. Is the family completely scoped? Two findings

**4.1 — Inside scope:** the 15/24/30/31 count-drift on HXC-268 itself (§0.4).

**4.2 — Outside scope, and the stronger signal:** the 31-site census is
explicitly scoped to "both modules of this repository." `submodules/helix_agent`
— the *consumer* of the endpoint this family fixes — was never in that census.
A read-only survey there found roughly **30 further unfixed sites of the
identical defect**, including service classes named verbatim in HXC-280's own
description:

| site | service class | in HXC-280's named list? |
|---|---|---|
| `internal/search/store/qdrant.go:46`, `chroma.go:45` | search databases | "a search database" |
| `internal/streaming/flink/config.go:127` | stream processor | "a stream processor" |
| `internal/cache/redis.go:56` | database | one of "two database systems" |
| `internal/messaging/broker.go:382-384` | message broker — unescaped password + unbracketed host in an AMQP URI | "a message broker"; a **second independent copy** of the bug HXC-280 tracks once |
| `cmd/helixagent/main.go:1948,4460,4508` | endpoint composition | structurally identical to sites HXC-268 just fixed, reimplemented independently |
| `internal/ports/ports.go:439,444`, `internal/llm/providers/helixllm/provider.go:132` | general helpers | same shape |
| `challenges/codebase/go_files/{opencode,crush,kilocode}_generator/*.go` | 21 sites, 3 files | independent duplicate copies of generator logic `llms_verifier` already fixed |

`qdrant.go` / `chroma.go` take host and port as constructor parameters
(defaulting only when empty) — configuration-reachable, not hardcoded, exactly
like the sites already fixed.

**This is the strongest evidence that the four-item family, scoped to one
submodule, is not the full extent of the defect class.** Not triaged here; a
scoping decision for the family's owner.

---

## 5. Per-item mapping

- **HXC-268** — §0.4 bears on whether it can close as filed (its own acceptance
  criteria require the census command re-run and reconciled). §1 and §3-Q1 bear
  on `normalizeHost` correctness, which all ~30 sites route through.
- **HXC-280** — §0.3 confirms the seventh site's exact shape. §3-Q3 is the
  load-bearing answer: no named external pattern; original work. §4.2 shows an
  independent duplicate of the same AMQP-URI bug in `helix_agent`.
- **HXC-283** — §0.2 confirms the bracket-retention mechanism precisely (correct
  split point, missing strip) and that the divergence is systematic. §1 and
  §3-Q4 ground the "standard routine" call and the malformed-input answer. §2's
  CVE-2026-39361 is the closest real precedent, same address shape.
- **HXC-284** — §0.1 independently re-verifies all three killer inputs against
  current source. §1's `splitHostZone` finding is new grounding: the guard's
  `Index` choice is inconsistent with Go's own stdlib convention. §3-Q1 explains
  why the doubled-marker case is the fragile one. §2 answers the item's
  "related observation" about the trailing-zone-marker inconsistency.

---

## Sources verified

- [RFC 3986 §3.2.2 / §2.1 / §2.4](https://www.rfc-editor.org/rfc/rfc3986.html) — accessed 2026-08-12
- [RFC 6874](https://www.rfc-editor.org/rfc/rfc6874.html) — accessed 2026-08-12
- [RFC 4007 §11](https://www.rfc-editor.org/rfc/rfc4007.html) — accessed 2026-08-12
- [pkg.go.dev/net — SplitHostPort / JoinHostPort](https://pkg.go.dev/net#SplitHostPort) — accessed 2026-08-12
- [golang/go `src/net/ipsock.go` (raw, master)](https://raw.githubusercontent.com/golang/go/master/src/net/ipsock.go) — accessed 2026-08-12
- [GHSA-gcwf-3p7h-wm79 (CVE-2026-39361)](https://github.com/openobserve/openobserve/security/advisories/GHSA-gcwf-3p7h-wm79) — accessed 2026-08-12
- [golang/go#77578 (CVE-2026-25679)](https://github.com/golang/go/issues/77578) · [Go security announcement](https://groups.google.com/g/golang-announce/c/EdhZqrQ98hk) · [pkg.go.dev/vuln/GO-2026-4601](https://pkg.go.dev/vuln/GO-2026-4601) — accessed 2026-08-12
- [CWE-636](https://cwe.mitre.org/data/definitions/636.html) · [CWE-180](https://cwe.mitre.org/data/definitions/180.html) — accessed 2026-08-12
- [draft-thomson-postel-was-wrong-02](https://datatracker.ietf.org/doc/html/draft-thomson-postel-was-wrong-02) — accessed 2026-08-12
- [OWASP API4:2019](https://owasp.org/API-Security/editions/2019/en/0xa4-lack-of-resources-and-rate-limiting/) — accessed 2026-08-12
- [OWASP SSRF Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html) — accessed 2026-08-12
- [Saltzer & Schroeder — secondary characterisation](https://handwiki.org/wiki/Saltzer_and_Schroeder's_design_principles) — accessed 2026-08-12 (primary 1975 text not independently fetched)
- OWASP ASVS 5.0 chapter listing plus direct fetches of
  `0x10-V1-Encoding-and-Sanitization.md` and `0x13-V4-API-and-Web-Service.md` —
  used to attempt, and fail, verification of a claimed "1.14.8" control.
  **Flagged unconfirmed; not reported as fact.** — accessed 2026-08-12

**Repository evidence:** `submodules/llms_verifier` at `3951010c`;
`llm-verifier/pkg/helixendpoint/helixendpoint.go`; `llm-verifier/api/middleware.go`;
`internal/messaging/rabbitmq/config.go`; `docs/workable_items.db` (HXC-268/280/283/284
rows); `submodules/helix_agent` read-only survey (files listed in §4.2) — all read
2026-08-12.
