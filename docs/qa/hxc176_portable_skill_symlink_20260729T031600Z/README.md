# HXC-176 — the skill installer baked the authoring machine's paths into tracked links

| | |
|---|---|
| Revision | 1 |
| Created | 2026-07-29 |
| Last modified | 2026-07-29 |
| Status | active |
| Item | HXC-176 (Bug / Medium) → `Fixed (→ Fixed.md)` |
| Scope | `constitution/scripts/`, `constitution/skills/*/register.sh` |
| Submodule commit | `01ba15db` (constitution, branch `main`, not pushed) |
| Host | Linux 6.12.41; bash 5.x |

## Table of contents

- [1. The defect](#1-the-defect)
- [2. Why a link that works in place proves nothing](#2-why-a-link-that-works-in-place-proves-nothing)
- [3. Full defect surface — wider than the item named](#3-full-defect-surface--wider-than-the-item-named)
- [4. The fix](#4-the-fix)
- [5. RED → GREEN evidence](#5-red--green-evidence)
- [6. Paired polarity proof](#6-paired-polarity-proof)
- [7. Convergence check](#7-convergence-check)
- [8. What was not verified](#8-what-was-not-verified)

## 1. The defect

Every skill-registration generator computed an **absolute** directory and did
`ln -sf "$abs_dir" "$link"`, planting a link in the consuming project whose stored
target was the installing machine's path. Such a link resolves only on the host
that created it.

This is not theoretical. Consuming-project commit `1bc82267` (2026-06-21) committed
`skills/media-validator` pointing at `/Volumes/T7/Projects/helix_code/...` — authored
on macOS against an external volume — and it had **never once resolved** in this Linux
checkout. The same commit rewrote a codegraph path the same way, so the pattern belonged
to the generator, not to one slip.

Commit `cc622528` repaired that single link by hand. The **generator** was untouched, so
re-running it recreated the defect. This closes that gap.

## 2. Why a link that works in place proves nothing

An absolute link is perfectly functional on the machine that created it. The invariant
only acquires meaning under **relocation**. The guard therefore generates links at one
absolute path, `mv`s the whole tree to a different absolute path so the original no
longer exists, and re-derives every link from disk with `readlink` — asserting both that
the stored target is relative and that it still resolves *inside* the relocated tree.

## 3. Full defect surface — wider than the item named

The item named `constitution/skills/*/register.sh`. The sweep found **six** link-creating
sites across two generator families, all affected:

| Site | Family |
|---|---|
| `scripts/install_cli_agent_plugins.sh` → `link_skills` | shared installer (used by `action-prefix-system`, `reporting-workable-items`, `workable-item-lifecycle`) |
| `scripts/install_cli_agent_plugins.sh` → `link_agent_commands` | shared installer (gemini/qwen/codex command links) |
| `skills/media-validator/register.sh` | direct |
| `skills/scheduled-work-queue/register.sh` | direct |
| `skills/session-sync/register.sh` (skill link) | direct |
| `skills/session-sync/register.sh` (convenience link) | direct |

## 4. The fix

New `constitution/scripts/portable_symlink_lib.sh` provides `hc_relpath` and
`hc_ln_relative`; all six sites now call it.

Constraints honoured:

- **Project-agnostic** (§11.4.28 / CONST-051(B)) — the helper takes two paths and returns
  a path. It knows nothing about any consuming project's name, layout or repository shape,
  so the shared submodule learns nothing HelixCode-specific.
- **Portable shell** — `realpath --relative-to` is GNU coreutils only; BSD/macOS lacks it,
  and the forensic case proves these scripts genuinely run on macOS. The computation is
  pure POSIX shell (`dirname` + parameter expansion), no coreutils-specific flags.
- **Honest fallback** (§11.4.6) — when two paths share no ancestor deeper than `/`, no
  relative expression can survive relocation; the helper warns and emits absolute rather
  than pretending it solved the problem.

## 5. RED → GREEN evidence

`evidence/01_RED_prefix_greenmode.txt` — the guard run against the **pre-fix** tree:

```
[absolute] .../green/B/.claude/skills/action-prefix-system -> .../green/A/constitution/skills/action-prefix-system
[DANGLING after relocation] .../green/B/.claude/skills/action-prefix-system
   ... (6 of 6 planted links absolute AND dangling) ...
FAIL  GREEN: at least one link planted by the real generators is host-absolute or
      fails to resolve after relocation — the generator bakes in the authoring machine's path
== summary: PASS=0 FAIL=1 SKIP=0 ==      exit 1
```

`evidence/02_GREEN_postfix.txt` — same guard, same command, after the fix:

```
[relative] .../green/B/.claude/skills/action-prefix-system -> ../../constitution/skills/action-prefix-system
[relative] .../green/B/scripts/sync_remote_session.sh -> ../constitution/skills/session-sync/session-sync.sh
   ... (7 of 7 planted links relative and resolving) ...
PASS  GREEN: every link planted by the real generators stores a RELATIVE target and still
      resolves inside the tree after relocation to a different absolute path
== summary: PASS=1 FAIL=0 SKIP=0 ==      exit 0
```

Seven links rather than six because the fixed harness also creates the consuming project's
`scripts/` directory, which exercises `session-sync`'s convenience link that the first run
silently skipped.

GREEN re-ran twice more, `exit 0` both times (§11.4.50 determinism).

## 6. Paired polarity proof

`evidence/03_RED_mode1_strawman.txt` — `RED_MODE=1` runs a strawman emitting the exact
pre-fix construct and asserts relocation **breaks** it, proving the harness is not blind:

```
PASS  RED cross-check: construct absent from working tree but present in git history —
      strawman mirrors code that genuinely shipped
PASS  RED: strawman absolute-path generator BREAKS under relocation — harness demonstrably
      detects the defect class
```

The cross-check greps the real generators for the pre-fix construct and falls back to
`git log -S` once the working tree is fixed, so the simulation cannot silently drift from
what actually shipped.

## 7. Convergence check

`evidence/04_relpath_units_and_convergence.txt` — six `hc_relpath` unit cases pass, and:

```
tracked : ../constitution/skills/media-validator
on-disk : ../constitution/skills/media-validator
```

The fixed generator now emits **byte-identically** what `cc622528` repaired by hand.
Re-running the installer reproduces the correct state instead of recreating the defect —
the generator is cured, not just its symptom.

## 8. What was not verified

- **No POSIX-`sh` execution proof.** `dash` is not installed on this host, so the pure-POSIX
  claim rests on construct inspection plus `bash -n`, not on a strict-POSIX interpreter run.
- **No macOS/BSD execution proof.** The portability fix is motivated by a macOS-authored
  defect but was exercised only on Linux. `realpath --relative-to` was avoided precisely
  because it is unavailable there; that avoidance is by inspection, not by execution.
- **The real generators were never run against the live project root.** Seven agents share
  this checkout, so running an installer against it would mutate shared state. Every run
  used a scratch replica built by copying the real generators verbatim.
- **The guard described in the item's text ("a guard now catches a dangling or host-absolute
  link") could not be located** anywhere under `scripts/`, `scripts/gates/`, or the
  constitution's script tree. The fix stands on its own regardless; the polarity guard added
  here is the mechanical enforcement.
