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

func TestSlackIntegration(t *testing.T) {
	// Start mock Slack server
	mockServer := testutil.NewMockSlackServer()
	defer mockServer.Close()

	// Create Slack channel with mock server URL
	channel := notification.NewSlackChannel(
		mockServer.URL,
		"#test-channel",
		"TestBot",
	)

	tests := []struct {
		name         string
		notification *notification.Notification
		wantRequests int
		checkContent func(*testing.T, testutil.SlackRequest)
	}{
		{
			name: "info notification",
			notification: &notification.Notification{
				Title:   "Test Info",
				Message: "This is a test",
				Type:    notification.NotificationTypeInfo,
			},
			wantRequests: 1,
			checkContent: func(t *testing.T, req testutil.SlackRequest) {
				assert.Equal(t, "#test-channel", req.Channel)
				assert.Equal(t, "TestBot", req.Username)
				assert.Contains(t, req.Text, "Test Info")
				assert.Contains(t, req.Text, "This is a test")
				assert.Equal(t, ":information_source:", req.IconEmoji)
			},
		},
		{
			name: "error notification",
			notification: &notification.Notification{
				Title:   "Test Error",
				Message: "An error occurred",
				Type:    notification.NotificationTypeError,
			},
			wantRequests: 1,
			checkContent: func(t *testing.T, req testutil.SlackRequest) {
				assert.Equal(t, "#test-channel", req.Channel)
				assert.Equal(t, "TestBot", req.Username)
				assert.Contains(t, req.Text, "Test Error")
				assert.Contains(t, req.Text, "An error occurred")
				assert.Equal(t, ":x:", req.IconEmoji)
			},
		},
		{
			name: "success notification",
			notification: &notification.Notification{
				Title:   "Test Success",
				Message: "Operation completed successfully",
				Type:    notification.NotificationTypeSuccess,
			},
			wantRequests: 1,
			checkContent: func(t *testing.T, req testutil.SlackRequest) {
				assert.Equal(t, "#test-channel", req.Channel)
				assert.Contains(t, req.Text, "Test Success")
				assert.Equal(t, ":white_check_mark:", req.IconEmoji)
			},
		},
		{
			name: "warning notification",
			notification: &notification.Notification{
				Title:   "Test Warning",
				Message: "Warning message",
				Type:    notification.NotificationTypeWarning,
			},
			wantRequests: 1,
			checkContent: func(t *testing.T, req testutil.SlackRequest) {
				assert.Equal(t, "#test-channel", req.Channel)
				assert.Contains(t, req.Text, "Test Warning")
				assert.Equal(t, ":warning:", req.IconEmoji)
			},
		},
		{
			name: "alert notification",
			notification: &notification.Notification{
				Title:   "Test Alert",
				Message: "Critical alert",
				Type:    notification.NotificationTypeAlert,
			},
			wantRequests: 1,
			checkContent: func(t *testing.T, req testutil.SlackRequest) {
				assert.Equal(t, "#test-channel", req.Channel)
				assert.Contains(t, req.Text, "Test Alert")
				assert.Equal(t, ":rotating_light:", req.IconEmoji)
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

func TestSlackIntegration_WithNotificationEngine(t *testing.T) {
	// Start mock Slack server
	mockServer := testutil.NewMockSlackServer()
	defer mockServer.Close()

	// Create notification engine
	engine := notification.NewNotificationEngine()

	// Create and register Slack channel
	slackChannel := notification.NewSlackChannel(mockServer.URL, "#alerts", "HelixBot")
	err := engine.RegisterChannel(slackChannel)
	require.NoError(t, err)

	// Add a rule: send all error notifications to Slack
	rule := notification.NotificationRule{
		Name:      "Error Alerts",
		Condition: "type==error",
		Channels:  []string{"slack"},
		Priority:  notification.NotificationPriorityHigh,
		Enabled:   true,
	}
	engine.AddRule(rule)

	// Send an error notification
	notification := &notification.Notification{
		Title:   "Critical Error",
		Message: "Database connection failed",
		Type:    notification.NotificationTypeError,
		Metadata: map[string]interface{}{
			"component": "database",
			"error":     "connection timeout",
		},
	}

	err = engine.SendNotification(context.Background(), notification)
	require.NoError(t, err)

	// Verify Slack received the notification
	requests := mockServer.GetRequests()
	require.Equal(t, 1, len(requests), "Should have received 1 Slack notification")

	slackMsg := requests[0]
	assert.Equal(t, "#alerts", slackMsg.Channel)
	assert.Equal(t, "HelixBot", slackMsg.Username)
	assert.Contains(t, slackMsg.Text, "Critical Error")
	assert.Contains(t, slackMsg.Text, "Database connection failed")
	assert.Equal(t, ":x:", slackMsg.IconEmoji)
}

func TestSlackIntegration_MultipleNotifications(t *testing.T) {
	mockServer := testutil.NewMockSlackServer()
	defer mockServer.Close()

	channel := notification.NewSlackChannel(mockServer.URL, "#test", "bot")

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

func TestSlackIntegration_ChannelDisabled(t *testing.T) {
	// Create channel with empty webhook (disabled)
	channel := notification.NewSlackChannel("", "#test", "bot")

	notif := &notification.Notification{
		Title:   "Test",
		Message: "Test",
		Type:    notification.NotificationTypeInfo,
	}

	err := channel.Send(context.Background(), notif)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}
