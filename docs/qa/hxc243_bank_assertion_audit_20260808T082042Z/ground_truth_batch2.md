# HXC-243 batch-2 ground truth — contracts read from analyzer source

The 8 batch-2 banks target services that were DOWN during this run
(`:18435`–`:18444`), so their request/response contract could not be measured
live. It was instead read out of each bank's **own paired analyzer** — the
binary that genuinely talks to that service today, and whose assertions are
already evidence-backed in the banks' prior QA records.

Every claim below carries a `file:line` citation into
`submodules/helix_qa/cmd/`. Where the contract could not be determined from
source it is recorded as UNDETERMINED rather than guessed (§11.4.6).

| # | bank → analyzer | path | request body | healthy status | success-only JSON path | encoding |
|---|---|---|---|---|---|---|
| 1 | `helixllm_embeddings` → `verify-embeddings` | `/v1/embeddings` | `{"model":"helix-embed","input":[…]}` | 200 | `$.data[0].embedding` | JSON |
| 2 | `helixllm_rag` → `verify-rag` | `/v1/embeddings` + `/v1/chat/completions` | embed `{"model":"helix-embed","input":[…]}`; chat `{"model":"coder","temperature":0,"max_tokens":200,"messages":[…]}` | 200 both | `$.data[0].embedding`; `$.choices[0].message.content` | JSON |
| 3 | `helixllm_tesseract` → `verify-tesseract` | `/v1/render` → `/v1/ocr` | render `{"text":…,"mode":"label","pointsize":48}` | 200 | render returns **binary PNG, not JSON** — no path exists; ocr `$.full_text` | render JSON req / binary resp; ocr raw octet-stream req |
| 4 | `helixllm_translate_nllb` → `verify-translate-nllb` | `/translate` | `{"q":…,"source":"eng_Latn","target":"deu_Latn"}` | 200 | `$.translatedText` | JSON |
| 5 | `helixllm_whisper` → `verify-whisper` | `/v1/audio/transcriptions` | multipart field `file` = raw WAV | 200 | `$.text` | **multipart — NOT expressible as YAML `body:`** |
| 6 | `helixllm_vision` → `verify-vision` | `/v1/chat/completions` | OpenAI multimodal `content` array w/ inline base64 image | 200 | `$.choices[0].message.content` | JSON |
| 7 | `helixllm_a2a` → `verify-a2a` | **`/a2a`** | `{"jsonrpc":"2.0","id":1,"method":"message/send","params":{"message":{"role":"user","parts":[{"kind":"text","text":…}]}}}` | 200 | `$.result.status.state` | JSON |
| 8 | `helixllm_mcp_gateway` → `verify-mcp-gateway` | **`/`** (endpoint root) | `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, no `Authorization` | **401** | n/a — status is the only signal | JSON |

## Citations

1. **embeddings** — default endpoint `http://localhost:18435/v1/embeddings`
   `verify-embeddings/main.go:127`; request `main.go:163`; non-200 = infra
   error `main.go:184`; response struct `main.go:63-70`.
2. **rag** — embed endpoint `main.go:150`, body `main.go:307`, status check
   `main.go:316`; chat endpoint `main.go:151`, body `main.go:327`, status
   check `main.go:336`; response structs `main.go:92-95`, `:107-114`.
3. **tesseract** — base URL `main.go:103`; `POST /v1/render` `main.go:151-152`;
   render response read as raw bytes via `io.ReadAll` `main.go:156-161`
   (never JSON-decoded); `POST /v1/ocr` `main.go:164` with
   `Content-Type: application/octet-stream`; response `main.go:68-73,175-178`.
4. **translate** — endpoint `main.go:100`; request `main.go:138` (keys
   `q`/`source`/`target`); status `main.go:159-160`; response `main.go:63-65`.
5. **whisper** — endpoint `main.go:122`; multipart built `main.go:162-173`,
   form field `file`, content-type `main.go:181`; status `main.go:196`;
   response `main.go:57-62`.
6. **vision** — endpoint + model `main.go:165-166`; request `main.go:211-224`
   (base64 image inlined in JSON, not multipart); check `main.go:250`;
   response `main.go:110-128`.
7. **a2a** — `flag.String("endpoint", envOr("HELIX_A2A_ENDPOINT",
   "http://localhost:18441/a2a"), …)` **`main.go:146`** — the real path `/a2a`
   is baked into the default flag value. Request `main.go:184-191`; non-200 =
   infra error `main.go:214`; envelope `main.go:105-110`; task struct
   `main.go:80-91`; success check `main.go:231-232`. A JSON-RPC error reply
   carries `$.error` in place of `$.result`, so asserting
   `$.result.status.state` discriminates a real task from an error envelope
   returned with HTTP 200.
8. **mcp-gateway** — `flag.String("endpoint", envOr(
   "HELIX_MCP_GATEWAY_ENDPOINT", "http://localhost:18444"), …)`
   **`main.go:117`** — the default carries **no path suffix**, and the
   unauth case posts to it directly (`main.go:215`), so the real path is `/`.
   Body literal `main.go:216`; no `Authorization` header is set anywhere in
   `runUnauth401`; the analyzer's own assertion is
   `v.Pass = resp.StatusCode == http.StatusUnauthorized` **`main.go:230`**.

## UNDETERMINED (recorded, not guessed)

- **mcp-gateway** `toolslist` / `generate` / `listmodels` cases open a real
  MCP session via `mcp.NewClient(...).Connect(...)` over Streamable-HTTP
  (`main.go:364-373`). The SDK owns the `initialize` handshake, session-id
  negotiation and JSON-RPC framing; the analyzer never constructs a
  hand-rollable body for them, and records no `http_status` (only
  `unauth401` populates `v.HTTPStatus`, `main.go:228`). No single-step
  `body`/`expect_json_path` can express them. Those 5 cases have no `http:`
  step and are reported SKIP by the runner — they were never part of the 72.
- **mcp-gateway** 401 response body is captured as an opaque JSON-quoted
  string (`main.go:229`), not parsed as an object, so no `expect_json_path`
  is available there — the status code is the only assertable signal, and
  that is exactly what the bank now asserts.

Source: read in full, no edits, no services started — all 8 analyzer
`main.go` files under `submodules/helix_qa/cmd/`.
