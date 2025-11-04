package testutil

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockSlackServer(t *testing.T) {
	server := NewMockSlackServer()
	defer server.Close()

	// Send a test request
	payload := SlackRequest{
		Channel:   "#test",
		Username:  "testbot",
		Text:      "Hello",
		IconEmoji: ":rocket:",
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(server.URL, "application/json", bytes.NewReader(jsonData))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify request was captured
	requests := server.GetRequests()
	assert.Equal(t, 1, len(requests))
	assert.Equal(t, "#test", requests[0].Channel)
	assert.Equal(t, "testbot", requests[0].Username)
	assert.Equal(t, "Hello", requests[0].Text)
	assert.Equal(t, ":rocket:", requests[0].IconEmoji)
}

func TestMockSlackServer_Reset(t *testing.T) {
	server := NewMockSlackServer()
	defer server.Close()

	// Send request
	payload := SlackRequest{Channel: "#test"}
	jsonData, _ := json.Marshal(payload)
	http.Post(server.URL, "application/json", bytes.NewReader(jsonData))

	assert.Equal(t, 1, server.GetRequestCount())

	// Reset
	server.Reset()
	assert.Equal(t, 0, server.GetRequestCount())
}

func TestMockSlackServer_InvalidMethod(t *testing.T) {
	server := NewMockSlackServer()
	defer server.Close()

	// Send GET request (should fail)
	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)

	// Verify no request was captured
	assert.Equal(t, 0, server.GetRequestCount())
}

func TestMockSlackServer_InvalidJSON(t *testing.T) {
	server := NewMockSlackServer()
	defer server.Close()

	// Send invalid JSON
	resp, err := http.Post(server.URL, "application/json", bytes.NewReader([]byte("invalid json")))
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Verify no request was captured
	assert.Equal(t, 0, server.GetRequestCount())
}

func TestMockSlackServer_MultipleRequests(t *testing.T) {
	server := NewMockSlackServer()
	defer server.Close()

	// Send multiple requests
	for i := 0; i < 5; i++ {
		payload := SlackRequest{
			Channel: "#test",
			Text:    "Message " + string(rune(i+'0')),
		}
		jsonData, _ := json.Marshal(payload)
		http.Post(server.URL, "application/json", bytes.NewReader(jsonData))
	}

	assert.Equal(t, 5, server.GetRequestCount())
	requests := server.GetRequests()
	assert.Equal(t, 5, len(requests))
}

func TestMockTelegramServer(t *testing.T) {
	server := NewMockTelegramServer()
	defer server.Close()

	// Send a test request
	payload := TelegramRequest{
		ChatID:    "123456789",
		Text:      "Hello Telegram",
		ParseMode: "Markdown",
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(server.URL, "application/json", bytes.NewReader(jsonData))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify response format
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.True(t, response["ok"].(bool))
	assert.NotNil(t, response["result"])

	// Verify request was captured
	requests := server.GetRequests()
	assert.Equal(t, 1, len(requests))
	assert.Equal(t, "123456789", requests[0].ChatID)
	assert.Equal(t, "Hello Telegram", requests[0].Text)
	assert.Equal(t, "Markdown", requests[0].ParseMode)
}

func TestMockTelegramServer_Reset(t *testing.T) {
	server := NewMockTelegramServer()
	defer server.Close()

	// Send request
	payload := TelegramRequest{ChatID: "123"}
	jsonData, _ := json.Marshal(payload)
	http.Post(server.URL, "application/json", bytes.NewReader(jsonData))

	assert.Equal(t, 1, server.GetRequestCount())

	// Reset
	server.Reset()
	assert.Equal(t, 0, server.GetRequestCount())
}

func TestMockTelegramServer_InvalidMethod(t *testing.T) {
	server := NewMockTelegramServer()
	defer server.Close()

	// Send GET request (should fail)
	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)

	// Verify no request was captured
	assert.Equal(t, 0, server.GetRequestCount())
}

func TestMockTelegramServer_InvalidJSON(t *testing.T) {
	server := NewMockTelegramServer()
	defer server.Close()

	// Send invalid JSON
	resp, err := http.Post(server.URL, "application/json", bytes.NewReader([]byte("invalid json")))
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Verify no request was captured
	assert.Equal(t, 0, server.GetRequestCount())
}

func TestMockTelegramServer_MessageIDIncrement(t *testing.T) {
	server := NewMockTelegramServer()
	defer server.Close()

	// Send multiple requests and verify message_id increments
	for i := 1; i <= 3; i++ {
		payload := TelegramRequest{ChatID: "123", Text: "Test"}
		jsonData, _ := json.Marshal(payload)
		resp, _ := http.Post(server.URL, "application/json", bytes.NewReader(jsonData))

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)
		result := response["result"].(map[string]interface{})
		messageID := int(result["message_id"].(float64))

		assert.Equal(t, i, messageID, "Message ID should increment")
	}
}

func TestMockDiscordServer(t *testing.T) {
	server := NewMockDiscordServer()
	defer server.Close()

	// Send a test request
	payload := DiscordRequest{
		Content: "Hello Discord",
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(server.URL, "application/json", bytes.NewReader(jsonData))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify request was captured
	requests := server.GetRequests()
	assert.Equal(t, 1, len(requests))
	assert.Equal(t, "Hello Discord", requests[0].Content)
}

func TestMockDiscordServer_Reset(t *testing.T) {
	server := NewMockDiscordServer()
	defer server.Close()

	// Send request
	payload := DiscordRequest{Content: "Test"}
	jsonData, _ := json.Marshal(payload)
	http.Post(server.URL, "application/json", bytes.NewReader(jsonData))

	assert.Equal(t, 1, server.GetRequestCount())

	// Reset
	server.Reset()
	assert.Equal(t, 0, server.GetRequestCount())
}

func TestMockDiscordServer_InvalidMethod(t *testing.T) {
	server := NewMockDiscordServer()
	defer server.Close()

	// Send GET request (should fail)
	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)

	// Verify no request was captured
	assert.Equal(t, 0, server.GetRequestCount())
}

func TestMockDiscordServer_InvalidJSON(t *testing.T) {
	server := NewMockDiscordServer()
	defer server.Close()

	// Send invalid JSON
	resp, err := http.Post(server.URL, "application/json", bytes.NewReader([]byte("invalid json")))
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Verify no request was captured
	assert.Equal(t, 0, server.GetRequestCount())
}

func TestMockDiscordServer_MultipleRequests(t *testing.T) {
	server := NewMockDiscordServer()
	defer server.Close()

	// Send multiple requests
	for i := 0; i < 5; i++ {
		payload := DiscordRequest{Content: "Message"}
		jsonData, _ := json.Marshal(payload)
		http.Post(server.URL, "application/json", bytes.NewReader(jsonData))
	}

	assert.Equal(t, 5, server.GetRequestCount())
	requests := server.GetRequests()
	assert.Equal(t, 5, len(requests))
}

// Test thread safety
func TestMockServers_ThreadSafety(t *testing.T) {
	t.Run("Slack thread safety", func(t *testing.T) {
		server := NewMockSlackServer()
		defer server.Close()

		// Send concurrent requests
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				payload := SlackRequest{Channel: "#test", Text: "Concurrent"}
				jsonData, _ := json.Marshal(payload)
				http.Post(server.URL, "application/json", bytes.NewReader(jsonData))
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		assert.Equal(t, 10, server.GetRequestCount())
	})

	t.Run("Telegram thread safety", func(t *testing.T) {
		server := NewMockTelegramServer()
		defer server.Close()

		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				payload := TelegramRequest{ChatID: "123", Text: "Concurrent"}
				jsonData, _ := json.Marshal(payload)
				http.Post(server.URL, "application/json", bytes.NewReader(jsonData))
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}

		assert.Equal(t, 10, server.GetRequestCount())
	})

	t.Run("Discord thread safety", func(t *testing.T) {
		server := NewMockDiscordServer()
		defer server.Close()

		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				payload := DiscordRequest{Content: "Concurrent"}
				jsonData, _ := json.Marshal(payload)
				http.Post(server.URL, "application/json", bytes.NewReader(jsonData))
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}

		assert.Equal(t, 10, server.GetRequestCount())
	})
}
