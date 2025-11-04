package notification

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNotificationQueue(t *testing.T) {
	engine := NewNotificationEngine()
	queue := NewNotificationQueue(engine, 2, 100)

	assert.NotNil(t, queue)
	assert.Equal(t, 2, queue.workers)
	assert.Equal(t, 100, queue.maxSize)
	assert.True(t, queue.IsEmpty())
}

func TestNotificationQueue_Enqueue(t *testing.T) {
	engine := NewNotificationEngine()
	queue := NewNotificationQueue(engine, 1, 10)

	notif := &Notification{
		Title:   "Test",
		Message: "Test",
		Type:    NotificationTypeInfo,
	}

	err := queue.Enqueue(notif, []string{"test"}, 3)
	require.NoError(t, err)

	assert.Equal(t, 1, queue.Size())
	assert.False(t, queue.IsEmpty())

	stats := queue.GetStats()
	assert.Equal(t, int64(1), stats.Enqueued)
}

func TestNotificationQueue_EnqueueFull(t *testing.T) {
	engine := NewNotificationEngine()
	queue := NewNotificationQueue(engine, 1, 2)

	notif := &Notification{
		Title:   "Test",
		Message: "Test",
		Type:    NotificationTypeInfo,
	}

	// Fill queue
	queue.Enqueue(notif, []string{"test"}, 3)
	queue.Enqueue(notif, []string{"test"}, 3)

	// Should fail
	err := queue.Enqueue(notif, []string{"test"}, 3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "queue is full")
}

func TestNotificationQueue_Dequeue(t *testing.T) {
	engine := NewNotificationEngine()
	queue := NewNotificationQueue(engine, 1, 10)

	notif := &Notification{
		Title:   "Test",
		Message: "Test Message",
		Type:    NotificationTypeInfo,
	}

	queue.Enqueue(notif, []string{"test"}, 3)

	item := queue.Dequeue()
	require.NotNil(t, item)
	assert.Equal(t, "Test", item.Notification.Title)
	assert.Equal(t, []string{"test"}, item.Channels)
	assert.Equal(t, 3, item.MaxRetries)
	assert.Equal(t, 0, item.Attempts)

	assert.True(t, queue.IsEmpty())

	stats := queue.GetStats()
	assert.Equal(t, int64(1), stats.Dequeued)
}

func TestNotificationQueue_DequeueEmpty(t *testing.T) {
	engine := NewNotificationEngine()
	queue := NewNotificationQueue(engine, 1, 10)

	item := queue.Dequeue()
	assert.Nil(t, item)
}

func TestNotificationQueue_Clear(t *testing.T) {
	engine := NewNotificationEngine()
	queue := NewNotificationQueue(engine, 1, 10)

	notif := &Notification{
		Title:   "Test",
		Message: "Test",
		Type:    NotificationTypeInfo,
	}

	queue.Enqueue(notif, []string{"test"}, 3)
	queue.Enqueue(notif, []string{"test"}, 3)
	assert.Equal(t, 2, queue.Size())

	queue.Clear()
	assert.True(t, queue.IsEmpty())
}

func TestNotificationQueue_Worker(t *testing.T) {
	engine := NewNotificationEngine()

	// Register mock channel
	sent := false
	mockCh := &retryMockChannel{
		sendFunc: func(ctx context.Context, notif *Notification) error {
			sent = true
			return nil
		},
	}
	engine.RegisterChannel(mockCh)

	queue := NewNotificationQueue(engine, 1, 10)
	queue.Start()
	defer queue.Stop()

	notif := &Notification{
		Title:   "Test",
		Message: "Test",
		Type:    NotificationTypeInfo,
	}

	queue.Enqueue(notif, []string{"mock"}, 3)

	// Wait for worker to process
	time.Sleep(300 * time.Millisecond)

	assert.True(t, sent)
	assert.True(t, queue.IsEmpty())

	stats := queue.GetStats()
	assert.Equal(t, int64(1), stats.Succeeded)
}

func TestNotificationQueue_GetQueueItems(t *testing.T) {
	engine := NewNotificationEngine()
	queue := NewNotificationQueue(engine, 1, 10)

	notif1 := &Notification{Title: "Test 1", Message: "M1", Type: NotificationTypeInfo}
	notif2 := &Notification{Title: "Test 2", Message: "M2", Type: NotificationTypeInfo}

	queue.Enqueue(notif1, []string{"test"}, 3)
	queue.Enqueue(notif2, []string{"test"}, 3)

	items := queue.GetQueueItems()
	assert.Equal(t, 2, len(items))
	assert.Equal(t, "Test 1", items[0].Notification.Title)
	assert.Equal(t, "Test 2", items[1].Notification.Title)
}

func TestNotificationQueue_ResetStats(t *testing.T) {
	engine := NewNotificationEngine()
	queue := NewNotificationQueue(engine, 1, 10)

	notif := &Notification{
		Title:   "Test",
		Message: "Test",
		Type:    NotificationTypeInfo,
	}

	queue.Enqueue(notif, []string{"test"}, 3)

	stats := queue.GetStats()
	assert.Equal(t, int64(1), stats.Enqueued)

	queue.ResetStats()

	stats = queue.GetStats()
	assert.Equal(t, int64(0), stats.Enqueued)
}
