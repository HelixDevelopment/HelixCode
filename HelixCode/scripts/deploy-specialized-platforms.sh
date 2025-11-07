#!/bin/bash
# HelixCode Specialized Platforms Deployment Script
# Unified deployment for Aurora OS and Harmony OS

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Functions
print_header() {
    echo -e "${CYAN}============================================${NC}"
    echo -e "${CYAN}  HelixCode Specialized Platforms Deployer${NC}"
    echo -e "${CYAN}============================================${NC}"
    echo ""
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

show_usage() {
    cat <<EOF
Usage: $0 [OPTIONS]

Deploy HelixCode clients for specialized platforms.

OPTIONS:
    -p, --platform PLATFORM    Platform to deploy (aurora, harmony, both)
    -b, --build                Build binaries before deployment
    -c, --clean                Clean existing installation before deploying
    -h, --help                 Show this help message

PLATFORMS:
    aurora    Deploy Aurora OS client
    harmony   Deploy Harmony OS client
    both      Deploy both Aurora and Harmony OS clients

EXAMPLES:
    # Deploy Aurora OS only
    $0 --platform aurora

    # Deploy both platforms with fresh builds
    $0 --platform both --build

    # Clean deploy Harmony OS
    $0 --platform harmony --clean

    # Interactive mode (no arguments)
    $0

EOF
}

detect_platform() {
    print_info "Detecting platform..."

    # Check for Aurora OS
    if [[ -f "/etc/aurora-release" ]] || grep -q "Aurora" /etc/os-release 2>/dev/null; then
        echo "aurora"
        return
    fi

    # Check for Harmony OS
    if command -v hdc >/dev/null 2>&1 || [[ -f "/system/etc/harmonyos-release" ]] || [[ -f "/etc/harmonyos-release" ]]; then
        echo "harmony"
        return
    fi

    # Default to asking user
    echo "unknown"
}

prompt_platform() {
    echo ""
    echo "Select platform to deploy:"
    echo "  1) Aurora OS"
    echo "  2) Harmony OS"
    echo "  3) Both platforms"
    echo "  4) Exit"
    echo ""
    read -p "Enter choice [1-4]: " choice

    case $choice in
        1) echo "aurora" ;;
        2) echo "harmony" ;;
        3) echo "both" ;;
        4) exit 0 ;;
        *)
            print_error "Invalid choice"
            exit 1
            ;;
    esac
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        print_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

build_platforms() {
    local platform=$1

    print_info "Building binaries..."

    cd "${SCRIPT_DIR}/.."

    case $platform in
        aurora)
            make aurora-os || {
                print_error "Failed to build Aurora OS client"
                exit 1
            }
            print_success "Aurora OS binary built"
            ;;
        harmony)
            make harmony-os || {
                print_error "Failed to build Harmony OS client"
                exit 1
            }
            print_success "Harmony OS binary built"
            ;;
        both)
            make aurora-harmony || {
                print_error "Failed to build platform binaries"
                exit 1
            }
            print_success "Both platform binaries built"
            ;;
    esac
}

clean_installation() {
    local service=$1

    print_info "Cleaning existing installation for $service..."

    # Stop service if running
    if command -v systemctl >/dev/null 2>&1; then
        systemctl stop "$service" 2>/dev/null || true
        systemctl disable "$service" 2>/dev/null || true
    elif [[ -f "/etc/init.d/$service" ]]; then
        /etc/init.d/"$service" stop 2>/dev/null || true
    fi

    # Remove service files
    rm -f "/etc/systemd/system/${service}.service"
    rm -f "/etc/init.d/${service}"

    # Clean directories (but preserve config and data)
    rm -f "/opt/helixcode/aurora-os"
    rm -f "/opt/helixcode/harmony-os"
    rm -f "/usr/local/bin/helixcode-aurora"
    rm -f "/usr/local/bin/helixcode-harmony"

    if command -v systemctl >/dev/null 2>&1; then
        systemctl daemon-reload 2>/dev/null || true
    fi

    print_success "Cleaned existing installation"
}

deploy_aurora() {
    print_header
    echo -e "${BLUE}Deploying Aurora OS Client${NC}"
    echo ""

    if [[ ! -f "${SCRIPT_DIR}/deploy-aurora-os.sh" ]]; then
        print_error "Aurora OS deployment script not found"
        exit 1
    fi

    bash "${SCRIPT_DIR}/deploy-aurora-os.sh"
}

deploy_harmony() {
    print_header
    echo -e "${BLUE}Deploying Harmony OS Client${NC}"
    echo ""

    if [[ ! -f "${SCRIPT_DIR}/deploy-harmony-os.sh" ]]; then
        print_error "Harmony OS deployment script not found"
        exit 1
    fi

    bash "${SCRIPT_DIR}/deploy-harmony-os.sh"
}

deploy_both() {
    print_header
    echo -e "${BLUE}Deploying Aurora and Harmony OS Clients${NC}"
    echo ""

    # Deploy Aurora OS
    echo -e "${CYAN}--- Aurora OS Deployment ---${NC}"
    if bash "${SCRIPT_DIR}/deploy-aurora-os.sh"; then
        print_success "Aurora OS deployment completed"
    else
        print_error "Aurora OS deployment failed"
    fi

    echo ""
    echo -e "${CYAN}--- Harmony OS Deployment ---${NC}"

    # Deploy Harmony OS
    if bash "${SCRIPT_DIR}/deploy-harmony-os.sh"; then
        print_success "Harmony OS deployment completed"
    else
        print_error "Harmony OS deployment failed"
    fi

    echo ""
    print_success "Both platforms deployed successfully"
}

show_deployment_summary() {
    local platform=$1

    echo ""
    echo -e "${GREEN}============================================${NC}"
    echo -e "${GREEN}  Deployment Summary${NC}"
    echo -e "${GREEN}============================================${NC}"
    echo ""

    case $platform in
        aurora)
            echo "Platform: Aurora OS"
            echo "Binary: /opt/helixcode/aurora-os"
            echo "Config: /etc/helixcode/aurora-config.yaml"
            echo "Service: helixcode-aurora.service"
            echo "Command: helixcode-aurora"
            ;;
        harmony)
            echo "Platform: Harmony OS"
            echo "Binary: /opt/helixcode/harmony-os"
            echo "Config: /etc/helixcode/harmony-config.yaml"
            echo "Service: helixcode-harmony.service"
            echo "Command: helixcode-harmony"
            ;;
        both)
            echo "Platforms: Aurora OS + Harmony OS"
            echo ""
            echo "Aurora OS:"
            echo "  Binary: /opt/helixcode/aurora-os"
            echo "  Config: /etc/helixcode/aurora-config.yaml"
            echo "  Service: helixcode-aurora.service"
            echo ""
            echo "Harmony OS:"
            echo "  Binary: /opt/helixcode/harmony-os"
            echo "  Config: /etc/helixcode/harmony-config.yaml"
            echo "  Service: helixcode-harmony.service"
            ;;
    esac

    echo ""
    echo "Logs: /var/log/helixcode/"
    echo "Data: /var/lib/helixcode/"
    echo ""
}

# Main function
main() {
    local platform=""
    local build=false
    local clean=false

    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -p|--platform)
                platform="$2"
                shift 2
                ;;
            -b|--build)
                build=true
                shift
                ;;
            -c|--clean)
                clean=true
                shift
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done

    # Check for root privileges
    check_root

    # If no platform specified, try to detect or prompt
    if [[ -z "$platform" ]]; then
        detected=$(detect_platform)
        if [[ "$detected" == "unknown" ]]; then
            platform=$(prompt_platform)
        else
            print_success "Detected platform: $detected"
            platform=$detected
        fi
    fi

    # Validate platform
    if [[ ! "$platform" =~ ^(aurora|harmony|both)$ ]]; then
        print_error "Invalid platform: $platform"
        print_info "Valid platforms: aurora, harmony, both"
        exit 1
    fi

    print_header
    echo "Platform: $platform"
    echo "Build: $build"
    echo "Clean: $clean"
    echo ""

    # Build if requested
    if $build; then
        build_platforms "$platform"
        echo ""
    fi

    # Clean if requested
    if $clean; then
        case $platform in
            aurora)
                clean_installation "helixcode-aurora"
                ;;
            harmony)
                clean_installation "helixcode-harmony"
                ;;
            both)
                clean_installation "helixcode-aurora"
                clean_installation "helixcode-harmony"
                ;;
        esac
        echo ""
    fi

    # Deploy
    case $platform in
        aurora)
            deploy_aurora
            ;;
        harmony)
            deploy_harmony
            ;;
        both)
            deploy_both
            ;;
    esac

    # Show summary
    show_deployment_summary "$platform"

    print_success "Deployment completed successfully!"
}

# Run main function
main "$@"
