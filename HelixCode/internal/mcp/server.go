package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// MCPServer implements the Model Context Protocol server
type MCPServer struct {
	upgrader   websocket.Upgrader
	sessions   map[uuid.UUID]*MCPSession
	sessionMux sync.RWMutex
	tools      map[string]*Tool
	toolMux    sync.RWMutex
}

// MCPSession represents an MCP session
type MCPSession struct {
	ID           uuid.UUID
	Conn         *websocket.Conn
	CreatedAt    time.Time
	LastActivity time.Time
	UserID       uuid.UUID
	Context      map[string]interface{}
}

// Tool represents an MCP tool
type Tool struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Handler     ToolHandler            `json:"-"`
	Permissions []string               `json:"permissions"`
}

// ToolHandler is the function signature for tool execution
type ToolHandler func(ctx context.Context, session *MCPSession, args map[string]interface{}) (interface{}, error)

// MCPMessage represents an MCP protocol message
type MCPMessage struct {
	ID     string          `json:"id"`
	Type   string          `json:"type"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	Result interface{}     `json:"result,omitempty"`
	Error  *MCPError       `json:"error,omitempty"`
}

// MCPError represents an MCP protocol error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewMCPServer creates a new MCP server
func NewMCPServer() *MCPServer {
	return &MCPServer{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// In production, you should validate the origin
				return true
			},
		},
		sessions: make(map[uuid.UUID]*MCPSession),
		tools:    make(map[string]*Tool),
	}
}

// RegisterTool registers a new tool with the MCP server
func (s *MCPServer) RegisterTool(tool *Tool) error {
	s.toolMux.Lock()
	defer s.toolMux.Unlock()

	if _, exists := s.tools[tool.ID]; exists {
		return fmt.Errorf("tool with ID %s already registered", tool.ID)
	}

	s.tools[tool.ID] = tool
	log.Printf("✅ MCP Tool registered: %s (%s)", tool.Name, tool.ID)
	return nil
}

// HandleWebSocket handles WebSocket connections for MCP
func (s *MCPServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ Failed to upgrade WebSocket connection: %v", err)
		return
	}

	// Create new session
	session := &MCPSession{
		ID:           uuid.New(),
		Conn:         conn,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		Context:      make(map[string]interface{}),
	}

	// Store session
	s.sessionMux.Lock()
	s.sessions[session.ID] = session
	s.sessionMux.Unlock()

	log.Printf("🔌 MCP Session started: %s", session.ID)

	// Handle session
	go s.handleSession(session)
}

// handleSession handles an individual MCP session
func (s *MCPServer) handleSession(session *MCPSession) {
	defer func() {
		session.Conn.Close()
		s.sessionMux.Lock()
		delete(s.sessions, session.ID)
		s.sessionMux.Unlock()
		log.Printf("🔌 MCP Session ended: %s", session.ID)
	}()

	for {
		var message MCPMessage
		err := session.Conn.ReadJSON(&message)
		if err != nil {
			log.Printf("❌ Failed to read MCP message: %v", err)
			break
		}

		session.LastActivity = time.Now()

		// Handle message
		go s.handleMessage(session, &message)
	}
}

// handleMessage handles an individual MCP message
func (s *MCPServer) handleMessage(session *MCPSession, message *MCPMessage) {
	ctx := context.Background()

	switch message.Method {
	case "initialize":
		s.handleInitialize(session, message)
	case "tools/list":
		s.handleListTools(session, message)
	case "tools/call":
		s.handleCallTool(ctx, session, message)
	case "notifications/capabilities":
		s.handleCapabilities(session, message)
	case "ping":
		s.handlePing(session, message)
	default:
		s.sendError(session, message.ID, -32601, "Method not found", nil)
	}
}

// handleInitialize handles the initialize method
func (s *MCPServer) handleInitialize(session *MCPSession, message *MCPMessage) {
	var params struct {
		ProtocolVersion string                 `json:"protocolVersion"`
		Capabilities    map[string]interface{} `json:"capabilities"`
		ClientInfo      struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"clientInfo"`
	}

	if err := json.Unmarshal(message.Params, &params); err != nil {
		s.sendError(session, message.ID, -32700, "Parse error", nil)
		return
	}

	response := MCPMessage{
		ID:   message.ID,
		Type: "response",
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"roots": map[string]interface{}{
					"listChanged": true,
				},
				"sampling": map[string]interface{}{
					"enabled": true,
				},
			},
			"serverInfo": map[string]interface{}{
				"name":    "HelixCode MCP Server",
				"version": "1.0.0",
			},
		},
	}

	s.sendMessage(session, &response)
}

// handleListTools handles the tools/list method
func (s *MCPServer) handleListTools(session *MCPSession, message *MCPMessage) {
	s.toolMux.RLock()
	defer s.toolMux.RUnlock()

	tools := make([]map[string]interface{}, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"parameters":  tool.Parameters,
		})
	}

	response := MCPMessage{
		ID:   message.ID,
		Type: "response",
		Result: map[string]interface{}{
			"tools": tools,
		},
	}

	s.sendMessage(session, &response)
}

// handleCallTool handles the tools/call method
func (s *MCPServer) handleCallTool(ctx context.Context, session *MCPSession, message *MCPMessage) {
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.Unmarshal(message.Params, &params); err != nil {
		s.sendError(session, message.ID, -32700, "Parse error", nil)
		return
	}

	s.toolMux.RLock()
	tool, exists := s.tools[params.Name]
	s.toolMux.RUnlock()

	if !exists {
		s.sendError(session, message.ID, -32601, "Tool not found", nil)
		return
	}

	// Execute tool
	result, err := tool.Handler(ctx, session, params.Arguments)
	if err != nil {
		s.sendError(session, message.ID, -32000, "Tool execution failed", err.Error())
		return
	}

	response := MCPMessage{
		ID:   message.ID,
		Type: "response",
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("%v", result),
				},
			},
		},
	}

	s.sendMessage(session, &response)
}

// handleCapabilities handles the capabilities notification
func (s *MCPServer) handleCapabilities(session *MCPSession, message *MCPMessage) {
	// Acknowledge capabilities notification
	response := MCPMessage{
		ID:   message.ID,
		Type: "response",
		Result: map[string]interface{}{
			"capabilities": map[string]interface{}{
				"experimental": map[string]interface{}{},
			},
		},
	}

	s.sendMessage(session, &response)
}

// handlePing handles the ping method
func (s *MCPServer) handlePing(session *MCPSession, message *MCPMessage) {
	response := MCPMessage{
		ID:   message.ID,
		Type: "response",
		Result: map[string]interface{}{
			"pong": true,
		},
	}

	s.sendMessage(session, &response)
}

// sendMessage sends a message to a session
func (s *MCPServer) sendMessage(session *MCPSession, message *MCPMessage) error {
	session.Conn.WriteJSON(message)
	return nil
}

// sendError sends an error response
func (s *MCPServer) sendError(session *MCPSession, id string, code int, message string, data interface{}) {
	errorResponse := MCPMessage{
		ID:   id,
		Type: "response",
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}

	s.sendMessage(session, &errorResponse)
}

// BroadcastNotification broadcasts a notification to all sessions
func (s *MCPServer) BroadcastNotification(method string, params interface{}) {
	s.sessionMux.RLock()
	defer s.sessionMux.RUnlock()

	// Convert params to JSON
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		log.Printf("❌ Failed to marshal notification params: %v", err)
		return
	}

	notification := MCPMessage{
		ID:     uuid.New().String(),
		Type:   "notification",
		Method: method,
		Params: json.RawMessage(paramsJSON),
	}

	for _, session := range s.sessions {
		go s.sendMessage(session, &notification)
	}
}

// GetSessionCount returns the number of active sessions
func (s *MCPServer) GetSessionCount() int {
	s.sessionMux.RLock()
	defer s.sessionMux.RUnlock()
	return len(s.sessions)
}

// GetToolCount returns the number of registered tools
func (s *MCPServer) GetToolCount() int {
	s.toolMux.RLock()
	defer s.toolMux.RUnlock()
	return len(s.tools)
}

// CloseSession closes a specific session
func (s *MCPServer) CloseSession(sessionID uuid.UUID) {
	s.sessionMux.Lock()
	defer s.sessionMux.Unlock()

	if session, exists := s.sessions[sessionID]; exists {
		session.Conn.Close()
		delete(s.sessions, sessionID)
	}
}

// CloseAllSessions closes all active sessions
func (s *MCPServer) CloseAllSessions() {
	s.sessionMux.Lock()
	defer s.sessionMux.Unlock()

	for _, session := range s.sessions {
		session.Conn.Close()
	}
	s.sessions = make(map[uuid.UUID]*MCPSession)
}
