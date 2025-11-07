# HelixCode Phase 3 - Integration Guide
## Practical Patterns for AI-Powered Development

**Version:** 1.0
**Date:** November 7, 2025

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [System Initialization](#system-initialization)
3. [Common Patterns](#common-patterns)
4. [Advanced Integration](#advanced-integration)
5. [Best Practices](#best-practices)
6. [Troubleshooting](#troubleshooting)

---

## Quick Start

### Minimal Setup

```go
package main

import (
    "dev.helix.code/internal/session"
    "dev.helix.code/internal/memory"
    "dev.helix.code/internal/persistence"
)

func main() {
    // Create managers
    sessions := session.NewManager()
    memory := memory.NewManager()
    store, _ := persistence.NewStore("./data")

    // Configure persistence
    store.SetSessionManager(sessions)
    store.SetMemoryManager(memory)

    // Load previous state
    store.LoadAll()

    // Your application logic...

    // Save on exit
    defer store.SaveAll()
}
```

---

## System Initialization

### Complete Initialization Pattern

```go
package app

import (
    "log"
    "time"

    "dev.helix.code/internal/session"
    "dev.helix.code/internal/memory"
    "dev.helix.code/internal/focus"
    "dev.helix.code/internal/context/builder"
    "dev.helix.code/internal/persistence"
    "dev.helix.code/internal/template"
)

// HelixSystem encapsulates all Phase 3 components
type HelixSystem struct {
    Sessions    *session.Manager
    Memory      *memory.Manager
    Focus       *focus.Manager
    Templates   *template.Manager
    Context     *builder.Builder
    Persistence *persistence.Store
}

// NewHelixSystem creates and initializes a complete system
func NewHelixSystem(dataPath string) (*HelixSystem, error) {
    system := &HelixSystem{
        Sessions:  session.NewManager(),
        Memory:    memory.NewManager(),
        Focus:     focus.NewManager(),
        Templates: template.NewManager(),
        Context:   builder.NewBuilder(),
    }

    // Initialize persistence
    store, err := persistence.NewStore(dataPath)
    if err != nil {
        return nil, err
    }
    system.Persistence = store

    // Configure persistence
    store.SetSessionManager(system.Sessions)
    store.SetMemoryManager(system.Memory)
    store.SetFocusManager(system.Focus)
    store.EnableAutoSave(5 * time.Minute)

    // Configure context builder
    system.Context.SetSessionManager(system.Sessions)
    system.Context.SetFocusManager(system.Focus)
    system.Context.RegisterDefaultTemplates()

    // Register built-in templates
    if err := system.Templates.RegisterBuiltinTemplates(); err != nil {
        return nil, err
    }

    // Set up callbacks for logging/monitoring
    system.setupCallbacks()

    // Load previous state
    if err := store.LoadAll(); err != nil {
        log.Printf("Warning: Failed to load previous state: %v", err)
    }

    return system, nil
}

// setupCallbacks configures event callbacks
func (s *HelixSystem) setupCallbacks() {
    // Log session events
    s.Sessions.OnCreate(func(sess *session.Session) {
        log.Printf("Session created: %s (%s)", sess.Name, sess.Mode)
    })

    s.Sessions.OnComplete(func(sess *session.Session) {
        log.Printf("Session completed: %s (duration: %v)", sess.Name, sess.Duration)
    })

    // Log persistence events
    s.Persistence.OnSave(func(metadata *persistence.SaveMetadata) {
        log.Printf("State saved: %d items (%d bytes)", metadata.Items, metadata.Size)
    })

    s.Persistence.OnError(func(err error) {
        log.Printf("Persistence error: %v", err)
    })
}

// Shutdown gracefully shuts down the system
func (s *HelixSystem) Shutdown() error {
    // Disable auto-save
    s.Persistence.DisableAutoSave()

    // Save final state
    if err := s.Persistence.SaveAll(); err != nil {
        return err
    }

    log.Println("System shutdown complete")
    return nil
}
```

---

## Common Patterns

### Pattern 1: AI-Assisted Feature Development

**Scenario:** Developer implements a new feature with AI assistance.

```go
func ImplementFeature(sys *HelixSystem, projectID, featureName, description string) error {
    // 1. Create development session
    sess, err := sys.Sessions.Create(projectID, featureName, description, session.ModeBuilding)
    if err != nil {
        return err
    }
    sys.Sessions.SetActive(sess.ID)

    // 2. Create conversation for this feature
    conv, err := sys.Memory.CreateConversation("Feature: " + featureName)
    if err != nil {
        return err
    }
    sys.Memory.SetActive(conv.ID)

    // 3. Create focus chain to track files
    chain, err := sys.Focus.CreateChain(featureName, true)
    if err != nil {
        return err
    }

    // 4. Developer asks AI for help
    sys.Memory.AddMessageToActive(
        memory.NewUserMessage("How should I structure the code for " + featureName + "?"))

    // 5. Build context for AI
    context, err := sys.Context.BuildWithTemplate("coding")
    if err != nil {
        return err
    }

    // 6. Get AI response (simulated)
    aiResponse := callAI(context)

    // 7. Store AI response
    sys.Memory.AddMessageToActive(memory.NewAssistantMessage(aiResponse))

    // 8. As developer works, track files
    sys.trackFile(chain, "src/features/new_feature.go")
    sys.trackFile(chain, "src/features/new_feature_test.go")

    // 9. Ask AI to generate code
    codePrompt := sys.generateCodePrompt(featureName)
    codeContext, _ := sys.Context.BuildWithTemplate("coding")
    generatedCode := callAI(codeContext, codePrompt)

    // 10. Store conversation
    sys.Memory.AddMessageToActive(memory.NewUserMessage("Generate implementation"))
    sys.Memory.AddMessageToActive(memory.NewAssistantMessage(generatedCode))

    // 11. Complete session when done
    sys.Sessions.Complete(sess.ID)

    return nil
}

func (s *HelixSystem) trackFile(chain *focus.Chain, filepath string) {
    f := focus.NewFocus(focus.FocusTypeFile, filepath)
    chain.Push(f)
}

func (s *HelixSystem) generateCodePrompt(featureName string) string {
    prompt, _ := s.Templates.RenderByName("Function", map[string]interface{}{
        "function_name": featureName,
        "parameters":    "ctx context.Context, req *Request",
        "return_type":   "(*Response, error)",
        "body":          "// TODO: Implement",
    })
    return prompt
}
```

### Pattern 2: Debugging with AI

**Scenario:** Developer encounters a bug and uses AI to help debug.

```go
func DebugWithAI(sys *HelixSystem, errorMsg, filepath string, lineNum int) error {
    // 1. Create debugging session
    sess, _ := sys.Sessions.Create("project-1", "Debug: "+errorMsg,
        "Fix error at "+filepath, session.ModeDebugging)
    sys.Sessions.SetActive(sess.ID)

    // 2. Create debugging conversation
    conv, _ := sys.Memory.CreateConversation("Debug: " + errorMsg)
    sys.Memory.SetActive(conv.ID)

    // 3. Add error context
    sys.Memory.AddMessageToActive(memory.NewUserMessage(fmt.Sprintf(
        "Error: %s at %s:%d", errorMsg, filepath, lineNum)))

    // 4. Build debugging prompt from template
    code := readFile(filepath)
    prompt, _ := sys.Templates.RenderByName("Bug Fix", map[string]interface{}{
        "language":          "Go",
        "error_message":     errorMsg,
        "code":              code,
        "expected_behavior": "No error",
        "actual_behavior":   errorMsg,
    })

    // 5. Build context with error history
    errorSource := builder.NewErrorSource()
    errorSource.AddError(errorMsg, filepath, lineNum, time.Now().Format(time.RFC3339))
    sys.Context.RegisterSource(errorSource)

    context, _ := sys.Context.BuildWithTemplate("debugging")

    // 6. Get AI analysis
    analysis := callAI(context, prompt)

    // 7. Store in conversation
    sys.Memory.AddMessageToActive(memory.NewAssistantMessage(analysis))

    // 8. Track file in focus
    chain, _ := sys.Focus.GetActiveChain()
    if chain == nil {
        chain, _ = sys.Focus.CreateChain("Debug Session", true)
    }
    f := focus.NewFocus(focus.FocusTypeFile, filepath)
    f.SetMetadata("error", errorMsg)
    f.SetMetadata("line", fmt.Sprintf("%d", lineNum))
    chain.Push(f)

    return nil
}
```

### Pattern 3: Code Review with Templates

**Scenario:** Automated code review using templates and AI.

```go
func ReviewCode(sys *HelixSystem, prNumber int, changedFiles []string) (string, error) {
    // 1. Create review session
    sess, _ := sys.Sessions.Create("project-1", fmt.Sprintf("Review PR #%d", prNumber),
        "Code review", session.ModePlanning)
    sys.Sessions.SetActive(sess.ID)

    // 2. Build context with changed files
    for _, file := range changedFiles {
        content := readFile(file)
        fileSource := builder.NewFileSource(file, content, builder.PriorityHigh)
        sys.Context.RegisterSource(fileSource)
    }

    // 3. Generate review prompt for each file
    var reviews []string
    for _, file := range changedFiles {
        content := readFile(file)

        prompt, _ := sys.Templates.RenderByName("Code Review", map[string]interface{}{
            "language":    detectLanguage(file),
            "code":        content,
            "focus_areas": "security, performance, best practices",
        })

        context, _ := sys.Context.BuildWithTemplate("review")
        review := callAI(context, prompt)
        reviews = append(reviews, fmt.Sprintf("## %s\n\n%s", file, review))
    }

    // 4. Store in conversation
    conv, _ := sys.Memory.CreateConversation(fmt.Sprintf("Review PR #%d", prNumber))
    sys.Memory.AddMessage(conv.ID, memory.NewUserMessage("Review these changes"))
    sys.Memory.AddMessage(conv.ID, memory.NewAssistantMessage(strings.Join(reviews, "\n\n")))

    // 5. Complete session
    sys.Sessions.Complete(sess.ID)

    return strings.Join(reviews, "\n\n"), nil
}
```

### Pattern 4: Interactive Development Session

**Scenario:** Developer has ongoing conversation with AI while developing.

```go
func InteractiveDevelopment(sys *HelixSystem) {
    // 1. Create long-running session
    sess, _ := sys.Sessions.Create("project-1", "Interactive Dev",
        "Build feature with AI assistance", session.ModeBuilding)
    sys.Sessions.SetActive(sess.ID)

    // 2. Create conversation
    conv, _ := sys.Memory.CreateConversation("Interactive Development")
    sys.Memory.SetActive(conv.ID)

    // 3. Create focus chain
    chain, _ := sys.Focus.CreateChain("Development", true)

    // 4. Development loop
    for {
        // User input
        userInput := getUserInput()
        if userInput == "exit" {
            break
        }

        // Track if they mention a file
        if filepath := extractFilePath(userInput); filepath != "" {
            f := focus.NewFocus(focus.FocusTypeFile, filepath)
            chain.Push(f)
        }

        // Add to conversation
        sys.Memory.AddMessageToActive(memory.NewUserMessage(userInput))

        // Build context with recent conversation and focus
        sys.Context.Clear()
        sessionSource := builder.NewSessionSource(sys.Sessions)
        focusSource := builder.NewFocusSource(sys.Focus, 10)
        sys.Context.RegisterSource(sessionSource)
        sys.Context.RegisterSource(focusSource)

        context, _ := sys.Context.BuildWithTemplate("coding")

        // Get AI response
        aiResponse := callAI(context, userInput)

        // Store response
        sys.Memory.AddMessageToActive(memory.NewAssistantMessage(aiResponse))

        // Display to user
        displayResponse(aiResponse)

        // Check if AI generated code
        if code := extractCodeBlocks(aiResponse); code != "" {
            // Optionally save generated code
            saveCode(code)
        }
    }

    // 5. Pause session
    sys.Sessions.Pause(sess.ID)
}
```

### Pattern 5: Template-Based Code Generation

**Scenario:** Generate multiple code files from templates.

```go
func GenerateAPIEndpoints(sys *HelixSystem, endpoints []EndpointSpec) error {
    // 1. Create session
    sess, _ := sys.Sessions.Create("api-project", "Generate API Endpoints",
        "Generate REST API endpoints", session.ModeBuilding)
    sys.Sessions.SetActive(sess.ID)

    // 2. Register custom API endpoint template
    apiTemplate := template.NewTemplate("API Endpoint",
        "Generate REST endpoint", template.TypeCode)
    apiTemplate.SetContent(`
// {{endpoint_name}} {{description}}
func {{handler_name}}(c *gin.Context) {
	{{input_validation}}

	{{business_logic}}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

// Register route
router.{{http_method}}("{{path}}", {{handler_name}})
`)
    apiTemplate.AddVariable(template.Variable{Name: "endpoint_name", Required: true})
    apiTemplate.AddVariable(template.Variable{Name: "description", Required: true})
    apiTemplate.AddVariable(template.Variable{Name: "handler_name", Required: true})
    apiTemplate.AddVariable(template.Variable{Name: "http_method", Required: true})
    apiTemplate.AddVariable(template.Variable{Name: "path", Required: true})
    apiTemplate.AddVariable(template.Variable{Name: "input_validation", Required: false, DefaultValue: "// No validation"})
    apiTemplate.AddVariable(template.Variable{Name: "business_logic", Required: true})

    sys.Templates.Register(apiTemplate)

    // 3. Generate endpoints
    for _, ep := range endpoints {
        code, err := sys.Templates.RenderByName("API Endpoint", map[string]interface{}{
            "endpoint_name":     ep.Name,
            "description":       ep.Description,
            "handler_name":      ep.HandlerName,
            "http_method":       ep.Method,
            "path":              ep.Path,
            "input_validation":  ep.ValidationCode,
            "business_logic":    ep.LogicCode,
        })
        if err != nil {
            return err
        }

        // Save generated code
        filename := fmt.Sprintf("handlers/%s.go", ep.HandlerName)
        saveFile(filename, code)

        // Track in focus
        chain, _ := sys.Focus.GetActiveChain()
        f := focus.NewFocus(focus.FocusTypeFile, filename)
        f.SetMetadata("generated", "true")
        chain.Push(f)
    }

    // 4. Complete session
    sys.Sessions.Complete(sess.ID)

    return nil
}

type EndpointSpec struct {
    Name           string
    Description    string
    HandlerName    string
    Method         string
    Path           string
    ValidationCode string
    LogicCode      string
}
```

---

## Advanced Integration

### Multi-Session Workflow

**Scenario:** Managing multiple concurrent development sessions.

```go
type DevelopmentWorkflow struct {
    system *HelixSystem
}

func NewDevelopmentWorkflow(sys *HelixSystem) *DevelopmentWorkflow {
    return &DevelopmentWorkflow{system: sys}
}

// StartFeature starts a new feature development session
func (w *DevelopmentWorkflow) StartFeature(name, description string) (string, error) {
    sess, err := w.system.Sessions.Create("project-1", name, description, session.ModeBuilding)
    if err != nil {
        return "", err
    }

    // Create associated conversation and focus chain
    conv, _ := w.system.Memory.CreateConversation("Feature: " + name)
    chain, _ := w.system.Focus.CreateChain(name, false)

    // Link them via metadata
    sess.SetMetadata("conversation_id", conv.ID)
    sess.SetMetadata("focus_chain_id", chain.ID)

    return sess.ID, nil
}

// SwitchSession switches to a different session
func (w *DevelopmentWorkflow) SwitchSession(sessionID string) error {
    // Get session
    sess, err := w.system.Sessions.Get(sessionID)
    if err != nil {
        return err
    }

    // Set as active
    w.system.Sessions.SetActive(sessionID)

    // Activate associated conversation
    if convID, ok := sess.GetMetadata("conversation_id"); ok {
        conv, _ := w.system.Memory.GetConversation(convID)
        if conv != nil {
            w.system.Memory.SetActive(convID)
        }
    }

    // Activate associated focus chain
    if chainID, ok := sess.GetMetadata("focus_chain_id"); ok {
        chain, _ := w.system.Focus.GetChain(chainID)
        if chain != nil {
            w.system.Focus.SetActiveChain(chainID)
        }
    }

    return nil
}

// GetSessionContext builds complete context for active session
func (w *DevelopmentWorkflow) GetSessionContext() (string, error) {
    // Build context with all active components
    w.system.Context.Clear()

    sessionSource := builder.NewSessionSource(w.system.Sessions)
    focusSource := builder.NewFocusSource(w.system.Focus, 20)

    w.system.Context.RegisterSource(sessionSource)
    w.system.Context.RegisterSource(focusSource)

    // Get active conversation
    activeConv := w.system.Memory.GetActive()
    if activeConv != nil {
        // Add recent conversation messages to context
        recent := activeConv.GetRecent(10)
        for i, msg := range recent {
            w.system.Context.AddText(
                fmt.Sprintf("Message %d", i+1),
                fmt.Sprintf("%s: %s", msg.Role, msg.Content),
                builder.PriorityNormal,
            )
        }
    }

    return w.system.Context.BuildWithTemplate("coding")
}
```

### Smart Context Building

**Scenario:** Intelligently select context based on current activity.

```go
type SmartContextBuilder struct {
    system *HelixSystem
}

func NewSmartContextBuilder(sys *HelixSystem) *SmartContextBuilder {
    return &SmartContextBuilder{system: sys}
}

// BuildContext builds context based on session mode
func (s *SmartContextBuilder) BuildContext() (string, error) {
    activeSession := s.system.Sessions.GetActive()
    if activeSession == nil {
        return s.system.Context.Build()
    }

    // Select template based on mode
    var templateName string
    switch activeSession.Mode {
    case session.ModePlanning:
        templateName = "planning"
    case session.ModeBuilding:
        templateName = "coding"
    case session.ModeTesting:
        templateName = "coding" // Testing uses coding context
    case session.ModeRefactoring:
        templateName = "refactoring"
    case session.ModeDebugging:
        templateName = "debugging"
    case session.ModeDeployment:
        templateName = "planning" // Deployment uses planning context
    default:
        templateName = "coding"
    }

    // Build with appropriate template
    return s.system.Context.BuildWithTemplate(templateName)
}

// BuildContextForPrompt builds context optimized for a specific prompt type
func (s *SmartContextBuilder) BuildContextForPrompt(promptType string) (string, error) {
    s.system.Context.Clear()

    // Always include session and focus
    sessionSource := builder.NewSessionSource(s.system.Sessions)
    focusSource := builder.NewFocusSource(s.system.Focus, 15)
    s.system.Context.RegisterSource(sessionSource)
    s.system.Context.RegisterSource(focusSource)

    // Add type-specific sources
    switch promptType {
    case "code_review":
        // Add recent file changes
        chain, _ := s.system.Focus.GetActiveChain()
        if chain != nil {
            recent := chain.GetRecent(10)
            for _, f := range recent {
                if f.Type == focus.FocusTypeFile {
                    content := readFile(f.Target)
                    fileSource := builder.NewFileSource(f.Target, content, builder.PriorityHigh)
                    s.system.Context.RegisterSource(fileSource)
                }
            }
        }

    case "bug_fix":
        // Add error history
        errorSource := builder.NewErrorSource()
        // Populate from logs or error tracking system
        s.system.Context.RegisterSource(errorSource)

    case "code_generation":
        // Add project context and recent files
        projectSource := builder.NewProjectSource("Current Project", "...", nil)
        s.system.Context.RegisterSource(projectSource)
    }

    return s.system.Context.Build()
}
```

### Template Library Management

**Scenario:** Organize and manage a library of templates.

```go
type TemplateLibrary struct {
    system *HelixSystem
}

func NewTemplateLibrary(sys *HelixSystem) *TemplateLibrary {
    return &TemplateLibrary{system: sys}
}

// LoadProjectTemplates loads templates from project directory
func (lib *TemplateLibrary) LoadProjectTemplates(projectPath string) error {
    templatesDir := filepath.Join(projectPath, ".helix", "templates")
    count, err := lib.system.Templates.LoadFromDirectory(templatesDir)
    if err != nil {
        return err
    }

    log.Printf("Loaded %d project templates", count)
    return nil
}

// SaveTemplate saves a template to project directory
func (lib *TemplateLibrary) SaveTemplate(templateID, projectPath string) error {
    templatesDir := filepath.Join(projectPath, ".helix", "templates")
    os.MkdirAll(templatesDir, 0755)

    tpl, err := lib.system.Templates.Get(templateID)
    if err != nil {
        return err
    }

    filename := filepath.Join(templatesDir, tpl.Name+".json")
    return lib.system.Templates.SaveToFile(templateID, filename)
}

// SearchTemplates searches templates with advanced filters
func (lib *TemplateLibrary) SearchTemplates(query string, filters TemplateFilters) []*template.Template {
    results := make([]*template.Template, 0)

    // Start with search
    if query != "" {
        results = lib.system.Templates.Search(query)
    } else {
        results = lib.system.Templates.GetAll()
    }

    // Apply type filter
    if filters.Type != "" {
        filtered := make([]*template.Template, 0)
        for _, tpl := range results {
            if tpl.Type == template.Type(filters.Type) {
                filtered = append(filtered, tpl)
            }
        }
        results = filtered
    }

    // Apply tag filter
    if len(filters.Tags) > 0 {
        filtered := make([]*template.Template, 0)
        for _, tpl := range results {
            hasAllTags := true
            for _, tag := range filters.Tags {
                if !tpl.HasTag(tag) {
                    hasAllTags = false
                    break
                }
            }
            if hasAllTags {
                filtered = append(filtered, tpl)
            }
        }
        results = filtered
    }

    return results
}

type TemplateFilters struct {
    Type string
    Tags []string
}
```

---

## Best Practices

### 1. Session Management

**DO:**
- ✅ Always create a session for significant development work
- ✅ Set descriptive names and descriptions
- ✅ Complete sessions when done
- ✅ Use appropriate modes (building, debugging, etc.)
- ✅ Clean up old sessions periodically

**DON'T:**
- ❌ Create sessions for trivial tasks
- ❌ Leave sessions active indefinitely
- ❌ Forget to set metadata for tracking
- ❌ Mix different types of work in one session

```go
// GOOD
sess, _ := sessions.Create("project-1", "Implement OAuth",
    "Add OAuth2 authentication", session.ModeBuilding)
defer sessions.Complete(sess.ID)

// BAD
sess, _ := sessions.Create("project-1", "Work", "", session.ModeBuilding)
// Never completed, vague name
```

### 2. Memory Management

**DO:**
- ✅ Create conversations for distinct topics
- ✅ Add context to user messages
- ✅ Search before creating duplicate conversations
- ✅ Set reasonable token limits
- ✅ Export important conversations

**DON'T:**
- ❌ Put everything in one conversation
- ❌ Let conversations grow unbounded
- ❌ Forget to set active conversation
- ❌ Skip adding assistant responses

```go
// GOOD
conv, _ := memory.CreateConversation("OAuth Implementation")
memory.SetActive(conv.ID)
memory.AddMessageToActive(memory.NewUserMessage("How to implement OAuth?"))
response := getAIResponse()
memory.AddMessageToActive(memory.NewAssistantMessage(response))

// BAD
memory.AddMessageToActive(memory.NewUserMessage("question"))
// No context, no response stored
```

### 3. Context Building

**DO:**
- ✅ Use templates for consistent context
- ✅ Set priority appropriately
- ✅ Clear context between different tasks
- ✅ Set size limits to prevent overflow
- ✅ Cache when possible

**DON'T:**
- ❌ Add irrelevant information
- ❌ Exceed token limits
- ❌ Forget to invalidate cache when data changes
- ❌ Use same context for different modes

```go
// GOOD
builder.Clear()
builder.AddText("Task", "Implement feature X", builder.PriorityHigh)
builder.SetMaxTokens(4000)
context, _ := builder.BuildWithTemplate("coding")

// BAD
builder.AddText("", "some text", 0)
context, _ := builder.Build() // No limit, no template
```

### 4. Template Usage

**DO:**
- ✅ Use descriptive names and descriptions
- ✅ Validate templates before registering
- ✅ Provide default values for optional variables
- ✅ Tag templates for easy finding
- ✅ Version templates when updating

**DON'T:**
- ❌ Hardcode values that should be variables
- ❌ Forget to declare required variables
- ❌ Use generic names like "Template1"
- ❌ Skip validation

```go
// GOOD
tpl := template.NewTemplate("REST Endpoint", "Generate API endpoint", template.TypeCode)
tpl.SetContent("func {{name}}(c *gin.Context) { {{body}} }")
tpl.AddVariable(template.Variable{Name: "name", Required: true})
tpl.AddVariable(template.Variable{Name: "body", Required: true})
tpl.AddTag("api")
tpl.Validate()

// BAD
tpl := template.NewTemplate("t1", "", template.TypeCode)
tpl.SetContent("func handler() { doStuff() }")
// No variables, no validation
```

### 5. Persistence

**DO:**
- ✅ Enable auto-save for safety
- ✅ Create backups before major changes
- ✅ Handle load errors gracefully
- ✅ Save before shutdown
- ✅ Monitor disk space

**DON'T:**
- ❌ Rely solely on auto-save
- ❌ Ignore save/load errors
- ❌ Save too frequently (performance)
- ❌ Forget to set all managers

```go
// GOOD
store.SetSessionManager(sessions)
store.SetMemoryManager(memory)
store.SetFocusManager(focus)
store.EnableAutoSave(5 * time.Minute)
defer store.SaveAll()

if err := store.LoadAll(); err != nil {
    log.Printf("Warning: Failed to load state: %v", err)
}

// BAD
store.EnableAutoSave(1 * time.Second) // Too frequent
// No error handling, missing managers
```

---

## Troubleshooting

### Common Issues

#### Issue: High Memory Usage

**Symptoms:** Application consuming excessive memory

**Solutions:**
```go
// Set conversation limits
memory.SetMaxMessages(100)
memory.SetMaxTokens(10000)

// Cleanup old conversations
memory.TrimConversations()

// Clean old sessions
sessions.CleanupOldSessions(24 * time.Hour)
```

#### Issue: Context Too Large

**Symptoms:** LLM errors due to token limits

**Solutions:**
```go
// Set strict limits
builder.SetMaxTokens(4000)
builder.SetMaxSize(50000)

// Use higher priority only for important items
builder.AddText("Important", content, builder.PriorityCritical)

// Reduce focus history
focusSource := builder.NewFocusSource(focus, 5) // Only 5 items
```

#### Issue: Slow Persistence

**Symptoms:** Save operations taking too long

**Solutions:**
```go
// Use compression for large data
gzipSerializer := persistence.NewJSONGzipSerializer()
store.SetSerializer(gzipSerializer)

// Increase auto-save interval
store.EnableAutoSave(10 * time.Minute)

// Use separate goroutine for saves
go func() {
    store.SaveAll()
}()
```

#### Issue: Template Rendering Errors

**Symptoms:** "template has unreplaced placeholders"

**Solutions:**
```go
// Check all required variables provided
vars := map[string]interface{}{
    "name": "value", // Must match template variables
}

// Validate template first
if err := tpl.Validate(); err != nil {
    log.Fatal(err)
}

// Check extracted variables
extractedVars := tpl.ExtractVariables()
fmt.Println("Template requires:", extractedVars)
```

---

## Conclusion

This integration guide demonstrates practical patterns for using HelixCode Phase 3 features together. The examples show real-world scenarios from simple setups to advanced workflows.

**Key Takeaways:**
1. Initialize all systems together for best integration
2. Use sessions to organize development work
3. Build context intelligently based on mode
4. Store conversations for continuity
5. Use templates for consistency
6. Persist state regularly

For more details, refer to individual feature documentation.

---

**End of Integration Guide**

**Version:** 1.0
**Created:** November 7, 2025
