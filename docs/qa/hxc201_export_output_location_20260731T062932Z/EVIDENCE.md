# HXC-201 — the documented regeneration command wrote to the wrong place

**Commits:** `428ff9db` (constitution submodule) · `b617680d` (docs + guard)
**Gate:** `CM-EXPORT-OUTPUT-LOCATION` (G24) — `gate_run.txt`, exit 0.

`go run -C <dir>` DOES change the child's working directory, so a relative
`--out-dir` landed inside the tool's own folder while printing correct-looking
destinations and reporting success. This refutes a claim committed in this repo's
history asserting the opposite — a claim "verified" by reading rather than running,
which is precisely how it survived.

Fixed at the tool (paths anchored at the invoking shell's directory) AND in the
docs, because a doc-only fix relies on humans copying correctly forever.

**Proven on real data, not a fixture:** the exact previously-broken relative-path
command was run against the live 409-item database and wrote all four documents to
the real `docs/` tree, with nothing landing in the tool directory.

The guard asserts output LOCATION, not exit code — the whole defect was reporting
success while writing nowhere useful.
