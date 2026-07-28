# What this run certifies

`3c8197cf` fixed the same defect shape in four places: a two-case `select` where
a real answer and a spurious signal were both ready, so Go's uniform random
choice could discard the answer (three UI update loops discarding "stop", one
discovery call discarding a delivered result in favour of its own timeout). It
also gave the three UI loops a done channel so teardown JOINS the goroutine
instead of merely asking it to stop.

Captured here: the extracted stop-prioritised step and the done channel in the
shipped source, eleven named guards executed under `-race -count=2` with
`--- PASS:` asserted per test, and all four affected packages race-clean.
