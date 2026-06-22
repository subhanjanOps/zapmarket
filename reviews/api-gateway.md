# api-gateway Review

## Executive Summary

The `api-gateway` is a well-conceived edge service: DB-backed routes with hot-reload via Postgres `LISTEN/NOTIFY`, gRPC auth delegation to `auth-service`, Redis sliding-window rate limiting, a circuit-breaker-backed reverse proxy, a fire-and-forget audit pipeline, and an admin API. The design instincts are mostly sound — atomic router swap, fail-open rate limiting, edge-trusted `RemoteAddr` for blocklist/rate-limit keys, and a deliberate CORS posture that disables the localhost wildcard outside development.

However, there are **two security-critical defects** and **one data race** that block production sign-off:

1. **Trusted identity-header injection** — the gateway forwards `X-User-ID/Email/Role` to upstreams but never strips client-supplied copies on `auth_mode: none` (or `method_split` GET) routes. A client can impersonate any user/admin to downstream services that trust these headers.
2. **Concurrent mutation of the shared `upstreams` map** without synchronization — `buildRouter` runs in the hot-reload goroutine while a previously-built router is still serving, and the plain map it mutates is read concurrently. This is a genuine data race that `go test -race` would flag.
3. **CORS reflects the request `Origin` for any allowed origin** while omitting `Access-Control-Allow-Credentials`; the dev-mode localhost wildcard is correctly gated, but the production reflection/credentials story needs tightening and validation.

Additionally there are **zero tests** in a service whose entire value is a security/routing control plane, and `main.go` is a ~270-line God function mixing wiring, closures, CORS, and audit middleware.

**Verdict: CHANGES REQUIRED.**

---

## Critical Findings

### C1. Client-supplied identity headers are not stripped (privilege escalation)
`internal/middleware/auth.go` lines 67-69 set `X-User-ID`, `X-User-Email`, `X-User-Role` on the request **only inside `Authenticate`**. For routes with `auth_mode: "none"` (e.g. `/v1/auth/login`, `/api/v1/categories`) and the public GET branch of `method_split`, the request reaches the proxy carrying whatever `X-User-*` headers the client sent. Per CLAUDE.md the gateway "forwards user identity to downstream services," so downstream services will trust a forged `X-User-Role: admin`.

This is a confused-deputy / header-injection vulnerability. On `required` routes the headers are merely overwritten (fine); the unauthenticated paths are wide open.

**Fix:** Unconditionally strip inbound `X-User-ID/Email/Role` (and any other trusted headers) at the very edge — e.g. in `RequestID` or a dedicated sanitizer that runs before any route handler — then let `Authenticate` re-populate them only after successful validation. Violates engineering-standards "Always validate Authentication / Authorization."

### C2. Data race on the shared `upstreams` map
`main.go` `getUpstream` (lines 126-136) reads and writes a plain `map[string]*proxy.Upstream` with no lock. `buildRouter` calls it for every route and is invoked both at startup and from the hot-reload goroutine (line 241) while the previously built router is still serving requests. Two rebuilds — or a rebuild racing the auto-binder-triggered reload — mutate the map concurrently. The map is also never pruned, so removed routes leak `Upstream` objects (and their breakers) forever.

**Fix:** Guard `upstreams` with a mutex / `sync.Map`, or build the upstream pool once outside `buildRouter` and pass it in. `go build -race` will confirm.

### C3. CORS reflection + credentials
`corsMiddleware` / `originAllowed` (lines 312-410): the dev-mode localhost wildcard gating and its inline reasoning are correct. Residual issues:
- No `Access-Control-Allow-Credentials` header is sent. Acceptable for a bearer-token, cookie-less API, but undocumented and inconsistent if any future flow uses cookies — state the bearer-only assumption explicitly.
- `EXTRA_ALLOWED_ORIGINS` is split on `,` with no validation; an empty env var yields `[""]`, saved only by the `TrimSpace != ""` guard. Brittle. Validate/normalize allowed origins at startup.

Flagged at Critical because CORS was a named focus area and origin reflection is security-sensitive.

---

## High Priority Findings

### H1. Audit drain uses a cancelled context (audit loss on shutdown)
`audit/writer.go` `Run` (lines 83-94): on `ctx.Done()` the drain calls `flush()` → `insert(ctx, ...)` with the **already-cancelled** `ctx`. `BeginTx`/`ExecContext` fail immediately with `context.Canceled`, so the final buffered audit entries are silently lost precisely during shutdown. `main.go` cancels the context (line 295) after `srv.Shutdown`, so in-flight requests can enqueue entries that then never persist.

**Fix:** Drain on `context.Background()` with a bounded timeout, decoupled from the lifecycle `ctx`.

### H2. Hot-reload watcher is a redundant 1s poll on top of LISTEN/NOTIFY
`main.go` lines 230-247 spin a 1-second ticker comparing `loader.Version()`. The loader already pushes changes via NOTIFY plus a 30s poll. The extra goroutine adds up to 1s of reload latency, rebuilds reactively on a timer instead of on change, and widens C2's race window (each rebuild re-creates the admin handler, re-parses CORS env, re-registers every route). Expose a reload channel from the loader and rebuild only on signal.

### H3. No shutdown ordering / wait for the audit writer
`main.go` lines 286-302: `srv.Shutdown` drains HTTP, then `cancel()` stops `Run`, but nothing waits for the writer's drain to finish before `main` returns and `defer db.Close()` fires — the DB can close mid-drain. Combined with H1 the shutdown path is unreliable. Use a `WaitGroup`/`done` channel and wait for the writer before closing the DB.

### H4. Route auth-mode precedence is untested at the core security boundary
Routes are sorted `length(path_prefix) DESC` in SQL (loader.go line 138) and registered most-specific-first, but chi does its own longest-match, so the SQL order is partly cosmetic. Registering both `PathPrefix+"/*"` and `PathPrefix` for overlapping prefixes across **different auth modes** (`/v1/auth` required vs `/v1/auth/login` none) can produce surprising precedence. Nothing asserts that `/v1/auth/login` resolves to `none` and `/v1/auth/profile` to `required`. This is the security boundary and must be tested (T1).

### H5. `method_split` GET branch inherits C1
`main.go` lines 204-214: the split handler runs `authMW.Authenticate(handler)` per write request (correct), but GET/HEAD take the public branch with no identity-header stripping — directly exposing C1. Test both branches.

---

## Medium Priority Findings

### M1. `main.go` is a God function (~270 lines)
Engineering standards cap functions at 80 lines. `main` mixes DB/Redis/auth setup, registry composition, the `resolve`/`getUpstream`/`buildRouter` closures, goroutine launches, and shutdown. Extract a `gateway.Server` with `New(cfg)` / `Run(ctx)`. `corsMiddleware`, `auditMiddleware`, `statusRecorder`, `originAllowed` live in `package main` and can't be unit-tested without a binary — move them into `middleware`.

### M2. `ratePerUser` tier is effectively dead code; over-limit requests are recorded
`ratelimit.go`: `rl.Limit` is a global middleware (main.go line 148) that runs **before** any per-route `Authenticate`, so `UserFromContext` is always nil there — every request is keyed per-IP and the documented `ratePerUser` (500) path never engages. Separately, the Lua script `ZADD`s the request before counting and rejects only on `count > limit`, so rejected requests still occupy slots and refresh the TTL, keeping an abuser's window permanently full. Run the limiter after identity is known (or key on validated identity), and don't record over-limit requests.

### M3. Circuit breaker / proxy write path is fragile
`proxy.go` lines 54-70: the breaker `Execute` closure returns an error on upstream 5xx purely to drive the breaker (reasonable, deserves a comment). The 5xx body is already written to the client by `rp.ServeHTTP`, and the `CIRCUIT_OPEN` body is only written on `ErrOpenState`/`ErrTooManyRequests`, so no double-write today — but `responseRecorder` doesn't track whether headers were already sent, so a future edit here easily introduces superfluous-WriteHeader bugs.

### M4. `getProxy` invalid-addr fallback masks config errors
`proxy.go` lines 90-94: a bad upstream addr falls back to `http://localhost:1` and is cached, so a misconfigured upstream fails forever as a connection error instead of surfacing the config problem. Return 502 immediately and don't cache the bad proxy.

### M5. `updateRoute` dynamic SQL assembly
`admin/handler.go` lines 176-187: values are parameterized (`$N`, no injection) but the hand-rolled `SET` builder is verbose/error-prone. An allow-list map of column → arg would read better.

### M6. `probeRoute` is an admin SSRF surface
`admin/handler.go` lines 437-521: admin-gated and capped (32KB/10s) — good — but it forwards `req.Headers` verbatim and concatenates `addr + req.Path` unvalidated. Admin compromise yields internal-network request forgery. Restrict probe targets to known upstreams and document the risk.

### M7. Registry round-robin is not stable
`registry.go` lines 114-126: `cursors[name]` increments unbounded (modulo handles wrap) but `Instances` returns Redis SCAN-order, which is non-deterministic, so rotation isn't reliable. Fine for load-spreading; don't rely on it for affinity.

---

## Low Priority Findings

- **L1.** `main.go` lines 332-338: the `originAllowed` doc comment is detached, sitting above `statusRecorder` while the function is at line 396. Move it.
- **L2.** `admin/handler.go` `getStats` uses package-global `slog.Warn` instead of an injected logger — contradicts the DI principle. Inject `*slog.Logger` into `Handler`.
- **L3.** Magic numbers throughout (rate limits 200/500/60s, breaker threshold 5, audit bufSize/batch 100, metrics 2000-sample ring). Promote to config.
- **L4.** `db.SetMaxOpenConns(10)` shares the pool across admin queries, audit batch inserts, and route refresh; may starve under load. Make configurable.
- **L5.** `loader.go` `listenLoop` retries NOTIFY silently every 5s; permanent breakage degrades to 30s polling with no observable signal. Emit a metric/health flag (observability is a required standard).
- **L6.** Three separate JSON-error helpers (`jsonError`, `jsonErr`, inline `w.Write`) across auth/admin/proxy/main. Consolidate per the "remove duplication" rule.

---

## Recommended Refactoring Plan

**Phase 1 — Security (blocking):**
1. C1/H5: Edge header-sanitizer strips inbound `X-User-*` before any handler; re-set only in `Authenticate`. Test that forged headers never reach upstreams on `none`/`method_split`-GET routes.
2. C2: Make the upstream pool concurrency-safe (or build-once) and prune on route removal.
3. C3: Document the bearer-only/no-credentials CORS contract; validate `EXTRA_ALLOWED_ORIGINS` at startup.

**Phase 2 — Reliability (blocking for prod):**
4. H1+H3: Drain audit on `context.Background()` with timeout; `WaitGroup` the writer into shutdown before `db.Close()`.
5. M2: Make `ratePerUser` actually apply (limit after identity is known) and stop recording over-limit requests.

**Phase 3 — Structure & testability:**
6. M1: Extract `gateway.Server`; move CORS/audit middleware into `middleware`.
7. H2: Replace the 1s poll with a loader reload channel.
8. L2/L6: Inject loggers; unify JSON error responses.

**Phase 4 — Tests:**
- T1: Route resolution + auth-mode precedence table tests (login→none, /v1/auth/x→required, products GET vs POST).
- T2: Middleware chain — blocklist, rate-limit fail-open on Redis error, request-id propagation, identity-header stripping.
- T3: Sliding-window rate limit against miniredis.
- T4: Auto-binder `deriveAuthMode`/`swaggerBasePath` table tests (pure functions, high value, trivially testable).
- T5: Proxy circuit-breaker open/half-open/closed transitions.

---

## Final Verdict

**CHANGES REQUIRED.**

The architecture is solid and the author clearly understands edge concerns (edge-trusted IPs, fail-open limiting, atomic swap, gated CORS wildcard). But two security defects (C1 trusted-header injection, C3 CORS), a real data race (C2), unreliable audit-on-shutdown (H1/H3), a per-user rate tier that never engages (M2), and the complete absence of tests on a security control plane mean this cannot ship as-is. Address Phases 1-2 and add the Phase 4 route/middleware tests, and this becomes APPROVED WITH RECOMMENDATIONS.
