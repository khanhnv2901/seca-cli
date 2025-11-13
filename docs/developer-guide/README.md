# Developer Guide

Documentation for developers working with SECA-CLI's extensibility features and API.

---

## Available Guides

### 🔌 [Plugin Development Guide](./plugin-development.md)
Learn how to create custom security checkers and extend SECA-CLI's functionality.

**For:** Backend developers, security tool authors
**Topics:** Plugin architecture, checker interface, custom checks, packaging

---

### 🌐 [API Service Guide](./api-guide.md)
Run SECA-CLI as a REST API service to power web applications and automation.

**For:** Backend developers, DevOps engineers
**Topics:** Server setup, endpoints, authentication, security, job management

**Key Features:**
- Graceful shutdown with signal handling
- Per-IP rate limiting (token bucket algorithm)
- Structured logging with zap
- CORS support for frontend integration
- Job retention policy (max 1000 jobs)
- Command timeouts (90s per job)
- Buffer limits (1MB stdout/stderr)

---

### 🎨 [Frontend Integration Guide](./frontend-integration-guide.md)
Complete guide for frontend developers to integrate with the SECA REST API.

**For:** Frontend developers, UI/UX engineers
**Topics:** API client setup, authentication, workflows, real-time updates, error handling

**Includes:**
- Quick start examples
- Complete API reference with request/response samples
- Common workflow implementations
- React, Vue.js, and TypeScript examples
- Server-Sent Events (SSE) for real-time updates
- Working HTML demo ([example-frontend.html](./example-frontend.html))

---

## Quick Reference

### Running the API Server

```bash
# Development
seca serve \
  --addr 127.0.0.1:8080 \
  --auth-token $(openssl rand -hex 16) \
  --cors-origins "http://localhost:3000" \
  --rate-limit 100

# Production
seca serve \
  --addr 0.0.0.0:8080 \
  --auth-token $SECA_API_TOKEN \
  --cors-origins "https://yourapp.com" \
  --rate-limit 10 \
  --rate-burst 20 \
  --shutdown-timeout 30s
```

### API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/health` | GET | Health check |
| `/api/engagements` | GET, POST | List/create engagements |
| `/api/engagements/{id}` | GET | Get single engagement |
| `/api/jobs` | GET, POST | List/create scan jobs |
| `/api/jobs/{id}` | GET | Get job status |
| `/api/jobs-stream` | GET | SSE stream for live updates |
| `/api/results/{id}` | GET | Get scan results |
| `/api/telemetry/{id}` | GET | Get audit logs |

### Authentication

All requests require authentication header:
```http
X-Auth-Token: your-secret-token
```

---

## Examples

### Try the Demo

1. **Start the server:**
   ```bash
   seca serve --addr 127.0.0.1:8080 --auth-token demo-token
   ```

2. **Open the demo UI:**
   Open [example-frontend.html](./example-frontend.html) in your browser

3. **Configure the demo:**
   Edit the JavaScript constants:
   ```javascript
   const API_BASE_URL = 'http://127.0.0.1:8080/api';
   const API_TOKEN = 'demo-token';
   ```

4. **Run a scan:**
   Enter a domain and click "Run Security Scan"

### Quick Test with curl

```bash
# Health check
curl -H "X-Auth-Token: demo-token" http://127.0.0.1:8080/api/health

# Create engagement
curl -X POST \
  -H "X-Auth-Token: demo-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Scan",
    "owner": "test@example.com",
    "roe": "Demo ROE",
    "roe_agree": true,
    "scope": ["https://example.com"]
  }' \
  http://127.0.0.1:8080/api/engagements

# Start scan job
curl -X POST \
  -H "X-Auth-Token: demo-token" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "http",
    "engagement_id": "YOUR_ENGAGEMENT_ID"
  }' \
  http://127.0.0.1:8080/api/jobs
```

---

## Architecture Overview

```
┌─────────────────┐
│  Frontend UI    │
│  (React/Vue)    │
└────────┬────────┘
         │ HTTP/REST
         │ + SSE
┌────────▼────────┐
│   API Server    │
│  (cmd/serve.go) │
│                 │
│  Middleware:    │
│  • CORS         │
│  • Rate Limit   │
│  • Logging      │
│  • Auth         │
└────────┬────────┘
         │
┌────────▼────────┐
│  Job Manager    │
│ (jobs.go)       │
│                 │
│  • Job Queue    │
│  • SSE Pub/Sub  │
│  • Retention    │
└────────┬────────┘
         │
┌────────▼────────┐
│  CLI Runner     │
│ (cliCheckRunner)│
│                 │
│  • Exec Checks  │
│  • Buffer Limit │
│  • Timeout      │
└─────────────────┘
```

---

## Security Considerations

### For API Developers

✅ **Implemented:**
- Request body size limits (1MB)
- Timing-safe auth token comparison
- Cryptographically secure job IDs
- Error sanitization (no info disclosure)
- Per-IP rate limiting
- Command execution timeouts (90s)
- Memory limits (stdout/stderr 1MB each)
- Job retention policy (1000 max)

⚠️ **Best Practices:**
- Always use HTTPS in production
- Rotate auth tokens regularly
- Run server on private network
- Use reverse proxy (nginx/Caddy) for TLS
- Monitor rate limit metrics
- Set up alerts for 5xx errors

### For Frontend Developers

✅ **Security Tips:**
- Never expose auth tokens in client code
- Validate all user input (domain, email, etc.)
- Implement CSRF protection
- Use Content Security Policy (CSP)
- Rate limit on frontend too (UX)
- Show clear error messages to users

---

## Performance Considerations

### API Server

- **Concurrent Requests:** Handles multiple requests simultaneously
- **Memory Usage:** ~50MB base + ~1MB per active job
- **Job Throughput:** Limited by command execution time (~90s max)
- **Cleanup:** Automatic cleanup every 5 minutes

### Scaling Options

1. **Horizontal:** Run multiple API servers behind load balancer
2. **Vertical:** Increase system resources for more concurrent jobs
3. **Storage:** Replace in-memory JobManager with Redis/PostgreSQL
4. **Queue:** Use RabbitMQ/SQS for job distribution

---

## Testing

### Unit Tests

```bash
go test ./cmd/...
go test ./internal/api/...
```

### Integration Tests

```bash
# Run test scripts
bash /tmp/test_resource_management.sh
bash /tmp/test_rate_limit.sh
bash /tmp/test_error_responses.sh
```

### Manual Testing

See the [Frontend Integration Guide](./frontend-integration-guide.md#testing-tips) for curl commands and browser testing.

---

## Troubleshooting

### Common Issues

**Q: CORS errors in browser**
A: Add your origin to `--cors-origins` flag

**Q: 401 Unauthorized**
A: Check `X-Auth-Token` header is set correctly

**Q: 429 Rate Limit Exceeded**
A: Increase `--rate-limit` and `--rate-burst` values

**Q: Jobs timing out**
A: Jobs have 90s timeout - increase system resources

**Q: SSE not working**
A: Some browsers limit EventSource - use fetch with ReadableStream

---

## Contributing

When adding new API features:

1. Update [api-guide.md](./api-guide.md) with backend details
2. Update [frontend-integration-guide.md](./frontend-integration-guide.md) with examples
3. Add tests for new endpoints
4. Update this README with relevant info
5. Follow security best practices (see above)

---

## Additional Resources

- [Main Documentation](../README.md)
- [Testing Guide](../technical/testing.md)
- [Deployment Guide](../technical/deployment.md)
- [Troubleshooting Guide](../reference/troubleshooting.md)

---

**Last Updated:** January 2025
