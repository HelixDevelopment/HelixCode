package llm

import (
	"bufio"
	"bytes"
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

	"dev.helix.code/internal/llm/promptcache"
	"dev.helix.code/internal/providers/httpclient"
	"github.com/google/uuid"
)

// DeepSeekProvider implements the Provider interface for the DeepSeek
// Cloud (api.deepseek.com) OpenAI-compatible endpoint. Round-41 F12
// fast-path expansion: a user with DEEPSEEK_API_KEY can run
// `HELIX_LLM_PROVIDER=deepseek ./bin/cli` without first editing
// config.yaml or starting the HelixCode server.
type DeepSeekProvider struct {
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
	catalogOnce sync.Once
	catalogMu   sync.RWMutex

	// prefixDetector watches the request prefix for mid-session drift.
	// Speed programme P1-T05: DeepSeek performs IMPLICIT context caching
	// (cache on disk) — there is no request flag, the provider transparently
	// reuses a cached prefix and reports the hit/miss split via
	// usage.prompt_cache_hit_tokens / prompt_cache_miss_tokens. The detector
	// freezes the prefix on the first request and logs a "cache break" if a
	// later request mutates it. Observability only — never alters a request.
	prefixDetector *promptcache.CacheBreakDetector
}

// NewDeepSeekProvider creates a new DeepSeek provider.
func NewDeepSeekProvider(config ProviderConfigEntry) (*DeepSeekProvider, error) {
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "https://api.deepseek.com/v1"
	}

	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("DEEPSEEK_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("DeepSeek API key is required (set config.APIKey or DEEPSEEK_API_KEY env var)")
	}

	p := &DeepSeekProvider{
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

	p.initializeModels()

	return p, nil
}

func (dp *DeepSeekProvider) GetType() ProviderType { return ProviderTypeDeepSeek }
func (dp *DeepSeekProvider) GetName() string       { return "DeepSeek" }

// GetModels returns available models. CONST-036 / F6-D-5: refreshed LIVE from
// DeepSeek's GET /models on first call (cached); seed list is offline fallback.
func (dp *DeepSeekProvider) GetModels() []ModelInfo {
	dp.refreshCatalogOnce()
	dp.catalogMu.RLock()
	defer dp.catalogMu.RUnlock()
	return dp.models
}

func (dp *DeepSeekProvider) refreshCatalogOnce() {
	dp.catalogOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		models, err := fetchOpenAICompatibleCatalog(ctx, dp.endpoint, dp.apiKey, dp.httpClient, ProviderTypeDeepSeek, 128000, 8192)
		if err != nil {
			log.Printf("⚠️  DeepSeek /models fetch failed (%v); keeping verified seed list", err)
			return
		}
		for i := range models {
			EnrichModelInfo(&models[i])
		}
		dp.catalogMu.Lock()
		dp.models = models
		dp.catalogMu.Unlock()
		log.Printf("✅ DeepSeek catalog refreshed with %d models (live /models)", len(models))
	})
}

func (dp *DeepSeekProvider) GetCapabilities() []ModelCapability {
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

func (dp *DeepSeekProvider) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	startTime := time.Now()

	openaiRequest, err := dp.convertToOpenAIRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %v", err)
	}

	response, err := dp.makeOpenAIRequest(ctx, openaiRequest)
	if err != nil {
		return nil, fmt.Errorf("DeepSeek request failed: %v", err)
	}

	return dp.convertFromOpenAIResponse(response, request.ID, time.Since(startTime)), nil
}

func (dp *DeepSeekProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
	defer close(ch)

	openaiRequest, err := dp.convertToOpenAIRequest(request)
	if err != nil {
		return fmt.Errorf("failed to convert request: %v", err)
	}
	openaiRequest.Stream = true

	return dp.makeOpenAIStreamRequest(ctx, openaiRequest, ch, request.ID)
}

func (dp *DeepSeekProvider) IsAvailable(ctx context.Context) bool {
	health, err := dp.GetHealth(ctx)
	return err == nil && health.Status == "healthy"
}

// GetHealth probes the DeepSeek API and returns the resulting health record.
//
// The returned record is a COPY the caller owns outright — mutating it cannot
// reach provider state (HXC-214).
func (dp *DeepSeekProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/models", dp.endpoint), nil)
	if err != nil {
		return dp.recordHealth("unhealthy", 0, errCountIncrement, modelCountUnchanged),
			fmt.Errorf("failed to create health check request: %v", err)
	}
	dp.setAuthHeaders(req)

	start := time.Now()
	resp, err := dp.httpClient.Do(req)
	latency := time.Since(start)
	if err != nil {
		return dp.recordHealth("unhealthy", latency, errCountIncrement, modelCountUnchanged),
			fmt.Errorf("health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return dp.recordHealth("unhealthy", latency, errCountIncrement, modelCountUnchanged),
			fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	var modelsResponse struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&modelsResponse); err != nil {
		// Still available: the endpoint answered, only the body was unreadable,
		// so this is not a new connectivity failure.
		return dp.recordHealth("degraded", latency, errCountKeep, modelCountUnchanged), nil
	}

	return dp.recordHealth("healthy", latency, errCountReset, len(modelsResponse.Data)), nil
}

func (dp *DeepSeekProvider) Close() error {
	dp.httpClient.CloseIdleConnections()
	return nil
}

// GetContextWindow returns DeepSeek's published max context window.
// DeepSeek-V3 / V4 / Reasoner all advertise 128k.
func (dp *DeepSeekProvider) GetContextWindow() int { return 128_000 }

func (dp *DeepSeekProvider) CountTokens(text string) (int, error) {
	return CharBasedTokenCount(text)
}

func (dp *DeepSeekProvider) initializeModels() {
	// CONST-036 / F6-D-5: verified SEED only (no network at construction). The
	// authoritative list is refreshed LIVE from GET /models on the first
	// GetModels() call (see refreshCatalogOnce).
	dp.models = []ModelInfo{
		{
			Name:        "deepseek-chat",
			Provider:    ProviderTypeDeepSeek,
			ContextSize: 128000,
			MaxTokens:   8192,
			Description: "DeepSeek Chat - flagship general-purpose model",
		},
		{
			Name:        "deepseek-reasoner",
			Provider:    ProviderTypeDeepSeek,
			ContextSize: 128000,
			MaxTokens:   8192,
			Description: "DeepSeek Reasoner - reasoning-focused model",
		},
		{
			Name:        "deepseek-coder",
			Provider:    ProviderTypeDeepSeek,
			ContextSize: 128000,
			MaxTokens:   8192,
			Description: "DeepSeek Coder - code-specialised model",
		},
	}

	for i := range dp.models {
		EnrichModelInfo(&dp.models[i])
	}

	log.Printf("✅ DeepSeek provider initialized with %d models", len(dp.models))
}

func (dp *DeepSeekProvider) convertToOpenAIRequest(request *LLMRequest) (*deepseekChatRequest, error) {
	var messages []OpenAIMessage
	var systemMsg string
	for _, msg := range request.Messages {
		if msg.Role == "system" {
			systemMsg = msg.Content
		}
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

	// Speed programme P1-T05: track prompt-cache prefix stability. DeepSeek
	// performs implicit context caching — a stable prefix across a session
	// is the precondition for a cache hit. Observational only; the DeepSeek
	// request body is byte-identical to the pre-P1-T05 behaviour.
	trackPromptCachePrefixGeneric(dp.prefixDetector, "deepseek", systemMsg, request.Tools)

	return &deepseekChatRequest{
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

func (dp *DeepSeekProvider) convertFromOpenAIResponse(openaiResp *deepseekChatResponse, requestID uuid.UUID, processingTime time.Duration) *LLMResponse {
	var content string
	var toolCalls []ToolCall
	if len(openaiResp.Choices) > 0 {
		content = openaiResp.Choices[0].Message.Content
		toolCalls = parseOpenAIWireToolCalls(openaiResp.Choices[0].Message.ToolCalls)
	}

	finish := ""
	if len(openaiResp.Choices) > 0 {
		finish = openaiResp.Choices[0].FinishReason
	}

	resp := &LLMResponse{
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
		// Round-50 LLMResponse.Err wiring (CONST-035 / Article XI §11.9):
		// DeepSeek is OpenAI-compatible — finish_reason "length" /
		// "content_filter" indicate truncation / content block. Reuse
		// the round-46 OpenAI mapper helper (same closed mapping per
		// https://api-docs.deepseek.com/api/create-chat-completion).
		Err: mapOpenAIFinishReasonToErr(finish),
	}

	// Speed programme P1-T05: surface implicit-context-cache accounting.
	// DeepSeek reports the hit/miss split via usage.prompt_cache_hit_tokens
	// / prompt_cache_miss_tokens. On a full cache MISS cacheMetadata()
	// returns nil and ProviderMetadata is left unset — byte-identical to
	// the pre-P1-T05 response shape.
	if meta := openaiResp.Usage.cacheMetadata(); meta != nil {
		resp.ProviderMetadata = meta
	}
	return resp
}

func (dp *DeepSeekProvider) makeOpenAIRequest(ctx context.Context, request *deepseekChatRequest) (*deepseekChatResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/chat/completions", dp.endpoint), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	dp.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := dp.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("DeepSeek API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response deepseekChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (dp *DeepSeekProvider) makeOpenAIStreamRequest(ctx context.Context, request *deepseekChatRequest, ch chan<- LLMResponse, requestID uuid.UUID) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/chat/completions", dp.endpoint), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	dp.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := dp.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("DeepSeek API returned status %d: %s", resp.StatusCode, string(body))
	}

	// DeepSeek (OpenAI-compatible) streams Server-Sent Events: each event is a
	// `data: {json}` line, events separated by blank lines, terminated by a
	// `data: [DONE]` sentinel. A raw json.Decoder over the body chokes on the
	// `data:` prefix ("invalid character 'd' looking for beginning of value"),
	// so parse line-by-line, strip the SSE field prefix, skip keep-alives and
	// the [DONE] sentinel, and json.Unmarshal each event payload.
	scanner := bufio.NewScanner(resp.Body)
	// SSE event payloads can exceed bufio's default 64 KiB line cap (large
	// tool-call argument deltas); raise the buffer ceiling to 1 MiB.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			// Non-data SSE fields (event:, id:, retry:, comments) are ignored.
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			if payload == "[DONE]" {
				break
			}
			continue
		}

		var streamResp OpenAIStreamResponse
		if err := json.Unmarshal([]byte(payload), &streamResp); err != nil {
			return fmt.Errorf("decode stream event: %w", err)
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
			// Round-50 LLMResponse.Err wiring for the streaming path
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
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read stream: %w", err)
	}

	return nil
}

func (dp *DeepSeekProvider) setAuthHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+dp.apiKey)
}

// recordHealth applies a health verdict to lastHealth under healthMu and
// returns a copy the caller owns outright (HXC-214). The whole verdict lands in
// ONE critical section, so a concurrent reader cannot observe it half-applied.
func (dp *DeepSeekProvider) recordHealth(
	status string,
	latency time.Duration,
	adj errCountAdjust,
	modelCount int,
) *ProviderHealth {
	dp.healthMu.Lock()
	defer dp.healthMu.Unlock()
	return applyProviderHealth(dp.lastHealth, status, latency, adj, modelCount)
}

// deepseekChatRequest mirrors the shared OpenAIRequest but adds the
// OpenAI-compatible function-calling fields (DeepSeek is OpenAI Chat
// Completions–compatible). omitempty keeps plain-chat requests byte-identical.
type deepseekChatRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Tools       []Tool          `json:"tools,omitempty"`
	ToolChoice  interface{}     `json:"tool_choice,omitempty"`
}

// deepseekChatResponse mirrors the shared OpenAIResponse but adds
// message.tool_calls parsing. It preserves the implicit-prompt-cache usage
// fields so convertFromOpenAIResponse's cacheMetadata() call still works.
type deepseekChatResponse struct {
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
		openAICacheUsageFields
	} `json:"usage"`
}
