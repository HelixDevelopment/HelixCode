---
description: Validate a recording, screenshot or capture by reading its actual content and returning a PASS/FAIL verdict with evidence
triggers:
  - "(?i)^validate (?:the )?(?:recording|video|screenshot|capture|media|image|evidence)\\b.*?(?P<path>\\S+\\.(?:mp4|png|jpg|jpeg|txt|wav|cast))\\s*$"
  - "(?i)^(?:check|verify) (?:the )?(?:recording|video|screenshot|capture)\\b.*?(?P<path>\\S+\\.(?:mp4|png|jpg|jpeg|txt|wav|cast))\\s*$"
requires_isolation: false
---

You are validating a captured media artifact for a HelixCode user.

Artifact under validation: `{{ARG.path}}`

The governing rule is that a recording is evidence ONLY if its CONTENT has been
read and checked. Duration, file size and mere existence prove nothing: a long
recording of an empty terminal is worth less than a short one that shows the
feature working. Apply this discipline:

1. STATE THE EXPECTATION FIRST. Before inspecting the artifact, write down what
   content SHOULD appear in it — the expected output lines, test verdicts, API
   responses or UI text. A validation with no stated expectation cannot fail
   honestly, and so cannot pass honestly either.

2. EXTRACT THE ACTUAL CONTENT. Read what the artifact really contains — OCR or
   frame extraction for video and images, transcription for audio, direct text
   parsing for terminal captures and logs.

3. COMPARE, AND SAY WHICH PATTERNS MATCHED. Report each expected pattern as
   found or not found. Never summarise as "looks correct" without naming the
   evidence that made it correct.

4. SCAN FOR FALSE-SUCCESS CONTENT. Treat empty output, an error banner, an
   unstarted process, a spinner, or text that merely claims success without
   showing it as a FAIL, not a pass.

5. RETURN A VERDICT: PASS or FAIL, plus the artifact path, plus the matched and
   unmatched patterns. On FAIL, pinpoint where — the frame, timestamp or line.

6. IF THE ARTIFACT IS MISSING OR UNREADABLE, say so plainly and return neither
   PASS nor FAIL — an absent artifact is an absent result, never a pass.

The executable validator that performs OCR and pattern matching lives at
`constitution/skills/media-validator/media-validator.sh`; the full skill
contract is documented alongside it. Use that script rather than re-implementing
extraction by hand when it is available on this machine.
