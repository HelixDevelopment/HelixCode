package notification

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmailChannel(t *testing.T) {
	tests := []struct {
		name        string
		smtpServer  string
		port        int
		username    string
		password    string
		from        string
		wantEnabled bool
	}{
		{
			name:        "valid configuration",
			smtpServer:  "smtp.gmail.com",
			port:        587,
			username:    "user@example.com",
			password:    "password",
			from:        "from@example.com",
			wantEnabled: true,
		},
		{
			name:        "empty smtp server - disabled",
			smtpServer:  "",
			port:        587,
			username:    "user@example.com",
			password:    "password",
			from:        "from@example.com",
			wantEnabled: false,
		},
		{
			name:        "empty username - disabled",
			smtpServer:  "smtp.gmail.com",
			port:        587,
			username:    "",
			password:    "password",
			from:        "from@example.com",
			wantEnabled: false,
		},
		{
			name:        "empty password - disabled",
			smtpServer:  "smtp.gmail.com",
			port:        587,
			username:    "user@example.com",
			password:    "",
			from:        "from@example.com",
			wantEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := NewEmailChannel(tt.smtpServer, tt.port, tt.username, tt.password, tt.from)
			assert.Equal(t, tt.wantEnabled, channel.IsEnabled())
			assert.Equal(t, "email", channel.GetName())
		})
	}
}

func TestEmailChannel_ExtractRecipients(t *testing.T) {
	channel := NewEmailChannel("smtp.example.com", 587, "user", "pass", "from@example.com")

	tests := []struct {
		name           string
		notification   *Notification
		wantRecipients []string
		wantErr        bool
		errContains    string
	}{
		{
			name: "single recipient as string",
			notification: &Notification{
				Title:   "Test",
				Message: "Test",
				Metadata: map[string]interface{}{
					"recipient": "admin@example.com",
				},
			},
			wantRecipients: []string{"admin@example.com"},
			wantErr:        false,
		},
		{
			name: "multiple recipients as []string",
			notification: &Notification{
				Title:   "Test",
				Message: "Test",
				Metadata: map[string]interface{}{
					"recipients": []string{"admin@example.com", "team@example.com"},
				},
			},
			wantRecipients: []string{"admin@example.com", "team@example.com"},
			wantErr:        false,
		},
		{
			name: "multiple recipients as []interface{}",
			notification: &Notification{
				Title:   "Test",
				Message: "Test",
				Metadata: map[string]interface{}{
					"recipients": []interface{}{"admin@example.com", "team@example.com"},
				},
			},
			wantRecipients: []string{"admin@example.com", "team@example.com"},
			wantErr:        false,
		},
		{
			name: "no metadata - error",
			notification: &Notification{
				Title:   "Test",
				Message: "Test",
			},
			wantErr:     true,
			errContains: "no metadata",
		},
		{
			name: "no recipients field - error",
			notification: &Notification{
				Title:   "Test",
				Message: "Test",
				Metadata: map[string]interface{}{
					"other_field": "value",
				},
			},
			wantErr:     true,
			errContains: "no recipient(s) specified",
		},
		{
			name: "empty recipients array - error",
			notification: &Notification{
				Title:   "Test",
				Message: "Test",
				Metadata: map[string]interface{}{
					"recipients": []string{},
				},
			},
			wantErr:     true,
			errContains: "recipients list is empty",
		},
		{
			name: "empty recipient string - error",
			notification: &Notification{
				Title:   "Test",
				Message: "Test",
				Metadata: map[string]interface{}{
					"recipient": "",
				},
			},
			wantErr:     true,
			errContains: "recipient email address is empty",
		},
		{
			name: "recipients with empty string - error",
			notification: &Notification{
				Title:   "Test",
				Message: "Test",
				Metadata: map[string]interface{}{
					"recipients": []string{"admin@example.com", ""},
				},
			},
			wantErr:     true,
			errContains: "one or more recipient email addresses are empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recipients, err := channel.extractRecipients(tt.notification)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantRecipients, recipients)
			}
		})
	}
}

func TestEmailChannel_Send_Disabled(t *testing.T) {
	channel := NewEmailChannel("", 587, "", "", "")

	notification := &Notification{
		Title:   "Test",
		Message: "Test",
		Metadata: map[string]interface{}{
			"recipient": "admin@example.com",
		},
	}

	err := channel.Send(context.Background(), notification)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestEmailChannel_GetConfig(t *testing.T) {
	channel := NewEmailChannel("smtp.example.com", 587, "user@example.com", "password", "from@example.com")
	config := channel.GetConfig()

	assert.Equal(t, "smtp.example.com", config["smtp_server"])
	assert.Equal(t, 587, config["port"])
	assert.Equal(t, "user@example.com", config["username"])
	assert.Equal(t, "from@example.com", config["from"])
	// Password should not be in config
	assert.NotContains(t, config, "password")
}
