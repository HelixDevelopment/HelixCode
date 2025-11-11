package builtin

import (
	"context"
	"strings"
	"testing"

	"dev.helix.code/internal/commands"
)

// TestNewTaskCommand tests the /newtask command
func TestNewTaskCommand(t *testing.T) {
	cmd := NewNewTaskCommand()

	// Test metadata
	if cmd.Name() != "newtask" {
		t.Errorf("Name() = %q, want newtask", cmd.Name())
	}

	aliases := cmd.Aliases()
	expectedAliases := []string{"nt", "task"}
	if len(aliases) != len(expectedAliases) {
		t.Errorf("Aliases() length = %d, want %d", len(aliases), len(expectedAliases))
	}

	// Test execution with valid description
	ctx := context.Background()
	cmdCtx := &commands.CommandContext{
		Args:      []string{"Implement", "auth", "system"},
		Flags:     map[string]string{"priority": "high"},
		UserID:    "user123",
		ProjectID: "project123",
	}

	result, err := cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed")
	}

	if len(result.Actions) == 0 {
		t.Error("Execute() should create actions")
	}

	if result.Actions[0].Type != "create_task" {
		t.Errorf("Action type = %q, want create_task", result.Actions[0].Type)
	}

	// Test execution without description
	cmdCtx.Args = []string{}
	result, err = cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.Success {
		t.Error("Execute() should fail without description")
	}

	// Test link-previous flag
	cmdCtx.Args = []string{"Follow-up", "task"}
	cmdCtx.Flags = map[string]string{"link-previous": "true"}
	cmdCtx.Metadata = map[string]interface{}{"current_task_id": "task-123"}

	result, err = cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed with link-previous")
	}

	// Should have 2 actions: create_task and link_tasks
	if len(result.Actions) != 2 {
		t.Errorf("Execute() with link-previous should create 2 actions, got %d", len(result.Actions))
	}
}

// TestCondenseCommand tests the /condense command
func TestCondenseCommand(t *testing.T) {
	cmd := NewCondenseCommand()

	// Test metadata
	if cmd.Name() != "condense" {
		t.Errorf("Name() = %q, want condense", cmd.Name())
	}

	aliases := cmd.Aliases()
	if len(aliases) != 3 {
		t.Errorf("Aliases() length = %d, want 3", len(aliases))
	}

	// Test execution with chat history
	ctx := context.Background()
	cmdCtx := &commands.CommandContext{
		ChatHistory: []commands.ChatMessage{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi"},
			{Role: "user", Content: "How are you?"},
			{Role: "assistant", Content: "I'm good"},
			{Role: "user", Content: "Great"},
			{Role: "assistant", Content: "Thanks"},
			{Role: "user", Content: "Bye"},
			{Role: "assistant", Content: "Goodbye"},
		},
		SessionID: "session123",
	}

	result, err := cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed")
	}

	if len(result.Actions) == 0 {
		t.Error("Execute() should create actions")
	}

	if result.Actions[0].Type != "condense_history" {
		t.Errorf("Action type = %q, want condense_history", result.Actions[0].Type)
	}

	// Test with empty history
	cmdCtx.ChatHistory = []commands.ChatMessage{}
	result, err = cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.Success {
		t.Error("Execute() should fail with empty history")
	}

	// Test with custom flags
	cmdCtx.ChatHistory = make([]commands.ChatMessage, 20)
	cmdCtx.Flags = map[string]string{
		"keep-last":       "10",
		"preserve-code":   "true",
		"preserve-errors": "true",
		"ratio":           "0.3",
	}

	result, err = cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed with custom flags")
	}
}

// TestNewRuleCommand tests the /newrule command
func TestNewRuleCommand(t *testing.T) {
	cmd := NewNewRuleCommand()

	// Test metadata
	if cmd.Name() != "newrule" {
		t.Errorf("Name() = %q, want newrule", cmd.Name())
	}

	// Test execution with category
	ctx := context.Background()
	cmdCtx := &commands.CommandContext{
		Args: []string{"coding-style"},
		ChatHistory: []commands.ChatMessage{
			{Role: "user", Content: "Always use tabs instead of spaces"},
			{Role: "user", Content: "Never use var, prefer const"},
			{Role: "user", Content: "You should add error handling"},
		},
		WorkingDir: "/home/user/project",
	}

	result, err := cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed")
	}

	if len(result.Actions) == 0 {
		t.Error("Execute() should create actions")
	}

	if result.Actions[0].Type != "generate_rule" {
		t.Errorf("Action type = %q, want generate_rule", result.Actions[0].Type)
	}

	// Test with global flag
	cmdCtx.Flags = map[string]string{"global": "true"}
	result, err = cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed with global flag")
	}

	// Test with custom name
	cmdCtx.Flags = map[string]string{"name": "my-custom-rule"}
	result, err = cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed with custom name")
	}

	// Test pattern analysis
	patterns := analyzeConversationPatterns(cmdCtx.ChatHistory)
	if len(patterns) == 0 {
		t.Error("analyzeConversationPatterns() should find patterns")
	}
}

// TestReportBugCommand tests the /reportbug command
func TestReportBugCommand(t *testing.T) {
	cmd := NewReportBugCommand()

	// Test metadata
	if cmd.Name() != "reportbug" {
		t.Errorf("Name() = %q, want reportbug", cmd.Name())
	}

	aliases := cmd.Aliases()
	if len(aliases) != 2 {
		t.Errorf("Aliases() length = %d, want 2", len(aliases))
	}

	// Test execution with description
	ctx := context.Background()
	cmdCtx := &commands.CommandContext{
		Args:      []string{"LLM", "timeout", "error"},
		SessionID: "session123",
		UserID:    "user123",
		ChatHistory: []commands.ChatMessage{
			{Role: "user", Content: "Generate code"},
			{Role: "assistant", Content: "Error: timeout"},
		},
	}

	result, err := cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed")
	}

	if len(result.Actions) == 0 {
		t.Error("Execute() should create actions")
	}

	if result.Actions[0].Type != "file_bug_report" {
		t.Errorf("Action type = %q, want file_bug_report", result.Actions[0].Type)
	}

	// Test with custom title and labels
	cmdCtx.Flags = map[string]string{
		"title":  "Critical Bug",
		"labels": "bug,critical,urgent",
	}

	result, err = cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed with custom flags")
	}

	// Test auto-submit flag
	cmdCtx.Flags = map[string]string{"auto-submit": "true"}
	result, err = cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed with auto-submit")
	}

	// Test system info collection
	sysInfo := collectSystemInfo()
	if sysInfo["go_version"] == "" {
		t.Error("collectSystemInfo() should include go_version")
	}
	if sysInfo["os"] == "" {
		t.Error("collectSystemInfo() should include os")
	}
}

// TestWorkflowsCommand tests the /workflows command
func TestWorkflowsCommand(t *testing.T) {
	cmd := NewWorkflowsCommand()

	// Test metadata
	if cmd.Name() != "workflows" {
		t.Errorf("Name() = %q, want workflows", cmd.Name())
	}

	aliases := cmd.Aliases()
	if len(aliases) != 2 {
		t.Errorf("Aliases() length = %d, want 2", len(aliases))
	}

	ctx := context.Background()

	// Test list workflows (default)
	cmdCtx := &commands.CommandContext{
		ProjectID:  "project123",
		WorkingDir: "/home/user/project",
	}

	result, err := cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed for list")
	}

	if result.Actions[0].Type != "list_workflows" {
		t.Errorf("Action type = %q, want list_workflows", result.Actions[0].Type)
	}

	// Test execute workflow
	cmdCtx.Args = []string{"planning"}
	cmdCtx.SessionID = "session123"

	result, err = cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed for workflow execution")
	}

	if result.Actions[0].Type != "execute_workflow" {
		t.Errorf("Action type = %q, want execute_workflow", result.Actions[0].Type)
	}

	// Test invalid workflow
	cmdCtx.Args = []string{"invalid-workflow"}
	result, err = cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.Success {
		t.Error("Execute() should fail for invalid workflow")
	}

	// Test with parameters
	cmdCtx.Args = []string{"testing"}
	cmdCtx.Flags = map[string]string{"params": "unit,integration"}
	result, err = cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed with parameters")
	}

	// Test async execution
	cmdCtx.Flags = map[string]string{"async": "true"}
	result, err = cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed with async flag")
	}

	// Test status check
	cmdCtx.Args = []string{}
	cmdCtx.Flags = map[string]string{"status": "workflow-123"}
	result, err = cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed for status check")
	}

	if result.Actions[0].Type != "check_workflow_status" {
		t.Errorf("Action type = %q, want check_workflow_status", result.Actions[0].Type)
	}

	// Test cancel workflow
	cmdCtx.Flags = map[string]string{"cancel": "workflow-123"}
	cmdCtx.UserID = "user123"
	result, err = cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed for cancel")
	}

	if result.Actions[0].Type != "cancel_workflow" {
		t.Errorf("Action type = %q, want cancel_workflow", result.Actions[0].Type)
	}
}

// TestDeepPlanningCommand tests the /deepplanning command
func TestDeepPlanningCommand(t *testing.T) {
	cmd := NewDeepPlanningCommand()

	// Test metadata
	if cmd.Name() != "deepplanning" {
		t.Errorf("Name() = %q, want deepplanning", cmd.Name())
	}

	aliases := cmd.Aliases()
	if len(aliases) != 3 {
		t.Errorf("Aliases() length = %d, want 3", len(aliases))
	}

	ctx := context.Background()

	// Test execution with topic
	cmdCtx := &commands.CommandContext{
		Args:       []string{"authentication", "system"},
		SessionID:  "session123",
		ProjectID:  "project123",
		WorkingDir: "/home/user/project",
		ChatHistory: []commands.ChatMessage{
			{Role: "user", Content: "We need OAuth support"},
			{Role: "user", Content: "Should use JWT tokens"},
		},
	}

	result, err := cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed")
	}

	if len(result.Actions) == 0 {
		t.Error("Execute() should create actions")
	}

	if result.Actions[0].Type != "start_deep_planning" {
		t.Errorf("Action type = %q, want start_deep_planning", result.Actions[0].Type)
	}

	// Test without topic (should fail)
	cmdCtx.Args = []string{}
	cmdCtx.Flags = map[string]string{}
	result, err = cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.Success {
		t.Error("Execute() should fail without topic or resume flag")
	}

	// Test with custom depth
	cmdCtx.Args = []string{"microservices"}
	cmdCtx.Flags = map[string]string{"depth": "5"}
	result, err = cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed with custom depth")
	}

	// Test with output file
	cmdCtx.Flags = map[string]string{"output": "plan.md"}
	result, err = cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed with output file")
	}

	// Test with diagrams
	cmdCtx.Flags = map[string]string{"include-diagrams": "true"}
	result, err = cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed with diagrams")
	}

	// Test with focus areas
	cmdCtx.Flags = map[string]string{"focus": "architecture,security,performance"}
	result, err = cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed with focus areas")
	}

	// Test with constraints
	cmdCtx.Flags = map[string]string{"constraints": "budget=low,timeline=2weeks"}
	result, err = cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed with constraints")
	}

	// Test resume
	cmdCtx.Args = []string{}
	cmdCtx.Flags = map[string]string{"resume": "plan-123"}
	cmdCtx.UserID = "user123"
	result, err = cmd.Execute(ctx, cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed for resume")
	}

	if result.Actions[0].Type != "resume_deep_planning" {
		t.Errorf("Action type = %q, want resume_deep_planning", result.Actions[0].Type)
	}
}

// TestRegisterBuiltinCommands tests command registration
func TestRegisterBuiltinCommands(t *testing.T) {
	registry := commands.NewRegistry()

	RegisterBuiltinCommands(registry)

	// Test that all commands are registered
	expectedCommands := GetBuiltinCommandNames()
	for _, name := range expectedCommands {
		_, exists := registry.Get(name)
		if !exists {
			t.Errorf("Command %q should be registered", name)
		}
	}

	// Test that all aliases work
	aliases := GetBuiltinCommandAliases()
	for alias := range aliases {
		_, exists := registry.Get(alias)
		if !exists {
			t.Errorf("Alias %q should be registered", alias)
		}
	}
}

// TestExtractFilesFromHistory tests file extraction
func TestExtractFilesFromHistory(t *testing.T) {
	history := []commands.ChatMessage{
		{Content: "Check internal/auth/handler.go"},
		{Content: "Also look at cmd/server/main.go"},
		{Content: "The config/config.yaml needs updating"},
		{Content: "Just some text without files"},
	}

	files := extractFilesFromHistory(history)

	if len(files) == 0 {
		t.Error("extractFilesFromHistory() should find files")
	}

	// Should find at least the .go files
	foundGo := false
	for _, file := range files {
		if file == "internal/auth/handler.go" || file == "cmd/server/main.go" {
			foundGo = true
			break
		}
	}

	if !foundGo {
		t.Error("extractFilesFromHistory() should find .go files")
	}
}

// ========================================
// Description and Usage Tests
// ========================================

func TestCondenseCommand_Description(t *testing.T) {
	cmd := &CondenseCommand{}
	desc := cmd.Description()

	if desc == "" {
		t.Error("Description() should not return empty string")
	}
	if !strings.Contains(strings.ToLower(desc), "condense") && !strings.Contains(strings.ToLower(desc), "summarize") {
		t.Error("Description() should mention condense or summarize")
	}
}

func TestCondenseCommand_Usage(t *testing.T) {
	cmd := &CondenseCommand{}
	usage := cmd.Usage()

	if usage == "" {
		t.Error("Usage() should not return empty string")
	}
	if !strings.Contains(usage, "/condense") {
		t.Error("Usage() should mention /condense command")
	}
}

func TestDeepPlanningCommand_Description(t *testing.T) {
	cmd := &DeepPlanningCommand{}
	desc := cmd.Description()

	if desc == "" {
		t.Error("Description() should not return empty string")
	}
	if !strings.Contains(strings.ToLower(desc), "plan") {
		t.Error("Description() should mention planning")
	}
}

func TestDeepPlanningCommand_Usage(t *testing.T) {
	cmd := &DeepPlanningCommand{}
	usage := cmd.Usage()

	if usage == "" {
		t.Error("Usage() should not return empty string")
	}
	if !strings.Contains(usage, "/deepplanning") {
		t.Error("Usage() should mention /deepplanning command")
	}
}

func TestNewRuleCommand_Description(t *testing.T) {
	cmd := &NewRuleCommand{}
	desc := cmd.Description()

	if desc == "" {
		t.Error("Description() should not return empty string")
	}
	if !strings.Contains(strings.ToLower(desc), "rule") {
		t.Error("Description() should mention rule")
	}
}

func TestNewRuleCommand_Usage(t *testing.T) {
	cmd := &NewRuleCommand{}
	usage := cmd.Usage()

	if usage == "" {
		t.Error("Usage() should not return empty string")
	}
	if !strings.Contains(usage, "/newrule") {
		t.Error("Usage() should mention /newrule command")
	}
}

func TestNewTaskCommand_Description(t *testing.T) {
	cmd := &NewTaskCommand{}
	desc := cmd.Description()

	if desc == "" {
		t.Error("Description() should not return empty string")
	}
	if !strings.Contains(strings.ToLower(desc), "task") {
		t.Error("Description() should mention task")
	}
}

func TestNewTaskCommand_Usage(t *testing.T) {
	cmd := &NewTaskCommand{}
	usage := cmd.Usage()

	if usage == "" {
		t.Error("Usage() should not return empty string")
	}
	if !strings.Contains(usage, "/newtask") {
		t.Error("Usage() should mention /newtask command")
	}
}

func TestReportBugCommand_Description(t *testing.T) {
	cmd := &ReportBugCommand{}
	desc := cmd.Description()

	if desc == "" {
		t.Error("Description() should not return empty string")
	}
	if !strings.Contains(strings.ToLower(desc), "bug") {
		t.Error("Description() should mention bug")
	}
}

func TestReportBugCommand_Usage(t *testing.T) {
	cmd := &ReportBugCommand{}
	usage := cmd.Usage()

	if usage == "" {
		t.Error("Usage() should not return empty string")
	}
	if !strings.Contains(usage, "/reportbug") {
		t.Error("Usage() should mention /reportbug command")
	}
}

func TestWorkflowsCommand_Description(t *testing.T) {
	cmd := &WorkflowsCommand{}
	desc := cmd.Description()

	if desc == "" {
		t.Error("Description() should not return empty string")
	}
	if !strings.Contains(strings.ToLower(desc), "workflow") {
		t.Error("Description() should mention workflow")
	}
}

func TestWorkflowsCommand_Usage(t *testing.T) {
	cmd := &WorkflowsCommand{}
	usage := cmd.Usage()

	if usage == "" {
		t.Error("Usage() should not return empty string")
	}
	if !strings.Contains(usage, "/workflows") {
		t.Error("Usage() should mention /workflows command")
	}
}
