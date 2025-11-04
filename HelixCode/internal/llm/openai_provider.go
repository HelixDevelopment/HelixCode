package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// OpenAIProvider implements the Provider interface for OpenAI models
type OpenAIProvider struct {
	config     ProviderConfigEntry
	endpoint   string
	apiKey     string
	httpClient *http.Client
	models     []ModelInfo
	lastHealth *ProviderHealth
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(config ProviderConfigEntry) (*OpenAIProvider, error) {
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}

	apiKey := config.APIKey
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	provider := &OpenAIProvider{
		config:   config,
		endpoint: endpoint,
		apiKey:   apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		lastHealth: &ProviderHealth{
			Status:    "unknown",
			LastCheck: time.Now(),
		},
	}

	// Initialize models
	provider.initializeModels()

	return provider, nil
}

// GetType returns the provider type
func (op *OpenAIProvider) GetType() ProviderType {
	return ProviderTypeOpenAI
}

// GetName returns the provider name
func (op *OpenAIProvider) GetName() string {
	return "OpenAI"
}

// GetModels returns available models
func (op *OpenAIProvider) GetModels() []ModelInfo {
	return op.models
}

// GetCapabilities returns provider capabilities
func (op *OpenAIProvider) GetCapabilities() []ModelCapability {
	return []ModelCapability{
		CapabilityTextGeneration,
		CapabilityCodeGeneration,
		CapabilityCodeAnalysis,
		CapabilityPlanning,
		CapabilityDebugging,
		CapabilityRefactoring,
		CapabilityTesting,
		CapabilityVision,
	}
}

// Generate generates a response using OpenAI models
func (op *OpenAIProvider) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	startTime := time.Now()

	// Convert to OpenAI-compatible format
	openaiRequest, err := op.convertToOpenAIRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %v", err)
	}

	// Make request to OpenAI API
	response, err := op.makeOpenAIRequest(ctx, openaiRequest)
	if err != nil {
		return nil, fmt.Errorf("OpenAI request failed: %v", err)
	}

	// Convert response
	llmResponse := op.convertFromOpenAIResponse(response, request.ID, time.Since(startTime))

	return llmResponse, nil
}

// GenerateStream generates a streaming response
func (op *OpenAIProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
	defer close(ch)

	// Convert to OpenAI-compatible format
	openaiRequest, err := op.convertToOpenAIRequest(request)
	if err != nil {
		return fmt.Errorf("failed to convert request: %v", err)
	}

	// Enable streaming
	openaiRequest.Stream = true

	// Make streaming request
	return op.makeOpenAIStreamRequest(ctx, openaiRequest, ch, request.ID)
}

// IsAvailable checks if the provider is available
func (op *OpenAIProvider) IsAvailable(ctx context.Context) bool {
	health, err := op.GetHealth(ctx)
	return err == nil && health.Status == "healthy"
}

// GetHealth returns provider health status
func (op *OpenAIProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	// Check if we can reach the OpenAI API
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/models", op.endpoint), nil)
	if err != nil {
		op.updateHealth("unhealthy", 0, 1)
		return op.lastHealth, fmt.Errorf("failed to create health check request: %v", err)
	}

	op.setAuthHeaders(req)

	start := time.Now()
	resp, err := op.httpClient.Do(req)
	latency := time.Since(start)

	if err != nil {
		op.updateHealth("unhealthy", latency, op.lastHealth.ErrorCount+1)
		return op.lastHealth, fmt.Errorf("health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		op.updateHealth("unhealthy", latency, op.lastHealth.ErrorCount+1)
		return op.lastHealth, fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	// Parse response to get model count
	var modelsResponse struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&modelsResponse); err != nil {
		op.updateHealth("degraded", latency, op.lastHealth.ErrorCount)
		return op.lastHealth, nil // Still consider it available
	}

	op.updateHealth("healthy", latency, 0)
	op.lastHealth.ModelCount = len(modelsResponse.Data)

	return op.lastHealth, nil
}

// Close closes the provider
func (op *OpenAIProvider) Close() error {
	op.httpClient.CloseIdleConnections()
	return nil
}

// Helper methods

func (op *OpenAIProvider) initializeModels() {
	// Predefined OpenAI models with their capabilities
	op.models = []ModelInfo{
		{
			Name:           "gpt-4o",
			Provider:       ProviderTypeOpenAI,
			ContextSize:    128000,
			Capabilities:   op.GetCapabilities(),
			MaxTokens:      4096,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "OpenAI's most advanced multimodal model",
		},
		{
			Name:           "gpt-4-turbo",
			Provider:       ProviderTypeOpenAI,
			ContextSize:    128000,
			Capabilities:   op.GetCapabilities(),
			MaxTokens:      4096,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "OpenAI's advanced multimodal model",
		},
		{
			Name:           "gpt-4",
			Provider:       ProviderTypeOpenAI,
			ContextSize:    8192,
			Capabilities:   op.GetCapabilities(),
			MaxTokens:      4096,
			SupportsTools:  true,
			SupportsVision: false,
			Description:    "OpenAI's powerful text model",
		},
		{
			Name:           "gpt-3.5-turbo",
			Provider:       ProviderTypeOpenAI,
			ContextSize:    16385,
			Capabilities:   op.GetCapabilities(),
			MaxTokens:      4096,
			SupportsTools:  true,
			SupportsVision: false,
			Description:    "OpenAI's fast and efficient model",
		},
	}

	log.Printf("✅ OpenAI provider initialized with %d models", len(op.models))
}

func (op *OpenAIProvider) convertToOpenAIRequest(request *LLMRequest) (*OpenAIRequest, error) {
	// Convert messages to OpenAI format
	var messages []OpenAIMessage
	for _, msg := range request.Messages {
		openaiMsg := OpenAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
		if msg.Name != "" {
			openaiMsg.Name = msg.Name
		}
		messages = append(messages, openaiMsg)
	}

	return &OpenAIRequest{
		Model:       request.Model,
		Messages:    messages,
		MaxTokens:   request.MaxTokens,
		Temperature: request.Temperature,
		TopP:        request.TopP,
		Stream:      request.Stream,
	}, nil
}

func (op *OpenAIProvider) convertFromOpenAIResponse(openaiResp *OpenAIResponse, requestID uuid.UUID, processingTime time.Duration) *LLMResponse {
	var content string

	if len(openaiResp.Choices) > 0 {
		choice := openaiResp.Choices[0]
		content = choice.Message.Content
	}

	return &LLMResponse{
		ID:        uuid.New(),
		RequestID: requestID,
		Content:   content,
		Usage: Usage{
			PromptTokens:     openaiResp.Usage.PromptTokens,
			CompletionTokens: openaiResp.Usage.CompletionTokens,
			TotalTokens:      openaiResp.Usage.TotalTokens,
		},
		FinishReason:   openaiResp.Choices[0].FinishReason,
		ProcessingTime: processingTime,
		CreatedAt:      time.Now(),
	}
}

func (op *OpenAIProvider) makeOpenAIRequest(ctx context.Context, request *OpenAIRequest) (*OpenAIResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/chat/completions", op.endpoint), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	op.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := op.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (op *OpenAIProvider) makeOpenAIStreamRequest(ctx context.Context, request *OpenAIRequest, ch chan<- LLMResponse, requestID uuid.UUID) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/chat/completions", op.endpoint), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	op.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := op.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Stream responses
	decoder := json.NewDecoder(resp.Body)
	for decoder.More() {
		var streamResp OpenAIStreamResponse
		if err := decoder.Decode(&streamResp); err != nil {
			return err
		}

		if len(streamResp.Choices) > 0 {
			choice := streamResp.Choices[0]
			if choice.Delta.Content != "" {
				response := LLMResponse{
					ID:        uuid.New(),
					RequestID: requestID,
					Content:   choice.Delta.Content,
					CreatedAt: time.Now(),
				}

				select {
				case ch <- response:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}

		if streamResp.Choices[0].FinishReason != "" {
			break
		}
	}

	return nil
}

func (op *OpenAIProvider) setAuthHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+op.apiKey)
}

func (op *OpenAIProvider) updateHealth(status string, latency time.Duration, errorCount int) {
	op.lastHealth.Status = status
	op.lastHealth.Latency = latency
	op.lastHealth.ErrorCount = errorCount
	op.lastHealth.LastCheck = time.Now()
}

// OpenAI API types

type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

type OpenAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type OpenAIStreamResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}
