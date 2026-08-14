#!/usr/bin/env bash
# scripts/git_hooks/pre-commit
# §11.4.75 mechanical-enforcement layer 1 (local pre-commit gate).
#
# Three staged-only invariants, evaluated against the index (what `git
# commit` is about to write), NEVER against the whole tree:
#
#   1. §11.4.75 governance-doc sibling check — a staged GOVERNED `.md`
#      MUST ship its `.html` + `.pdf` siblings in the SAME index. Scoped
#      to the GOVERNED doc set ONLY (root manuals + tracker docs +
#      Status docs); working specs under docs/{superpowers,research,
#      guides,testing} are md-only by convention and are NOT checked.
#   2. §11.4.30 / §11.4.10 forbidden-class check — no staged secret /
#      build-artifact / private-key class file.
#   3. §11.4.84 mutation-residue check — no staged file carrying a
#      paired-mutation marker (`MUTATED for paired`, `// always pass`, …).
#
# This is a COURTESY GATE; the constitutional clauses are the actual
# contract. On any genuine product/security risk it BLOCKS (exit 1);
# benign tooling gaps (no staged files, detached index) degrade with a
# WARN and exit 0 rather than blocking every commit.
#
# Bypass: `git commit --no-verify` skips this hook, but the commit-msg
# hook then demands a `Bypass-rationale:` footer (§11.4.75 audit trail).
#
# Inputs:   the git index (staged files).
# Outputs:  exit 0 = OK / exit 1 = BLOCKED. Diagnostics on stderr.
# Side-effects: none (read-only over the index).
# Dependencies: git, grep, POSIX sh utilities.
# Cross-references: §11.4.75 / §11.4.30 / §11.4.10 / §11.4.84 / §11.4.65;
#   scripts/git_hooks/{pre-push,post-commit,commit-msg};
#   scripts/install_git_hooks.sh.

set -uo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
cd "$REPO_ROOT" || exit 0

# §11.4.75 — drop a freshness marker so the commit-msg hook can detect a
# --no-verify bypass (which skips THIS hook, leaving the marker stale).
GITDIR=$(git rev-parse --git-dir 2>/dev/null || echo "$REPO_ROOT/.git")
: > "$GITDIR/ATMO_PRECOMMIT_RAN" 2>/dev/null || true

block=0

# ---------------------------------------------------------------------------
# Collect the staged (Added/Copied/Modified/Renamed) path list once.
# ---------------------------------------------------------------------------
staged=$(git diff --cached --name-only --diff-filter=ACMR 2>/dev/null || true)

if [ -z "$staged" ]; then
  # Nothing staged (e.g. `git commit` on an empty index, or amend with no
  # content change). Nothing for this hook to enforce.
  exit 0
fi

# ---------------------------------------------------------------------------
# 1. §11.4.65 / CONST-066 — markdown-sibling check (reconciled scope).
#
#    CONST-066 / §11.4.65 INCLUDE the whole `docs/**/*.md` tree (guides,
#    research, plans, specs, Status docs, trackers, root manuals). Full
#    enforcement of that scope is BLOCKED today only because ~270 pre-
#    existing docs still lack siblings (a tracked backfill task generates
#    them via scripts/testing/sync_all_markdown_exports.sh --regenerate-all).
#
#    To advance toward CONST-066 without blocking every doc commit while
#    the backfill is in flight, a staged `.md` is sibling-CHECKED iff:
#      (a) it is in the CORE GOVERNED set — root manual / tracker doc /
#          Status{,_Summary}.md — ALWAYS checked (unchanged behaviour), OR
#      (b) it ALREADY has at least one sibling (.html or .pdf) in the tree
#          or the index — i.e. it has opted into the export regime, so an
#          update that forgets to regenerate its siblings is caught.
#
#    A brand-new md-only working spec (no sibling yet) is NOT blocked — it
#    is part of the backfill scope, not a regression. This is the §11.4.120
#    reconciliation: the gate widens to the export-opted-in superset while
#    the §11.4.6 "don't guess wider than the mandate" guard is preserved
#    for not-yet-backfilled new docs.
#
#    TODO(CONST-066 full activation): once the 270-doc backfill lands and
#    `sync_all_markdown_exports.sh --check-only` exits 0 over the whole
#    tree, replace clause (b) with an unconditional `docs/**/*.md` check
#    (is_export_optin_md -> always-return-0 for docs/*.md), and update
#    test_hooks.sh cases 7-9 to expect BLOCK for sibling-less working specs.
# ---------------------------------------------------------------------------
is_core_governed_md() {
  # $1 = repo-relative path ending in .md — ALWAYS sibling-checked.
  case "$1" in
    CLAUDE.md|AGENTS.md|CONSTITUTION.md|CRUSH.md|QWEN.md) return 0 ;;
    docs/Issues.md|docs/Issues_Summary.md|docs/Fixed.md|docs/Fixed_Summary.md|docs/CONTINUATION.md) return 0 ;;
    docs/*/Status.md|docs/*/Status_Summary.md|docs/*/*/Status.md|docs/*/*/Status_Summary.md) return 0 ;;
  esac
  return 1
}

# A docs/**/*.md has "opted into" the export regime if it already ships
# at least one sibling (.html or .pdf) — staged or tracked. Updating such
# a doc without regenerating its siblings is a §11.4.65 sync regression.
has_existing_sibling() {
  # $1 = base path (no extension). True if .html or .pdf is staged or
  #      currently tracked (and not staged-for-deletion).
  local base="$1" ext
  for ext in html pdf; do
    if sibling_present "${base}.${ext}"; then return 0; fi
  done
  return 1
}

is_governed_md() {
  # $1 = repo-relative path ending in .md. Checked iff core-governed OR
  #      (under docs/ AND already export-opted-in).
  is_core_governed_md "$1" && return 0
  case "$1" in
    docs/*.md)
      if has_existing_sibling "${1%.md}"; then return 0; fi
      ;;
  esac
  return 1
}

# A staged path is "present in this commit" if it is staged OR already
# tracked (sibling that exists in the tree and is not being deleted).
sibling_present() {
  # $1 = sibling path. Present if staged in this commit OR currently tracked
  #      AND not staged-for-deletion.
  local p="$1"
  if printf '%s\n' "$staged" | grep -qxF "$p"; then
    return 0
  fi
  if git ls-files --error-unmatch "$p" >/dev/null 2>&1; then
    # tracked — ensure it is not being deleted in this same commit
    if git diff --cached --name-only --diff-filter=D 2>/dev/null | grep -qxF "$p"; then
      return 1
    fi
    return 0
  fi
  return 1
}

missing_siblings=""
while IFS= read -r f; do
  case "$f" in
    *.md) : ;;
    *) continue ;;
  esac
  is_governed_md "$f" || continue
  base="${f%.md}"
  for ext in html pdf; do
    if ! sibling_present "${base}.${ext}"; then
      missing_siblings="${missing_siblings}\n  ${f} -> missing ${base}.${ext}"
    fi
  done
done <<EOF
$staged
EOF

if [ -n "$missing_siblings" ]; then
  {
    echo ""
    echo "============================================================"
    echo "BLOCKED by pre-commit hook (§11.4.65 / CONST-066 md siblings)"
    echo "============================================================"
    echo "A staged sibling-CHECKED .md is missing its .html and/or .pdf sibling."
    echo "Checked set: core-governed docs (root manuals CLAUDE/AGENTS/"
    echo "CONSTITUTION/CRUSH/QWEN, docs/{Issues,Issues_Summary,Fixed,"
    echo "Fixed_Summary,CONTINUATION}.md, every docs/**/Status{,_Summary}.md)"
    echo "PLUS any docs/**/*.md that already ships a sibling (export-opted-in)."
    printf '%b\n' "$missing_siblings"
    echo ""
    echo "Regenerate siblings, then re-stage:"
    echo "  bash scripts/testing/sync_all_markdown_exports.sh --file <path.md>"
    echo "  # (renders both .html + .pdf via pandoc + weasyprint)"
    echo "============================================================"
  } >&2
  block=1
fi

# ---------------------------------------------------------------------------
# 2. §11.4.30 / §11.4.10 — forbidden-class staged files.
#
#    Secrets, private keys, real .env, build artifacts. `.env.example`,
#    `.env.sample`, `.env.full-test`, and `*.pem` under fixtures/examples
#    are NOT secrets (placeholders), so they are explicitly allowed.
# ---------------------------------------------------------------------------
forbidden=""
while IFS= read -r f; do
  base=$(basename "$f")
  case "$f" in
    # Allowed placeholders / fixtures first (skip).
    *.env.example|*.env.sample|*.env.full-test|*/.env.example|*/.env.sample) continue ;;
    *.example|*.sample) continue ;;
  esac
  case "$base" in
    .env|.env.*|*.env) forbidden="${forbidden}\n  ${f}  (real .env — §11.4.10)" ;;
    id_rsa|id_rsa.*|id_ed25519|id_ed25519.*|.netrc) forbidden="${forbidden}\n  ${f}  (private key / netrc — §11.4.10)" ;;
    *.pem|*.key|*.crt) forbidden="${forbidden}\n  ${f}  (credential material — §11.4.10)" ;;
    api_keys.sh) forbidden="${forbidden}\n  ${f}  (api keys — §11.4.10)" ;;
  esac
  case "$f" in
    */bin/*|bin/*|*/build/*|build/*|*/dist/*|dist/*|*/out/*|out/*) ;;  # path-class artifacts: warn only below
  esac
  case "$base" in
    *.so|*.dylib|*.exe|*.dll|*.class|*.pyc|*.o|*.a) forbidden="${forbidden}\n  ${f}  (build artifact — §11.4.30)" ;;
  esac
done <<EOF
$staged
EOF

if [ -n "$forbidden" ]; then
  {
    echo ""
    echo "============================================================"
    echo "BLOCKED by pre-commit hook (§11.4.30/§11.4.10 forbidden class)"
    echo "============================================================"
    echo "A staged file matches a secret / build-artifact forbidden class:"
    printf '%b\n' "$forbidden"
    echo ""
    echo "Un-stage it, add it to .gitignore, and (if a secret) rotate per"
    echo "§11.4.10 / CONST-042 if it already reached a remote."
    echo "============================================================"
  } >&2
  block=1
fi

# ---------------------------------------------------------------------------
# 3. §11.4.84 — mutation-residue scan over STAGED CONTENT.
#
#    A paired §1.1 mutation must be serialised (mutate -> assert FAIL ->
#    restore) so the working tree is clean before any unrelated commit. A
#    staged mutation marker is the residue that leaked an `// always pass`
#    JWT bypass into a logo commit (the forensic case behind §11.4.84).
#
#    This hook excludes ITSELF and its sibling hooks + their test (the
#    markers appear here as documentation/test fixtures, not residue).
#
#    False-positive fix (2026-07-11): a mutation-TEST script (e.g.
#    scripts/secret_scan_test.sh) legitimately embeds a residue-marker
#    string in its OWN source — as a comment documenting its paired §1.1
#    mutation logic, or as a literal it mutates-in-then-restores under a
#    `trap ... EXIT` — and is not itself residue. Such a file is EXEMPT
#    from the BLOCK iff it satisfies BOTH:
#      (a) EXPLICIT opt-in — carries the literal, greppable, auditable
#          marker `11.4.84-mutation-test-exempt` (conventionally as a
#          `# §11.4.84-mutation-test-exempt: ...` header comment). This is
#          never inferred — a file that does not carry this exact marker
#          gets ZERO benefit of the doubt, so real (non-test) files are
#          completely unaffected by this exemption.
#      (b) PROVEN restore idiom, SEMANTICALLY WIRED (tightened 2026-07-11
#          follow-up review): a `trap <target> EXIT` line where <target>
#          is EITHER an inline command containing the restore call itself
#          OR a named function whose OWN body (not merely "somewhere else
#          in the file") contains a `cp ... backup ...` / `git checkout --`
#          restore call. Requiring the restore command to live INSIDE the
#          trap's actual target — rather than the two earlier independent
#          greps ("a trap...EXIT line exists somewhere" AND "a cp...backup
#          line exists somewhere", regardless of whether they're related —
#          closes the abuse case where an UNRELATED cp of an UNRELATED
#          "backup"-named file elsewhere in the same staged file could
#          satisfy both greps without the trap actually restoring anything.
#    A file satisfying (a) but not (b) (or vice-versa) is NOT exempt and
#    is BLOCKED exactly as before this fix.
#
#    Every GRANTED exemption emits an auditable NOTICE to stderr (2026-07-11
#    follow-up) — never writes to the tree mid-commit (that would itself be
#    a §11.4.84 residue risk) — so a human reviewing hook output always sees
#    exactly which staged file(s) were exempted and why, rather than the
#    exemption being silent.
#
#    SECOND EXEMPTION PATH — CAPTURED EVIDENCE (HXC-223, 2026-08-06). The
#    (a)+(b) exemption above was designed for mutation-TEST SCRIPTS, and a
#    captured-evidence artifact is STRUCTURALLY UNABLE to qualify for it: an
#    evidence `.log` can carry the (a) opt-in marker but can NEVER satisfy
#    (b), because it is captured OUTPUT — no trap, no restore, nothing to
#    execute. That left §1.1 (which REQUIRES the paired-mutation proof be
#    CAPTURED) and §11.4.84 (which refused to let that capture be committed)
#    in direct contradiction as implemented: the proof of a correctly
#    serialised mutation was unlandable precisely BECAUSE it proved the
#    mutation. Forensic case: docs/qa/hxc220_module_identity_gate_
#    20260805T120714Z/3_mutation_proof.log.
#
#    The discriminator is NOT location alone. `docs/qa/**` genuinely DOES
#    contain tracked EXECUTABLES (harness `run_proof.sh`, mode 100755), so
#    "it's under docs/qa" would be an unsound test. What matters for
#    §11.4.84 is whether the residue is code that RUNS: the forensic case
#    behind the clause is an always-pass JWT-verify bypass sweeping into a
#    logo commit and becoming live. A marker inside a non-executable
#    captured-output artifact cannot become live — nothing ever executes it.
#    So a staged file is exempt under this SECOND path iff ALL FOUR hold:
#      (i)   it lives under the evidence tree `docs/qa/**` (§11.4.83);
#      (ii)  its extension is in the closed captured-output/narrative set
#            {log, txt, out, md} — NOT .sh/.go/.py/etc, and NOT .patch or
#            .diff (a patch is appliable, so it can still carry a mutation
#            into live code via `git apply`);
#      (iii) its STAGED mode is 100644 — a staged executable is code the
#            moment it lands, whatever it happens to be named; and
#      (iv)  its first staged line is not a `#!` shebang — a script-shaped
#            blob is treated as code regardless of extension.
#    Conditions (ii)+(iii)+(iv) are deliberately redundant with each other:
#    extension alone, mode alone, and shape alone are each individually
#    defeatable, so all three are required together.
#
#    NOTE — this second path deliberately does NOT require the (a) opt-in
#    marker. Captured evidence must stay VERBATIM (§11.4.5): editing an
#    opt-in header into a captured log would corrupt the very artifact whose
#    fidelity is the point. The four structural conditions carry the
#    exemption instead. Nothing outside `docs/qa/**` gets any benefit of the
#    doubt, and live source under scripts/ / helix_code/ / any submodule is
#    BLOCKED exactly as before.
#
#    Like the (a)+(b) path, every GRANTED evidence exemption emits an
#    auditable NOTICE to stderr naming the file and the reason — exemptions
#    are never silent.
#
#    Honest limitation (§11.4.6): this remains a TEXTUAL / heuristic check,
#    not a full semantic proof that the restore idiom actually executes
#    correctly at runtime (that would require actually invoking the staged
#    script in a sandbox and observing it restore a real mutated file,
#    which is not cleanly doable from a pre-commit hook operating on a
#    staged blob that may not even be executable yet, may depend on
#    repo-relative paths that don't exist in isolation, etc.). The
#    function-body scoping above is a genuine tightening — it requires the
#    restore call to be REACHABLE from the trap's own target, not merely
#    present anywhere in the file — but a sufficiently deliberate staged
#    file could still satisfy the shape without the idiom being correct.
#    This residual gap is accepted and documented rather than silently
#    claimed closed.
#
#    Residual gap of the captured-evidence path (§11.4.6, HXC-223): the four
#    conditions establish that the artifact is not executable BY ITS OWN
#    STAGED FORM. They do not prove no external caller could ever feed it to
#    an interpreter (`bash docs/qa/<run>/x.log`) or that a human will not
#    later `chmod +x` it. Both would be deliberate acts against a file whose
#    directory, extension and mode all declare it evidence, and a later
#    chmod is itself a staged mode change this scan re-evaluates on the next
#    commit. The gap is narrow and is documented, not claimed closed.
# ---------------------------------------------------------------------------
is_mutation_test_exempt() {
  # $1 = staged blob content of a file that already matched a residue
  # marker. Returns 0 (exempt) only if BOTH the explicit opt-in header AND
  # the semantically-scoped restore idiom (see above) are present.
  local blob="$1"
  printf '%s' "$blob" | grep -qF '11.4.84-mutation-test-exempt' || return 1

  local trap_line trap_target
  trap_line=$(printf '%s\n' "$blob" | grep -E 'trap[[:space:]].*EXIT' | head -1)
  [ -z "$trap_line" ] && return 1

  # Named-function form: `trap cleanup EXIT` / `trap "cleanup" EXIT`.
  trap_target=$(printf '%s' "$trap_line" | sed -nE "s/.*trap[[:space:]]+['\"]?([A-Za-z_][A-Za-z0-9_]*)['\"]?[[:space:]]+EXIT.*/\1/p")

  if [ -n "$trap_target" ]; then
    # Extract the trap TARGET function's own body — handles both a
    # multi-line function (`fn() {` ... `}` on its own line) and a
    # one-liner (`fn() { ...; }` all on one line) — and require the
    # restore call to live INSIDE that specific body, not merely anywhere
    # else in the staged file.
    local func_body
    func_body=$(printf '%s\n' "$blob" | awk -v fn="$trap_target" '
      BEGIN { infn = 0 }
      {
        if (!infn && $0 ~ ("^" fn "\\(\\)[[:space:]]*\\{")) {
          line = $0
          sub("^" fn "\\(\\)[[:space:]]*\\{", "", line)
          if (line ~ /\}[[:space:]]*$/) {
            sub(/\}[[:space:]]*$/, "", line)
            print line
            infn = 0
          } else {
            print line
            infn = 1
          }
          next
        }
        if (infn) {
          if ($0 ~ /^[[:space:]]*\}[[:space:]]*$/) { infn = 0; next }
          print
        }
      }
    ')
    printf '%s' "$func_body" | grep -qiE 'cp[[:space:]].*backup|git checkout --' || return 1
  else
    # Inline trap command form: `trap 'cp ... BACKUP ...' EXIT` — the
    # restore call must appear IN THE TRAP LINE ITSELF.
    printf '%s' "$trap_line" | grep -qiE 'cp[[:space:]].*backup|git checkout --' || return 1
  fi

  return 0
}

is_captured_evidence_exempt() {
  # $1 = staged PATH of a file that already matched a residue marker.
  # Returns 0 (exempt) only if ALL FOUR structural conditions hold — see the
  # "SECOND EXEMPTION PATH" block above. Deliberately path/mode/shape based:
  # captured evidence must remain verbatim, so no in-file opt-in is demanded.
  local f="$1" base ext mode firstline

  # (i) inside the §11.4.83 evidence tree.
  case "$f" in
    docs/qa/*) ;;
    *) return 1 ;;
  esac

  # (ii) closed captured-output / narrative extension set.
  base=${f##*/}
  case "$base" in
    *.*) ext=$(printf '%s' "${base##*.}" | tr 'A-Z' 'a-z') ;;
    *) return 1 ;;                       # extensionless: not a known-inert class
  esac
  case "$ext" in
    log|txt|out|md) ;;
    *) return 1 ;;
  esac

  # (iii) STAGED mode must be non-executable (100644). Read from the index,
  # not the worktree, so it reflects exactly what is about to be committed.
  mode=$(git ls-files --stage -- "$f" 2>/dev/null | awk 'NR==1{print $1}')
  [ "$mode" = "100644" ] || return 1

  # (iv) not script-shaped: a `#!` first line is code whatever its name.
  firstline=$(git show ":$f" 2>/dev/null | head -1)
  case "$firstline" in
    '#!'*) return 1 ;;
  esac

  return 0
}

mutation_hits=""
while IFS= read -r f; do
  case "$f" in
    scripts/git_hooks/*) continue ;;   # hook bodies + test legitimately name the markers
  esac
  [ -f "$f" ] || continue
  # Scan the STAGED blob (index), not the worktree, so we catch exactly
  # what is about to be committed.
  blob=$(git show ":$f" 2>/dev/null) || continue
  if printf '%s' "$blob" | grep -qE 'MUTATED for paired|// always pass|# always pass|MUTATION-RESIDUE|_mutated_'; then
    if is_mutation_test_exempt "$blob"; then
      # §11.4.84 audit trail: log the grant to stderr (never to the tree —
      # writing a file mid-commit would itself risk leaving residue).
      echo "NOTICE (§11.4.84 audit): mutation-test exemption granted for staged file: $f" >&2
      continue   # explicitly-marked + trap-restored mutation-test file: not residue
    fi
    if is_captured_evidence_exempt "$f"; then
      # §11.4.84 audit trail: same stderr-only discipline as the (a)+(b)
      # grant above — never written to the tree mid-commit.
      echo "NOTICE (§11.4.84 audit): captured-evidence exemption granted for staged file: $f" >&2
      echo "NOTICE (§11.4.84 audit):   reason: non-executable captured-output artifact under docs/qa/** (staged mode 100644, inert extension, no shebang) — it never executes, so the marker cannot leak a bypass into a running artifact; §1.1 requires this proof be captured." >&2
      continue   # captured evidence, structurally non-executable: not residue
    fi
    mutation_hits="${mutation_hits}\n  ${f}"
  fi
done <<EOF
$staged
EOF

if [ -n "$mutation_hits" ]; then
  {
    echo ""
    echo "============================================================"
    echo "BLOCKED by pre-commit hook (§11.4.84 mutation residue)"
    echo "============================================================"
    echo "A staged file carries a paired-mutation marker. A mutation gate"
    echo "MUST be restored (mutate -> assert FAIL -> restore) before any"
    echo "unrelated commit. Staged file(s):"
    printf '%b\n' "$mutation_hits"
    echo ""
    echo "Restore the mutated file(s) and re-stage, then commit."
    echo "============================================================"
  } >&2
  block=1
fi

# ---------------------------------------------------------------------------
# 4. §11.4.135 / §11.4.138 — key-shaped-secret content scan over STAGED
#    CONTENT (scripts/secret_scan.sh --staged), closing the committed-key
#    leak class (forensic anchor: commit 41372967 / docs/qa/
#    SECURITY_INCIDENT_gemini_key_leak_20260711.md — a real Google API key
#    was committed in a docs/qa/*.md evidence file; scripts/scan-secrets.sh
#    already existed but was wired ONLY to the pre-push hook, so nothing
#    blocked the COMMIT itself). Scans the staged git blobs (what is about
#    to be committed), not the working tree. Never prints the matched
#    secret value — only "<file>:<line>".
# ---------------------------------------------------------------------------
SECRET_SCANNER="$REPO_ROOT/scripts/secret_scan.sh"
if [ -x "$SECRET_SCANNER" ]; then
  secret_scan_output=$("$SECRET_SCANNER" --staged 2>&1)
  secret_scan_rc=$?
  if [ "$secret_scan_rc" -ne 0 ]; then
    {
      echo ""
      echo "============================================================"
      echo "BLOCKED by pre-commit hook (§11.4.135/§11.4.138 secret scan)"
      echo "============================================================"
      printf '%s\n' "$secret_scan_output"
      echo "============================================================"
    } >&2
    block=1
  fi
fi

if [ "$block" -eq 1 ]; then
  exit 1
fi
exit 0
