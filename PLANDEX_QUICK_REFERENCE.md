# Plandex Quick Reference for HelixCode Porting

## Architecture at a Glance

```
┌─────────────────────────────────────────────────────────────┐
│                    Plandex Architecture                      │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  CLI (Go/TypeScript)                                         │
│  ├─ REPL with fuzzy autocomplete                            │
│  ├─ Streaming response handling                             │
│  └─ File system operations                                  │
│                           │                                  │
│                           ▼                                  │
│  REST API (Go - Gorilla Mux)                                │
│  ├─ Port 8099                                               │
│  ├─ WebSocket support                                       │
│  └─ Streaming response endpoints                            │
│                           │                                  │
│        ┌──────────────────┼──────────────────┐              │
│        ▼                  ▼                  ▼               │
│   Database         LiteLLM Proxy      Git Integration       │
│   PostgreSQL       (Python)            (Go)                 │
│   (8000+ lines     Port 4000           Auto-commit          │
│    schema)         OAuth Support       Commit msgs          │
│                    Streaming           Revert support       │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

## Provider Fallback Chain

```
Requested Model
     │
     ▼
┌─────────────────────┐
│  Direct Provider    │  OPENAI_API_KEY, ANTHROPIC_API_KEY, etc.
│  (OpenAI, etc)      │  ✓ Preferred if set
└─────────────────────┘
     │ (if unavailable or fails)
     ▼
┌─────────────────────┐
│   OpenRouter        │  OPENROUTER_API_KEY
│   (Aggregator)      │  ✓ Fallback option
└─────────────────────┘
     │ (if unavailable)
     ▼
┌─────────────────────┐
│  Error/Manual       │  User intervention required
└─────────────────────┘
```

## Core Features Matrix

| Feature | Implementation | Key Files |
|---------|----------------|-----------|
| **Plans** | Conv.-like tasks | `server/db/plan_helpers.go` |
| **Context** | Files + URLs + Images | `server/syntax/file_map/` |
| **Diffs** | Git-based review | `server/diff/diff.go` |
| **Roles** | Specialized models | `shared/ai_models_roles.go` |
| **Autonomy** | 5 levels + flags | `shared/plan_config.go` |
| **Versioning** | Full history | `server/db/` (50+ migrations) |
| **Execution** | Build/test/deploy | `server/model/plan/build_*.go` |
| **Debugging** | Browser + Terminal | Auto-screenshot capture |

## Key Model Roles

```
User Prompt
    │
    ▼ (Planning Phase)
┌───────────────────────────┐
│  Planner/Architect Role   │  Large context, good reasoning
│  + Load Context           │  Selects relevant files
└───────────────────────────┘
    │
    ▼ (Implementation Phase)
┌───────────────────────────┐
│  Coder/Builder Roles      │  Instruction following, precise
│  Smart Context Filtering  │  Only loads files for this step
└───────────────────────────┘
    │
    ▼ (File Application)
┌───────────────────────────┐
│  Whole-File-Builder       │  Fallback if targeted edits fail
│  (Optional)               │
└───────────────────────────┘
```

## Autonomy Levels

```
None         Manual everything (pure sandbox)
    │
    ├─► Basic     Auto-continue + auto-build
    │
    ├─► Plus      Smart context + manual exec + auto-commit
    │
    ├─► Semi      Auto-load context + manual apply (DEFAULT)
    │
    └─► Full      Complete automation with rollback
```

Each level overridable with individual flags:
- `auto-continue` (manual → auto plan continuation)
- `auto-build` (manual → auto file diff building)
- `auto-apply` (manual → auto change application)
- `auto-exec` (manual → auto command execution)
- `auto-debug` (manual → auto error debugging)
- `auto-commit` (manual → auto git commits)

## Supported Providers Summary

| Provider | Type | Auth | Key Features |
|----------|------|------|--------------|
| OpenAI | Direct | API Key | JSON, strict schemas, latest models |
| Anthropic | Direct | API Key | Claude Max subscription support |
| Google AI Studio | Direct | API Key | Gemini, free tier available |
| Google Vertex | Enterprise | Service Account | Bedrock, high volume |
| Azure OpenAI | Cloud | API Key + URL | Enterprise deployment |
| AWS Bedrock | Cloud | IAM/Env | Anthropic via AWS |
| OpenRouter | Aggregator | API Key | 100+ models, failover |
| DeepSeek | Direct | API Key | R1 reasoning models |
| Perplexity | Direct | API Key | Web-aware reasoning |
| Ollama | Local | None | Local inference, free |
| Custom | Self-hosted | Config | OpenAI-compatible endpoints |

## Database Schema (Essentials)

```
Plans                    Contexts
├─ id                    ├─ id
├─ name                  ├─ path (file/url/image)
├─ status                ├─ type
├─ autonomy_config       ├─ content
├─ model_settings        └─ versions
└─ versions
    │
    ├─ Messages         Pending Changes
    │ ├─ role          ├─ file_path
    │ ├─ content       ├─ replacements
    │ └─ streaming_data└─ status

Branches & Git
├─ plan branches
├─ git commits
└─ rewind tracking
```

## File Edit Fallback Chain

```
Proposed Edit
    │
    ▼
Tree-sitter Structured Edit (Targeted)
    │
    ├─ Success → Apply
    │
    └─ Fail
        │
        ▼
    Syntax Validation & Fix
        │
        ├─ Success → Apply
        │
        └─ Fail
            │
            ▼
        Whole-File Builder Role Rewrite
            │
            ├─ Success → Apply
            │
            └─ Fail
                │
                ▼
            Alternative Builder Approach
                │
                ├─ Success → Apply
                │
                └─ Fail → Manual User Fix
```

## Context Management Pipeline

```
1. Discovery
   └─ Tree-sitter project map generation
      (30+ languages, 20M+ token support)

2. Selection
   └─ Architect role selects relevant files
      using project map + prompt

3. Filtering (Smart Context)
   └─ Load only files relevant to current step
      (slides context window up/down)

4. Caching (Optional)
   └─ Apply prompt cache directives for
      Anthropic, OpenAI, Google
```

## Configuration Hierarchy

```
Preset Autonomy Level
├─ none, basic, plus, semi, full
├─ Each has standard flags preset
└─ Individual flags override:
    ├─ auto-continue (true/false)
    ├─ auto-build (true/false)
    ├─ auto-apply (true/false)
    ├─ auto-exec (true/false)
    ├─ auto-debug (true/false)
    ├─ auto-commit (true/false)
    ├─ auto-load-context (true/false)
    ├─ smart-context (true/false)
    └─ auto-update-context (true/false)
```

## Model Pack Overview (16 Built-in)

**General Purpose:**
- DailyDriver (default) - balanced
- Strong - maximum capability
- Cheap - cost optimized
- OSS - open source only

**Provider Specific:**
- AnthropicOnly, OpenAIOnly, GoogleOnly
- OpusPlanner, O3Planner, R1Planner
- GeminiPlanner, PerplexityPlanner

**Local/Experimental:**
- OllamaExperimental
- OllamaAdaptiveOSS
- OllamaAdaptiveDaily (hybrid)

**Specialized:**
- Reasoning (o3-mini based)

## Critical Implementation Files

| Area | Files | LOC |
|------|-------|-----|
| DB Schema | `server/db/*.go` | 8,351 |
| Models Config | `shared/ai_models_*.go` | 120K+ |
| Plan Execution | `server/model/plan/` | ~2K |
| Handlers | `server/handlers/` | ~10K |
| Syntax/Edits | `server/syntax/` | ~80K |
| CLI Commands | `cli/cmd/` | ~15K |

## Credential Discovery (Environment First)

```
For each provider:
1. Check specific env var (OPENAI_API_KEY, etc)
2. For AWS: Check PLANDEX_AWS_PROFILE first
3. For AWS: Fall back to AWS credential chain
4. For Google: Check service account file/JSON string/base64
5. For Anthropic: Support OAuth tokens (sk-ant-oat*)
6. For Azure: Require both API_KEY and API_BASE
```

## Key Differences from Standard ChatGPT/Claude

| Aspect | Plandex | ChatGPT |
|--------|---------|---------|
| Context | Project-aware + file loading | Conversation only |
| Changes | Sandboxed diffs with approval | Direct file writing |
| Versioning | Full plan history + branching | Conversation history only |
| Execution | Auto build/test/deploy with rollback | No command execution |
| Models | Role-based routing | Single model per session |
| Autonomy | 5 configurable levels | None/Full only |
| Context Size | 2M tokens effective | Limited per request |
| Debugging | Browser + terminal snapshots | None |

## Quick Start for HelixCode Integration

### Phase 1: Foundation (Weeks 1-3)
- [ ] Implement LiteLLM proxy abstraction
- [ ] Set up provider fallback chain
- [ ] Create role-based model routing
- [ ] Store model configs in database

### Phase 2: Context & Storage (Weeks 4-6)
- [ ] Build context management pipeline
- [ ] Implement project map generation
- [ ] Create plan version storage schema
- [ ] Build smart context filtering

### Phase 3: Execution & Review (Weeks 7-9)
- [ ] Implement diff generation and review
- [ ] Build change application logic
- [ ] Create autonomy level system
- [ ] Add execution tracking

### Phase 4: Polish & Integration (Weeks 10-12)
- [ ] Browser debugging integration
- [ ] Git auto-commit functionality
- [ ] Plan branching and rewind
- [ ] Full CLI integration

## Environment Variables Quick Reference

```bash
# Providers
OPENAI_API_KEY=sk-...
OPENAI_ORG_ID=org-... (optional)
ANTHROPIC_API_KEY=sk-ant-...
GEMINI_API_KEY=...
AZURE_OPENAI_API_KEY=...
AZURE_API_BASE=...
AZURE_API_VERSION=2025-04-01-preview (optional)
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...
AWS_REGION=...
PLANDEX_AWS_PROFILE=... (for credential file)
DEEPSEEK_API_KEY=...
PERPLEXITY_API_KEY=...
OPENROUTER_API_KEY=...

# Google Vertex
GOOGLE_APPLICATION_CREDENTIALS=/path/or/json/or/base64
VERTEXAI_PROJECT=...
VERTEXAI_LOCATION=...

# Local
OLLAMA_BASE_URL=http://localhost:11434

# Database
DATABASE_URL=postgres://user:pass@host:5432/db

# Server
GOENV=development|production
LOCAL_MODE=1 (for self-hosting)
PLANDEX_BASE_DIR=/path/to/data
```

