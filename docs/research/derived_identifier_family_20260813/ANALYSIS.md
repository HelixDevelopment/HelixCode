# §11.4.150 deep multi-angle research pass — derived-identifier family

**Covers:** HXC-272, HXC-281, HXC-268. **Produced:** HXC-326 (High).
**Conducted:** 2026-08-13, main stream (T1/main - claude4), in parallel with four
background review streams per §11.4.150(F).
**Sources verified:** 2026-08-13 (URLs and access date below, per §11.4.99).

---

## Why these three are one family

Each is an identifier that was *derived* rather than *guaranteed*, and turned out
not to be unique or not to mean what it claimed:

- **HXC-272** — two runs starting in the same second derive the same results
  folder name, so they share it, and the recent-run pointer freezes.
- **HXC-281** — guards numbered by a scheme that drifted, so the number no longer
  identified the guard it named.
- **HXC-268** — a mutation count that disagreed with itself across four published
  figures.

The shared shape: **an identifier computed from something that is not unique**
(a clock at second resolution; a position in a list; a recount), then trusted as
if it were.

---

## Angle 1 — is timestamp-as-identifier a recognised antipattern?

Yes, explicitly, and the failure mode is exactly HXC-272's:

> Whoever uses timestamps to generate expected unique identifiers will be exposed
> to the same issues with collisions. … A timestamp function will give the same
> number if called twice within one time unit.

Timestamps are described as *"non-unique serial identifiers that introduce the
issue of ties."* The standard remedy is not "use a finer clock" — it is to add a
component that is unique by construction: a per-unit **sequence number**, a
machine identifier, or entropy. UUIDv7 is the canonical modern form (millisecond
time component + a counter for events within the same millisecond + random bits).

Worth noting because it defeats the obvious cheap fix: collisions are common even
at **nanosecond** resolution on real hardware, because clock granularity is not
the same as clock resolution. Moving `%H%M%S` to milliseconds narrows the window
without closing it. Only a sequence or entropy component actually closes it.

## Angle 2 — measure our own corpus

Swept for second-granularity timestamps used to compose a path or identifier.

- **1158** second-granularity timestamp sites overall (most are log lines and
  display strings — harmless, and the reason the raw count is a lead, not a
  finding).
- **~14** compose a path or identifier and are therefore collision-capable.

They fall into three consequence classes, and the ranking matters more than the
count:

| class | example | consequence of a collision |
|---|---|---|
| results/report dirs | `challenges/results/<suite>/%Y%m%d_%H%M%S` | two runs share output — HXC-272's own class |
| session ids | `MON_SESSION_ID="${name}_%Y%m%d_%H%M%S"` | two sessions merge in reporting |
| **backup dirs/files** | `configPath + ".backup." + 20060102-150405` | **the second backup overwrites the first** |

The third class is materially worse than the one HXC-272 describes. A shared
results folder mixes output; a colliding **backup** destroys the thing that
exists to be recoverable (§9.2). `os.WriteFile` truncates, so
`cmd/helix_config/main.go:917` genuinely leaves one file where the operator
believes there are two.

## Angle 3 — confirm-no-bigger-problem (§11.4.150(C))

Following the backup class is what paid, and what it found is not a collision
defect at all.

`cmd/security_fix_standalone/main.go` composes a backup directory, creates it,
and logs `💾 Backup created`. The identifier `backupDir` occurs **exactly three
times in the file**: composed (`:83`), `MkdirAll` (`:84`), logged (`:87`).
**There is no fourth reference — nothing is ever copied into it.** The directory
stays empty for the whole run.

The tool then rewrites source files in place: `os.WriteFile(issue.File,
[]byte(modifiedContent), 0644)` at `:386`. It builds cleanly (`go build` rc=0)
and is one of the operator-facing security tools named in the project manual.

So the collision question is moot — there is nothing in the folder to collide.
The real defect is a destructive tool announcing a safety net it never created.
**Filed as HXC-326 (High).**

Two smaller faults sit in the same six lines: a failed `MkdirAll` only logs a
warning and the run proceeds to modify files anyway; and the name is
second-granularity, so it will collide once it does hold something.

## What this changes about the family

1. **HXC-272 is one instance of a class with ~14 members**, not a one-off. Its
   closure should say so, or the class silently persists behind a closed ticket.
2. **The class has an internal severity ordering** that the original filing did
   not surface: results dirs < session ids < backups. Remediation should follow
   that order rather than file order.
3. **HXC-281 and HXC-268 are the same shape at the documentation layer** — an
   identifier (a guard number, a count) derived and then trusted. Their fixes are
   correct; what generalises is the discipline of deriving identifiers from
   something unique by construction rather than from something that merely
   *usually* differs.
4. **The most valuable finding came from following the worst-consequence branch
   first.** Sorting the sweep by what a collision would *cost* — rather than by
   how many sites matched — is what surfaced HXC-326.

## Honest boundary (§11.4.6)

This pass establishes the class is real, recognised, and present here, and it
found one materially worse instance. It does **not** audit all 1158 timestamp
sites — only those matching a path/identifier composition shape, which is a
filter that certainly misses sites composing identifiers in a form my patterns
did not match. It does not establish how often the affected tools are run, nor
that any loss has already occurred (HXC-326 is filed with reachability stated as
unestablished). And it does not substitute for §11.4.108 runtime-signature
verification or the §11.4.40 retest.

## Sources verified 2026-08-13

- https://hackmd.io/@mcaradec/time_collisions
- https://www.systemdesignhandbook.com/guides/design-a-unique-id-generator-in-distributed-systems/
- https://lobste.rs/s/d1oqcl/nanosecond_timestamp_collisions_are
- https://news.ycombinator.com/item?id=36810818
