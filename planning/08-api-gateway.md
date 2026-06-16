# Stage 8 — API Gateway

Corresponds to checklist **Phase 13**.

## Goal

Stand up a single public ingress so no individual service is directly internet-reachable,
per `design.md`'s architecture diagram. Sequenced after every service exists (Stages
3-7) because there's nothing meaningful to route to before that.

## Preconditions
- Stages 3-7 merged — Inventory, Payment, Order, Notification all running; Auth and
  Catalog already running.

## Tasks

### 8.1 Technology choice
- [ ] `design.md` lists Kong, Envoy, or Nginx+Lua as options. Recommend a thin Go service
      using `go-chi` (already a project convention via `product-catalog-service`) with
      `httputil.ReverseProxy` for routing, rather than introducing a new technology/runtime
      (Kong/Envoy) into a Go-only monorepo — confirm this choice with the user before
      building, since it's a meaningful architecture decision the checklist leaves open.

### 8.2 Routing
- [ ] Path-prefix routing per `design.md`'s table: `/users/*` → Auth, `/products/*` →
      Catalog, `/orders/*` → Order Management. Inventory and Payment are **not** routed
      publicly (no REST surface on them per design — they're gRPC-internal-only).

### 8.3 JWT validation
- [ ] Gateway calls Auth's `ValidateToken` gRPC RPC on every request (same pattern
      `product-catalog-service`'s `AuthMiddleware` already uses) — reuse that middleware
      code by promoting it to `pkg/middleware` if it isn't already shared.

### 8.4 Rate limiting
- [ ] Per-user and per-IP counters — `design.md` specifies Redis-backed, but Redis isn't
      live until Stage 9. Use an in-memory token-bucket limiter for this stage and swap to
      Redis-backed in Stage 9 (same sequencing pattern as idempotency keys in Stages 4-5).

### 8.5 Cross-cutting middleware
- [ ] Correlation/request ID generation and propagation (header injected at the gateway,
      forwarded to downstream services' logger context via `pkg/logger`).
- [ ] Circuit breakers per downstream service (e.g. `sony/gobreaker`) so one slow/down
      service doesn't cascade-fail the gateway.
- [ ] Request logging middleware.

## Out of scope
- TLS termination details / cert management — Stage 12 (production readiness).
- Service discovery beyond static config (env vars pointing at each service's address) —
  dynamic discovery (Consul/k8s DNS) is implicitly handled once Stage 11's k8s manifests
  exist; no separate discovery mechanism needed before that.

## Definition of done
- All public traffic flows through the gateway; direct calls to service ports still work
  for local debugging but the intended client path is gateway-only.
- A request with an invalid/expired JWT is rejected at the gateway, never reaching
  downstream services.
- A burst of requests past the configured rate limit gets `429`s.
