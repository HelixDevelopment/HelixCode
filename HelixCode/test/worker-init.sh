#!/bin/bash

# Worker initialization script for Docker testing
echo "Initializing HelixCode worker node: $WORKER_ID"

# Set default values if not provided
WORKER_ID=${WORKER_ID:-"worker-1"}
WORKER_CAPABILITIES=${WORKER_CAPABILITIES:-"code-generation,testing"}
WORKER_MAX_TASKS=${WORKER_MAX_TASKS:-5}
HELIX_SERVER=${HELIX_SERVER:-"localhost:2222"}

# Create worker configuration
cat > /tmp/worker-config.json << EOF
{
  "id": "$WORKER_ID",
  "capabilities": ["$WORKER_CAPABILITIES"],
  "max_concurrent_tasks": $WORKER_MAX_TASKS,
  "server": "$HELIX_SERVER",
  "resources": {
    "cpu_count": $(nproc),
    "total_memory": $(free -b | awk 'NR==2{print $2}'),
    "gpu_count": $(lspci 2>/dev/null | grep -i nvidia | wc -l || echo 0)
  }
}
EOF

echo "Worker configuration created:"
cat /tmp/worker-config.json

# Simulate worker registration (in real implementation, this would connect to server)
echo "Attempting to register with HelixCode server at $HELIX_SERVER..."

# Keep container running
echo "Worker $WORKER_ID ready and waiting for tasks..."
while true; do
  sleep 30
  echo "Worker $WORKER_ID heartbeat - $(date)"
done