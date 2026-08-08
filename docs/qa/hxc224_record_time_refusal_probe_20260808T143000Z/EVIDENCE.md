# HXC-224 — REFUSED CLOSURE. The record-time refusal does NOT exist.
Captured 2026-08-08T09:31:07Z

HXC-224 asks for three things. Two exist, one does not:
  (a) the pointer must resolve            -> EXISTS at validate time (HXC-217)
  (b) attached only to real completions   -> EXISTS at validate time (HXC-217)
  (c) REFUSED AT THE MOMENT OF RECORDING  -> ABSENT (proven below)

## Probe: close an item with an evidence path that does not exist
$ workable-items close HXC-159 --db <copy> --status completed \
      --evidence /tmp/DEFINITELY_NOT_A_REAL_PATH/x.log
close: moved HXC-159 Issues→Fixed (status=Completed (→ Fixed.md), evidence=/tmp/DEFINITELY_NOT_A_REAL_PATH/x.log)
CLOSE_EXIT=0   <-- ACCEPTED. Not refused.

## The mutation actually landed
post-close status of HXC-159 in the probe copy: Completed (→ Fixed.md)

## Why this matters
The rot HXC-224 describes is still creatable at will. The HXC-217 validator
catches it only AFTER the fact, on the next validate run -- which is precisely
the 'discovered months later' failure mode HXC-224 was filed to eliminate.
Closing HXC-224 on the strength of the HXC-217 validator would conceal the
unimplemented half.

## Scope note
This probe ran against a COPY (/tmp/hxc224-probe.db). The live records were
not mutated by it.

## Fix direction (not implemented here)
Reject a non-resolving --evidence in the close/move/diary write paths
(constitution/scripts/workable-items/cmd/workable-items/), reusing the SAME
resolveInvocationRelative + os.Stat predicate the validator already uses, so
record-time and validate-time cannot drift apart. Needs a RED-first test that
reproduces the acceptance above, plus a binary rebuild (see HXC-237).
