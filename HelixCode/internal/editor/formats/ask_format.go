package formats

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// AskFormat handles question/confirmation mode
type AskFormat struct{}

// NewAskFormat creates a new ask format handler
func NewAskFormat() *AskFormat {
	return &AskFormat{}
}

// Type returns the format type
func (askf *AskFormat) Type() FormatType {
	return FormatTypeAsk
}

// Name returns the human-readable name
func (askf *AskFormat) Name() string {
	return "Ask Mode"
}

// Description returns the format description
func (askf *AskFormat) Description() string {
	return "Question/confirmation mode for clarifying changes before applying"
}

// CanHandle checks if this format can handle the given content
func (askf *AskFormat) CanHandle(content string) bool {
	// Look for ask-mode markers
	markers := []string{
		"QUESTION:",
		"CLARIFICATION:",
		"CONFIRM:",
		"PROPOSED CHANGE:",
		"question:",
		"clarification:",
		"confirm:",
		"Should I",
		"Would you like",
		"Do you want",
	}

	markerCount := 0
	for _, marker := range markers {
		if strings.Contains(content, marker) {
			markerCount++
		}
	}

	// Require at least 1 marker and a question mark
	return markerCount >= 1 && strings.Contains(content, "?")
}

// Parse parses the edit content and returns structured edits
func (askf *AskFormat) Parse(ctx context.Context, content string) ([]*FileEdit, error) {
	edits := make([]*FileEdit, 0)

	// Parse questions and proposed changes
	questions := askf.parseQuestions(content)
	proposals := askf.parseProposals(content)

	if len(questions) == 0 && len(proposals) == 0 {
		return nil, fmt.Errorf("no questions or proposals found in ask format")
	}

	// Create edit with questions and proposals in metadata
	// This doesn't represent actual changes yet, just proposed changes
	edit := &FileEdit{
		FilePath:  "", // Will be filled from context
		Operation: EditOperationUpdate,
		Metadata: map[string]interface{}{
			"format":    "ask",
			"questions": questions,
			"proposals": proposals,
			"status":    "pending_confirmation",
		},
	}

	edits = append(edits, edit)

	return edits, nil
}

// Question represents a clarification question
type Question struct {
	FilePath    string
	QuestionText string
	Context     string
	Options     []string
}

// Proposal represents a proposed change
type Proposal struct {
	FilePath    string
	Description string
	OldContent  string
	NewContent  string
	Rationale   string
}

// parseQuestions parses questions from content
func (askf *AskFormat) parseQuestions(content string) []*Question {
	questions := make([]*Question, 0)

	// Pattern: QUESTION: <text>\nFile: <path>\nContext: <context>
	questionPattern := regexp.MustCompile(`(?mis)QUESTION:\s*([^\?]+)\?(?:\s*\nFile:\s*([^\n]+))?(?:\s*\nContext:\s*([^\n]+))?(?:(?:\n(?:QUESTION|PROPOSAL|CLARIFICATION))|$)`)
	matches := questionPattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		questionText := strings.TrimSpace(match[1]) + "?"
		filePath := ""
		if len(match) > 2 {
			filePath = strings.TrimSpace(match[2])
		}
		context := ""
		if len(match) > 3 {
			context = strings.TrimSpace(match[3])
		}

		questions = append(questions, &Question{
			FilePath:    filePath,
			QuestionText: questionText,
			Context:     context,
		})
	}

	// Also look for inline questions
	inlinePattern := regexp.MustCompile(`(?m)^(?:Should I|Would you like|Do you want) (.+\?)`)
	inlineMatches := inlinePattern.FindAllStringSubmatch(content, -1)

	for _, match := range inlineMatches {
		questions = append(questions, &Question{
			QuestionText: match[0],
		})
	}

	return questions
}

// parseProposals parses proposed changes from content
func (askf *AskFormat) parseProposals(content string) []*Proposal {
	proposals := make([]*Proposal, 0)

	// Pattern: PROPOSED CHANGE:\nFile: <path>\nDescription: <desc>\nRationale: <reason>
	proposalPattern := regexp.MustCompile(`(?mis)PROPOSED CHANGE:\s*\nFile:\s*([^\n]+)\nDescription:\s*([^\n]+)(?:\s*\nRationale:\s*([^\n]+))?(?:(?:\n(?:PROPOSED|QUESTION))|$)`)
	matches := proposalPattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		filePath := strings.TrimSpace(match[1])
		description := strings.TrimSpace(match[2])
		rationale := ""
		if len(match) > 3 {
			rationale = strings.TrimSpace(match[3])
		}

		proposals = append(proposals, &Proposal{
			FilePath:    filePath,
			Description: description,
			Rationale:   rationale,
		})
	}

	// Alternative pattern: CONFIRM: <action> for <file>?
	confirmPattern := regexp.MustCompile(`(?mi)CONFIRM:\s*(.+?)\s+for\s+(.+?)\?`)
	confirmMatches := confirmPattern.FindAllStringSubmatch(content, -1)

	for _, match := range confirmMatches {
		action := strings.TrimSpace(match[1])
		filePath := strings.TrimSpace(match[2])

		proposals = append(proposals, &Proposal{
			FilePath:    filePath,
			Description: action,
		})
	}

	return proposals
}

// Format formats structured edits into this format's representation
func (askf *AskFormat) Format(edits []*FileEdit) (string, error) {
	if len(edits) == 0 {
		return "", fmt.Errorf("no edits to format")
	}

	var sb strings.Builder

	for i, edit := range edits {
		if i > 0 {
			sb.WriteString("\n\n")
		}

		// Format questions
		if questions, ok := edit.Metadata["questions"].([]*Question); ok {
			for j, q := range questions {
				if j > 0 {
					sb.WriteString("\n")
				}
				sb.WriteString(fmt.Sprintf("QUESTION: %s\n", q.QuestionText))
				if q.FilePath != "" {
					sb.WriteString(fmt.Sprintf("File: %s\n", q.FilePath))
				}
				if q.Context != "" {
					sb.WriteString(fmt.Sprintf("Context: %s\n", q.Context))
				}
			}
		}

		// Format proposals
		if proposals, ok := edit.Metadata["proposals"].([]*Proposal); ok {
			for j, p := range proposals {
				if j > 0 || len(edit.Metadata["questions"].([]*Question)) > 0 {
					sb.WriteString("\n")
				}
				sb.WriteString("PROPOSED CHANGE:\n")
				sb.WriteString(fmt.Sprintf("File: %s\n", p.FilePath))
				sb.WriteString(fmt.Sprintf("Description: %s\n", p.Description))
				if p.Rationale != "" {
					sb.WriteString(fmt.Sprintf("Rationale: %s\n", p.Rationale))
				}
			}
		}
	}

	return sb.String(), nil
}

// PromptTemplate returns the prompt template for this format
func (askf *AskFormat) PromptTemplate() string {
	return `When you need clarification before making changes, use ask mode:

QUESTION: <your question>?
File: <file_path>
Context: <relevant context>

PROPOSED CHANGE:
File: <file_path>
Description: <what you plan to change>
Rationale: <why this change is needed>

Alternative formats:
CLARIFICATION: <what you need to know>?
CONFIRM: <action> for <file>?

Examples:

QUESTION: Should I use a mutex or a channel for concurrency control?
File: src/worker/pool.go
Context: Managing access to the worker queue

PROPOSED CHANGE:
File: src/auth/middleware.go
Description: Add JWT token validation middleware
Rationale: Current implementation doesn't validate token signatures

CLARIFICATION: Which error handling strategy do you prefer - return errors or panic?

CONFIRM: Delete the deprecated helper functions for utils/legacy.go?

Important:
- Ask before making significant architectural decisions
- Propose changes when unsure about approach
- Request clarification for ambiguous requirements
- Always include file paths when relevant
- Provide context to help with decision making
- Offer options when multiple approaches are valid
- Use this mode to prevent unnecessary refactoring
`
}

// Validate validates that the format is correctly used
func (askf *AskFormat) Validate(content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content cannot be empty")
	}

	// Check for question marks
	if !strings.Contains(content, "?") {
		return fmt.Errorf("ask mode must contain questions (marked with ?)")
	}

	// Try to parse and see if we get any edits
	edits, err := askf.Parse(context.Background(), content)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if len(edits) == 0 {
		return fmt.Errorf("no valid questions or proposals found")
	}

	// Check that we have either questions or proposals
	for _, edit := range edits {
		hasQuestions := false
		hasProposals := false

		if questions, ok := edit.Metadata["questions"].([]*Question); ok && len(questions) > 0 {
			hasQuestions = true
		}
		if proposals, ok := edit.Metadata["proposals"].([]*Proposal); ok && len(proposals) > 0 {
			hasProposals = true
		}

		if !hasQuestions && !hasProposals {
			return fmt.Errorf("ask mode must have at least one question or proposal")
		}
	}

	return nil
}
