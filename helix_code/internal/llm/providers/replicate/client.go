package replicate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"dev.helix.code/internal/llm"
)

const ReplicateBaseURL = "https://api.replicate.com/v1"

type Client struct {
	apiKey       string
	baseURL      string
	client       *http.Client
	pollInterval time.Duration // round-54: test-only override; 0 → default 2s
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: ReplicateBaseURL,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

type replicateInput struct {
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

type replicateRequest struct {
	Input replicateInput `json:"input"`
}

type replicatePrediction struct {
	ID     string      `json:"id"`
	Status string      `json:"status"`
	Output interface{} `json:"output,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// NewClientWithBaseURL constructs a Client whose API endpoint may be
// overridden — used by round-54 LLMResponse.Err pinning tests that
// dispatch against an httptest fixture instead of the real Replicate API.
func NewClientWithBaseURL(apiKey, baseURL string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

// SetPollInterval lets round-54 tests speed up waitForCompletion. Real
// production code retains the documented 2-second default.
func (c *Client) SetPollInterval(d time.Duration) { c.pollInterval = d }

func (c *Client) Generate(ctx context.Context, req *llm.LLMRequest) (*llm.LLMResponse, error) {
	prompt := ""
	if len(req.Messages) > 0 {
		prompt = req.Messages[len(req.Messages)-1].Content
	}
	model := req.Model
	if model == "" {
		model = "meta/meta-llama-3-70b-instruct"
	}
	body := replicateRequest{
		Input: replicateInput{
			Prompt:      prompt,
			MaxTokens:   req.MaxTokens,
			Temperature: req.Temperature,
		},
	}
	if body.Input.MaxTokens == 0 {
		body.Input.MaxTokens = 4096
	}
	if body.Input.Temperature == 0 {
		body.Input.Temperature = 0.7
	}
	data, _ := json.Marshal(body)
	predURL := c.baseURL + "/models/" + model + "/predictions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", predURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("build replicate prediction request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("replicate request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("replicate API error: HTTP %d", resp.StatusCode)
	}
	var pred replicatePrediction
	if err := json.NewDecoder(resp.Body).Decode(&pred); err != nil {
		return nil, fmt.Errorf("decode prediction: %w", err)
	}
	output, finalStatus, finalErrMsg, err := c.waitForCompletion(ctx, pred.ID)
	if err != nil {
		return nil, err
	}
	// Round-54 §11.4 anti-bluff: populate LLMResponse.Err per the Replicate
	// prediction-completion status field — see mapReplicateStatusToErr.
	return &llm.LLMResponse{
		Content: output,
		Err:     mapReplicateStatusToErr(finalStatus, finalErrMsg),
	}, nil
}

// waitForCompletion polls the Replicate predictions endpoint until terminal
// status. Returns (output, status, errorMessage, err). Round-54 added the
// status + errorMessage return values so the caller can populate
// LLMResponse.Err with the correct round-46 / round-54 sentinel.
func (c *Client) waitForCompletion(ctx context.Context, id string) (string, string, string, error) {
	interval := c.pollInterval
	if interval == 0 {
		interval = 2 * time.Second
	}
	for i := 0; i < 30; i++ {
		select {
		case <-ctx.Done():
			return "", "", "", ctx.Err()
		case <-time.After(interval):
		}
		req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/predictions/"+id, nil)
		if err != nil {
			return "", "", "", fmt.Errorf("build replicate poll request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		resp, err := c.client.Do(req)
		if err != nil {
			return "", "", "", err
		}
		var pred replicatePrediction
		json.NewDecoder(resp.Body).Decode(&pred)
		resp.Body.Close()
		switch pred.Status {
		case "succeeded":
			return fmt.Sprintf("%v", pred.Output), pred.Status, "", nil
		case "failed":
			// Round-54: return the failure WITHOUT wrapping in an error so
			// the caller can populate LLMResponse.Err with the round-54
			// sentinel that survives errors.Is() comparisons; the textual
			// message survives via mapReplicateStatusToErr's fmt.Errorf %w.
			return "", pred.Status, pred.Error, nil
		case "canceled":
			// Round-54: canceled is caller-initiated, not an LLM error —
			// surface as a clean LLMResponse with nil Err but flagged status.
			return "", pred.Status, "", nil
		}
	}
	return "", "", "", fmt.Errorf("replicate prediction timed out after 60s")
}

// mapReplicateStatusToErr returns the round-46 / round-54 LLMResponse.Err
// sentinel matching a Replicate prediction-completion `status` value, or
// nil for clean / non-error statuses. Closed mapping per Replicate API docs
// (https://replicate.com/docs/reference/http#predictions.get):
//
//   - "succeeded" → nil
//   - "canceled"  → nil  (caller-initiated cancellation, not an LLM error)
//   - "failed"    → wrap errMsg via fmt.Errorf("%w: %s",
//                                  ErrReplicatePredictionFailed, errMsg) —
//                   so errors.Is(...) succeeds AND the upstream message is
//                   preserved for diagnostics.
//   - other (empty / "processing" / "starting") → nil (non-terminal)
//
// Replicate does not natively expose truncation or content-filter signals
// on its prediction envelope. ErrResponseTruncated and
// ErrResponseContentBlocked will NOT fire from this provider unless the
// underlying model writes them into the output payload — documented for
// downstream callers + future-proofed if Replicate adds those signals.
func mapReplicateStatusToErr(status, errMsg string) error {
	if status == "failed" {
		if errMsg == "" {
			return llm.ErrReplicatePredictionFailed
		}
		return fmt.Errorf("%w: %s", llm.ErrReplicatePredictionFailed, errMsg)
	}
	return nil
}
