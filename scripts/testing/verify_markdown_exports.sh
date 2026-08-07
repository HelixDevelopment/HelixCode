#!/usr/bin/env bash
# scripts/testing/verify_markdown_exports.sh
# §11.4.168 — CONTENT verification of exported .html/.pdf siblings.
#
# Presence is not proof. A zero-byte HTML, a pandoc error page, a 0-page PDF,
# or a PDF whose body text is raw Mermaid source all SATISFY a presence gate
# while lying to the reader. This verifier checks CONTENT.
#
# Per .md source given on stdin (one path per line) or as args, asserts:
#   HTML : exists, non-empty, has >= MIN_CHARS of non-tag body text,
#          is not a pandoc/weasyprint error page.
#   PDF  : exists, pdfinfo reports >= 1 page, pdftotext yields >= MIN_CHARS.
#   BOTH : no raw diagram SOURCE leaking as body text (§11.4.168) — a
#          diagram-header keyword at line-start (graph/flowchart/gantt/
#          sequenceDiagram/...) means mmdc did not render that block.
#
# §11.4.6 ANTI-FALSE-NULL: this script NEVER captures an exit status after a
# pipe, and NEVER uses `grep -q` on a pipe. `grep -q` exits on first match,
# SIGPIPEs the upstream producer (pdftotext), and under `pipefail` the
# pipeline returns 141 — so the caller's `if` never fires and EVERY file
# reports clean. All extraction is done into a shell variable first (command
# substitution reads to EOF), and all counting uses `grep -c`, which also
# reads to EOF. Proven by --self-test below.
#
# Modes:
#   (paths...)   verify the siblings of each .md path
#   --self-test  build known-good + known-bad control fixtures, assert the
#                verifier PASSes the good one and FAILs each bad one.
#                Exit 0 only if every control behaves as expected.
#
# Exit: 0 = every checked doc's siblings verified; 1 = at least one BAD.

set -uo pipefail

MIN_CHARS="${MIN_CHARS:-40}"

# Diagram-source tokens that must never appear at line-start in rendered
# output. A match means the fence shipped as text, not as an image.
DIAGRAM_RE='^[[:space:]]*(graph[[:space:]]+(TB|TD|BT|RL|LR)|flowchart[[:space:]]+(TB|TD|BT|RL|LR)|sequenceDiagram|classDiagram|stateDiagram(-v2)?|erDiagram|gantt|journey|mindmap|gitGraph|dateFormat[[:space:]]|pie[[:space:]]+title)'

# Pandoc/weasyprint failure surfaces that can land in an otherwise-present file.
ERRPAGE_RE='(pandoc: |Error at |YAML parse exception|WeasyPrint could not|Traceback \(most recent call last\))'

bad=0
good=0

# Strip tags from HTML and return body text in $STRIPPED (no pipes escaping).
html_text() {
  local f="$1" raw
  raw="$(cat -- "$f" 2>/dev/null)"
  # drop <head>...</head>, <script>, <style>, then all tags, then entities
  STRIPPED="$(printf '%s' "$raw" \
    | sed -e 's/<head>.*<\/head>//g' \
          -e 's/<script[^>]*>.*<\/script>//g' \
          -e 's/<style[^>]*>.*<\/style>//g' \
    | tr '\n' '\001' \
    | sed -e 's/<[^>]*>/ /g' \
    | tr '\001' '\n' \
    | sed -e 's/&[a-zA-Z#0-9]*;/ /g' -e 's/[[:space:]]\{1,\}/ /g')"
}

# Count regex matches in a string WITHOUT any exit-after-pipe hazard.
# grep -c consumes all input and is the pipeline tail; `|| true` absorbs the
# documented exit-1-on-no-match so the count (0) is the signal, not the code.
count_matches() {
  # $1 = text  $2 = ERE
  local n
  n="$(printf '%s\n' "$1" | grep -c -E "$2" || true)"
  printf '%s' "${n:-0}"
}

check_one() {
  local src="$1" base html pdf reasons=""
  base="${src%.md}"; html="${base}.html"; pdf="${base}.pdf"

  # ---- HTML ----
  if [ ! -f "$html" ]; then
    reasons="${reasons}html-missing;"
  elif [ ! -s "$html" ]; then
    reasons="${reasons}html-empty;"
  else
    html_text "$html"
    local hlen=${#STRIPPED}
    [ "$hlen" -lt "$MIN_CHARS" ] && reasons="${reasons}html-no-body-text(${hlen}c);"
    [ "$(count_matches "$STRIPPED" "$ERRPAGE_RE")" -gt 0 ] && reasons="${reasons}html-error-page;"
    [ "$(count_matches "$STRIPPED" "$DIAGRAM_RE")" -gt 0 ] && reasons="${reasons}html-raw-diagram-source;"
  fi

  # ---- PDF ----
  if [ ! -f "$pdf" ]; then
    reasons="${reasons}pdf-missing;"
  elif [ ! -s "$pdf" ]; then
    reasons="${reasons}pdf-empty;"
  else
    local pages ptxt plen
    pages="$(pdfinfo -- "$pdf" 2>/dev/null | sed -n 's/^Pages:[[:space:]]*//p')"
    case "$pages" in ''|*[!0-9]*) reasons="${reasons}pdf-unreadable;" ;;
      0) reasons="${reasons}pdf-zero-pages;" ;;
    esac
    ptxt="$(pdftotext -q -- "$pdf" - 2>/dev/null)"
    plen=${#ptxt}
    [ "$plen" -lt "$MIN_CHARS" ] && reasons="${reasons}pdf-no-extractable-text(${plen}c);"
    [ "$(count_matches "$ptxt" "$ERRPAGE_RE")" -gt 0 ] && reasons="${reasons}pdf-error-page;"
    [ "$(count_matches "$ptxt" "$DIAGRAM_RE")" -gt 0 ] && reasons="${reasons}pdf-raw-diagram-source;"
  fi

  if [ -n "$reasons" ]; then
    echo "BAD  $src  ${reasons}"
    bad=$((bad+1)); return 1
  fi
  good=$((good+1)); return 0
}

# ------------------------- self-test (control needles) -------------------
if [ "${1:-}" = "--self-test" ]; then
  T="$(mktemp -d)"; trap 'rm -rf "$T"' EXIT
  fails=0
  assert() { # $1=label $2=expected(OK|BAD) $3=src
    local out rc
    out="$(check_one "$3" 2>&1)"; rc=$?
    local actual=OK; [ "$rc" -ne 0 ] && actual=BAD
    if [ "$actual" = "$2" ]; then
      echo "  control PASS: $1 -> $actual  ${out}"
    else
      echo "  control FAIL: $1 expected $2 got $actual  ${out}"; fails=$((fails+1))
    fi
  }

  # --- known-GOOD: real prose in both siblings ---
  printf 'Control good doc\n\nThis paragraph exists so the body text comfortably exceeds the minimum character floor used by the verifier.\n' > "$T/good.md"
  pandoc "$T/good.md" -s -o "$T/good.html" 2>/dev/null
  weasyprint "$T/good.html" "$T/good.pdf" >/dev/null 2>&1
  assert "known-good doc" OK "$T/good.md"

  # --- known-BAD 1: HTML present but body is empty (the shape a broken
  #     render produces: valid tags, no readable content) ---
  cp "$T/good.md" "$T/emptybody.md"
  printf '<html><head><title>x</title></head><body></body></html>\n' > "$T/emptybody.html"
  cp "$T/good.pdf" "$T/emptybody.pdf"
  assert "empty-body HTML" BAD "$T/emptybody.md"

  # --- known-BAD 2: PDF carrying RAW MERMAID SOURCE as body text
  #     (the exact §11.4.168 defect: mmdc failed, fence shipped as text) ---
  printf 'Control bad diagram doc\n\nThis document deliberately ships raw diagram source as body text.\n\n    graph TB\n    A[Start] --> B[End]\n' > "$T/rawdiag.md"
  pandoc "$T/rawdiag.md" -s -o "$T/rawdiag.html" 2>/dev/null
  weasyprint "$T/rawdiag.html" "$T/rawdiag.pdf" >/dev/null 2>&1
  assert "raw-diagram-source PDF" BAD "$T/rawdiag.md"

  # --- known-BAD 3: zero-byte PDF ---
  cp "$T/good.md" "$T/zerobyte.md"; cp "$T/good.html" "$T/zerobyte.html"
  : > "$T/zerobyte.pdf"
  assert "zero-byte PDF" BAD "$T/zerobyte.md"

  # --- known-BAD 4: missing sibling entirely ---
  cp "$T/good.md" "$T/nosib.md"
  assert "missing siblings" BAD "$T/nosib.md"

  echo "self-test control failures: $fails"
  [ "$fails" -eq 0 ] || exit 1
  echo "SELF-TEST PASS: verifier provably detects good AND bad."
  exit 0
fi

# ------------------------- normal verification ---------------------------
if [ $# -gt 0 ]; then
  for f in "$@"; do check_one "$f" || true; done
else
  while IFS= read -r f; do [ -n "$f" ] && { check_one "$f" || true; }; done
fi

echo "verified_ok=$good verified_bad=$bad"
[ "$bad" -gt 0 ] && exit 1
exit 0
