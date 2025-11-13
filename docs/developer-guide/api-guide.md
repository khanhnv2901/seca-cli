# SECA API Service Guide

Turn SECA-CLI into a headless service that powers your web UI. This document explains the current runtime, available endpoints, and how to orchestrate demo scans via the REST API + job queue.

**For Frontend Developers:** See the [Frontend Integration Guide](./frontend-integration-guide.md) for detailed examples, code snippets, and common workflows.

---

## 🚀 Current Implementation

Launch the server with the built-in command:

```bash
seca serve \
  --addr 127.0.0.1:8080 \
  --auth-token $(openssl rand -hex 16) \
  --telemetry-limit 20 \
  --shutdown-timeout 30s
  --rate-limit 10 \
  --rate-burst 20
```

* Shares the same binary/`AppContext` as the CLI—no new daemon to deploy.
* Reads/writes the existing `engagements.json`, `http_results.json`, and `telemetry.jsonl` files on disk. **No database** required.
* Supports graceful shutdown via `SIGTERM` or `Ctrl+C` to complete in-flight requests.
* Built-in HTTP timeouts prevent resource exhaustion (read: 15s, write: 30s, idle: 120s).

### Available Handlers (v1)

| Method | Path                    | Notes |
| ------ | ----------------------- | ----- |
| GET    | `/api/health`           | readiness probe |
| GET    | `/api/engagements`      | list from `engagements.json` |
| POST   | `/api/engagements`      | create engagement (validates ROE + scope) |
| GET    | `/api/engagements/{id}` | single engagement |
| GET    | `/api/results/{id}`     | streams `http_results.json` |
| GET    | `/api/telemetry/{id}`   | pull history (`?limit=`) |
| POST   | `/api/jobs`             | enqueue a scan (currently `type=“http”`) |
| GET    | `/api/jobs`             | list recent jobs |
| GET    | `/api/jobs/{id}`        | job status (`pending`, `running`, `done`, `error`) |
| GET    | `/api/jobs-stream`      | Server-Sent Events feed for live updates |

### Auth Header

If `--auth-token` is set, every request must include:

```
X-Auth-Token: <token>
```

Missing/mismatched tokens → `401 Unauthorized`.

### Sample Curl Session

```bash
TOKEN=supersecret
curl -H "X-Auth-Token: $TOKEN" http://127.0.0.1:8080/api/health

curl -H "X-Auth-Token: $TOKEN" http://127.0.0.1:8080/api/engagements

curl -X POST -H "X-Auth-Token: $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
           "name": "Web Audit",
           "owner": "secops@example.com",
           "roe": "Standard ROE",
           "roe_agree": true,
           "scope": ["https://app.example.com"]
         }' \
     http://127.0.0.1:8080/api/engagements
```

---

## 🧱 Implementation Notes

* `cmd/serve.go` registers the Cobra command, HTTP server, and service adapters.
* `internal/api/server.go` hosts the router, DTOs, token middleware, and new job handlers.
* `internal/api/jobs.go` implements a lightweight in-memory queue (`JobManager`). Swap it for Redis/SQLite later without touching the HTTP layer.
* Services (`engagementAPIService`, `resultsAPIService`, `telemetryAPIService`, `jobAPIService`) simply reuse existing business logic under `cmd/`.

### Security Enhancements

* **Request size limits**: 1MB max body size to prevent DoS attacks
* **Timing-safe auth**: Constant-time token comparison prevents timing attacks
* **Secure IDs**: Cryptographically random 128-bit job IDs prevent enumeration
* **Error sanitization**: 5xx errors return generic messages to prevent information disclosure
* **Graceful shutdown**: Server waits for in-flight requests before terminating
* **Rate limiting**: Per-IP request throttling prevents API abuse and DoS attacks
* **CORS headers**: Configurable cross-origin support for frontend integration
* **Structured logging**: Production-grade observability with request/response tracking

### Resource Management

* **Job retention policy**: Automatic cleanup of old completed jobs (max 1000 jobs in memory)
* **Command timeouts**: 90-second timeout per job execution prevents hung processes
* **Buffer limits**: 1MB max stdout/stderr buffering prevents memory exhaustion
* **Context cancellation**: Proper cleanup when jobs are cancelled or timeout

---

## 🔌 Web Demo Workflow (Try-Our-Auditor Form)

Wire your web form to SECA via the API server so visitors can run a limited HTTP scan.

### Backend flow

1. **Ensure `seca serve` is running** with an auth token.
2. **Create a temporary engagement** via `POST /api/engagements`, passing the visitor domain in `scope`.
3. **Enqueue a scan**:
   ```http
   POST /api/jobs
   X-Auth-Token: $SECA_TOKEN
   Content-Type: application/json

   {
     "type": "http",
     "engagement_id": "<id-from-step-2>"
   }
   ```
   The job manager shells out to `seca check http --id <id> --roe-confirm --progress=false`.
4. **Watch progress** via `GET /api/jobs/{jobID}` or the SSE stream (`GET /api/jobs-stream`). When the job reports `status = done`, fetch `GET /api/results/{engagementID}` (and telemetry if desired).
5. **Return JSON** to the frontend so the visitor can view findings.
6. (Optional) **Cleanup** demo engagements/results on a cron so the data dir stays lean.

### Frontend UX

* Form submits to `/demo/run` (your backend).
* Show “Running secure checks…” while the backend waits for the job to finish.
* Render the returned JSON (status table, TLS info, etc.).
* Handle errors gracefully (`400` invalid domain, `500` timeout, etc.).

### Ops Checklist

* **Validation:** Only allow domains you own or explicitly permit. Reject `localhost`/private IPs.
* **Timeouts:** Wrap the CLI invocation (job runner) in a reasonable deadline (60–90 s).
* **Rate limiting:** Protect `/demo/run` so the public demo can’t be abused.
* **Logging:** Pipe job stdout/stderr into your existing observability stack.
* **Security:** Keep `seca serve` on a private network behind your web server; never expose it directly.
* **Graceful shutdown:** Use systemd or process managers that send SIGTERM for clean shutdowns.
* **Monitoring:** Watch for dropped job updates in high-traffic scenarios (see SSE subscriber buffer).

---

## Job API Reference

```jsonc
{
  "id": "job_1737405426185000",
  "type": "http",
  "status": "running",
  "started_at": "2025-01-20T04:17:06Z",
  "finished_at": null,
  "result_id": "eng-123",
  "error": ""
}
```

* `POST /api/jobs` – body `{ "type": "http", "engagement_id": "eng-123" }`, returns the created job (status `pending`).
* `GET /api/jobs/{id}` – current status.
* `GET /api/jobs` – list, newest first, accepts `?limit=`.
* `GET /api/jobs-stream` – SSE; each update is `event: job` with the JSON payload.

Status lifecycle: `pending → running → done` or `error` (with `error` populated).

---

## Extending the API

1. Define the DTO/service method in `internal/api/server.go`.
2. Implement the concrete service in `cmd/serve.go` (and reuse existing `cmd/*.go` logic).
3. Register the new route in `Server.routes()`.
4. Update this guide so consumers know it exists.

That’s it! Your frontend can now drive SECA via REST + SSE, while the CLI continues to offer the same battle-tested engagement workflows.
