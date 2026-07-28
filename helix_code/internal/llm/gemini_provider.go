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

	"dev.helix.code/internal/llm/promptcache"
	"dev.helix.code/internal/providers/httpclient"
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

	// prefixDetector watches the request prefix (system instruction + tool
	// declarations) for mid-session drift. Speed programme P1-T05: Gemini 2.5
	// performs IMPLICIT context caching automatically for a stable prefix and
	// reports it via usageMetadata.cachedContentTokenCount. There is no
	// per-request flag for implicit caching, so prefix stability is the whole
	// game. The detector freezes the prefix on the first request and logs a
	// "cache break" if a later request mutates it. Observability only — it
	// never alters the request, so feature behaviour is unaffected.
	prefixDetector *promptcache.CacheBreakDetector
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
		// Shared tuned HTTP/2 transport (speed programme P1-T01,
		// R1 B03 / R3 §4.7) — connection pooling only; request
		// behaviour is unchanged.
		httpClient: httpclient.NewHTTPClient(120 * time.Second),
		models:     getGeminiModels(),
		// Speed programme P1-T05: prompt-cache prefix-stability detector.
		prefixDetector: promptcache.NewCacheBreakDetector(),
	}

	return provider, nil
}

// getGeminiModels returns all available Gemini models
// Based on OpenCode and Qwen Code implementations
func getGeminiModels() []ModelInfo {

	models := []ModelInfo{
		// Gemini 2.5 family (latest)
		{
			Name:        "gemini-2.5-pro",
			Provider:    ProviderTypeGemini,
			ContextSize: 2097152, // 2M tokens
			MaxTokens:   8192,
			Description: "Gemini 2.5 Pro - Most capable model with 2M context window",
		},
		{
			Name:        "gemini-2.5-flash",
			Provider:    ProviderTypeGemini,
			ContextSize: 1048576, // 1M tokens
			MaxTokens:   8192,
			Description: "Gemini 2.5 Flash - Fast and efficient with 1M context",
		},
		{
			Name:        "gemini-2.5-flash-lite",
			Provider:    ProviderTypeGemini,
			ContextSize: 1048576,
			MaxTokens:   8192,
			Description: "Gemini 2.5 Flash Lite - Cost-effective with full capabilities",
		},
		// Gemini 2.0 family
		{
			Name:        "gemini-2.0-flash-001",
			Provider:    ProviderTypeGemini,
			ContextSize: 1048576,
			MaxTokens:   8192,
			Description: "Gemini 2.0 Flash - Fast multimodal model",
		},
		{
			Name:        "gemini-2.0-flash",
			Provider:    ProviderTypeGemini,
			ContextSize: 1048576,
			MaxTokens:   8192,
			Description: "Gemini 2.0 Flash (Latest) - Always latest 2.0 Flash version",
		},
		{
			Name:        "gemini-2.0-flash-lite",
			Provider:    ProviderTypeGemini,
			ContextSize: 32000,
			MaxTokens:   8192,
			Description: "Gemini 2.0 Flash Lite - Fast image understanding with smaller context",
		},
		// Gemini 1.5 family
		{
			Name:        "gemini-1.5-pro",
			Provider:    ProviderTypeGemini,
			ContextSize: 2097152, // 2M tokens
			MaxTokens:   8192,
			Description: "Gemini 1.5 Pro - Powerful with 2M context window",
		},
		{
			Name:        "gemini-1.5-flash",
			Provider:    ProviderTypeGemini,
			ContextSize: 1048576,
			MaxTokens:   8192,
			Description: "Gemini 1.5 Flash - Fast and efficient",
		},
		{
			Name:        "gemini-1.5-flash-8b",
			Provider:    ProviderTypeGemini,
			ContextSize: 1048576,
			MaxTokens:   8192,
			Description: "Gemini 1.5 Flash 8B - Smaller, faster variant",
		},
		// Gemini 1.0 family
		{
			Name:        "gemini-1.0-pro",
			Provider:    ProviderTypeGemini,
			ContextSize: 32000,
			MaxTokens:   8192,
			Description: "Gemini 1.0 Pro - Reliable text-only model",
		},
		// Embedding models
		{
			Name:        "gemini-embedding-001",
			Provider:    ProviderTypeGemini,
			ContextSize: 2048,
			MaxTokens:   0,
			Description: "Gemini Embedding - Text embeddings for semantic search",
		},
	}

	for i := range models {
		EnrichModelInfo(&models[i])
	}

	return models
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

	// Round-50 LLMResponse.Err wiring (CONST-035 / Article XI §11.9):
	// Gemini signals partial-error conditions in the candidate's
	// finishReason rather than via HTTP status, so callers that only
	// checked the returned `error` previously dropped these signals on
	// the floor. Map well-known finishReason values to the canonical
	// sentinels. Bonus: if promptFeedback.blockReason is set the
	// prompt itself was blocked by safety filters → ContentBlocked.
	response.Err = mapGeminiFinishReasonToErr(candidate.FinishReason)
	if response.Err == nil && resp.PromptFeedback != nil && resp.PromptFeedback.BlockReason != "" {
		response.Err = ErrResponseContentBlocked
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

	// Speed programme P1-T05: track prompt-cache prefix stability. Gemini 2.5
	// performs implicit context caching for a byte-stable system-instruction
	// + tool-declaration prefix. This is purely observational — it does NOT
	// alter req — so the Gemini wire format is byte-identical to pre-P1-T05.
	trackPromptCachePrefixGeneric(gp.prefixDetector, "gemini", systemMsg, request.Tools)

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
	return gp.parseStreamingResponse(ctx, httpResp.Body, ch, requestID)
}

// parseStreamingResponse parses SSE streaming response. ctx is honored via a
// select-based guard on every channel send: without it, a caller that
// cancels ctx and stops draining ch (an HTTP client disconnect mid-stream)
// leaves this goroutine — and the open response body it holds — blocked/
// leaked forever on an unguarded send. Mirrors the pattern already used by
// ollama_provider.go / openai_compatible_provider.go in this package.
func (gp *GeminiProvider) parseStreamingResponse(ctx context.Context, body io.Reader, ch chan<- LLMResponse, requestID uuid.UUID) error {
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
					select {
					case ch <- LLMResponse{
						ID:        uuid.New(),
						RequestID: requestID,
						Content:   text,
						CreatedAt: time.Now(),
					}:
					case <-ctx.Done():
						return ctx.Err()
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

			// Round-50 LLMResponse.Err wiring for the streaming path
			// (CONST-035 / Article XI §11.9): map MAX_TOKENS / SAFETY /
			// RECITATION into the canonical sentinels so stream
			// consumers (notably tool_provider.go :201/:251) can
			// distinguish a clean stop from a partial-error stop.
			finalResponse.Err = mapGeminiFinishReasonToErr(candidate.FinishReason)

			select {
			case ch <- finalResponse:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return nil
}

// mapGeminiFinishReasonToErr returns the round-50 sentinel that matches
// a Gemini candidate.finishReason value, or nil if the reason indicates
// a clean stop ("STOP", "" — including the leniently-handled lowercase
// "stop" variant emitted by some preview SDKs). Documented values per
// https://ai.google.dev/api/generate-content#FinishReason :
//   - "STOP"                            : clean → nil
//   - "MAX_TOKENS"                      : token-cap   → ErrResponseTruncated
//   - "SAFETY"                          : safety       → ErrResponseContentBlocked
//   - "RECITATION"                      : recitation   → ErrResponseContentBlocked
//   - "BLOCKLIST"/"PROHIBITED_CONTENT"/"SPII" : policy → ErrResponseContentBlocked
//   - "OTHER"/"LANGUAGE"/"MALFORMED_FUNCTION_CALL"/...: unknown → nil
//     (we cannot synthesise the right sentinel from an unknown reason;
//     leave Err nil and rely on FinishReason for the literal value).
func mapGeminiFinishReasonToErr(reason string) error {
	switch reason {
	case "MAX_TOKENS":
		return ErrResponseTruncated
	case "SAFETY", "RECITATION", "BLOCKLIST", "PROHIBITED_CONTENT", "SPII":
		return ErrResponseContentBlocked
	default:
		return nil
	}
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

// GetContextWindow returns the model's context window size in tokens.
// Default: 200_000 (Gemini 1.5 Pro/Flash support 1M+; 200k is a safe conservative value).
func (gp *GeminiProvider) GetContextWindow() int {
	return 200_000
}

// CountTokens returns an estimated token count for text.
// Uses char-based fallback (1 token ≈ 3.5 chars) — Phase 3 will upgrade
// to the Gemini countTokens API endpoint.
func (gp *GeminiProvider) CountTokens(text string) (int, error) {
	return CharBasedTokenCount(text)
}
