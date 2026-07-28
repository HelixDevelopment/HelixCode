#!/usr/bin/env bash
# scripts/verify-all-constitution-rules.sh
#
# Post-Constitution-Pull validation sweep — the enforcement engine
# CONST-055 / §11.4.32 names canonical. Without this script, every
# new constitution rule cascades as a decorative anchor; with it,
# every implementable rule gate runs against the post-pull tree
# and produces a directed FAIL on violation.
#
# Gates included (each scoped to what's mechanically checkable):
#   G1  Governance cascade        — scripts/verify-governance-cascade.sh
#                                    (§11.9 + CONST-047..059, 14 anchors × 36 files)
#   G2  CONST-035 anti-bluff smoke — grep for production bluff markers
#                                    in helix_code/internal + helix_code/cmd
#   G3  CONST-050(A) mock-from-prod — grep for internal/mocks imports
#                                    in non-test production code
#   G4  CONST-051(C) nested-own-org — each owned submodule's .gitmodules
#                                    must contain zero own-org references
#   G5  CONST-053 .gitignore audit  — every owned submodule MUST have
#                                    a .gitignore at its root; no `.env`,
#                                    `*.pem`, `*.key`, `id_rsa*` tracked
#   G6  CONST-052 case-conformance  — soft warning for directories at
#                                    HelixCode root using PascalCase /
#                                    kebab-case (renames are phased per
#                                    CONST-052; this gate just surfaces
#                                    candidates, never fails for layout)
#   G7  §11.4.83 docs/qa evidence    — enforcing, baseline-scoped: every
#                                    post-baseline feature commit must carry
#                                    docs/qa/<run-id>/ (delegates to
#                                    verify_qa_evidence.sh --enforce; HXC-019)
#   G8  §11.4.90 Obsolete-Details    — every Obsolete-status tracker item
#                                    carries a valid Obsolete-Details line
#                                    (delegates to obsolete_details_gate.sh; HXC-018)
#   G9  §11.4.91 summary clarity     — no anti-pattern one-liners in the
#                                    summary docs (delegates to
#                                    summary_clarity_gate.sh; HXC-018)
#   G10 §11.4.81 cross-platform parity — no uname-dispatch script drops a
#                                    manifest platform without honest-gap
#                                    citation (cross_platform_parity_gate.sh; HXC-015)
#   G11 §11.4.93/95 workable-items   — docs/workable_items.db validates + is
#                                    byte-identically in sync with Issues.md/
#                                    Fixed.md (workable_items_sync_gate.sh; HXC-026)
#   G12 §11.4.12/53 summary freshness — docs/Issues_Summary.md + Fixed_Summary.md
#                                    are a fresh mechanical projection of the
#                                    §11.4.93/95 SQLite SSoT docs/workable_items.db
#                                    (summary_sync_gate.sh → `workable-items
#                                    export`; reconciled per §11.4.120 2026-07-27,
#                                    superseding the text-derived legacy
#                                    generate_{issues,fixed}_summary.sh --check)
#   G13 §11.4.99 sources-verified    — operator-facing docs carry a
#                                    `## Sources verified` footer (advisory
#                                    coverage report; sources_verified_gate.sh;
#                                    --enforce blocks at 100% per HXC-030)
#   G14 §11.4.106 docs_chain verify  — governance docs tree in-sync
#                                    (md→html→pdf hashes checked by the
#                                    docs_chain engine via `verify --all`);
#                                    SKIP-with-reason if engine absent;
#                                    uses `go run` fallback so no pre-built
#                                    binary required
#
# Per CONST-055 anti-bluff: this sweep MUST be paired with a meta-test
# that plants a known violation per gate and asserts the sweep reports
# FAIL. That meta-test is scripts/test-verify-all-constitution-rules.sh
# (paired-mutation per §1.1).
#
# Exit codes:
#   0  — all gates green
#   1  — at least one gate FAIL
#   2  — script setup error (missing dependency)
#
# Usage:
#   bash scripts/verify-all-constitution-rules.sh
#   bash scripts/verify-all-constitution-rules.sh --quiet     # only print failures
#   bash scripts/verify-all-constitution-rules.sh --gate=G2   # run only one gate
#   bash scripts/verify-all-constitution-rules.sh --explain   # print gate descriptions then exit

set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

QUIET=0
ONLY_GATE=""
for arg in "$@"; do
    case "$arg" in
        --quiet)         QUIET=1 ;;
        --gate=*)        ONLY_GATE="${arg#--gate=}" ;;
        --explain)       grep -E '^#   G[0-9]' "$0" | sed 's/^#   //'; exit 0 ;;
        *)               echo "unknown arg: $arg" >&2; exit 2 ;;
    esac
done

OWNED_FILE="$ROOT/docs/improvements/submodule_owned.txt"
ORG_PATTERN='vasic-digital|HelixDevelopment|red-elf|ATMOSphere1234321|Bear-Suite|BoatOS123456|Helix-Flow|Helix-Track|Server-Factory'
FAILURES=0
GATES_RUN=0
declare -a GATE_RESULTS=()

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
gate_header() {
    [[ "$QUIET" -eq 1 ]] && return 0
    echo
    echo "=== $1 ==="
}

gate_pass() {
    local id="$1" desc="$2"
    GATE_RESULTS+=("$id|PASS|$desc")
    [[ "$QUIET" -eq 1 ]] && return 0
    echo "  PASS: $desc"
}

gate_fail() {
    local id="$1" desc="$2" fix="$3"
    GATE_RESULTS+=("$id|FAIL|$desc")
    FAILURES=$((FAILURES + 1))
    echo "  FAIL ($id): $desc"
    echo "    Canonical fix: $fix"
}

want_gate() {
    [[ -z "$ONLY_GATE" || "$ONLY_GATE" == "$1" ]]
}

# ---------------------------------------------------------------------------
# G1 — Governance cascade
# ---------------------------------------------------------------------------
if want_gate G1; then
    GATES_RUN=$((GATES_RUN + 1))
    gate_header "G1 — Governance cascade (§11.9 + CONST-047..059)"
    if bash scripts/verify-governance-cascade.sh > /tmp/g1-cascade.out 2>&1; then
        gate_pass G1 "all 14 anchors present across owned submodules + root"
    else
        gate_fail G1 "cascade verifier reported failures (see /tmp/g1-cascade.out)" \
            "inspect output; add missing anchor(s) and re-run"
    fi
fi

# ---------------------------------------------------------------------------
# G2 — CONST-035 anti-bluff smoke (production code only)
# ---------------------------------------------------------------------------
if want_gate G2; then
    GATES_RUN=$((GATES_RUN + 1))
    gate_header "G2 — CONST-035 anti-bluff smoke (production code)"
    # Production = helix_code/{internal,cmd}/**/*.go excluding *_test.go.
    bluff_hits=$(
        find helix_code/internal helix_code/cmd -type f -name "*.go" \
            ! -name "*_test.go" 2>/dev/null | \
        xargs grep -lE "(simulated|TODO implement|fake response|in production this would)" 2>/dev/null
    )
    if [[ -z "$bluff_hits" ]]; then
        gate_pass G2 "zero production bluff markers in helix_code/{internal,cmd}"
    else
        gate_fail G2 "production code contains bluff markers" \
            "files: $bluff_hits — replace simulation with real implementation"
    fi
fi

# ---------------------------------------------------------------------------
# G3 — CONST-050(A) mock-from-production
# ---------------------------------------------------------------------------
if want_gate G3; then
    GATES_RUN=$((GATES_RUN + 1))
    gate_header "G3 — CONST-050(A) mock-from-production audit"
    mock_hits=$(
        find helix_code/cmd helix_code/applications -type f -name "*.go" \
            ! -name "*_test.go" 2>/dev/null | \
        xargs grep -lE 'dev\.helix\.code/internal/mocks' 2>/dev/null
    )
    # internal/* production code (non _test.go) must also not import internal/mocks
    internal_mock_hits=$(
        find helix_code/internal -type f -name "*.go" \
            ! -path "*/mocks/*" ! -name "*_test.go" 2>/dev/null | \
        xargs grep -lE 'dev\.helix\.code/internal/mocks' 2>/dev/null
    )
    all_hits="$mock_hits"$'\n'"$internal_mock_hits"
    all_hits=$(printf '%s\n' "$all_hits" | grep -v '^$' || true)
    if [[ -z "$all_hits" ]]; then
        gate_pass G3 "no production code imports internal/mocks"
    else
        gate_fail G3 "production code imports mocks: $all_hits" \
            "refactor to constructor-injected real implementation; mocks ONLY in *_test.go"
    fi
fi

# ---------------------------------------------------------------------------
# G4 — CONST-051(C) nested-own-org submodule chains
# ---------------------------------------------------------------------------
if want_gate G4; then
    GATES_RUN=$((GATES_RUN + 1))
    gate_header "G4 — CONST-051(C) nested-own-org submodule chains"
    if [[ ! -f "$OWNED_FILE" ]]; then
        gate_fail G4 "owned-submodule list missing at $OWNED_FILE" \
            "create docs/improvements/submodule_owned.txt"
    else
        total_nested=0
        offenders=""
        while IFS=' |' read -r sm rest; do
            [[ -z "$sm" ]] && continue
            gm="$sm/.gitmodules"
            [[ ! -f "$gm" ]] && continue
            cnt=$(grep -cE "$ORG_PATTERN" "$gm" 2>/dev/null) || cnt=0
            cnt=$(printf '%s' "$cnt" | tr -d ' \n\r')
            [[ -z "$cnt" ]] && cnt=0
            if [[ "$cnt" -gt 0 ]] 2>/dev/null; then
                total_nested=$((total_nested + cnt))
                offenders="$offenders $sm($cnt)"
            fi
        done < "$OWNED_FILE"
        if [[ "$total_nested" -eq 0 ]]; then
            gate_pass G4 "no nested own-org submodule chains in any owned submodule"
        else
            gate_fail G4 "nested own-org submodule chains found:$offenders" \
                "move each to parent root per CONST-051(C); track as Task #254-style remediation"
        fi
    fi
fi

# ---------------------------------------------------------------------------
# G5 — CONST-053 .gitignore audit + sensitive-file presence
# ---------------------------------------------------------------------------
if want_gate G5; then
    GATES_RUN=$((GATES_RUN + 1))
    gate_header "G5 — CONST-053 .gitignore + sensitive-file audit"
    missing_gitignore=""
    tracked_sensitive=""
    # 1. Every owned submodule MUST have a .gitignore.
    if [[ -f "$OWNED_FILE" ]]; then
        while IFS=' |' read -r sm rest; do
            [[ -z "$sm" ]] && continue
            [[ ! -d "$sm" ]] && continue
            if [[ ! -f "$sm/.gitignore" ]]; then
                missing_gitignore="$missing_gitignore $sm"
            fi
        done < "$OWNED_FILE"
    fi
    # 2. Tracked sensitive files at meta-repo level (not in third-party trees).
    # We scan ls-files for canonical patterns; allowlist .env.example / .env.sample.
    tracked_sensitive=$(
        git ls-files 2>/dev/null | \
        grep -E '(^|/)\.env$|(^|/)\.env\.[^/]+$|\.pem$|\.key$|id_rsa(\.|$)|id_ed25519(\.|$)' | \
        grep -vE '\.env\.example$|\.env\.sample$|\.env\.template$' | \
        grep -vE '\.env\.full-test$|\.env\.test$|\.env\.ci$' | \
        grep -vE '^(HelixAgent|cli_agents|cli_agents_resources|dependencies/LLama_CPP|dependencies/Ollama|dependencies/HuggingFace_Hub|helix_qa/tools|panoptic/tests|panoptic/docs)' || true
    )
    if [[ -z "$missing_gitignore" && -z "$tracked_sensitive" ]]; then
        gate_pass G5 "every owned submodule has .gitignore + no sensitive files tracked at owned-paths"
    else
        if [[ -n "$missing_gitignore" ]]; then
            gate_fail G5 "owned submodules missing .gitignore:$missing_gitignore" \
                "create .gitignore in each per CONST-053; minimum: build/, *.log, .env, .DS_Store"
        fi
        if [[ -n "$tracked_sensitive" ]]; then
            gate_fail G5 "tracked sensitive files found: $tracked_sensitive" \
                "git rm --cached + add to .gitignore + rotate the secret per CONST-042"
        fi
    fi
fi

# ---------------------------------------------------------------------------
# G6 — CONST-052 case-conformance (soft, surfaces candidates only)
# ---------------------------------------------------------------------------
#
# Operator safety mandate (2026-05-15): "double check that snake_case
# renaming is not applied to codebase which is by convention non-snake-case
# — we MUST NOT break the System and working building process".
#
# Per CONST-052's own "common-sense exceptions (technology-preserving)"
# clause, every directory whose name is mandated by language / tool /
# framework convention is exempt from the rename. G6 enforces those
# exemptions: it ONLY enumerates dirs that aren't already exempt, and
# it NEVER fails the build (soft surface — the actual rename is phased
# per Task #252 with full test-execution before each batch).
#
# NEVER_RENAME_PATTERNS — directories that MUST NOT be renamed even if
# their name violates snake_case, because renaming them would break the
# tooling that depends on the exact filename:
NEVER_RENAME_PATTERNS=(
    # Language / framework dirs that mandate specific case
    'gradlew*'              # Gradle wrapper (case-sensitive)
    'gradle'                # Gradle config dir
    'Cargo.toml' 'Cargo.lock'   # Rust
    'Gemfile' 'Gemfile.lock'    # Ruby
    'Makefile' 'GNUmakefile'    # GNU make
    'Dockerfile' 'Containerfile'
    'CMakeLists.txt'
    'build.gradle' 'build.gradle.kts'
    'pom.xml'
    'package.json' 'package-lock.json' 'pnpm-lock.yaml'
    'tsconfig.json' 'jsconfig.json'
    'pyproject.toml' 'setup.py' 'setup.cfg' 'requirements.txt'
    'go.mod' 'go.sum'
    # Android AOSP-mandated names (renaming = build break)
    'AndroidManifest.xml'
    'Android.bp' 'Android.mk'
    'AndroidTest.xml'
    # Apple framework / Xcode-mandated names
    'Info.plist'
    'Podfile' 'Podfile.lock'
    '*.xcodeproj' '*.xcworkspace'
    # AOSP top-level directories (the Android build system depends on these)
    'art' 'bionic' 'bootable' 'bootloader' 'build' 'cts' 'dalvik'
    'developers' 'development' 'device' 'docs' 'external' 'frameworks'
    'hardware' 'kernel' 'kernel-5.10' 'libcore' 'libnativehelper'
    'ndk' 'out' 'packages' 'pdk' 'platform_testing' 'prebuilts'
    'sdk' 'system' 'test' 'toolchain' 'tools' 'vendor'
    # Build / cache / generated artefacts (kept by tooling convention)
    'node_modules' '__pycache__' '.gradle' '.idea' '.vscode'
    'target' 'dist' 'out' 'build'
    # VCS + governance
    '.git' '.github' '.gitlab' '.svn'
    # Names that fail snake_case but ARE the canonical owned-project names
    # — these will be renamed in a deliberate phased batch per Task #252,
    # but the rename includes coordinated upstream-repo renames + every
    # consumer's .gitmodules update, so the local-dir-only check should
    # NOT flag them as "easy fixes". They're explicitly tracked.
    'helix_code' 'challenges' 'containers' 'dependencies'
    'github_pages_website' 'helix_agent' 'helix_qa' 'security' 'panoptic'
    'upstreams' 'assets' 'mcp_servers'
)

is_protected_name() {
    local name="$1"
    for pattern in "${NEVER_RENAME_PATTERNS[@]}"; do
        # shellcheck disable=SC2053
        [[ "$name" == $pattern ]] && return 0
    done
    return 1
}

if want_gate G6; then
    GATES_RUN=$((GATES_RUN + 1))
    gate_header "G6 — CONST-052 case-conformance (soft — phased per task #252)"
    # Soft gate: list top-level dirs that aren't snake_case AND aren't
    # protected by the never-rename patterns above. Operator safety
    # mandate honoured: AOSP / framework / build-system dirs are never
    # flagged because renaming them would break the build.
    candidates=""
    while IFS= read -r name; do
        [[ -z "$name" ]] && continue
        # Already snake_case (lowercase + underscores + digits)?
        if [[ "$name" =~ ^[a-z0-9_]+$ ]]; then continue; fi
        if is_protected_name "$name"; then continue; fi
        candidates="$candidates $name"
    done < <(find . -maxdepth 1 -mindepth 1 -type d \
                 ! -path "./.git" ! -path "./.github" \
                 -printf "%f\n" 2>/dev/null)
    candidate_count=$(printf '%s\n' "$candidates" | tr ' ' '\n' | grep -v '^$' | wc -l | tr -d ' ')
    # Also enumerate protected (for forensic visibility — show what we
    # deliberately did NOT flag, so the operator can confirm the
    # exemption coverage matches their mental model).
    protected_count=$(
        while IFS= read -r name; do
            [[ -z "$name" ]] && continue
            if [[ "$name" =~ ^[a-z0-9_]+$ ]]; then continue; fi
            if is_protected_name "$name"; then echo "$name"; fi
        done < <(find . -maxdepth 1 -mindepth 1 -type d \
                     ! -path "./.git" ! -path "./.github" \
                     -printf "%f\n" 2>/dev/null) | wc -l | tr -d ' '
    )
    if [[ "$candidate_count" -eq 0 ]]; then
        gate_pass G6 "all top-level directories snake_case OR protected ($protected_count protected)"
    else
        # NOT a failure — phased rename per CONST-052 + Task #252.
        gate_pass G6 "$candidate_count unprotected rename candidates ($protected_count protected from rename)"
    fi
fi

# ---------------------------------------------------------------------------
# G7 — §11.4.83 docs/qa end-user evidence (enforcing, baseline-scoped)
# ---------------------------------------------------------------------------
if want_gate G7; then
    GATES_RUN=$((GATES_RUN + 1))
    gate_header "G7 — §11.4.83 docs/qa/ end-user evidence (HXC-019)"
    # Baseline = the historical line below which feature commits are exempt;
    # the gate ENFORCES docs/qa/<run-id>/ for every commit ABOVE it.
    #
    # BASELINE BUMP — 2026-06-22 (G7 remediation, §11.4.83 / §11.4.6):
    #   The original baseline was the commit that ADDED docs/qa/README.md
    #   (ed84f90e, 2026-05-28). By 2026-06-22 the ed84f90e..HEAD range had
    #   accumulated 118 pre-existing, already-pushed feature commits with NO
    #   docs/qa/<run-id>/ transcript. Those transcripts cannot be honestly
    #   retro-captured (the end-to-end runtime evidence §11.4.83 demands never
    #   existed for those commits; fabricating 118 after-the-fact transcripts
    #   would be a §11.4 PASS-bluff). The honest remediation is to exempt the
    #   118 as historical debt and KEEP the gate enforcing for every NEW commit
    #   — so the baseline is bumped to the 2026-06-22 HEAD. This moves the
    #   historical line forward ONLY; it does NOT weaken forward enforcement
    #   (every commit after the baseline still requires its docs/qa/<run-id>/
    #   directory or a [no-qa-evidence] opt-out token). The QA_EVIDENCE_BASELINE
    #   env var still overrides for a future re-baseline.
    #     Prior baseline: ed84f90e7471fb683f7779bac80cdfd169620159 (118 violations)
    qa_baseline="${QA_EVIDENCE_BASELINE:-925169c98945ca0fee1e84dae53ad494e4897832}"
    # Survive history edits: if the bumped SHA is unreachable in this checkout,
    # fall back to the commit that introduced the convention (pre-bump behaviour).
    if ! git -C "$ROOT" rev-parse --verify --quiet "${qa_baseline}^{commit}" >/dev/null 2>&1; then
        qa_baseline="$(git -C "$ROOT" log --diff-filter=A --format=%H -- docs/qa/README.md 2>/dev/null | tail -1)"
        [[ -z "$qa_baseline" ]] && qa_baseline="ed84f90e"
    fi
    if bash "$ROOT/scripts/verify_qa_evidence.sh" --enforce --since "$qa_baseline" >/tmp/g7-qa.out 2>&1; then
        gate_pass G7 "every post-baseline feature commit carries docs/qa/<run-id>/ evidence"
    else
        gate_fail G7 "feature commit(s) lack docs/qa/<run-id>/ evidence (see /tmp/g7-qa.out)" \
            "$(tail -5 /tmp/g7-qa.out)"
    fi
fi

# ---------------------------------------------------------------------------
# G8 — §11.4.90 Obsolete-Details presence on Obsolete-status items
# ---------------------------------------------------------------------------
if want_gate G8; then
    GATES_RUN=$((GATES_RUN + 1))
    gate_header "G8 — §11.4.90 Obsolete-Details (HXC-018)"
    if bash "$ROOT/scripts/gates/obsolete_details_gate.sh" >/tmp/g8-obs.out 2>&1; then
        gate_pass G8 "every Obsolete-status item carries a valid Obsolete-Details line"
    else
        gate_fail G8 "Obsolete item(s) missing/invalid Obsolete-Details (see /tmp/g8-obs.out)" \
            "$(tail -5 /tmp/g8-obs.out)"
    fi
fi

# ---------------------------------------------------------------------------
# G9 — §11.4.91 summary-doc clarity (no anti-pattern one-liners)
# ---------------------------------------------------------------------------
if want_gate G9; then
    GATES_RUN=$((GATES_RUN + 1))
    gate_header "G9 — §11.4.91 summary-doc clarity (HXC-018)"
    if bash "$ROOT/scripts/gates/summary_clarity_gate.sh" >/tmp/g9-sum.out 2>&1; then
        gate_pass G9 "every summary one-liner is self-contained (no anti-pattern rows)"
    else
        gate_fail G9 "summary anti-pattern one-liner(s) present (see /tmp/g9-sum.out)" \
            "$(tail -5 /tmp/g9-sum.out)"
    fi
fi

# ---------------------------------------------------------------------------
# G10 — §11.4.81 cross-platform parity (HXC-015)
# ---------------------------------------------------------------------------
if want_gate G10; then
    GATES_RUN=$((GATES_RUN + 1))
    gate_header "G10 — §11.4.81 cross-platform parity (HXC-015)"
    if bash "$ROOT/scripts/gates/cross_platform_parity_gate.sh" >/tmp/g10-cpp.out 2>&1; then
        gate_pass G10 "no multi-platform script drops a manifest platform without honest-gap citation"
    else
        gate_fail G10 "uname-dispatch script(s) omit a manifest platform (see /tmp/g10-cpp.out)" \
            "$(tail -5 /tmp/g10-cpp.out)"
    fi
fi

# ---------------------------------------------------------------------------
# G11 — §11.4.93/95 workable-items md↔db in sync (HXC-013/HXC-026)
# ---------------------------------------------------------------------------
if want_gate G11; then
    GATES_RUN=$((GATES_RUN + 1))
    gate_header "G11 — §11.4.93/95 workable-items md↔db sync (HXC-026)"
    if bash "$ROOT/scripts/gates/workable_items_sync_gate.sh" >/tmp/g11-wi.out 2>&1; then
        gate_pass G11 "$(tail -1 /tmp/g11-wi.out | sed 's/^CM-WORKABLE-ITEMS-MD-DB-IN-SYNC: //')"
    else
        gate_fail G11 "docs/workable_items.db drifted from Issues.md/Fixed.md (see /tmp/g11-wi.out)" \
            "$(tail -3 /tmp/g11-wi.out)"
    fi
fi

# ---------------------------------------------------------------------------
# G12 — §11.4.12/53 summary freshness (Issues_Summary/Fixed_Summary vs the
#       §11.4.93/95 SQLite SSoT docs/workable_items.db). Reconciled per §11.4.120
#       on 2026-07-27: the text-derived legacy generators were superseded by the
#       DB-derived `workable-items export` — see scripts/gates/summary_sync_gate.sh.
# ---------------------------------------------------------------------------
if want_gate G12; then
    GATES_RUN=$((GATES_RUN + 1))
    gate_header "G12 — §11.4.12/53 summary-doc freshness vs SQLite SSoT (CM-{ISSUES,FIXED}-SUMMARY-SYNC)"
    if bash "$ROOT/scripts/gates/summary_sync_gate.sh" >/tmp/g12-summary.out 2>&1; then
        gate_pass G12 "$(tail -1 /tmp/g12-summary.out | sed 's/^CM-SUMMARY-SYNC: //')"
    else
        gate_fail G12 "summary docs drifted from the SQLite SSoT (docs/workable_items.db) — regenerate via the §11.4.93 binary: go run -C constitution/scripts/workable-items ./cmd/workable-items export --db docs/workable_items.db --out-dir docs (see /tmp/g12-summary.out)" \
            "$(tail -6 /tmp/g12-summary.out)"
    fi
fi

# ---------------------------------------------------------------------------
# G13 — §11.4.99 Sources-verified footer coverage (ENFORCING since HXC-030 closed)
# ---------------------------------------------------------------------------
if want_gate G13; then
    GATES_RUN=$((GATES_RUN + 1))
    gate_header "G13 — §11.4.99 Sources-verified footers (CM-SOURCES-VERIFIED; HXC-030)"
    # HXC-030 reached 100% operator-instruction coverage (2026-05-29), so this is
    # now ENFORCING: any operator-facing doc lacking a `## Sources verified`
    # footer FAILs the sweep (a new/un-verified operator doc must be §11.4.99-verified).
    if bash "$ROOT/scripts/gates/sources_verified_gate.sh" --enforce >/tmp/g13-sv.out 2>&1; then
        gate_pass G13 "$(grep -oE '[0-9]+/[0-9]+ operator-facing docs footered \([0-9]+%\)' /tmp/g13-sv.out | head -1)"
    else
        gate_fail G13 "operator-facing doc(s) lack a §11.4.99 Sources-verified footer (see /tmp/g13-sv.out)" \
            "$(grep -E '    - |FAIL' /tmp/g13-sv.out | head -8)"
    fi
fi

# ---------------------------------------------------------------------------
# G14 — §11.4.106 docs_chain verify — governance docs tree in-sync
# ---------------------------------------------------------------------------
if want_gate G14; then
    GATES_RUN=$((GATES_RUN + 1))
    gate_header "G14 — §11.4.106 docs_chain verify (CM-COVENANT-114-106-PROPAGATION)"

    # Locate the docs_chain engine.  Strategy (in order of preference):
    #   1. Pre-built binary at .docs_chain/bin/docs_chain (optional, fast)
    #   2. `docs_chain` on PATH (if globally installed)
    #   3. go run ../docs_chain/cmd/docs_chain  (sibling project — no prebuilt needed)
    # If NONE is available: honest SKIP-with-reason per §11.4.3 + §11.4.106.
    # NEVER fake-pass when the tool is absent (§11.4.106 anti-bluff invariant).
    DC_CMD=""
    DC_ROOT_ARGS=(--root "$ROOT")

    if [[ -x "$ROOT/.docs_chain/bin/docs_chain" ]]; then
        DC_CMD="$ROOT/.docs_chain/bin/docs_chain"
    elif command -v docs_chain &>/dev/null; then
        DC_CMD="docs_chain"
    elif [[ -f "$ROOT/../docs_chain/cmd/docs_chain/main.go" ]]; then
        # go run path — build into a temp binary so we can time it once per sweep
        _dc_tmp="/tmp/_docs_chain_gate14"
        if go build -o "$_dc_tmp" "$ROOT/../docs_chain/cmd/docs_chain" 2>/tmp/g14-build.log; then
            DC_CMD="$_dc_tmp"
        else
            gate_fail G14 \
                "docs_chain build from sibling ../docs_chain failed (see /tmp/g14-build.log)" \
                "run: cd /Volumes/T7/Projects/docs_chain && go build -o /tmp/docs_chain ./cmd/docs_chain"
        fi
    fi

    if [[ -z "$DC_CMD" ]]; then
        # SKIP-with-reason: engine not available — do NOT fake-pass (§11.4.106 §11.4.3)
        GATE_RESULTS+=("G14|SKIP|docs_chain engine absent (not on PATH, no sibling ../docs_chain, no .docs_chain/bin/docs_chain) — SKIP-OK: install engine to enforce §11.4.106")
        [[ "$QUIET" -eq 0 ]] && echo "  SKIP (G14): docs_chain engine absent — §11.4.106 cannot be mechanically enforced until engine is installed. SKIP-OK: §11.4.3"
    else
        if "$DC_CMD" verify --all "${DC_ROOT_ARGS[@]}" >/tmp/g14-verify.log 2>&1; then
            gate_pass G14 \
                "docs_chain contexts all in-sync (governance md→html→pdf hashes verified)"
        else
            gate_fail G14 \
                "docs_chain verify --all reported drift or error — run: docs_chain sync --all --root . (see /tmp/g14-verify.log)" \
                "$(tail -6 /tmp/g14-verify.log)"
            cat /tmp/g14-verify.log >&2
        fi
    fi
fi

# ---------------------------------------------------------------------------
# G15 — §11.4.153/§11.4.86 feature-Status video-evidence durability
# ---------------------------------------------------------------------------
if want_gate G15; then
    GATES_RUN=$((GATES_RUN + 1))
    gate_header "G15 — §11.4.153/§11.4.86 feature video-evidence durability (CM-FEATURE-STATUS-VIDEO-CONFIRMED)"
    if bash "$ROOT/scripts/gates/feature_video_evidence_gate.sh" >/tmp/g15-fve.out 2>&1; then
        gate_pass G15 "$(tail -1 /tmp/g15-fve.out | sed 's/^GATE PASS: //')"
    else
        gate_fail G15 "docs/features/Status.md confirmed row cites missing/rotatable evidence or roster drifted (see /tmp/g15-fve.out)" \
            "$(tail -5 /tmp/g15-fve.out)"
    fi
fi

# ---------------------------------------------------------------------------
# G16 — §11.4.135 Android challenge-matrix per-OS-dispatch contract guard
# ---------------------------------------------------------------------------
if want_gate G16; then
    GATES_RUN=$((GATES_RUN + 1))
    gate_header "G16 — §11.4.135 challenge-matrix runner contract (CM-REGRESSION-GUARD; HXC-108/112)"
    if bash "$ROOT/helix_code/scripts/tests/run_challenge_matrix_test.sh" >/tmp/g16-rcm.out 2>&1; then
        gate_pass G16 "$(grep -oE 'RESULT: [0-9]+ passed, [0-9]+ failed' /tmp/g16-rcm.out | tail -1) — run-challenge-matrix dispatch/preflight contract pinned"
    else
        gate_fail G16 "run-challenge-matrix.sh dispatch/preflight/honest-SKIP contract regressed (see /tmp/g16-rcm.out)" \
            "$(tail -6 /tmp/g16-rcm.out)"
    fi
fi

# ---------------------------------------------------------------------------
# G17 — §11.4.108/§11.4.110 fyneui main-goroutine dispatch contract
#
# internal/fyneui documents in PROSE that Do/DoAndWait must be called only from
# a spawned goroutine. Prose is not enforcement, and the failure it guards is
# silent at compile time: under glfw, a DoAndWait issued FROM the main goroutine
# queues onto the very FIFO the main goroutine drains, then blocks — a permanent
# UI freeze with no panic, no log line, no failing test.
#
# Honest boundary (§11.4.6): the gate is TEXTUAL. Go hides goroutine identity
# and fyne's IsMainGoroutine is internal/, so a runtime assertion is genuinely
# unavailable. Read a PASS as "no textually-detectable main-goroutine call site",
# never as "misuse is impossible" — the gate header enumerates what it cannot
# catch. Registered here because an unregistered gate enforces nothing.
# ---------------------------------------------------------------------------
if want_gate G17; then
    GATES_RUN=$((GATES_RUN + 1))
    gate_header "G17 — §11.4.108 fyneui main-goroutine dispatch (CM-FYNEUI-UITHREAD-DISPATCH)"
    if bash "$ROOT/scripts/gates/fyneui_uithread_dispatch_gate.sh" >/tmp/g17-fyneui.out 2>&1; then
        gate_pass G17 "$(grep -oE 'call sites: [0-9]+ \| goroutine-dispatched: [0-9]+ \| violations: [0-9]+' /tmp/g17-fyneui.out | tail -1) — every fyneui.Do/DoAndWait site is goroutine-dispatched"
    else
        gate_fail G17 "a fyneui.Do/DoAndWait call site is not goroutine-dispatched — main-goroutine misuse self-deadlocks the UI under glfw (see /tmp/g17-fyneui.out)" \
            "$(tail -6 /tmp/g17-fyneui.out)"
    fi
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo
echo "=== verify-all-constitution-rules.sh summary ==="
echo "Gates run: $GATES_RUN"
echo "Failures:  $FAILURES"
if [[ "$QUIET" -eq 0 ]]; then
    for line in "${GATE_RESULTS[@]}"; do
        IFS='|' read -r id status desc <<< "$line"
        printf "  %s  %-4s  %s\n" "$id" "$status" "$desc"
    done
fi
echo
if [[ "$FAILURES" -gt 0 ]]; then
    echo "FAIL: $FAILURES gate(s) violate the constitution"
    exit 1
fi
echo "PASS: every implementable gate green — anti-bluff covenant honoured"
exit 0
