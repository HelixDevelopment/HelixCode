# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

HelixCode is an enterprise-grade distributed AI development platform built in Go that enables intelligent task division, work preservation, and cross-platform development workflows. It features:

- **Distributed Computing**: SSH-based worker pools with automatic management and health monitoring
- **Multi-Provider LLM Integration**: Support for Llama.cpp, Ollama, OpenAI, Anthropic, Gemini, Qwen, xAI, OpenRouter, and Copilot
- **Development Workflows**: Automated planning, building, testing, refactoring, debugging, and deployment workflows
- **Task Management**: Intelligent task division with dependency tracking, checkpointing, and automatic rollback
- **MCP Protocol**: Full Model Context Protocol implementation with multi-transport support
- **Multi-Client Architecture**: REST API, CLI, Terminal UI, WebSocket, and mobile framework support

## Essential Build Commands

```bash
# Build the main server binary
cd HelixCode
make build                    # Builds to bin/helixcode

# Testing
make test                     # Run all tests
go test -v ./internal/auth    # Test specific package
go test -cover ./...          # Run with coverage

# Code quality
make fmt                      # Format code with go fmt
make lint                     # Lint with golangci-lint

# Development
make dev                      # Build and run with dev config
make clean                    # Clean build artifacts

# Production builds (cross-platform)
make prod                     # Build for Linux, macOS, Windows

# Mobile/embedded builds
make mobile-ios               # Build iOS framework
make mobile-android           # Build Android AAR
make aurora-os                # Build Aurora OS client
make harmony-os               # Build Harmony OS client
```

## Architecture Overview

### Core Service Layers

**API Layer**:
- REST API with Gin framework at `internal/server`
- WebSocket support for real-time communication
- MCP protocol implementation at `internal/mcp`

**Service Layer**:
- `internal/auth`: JWT-based authentication with session management
- `internal/worker`: SSH-based distributed worker pool with auto-installation
- `internal/task`: Task management with checkpointing, dependencies, and queue
- `internal/llm`: Multi-provider LLM integration with unified interface
- `internal/project`: Project lifecycle and session management
- `internal/workflow`: Workflow execution engine with step dependencies
- `internal/notification`: Multi-channel notifications (Slack, Discord, Email, Telegram)

**Data Layer**:
- `internal/database`: PostgreSQL for persistent storage
- `internal/redis`: Redis for caching and real-time state (optional)

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
- Local models (Llama.cpp, Ollama)
- Cloud APIs (OpenAI, Anthropic, Gemini)
- Chinese providers (Qwen)
- Aggregators (OpenRouter)
- Tool calling and reasoning capabilities

**Workflow Execution**: Workflows consist of typed steps (analysis, generation, execution, validation) with actions (analyze_code, generate_code, run_tests, etc.). Steps can have dependencies, forming a DAG that executes in proper order.

## Configuration

Primary configuration at `HelixCode/config/config.yaml`:

```yaml
server:
  address: "0.0.0.0"
  port: 8080

database:
  host: "localhost"
  port: 5432
  dbname: "helixcode"
  # Password via HELIX_DATABASE_PASSWORD env var

auth:
  # JWT secret via HELIX_AUTH_JWT_SECRET env var
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

**Important Environment Variables**:
- `HELIX_DATABASE_PASSWORD`: PostgreSQL password
- `HELIX_AUTH_JWT_SECRET`: JWT signing secret
- `HELIX_REDIS_PASSWORD`: Redis password (if Redis enabled)

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

## Project Structure Conventions

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
│   └── logo/                # Logo processing & assets
├── config/                  # Configuration files
├── scripts/                 # Build and utility scripts
└── docs/                    # Documentation
```

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

Module name: `dev.helix.code`
Go version: 1.24.0

Key dependencies:
- `github.com/gin-gonic/gin`: HTTP framework
- `github.com/jackc/pgx/v5`: PostgreSQL driver
- `github.com/golang-jwt/jwt/v4`: JWT tokens
- `github.com/spf13/viper`: Configuration
- `github.com/gorilla/websocket`: WebSocket support
- `golang.org/x/crypto/ssh`: SSH client for workers
- `github.com/google/uuid`: UUID generation

## Code Generation

Logo assets are auto-generated before build:
```bash
make logo-assets    # Generates from scripts/logo/generate_assets.go
```

This extracts colors and creates themed variations of the logo.

## Cross-Platform Support

The platform supports:
- **Standard**: Linux, macOS, Windows (via `make prod`)
- **Mobile**: iOS and Android (via gomobile bindings in `make mobile`)
- **Embedded**: Aurora OS (Russian platform via `make aurora-os`) and Harmony OS (Chinese platform via `make harmony-os`)

## Important Notes

- **SSH Worker Auto-Install**: When adding workers via SSH, the system automatically installs the Helix CLI binary on remote machines
- **Task Checkpointing**: Long-running tasks automatically checkpoint at configured intervals (default 300s) for work preservation
- **Provider Fallback**: LLM requests can fall back to alternative providers if the primary fails
- **Health Monitoring**: Workers are health-checked every 30s by default; unhealthy workers are removed from the active pool
- **Session Context**: Development sessions maintain context across interactions for continuity
- **MCP Protocol**: Supports both stdio and SSE transports for Model Context Protocol
