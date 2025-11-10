# HelixCode Agent Guidelines

## Build/Test Commands
- **Build**: `make build` or `go build -ldflags="-X main.version=1.0.0 -X main.buildTime=$(date +%Y-%m-%d_%H:%M:%S) -X main.gitCommit=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")" -o bin/helixcode ./cmd/server`
- **Test all**: `make test` or `go test -v ./...`
- **Test single**: `go test -v -run TestName ./path/to/package`
- **Test types**: `./scripts/run-tests.sh unit|integration|e2e|coverage|all`
- **Lint**: `make lint` or `golangci-lint run ./...`
- **Format**: `make fmt` or `go fmt ./...`

## Code Style (Go 1.24.0, module: dev.helix.code)
- **Imports**: std lib → third-party → internal (blank lines between groups)
- **Naming**: PascalCase exported, camelCase unexported; descriptive variables
- **Errors**: `fmt.Errorf("failed to X: %v", err)` with immediate checks; use `log` not `fmt.Printf`
- **Types**: Custom types for enums; meaningful struct names; comments on exported fields
- **Deps**: `go mod tidy`; key: gin, pgx, testify, viper, jwt