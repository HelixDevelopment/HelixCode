package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Enhanced LLM Provider Interface with Tool Calling

// ToolGenerationRequest represents a request for generation with tools
type ToolGenerationRequest struct {
	ID          uuid.UUID              `json:"id"`
	Prompt      string                 `json:"prompt"`
	Tools       []Tool                 `json:"tools"`
	MaxTokens   int                    `json:"max_tokens"`
	Temperature float64                `json:"temperature"`
	Stream      bool                   `json:"stream"`
	Context     map[string]interface{} `json:"context"`
}

// ToolGenerationResponse represents the response from tool-based generation
type ToolGenerationResponse struct {
	ID        uuid.UUID              `json:"id"`
	Text      string                 `json:"text"`
	ToolCalls []ToolCall             `json:"tool_calls"`
	Reasoning string                 `json:"reasoning"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// ToolStreamChunk represents a streaming chunk for tool-based generation
type ToolStreamChunk struct {
	ID        uuid.UUID  `json:"id"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls"`
	Reasoning string     `json:"reasoning"`
	Done      bool       `json:"done"`
	Error     string     `json:"error,omitempty"`
}

// EnhancedLLMProvider extends the base Provider with tool calling capabilities
type EnhancedLLMProvider interface {
	Provider
	GenerateWithTools(ctx context.Context, req ToolGenerationRequest) (*ToolGenerationResponse, error)
	StreamWithTools(ctx context.Context, req ToolGenerationRequest) (<-chan ToolStreamChunk, error)
	ListAvailableTools() []Tool
	RegisterTool(tool Tool) error
}

// ToolCallingProvider implements EnhancedLLMProvider with tool calling support
type ToolCallingProvider struct {
	baseProvider    Provider
	tools           map[string]Tool
	reasoningEngine *ReasoningEngine
}

// NewToolCallingProvider creates a new tool calling provider
func NewToolCallingProvider(baseProvider Provider) *ToolCallingProvider {
	return &ToolCallingProvider{
		baseProvider:    baseProvider,
		tools:           make(map[string]Tool),
		reasoningEngine: NewReasoningEngine(baseProvider),
	}
}

// GenerateWithTools performs generation with tool calling support
func (p *ToolCallingProvider) GenerateWithTools(ctx context.Context, req ToolGenerationRequest) (*ToolGenerationResponse, error) {
	startTime := time.Now()

	// Build tool-enhanced prompt
	enhancedPrompt := p.buildToolEnhancedPrompt(req.Prompt, req.Tools)

	// Generate initial response
	genReq := &LLMRequest{
		Model:       "default",
		Messages:    []Message{{Role: "user", Content: enhancedPrompt}},
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      false,
	}

	resp, err := p.baseProvider.Generate(ctx, genReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate with tools: %v", err)
	}

	// Parse tool calls from response
	toolCalls, reasoning := p.extractToolCallsAndReasoning(resp.Content)

	// Execute tool calls if any
	if len(toolCalls) > 0 {
		results, err := p.executeToolCalls(ctx, toolCalls)
		if err != nil {
			log.Printf("Warning: Some tool calls failed: %v", err)
		}

		// Generate final response with tool results
		finalPrompt := p.buildFinalPrompt(req.Prompt, resp.Content, results)
		genReq.Messages = []Message{{Role: "user", Content: finalPrompt}}

		finalResp, err := p.baseProvider.Generate(ctx, genReq)
		if err != nil {
			return nil, fmt.Errorf("failed to generate final response: %v", err)
		}
		resp = finalResp
	}

	return &ToolGenerationResponse{
		ID:        uuid.New(),
		Text:      resp.Content,
		ToolCalls: toolCalls,
		Reasoning: reasoning,
		Metadata: map[string]interface{}{
			"duration_ms": time.Since(startTime).Milliseconds(),
			"tools_used":  len(toolCalls),
		},
	}, nil
}

// StreamWithTools performs streaming generation with tool calling support
func (p *ToolCallingProvider) StreamWithTools(ctx context.Context, req ToolGenerationRequest) (<-chan ToolStreamChunk, error) {
	ch := make(chan ToolStreamChunk, 100)

	go func() {
		defer close(ch)

		// Build tool-enhanced prompt
		enhancedPrompt := p.buildToolEnhancedPrompt(req.Prompt, req.Tools)

		// Stream initial response
		streamReq := &LLMRequest{
			Model:       "default",
			Messages:    []Message{{Role: "user", Content: enhancedPrompt}},
			MaxTokens:   req.MaxTokens,
			Temperature: req.Temperature,
			Stream:      true,
		}

		streamCh := make(chan LLMResponse, 100)
		err := p.baseProvider.GenerateStream(ctx, streamReq, streamCh)
		if err != nil {
			ch <- ToolStreamChunk{
				ID:    uuid.New(),
				Error: fmt.Sprintf("Failed to start streaming: %v", err),
				Done:  true,
			}
			return
		}

		var fullResponse string
		var toolCalls []ToolCall
		var reasoning string

		for resp := range streamCh {
			// Check for errors (in a real implementation, you'd have error handling)
			// For now, we'll assume no errors in streaming

			fullResponse += resp.Content

			// Send streaming chunk
			ch <- ToolStreamChunk{
				ID:        uuid.New(),
				Content:   resp.Content,
				ToolCalls: []ToolCall{},
				Reasoning: "",
				Done:      false,
			}
		}

		// Parse tool calls after streaming completes
		toolCalls, reasoning = p.extractToolCallsAndReasoning(fullResponse)

		// Execute tool calls if any
		if len(toolCalls) > 0 {
			results, err := p.executeToolCalls(ctx, toolCalls)
			if err != nil {
				log.Printf("Warning: Some tool calls failed: %v", err)
			}

			// Generate final response with tool results
			finalPrompt := p.buildFinalPrompt(req.Prompt, fullResponse, results)

			// Stream final response
			finalStreamReq := &LLMRequest{
				Model:       "default",
				Messages:    []Message{{Role: "user", Content: finalPrompt}},
				MaxTokens:   req.MaxTokens,
				Temperature: req.Temperature,
				Stream:      true,
			}

			finalStreamCh := make(chan LLMResponse, 100)
			err = p.baseProvider.GenerateStream(ctx, finalStreamReq, finalStreamCh)
			if err != nil {
				ch <- ToolStreamChunk{
					ID:    uuid.New(),
					Error: fmt.Sprintf("Failed to stream final response: %v", err),
					Done:  true,
				}
				return
			}

			for resp := range finalStreamCh {
				// Check for errors (in a real implementation, you'd have error handling)
				// For now, we'll assume no errors in streaming

				ch <- ToolStreamChunk{
					ID:        uuid.New(),
					Content:   resp.Content,
					ToolCalls: toolCalls,
					Reasoning: reasoning,
					Done:      true, // Assume done when we get the final response
				}
			}
		} else {
			// No tool calls, send final chunk
			ch <- ToolStreamChunk{
				ID:        uuid.New(),
				Content:   "",
				ToolCalls: toolCalls,
				Reasoning: reasoning,
				Done:      true,
			}
		}
	}()

	return ch, nil
}

// ListAvailableTools returns all registered tools
func (p *ToolCallingProvider) ListAvailableTools() []Tool {
	tools := make([]Tool, 0, len(p.tools))
	for _, tool := range p.tools {
		tools = append(tools, tool)
	}
	return tools
}

// RegisterTool registers a new tool with the provider
func (p *ToolCallingProvider) RegisterTool(tool Tool) error {
	if _, exists := p.tools[tool.Function.Name]; exists {
		return fmt.Errorf("tool %s already registered", tool.Function.Name)
	}
	p.tools[tool.Function.Name] = tool

	// Also register with reasoning engine
	if p.reasoningEngine != nil {
		// Convert Tool to ReasoningTool
		reasoningTool := ReasoningTool{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			Parameters:  tool.Function.Parameters,
			Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
				// This would need to be implemented based on the actual tool functionality
				return "tool_executed", nil
			},
		}
		p.reasoningEngine.RegisterTool(reasoningTool)
	}

	log.Printf("Tool registered: %s", tool.Function.Name)
	return nil
}

// Implement base Provider interface

func (p *ToolCallingProvider) GetType() ProviderType {
	return p.baseProvider.GetType()
}

func (p *ToolCallingProvider) GetName() string {
	return p.baseProvider.GetName()
}

func (p *ToolCallingProvider) GetModels() []ModelInfo {
	return p.baseProvider.GetModels()
}

func (p *ToolCallingProvider) GetCapabilities() []ModelCapability {
	return p.baseProvider.GetCapabilities()
}

func (p *ToolCallingProvider) Generate(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	return p.baseProvider.Generate(ctx, req)
}

func (p *ToolCallingProvider) GenerateStream(ctx context.Context, req *LLMRequest, ch chan<- LLMResponse) error {
	return p.baseProvider.GenerateStream(ctx, req, ch)
}

func (p *ToolCallingProvider) IsAvailable(ctx context.Context) bool {
	return p.baseProvider.IsAvailable(ctx)
}

func (p *ToolCallingProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	return p.baseProvider.GetHealth(ctx)
}

func (p *ToolCallingProvider) Close() error {
	return p.baseProvider.Close()
}

// Helper methods

func (p *ToolCallingProvider) buildToolEnhancedPrompt(prompt string, tools []Tool) string {
	toolDescriptions := ""
	for _, tool := range tools {
		paramsJSON, _ := json.Marshal(tool.Function.Parameters)
		toolDescriptions += fmt.Sprintf("- %s: %s (parameters: %s)\n",
			tool.Function.Name, tool.Function.Description, string(paramsJSON))
	}

	return fmt.Sprintf(`You have access to the following tools:
%s

When you need to use a tool, specify it in this format:
TOOL_CALL: {"tool_name": "tool_name", "arguments": {...}}

After using tools, provide your final answer.

User request: %s

Your response:`, toolDescriptions, prompt)
}

func (p *ToolCallingProvider) extractToolCallsAndReasoning(text string) ([]ToolCall, string) {
	var toolCalls []ToolCall
	reasoning := ""

	// Simple parsing for tool calls
	// In a real implementation, you would use more sophisticated parsing
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if strings.Contains(line, "TOOL_CALL:") {
			// Extract JSON from tool call
			jsonStart := strings.Index(line, "{")
			jsonEnd := strings.LastIndex(line, "}")
			if jsonStart != -1 && jsonEnd != -1 {
				jsonStr := line[jsonStart : jsonEnd+1]
				var toolCall ToolCall
				if err := json.Unmarshal([]byte(jsonStr), &toolCall); err == nil {
					toolCalls = append(toolCalls, toolCall)
				}
			}
		} else if !strings.Contains(line, "TOOL_CALL:") {
			// Collect reasoning (non-tool-call lines)
			reasoning += line + "\n"
		}
	}

	return toolCalls, strings.TrimSpace(reasoning)
}

func (p *ToolCallingProvider) executeToolCalls(ctx context.Context, toolCalls []ToolCall) (map[string]interface{}, error) {
	results := make(map[string]interface{})

	for _, toolCall := range toolCalls {
		_, exists := p.tools[toolCall.Function.Name]
		if !exists {
			results[toolCall.Function.Name] = fmt.Sprintf("Tool not found: %s", toolCall.Function.Name)
			continue
		}

		result, err := p.executeToolHandler(ctx, toolCall.Function.Name, toolCall.Function.Arguments)
		if err != nil {
			results[toolCall.Function.Name] = fmt.Sprintf("Tool error: %v", err)
		} else {
			results[toolCall.Function.Name] = result
		}
	}

	return results, nil
}

// executeToolHandler executes a tool handler based on the tool name
func (p *ToolCallingProvider) executeToolHandler(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	// This is a placeholder implementation
	// In a real implementation, you would have actual tool handlers
	// For now, we'll return a simple response
	return fmt.Sprintf("Executed tool %s with args %v", toolName, args), nil
}

func (p *ToolCallingProvider) buildFinalPrompt(originalPrompt, initialResponse string, toolResults map[string]interface{}) string {
	resultsStr := ""
	for toolName, result := range toolResults {
		resultsStr += fmt.Sprintf("- %s: %v\n", toolName, result)
	}

	return fmt.Sprintf(`Original request: %s

Initial response: %s

Tool execution results:
%s

Based on the tool results, provide your final answer:`,
		originalPrompt, initialResponse, resultsStr)
}
