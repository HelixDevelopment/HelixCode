package llm

import (
	"errors"
	"fmt"
	"strings"
)

// provider_factory.go (P1-F12-T07): unified construction + selection for the
// four cloud backends covered by Feature 12 — Anthropic, Bedrock, Vertex AI,
// Azure OpenAI. The pre-existing NewProvider() in factory.go is the
// catch-all factory across every Provider type (cloud + local + OpenAI-
// compatible). NewCloudProvider() narrows scope to the cloud quartet so
// the wizard / selector path can refuse to construct anything outside that
// universe — and so a misconfigured "ollama" string in --provider does not
// silently fall through to a local backend.

// ErrNoProviderConfigured is returned by Select when none of the four sources
// (flag / env / config / wizard-default) yielded a value. Callers running in
// interactive mode should launch the wizard on this error; non-interactive
// callers should surface it to the user with a remediation hint.
var ErrNoProviderConfigured = errors.New(
	"no provider configured: pass --provider, set HELIX_LLM_PROVIDER, " +
		"populate provider in config, or run `helixcode wizard`")

// SelectorInput captures the four sources of provider-type selection in the
// precedence order the Selector applies (flag > env > config > wizard).
// Each field is the raw, untrimmed string the caller obtained from that
// source. Empty strings are treated as "source did not provide a value" and
// the next source is consulted.
type SelectorInput struct {
	// Flag is the CLI flag value (e.g., --provider=bedrock). Highest
	// precedence — explicit user instruction for this invocation.
	Flag string

	// Env is the value of HELIX_LLM_PROVIDER (or whatever the caller has
	// chosen as its env var). Mid precedence — runtime override without
	// rewriting config.
	Env string

	// Config is the value loaded from a persisted config file
	// (e.g., $XDG_CONFIG_HOME/helixcode/config.yaml's provider field).
	// Lowest precedence among "user-provided" sources.
	Config string
}

// Select resolves the cloud ProviderType to use using flag > env > config
// precedence. It returns ErrNoProviderConfigured (sentinel, errors.Is-able)
// when every source is empty — at which point the caller should launch the
// interactive wizard if running interactively. Unknown/unsupported provider
// strings return a non-sentinel error so the caller can distinguish the
// "needs wizard" case from "user typed garbage".
//
// The Selector is pure: no env reads, no file IO, no construction. It is
// intentionally trivial so it can be exercised exhaustively by unit tests.
func Select(input SelectorInput) (ProviderType, error) {
	raw := firstNonEmpty(input.Flag, input.Env, input.Config)
	if raw == "" {
		return "", ErrNoProviderConfigured
	}
	return parseCloudProviderType(raw)
}

// NewCloudProvider constructs the concrete cloud Provider for the given
// resolved ProviderType using cfg. It only handles the four Feature-12
// cloud backends. Local / OpenAI-compatible / hosted-OpenAI providers are
// rejected — callers needing those should use NewProvider() in factory.go.
//
// Returned Provider already implements the full Provider interface; any
// credential / endpoint resolution failures bubble up from the underlying
// New<X>Provider constructor. Per the audits in T03–T06, all four
// constructors defer credential validation where the SDK permits it, so
// this function can succeed in offline / no-creds environments and let
// runtime calls surface real auth errors (anti-bluff: no fake-success
// "available" status when nothing actually works).
func NewCloudProvider(t ProviderType, cfg ProviderConfigEntry) (Provider, error) {
	// Normalise cfg.Type to match t so downstream provider code that reads
	// cfg.Type sees a coherent value even if the caller passed a config
	// loaded from a different source.
	cfg.Type = t

	switch t {
	case ProviderTypeAnthropic:
		return NewAnthropicProvider(cfg)
	case ProviderTypeBedrock:
		return NewBedrockProvider(cfg)
	case ProviderTypeVertexAI:
		return NewVertexAIProvider(cfg)
	case ProviderTypeAzure:
		return NewAzureProvider(cfg)
	case ProviderTypeGroq:
		return NewGroqProvider(cfg)
	case ProviderTypeOpenAI:
		return NewOpenAIProvider(cfg)
	case ProviderTypeGemini:
		return NewGeminiProvider(cfg)
	case ProviderTypeOpenRouter:
		return NewOpenRouterProvider(cfg)
	case ProviderTypeXAI:
		return NewXAIProvider(cfg)
	case ProviderTypeQwen:
		return NewQwenProvider(cfg)
	case ProviderTypeCopilot:
		return NewCopilotProvider(cfg)
	case ProviderTypeMistral:
		return NewMistralProvider(cfg)
	case ProviderTypeDeepSeek:
		return NewDeepSeekProvider(cfg)
	case ProviderTypeOllama:
		return newOllamaFromEntry(cfg)
	case ProviderTypeLlamaCpp:
		return newLlamaCPPFromEntry(cfg)
	case ProviderTypeReplicate:
		return NewReplicateProvider(cfg)
	default:
		return nil, fmt.Errorf(
			"NewCloudProvider: %q is not a cloud provider type (supported: %s)",
			t, supportedCloudProviderList())
	}
}

// firstNonEmpty returns the first non-empty (after trim) string in args.
func firstNonEmpty(args ...string) string {
	for _, a := range args {
		if strings.TrimSpace(a) != "" {
			return a
		}
	}
	return ""
}

// parseCloudProviderType normalises the raw user-provided string into a
// canonical cloud ProviderType. Returns a non-sentinel error for unknown
// values so callers can distinguish "no source" (ErrNoProviderConfigured)
// from "user typed garbage".
//
// Anti-bluff (CONST-035): if the user typed a name that IS a known
// provider type but NOT one of F12's four direct-cloud backends (e.g.
// "groq", "openai", "gemini", "deepseek", "xai", "openrouter",
// "mistral", "ollama", "llamacpp"), surface a directed error that
// names the right path (server-mediated provider manager) rather than
// the generic "unknown cloud provider" message — which previously
// implied those providers aren't supported at all.
func parseCloudProviderType(raw string) (ProviderType, error) {
	norm := strings.ToLower(strings.TrimSpace(raw))
	switch norm {
	case "anthropic":
		return ProviderTypeAnthropic, nil
	case "bedrock", "aws", "aws-bedrock":
		return ProviderTypeBedrock, nil
	case "vertexai", "vertex", "vertex-ai", "gcp", "gcp-vertex":
		return ProviderTypeVertexAI, nil
	case "azure", "azure-openai", "azureopenai":
		return ProviderTypeAzure, nil
	case "groq":
		return ProviderTypeGroq, nil
	case "openai", "open-ai":
		return ProviderTypeOpenAI, nil
	case "gemini", "google":
		return ProviderTypeGemini, nil
	case "openrouter", "open-router":
		return ProviderTypeOpenRouter, nil
	case "xai", "grok":
		return ProviderTypeXAI, nil
	case "qwen":
		return ProviderTypeQwen, nil
	case "copilot", "github-copilot":
		return ProviderTypeCopilot, nil
	case "mistral":
		return ProviderTypeMistral, nil
	case "deepseek":
		return ProviderTypeDeepSeek, nil
	case "ollama":
		return ProviderTypeOllama, nil
	case "llamacpp", "llama-cpp", "llama.cpp":
		return ProviderTypeLlamaCpp, nil
	case "replicate":
		return ProviderTypeReplicate, nil
	case "vllm", "localai", "lmstudio":
		return "", fmt.Errorf(
			"provider %q is supported by HelixCode but not via the F12 direct-cloud-provider CLI path "+
				"(supported direct-cloud backends: %s). "+
				"Configure %q in HelixCode/config/config.yaml under llm.providers: and access it via the HelixCode server "+
				"(see docs/user_manual/ZERO_BLUFF_USER_MANUAL.md §2.4 'LLM Providers (F12)'). "+
				"The full provider list per CONST-039 is in docs/llms_verifier/.",
			raw, supportedCloudProviderList(), raw)
	default:
		return "", fmt.Errorf(
			"unknown provider %q (F12 direct-cloud supports: %s; "+
				"the full HelixCode provider catalogue per CONST-039 is accessed via the server-side provider manager — "+
				"see docs/user_manual/ZERO_BLUFF_USER_MANUAL.md §2.4)",
			raw, supportedCloudProviderList())
	}
}

// supportedCloudProviderList returns a stable, human-readable list of the
// canonical direct-cloud-provider names for error messages. Expanded from
// the original 4 (anthropic/bedrock/vertexai/azure) to also cover Groq +
// OpenAI in round-41-continued, closing the most-common readiness gap
// for users who expect modern-CLI-agent parity (just plug in API key
// and go) with the two most-used cloud providers beyond the original
// four. Other providers still require server-mediated config.yaml setup
// — see ZERO_BLUFF_USER_MANUAL.md §2.4 Path A vs Path B.
func supportedCloudProviderList() string {
	return "anthropic, bedrock, vertexai, azure, groq, openai, gemini, openrouter, xai, qwen, copilot, mistral, deepseek, ollama, llamacpp, replicate"
}

// newOllamaFromEntry adapts the generic ProviderConfigEntry into the
// OllamaConfig that NewOllamaProvider expects. Defaults match the
// constructor's own defaults (BaseURL http://localhost:11434, 30s
// timeout, stream enabled). Endpoint from the entry overrides
// BaseURL. Per CONST-035: the adapter never silently swallows a
// construction error; failures bubble up to the caller.
func newOllamaFromEntry(cfg ProviderConfigEntry) (Provider, error) {
	oc := OllamaConfig{
		BaseURL:       cfg.Endpoint,
		StreamEnabled: true,
	}
	if oc.BaseURL == "" {
		oc.BaseURL = "http://localhost:11434"
	}
	if len(cfg.Models) > 0 {
		oc.DefaultModel = cfg.Models[0]
	}
	return NewOllamaProvider(oc)
}

// newLlamaCPPFromEntry adapts ProviderConfigEntry into LlamaConfig.
// ServerHost comes from cfg.Endpoint when present, otherwise the
// constructor's default-localhost path applies. Model defaults to
// cfg.Models[0] if supplied. Anti-bluff: failures bubble up.
func newLlamaCPPFromEntry(cfg ProviderConfigEntry) (Provider, error) {
	lc := LlamaConfig{
		ServerHost: cfg.Endpoint,
	}
	if len(cfg.Models) > 0 {
		lc.Model = cfg.Models[0]
	}
	return NewLlamaCPPProvider(lc)
}

// ParseCloudProviderType is the exported counterpart of parseCloudProviderType.
// Callers outside this package (e.g., the wizard cobra subcommand) use it to
// normalise user-supplied --provider strings without re-implementing the
// alias table here. Returns the same non-sentinel error on unknown input.
func ParseCloudProviderType(raw string) (ProviderType, error) {
	return parseCloudProviderType(raw)
}
