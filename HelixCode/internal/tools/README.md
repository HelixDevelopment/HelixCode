# HelixCode Tools

> A comprehensive tool ecosystem for AI-powered development workflows

## Quick Start

```go
import "dev.helix.code/internal/tools"

// Create registry
registry, err := tools.NewToolRegistry(tools.DefaultRegistryConfig())
if err != nil {
    log.Fatal(err)
}
defer registry.Close()

// Execute a tool
ctx := context.Background()
result, err := registry.Execute(ctx, "fs_read", map[string]interface{}{
    "path": "/path/to/file.go",
})
```

## Available Packages

| Package | Purpose | Tools |
|---------|---------|-------|
| **filesystem/** | File operations | fs_read, fs_write, fs_edit, glob, grep |
| **shell/** | Command execution | shell, shell_background, shell_output, shell_kill |
| **web/** | Web scraping | web_fetch, web_search |
| **git/** | Git automation | (integrated via git package) |
| **browser/** | Browser control | browser_launch, browser_navigate, browser_screenshot, browser_close |
| **voice/** | Voice input | (integrated via voice package) |
| **mapping/** | Code analysis | codebase_map, file_definitions |
| **multiedit/** | Transactional editing | multiedit_begin, multiedit_add, multiedit_preview, multiedit_commit |
| **confirmation/** | User prompts | (integrated via confirmation package) |

## Tool Registry

The `ToolRegistry` provides a unified interface for all tools:

```go
// Get a tool
tool, err := registry.Get("fs_read")

// Validate parameters
err := tool.Validate(params)

// Get schema
schema := tool.Schema()

// Execute
result, err := tool.Execute(ctx, params)

// List by category
fsTools := registry.ListByCategory(tools.CategoryFileSystem)
```

## Categories

- **CategoryFileSystem** - File operations
- **CategoryShell** - Command execution
- **CategoryWeb** - Web scraping and search
- **CategoryBrowser** - Browser automation
- **CategoryMapping** - Codebase analysis
- **CategoryMultiEdit** - Multi-file editing
- **CategoryInteractive** - User interaction
- **CategoryNotebook** - Jupyter notebooks

## Security

All tools implement comprehensive security:

- Path validation and workspace boundaries
- Command blocklist for dangerous operations
- Resource limits (CPU, memory, processes)
- Timeout enforcement
- Audit logging
- Sandbox isolation
- Sensitive file detection

## Examples

### File Operations

```go
// Read a file
content, _ := registry.Execute(ctx, "fs_read", map[string]interface{}{
    "path": "main.go",
})

// Write a file
registry.Execute(ctx, "fs_write", map[string]interface{}{
    "path": "output.txt",
    "content": "Hello, World!",
})

// Edit a file
registry.Execute(ctx, "fs_edit", map[string]interface{}{
    "path": "config.go",
    "old_string": "localhost:8080",
    "new_string": "0.0.0.0:8080",
})

// Search files
matches, _ := registry.Execute(ctx, "grep", map[string]interface{}{
    "pattern": "TODO:",
    "regex": false,
})
```

### Shell Execution

```go
// Execute command
result, _ := registry.Execute(ctx, "shell", map[string]interface{}{
    "command": "go test ./...",
    "timeout": 300,
})

// Background execution
execution, _ := registry.Execute(ctx, "shell_background", map[string]interface{}{
    "command": "npm run build",
})

// Check status
status, _ := registry.Execute(ctx, "shell_output", map[string]interface{}{
    "execution_id": execution.ID,
})
```

### Web Operations

```go
// Fetch webpage
result, _ := registry.Execute(ctx, "web_fetch", map[string]interface{}{
    "url": "https://example.com/docs",
    "parse_markdown": true,
})

// Search
results, _ := registry.Execute(ctx, "web_search", map[string]interface{}{
    "query": "golang best practices 2025",
    "max_results": 5,
})
```

### Multi-File Editing

```go
// Begin transaction
tx, _ := registry.Execute(ctx, "multiedit_begin", map[string]interface{}{
    "description": "Refactor authentication",
})

// Add edits
registry.Execute(ctx, "multiedit_add", map[string]interface{}{
    "transaction_id": tx.ID,
    "file_path": "internal/auth/handler.go",
    "operation": "update",
    "new_content": updatedContent,
})

// Preview
preview, _ := registry.Execute(ctx, "multiedit_preview", map[string]interface{}{
    "transaction_id": tx.ID,
})

// Commit
registry.Execute(ctx, "multiedit_commit", map[string]interface{}{
    "transaction_id": tx.ID,
})
```

## Documentation

- **[Complete Tool Reference](../../docs/TOOLS.md)** - Detailed documentation for all tools
- **[Implementation Summary](./SUMMARY.md)** - Package status and statistics
- **[Tests](./registry_test.go)** - Integration tests and examples

## Testing

```bash
# Run all tool tests
go test -v ./internal/tools/...

# Run specific package tests
go test -v ./internal/tools/filesystem/...

# Run integration tests
go test -v ./internal/tools/ -run TestIntegration

# Run with coverage
go test -cover ./internal/tools/...

# Benchmarks
go test -bench=. ./internal/tools/
```

## Configuration

Customize tool behavior with configuration:

```go
config := tools.DefaultRegistryConfig()

// FileSystem
config.FileSystemConfig.MaxFileSize = 100 * 1024 * 1024 // 100 MB
config.FileSystemConfig.CacheEnabled = true

// Shell
config.ShellConfig.MaxConcurrent = 10
config.ShellConfig.DefaultTimeout = 30 * time.Second

// Web
config.WebConfig.RateLimitEnabled = true
config.WebConfig.MaxContentSize = 10 * 1024 * 1024

// Browser
config.BrowserConfig.MaxConcurrentBrowsers = 5

registry, _ := tools.NewToolRegistry(config)
```

## Extending

Create custom tools by implementing the `Tool` interface:

```go
type CustomTool struct {
    registry *ToolRegistry
}

func (t *CustomTool) Name() string { return "custom_tool" }
func (t *CustomTool) Description() string { return "Custom functionality" }
func (t *CustomTool) Category() ToolCategory { return CategoryFileSystem }
func (t *CustomTool) Schema() ToolSchema { /* return schema */ }
func (t *CustomTool) Validate(params map[string]interface{}) error { /* validate */ }
func (t *CustomTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    // Implementation
}

// Register
registry.Register(&CustomTool{registry: registry})
```

## Architecture

```
┌─────────────────────────────────────────────┐
│           Tool Registry                     │
│  - Tool management                          │
│  - Schema validation                        │
│  - Resource lifecycle                       │
└─────────────────┬───────────────────────────┘
                  │
      ┌───────────┴────────────┐
      │                        │
┌─────▼─────┐          ┌───────▼───────┐
│ FileSystem│          │    Shell      │
│  Tools    │          │   Executor    │
└─────┬─────┘          └───────┬───────┘
      │                        │
┌─────▼─────┐          ┌───────▼───────┐
│    Web    │          │   Browser     │
│   Tools   │          │   Tools       │
└─────┬─────┘          └───────┬───────┘
      │                        │
┌─────▼─────┐          ┌───────▼───────┐
│  Mapping  │          │  MultiEdit    │
│   Tools   │          │   Manager     │
└───────────┘          └───────────────┘
```

## Performance

- **Caching**: Intelligent caching with TTL
- **Concurrency**: Parallel execution where safe
- **Resource Limits**: CPU, memory, process limits
- **Streaming**: Large file handling
- **Incremental**: Updates without full rebuilds

## License

Copyright (c) 2025 HelixCode Project
