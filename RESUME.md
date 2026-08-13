# RESUME — session resumption record (§11.4.131)

**Rev 18 · updated 2026-08-13 ~13:50 +05.**
Supersedes rev 17 (~13:40). §6 corrected — the §11.4.150 gate is CLEAR for
all seven items rev 15 listed as needing a pass. Live streams at rounds 12-18.

> **Governing rule for this document.** Every count, list, hash and stream-state
> below is a **snapshot**. Re-derive before acting on any of it — commands are
> inline. Counts in this project have moved under scrutiny repeatedly; one item
> carried four disagreeing published figures, and this session two of my own
> measurements were later disproved (§7).

---

## 1 · Start here

```bash
cd /home/milos/Factory/projects/tools_and_research/helix_code
git fetch --all --prune
git log --oneline -1                       # expect d2547daf or later
git status --porcelain                     # ~10 entries, all live-stream work
sqlite3 docs/workable_items.db "SELECT COUNT(*) FROM items WHERE status NOT IN ('Fixed (→ Fixed.md)','Implemented (→ Fixed.md)','Completed (→ Fixed.md)','Obsolete (→ Fixed.md)');"
```

Read `.remember/remember.md` and `docs/CONTINUATION.md` first if present.
**Derive the agent alias, never recall it** — it changed mid-session from `claude1`
to `claude4` and the PreToolUse guard blocked a dispatch carrying the stale label.
Get it from `scripts/multitrack/track_branch_label.sh`; the §11.4.182 form is
`(T1/main - <alias> - <model> - <effort>)`, and the guard enforces it on `Agent`
**and** `TaskCreate` descriptions.

## 2 · State at snapshot

| repo | HEAD | published |
|---|---|---|
| main | `d2547daf` | all four mirror endpoints verified |
| `submodules/helix_agent` | `36009d62` | unchanged this window (work uncommitted) |
| `submodules/llms_verifier` | `9bf5457d` | unchanged this window (work uncommitted) |
| `submodules/helix_qa` | `0634b1b` | unchanged this window (work uncommitted) |

Tracker: **516 items, 109 open** — 2 Critical, 31 High, 45 Medium, 31 Low.

**The 2 Critical:** `HXC-227` (published provider key, `Operator-blocked` —
**only the operator can clear it**) and `HXC-243` (`Ready for testing`, fix
landed at `0634b1b`, closure gated on §11.4.150).

## 3 · Live streams — uncommitted work on disk

Four review/remediation streams. **Work is preserved but uncommitted.** Verify
by hash before resuming any of them.

| item | scope | state at snapshot |
|---|---|---|
| HXC-282 | meta `scripts/git_hooks/`, `scripts/lib/` | **round 18** — r17 found the credential scanner's *tree mode* still fail-open, same class r16 fixed in the same file |
| HXC-286 | `helix_agent`, `internal/netaddr/` + 9 files | **round 16 due** — substance verified exact; r14's own +49-line edit shifted all five of its line citations |
| HXC-298 | `llms_verifier` `clientip/` + 3 consumers | **round 12** — r11 NO-GO on 7; the decision to drop the corroboration guard is CONFIRMED, do not revisit |
| HXC-305 | `helix_qa/pkg/testbank/` | **round 12** — r11 NO-GO *narrowly*, 2 self-description findings, **zero code defects**; closest to commit |

Each has now survived 11–18 rounds and **every round has found something real —
not one has returned a clean GO first time.** Do not read "round N" as "nearly
done"; read it as "this area rewards scrutiny."

**The strongest recurring signal: a correction carries this defect class at the
same rate as any other change.** Twice now a round fixed a false claim and
introduced a fresh instance of the identical class *in the same edit* — once four
lines from the sentence warning against it, once by adding comment lines that
invalidated the very citations the new comment depended on. **Always re-review
the correction.**

## 4 · The finding that most changes what a green build means

**The pre-commit gate was failing open.** The §11.4.84 marker sweep was blind to
any marker in a blob above roughly 12 KiB. `grep -q` exits at first match → the
producer dies of SIGPIPE → `set -o pipefail` promotes 141 → `if` reads it as
*no match*. Proven end-to-end: a 30 KB file carrying a genuine paired-mutation
residue marker was **ALLOWED** before the fix and **BLOCKED** after, through a
real `git commit`.

> **Note for whoever edits this file.** Do not quote the residue marker
> literally here — the live hook scans staged blobs for it and will refuse this
> document, exactly as it refused rev 15's first draft. That refusal is the gate
> working, not a bug; describe the marker, never spell it.

Consequence: every "residue clean" result produced through that sweep on a large
blob this session was unreliable. **Checked directly** — all 95 commits across
all four repos since 2026-08-09 carry **zero** residue-shaped added lines. Limit
stated: that finds residue that was *added*; residue that works by *deleting*
code leaves nothing to grep for, which is the gap HXC-282 exists to close.

**A second hole survives that fix** (round 8, being repaired in round 9):
`pre-commit:500` gates on `[ -f "$f" ]` — worktree existence — while its own
comment says it scans the staged blob. Stage a marker file, `rm` it, commit →
**allowed**.

## 5 · Path to a release build

**Mine:** drive the four streams to a zero-finding GO (§11.4.134) → full-suite
retest **on a quiescent host** → remaining §11.4.150 research passes.

**Operator only:** rotate `HXC-227` and `HXC-168` · §11.4.185 manual-QA sign-off
· the 1.1.0 retro-tagging decision for three submodules.

**Open decisions, none blocking:** `HXC-310` / `HXC-318` (mount the middlewares
or record them as deliberately absent) · `HXC-321` (below) · the orphan fixture
`scripts/lib/testdata/hxc282_pre_r1_mutation_baseline.sh` — a 28 KB never-tracked
frozen duplicate; round 8 recommends removal, §11.4.122 says ask first, and it
blocks nothing.

## 6 · §11.4.150 research passes — the closure gate

**Five** families now have one; each covers several items.

| artifact | covers | what it found |
|---|---|---|
| `docs/research/address_composition_family_20260812/` | HXC-268/280/283/284 | CVE precedent, same address shape |
| `docs/research/client_ip_trust_family_20260813/` | HXC-292/298/299 | **HXC-321** |
| `docs/research/test_validity_family_20260813/` | HXC-243/287/291 | **HXC-322** |
| `docs/research/derived_identifier_family_20260813/` | HXC-268/272/281 | **HXC-326** |
| `docs/research/accessor_side_effect_family_20260813/` | HXC-274 | negative, with the instrument stated |

**The §11.4.150 gate is CLEAR for all of these** — verified 2026-08-13: every one
of HXC-268/272/274/280/281/283/284 is named in a pass header with substantive
coverage. Rev 15's "still needing a pass: HXC-268, 272, 274, 281, 283" was
**stale**; those items are Queued/Ready-for-testing on *validation*, not research.

> **Two grep traps in this directory, both of which I fell into.** (1) The
> 2026-08-12 pass heads its list `**Scope:**` while the 2026-08-13 ones use
> `**Covers:**` — grepping only for `Covers` makes the oldest pass look like it
> covers nothing, and made HXC-283 look uncovered when it has a section titled
> "HXC-283's exact code". (2) `grep -rl HXC-283 docs/research/` returns three
> hits that read like coverage but are two "bears on" mentions plus one real one.
> **Match on the header line and confirm substantively; never on a bare ID grep.**

**HXC-321 (High), from the client-IP pass.** Gin trusts **all** proxies by
default (`0.0.0.0/0`, `::/0`) — the unsafe posture is the default, so no code
needs writing to be exposed; the one line that makes it safe has to be. Six
production Gin engines in this tree; **zero** call `SetTrustedProxies`. Traced
to a live path: `cmd/server/main.go:156` → `gin.New()` → an **unauthenticated**
`POST /login` → `handlers.go:218` `c.ClientIP()` → `auth.go:230` → persisted at
`auth_db.go:197`. Not injection (`ParseIP` rejects non-IPs) — the stored address
is well-formed and entirely caller-chosen.

## 7 · Corrections carried forward — do not re-inherit the old versions

- **HXC-309 is a RACE, not a threshold, and is producer-independent.** 40 reps
  per size: 12978 clean 40/40, **16018 fails open 16/40**, 20018 fails open
  40/40. It tracks *needle position*, not file size. `git show`, `cat` and
  builtin `printf` all fail open, so the recorded "belongs to the built-in
  writer" conclusion is **disproved**.
- **My own non-reproduction of it was a false negative.** My filler was one
  unbroken 40 KB line; line-oriented grep cannot early-exit on a single line, so
  it must buffer to EOF — the exact condition under which the fault cannot
  occur. Any re-test **must** use content with line breaks and vary both size
  and needle position. A lucky single probe has now produced a false all-clear
  twice.
- **The address defect breaks on the UNBRACKETED form.** A bracketed host
  composes correctly under `Sprintf` *by coincidence*. The `[::1]:6333:6333`
  shape cited in older notes requires a host that already carries a port.
- **`atmosphere.json`'s "129" is a CASE COUNT, not http conversions** (it has
  zero `http:` steps). Executable steps are 49/395 JSON vs 24/566 YAML.
- **`grep` is a wrapper function** in interactive shells and does **not**
  propagate into a script run as a file.
- `cmd | head && echo $?` reports **head's** status. Also `rc=$?` is itself a
  command and **resets `PIPESTATUS`** — declare the variable before the pipeline.
- A count is a LEAD, never a fact: 16→32, 46→50, 60→41, "133 passing" measured
  as rc=1.

## 7b · URGENT — ~2,000 lines of reviewed work are UNTRACKED

Every stream's new package is untracked, in a checkout where four agents run
concurrently. **23 untracked files, 6.9 MB**, including:

- `submodules/llms_verifier/llm-verifier/clientip/` — the whole package plus its
  1,270-line test suite, **eight review rounds** of work
- `submodules/helix_agent/internal/netaddr/` — the new package
- `submodules/helix_qa/banks/.bank-id-floor.txt` + the HXC-305 guard suite

Git holds **no copy**. A single `git clean -fdx`, stray checkout, or
`git add -A` mishap destroys all of it with no recovery path. Found by the
HXC-298 round-7 reviewer, who correctly rated it above the code findings.

**Backed up out-of-tree** to `scratchpad/untracked-backup-<UTC>/` (current path in
`scratchpad/LAST_UNTRACKED_BACKUP`; each snapshot carries a self-verifying
`MANIFEST.sha256`, and the earlier snapshot is retained). Session-local; a
stopgap, not the fix.

> **Trap that silently halved the first refresh.** `git status --porcelain`
> collapses an untracked *directory* into one `dir/` entry, so a copy loop
> guarded by `[ -f "$f" ]` skips it entirely. The first refreshed snapshot
> contained **zero** files from `internal/netaddr/` and **zero** from
> `clientip/` — the two most-reviewed packages — while reporting 52 files
> copied and looking complete. Use **`git status --porcelain -uall`**, which
> expands directories into individual paths, and verify the critical packages
> by name afterwards rather than trusting the file count.

**The fix is to commit each package into its own submodule the moment its
stream returns a clean GO.** Deliberately not done mid-review: four agents were
briefed that this work is uncommitted, and changing that under them is worse
than the exposure. If you are resuming and any stream is idle, commit it.

**Staging hazard — verify the staged set is self-consistent before committing.**
In `helix_qa` the index held *only* `banks/.bank-id-floor.txt` while `HEAD`'s
`loader.go` contains **zero** occurrences of `checkBankIDFloor`/`bankIDFloorFile`
and no `hxc305_*` test file exists at `HEAD`. Committing as-staged would have
landed a 3,046-id data file with **neither its enforcement code nor its guards** —
inert, and indistinguishable in the log from the fix landing. Before any commit
here, confirm a checkout of the index *alone* would compile and run the guards,
and exclude sibling-stream files (`cmd/helixqa/{http,main}.go`,
`pkg/testbank/manager.go`, `loader_test.go`).

## 7c · Inter-round baselines: say which KIND of hash, and WHERE it lives

**Corrected in rev 17 — rev 16's version of this section was wrong, and the way
it was wrong is the lesson.**

Hashes quoted between rounds here are **content sha256s, not git objects**.
`internal/netaddr/` is untracked (`git ls-files` → 0 rows), so `git cat-file` will
never resolve them and a round that tries concludes "invalid object name". That
much rev 16 had right, and the reviewer who first flagged it was right to refuse
to restate a claim it could not reproduce.

**What rev 16 got wrong: I then asserted the artifacts were gone. They are not.**
Every one resolves inside `scratchpad/`, in per-round backups:

| hash | file | lines |
|---|---|---|
| `183c144b` | `r12-backup-20260813T080906Z/netaddr_test.go` | 907 |
| `ee5ebf07` | `r12-backup-20260813T080906Z/ports_test.go` | 377 |
| `417a3662` | `r13-backup/netaddr_test.go` | 935 |
| `9bb3ff06` | `untracked-backup-20260812T215041Z/…/netaddr.go` | — |

So the earlier round's exec-diff result **is** reproducible. My "resolves to
nothing" came from a `find` scoped through `LAST_UNTRACKED_BACKUP` — **one of
seventeen** backup directories in scratchpad, and not the one holding them. A
negative from a search that could not have found the target is a fact about the
search. **Run a positive control first:** searching for a hash you already know
exists would have exposed the scope error immediately.

The durable point survives in a weaker, truer form: these baselines live only as
long as the **session-local** scratchpad, and are findable only if the citation
says where. So when quoting a hash between rounds, give **kind and location** —
`content sha256, scratchpad/r13-backup/netaddr_test.go` — never a bare hex
prefix. Committing the work is what replaces this with `git show HEAD:<path>`.

## 8 · Standing hazards

- **HXC-247 stays unassigned until HXC-248 closes.** `:8100` is held by
  `llm-verifier`, not the agent; HXC-248's teardown guard probes `:8100` and so
  reaches the wrong process. That mismatch protects 41+ live containers. Fixing
  HXC-247 first would ARM the teardown.
- Live stack: 41+ containers. **`helixllm-coder` holds a 30B model — never
  restart it.** No sudo, rootless podman only, no host power-state change, no
  force-push (§11.4.113, absolute).
- Never fire real provider calls — startup discovers live credentials
  (§11.4.101). Use `httptest` loopback or a closed port.
- `git add -A` is forbidden here (§11.4.30) — four streams hold live work.
- ~20 `tools/opensource/*` pointer drifts in `helix_qa` are pre-existing.

## 9 · What worked, and is worth continuing

**Independent review is carrying this work.** Every round of every stream has
returned findings; not one has returned a clean GO first time. Caught before
landing: a lock recording dead pids; a change that would have deleted 49
regression guards while all its own tests passed; an exploitable guard that was
pure decoration; a fix that regressed against its own sibling one import away;
a fix that reintroduced its own defect class in the opposite direction; and a
one-character tie-break whose mutation silently drops 204 honest-skip markers
while passing the entire package.

**~24 times this session the defect lived in the artifact's description of
itself** rather than its logic — a promise the code does not keep, a count that
disagrees with reality, a refactor claiming a uniqueness it did not achieve, a
comment contradicted ten lines away in the same package. **Every one passed its
own tests. Every one was caught by an independent party checking claim against
code.** Tests assert what code does; they never assert that the prose above it
is true. See `memory/defects-live-in-self-description.md`.

Reviews run on Fable at xhigh (§11.4.209); **Fable hit its limit, so they
dispatch to Opus at xhigh**, that clause's specified fallback.
