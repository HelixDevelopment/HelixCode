# RESUME — session resumption record (§11.4.131)

**Rev 22 · 2026-09-03 ~08:20 CEST.** Supersedes rev 21 (same morning) and
rev 20 (2026-08-14). Rev 21 fixed the broken paths; this revision brings the
work state current after a long fix round.

Rev 20 was three weeks stale and **its first command did not work**: it said to
`cd /home/milos/Factory/projects/tools_and_research/helix_code`, which does not
exist on this host. `docs/CONTINUATION.md`'s own TL;DR names a third, also
non-existent path (`/run/media/milosvasic/DATA4TB/Projects/HelixCode`). The
real checkout is below. Rev 20 also reported "every repository is clean and
pushed" and declared no active programme; neither is true now.

> **Governing rule (kept verbatim from rev 20 — it has earned its place).**
> Every count, list, hash and stream-state below is a **snapshot**. Re-derive
> before acting on any of it; the commands are inline. Counts here have moved
> under scrutiny repeatedly.
>
> **This revision has a stronger reason than usual.** It was written while
> **six subagents were committing concurrently** to `submodules/helix_llm`,
> `helix_code/internal/config` and the Claude Toolkit. Every hash and count
> below was already moving as it was recorded. Treat all of it as "roughly
> here", never as current.

> **Instrument warning (kept from rev 20, measured 2026-08-14).** `grep` is a
> **shell function** in the agent shell (ugrep with `--ignore-files --hidden`)
> and **silently skips gitignored paths**: bare `grep -r` found 1 hit where
> `/usr/bin/grep -r` found 2. It is not exported, so `bash script.sh` children
> get real GNU grep. **Use `/usr/bin/grep` for every inline count** and say
> which instrument produced each number.

---

## 1 · Start here

```bash
cd /home/milosvasic/Projects/helix_code     # the ONLY correct path; three docs said otherwise
git fetch --all --prune
git log --oneline -1                        # rev 21 was written at 7d45aed4
git status --porcelain | wc -l              # 12 when written, and moving
git -C submodules/helix_llm log --oneline -1   # 6e278e8 when written
```

Read `.remember/now.md` first, then this file, then
`specs/002-adaptive-local-model-serving/progress.yml` — that last one is the
real record of what is broken and what has been proven about it.

---

## 2 · What is actually happening

**There IS an active programme.** Rev 20 and `docs/CONTINUATION.md` both say
there is not. That is the single most misleading thing in the older docs.

Feature **002 `adaptive-local-model-serving`** is mid-execution:
`specs/002-adaptive-local-model-serving/`.

- **89 of 97 tasks** complete (`/usr/bin/grep -c '^- \[x\]' .../tasks.md`).
- **93 findings** recorded in `progress.yml`. Roughly 20 open, and the open ones
  are now mostly decisions rather than unfinished work — see §5.
- Nothing is pushed. When this was written: meta-repo 13 ahead of upstream,
  `helix_llm` 20 ahead, `helix_agent` 3 ahead.

The 8 open tasks are NOT stalled work. Four (`T037`, `T055`, `T068`, `T084`)
are `[REVIEW]` tasks that **already ran and returned findings**; §11.4.134
requires iterating to a zero-finding GO, so they stay open until the fixes land
and a re-review comes back clean. `T052`/`T053` need a running system.
`T054` is an outward-facing publish and is explicitly an operator checkpoint.
`T097` is the final review.

---

## 3 · What was fixed in the 2026-09-02/03 session

Every one of these was **reproduced before being fixed**, and each carries a
paired mutation that was `diff`-verified as actually applied.

| What was wrong | Where |
|---|---|
| Published model identifiers were decorative. `ResolveModelName` was tested but never on the request path; `fallback.Chain` overwrote `req.Model` with each provider's FIRST model. A client asking for a `gpu-01` identifier was answered by `chutes` with `deepseek-chat`. | `helix_llm` `a5a2eb9` |
| Selection never read VRAM — it checked whether accelerators *existed*, then compared requirements against host RAM. `ResourceAccelerator` was declared in `option.go` and referenced nowhere else in the repo. | `helix_llm` `39701cd` |
| The VRAM broker parsed "the FIRST GPU row" and a test asserted that as correct. On a two-GPU host it refused a 6 GiB request while the other card had 12689 MiB free. | `helix_llm` `39701cd` |
| The attestation proof was `HMAC(secret, nonce)` — it proved someone held the secret, not *who answered*. A relay could borrow a genuine instance's answer to our own nonce. Reproduced with the user's prompt, an opened file and an upstream credential arriving at a hostile host. | `helix_llm` `3efc367` |
| The binding was established on the probe's connection but `Send` re-resolved the name, so DNS could move between the two requests. | `helix_llm` `6e278e8` |
| Unexpanded `${...}` used verbatim as a credential. With the variable unset, every JWT was signed with a string committed to this repository. | `helix_llm` `5b68ffa`, meta `ba7f6133` |
| `redis.db` was mis-tagged `database`, so **every deployment used Redis DB 0** whatever the operator configured. | meta `ba7f6133` |
| The videogen lane could plan a build the service refuses to load (two precision lists disagreed). | `helix_llm` `2167525` |

Landed later the same morning:

| What was wrong | Where |
|---|---|
| The HelixCode and OpenCode exporters had NO CALLER — those artifacts were unobtainable by any user. Git history showed never-completed wiring, not rot, so completing was right. | `helix_llm` `f63b96f` |
| The published identity named `127.0.0.1` on every machine, so it named no machine AND collided across machines; the toolkit's `group_by` then silently dropped the second host. | `helix_llm` `f63b96f` |
| Selection and the broker disagreed about headroom: a 10 GiB model on an 11.5 GiB card was offered and then refused. Two placements could also each be told the same card was free. | `helix_llm` `087947d` |
| `agentgen-boot` took its model from configuration and its VRAM figure from a SECOND env var a human had to keep in step. Naming a 19.5 GiB model with the figure untouched gave ADMIT-OK and exit 0. | `helix_llm` `0c43b11` |
| A listing of nothing returned `"data": null` with no reason — a body that reads as malformed, from the branch next to the one documenting that exact rule. | `helix_llm` `ab34fa8` |
| A request cancelled BEFORE it started could still complete: `select` raced an already-ready `ctx.Done()` against an already-ready response, and Go picks at random. 39/40 before, 40/40 after. | `helix_llm` `6fba621` |
| An unpaired bracket in a configured host silently discarded a NAMED PRODUCTION HOST and connected to loopback (`db.prod.internal[::1` → `::1`). | `helix_agent` `be54764d` |
| `production-config.yaml` was not valid YAML — an AI assistant's transcript had been pasted into it — and beneath that, 210 lines YAML was already discarding. | meta `4e2d742c` |
| The `notifications:` block reached nothing; and the expander used `os.Expand`, which eats `$$`, so an SMTP password `pa$$word` became `pa`. | meta `721f6c6e` |
| A whitespace-only credential validated cleanly and then rejected every request including the correct one. | `helix_llm` `ad813ef` |

**Two lessons worth carrying forward**, both recorded in `progress.yml`:

- *A fix that breaks a sibling gate is not automatically a stale gate.* The
  placeholder fix segfaulted on a nil optional section; its own tests missed it
  because their fixture populated that section, and a bcrypt guard caught it.
  Reconciling the gate instead of investigating would have shipped a crash.
- *"No output" is ambiguous between "clean" and "broken".* This bit FIVE times,
  including twice against me: a `pgrep` waiter that matched itself, a mutation
  `gofmt` silently prevented from applying, a crashed `grep`, a test log I had
  filtered myself and then counted, and a rejection message my own filter cut
  off. Always assert the check ran.
- *A flaky test and a real race look identical from the failure rate.*
  `TestLSPClient_ContextCancellation` was written off as flaky by two separate
  agents. The question that settled it was not how often it failed or whether it
  passed in isolation — both were true — but whether the scenario contained any
  LEGITIMATE timing. A context cancelled before the call contains none, so a
  random outcome could only be a defect.
- *A review finding is a place to look, not a thing to implement.* Three
  severity claims from one review did not survive verification, and in each case
  the fix that shipped differs from the fix the report implied. Two premises of
  MY OWN also turned out false and were caught by the agents I gave them to.

---

## 4 · Behaviour changes a user could notice

Not defects — deliberate, and worth knowing before someone reports them as bugs.

1. **Naming a model now pins it.** `gpt-4o` no longer cross-falls-back to
   another provider on failure. A named model is a choice, not a hint.
2. **A re-addressed instance needs a fresh `Discover`.** `Send` dials the
   address that authenticated, so a DHCP or container-restart address change is
   not followed until rediscovery. The alternative — re-resolving — is the hole.
3. **Startup refuses an unexpanded `${...}` credential.** A deployment relying
   on the old silent behaviour will now fail to start, with the field and the
   variable named in the error.

---

## 5 · Open, and needing a decision rather than more work

- **Selection uses zero VRAM headroom; the broker reserves 2 GiB.** A 10 GiB
  model on an 11.5 GiB card is offered and then refused — the user is shown
  something that cannot start. A seam exists (`Reserve.AcceleratorHeadroomBytes`)
  and nothing sets it. `OPEN-17`.
- **Concurrent placements do not draw down device memory.** `Commitment` has no
  accelerator dimension, so two placements can each be told the same card is
  free. `OPEN-18`.
- **Two integration model-listing tests fail at baseline** and three agents each
  independently proved the failures pre-date their work. Believed to need live
  services — but that is the assumption, not a finding. `OPEN-19`.
- **The shipped config writes `${...}` into fields nothing expands**, so a
  provider endpoint is a literal that is not a URL. `OPEN-15`.
- **13 credentials remain exposed and unrotated.** The operator's standing
  decision is "just keep the record" — do not add rotation tooling; keep the
  documented list current.
- **The agent lane's three former candidates are not in the catalogue**, so an
  operator serving Mistral-Nemo-2407, GLM-4.7-Flash or DeepSeek-Coder-V2-Lite
  through it will now be REFUSED. That is the flip side of the FR-056 fix: those
  three never had a measured footprint, which is exactly what the fixed 9 GiB
  placeholder stood in for. Resolve by measuring them on a host that HAS the
  GGUFs and adding catalogue entries with recorded provenance — or by accepting
  the narrower set. Do not estimate the figures. `OPEN-24`.
- **Five of six text catalogue entries carry `requires_accelerator: false`**, so
  selection skips the device axis and a host-RAM figure reaches a VRAM broker.
  The CRITICAL-5 shape in a narrower form; the vision lane has it too. `OPEN-25`.
- **`internal/lifecycle`'s concurrent-evict test fails under CPU contention**,
  on its own LIVENESS precondition rather than the invariant it guards. Unlike
  the LSP case this scenario does contain legitimate timing, so it is a genuine
  flake — but it should retry or SKIP with a reason rather than FAIL and imply
  the invariant broke. It has already cost two agents time. `OPEN-23`.

---

## 6 · House rules that bit someone this session

- **Stage by path, never `git add -A`.** Multiple agents share this checkout.
  One broad `git add` earlier swept another agent's in-flight file into an
  unrelated 46-file commit.
- **Prove a mutation applied before believing its result** (`diff` it). A
  mutation that `gofmt` had realigned past reported "ok" and proved nothing.
- **Force-push is forbidden, no exception** (§11.4.113). Integrate by merging
  onto latest main and push fast-forward only.
- **Every change gets independent review before commit/build** (§11.4.142),
  iterated to a zero-finding GO (§11.4.134).
- **`services/vectorize/__pycache__/`** and friends are now gitignored — they
  were one careless `git add` from being versioned.

---

## 7 · Resume prompt

```
Read RESUME.md then specs/002-adaptive-local-model-serving/progress.yml, run
`git fetch --all --prune --tags`, and continue feature 002. Use
subagent-driven development (§11.4.70) by default and fan out on disjoint file
scopes. Reproduce every reported finding before fixing it — several turned out
to be real and one turned out to be my own regression. Every fix needs a paired
mutation, diff-verified as actually applied. Stage by path, never `git add -A`;
other agents share this checkout. Force-push is forbidden (§11.4.113).
```
