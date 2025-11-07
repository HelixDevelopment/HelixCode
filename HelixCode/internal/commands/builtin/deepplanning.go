package builtin

import (
	"context"
	"fmt"
	"strings"

	"dev.helix.code/internal/commands"
)

// DeepPlanningCommand enters extended planning mode
type DeepPlanningCommand struct{}

// NewDeepPlanningCommand creates a new /deepplanning command
func NewDeepPlanningCommand() *DeepPlanningCommand {
	return &DeepPlanningCommand{}
}

// Name returns the command name
func (c *DeepPlanningCommand) Name() string {
	return "deepplanning"
}

// Aliases returns command aliases
func (c *DeepPlanningCommand) Aliases() []string {
	return []string{"deepplan", "dp", "architect"}
}

// Description returns command description
func (c *DeepPlanningCommand) Description() string {
	return "Enter extended planning mode with detailed analysis and architecture design"
}

// Usage returns usage information
func (c *DeepPlanningCommand) Usage() string {
	return `/deepplanning [topic] [options]

Enters extended planning mode with comprehensive analysis, architecture design,
and detailed implementation planning.

Examples:
  /deepplanning "new authentication system"
  /deepplanning --depth 3 --output plan.md
  /deepplanning microservices --include-diagrams
  /deepplanning --resume plan-123

Planning Phases:
  1. Requirements Analysis: Gather and analyze requirements
  2. Architecture Design: Design system architecture and components
  3. Technology Selection: Choose appropriate technologies and tools
  4. Implementation Planning: Break down into tasks and milestones
  5. Risk Assessment: Identify potential risks and mitigation strategies
  6. Resource Estimation: Estimate time, team size, and resources

Flags:
  --depth: Planning depth (1-5, default: 3)
  --output: Save plan to file (markdown or JSON)
  --include-diagrams: Generate architecture diagrams (ASCII/Mermaid)
  --resume: Resume previous planning session
  --focus: Focus areas (comma-separated: architecture,security,performance,scalability)
  --constraints: Specify constraints (e.g., "budget=low,timeline=2weeks")`
}

// Execute runs the command
func (c *DeepPlanningCommand) Execute(ctx context.Context, cmdCtx *commands.CommandContext) (*commands.CommandResult, error) {
	// Extract planning topic
	topic := strings.Join(cmdCtx.Args, " ")
	if topic == "" && cmdCtx.Flags["resume"] == "" {
		return &commands.CommandResult{
			Success: false,
			Message: "Planning topic is required. Usage: /deepplanning <topic> or /deepplanning --resume <plan-id>",
		}, nil
	}

	// Check for resume
	if resumeID, ok := cmdCtx.Flags["resume"]; ok {
		return c.resumePlanning(resumeID, cmdCtx)
	}

	// Parse planning depth
	depth := 3 // default
	if val, ok := cmdCtx.Flags["depth"]; ok {
		fmt.Sscanf(val, "%d", &depth)
		if depth < 1 {
			depth = 1
		} else if depth > 5 {
			depth = 5
		}
	}

	// Parse output format
	outputFile := cmdCtx.Flags["output"]
	outputFormat := "markdown"
	if outputFile != "" {
		if strings.HasSuffix(outputFile, ".json") {
			outputFormat = "json"
		}
	}

	// Parse focus areas
	focusAreas := []string{"architecture", "implementation"}
	if focus, ok := cmdCtx.Flags["focus"]; ok {
		focusAreas = strings.Split(focus, ",")
		for i, area := range focusAreas {
			focusAreas[i] = strings.TrimSpace(area)
		}
	}

	// Parse constraints
	constraints := parseConstraints(cmdCtx.Flags["constraints"])

	// Include diagrams flag
	includeDiagrams := cmdCtx.Flags["include-diagrams"] == "true"

	// Extract context from chat history
	planningContext := extractPlanningContext(cmdCtx.ChatHistory, topic)

	// Create action to start deep planning
	actions := []commands.Action{
		{
			Type: "start_deep_planning",
			Data: map[string]interface{}{
				"topic":            topic,
				"depth":            depth,
				"focus_areas":      focusAreas,
				"constraints":      constraints,
				"include_diagrams": includeDiagrams,
				"output_file":      outputFile,
				"output_format":    outputFormat,
				"context":          planningContext,
				"session_id":       cmdCtx.SessionID,
				"project_id":       cmdCtx.ProjectID,
				"working_dir":      cmdCtx.WorkingDir,
			},
		},
	}

	message := fmt.Sprintf("Starting deep planning for: %s (depth: %d)", topic, depth)
	if outputFile != "" {
		message += fmt.Sprintf("\nPlan will be saved to: %s", outputFile)
	}

	return &commands.CommandResult{
		Success:     true,
		Message:     message,
		Actions:     actions,
		ShouldReply: true,
		Metadata: map[string]interface{}{
			"topic":       topic,
			"depth":       depth,
			"focus_areas": focusAreas,
			"output_file": outputFile,
		},
	}, nil
}

// resumePlanning resumes a previous planning session
func (c *DeepPlanningCommand) resumePlanning(planID string, cmdCtx *commands.CommandContext) (*commands.CommandResult, error) {
	actions := []commands.Action{
		{
			Type: "resume_deep_planning",
			Data: map[string]interface{}{
				"plan_id":    planID,
				"session_id": cmdCtx.SessionID,
				"user_id":    cmdCtx.UserID,
			},
		},
	}

	return &commands.CommandResult{
		Success:     true,
		Message:     fmt.Sprintf("Resuming deep planning session: %s", planID),
		Actions:     actions,
		ShouldReply: true,
		Metadata: map[string]interface{}{
			"plan_id": planID,
			"resumed": true,
		},
	}, nil
}

// parseConstraints parses constraint string into map
func parseConstraints(constraintStr string) map[string]string {
	constraints := make(map[string]string)

	if constraintStr == "" {
		return constraints
	}

	pairs := strings.Split(constraintStr, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) == 2 {
			constraints[parts[0]] = parts[1]
		}
	}

	return constraints
}

// extractPlanningContext extracts relevant context for planning
func extractPlanningContext(history []commands.ChatMessage, topic string) map[string]interface{} {
	context := make(map[string]interface{})

	context["topic"] = topic
	context["history_size"] = len(history)

	// Extract recent user requirements
	requirements := make([]string, 0)
	for i := len(history) - 1; i >= 0 && len(requirements) < 10; i-- {
		msg := history[i]
		if msg.Role == "user" {
			// Look for requirement-related keywords
			content := strings.ToLower(msg.Content)
			if strings.Contains(content, "need") ||
				strings.Contains(content, "should") ||
				strings.Contains(content, "require") ||
				strings.Contains(content, "must") ||
				strings.Contains(content, "want") {
				requirements = append(requirements, msg.Content)
			}
		}
	}
	context["requirements"] = requirements

	// Extract mentioned technologies
	technologies := extractTechnologies(history)
	if len(technologies) > 0 {
		context["mentioned_technologies"] = technologies
	}

	return context
}

// extractTechnologies extracts technology mentions from history
func extractTechnologies(history []commands.ChatMessage) []string {
	techKeywords := []string{
		"go", "golang", "python", "javascript", "typescript", "rust", "java",
		"postgres", "postgresql", "mysql", "mongodb", "redis",
		"docker", "kubernetes", "aws", "gcp", "azure",
		"react", "vue", "angular", "svelte",
		"gin", "echo", "fiber", "chi",
		"grpc", "graphql", "rest", "websocket",
	}

	found := make(map[string]bool)
	technologies := make([]string, 0)

	for _, msg := range history {
		content := strings.ToLower(msg.Content)
		for _, tech := range techKeywords {
			if strings.Contains(content, tech) && !found[tech] {
				found[tech] = true
				technologies = append(technologies, tech)
			}
		}
	}

	return technologies
}
