package editor

import (
	"testing"
)

func TestSelectFormatForModel(t *testing.T) {
	tests := []struct {
		name           string
		modelName      string
		expectedFormat EditFormat
	}{
		// OpenAI models
		{"GPT-4o", "gpt-4o", EditFormatDiff},
		{"GPT-4 Turbo", "gpt-4-turbo", EditFormatDiff},
		{"GPT-3.5", "gpt-3.5-turbo", EditFormatSearchReplace},

		// Claude models
		{"Claude Opus", "claude-3-opus", EditFormatSearchReplace},
		{"Claude Sonnet", "claude-3-sonnet", EditFormatSearchReplace},
		{"Claude 3.5 Sonnet", "claude-3.5-sonnet", EditFormatSearchReplace},

		// Gemini models
		{"Gemini Pro", "gemini-pro", EditFormatWhole},
		{"Gemini 1.5 Pro", "gemini-1.5-pro", EditFormatDiff},

		// Llama models
		{"Llama 2 70B", "llama-2-70b", EditFormatDiff},
		{"Llama 3 8B", "llama-3-8b", EditFormatWhole},

		// Code models
		{"CodeLlama 34B", "codellama-34b", EditFormatWhole},
		{"DeepSeek Coder", "deepseek-coder", EditFormatDiff},

		// Mistral models
		{"Mistral Large", "mistral-large", EditFormatDiff},
		{"Mixtral 8x7B", "mixtral-8x7b", EditFormatSearchReplace},

		// Unknown model - should default
		{"Unknown Model", "unknown-model-123", EditFormatSearchReplace},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format := SelectFormatForModel(tt.modelName)
			if format != tt.expectedFormat {
				t.Errorf("Format mismatch for %s: got %s, want %s",
					tt.modelName, format, tt.expectedFormat)
			}
		})
	}
}

func TestGetModelCapability(t *testing.T) {
	tests := []struct {
		name          string
		modelName     string
		expectDiff    bool
		expectSearch  bool
		expectLines   bool
		expectWhole   bool
	}{
		{
			name:         "GPT-4",
			modelName:    "gpt-4",
			expectDiff:   true,
			expectSearch: true,
			expectLines:  true,
			expectWhole:  true,
		},
		{
			name:         "Claude",
			modelName:    "claude-3-sonnet",
			expectDiff:   true,
			expectSearch: true,
			expectLines:  true,
			expectWhole:  true,
		},
		{
			name:         "Llama",
			modelName:    "llama-2-70b",
			expectDiff:   true,
			expectSearch: true,
			expectLines:  false,
			expectWhole:  true,
		},
		{
			name:         "Unknown model - default capability",
			modelName:    "unknown-xyz",
			expectDiff:   true,
			expectSearch: true,
			expectLines:  true,
			expectWhole:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cap := GetModelCapability(tt.modelName)

			if cap.SupportsDiff != tt.expectDiff {
				t.Errorf("SupportsDiff: got %v, want %v", cap.SupportsDiff, tt.expectDiff)
			}
			if cap.SupportsSearchReplace != tt.expectSearch {
				t.Errorf("SupportsSearchReplace: got %v, want %v", cap.SupportsSearchReplace, tt.expectSearch)
			}
			if cap.SupportsLines != tt.expectLines {
				t.Errorf("SupportsLines: got %v, want %v", cap.SupportsLines, tt.expectLines)
			}
			if cap.SupportsWhole != tt.expectWhole {
				t.Errorf("SupportsWhole: got %v, want %v", cap.SupportsWhole, tt.expectWhole)
			}
		})
	}
}

func TestSupportsFormat(t *testing.T) {
	tests := []struct {
		name        string
		modelName   string
		format      EditFormat
		expectTrue  bool
	}{
		{"GPT-4 supports diff", "gpt-4", EditFormatDiff, true},
		{"GPT-4 supports search/replace", "gpt-4", EditFormatSearchReplace, true},
		{"Llama doesn't support lines", "llama-2-70b", EditFormatLines, false},
		{"Claude supports all", "claude-3-sonnet", EditFormatDiff, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SupportsFormat(tt.modelName, tt.format)
			if result != tt.expectTrue {
				t.Errorf("SupportsFormat: got %v, want %v", result, tt.expectTrue)
			}
		})
	}
}

func TestSelectBestFormat(t *testing.T) {
	tests := []struct {
		name           string
		modelName      string
		fileSize       int64
		expectedFormat EditFormat
	}{
		{
			name:           "Small file - use preferred",
			modelName:      "gpt-4o",
			fileSize:       5 * 1024, // 5KB
			expectedFormat: EditFormatDiff,
		},
		{
			name:           "Large file - use whole",
			modelName:      "gpt-4o",
			fileSize:       150 * 1024, // 150KB
			expectedFormat: EditFormatWhole,
		},
		{
			name:           "Medium file - use diff",
			modelName:      "gpt-4o",
			fileSize:       50 * 1024, // 50KB
			expectedFormat: EditFormatDiff,
		},
		{
			name:           "Medium file - search/replace preferred",
			modelName:      "claude-3-sonnet",
			fileSize:       50 * 1024, // 50KB
			expectedFormat: EditFormatSearchReplace,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format := SelectBestFormat(tt.modelName, tt.fileSize)
			if format != tt.expectedFormat {
				t.Errorf("Format mismatch: got %s, want %s", format, tt.expectedFormat)
			}
		})
	}
}

func TestGetFormatComplexity(t *testing.T) {
	tests := []struct {
		format             EditFormat
		expectedComplexity FormatComplexity
	}{
		{EditFormatWhole, ComplexitySimple},
		{EditFormatSearchReplace, ComplexityMedium},
		{EditFormatLines, ComplexityMedium},
		{EditFormatDiff, ComplexityComplex},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			complexity := GetFormatComplexity(tt.format)
			if complexity != tt.expectedComplexity {
				t.Errorf("Complexity mismatch: got %d, want %d", complexity, tt.expectedComplexity)
			}
		})
	}
}

func TestSelectFormatByComplexity(t *testing.T) {
	tests := []struct {
		name           string
		modelName      string
		complexity     FormatComplexity
		expectedFormat EditFormat
	}{
		{
			name:           "Simple complexity",
			modelName:      "gpt-4",
			complexity:     ComplexitySimple,
			expectedFormat: EditFormatWhole,
		},
		{
			name:           "Medium complexity",
			modelName:      "gpt-4",
			complexity:     ComplexityMedium,
			expectedFormat: EditFormatSearchReplace,
		},
		{
			name:           "Complex complexity",
			modelName:      "gpt-4",
			complexity:     ComplexityComplex,
			expectedFormat: EditFormatDiff,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format := SelectFormatByComplexity(tt.modelName, tt.complexity)
			if format != tt.expectedFormat {
				t.Errorf("Format mismatch: got %s, want %s", format, tt.expectedFormat)
			}
		})
	}
}

func TestRecommendFormat(t *testing.T) {
	tests := []struct {
		name             string
		modelName        string
		fileSize         int64
		editComplexity   FormatComplexity
		expectedFormat   EditFormat
		minConfidence    float64
	}{
		{
			name:           "Small file, preferred format",
			modelName:      "gpt-4o",
			fileSize:       5 * 1024,
			editComplexity: ComplexityMedium,
			expectedFormat: EditFormatDiff,
			minConfidence:  0.90,
		},
		{
			name:           "Large file, whole replacement",
			modelName:      "gpt-4o",
			fileSize:       150 * 1024,
			editComplexity: ComplexityMedium,
			expectedFormat: EditFormatWhole,
			minConfidence:  0.80,
		},
		{
			name:           "Simple edit complexity",
			modelName:      "claude-3-sonnet",
			fileSize:       20 * 1024,
			editComplexity: ComplexitySimple,
			expectedFormat: EditFormatWhole,
			minConfidence:  0.85,
		},
		{
			name:           "Complex edit complexity",
			modelName:      "gpt-4",
			fileSize:       30 * 1024,
			editComplexity: ComplexityComplex,
			expectedFormat: EditFormatDiff,
			minConfidence:  0.85,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recommendation := RecommendFormat(tt.modelName, tt.fileSize, tt.editComplexity)

			if recommendation.Format != tt.expectedFormat {
				t.Errorf("Format mismatch: got %s, want %s", recommendation.Format, tt.expectedFormat)
			}

			if recommendation.Confidence < tt.minConfidence {
				t.Errorf("Confidence too low: got %.2f, want >= %.2f",
					recommendation.Confidence, tt.minConfidence)
			}

			if recommendation.Reasoning == "" {
				t.Error("Expected reasoning to be provided")
			}
		})
	}
}

func TestModelFormatPreferencesCoverage(t *testing.T) {
	// Ensure we have preferences for major model families
	modelFamilies := []string{
		"gpt-4",
		"claude",
		"gemini",
		"llama",
		"codellama",
		"mistral",
		"qwen",
	}

	for _, family := range modelFamilies {
		t.Run(family, func(t *testing.T) {
			format := SelectFormatForModel(family)
			if format == "" {
				t.Errorf("No format selected for %s", family)
			}

			// Verify the format is valid
			switch format {
			case EditFormatDiff, EditFormatWhole, EditFormatSearchReplace, EditFormatLines:
				// Valid format
			default:
				t.Errorf("Invalid format %s for %s", format, family)
			}
		})
	}
}

func TestModelCapabilitiesConsistency(t *testing.T) {
	// Test that capabilities are consistent with preferences
	for modelName, preferredFormat := range ModelFormatPreferences {
		t.Run(modelName, func(t *testing.T) {
			if !SupportsFormat(modelName, preferredFormat) {
				t.Errorf("Model %s has preferred format %s but doesn't support it",
					modelName, preferredFormat)
			}
		})
	}
}

func TestSelectFormatCaseInsensitivity(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
	}{
		{"Uppercase", "GPT-4O"},
		{"Lowercase", "gpt-4o"},
		{"Mixed case", "GpT-4o"},
		{"With spaces", " gpt-4o "},
	}

	expectedFormat := EditFormatDiff

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format := SelectFormatForModel(tt.modelName)
			if format != expectedFormat {
				t.Errorf("Case sensitivity issue: got %s, want %s", format, expectedFormat)
			}
		})
	}
}

func TestSelectBestFormatEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		fileSize  int64
	}{
		{"Zero size", "gpt-4", 0},
		{"Exactly 10KB", "gpt-4", 10 * 1024},
		{"Exactly 100KB", "gpt-4", 100 * 1024},
		{"Very large", "gpt-4", 10 * 1024 * 1024}, // 10MB
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format := SelectBestFormat(tt.modelName, tt.fileSize)
			if format == "" {
				t.Error("Expected valid format, got empty string")
			}
		})
	}
}

func TestRecommendFormatReasoningQuality(t *testing.T) {
	recommendation := RecommendFormat("gpt-4", 50*1024, ComplexityMedium)

	if recommendation.Format == "" {
		t.Error("Expected non-empty format")
	}

	if recommendation.Confidence < 0.0 || recommendation.Confidence > 1.0 {
		t.Errorf("Confidence out of range: %.2f", recommendation.Confidence)
	}

	if len(recommendation.Reasoning) < 10 {
		t.Error("Reasoning too short, expected meaningful explanation")
	}
}
