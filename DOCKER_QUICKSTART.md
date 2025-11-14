# Docker Quick Start Guide

**Date:** 2025-11-14

---

## Prerequisites Check

```bash
# Check Docker is installed
docker --version
# Expected: Docker version 28.5.1 or later

# Check Docker Compose is installed
docker compose version
# Expected: Docker Compose version v2.40.2 or later
```

---

## 1. Start Docker Daemon

The Docker daemon must be running before you can use Docker Compose.

```bash
# Start Docker daemon (requires sudo)
sudo systemctl start docker

# Verify Docker is running
sudo systemctl status docker

# Enable Docker to start on boot (optional)
sudo systemctl enable docker
```

**Alternative**: If you don't want to use sudo every time:

```bash
# Add your user to the docker group
sudo usermod -aG docker $USER

# Log out and log back in for the change to take effect
# Or run: newgrp docker
```

---

## 2. Setup Environment

```bash
# Copy environment template (already done if .env exists)
cp .env.example .env

# Optional: Edit .env for custom configuration
nano .env
```

---

## 3. Start Services

### Option A: Basic Stack (API + Database + Redis)

```bash
# Start services
docker compose up -d

# View logs
docker compose logs -f seca-api
```

### Option B: Full Stack with Monitoring

```bash
# Start with monitoring tools
docker compose --profile monitoring --profile tools up -d

# View all services
docker compose ps
```

---

## 4. Verify Services

```bash
# Check service health
docker compose ps

# Test API endpoints
curl http://localhost:8080/api/v1/health
curl http://localhost:8080/api/v1/ready

# Expected response:
# {"status":"ok"}
# {"status":"ready"}
```

---

## 5. Access Services

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

## 6. Common Operations

### View Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f seca-api

# Last 100 lines
docker compose logs --tail=100 seca-api
```

### Restart Services

```bash
# Restart all
docker compose restart

# Restart specific service
docker compose restart seca-api
```

### Stop Services

```bash
# Stop all services
docker compose down

# Stop and remove volumes (WARNING: deletes data!)
docker compose down -v
```

### Execute Commands in Containers

```bash
# Shell into API container
docker compose exec seca-api sh

# Run SECA commands
docker compose exec seca-api ./seca engagement list
docker compose exec seca-api ./seca check http --help

# PostgreSQL shell
docker compose exec postgres psql -U seca

# Redis CLI
docker compose exec redis redis-cli -a seca_redis_password
```

---

## 7. Rebuild After Code Changes

```bash
# Rebuild API image and restart
docker compose up -d --build seca-api

# Or rebuild everything
docker compose down
docker compose up -d --build
```

---

## 8. Development Mode (Hot Reload)

```bash
# Use development override
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d

# View logs
docker compose logs -f seca-api
```

**Features:**
- Source code mounted for live changes
- Debug logging enabled
- No password authentication for DB/Redis
- Faster rebuild times

---

## Troubleshooting

### Docker Daemon Not Running

**Error:** `Cannot connect to the Docker daemon at unix:///var/run/docker.sock`

**Fix:**
```bash
sudo systemctl start docker
```

### Permission Denied

**Error:** `permission denied while trying to connect to the Docker daemon socket`

**Fix:**
```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Log out and log back in
# Or run: newgrp docker
```

### Port Already in Use

**Error:** `Bind for 0.0.0.0:8080 failed: port is already allocated`

**Fix:**
```bash
# Option 1: Stop conflicting service
lsof -ti:8080 | xargs kill

# Option 2: Change port in .env
echo "API_PORT=8081" >> .env
docker compose up -d
```

### Service Won't Start

```bash
# Check logs for errors
docker compose logs seca-api

# Check health status
docker compose ps

# Restart service
docker compose restart seca-api
```

### Database Connection Error

```bash
# Verify PostgreSQL is running
docker compose ps postgres

# Check PostgreSQL logs
docker compose logs postgres

# Test connection
docker compose exec seca-api sh -c 'ping postgres'
```

---

## Complete Setup Example

```bash
# 1. Start Docker daemon
sudo systemctl start docker

# 2. Create .env file (if not exists)
cp .env.example .env

# 3. Start basic stack
docker compose up -d

# 4. Wait a few seconds for services to initialize
sleep 10

# 5. Verify services are healthy
docker compose ps

# 6. Test API
curl http://localhost:8080/api/v1/health
curl http://localhost:8080/api/v1/ready

# 7. View logs
docker compose logs -f seca-api
```

**Expected Output:**

```
$ docker compose ps
NAME                IMAGE                        STATUS              PORTS
seca-api            seca-cli:latest              Up (healthy)        0.0.0.0:8080->8080/tcp
seca-postgres       postgres:16-alpine           Up (healthy)        0.0.0.0:5432->5432/tcp
seca-redis          redis:7-alpine               Up (healthy)        0.0.0.0:6379->6379/tcp

$ curl http://localhost:8080/api/v1/health
{"status":"ok"}

$ curl http://localhost:8080/api/v1/ready
{"status":"ready"}
```

---

## Next Steps

1. **Add Monitoring**: Start with `--profile monitoring` to enable Prometheus and Grafana
2. **Setup Database**: Connect to PostgreSQL and verify schema is initialized
3. **Run Tests**: Execute `docker compose exec seca-api go test ./...`
4. **Configure Production**: Update `.env` with production credentials

---

## Related Documentation

- [DOCKER_SETUP.md](DOCKER_SETUP.md) - Comprehensive Docker setup documentation
- [API_VERSIONING.md](API_VERSIONING.md) - API versioning guide
- [READINESS_PROBE.md](READINESS_PROBE.md) - Health check documentation
- [SCALING_RECOMMENDATIONS.md](SCALING_RECOMMENDATIONS.md) - Scaling best practices

---

**Created:** 2025-11-14
**Author:** Claude Code Assistant
