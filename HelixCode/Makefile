.PHONY: all build test clean logo-assets setup-deps sync-manual manual-html docs release

# Build variables
BINARY_NAME=helixcode
VERSION=1.0.0
BUILD_TIME=$(shell date +%Y-%m-%d_%H:%M:%S)
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go variables
GO=go
GO_BUILD=$(GO) build
GO_TEST=$(GO) test
GO_MOD=$(GO) mod
GOFMT=$(GO) fmt
GOLINT=golangci-lint run

# Directories
CMD_DIR=cmd
INTERNAL_DIR=internal
PKG_DIR=pkg
ASSETS_DIR=assets
SCRIPTS_DIR=scripts

all: build

# Build the application
build: logo-assets
	@echo "🚀 Building $(BINARY_NAME)..."
	$(GO_BUILD) -ldflags="-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)" -o bin/$(BINARY_NAME) ./$(CMD_DIR)/server
	@echo "✅ Build complete: bin/$(BINARY_NAME)"

# Run tests
test:
	@echo "🧪 Running tests..."
	$(GO_TEST) -v ./...
	@echo "✅ Tests complete"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf bin/
	rm -rf dist/
	rm -rf coverage.out
	@echo "✅ Clean complete"

# Generate logo assets
logo-assets:
	@echo "🎨 Generating logo assets..."
	cd $(SCRIPTS_DIR)/logo && $(GO) run generate_assets.go
	@echo "✅ Logo assets generated"

# Setup dependencies
setup-deps:
	@echo "📦 Setting up dependencies..."
	$(GO_MOD) download
	$(GO_MOD) tidy
	@echo "✅ Dependencies setup complete"

# Format code
fmt:
	@echo "🎨 Formatting code..."
	$(GOFMT) ./...
	@echo "✅ Format complete"

# Lint code
lint:
	@echo "🔍 Linting code..."
	$(GOLINT) ./...
	@echo "✅ Lint complete"

# Run the application
dev: build
	@echo "🚀 Starting development server..."
	./bin/$(BINARY_NAME) --config config/dev/config.yaml

# Build for production
prod: clean build
	@echo "🏗️ Building for production..."
	GOOS=linux GOARCH=amd64 $(GO_BUILD) -ldflags="-s -w" -o bin/$(BINARY_NAME)-linux ./$(CMD_DIR)/server
	GOOS=darwin GOARCH=amd64 $(GO_BUILD) -ldflags="-s -w" -o bin/$(BINARY_NAME)-macos ./$(CMD_DIR)/server
	GOOS=windows GOARCH=amd64 $(GO_BUILD) -ldflags="-s -w" -o bin/$(BINARY_NAME)-windows.exe ./$(CMD_DIR)/server
	@echo "✅ Production builds complete"

# Install dependencies for logo processing
logo-deps:
	@echo "📦 Installing logo processing dependencies..."
	$(GO) get github.com/nfnt/resize
	@echo "✅ Logo dependencies installed"

# Generate all assets (logo + documentation)
assets: logo-assets sync-manual
	@echo "✅ All assets generated"

# Sync user manual to website
sync-manual:
	@echo "📚 Syncing user manual to website..."
	$(SCRIPTS_DIR)/sync-manual.sh
	@echo "✅ Manual synced"

# Convert manual Markdown to HTML
manual-html:
	@echo "📄 Converting manual to HTML..."
	@if command -v pandoc > /dev/null; then \
		mkdir -p website/manual; \
		pandoc docs/USER_MANUAL.md -f markdown -t html5 --standalone --toc --toc-depth=3 \
			--css=style.css --metadata title="HelixCode User Manual" \
			-o website/manual/USER_MANUAL.html; \
		echo "✅ HTML manual generated"; \
	else \
		echo "⚠️  pandoc not installed - skipping HTML conversion"; \
		echo "   Install with: brew install pandoc (macOS) or apt-get install pandoc (Linux)"; \
	fi

# Build documentation
docs: manual-html sync-manual
	@echo "📚 Building documentation..."
	@echo "✅ Documentation built"

# Release build with all assets and documentation
release: clean logo-assets docs build test
	@echo "🚀 Building release..."
	@echo "✅ Release build complete"

help:
	@echo "HelixCode Build System"
	@echo ""
	@echo "Usage:"
	@echo "  make build           - Build the application"
	@echo "  make test            - Run tests"
	@echo "  make clean           - Clean build artifacts"
	@echo "  make logo-assets     - Generate logo assets and themes"
	@echo "  make setup-deps      - Setup dependencies"
	@echo "  make fmt             - Format code"
	@echo "  make lint            - Lint code"
	@echo "  make dev             - Run development server"
	@echo "  make prod            - Build for production"
	@echo "  make assets          - Generate all assets"
	@echo ""
	@echo "Documentation:"
	@echo "  make sync-manual     - Sync user manual to website"
	@echo "  make manual-html     - Convert manual Markdown to HTML"
	@echo "  make docs            - Build all documentation"
	@echo "  make release         - Full release build (code + docs + tests)"
	@echo ""
	@echo "Mobile:"
	@echo "  make mobile-init     - Initialize gomobile"
	@echo "  make mobile-ios      - Build iOS framework"
	@echo "  make mobile-android  - Build Android AAR"
	@echo "  make mobile          - Build all mobile bindings"
	@echo ""
	@echo "Platform-Specific:"
	@echo "  make aurora-os       - Build Aurora OS client"
	@echo "  make harmony-os      - Build Harmony OS client"
	@echo "  make aurora-harmony  - Build Aurora and Harmony OS clients"
	@echo ""
	@echo "  make help            - Show this help"

# Mobile build targets
mobile-init:
	@echo "📱 Initializing gomobile..."
	$(GO) run golang.org/x/mobile/cmd/gomobile init

mobile-ios: mobile-init
	@echo "🍎 Building iOS framework..."
	gomobile bind -target=ios -o HelixCore.xcframework ./shared/mobile-core
	@echo "✅ iOS framework built: HelixCore.xcframework"

mobile-android: mobile-init
	@echo "🤖 Building Android AAR..."
	gomobile bind -target=android -o mobile.aar ./shared/mobile-core
	@echo "✅ Android AAR built: mobile.aar"

mobile: mobile-ios mobile-android
	@echo "📱 All mobile bindings built"

# Aurora OS and Harmony OS build targets
aurora-os:
	@echo "🌟 Building Aurora OS client..."
	$(GO_BUILD) -ldflags="-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)" -o bin/aurora-os ./applications/aurora-os
	@echo "✅ Aurora OS client built: bin/aurora-os"

harmony-os:
	@echo "🔶 Building Harmony OS client..."
	$(GO_BUILD) -ldflags="-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)" -o bin/harmony-os ./applications/harmony-os
	@echo "✅ Harmony OS client built: bin/harmony-os"

aurora-harmony: aurora-os harmony-os
	@echo "🌟🔶 Aurora OS and Harmony OS clients built"

# Development tools setup
dev-setup: setup-deps logo-deps
	@echo "🔧 Development environment setup complete"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Run 'make logo-assets' to generate logo assets"
	@echo "  2. Run 'make build' to build the application"
	@echo "  3. Run 'make dev' to start development server"
	@echo "  4. Run 'make mobile-ios' to build iOS framework"
	@echo "  5. Run 'make mobile-android' to build Android AAR"
	@echo "  6. Run 'make aurora-os' to build Aurora OS client"
	@echo "  7. Run 'make harmony-os' to build Harmony OS client"