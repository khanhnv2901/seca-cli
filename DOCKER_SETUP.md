# Docker Setup for SECA-CLI

**Date:** 2025-11-14
**Status:** ✅ Production-Ready

---

## Overview

Complete Docker Compose setup for local development and production deployment of SECA-CLI with:
- **SECA API Server** - Main application
- **PostgreSQL** - Database (for future migration from JSON)
- **Redis** - Caching and job queues
- **Prometheus** - Metrics collection
- **Grafana** - Monitoring dashboards
- **pgAdmin** - PostgreSQL GUI
- **Redis Commander** - Redis GUI

---

## Quick Start

### 1. Prerequisites

```bash
# Install Docker and Docker Compose
docker --version  # Docker 20.10+
docker-compose --version  # Docker Compose 2.0+
```

### 2. Setup Environment

```bash
# Copy environment template
cp .env.example .env

# Edit .env with your settings (optional for local dev)
nano .env
```

### 3. Start Basic Stack

```bash
# Start SECA API + PostgreSQL + Redis
docker-compose up -d

# View logs
docker-compose logs -f seca-api
```

### 4. Verify Services

```bash
# Check service health
docker-compose ps

# Test API health endpoint
curl http://localhost:8080/api/v1/health
curl http://localhost:8080/api/v1/ready
```

---

## Service Access

| Service | URL | Credentials |
|---------|-----|-------------|
| **SECA API** | http://localhost:8080 | - |
| **PostgreSQL** | localhost:5432 | seca / seca_dev_password |
| **Redis** | localhost:6379 | Password: seca_redis_password |
| **Prometheus** | http://localhost:9090 | - |
| **Grafana** | http://localhost:3000 | admin / admin |
| **pgAdmin** | http://localhost:5050 | admin@seca.local / admin |
| **Redis Commander** | http://localhost:8081 | - |

---

## Docker Compose Profiles

### Default Profile (Minimal)

```bash
# Start only API + Database + Redis
docker-compose up -d
```

**Services:** seca-api, postgres, redis

---

### Monitoring Profile

```bash
# Start with monitoring stack
docker-compose --profile monitoring up -d
```

**Services:** + prometheus, grafana

---

### Tools Profile

```bash
# Start with database GUIs
docker-compose --profile tools up -d
```

**Services:** + pgadmin, redis-commander

---

### Full Profile

```bash
# Start everything including background workers
docker-compose --profile full --profile monitoring --profile tools up -d
```

**Services:** All services including seca-worker

---

## Development Mode

For hot-reload during development:

```bash
# Use development override
docker-compose -f docker-compose.yml -f docker-compose.dev.yml up -d

# View logs
docker-compose logs -f seca-api
```

**Features:**
- Source code mounted for live changes
- Debug logging enabled
- No password authentication for DB/Redis
- Faster rebuild times

---

## Common Commands

### Starting Services

```bash
# Start in background
docker-compose up -d

# Start specific service
docker-compose up -d seca-api

# Start with monitoring
docker-compose --profile monitoring up -d

# Rebuild and start
docker-compose up -d --build
```

### Stopping Services

```bash
# Stop all services
docker-compose down

# Stop and remove volumes (WARNING: deletes data!)
docker-compose down -v

# Stop specific service
docker-compose stop seca-api
```

### Viewing Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f seca-api

# Last 100 lines
docker-compose logs --tail=100 seca-api
```

### Executing Commands

```bash
# Shell into API container
docker-compose exec seca-api sh

# Run SECA commands
docker-compose exec seca-api ./seca engagement list
docker-compose exec seca-api ./seca check http --help

# PostgreSQL shell
docker-compose exec postgres psql -U seca

# Redis CLI
docker-compose exec redis redis-cli -a seca_redis_password
```

### Database Operations

```bash
# Access PostgreSQL
docker-compose exec postgres psql -U seca -d seca

# Backup database
docker-compose exec postgres pg_dump -U seca seca > backup.sql

# Restore database
cat backup.sql | docker-compose exec -T postgres psql -U seca -d seca

# View tables
docker-compose exec postgres psql -U seca -d seca -c "\dt"
```

---

## File Structure

```
seca-cli/
├── Dockerfile                          # Multi-stage build
├── .dockerignore                       # Build exclusions
├── docker-compose.yml                  # Main compose file
├── docker-compose.dev.yml              # Development overrides
├── .env.example                        # Environment template
├── scripts/
│   └── init-db.sql                     # Database initialization
└── monitoring/
    ├── prometheus.yml                  # Prometheus config
    └── grafana/
        ├── dashboards/
        │   └── dashboard.yml           # Dashboard provisioning
        └── datasources/
            └── prometheus.yml          # Datasource config
```

---

## Environment Variables

### Application

| Variable | Default | Description |
|----------|---------|-------------|
| `SECA_ENV` | development | Environment (development/staging/production) |
| `VERSION` | dev | Application version |
| `LOG_LEVEL` | info | Log level (debug/info/warn/error) |

### Ports

| Variable | Default | Description |
|----------|---------|-------------|
| `API_PORT` | 8080 | SECA API port |
| `DB_PORT` | 5432 | PostgreSQL port |
| `REDIS_PORT` | 6379 | Redis port |
| `PROMETHEUS_PORT` | 9090 | Prometheus port |
| `GRAFANA_PORT` | 3000 | Grafana port |
| `PGADMIN_PORT` | 5050 | pgAdmin port |

### Database

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_PASSWORD` | seca_dev_password | PostgreSQL password |
| `DB_USER` | seca | PostgreSQL user |
| `DB_NAME` | seca | PostgreSQL database |

### Redis

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_PASSWORD` | seca_redis_password | Redis password |

---

## Volumes

All data is persisted in named Docker volumes:

| Volume | Purpose | Data |
|--------|---------|------|
| `postgres_data` | Database | PostgreSQL data |
| `redis_data` | Cache | Redis AOF file |
| `seca_data` | Application | Results, engagements, audit logs |
| `prometheus_data` | Monitoring | Prometheus metrics |
| `grafana_data` | Dashboards | Grafana configuration |
| `pgadmin_data` | Tools | pgAdmin settings |

### Backup Volumes

```bash
# List volumes
docker volume ls | grep seca

# Backup a volume
docker run --rm -v seca-cli_postgres_data:/data -v $(pwd):/backup \
  alpine tar czf /backup/postgres_backup.tar.gz /data

# Restore a volume
docker run --rm -v seca-cli_postgres_data:/data -v $(pwd):/backup \
  alpine tar xzf /backup/postgres_backup.tar.gz -C /
```

---

## Networking

All services run on the `seca-network` bridge network:

```bash
# Inspect network
docker network inspect seca-cli_seca-network

# Services can communicate using service names
# Example: seca-api can reach postgres at postgres:5432
```

---

## Health Checks

All services have health checks configured:

```bash
# View health status
docker-compose ps

# Services show as "healthy" when ready
```

| Service | Endpoint/Command | Interval |
|---------|------------------|----------|
| seca-api | `curl http://localhost:8080/api/v1/health` | 30s |
| postgres | `pg_isready -U seca` | 10s |
| redis | `redis-cli incr ping` | 10s |

---

## Production Deployment

### 1. Update Environment

```bash
# Create production .env
cp .env.example .env.production

# Edit for production
nano .env.production
```

```env
SECA_ENV=production
DB_PASSWORD=strong_random_password_here
REDIS_PASSWORD=strong_random_password_here
GRAFANA_PASSWORD=strong_random_password_here
API_AUTH_TOKEN=your_api_token_here
LOG_LEVEL=warn
```

### 2. Use Production Compose File

```bash
# Start with production settings
docker-compose --env-file .env.production up -d
```

### 3. Enable HTTPS (Recommended)

Add Nginx or Traefik as reverse proxy:

```yaml
# docker-compose.prod.yml
services:
  nginx:
    image: nginx:alpine
    ports:
      - "443:443"
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./certs:/etc/nginx/certs:ro
    depends_on:
      - seca-api
```

### 4. Setup Monitoring Alerts

```yaml
# Add Alertmanager
alertmanager:
  image: prom/alertmanager:latest
  ports:
    - "9093:9093"
  volumes:
    - ./alertmanager.yml:/etc/alertmanager/alertmanager.yml
```

---

## Troubleshooting

### Service Won't Start

```bash
# Check logs
docker-compose logs seca-api

# Check health
docker-compose ps

# Restart service
docker-compose restart seca-api
```

### Database Connection Error

```bash
# Verify PostgreSQL is healthy
docker-compose ps postgres

# Check PostgreSQL logs
docker-compose logs postgres

# Test connection
docker-compose exec seca-api sh -c 'psql -h postgres -U seca -c "SELECT 1"'
```

### Port Already in Use

```bash
# Change port in .env
API_PORT=8081

# Or stop conflicting service
lsof -ti:8080 | xargs kill
```

### Out of Disk Space

```bash
# Check Docker disk usage
docker system df

# Clean up unused images
docker image prune -a

# Clean up unused volumes
docker volume prune

# Full cleanup (WARNING: removes all stopped containers)
docker system prune -a --volumes
```

### Permission Errors

```bash
# Fix volume permissions
docker-compose exec seca-api chown -R seca:seca /app/data

# Or recreate with correct user
docker-compose down -v
docker-compose up -d
```

---

## CI/CD Integration

### GitHub Actions Example

```yaml
# .github/workflows/docker.yml
name: Docker Build

on:
  push:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Build Docker image
        run: |
          docker build -t seca-cli:${{ github.sha }} .

      - name: Run tests in Docker
        run: |
          docker-compose -f docker-compose.yml -f docker-compose.test.yml run seca-api go test ./...

      - name: Push to registry
        run: |
          echo ${{ secrets.DOCKER_PASSWORD }} | docker login -u ${{ secrets.DOCKER_USERNAME }} --password-stdin
          docker push seca-cli:${{ github.sha }}
```

---

## Database Migrations

When PostgreSQL schema changes:

```bash
# Create migration file
cat > scripts/migrations/001_add_users.sql <<EOF
ALTER TABLE engagements ADD COLUMN user_id UUID;
EOF

# Run migration
docker-compose exec postgres psql -U seca -d seca -f /docker-entrypoint-initdb.d/migrations/001_add_users.sql
```

---

## Performance Tuning

### PostgreSQL

```yaml
# docker-compose.yml
postgres:
  command: |
    postgres
    -c shared_buffers=256MB
    -c max_connections=200
    -c work_mem=4MB
```

### Redis

```yaml
redis:
  command: |
    redis-server
    --maxmemory 512mb
    --maxmemory-policy allkeys-lru
```

---

## Security Best Practices

1. **Change Default Passwords**
   ```bash
   # Never use default passwords in production!
   DB_PASSWORD=$(openssl rand -base64 32)
   REDIS_PASSWORD=$(openssl rand -base64 32)
   ```

2. **Use Secrets Management**
   ```bash
   # Docker secrets (Swarm mode)
   docker secret create db_password db_password.txt
   ```

3. **Limit Network Exposure**
   ```yaml
   # Only expose necessary ports
   seca-api:
     ports:
       - "127.0.0.1:8080:8080"  # Localhost only
   ```

4. **Run as Non-Root**
   ```dockerfile
   # Already configured in Dockerfile
   USER seca
   ```

5. **Keep Images Updated**
   ```bash
   # Regularly update base images
   docker-compose pull
   docker-compose up -d
   ```

---

## Monitoring Metrics

### Prometheus Queries

```promql
# API request rate
rate(http_requests_total[5m])

# Error rate
rate(http_requests_total{status=~"5.."}[5m])

# Response time P95
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
```

### Grafana Dashboards

1. Import pre-built dashboards
2. Create custom panels
3. Setup alerts

---

## Related Documentation

- [Main README](README.md)
- [API Versioning](API_VERSIONING.md)
- [Readiness Probe](READINESS_PROBE.md)
- [Scaling Recommendations](SCALING_RECOMMENDATIONS.md)

---

## Support

For issues or questions:
- GitHub Issues: https://github.com/khanhnv2901/seca-cli/issues
- Documentation: ./docs/

---

**Created:** 2025-11-14
**Author:** DevOps Team
