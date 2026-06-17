# Stage 8b — Gateway Robustness: DB Integration, Service Discovery & Auto-Binding

## Problem with the current gateway

The current gateway (`services/api-gateway/main.go`) has three hard limitations:

1. **Routes are hardcoded in Go.** Adding a new service or a new route requires a code change, a rebuild, and a redeploy of the gateway.
2. **Upstream addresses come from env vars.** There is no health-awareness — the gateway doesn't know if a downstream is up, degraded, or which replica to use. Env vars can't change without a restart.
3. **No audit trail.** Rate limit hits, auth rejections, and upstream errors are logged to stdout but nothing is persisted — you can't query "how many 429s did user X get this week?"

The three pillars of the proposed improvement address each of these.

---

## Pillar 1 — DB Integration (Route Config + Audit Log)

### What it is

A Postgres table (`gateway_routes`) acts as a live route registry. The gateway polls it (or listens on a Postgres NOTIFY channel) and rebuilds its chi router in-place when routes change — no restart needed.

A second table (`gateway_audit_log`) records every significant gateway event (auth rejection, rate-limit hit, upstream 5xx, circuit open) with enough context for analytics and compliance.

### Schema

```sql
-- Route definitions managed via admin API or direct SQL.
CREATE TABLE gateway_routes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    path_prefix TEXT NOT NULL UNIQUE,      -- e.g. '/v1/orders'
    upstream    TEXT NOT NULL,             -- service name, e.g. 'order-management-service'
    auth_mode   TEXT NOT NULL DEFAULT 'required',
                                           -- 'none' | 'required' | 'method_split'
    strip_prefix BOOLEAN NOT NULL DEFAULT false,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ DEFAULT now(),
    updated_at  TIMESTAMPTZ DEFAULT now()
);

-- Append-only audit log for gateway-level events.
CREATE TABLE gateway_audit_log (
    id          BIGSERIAL PRIMARY KEY,
    ts          TIMESTAMPTZ DEFAULT now(),
    request_id  TEXT,
    user_id     TEXT,
    ip          TEXT,
    method      TEXT,
    path        TEXT,
    upstream    TEXT,
    status_code INT,
    event       TEXT,  -- 'AUTH_REJECTED' | 'RATE_LIMITED' | 'UPSTREAM_5XX' | 'CIRCUIT_OPEN'
    detail      TEXT
);
CREATE INDEX ON gateway_audit_log (ts DESC);
CREATE INDEX ON gateway_audit_log (user_id, ts DESC);
```

### How the gateway uses it

1. On startup: load all `enabled = true` routes from DB, build the chi router.
2. Background goroutine: `LISTEN gateway_route_changed` on a Postgres NOTIFY channel. Any INSERT/UPDATE/DELETE on `gateway_routes` fires the trigger, which sends a NOTIFY. The gateway receives it and atomically swaps the router pointer (protected by `sync/atomic`).
3. Fallback poll every 30s: if the NOTIFY connection drops, the poll catches any missed changes.
4. Audit writes: async, via a buffered channel → batch INSERT every 1s or 100 events. Gateway never blocks on audit writes. Channel backpressure drops events (logged as a metric) rather than slowing requests.

### Route config format

Each row in `gateway_routes` maps one path prefix to one named upstream. The `upstream` column is a logical name (`order-management-service`) — the actual address is resolved by Pillar 2 (service discovery). This decouples routing policy from topology.

### Admin API

A small set of endpoints on the gateway itself (protected by `role=admin`) for managing routes without direct DB access:

```
GET    /gateway/v1/routes          # list all routes
POST   /gateway/v1/routes          # add a route
PUT    /gateway/v1/routes/:id      # update a route
DELETE /gateway/v1/routes/:id      # soft-delete (sets enabled=false)
GET    /gateway/v1/audit?user=&from=&to=  # query audit log
```

---

## Pillar 2 — Dynamic Service Discovery

### The problem with env vars

Currently: `AUTH_SERVICE_HTTP_URL=http://localhost:8080`. This is static — it doesn't know about replicas, health, or restarts. If auth-service moves to a different container IP, the gateway needs a restart.

### Proposed approach: Redis-backed service registry (no Consul/etcd dependency)

Each service registers itself in Redis on startup and refreshes every 10s:

```
SET svc:registry:{service-name}:{instance-id}  {"addr":"http://10.x.x.x:8080","healthy":true,"started_at":"..."}  EX 30
```

Key expires in 30s — if the service stops refreshing (crash, OOM), the key disappears naturally. The gateway reads `SCAN svc:registry:{service-name}:*` to get all live instances of a service.

### Load balancing strategy (per upstream)

The gateway maintains a per-service round-robin cursor. On each request:

1. SCAN Redis for all healthy instances of the target service.
2. Pick the next instance (round-robin mod len).
3. Create or reuse the `httputil.ReverseProxy` for that instance address (proxies are pooled by address).
4. If the instance's circuit breaker is open, skip to the next instance. If all are open, return `503`.

### Why Redis instead of Consul/etcd

- Already running in the project, no new infra dependency.
- The TTL-based health model is simple and correct for this scale.
- When Stage 11 (k8s) lands, k8s Services + DNS (`zapmarket-auth-service:8080`) become the resolver and this Redis registry becomes optional/redundant — the gateway's discovery interface is behind an interface (`ServiceRegistry`) so the backend can be swapped without changing routing logic.

### `ServiceRegistry` interface

```go
type Instance struct {
    Addr    string
    Healthy bool
}

type ServiceRegistry interface {
    Instances(ctx context.Context, name string) ([]Instance, error)
    Register(ctx context.Context, name, instanceID, addr string) error
    Deregister(ctx context.Context, name, instanceID string) error
}
```

Two implementations:
- `RedisRegistry` — uses the Redis key pattern above (for Docker Compose / standalone).
- `StaticRegistry` — reads from env vars (current behavior, kept as fallback for local dev without Redis service registration).

Each service's `main.go` gets a ~10-line addition to register itself on startup and refresh every 10s.

---

## Pillar 3 — Auto Service Binding

### What it is

"Auto-binding" means: when a service starts, it not only registers its address but also declares its routes. The gateway picks these up and adds them to the route table automatically — without a human editing `gateway_routes`.

### Mechanism: service manifest in Redis

On startup each service writes a manifest key:

```
SET svc:manifest:{service-name}  {routes:[...], version:"1.2.3"}  EX 60
```

The manifest is a JSON blob declaring what routes the service owns:

```json
{
  "service": "order-management-service",
  "version": "1.0.0",
  "routes": [
    { "path_prefix": "/v1/orders", "auth_mode": "required", "strip_prefix": false }
  ]
}
```

The gateway's background watcher:
1. Watches for new/changed `svc:manifest:*` keys (SCAN every 15s or SUBSCRIBE to a Redis pub/sub channel services publish to on startup).
2. Diffs incoming manifest against current `gateway_routes` table.
3. Auto-inserts missing routes, marks removed routes as `enabled=false`.
4. Logs every binding change at `INFO` level; treats conflicts (two services claiming the same prefix) as `ERROR` and does not auto-apply.

### Conflict handling

If two services claim the same path prefix, the gateway:
- Keeps the existing route (first-writer wins).
- Logs `ERROR: route conflict — {prefix} claimed by {service-A} and {service-B}`.
- Writes a row to `gateway_audit_log` with `event='ROUTE_CONFLICT'`.
- Does **not** route to either new service until a human resolves the conflict via the admin API.

### Safety: opt-in flag

Auto-binding is controlled by a config flag: `GATEWAY_AUTO_BIND=true`. Default is `false` in production env, `true` in development env. This means:
- Dev: services self-register routes automatically.
- Production: routes are managed manually (or via CI pipeline that calls the admin API), and auto-binding is a read-only informational log.

---

## Implementation sequence

These three pillars are independent and can be shipped in order, each behind a feature flag:

| Step | What | Flag | Effort |
|---|---|---|---|
| **8b.1** | DB schema + migration for `gateway_routes` + `gateway_audit_log` | — | Small |
| **8b.2** | Route config loader: read routes from DB, build router, NOTIFY watcher + poll fallback | `GATEWAY_DYNAMIC_ROUTES=true` | Medium |
| **8b.3** | Audit log writer: buffered async channel → batch insert | always on if DB available | Small |
| **8b.4** | Admin API for route CRUD + audit query | — | Small |
| **8b.5** | `ServiceRegistry` interface + `RedisRegistry` impl + `StaticRegistry` fallback | — | Medium |
| **8b.6** | Per-service registration heartbeat (10-line addition to each service's `main.go`) | — | Small (x6 services) |
| **8b.7** | Gateway uses `ServiceRegistry` for upstream resolution + round-robin load balancing | `GATEWAY_SERVICE_DISCOVERY=true` | Medium |
| **8b.8** | Service manifest auto-binding watcher | `GATEWAY_AUTO_BIND=true` | Medium |
| **8b.9** | Integration test: start two auth-service replicas, verify round-robin; kill one, verify failover | — | Small |

Total estimated effort: **~2 days** if done sequentially. Steps 8b.1–8b.4 (DB + dynamic routes) are the most immediately useful and can ship independently.

---

## What stays the same

- chi router (still used, but rebuilt dynamically).
- `sony/gobreaker` circuit breakers (still one per upstream, now keyed by instance address rather than service name).
- Redis rate limiting (unchanged).
- JWT validation middleware (unchanged).
- Dockerfile, docker-compose entry (unchanged).

## What changes

- `main.go` shrinks: route registration moves out of Go code into `gateway_routes` table.
- `internal/proxy/proxy.go`: upstream address is resolved at request time via `ServiceRegistry`, not at startup.
- Each service's `main.go`: adds ~10 lines for heartbeat registration.
- New `internal/registry/` package for service discovery.
- New `internal/routes/` package for DB-backed route loading.
- New `internal/audit/` package for async audit writing.
- New migration under `services/api-gateway/migrations/`.

## Open questions to decide before building

1. **Auto-bind in prod?** Default-off is the safe choice, but it means manual route management. Do you want a CI step that calls the admin API on every deploy instead?
2. **Redis vs Consul for discovery?** Redis is already there and keeps infra surface small. Consul makes sense if Stage 11 k8s is coming soon (k8s has its own DNS that makes both redundant anyway).
3. **Should the gateway have its own Postgres database?** Currently all services share the same Postgres instance (different databases). A dedicated `gatewaydb` is cleanest but adds a migration step. Alternatively, `gateway_routes` and `gateway_audit_log` could live in one of the existing DBs (e.g., `userauth`). Recommended: new dedicated DB `apigateway`.
4. **Admin API auth: role=admin JWT, or a separate gateway API key?** JWT role check is consistent with the rest of the system. API key is simpler for CI pipelines calling the admin API.
