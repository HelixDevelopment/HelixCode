# HXC-212 — a fix activated a dormant any-origin hole

**Commits:** `e27581da` (submodule) · `58984d27` (G26 + pointer)
**Gate:** `CM-MCP-SERVER-CORS-ALLOWLIST` — `gate_run.txt`, exit 0, 4 assertions.

The SSE transport answered every browser with a wildcard Allow-Origin on both the
preflight and the response path. It was DORMANT: the service only spoke stdio, so
no socket was opened and `runSSE()` was unreachable.

`30c81925` correctly repaired the inert-configuration defect — and part of that
repair sets `MCP_TRANSPORT=sse`. The service now listens on a published port, and
the wildcard became reachable. **A correct fix to one defect activated a second
that had only ever been safe by accident.**

Two layers asserted separately (§11.4.108): the tracked `dist/` carried the
wildcard independently of `src/`, so a source-only check would pass while a fresh
clone running `node dist/index.js` without building still served the hole.

Mutation-tested by the orchestrator in BOTH layers: wildcard planted in `src/` →
exit 1; planted in tracked `dist/` → exit 1 naming the fresh-clone consequence;
restored → exit 0.

**Class NOT closed:** 8 further wildcard sites remain elsewhere in the submodule.
A class-closed claim is only as wide as the tree that was swept.
