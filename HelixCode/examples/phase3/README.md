# Phase 3 Examples

Complete working examples demonstrating all Phase 3 features.

## Examples

### 1. Basic Usage (`basic/`)
**What it demonstrates:**
- Initializing all Phase 3 managers
- Creating a session
- Building a conversation
- Using templates
- Saving and loading state

**Run it:**
```bash
cd basic
go run main.go
```

### 2. Feature Development (`feature-dev/`)
**What it demonstrates:**
- Complete feature implementation workflow
- Multiple session modes (planning, building, testing)
- Template-based code generation
- Cross-session context

**Run it:**
```bash
cd feature-dev
go run main.go
```

### 3. Code Review (`code-review/`)
**What it demonstrates:**
- Automated code review workflow
- Using the Code Review template
- Capturing review feedback
- Exporting review conversations

**Run it:**
```bash
cd code-review
go run main.go
```

### 4. Debugging (`debugging/`)
**What it demonstrates:**
- Debugging session workflow
- Using the Bug Fix template
- Tracking bug resolution
- Session metadata for issues

**Run it:**
```bash
cd debugging
go run main.go
```

### 5. Templates (`templates/`)
**What it demonstrates:**
- Creating custom templates
- Using built-in templates
- Template library management
- Variable substitution

**Run it:**
```bash
cd templates
go run main.go
```

### 6. Multi-Session Workflow (`multi-session/`)
**What it demonstrates:**
- Working on multiple tasks
- Pausing and resuming sessions
- Session switching
- Concurrent development

**Run it:**
```bash
cd multi-session
go run main.go
```

## Prerequisites

All examples require:
- Go 1.24.0 or higher
- HelixCode Phase 3 modules installed

## Learning Path

1. Start with `basic/` to understand fundamentals
2. Try `templates/` to learn template system
3. Run `feature-dev/` for complete workflow
4. Explore `code-review/` and `debugging/` for specific use cases
5. Master `multi-session/` for advanced workflows

## Common Patterns

### Initialize Phase 3
```go
sessionMgr := session.NewManager()
memoryMgr := memory.NewManager()
templateMgr := template.NewManager()

store := persistence.NewStore("./data")
store.SetSessionManager(sessionMgr)
store.SetMemoryManager(memoryMgr)
store.SetTemplateManager(templateMgr)
store.EnableAutoSave(300)

templateMgr.RegisterBuiltinTemplates()
store.Load()
```

### Create and Use Session
```go
sess := sessionMgr.Create("task-name", session.ModeBuilding, "project")
sess.AddTag("tag")
sessionMgr.Start(sess.ID)

// ... work

sessionMgr.Complete(sess.ID)
```

### Use Templates
```go
result, _ := templateMgr.RenderByName("Function", map[string]interface{}{
    "function_name": "MyFunc",
    "parameters":    "x int",
    "return_type":   "int",
    "body":          "return x * 2",
})
```

## More Information

- [Phase 3 Features Guide](../../docs/PHASE_3_FEATURES.md)
- [API Reference](../../docs/PHASE_3_API_REFERENCE.md)
- [Quick Reference](../../docs/PHASE_3_QUICK_REFERENCE.md)
