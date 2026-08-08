#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""HXC-243 batch-2 assertion self-validation stub (§11.4.107(10)).

The 8 batch-2 banks target services that are DOWN, so their GREEN polarity
cannot be proven against the real thing. This stub proves the next-best,
still-load-bearing property: that the assertions I wrote are SATISFIABLE
and DISCRIMINATING — i.e. they are not accidentally always-red, and they
do reject a plausible near-miss.

Two modes, one server (the golden-good / golden-bad fixture pair):

  good : serves exactly the response shape each bank's paired analyzer
         decodes.  Banks MUST pass.
  bad  : serves HTTP 200 for every route with an error envelope in the
         body — the precise bluff shape that HelixAgent :7061 exhibits on
         /v1/embeddings, and the one a status-only assertion cannot catch.
         Banks MUST fail.

If the banks pass in BOTH modes, my assertions are worthless and this
script says so by failing the pair.

Usage: contract_stub.py <port> <good|bad>
"""
import json
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer

# 1x1 PNG — real magic bytes, so expect_body_contains "PNG" is a genuine
# content check on the tesseract render leg rather than a status-only pass.
PNG = (b"\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01"
       b"\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00\nIDATx\x9cc\x00"
       b"\x01\x00\x00\x05\x00\x01\r\n-\xb4\x00\x00\x00\x00IEND\xaeB`\x82")

GOOD = {
    "/v1/embeddings": (200, {"model": "helix-embed",
                             "data": [{"embedding": [0.11, 0.22, 0.33]}]}),
    "/v1/chat/completions": (200, {"choices": [
        {"index": 0, "message": {"role": "assistant", "content": "PONG"}}]}),
    "/translate": (200, {"translatedText": "Das Haus ist blau."}),
    "/a2a": (200, {"jsonrpc": "2.0", "id": 1,
                   "result": {"id": "task-1",
                              "status": {"state": "completed",
                                         "timestamp": "2026-08-08T00:00:00Z"}}}),
    "/": (401, {"error": "unauthorized"}),
}

# Every route answers 200 with a JSON-RPC/OpenAI-style ERROR envelope: the
# canonical "absence of error is not evidence of success" trap.
BAD_BODY = {"jsonrpc": "2.0",
            "error": {"code": -32700, "message": "Parse error", "data": "EOF"}}


class Handler(BaseHTTPRequestHandler):
    mode = "good"

    def do_POST(self):  # noqa: N802
        path = self.path.split("?")[0]
        if self.mode == "bad":
            return self._json(200, BAD_BODY)
        if path == "/v1/render":
            self.send_response(200)
            self.send_header("Content-Type", "image/png")
            self.send_header("Content-Length", str(len(PNG)))
            self.end_headers()
            self.wfile.write(PNG)
            return
        status, body = GOOD.get(path, (404, {"error": "no such route"}))
        return self._json(status, body)

    def do_GET(self):  # noqa: N802
        return self.do_POST()

    def _json(self, status, body):
        raw = json.dumps(body).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)

    def log_message(self, *_args):
        pass


if __name__ == "__main__":
    port, mode = int(sys.argv[1]), sys.argv[2]
    Handler.mode = mode
    HTTPServer(("127.0.0.1", port), Handler).serve_forever()
