# Model identity on the wire — QA evidence (§11.4.83)

**Source commit:** `e330149f499737c07040fa3aef664e0aadfd1a1b`
**Subject:** fix(llm/server): report the model that ACTUALLY served the request
**Verdict:** PASS
**Captured (UTC):** 2026-07-28T11:34:41Z
**Artifact:** helix_code/bin/helixcode rebuilt 2026-07-28 16:34:03 (contains the fix)

## What the end user gets

A client asking for the alias `default` against the local HelixLLM coder used
to be told `"model":"default"` while a Qwen3-Coder gguf actually served the
request — a model identity that never served anything (CONST-036 / CONST-037).

Captured on the wire during this run, against a freshly rebuilt artifact and a
real coder on :18434 — the request half and the response half of the same
exchange:

```
request  "model":"default"
response "model":"/models/Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf"
```

The full bidirectional transcript is `transcripts/eph_helixllm_generate.http`
(request lines sent, response headers and body received). The live route's
per-assertion verdicts are `live_route_verdicts.tsv` — 8 PASS, 0 FAIL, 0 SKIP,
including the assertion that the response body contains the coder-advertised
model id, whose expected value is discovered at runtime from the coder's own
/v1/models rather than hardcoded.

## Why this is not a metadata-only PASS

The assertion is the routing proof and the sink-side binding (§11.4.13): the
facade can only echo the coder's own model id if the request genuinely reached
:18434. Had it fallen through to another backend, the id would be that
backend's. The value is read from the sink, not asserted from configuration.

## Guards, each proven by mutation (§1.1 / §11.4.135)

Every guard was verified by DELETING the fix it covers, observing the failure,
then restoring (sha256 re-verified identical):

| Mutation | Guard that FAILed |
|---|---|
| neuter the facade's served-model preference | wire_facade_model_identity_test.go (both wires) |
| strip `Model` from `MarshalJSON` | round-trip guard: "marshalled payload lost the model field entirely" |
| strip `Model` from `UnmarshalJSON` | round-trip guard: "model did not survive the round trip" |
| delete `Model: streamResponse.Model` | openai_compatible_stream_model_test.go: "streamed chunk 1 dropped the backend-reported model" |

`RED_MODE=1` flips the polarity guards to assert the pre-fix echo; they
correctly FAIL on the fixed artifact, reporting the defect is no longer
reproducible.

## Honest boundary (§11.4.6)

Anthropic streaming `message_start` still carries the requested alias. That
event must be emitted before any chunk exists, so the served model is genuinely
unknowable at that point; deferring it would change client-visible first-token
timing. Documented in-source at the site. Every other surface — OpenAI
non-stream, OpenAI streaming, Anthropic non-stream — reports the served model.

## Review history

Two independent review rounds (§11.4.125 / §11.4.134) returned NO-GO before
this landed: round 1 found the streaming path unfed by the provider and the
Anthropic wire unfixed; round 2 proved the streaming guard was missing by
deleting the fix and watching the whole suite stay green. Both were remediated;
the mutation table above is the result.
