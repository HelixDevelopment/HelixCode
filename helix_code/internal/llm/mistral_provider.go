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

// MistralProvider implements the Provider interface for the Mistral
// Cloud (api.mistral.ai) OpenAI-compatible endpoint. Round-41 F12
// fast-path expansion: a user with MISTRAL_API_KEY can run
// `HELIX_LLM_PROVIDER=mistral ./bin/cli` without first editing
// config.yaml or starting the HelixCode server.
type MistralProvider struct {
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
}

// NewMistralProvider creates a new Mistral provider.
func NewMistralProvider(config ProviderConfigEntry) (*MistralProvider, error) {
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "https://api.mistral.ai/v1"
	}

	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("MISTRAL_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("Mistral API key is required (set config.APIKey or MISTRAL_API_KEY env var)")
	}

	p := &MistralProvider{
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

	p.initializeModels()

	return p, nil
}

func (mp *MistralProvider) GetType() ProviderType { return ProviderTypeMistral }
func (mp *MistralProvider) GetName() string       { return "Mistral" }

// GetModels returns available models. CONST-036 / F6-D-5: refreshed LIVE from
// Mistral's GET /models on first call (cached); seed list is offline fallback.
func (mp *MistralProvider) GetModels() []ModelInfo {
	mp.refreshCatalogOnce()
	mp.catalogMu.RLock()
	defer mp.catalogMu.RUnlock()
	return mp.models
}

func (mp *MistralProvider) refreshCatalogOnce() {
	mp.catalogOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		models, err := fetchOpenAICompatibleCatalog(ctx, mp.endpoint, mp.apiKey, mp.httpClient, ProviderTypeMistral, 131072, 8192)
		if err != nil {
			log.Printf("⚠️  Mistral /models fetch failed (%v); keeping verified seed list", err)
			return
		}
		for i := range models {
			EnrichModelInfo(&models[i])
		}
		mp.catalogMu.Lock()
		mp.models = models
		mp.catalogMu.Unlock()
		log.Printf("✅ Mistral catalog refreshed with %d models (live /models)", len(models))
	})
}

func (mp *MistralProvider) GetCapabilities() []ModelCapability {
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

// Generate makes a chat-completions request against api.mistral.ai.
func (mp *MistralProvider) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	startTime := time.Now()

	openaiRequest, err := mp.convertToOpenAIRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %v", err)
	}

	response, err := mp.makeOpenAIRequest(ctx, openaiRequest)
	if err != nil {
		return nil, fmt.Errorf("Mistral request failed: %v", err)
	}

	return mp.convertFromOpenAIResponse(response, request.ID, time.Since(startTime)), nil
}

func (mp *MistralProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
	defer close(ch)

	openaiRequest, err := mp.convertToOpenAIRequest(request)
	if err != nil {
		return fmt.Errorf("failed to convert request: %v", err)
	}
	openaiRequest.Stream = true

	return mp.makeOpenAIStreamRequest(ctx, openaiRequest, ch, request.ID)
}

func (mp *MistralProvider) IsAvailable(ctx context.Context) bool {
	health, err := mp.GetHealth(ctx)
	return err == nil && health.Status == "healthy"
}

// GetHealth probes the Mistral API and returns the resulting health record.
//
// The returned record is a COPY the caller owns outright — mutating it cannot
// reach provider state (HXC-214).
func (mp *MistralProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/models", mp.endpoint), nil)
	if err != nil {
		return mp.recordHealth("unhealthy", 0, errCountIncrement, modelCountUnchanged),
			fmt.Errorf("failed to create health check request: %v", err)
	}
	mp.setAuthHeaders(req)

	start := time.Now()
	resp, err := mp.httpClient.Do(req)
	latency := time.Since(start)
	if err != nil {
		return mp.recordHealth("unhealthy", latency, errCountIncrement, modelCountUnchanged),
			fmt.Errorf("health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mp.recordHealth("unhealthy", latency, errCountIncrement, modelCountUnchanged),
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
		return mp.recordHealth("degraded", latency, errCountKeep, modelCountUnchanged), nil
	}

	return mp.recordHealth("healthy", latency, errCountReset, len(modelsResponse.Data)), nil
}

func (mp *MistralProvider) Close() error {
	mp.httpClient.CloseIdleConnections()
	return nil
}

// GetContextWindow returns the largest known Mistral context window.
// Codestral / Mistral-Large support 128k+; 128k is the published max.
func (mp *MistralProvider) GetContextWindow() int { return 128_000 }

func (mp *MistralProvider) CountTokens(text string) (int, error) {
	return CharBasedTokenCount(text)
}

func (mp *MistralProvider) initializeModels() {
	// CONST-036 / F6-D-5: verified SEED only (no network at construction). The
	// authoritative list is refreshed LIVE from GET /models on the first
	// GetModels() call (see refreshCatalogOnce).
	mp.models = []ModelInfo{
		{
			Name:        "mistral-small-latest",
			Provider:    ProviderTypeMistral,
			ContextSize: 131072,
			MaxTokens:   8192,
			Description: "Mistral Small (latest) - fast, efficient general-purpose model",
		},
		{
			Name:        "mistral-medium-latest",
			Provider:    ProviderTypeMistral,
			ContextSize: 131072,
			MaxTokens:   8192,
			Description: "Mistral Medium (latest) - balanced cost/quality",
		},
		{
			Name:        "mistral-large-latest",
			Provider:    ProviderTypeMistral,
			ContextSize: 131072,
			MaxTokens:   8192,
			Description: "Mistral Large (latest) - flagship reasoning model",
		},
		{
			Name:        "codestral-latest",
			Provider:    ProviderTypeMistral,
			ContextSize: 256000,
			MaxTokens:   8192,
			Description: "Codestral (latest) - code-specialised model with 256k context",
		},
		{
			Name:        "pixtral-12b-2409",
			Provider:    ProviderTypeMistral,
			ContextSize: 131072,
			MaxTokens:   8192,
			Description: "Pixtral 12B - multimodal (vision + text)",
		},
		{
			Name:        "open-mixtral-8x22b",
			Provider:    ProviderTypeMistral,
			ContextSize: 65536,
			MaxTokens:   8192,
			Description: "Open Mixtral 8x22B - open-weights MoE",
		},
	}

	for i := range mp.models {
		EnrichModelInfo(&mp.models[i])
	}

	log.Printf("✅ Mistral provider initialized with %d models", len(mp.models))
}

func (mp *MistralProvider) convertToOpenAIRequest(request *LLMRequest) (*mistralChatRequest, error) {
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

	return &mistralChatRequest{
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

func (mp *MistralProvider) convertFromOpenAIResponse(openaiResp *mistralChatResponse, requestID uuid.UUID, processingTime time.Duration) *LLMResponse {
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
		// Round-50 LLMResponse.Err wiring (CONST-035 / Article XI §11.9):
		// Mistral OpenAI-compat with provider-specific extensions —
		// uses both "length" and "model_length" for truncation (per
		// https://docs.mistral.ai/api/#tag/chat). No content-filter at
		// protocol level; safety lives in separate guardrails layer.
		Err: mapMistralFinishReasonToErr(finish),
	}
}

// mapMistralFinishReasonToErr returns the round-50 sentinel that matches
// a Mistral finish_reason value, or nil for clean stops. Documented
// values per https://docs.mistral.ai/api/#tag/chat :
//   - "stop" / "tool_calls" / "" : clean → nil
//   - "length" / "model_length"  : both = max-tokens-reached → ErrResponseTruncated
//
// Mistral surfaces NO content-filter signal on the wire (safety lives
// in a separate guardrails layer), so ErrResponseContentBlocked is not
// reachable here.
func mapMistralFinishReasonToErr(reason string) error {
	switch reason {
	case "length", "model_length":
		return ErrResponseTruncated
	default:
		return nil
	}
}

func (mp *MistralProvider) makeOpenAIRequest(ctx context.Context, request *mistralChatRequest) (*mistralChatResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/chat/completions", mp.endpoint), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	mp.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := mp.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Mistral API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response mistralChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (mp *MistralProvider) makeOpenAIStreamRequest(ctx context.Context, request *mistralChatRequest, ch chan<- LLMResponse, requestID uuid.UUID) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/chat/completions", mp.endpoint), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	mp.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := mp.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Mistral API returned status %d: %s", resp.StatusCode, string(body))
	}

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
			// Round-50 LLMResponse.Err wiring for the streaming path
			// (CONST-035 / Article XI §11.9): when the final frame
			// carries finish_reason="length"/"model_length", emit a
			// terminal LLMResponse with Err populated so stream
			// consumers (notably tool_provider.go :201/:251) can
			// distinguish a clean stop from a truncation.
			finishReason := streamResp.Choices[0].FinishReason
			if errSentinel := mapMistralFinishReasonToErr(finishReason); errSentinel != nil {
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

func (mp *MistralProvider) setAuthHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+mp.apiKey)
}

// recordHealth applies a health verdict to lastHealth under healthMu and
// returns a copy the caller owns outright (HXC-214). The whole verdict lands in
// ONE critical section, so a concurrent reader cannot observe it half-applied.
func (mp *MistralProvider) recordHealth(
	status string,
	latency time.Duration,
	adj errCountAdjust,
	modelCount int,
) *ProviderHealth {
	mp.healthMu.Lock()
	defer mp.healthMu.Unlock()
	return applyProviderHealth(mp.lastHealth, status, latency, adj, modelCount)
}

// mistralChatRequest mirrors the shared OpenAIRequest but adds the
// OpenAI-compatible function-calling fields (Mistral is OpenAI Chat
// Completions–compatible). omitempty keeps plain-chat requests byte-identical.
type mistralChatRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Tools       []Tool          `json:"tools,omitempty"`
	ToolChoice  interface{}     `json:"tool_choice,omitempty"`
}

// mistralChatResponse mirrors the shared OpenAIResponse but adds
// message.tool_calls parsing.
type mistralChatResponse struct {
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
