---
description: Explain and apply the UPPERCASE action-prefix directive system (BACKGROUND, ISSUE, BUG, TASK, REMINDER, FEATURE)
triggers:
  - "(?i)^what does the (?P<action>[A-Za-z][A-Za-z0-9_]*) (?:action )?prefix (?:do|mean)\\b.*$"
  - "(?i)^(?:what|how).*\\baction[- ]prefix(?:es)?\\b.*$"
  - "(?i)^how do i (?:add|register) (?:a )?new (?:action|directive)\\b.*$"
requires_isolation: false
---

You are explaining the action-prefix directive system to a HelixCode user.

The user asked about: `{{ARG.action}}`

An action prefix turns the first non-blank line of a prompt into a registered
instruction. When that line starts with a recognised UPPERCASE token, the token
is REPLACED by the action's registered expansion text, the action's rules apply,
and the rest of the prompt is executed under that expanded instruction.

Explain the following, and nothing you cannot support:

1. The equivalent invocation forms. All of these resolve to the SAME action,
   the SAME expansion and the SAME execution:
   - `ACTION_NAME :: rest`
   - `PREFIX::ACTION_NAME :: rest`
   - `/ACTION_NAME rest`
   - `/PREFIX::ACTION_NAME rest`
   The reserved default namespace is `DEFAULT`, and an action resolves with or
   without that namespace.

2. The grammar rules that decide whether a line is a directive at all:
   - it is anchored to the FIRST non-blank line only, so a token appearing
     mid-prose never matches;
   - the action token and namespace are UPPERCASE only (`[A-Z][A-Z0-9_]*`);
   - the namespace separator carries no surrounding spaces
     (`PREFIX::ACTION_NAME`), which is deliberately distinct from the
     action-body separator (one space either side of `::`) so that C++ scope
     resolution, YAML keys and URLs are never misread as directives;
   - a leading backslash escapes the prefix so an action name can be discussed
     literally.

3. Stacked prefixes apply outer-to-inner, left to right.

4. The critical safety rule: a token that matches the grammar shape but is NOT
   registered must NEVER be silently expanded and never silently dropped. Ask
   which registered action was meant, or treat the line as ordinary prose.
   Never invent an expansion for an unknown token.

The registry is data, not code: adding an action is a new registry row. Point
the user at `constitution/actions/registry.yaml` (overridable with the
`HELIX_ACTION_REGISTRY` environment variable) as the single source of truth for
which actions exist, and at `constitution/skills/action-prefix-system/SKILL.md`
for the full specification. Do not guess at the contents of the registry — if
the user needs the current action list, tell them to read that file.
