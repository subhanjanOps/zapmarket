# Stage 8 — API Gateway ✅ Complete

Corresponds to checklist **Phase 13**.

## Goal

Stand up a single public ingress so no individual service is directly internet-reachable,
per `design.md`'s architecture diagram.

## Implementation notes

**Technology choice**: thin Go service using `go-chi/chi/v5` + `net/http/httputil.ReverseProxy`
rather than Kong/Envoy/Nginx — consistent with the existing project stack, no new runtime
dependency, full control over middleware chain.

**Circuit breaker**: `sony/gobreaker/v2` — one breaker per upstream, trips after 5 consecutive
failures, 10s timeout before half-open probe.

**Rate limiting**: Redis-backed INCR counter with 60s TTL window. Per-IP (200 req/min) for
anonymous traffic; per-user (500 req/min) for authenticated traffic. `X-RateLimit-Limit`
and `X-RateLimit-Remaining` headers returned on every response.

**Request ID**: `X-Request-ID` generated (uuid v4) or preserved from client header; forwarded
downstream; set in response header.

**Port**: 8000 — single public-facing port.

## Route table

| Path prefix | Upstream | Auth required |
|---|---|---|
| `POST /v1/auth/register` | auth-service:8080 | No |
| `POST /v1/auth/login` | auth-service:8080 | No |
| `POST /v1/auth/refresh` | auth-service:8080 | No |
| `/v1/auth/oauth/*` | auth-service:8080 | No |
| `GET /v1/auth/me` | auth-service:8080 | Yes |
| `POST /v1/auth/logout` | auth-service:8080 | Yes |
| `GET /api/v1/categories/*` | product-catalog-service:8081 | No |
| `GET /api/v1/products/*` | product-catalog-service:8081 | No |
| `POST/PUT/DELETE /api/v1/products/*` | product-catalog-service:8081 | Yes |
| `/v1/orders/*` | order-management-service:8084 | Yes |

Inventory and Payment are gRPC-internal only — not routed publicly.

## Completed tasks

### 8.1 Technology choice
- [x] Thin Go service confirmed: chi + httputil.ReverseProxy

### 8.2 Routing
- [x] Path-prefix routing per design.md table implemented in `main.go`
- [x] Inventory and Payment excluded from public routing (gRPC-internal only)

### 8.3 JWT validation
- [x] `NewAuthMiddleware(addr)` dials auth-service gRPC at startup
- [x] `Authenticate` middleware calls `ValidateToken` RPC on every protected request
- [x] On success: sets `X-User-ID`, `X-User-Email`, `X-User-Role` headers before forwarding
- [x] Public routes (register, login, refresh, catalog GET) skip auth middleware

### 8.4 Rate limiting
- [x] Redis-backed INCR/EXPIRE rate limiter (no in-memory interim step needed — Stage 9 done)
- [x] 200 req/min per IP for anonymous, 500 req/min per authenticated user
- [x] Returns `429 RATE_LIMITED` when exceeded; rate-limit headers always present

### 8.5 Cross-cutting middleware
- [x] `RequestID` middleware: generate/preserve `X-Request-ID`, forward downstream
- [x] Circuit breaker per upstream via `sony/gobreaker/v2`
- [x] `responseRecorder` captures status so circuit breaker counts 5xx as failures
- [x] `chimw.Recoverer` — panic recovery, returns 500 instead of crashing
- [x] `chimw.RealIP` — respects `X-Forwarded-For` / `X-Real-IP`

### 8.6 Infrastructure
- [x] `services/api-gateway/Dockerfile` — multi-stage build, GOWORK=off
- [x] `docker-compose.yml` — `api-gateway` service on port 8000, depends on redis + auth-service
- [x] Added to `go.work`
- [x] `.env.example` with all config vars documented

## Verified

- `GET /health` → `{"status":"ok","service":"api-gateway"}`
- `GET /v1/auth/me` without token → `401 MISSING_TOKEN`
- `GET /v1/auth/me` with valid token → `200` (forwarded response from auth-service)
- `X-RateLimit-Limit: 200`, `X-RateLimit-Remaining: 199` on first request
- `X-Request-Id` present on all responses

## Definition of done — met

- All public traffic flows through gateway on port 8000 ✅
- Invalid/missing JWT rejected at gateway; downstream services never reached ✅
- Rate limiting active; `429` returned when burst exceeded ✅
- Circuit breaker trips after 5 consecutive upstream 5xx; returns `503 CIRCUIT_OPEN` ✅
- Request ID propagated to all downstream services ✅
