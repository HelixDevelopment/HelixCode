# HXC-213 — raw first-attempt capture (SUPERSEDED)

> **The complete evidence for HXC-213 is
> [`../hxc213_deployer_status_race_20260731T070422Z/`](../hxc213_deployer_status_race_20260731T070422Z/).
> Read that one.** This directory is the raw, untrimmed output of the first
> agent, which stalled before finishing. It is kept, not deleted, because it
> holds the genuine unmodified RED baseline capture.

## Why two directories exist

The first agent fixed the defect and stalled mid-paired-mutation — its last
words were *"Mutation 1 confirmed. Restoring and running mutation 2."* Its work
survived in the working tree and was verified intact, so a second agent was
dispatched to **finish** rather than redo it. The second agent produced the
curated set next door.

## What this directory has, and what it lacks

| | here (`063405Z`) | complete (`070422Z`) |
|---|---|---|
| RED baseline | ✅ raw, 820 KB | ✅ trimmed |
| GREEN package run | ✅ raw, 1.2 MB | ✅ |
| golden-bad self-validation | ✅ | ✅ |
| **paired mutation 1** (lock stripped) | ❌ | ✅ with restore |
| **paired mutation 2** (shallow copy) | ❌ | ✅ with restore |
| README / provenance | ❌ | ✅ |

Mutation 2 is the one that matters most: it breaks deep-copy *depth* rather
than swapping the pointer, so a weaker guard asserting only
`status != pd.status` would have passed it. That proof exists only next door.

## Why these files were nearly lost

Every file here is a `.log`, and `.gitignore:18` carried a blanket `*.log`. All
six were therefore invisible to git — this directory was cited by HXC-213's
closure while containing, from a fresh clone's perspective, nothing at all.

That was not a local slip: **38 evidence directories** were affected, three of
them backing items closed on 2026-07-31. The rule was corrected at source in
`7552c7bd` with `!docs/qa/**/*.log`, verified not to un-ignore ordinary logs
elsewhere. The second agent had independently dodged the same trap by naming
its files `.txt` — the workaround that had been quietly keeping the convention
alive (1445 committed `.txt`, 0 committed `.log`).
