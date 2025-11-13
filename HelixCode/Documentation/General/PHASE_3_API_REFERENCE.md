# Phase 3 API Reference

Complete API documentation for all Phase 3 features.

---

## Table of Contents

1. [Session Management](#session-management)
2. [Memory System](#memory-system)
3. [State Persistence](#state-persistence)
4. [Template System](#template-system)

---

## Session Management

**Package:** `dev.helix.code/internal/session`

### Types

#### Mode
```go
type Mode string

const (
    ModePlanning    Mode = "planning"     // Design and architecture
    ModeBuilding    Mode = "building"     // Implementation
    ModeTesting     Mode = "testing"      // Quality assurance
    ModeRefactoring Mode = "refactoring"  // Code improvement
    ModeDebugging   Mode = "debugging"    // Issue investigation
    ModeDeployment  Mode = "deployment"   // Release preparation
)
```

#### Status
```go
type Status string

const (
    StatusIdle      Status = "idle"       // Created but not started
    StatusActive    Status = "active"     // Currently running
    StatusPaused    Status = "paused"     // Temporarily stopped
    StatusCompleted Status = "completed"  // Successfully finished
    StatusFailed    Status = "failed"     // Ended with failure
)
```

#### Session
```go
type Session struct {
    ID          string            // Unique identifier
    Name        string            // Human-readable name
    Mode        Mode              // Session mode
    Status      Status            // Current status
    ProjectID   string            // Associated project
    Tags        []string          // Searchable tags
    Metadata    map[string]string // Custom metadata
    Context     map[string]string // Session context data
    CreatedAt   time.Time         // Creation timestamp
    StartedAt   time.Time         // Start timestamp
    EndedAt     time.Time         // End timestamp
    LastPausedAt time.Time        // Last pause timestamp
}
```

### Manager

#### Creation
```go
func NewManager() *Manager
```

Creates a new session manager.

**Example:**
```go
mgr := session.NewManager()
```

#### Create Session
```go
func (m *Manager) Create(name string, mode Mode, projectID string) (*Session, error)
```

Creates a new session.

**Parameters:**
- `name`: Session name
- `mode`: Session mode (planning, building, etc.)
- `projectID`: Associated project ID

**Returns:**
- `*Session`: Created session
- `error`: Error if creation fails

**Example:**
```go
sess, err := mgr.Create("implement-auth", session.ModeBuilding, "api-server")
if err != nil {
    log.Fatal(err)
}
```

#### Get Session
```go
func (m *Manager) Get(id string) (*Session, error)
```

Retrieves a session by ID.

**Example:**
```go
sess, err := mgr.Get("session-id")
```

#### Start Session
```go
func (m *Manager) Start(id string) error
```

Starts a session (idle → active).

**Example:**
```go
err := mgr.Start(sess.ID)
```

#### Pause Session
```go
func (m *Manager) Pause(id string) error
```

Pauses a session (active → paused).

**Example:**
```go
err := mgr.Pause(sess.ID)
```

#### Resume Session
```go
func (m *Manager) Resume(id string) error
```

Resumes a paused session (paused → active).

**Example:**
```go
err := mgr.Resume(sess.ID)
```

#### Complete Session
```go
func (m *Manager) Complete(id string) error
```

Completes a session (active → completed).

**Example:**
```go
err := mgr.Complete(sess.ID)
```

#### Fail Session
```go
func (m *Manager) Fail(id string, reason string) error
```

Marks session as failed (active → failed).

**Example:**
```go
err := mgr.Fail(sess.ID, "Tests failed")
```

#### Delete Session
```go
func (m *Manager) Delete(id string) error
```

Deletes a session. Cannot delete active sessions.

**Example:**
```go
err := mgr.Delete(sess.ID)
```

#### Query Methods
```go
func (m *Manager) GetAll() []*Session
func (m *Manager) GetByProject(projectID string) []*Session
func (m *Manager) GetByMode(mode Mode) []*Session
func (m *Manager) GetByStatus(status Status) []*Session
func (m *Manager) GetByTag(tag string) []*Session
func (m *Manager) GetRecent(limit int) []*Session
func (m *Manager) FindByName(name string) []*Session
func (m *Manager) Count() int
func (m *Manager) CountByStatus() map[Status]int
```

#### Export/Import
```go
func (m *Manager) Export(id string) (*SessionSnapshot, error)
func (m *Manager) Import(snapshot *SessionSnapshot) error
```

**Example:**
```go
// Export
snapshot, err := mgr.Export(sess.ID)
data, _ := json.Marshal(snapshot)

// Import
var snapshot session.SessionSnapshot
json.Unmarshal(data, &snapshot)
mgr.Import(&snapshot)
```

#### Callbacks
```go
func (m *Manager) OnCreate(callback SessionCallback)
func (m *Manager) OnStart(callback SessionCallback)
func (m *Manager) OnPause(callback SessionCallback)
func (m *Manager) OnComplete(callback SessionCallback)
func (m *Manager) OnSwitch(callback SwitchCallback)
```

**Example:**
```go
mgr.OnCreate(func(s *session.Session) {
    log.Printf("Session created: %s\n", s.Name)
})
```

---

## Memory System

**Package:** `dev.helix.code/internal/memory`

### Types

#### Role
```go
type Role string

const (
    RoleUser      Role = "user"      // User messages
    RoleAssistant Role = "assistant" // AI assistant messages
    RoleSystem    Role = "system"    // System/context messages
)
```

#### Message
```go
type Message struct {
    ID        string            // Unique identifier
    Role      Role              // Message role
    Content   string            // Message content
    Metadata  map[string]string // Custom metadata
    Tokens    int               // Estimated token count
    CreatedAt time.Time         // Creation timestamp
}
```

#### Conversation
```go
type Conversation struct {
    ID        string            // Unique identifier
    Title     string            // Conversation title
    SessionID string            // Associated session ID
    Metadata  map[string]string // Custom metadata
    CreatedAt time.Time         // Creation timestamp
    UpdatedAt time.Time         // Last update timestamp
    // Internal: messages slice
}
```

### Message Creation
```go
func NewMessage(role Role, content string) *Message
func NewUserMessage(content string) *Message
func NewAssistantMessage(content string) *Message
func NewSystemMessage(content string) *Message
```

**Example:**
```go
msg := memory.NewUserMessage("Help me debug this code")
msg.SetMetadata("file", "main.go")
```

### Manager

#### Creation
```go
func NewManager() *Manager
```

Creates a new memory manager.

**Example:**
```go
mgr := memory.NewManager()
```

#### Create Conversation
```go
func (m *Manager) CreateConversation(title string) *Conversation
```

Creates a new conversation.

**Example:**
```go
conv := mgr.CreateConversation("Feature Implementation")
conv.SessionID = "session-123"
```

#### Get Conversation
```go
func (m *Manager) Get(id string) (*Conversation, error)
```

Retrieves a conversation by ID.

**Example:**
```go
conv, err := mgr.Get("conv-id")
```

#### Add Message
```go
func (m *Manager) AddMessage(conversationID string, message *Message) error
func (m *Manager) AddMessageToActive(message *Message) error
```

Adds a message to a conversation.

**Example:**
```go
mgr.AddMessage(conv.ID, memory.NewUserMessage("Hello"))
```

#### Delete Conversation
```go
func (m *Manager) DeleteConversation(id string) error
```

Deletes a conversation.

**Example:**
```go
err := mgr.DeleteConversation(conv.ID)
```

#### Clear Conversation
```go
func (m *Manager) ClearConversation(id string) error
```

Clears all messages from a conversation.

**Example:**
```go
err := mgr.ClearConversation(conv.ID)
```

#### Query Methods
```go
func (m *Manager) GetAll() []*Conversation
func (m *Manager) GetBySession(sessionID string) []*Conversation
func (m *Manager) GetRecent(limit int) []*Conversation
func (m *Manager) Search(query string) []*Conversation
func (m *Manager) SearchMessages(query string) []*Message
func (m *Manager) TotalMessages() int
func (m *Manager) TotalTokens() int
```

#### Limits
```go
func (m *Manager) SetMaxMessages(max int)
func (m *Manager) SetMaxTokens(max int)
func (m *Manager) TrimConversations(keep int)
```

**Example:**
```go
mgr.SetMaxMessages(500)
mgr.SetMaxTokens(50000)
mgr.TrimConversations(50)  // Keep last 50 conversations
```

#### Export/Import
```go
func (m *Manager) ExportConversation(id string) (*ConversationSnapshot, error)
func (m *Manager) ImportConversation(snapshot *ConversationSnapshot) error
```

#### Callbacks
```go
func (m *Manager) OnCreate(callback ConversationCallback)
func (m *Manager) OnMessage(callback MessageCallback)
func (m *Manager) OnClear(callback ConversationCallback)
func (m *Manager) OnDelete(callback ConversationCallback)
```

### Conversation Methods
```go
func (c *Conversation) GetMessages() []*Message
func (c *Conversation) GetByRole(role Role) []*Message
func (c *Conversation) GetRecent(limit int) []*Message
func (c *Conversation) GetRange(start, end int) []*Message
func (c *Conversation) Search(query string) []*Message
func (c *Conversation) TotalTokens() int
func (c *Conversation) SetMaxMessages(max int)
func (c *Conversation) SetMaxTokens(max int)
func (c *Conversation) Truncate(keep int)
func (c *Conversation) ToText() string
func (c *Conversation) Clone() *Conversation
```

**Example:**
```go
// Get recent messages
recent := conv.GetRecent(10)

// Search
results := conv.Search("authentication")

// Token management
conv.SetMaxTokens(10000)
total := conv.TotalTokens()

// Export as text
text := conv.ToText()
```

---

## State Persistence

**Package:** `dev.helix.code/internal/persistence`

### Types

#### Format
```go
type Format string

const (
    FormatJSON        Format = "json"         // Human-readable JSON
    FormatCompactJSON Format = "compact-json" // Compact JSON
    FormatJSONGZIP    Format = "json-gzip"    // Compressed JSON
)
```

#### Store
```go
type Store struct {
    // Internal fields
}
```

### Store Creation
```go
func NewStore(storagePath string) *Store
```

Creates a new persistence store.

**Parameters:**
- `storagePath`: Directory for storing data

**Example:**
```go
store := persistence.NewStore("./helixcode_data")
```

### Configuration
```go
func (s *Store) SetSessionManager(mgr *session.Manager)
func (s *Store) SetMemoryManager(mgr *memory.Manager)
func (s *Store) SetTemplateManager(mgr *template.Manager)
func (s *Store) SetFocusManager(mgr *focus.Manager)
func (s *Store) SetFormat(format Format)
func (s *Store) SetSerializer(serializer Serializer)
```

**Example:**
```go
store.SetSessionManager(sessionMgr)
store.SetMemoryManager(memoryMgr)
store.SetFormat(persistence.FormatJSONGZIP)
```

### Save Operations
```go
func (s *Store) Save() error
func (s *Store) SaveSessions() error
func (s *Store) SaveConversations() error
func (s *Store) SaveFocusChains() error
func (s *Store) SaveTemplates() error
```

**Example:**
```go
// Save everything
if err := store.Save(); err != nil {
    log.Fatal(err)
}

// Save specific component
store.SaveSessions()
```

### Load Operations
```go
func (s *Store) Load() error
func (s *Store) LoadSessions() error
func (s *Store) LoadConversations() error
func (s *Store) LoadFocusChains() error
func (s *Store) LoadTemplates() error
```

**Example:**
```go
// Load everything
if err := store.Load(); err != nil {
    log.Println("No previous state found")
}

// Load specific component
store.LoadSessions()
```

### Auto-Save
```go
func (s *Store) EnableAutoSave(intervalSeconds int)
func (s *Store) DisableAutoSave()
func (s *Store) IsAutoSaveEnabled() bool
func (s *Store) GetAutoSaveInterval() int
```

**Example:**
```go
// Enable auto-save every 5 minutes
store.EnableAutoSave(300)

// Check status
if store.IsAutoSaveEnabled() {
    interval := store.GetAutoSaveInterval()
    fmt.Printf("Auto-save every %d seconds\n", interval)
}

// Disable
store.DisableAutoSave()
```

### Backup and Restore
```go
func (s *Store) Backup(backupPath string) error
func (s *Store) Restore(backupPath string) error
```

**Example:**
```go
// Create backup
backupPath := fmt.Sprintf("./backups/backup-%s", time.Now().Format("20060102-150405"))
if err := store.Backup(backupPath); err != nil {
    log.Fatal(err)
}

// Restore from backup
if err := store.Restore(backupPath); err != nil {
    log.Fatal(err)
}
```

### Utility Methods
```go
func (s *Store) Clear() error
func (s *Store) LastSaveTime() time.Time
```

### Callbacks
```go
func (s *Store) OnSave(callback func())
func (s *Store) OnLoad(callback func())
func (s *Store) OnError(callback func(error))
```

**Example:**
```go
store.OnSave(func() {
    log.Println("State saved successfully")
})

store.OnError(func(err error) {
    log.Printf("Persistence error: %v\n", err)
})
```

### Format Detection
```go
func DetectFormat(filename string) Format
func ValidateFormat(filename string, format Format) error
```

**Example:**
```go
format := persistence.DetectFormat("./data/sessions.json")
if err := persistence.ValidateFormat("./data/sessions.json", format); err != nil {
    log.Fatal(err)
}
```

---

## Template System

**Package:** `dev.helix.code/internal/template`

### Types

#### Type
```go
type Type string

const (
    TypeCode          Type = "code"          // Code generation
    TypePrompt        Type = "prompt"        // AI prompts
    TypeWorkflow      Type = "workflow"      // Process templates
    TypeDocumentation Type = "documentation" // Documentation
    TypeEmail         Type = "email"         // Email templates
    TypeCustom        Type = "custom"        // Custom use
)
```

#### Variable
```go
type Variable struct {
    Name         string // Variable name
    Description  string // Variable description
    Required     bool   // Is required?
    DefaultValue string // Default if not provided
    Type         string // Variable type hint
}
```

#### Template
```go
type Template struct {
    ID          string            // Unique identifier
    Name        string            // Template name
    Description string            // Template description
    Type        Type              // Template type
    Content     string            // Template content with {{placeholders}}
    Variables   []Variable        // Variable definitions
    Metadata    map[string]string // Custom metadata
    Tags        []string          // Searchable tags
    Author      string            // Template author
    Version     string            // Template version
    CreatedAt   time.Time         // Creation timestamp
    UpdatedAt   time.Time         // Last update timestamp
}
```

### Template Creation
```go
func NewTemplate(name, description string, templateType Type) *Template
```

Creates a new template.

**Example:**
```go
tpl := template.NewTemplate(
    "API Handler",
    "Generate HTTP handler",
    template.TypeCode,
)
```

### Template Methods
```go
func (t *Template) SetContent(content string)
func (t *Template) AddVariable(variable Variable)
func (t *Template) Render(vars map[string]interface{}) (string, error)
func (t *Template) Validate() error
func (t *Template) Clone() *Template
func (t *Template) ExtractVariables() []string
func (t *Template) SetMetadata(key, value string)
func (t *Template) GetMetadata(key string) string
func (t *Template) AddTag(tag string)
func (t *Template) HasTag(tag string) bool
```

**Example:**
```go
tpl.SetContent(`func {{name}}() {{return_type}} {
    {{body}}
}`)

tpl.AddVariable(template.Variable{
    Name:     "name",
    Required: true,
    Type:     "string",
})

tpl.AddVariable(template.Variable{
    Name:         "return_type",
    Required:     false,
    DefaultValue: "void",
    Type:         "string",
})

result, err := tpl.Render(map[string]interface{}{
    "name": "ProcessData",
    "body": "return nil",
})
```

### Manager

#### Creation
```go
func NewManager() *Manager
```

Creates a new template manager.

**Example:**
```go
mgr := template.NewManager()
```

#### Register Template
```go
func (m *Manager) Register(template *Template) error
```

Registers a template.

**Example:**
```go
if err := mgr.Register(tpl); err != nil {
    log.Fatal(err)
}
```

#### Get Template
```go
func (m *Manager) Get(id string) (*Template, error)
func (m *Manager) GetByName(name string) (*Template, error)
func (m *Manager) GetByType(templateType Type) []*Template
func (m *Manager) GetByTag(tag string) []*Template
func (m *Manager) GetAll() []*Template
```

**Example:**
```go
tpl, err := mgr.GetByName("Function")
codeTpls := mgr.GetByType(template.TypeCode)
```

#### Render
```go
func (m *Manager) Render(templateID string, vars map[string]interface{}) (string, error)
func (m *Manager) RenderByName(name string, vars map[string]interface{}) (string, error)
```

**Example:**
```go
result, err := mgr.RenderByName("Function", map[string]interface{}{
    "function_name": "ProcessData",
    "parameters":    "data []byte",
    "return_type":   "error",
    "body":          "return nil",
})
```

#### Update and Delete
```go
func (m *Manager) Update(id string, updater func(*Template)) error
func (m *Manager) Delete(id string) error
```

**Example:**
```go
mgr.Update(tpl.ID, func(t *template.Template) {
    t.Version = "2.0.0"
    t.AddTag("updated")
})

mgr.Delete(tpl.ID)
```

#### Search
```go
func (m *Manager) Search(query string) []*Template
func (m *Manager) Count() int
func (m *Manager) CountByType() map[Type]int
```

**Example:**
```go
results := mgr.Search("handler")
total := mgr.Count()
byType := mgr.CountByType()
```

#### File Operations
```go
func (m *Manager) LoadFromFile(filename string) error
func (m *Manager) LoadFromDirectory(dirPath string) (int, error)
func (m *Manager) SaveToFile(templateID, filename string) error
```

**Example:**
```go
// Load from file
mgr.LoadFromFile("./templates/handler.json")

// Load directory
count, _ := mgr.LoadFromDirectory("./templates")

// Save to file
mgr.SaveToFile(tpl.ID, "./templates/my-template.json")
```

#### Export/Import
```go
func (m *Manager) Export(templateID string) (*TemplateSnapshot, error)
func (m *Manager) Import(snapshot *TemplateSnapshot) error
```

**Example:**
```go
// Export
snapshot, err := mgr.Export(tpl.ID)
data, _ := json.Marshal(snapshot)

// Import
var snapshot template.TemplateSnapshot
json.Unmarshal(data, &snapshot)
mgr.Import(&snapshot)
```

#### Built-in Templates
```go
func (m *Manager) RegisterBuiltinTemplates() error
```

Registers 5 built-in templates: Function, Code Review, Bug Fix, Function Documentation, Status Update Email.

**Example:**
```go
if err := mgr.RegisterBuiltinTemplates(); err != nil {
    log.Fatal(err)
}
```

#### Callbacks
```go
func (m *Manager) OnCreate(callback TemplateCallback)
func (m *Manager) OnUpdate(callback TemplateCallback)
func (m *Manager) OnDelete(callback TemplateCallback)
```

#### Statistics
```go
func (m *Manager) GetStatistics() *ManagerStatistics
```

**Example:**
```go
stats := mgr.GetStatistics()
fmt.Printf("Total templates: %d\n", stats.TotalTemplates)
fmt.Printf("By type: %v\n", stats.ByType)
fmt.Printf("Tag cloud: %v\n", stats.TagCloud)
```

---

## Error Handling

All methods that can fail return an `error` type. Always check errors:

```go
sess, err := sessionMgr.Create("name", session.ModeBuilding, "project")
if err != nil {
    log.Fatalf("Failed to create session: %v", err)
}
```

Common error types:
- `ErrNotFound` - Resource not found
- `ErrValidation` - Validation failed
- `ErrConflict` - Duplicate or conflicting resource
- `ErrState` - Invalid state transition

---

## Thread Safety

All managers are thread-safe and can be used concurrently:
- Protected with `sync.RWMutex`
- Safe for concurrent reads
- Safe for concurrent writes
- No external synchronization needed

---

## Best Practices

1. **Always check errors** - Never ignore error return values
2. **Use contexts** - Store relevant data in metadata and context fields
3. **Set limits** - Configure max messages/tokens to prevent unbounded growth
4. **Enable auto-save** - Prevent data loss with automatic persistence
5. **Clean up** - Delete or archive old sessions and conversations
6. **Use callbacks** - Monitor events for logging and analytics
7. **Tag everything** - Makes searching and filtering easier
8. **Export important work** - Backup sessions, conversations, and templates

---

## Examples

See `/examples/phase3/` for complete working examples:
- Basic usage
- Feature development workflow
- Code review automation
- Debugging workflow
- Template library creation

---

**For more information, see the [Phase 3 Features Guide](./PHASE_3_FEATURES.md)**
