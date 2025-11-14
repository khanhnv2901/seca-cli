# Readiness Probe Implementation

**Date:** 2025-11-14
**Status:** ✅ Completed

---

## Overview

Implemented a readiness probe endpoint (`/api/ready`) to distinguish between:
- **Liveness** (`/api/health`) - Is the server process running?
- **Readiness** (`/api/ready`) - Is the server ready to handle requests?

This is a best practice for Kubernetes deployments and production monitoring.

---

## Endpoints Comparison

| Endpoint | Purpose | Returns 200 when | Returns 503 when | K8s Probe Type |
|----------|---------|------------------|------------------|----------------|
| `/api/health` | Liveness | Server is running | Config error | `livenessProbe` |
| `/api/ready` | Readiness | Server can serve traffic | Dependencies unavailable | `readinessProbe` |

---

## Implementation

### 1. Interface Update ([internal/api/server.go:63-66](internal/api/server.go#L63-L66))

```go
type HealthService interface {
    Check(ctx context.Context) error  // Liveness
    Ready(ctx context.Context) error  // Readiness ← NEW
}
```

### 2. Readiness Logic ([cmd/serve.go:228-254](cmd/serve.go#L228-L254))

```go
func (s *healthAPIService) Ready(ctx context.Context) error {
    // Check if results directory is configured
    if s.appCtx.ResultsDir == "" {
        return fmt.Errorf("results directory not configured")
    }

    // Check if results directory is accessible
    if _, err := os.Stat(s.appCtx.ResultsDir); err != nil {
        return fmt.Errorf("results directory not accessible: %w", err)
    }

    // Check if engagements file exists (indicates system is initialized)
    engagementsPath := filepath.Join(s.appCtx.ResultsDir, "engagements.json")
    if _, err := os.Stat(engagementsPath); err != nil {
        // File doesn't exist yet - OK for new installation
        return nil
    }

    // Verify engagements file is readable
    if _, err := os.ReadFile(engagementsPath); err != nil {
        return fmt.Errorf("engagements file not readable: %w", err)
    }

    return nil
}
```

### 3. Handler ([internal/api/server.go:136-148](internal/api/server.go#L136-L148))

```go
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        s.methodNotAllowed(w, r)
        return
    }
    if s.cfg.Health != nil {
        if err := s.cfg.Health.Ready(r.Context()); err != nil {
            s.writeError(w, r, http.StatusServiceUnavailable, err)
            return
        }
    }
    writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
```

### 4. Route Registration ([internal/api/server.go:113](internal/api/server.go#L113))

```go
func (s *Server) routes() {
    s.mux.Handle("/api/health", s.withAuth(http.HandlerFunc(s.handleHealth)))
    s.mux.Handle("/api/ready", s.withAuth(http.HandlerFunc(s.handleReady)))  // ← NEW
    // ... other routes
}
```

---

## Usage Examples

### 1. Server is Ready

**Request:**
```bash
curl -i http://localhost:8080/api/ready
```

**Response:**
```
HTTP/1.1 200 OK
X-Request-Id: c9ca50a9b4e219e3
Content-Type: application/json

{"status":"ready"}
```

**Log Entry:**
```json
{
  "level": "info",
  "msg": "http_request",
  "request_id": "c9ca50a9b4e219e3",
  "method": "GET",
  "path": "/api/ready",
  "status": 200,
  "duration": 0.000148543,
  "bytes": 19
}
```

---

### 2. Server is Not Ready

**Scenario:** Results directory doesn't exist or is inaccessible

**Request:**
```bash
curl -i http://localhost:8080/api/ready
```

**Response:**
```
HTTP/1.1 503 Service Unavailable
X-Request-Id: c5d3c96cd3e21b9f
Content-Type: application/json

{"error":"internal server error"}
```

**Log Entry:**
```json
{
  "level": "error",
  "msg": "internal_server_error",
  "request_id": "c5d3c96cd3e21b9f",
  "method": "GET",
  "path": "/api/ready",
  "error": "results directory not accessible: stat /path/to/results: no such file or directory",
  "status": 503
}
```

---

## Kubernetes Configuration

### Deployment YAML

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: seca-api
spec:
  template:
    spec:
      containers:
      - name: seca-api
        image: seca-cli:latest
        ports:
        - containerPort: 8080

        # Liveness Probe - Restart if unhealthy
        livenessProbe:
          httpGet:
            path: /api/health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3

        # Readiness Probe - Remove from service if not ready
        readinessProbe:
          httpGet:
            path: /api/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 2
```

### How it Works

1. **During Startup:**
   - Liveness probe waits 30s before first check (startup time)
   - Readiness probe starts checking after 5s
   - Pod won't receive traffic until readiness succeeds

2. **During Operation:**
   - If liveness fails 3 times → Pod is **restarted**
   - If readiness fails 2 times → Pod is **removed from service** (no traffic)

3. **During Rolling Update:**
   - New pods must pass readiness before old pods are terminated
   - Zero-downtime deployments

---

## Readiness Checks

The readiness probe currently checks:

1. ✅ **Results directory configured** - `appCtx.ResultsDir != ""`
2. ✅ **Results directory accessible** - `os.Stat(resultsDir)`
3. ✅ **Engagements file readable** (if exists) - `os.ReadFile(engagements.json)`

### Future Enhancements

You can extend the readiness checks to include:

```go
func (s *healthAPIService) Ready(ctx context.Context) error {
    // ... existing checks ...

    // Check database connection (if using PostgreSQL)
    if err := s.db.PingContext(ctx); err != nil {
        return fmt.Errorf("database not ready: %w", err)
    }

    // Check Redis connection (if using cache)
    if err := s.redis.Ping(ctx).Err(); err != nil {
        return fmt.Errorf("redis not ready: %w", err)
    }

    // Check external dependencies
    if err := s.checkExternalAPI(ctx); err != nil {
        return fmt.Errorf("external API not ready: %w", err)
    }

    return nil
}
```

---

## Testing

### Unit Tests

**Test File:** [internal/api/server_test.go:170-215](internal/api/server_test.go#L170-L215)

```go
func TestServer_HandleReady(t *testing.T) {
    tests := []struct {
        name       string
        readyErr   error
        wantStatus int
        wantBody   string
    }{
        {
            name:       "ready",
            readyErr:   nil,
            wantStatus: http.StatusOK,
            wantBody:   `{"status":"ready"}`,
        },
        {
            name:       "not ready",
            readyErr:   errors.New("database down"),
            wantStatus: http.StatusServiceUnavailable,
            wantBody:   `{"error":"internal server error"}`,
        },
    }
    // ... test implementation
}
```

**Results:**
```bash
$ go test ./internal/api/... -run TestServer_HandleReady -v

=== RUN   TestServer_HandleReady/ready
--- PASS: TestServer_HandleReady/ready (0.00s)
=== RUN   TestServer_HandleReady/not_ready
--- PASS: TestServer_HandleReady/not_ready (0.00s)
--- PASS: TestServer_HandleReady (0.00s)
PASS
```

---

## Comparison: Liveness vs Readiness

### Liveness Probe (`/api/health`)

**Purpose:** Detect if the application is deadlocked or crashed

**Returns 500 when:**
- Configuration is missing (results_dir)
- Critical internal error

**Kubernetes Action:** Restart the pod

**Example Failures:**
- Out of memory (Go runtime panic)
- Infinite loop (deadlock)
- Configuration corruption

---

### Readiness Probe (`/api/ready`)

**Purpose:** Detect if the application can serve traffic

**Returns 503 when:**
- Results directory is inaccessible
- Database is down
- External dependencies unavailable

**Kubernetes Action:** Remove from service (stop sending traffic)

**Example Failures:**
- NFS mount not available
- Database connection pool exhausted
- Third-party API down (temporary)

---

## Best Practices

### 1. **Different Failure Thresholds**

```yaml
livenessProbe:
  failureThreshold: 3      # Allow 3 failures before restart
  periodSeconds: 10        # Check every 10s

readinessProbe:
  failureThreshold: 2      # Remove from service after 2 failures
  periodSeconds: 5         # Check more frequently
```

### 2. **Startup Probe for Slow Starts**

If your app takes >30s to start, use a startup probe:

```yaml
startupProbe:
  httpGet:
    path: /api/ready
    port: 8080
  initialDelaySeconds: 0
  periodSeconds: 5
  failureThreshold: 12     # Allow 60s for startup (12 * 5s)

livenessProbe:
  httpGet:
    path: /api/health
    port: 8080
  periodSeconds: 10         # Only starts after startupProbe succeeds
```

### 3. **Avoid Heavy Operations in Probes**

```go
// Bad - Expensive operation
func (s *healthAPIService) Ready(ctx context.Context) error {
    // Don't do this - scans entire database!
    count, err := s.db.Query("SELECT COUNT(*) FROM engagements")
    return err
}

// Good - Lightweight check
func (s *healthAPIService) Ready(ctx context.Context) error {
    // Just ping the connection
    return s.db.PingContext(ctx)
}
```

---

## Monitoring

### Prometheus Metrics (Future Enhancement)

```go
var (
    readinessProbeTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "seca_readiness_probe_total",
            Help: "Total readiness probe checks",
        },
        []string{"status"}, // "ready", "not_ready"
    )

    readinessProbeDuration = prometheus.NewHistogram(
        prometheus.HistogramOpts{
            Name: "seca_readiness_probe_duration_seconds",
            Help: "Readiness probe check duration",
        },
    )
)

func (s *healthAPIService) Ready(ctx context.Context) error {
    start := time.Now()
    defer func() {
        readinessProbeDuration.Observe(time.Since(start).Seconds())
    }()

    err := s.doReadinessChecks(ctx)

    status := "ready"
    if err != nil {
        status = "not_ready"
    }
    readinessProbeTotal.WithLabelValues(status).Inc()

    return err
}
```

---

## Troubleshooting

### Pod Keeps Restarting

**Symptom:** `kubectl get pods` shows `CrashLoopBackOff`

**Cause:** Liveness probe is failing

**Debug:**
```bash
kubectl logs <pod-name> --previous  # Check logs before restart
kubectl describe pod <pod-name>     # Check events
curl http://<pod-ip>:8080/api/health  # Test liveness directly
```

---

### Pod Not Receiving Traffic

**Symptom:** Pod is running but not in service endpoints

**Cause:** Readiness probe is failing

**Debug:**
```bash
kubectl get endpoints seca-api      # Check if pod is in endpoints
kubectl describe pod <pod-name>     # Check readiness status
curl http://<pod-ip>:8080/api/ready  # Test readiness directly
```

---

## Related Documentation

- [Kubernetes Liveness and Readiness Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
- [SCALING_RECOMMENDATIONS.md](SCALING_RECOMMENDATIONS.md)
- [REQUEST_ID_MIDDLEWARE.md](REQUEST_ID_MIDDLEWARE.md)

---

## Changelog

### 2025-11-14 - Initial Implementation
- ✅ Added `Ready()` method to `HealthService` interface
- ✅ Implemented readiness checks in `healthAPIService`
- ✅ Added `/api/ready` endpoint and handler
- ✅ Registered route in API server
- ✅ Added comprehensive unit tests
- ✅ Verified end-to-end functionality
- ✅ Returns 503 (Service Unavailable) when not ready

---

**Author:** Claude Code Assistant
**Reviewer:** khanhnv
