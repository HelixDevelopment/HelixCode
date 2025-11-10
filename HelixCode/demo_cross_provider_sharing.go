package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"dev.helix.code/internal/hardware"
	"dev.helix.code/internal/llm"
)

// CrossProviderModelSharingDemo demonstrates the complete cross-provider model sharing functionality
func demoCrossProviderSharing() {
	fmt.Println("🚀 Cross-Provider Model Sharing Demo")
	fmt.Println("=====================================")

	// Initialize context
	ctx := context.Background()

	// Setup base directory
	baseDir := "/tmp/helix-cross-provider-demo"

	// Initialize hardware detector
	fmt.Println("🔍 Detecting hardware capabilities...")
	hardwareDetector := hardware.NewDetector()
	_, err := hardwareDetector.Detect()
	if err != nil {
		log.Printf("Warning: Hardware detection failed: %v", err)
	}

	// Initialize Local LLM Manager
	fmt.Println("🔧 Initializing Local LLM Manager...")
	manager := llm.NewLocalLLMManager(baseDir)

	// Initialize cross-provider registry
	fmt.Println("📋 Initializing cross-provider registry...")
	registry := llm.NewCrossProviderRegistry(baseDir)

	// Demo 1: Initialize providers
	fmt.Println("\n🏗️  Demo 1: Initializing Providers")
	fmt.Println("-----------------------------------")
	if err := manager.Initialize(ctx); err != nil {
		log.Printf("Failed to initialize manager: %v", err)
		return
	}

	// Get provider status
	status := manager.GetProviderStatus(ctx)
	fmt.Printf("✅ Successfully initialized %d providers\n", len(status))
	for name, provider := range status {
		fmt.Printf("   • %s: %s (port: %d)\n", name, provider.Status, provider.DefaultPort)
	}

	// Demo 2: List available models
	fmt.Println("\n📚 Demo 2: Available Models")
	fmt.Println("-----------------------------")
	downloadManager := llm.NewModelDownloadManager(baseDir)
	models := downloadManager.GetAvailableModels()
	fmt.Printf("Found %d available models:\n", len(models))
	for i, model := range models {
		if i < 3 { // Show first 3 models
			fmt.Printf("   • %s (%s) - %s\n", model.ID, model.ModelSize, model.Description)
		}
	}

	// Demo 3: Check model compatibility across providers
	fmt.Println("\n🔗 Demo 3: Cross-Provider Compatibility")
	fmt.Println("------------------------------------")
	testModelID := "llama-3-8b-instruct"
	testFormat := llm.FormatGGUF

	providers := []string{"vllm", "llamacpp", "ollama", "localai", "fastchat"}
	fmt.Printf("Checking compatibility for %s (%s):\n", testModelID, testFormat)

	for _, provider := range providers {
		query := llm.ModelCompatibilityQuery{
			ModelID:        testModelID,
			SourceFormat:   testFormat,
			TargetProvider: provider,
		}

		result, err := registry.CheckCompatibility(query)
		if err != nil {
			fmt.Printf("   • %s: ERROR - %v\n", provider, err)
			continue
		}

		if result.IsCompatible {
			status := "✅ Compatible"
			if result.ConversionRequired {
				status += fmt.Sprintf(" (conversion required: %.0f min)", result.EstimatedTime)
			}
			fmt.Printf("   • %s: %s\n", provider, status)
		} else {
			fmt.Printf("   • %s: ❌ Not compatible\n", provider)
		}
		if len(result.Warnings) > 0 {
			fmt.Printf("     Warnings: %v\n", result.Warnings)
		}
		if len(result.Recommendations) > 0 {
			fmt.Printf("     Recommendations: %v\n", result.Recommendations)
		}
	}

	// Demo 4: Download model for all providers
	fmt.Println("\n📥 Demo 4: Download Model for All Providers")
	fmt.Println("--------------------------------------------")
	fmt.Printf("Downloading %s in GGUF format for all providers...\n", testModelID)

	// Start download (in demo, we'll simulate)
	fmt.Println("⏳ Simulating download...")
	time.Sleep(2 * time.Second)
	fmt.Printf("✅ Model downloaded to: %s/shared/%s/model.gguf\n", baseDir, testModelID)

	// Demo 5: Share model across providers
	fmt.Println("\n🔗 Demo 5: Share Model Across Providers")
	fmt.Println("--------------------------------------")
	modelPath := fmt.Sprintf("%s/shared/%s/model.gguf", baseDir, testModelID)
	fmt.Printf("Sharing %s with all compatible providers...\n", modelPath)

	err = manager.ShareModelWithProviders(ctx, modelPath, testModelID)
	if err != nil {
		log.Printf("Failed to share model: %v", err)
		return
	}
	fmt.Println("✅ Model shared successfully!")

	// Demo 6: List shared models
	fmt.Println("\n📋 Demo 6: List Shared Models")
	fmt.Println("----------------------------")
	shared, err := manager.GetSharedModels(ctx)
	if err != nil {
		log.Printf("Failed to get shared models: %v", err)
		return
	}

	fmt.Printf("Models shared across %d providers:\n", len(shared))
	for provider, models := range shared {
		if len(models) > 0 {
			fmt.Printf("📦 %s:\n", provider)
			for _, model := range models {
				fmt.Printf("   • %s\n", model)
			}
		}
	}

	// Demo 7: Optimize model for specific provider
	fmt.Println("\n⚡ Demo 7: Optimize Model for Provider")
	fmt.Println("------------------------------------")
	targetProvider := "vllm"
	fmt.Printf("Optimizing %s for %s provider...\n", testModelID, targetProvider)

	err = manager.OptimizeModelForProvider(ctx, modelPath, targetProvider)
	if err != nil {
		log.Printf("Failed to optimize model: %v", err)
		return
	}
	fmt.Printf("✅ Model optimized and shared for %s!\n", targetProvider)

	// Demo 8: Find optimal provider for model
	fmt.Println("\n🎯 Demo 8: Find Optimal Provider")
	fmt.Println("-------------------------------")
	constraints := map[string]interface{}{
		"gpu_required": true,
		"low_latency":  true,
	}

	optimalProvider, err := registry.FindOptimalProvider(testModelID, llm.FormatGGUF, constraints)
	if err != nil {
		log.Printf("Failed to find optimal provider: %v", err)
		return
	}

	fmt.Printf("🏆 Optimal provider for %s: %s\n", testModelID, optimalProvider.Name)
	fmt.Printf("📝 Description: %s\n", optimalProvider.Description)
	fmt.Printf("🌐 Endpoint: %s\n", optimalProvider.Endpoint)

	// Demo 9: Full synchronization
	fmt.Println("\n🔄 Demo 9: Full Model Synchronization")
	fmt.Println("-----------------------------------")
	fmt.Println("Synchronizing all models across all providers...")
	fmt.Println("(This would normally convert and share all models)")
	time.Sleep(2 * time.Second)
	fmt.Println("✅ Full synchronization completed!")

	// Demo 10: Show provider capabilities
	fmt.Println("\n📊 Demo 10: Provider Capabilities Summary")
	fmt.Println("-----------------------------------------")
	allProviders := registry.ListProviders()
	for _, provider := range allProviders {
		compat, _ := registry.GetCompatibleFormats(provider.Name)
		fmt.Printf("🏭 %s:\n", provider.Name)
		fmt.Printf("   • Type: %s\n", provider.Type)
		fmt.Printf("   • Formats: %v\n", compat)
		fmt.Printf("   • Tags: %v\n", provider.Tags)
		fmt.Println()
	}

	// Cleanup
	fmt.Println("🧹 Cleaning up demo resources...")
	manager.Cleanup(ctx)

	fmt.Println("\n🎉 Demo completed successfully!")
	fmt.Println("=====================================")
	fmt.Println("Key features demonstrated:")
	fmt.Println("✅ Cross-provider compatibility checking")
	fmt.Println("✅ Model download and format selection")
	fmt.Println("✅ Automatic model sharing across providers")
	fmt.Println("✅ Provider-specific model optimization")
	fmt.Println("✅ Optimal provider selection based on constraints")
	fmt.Println("✅ Full model synchronization")
	fmt.Println("✅ Provider capability management")

	fmt.Println("\nNext steps:")
	fmt.Println("1. Try: helix local-llm init")
	fmt.Println("2. Download: helix local-llm models download llama-3-8b-instruct")
	fmt.Println("3. Share: helix local-llm share ./model.gguf")
	fmt.Println("4. Optimize: helix local-llm optimize ./model.gguf --provider vllm")
	fmt.Println("5. Sync: helix local-llm sync")
}
