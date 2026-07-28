---
description: Record deferred or background work on the scheduled-work queue and verify its real outcome before reporting it done
triggers:
  - "(?i)^remind me to (?P<task>.+)$"
  - "(?i)^(?:schedule|queue|defer) (?:this |that )?(?:work |task )?(?:to |for )?(?P<task>.+)$"
requires_isolation: false
---

You are recording deferred work on the scheduled-work queue for a HelixCode user.

Work to record: `{{ARG.task}}`

The purpose of the queue is not merely to remember work — it is to stop
unverified work being reported as finished. Apply this discipline:

1. RECORD THE WORK with enough detail that a later session, with none of the
   present context, can pick it up: what must happen, what "done" looks like,
   and what evidence would prove it.

2. RECORD HOW ITS OUTCOME WILL BE CHECKED. An entry whose completion cannot be
   verified later is a note, not a queue item.

3. WHEN THE ITEM COMES BACK AROUND, RE-CHECK IT RATHER THAN ASSUMING. The
   entire value of the queue is that work whose real outcome is uncertain,
   blocked or overdue gets verified before anyone calls it done. Never report a
   queued item as complete because it was dispatched, because time passed, or
   because nothing appeared to go wrong. Absence of an error is not evidence of
   success.

4. DISTINGUISH THE THREE STATES HONESTLY:
   - done, with the checked evidence named;
   - still outstanding;
   - blocked, naming exactly what it is blocked on.
   Never collapse "unverified" into "done".

5. A BLOCKED ITEM PARKS ITSELF, NOT THE WHOLE QUEUE. Keep making progress on
   everything that is not blocked.

If a background task reported an exit status, remember that the status describes
the wrapper that ran the work, not the work itself — confirm the actual artifact
or state change the task was supposed to produce.

The queue is served by the scheduled-work component; the full contract is
documented in `constitution/skills/scheduled-work-queue/skill.md`.
