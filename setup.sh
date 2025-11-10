#!/bin/bash

# HelixCode Setup Script
# This script sets up the entire HelixCode development environment

set -e

echo "=================================="
echo "HelixCode Development Setup"
echo "=================================="
echo ""

# Check if git is available
if ! command -v git &> /dev/null; then
    echo "❌ Git is not installed. Please install git first."
    exit 1
fi

echo "📦 Initializing git submodules..."
./scripts/init-submodules.sh

echo ""
echo "🔧 Installing system dependencies..."
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    ./install_missing_libs.sh
elif [[ "$OSTYPE" == "darwin"* ]]; then
    echo "macOS detected. Please ensure you have Xcode Command Line Tools installed."
    echo "Run: xcode-select --install"
else
    echo "Unsupported OS. Please check the documentation for manual setup."
fi

echo ""
echo "🏗️ Building HelixCode..."
cd HelixCode
make build

echo ""
echo "✅ Setup complete!"
echo ""
echo "Next steps:"
echo "1. Set up your database (PostgreSQL)"
echo "2. Configure environment variables"
echo "3. Run the application: ./bin/helixcode"
echo ""
echo "For detailed instructions, see the README.md file."