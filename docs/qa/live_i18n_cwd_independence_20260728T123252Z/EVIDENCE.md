# LIVE i18n cwd-independence — QA evidence (§11.4.83)

**Source commit:** `8cec90baadad281eb26a5671b792921f03eb7386` (tool_schema submodule)
**Subject:** fix(i18n): embed translation bundles so lookups work from any directory
**Verdict:** PASS
**Captured (UTC):** 2026-07-28T12:32:52Z

## Why this run exists

The fix had go-test coverage only. The defect was cwd-dependent bundle
resolution, so the honest runtime proof is a REAL COMPILED BINARY executed
from a working directory unrelated to the module — the exact condition that
broke.

## Live surface — stated honestly (§11.4.6)

This library has NO live HTTP surface on the running agent to probe: the
running helixagent's `/v1/mcp/tools` returns `{"tools":[]}`, consistent with
the independently-established finding that RegisterTool has zero production
call sites. So the strongest available runtime evidence is a compiled
consumer binary, not an HTTP transcript. That is what was captured.

## What was driven

A real Go program in the CONSUMER module (helix_agent) calling
`GitHandler.GenerateDefaultArgs("commit")`, compiled to a binary and executed
from two unrelated working directories.

## Result

    cwd=/     -> "description":"Create git commit"
    cwd=/tmp  -> "description":"Create git commit"

A real translated string. Pre-fix this returned the RAW KEY
`toolschema_git_desc_commit` whenever the working directory was not
tool_schema's own root. Captures in `transcripts/`.
