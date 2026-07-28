# HXC-160 captured verification — 2026-07-28T21:07:02Z

## 1. Broken remediation command BEFORE fix (bash -n)
bash: line 2: syntax error near unexpected token `export'
bash: line 2: `    export --db docs/workable_items.db --out-dir docs'
(exit 2 — nonzero = syntax error, as expected)

## 2. Replacement command AFTER fix (bash -n)
(exit 0 — 0 = valid syntax)

## 3. Replacement command ACTUALLY RUNS (out-dir -> temp, docs untouched)
```
export: wrote /tmp/hxc160ev.3HF1/Issues.md
export: wrote /tmp/hxc160ev.3HF1/Fixed.md
export: wrote /tmp/hxc160ev.3HF1/Issues_Summary.md
export: wrote /tmp/hxc160ev.3HF1/Fixed_Summary.md
export: --no-formats set; skipped HTML/PDF/DOCX sibling generation
exit=0
```

## 4. Gate baseline + paired §1.1 mutations
```
CM-SUPERSEDED-GENERATOR-NAMING: 21 correctly-marked reference(s) across 12 governed carrier(s), 0 violation(s)
baseline exit=0
FAIL CM-SUPERSEDED-GENERATOR-NAMING  GEMINI.md:298 names a superseded shell summary generator with no supersession marker
     line: The `(→ Fixed.md)` suffix is preserved across all three so the existing migration-discipline tooling (atomic Issues.md...
CM-SUPERSEDED-GENERATOR-NAMING: 20 correctly-marked reference(s) across 12 governed carrier(s), 1 violation(s)
MUTATION-1 exit=1 (1 = correctly FAILs)
CM-SUPERSEDED-GENERATOR-NAMING: 21 correctly-marked reference(s) across 12 governed carrier(s), 0 violation(s)
restored exit=0
```
RESIDUE CHECK: GEMINI.md identical to HEAD
