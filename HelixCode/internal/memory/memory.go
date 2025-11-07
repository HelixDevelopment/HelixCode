package memory

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

// Role represents the role of a message sender
type Role string

const (
	RoleUser      Role = "user"       // User message
	RoleAssistant Role = "assistant"  // AI assistant response
	RoleSystem    Role = "system"     // System message
	RoleTool      Role = "tool"       // Tool/function result
)

// IsValid checks if role is valid
func (r Role) IsValid() bool {
	switch r {
	case RoleUser, RoleAssistant, RoleSystem, RoleTool:
		return true
	}
	return false
}

// String returns string representation
func (r Role) String() string {
	return string(r)
}

// Message represents a single message in conversation history
type Message struct {
	ID        string            // Unique message ID
	Role      Role              // Message role
	Content   string            // Message content
	Timestamp time.Time         // When message was created
	Metadata  map[string]string // Additional metadata
	TokenCount int              // Approximate token count
	Size      int               // Size in bytes
}

// NewMessage creates a new message
func NewMessage(role Role, content string) *Message {
	return &Message{
		ID:         generateMessageID(),
		Role:       role,
		Content:    content,
		Timestamp:  time.Now(),
		Metadata:   make(map[string]string),
		TokenCount: estimateTokens(content),
		Size:       len(content),
	}
}

// NewUserMessage creates a new user message
func NewUserMessage(content string) *Message {
	return NewMessage(RoleUser, content)
}

// NewAssistantMessage creates a new assistant message
func NewAssistantMessage(content string) *Message {
	return NewMessage(RoleAssistant, content)
}

// NewSystemMessage creates a new system message
func NewSystemMessage(content string) *Message {
	return NewMessage(RoleSystem, content)
}

// SetMetadata sets a metadata value
func (m *Message) SetMetadata(key, value string) {
	m.Metadata[key] = value
}

// GetMetadata gets a metadata value
func (m *Message) GetMetadata(key string) (string, bool) {
	value, ok := m.Metadata[key]
	return value, ok
}

// Clone creates a copy of the message
func (m *Message) Clone() *Message {
	clone := &Message{
		ID:         m.ID,
		Role:       m.Role,
		Content:    m.Content,
		Timestamp:  m.Timestamp,
		Metadata:   make(map[string]string),
		TokenCount: m.TokenCount,
		Size:       m.Size,
	}

	for k, v := range m.Metadata {
		clone.Metadata[k] = v
	}

	return clone
}

// String returns a string representation
func (m *Message) String() string {
	return fmt.Sprintf("[%s] %s: %s", m.Timestamp.Format("15:04:05"), m.Role, truncate(m.Content, 50))
}

// Validate validates the message
func (m *Message) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("message ID cannot be empty")
	}

	if !m.Role.IsValid() {
		return fmt.Errorf("invalid role: %s", m.Role)
	}

	if m.Content == "" {
		return fmt.Errorf("message content cannot be empty")
	}

	return nil
}

// Conversation represents a conversation with message history
type Conversation struct {
	ID          string            // Unique conversation ID
	Title       string            // Conversation title
	SessionID   string            // Associated session ID
	Messages    []*Message        // Conversation messages
	Metadata    map[string]string // Additional metadata
	CreatedAt   time.Time         // When created
	UpdatedAt   time.Time         // Last updated
	Summary     string            // Conversation summary
	TokenCount  int               // Total tokens
	MessageCount int              // Total messages
}

// NewConversation creates a new conversation
func NewConversation(title string) *Conversation {
	return &Conversation{
		ID:           generateConversationID(),
		Title:        title,
		Messages:     make([]*Message, 0),
		Metadata:     make(map[string]string),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		TokenCount:   0,
		MessageCount: 0,
	}
}

// AddMessage adds a message to the conversation
func (c *Conversation) AddMessage(message *Message) {
	c.Messages = append(c.Messages, message)
	c.TokenCount += message.TokenCount
	c.MessageCount++
	c.UpdatedAt = time.Now()
}

// GetMessages returns all messages
func (c *Conversation) GetMessages() []*Message {
	messages := make([]*Message, len(c.Messages))
	copy(messages, c.Messages)
	return messages
}

// GetMessagesByRole returns messages with specific role
func (c *Conversation) GetMessagesByRole(role Role) []*Message {
	messages := make([]*Message, 0)
	for _, msg := range c.Messages {
		if msg.Role == role {
			messages = append(messages, msg)
		}
	}
	return messages
}

// GetRecent returns the N most recent messages
func (c *Conversation) GetRecent(n int) []*Message {
	if n <= 0 || n > len(c.Messages) {
		n = len(c.Messages)
	}

	start := len(c.Messages) - n
	messages := make([]*Message, n)
	copy(messages, c.Messages[start:])
	return messages
}

// GetRange returns messages in a range
func (c *Conversation) GetRange(start, end int) []*Message {
	if start < 0 {
		start = 0
	}
	if end > len(c.Messages) {
		end = len(c.Messages)
	}
	if start >= end {
		return []*Message{}
	}

	messages := make([]*Message, end-start)
	copy(messages, c.Messages[start:end])
	return messages
}

// Search searches for messages containing text
func (c *Conversation) Search(query string) []*Message {
	query = strings.ToLower(query)
	messages := make([]*Message, 0)

	for _, msg := range c.Messages {
		if strings.Contains(strings.ToLower(msg.Content), query) {
			messages = append(messages, msg)
		}
	}

	return messages
}

// Clear clears all messages
func (c *Conversation) Clear() {
	c.Messages = make([]*Message, 0)
	c.TokenCount = 0
	c.MessageCount = 0
	c.UpdatedAt = time.Now()
}

// Truncate keeps only the last N messages
func (c *Conversation) Truncate(keepLast int) int {
	if keepLast <= 0 || keepLast >= len(c.Messages) {
		return 0
	}

	removed := len(c.Messages) - keepLast
	c.Messages = c.Messages[len(c.Messages)-keepLast:]

	// Recalculate token count
	c.TokenCount = 0
	for _, msg := range c.Messages {
		c.TokenCount += msg.TokenCount
	}

	c.MessageCount = len(c.Messages)
	c.UpdatedAt = time.Now()

	return removed
}

// SetMetadata sets a metadata value
func (c *Conversation) SetMetadata(key, value string) {
	c.Metadata[key] = value
}

// GetMetadata gets a metadata value
func (c *Conversation) GetMetadata(key string) (string, bool) {
	value, ok := c.Metadata[key]
	return value, ok
}

// Clone creates a copy of the conversation
func (c *Conversation) Clone() *Conversation {
	clone := &Conversation{
		ID:           c.ID,
		Title:        c.Title,
		SessionID:    c.SessionID,
		Messages:     make([]*Message, len(c.Messages)),
		Metadata:     make(map[string]string),
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
		Summary:      c.Summary,
		TokenCount:   c.TokenCount,
		MessageCount: c.MessageCount,
	}

	// Deep copy messages
	for i, msg := range c.Messages {
		clone.Messages[i] = msg.Clone()
	}

	// Copy metadata
	for k, v := range c.Metadata {
		clone.Metadata[k] = v
	}

	return clone
}

// String returns a string representation
func (c *Conversation) String() string {
	return fmt.Sprintf("Conversation %s: %s (%d messages, %d tokens)",
		c.ID, c.Title, c.MessageCount, c.TokenCount)
}

// Validate validates the conversation
func (c *Conversation) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("conversation ID cannot be empty")
	}

	if c.Title == "" {
		return fmt.Errorf("conversation title cannot be empty")
	}

	return nil
}

// ToText converts conversation to text format
func (c *Conversation) ToText() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Conversation: %s\n", c.Title))
	builder.WriteString(fmt.Sprintf("Created: %s\n", c.CreatedAt.Format("2006-01-02 15:04:05")))
	builder.WriteString(fmt.Sprintf("Messages: %d\n\n", c.MessageCount))

	for _, msg := range c.Messages {
		builder.WriteString(fmt.Sprintf("[%s] %s:\n%s\n\n",
			msg.Timestamp.Format("15:04:05"),
			msg.Role,
			msg.Content))
	}

	return builder.String()
}

// Statistics contains conversation statistics
type Statistics struct {
	TotalMessages   int               // Total messages
	ByRole          map[Role]int      // Count by role
	TotalTokens     int               // Total tokens
	AverageTokens   float64           // Average tokens per message
	TotalSize       int               // Total size in bytes
	OldestMessage   time.Time         // Oldest message timestamp
	NewestMessage   time.Time         // Newest message timestamp
}

// GetStatistics returns conversation statistics
func (c *Conversation) GetStatistics() *Statistics {
	stats := &Statistics{
		TotalMessages: len(c.Messages),
		ByRole:        make(map[Role]int),
		TotalTokens:   c.TokenCount,
		TotalSize:     0,
	}

	if len(c.Messages) == 0 {
		return stats
	}

	for _, msg := range c.Messages {
		stats.ByRole[msg.Role]++
		stats.TotalSize += msg.Size
	}

	stats.AverageTokens = float64(stats.TotalTokens) / float64(stats.TotalMessages)
	stats.OldestMessage = c.Messages[0].Timestamp
	stats.NewestMessage = c.Messages[len(c.Messages)-1].Timestamp

	return stats
}

// Counters for unique ID generation
var (
	messageCounter      uint64
	conversationCounter uint64
)

// generateMessageID generates a unique message ID
func generateMessageID() string {
	count := atomic.AddUint64(&messageCounter, 1)
	return fmt.Sprintf("msg-%d-%d", time.Now().UnixNano(), count)
}

// generateConversationID generates a unique conversation ID
func generateConversationID() string {
	count := atomic.AddUint64(&conversationCounter, 1)
	return fmt.Sprintf("conv-%d-%d", time.Now().UnixNano(), count)
}

// estimateTokens estimates the number of tokens in text
// Rough approximation: 1 token ~= 4 characters
func estimateTokens(text string) int {
	return len(text) / 4
}

// truncate truncates a string to max length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
