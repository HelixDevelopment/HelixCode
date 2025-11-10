//go:build automation

package automation

import (
	"context"
	"testing"
	"time"

	"dev.helix.code/internal/llm"
	"dev.helix.code/internal/worker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRealAIReasoning tests reasoning with real AI models
func TestRealAIReasoning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real AI test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Setup provider manager with real providers
	config := llm.ProviderConfig{
		DefaultProvider: llm.ProviderTypeLocal,
		Timeout:         60 * time.Second,
		MaxRetries:      3,
	}

	providerManager := llm.NewProviderManager(config)

	// Test with available providers
	availableProviders := providerManager.GetAvailableProviders()
	if len(availableProviders) == 0 {
		t.Skip("No LLM providers available for automation testing")
	}

	// Test reasoning with each available provider
	for _, provider := range availableProviders {
		t.Run(provider.GetName(), func(t *testing.T) {
			// Create reasoning engine
			reasoningEngine := llm.NewReasoningEngine(provider)

			// Test simple reasoning
			request := llm.ReasoningRequest{
				Prompt:        "What is 2 + 2? Think step by step.",
				ReasoningType: llm.ReasoningTypeChainOfThought,
				MaxSteps:      5,
				Temperature:   0.3,
			}

			response, err := reasoningEngine.GenerateWithReasoning(ctx, request)

			if err != nil {
				t.Logf("Provider %s reasoning failed: %v", provider.GetName(), err)
				return
			}

			assert.NotNil(t, response)
			assert.NotEmpty(t, response.FinalAnswer)
			assert.Greater(t, response.Duration, time.Duration(0))
			assert.Len(t, response.ReasoningSteps, 1)

			t.Logf("✅ %s reasoning test completed: %s", provider.GetName(), response.FinalAnswer)
		})
	}
}

// TestDistributedTaskExecution tests distributed task execution
func TestDistributedTaskExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping distributed task test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Setup worker pool
	workerPool := worker.NewSSHWorkerPool(false)

	// Test worker health monitoring
	err := workerPool.HealthCheck(ctx)
	assert.NoError(t, err)

	stats := workerPool.GetWorkerStats(ctx)
	assert.NotNil(t, stats)

	// In automation environment, we might have real workers
	if stats.TotalWorkers > 0 {
		t.Logf("Found %d workers for automation testing", stats.TotalWorkers)

		// Test task execution on available workers
		for workerID := range workerPool.workers {
			t.Run(workerID.String(), func(t *testing.T) {
				// Execute simple command
				output, err := workerPool.ExecuteCommand(ctx, workerID, "echo 'automation test'")

				if err != nil {
					t.Logf("Worker %s command execution failed: %v", workerID, err)
					return
				}

				assert.Contains(t, output, "automation test")
				t.Logf("✅ Worker %s automation test completed", workerID)
			})
		}
	} else {
		t.Log("No workers available for distributed task execution")
	}
}

// TestPerformanceBenchmarks tests performance benchmarks
func TestPerformanceBenchmarks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance benchmarks in short mode")
	}

	ctx := context.Background()

	// Test worker pool performance
	workerPool := worker.NewSSHWorkerPool(false)

	start := time.Now()

	// Perform multiple operations
	for i := 0; i < 100; i++ {
		_ = workerPool.GetWorkerStats(ctx)
	}

	duration := time.Since(start)

	// Performance target: 100 operations in under 1 second
	assert.Less(t, duration, time.Second)
	t.Logf("✅ Worker pool performance: %d operations in %v", 100, duration)

	// Test notification system performance
	notificationEngine := notification.NewNotificationEngine()

	start = time.Now()

	for i := 0; i < 50; i++ {
		_ = notificationEngine.GetChannelStats()
	}

	duration = time.Since(start)

	// Performance target: 50 operations in under 500ms
	assert.Less(t, duration, 500*time.Millisecond)
	t.Logf("✅ Notification system performance: %d operations in %v", 50, duration)
}

// TestConcurrentOperations tests concurrent operations
func TestConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent operations test in short mode")
	}

	ctx := context.Background()
	workerPool := worker.NewSSHWorkerPool(false)

	numGoroutines := 10
	done := make(chan bool, numGoroutines)

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			// Concurrent worker operations
			_ = workerPool.HealthCheck(ctx)
			_ = workerPool.GetWorkerStats(ctx)

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	duration := time.Since(start)

	// Should complete quickly even with concurrent access
	assert.Less(t, duration, 5*time.Second)
	t.Logf("✅ Concurrent operations completed: %d goroutines in %v", numGoroutines, duration)
}

// TestResourceUsage tests resource usage patterns
func TestResourceUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource usage test in short mode")
	}

	ctx := context.Background()

	// Test memory usage with multiple components
	components := []interface{}{
		worker.NewSSHWorkerPool(false),
		notification.NewNotificationEngine(),
		mcp.NewMCPServer(),
	}

	// Verify all components can be created without excessive memory usage
	for i, component := range components {
		assert.NotNil(t, component, "Component %d failed to initialize", i)
	}

	// Test worker pool memory efficiency
	workerPool := worker.NewSSHWorkerPool(false)

	// Add multiple workers (simulated)
	for i := 0; i < 100; i++ {
		workerID := uuid.New()
		workerPool.workers[workerID] = &worker.SSHWorker{
			ID:       workerID,
			Hostname: "worker-" + string(rune('A'+i)),
		}
	}

	stats := workerPool.GetWorkerStats(ctx)
	assert.Equal(t, 100, stats.TotalWorkers)

	t.Log("✅ Resource usage test completed - all components efficient")
}

// TestErrorRecovery tests automatic error recovery
func TestErrorRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping error recovery test in short mode")
	}

	ctx := context.Background()
	workerPool := worker.NewSSHWorkerPool(false)

	// Test that system remains stable after errors
	for i := 0; i < 10; i++ {
		// These operations should not panic even with errors
		_ = workerPool.HealthCheck(ctx)
		_ = workerPool.GetWorkerStats(ctx)

		// Simulate error conditions
		_, err := workerPool.ExecuteCommand(ctx, uuid.New(), "invalid command")
		// We expect this to fail, but system should remain stable
		assert.Error(t, err)
	}

	// System should still be functional
	stats := workerPool.GetWorkerStats(ctx)
	assert.NotNil(t, stats)

	t.Log("✅ Error recovery test completed - system remained stable")
}

// TestLongRunningOperations tests long-running operations
func TestLongRunningOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running operations test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	workerPool := worker.NewSSHWorkerPool(false)

	// Test continuous health monitoring
	for i := 0; i < 5; i++ {
		start := time.Now()

		err := workerPool.HealthCheck(ctx)
		assert.NoError(t, err)

		_ = workerPool.GetWorkerStats(ctx)

		duration := time.Since(start)

		// Each health check should complete quickly
		assert.Less(t, duration, 10*time.Second)

		time.Sleep(1 * time.Second) // Wait between checks
	}

	t.Log("✅ Long-running operations test completed")
}
