package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// KoboldAIProvider implements the Provider interface for KoboldAI
// KoboldAI has a custom API format that differs from OpenAI
type KoboldAIProvider struct {
	config     KoboldAIConfig
	httpClient *http.Client
	models     []ModelInfo
	lastHealth *ProviderHealth
	isRunning  bool
}

// KoboldAIConfig holds configuration for KoboldAI
type KoboldAIConfig struct {
	BaseURL          string            `json:"base_url"`
	APIKey           string            `json:"api_key"`
	DefaultModel     string            `json:"default_model"`
	Timeout          time.Duration     `json:"timeout"`
	MaxRetries       int               `json:"max_retries"`
	Headers          map[string]string `json:"headers"`
	StreamingSupport bool              `json:"streaming_support"`
}

// KoboldAIRequest represents a request to the KoboldAI API
type KoboldAIRequest struct {
	Prompt         string   `json:"prompt"`
	MaxLength      int      `json:"max_length,omitempty"`
	Temperature    float64  `json:"temperature,omitempty"`
	TopP           float64  `json:"top_p,omitempty"`
	TopK           int      `json:"top_k,omitempty"`
	RepPen         float64  `json:"rep_pen,omitempty"`
	RepPenRange    int      `json:"rep_pen_range,omitempty"`
	StopSequences  []string `json:"stop_sequence,omitempty"`
	Stream         bool     `json:"stream,omitempty"`
	UseStory       bool     `json:"use_story,omitempty"`
	UseMemory      bool     `json:"use_memory,omitempty"`
	UseAuthorsNote bool     `json:"use_authors_note,omitempty"`
	UseWorldInfo   bool     `json:"use_world_info,omitempty"`
}

// KoboldAIResponse represents a response from the KoboldAI API
type KoboldAIResponse struct {
	Results []KoboldAIResult `json:"results"`
	Detail  string           `json:"detail,omitempty"`
}

// KoboldAIResult represents a result in the response
type KoboldAIResult struct {
	Text      string    `json:"text"`
	Generated bool      `json:"generated"`
	Logits    []float64 `json:"logits,omitempty"`
}

// KoboldAIStreamResponse represents a streaming response from KoboldAI
type KoboldAIStreamResponse struct {
	Token string `json:"token"`
	Data  string `json:"data,omitempty"`
}

// KoboldAIModel represents a KoboldAI model
type KoboldAIModel struct {
	Name     string `json:"name"`
	Filename string `json:"filename"`
	Size     string `json:"size"`
	Modified string `json:"modified"`
}

// NewKoboldAIProvider creates a new KoboldAI provider
func NewKoboldAIProvider(config KoboldAIConfig) (*KoboldAIProvider, error) {
	provider := &KoboldAIProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		isRunning: true,
		lastHealth: &ProviderHealth{
			Status:    "unknown",
			LastCheck: time.Now(),
		},
	}

	// Discover available models
	if err := provider.discoverModels(); err != nil {
		log.Printf("Warning: Failed to discover KoboldAI models: %v", err)
	}

	log.Printf("✅ KoboldAI provider initialized with %d models", len(provider.models))
	return provider, nil
}

// GetType returns the provider type
func (p *KoboldAIProvider) GetType() ProviderType {
	return ProviderTypeKoboldAI
}

// GetName returns the provider name
func (p *KoboldAIProvider) GetName() string {
	return "koboldai"
}

// GetModels returns available models
func (p *KoboldAIProvider) GetModels() []ModelInfo {
	return p.models
}

// GetCapabilities returns model capabilities
func (p *KoboldAIProvider) GetCapabilities() []ModelCapability {
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

// Generate generates a response using KoboldAI
func (p *KoboldAIProvider) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	if !p.isRunning {
		return nil, ErrProviderUnavailable
	}

	// Convert messages to prompt
	prompt := p.convertMessagesToPrompt(request.Messages)

	// Prepare API request
	apiRequest := &KoboldAIRequest{
		Prompt:         prompt,
		MaxLength:      request.MaxTokens,
		Temperature:    request.Temperature,
		TopP:           request.TopP,
		Stream:         false,
		UseStory:       false,
		UseMemory:      false,
		UseAuthorsNote: false,
		UseWorldInfo:   false,
	}

	// Make API call
	startTime := time.Now()
	response, err := p.makeAPIRequest(ctx, apiRequest)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	processingTime := time.Since(startTime)

	// Convert response
	llmResponse := p.convertFromKoboldAIResponse(response, request.ID, processingTime)

	return llmResponse, nil
}

// GenerateStream generates a streaming response
func (p *KoboldAIProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
	if !p.isRunning {
		return ErrProviderUnavailable
	}

	if !p.config.StreamingSupport {
		// Fallback to non-streaming
		response, err := p.Generate(ctx, request)
		if err != nil {
			return err
		}
		select {
		case ch <- *response:
		case <-ctx.Done():
			return ctx.Err()
		}
		return nil
	}

	// Convert messages to prompt
	prompt := p.convertMessagesToPrompt(request.Messages)

	// Prepare streaming API request
	apiRequest := &KoboldAIRequest{
		Prompt:         prompt,
		MaxLength:      request.MaxTokens,
		Temperature:    request.Temperature,
		TopP:           request.TopP,
		Stream:         true,
		UseStory:       false,
		UseMemory:      false,
		UseAuthorsNote: false,
		UseWorldInfo:   false,
	}

	// Make streaming request
	return p.makeStreamingRequest(ctx, apiRequest, ch, request.ID)
}

// IsAvailable checks if the provider is available
func (p *KoboldAIProvider) IsAvailable(ctx context.Context) bool {
	if !p.isRunning {
		return false
	}

	health, err := p.GetHealth(ctx)
	return err == nil && health.Status == "healthy"
}

// GetHealth returns provider health status
func (p *KoboldAIProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	if !p.isRunning {
		return &ProviderHealth{
			Status:     "unhealthy",
			LastCheck:  time.Now(),
			ErrorCount: 1,
		}, nil
	}

	// Test basic API endpoint
	start := time.Now()
	url := p.getAPIURL("/api/v1/model")
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		p.updateHealth("unhealthy", 0, p.lastHealth.ErrorCount+1)
		return p.lastHealth, fmt.Errorf("failed to create health check request: %v", err)
	}

	// Set headers
	for key, value := range p.config.Headers {
		req.Header.Set(key, value)
	}
	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	latency := time.Since(start)

	if err != nil {
		p.updateHealth("unhealthy", latency, p.lastHealth.ErrorCount+1)
		return p.lastHealth, fmt.Errorf("health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		p.updateHealth("unhealthy", latency, p.lastHealth.ErrorCount+1)
		return p.lastHealth, fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	p.updateHealth("healthy", latency, 0)
	p.lastHealth.ModelCount = len(p.models)

	return p.lastHealth, nil
}

// Close stops the provider
func (p *KoboldAIProvider) Close() error {
	p.isRunning = false
	if p.httpClient != nil {
		p.httpClient.CloseIdleConnections()
	}
	log.Println("✅ KoboldAI provider closed")
	return nil
}

// Private helper methods

func (p *KoboldAIProvider) discoverModels() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := p.getAPIURL("/api/v1/model")
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create models request: %w", err)
	}

	// Set headers
	for key, value := range p.config.Headers {
		req.Header.Set(key, value)
	}
	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse model name from response
	var modelResponse struct {
		Result string `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&modelResponse); err != nil {
		return fmt.Errorf("failed to decode model response: %w", err)
	}

	// Create a single model info with the loaded model
	if modelResponse.Result != "" {
		modelInfo := ModelInfo{
			Name:           modelResponse.Result,
			Provider:       ProviderTypeKoboldAI,
			ContextSize:    2048, // Default for KoboldAI models
			MaxTokens:      1024,
			Capabilities:   p.GetCapabilities(),
			SupportsTools:  false,
			SupportsVision: false,
			Description:    fmt.Sprintf("KoboldAI model: %s", modelResponse.Result),
		}
		p.models = append(p.models, modelInfo)
	}

	return nil
}

func (p *KoboldAIProvider) convertMessagesToPrompt(messages []Message) string {
	var prompt strings.Builder

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			prompt.WriteString(fmt.Sprintf("System: %s\n\n", msg.Content))
		case "user":
			prompt.WriteString(fmt.Sprintf("User: %s\n\n", msg.Content))
		case "assistant":
			prompt.WriteString(fmt.Sprintf("Assistant: %s\n\n", msg.Content))
		}
	}

	prompt.WriteString("Assistant: ")
	return prompt.String()
}

func (p *KoboldAIProvider) convertFromKoboldAIResponse(response *KoboldAIResponse, requestID uuid.UUID, processingTime time.Duration) *LLMResponse {
	llmResponse := &LLMResponse{
		ID:             uuid.New(),
		RequestID:      requestID,
		Content:        "",
		Usage:          Usage{},
		ProcessingTime: processingTime,
		CreatedAt:      time.Now(),
	}

	if len(response.Results) > 0 {
		result := response.Results[0]
		llmResponse.Content = result.Text

		// Estimate token usage (KoboldAI doesn't provide exact counts)
		promptTokens := 100 // Rough estimate
		completionTokens := len(strings.Split(result.Text, " "))
		llmResponse.Usage = Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		}
	}

	return llmResponse
}

func (p *KoboldAIProvider) makeAPIRequest(ctx context.Context, request *KoboldAIRequest) (*KoboldAIResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := p.getAPIURL("/api/v1/generate")
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range p.config.Headers {
		req.Header.Set(key, value)
	}
	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response KoboldAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

func (p *KoboldAIProvider) makeStreamingRequest(ctx context.Context, request *KoboldAIRequest, ch chan<- LLMResponse, requestID uuid.UUID) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := p.getAPIURL("/api/v1/generate")
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	for key, value := range p.config.Headers {
		req.Header.Set(key, value)
	}
	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Process SSE stream
	for {
		var line string
		line, err = readSSELine(resp.Body)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read SSE line: %w", err)
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var streamResponse KoboldAIStreamResponse
			if err := json.Unmarshal([]byte(data), &streamResponse); err != nil {
				continue // Skip malformed JSON
			}

			if streamResponse.Token != "" || streamResponse.Data != "" {
				content := streamResponse.Token
				if content == "" {
					content = streamResponse.Data
				}

				response := LLMResponse{
					ID:        uuid.New(),
					RequestID: requestID,
					Content:   content,
					CreatedAt: time.Now(),
				}

				select {
				case ch <- response:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}

	return nil
}

func (p *KoboldAIProvider) getModelName(requestedModel string) string {
	if requestedModel != "" {
		return requestedModel
	}

	if p.config.DefaultModel != "" {
		return p.config.DefaultModel
	}

	// Return first available model
	if len(p.models) > 0 {
		return p.models[0].Name
	}

	return "default" // Fallback
}

func (p *KoboldAIProvider) getAPIURL(endpoint string) string {
	baseURL := p.config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:5001"
	}

	return strings.TrimSuffix(baseURL, "/") + endpoint
}

func (p *KoboldAIProvider) updateHealth(status string, latency time.Duration, errorCount int) {
	p.lastHealth.Status = status
	p.lastHealth.Latency = latency
	p.lastHealth.ErrorCount = errorCount
	p.lastHealth.LastCheck = time.Now()
}
