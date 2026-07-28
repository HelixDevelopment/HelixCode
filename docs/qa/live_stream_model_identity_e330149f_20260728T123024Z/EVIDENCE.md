# LIVE streaming model identity — QA evidence (§11.4.83)

**Source commit:** `e330149f499737c07040fa3aef664e0aadfd1a1b`
**Subject:** fix(llm/server): report the model that ACTUALLY served the request
**Verdict:** PASS
**Captured (UTC):** 2026-07-28T12:30:56Z

## Why this run exists

The non-stream path was already proven live. The STREAMING path was proven
only by unit test and an httptest-backed provider test — never driven
`stream:true` against the actual running server and the real coder. Source
and unit-level green is not live proof (§11.4.108: SOURCE and ARTIFACT green
say nothing about RUNTIME). This run closes that gap.

## What was driven

A freshly built server booted ephemerally with HELIX_LLM_PROVIDER=helixllm
pointed at the real coder on :18434, then a real streaming request:

    POST /v1/chat/completions   {"model":"default","stream":true,...}

The caller sent the ALIAS `default`. The coder advertises, via its own
/v1/models, the concrete model:

    /models/Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf

## Result

    SSE frames captured                : 13
    distinct model ids across frames   : 1
    the id present in every frame      : /models/Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf
    frames carrying the alias "default": 0

Every emitted `chat.completion.chunk` names the model that actually served
the request. Pre-fix, every frame would have carried `"default"`.

Full wire capture: `transcripts/live_stream.sse`.

## Additional finding

An independent review raised a NIT that a backend reporting no model on early
frames could produce an alias→concrete transition mid-stream, which the OpenAI
wire convention discourages. This run shows that does not occur on the real
path: the distinct-model-id count across all 13 frames is 1, so the
identity is constant for the whole stream.

## Honest boundary (§11.4.6)

This proves the OpenAI streaming surface against ONE backend (the local coder).
Backends that omit `model` from their SSE frames fall back to the requested
model by design — that fallback is covered by unit tests, not by this run.
Anthropic streaming `message_start` still carries the requested alias for the
structural reason documented in-source.
