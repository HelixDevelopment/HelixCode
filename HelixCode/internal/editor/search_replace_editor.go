package editor

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// SearchReplaceEditor performs search and replace operations
type SearchReplaceEditor struct{}

// NewSearchReplaceEditor creates a new search/replace editor
func NewSearchReplaceEditor() *SearchReplaceEditor {
	return &SearchReplaceEditor{}
}

// SearchReplace represents a single search/replace operation
type SearchReplace struct {
	Search  string // Text to search for
	Replace string // Replacement text
	Count   int    // Number of replacements (-1 for all, 0 for none, >0 for specific count)
	Regex   bool   // Whether to use regex matching
}

// Apply applies search/replace operations to a file
func (sre *SearchReplaceEditor) Apply(edit Edit) error {
	operations, ok := edit.Content.([]SearchReplace)
	if !ok {
		return fmt.Errorf("search/replace content must be []SearchReplace")
	}

	if len(operations) == 0 {
		return fmt.Errorf("no search/replace operations provided")
	}

	// Read the file content
	content, err := sre.readFile(edit.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Apply each operation in sequence
	for i, op := range operations {
		content, err = sre.applyOperation(content, op)
		if err != nil {
			return fmt.Errorf("operation %d failed: %w", i+1, err)
		}
	}

	// Write the modified content back
	if err := sre.writeFile(edit.FilePath, content); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// applyOperation applies a single search/replace operation
func (sre *SearchReplaceEditor) applyOperation(content string, op SearchReplace) (string, error) {
	if op.Search == "" {
		return "", fmt.Errorf("search string cannot be empty")
	}

	if op.Count == 0 {
		return content, nil // No replacements requested
	}

	if op.Regex {
		return sre.applyRegexOperation(content, op)
	}

	return sre.applyLiteralOperation(content, op)
}

// applyLiteralOperation applies a literal (non-regex) search/replace
func (sre *SearchReplaceEditor) applyLiteralOperation(content string, op SearchReplace) (string, error) {
	if op.Count < 0 {
		// Replace all occurrences
		return strings.ReplaceAll(content, op.Search, op.Replace), nil
	}

	// Replace specific number of occurrences
	result := content
	replaced := 0

	for replaced < op.Count {
		index := strings.Index(result, op.Search)
		if index == -1 {
			// No more occurrences found
			break
		}

		// Replace this occurrence
		result = result[:index] + op.Replace + result[index+len(op.Search):]
		replaced++
	}

	if replaced == 0 {
		return "", fmt.Errorf("search string not found: %q", op.Search)
	}

	return result, nil
}

// applyRegexOperation applies a regex search/replace
func (sre *SearchReplaceEditor) applyRegexOperation(content string, op SearchReplace) (string, error) {
	re, err := regexp.Compile(op.Search)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
	}

	if op.Count < 0 {
		// Replace all matches
		return re.ReplaceAllString(content, op.Replace), nil
	}

	// Replace specific number of matches
	matches := re.FindAllStringIndex(content, op.Count)
	if len(matches) == 0 {
		return "", fmt.Errorf("regex pattern not found: %q", op.Search)
	}

	// Build result by replacing matches in reverse order
	// (to avoid offset issues)
	result := content
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		matchText := content[match[0]:match[1]]
		replacement := re.ReplaceAllString(matchText, op.Replace)
		result = result[:match[0]] + replacement + result[match[1]:]
	}

	return result, nil
}

// ApplyToLines applies search/replace operations line by line
func (sre *SearchReplaceEditor) ApplyToLines(edit Edit) error {
	operations, ok := edit.Content.([]SearchReplace)
	if !ok {
		return fmt.Errorf("search/replace content must be []SearchReplace")
	}

	// Read the file lines
	lines, err := sre.readFileLines(edit.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Apply operations to each line
	for i, line := range lines {
		for _, op := range operations {
			modifiedLine, err := sre.applyOperation(line, op)
			if err != nil {
				// Skip lines where pattern doesn't match
				continue
			}
			lines[i] = modifiedLine
		}
	}

	// Write the modified lines back
	if err := sre.writeFileLines(edit.FilePath, lines); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ValidateOperation validates a search/replace operation
func (sre *SearchReplaceEditor) ValidateOperation(op SearchReplace) error {
	if op.Search == "" {
		return fmt.Errorf("search string cannot be empty")
	}

	if op.Regex {
		// Validate regex pattern
		if _, err := regexp.Compile(op.Search); err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	return nil
}

// CountMatches returns the number of matches for a search operation
func (sre *SearchReplaceEditor) CountMatches(content string, search string, regex bool) (int, error) {
	if regex {
		re, err := regexp.Compile(search)
		if err != nil {
			return 0, fmt.Errorf("invalid regex pattern: %w", err)
		}
		return len(re.FindAllString(content, -1)), nil
	}

	return strings.Count(content, search), nil
}

// readFile reads entire file content as string
func (sre *SearchReplaceEditor) readFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// writeFile writes string content to file
func (sre *SearchReplaceEditor) writeFile(filePath string, content string) error {
	return os.WriteFile(filePath, []byte(content), 0644)
}

// readFileLines reads file content as lines
func (sre *SearchReplaceEditor) readFileLines(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

// writeFileLines writes lines to file
func (sre *SearchReplaceEditor) writeFileLines(filePath string, lines []string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return err
		}
	}

	return writer.Flush()
}

// SearchReplaceStats contains statistics about search/replace operations
type SearchReplaceStats struct {
	OperationCount int
	TotalMatches   int
	TotalReplaced  int
	LinesModified  int
}

// GetStats returns statistics for a set of operations on content
func (sre *SearchReplaceEditor) GetStats(content string, operations []SearchReplace) (*SearchReplaceStats, error) {
	stats := &SearchReplaceStats{
		OperationCount: len(operations),
	}

	tempContent := content
	for _, op := range operations {
		matches, err := sre.CountMatches(tempContent, op.Search, op.Regex)
		if err != nil {
			return nil, err
		}

		stats.TotalMatches += matches

		// Determine how many will actually be replaced
		if op.Count < 0 {
			stats.TotalReplaced += matches
		} else if op.Count > 0 {
			if op.Count > matches {
				stats.TotalReplaced += matches
			} else {
				stats.TotalReplaced += op.Count
			}
		}

		// Apply to get modified content for next operation
		tempContent, err = sre.applyOperation(tempContent, op)
		if err != nil {
			continue
		}
	}

	// Count modified lines
	originalLines := strings.Split(content, "\n")
	modifiedLines := strings.Split(tempContent, "\n")

	for i := 0; i < len(originalLines) && i < len(modifiedLines); i++ {
		if originalLines[i] != modifiedLines[i] {
			stats.LinesModified++
		}
	}

	return stats, nil
}
