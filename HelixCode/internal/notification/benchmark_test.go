package notification

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// Benchmark individual channel sending

func BenchmarkSlackChannel_Send(b *testing.B) {
	channel := NewSlackChannel("http://localhost:9999/webhook", "#test", "bot")
	notification := &Notification{
		Title:   "Benchmark Test",
		Message: "This is a benchmark notification",
		Type:    NotificationTypeInfo,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = channel.Send(context.Background(), notification)
	}
}

func BenchmarkDiscordChannel_Send(b *testing.B) {
	channel := NewDiscordChannel("http://localhost:9999/webhook")
	notification := &Notification{
		Title:   "Benchmark Test",
		Message: "This is a benchmark notification",
		Type:    NotificationTypeInfo,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = channel.Send(context.Background(), notification)
	}
}

func BenchmarkTelegramChannel_Send(b *testing.B) {
	channel := NewTelegramChannel("test-token", "12345")
	notification := &Notification{
		Title:   "Benchmark Test",
		Message: "This is a benchmark notification",
		Type:    NotificationTypeInfo,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = channel.Send(context.Background(), notification)
	}
}

// Benchmark notification engine

func BenchmarkNotificationEngine_RegisterChannel(b *testing.B) {
	engine := NewNotificationEngine()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		channel := NewSlackChannel(fmt.Sprintf("http://localhost/%d", i), "#test", "bot")
		_ = engine.RegisterChannel(channel)
	}
}

func BenchmarkNotificationEngine_SendDirect(b *testing.B) {
	engine := NewNotificationEngine()
	mockCh := &mockChannel{name: "mock"}
	engine.RegisterChannel(mockCh)

	notification := &Notification{
		Title:   "Benchmark Test",
		Message: "This is a benchmark notification",
		Type:    NotificationTypeInfo,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.SendDirect(context.Background(), notification, []string{"mock"})
	}
}

func BenchmarkNotificationEngine_SendNotificationWithRules(b *testing.B) {
	engine := NewNotificationEngine()
	mockCh := &mockChannel{name: "mock"}
	engine.RegisterChannel(mockCh)

	engine.AddRule(NotificationRule{
		Name:      "Test Rule",
		Condition: "type==info",
		Channels:  []string{"mock"},
		Enabled:   true,
	})

	notification := &Notification{
		Title:   "Benchmark Test",
		Message: "This is a benchmark notification",
		Type:    NotificationTypeInfo,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.SendNotification(context.Background(), notification)
	}
}

// Benchmark retry mechanism

func BenchmarkRetryableChannel_SendSuccess(b *testing.B) {
	baseCh := &retryMockChannel{
		sendFunc: func(ctx context.Context, notif *Notification) error {
			return nil
		},
	}

	config := DefaultRetryConfig()
	channel := NewRetryableChannel(baseCh, config)

	notification := &Notification{
		Title:   "Benchmark Test",
		Message: "This is a benchmark notification",
		Type:    NotificationTypeInfo,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = channel.Send(context.Background(), notification)
	}
}

func BenchmarkRetryableChannel_SendWithRetries(b *testing.B) {
	attempts := 0
	baseCh := &retryMockChannel{
		sendFunc: func(ctx context.Context, notif *Notification) error {
			attempts++
			if attempts%3 == 0 {
				return nil
			}
			return fmt.Errorf("temporary error")
		},
	}

	config := RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
		BackoffFactor:  2.0,
		Enabled:        true,
	}
	channel := NewRetryableChannel(baseCh, config)

	notification := &Notification{
		Title:   "Benchmark Test",
		Message: "This is a benchmark notification",
		Type:    NotificationTypeInfo,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		attempts = 0
		_ = channel.Send(context.Background(), notification)
	}
}

// Benchmark rate limiting

func BenchmarkRateLimiter_Allow(b *testing.B) {
	limiter := NewRateLimiter(1000, 1*time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow()
	}
}

func BenchmarkRateLimitedChannel_Send(b *testing.B) {
	baseCh := &mockChannel{name: "mock"}
	limiter := NewRateLimiter(1000, 1*time.Second)
	channel := NewRateLimitedChannel(baseCh, limiter)

	notification := &Notification{
		Title:   "Benchmark Test",
		Message: "This is a benchmark notification",
		Type:    NotificationTypeInfo,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = channel.Send(context.Background(), notification)
	}
}

// Benchmark queue operations

func BenchmarkNotificationQueue_Enqueue(b *testing.B) {
	engine := NewNotificationEngine()
	queue := NewNotificationQueue(engine, 1, 100000)

	notification := &Notification{
		Title:   "Benchmark Test",
		Message: "This is a benchmark notification",
		Type:    NotificationTypeInfo,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = queue.Enqueue(notification, []string{"test"}, 3)
	}
}

func BenchmarkNotificationQueue_Dequeue(b *testing.B) {
	engine := NewNotificationEngine()
	queue := NewNotificationQueue(engine, 1, 100000)

	notification := &Notification{
		Title:   "Benchmark Test",
		Message: "This is a benchmark notification",
		Type:    NotificationTypeInfo,
	}

	// Pre-fill queue
	for i := 0; i < b.N; i++ {
		_ = queue.Enqueue(notification, []string{"test"}, 3)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		queue.Dequeue()
	}
}

func BenchmarkNotificationQueue_Throughput(b *testing.B) {
	engine := NewNotificationEngine()
	mockCh := &mockChannel{name: "mock"}
	engine.RegisterChannel(mockCh)

	queue := NewNotificationQueue(engine, 10, 100000)
	queue.Start()
	defer queue.Stop()

	notification := &Notification{
		Title:   "Benchmark Test",
		Message: "This is a benchmark notification",
		Type:    NotificationTypeInfo,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = queue.Enqueue(notification, []string{"mock"}, 3)
	}

	// Wait for queue to drain
	for !queue.IsEmpty() {
		time.Sleep(1 * time.Millisecond)
	}
}

// Benchmark metrics

func BenchmarkMetrics_RecordSent(b *testing.B) {
	metrics := NewMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordSent("test-channel", 100*time.Millisecond)
	}
}

func BenchmarkMetrics_RecordFailed(b *testing.B) {
	metrics := NewMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordFailed("test-channel")
	}
}

func BenchmarkMetrics_GetMetrics(b *testing.B) {
	metrics := NewMetrics()
	metrics.RecordSent("test", 100*time.Millisecond)
	metrics.RecordFailed("test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = metrics.GetMetrics()
	}
}

func BenchmarkMetrics_GetSuccessRate(b *testing.B) {
	metrics := NewMetrics()
	metrics.RecordSent("test", 100*time.Millisecond)
	metrics.RecordFailed("test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = metrics.GetSuccessRate()
	}
}

// Benchmark concurrent operations

func BenchmarkNotificationEngine_ConcurrentSends(b *testing.B) {
	engine := NewNotificationEngine()
	mockCh := &mockChannel{name: "mock"}
	engine.RegisterChannel(mockCh)

	notification := &Notification{
		Title:   "Benchmark Test",
		Message: "This is a benchmark notification",
		Type:    NotificationTypeInfo,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = engine.SendDirect(context.Background(), notification, []string{"mock"})
		}
	})
}

func BenchmarkMetrics_ConcurrentWrites(b *testing.B) {
	metrics := NewMetrics()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			metrics.RecordSent("test", 100*time.Millisecond)
		}
	})
}

func BenchmarkRateLimiter_ConcurrentAllow(b *testing.B) {
	limiter := NewRateLimiter(1000, 1*time.Second)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			limiter.Allow()
		}
	})
}

func BenchmarkNotificationQueue_ConcurrentEnqueue(b *testing.B) {
	engine := NewNotificationEngine()
	queue := NewNotificationQueue(engine, 10, 1000000)

	notification := &Notification{
		Title:   "Benchmark Test",
		Message: "This is a benchmark notification",
		Type:    NotificationTypeInfo,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = queue.Enqueue(notification, []string{"test"}, 3)
		}
	})
}

// Benchmark memory allocations

func BenchmarkNotification_Creation(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = &Notification{
			Title:    "Test",
			Message:  "Test message",
			Type:     NotificationTypeInfo,
			Priority: NotificationPriorityMedium,
			Metadata: map[string]interface{}{
				"key": "value",
			},
		}
	}
}

func BenchmarkEngine_SendDirect_Allocations(b *testing.B) {
	engine := NewNotificationEngine()
	mockCh := &mockChannel{name: "mock"}
	engine.RegisterChannel(mockCh)

	notification := &Notification{
		Title:   "Benchmark Test",
		Message: "This is a benchmark notification",
		Type:    NotificationTypeInfo,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = engine.SendDirect(context.Background(), notification, []string{"mock"})
	}
}

// Benchmark large-scale operations

func BenchmarkEngine_1000Channels(b *testing.B) {
	engine := NewNotificationEngine()

	// Register 1000 channels
	for i := 0; i < 1000; i++ {
		mockCh := &mockChannel{name: fmt.Sprintf("mock-%d", i)}
		engine.RegisterChannel(mockCh)
	}

	notification := &Notification{
		Title:   "Benchmark Test",
		Message: "This is a benchmark notification",
		Type:    NotificationTypeInfo,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.SendDirect(context.Background(), notification, []string{"mock-500"})
	}
}

func BenchmarkQueue_HighVolume_10Workers(b *testing.B) {
	engine := NewNotificationEngine()
	mockCh := &mockChannel{name: "mock"}
	engine.RegisterChannel(mockCh)

	queue := NewNotificationQueue(engine, 10, 1000000)
	queue.Start()
	defer queue.Stop()

	notification := &Notification{
		Title:   "Benchmark Test",
		Message: "This is a benchmark notification",
		Type:    NotificationTypeInfo,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = queue.Enqueue(notification, []string{"mock"}, 3)
	}

	// Wait for processing
	for !queue.IsEmpty() {
		time.Sleep(1 * time.Millisecond)
	}
}

// Benchmark complex scenarios

func BenchmarkCompleteStack_WithRetryAndRateLimit(b *testing.B) {
	engine := NewNotificationEngine()

	// Create base channel
	baseCh := &mockChannel{name: "mock"}

	// Add retry
	retryConfig := DefaultRetryConfig()
	retryConfig.InitialBackoff = 1 * time.Millisecond
	retryableChannel := NewRetryableChannel(baseCh, retryConfig)

	// Add rate limiting
	limiter := NewRateLimiter(1000, 1*time.Second)
	rateLimitedChannel := NewRateLimitedChannel(retryableChannel, limiter)

	engine.RegisterChannel(rateLimitedChannel)

	notification := &Notification{
		Title:   "Benchmark Test",
		Message: "This is a benchmark notification",
		Type:    NotificationTypeInfo,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.SendDirect(context.Background(), notification, []string{"mock"})
	}
}

func BenchmarkCompleteStack_WithQueueAndMetrics(b *testing.B) {
	engine := NewNotificationEngine()
	mockCh := &mockChannel{name: "mock"}
	engine.RegisterChannel(mockCh)

	queue := NewNotificationQueue(engine, 10, 100000)
	queue.Start()
	defer queue.Stop()

	metrics := NewMetrics()

	notification := &Notification{
		Title:   "Benchmark Test",
		Message: "This is a benchmark notification",
		Type:    NotificationTypeInfo,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		_ = queue.Enqueue(notification, []string{"mock"}, 3)
		metrics.RecordSent("mock", time.Since(start))
	}

	// Wait for processing
	for !queue.IsEmpty() {
		time.Sleep(1 * time.Millisecond)
	}
}

// Benchmark parallel processing with multiple workers

func BenchmarkQueue_Parallel_1Worker(b *testing.B) {
	benchmarkQueueParallel(b, 1)
}

func BenchmarkQueue_Parallel_5Workers(b *testing.B) {
	benchmarkQueueParallel(b, 5)
}

func BenchmarkQueue_Parallel_10Workers(b *testing.B) {
	benchmarkQueueParallel(b, 10)
}

func BenchmarkQueue_Parallel_20Workers(b *testing.B) {
	benchmarkQueueParallel(b, 20)
}

func benchmarkQueueParallel(b *testing.B, workers int) {
	engine := NewNotificationEngine()
	mockCh := &mockChannel{name: "mock"}
	engine.RegisterChannel(mockCh)

	queue := NewNotificationQueue(engine, workers, 1000000)
	queue.Start()
	defer queue.Stop()

	notification := &Notification{
		Title:   "Benchmark Test",
		Message: "This is a benchmark notification",
		Type:    NotificationTypeInfo,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = queue.Enqueue(notification, []string{"mock"}, 3)
		}
	})

	// Wait for processing
	for !queue.IsEmpty() {
		time.Sleep(1 * time.Millisecond)
	}
}

// Benchmark notification with different payload sizes

func BenchmarkEngine_SmallPayload(b *testing.B) {
	engine := NewNotificationEngine()
	mockCh := &mockChannel{name: "mock"}
	engine.RegisterChannel(mockCh)

	notification := &Notification{
		Title:   "Test",
		Message: "Short message",
		Type:    NotificationTypeInfo,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.SendDirect(context.Background(), notification, []string{"mock"})
	}
}

func BenchmarkEngine_MediumPayload(b *testing.B) {
	engine := NewNotificationEngine()
	mockCh := &mockChannel{name: "mock"}
	engine.RegisterChannel(mockCh)

	notification := &Notification{
		Title:   "Medium Test Notification",
		Message: "This is a medium-sized notification message with some additional context and details about what happened in the system.",
		Type:    NotificationTypeInfo,
		Metadata: map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.SendDirect(context.Background(), notification, []string{"mock"})
	}
}

func BenchmarkEngine_LargePayload(b *testing.B) {
	engine := NewNotificationEngine()
	mockCh := &mockChannel{name: "mock"}
	engine.RegisterChannel(mockCh)

	largeMessage := ""
	for i := 0; i < 100; i++ {
		largeMessage += "This is a large notification message with lots of content. "
	}

	metadata := make(map[string]interface{})
	for i := 0; i < 50; i++ {
		metadata[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
	}

	notification := &Notification{
		Title:    "Large Test Notification with Extended Title",
		Message:  largeMessage,
		Type:     NotificationTypeInfo,
		Priority: NotificationPriorityHigh,
		Metadata: metadata,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.SendDirect(context.Background(), notification, []string{"mock"})
	}
}

// Benchmark rule evaluation

func BenchmarkEngine_RuleEvaluation_NoRules(b *testing.B) {
	engine := NewNotificationEngine()
	mockCh := &mockChannel{name: "mock"}
	engine.RegisterChannel(mockCh)

	notification := &Notification{
		Title:   "Test",
		Message: "Test",
		Type:    NotificationTypeInfo,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.SendNotification(context.Background(), notification)
	}
}

func BenchmarkEngine_RuleEvaluation_10Rules(b *testing.B) {
	engine := NewNotificationEngine()
	mockCh := &mockChannel{name: "mock"}
	engine.RegisterChannel(mockCh)

	// Add 10 rules
	for i := 0; i < 10; i++ {
		engine.AddRule(NotificationRule{
			Name:      fmt.Sprintf("Rule %d", i),
			Condition: fmt.Sprintf("type==type%d", i),
			Channels:  []string{"mock"},
			Enabled:   true,
		})
	}

	notification := &Notification{
		Title:   "Test",
		Message: "Test",
		Type:    NotificationTypeInfo,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.SendNotification(context.Background(), notification)
	}
}

// Benchmark stress test scenarios

func BenchmarkStress_100ConcurrentSends(b *testing.B) {
	engine := NewNotificationEngine()
	mockCh := &mockChannel{name: "mock"}
	engine.RegisterChannel(mockCh)

	notification := &Notification{
		Title:   "Stress Test",
		Message: "Stress test message",
		Type:    NotificationTypeInfo,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < 100; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = engine.SendDirect(context.Background(), notification, []string{"mock"})
			}()
		}
		wg.Wait()
	}
}

func BenchmarkStress_RateLimiterContention(b *testing.B) {
	limiter := NewRateLimiter(100, 1*time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < 50; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				limiter.Allow()
			}()
		}
		wg.Wait()
	}
}
