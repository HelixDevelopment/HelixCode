package focus

import (
	"fmt"
	"time"
)

// Chain represents a sequence of focuses forming a conversation/work context
type Chain struct {
	ID          string                 // Unique identifier
	Name        string                 // Chain name
	Description string                 // Chain description
	Focuses     []*Focus               // Ordered list of focuses
	CurrentIdx  int                    // Index of current focus
	MaxSize     int                    // Maximum number of focuses to keep (0 = unlimited)
	Context     map[string]interface{} // Shared context across focuses
	CreatedAt   time.Time              // When chain was created
	UpdatedAt   time.Time              // Last update time
	Metadata    map[string]string      // Custom metadata
}

// NewChain creates a new focus chain
func NewChain(name string) *Chain {
	now := time.Now()
	return &Chain{
		ID:         generateChainID(name),
		Name:       name,
		Focuses:    make([]*Focus, 0),
		CurrentIdx: -1,
		MaxSize:    0, // Unlimited by default
		Context:    make(map[string]interface{}),
		CreatedAt:  now,
		UpdatedAt:  now,
		Metadata:   make(map[string]string),
	}
}

// NewChainWithSize creates a new focus chain with a maximum size
func NewChainWithSize(name string, maxSize int) *Chain {
	chain := NewChain(name)
	chain.MaxSize = maxSize
	return chain
}

// Push adds a new focus to the end of the chain
func (c *Chain) Push(focus *Focus) error {
	if err := focus.Validate(); err != nil {
		return fmt.Errorf("invalid focus: %w", err)
	}

	// Remove expired focuses before adding
	c.removeExpired()

	// If max size is set and we're at capacity, remove oldest
	if c.MaxSize > 0 && len(c.Focuses) >= c.MaxSize {
		c.Focuses = c.Focuses[1:]
		if c.CurrentIdx > 0 {
			c.CurrentIdx--
		}
	}

	c.Focuses = append(c.Focuses, focus)
	c.CurrentIdx = len(c.Focuses) - 1
	c.UpdatedAt = time.Now()

	return nil
}

// Pop removes and returns the last focus
func (c *Chain) Pop() (*Focus, error) {
	if len(c.Focuses) == 0 {
		return nil, fmt.Errorf("chain is empty")
	}

	focus := c.Focuses[len(c.Focuses)-1]
	c.Focuses = c.Focuses[:len(c.Focuses)-1]

	// Adjust current index
	if c.CurrentIdx >= len(c.Focuses) {
		c.CurrentIdx = len(c.Focuses) - 1
	}

	c.UpdatedAt = time.Now()
	return focus, nil
}

// Current returns the current focus
func (c *Chain) Current() (*Focus, error) {
	if c.CurrentIdx < 0 || c.CurrentIdx >= len(c.Focuses) {
		return nil, fmt.Errorf("no current focus")
	}
	return c.Focuses[c.CurrentIdx], nil
}

// SetCurrent sets the current focus by index
func (c *Chain) SetCurrent(index int) error {
	if index < 0 || index >= len(c.Focuses) {
		return fmt.Errorf("index out of range: %d", index)
	}
	c.CurrentIdx = index
	c.UpdatedAt = time.Now()
	return nil
}

// Next moves to the next focus in the chain
func (c *Chain) Next() (*Focus, error) {
	if c.CurrentIdx >= len(c.Focuses)-1 {
		return nil, fmt.Errorf("already at last focus")
	}
	c.CurrentIdx++
	c.UpdatedAt = time.Now()
	return c.Focuses[c.CurrentIdx], nil
}

// Previous moves to the previous focus in the chain
func (c *Chain) Previous() (*Focus, error) {
	if c.CurrentIdx <= 0 {
		return nil, fmt.Errorf("already at first focus")
	}
	c.CurrentIdx--
	c.UpdatedAt = time.Now()
	return c.Focuses[c.CurrentIdx], nil
}

// First returns the first focus
func (c *Chain) First() (*Focus, error) {
	if len(c.Focuses) == 0 {
		return nil, fmt.Errorf("chain is empty")
	}
	return c.Focuses[0], nil
}

// Last returns the last focus
func (c *Chain) Last() (*Focus, error) {
	if len(c.Focuses) == 0 {
		return nil, fmt.Errorf("chain is empty")
	}
	return c.Focuses[len(c.Focuses)-1], nil
}

// Get returns the focus at the specified index
func (c *Chain) Get(index int) (*Focus, error) {
	if index < 0 || index >= len(c.Focuses) {
		return nil, fmt.Errorf("index out of range: %d", index)
	}
	return c.Focuses[index], nil
}

// GetByID returns the focus with the specified ID
func (c *Chain) GetByID(id string) (*Focus, error) {
	for _, focus := range c.Focuses {
		if focus.ID == id {
			return focus, nil
		}
	}
	return nil, fmt.Errorf("focus not found: %s", id)
}

// Remove removes a focus by ID
func (c *Chain) Remove(id string) error {
	for i, focus := range c.Focuses {
		if focus.ID == id {
			c.Focuses = append(c.Focuses[:i], c.Focuses[i+1:]...)

			// Adjust current index
			if c.CurrentIdx >= len(c.Focuses) {
				c.CurrentIdx = len(c.Focuses) - 1
			}

			c.UpdatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("focus not found: %s", id)
}

// Clear removes all focuses from the chain
func (c *Chain) Clear() {
	c.Focuses = make([]*Focus, 0)
	c.CurrentIdx = -1
	c.UpdatedAt = time.Now()
}

// Size returns the number of focuses in the chain
func (c *Chain) Size() int {
	return len(c.Focuses)
}

// IsEmpty checks if the chain is empty
func (c *Chain) IsEmpty() bool {
	return len(c.Focuses) == 0
}

// GetRecent returns the N most recent focuses
func (c *Chain) GetRecent(n int) []*Focus {
	if n <= 0 {
		return []*Focus{}
	}

	if n >= len(c.Focuses) {
		// Return all focuses
		result := make([]*Focus, len(c.Focuses))
		copy(result, c.Focuses)
		return result
	}

	// Return last n focuses
	result := make([]*Focus, n)
	copy(result, c.Focuses[len(c.Focuses)-n:])
	return result
}

// GetByType returns all focuses of a specific type
func (c *Chain) GetByType(focusType FocusType) []*Focus {
	result := make([]*Focus, 0)
	for _, focus := range c.Focuses {
		if focus.Type == focusType {
			result = append(result, focus)
		}
	}
	return result
}

// GetByTag returns all focuses with a specific tag
func (c *Chain) GetByTag(tag string) []*Focus {
	result := make([]*Focus, 0)
	for _, focus := range c.Focuses {
		if focus.HasTag(tag) {
			result = append(result, focus)
		}
	}
	return result
}

// GetByPriority returns all focuses with priority >= specified level
func (c *Chain) GetByPriority(minPriority FocusPriority) []*Focus {
	result := make([]*Focus, 0)
	for _, focus := range c.Focuses {
		if focus.Priority >= minPriority {
			result = append(result, focus)
		}
	}
	return result
}

// SetContext sets a shared context value
func (c *Chain) SetContext(key string, value interface{}) {
	c.Context[key] = value
	c.UpdatedAt = time.Now()
}

// GetContext gets a shared context value
func (c *Chain) GetContext(key string) (interface{}, bool) {
	value, ok := c.Context[key]
	return value, ok
}

// SetMetadata sets a metadata value
func (c *Chain) SetMetadata(key, value string) {
	c.Metadata[key] = value
	c.UpdatedAt = time.Now()
}

// GetMetadata gets a metadata value
func (c *Chain) GetMetadata(key string) (string, bool) {
	value, ok := c.Metadata[key]
	return value, ok
}

// removeExpired removes expired focuses from the chain
func (c *Chain) removeExpired() {
	newFocuses := make([]*Focus, 0, len(c.Focuses))
	removedCount := 0

	for i, focus := range c.Focuses {
		if !focus.IsExpired() {
			newFocuses = append(newFocuses, focus)
		} else {
			// Track removed focuses to adjust current index
			if i <= c.CurrentIdx {
				removedCount++
			}
		}
	}

	c.Focuses = newFocuses
	c.CurrentIdx -= removedCount

	if c.CurrentIdx < 0 && len(c.Focuses) > 0 {
		c.CurrentIdx = 0
	} else if c.CurrentIdx >= len(c.Focuses) {
		c.CurrentIdx = len(c.Focuses) - 1
	}
}

// CleanExpired explicitly removes expired focuses
func (c *Chain) CleanExpired() int {
	oldSize := len(c.Focuses)
	c.removeExpired()
	c.UpdatedAt = time.Now()
	return oldSize - len(c.Focuses)
}

// Clone creates a deep copy of the chain
func (c *Chain) Clone() *Chain {
	clone := &Chain{
		ID:          c.ID,
		Name:        c.Name,
		Description: c.Description,
		Focuses:     make([]*Focus, len(c.Focuses)),
		CurrentIdx:  c.CurrentIdx,
		MaxSize:     c.MaxSize,
		Context:     make(map[string]interface{}),
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		Metadata:    make(map[string]string),
	}

	// Clone focuses
	for i, focus := range c.Focuses {
		clone.Focuses[i] = focus.Clone()
	}

	// Copy context
	for k, v := range c.Context {
		clone.Context[k] = v
	}

	// Copy metadata
	for k, v := range c.Metadata {
		clone.Metadata[k] = v
	}

	return clone
}

// Merge merges another chain into this one
func (c *Chain) Merge(other *Chain) error {
	if other == nil {
		return fmt.Errorf("cannot merge nil chain")
	}

	// Add all focuses from other chain
	for _, focus := range other.Focuses {
		if err := c.Push(focus.Clone()); err != nil {
			return fmt.Errorf("failed to merge focus: %w", err)
		}
	}

	// Merge context (other chain values take precedence)
	for k, v := range other.Context {
		c.Context[k] = v
	}

	// Merge metadata (other chain values take precedence)
	for k, v := range other.Metadata {
		c.Metadata[k] = v
	}

	c.UpdatedAt = time.Now()
	return nil
}

// Split splits the chain at the specified index, returning a new chain with focuses from index onwards
func (c *Chain) Split(index int) (*Chain, error) {
	if index < 0 || index >= len(c.Focuses) {
		return nil, fmt.Errorf("index out of range: %d", index)
	}

	// Create new chain with focuses from index onwards
	newChain := NewChain(fmt.Sprintf("%s-split", c.Name))
	newChain.Focuses = make([]*Focus, len(c.Focuses)-index)
	copy(newChain.Focuses, c.Focuses[index:])

	// Copy context and metadata
	for k, v := range c.Context {
		newChain.Context[k] = v
	}
	for k, v := range c.Metadata {
		newChain.Metadata[k] = v
	}

	// Remove focuses from original chain
	c.Focuses = c.Focuses[:index]

	// Adjust current index
	if c.CurrentIdx >= index {
		c.CurrentIdx = index - 1
	}

	c.UpdatedAt = time.Now()
	return newChain, nil
}

// Reverse reverses the order of focuses in the chain
func (c *Chain) Reverse() {
	for i, j := 0, len(c.Focuses)-1; i < j; i, j = i+1, j-1 {
		c.Focuses[i], c.Focuses[j] = c.Focuses[j], c.Focuses[i]
	}

	// Adjust current index
	if c.CurrentIdx >= 0 && c.CurrentIdx < len(c.Focuses) {
		c.CurrentIdx = len(c.Focuses) - 1 - c.CurrentIdx
	}

	c.UpdatedAt = time.Now()
}

// String returns a string representation of the chain
func (c *Chain) String() string {
	return fmt.Sprintf("Chain %s: %d focuses (current: %d)", c.Name, len(c.Focuses), c.CurrentIdx)
}

// generateChainID generates a unique ID for a chain
func generateChainID(name string) string {
	return fmt.Sprintf("chain-%s-%d", sanitizeForID(name), time.Now().UnixNano())
}
