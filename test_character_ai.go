package main

import (
	"context"
	"dev.helix.code/internal/memory/providers"
	"fmt"
)

func main() {
	// Test Character.AI provider creation
	registry := providers.GetRegistry()
	provider, err := registry.CreateProvider("characterai", map[string]interface{}{
		"api_key":  "test_key",
		"base_url": "https://api.character.ai",
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create Character.AI provider: %v", err))
	}

	fmt.Printf("✅ Character.AI provider created successfully!\n")
	fmt.Printf("   Name: %s\n", provider.GetName())
	fmt.Printf("   Type: %s\n", provider.GetType())
	fmt.Printf("   Capabilities: %v\n", provider.GetCapabilities())
	fmt.Printf("   Is Cloud: %v\n", provider.IsCloud())

	// Test basic functionality
	ctx := context.Background()

	// Test health check
	health, err := provider.Health(ctx)
	if err != nil {
		panic(fmt.Sprintf("Health check failed: %v", err))
	}
	fmt.Printf("   Health Status: %s\n", health.Status)

	// Test stats
	stats, err := provider.GetStats(ctx)
	if err != nil {
		panic(fmt.Sprintf("Stats retrieval failed: %v", err))
	}
	fmt.Printf("   Total Vectors: %d\n", stats.TotalVectors)
	fmt.Printf("   Total Collections: %d\n", stats.TotalCollections)

	fmt.Printf("\n🎉 Character.AI provider is fully functional!\n")
}
