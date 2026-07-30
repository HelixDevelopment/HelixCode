#!/usr/bin/env bash
# scripts/gates/no_hardcoded_db_credential_gate.sh
#
# HXC-168 / CONST-042 (Article XII §12.1) regression guard: no database (or
# adjacent service) credential may appear as a LITERAL in the setup and
# container files this project publishes. Those files must read the value from
# the private, gitignored configuration source (.env -> environment).
#
# Why this gate exists (§11.4.135): a published credential cannot be un-published
# — it lives in git history on every mirror forever. The only remedy for a leak
# is rotation, so the ONLY thing code can guarantee is that a new literal never
# lands again. That is exactly what this gate pins.
#
# Two independent checks:
#
#   CHECK 1 — historical-literal denylist.
#       The specific credential HXC-168 exposed must not reappear in ANY tracked
#       file, outside third-party submodules and documentation prose (prose that
#       DISCUSSES the incident is legitimate; a config that USES the value is not).
#
#   CHECK 2 — credential-sourcing check (generic, forward-looking).
#       In the deployed setup/container/config scope, every credential assignment
#       must reference an environment variable (`${VAR}` / `${VAR:?...}`) rather
#       than carry a literal. This catches a BRAND-NEW hardcoded password, not
#       only the historical one — a denylist alone would not.
#
# Usage:
#   no_hardcoded_db_credential_gate.sh              # run the gate (exit 1 on any finding)
#   no_hardcoded_db_credential_gate.sh --self-test  # §1.1 paired mutation: prove the
#                                                   # gate FAILS on a planted literal
#                                                   # and PASSES on the real tree.
#
# The self-test NEVER writes inside the repository (§11.4.84 working-tree
# quiescence): it plants its mutation in a mktemp directory only.
set -uo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT" || exit 2

RED=$'\033[31m'; GREEN=$'\033[32m'; YELLOW=$'\033[33m'; BOLD=$'\033[1m'; OFF=$'\033[0m'

# The historical literal, assembled at runtime so this gate file does not itself
# contain the credential (the gate would otherwise flag itself, and committing
# the value here would re-publish it).
HISTORICAL_LITERAL="helix""pass"

# Deployed setup / container / config scope for CHECK 2. Explicit rather than
# globbed so the scope is reviewable and cannot silently drift.
SCOPE_FILES=(
  "Dockerfile"
  "docker-compose.helix.yml"
  ".env.example"
  "helix_code/docker-compose.yml"
  "helix_code/docker-compose.builder.yml"
  "helix_code/docker/docker-compose.yml"
  "helix_code/docker/autoboot/docker-compose.autoboot.yml"
  "helix_code/config/config.yaml"
  "helix_code/config/production-config.yaml"
  "helix_code/config/working-config.yaml"
  "helix_code/config/fixed-config.yaml"
  "helix_code/config/minimal-test-config.yaml"
  "docs/distribution/docker-compose.mistborn.yml"
  "tests/e2e/challenges/helix_qa_live_anti_bluff.sh"
)

# Credential keys whose assigned value must be env-sourced.
CRED_KEY_RE='(POSTGRES_PASSWORD|HELIX_DATABASE_PASSWORD|HELIX_AUTOBOOT_PG_PASSWORD|HELIX_POSTGRES_PASSWORD|HELIX_REDIS_PASSWORD|GF_SECURITY_ADMIN_PASSWORD|SONAR_JDBC_PASSWORD)'

# ---------------------------------------------------------------------------
# CHECK 2 implementation, factored so --self-test can run it against a planted
# copy in a temp dir without touching the working tree.
# Prints one finding per line; returns 1 if any finding was printed.
# ---------------------------------------------------------------------------
scan_file_for_literals() {
  local f="$1" label="${2:-$1}" found=0 line lineno value

  [ -f "$f" ] || return 0

  # (a) KEY=value / KEY: value assignments.
  while IFS=: read -r lineno line; do
    [ -n "${lineno:-}" ] || continue
    # Strip a leading comment / YAML-comment line.
    case "$(printf '%s' "$line" | sed 's/^[[:space:]]*//')" in
      '#'*) continue ;;
    esac
    # An `${...}` anywhere on the line means the value is env-sourced. Testing
    # for containment (rather than parsing the value out) is deliberate: a
    # `${VAR:?long message}` default contains both `:` and the key name again,
    # which defeats any greedy value-extraction regex.
    case "$line" in
      *'${'*) continue ;;
    esac
    # No env reference on the line — the value is a literal unless it is an
    # explicitly allowed placeholder.
    value="$(printf '%s' "$line" | sed -E "s/^.*${CRED_KEY_RE}[[:space:]]*[:=][[:space:]]*//")"
    # Order matters: drop the shell line-continuation, trailing comment and
    # trailing whitespace BEFORE unquoting, or an intentionally-empty `""`
    # value is mis-read as a literal.
    value="$(printf '%s' "$value" | sed -E 's/[[:space:]]*\\$//; s/[[:space:]]*#.*$//; s/[[:space:]]*$//; s/^["'\'']//; s/["'\'']$//')"
    case "$value" in
      ''|CHANGE_ME*|'<REDACTED>') continue ;;
    esac
    printf '  %s%s:%s%s  credential key assigned a LITERAL value\n' "$BOLD" "$label" "$lineno" "$OFF"
    found=1
  done < <(grep -nE "${CRED_KEY_RE}[[:space:]]*[:=]" "$f" 2>/dev/null)

  # (b) postgres:// URLs carrying an inline password that is not an env ref.
  while IFS=: read -r lineno line; do
    [ -n "${lineno:-}" ] || continue
    case "$(printf '%s' "$line" | sed 's/^[[:space:]]*//')" in
      '#'*) continue ;;
    esac
    printf '  %s%s:%s%s  connection URL embeds a LITERAL password\n' "$BOLD" "$label" "$lineno" "$OFF"
    found=1
  done < <(grep -nE 'postgres(ql)?://[A-Za-z0-9_.-]+:[^$@[:space:]]+@' "$f" 2>/dev/null)

  return $found
}

# scan_repo_for_historical_literal implements CHECK 1 against an arbitrary repo
# root, so --self-test can exercise the SAME pipeline against a throwaway repo
# containing a planted literal. Prints matching "path:line:text" rows.
#
# NOTE: this gate file deliberately assembles the literal from two fragments at
# runtime, so its own bytes never contain it and it needs no self-exclusion —
# a self-exclusion would be a hole an attacker could hide a credential in.
scan_repo_for_historical_literal() {
  local repo="$1"
  git -C "$repo" ls-files -z 2>/dev/null \
    | (cd "$repo" && xargs -0 grep -In -e "$HISTORICAL_LITERAL" 2>/dev/null) \
    | grep -vE '^submodules/' \
    | grep -vE '^[^:]+\.(md|html|pdf):' \
    || true
}

run_gate() {
  local rc=0 f hits

  printf '%s== CHECK 1: historical-literal denylist ==%s\n' "$BOLD" "$OFF"
  # Tracked files only; exclude third-party submodules and documentation prose.
  hits="$(scan_repo_for_historical_literal "$ROOT")"
  if [ -n "$hits" ]; then
    printf '%sFAIL%s — the HXC-168 credential literal is present in tracked config/source:\n' "$RED" "$OFF"
    printf '%s\n' "$hits" | sed 's/^/  /'
    rc=1
  else
    printf '%sPASS%s — literal absent from tracked config/source.\n' "$GREEN" "$OFF"
  fi

  printf '\n%s== CHECK 2: credentials are env-sourced in the deployed scope ==%s\n' "$BOLD" "$OFF"
  local check2_rc=0
  for f in "${SCOPE_FILES[@]}"; do
    if [ ! -f "$f" ]; then
      printf '%sWARN%s — scoped file missing (scope drift?): %s\n' "$YELLOW" "$OFF" "$f"
      continue
    fi
    scan_file_for_literals "$f" || check2_rc=1
  done
  if [ "$check2_rc" -ne 0 ]; then
    printf '%sFAIL%s — the lines above must read the value from .env, e.g.\n' "$RED" "$OFF"
    printf '        POSTGRES_PASSWORD: ${HELIX_DATABASE_PASSWORD:?set it in .env}\n'
    rc=1
  else
    printf '%sPASS%s — every scoped credential assignment is env-sourced.\n' "$GREEN" "$OFF"
  fi

  return $rc
}

# ---------------------------------------------------------------------------
# --self-test: §1.1 paired mutation. Proves the gate is falsifiable — a gate
# that cannot be shown to FAIL is a bluff gate (§11.4 / §11.4.107(10)).
# ---------------------------------------------------------------------------
self_test() {
  local tmp rc_clean rc_planted overall=0
  tmp="$(mktemp -d)" || { printf 'cannot mktemp\n' >&2; exit 2; }
  # shellcheck disable=SC2064
  trap "rm -rf -- '$tmp'" EXIT

  printf '%s== SELF-TEST 1/4: gate PASSES on the real tree (golden-good) ==%s\n' "$BOLD" "$OFF"
  if run_gate >/dev/null 2>&1; then
    rc_clean=0; printf '%sPASS%s — gate reports clean on the working tree.\n\n' "$GREEN" "$OFF"
  else
    rc_clean=1
    printf '%sFAIL%s — gate does not pass on the working tree; full output:\n' "$RED" "$OFF"
    run_gate || true
    overall=1
  fi

  printf '%s== SELF-TEST 2/4: gate FAILS on a planted literal (golden-bad) ==%s\n' "$BOLD" "$OFF"
  # Plant a credential literal in a COPY, in the temp dir only.
  printf 'services:\n  postgres:\n    environment:\n      POSTGRES_PASSWORD: %s\n' \
    "$HISTORICAL_LITERAL" > "$tmp/planted-compose.yml"
  if scan_file_for_literals "$tmp/planted-compose.yml" "planted-compose.yml" >"$tmp/out" 2>&1; then
    rc_planted=0
    printf '%sFAIL%s — gate did NOT flag a planted credential literal. The gate is a bluff.\n' "$RED" "$OFF"
    overall=1
  else
    rc_planted=1
    printf '%sPASS%s — gate flagged the planted literal:\n' "$GREEN" "$OFF"
    sed 's/^/    /' "$tmp/out"
  fi
  printf '\n'

  printf '%s== SELF-TEST 3/4: gate FAILS on a planted inline-URL password ==%s\n' "$BOLD" "$OFF"
  printf 'services:\n  app:\n    environment:\n      - DB=postgres://helix:s3cr3t-planted@postgres:5432/db\n' \
    > "$tmp/planted-url.yml"
  if scan_file_for_literals "$tmp/planted-url.yml" "planted-url.yml" >"$tmp/out2" 2>&1; then
    printf '%sFAIL%s — gate did NOT flag an inline-URL credential.\n' "$RED" "$OFF"
    overall=1
  else
    printf '%sPASS%s — gate flagged the planted inline-URL credential:\n' "$GREEN" "$OFF"
    sed 's/^/    /' "$tmp/out2"
  fi

  printf '\n%s== SELF-TEST 4/4: CHECK 1 FAILS on a tracked planted literal ==%s\n' "$BOLD" "$OFF"
  # Exercise the REAL CHECK 1 pipeline (git ls-files + grep + exclusions) against
  # a throwaway repository, so the denylist half is proven falsifiable without
  # ever dirtying this working tree (§11.4.84 — other agents share this checkout).
  mkdir -p "$tmp/repo"
  git -C "$tmp/repo" init -q 2>/dev/null
  printf 'POSTGRES_PASSWORD: %s\n' "$HISTORICAL_LITERAL" > "$tmp/repo/compose.yml"
  # A prose file must NOT trip the gate — discussing the incident is legitimate.
  printf 'The leaked value was `%s`.\n' "$HISTORICAL_LITERAL" > "$tmp/repo/NOTES.md"
  git -C "$tmp/repo" add compose.yml NOTES.md 2>/dev/null
  git -C "$tmp/repo" -c user.email=gate@test -c user.name=gate commit -qm planted 2>/dev/null

  local planted_hits
  planted_hits="$(scan_repo_for_historical_literal "$tmp/repo")"
  if [ -z "$planted_hits" ]; then
    printf '%sFAIL%s — CHECK 1 did NOT flag a tracked planted literal. The denylist is a bluff.\n' "$RED" "$OFF"
    overall=1
  elif printf '%s' "$planted_hits" | grep -q '^NOTES\.md:'; then
    printf '%sFAIL%s — CHECK 1 flagged documentation prose; the exclusion is broken.\n' "$RED" "$OFF"
    overall=1
  else
    printf '%sPASS%s — CHECK 1 flagged the tracked config and correctly ignored prose:\n' "$GREEN" "$OFF"
    printf '%s\n' "$planted_hits" | sed 's/^/    /'
  fi

  printf '\n%s== SELF-TEST RESULT ==%s\n' "$BOLD" "$OFF"
  if [ "$overall" -eq 0 ]; then
    printf '%sPASS%s — gate proven in BOTH directions (clean tree: PASS, planted literals: FAIL).\n' "$GREEN" "$OFF"
  else
    printf '%sFAIL%s — see above.\n' "$RED" "$OFF"
  fi
  # Guard against a vacuous self-test.
  if [ "${rc_clean:-1}" -eq 0 ] && [ "${rc_planted:-0}" -eq 0 ]; then
    printf '%sFAIL%s — self-test is vacuous: the gate never failed on the mutation.\n' "$RED" "$OFF"
    overall=1
  fi
  return $overall
}

case "${1:-}" in
  --self-test) self_test; exit $? ;;
  -h|--help)   sed -n '2,32p' "$0"; exit 0 ;;
  '')          run_gate; exit $? ;;
  *)           printf 'unknown option: %s (try --help)\n' "$1" >&2; exit 2 ;;
esac
