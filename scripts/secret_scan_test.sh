#!/usr/bin/env bash
# scripts/secret_scan_test.sh
#
# Hermetic test suite for scripts/secret_scan.sh (§11.4.135 / §11.4.138
# permanent secret-scan guard).
#
# COUNT (kept honest per §11.4.6 — the header previously said "22 … cases"
# while the suite ran more than that, a small self-description defect of
# exactly the kind this file exists to catch elsewhere). MEASURED by running
# it, round 20: 55 assertions in the standing GREEN polarity — 53 distinct
# `Test N` labels, EACH FIRING EXACTLY ONCE, plus 2 unlabelled paired-mutation
# assertions. That difference of 2 is the whole reason the label count and the
# assertion count are not equal.
#
#   bash scripts/secret_scan_test.sh > /tmp/green.txt
#   grep -cE '^(PASS|FAIL)[: ]'     /tmp/green.txt   -> 55  (all assertions)
#   grep -cE '^(PASS|FAIL): Test '  /tmp/green.txt   -> 53  (labelled)
#   grep -cE '^(PASS|FAIL) \(mutation' /tmp/green.txt -> 2  (unlabelled)
#   grep -E '^(PASS|FAIL): Test ' /tmp/green.txt \
#     | sed -E 's/^(PASS|FAIL): (Test [0-9]+[a-z]?):.*/\2/' \
#     | sort -V | uniq -c | awk '$1!=1'              -> empty (none twice)
#
# COUNTING GOTCHA, stated because it caused a wrong count twice: the two
# mutation assertions print "PASS (mutation confirmed load-bearing): …", NOT
# "PASS: …", so the obvious `grep -c '^PASS:'` UNDERCOUNTS by exactly 2 and
# invites "correcting" this header to the wrong number. Round 18 wrote "37
# assertions … from 35 labels … two of them assert twice across a mutate/assert
# cycle"; the last clause was simply false (no label has ever fired twice) and
# it also contradicted its own arithmetic, since 35 + 2 already equals 37.
# Labels are not a contiguous run — 12 and 21 are post-mutation re-checks and
# several carry a `b` suffix — so the highest label number is not the count
# either. Trust the commands above, not this prose.
#
#   Tests 1-7b, 13, 15, 17, 19          12  real-key-shaped fixtures (MUST detect)
#   Tests 8-11, 14, 16, 18, 20           8  allowlisted / near-miss (MUST NOT detect)
#   (2 paired §1.1 mutation assertions)  2  Google + Anthropic patterns load-bearing
#   Tests 12, 21                         2  post-mutation re-checks
#   Tests 22-25 (+22b/23b/24b)           7  `--staged` non-ASCII path (round 16)
#   Tests 26-29 (+26b/27b)               6  TREE-mode non-ASCII path (round 18)
#   Tests 30-33 (+30b/33b)               6  tracked-symlink content (round 20)
#   Tests 34-36 (+35b)                   4  staged gitlinks — over-block guard
#                                           (round 20)
#   Tests 37-40 (+37b)                   5  dir-shadowed tracked files +
#                                           failed-index-lookup (round 20)
#   Tests 41-43                          3  index lookup answers about $f,
#                                           not a neighbour (round 20)
#
# The paired §1.1 mutations prove the guard is genuinely load-bearing (not a
# tautology): (1) neuter the Google-pattern line, plant a real-shaped Google
# key, show the guard WRONGLY passes, then show the unmutated scanner
# correctly fails again; (2) the same cycle for the Anthropic-key pattern.
# Both mutations are applied to a COPY (see MIRROR below), never to the live
# tracked scanner. Never
# echoes a real secret value to stdout in this test's own output beyond
# the deliberately-fake fixture strings written to disk (none of the
# fixture strings are real credentials — see NOTE below).
#
# NOTE: every "real-shaped" fixture below is a FABRICATED placeholder built
# to match the shape (prefix + length) of a real key, never a value that
# could authenticate against any real service. §11.4.10 governs REAL
# secrets; these are synthetic test inputs, exactly like scan-secrets.sh's
# own test suite (scripts/test-scan-secrets.sh) and the .scan-secrets-allow
# entries that document this same convention project-wide.
#
# RUNTIME-ASSEMBLED FIXTURES (2026-07-11 fix, closes a real GitHub
# push-protection block): every "MUST be detected" fixture below (the ones
# our guard is required to catch — Tests 1-7b/13/15/17/19) is assembled at
# RUNTIME from separate prefix/body fragments (`printf '...%s' "$frag"`)
# rather than written as one contiguous literal, so NO contiguous
# secret-shaped token ever appears in this file's SOURCE TEXT — only in the
# ephemeral $WORKDIR temp file the assembled value is written to (never
# committed, never even tracked by git). GitHub's server-side push
# protection scans the DIFF TEXT of what's being pushed; a literal fixture
# with the real Slack-bot-token shape (xoxb- + a digit run + "-" + a digit
# run + "-" + an alphanumeric run — deliberately NOT reproduced verbatim
# even here in prose, per the same fix this comment documents) sitting in
# committed source is indistinguishable to that scanner from a real leaked
# token and gets blocked (this happened on 2026-07-11 for that fixture).
# Splitting the prefix (kept inline in the printf FORMAT string,
# far too short alone to match any pattern) from the long fabricated body
# (held in a shell variable, never adjacent to its defining prefix in
# source) breaks every pattern's required contiguous shape in the SOURCE
# while leaving the value the guard actually tests — the text written to
# the WORKDIR temp file at runtime — byte-identical to before. The
# allowlisted / non-matching fixtures (Tests 8,9,10,11,14,16,18,20) are
# deliberately NOT real-shaped to begin with (that's the point of those
# cases) and are left as plain literals.
#
# §11.4.84-mutation-test-exempt: this file's markers are test logic, not
# residue from an interrupted experiment. The literal string "MUTATED for
# paired" below documents this suite's own mutate -> assert cycle. As of
# round 18 (2026-08-13) that cycle no longer touches the live tracked
# scanner AT ALL: the mutation is applied to a copy under $WORKDIR (see
# MIRROR / mutate_mirror below), so there is no window in which the
# repository's credential scanner sits neutered on disk for a concurrent
# `git add` to stage — see the mirror-based-mutation note there for the
# hazard this replaced and why `trap ... EXIT` was not sufficient cover.

set -uo pipefail
REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

# SCANNER_UNDER_TEST override (round 18): lets the §11.4.115 RED polarity be
# driven against a PRE-FIX COPY of the scanner without mutating the live
# tracked file — the same §11.4.84 hazard the mirror-based mutation below
# removes. Unset (the normal case) it is the live scanner, so the standing
# GREEN suite tests exactly what ships.
#
# TWO DISTINCT PATHS, and conflating them was a real defect (round 21 fix).
#
#   $SCANNER      — the thing UNDER TEST. Follows the override. Every
#                   assertion in this file runs against it; that is correct
#                   and is what the override exists for.
#   $LIVE_SCANNER — the tracked repository file, ALWAYS, override or not.
#                   The §11.4.84 quiescence machinery (pristine sha, the
#                   mutate_mirror live-drift check, and cleanup's assert +
#                   preserve-and-fail — round 22 removed the auto-restore
#                   that used to sit there, see NO-AUTO-RESTORE below) is
#                   pinned to THIS, never to $SCANNER.
#
# MEASURED DEFECT this replaces: before the split, all three §11.4.84
# mechanisms read $SCANNER, so they retargeted with the override. Run with
# SCANNER_UNDER_TEST pointed at a copy, the suite printed
#   SCANNER_RESTORED=n/a (live scanner never mutated; sha stable 00b1206aa2…)
# — the OVERRIDE's sha, while the live scanner's was 1d86d10439f98fb4. In
# exactly the mode used for the paired mutation experiments the guard was
# INERT: it watched the copy, reported success, and would have missed a write
# to the tracked file. Worse, cleanup's drift branch did `cp "$BACKUP"
# "$SCANNER"` — with an override set that "restore" would have written to the
# override, leaving a genuinely drifted live scanner in place while claiming
# a repair. $BACKUP is therefore now a copy of $LIVE_SCANNER (the file the
# drift assertion actually watches), not of the thing under test; a backup of
# one file used to reason about another is how that incoherence arose. Round
# 22 went further and deleted the auto-restore outright: $BACKUP is now only
# preserved for a human to apply MANUALLY, and is never applied by this
# script — see NO-AUTO-RESTORE above cleanup().
LIVE_SCANNER="$REPO_ROOT/scripts/secret_scan.sh"
SCANNER="${SCANNER_UNDER_TEST:-$LIVE_SCANNER}"
WORKDIR=$(mktemp -d)
BACKUP="$WORKDIR/secret_scan.sh.orig-backup"
cp "$LIVE_SCANNER" "$BACKUP"
SCANNER_SHA_PRISTINE=$(sha256sum < "$LIVE_SCANNER" | cut -d' ' -f1)

# ---------------------------------------------------------------------------
# MIRROR-BASED MUTATION (round 18, 2026-08-13) — §11.4.84 working-tree
# quiescence, closing a hazard this suite itself created.
#
# The two paired §1.1 mutations below used to run `sed -i` against the LIVE
# TRACKED SCANNER at $SCANNER, neutering its Google and Anthropic key
# patterns, and relied on `trap cleanup EXIT` to put it back. That opened a
# real window: for the duration of each mutation the repository's credential
# scanner on disk could not match those key classes, and THREE sibling
# streams commit in this checkout. A concurrent `git add`/commit landing in
# that window would stage a neutered credential scanner — precisely the
# §11.4.84 mutation-residue class this work item exists to close, produced by
# its own test suite. `trap ... EXIT` covers normal exit and interrupts but
# NOT `kill -9`, and there was no flock or MUTATION_IN_PROGRESS lock, so a
# hard kill mid-experiment would have left the neutered scanner on disk.
#
# The mutation now targets a COPY in $WORKDIR. That REMOVES the window rather
# than narrowing it: the live scanner is never written to at all, so no
# lock, no restore ordering and no kill-signal coverage is required for
# correctness. $BACKUP is kept only as material for a MANUAL, human-authorised
# repair (round 22 — see NO-AUTO-RESTORE below), and cleanup now ASSERTS the
# live file was never touched instead of assuming it.
# ---------------------------------------------------------------------------
MIRROR="$WORKDIR/secret_scan.mirror.sh"
MIRROR_TAINT=0

mutate_mirror() {
  # mutate_mirror <sed-expression> — copy the scanner UNDER TEST ($SCANNER) to
  # $MIRROR and apply the expression to the COPY. Asserts the mutation
  # genuinely landed (exactly ONE substitution; mirror sha differs from
  # pristine; mirror still parses) and that the TRACKED file ($LIVE_SCANNER,
  # never $SCANNER — see the LIVE_SCANNER note above) is byte-identical
  # afterwards. Without those assertions a silently-failed `sed` would leave
  # the paired test comparing an unmutated scanner against itself and
  # reporting a result.
  #
  # The mirror is copied from $SCANNER (the override follows, as it must — the
  # RED polarity mutates the pre-fix scanner) while the drift assertion reads
  # $LIVE_SCANNER (the override must NOT follow — the invariant being asserted
  # is about the repository's tracked credential scanner). Those are two
  # different files whenever an override is set, and reading $SCANNER for both
  # is what made this check inert under exactly the mode that uses it.
  local expr="$1"
  local token='ZZZ_THIS_PATTERN_CAN_NEVER_MATCH_ANYTHING_ZZZ'
  cp "$SCANNER" "$MIRROR"
  chmod +x "$MIRROR"
  local pristine_sha mirror_sha pre post live_sha
  pristine_sha=$(sha256sum < "$MIRROR" | cut -d' ' -f1)
  pre=$(grep -cF "$token" "$MIRROR" || true)
  sed -i.bak "$expr" "$MIRROR"
  rm -f "${MIRROR}.bak"
  post=$(grep -cF "$token" "$MIRROR" || true)
  mirror_sha=$(sha256sum < "$MIRROR" | cut -d' ' -f1)
  live_sha=$(sha256sum < "$LIVE_SCANNER" | cut -d' ' -f1)
  local ok=1
  [ "$pre" -eq 0 ] || { echo "FAIL(mutation-setup): neutering token already present before sed (count=$pre)"; ok=0; }
  [ "$((post - pre))" -eq 1 ] || { echo "FAIL(mutation-setup): expected exactly 1 substitution, got $((post - pre))"; ok=0; }
  [ "$mirror_sha" != "$pristine_sha" ] || { echo "FAIL(mutation-setup): mirror sha unchanged — sed did not apply"; ok=0; }
  bash -n "$MIRROR" 2>/dev/null || { echo "FAIL(mutation-setup): mutated mirror does not parse"; ok=0; }
  if [ "$live_sha" != "$SCANNER_SHA_PRISTINE" ]; then
    echo "FAIL(§11.4.84): the LIVE scanner changed during a mirror mutation (sha $live_sha != $SCANNER_SHA_PRISTINE)"
    MIRROR_TAINT=1
    ok=0
  fi
  if [ "$ok" -ne 1 ]; then
    FAIL=$((FAIL + 1))
    return 1
  fi
  return 0
}

# ---------------------------------------------------------------------------
# NO-AUTO-RESTORE ON DRIFT (round 22 / F8, 2026-08-13) — §9.2 absolute data
# safety, §11.4.174 never act on state you have not proven is yours, §11.4.122
# no silent removal of someone else's work.
#
# cleanup() used to react to drift by doing `cp "$BACKUP" "$LIVE_SCANNER"` —
# auto-restoring the TRACKED scanner — and then `rm -rf "$WORKDIR"`, which
# destroyed $BACKUP itself. Since round 18 that reaction is ALWAYS WRONG, and
# wrong BY CONSTRUCTION rather than by bad luck:
#
#   the paired §1.1 mutations are MIRROR-BASED (see above) — they copy the
#   scanner into $WORKDIR and mutate the COPY. This suite therefore CANNOT
#   write to the live scanner. So any drift cleanup observes is, necessarily,
#   SOMEBODY ELSE'S write — and THREE sibling streams commit in this checkout.
#
# The auto-restore could thus only ever do one thing: silently revert a
# sibling stream's legitimate in-flight edit. It was loud in THIS suite's
# stdout, but the person whose work was reverted is not reading this suite's
# stdout — they just watch their edit vanish, unexplained, courtesy of a
# script they never ran. That is the §9.2 data-loss class, self-inflicted by
# a safety check.
#
# MEASURED, NOT HYPOTHETICAL — two independent measurements:
#
#   (1) During the round-21 run the live scanner legitimately changed from
#       1d86d10439f98fb4 to fa3cbd5cb4d2a6be — a sibling stream's edit,
#       mtime-ordered proof. It happened not to trip the restore, but the
#       path was armed the whole time.
#   (2) Reproduced end-to-end pre-fix against a STAND-IN scanner in a scratch
#       git repo (never the live tree: the suite takes its root from
#       `git rev-parse --show-toplevel`, so a scratch root retargets
#       $LIVE_SCANNER at the stand-in). A wrapper drifted the stand-in
#       mid-run at t=3.4s — i.e. AFTER the last mutate_mirror call:
#           DRIFTED_sha=5dfeff0ae51e5d6f   ("sibling's" edit landed)
#           AFTER_sha=db1b83e78115d279     (edit GONE — reverted)
#           AFTER_marker_count=0           (the added marker line destroyed)
#           Results: 55 passed, 0 failed
#           SUITE_RC=0
#       The suite destroyed a third party's edit AND reported success. A
#       drift at t=1.5s instead landed inside a mutation window and was
#       caught by mutate_mirror's own live-drift check (MIRROR_TAINT=1,
#       rc=1) — so pre-fix the run failed only when the foreign write
#       happened to collide with a mutation; the late case had no guard at
#       all and exited 0.
#
# The reaction is therefore now: assert, DO NOT WRITE, preserve, fail.
#   - never `cp` over $LIVE_SCANNER — nothing in this file restores it, ever;
#   - report loudly on stderr with expected vs actual sha, as before;
#   - copy $BACKUP OUT of $WORKDIR before the `rm -rf` below destroys it, to
#     a timestamped path under the system temp dir, and print the EXACT
#     one-line command a human can run to apply it MANUALLY;
#   - say plainly that the drift is far more likely a sibling's edit than
#     this run's residue, so nobody "restores" over live work reflexively;
#   - set MIRROR_TAINT so the run FAILS, reusing the existing taint mechanism
#     at the tail of this file. Because cleanup runs from the EXIT trap — i.e.
#     AFTER that final `[ "$FAIL" -eq 0 ] && [ "$MIRROR_TAINT" -eq 0 ]` line
#     has already fixed the status — setting the flag alone cannot change the
#     exit code, so the trap re-checks it and `exit 1`s. A clean run falls
#     through and keeps whatever status the suite already produced (verified
#     empirically: taint -> 1; clean + true -> 0; clean + false -> 1).
#
# The `cp "$BACKUP" ...` retained below is also what keeps this file eligible
# for the pre-commit §11.4.84 mutation-test exemption, which requires a
# `cp <space> ... backup` (or `git checkout --`) INSIDE the EXIT trap's target
# function. That condition is still met — the copy now goes to a preservation
# path instead of over the tracked file.
# ---------------------------------------------------------------------------
cleanup() {
  # §11.4.84 working-tree quiescence. The live scanner is never mutated (see
  # MIRROR above), so this ASSERTS that invariant rather than relying on a
  # restore — and, per NO-AUTO-RESTORE above, reacts to a violation by
  # preserving evidence and failing, NEVER by writing to the tracked file.
  #
  # Reads $LIVE_SCANNER, NEVER $SCANNER — under an override those are
  # different files, and watching $SCANNER made this assertion report success
  # about a temp-dir copy while the tracked file went unwatched (see the
  # MEASURED DEFECT under LIVE_SCANNER above). The sha printed below is
  # therefore always the tracked scanner's.
  local live_sha preserved
  live_sha=$(sha256sum < "$LIVE_SCANNER" 2>/dev/null | cut -d' ' -f1)
  if [ "$live_sha" != "$SCANNER_SHA_PRISTINE" ]; then
    echo "ERROR(§11.4.84): live scanner drifted during this run — deliberately NOT restored." >&2
    echo "  live scanner: $LIVE_SCANNER" >&2
    echo "  expected sha: $SCANNER_SHA_PRISTINE" >&2
    echo "  actual   sha: $live_sha" >&2
    echo "  This suite mutates a COPY only (see MIRROR above) and never writes to the" >&2
    echo "  live scanner, so this drift is MUCH more likely a concurrent sibling" >&2
    echo "  stream's legitimate in-flight edit than residue from this run." >&2
    echo "  Auto-restoring would silently revert that stream's work, so this run does" >&2
    echo "  NOT touch the file (§9.2 data safety, §11.4.174 prove-it-is-yours," >&2
    echo "  §11.4.122 no silent removal). PROVE whose write it is before acting:" >&2
    echo "  git status/diff on the path above, and ask the other streams." >&2
    preserved="${TMPDIR:-/tmp}/secret_scan.sh.pre-run-copy.$(date -u +%Y%m%dT%H%M%SZ).$$"
    if cp "$BACKUP" "$preserved" 2>/dev/null; then
      echo "  This run's PRE-RUN copy of the live scanner is preserved (outside \$WORKDIR) at:" >&2
      echo "    $preserved" >&2
      echo "  ONLY if you have PROVEN the drift is this run's residue, apply it manually:" >&2
      printf '    cp %q %q\n' "$preserved" "$LIVE_SCANNER" >&2
    else
      echo "  WARNING: could not preserve this run's pre-run copy at $preserved —" >&2
      echo "  it dies with \$WORKDIR. Read the committed version instead:" >&2
      printf '    git show HEAD:%s\n' 'scripts/secret_scan.sh' >&2
    fi
    MIRROR_TAINT=1
    echo "SCANNER_RESTORED=no (drift detected — NOT auto-restored; see stderr)"
  else
    echo "SCANNER_RESTORED=n/a (live scanner never mutated; sha stable ${SCANNER_SHA_PRISTINE:0:16})"
  fi
  rm -rf "$WORKDIR"
  # Reuse the existing taint mechanism (tail of this file) rather than adding
  # a second one. See NO-AUTO-RESTORE above for why the re-check + explicit
  # exit is required here and why a clean run must fall through untouched.
  [ "$MIRROR_TAINT" -eq 0 ] || exit 1
}
trap cleanup EXIT

PASS=0
FAIL=0

check() {
  # check <description> <expected-rc-class: zero|nonzero> <file>
  local desc="$1" expect="$2" file="$3"
  local out rc
  out=$("$SCANNER" "$file" 2>&1)
  rc=$?
  case "$expect" in
    zero)
      if [ "$rc" -eq 0 ]; then
        echo "PASS: $desc (exit 0 as expected)"
        PASS=$((PASS + 1))
      else
        echo "FAIL: $desc (expected exit 0, got $rc)"
        printf '%s\n' "$out" | sed 's/^/    /'
        FAIL=$((FAIL + 1))
      fi
      ;;
    nonzero)
      if [ "$rc" -ne 0 ]; then
        echo "PASS: $desc (non-zero exit as expected)"
        PASS=$((PASS + 1))
      else
        echo "FAIL: $desc (expected non-zero exit, got 0)"
        printf '%s\n' "$out" | sed 's/^/    /'
        FAIL=$((FAIL + 1))
      fi
      ;;
  esac
}

# ---------------------------------------------------------------------------
# Real-key-shaped fixtures (each a fabricated placeholder, never a real
# credential — see file-level NOTE) → MUST be detected (non-zero exit).
# ---------------------------------------------------------------------------

frag="SyD-fabricated0123456789abcdef"
printf 'GEMINI_API_KEY=AIza%s\n' "$frag" > "$WORKDIR/1_google.txt"
check "Test 1: real-shaped Google API key" nonzero "$WORKDIR/1_google.txt"

frag="fabricated0123456789ABCDEFGHIJKLMN"
printf 'OPENAI_API_KEY=sk-%s\n' "$frag" > "$WORKDIR/2_openai.txt"
check "Test 2: real-shaped OpenAI API key" nonzero "$WORKDIR/2_openai.txt"

frag="ABCDEFGHIJKLMNOP"
printf 'AWS_ACCESS_KEY_ID=AKIA%s\n' "$frag" > "$WORKDIR/3_aws.txt"
check "Test 3: real-shaped AWS access key" nonzero "$WORKDIR/3_aws.txt"

frag="fabricated0123456789ABCDEFGHIJKLMNOPQR"
printf 'GITHUB_TOKEN=ghp_%s\n' "$frag" > "$WORKDIR/4_github.txt"
check "Test 4: real-shaped GitHub PAT" nonzero "$WORKDIR/4_github.txt"

# PEM headers: the leading "-----" run is kept in a separate variable from
# the "BEGIN ... PRIVATE KEY"/"END ... PRIVATE KEY" words so the full
# 5-dash + BEGIN/END + key-type + "PRIVATE KEY" + 5-dash marker the guard
# (and GitHub's own private-key scanner) looks for is never contiguous in
# this file's source — not even spelled out verbatim in this comment.
dash="-----"
frag="MIIfabricatedNotARealKeyBodyAtAll"
printf '%sBEGIN RSA PRIVATE KEY%s\n%s\n%sEND RSA PRIVATE KEY%s\n' \
  "$dash" "$dash" "$frag" "$dash" "$dash" > "$WORKDIR/5_pem.txt"
check "Test 5: private-key header (RSA)" nonzero "$WORKDIR/5_pem.txt"

dash="-----"
frag="MIIfabricatedGenericPKCS8Body"
printf '%sBEGIN PRIVATE KEY%s\n%s\n%sEND PRIVATE KEY%s\n' \
  "$dash" "$dash" "$frag" "$dash" "$dash" > "$WORKDIR/6_pem_generic.txt"
check "Test 6: private-key header (generic PKCS8)" nonzero "$WORKDIR/6_pem_generic.txt"

printf 'SLACK_BOT_TOKEN=xoxb-%s-%s-%s\n' \
  "123456789012" "123456789012" "abcdefghijklmnopqrstuvwx" > "$WORKDIR/7_slack_real.txt"
check "Test 7: real-shaped Slack bot token (xoxb-N-N-alnum)" nonzero "$WORKDIR/7_slack_real.txt"

# HuggingFace hf_ token — the exact class that partially leaked into the
# session transcript on 2026-07-11 (catalogue-providers stream); added to
# the guard patterns per §11.4.138 (close the class that escaped).
frag="fabricated0123456789ABCDEFGHIJKLMNOP"
printf 'HF_TOKEN=hf_%s\n' "$frag" > "$WORKDIR/7b_hf.txt"
check "Test 7b: real-shaped HuggingFace token (hf_...)" nonzero "$WORKDIR/7b_hf.txt"

# ---------------------------------------------------------------------------
# Defense-in-depth hardening pass (§11.4.138, 2026-07-11 follow-up): more
# high-value key classes real .env files in this project could hold (the
# project ships ~30 provider .env aliases including xai, gcp, azure).
# Thresholds were tuned against the whole tracked tree (see
# scratchpad/r41_guard_hardening.md) so they clear every existing
# legitimate doc placeholder / Go unit-test fixture already committed.
# ---------------------------------------------------------------------------

frag="fabricated0123456789ABCDEFGHIJKLMNOP"
printf 'XAI_API_KEY=xai-%s\n' "$frag" > "$WORKDIR/13_xai.txt"
check "Test 13: real-shaped xAI API key (xai-...)" nonzero "$WORKDIR/13_xai.txt"

# "sk-ant-api03-" (13 chars total after "sk-ant-": "api03-" is only 6 chars,
# far short of the guard's 30-char threshold) stays a literal prefix in the
# printf FORMAT string below — only the long fabricated body is a variable,
# so "sk-ant-[A-Za-z0-9_-]{30,}" never matches this file's source text.
frag="fabricatedNotARealKey0123456789ABCDEFGHIJ"
printf 'ANTHROPIC_API_KEY=sk-ant-api03-%s\n' "$frag" > "$WORKDIR/15_anthropic.txt"
check "Test 15: real-shaped Anthropic API key (sk-ant-api03-...)" nonzero "$WORKDIR/15_anthropic.txt"

# Fabricated GCP service-account JSON credential blob (fake project id,
# fake hex-looking private_key_id, no PEM body at all — this fixture is
# deliberately narrower than a real credentials file so it isolates the
# two new JSON-marker patterns from the pre-existing PEM pattern). The
# guard's two GCP patterns each require a JSON key name IMMEDIATELY
# followed (mod whitespace) by its colon — so each key name and its
# ": value" suffix are assembled from two separate string literals that are
# never adjacent in source, only in the printf-assembled output line.
gcp_frag="fabricated0123456789abcdef01234567890123"
gcp_type_suffix=': "service_account"'
gcp_pkid_suffix=": \"$gcp_frag\""
{
  printf '{\n'
  printf '  "type"%s,\n' "$gcp_type_suffix"
  printf '  "project_id": "fabricated-test-project",\n'
  printf '  "private_key_id"%s,\n' "$gcp_pkid_suffix"
  printf '  "client_email": "fabricated@fabricated-test-project.iam.gserviceaccount.com"\n'
  printf '}\n'
} > "$WORKDIR/17_gcp.txt"
check "Test 17: fabricated GCP service-account JSON blob (type + private_key_id)" nonzero "$WORKDIR/17_gcp.txt"

frag="1a2b3c4d5e6f70819203a4b5c6d7e8f9"
printf 'AZURE_OPENAI_API_KEY=%s\n' "$frag" > "$WORKDIR/19_azure.txt"
check "Test 19: real-shaped Azure key (AZURE_*KEY=<32-hex>)" nonzero "$WORKDIR/19_azure.txt"

# ---------------------------------------------------------------------------
# Allowlisted / non-matching fixtures → MUST NOT be detected (exit 0).
# ---------------------------------------------------------------------------

echo 'GEMINI_API_KEY=<REDACTED-GEMINI-KEY-CONST-042-...>' > "$WORKDIR/8_redacted.txt"
check "Test 8: redaction marker (<REDACTED-...>) is allowlisted" zero "$WORKDIR/8_redacted.txt"

echo 'sample key shape: AIzaSyEXAMPLE1234567890abcdefghi' > "$WORKDIR/9_example.txt"
check "Test 9: EXAMPLE placeholder is allowlisted" zero "$WORKDIR/9_example.txt"

# This is the exact false-positive class the post-mortem flagged: a doc
# mentioning "xoxb-" prose/pattern without the real N-N-alnum token shape
# (the OLD scan-secrets.sh regex xoxb-[A-Za-z0-9-]{16,} would match this;
# the tightened xoxb-[0-9]{9,}-[0-9]{9,}-[A-Za-z0-9]{20,} correctly does not).
echo 'Slack bot tokens look like xoxb-this-is-not-a-real-token-shape-example' > "$WORKDIR/10_slack_false_positive.txt"
check "Test 10: xoxb- prose mention (not real token shape) is NOT flagged" zero "$WORKDIR/10_slack_false_positive.txt"

echo 'nothing sensitive here, just prose' > "$WORKDIR/11_clean.txt"
check "Test 11: clean file" zero "$WORKDIR/11_clean.txt"

# --- Clean-prose / near-miss fixtures for the 2026-07-11 hardening pass ---
# Each of these is deliberately a NEAR MISS (mentions the same prefix/word
# as its matching pattern) rather than generic unrelated prose, so a PASS
# here is real evidence the pattern is shape-specific and not over-broad
# (§11.4.6) — not just "the pattern doesn't match totally unrelated text".

echo 'Configure XAI_API_KEY for Grok; the prefix is xai- for xAI keys.' > "$WORKDIR/14_xai_prose.txt"
check "Test 14: xai- prose mention (not real token shape) is NOT flagged" zero "$WORKDIR/14_xai_prose.txt"

echo 'export ANTHROPIC_API_KEY="sk-ant-your-key"' > "$WORKDIR/16_anthropic_placeholder.txt"
check "Test 16: sk-ant-your-key doc placeholder (too short) is NOT flagged" zero "$WORKDIR/16_anthropic_placeholder.txt"

echo 'GCP uses a service_account JSON credentials file with a type field.' > "$WORKDIR/18_gcp_prose.txt"
check "Test 18: service_account prose mention (no JSON key:value shape) is NOT flagged" zero "$WORKDIR/18_gcp_prose.txt"

echo 'AZURE_OPENAI_API_KEY=mock-azure-key-for-testing' > "$WORKDIR/20_azure_placeholder.txt"
check "Test 20: AZURE_*_KEY=mock-... placeholder (non-hex value) is NOT flagged" zero "$WORKDIR/20_azure_placeholder.txt"

# ---------------------------------------------------------------------------
# Paired §1.1 mutation: neuter the Google-pattern line in secret_scan.sh,
# re-run Test 1's fixture, and assert it now WRONGLY passes (proving Test 1
# is load-bearing — it genuinely depends on that pattern line, not a
# tautology). Then restore and assert Test 1 passes again.
# ---------------------------------------------------------------------------

echo ""
echo "--- Paired mutation: neuter Google-key pattern, expect Test 1 fixture to WRONGLY pass ---"

# Replace the Google pattern's regex with one that can never match anything.
# The substitution is applied to a MIRROR in $WORKDIR (see mutate_mirror
# above), never to the live tracked scanner — "MUTATED for paired §1.1
# mutation test" describes a copy in a temp dir that no `git add` can reach.
mutate_mirror 's#Google API key|AIza\[0-9A-Za-z_-\]{20,}#Google API key|ZZZ_THIS_PATTERN_CAN_NEVER_MATCH_ANYTHING_ZZZ#'
mutation_out=$("$MIRROR" "$WORKDIR/1_google.txt" 2>&1)
mutation_rc=$?
if [ "$mutation_rc" -eq 0 ]; then
  echo "PASS (mutation confirmed load-bearing): with the Google pattern neutered, the real-shaped Google key fixture WRONGLY passed (exit 0) — Test 1 genuinely depends on that pattern line."
  PASS=$((PASS + 1))
else
  echo "FAIL (mutation did NOT flip the result): neutering the Google pattern still produced exit $mutation_rc — Test 1 may not be exercising the pattern this mutation targets, or another pattern independently caught it."
  printf '%s\n' "$mutation_out" | sed 's/^/    /'
  FAIL=$((FAIL + 1))
fi

# No restore step is needed: the mutation was applied to $MIRROR, so the live
# scanner Test 12 re-runs below was never modified in the first place.
rm -f "$MIRROR"

echo "--- Paired mutation: live scanner (never mutated), expect Test 1 fixture to correctly fail again ---"
check "Test 12: post-restore, real-shaped Google key correctly detected again" nonzero "$WORKDIR/1_google.txt"

# ---------------------------------------------------------------------------
# Second paired §1.1 mutation (2026-07-11 hardening pass): neuter the new
# "Anthropic API key (explicit)" pattern line, re-run Test 15's fixture,
# and assert it now WRONGLY passes — proving Test 15 genuinely depends on
# that new pattern line (not merely on the pre-existing generic OpenAI
# sk- pattern, which was independently confirmed NOT to match sk-ant-...
# shapes during this hardening pass — see scratchpad/r41_guard_hardening.md).
# Then restore and assert Test 15 passes again.
# ---------------------------------------------------------------------------

echo ""
echo "--- Paired mutation: neuter Anthropic-key pattern, expect Test 15 fixture to WRONGLY pass ---"

# Applied to a MIRROR in $WORKDIR, never to the live tracked scanner
# (MUTATED for paired §1.1 mutation test — see mutate_mirror above).
mutate_mirror 's#Anthropic API key (explicit)|sk-ant-\[A-Za-z0-9_-\]{30,}#Anthropic API key (explicit)|ZZZ_THIS_PATTERN_CAN_NEVER_MATCH_ANYTHING_ZZZ#'
mutation2_out=$("$MIRROR" "$WORKDIR/15_anthropic.txt" 2>&1)
mutation2_rc=$?
if [ "$mutation2_rc" -eq 0 ]; then
  echo "PASS (mutation confirmed load-bearing): with the Anthropic pattern neutered, the real-shaped Anthropic key fixture WRONGLY passed (exit 0) — Test 15 genuinely depends on that pattern line."
  PASS=$((PASS + 1))
else
  echo "FAIL (mutation did NOT flip the result): neutering the Anthropic pattern still produced exit $mutation2_rc — Test 15 may not be exercising the pattern this mutation targets, or another pattern independently caught it."
  printf '%s\n' "$mutation2_out" | sed 's/^/    /'
  FAIL=$((FAIL + 1))
fi

# No restore step needed — the live scanner was never mutated (mirror-based).
rm -f "$MIRROR"

echo "--- Paired mutation: live scanner (never mutated), expect Test 15 fixture to correctly fail again ---"
check "Test 21: post-restore, real-shaped Anthropic key correctly detected again" nonzero "$WORKDIR/15_anthropic.txt"

# ---------------------------------------------------------------------------
# Tests 22-24 (round 16, 2026-08-13) — `--staged` mode over a NON-ASCII path.
#
# §11.4.115 polarity switch: RED_MODE=1 reproduces the defect on the PRE-FIX
# scanner (asserts the leak is PRESENT); RED_MODE=0 (default) is the standing
# GREEN regression guard (asserts it is ABSENT).
#
# THE DEFECT this guards (measured 2026-08-13, pre-fix scanner):
#   the `--staged` branch enumerated the index with a newline-delimited
#   `git diff --cached --name-only`, which yields git's C-QUOTED display form
#   for any non-ASCII path — the literal string `"s\303\251cret.md"`. The
#   subsequent `git show ":$f"` could not resolve that string and the former
#   `|| return 0` reported the file CLEAN. Filename the only variable:
#       CONTROL  secret.md   rc=1  FAIL reported   not landed
#       ATTACK   sécret.md   rc=0  "OK: no ... found"  COMMITTED
#   i.e. one accented character put a credential in a commit (§11.4.10 /
#   CONST-042). These tests drive the SCANNER DIRECTLY — the pre-commit hook
#   shells out to it (step 4), so a hook-level test alone does NOT cover this
#   consumer, which is exactly how the gap survived the earlier array fix.
#
# The fixture key is assembled from a fragment (same convention as Test 1) so
# THIS FILE's own source text never contains a complete key-shaped literal.
# ---------------------------------------------------------------------------
echo ""
echo "--- Staged-mode non-ASCII path (round 16; RED_MODE=${RED_MODE:-0}) ---"

# ---------------------------------------------------------------------------
# ONE-SIDEDNESS FIX (round 18, 2026-08-13). Tests 22-24 previously asserted
# ONLY `rc != 0`. Measured with a vacuity oracle (the scanner forced, in a
# mirror, to a fixed exit status):
#
#     forced scanner   suite rc   counts            tests 22/23/24
#     exit 0 always        1      10 pass/17 fail   all three FAIL
#     exit 1 always        1      17 pass/10 fail   all three PASS
#
# i.e. all three passed against a scanner that refuses EVERYTHING and detects
# nothing. Ten other tests in this suite do catch an always-block scanner, so
# the suite as a whole was not vacuous — but these three carried no
# fail-closed companion of their own. Two things are added:
#   (a) CONTENT assertions — the finding must NAME the expected file and the
#       expected pattern label, not merely be non-zero; and
#   (b) a CLEAN-file companion per mode (Tests 25/28) that must exit 0 on the
#       SAME non-ASCII path, which an always-block scanner cannot satisfy.
# ---------------------------------------------------------------------------
SS_RC=""
SS_OUT=""
staged_scan() {
  # staged_scan <filename> [clean] -> sets SS_RC / SS_OUT for `--staged` mode
  local fname="$1" content_mode="${2:-key}"
  local repo; repo=$(mktemp -d)
  (
    cd "$repo" || exit 99
    git init -q . >/dev/null 2>&1
    git config user.email t@t.t; git config user.name t
    if [ "$content_mode" = "clean" ]; then
      printf 'ordinary prose, nothing sensitive on this line\n' > "$fname"
    else
      frag="SyD-fabricated0123456789abcdef"
      printf 'GEMINI_API_KEY=AIza%s\n' "$frag" > "$fname"
    fi
    git add -- "$fname" >/dev/null 2>&1
    out=$("$SCANNER" --staged 2>&1)
    echo $? > "$repo/.rc"
    printf '%s' "$out" > "$repo/.out"
  ) >/dev/null 2>&1
  SS_RC=$(cat "$repo/.rc" 2>/dev/null || echo 99)
  SS_OUT=$(cat "$repo/.out" 2>/dev/null || echo "")
  rm -rf "$repo"
}

tree_scan() {
  # tree_scan <filename> [clean] -> sets SS_RC / SS_OUT for TREE mode (no args)
  local fname="$1" content_mode="${2:-key}"
  local repo; repo=$(mktemp -d)
  (
    cd "$repo" || exit 99
    git init -q . >/dev/null 2>&1
    git config user.email t@t.t; git config user.name t
    if [ "$content_mode" = "clean" ]; then
      printf 'ordinary prose, nothing sensitive on this line\n' > "$fname"
    else
      frag="SyD-fabricated0123456789abcdef"
      printf 'GEMINI_API_KEY=AIza%s\n' "$frag" > "$fname"
    fi
    git add -- "$fname" >/dev/null 2>&1
    out=$("$SCANNER" 2>&1)
    echo $? > "$repo/.rc"
    printf '%s' "$out" > "$repo/.out"
  ) >/dev/null 2>&1
  SS_RC=$(cat "$repo/.rc" 2>/dev/null || echo 99)
  SS_OUT=$(cat "$repo/.out" 2>/dev/null || echo "")
  rm -rf "$repo"
}

# tree_scan_symlink <linkname> <kind> — like tree_scan, but the throwaway
# repo's tracked entry is a SYMLINK (git mode 120000). Kinds:
#   dir-key       link -> a DIRECTORY whose NAME carries the key shape, so the
#                 key lives in the LINK'S OWN content (the target path string,
#                 which is what git stores in a 120000 blob). `[ -d ]` FOLLOWS
#                 the link, so the pre-r20 scanner judged this entry by the
#                 directory it pointed AT and skipped it at the gitlink branch.
#   dir-clean     link -> an ordinary directory (the shape this repo actually
#                 has: skills/media-validator -> ../constitution/skills/…).
#   outside-key   link -> an ABSOLUTE path to a file OUTSIDE the repo holding a
#                 key. Nothing here is repo content; reading it at all is the
#                 repo-escape read `[ -f ]`'s dereference makes possible.
#   tracked-key   a TRACKED regular file holding a key, plus a tracked symlink
#                 to it — proves in-repo content is still caught (under its own
#                 canonical path) once symlinks stop being dereferenced.
tree_scan_symlink() {
  local link="$1" kind="$2"
  local repo outside
  repo=$(mktemp -d)
  outside=$(mktemp -d)
  (
    cd "$repo" || exit 99
    git init -q . >/dev/null 2>&1
    git config user.email t@t.t; git config user.name t
    frag="SyD-fabricated0123456789abcdef"
    case "$kind" in
      dir-key)
        printf -v tgt 'AIza%s' "$frag"; mkdir -p -- "$tgt"; ln -s -- "$tgt" "$link"
        git add -- "$link" >/dev/null 2>&1 ;;
      dir-clean)
        tgt="ordinary_target_dir"; mkdir -p -- "$tgt"; ln -s -- "$tgt" "$link"
        git add -- "$link" >/dev/null 2>&1 ;;
      outside-key)
        printf 'GEMINI_API_KEY=AIza%s\n' "$frag" > "$outside/outside_secret.txt"
        ln -s -- "$outside/outside_secret.txt" "$link"
        git add -- "$link" >/dev/null 2>&1 ;;
      tracked-key)
        printf 'GEMINI_API_KEY=AIza%s\n' "$frag" > "canonical.md"
        ln -s -- "canonical.md" "$link"
        git add -- "canonical.md" "$link" >/dev/null 2>&1 ;;
      *) exit 98 ;;
    esac
    out=$("$SCANNER" 2>&1)
    echo $? > "$repo/.rc"
    printf '%s' "$out" > "$repo/.out"
  ) >/dev/null 2>&1
  SS_RC=$(cat "$repo/.rc" 2>/dev/null || echo 99)
  SS_OUT=$(cat "$repo/.out" 2>/dev/null || echo "")
  rm -rf "$repo" "$outside"
}

assert_named_finding() {
  # assert_named_finding <test-label> <expected "file:line"> — the finding must
  # NAME the offending path AND the pattern label. Guards against a scanner
  # that merely exits non-zero (see the vacuity oracle above).
  local label="$1" needle="$2"
  if printf '%s' "$SS_OUT" | grep -qF -- "$needle" && \
     printf '%s' "$SS_OUT" | grep -qF -- '(Google API key)'; then
    echo "PASS: $label (finding names \"$needle\" and the Google API key pattern)"
    PASS=$((PASS + 1))
  else
    echo "FAIL: $label (exit status was non-zero but the finding does NOT name \"$needle\" + '(Google API key)')"
    printf '%s\n' "$SS_OUT" | sed 's/^/    /'
    FAIL=$((FAIL + 1))
  fi
}

# Test 22 — CONTROL (plain ASCII). Proves the instrument can detect a
# KNOWN-PRESENT key in --staged mode, so a null result on Test 23 cannot be
# the harness's own blind spot. Expected non-zero in BOTH polarities.
staged_scan "secret.md"
if [ "$SS_RC" -ne 0 ]; then
  echo "PASS: Test 22: --staged CONTROL, ASCII path, key detected (rc=$SS_RC)"
  PASS=$((PASS + 1))
  assert_named_finding "Test 22b: --staged CONTROL finding is specific" "secret.md:1"
else
  echo "FAIL: Test 22: --staged CONTROL, ASCII path — key MISSED (rc=0); harness invalid"
  FAIL=$((FAIL + 1))
  echo "FAIL: Test 22b: --staged CONTROL finding is specific (no finding to inspect)"
  FAIL=$((FAIL + 1))
fi

# Test 23 — ATTACK (non-ASCII path), the polarity-switched assertion.
staged_scan "sécret.md"
if [ "${RED_MODE:-0}" = "1" ]; then
  if [ "$SS_RC" -eq 0 ]; then
    echo "PASS: Test 23 [RED]: pre-fix scanner LEAKS non-ASCII staged path (rc=0 — defect reproduced)"
    PASS=$((PASS + 1))
  else
    echo "FAIL: Test 23 [RED]: expected the pre-fix leak (rc=0), got rc=$SS_RC — defect NOT reproduced"
    FAIL=$((FAIL + 1))
  fi
else
  if [ "$SS_RC" -ne 0 ]; then
    echo "PASS: Test 23: --staged non-ASCII path, key detected (rc=$SS_RC)"
    PASS=$((PASS + 1))
    assert_named_finding "Test 23b: --staged non-ASCII finding is specific" "sécret.md:1"
  else
    echo "FAIL: Test 23: --staged non-ASCII path LEAKED (rc=0) — §11.4.10 credential leak"
    FAIL=$((FAIL + 1))
    echo "FAIL: Test 23b: --staged non-ASCII finding is specific (no finding to inspect)"
    FAIL=$((FAIL + 1))
  fi
fi

# Test 24 — the scanner must never report a false "clean" when it cannot
# enumerate: an empty file list must not be inferred from a FAILED `git diff`.
shimmed_enum_scan() {
  # shimmed_enum_scan <shim-body-marker> <scanner-args...> — run the scanner
  # with a `git` shim on PATH that fails the named enumeration, capturing BOTH
  # rc and output into SS_RC / SS_OUT.
  local marker="$1"; shift
  local repo shim realgit
  repo=$(mktemp -d); shim=$(mktemp -d)
  realgit=$(command -v git)
  if [ "$marker" = "diff-cached" ]; then
    printf '#!/usr/bin/env bash\nfor a in "$@"; do case "$a" in --diff-filter=ACMR)\n for b in "$@"; do [ "$b" = "--cached" ] && exit 128; done ;; esac; done\nexec %s "$@"\n' "$realgit" > "$shim/git"
  else
    printf '#!/usr/bin/env bash\n[ "${1:-}" = "ls-files" ] && exit 128\nexec %s "$@"\n' "$realgit" > "$shim/git"
  fi
  chmod +x "$shim/git"
  (
    cd "$repo" || exit 99
    "$realgit" init -q . >/dev/null 2>&1
    "$realgit" config user.email t@t.t; "$realgit" config user.name t
    echo x > a.txt; "$realgit" add a.txt >/dev/null 2>&1
    out=$(PATH="$shim:$PATH" "$SCANNER" "$@" 2>&1)
    echo $? > "$repo/.rc"
    printf '%s' "$out" > "$repo/.out"
  ) >/dev/null 2>&1
  SS_RC=$(cat "$repo/.rc" 2>/dev/null || echo 99)
  SS_OUT=$(cat "$repo/.out" 2>/dev/null || echo "")
  rm -rf "$repo" "$shim"
}

shimmed_enum_scan diff-cached --staged
if [ "${RED_MODE:-0}" = "1" ]; then
  if [ "$SS_RC" -eq 0 ]; then
    echo "PASS: Test 24 [RED]: pre-fix scanner reports clean on a FAILED enumeration (rc=0)"
    PASS=$((PASS + 1))
  else
    echo "FAIL: Test 24 [RED]: expected pre-fix false-clean (rc=0), got rc=$SS_RC"
    FAIL=$((FAIL + 1))
  fi
else
  if [ "$SS_RC" -ne 0 ]; then
    echo "PASS: Test 24: --staged failed enumeration refuses to certify (rc=$SS_RC)"
    PASS=$((PASS + 1))
  else
    echo "FAIL: Test 24: --staged failed enumeration reported CLEAN (rc=0) — false certification"
    FAIL=$((FAIL + 1))
  fi
  # Fail-closed companion: a bare non-zero exit is also what an
  # always-block scanner produces. The REASON must be the enumeration.
  if printf '%s' "$SS_OUT" | grep -qF -- 'enumeration failed'; then
    echo "PASS: Test 24b: refusal names the enumeration failure, not a generic non-zero exit"
    PASS=$((PASS + 1))
  else
    echo "FAIL: Test 24b: refusal does NOT name the enumeration failure"
    printf '%s\n' "$SS_OUT" | sed 's/^/    /'
    FAIL=$((FAIL + 1))
  fi
fi

# Test 25 — FAIL-CLOSED COMPANION for Tests 22/23. Same harness, same
# non-ASCII path, CLEAN content: must exit 0. A scanner that blanket-refuses
# (the vacuity oracle's "exit 1 always") passes 22 and 23 but CANNOT pass this,
# so the three assertions together are two-sided.
staged_scan "sécret.md" clean
if [ "$SS_RC" -eq 0 ]; then
  echo "PASS: Test 25: --staged non-ASCII path with CLEAN content exits 0 (not a blanket refusal)"
  PASS=$((PASS + 1))
else
  echo "FAIL: Test 25: --staged non-ASCII path with CLEAN content wrongly refused (rc=$SS_RC)"
  printf '%s\n' "$SS_OUT" | sed 's/^/    /'
  FAIL=$((FAIL + 1))
fi

# ---------------------------------------------------------------------------
# Tests 26-29 (round 18, 2026-08-13) — TREE mode over a NON-ASCII path.
#
# §11.4.115 polarity switch, same shape as Tests 22-25 but for the mode the
# header documents as "scan the whole tracked git tree" (no arguments).
#
# THE DEFECT this guards (measured 2026-08-13, round-17-exit scanner
# b3e0ab4d…): round 16 converted `--staged` to a NUL-delimited enumeration
# and left tree mode on the newline-delimited `git ls-files`, so the IDENTICAL
# C-quoting fail-open survived in the other mode of the same file. One tracked
# file, synthetic Google-shaped key, filename the ONLY variable:
#     CONTROL  secret.md   rc=1  FAIL reported
#     ATTACK   sécret.md   rc=0  "OK: no ... found (mode=tree)"   tracked as
#                                "s\303\251cret.md"
# The same scanner's `--staged` mode returned rc=1 for BOTH names in that run.
# This matters beyond the scan: docs/audit/bypass_events.md ROW 2 cites
# tree-mode exit 0 as its clean-tree evidence (§11.4.75 audit trail). Round-20
# correction: this said "rows 1 and 2 both" — row 1 cites files mode on two
# named paths (raw argv, never C-quoted), and no tracked path in this repo is
# C-quotable at either cited rev, so row 2 is contradicted by the pre-existing
# HXC-331 findings rather than by this defect.
# ---------------------------------------------------------------------------
echo ""
echo "--- Tree-mode non-ASCII path (round 18; RED_MODE=${RED_MODE:-0}) ---"

# Test 26 — CONTROL (plain ASCII), proves the tree-mode instrument works.
tree_scan "secret.md"
if [ "$SS_RC" -ne 0 ]; then
  echo "PASS: Test 26: tree CONTROL, ASCII path, key detected (rc=$SS_RC)"
  PASS=$((PASS + 1))
  assert_named_finding "Test 26b: tree CONTROL finding is specific" "secret.md:1"
else
  echo "FAIL: Test 26: tree CONTROL, ASCII path — key MISSED (rc=0); harness invalid"
  FAIL=$((FAIL + 1))
  echo "FAIL: Test 26b: tree CONTROL finding is specific (no finding to inspect)"
  FAIL=$((FAIL + 1))
fi

# Test 27 — ATTACK (non-ASCII path) in tree mode: the polarity-switched
# assertion for the round-18 fix.
tree_scan "sécret.md"
if [ "${RED_MODE:-0}" = "1" ]; then
  if [ "$SS_RC" -eq 0 ]; then
    echo "PASS: Test 27 [RED]: pre-fix scanner LEAKS non-ASCII path in TREE mode (rc=0 — defect reproduced)"
    PASS=$((PASS + 1))
  else
    echo "FAIL: Test 27 [RED]: expected the pre-fix leak (rc=0), got rc=$SS_RC — defect NOT reproduced"
    FAIL=$((FAIL + 1))
  fi
else
  if [ "$SS_RC" -ne 0 ]; then
    echo "PASS: Test 27: tree non-ASCII path, key detected (rc=$SS_RC)"
    PASS=$((PASS + 1))
    assert_named_finding "Test 27b: tree non-ASCII finding is specific" "sécret.md:1"
  else
    echo "FAIL: Test 27: tree non-ASCII path LEAKED (rc=0) — §11.4.10 credential leak"
    FAIL=$((FAIL + 1))
    echo "FAIL: Test 27b: tree non-ASCII finding is specific (no finding to inspect)"
    FAIL=$((FAIL + 1))
  fi
fi

# Test 28 — FAIL-CLOSED COMPANION for Tests 26/27 (see Test 25).
tree_scan "sécret.md" clean
if [ "$SS_RC" -eq 0 ]; then
  echo "PASS: Test 28: tree non-ASCII path with CLEAN content exits 0 (not a blanket refusal)"
  PASS=$((PASS + 1))
else
  echo "FAIL: Test 28: tree non-ASCII path with CLEAN content wrongly refused (rc=$SS_RC)"
  printf '%s\n' "$SS_OUT" | sed 's/^/    /'
  FAIL=$((FAIL + 1))
fi

# Test 29 — tree mode must not infer "nothing tracked" from a FAILED
# `git ls-files`, mirroring Test 24 for the other enumeration.
shimmed_enum_scan ls-files
if [ "${RED_MODE:-0}" = "1" ]; then
  if [ "$SS_RC" -eq 0 ]; then
    echo "PASS: Test 29 [RED]: pre-fix tree mode reports clean on a FAILED enumeration (rc=0)"
    PASS=$((PASS + 1))
  else
    echo "FAIL: Test 29 [RED]: expected pre-fix false-clean (rc=0), got rc=$SS_RC"
    FAIL=$((FAIL + 1))
  fi
else
  if [ "$SS_RC" -ne 0 ] && printf '%s' "$SS_OUT" | grep -qF -- 'enumeration failed'; then
    echo "PASS: Test 29: tree failed enumeration refuses to certify, naming the reason (rc=$SS_RC)"
    PASS=$((PASS + 1))
  else
    echo "FAIL: Test 29: tree failed enumeration did not refuse-with-reason (rc=$SS_RC)"
    printf '%s\n' "$SS_OUT" | sed 's/^/    /'
    FAIL=$((FAIL + 1))
  fi
fi

# ---------------------------------------------------------------------------
# Tests 30-33 — TRACKED SYMLINKS (git mode 120000), round 20, 2026-08-13.
#
# THE DEFECT this guards (measured against the round-19-exit scanner,
# e8e175e0c46f4cb1…): scan_disk_file branched on `[ -f ]` then `[ -d ]`, and
# POSIX makes BOTH of those FOLLOW a symlink — verbatim: "With the exception of
# the -h pathname and -L pathname primaries, if a pathname argument is a
# symbolic link, test shall evaluate the expression by resolving the symbolic
# link and using the file referenced by the link" (Open Group Base
# Specifications Issue 7, `test` utility; quote re-verified against
# pubs.opengroup.org/onlinepubs/9699919799/utilities/test.html on 2026-08-13,
# because an earlier draft of this very comment misquoted it — it dropped both
# "pathname" words and truncated the clause). So a tracked symlink was never
# judged by its OWN content — the target path string, which is exactly what git
# stores in a 120000 blob — but by whatever it happened to point AT on that
# filesystem, at that moment:
#   * pointing at a directory  -> the gitlink branch skipped it silently, so a
#     key in the link string was never scanned (Test 30);
#   * pointing at a regular file -> `cat` opened and scanned THAT file, even if
#     it lay outside the repository entirely (Test 32) — the repo-escape read
#     documented in CVE-2025-8110 / CVE-2026-52811 / CVE-2026-71556, and a
#     result that varies with ambient filesystem state rather than being a pure
#     function of the committed tree (§11.4.50 determinism).
# Round 20 routes every symlink to its own content via `readlink` (which reads
# the link, never opens the target) BEFORE the dereferencing tests run.
# ---------------------------------------------------------------------------
echo ""
echo "--- Tracked symlink content (round 20; RED_MODE=${RED_MODE:-0}) ---"

# Test 30 — the polarity-switched assertion for the round-20 fix: the key is in
# the LINK'S OWN content and the link resolves to a directory.
tree_scan_symlink "link_to_dir" dir-key
if [ "${RED_MODE:-0}" = "1" ]; then
  if [ "$SS_RC" -eq 0 ]; then
    echo "PASS: Test 30 [RED]: pre-fix scanner skips a symlink at the gitlink branch, key in link string MISSED (rc=0 — defect reproduced)"
    PASS=$((PASS + 1))
  else
    echo "FAIL: Test 30 [RED]: expected the pre-fix silent skip (rc=0), got rc=$SS_RC — defect NOT reproduced"
    FAIL=$((FAIL + 1))
  fi
else
  if [ "$SS_RC" -ne 0 ]; then
    echo "PASS: Test 30: key in a tracked symlink's own content detected (rc=$SS_RC)"
    PASS=$((PASS + 1))
    assert_named_finding "Test 30b: symlink finding is specific" "link_to_dir:1"
  else
    echo "FAIL: Test 30: key in a tracked symlink's own content MISSED (rc=0) — silent skip"
    FAIL=$((FAIL + 1))
    echo "FAIL: Test 30b: symlink finding is specific (no finding to inspect)"
    FAIL=$((FAIL + 1))
  fi
fi

# Test 31 — FAIL-CLOSED COMPANION for Test 30, and the shape this repo really
# has (skills/media-validator -> ../constitution/skills/media-validator): an
# ordinary symlink to a directory must NOT become a blanket refusal.
tree_scan_symlink "link_to_dir" dir-clean
if [ "$SS_RC" -eq 0 ]; then
  echo "PASS: Test 31: clean symlink-to-directory exits 0 (not a blanket refusal)"
  PASS=$((PASS + 1))
else
  echo "FAIL: Test 31: clean symlink-to-directory wrongly refused (rc=$SS_RC)"
  printf '%s\n' "$SS_OUT" | sed 's/^/    /'
  FAIL=$((FAIL + 1))
fi

# Test 32 — NO REPO-ESCAPE READ. The link points at a file OUTSIDE the repo.
# That file is not repo content, and opening it is the dereference this fix
# removes; the scanner must judge the link by its own content instead.
tree_scan_symlink "link_out" outside-key
if [ "${RED_MODE:-0}" = "1" ]; then
  if [ "$SS_RC" -ne 0 ]; then
    echo "PASS: Test 32 [RED]: pre-fix scanner DEREFERENCED the link and read a file outside the repo (rc=$SS_RC — defect reproduced)"
    PASS=$((PASS + 1))
  else
    echo "FAIL: Test 32 [RED]: expected the pre-fix repo-escape read (rc!=0), got rc=0 — defect NOT reproduced"
    FAIL=$((FAIL + 1))
  fi
else
  if [ "$SS_RC" -eq 0 ]; then
    echo "PASS: Test 32: symlink to an outside file is judged by its own content, not dereferenced (rc=0)"
    PASS=$((PASS + 1))
  else
    echo "FAIL: Test 32: scanner still reads outside the repository through a tracked symlink (rc=$SS_RC)"
    printf '%s\n' "$SS_OUT" | sed 's/^/    /'
    FAIL=$((FAIL + 1))
  fi
fi

# Test 33 — NO IN-REPO COVERAGE LOST. A key in a TRACKED file is still found,
# under its own canonical path, when a tracked symlink also points at it.
# Expected non-zero in BOTH polarities: this invariant holds before and after.
tree_scan_symlink "link_to_file" tracked-key
if [ "$SS_RC" -ne 0 ]; then
  echo "PASS: Test 33: tracked file's key still detected alongside a symlink to it (rc=$SS_RC)"
  PASS=$((PASS + 1))
  assert_named_finding "Test 33b: the finding names the canonical path" "canonical.md:1"
else
  echo "FAIL: Test 33: tracked file's key MISSED when a symlink also points at it (rc=0)"
  FAIL=$((FAIL + 1))
  echo "FAIL: Test 33b: the finding names the canonical path (no finding to inspect)"
  FAIL=$((FAIL + 1))
fi

# ---------------------------------------------------------------------------
# Tests 34-36 — STAGED GITLINKS (submodule pointers, mode 160000), round 20.
#
# THE DEFECT this guards: scan_staged_file reads content with
# `git show ":$f"`, and round 16 correctly made an unreadable path refuse to
# certify instead of silently passing. Its comment justified that as safe with
# a FALSE measured claim — that `git show` returns rc=0 for a gitlink. It does
# not: a submodule's commit object lives in the SUBMODULE's object store, so
# the parent returns rc=128. Every staged submodule-pointer bump therefore hit
# the refusal path and pre-commit step 4 BLOCKED the commit, in a repo with 130
# gitlinks. Neither suite caught it because neither ever staged one.
# ---------------------------------------------------------------------------
staged_index_scan() {
  # staged_index_scan <kind> -> SS_RC / SS_OUT for --staged mode.
  #   gitlink         a submodule pointer whose sha is absent from this repo's
  #                   object store (the realistic shape — `git show` rc=128)
  #   gitlink+secret  the same pointer PLUS a normal staged file holding a key
  #   absent-blob     mode 100644 whose blob sha is absent: unreadable, but NOT
  #                   a gitlink, so it must still refuse (narrowness control)
  local kind="$1"
  local repo; repo=$(mktemp -d)
  local absent_sha=0123456789012345678901234567890123456789
  (
    cd "$repo" || exit 99
    git init -q . >/dev/null 2>&1
    git config user.email t@t.t; git config user.name t
    echo base > base.txt; git add base.txt; git commit -qm base >/dev/null 2>&1
    case "$kind" in
      gitlink)
        git update-index --add --cacheinfo "160000,$absent_sha,submodules/some_module" ;;
      gitlink+secret)
        git update-index --add --cacheinfo "160000,$absent_sha,submodules/some_module"
        frag="SyD-fabricated0123456789abcdef"
        printf 'GEMINI_API_KEY=AIza%s\n' "$frag" > leak.md
        git add -- leak.md >/dev/null 2>&1 ;;
      absent-blob)
        git update-index --add --cacheinfo "100644,$absent_sha,ghost.txt" ;;
      *) exit 98 ;;
    esac
    out=$("$SCANNER" --staged 2>&1)
    echo $? > "$repo/.rc"
    printf '%s' "$out" > "$repo/.out"
  ) >/dev/null 2>&1
  SS_RC=$(cat "$repo/.rc" 2>/dev/null || echo 99)
  SS_OUT=$(cat "$repo/.out" 2>/dev/null || echo "")
  rm -rf "$repo"
}

echo ""
echo "--- Staged gitlinks (round 20; RED_MODE=${RED_MODE:-0}) ---"

# Test 34 — the polarity-switched assertion: a staged submodule-pointer bump
# must not be refused. This is a §11.4.201 false-positive-refusal guard, the
# mirror image of the fail-open guards above.
staged_index_scan gitlink
if [ "${RED_MODE:-0}" = "1" ]; then
  if [ "$SS_RC" -ne 0 ]; then
    echo "PASS: Test 34 [RED]: pre-fix scanner REFUSES a staged submodule bump (rc=$SS_RC — over-block reproduced)"
    PASS=$((PASS + 1))
  else
    echo "FAIL: Test 34 [RED]: expected the pre-fix over-block (rc!=0), got rc=0 — defect NOT reproduced"
    FAIL=$((FAIL + 1))
  fi
else
  if [ "$SS_RC" -eq 0 ]; then
    echo "PASS: Test 34: staged submodule-pointer bump is not refused (rc=0)"
    PASS=$((PASS + 1))
  else
    echo "FAIL: Test 34: staged submodule-pointer bump WRONGLY refused (rc=$SS_RC) — every submodule bump is blocked"
    printf '%s\n' "$SS_OUT" | sed 's/^/    /'
    FAIL=$((FAIL + 1))
  fi
fi

# Test 35 — the gitlink exemption must not BLIND the scan: a real staged key in
# the same commit is still caught, and the finding still names that file.
# Expected non-zero in BOTH polarities.
staged_index_scan gitlink+secret
if [ "$SS_RC" -ne 0 ]; then
  echo "PASS: Test 35: a real staged key alongside a gitlink is still detected (rc=$SS_RC)"
  PASS=$((PASS + 1))
  assert_named_finding "Test 35b: the finding names the leaking file" "leak.md:1"
else
  echo "FAIL: Test 35: gitlink exemption BLINDED the staged scan — real key MISSED (rc=0)"
  FAIL=$((FAIL + 1))
  echo "FAIL: Test 35b: the finding names the leaking file (no finding to inspect)"
  FAIL=$((FAIL + 1))
fi

# Test 36 — NARROWNESS CONTROL. An unreadable staged path that is NOT a gitlink
# (mode 100644, blob absent) must STILL refuse to certify. Without this, the
# Test 34 exemption could be widened into a fresh silent skip — the exact
# defect round 16 removed. Expected non-zero in BOTH polarities.
staged_index_scan absent-blob
if [ "$SS_RC" -ne 0 ] && printf '%s' "$SS_OUT" | grep -qF -- 'NOT certified clean'; then
  echo "PASS: Test 36: unreadable NON-gitlink staged path still refuses to certify (rc=$SS_RC)"
  PASS=$((PASS + 1))
else
  echo "FAIL: Test 36: unreadable non-gitlink staged path was certified clean (rc=$SS_RC) — silent skip"
  printf '%s\n' "$SS_OUT" | sed 's/^/    /'
  FAIL=$((FAIL + 1))
fi

# ---------------------------------------------------------------------------
# Tests 37-39 — DIR-SHADOWED TRACKED FILES, round 20.
#
# THE DEFECT this guards: `rm f && mkdir f` leaves a tracked regular file at
# mode 100644 with its content still readable from the index, while `[ -d ]`
# routes the path to the gitlink branch, which returned it CLEAN. Found by
# independent review of the round-20 diff, whose comment had claimed that
# branch could only see gitlinks. The behaviour predates round 20 — HEAD's
# scanner returns rc=0 on the same fixture — but nothing tested it.
# ---------------------------------------------------------------------------
tree_scan_dirshadow() {
  # tree_scan_dirshadow <kind>: shadowed-key | shadowed-clean | untracked-dir
  local kind="$1"
  local repo; repo=$(mktemp -d)
  (
    cd "$repo" || exit 99
    git init -q . >/dev/null 2>&1
    git config user.email t@t.t; git config user.name t
    case "$kind" in
      shadowed-key)
        frag="SyD-fabricated0123456789abcdef"
        printf 'GEMINI_API_KEY=AIza%s\n' "$frag" > shadowed.txt ;;
      shadowed-clean)
        printf 'ordinary prose, nothing sensitive on this line\n' > shadowed.txt ;;
      untracked-dir|broken-index)
        frag="SyD-fabricated0123456789abcdef"
        printf 'GEMINI_API_KEY=AIza%s\n' "$frag" > shadowed.txt ;;
    esac
    git add -- shadowed.txt >/dev/null 2>&1
    git commit -qm base >/dev/null 2>&1
    if [ "$kind" = "untracked-dir" ]; then
      mkdir -p plain_dir                       # never tracked; files-mode arg
      out=$("$SCANNER" plain_dir 2>&1)
    elif [ "$kind" = "broken-index" ]; then
      rm shadowed.txt && mkdir shadowed.txt    # worktree slot becomes a dir
      printf 'CORRUPT' > .git/index            # index lookup can no longer answer
      out=$("$SCANNER" shadowed.txt 2>&1)      # files mode: the reachable surface
    else
      rm shadowed.txt && mkdir shadowed.txt    # worktree slot becomes a dir
      out=$("$SCANNER" 2>&1)
    fi
    echo $? > "$repo/.rc"
    printf '%s' "$out" > "$repo/.out"
  ) >/dev/null 2>&1
  SS_RC=$(cat "$repo/.rc" 2>/dev/null || echo 99)
  SS_OUT=$(cat "$repo/.out" 2>/dev/null || echo "")
  rm -rf "$repo"
}

echo ""
echo "--- Dir-shadowed tracked files (round 20; RED_MODE=${RED_MODE:-0}) ---"

# Test 37 — polarity-switched assertion: the key is still in the index, so the
# scan must find it even though the worktree slot is now a directory.
tree_scan_dirshadow shadowed-key
if [ "${RED_MODE:-0}" = "1" ]; then
  if [ "$SS_RC" -eq 0 ]; then
    echo "PASS: Test 37 [RED]: pre-fix scanner certifies a dir-shadowed tracked file CLEAN (rc=0 — defect reproduced)"
    PASS=$((PASS + 1))
  else
    echo "FAIL: Test 37 [RED]: expected the pre-fix silent skip (rc=0), got rc=$SS_RC — defect NOT reproduced"
    FAIL=$((FAIL + 1))
  fi
else
  if [ "$SS_RC" -ne 0 ]; then
    echo "PASS: Test 37: key in a dir-shadowed tracked file detected from the index (rc=$SS_RC)"
    PASS=$((PASS + 1))
    assert_named_finding "Test 37b: dir-shadow finding is specific" "shadowed.txt:1"
  else
    echo "FAIL: Test 37: dir-shadowed tracked file certified CLEAN (rc=0) — silent skip"
    FAIL=$((FAIL + 1))
    echo "FAIL: Test 37b: dir-shadow finding is specific (no finding to inspect)"
    FAIL=$((FAIL + 1))
  fi
fi

# Test 38 — FAIL-CLOSED COMPANION: a dir-shadowed tracked file with CLEAN
# content must not become a blanket refusal. Non-zero here would mean the
# Test 37 fix turned an ordinary diverged worktree into a commit blocker.
tree_scan_dirshadow shadowed-clean
if [ "$SS_RC" -eq 0 ]; then
  echo "PASS: Test 38: dir-shadowed tracked file with CLEAN content exits 0 (not a blanket refusal)"
  PASS=$((PASS + 1))
else
  echo "FAIL: Test 38: dir-shadowed tracked file with CLEAN content wrongly refused (rc=$SS_RC)"
  printf '%s\n' "$SS_OUT" | sed 's/^/    /'
  FAIL=$((FAIL + 1))
fi

# Test 39 — NARROWNESS CONTROL: an UNTRACKED plain directory passed as a
# files-mode argument has no index entry, so it must still be skipped rather
# than refused. Guards the empty-mode arm of the Test 37 fix.
tree_scan_dirshadow untracked-dir
if [ "$SS_RC" -eq 0 ]; then
  echo "PASS: Test 39: untracked directory argument is skipped, not refused (rc=0)"
  PASS=$((PASS + 1))
else
  echo "FAIL: Test 39: untracked directory argument wrongly refused (rc=$SS_RC)"
  printf '%s\n' "$SS_OUT" | sed 's/^/    /'
  FAIL=$((FAIL + 1))
fi

# Test 40 — a FAILED index lookup is not an answer. `rc=0 + empty` means the
# path is untracked; `rc!=0` means the lookup could not answer at all, and the
# first version of the Test 37 fix piped git into awk, discarding git's status
# and treating both as "untracked" — certifying a key-bearing tracked file
# clean whenever the index was unreadable. Found by independent review.
tree_scan_dirshadow broken-index
if [ "${RED_MODE:-0}" = "1" ]; then
  if [ "$SS_RC" -eq 0 ]; then
    echo "PASS: Test 40 [RED]: pre-fix scanner certifies CLEAN when the index lookup FAILS (rc=0 — defect reproduced)"
    PASS=$((PASS + 1))
  else
    echo "FAIL: Test 40 [RED]: expected the pre-fix false-clean (rc=0), got rc=$SS_RC — defect NOT reproduced"
    FAIL=$((FAIL + 1))
  fi
else
  if [ "$SS_RC" -ne 0 ] && printf '%s' "$SS_OUT" | grep -qF -- 'NOT certified clean'; then
    echo "PASS: Test 40: a failed index lookup refuses to certify, naming the reason (rc=$SS_RC)"
    PASS=$((PASS + 1))
  else
    echo "FAIL: Test 40: failed index lookup was treated as 'untracked' and certified clean (rc=$SS_RC)"
    printf '%s\n' "$SS_OUT" | sed 's/^/    /'
    FAIL=$((FAIL + 1))
  fi
fi

# ---------------------------------------------------------------------------
# Tests 41-42 — the index lookup must answer about $f, not about a neighbour.
# Both found by independent review of round 20.
#   41: `:(literal)` still matches by DIRECTORY PREFIX, so a tracked-content
#       directory passed to files mode answered with a CHILD's row; the child's
#       mode (100644) then drove a `git show ":<dir>"` that could only fail,
#       refusing an ordinary directory with a false "shadowed" message. NEW in
#       round 20 (HEAD and the round-19 mirror both exit 0).
#   42: a gitlink whose worktree directory is absent reaches the else branch and
#       `git show` cannot resolve it either, so it was refused as "not in the
#       index" — false, it is in the index at mode 160000. PRE-EXISTING: the
#       round-19 mirror refuses it too (measured), so this test's RED polarity
#       reproduces against that mirror as well.
# ---------------------------------------------------------------------------
index_neighbour_scan() {
  # <kind>: tracked-content-dir | gitlink-dir-absent | gitlink-dir-present
  local kind="$1"
  local repo; repo=$(mktemp -d)
  (
    cd "$repo" || exit 99
    git init -q . >/dev/null 2>&1
    git config user.email t@t.t; git config user.name t
    echo base > base.txt; git add base.txt; git commit -qm base >/dev/null 2>&1
    case "$kind" in
      tracked-content-dir)
        mkdir qa; printf 'ordinary prose, nothing sensitive\n' > qa/file.txt
        git add -- qa/file.txt >/dev/null 2>&1
        git commit -qm add >/dev/null 2>&1
        out=$("$SCANNER" qa 2>&1) ;;                    # files mode, dir arg
      gitlink-dir-absent|gitlink-dir-present)
        git update-index --add --cacheinfo \
          160000,0123456789012345678901234567890123456789,submodules/some_module
        [ "$kind" = "gitlink-dir-present" ] && mkdir -p submodules/some_module
        out=$("$SCANNER" 2>&1) ;;                       # tree mode
      *) exit 98 ;;
    esac
    echo $? > "$repo/.rc"
    printf '%s' "$out" > "$repo/.out"
  ) >/dev/null 2>&1
  SS_RC=$(cat "$repo/.rc" 2>/dev/null || echo 99)
  SS_OUT=$(cat "$repo/.out" 2>/dev/null || echo "")
  rm -rf "$repo"
}

echo ""
echo "--- Index lookup answers about \$f (round 20; RED_MODE=${RED_MODE:-0}) ---"

# Test 41 — a directory that merely CONTAINS tracked files has no entry of its
# own; it must be skipped, never refused as a shadowed tracked file.
# Expected rc=0 in BOTH polarities. Unlike Tests 30/32/34/37/40/42, the defect
# this guards never existed outside round 20 — it was introduced by the first
# draft of the round-20 `-d` index lookup (awk NR==1 on a prefix-matched row)
# and removed in the same round, so the round-19 mirror is NOT a broken artifact
# for it and a RED branch here would assert a defect that artifact never had.
# It is load-bearing all the same: reverting the lookup to that draft makes this
# scenario exit 1 (captured against a reconstructed mutant, never the live file).
index_neighbour_scan tracked-content-dir
if [ "$SS_RC" -eq 0 ]; then
  echo "PASS: Test 41: directory containing tracked files is skipped, not refused (rc=0)"
  PASS=$((PASS + 1))
else
  echo "FAIL: Test 41: directory containing tracked files WRONGLY refused (rc=$SS_RC) — the lookup answered about a child"
  printf '%s\n' "$SS_OUT" | sed 's/^/    /'
  FAIL=$((FAIL + 1))
fi

# Test 42 — a gitlink whose worktree directory is absent is still a gitlink.
index_neighbour_scan gitlink-dir-absent
if [ "${RED_MODE:-0}" = "1" ]; then
  if [ "$SS_RC" -ne 0 ]; then
    echo "PASS: Test 42 [RED]: pre-fix scanner refuses a gitlink with no worktree directory (rc=$SS_RC — defect reproduced)"
    PASS=$((PASS + 1))
  else
    echo "FAIL: Test 42 [RED]: expected the pre-fix over-block (rc!=0), got rc=0 — defect NOT reproduced"
    FAIL=$((FAIL + 1))
  fi
else
  if [ "$SS_RC" -eq 0 ]; then
    echo "PASS: Test 42: gitlink with no worktree directory is skipped, not refused (rc=0)"
    PASS=$((PASS + 1))
  else
    echo "FAIL: Test 42: gitlink with no worktree directory WRONGLY refused (rc=$SS_RC)"
    printf '%s\n' "$SS_OUT" | sed 's/^/    /'
    FAIL=$((FAIL + 1))
  fi
fi

# Test 43 — PARITY check for Test 42: the ordinary present-worktree-directory
# gitlink must behave identically to the absent-directory one, so the round-20
# fix neither special-cased the absent shape nor regressed the normal shape.
#
# WHAT IT DOES NOT PROVE (corrected round 21). This comment previously claimed
# "Proves 42's PASS is not 'everything exits 0'". That was false: Test 43
# expects rc=0 — the SAME outcome as the test it controls — so it has no
# discriminating power for that claim. A control that still passes when its
# subject's assertion has gone vacuous is a parity check, not a discriminator.
# MEASURED against an always-exit-0 stand-in scanner (a two-line script whose
# body is just `exit 0`, driven in via SCANNER_UNDER_TEST): Tests 41, 42 AND 43
# all PASS, while the suite as a whole correctly fails —
#   Results: 20 passed, 37 failed
# so the COVERAGE was never the problem, only this sentence describing it.
#
# The genuine "the scanner does not simply exit 0 for everything" discriminator
# is the MUST-detect class — Tests 1-7b, 13, 15, 17, 19 — each of which expects
# NON-ZERO and therefore cannot pass a scanner that always exits 0. Verified
# against that same stand-in, all 12 fail and none pass:
#   grep -cE '^FAIL: Test (1|2|3|4|5|6|7|7b|13|15|17|19): ' out.txt  -> 12
#   grep -cE '^PASS: Test (1|2|3|4|5|6|7|7b|13|15|17|19): ' out.txt  ->  0
index_neighbour_scan gitlink-dir-present
if [ "$SS_RC" -eq 0 ]; then
  echo "PASS: Test 43: gitlink with its worktree directory present is skipped (rc=0)"
  PASS=$((PASS + 1))
else
  echo "FAIL: Test 43: ordinary gitlink wrongly refused (rc=$SS_RC)"
  printf '%s\n' "$SS_OUT" | sed 's/^/    /'
  FAIL=$((FAIL + 1))
fi

echo ""
echo "Results: $PASS passed, $FAIL failed"
[ "$MIRROR_TAINT" -eq 0 ] || echo "Results: MIRROR_TAINT=1 — the live scanner was written to during this run (§11.4.84)"
[ "$FAIL" -eq 0 ] && [ "$MIRROR_TAINT" -eq 0 ]
