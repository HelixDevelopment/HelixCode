package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"
)

// bedrockClientInterface defines the interface for Bedrock client operations
type bedrockClientInterface interface {
	InvokeModel(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error)
	InvokeModelWithResponseStream(ctx context.Context, params *bedrockruntime.InvokeModelWithResponseStreamInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelWithResponseStreamOutput, error)
}

// BedrockProvider implements the Provider interface for AWS Bedrock
type BedrockProvider struct {
	config                ProviderConfigEntry
	awsConfig             aws.Config
	bedrockClient         bedrockClientInterface
	models                []ModelInfo
	region                string
	crossRegionInference  bool
	inferenceProfileArn   string
	lastHealth            *ProviderHealth
}

// Model family types for request/response transformation
type bedrockModelFamily string

const (
	modelFamilyClaude   bedrockModelFamily = "claude"
	modelFamilyTitan    bedrockModelFamily = "titan"
	modelFamilyJurassic bedrockModelFamily = "jurassic"
	modelFamilyCommand  bedrockModelFamily = "command"
	modelFamilyLlama    bedrockModelFamily = "llama"
)

// Claude request/response structures (Anthropic format via Bedrock)
type bedrockClaudeRequest struct {
	Messages      []anthropicMessage       `json:"messages"`
	System        interface{}              `json:"system,omitempty"`
	MaxTokens     int                      `json:"max_tokens"`
	Temperature   float64                  `json:"temperature,omitempty"`
	TopP          float64                  `json:"top_p,omitempty"`
	Tools         []anthropicTool          `json:"tools,omitempty"`
	ToolChoice    interface{}              `json:"tool_choice,omitempty"`
	StopSequences []string                 `json:"stop_sequences,omitempty"`
}

type bedrockClaudeResponse struct {
	ID           string                  `json:"id"`
	Type         string                  `json:"type"`
	Role         string                  `json:"role"`
	Content      []anthropicContentBlock `json:"content"`
	Model        string                  `json:"model"`
	StopReason   string                  `json:"stop_reason"`
	StopSequence string                  `json:"stop_sequence,omitempty"`
	Usage        anthropicUsage          `json:"usage"`
}

// Titan request/response structures
type bedrockTitanRequest struct {
	InputText       string                    `json:"inputText"`
	TextGenerationConfig *titanTextGenConfig `json:"textGenerationConfig,omitempty"`
}

type titanTextGenConfig struct {
	Temperature   float64  `json:"temperature,omitempty"`
	TopP          float64  `json:"topP,omitempty"`
	MaxTokenCount int      `json:"maxTokenCount,omitempty"`
	StopSequences []string `json:"stopSequences,omitempty"`
}

type bedrockTitanResponse struct {
	InputTextTokenCount int                   `json:"inputTextTokenCount"`
	Results             []titanResult         `json:"results"`
}

type titanResult struct {
	TokenCount       int    `json:"tokenCount"`
	OutputText       string `json:"outputText"`
	CompletionReason string `json:"completionReason"`
}

// AI21 Jurassic request/response structures
type bedrockJurassicRequest struct {
	Prompt      string   `json:"prompt"`
	MaxTokens   int      `json:"maxTokens"`
	Temperature float64  `json:"temperature,omitempty"`
	TopP        float64  `json:"topP,omitempty"`
	StopSequences []string `json:"stopSequences,omitempty"`
}

type bedrockJurassicResponse struct {
	ID          string              `json:"id"`
	Prompt      jurassicPrompt      `json:"prompt"`
	Completions []jurassicCompletion `json:"completions"`
}

type jurassicPrompt struct {
	Text   string `json:"text"`
	Tokens []interface{} `json:"tokens"`
}

type jurassicCompletion struct {
	Data         jurassicData `json:"data"`
	FinishReason jurassicFinishReason `json:"finishReason"`
}

type jurassicData struct {
	Text   string        `json:"text"`
	Tokens []interface{} `json:"tokens"`
}

type jurassicFinishReason struct {
	Reason string `json:"reason"`
}

// Cohere Command request/response structures
type bedrockCommandRequest struct {
	Message       string                  `json:"message"`
	ChatHistory   []cohereMessage         `json:"chat_history,omitempty"`
	Temperature   float64                 `json:"temperature,omitempty"`
	P             float64                 `json:"p,omitempty"`
	MaxTokens     int                     `json:"max_tokens,omitempty"`
	StopSequences []string                `json:"stop_sequences,omitempty"`
	Tools         []cohereTool            `json:"tools,omitempty"`
}

type cohereMessage struct {
	Role    string `json:"role"`
	Message string `json:"message"`
}

type cohereTool struct {
	Name                 string                 `json:"name"`
	Description          string                 `json:"description"`
	ParameterDefinitions map[string]interface{} `json:"parameter_definitions"`
}

type bedrockCommandResponse struct {
	ResponseID   string                 `json:"response_id"`
	Text         string                 `json:"text"`
	GenerationID string                 `json:"generation_id"`
	ChatHistory  []cohereMessage        `json:"chat_history"`
	FinishReason string                 `json:"finish_reason"`
	Meta         cohereMetaData         `json:"meta"`
	ToolCalls    []cohereToolCall       `json:"tool_calls,omitempty"`
}

type cohereMetaData struct {
	APIVersion struct {
		Version string `json:"version"`
	} `json:"api_version"`
	BilledUnits struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"billed_units"`
}

type cohereToolCall struct {
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters"`
}

// Meta Llama request/response structures
type bedrockLlamaRequest struct {
	Prompt      string  `json:"prompt"`
	MaxGenLen   int     `json:"max_gen_len"`
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
}

type bedrockLlamaResponse struct {
	Generation         string `json:"generation"`
	PromptTokenCount   int    `json:"prompt_token_count"`
	GenerationTokenCount int  `json:"generation_token_count"`
	StopReason         string `json:"stop_reason"`
}

// Streaming chunk structures
type bedrockStreamChunk struct {
	Delta        string `json:"delta,omitempty"`
	Text         string `json:"text,omitempty"`
	StopReason   string `json:"stop_reason,omitempty"`
	OutputText   string `json:"outputText,omitempty"` // Titan
	Type         string `json:"type,omitempty"`        // Claude
	ContentBlock *anthropicContentBlock `json:"content_block,omitempty"` // Claude
}

// NewBedrockProvider creates a new AWS Bedrock provider
func NewBedrockProvider(config ProviderConfigEntry) (*BedrockProvider, error) {
	ctx := context.Background()

	// Get region from config or environment
	region := getBedrockRegion(config)

	// Load AWS configuration
	var awsCfg aws.Config
	var err error

	// Check if explicit credentials are provided
	if accessKey, ok := config.Parameters["aws_access_key_id"].(string); ok && accessKey != "" {
		secretKey, _ := config.Parameters["aws_secret_access_key"].(string)
		sessionToken, _ := config.Parameters["aws_session_token"].(string)

		awsCfg, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(region),
			awsconfig.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(accessKey, secretKey, sessionToken),
			),
		)
	} else {
		// Use default AWS credentials (IAM role, env vars, or credentials file)
		awsCfg, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(region),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %v", err)
	}

	// Create Bedrock Runtime client
	client := bedrockruntime.NewFromConfig(awsCfg)

	// Get cross-region inference settings
	crossRegionInference := false
	if cri, ok := config.Parameters["cross_region_inference"].(bool); ok {
		crossRegionInference = cri
	}

	inferenceProfileArn := ""
	if arn, ok := config.Parameters["inference_profile_arn"].(string); ok {
		inferenceProfileArn = arn
	}

	provider := &BedrockProvider{
		config:               config,
		awsConfig:            awsCfg,
		bedrockClient:        client,
		models:               getBedrockModels(),
		region:               region,
		crossRegionInference: crossRegionInference,
		inferenceProfileArn:  inferenceProfileArn,
	}

	return provider, nil
}

// getBedrockRegion extracts region from config or environment
func getBedrockRegion(config ProviderConfigEntry) string {
	// Check config parameters first
	if region, ok := config.Parameters["region"].(string); ok && region != "" {
		return region
	}

	// Check environment variable
	if region := os.Getenv("AWS_REGION"); region != "" {
		return region
	}

	if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
		return region
	}

	// Default to us-east-1
	return "us-east-1"
}

// getBedrockModels returns all available Bedrock models
func getBedrockModels() []ModelInfo {
	allCapabilities := []ModelCapability{
		CapabilityTextGeneration,
		CapabilityCodeGeneration,
		CapabilityCodeAnalysis,
		CapabilityPlanning,
		CapabilityDebugging,
		CapabilityRefactoring,
		CapabilityTesting,
	}

	textCapabilities := []ModelCapability{
		CapabilityTextGeneration,
		CapabilityCodeGeneration,
	}

	return []ModelInfo{
		// Claude 4 family via Bedrock
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
		// Claude 3.7 family via Bedrock
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
		// Claude 3.5 family via Bedrock
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
		{
			Name:           "anthropic.claude-3-5-haiku-20241022-v1:0",
			Provider:       ProviderTypeBedrock,
			ContextSize:    200000,
			MaxTokens:      8192,
			Capabilities:   allCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Claude 3.5 Haiku via AWS Bedrock - Fast and efficient",
		},
		// Claude 3 family via Bedrock
		{
			Name:           "anthropic.claude-3-opus-20240229-v1:0",
			Provider:       ProviderTypeBedrock,
			ContextSize:    200000,
			MaxTokens:      4096,
			Capabilities:   allCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Claude 3 Opus via AWS Bedrock - Most powerful",
		},
		{
			Name:           "anthropic.claude-3-sonnet-20240229-v1:0",
			Provider:       ProviderTypeBedrock,
			ContextSize:    200000,
			MaxTokens:      4096,
			Capabilities:   allCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Claude 3 Sonnet via AWS Bedrock - Balanced",
		},
		{
			Name:           "anthropic.claude-3-haiku-20240307-v1:0",
			Provider:       ProviderTypeBedrock,
			ContextSize:    200000,
			MaxTokens:      4096,
			Capabilities:   allCapabilities,
			SupportsTools:  true,
			SupportsVision: true,
			Description:    "Claude 3 Haiku via AWS Bedrock - Fast",
		},
		// Amazon Titan
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
		// AI21 Jurassic
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
		{
			Name:           "ai21.j2-ultra-v1",
			Provider:       ProviderTypeBedrock,
			ContextSize:    8192,
			MaxTokens:      8192,
			Capabilities:   textCapabilities,
			SupportsTools:  false,
			SupportsVision: false,
			Description:    "AI21 Jurassic-2 Ultra - Advanced language model",
		},
		// Cohere Command
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
		{
			Name:           "cohere.command-r-v1:0",
			Provider:       ProviderTypeBedrock,
			ContextSize:    128000,
			MaxTokens:      4000,
			Capabilities:   allCapabilities,
			SupportsTools:  true,
			SupportsVision: false,
			Description:    "Cohere Command R - Balanced RAG model",
		},
		// Meta Llama
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
		{
			Name:           "meta.llama3-1-70b-instruct-v1:0",
			Provider:       ProviderTypeBedrock,
			ContextSize:    128000,
			MaxTokens:      8192,
			Capabilities:   allCapabilities,
			SupportsTools:  true,
			SupportsVision: false,
			Description:    "Meta Llama 3.1 70B Instruct via Bedrock",
		},
	}
}

// GetType returns the provider type
func (bp *BedrockProvider) GetType() ProviderType {
	return ProviderTypeBedrock
}

// GetName returns the provider name
func (bp *BedrockProvider) GetName() string {
	return "AWS Bedrock"
}

// GetModels returns available models
func (bp *BedrockProvider) GetModels() []ModelInfo {
	return bp.models
}

// GetCapabilities returns provider capabilities
func (bp *BedrockProvider) GetCapabilities() []ModelCapability {
	return []ModelCapability{
		CapabilityTextGeneration,
		CapabilityCodeGeneration,
		CapabilityCodeAnalysis,
		CapabilityPlanning,
		CapabilityDebugging,
		CapabilityRefactoring,
		CapabilityTesting,
		CapabilityVision,
	}
}

// Generate generates a response using Bedrock
func (bp *BedrockProvider) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	startTime := time.Now()

	// Determine model family
	modelFamily := bp.getModelFamily(request.Model)

	// Build model-specific request
	requestBody, err := bp.buildBedrockRequest(request, modelFamily)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %v", err)
	}

	// Get model ID (use inference profile ARN if cross-region inference is enabled)
	modelID := bp.getModelID(request.Model)

	// Invoke model
	output, err := bp.bedrockClient.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(modelID),
		Body:        requestBody,
		Accept:      aws.String("application/json"),
		ContentType: aws.String("application/json"),
	})

	if err != nil {
		return nil, bp.handleBedrockError(err)
	}

	// Parse model-specific response
	response, err := bp.parseBedrockResponse(output.Body, modelFamily, request.ID, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return response, nil
}

// GenerateStream generates a streaming response
func (bp *BedrockProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
	defer close(ch)

	// Determine model family
	modelFamily := bp.getModelFamily(request.Model)

	// Build model-specific request
	requestBody, err := bp.buildBedrockRequest(request, modelFamily)
	if err != nil {
		return fmt.Errorf("failed to build request: %v", err)
	}

	// Get model ID
	modelID := bp.getModelID(request.Model)

	// Invoke model with streaming
	output, err := bp.bedrockClient.InvokeModelWithResponseStream(ctx, &bedrockruntime.InvokeModelWithResponseStreamInput{
		ModelId:     aws.String(modelID),
		Body:        requestBody,
		Accept:      aws.String("application/json"),
		ContentType: aws.String("application/json"),
	})

	if err != nil {
		return bp.handleBedrockError(err)
	}

	// Process event stream
	reader := output.GetStream()
	defer reader.Close()

	return bp.processEventStream(reader, ch, request.ID, modelFamily)
}

// getModelFamily determines the model family from the model ID
func (bp *BedrockProvider) getModelFamily(modelID string) bedrockModelFamily {
	if strings.HasPrefix(modelID, "anthropic.claude") {
		return modelFamilyClaude
	} else if strings.HasPrefix(modelID, "amazon.titan") {
		return modelFamilyTitan
	} else if strings.HasPrefix(modelID, "ai21.") {
		return modelFamilyJurassic
	} else if strings.HasPrefix(modelID, "cohere.command") {
		return modelFamilyCommand
	} else if strings.HasPrefix(modelID, "meta.llama") {
		return modelFamilyLlama
	}
	return modelFamilyClaude // Default to Claude
}

// getModelID returns the model ID, using inference profile ARN if configured
func (bp *BedrockProvider) getModelID(modelID string) string {
	if bp.crossRegionInference && bp.inferenceProfileArn != "" {
		return bp.inferenceProfileArn
	}
	return modelID
}

// buildBedrockRequest builds a model-specific request
func (bp *BedrockProvider) buildBedrockRequest(request *LLMRequest, family bedrockModelFamily) ([]byte, error) {
	switch family {
	case modelFamilyClaude:
		return bp.buildClaudeRequest(request)
	case modelFamilyTitan:
		return bp.buildTitanRequest(request)
	case modelFamilyJurassic:
		return bp.buildJurassicRequest(request)
	case modelFamilyCommand:
		return bp.buildCommandRequest(request)
	case modelFamilyLlama:
		return bp.buildLlamaRequest(request)
	default:
		return nil, fmt.Errorf("unsupported model family: %s", family)
	}
}

// buildClaudeRequest builds a Claude (Anthropic) request
func (bp *BedrockProvider) buildClaudeRequest(request *LLMRequest) ([]byte, error) {
	req := &bedrockClaudeRequest{
		MaxTokens:   request.MaxTokens,
		Temperature: request.Temperature,
		TopP:        request.TopP,
	}

	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}

	// Convert messages
	systemMsg, messages := bp.convertMessagesToAnthropic(request.Messages)
	req.Messages = messages

	// Set system message
	if systemMsg != "" {
		req.System = systemMsg
	}

	// Convert tools
	if len(request.Tools) > 0 {
		req.Tools = bp.convertToolsToAnthropic(request.Tools)
	}

	return json.Marshal(req)
}

// buildTitanRequest builds an Amazon Titan request
func (bp *BedrockProvider) buildTitanRequest(request *LLMRequest) ([]byte, error) {
	// Combine all messages into a single prompt
	prompt := bp.combineMessagesToPrompt(request.Messages)

	req := &bedrockTitanRequest{
		InputText: prompt,
		TextGenerationConfig: &titanTextGenConfig{
			Temperature:   request.Temperature,
			TopP:          request.TopP,
			MaxTokenCount: request.MaxTokens,
		},
	}

	if req.TextGenerationConfig.MaxTokenCount == 0 {
		req.TextGenerationConfig.MaxTokenCount = 4096
	}

	return json.Marshal(req)
}

// buildJurassicRequest builds an AI21 Jurassic request
func (bp *BedrockProvider) buildJurassicRequest(request *LLMRequest) ([]byte, error) {
	prompt := bp.combineMessagesToPrompt(request.Messages)

	req := &bedrockJurassicRequest{
		Prompt:      prompt,
		MaxTokens:   request.MaxTokens,
		Temperature: request.Temperature,
		TopP:        request.TopP,
	}

	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}

	return json.Marshal(req)
}

// buildCommandRequest builds a Cohere Command request
func (bp *BedrockProvider) buildCommandRequest(request *LLMRequest) ([]byte, error) {
	// Extract last user message
	var lastMessage string
	var chatHistory []cohereMessage

	for i, msg := range request.Messages {
		if msg.Role == "system" {
			continue
		}

		if i == len(request.Messages)-1 && msg.Role == "user" {
			lastMessage = msg.Content
		} else {
			role := msg.Role
			if role == "assistant" {
				role = "CHATBOT"
			} else {
				role = "USER"
			}
			chatHistory = append(chatHistory, cohereMessage{
				Role:    role,
				Message: msg.Content,
			})
		}
	}

	req := &bedrockCommandRequest{
		Message:     lastMessage,
		ChatHistory: chatHistory,
		Temperature: request.Temperature,
		P:           request.TopP,
		MaxTokens:   request.MaxTokens,
	}

	if req.MaxTokens == 0 {
		req.MaxTokens = 4000
	}

	// Convert tools
	if len(request.Tools) > 0 {
		req.Tools = bp.convertToolsToCohere(request.Tools)
	}

	return json.Marshal(req)
}

// buildLlamaRequest builds a Meta Llama request
func (bp *BedrockProvider) buildLlamaRequest(request *LLMRequest) ([]byte, error) {
	prompt := bp.combineMessagesToPrompt(request.Messages)

	req := &bedrockLlamaRequest{
		Prompt:      prompt,
		MaxGenLen:   request.MaxTokens,
		Temperature: request.Temperature,
		TopP:        request.TopP,
	}

	if req.MaxGenLen == 0 {
		req.MaxGenLen = 4096
	}

	return json.Marshal(req)
}

// parseBedrockResponse parses a model-specific response
func (bp *BedrockProvider) parseBedrockResponse(body []byte, family bedrockModelFamily, requestID uuid.UUID, startTime time.Time) (*LLMResponse, error) {
	switch family {
	case modelFamilyClaude:
		return bp.parseClaudeResponse(body, requestID, startTime)
	case modelFamilyTitan:
		return bp.parseTitanResponse(body, requestID, startTime)
	case modelFamilyJurassic:
		return bp.parseJurassicResponse(body, requestID, startTime)
	case modelFamilyCommand:
		return bp.parseCommandResponse(body, requestID, startTime)
	case modelFamilyLlama:
		return bp.parseLlamaResponse(body, requestID, startTime)
	default:
		return nil, fmt.Errorf("unsupported model family: %s", family)
	}
}

// parseClaudeResponse parses a Claude response
func (bp *BedrockProvider) parseClaudeResponse(body []byte, requestID uuid.UUID, startTime time.Time) (*LLMResponse, error) {
	var claudeResp bedrockClaudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return nil, fmt.Errorf("failed to parse Claude response: %v", err)
	}

	response := &LLMResponse{
		ID:             uuid.New(),
		RequestID:      requestID,
		ProcessingTime: time.Since(startTime),
		CreatedAt:      time.Now(),
		FinishReason:   claudeResp.StopReason,
		Usage: Usage{
			PromptTokens:     claudeResp.Usage.InputTokens,
			CompletionTokens: claudeResp.Usage.OutputTokens,
			TotalTokens:      claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
		},
	}

	// Extract content and tool calls
	for _, block := range claudeResp.Content {
		switch block.Type {
		case "text":
			response.Content += block.Text
		case "tool_use":
			response.ToolCalls = append(response.ToolCalls, ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: ToolCallFunction{
					Name:      block.Name,
					Arguments: block.Input,
				},
			})
		}
	}

	return response, nil
}

// parseTitanResponse parses a Titan response
func (bp *BedrockProvider) parseTitanResponse(body []byte, requestID uuid.UUID, startTime time.Time) (*LLMResponse, error) {
	var titanResp bedrockTitanResponse
	if err := json.Unmarshal(body, &titanResp); err != nil {
		return nil, fmt.Errorf("failed to parse Titan response: %v", err)
	}

	if len(titanResp.Results) == 0 {
		return nil, fmt.Errorf("no results in Titan response")
	}

	result := titanResp.Results[0]
	totalTokens := titanResp.InputTextTokenCount + result.TokenCount

	return &LLMResponse{
		ID:             uuid.New(),
		RequestID:      requestID,
		Content:        result.OutputText,
		ProcessingTime: time.Since(startTime),
		CreatedAt:      time.Now(),
		FinishReason:   result.CompletionReason,
		Usage: Usage{
			PromptTokens:     titanResp.InputTextTokenCount,
			CompletionTokens: result.TokenCount,
			TotalTokens:      totalTokens,
		},
	}, nil
}

// parseJurassicResponse parses a Jurassic response
func (bp *BedrockProvider) parseJurassicResponse(body []byte, requestID uuid.UUID, startTime time.Time) (*LLMResponse, error) {
	var jurassicResp bedrockJurassicResponse
	if err := json.Unmarshal(body, &jurassicResp); err != nil {
		return nil, fmt.Errorf("failed to parse Jurassic response: %v", err)
	}

	if len(jurassicResp.Completions) == 0 {
		return nil, fmt.Errorf("no completions in Jurassic response")
	}

	completion := jurassicResp.Completions[0]
	promptTokens := len(jurassicResp.Prompt.Tokens)
	completionTokens := len(completion.Data.Tokens)

	return &LLMResponse{
		ID:             uuid.New(),
		RequestID:      requestID,
		Content:        completion.Data.Text,
		ProcessingTime: time.Since(startTime),
		CreatedAt:      time.Now(),
		FinishReason:   completion.FinishReason.Reason,
		Usage: Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}, nil
}

// parseCommandResponse parses a Cohere Command response
func (bp *BedrockProvider) parseCommandResponse(body []byte, requestID uuid.UUID, startTime time.Time) (*LLMResponse, error) {
	var commandResp bedrockCommandResponse
	if err := json.Unmarshal(body, &commandResp); err != nil {
		return nil, fmt.Errorf("failed to parse Command response: %v", err)
	}

	response := &LLMResponse{
		ID:             uuid.New(),
		RequestID:      requestID,
		Content:        commandResp.Text,
		ProcessingTime: time.Since(startTime),
		CreatedAt:      time.Now(),
		FinishReason:   commandResp.FinishReason,
		Usage: Usage{
			PromptTokens:     commandResp.Meta.BilledUnits.InputTokens,
			CompletionTokens: commandResp.Meta.BilledUnits.OutputTokens,
			TotalTokens:      commandResp.Meta.BilledUnits.InputTokens + commandResp.Meta.BilledUnits.OutputTokens,
		},
	}

	// Extract tool calls
	for _, toolCall := range commandResp.ToolCalls {
		response.ToolCalls = append(response.ToolCalls, ToolCall{
			ID:   uuid.New().String(),
			Type: "function",
			Function: ToolCallFunction{
				Name:      toolCall.Name,
				Arguments: toolCall.Parameters,
			},
		})
	}

	return response, nil
}

// parseLlamaResponse parses a Llama response
func (bp *BedrockProvider) parseLlamaResponse(body []byte, requestID uuid.UUID, startTime time.Time) (*LLMResponse, error) {
	var llamaResp bedrockLlamaResponse
	if err := json.Unmarshal(body, &llamaResp); err != nil {
		return nil, fmt.Errorf("failed to parse Llama response: %v", err)
	}

	return &LLMResponse{
		ID:             uuid.New(),
		RequestID:      requestID,
		Content:        llamaResp.Generation,
		ProcessingTime: time.Since(startTime),
		CreatedAt:      time.Now(),
		FinishReason:   llamaResp.StopReason,
		Usage: Usage{
			PromptTokens:     llamaResp.PromptTokenCount,
			CompletionTokens: llamaResp.GenerationTokenCount,
			TotalTokens:      llamaResp.PromptTokenCount + llamaResp.GenerationTokenCount,
		},
	}, nil
}

// processEventStream processes the Bedrock event stream
func (bp *BedrockProvider) processEventStream(reader bedrockruntime.ResponseStreamReader, ch chan<- LLMResponse, requestID uuid.UUID, family bedrockModelFamily) error {
	var contentBuilder strings.Builder

	for event := range reader.Events() {
		switch e := event.(type) {
		case *types.ResponseStreamMemberChunk:
			// Parse chunk based on model family
			chunk, err := bp.parseStreamChunk(e.Value.Bytes, family)
			if err != nil {
				log.Printf("Error parsing chunk: %v", err)
				continue
			}

			delta := chunk.Delta
			if delta == "" {
				delta = chunk.Text
			}
			if delta == "" {
				delta = chunk.OutputText
			}

			if delta != "" {
				contentBuilder.WriteString(delta)

				// Send incremental response
				ch <- LLMResponse{
					ID:        uuid.New(),
					RequestID: requestID,
					Content:   delta,
					CreatedAt: time.Now(),
				}
			}

		default:
			// Handle other event types if needed
			log.Printf("Received unknown event type: %T", e)
		}
	}

	// Check for stream errors
	if err := reader.Err(); err != nil {
		return bp.handleBedrockError(err)
	}

	// Send final complete response
	ch <- LLMResponse{
		ID:           uuid.New(),
		RequestID:    requestID,
		Content:      contentBuilder.String(),
		FinishReason: "stop",
		CreatedAt:    time.Now(),
	}

	return nil
}

// parseStreamChunk parses a streaming chunk
func (bp *BedrockProvider) parseStreamChunk(data []byte, family bedrockModelFamily) (*bedrockStreamChunk, error) {
	var chunk bedrockStreamChunk
	if err := json.Unmarshal(data, &chunk); err != nil {
		return nil, err
	}
	return &chunk, nil
}

// Helper functions for message/tool conversion

func (bp *BedrockProvider) convertMessagesToAnthropic(messages []Message) (string, []anthropicMessage) {
	var systemMsg string
	var anthropicMsgs []anthropicMessage

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			systemMsg = msg.Content
		case "user", "assistant":
			anthropicMsgs = append(anthropicMsgs, anthropicMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	return systemMsg, anthropicMsgs
}

func (bp *BedrockProvider) convertToolsToAnthropic(tools []Tool) []anthropicTool {
	anthropicTools := make([]anthropicTool, len(tools))

	for i, tool := range tools {
		anthropicTools[i] = anthropicTool{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			InputSchema: tool.Function.Parameters,
		}
	}

	return anthropicTools
}

func (bp *BedrockProvider) convertToolsToCohere(tools []Tool) []cohereTool {
	cohereTools := make([]cohereTool, len(tools))

	for i, tool := range tools {
		cohereTools[i] = cohereTool{
			Name:                 tool.Function.Name,
			Description:          tool.Function.Description,
			ParameterDefinitions: tool.Function.Parameters,
		}
	}

	return cohereTools
}

func (bp *BedrockProvider) combineMessagesToPrompt(messages []Message) string {
	var prompt strings.Builder

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			prompt.WriteString(msg.Content)
			prompt.WriteString("\n\n")
		case "user":
			prompt.WriteString("User: ")
			prompt.WriteString(msg.Content)
			prompt.WriteString("\n\n")
		case "assistant":
			prompt.WriteString("Assistant: ")
			prompt.WriteString(msg.Content)
			prompt.WriteString("\n\n")
		}
	}

	return strings.TrimSpace(prompt.String())
}

// handleBedrockError converts AWS SDK errors to provider errors
func (bp *BedrockProvider) handleBedrockError(err error) error {
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

// IsAvailable checks if the provider is available
func (bp *BedrockProvider) IsAvailable(ctx context.Context) bool {
	return bp.bedrockClient != nil
}

// GetHealth returns the health status of the provider
func (bp *BedrockProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	startTime := time.Now()

	health := &ProviderHealth{
		LastCheck:  time.Now(),
		ModelCount: len(bp.models),
	}

	// Test with a minimal request using Claude Haiku (fastest model)
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

// Close closes the provider and cleans up resources
func (bp *BedrockProvider) Close() error {
	log.Printf("AWS Bedrock provider closed")
	return nil
}
