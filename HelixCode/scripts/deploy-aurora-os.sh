#!/bin/bash
# Aurora OS Deployment Script
# Deploys HelixCode Aurora OS client to Aurora OS systems

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="aurora-os"
BINARY_PATH="./bin/${BINARY_NAME}"
INSTALL_DIR="/opt/helixcode"
CONFIG_DIR="/etc/helixcode"
SYSTEMD_SERVICE="helixcode-aurora.service"

# Functions
print_header() {
    echo -e "${BLUE}================================${NC}"
    echo -e "${BLUE}  HelixCode Aurora OS Deployer${NC}"
    echo -e "${BLUE}================================${NC}"
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

check_root() {
    if [[ $EUID -ne 0 ]]; then
        print_error "This script must be run as root (use sudo)"
        exit 1
    fi
    print_success "Root privileges confirmed"
}

check_aurora_os() {
    if [[ ! -f "/etc/aurora-release" ]] && [[ ! -f "/etc/os-release" ]]; then
        print_warning "Aurora OS detection uncertain, continuing anyway..."
    elif grep -q "Aurora" /etc/os-release 2>/dev/null || [[ -f "/etc/aurora-release" ]]; then
        print_success "Aurora OS detected"
    else
        print_warning "Not running on Aurora OS, some features may not work"
    fi
}

check_binary() {
    if [[ ! -f "${BINARY_PATH}" ]]; then
        print_error "Binary not found at ${BINARY_PATH}"
        print_info "Run 'make aurora-os' to build the binary first"
        exit 1
    fi
    print_success "Binary found: ${BINARY_PATH}"
}

check_dependencies() {
    print_info "Checking system dependencies..."

    local missing_deps=()

    # Check for required packages
    command -v systemctl >/dev/null 2>&1 || missing_deps+=("systemd")

    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        print_warning "Missing dependencies: ${missing_deps[*]}"
        print_info "Install with: sudo apt install ${missing_deps[*]}"
    else
        print_success "All dependencies satisfied"
    fi
}

create_directories() {
    print_info "Creating installation directories..."

    mkdir -p "${INSTALL_DIR}"
    mkdir -p "${CONFIG_DIR}"
    mkdir -p "/var/log/helixcode"
    mkdir -p "/var/lib/helixcode"

    print_success "Directories created"
}

install_binary() {
    print_info "Installing Aurora OS binary..."

    cp "${BINARY_PATH}" "${INSTALL_DIR}/"
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

    # Create symlink for easy access
    ln -sf "${INSTALL_DIR}/${BINARY_NAME}" /usr/local/bin/helixcode-aurora

    print_success "Binary installed to ${INSTALL_DIR}/${BINARY_NAME}"
}

create_config() {
    print_info "Creating default configuration..."

    cat > "${CONFIG_DIR}/aurora-config.yaml" <<EOF
# HelixCode Aurora OS Configuration
server:
  address: "0.0.0.0"
  port: 8080

database:
  host: "localhost"
  port: 5432
  user: "helixcode"
  dbname: "helixcode"
  # Set password via HELIX_DATABASE_PASSWORD environment variable

redis:
  enabled: false
  host: "localhost"
  port: 6379

auth:
  token_expiry: 86400
  session_expiry: 604800

aurora:
  enable_security_features: true
  enable_system_monitoring: true
  enable_native_integration: true
  security_level: "enhanced"

workers:
  health_check_interval: 30
  max_concurrent_tasks: 10

logging:
  level: "info"
  file: "/var/log/helixcode/aurora-os.log"
EOF

    chmod 644 "${CONFIG_DIR}/aurora-config.yaml"
    print_success "Configuration created at ${CONFIG_DIR}/aurora-config.yaml"
}

create_systemd_service() {
    print_info "Creating systemd service..."

    cat > "/etc/systemd/system/${SYSTEMD_SERVICE}" <<EOF
[Unit]
Description=HelixCode Aurora OS Client
After=network.target postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=helixcode
Group=helixcode
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/${BINARY_NAME}
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

# Environment
Environment="HELIX_CONFIG_PATH=${CONFIG_DIR}/aurora-config.yaml"
EnvironmentFile=-/etc/helixcode/aurora.env

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/helixcode /var/lib/helixcode

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    print_success "Systemd service created: ${SYSTEMD_SERVICE}"
}

create_user() {
    if id "helixcode" &>/dev/null; then
        print_info "User 'helixcode' already exists"
    else
        print_info "Creating helixcode user..."
        useradd -r -s /bin/false -d /var/lib/helixcode helixcode
        print_success "User 'helixcode' created"
    fi
}

set_permissions() {
    print_info "Setting file permissions..."

    chown -R helixcode:helixcode "${INSTALL_DIR}"
    chown -R helixcode:helixcode "${CONFIG_DIR}"
    chown -R helixcode:helixcode "/var/log/helixcode"
    chown -R helixcode:helixcode "/var/lib/helixcode"

    chmod 750 "${INSTALL_DIR}"
    chmod 750 "${CONFIG_DIR}"
    chmod 755 "/var/log/helixcode"
    chmod 755 "/var/lib/helixcode"

    print_success "Permissions set"
}

print_next_steps() {
    echo ""
    echo -e "${GREEN}================================${NC}"
    echo -e "${GREEN}  Installation Complete!${NC}"
    echo -e "${GREEN}================================${NC}"
    echo ""
    echo "Next steps:"
    echo ""
    echo "1. Configure database connection:"
    echo "   Edit ${CONFIG_DIR}/aurora-config.yaml"
    echo "   Set HELIX_DATABASE_PASSWORD in /etc/helixcode/aurora.env"
    echo ""
    echo "2. Start the service:"
    echo "   sudo systemctl start ${SYSTEMD_SERVICE}"
    echo ""
    echo "3. Enable auto-start on boot:"
    echo "   sudo systemctl enable ${SYSTEMD_SERVICE}"
    echo ""
    echo "4. Check status:"
    echo "   sudo systemctl status ${SYSTEMD_SERVICE}"
    echo ""
    echo "5. View logs:"
    echo "   sudo journalctl -u ${SYSTEMD_SERVICE} -f"
    echo ""
    echo "Binary location: ${INSTALL_DIR}/${BINARY_NAME}"
    echo "Config location: ${CONFIG_DIR}/aurora-config.yaml"
    echo "Command symlink: /usr/local/bin/helixcode-aurora"
    echo ""
}

# Main deployment flow
main() {
    print_header

    check_root
    check_aurora_os
    check_binary
    check_dependencies

    create_user
    create_directories
    install_binary
    create_config
    create_systemd_service
    set_permissions

    print_next_steps
}

# Run main function
main "$@"
