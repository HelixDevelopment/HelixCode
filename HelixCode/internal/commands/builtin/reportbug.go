package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"dev.helix.code/internal/commands"
	"dev.helix.code/internal/logging"
	"dev.helix.code/internal/version"
)

// ReportBugCommand files bug reports with system info
type ReportBugCommand struct{}

// NewReportBugCommand creates a new /reportbug command
func NewReportBugCommand() *ReportBugCommand {
	return &ReportBugCommand{}
}

// Name returns the command name
func (c *ReportBugCommand) Name() string {
	return "reportbug"
}

// Aliases returns command aliases
func (c *ReportBugCommand) Aliases() []string {
	return []string{"bug", "issue"}
}

// Description returns command description
func (c *ReportBugCommand) Description() string {
	return "File a bug report with system information and logs"
}

// Usage returns usage information
func (c *ReportBugCommand) Usage() string {
	return `/reportbug [description] [options]

Files a bug report to GitHub with system information, logs, and reproduction steps.

Examples:
  /reportbug "LLM timeout error"
  /reportbug --title "Memory leak in worker pool" --attach-logs
  /reportbug "Crash on startup" --labels bug,critical
  /reportbug --auto-submit

Flags:
  --title: Custom issue title (default: generated from description)
  --labels: Comma-separated labels (e.g., bug,high-priority)
  --attach-logs: Include recent logs (default: true)
  --auto-submit: Automatically submit to GitHub (requires auth)`
}

// Execute runs the command
func (c *ReportBugCommand) Execute(ctx context.Context, cmdCtx *commands.CommandContext) (*commands.CommandResult, error) {
	// Extract description
	description := strings.Join(cmdCtx.Args, " ")
	if description == "" {
		description = "Bug report from HelixCode"
	}

	// Parse flags
	title := cmdCtx.Flags["title"]
	if title == "" {
		title = fmt.Sprintf("Bug: %s", description)
	}

	labels := cmdCtx.Flags["labels"]
	if labels == "" {
		labels = "bug"
	}

	attachLogs := true
	if val, ok := cmdCtx.Flags["attach-logs"]; ok && val == "false" {
		attachLogs = false
	}

	autoSubmit := cmdCtx.Flags["auto-submit"] == "true"

	// Collect system information
	systemInfo := collectSystemInfo()

	// Collect recent logs if requested
	var recentLogs string
	if attachLogs {
		recentLogs = collectRecentLogs(cmdCtx.SessionID, 50)
	}

	// Format bug report
	bugReport := formatBugReport(title, description, systemInfo, recentLogs, cmdCtx.ChatHistory)

	// Create action to file bug report
	actions := []commands.Action{
		{
			Type: "file_bug_report",
			Data: map[string]interface{}{
				"title":       title,
				"labels":      strings.Split(labels, ","),
				"body":        bugReport,
				"auto_submit": autoSubmit,
				"session_id":  cmdCtx.SessionID,
				"user_id":     cmdCtx.UserID,
			},
		},
	}

	message := fmt.Sprintf("Bug report prepared: %s", title)
	if autoSubmit {
		message += " (submitting to GitHub...)"
	} else {
		message += " (review and submit manually)"
	}

	return &commands.CommandResult{
		Success:     true,
		Message:     message,
		Actions:     actions,
		ShouldReply: true,
		Metadata: map[string]interface{}{
			"title":       title,
			"labels":      labels,
			"auto_submit": autoSubmit,
			"report_size": len(bugReport),
		},
	}, nil
}

// collectSystemInfo gathers system information
func collectSystemInfo() map[string]string {
	return map[string]string{
		"go_version":    runtime.Version(),
		"os":            runtime.GOOS,
		"arch":          runtime.GOARCH,
		"num_cpu":       fmt.Sprintf("%d", runtime.NumCPU()),
		"helix_version": version.GetFullVersion(),
		"timestamp":     time.Now().Format(time.RFC3339),
		"goroutines":    fmt.Sprintf("%d", runtime.NumGoroutine()),
	}
}

// collectRecentLogs collects recent log entries
func collectRecentLogs(sessionID string, limit int) string {
	// Try to read from standard log locations
	logLocations := []string{
		filepath.Join(os.TempDir(), "helixcode", "logs", fmt.Sprintf("session_%s.log", sessionID)),
		filepath.Join(os.Getenv("HOME"), ".helixcode", "logs", fmt.Sprintf("session_%s.log", sessionID)),
		filepath.Join("logs", fmt.Sprintf("session_%s.log", sessionID)),
		filepath.Join("logs", "helixcode.log"),
	}

	var logContent strings.Builder
	logContent.WriteString(fmt.Sprintf("=== Recent Logs (Session: %s, Limit: %d) ===\n\n", sessionID, limit))

	foundLogs := false
	for _, logPath := range logLocations {
		if data, err := os.ReadFile(logPath); err == nil {
			foundLogs = true
			lines := strings.Split(string(data), "\n")

			// Get last N lines
			start := 0
			if len(lines) > limit {
				start = len(lines) - limit
			}

			logContent.WriteString(fmt.Sprintf("--- Log file: %s ---\n", logPath))
			for i := start; i < len(lines); i++ {
				if strings.TrimSpace(lines[i]) != "" {
					logContent.WriteString(lines[i])
					logContent.WriteString("\n")
				}
			}
			logContent.WriteString("\n")
			break // Found logs, no need to check other locations
		}
	}

	if !foundLogs {
		// Fall back to in-memory logging if available
		logger := logging.NewLoggerWithName("reportbug")
		logContent.WriteString("Note: No log files found. Log entries may be in memory or stdout.\n")
		logContent.WriteString(fmt.Sprintf("Logger configured: %s\n", logger.GetName()))
		logContent.WriteString("To enable file logging, configure log output in helixcode config.\n")
	}

	return logContent.String()
}

// formatBugReport formats the bug report for GitHub
func formatBugReport(title, description string, systemInfo map[string]string, logs string, history []commands.ChatMessage) string {
	var sb strings.Builder

	sb.WriteString("## Description\n\n")
	sb.WriteString(description)
	sb.WriteString("\n\n")

	sb.WriteString("## System Information\n\n")
	sb.WriteString("```\n")
	for key, value := range systemInfo {
		sb.WriteString(fmt.Sprintf("%s: %s\n", key, value))
	}
	sb.WriteString("```\n\n")

	sb.WriteString("## Reproduction Steps\n\n")
	sb.WriteString(extractReproductionSteps(history))
	sb.WriteString("\n\n")

	if logs != "" {
		sb.WriteString("## Recent Logs\n\n")
		sb.WriteString("```\n")
		sb.WriteString(logs)
		sb.WriteString("\n```\n\n")
	}

	sb.WriteString("## Expected Behavior\n\n")
	sb.WriteString("[Please describe what you expected to happen]\n\n")

	sb.WriteString("## Actual Behavior\n\n")
	sb.WriteString("[Please describe what actually happened]\n\n")

	sb.WriteString("---\n")
	sb.WriteString("*Generated by HelixCode /reportbug command*\n")

	return sb.String()
}

// extractReproductionSteps attempts to extract reproduction steps from chat history
func extractReproductionSteps(history []commands.ChatMessage) string {
	if len(history) == 0 {
		return "1. [Please describe the steps to reproduce]\n"
	}

	// Take last 5 user messages as potential reproduction steps
	steps := make([]string, 0, 5)
	count := 0
	for i := len(history) - 1; i >= 0 && count < 5; i-- {
		msg := history[i]
		if msg.Role == "user" {
			// Truncate long messages
			content := msg.Content
			if len(content) > 200 {
				content = content[:197] + "..."
			}
			steps = append([]string{content}, steps...)
			count++
		}
	}

	if len(steps) == 0 {
		return "1. [Please describe the steps to reproduce]\n"
	}

	var sb strings.Builder
	for i, step := range steps {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
	}
	return sb.String()
}
