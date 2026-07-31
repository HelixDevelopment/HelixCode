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

	"dev.helix.code/internal/llm/promptcache"
	"dev.helix.code/internal/providers/httpclient"
	"github.com/google/uuid"
)

// OpenAIProvider implements the Provider interface for OpenAI models
type OpenAIProvider struct {
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

	// catalogOnce/catalogMu guard the CONST-036 lazy live-catalog refresh.
	// Construction seeds a verified static list (no network at construction —
	// preserves the no-network-on-NewXProvider contract); the first GetModels()
	// call refreshes from the provider's live GET /models exactly once.
	catalogOnce sync.Once
	catalogMu   sync.RWMutex

	// prefixDetector watches the request prefix (system prompt + tool
	// definitions) for mid-session drift. Speed programme P1-T05: OpenAI
	// performs IMPLICIT prompt caching — there is no request flag, the
	// provider transparently reuses a cached prefix once it shares a
	// byte-identical prefix with a prior request. The detector freezes the
	// prefix on the first request and logs a "cache break" if a later
	// request mutates it. Observability only — it never alters the request.
	prefixDetector *promptcache.CacheBreakDetector
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(config ProviderConfigEntry) (*OpenAIProvider, error) {
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}

	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required (set config.APIKey or OPENAI_API_KEY env var)")
	}

	provider := &OpenAIProvider{
		config:   config,
		endpoint: endpoint,
		apiKey:   apiKey,
		// Shared tuned HTTP/2 transport (speed programme P1-T01,
		// R1 B03 / R3 §4.7) — connection pooling only; request
		// behaviour is unchanged.
		httpClient: httpclient.NewHTTPClient(60 * time.Second),
		lastHealth: &ProviderHealth{
			Status:    "unknown",
			LastCheck: time.Now(),
		},
		// Speed programme P1-T05: prompt-cache prefix-stability detector.
		prefixDetector: promptcache.NewCacheBreakDetector(),
	}

	// Initialize models
	provider.initializeModels()

	return provider, nil
}

// GetType returns the provider type
func (op *OpenAIProvider) GetType() ProviderType {
	return ProviderTypeOpenAI
}

// GetName returns the provider name
func (op *OpenAIProvider) GetName() string {
	return "OpenAI"
}

// GetModels returns available models. CONST-036 / F6-D-5: the list is refreshed
// LIVE from OpenAI's GET /models on first call (cached); the seed list set at
// construction is only the offline fallback.
func (op *OpenAIProvider) GetModels() []ModelInfo {
	op.refreshCatalogOnce()
	op.catalogMu.RLock()
	defer op.catalogMu.RUnlock()
	return op.models
}

// refreshCatalogOnce performs the live /models fetch exactly once. On failure it
// leaves the verified seed in place (honest fallback, never a fabricated list).
func (op *OpenAIProvider) refreshCatalogOnce() {
	op.catalogOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		models, err := fetchOpenAICompatibleCatalog(ctx, op.endpoint, op.apiKey, op.httpClient, ProviderTypeOpenAI, 128000, 4096)
		if err != nil {
			log.Printf("⚠️  OpenAI /models fetch failed (%v); keeping verified seed list", err)
			return
		}
		for i := range models {
			EnrichModelInfo(&models[i])
		}
		op.catalogMu.Lock()
		op.models = models
		op.catalogMu.Unlock()
		log.Printf("✅ OpenAI catalog refreshed with %d models (live /models)", len(models))
	})
}

// GetCapabilities returns provider capabilities
func (op *OpenAIProvider) GetCapabilities() []ModelCapability {
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

// Generate generates a response using OpenAI models
func (op *OpenAIProvider) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	startTime := time.Now()

	// Convert to OpenAI-compatible format
	openaiRequest, err := op.convertToOpenAIRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %v", err)
	}

	// Make request to OpenAI API
	response, err := op.makeOpenAIRequest(ctx, openaiRequest)
	if err != nil {
		return nil, fmt.Errorf("OpenAI request failed: %v", err)
	}

	// Convert response
	llmResponse := op.convertFromOpenAIResponse(response, request.ID, time.Since(startTime))

	return llmResponse, nil
}

// GenerateStream generates a streaming response
func (op *OpenAIProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
	defer close(ch)

	// Convert to OpenAI-compatible format
	openaiRequest, err := op.convertToOpenAIRequest(request)
	if err != nil {
		return fmt.Errorf("failed to convert request: %v", err)
	}

	// Enable streaming
	openaiRequest.Stream = true

	// Make streaming request
	return op.makeOpenAIStreamRequest(ctx, openaiRequest, ch, request.ID)
}

// IsAvailable checks if the provider is available
func (op *OpenAIProvider) IsAvailable(ctx context.Context) bool {
	health, err := op.GetHealth(ctx)
	return err == nil && health.Status == "healthy"
}

// GetHealth probes the OpenAI API and returns the resulting health record.
//
// The returned record is a COPY the caller owns outright — mutating it cannot
// reach provider state (HXC-214).
func (op *OpenAIProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	// Check if we can reach the OpenAI API
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/models", op.endpoint), nil)
	if err != nil {
		return op.recordHealth("unhealthy", 0, errCountIncrement, modelCountUnchanged),
			fmt.Errorf("failed to create health check request: %v", err)
	}

	op.setAuthHeaders(req)

	start := time.Now()
	resp, err := op.httpClient.Do(req)
	latency := time.Since(start)

	if err != nil {
		return op.recordHealth("unhealthy", latency, errCountIncrement, modelCountUnchanged),
			fmt.Errorf("health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return op.recordHealth("unhealthy", latency, errCountIncrement, modelCountUnchanged),
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
		return op.recordHealth("degraded", latency, errCountKeep, modelCountUnchanged), nil
	}

	return op.recordHealth("healthy", latency, errCountReset, len(modelsResponse.Data)), nil
}

// Close closes the provider
func (op *OpenAIProvider) Close() error {
	op.httpClient.CloseIdleConnections()
	return nil
}

// Helper methods

func (op *OpenAIProvider) initializeModels() {
	// CONST-036 / F6-D-5: this sets the verified SEED only (no network at
	// construction). The authoritative list is refreshed LIVE from GET /models
	// on the first GetModels() call (see refreshCatalogOnce).
	op.models = []ModelInfo{
		{
			Name:        "gpt-4o",
			Provider:    ProviderTypeOpenAI,
			ContextSize: 128000,
			MaxTokens:   4096,
			Description: "OpenAI's most advanced multimodal model",
		},
		{
			Name:        "gpt-4-turbo",
			Provider:    ProviderTypeOpenAI,
			ContextSize: 128000,
			MaxTokens:   4096,
			Description: "OpenAI's advanced multimodal model",
		},
		{
			Name:        "gpt-4",
			Provider:    ProviderTypeOpenAI,
			ContextSize: 8192,
			MaxTokens:   4096,
			Description: "OpenAI's powerful text model",
		},
		{
			Name:        "gpt-3.5-turbo",
			Provider:    ProviderTypeOpenAI,
			ContextSize: 16385,
			MaxTokens:   4096,
			Description: "OpenAI's fast and efficient model",
		},
	}

	for i := range op.models {
		EnrichModelInfo(&op.models[i])
	}

	log.Printf("✅ OpenAI provider initialized with %d models", len(op.models))
}

func (op *OpenAIProvider) convertToOpenAIRequest(request *LLMRequest) (*OpenAIRequest, error) {
	// Convert messages to OpenAI format
	var messages []OpenAIMessage
	var systemMsg string
	for _, msg := range request.Messages {
		if msg.Role == "system" {
			systemMsg = msg.Content
		}
		openaiMsg := OpenAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
		if msg.Name != "" {
			openaiMsg.Name = msg.Name
		}
		messages = append(messages, openaiMsg)
	}

	// Speed programme P1-T05: track prompt-cache prefix stability. OpenAI
	// performs implicit prompt caching — a stable system-prompt + tool
	// prefix across a session is the precondition for a cache hit. This is
	// purely observational; it does NOT alter the request body, so the
	// OpenAI wire format is byte-identical to the pre-P1-T05 behaviour.
	trackPromptCachePrefixGeneric(op.prefixDetector, "openai", systemMsg, request.Tools)

	return &OpenAIRequest{
		Model:       request.Model,
		Messages:    messages,
		MaxTokens:   request.MaxTokens,
		Temperature: request.Temperature,
		TopP:        request.TopP,
		Stream:      request.Stream,
	}, nil
}

func (op *OpenAIProvider) convertFromOpenAIResponse(openaiResp *OpenAIResponse, requestID uuid.UUID, processingTime time.Duration) *LLMResponse {
	var content string
	var finishReason string

	if len(openaiResp.Choices) > 0 {
		choice := openaiResp.Choices[0]
		content = choice.Message.Content
		finishReason = choice.FinishReason
	}

	resp := &LLMResponse{
		ID:        uuid.New(),
		RequestID: requestID,
		Content:   content,
		Usage: Usage{
			PromptTokens:     openaiResp.Usage.PromptTokens,
			CompletionTokens: openaiResp.Usage.CompletionTokens,
			TotalTokens:      openaiResp.Usage.TotalTokens,
		},
		FinishReason:   finishReason,
		ProcessingTime: processingTime,
		CreatedAt:      time.Now(),
	}

	// Round-46 LLMResponse.Err wiring (CONST-035 / Article XI §11.9):
	// OpenAI signals partial-error conditions in the choice's finish_reason
	// rather than the HTTP status code, so callers that only checked the
	// returned `error` previously dropped these signals on the floor.
	// Map well-known finish_reason values to the canonical sentinels.
	resp.Err = mapOpenAIFinishReasonToErr(finishReason)

	// Speed programme P1-T05: surface implicit-prompt-cache accounting.
	// OpenAI reports a cache hit in usage.prompt_tokens_details.cached_tokens.
	// On a cache MISS cacheMetadata() returns nil and ProviderMetadata is
	// left unset — so the response stays byte-identical to pre-P1-T05.
	if meta := openaiResp.Usage.cacheMetadata(); meta != nil {
		resp.ProviderMetadata = meta
	}
	return resp
}

// mapOpenAIFinishReasonToErr returns the round-46 sentinel that matches
// an OpenAI finish_reason, or nil if the reason indicates a clean stop
// ("stop", "tool_calls", "function_call", empty). Documented values:
// https://platform.openai.com/docs/api-reference/chat/object#chat/object-choices
func mapOpenAIFinishReasonToErr(reason string) error {
	switch reason {
	case "length":
		return ErrResponseTruncated
	case "content_filter":
		return ErrResponseContentBlocked
	default:
		return nil
	}
}

// MapOpenAIFinishReasonToErr is the exported façade over
// mapOpenAIFinishReasonToErr. It exists so that provider sub-packages
// (e.g. internal/llm/providers/cerebras — speed programme P5-T02) that
// thin-wrap the OpenAI-compatible wire format can reuse the canonical
// finish_reason → LLMResponse.Err mapping without re-implementing it
// (which would risk the closed-set vocabulary drifting per-provider).
// Behaviour is byte-identical to the unexported helper — pure delegation.
func MapOpenAIFinishReasonToErr(reason string) error {
	return mapOpenAIFinishReasonToErr(reason)
}

func (op *OpenAIProvider) makeOpenAIRequest(ctx context.Context, request *OpenAIRequest) (*OpenAIResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/chat/completions", op.endpoint), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	op.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := op.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (op *OpenAIProvider) makeOpenAIStreamRequest(ctx context.Context, request *OpenAIRequest, ch chan<- LLMResponse, requestID uuid.UUID) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/chat/completions", op.endpoint), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	op.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := op.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(body))
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

			// Round-46 LLMResponse.Err wiring for the streaming path
			// (CONST-035 / Article XI §11.9): when the final frame
			// carries a finish_reason of "length" or "content_filter",
			// emit a terminal LLMResponse with Err populated so
			// downstream stream consumers (notably tool_provider.go
			// :201/:251) can distinguish a clean stop from a
			// partial-error stop. The chunk is sent with FinishReason
			// preserved so callers that care about the literal reason
			// can still see it.
			if choice.FinishReason != "" {
				if errSentinel := mapOpenAIFinishReasonToErr(choice.FinishReason); errSentinel != nil {
					select {
					case ch <- LLMResponse{
						ID:           uuid.New(),
						RequestID:    requestID,
						FinishReason: choice.FinishReason,
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

	return nil
}

func (op *OpenAIProvider) setAuthHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+op.apiKey)
}

// recordHealth applies a health verdict to lastHealth under healthMu and
// returns a copy the caller owns outright (HXC-214). The whole verdict lands in
// ONE critical section, so a concurrent reader cannot observe it half-applied.
func (op *OpenAIProvider) recordHealth(
	status string,
	latency time.Duration,
	adj errCountAdjust,
	modelCount int,
) *ProviderHealth {
	op.healthMu.Lock()
	defer op.healthMu.Unlock()
	return applyProviderHealth(op.lastHealth, status, latency, adj, modelCount)
}

// GetContextWindow returns the model's context window size in tokens.
// Default: 200_000 — individual deployments SHOULD configure this to match
// the actual model (e.g. gpt-4o supports 128k, gpt-4-turbo supports 128k).
func (op *OpenAIProvider) GetContextWindow() int {
	return 200_000
}

// CountTokens returns an estimated token count for text.
// Uses char-based fallback (1 token ≈ 3.5 chars) — Phase 3 will upgrade
// to tiktoken for accurate OpenAI tokenization.
func (op *OpenAIProvider) CountTokens(text string) (int, error) {
	return CharBasedTokenCount(text)
}

// OpenAI API types

type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
	// Tool-conversation protocol fields, shared by every OpenAI-compatible
	// provider that reuses this type (DeepSeek, Mistral, OpenRouter). omitempty
	// ⇒ plain-chat messages serialise byte-identically. ToolCallID is REQUIRED
	// on every role:"tool" result message; ToolCalls carries the assistant
	// turn's requested calls so each result has a matching call.
	//
	// SEND-side wire type: ToolCalls uses wireSendToolCall (NOT llm.ToolCall)
	// because OpenAI/Groq/DeepSeek/Mistral/OpenRouter REQUIRE
	// function.arguments to be a JSON-encoded STRING (`"arguments":"{}"`), not
	// a JSON object. Marshalling llm.ToolCall directly emits the object form
	// and the provider rejects it: 'messages.N.tool_calls.0.function.arguments'
	// : value must be a string. Convert via toWireSendToolCalls at the
	// assignment site.
	ToolCallID string             `json:"tool_call_id,omitempty"`
	ToolCalls  []wireSendToolCall `json:"tool_calls,omitempty"`
}

// wireSendToolCall is the SEND-side on-wire shape of a single assistant
// tool_calls[] entry for OpenAI Chat Completions–compatible providers (OpenAI,
// Groq, DeepSeek, Mistral, OpenRouter, and the OpenAI-compatible local fan-out).
// Its `function.arguments` is a STRING (a serialized JSON object) — the
// canonical OpenAI encoding the providers require. It is the symmetric inverse
// of openAIWireToolCall / parseOpenAIWireToolCalls (which decode the same shape
// on the PARSE side).
type wireSendToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// toWireSendToolCalls converts []llm.ToolCall into []wireSendToolCall, encoding
// each call's Function.Arguments map as a JSON STRING. Rules:
//   - Type defaults to "function" when empty.
//   - A nil/empty arguments map (or a map that fails to marshal) is encoded as
//     the literal "{}" — never an empty string, never "null".
//
// Returns nil for an empty/absent input so plain-chat messages (no tool_calls)
// serialise byte-identically with omitempty.
func toWireSendToolCalls(cs []ToolCall) []wireSendToolCall {
	if len(cs) == 0 {
		return nil
	}
	out := make([]wireSendToolCall, 0, len(cs))
	for _, c := range cs {
		args := "{}"
		if len(c.Function.Arguments) > 0 {
			if raw, err := json.Marshal(c.Function.Arguments); err == nil && len(raw) > 0 {
				args = string(raw)
			}
		}
		callType := c.Type
		if callType == "" {
			callType = "function"
		}
		w := wireSendToolCall{ID: c.ID, Type: callType}
		w.Function.Name = c.Function.Name
		w.Function.Arguments = args
		out = append(out, w)
	}
	return out
}

type OpenAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
		// openAICacheUsageFields (speed programme P1-T05) carries the
		// implicit-prompt-cache accounting fields emitted by OpenAI
		// (`prompt_tokens_details.cached_tokens`) and DeepSeek
		// (`prompt_cache_hit_tokens` / `prompt_cache_miss_tokens`).
		// These are RESPONSE-only fields — embedding them here does not
		// change any request body. A provider that omits them leaves
		// them zero.
		openAICacheUsageFields
	} `json:"usage"`
}

type OpenAIStreamResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}
