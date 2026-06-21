# Phase 1 — Testing Foundation

**Goal:** Establish a baseline of automated tests before any further feature work. Every subsequent phase must not regress this baseline.

**Why first:** The codebase has zero test files. The complex distributed paths (saga, Lua reserve, outbox, idempotency) are entirely untested. Changes to payment, inventory, and order flows are high-risk without a safety net.

---

## 1.1 Service-Layer Unit Tests

Each service package has an interface defined for its repository dependency (e.g., `contracts.InventoryRepository`, `contracts.PaymentRepository`). These are the natural seam for unit tests with mock repositories.

### Target coverage

| Service | Key scenarios |
|---------|--------------|
| `inventory-service` | AddStock warms/skips Redis; ReserveStock Lua path, cache-miss path, DB-fallback path; ReleaseStock Redis increment; concurrent reserve race (two goroutines, one loses) |
| `payment-service` | ChargeCard idempotency replay (cache hit, DB hit, lock conflict); charge failure does not cache; RefundPayment status guard; HandleCaptureWebhook idempotency |
| `order-management-service` | Checkout saga success path; saga compensation when payment fails; saga compensation when inventory reserve fails; duplicate checkout with same idempotency key |
| `auth-service` | RegisterUserPassword happy path; duplicate email; LoginPassword wrong password; RefreshAccessToken expired token; Logout blacklists token |
| `product-catalog-service` | GetProductList filter/sort/pagination; UpdateProduct optimistic lock conflict (rows=0 → 409); slug conflict |
| `notification-service` | buildNotification for each event type; missing seller_id on inventory.depleted; duplicate outbox_id dedup |

### Tooling

- Use `gomock` (already referenced in `product-catalog-service` with `//go:generate mockgen`). Run `go generate ./...` per service to produce mocks.
- One `_test.go` file per service file, same package (`package service`).
- No network, no DB, no Redis in unit tests.

---

## 1.2 Repository Integration Tests

Test actual SQL against a real PostgreSQL instance. The test DB is already available via `docker-compose.yml`.

### Approach

- Use `testcontainers-go` **or** a dedicated `TEST_DB_DSN` env var pointing at a local test schema.
- Run migrations against the test DB before the test suite (`go test -run TestMain`).
- Each test wraps operations in a transaction and rolls back in `t.Cleanup` — no persistent state between tests.

### Target scenarios

| Repository | Key scenarios |
|------------|--------------|
| `inventory` | AddStock upsert; ReserveStock insufficient qty returns nil; DeductStock transitions status; ReleaseStock on already-released returns conflict |
| `payment` | CreatePayment; MarkCaptured writes ledger + outbox; MarkFailed writes outbox; CreateRefund writes refund + outbox; GetByIdempotencyKey |
| `order` | CreateOrder + items in one tx; CreateOutbox; GetOrders with filters |
| `auth` | CreateUser; duplicate email 23505; GetUserByEmail; CreateRefreshToken; MarkTokenUsed |
| `product-catalog` | CreateProduct slug conflict; UpdateProduct optimistic lock; GetProductList all filter combos |

---

## 1.3 HTTP Handler Tests

Use `net/http/httptest` to drive handlers end-to-end through the HTTP layer with a real (but in-memory mock) service.

### Target scenarios per handler

- Happy path returns correct status + body shape
- Missing auth header → 401
- Malformed JSON body → 400
- Service returns `pkgerrors.NotFound` → 404
- Service returns `pkgerrors.Conflict` → 409
- Service returns `pkgerrors.Validation` → 400

---

## 1.4 CI Integration

Add a GitHub Actions workflow (`.github/workflows/test.yml`):

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-alpine
        env: { POSTGRES_USER: zapuser, POSTGRES_PASSWORD: zappass123 }
      redis:
        image: redis:7-alpine
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.23' }
      - run: go work sync
      - run: go test ./... -count=1 -race -timeout 120s
```

---

## Acceptance criteria

- `go test ./...` passes across all six service modules
- No test uses `time.Sleep` for synchronization (use channels or mock clocks)
- Coverage ≥ 70% for all `internal/service` packages
- CI green on every PR

---

## Estimated effort

3–4 weeks (one developer), dominated by mocking and integration test infrastructure setup.
