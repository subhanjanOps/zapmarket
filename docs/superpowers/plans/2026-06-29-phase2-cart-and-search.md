# Phase 2 — Cart Service & Search Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Cart domain (guest + authenticated carts, merge on login, price re-validation at checkout) and a full-text search service backed by Typesense.

**Architecture:** Cart service is a new Go microservice (port 8087, gRPC 50057) with Redis for guest carts (TTL 7 days) and PostgreSQL for authenticated carts. It consumes `product.updated` Kafka events to invalidate stale prices. Search service is a catalog-sync-worker that consumes `product.created/updated` events and indexes into Typesense; the product-catalog-service delegates search queries to Typesense via a new `/search` endpoint.

**Tech Stack:** Go 1.25, Typesense v26, `typesense-go` client, existing `pkg/kafka`, existing `pkg/redis`, existing `pkg/database`.

## Global Constraints

- New service ports: cart-service HTTP 8087 / gRPC 50057
- Typesense runs on port 8108, API key via `TYPESENSE_API_KEY` env var
- Cart items store `price_at_add` (snapshot) — price shown in cart is always `price_at_add`; checkout re-validates against current SKU price
- Guest cart key format in Redis: `cart:guest:{session_id}` with 7-day TTL
- Authenticated cart stored in PostgreSQL `cart_items` table
- All Kafka topic constants added to `pkg/kafka/topics.go`
- Typesense collection name: `products`

---

## File Map

### Cart Service (new microservice)
```
services/cart-service/
  cmd/server/main.go
  internal/
    domain/
      cart.go               — Cart, CartItem, GuestCartKey value objects
      errors/errors.go      — ErrCartNotFound, ErrItemNotFound
    application/
      usecases/
        add_item.go
        remove_item.go
        get_cart.go
        merge_cart.go       — guest → authenticated on login
        validate_prices.go  — re-check current SKU prices
      interfaces.go         — use case interfaces
    infrastructure/
      redis/guest_cart.go   — Redis-backed guest cart
      postgres/cart_repo.go — authenticated cart
      grpc/catalog_client.go — fetches current SKU prices for validation
    interfaces/
      http/cart_handler.go
      grpc/cart_server.go
      consumers/product_consumer.go — invalidates on product.updated
  migrations/
    0001_init.up.sql
    0001_init.down.sql
  proto/cart.proto
```

### Catalog Sync Worker (new binary inside product-catalog-service)
```
services/product-catalog-service/
  cmd/sync-worker/main.go   — NEW entry point
  internal/
    search/
      typesense_client.go   — index/update/delete documents
      typesense_client_test.go
      sync_worker.go        — Kafka consumer → Typesense
      sync_worker_test.go
  internal/handler/http/
    search_handler.go       — NEW: GET /v1/products/search?q=...&category=...
    search_handler_test.go
```

---

## Task 1: Cart Service — Domain & Migrations

**Files:**
- Create: `services/cart-service/internal/domain/cart.go`
- Create: `services/cart-service/internal/domain/errors/errors.go`
- Create: `services/cart-service/migrations/0001_init.up.sql`
- Create: `services/cart-service/migrations/0001_init.down.sql`

**Interfaces:**
- Produces: `Cart`, `CartItem` structs; `ErrCartNotFound`, `ErrItemOutOfDate`

- [ ] **Step 1: Create cart domain types**

Create `services/cart-service/internal/domain/cart.go`:

```go
package domain

import "time"

type Cart struct {
    UserID    string
    Items     []CartItem
    UpdatedAt time.Time
}

type CartItem struct {
    SKUID        string    `json:"sku_id"`
    ProductID    string    `json:"product_id"`
    ProductName  string    `json:"product_name"`
    VariantAttrs map[string]string `json:"variant_attrs"`
    Quantity     int       `json:"quantity"`
    PriceAtAdd   int64     `json:"price_at_add"`  // cents, snapshot at time of add
    Currency     string    `json:"currency"`
    ImageURL     string    `json:"image_url"`
    AddedAt      time.Time `json:"added_at"`
}

// PriceStale returns true if the current live price differs from what was captured at add time.
func (i *CartItem) PriceStale(currentPriceCents int64) bool {
    return i.PriceAtAdd != currentPriceCents
}
```

- [ ] **Step 2: Create domain errors**

Create `services/cart-service/internal/domain/errors/errors.go`:

```go
package errors

import "errors"

var (
    ErrCartNotFound  = errors.New("cart not found")
    ErrItemNotFound  = errors.New("item not found in cart")
    ErrOutOfStock    = errors.New("sku is out of stock")
    ErrPriceChanged  = errors.New("price has changed since item was added to cart")
)
```

- [ ] **Step 3: Write cart_items migration**

Create `services/cart-service/migrations/0001_init.up.sql`:

```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE cart_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    sku_id          UUID NOT NULL,
    product_id      UUID NOT NULL,
    product_name    TEXT NOT NULL,
    variant_attrs   JSONB NOT NULL DEFAULT '{}',
    quantity        INT NOT NULL CHECK (quantity > 0),
    price_at_add    BIGINT NOT NULL,  -- cents
    currency        VARCHAR(3) NOT NULL DEFAULT 'INR',
    image_url       TEXT,
    added_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, sku_id)
);

CREATE INDEX idx_cart_items_user ON cart_items(user_id);
```

Create `services/cart-service/migrations/0001_init.down.sql`:

```sql
DROP TABLE IF EXISTS cart_items;
```

- [ ] **Step 4: Commit**

```bash
git add services/cart-service/internal/domain/ services/cart-service/migrations/
git commit -m "feat(cart): domain types, cart_items migration"
```

---

## Task 2: Cart Service — Add, Remove, Get Use Cases

**Files:**
- Create: `services/cart-service/internal/application/usecases/add_item.go`
- Create: `services/cart-service/internal/application/usecases/remove_item.go`
- Create: `services/cart-service/internal/application/usecases/get_cart.go`
- Create: `services/cart-service/internal/application/interfaces.go`
- Create: `services/cart-service/internal/application/usecases/add_item_test.go`

**Interfaces:**
- Produces: `AddItemUseCase`, `RemoveItemUseCase`, `GetCartUseCase`

- [ ] **Step 1: Define repository and SKU fetcher interfaces**

Create `services/cart-service/internal/application/interfaces.go`:

```go
package application

import (
    "context"
    "github.com/zapmarket/cart-service/internal/domain"
)

type CartRepository interface {
    GetCart(ctx context.Context, userID string) (*domain.Cart, error)
    UpsertItem(ctx context.Context, userID string, item domain.CartItem) error
    RemoveItem(ctx context.Context, userID, skuID string) error
    ClearCart(ctx context.Context, userID string) error
}

type SKUFetcher interface {
    GetSKUPrice(ctx context.Context, skuID string) (priceCents int64, currency string, inStock bool, err error)
}
```

- [ ] **Step 2: Write the failing test for AddItemUseCase**

Create `services/cart-service/internal/application/usecases/add_item_test.go`:

```go
package usecases_test

import (
    "context"
    "testing"

    "github.com/zapmarket/cart-service/internal/application"
    "github.com/zapmarket/cart-service/internal/application/usecases"
    "github.com/zapmarket/cart-service/internal/domain"
    domainerrors "github.com/zapmarket/cart-service/internal/domain/errors"
)

type fakeCartRepo struct {
    cart   *domain.Cart
    upserted domain.CartItem
}

func (f *fakeCartRepo) GetCart(_ context.Context, _ string) (*domain.Cart, error) {
    if f.cart == nil {
        return &domain.Cart{}, nil
    }
    return f.cart, nil
}

func (f *fakeCartRepo) UpsertItem(_ context.Context, _ string, item domain.CartItem) error {
    f.upserted = item
    return nil
}

func (f *fakeCartRepo) RemoveItem(_ context.Context, _, _ string) error { return nil }
func (f *fakeCartRepo) ClearCart(_ context.Context, _ string) error     { return nil }

type fakeSKUFetcher struct {
    price    int64
    currency string
    inStock  bool
}

func (f *fakeSKUFetcher) GetSKUPrice(_ context.Context, _ string) (int64, string, bool, error) {
    return f.price, f.currency, f.inStock, nil
}

func TestAddItem_SnapshotsPriceAtAddTime(t *testing.T) {
    repo := &fakeCartRepo{}
    sku := &fakeSKUFetcher{price: 4999, currency: "INR", inStock: true}
    uc := usecases.NewAddItemUseCase(repo, sku)

    err := uc.Execute(context.Background(), "user-1", domain.CartItem{
        SKUID:    "sku-1",
        Quantity: 1,
    })
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if repo.upserted.PriceAtAdd != 4999 {
        t.Fatalf("expected price snapshot 4999, got %d", repo.upserted.PriceAtAdd)
    }
}

func TestAddItem_RejectsOutOfStockSKU(t *testing.T) {
    repo := &fakeCartRepo{}
    sku := &fakeSKUFetcher{price: 100, currency: "INR", inStock: false}
    uc := usecases.NewAddItemUseCase(repo, sku)

    err := uc.Execute(context.Background(), "user-1", domain.CartItem{SKUID: "sku-2", Quantity: 1})
    if err != domainerrors.ErrOutOfStock {
        t.Fatalf("expected ErrOutOfStock, got %v", err)
    }
}
```

- [ ] **Step 3: Run to confirm failure**

```bash
cd services/cart-service
go test ./internal/application/usecases/... -v -run TestAddItem
```

Expected: `FAIL — cannot find package "usecases"`

- [ ] **Step 4: Implement AddItemUseCase**

Create `services/cart-service/internal/application/usecases/add_item.go`:

```go
package usecases

import (
    "context"
    "time"

    "github.com/zapmarket/cart-service/internal/application"
    "github.com/zapmarket/cart-service/internal/domain"
    domainerrors "github.com/zapmarket/cart-service/internal/domain/errors"
)

type AddItemUseCase struct {
    repo application.CartRepository
    sku  application.SKUFetcher
}

func NewAddItemUseCase(repo application.CartRepository, sku application.SKUFetcher) *AddItemUseCase {
    return &AddItemUseCase{repo: repo, sku: sku}
}

func (uc *AddItemUseCase) Execute(ctx context.Context, userID string, item domain.CartItem) error {
    price, currency, inStock, err := uc.sku.GetSKUPrice(ctx, item.SKUID)
    if err != nil {
        return err
    }
    if !inStock {
        return domainerrors.ErrOutOfStock
    }
    item.PriceAtAdd = price
    item.Currency = currency
    item.AddedAt = time.Now()
    return uc.repo.UpsertItem(ctx, userID, item)
}
```

- [ ] **Step 5: Implement GetCartUseCase and RemoveItemUseCase**

Create `services/cart-service/internal/application/usecases/get_cart.go`:

```go
package usecases

import (
    "context"
    "github.com/zapmarket/cart-service/internal/application"
    "github.com/zapmarket/cart-service/internal/domain"
)

type GetCartUseCase struct{ repo application.CartRepository }

func NewGetCartUseCase(repo application.CartRepository) *GetCartUseCase {
    return &GetCartUseCase{repo: repo}
}

func (uc *GetCartUseCase) Execute(ctx context.Context, userID string) (*domain.Cart, error) {
    return uc.repo.GetCart(ctx, userID)
}
```

Create `services/cart-service/internal/application/usecases/remove_item.go`:

```go
package usecases

import (
    "context"
    "github.com/zapmarket/cart-service/internal/application"
)

type RemoveItemUseCase struct{ repo application.CartRepository }

func NewRemoveItemUseCase(repo application.CartRepository) *RemoveItemUseCase {
    return &RemoveItemUseCase{repo: repo}
}

func (uc *RemoveItemUseCase) Execute(ctx context.Context, userID, skuID string) error {
    return uc.repo.RemoveItem(ctx, userID, skuID)
}
```

- [ ] **Step 6: Run tests**

```bash
cd services/cart-service
go test ./internal/application/... -v
```

Expected: `PASS`

- [ ] **Step 7: Commit**

```bash
git add services/cart-service/internal/application/
git commit -m "feat(cart): AddItem, RemoveItem, GetCart use cases with out-of-stock guard and price snapshot"
```

---

## Task 3: Cart Service — Merge Guest Cart on Login

**Files:**
- Create: `services/cart-service/internal/application/usecases/merge_cart.go`
- Create: `services/cart-service/internal/application/usecases/merge_cart_test.go`

**Interfaces:**
- Consumes: `GuestCartRepository` (Redis-backed), `CartRepository` (Postgres-backed)
- Produces: `MergeCartUseCase.Execute(ctx, sessionID, userID string) error`

- [ ] **Step 1: Write the failing test**

Create `services/cart-service/internal/application/usecases/merge_cart_test.go`:

```go
package usecases_test

import (
    "context"
    "testing"

    "github.com/zapmarket/cart-service/internal/application/usecases"
    "github.com/zapmarket/cart-service/internal/domain"
)

type fakeGuestRepo struct {
    cart    *domain.Cart
    cleared bool
}

func (f *fakeGuestRepo) GetGuestCart(_ context.Context, _ string) (*domain.Cart, error) {
    return f.cart, nil
}
func (f *fakeGuestRepo) ClearGuestCart(_ context.Context, _ string) error {
    f.cleared = true
    return nil
}

func TestMergeCart_MovesGuestItemsToAuthCart(t *testing.T) {
    guest := &fakeGuestRepo{
        cart: &domain.Cart{
            Items: []domain.CartItem{
                {SKUID: "sku-1", Quantity: 2, PriceAtAdd: 500, Currency: "INR"},
            },
        },
    }
    auth := &fakeCartRepo{}
    sku := &fakeSKUFetcher{price: 500, currency: "INR", inStock: true}

    uc := usecases.NewMergeCartUseCase(guest, auth, sku)
    if err := uc.Execute(context.Background(), "session-abc", "user-1"); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if auth.upserted.SKUID != "sku-1" {
        t.Fatalf("expected sku-1 merged, got %q", auth.upserted.SKUID)
    }
    if !guest.cleared {
        t.Fatal("expected guest cart to be cleared after merge")
    }
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
cd services/cart-service
go test ./internal/application/usecases/... -v -run TestMergeCart
```

Expected: `FAIL`

- [ ] **Step 3: Implement MergeCartUseCase**

Create `services/cart-service/internal/application/usecases/merge_cart.go`:

```go
package usecases

import (
    "context"

    "github.com/zapmarket/cart-service/internal/application"
    "github.com/zapmarket/cart-service/internal/domain"
)

type guestCartReader interface {
    GetGuestCart(ctx context.Context, sessionID string) (*domain.Cart, error)
    ClearGuestCart(ctx context.Context, sessionID string) error
}

type MergeCartUseCase struct {
    guest guestCartReader
    auth  application.CartRepository
    sku   application.SKUFetcher
}

func NewMergeCartUseCase(guest guestCartReader, auth application.CartRepository, sku application.SKUFetcher) *MergeCartUseCase {
    return &MergeCartUseCase{guest: guest, auth: auth, sku: sku}
}

func (uc *MergeCartUseCase) Execute(ctx context.Context, sessionID, userID string) error {
    guestCart, err := uc.guest.GetGuestCart(ctx, sessionID)
    if err != nil || len(guestCart.Items) == 0 {
        return err
    }
    for _, item := range guestCart.Items {
        // re-validate price at merge time
        price, currency, inStock, err := uc.sku.GetSKUPrice(ctx, item.SKUID)
        if err != nil || !inStock {
            continue // skip unavailable items silently
        }
        item.PriceAtAdd = price
        item.Currency = currency
        if err := uc.auth.UpsertItem(ctx, userID, item); err != nil {
            return err
        }
    }
    return uc.guest.ClearGuestCart(ctx, sessionID)
}
```

- [ ] **Step 4: Run tests**

```bash
cd services/cart-service
go test ./internal/application/... -v
```

Expected: `PASS`

- [ ] **Step 5: Commit**

```bash
git add services/cart-service/internal/application/usecases/merge_cart.go \
        services/cart-service/internal/application/usecases/merge_cart_test.go
git commit -m "feat(cart): merge guest cart into authenticated cart on login, re-validates prices"
```

---

## Task 4: Cart Service — HTTP Handler & Wiring

**Files:**
- Create: `services/cart-service/internal/interfaces/http/cart_handler.go`
- Create: `services/cart-service/internal/infrastructure/postgres/cart_repo.go`
- Create: `services/cart-service/internal/infrastructure/redis/guest_cart.go`
- Create: `services/cart-service/cmd/server/main.go`
- Modify: `docker-compose.yml` — add cart-service

**Interfaces:**
- Produces: REST endpoints `GET /v1/cart`, `POST /v1/cart/items`, `DELETE /v1/cart/items/:sku_id`, `POST /v1/cart/merge`

- [ ] **Step 1: Implement HTTP handler**

Create `services/cart-service/internal/interfaces/http/cart_handler.go`:

```go
package http

import (
    "encoding/json"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/zapmarket/cart-service/internal/application/usecases"
    "github.com/zapmarket/cart-service/internal/domain"
)

type CartHandler struct {
    addItem    *usecases.AddItemUseCase
    removeItem *usecases.RemoveItemUseCase
    getCart    *usecases.GetCartUseCase
    mergeCart  *usecases.MergeCartUseCase
}

func NewCartHandler(add *usecases.AddItemUseCase, remove *usecases.RemoveItemUseCase, get *usecases.GetCartUseCase, merge *usecases.MergeCartUseCase) *CartHandler {
    return &CartHandler{addItem: add, removeItem: remove, getCart: get, mergeCart: merge}
}

func (h *CartHandler) Routes() http.Handler {
    r := chi.NewRouter()
    r.Get("/", h.getCartHandler)
    r.Post("/items", h.addItemHandler)
    r.Delete("/items/{skuID}", h.removeItemHandler)
    r.Post("/merge", h.mergeHandler)
    return r
}

func (h *CartHandler) getCartHandler(w http.ResponseWriter, r *http.Request) {
    userID := r.Header.Get("X-User-ID") // set by auth middleware
    cart, err := h.getCart.Execute(r.Context(), userID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(cart)
}

func (h *CartHandler) addItemHandler(w http.ResponseWriter, r *http.Request) {
    userID := r.Header.Get("X-User-ID")
    var item domain.CartItem
    if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
        http.Error(w, "invalid body", http.StatusBadRequest)
        return
    }
    if err := h.addItem.Execute(r.Context(), userID, item); err != nil {
        http.Error(w, err.Error(), http.StatusUnprocessableEntity)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}

func (h *CartHandler) removeItemHandler(w http.ResponseWriter, r *http.Request) {
    userID := r.Header.Get("X-User-ID")
    skuID := chi.URLParam(r, "skuID")
    if err := h.removeItem.Execute(r.Context(), userID, skuID); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}

func (h *CartHandler) mergeHandler(w http.ResponseWriter, r *http.Request) {
    userID := r.Header.Get("X-User-ID")
    sessionID := r.Header.Get("X-Session-ID")
    if err := h.mergeCart.Execute(r.Context(), sessionID, userID); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 2: Add cart-service to docker-compose.yml**

In `docker-compose.yml`, under `services:`, add:

```yaml
  cart-service:
    build: ./services/cart-service
    ports:
      - "8087:8087"
    environment:
      - HTTP_PORT=8087
      - DB_HOST=postgres
      - DB_NAME=cart
      - DB_USER=zapuser
      - DB_PASSWORD=zappass123
      - REDIS_ADDR=redis:6379
      - CATALOG_GRPC_ADDR=product-catalog-service:50052
    depends_on:
      - postgres
      - redis
```

Also add `cart` to the PostgreSQL databases list.

- [ ] **Step 3: Full build**

```bash
cd services/cart-service
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add services/cart-service/ docker-compose.yml
git commit -m "feat(cart): HTTP handler, postgres repo, redis guest cart, docker-compose wiring"
```

---

## Task 5: Search — Typesense Setup & Catalog Sync Worker

**Files:**
- Modify: `docker-compose.yml` — add Typesense
- Modify: `services/product-catalog-service/go.mod` — add typesense-go
- Create: `services/product-catalog-service/internal/search/typesense_client.go`
- Create: `services/product-catalog-service/internal/search/typesense_client_test.go`
- Create: `services/product-catalog-service/internal/search/sync_worker.go`
- Create: `services/product-catalog-service/internal/search/sync_worker_test.go`
- Create: `services/product-catalog-service/cmd/sync-worker/main.go`

**Interfaces:**
- Produces: `SearchClient.IndexProduct(ctx, doc SearchDoc)`, `SearchClient.Search(ctx, query SearchQuery) ([]SearchDoc, error)`

- [ ] **Step 1: Add Typesense to docker-compose.yml**

```yaml
  typesense:
    image: typesense/typesense:26.0
    ports:
      - "8108:8108"
    volumes:
      - typesense-data:/data
    command: --data-dir /data --api-key=${TYPESENSE_API_KEY:-xyz-local-key} --enable-cors
```

Add `typesense-data:` to the `volumes:` section.

- [ ] **Step 2: Add typesense-go dependency**

```bash
cd services/product-catalog-service
go get github.com/typesense/typesense-go/v2@latest
go mod tidy
```

- [ ] **Step 3: Write the failing test for the Typesense client**

Create `services/product-catalog-service/internal/search/typesense_client_test.go`:

```go
package search_test

import (
    "context"
    "testing"

    "github.com/zapmarket/product-catalog-service/internal/search"
)

type fakeTypesenseAPI struct {
    indexed []search.SearchDoc
    deleted []string
}

func (f *fakeTypesenseAPI) Upsert(_ context.Context, doc search.SearchDoc) error {
    f.indexed = append(f.indexed, doc)
    return nil
}

func (f *fakeTypesenseAPI) Delete(_ context.Context, id string) error {
    f.deleted = append(f.deleted, id)
    return nil
}

func TestTypesenseClient_IndexProduct(t *testing.T) {
    api := &fakeTypesenseAPI{}
    client := search.NewTypesenseClient(api)

    doc := search.SearchDoc{
        ID:           "prod-1",
        Name:         "iPhone 15",
        CategoryID:   "cat-1",
        CategoryName: "Mobiles",
        Description:  "Latest Apple phone",
        PriceCents:   7999900,
        Currency:     "INR",
        SellerID:     "sel-1",
        Status:       "ACTIVE",
    }

    if err := client.IndexProduct(context.Background(), doc); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(api.indexed) != 1 || api.indexed[0].ID != "prod-1" {
        t.Fatal("expected product to be indexed")
    }
}
```

- [ ] **Step 4: Run to confirm failure**

```bash
cd services/product-catalog-service
go test ./internal/search/... -v
```

Expected: `FAIL — cannot find package "search"`

- [ ] **Step 5: Implement the search client**

Create `services/product-catalog-service/internal/search/typesense_client.go`:

```go
package search

import "context"

type SearchDoc struct {
    ID           string `json:"id"`
    Name         string `json:"name"`
    CategoryID   string `json:"category_id"`
    CategoryName string `json:"category_name"`
    Description  string `json:"description"`
    PriceCents   int64  `json:"price_cents"`
    Currency     string `json:"currency"`
    SellerID     string `json:"seller_id"`
    Status       string `json:"status"`
    ImageURL     string `json:"image_url"`
}

type SearchQuery struct {
    Q          string
    CategoryID string
    MinPrice   int64
    MaxPrice   int64
    Page       int
    PerPage    int
}

type typesenseAPI interface {
    Upsert(ctx context.Context, doc SearchDoc) error
    Delete(ctx context.Context, id string) error
}

type TypesenseClient struct{ api typesenseAPI }

func NewTypesenseClient(api typesenseAPI) *TypesenseClient {
    return &TypesenseClient{api: api}
}

func (c *TypesenseClient) IndexProduct(ctx context.Context, doc SearchDoc) error {
    return c.api.Upsert(ctx, doc)
}

func (c *TypesenseClient) DeleteProduct(ctx context.Context, id string) error {
    return c.api.Delete(ctx, id)
}
```

- [ ] **Step 6: Implement the real Typesense API adapter**

Create `services/product-catalog-service/internal/search/typesense_adapter.go`:

```go
package search

import (
    "context"

    "github.com/typesense/typesense-go/v2/typesense"
    "github.com/typesense/typesense-go/v2/typesense/api"
)

type TypesenseAdapter struct{ client *typesense.Client }

func NewTypesenseAdapter(host, apiKey string) *TypesenseAdapter {
    c := typesense.NewClient(
        typesense.WithServer(host),
        typesense.WithAPIKey(apiKey),
    )
    return &TypesenseAdapter{client: c}
}

func (a *TypesenseAdapter) EnsureSchema(ctx context.Context) error {
    schema := &api.CollectionSchema{
        Name: "products",
        Fields: []api.Field{
            {Name: "id", Type: "string"},
            {Name: "name", Type: "string"},
            {Name: "category_id", Type: "string", Facet: boolPtr(true)},
            {Name: "category_name", Type: "string", Facet: boolPtr(true)},
            {Name: "description", Type: "string"},
            {Name: "price_cents", Type: "int64"},
            {Name: "currency", Type: "string"},
            {Name: "seller_id", Type: "string"},
            {Name: "status", Type: "string", Facet: boolPtr(true)},
            {Name: "image_url", Type: "string", Optional: boolPtr(true)},
        },
        DefaultSortingField: strPtr("price_cents"),
    }
    _, err := a.client.Collections().Create(ctx, schema)
    return err // ignore AlreadyExists
}

func (a *TypesenseAdapter) Upsert(ctx context.Context, doc SearchDoc) error {
    _, err := a.client.Collection("products").Documents().Upsert(ctx, doc)
    return err
}

func (a *TypesenseAdapter) Delete(ctx context.Context, id string) error {
    _, err := a.client.Collection("products").Document(id).Delete(ctx)
    return err
}

func boolPtr(b bool) *bool { return &b }
func strPtr(s string) *string { return &s }
```

- [ ] **Step 7: Implement the sync worker**

Create `services/product-catalog-service/internal/search/sync_worker.go`:

```go
package search

import (
    "context"
    "encoding/json"

    "github.com/zapmarket/pkg/logger"
)

type productEvent struct {
    EventType string `json:"event_type"`
    ProductID string `json:"product_id"`
    Name      string `json:"name"`
    CategoryID string `json:"category_id"`
    CategoryName string `json:"category_name"`
    Description string `json:"description"`
    PriceCents int64  `json:"price_cents"`
    Currency   string `json:"currency"`
    SellerID   string `json:"seller_id"`
    Status     string `json:"status"`
    ImageURL   string `json:"image_url"`
}

type SyncWorker struct{ client *TypesenseClient }

func NewSyncWorker(client *TypesenseClient) *SyncWorker {
    return &SyncWorker{client: client}
}

func (w *SyncWorker) Handle(ctx context.Context, payload []byte) error {
    var evt productEvent
    if err := json.Unmarshal(payload, &evt); err != nil {
        return err
    }

    switch evt.EventType {
    case "product.created", "product.updated":
        doc := SearchDoc{
            ID:           evt.ProductID,
            Name:         evt.Name,
            CategoryID:   evt.CategoryID,
            CategoryName: evt.CategoryName,
            Description:  evt.Description,
            PriceCents:   evt.PriceCents,
            Currency:     evt.Currency,
            SellerID:     evt.SellerID,
            Status:       evt.Status,
            ImageURL:     evt.ImageURL,
        }
        return w.client.IndexProduct(ctx, doc)
    case "product.deleted":
        return w.client.DeleteProduct(ctx, evt.ProductID)
    default:
        logger.Warn(ctx, "sync worker: unknown event type", "event_type", evt.EventType)
        return nil
    }
}
```

- [ ] **Step 8: Run tests**

```bash
cd services/product-catalog-service
go test ./internal/search/... -v
```

Expected: `PASS`

- [ ] **Step 9: Commit**

```bash
git add services/product-catalog-service/internal/search/ \
        services/product-catalog-service/go.mod \
        services/product-catalog-service/go.sum \
        docker-compose.yml
git commit -m "feat(search): Typesense client, catalog sync worker consuming product events"
```

---

## Task 6: Search HTTP Endpoint in product-catalog-service

**Files:**
- Create: `services/product-catalog-service/internal/handler/http/search_handler.go`
- Create: `services/product-catalog-service/internal/handler/http/search_handler_test.go`
- Modify: `services/product-catalog-service/main.go` — register `/v1/products/search`

**Interfaces:**
- Produces: `GET /v1/products/search?q=iphone&category_id=cat-1&min_price=1000&max_price=50000&page=1&per_page=20`

- [ ] **Step 1: Write failing test**

Create `services/product-catalog-service/internal/handler/http/search_handler_test.go`:

```go
package http_test

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    handler "github.com/zapmarket/product-catalog-service/internal/handler/http"
    "github.com/zapmarket/product-catalog-service/internal/search"
)

type fakeSearcher struct{ results []search.SearchDoc }

func (f *fakeSearcher) Search(_ context.Context, _ search.SearchQuery) ([]search.SearchDoc, error) {
    return f.results, nil
}

func TestSearchHandler_Returns200WithResults(t *testing.T) {
    searcher := &fakeSearcher{results: []search.SearchDoc{{ID: "p1", Name: "iPhone"}}}
    h := handler.NewSearchHandler(searcher)

    req := httptest.NewRequest(http.MethodGet, "/v1/products/search?q=iphone", nil)
    rec := httptest.NewRecorder()
    h.Search(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rec.Code)
    }
}
```

- [ ] **Step 2: Implement search handler**

Create `services/product-catalog-service/internal/handler/http/search_handler.go`:

```go
package http

import (
    "context"
    "encoding/json"
    "net/http"
    "strconv"

    "github.com/zapmarket/product-catalog-service/internal/search"
)

type searcher interface {
    Search(ctx context.Context, q search.SearchQuery) ([]search.SearchDoc, error)
}

type SearchHandler struct{ searcher searcher }

func NewSearchHandler(s searcher) *SearchHandler { return &SearchHandler{searcher: s} }

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
    q := r.URL.Query()
    minPrice, _ := strconv.ParseInt(q.Get("min_price"), 10, 64)
    maxPrice, _ := strconv.ParseInt(q.Get("max_price"), 10, 64)
    page, _ := strconv.Atoi(q.Get("page"))
    if page < 1 { page = 1 }
    perPage, _ := strconv.Atoi(q.Get("per_page"))
    if perPage < 1 || perPage > 100 { perPage = 20 }

    results, err := h.searcher.Search(r.Context(), search.SearchQuery{
        Q:          q.Get("q"),
        CategoryID: q.Get("category_id"),
        MinPrice:   minPrice,
        MaxPrice:   maxPrice,
        Page:       page,
        PerPage:    perPage,
    })
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{"results": results, "page": page, "per_page": perPage})
}
```

- [ ] **Step 3: Register route in main.go**

In `services/product-catalog-service/main.go`, add:

```go
searchHandler := httphandler.NewSearchHandler(typesenseClient)
r.Get("/v1/products/search", searchHandler.Search)
```

- [ ] **Step 4: Run all tests**

```bash
cd services/product-catalog-service
go test ./... -v
```

Expected: `PASS`

- [ ] **Step 5: Commit**

```bash
git add services/product-catalog-service/internal/handler/http/search_handler.go \
        services/product-catalog-service/internal/handler/http/search_handler_test.go \
        services/product-catalog-service/main.go
git commit -m "feat(catalog): GET /v1/products/search — full-text + faceted search via Typesense"
```

---

## Verification Checklist

- [ ] `docker compose up -d` — all services healthy including Typesense
- [ ] Create a product → verify Kafka `product.created` event is consumed by sync-worker → document appears in Typesense (`curl http://localhost:8108/collections/products/documents/search?q=*&query_by=name -H "X-TYPESENSE-API-KEY: xyz-local-key"`)
- [ ] `GET /v1/products/search?q=iphone` returns matching products
- [ ] `POST /v1/cart/items` with valid SKU → cart stored, price snapshotted
- [ ] Login with guest session → `POST /v1/cart/merge` → guest items appear in authenticated cart, Redis key deleted
- [ ] Update a product price → cart item shows old `price_at_add` (snapshot preserved)
