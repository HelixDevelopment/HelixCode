package compression

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"dev.helix.code/internal/llm"
	"github.com/google/uuid"
)

// CompressionStrategy specifies the compression approach
type CompressionStrategy int

const (
	// StrategySlidingWindow keeps the last N messages
	StrategySlidingWindow CompressionStrategy = iota
	// StrategySemanticSummarization uses LLM to summarize old messages
	StrategySemanticSummarization
	// StrategyHybrid combines sliding window and semantic summarization
	StrategyHybrid
	// StrategyCustom allows for custom compression logic
	StrategyCustom
)

// String returns the string representation of a compression strategy
func (cs CompressionStrategy) String() string {
	switch cs {
	case StrategySlidingWindow:
		return "sliding_window"
	case StrategySemanticSummarization:
		return "semantic_summarization"
	case StrategyHybrid:
		return "hybrid"
	case StrategyCustom:
		return "custom"
	default:
		return "unknown"
	}
}

// Strategy interface for compression strategies
type Strategy interface {
	Execute(ctx context.Context, conv *Conversation, policy *RetentionPolicy) (*CompressionResult, error)
	Estimate(conv *Conversation, policy *RetentionPolicy) (*CompressionEstimate, error)
	Name() string
}

// CompressionEngine executes compression strategies
type CompressionEngine struct {
	strategies map[CompressionStrategy]Strategy
	llmClient  LLMClient
}

// LLMClient interface for LLM operations
type LLMClient interface {
	Summarize(ctx context.Context, prompt string) (string, error)
}

// llmProviderAdapter adapts llm.Provider to LLMClient
type llmProviderAdapter struct {
	provider llm.Provider
}

// Summarize implements LLMClient
func (a *llmProviderAdapter) Summarize(ctx context.Context, prompt string) (string, error) {
	request := &llm.LLMRequest{
		ID:          uuid.New(),
		Model:       "",
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   500,
		Temperature: 0.7,
		Stream:      false,
	}

	response, err := a.provider.Generate(ctx, request)
	if err != nil {
		return "", fmt.Errorf("LLM generation failed: %w", err)
	}

	return response.Content, nil
}

// NewCompressionEngine creates a new compression engine
func NewCompressionEngine(provider llm.Provider) *CompressionEngine {
	llmClient := &llmProviderAdapter{provider: provider}

	ce := &CompressionEngine{
		strategies: make(map[CompressionStrategy]Strategy),
		llmClient:  llmClient,
	}

	// Register default strategies
	ce.strategies[StrategySlidingWindow] = &SlidingWindowStrategy{
		windowSize: 20,
		keepPinned: true,
	}
	ce.strategies[StrategySemanticSummarization] = &SemanticSummarizationStrategy{
		llmClient:     llmClient,
		summaryLength: 200,
		chunkSize:     5000,
		preserveTypes: []MessageType{TypeCommand, TypeError, TypeToolCall},
	}
	ce.strategies[StrategyHybrid] = &HybridStrategy{
		slidingWindow: ce.strategies[StrategySlidingWindow].(*SlidingWindowStrategy),
		semantic:      ce.strategies[StrategySemanticSummarization].(*SemanticSummarizationStrategy),
		threshold:     80000,
	}

	return ce
}

// Compress compresses a conversation using the specified strategy
func (ce *CompressionEngine) Compress(ctx context.Context, conv *Conversation, strategy CompressionStrategy, policy *RetentionPolicy) (*CompressionResult, error) {
	strat, ok := ce.strategies[strategy]
	if !ok {
		return nil, fmt.Errorf("unknown strategy: %s", strategy)
	}

	return strat.Execute(ctx, conv, policy)
}

// GetStrategy returns a strategy by type
func (ce *CompressionEngine) GetStrategy(strategy CompressionStrategy) (Strategy, error) {
	strat, ok := ce.strategies[strategy]
	if !ok {
		return nil, fmt.Errorf("unknown strategy: %s", strategy)
	}
	return strat, nil
}

// RegisterStrategy registers a custom strategy
func (ce *CompressionEngine) RegisterStrategy(strategyType CompressionStrategy, strategy Strategy) {
	ce.strategies[strategyType] = strategy
}

// SlidingWindowStrategy keeps recent messages and drops older ones
type SlidingWindowStrategy struct {
	windowSize int
	keepPinned bool
}

// Name implements Strategy
func (sws *SlidingWindowStrategy) Name() string {
	return "sliding_window"
}

// Execute implements Strategy
func (sws *SlidingWindowStrategy) Execute(ctx context.Context, conv *Conversation, policy *RetentionPolicy) (*CompressionResult, error) {
	if len(conv.Messages) <= sws.windowSize {
		return &CompressionResult{
			Original:   conv,
			Compressed: conv,
			Strategy:   StrategySlidingWindow,
			Timestamp:  time.Now(),
		}, nil
	}

	compressed := &Conversation{
		ID:       conv.ID,
		Metadata: conv.Metadata,
		Messages: make([]*Message, 0, sws.windowSize),
	}

	// Always keep system messages
	for _, msg := range conv.Messages {
		if msg.Role == RoleSystem {
			compressed.Messages = append(compressed.Messages, msg)
		}
	}

	// Keep pinned messages if configured
	if sws.keepPinned {
		for _, msg := range conv.Messages {
			if msg.Pinned && msg.Role != RoleSystem {
				compressed.Messages = append(compressed.Messages, msg)
			}
		}
	}

	// Apply retention policy
	retained := make([]*Message, 0)
	for i, msg := range conv.Messages {
		if msg.Role == RoleSystem || (sws.keepPinned && msg.Pinned) {
			continue // Already added
		}

		position := MessagePosition{
			Index:    i,
			IsFirst:  i == 0,
			IsLast:   i == len(conv.Messages)-1,
			IsRecent: i >= len(conv.Messages)-sws.windowSize,
		}

		if policy.ShouldRetain(msg, position) {
			retained = append(retained, msg)
		}
	}

	// Keep last N messages from retained
	start := len(retained) - sws.windowSize
	if start < 0 {
		start = 0
	}
	compressed.Messages = append(compressed.Messages, retained[start:]...)

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
		Strategy:        StrategySlidingWindow,
		TokensSaved:     originalTokens - compressedTokens,
		MessagesRemoved: len(conv.Messages) - len(compressed.Messages),
		Timestamp:       time.Now(),
	}, nil
}

// Estimate implements Strategy
func (sws *SlidingWindowStrategy) Estimate(conv *Conversation, policy *RetentionPolicy) (*CompressionEstimate, error) {
	if len(conv.Messages) <= sws.windowSize {
		return &CompressionEstimate{
			TokensSaved:     0,
			MessagesRemoved: 0,
			MessagesKept:    len(conv.Messages),
			EstimatedRatio:  0,
		}, nil
	}

	messagesToRemove := len(conv.Messages) - sws.windowSize
	tokensToSave := 0

	for i := 0; i < messagesToRemove; i++ {
		msg := conv.Messages[i]
		if !msg.Pinned && msg.Role != RoleSystem {
			tokensToSave += msg.TokenCount
		}
	}

	return &CompressionEstimate{
		TokensSaved:     tokensToSave,
		MessagesRemoved: messagesToRemove,
		MessagesKept:    sws.windowSize,
		EstimatedRatio:  float64(tokensToSave) / float64(countTokens(conv.Messages)),
	}, nil
}

// SemanticSummarizationStrategy uses LLM to summarize old messages
type SemanticSummarizationStrategy struct {
	llmClient     LLMClient
	summaryLength int
	chunkSize     int
	preserveTypes []MessageType
}

// Name implements Strategy
func (sss *SemanticSummarizationStrategy) Name() string {
	return "semantic_summarization"
}

// Execute implements Strategy
func (sss *SemanticSummarizationStrategy) Execute(ctx context.Context, conv *Conversation, policy *RetentionPolicy) (*CompressionResult, error) {
	// Partition messages into compressible and non-compressible
	compressible, nonCompressible := sss.partitionMessages(conv.Messages, policy)

	if len(compressible) == 0 {
		return &CompressionResult{
			Original:   conv,
			Compressed: conv,
			Strategy:   StrategySemanticSummarization,
			Timestamp:  time.Now(),
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

// Estimate implements Strategy
func (sss *SemanticSummarizationStrategy) Estimate(conv *Conversation, policy *RetentionPolicy) (*CompressionEstimate, error) {
	compressible, _ := sss.partitionMessages(conv.Messages, policy)

	if len(compressible) == 0 {
		return &CompressionEstimate{
			TokensSaved:     0,
			MessagesRemoved: 0,
			MessagesKept:    len(conv.Messages),
			EstimatedRatio:  0,
		}, nil
	}

	chunks := sss.chunkMessages(compressible)
	tokensToSave := countTokens(compressible)
	estimatedSummaryTokens := len(chunks) * sss.summaryLength

	return &CompressionEstimate{
		TokensSaved:     tokensToSave - estimatedSummaryTokens,
		MessagesRemoved: len(compressible),
		MessagesKept:    len(conv.Messages) - len(compressible) + len(chunks),
		EstimatedRatio:  float64(tokensToSave-estimatedSummaryTokens) / float64(countTokens(conv.Messages)),
	}, nil
}

// partitionMessages separates compressible from non-compressible messages
func (sss *SemanticSummarizationStrategy) partitionMessages(messages []*Message, policy *RetentionPolicy) ([]*Message, []*Message) {
	var compressible, nonCompressible []*Message

	for i, msg := range messages {
		position := MessagePosition{
			Index:   i,
			IsFirst: i == 0,
			IsLast:  i == len(messages)-1,
		}

		// Don't compress system messages, pinned, or messages that should be retained
		if msg.Role == RoleSystem || msg.Pinned || policy.ShouldRetain(msg, position) || sss.shouldPreserve(msg) {
			nonCompressible = append(nonCompressible, msg)
		} else {
			compressible = append(compressible, msg)
		}
	}

	return compressible, nonCompressible
}

// shouldPreserve checks if a message type should be preserved
func (sss *SemanticSummarizationStrategy) shouldPreserve(msg *Message) bool {
	for _, t := range sss.preserveTypes {
		if msg.Metadata.Type == t {
			return true
		}
	}
	return false
}

// chunkMessages groups messages into chunks for summarization
func (sss *SemanticSummarizationStrategy) chunkMessages(messages []*Message) [][]*Message {
	var chunks [][]*Message
	var currentChunk []*Message
	currentTokens := 0

	for _, msg := range messages {
		if currentTokens+msg.TokenCount > sss.chunkSize && len(currentChunk) > 0 {
			chunks = append(chunks, currentChunk)
			currentChunk = []*Message{}
			currentTokens = 0
		}

		currentChunk = append(currentChunk, msg)
		currentTokens += msg.TokenCount
	}

	if len(currentChunk) > 0 {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}

// summarizeChunk creates a summary of a message chunk using LLM
func (sss *SemanticSummarizationStrategy) summarizeChunk(ctx context.Context, messages []*Message) (*Message, error) {
	// Build prompt
	prompt := sss.buildSummaryPrompt(messages)

	// Call LLM
	summary, err := sss.llmClient.Summarize(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM summarization failed: %w", err)
	}

	// Create summary message
	return &Message{
		ID:         uuid.New().String(),
		Role:       RoleAssistant,
		Content:    fmt.Sprintf("[SUMMARY] %s", summary),
		Timestamp:  messages[len(messages)-1].Timestamp,
		TokenCount: (len(summary) + 3) / 4, // Approximate token count
		Metadata: MessageMetadata{
			Type:    TypeNormal,
			Context: []string{"compression_summary"},
		},
	}, nil
}

// buildSummaryPrompt creates a prompt for LLM summarization
func (sss *SemanticSummarizationStrategy) buildSummaryPrompt(messages []*Message) string {
	var prompt strings.Builder

	prompt.WriteString("Summarize the following conversation messages concisely, ")
	prompt.WriteString("preserving key information, decisions, context, and any important details:\n\n")

	for i, msg := range messages {
		prompt.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, msg.Role, msg.Content))
	}

	prompt.WriteString(fmt.Sprintf("\n\nProvide a concise summary in approximately %d tokens that captures the essential information:\n", sss.summaryLength))

	return prompt.String()
}

// buildOverallSummary combines multiple summary messages
func (sss *SemanticSummarizationStrategy) buildOverallSummary(summaries []*Message) string {
	if len(summaries) == 0 {
		return ""
	}

	if len(summaries) == 1 {
		return summaries[0].Content
	}

	var overall strings.Builder
	overall.WriteString("Compressed conversation summary:\n")
	for i, summary := range summaries {
		overall.WriteString(fmt.Sprintf("Part %d: %s\n", i+1, summary.Content))
	}

	return overall.String()
}

// HybridStrategy combines sliding window and semantic summarization
type HybridStrategy struct {
	slidingWindow *SlidingWindowStrategy
	semantic      *SemanticSummarizationStrategy
	threshold     int
}

// Name implements Strategy
func (hs *HybridStrategy) Name() string {
	return "hybrid"
}

// Execute implements Strategy
func (hs *HybridStrategy) Execute(ctx context.Context, conv *Conversation, policy *RetentionPolicy) (*CompressionResult, error) {
	// First, apply sliding window to keep recent messages
	windowResult, err := hs.slidingWindow.Execute(ctx, conv, policy)
	if err != nil {
		return nil, fmt.Errorf("sliding window failed: %w", err)
	}

	// Check if we need semantic compression on older messages
	tokenCount := countTokens(windowResult.Compressed.Messages)

	if tokenCount > hs.threshold && len(conv.Messages) > len(windowResult.Compressed.Messages) {
		// Get messages that were removed by sliding window
		removedMessages := hs.getRemovedMessages(conv, windowResult.Compressed)

		if len(removedMessages) > 0 {
			// Create conversation with removed messages
			summaryConv := &Conversation{
				ID:       conv.ID,
				Messages: removedMessages,
			}

			// Summarize removed messages
			summaryResult, err := hs.semantic.Execute(ctx, summaryConv, policy)
			if err != nil {
				// If summarization fails, fall back to sliding window only
				return windowResult, nil
			}

			// Combine summary with recent messages
			combined := &Conversation{
				ID:       conv.ID,
				Metadata: conv.Metadata,
				Messages: append(summaryResult.Compressed.Messages, windowResult.Compressed.Messages...),
			}

			// Sort by timestamp
			sort.Slice(combined.Messages, func(i, j int) bool {
				return combined.Messages[i].Timestamp.Before(combined.Messages[j].Timestamp)
			})

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

	// If no semantic compression needed, return sliding window result
	return windowResult, nil
}

// Estimate implements Strategy
func (hs *HybridStrategy) Estimate(conv *Conversation, policy *RetentionPolicy) (*CompressionEstimate, error) {
	windowEstimate, err := hs.slidingWindow.Estimate(conv, policy)
	if err != nil {
		return nil, err
	}

	// Estimate semantic compression on older messages
	semanticEstimate, err := hs.semantic.Estimate(conv, policy)
	if err != nil {
		return windowEstimate, nil
	}

	// Combine estimates
	return &CompressionEstimate{
		TokensSaved:     windowEstimate.TokensSaved + semanticEstimate.TokensSaved/2,
		MessagesRemoved: windowEstimate.MessagesRemoved,
		MessagesKept:    windowEstimate.MessagesKept + 2, // Account for summary messages
		EstimatedRatio:  (windowEstimate.EstimatedRatio + semanticEstimate.EstimatedRatio) / 2,
	}, nil
}

// getRemovedMessages returns messages that were removed by compression
func (hs *HybridStrategy) getRemovedMessages(original, compressed *Conversation) []*Message {
	// Create a map of compressed message IDs
	compressedIDs := make(map[string]bool)
	for _, msg := range compressed.Messages {
		compressedIDs[msg.ID] = true
	}

	// Find removed messages
	var removed []*Message
	for _, msg := range original.Messages {
		if !compressedIDs[msg.ID] {
			removed = append(removed, msg)
		}
	}

	return removed
}

// countTokens counts tokens in a slice of messages
func countTokens(messages []*Message) int {
	total := 0
	for _, msg := range messages {
		if msg.TokenCount == 0 {
			// Approximate token count if not already set
			msg.TokenCount = (len(msg.Content) + 3) / 4
		}
		total += msg.TokenCount
	}
	return total
}
