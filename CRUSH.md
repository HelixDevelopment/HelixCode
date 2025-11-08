# CRUSH.md - HelixCode Development Guide

This document contains essential information for agents working with the HelixCode distributed AI development platform.

## Project Overview

HelixCode is an enterprise-grade distributed AI development platform built in Go that enables intelligent task division, work preservation, and cross-platform development workflows. It features:

- **Distributed Computing**: SSH-based worker pools with automatic management and health monitoring
- **Multi-Provider LLM Integration**: Support for Llama.cpp, Ollama, OpenAI, Anthropic, Gemini, Qwen, xAI, OpenRouter, GitHub Copilot, and more
- **Development Workflows**: Automated planning, building, testing, refactoring, debugging, and deployment workflows
- **Task Management**: Intelligent task division with dependency tracking, checkpointing, and automatic rollback
- **MCP Protocol**: Full Model Context Protocol implementation with multi-transport support
- **Multi-Client Architecture**: REST API, CLI, Terminal UI, WebSocket, mobile, and specialized platform support

## Project Structure

```
HelixCode/
├── cmd/                      # Application entry points
│   ├── server/              # Main HTTP server
│   ├── cli/                 # CLI client
│   ├── tui/                 # Terminal UI
│   └── desktop/             # Desktop client
├── internal/                # Internal packages (not importable externally)
│   ├── auth/                # Authentication & authorization
│   ├── worker/              # Worker pool & SSH management
│   ├── task/                # Task management & checkpoints
│   ├── llm/                 # LLM provider implementations
│   ├── mcp/                 # MCP protocol
│   ├── workflow/            # Workflow engine
│   ├── project/             # Project management
│   ├── session/             # Session tracking
│   ├── notification/        # Notification channels
│   ├── hardware/            # Hardware detection
│   ├── server/              # HTTP server & routes
│   ├── database/            # Database layer
│   ├── redis/               # Redis client
│   ├── config/              # Configuration management
│   ├── tools/               # Development tools
│   ├── editor/              # Code editing interfaces
│   ├── agent/               # AI agent coordination
│   ├── context/             # Context building
│   ├── focus/               # Focus management
│   ├── hooks/               # System hooks
│   ├── memory/              # Memory management
│   ├── persistence/         # Data persistence
│   ├── repomap/             # Repository mapping
│   ├── rules/               # Rule system
│   ├── template/            # Template system
│   ├── discovery/           # Service discovery
│   ├── event/               # Event bus
│   ├── commands/            # Command system
│   └── logo/                # Logo processing
├── applications/            # Platform-specific applications
│   ├── desktop/             # Desktop app (Fyne)
│   ├── terminal-ui/         # TUI app (tview)
│   ├── ios/                 # iOS app
│   ├── android/             # Android app
│   ├── aurora-os/           # Aurora OS client
│   └── harmony-os/          # Harmony OS client
├── config/                  # Configuration files
├── scripts/                 # Build and utility scripts
├── tests/                   # Test suites and frameworks
├── docker/                  # Docker configurations
├── docs/                    # Documentation
├── examples/                # Example usage
├── assets/                  # Static assets
├── shared/                  # Shared mobile code
└── mobile/                  # Mobile-specific code
```

## Essential Build Commands

### Development

```bash
cd HelixCode

# Build main application
make build                    # Builds to bin/helixcode

# Development server
make dev                      # Build and run with dev config

# Generate assets (logo, themes)
make logo-assets

# Setup development environment
make dev-setup                # Install dependencies and setup
```

### Testing

```bash
# Run all tests
make test

# Run comprehensive test suite
./scripts/run-tests.sh all

# Run specific test types
./scripts/run-tests.sh unit           # Unit tests only
./scripts/run-tests.sh integration    # Integration tests only
./scripts/run-tests.sh e2e           # End-to-end tests only
./scripts/run-tests.sh coverage       # Generate coverage report

# Performance tests
./scripts/run-tests.sh performance

# Test specific packages
go test -v ./internal/auth/
go test -v ./internal/worker/
go test -cover ./...
```

### Code Quality

```bash
# Format code
make fmt

# Lint code
make lint

# Clean build artifacts
make clean
```

### Production Builds

```bash
# Cross-platform builds
make prod                     # Build for Linux, macOS, Windows

# Mobile bindings
make mobile-ios               # Build iOS framework
make mobile-android           # Build Android AAR
make mobile                   # Build all mobile bindings

# Specialized platforms
make aurora-os                # Build Aurora OS client
make harmony-os               # Build Harmony OS client
make aurora-harmony           # Build both specialized platforms

# Full release build
make release                  # Clean + assets + docs + build + test
```

## Configuration Management

### Primary Configuration

Main configuration at `config/config.yaml` with environment variable overrides:

```yaml
server:
  address: "0.0.0.0"
  port: 8080

database:
  host: "localhost"
  port: 5432
  dbname: "helixcode"
  # Password via HELIX_DATABASE_PASSWORD

auth:
  # JWT secret via HELIX_AUTH_JWT_SECRET
  token_expiry: 86400
  session_expiry: 604800

workers:
  health_check_interval: 30
  max_concurrent_tasks: 10

tasks:
  max_retries: 3
  checkpoint_interval: 300

llm:
  default_provider: "local"
  max_tokens: 4096
  temperature: 0.7
```

### Environment Variables

**Required for Production:**
```bash
export HELIX_DATABASE_PASSWORD="your_secure_password"
export HELIX_AUTH_JWT_SECRET="your_jwt_secret"
export HELIX_REDIS_PASSWORD="your_redis_password"  # if Redis enabled
```

**LLM Provider Keys:**
```bash
# Free providers (optional for higher limits)
export GITHUB_TOKEN="ghp_your_github_token"         # GitHub Copilot
export OPENROUTER_API_KEY="sk-or-your-key"          # OpenRouter
export XAI_API_KEY="xai-your-key"                    # XAI/Grok

# Premium providers
export ANTHROPIC_API_KEY="sk-ant-your-key"          # Anthropic Claude
export GEMINI_API_KEY="your-gemini-key"              # Google Gemini
export OPENAI_API_KEY="sk-your-openai-key"           # OpenAI
```

**Notification Channels:**
```bash
export HELIX_SLACK_WEBHOOK_URL="https://hooks.slack.com/..."
export HELIX_TELEGRAM_BOT_TOKEN="your_bot_token"
export HELIX_TELEGRAM_CHAT_ID="your_chat_id"
export HELIX_EMAIL_SMTP_SERVER="smtp.gmail.com"
export HELIX_EMAIL_USERNAME="your_email"
export HELIX_EMAIL_PASSWORD="your_app_password"
```

## Code Style & Patterns

### Go Conventions

**Language Version:** Go 1.24.0 (toolchain go1.24.9)
**Module:** `dev.helix.code`

**Naming Conventions:**
- **Types**: PascalCase for exported, camelCase for unexported
- **Functions**: PascalCase for exported, camelCase for unexported
- **Variables**: camelCase, descriptive names
- **Constants**: PascalCase, grouped by functionality
- **Interfaces**: Simple capability names (Provider, Repository, Manager)

**Import Organization:**
1. Standard library imports
2. Third-party imports
3. Internal imports (dev.helix.code/internal/...)
4. Blank line between groups

**Error Handling:**
- Return errors with context using `fmt.Errorf("failed to X: %v", err)`
- Check errors immediately after operations
- Use package-level error variables: `ErrInvalidCredentials`
- Structured error responses in HTTP handlers

### Architectural Patterns

**Interface-Driven Design:**
- Core interfaces define contracts (Provider, Repository, Manager)
- Multiple implementations per interface (various LLM providers)
- Factory pattern for provider creation
- Easy mocking for unit tests

**Dependency Injection:**
- Constructor injection: `NewService(config Config, repo Repository)`
- Configuration-based service initialization
- Clean separation between layers

**Manager Pattern:**
- Centralized managers (TaskManager, WorkerManager, ProviderManager)
- Thread-safe operations with `sync.RWMutex`
- Encapsulate complex business logic

**Repository Pattern:**
- Data access abstraction via interfaces
- Database-agnostic implementations
- Redis caching integrated transparently

## Testing Patterns

### Test Structure

**File Organization:**
- Test files alongside source: `auth_test.go` next to `auth.go`
- Comprehensive test coverage required
- Separate test packages for complex scenarios

**Test Patterns:**
- Table-driven tests for multiple scenarios
- Subtests with `t.Run("name", func(t *testing.T) {...})`
- Setup/teardown with helper functions
- Mock implementations using `testify/mock`

**Assertions:**
```go
require.NoError(t, err)              # Critical operations
assert.Equal(t, expected, actual)     # Value comparisons
assert.Contains(t, result, substring) # String contains
```

**Mock Usage:**
- Interface-based mocking for isolation
- `mock.Mock` for behavior verification
- Test doubles for external dependencies

### Test Categories

**Unit Tests:**
- Fast, isolated tests for individual functions
- Mock all external dependencies
- Tagged with `unit` build tag

**Integration Tests:**
- Test interaction between components
- Use real databases/Redis in Docker
- Tagged with `integration` build tag

**End-to-End Tests:**
- Full workflow testing
- Docker Compose test environment
- Mock external services (LLM providers)
- Tagged with `e2e` build tag

## Database Setup

### PostgreSQL

```bash
# Create database and user
createdb helixcode
createuser helixcode

# Set password via environment variable
export HELIX_DATABASE_PASSWORD=your_password

# Schema is automatically created by the application
```

### Key Tables
- `users` - User authentication and profiles
- `workers` - Worker node registration and status
- `tasks` - Task management and state
- `projects` - Project metadata and sessions
- `sessions` - User session tracking
- `llm_providers` - LLM provider configurations
- `notifications` - Notification rules and history

## CLI Usage

### Build CLI

```bash
cd HelixCode
go build -o bin/cli ./cmd/cli
```

### Common Commands

```bash
# Interactive mode
./bin/cli

# Worker management
./bin/cli --list-workers
./bin/cli --worker worker-host --user helix --key ~/.ssh/id_rsa

# AI-powered generation
./bin/cli --prompt "Hello world" --model llama-3-8b --max-tokens 1000

# Notifications
./bin/cli --notify "Build complete" --notify-type "success"

# Health check
./bin/cli --health
```

## AI Provider Integration

### Free Providers (No API Key Required)

**XAI (Grok):**
```bash
helixcode llm provider set xai
# Models: grok-3-fast-beta, grok-3-mini-fast-beta, grok-3-beta
```

**OpenRouter:**
```bash
helixcode llm provider set openrouter
# Models: deepseek-r1-free, meta-llama/llama-3.2-3b-instruct:free
```

**GitHub Copilot:**
```bash
export GITHUB_TOKEN="ghp_your_github_token"
helixcode llm provider set copilot
# Models: gpt-4o, claude-3.5-sonnet, claude-3.7-sonnet, o1, gemini-2.0-flash
```

### Premium Providers

**Anthropic Claude (Extended Thinking & Caching):**
```bash
export ANTHROPIC_API_KEY="sk-ant-your-key"
helixcode llm provider set anthropic --model claude-4-sonnet
# Features: Extended thinking, prompt caching, 200K context, 50K output
```

**Google Gemini (2M Token Context):**
```bash
export GEMINI_API_KEY="your-gemini-key"
helixcode llm provider set gemini --model gemini-2.5-pro
# Features: 2M token context, multimodal, function calling
```

## Important Gotchas & Patterns

### SSH Worker Auto-Install
- When adding workers via SSH, the system automatically installs the Helix CLI binary on remote machines
- Requires SSH key-based authentication
- Workers are health-checked every 30s by default

### Task Checkpointing
- Long-running tasks automatically checkpoint at configured intervals (default 300s)
- Enables work preservation and recovery from failures
- Checkpoints stored in PostgreSQL for persistence

### Provider Fallback
- LLM requests can fall back to alternative providers if the primary fails
- Configurable provider priority and retry logic
- Automatic rate limiting and quota management

### Session Context
- Development sessions maintain context across interactions for continuity
- Context stored in Redis for fast access
- Automatic cleanup of expired sessions

### MCP Protocol
- Supports both stdio and SSE transports for Model Context Protocol
- Tool integration and execution through standardized interface
- Real-time bidirectional communication

### Mobile & Specialized Platforms
- Gomobile bindings for iOS/Android applications
- Aurora OS (Russian platform) and Harmony OS (Chinese platform) support
- Cross-platform UI with Fyne framework

## Development Workflow

### Before Making Changes
1. Read existing code in the relevant package
2. Check for similar patterns in other packages
3. Run tests to ensure baseline functionality
4. Make incremental changes with tests

### After Changes
1. Run `make test` to ensure all tests pass
2. Run `make fmt` and `make lint` for code quality
3. Test specific functionality with targeted tests
4. Update documentation if needed

### Commit Process
1. Use conventional commit messages
2. Include tests for new functionality
3. Update relevant documentation
4. Ensure CI/CD pipeline would pass

## Dependencies

### Core Dependencies
- `github.com/gin-gonic/gin`: HTTP framework
- `github.com/jackc/pgx/v5`: PostgreSQL driver
- `github.com/golang-jwt/jwt/v4`: JWT authentication
- `github.com/spf13/viper`: Configuration management
- `github.com/gorilla/websocket`: WebSocket support
- `golang.org/x/crypto/ssh`: SSH client for workers
- `github.com/google/uuid`: UUID generation
- `github.com/stretchr/testify`: Testing framework

### UI Frameworks
- `fyne.io/fyne/v2`: Desktop GUI framework
- `github.com/rivo/tview`: Terminal UI framework

### External Integrations
- AWS SDK v2: Bedrock integration
- Azure SDK: Azure AI services
- Chromedp: Browser automation
- Go-Redis: Redis client

## Cross-Platform Support

**Standard Platforms:**
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

**Mobile Platforms:**
- iOS (via gomobile)
- Android (via gomobile)

**Specialized Platforms:**
- Aurora OS (Russian platform)
- Harmony OS (Chinese platform)

## Code Generation & Assets

### Logo Assets

```bash
# Generate logo assets and themes
make logo-assets

# Manual generation
cd scripts/logo && go run generate_assets.go
```

This extracts colors and creates themed variations of the logo for different platforms and use cases.

## Monitoring & Observability

### Health Checks
- Application health endpoint: `/health`
- Worker health monitoring
- Database connectivity checks
- External service availability

### Metrics
- Prometheus metrics endpoint: `/metrics`
- Task execution statistics
- Worker performance metrics
- LLM provider response times
- Resource usage tracking

### Logging
- Structured logging with levels
- Contextual information in logs
- Integration with external logging systems
- Security event logging

## Security Considerations

- JWT-based authentication with configurable expiry
- Password hashing with bcrypt
- SSH key-based worker authentication
- Environment variable configuration (no secrets in code)
- Rate limiting on API endpoints
- Input validation and sanitization
- HTTPS enforcement in production

## Performance Optimization

### Database
- Connection pooling with pgx
- Query optimization and indexing
- Prepared statements for repeated queries
- Read replica support for scaling

### Caching
- Redis for session and task state
- LLM response caching with TTL
- Static asset caching
- Database query result caching

### Concurrency
- Goroutine pools for task execution
- Channel-based communication
- Mutex protection for shared state
- Context-based cancellation

This guide provides the essential information needed for effective development in the HelixCode codebase. Follow these patterns and conventions to maintain consistency and quality across the platform.