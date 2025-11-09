# Local LLM Providers Documentation

## Overview

HelixCode now supports comprehensive integration with all major local LLM services. This enables you to run AI models locally on your own hardware while maintaining the same unified interface across all providers.

## Supported Local LLM Providers

### OpenAI-Compatible Providers

These providers follow the OpenAI API format and can be used with the unified `OpenAICompatibleProvider` implementation:

#### VLLM
- **Description**: High-throughput inference engine optimized for production workloads
- **Default Port**: 8000
- **Features**: PagedAttention, continuous batching, tensor parallelism
- **Models**: Supports all Transformer-based models
- **Streaming**: ✅ Supported
- **Vision**: ✅ Supported (for multimodal models)
- **Tools**: ✅ Supported

```yaml
providers:
  vllm:
    type: vllm
    endpoint: "http://localhost:8000"
    api_key: ""  # Optional
    models: ["llama-2-7b-chat-hf"]
    parameters:
      timeout: 30.0
      max_retries: 3
      streaming_support: true
```

#### LocalAI
- **Description**: Drop-in OpenAI replacement with extensive model format support
- **Default Port**: 8080
- **Features**: GGML/GPTQ models, image generation, embeddings, audio
- **Models**: LLaMA, GPT-J, Pythia, OPT, Stable Diffusion
- **Streaming**: ✅ Supported
- **Vision**: ✅ Supported
- **Tools**: ✅ Supported

```yaml
providers:
  localai:
    type: localai
    endpoint: "http://localhost:8080"
    models: ["gpt-3.5-turbo"]
    parameters:
      timeout: 30.0
      max_retries: 3
      streaming_support: true
```

#### FastChat
- **Description**: Training and serving platform for large language models
- **Default Port**: 7860
- **Features**: Vicuna models, model training, evaluation
- **Models**: Vicuna, custom trained models
- **Streaming**: ✅ Supported
- **Vision**: ✅ Supported
- **Tools**: ✅ Supported

```yaml
providers:
  fastchat:
    type: fastchat
    endpoint: "http://localhost:7860"
    models: ["vicuna-13b-v1.5"]
    parameters:
      timeout: 30.0
      max_retries: 3
      streaming_support: true
```

#### Text Generation WebUI (Oobabooga)
- **Description**: Popular Gradio-based interface with extensive features
- **Default Port**: 5000
- **Features**: Character cards, worldbuilding, extensions
- **Models**: LLaMA, GPT-J, Pythia, OPT, and many more
- **Streaming**: ✅ Supported
- **Vision**: ✅ Supported
- **Tools**: ✅ Supported

```yaml
providers:
  textgen:
    type: textgen
    endpoint: "http://localhost:5000"
    models: ["llama-2-7b-chat-hf"]
    parameters:
      timeout: 30.0
      max_retries: 3
      streaming_support: true
```

#### LM Studio
- **Description**: User-friendly desktop application with built-in model management
- **Default Port**: 1234
- **Features**: Model downloading, GPU acceleration, quantization
- **Models**: GGUF format models
- **Streaming**: ✅ Supported
- **Vision**: ✅ Supported
- **Tools**: ✅ Supported

```yaml
providers:
  lmstudio:
    type: lmstudio
    endpoint: "http://localhost:1234"
    models: ["local-model"]
    parameters:
      timeout: 30.0
      max_retries: 3
      streaming_support: true
```

#### Jan AI
- **Description**: Open-source local AI assistant with RAG capabilities
- **Default Port**: 1337
- **Features**: Built-in RAG, cross-platform desktop app
- **Models**: Various open-source models
- **Streaming**: ✅ Supported
- **Vision**: ✅ Supported
- **Tools**: ✅ Supported

```yaml
providers:
  jan:
    type: jan
    endpoint: "http://localhost:1337"
    models: ["jan-model"]
    parameters:
      timeout: 30.0
      max_retries: 3
      streaming_support: true
```

#### TabbyAPI
- **Description**: High-performance inference server with advanced quantization
- **Default Port**: 5000
- **Features**: ExLlamaV2, AutoGPTQ support, advanced quantization
- **Models**: Quantized Transformer models
- **Streaming**: ✅ Supported
- **Vision**: ✅ Supported
- **Tools**: ✅ Supported

```yaml
providers:
  tabbyapi:
    type: tabbyapi
    endpoint: "http://localhost:5000"
    models: ["tabby-model"]
    parameters:
      timeout: 30.0
      max_retries: 3
      streaming_support: true
```

#### MLX LLM
- **Description**: Apple Silicon optimized inference framework
- **Default Port**: 8080
- **Features**: Metal Performance Shaders optimization
- **Models**: Optimized for macOS/iOS
- **Streaming**: ✅ Supported
- **Vision**: ✅ Supported
- **Tools**: ✅ Supported

```yaml
providers:
  mlx:
    type: mlx
    endpoint: "http://localhost:8080"
    models: ["mlx-model"]
    parameters:
      timeout: 30.0
      max_retries: 3
      streaming_support: true
```

#### Mistral RS
- **Description**: High-performance Rust-based inference engine
- **Default Port**: 8080
- **Features**: Memory efficient, fast inference
- **Models**: Mistral models and compatible Transformers
- **Streaming**: ✅ Supported
- **Vision**: ✅ Supported
- **Tools**: ✅ Supported

```yaml
providers:
  mistralrs:
    type: mistralrs
    endpoint: "http://localhost:8080"
    models: ["mistral-model"]
    parameters:
      timeout: 30.0
      max_retries: 3
      streaming_support: true
```

### Specialized Providers

These providers have unique API formats and require specialized implementations:

#### GPT4All
- **Description**: CPU-focused inference for low-resource environments
- **Default Port**: 4891
- **Features**: Lightweight models, CPU optimization
- **Models**: Small to medium quantized models
- **Streaming**: ❌ Not supported
- **Vision**: ❌ Not supported
- **Tools**: ❌ Not supported

```yaml
providers:
  gpt4all:
    type: gpt4all
    endpoint: "http://localhost:4891"
    models: ["gpt4all-model"]
    parameters:
      timeout: 30.0
      max_retries: 3
      streaming_support: false
```

#### KoboldAI
- **Description**: Writing-focused interface with creative assistance
- **Default Port**: 5001
- **Features**: Story assistance, worldbuilding, creative writing
- **Models**: Various models optimized for creative writing
- **Streaming**: ✅ Supported
- **Vision**: ❌ Not supported
- **Tools**: ❌ Not supported

```yaml
providers:
  koboldai:
    type: koboldai
    endpoint: "http://localhost:5001"
    models: ["kobold-model"]
    parameters:
      timeout: 30.0
      max_retries: 3
      streaming_support: true
```

## Configuration

### Basic Configuration

All local providers share this basic configuration structure:

```yaml
providers:
  [provider_name]:
    type: [provider_type]
    endpoint: "http://localhost:[port]"
    api_key: ""  # Optional for most local providers
    models: ["model-name"]
    parameters:
      timeout: 30.0          # Request timeout in seconds
      max_retries: 3         # Maximum retry attempts
      streaming_support: true # Enable streaming responses
      headers: {}           # Custom HTTP headers
```

### Advanced Configuration

#### Custom Headers
```yaml
providers:
  vllm:
    type: vllm
    endpoint: "http://localhost:8000"
    parameters:
      headers:
        "Custom-Header": "value"
        "Another-Header": "another-value"
```

#### Custom Endpoints
```yaml
providers:
  vllm:
    type: vllm
    endpoint: "http://vllm.example.com:8080"
    parameters:
      model_endpoint: "/custom/v1/models"
      chat_endpoint: "/custom/v1/chat/completions"
```

## Performance Optimization

### Hardware Acceleration

#### GPU Support
- **CUDA**: Most providers support NVIDIA GPU acceleration
- **ROCm**: AMD GPU support in some providers
- **Metal**: Apple Silicon support in MLX and some others

#### Memory Optimization
- **Quantization**: Use GGUF, GPTQ, or AWQ formats
- **Context Size**: Configure appropriate context windows
- **Batch Processing**: Enable continuous batching when available

### Provider-Specific Optimizations

#### VLLM Optimization
```bash
# Enable tensor parallelism
vllm serve model-name --tensor-parallel-size 2

# Use optimized attention
vllm serve model-name --attention-backend flashinfer
```

#### LM Studio Optimization
- Enable GPU acceleration in settings
- Use appropriate quantization levels
- Configure context size based on VRAM

#### MLX Optimization
- Use Apple Silicon Metal Performance Shaders
- Enable unified memory optimization
- Configure batch sizes for optimal throughput

## Troubleshooting

### Common Issues

#### Provider Not Available
1. Check if the service is running
2. Verify the endpoint URL and port
3. Check network connectivity
4. Review provider logs

#### Model Loading Issues
1. Verify model format compatibility
2. Check available memory (RAM/VRAM)
3. Review model file permissions
4. Ensure model is downloaded correctly

#### Streaming Not Working
1. Check if provider supports streaming
2. Verify `streaming_support` is enabled
3. Check firewall/proxy settings
4. Review provider documentation

#### Performance Issues
1. Enable hardware acceleration
2. Use appropriate quantization
3. Optimize batch sizes
4. Monitor resource usage

### Health Checks

All providers support health checking:

```bash
# Check provider health
helixcode llm health --provider [provider_name]

# Check all providers
helixcode llm health --all

# Monitor provider status
helixcode llm monitor --provider [provider_name]
```

### Logs and Debugging

Enable detailed logging:

```yaml
logging:
  level: debug
  providers:
    [provider_name]: debug
```

## Security Considerations

### Network Security
- Use TLS/HTTPS when accessing remote providers
- Configure firewall rules appropriately
- Use VPN for secure remote access

### Model Security
- Verify model sources and integrity
- Use sandboxed environments when possible
- Monitor for malicious model behavior

### Data Privacy
- Local providers keep data on-premises
- Configure appropriate data retention policies
- Regular security updates for all components

## Integration Examples

### Multi-Provider Setup
```yaml
providers:
  primary:
    type: vllm
    endpoint: "http://localhost:8000"
    models: ["llama-2-7b-chat-hf"]
  backup:
    type: ollama
    endpoint: "http://localhost:11434"
    models: ["llama2"]
  cpu_fallback:
    type: gpt4all
    endpoint: "http://localhost:4891"
    models: ["gpt4all-model"]
```

### Provider Selection Logic
```go
// Select provider based on task requirements
criteria := ModelSelectionCriteria{
    TaskType: "code_generation",
    RequiredCapabilities: []ModelCapability{
        CapabilityCodeGeneration,
        CapabilityCodeAnalysis,
    },
    MaxTokens: 4096,
    QualityPreference: "balanced",
}

selectedProvider, err := providerManager.SelectOptimalProvider(criteria)
if err != nil {
    log.Printf("Provider selection failed: %v", err)
    return
}
```

### Load Balancing
```go
// Implement simple round-robin load balancing
providers := []Provider{vllmProvider, localAIProvider}
currentProvider := providers[time.Now().Unix()%int64(len(providers))]

response, err := currentProvider.Generate(ctx, request)
```

## API Reference

### Provider Interface

All local providers implement the standard `Provider` interface:

```go
type Provider interface {
    GetType() ProviderType
    GetName() string
    GetModels() []ModelInfo
    GetCapabilities() []ModelCapability
    Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error)
    GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error
    IsAvailable(ctx context.Context) bool
    GetHealth(ctx context.Context) (*ProviderHealth, error)
    Close() error
}
```

### Configuration Structure

```go
type ProviderConfigEntry struct {
    Type       ProviderType           `json:"type"`
    Endpoint   string                 `json:"endpoint"`
    APIKey     string                 `json:"api_key"`
    Models     []string               `json:"models"`
    Enabled    bool                   `json:"enabled"`
    Parameters map[string]interface{} `json:"parameters"`
}
```

## Migration Guide

### From Existing Provider
To migrate from an existing provider configuration:

1. Update the provider type
2. Verify endpoint and model names
3. Test with the new provider
4. Update any custom parameters

### Example Migration

#### Before (Ollama)
```yaml
providers:
  ollama:
    type: local
    endpoint: "http://localhost:11434"
    models: ["llama2"]
```

#### After (VLLM)
```yaml
providers:
  vllm:
    type: vllm
    endpoint: "http://localhost:8000"
    models: ["llama-2-7b-chat-hf"]
    parameters:
      timeout: 30.0
      max_retries: 3
      streaming_support: true
```

## Best Practices

### Provider Selection
1. Use VLLM for high-throughput production workloads
2. Use LM Studio for user-friendly desktop applications
3. Use MLX for Apple Silicon optimization
4. Use GPT4All for CPU-only environments
5. Use KoboldAI for creative writing tasks

### Configuration Management
1. Use environment variables for sensitive data
2. Maintain separate configurations for different environments
3. Regularly update provider configurations
4. Monitor provider performance and health

### Performance Monitoring
1. Track response times and token usage
2. Monitor resource utilization (CPU, GPU, memory)
3. Set up alerts for provider failures
4. Implement fallback mechanisms

### Testing
1. Test with integration test suite
2. Verify streaming functionality
3. Test error handling and recovery
4. Perform load testing for production deployments