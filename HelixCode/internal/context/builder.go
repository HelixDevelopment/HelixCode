// Package context provides context building functionality for AI conversations
package context

import (
	"fmt"
	"strings"
	"sync"

	"dev.helix.code/internal/memory"
)

// Builder builds AI conversation context from various sources
type Builder struct {
	messages   []*memory.Message
	metadata   map[string]string
	systemRole string
	mu         sync.RWMutex
}

// NewBuilder creates a new context builder
func NewBuilder() *Builder {
	return &Builder{
		messages: make([]*memory.Message, 0),
		metadata: make(map[string]string),
	}
}

// SetSystemRole sets the system role message
func (b *Builder) SetSystemRole(role string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.systemRole = role
}

// AddMessage adds a message to the context
func (b *Builder) AddMessage(msg *memory.Message) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.messages = append(b.messages, msg)
}

// AddUserMessage adds a user message to the context
func (b *Builder) AddUserMessage(content string) {
	b.AddMessage(memory.NewUserMessage(content))
}

// AddAssistantMessage adds an assistant message to the context
func (b *Builder) AddAssistantMessage(content string) {
	b.AddMessage(memory.NewAssistantMessage(content))
}

// SetMetadata sets a metadata key-value pair
func (b *Builder) SetMetadata(key, value string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.metadata[key] = value
}

// GetMetadata retrieves metadata value
func (b *Builder) GetMetadata(key string) string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.metadata[key]
}

// Build builds the final conversation
func (b *Builder) Build() *memory.Conversation {
	b.mu.RLock()
	defer b.mu.RUnlock()

	title := b.metadata["title"]
	if title == "" {
		title = "Conversation"
	}

	conv := memory.NewConversation(title)

	// Add system message if set
	if b.systemRole != "" {
		conv.AddMessage(memory.NewSystemMessage(b.systemRole))
	}

	// Add all messages
	for _, msg := range b.messages {
		conv.AddMessage(msg)
	}

	// Set metadata
	for key, value := range b.metadata {
		conv.SetMetadata(key, value)
	}

	return conv
}

// ToText converts the context to plain text
func (b *Builder) ToText() string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var sb strings.Builder

	if b.systemRole != "" {
		sb.WriteString(fmt.Sprintf("[system] %s\n\n", b.systemRole))
	}

	for _, msg := range b.messages {
		sb.WriteString(fmt.Sprintf("[%s] %s\n\n", msg.Role, msg.Content))
	}

	return sb.String()
}

// MessageCount returns the number of messages
func (b *Builder) MessageCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.messages)
}

// Clear clears all messages and metadata
func (b *Builder) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.messages = make([]*memory.Message, 0)
	b.metadata = make(map[string]string)
	b.systemRole = ""
}

// Clone creates a copy of the builder
func (b *Builder) Clone() *Builder {
	b.mu.RLock()
	defer b.mu.RUnlock()

	clone := NewBuilder()
	clone.systemRole = b.systemRole

	for _, msg := range b.messages {
		clone.messages = append(clone.messages, msg.Clone())
	}

	for key, value := range b.metadata {
		clone.metadata[key] = value
	}

	return clone
}

// FromConversation creates a builder from an existing conversation
func FromConversation(conv *memory.Conversation) *Builder {
	builder := NewBuilder()

	messages := conv.GetMessages()
	for _, msg := range messages {
		if msg.Role == memory.RoleSystem {
			builder.SetSystemRole(msg.Content)
		} else {
			builder.AddMessage(msg)
		}
	}

	// Copy metadata
	if conv.Metadata != nil {
		for key, value := range conv.Metadata {
			builder.SetMetadata(key, value)
		}
	}

	return builder
}
