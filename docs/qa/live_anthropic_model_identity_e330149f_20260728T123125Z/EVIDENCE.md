# LIVE Anthropic-wire model identity — QA evidence (§11.4.83)

**Source commit:** `e330149f499737c07040fa3aef664e0aadfd1a1b`
**Subject:** fix(llm/server): report the model that ACTUALLY served the request
**Verdict:** PASS
**Captured (UTC):** 2026-07-28T12:32:52Z

## Why this run exists

The Anthropic non-stream path's model-identity fix had unit coverage only. A
second client-visible wire deserves its own live proof — unit green is not
runtime green (§11.4.108).

## What was driven

Ephemeral server on the freshly built binary, HELIX_LLM_PROVIDER=helixllm
against the real coder on :18434, then a real Anthropic Messages request:

    POST /v1/messages   {"model":"default","max_tokens":24,...}

The caller sent the ALIAS `default`. The coder advertises
`/models/Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf`.

## Result

The response's `model` field:

    "model":"/models/Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf"

The Anthropic wire now names the model that actually served the request.
Pre-fix it echoed `default`. Full capture: `transcripts/live_anthropic.json`.

## Honest boundary (§11.4.6)

This covers the Anthropic NON-STREAM surface. Anthropic streaming
`message_start` still carries the requested alias — that event must precede
any chunk, so the served model is genuinely unknowable there; documented
in-source at the site.
