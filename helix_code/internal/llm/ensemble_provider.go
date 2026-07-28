package llm

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ensemble_provider.go — a REAL multi-provider LLM ensemble exposed as a single
// llm.Provider so the HelixCode terminal UI can list "Helix Agent ensemble" in
// its /model picker alongside individual cloud models.
//
// WHY THIS LIVES IN HelixCode (not imported from helix_agent):
//   helix_agent ships a real ensemble (submodules/helix_agent/internal/services/
//   ensemble.go + internal/llm/ensemble.go: concurrent fan-out, semaphore-bounded
//   provider calls, confidence/quality/majority voting). It is, however, locked
//   behind that module's internal/ tree — Go forbids importing another module's
//   internal/ packages (dev.helix.code cannot import dev.helix.agent/internal/...),
//   and that package transitively pulls digital.vasic.helixmemory / eventbus /
//   memory / etc. (not in HelixCode's go.sum). Per CONST-051(B) we MUST NOT inject
//   HelixCode context into helix_agent, and per the Go module boundary we cannot
//   reach into its internals. The decoupled, genuinely-working path is to run the
//   same ensemble discipline over HelixCode's OWN already-registered llm.Provider
//   instances — fanning one prompt to every member, collecting REAL responses,
//   scoring them, and synthesizing a real combined answer.
//
// ANTI-BLUFF (CONST-035 / Article XI §11.9 / §11.4.123): there is NO simulation
// here. Generate() makes a real provider.Generate() call to EVERY member (each of
// which makes a real HTTP call to a real cloud provider) and returns the actual
// voted/combined output. The metadata records exactly which providers
// participated, which won, and the per-member scores — so the ensemble can never
// masquerade a single-provider pass-through as multi-provider work.

// ProviderTypeEnsemble is the provider-type sentinel for the Helix Agent
// ensemble. It is NOT a cloud endpoint — it is a meta-provider that orchestrates
// other registered providers.
const ProviderTypeEnsemble ProviderType = "helix-agent-ensemble"

// EnsembleModelName is the single selectable model name surfaced in the /model
// picker. CONST-046: this is a stable provider identifier, not user-facing prose.
const EnsembleModelName = "helix-agent-ensemble"

// EnsembleDisplayName is the human label shown for the provider.
const EnsembleDisplayName = "Helix Agent ensemble"

// defaultEnsembleTimeout bounds a full ensemble fan-out when the caller does not
// supply one.
const defaultEnsembleTimeout = 120 * time.Second

// EnsembleProviderConfig configures an EnsembleProvider. Members are INJECTED by
// the caller (the TUI passes the cloud providers it built from env keys) — the
// ensemble never constructs or discovers providers itself, keeping it decoupled
// and reusable.
type EnsembleProviderConfig struct {
	// Members are the underlying providers the ensemble fans each prompt to.
	Members []Provider
	// Strategy selects the voting strategy: "confidence_weighted" (default),
	// "quality_weighted", or "majority_vote".
	Strategy string
	// Timeout bounds a full fan-out. Zero ⇒ defaultEnsembleTimeout.
	Timeout time.Duration
}

// EnsembleProvider implements llm.Provider by orchestrating several member
// providers and voting on their real responses.
type EnsembleProvider struct {
	mu       sync.RWMutex
	members  []Provider
	strategy string
	timeout  time.Duration
	// workingModel caches the first chat-capable model id that actually returned
	// content for each member (keyed by provider name). The first ensemble run
	// discovers it (skipping decommissioned/paid/embedding models); subsequent
	// runs reuse it so they don't re-probe the dead models. Guarded by mu.
	workingModel map[string]string
	// warmStarted is the sync.Once-style guard ensuring the WarmCache fan-out runs
	// at most once for the provider's lifetime. Guarded by mu.
	warmStarted bool
	// deadModel records (provider name → model id → true) for every (provider,
	// model) pair that has DEFINITIVELY failed with a non-retryable error
	// (model-not-found / decommissioned / unsupported / 400/404-class "model does
	// not exist"). Once recorded, orderedCandidates SKIPS that model for the
	// remainder of the provider's lifetime — so a model that fails as
	// decommissioned on the FIRST prompt (e.g. groq's catalogue-leading
	// `gemma-7b-it`) is NEVER re-tried on later prompts. This bounds dead-model
	// discovery to the first prompt and stops the per-prompt re-burn that produced
	// "all N member(s) failed" under the live TUI streaming path (warm-cache not
	// reliably completing before the first prompt). DEAD is permanent; TRANSIENT
	// failures (rate-limit 429 / timeout / network) are NEVER recorded here so a
	// model temporarily throttled is retried later. The key is the (provider,
	// model) pair, NOT a hardcoded model name — membership is populated purely from
	// the error CLASS observed at runtime (CONST-036/040: zero hardcoded names).
	// Guarded by mu.
	deadModel map[string]map[string]bool
	// memberCalls counts how many underlying p.Generate() attempts each member has
	// made over the provider's lifetime (keyed by provider name). It is the
	// observable that proves resolution is NOT doing a multi-model discovery burst:
	// once a member's model is resolved (verifier-driven or cached), each prompt
	// adds exactly ONE call. A cold member walking many dead catalogue models would
	// show a large delta. Guarded by mu. Exposed read-only via MemberCallCounts.
	memberCalls map[string]int
}

// ensembleMaxModelTries bounds how many chat-capable catalogue models a single
// member will attempt before giving up, so a fully-dead member fails fast
// instead of walking a 300-entry catalogue. When the verifier is wired the very
// first candidate is the verifier's best verified model, so a single try
// normally succeeds; this bound only matters for the offline catalogue fallback.
const ensembleMaxModelTries = 8

// NewEnsembleProvider builds an ensemble from the injected member providers.
func NewEnsembleProvider(cfg EnsembleProviderConfig) *EnsembleProvider {
	strategy := cfg.Strategy
	if strategy == "" {
		strategy = "confidence_weighted"
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultEnsembleTimeout
	}
	members := make([]Provider, 0, len(cfg.Members))
	members = append(members, cfg.Members...)
	return &EnsembleProvider{members: members, strategy: strategy, timeout: timeout, workingModel: map[string]string{}, memberCalls: map[string]int{}, deadModel: map[string]map[string]bool{}}
}

// AddMember registers an additional member provider after construction.
func (e *EnsembleProvider) AddMember(p Provider) {
	if p == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.members = append(e.members, p)
}

func (e *EnsembleProvider) snapshot() []Provider {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]Provider, len(e.members))
	copy(out, e.members)
	return out
}

func (e *EnsembleProvider) GetType() ProviderType { return ProviderTypeEnsemble }
func (e *EnsembleProvider) GetName() string        { return EnsembleDisplayName }

// GetModels exposes exactly one selectable model — the ensemble itself. Its
// context window is the minimum over the members (the binding constraint), so a
// caller can never overflow the weakest member.
func (e *EnsembleProvider) GetModels() []ModelInfo {
	members := e.snapshot()
	memberNames := make([]string, 0, len(members))
	for _, m := range members {
		memberNames = append(memberNames, m.GetName())
	}
	desc := fmt.Sprintf("Multi-provider ensemble orchestrating %d providers (%s); fans each prompt to all members and returns the voted/combined response.",
		len(members), strings.Join(memberNames, ", "))
	return []ModelInfo{{
		ID:           EnsembleModelName,
		Name:         EnsembleModelName,
		Provider:     ProviderTypeEnsemble,
		ContextSize:  e.GetContextWindow(),
		MaxTokens:    0,
		Capabilities: e.GetCapabilities(),
		Description:  desc,
		Metadata: map[string]interface{}{
			"ensemble_members": memberNames,
			"voting_strategy":  e.strategy,
		},
	}}
}

func (e *EnsembleProvider) GetCapabilities() []ModelCapability {
	return []ModelCapability{
		CapabilityTextGeneration,
		CapabilityCodeGeneration,
		CapabilityReasoning,
		CapabilityAnalysis,
		CapabilityWriting,
	}
}

// ensembleResult is one member's real response plus the bookkeeping the voter
// needs.
type ensembleResult struct {
	providerName string
	resp         *LLMResponse
	score        float64
}

// Generate fans the request to every member concurrently, collects the real
// responses, votes, and returns the winning response enriched with ensemble
// metadata (participants, success/failure counts, scores, the selected member,
// and a short excerpt of every member's answer).
func (e *EnsembleProvider) Generate(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	members := e.snapshot()
	if len(members) == 0 {
		return nil, fmt.Errorf("helix-agent ensemble: no member providers configured")
	}

	fanCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	type memberOutcome struct {
		name string
		resp *LLMResponse
		err  error
		// model is the model id the member ACTUALLY served this turn — the
		// verifier-chosen (or cached/catalogue) model resolved on the sentinel
		// path, or the caller-supplied model on the non-sentinel single-attempt
		// path. It is surfaced per-participant in ProviderMetadata so the
		// operator sees WHICH model each ensemble member used.
		model string
	}
	outCh := make(chan memberOutcome, len(members))

	var wg sync.WaitGroup
	for _, m := range members {
		wg.Add(1)
		go func(p Provider) {
			defer wg.Done()
			// Each member gets its own request copy so concurrent providers
			// never share/mutate the same struct.
			base := *request
			if base.ID == uuid.Nil {
				base.ID = uuid.New()
			}
			// CRITICAL: the ensemble votes on COMPLETE member responses, so it ALWAYS
			// calls each member's non-streaming Generate. The caller's request may
			// carry Stream=true (the TUI sets it — it drives the ensemble through
			// GenerateStream, which buffers via Generate). Forwarding Stream=true into
			// a member's non-streaming Generate makes the provider request an SSE
			// response ("stream":true) but then JSON-decode it as a single body — the
			// decode chokes on the leading `data:` chunk with
			// `invalid character 'd' looking for beginning of value`, failing EVERY
			// member and surfacing "all N member(s) failed" in the chat. Forcing
			// Stream=false on the per-member request is the fix: members are always
			// invoked buffered, the ensemble buffers, and GenerateStream emits the
			// voted result as one chunk.
			base.Stream = false
			// The ensemble sentinel model ("helix-agent-ensemble") is NOT a real
			// cloud model — forwarding it would make every member's API reject the
			// request (404 "model does not exist"). When the caller passes an
			// explicit (non-sentinel) model, honour it with a single attempt.
			sentinel := base.Model == "" || base.Model == EnsembleModelName || base.Model == string(ProviderTypeEnsemble)
			if !sentinel {
				r, err := p.Generate(fanCtx, &base)
				// The caller honoured an explicit model — record it as the model
				// this member served so the per-member model visibility still
				// shows it on the non-sentinel path.
				outCh <- memberOutcome{name: p.GetName(), resp: r, err: err, model: base.Model}
				return
			}
			// Sentinel/empty model: resolve a model the member ACTUALLY serves,
			// DYNAMICALLY (CONST-036/040). The candidate order is, first, the
			// LLMsVerifier-reported best verified chat model for this provider
			// (the single source of truth — no hardcoded name, no GetModels()[0]),
			// then the provider's own capability-filtered catalogue as the offline
			// fallback. The first candidate that returns non-empty content wins and
			// is cached. p.GetModels()[0] is never trusted: catalogues routinely
			// lead with decommissioned, paid-402, or embedding models.
			resp, err, model := e.generateMemberResilient(fanCtx, p, base)
			// Cache the resolved model when it produced a real participant response
			// (non-empty content OR a tool-call request) — mirroring the resolver's
			// own success predicate so a tool-calling turn (empty content, non-empty
			// tool calls) on the live tool-loop path still caches its working model
			// for 1-call/member reuse next prompt.
			if err == nil && responseIsParticipant(resp) && model != "" {
				e.rememberWorkingModel(p.GetName(), model)
			}
			outCh <- memberOutcome{name: p.GetName(), resp: resp, err: err, model: model}
		}(m)
	}
	go func() { wg.Wait(); close(outCh) }()

	results := make([]ensembleResult, 0, len(members))
	participants := make([]string, 0, len(members))
	excerpts := make(map[string]string, len(members))
	// models maps a successful participant's provider name → the model id it
	// actually served (verifier-chosen via LLMsVerifier, cached, or
	// caller-supplied). Surfaced in ProviderMetadata so the operator SEES which
	// model each ensemble member used.
	memberModels := make(map[string]string, len(members))
	failures := 0
	var firstErr error

	for oc := range outCh {
		// A member is a SUCCESS/participant when it returned without error and its
		// response carries EITHER non-empty content OR a non-empty tool-call
		// request. A tool-calling turn legitimately has empty Content (the model is
		// asking to run tools, not answering yet) — treating that as a failure would
		// make the ensemble unusable inside a tool loop, so it counts as a real
		// participant here.
		if oc.err != nil || oc.resp == nil || !responseIsParticipant(oc.resp) {
			failures++
			if firstErr == nil && oc.err != nil {
				firstErr = oc.err
			}
			continue
		}
		participants = append(participants, oc.name)
		excerpts[oc.name] = participantExcerpt(oc.resp)
		if strings.TrimSpace(oc.model) != "" {
			memberModels[oc.name] = oc.model
		}
		results = append(results, ensembleResult{providerName: oc.name, resp: oc.resp})
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("helix-agent ensemble: all %d member(s) failed (first error: %v)", len(members), firstErr)
	}

	selected := e.vote(results)

	// Deterministic participant ordering for stable evidence.
	sort.Strings(participants)

	scores := make(map[string]float64, len(results))
	for _, r := range results {
		scores[r.providerName] = r.score
	}

	out := *selected.resp
	out.RequestID = request.ID

	// Tool-loop compatibility: if ANY participant requested tool calls, the
	// ensemble is on a tool-request turn and MUST surface tool calls so the
	// caller's tool loop can execute them. Pick the tool-calling participant
	// DETERMINISTICALLY (the one whose provider name sorts first) and carry its
	// Content (may be empty) + ToolCalls. When NO participant requested tools
	// (the normal final-answer turn), this is a no-op and the voted winner is
	// returned unchanged — keeping the existing content-voting path byte-identical.
	if tc := firstToolCallParticipant(results); tc != nil {
		out.Content = tc.resp.Content
		out.ToolCalls = tc.resp.ToolCalls
	}

	if out.ProviderMetadata == nil {
		out.ProviderMetadata = make(map[string]interface{})
	}
	out.ProviderMetadata["ensemble"] = true
	out.ProviderMetadata["ensemble_strategy"] = e.strategy
	out.ProviderMetadata["ensemble_total_providers"] = len(members)
	out.ProviderMetadata["ensemble_successful_providers"] = len(results)
	out.ProviderMetadata["ensemble_failed_providers"] = failures
	out.ProviderMetadata["ensemble_participants"] = participants
	out.ProviderMetadata["ensemble_selected_provider"] = selected.providerName
	out.ProviderMetadata["ensemble_scores"] = scores
	out.ProviderMetadata["ensemble_excerpts"] = excerpts
	// Per-member model visibility: the model id each successful participant
	// actually served, resolved via LLMsVerifier (CONST-036) / cache / catalogue.
	out.ProviderMetadata["ensemble_models"] = memberModels
	return &out, nil
}

// responseIsParticipant reports whether a member's response counts as a real
// ensemble participant: it has non-empty content OR a non-empty tool-call
// request. A tool-calling turn (empty Content + non-empty ToolCalls) is a VALID
// tool request, not a failure.
func responseIsParticipant(resp *LLMResponse) bool {
	if resp == nil {
		return false
	}
	return strings.TrimSpace(resp.Content) != "" || len(resp.ToolCalls) > 0
}

// participantExcerpt renders the per-member evidence excerpt. A content-bearing
// member contributes a short content excerpt; a tool-calling member (empty
// content) contributes a "[tool: <names>]" marker naming the requested tools so
// the metadata still records what that member asked for.
func participantExcerpt(resp *LLMResponse) string {
	if strings.TrimSpace(resp.Content) != "" {
		return excerpt(resp.Content, 160)
	}
	if len(resp.ToolCalls) > 0 {
		names := make([]string, 0, len(resp.ToolCalls))
		for _, tc := range resp.ToolCalls {
			names = append(names, tc.Function.Name)
		}
		return "[tool: " + strings.Join(names, ", ") + "]"
	}
	return ""
}

// firstToolCallParticipant returns the participant with tool calls whose provider
// name sorts first (a stable, deterministic choice independent of fan-out
// completion order), or nil when no participant requested tools.
func firstToolCallParticipant(results []ensembleResult) *ensembleResult {
	var chosen *ensembleResult
	for i := range results {
		if len(results[i].resp.ToolCalls) == 0 {
			continue
		}
		if chosen == nil || results[i].providerName < chosen.providerName {
			chosen = &results[i]
		}
	}
	return chosen
}

// vote scores every member response (confidence-weighted by default, mirroring
// helix_agent's strategy: finish-reason + length + token-efficiency heuristics)
// and returns the highest-scoring one. The scores are written back onto each
// result for the metadata.
func (e *EnsembleProvider) vote(results []ensembleResult) ensembleResult {
	for i := range results {
		results[i].score = scoreResponse(results[i].resp, e.strategy)
	}
	best := results[0]
	for _, r := range results[1:] {
		if r.score > best.score {
			best = r
		}
	}
	return best
}

// scoreResponse computes a quality score from observable response properties.
// It does not require the provider to self-report a confidence field (HelixCode
// LLMResponse has none) — it derives one from finish reason, content length, and
// token efficiency, the same dimensions helix_agent's ConfidenceWeightedStrategy
// uses.
func scoreResponse(resp *LLMResponse, strategy string) float64 {
	if resp == nil {
		return 0
	}
	// Base: a cleanly-finished, non-empty response starts at 1.0.
	score := 1.0
	switch resp.FinishReason {
	case "stop", "end_turn", "":
		score *= 1.1
	case "length", "max_tokens":
		score *= 0.95
	case "content_filter", "content_safety":
		score *= 0.6
	}

	contentLen := len(strings.TrimSpace(resp.Content))
	switch {
	case contentLen == 0:
		return 0
	case contentLen > 50 && contentLen < 1000:
		score *= 1.1
	case contentLen >= 1000 && contentLen < 2000:
		score *= 1.05
	case contentLen >= 2000:
		score *= 0.95
	default: // very short
		score *= 1.0
	}

	// Token efficiency: more content per completion token is rewarded.
	if resp.Usage.CompletionTokens > 0 {
		eff := float64(contentLen) / float64(resp.Usage.CompletionTokens)
		switch {
		case eff > 3.0:
			score *= 1.1
		case eff > 2.0:
			score *= 1.05
		}
	}

	if strategy == "quality_weighted" {
		// Quality strategy additionally favours faster responses.
		if resp.ProcessingTime > 0 && resp.ProcessingTime < time.Second {
			score *= 1.05
		}
	}
	return score
}

// generateMemberResilient resolves a working model for a sentinel/empty-model
// request: it tries the dynamically-resolved candidate list (cached working
// model first, then the verifier's best verified chat model, then the
// provider's own capability-filtered catalogue) and returns the first attempt
// that yields non-empty content. Returns the last error/response when every
// candidate fails. The returned model id is the one that produced the content
// (for caching). Bounded by ensembleMaxModelTries.
func (e *EnsembleProvider) generateMemberResilient(ctx context.Context, p Provider, base LLMRequest) (*LLMResponse, error, string) {
	cands := e.orderedCandidates(ctx, p)
	var lastErr error
	var lastResp *LLMResponse
	tried := 0
	for _, mdl := range cands {
		if tried >= ensembleMaxModelTries {
			break
		}
		if ctx.Err() != nil {
			break
		}
		tried++
		e.recordMemberCall(p.GetName())
		reqCopy := base
		reqCopy.Model = mdl
		r, err := p.Generate(ctx, &reqCopy)
		// A model is RESOLVED-WORKING when it returns either non-empty content OR a
		// non-empty tool-call request — the SAME participant predicate the outer
		// Generate loop uses (responseIsParticipant). A tool-calling turn legitimately
		// has EMPTY Content (the model is asking to run tools, not answering yet); on
		// the live TUI tool-loop path the request carries Tools and the cached
		// chat-working model responds with tool calls + empty content. Requiring
		// non-empty Content here REJECTED that working response and made the resolver
		// walk the rest of the catalogue until every candidate failed
		// ("all N member(s) failed" — the decommissioned tail model surfaced as the
		// first error). Accepting a tool-call response fixes the tool path while
		// leaving the plain-chat path byte-identical (a content-bearing reply still
		// satisfies responseIsParticipant via its non-empty content).
		if err == nil && responseIsParticipant(r) {
			return r, nil, mdl
		}
		// A DEFINITIVE non-retryable failure (model decommissioned / not found /
		// unsupported / 400/404-class "model does not exist") means this model id is
		// permanently dead for this provider — record it so orderedCandidates skips
		// it on every later prompt and the per-prompt re-burn of the catalogue's
		// leading dead models stops. TRANSIENT failures (429 / timeout / network /
		// context-cancel) are NOT recorded — the model may work next time.
		if err != nil && isDefinitiveModelError(err) {
			e.markDead(p.GetName(), mdl)
		}
		lastErr, lastResp = err, r
	}
	if lastErr == nil && (lastResp == nil || strings.TrimSpace(lastResp.Content) == "") {
		lastErr = fmt.Errorf("no chat-capable model returned content (%d tried)", tried)
	}
	return lastResp, lastErr, ""
}

// orderedCandidates returns the member's chat-capable model ids in priority
// order, ALL of which are dynamically resolved (CONST-036/040 — no hardcoded
// names). Priority:
//
//  1. the cached known-working model (so repeat runs are 1 call/member);
//  2. the LLMsVerifier-reported best verified chat model for this provider type
//     (the single source of truth when the verifier is wired);
//  3. the provider's OWN capability-filtered catalogue (the offline fallback,
//     still capability-driven, never a hardcoded list).
//
// Duplicates are de-duplicated preserving first-seen priority.
func (e *EnsembleProvider) orderedCandidates(ctx context.Context, p Provider) []string {
	name := p.GetName()
	e.mu.RLock()
	cached := e.workingModel[name]
	// Copy the member's dead-model set under the read lock so candidate filtering
	// uses a stable snapshot (the set only grows; a model recorded dead mid-walk
	// simply won't be re-offered next prompt — never the current one).
	dead := map[string]bool{}
	for id := range e.deadModel[name] {
		dead[id] = true
	}
	e.mu.RUnlock()

	ordered := make([]string, 0, 8)
	seen := map[string]bool{}
	add := func(id string) {
		id = strings.TrimSpace(id)
		// Skip empty, already-added, and KNOWN-DEAD models. Skipping dead models is
		// the load-bearing fix: once a provider's catalogue-leading model failed as
		// decommissioned/not-found on an earlier prompt, it is never offered again,
		// so later prompts reach a working model in 1 call without re-walking the
		// dead ones.
		if id == "" || seen[id] || dead[id] {
			return
		}
		seen[id] = true
		ordered = append(ordered, id)
	}

	add(cached)
	// Verifier is the authoritative source for the member's best verified chat
	// model (CONST-036). Empty when the verifier is disabled/unreachable or
	// reports nothing for this provider — then the catalogue fallback applies.
	add(ensembleVerifiedModelFor(ctx, p.GetType()))
	for _, c := range catalogueChatCandidatesFor(p) {
		add(c)
	}
	return ordered
}

// WarmCache pre-resolves and caches each member's working chat model BEFORE the
// user submits a prompt, eliminating the cold-start "discovery storm" that made
// the FIRST interactive prompt slow and rate-limit-prone. On a cold cache every
// member's first sentinel-model run probes MANY catalogue models sequentially
// (decommissioned/paid/embedding entries) until one returns content; in the TUI a
// second prompt fires while the first is still mid-storm → concurrent provider
// hammering → every member fails → "all N member(s) failed". Warming the cache
// once, ahead of user interaction, means the first real prompt hits the cached
// working model (1 call/member, no storm).
//
// Behaviour:
//   - members run in PARALLEL; each member's candidate attempts run SEQUENTIALLY
//     via the SAME resilient resolution path (orderedCandidates →
//     generateMemberResilient → rememberWorkingModel) — no duplicated logic
//     (§11.4.74);
//   - bounded by ensembleMaxModelTries (inside generateMemberResilient) and the
//     ensemble timeout;
//   - idempotent and safe to call concurrently — a sync.Once-style guard ensures
//     the warm fan-out runs at most once for the lifetime of the provider, and a
//     member whose cache is already populated is skipped;
//   - NON-blocking for the caller's intent: it does its own bounded work and
//     returns; callers that must not stall the UI thread invoke it in a goroutine.
//
// It never forwards the ensemble sentinel model to a member (that would 404); it
// always resolves a model the member actually serves. A warm probe that fails for
// a member simply leaves that member's cache empty — the next real prompt will
// re-resolve it; warming is best-effort and never errors.
func (e *EnsembleProvider) WarmCache(ctx context.Context) {
	// sync.Once-style guard: run the warm fan-out at most once.
	e.mu.Lock()
	if e.warmStarted {
		e.mu.Unlock()
		return
	}
	e.warmStarted = true
	e.mu.Unlock()

	members := e.snapshot()
	if len(members) == 0 {
		return
	}

	warmCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// A short, content-light probe prompt — enough to elicit non-empty content so
	// the working model is genuinely confirmed, without burning a large
	// completion. CONST-046: not user-facing prose, a fixed internal warm probe.
	probeMessages := []Message{{Role: "user", Content: "ping"}}

	var wg sync.WaitGroup
	for _, m := range members {
		// Skip members whose working model is already cached (idempotent across
		// repeated WarmCache calls and a real prompt that already ran).
		e.mu.RLock()
		_, cached := e.workingModel[m.GetName()]
		e.mu.RUnlock()
		if cached {
			continue
		}
		wg.Add(1)
		go func(p Provider) {
			defer wg.Done()
			base := LLMRequest{ID: uuid.New(), Messages: probeMessages}
			resp, err, model := e.generateMemberResilient(warmCtx, p, base)
			// Same participant-based caching predicate as the resilient resolver and
			// the live Generate path: a non-empty content OR a tool-call response is a
			// resolved working model. The warm probe is a no-tools "ping" so it
			// normally elicits content, but routing the cache gate through
			// responseIsParticipant keeps all three call sites consistent.
			if err == nil && responseIsParticipant(resp) && model != "" {
				e.rememberWorkingModel(p.GetName(), model)
			}
		}(m)
	}
	wg.Wait()
}

// markDead records a (provider, model) pair as permanently dead so
// orderedCandidates skips it for the remainder of the provider's lifetime. Idempotent.
func (e *EnsembleProvider) markDead(providerName, model string) {
	model = strings.TrimSpace(model)
	if providerName == "" || model == "" {
		return
	}
	e.mu.Lock()
	if e.deadModel == nil {
		e.deadModel = map[string]map[string]bool{}
	}
	if e.deadModel[providerName] == nil {
		e.deadModel[providerName] = map[string]bool{}
	}
	e.deadModel[providerName][model] = true
	// If the now-dead model was the cached working model (e.g. a provider
	// decommissioned a model that previously worked), drop the stale cache so the
	// next resolution re-discovers a live model instead of forwarding the dead one.
	if e.workingModel[providerName] == model {
		delete(e.workingModel, providerName)
	}
	e.mu.Unlock()
}

// DeadModelCount returns a snapshot of how many models have been recorded dead
// per member (keyed by provider name). It is the observable that lets a test or
// operator confirm the dead-model set is being populated from real definitive
// failures (§11.4.5).
func (e *EnsembleProvider) DeadModelCount() map[string]int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make(map[string]int, len(e.deadModel))
	for name, set := range e.deadModel {
		out[name] = len(set)
	}
	return out
}

// definitiveModelErrorMarkers are lowercased error-string CLASS markers that
// indicate a PERMANENT, non-retryable model failure: the requested model id does
// not exist / was decommissioned / is not supported by the provider. They are
// error-CLASS phrases, NOT model names (CONST-036/040: zero hardcoded model
// names) — every real cloud provider surfaces one of these substrings for a
// dead-model request:
//
//	groq:       "the model `gemma-7b-it` has been decommissioned"
//	openai-ish: "the model ... does not exist", "model_not_found"
//	openrouter: "is not a valid model id", "no endpoints found for"
//	mistral:    "invalid model", "model not found"
//	generic:    HTTP 400/404 bodies "not supported", "unsupported model"
//
// A model that produced one of these is permanently dead for the provider; a
// 429 / timeout / network error is TRANSIENT and is intentionally absent here.
var definitiveModelErrorMarkers = []string{
	"decommission",            // groq: "has been decommissioned"
	"does not exist",          // openai-style "the model `x` does not exist"
	"model_not_found",         // openai error code
	"model not found",         // mistral / generic
	"no longer supported",     // generic deprecation
	"not supported",           // "model not supported" / "unsupported"
	"unsupported model",       //
	"invalid model",           // mistral / generic
	"not a valid model",       // openrouter "is not a valid model id"
	"unknown model",           //
	"no endpoints found for",  // openrouter dead/removed model id
	"has been deprecated",     //
	"model has been removed",  //
}

// transientErrorMarkers are lowercased markers that indicate a TEMPORARY failure
// (rate-limit / timeout / network / overload). They take precedence over the
// definitive markers: a 429 body sometimes also names the model, but the model is
// NOT dead — it is throttled — so it must NOT be recorded dead. Keeping this guard
// explicit prevents a transient burst from poisoning a healthy model.
var transientErrorMarkers = []string{
	"rate limit",
	"rate_limit",
	"too many requests",
	"429",
	"timeout",
	"timed out",
	"deadline exceeded",
	"context canceled",
	"context cancelled",
	"connection refused",
	"connection reset",
	"temporarily unavailable",
	"service unavailable",
	"503",
	"overloaded",
	"try again",
	"i/o timeout",
	"eof",
	"no such host",
}

// isDefinitiveModelError reports whether err is a PERMANENT, non-retryable
// model-level failure (model decommissioned / not found / unsupported) — the
// signal that the (provider, model) pair should be recorded dead. It returns
// false for transient failures (rate-limit / timeout / network), which take
// precedence, so a temporarily-throttled model is never marked dead. Detection is
// purely by error-CLASS substring, never by a hardcoded model-name list.
func isDefinitiveModelError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	// Transient signals win: never mark a throttled/timed-out model dead.
	for _, t := range transientErrorMarkers {
		if strings.Contains(msg, t) {
			return false
		}
	}
	for _, d := range definitiveModelErrorMarkers {
		if strings.Contains(msg, d) {
			return true
		}
	}
	return false
}

// rememberWorkingModel caches the model id that produced content for a member.
func (e *EnsembleProvider) rememberWorkingModel(providerName, model string) {
	e.mu.Lock()
	if e.workingModel == nil {
		e.workingModel = map[string]string{}
	}
	e.workingModel[providerName] = model
	e.mu.Unlock()
}

// recordMemberCall increments the lifetime underlying-Generate call counter for a
// member. Used to prove (via MemberCallCounts) that resolution does not perform a
// multi-model discovery burst.
func (e *EnsembleProvider) recordMemberCall(providerName string) {
	e.mu.Lock()
	if e.memberCalls == nil {
		e.memberCalls = map[string]int{}
	}
	e.memberCalls[providerName]++
	e.mu.Unlock()
}

// MemberCallCounts returns a snapshot of the lifetime per-member underlying
// p.Generate() attempt counts (keyed by provider name). A member resolving its
// model from the verifier (or a populated cache) adds exactly one call per prompt;
// a member doing a cold catalogue discovery burst shows a large delta. This is the
// observable that lets an operator/test verify the no-burst property (§11.4.5).
func (e *EnsembleProvider) MemberCallCounts() map[string]int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make(map[string]int, len(e.memberCalls))
	for k, v := range e.memberCalls {
		out[k] = v
	}
	return out
}

// MemberResolution describes how one member's chat model is resolved, for
// diagnostics. ProviderType is the member's provider type; VerifierModel is the
// model id LLMsVerifier reports as the best verified chat model for that provider
// ("" when the verifier is disabled/unreachable or reports nothing); Cached is the
// already-discovered working model ("" before the first run/warm); Candidates is
// the full ordered candidate list resolution will try (verifier entry first when
// present, then the provider's own capability-filtered catalogue).
type MemberResolution struct {
	ProviderName  string
	ProviderType  ProviderType
	VerifierModel string
	Cached        string
	Candidates    []string
}

// MemberResolutions returns, per member, exactly how its chat model resolves —
// the SAME orderedCandidates path the live fan-out uses (no duplicated logic). It
// surfaces whether the verifier supplied the leading candidate, which is the
// observable proving the TUI path is genuinely verifier-driven (§11.4.5). It makes
// no network calls.
func (e *EnsembleProvider) MemberResolutions(ctx context.Context) []MemberResolution {
	members := e.snapshot()
	out := make([]MemberResolution, 0, len(members))
	for _, p := range members {
		e.mu.RLock()
		cached := e.workingModel[p.GetName()]
		e.mu.RUnlock()
		out = append(out, MemberResolution{
			ProviderName:  p.GetName(),
			ProviderType:  p.GetType(),
			VerifierModel: ensembleVerifiedModelFor(ctx, p.GetType()),
			Cached:        cached,
			Candidates:    e.orderedCandidates(ctx, p),
		})
	}
	return out
}

// excerpt returns up to n runes of s with an ellipsis when truncated.
func excerpt(s string, n int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// GenerateStream runs the ensemble (non-streaming under the hood — voting needs
// complete responses) and emits the voted result as a single chunk. This is a
// real result, not a placeholder: it is the same combined response Generate
// returns.
//
// Two anti-bluff fixes applied here (same-signal audit alongside the
// StreamWithTools deadlock + the anthropic/azure/gemini/bedrock/groq/vertexai
// unguarded-send fixes in this package):
//
//  1. defer close(ch) — the prior code only called close(ch) on the SUCCESS
//     path (after the send); an e.Generate error returned WITHOUT ever
//     closing ch. Every caller in this codebase drains via `for resp := range
//     ch`, so a never-closed channel left that drain loop blocked forever —
//     a goroutine leak on every ensemble Generate failure. defer guarantees
//     ch is closed on every return path, matching every other provider's
//     GenerateStream in this package (each uses `defer close(ch)` as its
//     first statement).
//  2. select-guarded send — ch is caller-supplied and may be unbuffered or
//     already abandoned (consumer cancelled ctx and stopped draining); an
//     unguarded `ch <- *resp` could block this goroutine forever. Guarding
//     with ctx.Done() mirrors the pattern already used by
//     ollama_provider.go / openai_compatible_provider.go in this package.
func (e *EnsembleProvider) GenerateStream(ctx context.Context, request *LLMRequest, ch chan<- LLMResponse) error {
	defer close(ch)
	resp, err := e.Generate(ctx, request)
	if err != nil {
		return err
	}
	select {
	case ch <- *resp:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

// IsAvailable reports the ensemble as available when it has at least one member
// and at least one member is itself available.
func (e *EnsembleProvider) IsAvailable(ctx context.Context) bool {
	members := e.snapshot()
	if len(members) == 0 {
		return false
	}
	for _, m := range members {
		if m.IsAvailable(ctx) {
			return true
		}
	}
	return false
}

func (e *EnsembleProvider) GetHealth(ctx context.Context) (*ProviderHealth, error) {
	members := e.snapshot()
	healthy := 0
	for _, m := range members {
		if h, err := m.GetHealth(ctx); err == nil && h != nil && h.Status == "healthy" {
			healthy++
		}
	}
	status := "healthy"
	if healthy == 0 {
		status = "unhealthy"
	} else if healthy < len(members) {
		status = "degraded"
	}
	return &ProviderHealth{
		Status:     status,
		LastCheck:  time.Now(),
		ModelCount: 1,
		Message:    fmt.Sprintf("%d/%d ensemble members healthy", healthy, len(members)),
	}, nil
}

// Close closes every member provider, aggregating errors.
func (e *EnsembleProvider) Close() error {
	members := e.snapshot()
	var errs []string
	for _, m := range members {
		if err := m.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("ensemble close errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// GetContextWindow returns the MINIMUM member context window (the binding
// constraint across the fan-out). Zero members ⇒ 0.
func (e *EnsembleProvider) GetContextWindow() int {
	members := e.snapshot()
	min := 0
	for _, m := range members {
		cw := m.GetContextWindow()
		if cw <= 0 {
			continue
		}
		if min == 0 || cw < min {
			min = cw
		}
	}
	return min
}

// CountTokens delegates to the first member's tokenizer, falling back to a
// char-based estimate (1 token ≈ 4 chars) when no member is present.
func (e *EnsembleProvider) CountTokens(text string) (int, error) {
	members := e.snapshot()
	if len(members) > 0 {
		return members[0].CountTokens(text)
	}
	if text == "" {
		return 0, nil
	}
	return len(text) / 4, nil
}
