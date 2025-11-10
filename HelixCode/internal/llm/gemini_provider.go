package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// GeminiProvider implements the Provider interface for Google's Gemini models
type GeminiProvider struct {
	config     ProviderConfigEntry
	apiKey     string
	endpoint   string
	httpClient *http.Client
	models     []ModelInfo
	lastHealth *ProviderHealth
}

// Gemini API structures
type geminiRequest struct {
	Contents          []geminiContent       `json:"contents"`
	SystemInstruction *geminiContent        `json:"systemInstruction,omitempty"`
	Tools             []geminiTool          `json:"tools,omitempty"`
	ToolConfig        *geminiToolConfig     `json:"toolConfig,omitempty"`
	SafetySettings    []geminiSafetySetting `json:"safetySettings,omitempty"`
	GenerationConfig  *geminiGenConfig      `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"` // "user" or "model"
	Parts []geminiPart `json:"parts"`
}

type geminiPart interface{}

type geminiTextPart struct {
	Text string `json:"text"`
}

type geminiInlineDataPart struct {
	InlineData *geminiBlob `json:"inlineData"`
}

type geminiBlob struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"` // base64 encoded
}

type geminiFunctionCallPart struct {
	FunctionCall *geminiFunctionCall `json:"functionCall"`
}

type geminiFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type geminiFunctionResponsePart struct {
	FunctionResponse *geminiFunctionResponse `json:"functionResponse"`
}

type geminiFunctionResponse struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDeclaration `json:"functionDeclarations,omitempty"`
}

type geminiFunctionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type geminiToolConfig struct {
	FunctionCallingConfig *geminiFunctionCallingConfig `json:"functionCallingConfig,omitempty"`
}

type geminiFunctionCallingConfig struct {
	Mode string `json:"mode"` // "AUTO", "ANY", "NONE"
}

type geminiSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type geminiGenConfig struct {
	Temperature     float64  `json:"temperature,omitempty"`
	TopP            float64  `json:"topP,omitempty"`
	TopK            int      `json:"topK,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

type geminiResponse struct {
	Candidates     []geminiCandidate     `json:"candidates"`
	UsageMetadata  *geminiUsageMetadata  `json:"usageMetadata,omitempty"`
	PromptFeedback *geminiPromptFeedback `json:"promptFeedback,omitempty"`
}

type geminiCandidate struct {
	Content       geminiContent        `json:"content"`
	FinishReason  string               `json:"finishReason"`
	SafetyRatings []geminiSafetyRating `json:"safetyRatings,omitempty"`
	Index         int                  `json:"index"`
}

type geminiSafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

type geminiPromptFeedback struct {
	BlockReason   string               `json:"blockReason,omitempty"`
	SafetyRatings []geminiSafetyRating `json:"safetyRatings,omitempty"`
}

type geminiUsageMetadata struct {
	PromptTokenCount        int `json:"promptTokenCount"`
	CandidatesTokenCount    int `json:"candidatesTokenCount"`
	TotalTokenCount         int `json:"totalTokenCount"`
	CachedContentTokenCount int `json:"cachedContentTokenCount,omitempty"`
}

type geminiError struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// NewGeminiProvider creates a new Gemini provider
func NewGeminiProvider(config ProviderConfigEntry) (*GeminiProvider, error) {
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}

	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}

	if apiKey == "" {
		return nil, fmt.Errorf("gemini API key not provided (set GEMINI_API_KEY or GOOGLE_API_KEY)")
	}

	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "https://generativelanguage.googleapis.com/v1beta"
	}

	provider := &GeminiProvider{
		config:   config,
		apiKey:   apiKey,
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		models: getGeminiModels(),
	}

	return provider, nil
}

// getGeminiModels returns all available Gemini models
// Based on OpenCode and Qwen Code implementations
func getGeminiModels() []ModelInfo {
	allCapabilities := []ModelCapability{
		CapabilityTextGeneration,
		CapabilityCodeGeneration,
		CapabilityCodeAnalysis,
		CapabilityPlanning,
		CapabilityDebugging,
		CapabilityRefactoring,
		CapabilityTesting,
	}

	visionCapabilities := append(allCapabilities, CapabilityVision)

	return []ModelInfo{
		// Gemini 2.5 family (latest)
		{
			Name:           "gemini-2.5-pro",
			Provider:       ProviderTypeGemini,
			ContextSize:    2097152, // 2M tokens
			MaxTokens:      8192,
			Capabilities:   visionCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Gemini 2.5 Pro - Most capable model with 2M context window",
		},
		{
			Name:           "gemini-2.5-flash",
			Provider:       ProviderTypeGemini,
			ContextSize:    1048576, // 1M tokens
			MaxTokens:      8192,
			Capabilities:   visionCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Gemini 2.5 Flash - Fast and efficient with 1M context",
		},
		{
			Name:           "gemini-2.5-flash-lite",
			Provider:       ProviderTypeGemini,
			ContextSize:    1048576,
			MaxTokens:      8192,
			Capabilities:   visionCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Gemini 2.5 Flash Lite - Cost-effective with full capabilities",
		},
		// Gemini 2.0 family
		{
			Name:           "gemini-2.0-flash-001",
			Provider:       ProviderTypeGemini,
			ContextSize:    1048576,
			MaxTokens:      8192,
			Capabilities:   visionCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Gemini 2.0 Flash - Fast multimodal model",
		},
		{
			Name:           "gemini-2.0-flash",
			Provider:       ProviderTypeGemini,
			ContextSize:    1048576,
			MaxTokens:      8192,
			Capabilities:   visionCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Gemini 2.0 Flash (Latest) - Always latest 2.0 Flash version",
		},
		{
			Name:           "gemini-2.0-flash-lite",
			Provider:       ProviderTypeGemini,
			ContextSize:    32000,
			MaxTokens:      8192,
			Capabilities:   visionCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Gemini 2.0 Flash Lite - Fast image understanding with smaller context",
		},
		// Gemini 1.5 family
		{
			Name:           "gemini-1.5-pro",
			Provider:       ProviderTypeGemini,
			ContextSize:    2097152, // 2M tokens
			MaxTokens:      8192,
			Capabilities:   visionCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Gemini 1.5 Pro - Powerful with 2M context window",
		},
		{
			Name:           "gemini-1.5-flash",
			Provider:       ProviderTypeGemini,
			ContextSize:    1048576,
			MaxTokens:      8192,
			Capabilities:   visionCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Gemini 1.5 Flash - Fast and efficient",
		},
		{
			Name:           "gemini-1.5-flash-8b",
			Provider:       ProviderTypeGemini,
			ContextSize:    1048576,
			MaxTokens:      8192,
			Capabilities:   visionCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Gemini 1.5 Flash 8B - Smaller, faster variant",
		},
		// Gemini 1.0 family
		{
			Name:           "gemini-1.0-pro",
			Provider:       ProviderTypeGemini,
			ContextSize:    32000,
			MaxTokens:      8192,
			Capabilities:   allCapabilities,
			SupportsTools:  true,
			SupportsVision: false,
			Description:    "Gemini 1.0 Pro - Reliable text-only model",
		},
		// Embedding models
		{
			Name:           "gemini-embedding-001",
			Provider:       ProviderTypeGemini,
			ContextSize:    2048,
			MaxTokens:      0,
			Capabilities:   []ModelCapability{CapabilityTextGeneration},
			SupportsTools:  false,
			SupportsVision: false,
			Description:    "Gemini Embedding - Text embeddings for semantic search",
		},
	}
}

// GetType returns the provider type
func (gp *GeminiProvider) GetType() ProviderType {
	return ProviderTypeGemini
}

// GetName returns the provider name
func (gp *GeminiProvider) GetName() string {
	return "Gemini"
}

// GetModels returns available models
func (gp *GeminiProvider) GetModels() []ModelInfo {
	return gp.models
}

// GetCapabilities returns provider capabilities
func (gp *GeminiProvider) GetCapabilities() []ModelCapability {
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

// Generate generates a response using Gemini
func (gp *GeminiProvider) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	startTime := time.Now()

	// Build Gemini request
	geminiReq, err := gp.buildRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %v", err)
	}

	// Make API request
	resp, err := gp.makeRequest(ctx, request.Model, geminiReq)
	if err != nil {
		return nil, err
	}

	// Parse response
	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	candidate := resp.Candidates[0]
	response := &LLMResponse{
		ID:             uuid.New(),
		RequestID:      request.ID,
		ProcessingTime: time.Since(startTime),
		CreatedAt:      time.Now(),
		FinishReason:   candidate.FinishReason,
	}

	// Extract content and tool calls
	for _, part := range candidate.Content.Parts {
		switch p := part.(type) {
		case map[string]interface{}:
			if text, ok := p["text"].(string); ok {
				response.Content += text
			}
			if fc, ok := p["functionCall"].(map[string]interface{}); ok {
				name, _ := fc["name"].(string)
				args, _ := fc["args"].(map[string]interface{})
				response.ToolCalls = append(response.ToolCalls, ToolCall{
					ID:   uuid.New().String(),
					Type: "function",
					Function: ToolCallFunc{
						Name:      name,
						Arguments: args,
					},
				})
			}
		}
	}

	// Add usage information
	if resp.UsageMetadata != nil {
		response.Usage = Usage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		}

		// Add caching metadata if present
		if resp.UsageMetadata.CachedContentTokenCount > 0 {
			response.ProviderMetadata = map[string]interface{}{
				"cached_content_tokens": resp.UsageMetadata.CachedContentTokenCount,
			}
		}
	}

	return response, nil
}

// GenerateStream generates a streaming response
func (gp *GeminiProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
	defer close(ch)

	// Build request
	geminiReq, err := gp.buildRequest(request)
	if err != nil {
		return fmt.Errorf("failed to build request: %v", err)
	}

	// Make streaming request
	return gp.makeStreamingRequest(ctx, request.Model, geminiReq, ch, request.ID)
}

// buildRequest constructs a Gemini API request
func (gp *GeminiProvider) buildRequest(request *LLMRequest) (*geminiRequest, error) {
	req := &geminiRequest{
		GenerationConfig: &geminiGenConfig{
			Temperature:     request.Temperature,
			TopP:            request.TopP,
			MaxOutputTokens: request.MaxTokens,
		},
	}

	// Default max tokens if not specified
	if req.GenerationConfig.MaxOutputTokens == 0 {
		req.GenerationConfig.MaxOutputTokens = 4096
	}

	// Convert messages
	systemMsg, contents := gp.convertMessages(request.Messages)
	req.Contents = contents

	// Set system instruction if present
	if systemMsg != "" {
		req.SystemInstruction = &geminiContent{
			Parts: []geminiPart{
				map[string]interface{}{"text": systemMsg},
			},
		}
	}

	// Convert tools
	if len(request.Tools) > 0 {
		req.Tools = gp.convertTools(request.Tools)
		req.ToolConfig = &geminiToolConfig{
			FunctionCallingConfig: &geminiFunctionCallingConfig{
				Mode: "AUTO",
			},
		}
	}

	// Set safety settings to permissive for development use
	// This prevents blocking on code-related content
	req.SafetySettings = []geminiSafetySetting{
		{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_ONLY_HIGH"},
		{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_ONLY_HIGH"},
		{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_ONLY_HIGH"},
		{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_ONLY_HIGH"},
	}

	return req, nil
}

// convertMessages converts LLM messages to Gemini format
func (gp *GeminiProvider) convertMessages(messages []Message) (string, []geminiContent) {
	var systemMsg string
	var contents []geminiContent

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			systemMsg = msg.Content
		case "user":
			contents = append(contents, geminiContent{
				Role: "user",
				Parts: []geminiPart{
					map[string]interface{}{"text": msg.Content},
				},
			})
		case "assistant":
			contents = append(contents, geminiContent{
				Role: "model", // Gemini uses "model" instead of "assistant"
				Parts: []geminiPart{
					map[string]interface{}{"text": msg.Content},
				},
			})
		}
	}

	return systemMsg, contents
}

// convertTools converts LLM tools to Gemini format
func (gp *GeminiProvider) convertTools(tools []Tool) []geminiTool {
	declarations := make([]geminiFunctionDeclaration, len(tools))

	for i, tool := range tools {
		declarations[i] = geminiFunctionDeclaration{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			Parameters:  tool.Function.Parameters,
		}
	}

	return []geminiTool{
		{FunctionDeclarations: declarations},
	}
}

// makeRequest makes a non-streaming API request
func (gp *GeminiProvider) makeRequest(ctx context.Context, model string, request *geminiRequest) (*geminiResponse, error) {
	// Build URL with API key
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", gp.endpoint, model, gp.apiKey)

	// Marshal request
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Make request
	httpResp, err := gp.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Handle errors
	if httpResp.StatusCode != http.StatusOK {
		var apiErr geminiError
		if err := json.Unmarshal(respBody, &apiErr); err == nil {
			return nil, fmt.Errorf("gemini API error (%d): %s - %s",
				apiErr.Error.Code, apiErr.Error.Status, apiErr.Error.Message)
		}
		return nil, fmt.Errorf("gemini API error (%d): %s", httpResp.StatusCode, string(respBody))
	}

	// Parse successful response
	var response geminiResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &response, nil
}

// makeStreamingRequest makes a streaming API request
func (gp *GeminiProvider) makeStreamingRequest(ctx context.Context, model string, request *geminiRequest,
	ch chan<- LLMResponse, requestID uuid.UUID) error {

	// Build URL for streaming
	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?key=%s&alt=sse", gp.endpoint, model, gp.apiKey)

	// Marshal request
	reqBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqBody)))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	// Make request
	httpResp, err := gp.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("gemini API error (%d): %s", httpResp.StatusCode, string(body))
	}

	// Parse streaming response
	return gp.parseStreamingResponse(httpResp.Body, ch, requestID)
}

// parseStreamingResponse parses SSE streaming response
func (gp *GeminiProvider) parseStreamingResponse(body io.Reader, ch chan<- LLMResponse, requestID uuid.UUID) error {
	decoder := json.NewDecoder(body)
	var currentContent strings.Builder

	for {
		var response geminiResponse
		if err := decoder.Decode(&response); err != nil {
			if err == io.EOF {
				break
			}
			continue // Skip decode errors (SSE comments, etc.)
		}

		if len(response.Candidates) == 0 {
			continue
		}

		candidate := response.Candidates[0]

		// Extract text content
		for _, part := range candidate.Content.Parts {
			if p, ok := part.(map[string]interface{}); ok {
				if text, ok := p["text"].(string); ok {
					currentContent.WriteString(text)

					// Send incremental response
					ch <- LLMResponse{
						ID:        uuid.New(),
						RequestID: requestID,
						Content:   text,
						CreatedAt: time.Now(),
					}
				}
			}
		}

		// Send final response if finished
		if candidate.FinishReason != "" && candidate.FinishReason != "STOP" {
			finalResponse := LLMResponse{
				ID:           uuid.New(),
				RequestID:    requestID,
				Content:      currentContent.String(),
				FinishReason: candidate.FinishReason,
				CreatedAt:    time.Now(),
			}

			if response.UsageMetadata != nil {
				finalResponse.Usage = Usage{
					PromptTokens:     response.UsageMetadata.PromptTokenCount,
					CompletionTokens: response.UsageMetadata.CandidatesTokenCount,
					TotalTokens:      response.UsageMetadata.TotalTokenCount,
				}
			}

			ch <- finalResponse
		}
	}

	return nil
}

// IsAvailable checks if the provider is available
func (gp *GeminiProvider) IsAvailable(ctx context.Context) bool {
	return gp.apiKey != ""
}

// GetHealth returns the health status of the provider
func (gp *GeminiProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	startTime := time.Now()

	health := &ProviderHealth{
		LastCheck:  time.Now(),
		ModelCount: len(gp.models),
	}

	// Test with a minimal request
	testReq := &LLMRequest{
		ID:          uuid.New(),
		Model:       "gemini-2.5-flash-lite",
		Messages:    []Message{{Role: "user", Content: "Hi"}},
		MaxTokens:   10,
		Temperature: 0.1,
	}

	_, err := gp.Generate(ctx, testReq)
	if err != nil {
		health.Status = "unhealthy"
		health.ErrorCount = 1
		return health, err
	}

	health.Status = "healthy"
	health.Latency = time.Since(startTime)
	gp.lastHealth = health

	return health, nil
}

// Close closes the provider and cleans up resources
func (gp *GeminiProvider) Close() error {
	gp.httpClient.CloseIdleConnections()
	log.Printf("Gemini provider closed")
	return nil
}
