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

// QwenProvider implements the Provider interface for Qwen models via DashScope API
type QwenProvider struct {
	config      ProviderConfigEntry
	endpoint    string
	apiKey      string
	oauthClient *QwenOAuth2Client
	httpClient  *http.Client
	models      []ModelInfo
	lastHealth  *ProviderHealth
}

// NewQwenProvider creates a new Qwen provider
func NewQwenProvider(config ProviderConfigEntry) (*QwenProvider, error) {
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}

	provider := &QwenProvider{
		config:   config,
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		lastHealth: &ProviderHealth{
			Status:    "unknown",
			LastCheck: time.Now(),
		},
	}

	// Initialize OAuth2 client
	provider.oauthClient = NewQwenOAuth2Client()

	// Try to get API key from OAuth2 first, fall back to config
	apiKey, err := provider.getAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get Qwen API key: %v", err)
	}
	provider.apiKey = apiKey

	// Initialize models
	provider.initializeModels()

	return provider, nil
}

// getAPIKey retrieves an API key, preferring OAuth2 over config
func (qp *QwenProvider) getAPIKey() (string, error) {
	// First try OAuth2
	if qp.oauthClient != nil {
		token, err := qp.oauthClient.GetValidToken()
		if err == nil && token != "" {
			return token, nil
		}
		// OAuth2 failed, log but continue to fallback
		log.Printf("OAuth2 authentication failed, falling back to API key: %v", err)
	}

	// Fallback to config API key
	if qp.config.APIKey != "" {
		return qp.config.APIKey, nil
	}

	return "", fmt.Errorf("no API key available - configure QWEN_API_KEY or complete OAuth2 authentication")
}

// AuthenticateWithOAuth2 performs OAuth2 authentication for Qwen
func (qp *QwenProvider) AuthenticateWithOAuth2(ctx context.Context, openBrowser func(url string) error) error {
	if qp.oauthClient == nil {
		qp.oauthClient = NewQwenOAuth2Client()
	}

	creds, err := qp.oauthClient.AuthenticateWithDeviceFlow(ctx, openBrowser)
	if err != nil {
		return fmt.Errorf("OAuth2 authentication failed: %v", err)
	}

	// Update API key with the new token
	qp.apiKey = creds.AccessToken

	log.Printf("✅ Qwen OAuth2 authentication successful")
	return nil
}

// GetType returns the provider type
func (qp *QwenProvider) GetType() ProviderType {
	return ProviderTypeQwen
}

// GetName returns the provider name
func (qp *QwenProvider) GetName() string {
	return "Qwen"
}

// GetModels returns available models
func (qp *QwenProvider) GetModels() []ModelInfo {
	return qp.models
}

// GetCapabilities returns provider capabilities
func (qp *QwenProvider) GetCapabilities() []ModelCapability {
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

// Generate generates a response using Qwen models
func (qp *QwenProvider) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	startTime := time.Now()

	// Convert to OpenAI-compatible format (DashScope uses OpenAI-compatible API)
	openaiRequest, err := qp.convertToOpenAIRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %v", err)
	}

	// Make request to DashScope API
	response, err := qp.makeOpenAIRequest(ctx, openaiRequest)
	if err != nil {
		return nil, fmt.Errorf("Qwen request failed: %v", err)
	}

	// Convert response
	llmResponse := qp.convertFromOpenAIResponse(response, request.ID, time.Since(startTime))

	return llmResponse, nil
}

// GenerateStream generates a streaming response
func (qp *QwenProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
	defer close(ch)

	// Convert to OpenAI-compatible format
	openaiRequest, err := qp.convertToOpenAIRequest(request)
	if err != nil {
		return fmt.Errorf("failed to convert request: %v", err)
	}

	// Enable streaming
	openaiRequest.Stream = true

	// Make streaming request
	return qp.makeOpenAIStreamRequest(ctx, openaiRequest, ch, request.ID)
}

// IsAvailable checks if the provider is available
func (qp *QwenProvider) IsAvailable(ctx context.Context) bool {
	health, err := qp.GetHealth(ctx)
	return err == nil && health.Status == "healthy"
}

// GetHealth returns provider health status
func (qp *QwenProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	// Check if we can reach the DashScope API
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/models", qp.endpoint), nil)
	if err != nil {
		qp.updateHealth("unhealthy", 0, qp.lastHealth.ErrorCount+1)
		return qp.lastHealth, fmt.Errorf("failed to create health check request: %v", err)
	}

	qp.setAuthHeaders(req)

	start := time.Now()
	resp, err := qp.httpClient.Do(req)
	latency := time.Since(start)

	if err != nil {
		qp.updateHealth("unhealthy", latency, qp.lastHealth.ErrorCount+1)
		return qp.lastHealth, fmt.Errorf("health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		qp.updateHealth("unhealthy", latency, qp.lastHealth.ErrorCount+1)
		return qp.lastHealth, fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	// Parse response to get model count
	var modelsResponse struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&modelsResponse); err != nil {
		qp.updateHealth("degraded", latency, qp.lastHealth.ErrorCount)
		return qp.lastHealth, nil // Still consider it available
	}

	qp.updateHealth("healthy", latency, 0)
	qp.lastHealth.ModelCount = len(modelsResponse.Data)

	return qp.lastHealth, nil
}

// Close closes the provider
func (qp *QwenProvider) Close() error {
	qp.httpClient.CloseIdleConnections()
	return nil
}

// Helper methods

func (qp *QwenProvider) initializeModels() {
	// Predefined Qwen models with their capabilities
	qp.models = []ModelInfo{
		{
			Name:           "qwen3-coder-plus",
			Provider:       ProviderTypeQwen,
			ContextSize:    128000,
			Capabilities:   qp.GetCapabilities(),
			MaxTokens:      8192,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Qwen3 Coder Plus - Advanced coding model with vision support",
		},
		{
			Name:           "qwen2.5-coder-32b-instruct",
			Provider:       ProviderTypeQwen,
			ContextSize:    128000,
			Capabilities:   qp.GetCapabilities(),
			MaxTokens:      8192,
			SupportsTools:  true,
			SupportsVision: false,
			Description:    "Qwen2.5 Coder 32B - Powerful coding model",
		},
		{
			Name:           "qwen2.5-coder-7b-instruct",
			Provider:       ProviderTypeQwen,
			ContextSize:    32000,
			Capabilities:   qp.GetCapabilities(),
			MaxTokens:      4096,
			SupportsTools:  true,
			SupportsVision: false,
			Description:    "Qwen2.5 Coder 7B - Efficient coding model",
		},
		{
			Name:           "qwen-vl-plus",
			Provider:       ProviderTypeQwen,
			ContextSize:    32000,
			Capabilities:   qp.GetCapabilities(),
			MaxTokens:      4096,
			SupportsTools:  false,
			SupportsVision: true,
			Description:    "Qwen VL Plus - Vision-language model",
		},
		{
			Name:        "qwen-turbo",
			Provider:    ProviderTypeQwen,
			ContextSize: 1000000,
			Capabilities: []ModelCapability{
				CapabilityTextGeneration,
				CapabilityCodeGeneration,
				CapabilityCodeAnalysis,
			},
			MaxTokens:      4096,
			SupportsTools:  false,
			SupportsVision: false,
			Description:    "Qwen Turbo - Fast and efficient model",
		},
	}

	log.Printf("✅ Qwen provider initialized with %d models", len(qp.models))
}

func (qp *QwenProvider) convertToOpenAIRequest(request *LLMRequest) (*OpenAIRequest, error) {
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

func (qp *QwenProvider) convertFromOpenAIResponse(openaiResp *OpenAIResponse, requestID uuid.UUID, processingTime time.Duration) *LLMResponse {
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

func (qp *QwenProvider) makeOpenAIRequest(ctx context.Context, request *OpenAIRequest) (*OpenAIResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/chat/completions", qp.endpoint), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	qp.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := qp.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Qwen API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (qp *QwenProvider) makeOpenAIStreamRequest(ctx context.Context, request *OpenAIRequest, ch chan<- LLMResponse, requestID uuid.UUID) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/chat/completions", qp.endpoint), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	qp.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := qp.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Qwen API returned status %d: %s", resp.StatusCode, string(body))
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

func (qp *QwenProvider) setAuthHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+qp.apiKey)
	req.Header.Set("X-DashScope-CacheControl", "enable")
}

func (qp *QwenProvider) updateHealth(status string, latency time.Duration, errorCount int) {
	qp.lastHealth.Status = status
	qp.lastHealth.Latency = latency
	qp.lastHealth.ErrorCount = errorCount
	qp.lastHealth.LastCheck = time.Now()
}

// Note: OpenAI API types are reused for DashScope compatibility
// They are declared in openai_provider.go and used here since they're in the same package
