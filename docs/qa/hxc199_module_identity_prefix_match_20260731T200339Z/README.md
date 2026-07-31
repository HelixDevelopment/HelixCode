# HXC-199 — root-module-identity check matched a name fragment, not the whole name

## What was broken

`scripts/probes/hxc159_env_facts.sh` (FACT 4, "UNSTATED BY ITEM: the thin root
module's identity") checked whether the root meta-repo `go.mod` still
collided with the inner `helix_code/go.mod` module path by testing
`[[ "$root_mod" == *"dev.helix.code"* ]]` — a **substring** match.

HXC-187 (D-7) renamed the root module from `dev.helix.code` to
`dev.helix.code/meta`. `dev.helix.code` is a literal **prefix** of
`dev.helix.code/meta`, so the substring test still matched after the rename
and the script kept reporting `GAP-IN-ITEM` (its wording for "the collision
is still present") on a tree where the collision was already resolved.
Re-running the check would tell a reader the fix is not in — and following
that advice risks reverting already-correct work.

## Fix

- `scripts/lib/module_identity.sh` (new): `module_paths_identical()` does
  EXACT string comparison on the extracted module-path tokens, and
  `go_mod_module_path()` extracts a bare module path from a `go.mod` file.
- `scripts/probes/hxc159_env_facts.sh`: FACT 4's root-module check now calls
  `module_paths_identical "$root_mod_path" "$inner_mod_path"` instead of the
  substring glob, and the "measured" text now states the real relationship
  (`distinct` vs `identical`) instead of unconditionally asserting
  "identical to the inner module".
- `CLAUDE.md`: 3 places that described the root module as bare
  `dev.helix.code` (the pre-HXC-187 arrangement) now say
  `dev.helix.code/meta` — §3.1 Module IDs line, the repo-layout tree comment
  for the root `go.mod`, and §3.2.1's prose description of the root module.
  A 4th and 5th mention (describing the INNER module, which genuinely still
  IS `dev.helix.code`) were left untouched — they were already correct.

## Evidence in this directory

- `red_false_positive.log` — the script AS IT EXISTED AT THIS COMMIT'S PARENT
  (`git show HEAD:scripts/probes/hxc159_env_facts.sh`, i.e. the pre-fix
  artifact), re-run against the CURRENT go.mod tree (root already
  `dev.helix.code/meta`, inner still `dev.helix.code`). Verdict: GAP-IN-ITEM
  — the false positive, reproduced on demand.
- `green_fixed.log` — the FIXED script run against the same current tree.
  Verdict: CONFIRMED, with an accurate "measured" description.
- `green_recurrence_still_caught.log` — proves the exact-match predicate is
  not a matcher that never fires: (1) a genuine recurrence (root path forced
  back to literally `dev.helix.code`, identical to inner) is still flagged
  TRUE; (2) the current fixed state (prefix relationship, not identical) is
  correctly FALSE; (3) an unrelated name that happens to share the same
  prefix (`dev.helix.codebase`) is also correctly FALSE — the fix does not
  overcorrect into ignoring genuine duplicates or falsely matching unrelated
  siblings.

## Honest gaps (§11.4.6)

- The standing regression guard required by §11.4.135 (a permanent,
  registered gate with a RED_MODE polarity switch, wired into
  `scripts/verify-all-constitution-rules.sh` as a numbered G-gate) is **NOT
  YET LANDED**. This commit lands the source fix + the handbook fix + this
  evidence only, under explicit operator time pressure (a session-ending
  request to land immediately). The guard is the next actionable item for
  whichever session picks this up next — everything needed to write it is in
  this directory and in `scripts/lib/module_identity.sh`'s doc comment.
- This evidence run used a live, uncontrolled working tree shared with other
  concurrent agents (see the commit's `git status --porcelain` at commit
  time); only the two files this ticket owns
  (`scripts/probes/hxc159_env_facts.sh`, `scripts/lib/module_identity.sh`)
  plus `CLAUDE.md` were staged and committed, verified by `git diff` on each
  path before staging to confirm no foreign changes were mixed in.
