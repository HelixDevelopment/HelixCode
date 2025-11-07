package commands

import (
	"context"
	"testing"
)

// TestParser tests the command parser
func TestParser(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name          string
		input         string
		expectCommand string
		expectArgs    []string
		expectFlags   map[string]string
		expectValid   bool
	}{
		{
			name:          "simple command",
			input:         "/help",
			expectCommand: "help",
			expectArgs:    []string{},
			expectFlags:   map[string]string{},
			expectValid:   true,
		},
		{
			name:          "command with args",
			input:         "/newtask Implement auth",
			expectCommand: "newtask",
			expectArgs:    []string{"Implement", "auth"},
			expectFlags:   map[string]string{},
			expectValid:   true,
		},
		{
			name:          "command with quoted args",
			input:         `/newtask "Fix bug in worker pool"`,
			expectCommand: "newtask",
			expectArgs:    []string{"Fix bug in worker pool"},
			expectFlags:   map[string]string{},
			expectValid:   true,
		},
		{
			name:          "command with flags",
			input:         "/newtask Fix auth --priority high",
			expectCommand: "newtask",
			expectArgs:    []string{"Fix", "auth"},
			expectFlags:   map[string]string{"priority": "high"},
			expectValid:   true,
		},
		{
			name:          "command with boolean flag",
			input:         "/condense --preserve-code",
			expectCommand: "condense",
			expectArgs:    []string{},
			expectFlags:   map[string]string{"preserve-code": "true"},
			expectValid:   true,
		},
		{
			name:          "command with flag equals syntax",
			input:         "/workflows testing --params=unit",
			expectCommand: "workflows",
			expectArgs:    []string{"testing"},
			expectFlags:   map[string]string{"params": "unit"},
			expectValid:   true,
		},
		{
			name:          "not a command",
			input:         "hello world",
			expectCommand: "",
			expectArgs:    nil,
			expectFlags:   nil,
			expectValid:   false,
		},
		{
			name:          "empty input",
			input:         "",
			expectCommand: "",
			expectArgs:    nil,
			expectFlags:   nil,
			expectValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, args, flags, valid := parser.Parse(tt.input)

			if valid != tt.expectValid {
				t.Errorf("Parse(%q) valid = %v, want %v", tt.input, valid, tt.expectValid)
			}

			if cmd != tt.expectCommand {
				t.Errorf("Parse(%q) command = %q, want %q", tt.input, cmd, tt.expectCommand)
			}

			if tt.expectValid {
				if len(args) != len(tt.expectArgs) {
					t.Errorf("Parse(%q) args length = %d, want %d", tt.input, len(args), len(tt.expectArgs))
				} else {
					for i, arg := range args {
						if arg != tt.expectArgs[i] {
							t.Errorf("Parse(%q) args[%d] = %q, want %q", tt.input, i, arg, tt.expectArgs[i])
						}
					}
				}

				if len(flags) != len(tt.expectFlags) {
					t.Errorf("Parse(%q) flags length = %d, want %d", tt.input, len(flags), len(tt.expectFlags))
				} else {
					for key, val := range tt.expectFlags {
						if flags[key] != val {
							t.Errorf("Parse(%q) flags[%q] = %q, want %q", tt.input, key, flags[key], val)
						}
					}
				}
			}
		})
	}
}

// TestParserIsCommand tests the IsCommand method
func TestParserIsCommand(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		input  string
		expect bool
	}{
		{"/help", true},
		{"/newtask", true},
		{"  /condense  ", true},
		{"hello", false},
		{"", false},
		{"not a command", false},
	}

	for _, tt := range tests {
		result := parser.IsCommand(tt.input)
		if result != tt.expect {
			t.Errorf("IsCommand(%q) = %v, want %v", tt.input, result, tt.expect)
		}
	}
}

// TestParserExtractCommandName tests command name extraction
func TestParserExtractCommandName(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		input  string
		expect string
	}{
		{"/help", "help"},
		{"/newtask Fix bug", "newtask"},
		{"/condense --preserve-code", "condense"},
		{"not a command", ""},
		{"", ""},
	}

	for _, tt := range tests {
		result := parser.ExtractCommandName(tt.input)
		if result != tt.expect {
			t.Errorf("ExtractCommandName(%q) = %q, want %q", tt.input, result, tt.expect)
		}
	}
}

// MockCommand is a mock command for testing
type MockCommand struct {
	name        string
	aliases     []string
	description string
	executed    bool
	result      *CommandResult
	err         error
}

func (m *MockCommand) Name() string {
	return m.name
}

func (m *MockCommand) Aliases() []string {
	return m.aliases
}

func (m *MockCommand) Description() string {
	return m.description
}

func (m *MockCommand) Usage() string {
	return "Mock command usage"
}

func (m *MockCommand) Execute(ctx context.Context, cmdCtx *CommandContext) (*CommandResult, error) {
	m.executed = true
	return m.result, m.err
}

// TestRegistry tests the command registry
func TestRegistry(t *testing.T) {
	registry := NewRegistry()

	// Test registration
	cmd := &MockCommand{
		name:        "test",
		aliases:     []string{"t", "tst"},
		description: "Test command",
	}

	err := registry.Register(cmd)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Test retrieval by name
	retrieved, exists := registry.Get("test")
	if !exists {
		t.Error("Get(test) should exist")
	}
	if retrieved != cmd {
		t.Error("Get(test) returned wrong command")
	}

	// Test retrieval by alias
	retrieved, exists = registry.Get("t")
	if !exists {
		t.Error("Get(t) should exist")
	}
	if retrieved != cmd {
		t.Error("Get(t) returned wrong command")
	}

	// Test duplicate registration
	err = registry.Register(cmd)
	if err == nil {
		t.Error("Register() should error on duplicate")
	}

	// Test list
	commands := registry.List()
	if len(commands) != 1 {
		t.Errorf("List() length = %d, want 1", len(commands))
	}

	// Test list names
	names := registry.ListNames()
	if len(names) != 1 || names[0] != "test" {
		t.Errorf("ListNames() = %v, want [test]", names)
	}

	// Test unregister
	registry.Unregister("test")
	_, exists = registry.Get("test")
	if exists {
		t.Error("Get(test) should not exist after Unregister")
	}
}

// TestExecutor tests the command executor
func TestExecutor(t *testing.T) {
	registry := NewRegistry()
	executor := NewExecutor(registry)

	// Register mock command
	mockCmd := &MockCommand{
		name:    "mock",
		aliases: []string{"m"},
		result: &CommandResult{
			Success: true,
			Message: "Mock executed",
		},
	}
	registry.Register(mockCmd)

	// Test execution
	ctx := context.Background()
	cmdCtx := &CommandContext{
		UserID:    "user123",
		SessionID: "session123",
	}

	result, err := executor.Execute(ctx, "/mock", cmdCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed")
	}

	if !mockCmd.executed {
		t.Error("Command should be executed")
	}

	// Test invalid command
	_, err = executor.Execute(ctx, "/unknown", cmdCtx)
	if err == nil {
		t.Error("Execute() should error on unknown command")
	}

	// Test non-command input
	_, err = executor.Execute(ctx, "not a command", cmdCtx)
	if err == nil {
		t.Error("Execute() should error on non-command")
	}
}

// TestExecutorCanExecute tests command validation
func TestExecutorCanExecute(t *testing.T) {
	registry := NewRegistry()
	executor := NewExecutor(registry)

	mockCmd := &MockCommand{name: "test"}
	registry.Register(mockCmd)

	tests := []struct {
		input  string
		expect bool
	}{
		{"/test", true},
		{"/unknown", false},
		{"not a command", false},
	}

	for _, tt := range tests {
		result := executor.CanExecute(tt.input)
		if result != tt.expect {
			t.Errorf("CanExecute(%q) = %v, want %v", tt.input, result, tt.expect)
		}
	}
}

// TestExecutorAutocomplete tests command autocompletion
func TestExecutorAutocomplete(t *testing.T) {
	registry := NewRegistry()
	executor := NewExecutor(registry)

	registry.Register(&MockCommand{name: "test"})
	registry.Register(&MockCommand{name: "task"})
	registry.Register(&MockCommand{name: "help"})

	tests := []struct {
		input  string
		expect []string
	}{
		{"/t", []string{"/task", "/test"}},
		{"/te", []string{"/test"}},
		{"/ta", []string{"/task"}},
		{"/h", []string{"/help"}},
		{"/", []string{"/help", "/task", "/test"}},
		{"not", nil},
	}

	for _, tt := range tests {
		results := executor.Autocomplete(tt.input)

		if len(results) != len(tt.expect) {
			t.Errorf("Autocomplete(%q) returned %d results, want %d", tt.input, len(results), len(tt.expect))
			continue
		}

		for i, result := range results {
			if result != tt.expect[i] {
				t.Errorf("Autocomplete(%q)[%d] = %q, want %q", tt.input, i, result, tt.expect[i])
			}
		}
	}
}

// TestExecutorValidateContext tests context validation
func TestExecutorValidateContext(t *testing.T) {
	registry := NewRegistry()
	executor := NewExecutor(registry)

	tests := []struct {
		name       string
		ctx        *CommandContext
		required   []string
		shouldFail bool
	}{
		{
			name: "valid context",
			ctx: &CommandContext{
				UserID:    "user123",
				SessionID: "session123",
			},
			required:   []string{"user_id", "session_id"},
			shouldFail: false,
		},
		{
			name: "missing user_id",
			ctx: &CommandContext{
				SessionID: "session123",
			},
			required:   []string{"user_id"},
			shouldFail: true,
		},
		{
			name: "missing session_id",
			ctx: &CommandContext{
				UserID: "user123",
			},
			required:   []string{"session_id"},
			shouldFail: true,
		},
		{
			name:       "empty context",
			ctx:        &CommandContext{},
			required:   []string{"user_id", "session_id", "project_id", "working_dir"},
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.ValidateContext(tt.ctx, tt.required)
			if tt.shouldFail && err == nil {
				t.Error("ValidateContext() should have failed")
			}
			if !tt.shouldFail && err != nil {
				t.Errorf("ValidateContext() unexpected error: %v", err)
			}
		})
	}
}

// TestCommandError tests command error formatting
func TestCommandError(t *testing.T) {
	err := &CommandError{
		Command: "test",
		Message: "failed",
	}

	expected := "test: failed"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

// TestCommandContext tests command context structure
func TestCommandContext(t *testing.T) {
	ctx := &CommandContext{
		UserID:     "user123",
		SessionID:  "session123",
		ProjectID:  "project123",
		Args:       []string{"arg1", "arg2"},
		Flags:      map[string]string{"flag1": "value1"},
		RawInput:   "/test arg1 arg2 --flag1 value1",
		WorkingDir: "/home/user/project",
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}

	if ctx.UserID != "user123" {
		t.Errorf("UserID = %q, want user123", ctx.UserID)
	}

	if len(ctx.Args) != 2 {
		t.Errorf("Args length = %d, want 2", len(ctx.Args))
	}

	if ctx.Flags["flag1"] != "value1" {
		t.Errorf("Flags[flag1] = %q, want value1", ctx.Flags["flag1"])
	}
}

// TestCommandResult tests command result structure
func TestCommandResult(t *testing.T) {
	result := &CommandResult{
		Success: true,
		Message: "Success message",
		Actions: []Action{
			{
				Type: "test_action",
				Data: map[string]interface{}{
					"key": "value",
				},
			},
		},
		ShouldReply: true,
		Metadata: map[string]interface{}{
			"result_key": "result_value",
		},
	}

	if !result.Success {
		t.Error("Success should be true")
	}

	if len(result.Actions) != 1 {
		t.Errorf("Actions length = %d, want 1", len(result.Actions))
	}

	if result.Actions[0].Type != "test_action" {
		t.Errorf("Action type = %q, want test_action", result.Actions[0].Type)
	}

	if !result.ShouldReply {
		t.Error("ShouldReply should be true")
	}
}
