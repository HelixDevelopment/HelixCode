# Email (SMTP) Integration Setup Guide

This guide will walk you through setting up email notifications for HelixCode using SMTP.

## Prerequisites

- SMTP server access (Gmail, Office 365, or custom server)
- Email account credentials or app password

## Supported Email Providers

- **Gmail** (recommended for personal use)
- **Office 365 / Outlook**
- **Custom SMTP servers**
- **SendGrid, Mailgun** (transactional email services)

## Setup for Gmail

### Step 1: Enable 2-Factor Authentication

1. Go to https://myaccount.google.com/security
2. Enable **"2-Step Verification"**

### Step 2: Generate App Password

1. Go to https://myaccount.google.com/apppasswords
2. Select **"Mail"** as the app
3. Select your device or enter custom name: `HelixCode`
4. Click **"Generate"**
5. **Copy the 16-character password** (e.g., `abcd efgh ijkl mnop`)

### Step 3: Configure HelixCode

Add to your `.env` file:

```bash
HELIX_EMAIL_SMTP_SERVER=smtp.gmail.com
HELIX_EMAIL_SMTP_PORT=587
HELIX_EMAIL_USERNAME=your-email@gmail.com
HELIX_EMAIL_PASSWORD=abcdefghijklmnop  # App password (no spaces)
HELIX_EMAIL_FROM=your-email@gmail.com
HELIX_EMAIL_RECIPIENTS=admin@example.com,team@example.com
```

## Setup for Office 365 / Outlook

### Step 1: Get Credentials

- Use your Office 365 email and password
- Ensure SMTP is enabled for your account (check with admin)

### Step 2: Configure HelixCode

Add to your `.env` file:

```bash
HELIX_EMAIL_SMTP_SERVER=smtp.office365.com
HELIX_EMAIL_SMTP_PORT=587
HELIX_EMAIL_USERNAME=your-email@company.com
HELIX_EMAIL_PASSWORD=your-password
HELIX_EMAIL_FROM=your-email@company.com
HELIX_EMAIL_RECIPIENTS=admin@company.com
```

## Setup for Custom SMTP Server

### Step 1: Get SMTP Details

Contact your email administrator for:
- SMTP server address
- SMTP port (usually 587 for TLS, 465 for SSL, 25 for unencrypted)
- Authentication credentials

### Step 2: Configure HelixCode

Add to your `.env` file:

```bash
HELIX_EMAIL_SMTP_SERVER=smtp.yourcompany.com
HELIX_EMAIL_SMTP_PORT=587
HELIX_EMAIL_USERNAME=notifications@yourcompany.com
HELIX_EMAIL_PASSWORD=your-password
HELIX_EMAIL_FROM=HelixCode <notifications@yourcompany.com>
HELIX_EMAIL_RECIPIENTS=team@yourcompany.com
```

## Configure config.yaml

Edit `config/config.yaml`:

```yaml
notifications:
  enabled: true
  channels:
    email:
      enabled: true
      smtp:
        server: "${HELIX_EMAIL_SMTP_SERVER}"
        port: 587
        username: "${HELIX_EMAIL_USERNAME}"
        password: "${HELIX_EMAIL_PASSWORD}"
        from: "HelixCode <${HELIX_EMAIL_FROM}>"
        tls: true
      recipients:
        default: ["admin@example.com"]
        critical: ["admin@example.com", "oncall@example.com"]
      timeout: 30
```

## Test Integration

### Using HelixCode CLI (if available)

```bash
helix notify test --channel email --message "Test email notification"
```

### Using API

```bash
curl -X POST http://localhost:8080/api/v1/notifications/test \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "channel": "email",
    "title": "Test Email Notification",
    "message": "Testing email integration from HelixCode",
    "type": "info",
    "metadata": {
      "recipients": ["admin@example.com"]
    }
  }'
```

### Using Go Code

```go
import "dev.helix.code/internal/notification"

// Create notification engine
engine := notification.NewNotificationEngine()

// Register email channel
emailChannel := notification.NewEmailChannel(
    "smtp.gmail.com",
    587,
    "your-email@gmail.com",
    "app-password",
    "HelixCode <your-email@gmail.com>",
)
engine.RegisterChannel(emailChannel)

// Send test notification
testNotif := &notification.Notification{
    Title:   "Test Email",
    Message: "Testing email integration",
    Type:    notification.NotificationTypeInfo,
    Metadata: map[string]interface{}{
        "recipients": []string{"admin@example.com"},
    },
}
err := engine.SendDirect(context.Background(), testNotif, []string{"email"})
```

## Recipient Management

### Single Recipient

```go
Metadata: map[string]interface{}{
    "recipient": "admin@example.com",  // Note: singular "recipient"
}
```

### Multiple Recipients

```go
Metadata: map[string]interface{}{
    "recipients": []string{  // Note: plural "recipients"
        "admin@example.com",
        "team@example.com",
        "oncall@example.com",
    },
}
```

### Using Default Recipients from Config

Configure in `config.yaml`:

```yaml
email:
  recipients:
    default: ["admin@example.com", "team@example.com"]
    critical: ["admin@example.com", "oncall@example.com", "cto@example.com"]
```

Then use rules to select recipient lists:

```yaml
notifications:
  rules:
    - name: "Critical errors to oncall"
      condition: "type==error"
      channels: ["email"]
      priority: urgent
      template: "critical_alert"
```

## Common SMTP Ports

| Port | Protocol | Description | Recommended |
|------|----------|-------------|-------------|
| 25 | SMTP (unencrypted) | Legacy, often blocked by ISPs | ❌ No |
| 465 | SMTPS (SSL) | Deprecated but still works | ⚠️ Use if required |
| 587 | SMTP + STARTTLS | Modern, secure | ✅ **Recommended** |

## Troubleshooting

### Error: "Authentication failed"
- **Gmail:** Ensure you're using an app password, not your account password
- **Office 365:** Verify your credentials and that SMTP is enabled
- **Custom:** Check username and password with your admin

### Error: "Connection refused"
- **Cause:** Incorrect SMTP server or port
- **Solution:** Verify SMTP server address and port number
- **Gmail:** Use `smtp.gmail.com:587`
- **Office 365:** Use `smtp.office365.com:587`

### Error: "TLS handshake failed"
- **Cause:** TLS/SSL configuration issue
- **Solution:**
  - Try port 465 with `tls: false` (uses SSL instead)
  - Verify server supports STARTTLS on port 587

### Error: "Recipient not specified"
- **Cause:** Missing recipients in notification metadata
- **Solution:** Always include recipients in metadata:
  ```go
  Metadata: map[string]interface{}{
      "recipients": []string{"admin@example.com"},
  }
  ```

### Emails going to spam
- **Cause:** Missing SPF, DKIM, DMARC records
- **Solution:**
  - For Gmail/Office365: Emails should not go to spam
  - For custom domains: Configure SPF, DKIM, DMARC DNS records
  - Add "HelixCode" to subject to help filtering

### Rate limiting
- **Gmail:** 500 emails/day (free), 2000/day (Workspace)
- **Office 365:** Varies by plan (typically 10,000/day)
- **Solution:** Use dedicated transactional email service for high volume

## Advanced Configuration

### HTML Email Templates (Future Feature)

Future versions will support HTML templates:

```yaml
email:
  templates:
    enabled: true
    html: true
    path: "/etc/helixcode/email-templates"
```

### Attachments (Future Feature)

Future versions will support file attachments:

```go
Metadata: map[string]interface{}{
    "recipients": []string{"admin@example.com"},
    "attachments": []string{
        "/tmp/logs/error.log",
        "/tmp/reports/summary.pdf",
    },
}
```

### Email Priority

Set email priority/importance:

```yaml
email:
  priority_header: true  # Adds X-Priority header
```

## Security Best Practices

1. **Never commit passwords to version control**
   - Use environment variables
   - Add `.env` to `.gitignore`

2. **Use app passwords, not account passwords**
   - Gmail: Use app passwords
   - Office 365: Use app passwords where supported

3. **Enable TLS/SSL**
   - Always use encrypted connections
   - Prefer port 587 with STARTTLS

4. **Rotate credentials regularly**
   - Change passwords periodically
   - Regenerate app passwords

5. **Use dedicated email accounts**
   - Don't use personal email for system notifications
   - Create dedicated accounts like `notifications@company.com`

6. **Implement SPF/DKIM/DMARC** (for custom domains)
   - Prevents email spoofing
   - Improves deliverability
   - Reduces spam classification

## Using Transactional Email Services

For high-volume or production use, consider dedicated services:

### SendGrid

```bash
HELIX_EMAIL_SMTP_SERVER=smtp.sendgrid.net
HELIX_EMAIL_SMTP_PORT=587
HELIX_EMAIL_USERNAME=apikey
HELIX_EMAIL_PASSWORD=your-sendgrid-api-key
```

### Mailgun

```bash
HELIX_EMAIL_SMTP_SERVER=smtp.mailgun.org
HELIX_EMAIL_SMTP_PORT=587
HELIX_EMAIL_USERNAME=postmaster@your-domain.mailgun.org
HELIX_EMAIL_PASSWORD=your-mailgun-password
```

### Amazon SES

```bash
HELIX_EMAIL_SMTP_SERVER=email-smtp.us-east-1.amazonaws.com
HELIX_EMAIL_SMTP_PORT=587
HELIX_EMAIL_USERNAME=your-ses-smtp-username
HELIX_EMAIL_PASSWORD=your-ses-smtp-password
```

## Resources

- [Gmail SMTP Settings](https://support.google.com/mail/answer/7126229)
- [Office 365 SMTP Settings](https://support.microsoft.com/en-us/office/pop-imap-and-smtp-settings-8361e398-8af4-4e97-b147-6c6c4ac95353)
- [SMTP RFC Documentation](https://tools.ietf.org/html/rfc5321)
- [HelixCode Notification API Documentation](../API.md)

## Support

If you encounter issues:
1. Check the [Troubleshooting](#troubleshooting) section
2. Verify SMTP settings with your email provider
3. Check HelixCode logs for detailed error messages
4. Test SMTP connection with tools like `telnet` or `swaks`
5. Open an issue on GitHub
