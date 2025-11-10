# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

HelixCode is an enterprise-grade distributed AI development platform built in Go that enables intelligent task division, work preservation, and cross-platform development workflows. It features:

- **Distributed Computing**: SSH-based worker pools with automatic management and health monitoring
- **Multi-Provider LLM Integration**: Support for local providers (Llama.cpp, Ollama, vLLM, LocalAI, LM Studio, Jan, GPT4All, etc.) and cloud APIs (OpenAI, Anthropic, Gemini, Vertex AI, Qwen, xAI, OpenRouter, Copilot, Bedrock, Azure, Groq)
- **Development Workflows**: Automated planning, building, testing, refactoring, debugging, and deployment workflows
- **Task Management**: Intelligent task division with dependency tracking, checkpointing, and automatic rollback
- **MCP Protocol**: Full Model Context Protocol implementation with multi-transport support
- **Multi-Client Architecture**: REST API, CLI, Terminal UI, Desktop, WebSocket, and mobile framework support
- **Memory Systems**: Integration with Mem0, Zep, Memonto, and BaseAI for long-term memory and context management

## Essential Build Commands

**IMPORTANT**: All build commands must be run from the `HelixCode/` subdirectory (not the repository root).

```bash
# Navigate to the Go module directory
cd HelixCode

# Build the main server binary
make build                    # Builds to bin/helixcode

# Testing
make test                     # Run all tests with go test -v ./...
go test -v ./internal/auth    # Test specific package
go test -cover ./...          # Run with coverage

# Code quality
make fmt                      # Format code with go fmt
make lint                     # Lint with golangci-lint (if installed)

# Development
make dev                      # Build and run with config/dev/config.yaml
make clean                    # Clean build artifacts (bin/, dist/, coverage.out)

# Production builds (cross-platform)
make prod                     # Build for Linux, macOS, Windows

# Platform-specific builds
make aurora-os                # Build Aurora OS client to bin/aurora-os
make harmony-os               # Build Harmony OS client to bin/harmony-os

# Mobile builds (requires gomobile)
make mobile-init              # Initialize gomobile
make mobile-ios               # Build iOS framework (HelixCore.xcframework)
make mobile-android           # Build Android AAR (mobile.aar)
make mobile                   # Build all mobile bindings

# Assets and documentation
make logo-assets              # Generate logo assets from scripts/logo/generate_assets.go
make sync-manual              # Sync user manual to website
make manual-html              # Convert manual to HTML (requires pandoc)
make docs                     # Build all documentation
make release                  # Full release: clean, logo-assets, docs, build, test
```

## Architecture Overview

### Core Service Layers

**Application Entry Points** (`cmd/`):
- `cmd/server`: HTTP server with REST API and WebSocket support
- `cmd/cli`: Command-line interface client
- Additional applications in `applications/`: terminal-ui, desktop, aurora-os, harmony-os

**Service Layer** (`internal/`):
- `internal/auth`: JWT-based authentication with session management
- `internal/worker`: SSH-based distributed worker pool with auto-installation
- `internal/task`: Task management with checkpointing, dependencies, and queue
- `internal/llm`: Multi-provider LLM integration with unified Provider interface
- `internal/project`: Project lifecycle and session management
- `internal/workflow`: Workflow execution engine with step dependencies
- `internal/notification`: Multi-channel notifications (Slack, Discord, Email, Telegram)
- `internal/mcp`: Model Context Protocol implementation
- `internal/server`: HTTP server, routing, and API handlers
- `internal/memory`: Long-term memory integration (Mem0, Zep, Memonto, BaseAI)
- `internal/agent`: Multi-agent orchestration and coordination
- `internal/tools`: Tool calling and code analysis capabilities
- `internal/session`: Session tracking and context management
- `internal/config`: Configuration management with Viper

**Data Layer**:
- `internal/database`: PostgreSQL for persistent storage
- `internal/redis`: Redis for caching and real-time state (optional, configurable)

**Cross-Platform**:
- `shared/mobile-core`: Shared code for mobile bindings (iOS, Android)

### Key Architecture Patterns

**Task Distribution**: Tasks are intelligently divided based on complexity and worker capabilities. The `task.Manager` maintains a queue with priority-based scheduling. Each task has:
- Type (planning, building, testing, refactoring, debugging, etc.)
- Priority levels (low, normal, high, critical)
- Status tracking (pending, assigned, running, completed, failed)
- Checkpoint system for work preservation
- Dependency resolution

**Worker Management**: The `worker.SSHWorkerPool` manages distributed workers over SSH. Features include:
- Automatic Helix CLI installation on new workers
- Health monitoring with configurable intervals
- Resource tracking (CPU, memory, GPU)
- Capability-based task assignment
- Connection pooling and retry logic

**LLM Provider Abstraction**: Unified `llm.Provider` interface supports multiple backends:
- Local inference servers: Llama.cpp, Ollama, vLLM, LocalAI, FastChat, text-generation-webui, LM Studio, Jan, GPT4All, KoboldAI, TabbyAPI, MLX, MistralRS
- Cloud APIs: OpenAI, Anthropic, Gemini, Vertex AI, Qwen, xAI, Groq
- Enterprise: Azure OpenAI, AWS Bedrock
- Aggregators: OpenRouter, GitHub Copilot
- All providers implement a common interface with Generate/GenerateStream methods
- Provider selection strategies: performance, cost, availability, round-robin with automatic fallback

**Workflow Execution**: Workflows consist of typed steps (analysis, generation, execution, validation) with actions (analyze_code, generate_code, run_tests, etc.). Steps can have dependencies, forming a DAG that executes in proper order.

## Configuration

Primary configuration at `HelixCode/config/config.yaml`. The system uses Viper for configuration management with environment variable overrides.

**Configuration File Locations** (searched in order):
1. Path specified via command-line flag
2. `./config/config.yaml` (relative to HelixCode/ directory)
3. `./config.yaml`
4. `$HOME/.config/helixcode/config.yaml`
5. `/etc/helixcode/config.yaml`

**Key Configuration Sections**:
- `server`: HTTP server settings (address, port, timeouts)
- `database`: PostgreSQL connection settings
- `redis`: Redis connection settings (optional, can be disabled via `enabled: false`)
- `auth`: JWT authentication settings
- `workers`: Worker pool health checks and concurrency
- `tasks`: Task retry and checkpoint intervals
- `llm`: LLM provider configuration and selection strategy
- `llm.providers`: Individual provider configurations (see config.yaml for full list)
- `notifications`: Multi-channel notification rules and channel configs
- `logging`: Log level, format, and output

**Critical Environment Variables** (override config file):
- `HELIX_AUTH_JWT_SECRET`: JWT signing secret (required for auth)
- `HELIX_DATABASE_PASSWORD`: PostgreSQL password
- `HELIX_DATABASE_HOST`: PostgreSQL host (default: localhost)
- `HELIX_DATABASE_PORT`: PostgreSQL port (default: 5432)
- `HELIX_DATABASE_USER`: PostgreSQL user (default: helixcode)
- `HELIX_DATABASE_NAME`: PostgreSQL database name (default: helixcode)
- `HELIX_REDIS_PASSWORD`: Redis password (if Redis enabled)
- `HELIX_REDIS_HOST`: Redis host (default: localhost)
- `HELIX_REDIS_PORT`: Redis port (default: 6379)

**Notification Channel Environment Variables** (optional):
- `HELIX_SLACK_WEBHOOK_URL`: Slack webhook for notifications
- `HELIX_TELEGRAM_BOT_TOKEN`, `HELIX_TELEGRAM_CHAT_ID`: Telegram bot configuration
- `HELIX_DISCORD_WEBHOOK_URL`: Discord webhook for notifications
- `HELIX_EMAIL_SMTP_SERVER`, `HELIX_EMAIL_SMTP_PORT`, `HELIX_EMAIL_USERNAME`, `HELIX_EMAIL_PASSWORD`, `HELIX_EMAIL_FROM`: SMTP email configuration

## Database Setup

```bash
# Create database and user
createdb helixcode
createuser helixcode

# Set password via environment variable
export HELIX_DATABASE_PASSWORD=your_password

# Schema is automatically created by the application
# Tables: users, workers, tasks, projects, sessions, llm_providers, notifications, etc.
```

## CLI Usage

The CLI client is at `cmd/cli/main.go`:

```bash
# Build CLI
cd HelixCode
go build -o bin/cli ./cmd/cli

# Interactive mode
./bin/cli

# List workers
./bin/cli --list-workers

# Add SSH worker (auto-installs Helix CLI)
./bin/cli --worker worker-host --user helix --key ~/.ssh/id_rsa

# Generate with LLM
./bin/cli --prompt "Hello world" --model llama-3-8b --max-tokens 1000

# Send notifications
./bin/cli --notify "Build complete" --notify-type "success"

# Health check
./bin/cli --health
```

## Repository Structure

**IMPORTANT**: This repository has a nested structure. The repository root contains documentation and example projects, while the main Go application is in the `HelixCode/` subdirectory.

```
/ (repository root)
├── HelixCode/                    # Main Go application (go.mod is here)
│   ├── cmd/                      # Application entry points
│   │   ├── server/               # Main HTTP server
│   │   └── cli/                  # CLI client
│   ├── applications/             # Platform-specific apps
│   │   ├── terminal-ui/          # Terminal UI (TUI)
│   │   ├── desktop/              # Desktop GUI
│   │   ├── aurora-os/            # Aurora OS client
│   │   └── harmony-os/           # Harmony OS client
│   ├── internal/                 # Internal packages (not importable externally)
│   │   ├── auth/                 # Authentication & authorization
│   │   ├── worker/               # Worker pool & SSH management
│   │   ├── task/                 # Task management & checkpoints
│   │   ├── llm/                  # LLM provider implementations
│   │   ├── mcp/                  # MCP protocol
│   │   ├── workflow/             # Workflow engine
│   │   ├── project/              # Project management
│   │   ├── session/              # Session tracking
│   │   ├── memory/               # Long-term memory systems
│   │   ├── agent/                # Multi-agent coordination
│   │   ├── tools/                # Tool calling capabilities
│   │   ├── notification/         # Notification channels
│   │   ├── server/               # HTTP server & routes
│   │   ├── database/             # Database layer
│   │   ├── redis/                # Redis client
│   │   ├── config/               # Configuration management
│   │   └── [... other services]
│   ├── shared/                   # Shared code
│   │   └── mobile-core/          # Mobile platform bindings
│   ├── config/                   # Configuration files
│   ├── scripts/                  # Build and utility scripts
│   ├── docs/                     # Technical documentation
│   ├── tests/                    # Integration and E2E tests
│   ├── go.mod                    # Go module definition
│   ├── Makefile                  # Build system
│   └── README.md                 # HelixCode-specific README
├── Example_Projects/             # Reference implementations
├── Dependencies/                 # Git submodules (LLama_CPP, etc.)
├── Documentation/                # Project-wide documentation
├── Specification/                # Technical specifications
├── Implementation_Guide/         # Implementation guides
├── Design/                       # Design assets
├── README.md                     # Repository overview
└── CLAUDE.md                     # This file
```

**Working Directory**: All Go commands and make targets must be executed from `HelixCode/` subdirectory, not the repository root.

## Development Workflows

The system implements automated workflows for different development phases:

**Planning Mode**: Analyzes requirements, creates technical specifications, breaks down into tasks
**Building Mode**: Code generation, dependency management, integration
**Testing Mode**: Unit tests, integration tests, test execution
**Refactoring Mode**: Code analysis, optimization, restructuring
**Debugging Mode**: Error analysis, root cause identification, fixes
**Deployment Mode**: Build, package, deploy to targets

Each workflow is defined with typed steps and dependencies in `internal/workflow`.

## Testing Patterns

- Unit tests are alongside source files (e.g., `manager_test.go` next to `manager.go`)
- Use testify for assertions: `github.com/stretchr/testify`
- Mock interfaces for database and external services
- Test with `go test -v ./...` from the `HelixCode/` directory

## Module and Dependencies

**Module name**: `dev.helix.code`
**Go version**: 1.25.2

**Core dependencies** (check `go.mod` for complete list):
- `github.com/google/uuid`: UUID generation
- `github.com/pkg/errors`: Error handling
- `gopkg.in/yaml.v2`: YAML parsing
- `github.com/spf13/viper`: Configuration management (referenced in code)

**Note**: The project uses standard library packages extensively. Additional dependencies for HTTP frameworks, database drivers, JWT, WebSocket, and SSH are imported but may need to be added via `go get` or `go mod tidy` if not already in go.mod.

## Code Generation

Logo assets are auto-generated before build:
```bash
make logo-assets    # Generates from scripts/logo/generate_assets.go
```

This extracts colors and creates themed variations of the logo.

## Cross-Platform Support

The platform supports multiple deployment targets:
- **Standard Desktop/Server**: Linux, macOS, Windows (via `make prod`)
- **Mobile**: iOS (xcframework) and Android (AAR) via gomobile bindings (`make mobile`)
- **Specialized Platforms**:
  - Aurora OS (Russian mobile platform, via `make aurora-os`)
  - Harmony OS (Chinese ecosystem, via `make harmony-os`)
- **Applications**: Terminal UI, Desktop GUI, CLI client (all built from `applications/` and `cmd/`)

## Important Implementation Notes

- **Nested Repository Structure**: The main Go application is in `HelixCode/` subdirectory. Always `cd HelixCode` before running build/test commands.
- **SSH Worker Auto-Install**: When adding workers via SSH, the system automatically installs the Helix CLI binary on remote machines
- **Task Checkpointing**: Long-running tasks automatically checkpoint at configured intervals (default 300s) for work preservation
- **Provider Fallback**: LLM requests can fall back to alternative providers if the primary fails (configurable via `llm.selection.fallback_enabled`)
- **Health Monitoring**: Workers are health-checked every 30s by default; unhealthy workers are removed from the active pool
- **Session Context**: Development sessions maintain context across interactions for continuity
- **MCP Protocol**: Supports both stdio and SSE transports for Model Context Protocol
- **Database Schema Auto-Init**: The server automatically creates database schema on startup via `db.InitializeSchema()`
- **Redis is Optional**: Redis can be disabled by setting `redis.enabled: false` in config; the system will function without it
- **Environment Variables Override Config**: All `HELIX_*` environment variables take precedence over config file values
- **Provider Selection Strategy**: Configurable via `llm.selection.strategy` (performance, cost, availability, round-robin)

## Testing Infrastructure

The project includes multiple test levels:
- **Unit tests**: Alongside source files (`*_test.go`), run with `go test -v ./internal/<package>`
- **Integration tests**: In `tests/` directory
- **E2E tests**: In `test/e2e/` directory
- **Test helpers**: Mock implementations in `internal/mocks/`
- **Test scripts**: `run_tests.sh`, `run_integration_tests.sh`, `run_all_tests.sh`
- **Coverage**: Generate with `go test -cover ./...` or check `coverage.out`
