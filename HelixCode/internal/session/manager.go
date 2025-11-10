package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"dev.helix.code/internal/focus"
	"dev.helix.code/internal/hooks"
)

// Manager manages development sessions
type Manager struct {
	sessions      map[string]*Session // All sessions by ID
	activeSession *Session            // Currently active session
	focusManager  *focus.Manager      // Focus chain manager
	hooksManager  *hooks.Manager      // Hooks manager
	mu            sync.RWMutex        // Thread-safety
	onCreate      []SessionCallback   // Callbacks on session creation
	onStart       []SessionCallback   // Callbacks on session start
	onPause       []SessionCallback   // Callbacks on session pause
	onResume      []SessionCallback   // Callbacks on session resume
	onComplete    []SessionCallback   // Callbacks on session completion
	onDelete      []SessionCallback   // Callbacks on session deletion
	onSwitch      []SwitchCallback    // Callbacks on session switch
	maxHistory    int                 // Maximum sessions to keep
}

// SessionCallback is called for session lifecycle events
type SessionCallback func(*Session)

// SwitchCallback is called when switching sessions
type SwitchCallback func(from, to *Session)

// NewManager creates a new session manager
func NewManager() *Manager {
	return &Manager{
		sessions:     make(map[string]*Session),
		focusManager: focus.NewManager(),
		hooksManager: hooks.NewManager(),
		onCreate:     make([]SessionCallback, 0),
		onStart:      make([]SessionCallback, 0),
		onPause:      make([]SessionCallback, 0),
		onResume:     make([]SessionCallback, 0),
		onComplete:   make([]SessionCallback, 0),
		onDelete:     make([]SessionCallback, 0),
		onSwitch:     make([]SwitchCallback, 0),
		maxHistory:   100,
	}
}

// NewManagerWithIntegrations creates manager with existing focus and hooks managers
func NewManagerWithIntegrations(focusMgr *focus.Manager, hooksMgr *hooks.Manager) *Manager {
	m := NewManager()
	m.focusManager = focusMgr
	m.hooksManager = hooksMgr
	return m
}

// Create creates a new session
func (m *Manager) Create(projectID, name, description string, mode Mode) (*Session, error) {
	if projectID == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}

	if name == "" {
		return nil, fmt.Errorf("session name cannot be empty")
	}

	if !mode.IsValid() {
		return nil, fmt.Errorf("invalid session mode: %s", mode)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Create session
	session := &Session{
		ID:          generateSessionID(),
		ProjectID:   projectID,
		Name:        name,
		Description: description,
		Mode:        mode,
		Status:      StatusPaused,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Context:     make(map[string]interface{}),
		Metadata:    make(map[string]string),
		Tags:        make([]string, 0),
	}

	// Create dedicated focus chain for session
	chain, err := m.focusManager.CreateChain(fmt.Sprintf("session-%s", session.ID), false)
	if err != nil {
		return nil, fmt.Errorf("failed to create focus chain: %w", err)
	}
	session.FocusChainID = chain.ID

	// Store session
	m.sessions[session.ID] = session

	// Trigger callbacks
	for _, callback := range m.onCreate {
		callback(session)
	}

	// Emit hook
	m.emitHook(hooks.HookTypeCustom, "session_created", session)

	return session, nil
}

// Start starts a session and makes it active
func (m *Manager) Start(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if session.Status == StatusCompleted {
		return fmt.Errorf("cannot start completed session")
	}

	if session.Status == StatusFailed {
		return fmt.Errorf("cannot start failed session")
	}

	oldActive := m.activeSession
	session.Status = StatusActive
	session.UpdatedAt = time.Now()
	session.StartedAt = time.Now()
	m.activeSession = session

	// Set focus chain as active
	if session.FocusChainID != "" {
		m.focusManager.SetActiveChain(session.FocusChainID)
	}

	// Trigger callbacks
	for _, callback := range m.onStart {
		callback(session)
	}

	// Trigger switch callbacks
	if oldActive != session {
		for _, callback := range m.onSwitch {
			callback(oldActive, session)
		}
	}

	// Emit hook
	m.emitHook(hooks.HookTypeCustom, "session_started", session)

	return nil
}

// Pause pauses the active session
func (m *Manager) Pause(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if session.Status != StatusActive {
		return fmt.Errorf("session is not active")
	}

	session.Status = StatusPaused
	session.UpdatedAt = time.Now()

	// Update duration
	if !session.StartedAt.IsZero() {
		session.Duration += time.Since(session.StartedAt)
		session.StartedAt = time.Time{} // Reset
	}

	if m.activeSession == session {
		m.activeSession = nil
	}

	// Trigger callbacks
	for _, callback := range m.onPause {
		callback(session)
	}

	// Emit hook
	m.emitHook(hooks.HookTypeCustom, "session_paused", session)

	return nil
}

// Resume resumes a paused session
func (m *Manager) Resume(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if session.Status != StatusPaused {
		return fmt.Errorf("session is not paused")
	}

	oldActive := m.activeSession
	session.Status = StatusActive
	session.UpdatedAt = time.Now()
	session.StartedAt = time.Now()
	m.activeSession = session

	// Set focus chain as active
	if session.FocusChainID != "" {
		m.focusManager.SetActiveChain(session.FocusChainID)
	}

	// Trigger callbacks
	for _, callback := range m.onResume {
		callback(session)
	}

	// Trigger switch callbacks
	if oldActive != session {
		for _, callback := range m.onSwitch {
			callback(oldActive, session)
		}
	}

	// Emit hook
	m.emitHook(hooks.HookTypeCustom, "session_resumed", session)

	return nil
}

// Complete marks a session as completed
func (m *Manager) Complete(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if session.Status == StatusCompleted {
		return fmt.Errorf("session already completed")
	}

	session.Status = StatusCompleted
	session.UpdatedAt = time.Now()
	session.CompletedAt = time.Now()

	// Update duration for active sessions
	if !session.StartedAt.IsZero() {
		session.Duration += time.Since(session.StartedAt)
		session.StartedAt = time.Time{} // Reset
	}

	if m.activeSession == session {
		m.activeSession = nil
	}

	// Trigger callbacks
	for _, callback := range m.onComplete {
		callback(session)
	}

	// Emit hook
	m.emitHook(hooks.HookTypeCustom, "session_completed", session)

	return nil
}

// Fail marks a session as failed
func (m *Manager) Fail(sessionID string, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.Status = StatusFailed
	session.UpdatedAt = time.Now()
	session.SetMetadata("failure_reason", reason)

	// Update duration for active sessions
	if !session.StartedAt.IsZero() {
		session.Duration += time.Since(session.StartedAt)
		session.StartedAt = time.Time{} // Reset
	}

	if m.activeSession == session {
		m.activeSession = nil
	}

	// Emit hook
	m.emitHook(hooks.HookTypeCustom, "session_failed", session)

	return nil
}

// Delete deletes a session
func (m *Manager) Delete(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Don't delete active session
	if session.Status == StatusActive {
		return fmt.Errorf("cannot delete active session")
	}

	// Delete focus chain
	if session.FocusChainID != "" {
		m.focusManager.DeleteChain(session.FocusChainID)
	}

	// Remove session
	delete(m.sessions, sessionID)

	// Trigger callbacks
	for _, callback := range m.onDelete {
		callback(session)
	}

	// Emit hook
	m.emitHook(hooks.HookTypeCustom, "session_deleted", session)

	return nil
}

// Get retrieves a session by ID
func (m *Manager) Get(sessionID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session, nil
}

// GetActive returns the currently active session
func (m *Manager) GetActive() *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.activeSession
}

// GetAll returns all sessions
func (m *Manager) GetAll() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// GetByProject returns all sessions for a project
func (m *Manager) GetByProject(projectID string) []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0)
	for _, session := range m.sessions {
		if session.ProjectID == projectID {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// GetByMode returns all sessions with a specific mode
func (m *Manager) GetByMode(mode Mode) []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0)
	for _, session := range m.sessions {
		if session.Mode == mode {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// GetByStatus returns all sessions with a specific status
func (m *Manager) GetByStatus(status Status) []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0)
	for _, session := range m.sessions {
		if session.Status == status {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// GetByTag returns all sessions with a specific tag
func (m *Manager) GetByTag(tag string) []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0)
	for _, session := range m.sessions {
		if session.HasTag(tag) {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// GetRecent returns the N most recently updated sessions
func (m *Manager) GetRecent(n int) []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get all sessions
	all := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		all = append(all, session)
	}

	// Sort by UpdatedAt (descending)
	for i := 0; i < len(all)-1; i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].UpdatedAt.After(all[i].UpdatedAt) {
				all[i], all[j] = all[j], all[i]
			}
		}
	}

	// Return top N
	if n <= 0 || n > len(all) {
		n = len(all)
	}

	return all[:n]
}

// FindByName finds sessions by name (case-insensitive substring match)
func (m *Manager) FindByName(nameSubstring string) []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0)
	lowerSearch := toLower(nameSubstring)

	for _, session := range m.sessions {
		if contains(toLower(session.Name), lowerSearch) {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// Count returns the total number of sessions
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.sessions)
}

// CountByStatus returns count of sessions by status
func (m *Manager) CountByStatus(status Status) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, session := range m.sessions {
		if session.Status == status {
			count++
		}
	}

	return count
}

// Clear removes all sessions
func (m *Manager) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Don't clear if there's an active session
	if m.activeSession != nil {
		return fmt.Errorf("cannot clear while session is active")
	}

	// Delete all focus chains
	for _, session := range m.sessions {
		if session.FocusChainID != "" {
			m.focusManager.DeleteChain(session.FocusChainID)
		}
	}

	m.sessions = make(map[string]*Session)

	return nil
}

// GetFocusManager returns the focus manager
func (m *Manager) GetFocusManager() *focus.Manager {
	return m.focusManager
}

// GetHooksManager returns the hooks manager
func (m *Manager) GetHooksManager() *hooks.Manager {
	return m.hooksManager
}

// GetStatistics returns session statistics
func (m *Manager) GetStatistics() *Statistics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &Statistics{
		Total:    len(m.sessions),
		ByStatus: make(map[Status]int),
		ByMode:   make(map[Mode]int),
	}

	var totalDuration time.Duration
	for _, session := range m.sessions {
		stats.ByStatus[session.Status]++
		stats.ByMode[session.Mode]++

		// Calculate total duration
		duration := session.Duration
		if !session.StartedAt.IsZero() {
			// Session is currently active
			duration += time.Since(session.StartedAt)
		}
		totalDuration += duration
	}

	if stats.Total > 0 {
		stats.AverageDuration = totalDuration / time.Duration(stats.Total)
	}

	return stats
}

// OnCreate registers a callback for session creation
func (m *Manager) OnCreate(callback SessionCallback) {
	m.onCreate = append(m.onCreate, callback)
}

// OnStart registers a callback for session start
func (m *Manager) OnStart(callback SessionCallback) {
	m.onStart = append(m.onStart, callback)
}

// OnPause registers a callback for session pause
func (m *Manager) OnPause(callback SessionCallback) {
	m.onPause = append(m.onPause, callback)
}

// OnResume registers a callback for session resume
func (m *Manager) OnResume(callback SessionCallback) {
	m.onResume = append(m.onResume, callback)
}

// OnComplete registers a callback for session completion
func (m *Manager) OnComplete(callback SessionCallback) {
	m.onComplete = append(m.onComplete, callback)
}

// OnDelete registers a callback for session deletion
func (m *Manager) OnDelete(callback SessionCallback) {
	m.onDelete = append(m.onDelete, callback)
}

// OnSwitch registers a callback for session switching
func (m *Manager) OnSwitch(callback SwitchCallback) {
	m.onSwitch = append(m.onSwitch, callback)
}

// SetMaxHistory sets the maximum number of sessions to keep
func (m *Manager) SetMaxHistory(max int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.maxHistory = max
}

// TrimHistory removes old completed sessions beyond maxHistory
func (m *Manager) TrimHistory() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get completed sessions
	completed := make([]*Session, 0)
	for _, session := range m.sessions {
		if session.Status == StatusCompleted || session.Status == StatusFailed {
			completed = append(completed, session)
		}
	}

	// Sort by completion time (oldest first)
	for i := 0; i < len(completed)-1; i++ {
		for j := i + 1; j < len(completed); j++ {
			if completed[i].CompletedAt.After(completed[j].CompletedAt) {
				completed[i], completed[j] = completed[j], completed[i]
			}
		}
	}

	// Remove oldest if exceeds maxHistory
	removed := 0
	if len(completed) > m.maxHistory {
		toRemove := len(completed) - m.maxHistory
		for i := 0; i < toRemove; i++ {
			session := completed[i]
			// Delete focus chain
			if session.FocusChainID != "" {
				m.focusManager.DeleteChain(session.FocusChainID)
			}
			delete(m.sessions, session.ID)
			removed++
		}
	}

	return removed
}

// emitHook emits a hook event
func (m *Manager) emitHook(hookType hooks.HookType, eventName string, session *Session) {
	event := hooks.NewEventWithContext(context.Background(), hookType)
	event.SetData("event", eventName)
	event.SetData("session_id", session.ID)
	event.SetData("session_name", session.Name)
	event.SetData("project_id", session.ProjectID)
	event.SetData("mode", string(session.Mode))
	event.SetData("status", string(session.Status))
	event.Source = "session-manager"

	m.hooksManager.TriggerEvent(event)
}

// Statistics contains session statistics
type Statistics struct {
	Total           int            // Total sessions
	ByStatus        map[Status]int // Count by status
	ByMode          map[Mode]int   // Count by mode
	AverageDuration time.Duration  // Average session duration
}

// String returns a string representation of the statistics
func (s *Statistics) String() string {
	return fmt.Sprintf("Sessions: %d total, Avg Duration: %v", s.Total, s.AverageDuration)
}

// Export exports session metadata (without focus chain)
func (m *Manager) Export(sessionID string) (*SessionSnapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Export focus chain
	var focusChain *focus.ChainSnapshot
	if session.FocusChainID != "" {
		chain, err := m.focusManager.ExportChain(session.FocusChainID)
		if err == nil {
			focusChain = chain
		}
	}

	return &SessionSnapshot{
		Session:    session.Clone(),
		FocusChain: focusChain,
	}, nil
}

// Import imports a session from snapshot
func (m *Manager) Import(snapshot *SessionSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for duplicate ID
	if _, exists := m.sessions[snapshot.Session.ID]; exists {
		return fmt.Errorf("session with ID '%s' already exists", snapshot.Session.ID)
	}

	// Import focus chain if present
	if snapshot.FocusChain != nil {
		err := m.focusManager.ImportChain(snapshot.FocusChain, false)
		if err != nil {
			return fmt.Errorf("failed to import focus chain: %w", err)
		}
		// Use the chain ID from snapshot
		snapshot.Session.FocusChainID = snapshot.FocusChain.Chain.ID
	}

	// Store session
	m.sessions[snapshot.Session.ID] = snapshot.Session

	return nil
}

// SessionSnapshot represents exported session data
type SessionSnapshot struct {
	Session    *Session             `json:"session"`
	FocusChain *focus.ChainSnapshot `json:"focus_chain,omitempty"`
}

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return fmt.Sprintf("session-%d", time.Now().UnixNano())
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// toLower converts a string to lowercase
func toLower(s string) string {
	result := ""
	for _, ch := range s {
		if ch >= 'A' && ch <= 'Z' {
			result += string(ch + 32)
		} else {
			result += string(ch)
		}
	}
	return result
}
