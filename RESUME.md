# RESUME — session resumption record (§11.4.131)

**Rev 20 · updated 2026-08-14 ~13:40 +05.**
Supersedes rev 19. **§7b is CLOSED** — the ~2,000 lines of reviewed work that
rev 19 flagged as untracked are now committed and pushed. Every repository is
clean. Two committed packages carry open NO-GO findings, stated in their own
commit messages; see §3.

> **Governing rule for this document.** Every count, list, hash and stream-state
> below is a **snapshot**. Re-derive before acting on any of it — commands are
> inline. Counts here have moved under scrutiny repeatedly: this session alone,
> "0 Critical" became 3, "8 landed-but-open" became 16, a 35-hit grep became 16,
> and a `validate` I reported as rc=0 was rc=1 all along (I read `$?` after a
> pipe, which reports `tail`). See §7.

---

## 1 · Start here

```bash
cd /home/milos/Factory/projects/tools_and_research/helix_code
git fetch --all --prune
git log --oneline -1                       # expect a47b9e98 or later
git status --porcelain                     # expect EMPTY
git submodule foreach --quiet 'git status --porcelain | head -1'   # expect EMPTY
sqlite3 docs/workable_items.db "SELECT COUNT(*) FROM items WHERE current_location='Issues' AND status NOT LIKE '%Fixed.md%';"
```

Read `.remember/now.md` first, then this file. `docs/CONTINUATION.md` if present.

**Instrument warning, measured 2026-08-14 — this one bites every count you take.**
`grep` is a **shell function** in the agent/tool shell (ugrep 7.5.0 with
`--ignore-files --hidden -I`) and **silently skips gitignored paths**. Measured in
a throwaway repo with the same needle in `ignored/` and `tracked/`:
bare `grep -r` → **1 hit**, `/usr/bin/grep -r` → **2**. It is **not exported**
(`export -p | grep -c BASH_FUNC_grep` = 0), so `bash script.sh` children resolve
real GNU grep 3.11 and are unaffected. **Use `/usr/bin/grep` for every inline
count and state which instrument produced each number.** I got this backwards
twice before measuring it.

---

## 2 · State at snapshot

| repo | HEAD | worktree |
|---|---|---|
| meta | `a47b9e98` (13 ahead of origin/main at time of writing) | clean |
| `submodules/helix_agent` | `ba578311` | clean, pushed |
| `submodules/helix_qa` | `7483a69` | clean, pushed |
| `submodules/llms_verifier` | `109265f8` | clean, pushed |

Tracker: **407 closed / 131 open** — 3 Critical, 42 High, 51 Medium, 35 Low.
DB sha256 (first 16) `cb8e8996172f8e17`.
Fleet: **41 running / 57 total**; `helixllm-coder` Up 40h — **never disturb it**.

The 3 Criticals: **HXC-227** (published credential — OPERATOR ONLY, rotation),
**HXC-243** (our own suites cannot fail — the systemic one, see §4),
**HXC-333** (image without its program — source fixed this session, fleet not).

---

## 3 · Committed this session, and what is still open in it

Everything below is **committed and pushed**. Nothing is at risk on disk.

| commit | what | review state |
|---|---|---|
| `0c26407e` helix_agent | HXC-333/337 Go-image build fix | author-complete, unreviewed |
| `70554466` helix_agent | HXC-334 MCP health check | author r1–r2 done; **independent r3 never returned** |
| `9e4f6313` helix_agent | HXC-244 health verdict | **NO-GO — 1 blocking finding** |
| `109265f8` llms_verifier | HXC-298 shared client-IP package | **NO-GO — 3 findings** |
| `d236bab6` meta | HXC-282 hooks refuse-not-fail-open | r24 done; **r25 never returned** |
| `7d1cba84` `a47b9e98` meta | HXC-340..344 filed | n/a |

**The two NO-GOs, precisely — these are the first things to pick up.**

**HXC-244** (`9e4f6313`): the in-code claim that no other consumer reads the
`status` string is FALSE. Three consumers key on the literal `"healthy"` and are
NOT reconciled:
`tests/integration/models_dev_integration_test.go:159-168` (boots a REAL router,
no build tag, masked locally only by `RequirePostgres` — it runs and fails in a
real integration environment), `challenges/scripts/partitioned_distribution_challenge.sh:167-173`,
`challenges/scripts/runtime_debate_system_challenge.sh:53-54`.
The reviewer's diagnosis is the reusable part: the `testutil` reasoning was
exactly right; the defect was generalising *"this one consumer doesn't read it"*
into *"no consumer reads it."*

**HXC-298** (`109265f8`): three findings, all in **comment prose**, all introduced
by the previous round's own corrections — F1 an unsourced gloss on nginx
"appending" (the doc never defines it; its only other use means the opposite),
F2 a false scope-invariance claim (XSS shifts 33 → 82 tree-wide), F3 a sentence
asserting an ID beats a quote "which revision drift can invalidate" **while
keeping the quote**, which postdates its section by ~10 months.
The code is proven unchanged across every revision: AST-stripped digest
`5af6dc32ffc0af34…`, 164 lines / 3957 bytes, identical at pristine(848),
pre-edit(675) and current(694).

**The next round on HXC-298 is a SPLIT, not a cut.** All ~17 external citations
verify clean; the risk has migrated entirely into the connective prose between
them — a sentence of authored inference sitting next to a verified quote inherits
the quote's authority without its own evidence. Every sentence should either
carry its measurement anchor or be visibly marked as inference, with the citation
corpus lifted into a companion table and every source ref-pinned (`nginx.tmpl` is
cited with no commit or date; measured `main` = `dbb11b92ddb77bee9c35e462479129354c84f939`).
**Do not chase a line target** — rev 19's "~120 cuttable lines" did not survive
re-derivation; only 9 were, because two of the three regions were load-bearing.

---

## 4 · The finding that most changes what a green build means

**HXC-243 is the systemic defect, and this session produced five more sightings
of it.** The shape: *a guard tests a cheap proxy while documenting the expensive
property.* Each survived for months **because it could not fail**.

1. `has_existing_sibling()` tested presence, claimed sync.
2. `nc -z` tested a socket, claimed health — socat's `fork` mode accepts per
   connection, so a TCP connect proves nothing about the server.
3. `grep 'return nil, err'` tested one spelling, claimed all exits converted.
4. `/v1/health` **gathered** provider health, **discarded it**, hardcoded `"healthy"`.
5. `assert_block` tested only the commit exit code — and the hook has six other
   blocking gates, so `rc != 0` was satisfiable by anything.

And the tests certifying (4) were themselves tautologies: two register their own
inline stub handlers and assert a literal written two lines above; a third booted
the real router and asserted `"healthy"` — **encoding the defect as the spec**.

**Treat these as one architectural defect, not five bugs.** §11.4.108's meta-rule:
three such discoveries in a cycle means the verification pipeline is the defect.

---

## 5 · Release readiness — what actually blocks

**Source is fixed; the fleet is not.** Both health fixes this session changed
source only. Verified live: `podman inspect helixagent-mcp-git` still returns
`["CMD","nc","-z","localhost","9000"]`. Activating them requires recreating
containers — an **operator decision**, not taken.

**HXC-333 root cause, proven end-to-end** (do not re-derive):
`docker/mcp/Dockerfile.mcp-go` built `FROM golang:1.25-alpine` (go1.25.12) while
`MCP/submodules/kubernetes-mcp/go.mod` requires `go 1.26.3`. `GOTOOLCHAIN=local`
blocks auto-download, so the build hard-errors; three `2>/dev/null || true`
swallows discarded it; `COPY --from=builder /app/bin/ /app/bin/` then copied an
**empty directory and succeeded**. The entrypoint found no binary, exited 1, and
`restart: unless-stopped` restarted it **291,505 times**.
RED `golang:1.25-alpine` → `go: go.mod requires go >= 1.26.3` rc=1.
CTRL `golang:1.26-alpine` (go1.26.5) → that error gone.
Fixed with a pinned 1.26 base, `set -eux`, and **two** artifact assertions —
builder-side and again on the **final shipped image**, because
`COPY <dir>/ <dir>/` succeeds on an empty source.

Separate and NOT the cause: 26 of 35 `MCP/dockerfiles/Dockerfile.*` carry
`npm install && npm run build 2>/dev/null || true` (9 clean, so the grep
discriminates). Fixing only that would have left the restart loop untouched.

**Blocked on the operator, unchanged:**
- **HXC-227 / HXC-168** — credential rotation.
- **§11.4.185 manual-QA sign-off** for `helix-code-1.2.0-dev-0.0.1`.
- **A real reboot test** — boot *wiring* is proven (target chain, symlinks in
  both `.wants`, linger=yes, ExecStart binaries exist, negative control). That a
  reboot *succeeds* is not.
- **Whether readiness should return 503 when unservable.** The fix deliberately
  kept HTTP 200 because `internal/testutil/infra.go:352` REQUIRES `StatusOK` and
  would otherwise treat a provider-less dev agent as absent. Contract change.
- **Recreating containers** to activate the health fixes.

---

## 6 · The tracker can delete itself — do not run `db-to-md`

`workable-items validate --db docs/workable_items.db` **exits 1** on **56 items**
that have no `doc_segments` row. Running the supported `sync db-to-md` would
**silently delete all 56** from the markdown trackers: no error, no warning.
Filed as **HXC-343**. `diff` reports **542** DB-vs-markdown differences.

So the trackers are stale *by design decision*: stale beats regenerated-with-56-items-missing.
Newly-added items are unaffected — `add` creates their segments correctly.

```bash
./constitution/scripts/workable-items/workable-items validate --db docs/workable_items.db; echo "rc=$?"
# rc=1 with 56 'missing item-segment' lines is the CURRENT EXPECTED state
```

---

## 7 · Corrections carried forward — do not re-inherit the old versions

Every one of these was a confident claim of mine that measurement disproved.

1. **`grep` in the tool shell is shimmed** and skips gitignored paths. I asserted
   the opposite twice. See §1.
2. **A blocked commit leaves its files STAGED.** I staged the HXC-282 hooks
   package, the hook blocked the commit, and my next `git commit` — which I
   believed was evidence-only — swept all 12 hook files plus the DB into a commit
   labelled `docs(qa): captured evidence`. Caught by inspecting the commit,
   fixed by `reset --soft` while unpushed. **Check `git diff --cached --name-only`
   immediately before every commit**, not the tool output.
3. **`rc=$?` after a pipe reports the last element.** My "validate rc=0" was
   `tail`'s exit code; validate has been rc=1 throughout.
4. **`--stat` output is not a file list.** My "7 non-scripts entries" check was
   counting `--stat` formatting lines. Use `--name-only`.
5. **"A sibling stream owns pre-commit" was WRONG.** The sha movement
   `b86464bb → b254271 → f0e7b015 → be4f3bc1` was entirely that stream's own
   three edits, proven against its pre-op backup. I inferred a second author from
   sha movement without checking authorship, and propagated it into briefs.
6. **Direction is the whole decision on a dirty gitlink.** helix_qa showed 20
   modified `tools/opensource/*`. `git status` says only "modified" — all 20 were
   **backward** moves (scrcpy was 42 commits behind its recorded pin). Committing
   blind would have silently downgraded 20 third-party dependencies. Resolved
   with `git submodule update --init`, not a commit.
   ```bash
   git merge-base --is-ancestor "$old" "$new"   # forward = real advance
   ```

---

## 8 · Standing hazards

- **Never force-push** (§11.4.113), in any form, for any reason.
- **Never change host power state** (§11.4.133 / CONST-033).
- **Rootless podman only. No sudo. Never `docker`.**
- **`helixllm-coder` holds a 30B model** — do not stop, restart or disturb it.
- **Never print a matched credential value** — path, line, pattern label only.
- **No real provider calls** — startup discovers live gemini/deepseek/mistral
  credentials. Use `httptest` loopback or a closed port.
- **Verify process/container ownership before attributing or signalling** — this
  host runs other projects' work. Never bare `pkill -f` (it matches its own argv).
- **41 containers are live.** Do not start/stop/restart/recreate.
- **A crashed agent leaves its mutation applied**, and the tree still builds
  clean. After any crash, re-verify pristine shas **before measuring anything**.

**Parked, not lost:** `scripts/lib/testdata/hxc282_pre_r1_mutation_baseline.sh`
was refused by the pre-commit gate (mutation marker, no exemption header,
referenced by nothing — unlike its two properly-headered siblings). Copied to the
session scratchpad and removed from the worktree; **HXC-344** tracks whether to
wire it in or retire it. Adding a header to quiet the gate would have been
fake-passing a correct check.

---

## 9 · What worked, and is worth continuing

- **Independent review catches what tests cannot.** Every self-description defect
  this session was caught by a reviewer comparing claim against artifact; **zero**
  were caught by a test suite. Both NO-GOs came from reviewers re-deriving a
  number the author had asserted.
- **A correction carries the defect at the same rate as original work.** HXC-298
  is five-for-five: every round's fix introduced a new false claim. Re-review the
  correction; never assume it is clean because its author had just understood the
  problem.
- **Three-pole calibration.** pristine (reached+passed) / forced-fail (unreached)
  / mutated (reached+fired). Two poles cannot distinguish *unreached* from
  *reached-and-passed* — a failure log is silent about success.
- **The vacuity oracle.** Force the subject to always-pass, then always-fail.
  Anything green under BOTH is decorative.
- **State the scope inside the claim**, so the obvious command returns what the
  sentence says. Most bad counts here were right in some scope and stated in none.
