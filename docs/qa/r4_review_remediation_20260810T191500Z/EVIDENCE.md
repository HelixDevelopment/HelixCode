# Round-4 review remediation — captured evidence

Captured 2026-08-10T19:15:00Z. Main repo HEAD at start: `69573772`.
Commits under review: `be5d56be` (G30-G32 wiring), `04f88a23` (tracker repair).

The round-4 review returned GO on both commits with no blocking defect, but with
1 HIGH, 3 MEDIUM and 4 LOW findings. §11.4.134 terminates only on a clean GO with
zero findings and zero warnings, so each is dispositioned below with the evidence
that closes it — or, where it is NOT closed, with the reason stated plainly.

Two of these findings correct claims that live in COMMIT MESSAGES. Commit
messages cannot be edited without a history rewrite, which is forbidden outright
(§11.4.113). **This document is therefore the correcting record, and it
supersedes the commit-message text it contradicts.** Where the two disagree,
this file is right and the commit message is wrong.

> ### ⚠ PARTIALLY SUPERSEDED BY THE ROUND-5 RECORD
>
> A round-5 independent review found that two classes of figure in THIS file are
> themselves wrong, and both are corrected in place below (search for
> "ROUND-5 CORRECTION"):
>
> 1. **the rc named for the `empty` stub.** It is **56**, not 52. The stub closed
>    without reading the request, which forces RST rather than FIN. Every
>    "rc52" label in the MEDIUM-1 section below described a subject that was
>    actually producing 56.
> 2. **the battery assertion counts.** The `27 ok / 2 not ok` and `29 ok /
>    2 not ok` figures below were captured on INTERMEDIATE trees and do not
>    describe the committed battery, which carried 32 assertions at `f10c9a1e`.
>
> Both are corrected inline. The remediation that fixes the underlying defects
> is recorded in `docs/qa/r5_review_remediation_20260811T024500Z/EVIDENCE.md`,
> which supersedes this file wherever they disagree.

---

## HIGH — the sync gate's advice was a §9.2 data-destruction trap (HXC-252)

`scripts/gates/workable_items_sync_gate.sh` compares a normalized `$ISSUES`
against a normalized DB projection with a SYMMETRIC `diff -q`. A symmetric diff
establishes THAT two sides differ; which side is stale is not in the comparison
and cannot be recovered from it — no mtime is read, no history consulted, no
revision compared. The gate nonetheless asserted a direction and prescribed the
remedy following from it:

```
fail "committed $DB is STALE vs $ISSUES — regenerate it (sync md-to-db) + WAL-checkpoint + recommit"
```

In the DB-newer case — the one actually hit on 2026-08-09 — that diagnosis is
false and the remedy destroys the §11.4.95 SSoT: `md-to-db` replaces the DB with
the derived document. Measured at the time: a 36-item stale doc over a 46-item
DB, deleting 10 items and resurrecting 8 closed ones.

**Disposition: FIXED.** The message is now direction-neutral — it states what was
detected, enumerates both remedies with the condition under which each applies,
and warns which one is destructive. Full rendered text is in §"Rendered message"
below.

### §11.4.115 polarity — RED reproduces on the broken artifact, GREEN on the fixed one

New standing guard: `scripts/tests/sync_gate_direction_neutrality_meta_test.sh`.
It takes `SYNC_GATE_PATH` so RED stays re-runnable against the pre-fix revision
forever, which is required once the fix lands and the broken artifact exists only
in history.

```
$ git show be5d56be:scripts/gates/workable_items_sync_gate.sh > /tmp/prefix_gate.sh
$ SYNC_GATE_PATH=/tmp/prefix_gate.sh RED_MODE=1 bash scripts/tests/sync_gate_direction_neutrality_meta_test.sh
RED confirmed: the gate ASSERTS a direction ('is STALE vs') that a symmetric diff
cannot compute, and never names the db-to-md remedy — so the only advice it gives
is the one that overwrites the §11.4.95 SSoT.
exit=0   (reproduced on the pre-fix artifact)

$ RED_MODE=1 bash scripts/tests/sync_gate_direction_neutrality_meta_test.sh
RED baseline did NOT reproduce: ... This artifact already carries the fix
exit=1   (does not reproduce on the fixed artifact)

$ bash scripts/tests/sync_gate_direction_neutrality_meta_test.sh
  ok    no bare 'is STALE vs' directional assertion
  ok    names the db-to-md remedy (DB newer -> regen docs)
  ok    names the md-to-db remedy (docs newer)
  ok    warns that one remedy overwrites the SSoT
--- 4 ok, 0 not ok
exit=0
```

### The instrument was wrong twice first, and both errors failed OPEN

Recorded because both would have certified the UNFIXED gate as fixed (§11.4.201),
and because the second survived the first correction:

1. Scanning the whole `# (3)` region matched line 78's `"$BIN" sync db-to-md ...`
   — the gate's own INVOCATION of the renderer, not advice to anybody.
2. Scanning every `fail "` message matched `fail "db-to-md on committed DB
   failed: ..."` — a report that a TOOL broke, not a remedy for drift.

Neither string is advice a maintainer can act on, yet both satisfied a naive
`grep db-to-md`. The extraction is now anchored structurally on the
`if ! diff -q ...; then fail ...; fi` blocks (and, post-fix, on the
`drift_remedy()` helper), and additionally asserts the helper is WIRED into both
branches — a perfectly-worded helper nobody calls is not advice.

### The gate still detects real drift — §11.4.120, not a tautology

The pre-existing paired mutation plants a phantom item in `docs/Issues.md`:

```
$ bash scripts/tests/workable_items_sync_meta_test.sh
  PASS: baseline gate PASS (md⟷db in sync) (exit 0)
  PASS: mutated (drift) gate FAILS (exit 1)
  PASS: restored gate PASS again (exit 0)
PASS: gate genuinely detects md↔db drift (§1.1 honoured)
```

### Rendered message (runtime, on the planted drift — not a source grep)

```
CM-WORKABLE-ITEMS-MD-DB-IN-SYNC: FAIL — docs/workable_items.db and docs/Issues.md
DISAGREE. This gate compared them SYMMETRICALLY (diff -q) and therefore CANNOT
tell which side is stale — it did not read mtimes, history, or revisions. DO NOT
run a sync until you have determined the direction from evidence (git log on both
paths; which one a recent commit touched). THEN: (a) if the DB is the newer side
— the usual case, and the one §11.4.95 assumes, since the DB is the source and
the docs are derived — run 'sync db-to-md' + export to regenerate docs/Issues.md
from docs/workable_items.db. (b) ONLY if docs/Issues.md genuinely carries newer
hand-authored content than docs/workable_items.db — run 'sync md-to-db' +
WAL-checkpoint + recommit. WARNING: remedy (b) OVERWRITES the §11.4.95 SSoT with
the derived document; if you pick it while the DB is actually newer, every item
the DB holds and the doc lacks is DELETED and every item the doc still shows as
open is RESURRECTED (§9.2 — take a backup first either way).
```

### Scope boundary

The review also noted that line 59's `cp "$DB" "$TMP/committed.db"` never reads
from git, so a variable named `committed.db` holds WORKING-TREE content. That is
a distinct defect, filed separately as **HXC-257**, and its logic was left
untouched here by design. The new message deliberately names the DB by PATH
rather than calling it "the committed DB", so it does not inherit or restate that
false claim while HXC-257 is open.

---

## MEDIUM 1 — the SKIP contract's negative space was unpinned

Both HTTP guards' headers assert a CLOSED set: curl rc 6 and 7 are SKIP, and
28/35/52/56/60 stay FAIL. The positive half was asserted by the battery; the
negative half never was. The review measured today's behaviour as correct and
still filed it — correctly, because an unasserted invariant is one edit from
being untrue and this one fails OPEN.

**Disposition: FIXED.** Six assertions added to
`scripts/testing/guard_live_service_falsification.sh`, each against a REAL
listener that accepts the connection and then misbehaves, so it is reachable by
construction and can never legitimately reach the SKIP branch:

```
  ok    233: rc28 timeout (listener accepted, then wedged)   FAIL(1)
  ok    244: rc28 timeout (listener accepted, then wedged)   FAIL(1)
  ok    233: rc52 empty reply (connected, zero bytes)        FAIL(1)
  ok    244: rc52 empty reply (connected, zero bytes)        FAIL(1)
  ok    233: rc35 TLS handshake vs plaintext listener        FAIL(1)
  ok    244: rc35 TLS handshake vs plaintext listener        FAIL(1)
```

> **ROUND-5 CORRECTION — the two "rc52" lines above exercised rc 56, not 52.**
> The `empty` stub called `close()` without first `recv()`ing the request.
> Discarding data still sitting in the receive queue obliges the kernel to send
> RST instead of FIN, so curl returned **56** (peer reset), not 52 (empty
> reply). Measured 3/3 directly against the stub as committed; the guards' own
> diagnostic line printed `curl rc=56` throughout.
>
> The assertions were still *correct* — the guards FAIL on both — but nothing in
> the battery could tell 52 from 56, because `assert 1` only records the
> guard's verdict and the verdict is identical for every non-6/7 rc. So the
> label drifted from the wire with every assertion green.
>
> This mattered beyond the label: **rc 52 was the one declared member still
> fail-open.** Re-running mutation N1 with the SKIP set broadened to include
> only rc 52 left the battery fully green (32 ok, 0 not ok, exit 0) — the exact
> fail-open shape this section claimed to have closed, on the very rc it named.
> Fixed in round 5: the `empty` stub now drains the request before closing
> (genuine 52), a separate `reset` shape covers 56, rc 60 is covered through the
> classifier, and each subject's rc is now MEASURED and pinned before its
> verdict is asserted.

### Review mutation N1 re-run — previously NOT caught, now caught

N1 broadens the SKIP set by a single rc. Applied to both guards
(`|| [ "$CURL_RC" -eq 28 ]`):

```
  NOT OK 233: rc28 timeout (listener accepted, then wedged)  expected FAIL(1), got exit 2
  NOT OK 244: rc28 timeout (listener accepted, then wedged)  expected FAIL(1), got exit 2
=== falsification battery: 27 ok, 2 not ok ===
BATTERY FAILED — a guard did not behave as its exit contract claims
exit=1
```

> **ROUND-5 CORRECTION — `27 ok / 2 not ok` totals 29, but the battery committed
> at `f10c9a1e` carries 32 assertions.** This capture was taken on an
> intermediate tree, before the R4 crash-loop and vacuous-window probes landed,
> so it certifies a tree that was never committed. Re-run against the committed
> battery, the same mutation yields **30 ok / 2 not ok**. The qualitative claim
> — N1 is now caught, and was not before — holds unchanged; only the pasted
> figures were stale.
>
> This is a RECURRENCE of the "22 ok → 23" defect that this very session filed
> and corrected as LOW a below: a count pasted from a run that predates the
> final artifact. Two instances in one batch make it a pattern rather than a
> slip — the counts were being copied forward from earlier runs instead of
> re-measured against the tree actually being committed.

Guards restored; `git diff` empty for both. Before this change the same mutation
produced a fully green battery — a wedged or TLS-broken gateway would have been
reported as "not deployed here".

---

## MEDIUM 2 — G30 was in permanent SKIP on the production host

G30's first real sweep SKIPped: the journal window holds ~97-125 lines but no
startup marker, because the gateway has been up since 2026-08-08 and journald
rotated the startup line. Check 1 — `GIN_MODE=release` read from the live
process's own `/proc/<pid>/environ` — DID run and DID pass, and the guard threw
that result away and reported "certified nothing". A release-wired gate that can
never go red is muted just as effectively as one nobody runs.

**Disposition: FIXED.** The unprovable window now yields PASS-with-caveat: it
reports the check that genuinely ran and passed, and names the cross-check that
could not run and why. It is explicitly a WEAKER pass than the full GREEN and
says so rather than presenting itself as equivalent.

### It cannot fail open — proven from both directions

Ordering guarantees it: check 1 FAILs outright before this branch is reached, and
the GIN-debug scan was MOVED ABOVE the branch so it FAILs on any hit regardless
of window provability. Two new probes assert exactly that on this window shape:

```
  ok    229: unprovable window -> PASS-with-caveat           PASS(0)
  ok    229: unprovable window + GIN-debug still FAILs       FAIL(1)
  ok    229: unprovable window + no GIN_MODE still FAILs     FAIL(1)
```

### §11.4.120 — the reconciled assertion is load-bearing, not fake-passed

The battery previously asserted `SKIP(2)` here. That assertion was RECONCILED to
track the new mechanism, not edited until it agreed. Proof: neutering the
GIN-debug check (`-eq 0` → `-ge 0`) makes the battery FAIL, including on the new
probe:

```
  NOT OK 229: process emits GIN-debug route dumps            expected FAIL(1), got exit 0
  NOT OK 229: unprovable window + GIN-debug still FAILs      expected FAIL(1), got exit 0
=== falsification battery: 29 ok, 2 not ok ===
```

Guard restored; mutation string absent afterwards.

### Live verdict after the fix

```
$ bash scripts/testing/guard_hxc229_gateway_release_mode.sh
PASS-with-caveat (HXC-229): pid=107268 carries GIN_MODE=release, read from the
live process's own /proc/107268/environ — the primary invariant is ENFORCED and
holds. CAVEAT: the corroborating route-dump cross-check could NOT run — the
journal window since '2026-08-08 22:08:59 +05' holds 125 line(s) but no startup
marker (rotated or vacuumed; expected on a long-lived process). ...
exit=0
```

---

## MEDIUM 3 — the "NO DATA LOST" attestation in `04f88a23` mis-enumerates

`04f88a23`'s commit message states:

> The DB delta versus HEAD is exactly the two doc_segments rows plus HXC-230's
> body line: items, item_history and obsolete_details are row-identical and no
> existing seq was mutated.

**That enumeration is wrong. The conclusion it supports is right.** Measured
`04f88a23^` → `04f88a23`, the delta is **5 changes across 4 tables**:

| # | Table | Change | Claimed row-identical? |
|---|---|---|---|
| 1 | `doc_segments` | +2 rows (449 → 451) — the repair itself | no (correctly claimed) |
| 2 | `items` | `HXC-230.body_md` +1 line (the `**Obsolete-Details:**` line) | **claimed row-identical — WRONG** |
| 3 | `items` | `HXC-230.last_modified` `2026-08-06 21:10:37` → `2026-08-09 13:03:47` | **claimed row-identical — WRONG** |
| 4 | `item_history` | +1 row (400 → 401), id 403 | **claimed row-identical — WRONG** |
| 5 | `sqlite_sequence` | `doc_segments` 1216→1218, `item_history` 402→403 | not mentioned at all |
| — | `obsolete_details` | 3 → 3, dump byte-identical | row-identical (correctly claimed) |

`obsolete_details` is the ONLY one of the three named tables that is genuinely
row-identical.

The new `item_history` row is:

```
403|HXC-230|Obsolete|AI|2026-08-09|duplicate-of|
docs/qa/live_systemd_boot_20260806T145857Z/11_helixagent_coldboot_race.txt|2026-08-09 13:03:47|
```

**Disposition: CORRECTED HERE (commit message unrewritable, §11.4.113).** No data
was lost. Every one of the five changes is an ADDITION or a legitimate touch:
row 403 is a DESIRABLE §11.4.34 audit event recording the Obsolete transition —
precisely the kind of record whose ABSENCE is filed as HXC-259 below — and the
`sqlite_sequence` bumps are the mechanical consequence of the inserts. The
attestation's conclusion stands; its enumeration did not, and the enumeration is
the part that was doing the certifying.

---

## MEDIUM 4 — `meta.last_sync_direction` misreports its own provenance

Stored value in the committed DB at HEAD:

```
last_sync_direction | md-to-db | 2026-07-11 10:20:44
```

The sync performed in `04f88a23` was `db-to-md`, on 2026-08-09 — in a commit
whose entire thesis is that sync DIRECTION matters.

**Root cause (measured, not inferred).** The field has EXACTLY ONE writer:

```
constitution/scripts/workable-items/cmd/workable-items/db.go:560
    UPDATE meta SET value=?, ... WHERE key='last_sync_direction'   -- literal "md-to-db"
```

It lives in `replaceDocument()`, whose only non-test callers are `sync.go:45` and
`sync.go:59` — both on the md-to-db path. The value written is a hardcoded string
literal, not the direction that ran. The `db-to-md` render path never touches
`meta` at all.

Therefore the field **cannot ever read `db-to-md`**: it is structurally incapable
of describing a db-to-md sync. The stored value is not a wrong value — it is a
truthful record of the last *md-to-db* sync (2026-07-11), sitting under a name
that promises "direction of last sync". A reader consulting it concludes the
reverse of what happened.

**Disposition: ROOT-CAUSED, DELIBERATELY NOT HAND-PATCHED. Filed as HXC-258.**
The brief invited correcting the stored value. I did not, and the evidence is the
reason: writing `db-to-md` by hand would (a) store a value no code path can
produce, which the next md-to-db sync silently reverts — a fix with a built-in
expiry date; (b) require a direct write to the tracked SSoT, which is the exact
anti-pattern that caused CAUSE A and the third defect in `04f88a23` (direct
INSERTs bypassing the sanctioned tooling); and (c) leave the actual defect — a
one-directional field — in place while making it look repaired. The real fix is
in `constitution/`, a separate submodule whose modification carries the §11.4.26
fetch-first-and-push-all-upstreams workflow, out of scope for this remediation
and not pushable under the instruction not to push.

---

## LOW a — `be5d56be` says "22 ok"; measured is 23

The commit message states `-> 22 ok, 0 not ok`. The battery at that commit
carried 23 assertions and reported 23. Confirmed by subtraction against this
session's first run: 29 ok after adding exactly 6 SKIP-contract assertions.

**Disposition: CORRECTED HERE (commit message unrewritable).** The direction is
harmless — it understates — but it is exactly the uncited-count class
(§11.4.226) that `be5d56be` itself prosecutes, so it is recorded rather than
waved through. Current count after this remediation: **32 ok, 0 not ok.**

> **ROUND-5 CORRECTION — the subtraction above is itself built on a stale
> figure.** "29 ok after adding exactly 6 SKIP-contract assertions" describes an
> intermediate tree, not the committed one. The final count for `f10c9a1e` —
> **32 ok, 0 not ok** — is correct as stated in the last sentence, and is the
> figure a re-review reproduces. After the round-5 remediation the battery
> carries **42 ok, 0 not ok** (10 assertions added: 4 rc pins, 2 for the new
> rc56 `reset` subject, 2 for rc60, 2 for rc6).

---

## LOW b — the R4 fixture did not reproduce the production restart policy

The R4 case's comment claimed it "reproduces the production restart policy
exactly" while using `RestartSec=1`. Measured on the live unit:

```
$ systemctl --user show helixllm-gateway -p Restart -p RestartUSec -p StartLimitBurst -p StartLimitIntervalUSec
Restart=on-failure
RestartUSec=10s
StartLimitIntervalUSec=10s
StartLimitBurst=5
```

**This was more than a comment error — it concealed a live fail-open.** The guard
has two crash-loop signals: `SubState=auto-restart`, and `NRestarts` RISING
across its 2.5s sample. Measured at each timing:

```
RestartSec=1   -> restarts inside the sample window, NRestarts RISES
RestartSec=10  -> ActiveState=activating SubState=auto-restart, NRestarts=0 (static)
```

So the fast fixture was carried entirely by the rising-counter path, and the
auto-restart path — the ONLY one that fires under the production policy — was
never exercised. Proof, by deleting the auto-restart signal from the guard:

```
  NOT OK 229: crash-loop @ production RestartSec=10 (auto-restart path)  expected FAIL(1), got exit 2
  ok     229: crash-loop @ RestartSec=1 (rising-counter path)            FAIL(1)
```

The old battery would have stayed fully green while every real production
crash-loop went undetected as a SKIP.

**Disposition: FIXED — fixture changed to match production, and both kept.** The
production timing is asserted because it is what deploys; the fast timing is kept
because the rising-counter path is real code that also needs a subject. Neither
substitutes for the other. The comment now states the measured timings rather
than claiming exactness.

---

## LOW c — `--explain` had drifted to G1-G14 while 32 gates exist

`--explain` is `grep -E '^#   G[0-9]'` over the script's own header. That header
stalled at G14; `be5d56be` widened the gap by three without noting it.

**Disposition: FIXED for the 3 added, and the residual gap is now MECHANICALLY
SELF-REPORTING rather than silent.** `--explain` now diffs its description list
against the live `want_gate GN` set and names what it cannot describe:

```
G30 §11.4.135 HXC-229 guard      — gateway serves in Gin RELEASE mode (live process)
G31 §11.4.135 HXC-233 guard      — completion path returns a REAL generation (live e2e)
G32 §11.4.135 HXC-244 guard      — health endpoint names the components it checked

NOTE: 15 gate(s) run in this sweep but have NO description above: G15 G16 G17 G18
      G19 G20 G21 G22 G23 G24 G25 G26 G27 G28 G29
      They still EXECUTE and are still release-blocking — only this listing is incomplete.
```

G15-G29 remain undescribed. That is stated on every invocation rather than left
to be rediscovered. A drifted `--explain` is quietly wrong in the worst way: it
answers confidently and omits, so a reader concludes a gate does not exist.

The meta-test still mutates only G1-G6. That was pre-existing, is NOT closed here,
and is not claimed to be.

---

## LOW d — zero `item_history` is systemic, not a two-item anomaly

`04f88a23` reported HXC-247/248 as having zero `item_history` rows and left them
visible rather than back-filling. The review found the condition is systemic.
Confirmed and re-measured:

| Revision | items | zero-history rows |
|---|---|---|
| `04f88a23` / `be5d56be` (review baseline) | 442 | 146 |
| `69573772` / HEAD | 449 / 451 | 145 |

At HEAD: **81 distinct atm_ids across 145 item rows** have no `item_history`
entry. The review's "82 distinct (146 rows)" was correct at its baseline; one item
gained history since. Distribution by location:

```
Fixed  | 144
Issues |   1
```

So the gap is overwhelmingly in the ALREADY-CLOSED record — the part an audit
would most want a trail for.

**Disposition: RECORDED, NOT BACK-FILLED. Filed as HXC-259.** Nothing was
invented. Back-filling would require fabricating `on_date` and actor values for
events nobody witnessed, producing an audit trail that is worse than an absent
one because it would look authoritative (§11.4.6).

---

## Sweep results — before and after

Both runs: `bash scripts/verify-all-constitution-rules.sh`, 32 gates.

```
BEFORE (HEAD 69573772):  30 PASS,  2 SKIP (G14, G30),  0 FAIL
AFTER  (this change):    31 PASS,  1 SKIP (G14),       0 FAIL
```

The only change to the result set is **G30: SKIP → PASS**, which is the intended
effect of the MEDIUM-2 fix and is explained by it. G14 remains SKIP because the
docs_chain engine is not installed on this host — pre-existing and unrelated.

**The baseline stated in my brief was wrong and is corrected here.** It described
"7 pre-existing environmental failures (G1/G5/G22/G24/G25/G26/G29)". Measured on
this host at `69573772`, every one of those seven gates PASSES and the sweep
reports zero failures overall. I did not inherit a 7-failure baseline and did not
resolve one; the sweep was already green.

## Honest boundaries (§11.4.6)

- **G11/G12 FAIL→PASS flip: CONFIRMED here.** The review could not check it — the
  workable-items binary is absent in its clone, so it SKIP-OK'd. In this checkout
  the binary is present (`constitution/scripts/workable-items/bin/workable-items`)
  and both gates run for real: G11 and G12 PASS in both the before and after
  sweeps above.
- **MEDIUM 4 is not closed.** It is root-caused and filed (HXC-258). The fix lives
  in another submodule.
- **The meta-test's G1-G6 mutation scope is not closed** and was not touched.
- **G15-G29 remain undescribed** in `--explain`; only the drift's invisibility was
  fixed.
- This remediation proves the guards behave as their exit contracts claim on
  constructed subjects and on the live stack. It does not establish unattended
  enforcement: the sweep is still operator-invoked, as `HXC-253` already records.
