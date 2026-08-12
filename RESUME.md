# RESUME — session resumption record (§11.4.131)

**Rev 14 · rebuilt 2026-08-13 ~00:25 +05.**
Rev 13 predated an entire working session and was unusable; this replaces it.

> **Governing rule for this document.** Every count, list, hash, range and
> dirty-state below is a **snapshot**. Re-derive before acting on any of it —
> commands are given inline. Every narrative claim about code state cites the
> commit it was last checked against, **per sub-assertion, not per sentence**.
> Counts in this project have moved under scrutiny repeatedly; one item carried
> four disagreeing published figures.

---

## 1 · Start here

```bash
cd /home/milos/Factory/projects/tools_and_research/helix_code
git fetch --all --prune
git log --oneline -1                       # expect 7bf5f45e or later
git status --porcelain                     # ~10 entries, all live-stream work
sqlite3 docs/workable_items.db "SELECT COUNT(*) FROM items WHERE status NOT IN ('Fixed (→ Fixed.md)','Implemented (→ Fixed.md)','Completed (→ Fixed.md)','Obsolete (→ Fixed.md)');"
```

Read `.remember/remember.md` and `docs/CONTINUATION.md` first if present.

## 2 · State at snapshot

| repo | HEAD | published |
|---|---|---|
| main | `7bf5f45e` | all mirrors verified |
| `submodules/helix_agent` | `36009d62` | both mirrors verified |
| `submodules/llms_verifier` | `9bf5457d` | both mirrors verified |
| `submodules/helix_qa` | `0634b1b` | all three mirrors verified |

Tracker: **514 items, 107 open** — 2 Critical, 30 High, 44 Medium, 31 Low.

**The 2 Critical:** `HXC-227` (published provider key, `Operator-blocked`,
unblock choices enumerated — **only the operator can clear it**) and `HXC-243`
(`Ready for testing`, fix landed at `0634b1b`, closure blocked on the §11.4.150
research gate). I misreported this as "1 Critical" repeatedly during the
session by reading the fix's landing as its closure; corrected here.

## 3 · Live streams — uncommitted work on disk

Four agents were mid-flight at snapshot. **Their work is preserved but
uncommitted.** Verify by hash before resuming any of them; a marker grep will
NOT detect a deletion, and it is separately blind above ~12KB on this host
(HXC-309).

| item | scope | state |
|---|---|---|
| HXC-282 | meta `scripts/git_hooks/`, `scripts/lib/` | **round 7.** Round 6 NO-GO. |
| HXC-286 | `helix_agent` (9 files + new `internal/netaddr/`) | remediating 3 must-fixes |
| HXC-298 | `llms_verifier` (`clientip/`, `enhanced/enterprise/`) | remediating an exploitable blocker |
| HXC-305 | `helix_qa/pkg/testbank/` | in re-review after a merge redesign |

**HXC-282 is the hardest and will not land quickly.** Six consecutive rounds
have each found a *false safety comment* in that file. Round 6's structural
blocker: **the round's own deliverables cannot be committed through the round's
own hook** — a file about mutation residue necessarily contains the marker
strings, and the library is production code that cannot satisfy the exemption's
restore-idiom condition. Solve that first; it constrains everything else.

## 4 · Landed and published this session (10)

`HXC-243` (the release gate) · `268` · `272` · `274` · `281` · `283` · `287` ·
`291` · `292` · `299`.

**None is CLOSED.** §11.4.150 requires a documented multi-angle research pass
before any item may be marked fixed, and only the address-composition family
has one (`docs/research/address_composition_family_20260812/`). HXC-268
additionally cannot close until HXC-288 reconciles its four disagreeing counts.

## 5 · Path to a release build

**Mine:** land the four in-flight items → full-suite retest **on a quiescent
host** (running it with streams live measures contention, not correctness) →
research passes for closure.

**Operator only:** rotate `HXC-227` and `HXC-168` · §11.4.185 manual-QA
sign-off · the 1.1.0 retro-tagging decision for three submodules.

**Two decisions, neither blocking:** `HXC-310` — mount the rate limiter (with
`SetTrustedProxies` configured, or it will not limit anything) or record it as
deliberately absent. `HXC-318` — the same question for five further middlewares.

## 6 · The finding that most changes what a green build means

**HXC-318 (High).** The only production HTTP server registers its handlers on a
bare `ServeMux` wrapping **nothing**. `api` defines **five** middlewares and
applies **zero**; three have no call site anywhere, *not even tests*.
`security.NewSecurityManager` has zero callers at all.

Six items filed High on a defect's *shape* proved to sit in code that never
runs — the enterprise subsystem (JWT auth + RBAC + audit log), the rate limiter
(`HXC-310`), the audit trail (`HXC-299`), an enterprise rate-limit stub
(`HXC-314`), and this. Each was restated as evidence arrived.

For the release record, **both halves are true**: most vulnerabilities found
this session were never reachable, **and** the protections a reader would
assume are in force are applied to no request. The tests certify each mechanism
in isolation, so nothing reports the gap.

## 7 · Corrections carried forward — do not re-inherit the old versions

- **The address defect breaks on the UNBRACKETED form.** A bracketed host
  composes correctly under `Sprintf` *by coincidence*. The `[::1]:6333:6333`
  shape cited all session requires a host that **already carries a port** — a
  different input. The shape that breaks a bracketed host is
  `JoinHostPort("[::1]","6333")` → `[[::1]]:6333`, which is why `unbracket()`
  exists. The old framing conflated two inputs; several items need restating.
- **HXC-309 is real but host-dependent.** Specific to bash's **builtin**
  `printf`, threshold set by pipe-buffer size — 8192 bytes here, boundary
  pinned at 12288 MATCH / 12400 NO-MATCH. Not the universal "~16KB" first filed.
- **`grep` is a wrapper function** in interactive shells and does **not**
  propagate into a script run as a file. It produced false results four times
  in one session. Carry a positive control on every negative.
- `cmd | head && echo $?` reports **head's** status. Two false greens came from
  this, both mine, one on the release-critical build question.
- A count is a LEAD, never a fact: 16→32, 15/24/30/31 unreconciled, 46→50,
  "133 passing" measured as rc=1.

## 8 · Standing hazards

- **HXC-247 stays unassigned until HXC-248 closes.** `:8100` is held by
  `llm-verifier`, not the agent; HXC-248's teardown guard probes `:8100` and so
  reaches the wrong process. That mismatch is the coincidence protecting 41+
  live containers. Fixing HXC-247 first would ARM the teardown.
- Live stack: 41+ containers. **`helixllm-coder` holds a 30B model — never
  restart it.** No sudo, rootless podman only, no host power-state change, no
  force-push (§11.4.113, absolute).
- Never fire real provider calls — startup discovers live credentials
  (§11.4.101). Use `httptest` loopback or a closed port.
- `git add -A` is forbidden in a repo with live streams (§11.4.30). It nearly
  swept a half-finished pre-commit hook into an unrelated commit.
- ~20 `tools/opensource/*` pointer drifts in `helix_qa` are pre-existing and
  **not** ours.

## 9 · What worked, and is worth continuing

Independent review caught, **before landing**: a lock recording dead pids; a
change that would have deleted 49 regression guards while all its own tests
passed; an exploitable validation guard that was pure decoration; and a fix
that regressed against its own sibling one import away.

Fourteen times this session the defect lived in **the artifact's description of
itself** rather than its logic — a promise the code did not keep, a comment
asserting a false mechanism, a count disagreeing with its own package doc.
Every one passed its own tests. Every one was caught by an independent party
checking claim against code.

Reviews run on Fable at xhigh (§11.4.209); **Fable hit its limit during this
session, so they now dispatch to Opus at xhigh**, which is that clause's
specified fallback.
