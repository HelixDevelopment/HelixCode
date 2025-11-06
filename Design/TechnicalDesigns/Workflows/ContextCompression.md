# Conversation Context Compression - Technical Design

## Overview

Context compression manages conversation history by automatically summarizing and compressing older messages to stay within token budgets while preserving semantic meaning and important context.

## Architecture

### Component Diagram

```
┌────────────────────────────────────────────────────────────┐
│              CompressionCoordinator                         │
│  - Manages compression lifecycle                            │
│  - Token budget tracking                                    │
└────────────┬───────────────────────────────┬───────────────┘
             │                               │
             ▼                               ▼
┌────────────────────────┐      ┌────────────────────────┐
│   CompressionEngine    │      │    TokenCounter        │
│  - Strategy execution  │      │  - Token counting      │
│  - Window management   │      │  - Budget tracking     │
└──────────┬─────────────┘      └────────────────────────┘
           │
           ├──────────────┬──────────────┬────────────────┐
           ▼              ▼              ▼                ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│   Sliding    │  │  Semantic    │  │   Hybrid     │  │   Custom     │
│   Window     │  │Summarization │  │  Strategy    │  │  Strategy    │
└──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘
           │              │              │                │
           └──────────────┴──────────────┴────────────────┘
                          ▼
                  ┌──────────────┐
                  │RetentionPolicy│
                  │  - Rules      │
                  │  - Priorities │
                  └──────────────┘
```

### Core Components

#### 1. CompressionCoordinator

```go
package compression

import (
    "context"
    "sync"
)

// CompressionCoordinator manages conversation compression
type CompressionCoordinator struct {
    engine          *CompressionEngine
    tokenCounter    *TokenCounter
    retentionPolicy *RetentionPolicy
    config          *Config

    mu               sync.RWMutex
    currentBudget    int
    compressionCount int
}

// NewCompressionCoordinator creates a new coordinator
func NewCompressionCoordinator(opts ...Option) *CompressionCoordinator {
    cc := &CompressionCoordinator{
        engine:          NewCompressionEngine(),
        tokenCounter:    NewTokenCounter(),
        retentionPolicy: DefaultRetentionPolicy(),
        config:          DefaultConfig(),
    }

    for _, opt := range opts {
        opt(cc)
    }

    return cc
}

// Compress compresses conversation history
func (cc *CompressionCoordinator) Compress(ctx context.Context, conv *Conversation) (*CompressionResult, error)

// ShouldCompress checks if compression is needed
func (cc *CompressionCoordinator) ShouldCompress(conv *Conversation) (bool, string)

// EstimateCompression estimates compression results
func (cc *CompressionCoordinator) EstimateCompression(conv *Conversation) (*CompressionEstimate, error)

// GetStats returns compression statistics
func (cc *CompressionCoordinator) GetStats() *CompressionStats
```

#### 2. Conversation & Message Models

```go
// Conversation represents a conversation with messages
type Conversation struct {
    ID           string
    Messages     []*Message
    Metadata     map[string]interface{}
    CreatedAt    time.Time
    UpdatedAt    time.Time
    TokenCount   int
    Compressed   bool
    CompressionHistory []*CompressionRecord
}

// Message represents a single message
type Message struct {
    ID        string
    Role      MessageRole
    Content   string
    Timestamp time.Time
    TokenCount int
    Metadata  MessageMetadata
    Pinned    bool  // Prevent compression
    Important bool  // Prioritize retention
}

// MessageRole specifies message role
type MessageRole string

const (
    RoleSystem    MessageRole = "system"
    RoleUser      MessageRole = "user"
    RoleAssistant MessageRole = "assistant"
)

// MessageMetadata stores message metadata
type MessageMetadata struct {
    Type         MessageType
    Context      []string
    References   []string
    Tools        []string
    FilesPaths   []string
    CodeBlocks   int
    HasError     bool
}

// MessageType categorizes messages
type MessageType string

const (
    TypeNormal     MessageType = "normal"
    TypeCommand    MessageType = "command"
    TypeToolCall   MessageType = "tool_call"
    TypeToolResult MessageType = "tool_result"
    TypeError      MessageType = "error"
)

// CompressionRecord tracks compression history
type CompressionRecord struct {
    Timestamp      time.Time
    Strategy       CompressionStrategy
    MessagesBefore int
    MessagesAfter  int
    TokensBefore   int
    TokensAfter    int
    CompressionRatio float64
}
```

#### 3. CompressionEngine

```go
// CompressionEngine executes compression strategies
type CompressionEngine struct {
    strategies map[CompressionStrategy]Strategy
    llmClient  LLMClient
}

// Compress compresses conversation using specified strategy
func (ce *CompressionEngine) Compress(ctx context.Context, conv *Conversation, strategy CompressionStrategy) (*CompressionResult, error) {
    strat, ok := ce.strategies[strategy]
    if !ok {
        return nil, fmt.Errorf("unknown strategy: %s", strategy)
    }

    return strat.Execute(ctx, conv)
}

// CompressionResult contains compression results
type CompressionResult struct {
    Original       *Conversation
    Compressed     *Conversation
    Strategy       CompressionStrategy
    TokensSaved    int
    MessagesRemoved int
    Summary        string
    Timestamp      time.Time
}

// CompressionStrategy specifies compression approach
type CompressionStrategy int

const (
    StrategySlidingWindow CompressionStrategy = iota
    StrategySemanticSummarization
    StrategyHybrid
    StrategyCustom
)

// Strategy interface for compression strategies
type Strategy interface {
    Execute(ctx context.Context, conv *Conversation) (*CompressionResult, error)
    Estimate(conv *Conversation) (*CompressionEstimate, error)
    Name() string
}
```

#### 4. Sliding Window Strategy

```go
// SlidingWindowStrategy keeps recent messages
type SlidingWindowStrategy struct {
    windowSize int
    keepPinned bool
}

// Execute implements Strategy
func (sws *SlidingWindowStrategy) Execute(ctx context.Context, conv *Conversation) (*CompressionResult, error) {
    if len(conv.Messages) <= sws.windowSize {
        return &CompressionResult{
            Original:   conv,
            Compressed: conv,
            Strategy:   StrategySlidingWindow,
        }, nil
    }

    compressed := &Conversation{
        ID:       conv.ID,
        Metadata: conv.Metadata,
        Messages: make([]*Message, 0, sws.windowSize),
    }

    // Keep system messages
    for _, msg := range conv.Messages {
        if msg.Role == RoleSystem {
            compressed.Messages = append(compressed.Messages, msg)
        }
    }

    // Keep pinned messages
    if sws.keepPinned {
        for _, msg := range conv.Messages {
            if msg.Pinned {
                compressed.Messages = append(compressed.Messages, msg)
            }
        }
    }

    // Keep last N messages
    start := len(conv.Messages) - sws.windowSize
    if start < 0 {
        start = 0
    }
    compressed.Messages = append(compressed.Messages, conv.Messages[start:]...)

    // Calculate savings
    originalTokens := countTokens(conv.Messages)
    compressedTokens := countTokens(compressed.Messages)

    return &CompressionResult{
        Original:        conv,
        Compressed:      compressed,
        Strategy:        StrategySlidingWindow,
        TokensSaved:     originalTokens - compressedTokens,
        MessagesRemoved: len(conv.Messages) - len(compressed.Messages),
        Timestamp:       time.Now(),
    }, nil
}

// Estimate estimates compression results
func (sws *SlidingWindowStrategy) Estimate(conv *Conversation) (*CompressionEstimate, error) {
    if len(conv.Messages) <= sws.windowSize {
        return &CompressionEstimate{
            TokensSaved:     0,
            MessagesRemoved: 0,
            MessagesKept:    len(conv.Messages),
        }, nil
    }

    messagesToRemove := len(conv.Messages) - sws.windowSize
    tokensToSave := 0

    for i := 0; i < messagesToRemove; i++ {
        if !conv.Messages[i].Pinned {
            tokensToSave += conv.Messages[i].TokenCount
        }
    }

    return &CompressionEstimate{
        TokensSaved:     tokensToSave,
        MessagesRemoved: messagesToRemove,
        MessagesKept:    sws.windowSize,
    }, nil
}

// CompressionEstimate estimates compression impact
type CompressionEstimate struct {
    TokensSaved     int
    MessagesRemoved int
    MessagesKept    int
    EstimatedRatio  float64
}
```

#### 5. Semantic Summarization Strategy

```go
// SemanticSummarizationStrategy uses LLM to summarize
type SemanticSummarizationStrategy struct {
    llmClient      LLMClient
    summaryLength  int
    chunkSize      int
    preserveTypes  []MessageType
}

// Execute implements Strategy
func (sss *SemanticSummarizationStrategy) Execute(ctx context.Context, conv *Conversation) (*CompressionResult, error) {
    // Separate messages into compressible and non-compressible
    compressible, nonCompressible := sss.partitionMessages(conv.Messages)

    if len(compressible) == 0 {
        return &CompressionResult{
            Original:   conv,
            Compressed: conv,
            Strategy:   StrategySemanticSummarization,
        }, nil
    }

    // Chunk messages for summarization
    chunks := sss.chunkMessages(compressible)

    // Summarize each chunk
    summaries := make([]*Message, 0, len(chunks))
    for _, chunk := range chunks {
        summary, err := sss.summarizeChunk(ctx, chunk)
        if err != nil {
            return nil, fmt.Errorf("summarize chunk: %w", err)
        }
        summaries = append(summaries, summary)
    }

    // Build compressed conversation
    compressed := &Conversation{
        ID:       conv.ID,
        Metadata: conv.Metadata,
        Messages: make([]*Message, 0, len(nonCompressible)+len(summaries)),
    }

    // Add non-compressible messages and summaries
    compressed.Messages = append(compressed.Messages, nonCompressible...)
    compressed.Messages = append(compressed.Messages, summaries...)

    // Sort by timestamp
    sort.Slice(compressed.Messages, func(i, j int) bool {
        return compressed.Messages[i].Timestamp.Before(compressed.Messages[j].Timestamp)
    })

    // Calculate savings
    originalTokens := countTokens(conv.Messages)
    compressedTokens := countTokens(compressed.Messages)

    return &CompressionResult{
        Original:        conv,
        Compressed:      compressed,
        Strategy:        StrategySemanticSummarization,
        TokensSaved:     originalTokens - compressedTokens,
        MessagesRemoved: len(conv.Messages) - len(compressed.Messages),
        Summary:         sss.buildOverallSummary(summaries),
        Timestamp:       time.Now(),
    }, nil
}

// partitionMessages separates compressible from non-compressible
func (sss *SemanticSummarizationStrategy) partitionMessages(messages []*Message) ([]*Message, []*Message) {
    var compressible, nonCompressible []*Message

    for _, msg := range messages {
        // Don't compress system messages, pinned, or certain types
        if msg.Role == RoleSystem || msg.Pinned || sss.shouldPreserve(msg) {
            nonCompressible = append(nonCompressible, msg)
        } else {
            compressible = append(compressible, msg)
        }
    }

    return compressible, nonCompressible
}

// shouldPreserve checks if message should be preserved
func (sss *SemanticSummarizationStrategy) shouldPreserve(msg *Message) bool {
    for _, t := range sss.preserveTypes {
        if msg.Metadata.Type == t {
            return true
        }
    }
    return false
}

// chunkMessages groups messages into chunks
func (sss *SemanticSummarizationStrategy) chunkMessages(messages []*Message) [][]*Message {
    var chunks [][]*Message
    var currentChunk []*Message
    currentTokens := 0

    for _, msg := range messages {
        if currentTokens+msg.TokenCount > sss.chunkSize {
            if len(currentChunk) > 0 {
                chunks = append(chunks, currentChunk)
                currentChunk = []*Message{}
                currentTokens = 0
            }
        }

        currentChunk = append(currentChunk, msg)
        currentTokens += msg.TokenCount
    }

    if len(currentChunk) > 0 {
        chunks = append(chunks, currentChunk)
    }

    return chunks
}

// summarizeChunk creates a summary of message chunk
func (sss *SemanticSummarizationStrategy) summarizeChunk(ctx context.Context, messages []*Message) (*Message, error) {
    // Build prompt
    prompt := sss.buildSummaryPrompt(messages)

    // Call LLM
    summary, err := sss.llmClient.Summarize(ctx, prompt)
    if err != nil {
        return nil, err
    }

    // Create summary message
    return &Message{
        ID:        uuid.New().String(),
        Role:      RoleAssistant,
        Content:   fmt.Sprintf("[SUMMARY] %s", summary),
        Timestamp: messages[len(messages)-1].Timestamp,
        Metadata: MessageMetadata{
            Type: TypeNormal,
            Context: []string{"compression_summary"},
        },
    }, nil
}

// buildSummaryPrompt creates prompt for summarization
func (sss *SemanticSummarizationStrategy) buildSummaryPrompt(messages []*Message) string {
    var prompt strings.Builder

    prompt.WriteString("Summarize the following conversation messages concisely, ")
    prompt.WriteString("preserving key information, decisions, and context:\n\n")

    for i, msg := range messages {
        prompt.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, msg.Role, msg.Content))
    }

    prompt.WriteString(fmt.Sprintf("\n\nProvide a summary in approximately %d tokens:\n", sss.summaryLength))

    return prompt.String()
}
```

#### 6. Hybrid Strategy

```go
// HybridStrategy combines multiple strategies
type HybridStrategy struct {
    slidingWindow *SlidingWindowStrategy
    semantic      *SemanticSummarizationStrategy
    threshold     int // Token count threshold for semantic
}

// Execute implements Strategy
func (hs *HybridStrategy) Execute(ctx context.Context, conv *Conversation) (*CompressionResult, error) {
    // Use sliding window for recent messages
    windowResult, err := hs.slidingWindow.Execute(ctx, conv)
    if err != nil {
        return nil, err
    }

    // If we're still over threshold, use semantic compression on older messages
    if windowResult.Compressed.TokenCount > hs.threshold {
        // Get messages that were removed by sliding window
        removedMessages := hs.getRemovedMessages(conv, windowResult.Compressed)

        // Summarize removed messages
        if len(removedMessages) > 0 {
            summaryConv := &Conversation{
                ID:       conv.ID,
                Messages: removedMessages,
            }
            summaryResult, err := hs.semantic.Execute(ctx, summaryConv)
            if err != nil {
                return nil, err
            }

            // Combine summary with recent messages
            combined := &Conversation{
                ID:       conv.ID,
                Metadata: conv.Metadata,
                Messages: append(summaryResult.Compressed.Messages, windowResult.Compressed.Messages...),
            }

            return &CompressionResult{
                Original:        conv,
                Compressed:      combined,
                Strategy:        StrategyHybrid,
                TokensSaved:     windowResult.TokensSaved + summaryResult.TokensSaved,
                MessagesRemoved: windowResult.MessagesRemoved,
                Summary:         summaryResult.Summary,
                Timestamp:       time.Now(),
            }, nil
        }
    }

    return windowResult, nil
}
```

#### 7. RetentionPolicy

```go
// RetentionPolicy defines what to keep
type RetentionPolicy struct {
    rules []RetentionRule
}

// ShouldRetain checks if message should be retained
func (rp *RetentionPolicy) ShouldRetain(msg *Message, position MessagePosition) bool {
    for _, rule := range rp.rules {
        if rule.Match(msg, position) {
            return rule.Action == ActionRetain
        }
    }
    return false
}

// RetentionRule defines retention logic
type RetentionRule struct {
    Priority int
    Match    func(*Message, MessagePosition) bool
    Action   RetentionAction
    Reason   string
}

// RetentionAction specifies what to do
type RetentionAction int

const (
    ActionRetain RetentionAction = iota
    ActionCompress
    ActionRemove
)

// MessagePosition provides context about message position
type MessagePosition struct {
    Index       int
    IsFirst     bool
    IsLast      bool
    AgeDuration time.Duration
    IsRecent    bool
}

// Default retention rules
var defaultRetentionRules = []RetentionRule{
    {
        Priority: 10,
        Match: func(msg *Message, pos MessagePosition) bool {
            return msg.Role == RoleSystem
        },
        Action: ActionRetain,
        Reason: "system messages",
    },
    {
        Priority: 9,
        Match: func(msg *Message, pos MessagePosition) bool {
            return msg.Pinned
        },
        Action: ActionRetain,
        Reason: "pinned messages",
    },
    {
        Priority: 8,
        Match: func(msg *Message, pos MessagePosition) bool {
            return msg.Important
        },
        Action: ActionRetain,
        Reason: "important messages",
    },
    {
        Priority: 7,
        Match: func(msg *Message, pos MessagePosition) bool {
            return msg.Metadata.Type == TypeCommand
        },
        Action: ActionRetain,
        Reason: "command messages",
    },
    {
        Priority: 6,
        Match: func(msg *Message, pos MessagePosition) bool {
            return pos.IsRecent
        },
        Action: ActionRetain,
        Reason: "recent messages",
    },
    {
        Priority: 5,
        Match: func(msg *Message, pos MessagePosition) bool {
            return msg.Metadata.HasError
        },
        Action: ActionRetain,
        Reason: "error messages",
    },
}
```

#### 8. TokenCounter

```go
// TokenCounter counts tokens in messages
type TokenCounter struct {
    tokenizer Tokenizer
    cache     *TokenCache
}

// Count counts tokens in a message
func (tc *TokenCounter) Count(content string) int {
    // Check cache
    if count, ok := tc.cache.Get(content); ok {
        return count
    }

    // Count tokens
    count := tc.tokenizer.Count(content)

    // Cache result
    tc.cache.Set(content, count)

    return count
}

// CountConversation counts total tokens in conversation
func (tc *TokenCounter) CountConversation(conv *Conversation) int {
    total := 0
    for _, msg := range conv.Messages {
        if msg.TokenCount == 0 {
            msg.TokenCount = tc.Count(msg.Content)
        }
        total += msg.TokenCount
    }
    return total
}

// Tokenizer interface for different tokenization methods
type Tokenizer interface {
    Count(text string) int
    Encode(text string) []int
    Decode(tokens []int) string
}

// TikTokenizer uses tiktoken for counting
type TikTokenizer struct {
    encoding *tiktoken.Encoding
}

// Count implements Tokenizer
func (tt *TikTokenizer) Count(text string) int {
    tokens := tt.encoding.Encode(text, nil, nil)
    return len(tokens)
}

// TokenCache caches token counts
type TokenCache struct {
    mu    sync.RWMutex
    cache map[string]int
    maxSize int
}

// Get retrieves cached count
func (tc *TokenCache) Get(content string) (int, bool) {
    tc.mu.RLock()
    defer tc.mu.RUnlock()

    hash := hashString(content)
    count, ok := tc.cache[hash]
    return count, ok
}

// Set stores count in cache
func (tc *TokenCache) Set(content string, count int) {
    tc.mu.Lock()
    defer tc.mu.Unlock()

    if len(tc.cache) >= tc.maxSize {
        // Evict random entry
        for k := range tc.cache {
            delete(tc.cache, k)
            break
        }
    }

    hash := hashString(content)
    tc.cache[hash] = count
}
```

### State Machine

```
┌─────────┐
│  Active │  (Normal conversation)
└────┬────┘
     │ Token count > threshold
     ▼
┌──────────────┐
│  Analyzing   │
│ - Count tok  │
│ - Check pol  │
└──────┬───────┘
       │
       ├── Below threshold ──┐
       │                     ▼
       │               ┌─────────┐
       │               │  Active │
       │               └─────────┘
       │
       └── Above threshold
           ▼
    ┌──────────────┐
    │ Compressing  │
    │ - Select str │
    │ - Execute    │
    └──────┬───────┘
           │
           ├──Success──┐
           │           ▼
           │    ┌─────────────┐
           │    │ Compressed  │
           │    │ - Update    │
           │    │ - Record    │
           │    └──────┬──────┘
           │           │
           │           ▼
           │    ┌─────────┐
           │    │  Active │
           │    └─────────┘
           │
           └──Error────┐
                       ▼
                 ┌───────────┐
                 │  Failed   │
                 │ - Log err │
                 └───────────┘
```

## Commands Integration

### /compress Command

```go
// CompressCommand implements the /compress command
type CompressCommand struct {
    coordinator *CompressionCoordinator
}

// Execute runs the compress command
func (cc *CompressCommand) Execute(ctx context.Context, args []string) (*CommandResult, error) {
    // Parse arguments
    opts := parseCompressOptions(args)

    // Get current conversation
    conv := getCurrentConversation(ctx)

    // Compress
    result, err := cc.coordinator.Compress(ctx, conv)
    if err != nil {
        return nil, fmt.Errorf("compress: %w", err)
    }

    // Format output
    output := formatCompressionResult(result)

    return &CommandResult{
        Success: true,
        Message: output,
        Data:    result,
    }, nil
}

// CompressOptions configures compress command
type CompressOptions struct {
    Strategy    CompressionStrategy
    DryRun      bool
    ShowDiff    bool
    KeepRecent  int
}

// parseCompressOptions parses command arguments
func parseCompressOptions(args []string) CompressOptions {
    opts := CompressOptions{
        Strategy:   StrategyHybrid,
        KeepRecent: 10,
    }

    for i := 0; i < len(args); i++ {
        switch args[i] {
        case "--strategy":
            if i+1 < len(args) {
                opts.Strategy = parseStrategy(args[i+1])
                i++
            }
        case "--dry-run":
            opts.DryRun = true
        case "--show-diff":
            opts.ShowDiff = true
        case "--keep":
            if i+1 < len(args) {
                opts.KeepRecent, _ = strconv.Atoi(args[i+1])
                i++
            }
        }
    }

    return opts
}
```

### Auto-Compression

```go
// AutoCompressor monitors and auto-compresses
type AutoCompressor struct {
    coordinator *CompressionCoordinator
    enabled     bool
    threshold   int
    interval    time.Duration
}

// Start begins auto-compression monitoring
func (ac *AutoCompressor) Start(ctx context.Context) {
    if !ac.enabled {
        return
    }

    ticker := time.NewTicker(ac.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            ac.checkAndCompress(ctx)
        }
    }
}

// checkAndCompress checks if compression is needed
func (ac *AutoCompressor) checkAndCompress(ctx context.Context) {
    conv := getCurrentConversation(ctx)

    should, reason := ac.coordinator.ShouldCompress(conv)
    if !should {
        return
    }

    log.Printf("Auto-compressing: %s", reason)

    result, err := ac.coordinator.Compress(ctx, conv)
    if err != nil {
        log.Printf("Auto-compression failed: %v", err)
        return
    }

    log.Printf("Auto-compressed: saved %d tokens, removed %d messages",
        result.TokensSaved, result.MessagesRemoved)
}
```

## Configuration Schema

```yaml
# context_compression.yaml

compression:
  # Enable compression
  enabled: true

  # Auto-compression
  auto:
    enabled: true
    threshold: 100000  # tokens
    interval: 5m
    strategy: hybrid

  # Strategies
  strategies:
    # Sliding window
    sliding_window:
      size: 20
      keep_pinned: true

    # Semantic summarization
    semantic:
      # LLM settings
      provider: claude
      model: claude-3-5-sonnet-20241022

      # Summarization
      summary_length: 200  # tokens per chunk
      chunk_size: 5000     # tokens per chunk
      preserve_types:
        - command
        - error
        - tool_call

    # Hybrid strategy
    hybrid:
      window_size: 15
      summary_threshold: 80000
      summary_length: 300

  # Retention policy
  retention:
    # Always retain
    always_retain:
      - system_messages: true
      - pinned_messages: true
      - important_messages: true
      - recent_messages: 10

    # Message types to preserve
    preserve_types:
      - command
      - error
      - tool_call

    # Time-based retention
    min_age_to_compress: 1h
    keep_recent_duration: 30m

  # Token budgets
  budgets:
    total: 200000
    warning_threshold: 150000
    compression_threshold: 180000

  # Performance
  performance:
    cache_size: 1000
    parallel_chunks: 4

  # Output
  output:
    show_summary: true
    show_stats: true
    log_compressions: true
```

```go
// Config represents compression configuration
type Config struct {
    Enabled   bool              `yaml:"enabled"`
    Auto      AutoConfig        `yaml:"auto"`
    Strategies StrategiesConfig `yaml:"strategies"`
    Retention RetentionConfig   `yaml:"retention"`
    Budgets   BudgetsConfig     `yaml:"budgets"`
    Performance PerformanceConfig `yaml:"performance"`
    Output    OutputConfig      `yaml:"output"`
}

// AutoConfig configures auto-compression
type AutoConfig struct {
    Enabled   bool              `yaml:"enabled"`
    Threshold int               `yaml:"threshold"`
    Interval  time.Duration     `yaml:"interval"`
    Strategy  CompressionStrategy `yaml:"strategy"`
}

// StrategiesConfig configures strategies
type StrategiesConfig struct {
    SlidingWindow SlidingWindowConfig `yaml:"sliding_window"`
    Semantic      SemanticConfig      `yaml:"semantic"`
    Hybrid        HybridConfig        `yaml:"hybrid"`
}

// BudgetsConfig configures token budgets
type BudgetsConfig struct {
    Total                  int `yaml:"total"`
    WarningThreshold       int `yaml:"warning_threshold"`
    CompressionThreshold   int `yaml:"compression_threshold"`
}
```

## Testing Strategy

### Unit Tests

```go
func TestSlidingWindowStrategy_Execute(t *testing.T) {
    strategy := &SlidingWindowStrategy{
        windowSize: 5,
        keepPinned: true,
    }

    conv := &Conversation{
        Messages: make([]*Message, 10),
    }
    for i := range conv.Messages {
        conv.Messages[i] = &Message{
            ID:      fmt.Sprintf("msg-%d", i),
            Content: fmt.Sprintf("Message %d", i),
            Role:    RoleUser,
        }
    }

    result, err := strategy.Execute(context.Background(), conv)
    require.NoError(t, err)
    assert.Len(t, result.Compressed.Messages, 5)
    assert.Equal(t, 5, result.MessagesRemoved)
}

func TestSemanticSummarizationStrategy_Execute(t *testing.T) {
    mockLLM := &MockLLMClient{
        summarizeFunc: func(ctx context.Context, prompt string) (string, error) {
            return "Summary of messages", nil
        },
    }

    strategy := &SemanticSummarizationStrategy{
        llmClient:     mockLLM,
        summaryLength: 100,
        chunkSize:     1000,
    }

    conv := createTestConversation(20)

    result, err := strategy.Execute(context.Background(), conv)
    require.NoError(t, err)
    assert.Less(t, len(result.Compressed.Messages), len(conv.Messages))
    assert.Greater(t, result.TokensSaved, 0)
}

func TestRetentionPolicy_ShouldRetain(t *testing.T) {
    policy := DefaultRetentionPolicy()

    tests := []struct {
        name     string
        message  *Message
        position MessagePosition
        want     bool
    }{
        {
            name: "system message",
            message: &Message{
                Role: RoleSystem,
            },
            want: true,
        },
        {
            name: "pinned message",
            message: &Message{
                Role:   RoleUser,
                Pinned: true,
            },
            want: true,
        },
        {
            name: "recent message",
            message: &Message{
                Role: RoleUser,
            },
            position: MessagePosition{
                IsRecent: true,
            },
            want: true,
        },
        {
            name: "old message",
            message: &Message{
                Role: RoleUser,
            },
            position: MessagePosition{
                IsRecent: false,
            },
            want: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := policy.ShouldRetain(tt.message, tt.position)
            assert.Equal(t, tt.want, got)
        })
    }
}

func TestTokenCounter_Count(t *testing.T) {
    tc := NewTokenCounter()

    tests := []struct {
        name    string
        content string
        want    int
    }{
        {
            name:    "empty",
            content: "",
            want:    0,
        },
        {
            name:    "simple text",
            content: "Hello world",
            want:    2, // Approximate
        },
        {
            name:    "code",
            content: "func main() { println(\"Hello\") }",
            want:    10, // Approximate
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := tc.Count(tt.content)
            // Allow some variance due to tokenization differences
            assert.InDelta(t, tt.want, got, float64(tt.want)*0.2)
        })
    }
}
```

### Integration Tests

```go
func TestCompressionCoordinator_EndToEnd(t *testing.T) {
    coordinator := NewCompressionCoordinator(
        WithStrategy(StrategyHybrid),
        WithThreshold(1000),
    )

    // Create conversation with many messages
    conv := createLargeConversation(100, 50) // 100 messages, ~50 tokens each

    // Should need compression
    should, reason := coordinator.ShouldCompress(conv)
    assert.True(t, should)
    assert.NotEmpty(t, reason)

    // Compress
    result, err := coordinator.Compress(context.Background(), conv)
    require.NoError(t, err)
    assert.Less(t, len(result.Compressed.Messages), len(conv.Messages))
    assert.Greater(t, result.TokensSaved, 0)

    // Verify important messages retained
    hasSystemMsg := false
    for _, msg := range result.Compressed.Messages {
        if msg.Role == RoleSystem {
            hasSystemMsg = true
            break
        }
    }
    assert.True(t, hasSystemMsg)
}

func TestAutoCompressor_AutoCompress(t *testing.T) {
    coordinator := NewCompressionCoordinator(
        WithThreshold(1000),
    )

    autoCompressor := &AutoCompressor{
        coordinator: coordinator,
        enabled:     true,
        threshold:   1000,
        interval:    100 * time.Millisecond,
    }

    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()

    // Start auto-compressor
    go autoCompressor.Start(ctx)

    // Create conversation that needs compression
    conv := createLargeConversation(50, 100)
    setCurrentConversation(ctx, conv)

    // Wait for auto-compression
    time.Sleep(200 * time.Millisecond)

    // Verify compression occurred
    stats := coordinator.GetStats()
    assert.Greater(t, stats.CompressionCount, 0)
}
```

### Compression Quality Tests

```go
func TestCompressionQuality(t *testing.T) {
    tests := []struct {
        name           string
        conversation   *Conversation
        strategy       CompressionStrategy
        minRatio       float64
        mustPreserve   []string
    }{
        {
            name: "preserve system messages",
            conversation: &Conversation{
                Messages: []*Message{
                    {Role: RoleSystem, Content: "System prompt"},
                    {Role: RoleUser, Content: "User message 1"},
                    {Role: RoleAssistant, Content: "Response 1"},
                    // ... more messages
                },
            },
            strategy:     StrategySlidingWindow,
            minRatio:     0.3,
            mustPreserve: []string{"System prompt"},
        },
        {
            name: "semantic preservation",
            conversation: createTestConversation(50),
            strategy:     StrategySemanticSummarization,
            minRatio:     0.5,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            engine := NewCompressionEngine()
            result, err := engine.Compress(context.Background(), tt.conversation, tt.strategy)
            require.NoError(t, err)

            // Check compression ratio
            ratio := float64(result.TokensSaved) / float64(countTokens(tt.conversation.Messages))
            assert.GreaterOrEqual(t, ratio, tt.minRatio)

            // Check preservation
            for _, preserve := range tt.mustPreserve {
                found := false
                for _, msg := range result.Compressed.Messages {
                    if strings.Contains(msg.Content, preserve) {
                        found = true
                        break
                    }
                }
                assert.True(t, found, "must preserve: %s", preserve)
            }
        })
    }
}
```

## Performance Considerations

### Parallel Processing

```go
// CompressParallel compresses multiple chunks in parallel
func (sss *SemanticSummarizationStrategy) CompressParallel(ctx context.Context, conv *Conversation) (*CompressionResult, error) {
    chunks := sss.chunkMessages(conv.Messages)

    var wg sync.WaitGroup
    results := make([]*Message, len(chunks))
    errors := make([]error, len(chunks))

    // Limit parallelism
    semaphore := make(chan struct{}, 4)

    for i, chunk := range chunks {
        wg.Add(1)
        go func(idx int, msgs []*Message) {
            defer wg.Done()
            semaphore <- struct{}{}
            defer func() { <-semaphore }()

            summary, err := sss.summarizeChunk(ctx, msgs)
            if err != nil {
                errors[idx] = err
                return
            }
            results[idx] = summary
        }(i, chunk)
    }

    wg.Wait()

    // Check for errors
    for _, err := range errors {
        if err != nil {
            return nil, err
        }
    }

    // Build result
    compressed := &Conversation{
        ID:       conv.ID,
        Messages: results,
    }

    return &CompressionResult{
        Original:   conv,
        Compressed: compressed,
        Strategy:   StrategySemanticSummarization,
    }, nil
}
```

### Caching

```go
// SummaryCache caches summaries
type SummaryCache struct {
    mu    sync.RWMutex
    cache map[string]*CachedSummary
    ttl   time.Duration
}

type CachedSummary struct {
    Summary   string
    Timestamp time.Time
}

// Get retrieves cached summary
func (sc *SummaryCache) Get(messagesHash string) (string, bool) {
    sc.mu.RLock()
    defer sc.mu.RUnlock()

    cached, ok := sc.cache[messagesHash]
    if !ok {
        return "", false
    }

    if time.Since(cached.Timestamp) > sc.ttl {
        return "", false
    }

    return cached.Summary, true
}
```

## Security Considerations

### Sensitive Data Handling

```go
// RedactionFilter removes sensitive data before compression
type RedactionFilter struct {
    patterns []*regexp.Regexp
}

// Filter redacts sensitive information
func (rf *RedactionFilter) Filter(message *Message) *Message {
    filtered := *message

    for _, pattern := range rf.patterns {
        filtered.Content = pattern.ReplaceAllString(filtered.Content, "[REDACTED]")
    }

    return &filtered
}

// Default redaction patterns
var defaultRedactionPatterns = []string{
    `(?i)password\s*[:=]\s*\S+`,
    `(?i)api[_-]?key\s*[:=]\s*\S+`,
    `(?i)token\s*[:=]\s*\S+`,
    `\b\d{3}-\d{2}-\d{4}\b`, // SSN
    `\b\d{16}\b`,            // Credit card
}
```

## References

### Qwen Code Compression

- **Feature**: Context compression for long conversations
- **Implementation**: Sliding window + summarization
- **Strategy**: Keep recent messages, summarize old ones

### Cline Chat Compression

- **Repository**: `src/core/chat-manager.ts`
- **Features**:
  - Automatic compression on threshold
  - Preserve system messages
  - Token counting

### Plandex Summarization

- **Feature**: Conversation summarization
- **Implementation**: LLM-powered semantic summarization
- **Strategy**: Chunk and summarize historical context

### Key Insights

1. **Hybrid Approach**: Combine sliding window with semantic summarization
2. **Preserve Important**: Always keep system, pinned, and recent messages
3. **Token Awareness**: Track tokens accurately for budget management
4. **Semantic Preservation**: Use LLM to maintain semantic meaning
5. **Configurable**: Allow users to configure compression behavior

## Usage Examples

```go
// Example 1: Manual compression
func ExampleManualCompress() {
    coordinator := NewCompressionCoordinator()
    conv := getCurrentConversation()

    result, _ := coordinator.Compress(context.Background(), conv)
    fmt.Printf("Compressed: %d -> %d messages\n",
        len(result.Original.Messages),
        len(result.Compressed.Messages))
    fmt.Printf("Saved %d tokens\n", result.TokensSaved)
}

// Example 2: Auto-compression
func ExampleAutoCompress() {
    coordinator := NewCompressionCoordinator(
        WithAutoCompress(true),
        WithThreshold(100000),
    )

    autoCompressor := &AutoCompressor{
        coordinator: coordinator,
        enabled:     true,
        threshold:   100000,
        interval:    5 * time.Minute,
    }

    ctx := context.Background()
    go autoCompressor.Start(ctx)
}

// Example 3: Custom retention policy
func ExampleCustomPolicy() {
    policy := &RetentionPolicy{
        rules: []RetentionRule{
            {
                Priority: 10,
                Match: func(msg *Message, pos MessagePosition) bool {
                    return msg.Metadata.Type == TypeCommand
                },
                Action: ActionRetain,
                Reason: "preserve commands",
            },
        },
    }

    coordinator := NewCompressionCoordinator(
        WithRetentionPolicy(policy),
    )
}
```

## Future Enhancements

1. **Adaptive Compression**: Learn optimal compression points
2. **Multi-Level Summaries**: Hierarchical summarization
3. **Lossless Compression**: Store full history externally
4. **Query Support**: Search compressed conversations
5. **Visualization**: Show compression history and impact
