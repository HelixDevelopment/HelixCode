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
