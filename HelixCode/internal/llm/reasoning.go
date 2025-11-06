package llm

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// ReasoningConfig holds configuration for reasoning model support
type ReasoningConfig struct {
	// Enabled indicates whether reasoning mode is enabled
	Enabled bool `json:"enabled"`

	// ExtractThinking determines if thinking blocks should be extracted
	ExtractThinking bool `json:"extract_thinking"`

	// HideFromUser determines if thinking should be hidden from the end user
	HideFromUser bool `json:"hide_from_user"`

	// ThinkingTags specifies the XML tags used for thinking blocks (e.g., "thinking", "internal_thoughts")
	ThinkingTags string `json:"thinking_tags"`

	// ThinkingBudget specifies the token budget for thinking (0 = unlimited)
	ThinkingBudget int `json:"thinking_budget"`

	// ReasoningEffort specifies the effort level for reasoning models
	// Valid values: "low", "medium", "high"
	ReasoningEffort string `json:"reasoning_effort"`

	// Model-specific configuration
	ModelType ReasoningModelType `json:"model_type"`
}

// ReasoningModelType identifies the type of reasoning model
type ReasoningModelType string

const (
	// OpenAI o1/o3/o4 series
	ReasoningModelOpenAI_O1     ReasoningModelType = "openai_o1"
	ReasoningModelOpenAI_O3     ReasoningModelType = "openai_o3"
	ReasoningModelOpenAI_O4     ReasoningModelType = "openai_o4"

	// Claude reasoning modes
	ReasoningModelClaude_Opus   ReasoningModelType = "claude_opus"
	ReasoningModelClaude_Sonnet ReasoningModelType = "claude_sonnet"

	// DeepSeek reasoning models
	ReasoningModelDeepSeek_R1       ReasoningModelType = "deepseek_r1"
	ReasoningModelDeepSeek_Reasoner ReasoningModelType = "deepseek_reasoner"

	// QwQ-32B reasoning model
	ReasoningModelQwQ_32B ReasoningModelType = "qwq_32b"

	// Generic reasoning model
	ReasoningModelGeneric ReasoningModelType = "generic"
)

// ReasoningEffortLevel defines the effort level for reasoning
type ReasoningEffortLevel string

const (
	ReasoningEffortLow    ReasoningEffortLevel = "low"
	ReasoningEffortMedium ReasoningEffortLevel = "medium"
	ReasoningEffortHigh   ReasoningEffortLevel = "high"
)

// ReasoningTrace represents extracted reasoning/thinking information
type ReasoningTrace struct {
	// ThinkingContent contains the extracted thinking blocks
	ThinkingContent []string `json:"thinking_content"`

	// OutputContent contains the final output without thinking
	OutputContent string `json:"output_content"`

	// ThinkingTokens tracks tokens used for thinking
	ThinkingTokens int `json:"thinking_tokens"`

	// OutputTokens tracks tokens used for output
	OutputTokens int `json:"output_tokens"`

	// TotalTokens is the sum of thinking and output tokens
	TotalTokens int `json:"total_tokens"`
}

// DefaultReasoningConfig returns a default reasoning configuration
func DefaultReasoningConfig() *ReasoningConfig {
	return &ReasoningConfig{
		Enabled:         false,
		ExtractThinking: true,
		HideFromUser:    false,
		ThinkingTags:    "thinking",
		ThinkingBudget:  0, // unlimited
		ReasoningEffort: string(ReasoningEffortMedium),
		ModelType:       ReasoningModelGeneric,
	}
}

// NewReasoningConfig creates a new reasoning config for a specific model
func NewReasoningConfig(modelType ReasoningModelType) *ReasoningConfig {
	config := DefaultReasoningConfig()
	config.ModelType = modelType
	config.Enabled = true

	// Set model-specific defaults
	switch modelType {
	case ReasoningModelOpenAI_O1, ReasoningModelOpenAI_O3, ReasoningModelOpenAI_O4:
		config.ThinkingTags = "thinking"
		config.ExtractThinking = true
		config.HideFromUser = false
		config.ThinkingBudget = 10000 // OpenAI o1 uses thinking tokens

	case ReasoningModelClaude_Opus, ReasoningModelClaude_Sonnet:
		config.ThinkingTags = "thinking"
		config.ExtractThinking = true
		config.HideFromUser = false
		config.ThinkingBudget = 5000 // Claude extended thinking budget

	case ReasoningModelDeepSeek_R1, ReasoningModelDeepSeek_Reasoner:
		config.ThinkingTags = "think"
		config.ExtractThinking = true
		config.HideFromUser = false
		config.ThinkingBudget = 8000

	case ReasoningModelQwQ_32B:
		config.ThinkingTags = "thinking"
		config.ExtractThinking = true
		config.HideFromUser = false
		config.ThinkingBudget = 7000
	}

	return config
}

// ExtractReasoningTrace extracts thinking blocks and separates them from output
func ExtractReasoningTrace(content string, config *ReasoningConfig) *ReasoningTrace {
	if !config.Enabled || !config.ExtractThinking {
		return &ReasoningTrace{
			ThinkingContent: []string{},
			OutputContent:   content,
			ThinkingTokens:  0,
			OutputTokens:    estimateTokens(content),
			TotalTokens:     estimateTokens(content),
		}
	}

	trace := &ReasoningTrace{
		ThinkingContent: []string{},
	}

	// Extract thinking blocks based on configured tags
	tags := strings.Split(config.ThinkingTags, ",")
	remainingContent := content

	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		thinkingBlocks, cleanContent := extractThinkingBlocks(remainingContent, tag)
		trace.ThinkingContent = append(trace.ThinkingContent, thinkingBlocks...)
		remainingContent = cleanContent
	}

	trace.OutputContent = strings.TrimSpace(remainingContent)

	// Calculate token counts
	for _, thinking := range trace.ThinkingContent {
		trace.ThinkingTokens += estimateTokens(thinking)
	}
	trace.OutputTokens = estimateTokens(trace.OutputContent)
	trace.TotalTokens = trace.ThinkingTokens + trace.OutputTokens

	return trace
}

// extractThinkingBlocks extracts content from specified XML-style tags
func extractThinkingBlocks(content string, tag string) ([]string, string) {
	var blocks []string

	// Create regex pattern for opening and closing tags
	// Handles both <tag> and <tag ...> formats
	// (?s) enables dot to match newlines (DOTALL mode)
	pattern := fmt.Sprintf(`(?s)<%s[^>]*>(.*?)</%s>`, regexp.QuoteMeta(tag), regexp.QuoteMeta(tag))
	re := regexp.MustCompile(pattern)

	// Extract all thinking blocks
	matches := re.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			blocks = append(blocks, strings.TrimSpace(match[1]))
		}
	}

	// Remove thinking blocks from content
	cleanContent := re.ReplaceAllString(content, "")

	return blocks, cleanContent
}

// ApplyReasoningBudget enforces token budget constraints for reasoning
func ApplyReasoningBudget(trace *ReasoningTrace, config *ReasoningConfig) *ReasoningTrace {
	if !config.Enabled || config.ThinkingBudget == 0 {
		return trace
	}

	// If thinking tokens exceed budget, truncate thinking content
	if trace.ThinkingTokens > config.ThinkingBudget {
		budgetedTrace := &ReasoningTrace{
			ThinkingContent: []string{},
			OutputContent:   trace.OutputContent,
			OutputTokens:    trace.OutputTokens,
		}

		remainingBudget := config.ThinkingBudget

		for _, thinking := range trace.ThinkingContent {
			thinkingTokens := estimateTokens(thinking)

			if remainingBudget >= thinkingTokens {
				budgetedTrace.ThinkingContent = append(budgetedTrace.ThinkingContent, thinking)
				budgetedTrace.ThinkingTokens += thinkingTokens
				remainingBudget -= thinkingTokens
			} else if remainingBudget > 0 {
				// Truncate the last block to fit budget
				truncated := truncateToTokenBudget(thinking, remainingBudget)
				budgetedTrace.ThinkingContent = append(budgetedTrace.ThinkingContent, truncated)
				budgetedTrace.ThinkingTokens += remainingBudget
				remainingBudget = 0
				break
			} else {
				break
			}
		}

		budgetedTrace.TotalTokens = budgetedTrace.ThinkingTokens + budgetedTrace.OutputTokens
		return budgetedTrace
	}

	return trace
}

// ValidateReasoningEffort validates and normalizes the reasoning effort level
func ValidateReasoningEffort(effort string) (ReasoningEffortLevel, error) {
	normalized := strings.ToLower(strings.TrimSpace(effort))

	switch normalized {
	case "low":
		return ReasoningEffortLow, nil
	case "medium", "med", "":
		return ReasoningEffortMedium, nil
	case "high":
		return ReasoningEffortHigh, nil
	default:
		return "", fmt.Errorf("invalid reasoning effort level: %s (valid: low, medium, high)", effort)
	}
}

// FormatReasoningPrompt formats a prompt for reasoning models
func FormatReasoningPrompt(prompt string, config *ReasoningConfig) string {
	if !config.Enabled {
		return prompt
	}

	var formatted strings.Builder

	// Add reasoning mode instructions based on model type
	switch config.ModelType {
	case ReasoningModelOpenAI_O1, ReasoningModelOpenAI_O3, ReasoningModelOpenAI_O4:
		// OpenAI o-series models handle reasoning automatically
		// No special prompt formatting needed
		formatted.WriteString(prompt)

	case ReasoningModelClaude_Opus, ReasoningModelClaude_Sonnet:
		// Claude extended thinking - encourage use of thinking tags
		formatted.WriteString("Please approach this problem step-by-step, showing your reasoning process.\n\n")
		formatted.WriteString(prompt)

	case ReasoningModelDeepSeek_R1, ReasoningModelDeepSeek_Reasoner:
		// DeepSeek models expect <think> tags
		formatted.WriteString("Use <think></think> tags to show your reasoning process.\n\n")
		formatted.WriteString(prompt)

	case ReasoningModelQwQ_32B:
		// QwQ-32B expects detailed reasoning
		formatted.WriteString("Think through this problem carefully, showing your reasoning:\n\n")
		formatted.WriteString(prompt)

	default:
		// Generic reasoning model
		formatted.WriteString("Please reason through this problem step-by-step:\n\n")
		formatted.WriteString(prompt)
	}

	// Add effort level guidance
	if config.ReasoningEffort != "" {
		effort, err := ValidateReasoningEffort(config.ReasoningEffort)
		if err == nil {
			switch effort {
			case ReasoningEffortLow:
				formatted.WriteString("\n\nProvide a brief analysis.")
			case ReasoningEffortMedium:
				formatted.WriteString("\n\nProvide a thorough analysis.")
			case ReasoningEffortHigh:
				formatted.WriteString("\n\nProvide a comprehensive, detailed analysis with careful consideration of all aspects.")
			}
		}
	}

	return formatted.String()
}

// IsReasoningModel checks if a model name indicates a reasoning model
func IsReasoningModel(modelName string) (bool, ReasoningModelType) {
	modelName = strings.ToLower(modelName)

	// OpenAI o-series
	if strings.Contains(modelName, "o1") || strings.Contains(modelName, "o1-preview") || strings.Contains(modelName, "o1-mini") {
		return true, ReasoningModelOpenAI_O1
	}
	if strings.Contains(modelName, "o3") {
		return true, ReasoningModelOpenAI_O3
	}
	if strings.Contains(modelName, "o4") {
		return true, ReasoningModelOpenAI_O4
	}

	// Claude models with extended thinking
	if strings.Contains(modelName, "claude") && strings.Contains(modelName, "opus") {
		return true, ReasoningModelClaude_Opus
	}
	if strings.Contains(modelName, "claude") && strings.Contains(modelName, "sonnet") {
		return true, ReasoningModelClaude_Sonnet
	}

	// DeepSeek reasoning models
	if strings.Contains(modelName, "deepseek") && (strings.Contains(modelName, "r1") || strings.Contains(modelName, "reasoner")) {
		return true, ReasoningModelDeepSeek_R1
	}

	// QwQ-32B
	if strings.Contains(modelName, "qwq") {
		return true, ReasoningModelQwQ_32B
	}

	return false, ReasoningModelGeneric
}

// CalculateReasoningCost calculates the cost of a reasoning request
// Returns (thinkingCost, outputCost, totalCost)
func CalculateReasoningCost(trace *ReasoningTrace, config *ReasoningConfig, modelType ReasoningModelType) (float64, float64, float64) {
	// Cost per 1M tokens (in USD)
	var thinkingCostPer1M, outputCostPer1M float64

	switch modelType {
	case ReasoningModelOpenAI_O1:
		// o1-preview pricing: $15/1M input, $60/1M output, thinking tokens at input rate
		thinkingCostPer1M = 15.0
		outputCostPer1M = 60.0

	case ReasoningModelOpenAI_O3:
		// o3 pricing (estimated): $20/1M input, $80/1M output
		thinkingCostPer1M = 20.0
		outputCostPer1M = 80.0

	case ReasoningModelOpenAI_O4:
		// o4 pricing (estimated): $25/1M input, $100/1M output
		thinkingCostPer1M = 25.0
		outputCostPer1M = 100.0

	case ReasoningModelClaude_Opus:
		// Claude Opus with extended thinking: $15/1M input, $75/1M output
		thinkingCostPer1M = 15.0
		outputCostPer1M = 75.0

	case ReasoningModelClaude_Sonnet:
		// Claude Sonnet with extended thinking: $3/1M input, $15/1M output
		thinkingCostPer1M = 3.0
		outputCostPer1M = 15.0

	case ReasoningModelDeepSeek_R1, ReasoningModelDeepSeek_Reasoner:
		// DeepSeek R1: $2.19/1M input, $8.19/1M output
		thinkingCostPer1M = 2.19
		outputCostPer1M = 8.19

	case ReasoningModelQwQ_32B:
		// QwQ-32B (often free or very cheap): $0.50/1M input, $1.50/1M output
		thinkingCostPer1M = 0.50
		outputCostPer1M = 1.50

	default:
		// Generic model - use average pricing
		thinkingCostPer1M = 5.0
		outputCostPer1M = 15.0
	}

	thinkingCost := float64(trace.ThinkingTokens) / 1_000_000.0 * thinkingCostPer1M
	outputCost := float64(trace.OutputTokens) / 1_000_000.0 * outputCostPer1M
	totalCost := thinkingCost + outputCost

	return thinkingCost, outputCost, totalCost
}

// GetReasoningBudgetRecommendation returns recommended token budgets for different use cases
func GetReasoningBudgetRecommendation(useCase string) int {
	useCase = strings.ToLower(strings.TrimSpace(useCase))

	switch useCase {
	case "simple", "quick", "basic":
		return 2000 // Simple queries
	case "standard", "normal", "medium", "":
		return 5000 // Standard queries
	case "complex", "detailed", "thorough":
		return 10000 // Complex queries
	case "research", "deep", "comprehensive":
		return 20000 // Research/deep analysis
	default:
		return 5000 // Default
	}
}

// OptimizeReasoningConfig optimizes reasoning configuration based on context
func OptimizeReasoningConfig(config *ReasoningConfig, ctx context.Context) *ReasoningConfig {
	if !config.Enabled {
		return config
	}

	optimized := *config

	// Adjust budget based on effort level
	if optimized.ThinkingBudget == 0 {
		effort, err := ValidateReasoningEffort(optimized.ReasoningEffort)
		if err == nil {
			switch effort {
			case ReasoningEffortLow:
				optimized.ThinkingBudget = 3000
			case ReasoningEffortMedium:
				optimized.ThinkingBudget = 7000
			case ReasoningEffortHigh:
				optimized.ThinkingBudget = 15000
			}
		}
	}

	return &optimized
}

// Helper functions

// estimateTokens provides a rough estimate of token count
// More accurate tokenization would require model-specific tokenizers
func estimateTokens(text string) int {
	// Rough estimation: ~4 characters per token on average
	// This is a simplification; actual tokenization varies by model
	return len(text) / 4
}

// truncateToTokenBudget truncates text to fit within a token budget
func truncateToTokenBudget(text string, budget int) string {
	estimatedTokens := estimateTokens(text)
	if estimatedTokens <= budget {
		return text
	}

	// Rough truncation based on character count
	charsPerToken := 4
	targetChars := budget * charsPerToken

	if targetChars >= len(text) {
		return text
	}

	// Truncate and add indicator
	truncated := text[:targetChars]
	return truncated + "... [truncated due to budget]"
}

// MergeReasoningConfigs merges two reasoning configs, with override taking precedence
func MergeReasoningConfigs(base *ReasoningConfig, override *ReasoningConfig) *ReasoningConfig {
	if override == nil {
		return base
	}
	if base == nil {
		return override
	}

	merged := *base

	if override.Enabled {
		merged.Enabled = override.Enabled
	}
	if override.ExtractThinking {
		merged.ExtractThinking = override.ExtractThinking
	}
	if override.HideFromUser {
		merged.HideFromUser = override.HideFromUser
	}
	if override.ThinkingTags != "" {
		merged.ThinkingTags = override.ThinkingTags
	}
	if override.ThinkingBudget > 0 {
		merged.ThinkingBudget = override.ThinkingBudget
	}
	if override.ReasoningEffort != "" {
		merged.ReasoningEffort = override.ReasoningEffort
	}
	if override.ModelType != "" {
		merged.ModelType = override.ModelType
	}

	return &merged
}
