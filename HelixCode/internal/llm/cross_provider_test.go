package llm

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"dev.helix.code/internal/hardware"
)

// TestCrossProviderCompatibility tests compatibility checking across providers
func TestCrossProviderCompatibility(t *testing.T) {
	registry := NewCrossProviderRegistry(t.TempDir())
	
	// Test GGUF format (most compatible)
	query := ModelCompatibilityQuery{
		ModelID:       "llama-3-8b-instruct",
		SourceFormat:  FormatGGUF,
		TargetProvider: "vllm",
	}
	
	result, err := registry.CheckCompatibility(query)
	if err != nil {
		t.Fatalf("Failed to check compatibility: %v", err)
	}
	
	if !result.IsCompatible {
		t.Error("GGUF should be compatible with VLLM")
	}
	
	if result.ConversionRequired {
		t.Error("GGUF should not require conversion for VLLM")
	}
	
	// Test HF format requiring conversion
	query.SourceFormat = FormatHF
	result, err = registry.CheckCompatibility(query)
	if err != nil {
		t.Fatalf("Failed to check compatibility: %v", err)
	}
	
	if !result.IsCompatible {
		t.Error("HF should be compatible with VLLM (with conversion)")
	}
	
	if !result.ConversionRequired {
		t.Error("HF should require conversion for VLLM")
	}
}

// TestCrossProviderRegistry tests the registry functionality
func TestCrossProviderRegistry(t *testing.T) {
	baseDir := t.TempDir()
	registry := NewCrossProviderRegistry(baseDir)
	
	// Test provider listing
	providers := registry.ListProviders()
	if len(providers) == 0 {
		t.Error("Registry should have default providers")
	}
	
	// Test finding optimal provider
	provider, err := registry.FindOptimalProvider("test-model", FormatGGUF, map[string]interface{}{
		"gpu_required": false,
		"cpu_only":    true,
	})
	if err != nil {
		t.Fatalf("Failed to find optimal provider: %v", err)
	}
	
	if provider == nil {
		t.Error("Should find optimal provider for GGUF format")
	}
	
	// Test model registration
	model := &DownloadedModel{
		ModelID:  "test-model",
		Provider: "test",
		Format:    FormatGGUF,
		Path:      "/test/path",
		Size:      1000000,
		Checksum:  "abc123",
	}
	
	err = registry.RegisterDownloadedModel(model)
	if err != nil {
		t.Fatalf("Failed to register model: %v", err)
	}
	
	// Test retrieving downloaded models
	models := registry.GetDownloadedModels()
	if len(models) == 0 {
		t.Error("Should have at least one downloaded model")
	}
	
	found := false
	for _, m := range models {
		if m.ModelID == "test-model" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Should find registered test model")
	}
}

// TestLocalLLMManagerCrossProvider tests the manager's cross-provider functionality
func TestLocalLLMManagerCrossProvider(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewLocalLLMManager(baseDir)
	ctx := context.Background()
	
	// Test initialization
	err := manager.Initialize(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}
	
	// Test format detection
	testCases := []struct {
		path     string
		expected ModelFormat
	}{
		{"/test/model.gguf", FormatGGUF},
		{"/test/model.safetensors", FormatHF},
		{"/test/model.bin", FormatGPTQ},
	}
	
	for _, tc := range testCases {
		format := manager.detectModelFormat(tc.path)
		if format != tc.expected {
			t.Errorf("Expected format %v for %s, got %v", tc.expected, tc.path, format)
		}
	}
	
	// Test format compatibility checking
	if !manager.isFormatCompatibleWithProvider(FormatGGUF, "vllm") {
		t.Error("GGUF should be compatible with VLLM")
	}
	
	if !manager.isFormatCompatibleWithProvider(FormatGGUF, "llamacpp") {
		t.Error("GGUF should be compatible with Llama.cpp")
	}
	
	// Test finding most compatible format
	bestFormat := manager.findMostCompatibleFormat(FormatHF)
	if bestFormat != FormatGGUF {
		t.Errorf("Expected GGUF as most compatible format, got %v", bestFormat)
	}
	
	// Test provider-specific optimization
	if manager.getOptimalFormatForProvider("vllm") != FormatGGUF {
		t.Error("VLLM optimal format should be GGUF")
	}
	
	if manager.getOptimalFormatForProvider("llamacpp") != FormatGGUF {
		t.Error("Llama.cpp optimal format should be GGUF")
	}
	
	if manager.getOptimizationTarget("vllm") != "gpu" {
		t.Error("VLLM optimization target should be GPU")
	}
	
	if manager.getOptimizationTarget("llamacpp") != "cpu" {
		t.Error("Llama.cpp optimization target should be CPU")
	}
}

// TestModelSharing tests model sharing across providers
func TestModelSharing(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewLocalLLMManager(baseDir)
	ctx := context.Background()
	
	// Initialize manager
	err := manager.Initialize(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}
	
	// Create a test model file
	testModelPath := filepath.Join(baseDir, "test-model.gguf")
	testData := []byte("test model data")
	err = os.WriteFile(testModelPath, testData, 0644)
	if err != nil {
		t.Fatalf("Failed to create test model file: %v", err)
	}
	
	// Test sharing model
	err = manager.ShareModelWithProviders(ctx, testModelPath, "test-model.gguf")
	if err != nil {
		t.Fatalf("Failed to share model: %v", err)
	}
	
	// Test getting shared models
	shared, err := manager.GetSharedModels(ctx)
	if err != nil {
		t.Fatalf("Failed to get shared models: %v", err)
	}
	
	// Should have at least one provider with the shared model
	found := false
	for provider, models := range shared {
		for _, model := range models {
			if model == "test-model.gguf" {
				found = true
				t.Logf("Found shared model %s in provider %s", model, provider)
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Error("Should find shared model in at least one provider")
	}
}

// TestModelConversion tests model conversion for cross-provider compatibility
func TestModelConversion(t *testing.T) {
	baseDir := t.TempDir()
	converter := NewModelConverter(baseDir)
	ctx := context.Background()
	
	// Test validation
	result, err := converter.ValidateConversion(FormatHF, FormatGGUF)
	if err != nil {
		t.Fatalf("Failed to validate conversion: %v", err)
	}
	
	if !result.IsPossible {
		t.Error("HF to GGUF conversion should be possible")
	}
	
	// Test unsupported conversion
	result, err = converter.ValidateConversion(FormatGGUF, FormatHF)
	if err != nil {
		t.Fatalf("Failed to validate conversion: %v", err)
	}
	
	// GGUF to HF might not be supported in all implementations
	t.Logf("GGUF to HF conversion possible: %v", result.IsPossible)
	
	// Test installed tools
	tools := converter.GetInstalledConversionTools()
	if len(tools) == 0 {
		t.Log("No conversion tools installed (expected in test environment)")
	}
}

// TestHardwareCompatibility tests hardware-aware provider selection
func TestHardwareCompatibility(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewLocalLLMManager(baseDir)
	ctx := context.Background()
	
	// Initialize with hardware detection
	err := manager.Initialize(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}
	
	// Mock hardware detector for testing
	detector := hardware.NewDetector()
	_, err = detector.Detect()
	if err != nil {
		t.Logf("Hardware detection failed (expected in test): %v", err)
	}
	
	// Test format compatibility based on hardware constraints
	constraints := map[string]interface{}{
		"gpu_required": true,
		"cpu_only":    false,
	}
	
	// Test that GPU providers are preferred for GPU-required scenarios
	registry := NewCrossProviderRegistry(baseDir)
	provider, err := registry.FindOptimalProvider("test-model", FormatGGUF, constraints)
	if err != nil {
		t.Fatalf("Failed to find optimal provider: %v", err)
	}
	
	if provider == nil {
		t.Error("Should find GPU-compatible provider")
	}
	
	t.Logf("Selected optimal provider: %s", provider.Name)
}

// BenchmarkCrossProviderPerformance benchmarks provider performance
func BenchmarkCrossProviderPerformance(b *testing.B) {
	baseDir := b.TempDir()
	registry := NewCrossProviderRegistry(baseDir)
	
	// Benchmark compatibility checking
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := ModelCompatibilityQuery{
			ModelID:       "llama-3-8b-instruct",
			SourceFormat:  FormatGGUF,
			TargetProvider: "vllm",
		}
		registry.CheckCompatibility(query)
	}
}

// BenchmarkModelSharing benchmarks model sharing performance
func BenchmarkModelSharing(b *testing.B) {
	baseDir := b.TempDir()
	manager := NewLocalLLMManager(baseDir)
	ctx := context.Background()
	
	// Initialize manager
	manager.Initialize(ctx)
	
	// Create test model file
	testModelPath := filepath.Join(baseDir, "benchmark-model.gguf")
	testData := make([]byte, 1024) // 1KB test data
	for i := range testData {
		testData[i] = byte(i % 256)
	}
	os.WriteFile(testModelPath, testData, 0644)
	
	// Benchmark sharing
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.ShareModelWithProviders(ctx, testModelPath, "benchmark-model.gguf")
	}
}

// TestIntegrationWorkflow tests complete integration workflow
func TestIntegrationWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	baseDir := t.TempDir()
	manager := NewLocalLLMManager(baseDir)
	registry := NewCrossProviderRegistry(baseDir)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// 1. Initialize all components
	t.Log("Step 1: Initializing components...")
	err := manager.Initialize(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}
	
	// 2. Test compatibility checking
	t.Log("Step 2: Testing compatibility checking...")
	providers := []string{"vllm", "llamacpp", "ollama"}
	modelID := "test-model"
	format := FormatGGUF
	
	for _, provider := range providers {
		query := ModelCompatibilityQuery{
			ModelID:       modelID,
			SourceFormat:  format,
			TargetProvider: provider,
		}
		
		result, err := registry.CheckCompatibility(query)
		if err != nil {
			t.Errorf("Compatibility check failed for %s: %v", provider, err)
			continue
		}
		
		t.Logf("Provider %s: compatible=%v, conversion=%v", 
			provider, result.IsCompatible, result.ConversionRequired)
		
		if !result.IsCompatible {
			t.Errorf("Provider %s should be compatible with GGUF", provider)
		}
	}
	
	// 3. Test model sharing simulation
	t.Log("Step 3: Testing model sharing...")
	testModelPath := filepath.Join(baseDir, "integration-test.gguf")
	testData := []byte("integration test model data")
	err = os.WriteFile(testModelPath, testData, 0644)
	if err != nil {
		t.Fatalf("Failed to create test model: %v", err)
	}
	
	err = manager.ShareModelWithProviders(ctx, testModelPath, "integration-test.gguf")
	if err != nil {
		t.Errorf("Failed to share model: %v", err)
	}
	
	// 4. Test listing shared models
	t.Log("Step 4: Testing shared model listing...")
	shared, err := manager.GetSharedModels(ctx)
	if err != nil {
		t.Errorf("Failed to get shared models: %v", err)
	} else {
		t.Logf("Found shared models in %d providers", len(shared))
	}
	
	// 5. Test provider status
	t.Log("Step 5: Testing provider status...")
	status := manager.GetProviderStatus(ctx)
	t.Logf("Provider status: %d providers", len(status))
	for name, provider := range status {
		t.Logf("  %s: %s (port: %d)", name, provider.Status, provider.DefaultPort)
	}
	
	t.Log("Integration workflow completed successfully!")
}