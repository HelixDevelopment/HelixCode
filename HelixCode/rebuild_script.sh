#!/bin/bash

# Comprehensive rebuild script for HelixCode
set -e

echo "🧹 Cleaning up temporary files and logs..."

# Clean build artifacts
rm -rf bin/
rm -rf dist/
rm -rf coverage.out
rm -rf coverage-*.out
rm -rf cov_*.out
rm -rf auth_coverage.out
rm -rf coverage_auth.out
rm -rf coverage_full.out
rm -rf coverage_types.out
rm -rf coverage-web.out
rm -rf coverage-final.out
rm -rf cover.out

# Clean compiled binaries
rm -f cli desktop aurora-os harmony-os main multi-agent-system performance-optimization-standalone

# Clean test artifacts
rm -rf test-results/
rm -rf reports/
rm -rf logs/

# Clean Docker containers and images
echo "🐳 Stopping and removing Docker containers..."
docker-compose down --remove-orphans || true
docker-compose -f docker-compose.test.yml down --remove-orphans || true
docker-compose -f docker-compose.aurora-os.yml down --remove-orphans || true
docker-compose -f docker-compose.harmony-os.yml down --remove-orphans || true

# Clean Docker images
echo "🐳 Removing Docker images..."
docker system prune -f || true

echo "✅ Cleanup complete"

# Setup dependencies
echo "📦 Setting up dependencies..."
go mod download
go mod tidy

echo "✅ Dependencies setup complete"

# Generate logo assets
echo "🎨 Generating logo assets..."
cd scripts/logo && go run generate_assets.go && cd ../..

echo "✅ Logo assets generated"

# Build main server
echo "🚀 Building main server..."
go build -ldflags="-X main.version=1.0.0 -X main.buildTime=$(date +%Y-%m-%d_%H:%M:%S) -X main.gitCommit=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')" -o bin/helixcode ./cmd/server

echo "✅ Main server built: bin/helixcode"

# Build CLI
echo "💻 Building CLI..."
go build -o bin/cli ./cmd/cli

echo "✅ CLI built: bin/cli"

# Build Terminal UI
echo "🖥️ Building Terminal UI..."
go build -o bin/tui ./applications/terminal-ui

echo "✅ Terminal UI built: bin/tui"

# Build Desktop app
echo "🖥️ Building Desktop app..."
go build -o bin/desktop ./applications/desktop

echo "✅ Desktop app built: bin/desktop"

# Build Aurora OS client
echo "🌟 Building Aurora OS client..."
go build -o bin/aurora-os ./applications/aurora-os

echo "✅ Aurora OS client built: bin/aurora-os"

# Build Harmony OS client
echo "🔶 Building Harmony OS client..."
go build -o bin/harmony-os ./applications/harmony-os

echo "✅ Harmony OS client built: bin/harmony-os"

# Build mobile bindings if gomobile is available
echo "📱 Checking for mobile build tools..."
if command -v gomobile &> /dev/null; then
    echo "📱 Building iOS framework..."
    gomobile bind -target=ios -o HelixCore.xcframework ./shared/mobile-core
    echo "✅ iOS framework built: HelixCore.xcframework"
    
    echo "🤖 Building Android AAR..."
    gomobile bind -target=android -o mobile.aar ./shared/mobile-core
    echo "✅ Android AAR built: mobile.aar"
else
    echo "⚠️  gomobile not found - skipping mobile builds"
    echo "   Install with: go install golang.org/x/mobile/cmd/gomobile@latest"
fi

# Recreate Docker containers
echo "🐳 Recreating Docker containers..."
docker-compose up --build -d

echo "✅ Docker containers recreated"

# Run all tests
echo "🧪 Running all tests..."
./scripts/run_all_tests.sh

echo "✅ All tests completed successfully"

echo "🎉 Rebuild process completed successfully!"
echo ""
echo "📋 Built applications:"
echo "  • Main server: bin/helixcode"
echo "  • CLI: bin/cli"
echo "  • Terminal UI: bin/tui"
echo "  • Desktop: bin/desktop"
echo "  • Aurora OS: bin/aurora-os"
echo "  • Harmony OS: bin/harmony-os"
if [ -f "HelixCore.xcframework" ]; then
    echo "  • iOS: HelixCore.xcframework"
fi
if [ -f "mobile.aar" ]; then
    echo "  • Android: mobile.aar"
fi