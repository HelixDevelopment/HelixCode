- 2026-07-11 commit 358ddc13 --no-verify: §11.4.84 residue-check false-positive on scripts/secret_scan_test.sh (mutation-test file legitimately holds the marker string); both files verified secret-clean via scripts/secret_scan.sh exit 0. Follow-up: exempt mutation-test files from residue check.
- 2026-07-11 root push HEAD 9e4de2ba --no-verify: pre-push scan-secrets.sh false-positives on ~40 FABRICATED test fixtures / doc-examples / scanner-literals / untracked r41-report mentions across the tree (mostly pre-existing submodule test files). Stricter content-based secret_scan.sh confirms 0 real secrets; git tree verified secret-clean. ff-only (no force). Follow-up: complete scan-secrets.sh allowlist / consolidate the 2 scanners / scope pre-push to pushed-commits not full working tree.

### CORRECTIONS 2026-08-13 (HXC-331)

Both 2026-07-11 rows are left verbatim above — this is an append-only audit
trail, so the record of what was claimed at bypass time is not rewritten. What
follows corrects what those claims ESTABLISHED. Every count below names the
scanner that produced it, by blob, with its deriving command; a count from one
scanner is never carried over to another.

**Row 2 (9e4de2ba) — "Stricter content-based secret_scan.sh confirms 0 real
secrets; git tree verified secret-clean" is RETRACTED.** No scanner available at
push time established it:

- `secret_scan.sh` AT that revision (blob `acaf04c7`) had NO path-based
  allowlist at all — only `is_self()`:
  `git show 9e4de2ba:scripts/secret_scan.sh | grep -c is_path_allowlisted` -> `0`
  (likewise `ALLOW_PATTERNS` -> `0`; its only functions are `is_self`,
  `scan_file_content`, `scan_disk_file`, `scan_staged_file`). The single
  `scan-secrets-allow` string in that file is a COMMENT on line 44. The
  `.scan-secrets-allow` wiring landed 68 minutes LATER, in `6ab4de36`
  (`2026-07-11T18:27:49+05:00` vs the row's `17:19:19+05:00` push).
- Run today against a file byte-identical to its 9e4de2ba state
  (`git diff 9e4de2ba -- <path> | wc -l` -> `0`), that historical scanner exits
  1 on ONE file: `bash <9e4de2ba scanner> helix_code/internal/autocommit/secret_filter_test.go`
  -> `rc=1`, **4 findings** — a file today's allowlist suppresses (by the
  `.scan-secrets-allow` entry `helix_code/internal/autocommit/secret_filter_test.go`,
  under the "F22 autocommit secret-filter tests" heading — cited by entry, not by
  line number, because an earlier revision of this correction cited `:42` and the
  very edit that added the four `.html` entries above pushed it to `:70`, then a
  follow-up pushed it to `:88`, leaving the citation pointing at a comment;
  today's scanner on the same path -> `rc=0`, with a positive control on another
  path -> `rc=1`, so that 0 is a real suppression and not a dead instrument). A whole-tree run of THAT scanner therefore exits 1
  with far more findings than today's. "Verified secret-clean" was never a
  measured property of that revision.

**Do not conflate that with today's count.** Today's scanner (worktree blob
`5147e88d`, 2026-08-13) over the whole tracked tree at HEAD `48cd8639` reports
**6** findings, all in four rendered `.html` exports — `bash scripts/secret_scan.sh`
-> `rc=1`, 640 s wall-clock (11:19:20Z->11:30:00Z) — plus **1** more in
`scripts/test-verify-all-constitution-rules.sh` measured separately in files mode
(`rc=1`, at the PEM header line) before it was fixed in this same round, for
**7 total**. That 7 is TODAY'S SCANNER APPLIED RETROACTIVELY; it is not a number
any scanner in existence on 2026-07-11 produced, and it is not evidence about
the state of the tree on that date. All 7 were triaged as fabricated or
illustrative fixtures — zero real credentials — under HXC-331.

*Re-verification note (same day, later round).* `secret_scan.sh` is under active
concurrent development and the worktree blob named just above has since been
overwritten — `git hash-object scripts/secret_scan.sh` now yields `89c81b00`
(sha256[0:16] `1d86d10439f98fb4`, 900 lines), so `5147e88d` is no longer
resolvable and its run cannot be re-executed. The COUNTS were therefore
re-derived against `89c81b00`, pinned by copying the script before use: the same
**6** `.html` findings reproduce — identical paths, lines and labels — from a
files-mode run over the four exports with the pre-fix allowlist restored, and the
7th reproduces at `scripts/test-verify-all-constitution-rules.sh:128`
("Private key header (generic)") from the file's pre-fix HEAD content. A fresh
whole-tree run of `89c81b00` over HEAD `48cd8639` WITH both fixes applied exits
**`rc=0`** ("OK: no unallowlisted key-shaped secret pattern found (mode=tree)"),
641 s wall-clock (14:27:12Z->14:37:53Z). Pin every future count to a blob AND
copy the script before running it: naming a worktree blob that a sibling stream
then overwrites leaves the citation unverifiable, which is the same defect class
this section exists to correct.

**Row 1 (358ddc13) is not implicated by the above, but its basis is narrower
than it reads.** Stated as of ITS revision: `secret_scan.sh` at 358ddc13 also had
no path-based allowlist (`is_path_allowlisted` -> `0`, `ALLOW_PATTERNS` -> `0`),
so only `is_self()` applied — and the two files the row names are EXACTLY the two
paths `is_self()` exempts (`scripts/secret_scan.sh`, `scripts/secret_scan_test.sh`).
Their `exit 0` was therefore structural, not a read: at that revision those files
held 1 and 4 pattern-bearing lines respectively
(`git show 358ddc13:<f> | grep -cE 'AKIA…|sk-…|-----BEGIN … PRIVATE KEY-----'`).
The content is benign (the scanner's own pattern literals; its test's planted
fixtures), but a self-exempted path cannot be certified by the scanner that
exempts it, and the row should not be read as if it had been.

**SCOPE BOUNDARY (applies to every "secret-clean" claim in this file).** A clean
exit from either scanner is NOT a clean-tree proof. Measured on this tree:
130 gitlinks are never walked (`git ls-files -s | awk '$1==160000' | wc -l` -> 130;
submodule content is out of scope and must be scanned inside each submodule);
**53** active `.scan-secrets-allow` entries suppress whole files and subtrees
(`grep -vcE '^[[:space:]]*(#|$)' .scan-secrets-allow` -> 53 — an earlier revision
of this paragraph said "~90", which counted no measured quantity; every remaining
line in that file is a comment or a blank, so quote the ACTIVE-entry count, which
is what the scanner loads, and re-derive it rather than copying it: the raw line
total moves every time a comment is added); and binary files are skipped by
`secret_scan.sh` — but NOT by `scan-secrets.sh`, which has no binary probe at
all. The four `.pdf` doc exports are unreached by both scanners, for two
DIFFERENT reasons, and only one of them is a binary skip: `grep -Iq . -- <pdf>`
reports them binary (`rc=1`) so `secret_scan.sh` never reads them, whereas
`scan-secrets.sh` never selects them in the first place — its full-repo walk is
restricted by a `--include=` extension list carrying neither `.pdf` nor `.html`
(measured in that mode: a `.md` plant is flagged, `.html` and `.pdf` plants are
not; that filter is applied ONLY when the scan target is `.`, so probing it with
an explicit directory argument exercises the unfiltered else-branch and
misleadingly appears to scan both). Independently, the strings are not present in
the PDFs as raw bytes (`grep -acE -e '<key shapes>' <pdf>` -> 0) — they sit
inside compressed streams. Unreached is not clean: `pdftotext` shows they DO
carry recoverable key-shaped strings (five of the six
`.html` findings resurface after extraction; the sixth does not, because
extraction mangles that file's dash runs — a dash-tolerant variant matches it,
and a 31,919-byte positive control confirms extraction itself worked). Any line
containing `redacted` / `example` / `...` is skipped by content regardless of
what else is on it. "Scanner exited 0" means "no unallowlisted key-shaped
pattern in the scanned subset", never "this repository contains no secrets".

## Sources verified 2026-07-27: INTERNALLY DERIVED — this file is the §11.4.75 `--no-verify` bypass audit trail; it has NO external source and cites none. Its entries are derived solely from this repository's own state, and each referenced artefact was confirmed present on this host in this pass: the two cited commits resolve in local git history (`git log --oneline -1 358ddc13` → "feat(security): add HuggingFace hf_ token pattern to secret-scan guard (§11.4.138)"; `git log --oneline -1 9e4de2ba` → "feat(ops): §11.4.111 coder-boot script — regenerate CDI spec + rootless start"), and the two scanners plus the hook set the entries describe exist at `scripts/secret_scan.sh`, `scripts/scan-secrets.sh`, and `scripts/git_hooks/{pre-commit,pre-push,post-commit,commit-msg}`. Source of truth for every future row: the local git history of the bypassing commit/push plus the scanner output captured at bypass time — never an external URL. Honest boundary (§11.4.6 / §11.4.99(B)): no external documentation applies to or was consulted for this document, and the follow-up items recorded above remain open.
