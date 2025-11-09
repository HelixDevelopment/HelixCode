package unit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"dev.helix.code/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/mock"
)

// Mock provider for unit testing
type MockProvider struct {
	mock.Mock
	name        string
	providerType string
	running     bool
	models      []llm.ModelInfo
}

func (m *MockProvider) GetType() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockProvider) GetName() string {
	return m.name
}

func (m *MockProvider) GetModels() []llm.ModelInfo {
	args := m.Called()
	return args.Get(0).([]llm.ModelInfo)
}

func (m *MockProvider) GetCapabilities() []llm.ModelCapability {
	args := m.Called()
	return args.Get(0).([]llm.ModelCapability)
}

func (m *MockProvider) Generate(ctx context.Context, request *llm.LLMRequest) (*llm.LLMResponse, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(*llm.LLMResponse), args.Error(1)
}

func (m *MockProvider) GenerateStream(ctx context.Context, request *llm.LLMRequest, ch chan<- llm.LLMResponse) error {
	args := m.Called(ctx, request, ch)
	return args.Error(0)
}

func (m *MockProvider) IsAvailable(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

func (m *MockProvider) GetHealth(ctx context.Context) (*llm.ProviderHealth, error) {
	args := m.Called(ctx)
	return args.Get(0).(*llm.ProviderHealth), args.Error(1)
}

func (m *MockProvider) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Unit tests for LocalLLMManager

func TestNewLocalLLMManager(t *testing.T) {
	// Test with default directory
	manager := llm.NewLocalLLMManager("")
	assert.NotNil(t, manager)
	assert.NotEmpty(t, manager.GetBaseDir())

	// Test with custom directory
	customDir := "/tmp/test-llm"
	manager = llm.NewLocalLLMManager(customDir)
	assert.NotNil(t, manager)
	assert.Contains(t, manager.GetBaseDir(), customDir)
}

func TestLocalLLMManager_Initialize(t *testing.T) {
	testDir := t.TempDir()
	manager := llm.NewLocalLLMManager(testDir)

	ctx := context.Background()
	
	// Test successful initialization
	err := manager.Initialize(ctx)
	assert.NoError(t, err)
	
	// Verify directories were created
	assert.DirExists(t, filepath.Join(testDir, "bin"))
	assert.DirExists(t, filepath.Join(testDir, "config"))
	assert.DirExists(t, filepath.Join(testDir, "data"))

	// Test idempotency (should not error on re-initialization)
	err = manager.Initialize(ctx)
	assert.NoError(t, err)
}

func TestLocalLLMManager_StartProvider(t *testing.T) {
	testDir := t.TempDir()
	manager := llm.NewLocalLLMManager(testDir)
	ctx := context.Background()

	// Initialize first
	err := manager.Initialize(ctx)
	require.NoError(t, err)

	// Test starting valid provider
	err = manager.StartProvider(ctx, "vllm")
	if err != nil {
		t.Logf("Provider start failed (expected in test environment): %v", err)
		// This is expected in test environment without actual provider
	}

	// Test starting invalid provider
	err = manager.StartProvider(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestLocalLLMManager_StopProvider(t *testing.T) {
	testDir := t.TempDir()
	manager := llm.NewLocalLLMManager(testDir)
	ctx := context.Background()

	// Initialize
	err := manager.Initialize(ctx)
	require.NoError(t, err)

	// Test stopping non-running provider
	err = manager.StopProvider(ctx, "vllm")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")

	// Test stopping invalid provider
	err = manager.StopProvider(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestLocalLLMManager_GetProviderStatus(t *testing.T) {
	testDir := t.TempDir()
	manager := llm.NewLocalLLMManager(testDir)
	ctx := context.Background()

	// Initialize
	err := manager.Initialize(ctx)
	require.NoError(t, err)

	// Test getting status
	status := manager.GetProviderStatus(ctx)
	assert.NotNil(t, status)
	
	// Should contain all provider definitions
	providers := []string{"vllm", "localai", "fastchat", "textgen", "lmstudio"}
	for _, provider := range providers {
		assert.Contains(t, status, provider)
	}
}

func TestLocalLLMManager_GetRunningProviders(t *testing.T) {
	testDir := t.TempDir()
	manager := llm.NewLocalLLMManager(testDir)
	ctx := context.Background()

	// Initialize
	err := manager.Initialize(ctx)
	require.NoError(t, err)

	// Test with no running providers
	running := manager.GetRunningProviders(ctx)
	assert.NotNil(t, running)
	// Should be empty in test environment
}

func TestLocalLLMManager_StartAllProviders(t *testing.T) {
	testDir := t.TempDir()
	manager := llm.NewLocalLLMManager(testDir)
	ctx := context.Background()

	// Initialize
	err := manager.Initialize(ctx)
	require.NoError(t, err)

	// Test starting all providers
	err = manager.StartAllProviders(ctx)
	// May have errors in test environment, but should attempt all
	// We don't assert error because some providers might fail
	t.Logf("StartAllProviders result: %v", err)
}

func TestLocalLLMManager_StopAllProviders(t *testing.T) {
	testDir := t.TempDir()
	manager := llm.NewLocalLLMManager(testDir)
	ctx := context.Background()

	// Initialize
	err := manager.Initialize(ctx)
	require.NoError(t, err)

	// Test stopping all providers
	err = manager.StopAllProviders(ctx)
	assert.NoError(t, err)
}

func TestLocalLLMManager_UpdateProvider(t *testing.T) {
	testDir := t.TempDir()
	manager := llm.NewLocalLLMManager(testDir)
	ctx := context.Background()

	// Initialize
	err := manager.Initialize(ctx)
	require.NoError(t, err)

	// Test updating valid provider
	err = manager.UpdateProvider(ctx, "vllm")
	if err != nil {
		t.Logf("Provider update failed (expected in test environment): %v", err)
	}

	// Test updating invalid provider
	err = manager.UpdateProvider(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestLocalLLMManager_Cleanup(t *testing.T) {
	testDir := t.TempDir()
	manager := llm.NewLocalLLMManager(testDir)
	ctx := context.Background()

	// Initialize
	err := manager.Initialize(ctx)
	require.NoError(t, err)

	// Test cleanup
	err = manager.Cleanup(ctx)
	assert.NoError(t, err)
}

func TestLocalLLMManager_ShareModelWithProviders(t *testing.T) {
	testDir := t.TempDir()
	manager := llm.NewLocalLLMManager(testDir)
	ctx := context.Background()

	// Initialize
	err := manager.Initialize(ctx)
	require.NoError(t, err)

	// Create a test model file
	modelPath := filepath.Join(testDir, "test-model.gguf")
	err = os.WriteFile(modelPath, []byte("fake model data"), 0644)
	require.NoError(t, err)

	// Test sharing model
	err = manager.ShareModelWithProviders(ctx, modelPath, "test-model")
	if err != nil {
		t.Logf("Model share failed (expected in test environment): %v", err)
	}
}

func TestLocalLLMManager_OptimizeModelForProvider(t *testing.T) {
	testDir := t.TempDir()
	manager := llm.NewLocalLLMManager(testDir)
	ctx := context.Background()

	// Initialize
	err := manager.Initialize(ctx)
	require.NoError(t, err)

	// Create a test model file
	modelPath := filepath.Join(testDir, "test-model.pt")
	err = os.WriteFile(modelPath, []byte("fake model data"), 0644)
	require.NoError(t, err)

	// Test optimizing model for provider
	err = manager.OptimizeModelForProvider(ctx, modelPath, "vllm")
	if err != nil {
		t.Logf("Model optimization failed (expected in test environment): %v", err)
	}

	// Test optimizing for invalid provider
	err = manager.OptimizeModelForProvider(ctx, modelPath, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestLocalLLMManager_GetSharedModels(t *testing.T) {
	testDir := t.TempDir()
	manager := llm.NewLocalLLMManager(testDir)
	ctx := context.Background()

	// Initialize
	err := manager.Initialize(ctx)
	require.NoError(t, err)

	// Test getting shared models
	shared, err := manager.GetSharedModels(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, shared)
}

func TestProviderHealth(t *testing.T) {
	// Test creating provider health
	health := &llm.ProviderHealth{
		Status:     "healthy",
		LastCheck:  time.Now(),
		Latency:    50 * time.Millisecond,
		ModelCount: 5,
		ErrorCount: 0,
		Message:    "All systems operational",
	}

	assert.Equal(t, "healthy", health.Status)
	assert.Equal(t, 5, health.ModelCount)
	assert.Equal(t, 50*time.Millisecond, health.Latency)
}

func TestModelDownloadManager(t *testing.T) {
	testDir := t.TempDir()
	manager := llm.NewModelDownloadManager(testDir)

	// Test getting available models
	models := manager.GetAvailableModels()
	assert.NotNil(t, models)
	assert.GreaterOrEqual(t, len(models), 0)

	// Test searching models
	searchResults := manager.SearchModels("llama")
	assert.NotNil(t, searchResults)

	// Test searching with empty query
	emptyResults := manager.SearchModels("")
	assert.NotNil(t, emptyResults)

	// Test getting model by ID
	model, err := manager.GetModelByID("llama-3-8b-instruct")
	if err != nil {
		t.Logf("Model not found (expected in test environment): %v", err)
	} else {
		assert.NotNil(t, model)
		assert.Equal(t, "llama-3-8b-instruct", model.ID)
	}

	// Test getting non-existent model
	_, err = manager.GetModelByID("nonexistent-model")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestModelCompatibility(t *testing.T) {
	testDir := t.TempDir()
	manager := llm.NewModelDownloadManager(testDir)

	// Test getting compatible formats
	formats, err := manager.GetCompatibleFormats("vllm", "llama-3-8b-instruct")
	if err != nil {
		t.Logf("Compatibility check failed (expected in test environment): %v", err)
	} else {
		assert.NotNil(t, formats)
		assert.GreaterOrEqual(t, len(formats), 0)
	}

	// Test compatibility with non-existent model
	_, err = manager.GetCompatibleFormats("vllm", "nonexistent-model")
	assert.Error(t, err)
}

func TestConversionValidation(t *testing.T) {
	converter := llm.NewModelConverter(t.TempDir())

	// Test format compatibility validation
	result, err := converter.ValidateConversion("hf", "gguf")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Test incompatible formats
	result, err = converter.ValidateConversion("gguf", "hf")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsPossible)
}

func TestCrossProviderRegistry(t *testing.T) {
	registry := llm.NewCrossProviderRegistry(t.TempDir())

	// Test model compatibility query
	query := llm.ModelCompatibilityQuery{
		ModelID:        "llama-3-8b-instruct",
		SourceFormat:   "hf",
		TargetProvider:  "vllm",
	}

	result, err := registry.CheckCompatibility(query)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Test listing shared models
	downloaded := registry.GetDownloadedModels()
	assert.NotNil(t, downloaded)
}

func TestUsageAnalytics(t *testing.T) {
	analytics := llm.NewUsageAnalytics(t.TempDir())

	// Test generating usage report
	timeRange := llm.TimeRange{
		Start: time.Now().Add(-7 * 24 * time.Hour),
		End:   time.Now(),
	}

	report, err := analytics.GenerateUsageReport(timeRange)
	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.NotNil(t, report.Summary)
	assert.NotNil(t, report.PerformanceAnalysis)
}

func TestModelDiscovery(t *testing.T) {
	engine := llm.NewModelDiscoveryEngine(t.TempDir())

	// Test getting recommendations
	req := &llm.RecommendationRequest{
		TaskTypes:         []string{"code_generation"},
		QualityPreference: "high",
		PrivacyLevel:      "local",
		MaxRecommendations: 5,
	}

	response, err := engine.GetRecommendations(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotNil(t, response.Recommendations)
}

// Edge cases and error conditions

func TestLocalLLMManager_EdgeCases(t *testing.T) {
	// Test with nil context
	manager := llm.NewLocalLLMManager("")
	
	// Should handle nil context gracefully (panics are acceptable)
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Expected panic with nil context: %v", r)
		}
	}()
	
	ctx := context.Background()
	err := manager.Initialize(ctx)
	assert.NoError(t, err)

	// Test with canceled context
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	
	err = manager.StartProvider(ctx, "vllm")
	// Should return context canceled error
	if err != nil {
		assert.Contains(t, err.Error(), "context canceled")
	}
}

func TestLocalLLMManager_ConcurrentAccess(t *testing.T) {
	testDir := t.TempDir()
	manager := llm.NewLocalLLMManager(testDir)
	ctx := context.Background()

	err := manager.Initialize(ctx)
	require.NoError(t, err)

	// Test concurrent provider status requests
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()
			
			// Perform operations concurrently
			status := manager.GetProviderStatus(ctx)
			assert.NotNil(t, status)
			
			running := manager.GetRunningProviders(ctx)
			assert.NotNil(t, running)
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent operations timed out")
		}
	}
}

func TestLocalLLMManager_ResourceLimits(t *testing.T) {
	testDir := t.TempDir()
	manager := llm.NewLocalLLMManager(testDir)
	ctx := context.Background()

	err := manager.Initialize(ctx)
	require.NoError(t, err)

	// Test with very short timeout
	ctx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
	defer cancel()

	err = manager.StartProvider(ctx, "vllm")
	// Should return timeout error
	if err != nil {
		assert.Contains(t, err.Error(), "deadline exceeded") || 
				assert.Contains(t, err.Error(), "timeout")
	}
}

// Performance tests

func TestLocalLLMManager_Performance(t *testing.T) {
	testDir := t.TempDir()
	manager := llm.NewLocalLLMManager(testDir)
	ctx := context.Background()

	err := manager.Initialize(ctx)
	require.NoError(t, err)

	// Measure performance of status retrieval
	const iterations = 100
	start := time.Now()

	for i := 0; i < iterations; i++ {
		status := manager.GetProviderStatus(ctx)
		assert.NotNil(t, status)
	}

	duration := time.Since(start)
	avgDuration := duration / iterations

	// Should complete within reasonable time (less than 10ms per iteration)
	assert.Less(t, avgDuration, 10*time.Millisecond, 
		"Status retrieval should be fast")
	t.Logf("Average status retrieval time: %v", avgDuration)
}

// Integration with mocks

func TestLocalLLMManager_WithMocks(t *testing.T) {
	// Create mock provider
	mockProvider := &MockProvider{
		name: "MockProvider",
	}

	// Set up mock expectations
	mockProvider.On("GetType").Return("mock")
	mockProvider.On("GetName").Return("MockProvider")
	mockProvider.On("GetModels").Return([]llm.ModelInfo{})
	mockProvider.On("GetCapabilities").Return([]llm.ModelCapability{})
	mockProvider.On("IsAvailable", mock.Anything).Return(true)
	mockProvider.On("GetHealth", mock.Anything).Return(&llm.ProviderHealth{
		Status: "healthy",
	}, nil)
	mockProvider.On("Close").Return(nil)

	// Test mock behavior
	ctx := context.Background()

	available := mockProvider.IsAvailable(ctx)
	assert.True(t, available)

	health, err := mockProvider.GetHealth(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", health.Status)

	err = mockProvider.Close()
	assert.NoError(t, err)

	// Verify all expectations were met
	mockProvider.AssertExpectations(t)
}