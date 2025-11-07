package mentions

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// TerminalMentionHandler handles @terminal mentions
type TerminalMentionHandler struct {
	terminalHistory []string
}

// NewTerminalMentionHandler creates a new terminal mention handler
func NewTerminalMentionHandler() *TerminalMentionHandler {
	return &TerminalMentionHandler{
		terminalHistory: make([]string, 0),
	}
}

// Type returns the mention type
func (h *TerminalMentionHandler) Type() MentionType {
	return MentionTypeTerminal
}

// CanHandle checks if this handler can handle the mention
func (h *TerminalMentionHandler) CanHandle(mention string) bool {
	return strings.HasPrefix(mention, "@terminal")
}

// Resolve resolves the terminal mention
func (h *TerminalMentionHandler) Resolve(ctx context.Context, target string, options map[string]string) (*MentionContext, error) {
	lines := 50 // Default number of lines
	if linesStr, exists := options["lines"]; exists {
		if l, err := strconv.Atoi(linesStr); err == nil {
			lines = l
		}
	}

	// Get last N lines from history
	start := 0
	if len(h.terminalHistory) > lines {
		start = len(h.terminalHistory) - lines
	}

	content := strings.Join(h.terminalHistory[start:], "\n")
	tokenCount := len(content) / 4

	return &MentionContext{
		Type:       MentionTypeTerminal,
		Target:     fmt.Sprintf("last %d lines", lines),
		Content:    content,
		TokenCount: tokenCount,
		Metadata: map[string]interface{}{
			"total_lines": len(h.terminalHistory),
			"shown_lines": len(h.terminalHistory[start:]),
		},
		ResolvedAt: time.Now(),
	}, nil
}

// AddOutput adds terminal output to history
func (h *TerminalMentionHandler) AddOutput(output string) {
	h.terminalHistory = append(h.terminalHistory, output)

	// Keep only last 1000 lines
	if len(h.terminalHistory) > 1000 {
		h.terminalHistory = h.terminalHistory[len(h.terminalHistory)-1000:]
	}
}
