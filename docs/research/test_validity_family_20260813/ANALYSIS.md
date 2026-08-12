# §11.4.150 deep multi-angle research pass — test-validity family

**Covers:** HXC-243, HXC-287, HXC-291 (and produced HXC-322).
**Conducted:** 2026-08-13, main stream (T1/main - claude1), in parallel with four
background review streams per §11.4.150(F).
**Sources verified:** 2026-08-13 (URLs and access date below, per §11.4.99).

---

## Why this family needed a pass

These three items share one question: *when is a test entitled to report PASS?*

- **HXC-243** — 125 checks across the test banks asserted nothing at all.
- **HXC-287** — a penetration test that could not fail.
- **HXC-291** — a guard that certified neither polarity.

§11.4.150 requires a documented multi-angle pass before any may close, and
requires it to confirm we are not sitting on a bigger instance elsewhere.

---

## Angle 1 — is this a recognised class, and what does the literature prescribe?

Yes, and the prescribed remedy is the one this project already mandates.

The research literature on test smells treats **weak assertions** as a distinct
defect: assertions that are *"trivial to satisfy and would not trigger any error
if the target class had incorrect behavior, such as tautologies."* The
neighbouring smell **Assertion Roulette** (assertions without explanatory
messages) is measured at ~42% of test cases in the studied corpora — so
assertion quality problems are the common case, not the exception.

The prescribed detection method is worth quoting because it is exactly §1.1:

> Mutation analysis is used to discard weak assertions by identifying assertions
> that **hold for every mutant's execution** and are therefore weak, whereas
> assertions that **do not hold for at least one mutant** are useful, because
> they distinguish buggy versions of the code.

That is the paired-mutation discipline, arrived at independently by the research
community. Our §1.1 requirement is not a local invention — it is the
field-standard method for exactly this defect, which materially strengthens the
closure case for all three items: they were each closed by making a guard
*mutation-detectable*, which is the literature's own definition of a useful
assertion.

Empirical support for prioritising it: mutation testing at method level shows
low-quality test methods are **over-concentrated** in the critical test smells —
so the assertion-free tests are not evenly distributed noise, they cluster where
quality is already weakest.

## Angle 2 — measure our own corpus rather than reason about it

Swept every Go test function in the four owned repositories for functions
containing no assertion of any kind (no `require.`/`assert.`/`t.Error`/
`t.Fatal`/`want`/`expect`).

**Instrument caveat, stated because it changes the numbers.** A first pass using
line-oriented `awk` gave 60 flagged in `helix_qa`; a structure-aware pass over
whole function bodies gave **41**. The two disagree because the crude version
mis-splits function boundaries. The 41 supersedes it. Both undercount in one
direction and overcount in another: a test whose assertions live in a *helper*
taking `t` (e.g. `runCycle(t, …)`) is flagged as assertion-free when it is not.
So these figures are a **lead, not a finding** — consistent with the standing
rule that a count is never converted into a claim without reading the lines.

Reading the lines is what produced the result below.

## Angle 3 — confirm-no-bigger-problem (§11.4.150(C))

The flagged tests in `helix_qa` turned out to be **already swept and
deliberately marked** — 27 sites carry an explicit, greppable justification:

```
// bluff-scan: no-assert-ok (concurrency test — go test -race catches data
// races; absence of panic == correctness)
```

That is good practice: the exemption is explicit, attributed, and searchable.
But the justifications split into two materially different kinds.

**Kind A — 4 sites naming a real detector.** These cite `go test -race`. The
reasoning is sound: the race detector *is* a positive detector, so a
concurrency test with no explicit assertion still has something that can fail.

The reasoning has a load-bearing precondition, and it does not hold:

| target | command | detector |
|---|---|---|
| `helix_qa` `make test` (the ordinary one) | `go test ./... -count=1` | **not engaged** |
| `helix_qa` `make test-race` (separate, opt-in) | `go test ./... -race -count=1` | engaged |
| `helix_agent` default test targets | `go test … -race -coverprofile=…` | engaged |
| `llms_verifier` default test target | `go test -v -race -coverprofile=…` | engaged |

Measured directly on the four affected tests:

- without `-race`: `ok` in **0.008s** — the detector is not running
- with `-race`: `ok` in **1.024s** — the detector is running

Both pass, because no race exists today. The point is that under the command
the project presents as its ordinary one, **the guard is disarmed** and would
not catch a race introduced later. `helix_qa` is the only one of the three
repositories where the flag is optional — most likely an oversight.

**Filed as HXC-322 (Medium).**

**Kind B — ~23 sites justified as "must not panic".** Unlike Kind A there is no
detector to enable; the justification is *absence of error*, which §11.4 /
§11.4.1 name explicitly as an insufficient basis for PASS. These are recorded in
HXC-322 as an observation rather than filed as a defect: they may retain modest
value as coarse smoke checks, but they should not be counted as coverage. That
is a judgement call for the operator, not a repair.

---

## What this changes about the family

1. **All three closures are strengthened, not weakened.** Each was closed by
   making a guard mutation-detectable, which is precisely the literature's
   definition of a useful (non-weak) assertion. The method is field-standard.
2. **The class is not exhausted by the three items.** HXC-322 is a fourth
   instance in a different shape: not a missing assertion, but an assertion
   whose *detector* is not switched on by the path people use.
3. **Explicit exemption markers are the right pattern and should be kept.** The
   `bluff-scan: no-assert-ok` convention is what made this analysis possible at
   all — the exemptions were greppable and carried their reasoning. The defect
   was in one reason's precondition, not in the practice.

## Honest boundary (§11.4.6)

This pass establishes the class is real and field-recognised, and found one new
instance. It does **not** establish that the remaining ~1,100 flagged functions
across the other three repositories are sound — those were counted, not read,
and the count is known to be inaccurate in both directions. That is an
enumerated, un-exercised gap (§11.4.118), stated here rather than implied clean.
It also does not substitute for §11.4.108 runtime-signature verification or the
§11.4.40 full-suite retest.

## Sources verified 2026-08-13

- https://testsmells.org/pages/testsmells.html
- https://arxiv.org/pdf/2207.05539 — *Refactoring Assertion Roulette and Duplicate Assert test smells: a controlled experiment*
- https://arxiv.org/pdf/2301.12284 — *Assertion Inferring Mutants*
- https://arxiv.org/pdf/2203.12085 — *Characterizing High-Quality Test Methods: A First Empirical Study*
- https://www.researchgate.net/publication/346592488_Revisiting_Test_Smells_in_Automatically_Generated_Tests
