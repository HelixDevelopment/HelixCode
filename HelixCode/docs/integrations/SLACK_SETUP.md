# Slack Integration Setup Guide

This guide will walk you through setting up Slack notifications for HelixCode.

## Prerequisites

- Slack workspace admin access
- Ability to create incoming webhooks

## Step 1: Create Slack App

1. Go to https://api.slack.com/apps
2. Click **"Create New App"**
3. Select **"From scratch"**
4. Enter app name: `HelixCode Notifications`
5. Select your workspace
6. Click **"Create App"**

## Step 2: Enable Incoming Webhooks

1. In your app settings, navigate to **"Incoming Webhooks"** (left sidebar)
2. Toggle **"Activate Incoming Webhooks"** to **On**
3. Scroll down and click **"Add New Webhook to Workspace"**
4. Select the channel where you want notifications (e.g., `#helix-notifications`)
5. Click **"Allow"**

## Step 3: Copy Webhook URL

1. You'll see your webhook URL listed (it looks like: `https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXX`)
2. Click **"Copy"** to copy the webhook URL

## Step 4: Configure HelixCode

### Option A: Using Environment Variables

Add to your `.env` file:

```bash
HELIX_SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/WEBHOOK/URL
```

### Option B: Using config.yaml

Edit `config/config.yaml`:

```yaml
notifications:
  enabled: true
  channels:
    slack:
      enabled: true
      webhook_url: "${HELIX_SLACK_WEBHOOK_URL}"  # Or paste URL directly (not recommended for security)
      channel: "#helix-notifications"
      username: "HelixCode Bot"
      timeout: 10
```

## Step 5: Test Integration

### Using HelixCode CLI (if available)

```bash
helix notify test --channel slack --message "Test notification"
```

### Using API

```bash
curl -X POST http://localhost:8080/api/v1/notifications/test \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "channel": "slack",
    "title": "Test Notification",
    "message": "Testing Slack integration from HelixCode",
    "type": "info"
  }'
```

### Using Go Code

```go
import "dev.helix.code/internal/notification"

// Create notification engine
engine := notification.NewNotificationEngine()

// Register Slack channel
slackChannel := notification.NewSlackChannel(
    "https://hooks.slack.com/services/...",
    "#helix-notifications",
    "HelixCode Bot",
)
engine.RegisterChannel(slackChannel)

// Send test notification
testNotif := &notification.Notification{
    Title:   "Test Notification",
    Message: "Testing Slack integration",
    Type:    notification.NotificationTypeInfo,
}
err := engine.SendDirect(context.Background(), testNotif, []string{"slack"})
```

## Step 6: Configure Notification Rules

Edit `config.yaml` to define when to send Slack notifications:

```yaml
notifications:
  rules:
    - name: "Critical Failures to Slack"
      condition: "type==error"
      channels: ["slack"]
      priority: urgent
      enabled: true

    - name: "Success Notifications"
      condition: "type==success"
      channels: ["slack"]
      priority: medium
      enabled: true
```

## Notification Types & Icons

HelixCode automatically selects appropriate icons for each notification type:

| Type | Icon | Example Use Case |
|------|------|------------------|
| `info` | ℹ️ `:information_source:` | General information |
| `success` | ✅ `:white_check_mark:` | Task completed successfully |
| `warning` | ⚠️ `:warning:` | Non-critical issues |
| `error` | ❌ `:x:` | Task failures, errors |
| `alert` | 🚨 `:rotating_light:` | Critical system alerts |

## Troubleshooting

### Error: "invalid_token"
- **Cause:** Webhook URL is incorrect or invalid
- **Solution:** Verify the webhook URL is copied correctly from Slack

### Error: "channel_not_found"
- **Cause:** The specified channel doesn't exist or bot lacks access
- **Solution:** Verify the channel exists and the webhook is configured for it

### No notifications appearing
- **Cause:** Channel might be disabled in config
- **Solution:** Check `enabled: true` in `config.yaml`

### Rate limiting issues
- **Cause:** Slack has a rate limit of 1 message per second per webhook
- **Solution:** HelixCode automatically handles rate limiting (future feature)

## Advanced Configuration

### Custom Username & Icon

```yaml
slack:
  username: "My Custom Bot"
  # Note: icon_emoji is set automatically based on notification type
```

### Multiple Slack Channels

You can create multiple Slack channels for different purposes:

```go
// Production alerts
prodChannel := notification.NewSlackChannel(
    webhookProd,
    "#production-alerts",
    "Production Bot",
)

// Development notifications
devChannel := notification.NewSlackChannel(
    webhookDev,
    "#dev-notifications",
    "Dev Bot",
)
```

### Future: Slack Block Kit (Rich Formatting)

Future versions will support Slack's Block Kit for rich, interactive messages with:
- Structured layouts
- Action buttons
- Context blocks
- File attachments

## Security Best Practices

1. **Never commit webhook URLs to version control**
   - Use environment variables
   - Add `.env` to `.gitignore`

2. **Rotate webhook URLs regularly**
   - Delete old webhooks in Slack
   - Generate new ones periodically

3. **Use separate webhooks for different environments**
   - Production webhook → `#production-alerts`
   - Staging webhook → `#staging-alerts`
   - Development webhook → `#dev-notifications`

## Resources

- [Slack Incoming Webhooks Documentation](https://api.slack.com/messaging/webhooks)
- [Slack Block Kit Builder](https://api.slack.com/block-kit)
- [HelixCode Notification API Documentation](../API.md)

## Support

If you encounter issues:
1. Check the [Troubleshooting](#troubleshooting) section
2. Review HelixCode logs for error messages
3. Open an issue on GitHub
