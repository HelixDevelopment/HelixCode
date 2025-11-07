package focus

import (
	"fmt"
	"time"
)

// FocusType represents the type of focus
type FocusType string

const (
	FocusTypeFile      FocusType = "file"      // Single file
	FocusTypeDirectory FocusType = "directory" // Directory
	FocusTypeTask      FocusType = "task"      // Task/feature
	FocusTypeError     FocusType = "error"     // Error/bug
	FocusTypeTest      FocusType = "test"      // Test case
	FocusTypeFunction  FocusType = "function"  // Specific function
	FocusTypeClass     FocusType = "class"     // Class/struct
	FocusTypePackage   FocusType = "package"   // Package/module
	FocusTypeProject   FocusType = "project"   // Entire project
	FocusTypeCustom    FocusType = "custom"    // Custom focus
)

// FocusPriority represents the importance of a focus
type FocusPriority int

const (
	PriorityLow      FocusPriority = 1
	PriorityNormal   FocusPriority = 5
	PriorityHigh     FocusPriority = 10
	PriorityCritical FocusPriority = 20
)

// Focus represents a single point of attention in the development process
type Focus struct {
	ID          string                 // Unique identifier
	Type        FocusType              // Type of focus
	Target      string                 // Target (file path, task name, etc.)
	Description string                 // Human-readable description
	Priority    FocusPriority          // Priority level
	Context     map[string]interface{} // Additional context
	CreatedAt   time.Time              // When focus was created
	UpdatedAt   time.Time              // Last update time
	ExpiresAt   *time.Time             // Optional expiration time
	Parent      *Focus                 // Parent focus (for hierarchical focus)
	Children    []*Focus               // Child focuses
	Tags        []string               // Tags for categorization
	Metadata    map[string]string      // Custom metadata
}

// NewFocus creates a new focus with the given parameters
func NewFocus(focusType FocusType, target string) *Focus {
	now := time.Now()
	return &Focus{
		ID:          generateFocusID(focusType, target),
		Type:        focusType,
		Target:      target,
		Priority:    PriorityNormal,
		Context:     make(map[string]interface{}),
		CreatedAt:   now,
		UpdatedAt:   now,
		Children:    make([]*Focus, 0),
		Tags:        make([]string, 0),
		Metadata:    make(map[string]string),
	}
}

// NewFocusWithPriority creates a new focus with specified priority
func NewFocusWithPriority(focusType FocusType, target string, priority FocusPriority) *Focus {
	f := NewFocus(focusType, target)
	f.Priority = priority
	return f
}

// AddChild adds a child focus
func (f *Focus) AddChild(child *Focus) {
	child.Parent = f
	f.Children = append(f.Children, child)
	f.UpdatedAt = time.Now()
}

// RemoveChild removes a child focus by ID
func (f *Focus) RemoveChild(childID string) bool {
	for i, child := range f.Children {
		if child.ID == childID {
			f.Children = append(f.Children[:i], f.Children[i+1:]...)
			child.Parent = nil
			f.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// AddTag adds a tag to the focus
func (f *Focus) AddTag(tag string) {
	// Check if tag already exists
	for _, t := range f.Tags {
		if t == tag {
			return
		}
	}
	f.Tags = append(f.Tags, tag)
	f.UpdatedAt = time.Now()
}

// HasTag checks if focus has a specific tag
func (f *Focus) HasTag(tag string) bool {
	for _, t := range f.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// SetContext sets a context value
func (f *Focus) SetContext(key string, value interface{}) {
	f.Context[key] = value
	f.UpdatedAt = time.Now()
}

// GetContext gets a context value
func (f *Focus) GetContext(key string) (interface{}, bool) {
	value, ok := f.Context[key]
	return value, ok
}

// SetMetadata sets a metadata value
func (f *Focus) SetMetadata(key, value string) {
	f.Metadata[key] = value
	f.UpdatedAt = time.Now()
}

// GetMetadata gets a metadata value
func (f *Focus) GetMetadata(key string) (string, bool) {
	value, ok := f.Metadata[key]
	return value, ok
}

// SetExpiration sets an expiration time for the focus
func (f *Focus) SetExpiration(duration time.Duration) {
	expiresAt := time.Now().Add(duration)
	f.ExpiresAt = &expiresAt
	f.UpdatedAt = time.Now()
}

// IsExpired checks if the focus has expired
func (f *Focus) IsExpired() bool {
	if f.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*f.ExpiresAt)
}

// Touch updates the UpdatedAt timestamp
func (f *Focus) Touch() {
	f.UpdatedAt = time.Now()
}

// Clone creates a deep copy of the focus
func (f *Focus) Clone() *Focus {
	clone := &Focus{
		ID:          f.ID,
		Type:        f.Type,
		Target:      f.Target,
		Description: f.Description,
		Priority:    f.Priority,
		Context:     make(map[string]interface{}),
		CreatedAt:   f.CreatedAt,
		UpdatedAt:   f.UpdatedAt,
		ExpiresAt:   f.ExpiresAt,
		Children:    make([]*Focus, 0),
		Tags:        make([]string, len(f.Tags)),
		Metadata:    make(map[string]string),
	}

	// Copy context
	for k, v := range f.Context {
		clone.Context[k] = v
	}

	// Copy tags
	copy(clone.Tags, f.Tags)

	// Copy metadata
	for k, v := range f.Metadata {
		clone.Metadata[k] = v
	}

	// Clone children
	for _, child := range f.Children {
		childClone := child.Clone()
		clone.AddChild(childClone)
	}

	return clone
}

// String returns a string representation of the focus
func (f *Focus) String() string {
	return fmt.Sprintf("%s: %s (%s)", f.Type, f.Target, f.Description)
}

// Validate validates the focus
func (f *Focus) Validate() error {
	if f.ID == "" {
		return fmt.Errorf("focus ID cannot be empty")
	}

	if f.Type == "" {
		return fmt.Errorf("focus type cannot be empty")
	}

	if f.Target == "" {
		return fmt.Errorf("focus target cannot be empty")
	}

	if f.Priority < PriorityLow || f.Priority > PriorityCritical {
		return fmt.Errorf("invalid priority: %d", f.Priority)
	}

	// Validate expiration time
	if f.ExpiresAt != nil && f.ExpiresAt.Before(f.CreatedAt) {
		return fmt.Errorf("expiration time cannot be before creation time")
	}

	return nil
}

// GetDepth returns the depth of this focus in the hierarchy
func (f *Focus) GetDepth() int {
	depth := 0
	current := f.Parent
	for current != nil {
		depth++
		current = current.Parent
	}
	return depth
}

// GetRoot returns the root focus in the hierarchy
func (f *Focus) GetRoot() *Focus {
	current := f
	for current.Parent != nil {
		current = current.Parent
	}
	return current
}

// GetPath returns the path from root to this focus
func (f *Focus) GetPath() []*Focus {
	path := make([]*Focus, 0)
	current := f
	for current != nil {
		path = append([]*Focus{current}, path...)
		current = current.Parent
	}
	return path
}

// FindChild finds a child by ID (recursive)
func (f *Focus) FindChild(id string) *Focus {
	if f.ID == id {
		return f
	}

	for _, child := range f.Children {
		if found := child.FindChild(id); found != nil {
			return found
		}
	}

	return nil
}

// CountDescendants returns the total number of descendants
func (f *Focus) CountDescendants() int {
	count := len(f.Children)
	for _, child := range f.Children {
		count += child.CountDescendants()
	}
	return count
}

// generateFocusID generates a unique ID for a focus
func generateFocusID(focusType FocusType, target string) string {
	return fmt.Sprintf("%s-%s-%d", focusType, sanitizeForID(target), time.Now().UnixNano())
}

// sanitizeForID sanitizes a string for use in an ID
func sanitizeForID(s string) string {
	// Replace non-alphanumeric characters with hyphens
	result := ""
	for _, ch := range s {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			result += string(ch)
		} else {
			result += "-"
		}
	}
	return result
}
