# HelixCode Constitution — inheritance pointer (NOT a source of truth)

> **This file is a POINTER, not a constitution.** It exists because GitHub SpecKit
> requires a file at this path and reads it during `/speckit-plan`'s
> "Constitution Check" gate. It deliberately carries **no principles of its own**.
>
> Authoring principles here would create a **third** constitution in this
> repository, which **CONST-059** (canonical-root inheritance clarity) forbids.
> Tracked as risk **R-23** in
> `docs/research/fully_incorporate_the_helixskills_system_into_helixcode_heli_20260728T192622Z_2557683/06_risk_register.md`.

## INHERITED FROM constitution/Constitution.md

The canonical governance corpus for this repository is, in precedence order:

1. **`constitution/Constitution.md`** — the HelixConstitution submodule. The single
   source of truth for every universal rule (§11.4.x anchors, CONST-0NN mandates).
   Its sibling carriers `constitution/{CLAUDE,AGENTS,QWEN,GEMINI}.md` are the
   five-carrier lockstep set (§11.4.157).
2. **`<repo-root>/CLAUDE.md`** (and `AGENTS.md` / `QWEN.md` / `GEMINI.md` /
   `CRUSH.md`) — HelixCode's consumer extensions. Project-specific rules only;
   they extend and never weaken the canonical root (§11.4.35 / CONST-059).
3. **This file** — a pointer. Zero independent authority.

Where this file appears to disagree with either of the above, **both of the above
win**. There is no case in which a rule stated only here is binding.

## What the SpecKit "Constitution Check" gate MUST evaluate against

When `/speckit-plan` reaches its Constitution Check gate, it evaluates the plan
against the anchors in `constitution/Constitution.md` — **not** against this file.
The anchors that gate essentially every plan in this repository:

| Anchor | What it gates |
|---|---|
| §11.4 / §11.4.1 / §11.4.123 | Anti-bluff: no PASS without captured runtime evidence |
| §11.4.6 | No guessing — `UNCONFIRMED:` / `UNKNOWN:` rather than plausible fill-in |
| §11.4.108 | Four-layer verification; a change is done only when its **runtime signature** verifies on a clean target — never a source grep |
| §11.4.169 | Mandatory test-type coverage (13 enumerated types) |
| §11.4.125 / §11.4.134 / §11.4.142 | Independent code review, iterated to a zero-finding GO |
| §11.4.224 | Test-first (TDD) for all work; ≥ 85 % coverage floor |
| §11.4.74 / §11.4.28 | Reuse-before-rewrite; owned submodules stay project-agnostic |
| CONST-051(C) / §11.4.28(C) | No nested own-org submodule chains |
| §11.4.65 / §11.4.212 | Four-format doc exports; every doc reachable from README |
| §1.1 | Every gate ships a paired mutation that makes it FAIL |

## Do not "fill this in"

Running `/speckit-constitution` against this file would overwrite this pointer with
generated principles and re-open R-23. If a SpecKit workflow demands a populated
constitution, the correct action is to point it at `constitution/Constitution.md`,
never to author principles here.

Planned mechanical enforcement (see
`specs/001-helixskills-incorporation/tasks.md`, task **T-P1.06**): gate
`CM-CANONICAL-ROOT-CLARITY` extended to assert this file opens with the
`## INHERITED FROM constitution/Constitution.md` heading; paired §1.1 mutation
strips the heading → gate MUST fail.

---

| Field | Value |
|---|---|
| **Version** | 1.0.0 |
| **Ratified** | 2026-07-29 |
| **Last amended** | 2026-07-29 |
| **Authority** | none — pointer only (CONST-059) |
| **Canonical root** | `constitution/Constitution.md` |
| **Replaces** | SpecKit stock template (2 346 B placeholder installed by `specify init`, commit `019dcc86`). The unmodified template remains at `.specify/templates/constitution-template.md`. |
