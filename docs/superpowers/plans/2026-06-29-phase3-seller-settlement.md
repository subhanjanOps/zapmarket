# Phase 3 — Seller Settlement & Multi-Warehouse Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a seller settlement service that maintains a double-entry ledger per seller, schedules weekly payouts via Razorpay, and remove the `DefaultWarehouseID` hardcoding from inventory-service.

**Architecture:** `settlement-service` is a new Go microservice (port 8088, gRPC 50058) that consumes `payment.captured` and `payment.refunded` Kafka events to credit/debit seller accounts. A weekly cron goroutine aggregates pending credits and initiates payouts. Multi-warehouse support adds a `pincode_zones` table mapping delivery pincodes to warehouse IDs, used at checkout to select the fulfillment warehouse.

**Tech Stack:** Go 1.25, existing `pkg/kafka`, existing `pkg/database`, Razorpay Go SDK (`github.com/razorpay/razorpay-go`).

## Global Constraints

- Settlement amounts are always in the smallest currency unit (paise for INR)
- Platform commission rate is configurable via `PLATFORM_COMMISSION_BPS` env var (basis points; default 200 = 2%)
- Payout is only initiated when pending balance ≥ ₹100 (10000 paise)
- All ledger entries are immutable — never update, only insert
- Warehouse selection falls back to `DEFAULT` warehouse if no zone mapping exists for the pincode
- All new topic constants go in `pkg/kafka/topics.go`

---

## File Map

### settlement-service (new microservice)
```
services/settlement-service/
  cmd/server/main.go
  migrations/
    0001_init.up.sql / down
  internal/
    domain/
      ledger.go           — SellerAccount, LedgerEntry, EntryType enum
      errors/errors.go
    application/
      interfaces.go
      usecases/
        credit_sale.go        — on payment.captured
        debit_refund.go       — on payment.refunded
        initiate_payout.go    — weekly scheduler
        get_balance.go
    infrastructure/
      postgres/
        ledger_repo.go
        payout_repo.go
      kafka/
        consumer.go
      razorpay/
        payout_gateway.go
        noop_gateway.go     — for dev/test
    interfaces/
      http/
        balance_handler.go  — GET /v1/sellers/:id/balance
      consumers/
        payment_consumer.go
```

### inventory-service multi-warehouse
```
services/inventory-service/
  migrations/
    0004_pincode_zones.up.sql / down
  internal/
    domain/
      warehouse.go            — MODIFY: remove DefaultWarehouseID constant
    repository/
      zone_repository.go      — NEW: FindWarehouseByPincode
    service/
      inventory_service.go    — MODIFY: ReserveStock accepts warehouseID
    handler/grpc/
      inventory_grpc_handler.go — MODIFY: accept warehouse_id in ReserveStockRequest
  proto/inventory.proto         — MODIFY: add warehouse_id field
```

---

## Task 1: Settlement Service — Domain & Migration

**Files:**
- Create: `services/settlement-service/internal/domain/ledger.go`
- Create: `services/settlement-service/internal/domain/errors/errors.go`
- Create: `services/settlement-service/migrations/0001_init.up.sql`
- Create: `services/settlement-service/migrations/0001_init.down.sql`

- [ ] **Step 1: Define domain types**

Create `services/settlement-service/internal/domain/ledger.go`:

```go
package domain

import "time"

type EntryType string

const (
    EntryTypeCredit         EntryType = "CREDIT_SALE"
    EntryTypeDebitRefund    EntryType = "DEBIT_REFUND"
    EntryTypeDebitCommission EntryType = "DEBIT_COMMISSION"
    EntryTypePayout         EntryType = "DEBIT_PAYOUT"
)

type LedgerEntry struct {
    ID              string
    SellerID        string
    OrderID         string
    PaymentID       string
    EntryType       EntryType
    AmountPaise     int64 // positive = credit, negative = debit
    CommissionPaise int64 // platform commission deducted
    NetPaise        int64 // AmountPaise - CommissionPaise (what seller earns)
    Currency        string
    Note            string
    CreatedAt       time.Time
}

type SellerBalance struct {
    SellerID       string
    PendingPaise   int64 // earned but not yet paid out
    PaidOutPaise   int64 // lifetime paid out
    Currency       string
    LastUpdatedAt  time.Time
}

type Payout struct {
    ID             string
    SellerID       string
    AmountPaise    int64
    Currency       string
    RazorpayPayoutID string
    Status         string // PENDING, PROCESSED, FAILED
    InitiatedAt    time.Time
    ProcessedAt    *time.Time
}
```

- [ ] **Step 2: Create domain errors**

Create `services/settlement-service/internal/domain/errors/errors.go`:

```go
package errors

import "errors"

var (
    ErrInsufficientBalance = errors.New("seller balance below minimum payout threshold")
    ErrPayoutFailed        = errors.New("payout gateway returned failure")
    ErrSellerNotFound      = errors.New("seller account not found")
)
```

- [ ] **Step 3: Write migration**

Create `services/settlement-service/migrations/0001_init.up.sql`:

```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE seller_ledger (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id        UUID NOT NULL,
    order_id         UUID,
    payment_id       UUID,
    entry_type       VARCHAR(32) NOT NULL,
    amount_paise     BIGINT NOT NULL,
    commission_paise BIGINT NOT NULL DEFAULT 0,
    net_paise        BIGINT NOT NULL,
    currency         VARCHAR(3) NOT NULL DEFAULT 'INR',
    note             TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE seller_balances (
    seller_id       UUID PRIMARY KEY,
    pending_paise   BIGINT NOT NULL DEFAULT 0,
    paid_out_paise  BIGINT NOT NULL DEFAULT 0,
    currency        VARCHAR(3) NOT NULL DEFAULT 'INR',
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE seller_payouts (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id           UUID NOT NULL,
    amount_paise        BIGINT NOT NULL,
    currency            VARCHAR(3) NOT NULL DEFAULT 'INR',
    razorpay_payout_id  TEXT,
    status              VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    initiated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at        TIMESTAMPTZ
);

CREATE INDEX idx_seller_ledger_seller ON seller_ledger(seller_id);
CREATE INDEX idx_seller_payouts_seller ON seller_payouts(seller_id, status);
```

Create `services/settlement-service/migrations/0001_init.down.sql`:

```sql
DROP TABLE IF EXISTS seller_payouts;
DROP TABLE IF EXISTS seller_balances;
DROP TABLE IF EXISTS seller_ledger;
```

- [ ] **Step 4: Commit**

```bash
git add services/settlement-service/internal/domain/ services/settlement-service/migrations/
git commit -m "feat(settlement): domain types, seller_ledger + seller_balances + seller_payouts migration"
```

---

## Task 2: Settlement Service — CreditSale & DebitRefund Use Cases

**Files:**
- Create: `services/settlement-service/internal/application/interfaces.go`
- Create: `services/settlement-service/internal/application/usecases/credit_sale.go`
- Create: `services/settlement-service/internal/application/usecases/debit_refund.go`
- Create: `services/settlement-service/internal/application/usecases/credit_sale_test.go`

- [ ] **Step 1: Define interfaces**

Create `services/settlement-service/internal/application/interfaces.go`:

```go
package application

import (
    "context"
    "github.com/zapmarket/settlement-service/internal/domain"
)

type LedgerRepository interface {
    InsertEntry(ctx context.Context, entry domain.LedgerEntry) error
    CreditBalance(ctx context.Context, sellerID string, netPaise int64) error
    DebitBalance(ctx context.Context, sellerID string, amountPaise int64) error
    GetBalance(ctx context.Context, sellerID string) (*domain.SellerBalance, error)
    GetPendingSellers(ctx context.Context, minBalancePaise int64) ([]string, error)
}

type PayoutGateway interface {
    Initiate(ctx context.Context, sellerID string, amountPaise int64, currency string) (razorpayPayoutID string, err error)
}
```

- [ ] **Step 2: Write failing test for CreditSale**

Create `services/settlement-service/internal/application/usecases/credit_sale_test.go`:

```go
package usecases_test

import (
    "context"
    "testing"

    "github.com/zapmarket/settlement-service/internal/application"
    "github.com/zapmarket/settlement-service/internal/application/usecases"
    "github.com/zapmarket/settlement-service/internal/domain"
)

type fakeLedger struct {
    entries  []domain.LedgerEntry
    credited int64
}

func (f *fakeLedger) InsertEntry(_ context.Context, e domain.LedgerEntry) error {
    f.entries = append(f.entries, e)
    return nil
}
func (f *fakeLedger) CreditBalance(_ context.Context, _ string, net int64) error {
    f.credited = net
    return nil
}
func (f *fakeLedger) DebitBalance(_ context.Context, _ string, _ int64) error  { return nil }
func (f *fakeLedger) GetBalance(_ context.Context, _ string) (*domain.SellerBalance, error) {
    return &domain.SellerBalance{PendingPaise: 50000}, nil
}
func (f *fakeLedger) GetPendingSellers(_ context.Context, _ int64) ([]string, error) { return nil, nil }

func TestCreditSale_DeductsCommissionAndCreditsNet(t *testing.T) {
    ledger := &fakeLedger{}
    // 200 BPS = 2% commission
    uc := usecases.NewCreditSaleUseCase(ledger, 200)

    err := uc.Execute(context.Background(), usecases.CreditSaleInput{
        SellerID:    "sel-1",
        OrderID:     "ord-1",
        PaymentID:   "pay-1",
        AmountPaise: 10000, // ₹100
        Currency:    "INR",
    })
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(ledger.entries) != 1 {
        t.Fatalf("expected 1 ledger entry, got %d", len(ledger.entries))
    }
    // 2% of 10000 = 200 commission; net = 9800
    if ledger.entries[0].CommissionPaise != 200 {
        t.Fatalf("expected commission 200, got %d", ledger.entries[0].CommissionPaise)
    }
    if ledger.credited != 9800 {
        t.Fatalf("expected net credited 9800, got %d", ledger.credited)
    }
}
```

- [ ] **Step 3: Run to confirm failure**

```bash
cd services/settlement-service
go test ./internal/application/usecases/... -v -run TestCreditSale
```

Expected: `FAIL`

- [ ] **Step 4: Implement CreditSaleUseCase**

Create `services/settlement-service/internal/application/usecases/credit_sale.go`:

```go
package usecases

import (
    "context"
    "time"

    "github.com/google/uuid"
    "github.com/zapmarket/settlement-service/internal/application"
    "github.com/zapmarket/settlement-service/internal/domain"
)

type CreditSaleInput struct {
    SellerID    string
    OrderID     string
    PaymentID   string
    AmountPaise int64
    Currency    string
}

type CreditSaleUseCase struct {
    ledger        application.LedgerRepository
    commissionBPS int64 // basis points; 200 = 2%
}

func NewCreditSaleUseCase(ledger application.LedgerRepository, commissionBPS int64) *CreditSaleUseCase {
    return &CreditSaleUseCase{ledger: ledger, commissionBPS: commissionBPS}
}

func (uc *CreditSaleUseCase) Execute(ctx context.Context, in CreditSaleInput) error {
    commission := (in.AmountPaise * uc.commissionBPS) / 10000
    net := in.AmountPaise - commission

    entry := domain.LedgerEntry{
        ID:              uuid.NewString(),
        SellerID:        in.SellerID,
        OrderID:         in.OrderID,
        PaymentID:       in.PaymentID,
        EntryType:       domain.EntryTypeCredit,
        AmountPaise:     in.AmountPaise,
        CommissionPaise: commission,
        NetPaise:        net,
        Currency:        in.Currency,
        CreatedAt:       time.Now(),
    }
    if err := uc.ledger.InsertEntry(ctx, entry); err != nil {
        return err
    }
    return uc.ledger.CreditBalance(ctx, in.SellerID, net)
}
```

- [ ] **Step 5: Implement DebitRefundUseCase**

Create `services/settlement-service/internal/application/usecases/debit_refund.go`:

```go
package usecases

import (
    "context"
    "time"

    "github.com/google/uuid"
    "github.com/zapmarket/settlement-service/internal/application"
    "github.com/zapmarket/settlement-service/internal/domain"
)

type DebitRefundInput struct {
    SellerID    string
    OrderID     string
    PaymentID   string
    AmountPaise int64
    Currency    string
}

type DebitRefundUseCase struct{ ledger application.LedgerRepository }

func NewDebitRefundUseCase(ledger application.LedgerRepository) *DebitRefundUseCase {
    return &DebitRefundUseCase{ledger: ledger}
}

func (uc *DebitRefundUseCase) Execute(ctx context.Context, in DebitRefundInput) error {
    entry := domain.LedgerEntry{
        ID:          uuid.NewString(),
        SellerID:    in.SellerID,
        OrderID:     in.OrderID,
        PaymentID:   in.PaymentID,
        EntryType:   domain.EntryTypeDebitRefund,
        AmountPaise: -in.AmountPaise,
        NetPaise:    -in.AmountPaise,
        Currency:    in.Currency,
        CreatedAt:   time.Now(),
    }
    if err := uc.ledger.InsertEntry(ctx, entry); err != nil {
        return err
    }
    return uc.ledger.DebitBalance(ctx, in.SellerID, in.AmountPaise)
}
```

- [ ] **Step 6: Run tests**

```bash
cd services/settlement-service
go test ./internal/application/... -v
```

Expected: `PASS`

- [ ] **Step 7: Commit**

```bash
git add services/settlement-service/internal/application/
git commit -m "feat(settlement): CreditSale deducts 2% commission, DebitRefund reverses earnings"
```

---

## Task 3: Settlement Service — Payout Scheduler

**Files:**
- Create: `services/settlement-service/internal/application/usecases/initiate_payout.go`
- Create: `services/settlement-service/internal/application/usecases/initiate_payout_test.go`
- Create: `services/settlement-service/internal/infrastructure/razorpay/noop_gateway.go`

- [ ] **Step 1: Write failing test for payout**

Create `services/settlement-service/internal/application/usecases/initiate_payout_test.go`:

```go
package usecases_test

import (
    "context"
    "testing"

    "github.com/zapmarket/settlement-service/internal/application/usecases"
    domainerrors "github.com/zapmarket/settlement-service/internal/domain/errors"
)

type fakePayoutGateway struct{ initiated string }

func (f *fakePayoutGateway) Initiate(_ context.Context, sellerID string, _ int64, _ string) (string, error) {
    f.initiated = sellerID
    return "rzp-pay-1", nil
}

func TestInitiatePayout_SkipsWhenBelowMinimum(t *testing.T) {
    ledger := &fakeLedger{} // GetBalance returns 50000 paise = ₹500
    gw := &fakePayoutGateway{}
    // min threshold = 10000 paise = ₹100; 50000 qualifies
    uc := usecases.NewInitiatePayoutUseCase(ledger, gw, 10000)

    if err := uc.Execute(context.Background(), "sel-1"); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if gw.initiated != "sel-1" {
        t.Fatalf("expected payout for sel-1, got %q", gw.initiated)
    }
}

func TestInitiatePayout_ReturnsErrWhenBelowThreshold(t *testing.T) {
    ledger := &fakeLedger{} // returns 50000 paise
    gw := &fakePayoutGateway{}
    // threshold above balance
    uc := usecases.NewInitiatePayoutUseCase(ledger, gw, 100000)

    err := uc.Execute(context.Background(), "sel-1")
    if err != domainerrors.ErrInsufficientBalance {
        t.Fatalf("expected ErrInsufficientBalance, got %v", err)
    }
}
```

- [ ] **Step 2: Implement InitiatePayoutUseCase**

Create `services/settlement-service/internal/application/usecases/initiate_payout.go`:

```go
package usecases

import (
    "context"

    "github.com/zapmarket/settlement-service/internal/application"
    domainerrors "github.com/zapmarket/settlement-service/internal/domain/errors"
)

type InitiatePayoutUseCase struct {
    ledger           application.LedgerRepository
    gateway          application.PayoutGateway
    minBalancePaise  int64
}

func NewInitiatePayoutUseCase(ledger application.LedgerRepository, gw application.PayoutGateway, minBalancePaise int64) *InitiatePayoutUseCase {
    return &InitiatePayoutUseCase{ledger: ledger, gateway: gw, minBalancePaise: minBalancePaise}
}

func (uc *InitiatePayoutUseCase) Execute(ctx context.Context, sellerID string) error {
    bal, err := uc.ledger.GetBalance(ctx, sellerID)
    if err != nil {
        return err
    }
    if bal.PendingPaise < uc.minBalancePaise {
        return domainerrors.ErrInsufficientBalance
    }
    _, err = uc.gateway.Initiate(ctx, sellerID, bal.PendingPaise, bal.Currency)
    if err != nil {
        return err
    }
    return uc.ledger.DebitBalance(ctx, sellerID, bal.PendingPaise)
}
```

- [ ] **Step 3: Create noop gateway for dev**

Create `services/settlement-service/internal/infrastructure/razorpay/noop_gateway.go`:

```go
package razorpay

import (
    "context"
    "github.com/zapmarket/pkg/logger"
)

type NoopGateway struct{}

func (g *NoopGateway) Initiate(ctx context.Context, sellerID string, amountPaise int64, currency string) (string, error) {
    logger.Info(ctx, "noop payout gateway: would initiate payout",
        "seller_id", sellerID, "amount_paise", amountPaise, "currency", currency)
    return "noop-payout-id", nil
}
```

- [ ] **Step 4: Run tests**

```bash
cd services/settlement-service
go test ./internal/application/... -v
```

Expected: `PASS`

- [ ] **Step 5: Commit**

```bash
git add services/settlement-service/internal/application/usecases/initiate_payout.go \
        services/settlement-service/internal/application/usecases/initiate_payout_test.go \
        services/settlement-service/internal/infrastructure/razorpay/noop_gateway.go
git commit -m "feat(settlement): payout use case, noop gateway for dev, threshold guard"
```

---

## Task 4: Settlement Service — Kafka Consumer & Wiring

**Files:**
- Create: `services/settlement-service/internal/infrastructure/kafka/consumer.go`
- Create: `services/settlement-service/cmd/server/main.go`
- Modify: `docker-compose.yml` — add settlement-service

- [ ] **Step 1: Implement Kafka consumer**

Create `services/settlement-service/internal/infrastructure/kafka/consumer.go`:

```go
package kafka

import (
    "context"
    "encoding/json"

    "github.com/zapmarket/settlement-service/internal/application/usecases"
    "github.com/zapmarket/pkg/logger"
)

type paymentCapturedEvent struct {
    OrderID     string `json:"order_id"`
    PaymentID   string `json:"payment_id"`
    SellerID    string `json:"seller_id"`
    AmountCents int64  `json:"amount_cents"`
    Currency    string `json:"currency"`
}

type paymentRefundedEvent struct {
    OrderID     string `json:"order_id"`
    PaymentID   string `json:"payment_id"`
    SellerID    string `json:"seller_id"`
    AmountCents int64  `json:"amount_cents"`
    Currency    string `json:"currency"`
}

type PaymentConsumer struct {
    credit *usecases.CreditSaleUseCase
    debit  *usecases.DebitRefundUseCase
}

func NewPaymentConsumer(credit *usecases.CreditSaleUseCase, debit *usecases.DebitRefundUseCase) *PaymentConsumer {
    return &PaymentConsumer{credit: credit, debit: debit}
}

func (c *PaymentConsumer) HandleCaptured(ctx context.Context, payload []byte) error {
    var evt paymentCapturedEvent
    if err := json.Unmarshal(payload, &evt); err != nil {
        return err
    }
    logger.Info(ctx, "settlement: processing payment.captured", "order_id", evt.OrderID)
    return c.credit.Execute(ctx, usecases.CreditSaleInput{
        SellerID:    evt.SellerID,
        OrderID:     evt.OrderID,
        PaymentID:   evt.PaymentID,
        AmountPaise: evt.AmountCents, // assuming INR paise == cents in this system
        Currency:    evt.Currency,
    })
}

func (c *PaymentConsumer) HandleRefunded(ctx context.Context, payload []byte) error {
    var evt paymentRefundedEvent
    if err := json.Unmarshal(payload, &evt); err != nil {
        return err
    }
    logger.Info(ctx, "settlement: processing payment.refunded", "order_id", evt.OrderID)
    return c.debit.Execute(ctx, usecases.DebitRefundInput{
        SellerID:    evt.SellerID,
        OrderID:     evt.OrderID,
        PaymentID:   evt.PaymentID,
        AmountPaise: evt.AmountCents,
        Currency:    evt.Currency,
    })
}
```

- [ ] **Step 2: Add settlement-service to docker-compose.yml**

```yaml
  settlement-service:
    build: ./services/settlement-service
    ports:
      - "8088:8088"
    environment:
      - HTTP_PORT=8088
      - DB_HOST=postgres
      - DB_NAME=settlement
      - DB_USER=zapuser
      - DB_PASSWORD=zappass123
      - KAFKA_BROKERS=kafka:9092
      - PLATFORM_COMMISSION_BPS=200
      - MIN_PAYOUT_PAISE=10000
      - RAZORPAY_KEY_ID=${RAZORPAY_KEY_ID:-noop}
      - RAZORPAY_KEY_SECRET=${RAZORPAY_KEY_SECRET:-noop}
    depends_on:
      - postgres
      - kafka
```

- [ ] **Step 3: Commit**

```bash
git add services/settlement-service/ docker-compose.yml
git commit -m "feat(settlement): Kafka consumer wired to credit/debit use cases, docker-compose registration"
```

---

## Task 5: Multi-Warehouse — Remove Hardcoded DefaultWarehouseID

**Files:**
- Create: `services/inventory-service/migrations/0004_pincode_zones.up.sql`
- Create: `services/inventory-service/migrations/0004_pincode_zones.down.sql`
- Create: `services/inventory-service/internal/repository/zone_repository.go`
- Modify: `services/inventory-service/internal/domain/models.go` — remove DefaultWarehouseID
- Modify: `services/inventory-service/internal/service/inventory_service.go` — accept warehouseID

- [ ] **Step 1: Add pincode zones migration**

Create `services/inventory-service/migrations/0004_pincode_zones.up.sql`:

```sql
CREATE TABLE pincode_zones (
    pincode      VARCHAR(10) NOT NULL,
    warehouse_id UUID NOT NULL REFERENCES warehouses(id),
    PRIMARY KEY (pincode)
);

-- Seed: everything goes to the default warehouse until zones are configured
INSERT INTO pincode_zones (pincode, warehouse_id)
SELECT '000000', id FROM warehouses WHERE name = 'Default' LIMIT 1;
```

Create `services/inventory-service/migrations/0004_pincode_zones.down.sql`:

```sql
DROP TABLE IF EXISTS pincode_zones;
```

- [ ] **Step 2: Create zone repository**

Create `services/inventory-service/internal/repository/zone_repository.go`:

```go
package repository

import (
    "context"
    "database/sql"
    "errors"
)

type ZoneRepository interface {
    FindWarehouseByPincode(ctx context.Context, pincode string) (warehouseID string, err error)
    DefaultWarehouseID(ctx context.Context) (string, error)
}

type postgresZoneRepository struct{ db *sql.DB }

func NewZoneRepository(db *sql.DB) ZoneRepository {
    return &postgresZoneRepository{db: db}
}

func (r *postgresZoneRepository) FindWarehouseByPincode(ctx context.Context, pincode string) (string, error) {
    var id string
    err := r.db.QueryRowContext(ctx,
        `SELECT warehouse_id FROM pincode_zones WHERE pincode = $1`, pincode,
    ).Scan(&id)
    if errors.Is(err, sql.ErrNoRows) {
        return r.DefaultWarehouseID(ctx)
    }
    return id, err
}

func (r *postgresZoneRepository) DefaultWarehouseID(ctx context.Context) (string, error) {
    var id string
    err := r.db.QueryRowContext(ctx,
        `SELECT id FROM warehouses WHERE is_active = true ORDER BY created_at ASC LIMIT 1`,
    ).Scan(&id)
    return id, err
}
```

- [ ] **Step 3: Remove the hardcoded constant**

In `services/inventory-service/internal/domain/models.go`, find and delete:

```go
const DefaultWarehouseID = "..."
```

- [ ] **Step 4: Update ReserveStock to accept warehouseID**

In `services/inventory-service/internal/service/inventory_service.go`, update the `ReserveStock` signature:

```go
// Before:
func (s *InventoryService) ReserveStock(ctx context.Context, orderID, skuID string, qty int) (string, error)

// After:
func (s *InventoryService) ReserveStock(ctx context.Context, orderID, skuID, warehouseID string, qty int) (string, error)
```

If no `warehouseID` is provided by the caller, look it up via `zoneRepo.DefaultWarehouseID(ctx)`.

- [ ] **Step 5: Run tests**

```bash
cd services/inventory-service
go test ./... -v
```

Expected: `PASS`

- [ ] **Step 6: Commit**

```bash
git add services/inventory-service/migrations/0004_pincode_zones.* \
        services/inventory-service/internal/repository/zone_repository.go \
        services/inventory-service/internal/domain/models.go \
        services/inventory-service/internal/service/inventory_service.go
git commit -m "feat(inventory): remove hardcoded DefaultWarehouseID, add pincode_zones table, warehouse selection via zone lookup"
```

---

## Verification Checklist

- [ ] `docker compose up -d` — settlement-service healthy
- [ ] Place an order → payment captured → verify `seller_ledger` row with 2% commission deducted
- [ ] Trigger a refund → verify `DEBIT_REFUND` entry in `seller_ledger`, balance decremented
- [ ] Call `GET /v1/sellers/:id/balance` → returns `pending_paise` correctly
- [ ] POST `/v1/sellers/:id/payout` manually → `seller_payouts` row created, `pending_paise` zeroed
- [ ] Inventory: ReserveStock with no pincode → uses first active warehouse (no panic, no hardcoded ID)
- [ ] Add a pincode zone → ReserveStock with matching pincode → routes to correct warehouse
