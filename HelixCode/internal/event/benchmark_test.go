package event

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

// Benchmark event bus operations

func BenchmarkEventBus_Subscribe(b *testing.B) {
	bus := NewEventBus(false)
	handler := func(ctx context.Context, evt Event) error {
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Subscribe(EventTaskCreated, handler)
	}
}

func BenchmarkEventBus_Unsubscribe(b *testing.B) {
	bus := NewEventBus(false)
	handler := func(ctx context.Context, evt Event) error {
		return nil
	}

	// Subscribe handlers
	for i := 0; i < b.N; i++ {
		bus.Subscribe(EventTaskCreated, handler)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Unsubscribe(EventTaskCreated)
	}
}

func BenchmarkEventBus_PublishSync(b *testing.B) {
	bus := NewEventBus(false)

	bus.Subscribe(EventTaskCreated, func(ctx context.Context, evt Event) error {
		return nil
	})

	event := Event{
		Type:     EventTaskCreated,
		Severity: SeverityInfo,
		Source:   "test",
		TaskID:   "task-123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(context.Background(), event)
	}
}

func BenchmarkEventBus_PublishAsync(b *testing.B) {
	bus := NewEventBus(true)

	bus.Subscribe(EventTaskCreated, func(ctx context.Context, evt Event) error {
		return nil
	})

	event := Event{
		Type:     EventTaskCreated,
		Severity: SeverityInfo,
		Source:   "test",
		TaskID:   "task-123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(context.Background(), event)
	}
}

func BenchmarkEventBus_PublishNoSubscribers(b *testing.B) {
	bus := NewEventBus(false)

	event := Event{
		Type:     EventTaskCreated,
		Severity: SeverityInfo,
		Source:   "test",
		TaskID:   "task-123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(context.Background(), event)
	}
}

func BenchmarkEventBus_PublishMultipleSubscribers(b *testing.B) {
	bus := NewEventBus(false)

	// Subscribe 10 handlers
	for i := 0; i < 10; i++ {
		bus.Subscribe(EventTaskCreated, func(ctx context.Context, evt Event) error {
			return nil
		})
	}

	event := Event{
		Type:     EventTaskCreated,
		Severity: SeverityInfo,
		Source:   "test",
		TaskID:   "task-123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(context.Background(), event)
	}
}

// Benchmark concurrent operations

func BenchmarkEventBus_ConcurrentPublish(b *testing.B) {
	bus := NewEventBus(true)

	bus.Subscribe(EventTaskCreated, func(ctx context.Context, evt Event) error {
		return nil
	})

	event := Event{
		Type:     EventTaskCreated,
		Severity: SeverityInfo,
		Source:   "test",
		TaskID:   "task-123",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = bus.Publish(context.Background(), event)
		}
	})
}

func BenchmarkEventBus_ConcurrentSubscribe(b *testing.B) {
	bus := NewEventBus(false)
	handler := func(ctx context.Context, evt Event) error {
		return nil
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bus.Subscribe(EventTaskCreated, handler)
		}
	})
}

func BenchmarkEventBus_ConcurrentMixed(b *testing.B) {
	bus := NewEventBus(true)

	handler := func(ctx context.Context, evt Event) error {
		return nil
	}

	event := Event{
		Type:     EventTaskCreated,
		Severity: SeverityInfo,
		Source:   "test",
		TaskID:   "task-123",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Mix of operations
			bus.Subscribe(EventTaskCreated, handler)
			_ = bus.Publish(context.Background(), event)
		}
	})
}

// Benchmark different event types

func BenchmarkEventBus_TaskEvents(b *testing.B) {
	bus := NewEventBus(false)

	bus.Subscribe(EventTaskCreated, func(ctx context.Context, evt Event) error {
		return nil
	})

	event := Event{
		Type:     EventTaskCreated,
		Severity: SeverityInfo,
		Source:   "task_manager",
		TaskID:   "task-123",
		Data: map[string]interface{}{
			"task_type": "build",
			"priority":  "high",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(context.Background(), event)
	}
}

func BenchmarkEventBus_WorkflowEvents(b *testing.B) {
	bus := NewEventBus(false)

	bus.Subscribe(EventWorkflowStarted, func(ctx context.Context, evt Event) error {
		return nil
	})

	event := Event{
		Type:     EventWorkflowStarted,
		Severity: SeverityInfo,
		Source:   "workflow_engine",
		Data: map[string]interface{}{
			"workflow_id":   "workflow-123",
			"workflow_name": "build_and_test",
			"steps":         5,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(context.Background(), event)
	}
}

func BenchmarkEventBus_WorkerEvents(b *testing.B) {
	bus := NewEventBus(false)

	bus.Subscribe(EventWorkerConnected, func(ctx context.Context, evt Event) error {
		return nil
	})

	event := Event{
		Type:     EventWorkerConnected,
		Severity: SeverityInfo,
		Source:   "worker_pool",
		WorkerID: "worker-123",
		Data: map[string]interface{}{
			"hostname": "worker-node-1",
			"capacity": 10,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(context.Background(), event)
	}
}

func BenchmarkEventBus_SystemEvents(b *testing.B) {
	bus := NewEventBus(false)

	bus.Subscribe(EventSystemError, func(ctx context.Context, evt Event) error {
		return nil
	})

	event := Event{
		Type:     EventSystemError,
		Severity: SeverityError,
		Source:   "system",
		Data: map[string]interface{}{
			"error":     "Connection timeout",
			"component": "database",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(context.Background(), event)
	}
}

// Benchmark with different payload sizes

func BenchmarkEventBus_SmallPayload(b *testing.B) {
	bus := NewEventBus(false)

	bus.Subscribe(EventTaskCreated, func(ctx context.Context, evt Event) error {
		return nil
	})

	event := Event{
		Type:     EventTaskCreated,
		Severity: SeverityInfo,
		Source:   "test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(context.Background(), event)
	}
}

func BenchmarkEventBus_MediumPayload(b *testing.B) {
	bus := NewEventBus(false)

	bus.Subscribe(EventTaskCreated, func(ctx context.Context, evt Event) error {
		return nil
	})

	event := Event{
		Type:     EventTaskCreated,
		Severity: SeverityInfo,
		Source:   "test",
		TaskID:   "task-123",
		Data: map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
			"key4": "value4",
			"key5": "value5",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(context.Background(), event)
	}
}

func BenchmarkEventBus_LargePayload(b *testing.B) {
	bus := NewEventBus(false)

	bus.Subscribe(EventTaskCreated, func(ctx context.Context, evt Event) error {
		return nil
	})

	data := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		data[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d with some additional data", i)
	}

	event := Event{
		Type:     EventTaskCreated,
		Severity: SeverityInfo,
		Source:   "test",
		TaskID:   "task-123",
		Data:     data,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(context.Background(), event)
	}
}

// Benchmark global bus

func BenchmarkGlobalBus_Publish(b *testing.B) {
	bus := GetGlobalBus()

	bus.Subscribe(EventTaskCreated, func(ctx context.Context, evt Event) error {
		return nil
	})

	event := Event{
		Type:     EventTaskCreated,
		Severity: SeverityInfo,
		Source:   "test",
		TaskID:   "task-123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(context.Background(), event)
	}
}

// Benchmark memory allocations

func BenchmarkEvent_Creation(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Event{
			Type:     EventTaskCreated,
			Severity: SeverityInfo,
			Source:   "test",
			TaskID:   "task-123",
			Data: map[string]interface{}{
				"key": "value",
			},
		}
	}
}

func BenchmarkEventBus_PublishAllocations(b *testing.B) {
	bus := NewEventBus(false)

	bus.Subscribe(EventTaskCreated, func(ctx context.Context, evt Event) error {
		return nil
	})

	event := Event{
		Type:     EventTaskCreated,
		Severity: SeverityInfo,
		Source:   "test",
		TaskID:   "task-123",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = bus.Publish(context.Background(), event)
	}
}

// Benchmark stress scenarios

func BenchmarkStress_HighVolumePublish(b *testing.B) {
	bus := NewEventBus(true)

	bus.Subscribe(EventTaskCreated, func(ctx context.Context, evt Event) error {
		return nil
	})

	event := Event{
		Type:     EventTaskCreated,
		Severity: SeverityInfo,
		Source:   "test",
		TaskID:   "task-123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100; j++ {
			_ = bus.Publish(context.Background(), event)
		}
	}
}

func BenchmarkStress_ManySubscribers(b *testing.B) {
	bus := NewEventBus(false)

	// Subscribe 100 handlers
	for i := 0; i < 100; i++ {
		bus.Subscribe(EventTaskCreated, func(ctx context.Context, evt Event) error {
			return nil
		})
	}

	event := Event{
		Type:     EventTaskCreated,
		Severity: SeverityInfo,
		Source:   "test",
		TaskID:   "task-123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(context.Background(), event)
	}
}

func BenchmarkStress_ConcurrentSubscribePublish(b *testing.B) {
	bus := NewEventBus(true)

	handler := func(ctx context.Context, evt Event) error {
		return nil
	}

	event := Event{
		Type:     EventTaskCreated,
		Severity: SeverityInfo,
		Source:   "test",
		TaskID:   "task-123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < 50; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				bus.Subscribe(EventTaskCreated, handler)
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = bus.Publish(context.Background(), event)
			}()
		}
		wg.Wait()
	}
}

// Benchmark all event types

func BenchmarkEventBus_AllTaskEventTypes(b *testing.B) {
	bus := NewEventBus(false)

	eventTypes := []EventType{
		EventTaskCreated,
		EventTaskAssigned,
		EventTaskStarted,
		EventTaskCompleted,
		EventTaskFailed,
		EventTaskPaused,
		EventTaskResumed,
		EventTaskCancelled,
	}

	for _, eventType := range eventTypes {
		bus.Subscribe(eventType, func(ctx context.Context, evt Event) error {
			return nil
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, eventType := range eventTypes {
			event := Event{
				Type:     eventType,
				Severity: SeverityInfo,
				Source:   "test",
				TaskID:   "task-123",
			}
			_ = bus.Publish(context.Background(), event)
		}
	}
}

// Benchmark error handling

func BenchmarkEventBus_PublishWithErrors(b *testing.B) {
	bus := NewEventBus(false)

	bus.Subscribe(EventTaskCreated, func(ctx context.Context, evt Event) error {
		return fmt.Errorf("handler error")
	})

	event := Event{
		Type:     EventTaskCreated,
		Severity: SeverityInfo,
		Source:   "test",
		TaskID:   "task-123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(context.Background(), event)
	}
}

func BenchmarkEventBus_GetErrors(b *testing.B) {
	bus := NewEventBus(false)

	bus.Subscribe(EventTaskCreated, func(ctx context.Context, evt Event) error {
		return fmt.Errorf("handler error")
	})

	event := Event{
		Type:     EventTaskCreated,
		Severity: SeverityInfo,
		Source:   "test",
		TaskID:   "task-123",
	}

	// Generate some errors
	for i := 0; i < 10; i++ {
		_ = bus.Publish(context.Background(), event)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.GetErrors()
	}
}

// Benchmark comparison: sync vs async

func BenchmarkComparison_SyncVsAsync_Sync(b *testing.B) {
	bus := NewEventBus(false)

	bus.Subscribe(EventTaskCreated, func(ctx context.Context, evt Event) error {
		return nil
	})

	event := Event{
		Type:     EventTaskCreated,
		Severity: SeverityInfo,
		Source:   "test",
		TaskID:   "task-123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(context.Background(), event)
	}
}

func BenchmarkComparison_SyncVsAsync_Async(b *testing.B) {
	bus := NewEventBus(true)

	bus.Subscribe(EventTaskCreated, func(ctx context.Context, evt Event) error {
		return nil
	})

	event := Event{
		Type:     EventTaskCreated,
		Severity: SeverityInfo,
		Source:   "test",
		TaskID:   "task-123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(context.Background(), event)
	}
}
