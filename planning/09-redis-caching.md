# Stage 9 — Redis Caching Layer

Corresponds to checklist **Phase 14**, plus closing out every Redis TODO deferred in
Stages 3, 4, 5, 6, and 8.

## Goal

Turn on Redis platform-wide and replace every Postgres-only/in-memory placeholder built
in earlier stages with the intended Redis-backed implementation from `design.md`.

## Preconditions
- Stages 3-8 merged. This stage is explicitly a "go back and upgrade" pass — it has the
  most cross-references to earlier stages of any stage in this plan.

## Tasks

### 9.1 Turn on infra
- [ ] Uncomment the `redis` block in `docker-compose.yml`.
- [ ] Add `pkg/redis` — client wrapper, connection pool config, health check, matching
      the `pkg/database` pattern already established.

### 9.2 Inventory: Lua-script atomic reservation (replaces Stage 3's Postgres-only path)
- [ ] Implement the check-and-decrement Lua script (`DECRBY inv:stock:{sku} qty` with a
      guard against going negative) per `design.md`'s Redis Usage Breakdown table.
- [ ] Postgres `inventory_ledger` becomes the async durable record, written after the
      Redis operation succeeds, not the primary check-and-decrement path anymore.
- [ ] Add a reconciliation job/check that Redis counters and Postgres ledger agree (skew
      here is a correctness bug, not a nice-to-have — surface it loudly if found).

### 9.3 Payment & Order: Redis idempotency keys (replaces Postgres-unique-constraint
      fallback from Stages 4-5)
- [ ] `SET {svc}:idem:{key} {result} EX 86400 NX` per `design.md`.
- [ ] Keep the Postgres unique constraint as a defense-in-depth backstop — don't remove it,
      just stop relying on it as the primary mechanism (Redis is faster; Postgres is the
      correctness guarantee if Redis data is ever lost/flushed).

### 9.4 Catalog: product/category page cache
- [ ] `SET product:page:{id} {json} EX 300` cache-aside pattern wrapping the list
      endpoints built in Stage 2. Invalidate on product/category mutation.

### 9.5 Auth: session store & token blacklist
- [ ] `auth:session:{token}` and `auth:blacklist:{jti}` per `design.md`.
- [ ] This enables actual refresh-token revocation, which the Phase 18 checklist calls
      out as a production-readiness gap — implementing it here unblocks that later item.

### 9.6 Gateway: Redis-backed rate limiting (replaces in-memory limiter from Stage 8)
- [ ] Swap the token-bucket limiter to Redis counters so rate limits are consistent
      across multiple gateway replicas (relevant once Stage 11's k8s HPA can scale the
      gateway horizontally).

### 9.7 Order: cart data
- [ ] `HSET order:cart:{user_id} {items}` if cart-before-checkout is in scope — confirm
      with the user whether carts are a feature being built or whether checkout is
      always "buy these items now" with no persistent cart; the current Stage 5 plan
      didn't model a cart, so this may be new scope, not a gap-fill.

### 9.8 Notification: dedup & rate limiting
- [ ] Replace Stage 6's in-memory LRU dedup with `SET notif:dedup:{event_id} 1 EX 3600 NX`.

## Out of scope
- Redis Cluster / HA topology — single-node Redis is fine for this stage; clustering is
  a Stage 12/production concern if needed at all.

## Definition of done
- Every Redis usage row in `design.md`'s "Redis Usage Breakdown" table is implemented.
- Inventory oversell test from Stage 3 (20 concurrent reservations against stock of 10)
  still passes, now backed by the Lua script instead of the Postgres-only guard.
- Killing the Redis container and restarting it doesn't corrupt Postgres ledger data
  (Redis is cache/fast-path, Postgres remains durable source of truth where applicable).
