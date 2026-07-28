# fixed_h2_pipe_row_parity_gate.sh

**Revision:** 2
**Last modified:** 2026-07-29T00:00:00Z
**Gate ID:** `CM-FIXED-SUMMARY-ITEM-VISIBILITY` (was `CM-FIXED-H2-PIPE-ROW-PARITY`)

## Overview

§11.4.135 standing regression guard for the §11.4.90 / §11.4.91 / §11.4.53
docs-tooling drift that hid **HXC-044** (`Bug | Obsolete (→ Fixed.md)`) from
`docs/Fixed_Summary.md`.

`docs/Fixed.md` is MIXED: a pipe table
(`| Closure | Title | Type | Status | Round | Commit(s) | Evidence |`) **AND**
H2 detail sections (`## HXC/ATM-NNN — …`).

> **The file name is now historical.** It names the *mechanism* this guard used
> before the 2026-07-29 reconciliation below, not what it asserts. The path is
> retained deliberately — `docs/Issues.md`, `docs/Issues_Summary.md` and this
> document reference it, and tracker-doc regeneration is operator-deferred, so a
> rename would strand those references. Renaming is tracked as follow-up, never
> done silently (§11.4.124).

## §11.4.120 reconciliation (2026-07-29, HXC-161)

**Original invariant.** The guard asserted "every H2 closure heading has a
matching pipe-table row" as a **proxy** for what actually mattered — *no closed
item is missing from `Fixed_Summary.md`*. Under the then-canonical generator
`scripts/generate_fixed_summary.sh`, which read **only** the pipe table, the
proxy was sound: having a pipe row was exactly what made an item visible.

**What changed — the consumer, not the data.** Commit `8494380a` (2026-07-29):

- `scripts/generate_fixed_summary.sh` is **superseded and refuses to run**
  (exits 2 in every mode, writes nothing) after a hand invocation at `0a4df699`
  rewrote the closed-item tally 344 → 188, dropping 156 items of tracked state.
- `docs/Fixed_Summary.md` is now **derived from `docs/workable_items.db`** by the
  constitution submodule's Go exporter (`renderFixedSummary`, `export.go`), which
  selects every item with `current_location == 'Fixed'` and is
  **representation-agnostic** — a `section` item and a `table` item land in the
  summary identically.

**Measured consequence** (HEAD `bbd236c9`): 56 H2 closure sections have no pipe
row, and **all 56 are present in `Fixed_Summary.md`**. The invisibility the proxy
existed to prevent no longer occurs; the proxy had become a standing 57-failure
red asserting an obsolete era's mechanism. Per §11.4.120 the gate was
**reconciled** to the new mechanism — never fake-passed, never reverted-against.

**Why this is a strengthening, not a loosening.** Against `0a4df699`, the real
data-loss commit (summary corrupted down to 20 lines):

| | summary-visibility failures reported |
|---|---|
| retired pipe-row invariant | **0** — it never noticed (it exited 1, but only on pipe-row noise: a 57-failure standing red is exactly how a real failure gets masked) |
| reconciled invariant | **246** closed items flagged missing |

## What it asserts

- **(A)** every `docs/Fixed.md` H2 closure heading (`## HXC-NNN` / `## ATM-NNN`)
  appears in `docs/Fixed_Summary.md`. **Hard FAIL.**
- **(B)** every `docs/Fixed.md` pipe-table row whose Title cell carries a ticket
  id appears in `docs/Fixed_Summary.md`. **Hard FAIL.**
- **(C)** every `Obsolete (→ Fixed.md)` pipe-table item appears in
  `docs/Fixed_Summary.md`. **Hard FAIL.** A strict subset of (B), retained as an
  explicitly-named check because it is the original HXC-044 forensic anchor, so
  an Obsolete-specific regression is reported as such.

`Fixed.md` and `Fixed_Summary.md` are rendered by **different code paths**
(a `doc_segments` walk vs. `renderFixedSummary`'s item select), so agreement
between them is a real, falsifiable cross-check — not a tautology, as `0a4df699`
proved by breaking it.

## Honest boundary (§11.4.6 / §11.4.118)

- Ticket-id matching is **bounded** (non-alphanumeric on both sides) so `ATM-97`
  is not spuriously satisfied by `ATM-970`.
- Legacy pipe rows whose Title cell carries **no ticket id** (the 2026-05-19
  i18n-round rows, `Tracker HTML + PDF exports per §11.4.19`, …) cannot be keyed
  by id and are **out of scope** for (B). Their count is **printed on every run**
  so the exclusion is visible, not hidden — 87 at HEAD.
- This is a **document-parity** guard (`Fixed.md` ↔ `Fixed_Summary.md`) and reads
  no database. It proves the two rendered documents agree, **not** that either
  matches the SQLite SSoT — that md⟷db direction belongs to
  `CM-WORKABLE-ITEMS-MD-DB-IN-SYNC` and `CM-SUMMARY-SYNC`, both still required.
- The invariant is **architecture-current, not retroactively green**: it also
  fails against `a665b02d`, because that era's summary was aggregate-counts-only
  and listed exactly one item id. That is a true property of the old document
  shape, not a false positive. The invariant became satisfiable at `8494380a`,
  when the summary gained its DB-derived per-item table.
- An **anti-tautology tripwire** refuses a vacuous PASS: if the extractors match
  0 H2 sections *and* 0 ticket-id pipe rows (format drift), the gate FAILs rather
  than reporting a green it did not earn.

## Prerequisites

`awk`, `grep`, `mktemp`, `bash`. No network, no credentials, no database.

## Usage

```bash
scripts/gates/fixed_h2_pipe_row_parity_gate.sh
# RED-baseline (§11.4.115) — reproduce the defect on temp copies and assert the
# guard genuinely catches it (exit 0 = guard is not blind):
RED_MODE=1 scripts/gates/fixed_h2_pipe_row_parity_gate.sh
```

`FIXED_PARITY_STRICT` is **gone** — it toggled the retired pipe-row invariant
between WARN and FAIL. All three current invariants are unconditional hard FAILs.

## Edge cases

- `RED_MODE=1` operates on copies under `$TMPDIR` (cleaned on `EXIT`); it never
  mutates the live `docs/Fixed.md` / `docs/Fixed_Summary.md`.
- `RED_MODE=1` deletes **two** victims covering **both** representations —
  `HXC-044` (Obsolete, has a pipe row → trips (B)+(C)) and `HXC-157`
  (section-only, no pipe row → trips (A)). `HXC-157` is the load-bearing one:
  under the retired invariant its disappearance from the summary was completely
  invisible. A future edit that silently narrows the guard to one representation
  is caught here.

## Internal behaviour

Parses pipe data rows by column (col1 = Closure date, col2 = Title, col3 = Type,
col4 = Status, …; awk fields shift by one because each line starts with `|`),
extracts the leading `(HXC|ATM)-<id>` token from Title cells and H2 headings, and
bounded-matches each id against `docs/Fixed_Summary.md`.

## Related scripts

- `constitution/scripts/workable-items` (`renderFixedSummary` in `export.go`) —
  the canonical DB-derived generator this guard now protects.
- `scripts/generate_fixed_summary.sh` — the **superseded** markdown-derived
  generator (refuses to run; retirement tracked as HXC-160).
- `scripts/gates/summary_sync_gate.sh`, `scripts/gates/workable_items_sync_gate.sh`
  — the md⟷db direction this guard deliberately does not cover.
- `scripts/gates/obsolete_details_gate.sh`, `scripts/gates/obsolete_colorize.sh`
  — §11.4.90 sibling gates.

## Last verified

2026-07-29 (HEAD `bbd236c9`) — GREEN (exit 0): 153 H2 sections, 99 ticket-id pipe
rows, 1 Obsolete item all present in `Fixed_Summary.md`; 87 legacy rows reported
out of scope. `RED_MODE=1` exit 0 (guard correctly FAILs on the broken artifact).
Mutation matrix, all exit 1: delete a section-only closure → (A) fires; delete a
pipe-row closure → (A)+(B)+(C) fire; empty the summary → 253 failures; empty
`Fixed.md` → anti-tautology tripwire fires.
