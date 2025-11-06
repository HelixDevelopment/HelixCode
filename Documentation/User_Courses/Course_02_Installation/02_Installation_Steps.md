# Chapter 2: Installation Steps

**Duration:** 12 minutes
**Learning Objectives:**
- Install HelixCode on your operating system
- Set up a virtual environment
- Verify the installation
- Troubleshoot common installation issues

---

## Video Script

[00:00 - 00:30] Introduction

Welcome back! Now that we've verified your system meets the requirements, let's install HelixCode. I'll walk through the process for Linux, macOS, and Windows. The basic steps are similar across platforms, but there are some OS-specific considerations we'll cover.

[00:30 - 02:00] Installation Methods Overview

HelixCode can be installed several ways:

**pip (recommended for most users)** - Install directly from PyPI using pip. This is the simplest method and works on all platforms.

**pipx (recommended for CLI tools)** - Installs HelixCode in an isolated environment with global command access. Great if you want to avoid dependency conflicts.

**From source (for developers)** - Clone the repository and install in development mode. This is useful if you want to contribute to HelixCode or stay on the bleeding edge.

**Package managers** - Some platforms have HelixCode in their package repositories:
- Homebrew on macOS
- AUR on Arch Linux
- Snap packages (coming soon)

We'll focus on pip installation as it's the most universal approach, then cover the alternatives.

[02:00 - 04:30] Linux Installation

Let's start with Linux, as it's the most straightforward.

**Step 1: Ensure Python and pip are installed**
```bash
python3 --version
pip3 --version
```

If pip isn't installed:
```bash
# Debian/Ubuntu
sudo apt update
sudo apt install python3-pip python3-venv

# Fedora
sudo dnf install python3-pip

# Arch
sudo pacman -S python-pip
```

**Step 2: Create a virtual environment (recommended)**
```bash
mkdir -p ~/.local/helixcode
cd ~/.local/helixcode
python3 -m venv venv
source venv/bin/activate
```

**Step 3: Install HelixCode**
```bash
pip install helixcode
```

This downloads HelixCode and all its dependencies. It may take a minute or two.

**Step 4: Verify installation**
```bash
helixcode --version
```

You should see the version number displayed. Congratulations, HelixCode is installed!

**Step 5: Make it globally available**

To use HelixCode from any directory without activating the virtual environment, add it to your PATH. Add this to your `~/.bashrc` or `~/.zshrc`:

```bash
export PATH="$HOME/.local/helixcode/venv/bin:$PATH"
```

Then reload your shell configuration:
```bash
source ~/.bashrc  # or ~/.zshrc
```

[04:30 - 06:45] macOS Installation

macOS installation is very similar to Linux, with a few Mac-specific considerations.

**Step 1: Install Homebrew (if not already installed)**

Homebrew is the easiest way to manage dependencies on macOS:
```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

**Step 2: Install Python**

macOS comes with Python, but it's often outdated. Install a recent version via Homebrew:
```bash
brew install python@3.11
```

**Step 3: Verify Python installation**
```bash
python3 --version
pip3 --version
```

**Step 4: Create virtual environment**
```bash
mkdir -p ~/Library/Application\ Support/helixcode
cd ~/Library/Application\ Support/helixcode
python3 -m venv venv
source venv/bin/activate
```

**Step 5: Install HelixCode**
```bash
pip install helixcode
```

**Step 6: Verify installation**
```bash
helixcode --version
```

**Step 7: PATH configuration**

Add to your `~/.zshrc` (zsh is default on modern macOS):
```bash
export PATH="$HOME/Library/Application Support/helixcode/venv/bin:$PATH"
```

Reload: `source ~/.zshrc`

**Alternative: Homebrew installation (if available)**
```bash
brew install helixcode
```

This handles everything automatically - no virtual environment needed.

[06:45 - 09:00] Windows Installation (WSL2)

Windows installation is best done through WSL2 for full compatibility.

**Step 1: Install WSL2**

Open PowerShell as Administrator and run:
```powershell
wsl --install
```

This installs WSL2 with Ubuntu by default. Restart your computer when prompted.

**Step 2: Launch WSL2**

Open Windows Terminal or search for "Ubuntu" in the Start menu. You'll be prompted to create a username and password.

**Step 3: Update system packages**
```bash
sudo apt update
sudo apt upgrade
```

**Step 4: Install Python and pip**
```bash
sudo apt install python3 python3-pip python3-venv git
```

**Step 5: Create virtual environment**
```bash
mkdir -p ~/.local/helixcode
cd ~/.local/helixcode
python3 -m venv venv
source venv/bin/activate
```

**Step 6: Install HelixCode**
```bash
pip install helixcode
```

**Step 7: Verify installation**
```bash
helixcode --version
```

**Step 8: PATH configuration**

Add to `~/.bashrc`:
```bash
export PATH="$HOME/.local/helixcode/venv/bin:$PATH"
```

Reload: `source ~/.bashrc`

**Native Windows Installation (Alternative)**

If you prefer native Windows without WSL2:

1. Install Python from python.org (ensure "Add to PATH" is checked)
2. Open PowerShell or CMD
3. Create virtual environment:
   ```powershell
   python -m venv C:\Users\YourUsername\helixcode-env
   C:\Users\YourUsername\helixcode-env\Scripts\Activate.ps1
   ```
4. Install: `pip install helixcode`
5. Run: `helixcode --version`

Note: Some features may be limited on native Windows.

[09:00 - 10:30] Alternative: pipx Installation

pipx is excellent for CLI tools like HelixCode. It installs in an isolated environment but makes the command globally available.

**Install pipx:**
```bash
# Linux/WSL
python3 -m pip install --user pipx
python3 -m pipx ensurepath

# macOS
brew install pipx
pipx ensurepath
```

**Install HelixCode via pipx:**
```bash
pipx install helixcode
```

That's it! pipx handles the virtual environment automatically, and `helixcode` command is immediately available everywhere.

**Benefits:**
- No manual PATH configuration needed
- Isolated from other Python packages
- Easy to upgrade: `pipx upgrade helixcode`
- Easy to uninstall: `pipx uninstall helixcode`

[10:30 - 11:30] Development Installation

If you want to contribute to HelixCode or use the latest development version:

**Step 1: Clone the repository**
```bash
git clone https://github.com/helix-editor/helixcode.git
cd helixcode
```

**Step 2: Create virtual environment**
```bash
python3 -m venv venv
source venv/bin/activate  # or venv\Scripts\activate on Windows
```

**Step 3: Install in development mode**
```bash
pip install -e .
```

The `-e` flag installs in "editable" mode, meaning changes to the source code take effect immediately without reinstalling.

**Step 4: Install development dependencies**
```bash
pip install -e ".[dev]"
```

This includes testing tools, linters, and other dev utilities.

[11:30 - 12:00] Verification and Next Steps

After installation, verify everything works:

```bash
# Check version
helixcode --version

# Check configuration
helixcode --help

# Test basic functionality (without AI providers yet)
helixcode --list-models
```

If you see any errors, refer to the troubleshooting section in the exercises below.

In the next chapter, we'll configure HelixCode with your first AI provider and prepare for your first session. See you there!

---

## Slide Outline

**Slide 1:** "Installation Methods"
- pip (recommended)
- pipx (CLI tools)
- From source (developers)
- Package managers

**Slide 2:** "Linux Installation"
1. Install Python/pip
2. Create virtual environment
3. Install HelixCode
4. Configure PATH

**Slide 3:** "macOS Installation"
1. Install Homebrew
2. Install Python
3. Create virtual environment
4. Install HelixCode
5. Configure PATH

**Slide 4:** "Windows (WSL2) Installation"
1. Install WSL2
2. Update packages
3. Install Python
4. Install HelixCode
5. Configure PATH

**Slide 5:** "pipx Installation"
- One command installation
- Automatic isolation
- No PATH configuration needed
- Easy updates

**Slide 6:** "Development Installation"
- Clone repository
- Install in editable mode
- Include dev dependencies

**Slide 7:** "Verification Steps"
```
helixcode --version
helixcode --help
helixcode --list-models
```

---

## Chapter Exercises

1. **Complete Installation:** Follow the steps for your operating system and install HelixCode. Verify with `helixcode --version`.

2. **Virtual Environment Practice:** Create a test virtual environment, activate it, install a package (try `requests`), then deactivate and delete the environment.

3. **PATH Verification:** Open a new terminal window and verify that `helixcode` command is available without activating any virtual environment.

---

## Code Examples

```bash
# Quick Linux/macOS installation script
curl -sSL https://raw.githubusercontent.com/helix-editor/helixcode/main/install.sh | bash

# Or manual installation
python3 -m venv ~/.helixcode-env
source ~/.helixcode-env/bin/activate
pip install helixcode
echo 'export PATH="$HOME/.helixcode-env/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

```bash
# Upgrade HelixCode to latest version
pip install --upgrade helixcode

# Or with pipx
pipx upgrade helixcode
```

```bash
# Uninstall HelixCode
pip uninstall helixcode

# Or with pipx
pipx uninstall helixcode
```

---

## Troubleshooting

**"helixcode: command not found"**
- Virtual environment not activated
- PATH not configured correctly
- Installation failed silently

Solution: Check installation with `pip list | grep helixcode`

**Permission denied errors**
- Don't use sudo with pip in virtual environment
- On Linux, ensure ~/.local/bin is in PATH

Solution: Use virtual environment or pipx

**SSL certificate errors**
- Corporate proxy or outdated system certificates

Solution: `pip install --trusted-host pypi.org --trusted-host files.pythonhosted.org helixcode`

**Module import errors**
- Conflicting Python packages
- Corrupted virtual environment

Solution: Delete virtual environment and recreate

---

## Additional Resources

- Official Installation Guide
- Platform-specific Troubleshooting
- Video: Installation Walkthrough (all platforms)
- Community Installation Scripts
- Docker Installation (alternative)

---

## Next Chapter

Chapter 3: Initial Configuration - Setting up your first AI provider and configuring HelixCode preferences.
