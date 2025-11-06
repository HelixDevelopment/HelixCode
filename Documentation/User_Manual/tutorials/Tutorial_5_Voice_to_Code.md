# Tutorial 5: Voice-to-Code Workflow

**Duration**: 15 minutes
**Level**: Beginner

## Overview

Code hands-free using voice input:
- Setup Whisper transcription
- Voice commands
- Dictate code
- Voice-controlled refactoring

## Step 1: Setup Voice Input

```bash
# Requires OpenAI API key for Whisper
export OPENAI_API_KEY=sk-...

helixcode voice init

# Testing microphone... ✓
# Whisper API configured ✓
```

## Step 2: Voice Commands

```bash
helixcode voice

# Say: "Create a function to validate email addresses"
# HelixCode transcribes and generates code
```

**Generated**:

```go
func ValidateEmail(email string) bool {
    regex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
    return regex.MatchString(email)
}
```

## Step 3: Voice-Dictated Refactoring

```bash
# Say: "Refactor the authentication module to use dependency injection
#       Extract an interface called AuthProvider
#       Implement JWT provider
#       Add unit tests"

# HelixCode transcribes and executes plan
```

## Step 4: Code Review by Voice

```bash
helixcode voice review

# Say: "Review the user service for security issues"

# HelixCode analyzes and responds verbally
# TTS: "I found 3 potential issues:
#       1. SQL injection in GetUserByEmail
#       2. Missing input validation in CreateUser
#       3. Plain text password in logs"
```

## Step 5: Hands-Free Development

```bash
# Start voice session
helixcode voice session

# Voice commands:
# "Create REST API endpoint for user registration"
# "Add validation"
# "Write unit tests"
# "Commit with message: Add user registration"
# "Push to GitHub"
```

## Configuration

```yaml
tools:
  voice:
    enabled: true
    model: "whisper-1"
    language: "en"
    sample_rate: 16000
    silence_timeout: 2s

    # Wake word (optional)
    wake_word: "helix"

    # Text-to-speech for responses
    tts:
      enabled: true
      voice: "nova"
```

## Use Cases

- **Accessibility**: Developers with physical limitations
- **Multitasking**: Code while cooking, walking, etc.
- **Brainstorming**: Verbal architecture discussions
- **Pair Programming**: Voice-driven collaborative coding

## Results

- **Hands-Free**: Code without keyboard
- **Fast Input**: Speak faster than you type
- **Natural**: Conversational programming

---

Continue to [Tutorial 6: Multi-File Atomic Edits](Tutorial_6_Multi_File_Atomic_Edits.md)
