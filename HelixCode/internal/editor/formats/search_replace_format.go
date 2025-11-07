package formats

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// SearchReplaceFormat handles regex-based search and replace
type SearchReplaceFormat struct{}

// NewSearchReplaceFormat creates a new search/replace format handler
func NewSearchReplaceFormat() *SearchReplaceFormat {
	return &SearchReplaceFormat{}
}

// Type returns the format type
func (srf *SearchReplaceFormat) Type() FormatType {
	return FormatTypeSearchReplace
}

// Name returns the human-readable name
func (srf *SearchReplaceFormat) Name() string {
	return "Search/Replace"
}

// Description returns the format description
func (srf *SearchReplaceFormat) Description() string {
	return "Regex-based search and replace operations"
}

// CanHandle checks if this format can handle the given content
func (srf *SearchReplaceFormat) CanHandle(content string) bool {
	// Look for search/replace markers
	markers := []string{
		"SEARCH:",
		"REPLACE:",
		"<<<<<<< SEARCH",
		">>>>>>> REPLACE",
		"search:",
		"replace:",
	}

	markerCount := 0
	for _, marker := range markers {
		if strings.Contains(strings.ToLower(content), strings.ToLower(marker)) {
			markerCount++
		}
	}

	// Require at least one pair of markers
	return markerCount >= 2
}

// Parse parses the edit content and returns structured edits
func (srf *SearchReplaceFormat) Parse(ctx context.Context, content string) ([]*FileEdit, error) {
	edits := make([]*FileEdit, 0)

	// Try different search/replace patterns
	patterns := []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{
			name: "block_style",
			// File: <path>
			// <<<<<<< SEARCH
			// <search pattern>
			// =======
			// <replace with>
			// >>>>>>> REPLACE
			pattern: regexp.MustCompile(`(?ms)File:\s*(.+?)\s*\n<<<<<<< SEARCH\n(.*?)\n=======\n(.*?)\n>>>>>>> REPLACE`),
		},
		{
			name: "keyword_style",
			// File: <path>
			// SEARCH:
			// <search pattern>
			// REPLACE:
			// <replace with>
			pattern: regexp.MustCompile(`(?mis)File:\s*(.+?)\s*\nSEARCH:\s*\n(.*?)(?:\nREPLACE:\s*\n)(.*?)(?:\n(?:File:|$))`),
		},
		{
			name: "inline_style",
			// File: <path>
			// search: <pattern>
			// replace: <text>
			pattern: regexp.MustCompile(`(?mi)File:\s*(.+?)\s*\nsearch:\s*(.+?)\s*\nreplace:\s*(.+?)(?:\n|$)`),
		},
	}

	for _, p := range patterns {
		matches := p.pattern.FindAllStringSubmatch(content, -1)
		if len(matches) > 0 {
			for _, match := range matches {
				filePath := strings.TrimSpace(match[1])
				searchPattern := strings.TrimSpace(match[2])
				replaceWith := strings.TrimSpace(match[3])

				edit := &FileEdit{
					FilePath:      filePath,
					Operation:     EditOperationUpdate,
					SearchPattern: searchPattern,
					ReplaceWith:   replaceWith,
					Metadata: map[string]interface{}{
						"format":  "search-replace",
						"pattern": p.name,
					},
				}

				if err := ValidateEdit(edit); err != nil {
					return nil, fmt.Errorf("invalid edit for %s: %w", filePath, err)
				}

				edits = append(edits, edit)
			}
		}
	}

	if len(edits) == 0 {
		return nil, fmt.Errorf("no valid search/replace edits found in content")
	}

	return edits, nil
}

// Format formats structured edits into this format's representation
func (srf *SearchReplaceFormat) Format(edits []*FileEdit) (string, error) {
	if len(edits) == 0 {
		return "", fmt.Errorf("no edits to format")
	}

	var sb strings.Builder

	for i, edit := range edits {
		if i > 0 {
			sb.WriteString("\n\n")
		}

		// Use block style format
		sb.WriteString(fmt.Sprintf("File: %s\n", edit.FilePath))
		sb.WriteString("<<<<<<< SEARCH\n")
		sb.WriteString(edit.SearchPattern)
		if !strings.HasSuffix(edit.SearchPattern, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("=======\n")
		sb.WriteString(edit.ReplaceWith)
		if !strings.HasSuffix(edit.ReplaceWith, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString(">>>>>>> REPLACE")
	}

	return sb.String(), nil
}

// PromptTemplate returns the prompt template for this format
func (srf *SearchReplaceFormat) PromptTemplate() string {
	return `When editing files, use search/replace blocks:

File: <file_path>
<<<<<<< SEARCH
<exact text to find>
=======
<replacement text>
>>>>>>> REPLACE

Alternative formats:

File: <file_path>
SEARCH:
<text to find>
REPLACE:
<replacement text>

OR

File: <file_path>
search: <pattern>
replace: <text>

Examples:

File: src/main.go
<<<<<<< SEARCH
func oldFunction() {
    return "old"
}
=======
func newFunction() {
    return "new"
}
>>>>>>> REPLACE

File: config.yaml
SEARCH:
timeout: 30
REPLACE:
timeout: 60

File: README.md
search: version 1.0
replace: version 2.0

Important:
- Search pattern must match EXACTLY (whitespace matters)
- Use exact text from the file, not approximations
- For regex patterns, escape special characters
- Multiple search/replace blocks can be in one response
- Each block is applied sequentially
`
}

// Validate validates that the format is correctly used
func (srf *SearchReplaceFormat) Validate(content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content cannot be empty")
	}

	// Check for required markers
	hasSearch := strings.Contains(strings.ToLower(content), "search")
	hasReplace := strings.Contains(strings.ToLower(content), "replace")

	if !hasSearch || !hasReplace {
		return fmt.Errorf("content must contain both SEARCH and REPLACE markers")
	}

	// Try to parse and see if we get any edits
	edits, err := srf.Parse(context.Background(), content)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if len(edits) == 0 {
		return fmt.Errorf("no valid edits found")
	}

	// Validate each edit has non-empty search pattern
	for _, edit := range edits {
		if edit.SearchPattern == "" {
			return fmt.Errorf("search pattern cannot be empty for %s", edit.FilePath)
		}
	}

	return nil
}
