package hooks

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

// TestNewExecutorWithLimit tests executor creation with concurrency limit
func TestNewExecutorWithLimit(t *testing.T) {
	t.Run("NewExecutorWithLimit", func(t *testing.T) {
		executor := NewExecutorWithLimit(5)

		if executor == nil {
			t.Fatal("expected executor to be created")
		}
		if executor.maxConcurrent != 5 {
			t.Errorf("expected maxConcurrent to be 5, got %d", executor.maxConcurrent)
		}
		if cap(executor.semaphore) != 5 {
			t.Errorf("expected semaphore capacity to be 5, got %d", cap(executor.semaphore))
		}
	})
}

// TestExecuteSync tests synchronous execution
func TestExecuteSync(t *testing.T) {
	t.Run("ExecuteSync", func(t *testing.T) {
		executor := NewExecutor()
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error {
			return nil
		})
		event := NewEvent(HookTypeBeforeTask)

		results := executor.ExecuteSync(context.Background(), []*Hook{hook}, event)

		if len(results) == 0 {
			t.Fatal("expected results to be non-empty")
		}
		if results[0].Status != StatusCompleted {
			t.Errorf("expected completed status, got %s", results[0].Status)
		}
	})

	t.Run("ExecuteSync_WithError", func(t *testing.T) {
		executor := NewExecutor()
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error {
			return fmt.Errorf("test error")
		})
		event := NewEvent(HookTypeBeforeTask)

		results := executor.ExecuteSync(context.Background(), []*Hook{hook}, event)

		if len(results) == 0 {
			t.Fatal("expected results to be non-empty")
		}
		if results[0].Status != StatusFailed {
			t.Errorf("expected failed status, got %s", results[0].Status)
		}
		if results[0].Error == nil {
			t.Error("expected error to be set")
		}
	})
}

// TestGetResults tests result retrieval
func TestGetResults(t *testing.T) {
	t.Run("GetResults", func(t *testing.T) {
		executor := NewExecutor()
		hook1 := NewHook("test1", HookTypeBeforeTask, func(ctx context.Context, e *Event) error {
			return nil
		})
		hook2 := NewHook("test2", HookTypeBeforeTask, func(ctx context.Context, e *Event) error {
			return nil
		})
		event := NewEvent(HookTypeBeforeTask)

		executor.Execute(context.Background(), hook1, event)
		executor.Execute(context.Background(), hook2, event)

		results := executor.GetResults(1)
		if len(results) != 1 {
			t.Errorf("expected 1 result for test1, got %d", len(results))
		}
	})

	t.Run("GetResults_Empty", func(t *testing.T) {
		executor := NewExecutor()
		results := executor.GetResults(10)

		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}
	})
}

// TestGetAllResults tests getting all results
func TestGetAllResults(t *testing.T) {
	t.Run("GetAllResults", func(t *testing.T) {
		executor := NewExecutor()
		hook1 := NewHook("test1", HookTypeBeforeTask, func(ctx context.Context, e *Event) error {
			return nil
		})
		hook2 := NewHook("test2", HookTypeBeforeTask, func(ctx context.Context, e *Event) error {
			return nil
		})
		event := NewEvent(HookTypeBeforeTask)

		executor.Execute(context.Background(), hook1, event)
		executor.Execute(context.Background(), hook2, event)

		allResults := executor.GetAllResults()
		if len(allResults) != 2 {
			t.Errorf("expected 2 results, got %d", len(allResults))
		}
	})
}

// TestGetResultsByStatus tests filtering results by status
func TestGetResultsByStatus(t *testing.T) {
	t.Run("GetResultsByStatus_Success", func(t *testing.T) {
		executor := NewExecutor()
		hook1 := NewHook("test1", HookTypeBeforeTask, func(ctx context.Context, e *Event) error {
			return nil
		})
		hook2 := NewHook("test2", HookTypeBeforeTask, func(ctx context.Context, e *Event) error {
			return fmt.Errorf("error")
		})
		event := NewEvent(HookTypeBeforeTask)

		executor.ExecuteSync(context.Background(), []*Hook{hook1}, event)
		executor.ExecuteSync(context.Background(), []*Hook{hook2}, event)

		successResults := executor.GetResultsByStatus(StatusCompleted)
		if len(successResults) != 1 {
			t.Errorf("expected 1 success result, got %d", len(successResults))
		}

		failedResults := executor.GetResultsByStatus(StatusFailed)
		if len(failedResults) != 1 {
			t.Errorf("expected 1 failed result, got %d", len(failedResults))
		}
	})
}

// TestClearResults tests clearing results
func TestClearResults(t *testing.T) {
	t.Run("ClearResults", func(t *testing.T) {
		executor := NewExecutor()
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error {
			return nil
		})
		event := NewEvent(HookTypeBeforeTask)

		executor.Execute(context.Background(), hook, event)
		if len(executor.GetAllResults()) == 0 {
			t.Error("expected results to be present before clearing")
		}

		executor.ClearResults()

		if len(executor.GetAllResults()) != 0 {
			t.Errorf("expected 0 results after clearing, got %d", len(executor.GetAllResults()))
		}
	})
}

// TestOnComplete and TestOnError test callbacks
func TestCallbackRegistration(t *testing.T) {
	t.Run("OnComplete", func(t *testing.T) {
		executor := NewExecutor()
		called := false

		executor.OnComplete(func(r *ExecutionResult) {
			called = true
		})

		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error {
			return nil
		})
		event := NewEvent(HookTypeBeforeTask)

		executor.ExecuteSync(context.Background(), []*Hook{hook}, event)

		if !called {
			t.Error("expected OnComplete callback to be called")
		}
	})

	t.Run("OnError", func(t *testing.T) {
		executor := NewExecutor()
		errorCalled := false

		executor.OnError(func(r *ExecutionResult) {
			errorCalled = true
		})

		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error {
			return fmt.Errorf("test error")
		})
		event := NewEvent(HookTypeBeforeTask)

		executor.ExecuteSync(context.Background(), []*Hook{hook}, event)

		if !errorCalled {
			t.Error("expected OnError callback to be called")
		}
	})

	t.Run("MultipleCallbacks", func(t *testing.T) {
		executor := NewExecutor()
		completeCount := 0

		executor.OnComplete(func(r *ExecutionResult) {
			completeCount++
		})
		executor.OnComplete(func(r *ExecutionResult) {
			completeCount++
		})

		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error {
			return nil
		})
		event := NewEvent(HookTypeBeforeTask)

		executor.ExecuteSync(context.Background(), []*Hook{hook}, event)

		if completeCount != 2 {
			t.Errorf("expected 2 callback calls, got %d", completeCount)
		}
	})
}

// TestSetMaxConcurrent tests setting max concurrent
func TestSetMaxConcurrent(t *testing.T) {
	t.Run("SetMaxConcurrent", func(t *testing.T) {
		executor := NewExecutor()
		executor.SetMaxConcurrent(10)

		if executor.maxConcurrent != 10 {
			t.Errorf("expected maxConcurrent to be 10, got %d", executor.maxConcurrent)
		}
		if cap(executor.semaphore) != 10 {
			t.Errorf("expected semaphore capacity to be 10, got %d", cap(executor.semaphore))
		}
	})
}

// TestSetMaxResults tests setting max results
func TestSetMaxResults(t *testing.T) {
	t.Run("SetMaxResults", func(t *testing.T) {
		executor := NewExecutor()
		executor.SetMaxResults(5)

		if executor.maxResults != 5 {
			t.Errorf("expected maxResults to be 5, got %d", executor.maxResults)
		}
	})

	t.Run("SetMaxResults_WithTrimming", func(t *testing.T) {
		executor := NewExecutor()
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error {
			return nil
		})
		event := NewEvent(HookTypeBeforeTask)

		// Add 10 results
		for i := 0; i < 10; i++ {
			executor.Execute(context.Background(), hook, event)
		}

		// Set max to 5
		executor.SetMaxResults(5)

		// Should have only 5 results now
		if len(executor.GetAllResults()) > 5 {
			t.Errorf("expected at most 5 results after trimming, got %d", len(executor.GetAllResults()))
		}
	})
}

// TestExecutionResultString tests String method
func TestExecutionResultString(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error {
			return nil
		})
		result := NewExecutionResult(hook)
		result.Complete(nil)

		str := result.String()

		if str == "" {
			t.Error("expected non-empty string")
		}
		if !strings.Contains(str, "test") {
			t.Error("expected string to contain hook name")
		}
		if !strings.Contains(str, string(StatusCompleted)) {
			t.Error("expected string to contain status")
		}
	})
}

// TestExecuteWithTimeout tests execution with timeout
func TestExecuteWithTimeout(t *testing.T) {
	t.Run("ExecuteWithTimeout_Success", func(t *testing.T) {
		executor := NewExecutor()
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
		event := NewEvent(HookTypeBeforeTask)

		result := executor.ExecuteWithTimeout(hook, event, 500*time.Millisecond)

		if result.Status != StatusCompleted {
			t.Errorf("expected completed, got %s", result.Status)
		}
	})

	t.Run("ExecuteWithTimeout_Timeout", func(t *testing.T) {
		executor := NewExecutor()
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(500 * time.Millisecond):
				return nil
			}
		})
		event := NewEvent(HookTypeBeforeTask)

		result := executor.ExecuteWithTimeout(hook, event, 50*time.Millisecond)

		if result.Status != StatusFailed {
			t.Errorf("expected failed due to timeout, got %s", result.Status)
		}
	})
}

// TestExecuteWithDeadline tests execution with deadline
func TestExecuteWithDeadline(t *testing.T) {
	t.Run("ExecuteWithDeadline_Success", func(t *testing.T) {
		executor := NewExecutor()
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
		event := NewEvent(HookTypeBeforeTask)
		deadline := time.Now().Add(500 * time.Millisecond)

		result := executor.ExecuteWithDeadline(hook, event, deadline)

		if result.Status != StatusCompleted {
			t.Errorf("expected completed, got %s", result.Status)
		}
	})

	t.Run("ExecuteWithDeadline_Exceeded", func(t *testing.T) {
		executor := NewExecutor()
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(500 * time.Millisecond):
				return nil
			}
		})
		event := NewEvent(HookTypeBeforeTask)
		deadline := time.Now().Add(50 * time.Millisecond)

		result := executor.ExecuteWithDeadline(hook, event, deadline)

		if result.Status != StatusFailed {
			t.Errorf("expected failed due to deadline, got %s", result.Status)
		}
	})
}

// TestHookMetadata tests metadata methods
func TestHookMetadata(t *testing.T) {
	t.Run("SetMetadata_GetMetadata", func(t *testing.T) {
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error {
			return nil
		})

		hook.SetMetadata("key1", "value1")
		hook.SetMetadata("key2", "value2")

		val1, ok1 := hook.GetMetadata("key1")
		if !ok1 {
			t.Error("expected key1 to exist")
		}
		if val1 != "value1" {
			t.Errorf("expected 'value1', got %v", val1)
		}

		val2, ok2 := hook.GetMetadata("key2")
		if !ok2 {
			t.Error("expected key2 to exist")
		}
		if val2 != "value2" {
			t.Errorf("expected 'value2', got %v", val2)
		}

		val3, ok3 := hook.GetMetadata("nonexistent")
		if ok3 {
			t.Errorf("expected nonexistent key to not exist, got %v", val3)
		}
	})

	t.Run("SetMetadata_Overwrite", func(t *testing.T) {
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error {
			return nil
		})

		hook.SetMetadata("key", "value1")
		hook.SetMetadata("key", "value2")

		val, ok := hook.GetMetadata("key")
		if !ok {
			t.Error("expected key to exist")
		}
		if val != "value2" {
			t.Errorf("expected 'value2', got %v", val)
		}
	})
}

// TestManagerFunctions tests Manager functions
func TestManagerFunctions(t *testing.T) {
	t.Run("NewManagerWithExecutor", func(t *testing.T) {
		executor := NewExecutorWithLimit(5)
		manager := NewManagerWithExecutor(executor)

		if manager.GetExecutor() != executor {
			t.Error("expected manager to use custom executor")
		}
	})

	t.Run("RegisterMany", func(t *testing.T) {
		manager := NewManager()
		hooks := []*Hook{
			NewHook("test1", HookTypeBeforeTask, func(ctx context.Context, e *Event) error { return nil }),
			NewHook("test2", HookTypeAfterTask, func(ctx context.Context, e *Event) error { return nil }),
		}

		err := manager.RegisterMany(hooks)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if manager.Count() != 2 {
			t.Errorf("expected 2 hooks, got %d", manager.Count())
		}
	})

	t.Run("GetAll", func(t *testing.T) {
		manager := NewManager()
		hook1 := NewHook("test1", HookTypeBeforeTask, func(ctx context.Context, e *Event) error { return nil })
		hook2 := NewHook("test2", HookTypeAfterTask, func(ctx context.Context, e *Event) error { return nil })

		manager.Register(hook1)
		manager.Register(hook2)

		allHooks := manager.GetAll()
		if len(allHooks) != 2 {
			t.Errorf("expected 2 hooks, got %d", len(allHooks))
		}
	})

	t.Run("GetEnabled", func(t *testing.T) {
		manager := NewManager()
		hook1 := NewHook("test1", HookTypeBeforeTask, func(ctx context.Context, e *Event) error { return nil })
		hook2 := NewHook("test2", HookTypeAfterTask, func(ctx context.Context, e *Event) error { return nil })

		manager.Register(hook1)
		manager.Register(hook2)
		manager.Disable(hook2.ID)

		enabled := manager.GetEnabled()
		if len(enabled) != 1 {
			t.Errorf("expected 1 enabled hook, got %d", len(enabled))
		}
	})

	t.Run("EnableAll", func(t *testing.T) {
		manager := NewManager()
		hook1 := NewHook("test1", HookTypeBeforeTask, func(ctx context.Context, e *Event) error { return nil })
		hook2 := NewHook("test2", HookTypeAfterTask, func(ctx context.Context, e *Event) error { return nil })

		manager.Register(hook1)
		manager.Register(hook2)
		manager.Disable(hook1.ID)
		manager.Disable(hook2.ID)

		manager.EnableAll()

		enabled := manager.GetEnabled()
		if len(enabled) != 2 {
			t.Errorf("expected 2 enabled hooks, got %d", len(enabled))
		}
	})

	t.Run("DisableAll", func(t *testing.T) {
		manager := NewManager()
		hook1 := NewHook("test1", HookTypeBeforeTask, func(ctx context.Context, e *Event) error { return nil })
		hook2 := NewHook("test2", HookTypeAfterTask, func(ctx context.Context, e *Event) error { return nil })

		manager.Register(hook1)
		manager.Register(hook2)

		manager.DisableAll()

		enabled := manager.GetEnabled()
		if len(enabled) != 0 {
			t.Errorf("expected 0 enabled hooks, got %d", len(enabled))
		}
	})

	t.Run("CountByType", func(t *testing.T) {
		manager := NewManager()
		hook1 := NewHook("test1", HookTypeBeforeTask, func(ctx context.Context, e *Event) error { return nil })
		hook2 := NewHook("test2", HookTypeBeforeTask, func(ctx context.Context, e *Event) error { return nil })
		hook3 := NewHook("test3", HookTypeAfterTask, func(ctx context.Context, e *Event) error { return nil })

		manager.Register(hook1)
		manager.Register(hook2)
		manager.Register(hook3)

		beforeCount := manager.CountByType(HookTypeBeforeTask)
		if beforeCount != 2 {
			t.Errorf("expected 2 before_task hooks, got %d", beforeCount)
		}

		afterCount := manager.CountByType(HookTypeAfterTask)
		if afterCount != 1 {
			t.Errorf("expected 1 after_task hook, got %d", afterCount)
		}
	})

	t.Run("Clear", func(t *testing.T) {
		manager := NewManager()
		hook1 := NewHook("test1", HookTypeBeforeTask, func(ctx context.Context, e *Event) error { return nil })
		hook2 := NewHook("test2", HookTypeAfterTask, func(ctx context.Context, e *Event) error { return nil })

		manager.Register(hook1)
		manager.Register(hook2)

		if manager.Count() != 2 {
			t.Errorf("expected 2 hooks before clear, got %d", manager.Count())
		}

		manager.Clear()

		if manager.Count() != 0 {
			t.Errorf("expected 0 hooks after clear, got %d", manager.Count())
		}
	})

	t.Run("TriggerSync", func(t *testing.T) {
		manager := NewManager()
		executed := false
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error {
			executed = true
			return nil
		})

		manager.Register(hook)
		results := manager.TriggerSync(context.Background(), HookTypeBeforeTask)

		if !executed {
			t.Error("expected hook to be executed")
		}
		if len(results) != 1 {
			t.Errorf("expected 1 result, got %d", len(results))
		}
	})

	t.Run("TriggerEventSync", func(t *testing.T) {
		manager := NewManager()
		executed := false
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error {
			executed = true
			return nil
		})

		manager.Register(hook)
		event := NewEvent(HookTypeBeforeTask)
		results := manager.TriggerEventSync(event)

		if !executed {
			t.Error("expected hook to be executed")
		}
		if len(results) != 1 {
			t.Errorf("expected 1 result, got %d", len(results))
		}
	})

	t.Run("Wait", func(t *testing.T) {
		manager := NewManager()
		hook := NewAsyncHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})

		manager.Register(hook)
		manager.Trigger(context.Background(), HookTypeBeforeTask)

		// Should not hang
		manager.Wait()
	})

	t.Run("OnRemove", func(t *testing.T) {
		manager := NewManager()
		removed := false

		manager.OnRemove(func(h *Hook) {
			removed = true
		})

		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error { return nil })
		manager.Register(hook)
		manager.Unregister(hook.ID)

		if !removed {
			t.Error("expected OnRemove callback to be called")
		}
	})

	t.Run("FindByName", func(t *testing.T) {
		manager := NewManager()
		hook := NewHook("mytest", HookTypeBeforeTask, func(ctx context.Context, e *Event) error { return nil })
		manager.Register(hook)

		found := manager.FindByName("mytest")
		if len(found) != 1 {
			t.Errorf("expected 1 hook, got %d", len(found))
		}

		notFound := manager.FindByName("nonexistent")
		if len(notFound) != 0 {
			t.Errorf("expected 0 hooks, got %d", len(notFound))
		}
	})

	t.Run("UpdatePriority", func(t *testing.T) {
		manager := NewManager()
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error { return nil })
		manager.Register(hook)

		err := manager.UpdatePriority(hook.ID, PriorityHigh)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		retrieved, _ := manager.Get(hook.ID)
		if retrieved.Priority != PriorityHigh {
			t.Errorf("expected priority %d, got %d", PriorityHigh, retrieved.Priority)
		}
	})
}

// TestHookClone tests Hook.Clone
func TestHookClone(t *testing.T) {
	t.Run("Clone", func(t *testing.T) {
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error { return nil })
		hook.SetMetadata("key", "value")
		hook.AddTag("tag1")

		clone := hook.Clone()

		if clone.ID != hook.ID {
			t.Error("expected clone to have same ID")
		}
		if clone.Name != hook.Name {
			t.Error("expected clone to have same name")
		}

		val, _ := clone.GetMetadata("key")
		if val != "value" {
			t.Error("expected clone to have same metadata")
		}

		if !clone.HasTag("tag1") {
			t.Error("expected clone to have same tags")
		}
	})
}

// TestExecutionResultCancelAndSkip tests Cancel and Skip methods
func TestExecutionResultCancelAndSkip(t *testing.T) {
	t.Run("Cancel", func(t *testing.T) {
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error { return nil })
		result := NewExecutionResult(hook)

		result.Cancel()

		if result.Status != StatusCanceled {
			t.Errorf("expected status %s, got %s", StatusCanceled, result.Status)
		}
		if result.Duration == 0 {
			t.Error("expected duration to be set")
		}
	})

	t.Run("Skip", func(t *testing.T) {
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error { return nil })
		result := NewExecutionResult(hook)

		result.Skip()

		if result.Status != StatusSkipped {
			t.Errorf("expected status %s, got %s", StatusSkipped, result.Status)
		}
	})
}

// TestStringMethods tests String() methods
func TestStringMethods(t *testing.T) {
	t.Run("Hook.String", func(t *testing.T) {
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error { return nil })
		str := hook.String()

		if str == "" {
			t.Error("expected non-empty string")
		}
		if !strings.Contains(str, "test") {
			t.Error("expected string to contain hook name")
		}
	})

	t.Run("Event.String", func(t *testing.T) {
		event := NewEvent(HookTypeBeforeTask)
		event.Source = "test-source"
		str := event.String()

		if str == "" {
			t.Error("expected non-empty string")
		}
		if !strings.Contains(str, string(HookTypeBeforeTask)) {
			t.Error("expected string to contain event type")
		}
	})

	t.Run("ExecutorStatistics.String", func(t *testing.T) {
		stats := &ExecutorStatistics{
			TotalExecutions: 10,
			SuccessRate:     0.8,
			AverageDuration: 100 * time.Millisecond,
			ByStatus:        make(map[HookStatus]int),
		}

		str := stats.String()

		if str == "" {
			t.Error("expected non-empty string")
		}
		if !strings.Contains(str, "10") {
			t.Error("expected string to contain total executions")
		}
	})
}

// TestGetExecutor tests GetExecutor
func TestGetExecutor(t *testing.T) {
	t.Run("GetExecutor", func(t *testing.T) {
		manager := NewManager()
		executor := manager.GetExecutor()

		if executor == nil {
			t.Error("expected executor to be non-nil")
		}
	})
}

// TestManagerCloneAndExport tests Clone and Export
func TestManagerCloneAndExport(t *testing.T) {
	t.Run("Clone", func(t *testing.T) {
		manager := NewManager()
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error { return nil })
		hook.SetMetadata("key", "value")
		manager.Register(hook)

		cloned, err := manager.Clone(hook.ID)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if cloned.ID != hook.ID {
			t.Error("expected clone to have same ID")
		}

		val, _ := cloned.GetMetadata("key")
		if val != "value" {
			t.Error("expected clone to have same metadata")
		}
	})

	t.Run("Clone_NotFound", func(t *testing.T) {
		manager := NewManager()
		_, err := manager.Clone("nonexistent")

		if err == nil {
			t.Error("expected error for nonexistent hook")
		}
	})

	t.Run("Export", func(t *testing.T) {
		manager := NewManager()
		hook1 := NewHook("test1", HookTypeBeforeTask, func(ctx context.Context, e *Event) error { return nil })
		hook2 := NewHook("test2", HookTypeAfterTask, func(ctx context.Context, e *Event) error { return nil })

		manager.Register(hook1)
		manager.Register(hook2)

		exported := manager.Export()
		if len(exported) != 2 {
			t.Errorf("expected 2 exported hooks, got %d", len(exported))
		}

		// Check metadata structure
		if exported[0].ID == "" || exported[0].Name == "" {
			t.Error("expected exported hook to have ID and name")
		}
	})
}

// TestManagerStatisticsString tests ManagerStatistics.String
func TestManagerStatisticsString(t *testing.T) {
	t.Run("ManagerStatistics.String", func(t *testing.T) {
		stats := &ManagerStatistics{
			TotalHooks:    10,
			EnabledHooks:  8,
			DisabledHooks: 2,
			ByType:        make(map[HookType]int),
			ExecutorStats: &ExecutorStatistics{
				TotalExecutions: 5,
				SuccessRate:     0.8,
				AverageDuration: 50 * time.Millisecond,
				ByStatus:        make(map[HookStatus]int),
			},
		}

		str := stats.String()
		if str == "" {
			t.Error("expected non-empty string")
		}
		if !strings.Contains(str, "10") {
			t.Error("expected string to contain total hooks")
		}
		if !strings.Contains(str, "8") {
			t.Error("expected string to contain enabled count")
		}
	})
}

// TestHookValidate tests more validation edge cases
func TestHookValidate(t *testing.T) {
	t.Run("Validate_EmptyID", func(t *testing.T) {
		hook := &Hook{
			ID:       "",
			Name:     "test",
			Type:     HookTypeBeforeTask,
			Handler:  func(ctx context.Context, e *Event) error { return nil },
			Priority: PriorityNormal,
		}

		err := hook.Validate()
		if err == nil {
			t.Error("expected error for empty ID")
		}
	})

	t.Run("Validate_InvalidPriority", func(t *testing.T) {
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error { return nil })
		hook.Priority = 200 // Invalid priority

		err := hook.Validate()
		if err == nil {
			t.Error("expected error for invalid priority")
		}
	})
}

// TestRegisterManyError tests RegisterMany with errors
func TestRegisterManyError(t *testing.T) {
	t.Run("RegisterMany_WithInvalidHook", func(t *testing.T) {
		manager := NewManager()
		invalidHook := &Hook{
			ID:       "",
			Name:     "test",
			Type:     HookTypeBeforeTask,
			Handler:  func(ctx context.Context, e *Event) error { return nil },
			Priority: PriorityNormal,
		}

		err := manager.RegisterMany([]*Hook{invalidHook})
		if err == nil {
			t.Error("expected error for invalid hook")
		}
	})
}

// TestShouldExecute tests edge cases
func TestShouldExecuteEdgeCases(t *testing.T) {
	t.Run("ShouldExecute_DisabledHook", func(t *testing.T) {
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error { return nil })
		hook.Enabled = false
		event := NewEvent(HookTypeBeforeTask)

		if hook.ShouldExecute(event) {
			t.Error("expected disabled hook to not execute")
		}
	})

	t.Run("ShouldExecute_WithCondition", func(t *testing.T) {
		hook := NewHook("test", HookTypeBeforeTask, func(ctx context.Context, e *Event) error { return nil })
		hook.Condition = func(e *Event) bool {
			return false
		}
		event := NewEvent(HookTypeBeforeTask)

		if hook.ShouldExecute(event) {
			t.Error("expected hook with false condition to not execute")
		}
	})
}
