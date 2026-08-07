# Closure-readiness verification — 8 candidate items (2026-08-07)

Independent verification session. Every exit code below was observed in THIS
session by re-running the instrument; no prior agent's report is relied upon
(§11.4.123, `superpowers:verification-before-completion`). Verdict per item:
4 CLOSE, 4 NOT-CLOSE.

Parent HEAD at verification: `6d8a4920`.
Working tree at verification: clean except `docs/workable_items.db` (this
session) and `submodules/containers` (pointer intentionally unbumped, see
HXC-218).

## Artifact-layer facts (§11.4.108 SOURCE → ARTIFACT)

`submodules/helix_agent` pinned at `e81a474a`. All three candidate fixes are
ancestors of that pin — verified with `git merge-base --is-ancestor`, exit 0 for
each:

| commit | subject | ancestor-of-pin |
|---|---|---|
| `d0a53e0b` | HXC-172 DNS-rebinding defence | exit 0 |
| `9a50509d` | HXC-221 entropy fail-closed | exit 0 |
| `0a48f8c2` | HXC-222 untrack node_modules | exit 0 |

`submodules/containers` pinned at `a432efa8`; worktree at `11280cc`. The pin is
an *ancestor* of the fix, so the fix is strictly ahead of what the parent
builds. `internal/netaddr/netaddr.go` — the file carrying the bracketing fix —
does not exist at `a432efa8` at all.

## Guard runs — both polarities

A guard that passes at both polarities is blind and cannot support a closure.
Every guard below produced *different* results at the two polarities.

| guard | polarity | exit | result |
|---|---|---|---|
| `scripts/git_hooks/test_mutation_residue_evidence_exempt.sh` | `RED_MODE=1` | 1 | 8 PASS / 1 FAIL — RED no longer reproduces (defect gone) |
| same | `RED_MODE=0` | 0 | 10 PASS / 0 FAIL — standing green guard |
| `go test ./internal/services -run HXC221` (helix_agent) | `RED_MODE=0` | 0 | PASS incl. 8-case extended set (§11.4.146 STEP 3) |
| same | `RED_MODE=1` | 1 | RED correctly fails to reproduce on the fixed artifact |
| `go test ./cmd/security_scan -run HXC218` | `RED_MODE=0` | 0 | PASS (but see HXC-218 caveat below) |
| same | `RED_MODE=1` | 1 | RED fails to reproduce — because the build sees the *worktree* |

## HXC-217 — the finding that blocks its own closure

`workable-items validate` reports `OK — 430 items` and exits 0. That result comes
from `constitution/scripts/workable-items/bin/workable-items`, a prebuilt binary
dated **2026-07-27**, which predates the HXC-217 guard landed in `8214bd93` on
2026-08-06. The binary does not contain the guard's diagnostic string.

A/B on an identical mutated DB copy with an identical `PWD`, one closed item's
`item_history.evidence_path` replaced by narrative prose:

- `bin/workable-items validate` → **exit 0**, `OK — 430 items` (guard absent)
- `go run ./cmd/workable-items validate` → **exit 1**, 1 violation correctly
  identified as "narrative or multi-value text in a single-path field"

So the guard is correct in source and absent from the artifact everyone runs.

**Self-correction (§11.4.6, recorded rather than hidden):** an earlier run of
mine reported 119 violations against the real DB. That was my own invocation
error, not a defect — `resolveInvocationRelative` anchors relative paths on the
invoking shell's `$PWD`, and I had run from
`constitution/scripts/workable-items`. Re-run with `PWD` at the repo root, the
source validator exits 0 on the real DB. This is exactly the §11.4.201(1)
false-positive the code comments warn about, and it nearly became a false
finding in this report.

## Per-item verdicts

**CLOSE — HXC-221.** Fix in the pinned artifact; `generateID(entropy io.Reader)
(string, error)` confirmed by reading the artifact, no clock fallback; both
polarities distinguish; extended case set covers partial reads, empty source,
nil-defaults-to-crypto/rand, and concurrency.

**CLOSE — HXC-222.** 0 tracked `node_modules` files; `.gitignore` covers the
trees; fix in the pinned artifact. The two sdk/web defects surfaced are
pre-existing and out of scope for a hygiene ticket, and the regenerated tree
repaired a dangling `tsc` symlink that the committed tree had.

**CLOSE — HXC-223.** Guard distinguishes at both polarities. The exemption is
keyed on inertness, not location, and the N-cases pin the boundary (executable
`.sh` under `docs/qa`, `.log` outside the evidence tree, `.sh` extension at mode
100644, executable-mode `.log` — all still blocked). Residual gap acknowledged by
the author is filed separately rather than absorbed into this closure.

**CLOSE — HXC-228.** `wait-infra-ready.sh` present in the artifact and wired as
`ExecStartPost` on the infra unit; `bash -n` exit 0; the script genuinely polls
container running+healthy state with a deadline rather than returning
immediately. Live cold-boot measurement is the operator's own, not an agent
report: pre-fix baseline `NRestarts=1` / `ExecMainStatus=1`, post-fix 0 restarts
across a full-target cold boot.

**NOT-CLOSE — HXC-172.** Status is already `Operator-blocked`. The item's own
expected outcome requires the protective setting "explicitly verified as enabled
in our configuration rather than assumed" — `MCP_ALLOWED_HOSTS` appears in 0
tracked `.md` files in helix_agent, so it is implemented but undocumented and
unverified as enabled. The §1.1 paired mutation was never run.

**NOT-CLOSE — HXC-218.** The fix is not in the consuming build. The parent pins
`a432efa8`, which lacks `internal/netaddr/netaddr.go` entirely. The consuming
guard passes only because `helix_code/go.mod` carries
`replace digital.vasic.containers => ../submodules/containers`, resolving to the
worktree at `11280cc`. A fresh clone builds the pre-fix code. The containers
submodule is additionally 129 commits behind `origin/main`.

**NOT-CLOSE — HXC-217.** Its own guard is absent from the shipped binary — see
above. Closing it would certify an enforcement that the artifact does not
perform.

**NOT-CLOSE — HXC-166.** The triage (0 of 210 reachable) is real work, but the
item's expected outcome is to "upgrade or replace the dependencies behind the
critical and high findings first". No upgrade or replacement has been performed.
The separation half is done; the remediation half is untouched.
