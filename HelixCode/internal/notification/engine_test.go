package notification

import (
	"context"
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
	assert.Equal(t, "test-token", config["bot_token"])
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
