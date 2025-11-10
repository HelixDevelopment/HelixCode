package commands

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Executor executes commands
type Executor struct {
	registry *Registry
	parser   *Parser
}

// NewExecutor creates a new command executor
func NewExecutor(registry *Registry) *Executor {
	return &Executor{
		registry: registry,
		parser:   NewParser(),
	}
}

// Execute parses and executes a command
func (e *Executor) Execute(ctx context.Context, input string, cmdCtx *CommandContext) (*CommandResult, error) {
	// Parse the command
	commandName, args, flags, isCommand := e.parser.Parse(input)
	if !isCommand {
		return nil, &CommandError{
			Command: input,
			Message: "not a valid command",
		}
	}

	// Get the command
	cmd, exists := e.registry.Get(commandName)
	if !exists {
		return nil, &CommandError{
			Command: commandName,
			Message: "command not found",
		}
	}

	// Prepare command context
	if cmdCtx == nil {
		cmdCtx = &CommandContext{}
	}
	cmdCtx.Args = args
	cmdCtx.Flags = flags
	cmdCtx.RawInput = input

	// Execute the command
	start := time.Now()
	log.Printf("Executing command: /%s with args: %v", commandName, args)

	result, err := cmd.Execute(ctx, cmdCtx)

	duration := time.Since(start)
	log.Printf("Command /%s completed in %v (success: %t)", commandName, duration, err == nil && result != nil && result.Success)

	if err != nil {
		return nil, &CommandError{
			Command: commandName,
			Message: "execution failed",
			Err:     err,
		}
	}

	return result, nil
}

// ExecuteWithDefault executes a command with default context
func (e *Executor) ExecuteWithDefault(ctx context.Context, input string) (*CommandResult, error) {
	return e.Execute(ctx, input, &CommandContext{})
}

// CanExecute checks if input is a valid command
func (e *Executor) CanExecute(input string) bool {
	commandName, _, _, isCommand := e.parser.Parse(input)
	if !isCommand {
		return false
	}

	_, exists := e.registry.Get(commandName)
	return exists
}

// GetHelp returns help text for a command
func (e *Executor) GetHelp(commandName string) string {
	if commandName == "" {
		return e.registry.GetAllHelp()
	}
	return e.registry.GetHelp(commandName)
}

// ListCommands returns all available commands
func (e *Executor) ListCommands() []Command {
	return e.registry.List()
}

// Autocomplete provides command name autocompletion
func (e *Executor) Autocomplete(partial string) []string {
	if !e.parser.IsCommand(partial) {
		return nil
	}

	partial = partial[1:]  // Remove leading /
	partial = partial + "" // ensure string

	matches := make([]string, 0)
	names := e.registry.ListNames()

	for _, name := range names {
		if partial == "" || len(partial) < len(name) && name[:len(partial)] == partial {
			matches = append(matches, "/"+name)
		}
	}

	return matches
}

// ValidateContext validates that the command context has required fields
func (e *Executor) ValidateContext(cmdCtx *CommandContext, required []string) error {
	for _, field := range required {
		switch field {
		case "user_id":
			if cmdCtx.UserID == "" {
				return fmt.Errorf("user_id is required")
			}
		case "session_id":
			if cmdCtx.SessionID == "" {
				return fmt.Errorf("session_id is required")
			}
		case "project_id":
			if cmdCtx.ProjectID == "" {
				return fmt.Errorf("project_id is required")
			}
		case "working_dir":
			if cmdCtx.WorkingDir == "" {
				return fmt.Errorf("working_dir is required")
			}
		}
	}
	return nil
}
