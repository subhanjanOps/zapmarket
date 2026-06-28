# Registration Flow Plan

## Wizard Steps & Fields

| Field | Step | Buyer | Seller | Notes |
|-------|------|-------|--------|-------|
| full_name | 1 | ✓ | ✓ | |
| email | 1 | ✓ | ✓ | uniqueness check on blur |
| phone | 1 | ✓ | ✓ | uniqueness check on blur; E.164 format |
| password | 1 | ✓ | ✓ | 8–72 chars (matches existing policy) |
| confirm_password | 1 | ✓ | ✓ | client-only match check |
| role | 1 | pre-set (buyer) | pre-set (seller) | determined by entry point |
| email OTP | 2 | ✓ | ✓ | 6-digit, 10min expiry, resend w/ 60s cooldown |
| phone OTP | 2 | ✓ | ✓ | 6-digit, 10min expiry, resend w/ 60s cooldown |
| date_of_birth | 3 | ✓ | ✓ | must be ≥18 yrs ago; stored as DATE |
| gender | 3 | ✓ | ✓ | optional; male/female/non-binary/prefer-not-to-say |
| profile_picture | 3 | ✓ | ✓ | optional/skippable; JPEG/PNG ≤5MB |
| address (line1, city, state, country, pincode) | 3 | ✓ | ✓ | stored in existing `addresses` table |
| store_name | 3 | — | ✓ | required |
| business_type | 3 | — | ✓ | individual / registered_business |
| tax_id | 3 | — | ✓ | optional (GSTIN/PAN) |
| business_address | 3 | — | ✓ | checkbox "same as personal"; if unchecked, full address fields |
| payout_placeholder | 3 | — | ✓ | text only: "Payout setup available after approval" |
| terms_accepted | 4 | ✓ | ✓ | required checkbox |
| review summary | 4 | ✓ | ✓ | read-only display of all entered data |

---

## Data Model Changes

### `users` table — new columns (migration 0008)
```sql
ALTER TABLE users ADD COLUMN dob          DATE;
ALTER TABLE users ADD COLUMN gender        VARCHAR(20);  -- nullable
ALTER TABLE users ADD COLUMN pfp_url       TEXT;         -- nullable
ALTER TABLE users ADD COLUMN terms_accepted_at TIMESTAMPTZ; -- set at final submit
ALTER TABLE users ADD COLUMN phone_verified BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN registration_step SMALLINT NOT NULL DEFAULT 0;
-- 0 = fresh, 1 = basic info saved, 2 = both OTPs verified, 3 = profile saved, 4 = completed
```

### `seller_profiles` table — new columns (migration 0008)
```sql
ALTER TABLE seller_profiles ADD COLUMN business_type VARCHAR(30) NOT NULL DEFAULT 'individual';
ALTER TABLE seller_profiles ADD COLUMN tax_id        TEXT;       -- GSTIN or PAN, nullable
ALTER TABLE seller_profiles ADD COLUMN biz_line1     TEXT;
ALTER TABLE seller_profiles ADD COLUMN biz_city      VARCHAR(100);
ALTER TABLE seller_profiles ADD COLUMN biz_state     VARCHAR(100);
ALTER TABLE seller_profiles ADD COLUMN biz_country   CHAR(2);
ALTER TABLE seller_profiles ADD COLUMN biz_pincode   VARCHAR(20);
```

### `users` domain model — new fields
Add to `domain.User`: `DOB *time.Time`, `Gender *string`, `PfpURL *string`, `TermsAcceptedAt *time.Time`, `PhoneVerified bool`, `RegistrationStep int`.

---

## Validation Rules

| Rule | Source |
|------|--------|
| Password 8–72 chars | matches existing handlers.go policy |
| Email unique | `GetUserByEmail` — return 409 if found |
| Phone unique | `GetUserByPhone` (already exists in user_repository) |
| Age ≥ 18 | computed from dob at step 3 submit; error if `now - dob < 18 years` |
| Gender optional | skip allowed; stored NULL |
| PFP file type | JPEG/PNG only; max 5 MB; client-side + server-side |
| Terms required | boolean must be true on final submit |

---

## Reused Infrastructure

| Piece | Where it lives | Status |
|-------|---------------|--------|
| Email OTP send/verify | `auth_service.SendEmailOTP` / `VerifyEmailOTP` | ✓ exists |
| Phone OTP send | `auth_service.RequestPasswordResetOTP` pattern + `OTPPurposePhoneVerify` | partial — purpose constant exists, method needs adding |
| SMS sender | `internal/sms/smser.go` | ✓ exists |
| `addresses` table + `Address` domain type | `migrations/0001_init.up.sql` | ✓ exists (no repository yet) |
| Password hashing | `crypto.HashPassword` | ✓ exists |
| `buyer_token` cookie | `app/api/auth/register/route.ts` | ✓ exists |

---

## New Infrastructure Needed

| Item | Notes |
|------|-------|
| Address repository | `AddressRepository` with `Create` method; simple INSERT |
| `SendPhoneOTP` / `VerifyPhoneOTP` service methods | analogous to email OTP; uses `OTPPurposePhoneVerify` |
| PFP upload endpoint | `POST /v1/auth/profile/picture` — accepts multipart, stores file to disk or in DB as base64; **no MinIO/S3 in current infra** — propose: store as `/public/pfp/<uuid>.<ext>` served as static file from Next.js, base64 in DB is too large; flag as decision point |
| `registration_step` progression API | `PATCH /v1/auth/registration/step` — updates step + data for that step; replaces having one giant submit endpoint |

---

## API Endpoints

| Method | Path | Step | Purpose |
|--------|------|------|---------|
| `POST` | `/v1/auth/register` | — | Step 1: create partial user (step=1), return user_id + temp session token |
| `POST` | `/v1/auth/otp/send` | 2 | Existing; email OTP (already wired) |
| `POST` | `/v1/auth/otp/verify` | 2 | Existing; email OTP verify |
| `POST` | `/v1/auth/phone-otp/send` | 2 | New; phone OTP via SMS |
| `POST` | `/v1/auth/phone-otp/verify` | 2 | New; phone OTP verify |
| `PATCH` | `/v1/auth/registration/profile` | 3 | Save profile fields (dob, gender, address, seller fields) |
| `POST` | `/v1/auth/registration/complete` | 4 | Accept terms, finalize account (set registration_step=4, terms_accepted_at) |
| `POST` | `/v1/auth/profile/picture` | 3 | Upload PFP (multipart) |

Step 1 creates the user row with `registration_step=1`. The account is not usable until step 4 completes (login rejects `registration_step < 4` or returns a `registration_incomplete` error).

---

## Partial Registration Handling — Decision Point

**Proposal:** Create the user row at step 1 (after basic info), but block login until `registration_step = 4`. A `GET /v1/auth/registration/status` endpoint (or embed in login response) returns current step so the frontend can resume the wizard. Unfinished registrations older than 7 days are pruned by a background job (or a cron at DB level).

**Alternative:** Only create user at final submit, store wizard state entirely in client (localStorage + cookie). Simpler but loses OTP association server-side.

**Recommended:** Server-side partial row (proposal above).

---

## PFP Upload Decision Point

No file storage infra (MinIO/S3/disk) exists yet. Options:
1. **Skip for now** — make profile picture truly optional with no upload in this task; add a `pfp_url` column that can be set later via settings. ← **Recommended** (keeps scope tight)
2. **Local disk** — store under `services/auth-service/uploads/pfp/` served via a static handler; works for dev, not prod-ready.
3. **Base64 in DB** — only viable for small avatars (<100KB after resize); ugly but self-contained.

---

## Frontend Entry Points

- `/register` → role chooser → `/register/buyer` or `/register/seller`
- Both buyer/seller pages will be replaced by the new multi-step wizard (shared component, role passed as prop/context)
- Wizard state held in React `useReducer` + `sessionStorage` backup for reload resilience

---

## Seller-Only Required vs Optional (Decision Point)

| Field | Proposed |
|-------|---------|
| store_name | **required** |
| business_type | **required** (default: individual) |
| tax_id | **optional** (GSTIN or PAN — at least one, matches existing seller_profiles constraint) |
| business_address | **optional** (checkbox "same as personal address") |
