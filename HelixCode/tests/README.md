# 🧪 HelixCode Local LLM - Comprehensive Test Suite

This directory contains a complete, production-ready test suite for the HelixCode Local LLM Management System. The test suite covers all aspects of the system with real hardware automation and multi-provider validation.

## 📋 Test Suite Overview

### **Test Categories**

| Category | Description | Files | Execution Time |
|----------|-------------|---------|---------------|
| **Security** | OWASP Top 10, SonarQube, Snyk compliance | `security/security_test.go` | 30-60s |
| **Unit** | Component-level tests with mocks | `unit/local_llm_manager_test.go` | 2-5min |
| **Integration** | Real provider integration tests | `integration/provider_integration_test.go` | 5-15min |
| **E2E** | Complete user workflow tests | `e2e/complete_workflow_test.go` | 5-10min |
| **Automation** | Hardware-specific automation | `automation/hardware_test.go` | 10-30min |

---

## 🚀 Quick Start

### **Run All Tests**
```bash
# Complete test suite
./run_tests.sh --all

# With coverage
./run_tests.sh --all --coverage

# Skip expensive tests
./run_tests.sh --all --skip-expensive
```

### **Run Specific Test Categories**
```bash
# Security and compliance
./run_tests.sh --security

# Unit tests only
./run_tests.sh --unit

# Integration tests
./run_tests.sh --integration

# End-to-end tests
./run_tests.sh --e2e

# Hardware automation tests
./run_tests.sh --automation
```

### **Advanced Options**
```bash
# Parallel execution
./run_tests.sh --all --parallel=8

# Custom timeout
./run_tests.sh --all --timeout=20m

# Skip hardware-dependent tests
./run_tests.sh --all --skip-hardware

# Generate performance benchmarks
./run_tests.sh --benchmarks

# Coverage report
./run_tests.sh --coverage
```

---

## 🔒 Security Tests

### **Coverage Areas**
- **Input Validation**: Path traversal, injection attacks, XSS
- **Authentication**: Token validation, session management, auth bypass
- **Authorization**: Access control, privilege escalation
- **Cryptography**: TLS configuration, secure defaults
- **Infrastructure**: Network security, dependency scanning
- **OWASP Top 10**: All categories covered

### **Security Test Execution**
```bash
# Run security tests
./run_tests.sh --security

# With detailed logging
SECURITY_DEBUG=1 ./run_tests.sh --security

# Generate security report
./test_runner -security --output security-report.json
```

### **Security Compliance**
- ✅ SonarQube rules compliance
- ✅ Snyk vulnerability scanning
- ✅ OWASP Top 10 coverage
- ✅ TLS 1.2+ enforcement
- ✅ No hardcoded credentials
- ✅ Input sanitization
- ✅ Authentication bypass resistance

---

## 🧪 Unit Tests

### **Test Coverage**
- **Provider Management**: Start/stop, status, health checks
- **Model Management**: Download, sharing, conversion
- **Configuration**: YAML parsing, validation
- **Analytics**: Usage tracking, performance metrics
- **Error Handling**: Edge cases, failure scenarios
- **Resource Management**: Cleanup, memory management

### **Mock Strategy**
- **Provider Mocks**: Realistic provider behavior simulation
- **HTTP Client Mocks**: API response simulation
- **File System Mocks**: Safe file operations
- **Timer Mocks**: Deterministic time-based tests

### **Unit Test Execution**
```bash
# Run unit tests
./run_tests.sh --unit

# With race detection
./run_tests.sh --unit && go test -race ./tests/unit/

# With coverage
./run_tests.sh --unit --coverage
```

---

## 🔗 Integration Tests

### **Real Provider Testing**
Tests against actual local LLM providers when available:

| Provider | Tested Features | Requirements |
|----------|----------------|--------------|
| **VLLM** | API compatibility, GPU acceleration, batching | Python + vllm package |
| **LocalAI** | OpenAI API compatibility, multi-format support | localai binary |
| **Ollama** | Model management, CLI interface | ollama binary |
| **Llama.cpp** | GGUF support, CPU optimization | llama.cpp binary |
| **MLX** | Apple Silicon optimization, Metal | Python + mlx package |
| **FastChat** | Vicuna models, training interface | Python + fastchat |
| **TextGen** | Web UI, character cards | textgen-webui |
| **LM Studio** | Desktop app, model management | LM Studio |
| **Jan AI** | RAG capabilities, cross-platform | Jan AI |
| **KoboldAI** | Creative writing, storytelling | KoboldAI |
| **GPT4All** | CPU optimization, low resource | GPT4All |
| **TabbyAPI** | Quantization, performance | TabbyAPI |
| **MistralRS** | Rust implementation, performance | mistralrs |

### **Integration Test Features**
- **Provider Lifecycle**: Start, health check, stop
- **Model Loading**: Real model download and loading
- **API Compatibility**: Request/response validation
- **Performance Testing**: TPS, latency, memory usage
- **Failover Testing**: Provider failure recovery
- **Cross-Provider**: Model sharing and compatibility

### **Integration Test Execution**
```bash
# Run integration tests
./run_tests.sh --integration

# Include expensive tests
./run_tests.sh --integration --include-expensive

# With specific providers
PROVIDERS=vllm,ollama ./run_tests.sh --integration
```

---

## 🎯 End-to-End Tests

### **User Workflows Tested**
1. **New User Setup**: Initial system configuration
2. **Advanced Workflow**: Multi-provider orchestration
3. **Production Deployment**: Scalable setup
4. **Model Management**: Download, share, optimize
5. **Analytics Monitoring**: Performance tracking

### **E2E Test Features**
- **CLI Testing**: Complete command-line interface
- **Configuration Management**: Real config files
- **System Integration**: OS-specific optimizations
- **Performance Benchmarking**: Real-world metrics
- **Resource Utilization**: Memory, CPU, GPU monitoring

### **E2E Test Execution**
```bash
# Run E2E tests
./run_tests.sh --e2e

# Skip expensive operations
./run_tests.sh --e2e --skip-expensive

# With specific hardware profile
HARDWARE_PROFILE=apple-silicon ./run_tests.sh --e2e
```

---

## 🤖 Hardware Automation Tests

### **Hardware Detection**
- **CPU Info**: Cores, threads, frequency, architecture
- **GPU Info**: Name, VRAM, drivers (CUDA, Metal, Vulkan)
- **Memory Info**: Total, available, usage
- **OS Info**: Distribution, version, architecture

### **Platform Optimization**
- **Apple Silicon**: MLX, Metal, unified memory
- **NVIDIA GPUs**: CUDA acceleration, tensor cores
- **AMD GPUs**: ROCm, Vulkan, OpenCL
- **CPU Optimization**: AVX, neon, instruction sets

### **Hardware-Specific Testing**
- **Model Selection**: Appropriate models for hardware
- **Provider Optimization**: Hardware-specific tuning
- **Performance Benchmarks**: Real throughput testing
- **Resource Monitoring**: Memory, CPU, GPU utilization

### **Hardware Test Execution**
```bash
# Run hardware automation
./run_tests.sh --automation

# Skip hardware tests (for CI)
./run_tests.sh --automation --skip-hardware

# With specific hardware constraints
MIN_RAM=8GB MIN_VRAM=4GB ./run_tests.sh --automation
```

---

## 📊 Test Configuration

### **Environment Variables**
```bash
# General configuration
PARALLEL_JOBS=8              # Parallel test execution
TEST_TIMEOUT=10m              # Test timeout
SKIP_EXPENSIVE_TESTS=true     # Skip time-consuming tests
SKIP_HARDWARE_TESTS=true      # Skip hardware dependencies

# Testing configuration
TESTING_SHORT=true            # Enable short test mode
CI_MODE=true                  # CI-specific optimizations
DEBUG_TESTS=true              # Enable debug output

# Provider configuration
PROVIDERS=vllm,ollama         # Specific providers to test
MODEL_CACHE_DIR=/tmp/models   # Model download directory
CONFIG_DIR=/tmp/config       # Test configuration directory
```

### **Test Configuration Files**
- `test-runner-config.yaml`: Global test configuration
- `provider-config.yaml`: Test provider configurations
- `hardware-profile.yaml`: Hardware test profiles
- `test-models.yaml`: Model test configurations

---

## 📈 Test Results & Reporting

### **Result Files**
```
test-results/
├── security-20241201-143022.log
├── unit-20241201-143045.log
├── integration-20241201-143235.log
├── e2e-20241201-143545.log
├── automation-20241201-144200.log
├── coverage-20241201-144321.out
├── coverage-20241201-144321.html
├── test-report-20241201-144500.md
└── benchmark-20241201-144625.txt
```

### **Coverage Reports**
- **HTML Report**: Interactive coverage visualization
- **Text Report**: Detailed line-by-line coverage
- **Summary**: Overall coverage percentage
- **Trends**: Coverage changes over time

### **Performance Benchmarks**
- **TPS Metrics**: Tokens per second by provider
- **Latency Analysis**: Response time distribution
- **Memory Usage**: Peak and average memory consumption
- **GPU Utilization**: GPU memory and compute usage

---

## 🛠️ Test Development

### **Adding New Tests**

1. **Unit Tests**: Add to `tests/unit/`
   ```go
   func TestNewFeature(t *testing.T) {
       // Test implementation
   }
   ```

2. **Integration Tests**: Add to `tests/integration/`
   ```go
   func TestNewProviderIntegration(t *testing.T) {
       if testing.Short() {
           t.Skip("Skipping integration test")
       }
       // Test implementation
   }
   ```

3. **E2E Tests**: Add to `tests/e2e/`
   ```go
   func TestNewUserWorkflow(t *testing.T) {
       // Workflow test implementation
   }
   ```

4. **Hardware Tests**: Add to `tests/automation/`
   ```go
   func TestNewHardwareFeature(t *testing.T) {
       hwInfo, err := detectHardware()
       require.NoError(t, err)
       // Hardware-specific test
   }
   ```

### **Test Naming Conventions**
- **Unit Tests**: `Test<Component>_<Function>_<Scenario>`
- **Integration Tests**: `Test<Provider>_<Feature>_<Scenario>`
- **E2E Tests**: `Test<UserWorkflow>_<Scenario>`
- **Hardware Tests**: `Test<Platform>_<Feature>_<Scenario>`

### **Mock Implementation**
```go
type MockProvider struct {
    mock.Mock
}

func (m *MockProvider) Generate(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
    args := m.Called(ctx, req)
    return args.Get(0).(*LLMResponse), args.Error(1)
}
```

---

## 🔧 Troubleshooting

### **Common Issues**

#### **Test Failures**
```bash
# Check test logs
cat test-results/unit-latest.log

# Run with verbose output
./run_tests.sh --unit --verbose

# Run specific test
go test -v ./tests/unit/ -run TestSpecificFunction
```

#### **Provider Test Failures**
```bash
# Check provider availability
./run_tests.sh --preflight

# Skip missing providers
SKIP_MISSING_PROVIDERS=true ./run_tests.sh --integration

# Test with mock providers
USE_MOCK_PROVIDERS=true ./run_tests.sh --integration
```

#### **Hardware Test Failures**
```bash
# Skip hardware tests
./run_tests.sh --automation --skip-hardware

# Check hardware detection
./test_runner --preflight

# Test with simulated hardware
SIMULATE_HARDWARE=true ./run_tests.sh --automation
```

#### **Memory Issues**
```bash
# Reduce parallel execution
./run_tests.sh --all --parallel=1

# Increase swap if needed
sudo swapon /swapfile

# Run with memory profiling
GOMEMLIMIT=1GiB ./run_tests.sh --unit
```

### **CI/CD Integration**

#### **GitHub Actions**
```yaml
- name: Run Tests
  run: |
    ./run_tests.sh --all --coverage
    bash <(curl -s https://codecov.io/bash) -f test-results/coverage-latest.out
```

#### **Docker Testing**
```dockerfile
FROM golang:1.21
COPY . /app
WORKDIR /app
RUN ./run_tests.sh --all --skip-hardware
```

---

## 📚 Best Practices

### **Test Design Principles**
1. **Isolation**: Tests should not depend on each other
2. **Determinism**: Same input should produce same output
3. **Fast Execution**: Unit tests should run quickly
4. **Clear Assertions**: Use descriptive assertion messages
5. **Resource Cleanup**: Clean up after tests

### **Performance Guidelines**
- Use table-driven tests for multiple scenarios
- Parallelize independent tests
- Cache expensive setup operations
- Use time-outs for external operations
- Profile slow tests

### **Security Guidelines**
- Never use real credentials in tests
- Validate all inputs and outputs
- Test error conditions and edge cases
- Use safe temporary directories
- Clean up sensitive data

### **Hardware Test Guidelines**
- Detect hardware capabilities before testing
- Provide fallbacks for missing hardware
- Test multiple hardware configurations
- Monitor resource usage
- Handle hardware-specific errors gracefully

---

## 🎯 Test Success Criteria

### **Definition of Done**
- [ ] All tests pass with 100% success rate
- [ ] Code coverage ≥ 80%
- [ ] Security scans pass (0 vulnerabilities)
- [ ] Performance benchmarks meet requirements
- [ ] Hardware tests pass on supported platforms
- [ ] Documentation is complete and up-to-date

### **Quality Gates**
- **Unit Tests**: 100% pass rate, ≥90% coverage
- **Integration Tests**: 100% pass rate on supported providers
- **E2E Tests**: 100% pass rate on supported workflows
- **Security Tests**: 0 critical/high vulnerabilities
- **Performance Tests**: Meet or exceed baseline metrics

### **Release Requirements**
```bash
# Complete release test suite
./run_tests.sh --all --coverage --benchmarks

# Verify all criteria
./verify_release_quality.sh
```

---

## 📞 Support & Contributing

### **Getting Help**
- **Issues**: Create GitHub issue with detailed description
- **Discussions**: Use GitHub Discussions for questions
- **Documentation**: Check test comments and this README
- **Logs**: Include relevant test logs in issue reports

### **Contributing Tests**
1. Follow test naming conventions
2. Add appropriate documentation
3. Ensure tests pass on CI
4. Update test coverage reports
5. Add integration with test runner

### **Test Infrastructure**
- **Test Runner**: `test_runner.go` - Go-based test execution
- **Shell Scripts**: `run_tests.sh` - Bash automation scripts
- **Configuration**: YAML-based test configuration
- **Reporting**: Markdown/HTML test reports
- **CI/CD**: GitHub Actions integration

---

**🧪 The HelixCode Local LLM test suite ensures production quality, security, and performance across all supported platforms and providers.**