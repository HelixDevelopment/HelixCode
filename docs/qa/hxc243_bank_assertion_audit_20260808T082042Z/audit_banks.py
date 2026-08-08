#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""HXC-243 blast-radius audit.

Counts, across every HelixQA bank file, the `http:` steps that declare
NO assertion at all (no expect_status / expect_body_contains /
expect_json_path). Such a step scores PASS on ANY HTTP response --
404, 503, anything -- per pkg/autonomous/http_executor.go:319-339,
which only compares when the corresponding field is non-zero/non-empty.

A bank whose http steps are ALL assertion-free is UNFALSIFIABLE: no
service state can make it FAIL.

Usage: python3 audit_banks.py <banks-dir>
"""
import json
import os
import sys

import yaml

ASSERT_KEYS = ("expect_status", "expect_body_contains", "expect_json_path")


def step_is_http(step):
    action = step.get("action") or ""
    return isinstance(action, str) and action.strip().lower().startswith("http:")


def main():
    banks_dir = sys.argv[1]
    rows = []
    for name in sorted(os.listdir(banks_dir)):
        if not name.endswith((".yaml", ".yml")):
            continue
        path = os.path.join(banks_dir, name)
        try:
            with open(path, "r", encoding="utf-8") as fh:
                doc = yaml.safe_load(fh)
        except Exception as exc:  # parse failure is a finding, not a skip
            rows.append({"bank": name, "parse_error": str(exc)})
            continue
        if not isinstance(doc, dict):
            rows.append({"bank": name, "parse_error": "top-level not a mapping"})
            continue
        cases = doc.get("test_cases") or []
        http_steps = 0
        bare_steps = 0
        skipped_steps = 0
        bare_case_ids = []
        dispatch_cases = 0
        for case in cases:
            if not isinstance(case, dict):
                continue
            if case.get("dispatches_to"):
                dispatch_cases += 1
            case_bare = 0
            for step in case.get("steps") or []:
                if not isinstance(step, dict):
                    continue
                if not step_is_http(step):
                    continue
                http_steps += 1
                if any(step.get(k) for k in ASSERT_KEYS):
                    continue
                # An explicit `_skip: true` WITH a reason is not a bluff --
                # it is the honest §11.4.3 disposition for a step no
                # expectation can truthfully describe (multipart body, a
                # per-case port the single --base-url cannot express). The
                # runner reports SKIP, never PASS, so such a step cannot
                # manufacture a false green.
                if step.get("_skip") and str(step.get("_skip_reason", "")).strip():
                    skipped_steps += 1
                    continue
                bare_steps += 1
                case_bare += 1
            if case_bare:
                bare_case_ids.append(case.get("id"))
        rows.append({
            "bank": name,
            "cases": len(cases),
            "http_steps": http_steps,
            "bare_http_steps": bare_steps,
            "honest_skip_steps": skipped_steps,
            "dispatch_cases": dispatch_cases,
            "unfalsifiable": http_steps > 0 and bare_steps == http_steps,
            "bare_case_ids": bare_case_ids,
        })

    with_http = [r for r in rows if r.get("http_steps")]
    unfals = [r for r in with_http if r["unfalsifiable"]]
    partial = [r for r in with_http if r["bare_http_steps"] and not r["unfalsifiable"]]

    print("=== HXC-243 BANK ASSERTION AUDIT ===")
    print(f"banks scanned              : {len(rows)}")
    print(f"banks with http: steps     : {len(with_http)}")
    print(f"FULLY UNFALSIFIABLE banks  : {len(unfals)}  "
          "(every http step assertion-free -> cannot FAIL)")
    print(f"partially bare banks       : {len(partial)}")
    print(f"total http steps           : {sum(r['http_steps'] for r in with_http)}")
    print(f"honest _skip w/ reason     : {sum(r['honest_skip_steps'] for r in with_http)}  (§11.4.3 — runner reports SKIP, never PASS)")
    print(f"STILL assertion-free       : {sum(r['bare_http_steps'] for r in with_http)}  <-- the defect count; MUST be 0")
    print()
    print("--- FULLY UNFALSIFIABLE ---")
    for r in sorted(unfals, key=lambda r: -r["http_steps"]):
        print(f"  {r['bank']:<50} cases={r['cases']:<4} http_steps={r['http_steps']:<4} "
              f"dispatch_cases={r['dispatch_cases']}")
    print()
    print("--- PARTIALLY BARE ---")
    for r in sorted(partial, key=lambda r: -r["bare_http_steps"]):
        print(f"  {r['bank']:<50} bare={r['bare_http_steps']}/{r['http_steps']}")
    errs = [r for r in rows if r.get("parse_error")]
    if errs:
        print()
        print("--- PARSE ERRORS ---")
        for r in errs:
            print(f"  {r['bank']}: {r['parse_error']}")

    out = os.path.join(os.path.dirname(os.path.abspath(__file__)), "audit_rows.json")
    with open(out, "w", encoding="utf-8") as fh:
        json.dump(rows, fh, indent=2)
    print(f"\nrows written to {out}")


if __name__ == "__main__":
    main()
