package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// OllamaProvider implements the LLM provider interface for Ollama
type OllamaProvider struct {
	config    OllamaConfig
	apiClient *http.Client
	models    []OllamaModel
	isRunning bool
}

// OllamaConfig holds configuration for Ollama
type OllamaConfig struct {
	BaseURL       string        `json:"base_url"`
	DefaultModel  string        `json:"default_model"`
	Timeout       time.Duration `json:"timeout"`
	KeepAlive     time.Duration `json:"keep_alive"`
	StreamEnabled bool          `json:"stream_enabled"`
}

// OllamaModel represents an Ollama model
type OllamaModel struct {
	Name       string             `json:"name"`
	ModifiedAt time.Time          `json:"modified_at"`
	Size       int64              `json:"size"`
	Digest     string             `json:"digest"`
	Details    OllamaModelDetails `json:"details"`
}

// OllamaModelDetails contains model-specific details
type OllamaModelDetails struct {
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

// OllamaAPIRequest represents a request to the Ollama API
type OllamaAPIRequest struct {
	Model    string                 `json:"model"`
	Prompt   string                 `json:"prompt"`
	Messages []Message              `json:"messages"`
	Stream   bool                   `json:"stream"`
	Options  map[string]interface{} `json:"options"`
}

// OllamaAPIResponse represents a response from the Ollama API
type OllamaAPIResponse struct {
	Model              string `json:"model"`
	CreatedAt          string `json:"created_at"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	Context            []int  `json:"context"`
	TotalDuration      int64  `json:"total_duration"`
	LoadDuration       int64  `json:"load_duration"`
	PromptEvalCount    int    `json:"prompt_eval_count"`
	PromptEvalDuration int64  `json:"prompt_eval_duration"`
	EvalCount          int    `json:"eval_count"`
	EvalDuration       int64  `json:"eval_duration"`
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(config OllamaConfig) (*OllamaProvider, error) {
	provider := &OllamaProvider{
		config: config,
		apiClient: &http.Client{
			Timeout: config.Timeout,
		},
		isRunning: true,
	}

	// Discover available models
	if err := provider.discoverModels(); err != nil {
		log.Printf("Warning: Failed to discover Ollama models: %v", err)
	}

	log.Printf("✅ Ollama provider initialized with %d models", len(provider.models))
	return provider, nil
}

// GetType returns the provider type
func (p *OllamaProvider) GetType() ProviderType {
	return ProviderTypeLocal
}

// GetName returns the provider name
func (p *OllamaProvider) GetName() string {
	return "ollama"
}

// GetModels returns available models
func (p *OllamaProvider) GetModels() []ModelInfo {
	var modelInfos []ModelInfo

	for _, model := range p.models {
		modelInfos = append(modelInfos, ModelInfo{
			Name:           model.Name,
			Provider:       ProviderTypeLocal,
			ContextSize:    4096, // Default context size
			Capabilities:   []ModelCapability{CapabilityTextGeneration, CapabilityCodeGeneration, CapabilityCodeAnalysis},
			MaxTokens:      4096,
			SupportsTools:  false,
			SupportsVision: false,
			Description:    fmt.Sprintf("Ollama model: %s", model.Name),
		})
	}

	return modelInfos
}

// GetCapabilities returns model capabilities
func (p *OllamaProvider) GetCapabilities() []ModelCapability {
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

// Generate generates a response using Ollama
func (p *OllamaProvider) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	if !p.isRunning {
		return nil, ErrProviderUnavailable
	}

	// Prepare API request
	apiRequest := OllamaAPIRequest{
		Model:    p.getModelName(request.Model),
		Messages: request.Messages,
		Stream:   false,
		Options: map[string]interface{}{
			"temperature": request.Temperature,
			"top_p":       request.TopP,
			"num_predict": request.MaxTokens,
		},
	}

	// Make API call
	startTime := time.Now()
	response, err := p.makeAPIRequest(ctx, apiRequest)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	processingTime := time.Since(startTime)

	return &LLMResponse{
		ID:        uuid.New(),
		RequestID: request.ID,
		Content:   response.Response,
		Usage: Usage{
			PromptTokens:     response.PromptEvalCount,
			CompletionTokens: response.EvalCount,
			TotalTokens:      response.PromptEvalCount + response.EvalCount,
		},
		ProcessingTime: processingTime,
		CreatedAt:      time.Now(),
	}, nil
}

// GenerateStream generates a streaming response
func (p *OllamaProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
	if !p.isRunning {
		return ErrProviderUnavailable
	}

	// Prepare streaming API request
	apiRequest := OllamaAPIRequest{
		Model:    p.getModelName(request.Model),
		Messages: request.Messages,
		Stream:   true,
		Options: map[string]interface{}{
			"temperature": request.Temperature,
			"top_p":       request.TopP,
			"num_predict": request.MaxTokens,
		},
	}

	// Make streaming request
	return p.makeStreamingRequest(ctx, apiRequest, ch)
}

// IsAvailable checks if the provider is available
func (p *OllamaProvider) IsAvailable(ctx context.Context) bool {
	if !p.isRunning {
		return false
	}

	// Test API endpoint
	resp, err := p.apiClient.Get(p.getAPIURL("/api/tags"))
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GetHealth returns provider health status
func (p *OllamaProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	if !p.isRunning {
		return &ProviderHealth{
			Status:     "unhealthy",
			LastCheck:  time.Now(),
			ErrorCount: 1,
		}, nil
	}

	// Test API endpoint
	start := time.Now()
	resp, err := p.apiClient.Get(p.getAPIURL("/api/tags"))
	latency := time.Since(start)

	if err != nil || resp.StatusCode != http.StatusOK {
		return &ProviderHealth{
			Status:     "degraded",
			Latency:    latency,
			LastCheck:  time.Now(),
			ErrorCount: 1,
			ModelCount: len(p.models),
		}, nil
	}

	return &ProviderHealth{
		Status:     "healthy",
		Latency:    latency,
		LastCheck:  time.Now(),
		ErrorCount: 0,
		ModelCount: len(p.models),
	}, nil
}

// Close stops the Ollama provider
func (p *OllamaProvider) Close() error {
	p.isRunning = false
	log.Println("✅ Ollama provider closed")
	return nil
}

// Private helper methods

func (p *OllamaProvider) discoverModels() error {
	resp, err := p.apiClient.Get(p.getAPIURL("/api/tags"))
	if err != nil {
		return fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var response struct {
		Models []OllamaModel `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode models response: %w", err)
	}

	p.models = response.Models
	return nil
}

func (p *OllamaProvider) getModelName(requestedModel string) string {
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

	return "llama2" // Fallback default
}

func (p *OllamaProvider) getAPIURL(path string) string {
	baseURL := p.config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	return strings.TrimSuffix(baseURL, "/") + path
}

func (p *OllamaProvider) makeAPIRequest(ctx context.Context, request OllamaAPIRequest) (*OllamaAPIResponse, error) {
	url := p.getAPIURL("/api/chat")

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(requestBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.apiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var response OllamaAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

func (p *OllamaProvider) makeStreamingRequest(ctx context.Context, request OllamaAPIRequest, ch chan<- LLMResponse) error {
	// Simplified streaming implementation
	// In a real implementation, this would handle Server-Sent Events (SSE)

	url := p.getAPIURL("/api/chat")

	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(requestBody)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.apiClient.Do(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var response OllamaAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Send the complete response as a single chunk for now
	select {
	case <-ctx.Done():
		return ctx.Err()
	case ch <- LLMResponse{
		ID:        uuid.New(),
		Content:   response.Response,
		CreatedAt: time.Now(),
	}:
	}

	return nil
}
