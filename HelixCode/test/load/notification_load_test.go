package load

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"dev.helix.code/internal/event"
	"dev.helix.code/internal/notification"
)

// Load test for notification system under realistic production scenarios

func TestLoad_1000NotificationsPerSecond(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	engine := notification.NewNotificationEngine()
	mockCh := &mockChannel{name: "mock"}
	engine.RegisterChannel(mockCh)

	queue := notification.NewNotificationQueue(engine, 10, 10000)
	queue.Start()
	defer queue.Stop()

	// Test parameters
	duration := 10 * time.Second
	targetRate := 1000 // notifications per second
	totalNotifications := int(duration.Seconds()) * targetRate

	notif := &notification.Notification{
		Title:   "Load Test",
		Message: "Load test notification",
		Type:    notification.NotificationTypeInfo,
	}

	// Track metrics
	var sent int64
	var failed int64
	startTime := time.Now()

	// Send notifications at target rate
	ticker := time.NewTicker(time.Second / time.Duration(targetRate))
	defer ticker.Stop()

	timeout := time.After(duration)
	count := 0

	for {
		select {
		case <-timeout:
			goto done
		case <-ticker.C:
			err := queue.Enqueue(notif, []string{"mock"}, 3)
			if err != nil {
				atomic.AddInt64(&failed, 1)
			} else {
				atomic.AddInt64(&sent, 1)
			}
			count++
			if count >= totalNotifications {
				goto done
			}
		}
	}

done:
	// Wait for queue to drain
	deadline := time.Now().Add(30 * time.Second)
	for !queue.IsEmpty() && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}

	elapsed := time.Since(startTime)
	actualRate := float64(atomic.LoadInt64(&sent)) / elapsed.Seconds()

	t.Logf("Load Test Results:")
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Sent: %d", atomic.LoadInt64(&sent))
	t.Logf("  Failed: %d", atomic.LoadInt64(&failed))
	t.Logf("  Target Rate: %d/sec", targetRate)
	t.Logf("  Actual Rate: %.2f/sec", actualRate)
	t.Logf("  Queue Final Size: %d", queue.Size())

	// Verify acceptable performance
	if atomic.LoadInt64(&failed) > int64(totalNotifications/100) { // Allow 1% failure
		t.Errorf("Too many failures: %d (%.2f%%)", atomic.LoadInt64(&failed),
			float64(atomic.LoadInt64(&failed))/float64(totalNotifications)*100)
	}

	if actualRate < float64(targetRate)*0.9 { // Allow 10% deviation
		t.Errorf("Rate too low: %.2f/sec (target: %d/sec)", actualRate, targetRate)
	}
}

func TestLoad_ConcurrentChannels(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	engine := notification.NewNotificationEngine()

	// Register 10 channels
	numChannels := 10
	for i := 0; i < numChannels; i++ {
		ch := &mockChannel{name: fmt.Sprintf("mock-%d", i)}
		engine.RegisterChannel(ch)
	}

	// Test parameters
	duration := 5 * time.Second
	concurrentSenders := 20
	totalSent := int64(0)
	totalFailed := int64(0)

	notif := &notification.Notification{
		Title:   "Concurrent Load Test",
		Message: "Testing concurrent channel sends",
		Type:    notification.NotificationTypeInfo,
	}

	var wg sync.WaitGroup
	startTime := time.Now()

	// Start concurrent senders
	for i := 0; i < concurrentSenders; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			channelID := id % numChannels
			channelName := fmt.Sprintf("mock-%d", channelID)
			timeout := time.After(duration)

			for {
				select {
				case <-timeout:
					return
				default:
					err := engine.SendDirect(context.Background(), notif, []string{channelName})
					if err != nil {
						atomic.AddInt64(&totalFailed, 1)
					} else {
						atomic.AddInt64(&totalSent, 1)
					}
					time.Sleep(10 * time.Millisecond) // Throttle
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	t.Logf("Concurrent Channel Test Results:")
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Channels: %d", numChannels)
	t.Logf("  Concurrent Senders: %d", concurrentSenders)
	t.Logf("  Total Sent: %d", totalSent)
	t.Logf("  Total Failed: %d", totalFailed)
	t.Logf("  Rate: %.2f/sec", float64(totalSent)/elapsed.Seconds())
	t.Logf("  Success Rate: %.2f%%", float64(totalSent)/float64(totalSent+totalFailed)*100)
}

func TestLoad_QueueSaturation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	engine := notification.NewNotificationEngine()
	mockCh := &mockChannel{name: "mock", delay: 50 * time.Millisecond} // Slow channel
	engine.RegisterChannel(mockCh)

	// Small queue to test saturation
	queue := notification.NewNotificationQueue(engine, 5, 100)
	queue.Start()
	defer queue.Stop()

	notif := &notification.Notification{
		Title:   "Saturation Test",
		Message: "Testing queue saturation",
		Type:    notification.NotificationTypeInfo,
	}

	// Rapid fire notifications to saturate queue
	var enqueued int64
	var rejected int64

	for i := 0; i < 200; i++ {
		err := queue.Enqueue(notif, []string{"mock"}, 3)
		if err != nil {
			atomic.AddInt64(&rejected, 1)
		} else {
			atomic.AddInt64(&enqueued, 1)
		}
		time.Sleep(1 * time.Millisecond)
	}

	// Wait for processing
	time.Sleep(15 * time.Second)

	stats := queue.GetStats()

	t.Logf("Queue Saturation Test Results:")
	t.Logf("  Enqueued: %d", enqueued)
	t.Logf("  Rejected: %d", rejected)
	t.Logf("  Succeeded: %d", stats.Succeeded)
	t.Logf("  Failed: %d", stats.Failed)
	t.Logf("  Final Queue Size: %d", queue.Size())

	// Verify queue handled saturation gracefully
	if stats.Succeeded+stats.Failed < enqueued-10 { // Allow small margin
		t.Errorf("Many notifications lost: enqueued=%d, processed=%d",
			enqueued, stats.Succeeded+stats.Failed)
	}
}

func TestLoad_RetryStorm(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	// Create channel that fails 50% of the time
	attempts := int64(0)
	baseCh := &mockChannel{
		name: "flaky",
		sendFunc: func(ctx context.Context, notif *notification.Notification) error {
			count := atomic.AddInt64(&attempts, 1)
			if count%2 == 0 {
				return fmt.Errorf("simulated failure")
			}
			return nil
		},
	}

	retryConfig := notification.RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		BackoffFactor:  2.0,
		Enabled:        true,
	}

	retryChannel := notification.NewRetryableChannel(baseCh, retryConfig)

	engine := notification.NewNotificationEngine()
	engine.RegisterChannel(retryChannel)

	notif := &notification.Notification{
		Title:   "Retry Storm Test",
		Message: "Testing retry behavior under load",
		Type:    notification.NotificationTypeInfo,
	}

	// Send many notifications concurrently
	numGoroutines := 20
	notificationsPerGoroutine := 50

	var wg sync.WaitGroup
	var succeeded int64
	var failed int64
	startTime := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < notificationsPerGoroutine; j++ {
				err := engine.SendDirect(context.Background(), notif, []string{"flaky"})
				if err != nil {
					atomic.AddInt64(&failed, 1)
				} else {
					atomic.AddInt64(&succeeded, 1)
				}
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	total := numGoroutines * notificationsPerGoroutine
	successRate := float64(succeeded) / float64(total) * 100

	t.Logf("Retry Storm Test Results:")
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Total Attempts: %d", total)
	t.Logf("  Succeeded: %d", succeeded)
	t.Logf("  Failed: %d", failed)
	t.Logf("  Success Rate: %.2f%%", successRate)
	t.Logf("  Total Send Attempts: %d", attempts)
	t.Logf("  Avg Attempts Per Notification: %.2f", float64(attempts)/float64(total))

	// With 50% failure rate and 3 retries, we should get >50% success
	if successRate < 50.0 {
		t.Errorf("Success rate too low: %.2f%%", successRate)
	}
}

func TestLoad_RateLimiterStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	baseCh := &mockChannel{name: "mock"}
	limiter := notification.NewRateLimiter(100, 1*time.Second) // 100 per second
	rateLimitedChannel := notification.NewRateLimitedChannel(baseCh, limiter)

	engine := notification.NewNotificationEngine()
	engine.RegisterChannel(rateLimitedChannel)

	notif := &notification.Notification{
		Title:   "Rate Limiter Stress Test",
		Message: "Testing rate limiter under load",
		Type:    notification.NotificationTypeInfo,
	}

	// Try to send 500 notifications (5x rate limit)
	numNotifications := 500
	var sent int64
	startTime := time.Now()

	for i := 0; i < numNotifications; i++ {
		err := engine.SendDirect(context.Background(), notif, []string{"mock"})
		if err == nil {
			atomic.AddInt64(&sent, 1)
		}
	}

	elapsed := time.Since(startTime)
	actualRate := float64(sent) / elapsed.Seconds()

	t.Logf("Rate Limiter Stress Test Results:")
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Attempted: %d", numNotifications)
	t.Logf("  Sent: %d", sent)
	t.Logf("  Rate Limit: 100/sec")
	t.Logf("  Actual Rate: %.2f/sec", actualRate)

	// Rate should be near limit (allow some variance)
	if actualRate > 150.0 { // Should not exceed limit by much
		t.Errorf("Rate limiter not working: %.2f/sec (limit: 100/sec)", actualRate)
	}
}

func TestLoad_EventBusHighVolume(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	bus := event.NewEventBus(true) // Async mode

	// Subscribe handlers
	var processed int64
	for i := 0; i < 5; i++ {
		bus.Subscribe(event.EventTaskCreated, func(ctx context.Context, evt event.Event) error {
			atomic.AddInt64(&processed, 1)
			time.Sleep(1 * time.Millisecond) // Simulate processing
			return nil
		})
	}

	// Publish events at high rate
	numEvents := 10000
	var published int64
	startTime := time.Now()

	for i := 0; i < numEvents; i++ {
		evt := event.Event{
			Type:     event.EventTaskCreated,
			Severity: event.SeverityInfo,
			Source:   "load-test",
			TaskID:   fmt.Sprintf("task-%d", i),
		}

		err := bus.Publish(context.Background(), evt)
		if err == nil {
			atomic.AddInt64(&published, 1)
		}
	}

	// Wait for processing (with timeout)
	deadline := time.Now().Add(30 * time.Second)
	for atomic.LoadInt64(&processed) < int64(numEvents*5) && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}

	elapsed := time.Since(startTime)

	t.Logf("Event Bus High Volume Test Results:")
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Published: %d", published)
	t.Logf("  Processed: %d (expected: %d)", processed, numEvents*5)
	t.Logf("  Publish Rate: %.2f/sec", float64(published)/elapsed.Seconds())

	// Verify all events were processed
	expectedProcessed := int64(numEvents * 5)
	if processed < expectedProcessed-100 { // Allow small margin for async
		t.Errorf("Not all events processed: %d/%d", processed, expectedProcessed)
	}
}

func TestLoad_MetricsUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	metrics := notification.NewMetrics()

	// Simulate high-volume metric recording
	numGoroutines := 50
	recordsPerGoroutine := 1000

	var wg sync.WaitGroup
	startTime := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			channelName := fmt.Sprintf("channel-%d", id%10)

			for j := 0; j < recordsPerGoroutine; j++ {
				if j%2 == 0 {
					metrics.RecordSent(channelName, time.Duration(j)*time.Millisecond)
				} else {
					metrics.RecordFailed(channelName)
				}

				if j%5 == 0 {
					metrics.RecordRetry(channelName)
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	// Get metrics
	current := metrics.GetMetrics()
	successRate := metrics.GetSuccessRate()

	totalRecords := numGoroutines * recordsPerGoroutine

	t.Logf("Metrics Under Load Test Results:")
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Total Records: %d", totalRecords)
	t.Logf("  Total Sent: %d", current.TotalSent)
	t.Logf("  Total Failed: %d", current.TotalFailed)
	t.Logf("  Total Retries: %d", current.TotalRetries)
	t.Logf("  Success Rate: %.2f%%", successRate)
	t.Logf("  Recording Rate: %.2f/sec", float64(totalRecords)/elapsed.Seconds())

	// Verify metrics are accurate
	if current.TotalSent+current.TotalFailed != int64(totalRecords) {
		t.Errorf("Metric counts incorrect: sent=%d, failed=%d, total=%d",
			current.TotalSent, current.TotalFailed, totalRecords)
	}
}

// Mock channel for testing
type mockChannel struct {
	name     string
	delay    time.Duration
	sendFunc func(ctx context.Context, notif *notification.Notification) error
}

func (m *mockChannel) Send(ctx context.Context, notif *notification.Notification) error {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	if m.sendFunc != nil {
		return m.sendFunc(ctx, notif)
	}

	return nil
}

func (m *mockChannel) GetName() string {
	return m.name
}

func (m *mockChannel) IsEnabled() bool {
	return true
}

func (m *mockChannel) GetConfig() map[string]interface{} {
	return map[string]interface{}{"name": m.name}
}
