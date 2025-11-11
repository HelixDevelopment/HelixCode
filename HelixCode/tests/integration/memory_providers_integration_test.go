package integration

import (
	"testing"

	"dev.helix.code/internal/memory/providers"
)

func TestMem0ProviderIntegration(t *testing.T) {
	t.Skip("Mem0Provider not yet implemented")
	// config := map[string]interface{}{
	// 	"base_url": "https://api.mem0.ai",
	// 	"api_key":  "test-key",
	// }

	// provider, err := providers.NewMem0Provider(config)
	// if err != nil {
	// 	t.Fatalf("Failed to create Mem0 provider: %v", err)
	// }

	// // Test basic properties
	// if provider.GetName() != "mem0" {
	// 	t.Errorf("Expected name 'mem0', got '%s'", provider.GetName())
	// }

	// if string(provider.GetType()) != "mem0" {
	// 	t.Errorf("Expected type 'mem0', got '%s'", string(provider.GetType()))
	// }

	// // Test capabilities
	// caps := provider.GetCapabilities()
	// if len(caps) == 0 {
	// 	t.Error("Expected some capabilities")
	// }
}

func TestZepProviderIntegration(t *testing.T) {
	config := map[string]interface{}{
		"base_url": "https://api.getzep.com",
		"api_key":  "test-key",
	}

	provider, err := providers.NewZepProvider(config)
	if err != nil {
		t.Fatalf("Failed to create Zep provider: %v", err)
	}

	if provider.GetName() != "Zep" {
		t.Errorf("Expected name 'Zep', got '%s'", provider.GetName())
	}

	if string(provider.GetType()) != "zep" {
		t.Errorf("Expected type 'zep', got '%s'", string(provider.GetType()))
	}
}

func TestMemontoProviderIntegration(t *testing.T) {
	t.Skip("MemontoProvider not yet implemented")
}

func TestBaseAIProviderIntegration(t *testing.T) {
	t.Skip("BaseAIProvider not yet implemented")
}
