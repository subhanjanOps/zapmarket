# Phase 3 — Auth Hardening

**Goal:** Close the remaining auth gaps: refresh-token BFF flow (so UIs don't expire after 1 hour), email verification, password reset, and auth-endpoint rate limiting.

---

## 3.1 Refresh Token BFF Flow

**Problem:** All three UIs store the `access_token` in an httpOnly cookie with `maxAge` matching the 1-hour JWT expiry. After 1 hour, API calls silently return 401. The browser has no mechanism to renew the token without a full login.

The auth-service already issues refresh tokens (stored in DB, 7-day TTL). We just need a BFF route to use them.

### Implementation

**New BFF route in each UI:** `POST /api/auth/refresh`

```ts
// app/api/auth/refresh/route.ts
export async function POST(req: NextRequest) {
  const refreshToken = req.cookies.get("*_refresh_token")?.value;
  if (!refreshToken) return NextResponse.json({ error: "No refresh token" }, { status: 401 });

  const upstream = await fetch(`${GW}/v1/auth/refresh`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });

  const data = await upstream.json();
  if (!upstream.ok) return NextResponse.json({ error: data.error }, { status: upstream.status });

  const res = NextResponse.json({ ok: true });
  res.cookies.set("*_token", data.access_token, { httpOnly: true, maxAge: 60 * 60 });
  return res;
}
```

**Cookie additions at login:**
- Set `*_refresh_token` as httpOnly, `maxAge: 60 * 60 * 24 * 7` (7 days)
- Set `*_token` as httpOnly, `maxAge: 60 * 60` (1 hour)

**Client-side intercept:** In each UI's API lib, wrap all fetch calls:

```ts
async function apiFetch(url, opts) {
  let res = await fetch(url, opts);
  if (res.status === 401) {
    const refresh = await fetch("/api/auth/refresh", { method: "POST" });
    if (refresh.ok) res = await fetch(url, opts);  // retry once
    else { window.location.href = "/login"; return; }
  }
  return res;
}
```

**Logout:** Clear both `*_token` and `*_refresh_token` cookies.

---

## 3.2 Email Verification

**Current state:** `users.is_verified` exists in schema but registration doesn't send a verification email and nothing enforces the verified state.

### Flow

1. On `RegisterUserPassword`, generate a signed verification token (HMAC-SHA256 of `user_id + created_at`, 24h TTL) and store it in a new `email_verifications` table.
2. Enqueue a `user.registered` event to Kafka via outbox.
3. `notification-service` handles `user.registered`: sends verification email with link `https://app.zapmarket.com/verify?token=<token>`.
4. New endpoint `GET /v1/auth/verify?token=<token>`: validates token, sets `is_verified = true`, deletes verification row.
5. Optionally block seller-specific actions (e.g., product creation) until verified.

### Schema

```sql
CREATE TABLE email_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Email provider

Wire `notification-service` to send actual emails via SMTP (use `net/smtp` for simplicity, or `resend.com` API). Add `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD` to `notification-service` env.

---

## 3.3 Password Reset

### Flow

1. `POST /v1/auth/forgot-password` — receives `email`, generates a reset token (stored in `password_resets` table, 1-hour TTL), publishes `user.password_reset_requested` event.
2. `notification-service` handles the event: sends email with link `https://app.zapmarket.com/reset-password?token=<token>`.
3. `POST /v1/auth/reset-password` — receives `token + new_password`: validates token, bcrypt-hashes new password, updates user, invalidates all existing refresh tokens for that user, deletes the reset row.

### Schema

```sql
CREATE TABLE password_resets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Security notes

- Token is random 32 bytes, stored as SHA-256 hash (same pattern as refresh tokens).
- `POST /v1/auth/forgot-password` always returns 200 regardless of whether the email exists (prevents user enumeration).
- Reset invalidates all existing refresh tokens: `UPDATE refresh_tokens SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL`.

---

## 3.4 Auth Endpoint Rate Limiting

**Problem:** `/v1/auth/login` and `/v1/auth/register` have no endpoint-specific rate limits beyond the global 200 req/min/IP applied by the api-gateway middleware. A credential stuffing attack is not throttled meaningfully.

### Implementation in api-gateway

Extend the existing Redis sliding-window rate limiter to support per-route overrides:

```go
// In gateway_routes table: add a rate_limit_override column (nullable)
// e.g., for /v1/auth/login: rate_limit_override = '{"per_ip": 5, "window_sec": 60}'
```

Alternative (simpler): hardcode tighter limits in a `authRateLimiter` middleware applied only to auth routes:

- `/v1/auth/login`: 5 attempts / IP / 60s
- `/v1/auth/register`: 10 attempts / IP / 60s
- `/v1/auth/forgot-password`: 3 attempts / IP / 60s

On limit exceeded: return 429 with `Retry-After` header.

### Lockout on repeated failures

In `auth-service` `LoginPassword`: after 5 consecutive failed attempts for the same email within 15 minutes, temporarily lock the account for 15 minutes. Store lockout state in Redis: `auth:lockout:<user_id>` with TTL 15 minutes.

---

## 3.5 Session Management UI

Add a `/dashboard/security` page to each seller/admin UI showing:
- Active sessions (refresh tokens): device hint (user-agent), created_at, last_used_at
- "Revoke" button per session
- "Revoke all" button

Requires new endpoints:
- `GET /v1/auth/sessions` — list non-expired refresh tokens for the current user
- `DELETE /v1/auth/sessions/:id` — revoke a specific refresh token

---

## Acceptance criteria

- Access token refresh happens transparently; user never sees an expired-session 401
- Verification email is sent within 30 seconds of registration
- Password reset link expires after 1 hour and cannot be reused
- Login endpoint returns 429 after 5 failed attempts from the same IP within 60 seconds
- All new endpoints have unit + integration tests

---

## Estimated effort

2–3 weeks (refresh flow: 3 days; email verification: 3 days; password reset: 2 days; rate limiting: 3 days; session management: 4 days).
