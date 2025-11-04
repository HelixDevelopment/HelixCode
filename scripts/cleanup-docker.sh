#!/bin/bash
set -e

# Docker and System Cleanup Script for HelixCode
# Fixes BuildKit I/O errors caused by full disk

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

print_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
print_success() { echo -e "${GREEN}✅ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
print_error() { echo -e "${RED}❌ $1${NC}"; }
print_header() { echo -e "${CYAN}=== $1 ===${NC}"; }

echo ""
print_header "Docker & System Cleanup for HelixCode"
echo ""

# Check if Docker is running
if ! docker info &> /dev/null; then
    print_error "Docker daemon is not running. Please start Docker Desktop."
    exit 1
fi

# Show disk usage BEFORE cleanup
print_header "Disk Usage BEFORE Cleanup"
df -h / | grep -E '/$|Filesystem'
echo ""

print_header "Docker Disk Usage BEFORE Cleanup"
docker system df 2>/dev/null || echo "Unable to get Docker disk usage"
echo ""

# Step 1: Stop all running containers
print_info "Step 1/8: Stopping all running containers..."
RUNNING_CONTAINERS=$(docker ps -q)
if [ -n "$RUNNING_CONTAINERS" ]; then
    docker stop $RUNNING_CONTAINERS
    print_success "Stopped all running containers"
else
    print_info "No running containers to stop"
fi
echo ""

# Step 2: Remove all stopped containers
print_info "Step 2/8: Removing all stopped containers..."
docker container prune -f
print_success "Removed stopped containers"
echo ""

# Step 3: Remove all unused images
print_info "Step 3/8: Removing all unused images (this may take a while)..."
docker image prune -a -f
print_success "Removed unused images"
echo ""

# Step 4: Remove all unused volumes
print_info "Step 4/8: Removing all unused volumes..."
docker volume prune -f
print_success "Removed unused volumes"
echo ""

# Step 5: Remove all unused networks
print_info "Step 5/8: Removing all unused networks..."
docker network prune -f
print_success "Removed unused networks"
echo ""

# Step 6: Clear BuildKit cache (the main culprit)
print_info "Step 6/8: Clearing BuildKit cache completely..."
docker builder prune -a -f
print_success "Cleared BuildKit cache"
echo ""

# Step 7: Full system prune
print_info "Step 7/8: Running full Docker system prune..."
docker system prune -a -f --volumes
print_success "Completed full system prune"
echo ""

# Step 8: Clean Homebrew cache (if available)
print_info "Step 8/8: Cleaning Homebrew cache..."
if command -v brew &> /dev/null; then
    brew cleanup -s 2>/dev/null || true
    print_success "Cleaned Homebrew cache"
else
    print_info "Homebrew not installed, skipping"
fi
echo ""

# Show disk usage AFTER cleanup
print_header "Disk Usage AFTER Cleanup"
df -h / | grep -E '/$|Filesystem'
echo ""

print_header "Docker Disk Usage AFTER Cleanup"
docker system df 2>/dev/null || echo "Unable to get Docker disk usage"
echo ""

# Summary
print_header "Cleanup Complete!"
echo ""
print_success "Docker BuildKit cache has been cleared"
print_success "All unused Docker resources have been removed"
print_warning "You may need to restart Docker Desktop for best results"
echo ""
print_info "Next steps:"
echo "  1. Restart Docker Desktop (recommended)"
echo "  2. Run: cd /Users/milosvasic/Projects/HelixCode && helix start"
echo ""
