package commands

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// HXC-163 — the seven shared governance skills must be REGISTERED and
// INVOCABLE at runtime, not merely present on disk somewhere.
//
// # RUNTIME SIGNATURE (§11.4.108)
//
// The machine-checkable observable that proves this fix is BOTH active AND
// working — and which is NEVER a re-grep of the source — is:
//
//	a SkillLoader constructed with NO on-disk directories (built-in tier only,
//	exactly as a shipped binary sees the world) resolves all seven governance
//	skills, and for each one FindMatching() on a real user utterance returns
//	that skill and RenderWithCaptures() produces a non-empty body.
//
// That signature cannot pass unless the manifests actually ship inside the
// binary (embed.FS), actually parse, actually compile at least one trigger
// regex, and actually render. A manifest that is present but unresolvable,
// unparseable, trigger-less, or non-rendering FAILS here rather than being
// silently skipped — which is the specific failure mode HXC-163 describes.
//
// # POLARITY SWITCH — RED_MODE (§11.4.115, repo convention)
//
//   - RED_MODE unset / "0" (DEFAULT — the standing GREEN regression guard):
//     every governance skill MUST resolve, match and render. Run against the
//     PRE-FIX tree this FAILS (no governance manifest ships in the binary),
//     which is the captured proof the guard is not blind.
//
//   - RED_MODE=1 (defect reproduction / golden-bad harness): asserts the
//     PRE-FIX defect shape — that the governance skills are ABSENT from the
//     registry. Once the fix lands this branch FAILS, which is correct: the
//     defect is gone. It exists so the reproduction is re-runnable against an
//     old checkout and so the guard's own polarity is falsifiable.
//
// Run the RED reproduction:
//
//	RED_MODE=1 go test -count=1 -run TestGovernanceSkills ./internal/commands/
//
// Run the standing GREEN guard:
//
//	go test -count=1 -run TestGovernanceSkills ./internal/commands/
func governanceSkillsRedMode() bool { return os.Getenv("RED_MODE") == "1" }

// governanceSkillCase is one curated governance skill plus a real user
// utterance that MUST route to it.
//
// The set is deliberately SMALL and CURATED (seven entries). Registering an
// unbounded skill corpus measurably degrades agent selection accuracy through
// inter-skill shadowing, so this list is a closed set that is extended only by
// explicit decision — never by bulk import.
type governanceSkillCase struct {
	name  string // skill name == builtin_skills/<name>/ directory
	input string // an utterance that MUST select this skill via FindMatching
}

func governanceSkillCases() []governanceSkillCase {
	return []governanceSkillCase{
		{"action-prefix-system", "what does the BACKGROUND action prefix do"},
		{"media-validator", "validate the recording at /tmp/run.mp4"},
		{"multitrack", "how do I start a new multitrack work stream"},
		{"reporting-workable-items", "report a bug: the CLI drops skill triggers"},
		{"scheduled-work-queue", "remind me to re-check the nightly export"},
		{"session-sync", "sync my session from the remote host"},
		{"workable-item-lifecycle", "how do I close a workable item"},
	}
}

// TestGovernanceSkills_RegisteredAndInvocable is the HXC-163 runtime signature.
func TestGovernanceSkills_RegisteredAndInvocable(t *testing.T) {
	// No project dir, no user dir: the built-in tier ONLY. This is precisely
	// what a freshly installed binary sees, so a PASS here proves the skills
	// ship with the product rather than depending on a developer's checkout.
	reg := NewSkillRegistry()
	loader := NewSkillLoader(reg, "", "")
	require.NoError(t, loader.Load(), "built-in skill tier must load")

	cases := governanceSkillCases()

	if governanceSkillsRedMode() {
		// RED: reproduce the defect — none of the governance skills are
		// registered, so none can ever fire.
		var present []string
		for _, tc := range cases {
			if _, ok := reg.Get(tc.name); ok {
				present = append(present, tc.name)
			}
		}
		require.Empty(t, present,
			"RED_MODE=1 expected the governance skills to be ABSENT from the registry "+
				"(the HXC-163 defect: shipped on disk, never registered), but these resolved: %v. "+
				"If the fix has landed this failure is EXPECTED — run without RED_MODE for the standing guard.",
			present)
		return
	}

	// GREEN: every curated governance skill resolves, matches and renders.
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, ok := reg.Get(tc.name)
			require.True(t, ok,
				"governance skill %q is NOT registered: it can never be triggered by a user or an agent", tc.name)

			// Shipped inside the binary — not resolved from a stray on-disk
			// path that happens to exist on this machine. This is what catches
			// an entry that is "present" but unresolvable at runtime.
			assert.Equal(t, "builtin:"+tc.name+"/SKILL.md", s.SourcePath(),
				"governance skill %q must resolve from the embedded built-in tier", tc.name)

			// A skill with no description is invisible in `/skills list`.
			assert.NotEmpty(t, strings.TrimSpace(s.Description()),
				"governance skill %q must carry a description", tc.name)

			// Raw trigger patterns must exist...
			require.NotEmpty(t, s.TriggerPatterns(),
				"governance skill %q declares no triggers, so auto-trigger can never fire", tc.name)

			// ...and at least one must actually COMPILE and MATCH. Invalid
			// regexes are silently dropped at parse time, so a skill can
			// declare triggers yet be permanently unmatchable. Routing a real
			// utterance is the only assertion that catches that.
			matched, caps, found := reg.FindMatching(tc.input)
			require.True(t, found,
				"no skill matched %q; governance skill %q has no working trigger", tc.input, tc.name)
			require.Equal(t, tc.name, matched.Name(),
				"input %q routed to skill %q, expected %q (trigger collision or shadowing)",
				tc.input, matched.Name(), tc.name)

			// Invocable: this is the same render path `/skills invoke` uses.
			body, err := matched.RenderWithCaptures(nil, caps, "", "")
			require.NoError(t, err, "governance skill %q failed to render", tc.name)
			require.NotEmpty(t, strings.TrimSpace(body),
				"governance skill %q rendered an empty body", tc.name)
		})
	}
}

// TestGovernanceSkills_BuiltinSetIsCurated pins the built-in tier to a small,
// deliberate set.
//
// Registering an unbounded skill corpus measurably degrades agent selection
// accuracy — the loss comes from skills shadowing one another, and it grows
// with the number registered. Context/token overhead is NOT the driver, so the
// intuitive mitigations (shorter prose, lazy-loaded bodies) do not help. The
// only effective control is keeping the registered set small and deliberate.
//
// This guard fails if someone bulk-registers a large corpus into the binary.
func TestGovernanceSkills_BuiltinSetIsCurated(t *testing.T) {
	if governanceSkillsRedMode() {
		t.Skip("SKIP-OK: RED_MODE reproduces the absence defect; the curation bound is asserted by the GREEN guard")
	}
	reg := NewSkillRegistry()
	loader := NewSkillLoader(reg, "", "")
	require.NoError(t, loader.Load())

	const curatedCeiling = 16 // far below any measured degradation threshold
	got := len(reg.List())
	assert.LessOrEqual(t, got, curatedCeiling,
		"the built-in skill tier has grown to %d skills; keep it a small curated set "+
			"(inter-skill shadowing degrades selection accuracy as the registered count rises). "+
			"Raising this ceiling requires a deliberate decision, not a bulk import.", got)
}
