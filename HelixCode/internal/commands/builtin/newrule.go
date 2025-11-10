package builtin

import (
	"context"
	"fmt"
	"strings"

	"dev.helix.code/internal/commands"
)

// NewRuleCommand generates rules from conversation
type NewRuleCommand struct{}

// NewNewRuleCommand creates a new /newrule command
func NewNewRuleCommand() *NewRuleCommand {
	return &NewRuleCommand{}
}

// Name returns the command name
func (c *NewRuleCommand) Name() string {
	return "newrule"
}

// Aliases returns command aliases
func (c *NewRuleCommand) Aliases() []string {
	return []string{"rule", "guideline"}
}

// Description returns command description
func (c *NewRuleCommand) Description() string {
	return "Generate project rules/guidelines from conversation patterns"
}

// Usage returns usage information
func (c *NewRuleCommand) Usage() string {
	return `/newrule [category] [options]

Analyzes the conversation and generates coding rules or guidelines based on patterns.

Examples:
  /newrule
  /newrule coding-style
  /newrule testing --global
  /newrule "error handling"

Categories:
  coding-style: Code formatting and style preferences
  testing: Testing requirements and patterns
  architecture: Architectural decisions
  documentation: Documentation standards

Flags:
  --global: Save as global rule (default: workspace)
  --name: Custom rule name`
}

// Execute runs the command
func (c *NewRuleCommand) Execute(ctx context.Context, cmdCtx *commands.CommandContext) (*commands.CommandResult, error) {
	// Determine category
	category := "general"
	if len(cmdCtx.Args) > 0 {
		category = strings.Join(cmdCtx.Args, " ")
	}

	// Parse flags
	isGlobal := cmdCtx.Flags["global"] == "true"
	customName := cmdCtx.Flags["name"]

	// Analyze chat history for patterns
	patterns := analyzeConversationPatterns(cmdCtx.ChatHistory)

	// Determine rule scope
	scope := "workspace"
	if isGlobal {
		scope = "global"
	}

	// Generate rule name
	ruleName := customName
	if ruleName == "" {
		ruleName = fmt.Sprintf("%s-rules", strings.ReplaceAll(category, " ", "-"))
	}

	// Create action to generate and save rule
	actions := []commands.Action{
		{
			Type: "generate_rule",
			Data: map[string]interface{}{
				"category":    category,
				"scope":       scope,
				"name":        ruleName,
				"patterns":    patterns,
				"working_dir": cmdCtx.WorkingDir,
			},
		},
	}

	location := fmt.Sprintf(".helixrules/%s.md", ruleName)
	if isGlobal {
		location = fmt.Sprintf("~/Documents/HelixCode/Rules/%s.md", ruleName)
	}

	return &commands.CommandResult{
		Success:     true,
		Message:     fmt.Sprintf("Generating %s rule: %s\nLocation: %s", category, ruleName, location),
		Actions:     actions,
		ShouldReply: true,
		Metadata: map[string]interface{}{
			"category": category,
			"scope":    scope,
			"name":     ruleName,
			"location": location,
		},
	}, nil
}

// analyzeConversationPatterns extracts patterns from chat history
func analyzeConversationPatterns(history []commands.ChatMessage) []string {
	patterns := make([]string, 0)

	// Look for repeated requests or corrections
	corrections := make(map[string]int)
	preferences := make(map[string]bool)

	for _, msg := range history {
		content := strings.ToLower(msg.Content)

		// Detect correction patterns
		if strings.Contains(content, "instead") || strings.Contains(content, "should") ||
			strings.Contains(content, "prefer") || strings.Contains(content, "always") ||
			strings.Contains(content, "never") {
			patterns = append(patterns, msg.Content)
		}

		// Detect style preferences
		if strings.Contains(content, "use ") || strings.Contains(content, "don't use") {
			preferences[msg.Content] = true
		}

		// Detect repeated issues
		if strings.Contains(content, "again") || strings.Contains(content, "still") {
			corrections[msg.Content]++
		}
	}

	// Add frequently corrected items
	for pattern, count := range corrections {
		if count >= 2 {
			patterns = append(patterns, fmt.Sprintf("Repeated issue (x%d): %s", count, pattern))
		}
	}

	return patterns
}
