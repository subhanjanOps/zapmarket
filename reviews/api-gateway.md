# Code Review: `api-gateway`

**Reviewer:** Principal Engineer
**Date:** 2026-06-21
**Branch:** `features/cluster-setup`
**Verdict:** CHANGES REQUIRED

---

## Executive Summary

Well-conceived infrastructure with strong distributed systems thinking: atomic router hot-reload via `sync/atomic.Pointer`, Redis sliding-window rate limiting with Lua, per-upstream circuit breaking, Postgres LISTEN/NOTIFY for near-instant route propagation, and a clean audit pipeline. However: a closure capture bug that silently misroutes all traffic, XFF spoofing that bypasses rate limiting and blocklisting, committed `.env`, and a double-write on circuit open block production deployment.

---

## Critical Findings

### CRIT-1: Closure Capture Bug in `buildRouter` — All Routes Route to Last Upstream
**File:** `main.go:166`
```go
for _, route := range loader.Routes() {
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        addr, ok := resolve(r.Context(), route.Upstream)  // captures loop variable
```
All closures capture the same `route` variable — at serve time, every route resolves to the **last route's** upstream. Silent traffic misrouting.
**Fix:** Add `route := route` inside the loop body before the closure.

### CRIT-2: X-Forwarded-For Spoofing — Rate Limiter and Blocklist Can Be Bypassed
**Files:** `internal/middleware/ratelimit.go:97-110`, `internal/middleware/blocklist.go:16-24`
Both `ipKey` and `ClientIP` unconditionally trust client-supplied `X-Forwarded-For`. Any client can send `X-Forwarded-For: 1.2.3.4` to impersonate an IP and get a fresh rate-limit bucket or bypass the blocklist entirely.
**Fix:** Since the gateway IS the edge, use `r.RemoteAddr` as canonical IP. Only trust XFF from verified proxy CIDRs.

### CRIT-3: Committed `.env` File With Credentials
**File:** `services/api-gateway/.env`
Contains `DB_PASSWORD=zappass123` and Redis/auth service addresses. Must be removed from git history and `.gitignore`d.

### CRIT-4: Missing `rows.Err()` Check After `rows.Next()` Loop
**File:** `internal/admin/handler.go:279-291` (also `getStats:379-387`)
If the DB connection is severed mid-iteration, `rows.Next()` returns false and the handler returns a **truncated result set as if it were complete** — silent data corruption in audit responses.

---

## High Priority Findings

### HIGH-1: `updateRoute` Builds Dynamic SQL via String Concatenation and Leaks Raw DB Errors
**File:** `internal/admin/handler.go:139-196`
`id` path param not validated as UUID before being appended to args. Raw `err.Error()` returned to callers exposes internal schema details.

### HIGH-2: `queryAudit` Leaks Raw DB Errors to Admin Callers
**File:** `internal/admin/handler.go:272`
```go
jsonErr(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
```
Raw Postgres errors include table names, column names, constraint names.

### HIGH-3: Double Write to `ResponseWriter` When Circuit Opens After 5xx
**File:** `internal/proxy/proxy.go:54-74`
`rp.ServeHTTP(rec, r)` writes headers+body to `w`. If circuit then opens on next call and returns `ErrOpenState`, code writes a second response to already-flushed `ResponseWriter` — corrupted HTTP response.
**Fix:** `responseRecorder` must buffer and only flush to `w` after `breaker.Execute` returns.

### HIGH-4: gRPC Connection to Auth-Service Uses `insecure.NewCredentials()`
**File:** `internal/middleware/auth.go:31`
All JWT validation traffic is plaintext. Add `GATEWAY_AUTH_TLS=true/false` config toggle; document the plaintext assumption explicitly.

### HIGH-5: `AutoBinder.seen` Map Has No Mutex — Latent Data Race
**File:** `internal/registry/autobind.go:44-63`
Plain `map[string]string` with no synchronization. Safe today (single goroutine), but a race waiting to surface if `reconcile` is ever called concurrently.

### HIGH-6: Shutdown Order Cancels Context After Dependent Goroutines Are Still Running
**File:** `main.go:286-296`
`cancel()` is called after `srv.Shutdown()` — but `loader.Watch`, `auditWriter.Run`, `binder.Run` all use the same `ctx` and continue writing to DB/Redis while the HTTP server drains. Correct order: stop accepting → drain → cancel ctx → close auth gRPC → close Redis → close DB.

---

## Medium Priority Findings

- **MED-1:** `main()` is 264 lines — violates 80-line maximum; extract `setupDB`, `setupRedis`, `setupAuthMiddleware`, `buildRouter`, `buildHTTPServer`
- **MED-2:** `buildRouter` closure rebuilt on 1-second poll despite LISTEN/NOTIFY already providing instant notification — increase to 5-10s or trigger only on notification
- **MED-3:** `getProxy` cache in `proxy.go` is unbounded — stale `ReverseProxy` for changed upstream addresses never evicted
- **MED-4:** Redis errors in rate limiter fail-open with no log call — Redis outage makes rate limiting silently disappear
- **MED-5:** `createRoute` returns HTTP 409 for all DB errors including connection errors — should distinguish `23505` uniqueness errors from others
- **MED-6:** `removeFromBlocklist` doesn't validate IP format — should use `net.ParseIP` like `addToBlocklist` does
- **MED-7:** Audit writer uses transaction per batch for append-only inserts — unnecessary; use multi-row VALUES or COPY
- **MED-8:** `RedisRegistry.Pick` issues full SCAN on every proxied request — needs in-memory cache with background refresh
- **MED-9:** `routes.BuildDSN()` called twice — second call opens invisible `pq.NewListener` connection never closed on shutdown
- **MED-10:** CORS middleware missing `Access-Control-Allow-Credentials: true` — blocks credentialed cross-origin requests

---

## Low Priority Findings

- **LOW-1:** `X-Request-ID` trusted from client without length limit or sanitization — could pollute audit logs
- **LOW-2:** `getStats` discards DB errors with `_ =` — stats endpoint returns zeros without any error signal on DB outage
- **LOW-3:** `tracker.Track` latency includes circuit-breaker overhead, not just proxy latency
- **LOW-4:** `probeRoute` uses `LIKE` with inverted pattern — prevents index usage on `path_prefix`
- **LOW-5:** `metrics/tracker.go` uses growing slice with lock-held trim — should use ring buffer
- **LOW-6:** No `*_test.go` files anywhere in the service
- **LOW-7:** `envOrDefault` duplicated in `main.go` and `routes/loader.go`
- **LOW-8:** `EventCircuitOpen` constant defined but circuit-open events are never audited
- **LOW-9:** `listRegistry` creates new `http.Client` per request — should be stored on `Handler`

---

## Recommended Refactoring Plan

**Phase 1 — Immediate (Block Merge):**
1. Fix CRIT-1: Add `route := route` inside for-range loop in `buildRouter`
2. Fix CRIT-2: Strip client `X-Forwarded-For` at gateway edge; use `r.RemoteAddr`
3. Fix CRIT-3: Remove `.env` from git, rotate credentials, add to `.gitignore`
4. Fix CRIT-4: Add `rows.Err()` checks in all `rows.Next()` loops
5. Fix HIGH-3: Buffer `responseRecorder` — only flush to real `ResponseWriter` after `Execute` returns

**Phase 2 — Before Production:**
6. Add UUID validation on `id` path param; return opaque error codes (no raw DB errors)
7. Add `GATEWAY_AUTH_TLS` config toggle; document plaintext assumption
8. Add `sync.Mutex` to `AutoBinder.seen`
9. Fix shutdown order — cancel ctx before closing dependencies
10. Add in-memory instance cache to `RedisRegistry`; `Pick` serves from cache
11. Add structured log warning for Redis errors in rate limiter

**Phase 3 — Quality:**
12. Refactor `main()` into `App` struct with setup methods
13. Add unit tests for `middleware/`, `registry/autobind.go`, `proxy/`
14. Add `Access-Control-Allow-Credentials: true` to CORS middleware
15. Add eviction to `proxies` map on circuit-open
16. Emit `EventCircuitOpen` audit event from proxy
17. Promote `http.Client` in `listRegistry` to `Handler` field

---

## Final Verdict: CHANGES REQUIRED
