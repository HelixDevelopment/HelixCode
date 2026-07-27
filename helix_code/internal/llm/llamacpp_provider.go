package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Remove unused imports - io is not used

// LlamaCPPProvider implements the LLM provider interface for Llama.cpp
type LlamaCPPProvider struct {
	config    LlamaConfig
	isRunning bool
}

// LlamaConfig holds configuration for Llama.cpp
type LlamaConfig struct {
	Model         string        `json:"model"`
	ContextSize   int           `json:"context_size"`
	GPUEnabled    bool          `json:"gpu_enabled"`
	GPULayers     int           `json:"gpu_layers"`
	Threads       int           `json:"threads"`
	ServerHost    string        `json:"server_host"`
	ServerPort    int           `json:"server_port"`
	ServerTimeout time.Duration `json:"server_timeout"`
}

// NewLlamaCPPProvider creates a new Llama.cpp provider
func NewLlamaCPPProvider(config LlamaConfig) (*LlamaCPPProvider, error) {
	provider := &LlamaCPPProvider{
		config:    config,
		isRunning: true,
	}

	log.Printf("✅ Llama.cpp provider initialized with model: %s", config.Model)
	return provider, nil
}

// GetType returns the provider type
func (p *LlamaCPPProvider) GetType() ProviderType {
	return ProviderTypeLocal
}

// GetName returns the provider name
func (p *LlamaCPPProvider) GetName() string {
	return "llama-cpp"
}

// GetModels returns available models
// ANTI-BLUFF: Query REAL Llama.cpp server for available models
func (p *LlamaCPPProvider) GetModels() []ModelInfo {
	baseURL := p.config.ServerHost
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	// REAL HTTP call to Llama.cpp server
	resp, err := http.Get(baseURL + "/models")
	if err != nil {
		log.Printf("Failed to list Llama.cpp models: %v", err)
		// Return configured model as fallback
		models := []ModelInfo{
			{
				Name:        p.config.Model,
				Provider:    ProviderTypeLocal,
				ContextSize: p.config.ContextSize,
				MaxTokens:   p.config.ContextSize,
				Description: "Local Llama.cpp model",
			},
		}
		for i := range models {
			EnrichModelInfo(&models[i])
		}
		return models
	}
	defer resp.Body.Close()

	// Parse response
	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Failed to decode Llama.cpp models: %v", err)
		return []ModelInfo{}
	}

	models := make([]ModelInfo, len(result.Models))
	for i, m := range result.Models {
		models[i] = ModelInfo{
			Name:        m.Name,
			Provider:    ProviderTypeLocal,
			ContextSize: p.config.ContextSize,
			MaxTokens:   p.config.ContextSize,
			Description: "Local Llama.cpp model",
		}
	}
	for i := range models {
		EnrichModelInfo(&models[i])
	}
	return models
}

// GetCapabilities returns model capabilities
func (p *LlamaCPPProvider) GetCapabilities() []ModelCapability {
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

// Generate generates a response using Llama.cpp
// ANTI-BLUFF: This MUST make REAL HTTP call to Llama.cpp server
func (p *LlamaCPPProvider) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	if !p.isRunning {
		return nil, ErrProviderUnavailable
	}

	// REAL implementation - connect to Llama.cpp server
	baseURL := p.config.ServerHost
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	if p.config.ServerPort != 0 {
		baseURL = fmt.Sprintf("%s:%d", baseURL, p.config.ServerPort)
	}

	// Llama.cpp uses /v1/completions endpoint (OpenAI compatible)
	apiURL := baseURL + "/v1/completions"

	// Build request payload - use Messages if available, otherwise use Model
	var payload map[string]interface{}
	if len(request.Messages) > 0 {
		payload = map[string]interface{}{
			"model":       p.config.Model,
			"messages":    request.Messages,
			"max_tokens":  request.MaxTokens,
			"temperature": request.Temperature,
			"stream":      false,
		}
	} else {
		// Convert Messages to prompt string
		prompt := ""
		for _, msg := range request.Messages {
			prompt += msg.Role + ": " + msg.Content + "\n"
		}
		payload = map[string]interface{}{
			"model":       p.config.Model,
			"prompt":      prompt,
			"max_tokens":  request.MaxTokens,
			"temperature": request.Temperature,
			"stream":      false,
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make REAL HTTP call to Llama.cpp server
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Capture the request start BEFORE the round-trip so ProcessingTime
	// measures the real wall-clock latency of the call. (Round-63 anti-bluff:
	// the prior `time.Since(time.Now())` always evaluated to ~0, so every
	// llama.cpp response reported a meaningless near-zero ProcessingTime,
	// corrupting downstream latency consumers — notably the ensemble
	// quality_weighted speed bonus at ensemble_provider.go and usage analytics.)
	startTime := time.Now()
	client := &http.Client{Timeout: p.config.ServerTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llama.cpp request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("llama.cpp returned status %d", resp.StatusCode)
	}

	// Parse REAL response. llama.cpp's /v1/completions emits an
	// OpenAI-compatible envelope when the request used the chat-completions
	// shape (modern llama-server v0.0.5+); older payloads carry a top-level
	// `content` + legacy stop flags (stopped_eos / stopped_limit /
	// stopped_word). We decode BOTH shapes so the mapper sees whichever
	// signal the server actually emitted (round-53 anti-bluff: callers can
	// distinguish a clean stop from a truncated max-tokens stop on either
	// API path).
	var result struct {
		// OpenAI-compatible shape
		Choices []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		// Legacy llama.cpp shape
		Content      string `json:"content"`
		StoppedEOS   bool   `json:"stopped_eos"`
		StoppedLimit bool   `json:"stopped_limit"`
		StoppedWord  bool   `json:"stopped_word"`
		Usage        struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Round-53 LLMResponse.Err wiring (CONST-035 / Article XI §11.9):
	// resolve the effective content, finish_reason, and Err sentinel from
	// whichever response shape the server emitted. OpenAI-compatible path
	// reuses the round-46 mapOpenAIFinishReasonToErr helper; legacy path
	// uses the round-53-new mapLlamaCppStopFlagsToErr helper. Llama.cpp
	// exposes NO content-filter signal on the wire (no safety classifier),
	// so ErrResponseContentBlocked is not reachable here — documented in
	// the helper's doc comment.
	content := result.Content
	finishReason := ""
	var sentinel error
	if len(result.Choices) > 0 {
		// OpenAI-compatible path takes precedence when populated
		content = result.Choices[0].Message.Content
		finishReason = result.Choices[0].FinishReason
		sentinel = mapOpenAIFinishReasonToErr(finishReason)
	} else {
		// Legacy /completion or /v1/completions emitting legacy shape
		finishReason, sentinel = mapLlamaCppStopFlagsToErr(result.StoppedEOS, result.StoppedLimit, result.StoppedWord)
	}

	return &LLMResponse{
		ID:        uuid.New(),
		RequestID: request.ID,
		Content:   content,
		Usage: Usage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		},
		FinishReason:   finishReason,
		ProcessingTime: time.Since(startTime),
		CreatedAt:      time.Now(),
		Err:            sentinel,
	}, nil
}

// mapLlamaCppStopFlagsToErr maps llama.cpp's legacy `/completion` boolean
// stop-cause flags to the round-46 LLMResponse.Err sentinel + a synthetic
// finish_reason string for FinishReason. Reference:
//
//	https://github.com/ggerganov/llama.cpp/blob/master/examples/server/README.md
//
// (server response fields: stopped_eos / stopped_limit / stopped_word /
//
//	stopped_word_str). Closed mapping:
//	 - stopped_limit=true                                       → "length",  ErrResponseTruncated
//	 - stopped_eos=true                                         → "stop",    nil
//	 - stopped_word=true                                        → "stop",    nil  (custom stop seq hit)
//	 - all-false (mid-stream chunk or unknown termination)      → "",        nil
//
// llama.cpp exposes NO content-filter / safety-classifier signal on the
// wire, so ErrResponseContentBlocked is NOT reachable here. If a future
// llama.cpp release adds one, the helper + paired pinning test MUST be
// extended in the same commit (CONST-050(B) paired-mutation).
func mapLlamaCppStopFlagsToErr(stoppedEOS, stoppedLimit, stoppedWord bool) (string, error) {
	switch {
	case stoppedLimit:
		return "length", ErrResponseTruncated
	case stoppedEOS:
		return "stop", nil
	case stoppedWord:
		return "stop", nil
	default:
		return "", nil
	}
}

// GenerateStream generates a streaming response
// ANTI-BLUFF: This MUST stream REAL response from Llama.cpp server
func (p *LlamaCPPProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
	// Channel-ownership contract (see Provider.GenerateStream interface doc):
	// the provider (the SENDER) is the SOLE closer of ch, and MUST close it on
	// every return path — success, error, and ctx-cancel. The consumer never
	// closes ch; a double-close would panic in a spawned goroutine and crash the
	// process (server defect #5). defer guarantees close on every return path.
	defer close(ch)
	if !p.isRunning {
		return ErrProviderUnavailable
	}

	// REAL streaming implementation
	baseURL := p.config.ServerHost
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	if p.config.ServerPort != 0 {
		baseURL = fmt.Sprintf("%s:%d", baseURL, p.config.ServerPort)
	}

	// Build prompt from messages or use Model field
	prompt := ""
	if len(request.Messages) > 0 {
		for _, msg := range request.Messages {
			prompt += msg.Role + ": " + msg.Content + "\n"
		}
	} else {
		prompt = request.Model
	}

	payload := map[string]interface{}{
		"model":       p.config.Model,
		"prompt":      prompt,
		"max_tokens":  request.MaxTokens,
		"temperature": request.Temperature,
		"stream":      true,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/completion", bytes.NewReader(body))
	if err != nil {
		// A malformed URL makes NewRequestWithContext return (nil, err).
		// Discarding this error made the next line dereference a nil
		// *http.Request — an unrecoverable panic in the caller's goroutine
		// that killed the process. Mirrors Generate's handling above.
		return fmt.Errorf("llama.cpp stream request creation failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 0} // No timeout for streaming
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("llama.cpp stream request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read SSE stream from REAL server
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		// Llama.cpp's /completion SSE delivers per-token chunks with a
		// `content` field, then a terminal chunk with `stop:true` plus
		// the boolean cause flags (stopped_eos / stopped_limit /
		// stopped_word). Round-53 LLMResponse.Err wiring (CONST-035 /
		// Article XI §11.9): when the terminal chunk indicates
		// max-tokens truncation (stopped_limit=true), emit a final
		// LLMResponse with Err=ErrResponseTruncated so tool_provider.go
		// :201/:251 can distinguish a clean stop from a truncated stop.
		var chunk struct {
			Content      string `json:"content"`
			Stop         bool   `json:"stop"`
			StoppedEOS   bool   `json:"stopped_eos"`
			StoppedLimit bool   `json:"stopped_limit"`
			StoppedWord  bool   `json:"stopped_word"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		// Emit content-bearing chunks as before (backward-compat).
		if chunk.Content != "" {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case ch <- LLMResponse{
				ID:        uuid.New(),
				RequestID: request.ID,
				Content:   chunk.Content,
				CreatedAt: time.Now(),
			}:
			}
		}

		// On the terminal chunk, if the legacy stop flags signal a
		// known-degradation cause, emit one final Err-bearing
		// LLMResponse + break out of the loop.
		if chunk.Stop {
			finishReason, sentinel := mapLlamaCppStopFlagsToErr(chunk.StoppedEOS, chunk.StoppedLimit, chunk.StoppedWord)
			if sentinel != nil {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case ch <- LLMResponse{
					ID:           uuid.New(),
					RequestID:    request.ID,
					FinishReason: finishReason,
					CreatedAt:    time.Now(),
					Err:          sentinel,
				}:
				}
			}
			break
		}
	}

	return scanner.Err()
}

// IsAvailable checks if the provider is available
func (p *LlamaCPPProvider) IsAvailable(ctx context.Context) bool {
	return p.isRunning
}

// GetHealth returns provider health status
func (p *LlamaCPPProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	if !p.isRunning {
		return &ProviderHealth{
			Status:     "unhealthy",
			LastCheck:  time.Now(),
			ErrorCount: 1,
		}, nil
	}

	return &ProviderHealth{
		Status:     "healthy",
		LastCheck:  time.Now(),
		ErrorCount: 0,
		ModelCount: len(p.GetModels()),
	}, nil
}

// Close stops the Llama.cpp provider
func (p *LlamaCPPProvider) Close() error {
	p.isRunning = false
	log.Println("✅ Llama.cpp provider closed")
	return nil
}

// GetContextWindow returns the model's context window size in tokens.
// Uses the ContextSize from the LlamaConfig when set; falls back to 200_000.
func (p *LlamaCPPProvider) GetContextWindow() int {
	if p.config.ContextSize > 0 {
		return p.config.ContextSize
	}
	return 200_000
}

// CountTokens returns an estimated token count for text.
// Uses char-based fallback (1 token ≈ 3.5 chars) — Phase 3 will upgrade
// to the llama.cpp /tokenize endpoint.
func (p *LlamaCPPProvider) CountTokens(text string) (int, error) {
	return CharBasedTokenCount(text)
}
