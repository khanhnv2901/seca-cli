## 🚀 Current Implementation (2025 roadmap checkpoint)

The initial version of this guide has shipped! You can now boot the REST layer directly from the CLI:

```bash
seca serve \\
  --addr 127.0.0.1:8080 \\
  --auth-token $(openssl rand -hex 16) \\
  --telemetry-limit 20
```

This command installs an HTTP server backed by `internal/api` and the existing `cmd/` logic. No database layer was added—handlers continue to read/write the JSON/CSV telemetry artifacts on disk so the API and CLI stay perfectly in-sync.

### Available handlers (v1)

| Method | Path                    | Notes |
| ------ | ----------------------- | ----- |
| GET    | `/api/health`           | lightweight readiness probe |
| GET    | `/api/engagements`      | returns JSON list from `engagements.json` |
| POST   | `/api/engagements`      | creates a new engagement (validates ROE + scope) |
| GET    | `/api/engagements/{id}` | fetches a single engagement |
| GET    | `/api/results/{id}`     | streams the latest `results.json` blob |
| GET    | `/api/telemetry/{id}`   | uses `telemetry.jsonl`, respects `?limit=` |

Future sections of this doc will be updated as additional endpoints (e.g., `/api/check/http`) are implemented.

### Auth header

If `--auth-token` is provided, every request must include:

```
X-Auth-Token: <token>
```

Missing or mismatched tokens result in `401 Unauthorized`.

### Sample curl session

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

## 🧱 Implementation notes

* `cmd/serve.go` wires the Cobra command, spins up the HTTP server, and injects the CLI `AppContext` into the API services.
* `internal/api/server.go` defines the REST router, shared DTOs, token middleware, and JSON helpers.
* Services such as `engagementAPIService`, `resultsAPIService`, and `telemetryAPIService` simply call the existing functions in `cmd/` (no duplicated business logic).
* When additional endpoints are needed, add an interface method to `internal/api/server.go`, implement it in `cmd/serve.go`, and register the route in `routes()`.

With these pieces in place a frontend (Next.js, Tauri, etc.) can call SECA via REST while the classic CLI workflows remain untouched.
```
