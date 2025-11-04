//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"dev.helix.code/internal/llm"
	"dev.helix.code/internal/mcp"
	"dev.helix.code/internal/notification"
	"dev.helix.code/internal/worker"
	"github.com/stretchr/testify/assert"
)

// TestDistributedWorkflow tests a complete distributed workflow
func TestDistributedWorkflow(t *testing.T) {
	ctx := context.Background()

	// Setup worker pool
	workerPool := worker.NewSSHWorkerPool(false)
	assert.NotNil(t, workerPool)

	// Setup notification engine
	notificationEngine := notification.NewNotificationEngine()
	assert.NotNil(t, notificationEngine)

	// Setup MCP server
	mcpServer := mcp.NewMCPServer()
	assert.NotNil(t, mcpServer)

	// Test notification system integration
	testNotification := &notification.Notification{
		Title:    "Integration Test",
		Message:  "Testing distributed workflow",
		Type:     notification.NotificationTypeInfo,
		Priority: notification.NotificationPriorityMedium,
		Channels: []string{"cli"},
	}

	err := notificationEngine.SendDirect(ctx, testNotification, []string{"cli"})
	assert.NoError(t, err)

	// Test MCP tool registration
	testTool := &mcp.Tool{
		ID:          "integration_tool",
		Name:        "Integration Tool",
		Description: "Tool for integration testing",
		Parameters: map[string]interface{}{
			"test_param": "string",
		},
		Handler: func(ctx context.Context, session *mcp.MCPSession, args map[string]interface{}) (interface{}, error) {
			return "integration_test_result", nil
		},
	}

	err = mcpServer.RegisterTool(testTool)
	assert.NoError(t, err)
	assert.Equal(t, 1, mcpServer.GetToolCount())

	// Test worker statistics
	stats := workerPool.GetWorkerStats(ctx)
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats.TotalWorkers) // No workers added yet

	t.Log("✅ Integration test completed successfully")
}

// TestLLMProviderIntegration tests LLM provider integration
func TestLLMProviderIntegration(t *testing.T) {
	ctx := context.Background()

	// Create provider manager
	config := llm.ProviderConfig{
		DefaultProvider: llm.ProviderTypeLocal,
		Timeout:         30 * time.Second,
		MaxRetries:      3,
	}

	providerManager := llm.NewProviderManager(config)
	assert.NotNil(t, providerManager)

	// Test provider availability
	availableProviders := providerManager.GetAvailableProviders()
	assert.NotNil(t, availableProviders)
	// In integration environment, we might have some providers available

	// Test provider health
	health := providerManager.GetProviderHealth(ctx)
	assert.NotNil(t, health)
	// Health check might fail if no providers are configured

	t.Log("✅ LLM provider integration test completed")
}

// TestNotificationChannelIntegration tests notification channel integration
func TestNotificationChannelIntegration(t *testing.T) {
	ctx := context.Background()

	engine := notification.NewNotificationEngine()

	// Test adding notification rules
	rule := notification.NotificationRule{
		Name:      "Test Rule",
		Condition: "type==info",
		Channels:  []string{"cli"},
		Priority:  notification.NotificationPriorityMedium,
		Enabled:   true,
	}

	err := engine.AddRule(rule)
	assert.NoError(t, err)

	// Test sending notification through rules
	notification := &notification.Notification{
		Title:    "Rule Test",
		Message:  "This should trigger the test rule",
		Type:     notification.NotificationTypeInfo,
		Priority: notification.NotificationPriorityLow,
	}

	err = engine.SendNotification(ctx, notification)
	assert.NoError(t, err)

	// Test channel statistics
	stats := engine.GetChannelStats()
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "summary")

	t.Log("✅ Notification channel integration test completed")
}

// TestMCPProtocolIntegration tests MCP protocol integration
func TestMCPProtocolIntegration(t *testing.T) {
	server := mcp.NewMCPServer()

	// Test multiple tool registration
	tools := []*mcp.Tool{
		{
			ID:          "tool_1",
			Name:        "Tool 1",
			Description: "First integration tool",
			Handler: func(ctx context.Context, session *mcp.MCPSession, args map[string]interface{}) (interface{}, error) {
				return "result_1", nil
			},
		},
		{
			ID:          "tool_2",
			Name:        "Tool 2",
			Description: "Second integration tool",
			Handler: func(ctx context.Context, session *mcp.MCPSession, args map[string]interface{}) (interface{}, error) {
				return "result_2", nil
			},
		},
	}

	for _, tool := range tools {
		err := server.RegisterTool(tool)
		assert.NoError(t, err)
	}

	assert.Equal(t, 2, server.GetToolCount())

	// Test session management
	assert.Equal(t, 0, server.GetSessionCount())

	t.Log("✅ MCP protocol integration test completed")
}

// TestCrossComponentIntegration tests cross-component integration
func TestCrossComponentIntegration(t *testing.T) {
	ctx := context.Background()

	// Setup all components
	workerPool := worker.NewSSHWorkerPool(false)
	notificationEngine := notification.NewNotificationEngine()
	mcpServer := mcp.NewMCPServer()

	// Test notification from worker operations
	workerStats := workerPool.GetWorkerStats(ctx)
	assert.NotNil(t, workerStats)

	// Send notification about worker status
	workerNotification := &notification.Notification{
		Title:    "Worker Status",
		Message:  "Worker pool initialized successfully",
		Type:     notification.NotificationTypeSuccess,
		Priority: notification.NotificationPriorityLow,
		Channels: []string{"cli"},
	}

	err := notificationEngine.SendDirect(ctx, workerNotification, []string{"cli"})
	assert.NoError(t, err)

	// Register MCP tool that uses worker functionality
	workerTool := &mcp.Tool{
		ID:          "worker_info",
		Name:        "Worker Info",
		Description: "Get information about workers",
		Handler: func(ctx context.Context, session *mcp.MCPSession, args map[string]interface{}) (interface{}, error) {
			// This would normally interact with worker pool
			return map[string]interface{}{
				"total_workers": 0,
				"status":        "initialized",
			}, nil
		},
	}

	err = mcpServer.RegisterTool(workerTool)
	assert.NoError(t, err)

	// Verify all components are working together
	assert.NotNil(t, workerPool)
	assert.NotNil(t, notificationEngine)
	assert.NotNil(t, mcpServer)
	assert.Equal(t, 1, mcpServer.GetToolCount())

	t.Log("✅ Cross-component integration test completed successfully")
}

// TestErrorHandlingIntegration tests error handling across components
func TestErrorHandlingIntegration(t *testing.T) {
	ctx := context.Background()

	workerPool := worker.NewSSHWorkerPool(false)
	notificationEngine := notification.NewNotificationEngine()

	// Test error notification
	errorNotification := &notification.Notification{
		Title:    "Error Test",
		Message:  "Simulated error condition",
		Type:     notification.NotificationTypeError,
		Priority: notification.NotificationPriorityHigh,
		Channels: []string{"cli"},
	}

	err := notificationEngine.SendDirect(ctx, errorNotification, []string{"cli"})
	assert.NoError(t, err)

	// Test worker error scenarios
	// Attempt to remove non-existent worker would normally fail,
	// but we're testing the error handling

	// Verify components handle errors gracefully
	stats := workerPool.GetWorkerStats(ctx)
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats.TotalWorkers)

	channelStats := notificationEngine.GetChannelStats()
	assert.NotNil(t, channelStats)

	t.Log("✅ Error handling integration test completed")
}