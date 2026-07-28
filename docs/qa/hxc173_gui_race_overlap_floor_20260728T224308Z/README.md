# HXC-173 — GUI thread-affinity guards reported FAIL when the host was merely busy

| | |
|---|---|
| Revision | 1 |
| Created | 2026-07-29 |
| Last modified | 2026-07-29 |
| Status | active |
| Item | HXC-173 (Bug / Medium) |
| Fix commit | `82313504` |
| Pre-fix artifact | `b78b531e` |
| Status summary | Fixed — FAIL-under-load reproduced, now a slow PASS under the same load; SKIP branch proven on both guards |

## Table of contents

- [What was wrong](#what-was-wrong)
- [The fix](#the-fix)
- [Evidence](#evidence)
- [Honest boundaries](#honest-boundaries)

## What was wrong

Three Fyne GUI thread-affinity guards drove their production workers for a
**fixed** window, counted renderer repaints, and called `t.Fatalf` below a floor
of 2:

| Site | Test |
|---|---|
| `aurora_os/gui_thread_race_test.go:377` | LLM tab |
| `aurora_os/gui_thread_race_test.go:481` | dashboard stats ticker — the site named on the item |
| `harmony_os/gui_thread_race_test.go:332` | the sibling (inline render loop, `renderCount`) |

Whether a fixed window clears that floor is a property of the **host**, not of
the code under test. Failing on it turns host load into a verdict about
production code — a §11.4.201 FAIL-bluff: the guard fires without the condition
it names being true. Per §11.4.3 an inconclusive run must SKIP with a reason.

The floor's **polarity is correct and is unchanged**. It is an anti-vacuity
check: "no races detected" from a renderer that never overlapped a worker is not
evidence about the code. The defect was the *verdict*, not the floor.

## The fix

All three sites now drive **adaptively** through `driveForOverlap`: bounded
rounds of *{start a render loop → run the round body → stop and **join** → read
that loop's counter}*, accumulating until the floor is met or a 30 s deadline
elapses. A slow host produces a slow PASS; only an exhausted deadline yields
SKIP.

**The repaint counter stays unsynchronized.** `newAuroraRenderLoop` documents
that its counter is deliberately race-free-by-construction — written only by the
render goroutine, read only after the join — because a shared atomic in that hot
path could create happens-before edges that *mask the race under test*. The
adaptive loop therefore never polls it mid-window: each round is the **same**
race window the single-shot version used, and rounds merely repeat it. The join
at a round *boundary* orders across **distinct** windows, never within one; what
crosses a boundary is the **count**, never synchronization.

**Real-defect assertions keep FAIL polarity and run before the overlap verdict.**
A ticker that never fired, a send path that never ran, or a chat history missing
worker output still FAIL on any host, however loaded. Only the overlap axis is an
inconclusiveness axis.

The chat tests *re-drive* their sends in extra rounds — repainting against
finished workers buys no overlap — while the ticker test extends its window,
since the production 1 s ticker keeps firing.

## Evidence

`-tags=ci` is required (Fyne non-GL driver; no X11/GL headers on this host). Load
is stated with every measurement per §11.4.201. The 1-minute load average lags a
short window, so the real-time contention signal quoted is the **runnable-task
count** from `/proc/loadavg`.

| Capture | File | Outcome | Exit | Contention |
|---|---|---|---|---|
| RED, pre-fix, under load | [`evidence/red_prefix_under_load.txt`](evidence/red_prefix_under_load.txt) | `renderer completed only 1 repaints`, `--- FAIL` | 1 | 178 runnable / 64 CPUs |
| Post-fix, **same load** | [`evidence/after_same_load_slow_pass.txt`](evidence/after_same_load_slow_pass.txt) | 2 adaptive rounds → `--- PASS` (35.08s) | 0 | 71 runnable |
| Post-fix, SKIP branch | [`evidence/after_skip_branch_under_load.txt`](evidence/after_skip_branch_under_load.txt) | `INCONCLUSIVE …` `--- SKIP` | 0 | 176 runnable |
| GREEN, unloaded, all 3 guards | [`evidence/green_unloaded.txt`](evidence/green_unloaded.txt) | 18 / 11 / 18 renders, all `--- PASS` | 0 | load 14.76 |
| Sibling: PASS under load + §1.1 SKIP mutation | [`evidence/harmony_sibling.txt`](evidence/harmony_sibling.txt) | `--- PASS` at load 77.96; `--- SKIP` under mutation | 0 | 183 runnable |

The decisive pair is rows 1 and 2: **the same guard, the same load recipe,
opposite verdicts** — FAIL before, slow PASS after, with the log showing it took
2 adaptive rounds to reach the floor the single-shot version missed.

Row 4 is what stops this being a guard that always skips: unloaded, every site
clears the floor several times over in a **single** round.

## Honest boundaries

- **The SKIP branch was reached via the deadline knob, not by starving the host
  for a full 30 s.** `GUI_RACE_OVERLAP_DEADLINE_MS` shortens the budget so the
  branch is demonstrable. It can only make the harness give up *sooner* — it
  cannot turn a FAIL into a PASS, nor a real data race into a clean run. It is a
  self-validation knob, and it is also the one new way a run could be downgraded
  to SKIP if someone set it in earnest; it defaults to 30 s and nothing in the
  tree sets it.
- **The harmony sibling's SKIP branch was proven by a §1.1 paired mutation**
  (floor 2 → 1000000, restored byte-identically), not by load. At load 77.96 with
  183 runnable it still cleared the floor and PASSed, because its round-0 window
  inherently spans many repaints. That is good behaviour, but it means load alone
  did not exercise its SKIP path.
- Fixing the verdict does not make these guards prove more about thread-affinity
  than they did before; it stops them making claims the host, not the code,
  determined.
- The load figures come from busy-loop workers I spawned and reaped myself,
  matched on a private marker (§11.4.174). Other agents were active on this host
  throughout, so the absolute numbers are not reproducible — the *relative*
  before/after under the same recipe is the load-bearing part.
