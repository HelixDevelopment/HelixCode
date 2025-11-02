#!/bin/bash

# Generate SSH keys for testing
echo "Generating SSH keys for HelixCode testing..."

# Create keys directory
mkdir -p test/ssh-keys

# Generate server key
ssh-keygen -t rsa -b 2048 -f test/ssh-keys/id_rsa -N "" -C "helixcode-test"

# Generate worker keys (copy server key for simplicity in testing)
cp test/ssh-keys/id_rsa test/ssh-keys/worker_key
cp test/ssh-keys/id_rsa.pub test/ssh-keys/worker_key.pub

# Set proper permissions
chmod 600 test/ssh-keys/id_rsa
chmod 644 test/ssh-keys/id_rsa.pub
chmod 600 test/ssh-keys/worker_key
chmod 644 test/ssh-keys/worker_key.pub

# Create known_hosts with localhost entry (for testing)
ssh-keyscan -H localhost >> test/ssh-keys/known_hosts 2>/dev/null || echo "localhost ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhTfsHjvqFs7u0dB6A8" >> test/ssh-keys/known_hosts

echo "SSH keys generated successfully!"
echo "Server key: test/ssh-keys/id_rsa"
echo "Worker key: test/ssh-keys/worker_key"