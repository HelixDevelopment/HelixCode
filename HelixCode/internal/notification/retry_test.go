package notification

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Mock channel for retry testing
type retryMockChannel struct {
	sendFunc      func(ctx context.Context, notif *Notification) error
	getNameFunc   func() string
	isEnabledFunc func() bool
	getConfigFunc func() map[string]interface{}
}

func (m *retryMockChannel) Send(ctx context.Context, notification *Notification) error {
	if m.sendFunc != nil {
		return m.sendFunc(ctx, notification)
	}
	return nil
}

func (m *retryMockChannel) GetName() string {
	if m.getNameFunc != nil {
		return m.getNameFunc()
	}
	return "mock"
}

func (m *retryMockChannel) IsEnabled() bool {
	if m.isEnabledFunc != nil {
		return m.isEnabledFunc()
	}
	return true
}

func (m *retryMockChannel) GetConfig() map[string]interface{} {
	if m.getConfigFunc != nil {
		return m.getConfigFunc()
	}
	return map[string]interface{}{}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	assert.True(t, config.Enabled)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 1*time.Second, config.InitialBackoff)
	assert.Equal(t, 30*time.Second, config.MaxBackoff)
	assert.Equal(t, 2.0, config.BackoffFactor)
}

func TestRetryableChannel_Send_Success(t *testing.T) {
	mockCh := &retryMockChannel{sendFunc: func(ctx context.Context, notif *Notification) error {
		return nil
	}}

	config := DefaultRetryConfig()
	retryable := NewRetryableChannel(mockCh, config)

	notif := &Notification{
		Title:   "Test",
		Message: "Test",
		Type:    NotificationTypeInfo,
	}

	err := retryable.Send(context.Background(), notif)
	assert.NoError(t, err)

	stats := retryable.GetStats()
	assert.Equal(t, int64(1), stats.TotalAttempts)
	assert.Equal(t, int64(1), stats.SuccessfulSends)
	assert.Equal(t, int64(0), stats.FailedSends)
	assert.Equal(t, int64(0), stats.Retries)
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil error", err: nil, want: false},
		{name: "timeout error", err: errors.New("connection timeout"), want: true},
		{name: "connection refused", err: errors.New("connection refused"), want: true},
		{name: "503 error", err: errors.New("server returned 503"), want: true},
		{name: "rate limit error", err: errors.New("rate limit exceeded"), want: true},
		{name: "non-retryable error", err: errors.New("invalid credentials"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestRetryableChannel_GetStats(t *testing.T) {
	mockCh := &retryMockChannel{sendFunc: func(ctx context.Context, notif *Notification) error {
		return nil
	}}

	config := DefaultRetryConfig()
	retryable := NewRetryableChannel(mockCh, config)

	stats := retryable.GetStats()
	assert.Equal(t, int64(0), stats.TotalAttempts)
}
