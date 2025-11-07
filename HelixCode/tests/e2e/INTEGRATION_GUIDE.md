# Integrating Existing Tests into E2E Framework

## Overview

This guide explains how to integrate HelixCode's existing test suites into the new E2E testing framework, ensuring maximal consistency and a unified testing approach.

---

## 📊 Current Test Inventory

### Existing Test Structure

```
HelixCode/
├── test/                          # Legacy test directory
│   ├── integration/              # Service integration tests
│   │   ├── slack_integration_test.go
│   │   ├── discord_integration_test.go
│   │   ├── telegram_integration_test.go
│   │   └── integration_test.go
│   ├── automation/               # Provider automation tests
│   │   ├── anthropic_automation_test.go
│   │   ├── gemini_automation_test.go
│   │   ├── qwen_automation_test.go
│   │   ├── xai_automation_test.go
│   │   ├── openrouter_automation_test.go
│   │   └── free_providers_automation_test.go
│   ├── e2e/                      # End-to-end tests
│   │   ├── e2e_test.go
│   │   ├── comprehensive_e2e_test.go
│   │   ├── anthropic_gemini_e2e_test.go
│   │   └── qwen_e2e_test.go
│   └── load/                     # Load testing
│       └── notification_load_test.go
├── tests/                         # New test directory
│   ├── integration/              # New integration tests
│   │   └── integration_test.go
│   └── e2e/                      # New E2E framework
│       ├── E2E_TESTING_FRAMEWORK.md
│       ├── docker/
│       └── testbank/
├── benchmarks/                    # Performance benchmarks
│   └── performance_bench_test.go
├── internal/                      # Unit tests
│   ├── auth/*_test.go
│   ├── database/*_test.go
│   ├── llm/*_test.go
│   └── ...
└── applications/                  # Platform tests
    ├── harmony-os/main_test.go
    └── ...
```

### Test Categories Summary

| Category | Location | Count | Coverage |
|----------|----------|-------|----------|
| Unit Tests | `internal/**/*_test.go` | 50+ | Package-level |
| Integration Tests | `test/integration/` | 4 | Notifications |
| Automation Tests | `test/automation/` | 6 | LLM Providers |
| E2E Tests | `test/e2e/` | 4 | Full workflows |
| Load Tests | `test/load/` | 1 | Performance |
| Benchmarks | `benchmarks/` | 14 | Performance |
| Platform Tests | `applications/*/` | 12 | Harmony OS |

---

## 🔄 Migration Strategy

### Phase 1: Inventory and Analysis ✅

**Status**: Complete
**Output**: Test inventory documented above

### Phase 2: Create Wrapper Layer

**Goal**: Wrap existing tests to run in E2E framework without rewriting

**Approach**:
1. Create adapter functions
2. Convert test outputs to E2E format
3. Maintain existing test logic
4. Add E2E metadata

### Phase 3: Gradual Migration

**Goal**: Incrementally migrate tests to native E2E format

**Priority Order**:
1. High-value E2E tests
2. Integration tests
3. Automation tests
4. Unit tests (low priority - keep as-is)

### Phase 4: Consolidation

**Goal**: Single unified testing approach

**Outcome**:
- All tests runnable via E2E orchestrator
- Consistent reporting
- Unified CI/CD pipeline

---

## 🛠️ Integration Steps

### Step 1: Set Up Test Adapter

Create adapter to run existing Go tests within E2E framework:

```go
// tests/e2e/adapters/gotest_adapter.go
package adapters

import (
    "os/exec"
    "encoding/json"
)

type GoTestAdapter struct {
    PackagePath string
    TestName    string
}

func (a *GoTestAdapter) Run() (*TestResult, error) {
    cmd := exec.Command("go", "test", "-v", "-json",
        a.PackagePath, "-run", a.TestName)

    output, err := cmd.CombinedOutput()
    if err != nil {
        return nil, err
    }

    // Parse JSON output and convert to E2E result format
    result := parseGoTestJSON(output)
    return result, nil
}

func parseGoTestJSON(data []byte) *TestResult {
    // Implementation details...
}
```

### Step 2: Map Existing Tests to E2E Scenarios

Create mapping configuration:

```json
// tests/e2e/testbank/legacy-mappings.json
{
  "legacy_tests": [
    {
      "id": "TC-L001",
      "name": "Slack Integration Test",
      "category": "integration",
      "legacy_path": "test/integration",
      "legacy_test": "TestSlackIntegration",
      "adapter": "gotest",
      "priority": "high",
      "requires": ["SLACK_WEBHOOK_URL"]
    },
    {
      "id": "TC-L002",
      "name": "Anthropic Automation Test",
      "category": "automation",
      "legacy_path": "test/automation",
      "legacy_test": "TestAnthropicAutomation",
      "adapter": "gotest",
      "priority": "high",
      "requires": ["ANTHROPIC_API_KEY"]
    }
  ]
}
```

### Step 3: Create Test Bank Entries

For each existing test, create an E2E test case:

```json
// tests/e2e/testbank/scenarios/integration/slack.json
{
  "id": "TC-INT-001",
  "name": "Slack Notification Integration",
  "category": "integration",
  "priority": "high",
  "source": "legacy",
  "legacy_ref": "test/integration/slack_integration_test.go::TestSlackIntegration",
  "requires": ["SLACK_WEBHOOK_URL"],
  "adapter": {
    "type": "gotest",
    "package": "./test/integration",
    "test": "TestSlackIntegration"
  },
  "expected": {
    "status": "pass",
    "duration_max_seconds": 30
  }
}
```

### Step 4: Run Legacy Tests via E2E Orchestrator

```bash
# Run all legacy tests through E2E framework
cd tests/e2e
go run cmd/orchestrator/main.go run --legacy-all

# Run specific legacy test category
go run cmd/orchestrator/main.go run --legacy-category=integration

# Run specific legacy test
go run cmd/orchestrator/main.go run --test=TC-L001
```

---

## 📝 Step-by-Step Integration Examples

### Example 1: Integrating Slack Integration Test

#### Current Test
```go
// test/integration/slack_integration_test.go
func TestSlackIntegration(t *testing.T) {
    webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
    if webhookURL == "" {
        t.Skip("SLACK_WEBHOOK_URL not set")
    }

    client := slack.NewClient(webhookURL)
    err := client.SendMessage("Test message")
    assert.NoError(t, err)
}
```

#### E2E Scenario
```json
// tests/e2e/testbank/scenarios/integration/TC-INT-001-slack.json
{
  "id": "TC-INT-001",
  "name": "Slack Notification Integration",
  "category": "integration",
  "priority": "high",
  "tags": ["notifications", "slack", "legacy"],
  "requires": {
    "env_vars": ["SLACK_WEBHOOK_URL"],
    "services": ["mock-slack"]
  },
  "steps": [
    {
      "action": "run_legacy_test",
      "package": "./test/integration",
      "test": "TestSlackIntegration",
      "timeout": 30
    },
    {
      "action": "validate_mock_received",
      "service": "mock-slack",
      "endpoint": "/messages",
      "expected_calls": 1
    }
  ],
  "expected": {
    "status": "pass",
    "mock_calls_received": true
  }
}
```

#### Running the Test
```bash
# Via orchestrator
go run cmd/orchestrator/main.go run --test=TC-INT-001

# Direct via Go
E2E_MOCK_MODE=true go test ./test/integration -run TestSlackIntegration
```

### Example 2: Integrating Anthropic Automation Test

#### Current Test
```go
// test/automation/anthropic_automation_test.go
func TestAnthropicAutomation(t *testing.T) {
    apiKey := os.Getenv("ANTHROPIC_API_KEY")
    if apiKey == "" {
        t.Skip("ANTHROPIC_API_KEY not set")
    }

    client := anthropic.NewClient(apiKey)
    response, err := client.Complete("Hello, world!")
    assert.NoError(t, err)
    assert.NotEmpty(t, response)
}
```

#### E2E Scenario
```json
{
  "id": "TC-AUT-001",
  "name": "Anthropic Provider Automation",
  "category": "automation",
  "priority": "critical",
  "tags": ["llm", "anthropic", "automation", "legacy"],
  "requires": {
    "env_vars": ["ANTHROPIC_API_KEY"],
    "services": ["mock-llm-provider"]
  },
  "modes": {
    "mock": {
      "use_mock": true,
      "mock_service": "mock-llm-provider"
    },
    "real": {
      "use_mock": false,
      "rate_limit": "10/hour"
    }
  },
  "steps": [
    {
      "action": "configure_provider",
      "provider": "anthropic",
      "model": "claude-3-5-sonnet-20241022"
    },
    {
      "action": "run_legacy_test",
      "package": "./test/automation",
      "test": "TestAnthropicAutomation",
      "timeout": 60
    },
    {
      "action": "validate_response",
      "field": "content",
      "not_empty": true
    }
  ],
  "expected": {
    "status": "pass",
    "api_calls": 1,
    "response_valid": true
  }
}
```

### Example 3: Integrating E2E Comprehensive Test

#### Current Test
```go
// test/e2e/comprehensive_e2e_test.go
func TestComprehensiveE2E(t *testing.T) {
    // Setup
    server := setupTestServer(t)
    defer server.Close()

    // Create project
    project := createProject(t, server, "test-project")

    // Create task
    task := createTask(t, server, project.ID, "Build feature")

    // Assign worker
    worker := registerWorker(t, server)
    assignTask(t, server, task.ID, worker.ID)

    // Validate completion
    waitForCompletion(t, server, task.ID, 60*time.Second)

    result := getTaskResult(t, server, task.ID)
    assert.Equal(t, "completed", result.Status)
}
```

#### E2E Scenario
```json
{
  "id": "TC-E2E-001",
  "name": "Comprehensive Development Workflow",
  "category": "e2e",
  "priority": "critical",
  "tags": ["workflow", "full-stack", "legacy"],
  "timeout": 300,
  "requires": {
    "services": ["helixcode-server", "postgres-e2e", "redis-e2e"]
  },
  "steps": [
    {
      "action": "start_server",
      "wait_for_health": true,
      "timeout": 30
    },
    {
      "action": "run_legacy_test",
      "package": "./test/e2e",
      "test": "TestComprehensiveE2E",
      "timeout": 240
    },
    {
      "action": "collect_metrics",
      "metrics": ["task_duration", "api_calls", "database_queries"]
    }
  ],
  "expected": {
    "status": "pass",
    "task_completed": true,
    "no_errors": true
  }
}
```

---

## 🔧 Orchestrator Configuration

### Enable Legacy Test Support

```yaml
# tests/e2e/orchestrator/config.yml
orchestrator:
  legacy_support:
    enabled: true
    adapter_types:
      - gotest
      - pytest

  legacy_mappings:
    path: "../testbank/legacy-mappings.json"

  test_directories:
    - "./test/integration"
    - "./test/automation"
    - "./test/e2e"
    - "./test/load"

  execution:
    parallel_legacy_tests: 5
    retry_flaky_tests: 3
    timeout_default: 120

  reporting:
    include_legacy_in_dashboard: true
    legacy_test_badge: "🔄 Legacy"
```

### Orchestrator CLI Usage

```bash
# List all tests (including legacy)
go run cmd/orchestrator/main.go list --all

# List only legacy tests
go run cmd/orchestrator/main.go list --legacy

# Run all tests (E2E + legacy)
go run cmd/orchestrator/main.go run --all

# Run only legacy tests
go run cmd/orchestrator/main.go run --legacy-only

# Run mixed (some E2E, some legacy)
go run cmd/orchestrator/main.go run --category=integration  # Includes both

# Generate migration report
go run cmd/orchestrator/main.go migrate --analyze
```

---

## 📊 Migration Dashboard

### View Migration Progress

```bash
# Start test reporter with migration view
cd tests/e2e/reporter
go run cmd/main.go --enable-migration-dashboard

# Access at http://localhost:8088/migration
```

### Dashboard Features

- **Progress Tracker**: Shows which tests have been migrated
- **Coverage Comparison**: Legacy vs E2E test coverage
- **Performance Metrics**: Execution time comparison
- **Priority Matrix**: Which tests to migrate next

---

## 🎯 Best Practices

### 1. Gradual Migration

**Don't**: Rewrite all tests at once
**Do**: Migrate incrementally, starting with high-value tests

### 2. Maintain Legacy Tests

**Don't**: Delete legacy tests immediately after migration
**Do**: Keep both versions running in parallel initially

### 3. Consistent Naming

**Don't**: Use arbitrary IDs for migrated tests
**Do**: Use consistent naming scheme: `TC-{CATEGORY}-{NUMBER}`

### 4. Document Mappings

**Don't**: Lose track of what maps to what
**Do**: Maintain legacy-mappings.json with clear references

### 5. Preserve Test Intent

**Don't**: Change test logic during migration
**Do**: Keep test assertions identical, only adapt format

---

## 📋 Migration Checklist

### For Each Test Suite

- [ ] Inventory all tests in suite
- [ ] Document dependencies and requirements
- [ ] Create E2E test scenarios
- [ ] Add legacy mapping entries
- [ ] Configure orchestrator support
- [ ] Run tests via E2E framework
- [ ] Validate results match original
- [ ] Update CI/CD pipeline
- [ ] Document migration in changelog

### Quality Gates

- [ ] All legacy tests pass via E2E orchestrator
- [ ] Test execution time ≤ original
- [ ] No test failures introduced
- [ ] Coverage maintained or improved
- [ ] CI/CD integration complete
- [ ] Documentation updated

---

## 🚀 Quick Start Guide

### 1. Run Existing Tests via E2E Framework Today

```bash
# Set up environment
cd tests/e2e
cp .env.example .env
# Edit .env with your API keys

# Start Docker infrastructure
cd docker
docker-compose -f docker-compose.e2e.yml up -d

# Run all existing tests through adapter
cd ../orchestrator
go run cmd/main.go run --legacy-all --format=html --output=../reports/legacy-run.html

# View results
open ../reports/legacy-run.html
```

### 2. Migrate Your First Test

```bash
# Pick a test to migrate
cd tests/e2e

# Create E2E scenario
cat > testbank/scenarios/integration/my-test.json <<EOF
{
  "id": "TC-INT-999",
  "name": "My First Migrated Test",
  "category": "integration",
  "priority": "high",
  "adapter": {
    "type": "gotest",
    "package": "./test/integration",
    "test": "TestMyIntegration"
  },
  "expected": {
    "status": "pass"
  }
}
EOF

# Run migrated test
go run cmd/orchestrator/main.go run --test=TC-INT-999

# Validate it works
echo "✅ Test migrated successfully!"
```

### 3. Add to CI/CD

```yaml
# .github/workflows/e2e-tests.yml
name: E2E Tests (Including Legacy)

on: [push, pull_request]

jobs:
  e2e-all-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Start E2E Infrastructure
        run: |
          cd tests/e2e/docker
          docker-compose -f docker-compose.e2e.yml up -d

      - name: Run All Tests (E2E + Legacy)
        run: |
          cd tests/e2e/orchestrator
          go run cmd/main.go run --all --format=junit --output=results.xml

      - name: Upload Results
        uses: actions/upload-artifact@v3
        with:
          name: test-results
          path: tests/e2e/reports/
```

---

## 📖 Additional Resources

- **[E2E_TESTING_FRAMEWORK.md](./E2E_TESTING_FRAMEWORK.md)** - Complete framework documentation
- **[E2E_TESTING_IMPLEMENTATION_PLAN.md](./E2E_TESTING_IMPLEMENTATION_PLAN.md)** - Implementation roadmap
- **[README.md](./README.md)** - Quick start guide

---

## 🤝 Contributing

### Adding New Legacy Test Integration

1. Add entry to `testbank/legacy-mappings.json`
2. Create E2E scenario in `testbank/scenarios/`
3. Test locally
4. Submit PR with migration documentation

### Improving Adapters

1. Identify common patterns
2. Extend adapter functionality
3. Add tests for adapter
4. Document new features

---

**Document Version**: 1.0
**Last Updated**: 2025-11-07
**Status**: Ready for Implementation
