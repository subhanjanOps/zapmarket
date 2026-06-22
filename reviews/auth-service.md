# auth-service Review

## Executive Summary

`auth-service` is the security keystone of ZapMarket: it issues JWTs, validates them over gRPC for every other service, enforces RBAC, and now stores user preferences (Plan C). The service is generally well-structured — it uses constructor injection, depends on repository interfaces (DIP honoured), keeps SQL out of business logic, and never logs token bodies. The new preferences feature is clean, validated, and ships with tests.

However, several **security and correctness issues** undermine the "entry point for all users" role:
- No **refresh-token rotation** — a leaked refresh token stays valid for its full multi-day lifetime.
- **OAuth `state` is generated from `time.Now()` and never validated** in callbacks (CSRF gap).
- **Logout reports success even when the access-token blacklist silently no-ops** (Redis is optional).
- The gRPC `ValidateToken` hot path — called on every cross-service request — does a **DB round-trip per call with no caching**, making auth's database the platform throughput ceiling.

Layering deviates from `architecture-principles.md` (no `application/`/`infrastructure/`/`interfaces/` split), but this is consistent across the monorepo and acceptable as a house convention. Observability is missing the `trace_id`/`request_id`/`correlation_id` the standards mandate.

**Verdict: CHANGES REQUIRED.**

---

## Critical Findings

### C1. No refresh-token rotation; revocation path untested
`RefreshAccessToken` (`internal/service/auth_service.go:130`) issues a new access token but returns the **same** refresh token — it is neither rotated nor re-stored. A leaked refresh token remains valid for its full lifetime with no reuse detection. Standard practice for a public auth service is rotating refresh tokens: on each refresh revoke the presented token and issue a new one; on presentation of an already-revoked token, invalidate the user's entire token family (replay defense).

Compounding this, the only revocation mechanism is `Logout -> InvalidateUserTokens`, and **there is no test** exercising "revoke then attempt refresh." The revocation check itself lives in `GetRefreshTokenByHash` (`refresh_token_repository.go:94`) and does work today, but it is load-bearing and uncovered.

### C2. `RefreshToken.TokenHash` holds a *raw* token — latent secret-handling footgun
`generateRefreshToken` (`auth_service.go:225`) sets `refreshToken.TokenHash = tokenString`, overwriting the hash with the **raw** JWT before returning it to the client (also done in the gRPC paths, `auth_server.go:150,193`). The field named `TokenHash` therefore sometimes contains a hash (from the repo) and sometimes the raw secret (after this assignment). Any future code that trusts `TokenHash` to be a hash will compare or log a raw credential. Split into distinct `Token` (raw, never stored) and `TokenHash` (stored) fields.

### C3. OAuth `state` is predictable and never validated (CSRF)
`GoogleOAuthURL`/`FacebookOAuthURL` (`handlers.go:465,530`) derive `state` from `time.Now().String()` — predictable, not random — and the callbacks (`GoogleOAuthCallback:494`, `FacebookOAuthCallback:559`) read only `code`, never verifying `state`. This is a textbook OAuth login-CSRF vulnerability. Generate `state` with `crypto/rand`, persist it (signed cookie or Redis keyed to the flow), and reject callbacks whose `state` doesn't match.

---

## High Priority Findings

### H1. gRPC `ValidateToken` hot path: DB read per call, no caching
`ValidateAccessToken` (`auth_service.go:155`) calls `userRepo.GetUserByID` (a DB query) on **every** validation. Since every other service calls `ValidateToken` over gRPC for every authenticated request, auth's database becomes the throughput ceiling for the whole platform. Mitigate with a short-TTL Redis cache of user-by-ID (Redis is already wired in `main.go`), or trust signed JWT claims for routine authz and DB-check only on sensitive operations.

### H2. Logout returns 200 even when the access token is never blacklisted
`Logout` (`auth_service.go:183`) blacklists the access token only if `s.rdb != nil`. When Redis is down — an explicitly supported mode (`main.go:97`) — `BlacklistToken` is a silent no-op, the access token stays valid until natural expiry, yet the endpoint returns `200 logged out`. For a security operation this is misleading. Log a warning and/or document the guarantee; keep access-token TTLs short. (Refresh revocation still works via DB, so this is High not Critical.)

### H3. Missing tracing / correlation context (observability standard violated)
`engineering-standards.md` requires `trace_id`, `request_id`, `correlation_id` on all services. `LoggingMiddleware` (`handlers.go:63`) and the gRPC logging in `auth_server.go` log method/status/latency only — no correlation ID is generated, propagated through context, or forwarded over gRPC metadata. Cross-service auth calls are untraceable end-to-end. Add a request-ID middleware + gRPC unary interceptor.

### H4. gRPC `GetUser` maps every error to `codes.Internal`
`auth_server.go:97` returns `codes.Internal` for any `GetUserByID` error, including `USER_NOT_FOUND`. Callers cannot distinguish "not found" from a real fault. Map `pkgerrors` types to proper gRPC codes (`NotFound`, `InvalidArgument`, `Unauthenticated`), mirroring the `ValidateToken` mapping.

### H5. Input validation lives only in the HTTP handler, not the use case
Email/password presence and role checks are in `Register` (`handlers.go:222`); there is no email-format or password-strength validation anywhere, and the gRPC `RegisterUser` path (`auth_server.go:155`) bypasses even the presence checks. Per `architecture-principles.md`, invariant validation belongs in the application/use-case layer so every interface shares it. Move it into `RegisterUserPassword`.

---

## Medium Priority Findings

### M1. `BootstrapAdmin` existence check is racy (TOCTOU)
`auth_service.go:93` checks "does any admin exist?" then creates — two concurrent calls could both succeed. Low likelihood (guarded by a secret), but enforce at the DB with a partial unique index (`WHERE role='admin'`) or a transaction.

### M2. Auth logic duplicated across handlers; production test-backdoor
`PreferencesHandler.resolveUserID` (`preferences_handler.go:39`) re-implements the Bearer-parse + `ValidateAccessToken` block already in `AdminAuthMiddleware` (`admin_handler.go:30`), `Me` (`handlers.go:345`), and `Logout` (`handlers.go:433`) — four copies. Extract a single `Authenticate` middleware that injects the user into context. The `PrefsUserIDKey` context backdoor (`preferences_handler.go:23`) is test-only logic shipped in production code; prefer an injected authenticator interface.

### M3. Preferences routes rely on handler-internal auth, unlike admin routes
`main.go:143` wires preferences with only `LoggingMiddleware`; auth is done inside the handler, whereas admin routes use a real middleware (`main.go:140`). Functionally safe but inconsistent — easy to forget auth on the next endpoint added to this group.

### M4. `user_preferences` migration: open KV table, no `created_at`, no length guard
`0004_user_preferences.up.sql` is correct in mechanics (FK `ON DELETE CASCADE`, composite PK, idempotent `IF NOT EXISTS`), but models a generic `value TEXT` KV store for what is currently a single 3-char currency. Add `created_at`, and a `CHECK`/length bound (or a typed column) since the DB currently accepts arbitrary-length text for any key.

### M5. `Refresh` handler collapses all errors to 401
`handlers.go:324` returns 401 for any `RefreshAccessToken` error, including the internal access-token-generation failure (`auth_service.go:148`), masking 500s. Use `pkgerrors.HandleHTTP` like the other handlers.

### M6. Ignored JSON encode errors
`json.NewEncoder(w).Encode(...)` return values are dropped (`preferences_handler.go:83,112`). The standards say never ignore errors — at least log encode failures.

### M7. `GetPreferences` returns `{}` when unset
`preferences_handler.go:77` returns an empty object on no stored currency, pushing the default-handling onto every client. Returning an explicit platform default would centralize it.

---

## Low Priority Findings

- **L1.** Stale comments in `auth_server.go:21-27` ("placeholder ... until proto stubs are generated", "Uncomment after proto generation") — stubs exist and are used. Remove.
- **L2.** The start/duration/slog block is duplicated across all five gRPC methods. Extract a unary interceptor (also the natural home for correlation IDs, H3).
- **L3.** `Me`/`Logout` use `strings.Split` (`handlers.go:354,435`); admin/preferences use `SplitN(...,2)` + `EqualFold`. Standardize on the case-insensitive form.
- **L4.** Conflicting Swagger annotation blocks: `cmd/main.go:16` declares `@BasePath /v1/auth`, `handlers.go:16` declares `/auth`. Consolidate.
- **L5.** `domainUserToProto` (`auth_server.go:229`) omits `SellerStatus`, so gRPC consumers cannot see seller approval state that HTTP responses include.
- **L6.** `pathID` fallback (`admin_handler.go:81`) treats any 36-char segment as a UUID; with Go 1.22 `PathValue` this branch is dead — remove the heuristic.
- **L7.** `time.Now()` is called directly throughout; injecting a clock would make token-expiry logic testable.
- **L8.** `RedisURL` config is passed as go-redis `Addr` (`main.go:95`), which expects `host:port`, not a URL — confirm/rename to avoid misconfiguration.

---

## Recommended Refactoring Plan

1. **Token security (Critical):** implement refresh-token rotation with reuse detection (C1); split `Token`/`TokenHash` and add a revoke-then-refresh test (C1/C2); randomize and validate OAuth `state` (C3).
2. **Validation hot path (High):** add Redis-backed caching (or claims-trust mode) to `ValidateAccessToken` (H1).
3. **Consolidate auth:** one `Authenticate`/`RequireRole` middleware consumed by `Me`, `Logout`, preferences, and admin; drop the `PrefsUserIDKey` backdoor and the duplicated Bearer parsing (M2, M3, L3).
4. **Move invariant validation into the service layer** shared by HTTP and gRPC (H5).
5. **Observability:** request-ID/correlation middleware + gRPC interceptor threaded through logs and outbound metadata (H3, L2).
6. **Error mapping:** map `pkgerrors` to gRPC codes (H4); use `HandleHTTP` in `Refresh` (M5).
7. **Cleanup:** remove stale proto comments, dedupe Swagger, add `SellerStatus` to proto user, log ignored encode errors, harden the migration (L1, L4, L5, M4, M6).

---

## Final Verdict

**CHANGES REQUIRED**

The architecture and the new preferences feature are solid, but for the platform's security entry point the gaps in refresh-token rotation (C1/C2), unvalidated OAuth state (C3), and the un-cached per-request DB validation on the cross-service hot path (H1) must be addressed before production. The fixes are well-contained and largely additive.
