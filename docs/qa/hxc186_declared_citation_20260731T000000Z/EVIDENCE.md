# HXC-186 — G7 RULE 3 citation-vs-coincidence — QA evidence (§11.4.83)

**Run ID:** `hxc186_declared_citation_20260731T000000Z`
**Captured (UTC):** 2026-07-31
**Verdict:** PASS
**Evidence-For-Commit:** _(this run documents the HXC-186 fix commit; see
`git log --grep HXC-186`. The fix commit touches only `scripts/` + `docs/`, so
it is not itself a feature-shipping commit under G7's heuristic.)_

## The defect

`scripts/verify_qa_evidence.sh` RULE 3 cleared a feature commit whenever any
TRACKED file under `docs/qa/` contained that commit's 8-hex-char prefix
**anywhere** in its content (`git grep -F <needle>`).

Evidence files also record **provenance**. `qa_capture_grounding()` in
`scripts/qa/lib/sec_capture_lib.sh` writes:

```
## repo HEAD
<git log --oneline -1>
```

into `docs/qa/<run-id>/grounding.txt`. Measured in this repository on
2026-07-31, `git log --oneline` abbreviates to **exactly 8 hex chars** —
precisely RULE 3's needle width — so every such stamp was a live match
candidate.

That stamp is the SHA of whatever HEAD happened to be **when the capture ran**,
which under §11.4.103 (≥3 concurrent streams committing to main) is routinely a
*different* commit from the one the evidence documents. An unrelated feature
commit could therefore be cleared by evidence demonstrating something else — a
§11.4 PASS-bluff inside an **enforcing** release gate.

Confirmed instance of the mechanism (unrelated SHAs in one file):

```
docs/qa/netfix_ipv6_bracket_9d082a7b_20260728T233036Z/grounding.txt
  ## repo HEAD
  faadf59e docs(tracker): re-sync derived docs after HXC-201 was filed   <-- stamp
  ## fix commit under test
  9d082a7b fix(net): HXC-185 — bracket IPv6 authorities ...              <-- subject
```

**Not yet realised.** All 12 distinct HEAD stamps in the tracked corpus resolve
to non-feature commits (docs/governance/gates), so none of them currently needs
evidence. The report's "has not happened yet" claim is confirmed — the hole was
latent, not live.

**Broader than the stamp.** The corpus also carries SHAs in scratch-checkout
paths (`chdir /tmp/.../<sha>/helix_code`, 59 lines), request nonces and build
logs. A denylist of recognised stamp formats would therefore have reproduced the
same shape of hole it was meant to close.

## The fix

RULE 3 is now two-tier, keyed on `QA_CITATION_BASELINE` (default: repo HEAD when
HXC-186 landed, `249ae5dc`):

* **STRICT** (commits strictly after the baseline) — the SHA must sit on the
  **same line** as a declared citation label from a closed set:
  `Evidence-For-Commit:`, `Source commit:`, `Fix commit:`, `Commit under test:`.
  A provenance stamp carries no label, so it no longer clears anything. The
  required trailing colon is what separates a declaration from a heading
  (`## repo HEAD` puts its SHA on the *next* line).
* **LEGACY** (at or before the baseline) — historical substring behaviour,
  reported as `legacy substring, pre-baseline — not coincidence-proof` so it is
  never mistaken for a declaration.

Unresolvable baseline ⇒ **fails closed** (everything strict), because an unknown
baseline must never silently grant leniency (§11.4.6).

`qa_finish` now emits the canonical `**Evidence-For-Commit:**` field, always
`QA_COMMIT_SHA` (the commit under test), never HEAD. The grounding stamp is
deliberately **kept** — it is useful §11.4.108 provenance; it simply no longer
counts as a citation.

## Why the existing corpus was baseline-exempted, not migrated

At fix time, 42 of 57 in-range feature commits were cleared by RULE 3. **All 42
were inspected individually.** Every one resolves to a *deliberate* citation of
the commit under test — 37 via a labelled field, 5 via a markdown/HTML table cell
or prose (`corpus_audit_5_unlabelled_citations.txt`). **None was a coincidence.**

Applying the strict form retroactively would have turned those 5 honest,
human-written citations into release-blocking violations without improving
truthfulness. They are quarantined by the baseline — the same instrument G7
already uses for historical scope — and the strict rule binds everything forward.

This is a **quarantine, not a claim of cleanliness**: the legacy tier remains
coincidence-capable by construction, which is exactly why nothing new may enter
it.

## RED → GREEN (§11.4.115)

Guard: `scripts/tests/qa_evidence_citation_coincidence_meta_test.sh`, one source
with a `RED_MODE` polarity switch. Cases 2 and 3 are positive controls asserting
a genuine declared citation and RULE 1 still clear — a fix that merely rejected
everything would fail them.

| Run | Artifact | Expected | Result | Log |
|---|---|---|---|---|
| `RED_MODE=1` | pre-fix | defect present | **PASS** (rc 0) | `red_mode1_before_fix.log` |
| `RED_MODE=0` | pre-fix | defect absent | **FAIL** (rc 1) — polarity is real | `red_mode0_before_fix.log` |
| `RED_MODE=0` | post-fix | defect absent | **PASS** (rc 0) | `red_mode0_after_fix.log` |
| `RED_MODE=1` | post-fix | defect present | **FAIL** (rc 1) — defect gone | — |

## No regression on the real corpus

| Check | Before | After |
|---|---|---|
| G7 (`qa_evidence_gate.sh`) | PASS — 57 evaluated, 0 violations | PASS — 57 evaluated, 0 violations |
| Match kinds | 15 RULE 1 + 42 RULE 3 | 15 RULE 1 + 42 legacy (relabelled) |
| `verify_qa_evidence_meta_test.sh` | 10 PASS | 10 PASS, 0 FAIL |

Logs: `g7_before_fix.log`, `g7_after_fix_and_existing_metatest.log`.

## Standing guard (§11.4.135)

Registered as **G27** (`CM-QA-EVIDENCE-DECLARED-CITATION`) in
`scripts/verify-all-constitution-rules.sh`, running the guard at `RED_MODE=0`.
Run log: `g27_run.log`.

## Honest boundaries (§11.4.6)

1. **Deliberate false declaration still clears.** Writing
   `Evidence-For-Commit: X` into evidence that is not about X clears X. That is
   fraud, not coincidence, and no substring gate can detect it.
2. **Blanket-ledger risk persists** (pre-existing, documented in the script
   header): one document declaring many SHAs would clear each. The gate prints
   the citing file path on every pass so a reviewer can tell a per-feature
   transcript from a ledger.
3. **Legacy tier stays coincidence-capable** by construction — see the
   quarantine note above.
4. **Evidence depth is still not assessed** — the gate asserts a binding exists,
   not that the transcript meets §11.4.83's bidirectional bar. Pre-existing.
5. **One weak historical clearance.** `a7d8dbb5` is cleared by a line reading
   `| Pre-fix artifact | \`a7d8dbb5\` (parent of the fix) |` — a *deliberate*
   mention, but semantically it names the commit as the RED baseline rather than
   as the thing proven. Legitimate under the legacy tier; flagged rather than
   papered over.
6. **Runtime cost rose.** The strict tier adds a per-candidate-file `git grep`,
   which pushed a full G7 sweep past 120 s on this repo. Correctness-neutral, but
   worth optimising if it becomes a release-gate bottleneck.
