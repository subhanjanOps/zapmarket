# Backoffice UI — Phase 2: Users, Orders, Seller Verification, Bulk Import/Export

**Status: 📋 Planned (2026-06-18)**

Extends `services/backoffice-ui` (Phase 1 plan: `admin-catalog-ui.md`) with four
features that each require backend work before the UI can be built. Phase 1 (catalog
management) was entirely frontend work against existing APIs. Phase 2 is mixed:
new backend endpoints first, then new backoffice-ui pages.

---

## Feature overview

| Feature | Backend work | Frontend page | Priority |
|---|---|---|---|
| User Management | New admin endpoints in auth-service | `/dashboard/users` | High |
| Order Oversight | New admin endpoints in order-management-service | `/dashboard/orders` | High |
| Seller Verification | New `seller_status` column + admin endpoints + Kafka event | `/dashboard/sellers` | High |
| Bulk Import/Export | Export: none. Import: none (batch client-side calls) | modal in Categories + Products pages | Low |

---

## 1. User Management

### What exists

The `users` table and `UserRepository` already have everything needed:
- `GetUserByID`, `GetUserByEmail`, `UpdateUser`, `DeleteUser` all implemented
- `role` column with CHECK constraint (`buyer | seller | admin`)
- `is_verified` column, `deleted_at` soft-delete column

**Missing:** `ListUsers` method in the repository and contract. No admin HTTP endpoints.

### Backend work — auth-service

**New repository contract method** (`internal/domain/contracts/repositories.go`):
```go
ListUsers(ctx context.Context, params UserListParams) ([]*domain.User, int64, error)
```

```go
type UserListParams struct {
    Role      string   // filter by role; empty = all
    Search    string   // partial match on email or full_name
    Limit     int
    Offset    int
}
```

SQL: `SELECT ... FROM users WHERE deleted_at IS NULL [AND role = $n] [AND (email ILIKE $n OR full_name ILIKE $n)] ORDER BY created_at DESC LIMIT $n OFFSET $n`

**New admin HTTP handler** (`internal/handler/http/admin_handler.go`):

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/admin/users` | Paginated list — `?role=`, `?search=`, `?limit=`, `?offset=` |
| `GET` | `/v1/admin/users/{id}` | Single user detail |
| `PUT` | `/v1/admin/users/{id}/role` | Change role — body `{ "role": "admin" }` |
| `DELETE` | `/v1/admin/users/{id}` | Soft-delete (deactivate) — sets `deleted_at` |

**RBAC:** all four require `admin` JWT role. Wire in `cmd/main.go` with an
`AdminAuthMiddleware` that validates the JWT and checks `role == "admin"`.

**Response format:** use the existing `httpx.Paginated` for the list, `httpx.Success`
for single/mutation responses. Field set in list response: `id, email, full_name, role,
is_verified, created_at`.

**New gateway route** (insert into `gateway_routes`):
```sql
INSERT INTO gateway_routes (path_prefix, upstream, auth_mode, strip_prefix, enabled)
VALUES ('/v1/admin', 'auth-service', 'required', false, true);
```

### Frontend — `/dashboard/users`

```
[Page header] Users          [🔍 search email or name]   All | Buyer | Seller | Admin

┌──────────────────────────────────────────────────────────────────────────────────┐
│  Name            Email               Role      Verified  Joined     Actions      │
│  ──────────────────────────────────────────────────────────────────────────────  │
│  Test Seller     seller@example.com  seller    ✓         Jun 18     [Edit Role]  │
│                                                                      [Deactivate]│
└──────────────────────────────────────────────────────────────────────────────────┘
                                               Page 1 of 4   [← Prev]  [Next →]
```

**Edit Role modal:** dropdown `buyer | seller | admin` + confirm button.
`PUT /v1/admin/users/{id}/role`

**Deactivate:** confirm dialog ("This will immediately revoke the user's access."),
then `DELETE /v1/admin/users/{id}`.

No user creation — users self-register via auth service.

---

## 2. Order Oversight

### What exists

Order-management-service has:
- `ListOrders` (by user ID only — no admin all-orders query)
- `CancelOrder` (ownership-checked — user can only cancel their own orders)
- `GetBySellerID` (seller-scoped)

**Missing:** `ListAllOrders` (no user filter) and admin-scoped cancel (bypasses
ownership check).

### Backend work — order-management-service

**New repository method** (`internal/repository/order_repository.go`):
```go
func (r *OrderRepository) ListAll(ctx context.Context, params OrderListParams) ([]*domain.Order, int64, error)
```

```go
type OrderListParams struct {
    Status string
    UserID *uuid.UUID  // optional filter
    From   *time.Time
    To     *time.Time
    Limit  int
    Offset int
}
```

SQL: `SELECT ... FROM orders WHERE deleted_at IS NULL [AND status = $n] [AND user_id = $n] [AND created_at >= $n] [AND created_at <= $n] ORDER BY created_at DESC LIMIT $n OFFSET $n`

Add `COUNT(*) OVER()` window function or a separate count query for total.

**New service methods** (`internal/service/order_service.go`):
```go
ListAllOrders(ctx, params OrderListParams) ([]*domain.Order, int64, error)
AdminCancelOrder(ctx, orderID uuid.UUID) (*domain.Order, error)  // no ownership check
```

**New HTTP handler methods** in `order_handler.go`:

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/admin/orders` | All orders — `?status=`, `?user_id=`, `?from=`, `?to=`, `?limit=`, `?offset=` |
| `GET` | `/v1/admin/orders/{id}` | Single order + items (no ownership check) |
| `POST` | `/v1/admin/orders/{id}/cancel` | Force cancel any order — bypasses user ownership check |

All require `admin` role via `RequireRole("admin")` middleware.

Register in `main.go`:
```go
r.Route("/v1/admin/orders", func(r chi.Router) {
    r.Use(authMW.Authenticate)
    r.Use(authMW.RequireRole("admin"))
    r.Get("/", handler.AdminListOrders)
    r.Get("/{id}", handler.AdminGetOrder)
    r.Post("/{id}/cancel", handler.AdminCancelOrder)
})
```

**New gateway route:**
```sql
INSERT INTO gateway_routes (path_prefix, upstream, auth_mode, strip_prefix, enabled)
VALUES ('/v1/admin/orders', 'order-management-service', 'required', false, true);
```

> Note: The `/v1/admin` prefix on auth-service and `/v1/admin/orders` prefix on
> order-management-service are distinct routes — the gateway does prefix matching so
> both can coexist. Auth-service's `/v1/admin` catches `/v1/admin/users*`;
> order-management catches `/v1/admin/orders*`. Register the more-specific prefix
> (`/v1/admin/orders`) first so it takes priority.

### Frontend — `/dashboard/orders`

```
[Page header] Orders          [🔍 user_id]   All | PENDING | RESERVED | CONFIRMED | CANCELLED

  Date range: [from date] → [to date]

┌──────────────────────────────────────────────────────────────────────────────────────┐
│  Order ID       User              Total     Status      Date       Actions           │
│  abc123…        1a03bea2…         $120.00   CONFIRMED   Jun 18     [View]            │
│  def456…        2b14cbf3…         $45.00    PENDING     Jun 18     [View] [Cancel]   │
└──────────────────────────────────────────────────────────────────────────────────────┘
```

**Cancel** only shown for PENDING and RESERVED orders. Calls
`POST /v1/admin/orders/{id}/cancel`.

**View** → `/dashboard/orders/{id}` — full order detail: status timeline, line items
table, buyer user_id, payment_id. Same layout as seller-ui order detail but read-only
and without the seller-scoping check.

---

## 3. Seller Verification

### The problem with the current state

`is_verified` is set to `true` for all users at registration
(`auth-service/internal/service/auth_service.go` line 66). It was intended for email
verification but was auto-approved as a placeholder. Repurposing it for seller approval
would break buyers (who have no verification concept).

### Solution: `seller_status` column

A new nullable column on `users` — only meaningful when `role = 'seller'`.

**Migration** (`auth-service/migrations/0002_seller_status.up.sql`):
```sql
ALTER TABLE users ADD COLUMN IF NOT EXISTS
    seller_status VARCHAR(20)
    CHECK (seller_status IN ('PENDING', 'APPROVED', 'SUSPENDED'))
    DEFAULT NULL;

-- All existing sellers start PENDING (they were auto-approved before).
UPDATE users SET seller_status = 'APPROVED' WHERE role = 'seller' AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_seller_status
    ON users (seller_status) WHERE seller_status IS NOT NULL AND deleted_at IS NULL;
```

**Model update** (`internal/domain/models.go`):
```go
type User struct {
    ...
    SellerStatus *string  // nil for non-sellers; "PENDING" | "APPROVED" | "SUSPENDED"
}
```

**Registration change** (`internal/service/auth_service.go`): when `role == "seller"`,
set `SellerStatus = ptr("PENDING")`. Buyers/admins keep it nil.

**JWT claim** — do NOT add `seller_status` to the JWT. The JWT is long-lived; status
changes must take effect immediately. Instead, product-catalog-service calls
`ValidateToken` gRPC which returns user info — extend `ValidateTokenResponse` proto
to include `seller_status`. Product catalog then rejects `PENDING`/`SUSPENDED` sellers
from creating products.

Alternatively (simpler, lower-fidelity): add `seller_status` to the JWT and accept the
lag. Discussed in implementation — pick the gRPC path for correctness.

### Backend work — auth-service

**New repository contract** (`internal/domain/contracts/repositories.go`):
```go
UpdateSellerStatus(ctx context.Context, userID uuid.UUID, status string) error
ListSellers(ctx context.Context, status string, limit, offset int) ([]*domain.User, int64, error)
```

**New admin endpoints** (add to `admin_handler.go` from Feature 1):

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/admin/sellers` | List sellers — `?status=PENDING`, `?limit=`, `?offset=` |
| `PATCH` | `/v1/admin/sellers/{id}/status` | Body: `{ "status": "APPROVED" \| "SUSPENDED" }` |

**Kafka event on approval** (`user.seller_approved`): auth-service publishes to a
`users` Kafka topic via the outbox pattern (same as order/payment services). Payload:
```json
{ "user_id": "...", "email": "...", "full_name": "..." }
```

**Notification-service update** (`internal/consumer/handler.go`): add case
`user.seller_approved` to `buildNotification`:
```go
case "user.seller_approved":
    return notifier.Notification{
        UserID:    payload["user_id"],
        EventType: eventType,
        Subject:   "Your seller account has been approved",
        Body:      "Congratulations! Your ZapMarket seller account has been approved. You can now list products.",
    }, true
```

> Note: auth-service does not currently publish to Kafka. Two paths:
> (a) Add a Kafka producer + outbox table to auth-service (clean but more work).
> (b) Have the admin endpoint publish directly without outbox (simpler, loses durability).
> Recommended: option (a) — auth-service already has a DB, adding an outbox table is
> one migration and a small relay goroutine copied from order-management-service.

### Frontend — `/dashboard/sellers`

```
[Page header] Sellers        [🔍 search email]   All | PENDING | APPROVED | SUSPENDED

  3 pending approval ⚠

┌──────────────────────────────────────────────────────────────────────────────────────┐
│  Name            Email                Status     Products  Joined     Actions        │
│  Test Seller     seller@example.com   PENDING    —         Jun 18     [✓ Approve]    │
│                                                                        [✗ Suspend]   │
│  Jane Smith      jane@example.com     APPROVED   12        May 04     [✗ Suspend]    │
└──────────────────────────────────────────────────────────────────────────────────────┘
```

Approve → `PATCH /v1/admin/sellers/{id}/status` `{ status: "APPROVED" }`.
Suspend → same with `{ status: "SUSPENDED" }`.
Reinstate (shown for SUSPENDED) → same with `{ status: "APPROVED" }`.

Status badge colours: PENDING = yellow, APPROVED = green, SUSPENDED = red.

---

## 4. Bulk Import/Export

### Export (no backend needed)

Client-side: fetch all pages of data (looping until `offset + page_size >= total`),
build a CSV string in the browser, trigger `<a download>`. No new API endpoints.

Categories export columns: `id, name, slug, parent_id, created_at`
Products export columns: `id, name, slug, category_id, status, seller_id, created_at`
SKUs export columns: `id, product_id, sku_code, price_amount, price_currency, is_active`

### Import (frontend-only for simple cases)

Parse CSV client-side → validate each row → batch-call existing POST endpoints
sequentially (one request per row, with progress bar). This reuses existing endpoints
with no backend changes.

Category import: `POST /api/v1/categories` per row
Product import: `POST /api/v1/products` per row (admin creates on behalf of sellers
— needs `seller_id` column in CSV, forwarded as a field or defaulted to the admin's ID)

**Frontend UX:**

```
[Import CSV]  button on Categories and Products pages opens a modal:

┌─────────────────────────────────────────────────────────────────┐
│  Import Categories                                              │
│                                                                 │
│  [ Drop CSV here or click to browse ]                           │
│                                                                 │
│  Preview (first 5 rows):                                        │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ name         slug         parent_id                      │   │
│  │ Electronics  electronics  (empty)                        │   │
│  │ Phones       phones       (auto-resolve from name)       │   │
│  └──────────────────────────────────────────────────────────┘   │
│  12 rows detected  ·  0 errors                                  │
│                                                                 │
│  [Cancel]                              [Import 12 rows →]       │
└─────────────────────────────────────────────────────────────────┘
```

Progress bar appears during import showing `3 / 12 rows imported`.
Errors (e.g. duplicate slug) collected and shown in a summary after completion.

**CSV template download** button next to Import generates a blank CSV with the correct
headers — so users know the expected format.

### What stays out of scope for this feature

- Server-side bulk validation / transaction rollback on partial failure (would need a
  dedicated import endpoint)
- Product image bulk upload (requires ZIP + S3 — separate concern)
- Async import jobs (background processing for very large files — not needed at this scale)

---

## Implementation order (across both phases)

Phase 1 (catalog management) should land first. Within Phase 2:

1. **User management** backend (auth-service `ListUsers` + admin handler) — unblocks the
   users page and seller verification UI
2. **Seller verification** column + admin endpoints + notification event — builds on top
   of the user management admin handler
3. **Order oversight** backend (order-management-service admin routes)
4. **Backoffice-ui pages**: Users → Sellers → Orders (in that order, as backends land)
5. **Bulk import/export**: last — pure frontend, no backend dependency, lowest risk

---

## New gateway routes summary

| Path prefix | Upstream | Auth mode | Notes |
|---|---|---|---|
| `/v1/admin/orders` | `order-management-service` | `required` | Register before `/v1/admin` |
| `/v1/admin` | `auth-service` | `required` | Catches `/v1/admin/users`, `/v1/admin/sellers` |

Both use `required` (not `method_split`) — all admin operations need a valid JWT.
