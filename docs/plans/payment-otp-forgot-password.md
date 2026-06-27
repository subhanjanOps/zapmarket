# Payment, Email, OTP & Forgot-Password Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace fake payment/email stubs with real Stripe + Resend integrations; add email+phone OTP verification on registration; extend forgot-password to support phone-based OTP reset.

**Architecture:** All changes are confined to `auth-service` (email, OTP, forgot-password) and `payment-service` (Stripe). The `PaymentGateway` and `Emailer` interfaces are unchanged — only new concrete implementations are added and wired in `main.go` / `cmd/main.go`. A single new `otp_verifications` table in `userauth` DB backs all OTP flows; the existing `password_reset_tokens` table is reused for email-based reset links.

**Tech Stack:** Go 1.23, `stripe-go/v82`, `resend-go/v2`, `twilio-go`, `database/sql`, existing `pkg/errors`, `pkg/crypto`.

## Global Constraints

- No ORM — raw `database/sql` only.
- No new shared packages — changes live inside each service's `internal/`.
- All secrets from env vars; no hardcoding.
- Existing `Emailer` and `PaymentGateway` interfaces must not change signatures.
- Swagger annotations required on all new HTTP endpoints.
- Generic 200 responses on forgot-password to avoid account-existence leakage.
- `IsVerified` on `User` set to `false` at registration; flipped to `true` after email OTP confirmed.

---

## Feature 1: Stripe Payment Gateway

### Files
- **Create:** `services/payment-service/internal/gateway/stripe_gateway.go`
- **Modify:** `services/payment-service/internal/handler/http/webhook_handler.go` — replace placeholder `HandlePaymentWebhook` with real Stripe signature verification + event dispatch
- **Modify:** `services/payment-service/main.go` — swap `NewFakePaymentGateway()` for `NewStripeGateway()` when `STRIPE_SECRET_KEY` is set
- **Modify:** `services/payment-service/pkg/config/config.go` (or shared `pkg/config`) — add `StripeSecretKey`, `StripeWebhookSecret`

### New env vars
```
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...
```

### No DB changes.

### Implementation notes
- `StripeGateway.Charge` → `stripe.PaymentIntent.New` with `Confirm: true`, `PaymentMethod: pm_card_visa` (test) or a real `payment_method_id` passed in (requires `ChargeResult` to carry `ClientSecret` for front-end confirmation — but since the existing interface only returns `GatewayTxnID`, use `AutomaticPaymentMethods` with `Allow: true` and server-side confirmation).
- `StripeGateway.Refund` → `stripe.Refund.New`.
- Webhook handler: verify `Stripe-Signature` header with `webhook.ConstructEvent`, dispatch `payment_intent.succeeded` → `svc.MarkCaptured`, `payment_intent.payment_failed` → `svc.MarkFailed`.
- Fallback: if `STRIPE_SECRET_KEY == ""`, wire `FakePaymentGateway` (preserves dev/test workflow).

---

## Feature 2: Real Email Service (Resend)

**Choice: Resend** — Go SDK (`resend-go/v2`), free tier 3 000 emails/month, no SMTP config, single API key. SMTP already partially built but requires TLS setup; Resend is zero-friction. `LogEmailer` stays as fallback when `RESEND_API_KEY == ""`.

### Files
- **Create:** `services/auth-service/internal/email/resend_emailer.go`
- **Modify:** `services/auth-service/internal/email/emailer.go` — add `SendOTPEmail(ctx, to, otp string) error` and `SendPasswordResetSMSFallback` (no-op default) to the `Emailer` interface
- **Modify:** `services/auth-service/cmd/main.go` — wire `ResendEmailer` when `RESEND_API_KEY` is set, else `LogEmailer`

### New env vars
```
RESEND_API_KEY=re_...
EMAIL_FROM=noreply@zapmarket.io
```

### Emailer interface additions
```go
SendOTPEmail(ctx context.Context, to, otp string) error
```
(existing `SendPasswordResetEmail` stays unchanged)

### No DB changes.

---

## Feature 3: Email + Phone OTP Verification on Registration

### Files
- **Create:** `services/auth-service/migrations/0006_otp_verifications.up.sql`
- **Create:** `services/auth-service/migrations/0006_otp_verifications.down.sql`
- **Create:** `services/auth-service/internal/repository/otp_repository.go`
- **Create:** `services/auth-service/internal/domain/contracts/otp_repository.go`
- **Modify:** `services/auth-service/internal/domain/models.go` — add `OTPVerification` struct
- **Modify:** `services/auth-service/internal/service/auth_service.go` — add `SendEmailOTP`, `VerifyEmailOTP` methods; change `RegisterUserPassword` to set `IsVerified = false` and auto-send email OTP
- **Modify:** `services/auth-service/internal/handler/http/auth_service_iface.go` — add `SendEmailOTP`, `VerifyEmailOTP`
- **Modify:** `services/auth-service/internal/handler\http\handlers.go` (or new file) — add `POST /v1/auth/otp/send` and `POST /v1/auth/otp/verify` handlers
- **Modify:** `services/auth-service/cmd/main.go` — wire `OTPRepository` into `AuthService`

### New env vars
```
OTP_EXPIRY_MINUTES=10   # default 10
```

### DB migration (`0006_otp_verifications.up.sql`)
```sql
CREATE TABLE otp_verifications (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash     TEXT NOT NULL,
    purpose       TEXT NOT NULL CHECK (purpose IN ('email_verify','phone_verify','phone_password_reset')),
    recipient     TEXT NOT NULL,  -- email or E.164 phone number
    expires_at    TIMESTAMPTZ NOT NULL,
    used_at       TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_otp_user_purpose ON otp_verifications(user_id, purpose, used_at);
```

### New API endpoints
| Method | Path | Auth | Body | Description |
|--------|------|------|------|-------------|
| POST | `/v1/auth/otp/send` | none | `{"purpose":"email_verify"}` | Resend OTP to registered email |
| POST | `/v1/auth/otp/verify` | none | `{"purpose":"email_verify","code":"123456"}` + `Authorization: Bearer <access_token>` | Verify OTP, flip `is_verified=true` |

### Flow
1. `RegisterUserPassword` creates user with `is_verified=false`, generates 6-digit OTP, stores SHA-256 hash in `otp_verifications`, sends via `emailer.SendOTPEmail`.
2. `POST /v1/auth/otp/verify` — user submits code; service looks up latest un-used OTP for user+purpose, compares hash, sets `used_at=NOW()` and `users.is_verified=true`.
3. Callers may still log in when `is_verified=false` — the flag is informational for this iteration (enforce at checkout/order time if desired).

### SMS OTP for phone (optional at registration)
- Only triggered if `Phone` field is provided in registration request.
- Requires Twilio (see Feature 4 for wiring). If `TWILIO_ACCOUNT_SID == ""`, skip silently and log.

---

## Feature 4: Forgot Password — Phone OTP + Email Reset Link

### Files
- **Modify:** `services/auth-service/internal/handler/http/password_handler.go` — extend `ForgotPasswordRequest` to accept `phone` OR `email`; add `POST /v1/auth/password/forgot-otp` + `POST /v1/auth/password/reset-otp` handlers
- **Modify:** `services/auth-service/internal/service/auth_service.go` — add `RequestPasswordResetOTP(ctx, phone string) error`, `ResetPasswordWithOTP(ctx, phone, otp, newPassword string) error`
- **Modify:** `services/auth-service/internal/handler/http/auth_service_iface.go` — add the two new methods
- **Create:** `services/auth-service/internal/sms/smser.go` — `SMSer` interface + `TwilioSMSer` + `LogSMSer` (dev fallback)
- **Modify:** `services/auth-service/internal/service/auth_service.go` — inject `SMSer`
- **Modify:** `services/auth-service/cmd/main.go` — wire `TwilioSMSer` when env vars present

### New env vars
```
TWILIO_ACCOUNT_SID=AC...
TWILIO_AUTH_TOKEN=...
TWILIO_FROM_NUMBER=+1...
```

### No DB changes (reuses `otp_verifications` with `purpose='phone_password_reset'`).

### New API endpoints
| Method | Path | Auth | Body | Description |
|--------|------|------|------|-------------|
| POST | `/v1/auth/password/forgot` | none | `{"email":"..."}` | **existing** — email reset link |
| POST | `/v1/auth/password/forgot-otp` | none | `{"phone":"+91..."}` | Phone OTP reset initiation |
| POST | `/v1/auth/password/reset-otp` | none | `{"phone":"+91...","code":"123456","new_password":"..."}` | Verify OTP + set new password |

### Flow (phone branch)
1. `POST /v1/auth/password/forgot-otp` — look up user by phone; if found, generate OTP, store hash in `otp_verifications` with `purpose='phone_password_reset'`, send SMS. Always return `200` with generic message.
2. `POST /v1/auth/password/reset-otp` — look up user by phone, verify OTP hash+expiry+unused, hash new password, update `users.password_hash`, mark OTP used. Return `200`.

### SMSer interface
```go
type SMSer interface {
    SendOTP(ctx context.Context, to, otp string) error
}
```

---

## Decision Points (Needs Confirmation)

1. **Stripe Charge API**: Current `Charge` signature takes `amount, currency, idempotencyKey` but returns only `GatewayTxnID`. Real Stripe PaymentIntents require a `payment_method` string (card token from frontend). **Recommended default**: accept `payment_method_id` via the existing gRPC request payload's metadata/options field, pass it through to Stripe. Alternatively, keep server-side confirmation with a hardcoded test card for now and note this as a frontend-wiring TODO.

2. **OTP enforcement**: Should unverified users be blocked from placing orders? **Recommended default**: soft-block only — `is_verified=false` is visible in JWT claims; order-management can enforce it independently.

3. **SMS provider**: Twilio free trial requires verified recipient numbers (can't SMS arbitrary numbers). **Alternative**: Vonage/Nexmo (similar) or skip SMS OTP and do email-only OTP for phones too. **Decision: Twilio with `LogSMSer` fallback for dev** — revisit if Twilio free-trial friction is too high; alternative is Vonage (same Go SDK pattern) or email-only OTP for phone flows.

4. **Resend sender domain**: Free tier requires `onboarding@resend.dev` as sender unless a domain is verified. Fine for dev; production needs a verified domain.
