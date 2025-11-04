//go:build integration

package integration

import (
	"context"
	"testing"

	"dev.helix.code/internal/notification"
	"dev.helix.code/internal/notification/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscordIntegration(t *testing.T) {
	// Start mock Discord server
	mockServer := testutil.NewMockDiscordServer()
	defer mockServer.Close()

	// Create Discord channel with mock server URL
	channel := notification.NewDiscordChannel(mockServer.URL)

	tests := []struct {
		name         string
		notification *notification.Notification
		wantRequests int
		checkContent func(*testing.T, testutil.DiscordRequest)
	}{
		{
			name: "info notification",
			notification: &notification.Notification{
				Title:   "Info Message",
				Message: "This is an informational message",
				Type:    notification.NotificationTypeInfo,
			},
			wantRequests: 1,
			checkContent: func(t *testing.T, req testutil.DiscordRequest) {
				assert.Contains(t, req.Content, "Info Message")
				assert.Contains(t, req.Content, "This is an informational message")
			},
		},
		{
			name: "error notification",
			notification: &notification.Notification{
				Title:   "Error Alert",
				Message: "An error has occurred",
				Type:    notification.NotificationTypeError,
			},
			wantRequests: 1,
			checkContent: func(t *testing.T, req testutil.DiscordRequest) {
				assert.Contains(t, req.Content, "Error Alert")
				assert.Contains(t, req.Content, "An error has occurred")
			},
		},
		{
			name: "success notification",
			notification: &notification.Notification{
				Title:   "Deployment Successful",
				Message: "Application deployed to production",
				Type:    notification.NotificationTypeSuccess,
			},
			wantRequests: 1,
			checkContent: func(t *testing.T, req testutil.DiscordRequest) {
				assert.Contains(t, req.Content, "Deployment Successful")
				assert.Contains(t, req.Content, "Application deployed to production")
			},
		},
		{
			name: "warning notification",
			notification: &notification.Notification{
				Title:   "Resource Warning",
				Message: "Disk space running low",
				Type:    notification.NotificationTypeWarning,
			},
			wantRequests: 1,
			checkContent: func(t *testing.T, req testutil.DiscordRequest) {
				assert.Contains(t, req.Content, "Resource Warning")
				assert.Contains(t, req.Content, "Disk space running low")
			},
		},
		{
			name: "alert notification",
			notification: &notification.Notification{
				Title:   "Critical Alert",
				Message: "Service is down",
				Type:    notification.NotificationTypeAlert,
			},
			wantRequests: 1,
			checkContent: func(t *testing.T, req testutil.DiscordRequest) {
				assert.Contains(t, req.Content, "Critical Alert")
				assert.Contains(t, req.Content, "Service is down")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer.Reset()

			err := channel.Send(context.Background(), tt.notification)
			require.NoError(t, err)

			requests := mockServer.GetRequests()
			assert.Equal(t, tt.wantRequests, len(requests), "Expected %d requests, got %d", tt.wantRequests, len(requests))

			if len(requests) > 0 && tt.checkContent != nil {
				tt.checkContent(t, requests[0])
			}
		})
	}
}

func TestDiscordIntegration_WithNotificationEngine(t *testing.T) {
	// Start mock Discord server
	mockServer := testutil.NewMockDiscordServer()
	defer mockServer.Close()

	// Create notification engine
	engine := notification.NewNotificationEngine()

	// Create and register Discord channel
	discordChannel := notification.NewDiscordChannel(mockServer.URL)
	err := engine.RegisterChannel(discordChannel)
	require.NoError(t, err)

	// Add a rule: send all warnings to Discord
	rule := notification.NotificationRule{
		Name:      "Warning Alerts",
		Condition: "type==warning",
		Channels:  []string{"discord"},
		Priority:  notification.NotificationPriorityMedium,
		Enabled:   true,
	}
	engine.AddRule(rule)

	// Send a warning notification
	notification := &notification.Notification{
		Title:   "High Memory Usage",
		Message: "Memory usage is at 85%",
		Type:    notification.NotificationTypeWarning,
		Metadata: map[string]interface{}{
			"server":       "web-01",
			"memory_usage": "85%",
		},
	}

	err = engine.SendNotification(context.Background(), notification)
	require.NoError(t, err)

	// Verify Discord received the notification
	requests := mockServer.GetRequests()
	require.Equal(t, 1, len(requests), "Should have received 1 Discord notification")

	discordMsg := requests[0]
	assert.Contains(t, discordMsg.Content, "High Memory Usage")
	assert.Contains(t, discordMsg.Content, "Memory usage is at 85%")
}

func TestDiscordIntegration_MultipleNotifications(t *testing.T) {
	mockServer := testutil.NewMockDiscordServer()
	defer mockServer.Close()

	channel := notification.NewDiscordChannel(mockServer.URL)

	// Send multiple notifications
	for i := 0; i < 5; i++ {
		notif := &notification.Notification{
			Title:   "Notification",
			Message: "Message",
			Type:    notification.NotificationTypeInfo,
		}
		err := channel.Send(context.Background(), notif)
		require.NoError(t, err)
	}

	// Verify all were received
	assert.Equal(t, 5, mockServer.GetRequestCount())
}

func TestDiscordIntegration_ChannelDisabled(t *testing.T) {
	// Create channel with empty webhook (disabled)
	channel := notification.NewDiscordChannel("")

	notif := &notification.Notification{
		Title:   "Test",
		Message: "Test",
		Type:    notification.NotificationTypeInfo,
	}

	err := channel.Send(context.Background(), notif)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestDiscordIntegration_WithMetadata(t *testing.T) {
	mockServer := testutil.NewMockDiscordServer()
	defer mockServer.Close()

	channel := notification.NewDiscordChannel(mockServer.URL)

	notif := &notification.Notification{
		Title:   "Build Notification",
		Message: "Build #123 failed",
		Type:    notification.NotificationTypeError,
		Metadata: map[string]interface{}{
			"build_id":   "123",
			"branch":     "main",
			"commit":     "abc123",
			"failed_at":  "2025-11-04T10:30:00Z",
			"error_code": "BUILD_FAILED",
		},
	}

	err := channel.Send(context.Background(), notif)
	require.NoError(t, err)

	requests := mockServer.GetRequests()
	require.Equal(t, 1, len(requests))

	req := requests[0]
	assert.Contains(t, req.Content, "Build Notification")
	assert.Contains(t, req.Content, "Build #123 failed")
}

func TestDiscordIntegration_ConcurrentSending(t *testing.T) {
	mockServer := testutil.NewMockDiscordServer()
	defer mockServer.Close()

	channel := notification.NewDiscordChannel(mockServer.URL)

	// Send concurrent notifications
	done := make(chan bool)
	count := 10

	for i := 0; i < count; i++ {
		go func(id int) {
			notif := &notification.Notification{
				Title:   "Concurrent Test",
				Message: "Message",
				Type:    notification.NotificationTypeInfo,
			}
			err := channel.Send(context.Background(), notif)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < count; i++ {
		<-done
	}

	// Verify all were received
	assert.Equal(t, count, mockServer.GetRequestCount())
}

func TestDiscordIntegration_AllNotificationTypes(t *testing.T) {
	mockServer := testutil.NewMockDiscordServer()
	defer mockServer.Close()

	channel := notification.NewDiscordChannel(mockServer.URL)

	notificationTypes := []notification.NotificationType{
		notification.NotificationTypeInfo,
		notification.NotificationTypeSuccess,
		notification.NotificationTypeWarning,
		notification.NotificationTypeError,
		notification.NotificationTypeAlert,
	}

	for _, notifType := range notificationTypes {
		t.Run(string(notifType), func(t *testing.T) {
			mockServer.Reset()

			notif := &notification.Notification{
				Title:   "Test " + string(notifType),
				Message: "Testing " + string(notifType),
				Type:    notifType,
			}

			err := channel.Send(context.Background(), notif)
			require.NoError(t, err)

			requests := mockServer.GetRequests()
			require.Equal(t, 1, len(requests))

			req := requests[0]
			assert.Contains(t, req.Content, "Test "+string(notifType))
		})
	}
}
