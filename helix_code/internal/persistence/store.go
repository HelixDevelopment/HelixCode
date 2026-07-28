package persistence

import (
	stdctx "context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"dev.helix.code/internal/focus"
	"dev.helix.code/internal/memory"
	"dev.helix.code/internal/session"
	"dev.helix.code/internal/template"
)

// Store manages persistent storage of application state
type Store struct {
	basePath         string            // Base directory for storage
	autoSaveEnabled  bool              // Whether auto-save is enabled
	autoSaveInterval time.Duration     // Auto-save interval
	serializer       Serializer        // Serialization strategy
	sessionMgr       *session.Manager  // Session manager reference
	memoryMgr        *memory.Manager   // Memory manager reference
	focusMgr         *focus.Manager    // Focus manager reference
	templateMgr      *template.Manager // Template manager reference
	mu               sync.RWMutex      // Thread-safety
	stopAutoSave     chan struct{}     // Signal to stop auto-save
	lastSaveTime     time.Time         // Last save timestamp
	onSave           []SaveCallback    // Callbacks on save
	onLoad           []LoadCallback    // Callbacks on load
	onError          []ErrorCallback   // Callbacks on error
}

// SaveCallback is called after successful save
type SaveCallback func(*SaveMetadata)

// LoadCallback is called after successful load
type LoadCallback func(*LoadMetadata)

// ErrorCallback is called on errors
type ErrorCallback func(error)

// SaveMetadata contains information about a save operation
type SaveMetadata struct {
	Path      string    // Save path
	Format    Format    // Serialization format
	Size      int64     // Total size in bytes
	Timestamp time.Time // Save time
	Items     int       // Number of items saved
}

// LoadMetadata contains information about a load operation
type LoadMetadata struct {
	Path      string    // Load path
	Format    Format    // Serialization format
	Size      int64     // Total size in bytes
	Timestamp time.Time // Load time
	Items     int       // Number of items loaded
}

// NewStore creates a new persistence store
func NewStore(basePath string) (*Store, error) {
	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("%s: %w",
			tr(stdctx.Background(), "internal_persistence_base_path_create_failed",
				map[string]any{"Error": err.Error()}), err)
	}

	return &Store{
		basePath:         basePath,
		autoSaveEnabled:  false,
		autoSaveInterval: 5 * time.Minute,
		serializer:       NewJSONSerializer(),
		stopAutoSave:     make(chan struct{}),
		onSave:           make([]SaveCallback, 0),
		onLoad:           make([]LoadCallback, 0),
		onError:          make([]ErrorCallback, 0),
	}, nil
}

// SetSessionManager sets the session manager reference
func (s *Store) SetSessionManager(mgr *session.Manager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionMgr = mgr
}

// SetMemoryManager sets the memory manager reference
func (s *Store) SetMemoryManager(mgr *memory.Manager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.memoryMgr = mgr
}

// SetFocusManager sets the focus manager reference
func (s *Store) SetFocusManager(mgr *focus.Manager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.focusMgr = mgr
}

// SetTemplateManager sets the template manager reference
func (s *Store) SetTemplateManager(mgr *template.Manager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.templateMgr = mgr
}

// SetSerializer sets the serialization strategy
func (s *Store) SetSerializer(serializer Serializer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.serializer = serializer
}

// EnableAutoSave enables automatic saving at intervals.
//
// Channel-lifecycle correctness: each enable creates a FRESH stop channel
// (s.stopAutoSave = make(chan struct{})) before starting the loop, and that
// channel is captured and passed INTO autoSaveLoop. This makes the
// enable→disable→re-enable cycle correct: a prior loop already exited on its own
// (now-closed) channel, and the re-enabled loop selects on the NEW channel rather
// than an already-closed one (which would have returned immediately, silently
// killing auto-save). Double-enable (without an intervening disable) is a guarded
// no-op so we never leak a second goroutine. All channel/flag mutation happens
// under s.mu so it is race-free against DisableAutoSave and against the loop,
// which now reads only its captured-by-value parameters (never s.stopAutoSave /
// s.autoSaveInterval unguarded). §11.4.85.
func (s *Store) EnableAutoSave(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.autoSaveEnabled {
		return
	}

	s.autoSaveEnabled = true
	s.autoSaveInterval = interval

	// Fresh channel per enable — the previous one (if any) was closed by the
	// matching DisableAutoSave and its loop has already exited.
	stop := make(chan struct{})
	s.stopAutoSave = stop

	go s.autoSaveLoop(interval, stop)
}

// DisableAutoSave disables automatic saving. It closes the current stop channel
// exactly once (guarded by the autoSaveEnabled flag + s.mu) so:
//   - double-disable without an intervening enable is a safe no-op (the flag is
//     already false), and
//   - disable-after-re-enable closes the FRESH channel created by that re-enable,
//     never the already-closed prior channel (which would panic
//     "close of closed channel").
func (s *Store) DisableAutoSave() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.autoSaveEnabled {
		return
	}

	s.autoSaveEnabled = false
	close(s.stopAutoSave)
}

// autoSaveLoop runs the auto-save loop. The interval and stop channel are passed
// BY VALUE (captured under s.mu by EnableAutoSave) so the loop never reads the
// mutable s.autoSaveInterval / s.stopAutoSave fields without a lock — eliminating
// the data race a concurrent EnableAutoSave/DisableAutoSave would otherwise cause.
func (s *Store) autoSaveLoop(interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if !s.autoSaveTick(ticker.C, stop) {
			return
		}
	}
}

// autoSaveTick runs exactly one iteration of the auto-save loop's select
// logic. Extracted from autoSaveLoop (identical behaviour, same select
// statement) so the §11.4.115 "unprioritized select" regression guard
// (TestAutoSaveTick_StopAlwaysWinsOverPendingTick) can force the race
// deterministically — pre-buffering a tick AND pre-closing stop before the
// select ever runs — instead of depending on real wall-clock ticker timing or
// scheduler luck.
//
// ANTI-BLUFF FIX (§11.4.115 unprioritized-select race, third confirmed
// instance in this codebase after helix_agent's lazy_provider.go
// createProviderWithContext and debate_log_repository.go
// StartCleanupWorker): Go's `select` chooses UNIFORMLY AT RANDOM among ALL
// cases ready at the instant it is evaluated. A naked
// `select { case <-ticker.C: ...; case <-stop: return }` can therefore still
// pick the ticker.C branch and run one more SaveAll() even when stop was
// ALREADY closed before the select was reached — if the goroutine was
// scheduler-delayed long enough for a tick to also land in ticker.C by the
// time it finally runs. The non-blocking priority pre-check below closes
// that gap: it runs FIRST, on every loop iteration, and returns immediately
// whenever stop is already closed, so the caller's disable decision is never
// second-guessed by a coincidentally-pending tick in the blocking select
// that follows.
//
// Returns false when the loop must stop (stop was signalled — checked FIRST,
// non-blocking, and again in the blocking select below), true when a tick
// was processed and the loop should continue.
func (s *Store) autoSaveTick(tickerC <-chan time.Time, stop <-chan struct{}) bool {
	// Priority pre-check: if stop is already closed, return immediately
	// without ever entering the blocking select below, where an
	// also-ready ticker.C could otherwise be picked instead.
	select {
	case <-stop:
		return false
	default:
	}

	select {
	case <-tickerC:
		if err := s.SaveAll(); err != nil {
			s.triggerError(err)
		}
		return true
	case <-stop:
		return false
	}
}

// SaveAll saves all application state
func (s *Store) SaveAll() error {
	s.mu.Lock()
	// NOTE: callbacks are dispatched AFTER the lock is released (see below) so a
	// callback may safely re-enter the Store (e.g. register another callback or
	// read state) without a reentrant-deadlock, and so arbitrary user code never
	// runs under the write lock. §11.4.85.
	unlocked := false
	defer func() {
		if !unlocked {
			s.mu.Unlock()
		}
	}()

	totalSize := int64(0)
	totalItems := 0

	// Save sessions
	if s.sessionMgr != nil {
		size, items, err := s.saveSessions()
		if err != nil {
			return fmt.Errorf("%s: %w",
				tr(stdctx.Background(), "internal_persistence_save_sessions_failed",
					map[string]any{"Error": err.Error()}), err)
		}
		totalSize += size
		totalItems += items
	}

	// Save conversations
	if s.memoryMgr != nil {
		size, items, err := s.saveConversations()
		if err != nil {
			return fmt.Errorf("%s: %w",
				tr(stdctx.Background(), "internal_persistence_save_conversations_failed",
					map[string]any{"Error": err.Error()}), err)
		}
		totalSize += size
		totalItems += items
	}

	// Save focus chains
	if s.focusMgr != nil {
		size, items, err := s.saveFocusChains()
		if err != nil {
			return fmt.Errorf("%s: %w",
				tr(stdctx.Background(), "internal_persistence_save_focus_chains_failed",
					map[string]any{"Error": err.Error()}), err)
		}
		totalSize += size
		totalItems += items
	}

	s.lastSaveTime = time.Now()

	// Trigger callbacks
	metadata := &SaveMetadata{
		Path:      s.basePath,
		Format:    s.serializer.Format(),
		Size:      totalSize,
		Timestamp: time.Now(),
		Items:     totalItems,
	}

	// Snapshot the callback slice under the lock, then release the lock and
	// dispatch outside it (reentrancy- and race-safe).
	callbacks := make([]SaveCallback, len(s.onSave))
	copy(callbacks, s.onSave)
	s.mu.Unlock()
	unlocked = true

	for _, callback := range callbacks {
		invokeSaveCallback(callback, metadata)
	}

	return nil
}

// invokeSaveCallback dispatches a single OnSave callback with panic isolation.
// A user-supplied callback that panics MUST NOT crash the host process nor leave
// the Store's write lock leaked (the callbacks run while SaveAll holds s.mu);
// the recovered panic is logged as a CONST-046-compliant message and swallowed
// so the remaining callbacks and the save itself complete. §11.4.85(B).
func invokeSaveCallback(callback SaveCallback, metadata *SaveMetadata) {
	defer func() {
		if p := recover(); p != nil {
			log.Print(tr(stdctx.Background(), "internal_persistence_save_callback_panicked",
				map[string]any{"Panic": fmt.Sprintf("%v", p)}))
		}
	}()
	callback(metadata)
}

// saveSessions saves all sessions (internal, no lock)
func (s *Store) saveSessions() (int64, int, error) {
	sessions := s.sessionMgr.GetAll()
	if len(sessions) == 0 {
		return 0, 0, nil
	}

	// Create sessions directory
	sessionsPath := filepath.Join(s.basePath, "sessions")
	if err := os.MkdirAll(sessionsPath, 0755); err != nil {
		return 0, 0, err
	}

	// Export each session. A per-item failure (export / serialize / write) is a
	// real persistence failure, NOT something to silently swallow: a save where
	// items were dropped MUST surface an error and report only the items that were
	// actually persisted. Reporting len(sessions) saved while a write failed is a
	// §11.4 / Article XI §11.9 PASS-bluff — the user believes their data is durable
	// when it is not (silent data loss on next restart). §11.4.135.
	totalSize := int64(0)
	saved := 0
	var firstErr error
	failed := 0
	for _, sess := range sessions {
		snapshot, err := s.sessionMgr.Export(sess.ID)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			failed++
			continue
		}

		// Serialize
		data, err := s.serializer.Serialize(snapshot)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			failed++
			continue
		}

		// Write to file
		filename := filepath.Join(sessionsPath, sess.ID+s.serializer.Extension())
		if err := writeAtomic(filename, data); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			failed++
			continue
		}

		totalSize += int64(len(data))
		saved++
	}

	if failed > 0 {
		return totalSize, saved, fmt.Errorf("%s: %w",
			tr(stdctx.Background(), "internal_persistence_save_items_partial_failure",
				map[string]any{"Failed": fmt.Sprintf("%d", failed), "Total": fmt.Sprintf("%d", len(sessions))}),
			firstErr)
	}

	return totalSize, saved, nil
}

// saveConversations saves all conversations (internal, no lock)
func (s *Store) saveConversations() (int64, int, error) {
	conversations := s.memoryMgr.GetAll()
	if len(conversations) == 0 {
		return 0, 0, nil
	}

	// Create conversations directory
	convsPath := filepath.Join(s.basePath, "conversations")
	if err := os.MkdirAll(convsPath, 0755); err != nil {
		return 0, 0, err
	}

	// Export each conversation. Per-item failures are surfaced (not swallowed) and
	// the reported count reflects only what was actually persisted — see
	// saveSessions for the §11.4 / §11.9 PASS-bluff rationale.
	totalSize := int64(0)
	saved := 0
	var firstErr error
	failed := 0
	for _, conv := range conversations {
		snapshot, err := s.memoryMgr.Export(conv.ID)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			failed++
			continue
		}

		// Serialize
		data, err := s.serializer.Serialize(snapshot)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			failed++
			continue
		}

		// Write to file
		filename := filepath.Join(convsPath, conv.ID+s.serializer.Extension())
		if err := writeAtomic(filename, data); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			failed++
			continue
		}

		totalSize += int64(len(data))
		saved++
	}

	if failed > 0 {
		return totalSize, saved, fmt.Errorf("%s: %w",
			tr(stdctx.Background(), "internal_persistence_save_items_partial_failure",
				map[string]any{"Failed": fmt.Sprintf("%d", failed), "Total": fmt.Sprintf("%d", len(conversations))}),
			firstErr)
	}

	return totalSize, saved, nil
}

// saveFocusChains saves all focus chains (internal, no lock)
func (s *Store) saveFocusChains() (int64, int, error) {
	chains := s.focusMgr.GetAllChains()
	if len(chains) == 0 {
		return 0, 0, nil
	}

	// Create focus directory
	focusPath := filepath.Join(s.basePath, "focus")
	if err := os.MkdirAll(focusPath, 0755); err != nil {
		return 0, 0, err
	}

	// Export each chain. Per-item failures are surfaced (not swallowed) and the
	// reported count reflects only what was actually persisted — see saveSessions
	// for the §11.4 / §11.9 PASS-bluff rationale.
	totalSize := int64(0)
	saved := 0
	var firstErr error
	failed := 0
	for _, chain := range chains {
		snapshot, err := s.focusMgr.ExportChain(chain.ID)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			failed++
			continue
		}

		// Serialize
		data, err := s.serializer.Serialize(snapshot)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			failed++
			continue
		}

		// Write to file
		filename := filepath.Join(focusPath, chain.ID+s.serializer.Extension())
		if err := writeAtomic(filename, data); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			failed++
			continue
		}

		totalSize += int64(len(data))
		saved++
	}

	if failed > 0 {
		return totalSize, saved, fmt.Errorf("%s: %w",
			tr(stdctx.Background(), "internal_persistence_save_items_partial_failure",
				map[string]any{"Failed": fmt.Sprintf("%d", failed), "Total": fmt.Sprintf("%d", len(chains))}),
			firstErr)
	}

	return totalSize, saved, nil
}

// LoadAll loads all application state
func (s *Store) LoadAll() error {
	s.mu.Lock()
	// Callbacks dispatched after the lock is released — see SaveAll's note.
	unlocked := false
	defer func() {
		if !unlocked {
			s.mu.Unlock()
		}
	}()

	totalSize := int64(0)
	totalItems := 0

	// Load sessions
	if s.sessionMgr != nil {
		size, items, err := s.loadSessions()
		if err != nil {
			return fmt.Errorf("%s: %w",
				tr(stdctx.Background(), "internal_persistence_load_sessions_failed",
					map[string]any{"Error": err.Error()}), err)
		}
		totalSize += size
		totalItems += items
	}

	// Load conversations
	if s.memoryMgr != nil {
		size, items, err := s.loadConversations()
		if err != nil {
			return fmt.Errorf("%s: %w",
				tr(stdctx.Background(), "internal_persistence_load_conversations_failed",
					map[string]any{"Error": err.Error()}), err)
		}
		totalSize += size
		totalItems += items
	}

	// Load focus chains
	if s.focusMgr != nil {
		size, items, err := s.loadFocusChains()
		if err != nil {
			return fmt.Errorf("%s: %w",
				tr(stdctx.Background(), "internal_persistence_load_focus_chains_failed",
					map[string]any{"Error": err.Error()}), err)
		}
		totalSize += size
		totalItems += items
	}

	// Trigger callbacks
	metadata := &LoadMetadata{
		Path:      s.basePath,
		Format:    s.serializer.Format(),
		Size:      totalSize,
		Timestamp: time.Now(),
		Items:     totalItems,
	}

	callbacks := make([]LoadCallback, len(s.onLoad))
	copy(callbacks, s.onLoad)
	s.mu.Unlock()
	unlocked = true

	for _, callback := range callbacks {
		invokeLoadCallback(callback, metadata)
	}

	return nil
}

// invokeLoadCallback dispatches a single OnLoad callback with panic isolation.
// Mirrors invokeSaveCallback: the callbacks run while LoadAll holds s.mu, so a
// panicking callback would otherwise crash the process and could leak the lock.
// §11.4.85(B).
func invokeLoadCallback(callback LoadCallback, metadata *LoadMetadata) {
	defer func() {
		if p := recover(); p != nil {
			log.Print(tr(stdctx.Background(), "internal_persistence_load_callback_panicked",
				map[string]any{"Panic": fmt.Sprintf("%v", p)}))
		}
	}()
	callback(metadata)
}

// loadSessions loads all sessions (internal, no lock)
func (s *Store) loadSessions() (int64, int, error) {
	sessionsPath := filepath.Join(s.basePath, "sessions")
	if _, err := os.Stat(sessionsPath); os.IsNotExist(err) {
		return 0, 0, nil
	}

	// Read all session files
	entries, err := os.ReadDir(sessionsPath)
	if err != nil {
		return 0, 0, err
	}

	totalSize := int64(0)
	loaded := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Read file
		filename := filepath.Join(sessionsPath, entry.Name())
		data, err := os.ReadFile(filename)
		if err != nil {
			continue
		}

		// Deserialize
		var snapshot session.SessionSnapshot
		if err := s.serializer.Deserialize(data, &snapshot); err != nil {
			continue
		}

		// Import
		if err := s.sessionMgr.Import(&snapshot); err != nil {
			continue
		}

		totalSize += int64(len(data))
		loaded++
	}

	return totalSize, loaded, nil
}

// loadConversations loads all conversations (internal, no lock)
func (s *Store) loadConversations() (int64, int, error) {
	convsPath := filepath.Join(s.basePath, "conversations")
	if _, err := os.Stat(convsPath); os.IsNotExist(err) {
		return 0, 0, nil
	}

	// Read all conversation files
	entries, err := os.ReadDir(convsPath)
	if err != nil {
		return 0, 0, err
	}

	totalSize := int64(0)
	loaded := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Read file
		filename := filepath.Join(convsPath, entry.Name())
		data, err := os.ReadFile(filename)
		if err != nil {
			continue
		}

		// Deserialize
		var snapshot memory.ConversationSnapshot
		if err := s.serializer.Deserialize(data, &snapshot); err != nil {
			continue
		}

		// Import
		if err := s.memoryMgr.Import(&snapshot); err != nil {
			continue
		}

		totalSize += int64(len(data))
		loaded++
	}

	return totalSize, loaded, nil
}

// loadFocusChains loads all focus chains (internal, no lock)
func (s *Store) loadFocusChains() (int64, int, error) {
	focusPath := filepath.Join(s.basePath, "focus")
	if _, err := os.Stat(focusPath); os.IsNotExist(err) {
		return 0, 0, nil
	}

	// Read all focus files
	entries, err := os.ReadDir(focusPath)
	if err != nil {
		return 0, 0, err
	}

	totalSize := int64(0)
	loaded := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Read file
		filename := filepath.Join(focusPath, entry.Name())
		data, err := os.ReadFile(filename)
		if err != nil {
			continue
		}

		// Deserialize
		var snapshot focus.ChainSnapshot
		if err := s.serializer.Deserialize(data, &snapshot); err != nil {
			continue
		}

		// Import (setActive = false for loading)
		if err := s.focusMgr.ImportChain(&snapshot, false); err != nil {
			continue
		}

		totalSize += int64(len(data))
		loaded++
	}

	return totalSize, loaded, nil
}

// Clear removes all persisted data
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove all subdirectories
	subdirs := []string{"sessions", "conversations", "focus"}
	for _, subdir := range subdirs {
		path := filepath.Join(s.basePath, subdir)
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}

	return nil
}

// Backup creates a backup of all data
func (s *Store) Backup(backupPath string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create backup directory
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return err
	}

	// Copy all subdirectories
	subdirs := []string{"sessions", "conversations", "focus"}
	for _, subdir := range subdirs {
		srcPath := filepath.Join(s.basePath, subdir)
		dstPath := filepath.Join(backupPath, subdir)

		if err := copyDir(srcPath, dstPath); err != nil {
			return err
		}
	}

	return nil
}

// Restore restores data from a backup
func (s *Store) Restore(backupPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Copy all subdirectories
	subdirs := []string{"sessions", "conversations", "focus"}
	for _, subdir := range subdirs {
		srcPath := filepath.Join(backupPath, subdir)
		dstPath := filepath.Join(s.basePath, subdir)

		// Remove existing
		if err := os.RemoveAll(dstPath); err != nil {
			return err
		}

		// Copy from backup
		if err := copyDir(srcPath, dstPath); err != nil {
			return err
		}
	}

	return nil
}

// GetLastSaveTime returns the last save time
func (s *Store) GetLastSaveTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastSaveTime
}

// OnSave registers a save callback. Guarded by s.mu so callback registration is
// race-free against the SaveAll path that reads s.onSave (§11.4.85 — concurrent
// registration + dispatch was a data race before this lock).
func (s *Store) OnSave(callback SaveCallback) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onSave = append(s.onSave, callback)
}

// OnLoad registers a load callback. Guarded by s.mu (see OnSave).
func (s *Store) OnLoad(callback LoadCallback) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onLoad = append(s.onLoad, callback)
}

// OnError registers an error callback. Guarded by s.mu (see OnSave).
func (s *Store) OnError(callback ErrorCallback) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onError = append(s.onError, callback)
}

// Load loads all persistent data (backward compatibility method)
func (s *Store) Load() error {
	return s.LoadAll()
}

// Save saves all persistent data (backward compatibility method)
func (s *Store) Save() error {
	return s.SaveAll()
}

// triggerError triggers error callbacks. The callback slice is snapshotted under
// RLock (race-free against concurrent OnError registration) and dispatched
// outside the lock so a callback may safely re-enter the Store. §11.4.85.
func (s *Store) triggerError(err error) {
	s.mu.RLock()
	callbacks := make([]ErrorCallback, len(s.onError))
	copy(callbacks, s.onError)
	s.mu.RUnlock()

	for _, callback := range callbacks {
		invokeErrorCallback(callback, err)
	}
}

// invokeErrorCallback dispatches a single OnError callback with panic isolation.
// triggerError is called from the autoSaveLoop background goroutine; a panicking
// error callback there would crash the whole process with no caller to recover
// it. The recovered panic is logged (CONST-046) and swallowed. §11.4.85(B).
func invokeErrorCallback(callback ErrorCallback, err error) {
	defer func() {
		if p := recover(); p != nil {
			log.Print(tr(stdctx.Background(), "internal_persistence_error_callback_panicked",
				map[string]any{"Panic": fmt.Sprintf("%v", p)}))
		}
	}()
	callback(err)
}

// writeAtomic writes data atomically (write to temp, then rename)
func writeAtomic(filename string, data []byte) error {
	// Write to temp file
	tempFile := filename + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return err
	}

	// Rename to final name
	return os.Rename(tempFile, filename)
}

// copyDir copies a directory recursively
func copyDir(src, dst string) error {
	// Check if source exists
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil // Source doesn't exist, skip
	}

	// Create destination
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}
