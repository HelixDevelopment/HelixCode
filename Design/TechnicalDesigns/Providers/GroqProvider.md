# Groq Provider - Technical Design

## Overview

The Groq Provider enables ultra-fast inference using Groq's specialized LPU (Language Processing Unit) hardware. This provider offers OpenAI-compatible API access to models like Llama 3.3 70B, Mixtral 8x7B, and Gemma 2 9B with exceptional performance characteristics including sub-100ms latency and high throughput (up to 500+ tokens/second). The provider is designed for applications requiring real-time AI responses, streaming conversations, and high-concurrency workloads.

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      GroqProvider                               │
├─────────────────────────────────────────────────────────────────┤
│  - config: ProviderConfigEntry                                  │
│  - apiKey: string                                               │
│  - baseURL: string (default: https://api.groq.com)              │
│  - httpClient: *http.Client (optimized for speed)               │
│  - models: []ModelInfo                                          │
│  - lastHealth: *ProviderHealth                                  │
│  - latencyMetrics: *LatencyTracker                              │
├─────────────────────────────────────────────────────────────────┤
│  Methods:                                                       │
│  + NewGroqProvider(config) (*GroqProvider, error)               │
│  + Generate(ctx, request) (*LLMResponse, error)                 │
│  + GenerateStream(ctx, request, ch) error                       │
│  + GetType() ProviderType                                       │
│  + GetName() string                                             │
│  + GetModels() []ModelInfo                                      │
│  + GetCapabilities() []ModelCapability                          │
│  + IsAvailable(ctx) bool                                        │
│  + GetHealth(ctx) (*ProviderHealth, error)                      │
│  + GetLatencyMetrics() *LatencyMetrics                          │
│  + Close() error                                                │
└─────────────────────────────────────────────────────────────────┘
              │
              │ OpenAI-Compatible API
              ▼
┌─────────────────────────────────────────────────────────────────┐
│                  Groq Cloud API                                 │
├─────────────────────────────────────────────────────────────────┤
│  - /openai/v1/chat/completions                                  │
│  - /openai/v1/completions                                       │
│  - /openai/v1/models                                            │
│  - OpenAI-compatible request/response format                    │
│  - Ultra-low latency (LPU hardware)                             │
│  - High throughput (500+ tokens/sec)                            │
│  - Streaming support with SSE                                   │
└─────────────────────────────────────────────────────────────────┘
```

### Component Breakdown

**GroqProvider Struct**:
```go
type GroqProvider struct {
    config          ProviderConfigEntry
    apiKey          string
    baseURL         string
    httpClient      *http.Client
    models          []ModelInfo
    lastHealth      *ProviderHealth
    latencyMetrics  *LatencyTracker
}

type LatencyTracker struct {
    mutex           sync.RWMutex
    samples         []time.Duration
    maxSamples      int
    firstTokenTimes []time.Duration
    totalTokenTimes []time.Duration
}
```

**Performance Optimizations**:
- HTTP/2 support with connection pooling
- Keep-alive connections (persistent)
- Optimized timeout settings (shorter than other providers)
- Latency tracking for monitoring
- Pre-warmed connection pool

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

**1. Meta Llama 3.3**
```go
{
    Name:           "llama-3.3-70b-versatile",
    Provider:       ProviderTypeGroq,
    ContextSize:    131072, // 128K tokens
    MaxTokens:      32768,
    Capabilities:   allCapabilities,
    SupportsTools:  true,
    SupportsVision: false,
    Description:    "Llama 3.3 70B - Most capable model on Groq with ultra-fast inference",
    PerformanceProfile: PerformanceProfile{
        AvgFirstTokenLatency: 50 * time.Millisecond,
        AvgTokensPerSecond:   500,
    },
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
    PerformanceProfile: PerformanceProfile{
        AvgFirstTokenLatency: 60 * time.Millisecond,
        AvgTokensPerSecond:   450,
    },
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
    PerformanceProfile: PerformanceProfile{
        AvgFirstTokenLatency: 30 * time.Millisecond,
        AvgTokensPerSecond:   800,
    },
},
```

**2. Mixtral**
```go
{
    Name:           "mixtral-8x7b-32768",
    Provider:       ProviderTypeGroq,
    ContextSize:    32768,
    MaxTokens:      32768,
    Capabilities:   allCapabilities,
    SupportsTools:  true,
    SupportsVision: false,
    Description:    "Mixtral 8x7B - Mixture of experts with fast inference",
    PerformanceProfile: PerformanceProfile{
        AvgFirstTokenLatency: 70 * time.Millisecond,
        AvgTokensPerSecond:   400,
    },
},
```

**3. Google Gemma**
```go
{
    Name:           "gemma2-9b-it",
    Provider:       ProviderTypeGroq,
    ContextSize:    8192,
    MaxTokens:      8192,
    Capabilities:   allCapabilities,
    SupportsTools:  false,
    SupportsVision: false,
    Description:    "Gemma 2 9B - Google's efficient open model on Groq",
    PerformanceProfile: PerformanceProfile{
        AvgFirstTokenLatency: 40 * time.Millisecond,
        AvgTokensPerSecond:   600,
    },
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
    PerformanceProfile: PerformanceProfile{
        AvgFirstTokenLatency: 35 * time.Millisecond,
        AvgTokensPerSecond:   700,
    },
},
```

**Performance Profile Type**:
```go
type PerformanceProfile struct {
    AvgFirstTokenLatency time.Duration
    AvgTokensPerSecond   int
}
```

## Request/Response Flow

### Non-Streaming Flow

```
Client Request
    │
    ▼
GroqProvider.Generate(ctx, request)
    │
    ├─> buildGroqRequest(request)
    │   └─> Convert to OpenAI-compatible format
    │       (Same as OpenAI API structure)
    │
    ├─> Track start time for latency measurement
    │
    ├─> POST https://api.groq.com/openai/v1/chat/completions
    │   Headers:
    │     - Authorization: Bearer {api_key}
    │     - Content-Type: application/json
    │   Body:
    │     {
    │       "model": "llama-3.3-70b-versatile",
    │       "messages": [...],
    │       "max_tokens": 1024,
    │       "temperature": 0.7,
    │       "stream": false
    │     }
    │
    ├─> parseGroqResponse(responseBody)
    │   └─> Convert from OpenAI format to LLMResponse
    │
    ├─> Record latency metrics
    │   └─> Track first token time, total time
    │
    └─> Return LLMResponse
```

### Streaming Flow (Optimized)

```
Client Request
    │
    ▼
GroqProvider.GenerateStream(ctx, request, ch)
    │
    ├─> buildGroqRequest(request) + set stream=true
    │
    ├─> Track start time
    │
    ├─> POST https://api.groq.com/openai/v1/chat/completions
    │   Headers:
    │     - Authorization: Bearer {api_key}
    │     - Accept: text/event-stream
    │
    ├─> Parse SSE stream
    │   └─> For each "data: {json}" line:
    │       ├─> Track first token latency
    │       ├─> Parse JSON chunk
    │       ├─> Extract delta content
    │       ├─> Send to channel (low latency)
    │       └─> data: [DONE] -> complete
    │
    ├─> Record streaming metrics
    │   └─> Track tokens/second throughput
    │
    └─> Close channel
```

## Authentication Mechanism

### Simple API Key Authentication

```go
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
        Timeout: 60 * time.Second, // Shorter timeout for fast responses
        Transport: &http.Transport{
            MaxIdleConns:        100,
            MaxIdleConnsPerHost: 100,
            IdleConnTimeout:     90 * time.Second,
            TLSHandshakeTimeout: 10 * time.Second,
            // Enable HTTP/2
            ForceAttemptHTTP2: true,
        },
    }

    provider := &GroqProvider{
        config:         config,
        apiKey:         apiKey,
        baseURL:        baseURL,
        httpClient:     httpClient,
        models:         getGroqModels(),
        latencyMetrics: NewLatencyTracker(100), // Track last 100 requests
    }

    return provider, nil
}

// Usage in requests
func (gp *GroqProvider) makeRequest(ctx context.Context, request interface{}) (*http.Response, error) {
    url := fmt.Sprintf("%s/openai/v1/chat/completions", gp.baseURL)

    reqBody, _ := json.Marshal(request)
    httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
    if err != nil {
        return nil, err
    }

    // Set headers
    httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gp.apiKey))
    httpReq.Header.Set("Content-Type", "application/json")

    return gp.httpClient.Do(httpReq)
}
```

## Error Handling Strategy

### Groq-Specific Error Handling

```go
type GroqError struct {
    Error struct {
        Message string `json:"message"`
        Type    string `json:"type"`
        Code    string `json:"code"`
    } `json:"error"`
}

func handleGroqError(statusCode int, body []byte) error {
    var groqErr GroqError
    if err := json.Unmarshal(body, &groqErr); err == nil {
        errInfo := groqErr.Error

        switch statusCode {
        case http.StatusBadRequest:
            if strings.Contains(errInfo.Message, "context_length_exceeded") {
                return ErrContextTooLong
            }
            return ErrInvalidRequest

        case http.StatusUnauthorized:
            return fmt.Errorf("unauthorized: invalid Groq API key")

        case http.StatusTooManyRequests:
            // Groq rate limits are aggressive
            return ErrRateLimited

        case http.StatusServiceUnavailable:
            return fmt.Errorf("groq service unavailable: %s", errInfo.Message)

        case 529: // Custom Groq overload code
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
```

### Rate Limiting Handling

Groq has aggressive rate limits but extremely high throughput. Handle smartly:

```go
type RateLimitHandler struct {
    requestsPerMinute int
    tokensPerMinute   int
    currentRequests   int
    currentTokens     int
    lastReset         time.Time
    mutex             sync.Mutex
}

func (rlh *RateLimitHandler) WaitIfNeeded(estimatedTokens int) error {
    rlh.mutex.Lock()
    defer rlh.mutex.Unlock()

    // Reset if minute passed
    if time.Since(rlh.lastReset) >= time.Minute {
        rlh.currentRequests = 0
        rlh.currentTokens = 0
        rlh.lastReset = time.Now()
    }

    // Check if we need to wait
    if rlh.currentRequests >= rlh.requestsPerMinute {
        waitTime := time.Minute - time.Since(rlh.lastReset)
        if waitTime > 0 {
            time.Sleep(waitTime)
            rlh.currentRequests = 0
            rlh.currentTokens = 0
            rlh.lastReset = time.Now()
        }
    }

    rlh.currentRequests++
    rlh.currentTokens += estimatedTokens

    return nil
}
```

## Streaming Implementation

### High-Performance SSE Streaming

```go
func (gp *GroqProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
    defer close(ch)
    startTime := time.Now()

    // Build request
    groqReq := gp.buildGroqRequest(request)
    groqReq.Stream = true

    // Make request
    url := fmt.Sprintf("%s/openai/v1/chat/completions", gp.baseURL)
    reqBody, _ := json.Marshal(groqReq)

    httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
    if err != nil {
        return err
    }

    httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gp.apiKey))
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Accept", "text/event-stream")

    httpResp, err := gp.httpClient.Do(httpReq)
    if err != nil {
        return err
    }
    defer httpResp.Body.Close()

    if httpResp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(httpResp.Body)
        return handleGroqError(httpResp.StatusCode, body)
    }

    // Parse SSE stream with latency tracking
    return gp.parseSSEStreamWithMetrics(httpResp.Body, ch, request.ID, startTime)
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

        // Track first token latency (Groq's key metric)
        if !firstTokenReceived {
            firstTokenTime = time.Since(startTime)
            firstTokenReceived = true
        }

        // Parse JSON chunk
        var chunk struct {
            ID      string `json:"id"`
            Choices []struct {
                Index int `json:"index"`
                Delta struct {
                    Content string `json:"content"`
                } `json:"delta"`
                FinishReason string `json:"finish_reason"`
            } `json:"choices"`
            Model   string `json:"model"`
            Usage   *struct {
                PromptTokens     int `json:"prompt_tokens"`
                CompletionTokens int `json:"completion_tokens"`
                TotalTokens      int `json:"total_tokens"`
            } `json:"usage,omitempty"`
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
```

## Latency Tracking

```go
type LatencyTracker struct {
    mutex            sync.RWMutex
    maxSamples       int
    firstTokenTimes  []time.Duration
    totalTimes       []time.Duration
    tokensPerSecond  []float64
}

func NewLatencyTracker(maxSamples int) *LatencyTracker {
    return &LatencyTracker{
        maxSamples:      maxSamples,
        firstTokenTimes: make([]time.Duration, 0, maxSamples),
        totalTimes:      make([]time.Duration, 0, maxSamples),
        tokensPerSecond: make([]float64, 0, maxSamples),
    }
}

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

type LatencyMetrics struct {
    AvgFirstTokenLatency time.Duration
    P50FirstTokenLatency time.Duration
    P95FirstTokenLatency time.Duration
    P99FirstTokenLatency time.Duration
    AvgTotalLatency      time.Duration
    AvgTokensPerSecond   float64
    SampleCount          int
}

func (gp *GroqProvider) GetLatencyMetrics() *LatencyMetrics {
    metrics := gp.latencyMetrics.GetMetrics()
    return &metrics
}
```

## Health Check Implementation

```go
func (gp *GroqProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
    startTime := time.Now()

    health := &ProviderHealth{
        LastCheck:  time.Now(),
        ModelCount: len(gp.models),
    }

    // Test with fast model
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
        return health, err
    }

    health.Status = "healthy"
    health.Latency = time.Since(startTime)
    gp.lastHealth = health

    return health, nil
}
```

## Testing Strategy

### 1. Latency Tests

```go
func TestGroqProvider_Latency(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping latency test")
    }

    provider, err := NewGroqProvider(ProviderConfigEntry{
        Type:   ProviderTypeGroq,
        APIKey: os.Getenv("GROQ_API_KEY"),
    })
    require.NoError(t, err)

    request := &LLMRequest{
        ID:       uuid.New(),
        Model:    "llama-3.3-70b-versatile",
        Messages: []Message{{Role: "user", Content: "Say hello"}},
        MaxTokens: 50,
    }

    startTime := time.Now()
    response, err := provider.Generate(context.Background(), request)
    latency := time.Since(startTime)

    assert.NoError(t, err)
    assert.NotNil(t, response)

    // Groq should be very fast
    assert.Less(t, latency.Milliseconds(), int64(2000), "Expected response within 2 seconds")

    t.Logf("Groq latency: %v", latency)
}
```

### 2. Streaming Throughput Tests

```go
func TestGroqProvider_StreamingThroughput(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping throughput test")
    }

    provider, err := NewGroqProvider(ProviderConfigEntry{
        Type:   ProviderTypeGroq,
        APIKey: os.Getenv("GROQ_API_KEY"),
    })
    require.NoError(t, err)

    request := &LLMRequest{
        ID:    uuid.New(),
        Model: "llama-3.3-70b-versatile",
        Messages: []Message{{
            Role:    "user",
            Content: "Write a 500-word essay about artificial intelligence.",
        }},
        MaxTokens:   1000,
        Temperature: 0.7,
        Stream:      true,
    }

    responseCh := make(chan LLMResponse)
    startTime := time.Now()
    var tokenCount int
    var firstTokenTime time.Duration
    var firstTokenReceived bool

    go func() {
        err := provider.GenerateStream(context.Background(), request, responseCh)
        assert.NoError(t, err)
    }()

    for response := range responseCh {
        if !firstTokenReceived && response.Content != "" {
            firstTokenTime = time.Since(startTime)
            firstTokenReceived = true
        }
        tokenCount += len(strings.Fields(response.Content))
    }

    totalTime := time.Since(startTime)
    tokensPerSecond := float64(tokenCount) / totalTime.Seconds()

    t.Logf("First token latency: %v", firstTokenTime)
    t.Logf("Total time: %v", totalTime)
    t.Logf("Tokens per second: %.2f", tokensPerSecond)

    // Groq should achieve high throughput
    assert.Greater(t, tokensPerSecond, 200.0, "Expected > 200 tokens/second")
    assert.Less(t, firstTokenTime.Milliseconds(), int64(200), "Expected first token < 200ms")
}
```

### 3. Mock Tests

```go
func TestGroqProvider_Generate(t *testing.T) {
    mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "/openai/v1/chat/completions", r.URL.Path)
        assert.Contains(t, r.Header.Get("Authorization"), "Bearer")

        response := map[string]interface{}{
            "id":      "chatcmpl-groq-123",
            "object":  "chat.completion",
            "created": time.Now().Unix(),
            "model":   "llama-3.3-70b-versatile",
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

    provider := &GroqProvider{
        apiKey:  "test-key",
        baseURL: mockServer.URL,
        httpClient: &http.Client{},
        latencyMetrics: NewLatencyTracker(100),
    }

    request := &LLMRequest{
        ID:       uuid.New(),
        Model:    "llama-3.3-70b-versatile",
        Messages: []Message{{Role: "user", Content: "Hello"}},
    }

    response, err := provider.Generate(context.Background(), request)
    assert.NoError(t, err)
    assert.NotNil(t, response)
    assert.Equal(t, "Hello! How can I help you?", response.Content)
}
```

## Configuration Schema

```yaml
# config/config.yaml
llm:
  providers:
    groq:
      type: "groq"
      enabled: true
      api_key: "${GROQ_API_KEY}"
      base_url: "https://api.groq.com"  # optional
      models:
        - "llama-3.3-70b-versatile"
        - "llama-3.1-8b-instant"
        - "mixtral-8x7b-32768"
        - "gemma2-9b-it"
      # Rate limits (requests per minute)
      rate_limit_rpm: 30  # Free tier
      # Performance tracking
      track_latency: true
      latency_samples: 100
```

**Environment Variables**:
```bash
export GROQ_API_KEY="gsk_..."
# Optional
export GROQ_HOST="https://api.groq.com"
```

## Example Usage

### Basic Generation

```go
config := ProviderConfigEntry{
    Type:   ProviderTypeGroq,
    APIKey: os.Getenv("GROQ_API_KEY"),
}

provider, err := NewGroqProvider(config)
if err != nil {
    log.Fatal(err)
}
defer provider.Close()

request := &LLMRequest{
    ID:    uuid.New(),
    Model: "llama-3.3-70b-versatile",
    Messages: []Message{
        {Role: "user", Content: "Explain Groq LPU technology."},
    },
    MaxTokens:   500,
    Temperature: 0.7,
}

startTime := time.Now()
response, err := provider.Generate(context.Background(), request)
latency := time.Since(startTime)

if err != nil {
    log.Fatal(err)
}

fmt.Println(response.Content)
fmt.Printf("Latency: %v\n", latency)
fmt.Printf("Tokens: %d\n", response.Usage.TotalTokens)
```

### High-Performance Streaming

```go
request := &LLMRequest{
    ID:    uuid.New(),
    Model: "llama-3.1-8b-instant",  // Fastest model
    Messages: []Message{
        {Role: "user", Content: "Write code to process data streams."},
    },
    MaxTokens:   2000,
    Temperature: 0.7,
    Stream:      true,
}

responseCh := make(chan LLMResponse)

go func() {
    if err := provider.GenerateStream(context.Background(), request, responseCh); err != nil {
        log.Printf("Stream error: %v", err)
    }
}()

// Process with minimal delay
for response := range responseCh {
    fmt.Print(response.Content)

    // Check metadata for performance info
    if metadata, ok := response.ProviderMetadata.(map[string]interface{}); ok {
        if tps, ok := metadata["tokens_per_second"].(float64); ok {
            log.Printf("Throughput: %.2f tokens/sec", tps)
        }
    }
}
```

### Latency Monitoring

```go
// After running some requests
metrics := provider.GetLatencyMetrics()

fmt.Printf("Average first token latency: %v\n", metrics.AvgFirstTokenLatency)
fmt.Printf("P95 first token latency: %v\n", metrics.P95FirstTokenLatency)
fmt.Printf("P99 first token latency: %v\n", metrics.P99FirstTokenLatency)
fmt.Printf("Average throughput: %.2f tokens/sec\n", metrics.AvgTokensPerSecond)
fmt.Printf("Sample count: %d\n", metrics.SampleCount)
```

## Migration Notes

### From OpenAI to Groq

**Key Advantages**:
- 5-10x faster inference
- OpenAI-compatible API (easy migration)
- Lower costs for many use cases
- Excellent for real-time applications

**Migration Steps**:
1. Get Groq API key from console.groq.com
2. Replace OpenAI provider with Groq provider
3. Update model names (e.g., `gpt-4` → `llama-3.3-70b-versatile`)
4. Test latency improvements
5. Adjust rate limits if needed

**Considerations**:
- Groq has more aggressive rate limits (free tier: 30 RPM)
- Model selection is more limited
- No GPT-4 or GPT-3.5 (uses open models)

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

    "github.com/google/uuid"
)
```

No additional dependencies required - uses standard library.

## References

- **Groq Documentation**: https://console.groq.com/docs
- **Groq Models**: https://console.groq.com/docs/models
- **Codename Goose Groq Implementation**: `/Users/milosvasic/Projects/HelixCode/Example_Projects/Codename_Goose/crates/goose/src/providers/groq.rs`
- **OpenAI Provider Reference**: `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/openai_provider.go`
- **Groq Console**: https://console.groq.com

## Performance Characteristics

- **First Token Latency**: 30-100ms (depending on model)
- **Throughput**: 200-800+ tokens/second
- **Best Models for Speed**:
  - llama-3.1-8b-instant: ~800 tokens/sec
  - gemma2-9b-it: ~600 tokens/sec
  - llama-3.3-70b-versatile: ~500 tokens/sec
- **Rate Limits**:
  - Free tier: 30 RPM, 14,400 TPD
  - Paid tier: Higher limits (check console)

## Use Cases

### Ideal Use Cases

1. **Real-Time Chat**: Sub-100ms responses for conversational AI
2. **High-Volume Processing**: Batch processing with high throughput
3. **Streaming Applications**: Live transcription, code completion
4. **Interactive Agents**: Multi-turn conversations with low latency
5. **API Services**: Fast API responses for production applications

### Not Ideal For

1. **Proprietary Models**: No GPT-4 or Claude access
2. **Very High Volume**: Rate limits may be restrictive
3. **Specialized Tasks**: Limited model selection

## Security Considerations

1. **API Key Protection**: Store securely, never commit to version control
2. **Rate Limiting**: Implement client-side rate limiting
3. **Error Handling**: Handle 529 overload errors gracefully
4. **Monitoring**: Track latency and throughput metrics
5. **Fallback**: Have backup provider for high-availability needs
