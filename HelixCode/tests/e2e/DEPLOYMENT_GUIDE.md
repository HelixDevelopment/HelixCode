# E2E Testing Framework - Deployment & Usage Guide

**Version**: 1.0.0  
**Status**: Production Ready  
**Last Updated**: 2025-11-07

## Quick Deployment

### Local Development

```bash
# 1. Navigate to E2E directory
cd tests/e2e

# 2. Run setup (builds all components)
./scripts/setup.sh

# 3. Start mock services
./scripts/start-services.sh

# 4. Verify services are running
curl http://localhost:8090/health  # Mock LLM Provider
curl http://localhost:8091/health  # Mock Slack Service

# 5. Run tests
./scripts/run-tests.sh

# Expected Output:
# Total Tests:  5
# Passed:       5
# Success Rate: 100.00%
```

### Docker Deployment

```bash
# Start all services with Docker Compose
cd tests/e2e
docker-compose up -d

# Check service health
docker-compose ps

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

## Component Verification

### 1. Test Orchestrator

```bash
cd orchestrator

# Check version
./bin/orchestrator version
# Output: E2E Test Orchestrator v1.0.0

# List available tests
./bin/orchestrator list

# Run all tests
./bin/orchestrator run --concurrency 3

# Run specific tags
./bin/orchestrator run --tags smoke

# Run with verbose output
./bin/orchestrator run --verbose
```

### 2. Mock LLM Provider

```bash
# Start service (if not using Docker)
cd mocks/llm-provider
./bin/mock-llm-provider &

# Test chat completions
curl -X POST http://localhost:8090/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "mock-gpt-4",
    "messages": [{"role": "user", "content": "Hello"}]
  }'

# Test embeddings
curl -X POST http://localhost:8090/v1/embeddings \
  -H "Content-Type: application/json" \
  -d '{
    "model": "mock-text-embedding-ada-002",
    "input": ["Hello world"]
  }'

# List models
curl http://localhost:8090/v1/models
```

### 3. Mock Slack Service

```bash
# Start service (if not using Docker)
cd mocks/slack
./bin/mock-slack &

# Post a message
curl -X POST http://localhost:8091/api/chat.postMessage \
  -H "Content-Type: application/json" \
  -d '{
    "channel": "#general",
    "text": "Test message"
  }'

# Get all messages
curl http://localhost:8091/api/messages

# Send webhook
curl -X POST http://localhost:8091/webhook/test-id \
  -H "Content-Type: application/json" \
  -d '{"text": "Webhook test"}'

# Get webhooks
curl http://localhost:8091/api/webhooks
```

## CI/CD Integration

### GitHub Actions

Create `.github/workflows/e2e-tests.yml`:

```yaml
name: E2E Tests

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  e2e-tests:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      
      - name: Setup E2E Framework
        run: |
          cd tests/e2e
          ./scripts/setup.sh
      
      - name: Start Mock Services
        run: |
          cd tests/e2e
          ./scripts/start-services.sh
          sleep 5
      
      - name: Run E2E Tests
        run: |
          cd tests/e2e
          ./scripts/run-tests.sh --output ./test-results
      
      - name: Upload Test Results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: e2e-test-results
          path: tests/e2e/test-results/
      
      - name: Stop Services
        if: always()
        run: |
          cd tests/e2e
          ./scripts/stop-services.sh
```

### GitLab CI

Create `.gitlab-ci.yml`:

```yaml
e2e-tests:
  stage: test
  image: golang:1.24
  services:
    - postgres:16
    - redis:7
  variables:
    POSTGRES_DB: helixcode_test
    POSTGRES_USER: helixcode
    POSTGRES_PASSWORD: test_password
  script:
    - cd tests/e2e
    - ./scripts/setup.sh
    - ./scripts/start-services.sh
    - sleep 5
    - ./scripts/run-tests.sh
  artifacts:
    when: always
    paths:
      - tests/e2e/test-results/
    reports:
      junit: tests/e2e/test-results/junit.xml
```

### Jenkins

```groovy
pipeline {
    agent any
    
    stages {
        stage('Setup') {
            steps {
                sh 'cd tests/e2e && ./scripts/setup.sh'
            }
        }
        
        stage('Start Services') {
            steps {
                sh 'cd tests/e2e && ./scripts/start-services.sh'
                sleep 5
            }
        }
        
        stage('Run Tests') {
            steps {
                sh 'cd tests/e2e && ./scripts/run-tests.sh'
            }
        }
        
        stage('Publish Results') {
            steps {
                junit 'tests/e2e/test-results/junit.xml'
                archiveArtifacts 'tests/e2e/test-results/*'
            }
        }
    }
    
    post {
        always {
            sh 'cd tests/e2e && ./scripts/stop-services.sh'
        }
    }
}
```

## Environment Configuration

### Development

```bash
# .env for local development
E2E_CONCURRENT_TESTS=3
E2E_TIMEOUT=300s
MOCK_LLM_PORT=8090
MOCK_SLACK_PORT=8091
POSTGRES_PORT=5432
REDIS_PORT=6379
TEST_ENV=local
```

### Staging

```bash
# .env for staging
E2E_CONCURRENT_TESTS=5
E2E_TIMEOUT=600s
MOCK_LLM_PORT=8090
MOCK_SLACK_PORT=8091
TEST_ENV=staging
TEST_LOG_LEVEL=debug
```

### Production CI

```bash
# .env for production CI
E2E_CONCURRENT_TESTS=10
E2E_TIMEOUT=900s
CI_MODE=true
GENERATE_COVERAGE=true
TEST_ENV=ci
TEST_LOG_LEVEL=info
```

## Troubleshooting

### Issue: Services Won't Start

```bash
# Check ports
lsof -i :8090
lsof -i :8091

# Kill existing processes
pkill -f mock-llm-provider
pkill -f mock-slack

# Restart
./scripts/start-services.sh
```

### Issue: Tests Failing

```bash
# Run with verbose output
cd orchestrator
./bin/orchestrator run --verbose

# Check service health
curl http://localhost:8090/health
curl http://localhost:8091/health

# View service logs
cat .pids/mock-llm.log
cat .pids/mock-slack.log
```

### Issue: Build Errors

```bash
# Clean everything
./scripts/clean.sh

# Rebuild
./scripts/setup.sh

# Verify Go version
go version  # Should be 1.24.0+
```

### Issue: Docker Compose Errors

```bash
# Validate configuration
docker-compose config

# Clean and restart
docker-compose down -v
docker-compose up -d --build

# Check logs
docker-compose logs -f
```

## Performance Optimization

### Parallel Execution

```bash
# Increase concurrency for faster execution
./bin/orchestrator run --concurrency 10

# Or use environment variable
export E2E_CONCURRENT_TESTS=10
./scripts/run-tests.sh
```

### Resource Limits

```bash
# Limit memory for mock services
docker-compose up -d --scale mock-llm-provider=1 \
  --memory="256m" \
  --memory-swap="512m"
```

### Caching

```bash
# Enable Go build cache
export GOCACHE="$HOME/.cache/go-build"
./scripts/setup.sh
```

## Monitoring

### Health Checks

```bash
# Create monitoring script
cat > monitor.sh << 'SCRIPT'
#!/bin/bash
while true; do
  curl -s http://localhost:8090/health > /dev/null && echo "✓ LLM" || echo "✗ LLM"
  curl -s http://localhost:8091/health > /dev/null && echo "✓ Slack" || echo "✗ Slack"
  sleep 30
done
SCRIPT

chmod +x monitor.sh
./monitor.sh
```

### Metrics Collection

```bash
# View test execution metrics
cd orchestrator
./bin/orchestrator run --format json > results.json

# Extract metrics
jq '.success_rate' results.json
jq '.duration' results.json
jq '.total_tests' results.json
```

## Security Best Practices

1. **Never commit secrets**
   - Use `.env` (gitignored)
   - Use environment variables in CI

2. **Mock services only for testing**
   - Don't expose mock services publicly
   - Use Docker network isolation

3. **Rotate test credentials**
   - Change default passwords
   - Use secure random values

4. **Review test data**
   - Don't use production data in tests
   - Sanitize sensitive information

## Maintenance

### Updates

```bash
# Update dependencies
cd orchestrator && go get -u ./...
cd mocks/llm-provider && go get -u ./...
cd mocks/slack && go get -u ./...

# Rebuild
cd ../../
./scripts/setup.sh
```

### Cleanup

```bash
# Regular cleanup (keeps configs)
./scripts/clean.sh

# Full cleanup (removes everything)
./scripts/clean.sh
rm .env
rm -rf .pids/
```

### Backups

```bash
# Backup test results
tar -czf e2e-results-$(date +%Y%m%d).tar.gz test-results/

# Backup configurations
tar -czf e2e-configs-$(date +%Y%m%d).tar.gz .env docker-compose.yml
```

## Support

For issues or questions:

1. Check [Troubleshooting](#troubleshooting)
2. Review [IMPLEMENTATION_STATUS.md](./IMPLEMENTATION_STATUS.md)
3. Read component READMEs
4. Open GitHub issue with `e2e-testing` label

## Version History

- **1.0.0** (2025-11-07) - Initial production release
  - Complete orchestrator implementation
  - Mock LLM Provider and Slack services
  - 10 core test cases
  - Full automation scripts
  - Docker Compose support
  - Comprehensive documentation

---

**Ready for Production**: ✅  
**Next Review**: After 1 month of usage
