# E2E Test Orchestrator

A powerful test orchestrator for running end-to-end tests with parallel execution, priority scheduling, and comprehensive reporting.

## Features

- **Parallel Execution**: Run multiple tests concurrently with configurable concurrency limits
- **Priority-Based Scheduling**: Tests execute in order: Critical → High → Normal → Low
- **Retry Logic**: Automatic retry for flaky tests with configurable retry count and delay
- **Fail-Fast Mode**: Stop execution on first failure to save time
- **Multiple Report Formats**: JSON and JUnit XML output for CI/CD integration
- **Tag-Based Filtering**: Run specific test subsets using tags or IDs
- **Assertion Framework**: Built-in validators for common assertions
- **Mock LLM Provider**: Configurable mock provider for testing LLM-dependent features
- **Timeout Management**: Per-test and global timeout configuration
- **Context Cancellation**: Graceful shutdown and cancellation support

## Quick Start

### Build
```bash
cd tests/e2e/orchestrator
go build -o bin/orchestrator ./cmd/main.go
```

### Run Tests
```bash
# Run all tests
./bin/orchestrator run

# Run with verbose output
./bin/orchestrator run --verbose

# Run specific tests by ID
./bin/orchestrator run --tests TC-001,TC-002

# Run tests by tag
./bin/orchestrator run --tags smoke,health

# Output to JUnit XML
./bin/orchestrator run --format junit --output results.xml
```

### List Available Tests
```bash
./bin/orchestrator list
```

### Show Version
```bash
./bin/orchestrator version
```

## Architecture

```
orchestrator/
├── cmd/
│   └── main.go              # CLI entry point
├── pkg/
│   ├── types.go             # Core types and interfaces
│   ├── executor/
│   │   └── executor.go      # Test execution engine
│   ├── scheduler/
│   │   └── scheduler.go     # Priority-based scheduler
│   ├── validator/
│   │   └── validator.go     # Assertion helpers
│   ├── reporter/
│   │   ├── reporter.go      # Reporter interface
│   │   ├── json.go          # JSON reporter
│   │   └── junit.go         # JUnit XML reporter
│   └── mock/
│       ├── llm_provider.go  # Mock LLM provider
│       └── llm_provider_test.go
└── bin/
    └── orchestrator         # Compiled binary
```

## CLI Reference

### Global Flags

```
--parallel, -p         Run tests in parallel (default: true)
--concurrency, -c      Maximum concurrent tests (default: 10)
--timeout, -t          Global timeout (default: 30m)
--retry, -r            Retry count for failed tests (default: 0)
--fail-fast, -f        Stop on first failure (default: false)
--verbose, -v          Verbose output (default: false)
--format, -o           Output format: json, junit (default: json)
--output               Output file path
--tests                Comma-separated test IDs to run
--tags                 Comma-separated tags to filter tests
```

### Commands

#### run
Run test suite with optional filtering and configuration.

```bash
./bin/orchestrator run [flags]
```

Examples:
```bash
# Run critical tests only
./bin/orchestrator run --tags smoke --fail-fast

# Run with retry and timeout
./bin/orchestrator run --retry 2 --timeout 10m

# Sequential execution
./bin/orchestrator run --parallel=false

# Export JUnit for CI
./bin/orchestrator run --format junit --output reports/junit.xml
```

#### list
Display all available tests with their properties.

```bash
./bin/orchestrator list
```

Output:
```
Test Suite: Sample E2E Test Suite
Total Tests: 5

ID: TC-001
  Name: Basic Health Check
  Description: Verify system health check endpoint
  Priority: 3
  Tags: [smoke health]
...
```

#### version
Show orchestrator version information.

```bash
./bin/orchestrator version
```

## Test Development

### Creating Test Cases

```go
import (
    "context"
    "time"
    "dev.helix.code/tests/e2e/orchestrator/pkg"
)

func MyTestSuite() *pkg.TestSuite {
    return &pkg.TestSuite{
        Name:        "My Test Suite",
        Description: "Description of the test suite",
        Tests: []*pkg.TestCase{
            {
                ID:          "TC-001",
                Name:        "Test Name",
                Description: "What this test does",
                Priority:    pkg.PriorityCritical,
                Timeout:     10 * time.Second,
                Tags:        []string{"smoke", "api"},
                
                Setup: func(ctx context.Context) error {
                    // Initialize resources
                    return nil
                },
                
                Execute: func(ctx context.Context) error {
                    // Run test logic
                    return nil
                },
                
                Teardown: func(ctx context.Context) error {
                    // Clean up resources
                    return nil
                },
            },
        },
    }
}
```

### Using the Validator

```go
import "dev.helix.code/tests/e2e/orchestrator/pkg/validator"

func TestExample(ctx context.Context) error {
    v := validator.NewValidator()
    
    // Basic assertions
    v.AssertTrue(1+1 == 2, "Math works")
    v.AssertEqual(expected, actual, "Values match")
    v.AssertNotNil(result, "Result exists")
    v.AssertContains(str, "substring", "Contains check")
    
    // Get assertion results
    assertions := v.GetAssertions()
    return nil
}
```

### Priority Levels

Tests are executed in priority order:

```go
pkg.PriorityCritical  // 3 - Must pass (smoke tests, health checks)
pkg.PriorityHigh      // 2 - Important features
pkg.PriorityNormal    // 1 - Standard tests
pkg.PriorityLow       // 0 - Nice-to-have
```

## Mock LLM Provider

The orchestrator includes a configurable mock LLM provider for testing LLM-dependent features.

### Basic Usage

```go
import "dev.helix.code/tests/e2e/orchestrator/pkg/mock"

// Create provider
provider := mock.NewMockLLMProvider("test-llm")

// Configure behavior
provider.SetResponseDelay(100 * time.Millisecond)
provider.SetErrorRate(0.1) // 10% error rate
provider.AddResponse("weather", "It's sunny today!")

// Generate response
request := &mock.LLMRequest{
    Model: "mock-model",
    Messages: []mock.Message{
        {Role: "user", Content: "What's the weather?"},
    },
}

response, err := provider.Generate(context.Background(), request)
```

### Mock Provider Features

- **Configurable Delays**: Simulate network latency
- **Error Simulation**: Test error handling with configurable error rates
- **Response Templates**: Map prompts to specific responses
- **Token Usage Tracking**: Realistic token counting
- **Availability Control**: Simulate provider outages
- **Request Counting**: Track API usage
- **Context Cancellation**: Proper timeout handling

### Mock Provider Methods

```go
provider.SetResponseDelay(delay time.Duration)
provider.SetErrorRate(rate float64)  // 0.0 to 1.0
provider.SetAvailable(available bool)
provider.AddResponse(pattern, response string)
provider.SetDefaultResponse(response string)
provider.GetRequestCount() int
provider.Reset()
```

## Report Formats

### JSON Report

```json
{
  "suite_name": "Sample E2E Test Suite",
  "duration": 301214750,
  "total_tests": 5,
  "passed": 5,
  "failed": 0,
  "skipped": 0,
  "timed_out": 0,
  "success_rate": 100,
  "results": [
    {
      "test_id": "TC-001",
      "test_name": "Basic Health Check",
      "status": "passed",
      "duration": 102047083
    }
  ]
}
```

### JUnit XML Report

```xml
<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="Sample E2E Test Suite" tests="5" failures="0" skipped="0" time="0.301">
  <testcase name="Basic Health Check" classname="Sample E2E Test Suite" time="0.102"/>
  <testcase name="Service Discovery" classname="Sample E2E Test Suite" time="0.202"/>
  ...
</testsuite>
```

## Test Status

- `pending`: Test not yet started
- `running`: Test currently executing
- `passed`: Test completed successfully
- `failed`: Test failed with error
- `skipped`: Test was skipped
- `timeout`: Test exceeded timeout

## Integration with CI/CD

### GitHub Actions

```yaml
name: E2E Tests
on: [push, pull_request]

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      
      - name: Build Orchestrator
        run: |
          cd tests/e2e/orchestrator
          go build -o bin/orchestrator ./cmd/main.go
      
      - name: Run E2E Tests
        run: |
          cd tests/e2e/orchestrator
          ./bin/orchestrator run --format junit --output results.xml
      
      - name: Publish Test Results
        uses: EnricoMi/publish-unit-test-result-action@v2
        if: always()
        with:
          files: tests/e2e/orchestrator/results.xml
```

## Performance

- **Build Time**: ~1-2 seconds
- **Binary Size**: ~5.9 MB
- **Startup Time**: <10ms
- **Test Overhead**: ~1ms per test
- **Parallel Efficiency**: Near-linear scaling up to 100 concurrent tests

## Testing

Run the orchestrator's own tests:

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test -v ./pkg/mock/...

# Run with race detector
go test -race ./...
```

## Dependencies

- `github.com/spf13/cobra` v1.8.1 - CLI framework
- `github.com/stretchr/testify` v1.10.0 - Testing assertions
- `github.com/google/uuid` v1.6.0 - UUID generation

## License

Part of the HelixCode project.
