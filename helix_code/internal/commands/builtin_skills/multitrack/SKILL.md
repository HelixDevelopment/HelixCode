---
description: Set up or reason about parallel multitrack work streams, their branches, isolation and identity labels
triggers:
  - "(?i)^(?:how do i |how to )?(?:start|create|spin up|set up|begin)\\b.*\\bmultitrack\\b.*$"
  - "(?i)^.*\\bmultitrack\\b.*\\b(?:work ?stream|track|branch|worktree)s?\\b.*$"
requires_isolation: false
---

You are helping a HelixCode user run parallel multitrack work streams.

Multitrack development runs several isolated checkouts at once, each on its own
branch, so independent work proceeds in parallel without the streams colliding.
Apply these invariants:

1. ISOLATION BY CONSTRUCTION. Each track is its own checkout with its OWN
   repository metadata directory, not a shared one. A shared metadata directory
   is a single point of failure: one stale lock stalls every track at once, and
   one corrupted object store loses all of them. Where disk space is the
   constraint, use a copy-on-write or reflink clone so extents are shared on
   disk while each track keeps its own independent metadata.

2. ONE FEATURE, ONE BRANCH NAME. A feature or logical group of work items maps
   to exactly ONE canonical branch name, used identically in the main
   repository and every submodule it touches. Never mint a second name for work
   that already has one, and never let a submodule's branch name drift from the
   main repository's.

3. TRACK-QUALIFIED IDENTITY. Never identify a track by the bare directory
   basename. Two checkouts that share a basename collapse into the same session
   name, lock path and log directory, and then cross-wire. Every session name,
   lock path, log directory and resource claim carries the track-qualified key.

4. EXCLUSIVE RESOURCES HAVE ONE OWNER. When tracks share a device, sink or any
   single-access resource, exactly one track drives it at a time; the others
   observe read-only. Concurrent drivers produce cross-contaminated results
   that cannot be trusted.

5. MERGE THE TRUNK IN OFTEN. Merge the canonical trunk INTO each long-lived
   branch regularly rather than only at the end, so the eventual integration
   stays small. Merge, never rebase, so no commit is lost.

Tell the user which of these applies to their situation and what to do next. If
you do not know this project's actual track layout, say so and ask rather than
assuming a number of tracks or a directory scheme. The orchestration scripts and
the full contract live under `constitution/scripts/multitrack/`.
