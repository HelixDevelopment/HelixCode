# HXC-215 — the per-commit compile-integrity gate accused innocent commits

**Run-id:** `hxc215_commit_compile_integrity_classifier_20260731T072605Z` (§11.4.83)
**Changed:** `scripts/gates/commit_compile_integrity_gate.sh`, `scripts/verify-all-constitution-rules.sh` (G19)

---

## 1. The defect (FACT)

G19's gate reported `DOES-NOT-COMPILE` naming a **different innocent commit almost
every run**, on unchanged history with identical build tags.

Two operator-captured runs:

| Run | Range | Accused | Packages blamed |
|---|---|---|---|
| `/tmp/g19-cci.out` | `HEAD~3..HEAD` | `8490a5d9` | `internal/project`, `internal/telemetry/i18n`, `internal/tools/lsp_fakeserver` |
| `/tmp/g19b.out` | `origin/main..HEAD` | `92840ade` | `internal/rules`, `internal/telemetry/i18n` |

Then an 8-run reproduction against the old logic on a **pinned** range
(`red_reproduction_old_gate/`) produced **4 PASS / 4 FAIL on identical input**, naming
**two further** innocent commits — `11861996` and `d99ce58c`. Four distinct innocent
commits in total.

Three independent proofs the accusations were false, none needing a hypothesis:

1. **Self-contradiction.** `g19b.out` blames `92840ade` while its own output lists
   `8490a5d9  COMPILES` — the commit the previous run had just blamed.
2. **Causal impossibility.** `92840ade` touches **only** `helix_code/tests/e2e/mocks/**`,
   yet was blamed for `internal/rules` and `internal/telemetry/i18n`. No package under
   `internal/` imports the e2e mocks (verified — the sole grep hit is a comment), so that
   commit could not break them by any mechanism.
3. **No diagnostics.** Every report carried `FAIL <pkg> [build failed]` with **zero**
   `file.go:line:col:` compiler diagnostics.

## 2. Root cause (FACT)

The gate **asserted the wrong condition** — it read `go test` exiting non-zero as
"this commit does not compile":

```sh
if [ $rc -eq 0 ]; then COMPILES; else DOES-NOT-COMPILE; fi
```

`go test ./...` also exits non-zero when the *toolchain* dies, printing `[build failed]`
for whichever packages were in flight — hence a different accused package set each run.
This is §11.4.201: a gate must assert the real condition; a false-positive refusal is a
FAIL-bluff that teaches everyone to dismiss the gate.

**Which transient — now FACT.** The reproduction preserved the build logs the original
runs destroyed. They name two causes, neither a source defect:

1. `fork/exec .../compile: resource temporarily unavailable`
   — **EAGAIN on `fork(2)`**: the kernel refused to create the process, so the Go driver
   could not spawn compile/link/vet workers. **No commit was ever read.** ~7 agents were
   live on this 64-core host (§12.12).

   *Which ceiling* (measured, with its confidence): the binding limit is the **cgroup
   pids controller on the per-project session slice — `pids.max = 4096`** — not the
   per-user `RLIMIT_NPROC` (262144 here) nor system-wide `threads-max` (2056078).
   `go test ./...` on 64 CPUs fans out to many concurrent tool processes, so a 4096-PID
   slice shared by several agents is reachable. **Honest boundary (§11.4.6):** the EAGAIN
   is FACT from the log and the 4096 ceiling is FACT from `/sys/fs/cgroup`, but
   `pids.current` was not sampled at the instant of failure — that this ceiling is the one
   that bound is a well-supported *inference* from the alternatives being three to four
   orders of magnitude larger.
2. `chdir .../cci-gate.7GVMhX/d99ce58c/helix_code: no such file or directory`
   — the gate's **own worktree vanished mid-build**: the trap defect below, caught in the
   wild.

So the two defects are **causally linked**: the trap bug *manufactured* the evidence the
classification bug then misread as a broken commit.

## 3. Second defect: `trap cleanup EXIT INT TERM` (FACT)

Bash runs a signal handler and then **resumes**. The reproducer
(`trap_resume_defect_reproducer.txt`, both scripts included) shows the old form running
cleanup mid-flight, resuming, running cleanup *again*, and **exiting 0** — a signal-killed
gate reporting SUCCESS. That is a PASS-bluff, strictly worse than the false accusation
because it is silent. Cause (2) above is this bug deleting `WORKDIR` out from under a live
`go test`.

Fixed to `trap cleanup EXIT` + `trap 'exit 130' INT` + `trap 'exit 143' TERM`, and observed
working on the real gate (`trap_fix_on_real_gate_sigterm.txt`).

## 4. The fix

A **failure classifier** with a third exit code, deliberately fail-safe:

| Verdict | Condition | `check_commit` | script exit | names a commit? |
|---|---|---|---|---|
| `DOES-NOT-COMPILE` | a `file.go:line:col: ` diagnostic exists | 1 | 1 | **yes** |
| `INFRA-FAIL` | known host/toolchain signature, no diagnostic | 3 | 3 | no |
| `BUILD-FAILED-UNCLASSIFIED` | neither | 4 | 3 | no |

Everything non-zero still **blocks**. Only a real compiler diagnostic names a commit.
Per-commit logs are now preserved outside the workdir (`$CCI_FAIL_LOG_DIR`), so the
evidence that separates these three survives the run — that preservation is what made
the root cause above knowable.

G19 branches on the exit **code**, never on truthiness, so an inconclusive build can no
longer be reported as a broken commit.

**Honest gap (§11.4.6 / §11.4.3):** the COMPILE-FAIL test is the presence of a *positional*
diagnostic, so a genuine source defect emitting none (import cycle, unsatisfied module
requirement) classifies UNCLASSIFIED. It still blocks — the gate under-*names* rather than
mis-names, the only safe direction. Widening it needs a captured log of that shape, not a
guess.

## 5. Third defect, found *by* the fix: the gate's own §1.1 pins had bit-rotted

The repaired `--self-test` reported **both** pinned fixtures as
`BUILD-FAILED-UNCLASSIFIED` in **0s**. The preserved logs gave the reason at once:
`go: updates to go.mod needed`.

`check_commit` symlinks the **live** submodule trees over each checkout. The pins are 130+
commits / 3 days old and their inner `go.mod` (`9c9b5912`) no longer resolves against
today's submodules (HEAD's is `4960895d`), so `go test` dies before compiling a line.

**The old gate could not see this.** It returned 1 for *any* non-zero exit, so the
known-bad half would have printed *"PASS: gate correctly FAILED the known-bad commit"*
while the compiler never ran — a falsifiability proof that proved nothing.

Fixed two ways: `--self-test` now exits **3 (INCONCLUSIVE)** with a §11.4.3 reason instead
of declaring the gate a bluff (that would be this ticket's own defect aimed at the gate);
and `synthetic_wolf_test.sh` supplies a falsifiability proof that works in the **current**
environment by building a broken commit from HEAD.

## 6. Determinism methodology (and its caveats)

The determinism series runs a **pinned SHA range** (`4590c638~1..3811f05b`, 4 commits, 2
compile-relevant) rather than G19's `--last 3`. Reason: another agent landed three commits
on `main` *during* an earlier attempt, so `--last 3` silently changed which commits it
examined between runs — a moving input cannot measure verdict stability. `--last 3` remains
what G19 invokes; only the measurement is pinned.

Controls, all visible in `determinism_summary.txt`:

* the gate's **sha256 is recorded on every run**, so a mid-series edit to the artifact
  cannot hide (two earlier attempts were discarded for exactly that reason);
* `HEAD at start` / `HEAD at end` are recorded, so concurrent-agent drift is visible;
* runs are **sequential** — parallel runs would contend for worktrees and would measure
  contention rather than the gate (§11.4.119).

## 7. Evidence index

| File | Proves |
|---|---|
| `red_reproduction_old_gate/` | §11.4.115 RED — 8 runs of the OLD logic on a pinned range: 4 PASS / 4 FAIL, two more innocent commits, plus the two real transients |
| `classifier_vs_real_flake_logs.txt` | the FINAL classifier exonerates **both** real flake logs — zero false accusations |
| `determinism_summary.txt`, `determinism_run_[1-5].out` | §11.4.50 — 5 sequential runs, pinned input, identical verdicts |
| `classifier_self_validation_and_mutations.txt` | §11.4.107(10) self-validation (9 fixtures) + §1.1 mutations failing in **both** directions |
| `synthetic_wolf_test.sh`, `synthetic_wolf_test.out` | §1.1 — a genuinely broken commit is caught, named alone, backed by a real diagnostic, exit 1 |
| `self_test_pins.txt`, `preserved_gate_logs/` | the built-in pins are bit-rotted and now say so honestly (exit 3) |
| `false_accusation_blame_mismatch.txt` | the accusations were **causally impossible**, independent of any compile |
| `g19_registration_exitcode_matrix.txt` | G19 maps exits 0/1/3/2/143 to four distinct verdicts |
| `g19_registration_paired_mutation.txt` | §1.1 — the pre-fix registration reports exit 3 as a broken commit |
| `trap_resume_defect_reproducer.txt`, `trapdemo_{old,new}.sh` | the trap defect (old form exits **0** when signal-killed) and its fix |
| `trap_fix_on_real_gate_sigterm.txt` | the trap fix observed on the real gate: stops and cleans up, does not resume |
| `preserve_log_failure_path_test.txt` | the evidence-preservation path (only reachable on failure) works |
