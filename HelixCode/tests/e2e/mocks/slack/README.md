# Mock Slack Service

A lightweight HTTP mock service that simulates Slack API endpoints for testing purposes. Compatible with Slack's messaging and webhook APIs.

## Features

- **Slack-compatible API**: `/api/chat.postMessage`, webhooks
- **Message Storage**: In-memory storage with configurable capacity
- **Webhook Handling**: Incoming webhook simulation
- **Testing Endpoints**: Custom endpoints for inspecting stored messages/webhooks
- **Configurable Delays**: Simulate network latency
- **CORS Support**: Works with frontend applications

## Quick Start

### Build and Run

```bash
# Build
go build -o bin/mock-slack ./cmd/main.go

# Run
./bin/mock-slack
```

### Docker

```bash
# Build image
docker build -t mock-slack .

# Run container
docker run -p 8091:8091 mock-slack
```

### Docker Compose

```bash
docker-compose up mock-slack
```

## Configuration

Configure via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `MOCK_SLACK_PORT` | `8091` | Server port |
| `MOCK_SLACK_DELAY_MS` | `50` | Response delay in milliseconds |
| `MOCK_SLACK_LOGGING` | `true` | Enable request logging |
| `MOCK_SLACK_STORAGE_CAPACITY` | `1000` | Max messages/webhooks to store |
| `MOCK_SLACK_WEBHOOK_SECRET` | `test-webhook-secret` | Webhook secret (for future validation) |

## API Endpoints

### Health Check

```bash
GET /health
```

Returns service health status.

### Post Message

```bash
POST /api/chat.postMessage
```

Posts a message to a channel (Slack-compatible).

**Request:**
```json
{
  "channel": "#general",
  "text": "Hello, world!",
  "username": "TestBot",
  "icon_emoji": ":robot_face:",
  "thread_ts": "1234567890.123456"
}
```

**Response:**
```json
{
  "ok": true,
  "channel": "#general",
  "ts": "550e8400-e29b-41d4-a716-446655440000",
  "message": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "channel": "#general",
    "text": "Hello, world!",
    "username": "TestBot",
    "icon_emoji": ":robot_face:",
    "timestamp": "2025-01-07T12:00:00Z",
    "thread_ts": "1234567890.123456"
  }
}
```

### Get Messages (Testing Endpoint)

```bash
GET /api/messages?channel=#general
```

Retrieves stored messages for inspection during testing.

**Response:**
```json
{
  "ok": true,
  "channel": "#general",
  "messages": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "channel": "#general",
      "text": "Hello, world!",
      "timestamp": "2025-01-07T12:00:00Z"
    }
  ],
  "count": 1
}
```

### Clear Messages (Testing Endpoint)

```bash
DELETE /api/messages
```

Clears all stored messages.

**Response:**
```json
{
  "ok": true,
  "message": "All messages cleared"
}
```

### Incoming Webhook

```bash
POST /webhook/:id
POST /services/:service/:key/:id
```

Simulates Slack incoming webhooks.

**Request:**
```json
{
  "text": "Deployment completed successfully",
  "channel": "#deployments",
  "username": "Deploy Bot",
  "icon_emoji": ":rocket:"
}
```

**Response:**
```
ok
```

### Get Webhooks (Testing Endpoint)

```bash
GET /api/webhooks
```

Retrieves all received webhook payloads for inspection.

**Response:**
```json
{
  "ok": true,
  "webhooks": [
    {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "text": "Deployment completed successfully",
      "channel": "#deployments",
      "timestamp": "2025-01-07T12:00:00Z",
      "extra": {
        "text": "Deployment completed successfully",
        "channel": "#deployments",
        "username": "Deploy Bot",
        "icon_emoji": ":rocket:"
      }
    }
  ],
  "count": 1
}
```

### Clear Webhooks (Testing Endpoint)

```bash
DELETE /api/webhooks
```

Clears all stored webhook payloads.

## Testing with curl

```bash
# Post a message
curl -X POST http://localhost:8091/api/chat.postMessage \
  -H "Content-Type: application/json" \
  -d '{
    "channel": "#general",
    "text": "Hello from mock Slack!"
  }'

# Get all messages
curl http://localhost:8091/api/messages

# Get messages for specific channel
curl http://localhost:8091/api/messages?channel=#general

# Send to webhook
curl -X POST http://localhost:8091/webhook/test-webhook-id \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Webhook test",
    "channel": "#alerts"
  }'

# Get received webhooks
curl http://localhost:8091/api/webhooks

# Clear all messages
curl -X DELETE http://localhost:8091/api/messages

# Clear all webhooks
curl -X DELETE http://localhost:8091/api/webhooks

# Health check
curl http://localhost:8091/health
```

## Use Cases

- **E2E Testing**: Test Slack notification integrations without real API calls
- **Development**: Develop notification features without Slack workspace
- **CI/CD**: Run automated tests with predictable notification behavior
- **Inspection**: Verify what messages/webhooks your application sends
- **Offline Development**: Work without internet connectivity

## Testing Workflow

1. **Start the mock service**
   ```bash
   ./bin/mock-slack
   ```

2. **Configure your application** to use the mock endpoint:
   ```
   SLACK_WEBHOOK_URL=http://localhost:8091/webhook/test-id
   SLACK_API_URL=http://localhost:8091/api
   ```

3. **Run your tests** - notifications will be captured

4. **Inspect results** using the testing endpoints:
   ```bash
   curl http://localhost:8091/api/messages
   curl http://localhost:8091/api/webhooks
   ```

5. **Clear storage** between test runs:
   ```bash
   curl -X DELETE http://localhost:8091/api/messages
   curl -X DELETE http://localhost:8091/api/webhooks
   ```

## Architecture

```
mock-slack/
├── cmd/
│   └── main.go              # Server entry point
├── config/
│   └── config.go            # Configuration management
├── handlers/
│   ├── messages.go          # Message posting handler
│   └── webhooks.go          # Webhook handler
├── Dockerfile
└── README.md
```

## Storage

The service stores messages and webhooks in memory with a configurable capacity (default: 1000 items). When capacity is reached, the oldest items are automatically removed (FIFO). Storage is cleared when the service restarts.

## License

Part of the HelixCode E2E Testing Framework.
