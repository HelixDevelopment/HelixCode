#!/bin/bash

# HelixCode Memory Provider Initialization Script
# This script clones external memory provider repositories when needed

set -e

echo "Initializing external memory provider repositories for HelixCode..."

# Create the external memory directory if it doesn't exist
mkdir -p HelixCode/external/memory

# Function to clone a repository if it doesn't exist
clone_if_not_exists() {
    local repo_name=$1
    local repo_url=$2
    local target_dir="HelixCode/external/memory/$repo_name"

    if [ -d "$target_dir" ]; then
        echo "Repository $repo_name already exists, skipping..."
        return 0
    fi

    echo "Cloning $repo_name from $repo_url..."
    git clone "$repo_url" "$target_dir"

    # For Go SDKs, we might want to keep only the Go parts
    if [ "$repo_name" = "zep" ]; then
        echo "Setting up Zep Go SDK..."
        # The Zep repository has Go SDK in legacy/src/go/
        # We can keep the whole repo for now as it contains the Go SDK
    fi
}

# Clone memory provider repositories
clone_if_not_exists "mem0" "https://github.com/mem0ai/mem0.git"
clone_if_not_exists "zep" "https://github.com/getzep/zep.git"
clone_if_not_exists "memonto" "https://github.com/memonto-ai/memonto.git"
clone_if_not_exists "baseai" "https://github.com/baseai/baseai.git"

echo "External memory provider repositories initialized successfully!"
echo ""
echo "Note: These repositories are cloned for reference and SDK access."
echo "They are git ignored and will be cloned automatically when needed."