# What this run certifies

`fa9f0247` closed a silent-empty-output defect: `ExecuteStream` could return exit
code 0 with ZERO captured output. `Cmd.Wait` closes the parent's read ends of
`StdoutPipe`/`StderrPipe`, and it was being called concurrently with the
scanners, so for a fast-exiting command Wait routinely won the race. `bufio.Scanner`
then reported that hard read error exactly as it reports clean EOF, and the
discarded `scanner.Err()` turned an I/O failure into a perfectly green empty
result — the exact shape of a §11.4 PASS-bluff at the tooling layer.

Captured here: the parent-owned pipe constructor and the surfaced scanner error
in the shipped source, and all three named guards executed under `-race -count=2`
with `--- PASS:` asserted per test.
