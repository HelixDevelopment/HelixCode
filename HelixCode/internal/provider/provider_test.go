package provider

import (
	"testing"
)

// ========================================
// ProviderType Tests
// ========================================

func TestProviderType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		provider ProviderType
		expected string
	}{
		{"OpenAI", ProviderTypeOpenAI, "openai"},
		{"Anthropic", ProviderTypeAnthropic, "anthropic"},
		{"Gemini", ProviderTypeGemini, "gemini"},
		{"VertexAI", ProviderTypeVertexAI, "vertexai"},
		{"Azure", ProviderTypeAzure, "azure"},
		{"Bedrock", ProviderTypeBedrock, "bedrock"},
		{"Groq", ProviderTypeGroq, "groq"},
		{"Qwen", ProviderTypeQwen, "qwen"},
		{"Copilot", ProviderTypeCopilot, "copilot"},
		{"OpenRouter", ProviderTypeOpenRouter, "openrouter"},
		{"XAI", ProviderTypeXAI, "xai"},
		{"Ollama", ProviderTypeOllama, "ollama"},
		{"Local", ProviderTypeLocal, "local"},
		{"LlamaCpp", ProviderTypeLlamaCpp, "llamacpp"},
		{"VLLM", ProviderTypeVLLM, "vllm"},
		{"LocalAI", ProviderTypeLocalAI, "localai"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.provider) != tt.expected {
				t.Errorf("ProviderType %s = %v, want %v", tt.name, string(tt.provider), tt.expected)
			}
		})
	}
}

func TestProviderType_String(t *testing.T) {
	tests := []struct {
		name     string
		provider ProviderType
		expected string
	}{
		{"OpenAI", ProviderTypeOpenAI, "openai"},
		{"Anthropic", ProviderTypeAnthropic, "anthropic"},
		{"Gemini", ProviderTypeGemini, "gemini"},
		{"VertexAI", ProviderTypeVertexAI, "vertexai"},
		{"Azure", ProviderTypeAzure, "azure"},
		{"Bedrock", ProviderTypeBedrock, "bedrock"},
		{"Groq", ProviderTypeGroq, "groq"},
		{"Qwen", ProviderTypeQwen, "qwen"},
		{"Copilot", ProviderTypeCopilot, "copilot"},
		{"OpenRouter", ProviderTypeOpenRouter, "openrouter"},
		{"XAI", ProviderTypeXAI, "xai"},
		{"Ollama", ProviderTypeOllama, "ollama"},
		{"Local", ProviderTypeLocal, "local"},
		{"LlamaCpp", ProviderTypeLlamaCpp, "llamacpp"},
		{"VLLM", ProviderTypeVLLM, "vllm"},
		{"LocalAI", ProviderTypeLocalAI, "localai"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.provider.String(); got != tt.expected {
				t.Errorf("ProviderType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestProviderType_String_CustomValue(t *testing.T) {
	customProvider := ProviderType("custom-provider")
	expected := "custom-provider"

	if got := customProvider.String(); got != expected {
		t.Errorf("ProviderType.String() = %v, want %v", got, expected)
	}
}

func TestProviderType_String_EmptyValue(t *testing.T) {
	emptyProvider := ProviderType("")
	expected := ""

	if got := emptyProvider.String(); got != expected {
		t.Errorf("ProviderType.String() = %v, want %v (empty string)", got, expected)
	}
}

// ========================================
// Provider Type Grouping Tests
// ========================================

func TestProviderType_CloudProviders(t *testing.T) {
	cloudProviders := []ProviderType{
		ProviderTypeOpenAI,
		ProviderTypeAnthropic,
		ProviderTypeGemini,
		ProviderTypeVertexAI,
		ProviderTypeAzure,
		ProviderTypeBedrock,
		ProviderTypeGroq,
		ProviderTypeQwen,
		ProviderTypeCopilot,
		ProviderTypeOpenRouter,
		ProviderTypeXAI,
	}

	for _, provider := range cloudProviders {
		t.Run(provider.String(), func(t *testing.T) {
			if provider.String() == "" {
				t.Errorf("Cloud provider %v should have non-empty string representation", provider)
			}
		})
	}
}

func TestProviderType_LocalProviders(t *testing.T) {
	localProviders := []ProviderType{
		ProviderTypeOllama,
		ProviderTypeLocal,
		ProviderTypeLlamaCpp,
		ProviderTypeVLLM,
		ProviderTypeLocalAI,
	}

	for _, provider := range localProviders {
		t.Run(provider.String(), func(t *testing.T) {
			if provider.String() == "" {
				t.Errorf("Local provider %v should have non-empty string representation", provider)
			}
		})
	}
}

// ========================================
// Provider Type Uniqueness Tests
// ========================================

func TestProviderType_AllConstantsUnique(t *testing.T) {
	allProviders := []ProviderType{
		ProviderTypeOpenAI,
		ProviderTypeAnthropic,
		ProviderTypeGemini,
		ProviderTypeVertexAI,
		ProviderTypeAzure,
		ProviderTypeBedrock,
		ProviderTypeGroq,
		ProviderTypeQwen,
		ProviderTypeCopilot,
		ProviderTypeOpenRouter,
		ProviderTypeXAI,
		ProviderTypeOllama,
		ProviderTypeLocal,
		ProviderTypeLlamaCpp,
		ProviderTypeVLLM,
		ProviderTypeLocalAI,
	}

	seen := make(map[string]bool)
	for _, provider := range allProviders {
		str := provider.String()
		if seen[str] {
			t.Errorf("Duplicate provider type found: %s", str)
		}
		seen[str] = true
	}

	expectedCount := 16
	if len(seen) != expectedCount {
		t.Errorf("Expected %d unique provider types, got %d", expectedCount, len(seen))
	}
}

// ========================================
// Provider Type Count Tests
// ========================================

func TestProviderType_Count(t *testing.T) {
	allProviders := []ProviderType{
		ProviderTypeOpenAI,
		ProviderTypeAnthropic,
		ProviderTypeGemini,
		ProviderTypeVertexAI,
		ProviderTypeAzure,
		ProviderTypeBedrock,
		ProviderTypeGroq,
		ProviderTypeQwen,
		ProviderTypeCopilot,
		ProviderTypeOpenRouter,
		ProviderTypeXAI,
		ProviderTypeOllama,
		ProviderTypeLocal,
		ProviderTypeLlamaCpp,
		ProviderTypeVLLM,
		ProviderTypeLocalAI,
	}

	expectedCount := 16
	if len(allProviders) != expectedCount {
		t.Errorf("Expected %d provider types, got %d", expectedCount, len(allProviders))
	}
}

// ========================================
// Provider Type Comparison Tests
// ========================================

func TestProviderType_Equality(t *testing.T) {
	provider1 := ProviderTypeOpenAI
	provider2 := ProviderTypeOpenAI
	provider3 := ProviderTypeAnthropic

	if provider1 != provider2 {
		t.Error("Same provider types should be equal")
	}

	if provider1 == provider3 {
		t.Error("Different provider types should not be equal")
	}
}

func TestProviderType_StringComparison(t *testing.T) {
	provider := ProviderTypeOpenAI
	str := "openai"

	if provider.String() != str {
		t.Errorf("Provider string representation should match: got %s, want %s", provider.String(), str)
	}

	if string(provider) != str {
		t.Errorf("Provider cast to string should match: got %s, want %s", string(provider), str)
	}
}

// ========================================
// Edge Cases
// ========================================

func TestProviderType_CaseSensitivity(t *testing.T) {
	provider := ProviderType("OpenAI")
	expected := "OpenAI"

	if provider.String() != expected {
		t.Error("ProviderType should preserve case sensitivity")
	}

	if provider == ProviderTypeOpenAI {
		t.Error("Case-different provider types should not be equal")
	}
}

func TestProviderType_SpecialCharacters(t *testing.T) {
	specialProvider := ProviderType("provider-with-dash")
	expected := "provider-with-dash"

	if specialProvider.String() != expected {
		t.Errorf("ProviderType should preserve special characters: got %s, want %s", specialProvider.String(), expected)
	}
}

func TestProviderType_Conversion(t *testing.T) {
	// Test conversion from string to ProviderType
	str := "ollama"
	provider := ProviderType(str)

	if provider != ProviderTypeOllama {
		t.Errorf("Conversion from string failed: got %v, want %v", provider, ProviderTypeOllama)
	}

	// Test conversion back to string
	if provider.String() != str {
		t.Errorf("Conversion to string failed: got %s, want %s", provider.String(), str)
	}
}

// ========================================
// Provider Type Length Tests
// ========================================

func TestProviderType_Length(t *testing.T) {
	tests := []struct {
		name     string
		provider ProviderType
		minLen   int
		maxLen   int
	}{
		{"XAI", ProviderTypeXAI, 1, 20},
		{"Azure", ProviderTypeAzure, 1, 20},
		{"Groq", ProviderTypeGroq, 1, 20},
		{"Qwen", ProviderTypeQwen, 1, 20},
		{"Ollama", ProviderTypeOllama, 1, 20},
		{"OpenRouter", ProviderTypeOpenRouter, 1, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			length := len(tt.provider.String())
			if length < tt.minLen || length > tt.maxLen {
				t.Errorf("Provider %s length %d outside expected range [%d, %d]", tt.name, length, tt.minLen, tt.maxLen)
			}
		})
	}
}

// ========================================
// Provider Type Switch Statement Tests
// ========================================

func TestProviderType_SwitchStatement(t *testing.T) {
	providers := []ProviderType{
		ProviderTypeOpenAI,
		ProviderTypeAnthropic,
		ProviderTypeOllama,
		ProviderTypeLocal,
	}

	for _, provider := range providers {
		t.Run(provider.String(), func(t *testing.T) {
			// Verify provider can be used in switch statements
			var category string
			switch provider {
			case ProviderTypeOpenAI, ProviderTypeAnthropic:
				category = "cloud"
			case ProviderTypeOllama, ProviderTypeLocal:
				category = "local"
			default:
				category = "unknown"
			}

			if category == "" {
				t.Errorf("Provider %s should have a category", provider.String())
			}
		})
	}
}
