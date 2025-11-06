package tools

import (
	"context"
	"fmt"
	"os"
	"time"

	"dev.helix.code/internal/tools/shell"
)

// ShellTool implements synchronous shell execution
type ShellTool struct {
	registry *ToolRegistry
}

func (t *ShellTool) Name() string { return "shell" }

func (t *ShellTool) Description() string {
	return "Execute a shell command synchronously"
}

func (t *ShellTool) Category() ToolCategory {
	return CategoryShell
}

func (t *ShellTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "Shell command to execute",
			},
			"workdir": map[string]interface{}{
				"type":        "string",
				"description": "Working directory for command execution",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"description": "Timeout in seconds (default: 30)",
			},
			"env": map[string]interface{}{
				"type":        "object",
				"description": "Environment variables",
			},
		},
		Required:    []string{"command"},
		Description: "Execute a shell command synchronously",
	}
}

func (t *ShellTool) Validate(params map[string]interface{}) error {
	if _, ok := params["command"]; !ok {
		return fmt.Errorf("command is required")
	}
	return nil
}

func (t *ShellTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	cmd := &shell.Command{
		ID:      fmt.Sprintf("shell-%d", time.Now().UnixNano()),
		Command: params["command"].(string),
	}

	if workdir, ok := params["workdir"].(string); ok {
		cmd.WorkDir = workdir
	}

	if timeout, ok := params["timeout"].(int); ok {
		cmd.Timeout = time.Duration(timeout) * time.Second
	}

	if env, ok := params["env"].(map[string]string); ok {
		cmd.Env = env
	}

	return t.registry.shell.Execute(ctx, cmd)
}

// ShellBackgroundTool implements asynchronous shell execution
type ShellBackgroundTool struct {
	registry *ToolRegistry
}

func (t *ShellBackgroundTool) Name() string { return "shell_background" }

func (t *ShellBackgroundTool) Description() string {
	return "Execute a shell command asynchronously in the background"
}

func (t *ShellBackgroundTool) Category() ToolCategory {
	return CategoryShell
}

func (t *ShellBackgroundTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "Shell command to execute",
			},
			"workdir": map[string]interface{}{
				"type":        "string",
				"description": "Working directory for command execution",
			},
			"env": map[string]interface{}{
				"type":        "object",
				"description": "Environment variables",
			},
		},
		Required:    []string{"command"},
		Description: "Execute a shell command asynchronously in the background",
	}
}

func (t *ShellBackgroundTool) Validate(params map[string]interface{}) error {
	if _, ok := params["command"]; !ok {
		return fmt.Errorf("command is required")
	}
	return nil
}

func (t *ShellBackgroundTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	cmd := &shell.Command{
		ID:      fmt.Sprintf("bg-%d", time.Now().UnixNano()),
		Command: params["command"].(string),
	}

	if workdir, ok := params["workdir"].(string); ok {
		cmd.WorkDir = workdir
	}

	if env, ok := params["env"].(map[string]string); ok {
		cmd.Env = env
	}

	return t.registry.shell.ExecuteAsync(ctx, cmd)
}

// ShellOutputTool retrieves output from a background execution
type ShellOutputTool struct {
	registry *ToolRegistry
}

func (t *ShellOutputTool) Name() string { return "shell_output" }

func (t *ShellOutputTool) Description() string {
	return "Get output from a background shell execution"
}

func (t *ShellOutputTool) Category() ToolCategory {
	return CategoryShell
}

func (t *ShellOutputTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"execution_id": map[string]interface{}{
				"type":        "string",
				"description": "ID of the background execution",
			},
		},
		Required:    []string{"execution_id"},
		Description: "Get output from a background shell execution",
	}
}

func (t *ShellOutputTool) Validate(params map[string]interface{}) error {
	if _, ok := params["execution_id"]; !ok {
		return fmt.Errorf("execution_id is required")
	}
	return nil
}

func (t *ShellOutputTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	executionID := params["execution_id"].(string)
	return t.registry.shell.GetStatus(executionID)
}

// ShellKillTool kills a running background execution
type ShellKillTool struct {
	registry *ToolRegistry
}

func (t *ShellKillTool) Name() string { return "shell_kill" }

func (t *ShellKillTool) Description() string {
	return "Kill a running background shell execution"
}

func (t *ShellKillTool) Category() ToolCategory {
	return CategoryShell
}

func (t *ShellKillTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"execution_id": map[string]interface{}{
				"type":        "string",
				"description": "ID of the background execution to kill",
			},
			"signal": map[string]interface{}{
				"type":        "string",
				"description": "Signal to send (default: SIGTERM)",
			},
		},
		Required:    []string{"execution_id"},
		Description: "Kill a running background shell execution",
	}
}

func (t *ShellKillTool) Validate(params map[string]interface{}) error {
	if _, ok := params["execution_id"]; !ok {
		return fmt.Errorf("execution_id is required")
	}
	return nil
}

func (t *ShellKillTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	executionID := params["execution_id"].(string)
	signal := os.Signal(os.Interrupt)

	if sigStr, ok := params["signal"].(string); ok {
		switch sigStr {
		case "SIGKILL":
			signal = os.Kill
		case "SIGTERM":
			signal = os.Interrupt
		}
	}

	return nil, t.registry.shell.Kill(executionID, signal)
}
