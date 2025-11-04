#!/bin/bash

# Aggressive Docker Cleanup Script
# Skips diagnostics and goes straight to cleanup

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { echo -e "${BLUE}▶ $1${NC}"; }
print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_error() { echo -e "${RED}✗ $1${NC}"; }

echo ""
echo -e "${YELLOW}╔════════════════════════════════════════════╗${NC}"
echo -e "${YELLOW}║  Aggressive Docker Cleanup - HelixCode     ║${NC}"
echo -e "${YELLOW}╔════════════════════════════════════════════╗${NC}"
echo ""

# Check Docker
if ! docker info &> /dev/null; then
    print_error "Docker is not running"
    exit 1
fi

# 1. Stop containers forcefully
print_info "Stopping all containers..."
docker ps -q | xargs -r docker stop 2>/dev/null || true
print_success "Containers stopped"

# 2. Remove all containers
print_info "Removing all containers..."
docker ps -aq | xargs -r docker rm -f 2>/dev/null || true
print_success "Containers removed"

# 3. Remove all images
print_info "Removing all images..."
docker images -q | xargs -r docker rmi -f 2>/dev/null || true
print_success "Images removed"

# 4. Remove all volumes
print_info "Removing all volumes..."
docker volume ls -q | xargs -r docker volume rm -f 2>/dev/null || true
print_success "Volumes removed"

# 5. Remove all networks (except defaults)
print_info "Removing custom networks..."
docker network ls -q | xargs -r docker network rm 2>/dev/null || true
print_success "Networks removed"

# 6. Clear BuildKit cache
print_info "Clearing BuildKit cache..."
docker builder prune -a -f 2>/dev/null || true
print_success "BuildKit cache cleared"

# 7. System prune
print_info "Final system prune..."
docker system prune -a -f --volumes 2>/dev/null || true
print_success "System pruned"

# 8. Clean Homebrew
print_info "Cleaning Homebrew..."
if command -v brew &> /dev/null; then
    brew cleanup -s 2>/dev/null || true
    print_success "Homebrew cleaned"
fi

echo ""
echo -e "${GREEN}╔════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  Cleanup Complete!                         ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════╝${NC}"
echo ""
print_info "Next: Restart Docker Desktop, then run 'helix start'"
echo ""
