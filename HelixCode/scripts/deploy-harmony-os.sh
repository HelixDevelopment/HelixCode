#!/bin/bash
# Harmony OS Deployment Script
# Deploys HelixCode Harmony OS client to Harmony OS systems

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="harmony-os"
BINARY_PATH="./bin/${BINARY_NAME}"
INSTALL_DIR="/opt/helixcode"
CONFIG_DIR="/etc/helixcode"
SERVICE_NAME="helixcode-harmony"

# Functions
print_header() {
    echo -e "${BLUE}================================${NC}"
    echo -e "${BLUE}  HelixCode Harmony OS Deployer${NC}"
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

check_harmony_os() {
    # Check for Harmony OS indicators
    if command -v hdc >/dev/null 2>&1; then
        print_success "Harmony OS development tools detected"
    elif [[ -f "/system/etc/harmonyos-release" ]] || [[ -f "/etc/harmonyos-release" ]]; then
        print_success "Harmony OS detected"
    else
        print_warning "Harmony OS detection uncertain"
        print_info "Continuing deployment - some features may require Harmony OS runtime"
    fi
}

check_binary() {
    if [[ ! -f "${BINARY_PATH}" ]]; then
        print_error "Binary not found at ${BINARY_PATH}"
        print_info "Run 'make harmony-os' to build the binary first"
        exit 1
    fi
    print_success "Binary found: ${BINARY_PATH}"
}

check_dependencies() {
    print_info "Checking system dependencies..."

    local missing_deps=()

    # Check for systemd (if available on Harmony OS)
    if command -v systemctl >/dev/null 2>&1; then
        print_success "Systemd available"
    else
        print_warning "Systemd not found - will create init script instead"
    fi
}

create_directories() {
    print_info "Creating installation directories..."

    mkdir -p "${INSTALL_DIR}"
    mkdir -p "${CONFIG_DIR}"
    mkdir -p "/var/log/helixcode"
    mkdir -p "/var/lib/helixcode"
    mkdir -p "/var/lib/helixcode/distributed"
    mkdir -p "/var/lib/helixcode/services"

    print_success "Directories created"
}

install_binary() {
    print_info "Installing Harmony OS binary..."

    cp "${BINARY_PATH}" "${INSTALL_DIR}/"
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

    # Create symlink for easy access
    ln -sf "${INSTALL_DIR}/${BINARY_NAME}" /usr/local/bin/helixcode-harmony

    print_success "Binary installed to ${INSTALL_DIR}/${BINARY_NAME}"
}

create_config() {
    print_info "Creating default configuration..."

    cat > "${CONFIG_DIR}/harmony-config.yaml" <<EOF
# HelixCode Harmony OS Configuration
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
  enabled: true
  host: "localhost"
  port: 6379

auth:
  token_expiry: 86400
  session_expiry: 604800

harmony:
  # Distributed computing features
  enable_distributed_computing: true
  enable_cross_device_sync: true
  sync_interval: 30

  # Resource management
  enable_resource_optimization: true
  enable_ai_acceleration: true
  gpu_enabled: true

  # System integration
  enable_system_monitoring: true
  enable_multi_screen: true
  enable_super_device: true

  # Service coordination
  service_discovery_enabled: true
  service_failover_enabled: true
  health_check_interval: 15

workers:
  health_check_interval: 30
  max_concurrent_tasks: 20
  distributed_mode: true

tasks:
  enable_distributed_execution: true
  checkpoint_interval: 300
  max_retries: 3

logging:
  level: "info"
  file: "/var/log/helixcode/harmony-os.log"
  enable_distributed_logging: true
EOF

    chmod 644 "${CONFIG_DIR}/harmony-config.yaml"
    print_success "Configuration created at ${CONFIG_DIR}/harmony-config.yaml"
}

create_systemd_service() {
    if ! command -v systemctl >/dev/null 2>&1; then
        print_info "Systemd not available, skipping service creation"
        return
    fi

    print_info "Creating systemd service..."

    cat > "/etc/systemd/system/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=HelixCode Harmony OS Client
After=network.target postgresql.service redis.service
Wants=postgresql.service redis.service

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
Environment="HELIX_CONFIG_PATH=${CONFIG_DIR}/harmony-config.yaml"
EnvironmentFile=-/etc/helixcode/harmony.env

# Resource limits for distributed computing
LimitNOFILE=65536
LimitNPROC=4096

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
    print_success "Systemd service created: ${SERVICE_NAME}.service"
}

create_init_script() {
    if command -v systemctl >/dev/null 2>&1; then
        print_info "Systemd available, skipping init script creation"
        return
    fi

    print_info "Creating init script for Harmony OS..."

    cat > "/etc/init.d/${SERVICE_NAME}" <<'EOF'
#!/bin/sh
### BEGIN INIT INFO
# Provides:          helixcode-harmony
# Required-Start:    $network $local_fs
# Required-Stop:     $network $local_fs
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: HelixCode Harmony OS Client
### END INIT INFO

NAME=helixcode-harmony
DAEMON=/opt/helixcode/harmony-os
PIDFILE=/var/run/${NAME}.pid
USER=helixcode
GROUP=helixcode

export HELIX_CONFIG_PATH=/etc/helixcode/harmony-config.yaml

case "$1" in
  start)
    echo "Starting $NAME..."
    start-stop-daemon --start --quiet --pidfile $PIDFILE \
      --chuid $USER:$GROUP --background --make-pidfile \
      --exec $DAEMON
    echo "$NAME started."
    ;;
  stop)
    echo "Stopping $NAME..."
    start-stop-daemon --stop --quiet --pidfile $PIDFILE
    rm -f $PIDFILE
    echo "$NAME stopped."
    ;;
  restart)
    $0 stop
    sleep 2
    $0 start
    ;;
  status)
    if [ -f $PIDFILE ]; then
      echo "$NAME is running (PID: $(cat $PIDFILE))"
    else
      echo "$NAME is not running"
    fi
    ;;
  *)
    echo "Usage: $0 {start|stop|restart|status}"
    exit 1
    ;;
esac

exit 0
EOF

    chmod +x "/etc/init.d/${SERVICE_NAME}"
    print_success "Init script created: /etc/init.d/${SERVICE_NAME}"
}

create_user() {
    if id "helixcode" &>/dev/null; then
        print_info "User 'helixcode' already exists"
    else
        print_info "Creating helixcode user..."
        useradd -r -s /bin/false -d /var/lib/helixcode helixcode 2>/dev/null || \
            adduser -S -s /bin/false -h /var/lib/helixcode helixcode 2>/dev/null || \
            print_warning "Could not create user, may need manual creation"
        print_success "User 'helixcode' created"
    fi
}

set_permissions() {
    print_info "Setting file permissions..."

    chown -R helixcode:helixcode "${INSTALL_DIR}" 2>/dev/null || \
        chown -R helixcode "${INSTALL_DIR}"
    chown -R helixcode:helixcode "${CONFIG_DIR}" 2>/dev/null || \
        chown -R helixcode "${CONFIG_DIR}"
    chown -R helixcode:helixcode "/var/log/helixcode" 2>/dev/null || \
        chown -R helixcode "/var/log/helixcode"
    chown -R helixcode:helixcode "/var/lib/helixcode" 2>/dev/null || \
        chown -R helixcode "/var/lib/helixcode"

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
    echo "1. Configure database and Redis:"
    echo "   Edit ${CONFIG_DIR}/harmony-config.yaml"
    echo "   Set HELIX_DATABASE_PASSWORD in /etc/helixcode/harmony.env"
    echo ""

    if command -v systemctl >/dev/null 2>&1; then
        echo "2. Start the service:"
        echo "   sudo systemctl start ${SERVICE_NAME}"
        echo ""
        echo "3. Enable auto-start on boot:"
        echo "   sudo systemctl enable ${SERVICE_NAME}"
        echo ""
        echo "4. Check status:"
        echo "   sudo systemctl status ${SERVICE_NAME}"
        echo ""
        echo "5. View logs:"
        echo "   sudo journalctl -u ${SERVICE_NAME} -f"
    else
        echo "2. Start the service:"
        echo "   sudo /etc/init.d/${SERVICE_NAME} start"
        echo ""
        echo "3. Check status:"
        echo "   sudo /etc/init.d/${SERVICE_NAME} status"
        echo ""
        echo "4. View logs:"
        echo "   sudo tail -f /var/log/helixcode/harmony-os.log"
    fi

    echo ""
    echo "Harmony OS Specific Features:"
    echo "  • Distributed computing engine"
    echo "  • Cross-device synchronization"
    echo "  • AI acceleration support"
    echo "  • Multi-screen collaboration"
    echo "  • Super Device integration"
    echo ""
    echo "Binary location: ${INSTALL_DIR}/${BINARY_NAME}"
    echo "Config location: ${CONFIG_DIR}/harmony-config.yaml"
    echo "Command symlink: /usr/local/bin/helixcode-harmony"
    echo ""
}

# Main deployment flow
main() {
    print_header

    check_root
    check_harmony_os
    check_binary
    check_dependencies

    create_user
    create_directories
    install_binary
    create_config
    create_systemd_service
    create_init_script
    set_permissions

    print_next_steps
}

# Run main function
main "$@"
