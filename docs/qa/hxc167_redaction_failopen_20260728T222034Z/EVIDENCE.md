# HXC-167 — QA transcript scrubbing could fail without anyone noticing

**Item:** HXC-167 (Bug, High)
**Fix:** `scripts/qa/lib/sec_capture_lib.sh` — transcript scrubbing block
**Guard:** `scripts/gates/qa_transcript_redaction_gate.sh` (`CM-QA-TRANSCRIPT-REDACTION-FAIL-CLOSED`)
**Anchors:** §11.4.1 / §11.4.10 / §11.4.74 / §11.4.107(10) / §11.4.115 / §11.4.135 / §11.4.146 / §11.4.201

All fixture secrets in this evidence are **synthetic** — obviously fake, structurally
realistic. No real credential appears in any artefact here, and no code path prints a
secret value (asserted by the guard, check 5).

---

## The defect (both halves confirmed as FACT before any fix)

**Half 2 — fail-open, silently. The serious one.** The redaction was:

```
sed -i "s|${p//|/\\|}|<EPHEMERAL_KEY_REDACTED>|g" "$f" 2>/dev/null || true
```

The secret was interpolated **into a regex** with only `|` escaped. A secret containing
any other metacharacter — canonically an unbalanced `[` — made sed abort with
`unterminated 's' command`, and `2>/dev/null || true` discarded **both** the message and
the exit status. The secret stayed in the transcript with no signal of any kind: a
§11.4.201 false-null inside the control that protects every other capture.

(Escaping `|` as `\|` was independently wrong — under GNU BRE `\|` is the *alternation*
operator, not an escaped literal.)

**Half 1 — allowlist-only.** Only secrets the script registered via `qa_redact` were
removed, so anything the **server** returned was written verbatim. Observable proof
already in the tree: expired Cloudflare `__cf_bm` session cookies committed verbatim in
`docs/qa/feat_cerebras_models_f8c38181_20260728T120731Z/transcripts/` (2 files).

---

## The fix

| Layer | Scope | Mechanism |
|-------|-------|-----------|
| 1 | Secrets the script registered | **Literal** awk `index()`/`substr()` splice — the secret is never a pattern — then **verified** with fixed-string `grep -F` |
| 2 | Server-originated / unknown credentials | Context-anchored scan of the **server-originated region only**, auto-redact then **re-verify** |
| 3 | Real provider key shapes, anywhere in the file | Delegates to the existing `scripts/secret_scan.sh` (**reused**, not reimplemented — §11.4.74) |

**Fail loud, never fail open.** Any layer that cannot certify the file calls
`_qa_scrub_fail`: the transcript is quarantined (removed — this run created it seconds
earlier, so nothing pre-existing is destroyed) and the process exits non-zero. A
transcript whose scrubbing status is unknown is never left on disk. There is no
`--skip-redaction` escape hatch.

The secret is passed to awk through `ENVIRON[]`, **not** `awk -v`: `-v` applies
escape-sequence processing, so a backslash-bearing secret would arrive mangled and
silently fail to match — the same bug class one layer down. Caught by the edge battery
during development and fixed before landing.

---

## Verdicts

| Artefact | What it proves | Result |
|----------|----------------|--------|
| `red_capture.txt` | Both halves reproduce on the **pre-fix artifact recovered from git history** (commit `1cc1619d`), with a working negative control | **RED PASS** (exit 0) |
| `green_capture.txt` | 14 invariants hold on the fixed artifact | **GREEN PASS** (exit 0) |
| `falsifiability_proof.txt` | The gate **fails** against a reverted library — and still fails with both source-greps disabled, so **7 checks detect the defect by observed behaviour alone** | **exit 1 both runs** (as required) |
| `abort_propagation.txt` | A failed scrub terminates the whole **capture script**, not just a subshell, and quarantines the file | **exit 1, file removed** |
| `edge_case_battery.txt` | 17 metacharacter/edge secrets: none survives at exit 0; no value leaks to output | **no fail-open** |
| `fp_calibration_shipped.txt` | Shipped scan vs all 125 committed transcripts | **2 findings, 2 true positives, 0 false positives** |
| `fp_calibration_entropy_rejected.txt` | The context-free entropy alternative, measured then rejected | **94.3% FP at its best threshold** |

Falsifiability matters more than the GREEN count: with the two source-grep checks
disabled, the reverted library still trips **7 independent behavioural checks** —
metacharacter survival, the battery, the server-originated cookie, the unregistered
provider key, the silent exit-0, non-propagating abort, and the un-quarantined file.
The guard cannot pass vacuously.

### RED → GREEN, the decisive case
A secret containing an unbalanced `[` **survived** redaction with no signal on the
pre-fix artifact, and is **removed** on the fixed artifact. Same test source, polarity
switched by `RED_MODE` (§11.4.115). The negative control confirms the pre-fix path *did*
redact a plain secret, so the survival is the metacharacter and not a broken harness.

---

## False-positive rate — measured, not assumed

The task's honest boundary was correct: strictness risks noise, and a scanner that cries
wolf gets disabled, which is worse than none (§11.4.201). Both candidate designs were
calibrated against this repo's own 125 committed transcripts (4,761 lines).

**Shipped (context-anchored, server-originated region only): 0 false positives.**
2 findings, both real — the two `__cf_bm` cookies.

Getting there required fixing two problems the measurement exposed:

1. **Unscoped, the scan produced 27 findings of which only 2 were real — a 92.6% FP
   rate.** All 25 false positives were the deliberately *invalid* probe credentials the
   security captures **send** to prove they are rejected (`Bearer wrong-key-qa-probe`,
   `x-api-key: qa-probe-not-a-real-key`). Those are not secrets — they are the test
   input, and their visibility in the transcript *is* the evidence. The discriminator
   was perfectly clean: all 2 real findings on `<` (received) lines, all 25 FPs on `>`
   (sent) lines or the `### curl args:` echo. Scoping Layer 2 to the server-originated
   region takes FP to 0 while keeping every true positive.

2. **The inherited allowlist was itself a fail-open hole.** `scripts/secret_scan.sh`
   treats `example` as a suppression marker — right for hand-written docs, wrong for HTTP
   transcripts, where `Domain=api.example.test` or `Host: example.com` would blanket-
   suppress a real cookie on the same line. Layer 2's allowlist is deliberately narrowed
   to the redaction marker only.

**Rejected alternative — context-free high-entropy scan.** Measured across three
thresholds:

| Threshold | Transcripts flagged | Token hits | True positives | **FP rate** |
|-----------|--------------------|-----------|----------------|-------------|
| 3.5 bits/char | 125 / 125 | 860 | 2 | **99.8%** |
| 4.0 bits/char | 65 / 125 | 135 | 2 | **98.5%** |
| 4.5 bits/char | 17 / 125 | 35 | 2 | **94.3%** |

Even at its best threshold it flags 33 benign tokens for every 2 real ones — model paths
(`/models/Qwen3-Coder-30B-A3B-…`), LLM response ids, and at 3.5 even header *names* like
`Access-Control-Allow-Headers`. Unusable; not shipped. Context is the signal here,
entropy is not.

---

## Two fail-opens found *inside* the fix, before it landed

The nine-pattern end-to-end probe (`nine_pattern_probe.txt`) caught the fix
reproducing the very defect class it exists to close. Recorded because "the fix was
clean first try" would be the bluff:

1. **Greedy per-line extraction redacted only the LAST credential on a line.** The
   value-extract expressions lead with `.*`, which is greedy, so on
   `{"access_token":"…","refresh_token":"…"}` only `refresh_token` was extracted.
   `access_token`, `client_secret` and `X-Amz-Signature` all survived.
2. **A line-level allowlist then HID the survivor.** Redacting the second credential
   put `<EPHEMERAL_KEY_REDACTED>` on that line; the re-scan's "line contains REDACTED"
   allowlist therefore skipped the whole line, so the surviving credential was never
   reported and the run **exited 0**.

Together those were two independent ways to leak a secret while reporting success —
the same shape as the original defect. Both are now closed structurally rather than by
patching symptoms: matches are isolated with `grep -oE` before extraction (no greedy
ambiguity to lose), and there is **no line-level allowlist at all** — termination is
guaranteed instead by the redaction marker starting with `<`, which no pattern's value
character class admits. Both are pinned by gate check (3c).

## Honest boundary (§11.4.6)

- **0 false positives is measured on *this* corpus** — 125 transcripts, 4,761 lines, from
  a small set of providers. It is not a claim about transcripts this repo has never seen.
  A new provider returning an unusual header shape could produce a first FP; the auto-
  redact-then-reverify path degrades to a redacted value plus a stderr notice rather than
  a dead run, which is the tolerable failure direction.
- **Residual coverage gap:** a real credential with a *non-standard shape* (matching no
  Layer 3 provider pattern) that a capture script **sends** without registering it via
  `qa_redact` is caught by none of the three layers. The control for that is `qa_redact`
  itself. Stated rather than papered over.
- **True photon-level assurance is not claimed.** These layers prove that registered
  secrets are gone and that known credential shapes in the server-originated region are
  gone. They do not prove a transcript contains no secret of a shape nobody has modelled.
- **The two `__cf_bm` cookies already committed are NOT rewritten by this change.**
  They are pre-existing evidence files owned by another work stream; the calibration was
  strictly read-only (confirmed by `git status`). They are expired Cloudflare bot-
  management cookies, not access credentials. Re-running those captures now scrubs them.
