# What this run certifies

`048ca700` removed a determinism hazard: the cache TTL and stale-window tests
asserted expiry by sleeping past an approximate wall-clock window. `time.Sleep`
guarantees only a LOWER bound on elapsed time, so under host load the verdict
became a function of machine load rather than of the code under test (§11.4.50).

This is a STRENGTHENING, not a weakening, and this run proves it as a fact
rather than repeating the claim: `git grep` finds NO surviving `time.Sleep` in
`cache_test.go`, the injectable clock is in the shipped source, and the boundary
tests — which can now assert at exactly `ttl` and at `ttl + 1ns`, a boundary no
sleep can express — run green under `-race -count=2`.

Scope: this commit changed a test seam. The production behaviour is unchanged by
construction — `clock()` falls back to `time.Now` when the field is unset, so a
zero-value Cache behaves exactly as before.
