package providers

import (
	"testing"
)

func TestCharacterAIProvider_GetType(t *testing.T) {
	config := map[string]interface{}{
		"api_key": "test_key",
	}

	provider, err := NewCharacterAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if provider.GetType() != string(ProviderTypeCharacterAI) {
		t.Errorf("Expected %s, got %v", ProviderTypeCharacterAI, provider.GetType())
	}
}

func TestCharacterAIProvider_GetName(t *testing.T) {
	config := map[string]interface{}{
		"api_key": "test_key",
	}

	provider, err := NewCharacterAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if provider.GetName() != "character_ai" {
		t.Errorf("Expected 'character_ai', got %v", provider.GetName())
	}
}
