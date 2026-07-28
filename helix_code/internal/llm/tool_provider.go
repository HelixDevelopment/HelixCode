package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"dev.helix.code/internal/tools/persistence"
)

// Enhanced LLM Provider Interface with Tool Calling

// ToolGenerationRequest represents a request for generation with tools
type ToolGenerationRequest struct {
	ID          uuid.UUID              `json:"id"`
	Prompt      string                 `json:"prompt"`
	Tools       []Tool                 `json:"tools"`
	MaxTokens   int                    `json:"max_tokens"`
	Temperature float64                `json:"temperature"`
	Stream      bool                   `json:"stream"`
	Context     map[string]interface{} `json:"context"`
}

// ToolGenerationResponse represents the response from tool-based generation
type ToolGenerationResponse struct {
	ID        uuid.UUID              `json:"id"`
	Text      string                 `json:"text"`
	ToolCalls []ToolCall             `json:"tool_calls"`
	Reasoning string                 `json:"reasoning"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// ToolStreamChunk represents a streaming chunk for tool-based generation
type ToolStreamChunk struct {
	ID        uuid.UUID  `json:"id"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls"`
	Reasoning string     `json:"reasoning"`
	Done      bool       `json:"done"`
	Error     string     `json:"error,omitempty"`
}

// EnhancedLLMProvider extends the base Provider with tool calling capabilities
type EnhancedLLMProvider interface {
	Provider
	GenerateWithTools(ctx context.Context, req ToolGenerationRequest) (*ToolGenerationResponse, error)
	StreamWithTools(ctx context.Context, req ToolGenerationRequest) (<-chan ToolStreamChunk, error)
	ListAvailableTools() []Tool
	RegisterTool(tool Tool) error
}

// ToolExecutor is an interface for executing tools
// This can be implemented by the tools.ToolRegistry
type ToolExecutor interface {
	Execute(ctx context.Context, name string, params map[string]interface{}) (interface{}, error)
}

// ToolCallResult is one tool call's outcome inside a dispatched LLM turn. The
// slice returned by the ordered execution path has exactly one entry per
// requested call, in the SAME order as the input []ToolCall (the LLM-requested
// order) — never completion order, never collapsed by tool name. CallID carries
// the LLM-assigned tool-call ID so callers can correlate two calls to the same
// tool inside one turn (which a name-keyed map would silently merge).
type ToolCallResult struct {
	// CallID is the LLM-assigned tool-call ID (e.g. Anthropic tool_use id).
	CallID string
	// ToolName is the dispatched tool name.
	ToolName string
	// Result is the tool's return value (an informative string on error or
	// when the tool is not registered).
	Result interface{}
	// RanParallel is true when the call ran in the concurrent wave. Diagnostic
	// only — surfaced as anti-bluff evidence that real parallelism happened.
	RanParallel bool
}

// BatchToolExecutor is an OPTIONAL extension a ToolExecutor may also implement
// to dispatch a whole LLM turn through the P3-T04 parallel tool-dispatch
// facility: independent (read-only / side-effect-free / non-conflicting) calls
// run concurrently through a bounded worker pool, while conflicting or
// ordering-dependent calls run serially in request order. Results are assembled
// back in the LLM-requested order, so the turn outcome is identical to fully
// serial execution.
//
// This interface is consumer-defined in internal/llm (rather than imported from
// internal/tools) deliberately: internal/tools transitively imports internal/llm
// — a direct internal/llm → internal/tools import would be an import cycle. The
// concrete tools.ToolRegistry satisfies BatchToolExecutor via its
// ExecuteToolBatch bridging method, which adapts to the lower-level
// tools.ToolRegistry.ExecuteBatch primitive.
//
// maxConcurrency <= 0 selects the registry default (10). The returned slice has
// one ToolCallResult per input call, in input (request) order.
type BatchToolExecutor interface {
	ToolExecutor
	ExecuteToolBatch(ctx context.Context, calls []ToolCall, maxConcurrency int) []ToolCallResult
}

// ToolCallingProvider implements EnhancedLLMProvider with tool calling support
type ToolCallingProvider struct {
	baseProvider       Provider
	tools              map[string]Tool
	toolExecutor       ToolExecutor
	persistenceManager *persistence.Manager
}

// NewToolCallingProvider creates a new tool calling provider
func NewToolCallingProvider(baseProvider Provider) *ToolCallingProvider {
	return &ToolCallingProvider{
		baseProvider: baseProvider,
		tools:        make(map[string]Tool),
	}
}

// SetToolExecutor sets the tool executor (typically a tools.ToolRegistry)
func (p *ToolCallingProvider) SetToolExecutor(executor ToolExecutor) {
	p.toolExecutor = executor
}

// SetPersistenceManager wires a persistence.Manager so that tool-result
// outputs above the threshold are written to disk and substituted with
// a path-reference in the final prompt. A nil manager disables persistence.
func (p *ToolCallingProvider) SetPersistenceManager(m *persistence.Manager) {
	p.persistenceManager = m
}

// persistedToolResult pairs a PersistedResult with its tool-call ID so the
// final-prompt builder can render results in the LLM-requested order. Keying by
// tool name alone (the pre-P3-T04 behaviour) silently merged two calls to the
// same tool in one turn and rendered results in random Go-map-iteration order —
// both correctness defects that this ID-keyed slice fixes.
type persistedToolResult struct {
	callID    string
	persisted *persistence.PersistedResult
}

// persistResults wraps each tool result through MaybePersist, PRESERVING the
// LLM-requested call order. Non-string results are stringified via
// fmt.Sprintf("%v", v) before the size check. The output slice has one entry
// per input ToolCallResult, in the same order — never collapsed by tool name.
func (p *ToolCallingProvider) persistResults(raw []ToolCallResult) []persistedToolResult {
	out := make([]persistedToolResult, 0, len(raw))
	for _, r := range raw {
		var s string
		switch v := r.Result.(type) {
		case string:
			s = v
		default:
			s = fmt.Sprintf("%v", v)
		}
		res, err := p.persistenceManager.MaybePersist(r.ToolName, "", s)
		if err != nil {
			res = &persistence.PersistedResult{Output: s, ToolName: r.ToolName}
		}
		out = append(out, persistedToolResult{callID: r.CallID, persisted: res})
	}
	return out
}

// GenerateWithTools performs generation with tool calling support
func (p *ToolCallingProvider) GenerateWithTools(ctx context.Context, req ToolGenerationRequest) (*ToolGenerationResponse, error) {
	startTime := time.Now()

	// Build tool-enhanced prompt
	enhancedPrompt := p.buildToolEnhancedPrompt(req.Prompt, req.Tools)

	// Generate initial response
	genReq := &LLMRequest{
		Model:       "default",
		Messages:    []Message{{Role: "user", Content: enhancedPrompt}},
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      false,
	}

	resp, err := p.baseProvider.Generate(ctx, genReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate with tools: %v", err)
	}

	// Parse tool calls from response
	toolCalls, reasoning := p.extractToolCallsAndReasoning(resp.Content)

	// Execute tool calls if any
	if len(toolCalls) > 0 {
		results, err := p.executeToolCalls(ctx, toolCalls)
		if err != nil {
			log.Printf("Warning: Some tool calls failed: %v", err)
		}

		// Generate final response with tool results — results are kept in the
		// LLM-requested order through persistResults + buildFinalPrompt.
		finalPrompt := p.buildFinalPrompt(req.Prompt, resp.Content, p.persistResults(results))
		genReq.Messages = []Message{{Role: "user", Content: finalPrompt}}

		finalResp, err := p.baseProvider.Generate(ctx, genReq)
		if err != nil {
			return nil, fmt.Errorf("failed to generate final response: %v", err)
		}
		resp = finalResp
	}

	return &ToolGenerationResponse{
		ID:        uuid.New(),
		Text:      resp.Content,
		ToolCalls: toolCalls,
		Reasoning: reasoning,
		Metadata: map[string]interface{}{
			"duration_ms": time.Since(startTime).Milliseconds(),
			"tools_used":  len(toolCalls),
		},
	}, nil
}

// StreamWithTools performs streaming generation with tool calling support
func (p *ToolCallingProvider) StreamWithTools(ctx context.Context, req ToolGenerationRequest) (<-chan ToolStreamChunk, error) {
	ch := make(chan ToolStreamChunk, 100)

	go func() {
		defer close(ch)

		// Build tool-enhanced prompt
		enhancedPrompt := p.buildToolEnhancedPrompt(req.Prompt, req.Tools)

		// Stream initial response
		streamReq := &LLMRequest{
			Model:       "default",
			Messages:    []Message{{Role: "user", Content: enhancedPrompt}},
			MaxTokens:   req.MaxTokens,
			Temperature: req.Temperature,
			Stream:      true,
		}

		// ANTI-BLUFF FIX (deterministic deadlock, not a race): the Provider
		// interface's documented contract (doc.go:83-93) requires
		// GenerateStream be invoked in its OWN goroutine so the caller can
		// drain the channel CONCURRENTLY while it streams. The prior code
		// called GenerateStream INLINE (synchronously) and only started
		// draining streamCh AFTER it returned. streamCh is buffered at 100;
		// any response exceeding 100 chunks made the provider's internal
		// `ch <- LLMResponse{...}` block FOREVER on a channel nobody was
		// draining yet — a guaranteed hang for any non-trivial generation,
		// not a probabilistic race. Fix: launch GenerateStream in its own
		// goroutine and begin draining immediately, per the documented
		// contract. genErrCh is buffered at 1 so the producer goroutine's
		// final send never blocks even if this function has already
		// returned via some other exit path (defensive; today it doesn't).
		streamCh := make(chan LLMResponse, 100)
		genErrCh := make(chan error, 1)
		go func() {
			genErrCh <- p.baseProvider.GenerateStream(ctx, streamReq, streamCh)
		}()

		var fullResponse string
		var toolCalls []ToolCall
		var reasoning string

		var firstStreamErr error
		for resp := range streamCh {
			// Round-46 LLMResponse.Err honoring (CONST-035 / Article XI
			// §11.9; closes round-33 anchored limitation): the streamCh
			// now carries a partial-error signal in resp.Err. Strategy
			// chosen for round 46: forward partial content + propagate
			// the error verbatim on the next ToolStreamChunk's Error
			// field so consumers see WHICH chunk degraded; also log the
			// error so operators see it in the helix-server log. We do
			// NOT abort the loop on a partial error — Content may hold
			// usable bytes the consumer wants. The first observed Err
			// is remembered and surfaced on the terminal chunk so that
			// downstream callers who only inspect the final frame still
			// observe the failure.
			if resp.Err != nil && firstStreamErr == nil {
				firstStreamErr = resp.Err
				log.Printf("tool_provider: stream chunk carried partial error: %v", resp.Err)
			}
			fullResponse += resp.Content

			// Send streaming chunk; surface partial-error on the chunk
			// that carried it so consumers can react chunk-by-chunk.
			chunk := ToolStreamChunk{
				ID:        uuid.New(),
				Content:   resp.Content,
				ToolCalls: []ToolCall{},
				Reasoning: "",
				Done:      false,
			}
			if resp.Err != nil {
				chunk.Error = resp.Err.Error()
			}
			ch <- chunk
		}
		// If the stream ended with a remembered partial error and we
		// have NO tool calls to chase, emit a terminal chunk so the
		// consumer's `for chunk := range ch` loop observes the error
		// even when the underlying streamCh closed silently. When tool
		// calls follow (handled below) the error rides on the next
		// loop's terminal chunk instead.
		if firstStreamErr != nil {
			ch <- ToolStreamChunk{
				ID:    uuid.New(),
				Error: fmt.Sprintf("upstream LLM partial error: %v", firstStreamErr),
				Done:  false,
			}
		}

		// Surface a hard GenerateStream failure (distinct from a per-chunk
		// partial error carried on resp.Err and handled above) ONLY AFTER
		// the drain loop has forwarded every chunk the provider emitted
		// before failing — never before, so partial content is never
		// silently discarded the way a pre-drain synchronous check would.
		// genErrCh is guaranteed to have exactly one value by the time
		// streamCh closes: every GenerateStream implementation in this
		// package closes its ch via `defer close(ch)` as its very first
		// deferred statement, so streamCh's close always happens no later
		// than the genErrCh send that immediately follows GenerateStream's
		// return in the producer goroutine above.
		if genErr := <-genErrCh; genErr != nil {
			ch <- ToolStreamChunk{
				ID:    uuid.New(),
				Error: fmt.Sprintf("Failed to start streaming: %v", genErr),
				Done:  true,
			}
			return
		}

		// Parse tool calls after streaming completes
		toolCalls, reasoning = p.extractToolCallsAndReasoning(fullResponse)

		// Execute tool calls if any
		if len(toolCalls) > 0 {
			results, err := p.executeToolCalls(ctx, toolCalls)
			if err != nil {
				log.Printf("Warning: Some tool calls failed: %v", err)
			}

			// Generate final response with tool results — results stay in the
			// LLM-requested order through persistResults + buildFinalPrompt.
			finalPrompt := p.buildFinalPrompt(req.Prompt, fullResponse, p.persistResults(results))

			// Stream final response
			finalStreamReq := &LLMRequest{
				Model:       "default",
				Messages:    []Message{{Role: "user", Content: finalPrompt}},
				MaxTokens:   req.MaxTokens,
				Temperature: req.Temperature,
				Stream:      true,
			}

			// Same deadlock class as the initial stream above (§11.4.115 /
			// doc.go:83-93 contract): GenerateStream MUST be invoked in its
			// own goroutine so the drain loop below runs CONCURRENTLY with
			// it, never after. finalStreamCh is buffered at 100 exactly
			// like streamCh above — a final response over 100 chunks would
			// deadlock this goroutine forever under the pre-fix synchronous
			// call-then-drain ordering.
			finalStreamCh := make(chan LLMResponse, 100)
			finalGenErrCh := make(chan error, 1)
			go func() {
				finalGenErrCh <- p.baseProvider.GenerateStream(ctx, finalStreamReq, finalStreamCh)
			}()

			for resp := range finalStreamCh {
				// Round-46 LLMResponse.Err honoring (CONST-035 /
				// Article XI §11.9; closes round-33 anchored
				// limitation): resp.Err now carries partial-error
				// state when the upstream provider hits truncation,
				// content-safety block, or mid-stream parse error.
				// Strategy: same as the first loop — log + propagate
				// on the chunk's Error field. Do NOT skip the chunk:
				// Content may still hold useful bytes.
				chunk := ToolStreamChunk{
					ID:        uuid.New(),
					Content:   resp.Content,
					ToolCalls: toolCalls,
					Reasoning: reasoning,
					Done:      true, // Assume done when we get the final response
				}
				if resp.Err != nil {
					log.Printf("tool_provider: final-stream chunk carried partial error: %v", resp.Err)
					chunk.Error = resp.Err.Error()
				}
				ch <- chunk
			}

			// Surface a hard GenerateStream failure AFTER every pre-error
			// chunk has already been forwarded above (same rationale as
			// the initial-stream genErrCh check).
			if err := <-finalGenErrCh; err != nil {
				ch <- ToolStreamChunk{
					ID:    uuid.New(),
					Error: fmt.Sprintf("Failed to stream final response: %v", err),
					Done:  true,
				}
				return
			}
		} else {
			// No tool calls, send final chunk
			ch <- ToolStreamChunk{
				ID:        uuid.New(),
				Content:   "",
				ToolCalls: toolCalls,
				Reasoning: reasoning,
				Done:      true,
			}
		}
	}()

	return ch, nil
}

// ListAvailableTools returns all registered tools
func (p *ToolCallingProvider) ListAvailableTools() []Tool {
	tools := make([]Tool, 0, len(p.tools))
	for _, tool := range p.tools {
		tools = append(tools, tool)
	}
	return tools
}

// RegisterTool registers a new tool with the provider
func (p *ToolCallingProvider) RegisterTool(tool Tool) error {
	if _, exists := p.tools[tool.Function.Name]; exists {
		return fmt.Errorf("tool %s already registered", tool.Function.Name)
	}
	p.tools[tool.Function.Name] = tool

	log.Printf("Tool registered: %s", tool.Function.Name)
	return nil
}

// Implement base Provider interface

func (p *ToolCallingProvider) GetType() ProviderType {
	return p.baseProvider.GetType()
}

func (p *ToolCallingProvider) GetName() string {
	return p.baseProvider.GetName()
}

func (p *ToolCallingProvider) GetModels() []ModelInfo {
	return p.baseProvider.GetModels()
}

func (p *ToolCallingProvider) GetCapabilities() []ModelCapability {
	return p.baseProvider.GetCapabilities()
}

func (p *ToolCallingProvider) Generate(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	return p.baseProvider.Generate(ctx, req)
}

func (p *ToolCallingProvider) GenerateStream(ctx context.Context, req *LLMRequest, ch chan<- LLMResponse) error {
	return p.baseProvider.GenerateStream(ctx, req, ch)
}

func (p *ToolCallingProvider) IsAvailable(ctx context.Context) bool {
	return p.baseProvider.IsAvailable(ctx)
}

func (p *ToolCallingProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	return p.baseProvider.GetHealth(ctx)
}

func (p *ToolCallingProvider) Close() error {
	return p.baseProvider.Close()
}

// GetContextWindow delegates to the underlying base provider.
func (p *ToolCallingProvider) GetContextWindow() int {
	return p.baseProvider.GetContextWindow()
}

// CountTokens delegates to the underlying base provider.
func (p *ToolCallingProvider) CountTokens(text string) (int, error) {
	return p.baseProvider.CountTokens(text)
}

// Helper methods

func (p *ToolCallingProvider) buildToolEnhancedPrompt(prompt string, tools []Tool) string {
	toolDescriptions := ""
	for _, tool := range tools {
		paramsJSON, _ := json.Marshal(tool.Function.Parameters)
		toolDescriptions += fmt.Sprintf("- %s: %s (parameters: %s)\n",
			tool.Function.Name, tool.Function.Description, string(paramsJSON))
	}

	return fmt.Sprintf(`You have access to the following tools:
%s

When you need to use a tool, specify it in this format:
TOOL_CALL: {"tool_name": "tool_name", "arguments": {...}}

After using tools, provide your final answer.

User request: %s

Your response:`, toolDescriptions, prompt)
}

func (p *ToolCallingProvider) extractToolCallsAndReasoning(text string) ([]ToolCall, string) {
	var toolCalls []ToolCall
	reasoning := ""

	// Line-by-line TOOL_CALL extractor: scan each line for the
	// TOOL_CALL: marker, slice from the first '{' to the last '}', and
	// json.Unmarshal into a ToolCall. Non-marker lines accumulate as
	// reasoning. This honest simple parser matches the prompt template
	// emitted by buildToolEnhancedPrompt above; it does not pretend to
	// be a streaming-tolerant or multi-call-per-line parser. If the
	// upstream prompt format ever changes, extend here rather than
	// claiming the existing logic is just a placeholder.
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if strings.Contains(line, "TOOL_CALL:") {
			// Extract JSON from tool call
			jsonStart := strings.Index(line, "{")
			jsonEnd := strings.LastIndex(line, "}")
			if jsonStart != -1 && jsonEnd != -1 {
				jsonStr := line[jsonStart : jsonEnd+1]
				var toolCall ToolCall
				if err := json.Unmarshal([]byte(jsonStr), &toolCall); err == nil {
					toolCalls = append(toolCalls, toolCall)
				}
			}
		} else if !strings.Contains(line, "TOOL_CALL:") {
			// Collect reasoning (non-tool-call lines)
			reasoning += line + "\n"
		}
	}

	return toolCalls, strings.TrimSpace(reasoning)
}

// executeToolCalls runs every tool call requested in a single LLM turn and
// returns an ORDERED slice with exactly one ToolCallResult per input call, in
// the LLM-requested order.
//
// P3-T04 follow-up — the live parallel-dispatch path. When the configured
// ToolExecutor also implements BatchToolExecutor (the concrete
// tools.ToolRegistry does), the whole turn is routed through ExecuteToolBatch:
// independent (read-only / side-effect-free / non-conflicting) calls run
// concurrently through a bounded worker pool, conflicting / ordering-dependent
// calls run serially in request order, and results are assembled back in
// request order — identical outcome to fully serial execution, only faster.
//
// When the executor does NOT implement BatchToolExecutor (or none is wired) the
// function degrades to the serial path below, which is byte-for-byte equivalent
// to the pre-P3-T04 behaviour except that it now returns an ordered slice
// instead of a name-keyed map (the name-keyed map silently merged two calls to
// the same tool inside one turn — a correctness defect).
func (p *ToolCallingProvider) executeToolCalls(ctx context.Context, toolCalls []ToolCall) ([]ToolCallResult, error) {
	// Fast path: the executor supports batch dispatch — route the whole turn
	// through the P3-T04 parallel facility so independent calls run concurrently.
	if batcher, ok := p.toolExecutor.(BatchToolExecutor); ok {
		batch := batcher.ExecuteToolBatch(ctx, toolCalls, 0)
		results := make([]ToolCallResult, len(batch))
		for i, b := range batch {
			results[i] = b
			// Normalise tool-not-found / tool-error into the same informative
			// string shape the serial path produces, so downstream prompt
			// rendering is identical regardless of dispatch path.
			if _, exists := p.tools[b.ToolName]; !exists && b.Result == nil {
				results[i].Result = fmt.Sprintf("Tool not found: %s", b.ToolName)
			}
		}
		return results, nil
	}

	// Serial path: no batch-capable executor wired — run calls in request order.
	results := make([]ToolCallResult, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		entry := ToolCallResult{CallID: toolCall.ID, ToolName: toolCall.Function.Name}
		if _, exists := p.tools[toolCall.Function.Name]; !exists {
			entry.Result = fmt.Sprintf("Tool not found: %s", toolCall.Function.Name)
			results = append(results, entry)
			continue
		}
		result, err := p.executeToolHandler(ctx, toolCall.Function.Name, toolCall.Function.Arguments)
		if err != nil {
			entry.Result = fmt.Sprintf("Tool error: %v", err)
		} else {
			entry.Result = result
		}
		results = append(results, entry)
	}
	return results, nil
}

// executeToolHandler executes a tool handler based on the tool name
func (p *ToolCallingProvider) executeToolHandler(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	// If a tool executor is configured, use it
	if p.toolExecutor != nil {
		result, err := p.toolExecutor.Execute(ctx, toolName, args)
		if err != nil {
			return nil, fmt.Errorf("tool execution failed for %s: %w", toolName, err)
		}
		return result, nil
	}

	// Verify tool is registered (for logging purposes)
	_, exists := p.tools[toolName]
	if !exists {
		log.Printf("Warning: Tool %s not found in registered tools", toolName)
	}

	// No executor available - return informative message
	// Tools in this provider are metadata only; actual execution requires a ToolExecutor
	log.Printf("Warning: Tool %s called but no executor configured. Configure a ToolExecutor with SetToolExecutor().", toolName)
	return map[string]interface{}{
		"status":  "no_executor",
		"tool":    toolName,
		"args":    args,
		"message": "Tool execution requires a configured ToolExecutor. Use SetToolExecutor() to configure the tools.ToolRegistry.",
	}, nil
}

// buildFinalPrompt renders the tool-execution results into the final-answer
// prompt. The toolResults slice is in the LLM-requested call order — each entry
// is rendered in that order so the LLM sees results positioned exactly as it
// requested them, even when a turn calls the same tool more than once.
func (p *ToolCallingProvider) buildFinalPrompt(originalPrompt, initialResponse string, toolResults []persistedToolResult) string {
	resultsStr := ""
	for _, entry := range toolResults {
		result := entry.persisted
		if result.WasPersisted {
			resultsStr += fmt.Sprintf("- %s: [persisted to %s — %d chars. Use Read with that path to fetch full content.]\n",
				result.ToolName, result.PersistedOutputPath, result.PersistedOutputSize)
		} else {
			resultsStr += fmt.Sprintf("- %s: %v\n", result.ToolName, result.Output)
		}
	}

	return fmt.Sprintf(`Original request: %s

Initial response: %s

Tool execution results:
%s

Based on the tool results, provide your final answer:`,
		originalPrompt, initialResponse, resultsStr)
}
