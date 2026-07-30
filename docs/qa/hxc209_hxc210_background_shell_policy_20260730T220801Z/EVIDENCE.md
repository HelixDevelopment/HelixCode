# HXC-209 (Critical) + HXC-210 — background shell path bypassed command-security policy and dropped the working directory

| Field | Value |
|---|---|
| Revision | 1 |
| Created | 2026-07-31 |
| Last modified | 2026-07-31 |
| Status | active |
| Items | HXC-209 (Bug, Critical), HXC-210 (Bug, High) |
| Sibling | `d99ce58c` (HXC-198) — same function, bounded-drain fix; guarded here against regression |

## Table of contents

- [What was wrong](#what-was-wrong)
- [Root cause](#root-cause)
- [What was changed](#what-was-changed)
- [Why not route through DefaultExecutor](#why-not-route-through-defaultexecutor)
- [Evidence index](#evidence-index)
- [Still open](#still-open)

## What was wrong

`ShellExecutor.ExecuteWithProgress` (`helix_code/internal/tools/shell/background.go`) is the
entry point every `run_in_background: true` shell call reaches, via
`ToolRegistry.adaptToolForBackground` (`registry.go:1222`) → `ShellTool.ExecuteWithProgress`
(`shell_tools.go:88`).

**HXC-209 (Critical).** It called no security validation at all. The three sibling entry points
each call `e.security.ValidateCommand` — `executor.go:199` (`Execute`), `:329` (`ExecuteAsync`),
`:357` (`ExecuteStream`). This one called nothing. Because the route is selected by an ordinary
caller-supplied boolean on the same tool, anyone able to request a shell command could request it
in the background and step around the blocklist, the allowlist, the dangerous-pattern screen and
the working-directory check together. The production registry wires `shell.DefaultConfig()`
(`registry.go:170`), so the live blocklist (`rm`, `dd`, `kill`, `shutdown`, `mkfs`, …) was the one
being bypassed — not a test-only policy.

**HXC-210 (High).** It read `params["cwd"]`. The published schema emits `workdir`
(`ShellTool.Schema`, `shell_tools.go:39`) and the synchronous path reads `workdir`
(`shell_tools.go:70`). The key sent was never the key read, and nothing reported it: no error, no
warning, no fallback notice. Commands ran wherever the server process happened to be. The hazard
is not failure — it is a command **succeeding against the wrong files**.

## Root cause

One cause, two symptoms: the function was written as a *parallel implementation* rather than as
another entry point onto `DefaultExecutor`. It built its `exec.Cmd` straight from the raw params
map, so every guarantee living in the shared policy was absent — not weaker, absent.

## What was changed

`ExecuteWithProgress` now reconstructs the same `*Command` the synchronous paths build, and hands
it to the **same** validator and the **same** command builder:

1. `workdir` (schema key) is read, with `cwd` retained as a fallback — it was undocumented and has
   zero in-tree producers, but it *did* take effect before this change, and §11.4.122 forbids
   quietly dropping a capability that worked.
2. `se.applyDefaults(spec)` — config-level defaults, exactly as the three siblings apply them.
3. `se.executor.security.ValidateCommand(spec)` — runs **before** `Start`, so a refused command is
   never executed.
4. Interpreter, argv, working directory and environment come from new shared helpers
   `commandArgv` / `applyCommandSpec`, extracted from `prepareCommand` as a pure refactor. This
   also retires the hardcoded `"sh"` that ignored `Command.Shell` and `Command.Args`.

`exec.CommandContext` is kept (rather than the shared `prepareCommand`'s `exec.Command`) because
the caller's context is this path's only stop signal.

## Why not route through DefaultExecutor

Routing `ExecuteWithProgress` through `DefaultExecutor.ExecuteStream` was evaluated and rejected on
evidence, not preference:

- **Concurrency budgets collide.** `ExecuteStream` acquires `e.semaphore`, sized `MaxConcurrent`
  (10 under `DefaultConfig`). Background work already has an independent budget of 64
  (`internal/workflow/background.go:250`). Routing through it would block or fail the 11th
  concurrent background task — a live regression well outside this defect.
- **The default timeout is wrong for this path.** `ExecuteStream` enforces `cmd.Timeout`
  (30s default). A background execution exists precisely to outlive that; enforcing it would kill
  the long-running work the entry point is for.
- **It would put the `d99ce58c` guard at risk.** That fix's drain accounting measures lines handed
  to the sink; `ExecuteStream`'s measures completed sends into a 100-buffered channel. Stacking a
  second drain loop underneath a 6s `graceDecisionDeadline` adds timing risk to the exact guard
  this work was told not to regress.

Both defects are **policy** defects, not **mechanism** defects. Sharing the policy components fixes
them at the root while leaving the carefully-tuned mechanism untouched — confirmed by the release
timing below, unchanged at one grace window.

## Evidence index

| File | Proves |
|---|---|
| `baseline_prefix_package.txt` | Package green before any change (exit 0) — contention ruled out per §11.4.201 |
| `guard_red_mode1_prefix.txt` | §11.4.115 RED on the **pre-fix** artifact: blocklisted `rm` **deleted its canary**, non-allowlisted `touch` **created its file**, `pwd` reported the process cwd instead of the requested dir |
| `guard_green_polarity_fails_on_prefix.txt` | The GREEN polarity **fails** on the pre-fix artifact (exit 1, 4 failures) — the guards are not tautologies |
| `guard_green_postfix.txt` | §11.4.115 GREEN post-fix: all 5 pass; refusals land **before** execution (canary survives) |
| `guard_e2e_real_dispatch_route_postfix.txt` | §11.4.108 layer 3 — enforcement holds through the real `ToolRegistry` route with the production config |
| `mutation_a_validatecommand_stripped.txt` | §1.1 paired mutation: strip `ValidateCommand` → e2e security guard **FAILs** (exit 1) |
| `mutation_b_workdir_key_dropped.txt` | §1.1 paired mutation: stop reading the schema `workdir` key → workdir guard **FAILs** (exit 1) |
| `d99ce58c_grandchild_guard_count3.txt` | HXC-198 bounded-drain guard: 12/12 PASS at `-count=3`, release still **2.01s** = exactly one grace window |
| `race_both_packages.txt` | `-race` clean on `internal/tools/shell` and `internal/tools` |

Both mutations were restored to the byte-identical pre-mutation artifact
(sha256 `1aa72fbae3d63d6d381ec95a5608f50d90c55fa6222658fd47db093868c276bb`) and the tree scanned
for residue before commit, per §11.4.84.

## Still open

Genuine remaining divergences between this path and `DefaultExecutor`, recorded rather than
silently implied fixed (§11.4.6). None is a security-policy gap:

- No sandbox / resource limits / `Setpgid` (so no process-group kill).
- No output-size cap — the aggregated `lines` slice is unbounded. `ExecuteStream` shares this gap;
  only `Execute` caps output, via `OutputCollector`.
- No signal/status registration, so `Kill`/`GetStatus`/`ListExecutions` can never target a
  background execution. Needs a caller-supplied execution ID, which the params map does not carry.
- `params["timeout"]` and `params["env"]` are still ignored (config-level `Env` now applies via
  `applyDefaults`). The timeout omission is deliberate, per the reasoning above.
