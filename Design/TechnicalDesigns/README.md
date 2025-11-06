# HelixCode Technical Design Documents

This directory contains comprehensive technical design documents for HelixCode's core features. Each document provides production-ready specifications for implementation.

## Documents

### 1. [FileSystemTools.md](./FileSystemTools.md)
**File System Operations**

Complete design for secure, efficient file system operations including:
- Reader, Writer, Editor, and Searcher components
- Security with path validation and permission checks
- Caching and performance optimization
- Batch operations and streaming for large files
- Comprehensive error handling
- Testing strategies

**Key Interfaces**: FileReader, FileWriter, FileEditor, FileSearcher

**References**: Cline's file tools, Qwen Code's file operations

---

### 2. [ShellExecution.md](./ShellExecution.md)
**Shell Command Execution**

Secure and controlled shell command execution with:
- Command validation and sandboxing
- Real-time output streaming
- Allowlist/blocklist security management
- Timeout and resource limit enforcement
- Signal handling and process control
- Environment isolation

**Key Interfaces**: CommandExecutor, ExecutionEnvironment, SecurityManager

**References**: Cline's shell execution, Aider's command execution

---

### 3. [PlanMode.md](./PlanMode.md)
**Two-Phase Planning Workflow**

Intelligent planning system that generates and executes implementation strategies:
- Plan generation with multiple options
- Option ranking and comparison
- User selection interface
- Progress tracking during execution
- State management across phases
- Mode transitions (Normal → Plan → Act)

**Key Interfaces**: Planner, OptionPresenter, Executor, ModeController

**References**: Cline's Plan Mode implementation

---

### 4. [BrowserControl.md](./BrowserControl.md)
**Browser Automation**

Browser automation and control capabilities:
- Chrome/Chromium discovery and launch
- Page navigation and interaction
- Screenshot capture with annotation
- Console monitoring
- Element selection and JavaScript evaluation
- Headless and visible modes

**Key Interfaces**: Controller, ActionExecutor, ChromeDiscovery

**References**: Cline's Puppeteer integration

**Go Libraries**: chromedp, go-rod

---

### 5. [CodebaseMapping.md](./CodebaseMapping.md)
**Tree-sitter Based Codebase Analysis**

Comprehensive codebase understanding through semantic parsing:
- 30+ programming languages support
- Definition extraction (functions, classes, methods, types)
- Import and dependency analysis
- Token counting for context management
- Disk-based caching (.helix.cache/)
- Complexity calculation
- Incremental updates

**Key Interfaces**: Mapper, TreeSitterParser, LanguageRegistry, CacheManager

**References**: Aider's repomap.py, Plandex's tree-sitter integration

---

## Common Patterns

### Security
All designs incorporate comprehensive security measures:
- Input validation and sanitization
- Path traversal prevention
- Permission checking
- Sandboxing and isolation
- Audit logging

### Performance
Optimization strategies across all components:
- Caching (LRU, disk-based)
- Concurrent operations with semaphores
- Streaming for large data
- Batch processing
- Resource limits

### Error Handling
Robust error handling:
- Typed errors with context
- Error wrapping and unwrapping
- Recovery strategies
- Detailed error messages

### Testing
Comprehensive testing strategies:
- Unit tests with table-driven tests
- Integration tests for workflows
- Benchmark tests for performance
- Mock implementations for dependencies
- Security-focused tests

## Implementation Guidelines

### Phase 1: Core Infrastructure
1. FileSystemTools - Foundation for all file operations
2. ShellExecution - Enable command execution
3. Basic error handling and logging

### Phase 2: Intelligence Layer
4. CodebaseMapping - Understand codebase structure
5. PlanMode - Intelligent planning and execution

### Phase 3: Advanced Features
6. BrowserControl - Web automation capabilities
7. Integration with AI providers
8. Advanced caching and optimization

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      HelixCode                              │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │   PlanMode   │  │   Browser    │  │   Codebase   │    │
│  │              │  │   Control    │  │   Mapping    │    │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘    │
│         │                  │                  │            │
│  ┌──────┴──────────────────┴──────────────────┴───────┐   │
│  │              Core Services                          │   │
│  └──────┬──────────────┬──────────────┬────────────────┘   │
│         │              │              │                    │
│  ┌──────┴─────┐  ┌─────┴──────┐  ┌───┴────────┐          │
│  │   File     │  │   Shell    │  │  Security  │          │
│  │  System    │  │ Execution  │  │  Manager   │          │
│  └────────────┘  └────────────┘  └────────────┘          │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Data Flow

### Typical User Request Flow

1. **User Input** → Plan Mode
2. **Plan Generation** → Codebase Mapping (analyze relevant files)
3. **Option Presentation** → User Selection
4. **Execution Phase**:
   - File System Tools (read/write/edit files)
   - Shell Execution (run commands)
   - Browser Control (web interactions if needed)
5. **Result Validation** → User Feedback

### Caching Strategy

```
.helix.cache/
├── codebase/
│   ├── <hash>.json     # Codebase maps
│   └── metadata.json   # Cache metadata
├── files/
│   └── <hash>          # Cached file contents
└── browser/
    └── screenshots/    # Browser screenshots
```

## Configuration

Each module supports configuration through:
- Environment variables
- Configuration files (YAML/JSON)
- Runtime options
- Sensible defaults

Example configuration structure:
```go
type Config struct {
    FileSystem    *FileSystemConfig
    Shell         *ShellConfig
    Browser       *BrowserConfig
    CodebaseMap   *CodebaseMapConfig
    PlanMode      *PlanModeConfig
}
```

## Dependencies

### External Libraries
- **chromedp**: Browser automation via Chrome DevTools Protocol
- **go-tree-sitter**: Tree-sitter bindings for Go
- **golang-lru**: LRU cache implementation
- **uuid**: UUID generation
- **fsnotify**: File system event notifications (future)

### Language Parsers
- tree-sitter-go
- tree-sitter-javascript
- tree-sitter-typescript
- tree-sitter-python
- tree-sitter-rust
- (30+ total languages)

## Standards

### Code Quality
- Go conventions and idioms
- Comprehensive documentation
- Type safety
- Interface-driven design
- Dependency injection

### Documentation
- GoDoc comments for all public APIs
- Design documents for complex features
- README files for modules
- Example code and tutorials

### Testing
- Minimum 80% code coverage
- Table-driven tests
- Integration test suites
- Benchmark tests for critical paths
- Security-focused tests

## References

### Inspiration Sources
1. **Cline** - VS Code extension with excellent file tools and browser control
2. **Aider** - Python-based AI coding assistant with great codebase mapping
3. **Plandex** - Go-based AI coding tool with tree-sitter integration
4. **Qwen Code** - Advanced file operations and caching strategies

### Additional Resources
- [Tree-sitter Documentation](https://tree-sitter.github.io/tree-sitter/)
- [Chrome DevTools Protocol](https://chromedevtools.github.io/devtools-protocol/)
- [Go Concurrency Patterns](https://go.dev/blog/pipelines)
- [Go Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)

## Contributing

When adding new technical designs:
1. Follow the existing document structure
2. Include ASCII architecture diagrams
3. Define clear interfaces in Go
4. Provide implementation examples
5. Include comprehensive testing strategies
6. Reference source materials
7. List future enhancements

## License

All design documents are part of the HelixCode project and follow the project's license.

---

**Last Updated**: November 2025

**Version**: 1.0.0

**Status**: Ready for Implementation
