package memory

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessage(t *testing.T) {
	t.Run("create_message", func(t *testing.T) {
		msg := NewMessage(RoleUser, "Hello")
		assert.NotEmpty(t, msg.ID)
		assert.Equal(t, RoleUser, msg.Role)
		assert.Equal(t, "Hello", msg.Content)
		assert.NotZero(t, msg.Timestamp)
		assert.Greater(t, msg.TokenCount, 0)
		assert.Greater(t, msg.Size, 0)
	})

	t.Run("create_user_message", func(t *testing.T) {
		msg := NewUserMessage("Hello")
		assert.Equal(t, RoleUser, msg.Role)
	})

	t.Run("create_assistant_message", func(t *testing.T) {
		msg := NewAssistantMessage("Hi there")
		assert.Equal(t, RoleAssistant, msg.Role)
	})

	t.Run("create_system_message", func(t *testing.T) {
		msg := NewSystemMessage("System ready")
		assert.Equal(t, RoleSystem, msg.Role)
	})

	t.Run("message_metadata", func(t *testing.T) {
		msg := NewUserMessage("Test")
		msg.SetMetadata("key", "value")

		value, ok := msg.GetMetadata("key")
		assert.True(t, ok)
		assert.Equal(t, "value", value)
	})

	t.Run("clone_message", func(t *testing.T) {
		msg := NewUserMessage("Test")
		msg.SetMetadata("key", "value")

		clone := msg.Clone()
		assert.Equal(t, msg.ID, clone.ID)
		assert.Equal(t, msg.Content, clone.Content)

		value, _ := clone.GetMetadata("key")
		assert.Equal(t, "value", value)

		// Modify clone shouldn't affect original
		clone.SetMetadata("new", "data")
		_, ok := msg.GetMetadata("new")
		assert.False(t, ok)
	})

	t.Run("validate_message", func(t *testing.T) {
		msg := NewUserMessage("Test")
		err := msg.Validate()
		assert.NoError(t, err)

		// Invalid role
		msg.Role = Role("invalid")
		err = msg.Validate()
		assert.Error(t, err)
	})
}

func TestRole(t *testing.T) {
	t.Run("role_is_valid", func(t *testing.T) {
		assert.True(t, RoleUser.IsValid())
		assert.True(t, RoleAssistant.IsValid())
		assert.True(t, RoleSystem.IsValid())
		assert.True(t, RoleTool.IsValid())
		assert.False(t, Role("invalid").IsValid())
	})

	t.Run("role_string", func(t *testing.T) {
		assert.Equal(t, "user", RoleUser.String())
		assert.Equal(t, "assistant", RoleAssistant.String())
	})
}

func TestConversation(t *testing.T) {
	t.Run("create_conversation", func(t *testing.T) {
		conv := NewConversation("Test Conversation")
		assert.NotEmpty(t, conv.ID)
		assert.Equal(t, "Test Conversation", conv.Title)
		assert.NotNil(t, conv.Messages)
		assert.Equal(t, 0, conv.MessageCount)
		assert.Equal(t, 0, conv.TokenCount)
	})

	t.Run("add_message", func(t *testing.T) {
		conv := NewConversation("Test")
		msg := NewUserMessage("Hello")

		conv.AddMessage(msg)
		assert.Equal(t, 1, conv.MessageCount)
		assert.Greater(t, conv.TokenCount, 0)
	})

	t.Run("get_messages", func(t *testing.T) {
		conv := NewConversation("Test")
		conv.AddMessage(NewUserMessage("Message 1"))
		conv.AddMessage(NewAssistantMessage("Message 2"))

		messages := conv.GetMessages()
		assert.Len(t, messages, 2)
	})

	t.Run("get_by_role", func(t *testing.T) {
		conv := NewConversation("Test")
		conv.AddMessage(NewUserMessage("User 1"))
		conv.AddMessage(NewAssistantMessage("Assistant 1"))
		conv.AddMessage(NewUserMessage("User 2"))

		userMsgs := conv.GetMessagesByRole(RoleUser)
		assert.Len(t, userMsgs, 2)

		assistantMsgs := conv.GetMessagesByRole(RoleAssistant)
		assert.Len(t, assistantMsgs, 1)
	})

	t.Run("get_recent", func(t *testing.T) {
		conv := NewConversation("Test")
		conv.AddMessage(NewUserMessage("Message 1"))
		conv.AddMessage(NewUserMessage("Message 2"))
		conv.AddMessage(NewUserMessage("Message 3"))

		recent := conv.GetRecent(2)
		assert.Len(t, recent, 2)
		assert.Contains(t, recent[0].Content, "Message 2")
		assert.Contains(t, recent[1].Content, "Message 3")
	})

	t.Run("get_range", func(t *testing.T) {
		conv := NewConversation("Test")
		conv.AddMessage(NewUserMessage("Message 1"))
		conv.AddMessage(NewUserMessage("Message 2"))
		conv.AddMessage(NewUserMessage("Message 3"))

		messages := conv.GetRange(0, 2)
		assert.Len(t, messages, 2)
		assert.Contains(t, messages[0].Content, "Message 1")
	})

	t.Run("search", func(t *testing.T) {
		conv := NewConversation("Test")
		conv.AddMessage(NewUserMessage("Hello world"))
		conv.AddMessage(NewUserMessage("Goodbye"))
		conv.AddMessage(NewUserMessage("Hello again"))

		results := conv.Search("hello")
		assert.Len(t, results, 2)
	})

	t.Run("clear", func(t *testing.T) {
		conv := NewConversation("Test")
		conv.AddMessage(NewUserMessage("Message 1"))
		conv.AddMessage(NewUserMessage("Message 2"))

		conv.Clear()
		assert.Equal(t, 0, conv.MessageCount)
		assert.Equal(t, 0, conv.TokenCount)
	})

	t.Run("truncate", func(t *testing.T) {
		conv := NewConversation("Test")
		for i := 0; i < 10; i++ {
			conv.AddMessage(NewUserMessage("Message"))
		}

		removed := conv.Truncate(5)
		assert.Equal(t, 5, removed)
		assert.Equal(t, 5, conv.MessageCount)
	})

	t.Run("metadata", func(t *testing.T) {
		conv := NewConversation("Test")
		conv.SetMetadata("key", "value")

		value, ok := conv.GetMetadata("key")
		assert.True(t, ok)
		assert.Equal(t, "value", value)
	})

	t.Run("clone", func(t *testing.T) {
		conv := NewConversation("Test")
		conv.AddMessage(NewUserMessage("Hello"))
		conv.SetMetadata("key", "value")

		clone := conv.Clone()
		assert.Equal(t, conv.ID, clone.ID)
		assert.Equal(t, conv.Title, clone.Title)
		assert.Len(t, clone.Messages, 1)

		value, _ := clone.GetMetadata("key")
		assert.Equal(t, "value", value)
	})

	t.Run("validate", func(t *testing.T) {
		conv := NewConversation("Test")
		err := conv.Validate()
		assert.NoError(t, err)

		conv.Title = ""
		err = conv.Validate()
		assert.Error(t, err)
	})

	t.Run("to_text", func(t *testing.T) {
		conv := NewConversation("Test")
		conv.AddMessage(NewUserMessage("Hello"))
		conv.AddMessage(NewAssistantMessage("Hi"))

		text := conv.ToText()
		assert.Contains(t, text, "Test")
		assert.Contains(t, text, "Hello")
		assert.Contains(t, text, "Hi")
	})

	t.Run("statistics", func(t *testing.T) {
		conv := NewConversation("Test")
		conv.AddMessage(NewUserMessage("Hello"))
		conv.AddMessage(NewAssistantMessage("Hi"))
		conv.AddMessage(NewUserMessage("How are you?"))

		stats := conv.GetStatistics()
		assert.Equal(t, 3, stats.TotalMessages)
		assert.Equal(t, 2, stats.ByRole[RoleUser])
		assert.Equal(t, 1, stats.ByRole[RoleAssistant])
		assert.Greater(t, stats.TotalTokens, 0)
		assert.Greater(t, stats.AverageTokens, float64(0))
	})
}

func TestManager(t *testing.T) {
	t.Run("create_manager", func(t *testing.T) {
		mgr := NewManager()
		assert.NotNil(t, mgr)
		assert.Equal(t, 0, mgr.Count())
	})

	t.Run("create_conversation", func(t *testing.T) {
		mgr := NewManager()
		conv, err := mgr.CreateConversation("Test")
		require.NoError(t, err)
		assert.NotEmpty(t, conv.ID)
		assert.Equal(t, "Test", conv.Title)
		assert.Equal(t, 1, mgr.Count())
	})

	t.Run("create_conversation_empty_title", func(t *testing.T) {
		mgr := NewManager()
		_, err := mgr.CreateConversation("")
		assert.Error(t, err)
	})

	t.Run("get_conversation", func(t *testing.T) {
		mgr := NewManager()
		conv, _ := mgr.CreateConversation("Test")

		retrieved, err := mgr.GetConversation(conv.ID)
		require.NoError(t, err)
		assert.Equal(t, conv.ID, retrieved.ID)
	})

	t.Run("get_nonexistent_conversation", func(t *testing.T) {
		mgr := NewManager()
		_, err := mgr.GetConversation("nonexistent")
		assert.Error(t, err)
	})

	t.Run("set_active", func(t *testing.T) {
		mgr := NewManager()
		conv, _ := mgr.CreateConversation("Test")

		err := mgr.SetActive(conv.ID)
		require.NoError(t, err)

		active := mgr.GetActive()
		assert.NotNil(t, active)
		assert.Equal(t, conv.ID, active.ID)
	})

	t.Run("add_message", func(t *testing.T) {
		mgr := NewManager()
		conv, _ := mgr.CreateConversation("Test")

		msg := NewUserMessage("Hello")
		err := mgr.AddMessage(conv.ID, msg)
		require.NoError(t, err)

		retrieved, _ := mgr.GetConversation(conv.ID)
		assert.Equal(t, 1, retrieved.MessageCount)
	})

	t.Run("add_message_to_active", func(t *testing.T) {
		mgr := NewManager()
		conv, _ := mgr.CreateConversation("Test")
		mgr.SetActive(conv.ID)

		msg := NewUserMessage("Hello")
		err := mgr.AddMessageToActive(msg)
		require.NoError(t, err)

		assert.Equal(t, 1, conv.MessageCount)
	})

	t.Run("delete_conversation", func(t *testing.T) {
		mgr := NewManager()
		conv, _ := mgr.CreateConversation("Test")

		err := mgr.DeleteConversation(conv.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, mgr.Count())
	})

	t.Run("clear_conversation", func(t *testing.T) {
		mgr := NewManager()
		conv, _ := mgr.CreateConversation("Test")
		mgr.AddMessage(conv.ID, NewUserMessage("Hello"))

		err := mgr.ClearConversation(conv.ID)
		require.NoError(t, err)

		retrieved, _ := mgr.GetConversation(conv.ID)
		assert.Equal(t, 0, retrieved.MessageCount)
	})
}

func TestManagerQueries(t *testing.T) {
	t.Run("get_all", func(t *testing.T) {
		mgr := NewManager()
		mgr.CreateConversation("Conv 1")
		mgr.CreateConversation("Conv 2")
		mgr.CreateConversation("Conv 3")

		all := mgr.GetAll()
		assert.Len(t, all, 3)
	})

	t.Run("get_by_session", func(t *testing.T) {
		mgr := NewManager()
		conv1, _ := mgr.CreateConversation("Conv 1")
		conv2, _ := mgr.CreateConversation("Conv 2")
		conv3, _ := mgr.CreateConversation("Conv 3")

		conv1.SessionID = "session-1"
		conv2.SessionID = "session-1"
		conv3.SessionID = "session-2"

		sessionConvs := mgr.GetBySession("session-1")
		assert.Len(t, sessionConvs, 2)
	})

	t.Run("get_recent", func(t *testing.T) {
		mgr := NewManager()
		_, _ = mgr.CreateConversation("Conv 1")
		time.Sleep(10 * time.Millisecond)
		conv2, _ := mgr.CreateConversation("Conv 2")
		time.Sleep(10 * time.Millisecond)
		conv3, _ := mgr.CreateConversation("Conv 3")

		recent := mgr.GetRecent(2)
		assert.Len(t, recent, 2)
		assert.Equal(t, conv3.ID, recent[0].ID) // Most recent first
		assert.Equal(t, conv2.ID, recent[1].ID)
	})

	t.Run("search", func(t *testing.T) {
		mgr := NewManager()
		conv1, _ := mgr.CreateConversation("Bug Fix")
		conv2, _ := mgr.CreateConversation("Feature")

		mgr.AddMessage(conv1.ID, NewUserMessage("Fix the login bug"))
		mgr.AddMessage(conv2.ID, NewUserMessage("Add new feature"))

		results := mgr.Search("bug")
		assert.Len(t, results, 1)
		assert.Equal(t, conv1.ID, results[0].ID)
	})

	t.Run("search_messages", func(t *testing.T) {
		mgr := NewManager()
		conv1, _ := mgr.CreateConversation("Conv 1")
		conv2, _ := mgr.CreateConversation("Conv 2")

		mgr.AddMessage(conv1.ID, NewUserMessage("Hello world"))
		mgr.AddMessage(conv2.ID, NewUserMessage("Goodbye world"))

		results := mgr.SearchMessages("world")
		assert.Len(t, results, 2)
	})

	t.Run("total_messages", func(t *testing.T) {
		mgr := NewManager()
		conv1, _ := mgr.CreateConversation("Conv 1")
		conv2, _ := mgr.CreateConversation("Conv 2")

		mgr.AddMessage(conv1.ID, NewUserMessage("Message 1"))
		mgr.AddMessage(conv1.ID, NewUserMessage("Message 2"))
		mgr.AddMessage(conv2.ID, NewUserMessage("Message 3"))

		assert.Equal(t, 3, mgr.TotalMessages())
	})

	t.Run("total_tokens", func(t *testing.T) {
		mgr := NewManager()
		conv, _ := mgr.CreateConversation("Test")
		mgr.AddMessage(conv.ID, NewUserMessage("Hello world"))

		assert.Greater(t, mgr.TotalTokens(), 0)
	})
}

func TestManagerLimits(t *testing.T) {
	t.Run("max_messages_limit", func(t *testing.T) {
		mgr := NewManager()
		mgr.SetMaxMessages(10)
		conv, _ := mgr.CreateConversation("Test")

		// Add 20 messages
		for i := 0; i < 20; i++ {
			mgr.AddMessage(conv.ID, NewUserMessage("Message"))
		}

		retrieved, _ := mgr.GetConversation(conv.ID)
		assert.LessOrEqual(t, retrieved.MessageCount, 10)
	})

	t.Run("max_tokens_limit", func(t *testing.T) {
		mgr := NewManager()
		mgr.SetMaxTokens(100)
		conv, _ := mgr.CreateConversation("Test")

		// Add many long messages
		longText := strings.Repeat("word ", 100) // ~400 chars = ~100 tokens
		for i := 0; i < 10; i++ {
			mgr.AddMessage(conv.ID, NewUserMessage(longText))
		}

		retrieved, _ := mgr.GetConversation(conv.ID)
		assert.LessOrEqual(t, retrieved.TokenCount, 150) // Some buffer
	})

	t.Run("trim_conversations", func(t *testing.T) {
		mgr := NewManager()
		mgr.SetMaxConversations(2)

		for i := 0; i < 5; i++ {
			mgr.CreateConversation("Conv")
			time.Sleep(5 * time.Millisecond)
		}

		removed := mgr.TrimConversations()
		assert.Equal(t, 3, removed)
		assert.Equal(t, 2, mgr.Count())
	})
}

func TestManagerCallbacks(t *testing.T) {
	t.Run("on_create", func(t *testing.T) {
		mgr := NewManager()
		called := false
		var createdConv *Conversation

		mgr.OnCreate(func(conv *Conversation) {
			called = true
			createdConv = conv
		})

		conv, _ := mgr.CreateConversation("Test")
		assert.True(t, called)
		assert.Equal(t, conv.ID, createdConv.ID)
	})

	t.Run("on_message", func(t *testing.T) {
		mgr := NewManager()
		called := false
		var receivedMsg *Message

		mgr.OnMessage(func(conv *Conversation, msg *Message) {
			called = true
			receivedMsg = msg
		})

		conv, _ := mgr.CreateConversation("Test")
		msg := NewUserMessage("Hello")
		mgr.AddMessage(conv.ID, msg)

		assert.True(t, called)
		assert.Equal(t, msg.ID, receivedMsg.ID)
	})

	t.Run("on_clear", func(t *testing.T) {
		mgr := NewManager()
		called := false

		mgr.OnClear(func(conv *Conversation) {
			called = true
		})

		conv, _ := mgr.CreateConversation("Test")
		mgr.AddMessage(conv.ID, NewUserMessage("Hello"))
		mgr.ClearConversation(conv.ID)

		assert.True(t, called)
	})

	t.Run("on_delete", func(t *testing.T) {
		mgr := NewManager()
		called := false

		mgr.OnDelete(func(conv *Conversation) {
			called = true
		})

		conv, _ := mgr.CreateConversation("Test")
		mgr.DeleteConversation(conv.ID)

		assert.True(t, called)
	})
}

func TestManagerStatistics(t *testing.T) {
	t.Run("get_statistics", func(t *testing.T) {
		mgr := NewManager()
		conv1, _ := mgr.CreateConversation("Conv 1")
		conv2, _ := mgr.CreateConversation("Conv 2")

		mgr.AddMessage(conv1.ID, NewUserMessage("User 1"))
		mgr.AddMessage(conv1.ID, NewAssistantMessage("Assistant 1"))
		mgr.AddMessage(conv2.ID, NewUserMessage("User 2"))

		stats := mgr.GetStatistics()
		assert.Equal(t, 2, stats.TotalConversations)
		assert.Equal(t, 3, stats.TotalMessages)
		assert.Equal(t, 2, stats.ByRole[RoleUser])
		assert.Equal(t, 1, stats.ByRole[RoleAssistant])
		assert.Greater(t, stats.AverageMessagesPerConv, float64(0))
		assert.Greater(t, stats.AverageTokensPerMessage, float64(0))
	})
}

func TestManagerExportImport(t *testing.T) {
	t.Run("export_conversation", func(t *testing.T) {
		mgr := NewManager()
		conv, _ := mgr.CreateConversation("Test")
		mgr.AddMessage(conv.ID, NewUserMessage("Hello"))

		snapshot, err := mgr.Export(conv.ID)
		require.NoError(t, err)
		assert.Equal(t, conv.ID, snapshot.Conversation.ID)
		assert.Len(t, snapshot.Conversation.Messages, 1)
	})

	t.Run("import_conversation", func(t *testing.T) {
		mgr1 := NewManager()
		conv, _ := mgr1.CreateConversation("Test")
		mgr1.AddMessage(conv.ID, NewUserMessage("Hello"))

		snapshot, _ := mgr1.Export(conv.ID)

		mgr2 := NewManager()
		err := mgr2.Import(snapshot)
		require.NoError(t, err)

		imported, err := mgr2.GetConversation(conv.ID)
		require.NoError(t, err)
		assert.Equal(t, conv.Title, imported.Title)
		assert.Len(t, imported.Messages, 1)
	})

	t.Run("import_duplicate_error", func(t *testing.T) {
		mgr := NewManager()
		conv, _ := mgr.CreateConversation("Test")

		snapshot, _ := mgr.Export(conv.ID)

		err := mgr.Import(snapshot)
		assert.Error(t, err) // Already exists
	})
}

func TestConcurrency(t *testing.T) {
	t.Run("concurrent_create", func(t *testing.T) {
		mgr := NewManager()
		var wg sync.WaitGroup

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				mgr.CreateConversation("Conv")
			}(i)
		}

		wg.Wait()
		assert.Equal(t, 10, mgr.Count())
	})

	t.Run("concurrent_add_message", func(t *testing.T) {
		mgr := NewManager()
		conv, _ := mgr.CreateConversation("Test")

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				mgr.AddMessage(conv.ID, NewUserMessage("Message"))
			}()
		}

		wg.Wait()
		retrieved, _ := mgr.GetConversation(conv.ID)
		assert.Equal(t, 10, retrieved.MessageCount)
	})

	t.Run("concurrent_read_write", func(t *testing.T) {
		mgr := NewManager()
		conv, _ := mgr.CreateConversation("Test")

		var wg sync.WaitGroup

		// Writers
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				mgr.AddMessage(conv.ID, NewUserMessage("Message"))
			}()
		}

		// Readers
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				mgr.GetAll()
				mgr.TotalMessages()
			}()
		}

		wg.Wait()
		// Should not panic or error
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("empty_manager", func(t *testing.T) {
		mgr := NewManager()
		assert.Equal(t, 0, mgr.Count())
		assert.Equal(t, 0, mgr.TotalMessages())
		assert.Equal(t, 0, mgr.TotalTokens())
	})

	t.Run("empty_conversation", func(t *testing.T) {
		conv := NewConversation("Empty")
		stats := conv.GetStatistics()
		assert.Equal(t, 0, stats.TotalMessages)
	})

	t.Run("truncate_to_zero", func(t *testing.T) {
		conv := NewConversation("Test")
		conv.AddMessage(NewUserMessage("Message"))

		removed := conv.Truncate(0)
		assert.Equal(t, 0, removed)
		assert.Equal(t, 1, conv.MessageCount)
	})

	t.Run("clear_manager", func(t *testing.T) {
		mgr := NewManager()
		mgr.CreateConversation("Conv 1")
		mgr.CreateConversation("Conv 2")

		mgr.Clear()
		assert.Equal(t, 0, mgr.Count())
	})
}
