package compression

import (
	"context"
	"fmt"
	"testing"
	"time"

	"dev.helix.code/internal/llm"
	"dev.helix.code/internal/llm/compressioniface"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLLMProvider implements llm.Provider for testing
type MockLLMProvider struct {
	summarizeFunc func(ctx context.Context, prompt string) (string, error)
}

func (m *MockLLMProvider) GetType() llm.ProviderType {
	return llm.ProviderTypeLocal
}

func (m *MockLLMProvider) GetName() string {
	return "mock"
}

func (m *MockLLMProvider) GetModels() []llm.ModelInfo {
	return []llm.ModelInfo{}
}

func (m *MockLLMProvider) GetCapabilities() []llm.ModelCapability {
	return []llm.ModelCapability{}
}

func (m *MockLLMProvider) Generate(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error) {
	if m.summarizeFunc != nil {
		summary, err := m.summarizeFunc(ctx, request.Messages[0].Content)
		if err != nil {
			return nil, err
		}

		return &llm.LLMResponse{
			ID:        uuid.New(),
			RequestID: request.ID,
			Content:   summary,
			Usage: llm.Usage{
				PromptTokens:     100,
				CompletionTokens: 50,
				TotalTokens:      150,
			},
			ProcessingTime: 100 * time.Millisecond,
			CreatedAt:      time.Now(),
		}, nil
	}

	return &llm.LLMResponse{
		ID:        uuid.New(),
		RequestID: request.ID,
		Content:   "Mock summary of the conversation",
		CreatedAt: time.Now(),
	}, nil
}

func (m *MockLLMProvider) GenerateStream(ctx context.Context, request *llm.LLMRequest, ch chan<- llm.LLMResponse) error {
	return nil
}

func (m *MockLLMProvider) IsAvailable(ctx context.Context) bool {
	return true
}

func (m *MockLLMProvider) GetHealth(ctx context.Context) (*llm.ProviderHealth, error) {
	return &llm.ProviderHealth{
		Status:    "healthy",
		Latency:   10 * time.Millisecond,
		LastCheck: time.Now(),
	}, nil
}

func (m *MockLLMProvider) Close() error {
	return nil
}

// Test helper functions

func createTestMessage(id string, role MessageRole, content string) *Message {
	return &Message{
		ID:         id,
		Role:       role,
		Content:    content,
		Timestamp:  time.Now(),
		TokenCount: (len(content) + 3) / 4,
		Metadata: MessageMetadata{
			Type:    TypeNormal,
			Context: []string{},
		},
		Pinned:    false,
		Important: false,
	}
}

func createTestConversation(messageCount int) *Conversation {
	messages := make([]*Message, messageCount)
	for i := 0; i < messageCount; i++ {
		role := RoleUser
		if i%2 == 1 {
			role = RoleAssistant
		}
		messages[i] = createTestMessage(
			fmt.Sprintf("msg-%d", i),
			role,
			fmt.Sprintf("Message %d with some content to test compression", i),
		)
	}

	return &Conversation{
		ID:       uuid.New().String(),
		Messages: messages,
		Metadata: make(map[string]interface{}),
	}
}

func createLargeConversation(messageCount int, tokensPerMsg int) *Conversation {
	messages := make([]*Message, messageCount)
	for i := 0; i < messageCount; i++ {
		role := RoleUser
		if i%2 == 1 {
			role = RoleAssistant
		}

		// Generate content with approximate token count
		content := ""
		for j := 0; j < tokensPerMsg*4; j++ {
			content += "a"
		}

		msg := createTestMessage(fmt.Sprintf("msg-%d", i), role, content)
		msg.TokenCount = tokensPerMsg
		messages[i] = msg
	}

	return &Conversation{
		ID:       uuid.New().String(),
		Messages: messages,
		Metadata: make(map[string]interface{}),
	}
}

// Test 1: Token Counter Basic Functionality
func TestTokenCounter_Count(t *testing.T) {
	tc := NewTokenCounter()

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "empty string",
			content: "",
		},
		{
			name:    "simple text",
			content: "Hello world",
		},
		{
			name:    "longer text",
			content: "This is a longer piece of text that should have more tokens",
		},
		{
			name:    "code block",
			content: "func main() { println(\"Hello\") }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := tc.Count(tt.content)
			assert.GreaterOrEqual(t, count, 0)

			// Test cache hit
			cachedCount := tc.Count(tt.content)
			assert.Equal(t, count, cachedCount)
		})
	}
}

// Test 2: Token Counter Conversation Counting
func TestTokenCounter_CountConversation(t *testing.T) {
	tc := NewTokenCounter()
	conv := createTestConversation(10)

	tokenCount := tc.CountConversation(conv)
	assert.Greater(t, tokenCount, 0)

	// Verify each message has token count set
	for _, msg := range conv.Messages {
		assert.Greater(t, msg.TokenCount, 0)
	}
}

// Test 3: Sliding Window Strategy Basic
func TestSlidingWindowStrategy_Execute(t *testing.T) {
	strategy := &SlidingWindowStrategy{
		windowSize: 5,
		keepPinned: true,
	}

	conv := createTestConversation(10)
	policy := DefaultRetentionPolicy()

	result, err := strategy.Execute(context.Background(), conv, policy)
	require.NoError(t, err)

	assert.Less(t, len(result.Compressed.Messages), len(conv.Messages))
	assert.Equal(t, 5, result.MessagesRemoved)
	assert.Greater(t, result.TokensSaved, 0)
	assert.Equal(t, StrategySlidingWindow, result.Strategy)
}

// Test 4: Sliding Window Strategy with Pinned Messages
func TestSlidingWindowStrategy_WithPinnedMessages(t *testing.T) {
	strategy := &SlidingWindowStrategy{
		windowSize: 5,
		keepPinned: true,
	}

	conv := createTestConversation(10)
	// Pin some messages
	conv.Messages[2].Pinned = true
	conv.Messages[5].Pinned = true

	policy := DefaultRetentionPolicy()

	result, err := strategy.Execute(context.Background(), conv, policy)
	require.NoError(t, err)

	// Check that pinned messages are retained
	pinnedFound := 0
	for _, msg := range result.Compressed.Messages {
		if msg.Pinned {
			pinnedFound++
		}
	}
	assert.Equal(t, 2, pinnedFound)
}

// Test 5: Sliding Window Strategy Estimate
func TestSlidingWindowStrategy_Estimate(t *testing.T) {
	strategy := &SlidingWindowStrategy{
		windowSize: 5,
		keepPinned: true,
	}

	conv := createTestConversation(20)
	policy := DefaultRetentionPolicy()

	estimate, err := strategy.Estimate(conv, policy)
	require.NoError(t, err)

	assert.Greater(t, estimate.TokensSaved, 0)
	assert.Greater(t, estimate.MessagesRemoved, 0)
	assert.Equal(t, 5, estimate.MessagesKept)
	assert.Greater(t, estimate.EstimatedRatio, 0.0)
}

// Test 6: Semantic Summarization Strategy
func TestSemanticSummarizationStrategy_Execute(t *testing.T) {
	mockProvider := &MockLLMProvider{
		summarizeFunc: func(ctx context.Context, prompt string) (string, error) {
			return "Summary of conversation messages", nil
		},
	}

	engine := NewCompressionEngine(mockProvider)
	strategy := engine.strategies[StrategySemanticSummarization].(*SemanticSummarizationStrategy)

	conv := createTestConversation(20)
	policy := DefaultRetentionPolicy()

	result, err := strategy.Execute(context.Background(), conv, policy)
	require.NoError(t, err)

	assert.Less(t, len(result.Compressed.Messages), len(conv.Messages))
	assert.Greater(t, result.TokensSaved, 0)
	assert.NotEmpty(t, result.Summary)
	assert.Equal(t, StrategySemanticSummarization, result.Strategy)
}

// Test 7: Semantic Summarization with Preserved Types
func TestSemanticSummarizationStrategy_PreserveTypes(t *testing.T) {
	mockProvider := &MockLLMProvider{
		summarizeFunc: func(ctx context.Context, prompt string) (string, error) {
			return "Summary", nil
		},
	}

	engine := NewCompressionEngine(mockProvider)
	strategy := engine.strategies[StrategySemanticSummarization].(*SemanticSummarizationStrategy)

	conv := createTestConversation(10)
	// Mark some messages with special types
	conv.Messages[2].Metadata.Type = TypeCommand
	conv.Messages[5].Metadata.Type = TypeError

	policy := DefaultRetentionPolicy()

	result, err := strategy.Execute(context.Background(), conv, policy)
	require.NoError(t, err)

	// Check that command and error messages are preserved
	commandFound := false
	errorFound := false
	for _, msg := range result.Compressed.Messages {
		if msg.Metadata.Type == TypeCommand {
			commandFound = true
		}
		if msg.Metadata.Type == TypeError {
			errorFound = true
		}
	}

	assert.True(t, commandFound, "Command message should be preserved")
	assert.True(t, errorFound, "Error message should be preserved")
}

// Test 8: Hybrid Strategy
func TestHybridStrategy_Execute(t *testing.T) {
	mockProvider := &MockLLMProvider{
		summarizeFunc: func(ctx context.Context, prompt string) (string, error) {
			return "Hybrid summary", nil
		},
	}

	engine := NewCompressionEngine(mockProvider)
	strategy := engine.strategies[StrategyHybrid].(*HybridStrategy)
	// Set a low threshold to force semantic compression
	strategy.threshold = 100

	conv := createLargeConversation(50, 100)
	policy := DefaultRetentionPolicy()

	result, err := strategy.Execute(context.Background(), conv, policy)
	require.NoError(t, err)

	assert.Less(t, len(result.Compressed.Messages), len(conv.Messages))
	assert.Greater(t, result.TokensSaved, 0)
	// Hybrid strategy may return sliding window or hybrid depending on conditions
	assert.True(t, result.Strategy == StrategyHybrid || result.Strategy == StrategySlidingWindow)
}

// Test 9: Retention Policy - System Messages
func TestRetentionPolicy_SystemMessages(t *testing.T) {
	policy := DefaultRetentionPolicy()

	systemMsg := createTestMessage("sys-1", RoleSystem, "System prompt")
	position := MessagePosition{Index: 0, IsFirst: true}

	shouldRetain := policy.ShouldRetain(systemMsg, position)
	assert.True(t, shouldRetain, "System messages should always be retained")
}

// Test 10: Retention Policy - Pinned Messages
func TestRetentionPolicy_PinnedMessages(t *testing.T) {
	policy := DefaultRetentionPolicy()

	pinnedMsg := createTestMessage("msg-1", RoleUser, "Important message")
	pinnedMsg.Pinned = true
	position := MessagePosition{Index: 5, IsRecent: false}

	shouldRetain := policy.ShouldRetain(pinnedMsg, position)
	assert.True(t, shouldRetain, "Pinned messages should be retained")
}

// Test 11: Retention Policy - Recent Messages
func TestRetentionPolicy_RecentMessages(t *testing.T) {
	policy := DefaultRetentionPolicy()

	recentMsg := createTestMessage("msg-1", RoleUser, "Recent message")
	position := MessagePosition{Index: 95, IsRecent: true}

	shouldRetain := policy.ShouldRetain(recentMsg, position)
	assert.True(t, shouldRetain, "Recent messages should be retained")
}

// Test 12: Retention Policy - Old Normal Messages
func TestRetentionPolicy_OldMessages(t *testing.T) {
	policy := DefaultRetentionPolicy()

	oldMsg := createTestMessage("msg-1", RoleUser, "Old message")
	position := MessagePosition{Index: 5, IsRecent: false}

	shouldRetain := policy.ShouldRetain(oldMsg, position)
	assert.False(t, shouldRetain, "Old normal messages should not be retained")
}

// Test 13: Compression Coordinator - ShouldCompress
func TestCompressionCoordinator_ShouldCompress(t *testing.T) {
	mockProvider := &MockLLMProvider{}
	coordinator := NewCompressionCoordinator(mockProvider, WithThreshold(1000))

	// Small conversation - should not compress
	smallConv := createTestConversation(10)
	ifaceSmallConv := ConvertToInterfaceConversation(smallConv)
	should, reason := coordinator.ShouldCompress(ifaceSmallConv)
	assert.False(t, should)
	assert.Empty(t, reason)

	// Large conversation - should compress
	largeConv := createLargeConversation(100, 50)
	ifaceLargeConv := ConvertToInterfaceConversation(largeConv)
	should, reason = coordinator.ShouldCompress(ifaceLargeConv)
	assert.True(t, should)
	assert.NotEmpty(t, reason)
}

// Test 14: Compression Coordinator - Full Compression Flow
func TestCompressionCoordinator_Compress(t *testing.T) {
	mockProvider := &MockLLMProvider{
		summarizeFunc: func(ctx context.Context, prompt string) (string, error) {
			return "Compressed summary", nil
		},
	}

	coordinator := NewCompressionCoordinator(
		mockProvider,
		WithStrategy(StrategySlidingWindow),
		WithThreshold(1000),
	)

	conv := createTestConversation(30)
	ifaceConv := ConvertToInterfaceConversation(conv)

	result, err := coordinator.Compress(context.Background(), ifaceConv)
	require.NoError(t, err)

	assert.NotNil(t, result)
	assert.Less(t, len(result.Compressed.Messages), len(conv.Messages))
	assert.Greater(t, result.TokensSaved, 0)

	// Check stats
	stats := coordinator.GetStats()
	assert.Equal(t, 1, stats.TotalCompressions)
	assert.Greater(t, stats.TotalTokensSaved, 0)
}

// Test 15: Compression Coordinator - EstimateCompression
func TestCompressionCoordinator_EstimateCompression(t *testing.T) {
	mockProvider := &MockLLMProvider{}
	coordinator := NewCompressionCoordinator(mockProvider)

	conv := createTestConversation(30)
	ifaceConv := ConvertToInterfaceConversation(conv)

	estimate, err := coordinator.EstimateCompression(ifaceConv)
	require.NoError(t, err)

	assert.Greater(t, estimate.TokensSaved, 0)
	assert.Greater(t, estimate.MessagesRemoved, 0)
	assert.Greater(t, estimate.MessagesKept, 0)
}

// Test 16: Policy Presets
func TestPolicyPresets(t *testing.T) {
	tests := []struct {
		name   string
		preset PolicyPreset
	}{
		{"conservative", PresetConservative},
		{"balanced", PresetBalanced},
		{"aggressive", PresetAggressive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := GetPolicyByPreset(tt.preset)
			assert.NotNil(t, policy)
			assert.NotEmpty(t, policy.GetRules())
		})
	}
}

// Test 17: Evaluate Policy
func TestEvaluatePolicy(t *testing.T) {
	policy := DefaultRetentionPolicy()
	messages := createTestConversation(20).Messages

	retained, compressed := EvaluatePolicy(policy, messages)

	assert.Greater(t, retained, 0)
	assert.GreaterOrEqual(t, compressed, 0)
	assert.Equal(t, len(messages), retained+compressed)
}

// Test 18: Analyze Policy
func TestAnalyzePolicy(t *testing.T) {
	policy := DefaultRetentionPolicy()
	messages := createTestConversation(20).Messages

	// Mark some messages as pinned or important
	messages[2].Pinned = true
	messages[5].Important = true

	stats := AnalyzePolicy(policy, messages)

	assert.Equal(t, len(messages), stats.TotalMessages)
	assert.Greater(t, stats.RetainedMessages, 0)
	assert.NotEmpty(t, stats.RuleMatches)
	assert.GreaterOrEqual(t, stats.RetentionRate, 0.0)
	assert.LessOrEqual(t, stats.RetentionRate, 1.0)
}

// Test 19: Policy Builder
func TestPolicyBuilder(t *testing.T) {
	policy := NewPolicyBuilder().
		WithRecentCount(15).
		WithMinAge(1 * time.Hour).
		WithDefaultRules().
		AddRule(RetentionRule{
			Priority: 11,
			Match: func(msg *Message, pos MessagePosition) bool {
				return msg.Content == "special"
			},
			Action: ActionRetain,
			Reason: "special_messages",
		}).
		Build()

	assert.NotNil(t, policy)
	assert.Equal(t, 15, policy.GetRecentCount())
	assert.Equal(t, 1*time.Hour, policy.GetMinAge())
	assert.NotEmpty(t, policy.GetRules())
}

// Test 20: Message Conversion
func TestMessageConversion(t *testing.T) {
	// Test LLM message to compression message
	llmMsg := llm.Message{
		Role:    "user",
		Content: "Test message",
	}

	compMsg := ConvertLLMMessage(llmMsg)
	assert.Equal(t, MessageRole(llmMsg.Role), compMsg.Role)
	assert.Equal(t, llmMsg.Content, compMsg.Content)

	// Test compression message to LLM message
	llmMsgBack := ConvertToLLMMessage(compMsg)
	assert.Equal(t, llmMsg.Role, llmMsgBack.Role)
	assert.Equal(t, llmMsg.Content, llmMsgBack.Content)
}

// Test 21: Token Cache
func TestTokenCache(t *testing.T) {
	cache := NewTokenCache(10)

	// Test set and get
	cache.Set("test content", 100)
	count, ok := cache.Get("test content")
	assert.True(t, ok)
	assert.Equal(t, 100, count)

	// Test cache miss
	_, ok = cache.Get("nonexistent")
	assert.False(t, ok)

	// Test cache eviction
	for i := 0; i < 15; i++ {
		cache.Set(fmt.Sprintf("content-%d", i), i*10)
	}

	// Cache should still be working
	cache.Set("final", 999)
	count, ok = cache.Get("final")
	assert.True(t, ok)
	assert.Equal(t, 999, count)

	// Test clear
	cache.Clear()
	_, ok = cache.Get("final")
	assert.False(t, ok)
}

// Test 22: Compression with System Messages
func TestCompression_PreserveSystemMessages(t *testing.T) {
	mockProvider := &MockLLMProvider{}
	coordinator := NewCompressionCoordinator(mockProvider)

	conv := createTestConversation(20)
	// Add system message at the start
	systemMsg := createTestMessage("sys-0", RoleSystem, "You are a helpful assistant")
	conv.Messages = append([]*Message{systemMsg}, conv.Messages...)
	ifaceConv := ConvertToInterfaceConversation(conv)

	result, err := coordinator.Compress(context.Background(), ifaceConv)
	require.NoError(t, err)

	// Verify system message is retained
	hasSystem := false
	for _, msg := range result.Compressed.Messages {
		if msg.Role == compressioniface.RoleSystem {
			hasSystem = true
			assert.Equal(t, "You are a helpful assistant", msg.Content)
			break
		}
	}
	assert.True(t, hasSystem, "System message should be preserved")
}

// Test 23: Compression Statistics
func TestCompressionCoordinator_Stats(t *testing.T) {
	mockProvider := &MockLLMProvider{}
	coordinator := NewCompressionCoordinator(mockProvider)

	// Initial stats
	stats := coordinator.GetStats()
	assert.Equal(t, 0, stats.TotalCompressions)

	// Perform compressions with large conversations to ensure compression happens
	for i := 0; i < 3; i++ {
		conv := createTestConversation(50) // Use 50 messages to exceed window size of 20
		ifaceConv := ConvertToInterfaceConversation(conv)
		_, err := coordinator.Compress(context.Background(), ifaceConv)
		require.NoError(t, err)
	}

	// Check updated stats
	stats = coordinator.GetStats()
	assert.Equal(t, 3, stats.TotalCompressions)
	assert.Greater(t, stats.TotalTokensSaved, 0)
	assert.Greater(t, stats.TotalMessagesRemoved, 0)
}

// Benchmark tests

func BenchmarkTokenCounter_Count(b *testing.B) {
	tc := NewTokenCounter()
	content := "This is a test message with some content"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tc.Count(content)
	}
}

func BenchmarkSlidingWindowStrategy(b *testing.B) {
	strategy := &SlidingWindowStrategy{windowSize: 20, keepPinned: true}
	conv := createTestConversation(100)
	policy := DefaultRetentionPolicy()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = strategy.Execute(context.Background(), conv, policy)
	}
}

func BenchmarkCompressionCoordinator_Compress(b *testing.B) {
	mockProvider := &MockLLMProvider{}
	coordinator := NewCompressionCoordinator(mockProvider)
	conv := createTestConversation(50)
	ifaceConv := ConvertToInterfaceConversation(conv)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = coordinator.Compress(context.Background(), ifaceConv)
	}
}
