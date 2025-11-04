package notification

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSlackChannel(t *testing.T) {
	tests := []struct {
		name        string
		webhook     string
		channel     string
		username    string
		wantEnabled bool
	}{
		{
			name:        "valid configuration",
			webhook:     "https://hooks.slack.com/services/T/B/X",
			channel:     "#helix",
			username:    "bot",
			wantEnabled: true,
		},
		{
			name:        "empty webhook - disabled",
			webhook:     "",
			channel:     "#helix",
			username:    "bot",
			wantEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := NewSlackChannel(tt.webhook, tt.channel, tt.username)
			assert.Equal(t, tt.wantEnabled, channel.IsEnabled())
			assert.Equal(t, "slack", channel.GetName())
		})
	}
}

func TestSlackChannel_Send(t *testing.T) {
	tests := []struct {
		name         string
		notification *Notification
		serverStatus int
		serverCheck  func(*testing.T, *http.Request)
		wantErr      bool
		errContains  string
	}{
		{
			name: "success - info notification",
			notification: &Notification{
				Title:   "Test Title",
				Message: "Test Message",
				Type:    NotificationTypeInfo,
			},
			serverStatus: http.StatusOK,
			serverCheck: func(t *testing.T, r *http.Request) {
				// Verify payload structure
				var payload map[string]interface{}
				body, _ := io.ReadAll(r.Body)
				err := json.Unmarshal(body, &payload)
				require.NoError(t, err)

				assert.Equal(t, "#helix", payload["channel"])
				assert.Contains(t, payload["text"], "Test Title")
				assert.Contains(t, payload["text"], "Test Message")
				assert.Equal(t, "bot", payload["username"])
				assert.Equal(t, ":information_source:", payload["icon_emoji"])
			},
			wantErr: false,
		},
		{
			name: "success - error notification",
			notification: &Notification{
				Title:   "Error Occurred",
				Message: "Something went wrong",
				Type:    NotificationTypeError,
			},
			serverStatus: http.StatusOK,
			serverCheck: func(t *testing.T, r *http.Request) {
				var payload map[string]interface{}
				body, _ := io.ReadAll(r.Body)
				json.Unmarshal(body, &payload)
				assert.Equal(t, ":x:", payload["icon_emoji"])
			},
			wantErr: false,
		},
		{
			name: "error - server error 500",
			notification: &Notification{
				Title:   "Test",
				Message: "Test",
				Type:    NotificationTypeInfo,
			},
			serverStatus: http.StatusInternalServerError,
			wantErr:      true,
			errContains:  "status 500",
		},
		{
			name: "error - unauthorized 401",
			notification: &Notification{
				Title:   "Test",
				Message: "Test",
				Type:    NotificationTypeInfo,
			},
			serverStatus: http.StatusUnauthorized,
			wantErr:      true,
			errContains:  "status 401",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.serverCheck != nil {
					tt.serverCheck(t, r)
				}
				w.WriteHeader(tt.serverStatus)
			}))
			defer server.Close()

			// Create channel with mock server URL
			channel := NewSlackChannel(server.URL, "#helix", "bot")

			// Send notification
			err := channel.Send(context.Background(), tt.notification)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSlackChannel_Send_Disabled(t *testing.T) {
	channel := NewSlackChannel("", "#test", "bot")

	notification := &Notification{
		Title:   "Test",
		Message: "Test",
		Type:    NotificationTypeInfo,
	}

	err := channel.Send(context.Background(), notification)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestSlackChannel_GetIconForType(t *testing.T) {
	channel := &SlackChannel{}

	tests := []struct {
		notifType NotificationType
		wantIcon  string
	}{
		{NotificationTypeSuccess, ":white_check_mark:"},
		{NotificationTypeWarning, ":warning:"},
		{NotificationTypeError, ":x:"},
		{NotificationTypeAlert, ":rotating_light:"},
		{NotificationTypeInfo, ":information_source:"},
	}

	for _, tt := range tests {
		t.Run(string(tt.notifType), func(t *testing.T) {
			icon := channel.getIconForType(tt.notifType)
			assert.Equal(t, tt.wantIcon, icon)
		})
	}
}

func TestSlackChannel_GetConfig(t *testing.T) {
	channel := NewSlackChannel("https://hooks.slack.com/test", "#channel", "username")
	config := channel.GetConfig()

	assert.Equal(t, "https://hooks.slack.com/test", config["webhook"])
	assert.Equal(t, "#channel", config["channel"])
	assert.Equal(t, "username", config["username"])
}
