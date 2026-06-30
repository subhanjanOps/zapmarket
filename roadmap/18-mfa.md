# Plan 18 — MFA (TOTP) at Login

**Effort:** M | **Impact:** M | **Depends on:** Plan 09

## Context
No MFA exists. Login is email+password only (or OAuth). For sellers and admins handling payment data, TOTP is the minimum acceptable second factor. Buyers can opt in.

## Scope
- TOTP enrollment and verification in auth-service
- Enforce MFA for `seller` and `admin` roles after login
- Optional MFA for `buyer` role

## Tasks
- [ ] Add `go get github.com/pquerna/otp` to auth-service go.mod
- [ ] Migration: add `totp_secret TEXT` (encrypted at rest), `totp_enabled BOOL DEFAULT FALSE`, `totp_enrolled_at TIMESTAMPTZ` to `users` table
- [ ] `POST /auth/mfa/enroll`: generate TOTP secret, return QR code URL (`otpauth://totp/...`) and backup codes
- [ ] `POST /auth/mfa/verify-enrollment`: accept TOTP code, set `totp_enabled = TRUE`
- [ ] `POST /auth/mfa/disable`: require current TOTP code to disable; admin can force-disable
- [ ] Modify login flow: on password success, if `totp_enabled = TRUE`, return `MFA_REQUIRED` with a short-lived `mfa_session_token` (5 min TTL in Redis) instead of full JWT
- [ ] `POST /auth/mfa/challenge`: accept `mfa_session_token` + TOTP code, issue full JWT on success
- [ ] Enforce MFA required for `seller` and `admin`: if `role IN ('seller','admin')` and `totp_enabled = FALSE`, post-login redirect to `/auth/mfa/enroll` (return `MFA_SETUP_REQUIRED` in login response)
- [ ] Backup codes: generate 8 one-time codes on enrollment, store hashed in `user_backup_codes` table; allow use in place of TOTP

## Done criteria
- Seller/admin cannot complete login without TOTP after enrollment
- Enrollment produces a valid QR code scannable by Google Authenticator / Authy
- Backup codes work as TOTP substitute (each usable once)
- Buyers can optionally enroll but are not forced
