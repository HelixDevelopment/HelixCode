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

// OpenRouterProvider implements the Provider interface for OpenRouter models
type OpenRouterProvider struct {
	config     ProviderConfigEntry
	endpoint   string
	apiKey     string
	httpClient *http.Client
	models     []ModelInfo
	lastHealth *ProviderHealth
}

// NewOpenRouterProvider creates a new OpenRouter provider
func NewOpenRouterProvider(config ProviderConfigEntry) (*OpenRouterProvider, error) {
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "https://openrouter.ai/api/v1"
	}

	apiKey := config.APIKey
	if apiKey == "" {
		return nil, fmt.Errorf("OpenRouter API key is required")
	}

	provider := &OpenRouterProvider{
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
func (orp *OpenRouterProvider) GetType() ProviderType {
	return ProviderTypeOpenRouter
}

// GetName returns the provider name
func (orp *OpenRouterProvider) GetName() string {
	return "OpenRouter"
}

// GetModels returns available models
func (orp *OpenRouterProvider) GetModels() []ModelInfo {
	return orp.models
}

// GetCapabilities returns provider capabilities
func (orp *OpenRouterProvider) GetCapabilities() []ModelCapability {
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

// Generate generates a response using OpenRouter models
func (orp *OpenRouterProvider) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	startTime := time.Now()

	// Convert to OpenAI-compatible format (OpenRouter uses OpenAI-compatible API)
	openaiRequest, err := orp.convertToOpenAIRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %v", err)
	}

	// Make request to OpenRouter API
	response, err := orp.makeOpenAIRequest(ctx, openaiRequest)
	if err != nil {
		return nil, fmt.Errorf("OpenRouter request failed: %v", err)
	}

	// Convert response
	llmResponse := orp.convertFromOpenAIResponse(response, request.ID, time.Since(startTime))

	return llmResponse, nil
}

// GenerateStream generates a streaming response
func (orp *OpenRouterProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
	defer close(ch)

	// Convert to OpenAI-compatible format
	openaiRequest, err := orp.convertToOpenAIRequest(request)
	if err != nil {
		return fmt.Errorf("failed to convert request: %v", err)
	}

	// Enable streaming
	openaiRequest.Stream = true

	// Make streaming request
	return orp.makeOpenAIStreamRequest(ctx, openaiRequest, ch, request.ID)
}

// IsAvailable checks if the provider is available
func (orp *OpenRouterProvider) IsAvailable(ctx context.Context) bool {
	health, err := orp.GetHealth(ctx)
	return err == nil && health.Status == "healthy"
}

// GetHealth returns provider health status
func (orp *OpenRouterProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	// Check if we can reach the OpenRouter API
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/models", orp.endpoint), nil)
	if err != nil {
		orp.updateHealth("unhealthy", 0, orp.lastHealth.ErrorCount+1)
		return orp.lastHealth, fmt.Errorf("failed to create health check request: %v", err)
	}

	orp.setAuthHeaders(req)

	start := time.Now()
	resp, err := orp.httpClient.Do(req)
	latency := time.Since(start)

	if err != nil {
		orp.updateHealth("unhealthy", latency, orp.lastHealth.ErrorCount+1)
		return orp.lastHealth, fmt.Errorf("health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		orp.updateHealth("unhealthy", latency, orp.lastHealth.ErrorCount+1)
		return orp.lastHealth, fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	// Parse response to get model count
	var modelsResponse struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&modelsResponse); err != nil {
		orp.updateHealth("degraded", latency, orp.lastHealth.ErrorCount)
		return orp.lastHealth, nil // Still consider it available
	}

	orp.updateHealth("healthy", latency, 0)
	orp.lastHealth.ModelCount = len(modelsResponse.Data)

	return orp.lastHealth, nil
}

// Close closes the provider
func (orp *OpenRouterProvider) Close() error {
	orp.httpClient.CloseIdleConnections()
	return nil
}

// Helper methods

func (orp *OpenRouterProvider) initializeModels() {
	// Predefined OpenRouter models with their capabilities
	// Focus on free/low-cost models
	orp.models = []ModelInfo{
		{
			Name:           "deepseek-r1-free",
			Provider:       ProviderTypeOpenRouter,
			ContextSize:    163840,
			Capabilities:   orp.GetCapabilities(),
			MaxTokens:      10000,
			SupportsTools:  true,
			SupportsVision: false,
			Description:    "DeepSeek R1 Free - Free reasoning model via OpenRouter",
		},
		{
			Name:        "meta-llama/llama-3.2-3b-instruct:free",
			Provider:    ProviderTypeOpenRouter,
			ContextSize: 131072,
			Capabilities: []ModelCapability{
				CapabilityTextGeneration,
				CapabilityCodeGeneration,
				CapabilityCodeAnalysis,
			},
			MaxTokens:      4096,
			SupportsTools:  false,
			SupportsVision: false,
			Description:    "Llama 3.2 3B Instruct Free - Free lightweight model",
		},
		{
			Name:           "microsoft/wizardlm-2-8x22b:free",
			Provider:       ProviderTypeOpenRouter,
			ContextSize:    65536,
			Capabilities:   orp.GetCapabilities(),
			MaxTokens:      4096,
			SupportsTools:  true,
			SupportsVision: false,
			Description:    "WizardLM-2 8x22B Free - Free instruction-tuned model",
		},
		{
			Name:        "mistralai/mistral-7b-instruct:free",
			Provider:    ProviderTypeOpenRouter,
			ContextSize: 32768,
			Capabilities: []ModelCapability{
				CapabilityTextGeneration,
				CapabilityCodeGeneration,
				CapabilityCodeAnalysis,
			},
			MaxTokens:      4096,
			SupportsTools:  false,
			SupportsVision: false,
			Description:    "Mistral 7B Instruct Free - Free Mistral model",
		},
		{
			Name:        "huggingface/zephyr-7b-beta:free",
			Provider:    ProviderTypeOpenRouter,
			ContextSize: 32768,
			Capabilities: []ModelCapability{
				CapabilityTextGeneration,
				CapabilityCodeGeneration,
			},
			MaxTokens:      4096,
			SupportsTools:  false,
			SupportsVision: false,
			Description:    "Zephyr 7B Beta Free - Free fine-tuned model",
		},
	}

	log.Printf("✅ OpenRouter provider initialized with %d models", len(orp.models))
}

func (orp *OpenRouterProvider) convertToOpenAIRequest(request *LLMRequest) (*OpenAIRequest, error) {
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

func (orp *OpenRouterProvider) convertFromOpenAIResponse(openaiResp *OpenAIResponse, requestID uuid.UUID, processingTime time.Duration) *LLMResponse {
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

func (orp *OpenRouterProvider) makeOpenAIRequest(ctx context.Context, request *OpenAIRequest) (*OpenAIResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/chat/completions", orp.endpoint), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	orp.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://helixcode.ai")
	req.Header.Set("X-Title", "HelixCode")

	resp, err := orp.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenRouter API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (orp *OpenRouterProvider) makeOpenAIStreamRequest(ctx context.Context, request *OpenAIRequest, ch chan<- LLMResponse, requestID uuid.UUID) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/chat/completions", orp.endpoint), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	orp.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://helixcode.ai")
	req.Header.Set("X-Title", "HelixCode")

	resp, err := orp.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OpenRouter API returned status %d: %s", resp.StatusCode, string(body))
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

func (orp *OpenRouterProvider) setAuthHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+orp.apiKey)
}

func (orp *OpenRouterProvider) updateHealth(status string, latency time.Duration, errorCount int) {
	orp.lastHealth.Status = status
	orp.lastHealth.Latency = latency
	orp.lastHealth.ErrorCount = errorCount
	orp.lastHealth.LastCheck = time.Now()
}

// Note: OpenAI API types are reused for OpenRouter compatibility
// They are declared in openai_provider.go and used here since they're in the same package
