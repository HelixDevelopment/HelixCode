# Google Vertex AI Provider - Technical Design

## Overview

The Vertex AI Provider enables access to Google's foundation models through Google Cloud's Vertex AI platform. This includes Gemini models (Google's flagship LLMs), Claude models via Model Garden, PaLM 2, and other third-party models. The provider integrates with Google Cloud SDK, supports service account authentication, project/location-based routing, streaming, and batch processing.

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                   VertexAIProvider                              │
├─────────────────────────────────────────────────────────────────┤
│  - config: ProviderConfigEntry                                  │
│  - credentials: *google.Credentials                             │
│  - projectID: string                                            │
│  - location: string (e.g., "us-central1")                       │
│  - endpoint: string                                             │
│  - httpClient: *http.Client                                     │
│  - models: []ModelInfo                                          │
│  - tokenProvider: *TokenProvider                                │
├─────────────────────────────────────────────────────────────────┤
│  Methods:                                                       │
│  + NewVertexAIProvider(config) (*VertexAIProvider, error)       │
│  + Generate(ctx, request) (*LLMResponse, error)                 │
│  + GenerateStream(ctx, request, ch) error                       │
│  + buildVertexRequest(request) (interface{}, error)             │
│  + getAccessToken(ctx) (string, error)                          │
│  + resolveEndpoint(model) string                                │
│  + GetType() ProviderType                                       │
│  + GetName() string                                             │
│  + GetModels() []ModelInfo                                      │
│  + GetCapabilities() []ModelCapability                          │
│  + IsAvailable(ctx) bool                                        │
│  + GetHealth(ctx) (*ProviderHealth, error)                      │
│  + Close() error                                                │
└─────────────────────────────────────────────────────────────────┘
              │
              │ Uses Google Cloud APIs
              ▼
┌─────────────────────────────────────────────────────────────────┐
│            Google Vertex AI Platform                            │
├─────────────────────────────────────────────────────────────────┤
│  - Gemini API (Google models)                                   │
│    └─> /v1/projects/{project}/locations/{location}/publishers/ │
│        google/models/{model}:generateContent                    │
│                                                                 │
│  - Model Garden (Third-party models including Claude)           │
│    └─> /v1/projects/{project}/locations/{location}/publishers/ │
│        anthropic/models/{model}:rawPredict                      │
│                                                                 │
│  - PaLM 2 API (Legacy)                                          │
│    └─> /v1/projects/{project}/locations/{location}/publishers/ │
│        google/models/{model}:predict                            │
│                                                                 │
│  - Streaming: streamGenerateContent / serverSentEvents          │
│  - Batch Processing: batchPredict                               │
└─────────────────────────────────────────────────────────────────┘
```

### Component Breakdown

**VertexAIProvider Struct**:
```go
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

type TokenProvider struct {
    credentials *google.Credentials
    tokenCache  *oauth2.Token
    mutex       sync.RWMutex
}
```

**Vertex AI Configuration**:
- Project ID (required)
- Location/Region (required, e.g., "us-central1", "europe-west1")
- Service account credentials (JSON file or ADC)
- Model publisher (google, anthropic, meta, etc.)
- Custom endpoint (optional)

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

**1. Gemini Models (Google Native)**
```go
{
    Name:           "gemini-2.5-pro",
    Provider:       ProviderTypeVertexAI,
    ContextSize:    2097152, // 2M tokens
    MaxTokens:      8192,
    Capabilities:   allCapabilities,
    SupportsTools:  true,
    SupportsVision: true,
    Description:    "Gemini 2.5 Pro via Vertex AI - Most capable model",
},
{
    Name:           "gemini-2.5-flash",
    Provider:       ProviderTypeVertexAI,
    ContextSize:    1048576, // 1M tokens
    MaxTokens:      8192,
    Capabilities:   allCapabilities,
    SupportsTools:  true,
    SupportsVision: true,
    Description:    "Gemini 2.5 Flash via Vertex AI - Fast and efficient",
},
{
    Name:           "gemini-2.0-flash-001",
    Provider:       ProviderTypeVertexAI,
    ContextSize:    1048576,
    MaxTokens:      8192,
    Capabilities:   allCapabilities,
    SupportsTools:  true,
    SupportsVision: true,
    Description:    "Gemini 2.0 Flash via Vertex AI",
},
{
    Name:           "gemini-1.5-pro",
    Provider:       ProviderTypeVertexAI,
    ContextSize:    2097152,
    MaxTokens:      8192,
    Capabilities:   allCapabilities,
    SupportsTools:  true,
    SupportsVision: true,
    Description:    "Gemini 1.5 Pro via Vertex AI",
},
{
    Name:           "gemini-1.5-flash",
    Provider:       ProviderTypeVertexAI,
    ContextSize:    1048576,
    MaxTokens:      8192,
    Capabilities:   allCapabilities,
    SupportsTools:  true,
    SupportsVision: true,
    Description:    "Gemini 1.5 Flash via Vertex AI",
},
```

**2. Claude Models via Model Garden**
```go
{
    Name:           "claude-sonnet-4@20250514",
    Provider:       ProviderTypeVertexAI,
    ContextSize:    200000,
    MaxTokens:      50000,
    Capabilities:   allCapabilities,
    SupportsTools:  true,
    SupportsVision: true,
    Description:    "Claude Sonnet 4 via Vertex AI Model Garden",
    Publisher:      "anthropic",
},
{
    Name:           "claude-opus-4@20250514",
    Provider:       ProviderTypeVertexAI,
    ContextSize:    200000,
    MaxTokens:      50000,
    Capabilities:   allCapabilities,
    SupportsTools:  true,
    SupportsVision: true,
    Description:    "Claude Opus 4 via Vertex AI Model Garden",
    Publisher:      "anthropic",
},
{
    Name:           "claude-3-7-sonnet@20250219",
    Provider:       ProviderTypeVertexAI,
    ContextSize:    200000,
    MaxTokens:      50000,
    Capabilities:   allCapabilities,
    SupportsTools:  true,
    SupportsVision: true,
    Description:    "Claude 3.7 Sonnet via Vertex AI Model Garden",
    Publisher:      "anthropic",
},
{
    Name:           "claude-3-5-sonnet-v2@20241022",
    Provider:       ProviderTypeVertexAI,
    ContextSize:    200000,
    MaxTokens:      8192,
    Capabilities:   allCapabilities,
    SupportsTools:  true,
    SupportsVision: true,
    Description:    "Claude 3.5 Sonnet v2 via Vertex AI Model Garden",
    Publisher:      "anthropic",
},
```

**3. PaLM 2 Models (Legacy)**
```go
{
    Name:           "text-bison@002",
    Provider:       ProviderTypeVertexAI,
    ContextSize:    8192,
    MaxTokens:      1024,
    Capabilities:   textCapabilities,
    SupportsTools:  false,
    SupportsVision: false,
    Description:    "PaLM 2 Text Bison - Legacy text model",
},
{
    Name:           "chat-bison@002",
    Provider:       ProviderTypeVertexAI,
    ContextSize:    8192,
    MaxTokens:      1024,
    Capabilities:   textCapabilities,
    SupportsTools:  false,
    SupportsVision: false,
    Description:    "PaLM 2 Chat Bison - Legacy chat model",
},
```

**4. Meta Llama via Model Garden**
```go
{
    Name:           "llama-4-maverick-17b-128e-instruct-maas",
    Provider:       ProviderTypeVertexAI,
    ContextSize:    1048576,
    MaxTokens:      8192,
    Capabilities:   allCapabilities,
    SupportsTools:  true,
    SupportsVision: false,
    Description:    "Llama 4 Maverick 17B via Vertex AI",
    Publisher:      "meta",
},
```

## Request/Response Flow

### Non-Streaming Flow (Gemini)

```
Client Request
    │
    ▼
VertexAIProvider.Generate(ctx, request)
    │
    ├─> getAccessToken(ctx)
    │   └─> Get OAuth2 token from service account credentials
    │
    ├─> buildVertexRequest(request)
    │   └─> Convert to Vertex AI format
    │       {
    │         "contents": [
    │           {"role": "user", "parts": [{"text": "..."}]}
    │         ],
    │         "generationConfig": {
    │           "temperature": 0.7,
    │           "maxOutputTokens": 1024,
    │           "topP": 0.95
    │         },
    │         "tools": [...]
    │       }
    │
    ├─> resolveEndpoint(request.Model)
    │   └─> Determine URL based on model and publisher
    │       Gemini: /v1/projects/{project}/locations/{location}/publishers/google/models/{model}:generateContent
    │       Claude:  /v1/projects/{project}/locations/{location}/publishers/anthropic/models/{model}:rawPredict
    │
    ├─> POST to Vertex AI endpoint
    │   Headers:
    │     - Authorization: Bearer {access_token}
    │     - Content-Type: application/json
    │
    ├─> parseVertexResponse(responseBody)
    │   └─> Convert from Vertex AI format to LLMResponse
    │
    └─> Return LLMResponse
```

### Streaming Flow (Gemini)

```
Client Request
    │
    ▼
VertexAIProvider.GenerateStream(ctx, request, ch)
    │
    ├─> getAccessToken(ctx)
    │
    ├─> buildVertexRequest(request)
    │
    ├─> resolveEndpoint(request.Model) + ":streamGenerateContent"
    │   └─> /v1/projects/{project}/locations/{location}/publishers/google/models/{model}:streamGenerateContent?alt=sse
    │
    ├─> POST to streaming endpoint
    │   Headers:
    │     - Authorization: Bearer {access_token}
    │     - Accept: text/event-stream
    │
    ├─> Parse SSE stream
    │   └─> For each event:
    │       ├─> Parse JSON chunk
    │       ├─> Extract text delta
    │       ├─> Send to channel
    │       └─> Handle completion
    │
    └─> Close channel
```

### Claude via Model Garden Flow

```
Client Request
    │
    ▼
VertexAIProvider.Generate(ctx, request)
    │
    ├─> buildAnthropicVertexRequest(request)
    │   └─> Convert to Anthropic format (wrapped for Vertex)
    │       {
    │         "anthropic_version": "vertex-2023-10-16",
    │         "messages": [...],
    │         "max_tokens": 1024,
    │         "stream": false
    │       }
    │
    ├─> POST to Model Garden endpoint
    │   URL: /v1/projects/{project}/locations/{location}/publishers/anthropic/models/{model}:rawPredict
    │
    ├─> parseAnthropicVertexResponse(responseBody)
    │   └─> Unwrap Vertex envelope and parse Anthropic format
    │
    └─> Return LLMResponse
```

## Authentication Mechanisms

### 1. Service Account JSON File (Recommended)

```go
import (
    "google.golang.org/api/option"
    "golang.org/x/oauth2/google"
)

func NewVertexAIProvider(config ProviderConfigEntry) (*VertexAIProvider, error) {
    // Load credentials from JSON file
    credentialsPath := config.Parameters["credentials_path"].(string)
    if credentialsPath == "" {
        credentialsPath = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
    }

    credentialsData, err := os.ReadFile(credentialsPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read credentials file: %w", err)
    }

    credentials, err := google.CredentialsFromJSON(
        context.Background(),
        credentialsData,
        "https://www.googleapis.com/auth/cloud-platform",
    )
    if err != nil {
        return nil, fmt.Errorf("failed to parse credentials: %w", err)
    }

    projectID := config.Parameters["project_id"].(string)
    location := config.Parameters["location"].(string)

    provider := &VertexAIProvider{
        config:        config,
        credentials:   credentials,
        projectID:     projectID,
        location:      location,
        endpoint:      fmt.Sprintf("https://%s-aiplatform.googleapis.com", location),
        httpClient:    &http.Client{Timeout: 120 * time.Second},
        tokenProvider: NewTokenProvider(credentials),
        models:        getVertexAIModels(),
    }

    return provider, nil
}
```

### 2. Application Default Credentials (ADC)

```go
func NewVertexAIProviderWithADC(config ProviderConfigEntry) (*VertexAIProvider, error) {
    ctx := context.Background()

    // Use Application Default Credentials
    // Automatically finds credentials from:
    // 1. GOOGLE_APPLICATION_CREDENTIALS env var
    // 2. gcloud CLI default credentials
    // 3. Compute Engine/GKE/Cloud Run metadata server
    credentials, err := google.FindDefaultCredentials(
        ctx,
        "https://www.googleapis.com/auth/cloud-platform",
    )
    if err != nil {
        return nil, fmt.Errorf("failed to find default credentials: %w", err)
    }

    projectID := config.Parameters["project_id"].(string)
    if projectID == "" {
        // Try to get project ID from credentials
        projectID = credentials.ProjectID
    }

    location := config.Parameters["location"].(string)

    provider := &VertexAIProvider{
        config:        config,
        credentials:   credentials,
        projectID:     projectID,
        location:      location,
        endpoint:      fmt.Sprintf("https://%s-aiplatform.googleapis.com", location),
        httpClient:    &http.Client{Timeout: 120 * time.Second},
        tokenProvider: NewTokenProvider(credentials),
        models:        getVertexAIModels(),
    }

    return provider, nil
}
```

### 3. Token Provider with Caching

```go
type TokenProvider struct {
    credentials *google.Credentials
    tokenCache  *oauth2.Token
    mutex       sync.RWMutex
}

func NewTokenProvider(credentials *google.Credentials) *TokenProvider {
    return &TokenProvider{
        credentials: credentials,
    }
}

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

func (vp *VertexAIProvider) getAccessToken(ctx context.Context) (string, error) {
    return vp.tokenProvider.GetToken(ctx)
}
```

## Error Handling Strategy

### GCP-Specific Error Handling

```go
type VertexAIError struct {
    Error struct {
        Code    int    `json:"code"`
        Message string `json:"message"`
        Status  string `json:"status"`
        Details []struct {
            Type     string `json:"@type"`
            Reason   string `json:"reason"`
            Domain   string `json:"domain"`
            Metadata map[string]string `json:"metadata"`
        } `json:"details"`
    } `json:"error"`
}

func handleVertexError(statusCode int, body []byte) error {
    var vertexErr VertexAIError
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
```

### Quota Management

```go
type QuotaTracker struct {
    projectID         string
    location          string
    requestsPerMinute int
    tokensPerMinute   int
    mutex             sync.Mutex
    lastReset         time.Time
    currentRequests   int
    currentTokens     int
}

func NewQuotaTracker(projectID, location string, rpm, tpm int) *QuotaTracker {
    return &QuotaTracker{
        projectID:         projectID,
        location:          location,
        requestsPerMinute: rpm,
        tokensPerMinute:   tpm,
        lastReset:         time.Now(),
    }
}

func (qt *QuotaTracker) CheckQuota(tokens int) error {
    qt.mutex.Lock()
    defer qt.mutex.Unlock()

    // Reset counters if minute has passed
    if time.Since(qt.lastReset) >= time.Minute {
        qt.currentRequests = 0
        qt.currentTokens = 0
        qt.lastReset = time.Now()
    }

    // Check request limit
    if qt.currentRequests >= qt.requestsPerMinute {
        return fmt.Errorf("request quota exceeded (%d RPM)", qt.requestsPerMinute)
    }

    // Check token limit
    if qt.currentTokens+tokens > qt.tokensPerMinute {
        return fmt.Errorf("token quota exceeded (%d TPM)", qt.tokensPerMinute)
    }

    qt.currentRequests++
    qt.currentTokens += tokens

    return nil
}
```

## Streaming Implementation

### Gemini Streaming

```go
func (vp *VertexAIProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
    defer close(ch)

    // Build request
    vertexReq := vp.buildVertexRequest(request)

    // Get access token
    token, err := vp.getAccessToken(ctx)
    if err != nil {
        return err
    }

    // Build streaming URL
    url := fmt.Sprintf("%s/v1/projects/%s/locations/%s/publishers/google/models/%s:streamGenerateContent?alt=sse",
        vp.endpoint, vp.projectID, vp.location, request.Model)

    // Create HTTP request
    reqBody, _ := json.Marshal(vertexReq)
    httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
    if err != nil {
        return err
    }

    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
    httpReq.Header.Set("Accept", "text/event-stream")

    // Make request
    httpResp, err := vp.httpClient.Do(httpReq)
    if err != nil {
        return err
    }
    defer httpResp.Body.Close()

    if httpResp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(httpResp.Body)
        return handleVertexError(httpResp.StatusCode, body)
    }

    // Parse SSE stream
    return vp.parseSSEStream(httpResp.Body, ch, request.ID)
}

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
        var streamResp struct {
            Candidates []struct {
                Content struct {
                    Parts []struct {
                        Text string `json:"text"`
                    } `json:"parts"`
                } `json:"content"`
                FinishReason string `json:"finishReason"`
            } `json:"candidates"`
            UsageMetadata struct {
                PromptTokenCount     int `json:"promptTokenCount"`
                CandidatesTokenCount int `json:"candidatesTokenCount"`
                TotalTokenCount      int `json:"totalTokenCount"`
            } `json:"usageMetadata"`
        }

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
            if part.Text != "" {
                contentBuilder.WriteString(part.Text)

                // Send incremental response
                ch <- LLMResponse{
                    ID:        uuid.New(),
                    RequestID: requestID,
                    Content:   part.Text,
                    CreatedAt: time.Now(),
                }
            }
        }

        // Check for completion
        if candidate.FinishReason != "" {
            ch <- LLMResponse{
                ID:           uuid.New(),
                RequestID:    requestID,
                Content:      contentBuilder.String(),
                FinishReason: candidate.FinishReason,
                Usage: Usage{
                    PromptTokens:     streamResp.UsageMetadata.PromptTokenCount,
                    CompletionTokens: streamResp.UsageMetadata.CandidatesTokenCount,
                    TotalTokens:      streamResp.UsageMetadata.TotalTokenCount,
                },
                CreatedAt: time.Now(),
            }
        }
    }

    return scanner.Err()
}
```

## Health Check Implementation

```go
func (vp *VertexAIProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
    startTime := time.Now()

    health := &ProviderHealth{
        LastCheck:  time.Now(),
        ModelCount: len(vp.models),
    }

    // Test with minimal request
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
```

## Testing Strategy

### 1. Mock HTTP Server Tests

```go
func TestVertexAIProvider_Generate(t *testing.T) {
    mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request
        assert.Contains(t, r.URL.Path, "/publishers/google/models")
        assert.NotEmpty(t, r.Header.Get("Authorization"))

        // Return mock response
        response := map[string]interface{}{
            "candidates": []map[string]interface{}{
                {
                    "content": map[string]interface{}{
                        "parts": []map[string]interface{}{
                            {"text": "Hello! How can I help you?"},
                        },
                    },
                    "finishReason": "STOP",
                },
            },
            "usageMetadata": map[string]interface{}{
                "promptTokenCount":     10,
                "candidatesTokenCount": 20,
                "totalTokenCount":      30,
            },
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    }))
    defer mockServer.Close()

    // Create provider with mock
    provider := &VertexAIProvider{
        projectID: "test-project",
        location:  "us-central1",
        endpoint:  mockServer.URL,
        httpClient: &http.Client{},
        tokenProvider: &TokenProvider{
            tokenCache: &oauth2.Token{
                AccessToken: "test-token",
                Expiry:      time.Now().Add(1 * time.Hour),
            },
        },
    }

    request := &LLMRequest{
        ID:       uuid.New(),
        Model:    "gemini-2.5-flash",
        Messages: []Message{{Role: "user", Content: "Hello"}},
    }

    response, err := provider.Generate(context.Background(), request)
    assert.NoError(t, err)
    assert.NotNil(t, response)
    assert.Equal(t, "Hello! How can I help you?", response.Content)
}
```

### 2. Credentials Tests

```go
func TestVertexAIProvider_Credentials(t *testing.T) {
    t.Run("Service Account JSON", func(t *testing.T) {
        // Create temporary credentials file
        credJSON := `{
            "type": "service_account",
            "project_id": "test-project",
            "private_key_id": "key-id",
            "private_key": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n",
            "client_email": "test@test-project.iam.gserviceaccount.com",
            "client_id": "123456789",
            "auth_uri": "https://accounts.google.com/o/oauth2/auth",
            "token_uri": "https://oauth2.googleapis.com/token"
        }`

        tmpFile, err := os.CreateTemp("", "credentials-*.json")
        require.NoError(t, err)
        defer os.Remove(tmpFile.Name())

        _, err = tmpFile.Write([]byte(credJSON))
        require.NoError(t, err)
        tmpFile.Close()

        config := ProviderConfigEntry{
            Type: ProviderTypeVertexAI,
            Parameters: map[string]interface{}{
                "credentials_path": tmpFile.Name(),
                "project_id":       "test-project",
                "location":         "us-central1",
            },
        }

        provider, err := NewVertexAIProvider(config)
        assert.NoError(t, err)
        assert.NotNil(t, provider)
        assert.Equal(t, "test-project", provider.projectID)
    })
}
```

### 3. Model Garden Tests

```go
func TestVertexAIProvider_ModelGarden(t *testing.T) {
    mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify Claude endpoint
        assert.Contains(t, r.URL.Path, "/publishers/anthropic/models")
        assert.Contains(t, r.URL.Path, "rawPredict")

        // Return Claude response format
        response := map[string]interface{}{
            "id":           "msg-123",
            "type":         "message",
            "role":         "assistant",
            "content":      []map[string]interface{}{{"type": "text", "text": "Hello from Claude!"}},
            "stop_reason":  "end_turn",
            "usage": map[string]interface{}{
                "input_tokens":  10,
                "output_tokens": 20,
            },
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    }))
    defer mockServer.Close()

    provider := &VertexAIProvider{
        projectID: "test-project",
        location:  "us-central1",
        endpoint:  mockServer.URL,
        httpClient: &http.Client{},
        tokenProvider: &TokenProvider{
            tokenCache: &oauth2.Token{AccessToken: "test-token"},
        },
    }

    request := &LLMRequest{
        ID:       uuid.New(),
        Model:    "claude-3-5-sonnet-v2@20241022",
        Messages: []Message{{Role: "user", Content: "Hello"}},
    }

    response, err := provider.Generate(context.Background(), request)
    assert.NoError(t, err)
    assert.Contains(t, response.Content, "Claude")
}
```

## Configuration Schema

```yaml
# config/config.yaml
llm:
  providers:
    vertex_ai:
      type: "vertex_ai"
      enabled: true
      project_id: "my-gcp-project"
      location: "us-central1"
      credentials_path: "/path/to/service-account.json"
      # OR use application default credentials
      use_adc: false
      models:
        # Gemini models
        - "gemini-2.5-pro"
        - "gemini-2.5-flash"
        - "gemini-1.5-pro"
        # Claude via Model Garden
        - "claude-sonnet-4@20250514"
        - "claude-3-7-sonnet@20250219"
      # Optional quota limits
      requests_per_minute: 60
      tokens_per_minute: 100000
```

**Environment Variables**:
```bash
# Service Account
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account.json"
export VERTEXAI_PROJECT="my-gcp-project"
export VERTEXAI_LOCATION="us-central1"

# Or use gcloud CLI default credentials
# (runs automatically if GOOGLE_APPLICATION_CREDENTIALS not set)
```

**Service Account JSON**:
```json
{
  "type": "service_account",
  "project_id": "my-gcp-project",
  "private_key_id": "key-id",
  "private_key": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n",
  "client_email": "helixcode@my-gcp-project.iam.gserviceaccount.com",
  "client_id": "123456789",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/..."
}
```

## Example Usage

### Basic Generation with Gemini

```go
config := ProviderConfigEntry{
    Type: ProviderTypeVertexAI,
    Parameters: map[string]interface{}{
        "credentials_path": "/path/to/service-account.json",
        "project_id":       "my-gcp-project",
        "location":         "us-central1",
    },
}

provider, err := NewVertexAIProvider(config)
if err != nil {
    log.Fatal(err)
}
defer provider.Close()

request := &LLMRequest{
    ID:    uuid.New(),
    Model: "gemini-2.5-pro",
    Messages: []Message{
        {Role: "user", Content: "Explain Vertex AI in one sentence."},
    },
    MaxTokens:   200,
    Temperature: 0.7,
}

response, err := provider.Generate(context.Background(), request)
if err != nil {
    log.Fatal(err)
}

fmt.Println(response.Content)
```

### Using Claude via Model Garden

```go
request := &LLMRequest{
    ID:    uuid.New(),
    Model: "claude-3-7-sonnet@20250219",
    Messages: []Message{
        {Role: "user", Content: "Write code to process files."},
    },
    MaxTokens:   1000,
    Temperature: 0.7,
}

response, err := provider.Generate(context.Background(), request)
// Claude response via Vertex AI
```

### Streaming with Gemini

```go
request := &LLMRequest{
    ID:    uuid.New(),
    Model: "gemini-2.5-flash",
    Messages: []Message{
        {Role: "user", Content: "Write a short story."},
    },
    MaxTokens:   2000,
    Temperature: 0.9,
    Stream:      true,
}

responseCh := make(chan LLMResponse)

go func() {
    if err := provider.GenerateStream(context.Background(), request, responseCh); err != nil {
        log.Printf("Stream error: %v", err)
    }
}()

for response := range responseCh {
    fmt.Print(response.Content)
}
```

## Migration Notes

### From Gemini API to Vertex AI

**Key Differences**:
- Authentication: OAuth2 service account instead of API key
- Endpoint: Regional endpoints instead of global
- URL format: Includes project ID and location
- Additional features: Model Garden, batch processing, monitoring

**Migration Steps**:
1. Create GCP project and enable Vertex AI API
2. Create service account with Vertex AI User role
3. Download service account JSON key
4. Update configuration to use Vertex AI provider
5. Test with simple request

### From Anthropic to Vertex AI Model Garden

**Advantages of Model Garden**:
- Same Claude models with GCP integration
- Unified billing with other GCP services
- VPC-SC and private endpoint support
- Integrated monitoring and logging

**Considerations**:
- Slightly different request format (wrapped)
- Need to enable Model Garden access
- Regional availability may differ

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

    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
    "google.golang.org/api/option"
    "github.com/google/uuid"
)
```

**go.mod additions**:
```
require (
    golang.org/x/oauth2 v0.24.0
    google.golang.org/api v0.210.0
)
```

## References

- **Vertex AI Documentation**: https://cloud.google.com/vertex-ai/docs
- **Gemini API on Vertex AI**: https://cloud.google.com/vertex-ai/generative-ai/docs/model-reference/gemini
- **Model Garden**: https://cloud.google.com/vertex-ai/docs/start/explore-models
- **Authentication**: https://cloud.google.com/docs/authentication
- **Forge Vertex Implementation**: `/Users/milosvasic/Projects/HelixCode/Example_Projects/Forge/vertex.json`
- **Gemini Provider Reference**: `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/gemini_provider.go`

## Performance Characteristics

- **Latency**: 300ms - 2s (first token)
- **Throughput**: Region-specific, configurable quotas
- **Rate Limits**: Project-level quotas (default: 60 RPM, adjustable)
- **Regional Availability**: us-central1, us-east4, europe-west1, asia-southeast1, etc.
- **Cost**: Pay-per-token pricing varies by model

## Security Considerations

1. **Service Account Permissions**: Use least privilege (Vertex AI User role)
2. **Credentials Storage**: Never commit service account JSON to version control
3. **VPC Service Controls**: Use VPC-SC for private endpoint access
4. **Audit Logging**: Enable Cloud Audit Logs for compliance
5. **Key Rotation**: Rotate service account keys regularly
6. **IAM Conditions**: Use IAM conditions for fine-grained access
7. **Workload Identity**: Use Workload Identity on GKE instead of keys
