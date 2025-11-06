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

// Helper methods

func getGroqModels() []ModelInfo {
	allCapabilities := []ModelCapability{
		CapabilityTextGeneration,
		CapabilityCodeGeneration,
		CapabilityCodeAnalysis,
		CapabilityPlanning,
		CapabilityDebugging,
		CapabilityRefactoring,
		CapabilityTesting,
	}

	textCapabilities := []ModelCapability{
		CapabilityTextGeneration,
		CapabilityCodeGeneration,
		CapabilityCodeAnalysis,
	}

	return []ModelInfo{
		{
			Name:           "llama-3.3-70b-versatile",
			Provider:       ProviderTypeGroq,
			ContextSize:    131072,
			MaxTokens:      32768,
			Capabilities:   allCapabilities,
			SupportsTools:  true,
			SupportsVision: false,
			Description:    "Llama 3.3 70B - Most capable model on Groq with ultra-fast inference",
		},
		{
			Name:           "llama-3.1-70b-versatile",
			Provider:       ProviderTypeGroq,
			ContextSize:    131072,
			MaxTokens:      8192,
			Capabilities:   allCapabilities,
			SupportsTools:  true,
			SupportsVision: false,
			Description:    "Llama 3.1 70B - Previous generation with excellent speed",
		},
		{
			Name:           "llama-3.1-8b-instant",
			Provider:       ProviderTypeGroq,
			ContextSize:    131072,
			MaxTokens:      8192,
			Capabilities:   allCapabilities,
			SupportsTools:  true,
			SupportsVision: false,
			Description:    "Llama 3.1 8B - Extremely fast, ideal for high-volume use",
		},
		{
			Name:           "mixtral-8x7b-32768",
			Provider:       ProviderTypeGroq,
			ContextSize:    32768,
			MaxTokens:      32768,
			Capabilities:   allCapabilities,
			SupportsTools:  true,
			SupportsVision: false,
			Description:    "Mixtral 8x7B - Mixture of experts with fast inference",
		},
		{
			Name:           "gemma2-9b-it",
			Provider:       ProviderTypeGroq,
			ContextSize:    8192,
			MaxTokens:      8192,
			Capabilities:   textCapabilities,
			SupportsTools:  false,
			SupportsVision: false,
			Description:    "Gemma 2 9B - Google's efficient open model on Groq",
		},
		{
			Name:           "gemma-7b-it",
			Provider:       ProviderTypeGroq,
			ContextSize:    8192,
			MaxTokens:      8192,
			Capabilities:   textCapabilities,
			SupportsTools:  false,
			SupportsVision: false,
			Description:    "Gemma 7B - Compact and fast",
		},
	}
}

func (gp *GroqProvider) buildGroqRequest(request *LLMRequest) (*GroqRequest, error) {
	// Convert messages to Groq (OpenAI-compatible) format
	var messages []GroqMessage
	for _, msg := range request.Messages {
		groqMsg := GroqMessage{
			Role:    msg.Role,
			Content: msg.Content,
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
		Usage: Usage{
			PromptTokens:     groqResp.Usage.PromptTokens,
			CompletionTokens: groqResp.Usage.CompletionTokens,
			TotalTokens:      groqResp.Usage.TotalTokens,
		},
		FinishReason:   finishReason,
		ProcessingTime: processingTime,
		CreatedAt:      time.Now(),
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

	var response GroqResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
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
	return gp.parseSSEStreamWithMetrics(resp.Body, ch, requestID, startTime)
}

func (gp *GroqProvider) parseSSEStreamWithMetrics(reader io.Reader, ch chan<- LLMResponse, requestID uuid.UUID, startTime time.Time) error {
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
			ch <- LLMResponse{
				ID:        uuid.New(),
				RequestID: requestID,
				Content:   delta,
				CreatedAt: time.Now(),
			}
		}

		// Handle completion
		if chunk.Choices[0].FinishReason != "" {
			totalTime := time.Since(startTime)
			tokensPerSecond := float64(tokenCount) / totalTime.Seconds()

			finalResponse := LLMResponse{
				ID:           uuid.New(),
				RequestID:    requestID,
				Content:      contentBuilder.String(),
				FinishReason: chunk.Choices[0].FinishReason,
				CreatedAt:    time.Now(),
				ProviderMetadata: map[string]interface{}{
					"first_token_latency_ms": firstTokenTime.Milliseconds(),
					"total_latency_ms":       totalTime.Milliseconds(),
					"tokens_per_second":      tokensPerSecond,
				},
			}

			if chunk.Usage != nil {
				finalResponse.Usage = Usage{
					PromptTokens:     chunk.Usage.PromptTokens,
					CompletionTokens: chunk.Usage.CompletionTokens,
					TotalTokens:      chunk.Usage.TotalTokens,
				}
			}

			ch <- finalResponse

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
}

type GroqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
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
