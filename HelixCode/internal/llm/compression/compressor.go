package compression

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"dev.helix.code/internal/llm"
	"dev.helix.code/internal/llm/compressioniface"
	"github.com/google/uuid"
)

// CompressionCoordinator manages conversation compression lifecycle
type CompressionCoordinator struct {
	engine          *CompressionEngine
	tokenCounter    *TokenCounter
	retentionPolicy *RetentionPolicy
	config          *Config

	mu               sync.RWMutex
	currentBudget    int
	compressionCount int
	stats            CompressionStats
}

// Ensure CompressionCoordinator implements the interface
var _ compressioniface.CompressionCoordinator = (*CompressionCoordinator)(nil)

// Compress implements compressioniface.CompressionCoordinator
func (cc *CompressionCoordinator) Compress(ctx context.Context, conv *compressioniface.Conversation) (*compressioniface.CompressionResult, error) {
	// Convert interface types to internal types
	internalConv := ConvertToInternalConversation(conv)

	// Execute compression
	result, err := cc.compressInternal(ctx, internalConv)
	if err != nil {
		return nil, err
	}

	// Convert back to interface types
	return ConvertToInterfaceResult(result), nil
}

// ShouldCompress implements compressioniface.CompressionCoordinator
func (cc *CompressionCoordinator) ShouldCompress(conv *compressioniface.Conversation) (bool, string) {
	internalConv := ConvertToInternalConversation(conv)
	return cc.shouldCompressInternal(internalConv)
}

// EstimateCompression implements compressioniface.CompressionCoordinator
func (cc *CompressionCoordinator) EstimateCompression(conv *compressioniface.Conversation) (*compressioniface.CompressionEstimate, error) {
	internalConv := ConvertToInternalConversation(conv)
	estimate, err := cc.estimateCompressionInternal(internalConv)
	if err != nil {
		return nil, err
	}
	return ConvertToInterfaceEstimate(estimate), nil
}

// GetStats implements compressioniface.CompressionCoordinator
func (cc *CompressionCoordinator) GetStats() *compressioniface.CompressionStats {
	stats := cc.getStatsInternal()
	return &compressioniface.CompressionStats{
		TotalCompressions:    stats.TotalCompressions,
		TotalTokensSaved:     stats.TotalTokensSaved,
		TotalMessagesRemoved: stats.TotalMessagesRemoved,
		LastCompression:      stats.LastCompression,
		AverageRatio:         stats.AverageRatio,
	}
}

// GetConfig implements compressioniface.CompressionCoordinator
func (cc *CompressionCoordinator) GetConfig() *compressioniface.Config {
	config := cc.getConfigInternal()
	return &compressioniface.Config{
		Enabled:              config.Enabled,
		DefaultStrategy:      compressioniface.CompressionStrategy(config.DefaultStrategy),
		TokenBudget:          config.TokenBudget,
		WarningThreshold:     config.WarningThreshold,
		CompressionThreshold: config.CompressionThreshold,
		AutoCompressEnabled:  config.AutoCompressEnabled,
		AutoCompressInterval: config.AutoCompressInterval,
	}
}

// UpdateConfig implements compressioniface.CompressionCoordinator
func (cc *CompressionCoordinator) UpdateConfig(config *compressioniface.Config) {
	internalConfig := &Config{
		Enabled:              config.Enabled,
		DefaultStrategy:      CompressionStrategy(config.DefaultStrategy),
		TokenBudget:          config.TokenBudget,
		WarningThreshold:     config.WarningThreshold,
		CompressionThreshold: config.CompressionThreshold,
		AutoCompressEnabled:  config.AutoCompressEnabled,
		AutoCompressInterval: config.AutoCompressInterval,
	}
	cc.updateConfigInternal(internalConfig)
}

// CompressionStats tracks compression statistics
type CompressionStats struct {
	TotalCompressions    int
	TotalTokensSaved     int
	TotalMessagesRemoved int
	LastCompression      time.Time
	AverageRatio         float64
}

// Config represents compression configuration
type Config struct {
	Enabled              bool
	DefaultStrategy      CompressionStrategy
	TokenBudget          int
	WarningThreshold     int
	CompressionThreshold int
	AutoCompressEnabled  bool
	AutoCompressInterval time.Duration
}

// DefaultConfig returns default compression configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:              true,
		DefaultStrategy:      StrategyHybrid,
		TokenBudget:          200000,
		WarningThreshold:     150000,
		CompressionThreshold: 180000,
		AutoCompressEnabled:  true,
		AutoCompressInterval: 5 * time.Minute,
	}
}

// Conversation represents a conversation with messages
type Conversation struct {
	ID                 string
	Messages           []*Message
	Metadata           map[string]interface{}
	CreatedAt          time.Time
	UpdatedAt          time.Time
	TokenCount         int
	Compressed         bool
	CompressionHistory []*CompressionRecord
}

// Message represents a single message in a conversation
type Message struct {
	ID         string
	Role       MessageRole
	Content    string
	Timestamp  time.Time
	TokenCount int
	Metadata   MessageMetadata
	Pinned     bool
	Important  bool
}

// MessageRole specifies the role of a message
type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
)

// MessageMetadata stores additional message information
type MessageMetadata struct {
	Type       MessageType
	Context    []string
	References []string
	Tools      []string
	FilePaths  []string
	CodeBlocks int
	HasError   bool
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

// CompressionRecord tracks a compression operation
type CompressionRecord struct {
	Timestamp        time.Time
	Strategy         CompressionStrategy
	MessagesBefore   int
	MessagesAfter    int
	TokensBefore     int
	TokensAfter      int
	CompressionRatio float64
}

// CompressionResult contains the result of a compression operation
type CompressionResult struct {
	Original        *Conversation
	Compressed      *Conversation
	Strategy        CompressionStrategy
	TokensSaved     int
	MessagesRemoved int
	Summary         string
	Timestamp       time.Time
}

// CompressionEstimate estimates compression impact
type CompressionEstimate struct {
	TokensSaved     int
	MessagesRemoved int
	MessagesKept    int
	EstimatedRatio  float64
}

// Option is a functional option for CompressionCoordinator
type Option func(*CompressionCoordinator)

// WithConfig sets the configuration
func WithConfig(config *Config) Option {
	return func(cc *CompressionCoordinator) {
		cc.config = config
	}
}

// WithStrategy sets the default compression strategy
func WithStrategy(strategy CompressionStrategy) Option {
	return func(cc *CompressionCoordinator) {
		cc.config.DefaultStrategy = strategy
	}
}

// WithThreshold sets the compression threshold
func WithThreshold(threshold int) Option {
	return func(cc *CompressionCoordinator) {
		cc.config.CompressionThreshold = threshold
	}
}

// WithRetentionPolicy sets the retention policy
func WithRetentionPolicy(policy *RetentionPolicy) Option {
	return func(cc *CompressionCoordinator) {
		cc.retentionPolicy = policy
	}
}

// WithAutoCompress enables/disables auto-compression
func WithAutoCompress(enabled bool) Option {
	return func(cc *CompressionCoordinator) {
		cc.config.AutoCompressEnabled = enabled
	}
}

// NewCompressionCoordinator creates a new compression coordinator
func NewCompressionCoordinator(provider llm.Provider, opts ...Option) *CompressionCoordinator {
	cc := &CompressionCoordinator{
		engine:          NewCompressionEngine(provider),
		tokenCounter:    NewTokenCounter(),
		retentionPolicy: DefaultRetentionPolicy(),
		config:          DefaultConfig(),
		currentBudget:   200000,
	}

	for _, opt := range opts {
		opt(cc)
	}

	return cc
}

// compressInternal compresses a conversation using the configured strategy (internal method)
func (cc *CompressionCoordinator) compressInternal(ctx context.Context, conv *Conversation) (*CompressionResult, error) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	// Update token counts
	cc.tokenCounter.CountConversation(conv)

	// Execute compression
	result, err := cc.engine.Compress(ctx, conv, cc.config.DefaultStrategy, cc.retentionPolicy)
	if err != nil {
		return nil, fmt.Errorf("compression failed: %w", err)
	}

	// Update statistics
	cc.compressionCount++
	cc.stats.TotalCompressions++
	cc.stats.TotalTokensSaved += result.TokensSaved
	cc.stats.TotalMessagesRemoved += result.MessagesRemoved
	cc.stats.LastCompression = time.Now()

	if cc.stats.TotalCompressions > 0 {
		cc.stats.AverageRatio = float64(cc.stats.TotalTokensSaved) /
			float64(cc.stats.TotalTokensSaved+cc.tokenCounter.CountConversation(result.Compressed))
	}

	// Record compression history
	record := &CompressionRecord{
		Timestamp:        result.Timestamp,
		Strategy:         result.Strategy,
		MessagesBefore:   len(conv.Messages),
		MessagesAfter:    len(result.Compressed.Messages),
		TokensBefore:     conv.TokenCount,
		TokensAfter:      cc.tokenCounter.CountConversation(result.Compressed),
		CompressionRatio: float64(result.TokensSaved) / float64(conv.TokenCount),
	}

	result.Compressed.CompressionHistory = append(conv.CompressionHistory, record)
	result.Compressed.Compressed = true

	return result, nil
}

// shouldCompressInternal determines if compression is needed (internal method)
func (cc *CompressionCoordinator) shouldCompressInternal(conv *Conversation) (bool, string) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	// Count tokens
	tokenCount := cc.tokenCounter.CountConversation(conv)
	conv.TokenCount = tokenCount

	// Check against threshold
	if tokenCount >= cc.config.CompressionThreshold {
		return true, fmt.Sprintf("token count (%d) exceeds compression threshold (%d)",
			tokenCount, cc.config.CompressionThreshold)
	}

	// Check warning threshold
	if tokenCount >= cc.config.WarningThreshold {
		return false, fmt.Sprintf("token count (%d) approaching threshold (%d)",
			tokenCount, cc.config.CompressionThreshold)
	}

	return false, ""
}

// estimateCompressionInternal estimates the result of compression without executing it (internal method)
func (cc *CompressionCoordinator) estimateCompressionInternal(conv *Conversation) (*CompressionEstimate, error) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	strategy, err := cc.engine.GetStrategy(cc.config.DefaultStrategy)
	if err != nil {
		return nil, err
	}

	return strategy.Estimate(conv, cc.retentionPolicy)
}

// getStatsInternal returns compression statistics (internal method)
func (cc *CompressionCoordinator) getStatsInternal() *CompressionStats {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	return &CompressionStats{
		TotalCompressions:    cc.stats.TotalCompressions,
		TotalTokensSaved:     cc.stats.TotalTokensSaved,
		TotalMessagesRemoved: cc.stats.TotalMessagesRemoved,
		LastCompression:      cc.stats.LastCompression,
		AverageRatio:         cc.stats.AverageRatio,
	}
}

// getConfigInternal returns the current configuration (internal method)
func (cc *CompressionCoordinator) getConfigInternal() *Config {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	return cc.config
}

// updateConfigInternal updates the configuration (internal method)
func (cc *CompressionCoordinator) updateConfigInternal(config *Config) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.config = config
}

// TokenCounter counts tokens in messages
type TokenCounter struct {
	tokenizer Tokenizer
	cache     *TokenCache
}

// NewTokenCounter creates a new token counter
func NewTokenCounter() *TokenCounter {
	return &TokenCounter{
		tokenizer: &SimpleTokenizer{},
		cache:     NewTokenCache(1000),
	}
}

// Count counts tokens in content
func (tc *TokenCounter) Count(content string) int {
	// Check cache first
	if count, ok := tc.cache.Get(content); ok {
		return count
	}

	// Count tokens
	count := tc.tokenizer.Count(content)

	// Cache result
	tc.cache.Set(content, count)

	return count
}

// CountConversation counts total tokens in a conversation
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

// CountMessages counts tokens in multiple messages
func (tc *TokenCounter) CountMessages(messages []*Message) int {
	total := 0
	for _, msg := range messages {
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

// SimpleTokenizer provides a simple word-based tokenization
// In production, this should be replaced with tiktoken or similar
type SimpleTokenizer struct{}

// Count implements Tokenizer
func (st *SimpleTokenizer) Count(text string) int {
	// Simple approximation: ~4 characters per token
	// This is a rough estimate; in production use tiktoken
	if len(text) == 0 {
		return 0
	}
	return (len(text) + 3) / 4
}

// Encode implements Tokenizer
func (st *SimpleTokenizer) Encode(text string) []int {
	// Simplified encoding
	tokens := make([]int, st.Count(text))
	for i := range tokens {
		tokens[i] = i
	}
	return tokens
}

// Decode implements Tokenizer
func (st *SimpleTokenizer) Decode(tokens []int) string {
	// Simplified decoding
	return fmt.Sprintf("decoded_%d_tokens", len(tokens))
}

// TokenCache caches token counts
type TokenCache struct {
	mu      sync.RWMutex
	cache   map[string]int
	maxSize int
}

// NewTokenCache creates a new token cache
func NewTokenCache(maxSize int) *TokenCache {
	return &TokenCache{
		cache:   make(map[string]int),
		maxSize: maxSize,
	}
}

// Get retrieves a cached token count
func (tc *TokenCache) Get(content string) (int, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	hash := hashString(content)
	count, ok := tc.cache[hash]
	return count, ok
}

// Set stores a token count in the cache
func (tc *TokenCache) Set(content string, count int) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Evict if cache is full
	if len(tc.cache) >= tc.maxSize {
		// Remove a random entry (simple eviction)
		for k := range tc.cache {
			delete(tc.cache, k)
			break
		}
	}

	hash := hashString(content)
	tc.cache[hash] = count
}

// Clear clears the cache
func (tc *TokenCache) Clear() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.cache = make(map[string]int)
}

// hashString creates a hash of a string for caching
func hashString(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil)[:8]) // Use first 8 bytes for efficiency
}

// ConvertLLMMessage converts an llm.Message to a compression.Message
func ConvertLLMMessage(msg llm.Message) *Message {
	return &Message{
		ID:         uuid.New().String(),
		Role:       MessageRole(msg.Role),
		Content:    msg.Content,
		Timestamp:  time.Now(),
		TokenCount: 0,
		Metadata: MessageMetadata{
			Type:     TypeNormal,
			Context:  []string{},
			HasError: false,
		},
		Pinned:    false,
		Important: false,
	}
}

// ConvertToLLMMessage converts a compression.Message to an llm.Message
func ConvertToLLMMessage(msg *Message) llm.Message {
	return llm.Message{
		Role:    string(msg.Role),
		Content: msg.Content,
		Name:    "",
	}
}

// ConvertToLLMMessages converts multiple compression messages to llm messages
func ConvertToLLMMessages(messages []*Message) []llm.Message {
	llmMessages := make([]llm.Message, len(messages))
	for i, msg := range messages {
		llmMessages[i] = ConvertToLLMMessage(msg)
	}
	return llmMessages
}

// Conversion functions for interface compatibility

// ConvertToInternalConversation converts interface Conversation to internal Conversation
func ConvertToInternalConversation(ifaceConv *compressioniface.Conversation) *Conversation {
	messages := make([]*Message, len(ifaceConv.Messages))
	for i, ifaceMsg := range ifaceConv.Messages {
		messages[i] = &Message{
			ID:         ifaceMsg.ID,
			Role:       MessageRole(ifaceMsg.Role),
			Content:    ifaceMsg.Content,
			Timestamp:  ifaceMsg.Timestamp,
			TokenCount: ifaceMsg.TokenCount,
			Metadata: MessageMetadata{
				Type:       MessageType(ifaceMsg.Metadata.Type),
				Context:    ifaceMsg.Metadata.Context,
				References: ifaceMsg.Metadata.References,
				Tools:      ifaceMsg.Metadata.Tools,
				FilePaths:  ifaceMsg.Metadata.FilePaths,
				CodeBlocks: ifaceMsg.Metadata.CodeBlocks,
				HasError:   ifaceMsg.Metadata.HasError,
			},
			Pinned:    ifaceMsg.Pinned,
			Important: ifaceMsg.Important,
		}
	}

	return &Conversation{
		ID:                 ifaceConv.ID,
		Messages:           messages,
		Metadata:           ifaceConv.Metadata,
		CreatedAt:          ifaceConv.CreatedAt,
		UpdatedAt:          ifaceConv.UpdatedAt,
		TokenCount:         ifaceConv.TokenCount,
		Compressed:         ifaceConv.Compressed,
		CompressionHistory: ConvertToInternalCompressionRecords(ifaceConv.CompressionHistory),
	}
}

// ConvertToInternalCompressionRecords converts interface CompressionRecord slice to internal
func ConvertToInternalCompressionRecords(ifaceRecords []*compressioniface.CompressionRecord) []*CompressionRecord {
	if ifaceRecords == nil {
		return nil
	}

	records := make([]*CompressionRecord, len(ifaceRecords))
	for i, ifaceRecord := range ifaceRecords {
		records[i] = &CompressionRecord{
			Timestamp:        ifaceRecord.Timestamp,
			Strategy:         CompressionStrategy(ifaceRecord.Strategy),
			MessagesBefore:   ifaceRecord.MessagesBefore,
			MessagesAfter:    ifaceRecord.MessagesAfter,
			TokensBefore:     ifaceRecord.TokensBefore,
			TokensAfter:      ifaceRecord.TokensAfter,
			CompressionRatio: ifaceRecord.CompressionRatio,
		}
	}
	return records
}

// ConvertToInterfaceResult converts internal CompressionResult to interface CompressionResult
func ConvertToInterfaceResult(internalResult *CompressionResult) *compressioniface.CompressionResult {
	return &compressioniface.CompressionResult{
		Original:        ConvertToInterfaceConversation(internalResult.Original),
		Compressed:      ConvertToInterfaceConversation(internalResult.Compressed),
		Strategy:        compressioniface.CompressionStrategy(internalResult.Strategy),
		TokensSaved:     internalResult.TokensSaved,
		MessagesRemoved: internalResult.MessagesRemoved,
		Summary:         internalResult.Summary,
		Timestamp:       internalResult.Timestamp,
	}
}

// ConvertToInterfaceConversation converts internal Conversation to interface Conversation
func ConvertToInterfaceConversation(internalConv *Conversation) *compressioniface.Conversation {
	messages := make([]*compressioniface.Message, len(internalConv.Messages))
	for i, internalMsg := range internalConv.Messages {
		messages[i] = &compressioniface.Message{
			ID:         internalMsg.ID,
			Role:       compressioniface.MessageRole(internalMsg.Role),
			Content:    internalMsg.Content,
			Timestamp:  internalMsg.Timestamp,
			TokenCount: internalMsg.TokenCount,
			Metadata: compressioniface.MessageMetadata{
				Type:       compressioniface.MessageType(internalMsg.Metadata.Type),
				Context:    internalMsg.Metadata.Context,
				References: internalMsg.Metadata.References,
				Tools:      internalMsg.Metadata.Tools,
				FilePaths:  internalMsg.Metadata.FilePaths,
				CodeBlocks: internalMsg.Metadata.CodeBlocks,
				HasError:   internalMsg.Metadata.HasError,
			},
			Pinned:    internalMsg.Pinned,
			Important: internalMsg.Important,
		}
	}

	return &compressioniface.Conversation{
		ID:                 internalConv.ID,
		Messages:           messages,
		Metadata:           internalConv.Metadata,
		CreatedAt:          internalConv.CreatedAt,
		UpdatedAt:          internalConv.UpdatedAt,
		TokenCount:         internalConv.TokenCount,
		Compressed:         internalConv.Compressed,
		CompressionHistory: ConvertToInterfaceCompressionRecords(internalConv.CompressionHistory),
	}
}

// ConvertToInterfaceCompressionRecords converts internal CompressionRecord slice to interface
func ConvertToInterfaceCompressionRecords(internalRecords []*CompressionRecord) []*compressioniface.CompressionRecord {
	if internalRecords == nil {
		return nil
	}

	records := make([]*compressioniface.CompressionRecord, len(internalRecords))
	for i, internalRecord := range internalRecords {
		records[i] = &compressioniface.CompressionRecord{
			Timestamp:        internalRecord.Timestamp,
			Strategy:         compressioniface.CompressionStrategy(internalRecord.Strategy),
			MessagesBefore:   internalRecord.MessagesBefore,
			MessagesAfter:    internalRecord.MessagesAfter,
			TokensBefore:     internalRecord.TokensBefore,
			TokensAfter:      internalRecord.TokensAfter,
			CompressionRatio: internalRecord.CompressionRatio,
		}
	}
	return records
}

// ConvertToInterfaceEstimate converts internal CompressionEstimate to interface CompressionEstimate
func ConvertToInterfaceEstimate(internalEstimate *CompressionEstimate) *compressioniface.CompressionEstimate {
	return &compressioniface.CompressionEstimate{
		TokensSaved:     internalEstimate.TokensSaved,
		MessagesRemoved: internalEstimate.MessagesRemoved,
		MessagesKept:    internalEstimate.MessagesKept,
		EstimatedRatio:  internalEstimate.EstimatedRatio,
	}
}

// init registers the factory function
func init() {
	compressioniface.NewCoordinatorFactory = func(provider compressioniface.Provider, config *compressioniface.Config) (compressioniface.CompressionCoordinator, error) {
		var llmProvider llm.Provider

		// If provider is provided, cast it
		if provider != nil {
			var ok bool
			llmProvider, ok = provider.(llm.Provider)
			if !ok {
				return nil, fmt.Errorf("provider does not implement llm.Provider interface")
			}
		}
		// If provider is nil, compression will work with basic token counting but no semantic summarization

		// Create coordinator with config options
		return NewCompressionCoordinator(llmProvider,
			WithConfig(&Config{
				Enabled:              config.Enabled,
				DefaultStrategy:      CompressionStrategy(config.DefaultStrategy),
				TokenBudget:          config.TokenBudget,
				WarningThreshold:     config.WarningThreshold,
				CompressionThreshold: config.CompressionThreshold,
				AutoCompressEnabled:  config.AutoCompressEnabled,
				AutoCompressInterval: config.AutoCompressInterval,
			})), nil
	}
}
