# HXC-237 — shipped workable-items binary was stale; RED->GREEN, captured 2026-08-08T09:29:04Z

## 1. RED — the shipped (pre-fix) binary is BLIND, proven on the real artifact from git
=== RED_MODE=1 — reproducing HXC-237 on the real pre-fix artifact ===
RED PASS — the pre-fix shipped binary is BLIND to an unresolvable
           closure evidence_path (blob a36be0b02f3f21969d5ffa923e32726ac94093bb). This is the
           HXC-237 defect, reproduced on the artifact that shipped.
RED_EXIT=0

## 2. GREEN — the rebuilt tracked binary carries the invariant
=== RED_MODE=0 — standing guard: the SHIPPED binary carries the invariant ===
  S1 yes — tracked binary reports the violation on the golden-bad fixture
  S2 yes — tracked binary stays silent when the evidence file exists
  S3 yes — no source file is newer than the tracked binary
GREEN PASS — the shipped workable-items binary genuinely carries the
             closure-evidence-resolvability invariant (behaviour-asserted,
             not mtime-inferred).
GREEN_EXIT=0

## 3. Live proof on a real corrupted copy of the records (one injected unresolvable evidence_path)
### pre-fix binary blob a36be0b0 (extracted from constitution git):
validate: OK — 437 items, all invariants satisfied
PREFIX_BIN_EXIT=0   <-- 0 = the bluff: reports OK on corrupted records
### rebuilt tracked binary, SAME corrupted db:
validate: 1 violation(s):
  - VEN-001: closure evidence_path does not resolve (well-formed path, but nothing exists there) — history id=392, event=Fixed, on=2026-08-08: "/tmp/hxc237-red/NO_SUCH_DIR/proof.log" (§11.4.5/§11.4.69/§11.4.123/§11.4.226 — a closure's captured proof must be producible on demand)
REBUILT_BIN_EXIT=1   <-- 1 = catches it
### control: rebuilt binary on the UNMODIFIED live records:
validate: OK — 437 items, all invariants satisfied
CONTROL_EXIT=0   <-- 0 = no false positive

## 4. Artifact identity
pre-fix blob : a36be0b02f3f21969d5ffa923e32726ac94093bb (constitution commit 159399c, 2026-07-21)
rebuilt blob : e8046c8b656af923bef3dd26f8ade413fb316b25

## 5. Gate wired into the standing suite
1078:if want_gate G29; then
