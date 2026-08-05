#!/usr/bin/env python3
"""HXC-217 remediation — repoint closure-evidence pointers at their REAL artefacts.

Every change below is justified by a referent this session PROVED producible
(git-tracked file, or submodule-tracked file reachable from the main repo's
committed submodule pointer). Narrative text displaced out of `evidence_path`
is preserved verbatim in the new additive `item_history.note` column — nothing
is discarded (§9.2 / §11.4.124).

Run from the repository root.
"""
import sqlite3
import sys

DB = "docs/workable_items.db"

# --- (1) the four HXC-107/108/112 rows -------------------------------------
# These are event_type='Updated', reason='operator-blocked' PROGRESS NOTES from
# 2026-06-23 — NOT closure claims. Each item's ACTUAL closure event (h7/h6/h5,
# 'Completed', 2026-07-11) already carries a cleanly-resolving evidence_path
# (docs/qa/hxc_residuals_operator_closure_20260711/DECISION.md, tracked).
# Their narrative is ALREADY preserved verbatim in operator_block_details.what,
# so moving it to `note` and clearing the path loses nothing. An in-progress
# note has no captured evidence; asserting one would be the bluff.
CLEAR_TO_NOTE = [1, 2, 3, 4]

# --- (2) path-drift rows ----------------------------------------------------
# Real, substantive, git-tracked evidence EXISTS — it was written to
# scratch/discovery/fixes/ and the DB recorded a docs/qa/... path that was never
# populated. Found via basename search of full git history; the earlier
# "never existed" verdict came from searching the recorded (wrong) full path.
REPOINT = {
    157: ("scratch/discovery/fixes/HXC124_evidence.md",
          "HIGH — 13506-byte tracked artefact, titled 'HXC-124 ... evidence'."),
    158: ("scratch/discovery/fixes/HXC131_evidence.md",
          "HIGH — 9207-byte tracked artefact, titled 'HXC-131 ... evidence'."),
    159: ("scratch/discovery/fixes/HXC131_evidence.md",
          "HIGH — duplicate history row for the same closure."),
    160: ("scratch/discovery/fixes/HXC132_evidence.md",
          "HIGH — 11301-byte tracked artefact, titled 'HXC-132 — Evidence'."),
    # --- (3) prose closure rows: narrative -> note, real path -> evidence_path
    260: ("helix_code/applications/harmony_os/gui_thread_race_test.go",
          "MEDIUM — the cited RED/GREEN polarity test IS the artefact and is "
          "tracked (sibling aurora_os test likewise; commits 9afc3da2 + "
          "8f072866 real). IMPRECISION: no captured run-log was committed; the "
          "race counts quoted in the narrative are re-derivable only by "
          "re-running the test."),
    272: ("docs/scripts/fixed_h2_pipe_row_parity_gate.md",
          "MEDIUM — gate DOC, not a runtime capture. Commits 0a4df699 / "
          "34afaf11 / 8494380a are real. IMPRECISION: the narrative refutes the "
          "ticket's own stated premise; no captured log accompanies it."),
    286: ("submodules/helix_agent/docs/qa/HXC-165/EVIDENCE.md",
          "HIGH — tracked in the helix_agent submodule at 5a7a819f, which is a "
          "PROVEN ANCESTOR of the main repo's committed pointer 9a50509d, so a "
          "fresh recursive clone produces it."),
    312: ("docs/qa/hxc167_redaction_failopen_20260728T222034Z",
          "HIGH — self-cited in the narrative; tracked (10 files)."),
    285: ("submodules/helix_agent/docs/qa/HXC-165/EVIDENCE.md",
          "MEDIUM — HXC-182 has NO artefact of its own; it shares HXC-165's "
          "EVIDENCE.md section 8. Producible, but not a dedicated capture."),
    290: ("docs/qa/racefix_shell_grace_fa1b52b3_20260728T232856Z",
          "MEDIUM — tracked (14 files). IMPRECISION: this directory certifies "
          "commit fa1b52b3 (round 2) while the history row describes commit "
          "a7d8dbb5 (round 1). Both commits are real; the capture does not "
          "cover the commit the row names."),
    320: ("docs/qa/hxc203_localllm_race_20260729T000014Z",
          "HIGH — tracked (13 files); commits b2215793 / be3e6605 / 1c2e7d07 "
          "are real."),
}

# --- (4) HXC-187: the ONE genuine gap --------------------------------------
# Its only producible referents are COMMITS (09a086a6, cc339fc0). No captured
# artefact exists anywhere: zero tracked paths and zero on-disk directories
# match hxc187. A commit reference is not captured runtime evidence under
# §11.4.5 / §11.4.123, so this closure's warrant cannot be produced on demand.
HXC187_NOTE = (
    "HXC-217 finding: this closure's evidence_path held a narrative citing "
    "commit 09a086a6 (go.mod, 1 file, +1/-1) plus unlogged build/vet claims. "
    "The commits are real, but NO captured artefact exists — a repo-wide search "
    "for any tracked path or on-disk directory matching 'hxc187' returns "
    "nothing. Commit-reference-only is not captured evidence (§11.4.5 / "
    "§11.4.123), so the closure cannot be produced on demand. Original "
    "narrative preserved verbatim below.\n\n--- original evidence_path ---\n")


def main() -> int:
    con = sqlite3.connect(DB)
    cur = con.cursor()

    cols = [c[1] for c in cur.execute("PRAGMA table_info(item_history)")]
    if "note" not in cols:
        cur.execute("ALTER TABLE item_history ADD COLUMN note TEXT")
        print("schema: added additive nullable item_history.note")
    else:
        print("schema: item_history.note already present")

    # (1) progress notes -> note, path cleared
    for hid in CLEAR_TO_NOTE:
        old = cur.execute(
            "SELECT evidence_path FROM item_history WHERE id=?", (hid,)).fetchone()[0]
        if old is None:
            print(f"  h{hid}: already cleared, skipping")
            continue
        cur.execute(
            "UPDATE item_history SET note=?, evidence_path=NULL WHERE id=?",
            ("HXC-217: not a closure event (event_type='Updated', "
             "reason='operator-blocked'). This narrative is a progress note, "
             "not captured evidence, and is preserved verbatim in "
             "operator_block_details.what. The item's real closure event "
             "resolves to docs/qa/hxc_residuals_operator_closure_20260711/"
             "DECISION.md.\n\n--- original evidence_path ---\n" + old, hid))
        print(f"  h{hid}: evidence_path -> NULL, narrative -> note")

    # (2)+(3) repoint at the proven artefact
    for hid, (path, why) in REPOINT.items():
        old = cur.execute(
            "SELECT evidence_path FROM item_history WHERE id=?", (hid,)).fetchone()[0]
        if old == path:
            print(f"  h{hid}: already repointed, skipping")
            continue
        cur.execute(
            "UPDATE item_history SET note=?, evidence_path=? WHERE id=?",
            (f"HXC-217 repoint. Confidence: {why}\n\n"
             "--- original evidence_path ---\n" + (old or ""), path, hid))
        print(f"  h{hid}: evidence_path -> {path}")

    # (4) HXC-187 — no artefact; clear the path and reopen the item
    old187 = cur.execute(
        "SELECT evidence_path FROM item_history WHERE id=306").fetchone()[0]
    if old187 is not None:
        cur.execute("UPDATE item_history SET note=?, evidence_path=NULL WHERE id=306",
                    (HXC187_NOTE + old187,))
        print("  h306 HXC-187: evidence_path -> NULL (no artefact exists)")

    con.commit()
    con.close()
    print("committed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
