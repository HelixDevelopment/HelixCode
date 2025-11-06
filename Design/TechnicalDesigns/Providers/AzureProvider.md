# Azure OpenAI Provider - Technical Design

## Overview

The Azure OpenAI Provider enables access to OpenAI models through Microsoft Azure OpenAI Service. This provider supports all OpenAI models (GPT-4, GPT-3.5, embeddings, etc.) deployed on Azure infrastructure with enterprise features including Microsoft Entra ID authentication, deployment-based routing, API versioning, content filtering, and managed identity support.

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      AzureProvider                              │
├─────────────────────────────────────────────────────────────────┤
│  - config: ProviderConfigEntry                                  │
│  - apiKey: string                                               │
│  - endpoint: string (Azure resource endpoint)                   │
│  - apiVersion: string                                           │
│  - deploymentMap: map[string]string (model -> deployment)       │
│  - httpClient: *http.Client                                     │
│  - models: []ModelInfo                                          │
│  - entraTokenProvider: *EntraTokenProvider (optional)           │
├─────────────────────────────────────────────────────────────────┤
│  Methods:                                                       │
│  + NewAzureProvider(config) (*AzureProvider, error)             │
│  + Generate(ctx, request) (*LLMResponse, error)                 │
│  + GenerateStream(ctx, request, ch) error                       │
│  + resolveDeployment(modelName) string                          │
│  + getAuthHeader(ctx) (string, error)                           │
│  + GetType() ProviderType                                       │
│  + GetName() string                                             │
│  + GetModels() []ModelInfo                                      │
│  + GetCapabilities() []ModelCapability                          │
│  + IsAvailable(ctx) bool                                        │
│  + GetHealth(ctx) (*ProviderHealth, error)                      │
│  + Close() error                                                │
└─────────────────────────────────────────────────────────────────┘
              │
              │ Uses Azure OpenAI API (OpenAI-compatible)
              ▼
┌─────────────────────────────────────────────────────────────────┐
│              Azure OpenAI Service API                           │
├─────────────────────────────────────────────────────────────────┤
│  - Deployment-based routing                                     │
│  - /deployments/{deployment-id}/chat/completions                │
│  - /deployments/{deployment-id}/completions                     │
│  - /deployments/{deployment-id}/embeddings                      │
│  - API versioning (e.g., 2025-04-01-preview)                    │
│  - Content filtering                                            │
└─────────────────────────────────────────────────────────────────┘
```

### Component Breakdown

**AzureProvider Struct**:
```go
type AzureProvider struct {
    config              ProviderConfigEntry
    apiKey              string
    endpoint            string         // e.g., https://myresource.openai.azure.com
    apiVersion          string         // e.g., 2025-04-01-preview
    deploymentMap       map[string]string  // model name -> deployment name
    httpClient          *http.Client
    models              []ModelInfo
    entraTokenProvider  *EntraTokenProvider  // for Entra ID auth
    lastHealth          *ProviderHealth
}

type EntraTokenProvider struct {
    credential    azidentity.TokenCredential
    tokenCache    *string
    tokenExpiry   time.Time
    mutex         sync.RWMutex
}
```

**Azure-Specific Configuration**:
- Resource endpoint (required)
- API key or Entra ID authentication
- API version (defaults to latest stable)
- Deployment mappings (model -> deployment name)
- Content filtering configuration
- Managed identity support

## Interface Implementation

Implements the `Provider` interface from `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/provider.go`:

```go
type Provider interface {
    GetType() ProviderType
    GetName() string
    GetModels() []ModelInfo
    GetCapabilities() []ModelCapability
    Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error)
    GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error
    IsAvailable(ctx context.Context) bool
    GetHealth(ctx context.Context) (*ProviderHealth, error)
    Close() error
}
```

## Model Definitions

### Supported Model Families

**1. GPT-4 Family**
```go
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
    Name:           "gpt-4-32k",
    Provider:       ProviderTypeAzure,
    ContextSize:    32768,
    MaxTokens:      4096,
    Capabilities:   allCapabilities,
    SupportsTools:  true,
    SupportsVision: false,
    Description:    "GPT-4 32K via Azure - Extended context window",
},
```

**2. GPT-3.5 Family**
```go
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
```

**3. o1 Reasoning Models**
```go
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
```

**4. Embedding Models**
```go
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
```

## Request/Response Flow

### Non-Streaming Flow

```
Client Request
    │
    ▼
AzureProvider.Generate(ctx, request)
    │
    ├─> resolveDeployment(request.Model)
    │   └─> Map model name to Azure deployment name
    │       Example: "gpt-4-turbo" -> "my-gpt4-deployment"
    │
    ├─> buildAzureRequest(request)
    │   └─> Convert to OpenAI-compatible format
    │       (Azure uses same format as OpenAI)
    │
    ├─> getAuthHeader(ctx)
    │   ├─> API Key: "api-key: {key}"
    │   └─> Entra ID: "Authorization: Bearer {token}"
    │
    ├─> POST https://{endpoint}/openai/deployments/{deployment}/chat/completions?api-version={version}
    │   Headers:
    │     - Content-Type: application/json
    │     - api-key: {key} OR Authorization: Bearer {token}
    │
    ├─> parseAzureResponse(responseBody)
    │   └─> Convert from OpenAI format to LLMResponse
    │       + Extract content filtering results
    │
    └─> Return LLMResponse
```

### Streaming Flow

```
Client Request
    │
    ▼
AzureProvider.GenerateStream(ctx, request, ch)
    │
    ├─> resolveDeployment(request.Model)
    │
    ├─> buildAzureRequest(request) + set stream=true
    │
    ├─> POST https://{endpoint}/openai/deployments/{deployment}/chat/completions?api-version={version}
    │   Headers:
    │     - Accept: text/event-stream
    │     - api-key: {key} OR Authorization: Bearer {token}
    │
    ├─> Parse SSE stream
    │   └─> For each "data: {json}" line:
    │       ├─> Parse JSON chunk
    │       ├─> Extract delta content
    │       ├─> Send to channel
    │       └─> data: [DONE] -> complete
    │
    └─> Close channel
```

## Authentication Mechanisms

### 1. API Key Authentication (Simplest)

```go
func NewAzureProvider(config ProviderConfigEntry) (*AzureProvider, error) {
    apiKey := config.APIKey
    if apiKey == "" {
        apiKey = os.Getenv("AZURE_OPENAI_API_KEY")
    }

    if apiKey == "" {
        return nil, fmt.Errorf("azure openai API key not provided")
    }

    endpoint := config.Parameters["endpoint"].(string)
    if endpoint == "" {
        return nil, fmt.Errorf("azure endpoint is required")
    }

    provider := &AzureProvider{
        config:     config,
        apiKey:     apiKey,
        endpoint:   endpoint,
        apiVersion: getAPIVersion(config),
        httpClient: &http.Client{Timeout: 120 * time.Second},
        models:     getAzureModels(),
    }

    return provider, nil
}

func (ap *AzureProvider) getAuthHeader(ctx context.Context) (string, error) {
    return ap.apiKey, nil
}

// Usage in request:
httpReq.Header.Set("api-key", apiKey)
```

### 2. Microsoft Entra ID Authentication (Recommended for Production)

```go
import (
    "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

type EntraTokenProvider struct {
    credential    azidentity.TokenCredential
    tokenCache    *string
    tokenExpiry   time.Time
    mutex         sync.RWMutex
}

func NewEntraTokenProvider(credential azidentity.TokenCredential) *EntraTokenProvider {
    return &EntraTokenProvider{
        credential: credential,
    }
}

func (etp *EntraTokenProvider) GetToken(ctx context.Context) (string, error) {
    etp.mutex.RLock()
    if etp.tokenCache != nil && time.Now().Before(etp.tokenExpiry) {
        token := *etp.tokenCache
        etp.mutex.RUnlock()
        return token, nil
    }
    etp.mutex.RUnlock()

    etp.mutex.Lock()
    defer etp.mutex.Unlock()

    // Double-check after acquiring write lock
    if etp.tokenCache != nil && time.Now().Before(etp.tokenExpiry) {
        return *etp.tokenCache, nil
    }

    // Get new token
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

// Usage in provider:
func (ap *AzureProvider) getAuthHeader(ctx context.Context) (string, error) {
    if ap.entraTokenProvider != nil {
        return ap.entraTokenProvider.GetToken(ctx)
    }
    return ap.apiKey, nil
}

// Usage in request:
if strings.HasPrefix(authValue, "Bearer") {
    httpReq.Header.Set("Authorization", authValue)
} else {
    httpReq.Header.Set("api-key", authValue)
}
```

### 3. Managed Identity Support

```go
// System-assigned managed identity
func NewAzureProviderWithManagedIdentity(config ProviderConfigEntry) (*AzureProvider, error) {
    credential, err := azidentity.NewManagedIdentityCredential(nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create managed identity credential: %w", err)
    }

    entraProvider := NewEntraTokenProvider(credential)

    provider := &AzureProvider{
        config:             config,
        endpoint:           config.Parameters["endpoint"].(string),
        apiVersion:         getAPIVersion(config),
        entraTokenProvider: entraProvider,
        httpClient:         &http.Client{Timeout: 120 * time.Second},
        models:             getAzureModels(),
    }

    return provider, nil
}

// User-assigned managed identity
func NewAzureProviderWithUserManagedIdentity(config ProviderConfigEntry, clientID string) (*AzureProvider, error) {
    credential, err := azidentity.NewManagedIdentityCredential(&azidentity.ManagedIdentityCredentialOptions{
        ID: azidentity.ClientID(clientID),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create user-assigned managed identity credential: %w", err)
    }

    entraProvider := NewEntraTokenProvider(credential)

    provider := &AzureProvider{
        config:             config,
        endpoint:           config.Parameters["endpoint"].(string),
        apiVersion:         getAPIVersion(config),
        entraTokenProvider: entraProvider,
        httpClient:         &http.Client{Timeout: 120 * time.Second},
        models:             getAzureModels(),
    }

    return provider, nil
}
```

## Deployment-Based Routing

### Deployment Mapping

Azure OpenAI uses deployment names instead of model names in API calls. A deployment mapping is required:

```go
// DeploymentMap maps model names to deployment names
type DeploymentMap map[string]string

// Example:
deploymentMap := DeploymentMap{
    "gpt-4-turbo":      "my-gpt4-deployment",
    "gpt-35-turbo":     "my-gpt35-deployment",
    "gpt-4-vision":     "my-vision-deployment",
}

func (ap *AzureProvider) resolveDeployment(modelName string) string {
    // Check explicit mapping
    if deployment, ok := ap.deploymentMap[modelName]; ok {
        return deployment
    }

    // Fallback: use model name as deployment name
    // (works if deployment name matches model name)
    return modelName
}
```

### Loading Deployment Map

```go
// From configuration file
func loadDeploymentMap(config ProviderConfigEntry) (map[string]string, error) {
    deploymentMapParam := config.Parameters["deployment_map"]

    // Can be:
    // 1. JSON string
    // 2. File path to JSON
    // 3. Map directly

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
                return nil, fmt.Errorf("failed to parse deployment map: %w", err)
            }
            return m, nil
        }

        // Try to parse as JSON
        var m map[string]string
        if err := json.Unmarshal([]byte(v), &m); err != nil {
            return nil, fmt.Errorf("failed to parse deployment map JSON: %w", err)
        }
        return m, nil

    case map[string]interface{}:
        // Convert to map[string]string
        m := make(map[string]string)
        for k, val := range v {
            m[k] = val.(string)
        }
        return m, nil

    case map[string]string:
        return v, nil

    default:
        return make(map[string]string), nil
    }
}
```

## Error Handling Strategy

### Azure-Specific Error Handling

```go
type AzureError struct {
    Error struct {
        Code    string `json:"code"`
        Message string `json:"message"`
        Type    string `json:"type"`
        Param   string `json:"param,omitempty"`
    } `json:"error"`
}

func handleAzureError(statusCode int, body []byte) error {
    var azureErr AzureError
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
        case "InvalidApiKey":
            return fmt.Errorf("invalid Azure API key")
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
```

### Content Filtering Handling

```go
type AzureResponse struct {
    // Standard OpenAI fields
    ID      string    `json:"id"`
    Object  string    `json:"object"`
    Created int64     `json:"created"`
    Model   string    `json:"model"`
    Choices []Choice  `json:"choices"`
    Usage   Usage     `json:"usage"`

    // Azure-specific: Content filtering results
    PromptFilterResults []ContentFilterResult `json:"prompt_filter_results,omitempty"`
}

type ContentFilterResult struct {
    PromptIndex          int                  `json:"prompt_index"`
    ContentFilterResults ContentFilterDetails `json:"content_filter_results"`
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

func (ap *AzureProvider) parseResponse(body []byte) (*LLMResponse, error) {
    var azureResp AzureResponse
    if err := json.Unmarshal(body, &azureResp); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }

    // Check content filtering
    for _, filterResult := range azureResp.PromptFilterResults {
        filters := filterResult.ContentFilterResults
        if filters.Hate.Filtered || filters.SelfHarm.Filtered ||
           filters.Sexual.Filtered || filters.Violence.Filtered {
            return nil, fmt.Errorf("content filtered by Azure: prompt contains prohibited content")
        }
    }

    // Parse standard OpenAI response
    response := &LLMResponse{
        ID:        uuid.New(),
        Content:   azureResp.Choices[0].Message.Content,
        Usage: Usage{
            PromptTokens:     azureResp.Usage.PromptTokens,
            CompletionTokens: azureResp.Usage.CompletionTokens,
            TotalTokens:      azureResp.Usage.TotalTokens,
        },
        FinishReason: azureResp.Choices[0].FinishReason,
    }

    return response, nil
}
```

## Streaming Implementation

### SSE Stream Processing

```go
func (ap *AzureProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
    defer close(ch)

    // Build request
    deployment := ap.resolveDeployment(request.Model)
    azureReq := ap.buildAzureRequest(request)
    azureReq.Stream = true

    // Create HTTP request
    url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
        ap.endpoint, deployment, ap.apiVersion)

    reqBody, _ := json.Marshal(azureReq)
    httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
    if err != nil {
        return err
    }

    // Set headers
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Accept", "text/event-stream")

    authValue, err := ap.getAuthHeader(ctx)
    if err != nil {
        return err
    }
    if strings.HasPrefix(authValue, "Bearer") {
        httpReq.Header.Set("Authorization", authValue)
    } else {
        httpReq.Header.Set("api-key", authValue)
    }

    // Make request
    httpResp, err := ap.httpClient.Do(httpReq)
    if err != nil {
        return err
    }
    defer httpResp.Body.Close()

    if httpResp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(httpResp.Body)
        return handleAzureError(httpResp.StatusCode, body)
    }

    // Parse SSE stream
    return ap.parseSSEStream(httpResp.Body, ch, request.ID)
}

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
        var chunk struct {
            ID      string `json:"id"`
            Choices []struct {
                Index int `json:"index"`
                Delta struct {
                    Content string `json:"content"`
                    Role    string `json:"role"`
                } `json:"delta"`
                FinishReason string `json:"finish_reason"`
            } `json:"choices"`
        }

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
```

## Health Check Implementation

```go
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
```

## Testing Strategy

### 1. Mock HTTP Server Tests

```go
func TestAzureProvider_Generate(t *testing.T) {
    // Create mock server
    mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request
        assert.Equal(t, "POST", r.Method)
        assert.Contains(t, r.URL.Path, "/openai/deployments")
        assert.NotEmpty(t, r.Header.Get("api-key"))

        // Return mock response
        response := map[string]interface{}{
            "id":      "chatcmpl-123",
            "object":  "chat.completion",
            "created": time.Now().Unix(),
            "model":   "gpt-4-turbo",
            "choices": []map[string]interface{}{
                {
                    "index": 0,
                    "message": map[string]interface{}{
                        "role":    "assistant",
                        "content": "Hello! How can I help you?",
                    },
                    "finish_reason": "stop",
                },
            },
            "usage": map[string]interface{}{
                "prompt_tokens":     10,
                "completion_tokens": 20,
                "total_tokens":      30,
            },
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    }))
    defer mockServer.Close()

    // Create provider with mock server
    config := ProviderConfigEntry{
        Type:   ProviderTypeAzure,
        APIKey: "test-key",
        Parameters: map[string]interface{}{
            "endpoint":    mockServer.URL,
            "api_version": "2025-04-01-preview",
        },
    }

    provider, err := NewAzureProvider(config)
    require.NoError(t, err)

    // Test generation
    request := &LLMRequest{
        ID:       uuid.New(),
        Model:    "gpt-4-turbo",
        Messages: []Message{{Role: "user", Content: "Hello"}},
        MaxTokens: 100,
    }

    response, err := provider.Generate(context.Background(), request)
    assert.NoError(t, err)
    assert.NotNil(t, response)
    assert.Equal(t, "Hello! How can I help you?", response.Content)
    assert.Equal(t, 30, response.Usage.TotalTokens)
}
```

### 2. Deployment Mapping Tests

```go
func TestAzureProvider_DeploymentMapping(t *testing.T) {
    tests := []struct {
        name          string
        deploymentMap map[string]string
        modelName     string
        expected      string
    }{
        {
            name: "explicit mapping",
            deploymentMap: map[string]string{
                "gpt-4-turbo": "my-gpt4-deployment",
            },
            modelName: "gpt-4-turbo",
            expected:  "my-gpt4-deployment",
        },
        {
            name:          "fallback to model name",
            deploymentMap: map[string]string{},
            modelName:     "gpt-35-turbo",
            expected:      "gpt-35-turbo",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            provider := &AzureProvider{
                deploymentMap: tt.deploymentMap,
            }

            deployment := provider.resolveDeployment(tt.modelName)
            assert.Equal(t, tt.expected, deployment)
        })
    }
}
```

### 3. Authentication Tests

```go
func TestAzureProvider_Authentication(t *testing.T) {
    t.Run("API Key Auth", func(t *testing.T) {
        config := ProviderConfigEntry{
            Type:   ProviderTypeAzure,
            APIKey: "test-api-key",
            Parameters: map[string]interface{}{
                "endpoint": "https://test.openai.azure.com",
            },
        }

        provider, err := NewAzureProvider(config)
        require.NoError(t, err)

        authValue, err := provider.getAuthHeader(context.Background())
        assert.NoError(t, err)
        assert.Equal(t, "test-api-key", authValue)
    })

    t.Run("Entra ID Auth", func(t *testing.T) {
        // Mock credential
        mockCred := &mockTokenCredential{
            token: "mock-entra-token",
        }

        provider := &AzureProvider{
            entraTokenProvider: NewEntraTokenProvider(mockCred),
        }

        authValue, err := provider.getAuthHeader(context.Background())
        assert.NoError(t, err)
        assert.Equal(t, "Bearer mock-entra-token", authValue)
    })
}
```

### 4. Content Filtering Tests

```go
func TestAzureProvider_ContentFiltering(t *testing.T) {
    mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        response := map[string]interface{}{
            "error": map[string]interface{}{
                "code":    "content_filter",
                "message": "The prompt contains content that was filtered",
            },
        }

        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(response)
    }))
    defer mockServer.Close()

    config := ProviderConfigEntry{
        Type:   ProviderTypeAzure,
        APIKey: "test-key",
        Parameters: map[string]interface{}{
            "endpoint": mockServer.URL,
        },
    }

    provider, err := NewAzureProvider(config)
    require.NoError(t, err)

    request := &LLMRequest{
        ID:       uuid.New(),
        Model:    "gpt-4-turbo",
        Messages: []Message{{Role: "user", Content: "filtered content"}},
    }

    _, err = provider.Generate(context.Background(), request)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "content filtered")
}
```

## Configuration Schema

```yaml
# config/config.yaml
llm:
  providers:
    azure:
      type: "azure"
      enabled: true
      api_key: "${AZURE_OPENAI_API_KEY}"  # or use Entra ID
      endpoint: "https://myresource.openai.azure.com"
      api_version: "2025-04-01-preview"
      deployment_map:
        gpt-4-turbo: "my-gpt4-turbo-deployment"
        gpt-35-turbo: "my-gpt35-deployment"
        gpt-4-vision: "my-vision-deployment"
      # Optional: Entra ID authentication
      use_entra_id: false
      managed_identity: false
      managed_identity_client_id: ""  # for user-assigned identity
      models:
        - "gpt-4-turbo"
        - "gpt-35-turbo"
        - "gpt-4-vision-preview"
```

**Environment Variables**:
```bash
# API Key Authentication
export AZURE_OPENAI_API_KEY="your-api-key"
export AZURE_API_BASE="https://myresource.openai.azure.com"
export AZURE_API_VERSION="2025-04-01-preview"

# Entra ID Authentication
export AZURE_TENANT_ID="your-tenant-id"
export AZURE_CLIENT_ID="your-client-id"
export AZURE_CLIENT_SECRET="your-client-secret"

# Deployment mapping (JSON file or string)
export AZURE_DEPLOYMENTS_MAP='{"gpt-4-turbo":"my-deployment"}'
# OR
export AZURE_DEPLOYMENTS_MAP="/path/to/deployments.json"
```

**Deployment Map JSON File**:
```json
{
  "gpt-4-turbo": "production-gpt4-deployment",
  "gpt-35-turbo": "production-gpt35-deployment",
  "gpt-4-vision-preview": "vision-deployment",
  "text-embedding-3-large": "embeddings-deployment"
}
```

## Example Usage

### Basic Generation

```go
// Initialize with API key
config := ProviderConfigEntry{
    Type:   ProviderTypeAzure,
    APIKey: os.Getenv("AZURE_OPENAI_API_KEY"),
    Parameters: map[string]interface{}{
        "endpoint":    "https://myresource.openai.azure.com",
        "api_version": "2025-04-01-preview",
        "deployment_map": map[string]string{
            "gpt-4-turbo": "my-gpt4-deployment",
        },
    },
}

provider, err := NewAzureProvider(config)
if err != nil {
    log.Fatal(err)
}
defer provider.Close()

// Generate response
request := &LLMRequest{
    ID:    uuid.New(),
    Model: "gpt-4-turbo",
    Messages: []Message{
        {Role: "user", Content: "Explain Azure OpenAI Service."},
    },
    MaxTokens:   500,
    Temperature: 0.7,
}

response, err := provider.Generate(context.Background(), request)
if err != nil {
    log.Fatal(err)
}

fmt.Println(response.Content)
```

### Entra ID Authentication

```go
import (
    "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// Create credential
credential, err := azidentity.NewDefaultAzureCredential(nil)
if err != nil {
    log.Fatal(err)
}

// Create Entra token provider
entraProvider := NewEntraTokenProvider(credential)

// Create Azure provider
provider := &AzureProvider{
    endpoint:           "https://myresource.openai.azure.com",
    apiVersion:         "2025-04-01-preview",
    entraTokenProvider: entraProvider,
    deploymentMap:      loadDeploymentMap(),
    httpClient:         &http.Client{Timeout: 120 * time.Second},
    models:             getAzureModels(),
}

// Use provider normally
response, err := provider.Generate(context.Background(), request)
```

## Migration Notes

### From OpenAI to Azure OpenAI

**Key Changes**:
1. **Endpoint**: Change from `api.openai.com` to your Azure resource endpoint
2. **Authentication**: Use `api-key` header instead of `Authorization: Bearer`
3. **API Version**: Add `api-version` query parameter
4. **Deployments**: Map model names to deployment names
5. **URL Format**: `/openai/deployments/{deployment}/chat/completions`

**Migration Steps**:
1. Deploy models in Azure OpenAI portal
2. Create deployment mappings
3. Update configuration to use Azure provider
4. Test with health check
5. Monitor content filtering

## Dependencies

```go
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
```

**go.mod additions**:
```
require (
    github.com/Azure/azure-sdk-for-go/sdk/azcore v1.16.0
    github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.8.0
)
```

## References

- **Azure OpenAI Documentation**: https://learn.microsoft.com/azure/ai-services/openai/
- **Azure OpenAI API Reference**: https://learn.microsoft.com/azure/ai-services/openai/reference
- **Azure SDK for Go**: https://github.com/Azure/azure-sdk-for-go
- **Plandex Azure Implementation**: `/Users/milosvasic/Projects/HelixCode/Example_Projects/Plandex/app/shared/ai_models_providers.go`
- **OpenAI Provider Reference**: `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/openai_provider.go`
- **Content Filtering**: https://learn.microsoft.com/azure/ai-services/openai/concepts/content-filter

## Performance Characteristics

- **Latency**: Similar to OpenAI (200ms - 2s first token)
- **Throughput**: Based on your Azure deployment tier
- **Rate Limits**: Configurable per deployment (TPM/RPM)
- **Regional Availability**: Deploy in regions close to your users
- **SLA**: 99.9% uptime for Standard tier

## Security Considerations

1. **Managed Identity**: Use Azure managed identities for production
2. **Private Endpoints**: Enable private endpoints for VNet isolation
3. **Content Filtering**: Configure content filters based on your use case
4. **RBAC**: Use Azure RBAC for fine-grained access control
5. **Audit Logging**: Enable diagnostic logging for compliance
6. **Key Rotation**: Implement regular API key rotation
7. **Network Security**: Use Azure Firewall rules to restrict access
