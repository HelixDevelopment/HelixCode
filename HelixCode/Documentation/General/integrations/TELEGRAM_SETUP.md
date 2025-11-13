# Telegram Integration Setup Guide

This guide will walk you through setting up Telegram notifications for HelixCode.

## Prerequisites

- Telegram account
- Telegram app (mobile or desktop)

## Step 1: Create Telegram Bot

1. Open Telegram and search for `@BotFather`
2. Start a chat with BotFather
3. Send the command: `/newbot`
4. BotFather will ask for a name for your bot
   - Enter: `HelixCode Notifications`
5. BotFather will ask for a username (must end in `bot`)
   - Enter: `helixcode_notifications_bot` or similar
6. **Save the bot token** - it looks like: `123456789:ABCdefGHIjklMNOpqrsTUVwxyz`

**Important:** Keep this token secret! It's like a password for your bot.

## Step 2: Get Your Chat ID

You need to know where to send notifications. You can send to:
- Your personal chat (DM with the bot)
- A group chat
- A channel

### Option A: Using @userinfobot (Easiest)

1. Search for `@userinfobot` in Telegram
2. Start a chat
3. It will display your chat ID (e.g., `123456789`)
4. **Copy this number** - this is your chat ID

### Option B: Manual Method

1. Start a chat with your newly created bot (search for its username)
2. Send any message to the bot (e.g., "Hello")
3. Open this URL in your browser:
   ```
   https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates
   ```
   Replace `<YOUR_BOT_TOKEN>` with your actual bot token
4. Look for `"chat":{"id":123456789}` in the JSON response
5. **Copy the ID number**

### For Group Notifications

1. Create a Telegram group
2. Add your bot to the group (search for it and add as member)
3. Send a message in the group
4. Use the `/getUpdates` URL method above
5. Group chat IDs start with a minus sign (e.g., `-987654321`)

## Step 3: Configure HelixCode

### Option A: Using Environment Variables

Add to your `.env` file:

```bash
HELIX_TELEGRAM_BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrsTUVwxyz
HELIX_TELEGRAM_CHAT_ID=123456789
```

### Option B: Using config.yaml

Edit `config/config.yaml`:

```yaml
notifications:
  enabled: true
  channels:
    telegram:
      enabled: true
      bot_token: "${HELIX_TELEGRAM_BOT_TOKEN}"
      chat_id: "${HELIX_TELEGRAM_CHAT_ID}"
      timeout: 10
```

## Step 4: Test Integration

### Using HelixCode CLI (if available)

```bash
helix notify test --channel telegram --message "Test notification"
```

### Using API

```bash
curl -X POST http://localhost:8080/api/v1/notifications/test \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "channel": "telegram",
    "title": "Test Notification",
    "message": "Testing Telegram integration from HelixCode",
    "type": "info"
  }'
```

### Using Go Code

```go
import "dev.helix.code/internal/notification"

// Create notification engine
engine := notification.NewNotificationEngine()

// Register Telegram channel
telegramChannel := notification.NewTelegramChannel(
    "123456789:ABCdefGHIjklMNOpqrsTUVwxyz",
    "123456789",
)
engine.RegisterChannel(telegramChannel)

// Send test notification
testNotif := &notification.Notification{
    Title:   "Test Notification",
    Message: "Testing Telegram integration",
    Type:    notification.NotificationTypeInfo,
    Metadata: map[string]interface{}{
        "task_id": "task-123",
    },
}
err := engine.SendDirect(context.Background(), testNotif, []string{"telegram"})
```

## Step 5: Configure Notification Rules

Edit `config.yaml` to define when to send Telegram notifications:

```yaml
notifications:
  rules:
    - name: "All failures to Telegram"
      condition: "type==error"
      channels: ["telegram"]
      priority: high
      enabled: true

    - name: "Critical alerts to Telegram"
      condition: "type==alert"
      channels: ["telegram", "slack", "email"]
      priority: urgent
      enabled: true
```

## Message Formatting

HelixCode uses HTML formatting for Telegram messages. Messages include:

- **Bold title**
- Message body
- Metadata (if provided) as formatted details

Example output:
```
**Task Failed**

Task execution failed on worker-123

Details:
• task_id: task-789
• worker_id: worker-123
• error: Connection timeout
```

## Troubleshooting

### Error: "Unauthorized"
- **Cause:** Bot token is incorrect
- **Solution:** Verify the bot token from BotFather

### Error: "Chat not found"
- **Cause:** Invalid chat ID or you haven't messaged the bot yet
- **Solution:**
  1. Send a message to your bot first
  2. Verify the chat ID is correct
  3. For groups, make sure the bot is still a member

### Error: "Bot was blocked by the user"
- **Cause:** You blocked the bot in Telegram
- **Solution:** Unblock the bot in Telegram settings

### No notifications appearing
- **Cause:** Channel might be disabled
- **Solution:** Check `enabled: true` in `config.yaml`

### Bot doesn't respond
- **Cause:** The bot only sends notifications, it doesn't respond to messages
- **Solution:** This is expected behavior - HelixCode bots are send-only

## Advanced Features

### HTML Formatting

Telegram supports HTML formatting. You can include:

```html
<b>bold text</b>
<i>italic text</i>
<code>monospace code</code>
<pre>code block</pre>
<a href="URL">link text</a>
```

### Metadata Display

Automatically includes metadata in notifications:

```go
notification := &notification.Notification{
    Title:   "Deployment Complete",
    Message: "Application deployed successfully",
    Metadata: map[string]interface{}{
        "environment": "production",
        "version":     "v2.1.0",
        "duration":    "45s",
    },
}
```

Output:
```
**Deployment Complete**

Application deployed successfully

Details:
• environment: production
• version: v2.1.0
• duration: 45s
```

### Future: Interactive Buttons

Future versions will support inline keyboards with action buttons:
- "Retry Task"
- "View Logs"
- "Mark as Resolved"

### Future: Media Support

Future versions will support:
- Sending images (charts, graphs)
- Sending documents (log files)
- Sending stickers for fun notifications

## Security Best Practices

1. **Never commit bot tokens to version control**
   - Use environment variables
   - Add `.env` to `.gitignore`

2. **Rotate bot tokens if compromised**
   - Use BotFather `/revoke` command
   - Generate new token with `/token`

3. **Use different bots for different environments**
   - Production bot → production chat
   - Staging bot → staging chat
   - Development bot → dev chat

4. **Limit bot permissions**
   - Bots only need send message permission
   - Don't make bots admins unless necessary

## Using with Groups vs Personal Chats

### Personal Chat (DM)
- **Pros:** Private, direct notifications
- **Cons:** Only you receive them
- **Use case:** Personal projects, development

### Group Chat
- **Pros:** Team visibility, collaboration
- **Cons:** Can get noisy
- **Use case:** Team projects, shared responsibility

### Telegram Channel
- **Pros:** Broadcast to many, read-only for subscribers
- **Cons:** One-way communication
- **Use case:** Status updates, announcements

## Resources

- [Telegram Bot API Documentation](https://core.telegram.org/bots/api)
- [BotFather Commands](https://core.telegram.org/bots#6-botfather)
- [Telegram Bot Features](https://core.telegram.org/bots/features)
- [HelixCode Notification API Documentation](../API.md)

## Support

If you encounter issues:
1. Check the [Troubleshooting](#troubleshooting) section
2. Verify bot token and chat ID are correct
3. Check HelixCode logs for error messages
4. Open an issue on GitHub
