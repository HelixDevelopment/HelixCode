#!/usr/bin/env python3
"""HXC-217 evidence_path resolve check.

Anchor definition (re-verified 2026-08-05, this session):
  Population = rows of `item_history` with a non-empty `evidence_path`
  whose `atm_id` belongs to an item whose CURRENT status is one of the four
  closed statuses. That population is the set of closure claims that assert
  captured evidence.

A row FAILS when its evidence_path does not resolve to an existing path on
disk (the `[ -e "$p" ]` test), interpreted relative to the repository root.

Failures are then split into two DIFFERENT defect classes:
  - "prose"            : the field holds a narrative paragraph, not a path
                         (contains a newline, or contains whitespace AND is
                         not a plausible single path token).
  - "wellformed-missing": the field holds a clean, well-formed path token
                         that simply does not exist on disk.

Usage:  python3 resolve_check.py [/path/to/repo-root]
Exit:   0 always (this is a reporting tool; the gate lives in the Go
        validator). Report goes to stdout as JSON + a human table.
"""
import json
import os
import sqlite3
import sys

CLOSED = (
    "Fixed (→ Fixed.md)",
    "Implemented (→ Fixed.md)",
    "Completed (→ Fixed.md)",
    "Obsolete (→ Fixed.md)",
)

REPO = os.path.abspath(sys.argv[1] if len(sys.argv) > 1 else ".")
DB = os.path.join(REPO, "docs", "workable_items.db")


def classify(p: str) -> str:
    """Split a non-resolving value into its defect class."""
    if "\n" in p or "\r" in p:
        return "prose"
    if any(ch.isspace() for ch in p):
        return "prose"
    return "wellformed-missing"


def main() -> int:
    con = sqlite3.connect(f"file:{DB}?mode=ro", uri=True)
    cur = con.cursor()
    rows = cur.execute(
        """
        SELECT h.id, h.atm_id, h.event_type, h.by, h.on_date, h.evidence_path,
               (SELECT i.status FROM items i WHERE i.atm_id = h.atm_id LIMIT 1)
        FROM item_history h
        WHERE h.evidence_path IS NOT NULL AND TRIM(h.evidence_path) <> ''
          AND EXISTS (SELECT 1 FROM items i
                      WHERE i.atm_id = h.atm_id AND i.status IN (?, ?, ?, ?))
        ORDER BY h.atm_id, h.id
        """,
        CLOSED,
    ).fetchall()
    con.close()

    failures = []
    for hid, atm, ev, by, on, path, status in rows:
        abspath = path if os.path.isabs(path) else os.path.join(REPO, path)
        if os.path.exists(abspath):
            continue
        failures.append(
            {
                "history_id": hid,
                "atm_id": atm,
                "event_type": ev,
                "by": by,
                "on_date": on,
                "status": status,
                "klass": classify(path),
                "evidence_path": path,
            }
        )

    prose = [f for f in failures if f["klass"] == "prose"]
    missing = [f for f in failures if f["klass"] == "wellformed-missing"]

    print(f"population (closed items, non-empty evidence_path) : {len(rows)}")
    print(f"resolve FAILURES                                   : {len(failures)}")
    print(f"  class prose (narrative in the path field)        : {len(prose)}")
    print(f"  class wellformed-missing (clean path, no file)   : {len(missing)}")
    print(f"distinct tickets failing                           : "
          f"{len({f['atm_id'] for f in failures})}")
    print()
    print("--- wellformed-missing ---")
    for f in missing:
        print(f"  h{f['history_id']:>4} {f['atm_id']} [{f['status']}] "
              f"{f['event_type']} {f['on_date']}  ->  {f['evidence_path']}")
    print()
    print("--- prose ---")
    for f in prose:
        first = f["evidence_path"].splitlines()[0][:88]
        print(f"  h{f['history_id']:>4} {f['atm_id']} [{f['status']}] "
              f"{f['event_type']} {f['on_date']}  ->  {first}...")

    out = os.path.join(os.path.dirname(os.path.abspath(__file__)),
                       "resolve_failures.json")
    with open(out, "w", encoding="utf-8") as fh:
        json.dump(
            {
                "population": len(rows),
                "failures": len(failures),
                "prose": len(prose),
                "wellformed_missing": len(missing),
                "rows": failures,
            },
            fh,
            indent=2,
            ensure_ascii=False,
        )
    print(f"\nwrote {out}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
