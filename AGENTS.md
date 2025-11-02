# HelixCode Agent Guidelines

## Build Commands
- **Build application**: `make build` or `go build -ldflags="-X main.version=1.0.0 -X main.buildTime=$(date +%Y-%m-%d_%H:%M:%S) -X main.gitCommit=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")" -o bin/helixcode ./cmd/server`
- **Build for production**: `make prod`
- **Clean build artifacts**: `make clean`

## Test Commands
- **Run all tests**: `make test` or `go test -v ./...`
- **Run single test**: `go test -v -run TestName ./path/to/package`
- **Run with coverage**: `go test -cover ./...`
- **Run specific test types**: `./scripts/run-tests.sh unit|integration|e2e|all`
- **Run comprehensive tests**: `./scripts/run-all-tests.sh all`
- **Generate coverage report**: `./scripts/run-tests.sh coverage`

## Lint & Format Commands
- **Lint code**: `make lint` or `golangci-lint run ./...`
- **Format code**: `make fmt` or `go fmt ./...`

## Code Style Guidelines

### Go Version & Toolchain
- **Go version**: 1.24.0 (toolchain go1.24.9)
- **Module**: `dev.helix.code`

### Imports
- Standard library imports first
- Third-party imports second
- Internal imports last
- Blank line between groups

### Naming Conventions
- **Types**: PascalCase for exported, camelCase for unexported
- **Functions**: PascalCase for exported, camelCase for unexported
- **Variables**: camelCase, descriptive names
- **Constants**: PascalCase, grouped by related functionality
- **Enums**: Custom types with const values (e.g., `TaskType`, `TaskPriority`)

### Error Handling
- Return errors with context using `fmt.Errorf("failed to X: %v", err)`
- Check errors immediately after operations
- Use `log` package for logging, not `fmt.Printf`

### Types & Structs
- Define custom types for enums/constants
- Use meaningful struct field names
- Add comments for exported types and fields
- Group related constants together

### Dependencies
- Use `go mod tidy` to clean dependencies
- Import path: `dev.helix.code/internal/...`
- Key dependencies: gin, pgx, testify, viper, jwt