package context

import (
	"testing"

	"dev.helix.code/internal/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder()
	assert.NotNil(t, builder)
	assert.Empty(t, builder.messages)
	assert.Empty(t, builder.metadata)
	assert.Equal(t, 0, builder.MessageCount())
}

func TestSetSystemRole(t *testing.T) {
	builder := NewBuilder()
	builder.SetSystemRole("You are a helpful assistant")

	conv := builder.Build()
	messages := conv.GetMessages()
	require.Greater(t, len(messages), 0)
	assert.Equal(t, memory.RoleSystem, messages[0].Role)
	assert.Equal(t, "You are a helpful assistant", messages[0].Content)
}

func TestAddMessages(t *testing.T) {
	builder := NewBuilder()

	builder.AddUserMessage("Hello")
	builder.AddAssistantMessage("Hi there!")
	builder.AddUserMessage("How are you?")

	assert.Equal(t, 3, builder.MessageCount())

	conv := builder.Build()
	messages := conv.GetMessages()
	assert.Equal(t, 3, len(messages))
	assert.Equal(t, memory.RoleUser, messages[0].Role)
	assert.Equal(t, memory.RoleAssistant, messages[1].Role)
	assert.Equal(t, memory.RoleUser, messages[2].Role)
}

func TestMetadata(t *testing.T) {
	builder := NewBuilder()
	builder.SetMetadata("title", "Test Conversation")
	builder.SetMetadata("project", "test-project")

	assert.Equal(t, "Test Conversation", builder.GetMetadata("title"))
	assert.Equal(t, "test-project", builder.GetMetadata("project"))

	conv := builder.Build()
	assert.Equal(t, "Test Conversation", conv.Title)
	projectMeta := conv.Metadata["project"]
	assert.Equal(t, "test-project", projectMeta)
}

func TestBuild(t *testing.T) {
	builder := NewBuilder()
	builder.SetMetadata("title", "Full Conversation")
	builder.SetSystemRole("You are an expert")
	builder.AddUserMessage("Question")
	builder.AddAssistantMessage("Answer")

	conv := builder.Build()
	require.NotNil(t, conv)
	assert.Equal(t, "Full Conversation", conv.Title)

	messages := conv.GetMessages()
	assert.Equal(t, 3, len(messages)) // system + user + assistant
	assert.Equal(t, memory.RoleSystem, messages[0].Role)
	assert.Equal(t, memory.RoleUser, messages[1].Role)
	assert.Equal(t, memory.RoleAssistant, messages[2].Role)
}

func TestToText(t *testing.T) {
	builder := NewBuilder()
	builder.SetSystemRole("System message")
	builder.AddUserMessage("User message")
	builder.AddAssistantMessage("Assistant message")

	text := builder.ToText()
	assert.Contains(t, text, "[system] System message")
	assert.Contains(t, text, "[user] User message")
	assert.Contains(t, text, "[assistant] Assistant message")
}

func TestClear(t *testing.T) {
	builder := NewBuilder()
	builder.SetSystemRole("Role")
	builder.AddUserMessage("Message")
	builder.SetMetadata("key", "value")

	builder.Clear()

	assert.Equal(t, 0, builder.MessageCount())
	assert.Empty(t, builder.GetMetadata("key"))
	assert.Empty(t, builder.systemRole)
}

func TestClone(t *testing.T) {
	builder := NewBuilder()
	builder.SetSystemRole("Original role")
	builder.AddUserMessage("Original message")
	builder.SetMetadata("key", "value")

	clone := builder.Clone()

	// Modify original
	builder.AddUserMessage("New message")
	builder.SetMetadata("key", "new value")

	// Clone should be unchanged
	assert.Equal(t, 1, clone.MessageCount())
	assert.Equal(t, "value", clone.GetMetadata("key"))
}

func TestFromConversation(t *testing.T) {
	// Create conversation
	conv := memory.NewConversation("Test")
	conv.AddMessage(memory.NewSystemMessage("System"))
	conv.AddMessage(memory.NewUserMessage("User"))
	conv.AddMessage(memory.NewAssistantMessage("Assistant"))
	conv.SetMetadata("project", "test")

	// Build from conversation
	builder := FromConversation(conv)

	assert.Equal(t, 2, builder.MessageCount()) // Excludes system
	assert.Equal(t, "System", builder.systemRole)
	assert.Equal(t, "test", builder.GetMetadata("project"))

	// Rebuild should match original
	rebuilt := builder.Build()
	assert.Equal(t, 3, len(rebuilt.GetMessages()))
}

func TestConcurrency(t *testing.T) {
	builder := NewBuilder()

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			builder.AddUserMessage("Test")
			builder.SetMetadata("key", "value")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	assert.Equal(t, 10, builder.MessageCount())
}
