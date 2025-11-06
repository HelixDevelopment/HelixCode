# Tutorial 3: Using Multiple AI Providers

**Duration**: 20 minutes
**Level**: Beginner-Intermediate

## Overview

Learn to leverage multiple AI providers for optimal results:
- Provider selection based on task
- Automatic fallback
- Cost optimization
- Performance tuning

## Provider Selection Strategy

| Task | Best Provider | Reason |
|------|--------------|--------|
| Complex reasoning | Claude 4 Sonnet | Extended thinking |
| Large codebase | Gemini 2.5 Pro | 2M context |
| Fast iterations | Groq | 500+ tok/s |
| Cost-sensitive | XAI Grok / Gemini Flash | Free / cheap |
| Privacy | Ollama | 100% local |

## Example Workflow

### Step 1: Setup Multiple Providers

```bash
export ANTHROPIC_API_KEY=sk-ant-...
export GEMINI_API_KEY=...
export GROQ_API_KEY=...
ollama serve
```

### Step 2: Architecture Design (Claude)

```bash
helixcode llm provider set anthropic --model claude-4-sonnet

helixcode plan "Design microservices architecture for e-commerce"

# Claude excels at complex reasoning
```

### Step 3: Full Codebase Analysis (Gemini)

```bash
helixcode llm provider set gemini --model gemini-2.5-pro

helixcode analyze --full-context

# Gemini handles 2M tokens for entire codebase
```

### Step 4: Rapid Prototyping (Groq)

```bash
helixcode llm provider set groq --model llama-3.3-70b-versatile

helixcode generate "Create user service"
helixcode generate "Add product catalog"
helixcode generate "Implement shopping cart"

# Groq provides instant feedback
```

### Step 5: Privacy-Sensitive Code (Ollama)

```bash
helixcode llm provider set ollama --model codellama:13b

# Process sensitive internal logic offline
helixcode refactor --file internal/payment/processor.go
```

## Automatic Fallback

```yaml
# config.yaml
llm:
  fallback:
    enabled: true
    providers:
      - anthropic/claude-3-5-sonnet-latest
      - openai/gpt-4o
      - groq/llama-3.3-70b-versatile
```

## Cost Optimization

```bash
# Use free tier for experiments
helixcode llm provider set xai --model grok-3-fast-beta

# Production: Use Claude with caching
helixcode llm provider set anthropic --enable-caching

# Monitor costs
helixcode llm usage --period month
```

## Results

- **Optimal Performance**: Right provider for each task
- **Cost Savings**: 70% reduction with strategic provider selection
- **Reliability**: Automatic fallback ensures uptime

---

Continue to [Tutorial 4: Browser Automation](Tutorial_4_Browser_Automation.md)
