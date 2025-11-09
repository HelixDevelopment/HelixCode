package main

import (
	"fmt"

	"dev.helix.code/internal/llm"
)

func main() {
	// Test model management system
	fmt.Println("🧪 Testing HelixCode Model Management System")
	
	// Create manager
	manager := llm.NewModelDownloadManager("/tmp/helix-test")
	
	// List available models
	fmt.Println("\n📋 Available Models:")
	models := manager.GetAvailableModels()
	if len(models) == 0 {
		fmt.Println("❌ No models found in registry")
	} else {
		for _, model := range models {
			fmt.Printf("✅ %s (%s) - %s\n", model.Name, model.ModelSize, model.Description)
		}
	}
	
	// Test search
	fmt.Println("\n🔍 Searching for 'instruct' models:")
	results := manager.SearchModels("instruct")
	for _, model := range results {
		fmt.Printf("📝 %s - %s\n", model.Name, model.Description)
	}
	
	// Test cross-provider registry
	fmt.Println("\n🔄 Testing Cross-Provider Registry:")
	registry := llm.NewCrossProviderRegistry("/tmp/helix-registry")
	
	providers := registry.ListProviders()
	fmt.Printf("📊 Found %d providers:\n", len(providers))
	for _, provider := range providers {
		fmt.Printf("🤖 %s - %s\n", provider.Name, provider.Description)
	}
	
	// Test integrated manager
	fmt.Println("\n🎯 Testing Integrated Manager:")
	integrated := llm.NewIntegratedModelManager("/tmp/helix-integrated")
	
	available, err := integrated.ListAvailableModels()
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Printf("📦 Found %d models available for integration\n", len(available))
	}
	
	fmt.Println("\n✅ Model Management System Test Completed!")
}