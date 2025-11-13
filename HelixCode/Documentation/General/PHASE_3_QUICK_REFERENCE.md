# Phase 3 Quick Reference

Quick reference guide for common Phase 3 operations.

---

## Initialization

```go
import (
    "dev.helix.code/internal/session"
    "dev.helix.code/internal/memory"
    "dev.helix.code/internal/persistence"
    "dev.helix.code/internal/template"
)

// Create managers
sessionMgr := session.NewManager()
memoryMgr := memory.NewManager()
templateMgr := template.NewManager()

// Set up persistence
store := persistence.NewStore("./helixcode_data")
store.SetSessionManager(sessionMgr)
store.SetMemoryManager(memoryMgr)
store.SetTemplateManager(templateMgr)
store.EnableAutoSave(300)  // 5 minutes

// Load built-in templates
templateMgr.RegisterBuiltinTemplates()

// Restore previous state
store.Load()
```

---

## Session Management

### Create and Start Session
```go
sess := sessionMgr.Create("feature-name", session.ModeBuilding, "project-id")
sess.AddTag("api")
sessionMgr.Start(sess.ID)
```

### Session Modes
```go
session.ModePlanning     // Design and architecture
session.ModeBuilding     // Implementation
session.ModeTesting      // Quality assurance
session.ModeRefactoring  // Code improvement
session.ModeDebugging    // Issue investigation
session.ModeDeployment   // Release preparation
```

### Session Lifecycle
```go
sessionMgr.Start(id)      // idle → active
sessionMgr.Pause(id)      // active → paused
sessionMgr.Resume(id)     // paused → active
sessionMgr.Complete(id)   // active → completed
sessionMgr.Fail(id, reason) // active → failed
```

### Query Sessions
```go
all := sessionMgr.GetAll()
byProject := sessionMgr.GetByProject("project-id")
byMode := sessionMgr.GetByMode(session.ModeBuilding)
byStatus := sessionMgr.GetByStatus(session.StatusActive)
byTag := sessionMgr.GetByTag("api")
recent := sessionMgr.GetRecent(10)
```

### Session Properties
```go
sess.SetMetadata("key", "value")
sess.SetContext("last_file", "main.go")
value := sess.GetMetadata("key")
```

### Export/Import
```go
// Export
snapshot, _ := sessionMgr.Export(sess.ID)
data, _ := json.Marshal(snapshot)

// Import
var snapshot session.SessionSnapshot
json.Unmarshal(data, &snapshot)
sessionMgr.Import(&snapshot)
```

---

## Memory System

### Create Conversation
```go
conv := memoryMgr.CreateConversation("Feature Implementation")
conv.SessionID = sess.ID  // Link to session
```

### Add Messages
```go
// User message
memoryMgr.AddMessage(conv.ID, memory.NewUserMessage("Help me debug this"))

// Assistant message
memoryMgr.AddMessage(conv.ID, memory.NewAssistantMessage("Sure, let's look at..."))

// System message
memoryMgr.AddMessage(conv.ID, memory.NewSystemMessage("You are a debugging expert"))
```

### Message Metadata
```go
msg := memory.NewUserMessage("content")
msg.SetMetadata("file", "main.go")
msg.SetMetadata("line", "42")
```

### Query Messages
```go
all := conv.GetMessages()
userOnly := conv.GetByRole(memory.RoleUser)
recent := conv.GetRecent(10)
range := conv.GetRange(10, 20)
results := conv.Search("debug")
text := conv.ToText()
```

### Conversation Limits
```go
conv.SetMaxMessages(500)
conv.SetMaxTokens(50000)
total := conv.TotalTokens()
conv.Truncate(100)  // Keep last 100 messages
```

### Query Conversations
```go
all := memoryMgr.GetAll()
bySession := memoryMgr.GetBySession("session-id")
recent := memoryMgr.GetRecent(10)
found := memoryMgr.Search("authentication")
messages := memoryMgr.SearchMessages("error")
```

### Manager Operations
```go
memoryMgr.SetMaxMessages(500)
memoryMgr.SetMaxTokens(50000)
memoryMgr.TrimConversations(50)  // Keep last 50
memoryMgr.ClearConversation(conv.ID)
memoryMgr.DeleteConversation(conv.ID)
```

---

## State Persistence

### Save Operations
```go
store.Save()              // Save all
store.SaveSessions()      // Save sessions only
store.SaveConversations() // Save conversations only
store.SaveTemplates()     // Save templates only
```

### Load Operations
```go
store.Load()              // Load all
store.LoadSessions()      // Load sessions only
store.LoadConversations() // Load conversations only
store.LoadTemplates()     // Load templates only
```

### Auto-Save
```go
store.EnableAutoSave(300)  // Every 5 minutes
store.DisableAutoSave()
enabled := store.IsAutoSaveEnabled()
interval := store.GetAutoSaveInterval()
```

### Backup and Restore
```go
// Backup
backupPath := "./backups/backup-" + time.Now().Format("20060102-150405")
store.Backup(backupPath)

// Restore
store.Restore(backupPath)
```

### Storage Formats
```go
store.SetFormat(persistence.FormatJSON)        // Human-readable
store.SetFormat(persistence.FormatCompactJSON) // Smaller
store.SetFormat(persistence.FormatJSONGZIP)    // Compressed
```

### Callbacks
```go
store.OnSave(func() {
    log.Println("Saved")
})

store.OnLoad(func() {
    log.Println("Loaded")
})

store.OnError(func(err error) {
    log.Printf("Error: %v\n", err)
})
```

---

## Template System

### Create Template
```go
tpl := template.NewTemplate("Handler", "HTTP handler", template.TypeCode)
tpl.SetContent(`func Handle{{name}}(w http.ResponseWriter, r *http.Request) {
    {{implementation}}
}`)

tpl.AddVariable(template.Variable{
    Name:     "name",
    Required: true,
    Type:     "string",
})

tpl.AddVariable(template.Variable{
    Name:         "implementation",
    Required:     false,
    DefaultValue: "// TODO",
    Type:         "string",
})

templateMgr.Register(tpl)
```

### Template Types
```go
template.TypeCode          // Code generation
template.TypePrompt        // AI prompts
template.TypeWorkflow      // Process templates
template.TypeDocumentation // Documentation
template.TypeEmail         // Email templates
template.TypeCustom        // Custom use
```

### Render Templates
```go
// By ID
result, _ := templateMgr.Render(tpl.ID, map[string]interface{}{
    "name": "Users",
    "implementation": "// Handle users",
})

// By name
result, _ := templateMgr.RenderByName("Function", map[string]interface{}{
    "function_name": "ProcessData",
    "parameters":    "data []byte",
    "return_type":   "error",
    "body":          "return nil",
})
```

### Built-in Templates
```go
templateMgr.RegisterBuiltinTemplates()

// 1. Function
templateMgr.RenderByName("Function", vars)

// 2. Code Review
templateMgr.RenderByName("Code Review", map[string]interface{}{
    "language": "Go",
    "code":     sourceCode,
})

// 3. Bug Fix
templateMgr.RenderByName("Bug Fix", map[string]interface{}{
    "language":         "Go",
    "error_message":    "nil pointer",
    "code":             buggyCode,
    "expected_behavior": "Handle nil",
    "actual_behavior":   "Crashes",
})

// 4. Function Documentation
templateMgr.RenderByName("Function Documentation", vars)

// 5. Status Update Email
templateMgr.RenderByName("Status Update Email", vars)
```

### Query Templates
```go
tpl, _ := templateMgr.Get("template-id")
tpl, _ := templateMgr.GetByName("Handler")
all := templateMgr.GetAll()
codeTpls := templateMgr.GetByType(template.TypeCode)
tagged := templateMgr.GetByTag("http")
results := templateMgr.Search("handler")
```

### Template Properties
```go
tpl.AddTag("http")
tpl.SetMetadata("language", "go")
tpl.Version = "2.0.0"
hasTag := tpl.HasTag("http")
meta := tpl.GetMetadata("language")
```

### File Operations
```go
// Load from file
templateMgr.LoadFromFile("./templates/handler.json")

// Load directory
count, _ := templateMgr.LoadFromDirectory("./templates")

// Save to file
templateMgr.SaveToFile(tpl.ID, "./templates/my-template.json")
```

### Export/Import
```go
// Export
snapshot, _ := templateMgr.Export(tpl.ID)
data, _ := json.Marshal(snapshot)

// Import
var snapshot template.TemplateSnapshot
json.Unmarshal(data, &snapshot)
templateMgr.Import(&snapshot)
```

---

## Integration Patterns

### Session + Conversation
```go
// Create linked session and conversation
sess := sessionMgr.Create("feature", session.ModeBuilding, "project")
sessionMgr.Start(sess.ID)

conv := memoryMgr.CreateConversation(sess.Name)
conv.SessionID = sess.ID
```

### Template + Conversation
```go
// Generate prompt from template
prompt, _ := templateMgr.RenderByName("Code Review", vars)

// Add to conversation
memoryMgr.AddMessage(conv.ID, memory.NewUserMessage(prompt))
```

### Complete Workflow
```go
// 1. Create session
sess := sessionMgr.Create("auth", session.ModeBuilding, "api")
sessionMgr.Start(sess.ID)

// 2. Create conversation
conv := memoryMgr.CreateConversation("Auth Implementation")
conv.SessionID = sess.ID

// 3. Use template
code, _ := templateMgr.RenderByName("Function", vars)

// 4. Add to conversation
memoryMgr.AddMessage(conv.ID, memory.NewAssistantMessage(code))

// 5. Auto-saved by persistence
```

---

## Common Tasks

### Start New Feature
```go
sess := sessionMgr.Create("feature-name", session.ModeBuilding, "project")
sess.AddTag("feature")
sessionMgr.Start(sess.ID)

conv := memoryMgr.CreateConversation("Feature: " + sess.Name)
conv.SessionID = sess.ID
```

### Debug Issue
```go
sess := sessionMgr.Create("bug-123", session.ModeDebugging, "project")
sess.SetMetadata("issue_id", "123")
sessionMgr.Start(sess.ID)

conv := memoryMgr.CreateConversation("Debug: Issue 123")
conv.SessionID = sess.ID

prompt, _ := templateMgr.RenderByName("Bug Fix", vars)
memoryMgr.AddMessage(conv.ID, memory.NewUserMessage(prompt))
```

### Code Review
```go
sess := sessionMgr.Create("review-pr-42", session.ModeBuilding, "project")
sess.AddTag("review")
sessionMgr.Start(sess.ID)

conv := memoryMgr.CreateConversation("PR #42 Review")
conv.SessionID = sess.ID

prompt, _ := templateMgr.RenderByName("Code Review", map[string]interface{}{
    "language": "Go",
    "code":     prCode,
})
memoryMgr.AddMessage(conv.ID, memory.NewUserMessage(prompt))
```

### Create Template Library
```go
// Create templates directory
os.MkdirAll("./templates", 0755)

// Save all templates
for _, tpl := range templateMgr.GetAll() {
    filename := fmt.Sprintf("./templates/%s.json", tpl.Name)
    templateMgr.SaveToFile(tpl.ID, filename)
}

// Load on startup
count, _ := templateMgr.LoadFromDirectory("./templates")
```

### Export Work
```go
// Export session
sessSnapshot, _ := sessionMgr.Export(sess.ID)
convSnapshot, _ := memoryMgr.ExportConversation(conv.ID)

// Save to files
json.NewEncoder(os.Create("session.json")).Encode(sessSnapshot)
json.NewEncoder(os.Create("conversation.json")).Encode(convSnapshot)
```

---

## Configuration

### Recommended Config
```yaml
persistence:
  storage_path: "./helixcode_data"
  format: "json-gzip"  # Compressed
  auto_save: true
  auto_save_interval: 300  # 5 minutes

session:
  max_history: 100
  default_mode: "building"

memory:
  max_messages_per_conversation: 500
  max_total_tokens: 50000

templates:
  template_directory: "./templates"
  load_builtin: true
```

### In Code
```go
// Set limits
conv.SetMaxMessages(500)
conv.SetMaxTokens(50000)
memoryMgr.SetMaxMessages(500)

// Configure auto-save
store.EnableAutoSave(300)

// Set format
store.SetFormat(persistence.FormatJSONGZIP)
```

---

## Callbacks

### Session Events
```go
sessionMgr.OnCreate(func(s *session.Session) {
    log.Printf("Created: %s\n", s.Name)
})

sessionMgr.OnComplete(func(s *session.Session) {
    log.Printf("Completed: %s\n", s.Name)
})
```

### Memory Events
```go
memoryMgr.OnMessage(func(conv *memory.Conversation, msg *memory.Message) {
    log.Printf("New message in %s\n", conv.Title)
})
```

### Persistence Events
```go
store.OnSave(func() {
    log.Println("State saved")
})

store.OnError(func(err error) {
    log.Printf("Error: %v\n", err)
})
```

### Template Events
```go
templateMgr.OnCreate(func(t *template.Template) {
    log.Printf("Template created: %s\n", t.Name)
})
```

---

## Error Handling

```go
// Always check errors
sess, err := sessionMgr.Create("name", mode, project)
if err != nil {
    log.Fatalf("Failed: %v", err)
}

// Graceful degradation
if err := store.Load(); err != nil {
    log.Printf("Could not load state: %v\n", err)
    // Continue with fresh state
}
```

---

## Performance Tips

```go
// 1. Set limits
conv.SetMaxMessages(500)
conv.SetMaxTokens(50000)

// 2. Use GZIP for large data
store.SetFormat(persistence.FormatJSONGZIP)

// 3. Trim regularly
sessionMgr.TrimHistory(50)
memoryMgr.TrimConversations(50)

// 4. Auto-save interval (not too frequent)
store.EnableAutoSave(300)  // 5 minutes

// 5. Delete old data
for _, sess := range sessionMgr.GetAll() {
    if sess.Status == session.StatusCompleted {
        sessionMgr.Export(sess.ID)  // Backup first
        sessionMgr.Delete(sess.ID)
    }
}
```

---

## Testing

```go
// Test with temporary storage
store := persistence.NewStore(t.TempDir())

// Create test data
sess := sessionMgr.Create("test", session.ModeBuilding, "test-project")
conv := memoryMgr.CreateConversation("test")

// Test save/load
store.Save()
newStore := persistence.NewStore(store.Path())
newStore.Load()

// Verify
assert.Equal(t, 1, sessionMgr.Count())
```

---

## Common Patterns

### Multi-Session Workflow
```go
authSess := sessionMgr.Create("auth", session.ModeBuilding, "api")
testSess := sessionMgr.Create("tests", session.ModeTesting, "api")

sessionMgr.Start(authSess.ID)
// ... work on auth
sessionMgr.Pause(authSess.ID)

sessionMgr.Start(testSess.ID)
// ... write tests
sessionMgr.Complete(testSess.ID)

sessionMgr.Resume(authSess.ID)
// ... continue auth
```

### Context Preservation
```go
// Save context in session
sess.SetContext("current_file", "auth.go")
sess.SetContext("current_line", "42")

// Retrieve later
file := sess.GetContext("current_file")
line := sess.GetContext("current_line")
```

### Template Composition
```go
header, _ := templateMgr.RenderByName("Header", vars)
body, _ := templateMgr.RenderByName("Body", vars)
footer, _ := templateMgr.RenderByName("Footer", vars)

complete := header + "\n" + body + "\n" + footer
```

---

**For complete documentation, see [Phase 3 API Reference](./PHASE_3_API_REFERENCE.md)**
