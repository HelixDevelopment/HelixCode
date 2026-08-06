# HXC-223 — §11.4.84 residue scan blocks captured §1.1 mutation evidence

**Run ID:** `hxc223_mutation_residue_evidence_exemption_20260806T181500Z`
**Date (UTC):** 2026-08-06
**Item:** HXC-223 — Bug
**Anchors:** §11.4.84 (mutation-residue quiescence) · §1.1 (paired mutation) ·
§11.4.115 (RED-baseline + polarity switch) · §11.4.83 (docs/qa evidence tree) ·
§11.4.5 / §11.4.6 (captured evidence, no guessing)

---

## 1. Defect (root cause, FACT)

`scripts/git_hooks/pre-commit` step 3 blocks any staged file whose blob matches
`MUTATED for paired|// always pass|# always pass|MUTATION-RESIDUE|_mutated_`.
Its single exemption (added 2026-07-11) requires **BOTH**:

- **(a)** the literal opt-in marker `11.4.84-mutation-test-exempt`; and
- **(b)** a PROVEN restore idiom — a `trap <target> EXIT` whose target's own
  body contains a `cp ... backup ...` or `git checkout --` restore call.

A **captured-evidence artifact can satisfy (a) but can never satisfy (b)**: it is
captured *output* — no trap, no restore, nothing to execute. The exemption was
designed for mutation-TEST *scripts*; evidence artifacts are structurally unable
to qualify.

Consequence, live and biting: §1.1 **requires** the paired-mutation proof be
captured, while §11.4.84 **refused to let that capture be committed** — the proof
of a correctly serialised mutation was unlandable precisely *because* it proved
the mutation. Blocked artifact:
`docs/qa/hxc220_module_identity_gate_20260805T120714Z/3_mutation_proof.log`
(mode 644, one marker at line 13).

## 2. Discriminator — verified, not assumed (§11.4.6)

The candidate discriminator "under `docs/qa/**` ⇒ inert" was **checked and found
unsound**: `git ls-files -s docs/qa` shows tracked **executables** in that tree
(mode `100755` harness `run_proof.sh` under `phase3_*`, `cpucaps_*`,
`vectorization_*`; `hxc204_*/loadgen.sh`; `hxc215_*/trapdemo_*.sh`). Location
alone would have been a real hole.

What §11.4.84 actually protects against is **residue that RUNS** (its forensic
case: an always-pass JWT-verify bypass swept into a logo commit and becoming
live). So the exemption keys on *inertness*, requiring **all four**:

| # | Condition | Defeats |
|---|-----------|---------|
| (i) | path under `docs/qa/**` | residue anywhere else |
| (ii) | extension ∈ {`log`,`txt`,`out`,`md`} — **not** `.sh`/`.go`/…, **not** `.patch`/`.diff` (appliable) | code by extension |
| (iii) | **staged** mode `100644` | a staged executable is code whatever it is named |
| (iv) | first staged line is not `#!` | script-shaped blob under an inert name |

(ii)+(iii)+(iv) are deliberately redundant: each is individually defeatable, so
all are required together. The path does **not** demand the (a) opt-in marker —
captured evidence must stay verbatim (§11.4.5); editing an opt-in header into a
captured log would corrupt the artifact whose fidelity is the point.

## 3. §11.4.115 RED → GREEN (one source, two roles)

Test: `scripts/git_hooks/test_mutation_residue_evidence_exempt.sh`
(`RED_MODE=1` default = assert defect PRESENT; `RED_MODE=0` = standing GREEN
guard). Fixture for Q1 is the **byte-for-byte real blocked artifact**, not a
synthetic stand-in. Runs real `git add`/`git commit` against the real hook in a
throwaway repo. **`--no-verify` is never used, anywhere.**

| Log | Run | Exit | Meaning |
|-----|-----|------|---------|
| `1_red_baseline_prefix_hook.log` | `RED_MODE=1`, **pre-fix** hook | `0` | 9/9 PASS — defect reproduced on the real artifact |
| `2_green_fails_prefix_hook.log` | `RED_MODE=0`, **pre-fix** hook | `1` | Q1 + Q1b FAIL — the guard is not blind |
| `3_green_postfix_hook.log` | `RED_MODE=0`, **post-fix** hook | `0` | 10/10 PASS — defect absent |
| `4_red_now_fails_postfix.log` | `RED_MODE=1`, **post-fix** hook | `1` | Q1 correctly inverts — polarity is real |

## 4. Four-quadrant + narrowness results (post-fix, `RED_MODE=0`, exit 0)

| Case | Expected | Observed |
|------|----------|----------|
| Q1 evidence `.log` w/ marker (real artifact) | ALLOW | ALLOW ✅ **(the fix)** |
| Q1b audit NOTICE emitted for the grant | present | present ✅ |
| Q2a live source `scripts/live_gate.sh` | BLOCK | BLOCK ✅ |
| Q2b live source `helix_code/internal/auth/jwt.go` | BLOCK | BLOCK ✅ |
| Q3 mutation-test script with (a)+(b) | ALLOW | ALLOW ✅ *(pre-existing path intact)* |
| Q4 claims (a), no (b), outside evidence dir | BLOCK | BLOCK ✅ |
| N1 **executable** `.sh` under `docs/qa` | BLOCK | BLOCK ✅ |
| N2 `.log` w/ marker **outside** `docs/qa` | BLOCK | BLOCK ✅ |
| N3 `.sh` under `docs/qa` staged mode 644 | BLOCK | BLOCK ✅ |
| N4 `.log` under `docs/qa` staged **755** | BLOCK | BLOCK ✅ |

Pre-existing suite `scripts/git_hooks/test_hooks.sh`: **43 PASS / 0 FAIL**
(`5_existing_suite_no_regression.log`) — no regression.

## 5. §1.1 paired mutations (run on COPIES; live tree never mutated)

| Mutation | Expectation | Observed |
|----------|-------------|----------|
| **M1** `is_captured_evidence_exempt() { return 0 … }` (blanket bypass) | guards must FAIL | Q2a, Q2b, Q4, N1, N2, N3, N4 **all FAIL** ✅ |
| **M2** delete only condition (iii), the staged-mode check | only N4 FAILs | **exactly N4 FAILs** ✅ |

M1 proves the exemption is not a blanket bypass; M2 proves condition (iii) is
individually load-bearing. Both ran in `mktemp -d` copies — §11.4.84 quiescence
was preserved in the working tree from which this fix is committed
(`7_paired_mutation_M2_targeted_and_quiescence.log` records `0` residue markers
added by the diff and the live hook's sha256).

## 6. Canonical source

`scripts/git_hooks/pre-commit` is canonical for this repo:
`.git/hooks/pre-commit` is a **symlink** → `../../scripts/git_hooks/pre-commit`,
and `scripts/install_git_hooks.sh` is symlink-based ("editing the source updates
the installed hook with no re-install"); re-running it reports
`4 already current`. The constitution submodule ships **no** `pre-commit` under
`constitution/scripts/hooks/` (it carries `guard-*`, `post-merge`, and their
tests), so there is **no constitution-side change** for this fix and no
submodule sha to report.

## 7. Residual gap, stated honestly (§11.4.6)

The four conditions establish the artifact is not executable **by its own staged
form**. They do **not** prove that no external caller could feed it to an
interpreter (`bash docs/qa/<run>/x.log`), nor that a human will not later
`chmod +x` it. Both are deliberate acts against a file whose directory,
extension and mode all declare it evidence — and a later `chmod` is itself a
staged mode change this scan re-evaluates on the next commit. The gap is narrow;
it is documented in the hook body, not claimed closed. This mirrors the prior
author's accepted limitation that the (a)+(b) path is textual, not a semantic
proof that the restore idiom executes correctly.
