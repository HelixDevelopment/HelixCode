package mcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMCPServer(t *testing.T) {
	server := NewMCPServer()

	assert.NotNil(t, server)
	assert.NotNil(t, server.sessions)
	assert.NotNil(t, server.tools)
	assert.Equal(t, 0, server.GetSessionCount())
	assert.Equal(t, 0, server.GetToolCount())
}

func TestRegisterTool(t *testing.T) {
	server := NewMCPServer()

	tool := &Tool{
		ID:          "test-tool",
		Name:        "Test Tool",
		Description: "A test tool",
		Parameters:  map[string]interface{}{},
		Handler: func(ctx context.Context, session *MCPSession, args map[string]interface{}) (interface{}, error) {
			return "test result", nil
		},
	}

	err := server.RegisterTool(tool)
	assert.NoError(t, err)
	assert.Equal(t, 1, server.GetToolCount())

	// Try to register the same tool again
	err = server.RegisterTool(tool)
	assert.Error(t, err)
	assert.Equal(t, 1, server.GetToolCount())
}

func TestCloseAllSessions(t *testing.T) {
	server := NewMCPServer()

	// Since we can't easily create real sessions without WebSocket, just test the method doesn't panic
	server.CloseAllSessions()
	assert.Equal(t, 0, server.GetSessionCount())
}
