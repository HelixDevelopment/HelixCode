# HelixCode Repository Analysis - Complete Documentation

This directory contains comprehensive analysis of the HelixCode distributed AI development platform.

## Documents in This Analysis

### 1. [HELIXCODE_COMPREHENSIVE_ANALYSIS.md](./HELIXCODE_COMPREHENSIVE_ANALYSIS.md) (28KB)
**Most detailed reference for architecture and implementation**

Covers:
- Complete project structure and architecture
- All 10 LLM provider implementations with specific file locations and line numbers
- Advanced features breakdown (prompt caching, extended thinking, vision support, MCP protocol)
- Complete configuration system documentation
- Context window and token management strategies
- Task management system design
- Distributed worker pool architecture
- Intelligent model selection algorithm
- Type definitions and data structures
- Recommended features to port to other projects

**Best for**: Deep understanding of the codebase, implementation details, feature research

### 2. [HELIXCODE_QUICK_REFERENCE.md](./HELIXCODE_QUICK_REFERENCE.md) (9KB)
**Fast lookup reference for developers**

Contains:
- Provider file locations table with API endpoints
- Core features location map
- Environment variables checklist
- Architectural patterns overview
- Advanced features matrix
- Build commands reference
- API endpoints list
- Important type definitions
- Model capabilities
- Debugging tips
- Performance considerations

**Best for**: Quick lookups while coding, finding where features are implemented

## Key Sections by Topic

### LLM Providers
**File**: HELIXCODE_COMPREHENSIVE_ANALYSIS.md - Section 2
- Anthropic Claude (extended thinking, prompt caching, vision)
- Google Gemini (2M token context, multimodal)
- OpenAI, Qwen, xAI, OpenRouter, GitHub Copilot
- Local providers (Llama.cpp, Ollama)

### Advanced Features
**File**: HELIXCODE_COMPREHENSIVE_ANALYSIS.md - Section 3 & 7
- Tool/Function calling system (lines 14-53 in tool_provider.go)
- Reasoning engine (4 types: chain-of-thought, tree-of-thoughts, self-reflection, progressive)
- MCP protocol (WebSocket, tool execution, broadcasting)
- Vision support (multi-provider image understanding)
- Streaming (token-by-token delivery)

### Architecture
**File**: HELIXCODE_COMPREHENSIVE_ANALYSIS.md - Section 1 & 6
- Provider abstraction pattern
- Model manager and intelligent selection
- Task distribution system
- Worker pool management
- Configuration system

### Configuration
**File**: HELIXCODE_COMPREHENSIVE_ANALYSIS.md - Section 4
- Server settings
- Database configuration
- LLM provider configuration
- Notification channels (Slack, Telegram, Email, Discord)
- Environment variables

## Project Statistics

- **Language**: Go 1.24.0
- **Module**: dev.helix.code
- **LLM Providers**: 10 (Anthropic, Google, OpenAI, Qwen, xAI, OpenRouter, Copilot, Llama.cpp, Ollama, Local)
- **Total Models Supported**: 100+
- **Core Packages**: 18 internal packages
- **Provider Files**: 10 separate implementations
- **Test Coverage**: Unit, integration, E2E, automation, load tests
- **Platforms**: Linux, macOS, Windows, iOS, Android, Aurora OS, Symphony OS

## Quick Navigation

### To Find Information About...

| Topic | Location |
|-------|----------|
| Anthropic Claude | COMPREHENSIVE - Section 2 (anthropic_provider.go:1-400+) |
| Google Gemini | COMPREHENSIVE - Section 2 (gemini_provider.go:1-400+) |
| Tool Calling | COMPREHENSIVE - Section 3.1 (tool_provider.go:14-404) |
| Reasoning Engine | COMPREHENSIVE - Section 3.2 (reasoning.go:13-332) |
| MCP Protocol | COMPREHENSIVE - Section 3.3 (mcp/server.go:1-383) |
| Task Management | COMPREHENSIVE - Section 3.4 (task/manager.go:1-200+) |
| Worker Pool | COMPREHENSIVE - Section 3.6 (worker/ssh_pool.go:17-300+) |
| Model Selection | COMPREHENSIVE - Section 3.7 (model_manager.go:74-420) |
| Configuration | COMPREHENSIVE - Section 4 (config/config.yaml) |
| Provider Pattern | COMPREHENSIVE - Section 6 (provider.go:112-361) |
| Build Commands | QUICK_REFERENCE - Build Commands |
| Environment Setup | QUICK_REFERENCE - Environment Variables |
| API Endpoints | QUICK_REFERENCE - API Endpoints |
| Type Definitions | QUICK_REFERENCE - Type Definitions |

## Key Code References by File

### Core Infrastructure
- `/internal/llm/provider.go` - Base provider interface (line 112)
- `/internal/llm/model_manager.go` - Model selection algorithm (line 75)
- `/internal/server/server.go` - HTTP server setup
- `/config/config.yaml` - Configuration system

### Provider Implementations
- `anthropic_provider.go` - Extended thinking, prompt caching, vision
- `gemini_provider.go` - 2M token context, multimodal
- `openai_provider.go` - Vision, function calling, reasoning
- `qwen_provider.go` - OAuth2, Chinese models
- `xai_provider.go` - Fast Grok models
- `openrouter_provider.go` - Multi-provider aggregation
- `copilot_provider.go` - GitHub integration
- `local_provider.go` - Llama.cpp integration
- `ollama_provider.go` - Docker-based models
- `llamacpp_provider.go` - Direct C++ integration

### Advanced Features
- `tool_provider.go` - Tool calling framework
- `reasoning.go` - Reasoning engine
- `mcp/server.go` - MCP protocol
- `task/manager.go` - Task management
- `worker/ssh_pool.go` - Worker pool
- `workflow/executor.go` - Workflow engine

## Recommended Learning Path

1. **Start Here**: QUICK_REFERENCE.md - Get overview of file locations and features
2. **Provider Pattern**: COMPREHENSIVE.md Section 6 - Understand how providers work
3. **Core Providers**: COMPREHENSIVE.md Section 2 - Study Anthropic and OpenAI implementations
4. **Advanced Features**: COMPREHENSIVE.md Section 3 - Learn tool calling, reasoning, MCP
5. **Architecture**: COMPREHENSIVE.md Section 1 - Understand overall system design

## Implementation Notes

### For New Projects
See COMPREHENSIVE.md Section 9 - "Recommended Features to Port"

Suggested phased approach:
1. **Phase 1**: Core provider system and manager
2. **Phase 2**: Model selection intelligence
3. **Phase 3**: Tool calling framework
4. **Phase 4**: Advanced features (caching, thinking, vision)
5. **Phase 5**: Task management and workflows

### Performance Optimizations
- Prompt caching reduces costs by 70-90% (Anthropic)
- Streaming for real-time responses
- Hardware-aware model selection
- Worker pool connection reuse
- Task checkpointing every 300 seconds

### Security Best Practices
- API keys in environment variables
- JWT token expiration (24 hours default)
- SSH key-based worker authentication
- MCP tool permission system
- Database password encryption

## Questions & Answers

**Q: How do I add a new LLM provider?**
A: Implement the Provider interface (provider.go:112) and register with ProviderFactory (provider.go:339)

**Q: How does model selection work?**
A: See ModelManager.SelectOptimalModel() in model_manager.go:75 - scores based on 6 factors

**Q: How is prompt caching implemented?**
A: Automatic in anthropic_provider.go:48 - system messages and last message cached with ephemeral cache control

**Q: How does distributed execution work?**
A: SSH worker pool (ssh_pool.go:17) with auto-installation, health monitoring, and capability-based assignment

**Q: What's the MCP protocol flow?**
A: WebSocket connection → Session creation → Tool list → Tool execution → Broadcast notifications

## File Statistics

- HELIXCODE_COMPREHENSIVE_ANALYSIS.md: 28KB, 500+ lines
- HELIXCODE_QUICK_REFERENCE.md: 9KB, 350+ lines
- This README: Reference index

## Contact & Updates

This analysis was generated from the HelixCode repository at:
`/Users/milosvasic/Projects/HelixCode/HelixCode`

For the latest code, refer to the actual source files at the above location.

---

**Last Updated**: November 5, 2025
**Analysis Scope**: Complete codebase examination with line-by-line references
