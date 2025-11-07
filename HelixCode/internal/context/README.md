# Context Builder Package

Provides context building functionality for AI conversations in HelixCode.

## Overview

The Context Builder package helps you construct AI conversation context from various sources. It provides a fluent API for building conversations with system roles, messages, and metadata.

## Features

- Fluent API for building conversations
- System role configuration
- Message management (user, assistant, system)
- Metadata support
- Thread-safe operations
- Conversion to/from Conversation objects
- Text export functionality

## Usage

### Basic Example

```go
import "dev.helix.code/internal/context"

// Create builder
builder := context.NewBuilder()

// Set system role
builder.SetSystemRole("You are a helpful coding assistant")

// Add messages
builder.AddUserMessage("Help me write a function")
builder.AddAssistantMessage("Sure! What should the function do?")
builder.AddUserMessage("Calculate factorial")

// Add metadata
builder.SetMetadata("title", "Factorial Implementation")
builder.SetMetadata("language", "Go")

// Build conversation
conv := builder.Build()
```

### From Existing Conversation

```go
// Create builder from existing conversation
builder := context.FromConversation(existingConv)

// Modify and rebuild
builder.AddUserMessage("Additional question")
newConv := builder.Build()
```

### Export to Text

```go
builder := context.NewBuilder()
builder.SetSystemRole("You are an expert")
builder.AddUserMessage("Question")
builder.AddAssistantMessage("Answer")

text := builder.ToText()
/*
Output:
[system] You are an expert

[user] Question

[assistant] Answer
*/
```

## API Reference

### Builder

#### Creation
- `NewBuilder() *Builder` - Creates a new builder
- `FromConversation(*Conversation) *Builder` - Creates builder from conversation

#### Configuration
- `SetSystemRole(string)` - Sets the system role message
- `SetMetadata(key, value string)` - Sets metadata
- `GetMetadata(key string) string` - Retrieves metadata

#### Messages
- `AddMessage(*Message)` - Adds a message
- `AddUserMessage(string)` - Adds a user message
- `AddAssistantMessage(string)` - Adds an assistant message

#### Building
- `Build() *Conversation` - Builds the final conversation
- `ToText() string` - Converts to plain text

#### Utility
- `MessageCount() int` - Returns number of messages
- `Clear()` - Clears all data
- `Clone() *Builder` - Creates a copy

## Thread Safety

All operations are thread-safe. The builder uses `sync.RWMutex` for concurrent access.

```go
builder := context.NewBuilder()

// Safe to use from multiple goroutines
go builder.AddUserMessage("Message 1")
go builder.AddUserMessage("Message 2")
```

## Integration with Memory System

The Context Builder integrates seamlessly with the Memory System:

```go
import (
    "dev.helix.code/internal/context"
    "dev.helix.code/internal/memory"
)

// Build context
builder := context.NewBuilder()
builder.SetSystemRole("Expert assistant")
builder.AddUserMessage("Question")

// Create conversation
conv := builder.Build()

// Add to memory manager
memoryMgr := memory.NewManager()
// Note: The builder creates a new conversation
// To add to existing manager, use:
for _, msg := range conv.GetMessages() {
    memoryMgr.AddMessage(existingConvID, msg)
}
```

## Best Practices

1. **Set system role first** - Establishes AI behavior
2. **Use metadata** - Store contextual information
3. **Clone for variations** - Reuse base context
4. **Clear when done** - Free resources
5. **Export for debugging** - Use ToText() to inspect

## Examples

See `/examples/phase3/` for complete working examples.

## Testing

Run tests:
```bash
go test ./internal/context -v
```

Test coverage:
```bash
go test ./internal/context -cover
```

## Package Structure

```
internal/context/
├── builder.go       # Main builder implementation
├── builder_test.go  # Comprehensive tests
└── README.md        # This file
```

## Related Packages

- `internal/memory` - Conversation and message management
- `internal/session` - Session management
- `internal/template` - Template system
