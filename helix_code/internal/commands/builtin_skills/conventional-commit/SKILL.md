---
description: Draft a Conventional Commits message from a short summary of the change
triggers:
  - "(?i)^commit message for (?P<summary>.+)$"
  - "(?i)^write a (?:conventional )?commit(?: message)? for (?P<summary>.+)$"
requires_isolation: false
variables:
  spec_url: "https://www.conventionalcommits.org/en/v1.0.0/"
---

You are an expert release engineer. Produce a single Conventional Commits 1.0.0
message for the following change:

    {{ARG.summary}}

Follow these rules exactly:

1. Header line: `<type>[optional scope]: <description>`
   - `type` is one of: feat, fix, docs, style, refactor, perf, test, build,
     ci, chore, revert.
   - `description` is imperative mood, lower-case, no trailing period, and at
     most 72 characters including the type and scope.
2. Leave one blank line, then an optional body that explains *what* changed and
   *why* (not *how*), wrapped at 72 columns.
3. If the change is backward-incompatible, add a `BREAKING CHANGE:` footer (or a
   `!` after the type/scope in the header) describing the migration impact.
4. Reference related issues in a footer such as `Refs: #123` when applicable.

Output only the commit message text, with no surrounding commentary, code
fences, or explanation. The authoritative specification is {{ARG.spec_url}}.
