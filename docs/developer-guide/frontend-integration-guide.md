# Frontend Integration Guide

This guide helps frontend developers integrate with the SECA REST API to build web UIs for security auditing workflows.

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [Authentication](#authentication)
3. [API Endpoints Reference](#api-endpoints-reference)
4. [Common Workflows](#common-workflows)
5. [Real-time Updates (SSE)](#real-time-updates-sse)
6. [Error Handling](#error-handling)
7. [Rate Limiting](#rate-limiting)
8. [CORS Configuration](#cors-configuration)
9. [Code Examples](#code-examples)

---

## Quick Start

### Starting the API Server

Your backend team will run the SECA API server:

```bash
seca serve \
  --addr 127.0.0.1:8080 \
  --auth-token your-secret-token-here \
  --cors-origins "http://localhost:3000,https://yourapp.com" \
  --rate-limit 10 \
  --rate-burst 20
```

### Base URL

```
http://127.0.0.1:8080
```

All API endpoints are prefixed with `/api/`.

---

## Authentication

Every API request requires an authentication token in the header:

```http
X-Auth-Token: your-secret-token-here
```

### Example with fetch:

```javascript
const API_TOKEN = 'your-secret-token-here';
const API_BASE_URL = 'http://127.0.0.1:8080/api';

async function apiRequest(endpoint, options = {}) {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    ...options,
    headers: {
      'X-Auth-Token': API_TOKEN,
      'Content-Type': 'application/json',
      ...options.headers,
    },
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'API request failed');
  }

  return response.json();
}
```

---

## API Endpoints Reference

### Health Check

Check if the API server is running.

```http
GET /api/health
```

**Response:**
```json
{
  "status": "ok"
}
```

---

### Engagements

#### List All Engagements

Get all security audit engagements.

```http
GET /api/engagements
```

**Response:**
```json
[
  {
    "id": "1762995581005162657",
    "name": "Production Web Audit",
    "owner": "security-team@company.com",
    "scope": ["https://app.example.com", "https://api.example.com"],
    "roe": "Standard ROE - no destructive testing",
    "created_at": "2025-01-20T10:30:00Z",
    "status": "active"
  }
]
```

#### Create New Engagement

Create a new security audit engagement.

```http
POST /api/engagements
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "Q1 Security Audit",
  "owner": "alice@company.com",
  "roe": "Standard penetration testing rules",
  "roe_agree": true,
  "scope": [
    "https://app.example.com",
    "https://api.example.com"
  ]
}
```

**Response:** (201 Created)
```json
{
  "id": "1762995581005162658",
  "name": "Q1 Security Audit",
  "owner": "alice@company.com",
  "scope": ["https://app.example.com", "https://api.example.com"],
  "roe": "Standard penetration testing rules",
  "created_at": "2025-01-20T11:00:00Z",
  "status": "active"
}
```

**Validation Rules:**
- `name`: Required, min 3 characters
- `owner`: Required, valid email format
- `roe`: Required
- `roe_agree`: Must be `true`
- `scope`: Required, array of valid URLs

#### Get Single Engagement

```http
GET /api/engagements/{id}
```

**Response:** Same as engagement object above

---

### Jobs (Scans)

#### Create a New Scan Job

Start a security scan for an engagement.

```http
POST /api/jobs
Content-Type: application/json
```

**Request Body:**
```json
{
  "type": "http",
  "engagement_id": "1762995581005162657"
}
```

**Response:** (202 Accepted)
```json
{
  "id": "job_de95e319d18112e4c4cfe8342277f13c",
  "type": "http",
  "status": "pending",
  "started_at": null,
  "finished_at": null,
  "result_id": "1762995581005162657",
  "error": ""
}
```

**Job Status Lifecycle:**
1. `pending` - Job queued, not started yet
2. `running` - Job is actively executing
3. `done` - Job completed successfully
4. `error` - Job failed (see `error` field)

**Important Notes:**
- Jobs run with a **90-second timeout**
- Maximum **1000 jobs** kept in memory (older completed jobs are auto-cleaned)
- Each job gets a cryptographically secure random ID

#### Get Job Status

Check the status of a running or completed job.

```http
GET /api/jobs/{job_id}
```

**Response:**
```json
{
  "id": "job_de95e319d18112e4c4cfe8342277f13c",
  "type": "http",
  "status": "done",
  "started_at": "2025-01-20T11:05:00Z",
  "finished_at": "2025-01-20T11:05:15Z",
  "result_id": "1762995581005162657",
  "error": ""
}
```

#### List Recent Jobs

Get a list of recent jobs (newest first).

```http
GET /api/jobs?limit=25
```

**Query Parameters:**
- `limit` (optional): Number of jobs to return (default: 25)

**Response:**
```json
[
  {
    "id": "job_f7789ca8985c051f97e892c7b011a5b9",
    "type": "http",
    "status": "running",
    "started_at": "2025-01-20T11:10:00Z",
    "finished_at": null,
    "result_id": "1762995581005162657",
    "error": ""
  },
  {
    "id": "job_de95e319d18112e4c4cfe8342277f13c",
    "type": "http",
    "status": "done",
    "started_at": "2025-01-20T11:05:00Z",
    "finished_at": "2025-01-20T11:05:15Z",
    "result_id": "1762995581005162657",
    "error": ""
  }
]
```

---

### Results

Get the security scan results for an engagement.

```http
GET /api/results/{engagement_id}
```

**Response:**
```json
{
  "engagement_id": "1762995581005162657",
  "timestamp": "2025-01-20T11:05:15Z",
  "findings": [
    {
      "category": "tls",
      "severity": "medium",
      "title": "TLS 1.0 Enabled",
      "description": "Server supports deprecated TLS 1.0 protocol",
      "recommendation": "Disable TLS 1.0 and use TLS 1.2+"
    },
    {
      "category": "headers",
      "severity": "low",
      "title": "Missing Security Header",
      "description": "X-Content-Type-Options header not set",
      "recommendation": "Add 'X-Content-Type-Options: nosniff' header"
    }
  ],
  "summary": {
    "total_checks": 25,
    "passed": 20,
    "failed": 5,
    "warnings": 3
  }
}
```

---

### Telemetry

Get audit logs and telemetry data for an engagement.

```http
GET /api/telemetry/{engagement_id}?limit=50
```

**Query Parameters:**
- `limit` (optional): Number of records to return (default: 10)

**Response:**
```json
[
  {
    "timestamp": "2025-01-20T11:05:10Z",
    "level": "info",
    "message": "TLS scan completed",
    "metadata": {
      "duration_ms": 1234,
      "protocols_tested": ["TLSv1.0", "TLSv1.2", "TLSv1.3"]
    }
  },
  {
    "timestamp": "2025-01-20T11:05:05Z",
    "level": "info",
    "message": "Starting HTTP security checks",
    "metadata": {
      "target": "https://app.example.com"
    }
  }
]
```

---

## Real-time Updates (SSE)

Use Server-Sent Events to get live job status updates without polling.

### Connect to Job Stream

```http
GET /api/jobs-stream
```

**Response:** (text/event-stream)
```
event: job
data: {"id":"job_xyz","type":"http","status":"pending",...}

event: job
data: {"id":"job_xyz","type":"http","status":"running",...}

event: job
data: {"id":"job_xyz","type":"http","status":"done",...}
```

### JavaScript Example:

```javascript
function subscribeToJobUpdates(onJobUpdate) {
  const eventSource = new EventSource(
    `${API_BASE_URL}/jobs-stream`,
    {
      headers: {
        'X-Auth-Token': API_TOKEN
      }
    }
  );

  eventSource.addEventListener('job', (event) => {
    const job = JSON.parse(event.data);
    onJobUpdate(job);
  });

  eventSource.onerror = (error) => {
    console.error('SSE connection error:', error);
    eventSource.close();
  };

  // Return cleanup function
  return () => eventSource.close();
}

// Usage:
const unsubscribe = subscribeToJobUpdates((job) => {
  console.log('Job update:', job);

  if (job.status === 'done') {
    console.log('Job completed successfully!');
  } else if (job.status === 'error') {
    console.error('Job failed:', job.error);
  }
});

// Clean up when component unmounts
// unsubscribe();
```

**Note:** Some browsers/libraries may not support custom headers with EventSource. In that case, consider using `fetch` with `ReadableStream` or add token as query parameter.

---

## Common Workflows

### Workflow 1: Run a Security Scan

**Step 1:** Create an engagement
```javascript
const engagement = await apiRequest('/engagements', {
  method: 'POST',
  body: JSON.stringify({
    name: 'Production Audit',
    owner: 'security@company.com',
    roe: 'Standard ROE',
    roe_agree: true,
    scope: ['https://app.example.com']
  })
});

console.log('Created engagement:', engagement.id);
```

**Step 2:** Start a scan job
```javascript
const job = await apiRequest('/jobs', {
  method: 'POST',
  body: JSON.stringify({
    type: 'http',
    engagement_id: engagement.id
  })
});

console.log('Job started:', job.id);
```

**Step 3:** Poll for job completion (or use SSE)
```javascript
async function waitForJob(jobId) {
  while (true) {
    const job = await apiRequest(`/jobs/${jobId}`);

    if (job.status === 'done') {
      return job;
    } else if (job.status === 'error') {
      throw new Error(job.error);
    }

    // Wait 2 seconds before next poll
    await new Promise(resolve => setTimeout(resolve, 2000));
  }
}

const completedJob = await waitForJob(job.id);
```

**Step 4:** Fetch results
```javascript
const results = await apiRequest(`/results/${engagement.id}`);
console.log('Scan results:', results);
```

---

### Workflow 2: Display Live Scan Progress

```javascript
async function runScanWithLiveUpdates(engagementId) {
  // Subscribe to updates first
  const unsubscribe = subscribeToJobUpdates((job) => {
    updateUI({
      status: job.status,
      startedAt: job.started_at,
      finishedAt: job.finished_at
    });

    if (job.status === 'done') {
      // Fetch and display results
      fetchResults(job.result_id);
      unsubscribe();
    }
  });

  // Start the job
  const job = await apiRequest('/jobs', {
    method: 'POST',
    body: JSON.stringify({
      type: 'http',
      engagement_id: engagementId
    })
  });

  return job.id;
}
```

---

### Workflow 3: "Try Our Auditor" Demo Form

For a public demo where visitors can scan their own domains:

```javascript
async function runDemoScan(userDomain) {
  try {
    // 1. Validate domain (client-side)
    if (!isValidDomain(userDomain)) {
      throw new Error('Invalid domain');
    }

    // 2. Create temporary engagement
    const engagement = await apiRequest('/engagements', {
      method: 'POST',
      body: JSON.stringify({
        name: `Demo Scan - ${userDomain}`,
        owner: 'demo@visitor',
        roe: 'Demo scan - limited checks only',
        roe_agree: true,
        scope: [`https://${userDomain}`]
      })
    });

    // 3. Start scan
    const job = await apiRequest('/jobs', {
      method: 'POST',
      body: JSON.stringify({
        type: 'http',
        engagement_id: engagement.id
      })
    });

    // 4. Show "Running scan..." message
    showLoadingState(job.id);

    // 5. Wait for completion
    const completedJob = await waitForJob(job.id);

    // 6. Fetch and display results
    const results = await apiRequest(`/results/${engagement.id}`);
    displayResults(results);

  } catch (error) {
    showError(error.message);
  }
}
```

**Important Security Notes for Public Demos:**
- **Server-side validation**: Never trust client input - validate domains on backend
- **Rate limiting**: Enforce strict rate limits per IP
- **Scope restriction**: Only allow specific domains or patterns
- **Reject private IPs**: Block localhost, 127.0.0.1, 192.168.x.x, etc.
- **Cleanup**: Delete demo engagements/results after 24 hours

---

## Error Handling

All errors return JSON with this format:

```json
{
  "error": "human-readable error message"
}
```

### HTTP Status Codes

| Code | Meaning | Example |
|------|---------|---------|
| 200 | Success | GET request succeeded |
| 201 | Created | Engagement created successfully |
| 202 | Accepted | Job queued for processing |
| 400 | Bad Request | Invalid JSON or missing required fields |
| 401 | Unauthorized | Missing or invalid auth token |
| 404 | Not Found | Engagement/job ID doesn't exist |
| 405 | Method Not Allowed | Wrong HTTP method (e.g., DELETE on /health) |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Server-side error (details logged server-side) |

### Error Handling Example

```javascript
async function apiRequestWithErrorHandling(endpoint, options) {
  try {
    const response = await fetch(`${API_BASE_URL}${endpoint}`, {
      ...options,
      headers: {
        'X-Auth-Token': API_TOKEN,
        'Content-Type': 'application/json',
        ...options.headers,
      },
    });

    if (!response.ok) {
      const error = await response.json();

      switch (response.status) {
        case 401:
          // Handle authentication error
          redirectToLogin();
          break;
        case 429:
          // Handle rate limiting
          showRateLimitMessage();
          break;
        case 404:
          // Handle not found
          showNotFoundMessage();
          break;
        default:
          throw new Error(error.error || 'API request failed');
      }
    }

    return response.json();

  } catch (error) {
    console.error('API error:', error);
    throw error;
  }
}
```

---

## Rate Limiting

The API enforces per-IP rate limiting to prevent abuse.

**Default Configuration:**
- **Rate limit**: 10 requests/second per IP
- **Burst size**: 20 requests (allows short bursts)

**Rate Limit Headers** (not currently implemented, but planned):
```http
X-RateLimit-Limit: 10
X-RateLimit-Remaining: 7
X-RateLimit-Reset: 1642687200
```

**When Rate Limited:**
```http
HTTP/1.1 429 Too Many Requests
Content-Type: application/json

{
  "error": "rate limit exceeded"
}
```

**Frontend Handling:**
```javascript
async function apiRequestWithRetry(endpoint, options, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await apiRequest(endpoint, options);
    } catch (error) {
      if (error.status === 429 && i < maxRetries - 1) {
        // Wait with exponential backoff
        const waitTime = Math.pow(2, i) * 1000;
        await new Promise(resolve => setTimeout(resolve, waitTime));
        continue;
      }
      throw error;
    }
  }
}
```

---

## CORS Configuration

The API supports CORS for cross-origin requests.

**Server Configuration:**
```bash
seca serve --cors-origins "http://localhost:3000,https://yourapp.com"
```

**CORS Headers Set by Server:**
```http
Access-Control-Allow-Origin: http://localhost:3000
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type, X-Auth-Token
Access-Control-Max-Age: 3600
```

**Preflight Request Handling:**
The server automatically handles `OPTIONS` preflight requests.

---

## Code Examples

### Complete Working Example

See [example-frontend.html](./example-frontend.html) for a complete, working HTML/JavaScript demo that you can open in your browser. It demonstrates:
- Form validation and submission
- Creating engagements via API
- Starting scan jobs
- Polling for job completion
- Displaying results with proper styling
- Error handling

**To use it:**
1. Start your SECA API server: `seca serve --addr 127.0.0.1:8080 --auth-token your-token`
2. Update `API_BASE_URL` and `API_TOKEN` in the HTML file
3. Open the file in your browser
4. Enter a domain and run a scan

---

### React Hook Example

```javascript
import { useState, useEffect } from 'react';

function useJobStatus(jobId) {
  const [job, setJob] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    if (!jobId) return;

    let unsubscribe;

    // Subscribe to real-time updates
    unsubscribe = subscribeToJobUpdates((updatedJob) => {
      if (updatedJob.id === jobId) {
        setJob(updatedJob);
        setLoading(false);

        // Stop listening when job is done
        if (updatedJob.status === 'done' || updatedJob.status === 'error') {
          unsubscribe?.();
        }
      }
    });

    // Also fetch current status
    apiRequest(`/jobs/${jobId}`)
      .then(setJob)
      .catch(setError)
      .finally(() => setLoading(false));

    return () => unsubscribe?.();
  }, [jobId]);

  return { job, loading, error };
}

// Usage in component:
function ScanStatus({ jobId }) {
  const { job, loading, error } = useJobStatus(jobId);

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <div>
      <h3>Job Status: {job.status}</h3>
      {job.status === 'done' && <button onClick={() => fetchResults(job.result_id)}>View Results</button>}
      {job.status === 'error' && <p>Error: {job.error}</p>}
    </div>
  );
}
```

---

### Vue.js Composition API Example

```javascript
import { ref, watch } from 'vue';

export function useSecurityScan() {
  const job = ref(null);
  const results = ref(null);
  const loading = ref(false);
  const error = ref(null);

  async function startScan(engagementId) {
    loading.value = true;
    error.value = null;

    try {
      // Start job
      const newJob = await apiRequest('/jobs', {
        method: 'POST',
        body: JSON.stringify({
          type: 'http',
          engagement_id: engagementId
        })
      });

      job.value = newJob;

      // Subscribe to updates
      const unsubscribe = subscribeToJobUpdates((updatedJob) => {
        if (updatedJob.id === newJob.id) {
          job.value = updatedJob;

          if (updatedJob.status === 'done') {
            fetchResults(engagementId);
            unsubscribe();
          }
        }
      });

    } catch (err) {
      error.value = err.message;
    } finally {
      loading.value = false;
    }
  }

  async function fetchResults(engagementId) {
    try {
      results.value = await apiRequest(`/results/${engagementId}`);
    } catch (err) {
      error.value = err.message;
    }
  }

  return {
    job,
    results,
    loading,
    error,
    startScan
  };
}
```

---

### TypeScript Types

```typescript
// API Types
interface Engagement {
  id: string;
  name: string;
  owner: string;
  scope: string[];
  roe: string;
  created_at: string;
  status: 'active' | 'completed' | 'archived';
}

interface Job {
  id: string;
  type: 'http';
  status: 'pending' | 'running' | 'done' | 'error';
  started_at: string | null;
  finished_at: string | null;
  result_id: string;
  error: string;
}

interface Finding {
  category: string;
  severity: 'critical' | 'high' | 'medium' | 'low' | 'info';
  title: string;
  description: string;
  recommendation: string;
}

interface ScanResults {
  engagement_id: string;
  timestamp: string;
  findings: Finding[];
  summary: {
    total_checks: number;
    passed: number;
    failed: number;
    warnings: number;
  };
}

interface TelemetryRecord {
  timestamp: string;
  level: 'info' | 'warn' | 'error';
  message: string;
  metadata?: Record<string, any>;
}

// API Client
class SecaApiClient {
  constructor(
    private baseUrl: string,
    private authToken: string
  ) {}

  private async request<T>(
    endpoint: string,
    options?: RequestInit
  ): Promise<T> {
    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      ...options,
      headers: {
        'X-Auth-Token': this.authToken,
        'Content-Type': 'application/json',
        ...options?.headers,
      },
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'API request failed');
    }

    return response.json();
  }

  // Engagements
  async listEngagements(): Promise<Engagement[]> {
    return this.request<Engagement[]>('/engagements');
  }

  async createEngagement(data: {
    name: string;
    owner: string;
    roe: string;
    roe_agree: boolean;
    scope: string[];
  }): Promise<Engagement> {
    return this.request<Engagement>('/engagements', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async getEngagement(id: string): Promise<Engagement> {
    return this.request<Engagement>(`/engagements/${id}`);
  }

  // Jobs
  async startJob(engagementId: string): Promise<Job> {
    return this.request<Job>('/jobs', {
      method: 'POST',
      body: JSON.stringify({
        type: 'http',
        engagement_id: engagementId,
      }),
    });
  }

  async getJob(jobId: string): Promise<Job> {
    return this.request<Job>(`/jobs/${jobId}`);
  }

  async listJobs(limit = 25): Promise<Job[]> {
    return this.request<Job[]>(`/jobs?limit=${limit}`);
  }

  // Results
  async getResults(engagementId: string): Promise<ScanResults> {
    return this.request<ScanResults>(`/results/${engagementId}`);
  }

  // Telemetry
  async getTelemetry(
    engagementId: string,
    limit = 10
  ): Promise<TelemetryRecord[]> {
    return this.request<TelemetryRecord[]>(
      `/telemetry/${engagementId}?limit=${limit}`
    );
  }
}

// Usage:
const client = new SecaApiClient(
  'http://127.0.0.1:8080/api',
  'your-auth-token'
);

const engagements = await client.listEngagements();
```

---

## Testing Tips

### Use curl for Quick Testing

```bash
# Health check
curl -H "X-Auth-Token: your-token" http://127.0.0.1:8080/api/health

# List engagements
curl -H "X-Auth-Token: your-token" http://127.0.0.1:8080/api/engagements

# Create engagement
curl -X POST \
  -H "X-Auth-Token: your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Engagement",
    "owner": "test@example.com",
    "roe": "Test ROE",
    "roe_agree": true,
    "scope": ["https://example.com"]
  }' \
  http://127.0.0.1:8080/api/engagements

# Start a job
curl -X POST \
  -H "X-Auth-Token: your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "http",
    "engagement_id": "1762995581005162657"
  }' \
  http://127.0.0.1:8080/api/jobs

# Check job status
curl -H "X-Auth-Token: your-token" \
  http://127.0.0.1:8080/api/jobs/job_xyz123

# Stream job updates
curl -H "X-Auth-Token: your-token" \
  -N \
  http://127.0.0.1:8080/api/jobs-stream
```

---

## Troubleshooting

### CORS Errors

**Problem:** Browser shows CORS policy error

**Solution:** Ask backend team to add your origin to `--cors-origins` flag:
```bash
seca serve --cors-origins "http://localhost:3000"
```

---

### 401 Unauthorized

**Problem:** All requests return 401

**Solution:**
1. Verify auth token is correct
2. Check `X-Auth-Token` header is being sent
3. Make sure token has no extra spaces or newlines

---

### 429 Rate Limit Exceeded

**Problem:** Getting rate limited during development

**Solution:** Ask backend team to increase rate limits for development:
```bash
seca serve --rate-limit 100 --rate-burst 200
```

---

### EventSource/SSE Not Working

**Problem:** Real-time updates not working

**Solutions:**
1. Some browsers don't support custom headers with EventSource
2. Use `fetch` with ReadableStream instead
3. Or add token as query parameter (requires backend change)
4. Fall back to polling every 2-5 seconds

---

## Support

For issues or questions:
- Check the [API Guide](./api-guide.md) for backend details
- Report bugs at: https://github.com/yourusername/seca-cli/issues
- Contact backend team for API server issues

---

## Changelog

**v1.0.0** (2025-01-20)
- Initial API release
- Basic CRUD operations for engagements
- Job queue with real-time updates via SSE
- Results and telemetry endpoints
- Authentication, rate limiting, CORS support
