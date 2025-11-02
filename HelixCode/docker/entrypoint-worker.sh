#!/bin/bash

# HelixCode Worker Entrypoint Script
# This script initializes a worker node for distributed testing

set -e

echo "🚀 Starting HelixCode Worker Node: $WORKER_ID"

# Wait for dependencies
echo "⏳ Waiting for services to be ready..."
sleep 10

# Configure SSH if keys are provided
if [ -f "/root/.ssh/id_rsa" ]; then
    echo "🔐 SSH keys found, configuring SSH..."
    chmod 600 /root/.ssh/id_rsa
    chmod 644 /root/.ssh/id_rsa.pub
    chmod 644 /root/.ssh/authorized_keys 2>/dev/null || true
else
    echo "⚠️  No SSH keys provided, generating temporary keys..."
    ssh-keygen -t rsa -b 2048 -f /root/.ssh/id_rsa -N ""
    cat /root/.ssh/id_rsa.pub >> /root/.ssh/authorized_keys
fi

# Start SSH daemon
echo "🔧 Starting SSH daemon..."
/usr/sbin/sshd

# Wait for SSH to be ready
sleep 5

# Test SSH connection to server
if [ -n "$HELIX_SERVER" ]; then
    echo "🔗 Testing connection to HelixCode server: $HELIX_SERVER"
    # Extract host from HELIX_SERVER (format: host:port)
    SERVER_HOST=$(echo $HELIX_SERVER | cut -d: -f1)
    SERVER_PORT=$(echo $HELIX_SERVER | cut -d: -f2)

    # Wait for server to be ready
    echo "⏳ Waiting for HelixCode server to be ready..."
    for i in {1..30}; do
        if nc -z $SERVER_HOST $SERVER_PORT 2>/dev/null; then
            echo "✅ HelixCode server is ready"
            break
        fi
        echo "⏳ Waiting for server... ($i/30)"
        sleep 2
    done
fi

# Initialize worker capabilities based on environment
WORKER_CAPS=${WORKER_CAPABILITIES:-"code-generation,testing"}
MAX_TASKS=${WORKER_MAX_TASKS:-5}

echo "⚙️  Worker Configuration:"
echo "   ID: $WORKER_ID"
echo "   Capabilities: $WORKER_CAPS"
echo "   Max Tasks: $MAX_TASKS"
echo "   Server: $HELIX_SERVER"

# Create worker registration script
cat > /app/register-worker.sh << 'EOF'
#!/bin/bash
# Register this worker with the HelixCode server

SERVER_URL="http://helixcode-server:8080"
WORKER_DATA=$(cat <<JSON
{
  "id": "'$WORKER_ID'",
  "hostname": "'$HOSTNAME'",
  "capabilities": ["'$(echo $WORKER_CAPS | sed 's/,/","/g')'"],
  "max_concurrent_tasks": '$MAX_TASKS',
  "resources": {
    "cpu_count": 2,
    "total_memory": 4294967296,
    "gpu_count": 0
  }
}
JSON
)

echo "📝 Registering worker with server..."
curl -X POST "$SERVER_URL/api/workers/register" \
  -H "Content-Type: application/json" \
  -d "$WORKER_DATA" \
  --connect-timeout 10 \
  --max-time 30 \
  2>/dev/null || echo "⚠️  Worker registration failed (server may not be ready yet)"

echo "✅ Worker registration attempt completed"
EOF

chmod +x /app/register-worker.sh

# Register worker with server (retry a few times)
echo "🔄 Attempting worker registration..."
for i in {1..5}; do
    /app/register-worker.sh && break
    echo "⏳ Registration attempt $i failed, retrying in 5 seconds..."
    sleep 5
done

# Start heartbeat service
echo "💓 Starting worker heartbeat..."
(
    while true; do
        curl -X POST "http://helixcode-server:8080/api/workers/heartbeat" \
          -H "Content-Type: application/json" \
          -d "{\"worker_id\": \"$WORKER_ID\"}" \
          --connect-timeout 5 \
          --max-time 10 \
          2>/dev/null || true
        sleep 30
    done
) &

# Keep container running and show logs
echo "🎯 Worker $WORKER_ID is ready and operational"
echo "📊 Monitoring for tasks..."

# Monitor for tasks (simplified - in real implementation this would be more sophisticated)
while true; do
    # Check for available tasks
    TASKS=$(curl -s "http://helixcode-server:8080/api/tasks/available?worker_id=$WORKER_ID" 2>/dev/null || echo "[]")

    if [ "$TASKS" != "[]" ] && [ "$TASKS" != "" ]; then
        echo "🎯 Task available, processing..."
        # In a real implementation, this would execute the task
        sleep 5
    fi

    sleep 10
done