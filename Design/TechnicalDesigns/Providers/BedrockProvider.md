# AWS Bedrock Provider - Technical Design

## Overview

The AWS Bedrock Provider enables access to foundation models through Amazon Bedrock, including Claude (Anthropic), Titan (Amazon), Jurassic (AI21), Command (Cohere), and others. This provider integrates with AWS SDK v2 for Go and supports IAM authentication, cross-region inference, streaming responses, and model invocation.

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      BedrockProvider                            │
├─────────────────────────────────────────────────────────────────┤
│  - config: ProviderConfigEntry                                  │
│  - awsConfig: aws.Config                                        │
│  - bedrockClient: *bedrockruntime.Client                        │
│  - models: []ModelInfo                                          │
│  - region: string                                               │
│  - crossRegionInference: bool                                   │
│  - inferenceProfileArn: string (optional)                       │
├─────────────────────────────────────────────────────────────────┤
│  Methods:                                                       │
│  + NewBedrockProvider(config) (*BedrockProvider, error)         │
│  + Generate(ctx, request) (*LLMResponse, error)                 │
│  + GenerateStream(ctx, request, ch) error                       │
│  + InvokeModel(ctx, modelId, body) ([]byte, error)              │
│  + InvokeModelWithResponseStream(ctx, modelId, body) (Stream)   │
│  + GetType() ProviderType                                       │
│  + GetName() string                                             │
│  + GetModels() []ModelInfo                                      │
│  + GetCapabilities() []ModelCapability                          │
│  + IsAvailable(ctx) bool                                        │
│  + GetHealth(ctx) (*ProviderHealth, error)                      │
│  + Close() error                                                │
└─────────────────────────────────────────────────────────────────┘
              │
              │ Uses AWS SDK v2
              ▼
┌─────────────────────────────────────────────────────────────────┐
│              AWS Bedrock Runtime API                            │
├─────────────────────────────────────────────────────────────────┤
│  - InvokeModel (synchronous)                                    │
│  - InvokeModelWithResponseStream (streaming)                    │
│  - Cross-Region Inference Profiles                              │
│  - Model-specific request/response formats                      │
└─────────────────────────────────────────────────────────────────┘
```

### Component Breakdown

**BedrockProvider Struct**:
```go
type BedrockProvider struct {
    config                ProviderConfigEntry
    awsConfig             aws.Config
    bedrockClient         *bedrockruntime.Client
    models                []ModelInfo
    region                string
    crossRegionInference  bool
    inferenceProfileArn   string
    lastHealth            *ProviderHealth
}
```

**AWS Configuration Options**:
- IAM role-based authentication
- Access key authentication
- Session token support
- Assumed role authentication
- Cross-region inference profiles
- Custom endpoint configuration

## Interface Implementation

Implements the `Provider` interface from `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/provider.go`:

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

## Model Definitions

### Supported Model Families

**1. Claude (Anthropic) via Bedrock**
```go
{
    Name:           "anthropic.claude-4-sonnet-20250514-v1:0",
    Provider:       ProviderTypeBedrock,
    ContextSize:    200000,
    MaxTokens:      50000,
    Capabilities:   allCapabilities,
    SupportsTools:  true,
    SupportsVision: true,
    Description:    "Claude 4 Sonnet via AWS Bedrock - Latest flagship model",
},
{
    Name:           "anthropic.claude-3-7-sonnet-20250219-v1:0",
    Provider:       ProviderTypeBedrock,
    ContextSize:    200000,
    MaxTokens:      50000,
    Capabilities:   allCapabilities,
    SupportsTools:  true,
    SupportsVision: true,
    Description:    "Claude 3.7 Sonnet via AWS Bedrock",
},
{
    Name:           "anthropic.claude-3-5-sonnet-20241022-v2:0",
    Provider:       ProviderTypeBedrock,
    ContextSize:    200000,
    MaxTokens:      8192,
    Capabilities:   allCapabilities,
    SupportsTools:  true,
    SupportsVision: true,
    Description:    "Claude 3.5 Sonnet v2 via AWS Bedrock",
},
```

**2. Amazon Titan**
```go
{
    Name:           "amazon.titan-text-premier-v1:0",
    Provider:       ProviderTypeBedrock,
    ContextSize:    32000,
    MaxTokens:      8192,
    Capabilities:   textCapabilities,
    SupportsTools:  false,
    SupportsVision: false,
    Description:    "Amazon Titan Text Premier - Enterprise text model",
},
{
    Name:           "amazon.titan-text-express-v1",
    Provider:       ProviderTypeBedrock,
    ContextSize:    8000,
    MaxTokens:      8192,
    Capabilities:   textCapabilities,
    SupportsTools:  false,
    SupportsVision: false,
    Description:    "Amazon Titan Text Express - Fast text generation",
},
```

**3. AI21 Jurassic**
```go
{
    Name:           "ai21.jamba-1-5-large-v1:0",
    Provider:       ProviderTypeBedrock,
    ContextSize:    256000,
    MaxTokens:      4096,
    Capabilities:   allCapabilities,
    SupportsTools:  true,
    SupportsVision: false,
    Description:    "AI21 Jamba 1.5 Large - Hybrid SSM-Transformer model",
},
```

**4. Cohere Command**
```go
{
    Name:           "cohere.command-r-plus-v1:0",
    Provider:       ProviderTypeBedrock,
    ContextSize:    128000,
    MaxTokens:      4000,
    Capabilities:   allCapabilities,
    SupportsTools:  true,
    SupportsVision: false,
    Description:    "Cohere Command R+ - RAG-optimized model",
},
```

**5. Meta Llama**
```go
{
    Name:           "meta.llama3-3-70b-instruct-v1:0",
    Provider:       ProviderTypeBedrock,
    ContextSize:    128000,
    MaxTokens:      8192,
    Capabilities:   allCapabilities,
    SupportsTools:  true,
    SupportsVision: false,
    Description:    "Meta Llama 3.3 70B Instruct via Bedrock",
},
```

## Request/Response Flow

### Non-Streaming Flow

```
Client Request
    │
    ▼
BedrockProvider.Generate(ctx, request)
    │
    ├─> buildBedrockRequest(request)
    │   └─> Convert to model-specific format
    │       ├─> Claude: Anthropic format
    │       ├─> Titan: Amazon format
    │       ├─> Jurassic: AI21 format
    │       └─> Command: Cohere format
    │
    ├─> InvokeModel(ctx, modelId, requestBody)
    │   └─> bedrockClient.InvokeModel(&bedrockruntime.InvokeModelInput{
    │           ModelId: aws.String(modelId),
    │           Body: requestBody,
    │           Accept: aws.String("application/json"),
    │           ContentType: aws.String("application/json"),
    │       })
    │
    ├─> parseBedrockResponse(responseBody, modelFamily)
    │   └─> Convert from model-specific format to LLMResponse
    │
    └─> Return LLMResponse
```

### Streaming Flow

```
Client Request
    │
    ▼
BedrockProvider.GenerateStream(ctx, request, ch)
    │
    ├─> buildBedrockRequest(request)
    │
    ├─> InvokeModelWithResponseStream(ctx, modelId, requestBody)
    │   └─> bedrockClient.InvokeModelWithResponseStream(&bedrockruntime.InvokeModelWithResponseStreamInput{
    │           ModelId: aws.String(modelId),
    │           Body: requestBody,
    │           Accept: aws.String("application/json"),
    │           ContentType: aws.String("application/json"),
    │       })
    │
    ├─> stream.Events()
    │   └─> For each event:
    │       ├─> PayloadPart: Parse chunk, send to channel
    │       ├─> ModelStreamError: Handle error
    │       └─> Complete: Send final response
    │
    └─> Close channel
```

## Authentication Mechanisms

### 1. IAM Role Authentication (Recommended for EC2/ECS/Lambda)

```go
func NewBedrockProvider(config ProviderConfigEntry) (*BedrockProvider, error) {
    ctx := context.Background()

    // Load default AWS configuration (uses IAM role if available)
    cfg, err := awsconfig.LoadDefaultConfig(ctx,
        awsconfig.WithRegion(getRegion(config)),
    )
    if err != nil {
        return nil, fmt.Errorf("unable to load AWS config: %v", err)
    }

    client := bedrockruntime.NewFromConfig(cfg)
    // ...
}
```

### 2. Access Key Authentication

```go
// Environment variables:
// AWS_ACCESS_KEY_ID
// AWS_SECRET_ACCESS_KEY
// AWS_REGION

cfg, err := awsconfig.LoadDefaultConfig(ctx,
    awsconfig.WithRegion(region),
    awsconfig.WithCredentialsProvider(
        credentials.NewStaticCredentialsProvider(
            os.Getenv("AWS_ACCESS_KEY_ID"),
            os.Getenv("AWS_SECRET_ACCESS_KEY"),
            "", // session token (optional)
        ),
    ),
)
```

### 3. Assumed Role Authentication

```go
cfg, err := awsconfig.LoadDefaultConfig(ctx,
    awsconfig.WithRegion(region),
)

stsClient := sts.NewFromConfig(cfg)
assumeRoleProvider := stscreds.NewAssumeRoleProvider(
    stsClient,
    roleARN,
    func(o *stscreds.AssumeRoleOptions) {
        o.RoleSessionName = "helixcode-session"
        o.Duration = 1 * time.Hour
    },
)

cfg.Credentials = aws.NewCredentialsCache(assumeRoleProvider)
client := bedrockruntime.NewFromConfig(cfg)
```

### 4. Cross-Region Inference

```go
// Use inference profile ARN for cross-region routing
inferenceProfileArn := config.Parameters["inference_profile_arn"].(string)

// Example: "arn:aws:bedrock:us-east-1:123456789012:inference-profile/us.anthropic.claude-3-5-sonnet-20241022-v2:0"

input := &bedrockruntime.InvokeModelInput{
    ModelId: aws.String(inferenceProfileArn),
    Body:    requestBody,
}
```

## Error Handling Strategy

### AWS-Specific Error Types

```go
type BedrockError struct {
    Type    string
    Message string
    Code    string
}

func handleBedrockError(err error) error {
    if err == nil {
        return nil
    }

    // Check for AWS SDK specific errors
    var apiErr smithy.APIError
    if errors.As(err, &apiErr) {
        switch apiErr.ErrorCode() {
        case "ThrottlingException":
            return ErrRateLimited
        case "ModelTimeoutException":
            return fmt.Errorf("model timeout: %w", err)
        case "ModelNotReadyException":
            return fmt.Errorf("model not ready: %w", err)
        case "ModelErrorException":
            return fmt.Errorf("model error: %w", err)
        case "ValidationException":
            return ErrInvalidRequest
        case "AccessDeniedException":
            return fmt.Errorf("access denied - check IAM permissions: %w", err)
        case "ResourceNotFoundException":
            return ErrModelNotFound
        case "ServiceQuotaExceededException":
            return fmt.Errorf("service quota exceeded: %w", err)
        default:
            return fmt.Errorf("bedrock API error: %s - %w", apiErr.ErrorCode(), err)
        }
    }

    return err
}
```

### Retry Strategy with Exponential Backoff

```go
func (bp *BedrockProvider) invokeModelWithRetry(ctx context.Context, input *bedrockruntime.InvokeModelInput) (*bedrockruntime.InvokeModelOutput, error) {
    maxRetries := 3
    baseDelay := 1 * time.Second

    for attempt := 0; attempt < maxRetries; attempt++ {
        output, err := bp.bedrockClient.InvokeModel(ctx, input)

        if err == nil {
            return output, nil
        }

        // Check if error is retryable
        if !isRetryableError(err) {
            return nil, handleBedrockError(err)
        }

        // Exponential backoff
        if attempt < maxRetries-1 {
            delay := baseDelay * time.Duration(1<<uint(attempt))
            log.Printf("Retrying Bedrock request after %v (attempt %d/%d)", delay, attempt+1, maxRetries)

            select {
            case <-time.After(delay):
                continue
            case <-ctx.Done():
                return nil, ctx.Err()
            }
        }
    }

    return nil, fmt.Errorf("max retries exceeded")
}

func isRetryableError(err error) bool {
    var apiErr smithy.APIError
    if errors.As(err, &apiErr) {
        switch apiErr.ErrorCode() {
        case "ThrottlingException", "ModelTimeoutException", "InternalServerException":
            return true
        }
    }
    return false
}
```

## Streaming Implementation

### Event Stream Processing

```go
func (bp *BedrockProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
    defer close(ch)

    // Build request
    requestBody, modelFamily, err := bp.buildBedrockRequest(request)
    if err != nil {
        return err
    }

    // Invoke streaming API
    output, err := bp.bedrockClient.InvokeModelWithResponseStream(ctx, &bedrockruntime.InvokeModelWithResponseStreamInput{
        ModelId:     aws.String(request.Model),
        Body:        requestBody,
        Accept:      aws.String("application/json"),
        ContentType: aws.String("application/json"),
    })
    if err != nil {
        return handleBedrockError(err)
    }

    // Process event stream
    stream := output.GetStream()
    defer stream.Close()

    var contentBuilder strings.Builder

    for event := range stream.Events() {
        switch e := event.(type) {
        case *types.ResponseStreamMemberChunk:
            // Parse chunk based on model family
            chunk, err := parseStreamChunk(e.Value.Bytes, modelFamily)
            if err != nil {
                log.Printf("Error parsing chunk: %v", err)
                continue
            }

            if chunk.Delta != "" {
                contentBuilder.WriteString(chunk.Delta)

                // Send incremental response
                ch <- LLMResponse{
                    ID:        uuid.New(),
                    RequestID: request.ID,
                    Content:   chunk.Delta,
                    CreatedAt: time.Now(),
                }
            }

        case *types.ResponseStreamMemberInternalServerException:
            return fmt.Errorf("internal server error: %s", *e.Value.Message)

        case *types.ResponseStreamMemberModelStreamErrorException:
            return fmt.Errorf("model stream error: %s", *e.Value.Message)

        case *types.ResponseStreamMemberThrottlingException:
            return ErrRateLimited

        case *types.ResponseStreamMemberValidationException:
            return fmt.Errorf("validation error: %s", *e.Value.Message)
        }
    }

    // Check for stream errors
    if err := stream.Err(); err != nil {
        return handleBedrockError(err)
    }

    // Send final complete response
    ch <- LLMResponse{
        ID:           uuid.New(),
        RequestID:    request.ID,
        Content:      contentBuilder.String(),
        FinishReason: "stop",
        CreatedAt:    time.Now(),
    }

    return nil
}
```

## Health Check Implementation

```go
func (bp *BedrockProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
    startTime := time.Now()

    health := &ProviderHealth{
        LastCheck:  time.Now(),
        ModelCount: len(bp.models),
    }

    // Test with a minimal request to Claude (fastest model)
    testReq := &LLMRequest{
        ID:          uuid.New(),
        Model:       "anthropic.claude-3-5-haiku-20241022-v1:0",
        Messages:    []Message{{Role: "user", Content: "Hi"}},
        MaxTokens:   10,
        Temperature: 0.1,
    }

    _, err := bp.Generate(ctx, testReq)
    if err != nil {
        health.Status = "unhealthy"
        health.ErrorCount = 1
        return health, err
    }

    health.Status = "healthy"
    health.Latency = time.Since(startTime)
    bp.lastHealth = health

    return health, nil
}
```

## Testing Strategy

### 1. Mock AWS API Testing

```go
// Use aws-sdk-go-v2 mock
type mockBedrockClient struct {
    bedrockruntimeiface.BedrockRuntimeAPI
    InvokeModelFunc                    func(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error)
    InvokeModelWithResponseStreamFunc  func(ctx context.Context, params *bedrockruntime.InvokeModelWithResponseStreamInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelWithResponseStreamOutput, error)
}

func TestBedrockProvider_Generate(t *testing.T) {
    mockClient := &mockBedrockClient{
        InvokeModelFunc: func(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
            // Return mock response
            responseBody := []byte(`{
                "completion": "Hello! How can I help you?",
                "stop_reason": "end_turn",
                "usage": {
                    "input_tokens": 10,
                    "output_tokens": 20
                }
            }`)

            return &bedrockruntime.InvokeModelOutput{
                Body:        responseBody,
                ContentType: aws.String("application/json"),
            }, nil
        },
    }

    provider := &BedrockProvider{
        bedrockClient: mockClient,
        models:        getBedrockModels(),
    }

    request := &LLMRequest{
        ID:       uuid.New(),
        Model:    "anthropic.claude-3-5-sonnet-20241022-v2:0",
        Messages: []Message{{Role: "user", Content: "Hello"}},
        MaxTokens: 100,
    }

    response, err := provider.Generate(context.Background(), request)
    assert.NoError(t, err)
    assert.NotNil(t, response)
    assert.NotEmpty(t, response.Content)
}
```

### 2. Integration Tests

```go
func TestBedrockProvider_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Requires AWS credentials
    config := ProviderConfigEntry{
        Type:    ProviderTypeBedrock,
        Enabled: true,
        Parameters: map[string]interface{}{
            "region": "us-east-1",
        },
    }

    provider, err := NewBedrockProvider(config)
    require.NoError(t, err)

    // Test availability
    assert.True(t, provider.IsAvailable(context.Background()))

    // Test generation
    request := &LLMRequest{
        ID:       uuid.New(),
        Model:    "anthropic.claude-3-5-haiku-20241022-v1:0",
        Messages: []Message{{Role: "user", Content: "Say hello"}},
        MaxTokens: 50,
    }

    response, err := provider.Generate(context.Background(), request)
    assert.NoError(t, err)
    assert.NotNil(t, response)
    assert.NotEmpty(t, response.Content)
}
```

### 3. Error Handling Tests

```go
func TestBedrockProvider_ErrorHandling(t *testing.T) {
    tests := []struct {
        name          string
        mockError     error
        expectedError error
    }{
        {
            name:          "ThrottlingException",
            mockError:     &smithy.GenericAPIError{Code: "ThrottlingException", Message: "Rate exceeded"},
            expectedError: ErrRateLimited,
        },
        {
            name:          "ValidationException",
            mockError:     &smithy.GenericAPIError{Code: "ValidationException", Message: "Invalid input"},
            expectedError: ErrInvalidRequest,
        },
        {
            name:          "ResourceNotFoundException",
            mockError:     &smithy.GenericAPIError{Code: "ResourceNotFoundException", Message: "Model not found"},
            expectedError: ErrModelNotFound,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockClient := &mockBedrockClient{
                InvokeModelFunc: func(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
                    return nil, tt.mockError
                },
            }

            provider := &BedrockProvider{
                bedrockClient: mockClient,
            }

            request := &LLMRequest{
                ID:       uuid.New(),
                Model:    "test-model",
                Messages: []Message{{Role: "user", Content: "test"}},
            }

            _, err := provider.Generate(context.Background(), request)
            assert.Error(t, err)
            assert.ErrorIs(t, err, tt.expectedError)
        })
    }
}
```

## Configuration Schema

```yaml
# config/config.yaml
llm:
  providers:
    bedrock:
      type: "bedrock"
      enabled: true
      region: "us-east-1"
      cross_region_inference: true
      inference_profile_arn: "arn:aws:bedrock:us-east-1:123456789012:inference-profile/us.anthropic.claude-3-5-sonnet-20241022-v2:0"
      models:
        - "anthropic.claude-4-sonnet-20250514-v1:0"
        - "anthropic.claude-3-7-sonnet-20250219-v1:0"
        - "anthropic.claude-3-5-sonnet-20241022-v2:0"
        - "anthropic.claude-3-5-haiku-20241022-v1:0"
        - "amazon.titan-text-premier-v1:0"
        - "ai21.jamba-1-5-large-v1:0"
        - "cohere.command-r-plus-v1:0"
        - "meta.llama3-3-70b-instruct-v1:0"
```

**Environment Variables**:
```bash
# IAM Role (preferred - no env vars needed)

# OR Access Keys
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"

# Optional
export AWS_SESSION_TOKEN="your-session-token"
export AWS_INFERENCE_PROFILE_ARN="arn:aws:bedrock:..."
```

## Example Usage

### Basic Generation

```go
// Initialize provider
config := ProviderConfigEntry{
    Type:    ProviderTypeBedrock,
    Enabled: true,
    Parameters: map[string]interface{}{
        "region": "us-east-1",
    },
}

provider, err := NewBedrockProvider(config)
if err != nil {
    log.Fatal(err)
}
defer provider.Close()

// Generate response
request := &LLMRequest{
    ID:    uuid.New(),
    Model: "anthropic.claude-3-5-sonnet-20241022-v2:0",
    Messages: []Message{
        {Role: "user", Content: "Explain AWS Bedrock in one sentence."},
    },
    MaxTokens:   200,
    Temperature: 0.7,
}

response, err := provider.Generate(context.Background(), request)
if err != nil {
    log.Fatal(err)
}

fmt.Println(response.Content)
fmt.Printf("Tokens used: %d\n", response.Usage.TotalTokens)
```

### Streaming Generation

```go
request := &LLMRequest{
    ID:    uuid.New(),
    Model: "anthropic.claude-3-5-sonnet-20241022-v2:0",
    Messages: []Message{
        {Role: "user", Content: "Write a short story about AI."},
    },
    MaxTokens:   1000,
    Temperature: 0.8,
    Stream:      true,
}

responseCh := make(chan LLMResponse)

go func() {
    if err := provider.GenerateStream(context.Background(), request, responseCh); err != nil {
        log.Printf("Stream error: %v", err)
    }
}()

// Process streaming responses
for response := range responseCh {
    fmt.Print(response.Content)
}
fmt.Println()
```

### Cross-Region Inference

```go
config := ProviderConfigEntry{
    Type:    ProviderTypeBedrock,
    Enabled: true,
    Parameters: map[string]interface{}{
        "region":                  "us-east-1",
        "cross_region_inference":  true,
        "inference_profile_arn":   "arn:aws:bedrock:us-east-1:123456789012:inference-profile/us.anthropic.claude-3-5-sonnet-20241022-v2:0",
    },
}

provider, err := NewBedrockProvider(config)
// Use inference profile for automatic cross-region routing
```

## Migration Notes

### From Direct Anthropic API to Bedrock

**API Differences**:
- Model IDs use `anthropic.` prefix (e.g., `anthropic.claude-3-5-sonnet-20241022-v2:0`)
- Request/response format is similar but wrapped in Bedrock envelope
- Authentication via AWS IAM instead of API keys
- Different error codes and rate limits

**Migration Steps**:
1. Set up AWS credentials (IAM role or access keys)
2. Enable Bedrock model access in AWS console
3. Update model names to Bedrock format
4. Update configuration to use `ProviderTypeBedrock`
5. Test with health check and small requests

### From OpenAI to Bedrock (Claude)

**Key Differences**:
- Replace `gpt-4` with `anthropic.claude-4-sonnet-20250514-v1:0`
- System messages handled differently (separate field)
- Tool calling format differs
- Streaming event format differs

## Dependencies

```go
import (
    "context"
    "errors"
    "fmt"
    "log"
    "os"
    "strings"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/credentials/stscreds"
    "github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
    "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
    "github.com/aws/aws-sdk-go-v2/service/sts"
    "github.com/aws/smithy-go"
    "github.com/google/uuid"
)
```

**go.mod additions**:
```
require (
    github.com/aws/aws-sdk-go-v2 v1.32.7
    github.com/aws/aws-sdk-go-v2/config v1.28.7
    github.com/aws/aws-sdk-go-v2/credentials v1.17.48
    github.com/aws/aws-sdk-go-v2/service/bedrockruntime v1.23.1
    github.com/aws/aws-sdk-go-v2/service/sts v1.33.7
    github.com/aws/smithy-go v1.22.1
)
```

## References

- **AWS Bedrock Documentation**: https://docs.aws.amazon.com/bedrock/
- **AWS SDK for Go v2**: https://aws.github.io/aws-sdk-go-v2/docs/
- **Bedrock Runtime API**: https://docs.aws.amazon.com/bedrock/latest/APIReference/API_Operations_Amazon_Bedrock_Runtime.html
- **Plandex Bedrock Implementation**: `/Users/milosvasic/Projects/HelixCode/Example_Projects/Plandex/app/shared/ai_models_providers.go`
- **Anthropic Provider Reference**: `/Users/milosvasic/Projects/HelixCode/HelixCode/internal/llm/anthropic_provider.go`
- **Cross-Region Inference**: https://docs.aws.amazon.com/bedrock/latest/userguide/cross-region-inference.html

## Performance Characteristics

- **Latency**: 500ms - 3s (first token)
- **Throughput**: Varies by model and region
- **Rate Limits**: Model and account-specific (configurable in AWS)
- **Cross-Region Inference**: Adds 50-200ms overhead but improves availability
- **Streaming**: ~50-100ms per chunk

## Security Considerations

1. **IAM Best Practices**: Use least-privilege IAM roles
2. **Credentials**: Never hardcode credentials, use IAM roles when possible
3. **VPC Endpoints**: Use VPC endpoints for private connectivity
4. **Logging**: Enable CloudTrail for audit logging
5. **Encryption**: Data encrypted in transit and at rest by default
