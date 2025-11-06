# HelixCode Build Scripts

This directory contains all build, test, and documentation scripts for HelixCode.

## Quick Reference

### Documentation Scripts

| Script | Purpose | Usage |
|--------|---------|-------|
| `sync-manual.sh` | Sync manual to website | `./scripts/sync-manual.sh` or `make sync-manual` |
| `update-docs.sh` | Master documentation updater | `./scripts/update-docs.sh [--pdf] [--changelog]` |

### Build Scripts

| Script | Purpose | Usage |
|--------|---------|-------|
| `build.sh` | Main build script | `./scripts/build.sh [--with-tests] [--no-docs]` |
| `generate-test-keys.sh` | Generate test SSH keys | `./scripts/generate-test-keys.sh` |

### Test Scripts

| Script | Purpose | Usage |
|--------|---------|-------|
| `run-tests.sh` | Run unit tests | `./scripts/run-tests.sh` |
| `run-all-tests.sh` | Run all test suites | `./scripts/run-all-tests.sh` |
| `run-docker-tests.sh` | Run tests in Docker | `./scripts/run-docker-tests.sh` |

### Utility Scripts

| Script | Purpose | Usage |
|--------|---------|-------|
| `pre-commit.template` | Git pre-commit hook template | Copy to `.git/hooks/pre-commit` |

## Documentation Workflow

### Quick Update

Update the manual and sync to website:

```bash
# Method 1: Using Make
make sync-manual

# Method 2: Direct script
./scripts/sync-manual.sh
```

### Full Documentation Update

Update all documentation with PDF and changelog:

```bash
# Basic update
./scripts/update-docs.sh

# With PDF generation
./scripts/update-docs.sh --pdf

# With PDF and changelog
./scripts/update-docs.sh --pdf --changelog
```

### What Each Script Does

#### sync-manual.sh

**Purpose**: Sync user manual from `docs/` to `website/manual/`

**What it does**:
1. Copies `USER_MANUAL.md` to website
2. Updates timestamps
3. Converts Markdown to HTML (if pandoc available)
4. Creates index.html and style.css
5. Generates sync metadata

**Output**:
- `website/manual/USER_MANUAL.md`
- `website/manual/USER_MANUAL.html`
- `website/manual/index.html`
- `website/manual/style.css`
- `website/manual/.sync-metadata.json`

#### update-docs.sh

**Purpose**: Master script for complete documentation updates

**What it does**:
1. Updates timestamps in all Markdown files
2. Converts Markdown to HTML
3. Syncs to website (calls `sync-manual.sh`)
4. Generates PDF (optional, requires `--pdf`)
5. Updates CHANGELOG.md (optional, requires `--changelog`)
6. Generates documentation metadata

**Options**:
- `--pdf`: Generate PDF versions
- `--changelog`: Update CHANGELOG.md
- `--verbose`: Show detailed output
- `--help`: Show help

**Example**:
```bash
./scripts/update-docs.sh --pdf --changelog
```

## Build Workflow

### Standard Build

Build the application with documentation:

```bash
# Method 1: Using Make
make build

# Method 2: Using build script
./scripts/build.sh
```

### Build Without Documentation

Skip documentation generation:

```bash
./scripts/build.sh --no-docs
```

### Build with Tests

Build and run tests:

```bash
./scripts/build.sh --with-tests
```

### Platform-Specific Builds

```bash
# Build for specific platform
./scripts/build.sh --platform linux
./scripts/build.sh --platform darwin
./scripts/build.sh --platform windows

# Build for all platforms (default)
./scripts/build.sh --platform all
```

### Release Build

Complete release build with everything:

```bash
# Using Make (recommended)
make release

# This runs:
# 1. make clean
# 2. make logo-assets
# 3. make docs (includes sync-manual)
# 4. make build
# 5. make test
```

## Integration with Makefile

The scripts are integrated with the Makefile for convenience:

```bash
# Documentation targets
make sync-manual      # Sync manual to website
make manual-html      # Convert Markdown to HTML
make docs            # Build all documentation

# Build targets
make build           # Build application
make assets          # Generate all assets (logo + docs)
make release         # Full release build

# Testing
make test            # Run tests

# Help
make help            # Show all available targets
```

## Pre-Commit Hook

### Installation

To automatically update documentation on commit:

```bash
# Copy the template
cp scripts/pre-commit.template .git/hooks/pre-commit

# Make it executable
chmod +x .git/hooks/pre-commit
```

### What It Does

When you commit changes to documentation files:

1. Detects if `docs/` files changed
2. Updates timestamps automatically
3. Regenerates HTML if `USER_MANUAL.md` changed
4. Stages the updated website files
5. Allows the commit to proceed

### Skipping the Hook

To skip the pre-commit hook for a specific commit:

```bash
git commit --no-verify -m "Your message"
```

## Directory Structure

```
scripts/
├── README.md                 # This file
├── sync-manual.sh           # Sync manual to website
├── update-docs.sh           # Master documentation script
├── build.sh                 # Main build script
├── pre-commit.template      # Git pre-commit hook template
├── generate-test-keys.sh    # Generate test SSH keys
├── run-tests.sh             # Run unit tests
├── run-all-tests.sh         # Run all test suites
├── run-docker-tests.sh      # Run Docker tests
└── logo/
    └── generate_assets.go   # Generate logo assets
```

## Prerequisites

### Required

- **Go 1.24.0+**: For building the application
- **Git**: For version control
- **Bash**: For running scripts (pre-installed on macOS/Linux)

### Optional (for full documentation)

- **pandoc**: For Markdown to HTML/PDF conversion
  ```bash
  # macOS
  brew install pandoc

  # Linux
  sudo apt-get install pandoc
  ```

- **LaTeX (for PDF)**: Required for PDF generation
  ```bash
  # macOS
  brew install basictex

  # Linux
  sudo apt-get install texlive-xetex
  ```

## Common Workflows

### Developer Making Documentation Changes

```bash
# 1. Edit documentation
vim docs/USER_MANUAL.md

# 2. Sync to website
make sync-manual

# 3. Preview locally
open website/manual/index.html

# 4. Commit changes
git add docs/ website/
git commit -m "docs: update user manual"
git push
```

### Preparing a Release

```bash
# 1. Update documentation
./scripts/update-docs.sh --pdf --changelog

# 2. Update version numbers
vim Makefile  # Update VERSION

# 3. Run full release build
make release

# 4. Verify artifacts
ls -lh bin/
ls -lh website/

# 5. Commit and tag
git add .
git commit -m "release: version X.Y.Z"
git tag vX.Y.Z
git push --tags
```

### CI/CD Integration

For continuous integration pipelines:

```yaml
# Example for GitHub Actions
steps:
  - name: Build Application
    run: make build

  - name: Build Documentation
    run: make docs

  - name: Run Tests
    run: make test

  - name: Create Release
    run: make release
```

## Troubleshooting

### Problem: "pandoc not found"

**Solution**: Install pandoc
```bash
brew install pandoc  # macOS
sudo apt-get install pandoc  # Linux
```

### Problem: HTML not updating

**Solution**: Clear and regenerate
```bash
rm -rf website/manual/*
make sync-manual
```

### Problem: Timestamps not updating

**Solution**: Run update script explicitly
```bash
./scripts/update-docs.sh
```

### Problem: Build script fails

**Solution**: Check prerequisites
```bash
# Verify Go installation
go version

# Verify Git
git --version

# Check script permissions
ls -l scripts/*.sh
```

### Problem: Pre-commit hook not running

**Solution**: Ensure it's executable
```bash
chmod +x .git/hooks/pre-commit

# Test manually
.git/hooks/pre-commit
```

## Environment Variables

Scripts support these environment variables:

- `VERBOSE=1`: Enable verbose output
- `SKIP_TESTS=1`: Skip tests in build script
- `SKIP_DOCS=1`: Skip documentation in build script

Example:
```bash
VERBOSE=1 ./scripts/build.sh
SKIP_DOCS=1 make build
```

## Contributing

See `docs/CONTRIBUTING.md` for detailed guidelines on:
- Documentation structure
- Writing guidelines
- Markdown style
- Review process

## Version History

- **v1.0.0** (2025-11-06): Initial comprehensive build system
  - Documentation sync and update scripts
  - Integrated build workflow
  - Pre-commit hook template
  - Makefile integration

## Support

For issues or questions:
- Open an issue on GitHub
- Check `docs/CONTRIBUTING.md`
- Review build logs in `/tmp/helixcode-*.log`

---

**Last Updated**: 2025-11-06
