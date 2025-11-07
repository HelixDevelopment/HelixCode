package focus

import (
	"fmt"
	"sync"
	"time"
)

// Manager manages multiple focus chains with thread-safe operations
type Manager struct {
	chains     map[string]*Chain // Chain ID -> Chain
	activeID   string            // ID of active chain
	maxChains  int               // Maximum number of chains to keep (0 = unlimited)
	mu         sync.RWMutex      // Thread-safety
	onCreate   []ChainCallback   // Callbacks on chain creation
	onDelete   []ChainCallback   // Callbacks on chain deletion
	onActivate []ChainCallback   // Callbacks on chain activation
}

// ChainCallback is a callback function for chain events
type ChainCallback func(*Chain)

// NewManager creates a new focus chain manager
func NewManager() *Manager {
	return &Manager{
		chains:     make(map[string]*Chain),
		activeID:   "",
		maxChains:  0, // Unlimited by default
		onCreate:   make([]ChainCallback, 0),
		onDelete:   make([]ChainCallback, 0),
		onActivate: make([]ChainCallback, 0),
	}
}

// NewManagerWithLimit creates a new manager with a maximum number of chains
func NewManagerWithLimit(maxChains int) *Manager {
	m := NewManager()
	m.maxChains = maxChains
	return m
}

// CreateChain creates a new chain and optionally sets it as active
func (m *Manager) CreateChain(name string, setActive bool) (*Chain, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if we're at max capacity
	if m.maxChains > 0 && len(m.chains) >= m.maxChains {
		// Remove oldest chain
		if err := m.removeOldest(); err != nil {
			return nil, fmt.Errorf("failed to make room for new chain: %w", err)
		}
	}

	chain := NewChain(name)
	m.chains[chain.ID] = chain

	if setActive {
		m.activeID = chain.ID
	}

	// Trigger callbacks
	for _, callback := range m.onCreate {
		callback(chain)
	}

	return chain, nil
}

// CreateChainWithSize creates a new chain with a maximum size
func (m *Manager) CreateChainWithSize(name string, maxSize int, setActive bool) (*Chain, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.maxChains > 0 && len(m.chains) >= m.maxChains {
		if err := m.removeOldest(); err != nil {
			return nil, fmt.Errorf("failed to make room for new chain: %w", err)
		}
	}

	chain := NewChainWithSize(name, maxSize)
	m.chains[chain.ID] = chain

	if setActive {
		m.activeID = chain.ID
	}

	for _, callback := range m.onCreate {
		callback(chain)
	}

	return chain, nil
}

// GetChain returns a chain by ID
func (m *Manager) GetChain(id string) (*Chain, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chain, ok := m.chains[id]
	if !ok {
		return nil, fmt.Errorf("chain not found: %s", id)
	}

	return chain, nil
}

// GetActiveChain returns the currently active chain
func (m *Manager) GetActiveChain() (*Chain, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.activeID == "" {
		return nil, fmt.Errorf("no active chain")
	}

	chain, ok := m.chains[m.activeID]
	if !ok {
		return nil, fmt.Errorf("active chain not found: %s", m.activeID)
	}

	return chain, nil
}

// SetActiveChain sets the active chain by ID
func (m *Manager) SetActiveChain(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	chain, ok := m.chains[id]
	if !ok {
		return fmt.Errorf("chain not found: %s", id)
	}

	m.activeID = id

	// Trigger callbacks
	for _, callback := range m.onActivate {
		callback(chain)
	}

	return nil
}

// DeleteChain removes a chain by ID
func (m *Manager) DeleteChain(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	chain, ok := m.chains[id]
	if !ok {
		return fmt.Errorf("chain not found: %s", id)
	}

	delete(m.chains, id)

	// If this was the active chain, clear active ID
	if m.activeID == id {
		m.activeID = ""
	}

	// Trigger callbacks
	for _, callback := range m.onDelete {
		callback(chain)
	}

	return nil
}

// GetAllChains returns all chains
func (m *Manager) GetAllChains() []*Chain {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chains := make([]*Chain, 0, len(m.chains))
	for _, chain := range m.chains {
		chains = append(chains, chain)
	}

	return chains
}

// Count returns the number of chains
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.chains)
}

// Clear removes all chains
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.chains = make(map[string]*Chain)
	m.activeID = ""
}

// PushToActive pushes a focus to the active chain
func (m *Manager) PushToActive(focus *Focus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeID == "" {
		return fmt.Errorf("no active chain")
	}

	chain, ok := m.chains[m.activeID]
	if !ok {
		return fmt.Errorf("active chain not found: %s", m.activeID)
	}

	return chain.Push(focus)
}

// GetCurrentFocus returns the current focus from the active chain
func (m *Manager) GetCurrentFocus() (*Focus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.activeID == "" {
		return nil, fmt.Errorf("no active chain")
	}

	chain, ok := m.chains[m.activeID]
	if !ok {
		return nil, fmt.Errorf("active chain not found: %s", m.activeID)
	}

	return chain.Current()
}

// FindChainsByName finds chains with names containing the specified string
func (m *Manager) FindChainsByName(nameSubstring string) []*Chain {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Chain, 0)
	for _, chain := range m.chains {
		if contains(chain.Name, nameSubstring) {
			result = append(result, chain)
		}
	}

	return result
}

// GetRecentChains returns the N most recently updated chains
func (m *Manager) GetRecentChains(n int) []*Chain {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if n <= 0 {
		return []*Chain{}
	}

	chains := make([]*Chain, 0, len(m.chains))
	for _, chain := range m.chains {
		chains = append(chains, chain)
	}

	// Sort by UpdatedAt (descending)
	for i := 0; i < len(chains)-1; i++ {
		for j := i + 1; j < len(chains); j++ {
			if chains[j].UpdatedAt.After(chains[i].UpdatedAt) {
				chains[i], chains[j] = chains[j], chains[i]
			}
		}
	}

	if n > len(chains) {
		n = len(chains)
	}

	return chains[:n]
}

// CleanExpiredFocuses removes expired focuses from all chains
func (m *Manager) CleanExpiredFocuses() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	totalRemoved := 0
	for _, chain := range m.chains {
		removed := chain.CleanExpired()
		totalRemoved += removed
	}

	return totalRemoved
}

// MergeChains merges source chain into target chain
func (m *Manager) MergeChains(targetID, sourceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	target, ok := m.chains[targetID]
	if !ok {
		return fmt.Errorf("target chain not found: %s", targetID)
	}

	source, ok := m.chains[sourceID]
	if !ok {
		return fmt.Errorf("source chain not found: %s", sourceID)
	}

	if err := target.Merge(source); err != nil {
		return fmt.Errorf("failed to merge chains: %w", err)
	}

	// Delete source chain after successful merge
	delete(m.chains, sourceID)

	// If source was active, set target as active
	if m.activeID == sourceID {
		m.activeID = targetID
	}

	return nil
}

// OnCreate registers a callback for chain creation
func (m *Manager) OnCreate(callback ChainCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.onCreate = append(m.onCreate, callback)
}

// OnDelete registers a callback for chain deletion
func (m *Manager) OnDelete(callback ChainCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.onDelete = append(m.onDelete, callback)
}

// OnActivate registers a callback for chain activation
func (m *Manager) OnActivate(callback ChainCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.onActivate = append(m.onActivate, callback)
}

// GetStatistics returns statistics about the manager
func (m *Manager) GetStatistics() *ManagerStatistics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &ManagerStatistics{
		TotalChains:  len(m.chains),
		TotalFocuses: 0,
		HasActive:    m.activeID != "",
	}

	for _, chain := range m.chains {
		stats.TotalFocuses += chain.Size()
	}

	if stats.TotalChains > 0 {
		stats.AverageFocusesPerChain = float64(stats.TotalFocuses) / float64(stats.TotalChains)
	}

	return stats
}

// removeOldest removes the oldest (by CreatedAt) non-active chain
func (m *Manager) removeOldest() error {
	if len(m.chains) == 0 {
		return fmt.Errorf("no chains to remove")
	}

	var oldestID string
	var oldestTime time.Time

	// Find oldest non-active chain
	first := true
	for id, chain := range m.chains {
		if id == m.activeID {
			continue // Don't remove active chain
		}

		if first || chain.CreatedAt.Before(oldestTime) {
			oldestID = id
			oldestTime = chain.CreatedAt
			first = false
		}
	}

	if oldestID == "" {
		return fmt.Errorf("cannot remove active chain")
	}

	delete(m.chains, oldestID)
	return nil
}

// ManagerStatistics contains statistics about the manager
type ManagerStatistics struct {
	TotalChains            int
	TotalFocuses           int
	AverageFocusesPerChain float64
	HasActive              bool
}

// ExportChain exports a chain as a snapshot
func (m *Manager) ExportChain(id string) (*ChainSnapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chain, ok := m.chains[id]
	if !ok {
		return nil, fmt.Errorf("chain not found: %s", id)
	}

	return &ChainSnapshot{
		Chain:      chain.Clone(),
		ExportedAt: time.Now(),
	}, nil
}

// ImportChain imports a chain from a snapshot
func (m *Manager) ImportChain(snapshot *ChainSnapshot, setActive bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.maxChains > 0 && len(m.chains) >= m.maxChains {
		if err := m.removeOldest(); err != nil {
			return fmt.Errorf("failed to make room for imported chain: %w", err)
		}
	}

	chain := snapshot.Chain.Clone()
	m.chains[chain.ID] = chain

	if setActive {
		m.activeID = chain.ID
	}

	return nil
}

// ChainSnapshot represents an exported chain
type ChainSnapshot struct {
	Chain      *Chain
	ExportedAt time.Time
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	// Simple case-insensitive contains
	sLower := toLower(s)
	substrLower := toLower(substr)

	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
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
