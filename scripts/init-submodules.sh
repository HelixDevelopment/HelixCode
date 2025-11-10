#!/bin/bash

# HelixCode Submodule Initialization Script
# This script initializes and updates all git submodules for the HelixCode project

set -e

echo "Initializing and updating git submodules for HelixCode..."

# Initialize and update all submodules recursively
git submodule update --init --recursive

echo "Submodules initialized successfully!"

# Optional: Check if any submodules failed to initialize
if git submodule status | grep -q "^-"; then
    echo "Warning: Some submodules may not be properly initialized."
    echo "Run 'git submodule status' to check the status."
else
    echo "All submodules are properly initialized."
fi