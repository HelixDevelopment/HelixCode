#!/usr/bin/env bash
# scripts/secret_scan.sh
#
# Permanent secret-scan guard closing the committed-key leak class per
# §11.4.135 (standing regression-guard suite) / §11.4.138 (operator-escape
# bluff-audit + permanent guard). Forensic anchor: commit 41372967 redacted
# a real Google (Gemini) API key that had been committed in plaintext in
# docs/qa/phase1_providers_20260708T141500Z/live_probe.md — see
# docs/qa/SECURITY_INCIDENT_gemini_key_leak_20260711.md. Root cause
# (§11.4.102): scripts/scan-secrets.sh already existed and already covers
# these key shapes, but was wired ONLY into the pre-push hook
# (scripts/git_hooks/pre-push) — no pre-commit content-level secret scan
# existed, so a committed evidence .md file with a real key never hit a
# gate before landing in a local commit. This script is wired into
# scripts/git_hooks/pre-commit (see section 4 there) to close that gap at
# the earliest possible point — before the secret is even committed, not
# merely before it is pushed.
#
# Scans a set of files, the whole tracked tree, or the staged diff for
# key-shaped patterns. Exits non-zero on any unallowlisted hit and prints
# ONLY "<file>:<line>" for each hit — the matched secret VALUE is never
# printed anywhere (§11.4.10 — never print any real key value).
#
# Usage:
#   scripts/secret_scan.sh                 # scan the whole tracked git tree
#   scripts/secret_scan.sh --staged        # scan staged (about-to-commit) content
#   scripts/secret_scan.sh <file> [<file>...]  # scan an explicit file set
#
# SCAN PATHS — the COMPLETE enumeration (round 18, 2026-08-13). Round 16
# asserted coverage instead of enumerating it and was wrong twice in a row
# (first "every consumer is covered by construction", then a file-scoped
# restatement of the same claim while ONE of this file's TWO enumerations
# was still unfixed). So the paths are listed, not claimed. There are exactly
# three modes, differing in WHERE the filename comes from:
#
#   mode    filename source                  quoting layer?  reader
#   ------  ------------------------------  --------------  --------------
#   tree    `git ls-files -z`               NUL (fixed r18) scan_disk_file
#   staged  `git diff --cached … -z`        NUL (fixed r16) scan_staged_file
#   files   this script's own "$@"          none (argv is   scan_disk_file
#                                           raw bytes)
#
# Only the two GIT-DERIVED modes ever pass through git's C-quoted display
# form, and both now share ONE enumerator (enumerate_git_z). `files` mode
# takes raw argv from the caller and has no quoting layer to defeat.
#
# WHAT A PATH IS READ AS (round 20 — the table above says where the NAME comes
# from; this says what CONTENT is scanned for it, which is a separate axis and
# used to be documented nowhere):
#
#   entry kind        tree / files (scan_disk_file)  staged (scan_staged_file)
#   ----------------  ----------------------------  -------------------------
#   regular file,     the working-tree bytes         the staged blob
#     present on disk  (`cat`, binaries filtered
#                      out by `grep -Iq .`)
#   regular file,     the INDEX content              the staged blob
#     shadowed by a    (`git show ":$f"`) — the
#     directory        disk slot holds no bytes
#   regular file,     the INDEX content              the staged blob
#     absent from      (`git show ":$f"`)
#     disk (plain rm)
#   symlink           the LINK's own target string   the blob = target string
#                     (`readlink`, never followed)   (`git show` rc=0)
#   gitlink 160000    skipped: no blob in this repo  skipped: no blob in this
#                                                    repo (`git show` rc=128)
#   untracked dir     skipped (files-mode argument)  n/a (never staged)
#   unreadable, or    recorded, NOT certified clean  recorded, NOT certified
#     index lookup
#     failed
#
# NOTE the deliberate FILES-MODE consequence: `scan_disk_file` is also the
# files-mode reader, so passing a SYMLINK as an explicit argument now scans the
# link's own content rather than dereferencing to the target file — measured,
# `secret_scan.sh <symlink-to-a-file-holding-a-key>` was rc=1 before round 20
# and is rc=0 after. That is correct for the committed-content charter (the
# target's bytes are not this repository's objects unless the target is itself
# tracked, in which case it is scanned under its own path), but a caller that
# deliberately hands this script a symlink expecting the TARGET scanned should
# resolve it first. No current caller does: all three production callers pass
# either `--staged` or a single regular file they just wrote themselves.
#
# HONEST BOUNDARY (§11.4.6): this enumerates the scan paths INSIDE THIS FILE.
# It says nothing about other processes: a consumer that runs its own
# `git diff`/`git ls-files` is covered only when IT is fixed — the precise
# error round 16 made.
#
# IN-REPO CALLERS — re-derive, do not trust this list (round 20: the round-18
# list was itself incomplete, omitting sec_capture_lib.sh; an enumeration that
# replaces an assertion has to be complete or it is the same defect one level
# up). Candidates come from:
#
#   git grep -n --untracked -e 'secret_scan\.sh' -- . \
#       ':!scripts/secret_scan.sh' ':!docs/' ':!*.db' \
#     | grep -vE ':[0-9]+:[[:space:]]*#'
#
# which printed 10 lines on 2026-08-13; five of them INVOKE this script:
#
#   caller                                          how it calls this script
#   ----------------------------------------------  ------------------------
#   scripts/git_hooks/pre-commit                     `--staged` (step 4)
#   scripts/gates/qa_transcript_redaction_gate.sh    files mode, 1 explicit path
#   scripts/qa/lib/sec_capture_lib.sh                files mode, 1 explicit path
#                                                    (Layer 3 of the QA scrub)
#   scripts/secret_scan_test.sh                      all three modes, in
#                                                    throwaway repos
#   scripts/git_hooks/testdata/                      `--staged` — UNTRACKED
#     hxc282_pre_fix_pre_commit.sh                   round-19 pre-fix fixture
#
# NONE of the five re-derives a file list of its own: the three production
# callers pass either `--staged` (this script's own enumerator) or a single
# explicit path they just wrote themselves.
#
# The other five lines are NOT invocations, and the next round should not
# re-add them: `.scan-secrets-allow` (an allowlist path glob),
# sec_capture_lib.sh's failure-MESSAGE text, two self-exclusion entries in the
# OTHER scanner (scripts/scan-secrets.sh), and this suite's backup filename.
# scripts/git_hooks/test_hooks.sh names this script only in a COMMENT — its
# own `SCANNER` variable is scripts/scan-secrets.sh — which is why the
# comment-stripping `grep -v` above drops it mechanically rather than by
# judgement (round 16 wrongly counted it as a caller; round 19 retracted that).
#
# Exit codes:
#   0 = no unallowlisted key-shaped pattern found
#   1 = one or more hits (caller must investigate, rotate if real, and
#       either remove the content or add an explicit redaction marker)
#   2 = usage / environment error (e.g. no grep found)
#
# Allowlist — TWO independent layers, either one suppresses a match:
#   1. Content-based (unchanged): a matched line is a false positive and
#      SKIPPED if it also contains (case-insensitive) any of "redacted",
#      "example", "..." (three literal dots — the "<...>" placeholder
#      shape). Covers redaction markers like
#      "<REDACTED-GEMINI-KEY-CONST-042-...>" and illustrative placeholders
#      like "AIzaSyEXAMPLE..." with no path-based entry needed.
#   2. Path-based (2026-07-11 addition): repo-relative-path globs read from
#      .scan-secrets-allow at the repo root — the SAME allowlist file
#      scripts/scan-secrets.sh already reads. Before this addition,
#      secret_scan.sh had NO path-based allowlist at all, so a confirmed
#      fabricated-fixture test file (e.g. a Go unit test embedding a PEM
#      body / GCP service-account JSON marker as a credential-parsing
#      fixture) could ONLY be silenced by adding a "redacted"/"example"/
#      "..." string INTO the fixture's own content — mutating test data
#      just to satisfy this scanner. Consulting .scan-secrets-allow lets a
#      confirmed false positive be allowlisted the same documented,
#      reviewable way scan-secrets.sh already supports, with no fixture
#      content changes and no drift between the two scanners' allowlists.
#
# Cross-references: §11.4.10 / §11.4.30 / §11.4.102 / §11.4.135 / §11.4.138 /
#   CONST-042; scripts/scan-secrets.sh (pre-push, broader file-type scan;
#   .scan-secrets-allow is now shared between both scanners);
#   scripts/git_hooks/pre-commit (wired here, section 4).

set -uo pipefail

# Capability guard (round-16, 2026-08-13) — the `--staged` mode enumerates the
# index with `mapfile -d ''` (NUL-delimited), which requires bash >= 4.4.
# Without this guard an older bash would fail the read, leave the file list
# unset, scan NOTHING, and — being a scanner whose silence means "clean" —
# print "OK: no unallowlisted key-shaped secret pattern found" and exit 0.
# That is the exact fail-open shape this round fixed in the pre-commit hook;
# it must not be re-introduced here by the very fix that closes the leak.
# The probe IS the condition (a version-number comparison would only be a
# proxy for it), and it BLOCKS rather than warning: a scanner that cannot
# read the index cannot certify it.
if ! ( mapfile -d '' -t _cap_probe < <(printf 'x\0') ) >/dev/null 2>&1; then
  {
    echo "FAIL: unsupported bash — cannot scan."
    echo "  This scanner needs \`mapfile -d ''\` (bash >= 4.4, 2016)."
    echo "  Probe \`mapfile -d '' -t v < <(printf 'x\\0')\` FAILED."
    echo "  bash running this script: ${BASH_VERSION:-<unknown>}"
    echo "  Refusing to report 'clean' on an index this bash cannot enumerate."
  } >&2
  exit 1
fi
unset _cap_probe

REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)

# ---------------------------------------------------------------------------
# grep binary resolution — avoid a hardcoded /usr/bin/grep (breaks on macOS
# Apple Silicon + Homebrew) and avoid an injected `ugrep` wrapper that some
# CLI-agent environments put on PATH ahead of GNU/BSD grep (its -I flag
# silently skips files this scanner needs to read).
# ---------------------------------------------------------------------------
GREP_BIN="$(command -v grep 2>/dev/null || true)"
if [ -z "$GREP_BIN" ] || [ "$GREP_BIN" = "$(command -v ugrep 2>/dev/null || true)" ]; then
  for candidate in /usr/bin/grep /usr/local/bin/grep /opt/homebrew/bin/grep /bin/grep; do
    if [ -x "$candidate" ]; then
      GREP_BIN="$candidate"
      break
    fi
  done
fi
if [ -z "$GREP_BIN" ]; then
  echo "ERROR: grep not found in PATH or standard locations" >&2
  exit 2
fi

# ---------------------------------------------------------------------------
# Key-shaped patterns (extended regex). Each entry: "label|pattern".
#
# COUNT — derive it, do not count the lines by eye (comment lines inside the
# array make an eyeball count wrong, which is how a round-18 note came to say
# "16"):
#
#   bash -c 'eval "$(awk "/^PATTERNS=\\(/,/^\\)/" scripts/secret_scan.sh)";
#            echo "${#PATTERNS[@]}"'          -> 13   (2026-08-13)
#
# Cost follows directly, and only tree mode pays it in bulk: scanning one file
# runs one `grep` per pattern plus one `grep -Iq .` binary probe, so a whole-
# tree scan is ~14 grep processes × the 9308 regular files in the census below.
# That is minutes, not seconds. Timed repeatedly on this host (64 CPUs, load
# ~4-5, warm page cache) across rounds 19 and 20 — 645 s, 643 s, 639 s, 639 s —
# i.e. ~10.7 minutes for the whole tracked tree, a spread under 1%. Comment-only
# edits between those runs cannot move it: the file is parsed once. That is why
# the pre-commit hook uses `--staged` (a handful of files) and never tree mode.
# Treat the duration as a property of THIS host and corpus, not of the scanner,
# and re-time it before quoting it: a round-18 note put it at "~45 min (16 grep
# subprocesses per file × 9308)", which was wrong on the duration (~4x), on the
# subprocess count (13 patterns, not 16) and on what 9308 counts (regular
# files, not `git ls-files` entries, of which there are 9439).
# ---------------------------------------------------------------------------
PATTERNS=(
  "Google API key|AIza[0-9A-Za-z_-]{20,}"
  "OpenAI API key|sk-[A-Za-z0-9]{20,}"
  "AWS access key|AKIA[0-9A-Z]{16}"
  "GitHub personal access token|ghp_[A-Za-z0-9]{30,}"
  "Private key header|-----BEGIN (RSA|EC|OPENSSH|DSA|PGP) PRIVATE KEY-----"
  "Private key header (generic)|-----BEGIN PRIVATE KEY-----"
  "Slack bot token|xoxb-[0-9]{9,}-[0-9]{9,}-[A-Za-z0-9]{20,}"
  "HuggingFace token|hf_[A-Za-z0-9]{30,}"
  # ---------------------------------------------------------------------
  # Defense-in-depth hardening pass (§11.4.138 — closes additional key
  # classes real .env files in this project could hold; the project ships
  # ~30 provider .env aliases including xai, gcp, azure). Each pattern
  # below was verified against the whole tracked tree before being added:
  # a low-detection-threshold version would have false-positived on
  # existing legitimate doc placeholders (e.g. "sk-ant-your-key") and Go
  # unit-test fixtures (e.g. "sk-ant-realvalue-1234567890", 20 chars) —
  # thresholds were tuned to clear the longest observed in-repo fixture
  # with a safety margin while staying far below real key entropy length.
  # See scratchpad/r41_guard_hardening.md for the full false-positive
  # audit this session performed before landing these patterns.
  # ---------------------------------------------------------------------
  # xAI (Grok) API key: "xai-" + a pure-alphanumeric body (no hyphens in
  # the real key body). Longest in-repo placeholder observed was
  # "xai-secret-key-12345" (16 alnum-run chars, broken by hyphens); the
  # 20-char pure-alnum-run requirement clears every such placeholder.
  "xAI API key|xai-[A-Za-z0-9]{20,}"
  # Anthropic API key, explicit label (diagnostic aid). NOTE: the existing
  # generic "OpenAI API key|sk-[A-Za-z0-9]{20,}" pattern does NOT already
  # cover this shape — real/leaked Anthropic keys are "sk-ant-api03-..."
  # and the hyphen after "ant" breaks the OpenAI pattern's required
  # 20-char pure-alnum run at only 3 characters ("ant"), so this is a
  # genuine coverage gap being closed, not merely a redundant label.
  # Threshold is 30 (not 20) specifically because the longest sk-ant-
  # fixture already committed in this repo's Go unit tests is
  # "sk-ant-realvalue-1234567890" (exactly 20 chars after "sk-ant-");
  # 30 clears it with a 10-char margin while staying far below real
  # Anthropic key length (~100+ chars for the sk-ant-api03-... format).
  "Anthropic API key (explicit)|sk-ant-[A-Za-z0-9_-]{30,}"
  # GCP service-account JSON credential marker: the literal
  # "type": "service_account" key-value pair is unique to GCP
  # service-account key files and does not occur in ordinary prose
  # (verified: doc files in this repo that merely discuss "service
  # account" in prose do not match this exact quoted-JSON shape).
  "GCP service-account JSON marker (type)|\"type\"[[:space:]]*:[[:space:]]*\"service_account\""
  # GCP service-account JSON credential marker: the "private_key_id" JSON
  # key is likewise unique to GCP service-account key files. Matching on
  # the key name + following colon (not a specific value shape) keeps
  # this simple and low-FP since the exact literal never occurs outside
  # that JSON shape.
  "GCP service-account JSON marker (private_key_id)|\"private_key_id\"[[:space:]]*:"
  # Azure key/secret, env-assignment shape: an AZURE_*KEY or AZURE_*SECRET
  # env-var name, an assignment operator, and a 32+ char hex value (the
  # shape used by Azure Cognitive Services / Azure OpenAI resource keys).
  # Deliberately narrow: Azure also issues base64 Storage-account keys and
  # mixed-charset AD client secrets, whose shapes are not "cleanly
  # definable" without materially raising false-positive risk (see
  # scratchpad/r41_guard_hardening.md for the honest skip note) — those
  # are NOT covered by this pattern.
  "Azure key/secret (env-assignment, hex)|AZURE_[A-Z0-9_]*(KEY|SECRET)[[:space:]]*[:=][[:space:]]*[\"']?[A-Fa-f0-9]{32,}[\"']?"
)

# Content-based allowlist markers (case-insensitive substrings): a matched
# line containing "redacted", "example" or "..." is a redaction marker /
# illustrative placeholder, not a real leaked secret.
#
# The markers live in the `case` inside scan_file_content — NOT in a variable
# here. A dead `ALLOW_MARKER_RE='redacted|example|\.\.\.'` stood at this spot
# from the file's birth commit (5aae8c0c, 2026-07-11) and was removed in round
# 21 after a §11.4.124 git-history investigation: it was NEVER wired, so no
# call site was lost. Evidence — `git log -S/-G 'ALLOW_MARKER_RE' --oneline
# --all` returns exactly ONE commit (5aae8c0c) against a validated instrument
# (`-S scan_staged_file` on the same file returns that same single commit);
# `git show 5aae8c0c:scripts/secret_scan.sh | grep -c ALLOW_MARKER_RE` -> 1,
# i.e. the definition and nothing else, while the hardcoded `case` was ALREADY
# present at line 114 of that same birth revision. Definition and real logic
# were born together and never met, across all four commits that have ever
# touched this file.
#
# It is called out rather than silently deleted because the failure mode was
# SILENT: editing that variable to add a marker changed NOTHING, and the next
# reader would have had no way to tell. If the marker set needs to change,
# edit the `case` in scan_file_content — and keep it in step with the
# operator-facing remediation text further down, which names the same three.

# Files this scanner itself must never flag (it legitimately quotes the
# pattern literals and label text above, and its own test plants fixture
# secrets in a temp dir, not in this file).
# _index_self_mode <path> — echo the index mode of the path's OWN entry, or
# nothing when the path has no entry of its own. Returns 0 when the lookup
# ANSWERED (even if the answer is "no entry"), non-zero when it could not.
#
# Two traps this exists to avoid, both found by review of round 20:
#
#  1. `:(literal)` disables GLOB expansion but NOT directory-prefix matching,
#     so `git ls-files --stage -- ':(literal)qa'` on a tracked-content
#     directory answers with the CHILD rows (`100644 … <tab>qa/file.txt`).
#     Reading `NR==1`'s mode therefore returns a DIFFERENT FILE's mode — the
#     chooser-reads-another-row hazard. Every row is matched on its PATH field.
#  2. The exit status must not be read through a pipeline, or it belongs to the
#     last stage instead of git. The rows are collected first, git's status is
#     carried out on a NUL sentinel (the `enumerate_git_z` idiom), and only
#     then are they inspected.
#
# `-z` + `mapfile -d ''` keeps this correct for paths containing tabs, quotes,
# newlines or non-ASCII bytes, which the plain (C-quoting) output would mangle.
_index_self_mode() {
  local _p="$1"
  local -a _rows=()
  mapfile -d '' -t _rows < <(
    env -u GIT_LITERAL_PATHSPECS -u GIT_GLOB_PATHSPECS \
        -u GIT_NOGLOB_PATHSPECS -u GIT_ICASE_PATHSPECS \
      git ls-files --stage -z -- ":(literal)$_p" 2>/dev/null
    printf 'ENUMRC=%s\0' "$?"
  )
  local _n=${#_rows[@]}
  [ "$_n" -eq 0 ] && return 90
  local _sent="${_rows[$((_n - 1))]}"
  case "$_sent" in ENUMRC=*) ;; *) return 91 ;; esac
  local _rc="${_sent#ENUMRC=}"
  [ "$_rc" -ne 0 ] && return "$_rc"
  unset "_rows[$((_n - 1))]"
  local _row _path
  for _row in ${_rows[@]+"${_rows[@]}"}; do
    # row shape: "<mode> <sha> <stage>\t<path>"
    _path="${_row#*$'\t'}"
    [ "$_path" = "$_p" ] || continue
    _row="${_row%%$'\t'*}"
    printf '%s\n' "${_row%% *}"
    return 0
  done
  return 0   # answered: the path has no entry of its own
}

is_self() {
  case "$1" in
    */scripts/secret_scan.sh|scripts/secret_scan.sh) return 0 ;;
    */scripts/secret_scan_test.sh|scripts/secret_scan_test.sh) return 0 ;;
  esac
  return 1
}

# ---------------------------------------------------------------------------
# Path-based allowlist (2026-07-11 addition) — reads the SAME
# .scan-secrets-allow file scripts/scan-secrets.sh already consumes, so a
# confirmed fabricated-fixture path is documented and allowlisted ONCE, for
# BOTH scanners, instead of drifting into two separate lists.
# ---------------------------------------------------------------------------
ALLOWLIST_FILE="$REPO_ROOT/.scan-secrets-allow"
ALLOW_PATTERNS=()
if [ -f "$ALLOWLIST_FILE" ]; then
  while IFS= read -r _line; do
    _line="${_line#"${_line%%[![:space:]]*}"}"
    _line="${_line%"${_line##*[![:space:]]}"}"
    if [ -z "$_line" ] || [ "${_line:0:1}" = "#" ]; then
      continue
    fi
    ALLOW_PATTERNS+=("$_line")
  done < "$ALLOWLIST_FILE"
fi

is_path_allowlisted() {
  # $1 = repo-relative (or ./-prefixed) file path.
  local filepath="${1#./}"
  local allow_glob basename_path
  for allow_glob in "${ALLOW_PATTERNS[@]}"; do
    allow_glob="${allow_glob#./}"
    basename_path="${filepath##*/}"
    # shellcheck disable=SC2254
    case "$filepath" in
      $allow_glob)     return 0 ;;
      */$allow_glob)   return 0 ;;
      */"$allow_glob") return 0 ;;
    esac
    # shellcheck disable=SC2254
    case "$basename_path" in
      $allow_glob) return 0 ;;
    esac
  done
  return 1
}

found=0
hits=""

# scan_file_content <label> <pattern> <display-path> <content>
# Greps $content for $pattern via a temp buffer (process substitution),
# applies the content-based allowlist per matched line, and records
# "<display-path>:<line>" for any unallowlisted hit. NEVER echoes the
# matched line/value.
scan_file_content() {
  local label="$1" pattern="$2" display_path="$3" content="$4"
  local lineno rest
  while IFS=: read -r lineno rest; do
    [ -z "$lineno" ] && continue
    case "$rest" in
      *[Rr][Ee][Dd][Aa][Cc][Tt][Ee][Dd]*|*[Ee][Xx][Aa][Mm][Pp][Ll][Ee]*|*'...'*)
        continue
        ;;
    esac
    hits="${hits}"$'\n'"  ${display_path}:${lineno}  (${label})"
    found=1
  # -e explicitly disambiguates the pattern from an option: several of our
  # patterns (the PEM private-key headers) start with "-----", which some
  # grep-compatible implementations (e.g. a ugrep shim providing a bare
  # "grep" on PATH) otherwise misparse as an unrecognized flag rather than
  # a positional PATTERN argument.
  done < <(printf '%s\n' "$content" | "$GREP_BIN" -nE -e "$pattern" 2>/dev/null || true)
}

# scan_disk_file <path>  — grep a real file on disk, path also used as the
# display path. Used for whole-tree scans and explicit-file-arg scans.
scan_disk_file() {
  local f="$1"
  is_self "$f" && return 0
  is_path_allowlisted "$f" && return 0
  local content _rc
  if [ -L "$f" ]; then
    # SYMLINKS ARE TESTED FIRST, AND NEVER FOLLOWED (round 20, 2026-08-13).
    #
    # POSIX: "with the exception of the -h pathname and -L pathname primaries,
    # if a pathname argument is a symbolic link, test shall evaluate the
    # expression by resolving the symbolic link and using the file referenced
    # by the link" (Open Group Base Specifications Issue 7, `test`). So both
    # `[ -f ]` and `[ -d ]` below FOLLOW a link. Before this branch existed a
    # tracked symlink was therefore judged by whatever it pointed AT, on that
    # filesystem, at that moment — never by its own content:
    #
    #   link -> a directory     the gitlink branch skipped it silently, so a
    #                           key in the LINK STRING was never scanned;
    #   link -> a regular file  `cat` opened and scanned THAT file even when it
    #                           lay outside the repository — a repo-escape read
    #                           (the class behind CVE-2025-8110 /
    #                           CVE-2026-52811 / CVE-2026-71556) whose result
    #                           depends on ambient filesystem state instead of
    #                           the committed tree (§11.4.50 determinism).
    #
    # A symlink DOES have content of its own here: git stores mode 120000 with
    # the target path string as the blob body, which is what `git show :<path>`
    # and `git grep` return — so scanning that string is the git-native answer,
    # and it is what git-secrets gets for free by driving everything through
    # `git grep`. `readlink` reads the link itself and never opens the target,
    # mirroring trufflehog's Lstat-before-decide pattern; gitleaks
    # (`--follow-symlinks` defaults to false) and detect-secrets (`ignores files
    # that are not files (e.g. links)`) skip instead, which is safe but leaves
    # a key smuggled into a target path string unscanned. Sources verified
    # 2026-08-13: pubs.opengroup.org/onlinepubs/9699919799/utilities/test.html;
    # git-scm.com/book/en/v2/Git-Internals-Git-Objects; git-scm.com/docs/
    # git-cat-file; github.com/gitleaks/gitleaks/blob/master/cmd/detect.go;
    # github.com/Yelp/detect-secrets/blob/master/docs/filters.md;
    # github.com/trufflesecurity/trufflehog/blob/main/pkg/sources/filesystem/
    # filesystem.go; snyk.io/blog/symlinks-are-still-scary/.
    #
    # NOTHING IN THE REPOSITORY LOSES COVERAGE: a target that is itself tracked
    # is enumerated and scanned under its own canonical path (Test 33); a target
    # that is NOT tracked is not this repository's content (Test 32).
    #
    # CORROBORATION (measured 2026-08-13) — this branch did not invent a policy,
    # it made disk mode match the one this scanner ALREADY had. `--staged` reads
    # content with `git show ":$f"`, which for mode 120000 returns the target
    # path string; so with a key placed in a tracked symlink's target string the
    # round-19 scanner returned rc=1 under `--staged` and rc=0 in tree mode —
    # two modes of ONE scanner disagreeing about the same key in the same repo,
    # exactly the round-18 finding one layer over. Both now return rc=1, and
    # both return rc=0 for a clean target string.
    # PORTABILITY, stated honestly (§11.4.81): `readlink -- "$f"` is verified
    # on this Linux host (GNU coreutils); the `--` matters because a tracked
    # path may begin with a dash. macOS/BSD `readlink` parses options with
    # getopt(3), which honours `--`, so it is expected to work — but that is
    # REASONED, not measured, and none of the sources cited above covers it.
    # An unverified platform is not a silent risk here: if readlink fails for
    # any reason the path is recorded NOT certified clean, i.e. fail-closed.
    content=$(readlink -- "$f" 2>/dev/null)
    _rc=$?
    if [ "$_rc" -ne 0 ]; then
      hits="${hits}"$'\n'"  ${f}  (UNREADABLE symlink — readlink rc=${_rc}; NOT certified clean)"
      found=1
      return 0
    fi
  elif [ -f "$f" ]; then
    # Skip binary files: a key-shaped ASCII secret cannot meaningfully live in
    # one, and reading NUL-containing content into a shell variable emits noisy
    # "ignored null byte" warnings on every commit. `grep -Iq .` reports a
    # binary file as not-matching (-I treats binary as no match).
    #
    # The exit status is TRIAGED, not merely tested for non-zero (round 18).
    # A bare `|| return 0` here conflated two different answers: measured with
    # GNU grep, `grep -Iq . <file>` returns 1 for "binary or empty" (a real
    # answer — nothing scannable) but 2 for "could not read it" (no answer at
    # all — e.g. mode 000). The old form certified an UNREADABLE file clean,
    # the same silent-skip shape round 16 removed from scan_staged_file and
    # round 18 removed from the enumeration.
    "$GREP_BIN" -Iq . -- "$f" 2>/dev/null
    _rc=$?
    if [ "$_rc" -eq 2 ]; then
      hits="${hits}"$'\n'"  ${f}  (UNREADABLE on disk — grep rc=2; NOT certified clean)"
      found=1
      return 0
    fi
    [ "$_rc" -ne 0 ] && return 0   # rc=1 => binary or empty: nothing to scan
    content=$(cat "$f" 2>/dev/null)
    _rc=$?
    if [ "$_rc" -ne 0 ]; then
      hits="${hits}"$'\n'"  ${f}  (UNREADABLE on disk — cat rc=${_rc}; NOT certified clean)"
      found=1
      return 0
    fi
  elif [ -d "$f" ]; then
    # The ROUTINE occupant of this branch is a gitlink — a submodule pointer,
    # mode 160000 — which has no file content in THIS repository; the
    # submodule's own tree is scanned by running this scanner inside that
    # submodule. Routine is not the same as only: see the SECOND CORRECTION
    # below for the other occupants, which is why the code asks the index
    # rather than inferring from `[ -d ]`. Skipping a gitlink is not, and
    # never was, a defect — it is the routine case, not
    # an anomaly, so blocking on it would be a false-positive refusal
    # (§11.4.201) rather than a finding.
    #
    # CENSUS, re-derive rather than trust (round 20 correction: round 18 said
    # "131 of 9439 entries are gitlinks", which was wrong — 131 was the number
    # of paths REACHING this branch, and the 131st was not a gitlink at all but
    # the tracked symlink skills/media-validator, which arrived here only
    # because `[ -d ]` follows links. It was being skipped under a rationale
    # that was false for it: it is not a gitlink, its content IS in this
    # repository, and no submodule scan covers it. That is now the `-L` branch
    # above. Symlinks aside, this branch still receives more than gitlinks —
    # see the SECOND CORRECTION below, which is why nothing here is inferred
    # from `[ -d ]` any more.)
    #
    #   git ls-files | wc -l                                    -> 9439
    #   git ls-files -s | awk '{print $1}' | sort | uniq -c      -> 7950 100644
    #                                                              1358 100755
    #                                                                 1 120000
    #                                                               130 160000
    #
    # measured 2026-08-13: 9308 regular files, 1 symlink, 130 gitlinks.
    #
    # ROUND-20 SECOND CORRECTION — "this branch sees gitlinks only" was ALSO
    # too strong, and an independent review produced the counterexample: a
    # tracked REGULAR file whose worktree slot has been replaced by a directory
    # (`rm f && mkdir f`) still has mode 100644 in the index and its content is
    # still readable there, but `[ -d ]` sent it here and it was returned
    # CLEAN. Measured on a throwaway repo: key-bearing tracked file -> rc=1;
    # same file dir-shadowed -> rc=0 "OK: no unallowlisted key-shaped secret
    # pattern found (mode=tree)", while `git show ":$f"` still yielded the
    # key-bearing blob. HEAD's scanner and the round-19 mirror behave
    # identically, so the BEHAVIOUR is pre-existing — what was new was a
    # comment asserting it could not happen.
    #
    # So this branch no longer asserts what it is looking at, it ASKS, exactly
    # as scan_staged_file now does: only a real gitlink (index mode 160000) is
    # skipped. A tracked non-gitlink gets its INDEX content scanned — the same
    # source the else-branch below already uses for a path that is not on disk
    # at all, which is the closest sibling case. An untracked path (empty mode,
    # e.g. a plain directory handed to files mode) is skipped as before.
    # The lookup's EXIT STATUS is kept separate from its output, because an
    # empty answer has two completely different meanings and conflating them
    # was itself a fail-open (found by review of the first version of this very
    # branch, which piped straight into `awk` and read only the text):
    #
    #   rc=0 + empty output  -> the path is genuinely NOT TRACKED (measured:
    #                           `git ls-files --stage -- ':(literal)<untracked>'`
    #                           exits 0 and prints nothing)
    #   rc!=0                -> the lookup FAILED and we know NOTHING (measured:
    #                           corrupt .git/index -> rc=128, empty output)
    #
    # Piping into `awk` discards git's status — `$?` then reports awk's — so a
    # corrupt index looked exactly like "untracked" and a key-bearing tracked
    # file was certified clean in files mode (rc=0 "OK"), while the identical
    # call on a healthy index returned rc=1. Tree and staged modes were shielded
    # (the same corrupt index trips `enumerate_git_z` first), but files mode is
    # a production surface. So: capture the output, then read the status, then
    # parse — never a pipeline whose status belongs to the wrong process.
    #
    # Consequence worth stating (§11.4.6): running files mode OUTSIDE any git
    # repository makes every lookup fail, so a directory argument there is now
    # REFUSED rather than skipped. That is the fail-closed direction, and no
    # caller does it — all three production callers run inside this repo and
    # pass either `--staged` or a regular file they just wrote.
    local _dmode _dls_rc
    _dmode=$(_index_self_mode "$f")
    _dls_rc=$?
    if [ "$_dls_rc" -ne 0 ]; then
      hits="${hits}"$'\n'"  ${f}  (index lookup FAILED — rc=${_dls_rc}; NOT certified clean)"
      found=1
      return 0
    fi
    # No entry of its own => a gitlink, an ordinary directory that merely
    # CONTAINS tracked files, or an untracked directory handed to files mode.
    # None of the three has content here; the tracked children are enumerated
    # and scanned under their own paths.
    if [ "$_dmode" = "160000" ] || [ -z "$_dmode" ]; then
      return 0
    fi
    content=$(git show ":$f" 2>/dev/null)
    _rc=$?
    if [ "$_rc" -ne 0 ]; then
      hits="${hits}"$'\n'"  ${f}  (tracked, but shadowed on disk by a directory and UNREADABLE from the index — rc=${_rc}; NOT certified clean)"
      found=1
      return 0
    fi
  else
    # Tracked (or explicitly named) path that is NOT on disk as a regular file
    # or directory: e.g. a tracked file deleted from the working tree with a
    # plain `rm` (still listed by `git ls-files`, content still in the index),
    # or a caller passing a path that does not exist.
    #
    # ROUND-20 CORRECTION: this used to say "e.g. a `git rm`-staged deletion
    # still listed by `git ls-files`". That case cannot occur — `git rm` drops
    # the index entry at command time, so the path is never enumerated here.
    # Measured: after `git rm f.txt`, `git ls-files | grep -cx f.txt` -> 0;
    # after a plain `rm f.txt` -> 1. The example named an impossible input
    # while the reachable one went unnamed.
    #
    # The former blanket `[ -f "$f" ] || return 0` reported ALL of these
    # CLEAN — indistinguishable, in the caller and in this script's exit
    # status, from "read it and found nothing" (the same silent-skip shape
    # round 16 removed from scan_staged_file). Instead: for a deletion still
    # present in the index the tracked CONTENT is still readable there, so
    # scan that; only when the content cannot be obtained AT ALL is the path
    # recorded as NOT certified.
    # A GITLINK whose worktree directory is absent (never initialised, or
    # rmdir'd) lands here rather than in the `-d` branch, and `git show` cannot
    # resolve it either — rc=128, because the submodule's commit object is not
    # in this repository. Refusing it would be a false diagnostic (the path IS
    # in the index, at mode 160000) and a §11.4.201 over-block on an ordinary
    # uninitialised submodule. Measured: with the directory present the tree
    # scan skips it (rc=0); with it absent, both this scanner and the round-19
    # one refused (rc=1) — i.e. PRE-EXISTING, not introduced by round 20, and
    # it also falsified the gitlink row of the table at the top of this file.
    # So the same question the `-d` branch asks is asked here.
    local _emode _erc
    _emode=$(_index_self_mode "$f")
    _erc=$?
    if [ "$_erc" -eq 0 ] && [ "$_emode" = "160000" ]; then
      return 0
    fi
    content=$(git show ":$f" 2>/dev/null)
    _rc=$?
    if [ "$_rc" -ne 0 ]; then
      hits="${hits}"$'\n'"  ${f}  (UNREADABLE — neither a readable file on disk nor readable from the index, rc=${_rc}; NOT certified clean)"
      found=1
      return 0
    fi
  fi
  local entry label pattern
  for entry in "${PATTERNS[@]}"; do
    label="${entry%%|*}"
    pattern="${entry#*|}"
    scan_file_content "$label" "$pattern" "$f" "$content"
  done
}

# scan_staged_file <path> — grep the STAGED BLOB (git index), not the
# working tree, so the guard checks exactly what is about to be committed
# (mirrors the §11.4.84 mutation-residue section of scripts/git_hooks/pre-commit).
scan_staged_file() {
  local f="$1"
  is_self "$f" && return 0
  is_path_allowlisted "$f" && return 0
  local content rc
  content=$(git show ":$f" 2>/dev/null)
  rc=$?
  if [ "$rc" -ne 0 ]; then
    # A scan that CANNOT READ a file must not silently pass it (round-16
    # fix, 2026-08-13). The former `|| return 0` reported the file clean —
    # indistinguishable, in the caller and in this script's exit status,
    # from "read it and found nothing". That silent-skip is exactly how a
    # staged credential reached a commit while this scanner printed
    # "OK: no unallowlisted key-shaped secret pattern found".
    #
    # ROUND-20 CORRECTION — the paragraph that stood here was FALSE, and the
    # false claim was hiding a live over-block. It read: "measured, `git show
    # ":$f"` returns rc=0 for a GITLINK (submodule pointer, mode 160000 — it
    # shows the commit object) … so ordinary submodule-pointer and symlink
    # commits are NOT affected by this becoming blocking."
    #
    # The SYMLINK half is true. The GITLINK half is not, and this same working
    # tree already said so 400 lines away in scripts/git_hooks/pre-commit:
    # "gitlink `git show ":submodules/helix_agent"` -> rc=128 -> skipped".
    # Re-measured 2026-08-13 (git 2.50.1):
    #
    #   git show ":submodules/helix_agent" ; echo $?   -> 128 (fatal: bad object)
    #   git show ":skills/media-validator" ; echo $?   ->   0 (target string)
    #
    # A submodule's commit object lives in the SUBMODULE's object store, not
    # the parent's, so the parent cannot resolve it. The earlier "measured"
    # rc=0 can only have come from a fixture whose gitlink sha happened to be
    # an object of the probing repo itself — a §11.4.199 non-representative
    # reproduction.
    #
    # CONSEQUENCE, measured end-to-end before this fix: staging ANY submodule
    # pointer bump made this scanner exit 1, so pre-commit step 4 BLOCKED the
    # commit. HEAD's committed scanner exits 0 on the same input. In a repo
    # with 130 gitlinks — three of them modified in this very working tree —
    # that is a §11.4.201 false-positive refusal on a routine operation, and
    # it was invisible to both suites because neither ever staged a gitlink.
    #
    # So the gitlink case is triaged EXPLICITLY, not covered by a claim. The
    # rationale is the one the disk-mode `-d` branch already uses: a gitlink
    # has no blob in THIS repository, and the submodule's own tree is scanned
    # by running this scanner inside that submodule. The check is deliberately
    # NARROW — only an index mode of exactly 160000 is exempted; every other
    # unreadable path still refuses to certify.
    #
    # ROUND-21 FIX: the lookup goes through `_index_self_mode`, the SAME
    # hardened helper the two disk-mode sites use — ONE idiom in this file,
    # not two. It keeps the env-scrub (a pathspec-magic environment variable
    # would otherwise make the mode come back empty, which here fails CLOSED:
    # empty != 160000 => still refused) AND answers about the path ITSELF.
    # The bare `… | awk 'NR==1{print $1}'` that stood here reads whatever row
    # git printed FIRST, and `:(literal)` suppresses GLOB expansion but NOT
    # directory-prefix matching, so for a path that is a directory prefix the
    # first row is a CHILD's. Measured on this repo, 2026-08-13:
    #
    #   raw    ':(literal)submodules' | awk 'NR==1{print $1}'  -> 160000
    #          (the child row `submodules/agentic`, a gitlink)
    #   helper _index_self_mode submodules                     -> "" (rc=0)
    #
    # 160000 is exactly the value that RETURNS EARLY here, so the raw form
    # would have certified such a path clean without reading anything.
    #
    # That input is NOT reachable in this mode — this is defence in depth,
    # not a live-bug repair — and the reason is STRUCTURAL, so it is recorded
    # here rather than left to be re-derived (measured, git 2.50.1):
    #
    #   (a) staged mode enumerates `--diff-filter=ACMR`, so a staged DELETION
    #       — the only listed path that can lack an index entry — never
    #       arrives here. With `gone.txt` deleted and `kept.txt` modified,
    #       plain `--name-only` lists BOTH; `--diff-filter=ACMR` lists only
    #       `kept.txt`, and `gone.txt` has 0 index rows.
    #   (b) the index cannot hold a blob at P and entries under P/ at once, so
    #       a path that HAS its own entry cannot also be a directory prefix.
    #       Porcelain and plumbing preserve that differently, and the
    #       difference is worth stating because only one of them is visible
    #       in normal use: `git add sub/child.txt` over a blob at `sub`
    #       silently REPLACES it (rc=0; the index then holds only
    #       `sub/child.txt`), while `git update-index --add --cacheinfo
    #       100644,<sha>,sub` REFUSES — "error: 'sub' appears as both a file
    #       and as a directory", rc=128. "git refuses" is therefore true of
    #       the plumbing form only; a `git add` will show you a replacement.
    local _mode _mrc
    _mode=$(_index_self_mode "$f")
    _mrc=$?
    if [ "$_mrc" -eq 0 ] && [ "$_mode" = "160000" ]; then
      return 0
    fi
    hits="${hits}"$'\n'"  ${f}  (UNREADABLE from the index — rc=${rc}; NOT certified clean)"
    found=1
    return 0
  fi
  local entry label pattern
  for entry in "${PATTERNS[@]}"; do
    label="${entry%%|*}"
    pattern="${entry#*|}"
    scan_file_content "$label" "$pattern" "$f" "$content"
  done
}

# ---------------------------------------------------------------------------
# enumerate_git_z <git-command…>  — run a git command that emits a
# NUL-DELIMITED path list (`-z`) and read it into the global array _ENUM_OUT.
# Returns git's OWN exit status, or 90 (no sentinel — the read itself failed)
# or 91 (sentinel malformed — output truncated/corrupt).
#
# ONE enumerator, shared by BOTH git-derived modes, because the defect it
# closes was caused by having TWO of them. This file carried two independent
# enumerations; round 16 converted `--staged` to `-z` and left `tree` on the
# newline-delimited `git ls-files`, so the identical C-quoting fail-open
# survived in tree mode for a full round. MEASURED against the pre-fix
# scanner, one tracked file, synthetic Google-shaped key, filename the ONLY
# variable (fresh repo per row, tree mode = no arguments):
#
#     CONTROL  secret.md   rc=1  "FAIL: key-shaped secret pattern(s) found"
#     ATTACK   sécret.md   rc=0  "OK: no unallowlisted … found (mode=tree)"
#              tracked as: "s\303\251cret.md"
#
# The same run's `--staged` mode returned rc=1 for BOTH names — i.e. the two
# modes of ONE scanner disagreed about the same key in the same repo, and the
# mode documented at the top of this file as "scan the whole tracked git tree"
# was the fail-open one.
#
# WHAT THAT DOES AND DOES NOT IMPLY for the §11.4.75 bypass audit trail
# (round-20 correction; round 18 wrote "docs/audit/bypass_events.md rows 1 and
# 2 both cite tree-mode exit 0 as their clean-tree evidence", and only row 2
# does). Row 1 cites "both files verified secret-clean via
# scripts/secret_scan.sh exit 0" — FILES mode on two named paths, which takes
# raw argv and never passes through git's C-quoted display form, so this defect
# could not have touched it. Row 2 cites "git tree verified secret-clean", and
# that IS a tree-scope claim resting on this mode.
#
# Even for row 2, this defect is not what contradicts it: the repository has
# ZERO tracked paths that git would C-quote, then and now —
#
#   git ls-tree -r -z --name-only <rev> | tr '\0' '\n' \
#     | LC_ALL=C grep -cP '[^\x20-\x7E]'
#
# returns 0 at HEAD, at 358ddc13 (row 1) and at 9e4de2ba (row 2) — so the
# fail-open had nothing here to hide. What contradicted row 2 was a set of
# pre-existing findings the tree scan REPORTED, all predating the row's date
# and unrelated to this enumeration; they were tracked as HXC-331.
#
# ROUND-21 UPDATE — that paragraph is now HISTORY, and its tense is corrected
# rather than deleted, because as written in the present it would SHIP FALSE.
# The HXC-331 remediation WORKED. Re-measured 2026-08-13 by running this
# scanner's tree mode end to end over all 9439 tracked entries (680 s):
#
#   scripts/secret_scan.sh   -> rc=0, ZERO finding lines
#   "OK: no unallowlisted key-shaped secret pattern found (mode=tree)"
#
# The silencing is ATTRIBUTED, not assumed: commit d634cc86 ("fix(HXC-331):
# triage the seven scanner findings, and correct the audit row that called the
# tree clean") added exactly four `.html` export entries to
# .scan-secrets-allow, and they are COMMITTED — the functional (non-comment)
# content of HEAD's allowlist and of this working tree's is identical, 53
# entries each, so the clean result rests on nothing uncommitted.
#
# "Clean" here means SUPPRESSED-BY-DECISION, not absent (§11.4.6). Control
# pair on those same four exports, run this round:
#
#   inside the repo, allowlist active   -> rc=0, silenced
#   outside it, no allowlist loadable   -> rc=1, 6 findings
#                                          (3x private-key header,
#                                           2x OpenAI key, 1x AWS access key)
#
# Delete those four entries and the six come straight back.
#
# RECONCILED (round 24, 2026-08-13 — this stood open as UNRECONCILED, reading
# "d634cc86's subject says SEVEN findings ... the seventh is not accounted for
# by anything measured here". That was true OF THE CONTROL and false of the
# commit, so it is corrected rather than deleted). d634cc86's subject is
# CORRECT: the true total is SEVEN. Six is the control's SCOPE, not the
# commit's claim — the control above measures the four allowlist entries, and
# the commit counted the whole triage. The decomposition is 7 findings across
# 5 files, and it was already named in docs/audit/bypass_events.md by
# d634cc86 ITSELF (`git blame -L 38,60 -- docs/audit/bypass_events.md` -> all
# d634cc86). Re-derived this round against a pinned copy of this scanner, not
# taken from that note:
#
#   6  the four `.html` exports above ......... rc=1, no allowlist loadable
#   1  scripts/test-verify-all-constitution-rules.sh:128
#                                              ("Private key header (generic)",
#                                               pre-fix content, d634cc86^)
#   -
#   7  total
#
# WHY THE CONTROL IS STRUCTURALLY BLIND TO THE SEVENTH — this is a property of
# the measurement, not an oversight in it: the seventh has NO allowlist entry
# to delete. d634cc86 fixed it with a LINE-LEVEL CONTENT MARKER on the BEGIN
# line of the G5 heredoc fixture, deliberately not an allowlist entry, because
# that gate matches on the `.pem` FILENAME and an entry would have blinded a
# whole live executable permanently. `grep test-verify .scan-secrets-allow`
# returns nothing at d634cc86 and nothing at HEAD. So "delete those four
# entries and count what returns" measures a strictly SMALLER set than the
# commit counted; six was never a number in contradiction with seven.
#
# Proof the two suppressions are independent mechanisms, run with NO allowlist
# loadable at all (which no allowlist edit can influence):
#
#   pre-fix content  (d634cc86^) -> rc=1, 1 finding at :128
#   post-fix content (d634cc86)  -> rc=0, silenced by the in-line marker
#
# Honest boundary (§11.4.6): this reconciles the COUNT — 6 + 1 = 7, and the
# subject line is sound. It re-states nothing about whether those seven were
# benign; that determination is d634cc86's own, and its scope note ("zero real
# credentials" is about THOSE SEVEN FINDINGS, not about the tree) still binds.
#
# MECHANISM: `git ls-files` and `git diff --cached --name-only` both emit
# git's C-QUOTED DISPLAY FORM (core.quotePath defaults to true) for any path
# holding a byte outside printable ASCII, a quote, a backslash or a control
# character — `sécret.md` arrives as the literal 18-character string
# `"s\303\251cret.md"`, quotes and octal escapes included. That string is not
# a path: `[ -f ]` on it is false and `git show ":$f"` cannot resolve it, so
# each mode's own silent-skip branch passed the file. `-z` is NUL-delimited
# and quotePath-independent; the stream is read into an ARRAY because command
# substitution silently drops NUL bytes.
#
# A single shared enumerator makes both modes correct by construction: a mode
# added later cannot re-introduce the defect by copying the old idiom, which
# is exactly how it survived round 16.
# ---------------------------------------------------------------------------
_ENUM_OUT=()
enumerate_git_z() {
  local -a _raw=()
  _ENUM_OUT=()
  # The sentinel is appended INSIDE the same process substitution, after git
  # exits, so it carries git's status across the boundary (a plain `$?` after
  # `mapfile` reports mapfile's status, not git's). `-z` output is
  # NUL-delimited, so the sentinel cannot collide with a filename.
  mapfile -d '' -t _raw < <(
    "$@" 2>/dev/null
    printf 'ENUMRC=%s\0' "$?"
  )
  local _n=${#_raw[@]}
  [ "$_n" -eq 0 ] && return 90
  local _sent="${_raw[$((_n - 1))]}"
  case "$_sent" in
    ENUMRC=*) ;;
    *) return 91 ;;
  esac
  local _rc="${_sent#ENUMRC=}"
  unset "_raw[$((_n - 1))]"
  _ENUM_OUT=( ${_raw[@]+"${_raw[@]}"} )
  return "$_rc"
}

# _enum_fail <what> <rc> — never infer "nothing to scan" from an empty list.
# A failed enumeration and an empty result are indistinguishable at the array,
# and guessing "empty" would print a false "OK: no … found".
_enum_fail() {
  {
    echo "FAIL: $1 enumeration failed (rc=$2) — cannot certify"
    echo "  (rc=90 => no sentinel: the read itself failed; rc=91 => sentinel"
    echo "   malformed: output truncated; any other value => git's own status.)"
  } >&2
  exit 1
}

mode="tree"
files=()
if [ "${1:-}" = "--staged" ]; then
  mode="staged"
elif [ "$#" -gt 0 ]; then
  mode="files"
  files=("$@")
fi

case "$mode" in
  tree)
    cd "$REPO_ROOT" || exit 2
    enumerate_git_z git ls-files -z
    _enum_rc=$?
    [ "$_enum_rc" -ne 0 ] && _enum_fail "tracked-file (git ls-files)" "$_enum_rc"
    for f in ${_ENUM_OUT[@]+"${_ENUM_OUT[@]}"}; do
      [ -n "$f" ] && scan_disk_file "$f"
    done
    ;;
  staged)
    cd "$REPO_ROOT" || exit 2
    # Round-16 fix (2026-08-13): the former newline-delimited
    # `git diff --cached --name-only` read git's C-quoted display form and
    # `git show ":$f"` could not resolve it, so the file was silently
    # skipped — `secret.md` rc=1 / `sécret.md` rc=0, one accented character
    # putting a credential in the commit (§11.4.10 / CONST-042). The
    # pre-commit hook's step 4 shells out to THIS script, so the hook's own
    # `-z` array conversion did NOT cover this consumer: it re-derived the
    # staged list here with the very idiom that was removed there. The `-z`
    # enumeration now lives in enumerate_git_z above, shared with tree mode.
    #
    # NOTE: $f is a RAW filename, so every downstream consumer of $f must be
    # escape-safe — see the `printf '%s'` / real-newline accumulator change
    # (a `printf '%b'` would newly introduce the truncation defect where a
    # filename containing `\c` silently drops every later offender from the
    # diagnostic). Round-20 correction: this said "four sites in the hook",
    # which miscounts. At HEAD the `%b` sites were THREE in the hook
    # (`git show HEAD:scripts/git_hooks/pre-commit | grep -n '%b'` -> 168, 216,
    # 466) plus ONE in this file (HEAD:scripts/secret_scan.sh:310) — four in
    # total, across two files.
    enumerate_git_z git diff --cached --name-only --diff-filter=ACMR -z
    _enum_rc=$?
    [ "$_enum_rc" -ne 0 ] && _enum_fail "staged-file (git diff --cached)" "$_enum_rc"
    for f in ${_ENUM_OUT[@]+"${_ENUM_OUT[@]}"}; do
      [ -n "$f" ] && scan_staged_file "$f"
    done
    ;;
  files)
    for f in "${files[@]}"; do
      scan_disk_file "$f"
    done
    ;;
esac

if [ "$found" -eq 0 ]; then
  echo "OK: no unallowlisted key-shaped secret pattern found (mode=$mode)"
  exit 0
fi

{
  echo ""
  echo "FAIL: key-shaped secret pattern(s) found (value never printed):"
  # `%s`, not `%b`: $hits now carries RAW filenames (the -z change above),
  # and `%b` interprets backslash escapes — a name containing `\c` would
  # TERMINATE OUTPUT and silently drop every later finding. Separators are
  # real newlines (appended as $'\n'), so no escape interpretation is needed.
  printf '%s\n' "$hits"
  echo ""
  echo "If real: rotate immediately, git rm --cached the file, and follow"
  echo "the CONST-042 / §11.4.10 post-mortem procedure before re-committing."
  echo "If a false positive (documentation example): add a redaction marker"
  echo "such as REDACTED / EXAMPLE / ... to the line so the allowlist"
  echo "recognizes it as intentional, non-secret content."
} >&2
exit 1
