# RESUME — HelixCode Session Handoff

**Revision:** 9
**Created:** 2026-07-08
**Last modified:** 2026-07-28T18:03Z (round-2 review corrections, §11.4.131/§12.10)
**Status:** active
**Maintainer:** CLI-agent main work stream
**Authority:** §11.4.131 Session-resumption file — point a fresh agent here. Composes §11.4.127 / §12.10 / §11.4.65 / §11.4.44.

> **Revision 9 (2026-07-28T18:03Z) — corrections from an independent round-2 review of `bf1ce692` + `c2b6945e`.** That review reproduced this session's four-quadrant RED/GREEN matrix cell-for-cell and confirmed `bf1ce692` clean; it also raised three defects in the rev-8 doc edit, ALL corrected here.
> **(i) ATTRIBUTION FIXED — rev-8 and rev-9 findings were spliced INTO the Revision 7 banner below, not given their own.** The "REPAIRED in `bf1ce692`", bisect-skip, and N2-closure text now sitting inside Revision 7's paragraph was written in RESUME rev 8 / CONTINUATION rev 20-21, NOT by the Revision 7 / Revision 19 doc-refresh pass — so Revision 7's own self-description ("this pass touched none of them") does NOT cover that text. Read it as rev-8-or-later. It is left in place rather than surgically un-spliced (moving it risks losing content); this banner is the correction.
> **(ii) TIMESTAMP FIXED — Revision 7's banner stamp was 43 minutes in the FUTURE of its own commit.** `dc9e4caa` committed at `2026-07-28T17:37:47Z` but stamped itself `18:20Z`, which made rev 8 (`17:45Z`) read as "modified before" rev 7. The Revision 7 stamp below is corrected to its true commit time. Forward-fix only — no history rewrite (§11.4.113).
> **(iii) ANCHORS REFRESHED — Revision 7's snapshot numbers are STALE and MUST NOT be quoted.** Live as of this revision: meta-repo HEAD **`683205eb`**, **25** commits ahead of all remotes (Revision 7 says `99ff7d8e`/20 — superseded). `helix_agent` **`5b88aea6`, FULLY PUSHED (0 ahead)** — its 13-commit delta reached a CLEAN GO on review round 3 and is published; the meta pin was bumped to match in `683205eb`. `debate_orchestrator` **`2a9485e`, 2 ahead, still UNPUSHED** — round-2 review returned Ready-to-merge=Yes with one optional Minor, which is being closed before push; its pin is deliberately NOT bumped while unpushed (a pin to an unpushed commit dangles for any clone). `tool_schema` `8cec90b`, pushed. **Re-derive these rather than quoting them** — four subagent streams were committing concurrently as this was written, so the numbers drift by design.
> **(iv) STILL OPEN, carried forward honestly:** an independent review found the replica-RED bluff class STILL LIVE in two sibling files (`internal/server/llm_default_model_regression_test.go:103,127,170` — `oldResolveDefaultModelReplica` reduces to `"" == ""`; and `llm_generate_regression_test.go:187-198`). Conversion is in flight. Note the discipline point: `3fd55a4d`'s claim that its own bluff was a "tracked follow-up" was verified UNBACKED — zero matching items in `docs/workable_items.db` or `docs/Issues.md` (§11.4.148). Prose is not tracking.

> **Revision 7 (2026-07-28T17:37Z) is a CORRECTION revision produced by a dedicated doc-refresh pass** (scope: this file + `docs/CONTINUATION.md` only, dispatched while four OTHER streams actively worked `helix_code/internal/llm/`, `helix_code/applications/`, `submodules/helix_agent/internal/agents/`, and `submodules/debate_orchestrator` — this pass touched none of them, read-only against the rest of the tree). Revision 6's numbers are STALE — the repo kept moving hard after it was written. Independently re-derived this pass (§11.4.6, nothing copied forward):
> **(A) Meta-repo HEAD moved from `c29e1dcc` to `99ff7d8e` — 12 more commits, `main` is now 20 commits ahead of all 4 remotes**, not 8. New commits include a self-inflicted-and-fixed §11.4.108 SOURCE→ARTIFACT gap: `3fd55a4d` recovered an orphaned regression guard whose composite literal referenced a struct field (`respModel`) that did not exist in the committed tree — `internal/server` did NOT compile at `3fd55a4d`/`3c8197cf`/`905a0b0a` — caught by independent review and fixed one commit later in `34e264e1`. `3fd55a4d`'s own commit message additionally DISCLOSES that its `RED_MODE=1` branch is itself a bluff (asserts on a local in-test replica of the pre-fix response construction rather than driving the real handler, so it passes on already-fixed code) — **REPAIRED in `bf1ce692`**: both polarities now drive the REAL handler over one shared fixture, proven by the full four-quadrant matrix (pre-fix RED=PASS defect-reproduced / pre-fix GREEN=FAIL / fixed RED=FAIL / fixed GREEN=PASS), with the pre-fix artifact reconstructed in a throwaway `git worktree` with `submodules/` symlinked in. An independent review also found the "tracked" claim in `3fd55a4d`'s message was UNBACKED — no entry existed in `docs/`, `Issues.md`, or the workable-items DB (§11.4.148); it was fixed rather than back-filled. **Bisecting across `3fd55a4d..905a0b0a` needs `git bisect skip`** — those three commits do not build `internal/server`, so a bisect run there reports a BUILD failure, not a test verdict, and must not be read as the defect's location. `c7484cc7` (scaling-harness signal-to-noise, Trials 3→5, confirmed in source: `MeasureTrialsPerStep = 5`) and `640db264` (aurora_os uint64-underflow fix) both carry pasted verification output in their own commit messages and are treated as independently-clean per the dispatching brief; this pass did not re-run an independent review of them. The newest commit, `99ff7d8e`, is a review artefact in its own right: reviewing `3c8197cf`'s new `*_racefix_test.go` files by READING (this host cannot build Fyne/glfw — no X11 dev headers) found the harmony_os test panics deterministically on first execution (nil `apiClient` dereference in `refreshData()`) and fixed it — proof that the "authored but never executed" GUI test gap (below) can hide real defects, not just theoretical ones.
> **(B) `helix_agent` moved from `a345c551` to `f451d342` — 9 more commits, now 11 commits ahead** (was 2), working tree still dirty (18 porcelain entries, not re-itemised this pass — re-derive). The dispatching brief for this task described a Critical review finding as "remediation in flight" (Dreamer's `memoryMu` not covering `cleanupPhase`'s unlocked `MEMORY.md` read-trim-write); **this pass verified by diff that the fix has actually LANDED**, in `f451d342` — `cleanupPhase` now goes through the same `beginMemoryWrite()`/`end()` helper as `saveMemories()`, with a dedicated `memoryWritersPeak` runtime-signature test proving no concurrent writers. What this pass could NOT verify from git alone: whether `f451d342` and `129094b0` (the 2 commits landed after the reported 9-commit/3-round CLEAN GO at `0367570e`) have themselves been through independent review — no review artefact was found. Treat that as open, not assumed clean.
> **(C) `tool_schema` is now FULLY PUSHED** — `8cec90b`, 0 commits ahead (was 1), clean, confirmed via `ls-remote` on all 4 configured remotes.
> **(D) `debate_orchestrator` is tracked in this document for the first time.** `b6c90e7`, 0 ahead, clean, confirmed pushed via `ls-remote` — but it carries only ONE remote (`origin` → `github.com:HelixDevelopment/DebateOrchestrator.git`), unlike every sibling submodule's 4-remote (github/gitlab/origin/upstream) fan-out. Flagging this for the operator as a decision point, not fixing it unilaterally — no scope in this pass to add remotes to a submodule four other streams may be relying on.
> **(E) The scaling-harness stability gap the dispatching brief described (no `-count>=10` sweep at the shipped `Trials=5`, one run reportedly hung at ~0.3% CPU and killed without diagnosis) could NOT be independently corroborated this pass** — no matching log, process, or docs/qa artefact was found for the "hung at 0.3%" detail specifically. It is carried forward as reported, unconfirmed by this pass's own evidence, and should be re-verified rather than trusted at face value (§11.4.6) — the underlying gap (no `-count>=10` stability run on record) IS independently confirmed: no such sweep log exists under `qa-results/`.
> **(F) Hard-won lesson from this session, worth stating plainly for whoever reads this next: a RED test that asserts against a local replica of the old behaviour, rather than driving the real production code path, is a bluff gate — it can never fail, so it proves nothing.** This surfaced independently at least twice this session in slightly different shapes: `3fd55a4d`'s own disclosure above, and (per the dispatching brief, not independently re-verified by this pass) an equivalent pattern in `debate_orchestrator`'s `protocol_convenience_form_red_test.go`, repaired in that submodule's `b0655d3`. When authoring or reviewing a §11.4.115 RED/GREEN polarity test, confirm the RED branch calls into the actual handler/function under test — not a hand-rolled reconstruction of what it used to do.
> Everything else in Revision 6 below not superseded by (A)-(F) above is unchanged and still holds — the constitution-gate-sweep GREEN result, the G7 resolution, the platform-up state, and the corrections it made to Revision 5 were all reconfirmed live during this pass (see the updated Live-state anchors table) and are not repeated here.

> **Revision 6 (2026-07-28T12:35Z) is a CORRECTION revision from a read-only audit.**
> Revision 5's central claims were ALREADY STALE by the time they were read — the repo was
> (and, as of this revision, still is) being actively advanced by other concurrent
> streams/agents while revision 5 was being written and after. Verified corrections:
> **(A) G7 NOW PASSES — 0 violations, not 13.** Independently re-run
> (`bash scripts/verify_qa_evidence.sh --enforce --since 925169c98945ca0fee1e84dae53ad494e4897832`):
> 28 feature-shipping commits evaluated, **0 violations**, 0 opt-outs. The retrospective
> QA-evidence commits rev 5 called "Class A/B/C, still violating" (including all 3 security
> fixes) are landed and now report `ok`. **The full constitution gate sweep is GREEN**:
> `bash scripts/verify-all-constitution-rules.sh` → 16 gates run, **15 PASS + 1 honest SKIP**
> (G14, docs_chain engine absent, SKIP-OK per §11.4.3) — **zero FAIL**, exit code 0.
> **(B) HEAD IS NOT `0a4eb8d0` AND NOT EVERYTHING IS PUSHED.** Rev 5's "nothing unpushed"
> was true only at `0a4eb8d0`. The main-stream track kept committing after rev 5 was
> written; as of **2026-07-28T12:34:54Z**, HEAD is `c29e1dcc` and **local `main` is 8
> commits ahead of every configured remote** (github/gitlab/origin/upstream — all 0 behind).
> This count is VOLATILE — HEAD moved 3 times (0a4eb8d0→e952f4d1→a5462e35→c29e1dcc) during
> this single audit; re-derive with `git log --oneline origin/main..HEAD` before trusting it.
> **(C) Constitution pin now MATCHES the checkout** (`ce3331a1`, both), not "6 behind" —
> resolved by commit `925aa859`. **(D) The §11.4.157 carrier-lockstep gap is effectively
> closed.** 5 of 6 root carriers (CLAUDE.md/AGENTS.md/QWEN.md/GEMINI.md/CONSTITUTION.md) are
> committed at HEAD reaching §11.4.235 (matching the constitution submodule's own ceiling);
> CRUSH.md is committed at §11.4.234 but its working-tree edit (already present, uncommitted)
> already reaches §11.4.235 — one commit away from full lockstep. **Everything else in rev 5
> not superseded above (Traps, Sweep commands, Corrections 1-5, Binding constraints) is
> unchanged and still holds.**

---

## SHORT variant (one paste, first sentence)

> Read `RESUME.md` then `docs/CONTINUATION.md`, run `git fetch --all --prune --tags`, and continue on `main` (Track 1 — §11.4.182 TRUNK RULE: trunk work is ALWAYS `(T1/main - <alias>)`). HEAD is `99ff7d8e` as of 2026-07-28T17:37Z (re-derived this pass — this repo has moved fast every time it has been checked; re-derive again with `git rev-parse HEAD` before trusting this number). **`main` is 20 commits ahead** of every configured remote (github/gitlab/origin/upstream, all 0 behind) — re-derive with `git log --oneline origin/main..HEAD`. `helix_agent` is **11 commits ahead** of its 4 remotes (dirty working tree, 18 porcelain entries); `tool_schema` is now **fully pushed** (0 ahead, was 1); `debate_orchestrator` is **fully pushed** but has only ONE remote (github), unlike its 4-remote siblings — flag for the operator, don't fix unilaterally. The working tree carries **73 porcelain entries** (46 untracked + 27 modified — mostly in-flight `docs/qa/<run-id>/` evidence dirs from a concurrent stream's G7 work; account for every one before `git add`, NEVER `git add -A`, §11.4.84). **The platform is UP and systemd-ENABLED** — 5 units active, ports 7061/8081/8100/8443 bound (reconfirmed this pass, unchanged); do NOT boot infra blindly, check first. **What blocks the tag**: (1) the §11.4.40 full Go-test retest on a QUIET host has still not concluded (the durable `helix_code_inner` sweep log is unchanged since 14:41 — hours before the newest commits landed; a fresh serial run is needed, not a re-read of that log); (2) pushing the now-20-ahead `main` (+ helix_agent's 11-ahead) before choosing a tag point; (3) §11.4.185 manual QA-team confirmation, for which **no record exists in this repo**; (4) a disclosed-but-unrepaired RED bluff in `3fd55a4d`'s F1 guard (asserts on a local replica, cannot fail) still needs its RED branch rewritten to drive the real handler; (5) three `main_racefix_test.go` files (desktop/harmony_os/aurora_os) remain authored-but-never-executed on this host (no X11 dev headers) — one was proven to hide a real bug (fixed in `99ff7d8e`) by reading alone, so treat the other two's coverage as unconfirmed. Force-push is absolutely forbidden (§11.4.113).

---

## FULL variant (paste-ready block)

> You are resuming the HelixCode test-suite remediation cycle on `main` = **Track 1** (§11.4.182 TRUNK RULE, operator mandate 2026-07-28: trunk work labels as `(T1/main - <alias>)`, never `(T?/…)`). The prior sessions landed real fixes with captured RED→GREEN evidence, and also made several attribution errors that are corrected below (and in Revision 6/7's blockquotes above) — read the corrections before trusting any earlier summary.
>
> **Read first:** `RESUME.md` (this file), `docs/CONTINUATION.md`. Then `git fetch --all --prune --tags` and `git log --oneline HEAD..@{u}` (expect empty — the risk is the reverse direction, `@{u}..HEAD`, see below).
>
> **State (as of 2026-07-28T17:37Z, doc-refresh pass — RE-DERIVE, do not trust, this repo moves fast):** meta-repo `main` @ `99ff7d8e`, **20 commits ahead** of all 4 configured remotes (github/gitlab/origin/upstream — `git log --oneline origin/main..HEAD`); helix_agent @ `f451d342` (**11 commits ahead** of all 4 of its remotes; working tree DIRTY, 18 porcelain entries, not re-itemised this pass); tool_schema @ `8cec90b` (**fully pushed, 0 ahead**, clean — was 1 ahead in rev 6); debate_orchestrator @ `b6c90e7` (**fully pushed, 0 ahead**, clean, but only ONE remote configured — github — unlike siblings' 4-remote fan-out; flag for operator, newly tracked in this document); helix_qa @ `88ef0579` and challenges @ `072724af` (both clean, 0 ahead/0 behind on all remotes, reconfirmed unchanged); constitution checkout @ `ce3331a1` — pin still MATCHES the checkout (unchanged, reconfirmed). The meta-repo working tree carries **73 porcelain entries** (46 untracked + 27 modified; re-derive with `git status --porcelain`) — overwhelmingly new `docs/qa/<run-id>/` evidence directories from a concurrent stream's ongoing G7 work, plus a handful of source edits from other active streams (per this task's scope fence: `helix_code/internal/llm/`, `helix_code/applications/`, and other in-flight edits belong to OTHER streams — do not stage them).
>
> **Terminal goal:** tag `helix-code-1.2.0-dev-0.0.1` on the main repo AND every owned submodule carrying changes, identical `helix-code-` prefix (§11.4.151), fast-forward-only to all upstreams (§11.4.113) — **conditional on the sweep being green**, reaffirmed by the operator: **tag ONLY when genuinely green**. The constitution-gate-sweep component was GREEN as of rev 6's audit (16 gates, 15 PASS + 1 honest SKIP, 0 FAIL) and was not re-run this pass (out of scope — doc-only). **The tag is still NOT authorised to proceed.** Beyond rev 6's blockers (a full Go-test retest of `helix_code` + `helix_agent` on a QUIET host — the durable `helix_code_inner` sweep log is unchanged since 14:41, hours before the newest commits landed, so it does NOT cover current HEAD; §11.4.185 manual QA-team confirmation, still no record in this repository; the unpushed-commit counts, now 20/11/0/0 for main/helix_agent/tool_schema/debate_orchestrator), this pass surfaces two ADDITIONAL open items: a disclosed RED bluff in `3fd55a4d` (the F1 guard's `RED_MODE=1` branch asserts on a local replica and cannot fail — tracked, not yet repaired) and unconfirmed independent review of helix_agent's newest 2 commits (`129094b0`, `f451d342`) landed after its reported 9-commit/3-round CLEAN GO. Also still flagged for the operator: the target version `1.2.0-dev-0.0.1` is a MINOR bump past the existing `1.1.0-dev-0.0.1/.2/.3` sequence — no CHANGELOG entry or other in-repo rationale for the minor bump (vs. continuing `1.1.0-dev-0.0.4`) was found; confirm this is intended.
>
> **Anti-bluff is binding (§11.4):** every closure needs pasted runtime output; every gate touched needs a paired §1.1 mutation; no assertion may be weakened and no bare `t.Skip()` added to force green. **This session's hard-won lesson: a RED test asserting against a local replica of old behaviour rather than driving the real code path is a bluff gate that can never fail** — check this specifically when reviewing any §11.4.115 polarity test.

---

## Live-state anchors

All values below verified against the live system on **2026-07-28T12:34:54Z** by a read-only audit. Re-derive before trusting (§11.4.6) — this table has been wrong before, AND the repo was actively moving DURING this very audit (HEAD advanced 3 times in ~10 minutes of observation).

| Key | Value | How verified |
|-----|-------|--------------|
| **Meta-repo HEAD** | ⚠️ **rev 6 row superseded — re-derived rev 7 (2026-07-28T17:37Z, doc-refresh pass):** `99ff7d8e` (branch `main`) — **20 commits ahead** of ALL 4 configured remotes (github/gitlab/origin/upstream), 0 behind (was `c29e1dcc`/8-ahead at rev 6, 12 more commits landed since — see the Revision 7 blockquote at the top of this file for what they contain, incl. a self-inflicted-and-fixed compile break at `3fd55a4d`→`34e264e1`). VOLATILE — this repo has moved every single time it has been checked this session; re-derive again | `git rev-parse HEAD`; `git fetch --all --prune --tags` then `git rev-list --left-right --count main...refs/remotes/<r>/main` per remote |
| **Working tree** | ⚠️ **superseded — re-derived rev 7:** **73 porcelain entries** (46 untracked + 27 modified; was ~68 at rev 6) — overwhelmingly new `docs/qa/<run-id>/` untracked evidence dirs from a concurrent stream's ongoing G7 work, plus source edits belonging to OTHER active streams per this task's own scope fence — VOLATILE, re-derive, never quote back, and never `git add -A` | `git status --porcelain` |
| **helix_agent** | ⚠️ **superseded — re-derived rev 7:** `f451d342` — **11 commits ahead** of all 4 of its remotes (was `a345c551`/2-ahead at rev 6; 9 more commits landed, including the reported CLEAN-GO range ending `0367570e` plus 2 further commits `129094b0`/`f451d342` whose independent review this pass could NOT confirm from any git-visible artefact); working tree **DIRTY** (18 porcelain entries this pass, not re-itemised — re-derive fresh rather than trusting rev 6's "14 modified + 4 untracked" breakdown). **CONFIRMED THIS PASS by reading the diff:** the Critical review finding described elsewhere as "remediation in flight" (Dreamer `memoryMu` not covering `cleanupPhase`'s unlocked `MEMORY.md` write) has actually LANDED, in `f451d342` — `cleanupPhase` now goes through the same `beginMemoryWrite()`/`end()` helper as `saveMemories()` | `git -C submodules/helix_agent status/log/fetch`; `git -C submodules/helix_agent show --stat f451d342` |
| **helix_qa** | `88ef0579` (gitlink matches worktree); clean; 0 ahead/0 behind — reconfirmed rev 7, unchanged | `git -C submodules/helix_qa status/fetch` |
| **challenges** | `072724af` (gitlink matches worktree); clean; 0 ahead/0 behind — reconfirmed rev 7, unchanged | `git -C submodules/challenges status/fetch` |
| **tool_schema** | ⚠️ **superseded — re-derived rev 7:** `8cec90b` — **fully pushed, 0 commits ahead** (was 1-ahead at rev 6) of all 4 remotes (vasic-digital), confirmed via `ls-remote`; clean working tree | `git -C submodules/tool_schema status/log/fetch`; `git -C submodules/tool_schema ls-remote --exit-code <remote> refs/heads/main` |
| **debate_orchestrator** | **NEW this revision — not tracked in rev 6 at all.** `b6c90e7` — fully pushed, 0 ahead, clean, confirmed via `ls-remote`. Carries only ONE configured remote (`origin` → `github.com:HelixDevelopment/DebateOrchestrator.git`), unlike every sibling submodule's 4-remote (github/gitlab/origin/upstream) fan-out — flagged for the operator as a decision point, not changed unilaterally in this pass (out of scope: this task's scope fence explicitly excludes `submodules/debate_orchestrator`) | `git -C submodules/debate_orchestrator status/log/fetch`; `git -C submodules/debate_orchestrator remote -v` |
| **constitution — checked out** | `ce3331a1` (`v1.0.0-18-gce3331a`); in sync with all **8** remote-tracking refs (ahead=0 behind=0, fresh fetch this audit) | `git -C constitution rev-parse HEAD`; per-remote `rev-list --count` |
| **constitution — meta-repo pin** | `ce3331a1` — **MATCHES the checkout exactly. RESOLVED** (was `731bf1d3`, 6 behind, in rev 5 — fixed by commit `925aa859`). The only remaining dirt under `constitution` is its OWN nested submodule `submodules/session_orchestrator` (checked out at `6961c998`, detached, but constitution's own index pins `78b77bb8` — a minor, unrelated nested-submodule drift, not a helix_code release blocker) | `git ls-tree HEAD constitution`; `git -C constitution rev-parse HEAD` (identical SHAs) |
| **Constitution ceiling vs carriers** | constitution defines **§11.4.235**; 5 of 6 root carriers (CLAUDE.md, AGENTS.md, QWEN.md, GEMINI.md, CONSTITUTION.md) are COMMITTED at HEAD reaching **§11.4.235** already — RESOLVED for those 5. CRUSH.md is committed at §11.4.234 but its **working-tree edit (present, uncommitted)** already reaches §11.4.235 — one commit away from full lockstep, not an open gap requiring new work | `git show HEAD:<file> \| grep -oE '11\.4\.[0-9]+' \| sort -t. -k3 -n \| tail -1` per carrier, vs. the same on the working-tree copy |
| **Release prefix** | `helix-code` (from `HELIX_RELEASE_PREFIX` in `.env`, mode 0600 — confirmed unchanged) | `grep '^HELIX_RELEASE_PREFIX' .env` |
| **Latest existing tag** | `helix-code-1.1.0-dev-0.0.3` → stated target `helix-code-1.2.0-dev-0.0.1` is prefix-consistent but is a MINOR bump past the existing `-1.1.0-dev-0.0.{1,2,3}` sequence; no CHANGELOG/doc rationale found for skipping `-1.1.0-dev-0.0.4` — flag for explicit operator confirmation | `git tag -l 'helix-code-*' \| sort -V`; `grep -c '1.2.0-dev-0.0.1' CHANGELOG.md` (0 hits) |
| **Platform** | **UP and ENABLED** — unchanged from rev 5, re-verified this audit (`helixagent`/`helixcode-server`/`helixllm-gateway`/`helixcode-infra`/`helixllm-coder` enabled+active; `helix.target` enabled+inactive; ports 7061/8081/8100/8443 bound) | `systemctl --user is-enabled/is-active`, `ss -ltnp` |
| **Constitution gates** | 16 run, **15 PASS + 1 honest SKIP (G14) — ZERO FAIL** as of rev 6's audit (2026-07-28T12:34Z). **NOT re-run this pass** (doc-refresh scope only, per this task's brief) — treat as last-known-good, not reconfirmed at current HEAD `99ff7d8e` | rev 6: `bash scripts/verify-all-constitution-rules.sh` (exit 0), `bash scripts/verify_qa_evidence.sh --enforce --since 925169c98945ca0fee1e84dae53ad494e4897832` |
| **Sweep logs (current)** | ⚠️ **Still true as of rev 7 — re-checked, unchanged.** `qa-results/full_retest/` newest entries as of this pass are `all_captures_20260728T104042Z.log` (15:41) and the same `helix_code_inner_20260728T093852Z.log` (mtime **still 14:41**) noted at rev 6 — **hours before the newest commits (`3fd55a4d` through `99ff7d8e`, timestamped 21:59-22:18) landed.** This durable log does NOT cover current HEAD; a fresh serial `go test` run is required, not a re-read of it. Whether PID 705363's wait-wrapper is still polling was not re-checked this pass (ephemeral, unlikely to still be the same PID hours later) | `ls -lat qa-results/full_retest/` |
| **Manual QA-team confirmation (§11.4.185)** | **NO RECORD FOUND** anywhere in this repo (`docs/qa/`, RESUME.md, CONTINUATION.md) of an actual manual QA-team sign-off for this release cycle. UNKNOWN whether it happened out-of-band; settle by asking the operator or checking any external QA tracker | `grep -rliE "manual QA.{0,30}(confirm\|sign.?off\|approv)" docs/` (no qualifying hit) |
| **Pre-gofmt tree backup** | `<scratchpad>/helix_agent_pre_gofmt.patch` (3322 lines) — **path not re-verified this revision either** | — |

---

## Operator decisions recorded 2026-07-28 (binding, newest first)

1. **`helixcode-infra` becomes the SOLE infra owner** — resolves the competing-orchestrator blocker (task #9). Set `HELIX_AUTOBOOT_INFRA=false` so `cmd/server` does not race a second boot (the flag is honoured at `helix_code/internal/infraboot/infraboot.go:118-125`), and `helix_code/config/replica-8081.yaml` is corrected (uncommitted, +102/-5).
2. **G7 is cleared by REAL retrospective QA runs — the 3 security commits first.** Not a baseline bump. See the G7 section.
3. **`helixcode-server` secret env names are reconciled.**
4. **Tag ONLY when genuinely green.** Reaffirms the conditional authorisation; the sweep is not green, so do not tag.

**Operator mandate 2026-07-28 — Track identity.** Work on `main` is **ALWAYS Track 1**; the label is `(T1/main - <alias>)`, never `(T?/main - …)`, regardless of checkout path. Landed in `constitution/Constitution.md` §11.4.182 as the TRUNK RULE and pushed; the constitution checkout is in sync with all **8** upstreams (ahead=0/behind=0 on every remote-tracking ref, as of last fetch — no fetch was performed while writing this revision).

## Operator decisions recorded 2026-07-27 (binding)

1. **Ports** — do NOT hand-assign. Use the **containers submodule's dynamic-port + service-discovery** capability; extend that submodule for anything missing (§11.4.74 extend-don't-reimplement) and cover the extension with all supported test types. All systems/services/infrastructure choose an available port at runtime, bind it, and expose it through discovery; services discover each other. This supersedes the earlier "which service moves off 8100" framing.
2. **Constitution pin** — advance to the checked-out HEAD **in this release**.
3. **Moving baselines** — fix the tests so they stop writing to tracked files (root-cause fix, not gitignore).
4. **Compose coupling** — **relocate `mcp_servers`** under helix_agent so no upward path traversal is needed.

---

## Landed this session, with captured evidence

- **gofmt across our Go code**: 507 → 0 unformatted. Vendored/archival trees deliberately EXCLUDED and reverted (see Correction 2). 431 format-only files.
- **Production nil-deref crash FIXED** — `internal/llm/llamacpp_provider.go` `GenerateStream`. `req, _ := http.NewRequestWithContext(...)` discarded the error, so the next line dereferenced a nil `*http.Request`. Fault address `0x38` == `offsetof(http.Request.Header)`, confirming it exactly. Guarded by `internal/llm/llamacpp_stream_malformed_url_test.go` (§11.4.115 `RED_MODE` polarity test). **Proven:** `RED_MODE=1` PASS pre-fix / FAIL post-fix; `RED_MODE=0` FAIL pre-fix / PASS post-fix; whole package `ok` 69.789s.
- **G8 gate reconciled** (§11.4.120) — was a FALSE POSITIVE, see Correction 3. Now PASS, with a 7-assertion paired mutation.
- **G13 now 82/82** — four honest Sources-verified footers, zero invented URLs, two recording explicit non-verification gaps.

---

## Platform state — VERIFIED 2026-07-28 (supersedes the 2026-07-27 "shutdown" claim)

**THE PLATFORM IS UP AND systemd-ENABLED.** Revision 4 of this file claimed *"PLATFORM IS DOWN AND DISABLED … nothing auto-starts"*. That is **FALSE as of 2026-07-28**. Live readings:

| Unit | enabled | active |
|---|---|---|
| `helixagent.service` | enabled | **active (running)** — HelixAgent on :7061 |
| `helixcode-server.service` | enabled | **active (running)** — HelixCode server on :8081 |
| `helixllm-gateway.service` | enabled | **active (running)** — LLM router |
| `helixcode-infra.service` | enabled | **active (exited)** — oneshot, infra up |
| `helixllm-coder.service` | enabled | **active (exited)** — oneshot |
| `helix.target` | enabled | inactive |
| `helix-{pg,redis}-data-volume.service` | generated | inactive |

Bound ports (`ss -ltnp`): **7061** (`helixagent`), **8081** (`helixcode`), **8100** (`llm-verifier`), **8443** (`helixllm`), plus `:8080` with no owning user process visible.

**Consequence for a resuming agent:** do NOT blindly boot infra — it is already up, and a second orchestrator racing the running one is exactly the task-#9 blocker below. Check `systemctl --user is-active` and `ss -ltnp` FIRST. Equally, do not assume the ports named in the older record are free: `8100` is held by `llm-verifier`, not by HelixAgent.

Untouched by design (other projects on the shared host, §11.4.174): `helixterm-*` + `77c45f5c104d-infra`, `penpot-*`, `helix_sonarqube*`, and **`atmosphere-aosp-build`** — do not assume the host is idle, and never kill a process you have not proven is ours (§11.4.174).

## Session 2026-07-27 evening — additions and corrections

### Landed — PORT_CONFLICT ownership fix (RED→GREEN captured)

`tests/precondition/containers_boot_test.go` + new `tests/precondition/port_conflict_ownership_test.go`. The gate held **two opposite §11.4.201 defects**:

- **Fail-closed:** probed `{port, port+10000}` and declared `PORT_CONFLICT` if both answered, with zero ownership check. Our stack publishes postgres on `15432 = 5432+10000`, so *any* foreign postgres on the standard port (here `helixterm`) produced a false failure.
- **Fail-open:** `findContainerInstances` ran `docker compose ps --filter name=…` (because `detectContainerRuntime` returns args `["compose"]`) — not a container-listing command. It returned nothing, so the duplicate-instance check **silently never fired**.

Replaced with runtime-driven ownership: `listRunningContainers` → `ourServiceInstances` (filters the `helixagent-` prefix) → `portConflict`. Evidence: `RED_MODE=1` reproduces (*"legacy blind probe flags a foreign listener as our own duplicate"*), `RED_MODE=0` GREEN, 3/3 both polarities, `ok dev.helix.agent/tests/precondition 0.005s`. The real-duplicate case is still caught (§11.4.120 — not weakened to a tautology).

### Correction 4 — gate status in the 1334Z log is STALE

Sweep `qa-results/full_retest/verify_rules_20260727T184002Z.log` (16 gates, 1 failure): **G1 PASS, G8 PASS, G11 PASS, G12 PASS, G13 PASS (82/82 footered)**, G15/G16 PASS, **G14 SKIP** (docs_chain engine absent — SKIP-OK). Only **G7** fails — at **18** violations in that log, and at **13** in the fresher `/tmp/g7-qa.out` (2026-07-28 14:07). Re-read the log, and re-run the sweep to regenerate the volatile detail file, before acting on any earlier gate claim.

### Correction 5 — the discovery "failures" were never code defects

`internal/discovery` `TestAllocatePort_RangeExhausted_WithEphemeral` / `TestConcurrentAllocations` / `TestConcurrentReleases` pass on a quiet host: `ok dev.helix.code/internal/discovery 0.013s`. The 1332Z run launched the helix_code (`133220Z`) and helix_agent (`133218Z`) sweeps **2 seconds apart**; helix_agent's stress/chaos saturated the ephemeral range `32768-60999`, so the tests' preferred ports `50000-50049` were all unavailable (forcing fallback into the 19-port `api` range → exactly the 16 unique ports logged) and `net.Listen(":0")` returned `EADDRINUSE`. Independent confirmation of Correction 1.

### NEW blocker — competing container orchestrators (task #9, architectural)

`helixagent.service`'s container adapter recreates `helixagent-postgres` from **`docker-compose.yml`, where postgres has NO `ports:` section** — so `15432` is never published. Manual `docker-compose.test.yml` runs DO publish it. With both in play postgres crash-loops: `could not bind IPv4 address "0.0.0.0": Address in use` (host `5432` is held by the unrelated `helixterm` stack). **While the service runs, the precondition gate cannot see 15432.** Compounding it, `scripts/ensure-infrastructure.sh` health-checks `${DB_PORT:-15432}` / `${REDIS_PORT:-16379}` but boots the default compose — it cannot satisfy its own health check (task #7).

**RESOLVED by operator decision, 2026-07-28 — no longer awaiting a call.** `helixcode-infra` becomes the **sole infra owner**: set `HELIX_AUTOBOOT_INFRA=false` so `cmd/server` does not race a second boot (honoured at `helix_code/internal/infraboot/infraboot.go:118-125` — returns `Skipped: true, Reason: "disabled via HELIX_AUTOBOOT_INFRA"`), and `helix_code/config/replica-8081.yaml` is corrected (edit present in the working tree, +102/-5, uncommitted). Remaining work is to land and verify that configuration, not to decide it.

---

## Determinism run 2026-07-28 — the 5 "failing" packages were CONTENTION, not defects

`qa-results/full_retest/determinism_20260728T093338Z.log` — each package run **3× on a quiet host**, every run `exit=0`:

| Package | Runs | Result |
|---|---|---|
| `internal/tools/shell` | 3/3 | ok (3.577s / 3.561s / 3.563s) |
| `internal/verifier` | 3/3 | ok (4.305s / 4.307s / 4.307s) |
| `internal/worker` | 3/3 | ok (43.419s / 43.385s / 43.384s) |
| `tests/scaling` | 3/3 | ok (5.677s / 5.700s / 5.636s) |
| `tests/stresschaos` | 3/3 | ok (0.936s / 0.938s / 0.944s) |

**15/15 PASS.** These are **contention artefacts, not code defects** — independent confirmation of Correction 1 (§11.4.119). Do not open defect items against them; do re-run serially before believing any future red from this set.

## Sweep results — status by stream

| Stream | Verdict | Detail | As of |
|---|---|---|---|
| helix_qa | GREEN (not re-verified this audit) | 142 ok / 0 FAIL / 23 no-test-files; submodule itself clean + in sync on all remotes | 2026-07-27 |
| governance cascade | GREEN (not re-run this audit) | 277 PASS / 0 FAIL; 227 anchors discovered; carriers NOW at §11.4.235 (5/6 committed, CRUSH.md pending commit — see Live-state anchors), superseding "all 6 to §11.4.234" | 2026-07-27 |
| constitution rules | **GREEN — audit re-run, exit 0** | 16 gates, **15 PASS + 1 honest SKIP (G14), ZERO FAIL.** G7 independently re-verified 0 violations (28 commits evaluated). SUPERSEDES the 2026-07-27 23:40 log below | **2026-07-28T12:34Z (this audit)** |
| helix_code inner | UNKNOWN / IN-FLIGHT — not concluded by the owning stream as of this audit | Newest log unchanged since 14:41 (`helix_code_inner_20260728T093852Z.log`); a companion wait-wrapper (PID 705363) has been polling since 14:21 without concluding (see Live-state anchors — likely a self-match pgrep bug, not a still-running test). **This audit does NOT compute or claim a pass/fail tally for this stream** — that determination belongs to the stream that dispatched it | 2026-07-28, not concluded |
| helix_agent | UNKNOWN / AMBIENT-SUSPECT — not a quiet-host run | A `go test -count=1 -timeout=900s ./...` run (PID 2226428, 17:23→~17:39) against a DIRTY working tree completed with 360 ok / 18 FAIL packages, including `tests/integration` hitting the exact 900s timeout wall. The host was NOT quiet during this run (this audit's own fetches + other concurrent activity were in progress). Per this file's own "Correction 1" precedent, this pattern (timeout-exact-hit + assorted package FAILs on a contended host) is ambient-suspect, NOT a confirmed defect list — a serial quiet-host re-run is required before treating any of these 18 as real | 2026-07-28T17:23-17:39, contended host |

**Net effect of this audit: the constitution-gate-sweep component of "the full sweep" is now fully GREEN (G7 included). What remains open is the Go-test component (§11.4.40) on a genuinely quiet host, which neither this audit nor the two concurrently-running streams had concluded as of 2026-07-28T12:34:54Z.**

---

## CORRECTIONS — read these before trusting earlier summaries

**Correction 1 — three concurrent sweeps contaminated the results (§11.4.119).** The prior session ran three full `go test ./...` sweeps simultaneously on one host. Captured proof of contamination: 112× `cannot assign requested address` and 8× `bind: address already in use`, with sibling sweep PIDs verified by cwd/argv. Test sweeps that bind ports are NOT parallelisable — partition by RESOURCE, not by repo. **Several reported failures are ambient-suspect, not proven code defects.** A serial, host-quiet re-run is REQUIRED before any of them are called real.

**Correction 2 — `docs/research/go-elder-plinius-v3` is NOT vendored.** It has no nested `.git` and no LICENSE; the nested repos under `docs/research/` belong to SIBLING directories (`Gandalf-Solutions`, `CL4R1T4S`, `LEAKHUB`, `AutoRedTeam`, `CLAUDE-CODE-SYSTEM-PROMPT`). Reverting gofmt there was still CORRECT, but for a different reason: `go list ./...` returns **0** packages from it — it is archival research intake outside the root module graph that no build/vet/test gate can validate.

**Correction 3 — G8 was a false-positive gate, not a documentation defect.** Both items already carried complete, truthful `Obsolete-Details`. The gate asserted `Triple-check:`; canonical (`constitution/Constitution.md:7687`) and the generator (`obsolete.go:187`) both say `Triple-check evidence:`. That markdown is machine-generated from the SQLite SSoT — hand-editing it would have broken **G11** (byte-identical md→db→md round-trip) and been reverted at next regeneration.

---

## G7 (§11.4.83) — RESOLVED, 0 violations (audit re-verification, 2026-07-28T12:34Z)

**G7 now PASSES.** Independently re-run this audit: `bash scripts/verify_qa_evidence.sh --enforce --since 925169c98945ca0fee1e84dae53ad494e4897832` → **28 feature-shipping commits evaluated, 0 opt-outs, 0 violations**, `RESULT: PASS`. This supersedes the entire section below, which is PRESERVED as the historical record of how the 13-violation figure was reached and cleared — do not act on it as current state; every commit it lists as "still violating" (including all 3 security fixes `4727a9d0`/`9c876819`/`2ff55c31`) now has `docs/qa/<run-id>/EVIDENCE.md` on disk and reports `ok`.

### Historical section (superseded — preserved for provenance)

Prior text described the situation as **13** violations, not 24 (Class C was FIXED at that point; Class A/B were not yet):

**Two stale figures were in circulation and both are superseded.** The "24 violations" three-way split below was the original analysis; the 2026-07-27 sweep log records **18**; the freshest gate detail output records **13**. Provenance, so you can re-derive rather than trust:

| Source | Count | Timestamp |
|---|---|---|
| Original three-way analysis (6+3+15) | 24 | 2026-07-27 13:34Z log |
| `qa-results/full_retest/verify_rules_20260727T184002Z.log` RESULT line | 18 | 2026-07-27 23:40 |
| `/tmp/g7-qa.out` `VIOL` lines — **freshest** | **13** | 2026-07-28 14:07 |

⚠️ `/tmp/g7-qa.out` is the gate's volatile detail file, **regenerated on every run and not durable**. Re-run `./scripts/verify-all-constitution-rules.sh` to regenerate it before relying on the 13.

- **Class C (6) — FIXED, confirmed.** These were FALSE violations (evidence existed; the gate could not match it — it required the COMMIT SUBJECT to contain the evidence directory basename, but run-ids are timestamped slugs, and flat `docs/qa/*.md` files were invisible to a directory-only enumeration). **All six now report `ok`**: `3e67fa1a`, `54a76c3c`, `ca76e14b`, `7210f373`, `02e9e505`, `1254e0a6` — verified individually in `/tmp/g7-qa.out`. This closes in-flight item #2 below.
- **Class B (3) — still violating**, all genuine non-features tripping the heuristic: `d6c05f76` (README only), `f9dcf6a6` (doc-comment only), `b058c7c2` (`const`→`var` test-robustness, prod default unchanged).
- **Class A — 10 remain** (down from 15; `fbfffd7d`, `0e3bb747`, `6efadd15`, `225cdf77`, `aa6b20b4` now report `ok` because their `EVIDENCE.md` cites the commit SHA). Still violating, **including all 3 security fixes**: `4727a9d0` (CORS wildcard+credentials), `9c876819` (CSWSH `/ws` auth), `2ff55c31` (wire-facade 401). Full remaining list: `eb233785 f8c38181 67c9a9bc f9dcf6a6 4727a9d0 9c876819 d6c05f76 2ff55c31 a21ad7ca 51c058b1 66f9c21e c9bad26a b058c7c2`.

**HARD CONSTRAINT:** the `[no-qa-evidence]` opt-out lives in the COMMIT MESSAGE and every violating commit is already pushed. Retroactive application requires history rewrite — **forbidden absolutely** (§11.4.113 / CONST-043). The opt-out is available only going forward.

**Class A is DECIDED (operator, 2026-07-28):** clear G7 by **real retrospective QA runs**, **the 3 security commits first** — not by a documented baseline bump. The evidence must come from genuine runs performed now and committed under `docs/qa/<run-id>/`. Class C is already fixed, so the debt is no longer overstated.

---

## Open defects — confirmed, NOT yet fixed

**RESOLVED since this section was written (audit 2026-07-28T12:34Z, verify before re-opening):**
- ~~§11.4.157 carrier lockstep gap~~ — 5/6 carriers already reach §11.4.235 at HEAD; CRUSH.md's uncommitted working-tree edit already reaches it too. See "Live-state anchors".
- ~~Constitution pin 6 commits behind~~ — now matches the checkout exactly (`ce3331a1`). See "Live-state anchors".
- ~~G7 13 violations~~ — 0 violations, see the G7 section above.

**Still open / unverified this audit:**

- **IPv6 bracket-unawareness — TWO independent bugs, one class, neither a regression** (no shared helper; both non-bracket-aware since creation):
  - `helix_code/internal/llm/response_err_round53_test.go:869` `mustSplitHostPort` re-joins the output of `net.SplitHostPort` (which strips IPv6 brackets by design) with plain concatenation → `http://::1:PORT`.
  - `helix_agent/internal/vectordb/qdrant/client_mock_test.go:31-35` uses `strings.Split(url, ":")` on a bracketed authority → `host == "["`, and a discarded `Sscanf` error silently defaults the port to 80 → `http://[:80`.
  - Trigger for both: `httptest.NewServer`'s IPv6 fallback when IPv4 loopback bind fails. **These tests are environment-flaky by construction.**
- **i18n key leak — LATENT, not live.** `tool_schema` ships a consumer-injected translator seam; **nothing anywhere calls `toolschema.SetTranslator`**, so `activeTr` stays `NoopTranslator` and `tr()` echoes the key. Bundle entry EXISTS and is correct (`active.en.yaml:50` → `"Create git commit"`). No production path currently reaches it, so no user sees it — but it arms on first use, and all 36 `tr()` sites include user-visible `ToolResult.Error` payloads. **Do NOT "fix" the test to expect the raw key** — that cements the unwired state and disarms the only cross-module check.
- **HXC-131 evidence path is dead** — `docs/qa/followup_fixes_20260712T085616Z/HXC131_evidence.md` never existed (that commit added only `HXC133_evidence.md`). Genuine evidence IS at tracked `scratch/discovery/fixes/HXC131_evidence.md`. Correct repair is a DB mutation on the SSoT (`workable-items obsolete-details --evidence …` + re-render), not a doc edit.
- **`sdk/web/node_modules/flatted/golang/pkg/flatted/flatted.go`** — genuinely third-party vendored code sitting INSIDE the gofmt gate. Clean today; an npm bump reintroduces the tension.
- **`go vet ./...` covers only the root module** — 36 nested modules (ours included) are never vetted. Real coverage hole, own item.
- ~~§11.4.157 carrier lockstep gap~~ / ~~Constitution pin 6 behind~~ — **both RESOLVED**, see the "RESOLVED since this section was written" note above this list. (Historical detail preserved: the gap was `constitution/Constitution.md` defining §11.4.235 while carriers topped out at §11.4.234, compounded by the pin `731bf1d3` predating checkout `32d75788`. Both are fixed at current HEAD.)

**NEW this revision (rev 7, 2026-07-28T17:37Z — found while re-deriving live state, not fixed in this pass, doc-only scope):**

- **Disclosed RED bluff, unrepaired — `3fd55a4d`'s F1 regression guard.** `llm_generate_native_model_regression_test.go`'s `RED_MODE=1` branch replicates the pre-fix response construction locally and asserts on that replica, so it PASSES even on the already-fixed artifact — a RED that cannot fail is evidence-shaped, not evidence (§11.4.1/§11.4.115). The commit message discloses this itself and tracks a follow-up to rewrite the RED branch to drive the real handler against pre-fix behaviour (the same repair pattern applied to `debate_orchestrator`'s `protocol_convenience_form_red_test.go` in that submodule's `b0655d3`, per the dispatching brief — not independently re-verified by this pass). The `RED_MODE=0` GREEN half is genuine and load-bearing; only the RED half is a bluff.
- **Three `main_racefix_test.go` files remain authored-but-never-executed on this host** (`helix_code/applications/{desktop,harmony_os,aurora_os}/main_racefix_test.go`) — this host has no X11/GL dev headers so `go build` (without `-tags nogui`) fails for all three packages identically, and the `nogui` build tag excludes these files entirely (confirmed: they carry `//go:build !nogui`). **This gap is proven to hide real defects, not merely theoretical risk** — the harmony_os file was found, by READING alone, to panic deterministically on first execution (a nil-pointer dereference in `refreshData()` when `apiClient` is nil), and was fixed in `99ff7d8e` — still never actually run. The desktop and aurora_os files were independently traced (not executed) and judged safe under both `RED_MODE` settings; that judgement is unconfirmed by execution. Closing this gap needs either an X11-capable host or a from-scratch nogui-compatible rewrite (out of scope, no such rewrite exists).
- **No `-count>=10` stability sweep exists for `tests/scaling` at its current `Trials=5` setting.** `c7484cc7` raised `MeasureTrialsPerStep` 3→5 for signal-to-noise (confirmed in source: `tests/scaling/scaling_harness.go:177`) with a handful of direct reruns pasted in its own commit message, but no durable `qa-results/` artefact records a `-count>=10` repetition. The dispatching brief for this task additionally reported a run that hung at ~0.3% CPU and was killed without diagnosis; **this pass could not corroborate that specific detail from any log, process, or docs/qa artefact** — carried forward as reported, not independently confirmed, and should not be treated as resolved or as ruled out.
- **`debate_orchestrator` has only ONE configured remote** (`origin` → GitHub), while every sibling submodule fans out to 4 (github/gitlab/origin/upstream). Flagged for the operator per the dispatching brief's framing — a decision point, not a defect this pass is authorised to fix.
- **Independent review of `helix_agent`'s newest 2 commits is unconfirmed.** `129094b0` and `f451d342` landed after the reported 9-commit/3-round CLEAN GO ending at `0367570e`; no review artefact for these 2 was found by this pass. `f451d342` itself does textually remediate a Critical finding (see the Live-state anchors `helix_agent` row) — the fix reads as correct on inspection, but "reads as correct" is not the same as a completed independent review.

---

## In flight at handoff — status re-derived 2026-07-28 (§11.4.147: a crashed agent is NOT complete)

Four background agents were dispatched at the 2026-07-27 handoff. Re-derived status:

1. Independent review (§11.4.142) of the llamacpp fix + its RED test — **NOT VERIFIED.** No review artefact located this revision. Treat as outstanding; re-dispatch. Re-checked rev 7 (2026-07-28T17:37Z): still no matching artefact found under `docs/qa/`; unchanged.
2. G7 Class-C matching-bug reconciliation + paired mutation — **LANDED, verified.** All six former Class-C commits now report `ok` in the gate detail output (see the G7 section). The paired mutation was **not** separately verified.
3. Extend the discarded-error guard to 4 sibling sites (`internal/verifier/embedded_server.go:84`, `together/client.go:72`, `replicate/client.go:87,129`), TDD RED-first — **NOT VERIFIED.** Re-checked rev 7: `internal/verifier/embedded_server.go:84` STILL discards the error (`req, _ := http.NewRequestWithContext(...)`), unaddressed. `together/client_nilrequest_test.go` and `replicate/client_nilrequest_test.go` DO exist, but both predate this remediation cycle (added in commit `56f7edf7`, an old "Auto-commit") — they were not verified to satisfy this specific item's scope. Treat as still open.
4. Apply the prepared carrier elision patch (48 entries, 6 carriers) + verify §11.4.157 lockstep + regenerate exports — 5 of 6 carriers now RESOLVED (see Live-state anchors "Constitution ceiling vs carriers", reconfirmed unchanged rev 7); CRUSH.md's committed HEAD still stops at §11.4.234 while its own working-tree edit already reaches §11.4.235 — one commit away, still NOT LANDED.

Items 1 and 3 must be re-dispatched or re-verified — do not assume completion. Item 4 is nearly closed (one commit).

## Uncommitted work in the tree (snapshot 2026-07-28 — VOLATILE, re-derive)

Not a backlog of unpushed commits — in-flight working-tree state. Account for every file before any `git add` (§11.4.84); never `git add -A`.

⚠️ **SUPERSEDED COUNT AGAIN (rev 7, 2026-07-28T17:37Z): 73 porcelain entries (46 untracked + 27 modified), not ~68.** Re-derived fresh via `git status --porcelain | wc -l`. Continuing the same pattern rev 6 already described: mostly more new `docs/qa/<run-id>/` evidence directories from the same ongoing G7-evidence stream, expected and not concerning. This task's own scope fence additionally confirms these ~73 entries (minus this doc pair) belong to OTHER active streams (`helix_code/internal/llm/`, `helix_code/applications/`, and others) — do not stage them, ever, and never `git add -A`. The itemised lists below (21 modified / 5 untracked) are the rev-5 SNAPSHOT, now doubly stale against the current 73; re-run `git status --porcelain` yourself rather than trusting this itemisation. Two items below are also stale: `constitution` (gitlink) is annotated "6-behind pin" below — that is RESOLVED (see Live-state anchors, pin now matches checkout); `submodules/helix_agent` (gitlink) is listed as modified — confirmed still true, and additionally the submodule's own internal working tree is separately dirty (18 files this pass, see Live-state anchors), which this list does not capture (it only reflects the meta-repo's view of the gitlink pointer, not the submodule's internal state).

**Modified (21, incl. this doc pair):** `.docs_chain/contexts/fixed.yaml`, `.docs_chain/contexts/issues.yaml`, `.superpowers/sdd/progress.md`, `RESUME.md`, `config/llmsverifier/config.yaml`, `constitution` (gitlink — the 6-behind pin), `docs/CONTINUATION.md`, `docs/SYSTEMD.md`, `docs/scripts/verify_qa_evidence.md`, `helix_code/config/replica-8081.yaml` (operator decision 1), `helix_code/internal/llm/response_err_round53_test.go`, `helix_code/internal/tools/shell/{executor,output}.go`, `helix_code/tests/e2e/challenges/{functional_validator,helixcode_server_client}.go`, `scripts/run-all-tests.sh`, `scripts/systemd/helixcode-server.service` (operator decision 3 — secret env names), `scripts/systemd/helixllm-gateway.service`, `scripts/tests/verify_qa_evidence_meta_test.sh`, `scripts/verify_qa_evidence.sh`, `submodules/helix_agent` (gitlink).

**Untracked (5):** `docs/qa/phase1_fullhttp_e2e_20260728T091257Z/` and `docs/qa/phase1_fullhttp_e2e_20260728T093918Z/` — 5 captured HTTP evidence files each (401-no-auth, OpenAI chat-completions 200, Anthropic messages 200, OpenAI tool-calls 200, Anthropic tool-use 200); `helix_code/internal/tools/shell/stream_race_guard_test.go`; `helix_code/tests/e2e/challenges/helixcode_server_client_hostport_test.go`; `scripts/lib/`.

---

## Traps that already cost time — do not re-learn these

- **A "package ok" may be ambient, not repair.** `internal/llm` reported `ok` after the crash fix — including 3 round-53 tests that had failed. Those pass because `httptest` bound IPv4 that run, NOT because the IPv6 defect was fixed. It is unfixed. Green here is environment.
- **The legacy summary generators DESTROY data.** `scripts/generate_{issues,fixed}_summary.sh` read the TEXT trackers; committed summaries are SQLite-derived (§11.4.93/.95). Running them rewrites 344 items down to 188 — 156 items of tracked state destroyed.
- **`make test` can never pass on a headless host** — runs `go test ./...` with no `-tags=nogui`.
- **A "completed / exit 0" notification describes the WRAPPER, not the work.** Verify the artefact — mtimes, `pgrep`, pasted output.
- **`replace_all` is unsafe when a NEW helper's body contains the pattern being replaced** — rewrites the helper into infinite self-recursion.
- **PDF export silently drops wide tables' rightmost column** (§11.4.168) while reporting `rendered=2 failed=0`. Keep load-bearing evidence out of wide tables.
- **A §11.4.115 RED test that asserts against a local replica of the old behaviour, instead of driving the real production code path, is a bluff gate — it can never fail, so a PASS proves nothing.** Found (at least) twice this session in this exact shape: `3fd55a4d`'s F1 guard replicates the pre-fix `generateLLM` response construction locally rather than calling the real handler; per the dispatching brief (not re-verified by this pass) `debate_orchestrator`'s `protocol_convenience_form_red_test.go` had the identical defect, repaired in `b0655d3`. When authoring or reviewing any RED/GREEN polarity test, check specifically whether the RED branch calls into the actual function/handler under test.
- **A logo-fix-class subagent commit can sweep an unrelated file's staged state into an unintended commit if `git add` is not scoped to explicit paths** — the reason §11.4.84 forbids `git add -A`/`git add .` in a shared checkout with concurrent streams. This session's scope fence exists specifically because of this risk class; when in doubt about what a broad `git add` would sweep in, stage explicit paths only and re-check `git status` immediately before `git commit`.
- **`git ls-files '*.go' | xargs gofmt -w` reaches vendored AND archival trees.** Always exclude `docs/research/` (and check `node_modules`) before formatting.
- **`podman pod rm -f pod_helix_agent` KILLS `helixagent.service`.** Its container adapter sees the pod vanish and shuts down gracefully (`"Shutting down container adapter..."`, `status=0/SUCCESS`) — it looks like a clean exit, not a casualty. Cost this session: `chaos/api` + `chaos/auth` regressed `ok` → `connection refused` on `:7061`, and a whole 30-min sweep ran against a dead server. Stop the *service* first, or don't touch its pod.
- **`pgrep -f 'go test'` matches YOUR OWN shell** — the loop's command line contains the literal string. It killed my own bash mid-command (exit 144). Always exclude `$$`/`$PPID` and verify by `cwd` (§11.4.174 / §11.4.201 carrier footgun; §12.12 names this exact class).
- **`:0` failing with `EADDRINUSE` is the ephemeral-exhaustion tell.** Binding port 0 asks the kernel for *any* free port; it can only fail when the range is genuinely exhausted. That single line is the fastest way to distinguish host contention from a real allocator bug.
- **The test stack and the platform stack use DIFFERENT ports.** Precondition demands `15432`/`16379`/`18081` (`docker-compose.test.yml`); the live platform uses `5433`/`6380` (`helixcode-infra-*`). Seeing `5432`/`6379` listening proves nothing about test readiness — those belong to another project.

---

## Sweep commands (exact) — RUN SERIALLY, ONE AT A TIME

```bash
# Host must be quiet. Three of these in parallel exhausts IPv4 ephemeral
# ports, forces httptest onto its IPv6 fallback, and produces failures that
# are contention artefacts rather than code defects (§11.4.119).
cd helix_code            && go test -tags=nogui -count=1 ./...
cd submodules/helix_agent && go test -count=1 ./...
cd submodules/helix_qa    && go test -count=1 ./...
./scripts/verify-all-constitution-rules.sh
./scripts/verify-governance-cascade.sh
```

`-tags=nogui` is REQUIRED for the inner module (headless host, no X11/GL dev headers).
`-count=1` is REQUIRED — an earlier run reported 250 `ok` of which 249 were `(cached)`.

---

## Binding constraints (restated — §11.4.131(C))

- **Anti-bluff §11.4** — every PASS needs captured runtime evidence; metadata-only / config-only / absence-of-error / grep-without-runtime PASS are critical defects.
- **§11.4.113 — force-push is absolutely forbidden**, no exception, no operator override. Integrate by merging onto latest `main`, push fast-forward only.
- **§11.4.151** — tags carry the `helix-code-` prefix, identical across main repo and every owned submodule in one release.
- **§11.4.142** — every change needs INDEPENDENT review before commit/build; author self-verification (§11.4.92) precedes but never satisfies it.
- **§11.4.84** — account for every modified file before `git add`; unaccounted entries → ABORT. Never `git add -A` here.
- **§11.4.119** — one owner per exclusive resource; partition parallel work by RESOURCE, not by repository.
- **§11.4.185** — manual QA-team confirmation is the final sufficiency gate; not self-certifiable.
