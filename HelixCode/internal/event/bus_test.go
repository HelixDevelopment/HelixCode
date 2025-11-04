package event

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEventBus(t *testing.T) {
	tests := []struct {
		name      string
		async     bool
		wantAsync bool
	}{
		{
			name:      "sync mode",
			async:     false,
			wantAsync: false,
		},
		{
			name:      "async mode",
			async:     true,
			wantAsync: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := NewEventBus(tt.async)
			assert.NotNil(t, bus)
			assert.Equal(t, tt.wantAsync, bus.IsAsync())
			assert.Equal(t, 0, bus.GetTotalSubscribers())
		})
	}
}

func TestEventBus_Subscribe(t *testing.T) {
	bus := NewEventBus(false)

	called := false
	handler := func(ctx context.Context, event Event) error {
		called = true
		return nil
	}

	bus.Subscribe(EventTaskCompleted, handler)

	assert.Equal(t, 1, bus.GetSubscriberCount(EventTaskCompleted))
	assert.Equal(t, 1, bus.GetTotalSubscribers())

	// Publish event
	event := Event{
		Type:     EventTaskCompleted,
		Source:   "test",
		Severity: SeverityInfo,
	}

	err := bus.Publish(context.Background(), event)
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestEventBus_SubscribeMultiple(t *testing.T) {
	bus := NewEventBus(false)

	callCount := int32(0)
	handler := func(ctx context.Context, event Event) error {
		atomic.AddInt32(&callCount, 1)
		return nil
	}

	eventTypes := []EventType{
		EventTaskCompleted,
		EventTaskFailed,
		EventWorkflowCompleted,
	}

	bus.SubscribeMultiple(eventTypes, handler)

	assert.Equal(t, 3, bus.GetTotalSubscribers())

	// Publish to each event type
	for _, eventType := range eventTypes {
		event := Event{
			Type:     eventType,
			Source:   "test",
			Severity: SeverityInfo,
		}
		bus.Publish(context.Background(), event)
	}

	assert.Equal(t, int32(3), atomic.LoadInt32(&callCount))
}

func TestEventBus_Unsubscribe(t *testing.T) {
	bus := NewEventBus(false)

	handler := func(ctx context.Context, event Event) error {
		return nil
	}

	bus.Subscribe(EventTaskCompleted, handler)
	assert.Equal(t, 1, bus.GetSubscriberCount(EventTaskCompleted))

	bus.Unsubscribe(EventTaskCompleted)
	assert.Equal(t, 0, bus.GetSubscriberCount(EventTaskCompleted))
}

func TestEventBus_MultipleHandlers(t *testing.T) {
	bus := NewEventBus(false)

	callCount := int32(0)
	handler1 := func(ctx context.Context, event Event) error {
		atomic.AddInt32(&callCount, 1)
		return nil
	}
	handler2 := func(ctx context.Context, event Event) error {
		atomic.AddInt32(&callCount, 10)
		return nil
	}

	bus.Subscribe(EventTaskFailed, handler1)
	bus.Subscribe(EventTaskFailed, handler2)

	event := Event{
		Type:     EventTaskFailed,
		Severity: SeverityError,
	}

	err := bus.Publish(context.Background(), event)
	assert.NoError(t, err)
	assert.Equal(t, int32(11), atomic.LoadInt32(&callCount)) // 1 + 10
}

func TestEventBus_NoSubscribers(t *testing.T) {
	bus := NewEventBus(false)

	event := Event{
		Type:     EventTaskCompleted,
		Severity: SeverityInfo,
	}

	err := bus.Publish(context.Background(), event)
	assert.NoError(t, err) // Should not error
}

func TestEventBus_AsyncMode(t *testing.T) {
	bus := NewEventBus(true)

	var wg sync.WaitGroup
	wg.Add(1)

	called := false
	handler := func(ctx context.Context, event Event) error {
		time.Sleep(100 * time.Millisecond)
		called = true
		wg.Done()
		return nil
	}

	bus.Subscribe(EventTaskCompleted, handler)

	event := Event{
		Type:     EventTaskCompleted,
		Severity: SeverityInfo,
	}

	start := time.Now()
	err := bus.Publish(context.Background(), event)
	duration := time.Since(start)

	assert.NoError(t, err)
	// Publish should return immediately in async mode
	assert.Less(t, duration, 50*time.Millisecond)

	// Wait for handler to complete
	wg.Wait()
	assert.True(t, called)
}

func TestEventBus_PublishAndWait(t *testing.T) {
	bus := NewEventBus(true)

	called := false
	handler := func(ctx context.Context, event Event) error {
		time.Sleep(100 * time.Millisecond)
		called = true
		return nil
	}

	bus.Subscribe(EventTaskCompleted, handler)

	event := Event{
		Type:     EventTaskCompleted,
		Severity: SeverityInfo,
	}

	start := time.Now()
	err := bus.PublishAndWait(context.Background(), event)
	duration := time.Since(start)

	assert.NoError(t, err)
	// Should wait for handler to complete
	assert.GreaterOrEqual(t, duration, 100*time.Millisecond)
	assert.True(t, called)
}

func TestEventBus_SyncModeErrors(t *testing.T) {
	bus := NewEventBus(false)

	handler1 := func(ctx context.Context, event Event) error {
		return errors.New("handler1 error")
	}
	handler2 := func(ctx context.Context, event Event) error {
		return nil // This should still run
	}
	handler3 := func(ctx context.Context, event Event) error {
		return errors.New("handler3 error")
	}

	bus.Subscribe(EventTaskFailed, handler1)
	bus.Subscribe(EventTaskFailed, handler2)
	bus.Subscribe(EventTaskFailed, handler3)

	event := Event{
		Type:     EventTaskFailed,
		Severity: SeverityError,
	}

	err := bus.Publish(context.Background(), event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "handler1 error")
	assert.Contains(t, err.Error(), "handler3 error")

	// Verify errors were logged
	errors := bus.GetErrors()
	assert.GreaterOrEqual(t, len(errors), 2)
}

func TestEventBus_AsyncModeErrors(t *testing.T) {
	bus := NewEventBus(true)

	var wg sync.WaitGroup
	wg.Add(1)

	handler := func(ctx context.Context, event Event) error {
		defer wg.Done()
		return errors.New("async handler error")
	}

	bus.Subscribe(EventTaskFailed, handler)

	event := Event{
		Type:     EventTaskFailed,
		Severity: SeverityError,
	}

	err := bus.Publish(context.Background(), event)
	assert.NoError(t, err) // Async mode doesn't return errors

	wg.Wait()
	time.Sleep(50 * time.Millisecond) // Allow error logging

	// Verify error was logged
	errors := bus.GetErrors()
	assert.GreaterOrEqual(t, len(errors), 1)
}

func TestEventBus_EventIDAndTimestamp(t *testing.T) {
	bus := NewEventBus(false)

	var receivedEvent Event
	handler := func(ctx context.Context, event Event) error {
		receivedEvent = event
		return nil
	}

	bus.Subscribe(EventTaskCompleted, handler)

	event := Event{
		Type:     EventTaskCompleted,
		Source:   "test",
		Severity: SeverityInfo,
		// No ID or Timestamp set
	}

	err := bus.Publish(context.Background(), event)
	require.NoError(t, err)

	// Verify ID and Timestamp were set
	assert.NotEmpty(t, receivedEvent.ID)
	assert.False(t, receivedEvent.Timestamp.IsZero())
}

func TestEventBus_EventData(t *testing.T) {
	bus := NewEventBus(false)

	var receivedEvent Event
	handler := func(ctx context.Context, event Event) error {
		receivedEvent = event
		return nil
	}

	bus.Subscribe(EventTaskCompleted, handler)

	event := Event{
		Type:     EventTaskCompleted,
		Source:   "task_manager",
		Severity: SeverityInfo,
		Data: map[string]interface{}{
			"task_id":  "task-123",
			"duration": "2m30s",
			"result":   "success",
		},
		UserID:    "user-456",
		ProjectID: "project-789",
		TaskID:    "task-123",
	}

	err := bus.Publish(context.Background(), event)
	require.NoError(t, err)

	assert.Equal(t, "task_manager", receivedEvent.Source)
	assert.Equal(t, "task-123", receivedEvent.Data["task_id"])
	assert.Equal(t, "user-456", receivedEvent.UserID)
	assert.Equal(t, "project-789", receivedEvent.ProjectID)
	assert.Equal(t, "task-123", receivedEvent.TaskID)
}

func TestEventBus_GetSubscribedEvents(t *testing.T) {
	bus := NewEventBus(false)

	handler := func(ctx context.Context, event Event) error {
		return nil
	}

	bus.Subscribe(EventTaskCompleted, handler)
	bus.Subscribe(EventTaskFailed, handler)
	bus.Subscribe(EventWorkflowStarted, handler)

	events := bus.GetSubscribedEvents()
	assert.Equal(t, 3, len(events))
	assert.Contains(t, events, EventTaskCompleted)
	assert.Contains(t, events, EventTaskFailed)
	assert.Contains(t, events, EventWorkflowStarted)
}

func TestEventBus_ClearErrors(t *testing.T) {
	bus := NewEventBus(false)

	handler := func(ctx context.Context, event Event) error {
		return errors.New("test error")
	}

	bus.Subscribe(EventTaskFailed, handler)

	event := Event{
		Type:     EventTaskFailed,
		Severity: SeverityError,
	}

	bus.Publish(context.Background(), event)

	errors := bus.GetErrors()
	assert.Greater(t, len(errors), 0)

	bus.ClearErrors()

	errors = bus.GetErrors()
	assert.Equal(t, 0, len(errors))
}

func TestEventBus_ConcurrentPublish(t *testing.T) {
	bus := NewEventBus(false)

	callCount := int32(0)
	handler := func(ctx context.Context, event Event) error {
		atomic.AddInt32(&callCount, 1)
		return nil
	}

	bus.Subscribe(EventTaskCompleted, handler)

	var wg sync.WaitGroup
	count := 100

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			event := Event{
				Type:     EventTaskCompleted,
				Source:   "test",
				Severity: SeverityInfo,
			}
			bus.Publish(context.Background(), event)
		}()
	}

	wg.Wait()
	assert.Equal(t, int32(count), atomic.LoadInt32(&callCount))
}

func TestEventBus_ConcurrentSubscribe(t *testing.T) {
	bus := NewEventBus(false)

	var wg sync.WaitGroup
	count := 50

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			handler := func(ctx context.Context, event Event) error {
				return nil
			}
			bus.Subscribe(EventTaskCompleted, handler)
		}()
	}

	wg.Wait()
	assert.Equal(t, count, bus.GetSubscriberCount(EventTaskCompleted))
}

func TestEventBus_AllEventTypes(t *testing.T) {
	bus := NewEventBus(false)

	receivedEvents := make(map[EventType]bool)
	var mu sync.Mutex

	handler := func(ctx context.Context, event Event) error {
		mu.Lock()
		defer mu.Unlock()
		receivedEvents[event.Type] = true
		return nil
	}

	// Subscribe to all event types
	allEvents := []EventType{
		EventTaskCreated, EventTaskAssigned, EventTaskStarted,
		EventTaskCompleted, EventTaskFailed, EventTaskPaused,
		EventWorkflowStarted, EventWorkflowCompleted, EventWorkflowFailed,
		EventWorkerConnected, EventWorkerDisconnected,
		EventUserRegistered, EventProjectCreated,
		EventSystemStartup, EventSystemError,
	}

	for _, eventType := range allEvents {
		bus.Subscribe(eventType, handler)
	}

	// Publish all event types
	for _, eventType := range allEvents {
		event := Event{
			Type:     eventType,
			Source:   "test",
			Severity: SeverityInfo,
		}
		bus.Publish(context.Background(), event)
	}

	assert.Equal(t, len(allEvents), len(receivedEvents))
}

func TestEventBus_ErrorLogLimit(t *testing.T) {
	bus := NewEventBus(false)

	handler := func(ctx context.Context, event Event) error {
		return errors.New("test error")
	}

	bus.Subscribe(EventTaskFailed, handler)

	event := Event{
		Type:     EventTaskFailed,
		Severity: SeverityError,
	}

	// Generate more than 100 errors
	for i := 0; i < 150; i++ {
		bus.Publish(context.Background(), event)
	}

	errors := bus.GetErrors()
	// Should keep only last 100
	assert.LessOrEqual(t, len(errors), 100)
}
