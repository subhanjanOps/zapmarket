# API Gateway — Service Onboarding Guide

This document explains how to connect a new (or existing) ZapMarket microservice to the
API Gateway so that its HTTP endpoints are publicly reachable through port **8000**.

There are two paths:

| Path | When to use |
|---|---|
| **Auto-bind** | Development, local `docker compose up`, rapid iteration. The gateway discovers routes automatically from a manifest the service publishes to Redis. |
| **Manual bind** | Staging, production, or any environment where `GATEWAY_AUTO_BIND=false`. Routes are registered explicitly via the Admin API or a migration. |

Both paths use the same underlying data store (`gateway_routes` in the `apigateway` Postgres
database), so a route registered either way behaves identically at runtime.

---

## Concepts

### Service name

Every service has a **logical name** — the stable string the gateway uses to identify it.
By convention this matches the directory name under `services/`:

```
auth-service
product-catalog-service
order-management-service
inventory-service
payment-service
notification-service
```

Use the same name everywhere: in the manifest, in the heartbeat, and in the `upstream`
column of `gateway_routes`.

### Route

A route maps a **path prefix** to a service and declares an auth policy:

| Field | Type | Description |
|---|---|---|
| `path_prefix` | string | URL prefix the gateway matches (longest-prefix wins). e.g. `/v1/orders` |
| `upstream` | string | Logical service name |
| `auth_mode` | `none` \| `required` \| `method_split` | `none` = no JWT check; `required` = all methods need a valid JWT; `method_split` = GET/HEAD are public, everything else needs a JWT |
| `strip_prefix` | bool | If `true`, the prefix is stripped before forwarding to the upstream |

### Auth modes in detail

```
auth_mode = "none"
  GET /api/v1/products          → forwarded as-is, no auth check

auth_mode = "required"
  GET /v1/orders                → gateway validates JWT, injects X-User-* headers, forwards
  POST /v1/orders               → same

auth_mode = "method_split"
  GET /api/v1/products          → no auth check
  POST /api/v1/products         → JWT required
```

### Service registry (Redis)

The gateway resolves the `upstream` name to an actual HTTP address at request time using
Redis heartbeat keys:

```
svc:registry:{service-name}:{instance-id}  →  {"addr":"http://10.x.x.x:8081","instance_id":"...","started_at":"..."}  (TTL 30s)
```

Each running instance refreshes its key every 10 seconds. When an instance stops (crash,
scale-down), its key expires within 30 seconds and the gateway stops sending traffic to
it. The gateway falls back to the static env-var address
(`{SERVICE_NAME}_HTTP_URL`) if no Redis entries are found.

---

## Path 1 — Auto-bind (development default)

Auto-bind is active when `GATEWAY_AUTO_BIND=true` (the default in `development` env).
The gateway watches for **service manifests** in Redis every 15 seconds and
auto-inserts any missing routes into `gateway_routes`.

### Step 1 — Add Redis to your service

Your service already connects to Redis for other purposes (caching, idempotency, etc.).
If it does not, add the connection following the pattern in any other service's `main.go`:

```go
import goredis "github.com/redis/go-redis/v9"

rdb := goredis.NewClient(&goredis.Options{Addr: cfg.RedisURL})
if err := rdb.Ping(ctx).Err(); err != nil {
    log.Error("redis unreachable", "error", err)
    os.Exit(1)
}
```

### Step 2 — Publish a heartbeat

In your service's `main.go`, after the Redis client is ready and **before** starting the
HTTP server, launch the heartbeat goroutine:

```go
import "github.com/zapmarket/zapmarket/services/api-gateway/internal/registry"

// instanceID must be unique per running process.
// Use the hostname in Docker/k8s; a UUID or PID works locally.
instanceID := os.Getenv("HOSTNAME")
if instanceID == "" {
    instanceID = fmt.Sprintf("local-%d", os.Getpid())
}

// addr is the full HTTP base URL of this instance as reachable by the gateway.
addr := fmt.Sprintf("http://%s:%d", os.Getenv("SERVICE_HOST"), cfg.HTTPPort)

go registry.Heartbeat(appCtx, rdb, "my-new-service", instanceID, addr, log)
```

> `appCtx` is the context you cancel on SIGTERM. When it is cancelled, the heartbeat
> removes the Redis key immediately (best-effort deregister), so the gateway stops routing
> to this instance before the process exits.

### Step 3 — Publish a manifest

Still in `main.go`, publish the route manifest immediately after starting the heartbeat:

```go
err := registry.PublishManifest(appCtx, rdb, registry.Manifest{
    Service: "my-new-service",
    Version: "1.0.0",
    Routes: []registry.ManifestRoute{
        {
            PathPrefix:  "/v1/my-resource",
            AuthMode:    "required",  // or "none" / "method_split"
            StripPrefix: false,
        },
        {
            PathPrefix:  "/v1/my-resource/public",
            AuthMode:    "none",
            StripPrefix: false,
        },
    },
})
if err != nil {
    log.Warn("failed to publish gateway manifest", "error", err)
    // Non-fatal: service still works; auto-bind just won't fire.
}
```

The manifest key has a **60-second TTL** and is not refreshed automatically. You can
re-publish it on a slow ticker (e.g., every 50s) if you want it to survive long-running
processes. For Docker Compose restarts a single publish on startup is sufficient.

### Step 4 — Wait for auto-bind

The gateway's auto-bind watcher runs every 15 seconds. Within 15 seconds of publishing
the manifest, you will see in the gateway logs:

```
level=INFO msg="auto-bind: route registered" prefix=/v1/my-resource service=my-new-service
level=INFO msg="route change notified" op=INSERT
level=INFO msg="routes refreshed" count=N
```

The route is now live. No gateway restart needed.

### Conflict handling

If another service already owns the path prefix you declared, auto-bind will **not**
overwrite it. You will see:

```
level=ERROR msg="auto-bind: route conflict"
    prefix=/v1/my-resource
    existing=other-service
    challenger=my-new-service
```

Resolve the conflict by either choosing a different prefix or manually updating the
existing route via the Admin API (see below).

---

## Path 2 — Manual bind (staging / production)

When `GATEWAY_AUTO_BIND=false`, routes must be registered explicitly before traffic
can be forwarded. You have two options.

### Option A — Admin API

The gateway exposes a route management API at `/gateway/v1/routes`. All endpoints require
a JWT with `role=admin`.

**Register a route:**

```bash
curl -X POST http://gateway:8000/gateway/v1/routes \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "path_prefix":  "/v1/my-resource",
    "upstream":     "my-new-service",
    "auth_mode":    "required",
    "strip_prefix": false
  }'
```

Response:
```json
{ "success": true, "data": { "id": "uuid-of-new-route" } }
```

**List all routes:**

```bash
curl http://gateway:8000/gateway/v1/routes \
  -H "Authorization: Bearer <admin-token>"
```

**Update a route** (e.g., change auth_mode):

```bash
curl -X PUT http://gateway:8000/gateway/v1/routes/<id> \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{ "auth_mode": "method_split" }'
```

**Disable a route** (soft delete — sets `enabled=false`):

```bash
curl -X DELETE http://gateway:8000/gateway/v1/routes/<id> \
  -H "Authorization: Bearer <admin-token>"
```

Route changes take effect **instantly** — the gateway receives a Postgres LISTEN/NOTIFY
event and rebuilds its router within milliseconds. No restart required.

### Option B — SQL migration (CI/CD pipeline)

For reproducible deployments, add the route registration to the gateway's migration files:

```sql
-- services/api-gateway/migrations/0002_add_my_service.up.sql
INSERT INTO gateway_routes (path_prefix, upstream, auth_mode, strip_prefix)
VALUES ('/v1/my-resource', 'my-new-service', 'required', false)
ON CONFLICT (path_prefix) DO NOTHING;
```

```sql
-- services/api-gateway/migrations/0002_add_my_service.down.sql
DELETE FROM gateway_routes WHERE path_prefix = '/v1/my-resource';
```

The gateway runs `migrate.Up` on startup (`MIGRATE_ON_BOOT=true` by default), so the
route is registered before the first request is served.

---

## Step-by-step checklist: onboarding a new service

```
□ 1. Choose a logical service name (matches services/ directory)
□ 2. Decide path prefix(es) and auth mode for each one
□ 3. Add redis heartbeat call to your service's main.go
□ 4. Add registry.PublishManifest call to your service's main.go
□ 5. Add your service's static fallback URL to the gateway's env:
       MY_NEW_SERVICE_HTTP_URL=http://zapmarket-my-new-service:PORT
     and add it to the StaticRegistry map in services/api-gateway/main.go
□ 6. Add the docker-compose service block with REDIS_URL env var
□ 7. (Dev) Start services → auto-bind registers routes within 15 s
   (Prod) Register routes via Admin API or SQL migration before deploy
□ 8. Verify: curl http://gateway:8000/v1/my-resource → correct upstream response
□ 9. Check gateway logs for "routes refreshed count=N" with the new total
```

---

## Headers forwarded to upstream services

The gateway injects these headers on every proxied request:

| Header | Value | Set when |
|---|---|---|
| `X-Request-ID` | UUID generated at gateway edge | Always |
| `X-User-ID` | Authenticated user's UUID | `auth_mode` is `required` or `method_split` (write methods) |
| `X-User-Email` | Authenticated user's email | Same as above |
| `X-User-Role` | Authenticated user's role (`buyer`, `seller`, `admin`) | Same as above |

Your service can trust these headers without re-validating the JWT. Do not expose your
service's port directly to clients in production — traffic must come through the gateway.

---

## Circuit breaker behaviour

The gateway wraps each upstream with a circuit breaker (Sony gobreaker):

- Opens after **5 consecutive 5xx responses** from a single upstream name.
- Stays open for **10 seconds**, then enters half-open (lets 5 probe requests through).
- When open: requests to that upstream return `503 CIRCUIT_OPEN` immediately, without
  hitting your service.

Monitor circuit breaker state changes in the gateway logs:

```
level=WARN msg="circuit breaker state change"
    upstream=my-new-service from=Closed to=Open
```

---

## Redis key reference

| Key | TTL | Purpose |
|---|---|---|
| `svc:registry:{name}:{instance-id}` | 30s | Live instance heartbeat |
| `svc:manifest:{name}` | 60s | Route manifest for auto-bind |
| `ratelimit:ip:{ip}` | 60s | Per-IP rate limit counter |
| `ratelimit:user:{user-id}` | 60s | Per-user rate limit counter |

---

## Audit log

Every auth rejection, rate-limit hit, circuit-open event, and route conflict is written
to `gateway_audit_log` in the `apigateway` database. Query via the Admin API:

```bash
# Last 50 auth rejections for a specific user
curl "http://gateway:8000/gateway/v1/audit?event=AUTH_REJECTED&user_id=<uuid>&limit=50" \
  -H "Authorization: Bearer <admin-token>"

# All rate-limit hits in the last hour
curl "http://gateway:8000/gateway/v1/audit?event=RATE_LIMITED&from=2026-06-17T12:00:00Z" \
  -H "Authorization: Bearer <admin-token>"
```

Available event types: `AUTH_REJECTED`, `RATE_LIMITED`, `UPSTREAM_5XX`, `CIRCUIT_OPEN`,
`ROUTE_CONFLICT`.
