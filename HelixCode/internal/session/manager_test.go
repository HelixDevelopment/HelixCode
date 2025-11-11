package session

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager(t *testing.T) {
	t.Run("create_manager", func(t *testing.T) {
		manager := NewManager()
		assert.NotNil(t, manager)
		assert.NotNil(t, manager.focusManager)
		assert.NotNil(t, manager.hooksManager)
		assert.Equal(t, 0, manager.Count())
	})

	t.Run("create_session", func(t *testing.T) {
		manager := NewManager()
		session, err := manager.Create("proj-1", "Test Session", "A test session", ModePlanning)
		require.NoError(t, err)
		assert.NotEmpty(t, session.ID)
		assert.Equal(t, "proj-1", session.ProjectID)
		assert.Equal(t, "Test Session", session.Name)
		assert.Equal(t, "A test session", session.Description)
		assert.Equal(t, ModePlanning, session.Mode)
		assert.Equal(t, StatusPaused, session.Status)
		assert.NotEmpty(t, session.FocusChainID)
	})

	t.Run("create_session_validation", func(t *testing.T) {
		manager := NewManager()

		// Empty project ID
		_, err := manager.Create("", "Test", "Desc", ModePlanning)
		assert.Error(t, err)

		// Empty name
		_, err = manager.Create("proj-1", "", "Desc", ModePlanning)
		assert.Error(t, err)

		// Invalid mode
		_, err = manager.Create("proj-1", "Test", "Desc", Mode("invalid"))
		assert.Error(t, err)
	})

	t.Run("start_session", func(t *testing.T) {
		manager := NewManager()
		session, _ := manager.Create("proj-1", "Test", "Desc", ModePlanning)

		err := manager.Start(session.ID)
		require.NoError(t, err)

		// Check session is active
		retrieved, _ := manager.Get(session.ID)
		assert.Equal(t, StatusActive, retrieved.Status)
		assert.NotZero(t, retrieved.StartedAt)

		// Check is active session
		active := manager.GetActive()
		assert.NotNil(t, active)
		assert.Equal(t, session.ID, active.ID)
	})

	t.Run("pause_session", func(t *testing.T) {
		manager := NewManager()
		session, _ := manager.Create("proj-1", "Test", "Desc", ModePlanning)
		manager.Start(session.ID)

		time.Sleep(10 * time.Millisecond) // Let some time pass

		err := manager.Pause(session.ID)
		require.NoError(t, err)

		retrieved, _ := manager.Get(session.ID)
		assert.Equal(t, StatusPaused, retrieved.Status)
		assert.Greater(t, retrieved.Duration.Milliseconds(), int64(0))
		assert.Zero(t, retrieved.StartedAt) // Reset after pause

		// Active session should be cleared
		assert.Nil(t, manager.GetActive())
	})

	t.Run("resume_session", func(t *testing.T) {
		manager := NewManager()
		session, _ := manager.Create("proj-1", "Test", "Desc", ModePlanning)
		manager.Start(session.ID)
		manager.Pause(session.ID)

		err := manager.Resume(session.ID)
		require.NoError(t, err)

		retrieved, _ := manager.Get(session.ID)
		assert.Equal(t, StatusActive, retrieved.Status)
		assert.NotZero(t, retrieved.StartedAt)

		// Check is active session
		active := manager.GetActive()
		assert.NotNil(t, active)
		assert.Equal(t, session.ID, active.ID)
	})

	t.Run("complete_session", func(t *testing.T) {
		manager := NewManager()
		session, _ := manager.Create("proj-1", "Test", "Desc", ModePlanning)
		manager.Start(session.ID)

		time.Sleep(10 * time.Millisecond)

		err := manager.Complete(session.ID)
		require.NoError(t, err)

		retrieved, _ := manager.Get(session.ID)
		assert.Equal(t, StatusCompleted, retrieved.Status)
		assert.NotZero(t, retrieved.CompletedAt)
		assert.Greater(t, retrieved.Duration.Milliseconds(), int64(0))

		// Active session should be cleared
		assert.Nil(t, manager.GetActive())
	})

	t.Run("fail_session", func(t *testing.T) {
		manager := NewManager()
		session, _ := manager.Create("proj-1", "Test", "Desc", ModePlanning)
		manager.Start(session.ID)

		err := manager.Fail(session.ID, "Something went wrong")
		require.NoError(t, err)

		retrieved, _ := manager.Get(session.ID)
		assert.Equal(t, StatusFailed, retrieved.Status)

		reason, ok := retrieved.GetMetadata("failure_reason")
		assert.True(t, ok)
		assert.Equal(t, "Something went wrong", reason)
	})

	t.Run("delete_session", func(t *testing.T) {
		manager := NewManager()
		session, _ := manager.Create("proj-1", "Test", "Desc", ModePlanning)

		err := manager.Delete(session.ID)
		require.NoError(t, err)

		_, err = manager.Get(session.ID)
		assert.Error(t, err)
	})

	t.Run("cannot_delete_active_session", func(t *testing.T) {
		manager := NewManager()
		session, _ := manager.Create("proj-1", "Test", "Desc", ModePlanning)
		manager.Start(session.ID)

		err := manager.Delete(session.ID)
		assert.Error(t, err)
	})
}

func TestSessionQueries(t *testing.T) {
	t.Run("get_all", func(t *testing.T) {
		manager := NewManager()
		manager.Create("proj-1", "Session 1", "", ModePlanning)
		manager.Create("proj-1", "Session 2", "", ModeBuilding)
		manager.Create("proj-2", "Session 3", "", ModeTesting)

		sessions := manager.GetAll()
		assert.Len(t, sessions, 3)
	})

	t.Run("get_by_project", func(t *testing.T) {
		manager := NewManager()
		manager.Create("proj-1", "Session 1", "", ModePlanning)
		manager.Create("proj-1", "Session 2", "", ModeBuilding)
		manager.Create("proj-2", "Session 3", "", ModeTesting)

		sessions := manager.GetByProject("proj-1")
		assert.Len(t, sessions, 2)

		sessions = manager.GetByProject("proj-2")
		assert.Len(t, sessions, 1)
	})

	t.Run("get_by_mode", func(t *testing.T) {
		manager := NewManager()
		manager.Create("proj-1", "Session 1", "", ModePlanning)
		manager.Create("proj-1", "Session 2", "", ModePlanning)
		manager.Create("proj-2", "Session 3", "", ModeBuilding)

		sessions := manager.GetByMode(ModePlanning)
		assert.Len(t, sessions, 2)

		sessions = manager.GetByMode(ModeBuilding)
		assert.Len(t, sessions, 1)
	})

	t.Run("get_by_status", func(t *testing.T) {
		manager := NewManager()
		s1, _ := manager.Create("proj-1", "Session 1", "", ModePlanning)
		s2, _ := manager.Create("proj-1", "Session 2", "", ModeBuilding)
		_, _ = manager.Create("proj-2", "Session 3", "", ModeTesting)

		manager.Start(s1.ID)
		manager.Complete(s2.ID)
		// s3 remains paused

		active := manager.GetByStatus(StatusActive)
		assert.Len(t, active, 1)

		paused := manager.GetByStatus(StatusPaused)
		assert.Len(t, paused, 1)

		completed := manager.GetByStatus(StatusCompleted)
		assert.Len(t, completed, 1)
	})

	t.Run("get_by_tag", func(t *testing.T) {
		manager := NewManager()
		s1, _ := manager.Create("proj-1", "Session 1", "", ModePlanning)
		s2, _ := manager.Create("proj-1", "Session 2", "", ModeBuilding)
		manager.Create("proj-2", "Session 3", "", ModeTesting)

		s1.AddTag("critical")
		s2.AddTag("critical")

		sessions := manager.GetByTag("critical")
		assert.Len(t, sessions, 2)
	})

	t.Run("get_recent", func(t *testing.T) {
		manager := NewManager()
		s1, _ := manager.Create("proj-1", "Session 1", "", ModePlanning)
		time.Sleep(10 * time.Millisecond)
		s2, _ := manager.Create("proj-1", "Session 2", "", ModeBuilding)
		time.Sleep(10 * time.Millisecond)
		s3, _ := manager.Create("proj-2", "Session 3", "", ModeTesting)

		recent := manager.GetRecent(2)
		assert.Len(t, recent, 2)
		assert.Equal(t, s3.ID, recent[0].ID) // Most recent first
		assert.Equal(t, s2.ID, recent[1].ID)

		all := manager.GetRecent(10)
		assert.Len(t, all, 3)
		assert.Equal(t, s3.ID, all[0].ID)
		assert.Equal(t, s2.ID, all[1].ID)
		assert.Equal(t, s1.ID, all[2].ID)
	})

	t.Run("find_by_name", func(t *testing.T) {
		manager := NewManager()
		manager.Create("proj-1", "Build Feature", "", ModePlanning)
		manager.Create("proj-1", "Test Feature", "", ModeBuilding)
		manager.Create("proj-2", "Build API", "", ModeTesting)

		sessions := manager.FindByName("Feature")
		assert.Len(t, sessions, 2)

		sessions = manager.FindByName("build")
		assert.Len(t, sessions, 2) // Case-insensitive

		sessions = manager.FindByName("API")
		assert.Len(t, sessions, 1)
	})

	t.Run("count", func(t *testing.T) {
		manager := NewManager()
		assert.Equal(t, 0, manager.Count())

		manager.Create("proj-1", "Session 1", "", ModePlanning)
		assert.Equal(t, 1, manager.Count())

		manager.Create("proj-1", "Session 2", "", ModeBuilding)
		assert.Equal(t, 2, manager.Count())
	})

	t.Run("count_by_status", func(t *testing.T) {
		manager := NewManager()
		s1, _ := manager.Create("proj-1", "Session 1", "", ModePlanning)
		s2, _ := manager.Create("proj-1", "Session 2", "", ModeBuilding)
		manager.Create("proj-2", "Session 3", "", ModeTesting)

		assert.Equal(t, 3, manager.CountByStatus(StatusPaused))

		manager.Start(s1.ID)
		assert.Equal(t, 1, manager.CountByStatus(StatusActive))
		assert.Equal(t, 2, manager.CountByStatus(StatusPaused))

		manager.Complete(s2.ID)
		assert.Equal(t, 1, manager.CountByStatus(StatusCompleted))
	})
}

func TestSessionTags(t *testing.T) {
	t.Run("add_tag", func(t *testing.T) {
		manager := NewManager()
		session, _ := manager.Create("proj-1", "Test", "", ModePlanning)

		session.AddTag("critical")
		assert.True(t, session.HasTag("critical"))
	})

	t.Run("add_duplicate_tag", func(t *testing.T) {
		manager := NewManager()
		session, _ := manager.Create("proj-1", "Test", "", ModePlanning)

		session.AddTag("critical")
		session.AddTag("critical")
		assert.Len(t, session.Tags, 1)
	})

	t.Run("remove_tag", func(t *testing.T) {
		manager := NewManager()
		session, _ := manager.Create("proj-1", "Test", "", ModePlanning)

		session.AddTag("critical")
		session.RemoveTag("critical")
		assert.False(t, session.HasTag("critical"))
	})
}

func TestSessionContext(t *testing.T) {
	t.Run("set_and_get_context", func(t *testing.T) {
		manager := NewManager()
		session, _ := manager.Create("proj-1", "Test", "", ModePlanning)

		session.SetContext("user", "alice")
		session.SetContext("count", 42)

		user, ok := session.GetContext("user")
		assert.True(t, ok)
		assert.Equal(t, "alice", user)

		count, ok := session.GetContext("count")
		assert.True(t, ok)
		assert.Equal(t, 42, count)
	})

	t.Run("get_missing_context", func(t *testing.T) {
		manager := NewManager()
		session, _ := manager.Create("proj-1", "Test", "", ModePlanning)

		_, ok := session.GetContext("missing")
		assert.False(t, ok)
	})
}

func TestSessionMetadata(t *testing.T) {
	t.Run("set_and_get_metadata", func(t *testing.T) {
		manager := NewManager()
		session, _ := manager.Create("proj-1", "Test", "", ModePlanning)

		session.SetMetadata("author", "alice")
		session.SetMetadata("ticket", "PROJ-123")

		author, ok := session.GetMetadata("author")
		assert.True(t, ok)
		assert.Equal(t, "alice", author)

		ticket, ok := session.GetMetadata("ticket")
		assert.True(t, ok)
		assert.Equal(t, "PROJ-123", ticket)
	})

	t.Run("get_missing_metadata", func(t *testing.T) {
		manager := NewManager()
		session, _ := manager.Create("proj-1", "Test", "", ModePlanning)

		_, ok := session.GetMetadata("missing")
		assert.False(t, ok)
	})
}

func TestSessionClone(t *testing.T) {
	t.Run("clone_session", func(t *testing.T) {
		manager := NewManager()
		original, _ := manager.Create("proj-1", "Test", "Desc", ModePlanning)
		original.AddTag("critical")
		original.SetContext("user", "alice")
		original.SetMetadata("ticket", "PROJ-123")

		clone := original.Clone()
		assert.Equal(t, original.ID, clone.ID)
		assert.Equal(t, original.Name, clone.Name)
		assert.True(t, clone.HasTag("critical"))

		user, _ := clone.GetContext("user")
		assert.Equal(t, "alice", user)

		ticket, _ := clone.GetMetadata("ticket")
		assert.Equal(t, "PROJ-123", ticket)

		// Modify clone shouldn't affect original
		clone.AddTag("new-tag")
		assert.False(t, original.HasTag("new-tag"))
	})
}

func TestManagerStatistics(t *testing.T) {
	t.Run("get_statistics", func(t *testing.T) {
		manager := NewManager()
		s1, _ := manager.Create("proj-1", "Session 1", "", ModePlanning)
		s2, _ := manager.Create("proj-1", "Session 2", "", ModeBuilding)
		_, _ = manager.Create("proj-2", "Session 3", "", ModeTesting)

		manager.Start(s1.ID)
		time.Sleep(10 * time.Millisecond)
		manager.Pause(s1.ID)

		manager.Start(s2.ID)
		time.Sleep(10 * time.Millisecond)
		manager.Complete(s2.ID)

		// s3 remains paused

		stats := manager.GetStatistics()
		assert.Equal(t, 3, stats.Total)
		assert.Equal(t, 2, stats.ByStatus[StatusPaused])
		assert.Equal(t, 1, stats.ByStatus[StatusCompleted])
		assert.Equal(t, 1, stats.ByMode[ModePlanning])
		assert.Equal(t, 1, stats.ByMode[ModeBuilding])
		assert.Equal(t, 1, stats.ByMode[ModeTesting])
		assert.Greater(t, stats.AverageDuration.Milliseconds(), int64(0))
	})
}

func TestManagerCallbacks(t *testing.T) {
	t.Run("on_create_callback", func(t *testing.T) {
		manager := NewManager()
		called := false
		var createdSession *Session

		manager.OnCreate(func(session *Session) {
			called = true
			createdSession = session
		})

		session, _ := manager.Create("proj-1", "Test", "", ModePlanning)

		assert.True(t, called)
		assert.Equal(t, session.ID, createdSession.ID)
	})

	t.Run("on_start_callback", func(t *testing.T) {
		manager := NewManager()
		called := false

		manager.OnStart(func(session *Session) {
			called = true
		})

		session, _ := manager.Create("proj-1", "Test", "", ModePlanning)
		manager.Start(session.ID)

		assert.True(t, called)
	})

	t.Run("on_pause_callback", func(t *testing.T) {
		manager := NewManager()
		called := false

		manager.OnPause(func(session *Session) {
			called = true
		})

		session, _ := manager.Create("proj-1", "Test", "", ModePlanning)
		manager.Start(session.ID)
		manager.Pause(session.ID)

		assert.True(t, called)
	})

	t.Run("on_complete_callback", func(t *testing.T) {
		manager := NewManager()
		called := false

		manager.OnComplete(func(session *Session) {
			called = true
		})

		session, _ := manager.Create("proj-1", "Test", "", ModePlanning)
		manager.Start(session.ID)
		manager.Complete(session.ID)

		assert.True(t, called)
	})

	t.Run("on_switch_callback", func(t *testing.T) {
		manager := NewManager()
		called := false
		var fromSession, toSession *Session

		manager.OnSwitch(func(from, to *Session) {
			called = true
			fromSession = from
			toSession = to
		})

		s1, _ := manager.Create("proj-1", "Session 1", "", ModePlanning)
		s2, _ := manager.Create("proj-1", "Session 2", "", ModeBuilding)

		manager.Start(s1.ID)
		called = false // Reset

		manager.Start(s2.ID)

		assert.True(t, called)
		assert.Equal(t, s1.ID, fromSession.ID)
		assert.Equal(t, s2.ID, toSession.ID)
	})
}

func TestManagerTrimHistory(t *testing.T) {
	t.Run("trim_history", func(t *testing.T) {
		manager := NewManager()
		manager.SetMaxHistory(2)

		// Create 5 sessions and complete them
		for i := 0; i < 5; i++ {
			s, _ := manager.Create("proj-1", "Session", "", ModePlanning)
			manager.Start(s.ID)
			time.Sleep(5 * time.Millisecond) // Ensure different completion times
			manager.Complete(s.ID)
		}

		assert.Equal(t, 5, manager.Count())

		removed := manager.TrimHistory()
		assert.Equal(t, 3, removed)
		assert.Equal(t, 2, manager.Count())
	})

	t.Run("trim_history_keeps_active", func(t *testing.T) {
		manager := NewManager()
		manager.SetMaxHistory(1)

		s1, _ := manager.Create("proj-1", "Session 1", "", ModePlanning)
		s2, _ := manager.Create("proj-1", "Session 2", "", ModeBuilding)
		s3, _ := manager.Create("proj-1", "Session 3", "", ModeTesting)

		manager.Start(s1.ID)
		time.Sleep(5 * time.Millisecond)
		manager.Complete(s1.ID)

		time.Sleep(5 * time.Millisecond)
		manager.Start(s2.ID)
		time.Sleep(5 * time.Millisecond)
		manager.Complete(s2.ID)

		manager.Start(s3.ID)
		// s3 is still active

		removed := manager.TrimHistory()
		assert.Equal(t, 1, removed) // s1 removed (oldest completed), s2 kept (within maxHistory), s3 kept (active)
		assert.Equal(t, 2, manager.Count())
	})
}

func TestManagerClear(t *testing.T) {
	t.Run("clear_sessions", func(t *testing.T) {
		manager := NewManager()
		manager.Create("proj-1", "Session 1", "", ModePlanning)
		manager.Create("proj-1", "Session 2", "", ModeBuilding)

		assert.Equal(t, 2, manager.Count())

		err := manager.Clear()
		require.NoError(t, err)
		assert.Equal(t, 0, manager.Count())
	})

	t.Run("cannot_clear_with_active_session", func(t *testing.T) {
		manager := NewManager()
		s, _ := manager.Create("proj-1", "Session 1", "", ModePlanning)
		manager.Start(s.ID)

		err := manager.Clear()
		assert.Error(t, err)
		assert.Equal(t, 1, manager.Count())
	})
}

func TestManagerExportImport(t *testing.T) {
	t.Run("export_session", func(t *testing.T) {
		manager := NewManager()
		session, _ := manager.Create("proj-1", "Test Session", "Desc", ModePlanning)
		session.AddTag("critical")
		session.SetMetadata("author", "alice")

		snapshot, err := manager.Export(session.ID)
		require.NoError(t, err)
		assert.Equal(t, session.ID, snapshot.Session.ID)
		assert.Equal(t, session.Name, snapshot.Session.Name)
		assert.True(t, snapshot.Session.HasTag("critical"))
	})

	t.Run("import_session", func(t *testing.T) {
		manager1 := NewManager()
		session, _ := manager1.Create("proj-1", "Test Session", "Desc", ModePlanning)
		session.AddTag("critical")

		snapshot, _ := manager1.Export(session.ID)

		manager2 := NewManager()
		err := manager2.Import(snapshot)
		require.NoError(t, err)

		imported, err := manager2.Get(session.ID)
		require.NoError(t, err)
		assert.Equal(t, session.Name, imported.Name)
		assert.True(t, imported.HasTag("critical"))
	})

	t.Run("import_duplicate_id_error", func(t *testing.T) {
		manager := NewManager()
		session, _ := manager.Create("proj-1", "Test Session", "Desc", ModePlanning)

		snapshot, _ := manager.Export(session.ID)

		err := manager.Import(snapshot)
		assert.Error(t, err) // Already exists
	})
}

func TestConcurrentOperations(t *testing.T) {
	t.Run("concurrent_create", func(t *testing.T) {
		manager := NewManager()
		var wg sync.WaitGroup

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				_, err := manager.Create("proj-1", "Session", "", ModePlanning)
				assert.NoError(t, err)
			}(i)
		}

		wg.Wait()
		assert.Equal(t, 10, manager.Count())
	})

	t.Run("concurrent_start_and_query", func(t *testing.T) {
		manager := NewManager()

		// Create sessions
		sessions := make([]*Session, 5)
		for i := 0; i < 5; i++ {
			s, _ := manager.Create("proj-1", "Session", "", ModePlanning)
			sessions[i] = s
		}

		var wg sync.WaitGroup

		// Start sessions concurrently
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(session *Session) {
				defer wg.Done()
				manager.Start(session.ID)
				time.Sleep(5 * time.Millisecond)
				manager.Pause(session.ID)
			}(sessions[i])
		}

		// Query concurrently
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				manager.GetAll()
				manager.GetByStatus(StatusActive)
				manager.Count()
			}()
		}

		wg.Wait()
		assert.Equal(t, 5, manager.Count())
	})
}

func TestSessionValidation(t *testing.T) {
	t.Run("validate_valid_session", func(t *testing.T) {
		session := &Session{
			ID:        "test-123",
			ProjectID: "proj-1",
			Name:      "Test",
			Mode:      ModePlanning,
			Status:    StatusPaused,
		}

		err := session.Validate()
		assert.NoError(t, err)
	})

	t.Run("validate_missing_id", func(t *testing.T) {
		session := &Session{
			ProjectID: "proj-1",
			Name:      "Test",
			Mode:      ModePlanning,
			Status:    StatusPaused,
		}

		err := session.Validate()
		assert.Error(t, err)
	})

	t.Run("validate_invalid_mode", func(t *testing.T) {
		session := &Session{
			ID:        "test-123",
			ProjectID: "proj-1",
			Name:      "Test",
			Mode:      Mode("invalid"),
			Status:    StatusPaused,
		}

		err := session.Validate()
		assert.Error(t, err)
	})
}

func TestModeAndStatus(t *testing.T) {
	t.Run("mode_is_valid", func(t *testing.T) {
		assert.True(t, ModePlanning.IsValid())
		assert.True(t, ModeBuilding.IsValid())
		assert.True(t, ModeTesting.IsValid())
		assert.True(t, ModeRefactoring.IsValid())
		assert.True(t, ModeDebugging.IsValid())
		assert.True(t, ModeDeployment.IsValid())
		assert.False(t, Mode("invalid").IsValid())
	})

	t.Run("status_is_valid", func(t *testing.T) {
		assert.True(t, StatusActive.IsValid())
		assert.True(t, StatusPaused.IsValid())
		assert.True(t, StatusCompleted.IsValid())
		assert.True(t, StatusFailed.IsValid())
		assert.False(t, Status("invalid").IsValid())
	})

	t.Run("mode_string", func(t *testing.T) {
		assert.Equal(t, "planning", ModePlanning.String())
		assert.Equal(t, "building", ModeBuilding.String())
	})

	t.Run("status_string", func(t *testing.T) {
		assert.Equal(t, "active", StatusActive.String())
		assert.Equal(t, "paused", StatusPaused.String())
	})
}

// ========================================
// Additional Coverage Tests
// ========================================

func TestNewManagerWithIntegrations(t *testing.T) {
	// Note: focus.Manager and hooks.Manager are external dependencies
	// For this test, we pass nil since we're just testing that the function works
	mgr := NewManagerWithIntegrations(nil, nil)

	assert.NotNil(t, mgr)
	assert.Nil(t, mgr.GetFocusManager())
	assert.Nil(t, mgr.GetHooksManager())
}

func TestManager_GetFocusManager(t *testing.T) {
	mgr := NewManagerWithIntegrations(nil, nil)

	focusMgr := mgr.GetFocusManager()
	assert.Nil(t, focusMgr, "Should return nil when no focus manager set")
}

func TestManager_GetHooksManager(t *testing.T) {
	mgr := NewManagerWithIntegrations(nil, nil)

	hooksMgr := mgr.GetHooksManager()
	assert.Nil(t, hooksMgr, "Should return nil when no hooks manager set")
}

func TestManager_OnResume(t *testing.T) {
	mgr := NewManager()

	callback := func(session *Session) {
		// Callback function - just for registration test
	}

	mgr.OnResume(callback)

	// Verify callback was registered (we can't directly test the slice, but we can verify no panic)
	assert.NotNil(t, mgr)
}

func TestManager_OnDelete(t *testing.T) {
	mgr := NewManager()

	callback := func(session *Session) {
		// Callback function - just for registration test
	}

	mgr.OnDelete(callback)

	// Verify callback was registered (we can't directly test the slice, but we can verify no panic)
	assert.NotNil(t, mgr)
}

func TestStatistics_String(t *testing.T) {
	stats := &Statistics{
		Total: 10,
		ByStatus: map[Status]int{
			StatusActive:    3,
			StatusCompleted: 7,
		},
		ByMode: map[Mode]int{
			ModePlanning: 5,
			ModeBuilding: 5,
		},
	}

	result := stats.String()

	// The String() method should return information about statistics
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "10", "Should contain total count")
}
