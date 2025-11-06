package editor

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
)

// LineEditor performs line-based edits
type LineEditor struct{}

// NewLineEditor creates a new line editor
func NewLineEditor() *LineEditor {
	return &LineEditor{}
}

// LineEdit represents a line-based edit operation
type LineEdit struct {
	StartLine  int    // 1-based line number
	EndLine    int    // 1-based line number (inclusive)
	NewContent string // New content for the line range
}

// Apply applies line-based edits to a file
func (le *LineEditor) Apply(edit Edit) error {
	edits, ok := edit.Content.([]LineEdit)
	if !ok {
		return fmt.Errorf("line editor content must be []LineEdit")
	}

	if len(edits) == 0 {
		return fmt.Errorf("no line edits provided")
	}

	// Read the file
	lines, err := le.readFile(edit.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Validate all edits first
	for i, lineEdit := range edits {
		if err := le.validateLineEdit(lineEdit, len(lines)); err != nil {
			return fmt.Errorf("invalid edit %d: %w", i+1, err)
		}
	}

	// Sort edits by start line (descending) to apply from bottom to top
	// This prevents line number shifts from affecting subsequent edits
	sortedEdits := make([]LineEdit, len(edits))
	copy(sortedEdits, edits)
	sort.Slice(sortedEdits, func(i, j int) bool {
		return sortedEdits[i].StartLine > sortedEdits[j].StartLine
	})

	// Check for overlapping edits
	if err := le.checkOverlaps(sortedEdits); err != nil {
		return fmt.Errorf("overlapping edits detected: %w", err)
	}

	// Apply edits from bottom to top
	for _, lineEdit := range sortedEdits {
		lines, err = le.applyLineEdit(lines, lineEdit)
		if err != nil {
			return fmt.Errorf("failed to apply edit: %w", err)
		}
	}

	// Write the modified content
	if err := le.writeFile(edit.FilePath, lines); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// validateLineEdit validates a single line edit
func (le *LineEditor) validateLineEdit(edit LineEdit, totalLines int) error {
	if edit.StartLine < 1 {
		return fmt.Errorf("start line must be >= 1, got %d", edit.StartLine)
	}

	if edit.EndLine < edit.StartLine {
		return fmt.Errorf("end line (%d) must be >= start line (%d)", edit.EndLine, edit.StartLine)
	}

	if edit.StartLine > totalLines+1 {
		return fmt.Errorf("start line %d exceeds file length %d", edit.StartLine, totalLines)
	}

	return nil
}

// checkOverlaps checks if any line edits overlap
func (le *LineEditor) checkOverlaps(edits []LineEdit) error {
	for i := 0; i < len(edits)-1; i++ {
		for j := i + 1; j < len(edits); j++ {
			if le.editsOverlap(edits[i], edits[j]) {
				return fmt.Errorf("edits overlap: [%d-%d] and [%d-%d]",
					edits[i].StartLine, edits[i].EndLine,
					edits[j].StartLine, edits[j].EndLine)
			}
		}
	}
	return nil
}

// editsOverlap checks if two line edits overlap
func (le *LineEditor) editsOverlap(a, b LineEdit) bool {
	return !(a.EndLine < b.StartLine || b.EndLine < a.StartLine)
}

// applyLineEdit applies a single line edit
func (le *LineEditor) applyLineEdit(lines []string, edit LineEdit) ([]string, error) {
	// Convert to 0-based indices
	startIdx := edit.StartLine - 1
	endIdx := edit.EndLine - 1

	// Handle insertion at end of file
	if startIdx > len(lines) {
		startIdx = len(lines)
		endIdx = len(lines)
	}

	// Ensure endIdx is within bounds
	if endIdx >= len(lines) {
		endIdx = len(lines) - 1
	}

	// Split new content into lines
	newLines := strings.Split(edit.NewContent, "\n")

	// Build result
	result := make([]string, 0, len(lines)-endIdx+startIdx+len(newLines))

	// Copy lines before the edit
	result = append(result, lines[:startIdx]...)

	// Add new content
	result = append(result, newLines...)

	// Copy lines after the edit
	if endIdx+1 < len(lines) {
		result = append(result, lines[endIdx+1:]...)
	}

	return result, nil
}

// InsertLines inserts lines at a specific position
func (le *LineEditor) InsertLines(lines []string, position int, newLines []string) []string {
	result := make([]string, 0, len(lines)+len(newLines))
	result = append(result, lines[:position]...)
	result = append(result, newLines...)
	result = append(result, lines[position:]...)
	return result
}

// DeleteLines deletes a range of lines
func (le *LineEditor) DeleteLines(lines []string, start, end int) []string {
	result := make([]string, 0, len(lines)-(end-start+1))
	result = append(result, lines[:start]...)
	if end+1 < len(lines) {
		result = append(result, lines[end+1:]...)
	}
	return result
}

// ReplaceLines replaces a range of lines with new content
func (le *LineEditor) ReplaceLines(lines []string, start, end int, newLines []string) []string {
	result := make([]string, 0, len(lines)-(end-start+1)+len(newLines))
	result = append(result, lines[:start]...)
	result = append(result, newLines...)
	if end+1 < len(lines) {
		result = append(result, lines[end+1:]...)
	}
	return result
}

// GetLineRange extracts a range of lines
func (le *LineEditor) GetLineRange(lines []string, start, end int) []string {
	if start < 0 || end >= len(lines) || start > end {
		return []string{}
	}
	result := make([]string, end-start+1)
	copy(result, lines[start:end+1])
	return result
}

// readFile reads a file and returns its lines
func (le *LineEditor) readFile(filePath string) ([]string, error) {
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

// writeFile writes lines to a file
func (le *LineEditor) writeFile(filePath string, lines []string) error {
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

// LineEditStats contains statistics about line edits
type LineEditStats struct {
	EditCount     int
	LinesInserted int
	LinesDeleted  int
	LinesModified int
}

// GetStats calculates statistics for a set of line edits
func (le *LineEditor) GetStats(originalLines []string, edits []LineEdit) *LineEditStats {
	stats := &LineEditStats{
		EditCount: len(edits),
	}

	for _, edit := range edits {
		linesAffected := edit.EndLine - edit.StartLine + 1
		newLineCount := len(strings.Split(edit.NewContent, "\n"))

		if newLineCount > linesAffected {
			stats.LinesInserted += newLineCount - linesAffected
		} else if newLineCount < linesAffected {
			stats.LinesDeleted += linesAffected - newLineCount
		}

		// Count as modified if same line count but different content
		if newLineCount == linesAffected {
			stats.LinesModified += newLineCount
		}
	}

	return stats
}

// ValidateLineRange validates that a line range is within bounds
func (le *LineEditor) ValidateLineRange(lines []string, start, end int) error {
	if start < 1 {
		return fmt.Errorf("start line must be >= 1")
	}

	if end < start {
		return fmt.Errorf("end line must be >= start line")
	}

	if start > len(lines) {
		return fmt.Errorf("start line %d exceeds file length %d", start, len(lines))
	}

	if end > len(lines) {
		return fmt.Errorf("end line %d exceeds file length %d", end, len(lines))
	}

	return nil
}

// ApplySingleLineEdit applies a single line edit without sorting
// Useful for interactive editing scenarios
func (le *LineEditor) ApplySingleLineEdit(filePath string, lineEdit LineEdit) error {
	lines, err := le.readFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if err := le.validateLineEdit(lineEdit, len(lines)); err != nil {
		return fmt.Errorf("invalid edit: %w", err)
	}

	lines, err = le.applyLineEdit(lines, lineEdit)
	if err != nil {
		return fmt.Errorf("failed to apply edit: %w", err)
	}

	if err := le.writeFile(filePath, lines); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
