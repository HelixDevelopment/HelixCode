# HXC-235 deploy — the fix is live, and the deployed artifact was 15 days stale

**Run:** 2026-08-08T13:52Z · **Method:** `superpowers:systematic-debugging` Phase 1→4

## Phase 1 — root cause (FACT, not inference)

An independent review refused to close HXC-235: correct and genuinely guarded,
but not live. I did not assume "just needs a restart" — the running binary was
inspected directly.

| | value |
|---|---|
| live binary | `/home/milos/.local/bin/helixllm` |
| **mtime** | **2026-07-24 23:58:08** |
| size / md5 | 40,747,273 · `47cd08ad2762f54b` |
| `semantic_embeddings` in it | **0** |
| freshly-built binary | builds EXIT=0, 59,760,382 bytes, `semantic_embeddings` **1** |

**Root cause:** the deployed artifact predates the fix by **15 days**, not the
13 hours the process-start time suggested. The process started 2026-08-07
23:40; the BINARY is from 2026-07-24. Four helix_llm commits had landed since
it was built — so HXC-233's fix was not live either, and two CPU-inference
feature commits had never been deployed.

This is why the process-start timestamp is not the artifact's age. Reading
`/proc/<pid>/exe` mtime is; §11.4.108 layer 2 asks what the ARTIFACT contains,
never when the process happened to start.

## Phase 4 — deploy, reversible-first

§9.2 backup taken before touching a live service:
`/tmp/wi-bak/helixllm.prefix.20260808T*` (40,747,273 bytes, md5 `47cd08ad…`).

Scope confirmed before restarting: `helixllm-coder` (Qwen3-Coder-30B) is
`MainPID=0` / `active (exited)` — a oneshot that cannot be disturbed by a
gateway restart. Only `helixllm-gateway` was restarted.

## The RED→GREEN flip, on the live endpoint

`POST https://localhost:8443/v1/embeddings`

```
pre-deploy   semantic_embeddings key count: 0     <- the defect, reproduced live
post-deploy  semantic_embeddings key count: 1     <- the fix, live
post-deploy  value: false                          <- honest: this IS hash fallback
             model text-embedding-ada-002, dim 768
```

`false` is the correct answer, not a failure: `HELIX_EMBEDDING_PROVIDER` is
unset, so the pipeline genuinely IS the HashEmbedder. The defect was that a
caller could not TELL. Now it can.

## No regression

| check | result |
|---|---|
| gateway `NRestarts` | 0 |
| gateway health | HTTP 200 |
| helixagent | HTTP 200 |
| containers | 12 |
| coder (30B) | untouched — MainPID=0, active |
| HXC-229 guard on the NEW process | GREEN — pid 933323, GIN_MODE=release, 0 debug in 16 lines, 1 startup marker |

The HXC-229 guard re-passing on a *different* PID is the useful part: it proves
the guard reads the live process rather than a remembered one.

## Honest boundary (§11.4.6)

This deploy also landed two CPU-inference feature commits (`08de5c5`,
`2cf2b5b`) that were never separately validated. They were already merged in
`helix_llm` main; deploying the fix could not exclude them. The gateway's own
surfaces are green, but those features carry no captured evidence of their own
and must not be reported as verified.
