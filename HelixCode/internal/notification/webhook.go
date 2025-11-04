package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// WebhookChannel implements generic webhook notifications
type WebhookChannel struct {
	name    string
	enabled bool
	url     string
	headers map[string]string
	method  string
}

// NewWebhookChannel creates a new generic webhook channel
func NewWebhookChannel(url string, headers map[string]string) *WebhookChannel {
	if headers == nil {
		headers = make(map[string]string)
	}

	return &WebhookChannel{
		name:    "webhook",
		enabled: url != "",
		url:     url,
		headers: headers,
		method:  "POST",
	}
}

func (c *WebhookChannel) Send(ctx context.Context, notification *Notification) error {
	if !c.enabled {
		return fmt.Errorf("webhook channel disabled")
	}

	payload := map[string]interface{}{
		"title":    notification.Title,
		"message":  notification.Message,
		"type":     string(notification.Type),
		"priority": string(notification.Priority),
		"metadata": notification.Metadata,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, c.method, c.url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *WebhookChannel) GetName() string {
	return c.name
}

func (c *WebhookChannel) IsEnabled() bool {
	return c.enabled
}

func (c *WebhookChannel) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"url":     c.url,
		"method":  c.method,
		"headers": len(c.headers),
	}
}

// TeamsChannel implements Microsoft Teams notifications
type TeamsChannel struct {
	name    string
	enabled bool
	webhook string
}

// NewTeamsChannel creates a new MS Teams channel
func NewTeamsChannel(webhook string) *TeamsChannel {
	return &TeamsChannel{
		name:    "teams",
		enabled: webhook != "",
		webhook: webhook,
	}
}

func (c *TeamsChannel) Send(ctx context.Context, notification *Notification) error {
	if !c.enabled {
		return fmt.Errorf("teams channel disabled")
	}

	// Teams Adaptive Card format
	payload := map[string]interface{}{
		"@type":      "MessageCard",
		"@context":   "https://schema.org/extensions",
		"summary":    notification.Title,
		"themeColor": c.getColorForType(notification.Type),
		"title":      notification.Title,
		"text":       notification.Message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal teams payload: %v", err)
	}

	resp, err := http.Post(c.webhook, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send to teams: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("teams returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *TeamsChannel) GetName() string {
	return c.name
}

func (c *TeamsChannel) IsEnabled() bool {
	return c.enabled
}

func (c *TeamsChannel) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"webhook": c.webhook,
	}
}

func (c *TeamsChannel) getColorForType(notifType NotificationType) string {
	switch notifType {
	case NotificationTypeSuccess:
		return "28a745" // Green
	case NotificationTypeWarning:
		return "ffc107" // Yellow
	case NotificationTypeError, NotificationTypeAlert:
		return "dc3545" // Red
	default:
		return "0078d7" // Blue
	}
}
