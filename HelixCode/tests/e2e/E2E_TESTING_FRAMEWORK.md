# HelixCode End-to-End Testing Framework

## Overview

Comprehensive AI-powered QA automation system that verifies all HelixCode components, flows, and integrations through real-world scenarios with actual AI execution.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Test Orchestrator (AI-Powered)              │
│  - Test Case Selection                                          │
│  - Execution Coordination                                       │
│  - Result Validation                                            │
│  - Report Generation                                            │
└──────────────────┬──────────────────────────────────────────────┘
                   │
    ┌──────────────┼──────────────┬──────────────┬────────────────┐
    │              │              │              │                │
┌───▼────┐  ┌─────▼─────┐  ┌────▼─────┐  ┌─────▼──────┐  ┌─────▼─────┐
│  Test  │  │   Mock    │  │  Real    │  │ Distributed│  │  Report   │
│  Bank  │  │ Services  │  │  Integ.  │  │  Testing   │  │  System   │
└────────┘  └───────────┘  └──────────┘  └────────────┘  └───────────┘
```

---

## Components

### 1. Test Orchestrator (`tests/e2e/orchestrator/`)

AI-powered test execution engine:
- **Test Selection**: Intelligently selects test cases based on changes
- **Parallel Execution**: Runs tests concurrently across multiple workers
- **Self-Healing**: Automatically retries transient failures
- **Adaptive Testing**: Learns from failures and adjusts test strategies

**Files**:
- `orchestrator.go` - Main orchestrator logic
- `ai_executor.go` - AI-based test execution
- `scheduler.go` - Test scheduling and parallelization
- `validator.go` - Result validation engine

### 2. Test Case Bank (`tests/e2e/testbank/`)

Structured repository of test scenarios:

```
testbank/
├── scenarios/           # Test scenario definitions
│   ├── core/           # Core functionality tests
│   ├── integrations/   # Provider integration tests
│   ├── distributed/    # Distributed computing tests
│   ├── platforms/      # Platform-specific tests
│   └── e2e/           # Full end-to-end workflows
├── fixtures/           # Test data and fixtures
├── expectations/       # Expected results
└── metadata.json       # Test case metadata
```

**Test Categories**:
1. **Core Tests**: Authentication, task management, worker pools
2. **Integration Tests**: LLM providers, notifications, databases
3. **Platform Tests**: Aurora OS, Harmony OS, mobile clients
4. **Distributed Tests**: Multi-node coordination, failover
5. **E2E Tests**: Complete development workflows

### 3. Mock Services (`tests/e2e/mocks/`)

Simulated external services:
- **Mock LLM Provider**: Simulates OpenAI, Anthropic, etc.
- **Mock Notification Services**: Slack, Discord, Email
- **Mock Storage**: S3, Database
- **Mock Worker Nodes**: Simulated SSH workers

**Features**:
- Configurable response times
- Error injection for failure testing
- Rate limiting simulation
- Network partition simulation

### 4. Real Integration Tests (`tests/e2e/real/`)

Tests with actual external services:
- **Local LLM**: llama.cpp, Ollama integration
- **Real Providers**: OpenAI, Anthropic (with test API keys)
- **Real Database**: PostgreSQL, Redis
- **Real Notifications**: Test channels

**Safety Measures**:
- Rate limiting to avoid API costs
- Test API keys with minimal quotas
- Automatic cleanup after tests
- Cost monitoring and alerts

### 5. Distributed Testing (`tests/e2e/distributed/`)

Multi-node test scenarios:
- **Worker Pool Tests**: SSH worker registration and coordination
- **Task Distribution**: Load balancing verification
- **Failover Tests**: Worker failure and recovery
- **Synchronization Tests**: Cross-device sync (Harmony OS)

**Infrastructure**:
- Docker Compose multi-container setup
- Simulated network conditions
- Node failure injection

### 6. Docker Compose Infrastructure (`tests/e2e/docker/`)

Complete testing environment:

```yaml
services:
  # Core Services
  postgres: PostgreSQL database
  redis: Redis cache
  helixcode: Main HelixCode server

  # Specialized Platforms
  aurora-os: Aurora OS server
  harmony-os-master: Harmony OS master
  harmony-os-worker-1: Harmony OS worker
  harmony-os-worker-2: Harmony OS worker

  # Mock Services
  mock-llm: Mock LLM provider
  mock-slack: Mock Slack API
  mock-storage: Mock S3

  # Test Infrastructure
  test-orchestrator: AI test executor
  test-reporter: Report aggregator

  # Real Local Services
  ollama: Local Ollama instance
  llama-cpp: Local llama.cpp server
```

### 7. AI-Powered QA Executor (`tests/e2e/ai-qa/`)

LLM-based test execution:
- **Test Understanding**: Parses test scenarios
- **Code Generation**: Creates actual test code
- **Execution**: Runs tests and monitors
- **Analysis**: Analyzes failures and suggests fixes

**Workflow**:
1. Read test scenario from test bank
2. Generate executable test code
3. Execute against HelixCode instance
4. Validate results against expectations
5. Generate detailed report with logs
6. Suggest fixes for failures

### 8. Reporting System (`tests/e2e/reports/`)

Comprehensive test reporting:
- **Real-time Dashboard**: Live test execution status
- **Detailed Logs**: Full execution traces
- **Metrics**: Success rate, performance, coverage
- **Failure Analysis**: Root cause identification
- **Trend Analysis**: Historical data and patterns

**Report Formats**:
- JSON (machine-readable)
- HTML (interactive dashboard)
- Markdown (documentation)
- JUnit XML (CI integration)

---

## Test Scenarios

### Core Functionality Tests

#### TC-001: Basic Authentication Flow
```json
{
  "id": "TC-001",
  "name": "Basic Authentication Flow",
  "category": "core",
  "priority": "critical",
  "steps": [
    {"action": "POST /api/v1/auth/login", "data": {"username": "admin", "password": "admin123"}},
    {"validate": "response.token", "type": "jwt"},
    {"validate": "response.user.roles", "contains": "admin"}
  ],
  "expected": {
    "status": 200,
    "token_valid": true,
    "session_created": true
  }
}
```

#### TC-002: Task Creation and Assignment
```json
{
  "id": "TC-002",
  "name": "Task Creation and Worker Assignment",
  "category": "core",
  "priority": "critical",
  "steps": [
    {"action": "authenticate"},
    {"action": "POST /api/v1/tasks", "data": {"title": "Test Task", "type": "planning"}},
    {"action": "POST /api/v1/workers/register", "data": {"hostname": "worker-1"}},
    {"validate": "task.assigned_worker", "equals": "worker-1"}
  ],
  "expected": {
    "task_created": true,
    "worker_assigned": true,
    "status": "assigned"
  }
}
```

### Integration Tests

#### TC-100: OpenAI Provider Integration
```json
{
  "id": "TC-100",
  "name": "OpenAI Provider Code Generation",
  "category": "integration",
  "priority": "high",
  "requires": ["OPENAI_API_KEY"],
  "steps": [
    {"action": "configure_provider", "provider": "openai", "model": "gpt-4"},
    {"action": "generate_code", "prompt": "Create a Python hello world function"},
    {"validate": "code", "contains": "def hello"},
    {"validate": "code", "syntax": "python"}
  ],
  "expected": {
    "code_generated": true,
    "syntax_valid": true,
    "executable": true
  }
}
```

#### TC-101: Local Ollama Integration
```json
{
  "id": "TC-101",
  "name": "Ollama Local Model Execution",
  "category": "integration",
  "priority": "high",
  "requires": ["ollama_service"],
  "steps": [
    {"action": "configure_provider", "provider": "ollama", "model": "llama3:8b"},
    {"action": "generate_text", "prompt": "Explain recursion in one sentence"},
    {"validate": "response.length", "min": 20},
    {"validate": "response", "contains": "function"}
  ],
  "expected": {
    "response_received": true,
    "latency_ms": "<2000"
  }
}
```

### Distributed Tests

#### TC-200: Multi-Node Task Distribution
```json
{
  "id": "TC-200",
  "name": "Distributed Task Execution Across 3 Nodes",
  "category": "distributed",
  "priority": "high",
  "requires": ["3_worker_nodes"],
  "steps": [
    {"action": "register_workers", "count": 3},
    {"action": "create_tasks", "count": 10, "type": "building"},
    {"validate": "tasks_distributed", "across_workers": 3},
    {"validate": "load_balance", "variance": "<20%"}
  ],
  "expected": {
    "all_tasks_assigned": true,
    "balanced_distribution": true,
    "no_worker_overload": true
  }
}
```

#### TC-201: Worker Failover Test
```json
{
  "id": "TC-201",
  "name": "Automatic Worker Failover on Node Failure",
  "category": "distributed",
  "priority": "critical",
  "requires": ["2_worker_nodes"],
  "steps": [
    {"action": "register_workers", "count": 2},
    {"action": "create_task", "assigned_to": "worker-1"},
    {"action": "simulate_failure", "worker": "worker-1"},
    {"validate": "task_reassigned", "to": "worker-2"},
    {"validate": "task_completed", "timeout": 60}
  ],
  "expected": {
    "failover_triggered": true,
    "task_reassigned": true,
    "no_data_loss": true
  }
}
```

### Platform-Specific Tests

#### TC-300: Aurora OS Security Levels
```json
{
  "id": "TC-300",
  "name": "Aurora OS Three-Level Security Enforcement",
  "category": "platform",
  "platform": "aurora-os",
  "priority": "critical",
  "steps": [
    {"action": "configure_security", "level": "standard"},
    {"validate": "audit_logging", "enabled": true},
    {"action": "configure_security", "level": "enhanced"},
    {"validate": "mfa_required", "enabled": true},
    {"action": "configure_security", "level": "maximum"},
    {"validate": "intrusion_detection", "enabled": true}
  ],
  "expected": {
    "all_levels_functional": true,
    "audit_logs_created": true,
    "security_enforced": true
  }
}
```

#### TC-301: Harmony OS Distributed Computing
```json
{
  "id": "TC-301",
  "name": "Harmony OS AI Acceleration with NPU/GPU",
  "category": "platform",
  "platform": "harmony-os",
  "priority": "high",
  "steps": [
    {"action": "enable_ai_acceleration", "devices": ["npu", "gpu"]},
    {"action": "create_ai_task", "model": "llama3", "precision": "FP16"},
    {"validate": "npu_utilized", "percentage": ">50"},
    {"validate": "performance", "tokens_per_sec": ">100"}
  ],
  "expected": {
    "acceleration_active": true,
    "performance_improved": true,
    "no_accuracy_loss": true
  }
}
```

### Real-World E2E Tests

#### TC-400: Complete Web Application Development
```json
{
  "id": "TC-400",
  "name": "AI-Generated Full-Stack Web Application",
  "category": "e2e",
  "priority": "critical",
  "timeout": 600,
  "steps": [
    {"action": "create_project", "name": "todo-app", "type": "web"},
    {"action": "ai_generate", "request": "Create a React + Node.js todo app with PostgreSQL"},
    {"validate": "files_created", "includes": ["frontend/", "backend/", "database/"]},
    {"action": "build_project"},
    {"validate": "build_successful", "no_errors": true},
    {"action": "run_tests"},
    {"validate": "tests_passed", "coverage": ">80%"},
    {"action": "deploy_local"},
    {"validate": "application_running", "url": "http://localhost:3000"}
  ],
  "expected": {
    "project_created": true,
    "code_quality": "high",
    "tests_passing": true,
    "application_functional": true
  }
}
```

#### TC-401: Microservices Architecture Generation
```json
{
  "id": "TC-401",
  "name": "Multi-Service Application with Docker Compose",
  "category": "e2e",
  "priority": "high",
  "timeout": 900,
  "steps": [
    {"action": "create_project", "name": "ecommerce", "architecture": "microservices"},
    {"action": "ai_generate", "request": "Create e-commerce with user, product, order services"},
    {"validate": "services_created", "count": 3},
    {"validate": "docker_compose", "exists": true},
    {"action": "docker_compose_up"},
    {"validate": "all_services_healthy", "timeout": 120},
    {"action": "test_api_endpoints"},
    {"validate": "api_functional", "all_endpoints": true}
  ],
  "expected": {
    "microservices_functional": true,
    "inter_service_communication": true,
    "data_persistence": true
  }
}
```

---

## Configuration

### Test Environment Variables

```bash
# Database
E2E_DATABASE_URL=postgresql://test:test@localhost:5432/helix_e2e
E2E_REDIS_URL=redis://localhost:6379/1

# API Keys (optional, for real integration tests)
E2E_OPENAI_API_KEY=sk-test-...
E2E_ANTHROPIC_API_KEY=sk-ant-test-...

# Test Configuration
E2E_PARALLEL_TESTS=10
E2E_TIMEOUT_SECONDS=300
E2E_RETRY_COUNT=3
E2E_MOCK_MODE=false  # true = use mocks, false = use real services

# AI QA Configuration
E2E_AI_MODEL=claude-3-5-sonnet-20241022
E2E_AI_PROVIDER=anthropic
E2E_AI_MAX_TOKENS=4096

# Reporting
E2E_REPORT_DIR=./tests/e2e/reports
E2E_LOG_LEVEL=debug
E2E_ENABLE_SCREENSHOTS=true

# Distributed Testing
E2E_WORKER_NODES=3
E2E_WORKER_MEMORY=2GB
E2E_WORKER_CPU=2
```

---

## Usage

### Run All Tests

```bash
# Full test suite with real integrations
cd tests/e2e
go test -v ./... -tags=e2e

# Or use the test orchestrator
./orchestrator run --all
```

### Run Specific Category

```bash
# Core functionality only
./orchestrator run --category=core

# Integration tests only (with real providers)
./orchestrator run --category=integration

# Distributed tests
./orchestrator run --category=distributed
```

### Run with Mock Services

```bash
# Use mocks for faster testing
E2E_MOCK_MODE=true ./orchestrator run --all
```

### Run AI-Powered QA

```bash
# Let AI execute and validate tests
./orchestrator run --ai-mode --model=claude-3-5-sonnet
```

### Generate Reports

```bash
# Generate HTML dashboard
./orchestrator report --format=html --output=./reports/dashboard.html

# Generate JUnit XML for CI
./orchestrator report --format=junit --output=./reports/results.xml
```

---

## CI/CD Integration

### GitHub Actions

```yaml
name: E2E Tests

on: [push, pull_request]

jobs:
  e2e-tests:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        category: [core, integration, distributed, platform, e2e]

    steps:
      - uses: actions/checkout@v3

      - name: Set up Docker Compose
        run: docker-compose -f tests/e2e/docker/docker-compose.yml up -d

      - name: Wait for services
        run: ./tests/e2e/scripts/wait-for-services.sh

      - name: Run E2E Tests
        run: |
          cd tests/e2e
          ./orchestrator run --category=${{ matrix.category }} --format=junit
        env:
          E2E_MOCK_MODE: true
          E2E_AI_MODEL: claude-3-5-sonnet

      - name: Upload Results
        uses: actions/upload-artifact@v3
        with:
          name: e2e-results-${{ matrix.category }}
          path: tests/e2e/reports/
```

---

## Maintenance

### Adding New Test Cases

1. Create test scenario JSON in `testbank/scenarios/`
2. Add test fixtures in `testbank/fixtures/`
3. Define expectations in `testbank/expectations/`
4. Run validator: `./orchestrator validate-test <test-id>`
5. Execute test: `./orchestrator run --test=<test-id>`

### Debugging Failed Tests

```bash
# Run with verbose logging
./orchestrator run --test=TC-400 --log-level=debug

# Enable screenshots on failure
./orchestrator run --test=TC-400 --screenshots

# Run with step-by-step mode
./orchestrator run --test=TC-400 --step-by-step
```

---

## Metrics and Reporting

### Success Metrics

- **Test Coverage**: % of code covered by E2E tests
- **Pass Rate**: % of tests passing
- **Failure Detection**: Time to detect regressions
- **Flakiness**: % of flaky tests
- **Execution Time**: Average test duration

### Reports Generated

1. **Test Execution Report**: Pass/fail status, duration, logs
2. **Coverage Report**: Code coverage, untested areas
3. **Performance Report**: Response times, resource usage
4. **Failure Analysis**: Root causes, suggested fixes
5. **Trend Report**: Historical data, patterns

---

## Best Practices

1. **Isolation**: Each test should be independent
2. **Cleanup**: Always clean up resources after tests
3. **Retries**: Implement smart retries for transient failures
4. **Timeouts**: Set appropriate timeouts for all operations
5. **Logging**: Comprehensive logging for debugging
6. **Mock Wisely**: Use mocks for external services, real for core logic
7. **CI Integration**: Run tests on every commit
8. **Cost Management**: Monitor API usage for real integrations
9. **Security**: Never commit real API keys
10. **Documentation**: Keep test scenarios documented

---

## Roadmap

### Phase 1 (Current)
- [x] Test framework architecture
- [ ] Test orchestrator implementation
- [ ] Test case bank structure
- [ ] Docker Compose infrastructure

### Phase 2 (Next)
- [ ] Mock services implementation
- [ ] Real integration tests
- [ ] AI-powered QA executor
- [ ] Reporting system

### Phase 3 (Future)
- [ ] Visual regression testing
- [ ] Performance benchmarking
- [ ] Chaos engineering tests
- [ ] Multi-region testing

---

## Support

- **Documentation**: `tests/e2e/docs/`
- **Examples**: `tests/e2e/examples/`
- **Issues**: GitHub Issues with `e2e-testing` label
- **Discussions**: GitHub Discussions #testing

---

**Last Updated**: 2025-11-07
**Version**: 1.0.0
**Status**: In Development
