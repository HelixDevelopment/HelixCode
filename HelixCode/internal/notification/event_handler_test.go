package notification

import (
	"context"
	"testing"
	"time"

	"dev.helix.code/internal/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEventNotificationHandler(t *testing.T) {
	engine := NewNotificationEngine()
	handler := NewEventNotificationHandler(engine)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.engine)
}

func TestEventNotificationHandler_HandleEvent_TaskCompleted(t *testing.T) {
	engine := NewNotificationEngine()
	handler := NewEventNotificationHandler(engine)

	// Create a mock channel to capture notifications
	var capturedNotification *Notification
	mockChannel := &mockChannelCapture{
		onSend: func(ctx context.Context, notif *Notification) error {
			capturedNotification = notif
			return nil
		},
	}
	engine.RegisterChannel(mockChannel)

	// Create rule to send all notifications to mock channel
	rule := NotificationRule{
		Name:      "Capture All",
		Condition: "type==success",
		Channels:  []string{"mock"},
		Enabled:   true,
	}
	engine.AddRule(rule)

	// Create task completed event
	evt := event.Event{
		Type:     event.EventTaskCompleted,
		Severity: event.SeverityInfo,
		Source:   "task_manager",
		Data: map[string]interface{}{
			"task_id":  "task-123",
			"duration": "2m30s",
		},
		TaskID:    "task-123",
		ProjectID: "project-456",
		UserID:    "user-789",
	}

	err := handler.HandleEvent(context.Background(), evt)
	require.NoError(t, err)
	require.NotNil(t, capturedNotification)

	assert.Equal(t, "Task Completed", capturedNotification.Title)
	assert.Contains(t, capturedNotification.Message, "task-123")
	assert.Contains(t, capturedNotification.Message, "2m30s")
	assert.Equal(t, NotificationTypeSuccess, capturedNotification.Type)
	assert.Equal(t, NotificationPriorityLow, capturedNotification.Priority)
	assert.Equal(t, "task-123", capturedNotification.Metadata["task_id"])
	assert.Equal(t, "project-456", capturedNotification.Metadata["project_id"])
}

func TestEventNotificationHandler_HandleEvent_TaskFailed(t *testing.T) {
	engine := NewNotificationEngine()
	handler := NewEventNotificationHandler(engine)

	var capturedNotification *Notification
	mockChannel := &mockChannelCapture{
		onSend: func(ctx context.Context, notif *Notification) error {
			capturedNotification = notif
			return nil
		},
	}
	engine.RegisterChannel(mockChannel)

	rule := NotificationRule{
		Name:      "Capture All",
		Condition: "type==error",
		Channels:  []string{"mock"},
		Enabled:   true,
	}
	engine.AddRule(rule)

	evt := event.Event{
		Type:     event.EventTaskFailed,
		Severity: event.SeverityError,
		Source:   "task_manager",
		Data: map[string]interface{}{
			"task_id": "task-456",
			"error":   "Connection timeout",
		},
		TaskID: "task-456",
	}

	err := handler.HandleEvent(context.Background(), evt)
	require.NoError(t, err)
	require.NotNil(t, capturedNotification)

	assert.Equal(t, "Task Failed", capturedNotification.Title)
	assert.Contains(t, capturedNotification.Message, "task-456")
	assert.Contains(t, capturedNotification.Message, "Connection timeout")
	assert.Equal(t, NotificationTypeError, capturedNotification.Type)
	assert.Equal(t, NotificationPriorityHigh, capturedNotification.Priority)
}

func TestEventNotificationHandler_HandleEvent_WorkflowCompleted(t *testing.T) {
	engine := NewNotificationEngine()
	handler := NewEventNotificationHandler(engine)

	var capturedNotification *Notification
	mockChannel := &mockChannelCapture{
		onSend: func(ctx context.Context, notif *Notification) error {
			capturedNotification = notif
			return nil
		},
	}
	engine.RegisterChannel(mockChannel)

	rule := NotificationRule{
		Name:      "Capture All",
		Condition: "type==success",
		Channels:  []string{"mock"},
		Enabled:   true,
	}
	engine.AddRule(rule)

	evt := event.Event{
		Type:     event.EventWorkflowCompleted,
		Severity: event.SeverityInfo,
		Source:   "workflow_engine",
		Data: map[string]interface{}{
			"workflow_id":   "workflow-789",
			"workflow_name": "Build and Deploy",
		},
	}

	err := handler.HandleEvent(context.Background(), evt)
	require.NoError(t, err)
	require.NotNil(t, capturedNotification)

	assert.Equal(t, "Workflow Completed", capturedNotification.Title)
	assert.Contains(t, capturedNotification.Message, "Build and Deploy")
	assert.Equal(t, NotificationTypeSuccess, capturedNotification.Type)
	assert.Equal(t, NotificationPriorityMedium, capturedNotification.Priority)
}

func TestEventNotificationHandler_HandleEvent_WorkflowFailed(t *testing.T) {
	engine := NewNotificationEngine()
	handler := NewEventNotificationHandler(engine)

	var capturedNotification *Notification
	mockChannel := &mockChannelCapture{
		onSend: func(ctx context.Context, notif *Notification) error {
			capturedNotification = notif
			return nil
		},
	}
	engine.RegisterChannel(mockChannel)

	rule := NotificationRule{
		Name:      "Capture All",
		Condition: "type==error",
		Channels:  []string{"mock"},
		Enabled:   true,
	}
	engine.AddRule(rule)

	evt := event.Event{
		Type:     event.EventWorkflowFailed,
		Severity: event.SeverityError,
		Source:   "workflow_engine",
		Data: map[string]interface{}{
			"workflow_id":   "workflow-789",
			"workflow_name": "Build and Deploy",
			"error":         "Build failed with exit code 1",
		},
	}

	err := handler.HandleEvent(context.Background(), evt)
	require.NoError(t, err)
	require.NotNil(t, capturedNotification)

	assert.Equal(t, "Workflow Failed", capturedNotification.Title)
	assert.Contains(t, capturedNotification.Message, "Build and Deploy")
	assert.Contains(t, capturedNotification.Message, "exit code 1")
	assert.Equal(t, NotificationTypeError, capturedNotification.Type)
}

func TestEventNotificationHandler_HandleEvent_WorkerDisconnected(t *testing.T) {
	engine := NewNotificationEngine()
	handler := NewEventNotificationHandler(engine)

	var capturedNotification *Notification
	mockChannel := &mockChannelCapture{
		onSend: func(ctx context.Context, notif *Notification) error {
			capturedNotification = notif
			return nil
		},
	}
	engine.RegisterChannel(mockChannel)

	rule := NotificationRule{
		Name:      "Capture All",
		Condition: "type==warning",
		Channels:  []string{"mock"},
		Enabled:   true,
	}
	engine.AddRule(rule)

	evt := event.Event{
		Type:     event.EventWorkerDisconnected,
		Severity: event.SeverityWarning,
		Source:   "worker_pool",
		Data: map[string]interface{}{
			"worker_id": "worker-001",
			"host":      "worker-01.example.com",
			"reason":    "SSH connection lost",
		},
		WorkerID: "worker-001",
	}

	err := handler.HandleEvent(context.Background(), evt)
	require.NoError(t, err)
	require.NotNil(t, capturedNotification)

	assert.Equal(t, "Worker Disconnected", capturedNotification.Title)
	assert.Contains(t, capturedNotification.Message, "worker-001")
	assert.Contains(t, capturedNotification.Message, "worker-01.example.com")
	assert.Contains(t, capturedNotification.Message, "SSH connection lost")
	assert.Equal(t, NotificationTypeWarning, capturedNotification.Type)
}

func TestEventNotificationHandler_HandleEvent_SystemError(t *testing.T) {
	engine := NewNotificationEngine()
	handler := NewEventNotificationHandler(engine)

	var capturedNotification *Notification
	mockChannel := &mockChannelCapture{
		onSend: func(ctx context.Context, notif *Notification) error {
			capturedNotification = notif
			return nil
		},
	}
	engine.RegisterChannel(mockChannel)

	rule := NotificationRule{
		Name:      "Capture All",
		Condition: "type==error",
		Channels:  []string{"mock"},
		Enabled:   true,
	}
	engine.AddRule(rule)

	evt := event.Event{
		Type:     event.EventSystemError,
		Severity: event.SeverityCritical,
		Source:   "database",
		Data: map[string]interface{}{
			"component": "database",
			"error":     "Connection pool exhausted",
		},
	}

	err := handler.HandleEvent(context.Background(), evt)
	require.NoError(t, err)
	require.NotNil(t, capturedNotification)

	assert.Equal(t, "System Error", capturedNotification.Title)
	assert.Contains(t, capturedNotification.Message, "database")
	assert.Contains(t, capturedNotification.Message, "Connection pool exhausted")
	assert.Equal(t, NotificationTypeError, capturedNotification.Type)
	assert.Equal(t, NotificationPriorityUrgent, capturedNotification.Priority)
}

func TestEventNotificationHandler_HandleEvent_SystemStartup(t *testing.T) {
	engine := NewNotificationEngine()
	handler := NewEventNotificationHandler(engine)

	var capturedNotification *Notification
	mockChannel := &mockChannelCapture{
		onSend: func(ctx context.Context, notif *Notification) error {
			capturedNotification = notif
			return nil
		},
	}
	engine.RegisterChannel(mockChannel)

	rule := NotificationRule{
		Name:      "Capture All",
		Condition: "type==info",
		Channels:  []string{"mock"},
		Enabled:   true,
	}
	engine.AddRule(rule)

	evt := event.Event{
		Type:     event.EventSystemStartup,
		Severity: event.SeverityInfo,
		Source:   "system",
		Data: map[string]interface{}{
			"version": "1.0.0",
		},
	}

	err := handler.HandleEvent(context.Background(), evt)
	require.NoError(t, err)
	require.NotNil(t, capturedNotification)

	assert.Equal(t, "System Started", capturedNotification.Title)
	assert.Contains(t, capturedNotification.Message, "1.0.0")
	assert.Equal(t, NotificationTypeInfo, capturedNotification.Type)
}

func TestEventNotificationHandler_HandleEvent_TaskStartedIgnored(t *testing.T) {
	engine := NewNotificationEngine()
	handler := NewEventNotificationHandler(engine)

	// Task started events should be ignored (return nil notification)
	evt := event.Event{
		Type:     event.EventTaskStarted,
		Severity: event.SeverityInfo,
		Source:   "task_manager",
		Data: map[string]interface{}{
			"task_id": "task-999",
		},
	}

	err := handler.HandleEvent(context.Background(), evt)
	require.NoError(t, err)
	// No notification should be sent for TaskStarted
}

func TestEventNotificationHandler_RegisterWithEventBus(t *testing.T) {
	engine := NewNotificationEngine()
	handler := NewEventNotificationHandler(engine)

	bus := event.NewEventBus(false)
	handler.RegisterWithEventBus(bus)

	// Verify subscriptions
	assert.Greater(t, bus.GetSubscriberCount(event.EventTaskCompleted), 0)
	assert.Greater(t, bus.GetSubscriberCount(event.EventTaskFailed), 0)
	assert.Greater(t, bus.GetSubscriberCount(event.EventWorkflowCompleted), 0)
	assert.Greater(t, bus.GetSubscriberCount(event.EventWorkflowFailed), 0)
	assert.Greater(t, bus.GetSubscriberCount(event.EventWorkerDisconnected), 0)
	assert.Greater(t, bus.GetSubscriberCount(event.EventSystemError), 0)
}

func TestEventNotificationHandler_EndToEnd(t *testing.T) {
	// Create event bus
	bus := event.NewEventBus(false)

	// Create notification engine
	engine := NewNotificationEngine()

	// Create and register event handler
	eventHandler := NewEventNotificationHandler(engine)
	eventHandler.RegisterWithEventBus(bus)

	// Create mock channel to capture notifications
	var notifications []*Notification
	mockChannel := &mockChannelCapture{
		onSend: func(ctx context.Context, notif *Notification) error {
			notifications = append(notifications, notif)
			return nil
		},
	}
	engine.RegisterChannel(mockChannel)

	// Add rule to send error notifications
	rule := NotificationRule{
		Name:      "Error Alerts",
		Condition: "type==error",
		Channels:  []string{"mock"},
		Priority:  NotificationPriorityHigh,
		Enabled:   true,
	}
	engine.AddRule(rule)

	// Publish task failed event
	evt := event.Event{
		ID:        "evt-123",
		Type:      event.EventTaskFailed,
		Timestamp: time.Now(),
		Severity:  event.SeverityError,
		Source:    "task_manager",
		Data: map[string]interface{}{
			"task_id": "task-999",
			"error":   "Unexpected error",
		},
		TaskID: "task-999",
	}

	err := bus.Publish(context.Background(), evt)
	require.NoError(t, err)

	// Verify notification was sent
	assert.Equal(t, 1, len(notifications))
	assert.Equal(t, "Task Failed", notifications[0].Title)
	assert.Contains(t, notifications[0].Message, "task-999")
	assert.Contains(t, notifications[0].Message, "Unexpected error")
}

// Mock channel for capturing notifications
type mockChannelCapture struct {
	onSend func(ctx context.Context, notif *Notification) error
}

func (m *mockChannelCapture) Send(ctx context.Context, notification *Notification) error {
	if m.onSend != nil {
		return m.onSend(ctx, notification)
	}
	return nil
}

func (m *mockChannelCapture) GetName() string {
	return "mock"
}

func (m *mockChannelCapture) IsEnabled() bool {
	return true
}

func (m *mockChannelCapture) GetConfig() map[string]interface{} {
	return map[string]interface{}{}
}
