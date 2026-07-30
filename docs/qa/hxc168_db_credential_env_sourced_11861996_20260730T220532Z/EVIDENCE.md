# HXC-168 — database credential sourced from .env, never a file literal

**Commit:** `11861996` — fix(security): source the database credential from .env, never a file literal (HXC-168, CONST-042)
**Captured:** 2026-07-31, by the orchestrator, by RE-RUNNING the guard.

## Results

| Run | File | Exit |
|---|---|---|
| Guard on the clean tree | `guard_clean_tree.txt` | **0** |
| Paired self-test (planted literals) | `guard_selftest_both_directions.txt` | **0** — "gate proven in BOTH directions (clean tree: PASS, planted literals: FAIL)" |

The self-test is what makes this evidence rather than assertion: a secret-scanner
that cannot be shown to FIRE proves nothing. It plants a literal, an inline-URL
credential, and a tracked config literal, requires each to FAIL, and requires
documentation prose carrying the same word to be correctly IGNORED — so the guard
is neither blind nor noisy. Mutations run only in `mktemp`; the working tree is
never dirtied, which matters with concurrent agents live (§11.4.84).

The guard earned its keep during authoring: it caught a Grafana admin password
the implementer had not inventoried.

## SCOPE — THIS COMMIT DOES NOT CLOSE HXC-168

Only the code half is done. The credential remains exposed:

- present in **23 commits** of history reaching back to **2025-11-02** (271 days);
- confirmed on the remote `main` of every configured upstream;
- `docs/distribution/docker-compose.mistborn.yml` publishes postgres `"5432:5432"`
  — Docker short syntax, no IP prefix, so it binds 0.0.0.0 — with a production
  database name on a separate network-reachable host;
- the autoboot stack publishes `"55432:5432"` the same way and **starts
  automatically** unless explicitly disabled.

That it is a deliberate pattern rather than an unknown Docker default is settled
by `helix_code/docker-compose.builder.yml:75`, which correctly writes
`127.0.0.1:5432:5432`. Exactly one file in the repository restricts to loopback.

Nothing in this commit reduces the exposure of the **existing** credential — it
only guarantees no new literal lands. **Rotation is operator action** and the item
stays open until it happens; a history rewrite is both forbidden (§11.4.113) and
futile, since the remotes already carry it.

An earlier attempt exists: `25d41351` (2026-06-04) removed the same literal from
the SQL and `.env.example` and deferred the container files to "a separate item".
This is that item, two months later.
