package unit

import (
	"testing"

	"dev.helix.code/internal/memory"
	"dev.helix.code/internal/memory/providers"
)

func TestMem0Provider(t *testing.T) {
	config := map[string]interface{}{
		"base_url": "https://api.mem0.ai",
		"api_key":  "test-key",
	}

	provider, err := providers.NewMem0Provider(config)
	if err != nil {
		t.Fatalf("Failed to create Mem0 provider: %v", err)
	}

	if provider == nil {
		t.Fatal("Provider is nil")
	}

	if provider.GetName() != "mem0" {
		t.Errorf("Expected name 'mem0', got '%s'", provider.GetName())
	}

	if provider.GetType() != memory.ProviderTypeMem0 {
		t.Errorf("Expected type %s, got %s", memory.ProviderTypeMem0, provider.GetType())
	}

	if !provider.IsCloud() {
		t.Error("Expected Mem0 to be cloud provider")
	}
}

func TestZepProvider(t *testing.T) {
	config := map[string]interface{}{
		"base_url": "https://api.getzep.com",
		"api_key":  "test-key",
	}

	provider, err := providers.NewZepProvider(config)
	if err != nil {
		t.Fatalf("Failed to create Zep provider: %v", err)
	}

	if provider == nil {
		t.Fatal("Provider is nil")
	}

	if provider.GetName() != "zep" {
		t.Errorf("Expected name 'zep', got '%s'", provider.GetName())
	}

	if provider.GetType() != memory.ProviderTypeZep {
		t.Errorf("Expected type %s, got %s", memory.ProviderTypeZep, provider.GetType())
	}

	if !provider.IsCloud() {
		t.Error("Expected Zep to be cloud provider")
	}
}

func TestMemontoProvider(t *testing.T) {
	config := map[string]interface{}{
		"base_url": "https://api.memonto.ai",
		"api_key":  "test-key",
	}

	provider, err := providers.NewMemontoProvider(config)
	if err != nil {
		t.Fatalf("Failed to create Memonto provider: %v", err)
	}

	if provider == nil {
		t.Fatal("Provider is nil")
	}

	if provider.GetName() != "memonto" {
		t.Errorf("Expected name 'memonto', got '%s'", provider.GetName())
	}

	if provider.GetType() != memory.ProviderTypeMemonto {
		t.Errorf("Expected type %s, got %s", memory.ProviderTypeMemonto, provider.GetType())
	}

	if !provider.IsCloud() {
		t.Error("Expected Memonto to be cloud provider")
	}
}

func TestBaseAIProvider(t *testing.T) {
	config := map[string]interface{}{
		"base_url": "https://api.baseai.com",
		"api_key":  "test-key",
	}

	provider, err := providers.NewBaseAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create BaseAI provider: %v", err)
	}

	if provider == nil {
		t.Fatal("Provider is nil")
	}

	if provider.GetName() != "baseai" {
		t.Errorf("Expected name 'baseai', got '%s'", provider.GetName())
	}

	if provider.GetType() != memory.ProviderTypeBaseAI {
		t.Errorf("Expected type %s, got %s", memory.ProviderTypeBaseAI, provider.GetType())
	}

	if !provider.IsCloud() {
		t.Error("Expected BaseAI to be cloud provider")
	}
}
