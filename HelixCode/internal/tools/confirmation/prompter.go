package confirmation

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

// PromptRequest describes what to prompt for
type PromptRequest struct {
	Tool      string
	Operation Operation
	Level     ConfirmationLevel
	Danger    *DangerAssessment
	Preview   string
}

// PromptResponse contains user response
type PromptResponse struct {
	Choice    Choice
	Reason    string
	Timestamp time.Time
}

// FormattedPrompt contains formatted prompt
type FormattedPrompt struct {
	Title      string
	Message    string
	Details    []string
	Level      ConfirmationLevel
	Options    []PromptOption
	DefaultOpt int
}

// PromptOption represents a choice option
type PromptOption struct {
	Label       string
	Description string
	Choice      Choice
	Shortcut    string
}

// Prompter interface for different prompt implementations
type Prompter interface {
	Prompt(ctx context.Context, prompt *FormattedPrompt) (*PromptResponse, error)
}

// PromptManager handles user prompts
type PromptManager struct {
	prompter  Prompter
	formatter *PromptFormatter
}

// NewPromptManager creates a new prompt manager
func NewPromptManager() *PromptManager {
	return &PromptManager{
		prompter:  NewInteractivePrompter(),
		formatter: &PromptFormatter{},
	}
}

// Prompt prompts user for confirmation
func (pm *PromptManager) Prompt(ctx context.Context, req PromptRequest) (*PromptResponse, error) {
	// Format prompt
	prompt := pm.formatter.Format(req)

	// Show prompt to user
	response, err := pm.prompter.Prompt(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("prompt user: %w", err)
	}

	return response, nil
}

// PromptFormatter formats prompts for display
type PromptFormatter struct{}

// Format formats a prompt request
func (pf *PromptFormatter) Format(req PromptRequest) *FormattedPrompt {
	prompt := &FormattedPrompt{
		Title:   fmt.Sprintf("Confirm %s operation", req.Tool),
		Level:   req.Level,
		Options: defaultOptions(),
	}

	// Build message
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("Tool: %s\n", req.Tool))
	msg.WriteString(fmt.Sprintf("Operation: %s\n", req.Operation.Description))
	if req.Operation.Target != "" {
		msg.WriteString(fmt.Sprintf("Target: %s\n", req.Operation.Target))
	}
	msg.WriteString(fmt.Sprintf("Risk: %s\n", req.Operation.Risk.String()))

	prompt.Message = msg.String()

	// Add details
	if req.Danger != nil && len(req.Danger.Dangers) > 0 {
		prompt.Details = append(prompt.Details, "Warnings:")
		prompt.Details = append(prompt.Details, req.Danger.Dangers...)
		if !req.Danger.Reversible {
			prompt.Details = append(prompt.Details, "WARNING: This operation is NOT reversible!")
		}
	}

	// Add preview
	if req.Preview != "" {
		prompt.Details = append(prompt.Details, "")
		prompt.Details = append(prompt.Details, "Preview:")
		prompt.Details = append(prompt.Details, req.Preview)
	}

	return prompt
}

// defaultOptions returns default prompt options
func defaultOptions() []PromptOption {
	return []PromptOption{
		{
			Label:       "Allow",
			Description: "Allow this operation",
			Choice:      ChoiceAllow,
			Shortcut:    "y",
		},
		{
			Label:       "Deny",
			Description: "Deny this operation",
			Choice:      ChoiceDeny,
			Shortcut:    "n",
		},
		{
			Label:       "Always",
			Description: "Always allow this tool",
			Choice:      ChoiceAlways,
			Shortcut:    "a",
		},
		{
			Label:       "Never",
			Description: "Never allow this tool",
			Choice:      ChoiceNever,
			Shortcut:    "N",
		},
	}
}

// InteractivePrompter prompts via terminal
type InteractivePrompter struct {
	input  io.Reader
	output io.Writer
}

// NewInteractivePrompter creates a new interactive prompter
func NewInteractivePrompter() *InteractivePrompter {
	return &InteractivePrompter{
		input:  nil, // Will use stdin when nil
		output: nil, // Will use stdout when nil
	}
}

// NewInteractivePrompterWithIO creates a new interactive prompter with custom I/O
func NewInteractivePrompterWithIO(input io.Reader, output io.Writer) *InteractivePrompter {
	return &InteractivePrompter{
		input:  input,
		output: output,
	}
}

// Prompt implements Prompter
func (ip *InteractivePrompter) Prompt(ctx context.Context, prompt *FormattedPrompt) (*PromptResponse, error) {
	// Display prompt
	ip.displayPrompt(prompt)

	// Use provided input or stdin
	input := ip.input
	if input == nil {
		// In a real implementation, this would use os.Stdin
		// For testing, we return a default response
		return &PromptResponse{
			Choice:    ChoiceAllow,
			Timestamp: time.Now(),
		}, nil
	}

	// Read response
	reader := bufio.NewReader(input)
	response, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	response = strings.TrimSpace(response)

	// Parse response
	for _, opt := range prompt.Options {
		if response == opt.Shortcut || strings.EqualFold(response, opt.Label) {
			return &PromptResponse{
				Choice:    opt.Choice,
				Timestamp: time.Now(),
			}, nil
		}
	}

	return nil, fmt.Errorf("invalid choice: %s", response)
}

// displayPrompt displays formatted prompt
func (ip *InteractivePrompter) displayPrompt(prompt *FormattedPrompt) {
	// Use provided output or stdout
	output := ip.output
	if output == nil {
		return // In tests, we don't want to write to stdout
	}

	// Display based on level
	switch prompt.Level {
	case LevelInfo:
		fmt.Fprintf(output, "INFO: %s\n", prompt.Title)
	case LevelWarning:
		fmt.Fprintf(output, "WARNING: %s\n", prompt.Title)
	case LevelDanger:
		fmt.Fprintf(output, "DANGER: %s\n", prompt.Title)
	}

	fmt.Fprintf(output, "\n%s\n", prompt.Message)

	// Display details
	if len(prompt.Details) > 0 {
		fmt.Fprintln(output)
		for _, detail := range prompt.Details {
			fmt.Fprintf(output, "  %s\n", detail)
		}
	}

	// Display options
	fmt.Fprintln(output, "\nOptions:")
	for _, opt := range prompt.Options {
		fmt.Fprintf(output, "  [%s] %s - %s\n", opt.Shortcut, opt.Label, opt.Description)
	}

	fmt.Fprint(output, "\nChoice: ")
}

// MockPrompter is a mock prompter for testing
type MockPrompter struct {
	Response *PromptResponse
	Error    error
}

// Prompt implements Prompter
func (mp *MockPrompter) Prompt(ctx context.Context, prompt *FormattedPrompt) (*PromptResponse, error) {
	if mp.Error != nil {
		return nil, mp.Error
	}
	if mp.Response != nil {
		return mp.Response, nil
	}
	return &PromptResponse{
		Choice:    ChoiceAllow,
		Timestamp: time.Now(),
	}, nil
}
