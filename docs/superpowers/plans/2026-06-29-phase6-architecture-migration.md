# Phase 6 — Architecture Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the five non-compliant services from the flat `internal/service/repository/handler` layout to the layered Clean Architecture used by `currency-service` (`domain / application / infrastructure / interfaces`). Also implement seller bulk CSV import.

**Architecture:** Each service migration is independent and safe because the external API contracts (HTTP routes, gRPC endpoints, Kafka topics) do not change — only the internal folder structure and dependency wiring change. `currency-service` is the reference implementation. Migrate one service per sprint. Use the Strangler Fig pattern: new folders coexist with old ones, old files deleted only after tests pass for the new layout.

**Tech Stack:** Go 1.25, no new dependencies — restructuring only.

## Global Constraints

- External API contracts (routes, proto signatures, Kafka topics) must not change during migration
- All existing tests must pass before deleting any old file
- Each service migration is a separate PR — do not batch multiple services
- New `application/usecases/interfaces.go` must define interfaces for every use case
- Domain layer must contain zero imports from `net/http`, `database/sql`, `kafka`, or `redis`
- Migration order: inventory → payment → order-management → product-catalog → auth

---

## Migration Pattern (apply identically to each service)

For each service, follow these steps:

### Step A — Create new folder structure alongside old one

```bash
mkdir -p services/<service>/internal/{domain/entities,domain/repositories,domain/errors,application/usecases,application/dto,application/ports,infrastructure/postgres,infrastructure/redis,infrastructure/kafka,interfaces/http,interfaces/grpc,interfaces/consumers}
```

### Step B — Move domain models

1. Copy the existing `internal/domain/models.go` entity structs to `internal/domain/entities/`
2. Copy repository interfaces to `internal/domain/repositories/`
3. Remove any infrastructure imports from the domain layer
4. Run `go build ./internal/domain/...` — must compile with zero infra imports

### Step C — Create use case layer

1. For each method in `internal/service/*.go`, create a corresponding `internal/application/usecases/<name>.go`
2. Each use case receives repository interfaces via constructor injection
3. Extract DTOs from handler layer into `internal/application/dto/`
4. Define all use case interfaces in `internal/application/usecases/interfaces.go`

### Step D — Move infrastructure

1. Move `internal/repository/*.go` → `internal/infrastructure/postgres/`
2. Move `internal/infrastructure/redisstore/` → `internal/infrastructure/redis/`
3. Implementations reference domain interfaces, not concrete types

### Step E — Move interfaces

1. Move `internal/handler/http/` → `internal/interfaces/http/`
2. Move `internal/handler/grpc/` → `internal/interfaces/grpc/`
3. Handlers call use cases only, never repositories directly

### Step F — Update main.go wiring

Rebuild the dependency graph in `main.go` / `cmd/server/main.go` using the new layer constructors.

### Step G — Delete old files

Only after `go test ./...` passes. Delete:
- `internal/service/`
- `internal/repository/`
- `internal/handler/`

---

## Task 1: Migrate inventory-service

**Files:** (full restructure of `services/inventory-service/internal/`)

**Reference:** `services/currency-service/` — compare each new file against its currency-service equivalent.

- [ ] **Step 1: Create new folder skeleton**

```bash
cd services/inventory-service
mkdir -p internal/{domain/entities,domain/repositories,domain/errors,application/usecases,application/dto,application/ports,infrastructure/postgres,infrastructure/redis,interfaces/grpc,interfaces/http}
```

- [ ] **Step 2: Move domain entities**

Create `services/inventory-service/internal/domain/entities/inventory.go`:

```go
package entities

import "time"

// Copy from existing models.go — remove any sql/kafka/redis imports

type Warehouse struct {
    ID        string
    Name      string
    City      string
    State     string
    Pincode   string
    IsActive  bool
    CreatedAt time.Time
}

type Inventory struct {
    ID           string
    SKUID        string
    WarehouseID  string
    QtyOnHand    int
    QtyReserved  int
    QtyAvailable int // computed: QtyOnHand - QtyReserved
    UpdatedAt    time.Time
}

type Reservation struct {
    ID          string
    InventoryID string
    OrderID     string
    SKUID       string
    Qty         int
    Status      string
    ExpiresAt   time.Time
    CreatedAt   time.Time
}

type LedgerEntry struct {
    ID           string
    InventoryID  string
    OrderID      string
    MovementType string
    QtyDelta     int
    QtyBefore    int
    QtyAfter     int
    Note         string
    CreatedAt    time.Time
}
```

- [ ] **Step 3: Define repository interfaces in domain layer**

Create `services/inventory-service/internal/domain/repositories/interfaces.go`:

```go
package repositories

import (
    "context"
    "time"

    "github.com/zapmarket/inventory-service/internal/domain/entities"
)

type InventoryRepository interface {
    GetBySKU(ctx context.Context, skuID, warehouseID string) (*entities.Inventory, error)
    UpdateReserved(ctx context.Context, inventoryID string, delta int) error
}

type ReservationRepository interface {
    Create(ctx context.Context, r *entities.Reservation) error
    FindByID(ctx context.Context, id string) (*entities.Reservation, error)
    Release(ctx context.Context, id string) error
    FindExpired(ctx context.Context, before time.Time) ([]*entities.Reservation, error)
}

type LedgerRepository interface {
    Insert(ctx context.Context, entry *entities.LedgerEntry) error
}

type WarehouseRepository interface {
    FindByID(ctx context.Context, id string) (*entities.Warehouse, error)
    FindDefault(ctx context.Context) (*entities.Warehouse, error)
}
```

- [ ] **Step 4: Build domain layer — verify no infra imports**

```bash
cd services/inventory-service
go build ./internal/domain/...
```

Expected: compiles. Any import of `database/sql`, `redis`, `kafka`, or `net/http` is a failure — remove it.

- [ ] **Step 5: Create use case interfaces**

Create `services/inventory-service/internal/application/usecases/interfaces.go`:

```go
package usecases

import "context"

type ReserveStockUseCase interface {
    Execute(ctx context.Context, req ReserveStockRequest) (*ReserveStockResult, error)
}

type ReleaseStockUseCase interface {
    Execute(ctx context.Context, reservationID string) error
}

type DeductStockUseCase interface {
    Execute(ctx context.Context, reservationID string) error
}

type AddStockUseCase interface {
    Execute(ctx context.Context, skuID, warehouseID string, qty int) error
}

type GetStockUseCase interface {
    Execute(ctx context.Context, skuID, warehouseID string) (*StockResult, error)
}
```

- [ ] **Step 6: Implement ReserveStockUseCase**

Create `services/inventory-service/internal/application/usecases/reserve_stock.go`:

```go
package usecases

import (
    "context"
    "time"

    "github.com/google/uuid"
    "github.com/zapmarket/inventory-service/internal/domain/entities"
    "github.com/zapmarket/inventory-service/internal/domain/repositories"
    "github.com/zapmarket/inventory-service/internal/application/ports"
)

type ReserveStockRequest struct {
    OrderID     string
    SKUID       string
    WarehouseID string
    Quantity    int
}

type ReserveStockResult struct {
    ReservationID string
}

type reserveStockUseCase struct {
    inventory    repositories.InventoryRepository
    reservations repositories.ReservationRepository
    ledger       repositories.LedgerRepository
    cache        ports.StockCache
}

func NewReserveStockUseCase(
    inv repositories.InventoryRepository,
    res repositories.ReservationRepository,
    led repositories.LedgerRepository,
    cache ports.StockCache,
) ReserveStockUseCase {
    return &reserveStockUseCase{inventory: inv, reservations: res, ledger: led, cache: cache}
}

func (uc *reserveStockUseCase) Execute(ctx context.Context, req ReserveStockRequest) (*ReserveStockResult, error) {
    // Use Redis Lua script for atomic check-and-decrement
    ok, err := uc.cache.AtomicDecrement(ctx, req.SKUID, req.WarehouseID, req.Quantity)
    if err != nil || !ok {
        return nil, ErrInsufficientStock
    }

    reservation := &entities.Reservation{
        ID:          uuid.NewString(),
        OrderID:     req.OrderID,
        SKUID:       req.SKUID,
        Qty:         req.Quantity,
        Status:      "RESERVED",
        ExpiresAt:   time.Now().Add(15 * time.Minute),
        CreatedAt:   time.Now(),
    }
    if err := uc.reservations.Create(ctx, reservation); err != nil {
        return nil, err
    }
    return &ReserveStockResult{ReservationID: reservation.ID}, nil
}
```

- [ ] **Step 7: Define ports (infrastructure interfaces from application layer)**

Create `services/inventory-service/internal/application/ports/cache.go`:

```go
package ports

import "context"

type StockCache interface {
    // AtomicDecrement decrements available qty if sufficient stock exists.
    // Returns false (not an error) if stock is insufficient.
    AtomicDecrement(ctx context.Context, skuID, warehouseID string, qty int) (bool, error)
    AtomicIncrement(ctx context.Context, skuID, warehouseID string, qty int) error
}
```

- [ ] **Step 8: Move postgres and redis implementations**

Move (copy, then delete original after tests pass):
- `internal/repository/inventory_repository.go` → `internal/infrastructure/postgres/inventory_repo.go`
- `internal/repository/reservation_repository.go` → `internal/infrastructure/postgres/reservation_repo.go`
- `internal/infrastructure/cache/` → `internal/infrastructure/redis/stock_cache.go`

Update package names and imports in moved files.

- [ ] **Step 9: Move gRPC handler to interfaces layer**

Move `internal/handler/grpc/inventory_grpc_handler.go` → `internal/interfaces/grpc/inventory_server.go`

Handler now calls use case interfaces, not the old service struct:

```go
func (s *InventoryServer) ReserveStock(ctx context.Context, req *inventorypb.ReserveStockRequest) (*inventorypb.ReserveStockResponse, error) {
    result, err := s.reserveUC.Execute(ctx, usecases.ReserveStockRequest{
        OrderID:  req.OrderId,
        SKUID:    req.SkuId,
        Quantity: int(req.Quantity),
    })
    if err != nil {
        return nil, status.Errorf(codes.ResourceExhausted, err.Error())
    }
    return &inventorypb.ReserveStockResponse{
        ReservationId: result.ReservationID,
        Success:       true,
    }, nil
}
```

- [ ] **Step 10: Rebuild main.go**

Rewrite `services/inventory-service/main.go` to wire the new layers:

```go
// Infrastructure
inventoryRepo := postgres.NewInventoryRepository(db)
reservationRepo := postgres.NewReservationRepository(db)
ledgerRepo := postgres.NewLedgerRepository(db)
zoneRepo := postgres.NewZoneRepository(db)
stockCache := redis.NewStockCache(redisClient)

// Use Cases
reserveUC := usecases.NewReserveStockUseCase(inventoryRepo, reservationRepo, ledgerRepo, stockCache)
releaseUC := usecases.NewReleaseStockUseCase(inventoryRepo, reservationRepo, ledgerRepo, stockCache)
deductUC  := usecases.NewDeductStockUseCase(inventoryRepo, reservationRepo, ledgerRepo, stockCache)
addUC     := usecases.NewAddStockUseCase(inventoryRepo, ledgerRepo)
getUC     := usecases.NewGetStockUseCase(inventoryRepo)

// Interfaces
grpcServer := grpchandler.NewInventoryServer(reserveUC, releaseUC, deductUC, addUC, getUC)
```

- [ ] **Step 11: Run full test suite**

```bash
cd services/inventory-service
go test ./... -v
```

Expected: all existing tests pass.

- [ ] **Step 12: Delete old folders**

```bash
rm -rf services/inventory-service/internal/service/
rm -rf services/inventory-service/internal/repository/
rm -rf services/inventory-service/internal/handler/
```

- [ ] **Step 13: Final build and test**

```bash
cd services/inventory-service
go build ./...
go test ./... -v
```

Expected: clean build, all tests pass.

- [ ] **Step 14: Commit**

```bash
git add services/inventory-service/
git commit -m "refactor(inventory): migrate to layered Clean Architecture — domain/application/infrastructure/interfaces"
```

---

## Task 2: Migrate payment-service

Follow the identical pattern as Task 1. Key mappings:

| Old | New |
|-----|-----|
| `internal/service/payment_service.go` | `internal/application/usecases/charge_card.go`, `refund_payment.go`, `get_payment_status.go` |
| `internal/repository/payment_repository.go` | `internal/infrastructure/postgres/payment_repo.go` |
| `internal/infrastructure/gateway/` | `internal/infrastructure/stripe/stripe_gateway.go` |
| `internal/handler/http/` | `internal/interfaces/http/` |
| `internal/handler/grpc/` | `internal/interfaces/grpc/` |

Port interface for Stripe:

```go
// internal/application/ports/payment_gateway.go
type PaymentGateway interface {
    Charge(ctx context.Context, req ChargeRequest) (*ChargeResult, error)
    Refund(ctx context.Context, paymentID string, amountCents int64) (*RefundResult, error)
}
```

- [ ] Follow Steps A–G from the migration pattern above for payment-service
- [ ] Run `go test ./... -v` — all pass
- [ ] Delete old folders
- [ ] `git commit -m "refactor(payment): migrate to layered Clean Architecture"`

---

## Task 3: Migrate order-management-service

Key mappings:

| Old | New |
|-----|-----|
| `internal/service/order_service.go` | `internal/application/usecases/create_order.go`, `cancel_order.go`, `get_order.go` |
| `internal/repository/order_repository.go` | `internal/infrastructure/postgres/order_repo.go` |
| `internal/clients/` | `internal/infrastructure/grpc/` (catalog, inventory, payment clients) |
| `internal/infrastructure/relay/` | `internal/infrastructure/kafka/outbox_relay.go` |
| `internal/handler/http/` | `internal/interfaces/http/` |
| `internal/consumer/` | `internal/interfaces/consumers/` |

Port interfaces:

```go
// internal/application/ports/event_publisher.go
type EventPublisher interface {
    Publish(ctx context.Context, topic, key string, payload []byte) error
}

// internal/application/ports/outbox.go
type OutboxRepository interface {
    Insert(ctx context.Context, aggregateID, eventType string, payload []byte) error
    ListUnpublished(ctx context.Context, limit int) ([]OutboxEntry, error)
    MarkPublished(ctx context.Context, id string) error
}
```

- [ ] Follow Steps A–G from the migration pattern above for order-management-service
- [ ] Run `go test ./... -v` — all pass
- [ ] Delete old folders
- [ ] `git commit -m "refactor(order): migrate to layered Clean Architecture"`

---

## Task 4: Migrate product-catalog-service

Key mappings:

| Old | New |
|-----|-----|
| `internal/service/product_service.go` | `internal/application/usecases/create_product.go`, `get_product.go`, `list_products.go`, `update_product.go` |
| `internal/repository/` | `internal/infrastructure/postgres/` |
| `internal/handler/http/` | `internal/interfaces/http/` |
| `internal/handler/grpc/` | `internal/interfaces/grpc/` |
| `internal/search/` | `internal/infrastructure/typesense/` |

- [ ] Follow Steps A–G from the migration pattern above for product-catalog-service
- [ ] Run `go test ./... -v` — all pass
- [ ] Delete old folders
- [ ] `git commit -m "refactor(catalog): migrate to layered Clean Architecture"`

---

## Task 5: Migrate auth-service (highest risk — do last)

Key mappings:

| Old | New |
|-----|-----|
| `internal/service/auth_service.go` | `internal/application/usecases/login.go`, `register.go`, `refresh_token.go`, `validate_token.go` |
| `internal/service/oauth_service.go` | `internal/application/usecases/oauth_login.go` |
| `internal/service/admin_service.go` | `internal/application/usecases/update_seller_status.go` |
| `internal/repository/` | `internal/infrastructure/postgres/` |
| `internal/infrastructure/redisstore/` | `internal/infrastructure/redis/` |
| `internal/handler/http/` | `internal/interfaces/http/` |
| `internal/handler/grpc/` | `internal/interfaces/grpc/` |
| `internal/email/` | `internal/infrastructure/email/` |
| `internal/sms/` | `internal/infrastructure/sms/` |

Port interfaces:

```go
// internal/application/ports/emailer.go
type Emailer interface {
    SendPasswordReset(ctx context.Context, to, link string) error
    SendOTP(ctx context.Context, to string, otp int) error
    SendWelcome(ctx context.Context, to, name string) error
}

// internal/application/ports/sms.go
type SMSSender interface {
    SendOTP(ctx context.Context, phone string, otp int) error
}

// internal/application/ports/token_store.go
type TokenBlacklist interface {
    Blacklist(ctx context.Context, token string, ttl time.Duration) error
    IsBlacklisted(ctx context.Context, token string) (bool, error)
}
```

- [ ] Run full regression: `go test ./... -v`
- [ ] Smoke test login → refresh → logout → validate (blacklisted) flow end-to-end
- [ ] Delete old folders
- [ ] `git commit -m "refactor(auth): migrate to layered Clean Architecture"`

---

## Task 6: Seller Bulk CSV Import

**Files:**
- Create: `services/import-service/` (new microservice, port 8090)
- Mirrors the previously planned async import design (memory: `project_async_csv_import_plan.md`)

### Key Design

```
POST /v1/import/products (multipart CSV upload)
  → Upload CSV to MinIO
  → Create import_job row (status=PENDING)
  → Return job_id

GET /v1/import/jobs/:id
  → Return job status, rows_processed, errors

Background worker:
  → Polls PENDING jobs
  → Reads CSV from MinIO row by row
  → Calls product-catalog-service POST /v1/categories/bulk in batches of 100
  → Updates job progress
  → Sets status=COMPLETE or FAILED
```

- [ ] **Step 1: Write migration**

Create `services/import-service/migrations/0001_init.up.sql`:

```sql
CREATE TABLE import_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id       UUID NOT NULL,
    file_key        TEXT NOT NULL,  -- MinIO object key
    status          VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    total_rows      INT,
    rows_processed  INT NOT NULL DEFAULT 0,
    rows_failed     INT NOT NULL DEFAULT 0,
    error_file_key  TEXT,           -- MinIO key of error report CSV
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

- [ ] **Step 2: Write failing test for import worker**

Create `services/import-service/internal/application/usecases/process_import_test.go`:

```go
package usecases_test

import (
    "context"
    "strings"
    "testing"

    "github.com/zapmarket/import-service/internal/application/usecases"
)

type fakeCatalogClient struct{ created int }

func (f *fakeCatalogClient) BulkCreateProducts(_ context.Context, rows []usecases.ProductRow) (int, error) {
    f.created += len(rows)
    return len(rows), nil
}

type fakeJobRepo struct{ status string }

func (f *fakeJobRepo) UpdateStatus(_ context.Context, id, status string, processed, failed int) error {
    f.status = status
    return nil
}

func TestProcessImport_BatchesRows(t *testing.T) {
    catalog := &fakeCatalogClient{}
    jobs := &fakeJobRepo{}
    uc := usecases.NewProcessImportUseCase(catalog, jobs, 100)

    csv := "name,sku_code,price,category\n" +
           "Widget A,SKU001,999,Electronics\n" +
           "Widget B,SKU002,1499,Electronics\n"

    if err := uc.Execute(context.Background(), "job-1", strings.NewReader(csv)); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if catalog.created != 2 {
        t.Fatalf("expected 2 products created, got %d", catalog.created)
    }
    if jobs.status != "COMPLETE" {
        t.Fatalf("expected COMPLETE status, got %q", jobs.status)
    }
}
```

- [ ] **Step 3: Implement ProcessImportUseCase**

Create `services/import-service/internal/application/usecases/process_import.go`:

```go
package usecases

import (
    "context"
    "encoding/csv"
    "io"
)

type ProductRow struct {
    Name     string
    SKUCode  string
    Price    string
    Category string
}

type catalogClient interface {
    BulkCreateProducts(ctx context.Context, rows []ProductRow) (created int, err error)
}

type jobUpdater interface {
    UpdateStatus(ctx context.Context, jobID, status string, processed, failed int) error
}

type ProcessImportUseCase struct {
    catalog   catalogClient
    jobs      jobUpdater
    batchSize int
}

func NewProcessImportUseCase(catalog catalogClient, jobs jobUpdater, batchSize int) *ProcessImportUseCase {
    return &ProcessImportUseCase{catalog: catalog, jobs: jobs, batchSize: batchSize}
}

func (uc *ProcessImportUseCase) Execute(ctx context.Context, jobID string, r io.Reader) error {
    reader := csv.NewReader(r)
    reader.Read() // skip header

    var batch []ProductRow
    total, failed := 0, 0

    flush := func() {
        if len(batch) == 0 {
            return
        }
        created, _ := uc.catalog.BulkCreateProducts(ctx, batch)
        total += created
        failed += len(batch) - created
        batch = batch[:0]
        uc.jobs.UpdateStatus(ctx, jobID, "IN_PROGRESS", total, failed)
    }

    for {
        record, err := reader.Read()
        if err == io.EOF {
            break
        }
        if err != nil || len(record) < 4 {
            failed++
            continue
        }
        batch = append(batch, ProductRow{Name: record[0], SKUCode: record[1], Price: record[2], Category: record[3]})
        if len(batch) >= uc.batchSize {
            flush()
        }
    }
    flush()

    status := "COMPLETE"
    if failed > 0 && total == 0 {
        status = "FAILED"
    }
    return uc.jobs.UpdateStatus(ctx, jobID, status, total, failed)
}
```

- [ ] **Step 4: Run tests**

```bash
cd services/import-service
go test ./internal/application/... -v
```

Expected: `PASS`

- [ ] **Step 5: Commit**

```bash
git add services/import-service/
git commit -m "feat(import): bulk CSV import service, 100-row batches to catalog, job progress tracking"
```

---

## Verification Checklist

After all migration tasks:

- [ ] For each migrated service: `go test ./... -v` passes
- [ ] For each migrated service: `go build ./...` clean
- [ ] Domain layer has no infra imports: `grep -r '"database/sql"' services/*/internal/domain/` → empty
- [ ] All HTTP routes and gRPC endpoints respond identically to pre-migration (use Pact contract tests from Phase 5)
- [ ] End-to-end checkout flow works after all migrations
- [ ] Seller uploads CSV → import job created → processed → products appear in catalog
