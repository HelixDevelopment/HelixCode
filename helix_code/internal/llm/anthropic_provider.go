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
	"sync"
	"time"

	"github.com/google/uuid"

	"dev.helix.code/internal/llm/promptcache"
)

// AnthropicProvider implements the Provider interface for Anthropic's Claude models
type AnthropicProvider struct {
	config     ProviderConfigEntry
	apiKey     string
	endpoint   string
	httpClient *http.Client
	models     []ModelInfo
	lastHealth *ProviderHealth

	// catalogOnce/catalogMu guard the CONST-036 lazy live-catalog refresh from
	// Anthropic's GET /v1/models (seed set at construction, no network there).
	catalogOnce sync.Once
	catalogMu   sync.RWMutex

	// cacheAwareness tracks the wall-clock time of the most recent successful
	// completion so callers can predict whether Anthropic's prompt-cache TTL
	// (~5 minutes) has elapsed before sending the next request.
	// Ported from gptme commit e896ed4ff.
	cacheAwareness *CacheAwareness

	// prefixDetector watches the request prefix (system prompt + tool
	// definitions) for mid-session drift. Speed programme P1-T04: provider-side
	// prompt caching only hits when the prefix is byte-stable across a session.
	// The detector freezes the prefix on the first request and reports a
	// "cache break" if a later request mutates it. This is observability only —
	// it never alters the request — so feature behaviour is unaffected.
	prefixDetector *promptcache.CacheBreakDetector
}

// Anthropic API structures
type anthropicRequest struct {
	Model         string                   `json:"model"`
	MaxTokens     int                      `json:"max_tokens"`
	Messages      []anthropicMessage       `json:"messages"`
	System        interface{}              `json:"system,omitempty"` // string or []anthropicSystemBlock
	Temperature   float64                  `json:"temperature,omitempty"`
	TopP          float64                  `json:"top_p,omitempty"`
	Stream        bool                     `json:"stream,omitempty"`
	Tools         []anthropicTool          `json:"tools,omitempty"`
	ToolChoice    interface{}              `json:"tool_choice,omitempty"`
	Thinking      *anthropicThinkingConfig `json:"thinking,omitempty"`
	StopSequences []string                 `json:"stop_sequences,omitempty"`
	Metadata      map[string]interface{}   `json:"metadata,omitempty"`
}

type anthropicMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or []anthropicContentBlock
}

// Prompt caching support - based on OpenCode and Codename Goose implementations
type anthropicContentBlock struct {
	Type         string                 `json:"type"` // "text", "image", "tool_use", "tool_result"
	Text         string                 `json:"text,omitempty"`
	Source       *anthropicImageSource  `json:"source,omitempty"`
	CacheControl *anthropicCacheControl `json:"cache_control,omitempty"` // For caching specific blocks
	ToolUseID    string                 `json:"tool_use_id,omitempty"`
	ID           string                 `json:"id,omitempty"`
	Name         string                 `json:"name,omitempty"`
	Input        map[string]interface{} `json:"input,omitempty"`
	Content      interface{}            `json:"content,omitempty"` // For tool results
	IsError      bool                   `json:"is_error,omitempty"`
}

type anthropicSystemBlock struct {
	Type         string                 `json:"type"` // "text"
	Text         string                 `json:"text"`
	CacheControl *anthropicCacheControl `json:"cache_control,omitempty"`
}

type anthropicCacheControl struct {
	Type string `json:"type"` // "ephemeral"
}

type anthropicImageSource struct {
	Type      string `json:"type"`       // "base64", "url"
	MediaType string `json:"media_type"` // "image/jpeg", "image/png", etc.
	Data      string `json:"data,omitempty"`
	URL       string `json:"url,omitempty"`
}

type anthropicTool struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	InputSchema  map[string]interface{} `json:"input_schema"`
	CacheControl *anthropicCacheControl `json:"cache_control,omitempty"` // For caching tools
}

// Extended thinking configuration - automatic based on prompt content
type anthropicThinkingConfig struct {
	Type   string `json:"type"`             // "enabled"
	Budget int    `json:"budget,omitempty"` // Token budget for thinking
}

type anthropicResponse struct {
	ID           string                  `json:"id"`
	Type         string                  `json:"type"` // "message"
	Role         string                  `json:"role"`
	Content      []anthropicContentBlock `json:"content"`
	Model        string                  `json:"model"`
	StopReason   string                  `json:"stop_reason"`
	StopSequence string                  `json:"stop_sequence,omitempty"`
	Usage        anthropicUsage          `json:"usage"`
}

type anthropicUsage struct {
	InputTokens         int `json:"input_tokens"`
	OutputTokens        int `json:"output_tokens"`
	CacheCreationTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// Streaming event types
type anthropicStreamEvent struct {
	Type         string                 `json:"type"`
	Message      *anthropicResponse     `json:"message,omitempty"`
	Index        int                    `json:"index,omitempty"`
	ContentBlock *anthropicContentBlock `json:"content_block,omitempty"`
	Delta        *anthropicDelta        `json:"delta,omitempty"`
	Usage        *anthropicUsage        `json:"usage,omitempty"`
}

type anthropicDelta struct {
	Type         string `json:"type"`
	Text         string `json:"text,omitempty"`
	PartialJSON  string `json:"partial_json,omitempty"`
	StopReason   string `json:"stop_reason,omitempty"`
	StopSequence string `json:"stop_sequence,omitempty"`
}

type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(config ProviderConfigEntry) (*AnthropicProvider, error) {
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	if apiKey == "" {
		return nil, fmt.Errorf("anthropic API key not provided")
	}

	// Endpoint precedence (P1-F12-T03):
	//   1. Explicit Config.Endpoint (highest — user wired a specific URL).
	//   2. ANTHROPIC_BASE_URL env var (mid — runtime override without
	//      rewriting config; mirrors official Anthropic SDKs).
	//   3. Canonical Anthropic API endpoint (default).
	// An empty ANTHROPIC_BASE_URL is treated as "unset" — sending requests
	// to "" would fail in non-obvious ways at the HTTP layer.
	endpoint := config.Endpoint
	if endpoint == "" {
		if envBase := os.Getenv("ANTHROPIC_BASE_URL"); envBase != "" {
			endpoint = envBase
		} else {
			endpoint = "https://api.anthropic.com/v1/messages"
		}
	}

	provider := &AnthropicProvider{
		config:   config,
		apiKey:   apiKey,
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		models:         getAnthropicModels(),
		cacheAwareness: NewCacheAwareness(),
		prefixDetector: promptcache.NewCacheBreakDetector(),
	}

	return provider, nil
}

// anthropicModelsBase derives the `/v1/models` catalog URL from the configured
// messages endpoint (which ends in `/v1/messages`). Falls back to the canonical
// host when the endpoint shape is unexpected.
func (ap *AnthropicProvider) anthropicModelsBase() string {
	ep := ap.endpoint
	if i := strings.LastIndex(ep, "/v1/"); i >= 0 {
		return ep[:i] + "/v1/models"
	}
	return "https://api.anthropic.com/v1/models"
}

// fetchModelCatalog calls Anthropic's GET /v1/models and returns the live model
// list (CONST-036). Returns nil (and no error surface) on any failure so the
// caller falls back to the verified seed — model metadata is sourced live,
// never invented.
func (ap *AnthropicProvider) fetchModelCatalog(ctx context.Context) []ModelInfo {
	req, err := http.NewRequestWithContext(ctx, "GET", ap.anthropicModelsBase(), nil)
	if err != nil {
		return nil
	}
	req.Header.Set("x-api-key", ap.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Accept", "application/json")

	resp, err := ap.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var cat struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&cat); err != nil {
		return nil
	}
	if len(cat.Data) == 0 {
		return nil
	}

	models := make([]ModelInfo, 0, len(cat.Data))
	for _, m := range cat.Data {
		if m.ID == "" {
			continue
		}
		desc := m.DisplayName
		if desc == "" {
			desc = m.ID
		}
		mi := ModelInfo{
			Name:        m.ID,
			Provider:    ProviderTypeAnthropic,
			ContextSize: 200000,
			MaxTokens:   8192,
			Description: desc,
		}
		EnrichModelInfo(&mi)
		models = append(models, mi)
	}
	if len(models) == 0 {
		return nil
	}
	return models
}

// CacheAwareness returns the provider's cache-coldness tracker. Callers may
// consult IsCacheLikelyCold() to decide whether to skip explicit
// cache_control markers on the next request (the entry is likely expired
// anyway, so attaching the marker would burn a cache-creation token cycle).
//
// Ported from gptme commit e896ed4ff.
func (ap *AnthropicProvider) CacheAwareness() *CacheAwareness {
	return ap.cacheAwareness
}

// getAnthropicModels returns all available Claude models with correct specifications
// Based on OpenCode's comprehensive model list
func getAnthropicModels() []ModelInfo {

	models := []ModelInfo{
		// Claude 4 family (latest)
		{
			Name:        "claude-4-sonnet",
			Provider:    ProviderTypeAnthropic,
			ContextSize: 200000,
			MaxTokens:   50000,
			Description: "Claude 4 Sonnet - Latest flagship model with extended thinking",
		},
		{
			Name:        "claude-4-opus",
			Provider:    ProviderTypeAnthropic,
			ContextSize: 200000,
			MaxTokens:   50000,
			Description: "Claude 4 Opus - Most powerful Claude model",
		},
		// Claude 3.7 family
		{
			Name:        "claude-3-7-sonnet-20250219",
			Provider:    ProviderTypeAnthropic,
			ContextSize: 200000,
			MaxTokens:   50000,
			Description: "Claude 3.7 Sonnet - Enhanced reasoning and analysis",
		},
		// Claude 3.5 family
		{
			Name:        "claude-3-5-sonnet-20241022",
			Provider:    ProviderTypeAnthropic,
			ContextSize: 200000,
			MaxTokens:   8192,
			Description: "Claude 3.5 Sonnet - Excellent for coding tasks",
		},
		{
			Name:        "claude-3-5-sonnet-latest",
			Provider:    ProviderTypeAnthropic,
			ContextSize: 200000,
			MaxTokens:   8192,
			Description: "Claude 3.5 Sonnet (Latest) - Always latest 3.5 Sonnet version",
		},
		{
			Name:        "claude-3-5-haiku-20241022",
			Provider:    ProviderTypeAnthropic,
			ContextSize: 200000,
			MaxTokens:   8192,
			Description: "Claude 3.5 Haiku - Fast and efficient",
		},
		{
			Name:        "claude-3-5-haiku-latest",
			Provider:    ProviderTypeAnthropic,
			ContextSize: 200000,
			MaxTokens:   8192,
			Description: "Claude 3.5 Haiku (Latest) - Always latest 3.5 Haiku version",
		},
		// Claude 3 family
		{
			Name:        "claude-3-opus-20240229",
			Provider:    ProviderTypeAnthropic,
			ContextSize: 200000,
			MaxTokens:   4096,
			Description: "Claude 3 Opus - Most powerful Claude 3 model",
		},
		{
			Name:        "claude-3-opus-latest",
			Provider:    ProviderTypeAnthropic,
			ContextSize: 200000,
			MaxTokens:   4096,
			Description: "Claude 3 Opus (Latest) - Always latest Opus version",
		},
		{
			Name:        "claude-3-sonnet-20240229",
			Provider:    ProviderTypeAnthropic,
			ContextSize: 200000,
			MaxTokens:   4096,
			Description: "Claude 3 Sonnet - Balanced performance",
		},
		{
			Name:        "claude-3-haiku-20240307",
			Provider:    ProviderTypeAnthropic,
			ContextSize: 200000,
			MaxTokens:   4096,
			Description: "Claude 3 Haiku - Fast and cost-effective",
		},
	}

	for i := range models {
		EnrichModelInfo(&models[i])
	}

	return models
}

// GetType returns the provider type
func (ap *AnthropicProvider) GetType() ProviderType {
	return ProviderTypeAnthropic
}

// GetName returns the provider name
func (ap *AnthropicProvider) GetName() string {
	return "Anthropic"
}

// GetModels returns available models. CONST-036 / F6-D-5: refreshed LIVE from
// Anthropic's GET /v1/models on first call (cached); getAnthropicModels() seed
// is the offline fallback.
func (ap *AnthropicProvider) GetModels() []ModelInfo {
	ap.refreshCatalogOnce()
	ap.catalogMu.RLock()
	defer ap.catalogMu.RUnlock()
	return ap.models
}

func (ap *AnthropicProvider) refreshCatalogOnce() {
	ap.catalogOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		live := ap.fetchModelCatalog(ctx)
		if len(live) == 0 {
			log.Printf("⚠️  Anthropic /v1/models fetch failed; keeping verified seed list")
			return
		}
		ap.catalogMu.Lock()
		ap.models = live
		ap.catalogMu.Unlock()
		log.Printf("✅ Anthropic catalog refreshed with %d models (live /v1/models)", len(live))
	})
}

// GetCapabilities returns provider capabilities
func (ap *AnthropicProvider) GetCapabilities() []ModelCapability {
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

// Generate generates a response using Claude
func (ap *AnthropicProvider) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	startTime := time.Now()

	// Build Anthropic request with all advanced features
	anthropicReq, err := ap.buildRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %v", err)
	}

	// Make API request
	resp, err := ap.makeRequest(ctx, anthropicReq)
	if err != nil {
		return nil, err
	}

	// Parse response
	response := &LLMResponse{
		ID:             uuid.New(),
		RequestID:      request.ID,
		ProcessingTime: time.Since(startTime),
		CreatedAt:      time.Now(),
		Usage: Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
		FinishReason: resp.StopReason,
		// Round-46 LLMResponse.Err wiring (CONST-035 / Article XI §11.9):
		// Anthropic surfaces partial-error conditions via stop_reason
		// ("max_tokens" → truncation, "refusal"/safety → content block).
		// Without this mapping the round-33 anchored limitation in
		// tool_provider.go:201/:251 (no way to distinguish OK-empty
		// from mid-stream failure) remained even after the HTTP layer
		// returned nil error.
		Err: mapAnthropicStopReasonToErr(resp.StopReason),
	}

	// Extract content and tool calls
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			response.Content += block.Text
		case "tool_use":
			response.ToolCalls = append(response.ToolCalls, ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: ToolCallFunc{
					Name:      block.Name,
					Arguments: block.Input,
				},
			})
		}
	}

	// Add caching info to metadata
	if resp.Usage.CacheCreationTokens > 0 || resp.Usage.CacheReadTokens > 0 {
		response.ProviderMetadata = map[string]interface{}{
			"cache_creation_tokens": resp.Usage.CacheCreationTokens,
			"cache_read_tokens":     resp.Usage.CacheReadTokens,
		}
	}

	// Record completion time so subsequent callers can consult
	// CacheAwareness.IsCacheLikelyCold() before sending the next request.
	// This mirrors the GENERATION_POST hook from gptme commit e896ed4ff.
	if ap.cacheAwareness != nil {
		ap.cacheAwareness.RecordCompletion(time.Now())
	}

	return response, nil
}

// mapAnthropicStopReasonToErr returns the round-46 sentinel matching an
// Anthropic stop_reason, or nil for clean stops ("end_turn", "stop_sequence",
// "tool_use", empty). Documented values:
// https://docs.anthropic.com/en/api/messages#response-stop-reason
func mapAnthropicStopReasonToErr(reason string) error {
	switch reason {
	case "max_tokens":
		return ErrResponseTruncated
	case "refusal", "safety":
		return ErrResponseContentBlocked
	default:
		return nil
	}
}

// GenerateStream generates a streaming response
func (ap *AnthropicProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
	defer close(ch)

	// Build request with streaming enabled
	anthropicReq, err := ap.buildRequest(request)
	if err != nil {
		return fmt.Errorf("failed to build request: %v", err)
	}
	anthropicReq.Stream = true

	// Make streaming request
	return ap.makeStreamingRequest(ctx, anthropicReq, ch, request.ID)
}

// buildRequest constructs an Anthropic API request with all advanced features
// Includes: prompt caching, extended thinking, tool caching
func (ap *AnthropicProvider) buildRequest(request *LLMRequest) (*anthropicRequest, error) {
	req := &anthropicRequest{
		Model:       request.Model,
		MaxTokens:   request.MaxTokens,
		Temperature: request.Temperature,
		TopP:        request.TopP,
		Stream:      request.Stream,
	}

	// Default max tokens if not specified
	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}

	// Convert messages
	systemMsg, messages := ap.convertMessages(request.Messages)
	req.Messages = messages

	// Apply prompt caching based on config
	cacheConfig := request.CacheConfig
	if cacheConfig == nil {
		defaultCache := DefaultCacheConfig()
		cacheConfig = &defaultCache
	}

	// Apply caching to system message if enabled
	if systemMsg != "" && cacheConfig.Enabled {
		req.System = []anthropicSystemBlock{
			{
				Type:         "text",
				Text:         systemMsg,
				CacheControl: &anthropicCacheControl{Type: "ephemeral"},
			},
		}
	} else if systemMsg != "" {
		req.System = systemMsg
	}

	// Apply prompt caching to messages based on strategy
	if cacheConfig.Enabled && len(messages) > 0 {
		switch cacheConfig.Strategy {
		case CacheStrategyContext, CacheStrategyAggressive:
			// Cache last message for context preservation
			lastMsg := &messages[len(messages)-1]
			if content, ok := lastMsg.Content.([]anthropicContentBlock); ok {
				if len(content) > 0 {
					content[len(content)-1].CacheControl = &anthropicCacheControl{Type: "ephemeral"}
					lastMsg.Content = content
				}
			}
		}
	}

	// Convert tools with caching
	if len(request.Tools) > 0 {
		req.Tools = ap.convertTools(request.Tools)

		// Apply caching to tools based on strategy
		if cacheConfig.Enabled && (cacheConfig.Strategy == CacheStrategyTools ||
			cacheConfig.Strategy == CacheStrategyContext ||
			cacheConfig.Strategy == CacheStrategyAggressive) {
			if len(req.Tools) > 0 {
				req.Tools[len(req.Tools)-1].CacheControl = &anthropicCacheControl{Type: "ephemeral"}
			}
		}
	}

	// Apply reasoning configuration
	reasoningConfig := request.Reasoning
	if reasoningConfig == nil {
		// Auto-detect if this is a reasoning model
		isReasoning, modelType := IsReasoningModel(request.Model)
		if isReasoning {
			reasoningConfig = NewReasoningConfig(modelType)
		} else if ap.shouldEnableThinking(request) {
			// Use generic reasoning for keyword-based detection
			reasoningConfig = NewReasoningConfig(ReasoningModelClaude_Sonnet)
		}
	}

	// Configure extended thinking if reasoning is enabled
	if reasoningConfig != nil && reasoningConfig.Enabled {
		// Use thinking budget from config or default to 80% of max tokens
		thinkingBudget := reasoningConfig.ThinkingBudget
		if thinkingBudget == 0 {
			thinkingBudget = int(float64(req.MaxTokens) * 0.8)
		}
		// Use request-level thinking budget if specified
		if request.ThinkingBudget > 0 {
			thinkingBudget = request.ThinkingBudget
		}

		req.Thinking = &anthropicThinkingConfig{
			Type:   "enabled",
			Budget: thinkingBudget,
		}

		// Adjust temperature for thinking mode if not explicitly set
		if req.Temperature == 0 {
			req.Temperature = 1.0
		}
	}

	// Speed programme P1-T04: track prompt-cache prefix stability.
	// The prefix (system prompt + tool definitions) must be byte-stable for
	// every request in a session or the provider cache never hits. The first
	// request freezes the prefix as the baseline; subsequent requests are
	// checked against it and a "cache break" is logged if the prefix drifted.
	// This is purely observational — it never alters req — so it cannot affect
	// feature behaviour for any provider or any caller.
	ap.trackPromptCachePrefix(systemMsg, req.Tools)

	return req, nil
}

// trackPromptCachePrefix records or verifies the prompt-cache prefix for the
// current session. On the first call it freezes the prefix as the baseline;
// on later calls it checks the prefix against the baseline and logs a warning
// if it changed (cache break). Errors are swallowed: prefix tracking is an
// optimization-observability concern and MUST NEVER fail a request.
func (ap *AnthropicProvider) trackPromptCachePrefix(systemMsg string, tools []anthropicTool) {
	if ap.prefixDetector == nil {
		return
	}
	prefixTools := make([]interface{}, len(tools))
	for i := range tools {
		prefixTools[i] = tools[i]
	}
	prefix := promptcache.PrefixComponents{
		SystemPrompt: systemMsg,
		Tools:        prefixTools,
	}
	if !ap.prefixDetector.IsFrozen() {
		if _, err := ap.prefixDetector.Freeze(prefix); err != nil {
			log.Printf("anthropic: prompt-cache prefix freeze failed: %v", err)
		}
		return
	}
	res, err := ap.prefixDetector.Check(prefix)
	if err != nil {
		log.Printf("anthropic: prompt-cache prefix check failed: %v", err)
		return
	}
	if res.Broken {
		log.Printf("anthropic: %s", res.Reason)
	}
}

// shouldEnableThinking determines if extended thinking should be enabled
// Based on Codename Goose pattern: enable if prompt contains thinking-related keywords
func (ap *AnthropicProvider) shouldEnableThinking(request *LLMRequest) bool {
	// Check if any message contains thinking-related keywords
	thinkingKeywords := []string{"think", "reason", "analyze", "consider", "explain why", "step by step"}

	for _, msg := range request.Messages {
		msgLower := strings.ToLower(msg.Content)
		for _, keyword := range thinkingKeywords {
			if strings.Contains(msgLower, keyword) {
				return true
			}
		}
	}

	return false
}

// convertMessages converts LLM messages to Anthropic format
func (ap *AnthropicProvider) convertMessages(messages []Message) (string, []anthropicMessage) {
	var systemMsg string
	var anthropicMsgs []anthropicMessage

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			systemMsg = msg.Content
		case "user", "assistant":
			anthropicMsgs = append(anthropicMsgs, anthropicMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	return systemMsg, anthropicMsgs
}

// convertTools converts LLM tools to Anthropic format.
//
// Speed programme P1-T04: the InputSchema is a map[string]interface{}. Go
// randomizes map iteration order, and JSON-Schema bodies frequently carry
// set-like arrays (`required`, `enum`) that callers assemble by ranging over a
// Go map — producing a randomly-ordered slice. encoding/json sorts top-level
// map keys but faithfully preserves that array randomness, so a naive prefix
// would mishash and NEVER hit the provider prompt cache.
//
// canonicalizeSchema runs each InputSchema through promptcache's deterministic
// serializer (sorted keys + sorted string-only arrays) so the serialized tool
// definition is byte-stable across every request in a session. This is a
// semantics-preserving normalization: the JSON content is logically identical
// (Anthropic does not care about key/required order) — only the byte order is
// frozen, which is exactly what the prompt cache needs.
func (ap *AnthropicProvider) convertTools(tools []Tool) []anthropicTool {
	anthropicTools := make([]anthropicTool, len(tools))

	for i, tool := range tools {
		anthropicTools[i] = anthropicTool{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			InputSchema: canonicalizeSchema(tool.Function.Parameters),
		}
	}

	return anthropicTools
}

// canonicalizeSchema returns a copy of schema whose nested set-like string
// arrays are sorted, so re-serialization is byte-deterministic. If schema is
// nil or canonicalization fails for any reason, the original map is returned
// unchanged — this normalization is a cache optimization and MUST NEVER break
// a request. (encoding/json already sorts map keys on marshal, so an
// unmodified return still produces correct, though potentially cache-missing,
// output.)
func canonicalizeSchema(schema map[string]interface{}) map[string]interface{} {
	if schema == nil {
		return nil
	}
	canon, err := promptcache.CanonicalJSONSorted(schema)
	if err != nil {
		return schema
	}
	var normalized map[string]interface{}
	if err := json.Unmarshal(canon, &normalized); err != nil {
		return schema
	}
	return normalized
}

// makeRequest makes a non-streaming API request
func (ap *AnthropicProvider) makeRequest(ctx context.Context, request *anthropicRequest) (*anthropicResponse, error) {
	// Marshal request
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", ap.endpoint, strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers - exact format from Anthropic API docs
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", ap.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Make request
	httpResp, err := ap.httpClient.Do(httpReq)
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
		var apiErr anthropicError
		if err := json.Unmarshal(respBody, &apiErr); err == nil {
			return nil, fmt.Errorf("anthropic API error (%d): %s - %s", httpResp.StatusCode, apiErr.Type, apiErr.Message)
		}
		return nil, fmt.Errorf("anthropic API error (%d): %s", httpResp.StatusCode, string(respBody))
	}

	// Parse successful response
	var response anthropicResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &response, nil
}

// makeStreamingRequest makes a streaming API request
func (ap *AnthropicProvider) makeStreamingRequest(ctx context.Context, request *anthropicRequest, ch chan<- LLMResponse, requestID uuid.UUID) error {
	// Marshal request
	reqBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", ap.endpoint, strings.NewReader(string(reqBody)))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers for streaming
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", ap.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Accept", "text/event-stream")

	// Make request
	httpResp, err := ap.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("anthropic API error (%d): %s", httpResp.StatusCode, string(body))
	}

	// Parse streaming response
	return ap.parseStreamingResponse(ctx, httpResp.Body, ch, requestID)
}

// parseStreamingResponse parses SSE streaming response. ctx is honored via a
// select-based guard on every channel send: without it, a caller that
// cancels ctx and stops draining ch (exactly what happens when an HTTP
// client disconnects mid-generation) leaves this goroutine — and the open
// httpResp.Body it holds — blocked/leaked forever on an unguarded send.
// Mirrors the pattern already used by ollama_provider.go /
// openai_compatible_provider.go in this package.
func (ap *AnthropicProvider) parseStreamingResponse(ctx context.Context, body io.Reader, ch chan<- LLMResponse, requestID uuid.UUID) error {
	decoder := json.NewDecoder(body)
	var currentContent strings.Builder
	var currentToolCalls []ToolCall
	// Round-46 LLMResponse.Err wiring: Anthropic SSE delivers the final
	// stop_reason via a `message_delta` event ahead of `message_stop`;
	// remember the most-recent one so the terminal frame can populate Err.
	var lastStopReason string

	for {
		var event anthropicStreamEvent
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			// Continue on decode errors (SSE comments, etc.)
			continue
		}

		switch event.Type {
		case "content_block_delta":
			if event.Delta != nil && event.Delta.Text != "" {
				currentContent.WriteString(event.Delta.Text)

				// Send incremental response
				select {
				case ch <- LLMResponse{
					ID:        uuid.New(),
					RequestID: requestID,
					Content:   event.Delta.Text,
					CreatedAt: time.Now(),
				}:
				case <-ctx.Done():
					return ctx.Err()
				}
			}

		case "content_block_start":
			if event.ContentBlock != nil && event.ContentBlock.Type == "tool_use" {
				currentToolCalls = append(currentToolCalls, ToolCall{
					ID:   event.ContentBlock.ID,
					Type: "function",
					Function: ToolCallFunc{
						Name:      event.ContentBlock.Name,
						Arguments: make(map[string]interface{}),
					},
				})
			}

		case "message_delta":
			// Round-46 LLMResponse.Err wiring (CONST-035 / Article XI
			// §11.9): Anthropic SSE delivers the final stop_reason via
			// a message_delta event BEFORE the message_stop event, so
			// capture it here and let it flow into the message_stop
			// terminal frame's Err field.
			if event.Delta != nil && event.Delta.StopReason != "" {
				// Defer to message_stop emission below; record on a
				// pseudo-current-event so the inspector sees it.
				lastStopReason = event.Delta.StopReason
			}

		case "message_stop":
			// Send final response with complete content
			finalResponse := LLMResponse{
				ID:           uuid.New(),
				RequestID:    requestID,
				Content:      currentContent.String(),
				ToolCalls:    currentToolCalls,
				CreatedAt:    time.Now(),
				FinishReason: lastStopReason,
				Err:          mapAnthropicStopReasonToErr(lastStopReason),
			}

			if event.Usage != nil {
				finalResponse.Usage = Usage{
					PromptTokens:     event.Usage.InputTokens,
					CompletionTokens: event.Usage.OutputTokens,
					TotalTokens:      event.Usage.InputTokens + event.Usage.OutputTokens,
				}
			}

			select {
			case ch <- finalResponse:
			case <-ctx.Done():
				return ctx.Err()
			}

			// Record completion time on stream end so the next caller can
			// consult CacheAwareness.IsCacheLikelyCold() before sending the
			// next request. Ported from gptme commit e896ed4ff
			// (GENERATION_POST hook).
			if ap.cacheAwareness != nil {
				ap.cacheAwareness.RecordCompletion(time.Now())
			}
		}
	}

	return nil
}

// IsAvailable checks if the provider is available
func (ap *AnthropicProvider) IsAvailable(ctx context.Context) bool {
	return ap.apiKey != ""
}

// GetHealth returns the health status of the provider
func (ap *AnthropicProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	startTime := time.Now()

	// Simple health check: try to list models or make a minimal request
	health := &ProviderHealth{
		LastCheck:  time.Now(),
		ModelCount: len(ap.models),
	}

	// Test with a minimal request
	testReq := &LLMRequest{
		ID:          uuid.New(),
		Model:       "claude-3-5-haiku-latest",
		Messages:    []Message{{Role: "user", Content: "Hi"}},
		MaxTokens:   10,
		Temperature: 0.1,
	}

	_, err := ap.Generate(ctx, testReq)
	if err != nil {
		health.Status = "unhealthy"
		health.ErrorCount = 1
		return health, err
	}

	health.Status = "healthy"
	health.Latency = time.Since(startTime)
	ap.lastHealth = health

	return health, nil
}

// Close closes the provider and cleans up resources
func (ap *AnthropicProvider) Close() error {
	ap.httpClient.CloseIdleConnections()
	log.Printf("Anthropic provider closed")
	return nil
}

// GetContextWindow returns the context window size in tokens for the configured model.
// All current Claude models (3.x, 4.x) support 200k tokens; claude-2 variants support 100k.
func (ap *AnthropicProvider) GetContextWindow() int {
	// Use the first configured model name as the active model.
	model := ""
	if len(ap.config.Models) > 0 {
		model = ap.config.Models[0]
	}
	switch {
	case strings.Contains(model, "claude-2"):
		return 100_000
	default:
		// claude-3, claude-3.5, claude-3.7, claude-4 — all 200k
		return 200_000
	}
}

// CountTokens returns an estimated token count for text.
// Uses char-based fallback (1 token ≈ 3.5 chars) — providers SHOULD
// override with their native tokenizer (Phase 3 sub-spec).
func (ap *AnthropicProvider) CountTokens(text string) (int, error) {
	return CharBasedTokenCount(text)
}
