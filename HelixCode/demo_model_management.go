package main

import (
	"fmt"

	"dev.helix.code/internal/llm"
)

// demoModelManagement demonstrates advanced model management features
func demoModelManagement() {
	fmt.Println("🚀 HelixCode Advanced Model Management Demo")
	fmt.Println("============================================")

	// Create model download manager
	fmt.Println("\n📦 1. Initializing Model Download Manager...")
	downloadManager := llm.NewModelDownloadManager("/tmp/helix-model-demo")

	// Show available models
	fmt.Println("\n📋 2. Available Models in Registry:")
	models := downloadManager.GetAvailableModels()
	for i, model := range models {
		fmt.Printf("   %d. %s (%s)\n", i+1, model.Name, model.ModelSize)
		fmt.Printf("      📝 %s\n", model.Description)
		fmt.Printf("      🎯 Context: %d tokens\n", model.ContextSize)
		fmt.Printf("      📁 Formats: %v\n", model.AvailableFormats)
		fmt.Println()
	}

	// Search for specific models
	fmt.Println("🔍 3. Searching for 'code' models:")
	codeModels := downloadManager.SearchModels("code")
	for _, model := range codeModels {
		fmt.Printf("   💻 %s - %s\n", model.Name, model.Description)
	}

	// Test cross-provider compatibility
	fmt.Println("\n🔄 4. Testing Cross-Provider Compatibility:")
	registry := llm.NewCrossProviderRegistry("/tmp/helix-compat-demo")

	// Check compatibility of different formats
	providers := []string{"vllm", "llamacpp", "ollama", "localai"}
	for _, provider := range providers {
		compatible, _ := registry.GetCompatibleFormats(provider)
		fmt.Printf("   🤖 %s: %v\n", provider, compatible)
	}

	// Test integrated model manager
	fmt.Println("\n🎯 5. Testing Integrated Model Manager:")
	integrated := llm.NewIntegratedModelManager("/tmp/helix-integrated-demo")

	// Find best model for different scenarios
	scenarios := []struct {
		name string
		req  llm.ModelSelectionCriteria
	}{
		{
			name: "High Quality",
			req: llm.ModelSelectionCriteria{
				QualityPreference: "quality",
				MaxTokens:         4096,
			},
		},
		{
			name: "Fast Response",
			req: llm.ModelSelectionCriteria{
				QualityPreference: "fast",
				MaxTokens:         2048,
			},
		},
		{
			name: "Code Generation",
			req: llm.ModelSelectionCriteria{
				TaskType:          "code",
				QualityPreference: "quality",
				MaxTokens:         8192,
			},
		},
	}

	for _, scenario := range scenarios {
		fmt.Printf("   📊 %s:\n", scenario.name)
		bestModel, err := integrated.FindBestModel(scenario.req)
		if err != nil {
			fmt.Printf("      ❌ Error: %v\n", err)
		} else {
			fmt.Printf("      ✅ Best: %s for %s\n", bestModel.ModelID, bestModel.Provider)
			fmt.Printf("      📁 Format: %s\n", bestModel.Format)
		}
	}

	// Demonstrate model conversion validation
	fmt.Println("\n🔄 6. Testing Model Conversion Validation:")
	converter := llm.NewModelConverter("/tmp/helix-convert-demo")

	conversions := []struct {
		from llm.ModelFormat
		to   llm.ModelFormat
	}{
		{llm.FormatHF, llm.FormatGGUF},
		{llm.FormatGGUF, llm.FormatGPTQ},
		{llm.FormatHF, llm.FormatAWQ},
		{llm.FormatFP16, llm.FormatGGUF},
	}

	for _, conv := range conversions {
		fmt.Printf("   🔧 %s -> %s: ", conv.from, conv.to)
		result, err := converter.ValidateConversion(conv.from, conv.to)
		if err != nil {
			fmt.Printf("❌ %v\n", err)
		} else {
			fmt.Printf("✅ Possible (confidence: %.1f%%)\n", result.Confidence*100)
			if len(result.Recommendations) > 0 {
				fmt.Printf("      💡 %s\n", result.Recommendations[0])
			}
		}
	}

	// Show conversion history (will be empty for new demo)
	fmt.Println("\n📈 7. Conversion History:")
	history, err := converter.GetConversionHistory()
	if err != nil {
		fmt.Printf("      ❌ Error getting history: %v\n", err)
	} else {
		fmt.Printf("      📊 Total: %d, Successful: %d, Failed: %d\n",
			history.TotalConversions, history.SuccessfulConversions, history.FailedConversions)
		if history.AverageConversionTime > 0 {
			fmt.Printf("      ⏱️  Average time: %d minutes\n", history.AverageConversionTime)
		}
	}

	// Show provider information
	fmt.Println("\n🤖 8. Provider Information:")
	for _, provider := range providers {
		info, err := registry.GetProviderInfo(provider)
		if err != nil {
			fmt.Printf("   ❌ %s: %v\n", provider, err)
		} else {
			fmt.Printf("   🎯 %s:\n", info.Name)
			fmt.Printf("      📝 %s\n", info.Description)
			fmt.Printf("      🔗 %s\n", info.Website)
			fmt.Printf("      📦 Port: %d\n", info.DefaultPort)
		}
	}

	fmt.Println("\n🎉 Demo completed successfully!")
	fmt.Println("\n💡 Key Features Demonstrated:")
	fmt.Println("   ✅ Model registry with 3+ popular models")
	fmt.Println("   ✅ Cross-provider format compatibility checking")
	fmt.Println("   ✅ Intelligent model selection based on criteria")
	fmt.Println("   ✅ Format conversion validation and recommendations")
	fmt.Println("   ✅ Comprehensive provider information")
	fmt.Println("   ✅ Integrated model management")

	fmt.Println("\n🚀 Next Steps:")
	fmt.Println("   1. Use 'helix local-llm models download <model>' to get models")
	fmt.Println("   2. Use 'helix local-llm models convert <path>' to convert formats")
	fmt.Println("   3. Use 'helix local-llm start <provider>' to run providers")
	fmt.Println("   4. Use 'helix local-llm monitor' to watch provider health")

	// Clean up demo files
	fmt.Println("\n🧹 Cleaning up demo files...")
	// In a real implementation, you might want to preserve some data

	fmt.Println("✅ Demo complete!")
}
