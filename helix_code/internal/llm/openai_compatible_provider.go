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

// OpenAICompatibleProvider implements the Provider interface for OpenAI-compatible local services
// This includes VLLM, Text Generation WebUI, LM Studio, LocalAI, FastChat, Jan AI, and many others
type OpenAICompatibleProvider struct {
	name       string
	config     OpenAICompatibleConfig
	httpClient *http.Client
	models     []ModelInfo
	lastHealth *ProviderHealth
	isRunning  bool
}

// OpenAICompatibleConfig holds configuration for OpenAI-compatible providers
type OpenAICompatibleConfig struct {
	BaseURL          string            `json:"base_url"`
	APIKey           string            `json:"api_key"`
	DefaultModel     string            `json:"default_model"`
	Timeout          time.Duration     `json:"timeout"`
	MaxRetries       int               `json:"max_retries"`
	Headers          map[string]string `json:"headers"`
	StreamingSupport bool              `json:"streaming_support"`
	ModelEndpoint    string            `json:"model_endpoint"`
	ChatEndpoint     string            `json:"chat_endpoint"`
}

// OpenAICompatibleRequest represents an OpenAI-compatible API request.
// Messages uses OpenAIMessage (not llm.Message) so the assistant turn's
// tool_calls[].function.arguments serialises as a JSON STRING via
// wireSendToolCall — local backends fronted here (VLLM, LMStudio, …) follow the
// same OpenAI contract requiring string-encoded arguments.
type OpenAICompatibleRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Tools       []Tool          `json:"tools,omitempty"`
	ToolChoice  interface{}     `json:"tool_choice,omitempty"`
}

// OpenAICompatibleResponse represents an OpenAI-compatible API response
type OpenAICompatibleResponse struct {
	ID      string                   `json:"id"`
	Object  string                   `json:"object"`
	Created int64                    `json:"created"`
	Model   string                   `json:"model"`
	Choices []OpenAICompatibleChoice `json:"choices"`
	Usage   OpenAICompatibleUsage    `json:"usage"`
}

// OpenAICompatibleChoice represents a choice in the response
type OpenAICompatibleChoice struct {
	Index        int                     `json:"index"`
	Message      OpenAICompatibleMessage `json:"message,omitempty"`
	Delta        OpenAICompatibleDelta   `json:"delta,omitempty"`
	FinishReason string                  `json:"finish_reason"`
}

// OpenAICompatibleMessage represents a message in the response.
// ToolCalls uses openAIWireToolCall (not llm.ToolCall) because
// `function.arguments` arrives as a JSON STRING on the wire (OpenAI's canonical
// encoding); decoding straight into llm.ToolCall.Function.Arguments
// (map[string]interface{}) fails. parseOpenAIWireToolCalls handles both the
// string-encoded and raw-object forms — the same path the Groq/DeepSeek/
// Mistral/OpenRouter providers use.
type OpenAICompatibleMessage struct {
	Role             string               `json:"role"`
	Content          string               `json:"content"`
	ToolCalls        []openAIWireToolCall `json:"tool_calls,omitempty"`
	ReasoningContent string               `json:"reasoning_content,omitempty"`
}

// OpenAICompatibleDelta represents a delta in streaming response
type OpenAICompatibleDelta struct {
	Role             string               `json:"role,omitempty"`
	Content          string               `json:"content,omitempty"`
	ToolCalls        []openAIWireToolCall `json:"tool_calls,omitempty"`
	ReasoningContent string               `json:"reasoning_content,omitempty"`
}

// OpenAICompatibleUsage represents token usage information
type OpenAICompatibleUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAICompatibleModel represents a model in the API response
type OpenAICompatibleModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// NewOpenAICompatibleProvider creates a new OpenAI-compatible provider
func NewOpenAICompatibleProvider(name string, config OpenAICompatibleConfig) (*OpenAICompatibleProvider, error) {
	// Set default endpoints if not specified
	if config.ModelEndpoint == "" {
		config.ModelEndpoint = "/v1/models"
	}
	if config.ChatEndpoint == "" {
		config.ChatEndpoint = "/v1/chat/completions"
	}

	provider := &OpenAICompatibleProvider{
		name:   name,
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
		log.Printf("Warning: Failed to discover models for %s: %v", name, err)
	}

	log.Printf("✅ %s provider initialized with %d models", name, len(provider.models))
	return provider, nil
}

// BaseURL returns the provider's configured base URL. Exposed so out-of-package
// callers (and tests) can assert which URL a provider was constructed with —
// load-bearing for the CONST-036/CONST-046 guarantee that the dynamic catalogue
// uses the verifier's api_url rather than a hardcoded literal. Carries no
// credential (CONST-042).
func (p *OpenAICompatibleProvider) BaseURL() string {
	return p.config.BaseURL
}

// GetType returns the provider type
func (p *OpenAICompatibleProvider) GetType() ProviderType {
	switch p.name {
	case "vllm":
		return ProviderTypeVLLM
	case "localai":
		return ProviderTypeLocalAI
	case "fastchat":
		return ProviderTypeFastChat
	case "textgen":
		return ProviderTypeTextGen
	case "lmstudio":
		return ProviderTypeLMStudio
	case "jan":
		return ProviderTypeJan
	case "koboldai":
		return ProviderTypeKoboldAI
	case "gpt4all":
		return ProviderTypeGPT4All
	case "tabbyapi":
		return ProviderTypeTabbyAPI
	case "mlx":
		return ProviderTypeMLX
	case "mistralrs":
		return ProviderTypeMistralRS
	case "", "local":
		// Genuinely-unnamed / explicitly-local backends keep the generic type.
		return ProviderTypeLocal
	default:
		// Hosted OpenAI-compatible catalogue providers (cerebras, fireworks,
		// novita, …) are constructed with a distinct name. Returning the generic
		// ProviderTypeLocal here would mislabel EVERY such provider's
		// ModelInfo.Provider as "local" and collide them into one bucket. Derive a
		// stable distinct ProviderType from the name instead so each provider's
		// models are attributed to the correct provider (CONST-036 single-source
		// attribution). Known local backends are handled by the cases above.
		return ProviderType(p.name)
	}
}

// GetName returns the provider name
func (p *OpenAICompatibleProvider) GetName() string {
	return p.name
}

// GetModels returns available models
func (p *OpenAICompatibleProvider) GetModels() []ModelInfo {
	return p.models
}

// GetCapabilities returns model capabilities
func (p *OpenAICompatibleProvider) GetCapabilities() []ModelCapability {
	capabilities := []ModelCapability{
		CapabilityTextGeneration,
		CapabilityCodeGeneration,
		CapabilityCodeAnalysis,
		CapabilityPlanning,
		CapabilityDebugging,
		CapabilityRefactoring,
		CapabilityTesting,
	}

	// Add vision capability for models that might support it
	if p.supportsVision() {
		capabilities = append(capabilities, CapabilityVision)
	}

	return capabilities
}

// Generate generates a response using the OpenAI-compatible API
func (p *OpenAICompatibleProvider) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	if !p.isRunning {
		return nil, ErrProviderUnavailable
	}

	// Prepare API request
	apiRequest := p.convertToOpenAIRequest(request)

	// Make API call
	startTime := time.Now()
	response, err := p.makeAPIRequest(ctx, apiRequest)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	processingTime := time.Since(startTime)

	// Convert response
	llmResponse := p.convertFromOpenAIResponse(response, request.ID, processingTime)

	return llmResponse, nil
}

// GenerateStream generates a streaming response
func (p *OpenAICompatibleProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
	// Channel-ownership contract (see Provider.GenerateStream interface doc):
	// the provider (the SENDER) is the SOLE closer of ch, and MUST close it on
	// every return path — success, error, and ctx-cancel. The consumer never
	// closes ch; a double-close would panic in a spawned goroutine and crash the
	// process (server defect #5). defer guarantees close on every return path.
	defer close(ch)
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

	// Prepare streaming API request
	apiRequest := p.convertToOpenAIRequest(request)
	apiRequest.Stream = true

	// Make streaming request
	return p.makeStreamingRequest(ctx, apiRequest, ch, request.ID)
}

// IsAvailable checks if the provider is available
func (p *OpenAICompatibleProvider) IsAvailable(ctx context.Context) bool {
	if !p.isRunning {
		return false
	}

	health, err := p.GetHealth(ctx)
	return err == nil && health.Status == "healthy"
}

// GetHealth returns provider health status
func (p *OpenAICompatibleProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	if !p.isRunning {
		return &ProviderHealth{
			Status:     "unhealthy",
			LastCheck:  time.Now(),
			ErrorCount: 1,
		}, nil
	}

	// Test model endpoint for basic availability
	start := time.Now()
	url := p.getAPIURL(p.config.ModelEndpoint)
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

	// Try to parse models to get model count
	var modelsResponse struct {
		Data []OpenAICompatibleModel `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&modelsResponse); err != nil {
		p.updateHealth("degraded", latency, p.lastHealth.ErrorCount)
		return p.lastHealth, nil // Still consider it available
	}

	p.updateHealth("healthy", latency, 0)
	p.lastHealth.ModelCount = len(modelsResponse.Data)

	return p.lastHealth, nil
}

// Close stops the provider
func (p *OpenAICompatibleProvider) Close() error {
	p.isRunning = false
	if p.httpClient != nil {
		p.httpClient.CloseIdleConnections()
	}
	log.Printf("✅ %s provider closed", p.name)
	return nil
}

// GetContextWindow returns the model's context window size in tokens.
// Default: 200_000 — OpenAI-compatible providers (VLLM, LMStudio, etc.) vary;
// 200k is a safe upper bound pending a Phase 3 model-aware upgrade.
func (p *OpenAICompatibleProvider) GetContextWindow() int {
	return 200_000
}

// CountTokens returns an estimated token count for text.
// Uses char-based fallback (1 token ≈ 3.5 chars) — Phase 3 will upgrade
// to the provider's tokenize endpoint when available.
func (p *OpenAICompatibleProvider) CountTokens(text string) (int, error) {
	return CharBasedTokenCount(text)
}

// Private helper methods

func (p *OpenAICompatibleProvider) discoverModels() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := p.getAPIURL(p.config.ModelEndpoint)
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

	var response struct {
		Data []OpenAICompatibleModel `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode models response: %w", err)
	}

	// Convert to ModelInfo
	for _, model := range response.Data {
		modelInfo := ModelInfo{
			Name:        model.ID,
			Provider:    p.GetType(),
			ContextSize: p.inferContextSize(model.ID),
			MaxTokens:   p.inferMaxTokens(model.ID),
			Description: fmt.Sprintf("%s model: %s", p.name, model.ID),
		}
		p.models = append(p.models, modelInfo)
	}

	for i := range p.models {
		EnrichModelInfo(&p.models[i])
	}

	return nil
}

func (p *OpenAICompatibleProvider) convertToOpenAIRequest(request *LLMRequest) *OpenAICompatibleRequest {
	// Convert each message to the OpenAI wire shape so the assistant turn's
	// tool_calls serialise function.arguments as a JSON STRING (the
	// OpenAI-compatible contract); marshalling llm.Message directly would emit
	// the object form the backends reject. omitempty keeps plain-chat messages
	// (no tool_calls) byte-identical on the wire.
	messages := make([]OpenAIMessage, 0, len(request.Messages))
	for _, msg := range request.Messages {
		m := OpenAIMessage{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
			ToolCalls:  toWireSendToolCalls(msg.ToolCalls),
		}
		if msg.Name != "" {
			m.Name = msg.Name
		}
		messages = append(messages, m)
	}
	return &OpenAICompatibleRequest{
		Model:       p.getModelName(request.Model),
		Messages:    messages,
		MaxTokens:   request.MaxTokens,
		Temperature: request.Temperature,
		TopP:        request.TopP,
		Stream:      request.Stream,
		Tools:       request.Tools,
		ToolChoice:  request.ToolChoice,
	}
}

func (p *OpenAICompatibleProvider) convertFromOpenAIResponse(response *OpenAICompatibleResponse, requestID uuid.UUID, processingTime time.Duration) *LLMResponse {
	llmResponse := &LLMResponse{
		ID:        uuid.New(),
		RequestID: requestID,
		Content:   "",
		Usage:     Usage{},
		// The backend reports which concrete model served the request. Carry it
		// through so callers surface the REAL identity instead of echoing the
		// requested alias (CONST-036 / CONST-037). Empty when the backend omits
		// it — consumers fall back to the requested model.
		Model:          response.Model,
		ProcessingTime: processingTime,
		CreatedAt:      time.Now(),
	}

	if len(response.Choices) > 0 {
		choice := response.Choices[0]
		llmResponse.Content = choice.Message.Content
		llmResponse.ToolCalls = parseOpenAIWireToolCalls(choice.Message.ToolCalls)
		llmResponse.FinishReason = choice.FinishReason

		// Capture reasoning_content if present (Xiaomi MiMo deep thinking)
		if choice.Message.ReasoningContent != "" {
			if llmResponse.ProviderMetadata == nil {
				llmResponse.ProviderMetadata = make(map[string]interface{})
			}
			llmResponse.ProviderMetadata["reasoning_content"] = choice.Message.ReasoningContent
		}
		// Round-53 LLMResponse.Err wiring (CONST-035 / Article XI §11.9):
		// the OpenAICompatibleProvider fans out to ~11 backends (VLLM,
		// LMStudio, Jan, LocalAI, FastChat, TextGen WebUI, KoboldAI,
		// GPT4All, TabbyAPI, MLX, MistralRS) — all advertise
		// OpenAI-compatible Chat Completions semantics including
		// `finish_reason: "length" | "stop" | "tool_calls" |
		// "content_filter"`. Reuse the round-46 OpenAI mapper. If any
		// individual backend diverges from this contract in the
		// future, TestRound53_OpenAICompatible_ReusesOpenAIMapper will
		// surface the regression and a backend-specific path MUST be
		// introduced in the same commit.
		llmResponse.Err = mapOpenAIFinishReasonToErr(choice.FinishReason)
	}

	llmResponse.Usage = Usage{
		PromptTokens:     response.Usage.PromptTokens,
		CompletionTokens: response.Usage.CompletionTokens,
		TotalTokens:      response.Usage.TotalTokens,
	}

	return llmResponse
}

func (p *OpenAICompatibleProvider) makeAPIRequest(ctx context.Context, request *OpenAICompatibleRequest) (*OpenAICompatibleResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := p.getAPIURL(p.config.ChatEndpoint)
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

	var response OpenAICompatibleResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

func (p *OpenAICompatibleProvider) makeStreamingRequest(ctx context.Context, request *OpenAICompatibleRequest, ch chan<- LLMResponse, requestID uuid.UUID) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := p.getAPIURL(p.config.ChatEndpoint)
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

			var streamResponse struct {
				ID      string                   `json:"id"`
				Object  string                   `json:"object"`
				Created int64                    `json:"created"`
				Model   string                   `json:"model"`
				Choices []OpenAICompatibleChoice `json:"choices"`
			}

			if err := json.Unmarshal([]byte(data), &streamResponse); err != nil {
				continue // Skip malformed JSON
			}

			if len(streamResponse.Choices) > 0 {
				choice := streamResponse.Choices[0]
				response := LLMResponse{
					ID:        uuid.New(),
					RequestID: requestID,
					Content:   choice.Delta.Content,
					// Carry the backend-reported model on EVERY chunk. Without
					// it the wire facade's streaming path has nothing to
					// upgrade from and keeps echoing the requested alias, so a
					// `stream:true` request reports "default" while a concrete
					// model serves it — the same CONST-036 / CONST-037 defect
					// the non-stream path fixes. Empty when the backend omits
					// it; the facade then keeps the requested model.
					Model:     streamResponse.Model,
					CreatedAt: time.Now(),
				}

				// Capture streaming reasoning_content if present (Xiaomi MiMo deep thinking)
				if choice.Delta.ReasoningContent != "" {
					response.ProviderMetadata = map[string]interface{}{
						"reasoning_content": choice.Delta.ReasoningContent,
					}
				}

				select {
				case ch <- response:
				case <-ctx.Done():
					return ctx.Err()
				}

				if choice.FinishReason != "" {
					// Round-53 LLMResponse.Err wiring for the streaming
					// path (CONST-035 / Article XI §11.9): when the
					// final frame carries finish_reason="length"/
					// "content_filter", emit a terminal LLMResponse
					// with Err populated so stream consumers (notably
					// tool_provider.go :201/:251) can distinguish a
					// clean stop from a partial-error stop on any of
					// the 11 OpenAI-compatible local backends fronted
					// by this provider (VLLM, LMStudio, Jan, etc.).
					if errSentinel := mapOpenAIFinishReasonToErr(choice.FinishReason); errSentinel != nil {
						select {
						case ch <- LLMResponse{
							ID:           uuid.New(),
							RequestID:    requestID,
							FinishReason: choice.FinishReason,
							Model:        streamResponse.Model,
							CreatedAt:    time.Now(),
							Err:          errSentinel,
						}:
						case <-ctx.Done():
							return ctx.Err()
						}
					}
					break
				}
			}
		}
	}

	return nil
}

func (p *OpenAICompatibleProvider) getModelName(requestedModel string) string {
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

	return "gpt-3.5-turbo" // Fallback default
}

func (p *OpenAICompatibleProvider) getAPIURL(endpoint string) string {
	baseURL := p.config.BaseURL
	if baseURL == "" {
		switch p.name {
		case "vllm":
			baseURL = "http://localhost:8000"
		case "textgen", "oobabooga":
			baseURL = "http://localhost:5000"
		case "lmstudio":
			baseURL = "http://localhost:1234"
		case "localai":
			baseURL = "http://localhost:8080"
		case "jan":
			baseURL = "http://localhost:1337"
		case "koboldai":
			baseURL = "http://localhost:5001"
		case "gpt4all":
			baseURL = "http://localhost:4891"
		case "tabbyapi":
			baseURL = "http://localhost:5000"
		case "fastchat":
			baseURL = "http://localhost:7860"
		default:
			baseURL = "http://localhost:8080"
		}
	}

	return strings.TrimSuffix(baseURL, "/") + endpoint
}

func (p *OpenAICompatibleProvider) updateHealth(status string, latency time.Duration, errorCount int) {
	p.lastHealth.Status = status
	p.lastHealth.Latency = latency
	p.lastHealth.ErrorCount = errorCount
	p.lastHealth.LastCheck = time.Now()
}

// Helper functions for model capabilities

func (p *OpenAICompatibleProvider) inferContextSize(modelName string) int {
	modelName = strings.ToLower(modelName)

	// Common context sizes based on model names
	if strings.Contains(modelName, "32k") || strings.Contains(modelName, "32k") {
		return 32768
	}
	if strings.Contains(modelName, "16k") {
		return 16384
	}
	if strings.Contains(modelName, "8k") {
		return 8192
	}
	if strings.Contains(modelName, "gpt-4") {
		return 8192
	}
	if strings.Contains(modelName, "claude") {
		return 100000
	}
	if strings.Contains(modelName, "llama") {
		return 4096
	}

	return 4096 // Default
}

func (p *OpenAICompatibleProvider) inferMaxTokens(modelName string) int {
	contextSize := p.inferContextSize(modelName)
	return contextSize / 2 // Conservative estimate
}

func (p *OpenAICompatibleProvider) supportsTools(modelName string) bool {
	modelName = strings.ToLower(modelName)

	// Most modern models support tools
	return strings.Contains(modelName, "gpt-4") ||
		strings.Contains(modelName, "claude-3") ||
		strings.Contains(modelName, "llama-3") ||
		strings.Contains(modelName, "mistral")
}

func (p *OpenAICompatibleProvider) supportsVision() bool {
	// Most modern providers support vision
	return p.name == "lmstudio" || p.name == "jan" || p.name == "textgen"
}

func (p *OpenAICompatibleProvider) supportsVisionModel(modelName string) bool {
	modelName = strings.ToLower(modelName)
	return strings.Contains(modelName, "vision") ||
		strings.Contains(modelName, "multimodal") ||
		strings.Contains(modelName, "clip") ||
		strings.Contains(modelName, "llava")
}

// readSSELine reads a line from Server-Sent Events stream
func readSSELine(r io.Reader) (string, error) {
	var line []byte
	buf := make([]byte, 1)

	for {
		n, err := r.Read(buf)
		if err != nil {
			return "", err
		}
		if n == 0 {
			continue
		}

		if buf[0] == '\n' {
			break
		}

		if buf[0] != '\r' {
			line = append(line, buf[0])
		}
	}

	return string(line), nil
}
