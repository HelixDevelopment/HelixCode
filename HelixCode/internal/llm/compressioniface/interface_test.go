package compressioniface

import (
	"testing"
	"time"
)

// ========================================
// CompressionStrategy Tests
// ========================================

func TestCompressionStrategy_String(t *testing.T) {
	tests := []struct {
		name     string
		strategy CompressionStrategy
		expected string
	}{
		{"SlidingWindow", StrategySlidingWindow, "sliding_window"},
		{"SemanticSummarization", StrategySemanticSummarization, "semantic_summarization"},
		{"Hybrid", StrategyHybrid, "hybrid"},
		{"Unknown", CompressionStrategy(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.strategy.String(); got != tt.expected {
				t.Errorf("CompressionStrategy.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCompressionStrategy_Constants(t *testing.T) {
	// Verify ordering (iota)
	if StrategySlidingWindow != 0 {
		t.Error("StrategySlidingWindow should be 0")
	}
	if StrategySemanticSummarization != 1 {
		t.Error("StrategySemanticSummarization should be 1")
	}
	if StrategyHybrid != 2 {
		t.Error("StrategyHybrid should be 2")
	}
}

func TestCompressionStrategy_Uniqueness(t *testing.T) {
	strategies := []CompressionStrategy{
		StrategySlidingWindow,
		StrategySemanticSummarization,
		StrategyHybrid,
	}

	seen := make(map[string]bool)
	for _, strategy := range strategies {
		str := strategy.String()
		if seen[str] {
			t.Errorf("Duplicate strategy string found: %s", str)
		}
		seen[str] = true
	}

	if len(seen) != 3 {
		t.Errorf("Expected 3 unique strategies, got %d", len(seen))
	}
}

// ========================================
// MessageRole Tests
// ========================================

func TestMessageRole_Constants(t *testing.T) {
	tests := []struct {
		name     string
		role     MessageRole
		expected string
	}{
		{"System", RoleSystem, "system"},
		{"User", RoleUser, "user"},
		{"Assistant", RoleAssistant, "assistant"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.role) != tt.expected {
				t.Errorf("MessageRole %s = %v, want %v", tt.name, string(tt.role), tt.expected)
			}
		})
	}
}

func TestMessageRole_Uniqueness(t *testing.T) {
	roles := []MessageRole{
		RoleSystem,
		RoleUser,
		RoleAssistant,
	}

	seen := make(map[string]bool)
	for _, role := range roles {
		str := string(role)
		if seen[str] {
			t.Errorf("Duplicate role found: %s", str)
		}
		seen[str] = true
	}

	if len(seen) != 3 {
		t.Errorf("Expected 3 unique roles, got %d", len(seen))
	}
}

// ========================================
// MessageType Tests
// ========================================

func TestMessageType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		msgType  MessageType
		expected string
	}{
		{"Normal", TypeNormal, "normal"},
		{"Command", TypeCommand, "command"},
		{"ToolCall", TypeToolCall, "tool_call"},
		{"ToolResult", TypeToolResult, "tool_result"},
		{"Error", TypeError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.msgType) != tt.expected {
				t.Errorf("MessageType %s = %v, want %v", tt.name, string(tt.msgType), tt.expected)
			}
		})
	}
}

func TestMessageType_Uniqueness(t *testing.T) {
	types := []MessageType{
		TypeNormal,
		TypeCommand,
		TypeToolCall,
		TypeToolResult,
		TypeError,
	}

	seen := make(map[string]bool)
	for _, msgType := range types {
		str := string(msgType)
		if seen[str] {
			t.Errorf("Duplicate message type found: %s", str)
		}
		seen[str] = true
	}

	if len(seen) != 5 {
		t.Errorf("Expected 5 unique message types, got %d", len(seen))
	}
}

// ========================================
// Config Struct Tests
// ========================================

func TestConfig_Creation(t *testing.T) {
	config := &Config{
		Enabled:              true,
		DefaultStrategy:      StrategyHybrid,
		TokenBudget:          200000,
		WarningThreshold:     150000,
		CompressionThreshold: 180000,
		AutoCompressEnabled:  true,
		AutoCompressInterval: 5 * time.Minute,
	}

	if !config.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if config.DefaultStrategy != StrategyHybrid {
		t.Error("Expected DefaultStrategy to be StrategyHybrid")
	}
	if config.TokenBudget != 200000 {
		t.Error("Expected TokenBudget to be 200000")
	}
	if config.WarningThreshold != 150000 {
		t.Error("Expected WarningThreshold to be 150000")
	}
	if config.CompressionThreshold != 180000 {
		t.Error("Expected CompressionThreshold to be 180000")
	}
	if !config.AutoCompressEnabled {
		t.Error("Expected AutoCompressEnabled to be true")
	}
	if config.AutoCompressInterval != 5*time.Minute {
		t.Error("Expected AutoCompressInterval to be 5 minutes")
	}
}

func TestConfig_ZeroValues(t *testing.T) {
	config := &Config{}

	if config.Enabled {
		t.Error("Expected Enabled to default to false")
	}
	if config.DefaultStrategy != StrategySlidingWindow {
		t.Error("Expected DefaultStrategy to default to StrategySlidingWindow (0)")
	}
	if config.TokenBudget != 0 {
		t.Error("Expected TokenBudget to default to 0")
	}
}

// ========================================
// Conversation Struct Tests
// ========================================

func TestConversation_Creation(t *testing.T) {
	now := time.Now()
	conv := &Conversation{
		ID:         "conv-123",
		Messages:   []*Message{},
		Metadata:   map[string]interface{}{"key": "value"},
		CreatedAt:  now,
		UpdatedAt:  now,
		TokenCount: 1000,
		Compressed: false,
	}

	if conv.ID != "conv-123" {
		t.Error("Expected ID to be 'conv-123'")
	}
	if len(conv.Messages) != 0 {
		t.Error("Expected Messages to be empty")
	}
	if conv.Metadata["key"] != "value" {
		t.Error("Expected Metadata to contain key=value")
	}
	if conv.TokenCount != 1000 {
		t.Error("Expected TokenCount to be 1000")
	}
	if conv.Compressed {
		t.Error("Expected Compressed to be false")
	}
}

func TestConversation_WithMessages(t *testing.T) {
	message := &Message{
		ID:        "msg-1",
		Role:      RoleUser,
		Content:   "Hello",
		Timestamp: time.Now(),
	}

	conv := &Conversation{
		ID:       "conv-123",
		Messages: []*Message{message},
	}

	if len(conv.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(conv.Messages))
	}
	if conv.Messages[0].ID != "msg-1" {
		t.Error("Expected message ID to be 'msg-1'")
	}
	if conv.Messages[0].Role != RoleUser {
		t.Error("Expected message role to be RoleUser")
	}
}

// ========================================
// Message Struct Tests
// ========================================

func TestMessage_Creation(t *testing.T) {
	now := time.Now()
	msg := &Message{
		ID:         "msg-123",
		Role:       RoleAssistant,
		Content:    "Test message",
		Timestamp:  now,
		TokenCount: 10,
		Pinned:     true,
		Important:  false,
	}

	if msg.ID != "msg-123" {
		t.Error("Expected ID to be 'msg-123'")
	}
	if msg.Role != RoleAssistant {
		t.Error("Expected Role to be RoleAssistant")
	}
	if msg.Content != "Test message" {
		t.Error("Expected Content to be 'Test message'")
	}
	if msg.TokenCount != 10 {
		t.Error("Expected TokenCount to be 10")
	}
	if !msg.Pinned {
		t.Error("Expected Pinned to be true")
	}
	if msg.Important {
		t.Error("Expected Important to be false")
	}
}

func TestMessage_WithMetadata(t *testing.T) {
	metadata := MessageMetadata{
		Type:       TypeToolCall,
		Context:    []string{"context1"},
		References: []string{"ref1", "ref2"},
		Tools:      []string{"tool1"},
		FilePaths:  []string{"/path/to/file"},
		CodeBlocks: 2,
		HasError:   false,
	}

	msg := &Message{
		ID:       "msg-123",
		Role:     RoleUser,
		Content:  "Test",
		Metadata: metadata,
	}

	if msg.Metadata.Type != TypeToolCall {
		t.Error("Expected Metadata.Type to be TypeToolCall")
	}
	if len(msg.Metadata.Context) != 1 {
		t.Error("Expected 1 context item")
	}
	if len(msg.Metadata.References) != 2 {
		t.Error("Expected 2 references")
	}
	if len(msg.Metadata.Tools) != 1 {
		t.Error("Expected 1 tool")
	}
	if len(msg.Metadata.FilePaths) != 1 {
		t.Error("Expected 1 file path")
	}
	if msg.Metadata.CodeBlocks != 2 {
		t.Error("Expected 2 code blocks")
	}
	if msg.Metadata.HasError {
		t.Error("Expected HasError to be false")
	}
}

// ========================================
// MessageMetadata Struct Tests
// ========================================

func TestMessageMetadata_AllFields(t *testing.T) {
	metadata := MessageMetadata{
		Type:       TypeError,
		Context:    []string{"ctx1", "ctx2", "ctx3"},
		References: []string{"ref1"},
		Tools:      []string{"bash", "read", "write"},
		FilePaths:  []string{"/a", "/b"},
		CodeBlocks: 5,
		HasError:   true,
	}

	if metadata.Type != TypeError {
		t.Error("Expected Type to be TypeError")
	}
	if len(metadata.Context) != 3 {
		t.Errorf("Expected 3 context items, got %d", len(metadata.Context))
	}
	if len(metadata.References) != 1 {
		t.Errorf("Expected 1 reference, got %d", len(metadata.References))
	}
	if len(metadata.Tools) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(metadata.Tools))
	}
	if len(metadata.FilePaths) != 2 {
		t.Errorf("Expected 2 file paths, got %d", len(metadata.FilePaths))
	}
	if metadata.CodeBlocks != 5 {
		t.Errorf("Expected 5 code blocks, got %d", metadata.CodeBlocks)
	}
	if !metadata.HasError {
		t.Error("Expected HasError to be true")
	}
}

// ========================================
// CompressionRecord Struct Tests
// ========================================

func TestCompressionRecord_Creation(t *testing.T) {
	now := time.Now()
	record := &CompressionRecord{
		Timestamp:        now,
		Strategy:         StrategySemanticSummarization,
		MessagesBefore:   100,
		MessagesAfter:    50,
		TokensBefore:     10000,
		TokensAfter:      5000,
		CompressionRatio: 0.5,
	}

	if record.Strategy != StrategySemanticSummarization {
		t.Error("Expected Strategy to be StrategySemanticSummarization")
	}
	if record.MessagesBefore != 100 {
		t.Error("Expected MessagesBefore to be 100")
	}
	if record.MessagesAfter != 50 {
		t.Error("Expected MessagesAfter to be 50")
	}
	if record.TokensBefore != 10000 {
		t.Error("Expected TokensBefore to be 10000")
	}
	if record.TokensAfter != 5000 {
		t.Error("Expected TokensAfter to be 5000")
	}
	if record.CompressionRatio != 0.5 {
		t.Errorf("Expected CompressionRatio to be 0.5, got %f", record.CompressionRatio)
	}
}

func TestCompressionRecord_RatioCalculation(t *testing.T) {
	tests := []struct {
		name         string
		before       int
		after        int
		expectedRatio float64
	}{
		{"50% compression", 10000, 5000, 0.5},
		{"75% compression", 10000, 2500, 0.25},
		{"No compression", 10000, 10000, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &CompressionRecord{
				TokensBefore:     tt.before,
				TokensAfter:      tt.after,
				CompressionRatio: float64(tt.after) / float64(tt.before),
			}

			if record.CompressionRatio != tt.expectedRatio {
				t.Errorf("Expected ratio %f, got %f", tt.expectedRatio, record.CompressionRatio)
			}
		})
	}
}

// ========================================
// CompressionResult Struct Tests
// ========================================

func TestCompressionResult_Creation(t *testing.T) {
	original := &Conversation{ID: "orig", TokenCount: 10000}
	compressed := &Conversation{ID: "comp", TokenCount: 5000}
	now := time.Now()

	result := &CompressionResult{
		Original:        original,
		Compressed:      compressed,
		Strategy:        StrategyHybrid,
		TokensSaved:     5000,
		MessagesRemoved: 50,
		Summary:         "Compression successful",
		Timestamp:       now,
	}

	if result.Original.ID != "orig" {
		t.Error("Expected Original ID to be 'orig'")
	}
	if result.Compressed.ID != "comp" {
		t.Error("Expected Compressed ID to be 'comp'")
	}
	if result.Strategy != StrategyHybrid {
		t.Error("Expected Strategy to be StrategyHybrid")
	}
	if result.TokensSaved != 5000 {
		t.Error("Expected TokensSaved to be 5000")
	}
	if result.MessagesRemoved != 50 {
		t.Error("Expected MessagesRemoved to be 50")
	}
	if result.Summary != "Compression successful" {
		t.Error("Expected Summary to match")
	}
}

// ========================================
// CompressionEstimate Struct Tests
// ========================================

func TestCompressionEstimate_Creation(t *testing.T) {
	estimate := &CompressionEstimate{
		TokensSaved:     3000,
		MessagesRemoved: 30,
		MessagesKept:    70,
		EstimatedRatio:  0.7,
	}

	if estimate.TokensSaved != 3000 {
		t.Error("Expected TokensSaved to be 3000")
	}
	if estimate.MessagesRemoved != 30 {
		t.Error("Expected MessagesRemoved to be 30")
	}
	if estimate.MessagesKept != 70 {
		t.Error("Expected MessagesKept to be 70")
	}
	if estimate.EstimatedRatio != 0.7 {
		t.Errorf("Expected EstimatedRatio to be 0.7, got %f", estimate.EstimatedRatio)
	}
}

// ========================================
// CompressionStats Struct Tests
// ========================================

func TestCompressionStats_Creation(t *testing.T) {
	now := time.Now()
	stats := &CompressionStats{
		TotalCompressions:    10,
		TotalTokensSaved:     50000,
		TotalMessagesRemoved: 500,
		LastCompression:      now,
		AverageRatio:         0.6,
	}

	if stats.TotalCompressions != 10 {
		t.Error("Expected TotalCompressions to be 10")
	}
	if stats.TotalTokensSaved != 50000 {
		t.Error("Expected TotalTokensSaved to be 50000")
	}
	if stats.TotalMessagesRemoved != 500 {
		t.Error("Expected TotalMessagesRemoved to be 500")
	}
	if stats.AverageRatio != 0.6 {
		t.Errorf("Expected AverageRatio to be 0.6, got %f", stats.AverageRatio)
	}
}

func TestCompressionStats_Accumulation(t *testing.T) {
	stats := &CompressionStats{}

	// Simulate adding compression results
	stats.TotalCompressions = 3
	stats.TotalTokensSaved = 15000
	stats.TotalMessagesRemoved = 150
	stats.AverageRatio = (0.5 + 0.6 + 0.7) / 3.0

	if stats.TotalCompressions != 3 {
		t.Errorf("Expected 3 compressions, got %d", stats.TotalCompressions)
	}

	expectedAvg := 0.6
	if stats.AverageRatio != expectedAvg {
		t.Errorf("Expected average ratio %f, got %f", expectedAvg, stats.AverageRatio)
	}
}

// ========================================
// Edge Cases and Special Scenarios
// ========================================

func TestConversation_EmptyMessages(t *testing.T) {
	conv := &Conversation{
		ID:       "empty-conv",
		Messages: []*Message{},
	}

	if conv.Messages == nil {
		t.Error("Expected Messages to be non-nil empty slice")
	}
	if len(conv.Messages) != 0 {
		t.Error("Expected 0 messages")
	}
}

func TestConversation_LargeMessageCount(t *testing.T) {
	messages := make([]*Message, 1000)
	for i := 0; i < 1000; i++ {
		messages[i] = &Message{
			ID:      string(rune('a' + i%26)),
			Role:    RoleUser,
			Content: "Test",
		}
	}

	conv := &Conversation{
		ID:       "large-conv",
		Messages: messages,
	}

	if len(conv.Messages) != 1000 {
		t.Errorf("Expected 1000 messages, got %d", len(conv.Messages))
	}
}

func TestMessage_EmptyContent(t *testing.T) {
	msg := &Message{
		ID:      "empty-msg",
		Role:    RoleSystem,
		Content: "",
	}

	if msg.Content != "" {
		t.Error("Expected empty content")
	}
}

func TestMessageMetadata_EmptyArrays(t *testing.T) {
	metadata := MessageMetadata{
		Type:       TypeNormal,
		Context:    []string{},
		References: []string{},
		Tools:      []string{},
		FilePaths:  []string{},
		CodeBlocks: 0,
		HasError:   false,
	}

	if len(metadata.Context) != 0 {
		t.Error("Expected empty Context array")
	}
	if len(metadata.References) != 0 {
		t.Error("Expected empty References array")
	}
	if len(metadata.Tools) != 0 {
		t.Error("Expected empty Tools array")
	}
	if len(metadata.FilePaths) != 0 {
		t.Error("Expected empty FilePaths array")
	}
}

func TestCompressionRecord_ZeroCompression(t *testing.T) {
	record := &CompressionRecord{
		Strategy:         StrategySlidingWindow,
		MessagesBefore:   100,
		MessagesAfter:    100,
		TokensBefore:     10000,
		TokensAfter:      10000,
		CompressionRatio: 1.0,
	}

	if record.CompressionRatio != 1.0 {
		t.Error("Expected no compression (ratio = 1.0)")
	}
	if record.MessagesBefore != record.MessagesAfter {
		t.Error("Expected same message count")
	}
}

func TestCompressionEstimate_MaxCompression(t *testing.T) {
	estimate := &CompressionEstimate{
		TokensSaved:     9000,
		MessagesRemoved: 90,
		MessagesKept:    10,
		EstimatedRatio:  0.1,
	}

	if estimate.EstimatedRatio < 0 || estimate.EstimatedRatio > 1 {
		t.Error("Ratio should be between 0 and 1")
	}
}

// ========================================
// Type Conversion Tests
// ========================================

func TestMessageRole_StringConversion(t *testing.T) {
	role := RoleUser
	str := string(role)

	if str != "user" {
		t.Errorf("Expected 'user', got %s", str)
	}

	converted := MessageRole(str)
	if converted != role {
		t.Error("Conversion back to MessageRole failed")
	}
}

func TestMessageType_StringConversion(t *testing.T) {
	msgType := TypeCommand
	str := string(msgType)

	if str != "command" {
		t.Errorf("Expected 'command', got %s", str)
	}

	converted := MessageType(str)
	if converted != msgType {
		t.Error("Conversion back to MessageType failed")
	}
}
