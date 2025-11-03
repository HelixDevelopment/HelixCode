#!/bin/bash

# HelixCode Dependencies Installation Script for Ubuntu
# This script installs all required system libraries for the HelixCode project

set -e  # Exit immediately if a command exits with a non-zero status

echo "=================================="
echo "HelixCode Dependencies Installer"
echo "=================================="
echo ""

# Check if running as root
if [[ $EUID -eq 0 ]]; then
   echo "This script should NOT be run as root. Please run without sudo."
   echo "The script will prompt for sudo when needed."
   exit 1
fi

echo "Updating package lists..."
sudo apt update

echo ""
echo "Installing development tools..."
sudo apt install -y build-essential pkg-config

echo ""
echo "Installing graphics and windowing libraries (required for GUI applications)..."
sudo apt install -y \
    libx11-dev \
    libxrandr-dev \
    libxinerama-dev \
    libxcursor-dev \
    libxcomposite-dev \
    libxdamage-dev \
    libxss-dev \
    libxrandr-dev \
    libxss-dev \
    libxtst-dev \
    libxkbcommon-dev \
    libxkbcommon-x11-dev \
    libxcb-cursor-dev \
    libxcb-randr0-dev \
    libxcb-xtest0-dev \
    libxcb-shape0-dev \
    libxcb-xinerama0-dev \
    libgl1-mesa-dev \
    libglu1-mesa-dev \
    libxext-dev \
    libxfixes-dev \
    libxi-dev \
    libxrender-dev \
    libxmu-dev \
    libxpm-dev \
    libxft-dev \
    libxxf86vm-dev

echo ""
echo "Installing additional dependencies..."
sudo apt install -y \
    gcc \
    g++ \
    make \
    cmake \
    git \
    curl \
    wget

echo ""
echo "Installing PostgreSQL client libraries..."
sudo apt install -y libpq-dev

echo ""
echo "Installing Redis client libraries..."
sudo apt install -y libhiredis-dev

echo ""
echo "Attempting to install webkit library (may vary by Ubuntu version)..."
if ! sudo apt install -y libwebkit2gtk-4.0-dev 2>/dev/null; then
    echo "libwebkit2gtk-4.0-dev not available, trying alternative..."
    if ! sudo apt install -y libwebkit2gtk-4.1-dev 2>/dev/null; then
        echo "No webkit2gtk dev packages found, skipping (may be needed for some GUI features)"
    fi
fi

echo ""
echo "Installing other development libraries..."
sudo apt install -y \
    libasound2-dev \
    libgtk-3-dev

echo ""
echo "Installing additional tools..."
sudo apt install -y \
    postgresql-client \
    redis-tools \
    ssh

echo ""
echo "Installing Go if not already installed..."
if ! command -v go &> /dev/null; then
    echo "Go not found, installing..."
    LATEST_GO=$(curl -s https://go.dev/VERSION?m=text | head -n1)
    echo "Installing ${LATEST_GO}..."
    wget -O go.tar.gz "https://go.dev/dl/${LATEST_GO}.linux-amd64.tar.gz"
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf go.tar.gz
    rm go.tar.gz
    
    echo ""
    echo "Go installed successfully!"
    echo "Please add Go to your PATH by adding the following line to your ~/.bashrc or ~/.profile:"
    echo "export PATH=\$PATH:/usr/local/go/bin"
else
    echo "Go is already installed."
fi

echo ""
echo "Verifying installation of key libraries..."
echo "Checking for Xxf86vm library..."
if ldconfig -p | grep -q libXxf86vm; then
    echo "✅ libXxf86vm found"
else
    echo "❌ libXxf86vm not found"
fi

echo "Checking for GL library..."
if ldconfig -p | grep -q libGL; then
    echo "✅ libGL found"
else
    echo "❌ libGL not found"
fi

echo "Checking for X11 library..."
if ldconfig -p | grep -q libX11; then
    echo "✅ libX11 found"
else
    echo "❌ libX11 not found"
fi

echo ""
echo "Installation complete!"
echo ""
echo "To use Go, please run:"
echo "export PATH=\$PATH:/usr/local/go/bin"
echo ""
echo "Or add that line to your ~/.bashrc file to make it permanent:"
echo 'echo "export PATH=\$PATH:/usr/local/go/bin" >> ~/.bashrc'
echo "Then run: source ~/.bashrc"
echo ""
echo "You should now be able to build and test the HelixCode project successfully."