package mentions

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Problem represents a workspace problem (error, warning, etc.)
type Problem struct {
	Type    string // error, warning, info
	File    string
	Line    int
	Column  int
	Message string
	Source  string // compiler, linter, test, etc.
}

// ProblemsMentionHandler handles @problems mentions
type ProblemsMentionHandler struct {
	problems []Problem
}

// NewProblemsMentionHandler creates a new problems mention handler
func NewProblemsMentionHandler() *ProblemsMentionHandler {
	return &ProblemsMentionHandler{
		problems: make([]Problem, 0),
	}
}

// Type returns the mention type
func (h *ProblemsMentionHandler) Type() MentionType {
	return MentionTypeProblems
}

// CanHandle checks if this handler can handle the mention
func (h *ProblemsMentionHandler) CanHandle(mention string) bool {
	return strings.HasPrefix(mention, "@problems")
}

// Resolve resolves the problems mention
func (h *ProblemsMentionHandler) Resolve(ctx context.Context, target string, options map[string]string) (*MentionContext, error) {
	filterType := options["type"] // errors, warnings, all
	if filterType == "" {
		filterType = "all"
	}

	var content strings.Builder
	errorCount := 0
	warningCount := 0

	for _, problem := range h.problems {
		// Filter by type if specified
		if filterType != "all" {
			if filterType == "errors" && problem.Type != "error" {
				continue
			}
			if filterType == "warnings" && problem.Type != "warning" {
				continue
			}
		}

		// Count problems
		if problem.Type == "error" {
			errorCount++
		} else if problem.Type == "warning" {
			warningCount++
		}

		// Format problem
		icon := "❌"
		if problem.Type == "warning" {
			icon = "⚠️"
		} else if problem.Type == "info" {
			icon = "ℹ️"
		}

		content.WriteString(fmt.Sprintf("%s %s:%d:%d [%s] %s\n",
			icon, problem.File, problem.Line, problem.Column, problem.Source, problem.Message))
	}

	if content.Len() == 0 {
		content.WriteString("No problems found! ✅\n")
	}

	summary := fmt.Sprintf("=== Workspace Problems ===\n"+
		"Errors: %d, Warnings: %d\n\n%s",
		errorCount, warningCount, content.String())

	tokenCount := len(summary) / 4

	return &MentionContext{
		Type:       MentionTypeProblems,
		Target:     fmt.Sprintf("%d errors, %d warnings", errorCount, warningCount),
		Content:    summary,
		TokenCount: tokenCount,
		Metadata: map[string]interface{}{
			"error_count":   errorCount,
			"warning_count": warningCount,
			"total_count":   len(h.problems),
		},
		ResolvedAt: time.Now(),
	}, nil
}

// AddProblem adds a problem to the list
func (h *ProblemsMentionHandler) AddProblem(problem Problem) {
	h.problems = append(h.problems, problem)
}

// ClearProblems clears all problems
func (h *ProblemsMentionHandler) ClearProblems() {
	h.problems = make([]Problem, 0)
}

// SetProblems sets the problem list
func (h *ProblemsMentionHandler) SetProblems(problems []Problem) {
	h.problems = problems
}
