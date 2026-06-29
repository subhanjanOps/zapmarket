# Phase 1 — Critical Hotfixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix three production-safety issues: the reservation expiry worker, the API gateway circuit breaker, and the synchronous checkout saga.

**Architecture:** G6 adds a background goroutine in inventory-service. G8 wraps each proxy target in a circuit breaker using `gobreaker`. G1 converts the synchronous gRPC chain in order-management-service into a choreography-based saga over Kafka — order-management publishes events, inventory and payment consume and respond with result events, order-management confirms or compensates.

**Tech Stack:** Go 1.25, `sony/gobreaker` v0.1.1, existing `pkg/kafka`, existing `pkg/relay`, existing gRPC clients retained as fallback for the transition period.

## Global Constraints

- All Go code lives under the service's `internal/` package — no new top-level packages
- Kafka topic names are constants in `pkg/kafka/topics.go` — add new ones there, never inline strings
- All new DB migrations follow the existing naming: `NNNN_description.up.sql` / `NNNN_description.down.sql`
- Use constructor injection — no globals or singletons
- Structured logging via `pkg/logger` — never `fmt.Println`
- Every new exported type gets a corresponding test
- Commit after every task

---

## File Map

### Task 1 — Reservation Expiry Worker (G6)
```
services/inventory-service/internal/worker/
  expiry_worker.go          NEW — ticker loop, queries expired reservations, calls release
  expiry_worker_test.go     NEW — fake repo, verifies release called for expired rows

services/inventory-service/internal/repository/
  reservation_repository.go MODIFY — add FindExpired(ctx, now time.Time) method

services/inventory-service/main.go
  MODIFY — start expiry worker goroutine, stop on shutdown signal
```

### Task 2 — API Gateway Circuit Breaker (G8)
```
services/api-gateway/go.mod
  MODIFY — add sony/gobreaker v0.1.1

services/api-gateway/internal/proxy/
  breaker.go                NEW — wraps http.RoundTripper with gobreaker per upstream host
  breaker_test.go           NEW — verifies open/half-open/close transitions

services/api-gateway/internal/proxy/proxy.go
  MODIFY — inject breaker-wrapped transport into each reverse proxy
```

### Task 3 — Checkout Saga: Inventory Side (G1)
```
pkg/kafka/topics.go
  MODIFY — add CheckoutRequested, InventoryReserved, InventoryReservationFailed constants

services/inventory-service/internal/consumer/
  checkout_consumer.go      NEW — Kafka consumer for checkout.requested events
  checkout_consumer_test.go NEW — fake service, verifies reserve called then result published

services/inventory-service/internal/domain/events/
  events.go                 NEW — CheckoutRequestedEvent, InventoryReservedEvent structs

services/inventory-service/main.go
  MODIFY — start checkout_consumer goroutine
```

### Task 4 — Checkout Saga: Payment Side (G1)
```
services/payment-service/internal/consumer/
  inventory_consumer.go      NEW — Kafka consumer for inventory.reserved events
  inventory_consumer_test.go NEW — fake gateway, verifies charge called then result published

services/payment-service/internal/domain/events/
  events.go                  NEW — InventoryReservedEvent, PaymentCapturedEvent, PaymentFailedEvent

services/payment-service/main.go
  MODIFY — start inventory_consumer goroutine
```

### Task 5 — Checkout Saga: Order Management Orchestrator (G1)
```
services/order-management-service/internal/consumer/
  saga_consumer.go           NEW — consumes inventory.reserved, payment.captured, *.failed
  saga_consumer_test.go      NEW — state machine transitions, compensation paths

services/order-management-service/internal/service/order_service.go
  MODIFY — CreateOrder publishes checkout.requested instead of calling gRPC directly
  MODIFY — add ConfirmOrder, CompensateOrder use cases

services/order-management-service/migrations/
  0003_saga_state.up.sql     NEW — add saga_status column to orders table
  0003_saga_state.down.sql   NEW

services/order-management-service/main.go
  MODIFY — start saga_consumer goroutine
```

---

## Task 1: Reservation Expiry Worker

**Files:**
- Create: `services/inventory-service/internal/worker/expiry_worker.go`
- Create: `services/inventory-service/internal/worker/expiry_worker_test.go`
- Modify: `services/inventory-service/internal/repository/reservation_repository.go`
- Modify: `services/inventory-service/main.go`

**Interfaces:**
- Consumes: `ReservationRepository` (existing interface in `internal/repository/`)
- Produces: `ExpiryWorker` struct with `Start(ctx context.Context)` and `Stop()` methods

- [ ] **Step 1: Add `FindExpired` to the reservation repository interface**

Open `services/inventory-service/internal/repository/reservation_repository.go`.
Add to the `ReservationRepository` interface:

```go
FindExpired(ctx context.Context, before time.Time) ([]*domain.Reservation, error)
```

Add the implementation on the postgres struct:

```go
func (r *postgresReservationRepository) FindExpired(ctx context.Context, before time.Time) ([]*domain.Reservation, error) {
    rows, err := r.db.QueryContext(ctx, `
        SELECT id, inventory_id, order_id, sku_id, qty, status, expires_at, created_at
        FROM reservations
        WHERE status = 'RESERVED' AND expires_at < $1
    `, before)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var out []*domain.Reservation
    for rows.Next() {
        var res domain.Reservation
        if err := rows.Scan(&res.ID, &res.InventoryID, &res.OrderID, &res.SKUID, &res.Qty, &res.Status, &res.ExpiresAt, &res.CreatedAt); err != nil {
            return nil, err
        }
        out = append(out, &res)
    }
    return out, rows.Err()
}
```

- [ ] **Step 2: Write the failing test for the expiry worker**

Create `services/inventory-service/internal/worker/expiry_worker_test.go`:

```go
package worker_test

import (
    "context"
    "testing"
    "time"

    "github.com/zapmarket/inventory-service/internal/domain"
    "github.com/zapmarket/inventory-service/internal/worker"
)

type fakeReservationRepo struct {
    expired  []*domain.Reservation
    released []string
}

func (f *fakeReservationRepo) FindExpired(_ context.Context, _ time.Time) ([]*domain.Reservation, error) {
    return f.expired, nil
}

func (f *fakeReservationRepo) Release(_ context.Context, id string) error {
    f.released = append(f.released, id)
    return nil
}

func TestExpiryWorker_ReleasesExpiredReservations(t *testing.T) {
    repo := &fakeReservationRepo{
        expired: []*domain.Reservation{
            {ID: "res-1", OrderID: "ord-1", Qty: 2},
            {ID: "res-2", OrderID: "ord-2", Qty: 1},
        },
    }

    w := worker.NewExpiryWorker(repo, 50*time.Millisecond)
    ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
    defer cancel()
    w.Start(ctx)

    if len(repo.released) != 2 {
        t.Fatalf("expected 2 releases, got %d", len(repo.released))
    }
    if repo.released[0] != "res-1" || repo.released[1] != "res-2" {
        t.Fatalf("unexpected released IDs: %v", repo.released)
    }
}
```

- [ ] **Step 3: Run the test to confirm it fails**

```bash
cd services/inventory-service
go test ./internal/worker/... -v -run TestExpiryWorker_ReleasesExpiredReservations
```

Expected: `FAIL — cannot find package "worker"`

- [ ] **Step 4: Implement the expiry worker**

Create `services/inventory-service/internal/worker/expiry_worker.go`:

```go
package worker

import (
    "context"
    "time"

    "github.com/zapmarket/inventory-service/internal/domain"
    "github.com/zapmarket/pkg/logger"
)

type reservationReleaser interface {
    FindExpired(ctx context.Context, before time.Time) ([]*domain.Reservation, error)
    Release(ctx context.Context, id string) error
}

type ExpiryWorker struct {
    repo     reservationReleaser
    interval time.Duration
}

func NewExpiryWorker(repo reservationReleaser, interval time.Duration) *ExpiryWorker {
    return &ExpiryWorker{repo: repo, interval: interval}
}

func (w *ExpiryWorker) Start(ctx context.Context) {
    ticker := time.NewTicker(w.interval)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case t := <-ticker.C:
            w.runOnce(ctx, t)
        }
    }
}

func (w *ExpiryWorker) runOnce(ctx context.Context, now time.Time) {
    expired, err := w.repo.FindExpired(ctx, now)
    if err != nil {
        logger.Error(ctx, "expiry worker: find expired failed", "error", err)
        return
    }
    for _, res := range expired {
        if err := w.repo.Release(ctx, res.ID); err != nil {
            logger.Error(ctx, "expiry worker: release failed", "reservation_id", res.ID, "error", err)
            continue
        }
        logger.Info(ctx, "expiry worker: released reservation", "reservation_id", res.ID, "order_id", res.OrderID)
    }
}
```

- [ ] **Step 5: Run the test to confirm it passes**

```bash
cd services/inventory-service
go test ./internal/worker/... -v -run TestExpiryWorker_ReleasesExpiredReservations
```

Expected: `PASS`

- [ ] **Step 6: Wire the worker into main.go**

In `services/inventory-service/main.go`, after the service is initialized and before `srv.ListenAndServe()`:

```go
expiryWorker := worker.NewExpiryWorker(reservationRepo, 60*time.Second)
go expiryWorker.Start(ctx) // ctx is the root context tied to shutdown signal
```

- [ ] **Step 7: Build and verify no compile errors**

```bash
cd services/inventory-service
go build ./...
```

Expected: no errors

- [ ] **Step 8: Commit**

```bash
git add services/inventory-service/internal/worker/ \
        services/inventory-service/internal/repository/reservation_repository.go \
        services/inventory-service/main.go
git commit -m "feat(inventory): add reservation expiry worker — polls every 60s, releases expired reservations"
```

---

## Task 2: API Gateway Circuit Breaker

**Files:**
- Modify: `services/api-gateway/go.mod`
- Create: `services/api-gateway/internal/proxy/breaker.go`
- Create: `services/api-gateway/internal/proxy/breaker_test.go`
- Modify: `services/api-gateway/internal/proxy/proxy.go`

**Interfaces:**
- Consumes: `http.RoundTripper` (standard library)
- Produces: `NewBreakerTransport(name string, inner http.RoundTripper) http.RoundTripper`

- [ ] **Step 1: Add gobreaker dependency**

```bash
cd services/api-gateway
go get github.com/sony/gobreaker@v0.1.1
go mod tidy
```

Expected: `go.mod` updated with `github.com/sony/gobreaker v0.1.1`

- [ ] **Step 2: Write the failing test**

Create `services/api-gateway/internal/proxy/breaker_test.go`:

```go
package proxy_test

import (
    "errors"
    "net/http"
    "testing"

    "github.com/zapmarket/api-gateway/internal/proxy"
)

type failingTransport struct{ calls int }

func (f *failingTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
    f.calls++
    return nil, errors.New("upstream down")
}

func TestBreakerTransport_OpensAfterThreshold(t *testing.T) {
    inner := &failingTransport{}
    transport := proxy.NewBreakerTransport("test-upstream", inner)

    req, _ := http.NewRequest("GET", "http://example.com", nil)

    // Trip the breaker (default threshold: 5 consecutive failures)
    for i := 0; i < 5; i++ {
        _, _ = transport.RoundTrip(req)
    }

    _, err := transport.RoundTrip(req)
    if err == nil {
        t.Fatal("expected circuit breaker error, got nil")
    }
    if inner.calls > 5 {
        t.Fatalf("breaker should be open after 5 failures, inner called %d times", inner.calls)
    }
}
```

- [ ] **Step 3: Run the test to confirm it fails**

```bash
cd services/api-gateway
go test ./internal/proxy/... -v -run TestBreakerTransport_OpensAfterThreshold
```

Expected: `FAIL — NewBreakerTransport not defined`

- [ ] **Step 4: Implement the breaker transport**

Create `services/api-gateway/internal/proxy/breaker.go`:

```go
package proxy

import (
    "net/http"
    "time"

    "github.com/sony/gobreaker"
)

type breakerTransport struct {
    cb    *gobreaker.CircuitBreaker
    inner http.RoundTripper
}

// NewBreakerTransport wraps inner with a circuit breaker named by upstream host.
// Opens after 5 consecutive failures; half-opens after 30s timeout.
func NewBreakerTransport(name string, inner http.RoundTripper) http.RoundTripper {
    settings := gobreaker.Settings{
        Name:        name,
        MaxRequests: 1,
        Interval:    60 * time.Second,
        Timeout:     30 * time.Second,
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            return counts.ConsecutiveFailures >= 5
        },
    }
    return &breakerTransport{
        cb:    gobreaker.NewCircuitBreaker(settings),
        inner: inner,
    }
}

func (t *breakerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    result, err := t.cb.Execute(func() (interface{}, error) {
        return t.inner.RoundTrip(req)
    })
    if err != nil {
        return nil, err
    }
    return result.(*http.Response), nil
}
```

- [ ] **Step 5: Run the test to confirm it passes**

```bash
cd services/api-gateway
go test ./internal/proxy/... -v -run TestBreakerTransport_OpensAfterThreshold
```

Expected: `PASS`

- [ ] **Step 6: Wire breaker into the proxy**

In `services/api-gateway/internal/proxy/proxy.go`, find where each reverse proxy is created. Replace the transport assignment:

```go
// Before:
proxy := httputil.NewSingleHostReverseProxy(target)

// After:
proxy := httputil.NewSingleHostReverseProxy(target)
proxy.Transport = NewBreakerTransport(target.Host, http.DefaultTransport)
```

- [ ] **Step 7: Build and verify**

```bash
cd services/api-gateway
go build ./...
```

Expected: no errors

- [ ] **Step 8: Commit**

```bash
git add services/api-gateway/internal/proxy/breaker.go \
        services/api-gateway/internal/proxy/breaker_test.go \
        services/api-gateway/internal/proxy/proxy.go \
        services/api-gateway/go.mod \
        services/api-gateway/go.sum
git commit -m "feat(gateway): add circuit breaker per upstream — opens after 5 consecutive failures, resets after 30s"
```

---

## Task 3: Checkout Saga — Kafka Event Types & Inventory Consumer

**Files:**
- Modify: `pkg/kafka/topics.go`
- Create: `services/inventory-service/internal/domain/events/events.go`
- Create: `services/inventory-service/internal/consumer/checkout_consumer.go`
- Create: `services/inventory-service/internal/consumer/checkout_consumer_test.go`
- Modify: `services/inventory-service/main.go`

**Interfaces:**
- Consumes: `pkg/kafka.Consumer`, `InventoryService.ReserveStock`
- Produces: Kafka messages on `inventory.reserved` and `inventory.reservation_failed` topics

- [ ] **Step 1: Add saga topic constants to pkg/kafka/topics.go**

Open `pkg/kafka/topics.go` and add:

```go
const (
    // existing topics ...

    // Checkout saga topics
    TopicCheckoutRequested        = "checkout.requested"
    TopicInventoryReserved        = "inventory.reserved"
    TopicInventoryReservationFailed = "inventory.reservation_failed"
    TopicPaymentCaptured          = "payment.captured"
    TopicPaymentFailed            = "payment.failed"
    TopicOrderConfirmed           = "order.confirmed"
    TopicOrderCancelled           = "order.cancelled"
)
```

- [ ] **Step 2: Define saga event structs**

Create `services/inventory-service/internal/domain/events/events.go`:

```go
package events

import "time"

type CheckoutRequestedEvent struct {
    OrderID     string    `json:"order_id"`
    UserID      string    `json:"user_id"`
    Items       []CheckoutItem `json:"items"`
    RequestedAt time.Time `json:"requested_at"`
}

type CheckoutItem struct {
    SKUID    string `json:"sku_id"`
    Quantity int    `json:"quantity"`
}

type InventoryReservedEvent struct {
    OrderID       string            `json:"order_id"`
    Reservations  []ReservationRef  `json:"reservations"`
    ReservedAt    time.Time         `json:"reserved_at"`
}

type ReservationRef struct {
    SKUID         string `json:"sku_id"`
    ReservationID string `json:"reservation_id"`
    Quantity      int    `json:"quantity"`
}

type InventoryReservationFailedEvent struct {
    OrderID   string    `json:"order_id"`
    Reason    string    `json:"reason"`
    FailedAt  time.Time `json:"failed_at"`
}
```

- [ ] **Step 3: Write the failing test for the checkout consumer**

Create `services/inventory-service/internal/consumer/checkout_consumer_test.go`:

```go
package consumer_test

import (
    "context"
    "encoding/json"
    "testing"
    "time"

    "github.com/zapmarket/inventory-service/internal/consumer"
    "github.com/zapmarket/inventory-service/internal/domain/events"
)

type fakeInventoryService struct {
    reservedOrderID string
    reserveErr      error
}

func (f *fakeInventoryService) ReserveStock(ctx context.Context, orderID, skuID string, qty int) (string, error) {
    f.reservedOrderID = orderID
    if f.reserveErr != nil {
        return "", f.reserveErr
    }
    return "res-abc", nil
}

type fakePublisher struct {
    topic   string
    payload []byte
}

func (f *fakePublisher) Publish(ctx context.Context, topic string, key string, payload []byte) error {
    f.topic = topic
    f.payload = payload
    return nil
}

func TestCheckoutConsumer_ReservesAndPublishesSuccess(t *testing.T) {
    svc := &fakeInventoryService{}
    pub := &fakePublisher{}
    c := consumer.NewCheckoutConsumer(svc, pub)

    evt := events.CheckoutRequestedEvent{
        OrderID:     "ord-1",
        UserID:      "usr-1",
        Items:       []events.CheckoutItem{{SKUID: "sku-1", Quantity: 2}},
        RequestedAt: time.Now(),
    }
    payload, _ := json.Marshal(evt)

    if err := c.Handle(context.Background(), payload); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if svc.reservedOrderID != "ord-1" {
        t.Fatalf("expected order ord-1 to be reserved, got %q", svc.reservedOrderID)
    }
    if pub.topic != "inventory.reserved" {
        t.Fatalf("expected publish to inventory.reserved, got %q", pub.topic)
    }
}

func TestCheckoutConsumer_PublishesFailureOnReserveError(t *testing.T) {
    svc := &fakeInventoryService{reserveErr: errors.New("insufficient stock")}
    pub := &fakePublisher{}
    c := consumer.NewCheckoutConsumer(svc, pub)

    evt := events.CheckoutRequestedEvent{
        OrderID: "ord-2",
        Items:   []events.CheckoutItem{{SKUID: "sku-1", Quantity: 999}},
    }
    payload, _ := json.Marshal(evt)

    if err := c.Handle(context.Background(), payload); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if pub.topic != "inventory.reservation_failed" {
        t.Fatalf("expected publish to inventory.reservation_failed, got %q", pub.topic)
    }
}
```

- [ ] **Step 4: Run to confirm failure**

```bash
cd services/inventory-service
go test ./internal/consumer/... -v
```

Expected: `FAIL — cannot find package "consumer"`

- [ ] **Step 5: Implement the checkout consumer**

Create `services/inventory-service/internal/consumer/checkout_consumer.go`:

```go
package consumer

import (
    "context"
    "encoding/json"
    "time"

    "github.com/zapmarket/inventory-service/internal/domain/events"
    kafkapkg "github.com/zapmarket/pkg/kafka"
    "github.com/zapmarket/pkg/logger"
)

type inventoryReserver interface {
    ReserveStock(ctx context.Context, orderID, skuID string, qty int) (reservationID string, err error)
}

type eventPublisher interface {
    Publish(ctx context.Context, topic, key string, payload []byte) error
}

type CheckoutConsumer struct {
    svc inventoryReserver
    pub eventPublisher
}

func NewCheckoutConsumer(svc inventoryReserver, pub eventPublisher) *CheckoutConsumer {
    return &CheckoutConsumer{svc: svc, pub: pub}
}

func (c *CheckoutConsumer) Handle(ctx context.Context, payload []byte) error {
    var evt events.CheckoutRequestedEvent
    if err := json.Unmarshal(payload, &evt); err != nil {
        return err
    }

    var refs []events.ReservationRef
    for _, item := range evt.Items {
        resID, err := c.svc.ReserveStock(ctx, evt.OrderID, item.SKUID, item.Quantity)
        if err != nil {
            logger.Error(ctx, "checkout consumer: reserve failed", "order_id", evt.OrderID, "sku_id", item.SKUID, "error", err)
            return c.publishFailure(ctx, evt.OrderID, err.Error())
        }
        refs = append(refs, events.ReservationRef{
            SKUID:         item.SKUID,
            ReservationID: resID,
            Quantity:      item.Quantity,
        })
    }

    result := events.InventoryReservedEvent{
        OrderID:      evt.OrderID,
        Reservations: refs,
        ReservedAt:   time.Now(),
    }
    b, _ := json.Marshal(result)
    return c.pub.Publish(ctx, kafkapkg.TopicInventoryReserved, evt.OrderID, b)
}

func (c *CheckoutConsumer) publishFailure(ctx context.Context, orderID, reason string) error {
    evt := events.InventoryReservationFailedEvent{
        OrderID:  orderID,
        Reason:   reason,
        FailedAt: time.Now(),
    }
    b, _ := json.Marshal(evt)
    return c.pub.Publish(ctx, kafkapkg.TopicInventoryReservationFailed, orderID, b)
}
```

- [ ] **Step 6: Run test to confirm pass**

```bash
cd services/inventory-service
go test ./internal/consumer/... -v
```

Expected: `PASS`

- [ ] **Step 7: Wire consumer into main.go**

In `services/inventory-service/main.go`, after existing setup:

```go
checkoutConsumer := consumer.NewCheckoutConsumer(inventoryService, kafkaPublisher)
go func() {
    if err := kafkaClient.Subscribe(ctx, kafka.TopicCheckoutRequested, func(msg []byte) error {
        return checkoutConsumer.Handle(ctx, msg)
    }); err != nil {
        logger.Error(ctx, "checkout consumer exited", "error", err)
    }
}()
```

- [ ] **Step 8: Commit**

```bash
git add pkg/kafka/topics.go \
        services/inventory-service/internal/domain/events/ \
        services/inventory-service/internal/consumer/ \
        services/inventory-service/main.go
git commit -m "feat(inventory): saga consumer — handles checkout.requested, publishes inventory.reserved or .reservation_failed"
```

---

## Task 4: Checkout Saga — Payment Consumer

**Files:**
- Create: `services/payment-service/internal/domain/events/events.go`
- Create: `services/payment-service/internal/consumer/inventory_consumer.go`
- Create: `services/payment-service/internal/consumer/inventory_consumer_test.go`
- Modify: `services/payment-service/main.go`

**Interfaces:**
- Consumes: `inventory.reserved` Kafka topic
- Produces: `payment.captured` or `payment.failed` Kafka topics

- [ ] **Step 1: Define payment-side event structs**

Create `services/payment-service/internal/domain/events/events.go`:

```go
package events

import "time"

// Incoming
type InventoryReservedEvent struct {
    OrderID      string           `json:"order_id"`
    Reservations []ReservationRef `json:"reservations"`
    ReservedAt   time.Time        `json:"reserved_at"`
}

type ReservationRef struct {
    SKUID         string `json:"sku_id"`
    ReservationID string `json:"reservation_id"`
    Quantity      int    `json:"quantity"`
}

// Outgoing
type PaymentCapturedEvent struct {
    OrderID       string    `json:"order_id"`
    PaymentID     string    `json:"payment_id"`
    AmountCents   int64     `json:"amount_cents"`
    Currency      string    `json:"currency"`
    CapturedAt    time.Time `json:"captured_at"`
}

type PaymentFailedEvent struct {
    OrderID  string    `json:"order_id"`
    Reason   string    `json:"reason"`
    FailedAt time.Time `json:"failed_at"`
}
```

- [ ] **Step 2: Write the failing test**

Create `services/payment-service/internal/consumer/inventory_consumer_test.go`:

```go
package consumer_test

import (
    "context"
    "encoding/json"
    "errors"
    "testing"
    "time"

    "github.com/zapmarket/payment-service/internal/consumer"
    "github.com/zapmarket/payment-service/internal/domain/events"
)

type fakePaymentService struct {
    chargedOrderID string
    chargeErr      error
    paymentID      string
}

func (f *fakePaymentService) ChargeForOrder(ctx context.Context, orderID string) (paymentID string, amountCents int64, currency string, err error) {
    f.chargedOrderID = orderID
    if f.chargeErr != nil {
        return "", 0, "", f.chargeErr
    }
    return "pay-123", 9900, "INR", nil
}

type fakePublisher struct{ topic string }

func (f *fakePublisher) Publish(_ context.Context, topic, key string, _ []byte) error {
    f.topic = topic
    return nil
}

func TestInventoryConsumer_ChargesAndPublishesSuccess(t *testing.T) {
    svc := &fakePaymentService{}
    pub := &fakePublisher{}
    c := consumer.NewInventoryConsumer(svc, pub)

    evt := events.InventoryReservedEvent{
        OrderID:    "ord-1",
        ReservedAt: time.Now(),
    }
    payload, _ := json.Marshal(evt)

    if err := c.Handle(context.Background(), payload); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if svc.chargedOrderID != "ord-1" {
        t.Fatalf("expected order ord-1 charged, got %q", svc.chargedOrderID)
    }
    if pub.topic != "payment.captured" {
        t.Fatalf("expected payment.captured, got %q", pub.topic)
    }
}

func TestInventoryConsumer_PublishesFailureOnChargeError(t *testing.T) {
    svc := &fakePaymentService{chargeErr: errors.New("card declined")}
    pub := &fakePublisher{}
    c := consumer.NewInventoryConsumer(svc, pub)

    evt := events.InventoryReservedEvent{OrderID: "ord-2"}
    payload, _ := json.Marshal(evt)
    _ = c.Handle(context.Background(), payload)

    if pub.topic != "payment.failed" {
        t.Fatalf("expected payment.failed, got %q", pub.topic)
    }
}
```

- [ ] **Step 3: Run to confirm failure**

```bash
cd services/payment-service
go test ./internal/consumer/... -v
```

Expected: `FAIL — cannot find package "consumer"`

- [ ] **Step 4: Implement inventory consumer**

Create `services/payment-service/internal/consumer/inventory_consumer.go`:

```go
package consumer

import (
    "context"
    "encoding/json"
    "time"

    "github.com/zapmarket/payment-service/internal/domain/events"
    kafkapkg "github.com/zapmarket/pkg/kafka"
    "github.com/zapmarket/pkg/logger"
)

type orderCharge interface {
    ChargeForOrder(ctx context.Context, orderID string) (paymentID string, amountCents int64, currency string, err error)
}

type eventPublisher interface {
    Publish(ctx context.Context, topic, key string, payload []byte) error
}

type InventoryConsumer struct {
    svc orderCharge
    pub eventPublisher
}

func NewInventoryConsumer(svc orderCharge, pub eventPublisher) *InventoryConsumer {
    return &InventoryConsumer{svc: svc, pub: pub}
}

func (c *InventoryConsumer) Handle(ctx context.Context, payload []byte) error {
    var evt events.InventoryReservedEvent
    if err := json.Unmarshal(payload, &evt); err != nil {
        return err
    }

    paymentID, amountCents, currency, err := c.svc.ChargeForOrder(ctx, evt.OrderID)
    if err != nil {
        logger.Error(ctx, "inventory consumer: charge failed", "order_id", evt.OrderID, "error", err)
        fail := events.PaymentFailedEvent{
            OrderID:  evt.OrderID,
            Reason:   err.Error(),
            FailedAt: time.Now(),
        }
        b, _ := json.Marshal(fail)
        return c.pub.Publish(ctx, kafkapkg.TopicPaymentFailed, evt.OrderID, b)
    }

    captured := events.PaymentCapturedEvent{
        OrderID:     evt.OrderID,
        PaymentID:   paymentID,
        AmountCents: amountCents,
        Currency:    currency,
        CapturedAt:  time.Now(),
    }
    b, _ := json.Marshal(captured)
    return c.pub.Publish(ctx, kafkapkg.TopicPaymentCaptured, evt.OrderID, b)
}
```

- [ ] **Step 5: Run test to confirm pass**

```bash
cd services/payment-service
go test ./internal/consumer/... -v
```

Expected: `PASS`

- [ ] **Step 6: Wire consumer into main.go**

In `services/payment-service/main.go`:

```go
inventoryConsumer := consumer.NewInventoryConsumer(paymentService, kafkaPublisher)
go func() {
    if err := kafkaClient.Subscribe(ctx, kafka.TopicInventoryReserved, func(msg []byte) error {
        return inventoryConsumer.Handle(ctx, msg)
    }); err != nil {
        logger.Error(ctx, "inventory consumer exited", "error", err)
    }
}()
```

- [ ] **Step 7: Commit**

```bash
git add services/payment-service/internal/domain/events/ \
        services/payment-service/internal/consumer/ \
        services/payment-service/main.go
git commit -m "feat(payment): saga consumer — handles inventory.reserved, publishes payment.captured or .failed"
```

---

## Task 5: Checkout Saga — Order Management Orchestrator

**Files:**
- Create: `services/order-management-service/migrations/0003_saga_status.up.sql`
- Create: `services/order-management-service/migrations/0003_saga_status.down.sql`
- Create: `services/order-management-service/internal/consumer/saga_consumer.go`
- Create: `services/order-management-service/internal/consumer/saga_consumer_test.go`
- Modify: `services/order-management-service/internal/service/order_service.go`
- Modify: `services/order-management-service/main.go`

**Interfaces:**
- Consumes: `payment.captured`, `payment.failed`, `inventory.reservation_failed` Kafka topics
- Produces: `order.confirmed`, `order.cancelled` Kafka topics; calls gRPC `ReleaseStock` and `RefundPayment` for compensation

- [ ] **Step 1: Add saga_status migration**

Create `services/order-management-service/migrations/0003_saga_status.up.sql`:

```sql
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS saga_status VARCHAR(32) NOT NULL DEFAULT 'AWAITING_INVENTORY';

CREATE INDEX idx_orders_saga_status ON orders(saga_status) WHERE saga_status != 'COMPLETE';
```

Create `services/order-management-service/migrations/0003_saga_status.down.sql`:

```sql
DROP INDEX IF EXISTS idx_orders_saga_status;
ALTER TABLE orders DROP COLUMN IF EXISTS saga_status;
```

- [ ] **Step 2: Modify CreateOrder to publish checkout.requested instead of calling gRPC**

In `services/order-management-service/internal/service/order_service.go`, replace the gRPC reservation block:

```go
// REMOVE this block:
// resID, err := s.inventoryClient.ReserveStock(ctx, ...)
// paymentID, err := s.paymentClient.ChargeCard(ctx, ...)

// ADD this instead — write order to DB then publish event:
func (s *OrderService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*domain.Order, error) {
    order := &domain.Order{
        ID:          uuid.NewString(),
        UserID:      req.UserID,
        Status:      domain.OrderStatusPending,
        SagaStatus:  "AWAITING_INVENTORY",
        TotalAmount: req.TotalAmount,
        Currency:    req.Currency,
        Items:       req.Items,
        CreatedAt:   time.Now(),
    }
    if err := s.orderRepo.Save(ctx, order); err != nil {
        return nil, err
    }

    evt := events.CheckoutRequestedEvent{
        OrderID:     order.ID,
        UserID:      order.UserID,
        Items:       toCheckoutItems(order.Items),
        RequestedAt: time.Now(),
    }
    b, _ := json.Marshal(evt)
    if err := s.publisher.Publish(ctx, kafka.TopicCheckoutRequested, order.ID, b); err != nil {
        return nil, err
    }
    return order, nil
}
```

- [ ] **Step 3: Write the failing test for saga_consumer**

Create `services/order-management-service/internal/consumer/saga_consumer_test.go`:

```go
package consumer_test

import (
    "context"
    "encoding/json"
    "testing"
    "time"

    "github.com/zapmarket/order-management-service/internal/consumer"
    "github.com/zapmarket/order-management-service/internal/domain/events"
)

type fakeOrderRepo struct {
    updatedStatus   string
    updatedSagaStatus string
}

func (f *fakeOrderRepo) UpdateStatus(ctx context.Context, orderID, status, sagaStatus string) error {
    f.updatedStatus = status
    f.updatedSagaStatus = sagaStatus
    return nil
}

type fakeCompensator struct {
    releasedOrderID string
    refundedOrderID string
}

func (f *fakeCompensator) ReleaseStock(ctx context.Context, orderID string) error {
    f.releasedOrderID = orderID
    return nil
}

func (f *fakeCompensator) RefundPayment(ctx context.Context, orderID string) error {
    f.refundedOrderID = orderID
    return nil
}

type fakePublisher struct{ topic string }

func (f *fakePublisher) Publish(_ context.Context, topic, key string, _ []byte) error {
    f.topic = topic
    return nil
}

func TestSagaConsumer_ConfirmsOrderOnPaymentCaptured(t *testing.T) {
    repo := &fakeOrderRepo{}
    comp := &fakeCompensator{}
    pub := &fakePublisher{}
    c := consumer.NewSagaConsumer(repo, comp, pub)

    evt := events.PaymentCapturedEvent{OrderID: "ord-1", PaymentID: "pay-1", CapturedAt: time.Now()}
    payload, _ := json.Marshal(evt)

    if err := c.HandlePaymentCaptured(context.Background(), payload); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if repo.updatedStatus != "CONFIRMED" {
        t.Fatalf("expected CONFIRMED, got %q", repo.updatedStatus)
    }
    if pub.topic != "order.confirmed" {
        t.Fatalf("expected order.confirmed, got %q", pub.topic)
    }
}

func TestSagaConsumer_CancelsOrderAndReleasesOnInventoryFailed(t *testing.T) {
    repo := &fakeOrderRepo{}
    comp := &fakeCompensator{}
    pub := &fakePublisher{}
    c := consumer.NewSagaConsumer(repo, comp, pub)

    evt := events.InventoryReservationFailedEvent{OrderID: "ord-2", Reason: "out of stock"}
    payload, _ := json.Marshal(evt)

    if err := c.HandleInventoryFailed(context.Background(), payload); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if repo.updatedStatus != "CANCELLED" {
        t.Fatalf("expected CANCELLED, got %q", repo.updatedStatus)
    }
    if comp.releasedOrderID != "" {
        // inventory never reserved so nothing to release — compensator should NOT be called
        t.Fatalf("release should not be called when inventory never reserved")
    }
}

func TestSagaConsumer_CancelsAndRefundsOnPaymentFailed(t *testing.T) {
    repo := &fakeOrderRepo{}
    comp := &fakeCompensator{}
    pub := &fakePublisher{}
    c := consumer.NewSagaConsumer(repo, comp, pub)

    evt := events.PaymentFailedEvent{OrderID: "ord-3", Reason: "card declined"}
    payload, _ := json.Marshal(evt)

    if err := c.HandlePaymentFailed(context.Background(), payload); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if repo.updatedStatus != "CANCELLED" {
        t.Fatalf("expected CANCELLED, got %q", repo.updatedStatus)
    }
    if comp.releasedOrderID != "ord-3" {
        t.Fatalf("expected inventory release for ord-3, got %q", comp.releasedOrderID)
    }
}
```

- [ ] **Step 4: Run to confirm failure**

```bash
cd services/order-management-service
go test ./internal/consumer/... -v
```

Expected: `FAIL — cannot find package "consumer"`

- [ ] **Step 5: Implement saga consumer**

Create `services/order-management-service/internal/consumer/saga_consumer.go`:

```go
package consumer

import (
    "context"
    "encoding/json"
    "time"

    "github.com/zapmarket/order-management-service/internal/domain/events"
    kafkapkg "github.com/zapmarket/pkg/kafka"
    "github.com/zapmarket/pkg/logger"
)

type orderStatusUpdater interface {
    UpdateStatus(ctx context.Context, orderID, status, sagaStatus string) error
}

type compensator interface {
    ReleaseStock(ctx context.Context, orderID string) error
    RefundPayment(ctx context.Context, orderID string) error
}

type eventPublisher interface {
    Publish(ctx context.Context, topic, key string, payload []byte) error
}

type SagaConsumer struct {
    repo orderStatusUpdater
    comp compensator
    pub  eventPublisher
}

func NewSagaConsumer(repo orderStatusUpdater, comp compensator, pub eventPublisher) *SagaConsumer {
    return &SagaConsumer{repo: repo, comp: comp, pub: pub}
}

func (c *SagaConsumer) HandlePaymentCaptured(ctx context.Context, payload []byte) error {
    var evt events.PaymentCapturedEvent
    if err := json.Unmarshal(payload, &evt); err != nil {
        return err
    }
    if err := c.repo.UpdateStatus(ctx, evt.OrderID, "CONFIRMED", "COMPLETE"); err != nil {
        return err
    }
    confirmed := map[string]interface{}{
        "order_id":     evt.OrderID,
        "payment_id":   evt.PaymentID,
        "confirmed_at": time.Now(),
    }
    b, _ := json.Marshal(confirmed)
    return c.pub.Publish(ctx, kafkapkg.TopicOrderConfirmed, evt.OrderID, b)
}

func (c *SagaConsumer) HandleInventoryFailed(ctx context.Context, payload []byte) error {
    var evt events.InventoryReservationFailedEvent
    if err := json.Unmarshal(payload, &evt); err != nil {
        return err
    }
    logger.Warn(ctx, "saga: inventory reservation failed — cancelling order", "order_id", evt.OrderID, "reason", evt.Reason)
    if err := c.repo.UpdateStatus(ctx, evt.OrderID, "CANCELLED", "COMPENSATED"); err != nil {
        return err
    }
    cancelled := map[string]interface{}{"order_id": evt.OrderID, "reason": evt.Reason, "cancelled_at": time.Now()}
    b, _ := json.Marshal(cancelled)
    return c.pub.Publish(ctx, kafkapkg.TopicOrderCancelled, evt.OrderID, b)
}

func (c *SagaConsumer) HandlePaymentFailed(ctx context.Context, payload []byte) error {
    var evt events.PaymentFailedEvent
    if err := json.Unmarshal(payload, &evt); err != nil {
        return err
    }
    logger.Warn(ctx, "saga: payment failed — releasing inventory and cancelling order", "order_id", evt.OrderID, "reason", evt.Reason)
    if err := c.comp.ReleaseStock(ctx, evt.OrderID); err != nil {
        logger.Error(ctx, "saga: release stock failed during compensation", "order_id", evt.OrderID, "error", err)
        // continue to cancel regardless
    }
    if err := c.repo.UpdateStatus(ctx, evt.OrderID, "CANCELLED", "COMPENSATED"); err != nil {
        return err
    }
    cancelled := map[string]interface{}{"order_id": evt.OrderID, "reason": evt.Reason, "cancelled_at": time.Now()}
    b, _ := json.Marshal(cancelled)
    return c.pub.Publish(ctx, kafkapkg.TopicOrderCancelled, evt.OrderID, b)
}
```

- [ ] **Step 6: Run test to confirm pass**

```bash
cd services/order-management-service
go test ./internal/consumer/... -v
```

Expected: `PASS`

- [ ] **Step 7: Wire three consumers into main.go**

In `services/order-management-service/main.go`:

```go
sagaConsumer := consumer.NewSagaConsumer(orderRepo, compensatorClient, kafkaPublisher)

subscribeAsync(ctx, kafkaClient, kafka.TopicPaymentCaptured, sagaConsumer.HandlePaymentCaptured)
subscribeAsync(ctx, kafkaClient, kafka.TopicPaymentFailed, sagaConsumer.HandlePaymentFailed)
subscribeAsync(ctx, kafkaClient, kafka.TopicInventoryReservationFailed, sagaConsumer.HandleInventoryFailed)

// helper to reduce boilerplate
func subscribeAsync(ctx context.Context, kc *kafka.Client, topic string, handler func(context.Context, []byte) error) {
    go func() {
        if err := kc.Subscribe(ctx, topic, func(msg []byte) error {
            return handler(ctx, msg)
        }); err != nil {
            logger.Error(ctx, "consumer exited", "topic", topic, "error", err)
        }
    }()
}
```

- [ ] **Step 8: Run migration**

```bash
cd services/order-management-service
go run ./cmd/migrate/... up   # or however migrations are run in this service
```

Expected: `0003_saga_status.up.sql applied`

- [ ] **Step 9: Full build check**

```bash
cd services/order-management-service
go build ./...
go test ./...
```

Expected: all pass

- [ ] **Step 10: Final commit**

```bash
git add services/order-management-service/migrations/0003_saga_status.* \
        services/order-management-service/internal/consumer/ \
        services/order-management-service/internal/service/order_service.go \
        services/order-management-service/main.go
git commit -m "feat(order): choreography saga — CreateOrder publishes checkout.requested; saga_consumer confirms or compensates on payment/inventory results"
```

---

## Verification Checklist

After all 5 tasks are complete, run this end-to-end smoke test:

- [ ] Start all services: `docker compose up -d`
- [ ] Create a product with stock via `POST /v1/products` + `POST /v1/inventory/add`
- [ ] Place an order via `POST /v1/orders` — observe `saga_status = AWAITING_INVENTORY` in DB
- [ ] Kafka relay publishes `checkout.requested`
- [ ] inventory-service consumes → publishes `inventory.reserved`
- [ ] payment-service consumes → publishes `payment.captured`
- [ ] order-management consumes → order status = `CONFIRMED`, saga_status = `COMPLETE`
- [ ] Repeat with insufficient stock — order should end `CANCELLED`, saga_status = `COMPENSATED`
- [ ] Kill inventory-service mid-checkout — after restart, reservation expiry worker should release after 15 min (set `RESERVATION_TTL=1m` for testing)
- [ ] Verify API gateway returns 503 with circuit breaker message after 5 upstream failures
