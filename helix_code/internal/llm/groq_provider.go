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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// GroqProvider implements the Provider interface for Groq's ultra-fast LPU inference
type GroqProvider struct {
	config         ProviderConfigEntry
	apiKey         string
	baseURL        string
	httpClient     *http.Client
	models         []ModelInfo
	lastHealth     *ProviderHealth
	latencyMetrics *LatencyTracker
}

// LatencyTracker tracks performance metrics for Groq requests
type LatencyTracker struct {
	mutex           sync.RWMutex
	maxSamples      int
	firstTokenTimes []time.Duration
	totalTimes      []time.Duration
	tokensPerSecond []float64
}

// LatencyMetrics represents aggregated latency statistics
type LatencyMetrics struct {
	AvgFirstTokenLatency time.Duration `json:"avg_first_token_latency"`
	P50FirstTokenLatency time.Duration `json:"p50_first_token_latency"`
	P95FirstTokenLatency time.Duration `json:"p95_first_token_latency"`
	P99FirstTokenLatency time.Duration `json:"p99_first_token_latency"`
	AvgTotalLatency      time.Duration `json:"avg_total_latency"`
	AvgTokensPerSecond   float64       `json:"avg_tokens_per_second"`
	SampleCount          int           `json:"sample_count"`
}

// NewLatencyTracker creates a new latency tracker
func NewLatencyTracker(maxSamples int) *LatencyTracker {
	return &LatencyTracker{
		maxSamples:      maxSamples,
		firstTokenTimes: make([]time.Duration, 0, maxSamples),
		totalTimes:      make([]time.Duration, 0, maxSamples),
		tokensPerSecond: make([]float64, 0, maxSamples),
	}
}

// RecordRequest records metrics for a request
func (lt *LatencyTracker) RecordRequest(firstToken, total time.Duration, tps float64) {
	lt.mutex.Lock()
	defer lt.mutex.Unlock()

	// Add to samples
	lt.firstTokenTimes = append(lt.firstTokenTimes, firstToken)
	lt.totalTimes = append(lt.totalTimes, total)
	lt.tokensPerSecond = append(lt.tokensPerSecond, tps)

	// Trim if over max
	if len(lt.firstTokenTimes) > lt.maxSamples {
		lt.firstTokenTimes = lt.firstTokenTimes[1:]
		lt.totalTimes = lt.totalTimes[1:]
		lt.tokensPerSecond = lt.tokensPerSecond[1:]
	}
}

// GetMetrics returns aggregated metrics
func (lt *LatencyTracker) GetMetrics() LatencyMetrics {
	lt.mutex.RLock()
	defer lt.mutex.RUnlock()

	if len(lt.firstTokenTimes) == 0 {
		return LatencyMetrics{}
	}

	return LatencyMetrics{
		AvgFirstTokenLatency: average(lt.firstTokenTimes),
		P50FirstTokenLatency: percentile(lt.firstTokenTimes, 0.5),
		P95FirstTokenLatency: percentile(lt.firstTokenTimes, 0.95),
		P99FirstTokenLatency: percentile(lt.firstTokenTimes, 0.99),
		AvgTotalLatency:      average(lt.totalTimes),
		AvgTokensPerSecond:   averageFloat(lt.tokensPerSecond),
		SampleCount:          len(lt.firstTokenTimes),
	}
}

// Helper functions for statistics
func average(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	var sum time.Duration
	for _, d := range durations {
		sum += d
	}
	return sum / time.Duration(len(durations))
}

func percentile(durations []time.Duration, p float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	index := int(float64(len(sorted)-1) * p)
	return sorted[index]
}

func averageFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// NewGroqProvider creates a new Groq provider
func NewGroqProvider(config ProviderConfigEntry) (*GroqProvider, error) {
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("GROQ_API_KEY")
	}

	if apiKey == "" {
		return nil, fmt.Errorf("groq API key not provided")
	}

	baseURL := config.Endpoint
	if baseURL == "" {
		baseURL = "https://api.groq.com"
	}

	// Optimized HTTP client for low latency
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			ForceAttemptHTTP2:   true,
		},
	}

	provider := &GroqProvider{
		config:         config,
		apiKey:         apiKey,
		baseURL:        baseURL,
		httpClient:     httpClient,
		models:         getGroqModels(),
		latencyMetrics: NewLatencyTracker(100),
		lastHealth: &ProviderHealth{
			Status:    "unknown",
			LastCheck: time.Now(),
		},
	}

	log.Printf("✅ Groq provider initialized with %d models", len(provider.models))
	return provider, nil
}

// GetType returns the provider type
func (gp *GroqProvider) GetType() ProviderType {
	return ProviderTypeGroq
}

// GetName returns the provider name
func (gp *GroqProvider) GetName() string {
	return "Groq"
}

// GetModels returns available models
func (gp *GroqProvider) GetModels() []ModelInfo {
	return gp.models
}

// GetCapabilities returns provider capabilities
func (gp *GroqProvider) GetCapabilities() []ModelCapability {
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

// Generate generates a response using Groq models
func (gp *GroqProvider) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	startTime := time.Now()

	// Convert to Groq (OpenAI-compatible) format
	groqRequest, err := gp.buildGroqRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %v", err)
	}

	// Make request to Groq API
	response, err := gp.makeGroqRequest(ctx, groqRequest)
	if err != nil {
		return nil, fmt.Errorf("groq request failed: %v", err)
	}

	// Convert response
	totalTime := time.Since(startTime)
	llmResponse := gp.convertFromGroqResponse(response, request.ID, totalTime)

	// Record metrics
	if response.Usage.CompletionTokens > 0 && totalTime > 0 {
		tps := float64(response.Usage.CompletionTokens) / totalTime.Seconds()
		gp.latencyMetrics.RecordRequest(totalTime/2, totalTime, tps) // Approximate first token time
	}

	return llmResponse, nil
}

// GenerateStream generates a streaming response
func (gp *GroqProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
	defer close(ch)
	startTime := time.Now()

	// Convert to Groq format with streaming enabled
	groqRequest, err := gp.buildGroqRequest(request)
	if err != nil {
		return fmt.Errorf("failed to build request: %v", err)
	}
	groqRequest.Stream = true

	// Make streaming request
	return gp.makeGroqStreamRequest(ctx, groqRequest, ch, request.ID, startTime)
}

// IsAvailable checks if the provider is available
func (gp *GroqProvider) IsAvailable(ctx context.Context) bool {
	// Simple availability check - verify API key is set
	return gp.apiKey != ""
}

// GetHealth returns provider health status
func (gp *GroqProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	startTime := time.Now()

	health := &ProviderHealth{
		LastCheck:  time.Now(),
		ModelCount: len(gp.models),
	}

	// Test with fastest model
	testReq := &LLMRequest{
		ID:          uuid.New(),
		Model:       "llama-3.1-8b-instant",
		Messages:    []Message{{Role: "user", Content: "Hi"}},
		MaxTokens:   10,
		Temperature: 0.1,
	}

	_, err := gp.Generate(ctx, testReq)
	if err != nil {
		health.Status = "unhealthy"
		health.ErrorCount = 1
		health.Latency = time.Since(startTime)
		gp.lastHealth = health
		return health, err
	}

	health.Status = "healthy"
	health.Latency = time.Since(startTime)
	gp.lastHealth = health

	return health, nil
}

// GetLatencyMetrics returns current latency metrics
func (gp *GroqProvider) GetLatencyMetrics() *LatencyMetrics {
	metrics := gp.latencyMetrics.GetMetrics()
	return &metrics
}

// Close closes the provider
func (gp *GroqProvider) Close() error {
	gp.httpClient.CloseIdleConnections()
	return nil
}

// GetContextWindow returns the model's context window size in tokens.
// Default: 200_000 — Groq models (Llama, Mixtral, Gemma) range from 8k–128k;
// 200k is a safe upper bound pending a Phase 3 model-aware upgrade.
func (gp *GroqProvider) GetContextWindow() int {
	return 200_000
}

// CountTokens returns an estimated token count for text.
// Uses char-based fallback (1 token ≈ 3.5 chars) — Phase 3 will upgrade
// to the Groq tokenize endpoint.
func (gp *GroqProvider) CountTokens(text string) (int, error) {
	return CharBasedTokenCount(text)
}

// Helper methods

func getGroqModels() []ModelInfo {
	models := []ModelInfo{
		{
			Name:        "llama-3.3-70b-versatile",
			Provider:    ProviderTypeGroq,
			ContextSize: 131072,
			MaxTokens:   32768,
			Description: "Llama 3.3 70B - Most capable model on Groq with ultra-fast inference",
		},
		{
			Name:        "llama-3.1-70b-versatile",
			Provider:    ProviderTypeGroq,
			ContextSize: 131072,
			MaxTokens:   8192,
			Description: "Llama 3.1 70B - Previous generation with excellent speed",
		},
		{
			Name:        "llama-3.1-8b-instant",
			Provider:    ProviderTypeGroq,
			ContextSize: 131072,
			MaxTokens:   8192,
			Description: "Llama 3.1 8B - Extremely fast, ideal for high-volume use",
		},
		{
			Name:        "mixtral-8x7b-32768",
			Provider:    ProviderTypeGroq,
			ContextSize: 32768,
			MaxTokens:   32768,
			Description: "Mixtral 8x7B - Mixture of experts with fast inference",
		},
		{
			Name:        "gemma2-9b-it",
			Provider:    ProviderTypeGroq,
			ContextSize: 8192,
			MaxTokens:   8192,
			Description: "Gemma 2 9B - Google's efficient open model on Groq",
		},
		{
			Name:        "gemma-7b-it",
			Provider:    ProviderTypeGroq,
			ContextSize: 8192,
			MaxTokens:   8192,
			Description: "Gemma 7B - Compact and fast",
		},
	}

	for i := range models {
		EnrichModelInfo(&models[i])
	}

	return models
}

func (gp *GroqProvider) buildGroqRequest(request *LLMRequest) (*GroqRequest, error) {
	// Convert messages to Groq (OpenAI-compatible) format
	var messages []GroqMessage
	for _, msg := range request.Messages {
		groqMsg := GroqMessage{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
			ToolCalls:  toWireSendToolCalls(msg.ToolCalls),
		}
		if msg.Name != "" {
			groqMsg.Name = msg.Name
		}
		messages = append(messages, groqMsg)
	}

	return &GroqRequest{
		Model:       request.Model,
		Messages:    messages,
		MaxTokens:   request.MaxTokens,
		Temperature: request.Temperature,
		TopP:        request.TopP,
		Stream:      request.Stream,
		// OpenAI-compatible function-calling: forward the agent's tool
		// definitions + tool_choice so the model can actually request a
		// tool. omitempty keeps the wire byte-identical for plain chat
		// (request.Tools empty → field elided).
		Tools:      request.Tools,
		ToolChoice: request.ToolChoice,
	}, nil
}

func (gp *GroqProvider) convertFromGroqResponse(groqResp *GroqResponse, requestID uuid.UUID, processingTime time.Duration) *LLMResponse {
	var content string
	var finishReason string

	if len(groqResp.Choices) > 0 {
		choice := groqResp.Choices[0]
		content = choice.Message.Content
		finishReason = choice.FinishReason
	}

	return &LLMResponse{
		ID:        uuid.New(),
		RequestID: requestID,
		Content:   content,
		ToolCalls: groqResp.toolCalls,
		Usage: Usage{
			PromptTokens:     groqResp.Usage.PromptTokens,
			CompletionTokens: groqResp.Usage.CompletionTokens,
			TotalTokens:      groqResp.Usage.TotalTokens,
		},
		FinishReason:   finishReason,
		ProcessingTime: processingTime,
		CreatedAt:      time.Now(),
		// Round-50 LLMResponse.Err wiring (CONST-035 / Article XI §11.9):
		// Groq is OpenAI-compatible — finish_reason "length" /
		// "content_filter" indicate truncation / content block. Reuse
		// the round-46 OpenAI mapper helper (same closed mapping per
		// https://console.groq.com/docs/api-reference#chat).
		Err: mapOpenAIFinishReasonToErr(finishReason),
	}
}

func (gp *GroqProvider) makeGroqRequest(ctx context.Context, request *GroqRequest) (*GroqResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/openai/v1/chat/completions", gp.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	gp.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := gp.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, gp.handleGroqError(resp.StatusCode, body)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response GroqResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	// Parse OpenAI-compatible function-calling tool_calls out-of-band so the
	// public GroqResponse.Choices struct stays byte-compatible with existing
	// literals (see GroqResponse.toolCalls doc). Groq returns tool_calls
	// nested under choices[0].message.tool_calls exactly like OpenAI.
	var toolEnv groqToolCallsEnvelope
	if err := json.Unmarshal(body, &toolEnv); err == nil && len(toolEnv.Choices) > 0 {
		response.toolCalls = parseOpenAIWireToolCalls(toolEnv.Choices[0].Message.ToolCalls)
	}

	return &response, nil
}

// groqToolCallsEnvelope extracts ONLY choices[].message.tool_calls from a Groq
// Chat Completions response, used as a second decode pass in makeGroqRequest so
// the public GroqResponse shape is left untouched.
type groqToolCallsEnvelope struct {
	Choices []struct {
		Message struct {
			ToolCalls []openAIWireToolCall `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
}

func (gp *GroqProvider) makeGroqStreamRequest(ctx context.Context, request *GroqRequest, ch chan<- LLMResponse, requestID uuid.UUID, startTime time.Time) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/openai/v1/chat/completions", gp.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	gp.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := gp.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return gp.handleGroqError(resp.StatusCode, body)
	}

	// Parse SSE stream with metrics tracking
	return gp.parseSSEStreamWithMetrics(ctx, resp.Body, ch, requestID, startTime)
}

// parseSSEStreamWithMetrics parses a Groq SSE stream. ctx is honored via a
// select-based guard on every channel send: without it, a caller that
// cancels ctx and stops draining ch (an HTTP client disconnect mid-stream)
// leaves this goroutine — and the open response body it holds — blocked/
// leaked forever on an unguarded send. Mirrors the pattern already used by
// ollama_provider.go / openai_compatible_provider.go in this package.
func (gp *GroqProvider) parseSSEStreamWithMetrics(ctx context.Context, reader io.Reader, ch chan<- LLMResponse, requestID uuid.UUID, startTime time.Time) error {
	scanner := bufio.NewScanner(reader)
	var contentBuilder strings.Builder
	var firstTokenReceived bool
	var firstTokenTime time.Duration
	tokenCount := 0

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			break
		}

		// Track first token latency
		if !firstTokenReceived {
			firstTokenTime = time.Since(startTime)
			firstTokenReceived = true
		}

		// Parse JSON chunk
		var chunk GroqStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			log.Printf("Error parsing chunk: %v", err)
			continue
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta.Content
		if delta != "" {
			contentBuilder.WriteString(delta)
			tokenCount++

			// Send incremental response
			select {
			case ch <- LLMResponse{
				ID:        uuid.New(),
				RequestID: requestID,
				Content:   delta,
				CreatedAt: time.Now(),
			}:
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Handle completion
		if chunk.Choices[0].FinishReason != "" {
			totalTime := time.Since(startTime)
			tokensPerSecond := float64(tokenCount) / totalTime.Seconds()

			// Anti-bluff visual-correctness fix (CONST-035): the prior
			// code sent the full accumulated content here in addition
			// to the per-delta chunks above, causing every streamed
			// response to render as "<tokens><FULL>" — e.g. "1 2 31 2 3"
			// when the model produced "1 2 3". Renderers (ansi + plain)
			// just WriteToken every chunk's Content, so the final
			// "full content" chunk was being re-painted. Leave Content
			// empty here: the deltas above carry the visible content;
			// this terminal chunk carries only the metadata.
			finalResponse := LLMResponse{
				ID:           uuid.New(),
				RequestID:    requestID,
				Content:      "",
				FinishReason: chunk.Choices[0].FinishReason,
				CreatedAt:    time.Now(),
				ProviderMetadata: map[string]interface{}{
					"first_token_latency_ms": firstTokenTime.Milliseconds(),
					"total_latency_ms":       totalTime.Milliseconds(),
					"tokens_per_second":      tokensPerSecond,
				},
				// Round-50 LLMResponse.Err wiring for the streaming path
				// (CONST-035 / Article XI §11.9): same OpenAI-compat
				// mapping as Generate above — terminal frame carries
				// Err on truncation / content-filter so tool_provider
				// :201/:251 can surface it on the ToolStreamChunk.
				Err: mapOpenAIFinishReasonToErr(chunk.Choices[0].FinishReason),
			}

			if chunk.Usage != nil {
				finalResponse.Usage = Usage{
					PromptTokens:     chunk.Usage.PromptTokens,
					CompletionTokens: chunk.Usage.CompletionTokens,
					TotalTokens:      chunk.Usage.TotalTokens,
				}
			}

			select {
			case ch <- finalResponse:
			case <-ctx.Done():
				return ctx.Err()
			}

			// Record metrics
			gp.latencyMetrics.RecordRequest(firstTokenTime, totalTime, tokensPerSecond)
		}
	}

	return scanner.Err()
}

func (gp *GroqProvider) setAuthHeaders(req *http.Request) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gp.apiKey))
}

func (gp *GroqProvider) handleGroqError(statusCode int, body []byte) error {
	var groqErr GroqError
	if err := json.Unmarshal(body, &groqErr); err == nil && groqErr.Error.Message != "" {
		errInfo := groqErr.Error

		switch statusCode {
		case http.StatusBadRequest:
			if strings.Contains(errInfo.Message, "context_length_exceeded") {
				return ErrContextTooLong
			}
			return fmt.Errorf("invalid request: %s", errInfo.Message)

		case http.StatusUnauthorized:
			return fmt.Errorf("unauthorized: invalid Groq API key")

		case http.StatusTooManyRequests:
			return ErrRateLimited

		case http.StatusServiceUnavailable:
			return fmt.Errorf("groq service unavailable: %s", errInfo.Message)

		case 529: // Groq overload
			return fmt.Errorf("groq overloaded: please retry after a moment")

		default:
			return fmt.Errorf("groq API error (%d): %s", statusCode, errInfo.Message)
		}
	}

	// Fallback
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("unauthorized: check API key")
	case http.StatusTooManyRequests:
		return ErrRateLimited
	default:
		return fmt.Errorf("groq API error (%d): %s", statusCode, string(body))
	}
}

// Groq API types (OpenAI-compatible)

type GroqRequest struct {
	Model       string        `json:"model"`
	Messages    []GroqMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	TopP        float64       `json:"top_p,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	// OpenAI-compatible function-calling fields. omitempty ⇒ plain-chat
	// requests (no tools) serialise byte-identically to the prior wire.
	Tools      []Tool      `json:"tools,omitempty"`
	ToolChoice interface{} `json:"tool_choice,omitempty"`
}

type GroqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
	// Tool-conversation protocol fields. omitempty ⇒ plain-chat messages
	// serialise byte-identically to the prior wire. ToolCallID is REQUIRED by
	// Groq on every role:"tool" result message; ToolCalls carries the
	// assistant turn's requested calls so each result has a matching call.
	//
	// SEND-side wire type: ToolCalls uses wireSendToolCall so
	// function.arguments serialises as a JSON STRING (Groq rejects the object
	// form: 'messages.N.tool_calls.0.function.arguments' : value must be a
	// string). Convert via toWireSendToolCalls in buildGroqRequest.
	ToolCallID string             `json:"tool_call_id,omitempty"`
	ToolCalls  []wireSendToolCall `json:"tool_calls,omitempty"`
}

type GroqResponse struct {
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
	} `json:"usage"`

	// toolCalls holds the parsed function-calling tool_calls of the FIRST
	// choice. It is populated out-of-band by makeGroqRequest (NOT a JSON tag
	// on GroqResponse) so the public anonymous Choices struct shape stays
	// byte-compatible with existing GroqResponse literals in tests.
	toolCalls []ToolCall
}

type GroqStreamChunk struct {
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
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

type GroqError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// openAIWireToolCall is the on-wire shape of a single `tool_calls[]` entry as
// emitted by OpenAI Chat Completions–compatible providers (Groq, DeepSeek,
// Mistral, OpenRouter, …). The crucial divergence from llm.ToolCall is that
// `function.arguments` arrives as a JSON **string** (a serialized object),
// not a JSON object — so it is captured as json.RawMessage and decoded by
// parseOpenAIWireToolCalls into the map[string]interface{} that
// llm.ToolCallFunc.Arguments (and every downstream tool-loop consumer) expects.
type openAIWireToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"function"`
}

// parseOpenAIWireToolCalls converts the OpenAI-compatible wire `tool_calls`
// array into []llm.ToolCall, decoding `function.arguments` whether the
// provider sent it as a JSON-string-encoded object (OpenAI's canonical form,
// e.g. "{\"path\":\".\"}") or as a raw JSON object (some compatible servers).
// Returns nil for an empty/absent array so plain-chat responses are unchanged.
func parseOpenAIWireToolCalls(wire []openAIWireToolCall) []ToolCall {
	if len(wire) == 0 {
		return nil
	}
	out := make([]ToolCall, 0, len(wire))
	for _, w := range wire {
		args := map[string]interface{}{}
		if raw := []byte(w.Function.Arguments); len(raw) > 0 {
			// First try: arguments is a JSON string holding an object
			// (the OpenAI canonical encoding).
			var asString string
			if err := json.Unmarshal(raw, &asString); err == nil {
				if asString != "" {
					if uerr := json.Unmarshal([]byte(asString), &args); uerr != nil {
						args = map[string]interface{}{}
					}
				}
			} else {
				// Fallback: arguments is already a JSON object.
				if uerr := json.Unmarshal(raw, &args); uerr != nil {
					args = map[string]interface{}{}
				}
			}
		}
		callType := w.Type
		if callType == "" {
			callType = "function"
		}
		out = append(out, ToolCall{
			ID:   w.ID,
			Type: callType,
			Function: ToolCallFunc{
				Name:      w.Function.Name,
				Arguments: args,
			},
		})
	}
	return out
}
