#!/bin/sh
set -e

echo "Starting HelixCode worker initialization..."

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL..."
while ! pg_isready -h $POSTGRES_HOST -U helixcode -d helixcode_test; do
  echo "PostgreSQL is unavailable - sleeping"
  sleep 2
done
echo "PostgreSQL is ready!"

# Wait for Redis to be ready
echo "Waiting for Redis..."
while ! redis-cli -h $REDIS_HOST ping | grep -q PONG; do
  echo "Redis is unavailable - sleeping"
  sleep 2
done
echo "Redis is ready!"

# Generate SSH host keys if they don't exist
if [ ! -f /etc/ssh/ssh_host_rsa_key ]; then
  echo "Generating SSH host keys..."
  ssh-keygen -A
fi

# Generate SSH keys if they don't exist
if [ ! -f /root/.ssh/id_rsa ]; then
  echo "Generating SSH keys..."
  ssh-keygen -t rsa -b 4096 -f /root/.ssh/id_rsa -N ""
fi

# Start SSH service in background
echo "Starting SSH service..."
/usr/sbin/sshd

# Register worker with the system
echo "Registering worker $WORKER_ID..."

# Debug: print environment
echo "Environment variables:"
env | grep -E "(HELIX|JWT)" || echo "No HELIX or JWT vars found"
env | grep -E "(DATABASE|REDIS)" | head -5

# Start the worker process
echo "Starting worker process..."
exec /app/server --worker --id=$WORKER_ID --postgres-host=$POSTGRES_HOST --redis-host=$REDIS_HOST