# Code Review: `auth-service`

**Reviewer:** Principal Engineer
**Date:** 2026-06-21
**Branch:** `features/cluster-setup`
**Verdict:** CHANGES REQUIRED

---

## Executive Summary

Well-intentioned skeleton with proper layer separation, interface-based repositories, structured logging, and a functional OAuth2 flow. However, several critical production blockers exist: a `.env` file with credentials committed to the repo, gRPC error handling that swallows errors as application-level messages, a concrete service struct injected into handlers defeating interfaces, Redis bypass in OAuthService, and zero tests.

---

## Critical Findings

### C-1: `.env` File Committed With Credentials
**File:** `services/auth-service/.env`
The `.env` file contains `DB_PASSWORD`, `JWT_SECRET_KEY`, `JWT_REFRESH_SECRET_KEY`. Must be removed from git history and added to `.gitignore`.

### C-2: `OAuthService.generateRefreshTokenForUser` Instantiates a New `AuthService`, Discarding Redis State
**File:** `services/auth-service/internal/service/oauth_service.go:246`
A brand-new `AuthService` is constructed on every OAuth token issuance — the fresh instance has `rdb == nil`. Token blacklisting and Redis-backed revocation are completely bypassed for all OAuth-authenticated users.

### C-3: Handler Layer Depends on Concrete Service Structs, Not Interfaces
**Files:** `internal/handler/http/handlers.go:134-136`, `internal/handler/grpc/auth_server.go:22-25`
Both HTTP and gRPC handlers hold `*service.AuthService` / `*service.OAuthService` directly. Violates DIP; makes handlers completely untestable.

### C-4: gRPC Error Handling Swallows Errors as Application-Level Messages
**File:** `internal/handler/grpc/auth_server.go:46-57, 86-102`
Every gRPC handler returns `nil` for the error and embeds the error string in `ErrorMessage`. Callers using `status.FromError` will see `codes.OK` on failures. All gRPC methods must return `status.Error(codes.Unauthenticated, ...)` on failure.

### C-5: Zero Test Coverage
No `*_test.go` files exist anywhere in the service.

---

## High Priority Findings

### H-1: `AdminHandler` Directly Queries Repository, Bypassing Service Layer
**File:** `internal/handler/http/admin_handler.go:19-26`
Handler calls `h.userRepo.ListUsers`, `UpdateUser`, `DeleteUser` etc. directly. Business logic for role updates belongs in a service.

### H-2: `AdminAuthMiddleware` Validates JWT Locally — Blacklist Bypassed
**File:** `internal/handler/http/admin_handler.go:29-52`
Middleware calls `crypto.ValidateAccessToken` directly, skipping `AuthService.ValidateAccessToken`. A logged-out admin's blacklisted token can still access all admin routes.

### H-3: `TokenHash` Field Used to Carry Raw Token String
**File:** `internal/service/auth_service.go:225`, `internal/handler/http/handlers.go:298`
`RefreshToken.TokenHash` is mutated to hold the raw JWT string after generation — the field name lies about what it contains.

### H-4: `getGoogleUserInfo` / `getFacebookUserInfo` Use `http.Get` Without Context or Timeout
**File:** `internal/service/oauth_service.go:188, 217`
No timeout, no context propagation. A hung upstream holds a goroutine indefinitely.

### H-5: OAuth State Parameter Not Validated — CSRF Vulnerability
**File:** `internal/handler/http/handlers.go:547-549, 573-602`
State parameter is generated on URL but never validated on callback. Classic OAuth CSRF.

### H-6: `oauth_service.go` Imports `net/http` in the Service Layer
**File:** `internal/service/oauth_service.go:8`
Service layer makes outbound HTTP calls. Violates Clean Architecture — must be behind an `OAuthProviderClient` interface in infrastructure.

### H-7: `AdminBootstrap` Handler Reads Secret Directly via `os.Getenv`
**File:** `internal/handler/http/handlers.go:447`
Bypasses the centralized config layer. Should be in `Config` struct.

---

## Medium Priority Findings

- **M-1:** Duplicate Swagger annotations with inconsistent `@BasePath` (`/auth` vs `/v1/auth`)
- **M-2:** `loggingResponseWriter` logs full response body including tokens (up to 500 chars)
- **M-3:** `ListUsers`/`ListSellers` issue two queries without a transaction — COUNT/data inconsistency
- **M-4:** OAuth doesn't handle email collision — existing email/password users can't log in via OAuth
- **M-5:** gRPC `RegisterUser` hardcodes role to `RoleBuyer` — sellers can never register via gRPC
- **M-6:** `pathID` fallback uses `len(p) == 36` heuristic — dead code in Go 1.22+
- **M-7:** `User.Role` typed as `string` instead of `domain.Role` — no compile-time safety
- **M-8:** gRPC method logging is 60+ lines of duplicated boilerplate — needs interceptor

---

## Low Priority Findings

- **L-1:** `scanUserRow` wraps `sql.ErrNoRows` as `DATABASE_ERROR` instead of `NOT_FOUND`
- **L-2:** Role comparison uses string literal `"seller"` instead of `domain.RoleSeller`
- **L-3:** OAuth state uses wall clock (`time.Now()`) instead of `crypto/rand`
- **L-4:** Package named `grpc` shadows `google.golang.org/grpc`
- **L-5:** `Address` and `NotificationPreference` models are orphaned — no repo, no endpoints
- **L-6:** Health check always returns 200 OK — never probes DB
- **L-7:** gRPC server has no graceful shutdown path
- **L-8:** `loggingResponseWriter.Write` buffers all response bodies in memory

---

## Recommended Refactoring Plan

**Phase 1 — Security & Blockers:**
1. Remove `.env` from git, add to `.gitignore`, rotate secrets
2. Fix `OAuthService` — inject AuthService reference or extract `TokenIssuer` interface
3. Fix gRPC error handling to return proper `status.Error` codes
4. Implement OAuth CSRF protection with `crypto/rand` state + Redis validation
5. Fix `AdminAuthMiddleware` to enforce blacklist via service call

**Phase 2 — Architecture:**
6. Define service interfaces; make handlers depend on abstractions
7. Create `AdminService` to house admin business logic
8. Extract `OAuthProviderClient` interface in `infrastructure/oauth/`
9. Type `User.Role` as `domain.Role`

**Phase 3 — Correctness:**
10. Implement OAuth account linking for existing email users
11. Fix `getGoogleUserInfo`/`getFacebookUserInfo` with context + timeout
12. Fix `RefreshToken.TokenHash` semantics
13. Fix LIST queries with transaction or window function

**Phase 4 — Quality:**
14. Extract gRPC logging into unary interceptor
15. Implement graceful gRPC shutdown
16. Fix health check to probe DB
17. Write comprehensive unit and handler tests

---

## Final Verdict: CHANGES REQUIRED
