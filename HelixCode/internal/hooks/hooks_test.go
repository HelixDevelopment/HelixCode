package hooks

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestHook tests basic hook functionality
func TestHook(t *testing.T) {
	t.Run("create_hook", func(t *testing.T) {
		handler := func(ctx context.Context, event *Event) error {
			return nil
		}

		hook := NewHook("test-hook", HookTypeBeforeTask, handler)

		if hook.ID == "" {
			t.Error("hook ID should not be empty")
		}

		if hook.Name != "test-hook" {
			t.Errorf("expected name 'test-hook', got %s", hook.Name)
		}

		if hook.Type != HookTypeBeforeTask {
			t.Errorf("expected type before_task, got %s", hook.Type)
		}

		if hook.Priority != PriorityNormal {
			t.Errorf("expected normal priority, got %d", hook.Priority)
		}

		if hook.Async {
			t.Error("hook should not be async by default")
		}

		if !hook.Enabled {
			t.Error("hook should be enabled by default")
		}
	})

	t.Run("create_async_hook", func(t *testing.T) {
		handler := func(ctx context.Context, event *Event) error {
			return nil
		}

		hook := NewAsyncHook("async-hook", HookTypeAfterTask, handler)

		if !hook.Async {
			t.Error("hook should be async")
		}
	})

	t.Run("create_with_priority", func(t *testing.T) {
		handler := func(ctx context.Context, event *Event) error {
			return nil
		}

		hook := NewHookWithPriority("priority-hook", HookTypeBeforeTask, handler, PriorityHigh)

		if hook.Priority != PriorityHigh {
			t.Errorf("expected high priority, got %d", hook.Priority)
		}
	})

	t.Run("validate", func(t *testing.T) {
		handler := func(ctx context.Context, event *Event) error {
			return nil
		}

		hook := NewHook("valid-hook", HookTypeBeforeTask, handler)

		if err := hook.Validate(); err != nil {
			t.Errorf("validation should pass: %v", err)
		}
	})

	t.Run("validate_empty_name", func(t *testing.T) {
		hook := &Hook{
			ID:      "test-id",
			Name:    "",
			Type:    HookTypeBeforeTask,
			Handler: func(ctx context.Context, event *Event) error { return nil },
		}

		if err := hook.Validate(); err == nil {
			t.Error("validation should fail for empty name")
		}
	})

	t.Run("validate_nil_handler", func(t *testing.T) {
		hook := &Hook{
			ID:      "test-id",
			Name:    "test",
			Type:    HookTypeBeforeTask,
			Handler: nil,
		}

		if err := hook.Validate(); err == nil {
			t.Error("validation should fail for nil handler")
		}
	})
}

// TestHookTags tests tag functionality
func TestHookTags(t *testing.T) {
	t.Run("add_tag", func(t *testing.T) {
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, event *Event) error { return nil })

		hook.AddTag("important")

		if !hook.HasTag("important") {
			t.Error("hook should have 'important' tag")
		}
	})

	t.Run("add_duplicate_tag", func(t *testing.T) {
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, event *Event) error { return nil })

		hook.AddTag("test")
		hook.AddTag("test")

		count := 0
		for _, tag := range hook.Tags {
			if tag == "test" {
				count++
			}
		}

		if count != 1 {
			t.Errorf("duplicate tag should not be added, found %d occurrences", count)
		}
	})
}

// TestHookCondition tests conditional execution
func TestHookCondition(t *testing.T) {
	t.Run("condition_true", func(t *testing.T) {
		executed := false
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			executed = true
			return nil
		})

		hook.Condition = func(event *Event) bool {
			return true
		}

		event := NewEvent(HookTypeBeforeTask)
		hook.Execute(context.Background(), event)

		if !executed {
			t.Error("hook should have executed")
		}
	})

	t.Run("condition_false", func(t *testing.T) {
		executed := false
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			executed = true
			return nil
		})

		hook.Condition = func(event *Event) bool {
			return false
		}

		event := NewEvent(HookTypeBeforeTask)
		hook.Execute(context.Background(), event)

		if executed {
			t.Error("hook should not have executed")
		}
	})
}

// TestEvent tests event functionality
func TestEvent(t *testing.T) {
	t.Run("create_event", func(t *testing.T) {
		event := NewEvent(HookTypeBeforeTask)

		if event.Type != HookTypeBeforeTask {
			t.Errorf("expected type before_task, got %s", event.Type)
		}

		if event.Data == nil {
			t.Error("data should be initialized")
		}

		if event.Metadata == nil {
			t.Error("metadata should be initialized")
		}
	})

	t.Run("set_and_get_data", func(t *testing.T) {
		event := NewEvent(HookTypeBeforeTask)

		event.SetData("key", "value")

		value, ok := event.GetData("key")
		if !ok {
			t.Error("data should exist")
		}

		if value != "value" {
			t.Errorf("expected value 'value', got %v", value)
		}
	})

	t.Run("set_and_get_metadata", func(t *testing.T) {
		event := NewEvent(HookTypeBeforeTask)

		event.SetMetadata("author", "test-user")

		value, ok := event.GetMetadata("author")
		if !ok {
			t.Error("metadata should exist")
		}

		if value != "test-user" {
			t.Errorf("expected metadata 'test-user', got %s", value)
		}
	})
}

// TestExecutor tests executor functionality
func TestExecutor(t *testing.T) {
	t.Run("create_executor", func(t *testing.T) {
		executor := NewExecutor()

		if executor == nil {
			t.Error("executor should not be nil")
		}
	})

	t.Run("execute_sync_hook", func(t *testing.T) {
		executor := NewExecutor()
		executed := false

		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			executed = true
			return nil
		})

		event := NewEvent(HookTypeBeforeTask)
		result := executor.Execute(context.Background(), hook, event)

		if !executed {
			t.Error("hook should have been executed")
		}

		if result.Status != StatusCompleted {
			t.Errorf("expected status completed, got %s", result.Status)
		}
	})

	t.Run("execute_async_hook", func(t *testing.T) {
		executor := NewExecutor()
		executed := false

		hook := NewAsyncHook("test", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			executed = true
			return nil
		})

		event := NewEvent(HookTypeBeforeTask)
		executor.Execute(context.Background(), hook, event)

		// Wait for async execution
		executor.Wait()

		if !executed {
			t.Error("async hook should have been executed")
		}
	})

	t.Run("execute_with_error", func(t *testing.T) {
		executor := NewExecutor()
		testError := errors.New("test error")

		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			return testError
		})

		event := NewEvent(HookTypeBeforeTask)
		result := executor.Execute(context.Background(), hook, event)

		if result.Status != StatusFailed {
			t.Errorf("expected status failed, got %s", result.Status)
		}

		if result.Error != testError {
			t.Errorf("expected error %v, got %v", testError, result.Error)
		}
	})

	t.Run("execute_with_timeout", func(t *testing.T) {
		executor := NewExecutor()

		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			// Check context and sleep longer
			select {
			case <-time.After(200 * time.Millisecond):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		hook.Timeout = 10 * time.Millisecond

		event := NewEvent(HookTypeBeforeTask)
		result := executor.Execute(context.Background(), hook, event)

		if result.Status != StatusFailed {
			t.Error("hook should have timed out")
		}

		if result.Error == nil {
			t.Error("should have error for timeout")
		}
	})

	t.Run("execute_all_priority_order", func(t *testing.T) {
		executor := NewExecutor()
		order := []int{}

		hook1 := NewHookWithPriority("low", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			order = append(order, 1)
			return nil
		}, PriorityLow)

		hook2 := NewHookWithPriority("high", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			order = append(order, 2)
			return nil
		}, PriorityHigh)

		hook3 := NewHookWithPriority("normal", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			order = append(order, 3)
			return nil
		}, PriorityNormal)

		event := NewEvent(HookTypeBeforeTask)
		executor.ExecuteAll(context.Background(), []*Hook{hook1, hook2, hook3}, event)

		// Should execute in order: high (2), normal (3), low (1)
		if len(order) != 3 {
			t.Errorf("expected 3 executions, got %d", len(order))
		}

		if order[0] != 2 || order[1] != 3 || order[2] != 1 {
			t.Errorf("wrong execution order: %v", order)
		}
	})

	t.Run("get_statistics", func(t *testing.T) {
		executor := NewExecutor()

		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			return nil
		})

		event := NewEvent(HookTypeBeforeTask)
		executor.Execute(context.Background(), hook, event)

		stats := executor.GetStatistics()

		if stats.TotalExecutions != 1 {
			t.Errorf("expected 1 execution, got %d", stats.TotalExecutions)
		}

		if stats.SuccessRate != 1.0 {
			t.Errorf("expected success rate 1.0, got %.2f", stats.SuccessRate)
		}
	})
}

// TestManager tests manager functionality
func TestManager(t *testing.T) {
	t.Run("create_manager", func(t *testing.T) {
		manager := NewManager()

		if manager.Count() != 0 {
			t.Error("new manager should have 0 hooks")
		}
	})

	t.Run("register_hook", func(t *testing.T) {
		manager := NewManager()
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			return nil
		})

		err := manager.Register(hook)
		if err != nil {
			t.Errorf("register should succeed: %v", err)
		}

		if manager.Count() != 1 {
			t.Errorf("expected 1 hook, got %d", manager.Count())
		}
	})

	t.Run("register_duplicate_id", func(t *testing.T) {
		manager := NewManager()
		hook1 := &Hook{
			ID:       "test-id",
			Name:     "test1",
			Type:     HookTypeBeforeTask,
			Handler:  func(ctx context.Context, event *Event) error { return nil },
			Priority: PriorityNormal,
			Enabled:  true,
		}

		hook2 := &Hook{
			ID:       "test-id",
			Name:     "test2",
			Type:     HookTypeBeforeTask,
			Handler:  func(ctx context.Context, event *Event) error { return nil },
			Priority: PriorityNormal,
			Enabled:  true,
		}

		manager.Register(hook1)
		err := manager.Register(hook2)

		if err == nil {
			t.Error("register should fail for duplicate ID")
		}
	})

	t.Run("unregister_hook", func(t *testing.T) {
		manager := NewManager()
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			return nil
		})

		manager.Register(hook)
		err := manager.Unregister(hook.ID)

		if err != nil {
			t.Errorf("unregister should succeed: %v", err)
		}

		if manager.Count() != 0 {
			t.Errorf("expected 0 hooks, got %d", manager.Count())
		}
	})

	t.Run("get_hook", func(t *testing.T) {
		manager := NewManager()
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			return nil
		})

		manager.Register(hook)

		got, err := manager.Get(hook.ID)
		if err != nil {
			t.Errorf("get should succeed: %v", err)
		}

		if got != hook {
			t.Error("should return the same hook")
		}
	})

	t.Run("get_by_type", func(t *testing.T) {
		manager := NewManager()

		hook1 := NewHook("test1", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			return nil
		})
		hook2 := NewHook("test2", HookTypeAfterTask, func(ctx context.Context, event *Event) error {
			return nil
		})
		hook3 := NewHook("test3", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			return nil
		})

		manager.Register(hook1)
		manager.Register(hook2)
		manager.Register(hook3)

		beforeHooks := manager.GetByType(HookTypeBeforeTask)

		if len(beforeHooks) != 2 {
			t.Errorf("expected 2 before_task hooks, got %d", len(beforeHooks))
		}
	})

	t.Run("get_by_tag", func(t *testing.T) {
		manager := NewManager()

		hook1 := NewHook("test1", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			return nil
		})
		hook1.AddTag("important")

		hook2 := NewHook("test2", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			return nil
		})
		hook2.AddTag("important")

		hook3 := NewHook("test3", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			return nil
		})

		manager.Register(hook1)
		manager.Register(hook2)
		manager.Register(hook3)

		important := manager.GetByTag("important")

		if len(important) != 2 {
			t.Errorf("expected 2 important hooks, got %d", len(important))
		}
	})

	t.Run("enable_disable", func(t *testing.T) {
		manager := NewManager()
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			return nil
		})

		manager.Register(hook)
		manager.Disable(hook.ID)

		got, _ := manager.Get(hook.ID)
		if got.Enabled {
			t.Error("hook should be disabled")
		}

		manager.Enable(hook.ID)
		got, _ = manager.Get(hook.ID)
		if !got.Enabled {
			t.Error("hook should be enabled")
		}
	})

	t.Run("trigger_hooks", func(t *testing.T) {
		manager := NewManager()
		executed := false

		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			executed = true
			return nil
		})

		manager.Register(hook)
		results := manager.Trigger(context.Background(), HookTypeBeforeTask)

		if !executed {
			t.Error("hook should have been executed")
		}

		if len(results) != 1 {
			t.Errorf("expected 1 result, got %d", len(results))
		}

		if results[0].Status != StatusCompleted {
			t.Errorf("expected status completed, got %s", results[0].Status)
		}
	})

	t.Run("trigger_with_event_data", func(t *testing.T) {
		manager := NewManager()
		var receivedValue string

		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			value, _ := event.GetData("test_key")
			receivedValue = value.(string)
			return nil
		})

		manager.Register(hook)

		event := NewEvent(HookTypeBeforeTask)
		event.SetData("test_key", "test_value")
		manager.TriggerEvent(event)

		if receivedValue != "test_value" {
			t.Errorf("expected 'test_value', got %s", receivedValue)
		}
	})

	t.Run("get_statistics", func(t *testing.T) {
		manager := NewManager()

		hook1 := NewHook("test1", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			return nil
		})
		hook2 := NewHook("test2", HookTypeAfterTask, func(ctx context.Context, event *Event) error {
			return nil
		})

		manager.Register(hook1)
		manager.Register(hook2)

		stats := manager.GetStatistics()

		if stats.TotalHooks != 2 {
			t.Errorf("expected 2 total hooks, got %d", stats.TotalHooks)
		}

		if stats.EnabledHooks != 2 {
			t.Errorf("expected 2 enabled hooks, got %d", stats.EnabledHooks)
		}
	})
}

// TestManagerCallbacks tests manager callbacks
func TestManagerCallbacks(t *testing.T) {
	t.Run("on_create_callback", func(t *testing.T) {
		manager := NewManager()
		var createdHook *Hook

		manager.OnCreate(func(hook *Hook) {
			createdHook = hook
		})

		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			return nil
		})

		manager.Register(hook)

		if createdHook != hook {
			t.Error("onCreate callback should have been called")
		}
	})

	t.Run("on_execute_callback", func(t *testing.T) {
		manager := NewManager()
		var executedEvent *Event

		manager.OnExecute(func(event *Event, results []*ExecutionResult) {
			executedEvent = event
		})

		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, event *Event) error {
			return nil
		})

		manager.Register(hook)
		manager.Trigger(context.Background(), HookTypeBeforeTask)

		if executedEvent == nil {
			t.Error("onExecute callback should have been called")
		}

		if executedEvent.Type != HookTypeBeforeTask {
			t.Errorf("expected type before_task, got %s", executedEvent.Type)
		}
	})
}

// TestConcurrency tests concurrent operations
func TestConcurrency(t *testing.T) {
	t.Run("concurrent_registration", func(t *testing.T) {
		manager := NewManager()
		done := make(chan bool)

		for i := 0; i < 10; i++ {
			go func(n int) {
				hook := NewHook(fmt.Sprintf("hook%d", n), HookTypeBeforeTask,
					func(ctx context.Context, event *Event) error {
						return nil
					})
				manager.Register(hook)
				done <- true
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}

		if manager.Count() != 10 {
			t.Errorf("expected 10 hooks, got %d", manager.Count())
		}
	})

	t.Run("concurrent_execution", func(t *testing.T) {
		manager := NewManager()
		counter := 0
		var mu sync.Mutex

		for i := 0; i < 5; i++ {
			hook := NewAsyncHook(fmt.Sprintf("hook%d", i), HookTypeBeforeTask,
				func(ctx context.Context, event *Event) error {
					mu.Lock()
					counter++
					mu.Unlock()
					return nil
				})
			manager.Register(hook)
		}

		manager.TriggerAndWait(context.Background(), HookTypeBeforeTask)

		if counter != 5 {
			t.Errorf("expected 5 executions, got %d", counter)
		}
	})
}
