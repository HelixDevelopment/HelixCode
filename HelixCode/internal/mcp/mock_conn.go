package mcp

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
)

// MockConn is a mock WebSocket connection for testing
type MockConn struct {
	writeCalled  bool
	lastMessage  []byte
	closeCalled  bool
	readMessages []interface{}
}

func (m *MockConn) ReadJSON(v interface{}) error {
	if len(m.readMessages) == 0 {
		return &websocket.CloseError{Code: websocket.CloseNormalClosure}
	}
	// Pop the first message
	msg := m.readMessages[0]
	m.readMessages = m.readMessages[1:]

	// Convert message to JSON bytes and unmarshal
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonBytes, v)
}

func (m *MockConn) WriteJSON(v interface{}) error {
	m.writeCalled = true
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return err
	}
	m.lastMessage = jsonBytes
	return nil
}

func (m *MockConn) Close() error {
	m.closeCalled = true
	return nil
}

func (m *MockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *MockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (m *MockConn) SetPongHandler(h func(string) error) {
	// No-op for testing
}

func (m *MockConn) WriteMessage(messageType int, data []byte) error {
	m.writeCalled = true
	m.lastMessage = data
	return nil
}
