# HXC-169 — container-sandbox advisories are structurally unreachable

**Resolution commit:** `95c27998` (guard) · **Analysis:** `docs/research/hxc169_container_sandbox_reachability_20260731/ANALYSIS.md`
**Gate:** `CM-OPENHANDS-DOCKER-SANDBOX-UNWIRED` — `gate_run.txt`, exit 0, 5 assertions / 0 violations.

Not "low risk": the sandbox calling the vulnerable copy operations exists and IS
vulnerable, and nothing imports it, so the code never enters a shipped artifact.

Re-verified by the orchestrator, not taken on report:
`go list -deps ./cmd/...` → 1097 deps, **zero** `github.com/docker/docker`, **zero**
`clis/openhands`; `master.go:19` imports the WIRED SIBLING `clis/agents/openhands`;
`git log --all -S` shows the sandbox was never imported at any point in history.

The apparent contradiction (the item said calls WERE traced) resolves mechanically:
the originating sweep ran `govulncheck ./...`, which treats every package as an
analysis root including unimported ones. Scoped: exit 3 on the package, exit 0 on
`./cmd/...`.

An unreachability verdict without a guard rots the day someone wires it up — hence
the gate, which fails if any file outside the package imports it.

**Open, operator-gated:** whether to finish, keep, or remove the unwired sandbox.
