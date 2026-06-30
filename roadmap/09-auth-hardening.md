# Plan 09 — Auth Hardening

**Effort:** S | **Impact:** H | **Depends on:** none

## Context
Three security gaps on login that do not require Kafka:
1. `/auth/login` and `/auth/register` have no rate limiting (IPRateLimit middleware exists on OTP routes — this is a one-line addition)
2. No account lockout after N failed login attempts (no attempt counter in schema)
3. No delivery address on orders (moved to Plan 04 — already covered)

These are the highest-risk omissions for a payment-handling marketplace.

## Tasks

### Rate limiting on login (auth-service)
- [ ] Apply existing `IPRateLimit` middleware to `POST /auth/login` route in `cmd/main.go`
- [ ] Apply existing `IPRateLimit` middleware to `POST /auth/register` route
- [ ] Tune limit: 5 attempts per IP per minute (vs OTP routes which are 3/min)

### Account lockout (auth-service)
- [ ] Migration: add `failed_login_attempts INT NOT NULL DEFAULT 0`, `locked_until TIMESTAMPTZ` to `users` table
- [ ] In login handler: on wrong password, `UPDATE users SET failed_login_attempts = failed_login_attempts + 1 WHERE id = $1`
- [ ] If `failed_login_attempts >= 5`: set `locked_until = now() + interval '30 minutes'`
- [ ] On login attempt: if `locked_until > now()`, return 429 with `ACCOUNT_LOCKED` error code and time remaining
- [ ] On successful login: reset `failed_login_attempts = 0`, `locked_until = NULL`
- [ ] On password reset success: also reset lockout fields

### Session visibility (optional but cheap)
- [ ] Migration: add `user_agent TEXT`, `ip_address INET`, `last_used_at TIMESTAMPTZ` to `refresh_tokens` table
- [ ] Populate from request headers on token issue
- [ ] Add `GET /auth/sessions` endpoint returning active refresh tokens (for "sign out all devices")

## Done criteria
- 6th consecutive failed login returns 429 with `ACCOUNT_LOCKED`
- Lockout clears after 30 minutes or on password reset
- `/auth/login` and `/auth/register` return 429 on IP rate limit breach
- Successful login resets attempt counter
