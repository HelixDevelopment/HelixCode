# HelixCode E2E Testing Framework

> **Status**: 🚧 In Development (Phase 1 Complete)
> **Version**: 1.0.0
> **Last Updated**: 2025-11-07

Comprehensive AI-powered end-to-end testing framework that validates all HelixCode components through real-world scenarios with actual AI execution.

---

## 📋 Table of Contents

1. [Quick Start](#quick-start)
2. [Architecture](#architecture)
3. [Documentation](#documentation)
4. [Current Status](#current-status)
5. [Next Steps](#next-steps)
6. [Contributing](#contributing)

---

## 🚀 Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.24+
- PostgreSQL 14+ (or use Docker)
- 8GB+ RAM recommended

### Start Test Environment

```bash
# Navigate to E2E directory
cd tests/e2e/docker

# Start all services (full profile)
docker-compose -f docker-compose.e2e.yml --profile full up -d

# Or start specific profiles:
# Core services only
docker-compose -f docker-compose.e2e.yml up -d

# With mocks
docker-compose -f docker-compose.e2e.yml --profile mocks up -d

# With local LLMs
docker-compose -f docker-compose.e2e.yml --profile local-llm up -d

# With monitoring
docker-compose -f docker-compose.e2e.yml --profile monitoring up -d
```

### Check Service Health

```bash
# All services should be healthy
docker-compose -f docker-compose.e2e.yml ps

# Or use the health check script
./scripts/wait-for-services.sh
```

### Access Services

| Service | URL | Credentials |
|---------|-----|-------------|
| HelixCode Server | http://localhost:8080 | admin/admin123 |
| Aurora OS | http://localhost:8081 | admin/admin123 |
| Harmony OS Master | http://localhost:8082 | admin/admin123 |
| Test Dashboard | http://localhost:8088 | - |
| Prometheus | http://localhost:9090 | - |
| Grafana | http://localhost:3001 | admin/admin |
| MinIO (Mock Storage) | http://localhost:9001 | minioadmin/minioadmin |

### Run Tests (Coming Soon)

```bash
# Once orchestrator is implemented:
cd tests/e2e/orchestrator
go run cmd/main.go run --all
```

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                Test Orchestrator (AI-Powered)                   │
│  - Test Selection & Execution                                   │
│  - Result Validation & Reporting                                │
└──────────────────┬────────────────────────────────────────────┘
                   │
    ┌──────────────┼──────────────┬──────────────┬────────────────┐
    │              │              │              │                │
┌───▼────┐  ┌─────▼─────┐  ┌────▼─────┐  ┌─────▼──────┐  ┌─────▼─────┐
│  Test  │  │   Mock    │  │  Real    │  │ Distributed│  │  Report   │
│  Bank  │  │ Services  │  │  Integ.  │  │  Testing   │  │  System   │
└────────┘  └───────────┘  └──────────┘  └────────────┘  └───────────┘
```

### Components

#### 1. Docker Infrastructure ✅
- Multi-container test environment
- All HelixCode platforms (main, Aurora OS, Harmony OS)
- Mock services (LLM, Slack, Storage)
- Local LLM services (Ollama, llama.cpp)
- Monitoring (Prometheus, Grafana)

#### 2. Test Orchestrator 🔄
- AI-powered test execution engine
- Parallel test execution
- Smart retry logic
- Result validation

#### 3. Test Case Bank 📝
- Structured test scenarios
- Core functionality tests
- Integration tests
- Platform-specific tests
- End-to-end workflows

#### 4. Mock Services 🎭
- Mock LLM Provider
- Mock Slack/Notifications
- Mock Storage (MinIO)

#### 5. Real Integrations 🌐
- Local Ollama
- Local llama.cpp
- Real API tests (with safety limits)

#### 6. Reporting System 📊
- Real-time dashboard
- Detailed execution logs
- Metrics and trends
- Failure analysis

---

## 📚 Documentation

### Core Documents

1. **[E2E_TESTING_FRAMEWORK.md](./E2E_TESTING_FRAMEWORK.md)** - Complete architecture and design
2. **[E2E_TESTING_IMPLEMENTATION_PLAN.md](./E2E_TESTING_IMPLEMENTATION_PLAN.md)** - Phased implementation plan
3. **[docker-compose.e2e.yml](./docker/docker-compose.e2e.yml)** - Docker infrastructure

### Test Categories

#### Core Tests
- Authentication & Authorization
- Task Management
- Worker Pool Management
- Project Lifecycle
- Session Management

#### Integration Tests
- LLM Providers (OpenAI, Anthropic, Ollama)
- Notification Services (Slack, Discord, Email)
- Database Operations
- Redis Caching

#### Platform Tests
- Aurora OS Security Levels
- Harmony OS Distributed Computing
- Mobile Client Integration
- Desktop Applications

#### Distributed Tests
- Multi-node Coordination
- Failover Scenarios
- Load Balancing
- Cross-device Synchronization

#### End-to-End Tests
- Complete Web App Development
- Microservices Generation
- Full Development Workflows

---

## 📊 Current Status

### ✅ Phase 1: Foundation (Complete)

- [x] Architecture design documented
- [x] Docker Compose infrastructure created
- [x] Service definitions (20+ services)
- [x] Network configuration
- [x] Volume management
- [x] Health checks
- [x] Implementation plan

### 🔄 Phase 2: Core Implementation (In Progress)

- [ ] Test orchestrator (CLI tool)
- [ ] Test case bank structure
- [ ] Sample test scenarios (10+)
- [ ] Mock LLM provider
- [ ] Mock notification services
- [ ] Basic reporting

### 📋 Phase 3-6: Advanced Features (Planned)

- [ ] AI-powered QA executor
- [ ] Real provider integrations
- [ ] Distributed testing scenarios
- [ ] Comprehensive reporting dashboard
- [ ] CI/CD integration
- [ ] Performance benchmarks

---

## 🎯 Next Steps (This Week)

### Priority 1: Test Orchestrator MVP
```bash
tests/e2e/orchestrator/
├── cmd/main.go          # CLI tool
├── pkg/
│   ├── executor/        # Test execution
│   ├── validator/       # Result validation
│   └── reporter/        # Report generation
└── Dockerfile
```

### Priority 2: Sample Test Cases
```bash
tests/e2e/testbank/
├── scenarios/
│   ├── core/           # 10 core tests
│   ├── integration/    # 5 integration tests
│   └── e2e/           # 3 end-to-end tests
└── metadata.json
```

### Priority 3: Mock Services
```bash
tests/e2e/mocks/
├── llm-provider/       # Mock LLM API
└── slack/              # Mock Slack API
```

### Priority 4: Quick Start Scripts
```bash
tests/e2e/scripts/
├── start-e2e-env.sh    # Start environment
├── wait-for-services.sh # Health check
└── run-tests.sh        # Execute tests
```

---

## 🔧 Development

### Building Components

```bash
# Build orchestrator
cd tests/e2e/orchestrator
go build -o bin/orchestrator cmd/main.go

# Build mock services
cd tests/e2e/mocks/llm-provider
docker build -t helix-mock-llm .
```

### Running Locally

```bash
# Set environment variables
export E2E_DATABASE_URL="postgresql://helix_test:test_password_123@localhost:5433/helix_e2e"
export E2E_REDIS_URL="redis://:test_redis_pass@localhost:6380/3"
export E2E_SERVER_URL="http://localhost:8080"

# Run orchestrator
cd tests/e2e/orchestrator
go run cmd/main.go run --test=TC-001
```

### Debugging

```bash
# View logs
docker-compose -f docker-compose.e2e.yml logs -f helixcode-server
docker-compose -f docker-compose.e2e.yml logs -f test-orchestrator

# Exec into containers
docker exec -it helix-e2e-postgres psql -U helix_test -d helix_e2e
docker exec -it helix-e2e-redis redis-cli -a test_redis_pass
```

---

## 🤝 Contributing

### Adding New Test Cases

1. Create test scenario JSON in `testbank/scenarios/`
2. Add test fixtures if needed
3. Define expected results
4. Test locally before committing

### Adding New Mock Services

1. Create service directory in `mocks/`
2. Implement HTTP server
3. Add Dockerfile
4. Update docker-compose.e2e.yml
5. Document endpoints

### Improving Orchestrator

1. Follow existing code structure
2. Add tests for new features
3. Update CLI help text
4. Document new commands

---

## 📈 Metrics & Goals

### Test Coverage Goals
- Unit Tests: 80%+
- Integration Tests: 70%+
- E2E Tests: 50%+
- Platform Tests: 60%+

### Performance Goals
- Full test suite: <30 minutes
- Parallel tests: 10+ concurrent
- Test reliability: >95%
- Report generation: <5 seconds

### Quality Goals
- All features documented
- CI on every PR
- Regression detection: <1 hour
- Average fix time: <24 hours

---

## 🐛 Troubleshooting

### Services Won't Start

```bash
# Check Docker resources
docker system df
docker system prune -a

# Check port conflicts
lsof -i :8080  # HelixCode
lsof -i :5433  # PostgreSQL
lsof -i :6380  # Redis

# Restart with clean slate
docker-compose -f docker-compose.e2e.yml down -v
docker-compose -f docker-compose.e2e.yml up -d
```

### Database Connection Issues

```bash
# Check PostgreSQL is running
docker exec helix-e2e-postgres pg_isready -U helix_test

# Check database exists
docker exec helix-e2e-postgres psql -U helix_test -l

# Reset database
docker-compose -f docker-compose.e2e.yml down -v postgres-e2e
docker-compose -f docker-compose.e2e.yml up -d postgres-e2e
```

### Tests Failing

```bash
# Run with verbose logging
E2E_LOG_LEVEL=debug go run cmd/main.go run --test=TC-001

# Check service health
curl http://localhost:8080/health
curl http://localhost:8081/health
curl http://localhost:8082/health

# View test logs
cat tests/e2e/reports/latest/test-TC-001.log
```

---

## 📞 Support

- **Issues**: GitHub Issues with `e2e-testing` label
- **Discussions**: GitHub Discussions #testing
- **Documentation**: [Full Framework Docs](./E2E_TESTING_FRAMEWORK.md)

---

## 📝 License

See main HelixCode repository for license information.

---

**Maintained by**: HelixCode Team
**Status**: Active Development
**Roadmap**: [Implementation Plan](./E2E_TESTING_IMPLEMENTATION_PLAN.md)
