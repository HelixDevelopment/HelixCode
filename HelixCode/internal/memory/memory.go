package memory

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

// Role represents the role of a message sender
type Role string

const (
	RoleUser      Role = "user"      // User message
	RoleAssistant Role = "assistant" // AI assistant response
	RoleSystem    Role = "system"    // System message
	RoleTool      Role = "tool"      // Tool/function result
)

// IsValid checks if role is valid
func (r Role) IsValid() bool {
	switch r {
	case RoleUser, RoleAssistant, RoleSystem, RoleTool:
		return true
	}
	return false
}

// String returns string representation
func (r Role) String() string {
	return string(r)
}

// Message represents a single message in conversation history
type Message struct {
	ID         string            // Unique message ID
	Role       Role              // Message role
	Content    string            // Message content
	Timestamp  time.Time         // When message was created
	Metadata   map[string]string // Additional metadata
	TokenCount int               // Approximate token count
	Size       int               // Size in bytes
}

// NewMessage creates a new message
func NewMessage(role Role, content string) *Message {
	return &Message{
		ID:         generateMessageID(),
		Role:       role,
		Content:    content,
		Timestamp:  time.Now(),
		Metadata:   make(map[string]string),
		TokenCount: estimateTokens(content),
		Size:       len(content),
	}
}

// NewUserMessage creates a new user message
func NewUserMessage(content string) *Message {
	return NewMessage(RoleUser, content)
}

// NewAssistantMessage creates a new assistant message
func NewAssistantMessage(content string) *Message {
	return NewMessage(RoleAssistant, content)
}

// NewSystemMessage creates a new system message
func NewSystemMessage(content string) *Message {
	return NewMessage(RoleSystem, content)
}

// SetMetadata sets a metadata value
func (m *Message) SetMetadata(key, value string) {
	m.Metadata[key] = value
}

// GetMetadata gets a metadata value
func (m *Message) GetMetadata(key string) (string, bool) {
	value, ok := m.Metadata[key]
	return value, ok
}

// Clone creates a copy of the message
func (m *Message) Clone() *Message {
	clone := &Message{
		ID:         m.ID,
		Role:       m.Role,
		Content:    m.Content,
		Timestamp:  m.Timestamp,
		Metadata:   make(map[string]string),
		TokenCount: m.TokenCount,
		Size:       m.Size,
	}

	for k, v := range m.Metadata {
		clone.Metadata[k] = v
	}

	return clone
}

// String returns a string representation
func (m *Message) String() string {
	return fmt.Sprintf("[%s] %s: %s", m.Timestamp.Format("15:04:05"), m.Role, truncate(m.Content, 50))
}

// Validate validates the message
func (m *Message) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("message ID cannot be empty")
	}

	if !m.Role.IsValid() {
		return fmt.Errorf("invalid role: %s", m.Role)
	}

	if m.Content == "" {
		return fmt.Errorf("message content cannot be empty")
	}

	return nil
}

// Conversation represents a conversation with message history
type Conversation struct {
	ID           string              // Unique conversation ID
	Title        string              // Conversation title
	SessionID    string              // Associated session ID
	CharacterID  string              // Character ID (for character AI conversations)
	UserID       string              // User ID (for character AI conversations)
	Messages     []*Message          // Conversation messages
	CharMessages []*CharacterMessage // Character messages (alternative to Messages)
	Metadata     map[string]string   // Additional metadata
	CreatedAt    time.Time           // When created
	UpdatedAt    time.Time           // Last updated
	Status       string              // Conversation status
	Summary      string              // Conversation summary
	TokenCount   int                 // Total tokens
	MessageCount int                 // Total messages
}

// NewConversation creates a new conversation
func NewConversation(title string) *Conversation {
	return &Conversation{
		ID:           generateConversationID(),
		Title:        title,
		Messages:     make([]*Message, 0),
		Metadata:     make(map[string]string),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		TokenCount:   0,
		MessageCount: 0,
	}
}

// AddMessage adds a message to the conversation
func (c *Conversation) AddMessage(message *Message) {
	c.Messages = append(c.Messages, message)
	c.TokenCount += message.TokenCount
	c.MessageCount++
	c.UpdatedAt = time.Now()
}

// GetMessages returns all messages
func (c *Conversation) GetMessages() []*Message {
	messages := make([]*Message, len(c.Messages))
	copy(messages, c.Messages)
	return messages
}

// GetMessagesByRole returns messages with specific role
func (c *Conversation) GetMessagesByRole(role Role) []*Message {
	messages := make([]*Message, 0)
	for _, msg := range c.Messages {
		if msg.Role == role {
			messages = append(messages, msg)
		}
	}
	return messages
}

// GetRecent returns the N most recent messages
func (c *Conversation) GetRecent(n int) []*Message {
	if n <= 0 || n > len(c.Messages) {
		n = len(c.Messages)
	}

	start := len(c.Messages) - n
	messages := make([]*Message, n)
	copy(messages, c.Messages[start:])
	return messages
}

// GetRange returns messages in a range
func (c *Conversation) GetRange(start, end int) []*Message {
	if start < 0 {
		start = 0
	}
	if end > len(c.Messages) {
		end = len(c.Messages)
	}
	if start >= end {
		return []*Message{}
	}

	messages := make([]*Message, end-start)
	copy(messages, c.Messages[start:end])
	return messages
}

// Search searches for messages containing text
func (c *Conversation) Search(query string) []*Message {
	query = strings.ToLower(query)
	messages := make([]*Message, 0)

	for _, msg := range c.Messages {
		if strings.Contains(strings.ToLower(msg.Content), query) {
			messages = append(messages, msg)
		}
	}

	return messages
}

// Clear clears all messages
func (c *Conversation) Clear() {
	c.Messages = make([]*Message, 0)
	c.TokenCount = 0
	c.MessageCount = 0
	c.UpdatedAt = time.Now()
}

// Truncate keeps only the last N messages
func (c *Conversation) Truncate(keepLast int) int {
	if keepLast <= 0 || keepLast >= len(c.Messages) {
		return 0
	}

	removed := len(c.Messages) - keepLast
	c.Messages = c.Messages[len(c.Messages)-keepLast:]

	// Recalculate token count
	c.TokenCount = 0
	for _, msg := range c.Messages {
		c.TokenCount += msg.TokenCount
	}

	c.MessageCount = len(c.Messages)
	c.UpdatedAt = time.Now()

	return removed
}

// SetMetadata sets a metadata value
func (c *Conversation) SetMetadata(key, value string) {
	c.Metadata[key] = value
}

// GetMetadata gets a metadata value
func (c *Conversation) GetMetadata(key string) (string, bool) {
	value, ok := c.Metadata[key]
	return value, ok
}

// Clone creates a copy of the conversation
func (c *Conversation) Clone() *Conversation {
	clone := &Conversation{
		ID:           c.ID,
		Title:        c.Title,
		SessionID:    c.SessionID,
		Messages:     make([]*Message, len(c.Messages)),
		Metadata:     make(map[string]string),
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
		Summary:      c.Summary,
		TokenCount:   c.TokenCount,
		MessageCount: c.MessageCount,
	}

	// Deep copy messages
	for i, msg := range c.Messages {
		clone.Messages[i] = msg.Clone()
	}

	// Copy metadata
	for k, v := range c.Metadata {
		clone.Metadata[k] = v
	}

	return clone
}

// String returns a string representation
func (c *Conversation) String() string {
	return fmt.Sprintf("Conversation %s: %s (%d messages, %d tokens)",
		c.ID, c.Title, c.MessageCount, c.TokenCount)
}

// Validate validates the conversation
func (c *Conversation) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("conversation ID cannot be empty")
	}

	if c.Title == "" {
		return fmt.Errorf("conversation title cannot be empty")
	}

	return nil
}

// ToText converts conversation to text format
func (c *Conversation) ToText() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Conversation: %s\n", c.Title))
	builder.WriteString(fmt.Sprintf("Created: %s\n", c.CreatedAt.Format("2006-01-02 15:04:05")))
	builder.WriteString(fmt.Sprintf("Messages: %d\n\n", c.MessageCount))

	for _, msg := range c.Messages {
		builder.WriteString(fmt.Sprintf("[%s] %s:\n%s\n\n",
			msg.Timestamp.Format("15:04:05"),
			msg.Role,
			msg.Content))
	}

	return builder.String()
}

// Statistics contains conversation statistics
type Statistics struct {
	TotalMessages int          // Total messages
	ByRole        map[Role]int // Count by role
	TotalTokens   int          // Total tokens
	AverageTokens float64      // Average tokens per message
	TotalSize     int          // Total size in bytes
	OldestMessage time.Time    // Oldest message timestamp
	NewestMessage time.Time    // Newest message timestamp
}

// GetStatistics returns conversation statistics
func (c *Conversation) GetStatistics() *Statistics {
	stats := &Statistics{
		TotalMessages: len(c.Messages),
		ByRole:        make(map[Role]int),
		TotalTokens:   c.TokenCount,
		TotalSize:     0,
	}

	if len(c.Messages) == 0 {
		return stats
	}

	for _, msg := range c.Messages {
		stats.ByRole[msg.Role]++
		stats.TotalSize += msg.Size
	}

	stats.AverageTokens = float64(stats.TotalTokens) / float64(stats.TotalMessages)
	stats.OldestMessage = c.Messages[0].Timestamp
	stats.NewestMessage = c.Messages[len(c.Messages)-1].Timestamp

	return stats
}

// Counters for unique ID generation
var (
	messageCounter      uint64
	conversationCounter uint64
	characterCounter    uint64
)

// generateMessageID generates a unique message ID
func generateMessageID() string {
	count := atomic.AddUint64(&messageCounter, 1)
	return fmt.Sprintf("msg-%d-%d", time.Now().UnixNano(), count)
}

// generateConversationID generates a unique conversation ID
func generateConversationID() string {
	count := atomic.AddUint64(&conversationCounter, 1)
	return fmt.Sprintf("conv-%d-%d", time.Now().UnixNano(), count)
}

// estimateTokens estimates the number of tokens in text
// Rough approximation: 1 token ~= 4 characters
func estimateTokens(text string) int {
	return len(text) / 4
}

// truncate truncates a string to max length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ============================================================================
// Specialized Memory Types for Advanced Providers
// ============================================================================

// Avatar represents an AI persona/avatar in the Anima system
type Avatar struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Personality *Personality      `json:"personality,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// NewAvatar creates a new avatar
func NewAvatar(name, description string) *Avatar {
	return &Avatar{
		ID:          generateAvatarID(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    make(map[string]string),
	}
}

// Activity represents an activity or action in the Anima system
type Activity struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Description string            `json:"description"`
	AvatarID    string            `json:"avatar_id"`
	Timestamp   time.Time         `json:"timestamp"`
	Duration    time.Duration     `json:"duration"`
	Status      string            `json:"status"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// NewActivity creates a new activity
func NewActivity(activityType, description, avatarID string) *Activity {
	return &Activity{
		ID:          generateActivityID(),
		Type:        activityType,
		Description: description,
		AvatarID:    avatarID,
		Timestamp:   time.Now(),
		Status:      "active",
		Metadata:    make(map[string]string),
	}
}

// EmotionalState represents emotional state data
type EmotionalState struct {
	ID        string    `json:"id"`
	AvatarID  string    `json:"avatar_id"`
	Mood      string    `json:"mood"`
	Intensity float64   `json:"intensity"`
	Timestamp time.Time `json:"timestamp"`
	Context   string    `json:"context,omitempty"`
}

// NewEmotionalState creates a new emotional state
func NewEmotionalState(avatarID, mood string, intensity float64) *EmotionalState {
	return &EmotionalState{
		ID:        generateEmotionalStateID(),
		AvatarID:  avatarID,
		Mood:      mood,
		Intensity: intensity,
		Timestamp: time.Now(),
	}
}

// Personality represents personality traits and characteristics
type Personality struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Traits      map[string]float64 `json:"traits"` // trait name -> intensity (0-1)
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// NewPersonality creates a new personality
func NewPersonality(name, description string) *Personality {
	return &Personality{
		ID:          generatePersonalityID(),
		Name:        name,
		Description: description,
		Traits:      make(map[string]float64),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// ConversationSession represents a conversation session for character AI
type ConversationSession struct {
	ID          string            `json:"id"`
	CharacterID string            `json:"character_id"`
	UserID      string            `json:"user_id"`
	StartedAt   time.Time         `json:"started_at"`
	EndedAt     *time.Time        `json:"ended_at,omitempty"`
	IsActive    bool              `json:"is_active"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// NewConversationSession creates a new conversation session
func NewConversationSession(characterID, userID string) *ConversationSession {
	return &ConversationSession{
		ID:          generateSessionID(),
		CharacterID: characterID,
		UserID:      userID,
		StartedAt:   time.Now(),
		IsActive:    true,
		Metadata:    make(map[string]string),
	}
}

// CharacterMessage represents a message in character AI conversation
type CharacterMessage struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	SenderID  string    `json:"sender_id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "user", "character", "system"
}

// NewCharacterMessage creates a new character message
func NewCharacterMessage(sessionID, senderID, content, msgType string) *CharacterMessage {
	return &CharacterMessage{
		ID:        generateCharacterMessageID(),
		SessionID: sessionID,
		SenderID:  senderID,
		Content:   content,
		Timestamp: time.Now(),
		Type:      msgType,
	}
}

// RelationshipData represents relationship information between characters/users
type RelationshipData struct {
	ID          string            `json:"id"`
	CharacterID string            `json:"character_id"`
	UserID      string            `json:"user_id"`
	Type        string            `json:"type"`
	Strength    float64           `json:"strength"` // 0-1
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// NewRelationshipData creates new relationship data
func NewRelationshipData(characterID, userID, relType string, strength float64) *RelationshipData {
	return &RelationshipData{
		ID:          generateRelationshipID(),
		CharacterID: characterID,
		UserID:      userID,
		Type:        relType,
		Strength:    strength,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    make(map[string]string),
	}
}

// ActivityPattern represents patterns in user/character activity
type ActivityPattern struct {
	ID          string    `json:"id"`
	CharacterID string    `json:"character_id"`
	Pattern     string    `json:"pattern"`
	Frequency   int       `json:"frequency"`
	LastSeen    time.Time `json:"last_seen"`
	Confidence  float64   `json:"confidence"`
}

// NewActivityPattern creates a new activity pattern
func NewActivityPattern(characterID, pattern string, frequency int, confidence float64) *ActivityPattern {
	return &ActivityPattern{
		ID:          generateActivityPatternID(),
		CharacterID: characterID,
		Pattern:     pattern,
		Frequency:   frequency,
		LastSeen:    time.Now(),
		Confidence:  confidence,
	}
}

// MoodData represents mood tracking data
type MoodData struct {
	ID          string    `json:"id"`
	CharacterID string    `json:"character_id"`
	Mood        string    `json:"mood"`
	Intensity   float64   `json:"intensity"`
	Context     string    `json:"context,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// NewMoodData creates new mood data
func NewMoodData(characterID, mood string, intensity float64, context string) *MoodData {
	return &MoodData{
		ID:          generateMoodDataID(),
		CharacterID: characterID,
		Mood:        mood,
		Intensity:   intensity,
		Context:     context,
		Timestamp:   time.Now(),
	}
}

// Crew represents a crew/team in CrewAI system
type Crew struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Agents      []*Agent          `json:"agents,omitempty"`
	Tasks       []*Task           `json:"tasks,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Status      string            `json:"status"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// NewCrew creates a new crew
func NewCrew(name, description string) *Crew {
	return &Crew{
		ID:          generateCrewID(),
		Name:        name,
		Description: description,
		Agents:      make([]*Agent, 0),
		Tasks:       make([]*Task, 0),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Status:      "active",
		Metadata:    make(map[string]string),
	}
}

// Agent represents an agent in CrewAI system
type Agent struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Role      string            `json:"role"`
	Goal      string            `json:"goal"`
	Backstory string            `json:"backstory"`
	CrewID    string            `json:"crew_id"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Status    string            `json:"status"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// NewAgent creates a new agent
func NewAgent(name, role, goal, backstory, crewID string) *Agent {
	return &Agent{
		ID:        generateAgentID(),
		Name:      name,
		Role:      role,
		Goal:      goal,
		Backstory: backstory,
		CrewID:    crewID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    "active",
		Metadata:  make(map[string]string),
	}
}

// Task represents a task in CrewAI system
type Task struct {
	ID          string            `json:"id"`
	Description string            `json:"description"`
	AgentID     string            `json:"agent_id"`
	CrewID      string            `json:"crew_id"`
	Status      string            `json:"status"`
	Priority    int               `json:"priority"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Deadline    *time.Time        `json:"deadline,omitempty"`
	Result      *TaskResult       `json:"result,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// NewTask creates a new task
func NewTask(description, agentID, crewID string, priority int) *Task {
	return &Task{
		ID:          generateTaskID(),
		Description: description,
		AgentID:     agentID,
		CrewID:      crewID,
		Status:      "pending",
		Priority:    priority,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    make(map[string]string),
	}
}

// TaskResult represents the result of a task execution
type TaskResult struct {
	ID          string            `json:"id"`
	TaskID      string            `json:"task_id"`
	Output      string            `json:"output"`
	Status      string            `json:"status"`
	CompletedAt time.Time         `json:"completed_at"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// NewTaskResult creates a new task result
func NewTaskResult(taskID, output, status string) *TaskResult {
	return &TaskResult{
		ID:          generateTaskResultID(),
		TaskID:      taskID,
		Output:      output,
		Status:      status,
		CompletedAt: time.Now(),
		Metadata:    make(map[string]string),
	}
}

// SharedMemory represents shared memory in CrewAI system
type SharedMemory struct {
	ID        string            `json:"id"`
	CrewID    string            `json:"crew_id"`
	Key       string            `json:"key"`
	Value     interface{}       `json:"value"`
	Type      string            `json:"type"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// NewSharedMemory creates new shared memory
func NewSharedMemory(crewID, key string, value interface{}, memType string) *SharedMemory {
	return &SharedMemory{
		ID:        generateSharedMemoryID(),
		CrewID:    crewID,
		Key:       key,
		Value:     value,
		Type:      memType,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]string),
	}
}

// CrewPerformance represents crew performance metrics
type CrewPerformance struct {
	ID             string        `json:"id"`
	CrewID         string        `json:"crew_id"`
	TasksCompleted int           `json:"tasks_completed"`
	TasksFailed    int           `json:"tasks_failed"`
	AvgTaskTime    time.Duration `json:"avg_task_time"`
	Efficiency     float64       `json:"efficiency"`
	Timestamp      time.Time     `json:"timestamp"`
}

// NewCrewPerformance creates new crew performance data
func NewCrewPerformance(crewID string, tasksCompleted, tasksFailed int, avgTaskTime time.Duration, efficiency float64) *CrewPerformance {
	return &CrewPerformance{
		ID:             generateCrewPerformanceID(),
		CrewID:         crewID,
		TasksCompleted: tasksCompleted,
		TasksFailed:    tasksFailed,
		AvgTaskTime:    avgTaskTime,
		Efficiency:     efficiency,
		Timestamp:      time.Now(),
	}
}

// MemoryBlock represents a block of memory in MemGPT system
type MemoryBlock struct {
	ID           string            `json:"id"`
	Type         string            `json:"type"`
	Content      string            `json:"content"`
	Importance   float64           `json:"importance"`
	AccessCount  int               `json:"access_count"`
	LastAccessed time.Time         `json:"last_accessed"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// NewMemoryBlock creates a new memory block
func NewMemoryBlock(blockType, content string, importance float64) *MemoryBlock {
	return &MemoryBlock{
		ID:           generateMemoryBlockID(),
		Type:         blockType,
		Content:      content,
		Importance:   importance,
		AccessCount:  0,
		LastAccessed: time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Metadata:     make(map[string]string),
	}
}

// WorkingMemory represents working memory in MemGPT system
type WorkingMemory struct {
	ID        string            `json:"id"`
	Blocks    []*MemoryBlock    `json:"blocks"`
	Capacity  int               `json:"capacity"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// NewWorkingMemory creates new working memory
func NewWorkingMemory(capacity int) *WorkingMemory {
	return &WorkingMemory{
		ID:        generateWorkingMemoryID(),
		Blocks:    make([]*MemoryBlock, 0),
		Capacity:  capacity,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]string),
	}
}

// ProcessingResult represents the result of memory processing
type ProcessingResult struct {
	ID           string            `json:"id"`
	Input        string            `json:"input"`
	Output       string            `json:"output"`
	MemoryBlocks []*MemoryBlock    `json:"memory_blocks,omitempty"`
	ProcessedAt  time.Time         `json:"processed_at"`
	Duration     time.Duration     `json:"duration"`
	Status       string            `json:"status"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// NewProcessingResult creates a new processing result
func NewProcessingResult(input, output, status string, duration time.Duration) *ProcessingResult {
	return &ProcessingResult{
		ID:           generateProcessingResultID(),
		Input:        input,
		Output:       output,
		MemoryBlocks: make([]*MemoryBlock, 0),
		ProcessedAt:  time.Now(),
		Duration:     duration,
		Status:       status,
		Metadata:     make(map[string]string),
	}
}

// SearchQuery represents a search query for memory retrieval
type SearchQuery struct {
	ID        string            `json:"id"`
	Query     string            `json:"query"`
	Type      string            `json:"type"`
	Limit     int               `json:"limit"`
	Threshold float64           `json:"threshold"`
	CreatedAt time.Time         `json:"created_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// NewSearchQuery creates a new search query
func NewSearchQuery(query, queryType string, limit int, threshold float64) *SearchQuery {
	return &SearchQuery{
		ID:        generateSearchQueryID(),
		Query:     query,
		Type:      queryType,
		Limit:     limit,
		Threshold: threshold,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]string),
	}
}

// SearchResult represents search results
type SearchResult struct {
	Query      *SearchQuery  `json:"query"`
	Results    []*MemoryItem `json:"results"`
	Total      int           `json:"total"`
	Duration   time.Duration `json:"duration"`
	SearchedAt time.Time     `json:"searched_at"`
}

// NewSearchResult creates a new search result
func NewSearchResult(query *SearchQuery, results []*MemoryItem, total int, duration time.Duration) *SearchResult {
	return &SearchResult{
		Query:      query,
		Results:    results,
		Total:      total,
		Duration:   duration,
		SearchedAt: time.Now(),
	}
}

// MemoryItem represents an item in memory search results
type MemoryItem struct {
	ID        string            `json:"id"`
	Content   string            `json:"content"`
	Score     float64           `json:"score"`
	Type      string            `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// NewMemoryItem creates a new memory item
func NewMemoryItem(id, content, itemType string, score float64, timestamp time.Time) *MemoryItem {
	return &MemoryItem{
		ID:        id,
		Content:   content,
		Score:     score,
		Type:      itemType,
		Timestamp: timestamp,
		Metadata:  make(map[string]string),
	}
}

// ConversationSummary represents a summary of a conversation
type ConversationSummary struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Summary        string    `json:"summary"`
	KeyPoints      []string  `json:"key_points"`
	Topics         []string  `json:"topics"`
	Sentiment      string    `json:"sentiment"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// NewConversationSummary creates a new conversation summary
func NewConversationSummary(conversationID, summary, sentiment string) *ConversationSummary {
	return &ConversationSummary{
		ID:             generateConversationSummaryID(),
		ConversationID: conversationID,
		Summary:        summary,
		KeyPoints:      make([]string, 0),
		Topics:         make([]string, 0),
		Sentiment:      sentiment,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// ContextData represents contextual information
type ContextData struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Data      interface{}       `json:"data"`
	Source    string            `json:"source"`
	Timestamp time.Time         `json:"timestamp"`
	Relevance float64           `json:"relevance"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// NewContextData creates new context data
func NewContextData(dataType string, data interface{}, source string, relevance float64) *ContextData {
	return &ContextData{
		ID:        generateContextDataID(),
		Type:      dataType,
		Data:      data,
		Source:    source,
		Timestamp: time.Now(),
		Relevance: relevance,
		Metadata:  make(map[string]string),
	}
}

// RetrievalQuery represents a query for memory retrieval
type RetrievalQuery struct {
	ID        string                 `json:"id"`
	Query     string                 `json:"query"`
	Type      string                 `json:"type"`
	Limit     int                    `json:"limit"`
	Filters   map[string]interface{} `json:"filters,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
}

// NewRetrievalQuery creates a new retrieval query
func NewRetrievalQuery(query, queryType string, limit int) *RetrievalQuery {
	return &RetrievalQuery{
		ID:        generateRetrievalQueryID(),
		Query:     query,
		Type:      queryType,
		Limit:     limit,
		Filters:   make(map[string]interface{}),
		CreatedAt: time.Now(),
		Metadata:  make(map[string]string),
	}
}

// RetrievalResult represents retrieval results
type RetrievalResult struct {
	Query       *RetrievalQuery `json:"query"`
	Results     []*MemoryItem   `json:"results"`
	Total       int             `json:"total"`
	Duration    time.Duration   `json:"duration"`
	RetrievedAt time.Time       `json:"retrieved_at"`
}

// NewRetrievalResult creates a new retrieval result
func NewRetrievalResult(query *RetrievalQuery, results []*MemoryItem, total int, duration time.Duration) *RetrievalResult {
	return &RetrievalResult{
		Query:       query,
		Results:     results,
		Total:       total,
		Duration:    duration,
		RetrievedAt: time.Now(),
	}
}

// Character represents an AI character in character AI systems
type Character struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Personality map[string]interface{} `json:"personality"`
	Traits      map[string]interface{} `json:"traits,omitempty"`
	Appearance  map[string]interface{} `json:"appearance,omitempty"`
	Backstory   string                 `json:"backstory,omitempty"`
	Avatar      string                 `json:"avatar,omitempty"`
	IsPublic    bool                   `json:"is_public"`
	IsActive    bool                   `json:"is_active"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Status      string                 `json:"status"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
}

// NewCharacter creates a new character
func NewCharacter(name, description string, personality map[string]interface{}) *Character {
	return &Character{
		ID:          generateCharacterID(),
		Name:        name,
		Description: description,
		Personality: personality,
		Traits:      make(map[string]interface{}),
		Appearance:  make(map[string]interface{}),
		IsPublic:    false,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Status:      "active",
		Metadata:    make(map[string]string),
	}
}

// SystemInfo represents system information
type SystemInfo struct {
	ID        string                 `json:"id"`
	Component string                 `json:"component"`
	Version   string                 `json:"version"`
	Status    string                 `json:"status"`
	LastCheck time.Time              `json:"last_check"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
}

// NewSystemInfo creates new system info
func NewSystemInfo(component, version, status string) *SystemInfo {
	return &SystemInfo{
		ID:        generateSystemInfoID(),
		Component: component,
		Version:   version,
		Status:    status,
		LastCheck: time.Now(),
		Details:   make(map[string]interface{}),
		Metadata:  make(map[string]string),
	}
}

// OptimizationRecommendation represents optimization recommendations
type OptimizationRecommendation struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Description string            `json:"description"`
	Priority    string            `json:"priority"`
	Impact      float64           `json:"impact"`
	CreatedAt   time.Time         `json:"created_at"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// NewOptimizationRecommendation creates a new optimization recommendation
func NewOptimizationRecommendation(recType, description, priority string, impact float64) *OptimizationRecommendation {
	return &OptimizationRecommendation{
		ID:          generateOptimizationRecommendationID(),
		Type:        recType,
		Description: description,
		Priority:    priority,
		Impact:      impact,
		CreatedAt:   time.Now(),
		Metadata:    make(map[string]string),
	}
}

// ============================================================================
// LLM Model Types (For AI Memory Providers)
// ============================================================================

// Model represents an AI model in memory systems
type Model struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Version   string            `json:"version"`
	Provider  string            `json:"provider"`
	Status    string            `json:"status"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// NewModel creates a new model
func NewModel(name, modelType, provider string) *Model {
	return &Model{
		ID:        generateModelID(),
		Name:      name,
		Type:      modelType,
		Provider:  provider,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]string),
	}
}

// Embedding represents a vector embedding
type Embedding struct {
	ID        string            `json:"id"`
	ModelID   string            `json:"model_id"`
	Text      string            `json:"text"`
	Vector    []float64         `json:"vector"`
	CreatedAt time.Time         `json:"created_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// NewEmbedding creates a new embedding
func NewEmbedding(modelID, text string, vector []float64) *Embedding {
	return &Embedding{
		ID:        generateEmbeddingID(),
		ModelID:   modelID,
		Text:      text,
		Vector:    vector,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]string),
	}
}

// ModelPerformance represents model performance metrics
type ModelPerformance struct {
	ID              string            `json:"id"`
	ModelID         string            `json:"model_id"`
	ResponseTime    time.Duration     `json:"response_time"`
	TokensPerSecond float64           `json:"tokens_per_second"`
	MemoryUsage     int64             `json:"memory_usage"`
	Accuracy        float64           `json:"accuracy"`
	Timestamp       time.Time         `json:"timestamp"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// NewModelPerformance creates new model performance data
func NewModelPerformance(modelID string, responseTime time.Duration, tokensPerSecond float64, memoryUsage int64, accuracy float64) *ModelPerformance {
	return &ModelPerformance{
		ID:              generateModelPerformanceID(),
		ModelID:         modelID,
		ResponseTime:    responseTime,
		TokensPerSecond: tokensPerSecond,
		MemoryUsage:     memoryUsage,
		Accuracy:        accuracy,
		Timestamp:       time.Now(),
		Metadata:        make(map[string]string),
	}
}

// GenerationOptions represents options for text generation
type GenerationOptions struct {
	MaxTokens        int                `json:"max_tokens"`
	Temperature      float64            `json:"temperature"`
	TopP             float64            `json:"top_p"`
	FrequencyPenalty float64            `json:"frequency_penalty"`
	PresencePenalty  float64            `json:"presence_penalty"`
	Stop             []string           `json:"stop"`
	Stream           bool               `json:"stream"`
	Callback         func(string) error `json:"-"`
}

// PersonalityMessage represents a message with personality traits
type PersonalityMessage struct {
	ID          string             `json:"id"`
	Personality *Personality       `json:"personality"`
	Content     string             `json:"content"`
	Timestamp   time.Time          `json:"timestamp"`
	Traits      map[string]float64 `json:"traits"`
	Metadata    map[string]string  `json:"metadata,omitempty"`
}

// NewPersonalityMessage creates a new personality message
func NewPersonalityMessage(personality *Personality, content string) *PersonalityMessage {
	return &PersonalityMessage{
		ID:          generatePersonalityMessageID(),
		Personality: personality,
		Content:     content,
		Timestamp:   time.Now(),
		Traits:      make(map[string]float64),
		Metadata:    make(map[string]string),
	}
}

// ============================================================================
// Vector Database Types (Shared across providers)
// ============================================================================

// VectorData represents a vector entry
type VectorData struct {
	ID         string                 `json:"id"`
	Vector     []float64              `json:"vector"`
	Metadata   map[string]interface{} `json:"metadata"`
	Collection string                 `json:"collection"`
	Timestamp  time.Time              `json:"timestamp"`
	TTL        *time.Duration         `json:"ttl,omitempty"`
	Namespace  string                 `json:"namespace,omitempty"`
}

// NewVectorData creates a new vector data entry
func NewVectorData(id string, vector []float64, metadata map[string]interface{}, collection string) *VectorData {
	return &VectorData{
		ID:         id,
		Vector:     vector,
		Metadata:   metadata,
		Collection: collection,
		Timestamp:  time.Now(),
	}
}

// VectorQuery represents a vector search query
type VectorQuery struct {
	Vector        []float64              `json:"vector"`
	Text          string                 `json:"text,omitempty"`
	Collection    string                 `json:"collection"`
	Namespace     string                 `json:"namespace,omitempty"`
	TopK          int                    `json:"top_k"`
	Threshold     float64                `json:"threshold,omitempty"`
	IncludeVector bool                   `json:"include_vector"`
	Filters       map[string]interface{} `json:"filters,omitempty"`
}

// NewVectorQuery creates a new vector query
func NewVectorQuery(vector []float64, collection string, topK int) *VectorQuery {
	return &VectorQuery{
		Vector:        vector,
		Collection:    collection,
		TopK:          topK,
		IncludeVector: false,
		Filters:       make(map[string]interface{}),
	}
}

// VectorSearchResult represents the result of a vector search
type VectorSearchResult struct {
	Results   []*VectorSearchResultItem `json:"results"`
	Total     int                       `json:"total"`
	Query     *VectorQuery              `json:"query"`
	Duration  time.Duration             `json:"duration"`
	Namespace string                    `json:"namespace"`
}

// NewVectorSearchResult creates a new vector search result
func NewVectorSearchResult(results []*VectorSearchResultItem, total int, query *VectorQuery, duration time.Duration) *VectorSearchResult {
	return &VectorSearchResult{
		Results:  results,
		Total:    total,
		Query:    query,
		Duration: duration,
	}
}

// VectorSearchResultItem represents a single search result
type VectorSearchResultItem struct {
	ID       string                 `json:"id"`
	Vector   []float64              `json:"vector,omitempty"`
	Metadata map[string]interface{} `json:"metadata"`
	Score    float64                `json:"score"`
	Distance float64                `json:"distance"`
}

// NewVectorSearchResultItem creates a new search result item
func NewVectorSearchResultItem(id string, score, distance float64, metadata map[string]interface{}) *VectorSearchResultItem {
	return &VectorSearchResultItem{
		ID:       id,
		Score:    score,
		Distance: distance,
		Metadata: metadata,
	}
}

// VectorSimilarityResult represents similarity search result
type VectorSimilarityResult struct {
	ID       string                 `json:"id"`
	Vector   []float64              `json:"vector,omitempty"`
	Metadata map[string]interface{} `json:"metadata"`
	Score    float64                `json:"score"`
	Distance float64                `json:"distance"`
}

// NewVectorSimilarityResult creates a new similarity result
func NewVectorSimilarityResult(id string, score, distance float64, metadata map[string]interface{}) *VectorSimilarityResult {
	return &VectorSimilarityResult{
		ID:       id,
		Score:    score,
		Distance: distance,
		Metadata: metadata,
	}
}

// CollectionConfig represents collection configuration
type CollectionConfig struct {
	Name       string                 `json:"name"`
	Dimension  int                    `json:"dimension"`
	Metric     string                 `json:"metric"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Replicas   int                    `json:"replicas,omitempty"`
	Shards     int                    `json:"shards,omitempty"`
}

// NewCollectionConfig creates a new collection configuration
func NewCollectionConfig(name string, dimension int, metric string) *CollectionConfig {
	return &CollectionConfig{
		Name:       name,
		Dimension:  dimension,
		Metric:     metric,
		Properties: make(map[string]interface{}),
		Replicas:   1,
		Shards:     1,
	}
}

// CollectionInfo represents collection information
type CollectionInfo struct {
	Name         string                 `json:"name"`
	Dimension    int                    `json:"dimension"`
	Metric       string                 `json:"metric"`
	VectorsCount int64                  `json:"vectors_count"`
	Status       string                 `json:"status"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Config       *CollectionConfig      `json:"config,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// NewCollectionInfo creates new collection info
func NewCollectionInfo(name string, dimension int, metric string, vectorsCount int64, status string) *CollectionInfo {
	return &CollectionInfo{
		Name:         name,
		Dimension:    dimension,
		Metric:       metric,
		VectorsCount: vectorsCount,
		Status:       status,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Config:       nil,
		Metadata:     make(map[string]interface{}),
	}
}

// IndexConfig represents index configuration
type IndexConfig struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Metric     string                 `json:"metric,omitempty"`
}

// NewIndexConfig creates a new index configuration
func NewIndexConfig(name, indexType string) *IndexConfig {
	return &IndexConfig{
		Name:       name,
		Type:       indexType,
		Parameters: make(map[string]interface{}),
	}
}

// IndexInfo represents index information
type IndexInfo struct {
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Status    string                 `json:"status"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Config    *IndexConfig           `json:"config,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewIndexInfo creates new index info
func NewIndexInfo(name, indexType, status string) *IndexInfo {
	return &IndexInfo{
		Name:      name,
		Type:      indexType,
		Status:    status,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Config:    nil,
		Metadata:  make(map[string]interface{}),
	}
}

// CostInfo represents cost information for providers
type CostInfo struct {
	Currency    string             `json:"currency"`
	ReadCost    float64            `json:"read_cost"`
	WriteCost   float64            `json:"write_cost"`
	StorageCost float64            `json:"storage_cost"`
	Details     map[string]float64 `json:"details,omitempty"`
}

// NewCostInfo creates new cost info
func NewCostInfo(currency string, readCost, writeCost, storageCost float64) *CostInfo {
	return &CostInfo{
		Currency:    currency,
		ReadCost:    readCost,
		WriteCost:   writeCost,
		StorageCost: storageCost,
		Details:     make(map[string]float64),
	}
}

// HealthStatus represents health status
type HealthStatus struct {
	Status    string                 `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// NewHealthStatus creates new health status
func NewHealthStatus(status, message string) *HealthStatus {
	return &HealthStatus{
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}
}

// ProviderStats represents provider statistics
type ProviderStats struct {
	Name             string                 `json:"name"`
	Type             string                 `json:"type"`
	Status           string                 `json:"status"`
	TotalOperations  int64                  `json:"total_operations"`
	SuccessfulOps    int64                  `json:"successful_ops"`
	FailedOps        int64                  `json:"failed_ops"`
	AvgResponseTime  time.Duration          `json:"avg_response_time"`
	TotalVectors     int64                  `json:"total_vectors"`
	TotalCollections int64                  `json:"total_collections"`
	StorageSize      int64                  `json:"storage_size"`
	LastHealthCheck  time.Time              `json:"last_health_check"`
	CostInfo         *CostInfo              `json:"cost_info,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// NewProviderStats creates new provider stats
func NewProviderStats(name, providerType, status string) *ProviderStats {
	return &ProviderStats{
		Name:             name,
		Type:             providerType,
		Status:           status,
		TotalOperations:  0,
		SuccessfulOps:    0,
		FailedOps:        0,
		AvgResponseTime:  0,
		TotalVectors:     0,
		TotalCollections: 0,
		StorageSize:      0,
		LastHealthCheck:  time.Now(),
		Metadata:         make(map[string]interface{}),
	}
}

// VectorProvider defines interface for vector database providers
type VectorProvider interface {
	// Core operations
	Store(ctx context.Context, vectors []*VectorData) error
	Retrieve(ctx context.Context, ids []string) ([]*VectorData, error)
	Update(ctx context.Context, id string, vector *VectorData) error
	Delete(ctx context.Context, ids []string) error

	// Search operations
	Search(ctx context.Context, query *VectorQuery) (*VectorSearchResult, error)
	FindSimilar(ctx context.Context, embedding []float64, k int, filters map[string]interface{}) ([]*VectorSimilarityResult, error)
	BatchFindSimilar(ctx context.Context, queries [][]float64, k int) ([][]*VectorSimilarityResult, error)

	// Collection management
	CreateCollection(ctx context.Context, name string, config *CollectionConfig) error
	DeleteCollection(ctx context.Context, name string) error
	ListCollections(ctx context.Context) ([]*CollectionInfo, error)
	GetCollection(ctx context.Context, name string) (*CollectionInfo, error)

	// Index management
	CreateIndex(ctx context.Context, collection string, config *IndexConfig) error
	DeleteIndex(ctx context.Context, collection, name string) error
	ListIndexes(ctx context.Context, collection string) ([]*IndexInfo, error)

	// Metadata operations
	AddMetadata(ctx context.Context, id string, metadata map[string]interface{}) error
	UpdateMetadata(ctx context.Context, id string, metadata map[string]interface{}) error
	GetMetadata(ctx context.Context, ids []string) (map[string]map[string]interface{}, error)
	DeleteMetadata(ctx context.Context, ids []string, keys []string) error

	// Management
	GetStats(ctx context.Context) (*ProviderStats, error)
	Optimize(ctx context.Context) error
	Backup(ctx context.Context, path string) error
	Restore(ctx context.Context, path string) error

	// Lifecycle
	Initialize(ctx context.Context, config interface{}) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) (*HealthStatus, error)

	// Metadata
	GetName() string
	GetType() string
	GetCapabilities() []string
	GetConfiguration() interface{}
	IsCloud() bool
	GetCostInfo() *CostInfo
}

// ProviderType represents different provider types
type ProviderType string

const (
	ProviderTypePinecone    ProviderType = "pinecone"
	ProviderTypeMilvus      ProviderType = "milvus"
	ProviderTypeWeaviate    ProviderType = "weaviate"
	ProviderTypeQdrant      ProviderType = "qdrant"
	ProviderTypeRedis       ProviderType = "redis"
	ProviderTypeChroma      ProviderType = "chroma"
	ProviderTypeOpenAI      ProviderType = "openai"
	ProviderTypeAnthropic   ProviderType = "anthropic"
	ProviderTypeCohere      ProviderType = "cohere"
	ProviderTypeHuggingFace ProviderType = "huggingface"
	ProviderTypeMistral     ProviderType = "mistral"
	ProviderTypeGemini      ProviderType = "gemini"
	ProviderTypeVertexAI    ProviderType = "vertexai"
	ProviderTypeClickHouse  ProviderType = "clickhouse"
	ProviderTypeSupabase    ProviderType = "supabase"
	ProviderTypeDeepLake    ProviderType = "deeplake"
	ProviderTypeFAISS       ProviderType = "faiss"
	ProviderTypeLlamaIndex  ProviderType = "llamaindex"
	ProviderTypeMemGPT      ProviderType = "memgpt"
	ProviderTypeCrewAI      ProviderType = "crewai"
	ProviderTypeCharacterAI ProviderType = "characterai"
	ProviderTypeReplika     ProviderType = "replika"
	ProviderTypeAnima       ProviderType = "anima"
	ProviderTypeGemma       ProviderType = "gemma"
	ProviderTypeMem0        ProviderType = "mem0"
	ProviderTypeZep         ProviderType = "zep"
	ProviderTypeMemonto     ProviderType = "memonto"
	ProviderTypeBaseAI      ProviderType = "baseai"
	ProviderTypeAgnostic    ProviderType = "agnostic"
)

// ============================================================================
// ID Generation Functions for Specialized Types
// ============================================================================

var (
	avatarCounter              uint64
	activityCounter            uint64
	emotionalStateCounter      uint64
	personalityCounter         uint64
	sessionCounter             uint64
	characterMessageCounter    uint64
	relationshipCounter        uint64
	activityPatternCounter     uint64
	moodDataCounter            uint64
	crewCounter                uint64
	agentCounter               uint64
	taskCounter                uint64
	taskResultCounter          uint64
	sharedMemoryCounter        uint64
	crewPerformanceCounter     uint64
	memoryBlockCounter         uint64
	workingMemoryCounter       uint64
	processingResultCounter    uint64
	searchQueryCounter         uint64
	conversationSummaryCounter uint64
	contextDataCounter         uint64
	retrievalQueryCounter      uint64
	systemInfoCounter          uint64
	optimizationRecCounter     uint64
	modelCounter               uint64
	embeddingCounter           uint64
	modelPerformanceCounter    uint64
	personalityMessageCounter  uint64
)

func generateAvatarID() string {
	count := atomic.AddUint64(&avatarCounter, 1)
	return fmt.Sprintf("avatar-%d-%d", time.Now().UnixNano(), count)
}

func generateActivityID() string {
	count := atomic.AddUint64(&activityCounter, 1)
	return fmt.Sprintf("activity-%d-%d", time.Now().UnixNano(), count)
}

func generateEmotionalStateID() string {
	count := atomic.AddUint64(&emotionalStateCounter, 1)
	return fmt.Sprintf("emotion-%d-%d", time.Now().UnixNano(), count)
}

func generatePersonalityID() string {
	count := atomic.AddUint64(&personalityCounter, 1)
	return fmt.Sprintf("personality-%d-%d", time.Now().UnixNano(), count)
}

func generateSessionID() string {
	count := atomic.AddUint64(&sessionCounter, 1)
	return fmt.Sprintf("session-%d-%d", time.Now().UnixNano(), count)
}

func generateCharacterMessageID() string {
	count := atomic.AddUint64(&characterMessageCounter, 1)
	return fmt.Sprintf("char-msg-%d-%d", time.Now().UnixNano(), count)
}

func generateRelationshipID() string {
	count := atomic.AddUint64(&relationshipCounter, 1)
	return fmt.Sprintf("relationship-%d-%d", time.Now().UnixNano(), count)
}

func generateActivityPatternID() string {
	count := atomic.AddUint64(&activityPatternCounter, 1)
	return fmt.Sprintf("pattern-%d-%d", time.Now().UnixNano(), count)
}

func generateMoodDataID() string {
	count := atomic.AddUint64(&moodDataCounter, 1)
	return fmt.Sprintf("mood-%d-%d", time.Now().UnixNano(), count)
}

func generateCrewID() string {
	count := atomic.AddUint64(&crewCounter, 1)
	return fmt.Sprintf("crew-%d-%d", time.Now().UnixNano(), count)
}

func generateAgentID() string {
	count := atomic.AddUint64(&agentCounter, 1)
	return fmt.Sprintf("agent-%d-%d", time.Now().UnixNano(), count)
}

func generateTaskID() string {
	count := atomic.AddUint64(&taskCounter, 1)
	return fmt.Sprintf("task-%d-%d", time.Now().UnixNano(), count)
}

func generateTaskResultID() string {
	count := atomic.AddUint64(&taskResultCounter, 1)
	return fmt.Sprintf("task-result-%d-%d", time.Now().UnixNano(), count)
}

func generateSharedMemoryID() string {
	count := atomic.AddUint64(&sharedMemoryCounter, 1)
	return fmt.Sprintf("shared-mem-%d-%d", time.Now().UnixNano(), count)
}

func generateCrewPerformanceID() string {
	count := atomic.AddUint64(&crewPerformanceCounter, 1)
	return fmt.Sprintf("crew-perf-%d-%d", time.Now().UnixNano(), count)
}

func generateMemoryBlockID() string {
	count := atomic.AddUint64(&memoryBlockCounter, 1)
	return fmt.Sprintf("mem-block-%d-%d", time.Now().UnixNano(), count)
}

func generateWorkingMemoryID() string {
	count := atomic.AddUint64(&workingMemoryCounter, 1)
	return fmt.Sprintf("working-mem-%d-%d", time.Now().UnixNano(), count)
}

func generateProcessingResultID() string {
	count := atomic.AddUint64(&processingResultCounter, 1)
	return fmt.Sprintf("proc-result-%d-%d", time.Now().UnixNano(), count)
}

func generateSearchQueryID() string {
	count := atomic.AddUint64(&searchQueryCounter, 1)
	return fmt.Sprintf("search-q-%d-%d", time.Now().UnixNano(), count)
}

func generateConversationSummaryID() string {
	count := atomic.AddUint64(&conversationSummaryCounter, 1)
	return fmt.Sprintf("conv-summary-%d-%d", time.Now().UnixNano(), count)
}

func generateContextDataID() string {
	count := atomic.AddUint64(&contextDataCounter, 1)
	return fmt.Sprintf("context-%d-%d", time.Now().UnixNano(), count)
}

func generateRetrievalQueryID() string {
	count := atomic.AddUint64(&retrievalQueryCounter, 1)
	return fmt.Sprintf("retrieval-q-%d-%d", time.Now().UnixNano(), count)
}

func generateSystemInfoID() string {
	count := atomic.AddUint64(&systemInfoCounter, 1)
	return fmt.Sprintf("sys-info-%d-%d", time.Now().UnixNano(), count)
}

func generateOptimizationRecommendationID() string {
	count := atomic.AddUint64(&optimizationRecCounter, 1)
	return fmt.Sprintf("opt-rec-%d-%d", time.Now().UnixNano(), count)
}

func generateCharacterID() string {
	count := atomic.AddUint64(&characterCounter, 1)
	return fmt.Sprintf("char-%d-%d", time.Now().UnixNano(), count)
}

func generateModelID() string {
	count := atomic.AddUint64(&modelCounter, 1)
	return fmt.Sprintf("model-%d-%d", time.Now().UnixNano(), count)
}

func generateEmbeddingID() string {
	count := atomic.AddUint64(&embeddingCounter, 1)
	return fmt.Sprintf("emb-%d-%d", time.Now().UnixNano(), count)
}

func generateModelPerformanceID() string {
	count := atomic.AddUint64(&modelPerformanceCounter, 1)
	return fmt.Sprintf("perf-%d-%d", time.Now().UnixNano(), count)
}

func generatePersonalityMessageID() string {
	count := atomic.AddUint64(&personalityMessageCounter, 1)
	return fmt.Sprintf("pers-msg-%d-%d", time.Now().UnixNano(), count)
}
