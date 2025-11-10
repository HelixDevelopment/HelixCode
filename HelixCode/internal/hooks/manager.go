package hooks

import (
	"context"
	"fmt"
	"sync"
)

// Manager manages hooks registration and execution
type Manager struct {
	hooks     map[HookType][]*Hook // Hooks organized by type
	hooksAll  map[string]*Hook     // All hooks by ID
	executor  *Executor            // Hook executor
	mu        sync.RWMutex         // Thread-safety
	onCreate  []HookCallback       // Callbacks on hook creation
	onRemove  []HookCallback       // Callbacks on hook removal
	onExecute []ExecuteCallback    // Callbacks on execution
}

// HookCallback is called for hook lifecycle events
type HookCallback func(*Hook)

// ExecuteCallback is called when hooks are executed
type ExecuteCallback func(*Event, []*ExecutionResult)

// NewManager creates a new hook manager
func NewManager() *Manager {
	return &Manager{
		hooks:     make(map[HookType][]*Hook),
		hooksAll:  make(map[string]*Hook),
		executor:  NewExecutor(),
		onCreate:  make([]HookCallback, 0),
		onRemove:  make([]HookCallback, 0),
		onExecute: make([]ExecuteCallback, 0),
	}
}

// NewManagerWithExecutor creates a new manager with custom executor
func NewManagerWithExecutor(executor *Executor) *Manager {
	m := NewManager()
	m.executor = executor
	return m
}

// Register registers a new hook
func (m *Manager) Register(hook *Hook) error {
	if err := hook.Validate(); err != nil {
		return fmt.Errorf("invalid hook: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for duplicate ID
	if _, exists := m.hooksAll[hook.ID]; exists {
		return fmt.Errorf("hook with ID '%s' already registered", hook.ID)
	}

	// Add to type-specific list
	if m.hooks[hook.Type] == nil {
		m.hooks[hook.Type] = make([]*Hook, 0)
	}
	m.hooks[hook.Type] = append(m.hooks[hook.Type], hook)

	// Add to all hooks map
	m.hooksAll[hook.ID] = hook

	// Trigger callbacks
	for _, callback := range m.onCreate {
		callback(hook)
	}

	return nil
}

// RegisterMany registers multiple hooks
func (m *Manager) RegisterMany(hooks []*Hook) error {
	for _, hook := range hooks {
		if err := m.Register(hook); err != nil {
			return err
		}
	}
	return nil
}

// Unregister removes a hook by ID
func (m *Manager) Unregister(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	hook, exists := m.hooksAll[id]
	if !exists {
		return fmt.Errorf("hook not found: %s", id)
	}

	// Remove from type-specific list
	typeHooks := m.hooks[hook.Type]
	for i, h := range typeHooks {
		if h.ID == id {
			m.hooks[hook.Type] = append(typeHooks[:i], typeHooks[i+1:]...)
			break
		}
	}

	// Remove from all hooks map
	delete(m.hooksAll, id)

	// Trigger callbacks
	for _, callback := range m.onRemove {
		callback(hook)
	}

	return nil
}

// Get returns a hook by ID
func (m *Manager) Get(id string) (*Hook, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	hook, exists := m.hooksAll[id]
	if !exists {
		return nil, fmt.Errorf("hook not found: %s", id)
	}

	return hook, nil
}

// GetByType returns all hooks for a specific type
func (m *Manager) GetByType(hookType HookType) []*Hook {
	m.mu.RLock()
	defer m.mu.RUnlock()

	hooks := m.hooks[hookType]
	if hooks == nil {
		return []*Hook{}
	}

	// Return copy
	result := make([]*Hook, len(hooks))
	copy(result, hooks)
	return result
}

// GetByTag returns all hooks with a specific tag
func (m *Manager) GetByTag(tag string) []*Hook {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Hook, 0)
	for _, hook := range m.hooksAll {
		if hook.HasTag(tag) {
			result = append(result, hook)
		}
	}
	return result
}

// GetAll returns all registered hooks
func (m *Manager) GetAll() []*Hook {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Hook, 0, len(m.hooksAll))
	for _, hook := range m.hooksAll {
		result = append(result, hook)
	}
	return result
}

// GetEnabled returns all enabled hooks
func (m *Manager) GetEnabled() []*Hook {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Hook, 0)
	for _, hook := range m.hooksAll {
		if hook.Enabled {
			result = append(result, hook)
		}
	}
	return result
}

// Enable enables a hook by ID
func (m *Manager) Enable(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	hook, exists := m.hooksAll[id]
	if !exists {
		return fmt.Errorf("hook not found: %s", id)
	}

	hook.Enabled = true
	return nil
}

// Disable disables a hook by ID
func (m *Manager) Disable(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	hook, exists := m.hooksAll[id]
	if !exists {
		return fmt.Errorf("hook not found: %s", id)
	}

	hook.Enabled = false
	return nil
}

// EnableAll enables all hooks
func (m *Manager) EnableAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, hook := range m.hooksAll {
		hook.Enabled = true
	}
}

// DisableAll disables all hooks
func (m *Manager) DisableAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, hook := range m.hooksAll {
		hook.Enabled = false
	}
}

// Count returns the total number of registered hooks
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.hooksAll)
}

// CountByType returns the number of hooks for a specific type
func (m *Manager) CountByType(hookType HookType) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.hooks[hookType])
}

// Clear removes all hooks
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.hooks = make(map[HookType][]*Hook)
	m.hooksAll = make(map[string]*Hook)
}

// Trigger triggers hooks for an event
func (m *Manager) Trigger(ctx context.Context, eventType HookType) []*ExecutionResult {
	event := NewEventWithContext(ctx, eventType)
	return m.TriggerEvent(event)
}

// TriggerEvent triggers hooks for an event
func (m *Manager) TriggerEvent(event *Event) []*ExecutionResult {
	m.mu.RLock()
	hooks := m.GetByType(event.Type)
	m.mu.RUnlock()

	// Execute hooks
	results := m.executor.ExecuteAll(event.Context, hooks, event)

	// Trigger execute callbacks
	for _, callback := range m.onExecute {
		callback(event, results)
	}

	return results
}

// TriggerAndWait triggers hooks and waits for all to complete
func (m *Manager) TriggerAndWait(ctx context.Context, eventType HookType) []*ExecutionResult {
	event := NewEventWithContext(ctx, eventType)
	return m.TriggerEventAndWait(event)
}

// TriggerEventAndWait triggers hooks for an event and waits for completion
func (m *Manager) TriggerEventAndWait(event *Event) []*ExecutionResult {
	m.mu.RLock()
	hooks := m.GetByType(event.Type)
	m.mu.RUnlock()

	// Execute hooks and wait
	results := m.executor.ExecuteAndWait(event.Context, hooks, event)

	// Trigger execute callbacks
	for _, callback := range m.onExecute {
		callback(event, results)
	}

	return results
}

// TriggerSync triggers hooks synchronously
func (m *Manager) TriggerSync(ctx context.Context, eventType HookType) []*ExecutionResult {
	event := NewEventWithContext(ctx, eventType)
	return m.TriggerEventSync(event)
}

// TriggerEventSync triggers hooks for an event synchronously
func (m *Manager) TriggerEventSync(event *Event) []*ExecutionResult {
	m.mu.RLock()
	hooks := m.GetByType(event.Type)
	m.mu.RUnlock()

	// Execute hooks synchronously
	results := m.executor.ExecuteSync(event.Context, hooks, event)

	// Trigger execute callbacks
	for _, callback := range m.onExecute {
		callback(event, results)
	}

	return results
}

// Wait waits for all async hook executions to complete
func (m *Manager) Wait() {
	m.executor.Wait()
}

// GetExecutor returns the executor
func (m *Manager) GetExecutor() *Executor {
	return m.executor
}

// GetStatistics returns execution statistics
func (m *Manager) GetStatistics() *ManagerStatistics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &ManagerStatistics{
		TotalHooks:    len(m.hooksAll),
		EnabledHooks:  0,
		DisabledHooks: 0,
		ByType:        make(map[HookType]int),
		ExecutorStats: m.executor.GetStatistics(),
	}

	for _, hook := range m.hooksAll {
		if hook.Enabled {
			stats.EnabledHooks++
		} else {
			stats.DisabledHooks++
		}
		stats.ByType[hook.Type]++
	}

	return stats
}

// OnCreate registers a callback for hook creation
func (m *Manager) OnCreate(callback HookCallback) {
	m.onCreate = append(m.onCreate, callback)
}

// OnRemove registers a callback for hook removal
func (m *Manager) OnRemove(callback HookCallback) {
	m.onRemove = append(m.onRemove, callback)
}

// OnExecute registers a callback for hook execution
func (m *Manager) OnExecute(callback ExecuteCallback) {
	m.onExecute = append(m.onExecute, callback)
}

// FindByName finds hooks by name (case-insensitive substring match)
func (m *Manager) FindByName(nameSubstring string) []*Hook {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Hook, 0)
	lowerSearch := toLower(nameSubstring)

	for _, hook := range m.hooksAll {
		if contains(toLower(hook.Name), lowerSearch) {
			result = append(result, hook)
		}
	}

	return result
}

// UpdatePriority updates the priority of a hook
func (m *Manager) UpdatePriority(id string, priority HookPriority) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	hook, exists := m.hooksAll[id]
	if !exists {
		return fmt.Errorf("hook not found: %s", id)
	}

	hook.Priority = priority
	return nil
}

// Clone creates a copy of a hook
func (m *Manager) Clone(id string) (*Hook, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	hook, exists := m.hooksAll[id]
	if !exists {
		return nil, fmt.Errorf("hook not found: %s", id)
	}

	return hook.Clone(), nil
}

// Export exports all hooks (without handlers)
func (m *Manager) Export() []*HookMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*HookMetadata, 0, len(m.hooksAll))
	for _, hook := range m.hooksAll {
		result = append(result, ExportHookMetadata(hook))
	}

	return result
}

// ManagerStatistics contains manager statistics
type ManagerStatistics struct {
	TotalHooks    int                 // Total registered hooks
	EnabledHooks  int                 // Number of enabled hooks
	DisabledHooks int                 // Number of disabled hooks
	ByType        map[HookType]int    // Count by hook type
	ExecutorStats *ExecutorStatistics // Executor statistics
}

// String returns a string representation of the statistics
func (s *ManagerStatistics) String() string {
	return fmt.Sprintf("Hooks: %d total (%d enabled, %d disabled), Executor: %s",
		s.TotalHooks, s.EnabledHooks, s.DisabledHooks, s.ExecutorStats.String())
}

// HookMetadata represents hook metadata (without handler function)
type HookMetadata struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        HookType          `json:"type"`
	Description string            `json:"description"`
	Priority    HookPriority      `json:"priority"`
	Async       bool              `json:"async"`
	Enabled     bool              `json:"enabled"`
	Tags        []string          `json:"tags"`
	Metadata    map[string]string `json:"metadata"`
}

// ExportHookMetadata exports hook metadata
func ExportHookMetadata(hook *Hook) *HookMetadata {
	return &HookMetadata{
		ID:          hook.ID,
		Name:        hook.Name,
		Type:        hook.Type,
		Description: hook.Description,
		Priority:    hook.Priority,
		Async:       hook.Async,
		Enabled:     hook.Enabled,
		Tags:        hook.Tags,
		Metadata:    hook.Metadata,
	}
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
