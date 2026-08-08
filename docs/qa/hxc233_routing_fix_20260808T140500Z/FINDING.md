# HXC-233 routing half — the gateway's completion path was pointed past a working model

**Run:** 2026-08-08T14:05Z · **Method:** `superpowers:systematic-debugging` Phase 1→4

## The finding

The gateway returned HTTP 500 on every completion — "all providers exhausted".
With no cloud keys configured, the local provider is its ONLY completion path.
The natural reading was "the local model is down". It was not.

| probe | result |
|---|---|
| `POST :8443/v1/chat/completions` | **HTTP 500** — `dial tcp 127.0.0.1:50052: connection refused` |
| `POST :18434/v1/chat/completions` | **200** — `"2 + 2 = 4"` from Qwen3-Coder-30B |
| `:50052` | nothing listening |
| `:18434` | LISTENING |

The capability was present the whole time. The ROUTING pointed past it.

## Root cause (traced to source, not inferred)

`internal/shared/config/config.go:49`
```go
LocalRPCPort int `env:"HELIX_LLM_LOCAL_RPC_PORT" default:"50052"`
```
The live process had the variable UNSET, so the code default applied — and 50052
matches no service in this deployment. `helixllm-coder.service`, part of the same
unit family, serves the local model on 18434.

## The fix

Two `Environment=` lines in the gateway unit, placed BEFORE `EnvironmentFile=`
so an operator `.env` can still override (the same pattern the GIN_MODE fix used):

```
Environment=HELIX_LLM_LOCAL_RPC_HOST=127.0.0.1
Environment=HELIX_LLM_LOCAL_RPC_PORT=18434
```

## The flip, on the public endpoint

```
before  HTTP 500  {"error":{"message":"brain error: all providers exhausted,
                   last error: llamacpp: ... dial tcp 127.0.0.1:50052: connection refused"}}
after   HTTP 200  {"choices":[{"message":{"role":"assistant",
                   "content":"2 + 2 equals 4."}}],
                   "model":"/models/Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf"}
```

A real answer, from the real 30B model, through the public HTTPS endpoint.

## Why the earlier error message was misleading

Before HXC-235's binary deploy, this same failure surfaced as a TLS error against
port 8080 — because the pre-fix code overrode the local target whenever an embed
was *attempted*, even when the embedded server never started. HXC-233's source
fix (`8990df4`) made the failure honest; deploying it exposed the true cause.
The honest error is what made this five-minute diagnosis possible.

## No regression

health 200 · /v1/models 200 · /v1/embeddings 200 with semantic_embeddings key ·
helixagent 200 · helixcode-server 200 · 12 containers · NRestarts=0 ·
coder (30B) untouched (MainPID=0, active) · HXC-229 guard GREEN on the new pid.

## Honest boundary (§11.4.6)

This routes the gateway's local provider at the Qwen3-Coder-30B backend. That is
the only local model this deployment runs, and the gateway's local provider is
documented in `main.go` as the "local safety net" — but a coder-tuned model is
now answering general completions. If a general-purpose local model is added,
this target should move to it. No standing guard yet asserts the completion
path stays alive; that is owed.
