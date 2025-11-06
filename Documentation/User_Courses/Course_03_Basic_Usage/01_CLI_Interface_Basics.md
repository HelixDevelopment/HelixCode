# Chapter 1: CLI Interface Basics

**Duration:** 8 minutes
**Learning Objectives:**
- Understand the HelixCode command-line interface
- Learn essential commands and options
- Navigate the interactive session
- Use help and documentation features

---

## Video Script

[00:00 - 00:30] Introduction

Welcome to Course 3: Basic Usage and Workflows! You've installed HelixCode and understand the theory - now it's time to get hands-on. In this first chapter, we'll explore the CLI interface, learning the commands and options you'll use every day.

HelixCode's CLI is designed to be intuitive and powerful. If you're comfortable with command-line tools, you'll feel right at home. If you're new to CLI tools, don't worry - we'll take it step by step.

[00:30 - 01:30] Starting HelixCode

The most basic usage is simply running `helixcode` in your project directory:

```bash
cd /path/to/your/project
helixcode
```

This starts an interactive session with default settings. HelixCode scans your repository, builds a map, and waits for your input.

But you have many options to customize the startup. Let's look at the most common command-line flags:

**Specify a model:**
```bash
helixcode --model openai/gpt-4
helixcode --model anthropic/claude-3-opus
```

**Start in read-only mode:**
```bash
helixcode --read
```
This lets you ask questions and explore code without making changes. Perfect for code review or learning a new codebase.

**Start with specific files:**
```bash
helixcode src/main.py src/utils.py
```
This adds files to the context immediately, focusing HelixCode's attention.

**Show verbose output:**
```bash
helixcode --verbose
```
Useful for debugging or understanding what HelixCode is doing behind the scenes.

[01:30 - 03:00] Interactive Commands

Once in a HelixCode session, you have access to slash commands. These are special commands that start with `/` and control the session.

**Essential Commands:**

`/help` - Show all available commands
```
> /help
Available commands:
  /add <file>    - Add file to context
  /drop <file>   - Remove file from context
  /clear         - Clear conversation history
  /tokens        - Show token usage
  /quit          - Exit HelixCode
  ...
```

`/add` - Add files to the conversation context
```
> /add src/database.py
Added src/database.py to context
```

`/drop` - Remove files from context
```
> /drop src/tests/old_test.py
Removed src/tests/old_test.py from context
```

`/ls` - List files in current context
```
> /ls
Files in context:
  - src/main.py (234 lines)
  - src/utils.py (156 lines)
  - README.md (89 lines)
Total: 3 files, 479 lines
```

`/clear` - Clear conversation history (keeps files in context)
```
> /clear
Conversation history cleared
```

`/tokens` - Show token usage and cost
```
> /tokens
Current context: 2,450 tokens
Messages sent: 5
Total tokens used: 12,300
Estimated cost: $0.25
```

`/quit` or `/exit` - End the session
```
> /quit
Goodbye!
```

[03:00 - 04:30] Working with Files

File management is crucial in HelixCode. Let's see how to work with files effectively.

**Adding files:**

Add a single file:
```
> /add src/models/user.py
```

Add multiple files:
```
> /add src/models/*.py
```

Add a directory:
```
> /add src/controllers/
```

**Viewing file content:**

You don't need a special command - just ask:
```
> Show me the User class
```

HelixCode will display the relevant code from the context.

**Dropping files to manage context:**

When your context gets too large, drop files you're not actively working on:
```
> /drop src/models/legacy_*.py
```

**File wildcards and patterns:**

HelixCode understands glob patterns:
```
> /add src/**/*.test.js         # All test files
> /add src/components/*.tsx     # All TSX components
> /drop **/old_*.py             # Remove all old_ files
```

[04:30 - 05:45] Model and Provider Commands

You can check and change AI models during your session.

**List available models:**
```
> /models
Available models:
  OpenAI:
    - gpt-4 (8K context, $0.03/1K tokens)
    - gpt-4-32k (32K context, $0.06/1K tokens)
    - gpt-3.5-turbo (4K context, $0.002/1K tokens)
  Anthropic:
    - claude-3-opus (200K context, $0.015/1K tokens)
    - claude-3-sonnet (200K context, $0.003/1K tokens)
  ...
```

**Switch models:**
```
> /model anthropic/claude-3-opus
Switched to anthropic/claude-3-opus
```

**Check current configuration:**
```
> /config
Current configuration:
  Model: openai/gpt-4
  Provider: openai
  Auto-commit: disabled
  Read-only: false
  Context window: 8192 tokens
```

[05:45 - 06:45] Git Integration Commands

HelixCode integrates deeply with Git. Here are the git-related commands:

**Check repository status:**
```
> /git-status
On branch main
Your branch is up to date with 'origin/main'.

Changes not staged for commit:
  modified:   src/main.py
  modified:   src/utils.py
```

**View changes:**
```
> /diff
Showing uncommitted changes:
[displays git diff output]
```

**Undo last change:**
```
> /undo
Reverted last edit to src/main.py
```

**Commit changes:**
```
> /commit
Committing changes...
[main 7f2a9c3] Add user authentication with JWT tokens
 2 files changed, 89 insertions(+), 12 deletions(-)
```

**Auto-commit toggle:**
```
> /auto-commit on
Auto-commit enabled. Changes will be committed automatically.
```

[06:45 - 07:30] Session Management

Manage your HelixCode session effectively:

**Save session:**
```
> /save my-feature-session
Session saved to .helixcode/sessions/my-feature-session.json
```

**Load session:**
```
> /load my-feature-session
Session loaded: 3 files in context, 12 messages in history
```

**Create snapshot:**
```
> /snapshot before-refactor
Snapshot created: before-refactor
Repository state saved
```

**Restore snapshot:**
```
> /snapshot restore before-refactor
Restoring snapshot: before-refactor
```

**Session history:**
```
> /history
Recent sessions:
  1. my-feature-session (30 minutes ago)
  2. bug-fix-auth (2 hours ago)
  3. refactor-api (yesterday)
```

[07:30 - 08:00] Tips and Tricks

Let me share some CLI tips to make you more productive:

**Tab completion** - Many terminals support tab completion for files and commands. Try typing `/ad` and press tab.

**Command history** - Use up/down arrows to navigate command history. Your conversation history persists across sessions.

**Aliases** - You can create shell aliases for common HelixCode commands:
```bash
alias hc='helixcode'
alias hcr='helixcode --read'
alias hcg4='helixcode --model openai/gpt-4'
```

**Multiple sessions** - You can run multiple HelixCode sessions in different terminals for different projects or branches.

**Background mode** - On Unix systems, you can background HelixCode with Ctrl+Z and `bg`, though this is rarely needed.

In the next chapter, we'll start our first real session and create a simple project. See you there!

---

## Slide Outline

**Slide 1:** "CLI Interface Basics"
- Command-line mastery
- Essential commands
- Interactive session control

**Slide 2:** "Starting HelixCode"
```bash
helixcode                    # Basic start
helixcode --model <model>    # Specify model
helixcode --read             # Read-only mode
helixcode <files>            # Start with files
```

**Slide 3:** "Essential Slash Commands"
- /help - Show commands
- /add - Add files
- /drop - Remove files
- /ls - List context
- /tokens - Usage info
- /quit - Exit

**Slide 4:** "File Management"
- /add <pattern> - Add files
- /drop <pattern> - Remove files
- Glob patterns supported
- Context optimization

**Slide 5:** "Model Commands"
- /models - List available
- /model <name> - Switch model
- /config - Show configuration
- /providers - List providers

**Slide 6:** "Git Integration"
- /git-status - Repository status
- /diff - Show changes
- /undo - Revert changes
- /commit - Commit work
- /auto-commit - Toggle auto-commit

**Slide 7:** "Session Management"
- /save - Save session
- /load - Load session
- /snapshot - Create snapshot
- /history - Recent sessions

**Slide 8:** "Productivity Tips"
- Tab completion
- Command history (↑↓)
- Shell aliases
- Multiple sessions
- Verbose mode for debugging

---

## Chapter Exercises

1. **Command Exploration:** Start HelixCode and run `/help`. Try at least 5 different commands to see what they do.

2. **File Context Practice:**
   - Start HelixCode in a project
   - Add 3 files with `/add`
   - List context with `/ls`
   - Drop one file with `/drop`
   - Verify with `/ls` again

3. **Model Switching:**
   - Run `/models` to see available models
   - Switch to a different model with `/model`
   - Check the change with `/config`

4. **Create Aliases:** Add these aliases to your shell config:
   ```bash
   alias hc='helixcode'
   alias hcv='helixcode --verbose'
   alias hcr='helixcode --read'
   ```

---

## Code Examples

```bash
# Start HelixCode with specific configuration
helixcode \
  --model anthropic/claude-3-opus \
  --auto-commit \
  --verbose \
  src/main.py README.md

# Inside session - build context for a feature
> /add src/auth/*.py
> /add src/models/user.py
> /add tests/test_auth.py
> /ls

# Check token usage before expensive operation
> /tokens
> Analyze the authentication flow and suggest improvements
> /tokens

# Save work for later
> /snapshot experiment-start
> Let's refactor the auth module to use async/await
> /git-status
> /diff
> /commit
```

```bash
# Session workflow
cd my-project
helixcode

# In HelixCode:
> /add src/api/routes.py
> Add input validation to all POST endpoints
> /diff
> Looks good!
> /commit
> /quit

# Later...
cd my-project
helixcode --model openai/gpt-4

> /load previous-session
> Continue where we left off...
```

---

## Quick Reference Card

```
╔═══════════════════════════════════════════╗
║     HelixCode CLI Quick Reference         ║
╠═══════════════════════════════════════════╣
║ STARTING                                  ║
║ helixcode                 Basic start     ║
║ helixcode --model <m>     Specify model   ║
║ helixcode --read          Read-only       ║
║                                           ║
║ FILE MANAGEMENT                           ║
║ /add <file>               Add to context  ║
║ /drop <file>              Remove from ctx ║
║ /ls                       List context    ║
║                                           ║
║ SESSION CONTROL                           ║
║ /help                     Show help       ║
║ /clear                    Clear history   ║
║ /quit                     Exit            ║
║                                           ║
║ GIT OPERATIONS                            ║
║ /diff                     Show changes    ║
║ /undo                     Revert edit     ║
║ /commit                   Commit work     ║
║                                           ║
║ MODEL MANAGEMENT                          ║
║ /models                   List models     ║
║ /model <name>             Switch model    ║
║ /tokens                   Usage info      ║
╚═══════════════════════════════════════════╝
```

---

## Additional Resources

- HelixCode CLI Documentation
- Command Reference (complete list)
- Video: CLI Deep Dive
- Cheat Sheet PDF (downloadable)

---

## Next Chapter

Chapter 2: Starting Your First Session - Create a simple project and make your first edits with HelixCode.
