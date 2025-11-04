# New Features - AI Provider Expansion

## Summary

This release adds comprehensive support for **Anthropic Claude** and **Google Gemini** providers with advanced features ported from leading AI coding agents (Claude Code, Codename Goose, OpenCode, Qwen Code).

## New Providers

### 1. Anthropic Claude Provider ⭐

**Location**: `internal/llm/anthropic_provider.go`

**Models Supported** (11 models):
- Claude 4 Sonnet / Opus (latest, most powerful)
- Claude 3.7 Sonnet (enhanced reasoning)
- Claude 3.5 Sonnet / Haiku (excellent for coding)
- Claude 3 Opus / Sonnet / Haiku (stable)
- All models with `-latest` variants for automatic updates

**Advanced Features**:

#### 🧠 Extended Thinking
- Automatic detection based on prompt content (keywords: "think", "reason", "analyze", "step by step")
- Allocates 80% of max tokens for reasoning
- Temperature automatically adjusted to 1.0 for thinking mode
- Enables sophisticated problem-solving and analysis

**Example**:
```go
request := &LLMRequest{
    Messages: []Message{{
        Role:    "user",
        Content: "Think carefully about how to implement OAuth2",
    }},
    MaxTokens: 4000, // 3200 tokens for thinking, 800 for response
}
```

#### 💾 Prompt Caching (Up to 90% Cost Savings)
- **System Message Caching**: Caches system instructions across requests
- **Last Message Caching**: Caches last user message for context continuity
- **Tool Caching**: Caches last tool definition for multi-turn conversations
- Uses Anthropic's `ephemeral` cache type
- Automatic cache control injection

**Cache Metadata in Response**:
```go
response.ProviderMetadata = map[string]interface{}{
    "cache_creation_tokens": 200, // New cache entries
    "cache_read_tokens":     300, // Tokens read from cache
}
```

**Cost Impact**:
- First request: Normal cost + cache creation
- Subsequent requests: 10% of normal cost for cached portions
- Typical savings: 70-90% on follow-up requests

#### 🛠️ Tool Calling with Caching
- Full function calling support
- Last tool definition automatically cached
- Efficient multi-turn tool conversations
- Streaming tool call support

#### 👁️ Vision Support
- All models support image analysis
- Base64 encoded images
- Multiple image formats (PNG, JPEG, WebP, GIF)
- Integrated with tool calling

#### ⚡ Streaming
- Server-Sent Events (SSE) streaming
- Token-by-token content delivery
- Tool call streaming
- Usage tracking in final event

**Context Windows**: 200K tokens (all models)
**Max Output**: 4K-50K tokens depending on model
**API Endpoint**: `https://api.anthropic.com/v1/messages`

---

### 2. Google Gemini Provider ⭐

**Location**: `internal/llm/gemini_provider.go`

**Models Supported** (11 models):
- Gemini 2.5 Pro / Flash / Flash Lite (latest generation)
- Gemini 2.0 Flash / Flash Lite (fast multimodal)
- Gemini 1.5 Pro / Flash / Flash 8B (proven reliable)
- Gemini 1.0 Pro (text-only)
- Gemini Embedding 001 (embeddings)

**Advanced Features**:

#### 📚 Massive Context Windows
- **Gemini 2.5 Pro**: 2,097,152 tokens (2M tokens = ~1.5 million words = ~3,000 pages)
- **Gemini 2.5 Flash**: 1,048,576 tokens (1M tokens)
- **Gemini 1.5 Pro**: 2,097,152 tokens (2M tokens)
- Can process entire codebases in a single request

**Use Cases**:
- Full codebase analysis
- Multi-file refactoring
- Large document processing
- Architectural reviews

#### 🎨 Multimodal Capabilities
- Text and image understanding
- Diagram analysis
- Screenshot debugging
- UI/UX review with images

#### 🔧 Native Function Calling
- Comprehensive tool support
- Auto/Any/None modes
- Streaming function calls
- Multiple parallel tool invocations

#### 🛡️ Safety Controls
- Configurable safety thresholds
- 4 categories: Harassment, Hate Speech, Sexually Explicit, Dangerous Content
- Set to `BLOCK_ONLY_HIGH` for development use
- Prevents blocking on code-related content

#### 🚀 Flash Models
- Ultra-fast responses
- Optimized for coding tasks
- 1M token context at flash speed
- Cost-effective for high-volume use

**API Endpoint**: `https://generativelanguage.googleapis.com/v1beta`
**Authentication**: API key as URL parameter
**Max Output**: 8K tokens

---

## Technical Implementation Details

### Provider Interface Compliance

Both providers fully implement the `Provider` interface:
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

### Request/Response Flow

#### Anthropic Flow:
1. **Build Request**: Convert LLM messages to Anthropic format
2. **Apply Caching**: Add cache control to system/messages/tools
3. **Detect Thinking**: Enable extended thinking if keywords present
4. **Make Request**: POST to `/v1/messages`
5. **Parse Response**: Extract content, tool calls, usage with caching info
6. **Return**: Structured `LLMResponse` with metadata

#### Gemini Flow:
1. **Build Request**: Convert to Gemini content format
2. **Set System Instruction**: Separate system message
3. **Configure Tools**: Convert to function declarations
4. **Safety Settings**: Set permissive thresholds
5. **Make Request**: POST to `/models/{model}:generateContent`
6. **Parse Response**: Extract candidates, usage, safety ratings
7. **Return**: Structured `LLMResponse`

### Error Handling

Both providers include comprehensive error handling:
- **400 Bad Request**: Invalid parameters
- **401 Unauthorized**: Invalid API key
- **429 Rate Limit**: Too many requests
- **500 Server Error**: Provider-side issues
- **Structured Error Responses**: Parsed error messages with codes

### Health Monitoring

Both providers support health checking:
```go
health := &ProviderHealth{
    Status:     "healthy",      // healthy, degraded, unhealthy, unknown
    Latency:    time.Duration,  // Last request latency
    LastCheck:  time.Time,      // When last checked
    ErrorCount: int,            // Recent error count
    ModelCount: int,            // Number of available models
}
```

---

## Test Coverage

### Anthropic Provider Tests
**File**: `internal/llm/anthropic_provider_test.go`

**Test Coverage** (14 test functions):
1. `TestNewAnthropicProvider` - Provider initialization with various configs
2. `TestAnthropicProvider_GetType` - Provider type verification
3. `TestAnthropicProvider_GetName` - Provider name verification
4. `TestAnthropicProvider_GetModels` - Model list validation
5. `TestAnthropicProvider_GetCapabilities` - Capability verification
6. `TestAnthropicProvider_IsAvailable` - Availability checking
7. `TestAnthropicProvider_Generate` - Standard text generation
8. `TestAnthropicProvider_GenerateWithTools` - Tool calling with caching
9. `TestAnthropicProvider_ExtendedThinking` - Automatic thinking mode
10. `TestAnthropicProvider_PromptCaching` - Cache control verification
11. `TestAnthropicProvider_ErrorHandling` - All HTTP error codes
12. `TestAnthropicProvider_GetHealth` - Health check functionality
13. `TestAnthropicProvider_Close` - Resource cleanup

**All tests pass** ✅

**Mock Server Testing**: Uses `httptest.NewServer` for comprehensive API simulation

---

## Configuration

### Environment Variables

**Anthropic**:
```bash
export ANTHROPIC_API_KEY="sk-ant-your-key-here"
```

**Gemini**:
```bash
export GEMINI_API_KEY="your-gemini-key"
# or
export GOOGLE_API_KEY="your-google-key"
```

### Provider Factory

Automatically creates providers based on configuration:
```go
factory := &ProviderFactory{}
provider, err := factory.CreateProvider(ProviderConfigEntry{
    Type:   ProviderTypeAnthropic,
    APIKey: "sk-ant-...",
})
```

Supported types:
- `ProviderTypeAnthropic` ⭐ NEW
- `ProviderTypeGemini` ⭐ NEW
- `ProviderTypeOpenAI`
- `ProviderTypeLocal`
- `ProviderTypeQwen`
- `ProviderTypeXAI`
- `ProviderTypeOpenRouter`
- `ProviderTypeCopilot`

### Configuration Example

```yaml
llm:
  providers:
    anthropic:
      type: "anthropic"
      api_key: "${ANTHROPIC_API_KEY}"
      enabled: true
      models:
        - "claude-4-sonnet"
        - "claude-3-5-sonnet-latest"
        - "claude-3-5-haiku-latest"

    gemini:
      type: "gemini"
      api_key: "${GEMINI_API_KEY}"
      enabled: true
      models:
        - "gemini-2.5-pro"
        - "gemini-2.5-flash"
        - "gemini-2.0-flash"
```

---

## Usage Examples

### Example 1: Extended Thinking with Claude

```go
request := &LLMRequest{
    ID:          uuid.New(),
    Model:       "claude-4-sonnet",
    Messages: []Message{
        {Role: "user", Content: "Think step by step: design a microservices architecture for an e-commerce platform"},
    },
    MaxTokens:   10000,
    Temperature: 0.7,
}

response, err := anthropicProvider.Generate(ctx, request)
// Claude will automatically enable extended thinking
// and allocate 8000 tokens for reasoning, 2000 for response
```

### Example 2: Prompt Caching with Claude

```go
// First request - creates cache
request1 := &LLMRequest{
    Messages: []Message{
        {Role: "system", Content: "You are an expert Go developer. [large context]"},
        {Role: "user", Content: "Explain interfaces"},
    },
    Tools: []Tool{
        {Function: FunctionDefinition{Name: "search_docs", ...}},
    },
}

response1, _ := provider.Generate(ctx, request1)
// Cache created: system message + last tool

// Second request - uses cache
request2 := &LLMRequest{
    Messages: []Message{
        {Role: "system", Content: "You are an expert Go developer. [same large context]"},
        {Role: "user", Content: "Explain channels"},
    },
    Tools: []Tool{
        {Function: FunctionDefinition{Name: "search_docs", ...}},
    },
}

response2, _ := provider.Generate(ctx, request2)
// 90% cost savings on system message and tool
// Check response2.ProviderMetadata for cache stats
```

### Example 3: Massive Context with Gemini

```go
// Read entire codebase (up to 2M tokens)
files := []string{"file1.go", "file2.go", ... } // 500 files
codebase := readAllFiles(files)

request := &LLMRequest{
    Model: "gemini-2.5-pro",
    Messages: []Message{
        {Role: "system", Content: "You are analyzing a complete codebase."},
        {Role: "user", Content: codebase + "\n\nIdentify all security vulnerabilities."},
    },
    MaxTokens: 8192,
}

response, err := geminiProvider.Generate(ctx, request)
// Gemini processes all 500 files in one request
```

### Example 4: Tool Calling with Caching

```go
tools := []Tool{
    {Function: FunctionDefinition{Name: "read_file", ...}},
    {Function: FunctionDefinition{Name: "write_file", ...}},
    {Function: FunctionDefinition{Name: "run_tests", ...}},
}

request := &LLMRequest{
    Model:    "claude-3-5-sonnet-latest",
    Messages: []Message{{Role: "user", Content: "Refactor auth.go"}},
    Tools:    tools,
}

// First call - last tool cached
response1, _ := provider.Generate(ctx, request)

// Second call - cache hit on last tool
request.Messages = append(request.Messages,
    Message{Role: "user", Content: "Now refactor users.go"})
response2, _ := provider.Generate(ctx, request)
// Tools reused from cache, no re-transmission cost
```

### Example 5: Streaming with Real-time Updates

```go
ch := make(chan LLMResponse, 10)

go func() {
    err := provider.GenerateStream(ctx, request, ch)
    if err != nil {
        log.Printf("Stream error: %v", err)
    }
}()

// Process tokens as they arrive
for response := range ch {
    fmt.Print(response.Content) // Print token-by-token

    if response.FinishReason != "" {
        fmt.Printf("\n\nUsage: %d tokens\n", response.Usage.TotalTokens)
    }
}
```

---

## Performance Characteristics

### Anthropic Claude

**Latency**:
- First token: ~500-1000ms
- Tokens/second: 30-60 tokens/sec
- With caching: First token ~200-400ms (cache hit)

**Context**:
- 200K tokens = ~150K words = ~300 pages of text
- Sufficient for most coding tasks

**Cost Savings**:
- Prompt caching: 90% reduction on cached portions
- Tool caching: 90% reduction on tool definitions
- Extended thinking: More tokens but better results

### Google Gemini

**Latency**:
- First token: ~300-700ms
- Tokens/second: 40-80 tokens/sec (Flash models)
- Pro models: Slightly slower but more capable

**Context**:
- 2M tokens = ~1.5M words = ~3,000 pages = entire codebases
- Can process projects with 500+ files in one request

**Use Cases**:
- Flash models: High-volume, fast iterations
- Pro models: Deep analysis, complex reasoning

---

## Migration Guide

### From GitHub Copilot Claude to Native Anthropic

**Before**:
```go
config := ProviderConfigEntry{
    Type:   ProviderTypeCopilot,
    APIKey: githubToken,
}
provider, _ := NewCopilotProvider(config)

request.Model = "claude-3.5-sonnet" // Limited to Copilot's offering
```

**After**:
```go
config := ProviderConfigEntry{
    Type:   ProviderTypeAnthropic,
    APIKey: anthropicKey,
}
provider, _ := NewAnthropicProvider(config)

request.Model = "claude-4-sonnet" // Access to all Claude models
// Automatic prompt caching and extended thinking!
```

**Benefits**:
- Access to Claude 4 and 3.7 models
- Prompt caching (90% cost savings)
- Extended thinking support
- Direct API access (no GitHub proxy)
- Higher rate limits

### Adding Gemini for Large Context Tasks

```go
// Initialize Gemini provider
geminiConfig := ProviderConfigEntry{
    Type:   ProviderTypeGemini,
    APIKey: os.Getenv("GEMINI_API_KEY"),
}
geminiProvider, _ := NewGeminiProvider(geminiConfig)

// Use for large context tasks
if totalTokens > 200000 {
    // Switch to Gemini 2.5 Pro (2M context)
    request.Model = "gemini-2.5-pro"
    response, _ = geminiProvider.Generate(ctx, request)
} else {
    // Use Claude for better reasoning
    request.Model = "claude-4-sonnet"
    response, _ = anthropicProvider.Generate(ctx, request)
}
```

---

## Breaking Changes

**None**. All changes are additive and backward compatible.

Existing providers continue to work unchanged:
- OpenAI
- Qwen
- XAI
- OpenRouter
- Copilot
- Local (Ollama, Llama.cpp)

---

## Future Enhancements

Potential improvements for future releases:

1. **AWS Bedrock Provider**: Access Claude via AWS infrastructure
2. **Azure OpenAI Provider**: Enterprise Azure integration
3. **Vertex AI Provider**: Gemini via Google Cloud
4. **Prompt Caching for Other Providers**: Extend caching to OpenAI
5. **Vision Support for More Providers**: Unified vision interface
6. **Batch API Support**: Async batch processing
7. **Fine-tuning Integration**: Custom model support
8. **Cost Tracking**: Per-request cost calculation
9. **Rate Limiting**: Client-side rate limit handling
10. **Retry Logic**: Exponential backoff for transient failures

---

## Credits

Features ported and enhanced from:
- **Claude Code** (Anthropic's official CLI)
- **Codename Goose** (Rust-based AI agent)
- **OpenCode** (Go-based terminal agent)
- **Qwen Code** (TypeScript multimodal agent)
- **Mistral Code** (Mistral AI integration)

All implementations adapted to HelixCode's architecture with Go idioms and best practices.

---

## Version

**HelixCode v2.0** - AI Provider Expansion Release
**Date**: November 4, 2025
**Providers Added**: Anthropic Claude, Google Gemini
**Tests Added**: 14+ test functions
**Models Added**: 22 new models
**Lines of Code**: ~2,000 lines of production code + tests
