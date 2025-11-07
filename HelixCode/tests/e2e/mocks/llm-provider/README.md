# Mock LLM Provider

A lightweight HTTP mock service that simulates LLM API endpoints for testing purposes. Compatible with OpenAI, Anthropic, and Ollama API formats.

## Features

- **OpenAI-compatible API**: `/v1/chat/completions`, `/v1/embeddings`, `/v1/models`
- **Multiple Provider Formats**: OpenAI, Anthropic (Claude), Ollama
- **Configurable Responses**: Pattern-based response matching
- **Realistic Token Counting**: Simulates token usage tracking
- **Configurable Delays**: Simulate network latency
- **Mock Embeddings**: Generate random embedding vectors
- **Multiple Models**: Supports various mock models (GPT-4, Claude, Llama, etc.)

## Quick Start

### Build and Run

```bash
# Build
go build -o bin/mock-llm-provider ./cmd/main.go

# Run
./bin/mock-llm-provider
```

### Docker

```bash
# Build image
docker build -t mock-llm-provider .

# Run container
docker run -p 8090:8090 mock-llm-provider
```

### Docker Compose

```bash
docker-compose up mock-llm-provider
```

## Configuration

Configure via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `MOCK_LLM_PORT` | `8090` | Server port |
| `MOCK_LLM_DELAY_MS` | `100` | Response delay in milliseconds |
| `MOCK_LLM_LOGGING` | `true` | Enable request logging |
| `MOCK_LLM_FIXTURES` | `./responses/fixtures.json` | Path to fixtures file |
| `MOCK_LLM_DEFAULT_MODEL` | `mock-gpt-4` | Default model name |

## API Endpoints

### Health Check

```bash
GET /health
```

Returns service health status.

### Chat Completions

```bash
POST /v1/chat/completions
POST /api/chat           # Anthropic-style
POST /api/generate       # Ollama-style
POST /completions        # Simple style
```

**Request:**
```json
{
  "model": "mock-gpt-4",
  "messages": [
    {"role": "user", "content": "Hello, how are you?"}
  ],
  "max_tokens": 100,
  "temperature": 0.7
}
```

**Response:**
```json
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "created": 1677610602,
  "model": "mock-gpt-4",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! I'm a mock LLM provider..."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 20,
    "total_tokens": 30
  }
}
```

### Embeddings

```bash
POST /v1/embeddings
```

**Request:**
```json
{
  "model": "mock-text-embedding-ada-002",
  "input": ["Hello world", "Test text"]
}
```

**Response:**
```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "embedding": [0.123, -0.456, ...],
      "index": 0
    }
  ],
  "model": "mock-text-embedding-ada-002",
  "usage": {
    "prompt_tokens": 5,
    "total_tokens": 5
  }
}
```

### Models

```bash
GET /v1/models              # List all models
GET /v1/models/:model       # Get specific model
```

## Response Patterns

The mock provider uses pattern matching to return contextual responses. Edit `responses/fixtures.json` to customize:

```json
{
  "patterns": {
    "hello": "Hello! I'm a mock LLM provider...",
    "weather": "I don't have real weather data...",
    "code": "Here's a code example..."
  }
}
```

## Supported Models

- `mock-gpt-4` (OpenAI GPT-4)
- `mock-gpt-3.5-turbo` (OpenAI GPT-3.5)
- `mock-claude-3` (Anthropic Claude 3)
- `mock-llama-3-8b` (Meta Llama 3)
- `mock-mixtral-8x7b` (Mistral Mixtral)
- `mock-text-embedding-ada-002` (OpenAI Embeddings)

## Testing with curl

```bash
# Chat completion
curl -X POST http://localhost:8090/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "mock-gpt-4",
    "messages": [{"role": "user", "content": "Hello"}]
  }'

# Embeddings
curl -X POST http://localhost:8090/v1/embeddings \
  -H "Content-Type: application/json" \
  -d '{
    "model": "mock-text-embedding-ada-002",
    "input": ["Hello world"]
  }'

# List models
curl http://localhost:8090/v1/models
```

## Use Cases

- **E2E Testing**: Test LLM integrations without real API calls
- **Development**: Develop LLM features without API costs
- **CI/CD**: Run automated tests with predictable responses
- **Load Testing**: Test system behavior under LLM load
- **Offline Development**: Work without internet connectivity

## Architecture

```
mock-llm-provider/
├── cmd/
│   └── main.go              # Server entry point
├── config/
│   └── config.go            # Configuration management
├── handlers/
│   ├── completions.go       # Chat completions handler
│   ├── embeddings.go        # Embeddings handler
│   └── models.go            # Models handler
├── responses/
│   ├── fixtures.json        # Response fixtures
│   └── templates.go         # Template management
├── Dockerfile
└── README.md
```

## License

Part of the HelixCode E2E Testing Framework.
