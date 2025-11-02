package llm

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ReasoningRequest represents a request for reasoning-based generation
type ReasoningRequest struct {
	ID            uuid.UUID
	Prompt        string
	Tools         []ReasoningTool
	ReasoningType ReasoningType
	MaxSteps      int
	Temperature   float64
	Context       map[string]interface{}
	Constraints   []string
}

// ReasoningResponse represents the response from reasoning-based generation
type ReasoningResponse struct {
	ID             uuid.UUID
	FinalAnswer    string
	ReasoningSteps []ReasoningStep
	ToolsUsed      []string
	Duration       time.Duration
	Confidence     float64
	Error          string
}

// ReasoningStep represents a single step in the reasoning process
type ReasoningStep struct {
	StepNumber int
	Thought    string
	Action     string
	ToolCall   *ReasoningToolCall
	Result     interface{}
	Confidence float64
}

// ReasoningToolCall represents a call to a tool during reasoning
type ReasoningToolCall struct {
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ReasoningType defines different types of reasoning approaches
type ReasoningType string

const (
	ReasoningTypeChainOfThought ReasoningType = "chain_of_thought"
	ReasoningTypeTreeOfThoughts ReasoningType = "tree_of_thoughts"
	ReasoningTypeSelfReflection ReasoningType = "self_reflection"
	ReasoningTypeProgressive    ReasoningType = "progressive"
)

// ReasoningTool represents a tool that can be used during reasoning
type ReasoningTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Handler     ReasoningToolHandler   `json:"-"`
}

// ReasoningToolHandler is the function signature for tool execution in reasoning
type ReasoningToolHandler func(ctx context.Context, args map[string]interface{}) (interface{}, error)

// ReasoningEngine handles advanced reasoning capabilities
type ReasoningEngine struct {
	provider    Provider
	tools       map[string]ReasoningTool
	maxSteps    int
	temperature float64
}

// NewReasoningEngine creates a new reasoning engine
func NewReasoningEngine(provider Provider) *ReasoningEngine {
	return &ReasoningEngine{
		provider:    provider,
		tools:       make(map[string]ReasoningTool),
		maxSteps:    10,
		temperature: 0.7,
	}
}

// RegisterTool registers a tool with the reasoning engine
func (e *ReasoningEngine) RegisterTool(tool ReasoningTool) error {
	if tool.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	if tool.Handler == nil {
		return fmt.Errorf("tool handler cannot be nil")
	}
	if _, exists := e.tools[tool.Name]; exists {
		return fmt.Errorf("tool %s already registered", tool.Name)
	}
	e.tools[tool.Name] = tool
	log.Printf("Reasoning tool registered: %s", tool.Name)
	return nil
}

// GenerateWithReasoning performs reasoning-based generation
func (e *ReasoningEngine) GenerateWithReasoning(ctx context.Context, req ReasoningRequest) (*ReasoningResponse, error) {
	startTime := time.Now()
	response := &ReasoningResponse{
		ID:             uuid.New(),
		ReasoningSteps: []ReasoningStep{},
		ToolsUsed:      []string{},
	}

	// Validate request
	if err := e.validateRequest(req); err != nil {
		response.Error = err.Error()
		return response, err
	}

	// Execute reasoning based on type
	var err error
	switch req.ReasoningType {
	case ReasoningTypeChainOfThought:
		err = e.executeChainOfThought(ctx, req, response)
	case ReasoningTypeTreeOfThoughts:
		err = e.executeTreeOfThoughts(ctx, req, response)
	case ReasoningTypeSelfReflection:
		err = e.executeSelfReflection(ctx, req, response)
	case ReasoningTypeProgressive:
		err = e.executeProgressiveReasoning(ctx, req, response)
	default:
		err = fmt.Errorf("unsupported reasoning type: %s", req.ReasoningType)
	}

	response.Duration = time.Since(startTime)
	if err != nil {
		response.Error = err.Error()
	}

	return response, err
}

// executeChainOfThought implements chain-of-thought reasoning
func (e *ReasoningEngine) executeChainOfThought(ctx context.Context, req ReasoningRequest, response *ReasoningResponse) error {
	currentThought := req.Prompt
	step := 1

	for step <= req.MaxSteps {
		// Generate next thought step
		thoughtPrompt := e.buildChainOfThoughtPrompt(currentThought, step, req.MaxSteps)
		thought, err := e.generateThought(ctx, thoughtPrompt, req.Temperature)
		if err != nil {
			return fmt.Errorf("failed to generate thought at step %d: %v", step, err)
		}

		// Check if we have a final answer
		if e.isFinalAnswer(thought) {
			response.FinalAnswer = e.extractFinalAnswer(thought)
			break
		}

		// Check if we need to use tools
		toolCall, shouldUseTool := e.shouldUseTool(thought)
		var result interface{}
		if shouldUseTool {
			result, err = e.executeTool(ctx, toolCall)
			if err != nil {
				log.Printf("Tool execution failed: %v", err)
				// Continue reasoning even if tool fails
				result = fmt.Sprintf("Tool error: %v", err)
			}
			response.ToolsUsed = append(response.ToolsUsed, toolCall.ToolName)
		}

		// Record reasoning step
		stepRecord := ReasoningStep{
			StepNumber: step,
			Thought:    thought,
			Action:     e.determineAction(thought, shouldUseTool),
			ToolCall:   toolCall,
			Result:     result,
			Confidence: e.calculateConfidence(thought),
		}
		response.ReasoningSteps = append(response.ReasoningSteps, stepRecord)

		// Update current thought with result
		if shouldUseTool && result != nil {
			currentThought = fmt.Sprintf("%s\nTool Result: %v", thought, result)
		} else {
			currentThought = thought
		}

		step++
	}

	// If no final answer found, use the last thought
	if response.FinalAnswer == "" && len(response.ReasoningSteps) > 0 {
		lastStep := response.ReasoningSteps[len(response.ReasoningSteps)-1]
		response.FinalAnswer = e.extractFinalAnswer(lastStep.Thought)
	}

	return nil
}

// Helper methods

func (e *ReasoningEngine) validateRequest(req ReasoningRequest) error {
	if req.Prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}
	if req.MaxSteps <= 0 {
		return fmt.Errorf("max steps must be positive")
	}
	if req.Temperature < 0 || req.Temperature > 2.0 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	return nil
}

func (e *ReasoningEngine) buildChainOfThoughtPrompt(currentThought string, step, maxSteps int) string {
	return fmt.Sprintf(`
Current reasoning step %d/%d:
%s

Think step by step. If you need to use a tool, specify which one and why.
If you have reached a final conclusion, state it clearly starting with "FINAL ANSWER:".
Next step:`, step, maxSteps, currentThought)
}

func (e *ReasoningEngine) generateThought(ctx context.Context, prompt string, temperature float64) (string, error) {
	genReq := &LLMRequest{
		Model:       "default",
		Messages:    []Message{{Role: "user", Content: prompt}},
		MaxTokens:   500,
		Temperature: temperature,
		Stream:      false,
	}

	resp, err := e.provider.Generate(ctx, genReq)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

func (e *ReasoningEngine) isFinalAnswer(thought string) bool {
	return strings.Contains(strings.ToLower(thought), "final answer:")
}

func (e *ReasoningEngine) extractFinalAnswer(thought string) string {
	if idx := strings.Index(strings.ToLower(thought), "final answer:"); idx != -1 {
		return strings.TrimSpace(thought[idx+len("final answer:"):])
	}
	return thought
}

func (e *ReasoningEngine) shouldUseTool(thought string) (*ReasoningToolCall, bool) {
	// Simple heuristic to detect tool usage
	for toolName := range e.tools {
		if strings.Contains(strings.ToLower(thought), strings.ToLower(toolName)) {
			return &ReasoningToolCall{
				ToolName:  toolName,
				Arguments: make(map[string]interface{}),
			}, true
		}
	}
	return nil, false
}

func (e *ReasoningEngine) executeTool(ctx context.Context, toolCall *ReasoningToolCall) (interface{}, error) {
	tool, exists := e.tools[toolCall.ToolName]
	if !exists {
		return nil, fmt.Errorf("tool not found: %s", toolCall.ToolName)
	}

	return tool.Handler(ctx, toolCall.Arguments)
}

func (e *ReasoningEngine) determineAction(thought string, usedTool bool) string {
	if usedTool {
		return "tool_execution"
	}
	if e.isFinalAnswer(thought) {
		return "final_answer"
	}
	return "reasoning_step"
}

func (e *ReasoningEngine) calculateConfidence(thought string) float64 {
	// Simple confidence calculation based on thought characteristics
	confidence := 0.5

	// Higher confidence for longer, more detailed thoughts
	if len(thought) > 100 {
		confidence += 0.2
	}

	// Higher confidence for thoughts that reference specific facts
	if strings.Contains(strings.ToLower(thought), "because") ||
		strings.Contains(strings.ToLower(thought), "therefore") ||
		strings.Contains(strings.ToLower(thought), "thus") {
		confidence += 0.2
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// Simplified implementations for other reasoning types

func (e *ReasoningEngine) executeTreeOfThoughts(ctx context.Context, req ReasoningRequest, response *ReasoningResponse) error {
	// For now, fall back to chain of thought
	return e.executeChainOfThought(ctx, req, response)
}

func (e *ReasoningEngine) executeSelfReflection(ctx context.Context, req ReasoningRequest, response *ReasoningResponse) error {
	// For now, fall back to chain of thought
	return e.executeChainOfThought(ctx, req, response)
}

func (e *ReasoningEngine) executeProgressiveReasoning(ctx context.Context, req ReasoningRequest, response *ReasoningResponse) error {
	// For now, fall back to chain of thought
	return e.executeChainOfThought(ctx, req, response)
}
