package monitoring

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// ========================================
// Mock Collector for Testing
// ========================================

type MockCollector struct {
	name    string
	metrics map[string]interface{}
	err     error
}

func (m *MockCollector) Name() string {
	return m.name
}

func (m *MockCollector) Collect() (map[string]interface{}, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.metrics, nil
}

func NewMockCollector(name string, metrics map[string]interface{}) *MockCollector {
	return &MockCollector{
		name:    name,
		metrics: metrics,
	}
}

func NewFailingMockCollector(name string, err error) *MockCollector {
	return &MockCollector{
		name: name,
		err:  err,
	}
}

// ========================================
// Constructor Tests
// ========================================

func TestNewMonitor(t *testing.T) {
	monitor := NewMonitor()
	if monitor == nil {
		t.Fatal("Expected monitor, got nil")
	}
	if monitor.logger == nil {
		t.Error("Expected logger to be initialized")
	}
	if monitor.metrics == nil {
		t.Error("Expected metrics map to be initialized")
	}
	if monitor.collectors == nil {
		t.Error("Expected collectors slice to be initialized")
	}
}

// ========================================
// AddCollector Tests
// ========================================

func TestMonitor_AddCollector(t *testing.T) {
	monitor := NewMonitor()
	collector := NewMockCollector("test", map[string]interface{}{"metric1": 100})

	monitor.AddCollector(collector)

	monitor.mutex.RLock()
	count := len(monitor.collectors)
	monitor.mutex.RUnlock()

	if count != 1 {
		t.Errorf("Expected 1 collector, got %d", count)
	}
}

func TestMonitor_AddCollector_Multiple(t *testing.T) {
	monitor := NewMonitor()

	collector1 := NewMockCollector("test1", map[string]interface{}{"metric1": 100})
	collector2 := NewMockCollector("test2", map[string]interface{}{"metric2": 200})
	collector3 := NewMockCollector("test3", map[string]interface{}{"metric3": 300})

	monitor.AddCollector(collector1)
	monitor.AddCollector(collector2)
	monitor.AddCollector(collector3)

	monitor.mutex.RLock()
	count := len(monitor.collectors)
	monitor.mutex.RUnlock()

	if count != 3 {
		t.Errorf("Expected 3 collectors, got %d", count)
	}
}

// ========================================
// CollectMetrics Tests
// ========================================

func TestMonitor_CollectMetrics(t *testing.T) {
	monitor := NewMonitor()
	ctx := context.Background()

	collector := NewMockCollector("test", map[string]interface{}{
		"cpu":    85.5,
		"memory": 75.2,
	})
	monitor.AddCollector(collector)

	err := monitor.CollectMetrics(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify metrics were collected
	if cpu, exists := monitor.GetMetric("cpu"); !exists {
		t.Error("Expected cpu metric to exist")
	} else if cpu != 85.5 {
		t.Errorf("Expected cpu=85.5, got %v", cpu)
	}

	if memory, exists := monitor.GetMetric("memory"); !exists {
		t.Error("Expected memory metric to exist")
	} else if memory != 75.2 {
		t.Errorf("Expected memory=75.2, got %v", memory)
	}
}

func TestMonitor_CollectMetrics_MultipleCollectors(t *testing.T) {
	monitor := NewMonitor()
	ctx := context.Background()

	collector1 := NewMockCollector("collector1", map[string]interface{}{
		"metric1": "value1",
		"metric2": 123,
	})
	collector2 := NewMockCollector("collector2", map[string]interface{}{
		"metric3": true,
		"metric4": 45.6,
	})

	monitor.AddCollector(collector1)
	monitor.AddCollector(collector2)

	err := monitor.CollectMetrics(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify all metrics were collected
	allMetrics := monitor.GetAllMetrics()
	if len(allMetrics) != 4 {
		t.Errorf("Expected 4 metrics, got %d", len(allMetrics))
	}
}

func TestMonitor_CollectMetrics_WithError(t *testing.T) {
	monitor := NewMonitor()
	ctx := context.Background()

	// Add a collector that fails
	failingCollector := NewFailingMockCollector("failing", errors.New("collection failed"))
	monitor.AddCollector(failingCollector)

	// Add a working collector
	workingCollector := NewMockCollector("working", map[string]interface{}{
		"metric1": 100,
	})
	monitor.AddCollector(workingCollector)

	// Should not return error, but should log it
	err := monitor.CollectMetrics(ctx)
	if err != nil {
		t.Errorf("Expected no error (errors are logged), got %v", err)
	}

	// Working collector should still have collected metrics
	if _, exists := monitor.GetMetric("metric1"); !exists {
		t.Error("Expected metric1 from working collector")
	}
}

func TestMonitor_CollectMetrics_NoCollectors(t *testing.T) {
	monitor := NewMonitor()
	ctx := context.Background()

	err := monitor.CollectMetrics(ctx)
	if err != nil {
		t.Errorf("Expected no error with no collectors, got %v", err)
	}

	allMetrics := monitor.GetAllMetrics()
	if len(allMetrics) != 0 {
		t.Errorf("Expected 0 metrics, got %d", len(allMetrics))
	}
}

// ========================================
// GetMetric Tests
// ========================================

func TestMonitor_GetMetric(t *testing.T) {
	monitor := NewMonitor()
	ctx := context.Background()

	collector := NewMockCollector("test", map[string]interface{}{
		"test_metric": "test_value",
	})
	monitor.AddCollector(collector)
	monitor.CollectMetrics(ctx)

	value, exists := monitor.GetMetric("test_metric")
	if !exists {
		t.Fatal("Expected metric to exist")
	}
	if value != "test_value" {
		t.Errorf("Expected value 'test_value', got %v", value)
	}
}

func TestMonitor_GetMetric_NotExists(t *testing.T) {
	monitor := NewMonitor()

	_, exists := monitor.GetMetric("non_existent")
	if exists {
		t.Error("Expected metric to not exist")
	}
}

func TestMonitor_GetMetric_DifferentTypes(t *testing.T) {
	monitor := NewMonitor()
	ctx := context.Background()

	collector := NewMockCollector("test", map[string]interface{}{
		"string_metric": "value",
		"int_metric":    42,
		"float_metric":  3.14,
		"bool_metric":   true,
	})
	monitor.AddCollector(collector)
	monitor.CollectMetrics(ctx)

	tests := []struct {
		key      string
		expected interface{}
	}{
		{"string_metric", "value"},
		{"int_metric", 42},
		{"float_metric", 3.14},
		{"bool_metric", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			value, exists := monitor.GetMetric(tt.key)
			if !exists {
				t.Fatalf("Expected %s to exist", tt.key)
			}
			if value != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, value)
			}
		})
	}
}

// ========================================
// GetAllMetrics Tests
// ========================================

func TestMonitor_GetAllMetrics(t *testing.T) {
	monitor := NewMonitor()
	ctx := context.Background()

	collector := NewMockCollector("test", map[string]interface{}{
		"metric1": 100,
		"metric2": 200,
		"metric3": 300,
	})
	monitor.AddCollector(collector)
	monitor.CollectMetrics(ctx)

	allMetrics := monitor.GetAllMetrics()

	if len(allMetrics) != 3 {
		t.Errorf("Expected 3 metrics, got %d", len(allMetrics))
	}

	if allMetrics["metric1"] != 100 {
		t.Error("Expected metric1 = 100")
	}
	if allMetrics["metric2"] != 200 {
		t.Error("Expected metric2 = 200")
	}
	if allMetrics["metric3"] != 300 {
		t.Error("Expected metric3 = 300")
	}
}

func TestMonitor_GetAllMetrics_ReturnsCopy(t *testing.T) {
	monitor := NewMonitor()
	ctx := context.Background()

	collector := NewMockCollector("test", map[string]interface{}{
		"metric1": 100,
	})
	monitor.AddCollector(collector)
	monitor.CollectMetrics(ctx)

	metrics1 := monitor.GetAllMetrics()
	metrics2 := monitor.GetAllMetrics()

	// Modify metrics1
	metrics1["metric1"] = 999
	metrics1["new_metric"] = "added"

	// metrics2 should be unchanged
	if metrics2["metric1"] != 100 {
		t.Error("GetAllMetrics should return a copy, not the same map")
	}
	if _, exists := metrics2["new_metric"]; exists {
		t.Error("Changes to returned map should not affect other calls")
	}
}

func TestMonitor_GetAllMetrics_Empty(t *testing.T) {
	monitor := NewMonitor()

	allMetrics := monitor.GetAllMetrics()

	if allMetrics == nil {
		t.Fatal("Expected empty map, got nil")
	}
	if len(allMetrics) != 0 {
		t.Errorf("Expected 0 metrics, got %d", len(allMetrics))
	}
}

// ========================================
// StartPeriodicCollection Tests
// ========================================

func TestMonitor_StartPeriodicCollection(t *testing.T) {
	monitor := NewMonitor()
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	collector := NewMockCollector("test", map[string]interface{}{
		"counter": 1,
	})
	monitor.AddCollector(collector)

	// Start collection in background
	go monitor.StartPeriodicCollection(ctx, 50*time.Millisecond)

	// Wait for context to timeout
	<-ctx.Done()

	// Verify metrics were collected at least once
	_, exists := monitor.GetMetric("counter")
	if !exists {
		t.Error("Expected periodic collection to have collected metrics")
	}
}

func TestMonitor_StartPeriodicCollection_ContextCancellation(t *testing.T) {
	monitor := NewMonitor()
	ctx, cancel := context.WithCancel(context.Background())

	collector := NewMockCollector("test", map[string]interface{}{
		"metric": 1,
	})
	monitor.AddCollector(collector)

	done := make(chan bool)
	go func() {
		monitor.StartPeriodicCollection(ctx, 100*time.Millisecond)
		done <- true
	}()

	// Let it run briefly
	time.Sleep(50 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for goroutine to exit
	select {
	case <-done:
		// Success - goroutine exited
	case <-time.After(1 * time.Second):
		t.Error("StartPeriodicCollection did not exit after context cancellation")
	}
}

// ========================================
// HealthCheck Tests
// ========================================

func TestMonitor_HealthCheck(t *testing.T) {
	monitor := NewMonitor()

	// No collectors - should fail
	err := monitor.HealthCheck()
	if err == nil {
		t.Error("Expected error when no collectors registered")
	}

	// Add collector - should pass
	collector := NewMockCollector("test", map[string]interface{}{})
	monitor.AddCollector(collector)

	err = monitor.HealthCheck()
	if err != nil {
		t.Errorf("Expected no error with collectors registered, got %v", err)
	}
}

// ========================================
// Concurrency Tests
// ========================================

func TestMonitor_Concurrency_AddAndCollect(t *testing.T) {
	monitor := NewMonitor()
	ctx := context.Background()

	var wg sync.WaitGroup
	const numGoroutines = 10

	// Concurrently add collectors
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			collector := NewMockCollector("collector", map[string]interface{}{
				"metric": id,
			})
			monitor.AddCollector(collector)
		}(i)
	}

	// Concurrently collect metrics
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			monitor.CollectMetrics(ctx)
		}()
	}

	wg.Wait()
	// If we reach here without deadlock or panic, the test passes
}

func TestMonitor_Concurrency_ReadAndWrite(t *testing.T) {
	monitor := NewMonitor()
	ctx := context.Background()

	collector := NewMockCollector("test", map[string]interface{}{
		"metric1": 100,
	})
	monitor.AddCollector(collector)
	monitor.CollectMetrics(ctx)

	var wg sync.WaitGroup
	const numReaders = 10
	const numWriters = 10

	// Concurrent readers
	wg.Add(numReaders)
	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()
			monitor.GetMetric("metric1")
			monitor.GetAllMetrics()
		}()
	}

	// Concurrent writers
	wg.Add(numWriters)
	for i := 0; i < numWriters; i++ {
		go func() {
			defer wg.Done()
			monitor.CollectMetrics(ctx)
		}()
	}

	wg.Wait()
	// If we reach here without race condition, the test passes
}

// ========================================
// Edge Cases
// ========================================

func TestMonitor_CollectMetrics_OverwriteExistingMetrics(t *testing.T) {
	monitor := NewMonitor()
	ctx := context.Background()

	// First collection
	collector1 := NewMockCollector("collector1", map[string]interface{}{
		"metric": "value1",
	})
	monitor.AddCollector(collector1)
	monitor.CollectMetrics(ctx)

	// Verify first value
	value, _ := monitor.GetMetric("metric")
	if value != "value1" {
		t.Errorf("Expected 'value1', got %v", value)
	}

	// Second collection with same key
	collector2 := NewMockCollector("collector2", map[string]interface{}{
		"metric": "value2",
	})
	monitor.AddCollector(collector2)
	monitor.CollectMetrics(ctx)

	// Verify value was overwritten
	value, _ = monitor.GetMetric("metric")
	if value != "value2" {
		t.Errorf("Expected 'value2', got %v", value)
	}
}

func TestMonitor_CollectMetrics_NilMetrics(t *testing.T) {
	monitor := NewMonitor()
	ctx := context.Background()

	// Collector that returns nil metrics
	collector := &MockCollector{
		name:    "nil-collector",
		metrics: nil,
		err:     nil,
	}
	monitor.AddCollector(collector)

	// Should not panic
	err := monitor.CollectMetrics(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
