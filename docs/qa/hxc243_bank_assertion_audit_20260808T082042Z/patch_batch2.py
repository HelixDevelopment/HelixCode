#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""HXC-243 batch-2 patcher: the 8 banks whose target service is DOWN
(:18435-:18441, :18444).

HONEST BOUNDARY (§11.4.6). Batch 1's assertions were verified in BOTH
polarities against live services. These cannot be: the services are not
running, so only the RED direction (they now FAIL against a broken
target) is empirically proven here. The GREEN direction is therefore
grounded in the strongest available substitute — the request/response
contract read verbatim out of each bank's OWN paired analyzer, i.e. the
binary that genuinely talks to that service today. Citations are recorded
per assertion in ground_truth_batch2.md beside this script. GREEN
re-verification is owed the moment the services boot; it is reported as
an open gap, not silently implied.

Two steps are deliberately NOT given assertions, because inventing one
would be the same bluff class this ticket exists to remove:
  - whisper: the analyzer posts multipart/form-data (form field `file` =
    raw WAV). A YAML `body:` cannot express multipart, and a bodyless
    POST to a healthy STT service does NOT return 200 — so any
    expect_status here would be fiction. Honest _skip (§11.4.3).
  - tesseract /v1/render: answers with raw PNG bytes, not JSON, so no
    json_path exists. It still gets expect_status + a PNG magic-byte
    body assertion, which is real content evidence rather than a bare
    status check.
"""
import os
import sys

EMBED = """\
{ind}# HXC-243: body + expectations mirror cmd/helixqa-verify-embeddings
{ind}# (main.go:163 request, main.go:63-70 response). $.data[0].embedding
{ind}# exists only in a genuine TEI payload — an error envelope has no
{ind}# `data` array, so this also kills the 200-with-error-body bluff.
{ind}body:
{ind}  model: "helix-embed"
{ind}  input:
{ind}    - "HXC-243 endpoint-contract probe"
{ind}expect_status: 200
{ind}expect_json_path: "$.data[0].embedding"
"""

CHAT = """\
{ind}# HXC-243: body + expectations mirror the analyzer's chat call
{ind}# (cmd/helixqa-verify-rag/main.go:327; cmd/helixqa-verify-vision
{ind}# /main.go:211-224 use the same OpenAI chat contract). The json_path
{ind}# demands a real completion, not merely "some HTTP response arrived".
{ind}body:
{ind}  model: "coder"
{ind}  messages:
{ind}    - role: "user"
{ind}      content: "HXC-243 endpoint-contract probe: reply with the single word PONG"
{ind}  max_tokens: 8
{ind}  temperature: 0
{ind}expect_status: 200
{ind}expect_json_path: "$.choices[0].message.content"
"""

RENDER = """\
{ind}# HXC-243: mirrors cmd/helixqa-verify-tesseract/main.go:151-152. The
{ind}# render response is raw PNG bytes (main.go:156-161), never JSON, so
{ind}# no json_path exists — the PNG magic-byte assertion is the real
{ind}# content evidence in its place, not a bare status check.
{ind}body:
{ind}  text: "HXC-243"
{ind}  mode: "label"
{ind}  pointsize: 48
{ind}expect_status: 200
{ind}expect_body_contains: "PNG"
"""

TRANSLATE = """\
{ind}# HXC-243: body + expectations mirror cmd/helixqa-verify-translate-nllb
{ind}# (main.go:138 request keys q/source/target, main.go:63-65 response).
{ind}expect_status: 200
{ind}expect_json_path: "$.translatedText"
{ind}body:
{ind}  q: "The house is blue."
{ind}  source: "eng_Latn"
{ind}  target: "deu_Latn"
"""

A2A = """\
{ind}# HXC-243: mirrors cmd/helixqa-verify-a2a/main.go:184-191. A JSON-RPC
{ind}# error reply carries $.error instead of $.result, so asserting
{ind}# $.result.status.state discriminates a real task from an error
{ind}# envelope returned with HTTP 200.
{ind}body:
{ind}  jsonrpc: "2.0"
{ind}  id: 1
{ind}  method: "message/send"
{ind}  params:
{ind}    message:
{ind}      role: "user"
{ind}      parts:
{ind}        - kind: "text"
{ind}          text: "HXC-243 endpoint-contract probe"
{ind}expect_status: 200
{ind}expect_json_path: "$.result.status.state"
"""

MCP_401 = """\
{ind}# HXC-243: mirrors cmd/helixqa-verify-mcp-gateway/main.go:216 (body)
{ind}# and main.go:230 (the analyzer's own assertion:
{ind}# Pass = StatusCode == http.StatusUnauthorized). Asserting 401 makes
{ind}# the case genuinely falsifiable in the direction that matters: a
{ind}# gateway that ever lets an UNAUTHENTICATED tools/list through with
{ind}# 200 now FAILS this step instead of passing it.
{ind}body:
{ind}  jsonrpc: "2.0"
{ind}  id: 1
{ind}  method: "tools/list"
{ind}expect_status: 401
"""

WHISPER_SKIP = """\
{ind}# HXC-243/§11.4.3: NOT assertable in http mode, and saying so is the
{ind}# honest answer. cmd/helixqa-verify-whisper/main.go:162-181 posts
{ind}# multipart/form-data (form field `file` = raw WAV bytes); a YAML
{ind}# `body:` cannot express multipart, and a bodyless POST to a healthy
{ind}# STT service does not return 200 — so any expect_status here would
{ind}# be invented. The real assertion stays with the dispatches_to
{ind}# analyzer, which does send the audio.
{ind}_skip: true
{ind}_skip_reason: >-
{ind}  HXC-243/§11.4.3: endpoint requires multipart/form-data (WAV upload);
{ind}  the http-step schema cannot express a multipart body, so no honest
{ind}  expectation exists. Asserted instead by bin/helixqa-verify-whisper
{ind}  (dispatches_to).
"""

SPEC = {
    "helixllm_embeddings.yaml": {106: EMBED, 139: EMBED, 177: EMBED},
    # Only the FIRST method+path of a compound action string is executed by
    # the runner (parseMethodPath takes fields[0]/fields[1]), so the three
    # "embeddings then chat" steps assert the embeddings leg; the fourth
    # step's action is a bare chat call and asserts the chat leg.
    "helixllm_rag.yaml": {112: EMBED, 138: EMBED, 171: EMBED, 210: CHAT},
    "helixllm_tesseract.yaml": {110: RENDER, 136: RENDER, 167: RENDER, 205: RENDER},
    "helixllm_translate_nllb.yaml": {
        120: TRANSLATE, 147: TRANSLATE, 181: TRANSLATE, 218: TRANSLATE},
    "helixllm_whisper.yaml": {
        112: WHISPER_SKIP, 137: WHISPER_SKIP, 170: WHISPER_SKIP, 206: WHISPER_SKIP},
    "helixllm_vision.yaml": {
        121: CHAT, 147: CHAT, 171: CHAT, 197: CHAT, 237: CHAT, 274: CHAT},
    "helixllm_a2a.yaml": {120: A2A, 154: A2A, 191: A2A},
    "helixllm_mcp_gateway.yaml": {131: MCP_401},
}

# The two placeholder paths are not paths at all — the runner requests a
# literal `/<a2a-endpoint>`, which 404s on every service and therefore
# passed unconditionally. Replace them with the analyzer's real compiled-in
# defaults: `/a2a` (verify-a2a/main.go:146) and `/` (verify-mcp-gateway
# /main.go:117 — that default carries no path suffix).
PATH_FIXES = {
    "helixllm_a2a.yaml": [("<a2a-endpoint>", "/a2a")],
    "helixllm_mcp_gateway.yaml": [("<mcp-gateway-endpoint>", "/")],
}


def main():
    banks_dir = sys.argv[1]
    total = 0
    for fname, spec in SPEC.items():
        path = os.path.join(banks_dir, fname)
        with open(path, "r", encoding="utf-8") as fh:
            lines = fh.readlines()

        for lineno in sorted(spec, reverse=True):
            idx = lineno - 1
            line = lines[idx]
            if 'action: "http:' not in line:
                raise SystemExit(
                    f"{fname}:{lineno} is not an http action line: {line!r}")
            ind = " " * (len(line) - len(line.lstrip()))
            lines.insert(idx + 1, spec[lineno].format(ind=ind))
            total += 1

        for old, new in PATH_FIXES.get(fname, []):
            for i, line in enumerate(lines):
                if 'action: "http:' in line and old in line:
                    lines[i] = line.replace(old, new)

        with open(path, "w", encoding="utf-8") as fh:
            fh.writelines(lines)
        print(f"patched {fname}: {len(spec)} steps")

    print(f"\ntotal batch-2 http steps given an explicit disposition: {total}")


if __name__ == "__main__":
    main()
