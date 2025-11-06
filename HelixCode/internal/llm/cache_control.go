package llm

// CacheControl represents cache control directives for LLM providers
// Currently supported by Anthropic for prompt caching to reduce costs
type CacheControl struct {
	Type string `json:"type"` // "ephemeral" for Anthropic
}

// CacheableMessage represents a message that can be cached
type CacheableMessage struct {
	Message
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

// CacheStats tracks cache hit/miss statistics
type CacheStats struct {
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
}

// CacheStrategy defines when and what to cache
type CacheStrategy string

const (
	// CacheStrategyNone disables caching
	CacheStrategyNone CacheStrategy = "none"

	// CacheStrategySystem caches only system messages
	CacheStrategySystem CacheStrategy = "system"

	// CacheStrategyTools caches system messages and tool definitions
	CacheStrategyTools CacheStrategy = "tools"

	// CacheStrategyContext caches system, tools, and context prefix
	CacheStrategyContext CacheStrategy = "context"

	// CacheStrategyAggressive caches everything possible
	CacheStrategyAggressive CacheStrategy = "aggressive"
)

// CacheConfig holds configuration for prompt caching
type CacheConfig struct {
	// Enabled determines if caching is active
	Enabled bool `json:"enabled"`

	// Strategy determines what to cache
	Strategy CacheStrategy `json:"strategy"`

	// MinTokensForCache minimum tokens required to enable caching
	MinTokensForCache int `json:"min_tokens_for_cache"`

	// CacheTTL cache time-to-live in seconds (default: 300)
	CacheTTL int `json:"cache_ttl"`
}

// DefaultCacheConfig returns default caching configuration
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		Enabled:           true,
		Strategy:          CacheStrategyTools,
		MinTokensForCache: 1024, // Anthropic's minimum for efficient caching
		CacheTTL:          300,  // 5 minutes
	}
}

// ApplyCacheControl applies cache control to messages based on strategy
func ApplyCacheControl(messages []Message, tools []Tool, config CacheConfig) []CacheableMessage {
	if !config.Enabled || config.Strategy == CacheStrategyNone {
		return convertToCacheable(messages)
	}

	cacheableMessages := make([]CacheableMessage, len(messages))

	for i, msg := range messages {
		cacheableMessages[i] = CacheableMessage{
			Message: msg,
		}

		// Apply caching based on strategy
		switch config.Strategy {
		case CacheStrategySystem:
			// Cache only system message
			if i == 0 && msg.Role == "system" {
				cacheableMessages[i].CacheControl = &CacheControl{Type: "ephemeral"}
			}

		case CacheStrategyTools:
			// Cache system message if tools are present
			if i == 0 && msg.Role == "system" && len(tools) > 0 {
				cacheableMessages[i].CacheControl = &CacheControl{Type: "ephemeral"}
			}

		case CacheStrategyContext:
			// Cache system message and recent context
			if i == 0 && msg.Role == "system" {
				cacheableMessages[i].CacheControl = &CacheControl{Type: "ephemeral"}
			}
			// Cache the last few user messages to preserve context
			if msg.Role == "user" && i >= len(messages)-3 {
				cacheableMessages[i].CacheControl = &CacheControl{Type: "ephemeral"}
			}

		case CacheStrategyAggressive:
			// Cache system message
			if i == 0 && msg.Role == "system" {
				cacheableMessages[i].CacheControl = &CacheControl{Type: "ephemeral"}
			}
			// Cache all user messages (for multi-turn conversations)
			if msg.Role == "user" {
				cacheableMessages[i].CacheControl = &CacheControl{Type: "ephemeral"}
			}
		}
	}

	return cacheableMessages
}

// convertToCacheable converts regular messages to cacheable messages without cache control
func convertToCacheable(messages []Message) []CacheableMessage {
	cacheable := make([]CacheableMessage, len(messages))
	for i, msg := range messages {
		cacheable[i] = CacheableMessage{Message: msg}
	}
	return cacheable
}

// CalculateCacheSavings calculates cost savings from caching
func CalculateCacheSavings(stats CacheStats, inputCostPer1K, cacheCostPer1K float64) CacheSavings {
	// Calculate tokens that would have been charged at full price
	fullPriceTokens := stats.InputTokens + stats.CacheCreationInputTokens

	// Calculate actual cost with caching
	cacheCost := float64(stats.CacheCreationInputTokens) / 1000.0 * inputCostPer1K
	cacheReadCost := float64(stats.CacheReadInputTokens) / 1000.0 * cacheCostPer1K
	regularCost := float64(stats.InputTokens-stats.CacheReadInputTokens) / 1000.0 * inputCostPer1K

	actualCost := cacheCost + cacheReadCost + regularCost

	// Calculate cost without caching
	costWithoutCache := float64(fullPriceTokens) / 1000.0 * inputCostPer1K

	// Calculate savings
	savings := costWithoutCache - actualCost
	savingsPercent := 0.0
	if costWithoutCache > 0 {
		savingsPercent = (savings / costWithoutCache) * 100.0
	}

	return CacheSavings{
		CostWithCache:    actualCost,
		CostWithoutCache: costWithoutCache,
		Savings:          savings,
		SavingsPercent:   savingsPercent,
		TokensCached:     stats.CacheCreationInputTokens,
		TokensRead:       stats.CacheReadInputTokens,
	}
}

// CacheSavings represents the cost savings from caching
type CacheSavings struct {
	CostWithCache    float64 `json:"cost_with_cache"`
	CostWithoutCache float64 `json:"cost_without_cache"`
	Savings          float64 `json:"savings"`
	SavingsPercent   float64 `json:"savings_percent"`
	TokensCached     int     `json:"tokens_cached"`
	TokensRead       int     `json:"tokens_read"`
}

// CacheMetrics tracks overall caching performance
type CacheMetrics struct {
	TotalRequests          int     `json:"total_requests"`
	RequestsWithCache      int     `json:"requests_with_cache"`
	CacheHitRate           float64 `json:"cache_hit_rate"`
	TotalTokensCached      int     `json:"total_tokens_cached"`
	TotalTokensRead        int     `json:"total_tokens_read"`
	TotalSavings           float64 `json:"total_savings"`
	AverageSavingsPercent  float64 `json:"average_savings_percent"`
}

// UpdateMetrics updates cache metrics with new stats
func (cm *CacheMetrics) UpdateMetrics(stats CacheStats, savings CacheSavings) {
	cm.TotalRequests++

	if stats.CacheReadInputTokens > 0 || stats.CacheCreationInputTokens > 0 {
		cm.RequestsWithCache++
	}

	cm.TotalTokensCached += stats.CacheCreationInputTokens
	cm.TotalTokensRead += stats.CacheReadInputTokens
	cm.TotalSavings += savings.Savings

	// Update cache hit rate
	if cm.TotalRequests > 0 {
		cm.CacheHitRate = float64(cm.RequestsWithCache) / float64(cm.TotalRequests)
	}

	// Update average savings percent
	if cm.RequestsWithCache > 0 {
		cm.AverageSavingsPercent = (cm.AverageSavingsPercent*float64(cm.RequestsWithCache-1) + savings.SavingsPercent) / float64(cm.RequestsWithCache)
	}
}
