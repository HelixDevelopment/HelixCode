# Round-5 review remediation — captured evidence

Captured 2026-08-11T02:45:00Z. Commit under review: `f10c9a1e` (the round-4
remediation). Main repo HEAD at start of this work: `84853455`.

The round-5 independent review determined the §11.4.134 loop CANNOT close on
`f10c9a1e`: 2 MEDIUM, 2 LOW and 2 NITs. Each is dispositioned below with the
measurement that closes it — or, where it is not closed, with the reason.

**This file supersedes the round-4 record**
(`docs/qa/r4_review_remediation_20260810T191500Z/EVIDENCE.md`) wherever the two
disagree, and it supersedes `f10c9a1e`'s commit message, which cannot be
rewritten (§11.4.113). The round-4 file has been annotated in place with
`ROUND-5 CORRECTION` blocks at each point it is now known to be wrong.

Six of the eight round-4 dispositions were independently re-verified by the
reviewer and are NOT in scope here; they were not touched.

> ### ⚠ PARTIALLY SUPERSEDED BY THE ROUND-6 RECORD
>
> A round-6 independent review found that the F3 census table below (under
> "Measurement (I re-took it rather than inherit the count)") undercounts the
> systemic gap: two of its three "invoked by" rows fail the table's own
> "executable code, excluding documentation references" test. Corrected in
> place, in the F3 section (search for `ROUND-6 CORRECTION`):
>
> - **the F3 wired-count.** Only **1 of 14** files under `scripts/tests/` had
>   a genuine invocation site before this commit (`f10c9a1e`); this commit
>   (`ca740516`) wired a second, bringing it to **2 of 14**. The systemic
>   unwired gap is **12 files, not 11** and not 3-wired-so-11-unwired as
>   originally recorded. `HXC-263` in `docs/Issues.md` already reads "Twelve
>   of the fourteen" and needs no change.
>
> Every other disposition in this file (F1, F2, F4, F5, F6, and the closing
> sweep/battery figures) was independently re-verified by round 6 and is
> unchanged; only the F3 census table's derived counts were wrong.

---

## Live-stack dependency — stated up front (§11.4.6)

All five service ports were listening for every measurement below
(`:7061 :8100 :8111 :5433 :6380`), and `https://localhost:8443/internal/health`
answered `rc=0`. Which results depend on that:

| Result | Depends on the live stack? |
|---|---|
| Falsification battery — 36 of 42 assertions | **No.** Self-constructed stubs + transient units. |
| Falsification battery — final 6 assertions (`LIVE STACK` block) | **Yes.** Absent the gateway they are an honest §11.4.3 SKIP and the total reads 36. |
| Sweep G30 / G31 / G32 | **Yes.** They read the live gateway process, completion path and health endpoint. |
| Sweep G22 / G25 / G26 | **Yes** (runtime-probed against running containers/services). |
| Sweep — the other 26 gates | **No.** |
| Every F1 result (meta-test + all 6 mutations) | **No.** Pure static text analysis. |
| The rc probes (52 / 56 / 60 / 35 / 28) | **No.** Local stubs on loopback. |

---

## F2 (MEDIUM, primary) — the SKIP fix was mislabeled, and the rc it claimed was the one still fail-open

### Root cause (§11.4.102)

The `empty` stub called `close()` immediately after `accept()`, never reading the
request. Closing a socket while data is still unread in the receive queue
obliges the kernel to send **RST** rather than FIN, so curl reports **56**
(recv failure: connection reset by peer), not **52** (empty reply from server).

Measured directly against the stub as committed, and against a `recv()`-first
variant, 3 runs each:

```
shape=empty       run=1 curl_rc=56    shape=empty_recv  run=1 curl_rc=52
shape=empty       run=2 curl_rc=56    shape=empty_recv  run=2 curl_rc=52
shape=empty       run=3 curl_rc=56    shape=empty_recv  run=3 curl_rc=52
```

### Why it survived — the deeper defect

`assert 1 <label>` records that the guard **FAILed**. It does not record **which
rc** produced that FAIL, and the guards FAIL identically on 28, 35, 52, 56 and
60. So no assertion in the battery could distinguish 52 from 56, and the label
was free to drift from the wire with every line green. The mislabel was not a
typo that slipped through review — it was **unobservable to the instrument**.

That left rc 52 — the member the batch reported as closed — as the one still
fail-open, exactly as the reviewer measured.

### Disposition: FIXED, in four parts

1. **`empty` now drains the request before closing** → genuine clean FIN → rc 52.
2. **A new `reset` shape** preserves rc-56 coverage (which the mislabeled stub
   had been providing accidentally). It `select()`s for the request to arrive
   and then closes it **unread** — the wait is load-bearing, see the flake note
   below.
3. **rc 60 is covered through the classifier**, because it has no network
   subject. Both guards pass `-k`, which waives peer verification outright, so
   no TLS listener can make *them* see 60. Measured against a self-signed
   listener:

   ```
   -- with -k (the guards' actual flags):   rc=0
   -- WITHOUT -k (self-signed, untrusted):  rc=60
   ```

   That is a §11.4.112 structural fact, not a missing fixture. Constructing an
   elaborate TLS subject and asserting FAIL would silently re-measure some other
   rc — precisely the mislabel being fixed. What the closed set actually claims
   is a property of the **classifier** ("SKIP iff rc ∈ {6,7}"), so the
   classifier is driven directly with a `curl` stub that exits a chosen code —
   the same technique the battery's R3 case already uses for the analyzer.

   **The injection is self-validating**, which matters more than the injection:
   each case is aimed at a subject whose *real* rc yields the *opposite*
   verdict. rc60 is asserted against a **closed port** (real curl → 7 → SKIP),
   so observing FAIL proves the stub ran; rc6 is asserted against a **live
   stub** (real curl → 52 → FAIL), so observing SKIP proves it in the other
   direction. A PATH injection that failed to take would flip both to NOT OK.

4. **Each subject's rc is now MEASURED and pinned** by a new `assert_rc()`
   before its verdict is asserted, using the same curl flags the guards use.
   This is the anti-recurrence mechanism: a label and its wire can no longer
   drift apart silently.

```
--- each subject's rc is MEASURED before its verdict is asserted ---
  ok    wedge stub really yields rc28 (timeout)              curl rc=28
  ok    empty stub really yields rc52 (empty reply)          curl rc=52
  ok    reset stub really yields rc56 (peer reset)           curl rc=56
  ok    plaintext listener really yields rc35 (TLS)          curl rc=35
  ok    233: rc28 timeout (listener accepted, then wedged)   FAIL(1)
  ok    244: rc28 timeout (listener accepted, then wedged)   FAIL(1)
  ok    233: rc52 empty reply (read, then zero bytes)        FAIL(1)
  ok    244: rc52 empty reply (read, then zero bytes)        FAIL(1)
  ok    233: rc56 peer reset (closed with request unread)    FAIL(1)
  ok    244: rc56 peer reset (closed with request unread)    FAIL(1)
  ok    233: rc35 TLS handshake vs plaintext listener        FAIL(1)
  ok    244: rc35 TLS handshake vs plaintext listener        FAIL(1)
  ok    233: rc60 TLS cert -> FAIL (vs a CLOSED port)        FAIL(1)
  ok    244: rc60 TLS cert -> FAIL (vs a CLOSED port)        FAIL(1)
  ok    233: rc6 unresolvable -> SKIP (vs a LIVE stub)       SKIP(2)
  ok    244: rc6 unresolvable -> SKIP (vs a LIVE stub)       SKIP(2)
```

### M3 re-run — the closed set is now covered end to end

Reviewer mutation M3 broadens the SKIP set by one rc. Run in an **out-of-repo
mirror** (`scripts/testing/` copied to scratch; the battery derives `ROOT` from
its own path), once per declared member, guards restored and `diff`-verified
after each:

| M3 broadens SKIP to | before (at `f10c9a1e`) | after |
|---|---|---|
| rc 28 | caught | **exit 1** — 40 ok, 2 not ok |
| rc 35 | caught | **exit 1** — 40 ok, 2 not ok |
| **rc 52** | **32 ok, 0 not ok — UNDETECTED** | **exit 1** — 40 ok, 2 not ok |
| rc 56 | not asserted as 56 | **exit 1** — 40 ok, 2 not ok |
| rc 60 | not covered | **exit 1** — 40 ok, 2 not ok |

Every mutation is caught by exactly the assertion that names its rc, e.g.:

```
== M3(rc60) NOT OK lines ==
  NOT OK 233: rc60 TLS cert -> FAIL (vs a CLOSED port)       expected FAIL(1), got exit 2
  NOT OK 244: rc60 TLS cert -> FAIL (vs a CLOSED port)       expected FAIL(1), got exit 2
```

```
=== restore check ===
  mirror restored identical to tracked
```

### A flake I introduced, found by reading lines instead of counts

The first M3 matrix reported **3** not-ok for rc52 where 2 were expected. The
third was `244: rc56 peer reset ... got exit 2` — my new `reset` stub had
produced 52 on that connection. A bare `close()` **races the client's send**:
when the server wins, nothing is unread, the close is a clean FIN, and curl
reports 52 — the exact 52/56 confusion the shape exists to distinguish.

Fixed by `select()`ing for the request before closing. Re-measured under
deliberate CPU load, 30 sequential connections per shape:

```
shape=reset rcs:56  | distribution: 30 56
shape=empty rcs:52  | distribution: 30 52
```

Had I trusted the count and not read the matched line, this would have shipped
as an intermittent false FAIL (§11.4.201).

### Battery totals

Deterministic across three consecutive post-fix runs:

```
=== falsification battery: 42 ok, 0 not ok ===   (exit 0, ×3)
```

32 → 42: +4 rc pins, +2 for the rc56 `reset` subject, +2 rc60, +2 rc6.

---

## F1 (MEDIUM) — the HIGH's guard did not falsify the defect class

### Root cause (§11.4.102)

All four assertions in `sync_gate_direction_neutrality_meta_test.sh` were
**token-presence** checks. The defect class is a **binding**: which remedy hangs
off which condition, and which one carries the data-loss warning. Presence of
the right words is not advice; the pairing is the advice.

### RED — the mutation survives on the shipped guard

Mutation M1 swaps the two remedies inside `drift_remedy()`, leaving
`(a) if the DB is the newer side ... run 'sync md-to-db'` — the destructive
prescription bound to the exact condition under which it deletes the SSoT —
with the WARNING now pointing at the safe remedy. Both tokens still present:

```
=== RED: current meta-test vs M1-mutated gate ===
  ok    no bare 'is STALE vs' directional assertion
  ok    names the db-to-md remedy (DB newer -> regen docs)
  ok    names the md-to-db remedy (docs newer)
  ok    warns that one remedy overwrites the SSoT
--- 4 ok, 0 not ok
GREEN (HXC-252): ...
exit=0
```

> My first M1 attempt was a bad instrument and produced a **false RED**: a
> naive two-step `sed` applied its second substitution to text the first had
> already rewritten, deleting `md-to-db` instead of swapping. The presence check
> caught *that*, which looked like the guard working. Redone as a true swap via
> a sentinel (both tokens verified still present, 1 each) before being believed.

### Disposition: FIXED — association, not presence

The message is flattened to the sentence a maintainer actually reads, then each
anchor is resolved to the remedy it governs **the way the prose reads**:

- `selects(cond)` = first remedy **after** the condition ("if X … run R"),
  falling back to the nearest preceding one;
- `attaches(warn)` = last remedy **before** the warning, since a warning
  back-refers, falling back to the first following one.

Three new assertions on top of the existing four (nothing removed, §11.4.120):

- **A** the overwrite warning must attach to `md-to-db`;
- **B** the DB-is-newer condition must select `db-to-md`;
- **C** the doc-is-newer condition must select `md-to-db`.

Raw nearest-neighbour by character distance was tried first and is **weaker**: a
condition at the start of its clause can sit closer to the *previous* clause's
remedy token than to its own, and a condition-swap scored one check satisfied
purely on distance. Measured, not assumed — the forward rule catches that
mutation on two checks instead of one.

### GREEN, and the full mutation matrix

```
GREEN  fixed gate                     exit=0  7 ok, 0 not ok
M1     remedy swap                    exit=1  4 ok, 3 not ok
M1b    condition swap                 exit=1  5 ok, 2 not ok
M1c    warning removed                exit=1  (loud: "cannot verify remedy/condition BINDING")
M6     one drift branch un-wired      exit=1  (pre-existing wiring check, still catches)
RED    on pre-fix gate (be5d56be)     exit=0  reproduces
RED    on fixed gate                  exit=1  does not reproduce
```

`M1c` matters as much as `M1`: removing the warning makes the binding
**unverifiable**, and an unverifiable binding is reported loudly rather than
scored as a pass — a check that matched nothing must not report clean for the
same reason a satisfied one does (§11.4.201).

§11.4.120 confirmed: the pre-existing `M6` wiring assertion and both RED
polarities still behave exactly as before. No assertion was weakened into a
tautology; three were added.

---

## F3 (LOW, systemic) — the guard had zero invocation sites

### Measurement (I re-took it rather than inherit the count)

14 files in `scripts/tests/`. Invocation sites in **executable** code, excluding
documentation references:

| meta-test | invoked by |
|---|---|
| `cross_platform_parity_meta_test.sh` | `scripts/gates/cross_platform_parity_gate.sh` |
| `obsolete_details_meta_test.sh` | `scripts/gates/obsolete_details_gate.sh` |
| `qa_evidence_citation_coincidence_meta_test.sh` | `scripts/verify-all-constitution-rules.sh` (G27) |
| the other 11 | **none** |

So **3 of 14** are wired into executable code, and **1 of 14** is wired into the
sweep. The reviewer's "1 of 14" is right for the sweep specifically; both
framings are recorded because the difference matters to whoever closes the
systemic gap.

> **ROUND-6 CORRECTION — two of the three "invoked by" rows above fail the
> table's own "executable code, excluding documentation references" test.
> The correct figures are 1 of 14 wired before this commit, 2 of 14 after
> it.**
>
> `cross_platform_parity_meta_test.sh` is **not** invoked by
> `cross_platform_parity_gate.sh`. The only reference to it anywhere outside
> its own file is a self-EXCLUSION case-arm inside that gate's own scan loop
> (`scripts/gates/cross_platform_parity_gate.sh:103`):
> ```
> */cross_platform_parity_gate.sh|*/cross_platform_parity_meta_test.sh) continue ;;
> ```
> — preceded by the comment "Never analyse the gate or its meta-test against
> itself." That is a skip, not a call site; the meta-test is never executed
> by it.
>
> `obsolete_details_meta_test.sh` is **not** invoked by
> `scripts/gates/obsolete_details_gate.sh` either. The only reference is a
> comment at line 29 ("scripts/tests/obsolete_details_meta_test.sh asserts an
> EMPTY …") describing where the paired mutation for that gate's fix lives —
> not a call site.
>
> Both rows were misclassified because the search that produced this table
> matched on basename presence and did not read what the matched line does —
> the same copied-forward-without-re-measuring failure mode that F4 below
> describes for assertion counts, here applied to a grep census instead of a
> test count. The third row, `qa_evidence_citation_coincidence_meta_test.sh`
> invoked by G27 (`scripts/verify-all-constitution-rules.sh:1076`), was
> correctly classified — that invocation predates this commit (blamed to
> `790d097c3`, 2026-07-31).
>
> **Corrected figures.** Before this commit (`f10c9a1e`): **1 of 14** files
> under `scripts/tests/` had a genuine invocation site in executable code
> (`qa_evidence_citation_coincidence_meta_test.sh`, via G27 — the same file
> the sweep-specific "1 of 14" above already named, so that framing was
> already right). This commit (`ca740516`) added the second — the G11
> invocation of `sync_gate_direction_neutrality_meta_test.sh` at
> `scripts/verify-all-constitution-rules.sh:513`, blamed to `ca740516a`
> itself. So **2 of 14** are wired into executable code after this commit,
> and the systemic unwired gap is **12 files, not 11**.
>
> Independently re-derived (2026-08-11, round 6): searched every tracked,
> non-documentation file for each of the 14 basenames under `scripts/tests/`,
> then read every matched line rather than trusting the match count
> (§11.4.201). The only hits outside each file's own definition were these five
> lines — two genuine invocations and three non-invocation references:
>
> - `scripts/verify-all-constitution-rules.sh:513` — genuine invocation (G11)
> - `scripts/verify-all-constitution-rules.sh:1076` — genuine invocation (G27)
> - `scripts/verify-all-constitution-rules.sh:46` — comment (G11 header inventory)
> - `scripts/gates/cross_platform_parity_gate.sh:103` — self-exclusion case-arm
> - `scripts/gates/obsolete_details_gate.sh:29` — comment
>
> The remaining ten files have zero references of any kind, anywhere in the
> tracked non-doc tree.

> **ROUND-7 CORRECTION (2026-08-11).** The paragraph above originally asserted
> "the four lines discussed above — two genuine invocations, two non-invocation
> references (a self-exclusion arm and a comment)". That under-counted by one:
> `scripts/verify-all-constitution-rules.sh:46`, an inert comment inside G11's
> header inventory that names `sync_gate_direction_neutrality_meta_test.sh`, and
> which `ca740516` itself added. Reproduce with:
>
> ```
> git grep -n --fixed-strings sync_gate_direction_neutrality_meta_test.sh \
>   -- ':!docs' ':!*.md' ':!*.html' \
>      ':!scripts/tests/sync_gate_direction_neutrality_meta_test.sh'
> ```
> → returns lines 46 and 513.
>
> **Nothing downstream changes.** The file line 46 names was already classified
> WIRED via its genuine invocation at line 513, so no figure, classification,
> per-file count, or tracker entry moves: 1-of-14 wired before `ca740516`,
> 2-of-14 after, systemic gap 12. HXC-263 remains correct as filed.
>
> **Why the shape changed, not just the number.** The aggregate count in this
> paragraph has now been wrong in two consecutive review rounds (round 6: "3 of
> 14" wired; round 7: "four lines" of references). A free-floating total is a
> claim that can drift silently; an enumerated list is checkable line by line
> against a single grep. The total was therefore replaced by the enumeration
> above rather than merely corrected to five — the same reasoning §11.4.201
> applies to counts ("a count is a lead; the lines are the finding"), applied to
> this document's own claims.

### Disposition: THIS guard WIRED (the systemic gap is NOT closed)

The task permitted wiring this one guard if genuinely low-risk. It is: pure
static text analysis of a tracked file, no network, no DB, no submodule,
deterministic, sub-second.

Wired **inside the existing G11 block** rather than as a new gate id. G27 is the
precedent for a new gate, and the sweep has no hardcoded gate count — but a
concurrent agent was mid-sweep against a 32-gate inventory, and G11 is precisely
the gate whose *advice* this guard protects. Same gate id, one verdict, no
change to the sweep's gate count.

It runs **unconditionally**, not only when the sync check passes: the advice is
static text, and it is most load-bearing exactly when drift EXISTS and a
maintainer is about to act on it. It is reported **ahead of** drift because a
wrong-direction prescription is a §9.2 data-destruction defect while drift is a
hygiene defect.

Falsified in both directions (§11.4.120 — a gate nobody can make fail is not a
gate):

```
G11 with a REGRESSED advice message (M1 mutant via SYNC_GATE_PATH):
  G11  FAIL  the sync gate's DRIFT ADVICE again prescribes a direction it never
             computed — a maintainer who follows it while the DB is the newer
             side DELETES items from the §11.4.95 SSoT (HXC-252, §9.2)

G11 with an UNLOCATABLE drift region:
  FAIL (G11): the drift-advice guard could not RUN (exit 2, §11.4.3 environment
             SKIP) — it certified nothing, so the gate's advice is unverified;
             this is NOT a detected regression
```

**The systemic 11-file gap is NOT closed and is not claimed to be.** It should
be filed as a tracked item. Filing was **not** done in this batch because the
workable-items DB, `Issues.md` and the summary docs were being actively
regenerated by the main stream during this work; adding rows would have
collided with that. Recorded here for filing.

> **ROUND-6 CORRECTION — the gap above is 12 files, not 11**; see the
> corrected F3 measurement above. `HXC-263` (`docs/Issues.md`) already
> reads "Twelve of the fourteen … the remaining twelve" and needs no change.

---

## F4 (LOW) — evidence figures certified an intermediate tree

`27 ok / 2 not ok` = 29 assertions, and `29 ok / 2 not ok` = 31, but the battery
committed at `f10c9a1e` carries **32**. Both captures predate assertions that
landed in the same commit.

**Disposition: CORRECTED** in the round-4 evidence, in place, at each site.

This is a **RECURRENCE** of the "22 ok → 23" LOW that the same batch filed and
corrected. Two instances in one batch make it a pattern, not a slip: counts were
being **copied forward from earlier runs** rather than re-measured against the
tree actually being committed. The mechanical fix is to re-run and re-paste at
staging time, which is what produced every figure in this file.

---

## F5 / F6 (NITs) — both FIXED

**F5** — `G30`-`G32` and the drift paragraph had been inserted **between** G14's
heading and its own continuation lines, orphaning four lines of G14's
description behind an unrelated paragraph. Harmless to `--explain` (it greps
`^#   G[0-9]`, which continuation lines do not match) but wrong for every human
reader. Continuation lines restored under their heading; a short note records
what happened so it is not re-introduced.

`--explain` verified unchanged, including its drift self-report:

```
G14 §11.4.106 docs_chain verify  — governance docs tree in-sync
G30 §11.4.135 HXC-229 guard      — gateway serves in Gin RELEASE mode (live process)
...
NOTE: 15 gate(s) run in this sweep but have NO description above: G15 ... G29
```

**F6** — the gate header's invariant 3 still read "the tracked DB is not stale
relative to the md": a **directional** claim, which is exactly what the body
comment 60 lines below now explicitly refuses to make. The header was still
asserting what the code had stopped asserting. It also described the check as a
comparison against "a fresh md→db of the live docs", which is invariant 2's
mechanism, not invariant 3's — so the line was wrong twice.

Rewritten to state **agreement** and to name the file by path, so it does not
restate the "committed DB" claim that HXC-257 already tracks; a parenthetical
points at HXC-257 for the remaining instances in invariants 1-2 rather than
silently widening this fix's scope.

---

## Sweep and battery, after

```
=== verify-all-constitution-rules.sh summary ===
Gates run: 32
Failures:  0
  G11  PASS  PASS — committed DB validates + md⟷db byte-identical in sync
             + drift advice direction-neutral (7 assertions)
  G14  SKIP  docs_chain engine absent — SKIP-OK
PASS: every implementable gate green — anti-bluff covenant honoured
```

```
=== falsification battery: 42 ok, 0 not ok ===   (exit 0, three consecutive runs)
```

---

## Not closed — stated rather than left for the next reviewer to find

### An intermittent false FAIL in the battery, pre-existing, NOT introduced here

The **first** baseline run at `f10c9a1e` reported `31 ok, 1 not ok`:

```
  NOT OK 229: crash-loop @ RestartSec=1 (rising-counter path) expected FAIL(1), got exit 2
    | SKIP(env): HXC-229 — unit 'gfals-2602880-loopfast' is active but exposes
      no MainPID — there is no process whose environment could be read.
```

Characterised rather than assumed:

- **In isolation: 6/6 correct** (`rc=1`, `SubState=auto-restart`, `MainPID=0`).
- Across all 13 battery runs in this session it appeared **once**, in that first
  run; it did not recur in the 3 post-fix GREEN runs or the 7 mutation runs.

Root cause is a TOCTOU **inside the guard**, not in the battery: `ACTIVE_STATE`
and `MainPID` are read by two separate `systemctl show` invocations. A
`RestartSec=1` `/bin/false` unit is `active` for a vanishingly short window, and
under load the two reads can straddle it — yielding `active` with the pid
already reaped, which falls through to the "active but exposes no MainPID" SKIP.

**Not fixed here.** It is outside the reviewer's eight findings, and the fix
touches the sampling logic of a guard currently under review — a change that
needs its own review rather than being folded into a remediation batch. Recorded
for filing alongside the F3 systemic gap. A round-6 run has roughly a 1-in-13
chance of seeing it; if it appears, it is this, and it is a false FAIL.

### Filing blocked

Both items above need tracker rows. The workable-items DB, `Issues.md` and the
summary docs were being regenerated by the main stream throughout this work
(the G12 clearance), so adding rows would have collided. They are recorded here
instead, and the reason is stated rather than left implicit.

## Sources verified

Measured on this host, 2026-08-11, `curl 8.21.0 (GnuTLS/3.8.10)`; all figures in
this document are from runs against the tree being committed, not carried
forward from earlier runs.

**Round-6 addendum.** The F3 census corrections above (search
`ROUND-6 CORRECTION`) were independently re-derived on this same host,
2026-08-11, against `HEAD=ca740516`, by searching every tracked, non-doc file
for each of the 14 `scripts/tests/` basenames with `git grep` and reading
every matched line (not just counting matches), then confirming the two
invocation sites' commit provenance with `git blame`.
