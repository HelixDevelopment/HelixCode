---
description: Turn a reported bug, defect, task or feature request into a properly tracked workable item instead of a prose acknowledgement
triggers:
  - "(?i)^report (?:a |an )?(?P<kind>bug|issue|defect|regression|task|feature)\\b[\\s:,-]+(?P<summary>.+)$"
  - "(?i)^(?:file|open|raise|log) (?:a |an )?(?P<kind>bug|issue|defect|ticket|task|feature)\\b[\\s:,-]+(?P<summary>.+)$"
requires_isolation: false
---

You are turning a report into a tracked workable item for a HelixCode user.

Reported kind: `{{ARG.kind}}`
Reported summary: `{{ARG.summary}}`

A report that is only acknowledged is a lost requirement. Answering "yes, that
looks like a bug, I will look into it" without creating a tracked item means the
requirement was accepted and then silently evaporated. Every report becomes a
real, fully populated tracked item — or it is explicitly and honestly declined.

Produce the item with all of the following, and do not leave any of them blank:

1. TYPE, from the closed set:
   - `Bug` — a product defect: a regression or user-visible broken behaviour.
   - `Task` — internal work: a refactor, documentation, infrastructure, a gate.
   - `Feature` — new user-visible capability.

2. STATUS — a new item opens as `Queued`.

3. A STABLE ID and a TITLE that names both the SUBJECT and the PROBLEM. A title
   must stand alone: a bare fragment like "Critical" or "Composes with" is not a
   title and will be rejected.

4. A COMPREHENSIVE, PLAIN-LANGUAGE DESCRIPTION aimed at a non-developer —
   several sentences covering WHAT it is, WHY it matters, HOW it manifests, HOW
   to reproduce it, and what measurable outcome closes it. Avoid jargon,
   unexpanded acronyms and code references as the primary explanation; put any
   technical detail after the plain-language part.

5. ATTRIBUTION: who created it, and who it is assigned to.

Before creating anything, CHECK WHETHER THE ITEM ALREADY EXISTS. If this report
is a recurrence of a defect that was previously closed, REOPEN the existing item
rather than minting a new identifier — a new id for a returning defect destroys
the history that shows the fix did not hold.

The single source of truth is the project's workable-items database; the Issues,
Fixed and summary documents are generated from it, so never hand-edit those.
The full procedure is documented in
`constitution/skills/reporting-workable-items/SKILL.md`.
