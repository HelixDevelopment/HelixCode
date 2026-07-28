package challenges

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"
)

// helixcode_server_client.go — HXC-146: the "rest" challenge interface must
// genuinely drive the deployed HelixCode server's own HTTP API surface, not
// the raw LLM provider APIs a fourth time. This file adds the HTTP client
// that talks to POST /api/v1/llm/generate (internal/server/llm_generate.go
// Server.generateLLM) using the ChallengeConfig.HelixCodeHost/Port/Auth
// fields that were previously declared but never read anywhere
// (dead configuration — see docs investigation for HXC-146).
//
// Anti-bluff (CONST-035 / §11.4.1): every byte in the CompletionResponse
// returned here originates from a real HTTP round-trip against the
// configured HelixCode server. There is no fallback to a raw provider call
// and no fabricated content of any kind.

// ErrHelixCodeServerUnreachable is returned (always wrapped with %w) when the
// configured HelixCode server cannot be reached. Callers use errors.Is to
// distinguish this honest "required dependency not running" precondition
// from a genuine execution failure, so the REST interface can SKIP (§11.4.3)
// rather than fail or silently fall back to a stub.
var ErrHelixCodeServerUnreachable = errors.New("helixcode server unreachable")

// helixCodeMessage mirrors the wire shape of internal/llm.Message's
// JSON-relevant fields (role/content) as consumed by
// internal/server/llm_generate.go's llmGenerateRequest.Messages.
type helixCodeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// helixCodeGenerateRequest mirrors internal/server/llm_generate.go's
// llmGenerateRequest wire shape exactly (POST /api/v1/llm/generate body).
type helixCodeGenerateRequest struct {
	Messages    []helixCodeMessage `json:"messages,omitempty"`
	Model       string             `json:"model,omitempty"`
	Provider    string             `json:"provider,omitempty"`
	MaxTokens   int                `json:"max_tokens,omitempty"`
	Temperature float64            `json:"temperature,omitempty"`
}

// helixCodeGenerateResponse mirrors the JSON body Server.generateLLM writes
// on both the success (200) and error (400/502) paths.
type helixCodeGenerateResponse struct {
	Status   string `json:"status"`
	Content  string `json:"content"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Usage    struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	FinishReason string `json:"finish_reason"`
	Error        string `json:"error"`
}

// helixCodeServerBaseURL builds the configured HelixCode server's base URL
// from the ChallengeConfig fields (previously dead configuration).
//
// The host and port MUST be joined with net.JoinHostPort rather than a plain
// "%s:%d" format: an IPv6 literal is only valid in a URL authority when
// bracketed (RFC 3986 §3.2.2). httptest.newLocalListener falls back to
// "[::1]:0" when it cannot bind IPv4 loopback, and net.SplitHostPort (used by
// the test helper that populates HelixCodeHost) strips those brackets, so
// HelixCodeHost legitimately arrives as the bare "::1". Formatting that
// unbracketed produced "http://::1:<port>", which url.Parse rejects with
// `invalid port "::1:<port>" after host`. Guarded by
// TestHelixCodeServerBaseURL_BracketsIPv6.
//
// A host supplied in the already-bracketed URL form ("[::1]") is unwrapped
// first, because net.JoinHostPort brackets unconditionally on seeing a colon
// and would otherwise emit the equally-invalid "[[::1]]:<port>".
func (e *ChallengeExecutor) helixCodeServerBaseURL() string {
	host := e.config.HelixCodeHost
	if len(host) >= 2 && host[0] == '[' && host[len(host)-1] == ']' {
		host = host[1 : len(host)-1]
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(e.config.HelixCodePort))
}

// checkHelixCodeServerReachable performs a bounded GET /health probe against
// the configured HelixCode server (internal/server/server.go registers
// GET /health via Server.healthCheck). It returns a wrapped
// ErrHelixCodeServerUnreachable when the server cannot be reached, so
// callers can honestly SKIP (§11.4.3) instead of failing or faking a result
// when the required server dependency simply is not running.
func (e *ChallengeExecutor) checkHelixCodeServerReachable(ctx context.Context) error {
	healthCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	url := e.helixCodeServerBaseURL() + "/health"
	httpReq, err := http.NewRequestWithContext(healthCtx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("%w: failed to build health check request: %v", ErrHelixCodeServerUnreachable, err)
	}

	resp, err := e.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%w: GET %s: %v", ErrHelixCodeServerUnreachable, url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: GET %s returned status %d", ErrHelixCodeServerUnreachable, url, resp.StatusCode)
	}

	return nil
}

// callHelixCodeGenerate POSTs a real generation request to the HelixCode
// server's native POST /api/v1/llm/generate endpoint and returns a
// CompletionResponse compatible with the existing parseAndSaveCode pipeline.
//
// This is a genuine HTTP call against the deployed HelixCode server — never
// the raw LLM provider API — so the caller (executeREST) actually exercises
// the server's documented HTTP surface end-to-end instead of re-testing a
// provider's API a fourth time under a different interface label.
func (e *ChallengeExecutor) callHelixCodeGenerate(ctx context.Context, prompt, systemPrompt string, provider LLMProviderType, model string, maxTokens int, temperature float64) (*CompletionResponse, error) {
	reqBody := helixCodeGenerateRequest{
		Messages: []helixCodeMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
		Model:       model,
		Provider:    string(provider),
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal HelixCode server request: %w", err)
	}

	url := e.helixCodeServerBaseURL() + "/api/v1/llm/generate"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to build HelixCode server request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if e.config.HelixCodeAuth != "" {
		httpReq.Header.Set("Authorization", "Bearer "+e.config.HelixCodeAuth)
	}

	resp, err := e.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: POST %s: %v", ErrHelixCodeServerUnreachable, url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read HelixCode server response: %w", err)
	}

	var apiResp helixCodeGenerateResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse HelixCode server response (status %d): %w; body: %s", resp.StatusCode, err, string(body))
	}

	if resp.StatusCode != http.StatusOK || apiResp.Status == "error" {
		errMsg := apiResp.Error
		if errMsg == "" {
			errMsg = string(body)
		}
		return nil, fmt.Errorf("HelixCode server generation failed (status %d): %s", resp.StatusCode, errMsg)
	}

	return &CompletionResponse{
		Content:      apiResp.Content,
		FinishReason: apiResp.FinishReason,
		TokensUsed:   apiResp.Usage.TotalTokens,
	}, nil
}
