# SDD progress ledger — helix_code platform remediation (2026-07-28)

Base commit at session start: 0a4eb8d0 (main, clean, pushed)
Rule: tasks marked complete here are DONE. Do not re-dispatch. Trust this + `git log` over recollection.

## Complete

- Task A: helixllm-gateway env-var-name defect — complete (main stream).
  Unit set HELIX_CACHE_REDIS_HOST/PORT; config.go:121-122 reads HELIX_REDIS_HOST/PORT.
  Also HELIX_MODELS_DIR unset -> config.go:70 default "/models" (container path).
  Fixed in ~/.config/systemd/user/helixllm-gateway.service.
  Runtime signature verified post-restart: '6379', 'Redis unreachable', 'mkdir /models' ALL ABSENT;
  "server listening" addr="0.0.0.0:8443". Warnings 8 -> 1.

- Task B: playwright-mcp crash-looping build — complete (main stream).
  Dockerfile.mcp-playwright bind-mounted packages/playwright-mcp/* ; pinned submodule v0.0.77 is FLAT
  (no packages/ dir). Fixed to flat layout. NOT YET build-verified.

- Task C: boot-survival wiring — complete (main stream).
  All 6 units + helix.target enabled. default.target.wants/ -> helix.target ->
  helix.target.wants/{6 services}. Linger=yes confirmed.
  CORRECTION: helix.target never needed editing; units declare WantedBy=helix.target,
  they were merely disabled.

- Task D: internal/llm IPv6 + perf classification — complete (subagent, GREEN proven).
  net.JoinHostPort fix + TestMustSplitHostPort_BracketsIPv6 guard.
  tests/performance/scenarios classified CONTENTION ARTEFACT with cross-sweep evidence.

- Task E: ensure-infrastructure.sh port contract — complete (subagent). bash -n exit=0.
  Ports resolve once for both boot and health-check. UNVERIFIED: not booted.

- Task F: sweep serialization — complete (subagent). Host-global flock in scripts/lib/port_sweep_lock.sh.
  Also repaired run-all-tests.sh being dead-on-arrival (cd HelixCode; ((N++)) under set -e).

- Task G: G7 qa-evidence gate matcher — complete (subagent). 18 -> 13 violations.
  Class C already fixed at 56f7edf7; found+fixed a second matcher bug (paired close-commit SHA citation),
  §1.1 mutations added.

## In flight (do not re-dispatch)

- e2e IPv6 bracket fix (tests/e2e/challenges) — subagent running
- docker-compose.test.yml defects (redis healthcheck auth, mock-llm port) — subagent running
- systemd unit env-var mismatch audit (read-only) — subagent running

## Blocked / open

- Task 2: helix_agent full sweep for a trustworthy failure list — needs quiet host, agents must land first
- Task 9: competing container orchestrators (architectural) — operator decision
- G7 residual: 13 (10 genuine Class A incl. 3 security fixes, 3 Class B non-features).
  RELEASE-GATE BLOCKER. [no-qa-evidence] lives in already-pushed commit messages;
  retroactive use needs history rewrite = FORBIDDEN (§11.4.113).

## Complete (round 2)

- Task H: e2e IPv6 bracket fix — complete (subagent). tests/e2e/challenges.
  KEY FINDING: a BARE net.JoinHostPort would have REGRESSED already-bracketed input
  ([::1] -> [[::1]]); measured, not assumed. Fix unwraps surrounding [] first.
  RED reproduced the log error verbatim; paired §1.1 mutation FAIL->restore->PASS, sha256 byte-exact.
  go vet exit=0. Guard: helixcode_server_client_hostport_test.go.

- Task I: docker-compose.test.yml defects — complete (subagent).
  D1 redis healthcheck had NO -a against --requirepass server. THIRD issue found beyond the brief:
  `incr ping` is a WRITE — incremented a key every 10s and grew the AOF forever. Fixed to
  authenticated `ping | grep -q PONG`, password anchored ONCE (&redis_password) so server,
  healthcheck and helixagent all resolve to one scalar.
  D2 helixagent provider URLs pointed at mock-llm:8081; mock-llm listens on 8090. Fixed (6 URLs).
  Validated: python yaml OK + `podman compose config --quiet` EXIT=0 + engine-rendered $$ unescape proven.
  REPORTED NOT CHANGED: oauth-mock missing SERVER_PORT (ci variant proves 7061 isn't default);
  helixagent PORT:8080 vs mapping 8080:7061 vs healthcheck :7061 contradiction; postgres pg_isready
  proves reachability not credentials; ollama healthcheck uses curl (presence UNCONFIRMED).

- Verified this session (main stream, live): build clean (helix_code -tags=nogui + helix_agent),
  go vet clean, anti-bluff smoke CLEAN (zero hits in internal/ cmd/).

## In flight (do not re-dispatch)
- systemd unit env-var mismatch audit (read-only)
- docs/SYSTEMD.md sync
- compose follow-ups (oauth-mock SERVER_PORT, helixagent PORT contradiction)
- helix_code inner sweep (serial, running)
- playwright-mcp image build (verifying Dockerfile fix)

## Complete (round 3)

- Task J: llmsverifier JWT secret P0 SECURITY — complete (main stream), runtime-verified.
  config/llmsverifier/config.yaml omitted api.jwt_secret with a comment claiming it came from
  LLM_VERIFIER_API_JWT_SECRET. Operator HAD set it in .env. viper SetEnvPrefix without
  SetEnvKeyReplacer maps api.jwt_secret -> "LLM_VERIFIER_API.JWT_SECRET" (dot preserved), so it
  was NEVER read; service ran on the committed placeholder (config_loader.go:190).
  Fixed with explicit ${LLM_VERIFIER_API_JWT_SECRET} placeholder (expandEnvVar DOES handle that form).
  VERIFIED: 'Using default JWT secret' ABSENT after restart; /api/scores 200 real payload;
  0 warn/error lines in journal.

- Task K: SOURCE<->ARTIFACT gap on the gateway fix — complete (main stream).
  My fix existed only in ~/.config/systemd/user/; scripts/systemd/helixllm-gateway.service (TRACKED)
  still had HELIX_CACHE_REDIS_*. Next install_systemd_units.sh run would have silently reverted it.
  Fixed template; verified IDENTICAL env-var name sets template vs installed.

- Task L: oauth-mock + helixagent PORT — complete (subagent).
  oauth-mock had no SERVER_PORT (upstream default 8080, not 7061 — verified against upstream README)
  AND probed a non-existent bare /.well-known/ path (correct form is /{issuerId}/.well-known/).
  helixagent PORT:8080 was LIVE config set to the WRONG value (bind chain traced:
  internal/config/config.go:315 -> cmd/helixagent/main.go:1418/1424/1792 -> transport/http3.go:113).
  Set to 7061. Both YAML + `podman compose config --quiet` EXIT=0.

- Task M: verifier wiring + dead providers — complete (subagent).
  CONST-036/037/038 was OFF: no verifier: section -> enabled=false, endpoint defaulted to
  http://localhost:8081 (the server itself). Added correct block from verifier_config.go schema
  (remote, :8100, polling 60s satisfying CONST-038, weights sum 1.0).
  TWO defects the audit missed: (1) ${VAR:default} expansion reaches only 5 fields
  (config.go:456-481) so provider endpoints were RAW LITERALS never URLs; (2) helix-llm and
  helix-debate provider TYPES are UNIMPLEMENTED (factory.go:101 unsupported provider type).
  Disabled both rather than repointing (a plausible URL would imply a route that doesn't exist).

- Task N: docs/SYSTEMD.md — complete (subagent). Rev 1 -> 2. Corrected stale health route
  (/internal/health, not /v1/models), verifier-scores gap now resolved, 11 infra services (not 9),
  boot chain + enable/disable procedure, 2 new gotchas. No .html/.pdf exist (flagged, not invented).

## OPEN — needs operator decision (§11.4.66), surfaced not decided
- helixcode-server calls infraboot.EnsureInfra which UNCONDITIONALLY overwrites DB/Redis config
  (infraboot.go:247-257) with its own 55432/56379 and boots a SECOND postgres+redis pair, while the
  unit declares Wants=helixcode-infra.service and ignores it. All DB/Redis lines in replica-8081.yaml
  are dead config. Opt-out: HELIX_AUTOBOOT_INFRA=false. Captured: "Infra auto-boot: endpoints already
  healthy (postgres:55432 redis:56379)".

## PENDING VERIFICATION
- 5 failing pkgs from the quiet-ish sweep: internal/tools/shell, internal/verifier, internal/worker,
  tests/scaling, tests/stresschaos. ALL timing/concurrency/stress. Confound: I ran a 104-package
  container build alongside (load only 2.71/64 though). TestStreamingExecution looks REAL:
  ExitCode==0 passed but ZERO stdout lines captured from `for i in 1 2 3 4 5; do echo $i; done`.
  NEXT: N-run determinism check on a quiet host (§11.4.50) to separate flaky from real.

## Complete (round 4) — operator decisions executed 2026-07-28

- Decision 1 (INFRA OWNER = helixcode-infra) — DONE, runtime-proven.
  Safety first: autoboot helixcode_prod held 15 tables / ZERO live rows (schema only) -> switch
  cost no data. Dumped anyway (qa-results/db_backups/, 40K, 15 CREATE TABLE) per §9.2, restored
  into infra helixcode_test (15 tables, 0 errors).
  HELIX_AUTOBOOT_INFRA=false in BOTH installed unit AND scripts/systemd/ template.
  replica-8081.yaml: 5432->5433, helix->helixcode, helixcode_prod->helixcode_test,
  redis "redis"->localhost, 6379->6380.
  PROOF: MainPID TCP -> 127.0.0.1:5433 (x6), :6380, :8100. Journal:
  "Infra auto-boot disabled — using configured infrastructure" / "Database connection established
  successfully" / "Database schema already exists". '55432' count=0. /health 200. NRestarts=0.
  Autoboot container now stopped. The :8100 connections confirm the CONST-036 verifier wiring is LIVE.

- Decision 3 (SECRET ENV NAMES) — DONE, adjusted with operator informed.
  Chose ADD-alongside rather than rename: DB_PASSWORD is read by 94 files and JWT_SECRET by 68,
  incl. scripts/systemd/helixagent.service — a straight rename would BREAK helixagent.
  Added HELIX_DATABASE_PASSWORD + HELIX_AUTH_JWT_SECRET to .env (mode 0600 preserved, gitignored,
  backup .env.bak.20260728 also gitignored). HELIX_DATABASE_PASSWORD synced to infra postgres's
  actual password so the connection genuinely authenticates (proven by the live 5433 conns above).

- Operator mandate: main = Track 1 — DONE, pushed to all 8 constitution upstreams (32d7578).
  scripts/multitrack/track_branch_label.sh now derives branch BEFORE track and maps main|master->1.
  Verified: (T1/main - claude2 - opus - xhigh). TRUNK RULE clause added inside the existing
  §11.4.182 block in all 5 carriers (no new anchor, §11.4.227(B)); mirrors byte-identical.

- Determinism (§11.4.50): 5 previously-failing pkgs pass 3/3 EACH on a quiet host, 15/15 total.
  Classified CONTENTION ARTEFACTS, not defects. shell passed WITH the race fix applied; the race's
  reality is proven by the agent's deterministic guard, not by this run.

## Corrections to my own earlier claims
- I fixed Dockerfile.mcp-playwright line 22 but NOT line 37; the build failed at exactly line 37.
  Now fixed. Rebuild pending.
- I said helix.target "does not pull in" the other services — WRONG. All units declare
  WantedBy=helix.target; they were merely disabled. No target edit was needed.

---

## Session 7237795e — publish sweep + Critical/High burn-down (2026-07-29 → 07-31)

Written retroactively. This session was compacted once and lost two agent fleets to
session limits, and the ledger had ZERO entries for any of it — exactly the
"controller loses its place" failure the skill names as most expensive. Recorded now
so a fresh session can recover from git rather than from my memory.

### Landed and PUBLISHED (main @ 342c948f; ahead=0 behind=0; all remotes ls-remote MATCH)

- HXC-203: complete — `b2215793` fix (LocalLLMManager race: query-named method mutated
  shared state, type had no mutex, returned the live internal map), `be3e6605` +
  `1c2e7d07` evidence. Independent review to clean GO.
  Honest limit recorded in that commit: no provider was ever `running` during the
  tests, so refresh phases 2-3 INCLUDING the CAS rest on reasoning + review, not on
  executed evidence.
- Tracker regen: `11f0b7e1` — 401 items from the SSoT, absolute paths (see HXC-201).
- Publish sweep, 6 commits `8d1100a7..342c948f`:
  - `8d1100a7` gitignore two 9 MB untracked ELF binaries (`/bridge`, `/qa`);
    provenance read from `go version -m`, patterns root-anchored so `docs/qa/`
    (126 evidence dirs) is not swallowed. Both directions verified.
  - `e850cd02` submodule pointers + inner `go.sum` as ONE atomic change —
    `helix_code/go.mod:240` has `replace dev.helix.agent => ../submodules/helix_agent`,
    so submodule dep bumps invalidate the parent's checksums. Either alone breaks build.
  - `72967fcd` track the never-drop directive queues (they were UNTRACKED).
  - `de622751` 36 QA evidence runs; the Bearer literal is a tracked test constant at
    `helix_code/internal/server/wire_facade_live_e2e_test.go:62`, not a leak.
  - `342c948f` reconcile the toolschema i18n gate (§11.4.120: the fix `8cec90b` made
    its RED anchor obsolete), upgrade grep→runtime via `go test -overlay`, register G22.
    Four-quadrant polarity proof re-run independently. Sweep: 22 gates, 0 failures.
- Submodules published + verified at remote: constitution `02a3520` (8/8 remotes),
  helix_agent `453f18ff` (4/4), github_pages_website `ead9b2b` (3/3).
  All 227 submodules clean, 0 dirty, 0 unpushed.

### IN FLIGHT — 6 parallel implementers, disjoint files (do NOT re-dispatch)

Dispatched 2026-07-31. If you are a fresh session and these are unreported, check
`git log` for their commits BEFORE re-dispatching — they may have landed.

| Item | Sev | Scope |
|---|---|---|
| HXC-205 | Critical | auto model manager: declares a mutex, writes shared state in 3 places without it. Apply the proven HXC-203 three-phase shape. 8th sibling-miss this cycle. |
| HXC-169 | Critical | container-sandbox library: 4 advisories on copy-in/out ops we call, 3 with NO fixed version. Reachability DECISION, not an upgrade. Disabling needs operator sign-off (§11.4.122) — recommend only. |
| HXC-198 | High | sibling hang in the progress-reporting fn, same file as the one just fixed. 6th sibling-miss. |
| HXC-194 | High | hand-rolled server accepts any origin. TRAP: the upstream advisory is a DISTINCT problem that merely looks the same — bumping the dep silences the scanner and leaves the hole open. |
| HXC-168 | High | DB password literal published on 4 remotes. CODE HALF ONLY; rotation is operator action. Item does NOT close on the code half. |
| HXC-201 | High | documented tracker-regen command writes to the wrong place (`go run -C` changes child cwd). Enshrined across ~21 G20-protected sites. |

Every dispatch requires: §11.4.115 RED-first on the pre-fix artifact, a registered
§11.4.135 guard in the same commit, and a SIBLING SWEEP — because 4 of these 6 are
themselves sibling-misses, which is one review-step gap, not N independent bugs.

### Still owed after the wave
- Independent code review of every landed change (§11.4.142) on Fable @ xhigh
  (§11.4.209), iterated to zero findings AND zero warnings (§11.4.134).
- Fresh full-suite retest on a quiescent host.
- Operator §11.4.185 manual-QA sign-off — CANNOT be self-certified; blocks the tag.
- No `helix-code-1.2.0-dev-*` tag exists yet; latest is `helix-code-1.1.0-dev-0.0.3`.

### Corrections to my own claims this session (§11.4.6)
- I twice captured `tail`'s exit code instead of the command's. `cmd | tail; echo $?`
  reports 0 on a failed build. Redirect to a file, capture `$?`, then read.
- I told a subagent its package broke the tree; per-package isolation showed it built
  fine — the real cause was the `replace`-edge go.sum invalidation above. Retracted.
- A doc in this repo claimed `go run -C` does NOT change the child's cwd. It DOES.
  That claim was "verified" by reading, not running. It is HXC-201's root cause.

### Agent registry — crash + respawn log (§11.4.147: crashed != complete)

2026-07-31. Two of the six implementers died on transient `API Error: Connection
closed mid-response` — the same class that killed two fleets earlier this session.

| Item | Outcome | Partial state | Action |
|---|---|---|---|
| HXC-168 | crashed mid-investigation | tree verified CLEAN — wrote nothing | RESPAWNED with its own lead carried forward, marked UNVERIFIED |
| HXC-198 | crashed just before writing the RED guard | tree verified CLEAN — wrote nothing | RESPAWNED fresh; it reported no findings, so nothing to inherit |

Quiescence check before respawning (§11.4.84): `git status --porcelain` = 0 lines,
zero mutation residue, HEAD unchanged at `d9ad0b20`. So this was a clean restart,
not a resume — no partial edits to reconcile and no risk of a half-written file
being swept into someone else's commit.

The HXC-168 lead worth preserving even though its author died mid-sentence: it
claimed the `.env` plumbing ALREADY exists and is correct (`.env.example:8`
placeholder, `setup.sh:123` copies at 0600, `.gitignore:106-108` protects it) and
that the container files simply do not USE it. If true the job is wiring, not
building. Passed to the respawn explicitly flagged as unverified — a dead agent's
last words are a lead, not a finding.

An eighth agent was also dispatched for the HXC-193 + HXC-195 CLUSTER. Those two
were filed separately but share one root cause (the service reads no environment
config, which makes the port knob inert AND the health check unreachable), so they
got ONE agent rather than two: parallel dispatch is for INDEPENDENT domains, and
related failures where one fix resolves both must not be split across agents that
would then contend on the same files.

### HXC-168 ESCALATED — reachability established, exposure is ~9 months old

The item said reachability "has not been established either way and should not be
assumed harmless." It is now ESTABLISHED, and the answer is the bad one.

**Exposed, not local-only.** `docs/distribution/docker-compose.mistborn.yml:31-39`
publishes postgres `"5432:5432"` with no IP prefix — Docker short syntax, so it binds
0.0.0.0 — carrying `POSTGRES_DB: helixcode_prod` on a SEPARATE network-reachable host.
The documented SSH tunnel is a convenience for the dev host; it does not restrict the
bind. ~13 other compose files follow the same pattern.

**Not a Docker default nobody knew about.** `helix_code/docker-compose.builder.yml:75`
uses `127.0.0.1:5432:5432`. Exactly one file restricts to loopback, which makes the
rest a pattern rather than an oversight. That contrast is the load-bearing evidence.

**Duration (measured, not estimated):**
  literal B — in history since `b80b1a38` 2025-11-02 — 271 days, 23 commits
  literal A — in history since `3af11059` 2025-11-07 — 266 days, 13 commits
Both present in the CURRENT tree (30 and 16 files respectively).

**Published everywhere.** Per-remote `git log -S` counts on all four remote mains were
reported identical to the local `--all` scan (13 / 23). I independently re-ran this for
`origin` and `github` and got exactly 13/23 on both; the `gitlab` and `upstream` checks
timed out under 7-agent load, so those two rest on the agent's report, not on my own
execution. Stated precisely rather than rounded up to "I verified all four."

**This is a REPEAT, not a discovery.** `25d41351` (2026-06-04) is titled
"fix(security): remove hardcoded DB password from tracked SQL + sanitize .env.example
(Wave3 W3E, CONST-042)". A fix was attempted almost two months ago and did not finish
— which is exactly what the tracker item means by "the project's own earlier security
review already recorded it as something that must be replaced, and that has still not
happened." Worth noting against §11.4.214: a returning defect should reopen its
original item rather than enter as a fresh id.

**Rotation is the ONLY remedy.** The literals are in commits already pushed to every
configured upstream. Editing files does not remove published history, and a history
rewrite is forbidden outright by §11.4.113 — and would not help anyway, since the
remotes already carry it. This half CANNOT be closed by an agent.

### Stall diagnosis — contention RULED OUT (saves the next session the hunt)

Four agent failures now: HXC-168 and HXC-198 died on `API Error: Connection closed
mid-response`; HXC-205 and HXC-169 STALLED with "no progress for 600s, stream watchdog
did not recover". Four of a class is past the §11.4.102 Phase-4.5 threshold where the
right question stops being "retry?" and becomes "what is actually wrong?".

Measured at the moment of the fourth failure, rather than assumed:
  load average 5.08 / 5.05 / 4.14   — low
  memory 44G of 251G (18%)          — low
  threads 5572 of 262144            — low
So HOST CONTENTION IS NOT THE CAUSE. The host was close to idle while agents stalled.
The failures sit at the API/stream layer, which is infrastructure I do not control.

Operationally that matters because it rules out the tempting wrong fix: throttling
concurrency would cost parallelism and change nothing. §11.4.147 respawn is the correct
response, and every respawn now carries explicit `-timeout` on go commands,
package-scoped test invocations, and output-to-file rather than streaming — mitigations
against long-blocking calls, not against load.

### HXC-169 — the lead its stalled agent died holding (potentially decisive)

Final words before the stall: *"Zero importers, and those two lines are the only copy
call sites in the entire tree. Let me confirm with the Go toolchain rather than grep."*

If that holds, it resolves the item favourably: the sandbox code calls the vulnerable
copy functions, but NOTHING calls the sandbox code — "vulnerable but unreachable"
rather than "vulnerable". Note this does NOT contradict the item's claim that an
automated analysis traced calls from our sandbox code to the affected functions. Both
are true at once, and that reconciliation IS the answer.

Two traps recorded for whoever finishes it:
  §11.4.124 — "zero importers ⇒ unreachable" is a GUESS until reflection, build tags,
  codegen, DI/plugin registries and init() side effects are each ruled out with the Go
  toolchain rather than grep. The predecessor knew this; it stalled mid-sentence
  saying so.
  An unreachability verdict MUST ship with a guard that FAILS when something becomes an
  importer. Otherwise the finding silently rots the day someone wires it up.

### Wave 2 (2026-07-31) — 10 items closed, zero Criticals open

Closed with captured evidence: HXC-169, 193, 194, 195, 198, 201, 205, 209, 210, 212.
Five had NO docs/qa directory, so each got a FRESH gate run captured to disk before
closing — the close tool takes an `--evidence` path, and pointing it at prose or a
non-resolving path would be the exact bluff §11.4.5 exists to stop.

HXC-168 deliberately LEFT OPEN: code half done and guard proven both directions, but
the credential is in 23 commits across 271 days on every upstream and rotation is
operator action. Closing on the code half would misrepresent a live exposure.

Remaining: 30 open (0 Critical / 3 High / 8 Medium / 12 Low / 7 unset), 379 closed.

### Agent attrition — SIX failures, and what survives them

| Agent | Mode | Work state when it died |
|---|---|---|
| HXC-168 | API disconnect | tree clean — nothing written |
| HXC-198 | API disconnect | tree clean — nothing written |
| HXC-205 | stall 600s | tree clean — nothing written |
| HXC-169 | stall 600s | tree clean — held a decisive lead |
| HXC-213 | stall 600s mid-mutation | **fix complete in tree, tests passing** |
| HXC-211 | stall 600s | **fix complete in tree, tests passing** |

**Contention is ruled out** — measured at the fourth failure: load 5.08, memory 18%,
threads 5572/262144. The host was near-idle. These sit at the API/stream layer.

**The load-bearing finding: the WORK survives the stall.** For 213 and 211 the tree
held a coherent fix that compiled and passed its own guards. What is lost is the
REPORT and the COMMIT, not the engineering. So the correct response is respawn-to-
FINISH ("your work is in the tree, do not redo it, verify and complete"), not
respawn-to-redo — which would waste the work and risk two agents editing one file.

**Two residue false-alarms I chased and cleared, recorded so the next reader doesn't:**
- `- mutex sync.RWMutex` appeared as a DELETION in the deployment diff. It was not
  deleted; the line changed because a doc comment was added above it. Lock sites went
  2 → 50, which is the fix.
- `MUTATED-BY-CALLER` matched the residue grep in a test file. It is a deliberate test
  assertion proving a caller cannot corrupt manager state through a returned pointer —
  the aliasing guard working, not leftover mutation.
Both looked exactly like §11.4.84 residue at grep distance. Only reading the diff
separated them.

### Standing warnings now injected into every dispatch
- A mutation whose verification step REBUILDS the artifact cannot be reverted from a
  file backup — `npm test` runs `npm run build`, so restoring `src/` leaves `dist/`
  contaminated. Restore from `git checkout --`. (Cost me one red sweep; the HXC-209
  agent hit the same class via a relative path in a trap after a cd.)
- G7 is ENFORCING: every fix commit needs `docs/qa/<run-id>/`. Two commits shipped
  without one today.

### CORRECTION to my own headroom method — `ulimit -u` is NOT the binding limit here

Every §12.12 headroom check I ran this session read `ulimit -u` (262144) and
concluded "plenty of headroom". **That number does not bind on this host.** The
cgroup hierarchy caps it far lower, and the LOWEST value in the chain wins:

```
max      /user.slice
678505   /user.slice/user-1000.slice
max      /user.slice/user-1000.slice/user@1000.service
max      .../tmxw.slice
4096     .../tmxw.slice/tmxw-helix\x2dcode\x2d6339.slice   <-- BINDS
308411   .../run-<scope>.scope
```

So the real ceiling is **4096**, not 262144 — a 64x overestimate, and neither the
leaf scope nor the rlimit reveals it. The HXC-215 agent found this from the other
direction: `fork/exec …/compile: resource temporarily unavailable` (EAGAIN) in the
preserved gate logs.

**How to check headroom correctly:** walk every level of
`/proc/self/cgroup`'s path, read `pids.max` at each, take the MINIMUM, and compare
against that level's `pids.current`. `ulimit -u` alone is misleading.

Honest boundary: this does NOT overturn the earlier "contention ruled out" finding
for the STALLS — load 5.08 and memory 18% were real, and stalls are an API/stream
failure. But the process ceiling was a genuine pressure I never measured, and the
EAGAIN failures are consistent with hitting it.

### HXC-215 — three defects, and the one that was hiding the others

Root cause (FACT): the gate asserted `if rc -eq 0 then COMPILES else DOES-NOT-COMPILE`.
`go test ./...` also exits non-zero when the TOOLCHAIN dies, so an infrastructure
failure was indistinguishable from a compile error. Confirmed three ways: the two
reports contradict each other on the same commit; `92840ade` touches only
`tests/e2e/mocks/**` yet was blamed for `internal/rules` (causally impossible); and
zero compiler diagnostics appear in either log.

A second defect MANUFACTURED the evidence the first misread: a trap bug deleted the
build workdir mid-run, producing `chdir …: no such file or directory`. Worse, the old
trap form ran cleanup, RESUMED, and **exited 0** — a signal-killed gate reporting
SUCCESS.

A third was found only because the fix exposed it: the built-in self-test pins had
BIT-ROTTED (`go.mod 9c9b5912` vs HEAD's `4960895d`) and failed in 0s. The old gate
could not see this — it would have "passed" its known-bad half while the compiler
never ran. So the gate's own falsifiability proof was itself a bluff. Falsifiability
is now carried by a synthetic wolf test built from HEAD via `git archive`:
c2 COMPILES, c3 DOES-NOT-COMPILE, blamed set exactly `{c3}`, real diagnostic, exit 1.

Verified after the fix: `non-compiling: 0`, one infra failure reported as exit 3
INCONCLUSIVE with "not a PASS for those, and not an accusation against them".
Against the REAL flake logs: 2/2 exonerated, 0 false accusations. 5/5 identical
verdicts on a pinned range. Classifier 9/9 fixtures, both §1.1 mutations FAIL.
