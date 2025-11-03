# HelixCode - Distributed AI Development Platform

## Project Overview

HelixCode is an enterprise-grade distributed AI development platform that enables intelligent task division, work preservation, and cross-platform development workflows. Built with Go and designed for scalability, HelixCode provides a robust foundation for distributed computing with automatic checkpointing, rollback functionality, and real-time monitoring.

The project is fully completed with 5 implementation phases:
- **Phase 1**: Foundation (Database schema, authentication, worker management)
- **Phase 2**: Core Services (Task division, LLM integration, MCP protocol)
- **Phase 3**: Workflows (Project management, development workflows)
- **Phase 4**: LLM Integration (Hardware detection, model management, CLI)
- **Phase 5**: Advanced Features (SSH worker pool, advanced LLM tooling)

Key technologies include Go 1.24+, PostgreSQL, Redis, Gin framework, and multiple LLM providers integration (Llama.cpp, Ollama, OpenAI).

## Architecture

The platform consists of:
- **API Layer**: REST + WebSocket + MCP
- **Core Services**: Authentication, worker management, task management, LLM providers
- **Database Layer**: PostgreSQL + Redis
- **Distributed Workers**: Cross-platform support
- **Multi-Client Interfaces**: CLI, TUI, REST, Mobile

## Building and Running

### Prerequisites
- Go 1.24.0+
- PostgreSQL 15+
- Redis 7+ (optional)

### Build Commands
```bash
cd HelixCode

# Setup dependencies
make setup-deps

# Generate logo assets
make logo-assets

# Build the application
make build

# Run all tests
make test

# Format code
make fmt

# Lint code
make lint

# Clean build artifacts
make clean

# Build for production with cross-platform support
make prod

# Run development server
make dev
```

### Manual Build and Execution
```bash
# Build server
go build -o bin/helixcode ./cmd/server

# Build CLI
go build -o bin/helixcode-cli ./cmd/cli

# Run server
./bin/helixcode

# Run with specific config
./bin/helixcode --config config/prod/config.yaml
```

### Environment Variables
The application uses environment variables with `HELIX_` prefix:
- `HELIX_DATABASE_PASSWORD` - Database password
- `HELIX_REDIS_PASSWORD` - Redis password
- `HELIX_AUTH_JWT_SECRET` - JWT secret
- `HELIX_CONFIG` - Custom config file path

### CLI Usage
```bash
# Interactive mode
./bin/helixcode-cli

# List workers
./bin/helixcode-cli --list-workers

# Add a worker
./bin/helixcode-cli --worker worker-host --user helix --key ~/.ssh/id_rsa

# Generate with LLM
./bin/helixcode-cli --prompt "Hello world" --model llama-3-8b

# Health check
./bin/helixcode-cli --health
```

## Development Conventions

### Go Code Structure
The codebase follows Go best practices with a well-organized internal structure:

- `cmd/` - Main applications (server and CLI)
- `internal/` - Private application code organized by domain:
  - `auth/` - Authentication system
  - `config/` - Configuration management
  - `database/` - Database layer
  - `hardware/` - Hardware detection
  - `llm/` - LLM providers and reasoning
  - `logo/` - Logo processing & assets
  - `mcp/` - MCP protocol implementation
  - `notification/` - Multi-channel notifications
  - `project/` - Project management
  - `redis/` - Redis utilities
  - `server/` - HTTP server & API
  - `session/` - Session management
  - `task/` - Task management & checkpoints
  - `theme/` - Theme management
  - `worker/` - Worker pool management
  - `workflow/` - Workflow execution
- `pkg/` - Shared libraries (public)
- `shared/` - Shared code for mobile bindings
- `scripts/` - Build and utility scripts
- `test/` - Test-specific code

### Code Style
- Go idiomatic code with clear function and variable names
- Structured configuration using Viper for environment and file configuration
- Proper error handling with descriptive error messages
- Comprehensive logging with structured logging approach
- Dependency injection for better testability

### Testing Practices
- Test files follow Go convention with `_test.go` suffix
- Uses `testify` package for assertions and require functions
- Comprehensive test coverage with unit, integration, and end-to-end tests
- Test helpers for temporary directories and environment setup
- Table-driven tests for multiple scenarios
- Mock implementations where needed for testing

### Configuration Management
- Centralized configuration using Viper
- Environment variable support with `HELIX_` prefix
- YAML configuration files with validation
- Default values for all configuration options
- Secure defaults (e.g., requires non-default JWT secret)

### Database Schema
- PostgreSQL database with 11 core tables:
  - `users`: User accounts and authentication
  - `workers`: Distributed worker nodes with SSH config
  - `tasks`: Task management with checkpoints and dependencies
  - `projects`: Project lifecycle management
  - `sessions`: Development sessions and context
  - `llm_providers`: Configured LLM provider instances
  - `notifications`: Multi-channel notification management

### API Endpoints
- REST API with versioning (v1)
- Standard HTTP status codes
- JSON request/response format
- Authentication via JWT tokens
- Comprehensive error responses with error codes

## Mobile Support
- iOS framework generation using gomobile
- Android AAR generation using gomobile
- Shared code in `pkg/mobile-core` for cross-platform functionality

## OS Support
- Linux, macOS, Windows
- Aurora OS and Symphony OS clients
- Cross-platform SSH worker management

## Documentation Files
- Architecture Overview
- Development Guide  
- User Guide
- API Documentation
- Phase implementation summaries (2, 4, 5)