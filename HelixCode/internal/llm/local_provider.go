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

// LocalProvider implements the Provider interface for local LLama.cpp models
type LocalProvider struct {
	config     ProviderConfigEntry
	endpoint   string
	httpClient *http.Client
	models     []ModelInfo
	lastHealth *ProviderHealth
}

// NewLocalProvider creates a new local provider
func NewLocalProvider(config ProviderConfigEntry) (*LocalProvider, error) {
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	provider := &LocalProvider{
		config:   config,
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		lastHealth: &ProviderHealth{
			Status:    "unknown",
			LastCheck: time.Now(),
		},
	}

	// Initialize models
	if err := provider.initializeModels(); err != nil {
		log.Printf("Warning: Failed to initialize local provider models: %v", err)
	}

	return provider, nil
}

// GetType returns the provider type
func (lp *LocalProvider) GetType() ProviderType {
	return ProviderTypeLocal
}

// GetName returns the provider name
func (lp *LocalProvider) GetName() string {
	return "Local LLama.cpp"
}

// GetModels returns available models
func (lp *LocalProvider) GetModels() []ModelInfo {
	return lp.models
}

// GetCapabilities returns provider capabilities
func (lp *LocalProvider) GetCapabilities() []ModelCapability {
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

// Generate generates a response using local models
func (lp *LocalProvider) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	startTime := time.Now()

	// Convert to Ollama-compatible format
	ollamaRequest, err := lp.convertToOllamaRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %v", err)
	}

	// Make request to Ollama API
	response, err := lp.makeOllamaRequest(ctx, ollamaRequest)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %v", err)
	}

	// Convert response
	llmResponse := lp.convertFromOllamaResponse(response, request.ID, time.Since(startTime))

	return llmResponse, nil
}

// GenerateStream generates a streaming response
func (lp *LocalProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
	defer close(ch)

	// Convert to Ollama-compatible format
	ollamaRequest, err := lp.convertToOllamaRequest(request)
	if err != nil {
		return fmt.Errorf("failed to convert request: %v", err)
	}

	// Enable streaming
	ollamaRequest.Stream = true

	// Make streaming request
	return lp.makeOllamaStreamRequest(ctx, ollamaRequest, ch, request.ID)
}

// IsAvailable checks if the provider is available
func (lp *LocalProvider) IsAvailable(ctx context.Context) bool {
	health, err := lp.GetHealth(ctx)
	return err == nil && health.Status == "healthy"
}

// GetHealth returns provider health status
func (lp *LocalProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	// Check if we can reach the Ollama API
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/tags", lp.endpoint), nil)
	if err != nil {
		lp.updateHealth("unhealthy", 0, 1)
		return lp.lastHealth, fmt.Errorf("failed to create health check request: %v", err)
	}

	start := time.Now()
	resp, err := lp.httpClient.Do(req)
	latency := time.Since(start)

	if err != nil {
		lp.updateHealth("unhealthy", latency, lp.lastHealth.ErrorCount+1)
		return lp.lastHealth, fmt.Errorf("health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		lp.updateHealth("unhealthy", latency, lp.lastHealth.ErrorCount+1)
		return lp.lastHealth, fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	// Parse response to get model count
	var tagsResponse struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tagsResponse); err != nil {
		lp.updateHealth("degraded", latency, lp.lastHealth.ErrorCount)
		return lp.lastHealth, nil // Still consider it available
	}

	lp.updateHealth("healthy", latency, 0)
	lp.lastHealth.ModelCount = len(tagsResponse.Models)

	return lp.lastHealth, nil
}

// Close closes the provider
func (lp *LocalProvider) Close() error {
	lp.httpClient.CloseIdleConnections()
	return nil
}

// Helper methods

func (lp *LocalProvider) initializeModels() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/tags", lp.endpoint), nil)
	if err != nil {
		return err
	}

	resp, err := lp.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get models: status %d", resp.StatusCode)
	}

	var tagsResponse struct {
		Models []struct {
			Name       string `json:"name"`
			ModifiedAt string `json:"modified_at"`
			Size       int64  `json:"size"`
			Digest     string `json:"digest"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tagsResponse); err != nil {
		return err
	}

	// Convert to ModelInfo
	for _, model := range tagsResponse.Models {
		modelInfo := ModelInfo{
			Name:           model.Name,
			Provider:       ProviderTypeLocal,
			ContextSize:    4096, // Default for most models
			MaxTokens:      2048,
			Capabilities:   lp.GetCapabilities(),
			SupportsTools:  false, // Ollama doesn't support tools yet
			SupportsVision: strings.Contains(strings.ToLower(model.Name), "vision"),
			Description:    fmt.Sprintf("Local model: %s", model.Name),
		}
		lp.models = append(lp.models, modelInfo)
	}

	log.Printf("✅ Local provider initialized with %d models", len(lp.models))
	return nil
}

func (lp *LocalProvider) convertToOllamaRequest(request *LLMRequest) (*OllamaRequest, error) {
	// Build prompt from messages
	var prompt strings.Builder
	for _, msg := range request.Messages {
		switch msg.Role {
		case "system":
			prompt.WriteString(fmt.Sprintf("System: %s\n", msg.Content))
		case "user":
			prompt.WriteString(fmt.Sprintf("User: %s\n", msg.Content))
		case "assistant":
			prompt.WriteString(fmt.Sprintf("Assistant: %s\n", msg.Content))
		}
	}
	prompt.WriteString("Assistant: ")

	return &OllamaRequest{
		Model:  request.Model,
		Prompt: prompt.String(),
		Options: map[string]interface{}{
			"temperature": request.Temperature,
			"top_p":       request.TopP,
			"num_predict": request.MaxTokens,
		},
		Stream: request.Stream,
	}, nil
}

func (lp *LocalProvider) convertFromOllamaResponse(ollamaResp *OllamaResponse, requestID uuid.UUID, processingTime time.Duration) *LLMResponse {
	return &LLMResponse{
		ID:        uuid.New(),
		RequestID: requestID,
		Content:   ollamaResp.Response,
		Usage: Usage{
			PromptTokens:     ollamaResp.PromptEvalCount,
			CompletionTokens: ollamaResp.EvalCount,
			TotalTokens:      ollamaResp.PromptEvalCount + ollamaResp.EvalCount,
		},
		FinishReason:   "stop",
		ProcessingTime: processingTime,
		CreatedAt:      time.Now(),
	}
}

func (lp *LocalProvider) makeOllamaRequest(ctx context.Context, request *OllamaRequest) (*OllamaResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/api/generate", lp.endpoint), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := lp.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (lp *LocalProvider) makeOllamaStreamRequest(ctx context.Context, request *OllamaRequest, ch chan<- LLMResponse, requestID uuid.UUID) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/api/generate", lp.endpoint), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := lp.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Stream responses
	decoder := json.NewDecoder(resp.Body)
	for decoder.More() {
		var streamResp OllamaStreamResponse
		if err := decoder.Decode(&streamResp); err != nil {
			return err
		}

		response := LLMResponse{
			ID:        uuid.New(),
			RequestID: requestID,
			Content:   streamResp.Response,
			CreatedAt: time.Now(),
		}

		select {
		case ch <- response:
		case <-ctx.Done():
			return ctx.Err()
		}

		if streamResp.Done {
			break
		}
	}

	return nil
}

func (lp *LocalProvider) updateHealth(status string, latency time.Duration, errorCount int) {
	lp.lastHealth.Status = status
	lp.lastHealth.Latency = latency
	lp.lastHealth.ErrorCount = errorCount
	lp.lastHealth.LastCheck = time.Now()
}

// Ollama API types

type OllamaRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Options map[string]interface{} `json:"options"`
	Stream  bool                   `json:"stream"`
}

type OllamaResponse struct {
	Model           string `json:"model"`
	CreatedAt       string `json:"created_at"`
	Response        string `json:"response"`
	Done            bool   `json:"done"`
	Context         []int  `json:"context"`
	TotalDuration   int64  `json:"total_duration"`
	LoadDuration    int64  `json:"load_duration"`
	PromptEvalCount int    `json:"prompt_eval_count"`
	EvalCount       int    `json:"eval_count"`
	EvalDuration    int64  `json:"eval_duration"`
}

type OllamaStreamResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
}
