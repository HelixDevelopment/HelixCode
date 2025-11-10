package compressioniface

import (
	"context"
	"time"
)

// CompressionCoordinator defines the interface for conversation compression
type CompressionCoordinator interface {
	// Compress compresses a conversation using the configured strategy
	Compress(ctx context.Context, conv *Conversation) (*CompressionResult, error)

	// ShouldCompress determines if compression is needed
	ShouldCompress(conv *Conversation) (bool, string)

	// EstimateCompression estimates the result of compression without executing it
	EstimateCompression(conv *Conversation) (*CompressionEstimate, error)

	// GetStats returns compression statistics
	GetStats() *CompressionStats

	// GetConfig returns the current configuration
	GetConfig() *Config

	// UpdateConfig updates the configuration
	UpdateConfig(config *Config)
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

// CompressionStrategy defines available compression strategies
type CompressionStrategy int

const (
	StrategySlidingWindow CompressionStrategy = iota
	StrategySemanticSummarization
	StrategyHybrid
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
	default:
		return "unknown"
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

// CompressionStats tracks compression statistics
type CompressionStats struct {
	TotalCompressions    int
	TotalTokensSaved     int
	TotalMessagesRemoved int
	LastCompression      time.Time
	AverageRatio         float64
}

// Provider interface for LLM providers (minimal interface)
type Provider interface {
	// Add methods as needed
}

// Factory creates compression coordinators
type Factory interface {
	// NewCoordinator creates a new compression coordinator
	NewCoordinator(provider Provider, config *Config) (CompressionCoordinator, error)
}

// NewCoordinatorFactory creates a factory for compression coordinators
// This function is implemented in the compression package
var NewCoordinatorFactory func(provider Provider, config *Config) (CompressionCoordinator, error)
