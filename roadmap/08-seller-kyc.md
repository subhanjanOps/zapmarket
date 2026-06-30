# Plan 08 — Seller KYC and Approval Workflow

**Effort:** S | **Impact:** M | **Depends on:** Plan 07

## Context
`seller_profiles` table stores GSTIN and PAN but has no approval status. Sellers are activated immediately on registration. Admin has no way to approve or reject a seller. Seller-ui has only `login/` and `register/` routes — no onboarding wizard.

## Scope
- Add approval status to seller_profiles
- Add admin approve/reject endpoints
- Add onboarding step tracking so seller-ui can show progress

## Tasks
- [ ] Migration: add `approval_status TEXT NOT NULL DEFAULT 'PENDING' CHECK IN ('PENDING','APPROVED','REJECTED','SUSPENDED')`, `approved_at TIMESTAMPTZ`, `rejected_at TIMESTAMPTZ`, `rejection_reason TEXT`, `reviewed_by UUID` to `seller_profiles`
- [ ] Block seller product listing if `approval_status != 'APPROVED'`: add check in product-catalog-service `CreateProduct` handler (call auth-service gRPC `GetSellerProfile` or embed status in JWT claim)
- [ ] Add `PUT /v1/admin/sellers/{id}/approve` and `PUT /v1/admin/sellers/{id}/reject` endpoints in auth-service
- [ ] Add `RequireRole("admin")` middleware to both endpoints
- [ ] Emit `seller.approved` / `seller.rejected` Kafka event so notification-service can send email/SMS
- [ ] Add onboarding steps enum in seller_profiles: `onboarding_steps_completed JSONB DEFAULT '{}'` — keys: `profile`, `gstin_pan`, `bank_account`, `approved`
- [ ] Update each step's completion via respective service (bank account added in Plan 07 → mark `bank_account` step done)

## Done criteria
- New seller starts in `PENDING` state and cannot list products
- Admin can approve/reject via API; seller notified via email/SMS
- `APPROVED` seller can list products
- Onboarding steps trackable via API
