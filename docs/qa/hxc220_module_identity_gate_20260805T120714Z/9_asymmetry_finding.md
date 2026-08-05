# HXC-220 — argument-order asymmetry in the HXC-199 gate: found, reproduced, closed

**Session:** respawn after the predecessor agent was killed mid-sentence by a session limit (§11.4.147).
**Gate under review:** `scripts/gates/hxc199_module_identity_exact_match_gate.sh` (landed in `40c8d5ce`).
**Scope of this document:** the falsifiability block only. HXC-199 and HXC-220 both remain **OPEN** — see
"Deliberately not closed" below.

---

## 1. The finding (reproduced independently, not taken on trust)

The predecessor's last recorded words were that it had found an asymmetry: `module_paths_identical a b`
and `module_paths_identical b a` were not equally protected. That is confirmed, and the consequence is
more serious than "one untested direction".

`module_paths_identical()` can be regressed back to substring behaviour in **either** of two shapes:

| | mutation | caught by the pre-fix S1b fixture? |
|---|---|---|
| **M1** | `[[ "$a" == *"$b"* ]]` | **NO** |
| **M2** | `[[ "$b" == *"$a"* ]]` | yes |

The pre-fix fixture called `module_paths_identical "dev.helix.code" "dev.helix.codebase"` — short
argument first. Under **M1** that asks "does `dev.helix.code` contain `dev.helix.codebase`?" → no → the
fixture reports clean. The falsifiability block was blind to M1 entirely.

## 2. Why the incidental coverage did not rescue it

M1 *was* still caught on today's tree — but by the **real call site** (`GREEN FAIL`), not by any test.
That catch exists only because today's root path `dev.helix.code/meta` happens to **contain** today's
inner path `dev.helix.code`. It is a property of the current data, not of the guard.

Removing that coincidence — the same M1 mutation, with the root module renamed to a non-containing
value `dev.helix.meta` (an entirely legitimate future rename) — the **pre-strengthening gate**:

```
S1b prefix-lookalike NOT falsely caught by exact-match  : yes   (want yes)
GREEN PASS — ... are exact-match distinct, verified via module_paths_identical() — never a
             substring/prefix test.
EXIT=0
```

**Exit 0, with a substring predicate installed, while printing "never a substring/prefix test".**
That is a latent PASS-bluff (§11.4 / §11.4.201): a future legitimate rename would have silently disarmed
the guard, with no signal to anyone. Full transcript: `10_pre_fix_asymmetry_reproduction.log`.

## 3. What was changed

In the falsifiability block (`S1`), and nowhere else:

- **S1b** — the prefix-lookalike fixture is now exercised in **both argument orders**.
- **S1d** (new) — the **historical R-26 pair itself** (`dev.helix.code/meta` vs `dev.helix.code`) is
  pinned as a **literal fixture**, both orders, so the block keeps proving exact-match semantics
  regardless of what the on-disk `go.mod` files happen to declare. This is what converts the
  data-dependent catch of §2 into a structural one. It also catches a family the lookalike fixture
  cannot: first-path-segment truncation (`${a%%/*} == ${b%%/*}`), which passes S1b but conflates
  exactly the two paths HXC-187 created to be distinct.

The guarded predicate in `scripts/lib/module_identity.sh` was **not** modified — it was already correct.
The gate's on-disk comparison, its RED_MODE=1 reproduction, and its exit-code contract are unchanged.

## 4. Verification (exit codes personally observed this session)

Live tree — `11_live_tree_red_green_and_paired_mutation.log`:

| run | exit | meaning |
|---|---|---|
| `RED_MODE=1` | **0** | historical substring false positive still reproduces on this tree |
| `RED_MODE=0` | **0** | standing guard green, unmutated |
| `RED_MODE=0` under paired M1 mutation | **1** | mutation caught — **by S1b and S1d**, not by the call site |
| after restore | **0** | green again; `git diff` vs HEAD = 0 lines |

The mutation was confirmed **applied** by a real non-empty `git diff` (13 lines) *before* the failing
result was trusted — an unapplied mutation is indistinguishable from a passing gate, and that failure
mode has already produced three false passes in this project.

Sandbox battery — `9_argument_order_mutation_battery.log`: **8 mutants × 2 data scenarios (real paths
and the renamed-root scenario) = 16 runs, all exit 1**, each caught by a falsifiability fixture. The
M1 + renamed-root cell, which was exit **0** before, is now exit **1**.

Sweep wiring re-verified: `bash scripts/verify-all-constitution-rules.sh --gate=G28` → `PASS`, exit 0.

## 5. Deliberately not closed

**HXC-199 and HXC-220 remain OPEN.** The predecessor declined to be both author and certifier of this
gate, and this session inherits that constraint rather than dissolving it: the strengthening above is
*more* author-side work on the same gate, so certifying it here would be exactly the conflict
§11.4.142 / §11.4.209 forbid. Closure needs an independent review (Fable @ xhigh, or Opus @ xhigh).

## 6. Residual doubt — stated, not smoothed over (§11.4.6)

- **The fixtures are literals.** S1b/S1d pin the strings that mattered historically. A regression whose
  behaviour differs from exact-match on *neither* fixture pair would still pass. The battery covers the
  8 shapes I could construct; it is not a proof of exhaustiveness.
- **Honest boundary unchanged.** This is a SOURCE-layer guard. It does not execute
  `hxc159_env_facts.sh`, and it proves nothing about a built artifact (§11.4.108) — module identity is
  a build-time concern, as the gate's own header already states.
- **`--explain` omits G15–G28 (pre-existing, not from this work).**
  `scripts/verify-all-constitution-rules.sh --explain` claims to print gate descriptions but lists only
  G1–G14, while 28 gates are implemented. A reader running `--explain` would conclude half the gates do
  not exist. G28 is affected but is *not* the cause — the drift predates it. Not fixed here (14 gates'
  descriptions is separate work, and the workable-items DB is concurrently modified by other live
  agents); **recommended for filing as its own item.**
