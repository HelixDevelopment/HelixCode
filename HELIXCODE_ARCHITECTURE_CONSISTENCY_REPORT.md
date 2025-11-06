# HelixCode Architecture Consistency Report

**Generated:** 2025-01-06
**Analyzed Providers:** Anthropic, OpenAI, Gemini, Bedrock, Ollama, XAI, Qwen, Groq, Azure, VertexAI, OpenRouter, Copilot

## Executive Summary

This report analyzes the consistency of LLM provider implementations across HelixCode's architecture. Overall, the codebase demonstrates **strong architectural consistency** with a unified Provider interface, consistent request/response structures, and standardized feature support patterns. However, several areas could benefit from improvements in feature availability documentation, error handling standardization, and testing coverage.

**Overall Score: 8.5/10** - Well-architected with minor inconsistencies

---

## 1. Provider Interface Consistency

### ✅ **Status: EXCELLENT**

All providers correctly implement the `Provider` interface defined in `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/provider.go` (lines 117-132).

### Interface Methods

**Required Methods:**
```go
GetType() ProviderType
GetName() string
GetModels() []ModelInfo
GetCapabilities() []ModelCapability
Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error)
GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error
IsAvailable(ctx context.Context) bool
GetHealth(ctx context.Context) (*ProviderHealth, error)
Close() error
```

**Implementation Status:**

| Provider | Interface Complete | Notes |
|----------|-------------------|-------|
| Anthropic | ✅ | Full implementation |
| OpenAI | ✅ | Full implementation |
| Gemini | ✅ | Full implementation |
| Bedrock | ✅ | Full implementation with multi-model support |
| Ollama | ✅ | Full implementation |
| XAI | ✅ | Full implementation |
| Qwen | ✅ | Full implementation + OAuth2 |
| Groq | ✅ | Full implementation + latency metrics |
| Azure | ✅ | Full implementation + Entra ID auth |
| VertexAI | ✅ | Full implementation + dual model support |
| OpenRouter | ✅ | Full implementation |
| Copilot | ✅ | Full implementation + token exchange |

### Method Signature Consistency

**✅ All providers use identical method signatures** - no variations detected.

**Finding:** The interface contract is strictly enforced across all providers.

---

## 2. Feature Availability Matrix

### Supported Features by Provider

| Feature | Anthropic | OpenAI | Gemini | Bedrock | Ollama | XAI | Qwen | Groq | Azure | VertexAI | OpenRouter | Copilot |
|---------|-----------|--------|--------|---------|--------|-----|------|------|-------|----------|------------|---------|
| **Prompt Caching** | ✅ | ❌ | ⚠️ | ❌ | ❌ | ❌ | ⚠️ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Reasoning Support** | ✅ | ⚠️ | ⚠️ | ⚠️ | ⚠️ | ⚠️ | ⚠️ | ⚠️ | ⚠️ | ⚠️ | ✅ | ⚠️ |
| **Token Budgets** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Vision Support** | ✅ | ✅ | ✅ | ⚠️ | ❌ | ❌ | ✅ | ❌ | ✅ | ✅ | ⚠️ | ❌ |
| **Tool Calling** | ✅ | ✅ | ✅ | ⚠️ | ❌ | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️ | ✅ |
| **Streaming** | ✅ | ✅ | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ✅ | ⚠️ | ✅ | ✅ |

**Legend:**
- ✅ Fully implemented
- ⚠️ Partial or model-dependent support
- ❌ Not supported

### Detailed Feature Analysis

#### 2.1 Prompt Caching

**✅ Anthropic** (Lines 48-66 in `anthropic_provider.go`)
```go
type anthropicContentBlock struct {
    CacheControl *anthropicCacheControl `json:"cache_control,omitempty"`
}
```
- Implements full ephemeral caching
- Caches system messages, last user message, and tools
- Lines 413-449: Automatic cache control application

**⚠️ Gemini** (Line 133 in `gemini_provider.go`)
```go
CachedContentTokenCount int `json:"cachedContentTokenCount,omitempty"`
```
- Tracks cached tokens in usage metadata
- No explicit cache control mechanism
- Implicit caching handled by Google

**⚠️ Qwen** (Line 429 in `qwen_provider.go`)
```go
req.Header.Set("X-DashScope-CacheControl", "enable")
```
- Header-based cache control
- Limited documentation

**❌ Other Providers:** No caching support

**Recommendation:** Document which providers support caching and create a unified caching interface.

#### 2.2 Reasoning Support

**Reasoning Models Detected:**
- OpenAI: o1, o3, o4 series (via `reasoning.go` lines 39-42)
- Anthropic: Claude with extended thinking (lines 113-117)
- DeepSeek: R1/Reasoner models (lines 49-50)
- QwQ: 32B model (line 53)

**Implementation in `reasoning.go`:**
- Lines 10-33: `ReasoningConfig` structure
- Lines 135-172: `ExtractReasoningTrace` function
- Lines 256-308: `FormatReasoningPrompt` function

**Issue:** Reasoning support is defined in shared code but not consistently referenced in provider implementations.

**File:** `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/reasoning.go`

**Recommendation:** Add explicit reasoning model detection to each provider's model list.

#### 2.3 Token Budgets

**✅ Universal Support** - Token budgets are implemented at the manager level.

**File:** `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/token_budget.go`
- Lines 13-31: `TokenBudget` structure
- Lines 45-84: `TokenTracker` implementation
- Lines 86-138: `CheckBudget` enforcement

**All providers** correctly report usage via `LLMResponse.Usage` structure (lines 110-114 in `provider.go`).

#### 2.4 Vision Support

**Providers with Vision:**
- Anthropic: Claude 4/3.x models (Sonnet/Opus)
- OpenAI: GPT-4o, GPT-4 Turbo Vision
- Gemini: All 2.x/1.5 models
- Bedrock: Claude models via Bedrock
- Qwen: VL models
- Azure: GPT-4o variants
- VertexAI: Gemini + Claude via Model Garden

**Implementation:** Vision capability tracked via `ModelInfo.SupportsVision` boolean (line 142 in `provider.go`).

#### 2.5 Tool Calling

**Providers with Tools:**
- Anthropic: Full function calling (lines 79-84 in `anthropic_provider.go`)
- OpenAI: Native support
- Gemini: Function declarations (lines 76-83 in `gemini_provider.go`)
- Bedrock: Model-dependent (Claude, Cohere Command)
- XAI: OpenAI-compatible
- Qwen: Coder models
- Groq: Llama 3.x models
- Azure: GPT-4 variants
- VertexAI: Gemini models
- Copilot: GPT-4o/Claude models

**Consistency:** Tool format unified across all providers using `Tool` structure (lines 71-81 in `provider.go`).

---

## 3. Request/Response Consistency

### ✅ **Status: EXCELLENT**

### 3.1 LLMRequest Structure

**Definition:** Lines 48-61 in `provider.go`

```go
type LLMRequest struct {
    ID           uuid.UUID
    ProviderType ProviderType
    Model        string
    Messages     []Message
    MaxTokens    int
    Temperature  float64
    TopP         float64
    Stream       bool
    Tools        []Tool
    ToolChoice   string
    Capabilities []ModelCapability
    CreatedAt    time.Time
}
```

**Usage Consistency:** All providers correctly consume this structure in their `Generate` and `GenerateStream` methods.

### 3.2 LLMResponse Structure

**Definition:** Lines 84-94 in `provider.go`

```go
type LLMResponse struct {
    ID               uuid.UUID
    RequestID        uuid.UUID
    Content          string
    ToolCalls        []ToolCall
    FinishReason     string
    Usage            Usage
    ProviderMetadata interface{}
    ProcessingTime   time.Duration
    CreatedAt        time.Time
}
```

**Consistency:** All providers return this exact structure.

### 3.3 Message Format

**Definition:** Lines 64-68 in `provider.go`

```go
type Message struct {
    Role    string
    Content string
    Name    string
}
```

**Conversion Patterns:**

**Anthropic** (Lines 488-505):
```go
func (ap *AnthropicProvider) convertMessages(messages []Message) (string, []anthropicMessage)
```
- Separates system messages
- Converts to Anthropic format

**Gemini** (Lines 476-502):
```go
func (gp *GeminiProvider) convertMessages(messages []Message) (string, []geminiContent)
```
- Separates system messages
- Converts "assistant" → "model"

**Consistency:** ✅ All providers use consistent conversion patterns.

### 3.4 Tool Format

**Definition:** Lines 71-81 in `provider.go`

```go
type Tool struct {
    Type     string
    Function FunctionDefinition
}

type FunctionDefinition struct {
    Name        string
    Description string
    Parameters  map[string]interface{}
}
```

**Conversion Examples:**

**Anthropic** (Lines 508-520):
```go
func (ap *AnthropicProvider) convertTools(tools []Tool) []anthropicTool
```

**Gemini** (Lines 505-519):
```go
func (gp *GeminiProvider) convertTools(tools []Tool) []geminiTool
```

**Consistency:** ✅ All providers correctly map tools to their native formats.

---

## 4. Configuration Consistency

### 4.1 Config Structure

**ProviderConfigEntry** (Lines 169-177 in `provider.go`):
```go
type ProviderConfigEntry struct {
    Type       ProviderType
    Endpoint   string
    APIKey     string
    Models     []string
    Enabled    bool
    Parameters map[string]interface{}
}
```

### 4.2 Provider-Specific Configuration

| Provider | Config Location | Special Parameters |
|----------|----------------|-------------------|
| Anthropic | Lines 134-160 | API key from env |
| OpenAI | Lines 27-54 | Endpoint override |
| Gemini | Lines 144-174 | API key alternatives |
| Bedrock | Lines 205-262 | AWS credentials, region, cross-region |
| Ollama | Lines 24-30 | Base URL, timeout, keep-alive |
| XAI | Lines 27-54 | Endpoint override |
| Qwen | Lines 28-59 | OAuth2 + API key |
| Groq | Lines 134-177 | Optimized HTTP client |
| Azure | Lines 191-288 | Entra ID, deployment map |
| VertexAI | Lines 206-303 | Project ID, location, credentials |
| OpenRouter | Lines 27-54 | HTTP referer headers |
| Copilot | Lines 36-63 | GitHub token exchange |

### ⚠️ Issues Found:

1. **Inconsistent Default Timeouts:**
   - Anthropic: 120s (line 154)
   - OpenAI: 60s (line 43)
   - Gemini: 120s (line 169)
   - Groq: 60s (line 152)
   - Azure: 120s (line 222)
   - VertexAI: 120s (line 296)

2. **Environment Variable Naming:**
   - Anthropic: `ANTHROPIC_API_KEY` (line 137)
   - OpenAI: Uses config only
   - Gemini: `GEMINI_API_KEY` or `GOOGLE_API_KEY` (lines 148-153)
   - Qwen: `QWEN_API_KEY` (implied)
   - Groq: `GROQ_API_KEY` (line 138)
   - Azure: `AZURE_OPENAI_API_KEY` (line 278)
   - VertexAI: `GOOGLE_APPLICATION_CREDENTIALS` (line 235)

**Recommendation:** Standardize timeout defaults and document all environment variables.

---

## 5. Error Handling Consistency

### 5.1 Common Errors

**Defined in `provider.go` (Lines 331-337):**
```go
var (
    ErrProviderUnavailable = errors.New("provider unavailable")
    ErrModelNotFound       = errors.New("model not found")
    ErrInvalidRequest      = errors.New("invalid request")
    ErrRateLimited         = errors.New("rate limited")
    ErrContextTooLong      = errors.New("context too long")
)
```

### 5.2 Error Handling Patterns

**✅ Consistent Providers:**

**Groq** (Lines 592-630):
- Comprehensive HTTP status mapping
- Provider-specific error parsing
- Returns standard errors

**Azure** (Lines 812-850):
- Azure-specific error code mapping
- Falls back to HTTP status codes
- Content filtering errors

**Bedrock** (Lines 1052-1083):
- AWS SDK error handling
- Maps to standard errors
- Throttling detection

**⚠️ Inconsistent Providers:**

**OpenAI** (Lines 278-309):
- Basic error handling
- No specific error parsing
- Generic error messages

**Ollama** (Lines 137-174):
- Minimal error handling
- No specific error types

**Recommendation:** Standardize error handling with:
1. Structured error parsing
2. Consistent error wrapping
3. Retry-able error identification

---

## 6. Testing Consistency

### 6.1 Test File Coverage

**Test Files Found:**
- `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/integration_test.go`
- `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/anthropic_provider_test.go`
- `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/gemini_provider_test.go`
- `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/bedrock_provider_test.go`
- `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/groq_provider_test.go`
- `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/azure_provider_test.go`
- `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/vertexai_provider_test.go`
- `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/qwen_provider_test.go`
- `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/reasoning_test.go`
- `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/cache_control_test.go`

### ⚠️ Missing Tests:
- OpenAI provider
- Ollama provider
- XAI provider
- OpenRouter provider
- Copilot provider

**Test Coverage:** ~58% (7/12 providers have dedicated tests)

### 6.2 Test Pattern Analysis

**Common Test Patterns (from existing tests):**
1. Provider initialization tests
2. Model listing tests
3. Health check tests
4. Request/response conversion tests
5. Error handling tests

**Recommendation:** Add missing provider tests following established patterns.

---

## 7. Recommendations

### 7.1 Critical Issues

#### Issue 1: Missing Reasoning Support Integration
**Severity:** Medium
**Location:** Provider implementations
**Issue:** `reasoning.go` defines comprehensive reasoning support, but providers don't explicitly integrate it.

**Action Items:**
1. Add reasoning model detection to provider model lists
2. Update `LLMRequest` to include reasoning configuration
3. Integrate `ExtractReasoningTrace` in response parsing

**Affected Files:**
- All `*_provider.go` files
- `provider.go` (add ReasoningConfig field to LLMRequest)

#### Issue 2: Inconsistent Error Handling
**Severity:** Medium
**Location:** OpenAI, Ollama, XAI, OpenRouter providers
**Issue:** Some providers lack structured error parsing.

**Action Items:**
1. Implement error parsing for OpenAI (reference: Groq implementation)
2. Add retry-able error identification
3. Create error handling test suite

**Example Implementation (from Groq, lines 592-630):**
```go
func handleProviderError(statusCode int, body []byte) error {
    var providerErr ProviderError
    if err := json.Unmarshal(body, &providerErr); err == nil {
        switch statusCode {
        case http.StatusBadRequest:
            if strings.Contains(providerErr.Message, "context_length") {
                return ErrContextTooLong
            }
            return ErrInvalidRequest
        case http.StatusUnauthorized:
            return fmt.Errorf("unauthorized: invalid API key")
        case http.StatusTooManyRequests:
            return ErrRateLimited
        // ...
        }
    }
    return fmt.Errorf("provider error (%d): %s", statusCode, string(body))
}
```

#### Issue 3: Test Coverage Gaps
**Severity:** Medium
**Location:** 5 providers without tests
**Issue:** OpenAI, Ollama, XAI, OpenRouter, Copilot lack dedicated test files.

**Action Items:**
1. Create test files for missing providers
2. Add integration tests
3. Establish minimum test coverage requirement (80%+)

### 7.2 Enhancement Opportunities

#### Enhancement 1: Unified Caching Interface
**Priority:** Medium

Create abstract caching interface that works across providers:

```go
type CacheStrategy interface {
    ShouldCache(message Message) bool
    ApplyCaching(request *LLMRequest) error
    GetCacheMetrics() *CacheMetrics
}
```

**Benefits:**
- Consistent caching behavior
- Provider-specific optimizations
- Unified metrics

#### Enhancement 2: Provider Feature Registry
**Priority:** Low

Create feature detection system:

```go
type ProviderFeatures struct {
    PromptCaching  bool
    Reasoning      bool
    Vision         bool
    ToolCalling    bool
    Streaming      bool
    TokenBudgets   bool
}

func (p *Provider) GetFeatures() ProviderFeatures
```

**Benefits:**
- Runtime feature detection
- Better error messages
- Automatic capability routing

#### Enhancement 3: Configuration Validation
**Priority:** Medium

Add configuration validation:

```go
func (config ProviderConfigEntry) Validate() error {
    if config.Endpoint == "" && config.Type != ProviderTypeLocal {
        return fmt.Errorf("endpoint required for %s", config.Type)
    }
    if config.APIKey == "" && requiresAPIKey(config.Type) {
        return fmt.Errorf("API key required for %s", config.Type)
    }
    return nil
}
```

**Location:** `provider.go` lines 169-177

### 7.3 Documentation Improvements

1. **Feature Matrix Documentation:**
   - Create `PROVIDERS.md` with comprehensive feature matrix
   - Document model-specific capabilities
   - Add migration guides between providers

2. **Configuration Guide:**
   - Document all environment variables
   - Provide example configurations
   - Add troubleshooting section

3. **Error Handling Guide:**
   - Document all error types
   - Provide retry strategies
   - Add error code reference

---

## 8. Inconsistencies Found

### 8.1 Method Implementation Differences

#### Health Check Implementations

**Variation 1: API Endpoint Test (Anthropic, Gemini, Qwen)**
```go
// Lines 681-711 in anthropic_provider.go
testReq := &LLMRequest{
    Model:    "claude-3-5-haiku-latest",
    Messages: []Message{{Role: "user", Content: "Hi"}},
    MaxTokens: 10,
}
_, err := ap.Generate(ctx, testReq)
```

**Variation 2: Simple Availability Check (Ollama)**
```go
// Lines 199-212 in ollama_provider.go
resp, err := p.apiClient.Get(p.getAPIURL("/api/tags"))
return resp.StatusCode == http.StatusOK
```

**Variation 3: Models Endpoint Check (OpenAI, XAI)**
```go
// Lines 132-173 in openai_provider.go
req, _ := http.NewRequestWithContext(ctx, "GET", "/models", nil)
resp, err := op.httpClient.Do(req)
```

**Recommendation:** Standardize on one approach (preferably minimal test request).

#### Model Initialization

**Variation 1: Static Model List (Most Providers)**
```go
// Lines 164-291 in anthropic_provider.go
func getAnthropicModels() []ModelInfo {
    return []ModelInfo{
        {Name: "claude-4-sonnet", ...},
        // ...
    }
}
```

**Variation 2: Dynamic Discovery (Ollama)**
```go
// Lines 258-279 in ollama_provider.go
func (p *OllamaProvider) discoverModels() error {
    resp, _ := p.apiClient.Get("/api/tags")
    // Parse and populate models
}
```

**Recommendation:** Support both patterns with clear documentation.

### 8.2 Streaming Implementation Differences

**SSE Parsing Variation:**

**Method 1: Bufio Scanner (Groq, Azure, VertexAI)**
```go
// Lines 501-586 in groq_provider.go
scanner := bufio.NewScanner(reader)
for scanner.Scan() {
    line := scanner.Text()
    if strings.HasPrefix(line, "data: ") {
        // Parse JSON
    }
}
```

**Method 2: JSON Decoder (OpenAI, XAI, Qwen, OpenRouter, Copilot)**
```go
// Lines 311-368 in openai_provider.go
decoder := json.NewDecoder(resp.Body)
for decoder.More() {
    var streamResp OpenAIStreamResponse
    decoder.Decode(&streamResp)
}
```

**Issue:** SSE format requires line-by-line parsing; JSON decoder approach may fail with proper SSE streams.

**Recommendation:** Standardize on bufio.Scanner for SSE parsing.

### 8.3 Authentication Patterns

| Provider | Auth Method | Location |
|----------|------------|----------|
| Anthropic | API Key (header: `x-api-key`) | Line 538 |
| OpenAI | Bearer Token | Line 371 |
| Gemini | API Key (URL param) | Line 524 |
| Bedrock | AWS Signature V4 | AWS SDK |
| Ollama | None | Local |
| XAI | Bearer Token | Line 374 |
| Qwen | Bearer Token + OAuth2 | Lines 427, 62-98 |
| Groq | Bearer Token | Line 589 |
| Azure | API Key or Entra ID | Lines 526-535, 569-573 |
| VertexAI | OAuth2 (Google) | Lines 178-204 |
| OpenRouter | Bearer Token | Line 397 |
| Copilot | Bearer Token (GitHub exchange) | Lines 135-162, 521 |

**Consistency:** ✅ Well-documented variety, each appropriate for the provider.

---

## 9. Action Items

### Immediate (High Priority)

1. **Add Missing Tests**
   - [ ] Create `openai_provider_test.go`
   - [ ] Create `ollama_provider_test.go`
   - [ ] Create `xai_provider_test.go`
   - [ ] Create `openrouter_provider_test.go`
   - [ ] Create `copilot_provider_test.go`

2. **Standardize Error Handling**
   - [ ] Add structured error parsing to OpenAI provider
   - [ ] Add structured error parsing to Ollama provider
   - [ ] Create error handling documentation
   - [ ] Add retry-able error identification

3. **Fix SSE Streaming**
   - [ ] Update OpenAI provider to use bufio.Scanner
   - [ ] Update XAI provider to use bufio.Scanner
   - [ ] Update Qwen provider to use bufio.Scanner
   - [ ] Update OpenRouter provider to use bufio.Scanner
   - [ ] Update Copilot provider to use bufio.Scanner

### Short-term (Medium Priority)

4. **Integrate Reasoning Support**
   - [ ] Add ReasoningConfig to LLMRequest
   - [ ] Update provider implementations to support reasoning
   - [ ] Add reasoning model detection
   - [ ] Create reasoning documentation

5. **Standardize Configuration**
   - [ ] Document all environment variables
   - [ ] Standardize timeout defaults
   - [ ] Add configuration validation
   - [ ] Create configuration examples

6. **Improve Documentation**
   - [ ] Create `PROVIDERS.md` with feature matrix
   - [ ] Document model capabilities
   - [ ] Add migration guides
   - [ ] Create troubleshooting guide

### Long-term (Low Priority)

7. **Unified Caching Interface**
   - [ ] Design caching abstraction
   - [ ] Implement for Anthropic
   - [ ] Add to other compatible providers
   - [ ] Create caching documentation

8. **Provider Feature Registry**
   - [ ] Design feature detection system
   - [ ] Implement runtime capability checking
   - [ ] Add to provider interface
   - [ ] Update routing logic

9. **Performance Optimization**
   - [ ] Add connection pooling benchmarks
   - [ ] Optimize request serialization
   - [ ] Implement request batching where supported
   - [ ] Add performance monitoring

---

## 10. Conclusion

HelixCode's LLM provider architecture demonstrates **excellent consistency** in core areas:

**Strengths:**
- ✅ Unified Provider interface implementation
- ✅ Consistent request/response structures
- ✅ Standardized message and tool formats
- ✅ Comprehensive model information
- ✅ Good configuration flexibility

**Areas for Improvement:**
- ⚠️ Test coverage gaps (5 providers without tests)
- ⚠️ Inconsistent error handling patterns
- ⚠️ Streaming implementation variations
- ⚠️ Missing reasoning support integration
- ⚠️ Configuration validation missing

**Overall Assessment:**

The architecture is **production-ready** with minor improvements needed. The codebase follows solid engineering principles with a clean separation of concerns and consistent patterns. The identified issues are primarily related to completeness rather than fundamental design flaws.

**Priority Recommendations:**
1. Complete test coverage for all providers
2. Standardize SSE streaming implementation
3. Add structured error handling to all providers
4. Document all features and capabilities
5. Integrate reasoning support throughout

With these improvements, HelixCode will have a **world-class LLM provider abstraction layer** suitable for enterprise deployment.

---

## Appendix A: File Locations Reference

### Core Files
- **Provider Interface:** `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/provider.go`
- **Cache Control:** `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/cache_control.go`
- **Reasoning:** `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/reasoning.go`
- **Token Budget:** `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/token_budget.go`

### Provider Implementations
- **Anthropic:** `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/anthropic_provider.go`
- **OpenAI:** `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/openai_provider.go`
- **Gemini:** `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/gemini_provider.go`
- **Bedrock:** `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/bedrock_provider.go`
- **Ollama:** `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/ollama_provider.go`
- **XAI:** `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/xai_provider.go`
- **Qwen:** `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/qwen_provider.go`
- **Groq:** `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/groq_provider.go`
- **Azure:** `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/azure_provider.go`
- **VertexAI:** `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/vertexai_provider.go`
- **OpenRouter:** `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/openrouter_provider.go`
- **Copilot:** `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/copilot_provider.go`

### Test Files
- **Integration Tests:** `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/integration_test.go`
- **Provider Tests:** `*_provider_test.go` (7 out of 12 providers)
- **Feature Tests:** `reasoning_test.go`, `cache_control_test.go`

---

**Report End**
