#!/usr/bin/env python3
"""HXC-217 part 2 — reopen HXC-187, correct stale path citations, file HXC-224."""
import sqlite3

DB = "docs/workable_items.db"
TODAY = "2026-08-05"
EVID = "docs/qa/hxc217_evidence_path_resolve_20260805T082033Z"

# Stale docs/qa/... citations that were never populated -> the real tracked path.
PATH_FIX = {
    "HXC-124": ("docs/qa/followup_fixes_20260712T085616Z/HXC124_evidence.md",
                "scratch/discovery/fixes/HXC124_evidence.md"),
    "HXC-131": ("docs/qa/followup_fixes_20260712T085616Z/HXC131_evidence.md",
                "scratch/discovery/fixes/HXC131_evidence.md"),
    "HXC-132": ("docs/qa/hxc132_20260712T090048Z/HXC132_evidence.md",
                "scratch/discovery/fixes/HXC132_evidence.md"),
}

REOPEN_DETAILS = (
    "**Reopened-Details:** By: AI; On: 2026-08-05; "
    "Reason: captured-evidence-contradicts; "
    f"Evidence: {EVID} — HXC-217 closure-evidence resolvability audit. This "
    "item's closure record cited commit 09a086a6 (go.mod, 1 file, +1/-1) and "
    "unlogged build/vet results, with no captured artefact. A repo-wide search "
    "returns ZERO tracked paths and ZERO on-disk directories matching 'hxc187'. "
    "The commits are real and the rename may well be correct — but a commit "
    "reference is not captured runtime evidence (§11.4.5 / §11.4.123), so the "
    "closure's warrant cannot be produced on demand. Reopened to capture a real "
    "runtime signature for the module-identity invariant (§11.4.108), not to "
    "redo the rename."
)

DESC_224 = (
    "When we mark a piece of work finished, we record a pointer to the proof "
    "that it really works. Nobody ever checked that those pointers actually "
    "lead anywhere. As a result three different kinds of rot accumulated "
    "silently and none of them were visible to any test or report. First, some "
    "pointers hold a whole paragraph of explanation instead of a location, so "
    "there is no way for a machine to follow them. Second, some point at a "
    "folder that was never created because the person doing the work saved the "
    "proof somewhere else, which means the proof exists but nobody can find it "
    "from the record. Third, some are attached to ordinary progress updates "
    "rather than to the actual completion, which makes routine notes look like "
    "formal proof. The effect is that a completed item can look fully "
    "evidence-backed in every summary and dashboard while the evidence is "
    "unreachable, and the only way anyone finds out is by manually hunting for "
    "it long afterwards. The work here is to make the completion pointer a "
    "checked field: it must point at something that genuinely exists, it must "
    "be attached only to real completions, and anything that fails those rules "
    "must be refused at the moment someone tries to record it rather than "
    "discovered months later. This protects everyone who relies on our "
    "finished-work reports being true, because a claim of proof that cannot be "
    "produced on demand is worse than no claim at all."
)

BODY_224 = f"""## HXC-224 — Closure-evidence pointers are unchecked free text, so proof of completed work silently becomes unreachable

**Status:** Queued
**Type:** Bug
**Severity:** High
**Created-By:** Claude
**Evidence:** {EVID}

{DESC_224}

### Forensic anchor (FACT, HXC-217 audit 2026-08-05)

Of 127 closure records carrying an evidence pointer, **16 (across 14 tickets) did
not resolve**. Systematically resolving every one:

- **4 rows** (HXC-107, HXC-108 x2, HXC-112) were `event_type='Updated'`,
  `reason='operator-blocked'` **progress notes, not closures**. Their real
  closure events resolve cleanly. These were false positives of the audit's own
  population query — a guard that refuses a valid state (§11.4.201(1)).
- **4 rows** (HXC-124, HXC-131 x2, HXC-132) were **path drift**: substantive
  git-tracked evidence exists at `scratch/discovery/fixes/*_evidence.md`; the DB
  recorded a `docs/qa/...` path that was never populated. An earlier pass
  concluded these files "never existed" because it searched the recorded (wrong)
  full path instead of the basename across full history.
- **7 rows** (HXC-158, 161, 165, 167, 182, 184, 203) held **narrative in the
  path field**; every referent proved producible and has been repointed.
- **1 row** (HXC-187) had **no producible artefact at all** — commit references
  only. Reopened separately under §11.4.34.

### Closure criteria

1. `workable-items validate` refuses a closure whose `evidence_path` does not
   resolve, scoped to genuine closure events so it cannot false-positive on
   progress notes.
2. A paired §1.1 mutation proves the guard is not a tautology.
3. The population query distinguishes closure events from historical updates.
"""


def main() -> int:
    con = sqlite3.connect(DB)
    cur = con.cursor()

    # --- stale path citations in body_md + obsolete_details ---
    for atm, (old, new) in PATH_FIX.items():
        body = cur.execute(
            "SELECT body_md FROM items WHERE atm_id=?", (atm,)).fetchone()[0] or ""
        if old in body:
            cur.execute(
                "UPDATE items SET body_md=?, last_modified=datetime('now') "
                "WHERE atm_id=?", (body.replace(old, new), atm))
            print(f"  {atm}: body_md path citation corrected")
    cur.execute(
        "UPDATE obsolete_details SET triple_check_evidence=? WHERE atm_id='HXC-131'",
        (PATH_FIX["HXC-131"][1],))
    print("  HXC-131: obsolete_details.triple_check_evidence corrected")

    # --- reopen HXC-187 ---
    st = cur.execute("SELECT status FROM items WHERE atm_id='HXC-187'").fetchone()[0]
    if st != "Reopened":
        body = cur.execute(
            "SELECT body_md FROM items WHERE atm_id='HXC-187'").fetchone()[0] or ""
        body = body.replace("**Status:** Fixed (→ Fixed.md)", "**Status:** Reopened", 1)
        # place Reopened-Details within 8 non-blank lines of the heading (CONST-058)
        lines = body.split("\n")
        for i, ln in enumerate(lines):
            if ln.startswith("**Status:**"):
                lines.insert(i + 1, REOPEN_DETAILS)
                break
        body = "\n".join(lines)
        cur.execute(
            "UPDATE items SET status='Reopened', current_location='Issues', "
            "body_md=?, closure_date=NULL, last_modified=datetime('now') "
            "WHERE atm_id='HXC-187'", (body,))
        cur.execute(
            "INSERT INTO item_history (atm_id, event_type, by, on_date, reason, "
            "evidence_path, note) VALUES (?,?,?,?,?,?,?)",
            ("HXC-187", "Reopened", "AI", TODAY, "captured-evidence-contradicts",
             EVID,
             "HXC-217 audit: closure cited commits only; zero tracked paths and "
             "zero on-disk dirs match 'hxc187'. Commit-reference-only is not "
             "captured evidence (§11.4.5/§11.4.123)."))
        print("  HXC-187: REOPENED (status + location + history + body)")
    else:
        print("  HXC-187: already Reopened, skipping")

    # --- file HXC-224 ---
    if not cur.execute("SELECT 1 FROM items WHERE atm_id='HXC-224'").fetchone():
        cur.execute(
            "INSERT INTO items (atm_id, type, status, severity, title, description, "
            "created_by, assigned_to, current_location, body_md, representation) "
            "VALUES (?,?,?,?,?,?,?,?,?,?,?)",
            ("HXC-224", "Bug", "Queued", "High",
             "Closure-evidence pointers are unchecked free text, so proof of "
             "completed work silently becomes unreachable",
             DESC_224, "Claude", "", "Issues", BODY_224, "section"))
        cur.execute(
            "INSERT INTO item_history (atm_id, event_type, by, on_date, reason, "
            "evidence_path, note) VALUES (?,?,?,?,?,?,?)",
            ("HXC-224", "Opened", "AI", TODAY, None, EVID,
             "Filed from the HXC-217 closure-evidence resolvability audit."))
        print("  HXC-224: FILED")
    else:
        print("  HXC-224: already exists, skipping")

    con.commit()
    con.close()
    print("committed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
