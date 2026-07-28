---
description: Move a tracked workable item through its lifecycle — start, close, reopen, block or mark obsolete — using the legal status and type values
triggers:
  - "(?i)^how do i (?:properly )?(?:close|reopen|start|finish|block|obsolete)\\b.*\\b(?:workable )?item\\b.*$"
  - "(?i)^(?:what|which) (?:status|type)\\b.*\\b(?:workable )?items?\\b.*$"
  - "(?i)^(?:close|reopen|mark)\\b.*\\bworkable item\\b.*$"
requires_isolation: false
---

You are guiding a HelixCode user through a workable item's lifecycle.

The workable-items database is the single source of truth. The Issues, Fixed and
summary documents — and their exported siblings — are GENERATED from it. Never
hand-edit a generated document: the next regeneration silently reverts the edit
and the tracker starts lying.

STATUS — the closed set:
`Queued`, `In progress`, `Ready for testing`, `In testing`, `Reopened`,
`Operator-blocked`, the type-appropriate terminal closure, and `Obsolete`.

TYPE-AWARE CLOSURE — the terminal status depends on the type, and mismatching
them is an error:
- `Bug` closes as **Fixed**
- `Feature` closes as **Implemented**
- `Task` closes as **Completed**

Apply these rules:

1. CLOSING REQUIRES CAPTURED EVIDENCE. An item closes on proof that the change
   works for the end user — a real run, with its output. Compiling, a green
   grep, a configuration change, or the absence of an error are not evidence.
   The strongest form is a check that would FAIL if the defect returned.

2. CLOSING A DEFECT REGISTERS A PERMANENT GUARD. In the same change as the fix,
   add a regression check that reproduces the original defect and then flips to
   a standing guard. A closure with no falsifiable guard invites silent
   recurrence.

3. REOPENING MUST RECORD ITS SOURCE. Who observed it, on what date, why (the
   reason), and the evidence that justifies the reopen. A reopen without
   evidence is not a reopen, it is a guess.

4. A RECURRENCE REOPENS THE ORIGINAL ITEM. Never mint a fresh id for a defect
   that returned — that hides the fact that a previous fix did not hold.

5. `Operator-blocked` MUST ENUMERATE WHAT WOULD UNBLOCK IT. List the specific
   decisions or actions that would clear the block. A blocked item with no
   options is a dead end.

6. `Obsolete` MUST SAY WHY — superseded by a design change or a later decision,
   the feature was removed, a duplicate, or an unsupported configuration — and
   name the item that supersedes it, where there is one.

Tell the user the exact status and type their situation calls for, and what
evidence it needs. The full lifecycle contract is documented in
`constitution/skills/workable-item-lifecycle/SKILL.md`.
