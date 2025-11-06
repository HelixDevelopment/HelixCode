// Package compression provides conversation context compression functionality for HelixCode.
//
// Context compression manages conversation history by automatically summarizing and compressing
// older messages to stay within token budgets while preserving semantic meaning and important context.
//
// # Overview
//
// The compression package implements multiple strategies for reducing conversation token count:
//
//   - Sliding Window: Keeps the most recent N messages and discards older ones
//   - Semantic Summarization: Uses LLM to create concise summaries of message groups
//   - Hybrid: Combines sliding window with semantic summarization for optimal results
//
// # Architecture
//
// The package is organized into several key components:
//
//   - CompressionCoordinator: Main entry point for compression operations
//   - CompressionEngine: Executes compression strategies
//   - Strategy implementations: SlidingWindowStrategy, SemanticSummarizationStrategy, HybridStrategy
//   - RetentionPolicy: Defines rules for which messages to retain
//   - TokenCounter: Counts and caches token counts for messages
//
// # Basic Usage
//
// Create a compression coordinator and compress a conversation:
//
//	provider := getYourLLMProvider() // Any llm.Provider implementation
//	coordinator := compression.NewCompressionCoordinator(provider)
//
//	conversation := &compression.Conversation{
//	    ID: "conv-123",
//	    Messages: messages, // Your message slice
//	}
//
//	// Check if compression is needed
//	if shouldCompress, reason := coordinator.ShouldCompress(conversation); shouldCompress {
//	    result, err := coordinator.Compress(context.Background(), conversation)
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    fmt.Printf("Saved %d tokens, removed %d messages\n",
//	        result.TokensSaved, result.MessagesRemoved)
//	}
//
// # Compression Strategies
//
// ## Sliding Window
//
// The sliding window strategy keeps the most recent N messages and discards older ones.
// System messages and pinned messages are always retained regardless of window size.
//
//	coordinator := compression.NewCompressionCoordinator(
//	    provider,
//	    compression.WithStrategy(compression.StrategySlidingWindow),
//	)
//
// ## Semantic Summarization
//
// Semantic summarization uses an LLM to create concise summaries of older messages,
// preserving the semantic meaning while drastically reducing token count.
//
//	coordinator := compression.NewCompressionCoordinator(
//	    provider,
//	    compression.WithStrategy(compression.StrategySemanticSummarization),
//	)
//
// ## Hybrid Strategy
//
// The hybrid strategy combines both approaches: it keeps recent messages using a
// sliding window and summarizes older messages for maximum efficiency.
//
//	coordinator := compression.NewCompressionCoordinator(
//	    provider,
//	    compression.WithStrategy(compression.StrategyHybrid),
//	)
//
// # Retention Policies
//
// Retention policies define which messages should be preserved during compression.
// The package provides three preset policies:
//
//   - Conservative: Retains more messages (30 recent messages, 1 hour minimum age)
//   - Balanced: Default policy (10 recent messages, 30 minute minimum age)
//   - Aggressive: Retains fewer messages (5 recent messages, 10 minute minimum age)
//
// Example using a preset policy:
//
//	policy := compression.GetPolicyByPreset(compression.PresetConservative)
//	coordinator := compression.NewCompressionCoordinator(
//	    provider,
//	    compression.WithRetentionPolicy(policy),
//	)
//
// # Custom Retention Policies
//
// You can create custom retention policies using the PolicyBuilder:
//
//	policy := compression.NewPolicyBuilder().
//	    WithRecentCount(15).
//	    WithMinAge(1 * time.Hour).
//	    WithDefaultRules().
//	    AddRule(compression.RetentionRule{
//	        Priority: 11,
//	        Match: func(msg *compression.Message, pos compression.MessagePosition) bool {
//	            return msg.Metadata.Type == compression.TypeCommand
//	        },
//	        Action: compression.ActionRetain,
//	        Reason: "preserve_commands",
//	    }).
//	    Build()
//
// # Message Types and Metadata
//
// Messages can be categorized by type to apply different retention rules:
//
//   - TypeNormal: Regular conversation messages
//   - TypeCommand: Special commands that should be preserved
//   - TypeToolCall: Tool/function calls
//   - TypeToolResult: Results from tool executions
//   - TypeError: Error messages
//
// Messages can also have metadata including:
//
//   - Context: Related context tags
//   - References: References to other messages or resources
//   - Tools: Tools used in the message
//   - FilePaths: File paths mentioned in the message
//   - CodeBlocks: Number of code blocks in the message
//   - HasError: Whether the message contains an error
//
// # Pinning and Importance
//
// Individual messages can be marked as important or pinned to ensure they're retained:
//
//	message.Pinned = true    // Always retained during compression
//	message.Important = true // High priority for retention
//
// # Token Counting
//
// The package includes a token counter with caching for performance:
//
//	tc := compression.NewTokenCounter()
//	tokenCount := tc.Count("Your message content")
//	conversationTokens := tc.CountConversation(conversation)
//
// # Configuration Options
//
// The CompressionCoordinator accepts various configuration options:
//
//	coordinator := compression.NewCompressionCoordinator(
//	    provider,
//	    compression.WithStrategy(compression.StrategyHybrid),
//	    compression.WithThreshold(180000),              // Compress when over 180k tokens
//	    compression.WithAutoCompress(true),             // Enable auto-compression
//	    compression.WithRetentionPolicy(customPolicy),  // Custom retention policy
//	)
//
// # Statistics and Monitoring
//
// Track compression statistics:
//
//	stats := coordinator.GetStats()
//	fmt.Printf("Total compressions: %d\n", stats.TotalCompressions)
//	fmt.Printf("Total tokens saved: %d\n", stats.TotalTokensSaved)
//	fmt.Printf("Average compression ratio: %.2f\n", stats.AverageRatio)
//
// # Estimation
//
// Estimate compression impact before applying:
//
//	estimate, err := coordinator.EstimateCompression(conversation)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Would save %d tokens, remove %d messages\n",
//	    estimate.TokensSaved, estimate.MessagesRemoved)
//
// # Policy Analysis
//
// Analyze how a policy would affect your messages:
//
//	stats := compression.AnalyzePolicy(policy, messages)
//	fmt.Printf("Retention rate: %.2f%%\n", stats.RetentionRate*100)
//	for rule, count := range stats.RuleMatches {
//	    fmt.Printf("  %s: %d messages\n", rule, count)
//	}
//
// # Integration with LLM Package
//
// The compression package integrates seamlessly with the llm package:
//
//	// Convert LLM messages to compression messages
//	llmMessages := []llm.Message{...}
//	compMessages := make([]*compression.Message, len(llmMessages))
//	for i, msg := range llmMessages {
//	    compMessages[i] = compression.ConvertLLMMessage(msg)
//	}
//
//	// Convert back after compression
//	llmMessages = compression.ConvertToLLMMessages(compMessages)
//
// # Best Practices
//
//   - Use StrategyHybrid for most use cases as it balances compression ratio and semantic preservation
//   - Set compression threshold to 80-85% of your model's context window
//   - Always keep system messages (handled automatically by default)
//   - Pin critical messages that must be preserved
//   - Use Conservative policy for complex workflows, Aggressive for simple conversations
//   - Monitor compression statistics to tune your configuration
//   - Test compression with representative conversations before deploying
//
// # Performance Considerations
//
// The package includes several performance optimizations:
//
//   - Token count caching to avoid repeated calculations
//   - Efficient message filtering and sorting
//   - Parallel chunk processing for semantic summarization (when applicable)
//   - Minimal memory allocations during compression
//
// # Error Handling
//
// All compression operations return errors that should be handled:
//
//	result, err := coordinator.Compress(ctx, conversation)
//	if err != nil {
//	    // Handle error - possibly fall back to sliding window only
//	    log.Printf("Compression failed: %v", err)
//	    return
//	}
//
// # Thread Safety
//
// The CompressionCoordinator is thread-safe and can be used concurrently from
// multiple goroutines. Token counter caching is also thread-safe.
//
// # Examples
//
// See the test file (compression_test.go) for comprehensive examples of all features.
package compression
