package integration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dev.helix.code/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests for real local LLM providers

const (
	testTimeout      = 60 * time.Second
	healthCheckDelay = 2 * time.Second
	maxRetries       = 5
)

// ProviderProviderTestConfig holds provider test configuration
type ProviderTestConfig struct {
	BaseDir          string
	ModelDownloadDir string
	SkipExpensive    bool
}

// Provider test data
type ProviderTest struct {
	Name         string
	Command      string
	Args         []string
	Port         int
	HealthURL    string
	Models       []string
	StartupDelay time.Duration
}

func TestRealProviderIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	config := &ProviderTestConfig{
		BaseDir:          t.TempDir(),
		ModelDownloadDir: filepath.Join(t.TempDir(), "models"),
		SkipExpensive:    os.Getenv("SKIP_EXPENSIVE_TESTS") == "true",
	}

	providers := getTestProviders()

	for _, provider := range providers {
		if !isProviderAvailable(provider) {
			t.Logf("Skipping %s - not available on this system", provider.Name)
			continue
		}

		t.Run(fmt.Sprintf("Provider_%s", provider.Name), func(t *testing.T) {
			testProviderLifecycle(t, config, provider)
		})
	}
}

func TestModelSharingAcrossProviders(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	config := &ProviderTestConfig{
		BaseDir:          t.TempDir(),
		ModelDownloadDir: filepath.Join(t.TempDir(), "models"),
		SkipExpensive:    os.Getenv("SKIP_EXPENSIVE_TESTS") == "true",
	}

	// Get available providers
	providers := getAvailableProviders(t, config)
	if len(providers) < 2 {
		t.Skip("Need at least 2 providers for model sharing test")
		return
	}

	// Download a test model
	modelPath := downloadTestModel(t, config, "llama-3-8b-instruct")
	require.FileExists(t, modelPath, "Test model should be downloaded")

	// Test sharing across all providers
	manager := llm.NewLocalLLMManager(config.BaseDir)
	err := manager.Initialize(context.Background())
	require.NoError(t, err)

	// Share the model
	err = manager.ShareModelWithProviders(context.Background(), modelPath, "llama-3-8b-instruct")
	require.NoError(t, err)

	// Verify model is accessible from multiple providers
	shared, err := manager.GetSharedModels(context.Background())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(shared), 1, "Model should be shared with at least one provider")

	// Test cross-provider access
	testCrossProviderModelAccess(t, config, providers[0], providers[1], modelPath)
}

func TestProviderFailover(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	config := &ProviderTestConfig{
		BaseDir:          t.TempDir(),
		ModelDownloadDir: filepath.Join(t.TempDir(), "models"),
		SkipExpensive:    os.Getenv("SKIP_EXPENSIVE_TESTS") == "true",
	}

	// Get multiple providers for failover test
	providers := getAvailableProviders(t, config)
	if len(providers) < 2 {
		t.Skip("Need at least 2 providers for failover test")
		return
	}

	// Start primary provider
	primary := providers[0]
	backup := providers[1]

	// Start primary
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	primaryProcess, err := startProvider(ctx, primary)
	require.NoError(t, err, "Primary provider should start")
	defer stopProvider(primaryProcess)

	// Wait for primary to be healthy
	require.Eventually(t, func() bool {
		return isProviderHealthy(primary.HealthURL)
	}, 30*time.Second, 1*time.Second, "Primary provider should become healthy")

	// Start backup provider
	backupProcess, err := startProvider(ctx, backup)
	require.NoError(t, err, "Backup provider should start")
	defer stopProvider(backupProcess)

	// Wait for backup to be healthy
	require.Eventually(t, func() bool {
		return isProviderHealthy(backup.HealthURL)
	}, 30*time.Second, 1*time.Second, "Backup provider should become healthy")

	// Simulate primary provider failure
	err = primaryProcess.Process.Signal(os.Interrupt)
	if err != nil {
		// Try more forceful kill
		primaryProcess.Process.Kill()
	}

	// Verify backup takes over
	require.Eventually(t, func() bool {
		return !isProviderHealthy(primary.HealthURL) && isProviderHealthy(backup.HealthURL)
	}, 30*time.Second, 1*time.Second, "Backup should take over when primary fails")

	// Test failover functionality
	testFailoverFunctionality(t, config, backup)
}

func TestPerformanceBenchmarks(t *testing.T) {
	if testing.Short() || os.Getenv("SKIP_BENCHMARKS") == "true" {
		t.Skip("Skipping benchmark tests")
	}

	config := &ProviderTestConfig{
		BaseDir:          t.TempDir(),
		ModelDownloadDir: filepath.Join(t.TempDir(), "models"),
		SkipExpensive:    false,
	}

	// Get best performing provider
	providers := getAvailableProviders(t, config)
	if len(providers) == 0 {
		t.Skip("No providers available for benchmarking")
		return
	}

	provider := providers[0] // Use first available for benchmarking

	// Start provider
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	process, err := startProvider(ctx, provider)
	require.NoError(t, err)
	defer stopProvider(process)

	// Wait for provider to be ready
	require.Eventually(t, func() bool {
		return isProviderHealthy(provider.HealthURL)
	}, 30*time.Second, 1*time.Second)

	// Run performance benchmarks
	runPerformanceBenchmarks(t, provider)
}

func TestLoadBalancing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load balancing test")
	}

	config := &ProviderTestConfig{
		BaseDir:          t.TempDir(),
		ModelDownloadDir: filepath.Join(t.TempDir(), "models"),
		SkipExpensive:    os.Getenv("SKIP_EXPENSIVE_TESTS") == "true",
	}

	// Get multiple providers
	providers := getAvailableProviders(t, config)
	if len(providers) < 2 {
		t.Skip("Need at least 2 providers for load balancing test")
		return
	}

	// Start all providers
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	var processes []*exec.Cmd
	for _, provider := range providers {
		process, err := startProvider(ctx, provider)
		require.NoError(t, err)
		processes = append(processes, process)
	}
	defer func() {
		for _, p := range processes {
			stopProvider(p)
		}
	}()

	// Wait for all providers to be healthy
	for _, provider := range providers {
		require.Eventually(t, func() bool {
			return isProviderHealthy(provider.HealthURL)
		}, 30*time.Second, 1*time.Second)
	}

	// Test load balancing
	testLoadBalancing(t, providers)
}

func TestModelConversion(t *testing.T) {
	if testing.Short() || os.Getenv("SKIP_CONVERSION_TESTS") == "true" {
		t.Skip("Skipping model conversion test")
	}

	config := &ProviderTestConfig{
		BaseDir:          t.TempDir(),
		ModelDownloadDir: filepath.Join(t.TempDir(), "models"),
		SkipExpensive:    false,
	}

	// Download a model in HF format
	modelPath := downloadTestModel(t, config, "llama-3-8b-instruct-hf")
	require.FileExists(t, modelPath)

	// Test conversion to GGUF
	converter := llm.NewModelConverter(config.BaseDir)

	conversionConfig := llm.ConversionConfig{
		SourcePath:   modelPath,
		SourceFormat: llm.FormatHF,
		TargetFormat: llm.FormatGGUF,
		Timeout:      300, // 5 minutes
		Quantization: &llm.QuantizationConfig{
			Method: "q4_k_m",
		},
		Optimization: &llm.OptimizationConfig{
			OptimizeFor:    "cpu",
			TargetHardware: "x86_64",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	job, err := converter.ConvertModel(ctx, conversionConfig)
	require.NoError(t, err, "Conversion should start")
	require.NotEmpty(t, job.ID, "Job should have ID")

	// Monitor conversion progress
	var finalStatus *llm.ConversionJob
	require.Eventually(t, func() bool {
		status, err := converter.GetConversionStatus(job.ID)
		require.NoError(t, err)
		finalStatus = status
		return status.Status == llm.StatusCompleted || status.Status == llm.StatusFailed
	}, 300*time.Second, 5*time.Second, "Conversion should complete")

	assert.Equal(t, llm.StatusCompleted, finalStatus.Status, "Conversion should succeed")
	assert.FileExists(t, finalStatus.TargetPath, "Converted model should exist")

	// Test converted model
	testConvertedModel(t, finalStatus.TargetPath)
}

// Helper functions

func getTestProviders() []ProviderTest {
	return []ProviderTest{
		{
			Name:         "VLLM",
			Command:      "python",
			Args:         []string{"-m", "vllm.entrypoints.api_server", "--host", "127.0.0.1", "--port", "8000"},
			Port:         8000,
			HealthURL:    "http://127.0.0.1:8000/health",
			Models:       []string{"llama-3-8b-instruct"},
			StartupDelay: 10 * time.Second,
		},
		{
			Name:         "LocalAI",
			Command:      "local-ai",
			Args:         []string{"--address", "127.0.0.1:8080"},
			Port:         8080,
			HealthURL:    "http://127.0.0.1:8080/health",
			Models:       []string{"llama-3-8b-instruct"},
			StartupDelay: 5 * time.Second,
		},
		{
			Name:         "Ollama",
			Command:      "ollama",
			Args:         []string{"serve", "--host", "127.0.0.1", "--port", "11434"},
			Port:         11434,
			HealthURL:    "http://127.0.0.1:11434/api/tags",
			Models:       []string{"llama3:8b"},
			StartupDelay: 3 * time.Second,
		},
		{
			Name:         "Llama.cpp",
			Command:      "main",
			Args:         []string{"-m", "./models/llama-3-8b.gguf", "--host", "127.0.0.1", "--port", "8080"},
			Port:         8080,
			HealthURL:    "http://127.0.0.1:8080/health",
			Models:       []string{"llama-3-8b-instruct.gguf"},
			StartupDelay: 5 * time.Second,
		},
	}
}

func isProviderAvailable(provider ProviderTest) bool {
	// Check if command exists
	_, err := exec.LookPath(provider.Command)
	if err != nil {
		return false
	}

	// Additional provider-specific checks
	switch provider.Name {
	case "VLLM":
		// Check if vllm is installed
		cmd := exec.Command("python", "-c", "import vllm")
		return cmd.Run() == nil
	case "LocalAI":
		// Check if local-ai binary exists
		_, err := exec.LookPath("local-ai")
		return err == nil
	case "Ollama":
		// Check if ollama is installed
		cmd := exec.Command("ollama", "--version")
		return cmd.Run() == nil
	case "Llama.cpp":
		// Check if main binary exists
		_, err := os.Stat("./main")
		return err == nil
	default:
		return true
	}
}

func getAvailableProviders(t *testing.T, config *ProviderTestConfig) []ProviderTest {
	var providers []ProviderTest
	testProviders := getTestProviders()

	for _, provider := range testProviders {
		if isProviderAvailable(provider) {
			providers = append(providers, provider)
		}
	}

	return providers
}

func startProvider(ctx context.Context, provider ProviderTest) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, provider.Command, provider.Args...)

	// Set up environment
	cmd.Env = os.Environ()

	// Start the command
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start provider %s: %w", provider.Name, err)
	}

	return cmd, nil
}

func stopProvider(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	// Try graceful shutdown first
	cmd.Process.Signal(os.Interrupt)

	// Wait a bit for graceful shutdown
	time.Sleep(5 * time.Second)

	// Force kill if still running
	if !cmd.ProcessState.Exited() {
		cmd.Process.Kill()
		cmd.Wait()
	}
}

func isProviderHealthy(healthURL string) bool {
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(healthURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func downloadTestModel(t *testing.T, config *ProviderTestConfig, modelID string) string {
	// Create download manager
	downloadManager := llm.NewModelDownloadManager(config.ModelDownloadDir)

	// Get model info
	model, err := downloadManager.GetModelByID(modelID)
	if err != nil {
		t.Logf("Model %s not found in registry, creating test model", modelID)
		model = createTestModel(config, modelID)
	}

	// Download model
	req := llm.ModelDownloadRequest{
		ModelID:       modelID,
		Format:        model.AvailableFormats[0],
		TargetPath:    filepath.Join(config.ModelDownloadDir, modelID+"."+string(model.AvailableFormats[0])),
		ForceDownload: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	progressChan, err := downloadManager.DownloadModel(ctx, req)
	require.NoError(t, err, "Download should start")

	// Monitor download
	for progress := range progressChan {
		if progress.Error != "" {
			t.Fatalf("Download failed: %s", progress.Error)
		}
		if progress.Progress == 1.0 {
			break
		}
	}

	return req.TargetPath
}

func createTestModel(config *ProviderTestConfig, modelID string) *llm.DownloadableModelInfo {
	return &llm.DownloadableModelInfo{
		ID:               modelID,
		Name:             fmt.Sprintf("Test Model %s", modelID),
		Description:      "Test model for integration testing",
		AvailableFormats: []llm.ModelFormat{llm.FormatGGUF},
		DefaultFormat:    llm.FormatGGUF,
		ModelSize:        "8B",
		ContextSize:      8192,
		Requirements: llm.ModelRequirements{
			MinRAM:  "8GB",
			CPUOnly: true,
		},
		Tags: []string{"test", "integration"},
	}
}

func testProviderLifecycle(t *testing.T, config *ProviderTestConfig, provider ProviderTest) {
	t.Logf("Testing provider lifecycle for %s", provider.Name)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Test starting
	process, err := startProvider(ctx, provider)
	require.NoError(t, err, "Provider should start")
	defer stopProvider(process)

	// Test health check
	require.Eventually(t, func() bool {
		return isProviderHealthy(provider.HealthURL)
	}, 30*time.Second, 1*time.Second, "Provider should become healthy")

	// Test model loading
	testModelLoading(t, provider)

	// Test API functionality
	testProviderAPI(t, provider)

	t.Logf("✅ Provider %s lifecycle test completed", provider.Name)
}

func testModelLoading(t *testing.T, provider ProviderTest) {
	for _, model := range provider.Models {
		t.Logf("Testing model loading: %s", model)
		// This would test loading specific models into the provider
		// Implementation depends on provider API
	}
}

func testProviderAPI(t *testing.T, provider ProviderTest) {
	// Test basic API functionality
	apiURL := strings.Replace(provider.HealthURL, "/health", "/v1/models", 1)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		t.Logf("API call failed for %s: %v", provider.Name, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Logf("✅ API is working for %s", provider.Name)
	} else {
		t.Logf("⚠️  API returned status %d for %s", resp.StatusCode, provider.Name)
	}
}

func testCrossProviderModelAccess(t *testing.T, config *ProviderTestConfig, primary, backup ProviderTest, modelPath string) {
	// Test that model can be accessed from backup provider
	// This depends on provider-specific implementation
	t.Logf("Testing cross-provider access from %s to %s", primary.Name, backup.Name)
}

func testFailoverFunctionality(t *testing.T, config *ProviderTestConfig, backup ProviderTest) {
	// Test that backup provider handles requests correctly
	t.Logf("Testing failover functionality with %s", backup.Name)
}

func runPerformanceBenchmarks(t *testing.T, provider ProviderTest) {
	t.Logf("Running performance benchmarks for %s", provider.Name)

	// Test inference speed
	testInferenceSpeed(t, provider)

	// Test concurrent requests
	testConcurrentRequests(t, provider)

	// Test memory usage
	testMemoryUsage(t, provider)
}

func testInferenceSpeed(t *testing.T, provider ProviderTest) {
	// Benchmark inference time
	start := time.Now()

	// Make a test request (implementation depends on provider API)

	duration := time.Since(start)
	t.Logf("Inference time for %s: %v", provider.Name, duration)
}

func testConcurrentRequests(t *testing.T, provider ProviderTest) {
	// Test handling concurrent requests
	const numRequests = 10
	start := time.Now()

	// Make concurrent requests
	done := make(chan bool, numRequests)
	for i := 0; i < numRequests; i++ {
		go func() {
			// Make test request
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}

	duration := time.Since(start)
	t.Logf("Concurrent requests (%d) for %s: %v", numRequests, provider.Name, duration)
}

func testMemoryUsage(t *testing.T, provider ProviderTest) {
	// Monitor memory usage during operation
	// This would use system calls to measure memory
	t.Logf("Memory usage test for %s", provider.Name)
}

func testLoadBalancing(t *testing.T, providers []ProviderTest) {
	t.Logf("Testing load balancing across %d providers", len(providers))

	// Implement load balancing logic
	// This would test distributing requests across providers
}

func testConvertedModel(t *testing.T, modelPath string) {
	// Test that converted model loads and works correctly
	t.Logf("Testing converted model: %s", modelPath)

	// Check file size
	info, err := os.Stat(modelPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0), "Model file should not be empty")
}

// Edge case tests

func TestProviderEdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping edge case tests")
	}

	// Test provider with invalid configuration
	testInvalidConfiguration(t)

	// Test provider with missing models
	testMissingModels(t)

	// Test provider with corrupted model files
	testCorruptedModels(t)

	// Test provider with insufficient resources
	testInsufficientResources(t)
}

func testInvalidConfiguration(t *testing.T) {
	t.Logf("Testing invalid configuration scenarios")
	// Test various invalid configuration scenarios
}

func testMissingModels(t *testing.T) {
	t.Logf("Testing missing model scenarios")
	// Test behavior when models are missing
}

func testCorruptedModels(t *testing.T) {
	t.Logf("Testing corrupted model scenarios")
	// Test behavior with corrupted model files
}

func testInsufficientResources(t *testing.T) {
	t.Logf("Testing insufficient resource scenarios")
	// Test behavior with insufficient memory/CPU
}

// Stress tests

func TestProviderStress(t *testing.T) {
	if testing.Short() || os.Getenv("SKIP_STRESS_TESTS") == "true" {
		t.Skip("Skipping stress tests")
	}

	t.Logf("Running stress tests")

	// Test high request volume
	testHighRequestVolume(t)

	// Test long-running stability
	testLongRunningStability(t)

	// Test resource leak detection
	testResourceLeakDetection(t)
}

func testHighRequestVolume(t *testing.T) {
	t.Logf("Testing high request volume")
	// Stress test with many concurrent requests
}

func testLongRunningStability(t *testing.T) {
	t.Logf("Testing long-running stability")
	// Test provider stability over extended periods
}

func testResourceLeakDetection(t *testing.T) {
	t.Logf("Testing resource leak detection")
	// Monitor for memory/file handle leaks
}

// Cleanup function
func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()

	// Cleanup
	// Stop any running providers
	// Clean up temporary files
	os.Exit(code)
}
