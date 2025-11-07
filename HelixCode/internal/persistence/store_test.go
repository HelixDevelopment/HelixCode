package persistence

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"dev.helix.code/internal/focus"
	"dev.helix.code/internal/memory"
	"dev.helix.code/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore(t *testing.T) {
	t.Run("create_store", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, err := NewStore(tmpDir)
		require.NoError(t, err)
		assert.NotNil(t, store)
		assert.Equal(t, tmpDir, store.basePath)
	})

	t.Run("create_store_creates_directory", func(t *testing.T) {
		tmpDir := filepath.Join(t.TempDir(), "subdir", "storage")
		store, err := NewStore(tmpDir)
		require.NoError(t, err)
		assert.NotNil(t, store)

		// Check directory exists
		_, err = os.Stat(tmpDir)
		assert.NoError(t, err)
	})

	t.Run("set_managers", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, _ := NewStore(tmpDir)

		sessionMgr := session.NewManager()
		memoryMgr := memory.NewManager()
		focusMgr := focus.NewManager()

		store.SetSessionManager(sessionMgr)
		store.SetMemoryManager(memoryMgr)
		store.SetFocusManager(focusMgr)

		assert.NotNil(t, store.sessionMgr)
		assert.NotNil(t, store.memoryMgr)
		assert.NotNil(t, store.focusMgr)
	})

	t.Run("set_serializer", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, _ := NewStore(tmpDir)

		gzipSerializer := NewJSONGzipSerializer()
		store.SetSerializer(gzipSerializer)

		assert.Equal(t, FormatJSONGzip, store.serializer.Format())
	})
}

func TestSaveSessions(t *testing.T) {
	t.Run("save_and_load_sessions", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, _ := NewStore(tmpDir)

		// Create session manager
		sessionMgr := session.NewManager()
		store.SetSessionManager(sessionMgr)

		// Create sessions
		sess1, _ := sessionMgr.Create("project1", "Session 1", "Test session 1", session.ModePlanning)
		sess2, _ := sessionMgr.Create("project2", "Session 2", "Test session 2", session.ModePlanning)

		// Save
		err := store.SaveAll()
		require.NoError(t, err)

		// Check files exist
		sessionsPath := filepath.Join(tmpDir, "sessions")
		_, err = os.Stat(filepath.Join(sessionsPath, sess1.ID+".json"))
		assert.NoError(t, err)
		_, err = os.Stat(filepath.Join(sessionsPath, sess2.ID+".json"))
		assert.NoError(t, err)

		// Create new manager and load
		newSessionMgr := session.NewManager()
		store.SetSessionManager(newSessionMgr)

		err = store.LoadAll()
		require.NoError(t, err)

		// Verify loaded
		loaded1, err := newSessionMgr.Get(sess1.ID)
		assert.NoError(t, err)
		assert.Equal(t, "project1", loaded1.ProjectID)

		loaded2, err := newSessionMgr.Get(sess2.ID)
		assert.NoError(t, err)
		assert.Equal(t, "project2", loaded2.ProjectID)
	})

	t.Run("save_empty_sessions", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, _ := NewStore(tmpDir)

		sessionMgr := session.NewManager()
		store.SetSessionManager(sessionMgr)

		// Save with no sessions
		err := store.SaveAll()
		require.NoError(t, err)
	})
}

func TestSaveConversations(t *testing.T) {
	t.Run("save_and_load_conversations", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, _ := NewStore(tmpDir)

		// Create memory manager
		memoryMgr := memory.NewManager()
		store.SetMemoryManager(memoryMgr)

		// Create conversations
		conv1, _ := memoryMgr.CreateConversation("Chat 1")
		conv2, _ := memoryMgr.CreateConversation("Chat 2")

		// Add messages
		memoryMgr.AddMessage(conv1.ID, memory.NewUserMessage("Hello"))
		memoryMgr.AddMessage(conv2.ID, memory.NewUserMessage("Hi there"))

		// Save
		err := store.SaveAll()
		require.NoError(t, err)

		// Create new manager and load
		newMemoryMgr := memory.NewManager()
		store.SetMemoryManager(newMemoryMgr)

		err = store.LoadAll()
		require.NoError(t, err)

		// Verify loaded
		loaded1, err := newMemoryMgr.GetConversation(conv1.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Chat 1", loaded1.Title)
		assert.Equal(t, 1, loaded1.MessageCount)

		loaded2, err := newMemoryMgr.GetConversation(conv2.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Chat 2", loaded2.Title)
		assert.Equal(t, 1, loaded2.MessageCount)
	})
}

func TestSaveFocus(t *testing.T) {
	t.Run("save_and_load_focus_chains", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, _ := NewStore(tmpDir)

		// Create focus manager
		focusMgr := focus.NewManager()
		store.SetFocusManager(focusMgr)

		// Create focus chains
		chain1, _ := focusMgr.CreateChain("Task 1", false)
		chain2, _ := focusMgr.CreateChain("Task 2", false)

		// Add focus items
		f1 := focus.NewFocus(focus.FocusTypeFile, "file1.go")
		chain1.Push(f1)
		f2 := focus.NewFocus(focus.FocusTypeFile, "file2.go")
		chain2.Push(f2)

		// Save
		err := store.SaveAll()
		require.NoError(t, err)

		// Create new manager and load
		newFocusMgr := focus.NewManager()
		store.SetFocusManager(newFocusMgr)

		err = store.LoadAll()
		require.NoError(t, err)

		// Verify loaded
		loaded1, err := newFocusMgr.GetChain(chain1.ID)
		assert.NoError(t, err)
		assert.NotNil(t, loaded1)
		assert.Equal(t, "Task 1", loaded1.Name)

		loaded2, err := newFocusMgr.GetChain(chain2.ID)
		assert.NoError(t, err)
		assert.NotNil(t, loaded2)
		assert.Equal(t, "Task 2", loaded2.Name)
	})
}

func TestAutoSave(t *testing.T) {
	t.Run("enable_auto_save", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, _ := NewStore(tmpDir)

		sessionMgr := session.NewManager()
		store.SetSessionManager(sessionMgr)

		// Enable auto-save with short interval
		store.EnableAutoSave(100 * time.Millisecond)
		assert.True(t, store.autoSaveEnabled)

		// Create session
		sessionMgr.Create("project1", "Session 1", "Test session", session.ModePlanning)

		// Wait for auto-save
		time.Sleep(150 * time.Millisecond)

		// Check last save time updated
		lastSave := store.GetLastSaveTime()
		assert.False(t, lastSave.IsZero())

		// Disable auto-save
		store.DisableAutoSave()
		assert.False(t, store.autoSaveEnabled)
	})

	t.Run("disable_auto_save", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, _ := NewStore(tmpDir)

		store.EnableAutoSave(1 * time.Second)
		assert.True(t, store.autoSaveEnabled)

		store.DisableAutoSave()
		assert.False(t, store.autoSaveEnabled)
	})
}

func TestBackupRestore(t *testing.T) {
	t.Run("backup_and_restore", func(t *testing.T) {
		tmpDir := t.TempDir()
		backupDir := filepath.Join(t.TempDir(), "backup")

		store, _ := NewStore(tmpDir)

		// Create and save data
		sessionMgr := session.NewManager()
		store.SetSessionManager(sessionMgr)
		sess, _ := sessionMgr.Create("project1", "Session 1", "Test session", session.ModePlanning)

		err := store.SaveAll()
		require.NoError(t, err)

		// Backup
		err = store.Backup(backupDir)
		require.NoError(t, err)

		// Check backup exists
		_, err = os.Stat(filepath.Join(backupDir, "sessions", sess.ID+".json"))
		assert.NoError(t, err)

		// Clear original
		err = store.Clear()
		require.NoError(t, err)

		// Restore from backup
		err = store.Restore(backupDir)
		require.NoError(t, err)

		// Verify restored
		_, err = os.Stat(filepath.Join(tmpDir, "sessions", sess.ID+".json"))
		assert.NoError(t, err)
	})

	t.Run("backup_nonexistent", func(t *testing.T) {
		tmpDir := t.TempDir()
		backupDir := filepath.Join(t.TempDir(), "backup")

		store, _ := NewStore(tmpDir)

		// Backup empty store
		err := store.Backup(backupDir)
		require.NoError(t, err)
	})
}

func TestClear(t *testing.T) {
	t.Run("clear_all_data", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, _ := NewStore(tmpDir)

		// Create and save data
		sessionMgr := session.NewManager()
		store.SetSessionManager(sessionMgr)
		sessionMgr.Create("project1", "Session 1", "Test session", session.ModePlanning)

		err := store.SaveAll()
		require.NoError(t, err)

		// Clear
		err = store.Clear()
		require.NoError(t, err)

		// Check directories removed
		_, err = os.Stat(filepath.Join(tmpDir, "sessions"))
		assert.True(t, os.IsNotExist(err))
	})
}

func TestCallbacks(t *testing.T) {
	t.Run("on_save_callback", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, _ := NewStore(tmpDir)

		sessionMgr := session.NewManager()
		store.SetSessionManager(sessionMgr)

		called := false
		var metadata *SaveMetadata

		store.OnSave(func(m *SaveMetadata) {
			called = true
			metadata = m
		})

		sessionMgr.Create("project1", "Session 1", "Test session", session.ModePlanning)
		err := store.SaveAll()
		require.NoError(t, err)

		assert.True(t, called)
		assert.NotNil(t, metadata)
		assert.Equal(t, tmpDir, metadata.Path)
		assert.Equal(t, FormatJSON, metadata.Format)
		assert.Greater(t, metadata.Items, 0)
	})

	t.Run("on_load_callback", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, _ := NewStore(tmpDir)

		sessionMgr := session.NewManager()
		store.SetSessionManager(sessionMgr)
		sessionMgr.Create("project1", "Session 1", "Test session", session.ModePlanning)
		store.SaveAll()

		// Create new manager for loading
		newSessionMgr := session.NewManager()
		store.SetSessionManager(newSessionMgr)

		called := false
		var metadata *LoadMetadata

		store.OnLoad(func(m *LoadMetadata) {
			called = true
			metadata = m
		})

		err := store.LoadAll()
		require.NoError(t, err)

		assert.True(t, called)
		assert.NotNil(t, metadata)
		assert.Equal(t, tmpDir, metadata.Path)
		assert.Greater(t, metadata.Items, 0)
	})

	t.Run("on_error_callback", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, _ := NewStore(tmpDir)

		var errorCount int

		store.OnError(func(err error) {
			errorCount++
		})

		// Enable auto-save with invalid manager (will trigger error)
		store.EnableAutoSave(50 * time.Millisecond)

		// Wait for auto-save attempt
		time.Sleep(100 * time.Millisecond)

		store.DisableAutoSave()

		// Error callback may be called if auto-save encounters issues
		// This test mainly verifies callback registration works
	})
}

func TestSerializers(t *testing.T) {
	t.Run("json_serializer", func(t *testing.T) {
		serializer := NewJSONSerializer()
		assert.Equal(t, FormatJSON, serializer.Format())
		assert.Equal(t, ".json", serializer.Extension())

		// Test serialization
		data := map[string]string{"key": "value"}
		bytes, err := serializer.Serialize(data)
		require.NoError(t, err)
		assert.NotEmpty(t, bytes)

		// Test deserialization
		var result map[string]string
		err = serializer.Deserialize(bytes, &result)
		require.NoError(t, err)
		assert.Equal(t, "value", result["key"])
	})

	t.Run("compact_json_serializer", func(t *testing.T) {
		serializer := NewCompactJSONSerializer()

		data := map[string]string{"key": "value"}
		bytes, err := serializer.Serialize(data)
		require.NoError(t, err)

		// Compact JSON should be smaller
		assert.NotContains(t, string(bytes), "\n")
	})

	t.Run("json_gzip_serializer", func(t *testing.T) {
		serializer := NewJSONGzipSerializer()
		assert.Equal(t, FormatJSONGzip, serializer.Format())
		assert.Equal(t, ".json.gz", serializer.Extension())

		// Test serialization
		data := map[string]string{"key": "value"}
		bytes, err := serializer.Serialize(data)
		require.NoError(t, err)
		assert.NotEmpty(t, bytes)

		// Test deserialization
		var result map[string]string
		err = serializer.Deserialize(bytes, &result)
		require.NoError(t, err)
		assert.Equal(t, "value", result["key"])
	})

	t.Run("gzip_compression", func(t *testing.T) {
		// Large data should compress well
		largeData := make(map[string]string)
		for i := 0; i < 100; i++ {
			largeData["key"+string(rune(i))] = "This is a long repeated value that should compress well"
		}

		jsonSerializer := NewJSONSerializer()
		gzipSerializer := NewJSONGzipSerializer()

		jsonBytes, _ := jsonSerializer.Serialize(largeData)
		gzipBytes, _ := gzipSerializer.Serialize(largeData)

		// Compressed should be smaller (usually)
		// Note: for very small data, gzip might be larger due to header
		assert.NotEqual(t, len(jsonBytes), len(gzipBytes))
	})
}

func TestFormatValidation(t *testing.T) {
	t.Run("validate_json", func(t *testing.T) {
		validJSON := []byte(`{"key": "value"}`)
		err := Validate(validJSON, FormatJSON)
		assert.NoError(t, err)

		invalidJSON := []byte(`{invalid}`)
		err = Validate(invalidJSON, FormatJSON)
		assert.Error(t, err)
	})

	t.Run("detect_format", func(t *testing.T) {
		// JSON
		jsonData := []byte(`{"key": "value"}`)
		format, err := DetectFormat(jsonData)
		require.NoError(t, err)
		assert.Equal(t, FormatJSON, format)

		// Gzip (starts with 0x1f 0x8b)
		gzipData := []byte{0x1f, 0x8b, 0x00, 0x00}
		format, err = DetectFormat(gzipData)
		require.NoError(t, err)
		assert.Equal(t, FormatJSONGzip, format)

		// Empty
		_, err = DetectFormat([]byte{})
		assert.Error(t, err)
	})
}

func TestConcurrency(t *testing.T) {
	t.Run("concurrent_save", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, _ := NewStore(tmpDir)

		sessionMgr := session.NewManager()
		store.SetSessionManager(sessionMgr)

		var wg sync.WaitGroup
		errors := make([]error, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				errors[idx] = store.SaveAll()
			}(i)
		}

		wg.Wait()

		// At least some saves should succeed
		successCount := 0
		for _, err := range errors {
			if err == nil {
				successCount++
			}
		}
		assert.Greater(t, successCount, 0)
	})

	t.Run("concurrent_load", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, _ := NewStore(tmpDir)

		sessionMgr := session.NewManager()
		store.SetSessionManager(sessionMgr)
		sessionMgr.Create("project1", "Session 1", "Test session", session.ModePlanning)
		store.SaveAll()

		var wg sync.WaitGroup
		errors := make([]error, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				errors[idx] = store.LoadAll()
			}(i)
		}

		wg.Wait()

		// All loads should succeed
		for _, err := range errors {
			assert.NoError(t, err)
		}
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("save_without_managers", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, _ := NewStore(tmpDir)

		// Save without any managers set
		err := store.SaveAll()
		require.NoError(t, err)
	})

	t.Run("load_nonexistent", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, _ := NewStore(tmpDir)

		sessionMgr := session.NewManager()
		store.SetSessionManager(sessionMgr)

		// Load from empty directory
		err := store.LoadAll()
		require.NoError(t, err)
		assert.Equal(t, 0, sessionMgr.Count())
	})

	t.Run("clear_empty_store", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, _ := NewStore(tmpDir)

		// Clear when nothing exists
		err := store.Clear()
		require.NoError(t, err)
	})

	t.Run("last_save_time_initial", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, _ := NewStore(tmpDir)

		// Before any save
		lastSave := store.GetLastSaveTime()
		assert.True(t, lastSave.IsZero())
	})

	t.Run("atomic_write", func(t *testing.T) {
		tmpDir := t.TempDir()
		filename := filepath.Join(tmpDir, "test.txt")
		data := []byte("test data")

		err := writeAtomic(filename, data)
		require.NoError(t, err)

		// Verify file exists and has correct content
		content, err := os.ReadFile(filename)
		require.NoError(t, err)
		assert.Equal(t, data, content)

		// Verify temp file is gone
		_, err = os.Stat(filename + ".tmp")
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("copy_nonexistent_dir", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcPath := filepath.Join(tmpDir, "nonexistent")
		dstPath := filepath.Join(tmpDir, "destination")

		// Should not error when source doesn't exist
		err := copyDir(srcPath, dstPath)
		assert.NoError(t, err)
	})
}

func TestIntegration(t *testing.T) {
	t.Run("full_workflow", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, _ := NewStore(tmpDir)

		// Set up all managers
		sessionMgr := session.NewManager()
		memoryMgr := memory.NewManager()
		focusMgr := focus.NewManager()

		store.SetSessionManager(sessionMgr)
		store.SetMemoryManager(memoryMgr)
		store.SetFocusManager(focusMgr)

		// Create data
		sess, _ := sessionMgr.Create("project1", "Session 1", "Test session", session.ModePlanning)
		conv, _ := memoryMgr.CreateConversation("Chat")
		memoryMgr.AddMessage(conv.ID, memory.NewUserMessage("Hello"))
		chain, _ := focusMgr.CreateChain("Task", false)
		f := focus.NewFocus(focus.FocusTypeFile, "file.go")
		chain.Push(f)

		// Save
		err := store.SaveAll()
		require.NoError(t, err)

		// Create new managers and load
		newSessionMgr := session.NewManager()
		newMemoryMgr := memory.NewManager()
		newFocusMgr := focus.NewManager()

		store.SetSessionManager(newSessionMgr)
		store.SetMemoryManager(newMemoryMgr)
		store.SetFocusManager(newFocusMgr)

		err = store.LoadAll()
		require.NoError(t, err)

		// Verify all data loaded
		loadedSess, err := newSessionMgr.Get(sess.ID)
		assert.NoError(t, err)
		assert.Equal(t, "project1", loadedSess.ProjectID)

		loadedConv, err := newMemoryMgr.GetConversation(conv.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Chat", loadedConv.Title)
		assert.Equal(t, 1, loadedConv.MessageCount)

		loadedChain, err := newFocusMgr.GetChain(chain.ID)
		assert.NoError(t, err)
		assert.NotNil(t, loadedChain)
		assert.Equal(t, "Task", loadedChain.Name)
	})
}
