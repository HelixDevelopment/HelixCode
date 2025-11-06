package editor

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// DiffEditor applies unified diff format edits
type DiffEditor struct{}

// NewDiffEditor creates a new diff editor
func NewDiffEditor() *DiffEditor {
	return &DiffEditor{}
}

// Apply applies a unified diff to a file
func (de *DiffEditor) Apply(edit Edit) error {
	diffContent, ok := edit.Content.(string)
	if !ok {
		return fmt.Errorf("diff content must be a string")
	}

	// Parse the diff
	hunks, err := de.parseDiff(diffContent)
	if err != nil {
		return fmt.Errorf("failed to parse diff: %w", err)
	}

	// Read the original file
	originalLines, err := de.readFile(edit.FilePath)
	if err != nil {
		// If file doesn't exist, treat as empty file
		if os.IsNotExist(err) {
			originalLines = []string{}
		} else {
			return fmt.Errorf("failed to read file: %w", err)
		}
	}

	// Apply hunks to get new content
	newLines, err := de.applyHunks(originalLines, hunks)
	if err != nil {
		return fmt.Errorf("failed to apply hunks: %w", err)
	}

	// Write the modified content
	if err := de.writeFile(edit.FilePath, newLines); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// DiffHunk represents a single hunk in a unified diff
type DiffHunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Lines    []DiffLine
}

// DiffLine represents a line in a diff hunk
type DiffLine struct {
	Type    byte   // '+', '-', or ' '
	Content string
}

// parseDiff parses a unified diff format string into hunks
func (de *DiffEditor) parseDiff(diffContent string) ([]DiffHunk, error) {
	var hunks []DiffHunk
	lines := strings.Split(diffContent, "\n")

	var currentHunk *DiffHunk
	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Skip file headers (--- and +++)
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
			continue
		}

		// Parse hunk header
		if strings.HasPrefix(line, "@@") {
			if currentHunk != nil {
				hunks = append(hunks, *currentHunk)
			}

			hunk, err := de.parseHunkHeader(line)
			if err != nil {
				return nil, err
			}
			currentHunk = &hunk
			continue
		}

		// Parse hunk lines
		if currentHunk != nil {
			if len(line) == 0 {
				// Empty line is treated as context
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    ' ',
					Content: "",
				})
			} else if line[0] == '+' || line[0] == '-' || line[0] == ' ' {
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    line[0],
					Content: line[1:],
				})
			}
		}
	}

	if currentHunk != nil {
		hunks = append(hunks, *currentHunk)
	}

	return hunks, nil
}

// parseHunkHeader parses a hunk header line like "@@ -1,3 +1,4 @@"
func (de *DiffEditor) parseHunkHeader(line string) (DiffHunk, error) {
	// Extract the range information
	parts := strings.Split(line, "@@")
	if len(parts) < 2 {
		return DiffHunk{}, fmt.Errorf("invalid hunk header: %s", line)
	}

	ranges := strings.TrimSpace(parts[1])
	rangeParts := strings.Split(ranges, " ")
	if len(rangeParts) != 2 {
		return DiffHunk{}, fmt.Errorf("invalid hunk range: %s", ranges)
	}

	// Parse old range
	oldRange := strings.TrimPrefix(rangeParts[0], "-")
	oldStart, oldCount, err := de.parseRange(oldRange)
	if err != nil {
		return DiffHunk{}, fmt.Errorf("invalid old range: %w", err)
	}

	// Parse new range
	newRange := strings.TrimPrefix(rangeParts[1], "+")
	newStart, newCount, err := de.parseRange(newRange)
	if err != nil {
		return DiffHunk{}, fmt.Errorf("invalid new range: %w", err)
	}

	return DiffHunk{
		OldStart: oldStart,
		OldCount: oldCount,
		NewStart: newStart,
		NewCount: newCount,
		Lines:    []DiffLine{},
	}, nil
}

// parseRange parses a range like "1,3" or "1"
func (de *DiffEditor) parseRange(rangeStr string) (int, int, error) {
	parts := strings.Split(rangeStr, ",")
	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}

	count := 1
	if len(parts) > 1 {
		count, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, err
		}
	}

	return start, count, nil
}

// applyHunks applies multiple hunks to the original lines
func (de *DiffEditor) applyHunks(originalLines []string, hunks []DiffHunk) ([]string, error) {
	result := make([]string, 0, len(originalLines))
	lineIndex := 0

	for _, hunk := range hunks {
		// Copy lines before this hunk
		targetLine := hunk.OldStart - 1 // Convert to 0-based index
		for lineIndex < targetLine && lineIndex < len(originalLines) {
			result = append(result, originalLines[lineIndex])
			lineIndex++
		}

		// Apply the hunk
		originalIndex := 0
		for _, diffLine := range hunk.Lines {
			switch diffLine.Type {
			case ' ':
				// Context line - verify and copy
				if lineIndex >= len(originalLines) {
					return nil, fmt.Errorf("hunk context mismatch: unexpected end of file")
				}
				if originalLines[lineIndex] != diffLine.Content {
					return nil, fmt.Errorf("hunk context mismatch at line %d: expected %q, got %q",
						lineIndex+1, diffLine.Content, originalLines[lineIndex])
				}
				result = append(result, diffLine.Content)
				lineIndex++
				originalIndex++
			case '-':
				// Removed line - verify and skip
				if lineIndex >= len(originalLines) {
					return nil, fmt.Errorf("hunk delete mismatch: unexpected end of file")
				}
				if originalLines[lineIndex] != diffLine.Content {
					return nil, fmt.Errorf("hunk delete mismatch at line %d: expected %q, got %q",
						lineIndex+1, diffLine.Content, originalLines[lineIndex])
				}
				lineIndex++
				originalIndex++
			case '+':
				// Added line - insert
				result = append(result, diffLine.Content)
			}
		}
	}

	// Copy remaining lines
	for lineIndex < len(originalLines) {
		result = append(result, originalLines[lineIndex])
		lineIndex++
	}

	return result, nil
}

// readFile reads a file and returns its lines
func (de *DiffEditor) readFile(filePath string) ([]string, error) {
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
func (de *DiffEditor) writeFile(filePath string, lines []string) error {
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
