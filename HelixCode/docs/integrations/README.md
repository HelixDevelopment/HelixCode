# HelixCode Integrations

HelixCode supports multiple notification and integration channels to keep you informed about your development workflows.

## Supported Integrations

### Communication Platforms

#### 1. Slack
Real-time team notifications via webhooks.

**Features:**
- Webhook integration
- Custom channels and usernames
- Type-based icon emojis
- Rich text formatting

**[Setup Guide](./SLACK_SETUP.md)**

---

#### 2. Telegram
Bot-powered notifications for individuals and teams.

**Features:**
- Telegram Bot API integration
- HTML message formatting
- Personal chats, groups, and channels
- Automatic metadata display

**[Setup Guide](./TELEGRAM_SETUP.md)**

---

#### 3. Email (SMTP)
Enterprise-grade email notifications.

**Features:**
- Support for Gmail, Office 365, custom SMTP
- TLS/SSL encryption
- Multiple recipients
- Configurable from addresses

**[Setup Guide](./EMAIL_SETUP.md)**

---

#### 4. Discord
Gaming-first webhook notifications.

**Features:**
- Webhook integration
- Markdown formatting
- Server and channel support
- Built-in rate limiting

**Setup:** Similar to Slack (webhook-based)

---

## Quick Start

1. **Choose your integration(s)** - Select one or more notification channels
2. **Follow the setup guide** - Each integration has a detailed setup guide
3. **Configure HelixCode** - Update your `.env` or `config.yaml`
4. **Test the integration** - Send a test notification
5. **Configure rules** - Define when to send notifications

## Configuration Overview

### Environment Variables

```bash
# Slack
HELIX_SLACK_WEBHOOK_URL=https://hooks.slack.com/services/...

# Telegram
HELIX_TELEGRAM_BOT_TOKEN=123456789:ABC...
HELIX_TELEGRAM_CHAT_ID=123456789

# Email
HELIX_EMAIL_SMTP_SERVER=smtp.gmail.com
HELIX_EMAIL_SMTP_PORT=587
HELIX_EMAIL_USERNAME=your-email@gmail.com
HELIX_EMAIL_PASSWORD=app-password
HELIX_EMAIL_FROM=your-email@gmail.com

# Discord
HELIX_DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/...
```

### config.yaml

```yaml
notifications:
  enabled: true

  rules:
    - name: "Critical Failures"
      condition: "type==error"
      channels: ["slack", "telegram", "email"]
      priority: urgent
      enabled: true

  channels:
    slack:
      enabled: true
      webhook_url: "${HELIX_SLACK_WEBHOOK_URL}"
      channel: "#helix-notifications"

    telegram:
      enabled: true
      bot_token: "${HELIX_TELEGRAM_BOT_TOKEN}"
      chat_id: "${HELIX_TELEGRAM_CHAT_ID}"

    email:
      enabled: true
      smtp:
        server: "${HELIX_EMAIL_SMTP_SERVER}"
        port: 587
        username: "${HELIX_EMAIL_USERNAME}"
        password: "${HELIX_EMAIL_PASSWORD}"
```

## Notification Types

HelixCode supports different notification types with appropriate visual indicators:

| Type | Description | Use Case |
|------|-------------|----------|
| `info` | General information | Status updates, informational messages |
| `success` | Successful operations | Task completed, workflow succeeded |
| `warning` | Non-critical issues | Performance degradation, warnings |
| `error` | Failures and errors | Task failed, execution error |
| `alert` | Critical system alerts | System down, critical failure |

## Notification Rules

Define when to send notifications based on:

- **Event type** - Task failure, workflow completion, etc.
- **Priority level** - Low, medium, high, urgent
- **Metadata conditions** - Custom conditions based on event data

Example rules:

```yaml
notifications:
  rules:
    # Send critical errors to all channels
    - name: "Critical Errors"
      condition: "type==error AND priority==urgent"
      channels: ["slack", "telegram", "email"]
      priority: urgent
      enabled: true

    # Send successes only to Slack
    - name: "Success Notifications"
      condition: "type==success"
      channels: ["slack"]
      priority: low
      enabled: true

    # Send all alerts to Telegram and Email
    - name: "System Alerts"
      condition: "type==alert"
      channels: ["telegram", "email"]
      priority: high
      enabled: true
```

## Testing Integrations

### Using CLI (if available)

```bash
helix notify test --channel slack --message "Test notification"
helix notify test --channel telegram --message "Test notification"
helix notify test --channel email --message "Test notification"
```

### Using API

```bash
curl -X POST http://localhost:8080/api/v1/notifications/test \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "channel": "slack",
    "title": "Test Notification",
    "message": "Testing integration",
    "type": "info"
  }'
```

### Using Go Code

```go
import "dev.helix.code/internal/notification"

engine := notification.NewNotificationEngine()

// Register channels
slackChannel := notification.NewSlackChannel(webhookURL, "#channel", "bot")
telegramChannel := notification.NewTelegramChannel(botToken, chatID)
emailChannel := notification.NewEmailChannel(server, port, user, pass, from)

engine.RegisterChannel(slackChannel)
engine.RegisterChannel(telegramChannel)
engine.RegisterChannel(emailChannel)

// Send notification
notif := &notification.Notification{
    Title:   "Test",
    Message: "Testing integrations",
    Type:    notification.NotificationTypeInfo,
}
err := engine.SendDirect(ctx, notif, []string{"slack", "telegram", "email"})
```

## Roadmap

### Planned Integrations

**High Priority:**
- Generic Webhooks (custom integrations)
- Microsoft Teams (Workflows API)

**Medium Priority:**
- PagerDuty (incident management)
- Jira (issue tracking)
- GitHub Issues (repository integration)

**Future Considerations:**
- Mattermost (self-hosted Slack alternative)
- Firebase Cloud Messaging (mobile notifications)
- Twilio SMS (critical alerts)

See the [Integration Report](../../NOTIFICATION_INTEGRATION_REPORT.md) for detailed implementation plans.

## Support

For help with integrations:

1. **Check setup guides** - Each integration has detailed instructions
2. **Review troubleshooting sections** - Common issues and solutions
3. **Check logs** - HelixCode logs contain detailed error messages
4. **Open an issue** - Report bugs or request help on GitHub

## Contributing

Want to add a new integration?

1. Implement the `NotificationChannel` interface
2. Add comprehensive tests (unit + integration)
3. Create a setup guide
4. Update this README
5. Submit a pull request

See the implementation report for architecture guidelines.
