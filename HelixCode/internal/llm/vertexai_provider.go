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

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// VertexAIProvider implements the Provider interface for Google's Vertex AI platform
// Supports both Gemini models (Google native) and Claude models via Model Garden
type VertexAIProvider struct {
	config        ProviderConfigEntry
	credentials   *google.Credentials
	projectID     string
	location      string
	endpoint      string
	httpClient    *http.Client
	models        []ModelInfo
	tokenProvider *TokenProvider
	lastHealth    *ProviderHealth
}

// TokenProvider manages OAuth2 tokens with caching
type TokenProvider struct {
	credentials *google.Credentials
	tokenCache  *oauth2.Token
	mutex       sync.RWMutex
}

// Vertex AI API structures for Gemini models
type vertexRequest struct {
	Contents          []vertexContent       `json:"contents"`
	SystemInstruction *vertexContent        `json:"systemInstruction,omitempty"`
	Tools             []vertexTool          `json:"tools,omitempty"`
	ToolConfig        *vertexToolConfig     `json:"toolConfig,omitempty"`
	SafetySettings    []vertexSafetySetting `json:"safetySettings,omitempty"`
	GenerationConfig  *vertexGenConfig      `json:"generationConfig,omitempty"`
}

type vertexContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []vertexPart `json:"parts"`
}

type vertexPart interface{}

type vertexTool struct {
	FunctionDeclarations []vertexFunctionDeclaration `json:"functionDeclarations,omitempty"`
}

type vertexFunctionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type vertexToolConfig struct {
	FunctionCallingConfig *vertexFunctionCallingConfig `json:"functionCallingConfig,omitempty"`
}

type vertexFunctionCallingConfig struct {
	Mode string `json:"mode"` // "AUTO", "ANY", "NONE"
}

type vertexSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type vertexGenConfig struct {
	Temperature     float64  `json:"temperature,omitempty"`
	TopP            float64  `json:"topP,omitempty"`
	TopK            int      `json:"topK,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

type vertexResponse struct {
	Candidates     []vertexCandidate     `json:"candidates"`
	UsageMetadata  *vertexUsageMetadata  `json:"usageMetadata,omitempty"`
	PromptFeedback *vertexPromptFeedback `json:"promptFeedback,omitempty"`
}

type vertexCandidate struct {
	Content       vertexContent        `json:"content"`
	FinishReason  string               `json:"finishReason"`
	SafetyRatings []vertexSafetyRating `json:"safetyRatings,omitempty"`
	Index         int                  `json:"index"`
}

type vertexSafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

type vertexPromptFeedback struct {
	BlockReason   string               `json:"blockReason,omitempty"`
	SafetyRatings []vertexSafetyRating `json:"safetyRatings,omitempty"`
}

type vertexUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// Anthropic/Claude structures for Model Garden
type anthropicVertexRequest struct {
	AnthropicVersion string                   `json:"anthropic_version"`
	Messages         []anthropicVertexMessage `json:"messages"`
	MaxTokens        int                      `json:"max_tokens"`
	Temperature      float64                  `json:"temperature,omitempty"`
	TopP             float64                  `json:"top_p,omitempty"`
	Stream           bool                     `json:"stream,omitempty"`
	System           string                   `json:"system,omitempty"`
}

type anthropicVertexMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicVertexResponse struct {
	ID         string                   `json:"id"`
	Type       string                   `json:"type"`
	Role       string                   `json:"role"`
	Content    []anthropicVertexContent `json:"content"`
	StopReason string                   `json:"stop_reason"`
	Usage      anthropicVertexUsage     `json:"usage"`
}

type anthropicVertexContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicVertexUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Vertex AI error structure
type vertexAIError struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
		Details []struct {
			Type     string            `json:"@type"`
			Reason   string            `json:"reason"`
			Domain   string            `json:"domain"`
			Metadata map[string]string `json:"metadata"`
		} `json:"details"`
	} `json:"error"`
}

// NewTokenProvider creates a new token provider
func NewTokenProvider(credentials *google.Credentials) *TokenProvider {
	return &TokenProvider{
		credentials: credentials,
	}
}

// GetToken retrieves a valid OAuth2 token (with caching)
func (tp *TokenProvider) GetToken(ctx context.Context) (string, error) {
	tp.mutex.RLock()
	if tp.tokenCache != nil && tp.tokenCache.Valid() {
		token := tp.tokenCache.AccessToken
		tp.mutex.RUnlock()
		return token, nil
	}
	tp.mutex.RUnlock()

	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	// Double-check after acquiring write lock
	if tp.tokenCache != nil && tp.tokenCache.Valid() {
		return tp.tokenCache.AccessToken, nil
	}

	// Get new token
	tokenSource := tp.credentials.TokenSource
	token, err := tokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	tp.tokenCache = token
	return token.AccessToken, nil
}

// NewVertexAIProvider creates a new Vertex AI provider
func NewVertexAIProvider(config ProviderConfigEntry) (*VertexAIProvider, error) {
	ctx := context.Background()

	// Get project ID from config or credentials
	projectID, ok := config.Parameters["project_id"].(string)
	if !ok || projectID == "" {
		projectID = os.Getenv("VERTEXAI_PROJECT")
		if projectID == "" {
			projectID = os.Getenv("GCP_PROJECT")
		}
	}

	// Get location/region
	location, ok := config.Parameters["location"].(string)
	if !ok || location == "" {
		location = os.Getenv("VERTEXAI_LOCATION")
		if location == "" {
			location = "us-central1" // Default location
		}
	}

	// Get credentials
	var credentials *google.Credentials
	var err error

	// Try credentials path first
	credentialsPath, _ := config.Parameters["credentials_path"].(string)
	if credentialsPath == "" {
		credentialsPath = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	}

	if credentialsPath != "" {
		// Load from file
		credentialsData, err := os.ReadFile(credentialsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read credentials file: %w", err)
		}

		credentials, err = google.CredentialsFromJSON(
			ctx,
			credentialsData,
			"https://www.googleapis.com/auth/cloud-platform",
		)
		if err != nil {
			return nil, fmt.Errorf("failed to parse credentials: %w", err)
		}

		// Extract project ID from credentials if not set
		if projectID == "" {
			var credJSON map[string]interface{}
			if err := json.Unmarshal(credentialsData, &credJSON); err == nil {
				if pid, ok := credJSON["project_id"].(string); ok {
					projectID = pid
				}
			}
		}
	} else {
		// Try Application Default Credentials
		credentials, err = google.FindDefaultCredentials(
			ctx,
			"https://www.googleapis.com/auth/cloud-platform",
		)
		if err != nil {
			return nil, fmt.Errorf("failed to find credentials: %w (set GOOGLE_APPLICATION_CREDENTIALS or configure gcloud)", err)
		}

		// Use credentials project ID if available
		if projectID == "" && credentials.ProjectID != "" {
			projectID = credentials.ProjectID
		}
	}

	if projectID == "" {
		return nil, fmt.Errorf("project_id is required for Vertex AI (set project_id parameter or VERTEXAI_PROJECT env var)")
	}

	// Build endpoint URL
	endpoint := fmt.Sprintf("https://%s-aiplatform.googleapis.com", location)
	if customEndpoint, ok := config.Parameters["endpoint"].(string); ok && customEndpoint != "" {
		endpoint = customEndpoint
	}

	provider := &VertexAIProvider{
		config:      config,
		credentials: credentials,
		projectID:   projectID,
		location:    location,
		endpoint:    endpoint,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		tokenProvider: NewTokenProvider(credentials),
		models:        getVertexAIModels(),
	}

	return provider, nil
}

// getVertexAIModels returns all available Vertex AI models
func getVertexAIModels() []ModelInfo {
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
		// Gemini 2.5 family (Google native)
		{
			Name:           "gemini-2.5-pro",
			Provider:       ProviderTypeVertexAI,
			ContextSize:    2097152, // 2M tokens
			MaxTokens:      8192,
			Capabilities:   visionCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Gemini 2.5 Pro via Vertex AI - Most capable model with 2M context",
		},
		{
			Name:           "gemini-2.5-flash",
			Provider:       ProviderTypeVertexAI,
			ContextSize:    1048576, // 1M tokens
			MaxTokens:      8192,
			Capabilities:   visionCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Gemini 2.5 Flash via Vertex AI - Fast and efficient with 1M context",
		},
		{
			Name:           "gemini-2.5-flash-lite",
			Provider:       ProviderTypeVertexAI,
			ContextSize:    1048576,
			MaxTokens:      8192,
			Capabilities:   visionCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Gemini 2.5 Flash Lite via Vertex AI - Cost-effective",
		},
		// Gemini 2.0 family
		{
			Name:           "gemini-2.0-flash-001",
			Provider:       ProviderTypeVertexAI,
			ContextSize:    1048576,
			MaxTokens:      8192,
			Capabilities:   visionCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Gemini 2.0 Flash via Vertex AI - Fast multimodal model",
		},
		{
			Name:           "gemini-2.0-flash",
			Provider:       ProviderTypeVertexAI,
			ContextSize:    1048576,
			MaxTokens:      8192,
			Capabilities:   visionCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Gemini 2.0 Flash (Latest) via Vertex AI",
		},
		// Gemini 1.5 family
		{
			Name:           "gemini-1.5-pro",
			Provider:       ProviderTypeVertexAI,
			ContextSize:    2097152, // 2M tokens
			MaxTokens:      8192,
			Capabilities:   visionCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Gemini 1.5 Pro via Vertex AI - Powerful with 2M context",
		},
		{
			Name:           "gemini-1.5-flash",
			Provider:       ProviderTypeVertexAI,
			ContextSize:    1048576,
			MaxTokens:      8192,
			Capabilities:   visionCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Gemini 1.5 Flash via Vertex AI - Fast and efficient",
		},
		{
			Name:           "gemini-1.5-flash-8b",
			Provider:       ProviderTypeVertexAI,
			ContextSize:    1048576,
			MaxTokens:      8192,
			Capabilities:   visionCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Gemini 1.5 Flash 8B via Vertex AI - Smaller, faster variant",
		},
		// Claude models via Model Garden
		{
			Name:           "claude-sonnet-4@20250514",
			Provider:       ProviderTypeVertexAI,
			ContextSize:    200000,
			MaxTokens:      50000,
			Capabilities:   visionCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Claude Sonnet 4 via Vertex AI Model Garden",
		},
		{
			Name:           "claude-opus-4@20250514",
			Provider:       ProviderTypeVertexAI,
			ContextSize:    200000,
			MaxTokens:      50000,
			Capabilities:   visionCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Claude Opus 4 via Vertex AI Model Garden",
		},
		{
			Name:           "claude-3-7-sonnet@20250219",
			Provider:       ProviderTypeVertexAI,
			ContextSize:    200000,
			MaxTokens:      50000,
			Capabilities:   visionCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Claude 3.7 Sonnet via Vertex AI Model Garden",
		},
		{
			Name:           "claude-3-5-sonnet-v2@20241022",
			Provider:       ProviderTypeVertexAI,
			ContextSize:    200000,
			MaxTokens:      8192,
			Capabilities:   visionCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Claude 3.5 Sonnet v2 via Vertex AI Model Garden",
		},
		// PaLM 2 models (legacy)
		{
			Name:           "text-bison@002",
			Provider:       ProviderTypeVertexAI,
			ContextSize:    8192,
			MaxTokens:      1024,
			Capabilities:   allCapabilities,
			SupportsTools:  false,
			SupportsVision: false,
			Description:    "PaLM 2 Text Bison via Vertex AI - Legacy text model",
		},
		{
			Name:           "chat-bison@002",
			Provider:       ProviderTypeVertexAI,
			ContextSize:    8192,
			MaxTokens:      1024,
			Capabilities:   allCapabilities,
			SupportsTools:  false,
			SupportsVision: false,
			Description:    "PaLM 2 Chat Bison via Vertex AI - Legacy chat model",
		},
	}
}

// GetType returns the provider type
func (vp *VertexAIProvider) GetType() ProviderType {
	return ProviderTypeVertexAI
}

// GetName returns the provider name
func (vp *VertexAIProvider) GetName() string {
	return "Vertex AI"
}

// GetModels returns available models
func (vp *VertexAIProvider) GetModels() []ModelInfo {
	return vp.models
}

// GetCapabilities returns provider capabilities
func (vp *VertexAIProvider) GetCapabilities() []ModelCapability {
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

// Generate generates a response using Vertex AI
func (vp *VertexAIProvider) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	startTime := time.Now()

	// Determine if this is a Claude model (Model Garden) or Gemini model
	if vp.isClaudeModel(request.Model) {
		return vp.generateClaude(ctx, request, startTime)
	}

	return vp.generateGemini(ctx, request, startTime)
}

// generateGemini handles Gemini model generation
func (vp *VertexAIProvider) generateGemini(ctx context.Context, request *LLMRequest, startTime time.Time) (*LLMResponse, error) {
	// Build request
	vertexReq, err := vp.buildVertexRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	// Get access token
	token, err := vp.tokenProvider.GetToken(ctx)
	if err != nil {
		return nil, err
	}

	// Build URL for Gemini
	url := fmt.Sprintf("%s/v1/projects/%s/locations/%s/publishers/google/models/%s:generateContent",
		vp.endpoint, vp.projectID, vp.location, request.Model)

	// Make request
	resp, err := vp.makeRequest(ctx, url, token, vertexReq)
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
	}

	return response, nil
}

// generateClaude handles Claude model generation via Model Garden
func (vp *VertexAIProvider) generateClaude(ctx context.Context, request *LLMRequest, startTime time.Time) (*LLMResponse, error) {
	// Build Claude request
	claudeReq := vp.buildClaudeRequest(request)

	// Get access token
	token, err := vp.tokenProvider.GetToken(ctx)
	if err != nil {
		return nil, err
	}

	// Build URL for Claude Model Garden
	url := fmt.Sprintf("%s/v1/projects/%s/locations/%s/publishers/anthropic/models/%s:rawPredict",
		vp.endpoint, vp.projectID, vp.location, request.Model)

	// Marshal request
	reqBody, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Make request
	httpResp, err := vp.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Handle errors
	if httpResp.StatusCode != http.StatusOK {
		return nil, handleVertexError(httpResp.StatusCode, respBody)
	}

	// Parse Claude response
	var claudeResp anthropicVertexResponse
	if err := json.Unmarshal(respBody, &claudeResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Build LLM response
	response := &LLMResponse{
		ID:             uuid.New(),
		RequestID:      request.ID,
		ProcessingTime: time.Since(startTime),
		CreatedAt:      time.Now(),
		FinishReason:   claudeResp.StopReason,
	}

	// Extract content
	for _, content := range claudeResp.Content {
		if content.Type == "text" {
			response.Content += content.Text
		}
	}

	// Add usage information
	response.Usage = Usage{
		PromptTokens:     claudeResp.Usage.InputTokens,
		CompletionTokens: claudeResp.Usage.OutputTokens,
		TotalTokens:      claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
	}

	return response, nil
}

// GenerateStream generates a streaming response
func (vp *VertexAIProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
	defer close(ch)

	// Determine if this is a Claude model or Gemini model
	if vp.isClaudeModel(request.Model) {
		// Claude streaming not yet implemented via Model Garden
		return fmt.Errorf("streaming not supported for Claude models via Model Garden")
	}

	return vp.generateGeminiStream(ctx, request, ch)
}

// generateGeminiStream handles streaming for Gemini models
func (vp *VertexAIProvider) generateGeminiStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
	// Build request
	vertexReq, err := vp.buildVertexRequest(request)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	// Get access token
	token, err := vp.tokenProvider.GetToken(ctx)
	if err != nil {
		return err
	}

	// Build streaming URL
	url := fmt.Sprintf("%s/v1/projects/%s/locations/%s/publishers/google/models/%s:streamGenerateContent?alt=sse",
		vp.endpoint, vp.projectID, vp.location, request.Model)

	// Marshal request
	reqBody, err := json.Marshal(vertexReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	httpReq.Header.Set("Accept", "text/event-stream")

	// Make request
	httpResp, err := vp.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return handleVertexError(httpResp.StatusCode, body)
	}

	// Parse SSE stream
	return vp.parseSSEStream(httpResp.Body, ch, request.ID)
}

// parseSSEStream parses Server-Sent Events stream
func (vp *VertexAIProvider) parseSSEStream(reader io.Reader, ch chan<- LLMResponse, requestID uuid.UUID) error {
	scanner := bufio.NewScanner(reader)
	var contentBuilder strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Parse JSON
		var streamResp vertexResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			log.Printf("Error parsing stream chunk: %v", err)
			continue
		}

		if len(streamResp.Candidates) == 0 {
			continue
		}

		candidate := streamResp.Candidates[0]

		// Extract text
		for _, part := range candidate.Content.Parts {
			if p, ok := part.(map[string]interface{}); ok {
				if text, ok := p["text"].(string); ok {
					contentBuilder.WriteString(text)

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

		// Check for completion
		if candidate.FinishReason != "" {
			finalResponse := LLMResponse{
				ID:           uuid.New(),
				RequestID:    requestID,
				Content:      contentBuilder.String(),
				FinishReason: candidate.FinishReason,
				CreatedAt:    time.Now(),
			}

			if streamResp.UsageMetadata != nil {
				finalResponse.Usage = Usage{
					PromptTokens:     streamResp.UsageMetadata.PromptTokenCount,
					CompletionTokens: streamResp.UsageMetadata.CandidatesTokenCount,
					TotalTokens:      streamResp.UsageMetadata.TotalTokenCount,
				}
			}

			ch <- finalResponse
		}
	}

	return scanner.Err()
}

// buildVertexRequest constructs a Vertex AI request for Gemini models
func (vp *VertexAIProvider) buildVertexRequest(request *LLMRequest) (*vertexRequest, error) {
	req := &vertexRequest{
		GenerationConfig: &vertexGenConfig{
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
	systemMsg, contents := vp.convertMessages(request.Messages)
	req.Contents = contents

	// Set system instruction if present
	if systemMsg != "" {
		req.SystemInstruction = &vertexContent{
			Parts: []vertexPart{
				map[string]interface{}{"text": systemMsg},
			},
		}
	}

	// Convert tools
	if len(request.Tools) > 0 {
		req.Tools = vp.convertTools(request.Tools)
		req.ToolConfig = &vertexToolConfig{
			FunctionCallingConfig: &vertexFunctionCallingConfig{
				Mode: "AUTO",
			},
		}
	}

	// Set safety settings to permissive for development use
	req.SafetySettings = []vertexSafetySetting{
		{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_ONLY_HIGH"},
		{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_ONLY_HIGH"},
		{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_ONLY_HIGH"},
		{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_ONLY_HIGH"},
	}

	return req, nil
}

// buildClaudeRequest constructs a request for Claude models via Model Garden
func (vp *VertexAIProvider) buildClaudeRequest(request *LLMRequest) *anthropicVertexRequest {
	req := &anthropicVertexRequest{
		AnthropicVersion: "vertex-2023-10-16",
		MaxTokens:        request.MaxTokens,
		Temperature:      request.Temperature,
		TopP:             request.TopP,
		Stream:           false,
	}

	// Default max tokens if not specified
	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}

	// Convert messages
	var systemMsg string
	for _, msg := range request.Messages {
		if msg.Role == "system" {
			systemMsg = msg.Content
		} else {
			req.Messages = append(req.Messages, anthropicVertexMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	if systemMsg != "" {
		req.System = systemMsg
	}

	return req
}

// convertMessages converts LLM messages to Vertex AI format
func (vp *VertexAIProvider) convertMessages(messages []Message) (string, []vertexContent) {
	var systemMsg string
	var contents []vertexContent

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			systemMsg = msg.Content
		case "user":
			contents = append(contents, vertexContent{
				Role: "user",
				Parts: []vertexPart{
					map[string]interface{}{"text": msg.Content},
				},
			})
		case "assistant":
			contents = append(contents, vertexContent{
				Role: "model", // Vertex AI uses "model" instead of "assistant"
				Parts: []vertexPart{
					map[string]interface{}{"text": msg.Content},
				},
			})
		}
	}

	return systemMsg, contents
}

// convertTools converts LLM tools to Vertex AI format
func (vp *VertexAIProvider) convertTools(tools []Tool) []vertexTool {
	declarations := make([]vertexFunctionDeclaration, len(tools))

	for i, tool := range tools {
		declarations[i] = vertexFunctionDeclaration{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			Parameters:  tool.Function.Parameters,
		}
	}

	return []vertexTool{
		{FunctionDeclarations: declarations},
	}
}

// makeRequest makes a non-streaming API request to Vertex AI
func (vp *VertexAIProvider) makeRequest(ctx context.Context, url, token string, request *vertexRequest) (*vertexResponse, error) {
	// Marshal request
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Make request
	httpResp, err := vp.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Handle errors
	if httpResp.StatusCode != http.StatusOK {
		return nil, handleVertexError(httpResp.StatusCode, respBody)
	}

	// Parse successful response
	var response vertexResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// handleVertexError handles Vertex AI API errors
func handleVertexError(statusCode int, body []byte) error {
	var vertexErr vertexAIError
	if err := json.Unmarshal(body, &vertexErr); err == nil {
		errInfo := vertexErr.Error

		switch errInfo.Code {
		case 400: // INVALID_ARGUMENT
			return ErrInvalidRequest
		case 401: // UNAUTHENTICATED
			return fmt.Errorf("authentication failed: %s", errInfo.Message)
		case 403: // PERMISSION_DENIED
			return fmt.Errorf("permission denied: %s - check service account permissions", errInfo.Message)
		case 404: // NOT_FOUND
			return ErrModelNotFound
		case 429: // RESOURCE_EXHAUSTED
			return ErrRateLimited
		case 503: // UNAVAILABLE
			return fmt.Errorf("vertex AI service unavailable: %s", errInfo.Message)
		case 504: // DEADLINE_EXCEEDED
			return fmt.Errorf("request timeout: %s", errInfo.Message)
		default:
			return fmt.Errorf("vertex AI error (%d - %s): %s",
				errInfo.Code, errInfo.Status, errInfo.Message)
		}
	}

	// Fallback to HTTP status code
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("unauthorized: check credentials")
	case http.StatusForbidden:
		return fmt.Errorf("forbidden: check service account permissions")
	case http.StatusNotFound:
		return ErrModelNotFound
	case http.StatusTooManyRequests:
		return ErrRateLimited
	default:
		return fmt.Errorf("vertex AI API error (%d): %s", statusCode, string(body))
	}
}

// isClaudeModel checks if a model is a Claude model via Model Garden
func (vp *VertexAIProvider) isClaudeModel(model string) bool {
	return strings.HasPrefix(model, "claude-")
}

// IsAvailable checks if the provider is available
func (vp *VertexAIProvider) IsAvailable(ctx context.Context) bool {
	return vp.credentials != nil && vp.projectID != ""
}

// GetHealth returns the health status of the provider
func (vp *VertexAIProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	startTime := time.Now()

	health := &ProviderHealth{
		LastCheck:  time.Now(),
		ModelCount: len(vp.models),
	}

	// Test with a minimal request using Gemini Flash Lite
	testReq := &LLMRequest{
		ID:          uuid.New(),
		Model:       "gemini-2.5-flash-lite",
		Messages:    []Message{{Role: "user", Content: "Hi"}},
		MaxTokens:   10,
		Temperature: 0.1,
	}

	_, err := vp.Generate(ctx, testReq)
	if err != nil {
		health.Status = "unhealthy"
		health.ErrorCount = 1
		return health, err
	}

	health.Status = "healthy"
	health.Latency = time.Since(startTime)
	vp.lastHealth = health

	return health, nil
}

// Close closes the provider and cleans up resources
func (vp *VertexAIProvider) Close() error {
	vp.httpClient.CloseIdleConnections()
	log.Printf("Vertex AI provider closed")
	return nil
}
