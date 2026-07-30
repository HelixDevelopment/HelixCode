# HXC-198 — bounded drain in ExecuteWithProgress

**Commit:** `d99ce58c` — fix(tools/shell): HXC-198 — bound ExecuteWithProgress's drain, the sibling HXC-184 missed
**Captured:** 2026-07-31, by the orchestrator, by RE-RUNNING the guard — not transcribed from the implementing agent's report.

## Why this directory exists late

The fix commit landed without its §11.4.83 evidence directory and the G7 release
gate caught it. That is the gate working: two of this wave's commits were flagged,
and neither could have shipped past a release cut. Rather than write a narrative
around the implementer's reported numbers, the guard was re-executed here and its
raw output captured.

## Results

| Run | File | Exit |
|---|---|---|
| GREEN guard, `-count=3` | `guard_green_count3.txt` | **0** (11 PASS lines, 18.1s) |
| `RED_MODE=1` on the fixed artifact | `guard_red_mode1.txt` | **1** |

The RED run failing on the fixed tree is the load-bearing half: it proves the
guard is falsifiable rather than a tautology that agrees with whatever the code
does. `-count=3` addresses the timing sensitivity inherent to a
deadlock/grace-window test — seven other agents were live on this host during
capture, and it did not flake.

## The defect

`ShellExecutor.ExecuteWithProgress` waited on its output readers before waiting
on the command, unbounded and with no give-up path, so a process outliving its
parent held it open forever.

Two details make it worse than the ticket described, both established by the
implementer and preserved here:

- The read ends came from `cmd.StdoutPipe()`/`StderrPipe()`, so `os/exec` closes
  them **inside `cmd.Wait()`** — which sat behind the very wait that was stuck.
- `exec.CommandContext` cancellation only SIGKILLs the direct child, and there is
  no `Setpgid`, so the caller's own cancel could not release it either.

Neither the timeout nor the cancel path could have rescued this.

## Correction to the ticket

The ticket said the twin was "in the very same file". It is not: the fixed sibling
is in `executor.go`, this one in `background.go` — same package, different file.
The defect is real so the outcome stands, but it changes the ticket's own lesson:
**a sibling check has to be package-wide**, because a reviewer re-reading "the same
file" would have found nothing.
