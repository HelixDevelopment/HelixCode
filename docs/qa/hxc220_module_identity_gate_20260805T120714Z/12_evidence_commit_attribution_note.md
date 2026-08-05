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
TMPIDX=$(mktemp -u)                          # a scratch index, not .git/index
GIT_INDEX_FILE="$TMPIDX" git read-tree HEAD
GIT_INDEX_FILE="$TMPIDX" git add -- <path>
GIT_INDEX_FILE="$TMPIDX" git commit -m ...   # shared .git/index never touched
rm -f "$TMPIDX"
git reset -q -- <path>                       # ← MANDATORY follow-up, see below
```

That closes the sweep window entirely: the file is never visible in the shared index, so no concurrent
commit can claim it.

### The follow-up `git reset` is not optional — measured, not assumed

Committing through a private index moves `HEAD` **without** updating the shared `.git/index`. The shared
index therefore still has no entry for the new path while `HEAD` does, and git reads that difference as
a **staged deletion**. Observed immediately after the commit above:

```
$ git diff --cached --stat -- <path>
 .../12_evidence_commit_attribution_note.md | 50 ----------------------
 1 file changed, 50 deletions(-)
$ git status --porcelain -- <path>
D  .../12_evidence_commit_attribution_note.md
?? .../12_evidence_commit_attribution_note.md
```

The next concurrent agent to commit the whole index would have **deleted the file that was just
committed** — trading the sweep hazard for a worse deletion hazard. `git reset -q -- <path>` refreshes
the shared index entry from `HEAD` and clears it (verified: `git diff --cached` empty afterwards, file
still on disk and still in `HEAD`).

So the pattern is *private index to commit, then `git reset` to resync* — never the first half alone.
