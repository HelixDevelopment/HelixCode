# HXC-243 — HelixQA banks that could not fail

**Run ID:** `hxc243_bank_assertion_audit_20260808T082042Z`
**Scope:** `submodules/helix_qa` (separate repo — commits `75ed248`, `d05d825`; NOT pushed)

## The defect

`pkg/autonomous/http_executor.go:319-339` compares a field only when the bank
declares it:

```go
if step.ExpectStatus != 0 && resp.StatusCode != step.ExpectStatus { ... }
if step.ExpectBodyContains != "" && !strings.Contains(...)          { ... }
if step.ExpectJSONPath != ""                                        { ... }
```

A step declaring none of the three therefore scores PASS on **any** HTTP
response — 404, 503, a JSON-RPC error envelope. 16 banks / 72 http steps were
in exactly that state: **unfalsifiable by construction.** A §11.4.1
absence-of-error PASS living inside the QA tooling itself.

## Blast radius (measured, `audit_report.txt`)

| | |
|---|---|
| banks scanned | 129 |
| banks with `http:` steps | 40 |
| **fully unfalsifiable banks** | **16** |
| **assertion-free http steps** | **72 of 243** |

All 16 are `dispatches_to` banks — their real verdict comes from an analyzer
binary, and the `action: "http: …"` strings are prose about what that analyzer
does. `helixqa http` runs them anyway, silently degrading each into an
assertion-free poke. Services covered: llama.cpp coder `:18434`,
helixcode-server `:8081`, TEI embeddings `:18435`, NLLB translate `:18436`,
Whisper STT `:18437`, Tesseract OCR `:18438`, vision `:18439`, A2A `:18441`,
MCP gateway `:18444`.

`before_baseline_vs_7061.txt` — all 16 banks against HelixAgent `:7061`
(dead completion path): **72 PASS, 0 FAIL, EXIT=0, every bank.**

## What was added

238 of 243 http steps now carry an explicit expectation; the other 5 carry an
explicit `_skip` + reason. **0 steps remain assertion-free** (`audit_report_AFTER.txt`).

Assertions are grounded, never guessed (§11.4.6):

- **Batch 1 (8 banks / 43 steps)** — target service live, so the contract was
  **measured** (`probe_ground_truth.txt`).
- **Batch 2 (8 banks / 29 steps)** — target service down, so the contract was
  lifted **verbatim from each bank's own paired analyzer** (`ground_truth_batch2.md`).

`expect_status` alone was deliberately not treated as sufficient. `:7061`
answers `GET /v1/models` with **HTTP 200** carrying an OpenAI `{object,data}`
shape, and answers `POST /v1/embeddings` with **HTTP 200** carrying a JSON-RPC
error envelope. A status-only assertion passes both. The paired
`expect_json_path` demands a real payload and rejects both.

Two steps got an honest non-assertion rather than an invented one (§11.4.3):
`whisper` (endpoint needs multipart/form-data, which a YAML `body:` cannot
express) and `COD-RED-UNREACHABLE-001` (needs a per-case port the single
`--base-url` cannot express). Both keep their real assertion in the analyzer.

## Proof

| evidence | result |
|---|---|
| `before_baseline_vs_7061.txt` | 16/16 banks PASS on a dead target, EXIT=0 |
| `after_RED_vs_7061.txt` | batch 1: 8/8 EXIT=1, 37 FAIL, 0 PASS |
| `after_GREEN_vs_correct_target.txt` | batch 1: 8/8 EXIT=0, 37 PASS, 0 FAIL |
| `after_RED_batch2_vs_7061.txt` | batch 2: 7/8 EXIT=1, 25 FAIL; whisper 4 SKIP. 0 PASS |
| `batch2_assertion_selfvalidation.txt` | good fixture 7/7 EXIT=0; bad fixture 7/7 EXIT=1 |
| `regression_control_rerun_rag_vs_7061.txt` | **4 PASS/EXIT 0 → 4 FAIL/EXIT 1** |

Batch 1 has true bidirectional proof: the same bank fails on the broken
service and passes on the working one. An always-red test carries as little
signal as an always-green one, so both directions were required.

Batch 2 cannot have that today — its services are down. It gets the next-best
load-bearing property instead: a golden-good / golden-bad fixture pair
(§11.4.107(10), `contract_stub.py`) proving the assertions are *satisfiable*
(not accidentally always-red) and *discriminating* (they reject the exact
200-with-error-envelope shape `:7061` exhibits).

## Open gap (§11.4.6)

**Batch 2's GREEN polarity is owed a re-run against the real services** once
`:18435`–`:18444` are booted. Contract-derived and fixture-proven is not the
same as service-proven, and is not reported as such. Command:

```bash
cd submodules/helix_qa
./bin/helixqa http --bank banks/helixllm_embeddings.yaml --base-url http://127.0.0.1:18435
```

Not addressed here (separate defect): the runner lets a `dispatches_to` case
be executed as a bare http probe with no warning. Assertions close the
false-PASS hole; they do not stop the mode confusion that opened it.

## Files

| file | what it is |
|---|---|
| `audit_banks.py` | blast-radius auditor; re-runnable as a gate (`STILL assertion-free` must be 0) |
| `audit_report.txt` / `audit_report_AFTER.txt` | before / after counts |
| `probe_ground_truth.txt` | measured responses behind every batch-1 assertion |
| `ground_truth_batch2.md` | analyzer-source contracts behind every batch-2 assertion |
| `patch_batch1.py` / `patch_batch2.py` | the edits, line-addressed and auditable |
| `contract_stub.py` | golden-good/golden-bad fixture server |
