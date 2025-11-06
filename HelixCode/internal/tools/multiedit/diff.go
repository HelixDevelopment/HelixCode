package multiedit

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"time"
)

// DiffManager generates and applies diffs
type DiffManager struct {
	format DiffFormat
}

// DiffFormat specifies the diff format
type DiffFormat int

const (
	FormatUnified DiffFormat = iota
	FormatContext
	FormatWholeLine
	FormatSearchReplace
)

// String returns the string representation of the format
func (f DiffFormat) String() string {
	switch f {
	case FormatUnified:
		return "unified"
	case FormatContext:
		return "context"
	case FormatWholeLine:
		return "whole_line"
	case FormatSearchReplace:
		return "search_replace"
	default:
		return "unknown"
	}
}

// NewDiffManager creates a new diff manager
func NewDiffManager(format DiffFormat) *DiffManager {
	return &DiffManager{
		format: format,
	}
}

// GenerateDiff creates a unified diff
func (dm *DiffManager) GenerateDiff(oldContent, newContent []byte, filePath string) (*Diff, error) {
	// Split content into lines
	oldLines := splitLines(string(oldContent))
	newLines := splitLines(string(newContent))

	// Compute edit operations using Myers algorithm (simplified version)
	edits := computeEdits(oldLines, newLines)

	// Generate unified diff format
	unified := dm.formatUnifiedDiff(filePath, oldLines, newLines, edits)

	// Parse hunks
	hunks := dm.parseHunks(unified, oldLines, newLines)

	return &Diff{
		FilePath:   filePath,
		OldContent: oldContent,
		NewContent: newContent,
		Unified:    unified,
		Hunks:      hunks,
		Stats:      calculateStats(oldLines, newLines),
	}, nil
}

// ApplyDiff applies a diff to content
func (dm *DiffManager) ApplyDiff(diff *Diff) ([]byte, error) {
	if len(diff.Hunks) == 0 {
		return diff.NewContent, nil
	}

	// Apply hunks in order
	lines := splitLines(string(diff.OldContent))

	for _, hunk := range diff.Hunks {
		var err error
		lines, err = dm.applyHunk(lines, hunk)
		if err != nil {
			return nil, fmt.Errorf("failed to apply hunk: %w", err)
		}
	}

	// Join lines back
	result := joinLines(lines)
	return []byte(result), nil
}

// ParseDiff parses a unified diff string
func (dm *DiffManager) ParseDiff(diffText string) (*Diff, error) {
	lines := strings.Split(diffText, "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("invalid diff format")
	}

	diff := &Diff{
		Unified: diffText,
		Hunks:   make([]*DiffHunk, 0),
	}

	// Parse header
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "--- ") {
			// Old file
			diff.FilePath = strings.TrimPrefix(line, "--- ")
		} else if strings.HasPrefix(line, "+++ ") {
			// New file
			// Skip
		} else if strings.HasPrefix(line, "@@") {
			// Start of hunk
			hunk, consumed, err := dm.parseHunk(lines[i:])
			if err != nil {
				return nil, fmt.Errorf("failed to parse hunk: %w", err)
			}
			diff.Hunks = append(diff.Hunks, hunk)
			i += consumed - 1
		}
	}

	return diff, nil
}

// Diff represents file differences
type Diff struct {
	FilePath   string
	OldContent []byte
	NewContent []byte
	Unified    string
	Hunks      []*DiffHunk
	Stats      DiffStats
}

// DiffHunk represents a single diff hunk
type DiffHunk struct {
	OldStart int
	OldLines int
	NewStart int
	NewLines int
	Lines    []DiffLine
}

// DiffLine represents a single line in a diff
type DiffLine struct {
	Type    LineType
	Content string
	LineNo  int
}

// LineType represents the type of line
type LineType int

const (
	LineContext LineType = iota
	LineAdd
	LineDelete
)

// String returns the string representation of the line type
func (lt LineType) String() string {
	switch lt {
	case LineContext:
		return "context"
	case LineAdd:
		return "add"
	case LineDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// DiffStats contains diff statistics
type DiffStats struct {
	LinesAdded   int
	LinesDeleted int
	LinesChanged int
}

// formatUnifiedDiff formats a unified diff
func (dm *DiffManager) formatUnifiedDiff(filePath string, oldLines, newLines []string, edits []Edit) string {
	var buf bytes.Buffer

	// Write header
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(&buf, "--- %s\t%s\n", filePath, timestamp)
	fmt.Fprintf(&buf, "+++ %s\t%s\n", filePath, timestamp)

	// Group edits into hunks
	hunks := groupEditsIntoHunks(edits, oldLines, newLines, 3)

	// Write hunks
	for _, hunk := range hunks {
		fmt.Fprintf(&buf, "@@ -%d,%d +%d,%d @@\n",
			hunk.oldStart+1, hunk.oldCount,
			hunk.newStart+1, hunk.newCount)

		for _, line := range hunk.lines {
			buf.WriteString(line)
			buf.WriteString("\n")
		}
	}

	return buf.String()
}

// parseHunks parses hunks from unified diff
func (dm *DiffManager) parseHunks(unified string, oldLines, newLines []string) []*DiffHunk {
	var hunks []*DiffHunk

	lines := strings.Split(unified, "\n")
	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "@@") {
			hunk, consumed, err := dm.parseHunk(lines[i:])
			if err != nil {
				continue
			}
			hunks = append(hunks, hunk)
			i += consumed - 1
		}
	}

	return hunks
}

// parseHunk parses a single hunk
func (dm *DiffManager) parseHunk(lines []string) (*DiffHunk, int, error) {
	if len(lines) == 0 || !strings.HasPrefix(lines[0], "@@") {
		return nil, 0, fmt.Errorf("invalid hunk header")
	}

	// Parse hunk header: @@ -oldStart,oldCount +newStart,newCount @@
	header := lines[0]
	var oldStart, oldCount, newStart, newCount int
	_, err := fmt.Sscanf(header, "@@ -%d,%d +%d,%d @@", &oldStart, &oldCount, &newStart, &newCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse hunk header: %w", err)
	}

	hunk := &DiffHunk{
		OldStart: oldStart - 1, // Convert to 0-based
		OldLines: oldCount,
		NewStart: newStart - 1, // Convert to 0-based
		NewLines: newCount,
		Lines:    make([]DiffLine, 0),
	}

	// Parse hunk lines
	consumed := 1
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "@@") {
			break // Next hunk
		}

		var lineType LineType
		var content string

		switch line[0] {
		case ' ':
			lineType = LineContext
			content = line[1:]
		case '+':
			lineType = LineAdd
			content = line[1:]
		case '-':
			lineType = LineDelete
			content = line[1:]
		default:
			break // End of hunk
		}

		hunk.Lines = append(hunk.Lines, DiffLine{
			Type:    lineType,
			Content: content,
		})
		consumed++
	}

	return hunk, consumed, nil
}

// applyHunk applies a single hunk to lines
func (dm *DiffManager) applyHunk(lines []string, hunk *DiffHunk) ([]string, error) {
	result := make([]string, 0, len(lines))
	result = append(result, lines[:hunk.OldStart]...)

	// Apply hunk changes
	for _, line := range hunk.Lines {
		switch line.Type {
		case LineContext:
			result = append(result, line.Content)
		case LineAdd:
			result = append(result, line.Content)
		case LineDelete:
			// Skip deleted lines
		}
	}

	// Append remaining lines
	endIdx := hunk.OldStart + hunk.OldLines
	if endIdx < len(lines) {
		result = append(result, lines[endIdx:]...)
	}

	return result, nil
}

// Edit represents an edit operation
type Edit struct {
	Type    EditType
	OldPos  int
	NewPos  int
	OldText string
	NewText string
}

// EditType represents the type of edit
type EditType int

const (
	EditEqual EditType = iota
	EditInsert
	EditDelete
)

// hunkGroup represents a group of edits
type hunkGroup struct {
	oldStart int
	oldCount int
	newStart int
	newCount int
	lines    []string
}

// computeEdits computes edit operations using a simplified Myers algorithm
func computeEdits(oldLines, newLines []string) []Edit {
	var edits []Edit

	// Simple line-by-line comparison
	// This is a simplified version - a real implementation would use Myers diff algorithm
	oldLen := len(oldLines)
	newLen := len(newLines)

	i, j := 0, 0
	for i < oldLen && j < newLen {
		if oldLines[i] == newLines[j] {
			edits = append(edits, Edit{
				Type:    EditEqual,
				OldPos:  i,
				NewPos:  j,
				OldText: oldLines[i],
				NewText: newLines[j],
			})
			i++
			j++
		} else {
			// Look ahead to find matches
			foundMatch := false
			for k := 1; k < 5 && i+k < oldLen; k++ {
				if oldLines[i+k] == newLines[j] {
					// Delete lines
					for l := 0; l < k; l++ {
						edits = append(edits, Edit{
							Type:    EditDelete,
							OldPos:  i + l,
							NewPos:  j,
							OldText: oldLines[i+l],
						})
					}
					i += k
					foundMatch = true
					break
				}
			}

			if !foundMatch {
				// Try looking ahead in new lines
				for k := 1; k < 5 && j+k < newLen; k++ {
					if oldLines[i] == newLines[j+k] {
						// Insert lines
						for l := 0; l < k; l++ {
							edits = append(edits, Edit{
								Type:    EditInsert,
								OldPos:  i,
								NewPos:  j + l,
								NewText: newLines[j+l],
							})
						}
						j += k
						foundMatch = true
						break
					}
				}
			}

			if !foundMatch {
				// Delete and insert (change)
				edits = append(edits, Edit{
					Type:    EditDelete,
					OldPos:  i,
					NewPos:  j,
					OldText: oldLines[i],
				})
				edits = append(edits, Edit{
					Type:    EditInsert,
					OldPos:  i,
					NewPos:  j,
					NewText: newLines[j],
				})
				i++
				j++
			}
		}
	}

	// Handle remaining deletions
	for i < oldLen {
		edits = append(edits, Edit{
			Type:    EditDelete,
			OldPos:  i,
			NewPos:  newLen,
			OldText: oldLines[i],
		})
		i++
	}

	// Handle remaining insertions
	for j < newLen {
		edits = append(edits, Edit{
			Type:    EditInsert,
			OldPos:  oldLen,
			NewPos:  j,
			NewText: newLines[j],
		})
		j++
	}

	return edits
}

// groupEditsIntoHunks groups edits into hunks with context
func groupEditsIntoHunks(edits []Edit, oldLines, newLines []string, contextLines int) []hunkGroup {
	if len(edits) == 0 {
		return nil
	}

	var hunks []hunkGroup
	var currentHunk *hunkGroup

	for i := 0; i < len(edits); i++ {
		edit := edits[i]

		if edit.Type == EditEqual {
			if currentHunk != nil {
				// Add context after changes
				if len(currentHunk.lines) > 0 {
					for j := 0; j < contextLines && i+j < len(edits) && edits[i+j].Type == EditEqual; j++ {
						currentHunk.lines = append(currentHunk.lines, " "+edits[i+j].OldText)
						currentHunk.oldCount++
						currentHunk.newCount++
					}
				}
			}
			continue
		}

		// Start new hunk or continue current
		if currentHunk == nil {
			currentHunk = &hunkGroup{
				oldStart: edit.OldPos,
				newStart: edit.NewPos,
				lines:    make([]string, 0),
			}

			// Add context before changes
			start := edit.OldPos - contextLines
			if start < 0 {
				start = 0
			}
			for j := start; j < edit.OldPos && j < len(oldLines); j++ {
				currentHunk.lines = append(currentHunk.lines, " "+oldLines[j])
				currentHunk.oldCount++
				currentHunk.newCount++
				currentHunk.oldStart = j
				currentHunk.newStart = edit.NewPos - (edit.OldPos - j)
			}
		}

		// Add edit to hunk
		switch edit.Type {
		case EditDelete:
			currentHunk.lines = append(currentHunk.lines, "-"+edit.OldText)
			currentHunk.oldCount++
		case EditInsert:
			currentHunk.lines = append(currentHunk.lines, "+"+edit.NewText)
			currentHunk.newCount++
		}

		// Check if we should close the hunk
		if i+1 < len(edits) {
			nextEdit := edits[i+1]
			if nextEdit.Type == EditEqual {
				// Check distance to next change
				distance := 0
				for j := i + 1; j < len(edits) && edits[j].Type == EditEqual; j++ {
					distance++
				}
				if distance > contextLines*2 {
					// Close current hunk
					hunks = append(hunks, *currentHunk)
					currentHunk = nil
				}
			}
		} else {
			// Last edit, close hunk
			hunks = append(hunks, *currentHunk)
			currentHunk = nil
		}
	}

	if currentHunk != nil {
		hunks = append(hunks, *currentHunk)
	}

	return hunks
}

// calculateStats calculates diff statistics
func calculateStats(oldLines, newLines []string) DiffStats {
	edits := computeEdits(oldLines, newLines)

	stats := DiffStats{}
	for _, edit := range edits {
		switch edit.Type {
		case EditInsert:
			stats.LinesAdded++
		case EditDelete:
			stats.LinesDeleted++
		}
	}

	return stats
}

// splitLines splits content into lines
func splitLines(content string) []string {
	if content == "" {
		return []string{}
	}

	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
}

// joinLines joins lines back into content
func joinLines(lines []string) string {
	return strings.Join(lines, "\n")
}
