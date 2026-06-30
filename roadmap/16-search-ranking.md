# Plan 16 — Search Ranking and Discovery

**Effort:** S | **Impact:** M | **Depends on:** Plan 11 (rating aggregation)

## Context
Typesense search, faceting, and autocomplete are implemented. Missing:
- No result ranking signals (sales rank, rating, seller tier)
- Trending/curated homepage sections have UI components but no backend data source
- `CountdownTimer` component exists but no backend-driven flash sale data
- No recently viewed (requires user event tracking)

## Scope
- Add boosting to Typesense search adapter (rating + sales rank)
- Add trending products endpoint (based on recent order counts)
- Wire flash sale data from promotions-service to buyer-ui CountdownTimer
- Recently viewed via localStorage (no backend needed for V1)

## Out of scope
- ML-based personalization (Plan 19)
- Collaborative filtering ("customers also bought")

## Tasks

### Typesense boosting (product-catalog-service)
- [ ] Add `sales_rank INT` and `avg_rating NUMERIC(3,2)` to Typesense product schema (sync worker already exists — extend it)
- [ ] Populate `sales_rank` from order-management-service: consume `order.confirmed` event → increment product sales counter in product-catalog DB → sync to Typesense on next sync cycle
- [ ] Populate `avg_rating` from `product_ratings` materialized view (Plan 11) on sync
- [ ] In `search_handler.go`: add Typesense `sort_by` clause `_text_match:desc,avg_rating:desc,sales_rank:asc` (sales_rank ascending = more sales = lower rank number)

### Trending products endpoint
- [ ] In product-catalog-service, add `GET /v1/products/trending?limit=20`: query products ordered by `sales_rank ASC` where `sales_rank IS NOT NULL`, limit 20
- [ ] Cache response in Redis with 15-minute TTL (avoids DB hit on every homepage load)
- [ ] Wire in buyer-ui homepage: replace static placeholder in hero/featured sections with this endpoint

### Flash sales from promotions-service
- [ ] Add `starts_at TIMESTAMPTZ`, `ends_at TIMESTAMPTZ` to promotions `coupons` table (migration)
- [ ] Add `GET /v1/promotions/active-sales` endpoint: return coupons with `starts_at <= now() <= ends_at` and `discount_type = 'PERCENT'`
- [ ] Wire buyer-ui `CountdownTimer` to consume `ends_at` from this endpoint

### Recently viewed (client-side V1)
- [ ] In buyer-ui product detail page: on mount, push `{product_id, name, image, timestamp}` to `localStorage['recently_viewed']` (max 10 items, FIFO)
- [ ] Add `RecentlyViewed` component on homepage reading from localStorage

## Done criteria
- Search results with higher avg_rating rank above lower-rated products (all else equal)
- Homepage trending section shows real top-selling products
- Active flash sales render with correct countdown timer
- Recently viewed products persist across sessions in localStorage
