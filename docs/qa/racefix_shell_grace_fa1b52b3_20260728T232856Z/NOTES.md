# What this run certifies

`fa1b52b3` corrected the drain grace introduced by `a7d8dbb5` (HXC-184). That
grace was a TOTAL budget: once the direct child was reaped, `ExecuteStream`
waited a fixed window and then tore the readers down. The reasoning behind it
("at most a pipe's worth of already-buffered data, milliseconds of work") held
only when the CONSUMER kept up, not when the kernel did — a command with no
grandchild can exit instantly with a full pipe, so a consumer following the
documented concurrent-drain contract at ~ms/line was cut off mid-stream. That
is output loss on a perfectly healthy command.

The fix makes the grace a NO-PROGRESS window: `OutputStreamer` counts completed
sends (`Progress()`), and the drain loop rearms its timer whenever delivery
advanced during the window, so only a FULL window with nothing moving counts as
stuck. It also stops deriving `OutputIncomplete` from which `select` branch won
(a timer/Done tie could flag a complete run as truncated) and stops letting a
teardown sentinel on stdout mask a genuine stderr failure.

Captured here (real command execution; `transcripts/` holds every byte):

* all four mechanisms are present in the tracked source that ships today —
  the `Progress()` counter, the rearm, the derived flag, the error priority;
* the commit's own I1 guard RAN and passed twice under `-race` (asserted on its
  `--- PASS:` line, not merely on exit 0);
* the three HXC-184 guards this fix had to preserve — grandchild-held pipes
  bounded, the `MaxConcurrent=1` executor slot recovered, Done-before-drain
  bounded rather than deadlocked — all still pass, so the rearm did not
  resurrect the wedge it was constrained by;
* RED_MODE=1 on this same fixed artifact FAILS both HXC-184 guards, which is
  what proves those guards can still detect the defect (§11.4.115 polarity —
  a guard that passes in both polarities certifies nothing);
* the whole package is race-clean.

Not certified: the exact throughput figures quoted in the commit message. This
run asserts the guard's OWN pass/fail contract, which is what the guard was
written to enforce; it does not independently re-measure line counts.
