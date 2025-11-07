package mock

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// LLMProvider is a mock LLM provider for E2E testing
type LLMProvider struct {
	mu              sync.RWMutex
	name            string
	providerType    string
	available       bool
	responseDelay   time.Duration
	errorRate       float64
	requestCount    int
	responseMap     map[string]string // Map prompts to responses
	defaultResponse string
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMRequest represents a request to the LLM
type LLMRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
}

// LLMResponse represents a response from the LLM
type LLMResponse struct {
	ID         uuid.UUID `json:"id"`
	Content    string    `json:"content"`
	Model      string    `json:"model"`
	FinishReason string  `json:"finish_reason"`
	Usage      Usage     `json:"usage"`
	CreatedAt  time.Time `json:"created_at"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewMockLLMProvider creates a new mock LLM provider
func NewMockLLMProvider(name string) *LLMProvider {
	return &LLMProvider{
		name:            name,
		providerType:    "mock",
		available:       true,
		responseDelay:   100 * time.Millisecond,
		errorRate:       0.0,
		requestCount:    0,
		responseMap:     make(map[string]string),
		defaultResponse: "This is a mock response from the LLM provider.",
	}
}

// SetResponseDelay sets the simulated response delay
func (p *LLMProvider) SetResponseDelay(delay time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.responseDelay = delay
}

// SetErrorRate sets the error rate (0.0 to 1.0)
func (p *LLMProvider) SetErrorRate(rate float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errorRate = rate
}

// SetAvailable sets whether the provider is available
func (p *LLMProvider) SetAvailable(available bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.available = available
}

// AddResponse adds a response template for a specific prompt pattern
func (p *LLMProvider) AddResponse(promptPattern, response string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.responseMap[promptPattern] = response
}

// SetDefaultResponse sets the default response when no pattern matches
func (p *LLMProvider) SetDefaultResponse(response string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.defaultResponse = response
}

// Generate generates a response for the given request
func (p *LLMProvider) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	p.mu.Lock()
	p.requestCount++
	available := p.available
	delay := p.responseDelay
	errorRate := p.errorRate
	p.mu.Unlock()

	// Check availability
	if !available {
		return nil, fmt.Errorf("mock LLM provider is not available")
	}

	// Simulate error rate
	if errorRate > 0 && float64(p.requestCount)/(float64(p.requestCount)+1.0) < errorRate {
		return nil, fmt.Errorf("simulated error from mock provider")
	}

	// Simulate processing delay
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Find matching response
	content := p.findResponse(request)

	// Calculate token usage (simple estimation)
	promptTokens := p.estimateTokens(request)
	completionTokens := len(strings.Fields(content))

	response := &LLMResponse{
		ID:           uuid.New(),
		Content:      content,
		Model:        request.Model,
		FinishReason: "stop",
		Usage: Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
		CreatedAt: time.Now(),
	}

	return response, nil
}

// findResponse finds the appropriate response for the request
func (p *LLMProvider) findResponse(request *LLMRequest) string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Get the last user message
	var userMessage string
	for i := len(request.Messages) - 1; i >= 0; i-- {
		if request.Messages[i].Role == "user" {
			userMessage = request.Messages[i].Content
			break
		}
	}

	// Check for pattern matches
	for pattern, response := range p.responseMap {
		if strings.Contains(strings.ToLower(userMessage), strings.ToLower(pattern)) {
			return response
		}
	}

	// Return default response
	return p.defaultResponse
}

// estimateTokens estimates the number of tokens in the request
func (p *LLMProvider) estimateTokens(request *LLMRequest) int {
	total := 0
	for _, msg := range request.Messages {
		total += len(strings.Fields(msg.Content))
	}
	return total
}

// IsAvailable checks if the provider is available
func (p *LLMProvider) IsAvailable() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.available
}

// GetRequestCount returns the number of requests processed
func (p *LLMProvider) GetRequestCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.requestCount
}

// Reset resets the provider state
func (p *LLMProvider) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.requestCount = 0
	p.responseMap = make(map[string]string)
	p.defaultResponse = "This is a mock response from the LLM provider."
	p.available = true
	p.errorRate = 0.0
}

// GetName returns the provider name
func (p *LLMProvider) GetName() string {
	return p.name
}

// GetType returns the provider type
func (p *LLMProvider) GetType() string {
	return p.providerType
}

// Close closes the provider (no-op for mock)
func (p *LLMProvider) Close() error {
	return nil
}
