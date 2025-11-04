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

// XAIProvider implements the Provider interface for XAI/Grok models
type XAIProvider struct {
	config     ProviderConfigEntry
	endpoint   string
	apiKey     string
	httpClient *http.Client
	models     []ModelInfo
	lastHealth *ProviderHealth
}

// NewXAIProvider creates a new XAI provider
func NewXAIProvider(config ProviderConfigEntry) (*XAIProvider, error) {
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "https://api.x.ai/v1"
	}

	apiKey := config.APIKey
	if apiKey == "" {
		return nil, fmt.Errorf("XAI API key is required")
	}

	provider := &XAIProvider{
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
func (xp *XAIProvider) GetType() ProviderType {
	return ProviderTypeXAI
}

// GetName returns the provider name
func (xp *XAIProvider) GetName() string {
	return "XAI (Grok)"
}

// GetModels returns available models
func (xp *XAIProvider) GetModels() []ModelInfo {
	return xp.models
}

// GetCapabilities returns provider capabilities
func (xp *XAIProvider) GetCapabilities() []ModelCapability {
	return []ModelCapability{
		CapabilityTextGeneration,
		CapabilityCodeGeneration,
		CapabilityCodeAnalysis,
		CapabilityPlanning,
		CapabilityDebugging,
		CapabilityRefactoring,
		CapabilityTesting,
	}
}

// Generate generates a response using XAI models
func (xp *XAIProvider) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	startTime := time.Now()

	// Convert to OpenAI-compatible format (XAI uses OpenAI-compatible API)
	openaiRequest, err := xp.convertToOpenAIRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %v", err)
	}

	// Make request to XAI API
	response, err := xp.makeOpenAIRequest(ctx, openaiRequest)
	if err != nil {
		return nil, fmt.Errorf("XAI request failed: %v", err)
	}

	// Convert response
	llmResponse := xp.convertFromOpenAIResponse(response, request.ID, time.Since(startTime))

	return llmResponse, nil
}

// GenerateStream generates a streaming response
func (xp *XAIProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
	defer close(ch)

	// Convert to OpenAI-compatible format
	openaiRequest, err := xp.convertToOpenAIRequest(request)
	if err != nil {
		return fmt.Errorf("failed to convert request: %v", err)
	}

	// Enable streaming
	openaiRequest.Stream = true

	// Make streaming request
	return xp.makeOpenAIStreamRequest(ctx, openaiRequest, ch, request.ID)
}

// IsAvailable checks if the provider is available
func (xp *XAIProvider) IsAvailable(ctx context.Context) bool {
	health, err := xp.GetHealth(ctx)
	return err == nil && health.Status == "healthy"
}

// GetHealth returns provider health status
func (xp *XAIProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	// Check if we can reach the XAI API
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/models", xp.endpoint), nil)
	if err != nil {
		xp.updateHealth("unhealthy", 0, xp.lastHealth.ErrorCount+1)
		return xp.lastHealth, fmt.Errorf("failed to create health check request: %v", err)
	}

	xp.setAuthHeaders(req)

	start := time.Now()
	resp, err := xp.httpClient.Do(req)
	latency := time.Since(start)

	if err != nil {
		xp.updateHealth("unhealthy", latency, xp.lastHealth.ErrorCount+1)
		return xp.lastHealth, fmt.Errorf("health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		xp.updateHealth("unhealthy", latency, xp.lastHealth.ErrorCount+1)
		return xp.lastHealth, fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	// Parse response to get model count
	var modelsResponse struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&modelsResponse); err != nil {
		xp.updateHealth("degraded", latency, xp.lastHealth.ErrorCount)
		return xp.lastHealth, nil // Still consider it available
	}

	xp.updateHealth("healthy", latency, 0)
	xp.lastHealth.ModelCount = len(modelsResponse.Data)

	return xp.lastHealth, nil
}

// Close closes the provider
func (xp *XAIProvider) Close() error {
	xp.httpClient.CloseIdleConnections()
	return nil
}

// Helper methods

func (xp *XAIProvider) initializeModels() {
	// Predefined XAI/Grok models with their capabilities
	xp.models = []ModelInfo{
		{
			Name:           "grok-3-fast-beta",
			Provider:       ProviderTypeXAI,
			ContextSize:    131072,
			Capabilities:   xp.GetCapabilities(),
			MaxTokens:      20000,
			SupportsTools:  true,
			SupportsVision: false,
			Description:    "Grok 3 Fast Beta - Fast and efficient Grok model for coding and general tasks",
		},
		{
			Name:           "grok-3-mini-fast-beta",
			Provider:       ProviderTypeXAI,
			ContextSize:    131072,
			Capabilities:   xp.GetCapabilities(),
			MaxTokens:      20000,
			SupportsTools:  true,
			SupportsVision: false,
			Description:    "Grok 3 Mini Fast Beta - Lightweight and fast Grok model",
		},
		{
			Name:           "grok-3-beta",
			Provider:       ProviderTypeXAI,
			ContextSize:    131072,
			Capabilities:   xp.GetCapabilities(),
			MaxTokens:      20000,
			SupportsTools:  true,
			SupportsVision: false,
			Description:    "Grok 3 Beta - Full-featured Grok model with advanced capabilities",
		},
		{
			Name:        "grok-3-mini-beta",
			Provider:    ProviderTypeXAI,
			ContextSize: 131072,
			Capabilities: []ModelCapability{
				CapabilityTextGeneration,
				CapabilityCodeGeneration,
				CapabilityCodeAnalysis,
			},
			MaxTokens:      20000,
			SupportsTools:  false,
			SupportsVision: false,
			Description:    "Grok 3 Mini Beta - Efficient Grok model for basic tasks",
		},
	}

	log.Printf("✅ XAI provider initialized with %d models", len(xp.models))
}

func (xp *XAIProvider) convertToOpenAIRequest(request *LLMRequest) (*OpenAIRequest, error) {
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

func (xp *XAIProvider) convertFromOpenAIResponse(openaiResp *OpenAIResponse, requestID uuid.UUID, processingTime time.Duration) *LLMResponse {
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

func (xp *XAIProvider) makeOpenAIRequest(ctx context.Context, request *OpenAIRequest) (*OpenAIResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/chat/completions", xp.endpoint), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	xp.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := xp.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("XAI API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (xp *XAIProvider) makeOpenAIStreamRequest(ctx context.Context, request *OpenAIRequest, ch chan<- LLMResponse, requestID uuid.UUID) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/chat/completions", xp.endpoint), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	xp.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := xp.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("XAI API returned status %d: %s", resp.StatusCode, string(body))
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

func (xp *XAIProvider) setAuthHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+xp.apiKey)
}

func (xp *XAIProvider) updateHealth(status string, latency time.Duration, errorCount int) {
	xp.lastHealth.Status = status
	xp.lastHealth.Latency = latency
	xp.lastHealth.ErrorCount = errorCount
	xp.lastHealth.LastCheck = time.Now()
}

// Note: OpenAI API types are reused for XAI compatibility
// They are declared in openai_provider.go and used here since they're in the same package
