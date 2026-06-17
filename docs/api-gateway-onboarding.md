# API Gateway — Service Onboarding Guide

This document explains how to connect a new (or existing) ZapMarket microservice to the
API Gateway so that its HTTP endpoints are publicly reachable through port **8000**.

There are two paths:

| Path | When to use |
|---|---|
| **Auto-bind** | Development, local `docker compose up`, rapid iteration. The gateway fetches the service's Swagger spec and registers routes automatically. |
| **Manual bind** | Staging, production, or any environment where `GATEWAY_AUTO_BIND=false`. Routes are registered explicitly via the Admin API or a DB migration. |

Both paths write to the same `gateway_routes` table in the `apigateway` Postgres database.
A route registered either way behaves identically at runtime.

---

## How auto-bind works

When `GATEWAY_AUTO_BIND=true` (the default in `development` env), the gateway runs a
background watcher that:

1. Scans Redis every 15 seconds for live service heartbeat keys
   (`svc:registry:{service-name}:{instance-id}`).
2. For each newly discovered service, fetches its Swagger spec from
   `{service-addr}/v1/docs/swagger.json`.
3. Reads `basePath` from the spec — this becomes the route's `path_prefix`.
4. Analyses the security fields across all operations to derive an `auth_mode`
   (see [Auth mode derivation](#auth-mode-derivation)).
5. Inserts the route into `gateway_routes` via `ON CONFLICT DO NOTHING`.
6. The Postgres `LISTEN/NOTIFY` trigger fires, and the gateway's router rebuilds
   in milliseconds — no restart required.

The spec is only fetched **once per (service, address) pair**. If the same service
restarts at the same address the spec is not re-fetched; to force a re-fetch, change
the instance ID or the address.

---

## Concepts

### Service name

Every service has a **logical name** — the stable string used in heartbeat keys and in the
`upstream` column of `gateway_routes`. By convention it matches the directory name under
`services/`:

```
auth-service
product-catalog-service
order-management-service
inventory-service
payment-service
notification-service
```

### Auth mode derivation

The gateway reads all operations from the Swagger spec and applies this logic:

| Condition | `auth_mode` |
|---|---|
| No operations have a `security` field | `none` |
| All operations have a `security` field | `required` |
| Read methods (GET/HEAD) are all unsecured; write methods are secured | `method_split` |
| Mixed (some reads secured, some writes not) | `required` *(safe fallback)* |

`method_split` means GET/HEAD requests are forwarded without a JWT check; all other
methods require a valid token. This is ideal for catalog-style endpoints where public
browsing is allowed but mutations require login.

After auto-bind, you can change `auth_mode` for any route via the Admin API without
re-deploying anything.

### Route fields

| Field | Type | Description |
|---|---|---|
| `path_prefix` | string | Longest-prefix match. e.g. `/v1/auth` matches `/v1/auth/me` and `/v1/auth/login` |
| `upstream` | string | Logical service name resolved via Redis registry (falls back to env-var static address) |
| `auth_mode` | `none` \| `required` \| `method_split` | JWT enforcement policy |
| `strip_prefix` | bool | Strip `path_prefix` from the URL before forwarding to upstream |

---

## Path 1 — Auto-bind (development default)

### Step 1 — Expose a Swagger endpoint

Your service must serve its Swagger JSON at the standard path:

```
GET /v1/docs/swagger.json
```

All ZapMarket services already do this via `pkg/swaggerx`. If you are building a new
service, add the swagger init and route following the pattern in `product-catalog-service`
or `auth-service`.

### Step 2 — Publish a heartbeat

In your service's `main.go`, after Redis is connected and **before** starting the HTTP
server, launch the heartbeat goroutine:

```go
import "github.com/zapmarket/zapmarket/services/api-gateway/internal/registry"

// instanceID must be unique per running process.
// Use the pod hostname in k8s; a UUID or PID works locally.
instanceID := os.Getenv("HOSTNAME")
if instanceID == "" {
    instanceID = fmt.Sprintf("local-%d", os.Getpid())
}

// addr is the full HTTP base URL of this instance as reachable by the gateway.
addr := fmt.Sprintf("http://%s:%d", os.Getenv("SERVICE_HOST"), cfg.HTTPPort)

go registry.Heartbeat(appCtx, rdb, "my-new-service", instanceID, addr, log)
```

> `appCtx` is the context you cancel on SIGTERM. When cancelled, the heartbeat removes
> the Redis key immediately so the gateway stops routing to this instance before the
> process exits.

That is all the service needs to do. The gateway handles the rest.

### Step 3 — Start services and observe

Within **15 seconds** of the heartbeat key appearing in Redis you will see in the
gateway logs:

```
level=INFO msg="auto-bind: route registered"
    prefix=/v1/my-resource service=my-new-service auth_mode=required
level=INFO msg="route change notified" op=INSERT
level=INFO msg="routes refreshed" count=N
```

The route is now live. Traffic to `http://gateway:8000/v1/my-resource/*` is forwarded
to your service.

### Conflict handling

If another service already owns the path prefix the gateway will **not** overwrite it:

```
level=ERROR msg="auto-bind: route conflict"
    prefix=/v1/my-resource existing=other-service challenger=my-new-service
```

Resolve by either choosing a different prefix or updating the existing route via the
Admin API (see below).

---

## Path 2 — Manual bind (staging / production)

When `GATEWAY_AUTO_BIND=false`, routes must be registered before traffic is forwarded.

### Option A — Admin API

All endpoints under `/gateway/v1` require a JWT with `role=admin`.

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
# → { "success": true, "data": { "id": "<uuid>" } }
```

**List routes:**

```bash
curl http://gateway:8000/gateway/v1/routes \
  -H "Authorization: Bearer <admin-token>"
```

**Update a route:**

```bash
curl -X PUT http://gateway:8000/gateway/v1/routes/<id> \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{ "auth_mode": "method_split" }'
```

**Disable a route** (soft-delete, sets `enabled=false`):

```bash
curl -X DELETE http://gateway:8000/gateway/v1/routes/<id> \
  -H "Authorization: Bearer <admin-token>"
```

Route changes take effect instantly — no gateway restart needed.

### Option B — SQL migration (CI/CD pipeline)

Add the route to the gateway's migration files for reproducible deployments:

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
□ 1. Confirm your service exposes GET /v1/docs/swagger.json
□ 2. Add registry.Heartbeat(...) call to your service's main.go (see Step 2 above)
□ 3. Add your service's static fallback URL to the gateway's env and StaticRegistry:
       MY_NEW_SERVICE_HTTP_URL=http://zapmarket-my-new-service:PORT
□ 4. Add the service to docker-compose with REDIS_URL env var set
□ 5. (Dev)  Start services → auto-bind fires within 15 s, routes appear in gateway logs
   (Prod) Register routes via Admin API or SQL migration before deploy
□ 6. Verify: curl http://gateway:8000/v1/my-resource → response from your service
□ 7. If auth_mode was wrong: update via Admin API, no restart needed
```

---

## Headers forwarded to upstream services

The gateway injects these headers on every proxied request:

| Header | Value | Set when |
|---|---|---|
| `X-Request-ID` | UUID generated at the gateway edge | Always |
| `X-User-ID` | Authenticated user's UUID | `auth_mode` is `required` or `method_split` write methods |
| `X-User-Email` | Authenticated user's email | Same |
| `X-User-Role` | Authenticated user's role (`buyer`, `seller`, `admin`) | Same |

Your service can trust these headers without re-validating the JWT. **Do not expose your
service's port directly to clients in production** — all traffic must come through the
gateway.

---

## Circuit breaker behaviour

Each upstream has a circuit breaker (Sony gobreaker v2):

- Opens after **5 consecutive 5xx responses**.
- Stays open for **10 seconds**, then enters half-open (allows 5 probe requests).
- While open: requests return `503 CIRCUIT_OPEN` immediately without hitting your service.

State changes appear in gateway logs:

```
level=WARN msg="circuit breaker state change"
    upstream=my-new-service from=Closed to=Open
```

---

## Redis key reference

| Key pattern | TTL | Written by |
|---|---|---|
| `svc:registry:{name}:{instance-id}` | 30 s | Each service's `registry.Heartbeat()` |
| `ratelimit:ip:{ip}` | 60 s | Gateway rate limiter |
| `ratelimit:user:{user-id}` | 60 s | Gateway rate limiter |

---

## Audit log

Every auth rejection, rate-limit hit, circuit-open event, and route conflict is written
to `gateway_audit_log`. Query via the Admin API:

```bash
# Last 50 auth rejections
curl "http://gateway:8000/gateway/v1/audit?event=AUTH_REJECTED&limit=50" \
  -H "Authorization: Bearer <admin-token>"

# Rate-limit hits for a specific user
curl "http://gateway:8000/gateway/v1/audit?event=RATE_LIMITED&user_id=<uuid>" \
  -H "Authorization: Bearer <admin-token>"
```

Event types: `AUTH_REJECTED`, `RATE_LIMITED`, `UPSTREAM_5XX`, `CIRCUIT_OPEN`,
`ROUTE_CONFLICT`.
