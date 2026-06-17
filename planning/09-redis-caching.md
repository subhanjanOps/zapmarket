# Stage 9 — Redis Caching Layer ✅ Complete

Corresponds to checklist **Phase 14**, plus closing out every Redis TODO deferred in
Stages 3, 4, 5, 6, and 8.

## Goal

Turn on Redis platform-wide and replace every Postgres-only/in-memory placeholder with
the intended Redis-backed implementation from `design.md`.

## Implementation notes

**`pkg/redis`**: new shared module (`pkg/redis/client.go`) wrapping `*goredis.Client`
with pool config and a Ping health check on construction. Added to `go.work`.

**Redis unavailability**: auth and product-catalog services treat Redis as optional —
they log a warning and continue without the cache if Redis is unreachable. Order,
inventory, and payment services treat Redis as required (exit on startup failure)
because their Redis paths are primary correctness paths, not just caches.

## Completed tasks

### 9.1 Infrastructure
- [x] `redis:7-alpine` running in `docker-compose.yml` with healthcheck and `redis-data` volume
- [x] `pkg/redis` module created: `New(addr)` returns a connected `*Client` or error

### 9.2 Inventory: Lua-script atomic reservation
- [x] Lua check-and-decrement script (`luaReserve`) in `inventory_service.go`:
      atomically checks `inv:stock:{sku_id}` and `DECRBY` if sufficient — no race
- [x] Cache miss path: on `-1` return, loads `qty_available` from DB, warms Redis, retries
- [x] Postgres still written after Redis gate passes (belt-and-suspenders check)
- [x] Redis rollback on Postgres failure: `INCRBY` restores the decremented value
- [x] `ReleaseStock` calls `GetReservationDetails` then `INCRBY` to restore available qty
- [x] `AddStock` increments Redis counter after the DB write succeeds
- [x] Fallback to DB-only path (`dbReserve`) if Redis returns an unexpected error
- [x] `CRITICAL` log if Redis rollback itself fails after a Postgres failure
- [x] Verified: `inv:stock:{sku}` key warmed on first reservation; value = remaining qty

### 9.3 Payment & Order: Redis idempotency keys
- [x] **Order**: `SET idempotency:order:{key} {order_json} EX 86400` — Redis first, DB fallback,
      DB fallback also backfills Redis; failed orders not cached (caller may retry)
- [x] **Payment**: `SET payment:idem:{key} {payment_json} EX 86400 NX` — same pattern;
      captured payments cached, failed payments not cached
- [x] Postgres unique constraint kept as defense-in-depth backstop in both services

### 9.4 Catalog: product/category page cache
- [x] `cachedProductService` decorator wrapping `ProductService` interface
      (`internal/service/product_cache.go`) — no change to handler or repository
- [x] `GET product:id:{uuid}` / `GET product:slug:{slug}` — 5 min TTL, populated on miss
- [x] `GET product:list:{md5(filters)}` — 5 min TTL, list pages cached by serialized filters
- [x] Both slug and ID keys populated on any single lookup (cross-warm)
- [x] `UpdateProduct` / `DeleteProduct` invalidate ID + slug keys
- [x] Redis optional: if unavailable at startup, service logs a warning and runs without cache

### 9.5 Auth: session store & token blacklist
- [x] `auth:blacklist:{sha256(token)}` set on logout with TTL = remaining token life
- [x] `ValidateAccessToken` checks blacklist before verifying JWT signature
- [x] `Logout` HTTP endpoint (`POST /v1/auth/logout`) — extracts Bearer token, blacklists it,
      invalidates refresh tokens in DB
- [x] Redis optional in auth-service: if unavailable, blacklist is disabled (token still
      expires naturally; refresh token revocation still works via DB)
- [x] Verified: `/me` returns 401 immediately after logout with same token

### 9.6 Gateway: Redis-backed rate limiting
- [ ] Deferred — Stage 8 (API Gateway) not yet built.

### 9.7 Order: cart data
- [ ] Not in scope — order service uses "buy these items now" checkout; no persistent cart.

### 9.8 Notification: dedup & rate limiting
- [x] `SET notif:dedup:{outbox_id} 1 EX 3600 NX` — Redis-backed dedup replacing in-memory

## Definition of done — met

- All implemented Redis usage rows from `design.md` are live ✅
- Inventory Lua script prevents oversell atomically; Redis warmed on first request ✅
- Token blacklist works: revoked token returns 401 immediately ✅
- Product list pages cached; Redis key created on first request ✅
- Killing Redis container: order/inventory/payment services exit and need restart
  (they treat Redis as required); auth and catalog degrade gracefully ✅
- Postgres ledger data unaffected by Redis state ✅

## Deferred to Stage 12 / production
- Redis Cluster / HA topology
- Rate limiting (needs Stage 8 API Gateway first — 9.6)
