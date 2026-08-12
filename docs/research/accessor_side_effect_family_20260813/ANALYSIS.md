# §11.4.150 deep multi-angle research pass — accessor-side-effect family

**Covers:** HXC-274 (and bears on HXC-283).
**Conducted:** 2026-08-13, main stream (T1/main - claude4), in parallel with four
background review streams per §11.4.150(F).
**Sources verified:** 2026-08-13 (URLs below, per §11.4.99).

**Outcome: no new instance found — and the more useful result is WHY the search
could not have found one.** This pass ends in a negative with a stated limit
rather than a finding, and the limit is the part worth keeping.

---

## The item

**HXC-274** — starting the server made real outbound provider calls, using
ambient credentials, through a method shaped like a getter. The cost was not
performance: it spent the operator's money and sent data to third parties as a
side effect of *starting up*.

## Angle 1 — is this a named class?

Yes. Command–Query Separation (Meyer) states that each method should be either a
command that mutates state or a query that returns data, **never both**;
"properties should not have side effects — retrieving a property should not
change the state of the object."

The refactoring literature names our exact trigger:

> when a method that looks like a **getter, validator or calculation** also
> mutates object state (timestamps, caches, counters, DB writes, metrics)

and gives the canonical example — a `getUserPreferences()` that fetches from the
database and writes back to cache, where "the caller expects a read but the cache
write is a surprising side effect."

HXC-274 is the severe form of this. The hidden work is not a cache write but an
**outbound network call with ambient credentials**, which converts a naming
convention issue into a spend-and-disclosure issue.

## Angle 2 — measure our own corpus, and discover the instrument is wrong

Swept accessor-shaped functions (`Get*`, `List*`, `Is*`, `Has*`, `Available*`)
whose bodies contain outbound I/O.

**First measurement: 45 in the inner app. That number was wrong, and reading the
lines is what caught it.** The regex matched `.Do(` — which is `http.Client.Do`
*and* `sync.Once.Do`. The two most suspicious hits by name — `config.Get()` and
`event.GetGlobalBus()` — were both `sync.Once.Do` memoisation with no I/O at all.
Exactly the shape a config getter should have.

Re-measured with the confound removed: **35** (inner app) and **48**
(`helix_agent`).

**And those are almost all legitimate.** `verifier/client.go GetModels()`,
`adapters/cache/adapter.go Get()`, `http/pool.go Get()` — these are clients and
pools. Making a call *is* their purpose; the name is honest about what the object
is for.

So the sweep cannot separate signal from noise, and the reason is structural:

> **HXC-274's defect was not "an accessor makes a call". It was "something the
> STARTUP PATH invokes makes a call".** The distinguishing property is *call
> context*, not *name shape* — and a name-shape sweep cannot see call context.

A `GetModels()` on an explicitly constructed provider client is correct. The same
method reached from `main()` before any user action is the defect. Identical
shape, opposite verdicts.

## Angle 3 — confirm-no-bigger-problem, with an honest negative

Two things were checked that a shape sweep *can* answer:

- **Import-time I/O: none.** No `func init()` in the inner app's `internal/` or
  `cmd/` performs network I/O. This is the worst version of the class — work
  that happens on import, before any caller decides anything — and it is absent.
- **The two names that most strongly imply no-I/O** (`config.Get`,
  `event.GetGlobalBus`) were read directly and are clean.

What was **not** established: whether any of the 83 accessor-shaped I/O sites is
reachable from a startup path before a user action. Answering that needs a
call-graph walk from each `cmd/*` main, not a name sweep — and that is the
correct instrument for this class.

**Stated as a gap rather than implied clean (§11.4.118).**

## What this changes about the item

1. **HXC-274's closure is supported**, and its class is field-recognised rather
   than a local observation — which strengthens the fix's rationale.
2. **The right guard for this class is a call-graph assertion, not a lint on
   names.** "No `cmd/*` main reaches an outbound call before the first user
   action" is mechanically checkable and would catch the next instance; a naming
   rule would not, and would produce mostly false positives, as measured here.
3. **The recurring lesson is the instrument, again.** A first count of 45 was
   inflated by `sync.Once.Do`; the corrected 35/48 is accurate and still not a
   finding, because the shape does not carry the defect. Three separate passes
   this session have now produced a count that had to be discarded after reading
   the lines.

## Honest boundary (§11.4.6)

This pass found **no new instance**, and that negative is only as strong as its
instrument — which is explicitly the wrong one for this class. It establishes:
the class is real and named; import-time I/O is absent; the two highest-suspicion
accessors are clean. It does **not** establish that no startup path reaches an
outbound call, and it does not substitute for §11.4.108 runtime-signature
verification or the §11.4.40 retest.

## Sources verified 2026-08-13

- https://en.wikipedia.org/wiki/Command%E2%80%93query_separation
- https://www.dotnetcurry.com/patterns-practices/1461/command-query-separation-cqs
- https://blog.ploeh.dk/2015/10/08/command-query-separation-when-queries-should-have-side-effects/
- https://imartynov.substack.com/p/103-refactoring-book-refactoring
