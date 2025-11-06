# Chapter 1: System Requirements

**Duration:** 8 minutes
**Learning Objectives:**
- Understand HelixCode's system requirements
- Check if your system is compatible
- Plan for optimal performance
- Understand dependencies

---

## Video Script

[00:00 - 00:30] Introduction

Welcome to Course 2: Installation and Setup! In this course, we'll get HelixCode running on your system. Before we dive into installation, we need to ensure your system meets the requirements. In this chapter, we'll cover everything you need to know about system compatibility and dependencies.

[00:30 - 01:30] Operating System Support

HelixCode is cross-platform and supports all major operating systems:

**Linux** - Fully supported on all modern distributions including:
- Ubuntu 20.04 LTS and newer
- Debian 10 and newer
- Fedora 35 and newer
- Arch Linux (rolling release)
- CentOS/RHEL 8 and newer
- Other distributions with Python 3.8+

**macOS** - Fully supported on:
- macOS 11 (Big Sur) and newer
- Both Intel and Apple Silicon (M1/M2/M3) Macs are supported
- Universal binary provides native performance on both architectures

**Windows** - Supported through multiple options:
- WSL2 (Windows Subsystem for Linux) - Recommended approach
- Native Windows with PowerShell
- Git Bash or MinGW environments
- Windows Terminal for best experience

HelixCode runs best on Unix-like systems due to its reliance on shell integration, but Windows support has improved significantly with WSL2.

[01:30 - 02:45] Python Requirements

HelixCode is built with Python, so Python is the primary dependency:

**Python Version** - Python 3.8 or newer is required. Python 3.10 or 3.11 is recommended for best performance and compatibility.

To check your Python version:
```bash
python3 --version
```

If you don't have Python installed or have an older version, you'll need to install or upgrade it. We'll cover this in detail in the installation chapter.

**Virtual Environment Support** - While not strictly required, using a virtual environment is highly recommended. This isolates HelixCode's dependencies from your system Python packages, preventing conflicts.

HelixCode supports all standard Python virtual environment tools:
- venv (built into Python)
- virtualenv
- conda/mamba
- pipenv
- poetry

[02:45 - 04:00] System Resources

Let's talk about hardware requirements:

**CPU** - No specific requirements, but multi-core processors benefit from HelixCode's async operations. Any modern processor from the last 5-7 years will work well.

**RAM** - Minimum 2GB of available RAM, 4GB recommended, 8GB+ for large projects. Memory usage scales with:
- Size of your codebase
- Number of files in the repository map
- Context window size of AI models being used
- Number of concurrent operations

**Disk Space** - HelixCode itself is lightweight (50-100MB installed), but you'll need additional space for:
- Python dependencies (200-500MB)
- Repository maps and cache (varies by project size)
- Log files (typically negligible)

**Network** - Stable internet connection required for cloud-based AI providers. Bandwidth requirements vary:
- Light usage: 1-5 Mbps sufficient
- Heavy usage: 10+ Mbps recommended
- Local model providers (Ollama, etc.) have no network requirements after initial setup

[04:00 - 05:15] Essential Dependencies

Beyond Python, HelixCode requires or benefits from these tools:

**Git** - Essential for version control integration. Git 2.20 or newer recommended. HelixCode uses git for:
- Repository mapping
- Change tracking
- Auto-commit features
- Branch management

**Text Editor** - You'll need a text editor for reviewing changes. HelixCode works with any editor, but these are popular:
- vim/neovim
- VS Code
- Sublime Text
- Emacs
- Nano (for simple edits)

**Terminal** - A modern terminal emulator is important for the best experience:
- Linux: GNOME Terminal, Konsole, Alacritty, Kitty
- macOS: Terminal.app, iTerm2, Alacritty, Kitty
- Windows: Windows Terminal, WSL2 terminal

**Shell** - Bash, Zsh, or Fish recommended. PowerShell works on Windows but some features may be limited.

[05:15 - 06:30] AI Provider Requirements

To use HelixCode, you need access to at least one AI provider:

**API Keys** - Most providers require API keys:
- OpenAI - Requires paid API access
- Anthropic - Requires API access (may have waitlist)
- Google (Gemini) - Free tier available
- Cohere - Free tier available
- Groq - Free tier available

**Cloud Provider Accounts** - For cloud-based models:
- AWS Bedrock - Requires AWS account
- Azure OpenAI - Requires Azure subscription
- Google Vertex AI - Requires Google Cloud account

**Local Model Infrastructure** - For local models:
- Ollama - Runs locally, requires sufficient RAM (8GB+ recommended)
- llama.cpp - Runs locally, supports quantized models
- LocalAI - Self-hosted API compatible with OpenAI format

You don't need all of these - just one provider is enough to get started. We'll cover provider setup in detail in Course 4.

[06:30 - 07:15] Optional Dependencies

These aren't required but enhance functionality:

**ripgrep (rg)** - Faster code searching. HelixCode can use ripgrep if available for improved performance when building repository maps.

**universal-ctags** - Enhanced code parsing for repository maps. Provides better language support and more accurate code structure understanding.

**tree-sitter** - Advanced syntax understanding. Used for precise code analysis and intelligent editing.

**docker** - For containerized workflows and testing in isolated environments.

**ssh** - Required only if using SSH workers for distributed development (covered in Course 7).

[07:15 - 08:00] Compatibility Checklist

Let's create a checklist to verify your system is ready:

- [ ] Operating System: Linux, macOS, or Windows with WSL2
- [ ] Python 3.8+ installed (3.10+ recommended)
- [ ] Git 2.20+ installed
- [ ] Terminal emulator available
- [ ] At least 4GB RAM available
- [ ] Stable internet connection
- [ ] API key for at least one AI provider
- [ ] Text editor installed
- [ ] 500MB free disk space

If you've checked all these boxes, you're ready to proceed with installation!

In the next chapter, we'll walk through the actual installation process step by step for each operating system. See you there!

---

## Slide Outline

**Slide 1:** "System Requirements"
- Cross-platform support
- Modest hardware needs
- Standard dependencies

**Slide 2:** "Operating Systems"
- Linux (all major distros)
- macOS 11+ (Intel & Apple Silicon)
- Windows (WSL2 recommended)

**Slide 3:** "Python Requirements"
- Python 3.8+ required
- Python 3.10-3.11 recommended
- Virtual environment support

**Slide 4:** "Hardware Requirements"
- CPU: Any modern processor
- RAM: 4GB+ recommended
- Disk: 500MB for installation
- Network: Stable connection

**Slide 5:** "Essential Dependencies"
- Git 2.20+
- Text editor
- Modern terminal
- Compatible shell

**Slide 6:** "AI Provider Access"
- API keys (OpenAI, Anthropic, etc.)
- Cloud provider accounts
- Or local model infrastructure

**Slide 7:** "Optional Enhancements"
- ripgrep (faster searching)
- universal-ctags (better parsing)
- tree-sitter (syntax understanding)
- Docker (containerization)

**Slide 8:** "Readiness Checklist"
[Display full checklist from script]

---

## Chapter Exercises

1. **System Check:** Run the commands below to verify your current system:
   ```bash
   python3 --version
   git --version
   echo $SHELL
   free -h  # Linux
   # or
   top -l 1 | grep PhysMem  # macOS
   ```

2. **Provider Research:** Visit the websites of three AI providers. Compare their pricing, free tiers, and API access requirements. Which one will you start with?

3. **Dependency Audit:** Make a list of what you already have installed and what you need to install. Prioritize the essential dependencies.

---

## Code Examples

```bash
# Check Python version
python3 --version
# Output should be 3.8.0 or higher

# Check if pip is installed
pip3 --version

# Check Git version
git --version
# Output should be 2.20.0 or higher

# Check available memory (Linux)
free -h

# Check available memory (macOS)
sysctl hw.memsize

# Verify you can create virtual environments
python3 -m venv test_env
source test_env/bin/activate
deactivate
rm -rf test_env
```

```bash
# Check for optional dependencies
which rg         # ripgrep
which ctags      # universal-ctags
which docker     # Docker
which ssh        # SSH client
```

---

## Troubleshooting

**Python Not Found:**
- Linux: `sudo apt install python3` (Debian/Ubuntu) or `sudo dnf install python3` (Fedora)
- macOS: Install via Homebrew: `brew install python3`
- Windows: Download from python.org or install via Windows Store

**Git Not Found:**
- Linux: `sudo apt install git` or `sudo dnf install git`
- macOS: Install Xcode Command Line Tools: `xcode-select --install`
- Windows: Download from git-scm.com

**Insufficient RAM:**
- Close unnecessary applications
- Consider using lighter AI models
- Use a cloud-based development environment

---

## Additional Resources

- Python Downloads: https://www.python.org/downloads/
- Git Downloads: https://git-scm.com/downloads
- WSL2 Installation Guide: https://docs.microsoft.com/en-us/windows/wsl/install
- Homebrew (macOS): https://brew.sh/
- AI Provider Comparison Chart (in course materials)

---

## Next Chapter

Chapter 2: Installation Process - Step-by-step guide to installing HelixCode on your system.
