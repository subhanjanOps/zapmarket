# Stage 14 — Async CSV Import Service (Batch Processor)

**Goal:** Build an internal `batch-processor` service that consumes Kafka import-request events, processes large CSVs from MinIO in batches, and publishes a completion event back to Kafka. The catalog service handles upload and event emission; the UI is notified of completion via SSE on the catalog service.

---

## Preconditions

- Stage 7 complete — Kafka is live, `pkg/kafka` producer/consumer wrappers exist.
- `product-catalog-service` `POST /api/v1/categories/bulk` endpoint is live (topological sort included).
- MinIO is active in `docker-compose.yml`.
- `pkg/storage` (MinIO client) already exists — used by catalog service for product images.

---

## Architecture

```
Browser
  │  POST /api/v1/categories/import  (multipart CSV, admin only)
  ▼
catalog-service
  │  1. Upload CSV → MinIO  (file_id = UUID key)
  │  2. INSERT outbox row → DB  (same transaction as import_jobs record)
  │  3. return {file_id, status: "PENDING"} immediately
  │
  │  GET /api/v1/categories/import/{file_id}/status  ← SSE stream
  │  pushes status updates to the browser as events arrive
  ▼
Kafka topic: "catalog.import.requested"
  payload: { file_id, bucket, entity_type, requested_by, requested_at }
  ▼
batch-processor  (internal service — no public HTTP)
  │  consume event
  │  download CSV from MinIO
  │  parse + batch (configurable batch size, default 100 rows)
  │    POST catalog-service /api/v1/categories/bulk  (service token)
  │    on partial error: collect row-level errors, continue
  │  publish completion event
  ▼
Kafka topic: "catalog.import.completed"
  payload: { file_id, entity_type, status, total_rows, processed, failed, errors[] }
  ▼
catalog-service  (consumes its own completion topic)
  │  UPDATE import_jobs SET status = COMPLETED | FAILED
  │  push SSE event to all subscribers watching this file_id
  ▼
Browser  ← SSE event: {status, processed, total_rows, failed}
  show success / error toast, refresh category list
```

---

## New Kafka Topics

Add to `pkg/kafka/topics.go`:

```go
TopicCatalogImportRequested = "catalog.import.requested"
TopicCatalogImportCompleted = "catalog.import.completed"
```

---

## Tasks

### 14.1 — catalog-service: import upload endpoint

- [ ] `migrations/` — add `import_jobs` table to catalog-service DB:

```sql
CREATE TABLE import_jobs (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  file_id      UUID NOT NULL UNIQUE,
  entity_type  TEXT NOT NULL,          -- 'categories'
  status       TEXT NOT NULL DEFAULT 'PENDING'
                 CHECK (status IN ('PENDING','PROCESSING','COMPLETED','FAILED')),
  total_rows   INT,
  processed    INT NOT NULL DEFAULT 0,
  failed       INT NOT NULL DEFAULT 0,
  error_log    JSONB NOT NULL DEFAULT '[]',
  requested_by UUID,                   -- user ID from JWT claims
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

- [ ] `internal/repository/import_job_repository.go` — `CreateJob`, `GetJobByFileID`, `UpdateJobStatus`, `UpdateJobProgress`
- [ ] `internal/service/import_service.go`:
  - `RequestImport(ctx, file, entityType, userID)` — uploads to MinIO, creates job row, writes outbox row for `catalog.import.requested`
  - `GetImportStatus(ctx, fileID)` — returns current job row
  - `HandleImportCompleted(ctx, event)` — updates job row from completion event payload
- [ ] `internal/handler/http/import_handler.go`:
  - `POST /api/v1/categories/import` — multipart, max 50 MB, admin-only; calls `RequestImport`, returns `{file_id, status}`
  - `GET /api/v1/categories/import/{file_id}/status` — SSE endpoint; streams status updates to browser

### 14.2 — catalog-service: Kafka wiring

- [ ] Add Kafka producer to `main.go` (same pattern as outbox relay in other services)
- [ ] Outbox relay already runs — just ensure `catalog.import.requested` rows are picked up
- [ ] Add Kafka consumer in `main.go` for `catalog.import.completed` topic:
  - On message: call `importSvc.HandleImportCompleted(ctx, event)`
  - Push SSE event to all active subscribers for that `file_id`

### 14.3 — catalog-service: SSE broker

- [ ] `internal/broker/sse_broker.go` — in-memory pub/sub:
  ```go
  type SSEBroker struct {
    mu          sync.RWMutex
    subscribers map[string][]chan SSEEvent  // keyed by file_id
  }
  func (b *SSEBroker) Subscribe(fileID string) (<-chan SSEEvent, func())
  func (b *SSEBroker) Publish(fileID string, event SSEEvent)
  ```
- [ ] `GET /api/v1/categories/import/{file_id}/status` sets `Content-Type: text/event-stream`, registers a subscriber, streams events, cleans up on client disconnect

### 14.4 — Scaffold batch-processor service

- [ ] Create `services/batch-processor/` with `go.mod` (`module github.com/zapmarket/zapmarket/services/batch-processor`)
- [ ] Add to `go.work`
- [ ] `pkg/config/config.go` — env vars:
  - `KAFKA_BROKERS` (default `localhost:9092`)
  - `CONSUMER_GROUP` (default `batch-processor`)
  - `CATALOG_SERVICE_ADDR` (default `http://localhost:8081`)
  - `CATALOG_SERVICE_TOKEN` — admin JWT for calling catalog-service bulk endpoint
  - `BATCH_SIZE` (default 100)
  - `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET`
- [ ] No public HTTP — only a `/healthz` endpoint for liveness checks (same pattern as notification-service)

### 14.5 — batch-processor: consumer + processor

- [ ] `internal/consumer/import_consumer.go` — subscribes to `catalog.import.requested`:
  - On event: call `processor.Process(ctx, event)`
  - On unrecoverable error: publish `catalog.import.completed` with `status: FAILED`
- [ ] `internal/processor/category_processor.go`:
  - Download CSV from MinIO via `pkg/storage`
  - Strip BOM, parse with `LazyQuotes=true`, `FieldsPerRecord=-1`
  - Slice rows into batches of `BATCH_SIZE`
  - For each batch: `POST catalog-service/api/v1/categories/bulk` with `Authorization: Bearer <CATALOG_SERVICE_TOKEN>`
  - Collect row-level errors from response body
  - After all batches: publish `catalog.import.completed` event to Kafka
- [ ] `internal/processor/processor.go` — `Processor` interface with `Process(ctx, ImportRequestedEvent) error`; pluggable for future entity types (products, SKUs)

### 14.6 — docker-compose integration

- [ ] Add `zapmarket-batch-processor` to `docker-compose.yml` (no exposed port, internal only)
- [ ] Environment: Kafka brokers, MinIO creds, catalog service addr, service token
- [ ] Depends on: Kafka, MinIO, catalog-service

### 14.7 — catalog-service: import history endpoint

- [ ] `GET /api/v1/categories/imports` — paginated list of all import jobs (admin only); supports `?status=FAILED&limit=20&offset=0`; returns `[{file_id, status, entity_type, total_rows, processed, failed, requested_by, created_at, updated_at}]` with total count
- [ ] `GET /api/v1/categories/imports/{file_id}` — single job detail including full `error_log`

### 14.8 — UI: import flow (categories page)

- [ ] `services/backoffice-ui/lib/api.ts`:
  - `requestCategoryImport(token, file): Promise<{file_id: string, status: string}>`
  - `getCategoryImports(token, params): Promise<{data: ImportJob[], total: number}>`
  - `getCategoryImport(token, fileId): Promise<ImportJob>`
  - `subscribeToCategoryImport(token, fileId, onEvent, onError): EventSource` — wraps browser `EventSource`
- [ ] `services/backoffice-ui/app/dashboard/categories/page.tsx`:
  - Restore Import button (file picker, accepts `.csv`)
  - On file select: call `requestCategoryImport` → get `file_id`
  - Show loading toast: "Importing… (0 / ? rows)"
  - Open `EventSource` on `/api/v1/categories/import/{file_id}/status`
  - On each SSE event: update toast message with `{processed}/{total_rows}`
  - On `COMPLETED`: close `EventSource`, dismiss loading toast, show success toast, refresh list
  - On `FAILED`: close `EventSource`, dismiss loading toast, show error toast with failed count + details
  - On page unmount: close `EventSource`

### 14.9 — UI: Import Jobs monitor tab

New page at `app/dashboard/categories/imports/page.tsx` — a dedicated tab/page where admins can monitor all past and active import jobs.

**Tab navigation**: add "Import Jobs" tab link alongside the existing "Categories" view in the sidebar or as a sub-nav within the categories section.

**Jobs table columns**: Status badge · Entity type · Total rows · Processed · Failed · Requested by · Started · Duration

**Status badges**:
- `PENDING` — muted grey pill
- `PROCESSING` — blue animated pill (pulsing dot)
- `COMPLETED` — green pill
- `FAILED` — red pill

**Live updates**: on page load, open a shared `EventSource` that subscribes to all active jobs (or poll `GET /categories/imports?status=PENDING,PROCESSING` every 3s as a simpler fallback). When a job transitions to `COMPLETED` or `FAILED`, update its row in place without a full reload.

**Row expand / detail drawer**: clicking a `FAILED` or `COMPLETED` row expands an inline detail panel (or slides open a drawer) showing the full `error_log` table — columns: Row # · Error message. Errors are paginated client-side (show 50 at a time) since `error_log` is a JSONB array.

**Filter bar**: filter by `status` (All / Pending / Processing / Completed / Failed) and `entity_type` (All / Categories / Products). Pagination matches the categories page pattern (20 per page, same `Pagination` component).

**Empty state**: "No import jobs yet — use the Import CSV button on the Categories page to get started."

- [ ] `app/dashboard/categories/imports/page.tsx` — implement the monitor page
- [ ] Add "Import Jobs" link to sidebar nav (under Categories section)
- [ ] Reuse `Pagination` component from `categories/page.tsx`
- [ ] Reuse `TableSkeleton` from `Skeleton` component

---

## Event Payloads

**`catalog.import.requested`**
```json
{
  "file_id": "uuid",
  "bucket": "zapmarket",
  "entity_type": "categories",
  "requested_by": "user-uuid",
  "requested_at": "2026-06-18T10:00:00Z"
}
```

**`catalog.import.completed`**
```json
{
  "file_id": "uuid",
  "entity_type": "categories",
  "status": "COMPLETED",
  "total_rows": 1200,
  "processed": 1195,
  "failed": 5,
  "errors": [
    { "row": 42, "message": "UNRESOLVABLE_PARENTS: parent not found: sports" }
  ]
}
```

---

## Out of scope

- **Real-time progress mid-batch** — the completion event carries final counts; per-batch progress updates would require the batch-processor to publish intermediate events, which is a future enhancement
- **Other entity types** (products, SKUs) — `Processor` interface is designed for it; add `ProductProcessor` later
- **Dead-letter queue** — failed events are logged; a dedicated DLQ topic is a future hardening item (Stage 12)
- **Re-import / retry UI** — re-upload is the recovery path for now

---

## Definition of done

- [ ] `go build ./...` passes for both `catalog-service` and `batch-processor`
- [ ] `POST /api/v1/categories/import` returns `{file_id}` within 1s for a 1000-row CSV
- [ ] Kafka topic `catalog.import.requested` receives the event (visible in Kafka UI at `:8090`)
- [ ] batch-processor logs batch progress to stdout; all rows inserted in catalog DB
- [ ] Kafka topic `catalog.import.completed` receives the completion event
- [ ] SSE stream on `/api/v1/categories/import/{file_id}/status` delivers the final status event
- [ ] UI Import button completes end-to-end: upload → loading toast → success/error toast → list refreshes
- [ ] Import Jobs monitor page shows all jobs with correct status badges; FAILED rows expand to show error_log
- [ ] A job that is PROCESSING shows a live animated badge; transitions to COMPLETED/FAILED in place without page reload
- [ ] `docker compose up -d` brings batch-processor up alongside the rest of the stack
