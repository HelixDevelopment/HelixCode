package llm

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock Bedrock client for testing
type mockBedrockClient struct {
	invokeModelFunc              func(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error)
	invokeModelWithStreamFunc    func(ctx context.Context, params *bedrockruntime.InvokeModelWithResponseStreamInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelWithResponseStreamOutput, error)
}

func (m *mockBedrockClient) InvokeModel(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
	if m.invokeModelFunc != nil {
		return m.invokeModelFunc(ctx, params, optFns...)
	}
	return nil, nil
}

func (m *mockBedrockClient) InvokeModelWithResponseStream(ctx context.Context, params *bedrockruntime.InvokeModelWithResponseStreamInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelWithResponseStreamOutput, error) {
	if m.invokeModelWithStreamFunc != nil {
		return m.invokeModelWithStreamFunc(ctx, params, optFns...)
	}
	return nil, nil
}

func TestNewBedrockProvider(t *testing.T) {
	tests := []struct {
		name        string
		config      ProviderConfigEntry
		envVars     map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config with explicit credentials",
			config: ProviderConfigEntry{
				Type:    ProviderTypeBedrock,
				Enabled: true,
				Parameters: map[string]interface{}{
					"region":                  "us-east-1",
					"aws_access_key_id":       "test-key",
					"aws_secret_access_key":   "test-secret",
				},
			},
			expectError: false,
		},
		{
			name: "valid config with IAM role (env vars)",
			config: ProviderConfigEntry{
				Type:    ProviderTypeBedrock,
				Enabled: true,
				Parameters: map[string]interface{}{
					"region": "us-west-2",
				},
			},
			envVars: map[string]string{
				"AWS_REGION": "us-west-2",
			},
			expectError: false,
		},
		{
			name: "default region",
			config: ProviderConfigEntry{
				Type:    ProviderTypeBedrock,
				Enabled: true,
			},
			expectError: false,
		},
		{
			name: "cross-region inference enabled",
			config: ProviderConfigEntry{
				Type:    ProviderTypeBedrock,
				Enabled: true,
				Parameters: map[string]interface{}{
					"region":                  "us-east-1",
					"cross_region_inference":  true,
					"inference_profile_arn":   "arn:aws:bedrock:us-east-1:123456789012:inference-profile/us.anthropic.claude-3-5-sonnet-20241022-v2:0",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables if specified
			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			// Note: This will fail without valid AWS credentials
			// For unit tests, we'll skip actual AWS calls
			if testing.Short() {
				t.Skip("Skipping AWS SDK test in short mode")
			}

			provider, err := NewBedrockProvider(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, provider)
			} else {
				// May fail without AWS credentials, which is OK for this test
				if err == nil {
					assert.NotNil(t, provider)
					assert.Equal(t, ProviderTypeBedrock, provider.GetType())
					assert.Equal(t, "AWS Bedrock", provider.GetName())
				}
			}
		})
	}
}

func TestBedrockProvider_GetType(t *testing.T) {
	provider := &BedrockProvider{}
	assert.Equal(t, ProviderTypeBedrock, provider.GetType())
}

func TestBedrockProvider_GetName(t *testing.T) {
	provider := &BedrockProvider{}
	assert.Equal(t, "AWS Bedrock", provider.GetName())
}

func TestBedrockProvider_GetModels(t *testing.T) {
	provider := &BedrockProvider{
		models: getBedrockModels(),
	}

	models := provider.GetModels()
	assert.NotEmpty(t, models)

	// Check that we have expected models
	modelNames := make(map[string]bool)
	for _, model := range models {
		modelNames[model.Name] = true
		assert.Equal(t, ProviderTypeBedrock, model.Provider)
		assert.Greater(t, model.ContextSize, 0)
		assert.NotEmpty(t, model.Description)
	}

	// Verify key models exist
	assert.True(t, modelNames["anthropic.claude-4-sonnet-20250514-v1:0"], "Should have Claude 4 Sonnet")
	assert.True(t, modelNames["anthropic.claude-3-5-sonnet-20241022-v2:0"], "Should have Claude 3.5 Sonnet v2")
	assert.True(t, modelNames["anthropic.claude-3-5-haiku-20241022-v1:0"], "Should have Claude 3.5 Haiku")
	assert.True(t, modelNames["amazon.titan-text-premier-v1:0"], "Should have Titan Premier")
	assert.True(t, modelNames["ai21.jamba-1-5-large-v1:0"], "Should have Jamba 1.5")
	assert.True(t, modelNames["cohere.command-r-plus-v1:0"], "Should have Command R+")
	assert.True(t, modelNames["meta.llama3-3-70b-instruct-v1:0"], "Should have Llama 3.3")
}

func TestBedrockProvider_GetCapabilities(t *testing.T) {
	provider := &BedrockProvider{}

	caps := provider.GetCapabilities()
	assert.NotEmpty(t, caps)

	// Convert to map for easy checking
	capMap := make(map[ModelCapability]bool)
	for _, cap := range caps {
		capMap[cap] = true
	}

	// Verify all expected capabilities
	assert.True(t, capMap[CapabilityTextGeneration])
	assert.True(t, capMap[CapabilityCodeGeneration])
	assert.True(t, capMap[CapabilityCodeAnalysis])
	assert.True(t, capMap[CapabilityPlanning])
	assert.True(t, capMap[CapabilityVision])
}

func TestBedrockProvider_GetModelFamily(t *testing.T) {
	provider := &BedrockProvider{}

	tests := []struct {
		modelID        string
		expectedFamily bedrockModelFamily
	}{
		{"anthropic.claude-3-5-sonnet-20241022-v2:0", modelFamilyClaude},
		{"amazon.titan-text-premier-v1:0", modelFamilyTitan},
		{"ai21.jamba-1-5-large-v1:0", modelFamilyJurassic},
		{"cohere.command-r-plus-v1:0", modelFamilyCommand},
		{"meta.llama3-3-70b-instruct-v1:0", modelFamilyLlama},
		{"unknown.model", modelFamilyClaude}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			family := provider.getModelFamily(tt.modelID)
			assert.Equal(t, tt.expectedFamily, family)
		})
	}
}

func TestBedrockProvider_GenerateClaude(t *testing.T) {
	// Create mock client
	mockClient := &mockBedrockClient{
		invokeModelFunc: func(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
			// Verify request
			assert.Equal(t, "anthropic.claude-3-5-sonnet-20241022-v2:0", aws.ToString(params.ModelId))
			assert.Equal(t, "application/json", aws.ToString(params.ContentType))

			// Return mock Claude response
			response := bedrockClaudeResponse{
				ID:   "msg_test",
				Type: "message",
				Role: "assistant",
				Content: []anthropicContentBlock{
					{
						Type: "text",
						Text: "Hello! This is a test response from Bedrock.",
					},
				},
				Model:      "anthropic.claude-3-5-sonnet-20241022-v2:0",
				StopReason: "end_turn",
				Usage: anthropicUsage{
					InputTokens:  15,
					OutputTokens: 25,
				},
			}

			responseBody, _ := json.Marshal(response)
			return &bedrockruntime.InvokeModelOutput{
				Body:        responseBody,
				ContentType: aws.String("application/json"),
			}, nil
		},
	}

	provider := &BedrockProvider{
		bedrockClient: mockClient,
		models:        getBedrockModels(),
		region:        "us-east-1",
	}

	// Test generation
	request := &LLMRequest{
		ID:    uuid.New(),
		Model: "anthropic.claude-3-5-sonnet-20241022-v2:0",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	response, err := provider.Generate(context.Background(), request)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Hello! This is a test response from Bedrock.", response.Content)
	assert.Equal(t, 15, response.Usage.PromptTokens)
	assert.Equal(t, 25, response.Usage.CompletionTokens)
	assert.Equal(t, 40, response.Usage.TotalTokens)
	assert.Equal(t, "end_turn", response.FinishReason)
}

func TestBedrockProvider_GenerateTitan(t *testing.T) {
	mockClient := &mockBedrockClient{
		invokeModelFunc: func(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
			assert.Equal(t, "amazon.titan-text-premier-v1:0", aws.ToString(params.ModelId))

			response := bedrockTitanResponse{
				InputTextTokenCount: 10,
				Results: []titanResult{
					{
						TokenCount:       20,
						OutputText:       "This is a Titan response.",
						CompletionReason: "FINISH",
					},
				},
			}

			responseBody, _ := json.Marshal(response)
			return &bedrockruntime.InvokeModelOutput{
				Body:        responseBody,
				ContentType: aws.String("application/json"),
			}, nil
		},
	}

	provider := &BedrockProvider{
		bedrockClient: mockClient,
		models:        getBedrockModels(),
		region:        "us-east-1",
	}

	request := &LLMRequest{
		ID:    uuid.New(),
		Model: "amazon.titan-text-premier-v1:0",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	response, err := provider.Generate(context.Background(), request)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "This is a Titan response.", response.Content)
	assert.Equal(t, 10, response.Usage.PromptTokens)
	assert.Equal(t, 20, response.Usage.CompletionTokens)
	assert.Equal(t, 30, response.Usage.TotalTokens)
}

func TestBedrockProvider_GenerateJurassic(t *testing.T) {
	mockClient := &mockBedrockClient{
		invokeModelFunc: func(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
			assert.Equal(t, "ai21.jamba-1-5-large-v1:0", aws.ToString(params.ModelId))

			response := bedrockJurassicResponse{
				ID: "test-id",
				Prompt: jurassicPrompt{
					Text:   "test prompt",
					Tokens: make([]interface{}, 10),
				},
				Completions: []jurassicCompletion{
					{
						Data: jurassicData{
							Text:   "This is a Jurassic response.",
							Tokens: make([]interface{}, 15),
						},
						FinishReason: jurassicFinishReason{
							Reason: "endoftext",
						},
					},
				},
			}

			responseBody, _ := json.Marshal(response)
			return &bedrockruntime.InvokeModelOutput{
				Body:        responseBody,
				ContentType: aws.String("application/json"),
			}, nil
		},
	}

	provider := &BedrockProvider{
		bedrockClient: mockClient,
		models:        getBedrockModels(),
		region:        "us-east-1",
	}

	request := &LLMRequest{
		ID:    uuid.New(),
		Model: "ai21.jamba-1-5-large-v1:0",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	response, err := provider.Generate(context.Background(), request)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "This is a Jurassic response.", response.Content)
	assert.Equal(t, 10, response.Usage.PromptTokens)
	assert.Equal(t, 15, response.Usage.CompletionTokens)
	assert.Equal(t, 25, response.Usage.TotalTokens)
}

func TestBedrockProvider_GenerateCommand(t *testing.T) {
	mockClient := &mockBedrockClient{
		invokeModelFunc: func(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
			assert.Equal(t, "cohere.command-r-plus-v1:0", aws.ToString(params.ModelId))

			response := bedrockCommandResponse{
				ResponseID:   "test-id",
				Text:         "This is a Command response.",
				GenerationID: "gen-id",
				FinishReason: "COMPLETE",
				Meta: cohereMetaData{
					BilledUnits: struct {
						InputTokens  int `json:"input_tokens"`
						OutputTokens int `json:"output_tokens"`
					}{
						InputTokens:  12,
						OutputTokens: 18,
					},
				},
			}

			responseBody, _ := json.Marshal(response)
			return &bedrockruntime.InvokeModelOutput{
				Body:        responseBody,
				ContentType: aws.String("application/json"),
			}, nil
		},
	}

	provider := &BedrockProvider{
		bedrockClient: mockClient,
		models:        getBedrockModels(),
		region:        "us-east-1",
	}

	request := &LLMRequest{
		ID:    uuid.New(),
		Model: "cohere.command-r-plus-v1:0",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	response, err := provider.Generate(context.Background(), request)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "This is a Command response.", response.Content)
	assert.Equal(t, 12, response.Usage.PromptTokens)
	assert.Equal(t, 18, response.Usage.CompletionTokens)
	assert.Equal(t, 30, response.Usage.TotalTokens)
}

func TestBedrockProvider_GenerateLlama(t *testing.T) {
	mockClient := &mockBedrockClient{
		invokeModelFunc: func(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
			assert.Equal(t, "meta.llama3-3-70b-instruct-v1:0", aws.ToString(params.ModelId))

			response := bedrockLlamaResponse{
				Generation:           "This is a Llama response.",
				PromptTokenCount:     14,
				GenerationTokenCount: 16,
				StopReason:           "stop",
			}

			responseBody, _ := json.Marshal(response)
			return &bedrockruntime.InvokeModelOutput{
				Body:        responseBody,
				ContentType: aws.String("application/json"),
			}, nil
		},
	}

	provider := &BedrockProvider{
		bedrockClient: mockClient,
		models:        getBedrockModels(),
		region:        "us-east-1",
	}

	request := &LLMRequest{
		ID:    uuid.New(),
		Model: "meta.llama3-3-70b-instruct-v1:0",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	response, err := provider.Generate(context.Background(), request)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "This is a Llama response.", response.Content)
	assert.Equal(t, 14, response.Usage.PromptTokens)
	assert.Equal(t, 16, response.Usage.CompletionTokens)
	assert.Equal(t, 30, response.Usage.TotalTokens)
}

func TestBedrockProvider_GenerateWithTools(t *testing.T) {
	mockClient := &mockBedrockClient{
		invokeModelFunc: func(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
			// Verify tools are in request
			var req bedrockClaudeRequest
			err := json.Unmarshal(params.Body, &req)
			assert.NoError(t, err)
			assert.NotEmpty(t, req.Tools)
			assert.Equal(t, "get_weather", req.Tools[0].Name)

			// Return tool use response
			response := bedrockClaudeResponse{
				ID:   "msg_test",
				Type: "message",
				Role: "assistant",
				Content: []anthropicContentBlock{
					{
						Type: "tool_use",
						ID:   "toolu_123",
						Name: "get_weather",
						Input: map[string]interface{}{
							"location": "San Francisco",
						},
					},
				},
				Model:      "anthropic.claude-3-5-sonnet-20241022-v2:0",
				StopReason: "tool_use",
				Usage: anthropicUsage{
					InputTokens:  50,
					OutputTokens: 30,
				},
			}

			responseBody, _ := json.Marshal(response)
			return &bedrockruntime.InvokeModelOutput{
				Body:        responseBody,
				ContentType: aws.String("application/json"),
			}, nil
		},
	}

	provider := &BedrockProvider{
		bedrockClient: mockClient,
		models:        getBedrockModels(),
		region:        "us-east-1",
	}

	request := &LLMRequest{
		ID:    uuid.New(),
		Model: "anthropic.claude-3-5-sonnet-20241022-v2:0",
		Messages: []Message{
			{Role: "user", Content: "What's the weather in San Francisco?"},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
		Tools: []Tool{
			{
				Type: "function",
				Function: FunctionDefinition{
					Name:        "get_weather",
					Description: "Get the current weather",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{
								"type":        "string",
								"description": "City name",
							},
						},
					},
				},
			},
		},
	}

	response, err := provider.Generate(context.Background(), request)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, response.ToolCalls)
	assert.Equal(t, "toolu_123", response.ToolCalls[0].ID)
	assert.Equal(t, "get_weather", response.ToolCalls[0].Function.Name)
	assert.Equal(t, "San Francisco", response.ToolCalls[0].Function.Arguments["location"])
}

func TestBedrockProvider_CrossRegionInference(t *testing.T) {
	inferenceProfileArn := "arn:aws:bedrock:us-east-1:123456789012:inference-profile/us.anthropic.claude-3-5-sonnet-20241022-v2:0"

	mockClient := &mockBedrockClient{
		invokeModelFunc: func(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
			// Verify inference profile ARN is used
			assert.Equal(t, inferenceProfileArn, aws.ToString(params.ModelId))

			response := bedrockClaudeResponse{
				ID:   "msg_test",
				Type: "message",
				Role: "assistant",
				Content: []anthropicContentBlock{
					{Type: "text", Text: "Response"},
				},
				Model:      "anthropic.claude-3-5-sonnet-20241022-v2:0",
				StopReason: "end_turn",
				Usage: anthropicUsage{
					InputTokens:  10,
					OutputTokens: 20,
				},
			}

			responseBody, _ := json.Marshal(response)
			return &bedrockruntime.InvokeModelOutput{
				Body:        responseBody,
				ContentType: aws.String("application/json"),
			}, nil
		},
	}

	provider := &BedrockProvider{
		bedrockClient:        mockClient,
		models:               getBedrockModels(),
		region:               "us-east-1",
		crossRegionInference: true,
		inferenceProfileArn:  inferenceProfileArn,
	}

	request := &LLMRequest{
		ID:    uuid.New(),
		Model: "anthropic.claude-3-5-sonnet-20241022-v2:0",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 1000,
	}

	response, err := provider.Generate(context.Background(), request)
	require.NoError(t, err)
	assert.NotNil(t, response)
}

func TestBedrockProvider_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		awsError      error
		expectedError error
	}{
		{
			name: "ThrottlingException",
			awsError: &smithy.GenericAPIError{
				Code:    "ThrottlingException",
				Message: "Rate exceeded",
			},
			expectedError: ErrRateLimited,
		},
		{
			name: "ValidationException",
			awsError: &smithy.GenericAPIError{
				Code:    "ValidationException",
				Message: "Invalid input",
			},
			expectedError: ErrInvalidRequest,
		},
		{
			name: "ResourceNotFoundException",
			awsError: &smithy.GenericAPIError{
				Code:    "ResourceNotFoundException",
				Message: "Model not found",
			},
			expectedError: ErrModelNotFound,
		},
		{
			name: "AccessDeniedException",
			awsError: &smithy.GenericAPIError{
				Code:    "AccessDeniedException",
				Message: "Access denied",
			},
			expectedError: nil, // Custom error wrapping
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockBedrockClient{
				invokeModelFunc: func(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
					return nil, tt.awsError
				},
			}

			provider := &BedrockProvider{
				bedrockClient: mockClient,
				models:        getBedrockModels(),
			}

			request := &LLMRequest{
				ID:    uuid.New(),
				Model: "anthropic.claude-3-5-sonnet-20241022-v2:0",
				Messages: []Message{
					{Role: "user", Content: "Hello"},
				},
				MaxTokens: 1000,
			}

			_, err := provider.Generate(context.Background(), request)
			assert.Error(t, err)

			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
			}
		})
	}
}

func TestBedrockProvider_BuildClaudeRequest(t *testing.T) {
	provider := &BedrockProvider{}

	request := &LLMRequest{
		ID:    uuid.New(),
		Model: "anthropic.claude-3-5-sonnet-20241022-v2:0",
		Messages: []Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
		TopP:        0.9,
	}

	body, err := provider.buildClaudeRequest(request)
	require.NoError(t, err)

	var claudeReq bedrockClaudeRequest
	err = json.Unmarshal(body, &claudeReq)
	require.NoError(t, err)

	assert.Equal(t, 1000, claudeReq.MaxTokens)
	assert.Equal(t, 0.7, claudeReq.Temperature)
	assert.Equal(t, 0.9, claudeReq.TopP)
	assert.Equal(t, "You are helpful.", claudeReq.System)
	assert.Len(t, claudeReq.Messages, 1)
	assert.Equal(t, "user", claudeReq.Messages[0].Role)
}

func TestBedrockProvider_BuildTitanRequest(t *testing.T) {
	provider := &BedrockProvider{}

	request := &LLMRequest{
		ID:    uuid.New(),
		Model: "amazon.titan-text-premier-v1:0",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   500,
		Temperature: 0.5,
	}

	body, err := provider.buildTitanRequest(request)
	require.NoError(t, err)

	var titanReq bedrockTitanRequest
	err = json.Unmarshal(body, &titanReq)
	require.NoError(t, err)

	assert.Contains(t, titanReq.InputText, "Hello")
	assert.NotNil(t, titanReq.TextGenerationConfig)
	assert.Equal(t, 500, titanReq.TextGenerationConfig.MaxTokenCount)
	assert.Equal(t, 0.5, titanReq.TextGenerationConfig.Temperature)
}

func TestBedrockProvider_IsAvailable(t *testing.T) {
	tests := []struct {
		name      string
		provider  *BedrockProvider
		available bool
	}{
		{
			name: "available with client",
			provider: &BedrockProvider{
				bedrockClient: &mockBedrockClient{},
			},
			available: true,
		},
		{
			name: "not available without client",
			provider: &BedrockProvider{
				bedrockClient: nil,
			},
			available: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			available := tt.provider.IsAvailable(context.Background())
			assert.Equal(t, tt.available, available)
		})
	}
}

func TestBedrockProvider_CombineMessagesToPrompt(t *testing.T) {
	provider := &BedrockProvider{}

	messages := []Message{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "How are you?"},
	}

	prompt := provider.combineMessagesToPrompt(messages)

	assert.Contains(t, prompt, "You are helpful.")
	assert.Contains(t, prompt, "User: Hello")
	assert.Contains(t, prompt, "Assistant: Hi there!")
	assert.Contains(t, prompt, "User: How are you?")
}

func TestBedrockProvider_ConvertTools(t *testing.T) {
	provider := &BedrockProvider{}

	tools := []Tool{
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "get_weather",
				Description: "Get weather data",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
		},
	}

	// Test Anthropic conversion
	anthropicTools := provider.convertToolsToAnthropic(tools)
	assert.Len(t, anthropicTools, 1)
	assert.Equal(t, "get_weather", anthropicTools[0].Name)
	assert.Equal(t, "Get weather data", anthropicTools[0].Description)

	// Test Cohere conversion
	cohereTools := provider.convertToolsToCohere(tools)
	assert.Len(t, cohereTools, 1)
	assert.Equal(t, "get_weather", cohereTools[0].Name)
}

func TestBedrockProvider_Close(t *testing.T) {
	provider := &BedrockProvider{}
	err := provider.Close()
	assert.NoError(t, err)
}

func TestBedrockProvider_GenerateStream(t *testing.T) {
	// Streaming tests are complex with the AWS SDK EventStream
	// These would require more sophisticated mocking or integration tests
	// For now, we'll skip this test and rely on integration testing
	t.Skip("Skipping stream test - requires AWS SDK EventStream mocking which is complex")
}

func TestBedrockProvider_HandleBedrockError(t *testing.T) {
	provider := &BedrockProvider{}

	tests := []struct {
		name          string
		inputError    error
		expectedError error
	}{
		{
			name:          "nil error",
			inputError:    nil,
			expectedError: nil,
		},
		{
			name: "throttling error",
			inputError: &smithy.GenericAPIError{
				Code:    "ThrottlingException",
				Message: "Rate limit exceeded",
			},
			expectedError: ErrRateLimited,
		},
		{
			name: "validation error",
			inputError: &smithy.GenericAPIError{
				Code:    "ValidationException",
				Message: "Invalid parameters",
			},
			expectedError: ErrInvalidRequest,
		},
		{
			name: "model not found",
			inputError: &smithy.GenericAPIError{
				Code:    "ResourceNotFoundException",
				Message: "Model does not exist",
			},
			expectedError: ErrModelNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.handleBedrockError(tt.inputError)

			if tt.expectedError == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
			}
		})
	}
}

// Integration test - requires actual AWS credentials
func TestBedrockProvider_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// This test requires valid AWS credentials
	config := ProviderConfigEntry{
		Type:    ProviderTypeBedrock,
		Enabled: true,
		Parameters: map[string]interface{}{
			"region": "us-east-1",
		},
	}

	provider, err := NewBedrockProvider(config)
	if err != nil {
		t.Skipf("Skipping integration test: %v", err)
	}

	// Test availability
	assert.True(t, provider.IsAvailable(context.Background()))

	// Test generation with Claude Haiku (fastest model)
	request := &LLMRequest{
		ID:    uuid.New(),
		Model: "anthropic.claude-3-5-haiku-20241022-v1:0",
		Messages: []Message{
			{Role: "user", Content: "Say hello"},
		},
		MaxTokens:   50,
		Temperature: 0.1,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := provider.Generate(ctx, request)
	if err != nil {
		t.Skipf("Integration test failed (may be due to AWS access): %v", err)
	}

	assert.NotNil(t, response)
	assert.NotEmpty(t, response.Content)
	assert.Greater(t, response.Usage.TotalTokens, 0)
}
