package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// NotificationEngine manages multi-channel notifications
type NotificationEngine struct {
	channels  map[string]NotificationChannel
	rules     []NotificationRule
	templates map[string]*template.Template
	mutex     sync.RWMutex
}

// NotificationChannel represents a notification channel
type NotificationChannel interface {
	Send(ctx context.Context, notification *Notification) error
	GetName() string
	IsEnabled() bool
	GetConfig() map[string]interface{}
}

// Notification represents a notification to be sent
type Notification struct {
	ID        uuid.UUID
	Title     string
	Message   string
	Type      NotificationType
	Priority  NotificationPriority
	Channels  []string
	Metadata  map[string]interface{}
	CreatedAt time.Time
}

// NotificationType defines the type of notification
type NotificationType string

const (
	NotificationTypeInfo    NotificationType = "info"
	NotificationTypeWarning NotificationType = "warning"
	NotificationTypeError   NotificationType = "error"
	NotificationTypeSuccess NotificationType = "success"
	NotificationTypeAlert   NotificationType = "alert"
)

// NotificationPriority defines the priority level
type NotificationPriority string

const (
	NotificationPriorityLow    NotificationPriority = "low"
	NotificationPriorityMedium NotificationPriority = "medium"
	NotificationPriorityHigh   NotificationPriority = "high"
	NotificationPriorityUrgent NotificationPriority = "urgent"
)

// NotificationRule defines when and how to send notifications
type NotificationRule struct {
	ID        uuid.UUID
	Name      string
	Condition string
	Channels  []string
	Priority  NotificationPriority
	Enabled   bool
	Template  string
}

// NewNotificationEngine creates a new notification engine
func NewNotificationEngine() *NotificationEngine {
	return &NotificationEngine{
		channels:  make(map[string]NotificationChannel),
		rules:     []NotificationRule{},
		templates: make(map[string]*template.Template),
	}
}

// RegisterChannel registers a notification channel
func (e *NotificationEngine) RegisterChannel(channel NotificationChannel) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	name := channel.GetName()
	if _, exists := e.channels[name]; exists {
		return fmt.Errorf("channel %s already registered", name)
	}

	e.channels[name] = channel
	log.Printf("Notification channel registered: %s", name)
	return nil
}

// AddRule adds a notification rule
func (e *NotificationEngine) AddRule(rule NotificationRule) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	rule.ID = uuid.New()
	e.rules = append(e.rules, rule)
	log.Printf("Notification rule added: %s", rule.Name)
	return nil
}

// LoadTemplate loads a notification template
func (e *NotificationEngine) LoadTemplate(name string, templateStr string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	tmpl, err := template.New(name).Parse(templateStr)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %v", name, err)
	}

	e.templates[name] = tmpl
	log.Printf("Notification template loaded: %s", name)
	return nil
}

// SendNotification sends a notification based on rules
func (e *NotificationEngine) SendNotification(ctx context.Context, notification *Notification) error {
	notification.ID = uuid.New()
	notification.CreatedAt = time.Now()

	// Apply rules to determine channels and priority
	e.applyRules(notification)

	// Send through specified channels
	return e.sendToChannels(ctx, notification)
}

// SendDirect sends a notification directly to specified channels
func (e *NotificationEngine) SendDirect(ctx context.Context, notification *Notification, channels []string) error {
	notification.ID = uuid.New()
	notification.CreatedAt = time.Now()
	notification.Channels = channels

	return e.sendToChannels(ctx, notification)
}

// applyRules applies notification rules to determine channels and priority
func (e *NotificationEngine) applyRules(notification *Notification) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}

		// Simple condition matching (in production, use a proper expression evaluator)
		if e.matchesCondition(notification, rule.Condition) {
			// Add channels from rule
			for _, channel := range rule.Channels {
				if !contains(notification.Channels, channel) {
					notification.Channels = append(notification.Channels, channel)
				}
			}

			// Apply rule priority if higher than current
			if e.getPriorityLevel(rule.Priority) > e.getPriorityLevel(notification.Priority) {
				notification.Priority = rule.Priority
			}

			// Apply template if specified
			if rule.Template != "" {
				e.applyTemplate(notification, rule.Template)
			}
		}
	}
}

// sendToChannels sends notification to all specified channels
func (e *NotificationEngine) sendToChannels(ctx context.Context, notification *Notification) error {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	var errors []string

	for _, channelName := range notification.Channels {
		channel, exists := e.channels[channelName]
		if !exists || !channel.IsEnabled() {
			log.Printf("Warning: Channel %s not found or disabled", channelName)
			continue
		}

		if err := channel.Send(ctx, notification); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", channelName, err))
			log.Printf("Failed to send notification via %s: %v", channelName, err)
		} else {
			log.Printf("Notification sent via %s: %s", channelName, notification.Title)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to send to some channels: %s", strings.Join(errors, "; "))
	}

	return nil
}

// Helper methods

func (e *NotificationEngine) matchesCondition(notification *Notification, condition string) bool {
	// Simple condition matching - in production, use a proper expression evaluator
	if condition == "" {
		return true
	}

	// Check for type matching
	if strings.Contains(condition, "type=="+string(notification.Type)) {
		return true
	}

	// Check for priority matching
	if strings.Contains(condition, "priority=="+string(notification.Priority)) {
		return true
	}

	// Check for title/message contains
	if strings.Contains(condition, "contains:") {
		keyword := strings.TrimPrefix(condition, "contains:")
		if strings.Contains(strings.ToLower(notification.Title), strings.ToLower(keyword)) ||
			strings.Contains(strings.ToLower(notification.Message), strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}

func (e *NotificationEngine) getPriorityLevel(priority NotificationPriority) int {
	switch priority {
	case NotificationPriorityLow:
		return 1
	case NotificationPriorityMedium:
		return 2
	case NotificationPriorityHigh:
		return 3
	case NotificationPriorityUrgent:
		return 4
	default:
		return 0
	}
}

func (e *NotificationEngine) applyTemplate(notification *Notification, templateName string) {
	tmpl, exists := e.templates[templateName]
	if !exists {
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, notification); err == nil {
		notification.Message = buf.String()
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GetChannelStats returns statistics about notification channels
func (e *NotificationEngine) GetChannelStats() map[string]interface{} {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	stats := make(map[string]interface{})
	enabledCount := 0

	for name, channel := range e.channels {
		stats[name] = map[string]interface{}{
			"enabled": channel.IsEnabled(),
			"config":  channel.GetConfig(),
		}
		if channel.IsEnabled() {
			enabledCount++
		}
	}

	stats["summary"] = map[string]interface{}{
		"total_channels":   len(e.channels),
		"enabled_channels": enabledCount,
		"total_rules":      len(e.rules),
		"active_rules":     e.countActiveRules(),
	}

	return stats
}

func (e *NotificationEngine) countActiveRules() int {
	count := 0
	for _, rule := range e.rules {
		if rule.Enabled {
			count++
		}
	}
	return count
}

// SlackChannel implements notification channel for Slack
type SlackChannel struct {
	name     string
	enabled  bool
	webhook  string
	channel  string
	username string
}

func NewSlackChannel(webhook, channel, username string) *SlackChannel {
	return &SlackChannel{
		name:     "slack",
		enabled:  webhook != "",
		webhook:  webhook,
		channel:  channel,
		username: username,
	}
}

func (c *SlackChannel) Send(ctx context.Context, notification *Notification) error {
	if !c.enabled {
		return fmt.Errorf("slack channel disabled")
	}

	payload := map[string]interface{}{
		"channel":    c.channel,
		"username":   c.username,
		"text":       fmt.Sprintf("*%s*\n%s", notification.Title, notification.Message),
		"icon_emoji": c.getIconForType(notification.Type),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %v", err)
	}

	resp, err := http.Post(c.webhook, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send to slack: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *SlackChannel) GetName() string {
	return c.name
}

func (c *SlackChannel) IsEnabled() bool {
	return c.enabled
}

func (c *SlackChannel) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"webhook":  c.webhook,
		"channel":  c.channel,
		"username": c.username,
	}
}

func (c *SlackChannel) getIconForType(notificationType NotificationType) string {
	switch notificationType {
	case NotificationTypeSuccess:
		return ":white_check_mark:"
	case NotificationTypeWarning:
		return ":warning:"
	case NotificationTypeError:
		return ":x:"
	case NotificationTypeAlert:
		return ":rotating_light:"
	default:
		return ":information_source:"
	}
}

// EmailChannel implements notification channel for Email
type EmailChannel struct {
	name       string
	enabled    bool
	smtpServer string
	port       int
	username   string
	password   string
	from       string
}

func NewEmailChannel(smtpServer string, port int, username, password, from string) *EmailChannel {
	return &EmailChannel{
		name:       "email",
		enabled:    smtpServer != "" && username != "" && password != "",
		smtpServer: smtpServer,
		port:       port,
		username:   username,
		password:   password,
		from:       from,
	}
}

func (c *EmailChannel) Send(ctx context.Context, notification *Notification) error {
	if !c.enabled {
		return fmt.Errorf("email channel disabled")
	}

	// Simple email sending implementation
	// In production, use a proper email library

	to := "" // Would come from notification metadata
	if to == "" {
		return fmt.Errorf("no recipient specified")
	}

	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n",
		c.from, to, notification.Title, notification.Message)

	auth := smtp.PlainAuth("", c.username, c.password, c.smtpServer)
	addr := fmt.Sprintf("%s:%d", c.smtpServer, c.port)

	return smtp.SendMail(addr, auth, c.from, []string{to}, []byte(msg))
}

func (c *EmailChannel) GetName() string {
	return c.name
}

func (c *EmailChannel) IsEnabled() bool {
	return c.enabled
}

func (c *EmailChannel) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"smtp_server": c.smtpServer,
		"port":        c.port,
		"username":    c.username,
		"from":        c.from,
	}
}

// DiscordChannel implements notification channel for Discord
type DiscordChannel struct {
	name    string
	enabled bool
	webhook string
}

func NewDiscordChannel(webhook string) *DiscordChannel {
	return &DiscordChannel{
		name:    "discord",
		enabled: webhook != "",
		webhook: webhook,
	}
}

func (c *DiscordChannel) Send(ctx context.Context, notification *Notification) error {
	if !c.enabled {
		return fmt.Errorf("discord channel disabled")
	}

	payload := map[string]interface{}{
		"content": fmt.Sprintf("**%s**\n%s", notification.Title, notification.Message),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal discord payload: %v", err)
	}

	resp, err := http.Post(c.webhook, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send to discord: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("discord returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *DiscordChannel) GetName() string {
	return c.name
}

func (c *DiscordChannel) IsEnabled() bool {
	return c.enabled
}

func (c *DiscordChannel) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"webhook": c.webhook,
	}
}

// TelegramChannel implements notification channel for Telegram
type TelegramChannel struct {
	name    string
	enabled bool
	botToken string
	chatID   string
}

func NewTelegramChannel(botToken, chatID string) *TelegramChannel {
	return &TelegramChannel{
		name:     "telegram",
		enabled:  botToken != "" && chatID != "",
		botToken: botToken,
		chatID:   chatID,
	}
}

func (c *TelegramChannel) Send(ctx context.Context, notification *Notification) error {
	if !c.enabled {
		return fmt.Errorf("telegram channel disabled")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.botToken)

	payload := map[string]interface{}{
		"chat_id": c.chatID,
		"text":    fmt.Sprintf("*%s*\n\n%s", notification.Title, notification.Message),
		"parse_mode": "Markdown",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram payload: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send to telegram: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram returned status %d", resp.StatusCode)
	}

	log.Printf("✅ Notification sent via Telegram: %s", notification.Title)
	return nil
}

func (c *TelegramChannel) GetName() string {
	return c.name
}

func (c *TelegramChannel) IsEnabled() bool {
	return c.enabled
}

func (c *TelegramChannel) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"bot_token": c.botToken,
		"chat_id":   c.chatID,
	}
}

// YandexMessengerChannel implements notification channel for Yandex Messenger
type YandexMessengerChannel struct {
	name    string
	enabled bool
	token   string
	chatID  string
}

func NewYandexMessengerChannel(token, chatID string) *YandexMessengerChannel {
	return &YandexMessengerChannel{
		name:    "yandex_messenger",
		enabled: token != "" && chatID != "",
		token:   token,
		chatID:  chatID,
	}
}

func (c *YandexMessengerChannel) Send(ctx context.Context, notification *Notification) error {
	if !c.enabled {
		return fmt.Errorf("yandex messenger channel disabled")
	}

	url := "https://botapi.messenger.yandex.net/bot/v1/messages/send"

	payload := map[string]interface{}{
		"chat_id": c.chatID,
		"text":    fmt.Sprintf("**%s**\n\n%s", notification.Title, notification.Message),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal yandex payload: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("OAuth %s", c.token))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send to yandex messenger: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("yandex messenger returned status %d", resp.StatusCode)
	}

	log.Printf("✅ Notification sent via Yandex Messenger: %s", notification.Title)
	return nil
}

func (c *YandexMessengerChannel) GetName() string {
	return c.name
}

func (c *YandexMessengerChannel) IsEnabled() bool {
	return c.enabled
}

func (c *YandexMessengerChannel) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"token":   c.token,
		"chat_id": c.chatID,
	}
}

// MaxChannel implements notification channel for Max (enterprise communication platform)
type MaxChannel struct {
	name     string
	enabled  bool
	apiKey   string
	endpoint string
	roomID   string
}

func NewMaxChannel(apiKey, endpoint, roomID string) *MaxChannel {
	return &MaxChannel{
		name:     "max",
		enabled:  apiKey != "" && endpoint != "" && roomID != "",
		apiKey:   apiKey,
		endpoint: endpoint,
		roomID:   roomID,
	}
}

func (c *MaxChannel) Send(ctx context.Context, notification *Notification) error {
	if !c.enabled {
		return fmt.Errorf("max channel disabled")
	}

	url := fmt.Sprintf("%s/api/v1/rooms/%s/messages", c.endpoint, c.roomID)

	payload := map[string]interface{}{
		"text": fmt.Sprintf("**%s**\n\n%s", notification.Title, notification.Message),
		"type": "text",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal max payload: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send to max: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("max returned status %d", resp.StatusCode)
	}

	log.Printf("✅ Notification sent via Max: %s", notification.Title)
	return nil
}

func (c *MaxChannel) GetName() string {
	return c.name
}

func (c *MaxChannel) IsEnabled() bool {
	return c.enabled
}

func (c *MaxChannel) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"api_key":  c.apiKey,
		"endpoint": c.endpoint,
		"room_id":  c.roomID,
	}
}
