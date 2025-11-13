# HelixCode Specialized Platforms - Quick Start Guide

## Overview

This guide helps you quickly get started with HelixCode on Aurora OS (Russian platform) and Harmony OS (Chinese platform). Both platforms offer specialized features optimized for their respective ecosystems.

---

## Quick Platform Comparison

| Feature | Aurora OS | Harmony OS |
|---------|-----------|------------|
| **Target Market** | Russian Federation | China / Global |
| **Focus** | Security & System Integration | Distributed Computing & AI |
| **Key Features** | Enhanced security, system monitoring | NPU acceleration, cross-device sync |
| **Hardware Support** | Standard x86/ARM | Kirin chipsets, NPU/GPU |
| **Best For** | Secure environments, government | Multi-device workflows, AI workloads |

---

## Aurora OS - Quick Start

### 5-Minute Setup

#### Step 1: Build

```bash
cd HelixCode
make aurora-os
```

**Output**: `bin/aurora-os` (approximately 56 MB)

#### Step 2: Deploy (Automated)

```bash
sudo ./scripts/deploy-aurora-os.sh
```

This script automatically:
- Installs binary to `/opt/helixcode/`
- Creates configuration files
- Sets up systemd service
- Configures permissions

#### Step 3: Configure Database

Edit `/etc/helixcode/aurora.env`:

```bash
HELIX_DATABASE_PASSWORD=your_secure_password
HELIX_AUTH_JWT_SECRET=your_jwt_secret_key
```

Create database:

```bash
sudo -u postgres createdb helixcode
sudo -u postgres createuser helixcode
sudo -u postgres psql -c "ALTER USER helixcode WITH PASSWORD 'your_secure_password';"
```

#### Step 4: Start Service

```bash
sudo systemctl start helixcode-aurora
sudo systemctl enable helixcode-aurora
```

#### Step 5: Verify

```bash
# Check status
sudo systemctl status helixcode-aurora

# View logs
sudo journalctl -u helixcode-aurora -f

# Test health endpoint
curl http://localhost:8080/health
```

**Done!** Access the UI at `http://localhost:8080`

### Quick Configuration

Minimal `/etc/helixcode/aurora-config.yaml`:

```yaml
server:
  address: "0.0.0.0"
  port: 8080

database:
  host: "localhost"
  port: 5432
  user: "helixcode"
  dbname: "helixcode"

aurora:
  enable_security_features: true
  enable_system_monitoring: true
  security_level: "enhanced"

workers:
  max_concurrent_tasks: 10

logging:
  level: "info"
  file: "/var/log/helixcode/aurora-os.log"
```

### Common Aurora OS Commands

```bash
# Start/stop service
sudo systemctl start helixcode-aurora
sudo systemctl stop helixcode-aurora

# View logs
sudo journalctl -u helixcode-aurora -f

# Check configuration
helixcode-aurora --config /etc/helixcode/aurora-config.yaml --validate

# System diagnostics
helixcode-aurora diagnostics

# Security status
helixcode-aurora security status
```

---

## Harmony OS - Quick Start

### 5-Minute Setup

#### Step 1: Build

```bash
cd HelixCode
make harmony-os
```

**Output**: `bin/harmony-os` (approximately 56 MB)

#### Step 2: Deploy (Automated)

```bash
sudo ./scripts/deploy-harmony-os.sh
```

This script automatically:
- Installs binary to `/opt/helixcode/`
- Creates configuration files
- Sets up systemd service or init script
- Configures directories and permissions

#### Step 3: Configure Database & Redis

Edit `/etc/helixcode/harmony.env`:

```bash
HELIX_DATABASE_PASSWORD=your_secure_password
HELIX_REDIS_PASSWORD=your_redis_password
HELIX_AUTH_JWT_SECRET=your_jwt_secret_key
```

Create database:

```bash
sudo -u postgres createdb helixcode
sudo -u postgres createuser helixcode
sudo -u postgres psql -c "ALTER USER helixcode WITH PASSWORD 'your_secure_password';"
```

Install Redis (optional but recommended):

```bash
sudo apt install redis-server
sudo systemctl enable redis-server
sudo systemctl start redis-server
```

#### Step 4: Start Service

```bash
# If systemd is available
sudo systemctl start helixcode-harmony
sudo systemctl enable helixcode-harmony

# Or using init script
sudo /etc/init.d/helixcode-harmony start
```

#### Step 5: Verify

```bash
# Check status
sudo systemctl status helixcode-harmony

# View logs
sudo journalctl -u helixcode-harmony -f

# Test health endpoint
curl http://localhost:8080/health

# Check distributed engine
helixcode-harmony distributed status
```

**Done!** Access the UI at `http://localhost:8080`

### Quick Configuration

Minimal `/etc/helixcode/harmony-config.yaml`:

```yaml
server:
  address: "0.0.0.0"
  port: 8080

database:
  host: "localhost"
  port: 5432
  user: "helixcode"
  dbname: "helixcode"

redis:
  enabled: true
  host: "localhost"
  port: 6379

harmony:
  enable_distributed_computing: true
  enable_cross_device_sync: true
  enable_ai_acceleration: true
  enable_resource_optimization: true

workers:
  max_concurrent_tasks: 20
  distributed_mode: true

logging:
  level: "info"
  file: "/var/log/helixcode/harmony-os.log"
```

### Common Harmony OS Commands

```bash
# Start/stop service
sudo systemctl start helixcode-harmony
sudo systemctl stop helixcode-harmony

# View logs
sudo journalctl -u helixcode-harmony -f

# Distributed computing
helixcode-harmony distributed status
helixcode-harmony distributed workers

# Device info
helixcode-harmony device list
helixcode-harmony device npu-status

# Synchronization
helixcode-harmony sync status
helixcode-harmony sync now
```

---

## Both Platforms - Combined Deployment

### Deploy Both at Once

```bash
# Build both platforms
make aurora-harmony

# Deploy both (interactive)
sudo ./scripts/deploy-specialized-platforms.sh

# Or specify both explicitly
sudo ./scripts/deploy-specialized-platforms.sh --platform both --build
```

### Managing Both Services

```bash
# Start both
sudo systemctl start helixcode-aurora helixcode-harmony

# Stop both
sudo systemctl stop helixcode-aurora helixcode-harmony

# Status check
sudo systemctl status helixcode-aurora helixcode-harmony

# Enable on boot
sudo systemctl enable helixcode-aurora helixcode-harmony
```

### Port Configuration

If running both on the same machine, configure different ports:

**Aurora OS** (`/etc/helixcode/aurora-config.yaml`):
```yaml
server:
  port: 8080
```

**Harmony OS** (`/etc/helixcode/harmony-config.yaml`):
```yaml
server:
  port: 8081
```

---

## Essential Operations

### Creating Your First Task

#### Aurora OS

```bash
# Via command line
helixcode-aurora task create \
  --name "Security Scan" \
  --type "analysis" \
  --priority "high"

# Via API
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Security Scan",
    "type": "analysis",
    "priority": "high"
  }'
```

#### Harmony OS

```bash
# Via command line (with distributed computing)
helixcode-harmony task create \
  --name "AI Model Training" \
  --type "training" \
  --priority "high" \
  --distributed \
  --use-npu

# Via API
curl -X POST http://localhost:8081/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "AI Model Training",
    "type": "training",
    "priority": "high",
    "distributed": true,
    "device": "npu"
  }'
```

### Adding Workers

#### Aurora OS

```bash
# Add SSH worker
helixcode-aurora worker add \
  --host 192.168.1.100 \
  --user helix \
  --key ~/.ssh/id_rsa

# List workers
helixcode-aurora worker list
```

#### Harmony OS

```bash
# Add distributed worker
helixcode-harmony worker add \
  --host 192.168.1.101 \
  --user helix \
  --key ~/.ssh/id_rsa \
  --capabilities npu,gpu

# Enable auto-discovery
helixcode-harmony distributed discovery enable
```

### Monitoring

#### Aurora OS

```bash
# System status
helixcode-aurora status

# Security monitoring
helixcode-aurora security monitor

# Resource usage
helixcode-aurora resources
```

#### Harmony OS

```bash
# Cluster status
helixcode-harmony distributed status

# Device acceleration
helixcode-harmony device status

# Sync status
helixcode-harmony sync status
```

---

## Platform-Specific Features

### Aurora OS Features

#### Enhanced Security

```bash
# Check security level
helixcode-aurora security level

# Set security level
helixcode-aurora security level --set enhanced

# Security audit
helixcode-aurora security audit
```

#### System Monitoring

```bash
# Enable advanced monitoring
helixcode-aurora monitor enable --deep-inspection

# View system metrics
helixcode-aurora monitor metrics

# Generate report
helixcode-aurora monitor report --format pdf
```

### Harmony OS Features

#### Distributed Computing

```bash
# Start distributed engine
helixcode-harmony distributed start

# Add workers to cluster
helixcode-harmony distributed workers add \
  --host worker1.local \
  --host worker2.local

# Check cluster health
helixcode-harmony distributed health
```

#### Cross-Device Sync

```bash
# Enable sync
helixcode-harmony sync enable

# Configure sync interval
helixcode-harmony sync interval --set 30

# Force sync
helixcode-harmony sync now --force
```

#### AI Acceleration

```bash
# Check NPU availability
helixcode-harmony device npu-status

# Enable NPU acceleration
helixcode-harmony device npu enable

# Run accelerated inference
helixcode-harmony infer \
  --model llama-3-8b \
  --device npu \
  --batch-size 4
```

---

## Troubleshooting

### Aurora OS Issues

#### Service Won't Start

```bash
# Check logs
sudo journalctl -u helixcode-aurora -n 50

# Validate configuration
helixcode-aurora --config /etc/helixcode/aurora-config.yaml --validate

# Check database connection
psql -h localhost -U helixcode -d helixcode
```

#### Security Features Not Working

```bash
# Verify Aurora OS
cat /etc/aurora-release

# Check security module
helixcode-aurora security diagnose

# Review security logs
tail -f /var/log/helixcode/security.log
```

### Harmony OS Issues

#### Distributed Computing Not Working

```bash
# Check network
helixcode-harmony network test

# Verify firewall
sudo firewall-cmd --list-ports

# Check worker connectivity
helixcode-harmony distributed ping <worker-ip>
```

#### NPU Not Detected

```bash
# Check NPU device
ls -la /dev/davinci*

# Verify Harmony OS
cat /etc/harmonyos-release

# Update drivers (if needed)
helixcode-harmony device npu-update
```

### Common Issues (Both Platforms)

#### Database Connection Failed

```bash
# Test database
psql -h localhost -U helixcode -d helixcode

# Reset password
sudo -u postgres psql -c "ALTER USER helixcode WITH PASSWORD 'new_password';"

# Update env file
sudo nano /etc/helixcode/aurora.env  # or harmony.env
```

#### Port Already in Use

```bash
# Check what's using the port
sudo lsof -i :8080

# Change port in config
sudo nano /etc/helixcode/aurora-config.yaml  # or harmony-config.yaml

# Restart service
sudo systemctl restart helixcode-aurora  # or helixcode-harmony
```

---

## Next Steps

### Aurora OS

1. **Review Security Configuration**: Read [Aurora OS Security Guide](AURORA_OS_SECURITY.md)
2. **Configure System Monitoring**: Set up advanced monitoring dashboards
3. **Integrate with Authentication**: Connect to your organization's auth system
4. **Set Up Backups**: Configure automated backup schedules

### Harmony OS

1. **Set Up Distributed Cluster**: Add more workers for distributed computing
2. **Enable AI Acceleration**: Configure NPU/GPU for optimal performance
3. **Configure Cross-Device Sync**: Set up Super Device integration
4. **Optimize Resources**: Fine-tune resource management policies

### Both Platforms

1. **Add Workers**: Expand your worker pool for better performance
2. **Configure Workflows**: Set up automated development workflows
3. **Enable Notifications**: Configure Slack, Discord, or email notifications
4. **Performance Tuning**: Optimize for your specific workload

---

## Additional Resources

### Documentation

- [Harmony OS Complete Guide](HARMONY_OS_GUIDE.md) - Comprehensive Harmony OS documentation
- [Aurora OS Complete Guide](AURORA_OS_GUIDE.md) - Comprehensive Aurora OS documentation
- [API Reference](API_REFERENCE.md) - REST API documentation
- [Configuration Reference](CONFIG_REFERENCE.md) - All configuration options

### Deployment Scripts

- `scripts/deploy-aurora-os.sh` - Aurora OS automated deployment
- `scripts/deploy-harmony-os.sh` - Harmony OS automated deployment
- `scripts/deploy-specialized-platforms.sh` - Combined deployment script

### Support

- **Documentation**: https://docs.helixcode.dev
- **GitHub Issues**: https://github.com/helixcode/helixcode/issues
- **Community Forum**: https://forum.helixcode.dev

---

## Quick Reference Card

### Aurora OS

| Task | Command |
|------|---------|
| Start | `sudo systemctl start helixcode-aurora` |
| Stop | `sudo systemctl stop helixcode-aurora` |
| Status | `sudo systemctl status helixcode-aurora` |
| Logs | `sudo journalctl -u helixcode-aurora -f` |
| Config | `/etc/helixcode/aurora-config.yaml` |
| Binary | `/opt/helixcode/aurora-os` |
| Port | `8080` (default) |

### Harmony OS

| Task | Command |
|------|---------|
| Start | `sudo systemctl start helixcode-harmony` |
| Stop | `sudo systemctl stop helixcode-harmony` |
| Status | `sudo systemctl status helixcode-harmony` |
| Logs | `sudo journalctl -u helixcode-harmony -f` |
| Config | `/etc/helixcode/harmony-config.yaml` |
| Binary | `/opt/helixcode/harmony-os` |
| Port | `8080` (default) |

---

**Last Updated**: 2025-11-07
**Version**: 1.0.0
**Platforms**: Aurora OS, Harmony OS
