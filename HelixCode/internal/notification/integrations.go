package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// PagerDutyChannel implements PagerDuty Event API v2
type PagerDutyChannel struct {
	name           string
	enabled        bool
	integrationKey string
	apiURL         string
}

// NewPagerDutyChannel creates a new PagerDuty channel
func NewPagerDutyChannel(integrationKey string) *PagerDutyChannel {
	return &PagerDutyChannel{
		name:           "pagerduty",
		enabled:        integrationKey != "",
		integrationKey: integrationKey,
		apiURL:         "https://events.pagerduty.com/v2/enqueue",
	}
}

func (c *PagerDutyChannel) Send(ctx context.Context, notification *Notification) error {
	if !c.enabled {
		return fmt.Errorf("pagerduty channel disabled")
	}

	payload := map[string]interface{}{
		"routing_key":  c.integrationKey,
		"event_action": "trigger",
		"payload": map[string]interface{}{
			"summary":  notification.Title,
			"source":   "helixcode",
			"severity": c.getSeverity(notification.Type),
			"custom_details": map[string]interface{}{
				"message":  notification.Message,
				"metadata": notification.Metadata,
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal pagerduty payload: %v", err)
	}

	resp, err := http.Post(c.apiURL, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send to pagerduty: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("pagerduty returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *PagerDutyChannel) GetName() string {
	return c.name
}

func (c *PagerDutyChannel) IsEnabled() bool {
	return c.enabled
}

func (c *PagerDutyChannel) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"integration_key": "****" + c.integrationKey[len(c.integrationKey)-4:],
		"api_url":         c.apiURL,
	}
}

func (c *PagerDutyChannel) getSeverity(notifType NotificationType) string {
	switch notifType {
	case NotificationTypeAlert:
		return "critical"
	case NotificationTypeError:
		return "error"
	case NotificationTypeWarning:
		return "warning"
	default:
		return "info"
	}
}

// JiraChannel implements Jira issue creation
type JiraChannel struct {
	name      string
	enabled   bool
	baseURL   string
	email     string
	apiToken  string
	projectKey string
}

// NewJiraChannel creates a new Jira channel
func NewJiraChannel(baseURL, email, apiToken, projectKey string) *JiraChannel {
	return &JiraChannel{
		name:       "jira",
		enabled:    baseURL != "" && email != "" && apiToken != "" && projectKey != "",
		baseURL:    baseURL,
		email:      email,
		apiToken:   apiToken,
		projectKey: projectKey,
	}
}

func (c *JiraChannel) Send(ctx context.Context, notification *Notification) error {
	if !c.enabled {
		return fmt.Errorf("jira channel disabled")
	}

	// Create Jira issue
	payload := map[string]interface{}{
		"fields": map[string]interface{}{
			"project": map[string]string{
				"key": c.projectKey,
			},
			"summary":     notification.Title,
			"description": notification.Message,
			"issuetype": map[string]string{
				"name": "Task",
			},
			"priority": map[string]string{
				"name": c.getPriority(notification.Priority),
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal jira payload: %v", err)
	}

	url := fmt.Sprintf("%s/rest/api/3/issue", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.SetBasicAuth(c.email, c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send to jira: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("jira returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *JiraChannel) GetName() string {
	return c.name
}

func (c *JiraChannel) IsEnabled() bool {
	return c.enabled
}

func (c *JiraChannel) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"base_url":    c.baseURL,
		"email":       c.email,
		"project_key": c.projectKey,
	}
}

func (c *JiraChannel) getPriority(priority NotificationPriority) string {
	switch priority {
	case NotificationPriorityUrgent:
		return "Highest"
	case NotificationPriorityHigh:
		return "High"
	case NotificationPriorityMedium:
		return "Medium"
	default:
		return "Low"
	}
}

// GitHubIssuesChannel implements GitHub issue creation
type GitHubIssuesChannel struct {
	name   string
	enabled bool
	token  string
	owner  string
	repo   string
	apiURL string
}

// NewGitHubIssuesChannel creates a new GitHub Issues channel
func NewGitHubIssuesChannel(token, owner, repo string) *GitHubIssuesChannel {
	return &GitHubIssuesChannel{
		name:    "github",
		enabled: token != "" && owner != "" && repo != "",
		token:   token,
		owner:   owner,
		repo:    repo,
		apiURL:  "https://api.github.com",
	}
}

func (c *GitHubIssuesChannel) Send(ctx context.Context, notification *Notification) error {
	if !c.enabled {
		return fmt.Errorf("github channel disabled")
	}

	// Create GitHub issue
	payload := map[string]interface{}{
		"title": notification.Title,
		"body":  notification.Message,
		"labels": []string{
			string(notification.Type),
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal github payload: %v", err)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/issues", c.apiURL, c.owner, c.repo)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send to github: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("github returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *GitHubIssuesChannel) GetName() string {
	return c.name
}

func (c *GitHubIssuesChannel) IsEnabled() bool {
	return c.enabled
}

func (c *GitHubIssuesChannel) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"owner":   c.owner,
		"repo":    c.repo,
		"api_url": c.apiURL,
	}
}
