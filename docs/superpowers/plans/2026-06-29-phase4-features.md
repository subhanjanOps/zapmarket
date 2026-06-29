# Phase 4 — Reviews, Promotions, Shipping & Notifications Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add four features in parallel tracks: (A) Reviews & Returns domain, (B) Promotions engine with coupon codes, (C) Carrier-agnostic shipping integration via Shiprocket, and (D) wire SMS + wishlist notifications.

**Architecture:** Each feature is a new microservice or an extension to an existing one. Reviews and returns are a single `review-return-service`. Promotions are a `promotions-service` that product-catalog-service calls at price-computation time. Shipping is a `logistics-service` that consumes `order.confirmed` events and creates shipments. SMS and wishlist are extensions to `notification-service` and a new `wishlist-service`.

**Tech Stack:** Go 1.25, Shiprocket REST API, Twilio SMS (SDK already in auth-service — reuse), existing `pkg/kafka`, existing `pkg/database`.

## Global Constraints

- Reviews require `verified_purchase = true` (order must be in CONFIRMED status for that SKU)
- Coupon codes are case-insensitive, stored uppercase
- Shipping labels are fetched from Shiprocket and stored as URLs in MinIO
- Wishlist is per-user, max 200 items
- SMS sends are fire-and-forget — failures are logged, not retried synchronously
- All new Kafka topics in `pkg/kafka/topics.go`

---

## Track A: Reviews & Returns Service

### File Map
```
services/review-return-service/
  cmd/server/main.go
  migrations/
    0001_init.up.sql / down
  internal/
    domain/
      review.go           — Review, Rating, ReturnRequest entities
      errors/errors.go
    application/
      interfaces.go
      usecases/
        submit_review.go
        list_reviews.go
        request_return.go
        approve_return.go
    infrastructure/
      postgres/
        review_repo.go
        return_repo.go
      grpc/
        order_client.go   — verifies purchase before allowing review
    interfaces/
      http/
        review_handler.go
        return_handler.go
```

### Task A1: Reviews Domain & Migration

- [ ] **Step 1: Create domain types**

Create `services/review-return-service/internal/domain/review.go`:

```go
package domain

import "time"

type Review struct {
    ID              string
    ProductID       string
    SKUID           string
    UserID          string
    OrderID         string
    VerifiedPurchase bool
    Rating          int    // 1–5
    Title           string
    Body            string
    ImageURLs       []string
    HelpfulCount    int
    Status          string // PENDING_MODERATION, PUBLISHED, REJECTED
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type ReturnRequest struct {
    ID          string
    OrderID     string
    OrderItemID string
    UserID      string
    Reason      string // DAMAGED, WRONG_ITEM, NOT_AS_DESCRIBED, CHANGED_MIND
    Description string
    Status      string // REQUESTED, APPROVED, REJECTED, REFUNDED
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

- [ ] **Step 2: Write migration**

Create `services/review-return-service/migrations/0001_init.up.sql`:

```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE reviews (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id        UUID NOT NULL,
    sku_id            UUID NOT NULL,
    user_id           UUID NOT NULL,
    order_id          UUID NOT NULL,
    verified_purchase BOOLEAN NOT NULL DEFAULT FALSE,
    rating            SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    title             TEXT NOT NULL,
    body              TEXT,
    image_urls        TEXT[] NOT NULL DEFAULT '{}',
    helpful_count     INT NOT NULL DEFAULT 0,
    status            VARCHAR(32) NOT NULL DEFAULT 'PENDING_MODERATION',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, order_id, sku_id)
);

CREATE TABLE return_requests (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id      UUID NOT NULL,
    order_item_id UUID NOT NULL,
    user_id       UUID NOT NULL,
    reason        VARCHAR(64) NOT NULL,
    description   TEXT,
    status        VARCHAR(32) NOT NULL DEFAULT 'REQUESTED',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reviews_product ON reviews(product_id, status);
CREATE INDEX idx_reviews_user ON reviews(user_id);
CREATE INDEX idx_returns_order ON return_requests(order_id);
```

- [ ] **Step 3: Write failing test for SubmitReviewUseCase**

Create `services/review-return-service/internal/application/usecases/submit_review_test.go`:

```go
package usecases_test

import (
    "context"
    "errors"
    "testing"

    "github.com/zapmarket/review-return-service/internal/application/usecases"
)

type fakeOrderVerifier struct{ confirmed bool }

func (f *fakeOrderVerifier) IsOrderConfirmedForSKU(_ context.Context, userID, orderID, skuID string) (bool, error) {
    return f.confirmed, nil
}

type fakeReviewRepo struct{ saved bool }

func (f *fakeReviewRepo) Save(_ context.Context, _ interface{}) error {
    f.saved = true
    return nil
}

func TestSubmitReview_RequiresVerifiedPurchase(t *testing.T) {
    verifier := &fakeOrderVerifier{confirmed: false}
    repo := &fakeReviewRepo{}
    uc := usecases.NewSubmitReviewUseCase(repo, verifier)

    err := uc.Execute(context.Background(), usecases.SubmitReviewInput{
        UserID: "u1", OrderID: "o1", SKUID: "s1", Rating: 5, Title: "Great",
    })
    if !errors.Is(err, usecases.ErrNotVerifiedPurchase) {
        t.Fatalf("expected ErrNotVerifiedPurchase, got %v", err)
    }
    if repo.saved {
        t.Fatal("review should not be saved without verified purchase")
    }
}

func TestSubmitReview_SavesWhenVerified(t *testing.T) {
    verifier := &fakeOrderVerifier{confirmed: true}
    repo := &fakeReviewRepo{}
    uc := usecases.NewSubmitReviewUseCase(repo, verifier)

    err := uc.Execute(context.Background(), usecases.SubmitReviewInput{
        UserID: "u1", OrderID: "o1", SKUID: "s1", Rating: 4, Title: "Good product",
    })
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !repo.saved {
        t.Fatal("expected review to be saved")
    }
}
```

- [ ] **Step 4: Implement SubmitReviewUseCase**

Create `services/review-return-service/internal/application/usecases/submit_review.go`:

```go
package usecases

import (
    "context"
    "errors"
    "time"

    "github.com/google/uuid"
    "github.com/zapmarket/review-return-service/internal/domain"
)

var ErrNotVerifiedPurchase = errors.New("review requires a verified purchase for this SKU")

type orderVerifier interface {
    IsOrderConfirmedForSKU(ctx context.Context, userID, orderID, skuID string) (bool, error)
}

type reviewSaver interface {
    Save(ctx context.Context, review *domain.Review) error
}

type SubmitReviewInput struct {
    UserID    string
    OrderID   string
    ProductID string
    SKUID     string
    Rating    int
    Title     string
    Body      string
    ImageURLs []string
}

type SubmitReviewUseCase struct {
    repo     reviewSaver
    verifier orderVerifier
}

func NewSubmitReviewUseCase(repo reviewSaver, verifier orderVerifier) *SubmitReviewUseCase {
    return &SubmitReviewUseCase{repo: repo, verifier: verifier}
}

func (uc *SubmitReviewUseCase) Execute(ctx context.Context, in SubmitReviewInput) error {
    confirmed, err := uc.verifier.IsOrderConfirmedForSKU(ctx, in.UserID, in.OrderID, in.SKUID)
    if err != nil {
        return err
    }
    if !confirmed {
        return ErrNotVerifiedPurchase
    }
    review := &domain.Review{
        ID:               uuid.NewString(),
        ProductID:        in.ProductID,
        SKUID:            in.SKUID,
        UserID:           in.UserID,
        OrderID:          in.OrderID,
        VerifiedPurchase: true,
        Rating:           in.Rating,
        Title:            in.Title,
        Body:             in.Body,
        ImageURLs:        in.ImageURLs,
        Status:           "PENDING_MODERATION",
        CreatedAt:        time.Now(),
        UpdatedAt:        time.Now(),
    }
    return uc.repo.Save(ctx, review)
}
```

- [ ] **Step 5: Run tests**

```bash
cd services/review-return-service
go test ./internal/application/... -v
```

Expected: `PASS`

- [ ] **Step 6: Commit**

```bash
git add services/review-return-service/
git commit -m "feat(reviews): SubmitReview requires verified purchase, PENDING_MODERATION default status, return_requests table"
```

---

## Track B: Promotions Engine

### File Map
```
services/promotions-service/
  cmd/server/main.go
  migrations/
    0001_init.up.sql / down
  internal/
    domain/
      coupon.go         — Coupon, DiscountType, UsageRecord
      errors/errors.go
    application/
      interfaces.go
      usecases/
        create_coupon.go
        validate_coupon.go   — called at checkout to apply discount
        record_usage.go
    infrastructure/
      postgres/
        coupon_repo.go
      redis/
        usage_cache.go      — rate-limit per-user coupon usage
    interfaces/
      http/
        coupon_handler.go   — POST /v1/coupons, POST /v1/coupons/validate
```

### Task B1: Promotions — Coupon Domain & Validate Use Case

- [ ] **Step 1: Create coupon domain**

Create `services/promotions-service/internal/domain/coupon.go`:

```go
package domain

import "time"

type DiscountType string

const (
    DiscountTypePercent    DiscountType = "PERCENT"
    DiscountTypeFixedPaise DiscountType = "FIXED_PAISE"
)

type Coupon struct {
    ID              string
    Code            string       // always stored UPPERCASE
    DiscountType    DiscountType
    DiscountValue   int64        // percent (1-100) or fixed paise amount
    MinOrderPaise   int64        // minimum cart value to apply
    MaxUsesTotal    int          // 0 = unlimited
    MaxUsesPerUser  int          // 0 = unlimited
    ExpiresAt       *time.Time
    IsActive        bool
    CreatedAt       time.Time
}

// Apply returns the discount amount in paise given cart total.
func (c *Coupon) Apply(cartTotalPaise int64) (discountPaise int64) {
    if cartTotalPaise < c.MinOrderPaise {
        return 0
    }
    switch c.DiscountType {
    case DiscountTypePercent:
        return (cartTotalPaise * c.DiscountValue) / 100
    case DiscountTypeFixedPaise:
        if c.DiscountValue > cartTotalPaise {
            return cartTotalPaise
        }
        return c.DiscountValue
    }
    return 0
}
```

- [ ] **Step 2: Write failing test for ValidateCouponUseCase**

Create `services/promotions-service/internal/application/usecases/validate_coupon_test.go`:

```go
package usecases_test

import (
    "context"
    "testing"
    "time"

    "github.com/zapmarket/promotions-service/internal/application/usecases"
    "github.com/zapmarket/promotions-service/internal/domain"
)

type fakeCouponRepo struct{ coupon *domain.Coupon }

func (f *fakeCouponRepo) FindByCode(_ context.Context, code string) (*domain.Coupon, error) {
    if f.coupon != nil && f.coupon.Code == code {
        return f.coupon, nil
    }
    return nil, domain.ErrCouponNotFound
}

func (f *fakeCouponRepo) CountUsageByUser(_ context.Context, _, _ string) (int, error) { return 0, nil }
func (f *fakeCouponRepo) CountUsageTotal(_ context.Context, _ string) (int, error)      { return 0, nil }

func TestValidateCoupon_AppliesPercentDiscount(t *testing.T) {
    expires := time.Now().Add(24 * time.Hour)
    repo := &fakeCouponRepo{coupon: &domain.Coupon{
        Code:           "SAVE10",
        DiscountType:   domain.DiscountTypePercent,
        DiscountValue:  10,
        MinOrderPaise:  50000,
        MaxUsesPerUser: 1,
        ExpiresAt:      &expires,
        IsActive:       true,
    }}
    uc := usecases.NewValidateCouponUseCase(repo)

    result, err := uc.Execute(context.Background(), usecases.ValidateCouponInput{
        Code:          "SAVE10",
        UserID:        "u1",
        CartTotalPaise: 100000, // ₹1000
    })
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result.DiscountPaise != 10000 { // 10% of 100000
        t.Fatalf("expected discount 10000, got %d", result.DiscountPaise)
    }
}

func TestValidateCoupon_RejectsExpiredCoupon(t *testing.T) {
    expired := time.Now().Add(-1 * time.Hour)
    repo := &fakeCouponRepo{coupon: &domain.Coupon{
        Code:      "OLD10",
        IsActive:  true,
        ExpiresAt: &expired,
    }}
    uc := usecases.NewValidateCouponUseCase(repo)

    _, err := uc.Execute(context.Background(), usecases.ValidateCouponInput{
        Code: "OLD10", UserID: "u1", CartTotalPaise: 100000,
    })
    if err == nil {
        t.Fatal("expected error for expired coupon")
    }
}
```

- [ ] **Step 3: Implement ValidateCouponUseCase**

Create `services/promotions-service/internal/application/usecases/validate_coupon.go`:

```go
package usecases

import (
    "context"
    "errors"
    "strings"
    "time"

    "github.com/zapmarket/promotions-service/internal/domain"
)

var ErrCouponExpired         = errors.New("coupon has expired")
var ErrCouponInactive        = errors.New("coupon is not active")
var ErrCartBelowMinimum      = errors.New("cart total is below coupon minimum order value")
var ErrUsageLimitReached     = errors.New("coupon usage limit has been reached")

type couponRepo interface {
    FindByCode(ctx context.Context, code string) (*domain.Coupon, error)
    CountUsageByUser(ctx context.Context, couponID, userID string) (int, error)
    CountUsageTotal(ctx context.Context, couponID string) (int, error)
}

type ValidateCouponInput struct {
    Code           string
    UserID         string
    CartTotalPaise int64
}

type ValidateCouponResult struct {
    CouponID      string
    DiscountPaise int64
    FinalPaise    int64
}

type ValidateCouponUseCase struct{ repo couponRepo }

func NewValidateCouponUseCase(repo couponRepo) *ValidateCouponUseCase {
    return &ValidateCouponUseCase{repo: repo}
}

func (uc *ValidateCouponUseCase) Execute(ctx context.Context, in ValidateCouponInput) (*ValidateCouponResult, error) {
    coupon, err := uc.repo.FindByCode(ctx, strings.ToUpper(in.Code))
    if err != nil {
        return nil, err
    }
    if !coupon.IsActive {
        return nil, ErrCouponInactive
    }
    if coupon.ExpiresAt != nil && time.Now().After(*coupon.ExpiresAt) {
        return nil, ErrCouponExpired
    }
    if in.CartTotalPaise < coupon.MinOrderPaise {
        return nil, ErrCartBelowMinimum
    }
    if coupon.MaxUsesPerUser > 0 {
        used, _ := uc.repo.CountUsageByUser(ctx, coupon.ID, in.UserID)
        if used >= coupon.MaxUsesPerUser {
            return nil, ErrUsageLimitReached
        }
    }
    if coupon.MaxUsesTotal > 0 {
        total, _ := uc.repo.CountUsageTotal(ctx, coupon.ID)
        if total >= coupon.MaxUsesTotal {
            return nil, ErrUsageLimitReached
        }
    }
    discount := coupon.Apply(in.CartTotalPaise)
    return &ValidateCouponResult{
        CouponID:      coupon.ID,
        DiscountPaise: discount,
        FinalPaise:    in.CartTotalPaise - discount,
    }, nil
}
```

- [ ] **Step 4: Run tests**

```bash
cd services/promotions-service
go test ./internal/application/... -v
```

Expected: `PASS`

- [ ] **Step 5: Commit**

```bash
git add services/promotions-service/
git commit -m "feat(promotions): coupon domain, ValidateCoupon use case, percent + fixed-paise discount types"
```

---

## Track C: Shipping / Logistics Service

### File Map
```
services/logistics-service/
  cmd/server/main.go
  migrations/
    0001_init.up.sql / down
  internal/
    domain/
      shipment.go       — Shipment, ShipmentStatus, TrackingEvent
    application/
      interfaces.go
      usecases/
        create_shipment.go   — called on order.confirmed event
        get_tracking.go
    infrastructure/
      shiprocket/
        client.go            — REST API calls to Shiprocket
        noop_client.go       — dev fallback
      postgres/
        shipment_repo.go
    interfaces/
      consumers/
        order_consumer.go    — consumes order.confirmed
      http/
        tracking_handler.go  — GET /v1/shipments/:order_id/tracking
```

### Task C1: Logistics — Domain & Create Shipment Use Case

- [ ] **Step 1: Create shipment domain**

Create `services/logistics-service/internal/domain/shipment.go`:

```go
package domain

import "time"

type ShipmentStatus string

const (
    ShipmentStatusCreated    ShipmentStatus = "CREATED"
    ShipmentStatusPickedUp   ShipmentStatus = "PICKED_UP"
    ShipmentStatusInTransit  ShipmentStatus = "IN_TRANSIT"
    ShipmentStatusOutForDelivery ShipmentStatus = "OUT_FOR_DELIVERY"
    ShipmentStatusDelivered  ShipmentStatus = "DELIVERED"
    ShipmentStatusFailed     ShipmentStatus = "DELIVERY_FAILED"
)

type Shipment struct {
    ID              string
    OrderID         string
    CarrierShipmentID string   // Shiprocket's AWB number
    Carrier         string
    TrackingURL     string
    Status          ShipmentStatus
    LabelURL        string    // MinIO URL to PDF label
    EstimatedDelivery *time.Time
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type TrackingEvent struct {
    Status      string
    Description string
    Location    string
    OccurredAt  time.Time
}
```

- [ ] **Step 2: Write failing test for CreateShipmentUseCase**

Create `services/logistics-service/internal/application/usecases/create_shipment_test.go`:

```go
package usecases_test

import (
    "context"
    "testing"

    "github.com/zapmarket/logistics-service/internal/application/usecases"
)

type fakeCarrierClient struct{ awb string }

func (f *fakeCarrierClient) CreateShipment(_ context.Context, _ usecases.ShipmentRequest) (awb, carrier, trackingURL string, err error) {
    return "AWB123", "Delhivery", "https://track.example.com/AWB123", nil
}

type fakeShipmentRepo struct{ saved bool }

func (f *fakeShipmentRepo) Save(_ context.Context, _ interface{}) error {
    f.saved = true
    return nil
}

func TestCreateShipment_SavesWithAWB(t *testing.T) {
    client := &fakeCarrierClient{}
    repo := &fakeShipmentRepo{}
    uc := usecases.NewCreateShipmentUseCase(client, repo)

    _, err := uc.Execute(context.Background(), usecases.ShipmentRequest{
        OrderID:         "ord-1",
        DeliveryPincode: "400001",
        WeightGrams:     500,
    })
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !repo.saved {
        t.Fatal("expected shipment to be saved")
    }
}
```

- [ ] **Step 3: Implement CreateShipmentUseCase**

Create `services/logistics-service/internal/application/usecases/create_shipment.go`:

```go
package usecases

import (
    "context"
    "time"

    "github.com/google/uuid"
    "github.com/zapmarket/logistics-service/internal/domain"
)

type ShipmentRequest struct {
    OrderID         string
    DeliveryPincode string
    WeightGrams     int
    SellerPincode   string
}

type carrierClient interface {
    CreateShipment(ctx context.Context, req ShipmentRequest) (awb, carrier, trackingURL string, err error)
}

type shipmentSaver interface {
    Save(ctx context.Context, s *domain.Shipment) error
}

type CreateShipmentUseCase struct {
    carrier carrierClient
    repo    shipmentSaver
}

func NewCreateShipmentUseCase(carrier carrierClient, repo shipmentSaver) *CreateShipmentUseCase {
    return &CreateShipmentUseCase{carrier: carrier, repo: repo}
}

func (uc *CreateShipmentUseCase) Execute(ctx context.Context, req ShipmentRequest) (*domain.Shipment, error) {
    awb, carrier, trackingURL, err := uc.carrier.CreateShipment(ctx, req)
    if err != nil {
        return nil, err
    }
    shipment := &domain.Shipment{
        ID:                uuid.NewString(),
        OrderID:           req.OrderID,
        CarrierShipmentID: awb,
        Carrier:           carrier,
        TrackingURL:       trackingURL,
        Status:            domain.ShipmentStatusCreated,
        CreatedAt:         time.Now(),
        UpdatedAt:         time.Now(),
    }
    if err := uc.repo.Save(ctx, shipment); err != nil {
        return nil, err
    }
    return shipment, nil
}
```

- [ ] **Step 4: Run tests**

```bash
cd services/logistics-service
go test ./internal/application/... -v
```

Expected: `PASS`

- [ ] **Step 5: Commit**

```bash
git add services/logistics-service/
git commit -m "feat(logistics): shipment domain, CreateShipment calls carrier API, saves AWB + tracking URL"
```

---

## Track D: Notifications — Wire SMS & Wishlist

### Task D1: Wire Twilio SMS into notification-service

**Files:**
- Modify: `services/notification-service/internal/notifier/notifier.go` — add SMS interface
- Create: `services/notification-service/internal/notifier/twilio_notifier.go`
- Modify: `services/notification-service/internal/consumer/handler.go` — call SMS for order events
- Modify: `services/notification-service/main.go` — inject SMS notifier

- [ ] **Step 1: Add SMS notifier interface**

In `services/notification-service/internal/notifier/notifier.go`, add:

```go
type SMSNotifier interface {
    SendSMS(ctx context.Context, phone, message string) error
}
```

- [ ] **Step 2: Implement Twilio notifier**

Create `services/notification-service/internal/notifier/twilio_notifier.go`:

```go
package notifier

import (
    "context"
    "fmt"
    "net/http"
    "net/url"
    "strings"

    "github.com/zapmarket/pkg/logger"
)

type TwilioNotifier struct {
    accountSID string
    authToken  string
    fromNumber string
    client     *http.Client
}

func NewTwilioNotifier(accountSID, authToken, fromNumber string) *TwilioNotifier {
    return &TwilioNotifier{
        accountSID: accountSID,
        authToken:  authToken,
        fromNumber: fromNumber,
        client:     &http.Client{},
    }
}

func (t *TwilioNotifier) SendSMS(ctx context.Context, phone, message string) error {
    endpoint := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", t.accountSID)
    data := url.Values{
        "To":   {phone},
        "From": {t.fromNumber},
        "Body": {message},
    }
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(data.Encode()))
    if err != nil {
        return err
    }
    req.SetBasicAuth(t.accountSID, t.authToken)
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := t.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    if resp.StatusCode >= 400 {
        logger.Error(ctx, "twilio SMS failed", "status", resp.StatusCode, "phone", phone)
        return fmt.Errorf("twilio returned %d", resp.StatusCode)
    }
    return nil
}
```

- [ ] **Step 3: Wire SMS calls in the handler for order events**

In `services/notification-service/internal/consumer/handler.go`, after sending email on `order.confirmed`:

```go
// Fire-and-forget SMS — failure is logged, not propagated
go func() {
    msg := fmt.Sprintf("Your ZapMarket order %s has been confirmed. Track it at zapmarket.in/orders/%s", orderID, orderID)
    if err := h.sms.SendSMS(ctx, userPhone, msg); err != nil {
        logger.Warn(ctx, "SMS notification failed", "order_id", orderID, "error", err)
    }
}()
```

- [ ] **Step 4: Run tests**

```bash
cd services/notification-service
go test ./... -v
```

Expected: `PASS` (SMS is fire-and-forget, existing tests unaffected)

- [ ] **Step 5: Commit**

```bash
git add services/notification-service/internal/notifier/twilio_notifier.go \
        services/notification-service/internal/notifier/notifier.go \
        services/notification-service/internal/consumer/handler.go \
        services/notification-service/main.go
git commit -m "feat(notifications): wire Twilio SMS for order.confirmed events, fire-and-forget"
```

---

### Task D2: Wishlist Service

**Files:**
- Create: `services/wishlist-service/` (new microservice, port 8089)
- Migrations, domain, use cases, HTTP handler — follows cart-service pattern

- [ ] **Step 1: Write migration**

Create `services/wishlist-service/migrations/0001_init.up.sql`:

```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE wishlist_items (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL,
    product_id  UUID NOT NULL,
    sku_id      UUID,
    added_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, product_id)
);

CREATE INDEX idx_wishlist_user ON wishlist_items(user_id);
```

- [ ] **Step 2: Write failing test for AddToWishlistUseCase**

Create `services/wishlist-service/internal/application/usecases/add_to_wishlist_test.go`:

```go
package usecases_test

import (
    "context"
    "testing"

    "github.com/zapmarket/wishlist-service/internal/application/usecases"
)

type fakeWishlistRepo struct {
    count int
    saved bool
}

func (f *fakeWishlistRepo) CountByUser(_ context.Context, _ string) (int, error) { return f.count, nil }
func (f *fakeWishlistRepo) Add(_ context.Context, _, _, _ string) error {
    f.saved = true
    return nil
}

func TestAddToWishlist_EnforcesMaxLimit(t *testing.T) {
    repo := &fakeWishlistRepo{count: 200}
    uc := usecases.NewAddToWishlistUseCase(repo, 200)

    err := uc.Execute(context.Background(), "u1", "prod-1", "sku-1")
    if err == nil {
        t.Fatal("expected error when wishlist is full")
    }
}

func TestAddToWishlist_SavesWhenUnderLimit(t *testing.T) {
    repo := &fakeWishlistRepo{count: 5}
    uc := usecases.NewAddToWishlistUseCase(repo, 200)

    if err := uc.Execute(context.Background(), "u1", "prod-1", "sku-1"); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !repo.saved {
        t.Fatal("expected item saved to wishlist")
    }
}
```

- [ ] **Step 3: Implement AddToWishlistUseCase**

Create `services/wishlist-service/internal/application/usecases/add_to_wishlist.go`:

```go
package usecases

import (
    "context"
    "errors"
)

var ErrWishlistFull = errors.New("wishlist is at maximum capacity")

type wishlistRepo interface {
    CountByUser(ctx context.Context, userID string) (int, error)
    Add(ctx context.Context, userID, productID, skuID string) error
}

type AddToWishlistUseCase struct {
    repo     wishlistRepo
    maxItems int
}

func NewAddToWishlistUseCase(repo wishlistRepo, maxItems int) *AddToWishlistUseCase {
    return &AddToWishlistUseCase{repo: repo, maxItems: maxItems}
}

func (uc *AddToWishlistUseCase) Execute(ctx context.Context, userID, productID, skuID string) error {
    count, err := uc.repo.CountByUser(ctx, userID)
    if err != nil {
        return err
    }
    if count >= uc.maxItems {
        return ErrWishlistFull
    }
    return uc.repo.Add(ctx, userID, productID, skuID)
}
```

- [ ] **Step 4: Run tests**

```bash
cd services/wishlist-service
go test ./internal/application/... -v
```

Expected: `PASS`

- [ ] **Step 5: Commit**

```bash
git add services/wishlist-service/
git commit -m "feat(wishlist): AddToWishlist use case, 200-item limit, wishlist_items table"
```

---

## Verification Checklist

- [ ] Submit a review without a confirmed order → `ErrNotVerifiedPurchase`
- [ ] Submit a review with confirmed order → stored with `PENDING_MODERATION`
- [ ] Admin approves review → status changes to `PUBLISHED`, appears in product listing
- [ ] Apply coupon `SAVE10` to ₹1000 cart → ₹900 final price
- [ ] Apply expired coupon → `ErrCouponExpired`
- [ ] Apply coupon below minimum order → `ErrCartBelowMinimum`
- [ ] Place order → `order.confirmed` event → logistics-service creates shipment → AWB stored
- [ ] `GET /v1/shipments/:order_id/tracking` returns tracking events
- [ ] Place order → SMS received on buyer phone (if Twilio configured)
- [ ] Add 200 items to wishlist → 201st returns `ErrWishlistFull`
