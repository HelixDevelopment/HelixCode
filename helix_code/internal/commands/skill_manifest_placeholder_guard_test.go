package commands

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// HXC-177 — a skill manifest's {{...}} body placeholders must all be tokens
// the skill-render engine actually substitutes at runtime, never a form the
// engine's own substitution pass leaves untouched.
//
// # THE DEFECT THIS GUARDS AGAINST
//
// builtin_skills/conventional-commit/SKILL.md declared:
//
//	variables:
//	  spec_url: "https://www.conventionalcommits.org/en/v1.0.0/"
//
// and referenced it in the body as the bare token `{{spec_url}}`. The render
// pipeline is Skill.RenderWithCaptures -> MarkdownCommand.render (see
// markdown_commands.go), which walks the body with substRegex:
//
//	`\{\{([A-Z_][A-Z0-9_]*(?:\.[A-Za-z0-9_]+)?(?::[^}]+)?)\}\}`
//
// substRegex requires the token to START with an uppercase letter or
// underscore. "spec_url" starts with a lowercase 's', so `{{spec_url}}` does
// not match substRegex AT ALL — ReplaceAllStringFunc never invokes the
// resolver for it, and the literal text "{{spec_url}}" is sent to the LLM
// verbatim as prompt noise. The declared variable is only reachable through
// the `{{ARG.<name>}}` form (buildResolver's `strings.HasPrefix(token,
// "ARG.")` case), which looks the name up in the SAME merged map that both
// frontmatter `variables:` entries and RenderWithCaptures's regex-capture
// map populate (see markdown_skills.go RenderWithCaptures and
// SkillRegistry.FindMatching's named-capture extraction).
//
// This file mechanically enforces two rules derived from that resolution
// contract so the class of bug cannot recur silently:
//
//  1. Every `{{...}}` token in a shipped manifest's body must match
//     substRegex — otherwise the renderer leaves it untouched and it reaches
//     the LLM as literal, unsubstituted text.
//  2. Every `{{ARG.<name>}}` token's <name> must be a REAL substitution
//     source: either a frontmatter `variables:` key, or a named capture
//     group (`(?P<name>...)`) from one of the skill's own triggers. Anything
//     else resolves through buildResolver's `ARG.` case to c.variables[name]
//     on a MISSING key, which Go returns as "" — a silent, harder-to-notice
//     sibling of rule 1 (the token vanishes into an empty string instead of
//     surviving as visible literal text).
//
// # POLARITY SWITCH — RED_MODE (§11.4.115, repo convention; see
// governance_skills_registered_test.go and permissions_session_store_red_test.go
// for the same shape elsewhere in this package)
//
//   - RED_MODE unset / "0" (DEFAULT — the standing GREEN regression guard):
//     sweeps every manifest actually embedded in the shipped binary
//     (builtin_skills/*/SKILL.md via the real //go:embed FS) and asserts
//     NONE contains an unresolvable placeholder.
//
//   - RED_MODE=1: runs the SAME checker against a controlled fixture that
//     faithfully reproduces the HXC-177 defect shape (a frontmatter
//     `variables:` entry referenced as a bare `{{name}}` instead of
//     `{{ARG.name}}`) and asserts the checker actually flags it. This is the
//     guard's own paired-mutation proof (§1.1): a checker that cannot catch
//     the known-bad fixture would be a bluff gate.
//
// Run the RED reproduction:
//
//	RED_MODE=1 go test -count=1 -run TestSkillManifestPlaceholders ./internal/commands/
//
// Run the standing GREEN guard:
//
//	go test -count=1 -run TestSkillManifestPlaceholders ./internal/commands/
func skillManifestPlaceholderRedMode() bool { return os.Getenv("RED_MODE") == "1" }

// allPlaceholderTokensPattern matches EVERY "{{...}}" occurrence in a skill
// body, whether or not it is a form the renderer resolves. This is
// deliberately broader than substRegex (defined in markdown_commands.go) so
// it also catches tokens whose shape the renderer's own regex would reject
// outright — that rejection is exactly what leaves them as unsubstituted
// literal text (rule 1 above). Matching substRegex's own definition here
// would make the check blind to its own defect class.
var allPlaceholderTokensPattern = regexp.MustCompile(`\{\{([^{}]*)\}\}`)

// hxc177BadFixtureBody faithfully reproduces the pre-fix
// conventional-commit/SKILL.md shape: a frontmatter-declared variable
// referenced with the bare `{{name}}` form instead of the `{{ARG.name}}`
// form the engine requires.
const hxc177BadFixtureBody = `---
description: HXC-177 reproduction fixture
triggers:
  - "^fixture$"
variables:
  spec_url: "https://example.invalid/spec"
---

See {{spec_url}} for the authoritative specification.
`

// hxc177GoodFixtureBody is the same fixture with the fix applied, used to
// prove the checker does not false-positive on the corrected form.
const hxc177GoodFixtureBody = `---
description: HXC-177 reproduction fixture (fixed)
triggers:
  - "^fixture$"
variables:
  spec_url: "https://example.invalid/spec"
---

See {{ARG.spec_url}} for the authoritative specification.
`

// findManifestPlaceholderProblems parses raw as a skill manifest and returns
// one human-readable problem string per unresolvable {{...}} placeholder
// found in its body. An empty (nil) slice means every placeholder in the
// body is a token the render engine actually substitutes at runtime.
func findManifestPlaceholderProblems(name, raw string) ([]string, error) {
	s, err := parseSkillFile(name, raw, "test:"+name+"/SKILL.md")
	if err != nil {
		return nil, err
	}

	// Every legitimate ARG.<name> source: frontmatter `variables:` keys plus
	// every named capture group declared across the skill's OWN triggers —
	// exactly the two sources RenderWithCaptures merges before rendering
	// (see markdown_skills.go) and exactly what FindMatching populates its
	// captures map from (see markdown_skills.go SkillRegistry.FindMatching).
	declared := map[string]bool{}
	for k := range s.variables {
		declared[k] = true
	}
	for _, re := range s.triggers {
		for _, sub := range re.SubexpNames() {
			if sub != "" {
				declared[sub] = true
			}
		}
	}

	var problems []string
	for _, m := range allPlaceholderTokensPattern.FindAllStringSubmatch(s.Body(), -1) {
		full := "{{" + m[1] + "}}"
		token := m[1]

		// Rule 1: the token must match the engine's own substitution pattern
		// (substRegex, markdown_commands.go). If it does not, the renderer's
		// ReplaceAllStringFunc never invokes the resolver for it at all, and
		// the literal text is sent to the LLM verbatim — the exact HXC-177
		// defect.
		if !substRegex.MatchString(full) {
			problems = append(problems, fmt.Sprintf(
				"%s: placeholder %s does not match the engine's substitution pattern (%s) "+
					"and is sent to the LLM VERBATIM, never substituted",
				name, full, substRegex.String()))
			continue
		}

		// Rule 2: an ARG.<name> token must resolve to a real source. Anything
		// else falls through buildResolver's ARG. case to a missing map key,
		// which Go returns as "" — silently dropping the reference instead of
		// leaving it visibly wrong.
		if strings.HasPrefix(token, "ARG.") {
			varName := strings.TrimPrefix(token, "ARG.")
			if !declared[varName] {
				problems = append(problems, fmt.Sprintf(
					"%s: placeholder %s references variable %q, which is declared neither in "+
						"frontmatter `variables:` nor as a named capture group (?P<%s>...) in any "+
						"trigger — it silently renders as an empty string",
					name, full, varName, varName))
			}
		}
	}
	return problems, nil
}

// TestSkillManifestPlaceholders_NoUnresolvableTokens is the HXC-177 standing
// guard (GREEN) plus its own paired-mutation proof (RED_MODE=1).
func TestSkillManifestPlaceholders_NoUnresolvableTokens(t *testing.T) {
	if skillManifestPlaceholderRedMode() {
		// RED: prove the checker actually catches the HXC-177 defect shape on
		// a controlled fixture, rather than merely asserting the (already
		// fixed) shipped tree is clean.
		problems, err := findManifestPlaceholderProblems("hxc177-fixture", hxc177BadFixtureBody)
		require.NoError(t, err, "fixture must parse as a valid skill manifest")
		require.NotEmpty(t, problems,
			"RED_MODE=1 expected the checker to flag the unresolvable {{spec_url}} token in the "+
				"HXC-177 reproduction fixture, but it found nothing — the guard is blind")
		t.Logf("RED_MODE reproduced the defect: %v", problems)
		return
	}

	// GREEN: sweep every manifest actually embedded in the shipped binary —
	// the same tier a fresh install sees (see loadBuiltinSkills in
	// markdown_skills.go) — and assert none contains an unresolvable
	// placeholder.
	entries, err := fs.ReadDir(builtinSkillsFS, builtinSkillsRoot)
	require.NoError(t, err, "must be able to read the embedded builtin_skills tree")

	swept := 0
	found := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		manifestPath := path.Join(builtinSkillsRoot, name, skillManifestName)
		data, readErr := builtinSkillsFS.ReadFile(manifestPath)
		if readErr != nil {
			// Not every subdirectory of builtin_skills/ need contain a
			// SKILL.md (see loadBuiltinSkills); skip non-skill dirs exactly
			// as the loader does.
			continue
		}
		swept++
		t.Run(name, func(t *testing.T) {
			problems, err := findManifestPlaceholderProblems(name, string(data))
			require.NoError(t, err, "shipped manifest %s must parse", name)
			found += len(problems)
			assert.Empty(t, problems, "manifest %s has unresolvable placeholder(s):\n%s",
				name, strings.Join(problems, "\n"))
		})
	}

	require.Greater(t, swept, 0,
		"swept zero builtin skill manifests — the walk is broken and this guard would vacuously pass")
	// Report what was actually observed. A summary line that asserts success
	// unconditionally is itself a §11.4 false-success surface: it would print
	// "zero unresolvable placeholders" on the very run whose subtests failed.
	if found == 0 {
		t.Logf("swept %d builtin skill manifest(s), zero unresolvable placeholders", swept)
	} else {
		t.Logf("swept %d builtin skill manifest(s), found %d unresolvable placeholder(s) — see subtest failures above",
			swept, found)
	}
}

// TestFindManifestPlaceholderProblems_AcceptsEverySupportedForm proves the
// checker does not false-positive on any of the token forms the engine
// actually supports (see buildResolver in markdown_commands.go), so the
// standing guard above cannot be satisfied by weakening the checker instead
// of fixing a manifest.
func TestFindManifestPlaceholderProblems_AcceptsEverySupportedForm(t *testing.T) {
	body := `---
description: all supported forms
triggers:
  - "(?P<thing>.+) please"
variables:
  greeting: hello
---

Positional: {{ARG1}} {{ARG2}}
Named (frontmatter): {{ARG.greeting}}
Named (trigger capture): {{ARG.thing}}
Context: {{SELECTION}} {{CURRENT_FILE}} {{CWD}}
Env: {{ENV.PATH}}
File: {{FILE:/tmp/does-not-need-to-exist-for-this-check}}
`
	problems, err := findManifestPlaceholderProblems("all-forms", body)
	require.NoError(t, err)
	assert.Empty(t, problems, "checker false-positived on a supported token form: %v", problems)
}

// TestFindManifestPlaceholderProblems_FixedFixtureIsClean proves the exact
// fix applied to conventional-commit/SKILL.md — {{spec_url}} rewritten to
// {{ARG.spec_url}} — makes the checker's RED fixture pass, closing the loop
// between the RED reproduction above and the real fix.
func TestFindManifestPlaceholderProblems_FixedFixtureIsClean(t *testing.T) {
	problems, err := findManifestPlaceholderProblems("hxc177-fixture-fixed", hxc177GoodFixtureBody)
	require.NoError(t, err)
	assert.Empty(t, problems, "fixed fixture must have zero unresolvable placeholders: %v", problems)
}
