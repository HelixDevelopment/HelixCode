# Where this directory's 2026-08-05 evidence actually landed (and why)

**Short version:** the four files below are committed and byte-identical to what was captured, but they
landed in a **different agent's commit** than the code they document. Nothing was lost. This note exists
so a future reader following the trail is not misled by the mismatch.

| file | commit that contains it |
|---|---|
| `9_asymmetry_finding.md` | `149d4e0f` |
| `9_argument_order_mutation_battery.log` | `149d4e0f` |
| `10_pre_fix_asymmetry_reproduction.log` | `149d4e0f` |
| `11_live_tree_red_green_and_paired_mutation.log` | `149d4e0f` |
| the gate change they certify | **`f021f1a4`** |

`149d4e0f` is *"docs(qa): capture live HXC-166 advisory census for helix_agent (208 open)"* — an
unrelated concurrent agent's commit.

## What happened (§11.4.84, observed live)

Committing an untracked file requires `git add` first (`git commit --only` cannot stage untracked
paths). That `git add` writes to the **shared `.git/index`**. In the window between that `git add` and
this session's `git commit --only`, a concurrent agent committed — and its commit swept these four
staged-but-not-yet-committed files in with its own.

This is the §11.4.84 forensic pattern exactly, seen from the other side: the canonical case is *your*
`git add` sweeping in someone else's residue; here it was *someone else's commit* sweeping in this
session's staged evidence. `git commit --only <paths>` protects the **committer** from the shared index,
but it does not protect **staged files** from another agent committing first. In a shared checkout with
live concurrent agents, `git add` of untracked files is itself the exposure.

## Not repaired by rewriting history

Deliberately. `149d4e0f` is another live agent's commit; rebasing or amending it to relocate these files
would rewrite history under a concurrent worker (§9.2 data safety, §11.4.113). The content is intact in
git, which is the property that matters. Attribution is corrected by this note, not by history surgery.

## Mitigation used for this note

This file was committed through a **private index** rather than the shared one:

```bash
export GIT_INDEX_FILE=$(mktemp -u)   # a scratch index, not .git/index
git read-tree HEAD
git add -- <path>
git commit -m ...                    # shared .git/index never touched
```

That closes the window entirely: the file is never visible in the shared index, so no concurrent commit
can sweep it. Recommended for any agent committing new untracked files in a shared checkout while other
agents are live.
