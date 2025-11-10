package multiedit

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TransactionManager handles transaction lifecycle
type TransactionManager struct {
	mu           sync.RWMutex
	transactions map[string]*EditTransaction
	maxDuration  time.Duration
}

// EditTransaction represents a multi-file edit operation
type EditTransaction struct {
	ID        string
	State     TransactionState
	Files     []*FileEdit
	CreatedAt time.Time
	UpdatedAt time.Time
	Options   EditOptions
	Metadata  map[string]interface{}

	mu          sync.RWMutex
	backupPaths map[string]string // file path -> backup path
}

// TransactionState represents the current state
type TransactionState int

const (
	StatePending TransactionState = iota
	StatePreview
	StateReady
	StateCommitting
	StateCommitted
	StateRollingBack
	StateRolledBack
	StateAborted
	StateFailed
)

// String returns the string representation of the state
func (s TransactionState) String() string {
	switch s {
	case StatePending:
		return "pending"
	case StatePreview:
		return "preview"
	case StateReady:
		return "ready"
	case StateCommitting:
		return "committing"
	case StateCommitted:
		return "committed"
	case StateRollingBack:
		return "rolling_back"
	case StateRolledBack:
		return "rolled_back"
	case StateAborted:
		return "aborted"
	case StateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// FileEdit represents a single file edit operation
type FileEdit struct {
	FilePath   string
	Operation  EditOperation
	OldContent []byte
	NewContent []byte
	Checksum   string // SHA256 of original content
	Applied    bool
	Error      error
}

// EditOperation type
type EditOperation int

const (
	OpCreate EditOperation = iota
	OpUpdate
	OpDelete
	OpRename
)

// String returns the string representation of the operation
func (o EditOperation) String() string {
	switch o {
	case OpCreate:
		return "create"
	case OpUpdate:
		return "update"
	case OpDelete:
		return "delete"
	case OpRename:
		return "rename"
	default:
		return "unknown"
	}
}

// EditOptions configures edit behavior
type EditOptions struct {
	DryRun         bool
	ConflictPolicy ConflictPolicy
	BackupEnabled  bool
	GitAware       bool
	PreCommitHooks bool
	MaxFileSize    int64
	AllowedPaths   []string
	DeniedPaths    []string
}

// ConflictPolicy defines how to handle conflicts
type ConflictPolicy int

const (
	ConflictPolicyAbort ConflictPolicy = iota
	ConflictPolicySkip
	ConflictPolicyOverwrite
	ConflictPolicyAsk
)

// String returns the string representation of the policy
func (p ConflictPolicy) String() string {
	switch p {
	case ConflictPolicyAbort:
		return "abort"
	case ConflictPolicySkip:
		return "skip"
	case ConflictPolicyOverwrite:
		return "overwrite"
	case ConflictPolicyAsk:
		return "ask"
	default:
		return "unknown"
	}
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(maxDuration time.Duration) *TransactionManager {
	return &TransactionManager{
		transactions: make(map[string]*EditTransaction),
		maxDuration:  maxDuration,
	}
}

// Begin starts a new transaction
func (tm *TransactionManager) Begin(ctx context.Context, opts EditOptions) (*EditTransaction, error) {
	tx := &EditTransaction{
		ID:          uuid.New().String(),
		State:       StatePending,
		Files:       make([]*FileEdit, 0),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Options:     opts,
		Metadata:    make(map[string]interface{}),
		backupPaths: make(map[string]string),
	}

	tm.mu.Lock()
	tm.transactions[tx.ID] = tx
	tm.mu.Unlock()

	// Start timeout monitor
	if tm.maxDuration > 0 {
		go tm.monitorTimeout(tx)
	}

	return tx, nil
}

// Get retrieves a transaction by ID
func (tm *TransactionManager) Get(txID string) (*EditTransaction, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tx, ok := tm.transactions[txID]
	if !ok {
		return nil, ErrTransactionNotFound
	}

	return tx, nil
}

// AddEdit adds a file edit to the transaction
func (tm *TransactionManager) AddEdit(tx *EditTransaction, edit *FileEdit) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.State != StatePending && tx.State != StatePreview && tx.State != StateReady {
		return ErrInvalidState
	}

	tx.Files = append(tx.Files, edit)
	tx.UpdatedAt = time.Now()

	return nil
}

// UpdateState updates the transaction state
func (tm *TransactionManager) UpdateState(tx *EditTransaction, newState TransactionState) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	// Validate state transition
	if !isValidStateTransition(tx.State, newState) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidState, tx.State, newState)
	}

	tx.State = newState
	tx.UpdatedAt = time.Now()

	return nil
}

// Delete removes a transaction
func (tm *TransactionManager) Delete(txID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	delete(tm.transactions, txID)
	return nil
}

// List returns all transactions
func (tm *TransactionManager) List() []*EditTransaction {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	transactions := make([]*EditTransaction, 0, len(tm.transactions))
	for _, tx := range tm.transactions {
		transactions = append(transactions, tx)
	}

	return transactions
}

// Cleanup removes old transactions
func (tm *TransactionManager) Cleanup(maxAge time.Duration) int {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	count := 0

	for id, tx := range tm.transactions {
		if tx.UpdatedAt.Before(cutoff) && (tx.State == StateCommitted || tx.State == StateRolledBack || tx.State == StateAborted) {
			delete(tm.transactions, id)
			count++
		}
	}

	return count
}

// monitorTimeout monitors transaction timeout
func (tm *TransactionManager) monitorTimeout(tx *EditTransaction) {
	timer := time.NewTimer(tm.maxDuration)
	defer timer.Stop()

	<-timer.C

	tx.mu.Lock()
	defer tx.mu.Unlock()

	// Only abort if still in progress
	if tx.State == StatePending || tx.State == StatePreview || tx.State == StateReady || tx.State == StateCommitting {
		tx.State = StateAborted
		tx.UpdatedAt = time.Now()
	}
}

// isValidStateTransition checks if a state transition is valid
func isValidStateTransition(from, to TransactionState) bool {
	// Define valid transitions
	validTransitions := map[TransactionState][]TransactionState{
		StatePending:     {StatePreview, StateAborted},
		StatePreview:     {StateReady, StateAborted, StatePending},
		StateReady:       {StateCommitting, StateAborted, StatePreview},
		StateCommitting:  {StateCommitted, StateFailed, StateRollingBack},
		StateRollingBack: {StateRolledBack, StateFailed},
		// Terminal states can't transition
		StateCommitted:  {},
		StateRolledBack: {},
		StateAborted:    {},
		StateFailed:     {},
	}

	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}

	for _, state := range allowed {
		if state == to {
			return true
		}
	}

	return false
}

// calculateChecksum calculates SHA256 checksum of content
func calculateChecksum(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash)
}

// ConflictResolver detects and resolves conflicts
type ConflictResolver struct {
	gitIntegration *GitIntegration
	detectGit      bool
}

// NewConflictResolver creates a new conflict resolver
func NewConflictResolver(detectGit bool) *ConflictResolver {
	return &ConflictResolver{
		detectGit: detectGit,
	}
}

// DetectConflicts checks for conflicts before applying edits
func (cr *ConflictResolver) DetectConflicts(ctx context.Context, tx *EditTransaction) ([]*Conflict, error) {
	var conflicts []*Conflict

	for _, edit := range tx.Files {
		conflict, err := cr.detectFileConflict(ctx, edit)
		if err != nil {
			return nil, err
		}
		if conflict != nil {
			conflicts = append(conflicts, conflict)
		}
	}

	return conflicts, nil
}

// detectFileConflict detects conflicts for a single file
func (cr *ConflictResolver) detectFileConflict(ctx context.Context, edit *FileEdit) (*Conflict, error) {
	// Skip for create operations
	if edit.Operation == OpCreate {
		return nil, nil
	}

	// Read current file content
	// This would use the filesystem tools in a real implementation
	// For now, we'll just check if checksum is provided and valid
	if edit.Checksum == "" {
		return nil, nil // No checksum to verify
	}

	// In a real implementation, we'd read the file and compare checksums
	// For now, we'll assume no conflicts

	return nil, nil
}

// Resolve attempts to resolve conflicts
func (cr *ConflictResolver) Resolve(ctx context.Context, conflict *Conflict, strategy ConflictStrategy) error {
	if conflict.Resolution == nil {
		conflict.Resolution = &ConflictResolution{
			Strategy:   strategy,
			ResolvedBy: "system",
			Timestamp:  time.Now(),
		}
	}

	switch strategy {
	case StrategyAbort:
		return ErrConflictDetected
	case StrategyTheirs:
		conflict.Resolution.Resolution = "use_current"
		return nil
	case StrategyOurs:
		conflict.Resolution.Resolution = "use_new"
		return nil
	case StrategyManual:
		return fmt.Errorf("manual resolution required")
	default:
		return fmt.Errorf("unknown strategy: %v", strategy)
	}
}

// Conflict represents a detected conflict
type Conflict struct {
	Type        ConflictType
	FilePath    string
	Expected    string // Expected checksum
	Actual      string // Actual checksum
	Description string
	Resolution  *ConflictResolution
}

// ConflictType categorizes conflicts
type ConflictType int

const (
	ConflictModified ConflictType = iota
	ConflictDeleted
	ConflictMoved
	ConflictPermissions
)

// String returns the string representation of the conflict type
func (ct ConflictType) String() string {
	switch ct {
	case ConflictModified:
		return "modified"
	case ConflictDeleted:
		return "deleted"
	case ConflictMoved:
		return "moved"
	case ConflictPermissions:
		return "permissions"
	default:
		return "unknown"
	}
}

// ConflictStrategy defines resolution approach
type ConflictStrategy int

const (
	StrategyAbort ConflictStrategy = iota
	StrategyTheirs
	StrategyOurs
	StrategyManual
)

// String returns the string representation of the strategy
func (cs ConflictStrategy) String() string {
	switch cs {
	case StrategyAbort:
		return "abort"
	case StrategyTheirs:
		return "theirs"
	case StrategyOurs:
		return "ours"
	case StrategyManual:
		return "manual"
	default:
		return "unknown"
	}
}

// ConflictResolution stores resolution result
type ConflictResolution struct {
	Strategy   ConflictStrategy
	ResolvedBy string
	Resolution string
	Timestamp  time.Time
}

// GitIntegration provides git-aware operations (placeholder)
type GitIntegration struct {
	enabled bool
}

// Error types
var (
	ErrTransactionNotFound = errors.New("transaction not found")
	ErrInvalidState        = errors.New("invalid transaction state")
	ErrConflictDetected    = errors.New("conflict detected")
	ErrBackupFailed        = errors.New("backup failed")
	ErrRollbackFailed      = errors.New("rollback failed")
	ErrFileNotFound        = errors.New("file not found")
	ErrChecksumMismatch    = errors.New("checksum mismatch")
	ErrPermissionDenied    = errors.New("permission denied")
	ErrTransactionTimeout  = errors.New("transaction timeout")
	ErrInvalidPath         = errors.New("invalid file path")
)

// ConflictError wraps conflict errors
type ConflictError struct {
	Type     ConflictType
	FilePath string
	Expected string
	Actual   string
	Message  string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("conflict [%s] in %s: %s (expected: %s, actual: %s)",
		e.Type, e.FilePath, e.Message, e.Expected, e.Actual)
}
