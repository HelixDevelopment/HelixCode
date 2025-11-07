package commands

import (
	"context"
	"time"
)

// Command represents a slash command
type Command interface {
	// Name returns the command name (without /)
	Name() string

	// Aliases returns alternative names for the command
	Aliases() []string

	// Description returns a short description
	Description() string

	// Usage returns usage information
	Usage() string

	// Execute runs the command
	Execute(ctx context.Context, cmdCtx *CommandContext) (*CommandResult, error)
}

// CommandContext contains context for command execution
type CommandContext struct {
	UserID      string                 `json:"user_id"`
	SessionID   string                 `json:"session_id"`
	ProjectID   string                 `json:"project_id"`
	Args        []string               `json:"args"`
	Flags       map[string]string      `json:"flags"`
	RawInput    string                 `json:"raw_input"`
	ChatHistory []ChatMessage          `json:"chat_history"`
	WorkingDir  string                 `json:"working_dir"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ChatMessage represents a message in chat history
type ChatMessage struct {
	Role      string    `json:"role"` // user, assistant, system
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// CommandResult contains the command execution result
type CommandResult struct {
	Success     bool                   `json:"success"`
	Message     string                 `json:"message"`
	Data        interface{}            `json:"data,omitempty"`
	Actions     []Action               `json:"actions,omitempty"`
	ShouldReply bool                   `json:"should_reply"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Action represents an action to be taken after command execution
type Action struct {
	Type string                 `json:"type"` // create_task, switch_mode, update_context, etc.
	Data map[string]interface{} `json:"data"`
}

// CommandError represents a command execution error
type CommandError struct {
	Command string
	Message string
	Err     error
}

func (e *CommandError) Error() string {
	if e.Err != nil {
		return e.Command + ": " + e.Message + ": " + e.Err.Error()
	}
	return e.Command + ": " + e.Message
}
