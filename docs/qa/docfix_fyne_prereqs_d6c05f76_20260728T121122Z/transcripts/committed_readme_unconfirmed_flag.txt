### case: committed_readme_unconfirmed_flag
### label: the exact honesty-boundary text this commit committed into README.md
### time: 2026-07-28T12:11:29Z
### command: git show d6c05f76fcc0000dfa366cf566420d244c60a23f:helix_code/applications/desktop/README.md
### cwd: /home/milos/Factory/projects/tools_and_research/helix_code

--- COMBINED STDOUT+STDERR (real command execution, captured verbatim) ---
# HelixCode Desktop Application

A cross-platform desktop application for HelixCode, built with Go and Fyne. Provides a graphical interface for managing AI-powered development workflows.

## Overview

The Desktop application offers:
- Native look and feel on Windows, macOS, and Linux
- Project management with visual file browser
- Task dashboard with drag-and-drop
- Workflow execution with progress visualization
- Worker management interface
- LLM provider configuration GUI
- Session management for development workflows
- System tray integration

## Build Modes

The desktop application supports two build modes:

1. **GUI Mode** (default): Full graphical interface using Fyne toolkit
2. **NoGUI Mode**: Command-line interface alternative for headless environments

### Building GUI Mode (default)
```bash
go build -o bin/helix-desktop ./applications/desktop
```

### Building NoGUI Mode (CLI)
```bash
go build -tags nogui -o bin/helix-desktop-cli ./applications/desktop
```

## Requirements

### Build Requirements

- Go 1.24+
- C compiler (gcc on Linux/Windows, clang on macOS)
- OpenGL development libraries (for GUI mode only)

**This is a build-host environment prerequisite, not a code defect (§11.4.77).**
`applications/desktop`, `applications/aurora_os`, and `applications/harmony_os`
all pull in Fyne's default desktop OpenGL driver, which transitively depends on
`github.com/go-gl/gl` (needs `gl.pc` via `pkg-config`) and
`github.com/go-gl/glfw` (needs `X11/Xlib.h` via CGO). On a build host missing
the X11/OpenGL dev headers below, `go build ./applications/desktop/...` (and
the `aurora_os`/`harmony_os` equivalents, which share the same Fyne GUI
dependency) fails with:

```
# github.com/go-gl/gl/v2.1/gl
# [pkg-config --cflags  -- gl gl]
Package gl was not found in the pkg-config search path.
Perhaps you should add the directory containing `gl.pc'
to the PKG_CONFIG_PATH environment variable
No package 'gl' found
...
# github.com/go-gl/glfw/v3.3/glfw
In file included from ./glfw/src/internal.h:188,
                 from ./glfw/src/context.c:30,
                 from .../go-gl/glfw/v3.3/glfw@.../c_glfw.go:4:
./glfw/src/x11_platform.h:33:10: fatal error: X11/Xlib.h: No such file or directory
   33 | #include <X11/Xlib.h>
      |          ^~~~~~~~~~~~
compilation terminated.
```

Installing the per-distro dev packages below (§11.4.99 — verified against the
current official Fyne prerequisites doc,
[docs.fyne.io/started/quick](https://docs.fyne.io/started/quick/), fetched
2026-07-08) resolves it. To sidestep the GUI toolchain entirely (e.g. on a
headless build host), use `-tags nogui` — see "Headless/Server Environments"
below.

### Linux Requirements

#### Debian/Ubuntu
```bash
# Essential build tools
sudo apt-get update
sudo apt-get install -y build-essential

# GUI dependencies (required for GUI mode)
sudo apt-get install -y \
    libx11-dev \
    libxcursor-dev \
    libxrandr-dev \
    libxinerama-dev \
    libxi-dev \
    libxxf86vm-dev \
    libgl1-mesa-dev \
    xorg-dev

# Optional: For Wayland support
sudo apt-get install -y libwayland-dev
```

#### Fedora/RHEL/CentOS
```bash
# Essential build tools
sudo dnf groupinstall -y "Development Tools"

# GUI dependencies (required for GUI mode)
sudo dnf install -y \
    gcc \
    mesa-libGL-devel \
    libX11-devel \
    libXcursor-devel \
    libXrandr-devel \
    libXinerama-devel \
    libXi-devel \
    libXxf86vm-devel

# Optional: For Wayland support
sudo dnf install -y wayland-devel
```

#### Arch Linux
```bash
# GUI dependencies (required for GUI mode)
sudo pacman -S \
    base-devel \
    libx11 \
    libxcursor \
    libxrandr \
    libxinerama \
    libxi \
    libxxf86vm \
    mesa
```

#### Alpine Linux
```bash
# GUI dependencies (required for GUI mode)
apk add \
    build-base \
    libx11-dev \
    libxcursor-dev \
    libxrandr-dev \
    libxinerama-dev \
    libxi-dev \
    mesa-dev
```

#### ALT Linux

**Verified on this host** (`ALT Workstation 11.1 "Prometheus"`, kernel
`6.12.41-6.12-alt1`, `x86_64-alt-linux-gcc (GCC) 15.2.1`, `pkg-config 0.29.2`)
via `apt-cache search`/`apt-cache show` against the live Sisyphus repo index —
these package names exist on this host's configured repositories as of
2026-07-08. `gcc` and `pkg-config` are already present on this host; only the
X11/OpenGL `-devel` packages are missing.

```bash
# Essential build tools (gcc/pkg-config are usually already present)
su -c 'apt-get install -y gcc gcc-c++ pkgconf'

# GUI dependencies (required for GUI mode) — mirrors the RPM/Fedora naming
# convention that ALT Linux (an independent RPM-based distro) follows
su -c 'apt-get install -y \
    libX11-devel \
    libXcursor-devel \
    libXrandr-devel \
    libXinerama-devel \
    libXi-devel \
    libXxf86vm-devel \
    libGL-devel \
    libxkbcommon-devel'

# Optional: for Wayland support
su -c 'apt-get install -y wayland-devel'
```

**Honest boundary (UNCONFIRMED, §11.4.6):** package *existence* was verified
live against this host's repo metadata (`apt-cache show`), but the exact
`.pc` filename each package installs (e.g. whether `libGL-devel` ships
`gl.pc` under that literal name, as Fedora's `mesa-libGL-devel` does) was
**not** independently confirmed by actually installing the packages and
re-running the build in this session — that step still needs to be run by
whoever installs these packages, to close the loop per §11.4.108
(source→artifact→runtime→user-visible). If `pkg-config --cflags gl` still
fails to find `gl.pc` after installing `libGL-devel`, check
`rpm -ql libGL-devel | grep '\.pc$'` for the actual provided `.pc` name/path.

### macOS Requirements

```bash
# Install Xcode Command Line Tools (includes clang compiler)
xcode-select --install

# No additional dependencies required - macOS includes all necessary frameworks
```

**Note**: macOS uses native Cocoa frameworks. No additional GUI libraries needed.

### Windows Requirements

#### Option 1: TDM-GCC (Recommended)
1. Download and install [TDM-GCC](https://jmeubank.github.io/tdm-gcc/)
2. Ensure `gcc` is in your PATH
3. No additional dependencies required

#### Option 2: MSYS2
```bash
# Install MSYS2 from https://www.msys2.org/
# Then in MSYS2 terminal:
pacman -S mingw-w64-x86_64-gcc mingw-w64-x86_64-pkg-config
```

#### Option 3: WSL2
Use WSL2 with Ubuntu and follow Linux instructions above.

**Note**: Windows uses native Win32 APIs. No additional GUI libraries needed after compiler setup.

### Headless/Server Environments

For headless servers or containers where GUI is not needed:

```bash
# Build without GUI dependencies
go build -tags nogui -o bin/helix-desktop-cli ./applications/desktop
```

This builds a CLI-only version that does not require any X11/OpenGL libraries.

## Building

```bash
cd HelixCode

# Build for current platform
go build -o bin/helix-desktop ./applications/desktop

# Cross-compile for all platforms
make desktop-all
```

### Platform-Specific Builds

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o bin/helix-desktop-linux ./applications/desktop

# macOS
GOOS=darwin GOARCH=amd64 go build -o bin/helix-desktop-macos ./applications/desktop

# Windows
GOOS=windows GOARCH=amd64 go build -o bin/helix-desktop.exe ./applications/desktop
```

## Running

```bash
./bin/helix-desktop
```

### Command Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `--server` | HelixCode server URL | http://localhost:8080 |
| `--config` | Configuration file path | ~/.config/helixcode/desktop.yaml |
| `--theme` | UI theme (light/dark) | system |
| `--scale` | UI scaling factor | 1.0 |

## Features

### Dashboard

The main dashboard provides:
- Quick stats overview (projects, tasks, workers)
- Recent activity feed
- Quick action buttons
- System status indicators

### Project Manager

- Visual project browser
- File tree navigation
- Project creation wizard
- Metadata editor
- Git integration status

### Task Center

- Kanban-style task board
- Task creation dialog
- Priority and status filtering
- Drag-and-drop task management
- Task detail view

### Workflow Runner

- One-click workflow execution
- Progress visualization
- Step-by-step status
- Error reporting
- Workflow history

### Worker Manager

- Worker list with health status
- Resource utilization graphs
- Worker configuration
- Task assignment view

### Settings

- Server connection configuration
- Theme customization
- Keyboard shortcuts
- Notification preferences
- LLM provider configuration

## Configuration

Create `~/.config/helixcode/desktop.yaml`:

```yaml
server:
  url: "http://localhost:8080"
  timeout: 30s
  auto_reconnect: true

appearance:
  theme: "dark"
  scale: 1.0
  font_size: 12
  animations: true

behavior:
  start_minimized: false
  minimize_to_tray: true
  auto_update: true
  confirm_exit: true

notifications:
  enabled: true
  sound: true
  task_complete: true
  task_failed: true
  worker_offline: true

keyboard:
  new_project: "Ctrl+N"
  new_task: "Ctrl+T"
  settings: "Ctrl+,"
  quit: "Ctrl+Q"
```

## Keyboard Shortcuts

### Global

| Shortcut | Action |
|----------|--------|
| `Ctrl+N` | New project |
| `Ctrl+T` | New task |
| `Ctrl+R` | Refresh |
| `Ctrl+,` | Open settings |
| `Ctrl+Q` | Quit |
| `F1` | Help |
| `F11` | Toggle fullscreen |

### Navigation

| Shortcut | Action |
|----------|--------|
| `Ctrl+1` | Dashboard |
| `Ctrl+2` | Projects |
| `Ctrl+3` | Tasks |
| `Ctrl+4` | Workers |
| `Ctrl+5` | Settings |

### Actions

| Shortcut | Action |
|----------|--------|
| `Enter` | Open/Select |
| `Delete` | Delete item |
| `Ctrl+S` | Save |
| `Escape` | Cancel/Close |

## System Tray

When minimize to tray is enabled:
- Left-click: Show/hide window
- Right-click: Context menu
  - Show window
  - Quick actions
  - Settings
  - Quit

## Themes

### Built-in Themes

- **Light**: Light background with dark text
- **Dark**: Dark background with light text
- **System**: Follow system preference

### Custom Themes

Create custom themes in `~/.config/helixcode/themes/`:

```yaml
name: "My Theme"
colors:
  background: "#1E1E1E"
  foreground: "#FFFFFF"
  primary: "#00ADD8"
  secondary: "#5DC9E2"
  success: "#4CAF50"
  warning: "#FF9800"
  error: "#F44336"
```

## Troubleshooting

### Application won't start

1. Check OpenGL support:
   ```bash
   glxinfo | grep "OpenGL version"
   ```

2. Try software rendering:
   ```bash
   FYNE_RENDER_SOFTWARE=1 ./bin/helix-desktop
   ```

### High DPI Issues

Adjust scaling:
```bash
./bin/helix-desktop --scale 1.5
```

Or set environment variable:
```bash
export FYNE_SCALE=1.5
./bin/helix-desktop
```

### Connection Issues

- Verify server URL in settings
- Check firewall settings
- Ensure server is running

### Performance

For better performance:
- Disable animations in settings
- Reduce refresh rate
- Use dark theme (less GPU usage)

## Dependencies

- [Fyne](https://fyne.io/) - Cross-platform GUI toolkit
- Go 1.24+
- OpenGL 2.0+

## Related Documentation

- [CLI Documentation](../../cmd/cli/README.md)
- [Terminal UI](../terminal-ui/README.md)
- [Server Documentation](../../cmd/server/README.md)

--- exit code: 0 ---
