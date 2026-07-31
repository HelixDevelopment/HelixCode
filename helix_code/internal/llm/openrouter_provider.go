package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
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

	// healthMu guards lastHealth (HXC-214). GetHealth is NAMED as a query but
	// genuinely mutates the record, so callers invoke it from concurrent paths
	// — health monitors, status endpoints, IsAvailable — assuming it is safe.
	// Every read and write of lastHealth goes through recordHealth, and no
	// network call is ever made with this lock held.
	healthMu sync.Mutex
}

// NewOpenRouterProvider creates a new OpenRouter provider
func NewOpenRouterProvider(config ProviderConfigEntry) (*OpenRouterProvider, error) {
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "https://openrouter.ai/api/v1"
	}

	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("OpenRouter API key is required (set config.APIKey or OPENROUTER_API_KEY env var)")
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
		return orp.recordHealth("unhealthy", 0, errCountIncrement, modelCountUnchanged),
			fmt.Errorf("failed to create health check request: %v", err)
	}

	orp.setAuthHeaders(req)

	start := time.Now()
	resp, err := orp.httpClient.Do(req)
	latency := time.Since(start)

	if err != nil {
		return orp.recordHealth("unhealthy", latency, errCountIncrement, modelCountUnchanged),
			fmt.Errorf("health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return orp.recordHealth("unhealthy", latency, errCountIncrement, modelCountUnchanged),
			fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	// Parse response to get model count
	var modelsResponse struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&modelsResponse); err != nil {
		// Still consider it available: the endpoint answered, only the body was
		// unreadable, so this is not a new connectivity failure.
		return orp.recordHealth("degraded", latency, errCountKeep, modelCountUnchanged), nil
	}

	return orp.recordHealth("healthy", latency, errCountReset, len(modelsResponse.Data)), nil
}

// Close closes the provider
func (orp *OpenRouterProvider) Close() error {
	orp.httpClient.CloseIdleConnections()
	return nil
}

// GetContextWindow returns the model's context window size in tokens.
// Default: 200_000 — OpenRouter routes to many models; 200k is a safe upper bound.
func (orp *OpenRouterProvider) GetContextWindow() int {
	return 200_000
}

// CountTokens returns an estimated token count for text.
// Uses char-based fallback (1 token ≈ 3.5 chars) — Phase 3 will upgrade
// to the OpenRouter tokenize endpoint.
func (orp *OpenRouterProvider) CountTokens(text string) (int, error) {
	return CharBasedTokenCount(text)
}

// Helper methods

// openRouterCatalogResponse is the shape of GET /models from OpenRouter.
type openRouterCatalogResponse struct {
	Data []struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		ContextLength int    `json:"context_length"`
		TopProvider   struct {
			MaxCompletionTokens int `json:"max_completion_tokens"`
		} `json:"top_provider"`
	} `json:"data"`
}

// preferredOpenRouterDefaults lists free-tier model IDs that probe
// PASS (HTTP 200, real completion) most reliably from our session
// 2026-05-14 anti-bluff sweep. Order = preference for default model
// when the CLI auto-selects the first entry. These are seeded ahead
// of the rest of the live catalog so HELIX_LLM_PROVIDER=openrouter
// works out of the box without the user having to discover a
// currently-uncongested free model by trial and error.
//
// CONST-036 note: this *bias* is hand-curated runtime evidence,
// not a hardcoded SOURCE of model metadata — the catalog itself
// is fetched live below. The fallback list further down is only
// used when /models is unreachable (no network / API key revoked).
var preferredOpenRouterDefaults = []string{
	"openai/gpt-oss-20b:free",
	"deepseek/deepseek-v4-flash:free",
	"z-ai/glm-4.5-air:free",
	"meta-llama/llama-3.3-70b-instruct:free",
	"meta-llama/llama-3.2-3b-instruct:free",
	"qwen/qwen3-coder:free",
}

// fetchCatalog calls OpenRouter's GET /models and returns the list
// of available models, or an error if the call fails.
func (orp *OpenRouterProvider) fetchCatalog(ctx context.Context) ([]ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", orp.endpoint+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("build /models request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+orp.apiKey)
	req.Header.Set("Accept", "application/json")
	resp, err := orp.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET /models: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GET /models returned %d: %s", resp.StatusCode, string(body))
	}
	var cat openRouterCatalogResponse
	if err := json.NewDecoder(resp.Body).Decode(&cat); err != nil {
		return nil, fmt.Errorf("decode /models response: %w", err)
	}
	if len(cat.Data) == 0 {
		return nil, fmt.Errorf("/models returned empty data array")
	}
	// Index by ID for re-ordering.
	byID := make(map[string]ModelInfo, len(cat.Data))
	all := make([]ModelInfo, 0, len(cat.Data))
	for _, m := range cat.Data {
		maxTok := m.TopProvider.MaxCompletionTokens
		if maxTok == 0 {
			maxTok = 4096
		}
		ctxLen := m.ContextLength
		if ctxLen == 0 {
			ctxLen = 32768
		}
		mi := ModelInfo{
			Name:        m.ID,
			Provider:    ProviderTypeOpenRouter,
			ContextSize: ctxLen,
			MaxTokens:   maxTok,
			Description: m.Name,
		}
		byID[m.ID] = mi
		all = append(all, mi)
	}
	// Re-order: preferred defaults first (if present in catalog), then the rest.
	ordered := make([]ModelInfo, 0, len(all))
	seen := make(map[string]bool, len(all))
	for _, pref := range preferredOpenRouterDefaults {
		if mi, ok := byID[pref]; ok {
			ordered = append(ordered, mi)
			seen[pref] = true
		}
	}
	for _, mi := range all {
		if !seen[mi.Name] {
			ordered = append(ordered, mi)
		}
	}
	return ordered, nil
}

func (orp *OpenRouterProvider) initializeModels() {
	// CONST-036 fix (round 41+): fetch the model catalog live from
	// OpenRouter at provider init so HelixCode reflects whatever
	// the provider actually serves today, instead of carrying a
	// stale hardcoded list. CONST-035 anti-bluff: the prior list
	// led with `deepseek-r1-free`, which OpenRouter long since
	// retired (returned HTTP 400 "is not a valid model ID"),
	// meaning every fresh user typing `HELIX_LLM_PROVIDER=openrouter`
	// got an immediate broken-default failure. With live fetch +
	// preferred-default re-ordering, that failure mode is gone.
	//
	// Fallback (when /models is unreachable) is a *small* curated
	// list of currently-verified free IDs — not a stale catalog
	// mined from training data. We accept that this fallback can
	// also drift; the next agent updating this file should re-probe.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if models, err := orp.fetchCatalog(ctx); err == nil {
		orp.models = models
		for i := range orp.models {
			EnrichModelInfo(&orp.models[i])
		}
		log.Printf("✅ OpenRouter provider initialized with %d models (live /models catalog)", len(orp.models))
		return
	} else {
		log.Printf("⚠️  OpenRouter /models fetch failed (%v); using verified seed list", err)
	}
	orp.models = []ModelInfo{
		{
			Name:        "openai/gpt-oss-20b:free",
			Provider:    ProviderTypeOpenRouter,
			ContextSize: 131072,
			MaxTokens:   4096,
			Description: "GPT-OSS 20B Free (round-41-verified working)",
		},
		{
			Name:        "deepseek/deepseek-v4-flash:free",
			Provider:    ProviderTypeOpenRouter,
			ContextSize: 131072,
			MaxTokens:   4096,
			Description: "DeepSeek v4 Flash Free (round-41-verified working)",
		},
		{
			Name:        "z-ai/glm-4.5-air:free",
			Provider:    ProviderTypeOpenRouter,
			ContextSize: 131072,
			MaxTokens:   4096,
			Description: "GLM 4.5 Air Free (round-41-verified working)",
		},
		{
			Name:        "meta-llama/llama-3.3-70b-instruct:free",
			Provider:    ProviderTypeOpenRouter,
			ContextSize: 131072,
			MaxTokens:   4096,
			Description: "Llama 3.3 70B Instruct Free (rate-limited but valid ID)",
		},
		{
			Name:        "meta-llama/llama-3.2-3b-instruct:free",
			Provider:    ProviderTypeOpenRouter,
			ContextSize: 131072,
			MaxTokens:   4096,
			Description: "Llama 3.2 3B Instruct Free (rate-limited but valid ID)",
		},
	}

	for i := range orp.models {
		EnrichModelInfo(&orp.models[i])
	}

	log.Printf("✅ OpenRouter provider initialized with %d models (fallback seed list)", len(orp.models))
}

func (orp *OpenRouterProvider) convertToOpenAIRequest(request *LLMRequest) (*openRouterChatRequest, error) {
	// Convert messages to OpenAI format
	var messages []OpenAIMessage
	for _, msg := range request.Messages {
		openaiMsg := OpenAIMessage{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
			ToolCalls:  toWireSendToolCalls(msg.ToolCalls),
		}
		if msg.Name != "" {
			openaiMsg.Name = msg.Name
		}
		messages = append(messages, openaiMsg)
	}

	return &openRouterChatRequest{
		Model:       request.Model,
		Messages:    messages,
		MaxTokens:   request.MaxTokens,
		Temperature: request.Temperature,
		TopP:        request.TopP,
		Stream:      request.Stream,
		// OpenAI-compatible function-calling: forward the agent's tool
		// definitions + tool_choice. omitempty ⇒ plain-chat wire unchanged.
		Tools:      request.Tools,
		ToolChoice: request.ToolChoice,
	}, nil
}

func (orp *OpenRouterProvider) convertFromOpenAIResponse(openaiResp *openRouterChatResponse, requestID uuid.UUID, processingTime time.Duration) *LLMResponse {
	var content string
	var finish string
	var toolCalls []ToolCall

	if len(openaiResp.Choices) > 0 {
		choice := openaiResp.Choices[0]
		content = choice.Message.Content
		finish = choice.FinishReason
		toolCalls = parseOpenAIWireToolCalls(choice.Message.ToolCalls)
	}

	return &LLMResponse{
		ID:        uuid.New(),
		RequestID: requestID,
		Content:   content,
		ToolCalls: toolCalls,
		Usage: Usage{
			PromptTokens:     openaiResp.Usage.PromptTokens,
			CompletionTokens: openaiResp.Usage.CompletionTokens,
			TotalTokens:      openaiResp.Usage.TotalTokens,
		},
		FinishReason:   finish,
		ProcessingTime: processingTime,
		CreatedAt:      time.Now(),
		// Round-53 LLMResponse.Err wiring (CONST-035 / Article XI §11.9):
		// OpenRouter is a proxy that fans out to many backend providers
		// (OpenAI, Anthropic, Cohere, etc.) and normalises the response
		// to OpenAI-compatible finish_reason values per
		// https://openrouter.ai/docs/responses. Reuse the round-46
		// OpenAI mapper helper (same closed mapping: "length",
		// "content_filter"). If OpenRouter diverges from OpenAI in the
		// future, TestRound53_OpenRouter_ReusesOpenAIMapper will fail
		// and a dedicated mapOpenRouterFinishReasonToErr helper MUST
		// be introduced in the same commit. Bonus fix: previously
		// openaiResp.Choices[0].FinishReason was dereferenced without
		// the len > 0 guard — now guarded.
		Err: mapOpenAIFinishReasonToErr(finish),
	}
}

func (orp *OpenRouterProvider) makeOpenAIRequest(ctx context.Context, request *openRouterChatRequest) (*openRouterChatResponse, error) {
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

	var response openRouterChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (orp *OpenRouterProvider) makeOpenAIStreamRequest(ctx context.Context, request *openRouterChatRequest, ch chan<- LLMResponse, requestID uuid.UUID) error {
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

		if len(streamResp.Choices) > 0 && streamResp.Choices[0].FinishReason != "" {
			// Round-53 LLMResponse.Err wiring for the streaming path
			// (CONST-035 / Article XI §11.9): when the final frame
			// carries finish_reason="length"/"content_filter", emit a
			// terminal LLMResponse with Err populated so stream
			// consumers (notably tool_provider.go :201/:251) can
			// distinguish a clean stop from a partial-error stop.
			finishReason := streamResp.Choices[0].FinishReason
			if errSentinel := mapOpenAIFinishReasonToErr(finishReason); errSentinel != nil {
				select {
				case ch <- LLMResponse{
					ID:           uuid.New(),
					RequestID:    requestID,
					FinishReason: finishReason,
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

	return nil
}

func (orp *OpenRouterProvider) setAuthHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+orp.apiKey)
}

// recordHealth applies a health verdict to lastHealth under healthMu and
// returns a copy the caller owns outright (HXC-214). The whole verdict lands in
// ONE critical section, so a concurrent reader cannot observe it half-applied.
func (orp *OpenRouterProvider) recordHealth(
	status string,
	latency time.Duration,
	adj errCountAdjust,
	modelCount int,
) *ProviderHealth {
	orp.healthMu.Lock()
	defer orp.healthMu.Unlock()
	return applyProviderHealth(orp.lastHealth, status, latency, adj, modelCount)
}

// Note: OpenAI API types are reused for OpenRouter compatibility
// They are declared in openai_provider.go and used here since they're in the same package

// openRouterChatRequest mirrors the shared OpenAIRequest but adds the
// OpenAI-compatible function-calling fields. OpenRouter normalises every
// backend to the OpenAI Chat Completions wire. omitempty keeps plain-chat
// requests byte-identical.
type openRouterChatRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Tools       []Tool          `json:"tools,omitempty"`
	ToolChoice  interface{}     `json:"tool_choice,omitempty"`
}

// openRouterChatResponse mirrors the shared OpenAIResponse but adds
// message.tool_calls parsing.
type openRouterChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role      string               `json:"role"`
			Content   string               `json:"content"`
			ToolCalls []openAIWireToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}
