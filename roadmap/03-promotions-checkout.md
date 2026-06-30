# Plan 03 — Wire Promotions into Checkout

**Effort:** S | **Impact:** H | **Depends on:** Plan 01

## Context
`promotions-service` has a complete `ValidateCouponUseCase` with PERCENT and FIXED_PAISE types, per-user and total usage limits. It is never called during order creation. The `orders` and `payments` tables have no `discount_amount` column.

## Scope
- Add `coupon_code` and `discount_paise` columns to `orders` table
- Call promotions-service from order-management-service during order creation (gRPC or HTTP)
- Deduct discount from payment charge amount
- Increment coupon usage count after successful charge

## Out of scope
- Coupon UI on buyer-ui (separate frontend task)
- Admin coupon management UI

## Tasks
- [ ] Add migration: `ALTER TABLE orders ADD COLUMN coupon_code TEXT, ADD COLUMN discount_paise BIGINT NOT NULL DEFAULT 0`
- [ ] Add migration: `ALTER TABLE payments ADD COLUMN discount_paise BIGINT NOT NULL DEFAULT 0`
- [ ] Decide transport: HTTP call from order-management-service to promotions-service (simpler; gRPC adds proto complexity)
- [ ] Add `promotions_service_url` env var to order-management-service config
- [ ] In order creation handler: if `coupon_code` present in request body, call `POST /v1/coupons/validate` on promotions-service before writing order
- [ ] On validation success: store `coupon_code` + `discount_paise` on order; pass reduced `amount_paise` to payment saga
- [ ] On payment capture success: call `POST /v1/coupons/{code}/redeem` on promotions-service to increment usage
- [ ] On payment failure: do NOT increment usage (no compensation needed — usage was never incremented)
- [ ] Handle promotions-service unavailability: fail open (allow order without coupon) or fail closed based on product decision; default to fail open with coupon ignored

## Done criteria
- Order created with valid coupon has `discount_paise > 0` and correct reduced payment amount
- Coupon `usage_count` increments exactly once per successful order
- Invalid/expired coupon returns 422 from order creation endpoint
- Promotions-service downtime does not block order creation
