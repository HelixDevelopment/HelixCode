package llm

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test ReasoningConfig creation and defaults
func TestDefaultReasoningConfig(t *testing.T) {
	config := DefaultReasoningConfig()

	assert.NotNil(t, config)
	assert.False(t, config.Enabled)
	assert.True(t, config.ExtractThinking)
	assert.False(t, config.HideFromUser)
	assert.Equal(t, "thinking", config.ThinkingTags)
	assert.Equal(t, 0, config.ThinkingBudget)
	assert.Equal(t, string(ReasoningEffortMedium), config.ReasoningEffort)
	assert.Equal(t, ReasoningModelGeneric, config.ModelType)
}

func TestNewReasoningConfig_OpenAI_O1(t *testing.T) {
	config := NewReasoningConfig(ReasoningModelOpenAI_O1)

	assert.NotNil(t, config)
	assert.True(t, config.Enabled)
	assert.True(t, config.ExtractThinking)
	assert.Equal(t, "thinking", config.ThinkingTags)
	assert.Equal(t, 10000, config.ThinkingBudget)
	assert.Equal(t, ReasoningModelOpenAI_O1, config.ModelType)
}

func TestNewReasoningConfig_Claude_Opus(t *testing.T) {
	config := NewReasoningConfig(ReasoningModelClaude_Opus)

	assert.NotNil(t, config)
	assert.True(t, config.Enabled)
	assert.True(t, config.ExtractThinking)
	assert.Equal(t, "thinking", config.ThinkingTags)
	assert.Equal(t, 5000, config.ThinkingBudget)
	assert.Equal(t, ReasoningModelClaude_Opus, config.ModelType)
}

func TestNewReasoningConfig_DeepSeek_R1(t *testing.T) {
	config := NewReasoningConfig(ReasoningModelDeepSeek_R1)

	assert.NotNil(t, config)
	assert.True(t, config.Enabled)
	assert.Equal(t, "think", config.ThinkingTags)
	assert.Equal(t, 8000, config.ThinkingBudget)
	assert.Equal(t, ReasoningModelDeepSeek_R1, config.ModelType)
}

func TestNewReasoningConfig_QwQ_32B(t *testing.T) {
	config := NewReasoningConfig(ReasoningModelQwQ_32B)

	assert.NotNil(t, config)
	assert.True(t, config.Enabled)
	assert.Equal(t, "thinking", config.ThinkingTags)
	assert.Equal(t, 7000, config.ThinkingBudget)
	assert.Equal(t, ReasoningModelQwQ_32B, config.ModelType)
}

// Test ExtractReasoningTrace
func TestExtractReasoningTrace_DisabledConfig(t *testing.T) {
	config := DefaultReasoningConfig()
	content := "<thinking>This is a thought</thinking>This is output"

	trace := ExtractReasoningTrace(content, config)

	assert.NotNil(t, trace)
	assert.Empty(t, trace.ThinkingContent)
	assert.Equal(t, content, trace.OutputContent)
	assert.Equal(t, 0, trace.ThinkingTokens)
	assert.Greater(t, trace.OutputTokens, 0)
}

func TestExtractReasoningTrace_SingleThinkingBlock(t *testing.T) {
	config := NewReasoningConfig(ReasoningModelClaude_Opus)
	content := "<thinking>Let me analyze this step by step.</thinking>The final answer is 42."

	trace := ExtractReasoningTrace(content, config)

	assert.NotNil(t, trace)
	assert.Len(t, trace.ThinkingContent, 1)
	assert.Equal(t, "Let me analyze this step by step.", trace.ThinkingContent[0])
	assert.Equal(t, "The final answer is 42.", trace.OutputContent)
	assert.Greater(t, trace.ThinkingTokens, 0)
	assert.Greater(t, trace.OutputTokens, 0)
	assert.Equal(t, trace.ThinkingTokens+trace.OutputTokens, trace.TotalTokens)
}

func TestExtractReasoningTrace_MultipleThinkingBlocks(t *testing.T) {
	config := NewReasoningConfig(ReasoningModelClaude_Opus)
	content := `<thinking>First thought process</thinking>
Some output here.
<thinking>Second thought process</thinking>
More output.`

	trace := ExtractReasoningTrace(content, config)

	assert.NotNil(t, trace)
	assert.Len(t, trace.ThinkingContent, 2)
	assert.Equal(t, "First thought process", trace.ThinkingContent[0])
	assert.Equal(t, "Second thought process", trace.ThinkingContent[1])
	assert.Contains(t, trace.OutputContent, "Some output here")
	assert.Contains(t, trace.OutputContent, "More output")
}

func TestExtractReasoningTrace_NestedTags(t *testing.T) {
	config := NewReasoningConfig(ReasoningModelClaude_Opus)
	content := `<thinking>Outer thought <inner>nested</inner> content</thinking>Final output`

	trace := ExtractReasoningTrace(content, config)

	assert.NotNil(t, trace)
	assert.Len(t, trace.ThinkingContent, 1)
	assert.Contains(t, trace.ThinkingContent[0], "nested")
	assert.Equal(t, "Final output", trace.OutputContent)
}

func TestExtractReasoningTrace_CustomTags(t *testing.T) {
	config := DefaultReasoningConfig()
	config.Enabled = true
	config.ThinkingTags = "think"
	content := "<think>Custom tag thinking</think>Output text"

	trace := ExtractReasoningTrace(content, config)

	assert.NotNil(t, trace)
	assert.Len(t, trace.ThinkingContent, 1)
	assert.Equal(t, "Custom tag thinking", trace.ThinkingContent[0])
	assert.Equal(t, "Output text", trace.OutputContent)
}

func TestExtractReasoningTrace_MultipleTags(t *testing.T) {
	config := DefaultReasoningConfig()
	config.Enabled = true
	config.ThinkingTags = "thinking,think,internal"
	content := `<thinking>First type</thinking>
<think>Second type</think>
<internal>Third type</internal>
Final output`

	trace := ExtractReasoningTrace(content, config)

	assert.NotNil(t, trace)
	assert.Len(t, trace.ThinkingContent, 3)
	assert.Equal(t, "First type", trace.ThinkingContent[0])
	assert.Equal(t, "Second type", trace.ThinkingContent[1])
	assert.Equal(t, "Third type", trace.ThinkingContent[2])
	assert.Equal(t, "Final output", strings.TrimSpace(trace.OutputContent))
}

func TestExtractReasoningTrace_NoThinkingBlocks(t *testing.T) {
	config := NewReasoningConfig(ReasoningModelClaude_Opus)
	content := "Just regular output without thinking blocks"

	trace := ExtractReasoningTrace(content, config)

	assert.NotNil(t, trace)
	assert.Empty(t, trace.ThinkingContent)
	assert.Equal(t, content, trace.OutputContent)
	assert.Equal(t, 0, trace.ThinkingTokens)
	assert.Greater(t, trace.OutputTokens, 0)
}

// Test ApplyReasoningBudget
func TestApplyReasoningBudget_UnlimitedBudget(t *testing.T) {
	config := DefaultReasoningConfig()
	config.Enabled = true
	config.ThinkingBudget = 0 // unlimited

	trace := &ReasoningTrace{
		ThinkingContent: []string{"Very long thinking content that would exceed any reasonable budget"},
		OutputContent:   "Output",
		ThinkingTokens:  10000,
		OutputTokens:    100,
		TotalTokens:     10100,
	}

	result := ApplyReasoningBudget(trace, config)

	assert.Equal(t, trace, result)
	assert.Equal(t, 10000, result.ThinkingTokens)
}

func TestApplyReasoningBudget_WithinBudget(t *testing.T) {
	config := DefaultReasoningConfig()
	config.Enabled = true
	config.ThinkingBudget = 5000

	trace := &ReasoningTrace{
		ThinkingContent: []string{"Short thinking"},
		OutputContent:   "Output",
		ThinkingTokens:  100,
		OutputTokens:    50,
		TotalTokens:     150,
	}

	result := ApplyReasoningBudget(trace, config)

	assert.Equal(t, trace, result)
	assert.Equal(t, 100, result.ThinkingTokens)
}

func TestApplyReasoningBudget_ExceedsBudget(t *testing.T) {
	config := DefaultReasoningConfig()
	config.Enabled = true
	config.ThinkingBudget = 100 // very small budget

	// Create content that will exceed budget
	longContent := strings.Repeat("a", 1000)
	trace := &ReasoningTrace{
		ThinkingContent: []string{longContent, longContent},
		OutputContent:   "Output",
		ThinkingTokens:  500,
		OutputTokens:    50,
		TotalTokens:     550,
	}

	result := ApplyReasoningBudget(trace, config)

	assert.NotEqual(t, trace, result)
	assert.LessOrEqual(t, result.ThinkingTokens, config.ThinkingBudget)
	assert.Less(t, len(result.ThinkingContent), len(trace.ThinkingContent))
}

// Test ValidateReasoningEffort
func TestValidateReasoningEffort_ValidLevels(t *testing.T) {
	testCases := []struct {
		input    string
		expected ReasoningEffortLevel
	}{
		{"low", ReasoningEffortLow},
		{"Low", ReasoningEffortLow},
		{"LOW", ReasoningEffortLow},
		{"medium", ReasoningEffortMedium},
		{"Medium", ReasoningEffortMedium},
		{"med", ReasoningEffortMedium},
		{"", ReasoningEffortMedium}, // default
		{"high", ReasoningEffortHigh},
		{"High", ReasoningEffortHigh},
		{"HIGH", ReasoningEffortHigh},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result, err := ValidateReasoningEffort(tc.input)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestValidateReasoningEffort_InvalidLevels(t *testing.T) {
	invalidInputs := []string{"invalid", "super", "minimal", "maximum", "123"}

	for _, input := range invalidInputs {
		t.Run(input, func(t *testing.T) {
			_, err := ValidateReasoningEffort(input)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid reasoning effort level")
		})
	}
}

// Test FormatReasoningPrompt
func TestFormatReasoningPrompt_DisabledConfig(t *testing.T) {
	config := DefaultReasoningConfig()
	prompt := "What is 2+2?"

	result := FormatReasoningPrompt(prompt, config)

	assert.Equal(t, prompt, result)
}

func TestFormatReasoningPrompt_OpenAI_O1(t *testing.T) {
	config := NewReasoningConfig(ReasoningModelOpenAI_O1)
	prompt := "Solve this problem"

	result := FormatReasoningPrompt(prompt, config)

	// OpenAI o-series doesn't add model-specific instructions, but may add effort level guidance
	assert.Contains(t, result, prompt)
}

func TestFormatReasoningPrompt_Claude(t *testing.T) {
	config := NewReasoningConfig(ReasoningModelClaude_Opus)
	prompt := "Solve this problem"

	result := FormatReasoningPrompt(prompt, config)

	assert.Contains(t, result, "step-by-step")
	assert.Contains(t, result, prompt)
}

func TestFormatReasoningPrompt_DeepSeek(t *testing.T) {
	config := NewReasoningConfig(ReasoningModelDeepSeek_R1)
	prompt := "Solve this problem"

	result := FormatReasoningPrompt(prompt, config)

	assert.Contains(t, result, "<think>")
	assert.Contains(t, result, prompt)
}

func TestFormatReasoningPrompt_WithEffortLevels(t *testing.T) {
	config := NewReasoningConfig(ReasoningModelGeneric)

	testCases := []struct {
		effort   string
		expected string
	}{
		{"low", "brief"},
		{"medium", "thorough"},
		{"high", "comprehensive"},
	}

	for _, tc := range testCases {
		t.Run(tc.effort, func(t *testing.T) {
			config.ReasoningEffort = tc.effort
			result := FormatReasoningPrompt("Test", config)
			assert.Contains(t, result, tc.expected)
		})
	}
}

// Test IsReasoningModel
func TestIsReasoningModel_OpenAI(t *testing.T) {
	testCases := []struct {
		modelName    string
		isReasoning  bool
		expectedType ReasoningModelType
	}{
		{"gpt-4o", false, ReasoningModelGeneric},
		{"o1-preview", true, ReasoningModelOpenAI_O1},
		{"o1-mini", true, ReasoningModelOpenAI_O1},
		{"o1", true, ReasoningModelOpenAI_O1},
		{"o3", true, ReasoningModelOpenAI_O3},
		{"o4", true, ReasoningModelOpenAI_O4},
	}

	for _, tc := range testCases {
		t.Run(tc.modelName, func(t *testing.T) {
			isReasoning, modelType := IsReasoningModel(tc.modelName)
			assert.Equal(t, tc.isReasoning, isReasoning)
			if tc.isReasoning {
				assert.Equal(t, tc.expectedType, modelType)
			}
		})
	}
}

func TestIsReasoningModel_Claude(t *testing.T) {
	testCases := []struct {
		modelName    string
		isReasoning  bool
		expectedType ReasoningModelType
	}{
		{"claude-4-opus", true, ReasoningModelClaude_Opus},
		{"claude-3-opus", true, ReasoningModelClaude_Opus},
		{"claude-4-sonnet", true, ReasoningModelClaude_Sonnet},
		{"claude-3.7-sonnet", true, ReasoningModelClaude_Sonnet},
		{"claude-haiku", false, ReasoningModelGeneric},
	}

	for _, tc := range testCases {
		t.Run(tc.modelName, func(t *testing.T) {
			isReasoning, modelType := IsReasoningModel(tc.modelName)
			assert.Equal(t, tc.isReasoning, isReasoning)
			if tc.isReasoning {
				assert.Equal(t, tc.expectedType, modelType)
			}
		})
	}
}

func TestIsReasoningModel_DeepSeek(t *testing.T) {
	testCases := []struct {
		modelName    string
		isReasoning  bool
		expectedType ReasoningModelType
	}{
		{"deepseek-r1", true, ReasoningModelDeepSeek_R1},
		{"deepseek-reasoner", true, ReasoningModelDeepSeek_R1},
		{"deepseek-chat", false, ReasoningModelGeneric},
		{"deepseek-coder", false, ReasoningModelGeneric},
	}

	for _, tc := range testCases {
		t.Run(tc.modelName, func(t *testing.T) {
			isReasoning, modelType := IsReasoningModel(tc.modelName)
			assert.Equal(t, tc.isReasoning, isReasoning)
			if tc.isReasoning {
				assert.Equal(t, tc.expectedType, modelType)
			}
		})
	}
}

func TestIsReasoningModel_QwQ(t *testing.T) {
	isReasoning, modelType := IsReasoningModel("qwq-32b")
	assert.True(t, isReasoning)
	assert.Equal(t, ReasoningModelQwQ_32B, modelType)
}

// Test CalculateReasoningCost
func TestCalculateReasoningCost_OpenAI_O1(t *testing.T) {
	trace := &ReasoningTrace{
		ThinkingTokens: 1000,
		OutputTokens:   500,
		TotalTokens:    1500,
	}
	config := NewReasoningConfig(ReasoningModelOpenAI_O1)

	thinkingCost, outputCost, totalCost := CalculateReasoningCost(trace, config, ReasoningModelOpenAI_O1)

	assert.Greater(t, thinkingCost, 0.0)
	assert.Greater(t, outputCost, 0.0)
	assert.Equal(t, thinkingCost+outputCost, totalCost)

	// Verify o1 pricing: $15/1M input, $60/1M output
	expectedThinking := (1000.0 / 1_000_000.0) * 15.0
	expectedOutput := (500.0 / 1_000_000.0) * 60.0
	assert.InDelta(t, expectedThinking, thinkingCost, 0.0001)
	assert.InDelta(t, expectedOutput, outputCost, 0.0001)
}

func TestCalculateReasoningCost_Claude_Sonnet(t *testing.T) {
	trace := &ReasoningTrace{
		ThinkingTokens: 2000,
		OutputTokens:   1000,
		TotalTokens:    3000,
	}
	config := NewReasoningConfig(ReasoningModelClaude_Sonnet)

	thinkingCost, outputCost, totalCost := CalculateReasoningCost(trace, config, ReasoningModelClaude_Sonnet)

	assert.Greater(t, thinkingCost, 0.0)
	assert.Greater(t, outputCost, 0.0)
	assert.InDelta(t, thinkingCost+outputCost, totalCost, 0.0001)

	// Verify Claude Sonnet pricing: $3/1M input, $15/1M output
	expectedThinking := (2000.0 / 1_000_000.0) * 3.0
	expectedOutput := (1000.0 / 1_000_000.0) * 15.0
	assert.InDelta(t, expectedThinking, thinkingCost, 0.0001)
	assert.InDelta(t, expectedOutput, outputCost, 0.0001)
}

func TestCalculateReasoningCost_DeepSeek_R1(t *testing.T) {
	trace := &ReasoningTrace{
		ThinkingTokens: 3000,
		OutputTokens:   1500,
		TotalTokens:    4500,
	}
	config := NewReasoningConfig(ReasoningModelDeepSeek_R1)

	thinkingCost, outputCost, totalCost := CalculateReasoningCost(trace, config, ReasoningModelDeepSeek_R1)

	assert.Greater(t, thinkingCost, 0.0)
	assert.Greater(t, outputCost, 0.0)
	assert.Equal(t, thinkingCost+outputCost, totalCost)

	// DeepSeek is cheaper than others
	assert.Less(t, totalCost, 0.05) // Should be very cheap for this token count
}

// Test GetReasoningBudgetRecommendation
func TestGetReasoningBudgetRecommendation(t *testing.T) {
	testCases := []struct {
		useCase  string
		expected int
	}{
		{"simple", 2000},
		{"quick", 2000},
		{"basic", 2000},
		{"standard", 5000},
		{"normal", 5000},
		{"medium", 5000},
		{"", 5000}, // default
		{"complex", 10000},
		{"detailed", 10000},
		{"thorough", 10000},
		{"research", 20000},
		{"deep", 20000},
		{"comprehensive", 20000},
		{"unknown", 5000}, // default for unknown
	}

	for _, tc := range testCases {
		t.Run(tc.useCase, func(t *testing.T) {
			result := GetReasoningBudgetRecommendation(tc.useCase)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Test OptimizeReasoningConfig
func TestOptimizeReasoningConfig_DisabledConfig(t *testing.T) {
	config := DefaultReasoningConfig()
	ctx := context.Background()

	optimized := OptimizeReasoningConfig(config, ctx)

	assert.Equal(t, config, optimized)
}

func TestOptimizeReasoningConfig_SetsBudgetBasedOnEffort(t *testing.T) {
	testCases := []struct {
		effort         string
		expectedBudget int
	}{
		{"low", 3000},
		{"medium", 7000},
		{"high", 15000},
	}

	for _, tc := range testCases {
		t.Run(tc.effort, func(t *testing.T) {
			config := DefaultReasoningConfig()
			config.Enabled = true
			config.ReasoningEffort = tc.effort
			config.ThinkingBudget = 0 // Let optimizer set it
			ctx := context.Background()

			optimized := OptimizeReasoningConfig(config, ctx)

			assert.Equal(t, tc.expectedBudget, optimized.ThinkingBudget)
		})
	}
}

func TestOptimizeReasoningConfig_PreservesExistingBudget(t *testing.T) {
	config := DefaultReasoningConfig()
	config.Enabled = true
	config.ThinkingBudget = 12345
	ctx := context.Background()

	optimized := OptimizeReasoningConfig(config, ctx)

	assert.Equal(t, 12345, optimized.ThinkingBudget)
}

// Test MergeReasoningConfigs
func TestMergeReasoningConfigs_NilCases(t *testing.T) {
	config := DefaultReasoningConfig()

	// Override is nil
	result := MergeReasoningConfigs(config, nil)
	assert.Equal(t, config, result)

	// Base is nil
	result = MergeReasoningConfigs(nil, config)
	assert.Equal(t, config, result)
}

func TestMergeReasoningConfigs_OverrideValues(t *testing.T) {
	base := DefaultReasoningConfig()
	base.Enabled = false
	base.ThinkingBudget = 1000

	override := &ReasoningConfig{
		Enabled:        true,
		ThinkingTags:   "custom_tag",
		ThinkingBudget: 5000,
		ReasoningEffort: "high",
	}

	result := MergeReasoningConfigs(base, override)

	assert.True(t, result.Enabled)
	assert.Equal(t, "custom_tag", result.ThinkingTags)
	assert.Equal(t, 5000, result.ThinkingBudget)
	assert.Equal(t, "high", result.ReasoningEffort)
}

func TestMergeReasoningConfigs_PreservesBaseWhenOverrideEmpty(t *testing.T) {
	base := DefaultReasoningConfig()
	base.Enabled = true
	base.ThinkingTags = "thinking"
	base.ThinkingBudget = 3000

	override := &ReasoningConfig{} // Empty override

	result := MergeReasoningConfigs(base, override)

	assert.Equal(t, base.ThinkingTags, result.ThinkingTags)
	assert.Equal(t, base.ThinkingBudget, result.ThinkingBudget)
}

// Test helper functions
func TestEstimateTokens(t *testing.T) {
	testCases := []struct {
		text           string
		expectedTokens int
	}{
		{"", 0},
		{"test", 1},
		{"hello world", 2},
		{strings.Repeat("a", 100), 25},
		{strings.Repeat("test ", 100), 125},
	}

	for _, tc := range testCases {
		t.Run(tc.text[:min(len(tc.text), 20)], func(t *testing.T) {
			result := estimateTokens(tc.text)
			assert.Equal(t, tc.expectedTokens, result)
		})
	}
}

func TestTruncateToTokenBudget_WithinBudget(t *testing.T) {
	text := "Short text"
	budget := 100

	result := truncateToTokenBudget(text, budget)

	assert.Equal(t, text, result)
	assert.NotContains(t, result, "truncated")
}

func TestTruncateToTokenBudget_ExceedsBudget(t *testing.T) {
	text := strings.Repeat("a", 1000)
	budget := 10 // Only 10 tokens = 40 chars

	result := truncateToTokenBudget(text, budget)

	assert.NotEqual(t, text, result)
	assert.Contains(t, result, "truncated")
	assert.Less(t, len(result), len(text))
}

// Integration tests
func TestReasoningWorkflow_EndToEnd(t *testing.T) {
	// Simulate a complete reasoning workflow
	config := NewReasoningConfig(ReasoningModelClaude_Opus)

	// 1. Format prompt
	originalPrompt := "What is the capital of France?"
	formattedPrompt := FormatReasoningPrompt(originalPrompt, config)
	assert.Contains(t, formattedPrompt, originalPrompt)

	// 2. Simulate model response with thinking
	modelResponse := `<thinking>
Let me recall what I know about France. France is a country in Western Europe.
Its capital city is Paris, which is also its largest city.
</thinking>
The capital of France is Paris.`

	// 3. Extract reasoning trace
	trace := ExtractReasoningTrace(modelResponse, config)
	require.NotNil(t, trace)
	assert.Len(t, trace.ThinkingContent, 1)
	assert.Contains(t, trace.ThinkingContent[0], "France is a country")
	assert.Equal(t, "The capital of France is Paris.", trace.OutputContent)

	// 4. Apply budget
	budgetedTrace := ApplyReasoningBudget(trace, config)
	assert.LessOrEqual(t, budgetedTrace.ThinkingTokens, config.ThinkingBudget)

	// 5. Calculate cost
	thinkingCost, outputCost, totalCost := CalculateReasoningCost(budgetedTrace, config, config.ModelType)
	assert.Greater(t, totalCost, 0.0)
	assert.InDelta(t, thinkingCost+outputCost, totalCost, 0.0001)
}

func TestReasoningWorkflow_MultipleModels(t *testing.T) {
	modelTypes := []ReasoningModelType{
		ReasoningModelOpenAI_O1,
		ReasoningModelClaude_Opus,
		ReasoningModelDeepSeek_R1,
		ReasoningModelQwQ_32B,
	}

	for _, modelType := range modelTypes {
		t.Run(string(modelType), func(t *testing.T) {
			config := NewReasoningConfig(modelType)
			assert.True(t, config.Enabled)
			assert.Greater(t, config.ThinkingBudget, 0)

			// Test that budget recommendation works
			budget := GetReasoningBudgetRecommendation("complex")
			assert.Greater(t, budget, 0)

			// Test cost calculation
			trace := &ReasoningTrace{
				ThinkingTokens: 1000,
				OutputTokens:   500,
				TotalTokens:    1500,
			}
			_, _, totalCost := CalculateReasoningCost(trace, config, modelType)
			assert.Greater(t, totalCost, 0.0)
		})
	}
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
