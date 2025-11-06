package editor

import (
	"strings"
)

// ModelFormatPreferences maps model names to their preferred edit formats
var ModelFormatPreferences = map[string]EditFormat{
	// OpenAI models
	"gpt-4o":         EditFormatDiff,
	"gpt-4-turbo":    EditFormatDiff,
	"gpt-4":          EditFormatDiff,
	"gpt-3.5-turbo":  EditFormatSearchReplace,
	"o1-preview":     EditFormatWhole,
	"o1-mini":        EditFormatWhole,

	// Anthropic Claude models
	"claude-3-opus":    EditFormatSearchReplace,
	"claude-3-sonnet":  EditFormatSearchReplace,
	"claude-3-haiku":   EditFormatSearchReplace,
	"claude-3.5-sonnet": EditFormatSearchReplace,
	"claude-sonnet-4":  EditFormatSearchReplace,

	// Google Gemini models
	"gemini-pro":       EditFormatWhole,
	"gemini-ultra":     EditFormatDiff,
	"gemini-1.5-pro":   EditFormatDiff,
	"gemini-1.5-flash": EditFormatSearchReplace,

	// Meta Llama models
	"llama-2-7b":       EditFormatWhole,
	"llama-2-13b":      EditFormatWhole,
	"llama-2-70b":      EditFormatDiff,
	"llama-3-8b":       EditFormatWhole,
	"llama-3-70b":      EditFormatDiff,
	"llama-3.1-8b":     EditFormatSearchReplace,
	"llama-3.1-70b":    EditFormatDiff,
	"llama-3.1-405b":   EditFormatDiff,

	// Code-specific models
	"codellama-7b":     EditFormatWhole,
	"codellama-13b":    EditFormatSearchReplace,
	"codellama-34b":    EditFormatWhole,
	"codellama-70b":    EditFormatDiff,
	"deepseek-coder":   EditFormatDiff,
	"starcoder":        EditFormatWhole,
	"wizardcoder":      EditFormatSearchReplace,

	// Mistral models
	"mistral-7b":       EditFormatWhole,
	"mistral-8x7b":     EditFormatSearchReplace,
	"mistral-large":    EditFormatDiff,
	"mixtral-8x7b":     EditFormatSearchReplace,
	"mixtral-8x22b":    EditFormatDiff,

	// Qwen models
	"qwen-72b":         EditFormatDiff,
	"qwen-14b":         EditFormatSearchReplace,
	"qwen-7b":          EditFormatWhole,
	"qwen2-72b":        EditFormatDiff,
	"qwen2.5-72b":      EditFormatDiff,
	"qwen-coder":       EditFormatSearchReplace,

	// xAI Grok models
	"grok-1":           EditFormatDiff,
	"grok-2":           EditFormatDiff,

	// Microsoft/GitHub models
	"phi-2":            EditFormatWhole,
	"phi-3":            EditFormatSearchReplace,
	"copilot":          EditFormatSearchReplace,

	// Other notable models
	"command-r":        EditFormatSearchReplace,
	"command-r-plus":   EditFormatDiff,
	"solar-10.7b":      EditFormatSearchReplace,
	"yi-34b":           EditFormatDiff,
}

// ModelCapability represents the editing capabilities of a model
type ModelCapability struct {
	SupportsDiff          bool
	SupportsSearchReplace bool
	SupportsLines         bool
	SupportsWhole         bool
	PreferredFormat       EditFormat
	MaxContextSize        int
}

// ModelCapabilities maps model families to their capabilities
var ModelCapabilities = map[string]ModelCapability{
	"gpt-4": {
		SupportsDiff:          true,
		SupportsSearchReplace: true,
		SupportsLines:         true,
		SupportsWhole:         true,
		PreferredFormat:       EditFormatDiff,
		MaxContextSize:        128000,
	},
	"claude": {
		SupportsDiff:          true,
		SupportsSearchReplace: true,
		SupportsLines:         true,
		SupportsWhole:         true,
		PreferredFormat:       EditFormatSearchReplace,
		MaxContextSize:        200000,
	},
	"gemini": {
		SupportsDiff:          true,
		SupportsSearchReplace: true,
		SupportsLines:         true,
		SupportsWhole:         true,
		PreferredFormat:       EditFormatDiff,
		MaxContextSize:        1000000,
	},
	"llama": {
		SupportsDiff:          true,
		SupportsSearchReplace: true,
		SupportsLines:         false,
		SupportsWhole:         true,
		PreferredFormat:       EditFormatWhole,
		MaxContextSize:        8192,
	},
	"codellama": {
		SupportsDiff:          true,
		SupportsSearchReplace: true,
		SupportsLines:         true,
		SupportsWhole:         true,
		PreferredFormat:       EditFormatSearchReplace,
		MaxContextSize:        16384,
	},
}

// SelectFormatForModel returns the preferred edit format for a given model
func SelectFormatForModel(modelName string) EditFormat {
	// Normalize model name
	normalized := strings.ToLower(strings.TrimSpace(modelName))

	// Try exact match first
	if format, ok := ModelFormatPreferences[normalized]; ok {
		return format
	}

	// Try partial matching for model families
	for prefix, format := range ModelFormatPreferences {
		if strings.HasPrefix(normalized, prefix) {
			return format
		}
	}

	// Try capability-based matching
	for family, capability := range ModelCapabilities {
		if strings.Contains(normalized, family) {
			return capability.PreferredFormat
		}
	}

	// Default to search/replace as it's most universally understood
	return EditFormatSearchReplace
}

// GetModelCapability returns the capabilities for a given model
func GetModelCapability(modelName string) ModelCapability {
	normalized := strings.ToLower(strings.TrimSpace(modelName))

	// Try to match by family
	for family, capability := range ModelCapabilities {
		if strings.Contains(normalized, family) {
			return capability
		}
	}

	// Return default capability
	return ModelCapability{
		SupportsDiff:          true,
		SupportsSearchReplace: true,
		SupportsLines:         true,
		SupportsWhole:         true,
		PreferredFormat:       EditFormatSearchReplace,
		MaxContextSize:        8192,
	}
}

// SupportsFormat checks if a model supports a specific edit format
func SupportsFormat(modelName string, format EditFormat) bool {
	capability := GetModelCapability(modelName)

	switch format {
	case EditFormatDiff:
		return capability.SupportsDiff
	case EditFormatSearchReplace:
		return capability.SupportsSearchReplace
	case EditFormatLines:
		return capability.SupportsLines
	case EditFormatWhole:
		return capability.SupportsWhole
	default:
		return false
	}
}

// SelectBestFormat selects the best format considering both model and file size
func SelectBestFormat(modelName string, fileSize int64) EditFormat {
	capability := GetModelCapability(modelName)
	preferredFormat := SelectFormatForModel(modelName)

	// For very large files (>100KB), prefer whole file replacement if supported
	if fileSize > 100*1024 {
		if capability.SupportsWhole {
			return EditFormatWhole
		}
	}

	// For medium files (10-100KB), prefer diff or search/replace
	if fileSize > 10*1024 && fileSize <= 100*1024 {
		if preferredFormat == EditFormatDiff && capability.SupportsDiff {
			return EditFormatDiff
		}
		if capability.SupportsSearchReplace {
			return EditFormatSearchReplace
		}
	}

	// For small files (<10KB), use preferred format
	return preferredFormat
}

// FormatComplexity represents the complexity level of an edit format
type FormatComplexity int

const (
	ComplexitySimple  FormatComplexity = 1 // Whole file replacement
	ComplexityMedium  FormatComplexity = 2 // Search/replace or lines
	ComplexityComplex FormatComplexity = 3 // Diff format
)

// GetFormatComplexity returns the complexity level of a format
func GetFormatComplexity(format EditFormat) FormatComplexity {
	switch format {
	case EditFormatWhole:
		return ComplexitySimple
	case EditFormatSearchReplace, EditFormatLines:
		return ComplexityMedium
	case EditFormatDiff:
		return ComplexityComplex
	default:
		return ComplexityMedium
	}
}

// SelectFormatByComplexity selects format based on desired complexity and model capabilities
func SelectFormatByComplexity(modelName string, complexity FormatComplexity) EditFormat {
	capability := GetModelCapability(modelName)

	switch complexity {
	case ComplexitySimple:
		if capability.SupportsWhole {
			return EditFormatWhole
		}
		fallthrough
	case ComplexityMedium:
		if capability.SupportsSearchReplace {
			return EditFormatSearchReplace
		}
		if capability.SupportsLines {
			return EditFormatLines
		}
		fallthrough
	case ComplexityComplex:
		if capability.SupportsDiff {
			return EditFormatDiff
		}
	}

	// Fallback to preferred format
	return SelectFormatForModel(modelName)
}

// FormatRecommendation provides a recommendation for edit format with reasoning
type FormatRecommendation struct {
	Format     EditFormat
	Confidence float64 // 0.0 to 1.0
	Reasoning  string
}

// RecommendFormat provides an intelligent format recommendation
func RecommendFormat(modelName string, fileSize int64, editComplexity FormatComplexity) FormatRecommendation {
	capability := GetModelCapability(modelName)
	preferredFormat := SelectFormatForModel(modelName)

	// High confidence for small files with preferred format
	if fileSize < 10*1024 {
		return FormatRecommendation{
			Format:     preferredFormat,
			Confidence: 0.95,
			Reasoning:  "Small file size, using model's preferred format",
		}
	}

	// Medium confidence for large files needing whole replacement
	if fileSize > 100*1024 && capability.SupportsWhole {
		return FormatRecommendation{
			Format:     EditFormatWhole,
			Confidence: 0.85,
			Reasoning:  "Large file size, whole file replacement recommended",
		}
	}

	// Consider edit complexity
	if editComplexity == ComplexitySimple && capability.SupportsWhole {
		return FormatRecommendation{
			Format:     EditFormatWhole,
			Confidence: 0.90,
			Reasoning:  "Simple edit complexity, whole file replacement is efficient",
		}
	}

	if editComplexity == ComplexityComplex && capability.SupportsDiff {
		return FormatRecommendation{
			Format:     EditFormatDiff,
			Confidence: 0.90,
			Reasoning:  "Complex edit requirements, diff format provides precision",
		}
	}

	// Default to preferred with medium confidence
	return FormatRecommendation{
		Format:     preferredFormat,
		Confidence: 0.70,
		Reasoning:  "Using model's preferred format as default",
	}
}
