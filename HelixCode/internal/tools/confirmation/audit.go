package confirmation

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// AuditStorage interface for audit storage
type AuditStorage interface {
	Store(ctx context.Context, entry AuditEntry) error
	Query(ctx context.Context, query AuditQuery) ([]AuditEntry, error)
	Clear(ctx context.Context) error
}

// AuditLogger logs all confirmation decisions
type AuditLogger struct {
	logger  *slog.Logger
	storage AuditStorage
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(path string) *AuditLogger {
	return &AuditLogger{
		logger:  slog.Default(),
		storage: NewFileAuditStorage(path),
	}
}

// NewAuditLoggerWithStorage creates a new audit logger with custom storage
func NewAuditLoggerWithStorage(storage AuditStorage) *AuditLogger {
	return &AuditLogger{
		logger:  slog.Default(),
		storage: storage,
	}
}

// Log logs a confirmation decision
func (al *AuditLogger) Log(ctx context.Context, entry AuditEntry) error {
	// Log to structured logger
	al.logger.InfoContext(ctx, "tool confirmation",
		"tool", entry.ToolName,
		"operation", entry.Operation.Type,
		"decision", entry.Decision.String(),
		"user", entry.User,
		"session", entry.SessionID,
	)

	// Store in audit storage
	if err := al.storage.Store(ctx, entry); err != nil {
		return fmt.Errorf("store audit entry: %w", err)
	}

	return nil
}

// Query queries audit log
func (al *AuditLogger) Query(ctx context.Context, query AuditQuery) ([]AuditEntry, error) {
	entries, err := al.storage.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query audit entries: %w", err)
	}
	return entries, nil
}

// Clear clears all audit entries
func (al *AuditLogger) Clear(ctx context.Context) error {
	if err := al.storage.Clear(ctx); err != nil {
		return fmt.Errorf("clear audit entries: %w", err)
	}
	return nil
}

// FileAuditStorage stores audit logs in files
type FileAuditStorage struct {
	path string
	mu   sync.Mutex
}

// NewFileAuditStorage creates a new file audit storage
func NewFileAuditStorage(path string) *FileAuditStorage {
	return &FileAuditStorage{
		path: path,
	}
}

// Store implements AuditStorage
func (fas *FileAuditStorage) Store(ctx context.Context, entry AuditEntry) error {
	fas.mu.Lock()
	defer fas.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(fas.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create audit directory: %w", err)
	}

	f, err := os.OpenFile(fas.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open audit file: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal audit entry: %w", err)
	}

	_, err = f.Write(append(data, '\n'))
	if err != nil {
		return fmt.Errorf("write audit entry: %w", err)
	}

	return nil
}

// Query implements AuditStorage
func (fas *FileAuditStorage) Query(ctx context.Context, query AuditQuery) ([]AuditEntry, error) {
	fas.mu.Lock()
	defer fas.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(fas.path); os.IsNotExist(err) {
		return []AuditEntry{}, nil
	}

	f, err := os.Open(fas.path)
	if err != nil {
		return nil, fmt.Errorf("open audit file: %w", err)
	}
	defer f.Close()

	var entries []AuditEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry AuditEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // Skip invalid entries
		}

		if matchesQuery(entry, query) {
			entries = append(entries, entry)
			if query.Limit > 0 && len(entries) >= query.Limit {
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan audit file: %w", err)
	}

	return entries, nil
}

// Clear implements AuditStorage
func (fas *FileAuditStorage) Clear(ctx context.Context) error {
	fas.mu.Lock()
	defer fas.mu.Unlock()

	// Remove the file
	if err := os.Remove(fas.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove audit file: %w", err)
	}

	return nil
}

// matchesQuery checks if an entry matches a query
func matchesQuery(entry AuditEntry, query AuditQuery) bool {
	if query.User != "" && entry.User != query.User {
		return false
	}
	if query.Tool != "" && entry.ToolName != query.Tool {
		return false
	}
	if !query.StartTime.IsZero() && entry.Timestamp.Before(query.StartTime) {
		return false
	}
	if !query.EndTime.IsZero() && entry.Timestamp.After(query.EndTime) {
		return false
	}
	if query.Decision != nil && entry.Decision != *query.Decision {
		return false
	}
	return true
}

// MemoryAuditStorage stores audit logs in memory (for testing)
type MemoryAuditStorage struct {
	mu      sync.RWMutex
	entries []AuditEntry
}

// NewMemoryAuditStorage creates a new memory audit storage
func NewMemoryAuditStorage() *MemoryAuditStorage {
	return &MemoryAuditStorage{
		entries: make([]AuditEntry, 0),
	}
}

// Store implements AuditStorage
func (mas *MemoryAuditStorage) Store(ctx context.Context, entry AuditEntry) error {
	mas.mu.Lock()
	defer mas.mu.Unlock()

	mas.entries = append(mas.entries, entry)
	return nil
}

// Query implements AuditStorage
func (mas *MemoryAuditStorage) Query(ctx context.Context, query AuditQuery) ([]AuditEntry, error) {
	mas.mu.RLock()
	defer mas.mu.RUnlock()

	var results []AuditEntry
	for _, entry := range mas.entries {
		if matchesQuery(entry, query) {
			results = append(results, entry)
			if query.Limit > 0 && len(results) >= query.Limit {
				break
			}
		}
	}

	return results, nil
}

// Clear implements AuditStorage
func (mas *MemoryAuditStorage) Clear(ctx context.Context) error {
	mas.mu.Lock()
	defer mas.mu.Unlock()

	mas.entries = make([]AuditEntry, 0)
	return nil
}

// GetAll returns all entries (for testing)
func (mas *MemoryAuditStorage) GetAll() []AuditEntry {
	mas.mu.RLock()
	defer mas.mu.RUnlock()

	result := make([]AuditEntry, len(mas.entries))
	copy(result, mas.entries)
	return result
}
