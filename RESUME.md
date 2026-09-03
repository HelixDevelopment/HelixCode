# RESUME — session resumption record (§11.4.131)

**Rev 23 · 2026-09-03 ~14:00 CEST.** Supersedes rev 22 (same morning),
rev 21 and rev 20 (2026-08-14).

**What changed since rev 22, and it is the headline: EVERYTHING IS NOW PUSHED,
and the system BOOTS AND SERVES.** Rev 22 said "Nothing is pushed"; that is no
longer true and §2 below is corrected. Three defects that each independently
stopped the stack from starting were found by actually running it, and fixed.

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

> **READ THIS FIRST: the naming/export batch is NO-GO.** An independent
> re-review confirmed four findings BY RUNNING CODE, and THREE OF THEM WERE
> CREATED BY THE FIXES. Two are user-facing and serious: a user holding a
> pre-fix `helixllm-127-0-0-1-…` identifier is now silently answered by the
> WRONG MODEL (the original CRITICAL, reproduced for the population the batch
> itself created), and `--apply` DELETES a user's entire configuration for a
> host that is merely restarting. Fixes are in flight. Do not treat this batch
> as done, and do not trust an earlier summary that says the fixes landed
> cleanly — they landed, and they each created something new.

- **89 of 97 tasks** complete (`/usr/bin/grep -c '^- \[x\]' .../tasks.md`).
- **93 findings** recorded in `progress.yml`. Roughly 20 open, and the open ones
  are now mostly decisions rather than unfinished work — see §5.
- **Everything is pushed** (2026-09-03 ~13:45 CEST), fast-forward, no force.
  Every remote was `behind=0` beforehand, so no merge was needed:

  | repo | HEAD | upstreams |
  |---|---|---|
  | meta | `b752a807` | GitHub `Helix-CLI`, GitLab `HelixCode` |
  | `helix_llm` | `1efda3b5` | GitHub, GitLab |
  | `helix_agent` | `64cf8921` | `HelixDevelopment/HelixAgent`, `vasic-digital/HelixAgent` |
  | `claude_toolkit` | `328cf27b` | GitFlic, GitHub, GitLab, GitVerse (branch `fix/helixllm-export-review-findings`, NOT merged to main) |

  Two upstreams reported a **move**: `HelixLLM` -> `HelixDevelopment/llm.git`
  and `Helix-CLI` -> `HelixDevelopment/code.git`. Pushes still succeed via the
  redirect; the configured URLs are stale and worth updating.
  GitHub also reports **81 Dependabot vulnerabilities** on
  `vasic-digital/HelixAgent` (1 critical, 56 high, 21 moderate, 3 low).

The 8 open tasks are NOT stalled work. Four (`T037`, `T055`, `T068`, `T084`)
are `[REVIEW]` tasks that **already ran and returned findings**; §11.4.134
requires iterating to a zero-finding GO, so they stay open until the fixes land
and a re-review comes back clean. `T052`/`T053` need a running system.
`T054` is an outward-facing publish and is explicitly an operator checkpoint.
`T097` is the final review.

---

## 2b · How to actually run the system (verified 2026-09-03)

The containerised route (`./helix start`) does NOT work yet — see `BOOT-4`. Use
the native route, which is the one Rule 4 names first (`make build` ->
`./bin/<app>`). The server auto-boots its own Postgres and Redis in podman per
§11.4.76, so no compose file is needed at all.

```bash
cd /home/milosvasic/Projects/helix_code

# .env is gitignored; generate the three REQUIRED secrets if absent
cp .env.example .env && chmod 600 .env
for k in HELIX_DATABASE_PASSWORD HELIX_REDIS_PASSWORD HELIX_AUTH_JWT_SECRET; do
  sed -i "s|^$k=.*|$k=$(openssl rand -hex 32)|" .env
done

cd helix_code && go build -o bin/helixcode ./cmd/server
set -a; . ../.env; set +a
HELIX_REDIS_HOST=localhost ./bin/helixcode
```

The server REFUSES to start if a secret is missing, naming the variable — that
is deliberate, and it is the placeholder guard added earlier in this programme:

    config validation failed: auth.jwt_secret is still the unexpanded
    placeholder for ${HELIX_AUTH_JWT_SECRET} ... Export it and start again.

On success it prints `Infra auto-boot: podman booted postgres:<p> redis:<p>`
and listens on **:8080**. Two containers appear, `helixcode-autoboot-postgres`
and `helixcode-autoboot-redis`. Opt out with `HELIX_AUTOBOOT_INFRA=false` to
use external infra instead.

**Do not run the test suites while the server is up.** Verified this session:
a live server on 8080 silently changes outcomes in BOTH directions — it wakes
the otherwise-vacuous `tests/memory` suite (`TEST-2`) and breaks a challenge
asserting a CLOSED port reports SKIPPED. See `METHOD-1`.

Verified working end-to-end:

| probe | result |
|---|---|
| `GET /health` | 200 `{"status":"healthy","version":"1.0.0"}` |
| `GET /api/v1/server/info` | 200, `database.connected=true` |
| `GET /api/v1/llm/providers` | 200, 8 providers |
| `GET /api/v1/llm/models` | 200, 8 models |
| `GET /api/v1/memory/systems` | 200, 6 systems |
| `GET /api/v1/metrics` | 200, live pool 6 active / 6 idle / 20 max |
| `POST /api/v1/auth/register` | 201, user persisted with a real UUID |
| `POST /api/v1/auth/login` | 200, 287-char JWT |
| `/tasks` `/workers` `/system/stats` | 401 without a token, 200 with one |
| `GET /api/v1/auth/me` | 404 despite being registered — `OBS-1` |

---

## 2b-2 · HelixLLM, HelixAgent, and the Claude Toolkit sync (verified 2026-09-03)

### HelixLLM — port 8443, **HTTPS**, self-signed

```bash
cd submodules/helix_llm
go build -o bin/helixllm ./cmd/helixllm
HELIX_MODE=full ./bin/helixllm            # listens on :8443
curl -sk https://127.0.0.1:8443/v1/models # -k, or use the CA bundle below
```

Plain HTTP is refused with `400 Client sent an HTTP request to an HTTPS server`
— that 400 is the TLS mismatch, not a broken endpoint.

> **CHECK THE RUNNING BINARY'S AGE BEFORE TRUSTING ANY PROBE.** A `helixllm`
> that had been up for 16 hours was still serving `{"object":"list","data":null}`
> — the defect `ab34fa8` had ALREADY fixed in source. Source-green said nothing
> about what was serving (§11.4.108 SOURCE→RUNTIME). Compare
> `ps -o lstart= -p <pid>` against `git log -1`, and rebuild before believing a
> live result.

### The Claude Toolkit sync — two gotchas that will cost you an hour

```bash
export CURL_CA_BUNDLE=$PWD/submodules/helix_llm/certs/cert.pem
claude-providers helixllm-export --host https://localhost:8443/v1
```

1. **The base URL must already contain `/v1`.** `_cma_helixllm_fetch_models`
   appends `/models`, so `--host https://…:8443` requests `/models` and misses.
2. **The self-signed cert needs a trust anchor.** The fetcher uses a bare
   `curl -sf` with no `--cacert`, so it fails with **curl exit 60** and the tool
   reports only *"host … did not answer with a model listing"* — which reads like
   the host is down when it is answering perfectly. `CURL_CA_BUNDLE` fixes it
   with no code change; the cert covers `localhost`, `127.0.0.1`, `192.168.0.241`.

**What it exports today: nothing, correctly.** HelixLLM catalogues exactly one
model, `helixllm/anton/Llama-3.1-70B-Instruct-Q4_K_M`, as
`availability: withheld / provider_unavailable`, and the toolkit refuses to
advertise a model that is not actually being served. `anton` IS this host, there
are no GGUF weights on it, and no llama.cpp server is running. Note also that
`HELIX_LLM_LOCAL_MODEL` defaults to that 70B Q4_K_M (~40 GB) on a box with
**30 GB RAM and 12 GB VRAM** — it could not run it even with the weights.

**To actually get a usable model into Claude Code** you need a served model:
place a GGUF this host can run (RTX 3060 / 12 GB — an 8B Q4_K_M at ~4.9 GB fits
comfortably), point `HELIX_LLM_LOCAL_MODEL` at it, run `llama-server`
(`/usr/bin/llama-server` is installed; ollama is NOT), then re-run the export.

### HelixAgent — boots its whole stack on podman, needs one secret

```bash
cd submodules/helix_agent
go build -o bin/helixagent ./cmd/helixagent
printf 'JWT_SECRET=%s\n' "$(openssl rand -hex 32)" >> .env && chmod 600 .env
set -a; . ./.env; set +a && ./bin/helixagent
```

Observed on a real run: `Container adapter initialized via Containers module
runtime=podman`, postgres+redis+chromadb started via `podman-compose` with all
three health checks PASSED, 1163 skills across 16 categories, 32 MCP servers,
liveness probe on `:8111` — then it stops with
`Failed to initialize auth middleware: JWT secret key is required`. That is the
only blocker; `.env` is gitignored there (`.gitignore` lines 3/38/51).

Its LLMsVerifier pipeline also completes with **0 providers discovered** out of
44 candidate env vars — HelixAgent exposes no models because no provider keys
are in ITS environment. The operator's keys live in `~/api_keys.sh`, which the
toolkit reads and `helixagent` does not.

---

## 2c · Full retest results (2026-09-03, and how to read them)

| repo | result |
|---|---|
| `helix_llm` | **exit 0 — 54 packages ok, 0 FAIL** |
| `helix_agent` | **exit 0 — 289 packages ok, 0 FAIL** |
| `helix_code` | exit 1 — 192 ok, 7 failing packages (below) |
| `claude_toolkit` | exit 1 — 2279 assertions pass, 12 fail = 2 root causes |

`helix_code`'s 7 failures are **3 real and 4 artefacts of the sweep itself**:

- **3 GUI packages** (`applications/{desktop,aurora_os,harmony_os}`) fail to
  BUILD. Genuine environment gap: this host has no OpenGL/X11 development
  headers, so the Fyne -> go-gl -> glfw chain cannot compile
  (`gl.pc` and `X11/Xlib.h` both absent). Not a code defect. `make
  desktop-nogui` exists precisely for this. Server and CLI build fine.
- **4 timing / connection-pool packages** fail inside `go test ./...` and
  PASS when run alone:

  | package | in the parallel sweep | alone, quiet host |
  |---|---|---|
  | `internal/providers/httpclient` | FAIL | ok 0.015s |
  | `tests/performance/scenarios` | FAIL | ok 3.980s |
  | `tests/regression` | FAIL | ok 1.792s |
  | `tests/memory` | FAIL | ok 98.991s (all 15 tests) |

**Why, and it is structural.** `go test ./...` runs package binaries in
parallel up to GOMAXPROCS. Measured on this host: the `helix_code` sweep alone
drove load to **70 on 16 CPUs**. Every one of those four measures something
load-sensitive — wall-clock stability, HTTP keep-alive pool reuse, post-GC live
heap. They cannot be measured reliably inside their own sweep.

So the honest reading is: **the suite is green apart from a documented
toolchain gap**, but the timing-sensitive packages need `-p 1` or a separate
target. Do not "fix" them by loosening their bounds — that is the §11.4.120
forbidden move, and their bounds are what make them worth having.

There is a good precedent already in the tree: `helix_llm`'s
`internal/testing` DETECTS the contention and skips that step with a reason
("host too loaded to measure concurrency ... re-run on a quieter host").
Its only flaw is that the wrapper asserts "passed" rather than accepting
"skipped". That pattern is worth generalising.

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
- **The containerised boot is still broken.** `helix_code/go.mod` has 43
  `replace` directives pointing at `../submodules/*`, outside the directory the
  Dockerfile copies, so `go mod download` fails inside the image. The targets
  total 4.2 GB (helix_qa 2.5 GB, helix_agent 2.0 GB, mostly vendored trees), so
  the fix needs a layout change plus a scoped `.dockerignore`. The native route
  works and needs no compose — see §2b. `BOOT-4`.
- **The whole `tests/memory` suite has been silently vacuous.** Every test in it
  probes `/health` first and `t.Skip`s when nothing answers, so with no server
  up the package reports `ok ... 0.077s` having exercised nothing. With a server
  up it runs for 23s and the concurrent-request test produced a rising live-heap
  signal (signed-R2 0.6565, rise 51.57% of mean). That signal is NOT yet a
  defect — it was taken at load 44, the heap is under 1 MB, and the shape is flat
  for 7 waves then a late jump, as consistent with transport idle-pool growth as
  with a leak. Needs a quiet-host re-run. The coverage gap is the real finding.
  `TEST-2`.
- **A Claude Toolkit test asserts against the operator's live `$HOME`.**
  `scripts/tests/test_claude.sh:7` hardcodes
  `ALIAS_FILE="$HOME/.local/share/claude-multi-account/aliases.sh"` and sources
  only `lib/assert.sh`, never `lib/sandbox.sh`. It FAILs `xiaomi uses
  cma_run_provider` purely because the operator has no `xiaomi` provider
  configured. Same class as helix_agent's `913d1f02` ("validate the config this
  repo ships, not the operator's").
- **`internal/lifecycle`'s concurrent-evict test fails under CPU contention**,
  on its own LIVENESS precondition rather than the invariant it guards. Unlike
  the LSP case this scenario does contain legitimate timing, so it is a genuine
  flake — but it should retry or SKIP with a reason rather than FAIL and imply
  the invariant broke. It has already cost two agents time. `OPEN-23`.

---

## 6 · House rules that bit someone this session

- **Staging by path is NOT enough. Use `git commit -- <paths>`.** Multiple
  agents share this checkout, and `git add <path>` adds to the index while
  `git commit` commits the WHOLE index — including whatever another agent
  staged before you got there. A broad `git add -A` swept an in-flight file
  into a 46-file commit early in the session; later, staging exactly one file
  by path swept 1424 lines of another agent's gateway work into a commit
  titled "docs(faq)". The second happened AFTER the first lesson was written
  into this file. Either use the pathspec form, or read
  `git diff --cached --stat` immediately before committing and confirm it
  lists only your files.
- **A git lock here is usually contention, not deadlock.** Something in this
  environment runs `git status --porcelain` periodically and it takes
  `.git/index.lock` (status refreshes the index). It is a live holder, so do
  NOT remove the lock — retry with a short backoff; it clears in about a
  second. Verify liveness before ever considering removal, and note that a
  `stat` on an absent lock returns 0, which turns an "age" calculation into
  epoch-seconds that look like data.
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
`git fetch --all --prune --tags`, and continue feature 002. Everything is
pushed as of rev 23 (meta b752a807, helix_llm 1efda3b5, helix_agent 64cf8921,
toolkit 328cf27b on a branch) and the system BOOTS — start it with §2b, not
`./helix start`, which is still broken (BOOT-4).

Use subagent-driven development (§11.4.70) by default and fan out on disjoint
file scopes. Reproduce every reported finding before fixing it — several turned
out to be real and one turned out to be my own regression. Every fix needs a
paired mutation, diff-verified as actually applied. Stage by path, and use
`git commit -- <paths>`; staging alone does NOT scope a commit, and other
agents share this checkout. Force-push is forbidden (§11.4.113).

NEVER run the test suites while the server is up, and never run several suites
at once. Both were measured this session to manufacture false failures —
timing, ephemeral-port exhaustion and live-server interference (METHOD-1). Note
this host carries a persistent ~50% background load from OTHER projects
(`kfl`, a `MainThread`, qbittorrent); verify process ownership by cwd before
attributing or acting on anything (§11.4.174).
```
