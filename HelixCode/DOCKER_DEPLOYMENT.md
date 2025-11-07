# Docker Deployment Guide

This guide covers Docker-based deployment for HelixCode specialized platforms (Aurora OS and Harmony OS).

## Quick Start

### Aurora OS (Security-Focused)

```bash
# Create environment file
cp .env.example .env
# Edit .env with your settings

# Start Aurora OS
docker-compose -f docker-compose.aurora-os.yml up -d

# Check status
docker-compose -f docker-compose.aurora-os.yml ps

# View logs
docker-compose -f docker-compose.aurora-os.yml logs -f aurora-os
```

**Access**: http://localhost:8080

### Harmony OS (Distributed Computing)

```bash
# Create environment file
cp .env.example .env
# Edit .env with your settings

# Start Harmony OS
docker-compose -f docker-compose.harmony-os.yml up -d

# Check status
docker-compose -f docker-compose.harmony-os.yml ps

# View logs
docker-compose -f docker-compose.harmony-os.yml logs -f harmony-os
```

**Access**: http://localhost:8080

### Both Platforms (Combined Deployment)

```bash
# Create environment file
cp .env.example .env
# Edit .env with your settings

# Start both platforms
docker-compose -f docker-compose.specialized-platforms.yml up -d

# Check status
docker-compose -f docker-compose.specialized-platforms.yml ps
```

**Access**:
- Aurora OS: http://localhost:8080
- Harmony OS: http://localhost:8081

---

## Configuration

### Environment Variables

Create a `.env` file in the project root:

```bash
# Database Configuration
DATABASE_PASSWORD=your_secure_password_here
DATABASE_SSL_MODE=disable

# Redis Configuration
REDIS_PASSWORD=your_redis_password_here
REDIS_MAX_MEMORY=512mb

# Aurora OS Configuration
AURORA_SECURITY_LEVEL=enhanced  # standard, enhanced, maximum
AURORA_MONITORING=true
AURORA_AUDIT_LOGGING=true
AURORA_ENCRYPTION=true
AURORA_RATE_LIMITING=true
AURORA_MFA_REQUIRED=false
AURORA_AUDIT_RETENTION=365
AURORA_PROMETHEUS=true

# Harmony OS Configuration
HARMONY_DISTRIBUTED_ENABLED=true
HARMONY_SYNC_ENABLED=true
HARMONY_SYNC_INTERVAL=30
HARMONY_AI_ACCEL=true
HARMONY_GPU_ENABLED=true
HARMONY_NPU_ENABLED=false
HARMONY_MONITORING=true
HARMONY_MULTI_SCREEN=true
HARMONY_SUPER_DEVICE=true
HARMONY_SERVICE_DISCOVERY=true
HARMONY_SERVICE_FAILOVER=true
HARMONY_MAX_TASKS=20

# Authentication
AUTH_TOKEN_EXPIRY=86400
AUTH_SESSION_EXPIRY=604800

# Logging
LOG_LEVEL=info

# Monitoring (for combined deployment)
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=admin
```

### PostgreSQL Tuning

For production deployments, create `config/postgres.conf`:

```conf
# Memory Settings
shared_buffers = 256MB
effective_cache_size = 1GB
work_mem = 4MB
maintenance_work_mem = 64MB

# Connection Settings
max_connections = 200

# WAL Settings
wal_buffers = 16MB
checkpoint_completion_target = 0.9
max_wal_size = 1GB
min_wal_size = 80MB

# Query Tuning
random_page_cost = 1.1
effective_io_concurrency = 200
default_statistics_target = 100
```

### Nginx Load Balancer

For the combined deployment with load balancing, create `config/nginx.conf`:

```nginx
events {
    worker_connections 1024;
}

http {
    upstream aurora_backend {
        server aurora-os:8080;
    }

    upstream harmony_backend {
        server harmony-os:8081;
    }

    # Aurora OS (Security)
    server {
        listen 80;
        server_name aurora.helixcode.local;

        location / {
            proxy_pass http://aurora_backend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        location /health {
            proxy_pass http://aurora_backend/health;
            access_log off;
        }
    }

    # Harmony OS (Distributed)
    server {
        listen 80;
        server_name harmony.helixcode.local;

        location / {
            proxy_pass http://harmony_backend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        location /health {
            proxy_pass http://harmony_backend/health;
            access_log off;
        }
    }

    # Health check endpoint
    server {
        listen 80 default_server;

        location /health {
            return 200 "OK\n";
            add_header Content-Type text/plain;
        }
    }
}
```

### Prometheus Monitoring

Create `config/prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'aurora-os'
    static_configs:
      - targets: ['aurora-os:9090']
        labels:
          platform: 'aurora'
          environment: 'production'

  - job_name: 'harmony-os'
    static_configs:
      - targets: ['harmony-os:8081']
        labels:
          platform: 'harmony'
          environment: 'production'

  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres:5432']

  - job_name: 'redis'
    static_configs:
      - targets: ['redis:6379']
```

---

## Docker Profiles

### Aurora OS with Redis

```bash
# Start Aurora OS with optional Redis
docker-compose -f docker-compose.aurora-os.yml --profile with-redis up -d
```

### Harmony OS with Distributed Workers

```bash
# Start Harmony OS with 2 worker nodes
docker-compose -f docker-compose.harmony-os.yml --profile distributed up -d
```

This starts:
- Main Harmony OS instance (port 8080)
- Worker 1 (port 8083)
- Worker 2 (port 8084)

### Combined Platform with Monitoring

```bash
# Start both platforms with Prometheus and Grafana
docker-compose -f docker-compose.specialized-platforms.yml --profile monitoring up -d
```

Access Grafana at: http://localhost:3000

### Production Deployment

```bash
# Start with Nginx load balancer
docker-compose -f docker-compose.specialized-platforms.yml --profile production up -d
```

---

## Building Images

### Build Aurora OS Image

```bash
docker build -f Dockerfile.aurora-os -t helixcode/aurora-os:latest .
```

### Build Harmony OS Image

```bash
docker build -f Dockerfile.harmony-os -t helixcode/harmony-os:latest .
```

### Build Both Images

```bash
docker build -f Dockerfile.aurora-os -t helixcode/aurora-os:latest .
docker build -f Dockerfile.harmony-os -t helixcode/harmony-os:latest .
```

---

## Volume Management

### Backup Volumes

```bash
# Backup PostgreSQL data
docker run --rm \
  -v helixcode_postgres_data:/data \
  -v $(pwd)/backups:/backup \
  alpine tar czf /backup/postgres-$(date +%Y%m%d).tar.gz /data

# Backup Aurora OS data
docker run --rm \
  -v helixcode_aurora_data:/data \
  -v $(pwd)/backups:/backup \
  alpine tar czf /backup/aurora-$(date +%Y%m%d).tar.gz /data

# Backup Harmony OS data
docker run --rm \
  -v helixcode_harmony_data:/data \
  -v $(pwd)/backups:/backup \
  alpine tar czf /backup/harmony-$(date +%Y%m%d).tar.gz /data
```

### Restore Volumes

```bash
# Restore PostgreSQL data
docker run --rm \
  -v helixcode_postgres_data:/data \
  -v $(pwd)/backups:/backup \
  alpine sh -c "cd /data && tar xzf /backup/postgres-20250107.tar.gz --strip 1"
```

### Clean Volumes

```bash
# Remove all volumes (WARNING: destroys all data)
docker-compose -f docker-compose.specialized-platforms.yml down -v
```

---

## Monitoring and Logs

### View Logs

```bash
# All services
docker-compose -f docker-compose.specialized-platforms.yml logs -f

# Specific service
docker-compose -f docker-compose.specialized-platforms.yml logs -f aurora-os
docker-compose -f docker-compose.specialized-platforms.yml logs -f harmony-os

# Last 100 lines
docker-compose -f docker-compose.specialized-platforms.yml logs --tail=100 aurora-os
```

### Access Container Shell

```bash
# Aurora OS
docker exec -it helixcode-aurora-os sh

# Harmony OS
docker exec -it helixcode-harmony-os sh

# PostgreSQL
docker exec -it helixcode-postgres psql -U helixcode
```

### Health Checks

```bash
# Check Aurora OS health
curl http://localhost:8080/health

# Check Harmony OS health
curl http://localhost:8081/health

# Check all services
docker-compose -f docker-compose.specialized-platforms.yml ps
```

---

## Troubleshooting

### Database Connection Issues

```bash
# Check if PostgreSQL is ready
docker exec helixcode-postgres pg_isready -U helixcode

# View PostgreSQL logs
docker logs helixcode-postgres

# Connect to database
docker exec -it helixcode-postgres psql -U helixcode -d helixcode
```

### Redis Connection Issues

```bash
# Check Redis connectivity
docker exec helixcode-redis redis-cli ping

# Authenticate and check
docker exec helixcode-redis redis-cli -a your_password ping

# View Redis logs
docker logs helixcode-redis
```

### Container Won't Start

```bash
# Check container logs
docker logs helixcode-aurora-os
docker logs helixcode-harmony-os

# Check resource usage
docker stats

# Inspect container
docker inspect helixcode-aurora-os
```

### Performance Issues

```bash
# Check resource usage
docker stats

# Check container resource limits
docker inspect helixcode-aurora-os | grep -A 10 Resources

# View PostgreSQL connections
docker exec helixcode-postgres psql -U helixcode -c "SELECT count(*) FROM pg_stat_activity;"
```

### Network Issues

```bash
# Inspect network
docker network inspect helixcode_helix-network

# Test connectivity between containers
docker exec helixcode-aurora-os ping postgres
docker exec helixcode-harmony-os ping redis
```

---

## Production Recommendations

### Security Hardening

1. **Change Default Passwords**: Update all passwords in `.env`
2. **Enable TLS**: Configure SSL certificates for Nginx
3. **Firewall Rules**: Restrict access to database and Redis ports
4. **Regular Updates**: Keep Docker images and base images updated
5. **Secrets Management**: Use Docker secrets or external vaults

### High Availability

1. **PostgreSQL Replication**: Set up streaming replication
2. **Redis Sentinel**: Configure Redis Sentinel for failover
3. **Load Balancing**: Use Nginx or HAProxy
4. **Container Orchestration**: Consider Kubernetes for larger deployments
5. **Backup Strategy**: Automated daily backups with retention policy

### Monitoring

1. **Enable Prometheus**: Use `--profile monitoring`
2. **Configure Alerts**: Set up alerting rules in Prometheus
3. **Grafana Dashboards**: Import pre-built dashboards
4. **Log Aggregation**: Consider ELK stack or Loki
5. **APM**: Application performance monitoring

### Resource Limits

Add resource limits to `docker-compose.yml`:

```yaml
services:
  aurora-os:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 2G
        reservations:
          cpus: '1.0'
          memory: 512M
```

---

## Upgrade Guide

### Upgrade Process

```bash
# 1. Backup data
docker-compose -f docker-compose.specialized-platforms.yml exec postgres \
  pg_dump -U helixcode helixcode > backup-$(date +%Y%m%d).sql

# 2. Pull new images
docker-compose -f docker-compose.specialized-platforms.yml pull

# 3. Stop services
docker-compose -f docker-compose.specialized-platforms.yml down

# 4. Start with new images
docker-compose -f docker-compose.specialized-platforms.yml up -d

# 5. Verify health
curl http://localhost:8080/health
curl http://localhost:8081/health
```

### Rollback

```bash
# Stop current version
docker-compose -f docker-compose.specialized-platforms.yml down

# Use specific image version
# Edit docker-compose.yml to specify old image tags

# Start with old version
docker-compose -f docker-compose.specialized-platforms.yml up -d
```

---

## References

- [Aurora OS Guide](docs/AURORA_OS_GUIDE.md)
- [Harmony OS Guide](docs/HARMONY_OS_GUIDE.md)
- [Specialized Platforms Deployment](docs/SPECIALIZED_PLATFORMS_DEPLOYMENT.md)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [PostgreSQL Docker](https://hub.docker.com/_/postgres)
- [Redis Docker](https://hub.docker.com/_/redis)
