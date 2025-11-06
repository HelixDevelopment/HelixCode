package autonomy

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PermissionManager handles permission checks
type PermissionManager struct {
	mu           sync.RWMutex
	capabilities *ModeCapabilities
	guardrails   *GuardrailsChecker
	confirmQueue *ConfirmQueue
	cache        *permissionCache
}

// ConfirmQueue manages pending confirmations
type ConfirmQueue struct {
	mu      sync.RWMutex
	pending map[string]*pendingConfirm
}

type pendingConfirm struct {
	Action    *Action
	CreatedAt time.Time
	ExpiresAt time.Time
	Response  chan bool
}

// permissionCache caches permission decisions
type permissionCache struct {
	mu      sync.RWMutex
	entries map[string]*cachedPermission
}

type cachedPermission struct {
	Permission *Permission
	ExpiresAt  time.Time
}

// NewPermissionManager creates a new permission manager
func NewPermissionManager(mode AutonomyMode, guardrails *GuardrailsChecker) *PermissionManager {
	return &PermissionManager{
		capabilities: GetCapabilities(mode),
		guardrails:   guardrails,
		confirmQueue: &ConfirmQueue{
			pending: make(map[string]*pendingConfirm),
		},
		cache: &permissionCache{
			entries: make(map[string]*cachedPermission),
		},
	}
}

// Check determines if an action is permitted
func (p *PermissionManager) Check(ctx context.Context, action *Action) (*Permission, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Check cache first
	if cached := p.cache.get(action); cached != nil {
		return cached, nil
	}

	// Check mode capabilities
	perm := p.checkCapabilities(action)
	if !perm.Granted {
		return perm, nil
	}

	// Check guardrails if enabled
	if p.guardrails != nil {
		passed, reasons, err := p.guardrails.Check(ctx, action)
		if err != nil {
			return &Permission{
				Granted: false,
				Reason:  fmt.Sprintf("guardrail check error: %v", err),
			}, err
		}

		if !passed {
			return &Permission{
				Granted: false,
				Reason:  fmt.Sprintf("guardrail violation: %s", reasons[0]),
			}, nil
		}
	}

	// Cache the permission
	p.cache.set(action, perm)

	return perm, nil
}

// checkCapabilities checks if mode capabilities allow the action
func (p *PermissionManager) checkCapabilities(action *Action) *Permission {
	caps := p.capabilities

	switch action.Type {
	case ActionLoadContext:
		if !caps.AutoContext {
			return &Permission{
				Granted:         true,
				RequiresConfirm: true,
				ConfirmPrompt:   "Load context for this task?",
				Reason:          "Manual context loading required",
			}
		}
		return &Permission{
			Granted: true,
			Reason:  "Auto-context enabled",
		}

	case ActionApplyChange:
		if !caps.AutoApply {
			return &Permission{
				Granted:         true,
				RequiresConfirm: true,
				ConfirmPrompt:   fmt.Sprintf("Apply changes to %d file(s)?", len(action.Context.FilesAffected)),
				Reason:          "Manual approval required",
			}
		}
		if caps.RequireConfirm && action.IsRisky() {
			return &Permission{
				Granted:         true,
				RequiresConfirm: true,
				ConfirmPrompt:   fmt.Sprintf("Apply risky changes (%s)?", action.Risk),
				Reason:          "Risky operation confirmation",
			}
		}
		return &Permission{
			Granted: true,
			Reason:  "Auto-apply enabled",
		}

	case ActionExecuteCmd:
		if !caps.AutoExecute {
			return &Permission{
				Granted:         true,
				RequiresConfirm: true,
				ConfirmPrompt:   fmt.Sprintf("Execute command: %s?", action.Context.CommandToRun),
				Reason:          "Manual execution required",
			}
		}
		if caps.RequireConfirm && action.IsRisky() {
			return &Permission{
				Granted:         true,
				RequiresConfirm: true,
				ConfirmPrompt:   fmt.Sprintf("Execute risky command (%s)?", action.Risk),
				Reason:          "Risky command confirmation",
			}
		}
		return &Permission{
			Granted: true,
			Reason:  "Auto-execute enabled",
		}

	case ActionDebugRetry:
		if !caps.AutoDebug {
			return &Permission{
				Granted: false,
				Reason:  "Auto-debug not enabled in this mode",
			}
		}
		return &Permission{
			Granted: true,
			Reason:  "Auto-debug enabled",
		}

	case ActionIteration:
		if caps.IterationLimit == 0 {
			return &Permission{
				Granted: false,
				Reason:  "Iterations not allowed in this mode",
			}
		}
		// Iteration count is 0-indexed, so iteration 0 is the first iteration
		// IterationLimit -1 means unlimited
		if caps.IterationLimit > 0 && action.Context.IterationCount >= caps.IterationLimit {
			return &Permission{
				Granted: false,
				Reason:  fmt.Sprintf("Iteration limit reached (%d)", caps.IterationLimit),
			}
		}
		return &Permission{
			Granted: true,
			Reason:  fmt.Sprintf("Iteration %d", action.Context.IterationCount+1),
		}

	case ActionFileDelete, ActionSystemChange:
		if !caps.AllowRisky {
			return &Permission{
				Granted: false,
				Reason:  "Risky operations not allowed in this mode",
			}
		}
		return &Permission{
			Granted:         true,
			RequiresConfirm: true,
			ConfirmPrompt:   fmt.Sprintf("Perform %s operation?", action.Type),
			Reason:          "Risky operation requires confirmation",
		}

	case ActionBulkEdit:
		if caps.RequireConfirm {
			return &Permission{
				Granted:         true,
				RequiresConfirm: true,
				ConfirmPrompt:   fmt.Sprintf("Edit %d files?", len(action.Context.FilesAffected)),
				Reason:          "Bulk operation confirmation",
			}
		}
		return &Permission{
			Granted: true,
			Reason:  "Bulk edit allowed",
		}

	default:
		// Default to requiring confirmation for unknown actions
		return &Permission{
			Granted:         true,
			RequiresConfirm: true,
			ConfirmPrompt:   fmt.Sprintf("Perform action: %s?", action.Type),
			Reason:          "Unknown action type",
		}
	}
}

// RequestConfirmation asks user to confirm an action
func (p *PermissionManager) RequestConfirmation(ctx context.Context, action *Action) (bool, error) {
	// This would integrate with the UI/CLI to get user confirmation
	// For now, we'll return a mock response
	// In production, this would block until user responds

	// Create pending confirmation
	pc := &pendingConfirm{
		Action:    action,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
		Response:  make(chan bool, 1),
	}

	p.confirmQueue.mu.Lock()
	confirmID := fmt.Sprintf("%s_%d", action.Type, time.Now().UnixNano())
	p.confirmQueue.pending[confirmID] = pc
	p.confirmQueue.mu.Unlock()

	// Wait for response or timeout
	select {
	case confirmed := <-pc.Response:
		p.confirmQueue.mu.Lock()
		delete(p.confirmQueue.pending, confirmID)
		p.confirmQueue.mu.Unlock()
		return confirmed, nil

	case <-time.After(5 * time.Minute):
		p.confirmQueue.mu.Lock()
		delete(p.confirmQueue.pending, confirmID)
		p.confirmQueue.mu.Unlock()
		return false, ErrConfirmationFailed

	case <-ctx.Done():
		p.confirmQueue.mu.Lock()
		delete(p.confirmQueue.pending, confirmID)
		p.confirmQueue.mu.Unlock()
		return false, ctx.Err()
	}
}

// GrantPermission explicitly grants permission for an action type
func (p *PermissionManager) GrantPermission(ctx context.Context, action *Action, duration time.Duration) error {
	perm := &Permission{
		Granted:   true,
		Reason:    "Explicitly granted",
		ExpiresAt: time.Now().Add(duration),
	}

	p.cache.setWithExpiry(action, perm, duration)
	return nil
}

// RevokePermission revokes a granted permission
func (p *PermissionManager) RevokePermission(ctx context.Context, actionType ActionType) error {
	p.cache.mu.Lock()
	defer p.cache.mu.Unlock()

	// Remove all cached permissions for this action type
	for key := range p.cache.entries {
		if key[:len(actionType)] == string(actionType) {
			delete(p.cache.entries, key)
		}
	}

	return nil
}

// UpdateCapabilities updates capabilities for mode change
func (p *PermissionManager) UpdateCapabilities(capabilities *ModeCapabilities) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.capabilities = capabilities

	// Clear cache when capabilities change
	p.cache.clear()
}

// permissionCache methods

func (pc *permissionCache) get(action *Action) *Permission {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	key := pc.cacheKey(action)
	cached, exists := pc.entries[key]
	if !exists {
		return nil
	}

	// Check expiry
	if time.Now().After(cached.ExpiresAt) {
		delete(pc.entries, key)
		return nil
	}

	return cached.Permission
}

func (pc *permissionCache) set(action *Action, perm *Permission) {
	pc.setWithExpiry(action, perm, 5*time.Minute)
}

func (pc *permissionCache) setWithExpiry(action *Action, perm *Permission, duration time.Duration) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	key := pc.cacheKey(action)
	pc.entries[key] = &cachedPermission{
		Permission: perm,
		ExpiresAt:  time.Now().Add(duration),
	}
}

func (pc *permissionCache) clear() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.entries = make(map[string]*cachedPermission)
}

func (pc *permissionCache) cacheKey(action *Action) string {
	// Simple cache key based on action type and risk
	return fmt.Sprintf("%s_%s", action.Type, action.Risk)
}

// GetPendingConfirmations returns all pending confirmations
func (p *PermissionManager) GetPendingConfirmations() []*Action {
	p.confirmQueue.mu.RLock()
	defer p.confirmQueue.mu.RUnlock()

	pending := make([]*Action, 0, len(p.confirmQueue.pending))
	for _, pc := range p.confirmQueue.pending {
		if time.Now().Before(pc.ExpiresAt) {
			pending = append(pending, pc.Action)
		}
	}

	return pending
}
