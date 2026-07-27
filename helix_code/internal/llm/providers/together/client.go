package together

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"dev.helix.code/internal/llm"
)

const TogetherBaseURL = "https://api.together.xyz/v1/chat/completions"

type Client struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: TogetherBaseURL,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

type togetherMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type togetherRequest struct {
	Model       string            `json:"model"`
	Messages    []togetherMessage `json:"messages"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
}

type togetherChoice struct {
	Message togetherMessage `json:"message"`
}

type togetherResponse struct {
	Choices []togetherChoice `json:"choices"`
}

func (c *Client) Generate(ctx context.Context, req *llm.LLMRequest) (*llm.LLMResponse, error) {
	model := req.Model
	if model == "" {
		model = "mistralai/Mixtral-8x22B-Instruct-v0.1"
	}
	msgs := make([]togetherMessage, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = togetherMessage{Role: m.Role, Content: m.Content}
	}
	body := togetherRequest{
		Model:       model,
		Messages:    msgs,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
	if body.MaxTokens == 0 {
		body.MaxTokens = 4096
	}
	if body.Temperature == 0 {
		body.Temperature = 0.7
	}
	data, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("build together request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("together request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("together API error: HTTP %d", resp.StatusCode)
	}
	var tr togetherResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if len(tr.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}
	return &llm.LLMResponse{Content: tr.Choices[0].Message.Content}, nil
}
