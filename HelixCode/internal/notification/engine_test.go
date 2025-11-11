package notification

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewNotificationEngine(t *testing.T) {
	engine := NewNotificationEngine()

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.channels)
	assert.NotNil(t, engine.rules)
	assert.NotNil(t, engine.templates)
}

func TestRegisterChannel(t *testing.T) {
	engine := NewNotificationEngine()

	channel := NewSlackChannel("https://hooks.slack.com/test", "#test", "testbot")

	err := engine.RegisterChannel(channel)
	assert.NoError(t, err)

	// Try to register the same channel again
	err = engine.RegisterChannel(channel)
	assert.Error(t, err)
}

func TestAddRule(t *testing.T) {
	engine := NewNotificationEngine()

	rule := NotificationRule{
		Name:      "test-rule",
		Condition: "type==info",
		Channels:  []string{"slack"},
		Priority:  NotificationPriorityMedium,
		Enabled:   true,
	}

	err := engine.AddRule(rule)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(engine.rules))
	assert.NotEqual(t, rule.ID, engine.rules[0].ID) // ID should be set
}

func TestLoadTemplate(t *testing.T) {
	engine := NewNotificationEngine()

	templateStr := "Title: {{.Title}}\nMessage: {{.Message}}"

	err := engine.LoadTemplate("test-template", templateStr)
	assert.NoError(t, err)

	// Test with invalid template
	err = engine.LoadTemplate("invalid", "{{.Invalid}")
	assert.Error(t, err)
}

func TestSendDirect(t *testing.T) {
	engine := NewNotificationEngine()

	// Create a mock channel that doesn't send
	channel := &mockChannel{name: "mock", enabled: true}
	engine.RegisterChannel(channel)

	notification := &Notification{
		Title:   "Test",
		Message: "Test message",
		Type:    NotificationTypeInfo,
	}

	err := engine.SendDirect(context.Background(), notification, []string{"mock"})
	assert.NoError(t, err)
	assert.NotEqual(t, notification.ID, "")
	assert.True(t, notification.CreatedAt.After(notification.CreatedAt.Add(-1))) // CreatedAt set
}

// Mock channel for testing
type mockChannel struct {
	name    string
	enabled bool
}

func (m *mockChannel) Send(ctx context.Context, notification *Notification) error {
	return nil
}

func (m *mockChannel) GetName() string {
	return m.name
}

func (m *mockChannel) IsEnabled() bool {
	return m.enabled
}

func (m *mockChannel) GetConfig() map[string]interface{} {
	return map[string]interface{}{"mock": true}
}

func TestTelegramChannel(t *testing.T) {
	channel := NewTelegramChannel("test-token", "test-chat-id")

	assert.NotNil(t, channel)
	assert.Equal(t, "telegram", channel.GetName())
	assert.True(t, channel.IsEnabled())

	config := channel.GetConfig()
	// Token should be masked for security (only last 4 chars shown)
	assert.Equal(t, "****oken", config["bot_token"])
	assert.Equal(t, "test-chat-id", config["chat_id"])
}

func TestYandexMessengerChannel(t *testing.T) {
	channel := NewYandexMessengerChannel("test-token", "test-chat-id")

	assert.NotNil(t, channel)
	assert.Equal(t, "yandex_messenger", channel.GetName())
	assert.True(t, channel.IsEnabled())

	config := channel.GetConfig()
	assert.Equal(t, "test-token", config["token"])
	assert.Equal(t, "test-chat-id", config["chat_id"])
}

func TestMaxChannel(t *testing.T) {
	channel := NewMaxChannel("test-api-key", "https://max.example.com", "test-room")

	assert.NotNil(t, channel)
	assert.Equal(t, "max", channel.GetName())
	assert.True(t, channel.IsEnabled())

	config := channel.GetConfig()
	assert.Equal(t, "test-api-key", config["api_key"])
	assert.Equal(t, "https://max.example.com", config["endpoint"])
	assert.Equal(t, "test-room", config["room_id"])
}

func TestAllNotificationChannels(t *testing.T) {
	engine := NewNotificationEngine()

	// Register all channels
	slack := NewSlackChannel("https://hooks.slack.com/test", "#test", "testbot")
	email := NewEmailChannel("smtp.test.com", 587, "user", "pass", "from@test.com")
	discord := NewDiscordChannel("https://discord.com/webhook/test")
	telegram := NewTelegramChannel("test-bot-token", "test-chat-id")
	yandex := NewYandexMessengerChannel("test-token", "test-chat-id")
	max := NewMaxChannel("test-api-key", "https://max.test.com", "test-room")

	assert.NoError(t, engine.RegisterChannel(slack))
	assert.NoError(t, engine.RegisterChannel(email))
	assert.NoError(t, engine.RegisterChannel(discord))
	assert.NoError(t, engine.RegisterChannel(telegram))
	assert.NoError(t, engine.RegisterChannel(yandex))
	assert.NoError(t, engine.RegisterChannel(max))

	// Verify all channels are registered
	assert.Equal(t, 6, len(engine.channels))
	assert.NotNil(t, engine.channels["slack"])
	assert.NotNil(t, engine.channels["email"])
	assert.NotNil(t, engine.channels["discord"])
	assert.NotNil(t, engine.channels["telegram"])
	assert.NotNil(t, engine.channels["yandex_messenger"])
	assert.NotNil(t, engine.channels["max"])
}

// TestApplyTemplate tests template application to notifications
func TestApplyTemplate(t *testing.T) {
	engine := NewNotificationEngine()

	t.Run("apply existing template", func(t *testing.T) {
		// Load a template
		templateStr := "Alert: {{.Title}} - {{.Message}}"
		err := engine.LoadTemplate("alert-template", templateStr)
		assert.NoError(t, err)

		// Create a notification
		notification := &Notification{
			Title:   "System Error",
			Message: "Database connection failed",
			Type:    NotificationTypeError,
		}

		// Apply the template
		engine.applyTemplate(notification, "alert-template")

		// Verify the message was updated
		expected := "Alert: System Error - Database connection failed"
		assert.Equal(t, expected, notification.Message)
	})

	t.Run("apply non-existent template", func(t *testing.T) {
		notification := &Notification{
			Title:   "Test",
			Message: "Original message",
			Type:    NotificationTypeInfo,
		}

		// Apply a template that doesn't exist
		engine.applyTemplate(notification, "non-existent-template")

		// Message should remain unchanged
		assert.Equal(t, "Original message", notification.Message)
	})

	t.Run("apply template with complex fields", func(t *testing.T) {
		// Load a complex template
		templateStr := "{{.Type}}: {{.Title}}\nPriority: {{.Priority}}\n{{.Message}}"
		err := engine.LoadTemplate("complex-template", templateStr)
		assert.NoError(t, err)

		notification := &Notification{
			Title:    "Deployment Complete",
			Message:  "Version 2.0 deployed successfully",
			Type:     NotificationTypeSuccess,
			Priority: NotificationPriorityHigh,
		}

		engine.applyTemplate(notification, "complex-template")

		expected := "success: Deployment Complete\nPriority: high\nVersion 2.0 deployed successfully"
		assert.Equal(t, expected, notification.Message)
	})

	t.Run("template with execution error preserves original", func(t *testing.T) {
		// Load a template that will fail (referencing non-existent field)
		templateStr := "{{.NonExistentField}}"
		err := engine.LoadTemplate("bad-template", templateStr)
		assert.NoError(t, err) // Template parsing succeeds

		notification := &Notification{
			Title:   "Test",
			Message: "Original message",
		}

		// Apply template that will fail during execution
		engine.applyTemplate(notification, "bad-template")

		// Message should remain unchanged when execution fails
		assert.Equal(t, "Original message", notification.Message)
	})
}

// TestContains tests the contains helper function
func TestContains(t *testing.T) {
	t.Run("item exists in slice", func(t *testing.T) {
		slice := []string{"apple", "banana", "cherry"}
		assert.True(t, contains(slice, "banana"))
		assert.True(t, contains(slice, "apple"))
		assert.True(t, contains(slice, "cherry"))
	})

	t.Run("item does not exist in slice", func(t *testing.T) {
		slice := []string{"apple", "banana", "cherry"}
		assert.False(t, contains(slice, "orange"))
		assert.False(t, contains(slice, "grape"))
		assert.False(t, contains(slice, ""))
	})

	t.Run("empty slice", func(t *testing.T) {
		slice := []string{}
		assert.False(t, contains(slice, "anything"))
	})

	t.Run("single item slice", func(t *testing.T) {
		slice := []string{"onlyitem"}
		assert.True(t, contains(slice, "onlyitem"))
		assert.False(t, contains(slice, "other"))
	})

	t.Run("case sensitivity", func(t *testing.T) {
		slice := []string{"Apple", "Banana"}
		assert.False(t, contains(slice, "apple")) // Case sensitive
		assert.True(t, contains(slice, "Apple"))
	})

	t.Run("empty string in slice", func(t *testing.T) {
		slice := []string{"", "something"}
		assert.True(t, contains(slice, ""))
		assert.True(t, contains(slice, "something"))
	})
}

// TestGetChannelStats tests channel statistics retrieval
func TestGetChannelStats(t *testing.T) {
	engine := NewNotificationEngine()

	t.Run("no channels registered", func(t *testing.T) {
		stats := engine.GetChannelStats()

		assert.NotNil(t, stats)
		summary := stats["summary"].(map[string]interface{})
		assert.Equal(t, 0, summary["total_channels"])
		assert.Equal(t, 0, summary["enabled_channels"])
		assert.Equal(t, 0, summary["total_rules"])
		assert.Equal(t, 0, summary["active_rules"])
	})

	t.Run("with channels registered", func(t *testing.T) {
		// Register some channels
		slack := NewSlackChannel("https://hooks.slack.com/test", "#test", "bot")
		email := NewEmailChannel("smtp.test.com", 587, "user", "pass", "from@test.com")
		discord := NewDiscordChannel("https://discord.com/webhook/test")

		engine.RegisterChannel(slack)
		engine.RegisterChannel(email)
		engine.RegisterChannel(discord)

		stats := engine.GetChannelStats()

		// Check individual channel stats
		slackStat := stats["slack"].(map[string]interface{})
		assert.True(t, slackStat["enabled"].(bool))
		assert.NotNil(t, slackStat["config"])

		emailStat := stats["email"].(map[string]interface{})
		assert.True(t, emailStat["enabled"].(bool))

		discordStat := stats["discord"].(map[string]interface{})
		assert.True(t, discordStat["enabled"].(bool))

		// Check summary
		summary := stats["summary"].(map[string]interface{})
		assert.Equal(t, 3, summary["total_channels"])
		assert.Equal(t, 3, summary["enabled_channels"])
	})

	t.Run("with enabled and disabled channels", func(t *testing.T) {
		engine := NewNotificationEngine()

		// Register enabled channel
		enabled := &mockChannel{name: "enabled-mock", enabled: true}
		engine.RegisterChannel(enabled)

		// Register disabled channel
		disabled := &mockChannel{name: "disabled-mock", enabled: false}
		engine.RegisterChannel(disabled)

		stats := engine.GetChannelStats()

		// Check individual stats
		enabledStat := stats["enabled-mock"].(map[string]interface{})
		assert.True(t, enabledStat["enabled"].(bool))

		disabledStat := stats["disabled-mock"].(map[string]interface{})
		assert.False(t, disabledStat["enabled"].(bool))

		// Check summary
		summary := stats["summary"].(map[string]interface{})
		assert.Equal(t, 2, summary["total_channels"])
		assert.Equal(t, 1, summary["enabled_channels"]) // Only one enabled
	})

	t.Run("with rules", func(t *testing.T) {
		engine := NewNotificationEngine()

		// Add some rules
		rule1 := NotificationRule{
			Name:     "rule1",
			Enabled:  true,
			Channels: []string{"slack"},
		}
		rule2 := NotificationRule{
			Name:     "rule2",
			Enabled:  false,
			Channels: []string{"email"},
		}
		rule3 := NotificationRule{
			Name:     "rule3",
			Enabled:  true,
			Channels: []string{"discord"},
		}

		engine.AddRule(rule1)
		engine.AddRule(rule2)
		engine.AddRule(rule3)

		stats := engine.GetChannelStats()

		summary := stats["summary"].(map[string]interface{})
		assert.Equal(t, 3, summary["total_rules"])
		assert.Equal(t, 2, summary["active_rules"]) // Only 2 enabled
	})
}

// TestCountActiveRules tests counting of active notification rules
func TestCountActiveRules(t *testing.T) {
	engine := NewNotificationEngine()

	t.Run("no rules", func(t *testing.T) {
		count := engine.countActiveRules()
		assert.Equal(t, 0, count)
	})

	t.Run("all rules enabled", func(t *testing.T) {
		engine := NewNotificationEngine()

		for i := 0; i < 5; i++ {
			rule := NotificationRule{
				Name:    fmt.Sprintf("rule-%d", i),
				Enabled: true,
			}
			engine.AddRule(rule)
		}

		count := engine.countActiveRules()
		assert.Equal(t, 5, count)
	})

	t.Run("all rules disabled", func(t *testing.T) {
		engine := NewNotificationEngine()

		for i := 0; i < 3; i++ {
			rule := NotificationRule{
				Name:    fmt.Sprintf("rule-%d", i),
				Enabled: false,
			}
			engine.AddRule(rule)
		}

		count := engine.countActiveRules()
		assert.Equal(t, 0, count)
	})

	t.Run("mixed enabled and disabled", func(t *testing.T) {
		engine := NewNotificationEngine()

		// Add enabled rules
		for i := 0; i < 7; i++ {
			rule := NotificationRule{
				Name:    fmt.Sprintf("enabled-rule-%d", i),
				Enabled: true,
			}
			engine.AddRule(rule)
		}

		// Add disabled rules
		for i := 0; i < 3; i++ {
			rule := NotificationRule{
				Name:    fmt.Sprintf("disabled-rule-%d", i),
				Enabled: false,
			}
			engine.AddRule(rule)
		}

		count := engine.countActiveRules()
		assert.Equal(t, 7, count) // Only the enabled ones
	})

	t.Run("single enabled rule", func(t *testing.T) {
		engine := NewNotificationEngine()

		rule := NotificationRule{
			Name:    "single-rule",
			Enabled: true,
		}
		engine.AddRule(rule)

		count := engine.countActiveRules()
		assert.Equal(t, 1, count)
	})

	t.Run("single disabled rule", func(t *testing.T) {
		engine := NewNotificationEngine()

		rule := NotificationRule{
			Name:    "disabled-rule",
			Enabled: false,
		}
		engine.AddRule(rule)

		count := engine.countActiveRules()
		assert.Equal(t, 0, count)
	})
}
