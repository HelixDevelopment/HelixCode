# Provider Feature Matrix

This document provides a comprehensive overview of which advanced features are supported by each LLM provider in HelixCode.

## Feature Support Matrix

| Provider | Prompt Caching | Reasoning/Thinking | Token Budgets | Notes |
|----------|---------------|-------------------|---------------|-------|
| **Anthropic** | ✅ Full | ✅ Extended Thinking | ✅ Full | Supports ephemeral cache, thinking budgets |
| **OpenAI** | ⚠️ Planned | ✅ O-series (o1/o3) | ✅ Full | Reasoning on o1/o3/o4 models |
| **Azure** | ⚠️ Planned | ✅ O-series support | ✅ Full | Azure OpenAI Service compatibility |
| **Gemini** | ⚠️ Context caching | ✅ Native reasoning | ✅ Full | Google's context caching |
| **VertexAI** | ⚠️ Via Gemini | ✅ Via Gemini | ✅ Full | GCP-hosted AI models |
| **Bedrock** | ⚠️ Via provider | ✅ Model-dependent | ✅ Full | AWS-hosted models (Claude, etc.) |
| **Groq** | ❌ Not supported | ✅ Model-dependent | ✅ Full | Fast inference, limited caching |
| **Qwen** | ❌ Not supported | ✅ QwQ-32B | ✅ Full | Alibaba Cloud models |
| **xAI** | ❌ Not supported | ✅ Model-dependent | ✅ Full | Grok models |
| **OpenRouter** | ⚠️ Via upstream | ✅ Via upstream | ✅ Full | Aggregator, feature depends on model |
| **Copilot** | ❌ Not supported | ✅ GPT-4 based | ✅ Full | GitHub Copilot integration |
| **Local** | ❌ Not applicable | ✅ Model-dependent | ✅ Full | Ollama, llama.cpp |

**Legend:**
- ✅ **Full**: Feature is fully supported with native implementation
- ⚠️ **Planned/Limited**: Feature is planned or has limited support
- ❌ **Not supported**: Feature is not available for this provider
- **Via**: Feature available through underlying provider

## Feature Details

### 1. Prompt Caching

Prompt caching reduces costs by caching frequently used context (system prompts, tool definitions, conversation history).

#### Anthropic Claude
- **Support**: Full native support via `cache_control` directive
- **Strategies**: System, Tools, Context, Aggressive
- **Cost Savings**: Up to 90% reduction on cached input tokens
- **Configuration**:
  ```go
  cacheConfig := &llm.CacheConfig{
      Enabled: true,
      Strategy: llm.CacheStrategyTools,
      MinTokensForCache: 1024,
      CacheTTL: 300, // 5 minutes
  }
  ```
- **Minimum tokens**: 1024 tokens for efficient caching
- **TTL**: 5 minutes default (configurable)

#### OpenAI
- **Support**: Planned (not yet in public API)
- **Workaround**: Manual context management to reduce redundant tokens

#### Gemini/VertexAI
- **Support**: Context caching available
- **Configuration**: Via GCP console or API
- **Cost**: Reduced pricing for cached context

#### Azure OpenAI
- **Support**: Following OpenAI roadmap
- **Status**: Monitoring for official release

### 2. Reasoning/Thinking Mode

Advanced reasoning capabilities for complex problem-solving tasks.

#### Anthropic Claude
- **Support**: Extended thinking mode (claude-4-sonnet, claude-3-7-sonnet)
- **Configuration**:
  ```go
  reasoningConfig := &llm.ReasoningConfig{
      Enabled: true,
      ThinkingBudget: 5000, // tokens for thinking
      ExtractThinking: true,
      HideFromUser: false,
      ReasoningEffort: "medium", // low, medium, high
  }
  ```
- **Thinking Budget**: Separate token allocation for internal reasoning
- **Auto-detection**: Enabled automatically for thinking-related keywords
- **Extraction**: Thinking blocks can be separated from final output

#### OpenAI O-Series
- **Models**: o1, o1-preview, o1-mini, o3, o4 (planned)
- **Support**: Native reasoning models with automatic thinking
- **Configuration**:
  ```go
  reasoningConfig := llm.NewReasoningConfig(llm.ReasoningModelOpenAI_O1)
  ```
- **Thinking tokens**: Counted separately in usage
- **Cost**: Higher per-token cost for reasoning

#### DeepSeek R1
- **Support**: Via OpenRouter or compatible APIs
- **Thinking tags**: Uses `<think>` tags for reasoning
- **Configuration**: Automatic when model name contains "deepseek" and "r1"

#### QwQ-32B (Qwen)
- **Support**: Dedicated reasoning model
- **Configuration**: Auto-detected by model name
- **Budget**: 7000 tokens default for thinking

### 3. Token Budgets

Comprehensive token and cost tracking across all providers.

#### Features
- **Per-request limits**: Maximum tokens per single request
- **Session limits**: Total tokens across a session
- **Cost tracking**: USD cost estimation per provider
- **Rate limiting**: Requests per minute throttling
- **Warning thresholds**: Configurable alerts at X% usage

#### Configuration
```go
budget := llm.TokenBudget{
    MaxTokensPerRequest: 10000,
    MaxTokensPerSession: 100000,
    MaxCostPerSession: 10.0,  // $10 USD
    MaxCostPerDay: 50.0,       // $50 USD
    MaxRequestsPerMinute: 60,
    WarnThreshold: 80.0,       // Warn at 80%
}

pm := llm.NewProviderManagerWithBudget(config, budget)
```

#### Usage Tracking
```go
// Get session usage
usage, err := pm.GetSessionUsage(sessionID)
fmt.Printf("Tokens used: %d / %d\n", usage.TotalTokens, budget.MaxTokensPerSession)
fmt.Printf("Cost: $%.2f / $%.2f\n", usage.TotalCost, budget.MaxCostPerSession)

// Get budget status
status := pm.GetBudgetStatus(sessionID)
fmt.Printf("Token usage: %.1f%%\n", status.TokenUsagePercent)
fmt.Printf("Cost usage: %.1f%%\n", status.CostUsagePercent)
```

#### Cost Estimation
Costs are estimated based on published provider pricing:
- **Anthropic Claude 3.5 Sonnet**: $3/$15 per 1M tokens (input/output)
- **OpenAI GPT-4o**: $2.50/$10 per 1M tokens
- **OpenAI O1**: $15/$60 per 1M tokens (with thinking)
- **Gemini Pro**: $0.50/$1.50 per 1M tokens
- **Groq**: Often free tier available, then $0.10/$0.10 per 1M tokens

## Implementation Guide

### Updating LLMRequest

All new features are integrated via the `LLMRequest` structure:

```go
request := &llm.LLMRequest{
    Model: "claude-3-5-sonnet-20241022",
    Messages: messages,
    MaxTokens: 4096,

    // Reasoning configuration
    Reasoning: &llm.ReasoningConfig{
        Enabled: true,
        ThinkingBudget: 5000,
        ReasoningEffort: "medium",
    },

    // Cache configuration
    CacheConfig: &llm.CacheConfig{
        Enabled: true,
        Strategy: llm.CacheStrategyTools,
    },

    // Token budget (set session ID for tracking)
    SessionID: sessionID,
    ThinkingBudget: 5000, // Override reasoning budget
}

response, err := providerManager.Generate(ctx, request)
```

### Auto-Detection

HelixCode automatically detects reasoning models and applies appropriate configurations:

```go
// These model names trigger automatic reasoning mode:
// - OpenAI: "o1", "o1-preview", "o1-mini", "o3", "o4"
// - Anthropic: "claude-4-sonnet", "claude-3-7-sonnet", "opus" (with keywords)
// - DeepSeek: "deepseek-r1", "deepseek-reasoner"
// - Qwen: "qwq-32b"

request := &llm.LLMRequest{
    Model: "o1-preview", // Automatically enables reasoning
    Messages: messages,
}
```

### Cache Strategy Selection

Choose caching strategy based on use case:

- **CacheStrategyNone**: Disable caching
- **CacheStrategySystem**: Cache only system messages
- **CacheStrategyTools**: Cache system + tool definitions (recommended)
- **CacheStrategyContext**: Cache system + tools + recent messages
- **CacheStrategyAggressive**: Cache everything possible

### Budget Management

```go
// Initialize with budget
pm := llm.NewProviderManagerWithBudget(config, llm.DefaultTokenBudget())

// Track requests
request.SessionID = "user-session-123"
response, err := pm.Generate(ctx, request)

// Monitor usage
if status := pm.GetBudgetStatus(request.SessionID); status != nil {
    if status.TokenUsagePercent > 90 {
        log.Printf("Warning: %s approaching token limit", request.SessionID)
    }
}

// Reset session when done
pm.ResetSession(request.SessionID)
```

## Testing

Integration tests verify feature support across providers:

```bash
cd HelixCode
go test -v ./internal/llm -run TestReasoningFeatures
go test -v ./internal/llm -run TestCacheControl
go test -v ./internal/llm -run TestTokenBudgets
```

## Migration Notes

### Existing Code

Existing code continues to work without changes:

```go
// Old code - still works
request := &llm.LLMRequest{
    Model: "claude-3-5-sonnet-20241022",
    Messages: messages,
    MaxTokens: 4096,
}
response, err := provider.Generate(ctx, request)
```

### Opt-in Features

New features are opt-in via configuration:

```go
// New code - with advanced features
request := &llm.LLMRequest{
    Model: "claude-3-5-sonnet-20241022",
    Messages: messages,
    MaxTokens: 4096,
    Reasoning: llm.NewReasoningConfig(llm.ReasoningModelClaude_Sonnet),
    CacheConfig: &llm.CacheConfig{Enabled: true, Strategy: llm.CacheStrategyTools},
    SessionID: sessionID,
}
```

## Performance Impact

### Prompt Caching
- **First request**: ~5-10ms overhead to create cache
- **Cached requests**: 2-5x faster response time
- **Cost savings**: 80-90% on cached input tokens
- **Best for**: Multi-turn conversations, repeated system prompts

### Reasoning Mode
- **Latency**: 2-3x longer due to thinking phase
- **Quality**: Significantly better for complex tasks
- **Cost**: Higher token usage (thinking + output)
- **Best for**: Planning, analysis, complex problem-solving

### Token Budgets
- **Overhead**: <1ms per request for tracking
- **Memory**: ~1KB per session
- **Best for**: Multi-user systems, cost control

## Roadmap

### Planned Improvements

1. **Anthropic**
   - ✅ Full prompt caching
   - ✅ Extended thinking
   - ⬜ Batch API support
   - ⬜ Tool caching optimization

2. **OpenAI**
   - ⬜ Prompt caching (when available)
   - ✅ O-series reasoning
   - ⬜ Structured outputs
   - ⬜ Batch API integration

3. **Cross-Provider**
   - ⬜ Unified caching layer
   - ⬜ Cost optimization engine
   - ⬜ Automatic failover with feature preservation
   - ⬜ Provider-agnostic reasoning interface

### Experimental Features

- **Reasoning trace visualization**: Display thinking process to users
- **Dynamic budget allocation**: Adjust budgets based on task complexity
- **Multi-provider reasoning**: Combine reasoning from multiple models
- **Cache analytics dashboard**: Visualize cache hit rates and savings

## Support

For issues or questions about provider features:
- Create an issue on GitHub
- Check the integration tests for examples
- Review provider-specific documentation in `/docs/providers/`

## References

- [Anthropic Prompt Caching](https://docs.anthropic.com/claude/docs/prompt-caching)
- [OpenAI O-Series Models](https://openai.com/research/learning-to-reason)
- [Google Gemini Context Caching](https://ai.google.dev/docs/caching)
- [AWS Bedrock Documentation](https://docs.aws.amazon.com/bedrock/)
