#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""HXC-243 batch-1 patcher: add explicit HTTP expectations to the
assertion-free `http:` steps of the 8 banks whose target service is LIVE
and therefore empirically verifiable in BOTH directions (GREEN against the
correct service, RED against a broken one).

Every assertion below is grounded in a MEASURED response, not a guess
(§11.4.6). Measurements taken 2026-08-08 and recorded in
probe_ground_truth.txt beside this script:

  :18434 llama.cpp coder (Qwen3-Coder-30B)
      POST /v1/chat/completions {model,messages,max_tokens,temperature}
          -> 200, body has $.choices[0].message.content   (60 ms)
      GET  /v1/models
          -> 200, body has $.models[0].name
  :8081  helixcode-server
      GET  /api/v1/llm/providers -> 200, body has $.providers[0].id
  :7061  HelixAgent (BROKEN control target)
      POST /v1/chat/completions {model:"local",...} -> 404 model_not_found
      GET  /v1/models  -> 200 BUT shape is {object,data} — NO `models` key
      GET  /api/v1/llm/providers -> 404

The `$.models[0].name` / `$.providers[0].id` json-path assertions are
deliberately chosen to also kill the *200-with-wrong-shape* bluff mode
that a bare status check would miss (:7061 answers /v1/models with 200).

Line-based insertion (not YAML round-trip) so the banks' extensive
governance comments survive byte-identical. Processed bottom-up per file
so earlier line numbers stay valid.
"""
import os
import re
import sys

# A real, minimal completion request. max_tokens is tiny so this is a
# genuine generation the resident 30B answers in ~60 ms — it neither
# hammers nor restarts the coder (§11.4.119 read-only shared resource).
CHAT = """\
{ind}body:
{ind}  model: "local"
{ind}  messages:
{ind}    - role: "user"
{ind}      content: "HXC-243 endpoint-contract probe: reply with the single word PONG"
{ind}  max_tokens: 8
{ind}  temperature: 0
{ind}# HXC-243: an explicit expectation is what makes this step falsifiable.
{ind}# 200 alone is NOT enough — a wrong service can answer 200 with an error
{ind}# envelope, so the json_path additionally demands a REAL completion body.
{ind}expect_status: 200
{ind}expect_json_path: "$.choices[0].message.content"
"""

MODELS = """\
{ind}# HXC-243: /v1/models must return THIS service's model roster. The
{ind}# json_path is load-bearing: HelixAgent :7061 also answers 200 here but
{ind}# with an OpenAI {{object,data}} shape that has no `models` key, so a
{ind}# status-only assertion would still pass against the wrong service.
{ind}expect_status: 200
{ind}expect_json_path: "$.models[0].name"
"""

PROVIDERS = """\
{ind}# HXC-243: explicit expectation — the providers roster must come back
{ind}# populated, not merely "some HTTP response arrived".
{ind}expect_status: 200
{ind}expect_json_path: "$.providers[0].id"
"""

SKIP_UNREACHABLE = """\
{ind}# HXC-243: this case's whole point is a DIFFERENT port (18433) that must
{ind}# be refused. `helixqa http` drives one operator-supplied --base-url, so
{ind}# the per-case target cannot be expressed here. An honest SKIP (§11.4.3)
{ind}# is correct; the connection-refused polarity is owned by the
{ind}# dispatches_to analyzer, which does control the port.
{ind}_skip: true
{ind}_skip_reason: >-
{ind}  HXC-243/§11.4.3: case targets port 18433 (must be connection-refused);
{ind}  the http runner drives a single --base-url and cannot express a
{ind}  per-case port. Polarity is asserted by bin/helixqa-verify-coder
{ind}  (dispatches_to), not by this http step.
"""

# file -> {action-line-number (1-based, pre-patch): block}
SPEC = {
    "helixllm_coder.yaml": {
        128: SKIP_UNREACHABLE,
        163: CHAT, 196: CHAT, 236: MODELS, 271: CHAT, 310: CHAT,
    },
    "helixllm_coder_bench.yaml": {
        129: CHAT, 161: CHAT, 194: CHAT, 231: CHAT, 265: CHAT, 302: CHAT, 343: CHAT,
    },
    "helixllm_coder_concurrency.yaml": {
        114: CHAT, 147: CHAT, 182: CHAT, 216: CHAT, 252: CHAT, 291: CHAT,
    },
    "helixllm_coder_memory.yaml": {
        126: CHAT, 160: CHAT, 193: CHAT, 228: CHAT, 268: CHAT,
    },
    "helixllm_coder_race.yaml": {
        164: CHAT, 201: CHAT, 235: CHAT, 275: CHAT, 302: CHAT,
    },
    "helixllm_coder_chaos.yaml": {
        188: CHAT, 225: CHAT,
    },
    "helixllm_network_provider.yaml": {
        144: CHAT, 173: CHAT, 200: CHAT, 244: CHAT, 287: CHAT, 319: CHAT, 361: CHAT,
    },
    "helixcode_coder_race.yaml": {
        119: PROVIDERS, 151: PROVIDERS, 183: PROVIDERS, 234: PROVIDERS, 291: PROVIDERS,
    },
}

# helixllm_coder.yaml writes absolute URLs in the action. The runner does
# `url = BaseURL + "/" + path`, so an absolute URL produces a mangled
# request path — the step could never assert anything real. Rewrite to the
# relative path the runner actually supports; the host comes from
# --base-url (CONST-051(B): never hardcode a target in bank data).
ABS_URL = re.compile(r'http://localhost:\d+(/[^\s"]*)')


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
            block = spec[lineno].format(ind=ind)
            lines.insert(idx + 1, block)
            total += 1

        if fname == "helixllm_coder.yaml":
            for i, line in enumerate(lines):
                if 'action: "http:' in line and "localhost:" in line:
                    lines[i] = ABS_URL.sub(r"\1", line)

        with open(path, "w", encoding="utf-8") as fh:
            fh.writelines(lines)
        print(f"patched {fname}: {len(spec)} steps")

    print(f"\ntotal http steps given explicit expectations: {total}")


if __name__ == "__main__":
    main()
