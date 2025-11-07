package builder

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"dev.helix.code/internal/focus"
	"dev.helix.code/internal/session"
)

// Priority determines the importance of context items
type Priority int

const (
	PriorityLow      Priority = 1
	PriorityNormal   Priority = 5
	PriorityHigh     Priority = 10
	PriorityCritical Priority = 20
)

// SourceType represents the type of context source
type SourceType string

const (
	SourceSession  SourceType = "session"
	SourceFocus    SourceType = "focus"
	SourceFile     SourceType = "file"
	SourceGit      SourceType = "git"
	SourceProject  SourceType = "project"
	SourceError    SourceType = "error"
	SourceLog      SourceType = "log"
	SourceCustom   SourceType = "custom"
)

// ContextItem represents a single piece of context
type ContextItem struct {
	Type     SourceType            // Type of source
	Priority Priority              // Priority for inclusion
	Title    string                // Item title/label
	Content  string                // Item content
	Metadata map[string]string     // Additional metadata
	Size     int                   // Size in bytes
}

// Builder builds context for LLM calls
type Builder struct {
	items          []*ContextItem        // All context items
	sessionMgr     *session.Manager      // Session manager
	focusMgr       *focus.Manager        // Focus manager
	maxSize        int                   // Maximum context size (bytes)
	maxTokens      int                   // Maximum tokens (approximate)
	templates      map[string]*Template  // Context templates
	sources        map[SourceType]Source // Registered sources
	mu             sync.RWMutex          // Thread-safety
	cache          *cache                // Context cache
	cacheEnabled   bool                  // Whether caching is enabled
}

// Source provides context from a specific source
type Source interface {
	GetContext() ([]*ContextItem, error)
	Type() SourceType
}

// Template defines a context structure
type Template struct {
	Name        string              // Template name
	Description string              // Template description
	Sections    []*TemplateSection  // Template sections
}

// TemplateSection represents a section in a template
type TemplateSection struct {
	Title    string     // Section title
	Types    []SourceType // Source types to include
	Priority Priority   // Priority for this section
	MaxItems int        // Maximum items in section (0 = unlimited)
}

// NewBuilder creates a new context builder
func NewBuilder() *Builder {
	return &Builder{
		items:        make([]*ContextItem, 0),
		maxSize:      100000,  // 100KB default
		maxTokens:    4000,    // ~4K tokens default
		templates:    make(map[string]*Template),
		sources:      make(map[SourceType]Source),
		cache:        newCache(),
		cacheEnabled: true,
	}
}

// NewBuilderWithManagers creates a builder with session and focus managers
func NewBuilderWithManagers(sessionMgr *session.Manager, focusMgr *focus.Manager) *Builder {
	b := NewBuilder()
	b.sessionMgr = sessionMgr
	b.focusMgr = focusMgr
	return b
}

// AddItem adds a context item
func (b *Builder) AddItem(item *ContextItem) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Calculate size if not set
	if item.Size == 0 {
		item.Size = len(item.Content)
	}

	b.items = append(b.items, item)

	// Invalidate cache
	b.cache.invalidate()
}

// AddText adds simple text context
func (b *Builder) AddText(title, content string, priority Priority) {
	b.AddItem(&ContextItem{
		Type:     SourceCustom,
		Priority: priority,
		Title:    title,
		Content:  content,
		Metadata: make(map[string]string),
		Size:     len(content),
	})
}

// AddSession adds session context
func (b *Builder) AddSession(sess *session.Session) error {
	if sess == nil {
		return fmt.Errorf("session cannot be nil")
	}

	content := fmt.Sprintf("Session: %s\nMode: %s\nStatus: %s\nDuration: %v\n",
		sess.Name,
		sess.Mode,
		sess.Status,
		sess.Duration,
	)

	// Add metadata
	if len(sess.Metadata) > 0 {
		content += "\nMetadata:\n"
		for k, v := range sess.Metadata {
			content += fmt.Sprintf("  %s: %s\n", k, v)
		}
	}

	// Add tags
	if len(sess.Tags) > 0 {
		content += "\nTags: " + strings.Join(sess.Tags, ", ") + "\n"
	}

	b.AddItem(&ContextItem{
		Type:     SourceSession,
		Priority: PriorityHigh,
		Title:    "Current Session",
		Content:  content,
		Metadata: sess.Metadata,
		Size:     len(content),
	})

	return nil
}

// AddFocusChain adds focus chain context
func (b *Builder) AddFocusChain(chain *focus.Chain, maxItems int) error {
	if chain == nil {
		return fmt.Errorf("chain cannot be nil")
	}

	if maxItems <= 0 {
		maxItems = 10 // Default
	}

	recent := chain.GetRecent(maxItems)

	content := fmt.Sprintf("Focus Chain: %s\nRecent focuses (%d):\n", chain.Name, len(recent))

	for i, f := range recent {
		content += fmt.Sprintf("%d. %s (%s)\n", i+1, f.Target, f.Type)

		// Add tags if present
		if len(f.Tags) > 0 {
			content += fmt.Sprintf("   Tags: %s\n", strings.Join(f.Tags, ", "))
		}
	}

	b.AddItem(&ContextItem{
		Type:     SourceFocus,
		Priority: PriorityHigh,
		Title:    "Recent Focus",
		Content:  content,
		Metadata: make(map[string]string),
		Size:     len(content),
	})

	return nil
}

// RegisterSource registers a custom source
func (b *Builder) RegisterSource(source Source) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.sources[source.Type()] = source

	// Invalidate cache
	b.cache.invalidate()
}

// RegisterTemplate registers a context template
func (b *Builder) RegisterTemplate(template *Template) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.templates[template.Name] = template
}

// Build builds the context string
func (b *Builder) Build() (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Check cache
	if b.cacheEnabled {
		if cached, ok := b.cache.get(); ok {
			return cached, nil
		}
	}

	// Sort items by priority (highest first)
	sorted := make([]*ContextItem, len(b.items))
	copy(sorted, b.items)
	b.sortByPriority(sorted)

	// Build context within size limits
	var result strings.Builder
	totalSize := 0

	for _, item := range sorted {
		// Check if adding this item would exceed limits
		if totalSize+item.Size > b.maxSize {
			break
		}

		// Add item
		if item.Title != "" {
			result.WriteString("## ")
			result.WriteString(item.Title)
			result.WriteString("\n\n")
		}

		result.WriteString(item.Content)
		result.WriteString("\n\n")

		totalSize += item.Size
	}

	context := result.String()

	// Cache result
	if b.cacheEnabled {
		b.cache.set(context)
	}

	return context, nil
}

// BuildWithTemplate builds context using a template
func (b *Builder) BuildWithTemplate(templateName string) (string, error) {
	b.mu.RLock()
	template, exists := b.templates[templateName]
	b.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("template not found: %s", templateName)
	}

	// Check cache with template key
	cacheKey := "template:" + templateName
	if b.cacheEnabled {
		if cached, ok := b.cache.getWithKey(cacheKey); ok {
			return cached, nil
		}
	}

	var result strings.Builder

	// Process each section
	for _, section := range template.Sections {
		// Get items for this section
		items := b.getItemsByTypes(section.Types)

		// Sort by priority
		b.sortByPriority(items)

		// Limit items if specified
		if section.MaxItems > 0 && len(items) > section.MaxItems {
			items = items[:section.MaxItems]
		}

		if len(items) == 0 {
			continue
		}

		// Add section header
		result.WriteString("## ")
		result.WriteString(section.Title)
		result.WriteString("\n\n")

		// Add items
		for _, item := range items {
			result.WriteString(item.Content)
			result.WriteString("\n\n")
		}
	}

	context := result.String()

	// Cache result with template key
	if b.cacheEnabled {
		b.cache.setWithKey(cacheKey, context)
	}

	return context, nil
}

// BuildFromSources builds context by fetching from registered sources
func (b *Builder) BuildFromSources() (string, error) {
	b.mu.RLock()
	sources := make([]Source, 0, len(b.sources))
	for _, source := range b.sources {
		sources = append(sources, source)
	}
	b.mu.RUnlock()

	// Fetch from all sources
	for _, source := range sources {
		items, err := source.GetContext()
		if err != nil {
			// Log error but continue with other sources
			continue
		}

		for _, item := range items {
			b.AddItem(item)
		}
	}

	return b.Build()
}

// Clear clears all context items
func (b *Builder) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.items = make([]*ContextItem, 0)
	b.cache.invalidate()
}

// GetItems returns all context items
func (b *Builder) GetItems() []*ContextItem {
	b.mu.RLock()
	defer b.mu.RUnlock()

	items := make([]*ContextItem, len(b.items))
	copy(items, b.items)
	return items
}

// GetItemsByType returns items of a specific type
func (b *Builder) GetItemsByType(sourceType SourceType) []*ContextItem {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.getItemsByTypes([]SourceType{sourceType})
}

// getItemsByTypes returns items matching any of the given types (internal, no lock)
func (b *Builder) getItemsByTypes(types []SourceType) []*ContextItem {
	items := make([]*ContextItem, 0)

	for _, item := range b.items {
		for _, t := range types {
			if item.Type == t {
				items = append(items, item)
				break
			}
		}
	}

	return items
}

// GetItemsByPriority returns items with at least the given priority
func (b *Builder) GetItemsByPriority(minPriority Priority) []*ContextItem {
	b.mu.RLock()
	defer b.mu.RUnlock()

	items := make([]*ContextItem, 0)

	for _, item := range b.items {
		if item.Priority >= minPriority {
			items = append(items, item)
		}
	}

	return items
}

// Count returns the number of context items
func (b *Builder) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.items)
}

// TotalSize returns the total size of all context items
func (b *Builder) TotalSize() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	total := 0
	for _, item := range b.items {
		total += item.Size
	}

	return total
}

// SetMaxSize sets the maximum context size in bytes
func (b *Builder) SetMaxSize(size int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.maxSize = size
	b.cache.invalidate()
}

// SetMaxTokens sets the maximum tokens (approximate)
func (b *Builder) SetMaxTokens(tokens int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.maxTokens = tokens
	// Rough approximation: 1 token ~= 4 bytes
	b.maxSize = tokens * 4
	b.cache.invalidate()
}

// EnableCache enables context caching
func (b *Builder) EnableCache(enabled bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.cacheEnabled = enabled
	if !enabled {
		b.cache.invalidate()
	}
}

// InvalidateCache invalidates the cached context
func (b *Builder) InvalidateCache() {
	b.cache.invalidate()
}

// sortByPriority sorts items by priority (highest first)
func (b *Builder) sortByPriority(items []*ContextItem) {
	// Simple bubble sort by priority
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].Priority > items[i].Priority {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

// Statistics contains builder statistics
type Statistics struct {
	TotalItems    int                   // Total context items
	TotalSize     int                   // Total size in bytes
	ByType        map[SourceType]int    // Count by source type
	ByPriority    map[Priority]int      // Count by priority
	CacheHits     int                   // Cache hits
	CacheMisses   int                   // Cache misses
}

// GetStatistics returns builder statistics
func (b *Builder) GetStatistics() *Statistics {
	b.mu.RLock()
	defer b.mu.RUnlock()

	stats := &Statistics{
		TotalItems: len(b.items),
		ByType:     make(map[SourceType]int),
		ByPriority: make(map[Priority]int),
	}

	totalSize := 0
	for _, item := range b.items {
		stats.ByType[item.Type]++
		stats.ByPriority[item.Priority]++
		totalSize += item.Size
	}

	stats.TotalSize = totalSize
	stats.CacheHits = b.cache.hits
	stats.CacheMisses = b.cache.misses

	return stats
}

// cache manages context caching
type cache struct {
	data       map[string]string // Cached contexts by key
	validUntil map[string]time.Time // Expiration times
	ttl        time.Duration     // Time to live
	hits       int               // Cache hits
	misses     int               // Cache misses
	mu         sync.RWMutex      // Thread-safety
}

// newCache creates a new cache
func newCache() *cache {
	return &cache{
		data:       make(map[string]string),
		validUntil: make(map[string]time.Time),
		ttl:        5 * time.Minute, // Default TTL
	}
}

// get gets cached context with default key
func (c *cache) get() (string, bool) {
	return c.getWithKey("default")
}

// getWithKey gets cached context with specific key
func (c *cache) getWithKey(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check if exists
	context, exists := c.data[key]
	if !exists {
		c.misses++
		return "", false
	}

	// Check if expired
	if time.Now().After(c.validUntil[key]) {
		c.misses++
		return "", false
	}

	c.hits++
	return context, true
}

// set sets cached context with default key
func (c *cache) set(context string) {
	c.setWithKey("default", context)
}

// setWithKey sets cached context with specific key
func (c *cache) setWithKey(key, context string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = context
	c.validUntil[key] = time.Now().Add(c.ttl)
}

// invalidate invalidates all cached contexts
func (c *cache) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]string)
	c.validUntil = make(map[string]time.Time)
}

// setTTL sets the cache time-to-live
func (c *cache) setTTL(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ttl = ttl
}
