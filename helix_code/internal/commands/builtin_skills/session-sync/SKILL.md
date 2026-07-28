---
description: Sync session transcripts, memories, settings and handoff documents between this machine and the same project on a remote host
triggers:
  - "(?i)^(?:sync|pull|push) (?:my |the )?sessions?\\b.*\\b(?:remote|host|machine|laptop|workstation)\\b.*$"
  - "(?i)^(?:sync|pull|push) (?:my |the )?(?:session|memories|transcripts|handoff)\\b.*\\bfrom\\b.*$"
requires_isolation: false
---

You are helping a HelixCode user sync session state with the same project on
another machine.

Session sync moves the project's agent-side state — memories, session
transcripts, settings, agent definitions and handoff documents — between a
remote host and this one. It supports pull (remote to local), push (local to
remote) and bidirectional sync.

Before running anything, establish these facts rather than assuming them:

1. DIRECTION. Pull, push, or bidirectional. Never guess: pushing when the user
   meant pull can overwrite newer local state.

2. THE REMOTE. Which host, which user, and the path to the SAME project there.
   Sync is defined between two checkouts of one project; it is not a general
   file copy.

3. WHAT IS IN SCOPE. Session transcripts and memories are the point. Anything
   holding credentials is not: never sync secrets, tokens, key material or
   `.env` files between machines as part of this operation.

4. CONFLICTS ARE SURFACED, NOT SILENTLY MERGED. When both sides changed, report
   the divergence and let the user decide. Silently picking a winner destroys
   whichever side lost.

5. THE OPERATION MUST BE REVERSIBLE. Confirm a backup or a dry run before any
   step that overwrites existing local state, and prefer showing what WOULD
   change first.

Report afterwards what actually transferred — which files, in which direction —
rather than reporting success generically. If the remote was unreachable, say
so; an unreachable remote is not a completed sync.

The sync implementation and its full contract live at
`constitution/skills/session-sync/` alongside the executable helper.
