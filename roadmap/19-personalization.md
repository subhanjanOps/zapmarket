# Plan 19 — Personalization and Recommendations

**Effort:** L | **Impact:** M | **Depends on:** Plans 16, 11

## Context
Zero personalization layer exists. No user event store, no browse history, no recommendation engine. Amazon's primary retention driver is "For You" recommendations — relevant but expensive to build correctly. This plan does a pragmatic V1 using simple collaborative signals, not ML.

## Scope
- User event tracking (view, add-to-cart, purchase)
- "Customers also bought" via co-purchase graph (PostgreSQL-based, no ML infra)
- "Based on your browsing" — top-N from browsed categories weighted by recency

## Out of scope
- ML model training
- Real-time feature store
- A/B testing framework

## Tasks

### User event store
- [ ] Create new `analytics-service` (Go) OR add `user_events` table to product-catalog-service DB
  - Recommended: separate `analytics-service` to avoid coupling; minimal Go module, single table
- [ ] `user_events` table: (`id UUID PK`, `user_id UUID`, `event_type TEXT CHECK IN ('view','cart_add','purchase','search')`, `product_id UUID`, `category_id UUID`, `session_id TEXT`, `metadata JSONB`, `occurred_at TIMESTAMPTZ`)
- [ ] `POST /v1/events` endpoint (fire-and-forget, async): accept event from buyer-ui, write to table
- [ ] buyer-ui: on product page mount, `POST /v1/events {type: "view", product_id}` (non-blocking, best-effort)
- [ ] On cart add: `POST /v1/events {type: "cart_add", product_id}`

### Co-purchase recommendations ("customers also bought")
- [ ] Populate `product_cooccurrences` table: (`product_a UUID`, `product_b UUID`, `count INT`) — updated by a nightly job that scans `orders` and counts products bought together in same order
- [ ] `GET /v1/products/{id}/recommendations`: return top 8 co-occurring products ordered by `count DESC`
- [ ] Wire in buyer-ui product detail page below the fold

### Category-based "For You" 
- [ ] `GET /v1/users/{id}/recommendations`: query user's top 3 browsed `category_id`s from `user_events` in last 30 days, return trending products from those categories (uses Plan 16's trending endpoint per category)
- [ ] Wire in buyer-ui homepage below hero

## Done criteria
- Product view events written to `user_events` on page load (verify in DB)
- "Customers also bought" section shows on product detail page with real co-purchase data (after nightly job runs)
- Homepage "For You" section shows category-relevant products for logged-in buyers
- Anonymous users see trending products instead (graceful fallback)
