# E2E Test Bank

A comprehensive collection of end-to-end test cases for the HelixCode platform, organized by category and priority.

## Structure

```
test-bank/
├── core/              # Core functionality tests (auth, API, basic operations)
├── integration/       # Integration tests (LLM providers, databases, external services)
├── distributed/       # Distributed system tests (worker pools, coordination)
├── platform/          # Platform-specific tests (Linux, macOS, Windows)
├── metadata/          # Test metadata and configuration
└── README.md          # This file
```

## Test Categories

### Core Tests (`core/`)
Tests for fundamental platform features that must always work:
- Authentication and authorization
- User management
- Project lifecycle
- Session management
- Basic API endpoints
- Configuration loading
- Database connectivity

**Priority**: CRITICAL
**Execution Frequency**: Every commit
**Est. Time**: 5-10 minutes

### Integration Tests (`integration/`)
Tests for integrations with external services and providers:
- LLM provider integrations (OpenAI, Anthropic, Ollama, etc.)
- Notification services (Slack, Discord, Email, Telegram)
- Database operations
- Redis caching
- MCP protocol
- Worker SSH connections

**Priority**: HIGH
**Execution Frequency**: Every PR
**Est. Time**: 10-20 minutes

### Distributed Tests (`distributed/`)
Tests for distributed system behavior:
- Multi-worker coordination
- Task distribution and scheduling
- Failover and recovery
- Load balancing
- Network partition handling
- Worker pool management

**Priority**: HIGH
**Execution Frequency**: Nightly
**Est. Time**: 15-30 minutes

### Platform Tests (`platform/`)
Tests for platform-specific functionality:
- Linux-specific features
- macOS-specific features
- Windows-specific features
- Aurora OS compatibility
- Harmony OS compatibility
- Mobile framework tests

**Priority**: MEDIUM
**Execution Frequency**: Weekly
**Est. Time**: 20-40 minutes

## Test Metadata

Each test case includes metadata in JSON format:

```json
{
  "id": "TC-001",
  "name": "User Authentication",
  "category": "core",
  "priority": "critical",
  "tags": ["auth", "security", "smoke"],
  "estimated_duration": "5s",
  "dependencies": ["database", "redis"],
  "timeout": "30s",
  "retry_count": 2,
  "platforms": ["linux", "macos", "windows"],
  "description": "Verify user can authenticate with valid credentials",
  "preconditions": [
    "Database is running",
    "User exists in database"
  ],
  "steps": [
    "Send login request with valid credentials",
    "Verify JWT token is returned",
    "Verify token contains correct user ID"
  ],
  "expected_results": [
    "HTTP 200 status code",
    "Valid JWT token returned",
    "Token expires in 24 hours"
  ]
}
```

## Running Tests

### Run All Tests
```bash
cd tests/e2e/orchestrator
./bin/orchestrator run
```

### Run by Category
```bash
./bin/orchestrator run --tags core
./bin/orchestrator run --tags integration
./bin/orchestrator run --tags distributed
```

### Run by Priority
```bash
./bin/orchestrator run --tags critical
./bin/orchestrator run --tags smoke
```

### Run Specific Test
```bash
./bin/orchestrator run --tests TC-001
./bin/orchestrator run --tests TC-001,TC-002,TC-003
```

## Writing New Tests

See the sample tests in each category directory for examples. All tests should:

1. Follow the Go testing conventions
2. Include metadata in the test file
3. Use the validator package for assertions
4. Handle cleanup properly (teardown functions)
5. Be idempotent (can run multiple times)
6. Be isolated (don't depend on test execution order)

Example test template:

```go
package core

import (
    "context"
    "testing"
    "time"
    "dev.helix.code/tests/e2e/orchestrator/pkg"
    "dev.helix.code/tests/e2e/orchestrator/pkg/validator"
)

// TestMetadata defines the test metadata
var TC001Metadata = pkg.TestCase{
    ID:          "TC-001",
    Name:        "User Authentication",
    Description: "Verify user can authenticate with valid credentials",
    Priority:    pkg.PriorityCritical,
    Timeout:     30 * time.Second,
    Tags:        []string{"auth", "security", "smoke"},

    Setup: func(ctx context.Context) error {
        // Initialize test resources
        return nil
    },

    Execute: func(ctx context.Context) error {
        v := validator.NewValidator()

        // Test logic here
        v.AssertTrue(result, "Authentication succeeded")

        return nil
    },

    Teardown: func(ctx context.Context) error {
        // Clean up test resources
        return nil
    },
}
```

## Test Coverage Goals

- **Core Tests**: 100% of critical functionality
- **Integration Tests**: All external service integrations
- **Distributed Tests**: All coordination scenarios
- **Platform Tests**: All supported platforms

## Maintenance

- Review and update tests quarterly
- Add tests for new features before release
- Archive obsolete tests (don't delete)
- Keep test execution time under 30 minutes total
- Maintain >95% test reliability

## Support

For questions or issues with tests, see:
- [E2E Testing Framework](../E2E_TESTING_FRAMEWORK.md)
- [Implementation Plan](../E2E_TESTING_IMPLEMENTATION_PLAN.md)
- [Integration Guide](../INTEGRATION_GUIDE.md)
