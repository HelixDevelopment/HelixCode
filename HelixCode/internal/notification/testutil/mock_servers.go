package testutil

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
)

// MockSlackServer simulates Slack webhook endpoint
type MockSlackServer struct {
	*httptest.Server
	Requests []SlackRequest
	mutex    sync.Mutex
}

type SlackRequest struct {
	Channel   string `json:"channel"`
	Username  string `json:"username"`
	Text      string `json:"text"`
	IconEmoji string `json:"icon_emoji"`
}

func NewMockSlackServer() *MockSlackServer {
	mock := &MockSlackServer{
		Requests: make([]SlackRequest, 0),
	}

	mock.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mock.mutex.Lock()
		defer mock.mutex.Unlock()

		// Verify it's POST
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Parse request
		var req SlackRequest
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		mock.Requests = append(mock.Requests, req)
		w.WriteHeader(http.StatusOK)
	}))

	return mock
}

func (m *MockSlackServer) GetRequests() []SlackRequest {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return append([]SlackRequest{}, m.Requests...)
}

func (m *MockSlackServer) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.Requests = make([]SlackRequest, 0)
}

func (m *MockSlackServer) GetRequestCount() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return len(m.Requests)
}

// MockTelegramServer simulates Telegram Bot API
type MockTelegramServer struct {
	*httptest.Server
	Requests []TelegramRequest
	mutex    sync.Mutex
}

type TelegramRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

func NewMockTelegramServer() *MockTelegramServer {
	mock := &MockTelegramServer{
		Requests: make([]TelegramRequest, 0),
	}

	mock.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mock.mutex.Lock()
		defer mock.mutex.Unlock()

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req TelegramRequest
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		mock.Requests = append(mock.Requests, req)

		// Return successful Telegram API response
		response := map[string]interface{}{
			"ok": true,
			"result": map[string]interface{}{
				"message_id": len(mock.Requests),
				"chat": map[string]interface{}{
					"id": req.ChatID,
				},
				"text": req.Text,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))

	return mock
}

func (m *MockTelegramServer) GetRequests() []TelegramRequest {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return append([]TelegramRequest{}, m.Requests...)
}

func (m *MockTelegramServer) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.Requests = make([]TelegramRequest, 0)
}

func (m *MockTelegramServer) GetRequestCount() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return len(m.Requests)
}

// MockDiscordServer simulates Discord webhook endpoint
type MockDiscordServer struct {
	*httptest.Server
	Requests []DiscordRequest
	mutex    sync.Mutex
}

type DiscordRequest struct {
	Content string `json:"content"`
}

func NewMockDiscordServer() *MockDiscordServer {
	mock := &MockDiscordServer{
		Requests: make([]DiscordRequest, 0),
	}

	mock.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mock.mutex.Lock()
		defer mock.mutex.Unlock()

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req DiscordRequest
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		mock.Requests = append(mock.Requests, req)
		w.WriteHeader(http.StatusNoContent)
	}))

	return mock
}

func (m *MockDiscordServer) GetRequests() []DiscordRequest {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return append([]DiscordRequest{}, m.Requests...)
}

func (m *MockDiscordServer) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.Requests = make([]DiscordRequest, 0)
}

func (m *MockDiscordServer) GetRequestCount() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return len(m.Requests)
}
