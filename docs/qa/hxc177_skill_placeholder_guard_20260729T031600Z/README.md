# HXC-177 — a skill manifest placeholder the engine cannot resolve, leaked into LLM prompts

| | |
|---|---|
| Revision | 1 |
| Created | 2026-07-29 |
| Last modified | 2026-07-29 |
| Status | active |
| Item | HXC-177 (Bug / Medium) → `Fixed (→ Fixed.md)` |
| Scope | `helix_code/internal/commands/` |
| Commits | `b741d7da` (fix + guard), `2b7661d5` (guard summary truthfulness) |
| Host | Linux 6.12.41; Go 1.26 |

## Table of contents

- [1. The defect and the exact resolution rule](#1-the-defect-and-the-exact-resolution-rule)
- [2. Sweep of all manifests](#2-sweep-of-all-manifests)
- [3. The fix](#3-the-fix)
- [4. The guard](#4-the-guard)
- [5. RED → GREEN evidence](#5-red--green-evidence)
- [6. A false-success line found in the guard itself](#6-a-false-success-line-found-in-the-guard-itself)
- [7. What was not verified](#7-what-was-not-verified)

## 1. The defect and the exact resolution rule

The shipped `conventional-commit` manifest wrote `{{spec_url}}`. The rule that governs
substitution is a single regex in `helix_code/internal/commands/markdown_commands.go`:

```go
var substRegex = regexp.MustCompile(`\{\{([A-Z_][A-Z0-9_]*(?:\.[A-Za-z0-9_]+)?(?::[^}]+)?)\}\}`)
```

The base token must begin `[A-Z_]`. `spec_url` begins with a lowercase `s`, so
`{{spec_url}}` **never matches the regex at all** — it is not merely unresolved, it never
reaches the resolver. The literal text is passed through to the model as prompt noise.

Named variables — whether declared in frontmatter `variables:` or captured by a trigger's
`(?P<name>...)` group, which `RenderWithCaptures` merges into one map — are reachable only
via `{{ARG.<name>}}` (`buildResolver`: `strings.HasPrefix(token, "ARG.")` →
`c.variables[varName]`). Verified independently against source; see
`evidence/01_engine_rule.txt`.

## 2. Sweep of all manifests

The engine embeds exactly one manifest tree (`//go:embed builtin_skills` in
`markdown_skills.go`). All **8** were swept; **1** was wrong:

| Manifest | Result |
|---|---|
| `conventional-commit` | **WRONG** — `{{spec_url}}` |
| `action-prefix-system` | OK (`{{ARG.action}}`) |
| `media-validator` | OK (`{{ARG.path}}`) |
| `reporting-workable-items` | OK (`{{ARG.kind}}`, `{{ARG.summary}}`) |
| `scheduled-work-queue` | OK (`{{ARG.task}}`) |
| `multitrack`, `session-sync`, `workable-item-lifecycle` | OK (no placeholders) |

The seven recently-authored manifests were the reference shape, and they were correct.

## 3. The fix

`{{spec_url}}` → `{{ARG.spec_url}}` in
`helix_code/internal/commands/builtin_skills/conventional-commit/SKILL.md`.

## 4. The guard

`helix_code/internal/commands/skill_manifest_placeholder_guard_test.go` extracts every
`{{...}}` token from every embedded manifest and fails on any token the engine cannot
resolve. It follows the repo's `RED_MODE` convention, and `require.Greater(t, swept, 0)`
prevents the walk from vacuously passing if it ever stops finding manifests. A companion
test accepts every supported token form (`{{ARG1}}`, `{{ARG.x}}`, `{{SELECTION}}`,
`{{CURRENT_FILE}}`, `{{CWD}}`, `{{ENV.X}}`, `{{FILE:...}}`) so the guard cannot be
satisfied by weakening the checker instead of fixing a manifest.

## 5. RED → GREEN evidence

The decisive proof is a paired mutation against a **real shipped manifest**, not only the
guard's own fixture — `evidence/02_RED_mutation_on_shipped_manifest.txt`:

```
--- FAIL: TestSkillManifestPlaceholders_NoUnresolvableTokens/conventional-commit
    Error: Should be empty, but was [conventional-commit: placeholder {{spec_url}} does not
    match the engine's substitution pattern (...) and is sent to the LLM VERBATIM, never substituted]
    swept 8 builtin skill manifest(s), found 1 unresolvable placeholder(s)
FAIL      mutated_exit=1
```

`evidence/03_GREEN_shipped_set.txt` — restored:

```
swept 8 builtin skill manifest(s), zero unresolvable placeholders
ok  dev.helix.code/internal/commands      green_exit=0
### manifest restored byte-identical:  CLEAN
```

`RED_MODE=1` (fixture polarity) also passes. Full package `go test ./internal/commands/`
is `ok`; `gofmt -l` clean on the guard file; `go vet` clean.

## 6. A false-success line found in the guard itself

Running the paired mutation surfaced a defect in the guard's own reporting: its closing
`t.Logf` printed `"swept N builtin skill manifest(s), zero unresolvable placeholders"`
**unconditionally** — including on the run whose subtests had just failed. A summary line
that asserts success independently of the result is a false-success surface (§11.4 /
§11.4.1), and it is exactly the line a reader skimming output trusts most.

Fixed in `2b7661d5`: the count is accumulated across subtests and reported honestly.
Both polarities re-verified after the change — mutated reports `found 1`, restored reports
`zero`.

## 7. What was not verified

- **No end-to-end prompt capture.** The proof that the token reaches the model verbatim is
  derived from the regex and resolver source, not from an intercepted outbound LLM request.
  The claim is therefore established by code inspection plus the guard, not by wire evidence.
- **Runtime skill tiers were not swept.** `~/.helix/skills` and `.helix/skills` are loader
  tiers but are not tracked/shipped directories in this repo, so a user-authored manifest
  placed there at runtime is outside the guard's embedded-set scope.
- **`constitution/skills/*/SKILL.md` was deliberately not swept** — it is not loaded by this
  Go engine, and that tree was owned by a parallel work stream during this session.
