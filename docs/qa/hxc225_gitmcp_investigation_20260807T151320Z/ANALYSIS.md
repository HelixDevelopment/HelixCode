# HXC-225 — `mcp-servers/GitMCP` origin & usage investigation

**Date (UTC):** 2026-08-07T15:13:20Z
**Scope:** investigation only — no removal, no dependency upgrade, no Dockerfile added.
**Operator decision:** "investigate origin and usage first" (chosen over remove / upgrade).
**Subject path:** `submodules/helix_agent/mcp-servers/GitMCP/`
**Meta-repo HEAD at investigation:** `724d2bb0`

> Every claim is FACT with a cited evidence file, or explicitly marked `UNCONFIRMED:`.
> No claim is inferred from absence of grep hits alone (§11.4.124).

---

## 1. Origin — vendored third-party, not ours

**FACT.** `mcp-servers/GitMCP` is a **verbatim vendored copy of the upstream
open-source project `idosal/git-mcp`** (the service behind https://gitmcp.io).

| Signal | Value |
|---|---|
| `LICENSE` | Apache 2.0, **"Copyright 2025 GitMCP Authors (Ido Salomon and Liad Yosef)"** |
| `package.json` name | `git-mcp` |
| `package.json` description | "GitMCP is a tool that allows you to get the documentation for a given repository." |
| `README.md` badge | `https://gitmcp.io/badge/idosal/git-mcp` |
| `SECURITY.md` | upstream's own vulnerability-disclosure form |
| `wrangler.jsonc` routes | `pattern: "gitmcp.io", custom_domain: true` |
| `wrangler.jsonc` kv_namespaces | hardcoded **upstream production** KV ids (`c5dd8e05…`, `bfc07868…`) |

The `wrangler.jsonc` is decisive: this artifact is configured to **deploy to
someone else's Cloudflare account and someone else's domain**. It is not our code
and was never adapted to be.

**FACT — how it entered the tree** (`01_gitmcp_path_history.txt`):

```
20cf3f39  Port 19 MCP Servers from HelixCode repositories
          Милош Васић <i@mvasic.ru>   2026-01-16 00:13:41 +0300
          "... 8. GitMCP - Git repository MCP integration ..."
```

Only 4 commits ever touched the path; the 2 substantive ones are the bulk-port
pair (`5f4f3dd6`, `20cf3f39`). The other two are incidental (`09235440` Gemini
work tracking env templates; `cee02cda` annotating a TS skip). There is **no
adaptation history** — nobody ever modified this code for our use.
169 files tracked; path is **not** gitignored (`git check-ignore` → not ignored).

### 1a. §11.4.74 catalogue-check: we already consume this upstream properly

**FACT** (`06_duplicate_copies.txt`). The same upstream project is **already
present a second time**, correctly, as a submodule of our own fork:

```
.gitmodules:  [submodule "cli_agents/git-mcp"]
              path = cli_agents/git-mcp
              url  = git@github.com:vasic-digital/caf-git-mcp.git
git ls-tree:  160000 commit c487a29895dc…  cli_agents/git-mcp
```

Two copies of one upstream project:

| | `cli_agents/git-mcp` | `mcp-servers/GitMCP` |
|---|---|---|
| Mechanism | submodule (160000) of **our fork** `vasic-digital/caf-git-mcp` | copy-pasted dir, 169 files tracked inline |
| Upstream tracking | yes — real git remote | none; frozen at the 2026-01-16 paste |
| `pnpm-lock.yaml` md5 | `727ff9b4…` | `cf0f1cc1…` (**different snapshot**) |
| Divergence | — | newer: has `ViewCounterDO.ts` + test, absent from the fork |

Exactly the situation §11.4.74 (catalogue-first, extend-don't-reimplement) and
§11.4.51(C) exist to prevent. The duplicate is the *undermanaged* copy — and it
is the one carrying the advisory load.

---

## 2. Reachability — nothing we ship reaches it

**Answer: NO.** Positive finding (a mechanism was traced), not merely "no imports found".

### 2a. The Go agent named `gitmcp` is a *different, independent* implementation

The strongest false-positive risk was `internal/clis/agents/gitmcp/gitmcp.go` — a
registered, real CLI agent with the same name. It is **unrelated** to the vendored
directory (`07_go_agent_vs_vendored.txt`):

```
### Does the Go gitmcp agent reference the vendored dir, node, or any JS entrypoint?
NONE — Go agent has zero reference to mcp-servers/GitMCP or any JS runtime
```

Its own doc comment states what it does instead:

```go
// Git MCP: Model Context Protocol for Git operations. Operations are performed
// by exec-ing the real `git` CLI in the configured repository …
const gitBinary = "git"
…  exec.LookPath(gitBinary);  exec.CommandContext(ctx, bin, args...)
```

It shells out to the system `git` binary. It does not spawn Node, does not read
the vendored tree, and would behave identically if the vendored directory were
deleted. **The name collision is the entire reason this component looks used.**

### 2b. No Go source references the vendored path

`grep -rn 'mcp-servers/GitMCP' --include='*.go'` → **no hits.** Every
`mcp-servers` hit in Go is a different thing: `~/.helixagent/mcp-servers` (runtime
install dir), `external/mcp-servers` (test fixture), `docker-compose.mcp-servers.yml`.

Contrast with the analogous shipped path: `cmd/helixagent/main.go` really does
spawn `node $HELIX_HOME/plugins/mcp-server/dist/index.js` (lines 2899, 2995, 4248).
**GitMCP has no analogous spawn path anywhere.**

### 2c. The only wiring is the compose service — and it cannot build

`docker-compose.protocols.yml:395-410`:

```yaml
  # GitMCP - Git operations via MCP
  mcp-gitmcp:
    build:
      context: ./mcp-servers/GitMCP
      dockerfile: Dockerfile
    environment:
      - MCP_SERVER_NAME=gitmcp
      - REPOS_DIR=/repos
    volumes:
      - mcp_git_repos:/repos:rw
```

The referenced `Dockerfile` does not exist (§3), so the image cannot be built and
the service can never start. That is why the 3 criticals are dormant, not running.

---

## 3. Why the Dockerfile is missing — it was **never written**

**FACT, not inference** (`02_dockerfile_forensics.txt`):

```
### Any Dockerfile EVER in mcp-servers/GitMCP (all history, incl deleted)?
(end — empty means never existed)

### Deletions ever under mcp-servers/GitMCP
(end)

### Every path named Dockerfile ever in helix_agent history under mcp-servers/
mcp-servers/postgres-mcp/Dockerfile
```

`git log --all --diff-filter=ADMR` over the path returns nothing; `--diff-filter=D`
returns nothing. Across *all* `mcp-servers/**` history exactly one Dockerfile ever
existed, and it belongs to `postgres-mcp`. Neither `.gitignore` mentions docker.
So: **never written, not deleted, not ignored.** The service was wired speculatively.

### 3a. Not a GitMCP-specific oversight — 11 of 12 services are phantom

`03_dockerfile_coverage.txt`: `docker-compose.protocols.yml` declares 12 services
with `context: ./mcp-servers/*`. Exactly **one** (`postgres-mcp`) has a Dockerfile.
The other 11 — `ai-experiment-logger`, `conversational-api-debugger`,
`design-to-code`, `domain-memory-agent`, `lumera-agent-memory`, **GitMCP**,
`huggingface-mcp`, `shell-tool-mcp`, `workflow-orchestrator`,
`project-health-auditor`, `mcp-wiki` — are identically unbuildable.

GitMCP is not a special case. The whole mcp-servers block of that compose file is
aspirational scaffolding.

### 3b. The compose entry describes software that does not exist here

The wiring does not merely lack a Dockerfile — it describes **the wrong program**:

- Comment says *"Git operations via MCP"*; env is `REPOS_DIR=/repos` with a
  `mcp_git_repos` volume — a local-git-operations server.
- The vendored code is a **Cloudflare Workers documentation-fetching service**:
  `src/index.ts` imports `McpAgent` from `agents/mcp`, uses Durable Objects, R2,
  Vectorize, Workers AI and a React-Router SSR handler. It fetches *remote GitHub
  docs*; it performs no local git operations and has no `/repos` concept.
- `grep 'REPOS_DIR|MCP_SERVER_NAME'` over the vendored tree → **no hits**
  (`04_compose_vs_code_mismatch.txt`). The service's environment contract is
  unknown to the code it points at.

A Cloudflare Worker cannot be containerised this way at all — it needs the
`workerd` runtime plus account-bound DO/R2/Vectorize/KV/AI bindings. Whoever
authored the compose entry worked from the *directory name*, not the code.

