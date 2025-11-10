package persistence

import (
	"fmt"
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
		return nil, fmt.Errorf("failed to create base path: %w", err)
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

// EnableAutoSave enables automatic saving at intervals
func (s *Store) EnableAutoSave(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.autoSaveEnabled {
		return
	}

	s.autoSaveEnabled = true
	s.autoSaveInterval = interval

	go s.autoSaveLoop()
}

// DisableAutoSave disables automatic saving
func (s *Store) DisableAutoSave() {
	s.mu.Lock()
	if !s.autoSaveEnabled {
		s.mu.Unlock()
		return
	}

	s.autoSaveEnabled = false
	s.mu.Unlock()

	close(s.stopAutoSave)
}

// autoSaveLoop runs the auto-save loop
func (s *Store) autoSaveLoop() {
	ticker := time.NewTicker(s.autoSaveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.SaveAll(); err != nil {
				s.triggerError(err)
			}
		case <-s.stopAutoSave:
			return
		}
	}
}

// SaveAll saves all application state
func (s *Store) SaveAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	totalSize := int64(0)
	totalItems := 0

	// Save sessions
	if s.sessionMgr != nil {
		size, items, err := s.saveSessions()
		if err != nil {
			return fmt.Errorf("failed to save sessions: %w", err)
		}
		totalSize += size
		totalItems += items
	}

	// Save conversations
	if s.memoryMgr != nil {
		size, items, err := s.saveConversations()
		if err != nil {
			return fmt.Errorf("failed to save conversations: %w", err)
		}
		totalSize += size
		totalItems += items
	}

	// Save focus chains
	if s.focusMgr != nil {
		size, items, err := s.saveFocusChains()
		if err != nil {
			return fmt.Errorf("failed to save focus chains: %w", err)
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

	for _, callback := range s.onSave {
		callback(metadata)
	}

	return nil
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

	// Export each session
	totalSize := int64(0)
	for _, sess := range sessions {
		snapshot, err := s.sessionMgr.Export(sess.ID)
		if err != nil {
			continue
		}

		// Serialize
		data, err := s.serializer.Serialize(snapshot)
		if err != nil {
			continue
		}

		// Write to file
		filename := filepath.Join(sessionsPath, sess.ID+s.serializer.Extension())
		if err := writeAtomic(filename, data); err != nil {
			continue
		}

		totalSize += int64(len(data))
	}

	return totalSize, len(sessions), nil
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

	// Export each conversation
	totalSize := int64(0)
	for _, conv := range conversations {
		snapshot, err := s.memoryMgr.Export(conv.ID)
		if err != nil {
			continue
		}

		// Serialize
		data, err := s.serializer.Serialize(snapshot)
		if err != nil {
			continue
		}

		// Write to file
		filename := filepath.Join(convsPath, conv.ID+s.serializer.Extension())
		if err := writeAtomic(filename, data); err != nil {
			continue
		}

		totalSize += int64(len(data))
	}

	return totalSize, len(conversations), nil
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

	// Export each chain
	totalSize := int64(0)
	for _, chain := range chains {
		snapshot, err := s.focusMgr.ExportChain(chain.ID)
		if err != nil {
			continue
		}

		// Serialize
		data, err := s.serializer.Serialize(snapshot)
		if err != nil {
			continue
		}

		// Write to file
		filename := filepath.Join(focusPath, chain.ID+s.serializer.Extension())
		if err := writeAtomic(filename, data); err != nil {
			continue
		}

		totalSize += int64(len(data))
	}

	return totalSize, len(chains), nil
}

// LoadAll loads all application state
func (s *Store) LoadAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	totalSize := int64(0)
	totalItems := 0

	// Load sessions
	if s.sessionMgr != nil {
		size, items, err := s.loadSessions()
		if err != nil {
			return fmt.Errorf("failed to load sessions: %w", err)
		}
		totalSize += size
		totalItems += items
	}

	// Load conversations
	if s.memoryMgr != nil {
		size, items, err := s.loadConversations()
		if err != nil {
			return fmt.Errorf("failed to load conversations: %w", err)
		}
		totalSize += size
		totalItems += items
	}

	// Load focus chains
	if s.focusMgr != nil {
		size, items, err := s.loadFocusChains()
		if err != nil {
			return fmt.Errorf("failed to load focus chains: %w", err)
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

	for _, callback := range s.onLoad {
		callback(metadata)
	}

	return nil
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

// OnSave registers a save callback
func (s *Store) OnSave(callback SaveCallback) {
	s.onSave = append(s.onSave, callback)
}

// OnLoad registers a load callback
func (s *Store) OnLoad(callback LoadCallback) {
	s.onLoad = append(s.onLoad, callback)
}

// OnError registers an error callback
func (s *Store) OnError(callback ErrorCallback) {
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

// triggerError triggers error callbacks
func (s *Store) triggerError(err error) {
	for _, callback := range s.onError {
		callback(err)
	}
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
