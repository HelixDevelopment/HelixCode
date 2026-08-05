# HXC-220 — module-identity exact-match regression gate — evidence

Run: 2026-08-05T12:07:14Z (UTC). Scope: author the missing §11.4.135 regression
guard HXC-186's closure-review flagged as absent for HXC-199. HXC-199 and
HXC-220 are left OPEN by this run — this is authoring evidence for an
independent reviewer, not a self-certification.

## What HXC-199's fix does and why it needed a guard

`scripts/probes/hxc159_env_facts.sh` originally compared the root and inner
Go module paths with a substring test (`[[ "$root_mod" == *"dev.helix.code"*
]]`). Because `dev.helix.code` is a literal PREFIX of `dev.helix.code/meta`
(the HXC-187 rename), that test still matched after the fix landed and kept
reporting the R-26 collision as unresolved on an already-fixed tree — a false
positive that could mislead a reader into reverting correct work. The fix
replaced it with `module_paths_identical()` (exact string equality,
`scripts/lib/module_identity.sh`), but landed with no permanent regression
guard. This run authors that guard.

## Files added / changed

- `scripts/gates/hxc199_module_identity_exact_match_gate.sh` (new) — the gate.
- `scripts/verify-all-constitution-rules.sh` (edited) — registers the gate as
  **G28** (`CM-MODULE-IDENTITY-EXACT-MATCH`).

No other file was modified. `scripts/lib/module_identity.sh` was mutated and
restored in-place for the paired-mutation proof below; its final state is
byte-identical to the pre-run state (`git diff` empty — see `5_restore_proof.log`).

## (a) Gate path + G-number + how the free slot was confirmed

- Gate: `scripts/gates/hxc199_module_identity_exact_match_gate.sh`
- Registered as **G28** in `scripts/verify-all-constitution-rules.sh`.
- Freedom confirmed empirically, not assumed: `7_g28_slot_free_proof.log`
  shows `git show HEAD:scripts/verify-all-constitution-rules.sh` (the
  revision this session started from) enumerates G1..G27 with zero G28, and
  the post-edit file enumerates G1..G28 with exactly one G28.

## (b) RED/GREEN exit codes personally observed

| Run | Command | File | Exit observed |
|---|---|---|---|
| GREEN (default) | `bash scripts/gates/hxc199_module_identity_exact_match_gate.sh` | `1_green_baseline.log` | **0** |
| RED | `RED_MODE=1 bash scripts/gates/hxc199_module_identity_exact_match_gate.sh` | `2_red_reproduction.log` | **0** |

RED_MODE=1 exits 0 because RED PASS = "the false positive was successfully
reproduced" (per §11.4.115 polarity convention, mirroring
`toolschema_i18n_seam_wired_gate.sh`): the reconstructed pre-HXC-199
substring predicate, applied to the CURRENT tree's real root `go.mod` line
(`module dev.helix.code/meta`), wrongly reports a collision, while the
exact-match predicate correctly reports the two modules as distinct. Both
runs' full stdout/stderr are captured verbatim in the named log files;
each ends with an explicit `exit=<N>` line captured via `rc=$?` (never after
a pipe, per §11.4.6).

## (c) Both-directions falsifiability proof

Captured in `1_green_baseline.log` / `2_red_reproduction.log` (`S1a`/`S1b`/`S1c`
lines), reproduced here:

```
S1a genuine recurrence caught (identical paths)          : yes   (want yes)
S1b prefix-lookalike NOT falsely caught by exact-match   : yes   (want yes)
S1c OLD substring predicate DOES misfire on the lookalike : yes   (want yes)
```

- **Genuine recurrence caught**: `module_paths_identical("dev.helix.code",
  "dev.helix.code")` returns true — a real collision would be caught.
- **Prefix-lookalike NOT falsely caught**: `module_paths_identical(
  "dev.helix.code", "dev.helix.codebase")` returns false — the exact-match
  predicate correctly distinguishes a merely-prefix-sharing name (the case
  the original fix agent used) from a genuine duplicate.
- **Negative control**: the reconstructed OLD substring predicate, applied to
  the same lookalike fixture, DOES misfire (reports a match) — proving the
  gate's "old vs new" comparison is alive and the RED reproduction is
  meaningful, not vacuous.

## (d) Mutation proof, including confirmation the mutation applied

Sequence, each step captured to its own numbered log:

1. `3_mutation_proof.log` — pre-mutation `git diff --stat` on
   `scripts/lib/module_identity.sh`: **empty** (exit 0, no output).
2. Mutation applied: `module_paths_identical()` changed from
   `"$a" == "$b"` to `"$a" == *"$b"*` (reintroduces containment/substring
   matching — the exact historical bug class).
3. `3_mutation_proof.log` (appended) — post-mutation `git diff`: **non-empty,
   15 lines**, confirming the mutation actually landed on disk before the
   gate was invoked (guards against the "unapplied mutation looks like a
   passing gate" failure mode named in the task).
4. `4_gate_under_mutation.log` — gate run against the mutated lib, default
   RED_MODE=0 (standing guard): **GREEN FAIL, exit 1**. The mutation is
   caught at the real production call site (the root-vs-inner comparison
   itself reports a false collision), not merely by the synthetic S1
   fixtures — S1a/S1b still read "yes"/"yes" under this specific mutation
   direction (the mutated predicate is `"$a" == *"$b"*`, i.e. "root contains
   inner", which does not flip the S1 lookalike fixture's outcome because
   the lookalike is passed as the *longer* argument there). This is reported
   honestly below as a finding, not smoothed over.
5. `5_restore_proof.log` — `git checkout -- scripts/lib/module_identity.sh`,
   then `git diff --stat`: **empty again** (exit 0), confirming clean
   restore.
6. `6_gate_after_restore.log` — gate re-run, RED_MODE=0: **GREEN PASS, exit
   0**.

## (e) Commit sha + evidence dir

- Evidence dir: `docs/qa/hxc220_module_identity_gate_20260805T120714Z/`
  (this directory).
- Commit sha: recorded in the commit message / reported separately after
  `git commit --only` runs (this file is written before that commit).

## (f) Doubts / honest boundary

- **S1a/S1b did not catch the planted mutation.** The mutation I planted
  (`"$a" == *"$b"*`, i.e. "does the first argument contain the second") is
  caught by the real GREEN check (root path contains inner path) but is
  *not* caught by the S1 synthetic fixtures, because in `S1b` the lookalike
  string is passed as the *second* argument and is *longer* than the first,
  so "first contains second" never fires there. The gate as a whole still
  correctly FAILs (exit 1) because the real call site is what matters, but
  this means the S1 falsifiability block is directionally asymmetric: it
  would not by itself catch every possible reintroduction of substring
  logic — specifically not one shaped like `"$a" == *"$b"*` tested only via
  the synthetic fixtures in isolation. I consider the gate sound because the
  invariant that actually matters (the real root-vs-inner call) is proven to
  FAIL under this mutation, and I would want a reviewer to independently
  re-derive this rather than take my word for it.
- **Scope is source-layer only.** As documented in the gate's own header,
  this proves `module_paths_identical()` is exact-match and correctly wired
  into the two real go.mod comparisons on disk. It does not execute
  `hxc159_env_facts.sh` end-to-end (that probe has unrelated environment
  preconditions — SSH reachability to a GitHub remote, etc.) and module
  identity is inherently a build-time, not runtime, concern, so there is no
  separate ARTIFACT/RUNTIME layer to additionally prove here.
- **HXC-199 and HXC-220 are intentionally left OPEN** by this run, per the
  task's explicit instruction. This authored the guard; it did not certify
  or close either item.
