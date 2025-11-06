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

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/google/uuid"
)

// AzureProvider implements the Provider interface for Azure OpenAI Service
type AzureProvider struct {
	config             ProviderConfigEntry
	apiKey             string
	endpoint           string         // e.g., https://myresource.openai.azure.com
	apiVersion         string         // e.g., 2025-04-01-preview
	deploymentMap      map[string]string // model name -> deployment name
	httpClient         *http.Client
	models             []ModelInfo
	entraTokenProvider *EntraTokenProvider // for Entra ID auth
	lastHealth         *ProviderHealth
}

// EntraTokenProvider handles Microsoft Entra ID authentication with token caching
type EntraTokenProvider struct {
	credential  azcore.TokenCredential
	tokenCache  *string
	tokenExpiry time.Time
	mutex       sync.RWMutex
}

// Azure API structures - OpenAI-compatible with Azure-specific extensions

type azureRequest struct {
	Model         string                 `json:"model,omitempty"` // Not used in URL, but can be in body
	Messages      []azureMessage         `json:"messages"`
	MaxTokens     int                    `json:"max_tokens,omitempty"`
	Temperature   float64                `json:"temperature,omitempty"`
	TopP          float64                `json:"top_p,omitempty"`
	Stream        bool                   `json:"stream,omitempty"`
	Tools         []azureTool            `json:"tools,omitempty"`
	ToolChoice    interface{}            `json:"tool_choice,omitempty"`
	StopSequences []string               `json:"stop,omitempty"`
}

type azureMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

type azureTool struct {
	Type     string                 `json:"type"`
	Function azureFunctionDefinition `json:"function"`
}

type azureFunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type azureResponse struct {
	ID                  string                  `json:"id"`
	Object              string                  `json:"object"`
	Created             int64                   `json:"created"`
	Model               string                  `json:"model"`
	Choices             []azureChoice           `json:"choices"`
	Usage               azureUsage              `json:"usage"`
	PromptFilterResults []ContentFilterResult   `json:"prompt_filter_results,omitempty"` // Azure-specific
	SystemFingerprint   string                  `json:"system_fingerprint,omitempty"`
}

type azureChoice struct {
	Index                int                   `json:"index"`
	Message              azureMessage          `json:"message"`
	FinishReason         string                `json:"finish_reason"`
	ContentFilterResults *ContentFilterDetails `json:"content_filter_results,omitempty"` // Azure-specific
}

type azureUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Azure-specific content filtering structures
type ContentFilterResult struct {
	PromptIndex          int                   `json:"prompt_index"`
	ContentFilterResults ContentFilterDetails  `json:"content_filter_results"`
}

type ContentFilterDetails struct {
	Hate     FilterCategory `json:"hate"`
	SelfHarm FilterCategory `json:"self_harm"`
	Sexual   FilterCategory `json:"sexual"`
	Violence FilterCategory `json:"violence"`
}

type FilterCategory struct {
	Filtered bool   `json:"filtered"`
	Severity string `json:"severity"` // "safe", "low", "medium", "high"
}

// Streaming structures
type azureStreamChunk struct {
	ID      string              `json:"id"`
	Object  string              `json:"object"`
	Created int64               `json:"created"`
	Model   string              `json:"model"`
	Choices []azureStreamChoice `json:"choices"`
}

type azureStreamChoice struct {
	Index                int                   `json:"index"`
	Delta                azureDelta            `json:"delta"`
	FinishReason         string                `json:"finish_reason,omitempty"`
	ContentFilterResults *ContentFilterDetails `json:"content_filter_results,omitempty"`
}

type azureDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// Azure error structure
type azureError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Type    string `json:"type"`
		Param   string `json:"param,omitempty"`
	} `json:"error"`
}

// NewEntraTokenProvider creates a new Entra ID token provider
func NewEntraTokenProvider(credential azcore.TokenCredential) *EntraTokenProvider {
	return &EntraTokenProvider{
		credential: credential,
	}
}

// GetToken retrieves a cached or new Entra ID token
func (etp *EntraTokenProvider) GetToken(ctx context.Context) (string, error) {
	// Check cache first (read lock)
	etp.mutex.RLock()
	if etp.tokenCache != nil && time.Now().Before(etp.tokenExpiry) {
		token := *etp.tokenCache
		etp.mutex.RUnlock()
		return token, nil
	}
	etp.mutex.RUnlock()

	// Acquire write lock to refresh token
	etp.mutex.Lock()
	defer etp.mutex.Unlock()

	// Double-check after acquiring write lock (another goroutine may have refreshed)
	if etp.tokenCache != nil && time.Now().Before(etp.tokenExpiry) {
		return *etp.tokenCache, nil
	}

	// Get new token from Azure
	tokenResp, err := etp.credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://cognitiveservices.azure.com/.default"},
	})
	if err != nil {
		return "", fmt.Errorf("failed to get Entra ID token: %w", err)
	}

	token := tokenResp.Token
	etp.tokenCache = &token
	etp.tokenExpiry = tokenResp.ExpiresOn.Add(-5 * time.Minute) // Refresh 5 min early

	return token, nil
}

// NewAzureProvider creates a new Azure OpenAI provider
func NewAzureProvider(config ProviderConfigEntry) (*AzureProvider, error) {
	// Get endpoint (required)
	endpoint, ok := config.Parameters["endpoint"].(string)
	if !ok || endpoint == "" {
		endpoint = os.Getenv("AZURE_OPENAI_ENDPOINT")
	}
	if endpoint == "" {
		return nil, fmt.Errorf("azure endpoint is required (set in config or AZURE_OPENAI_ENDPOINT env var)")
	}

	// Get API version (with default)
	apiVersion, ok := config.Parameters["api_version"].(string)
	if !ok || apiVersion == "" {
		apiVersion = os.Getenv("AZURE_API_VERSION")
	}
	if apiVersion == "" {
		apiVersion = "2025-04-01-preview" // Latest stable version
	}

	// Load deployment map
	deploymentMap, err := loadDeploymentMap(config)
	if err != nil {
		return nil, fmt.Errorf("failed to load deployment map: %w", err)
	}

	provider := &AzureProvider{
		config:        config,
		endpoint:      endpoint,
		apiVersion:    apiVersion,
		deploymentMap: deploymentMap,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		models: getAzureModels(),
	}

	// Configure authentication
	if err := provider.configureAuth(config); err != nil {
		return nil, fmt.Errorf("failed to configure authentication: %w", err)
	}

	log.Printf("✅ Azure OpenAI provider initialized: endpoint=%s, api_version=%s, deployments=%d",
		endpoint, apiVersion, len(deploymentMap))

	return provider, nil
}

// configureAuth sets up API key or Entra ID authentication
func (ap *AzureProvider) configureAuth(config ProviderConfigEntry) error {
	// Check if Entra ID authentication is requested
	useEntraID, _ := config.Parameters["use_entra_id"].(bool)

	if useEntraID {
		// Try to create Entra ID credential
		var credential azcore.TokenCredential
		var err error

		// Check for managed identity
		useManagedIdentity, _ := config.Parameters["managed_identity"].(bool)
		if useManagedIdentity {
			clientID, _ := config.Parameters["managed_identity_client_id"].(string)
			if clientID != "" {
				// User-assigned managed identity
				credential, err = azidentity.NewManagedIdentityCredential(&azidentity.ManagedIdentityCredentialOptions{
					ID: azidentity.ClientID(clientID),
				})
			} else {
				// System-assigned managed identity
				credential, err = azidentity.NewManagedIdentityCredential(nil)
			}
		} else {
			// Use default Azure credential (env vars, CLI, etc.)
			credential, err = azidentity.NewDefaultAzureCredential(nil)
		}

		if err != nil {
			return fmt.Errorf("failed to create Azure credential: %w", err)
		}

		ap.entraTokenProvider = NewEntraTokenProvider(credential)
		log.Printf("✅ Azure provider using Entra ID authentication")
		return nil
	}

	// Fall back to API key authentication
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("AZURE_OPENAI_API_KEY")
	}

	if apiKey == "" {
		return fmt.Errorf("azure API key not provided (set in config, AZURE_OPENAI_API_KEY env var, or use Entra ID)")
	}

	ap.apiKey = apiKey
	log.Printf("✅ Azure provider using API key authentication")
	return nil
}

// loadDeploymentMap loads the deployment mapping from config
func loadDeploymentMap(config ProviderConfigEntry) (map[string]string, error) {
	deploymentMapParam := config.Parameters["deployment_map"]

	if deploymentMapParam == nil {
		// Try environment variable
		if envMap := os.Getenv("AZURE_DEPLOYMENTS_MAP"); envMap != "" {
			deploymentMapParam = envMap
		} else {
			// Return empty map (will fall back to using model name as deployment name)
			return make(map[string]string), nil
		}
	}

	switch v := deploymentMapParam.(type) {
	case string:
		// Check if it's a file path
		if strings.HasSuffix(v, ".json") {
			data, err := os.ReadFile(v)
			if err != nil {
				return nil, fmt.Errorf("failed to read deployment map file: %w", err)
			}
			var m map[string]string
			if err := json.Unmarshal(data, &m); err != nil {
				return nil, fmt.Errorf("failed to parse deployment map file: %w", err)
			}
			return m, nil
		}

		// Try to parse as JSON string
		var m map[string]string
		if err := json.Unmarshal([]byte(v), &m); err != nil {
			return nil, fmt.Errorf("failed to parse deployment map JSON: %w", err)
		}
		return m, nil

	case map[string]interface{}:
		// Convert to map[string]string
		m := make(map[string]string)
		for k, val := range v {
			if strVal, ok := val.(string); ok {
				m[k] = strVal
			}
		}
		return m, nil

	case map[string]string:
		return v, nil

	default:
		return make(map[string]string), nil
	}
}

// getAzureModels returns all available Azure OpenAI models
func getAzureModels() []ModelInfo {
	allCapabilities := []ModelCapability{
		CapabilityTextGeneration,
		CapabilityCodeGeneration,
		CapabilityCodeAnalysis,
		CapabilityPlanning,
		CapabilityDebugging,
		CapabilityRefactoring,
		CapabilityTesting,
	}

	return []ModelInfo{
		// GPT-4 family
		{
			Name:           "gpt-4-turbo",
			Provider:       ProviderTypeAzure,
			ContextSize:    128000,
			MaxTokens:      4096,
			Capabilities:   allCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "GPT-4 Turbo via Azure - Latest GPT-4 model",
		},
		{
			Name:           "gpt-4",
			Provider:       ProviderTypeAzure,
			ContextSize:    8192,
			MaxTokens:      4096,
			Capabilities:   allCapabilities,
			SupportsTools:  true,
			SupportsVision: false,
			Description:    "GPT-4 via Azure - Powerful reasoning",
		},
		{
			Name:           "gpt-4-32k",
			Provider:       ProviderTypeAzure,
			ContextSize:    32768,
			MaxTokens:      4096,
			Capabilities:   allCapabilities,
			SupportsTools:  true,
			SupportsVision: false,
			Description:    "GPT-4 32K via Azure - Extended context window",
		},
		{
			Name:           "gpt-4-vision-preview",
			Provider:       ProviderTypeAzure,
			ContextSize:    128000,
			MaxTokens:      4096,
			Capabilities:   allCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "GPT-4 Vision via Azure - Multimodal capabilities",
		},
		{
			Name:           "gpt-4o",
			Provider:       ProviderTypeAzure,
			ContextSize:    128000,
			MaxTokens:      4096,
			Capabilities:   allCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "GPT-4o via Azure - Latest multimodal model",
		},
		{
			Name:           "gpt-4o-mini",
			Provider:       ProviderTypeAzure,
			ContextSize:    128000,
			MaxTokens:      4096,
			Capabilities:   allCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "GPT-4o Mini via Azure - Fast and efficient",
		},
		// GPT-3.5 family
		{
			Name:           "gpt-35-turbo",
			Provider:       ProviderTypeAzure,
			ContextSize:    16385,
			MaxTokens:      4096,
			Capabilities:   allCapabilities,
			SupportsTools:  true,
			SupportsVision: false,
			Description:    "GPT-3.5 Turbo via Azure - Fast and cost-effective",
		},
		{
			Name:           "gpt-35-turbo-16k",
			Provider:       ProviderTypeAzure,
			ContextSize:    16385,
			MaxTokens:      4096,
			Capabilities:   allCapabilities,
			SupportsTools:  true,
			SupportsVision: false,
			Description:    "GPT-3.5 Turbo 16K via Azure",
		},
		// o1 reasoning models
		{
			Name:           "o1-preview",
			Provider:       ProviderTypeAzure,
			ContextSize:    128000,
			MaxTokens:      32768,
			Capabilities:   allCapabilities,
			SupportsTools:  false,
			SupportsVision: false,
			Description:    "o1 Preview via Azure - Advanced reasoning model",
		},
		{
			Name:           "o1-mini",
			Provider:       ProviderTypeAzure,
			ContextSize:    128000,
			MaxTokens:      16384,
			Capabilities:   allCapabilities,
			SupportsTools:  false,
			SupportsVision: false,
			Description:    "o1 Mini via Azure - Faster reasoning model",
		},
		// Embedding models
		{
			Name:           "text-embedding-3-large",
			Provider:       ProviderTypeAzure,
			ContextSize:    8191,
			MaxTokens:      0,
			Capabilities:   []ModelCapability{CapabilityTextGeneration},
			SupportsTools:  false,
			SupportsVision: false,
			Description:    "Text Embedding 3 Large via Azure",
		},
		{
			Name:           "text-embedding-ada-002",
			Provider:       ProviderTypeAzure,
			ContextSize:    8191,
			MaxTokens:      0,
			Capabilities:   []ModelCapability{CapabilityTextGeneration},
			SupportsTools:  false,
			SupportsVision: false,
			Description:    "Ada-002 Embedding via Azure",
		},
	}
}

// GetType returns the provider type
func (ap *AzureProvider) GetType() ProviderType {
	return ProviderTypeAzure
}

// GetName returns the provider name
func (ap *AzureProvider) GetName() string {
	return "Azure OpenAI"
}

// GetModels returns available models
func (ap *AzureProvider) GetModels() []ModelInfo {
	return ap.models
}

// GetCapabilities returns provider capabilities
func (ap *AzureProvider) GetCapabilities() []ModelCapability {
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

// resolveDeployment maps a model name to an Azure deployment name
func (ap *AzureProvider) resolveDeployment(modelName string) string {
	// Check explicit mapping
	if deployment, ok := ap.deploymentMap[modelName]; ok {
		return deployment
	}

	// Fallback: use model name as deployment name
	// (works if deployment name matches model name)
	return modelName
}

// getAuthHeader returns the appropriate authentication header value
func (ap *AzureProvider) getAuthHeader(ctx context.Context) (string, error) {
	if ap.entraTokenProvider != nil {
		token, err := ap.entraTokenProvider.GetToken(ctx)
		if err != nil {
			return "", err
		}
		return "Bearer " + token, nil
	}
	return ap.apiKey, nil
}

// Generate generates a response using Azure OpenAI
func (ap *AzureProvider) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	startTime := time.Now()

	// Resolve deployment name
	deployment := ap.resolveDeployment(request.Model)

	// Build Azure request
	azureReq := ap.buildAzureRequest(request)

	// Create HTTP request
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		ap.endpoint, deployment, ap.apiVersion)

	reqBody, err := json.Marshal(azureReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	authValue, err := ap.getAuthHeader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth header: %w", err)
	}

	if strings.HasPrefix(authValue, "Bearer ") {
		httpReq.Header.Set("Authorization", authValue)
	} else {
		httpReq.Header.Set("api-key", authValue)
	}

	// Make request
	httpResp, err := ap.httpClient.Do(httpReq)
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
		return nil, handleAzureError(httpResp.StatusCode, respBody)
	}

	// Parse response
	response, err := ap.parseAzureResponse(respBody, request.ID, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response, nil
}

// GenerateStream generates a streaming response
func (ap *AzureProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
	defer close(ch)

	// Resolve deployment name
	deployment := ap.resolveDeployment(request.Model)

	// Build Azure request with streaming enabled
	azureReq := ap.buildAzureRequest(request)
	azureReq.Stream = true

	// Create HTTP request
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		ap.endpoint, deployment, ap.apiVersion)

	reqBody, err := json.Marshal(azureReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	authValue, err := ap.getAuthHeader(ctx)
	if err != nil {
		return fmt.Errorf("failed to get auth header: %w", err)
	}

	if strings.HasPrefix(authValue, "Bearer ") {
		httpReq.Header.Set("Authorization", authValue)
	} else {
		httpReq.Header.Set("api-key", authValue)
	}

	// Make request
	httpResp, err := ap.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return handleAzureError(httpResp.StatusCode, body)
	}

	// Parse SSE stream
	return ap.parseSSEStream(httpResp.Body, ch, request.ID)
}

// buildAzureRequest builds an Azure OpenAI request
func (ap *AzureProvider) buildAzureRequest(request *LLMRequest) *azureRequest {
	req := &azureRequest{
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
	for _, msg := range request.Messages {
		req.Messages = append(req.Messages, azureMessage{
			Role:    msg.Role,
			Content: msg.Content,
			Name:    msg.Name,
		})
	}

	// Convert tools
	if len(request.Tools) > 0 {
		for _, tool := range request.Tools {
			req.Tools = append(req.Tools, azureTool{
				Type: tool.Type,
				Function: azureFunctionDefinition{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters,
				},
			})
		}
	}

	return req
}

// parseAzureResponse parses an Azure OpenAI response
func (ap *AzureProvider) parseAzureResponse(body []byte, requestID uuid.UUID, startTime time.Time) (*LLMResponse, error) {
	var azureResp azureResponse
	if err := json.Unmarshal(body, &azureResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for content filtering on prompts
	for _, filterResult := range azureResp.PromptFilterResults {
		filters := filterResult.ContentFilterResults
		if filters.Hate.Filtered || filters.SelfHarm.Filtered ||
			filters.Sexual.Filtered || filters.Violence.Filtered {
			return nil, fmt.Errorf("content filtered by Azure: prompt contains prohibited content (hate=%v, self_harm=%v, sexual=%v, violence=%v)",
				filters.Hate.Severity, filters.SelfHarm.Severity, filters.Sexual.Severity, filters.Violence.Severity)
		}
	}

	if len(azureResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := azureResp.Choices[0]

	// Check for content filtering on completion
	if choice.ContentFilterResults != nil {
		filters := choice.ContentFilterResults
		if filters.Hate.Filtered || filters.SelfHarm.Filtered ||
			filters.Sexual.Filtered || filters.Violence.Filtered {
			return nil, fmt.Errorf("content filtered by Azure: completion contains prohibited content (hate=%v, self_harm=%v, sexual=%v, violence=%v)",
				filters.Hate.Severity, filters.SelfHarm.Severity, filters.Sexual.Severity, filters.Violence.Severity)
		}
	}

	response := &LLMResponse{
		ID:             uuid.New(),
		RequestID:      requestID,
		Content:        choice.Message.Content,
		ProcessingTime: time.Since(startTime),
		CreatedAt:      time.Now(),
		Usage: Usage{
			PromptTokens:     azureResp.Usage.PromptTokens,
			CompletionTokens: azureResp.Usage.CompletionTokens,
			TotalTokens:      azureResp.Usage.TotalTokens,
		},
		FinishReason: choice.FinishReason,
	}

	return response, nil
}

// parseSSEStream parses a Server-Sent Events stream
func (ap *AzureProvider) parseSSEStream(reader io.Reader, ch chan<- LLMResponse, requestID uuid.UUID) error {
	scanner := bufio.NewScanner(reader)
	var contentBuilder strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Parse SSE data
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Check for stream end
		if data == "[DONE]" {
			break
		}

		// Parse JSON chunk
		var chunk azureStreamChunk
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

			// Send incremental response
			ch <- LLMResponse{
				ID:        uuid.New(),
				RequestID: requestID,
				Content:   delta,
				CreatedAt: time.Now(),
			}
		}

		// Check for completion
		if chunk.Choices[0].FinishReason != "" {
			ch <- LLMResponse{
				ID:           uuid.New(),
				RequestID:    requestID,
				Content:      contentBuilder.String(),
				FinishReason: chunk.Choices[0].FinishReason,
				CreatedAt:    time.Now(),
			}
		}
	}

	return scanner.Err()
}

// handleAzureError converts Azure API errors to provider errors
func handleAzureError(statusCode int, body []byte) error {
	var azureErr azureError
	if err := json.Unmarshal(body, &azureErr); err == nil {
		// Azure-specific error codes
		switch azureErr.Error.Code {
		case "content_filter":
			return fmt.Errorf("content filtered by Azure: %s", azureErr.Error.Message)
		case "DeploymentNotFound":
			return ErrModelNotFound
		case "InvalidRequestError":
			return ErrInvalidRequest
		case "RateLimitError", "429":
			return ErrRateLimited
		case "QuotaExceeded":
			return fmt.Errorf("azure quota exceeded: %s", azureErr.Error.Message)
		case "InvalidApiKey", "Unauthorized":
			return fmt.Errorf("invalid Azure API key or Entra ID token")
		default:
			return fmt.Errorf("azure API error (%s): %s", azureErr.Error.Code, azureErr.Error.Message)
		}
	}

	// Fallback to HTTP status codes
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("unauthorized: check API key or Entra ID token")
	case http.StatusForbidden:
		return fmt.Errorf("forbidden: check resource access permissions")
	case http.StatusNotFound:
		return ErrModelNotFound
	case http.StatusTooManyRequests:
		return ErrRateLimited
	case http.StatusBadRequest:
		return ErrInvalidRequest
	default:
		return fmt.Errorf("azure API error (%d): %s", statusCode, string(body))
	}
}

// IsAvailable checks if the provider is available
func (ap *AzureProvider) IsAvailable(ctx context.Context) bool {
	return ap.apiKey != "" || ap.entraTokenProvider != nil
}

// GetHealth returns the health status of the provider
func (ap *AzureProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	startTime := time.Now()

	health := &ProviderHealth{
		LastCheck:  time.Now(),
		ModelCount: len(ap.models),
	}

	// Test with minimal request
	testReq := &LLMRequest{
		ID:          uuid.New(),
		Model:       "gpt-35-turbo",
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
func (ap *AzureProvider) Close() error {
	ap.httpClient.CloseIdleConnections()
	log.Printf("Azure OpenAI provider closed")
	return nil
}
