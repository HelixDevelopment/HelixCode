#!/bin/bash
set -e

# Function to check if a port is available
check_port() {
    local port=$1
    if command -v nc >/dev/null 2>&1; then
        if nc -z localhost $port 2>/dev/null; then
            return 1  # Port is in use
        else
            return 0  # Port is available
        fi
    else
        # Fallback using /proc/net/tcp
        if grep -q ":$(printf '%04X' $port)" /proc/net/tcp 2>/dev/null; then
            return 1
        else
            return 0
        fi
    fi
}

# Function to find available port
find_available_port() {
    local base_port=$1
    local port=$base_port
    
    while ! check_port $port; do
        port=$((port + 1))
        if [ $port -gt $((base_port + 100)) ]; then
            echo "ERROR: Could not find available port starting from $base_port" >&2
            exit 1
        fi
    done
    echo $port
}

# Function to broadcast configuration
broadcast_config() {
    local config_file="/shared/helix-config.json"
    local server_ip=$(hostname -i)
    
    cat > "$config_file" << EOF
{
    "server": {
        "host": "$server_ip",
        "api_port": $HELIX_API_PORT,
        "ssh_port": $HELIX_SSH_PORT,
        "web_port": $HELIX_WEB_PORT,
        "version": "1.0.0",
        "started_at": "$(date -Iseconds)"
    },
    "network": {
        "mode": "$HELIX_NETWORK_MODE",
        "broadcast_enabled": true,
        "discovery_port": 5353
    },
    "services": {
        "api": "http://$server_ip:$HELIX_API_PORT",
        "ssh": "$server_ip:$HELIX_SSH_PORT",
        "web": "http://$server_ip:$HELIX_WEB_PORT"
    }
}
EOF
    
    echo "📡 Configuration broadcasted to: $config_file"
}

# Function to start discovery service
start_discovery_service() {
    if [ "$HELIX_NETWORK_MODE" = "distributed" ]; then
        echo "🔍 Starting network discovery service..."
        # Simple discovery service using netcat for broadcasting
        while true; do
            echo "HELIX_DISCOVERY:$HELIX_API_PORT:$HELIX_SSH_PORT:$HELIX_WEB_PORT" | \
            nc -w 1 -u -b 255.255.255.255 5353 2>/dev/null || true
            sleep 30
        done &
        DISCOVERY_PID=$!
    fi
}

# Function to display startup information
display_startup_info() {
    local server_ip=$(hostname -i)
    
    echo ""
    echo "🚀 HelixCode Docker Container Started Successfully!"
    echo "=================================================="
    echo ""
    echo "📊 Available Services:"
    echo "   • REST API:       http://$server_ip:$HELIX_API_PORT"
    echo "   • SSH Workers:    $server_ip:$HELIX_SSH_PORT"
    echo "   • Web Interface:  http://$server_ip:$HELIX_WEB_PORT"
    echo ""
    echo "📁 Mounted Directories:"
    echo "   • Workspace:      /workspace"
    echo "   • Projects:       /projects"
    echo "   • Shared:         /shared"
    echo ""
    echo "🔧 Available Commands:"
    echo "   • helix server    - Start the REST API server"
    echo "   • helix cli       - Run the CLI interface"
    echo "   • helix tui       - Run the terminal UI"
    echo "   • helix help      - Show help information"
    echo ""
    echo "🌐 Network Mode: $HELIX_NETWORK_MODE"
    if [ "$HELIX_NETWORK_MODE" = "distributed" ]; then
        echo "🔍 Discovery: Enabled (port 5353)"
    fi
    echo ""
}

# Function to handle different commands
handle_command() {
    case "$1" in
        "server")
            echo "🌐 Starting HelixCode Server..."
            exec ./server
            ;;
        "cli")
            echo "💻 Starting HelixCode CLI..."
            exec ./cli "${@:2}"
            ;;
        "tui")
            echo "🖥️  Starting HelixCode Terminal UI..."
            exec ./terminal-ui
            ;;
        "help"|"--help"|"-h")
            echo "HelixCode Docker Container Usage:"
            echo ""
            echo "Commands:"
            echo "  server    - Start the REST API server"
            echo "  cli       - Run the CLI interface"
            echo "  tui       - Run the terminal UI"
            echo "  help      - Show this help"
            echo ""
            echo "Examples:"
            echo "  docker exec helixcode helix cli --help"
            echo "  docker exec helixcode helix tui"
            echo "  docker exec helixcode helix server"
            ;;
        *)
            if [ -z "$1" ]; then
                # Default behavior - start server
                echo "🌐 Starting HelixCode Server (default)..."
                exec ./server
            else
                echo "Unknown command: $1"
                echo "Use 'helix help' for available commands."
                exit 1
            fi
            ;;
    esac
}

# Main execution
main() {
    # Set default ports
    HELIX_API_PORT=${HELIX_API_PORT:-8080}
    HELIX_SSH_PORT=${HELIX_SSH_PORT:-2222}
    HELIX_WEB_PORT=${HELIX_WEB_PORT:-3000}
    HELIX_NETWORK_MODE=${HELIX_NETWORK_MODE:-standalone}
    
    # Check and adjust ports if needed
    if [ "$HELIX_AUTO_PORT" = "true" ]; then
        HELIX_API_PORT=$(find_available_port $HELIX_API_PORT)
        HELIX_SSH_PORT=$(find_available_port $HELIX_SSH_PORT)
        HELIX_WEB_PORT=$(find_available_port $HELIX_WEB_PORT)
        echo "🔧 Auto-port assignment enabled"
        echo "   API Port: $HELIX_API_PORT"
        echo "   SSH Port: $HELIX_SSH_PORT"
        echo "   Web Port: $HELIX_WEB_PORT"
    fi
    
    # Export adjusted ports
    export HELIX_API_PORT
    export HELIX_SSH_PORT
    export HELIX_WEB_PORT
    export HELIX_NETWORK_MODE
    
    # Broadcast configuration
    broadcast_config
    
    # Start discovery service if in distributed mode
    start_discovery_service
    
    # Display startup information
    display_startup_info
    
    # Handle the command
    handle_command "$@"
}

# Run main function
main "$@"