package mock

import (
	"context"
	"testing"
	"time"
)

func TestNewMockLLMProvider(t *testing.T) {
	provider := NewMockLLMProvider("test-provider")
	
	if provider.GetName() != "test-provider" {
		t.Errorf("Expected name 'test-provider', got '%s'", provider.GetName())
	}
	
	if provider.GetType() != "mock" {
		t.Errorf("Expected type 'mock', got '%s'", provider.GetType())
	}
	
	if !provider.IsAvailable() {
		t.Error("Provider should be available by default")
	}
}

func TestGenerate(t *testing.T) {
	provider := NewMockLLMProvider("test")
	provider.SetResponseDelay(10 * time.Millisecond)
	
	request := &LLMRequest{
		Model: "mock-model",
		Messages: []Message{
			{Role: "user", Content: "Hello, how are you?"},
		},
		MaxTokens:   100,
		Temperature: 0.7,
	}
	
	ctx := context.Background()
	response, err := provider.Generate(ctx, request)
	
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	
	if response == nil {
		t.Fatal("Response is nil")
	}
	
	if response.Content == "" {
		t.Error("Response content is empty")
	}
	
	if response.Usage.TotalTokens == 0 {
		t.Error("Token usage should not be zero")
	}
}

func TestCustomResponses(t *testing.T) {
	provider := NewMockLLMProvider("test")
	provider.SetResponseDelay(1 * time.Millisecond)
	provider.AddResponse("weather", "The weather is sunny today!")
	provider.AddResponse("time", "It's 3:00 PM right now.")
	
	tests := []struct {
		prompt   string
		expected string
	}{
		{"What's the weather like?", "The weather is sunny today!"},
		{"What time is it?", "It's 3:00 PM right now."},
	}
	
	for _, tt := range tests {
		request := &LLMRequest{
			Model: "mock-model",
			Messages: []Message{
				{Role: "user", Content: tt.prompt},
			},
		}
		
		response, err := provider.Generate(context.Background(), request)
		if err != nil {
			t.Fatalf("Generate failed for prompt '%s': %v", tt.prompt, err)
		}
		
		if response.Content != tt.expected {
			t.Errorf("For prompt '%s', expected '%s', got '%s'", 
				tt.prompt, tt.expected, response.Content)
		}
	}
}

func TestUnavailableProvider(t *testing.T) {
	provider := NewMockLLMProvider("test")
	provider.SetAvailable(false)
	
	request := &LLMRequest{
		Model: "mock-model",
		Messages: []Message{
			{Role: "user", Content: "Test"},
		},
	}
	
	_, err := provider.Generate(context.Background(), request)
	if err == nil {
		t.Error("Expected error when provider is unavailable")
	}
}

func TestRequestCount(t *testing.T) {
	provider := NewMockLLMProvider("test")
	provider.SetResponseDelay(1 * time.Millisecond)
	
	initialCount := provider.GetRequestCount()
	
	request := &LLMRequest{
		Model: "mock-model",
		Messages: []Message{
			{Role: "user", Content: "Test"},
		},
	}
	
	for i := 0; i < 5; i++ {
		provider.Generate(context.Background(), request)
	}
	
	finalCount := provider.GetRequestCount()
	expectedCount := initialCount + 5
	
	if finalCount != expectedCount {
		t.Errorf("Expected request count %d, got %d", expectedCount, finalCount)
	}
}

func TestReset(t *testing.T) {
	provider := NewMockLLMProvider("test")
	provider.SetResponseDelay(1 * time.Millisecond)
	provider.AddResponse("test", "response")
	
	request := &LLMRequest{
		Model: "mock-model",
		Messages: []Message{
			{Role: "user", Content: "Test"},
		},
	}
	
	// Generate some requests
	for i := 0; i < 3; i++ {
		provider.Generate(context.Background(), request)
	}
	
	if provider.GetRequestCount() == 0 {
		t.Error("Request count should not be zero before reset")
	}
	
	provider.Reset()
	
	if provider.GetRequestCount() != 0 {
		t.Error("Request count should be zero after reset")
	}
	
	if !provider.IsAvailable() {
		t.Error("Provider should be available after reset")
	}
}

func TestContextCancellation(t *testing.T) {
	provider := NewMockLLMProvider("test")
	provider.SetResponseDelay(500 * time.Millisecond)
	
	request := &LLMRequest{
		Model: "mock-model",
		Messages: []Message{
			{Role: "user", Content: "Test"},
		},
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	
	_, err := provider.Generate(ctx, request)
	if err == nil {
		t.Error("Expected context deadline exceeded error")
	}
}
