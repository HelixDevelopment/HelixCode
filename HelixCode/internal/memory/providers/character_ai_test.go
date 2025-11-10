package providers

import (
	"dev.helix.code/internal/memory"
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

	if provider.GetType() != memory.ProviderTypeCharacterAI {
		t.Errorf("Expected ProviderTypeCharacterAI, got %v", provider.GetType())
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
