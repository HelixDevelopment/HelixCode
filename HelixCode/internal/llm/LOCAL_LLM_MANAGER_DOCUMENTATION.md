# Local LLM Provider Management System Documentation

## Overview

The Local LLM Provider Management System is a comprehensive solution that automatically clones, builds, configures, and manages all major local LLM providers. This system transforms local AI inference from a complex manual setup into a seamless, automated experience.

## 🚀 Key Features

- **🔧 Automated Installation**: Clone and build all local LLM providers automatically
- **⚙️ Unified Management**: Single interface to manage all providers
- **🔄 Dynamic Provider Discovery**: Automatically detect and register running providers
- **🛡️ Health Monitoring**: Continuous health checks and automatic recovery
- **📊 Performance Optimization**: Intelligent load balancing and provider selection
- **🌐 Cross-Platform Support**: Linux, macOS, and Windows compatibility
- **📦 Dependency Management**: Automatic resolution of system dependencies
- **🔍 Zero-Configuration**: Works out of the box with sensible defaults

## 📋 Supported Local LLM Providers

### OpenAI-Compatible Providers (11 Providers)

| Provider | Description | Default Port | Key Features | Status |
|----------|-------------|---------------|--------------|--------|
| **VLLM** | High-throughput inference engine | 8000 | PagedAttention, continuous batching, tensor parallelism | ✅ Production Ready |
| **LocalAI** | Drop-in OpenAI replacement | 8080 | GGML/GPTQ models, image generation, embeddings | ✅ Production Ready |
| **FastChat** | Training and serving platform | 7860 | Vicuna models, model training, evaluation | ✅ Production Ready |
| **Text Generation WebUI** | Popular Gradio interface | 5000 | Character cards, worldbuilding, extensions | ✅ Production Ready |
| **LM Studio** | User-friendly desktop app | 1234 | Built-in model management, GPU acceleration | ✅ Production Ready |
| **Jan AI** | Open-source local AI assistant | 1337 | Built-in RAG, cross-platform desktop | ✅ Production Ready |
| **KoboldAI** | Writing-focused interface | 5001 | Creative writing, story generation | ✅ Production Ready |
| **GPT4All** | CPU-focused inference | 4891 | Lightweight models, CPU optimization | ✅ Production Ready |
| **TabbyAPI** | High-performance inference server | 5000 | ExLlamaV2, AutoGPTQ, advanced quantization | ✅ Production Ready |
| **MLX LLM** | Apple Silicon optimized | 8080 | Metal Performance Shaders, native optimization | ✅ Production Ready |
| **Mistral RS** | Rust-based inference engine | 8080 | Memory efficient, fast inference | ✅ Production Ready |

### Specialized Providers

| Provider | Description | Default Port | API Format | Streaming | Vision |
|----------|-------------|---------------|------------|----------|---------|
| **KoboldAI** | Creative writing assistant | 5001 | Custom API | ✅ | ❌ |
| **GPT4All** | CPU-optimized inference | 4891 | OpenAI-like | ❌ | ❌ |

## 🏗️ Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────────┐
│                    Local LLM Manager                         │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │
│  │   Builder   │  │   Monitor   │  │  Selector   │       │
│  │             │  │             │  │             │       │
│  │ • Clone     │  │ • Health    │  │ • Routing   │       │
│  │ • Build     │  │ • Status    │  │ • Balance   │       │
│  │ • Configure │  │ • Recovery  │  │ • Fallback  │       │
│  └─────────────┘  └─────────────┘  └─────────────┘       │
├─────────────────────────────────────────────────────────────────┤
│                      Provider Registry                        │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │
│  │  VLLM       │  │  LocalAI    │  │  FastChat   │  ...  │
│  │  TextGen    │  │  LMStudio   │  │  Jan AI     │       │
│  │  KoboldAI   │  │  GPT4All    │  │  TabbyAPI   │       │
│  └─────────────┘  └─────────────┘  └─────────────┘       │
├─────────────────────────────────────────────────────────────────┤
│                      Resource Layer                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │
│  │    Bin      │  │    Config   │  │     Data    │       │
│  │ Directory   │  │ Directory   │  │ Directory   │       │
│  │             │  │             │  │             │       │
│  │ • Executables│  │ • Settings  │  │ • Models    │       │
│  │ • Scripts   │  │ • Templates  │  │ • Cache     │       │
│  └─────────────┘  └─────────────┘  └─────────────┘       │
└─────────────────────────────────────────────────────────────────┘
```

### Directory Structure

```
~/.helixcode/local-llm/
├── bin/                          # Provider executables and scripts
│   ├── vllm.sh                  # VLLM startup script
│   ├── localai.sh                # LocalAI startup script
│   ├── fastchat.sh               # FastChat startup script
│   └── ...                       # Other provider scripts
├── config/                       # Provider configurations
│   ├── vllm/                    # VLLM config files
│   ├── localai/                  # LocalAI config files
│   └── ...                       # Other provider configs
├── data/                         # Provider repositories and data
│   ├── vllm/                    # VLLM cloned repository
│   ├── localai/                  # LocalAI cloned repository
│   ├── textgen/                  # TextGen cloned repository
│   └── ...                       # Other provider data
├── models/                       # Downloaded models (optional)
│   ├── llama2/                   # LLaMA 2 models
│   ├── vicuna/                   # Vicuna models
│   └── ...                       # Other models
├── logs/                         # Provider logs
│   ├── vllm.log                 # VLLM logs
│   ├── localai.log               # LocalAI logs
│   └── ...                       # Other provider logs
└── cache/                        # Build and download cache
    ├── pip/                      # Python package cache
    ├── npm/                      # Node.js package cache
    └── docker/                   # Docker image cache
```

## 🚀 Getting Started

### Prerequisites

#### System Requirements
- **OS**: Linux (Ubuntu 20.04+), macOS (11.0+), Windows 10+
- **CPU**: 64-bit processor, 4+ cores recommended
- **Memory**: 8GB RAM minimum, 16GB+ recommended
- **Storage**: 50GB free space for providers and models
- **GPU**: NVIDIA GPU with CUDA 11.0+ (optional but recommended)

#### Required Dependencies

**Linux (Ubuntu/Debian)**:
```bash
# Base dependencies
sudo apt update
sudo apt install -y git curl wget make cmake build-essential

# Python and Node.js
sudo apt install -y python3 python3-pip nodejs npm

# GPU support (optional)
sudo apt install -y nvidia-cuda-toolkit nvidia-driver-535
```

**macOS**:
```bash
# Install Homebrew if not present
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install dependencies
brew install git curl wget make cmake python3 node npm

# GPU support (Apple Silicon only)
# Metal drivers included in macOS
```

**Windows**:
```powershell
# Install Chocolatey if not present
Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))

# Install dependencies
choco install git curl wget make cmake python3 nodejs

# GPU support (optional)
# Install NVIDIA CUDA Toolkit from NVIDIA website
```

### Quick Installation

#### 1. Initialize Local LLM Manager

```go
package main

import (
    "context"
    "log"
    "github.com/helixcode/internal/llm"
)

func main() {
    // Create manager instance
    manager := llm.NewLocalLLMManager("")
    
    // Initialize (clone, build, configure)
    ctx := context.Background()
    if err := manager.Initialize(ctx); err != nil {
        log.Fatalf("Failed to initialize manager: %v", err)
    }
    
    log.Println("✅ Local LLM Manager initialized successfully!")
}
```

#### 2. Start All Providers

```go
// Start all providers
if err := manager.StartAllProviders(ctx); err != nil {
    log.Fatalf("Failed to start providers: %v", err)
}

log.Println("✅ All providers started successfully!")
```

#### 3. Check Provider Status

```go
// Get provider status
status := manager.GetProviderStatus(ctx)
for name, provider := range status {
    log.Printf("%s: %s (Port: %d)", name, provider.Status, provider.DefaultPort)
}
```

#### 4. Get Running Provider Endpoints

```go
// Get running endpoints for HelixCode integration
endpoints := manager.GetRunningProviders(ctx)
for _, endpoint := range endpoints {
    log.Printf("Available endpoint: %s", endpoint)
}
```

## ⚙️ Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|----------|
| `HELIX_LOCAL_LLM_DIR` | Base directory for local LLM providers | `~/.helixcode/local-llm` |
| `HELIX_LOCAL_LLM_AUTO_START` | Auto-start providers on initialization | `true` |
| `HELIX_LOCAL_LLM_TIMEOUT` | Provider startup timeout (seconds) | `60` |
| `HELIX_LOCAL_LLM_HEALTH_INTERVAL` | Health check interval (seconds) | `30` |

### Provider Configuration

Each provider can be configured through individual config files:

#### VLLM Configuration (`config/vllm/config.yaml`):
```yaml
server:
  host: "127.0.0.1"
  port: 8000
  max_num_seqs: 256
  max_model_len: 4096

models:
  - name: "llama-2-7b-chat-hf"
    model: "meta-llama/Llama-2-7b-chat-hf"
    tokenizer: "meta-llama/Llama-2-7b-chat-hf"
    dtype: "half"
    gpu_memory_utilization: 0.9
```

#### LocalAI Configuration (`config/localai/config.yaml`):
```yaml
address: "127.0.0.1:8080"
webui: true
galleries: native
models-path: "./models"

models:
  - name: "gpt-3.5-turbo"
    backend: "llama-cpp"
    context_size: 4096
    f16: true
```

### HelixCode Integration Configuration

Update `config/config.yaml` to include local providers:

```yaml
llm:
  default_provider: "local"
  providers:
    # Existing providers
    openai: ""
    anthropic: ""
    
    # Auto-discovered local providers
    local_auto_discovery: true
    local_health_check_interval: 30
    
    # Manual configuration (optional)
    vllm:
      type: vllm
      endpoint: "http://localhost:8000"
      models: ["llama-2-7b-chat-hf"]
      enabled: true
    
    localai:
      type: localai
      endpoint: "http://localhost:8080"
      models: ["gpt-3.5-turbo"]
      enabled: true
```

## 🔧 Advanced Usage

### Selective Provider Installation

Install only specific providers to save space and time:

```go
// Install only VLLM and LocalAI
selectedProviders := []string{"vllm", "localai"}
for _, name := range selectedProviders {
    if definition, exists := llm.ProviderDefinitions[name]; exists {
        provider := &llm.LocalLLMProvider{
            Name:         definition.Name,
            Repository:   definition.Repository,
            Version:      definition.Version,
            DefaultPort:  definition.DefaultPort,
            Dependencies: definition.Dependencies,
            BuildScript:  definition.BuildScript,
            StartupCmd:   definition.StartupCmd,
            Environment:  definition.Environment,
        }
        
        if err := manager.InstallProvider(ctx, provider); err != nil {
            log.Printf("Failed to install %s: %v", name, err)
        }
    }
}
```

### Custom Provider Definition

Add support for new providers:

```go
customProvider := &llm.LocalLLMProvider{
    Name:        "Custom LLM",
    Repository:  "https://github.com/user/custom-llm.git",
    Version:     "main",
    Description: "Custom local LLM provider",
    DefaultPort: 9999,
    Dependencies: []string{"git", "python3", "pip"},
    BuildScript: "pip install -e .",
    StartupCmd:  []string{"python3", "server.py"},
    Environment: map[string]string{
        "HOST": "127.0.0.1",
        "PORT": "9999",
    },
}

if err := manager.InstallProvider(ctx, customProvider); err != nil {
    log.Printf("Failed to install custom provider: %v", err)
}
```

### Provider Load Balancing

Implement intelligent load balancing across running providers:

```go
type LoadBalancer struct {
    manager *llm.LocalLLMManager
    current int
    mutex   sync.Mutex
}

func (lb *LoadBalancer) GetBestEndpoint(ctx context.Context) string {
    running := lb.manager.GetRunningProviders(ctx)
    if len(running) == 0 {
        return ""
    }
    
    // Simple round-robin
    lb.mutex.Lock()
    defer lb.mutex.Unlock()
    
    endpoint := running[lb.current%len(running)]
    lb.current++
    
    return endpoint
}
```

### Health Monitoring with Alerts

Set up comprehensive health monitoring:

```go
func MonitorProviders(ctx context.Context, manager *llm.LocalLLMManager) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            status := manager.GetProviderStatus(ctx)
            for name, provider := range status {
                switch provider.Status {
                case "failed", "unhealthy":
                    log.Printf("🚨 Alert: Provider %s is %s", name, provider.Status)
                    // Send notification, attempt restart, etc.
                    manager.RestartProvider(ctx, name)
                case "starting":
                    log.Printf("⏳ Provider %s is starting...", name)
                case "running":
                    log.Printf("✅ Provider %s is healthy", name)
                }
            }
        }
    }
}
```

## 🧪 Testing

### Unit Tests

```bash
# Run unit tests for local LLM manager
go test ./internal/llm -run TestLocalLLMManager -v
```

### Integration Tests

```bash
# Run integration tests (requires actual provider installation)
go test ./internal/llm -run TestLocalLLMManagerIntegration -v -tags=integration
```

### Provider Health Tests

```bash
# Test provider health and functionality
go test ./internal/llm -run TestProviderHealth -v
```

## 🐛 Troubleshooting

### Common Issues

#### 1. Dependency Installation Failed

**Problem**: `missing dependencies: git, cmake, gcc`

**Solution**:
```bash
# Linux
sudo apt install git cmake build-essential

# macOS
brew install git cmake

# Windows
choco install git cmake
```

#### 2. Provider Build Failed

**Problem**: `build failed: command not found: pip`

**Solution**:
```bash
# Install pip
python3 -m ensurepip --upgrade
python3 -m pip install --upgrade pip
```

#### 3. Provider Failed to Start

**Problem**: `provider failed to start: bind: address already in use`

**Solution**:
```bash
# Check what's using the port
lsof -i :8000

# Kill the process
kill -9 <PID>

# Or use a different port
export VLLM_PORT=8001
```

#### 4. Provider Unhealthy

**Problem**: Health checks failing

**Solution**:
```bash
# Check provider logs
tail -f ~/.helixcode/local-llm/logs/vllm.log

# Verify provider is accessible
curl http://localhost:8000/health

# Check system resources
top -p <PID>
```

#### 5. Out of Memory

**Problem**: Provider crashes due to insufficient memory

**Solution**:
```yaml
# Reduce model size in config
models:
  - name: "llama-2-7b-chat-hf"  # Use smaller model
    gpu_memory_utilization: 0.7  # Reduce GPU memory usage
    max_model_len: 2048  # Reduce context size
```

### Debug Mode

Enable verbose logging for troubleshooting:

```go
// Set debug logging
logger := log.New(os.Stdout, "[DEBUG] ", log.LstdFlags|log.Lshortfile)
manager.SetLogger(logger)
```

### Log Analysis

Analyze provider logs for issues:

```bash
# View all provider logs
tail -f ~/.helixcode/local-llm/logs/*.log

# Filter specific provider logs
grep "ERROR" ~/.helixcode/local-llm/logs/vllm.log

# Monitor resource usage
watch -n 1 'ps aux | grep python'
```

## 📊 Performance Optimization

### GPU Acceleration

#### NVIDIA CUDA Setup
```bash
# Install CUDA toolkit
sudo apt install nvidia-cuda-toolkit

# Set CUDA paths
export PATH=/usr/local/cuda/bin:$PATH
export LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH

# Verify installation
nvidia-smi
nvcc --version
```

#### Apple Silicon (Metal)
```bash
# Metal support included in macOS
# Install MLX for optimal performance
pip install mlx
```

### Memory Optimization

#### Model Quantization
```yaml
# Use quantized models
models:
  - name: "llama-2-7b-chat-hf"
    backend: "llama-cpp"
    model: "llama-2-7b-chat-hf.Q4_K_M.gguf"  # Quantized model
    f16: true
```

#### Context Size Management
```yaml
# Optimize context size for available memory
models:
  - name: "llama-2-7b-chat-hf"
    context_size: 2048  # Reduce for low memory
    max_tokens: 1024
```

### Batch Processing

#### Enable Continuous Batching
```yaml
# VLLM continuous batching
server:
  max_num_seqs: 256
  use_v2_block_manager: true
```

## 🔄 Updates and Maintenance

### Update Providers

```go
// Update all providers to latest versions
for name := range manager.Providers() {
    if err := manager.UpdateProvider(ctx, name); err != nil {
        log.Printf("Failed to update %s: %v", name, err)
    }
}
```

### Cleanup Old Resources

```go
// Clean up old models and cache
if err := manager.Cleanup(ctx); err != nil {
    log.Printf("Cleanup failed: %v", err)
}
```

### Backup Configuration

```bash
# Backup provider configurations
cp -r ~/.helixcode/local-llm/config ~/helix-backup/

# Backup downloaded models
cp -r ~/.helixcode/local-llm/models ~/helix-backup/
```

## 📚 API Reference

### LocalLLMManager

#### Methods

| Method | Description | Parameters | Returns |
|--------|-------------|-------------|---------|
| `Initialize(ctx)` | Initialize manager and install providers | `context.Context` | `error` |
| `InstallProvider(ctx, provider)` | Install a specific provider | `context.Context`, `*LocalLLMProvider` | `error` |
| `StartProvider(ctx, name)` | Start a specific provider | `context.Context`, `string` | `error` |
| `StopProvider(ctx, name)` | Stop a specific provider | `context.Context`, `string` | `error` |
| `StartAllProviders(ctx)` | Start all installed providers | `context.Context` | `error` |
| `StopAllProviders(ctx)` | Stop all running providers | `context.Context` | `error` |
| `GetProviderStatus(ctx)` | Get status of all providers | `context.Context` | `map[string]*LocalLLMProvider` |
| `GetRunningProviders(ctx)` | Get endpoints of running providers | `context.Context` | `[]string` |
| `Cleanup(ctx)` | Clean up all resources | `context.Context` | `error` |

### LocalLLMProvider

#### Fields

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Human-readable provider name |
| `Repository` | `string` | Git repository URL |
| `Version` | `string` | Git branch/tag/commit |
| `Description` | `string` | Provider description |
| `DefaultPort` | `int` | Default HTTP port |
| `BinaryPath` | `string` | Path to provider binary |
| `ConfigPath` | `string` | Path to configuration directory |
| `DataPath` | `string` | Path to data directory |
| `Status` | `string` | Current status (installed, running, etc.) |
| `Process` | `*os.Process` | Running process handle |
| `HealthURL` | `string` | Health check URL |
| `Dependencies` | `[]string` | Required system dependencies |
| `BuildScript` | `string` | Build/installation script |
| `StartupCmd` | `[]string` | Startup command |
| `Environment` | `map[string]string` | Environment variables |
| `LastCheck` | `time.Time` | Last health check timestamp |

## 🤝 Contributing

### Adding New Providers

1. **Add Provider Definition**:
```go
var providerDefinitions = map[string]*LocalLLMProvider{
    "newprovider": {
        Name:        "New Provider",
        Repository:  "https://github.com/user/newprovider.git",
        Version:     "main",
        Description: "Description of new provider",
        DefaultPort: 9999,
        Dependencies: []string{"git", "python3", "pip"},
        BuildScript: "pip install -e .",
        StartupCmd:  []string{"python3", "server.py"},
        Environment: map[string]string{
            "HOST": "127.0.0.1",
            "PORT": "9999",
        },
    },
}
```

2. **Add Tests**:
```go
func TestNewProvider(t *testing.T) {
    provider := providerDefinitions["newprovider"]
    assert.NotNil(t, provider)
    assert.Equal(t, "New Provider", provider.Name)
    assert.Equal(t, 9999, provider.DefaultPort)
}
```

3. **Update Documentation**:
   - Add provider to this documentation
   - Update README.md
   - Add configuration examples

### Running Tests

```bash
# Run all tests
go test ./internal/llm/... -v

# Run with race detection
go test ./internal/llm/... -race -v

# Run integration tests
go test ./internal/llm/... -tags=integration -v
```

## 📄 License

This Local LLM Provider Management System is part of the HelixCode project and is licensed under the MIT License. See the [LICENSE](https://github.com/helixcode/helixcode/blob/main/LICENSE) file for details.

## 🆘 Support

- **Issues**: [GitHub Issues](https://github.com/helixcode/helixcode/issues)
- **Discussions**: [GitHub Discussions](https://github.com/helixcode/helixcode/discussions)
- **Documentation**: [HelixCode Docs](https://docs.helixcode.dev)

---

**🎉 Congratulations!** You now have a complete, automated local LLM provider management system that handles 11+ providers with zero configuration required. Start building amazing AI-powered applications today! 🚀