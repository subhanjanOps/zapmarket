# Phase 4 — Import Service

**Goal:** Allow sellers and admins to bulk-import products via CSV upload. The prior session established the architecture; this document is the full implementation plan.

**Prior decision (from memory `project_async_csv_import_plan.md`):**
- New `import-service` on port 8085
- PostgreSQL job table + MinIO for file storage
- Worker polls `product-catalog-service /v1/categories/bulk` in batches of 100

---

## 4.1 Architecture Overview

```
Seller uploads CSV
      │
      ▼
[import-service HTTP]
  POST /v1/import/products
      │  1. Upload CSV to MinIO (bucket: zapmarket, prefix: imports/)
      │  2. Write job row to import_jobs (status=PENDING)
      │  Returns job_id immediately (202 Accepted)
      │
      ▼
[import-service Worker goroutine]
  Polls import_jobs WHERE status = 'PENDING'
      │  1. Download CSV from MinIO
      │  2. Parse + validate rows
      │  3. POST /v1/categories/bulk in batches of 100 via api-gateway
      │  4. POST /v1/products for each valid product row
      │  5. Update import_jobs: status = DONE / FAILED, error_log
      │
      ▼
Seller polls GET /v1/import/jobs/:id for status
```

---

## 4.2 Service Structure

```
services/import-service/
  cmd/main.go
  internal/
    domain/
      job.go          # ImportJob, ImportStatus, Row
    repository/
      job_repository.go
    service/
      importer.go     # orchestrates CSV parse → batch API calls
    handler/
      http/
        import_handler.go
    worker/
      worker.go       # polling loop
  pkg/config/config.go
  Dockerfile
```

---

## 4.3 Schema

```sql
CREATE TABLE import_jobs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id    UUID NOT NULL,
    file_key     TEXT NOT NULL,          -- MinIO object key
    status       TEXT NOT NULL DEFAULT 'PENDING',
                                         -- PENDING | PROCESSING | DONE | FAILED
    total_rows   INT,
    processed    INT NOT NULL DEFAULT 0,
    failed       INT NOT NULL DEFAULT 0,
    error_log    JSONB,                  -- [{row: N, error: "..."}]
    started_at   TIMESTAMPTZ,
    finished_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_import_jobs_pending ON import_jobs (id) WHERE status = 'PENDING';
CREATE INDEX idx_import_jobs_seller  ON import_jobs (seller_id, created_at DESC);
```

DB: use a new `import` database, consistent with service isolation principle.

---

## 4.4 CSV Format

```csv
name,slug,category_name,description,price,currency,stock_qty,attributes
"Blue T-Shirt","blue-t-shirt-m","Apparel","Cotton, size M",1999,INR,50,"{""color"":""blue"",""size"":""M""}"
```

| Column | Required | Notes |
|--------|----------|-------|
| `name` | Yes | Max 255 chars |
| `slug` | No | Auto-generated from name if absent (`slug.Make(name)`) |
| `category_name` | Yes | Resolved to `category_id` via `/v1/categories?name=` |
| `description` | No | |
| `price` | Yes | Integer, smallest currency unit (paise/cents) |
| `currency` | No | Default `INR` |
| `stock_qty` | No | If > 0, calls `inventory-service` AddStock after product creation |
| `attributes` | No | JSON string |

### Validation rules

- `name` non-empty
- `price` > 0
- `category_name` must resolve to a known category (pre-fetch all categories at worker start)
- `slug` uniqueness: on conflict, append `-2`, `-3`, etc.
- Max 10,000 rows per file (reject at upload time if `Content-Length` suggests more)
- Max file size: 5 MB

---

## 4.5 Worker Design

```go
func (w *Worker) Run(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    sem := make(chan struct{}, 3) // max 3 concurrent jobs
    for {
        select {
        case <-ctx.Done(): return
        case <-ticker.C:
            jobs := w.repo.ClaimPending(ctx, 3) // FOR UPDATE SKIP LOCKED
            for _, job := range jobs {
                sem <- struct{}{}
                go func(j *domain.ImportJob) {
                    defer func() { <-sem }()
                    w.processJob(ctx, j)
                }(job)
            }
        }
    }
}
```

`ClaimPending` uses `UPDATE ... SET status='PROCESSING' ... FOR UPDATE SKIP LOCKED LIMIT $1 RETURNING *` — safe under multiple worker instances.

### Batch API calls

The worker calls `product-catalog-service` via the api-gateway (not directly) so authentication and rate limiting are respected:

```go
// For each batch of 100 rows:
body := buildBulkRequest(rows)
resp, err := gatewayClient.Post(ctx, "/v1/products/bulk", token, body)
```

If `product-catalog-service` doesn't have a `/v1/products/bulk` endpoint yet, the worker calls `/v1/products` individually per row with a controlled concurrency of 10 goroutines.

---

## 4.6 HTTP Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/v1/import/products` | seller/admin | Upload CSV, returns `{job_id}` with 202 |
| `GET` | `/v1/import/jobs` | seller/admin | List own jobs, paginated |
| `GET` | `/v1/import/jobs/:id` | seller/admin | Get job status + error log |
| `DELETE` | `/v1/import/jobs/:id` | seller/admin | Cancel a PENDING job |

### Progress endpoint

`GET /v1/import/jobs/:id` returns:

```json
{
  "id": "...",
  "status": "PROCESSING",
  "total_rows": 500,
  "processed": 120,
  "failed": 3,
  "error_log": [
    {"row": 7, "error": "category 'Widgets' not found"},
    {"row": 42, "error": "price must be > 0"}
  ],
  "started_at": "2026-06-21T10:00:00Z"
}
```

---

## 4.7 seller-ui Integration

Add a "Bulk Import" button to `/dashboard/products`:

1. File picker (`.csv` only, max 5 MB client-side guard).
2. `POST /api/proxy/v1/import/products` with `multipart/form-data`.
3. On 202: show a progress bar that polls `GET /api/proxy/v1/import/jobs/:id` every 3 seconds.
4. On DONE: show summary — "495 products created, 5 failed" with a download link for the error rows.
5. On FAILED: show full error log.

---

## 4.8 docker-compose addition

```yaml
import-service:
  build:
    context: .
    dockerfile: services/import-service/Dockerfile
  container_name: zapmarket-import-service
  depends_on:
    postgres:
      condition: service_healthy
    minio-init:
      condition: service_completed_successfully
    api-gateway:
      condition: service_started
  environment:
    - DB_HOST=zapmarket-postgres
    - DB_NAME=import
    - HTTP_PORT=8085
    - MINIO_ENDPOINT=zapmarket-minio:9000
    - GATEWAY_URL=http://zapmarket-api-gateway:8000
    - APP_ENV=development
  ports:
    - "8085:8085"
  networks:
    - zapnet
```

---

## Acceptance criteria

- Upload a 1,000-row CSV, all valid rows → products visible in seller dashboard within 60 seconds
- Row-level errors don't abort the entire job; only failing rows are logged
- Duplicate slug is auto-resolved, not an error
- Job is idempotent: re-uploading the same file produces a new job (no dedup at file level)
- Worker survives product-catalog-service restart mid-job (resumes from last committed batch)

---

## Estimated effort

2 weeks (service skeleton + worker: 1 week; seller-ui progress UI: 3 days; testing: 4 days).
