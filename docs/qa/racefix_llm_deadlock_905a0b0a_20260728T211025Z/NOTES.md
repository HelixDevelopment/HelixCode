# What this run certifies

`905a0b0a` closed a deadlock that was guaranteed rather than probabilistic —
`StreamWithTools` collected a provider stream into a channel nothing was draining
past its buffer — and the sibling class across eight providers, where a stream
goroutine blocked forever on `ch <- …` after its consumer went away, leaking a
goroutine per abandoned request. Every send is now paired with `case <-ctx.Done()`.

Captured here: the ctx-aware parser signatures in the shipped source of two
providers, both named guards executed under `-race -count=2` with `--- PASS:`
asserted per test, and the whole `internal/llm` package race-clean.

Not certified: no live provider is contacted by this run. The guards drive the
stream machinery with in-process fakes, which is what makes them deterministic;
the wire behaviour of a real provider is a separate evidence class.
