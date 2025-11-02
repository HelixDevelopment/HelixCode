package llm

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockProvider is a mock implementation of Provider for testing
type MockProvider struct {
	mock.Mock
}

func (m *MockProvider) GetType() ProviderType {
	args := m.Called()
	return args.Get(0).(ProviderType)
}

func (m *MockProvider) GetName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockProvider) GetModels() []ModelInfo {
	args := m.Called()
	return args.Get(0).([]ModelInfo)
}

func (m *MockProvider) GetCapabilities() []ModelCapability {
	args := m.Called()
	return args.Get(0).([]ModelCapability)
}

func (m *MockProvider) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LLMResponse), args.Error(1)
}

func (m *MockProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
	args := m.Called(ctx, request, ch)
	return args.Error(0)
}

func (m *MockProvider) IsAvailable(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

func (m *MockProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ProviderHealth), args.Error(1)
}

func (m *MockProvider) Close() error {
	args := m.Called()
	return args.Error(0)
}

// TestReasoningEngine_Creation tests reasoning engine creation
func TestReasoningEngine_Creation(t *testing.T) {
	mockProvider := new(MockProvider)
	engine := NewReasoningEngine(mockProvider)

	assert.NotNil(t, engine)
	assert.Equal(t, mockProvider, engine.provider)
	assert.Equal(t, 10, engine.maxSteps)
	assert.Equal(t, 0.7, engine.temperature)
	assert.NotNil(t, engine.tools)
	assert.Empty(t, engine.tools)
}

// TestReasoningEngine_RegisterTool tests tool registration
func TestReasoningEngine_RegisterTool(t *testing.T) {
	mockProvider := new(MockProvider)
	engine := NewReasoningEngine(mockProvider)

	tool := ReasoningTool{
		Name:        "test_tool",
		Description: "A test tool for reasoning",
		Parameters: map[string]interface{}{
			"param1": "string",
			"param2": "number",
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return "test_result", nil
		},
	}

	// Test successful registration
	err := engine.RegisterTool(tool)
	assert.NoError(t, err)
	assert.Contains(t, engine.tools, "test_tool")
	assert.Equal(t, tool.Name, engine.tools["test_tool"].Name)
	assert.Equal(t, tool.Description, engine.tools["test_tool"].Description)
	assert.Equal(t, tool.Parameters, engine.tools["test_tool"].Parameters)
	assert.NotNil(t, engine.tools["test_tool"].Handler)

	// Test duplicate registration
	err = engine.RegisterTool(tool)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// Test registration with empty name
	err = engine.RegisterTool(ReasoningTool{Name: ""})
	assert.Error(t, err)
}

// TestReasoningEngine_ValidateRequest tests request validation
func TestReasoningEngine_ValidateRequest(t *testing.T) {
	mockProvider := new(MockProvider)
	engine := NewReasoningEngine(mockProvider)

	// Test valid request
	validRequest := ReasoningRequest{
		Prompt:        "Test prompt",
		MaxSteps:      5,
		Temperature:   0.5,
		ReasoningType: ReasoningTypeChainOfThought,
	}

	err := engine.validateRequest(validRequest)
	assert.NoError(t, err)

	// Test invalid requests
	testCases := []struct {
		name    string
		request ReasoningRequest
		error   string
	}{
		{
			name: "empty prompt",
			request: ReasoningRequest{
				Prompt:      "",
				MaxSteps:    5,
				Temperature: 0.5,
			},
			error: "prompt cannot be empty",
		},
		{
			name: "zero max steps",
			request: ReasoningRequest{
				Prompt:      "Test",
				MaxSteps:    0,
				Temperature: 0.5,
			},
			error: "max steps must be positive",
		},
		{
			name: "negative temperature",
			request: ReasoningRequest{
				Prompt:      "Test",
				MaxSteps:    5,
				Temperature: -1.0,
			},
			error: "temperature must be between 0 and 2",
		},
		{
			name: "temperature too high",
			request: ReasoningRequest{
				Prompt:      "Test",
				MaxSteps:    5,
				Temperature: 3.0,
			},
			error: "temperature must be between 0 and 2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := engine.validateRequest(tc.request)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.error)
		})
	}
}

// TestReasoningEngine_GenerateWithReasoning tests reasoning-based generation
func TestReasoningEngine_GenerateWithReasoning(t *testing.T) {
	mockProvider := new(MockProvider)
	engine := NewReasoningEngine(mockProvider)
	ctx := context.Background()

	// Setup mock responses
	mockProvider.On("Generate", mock.Anything, mock.Anything).Return(&LLMResponse{
		Content: "This is a test response with FINAL ANSWER: The answer is 42.",
	}, nil)

	request := ReasoningRequest{
		ID:            uuid.New(),
		Prompt:        "What is the answer to life, the universe, and everything?",
		ReasoningType: ReasoningTypeChainOfThought,
		MaxSteps:      3,
		Temperature:   0.7,
	}

	response, err := engine.GenerateWithReasoning(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "The answer is 42.", response.FinalAnswer)
	assert.NotEmpty(t, response.ID)
	assert.Greater(t, response.Duration, time.Duration(0))
	assert.Empty(t, response.Error)

	mockProvider.AssertExpectations(t)
}
