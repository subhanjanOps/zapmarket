# Plan 11 — Reviews and Returns

**Effort:** M | **Impact:** M | **Depends on:** Plans 07, 10

## Context
review-return-service has:
- Reviews: moderation and verified purchase implemented ✅; rating aggregation missing ❌
- Returns: single-item only, no multi-item, no image evidence, no approval workflow wired, no seller debit

## Scope
- Wire return approval state machine
- Add return image evidence
- Add multi-item return support
- Add rating aggregation materialized view
- Debit seller on return approval (calls settlement-service)

## Tasks

### Return improvements
- [ ] Migration: add `image_urls TEXT[]` to `return_requests`
- [ ] Migration: add `return_items` table (`id UUID PK`, `return_request_id UUID FK`, `order_item_id UUID`, `quantity INT`, `reason TEXT`) — replaces single `order_item_id` on `return_requests`
- [ ] Add `APPROVED`, `REJECTED`, `PICKUP_SCHEDULED`, `COMPLETED` to return_request status enum
- [ ] Implement `ApproveReturn(id)` use case: set status `APPROVED`, call logistics-service `POST /v1/return-shipments` (Plan 10), emit `return.approved` event
- [ ] Implement `RejectReturn(id)` use case: set status `REJECTED`, notify buyer
- [ ] Add `PUT /v1/returns/{id}/approve` and `PUT /v1/returns/{id}/reject` (admin/seller endpoints with `RequireRole`)
- [ ] On `return.completed` event (reverse pickup delivered): call settlement-service to write `DEBIT_RETURN` ledger entry against seller

### Rating aggregation
- [ ] Migration: create materialized view `product_ratings` (`product_id UUID`, `avg_rating NUMERIC(3,2)`, `review_count INT`) — `SELECT product_id, AVG(rating), COUNT(*) FROM reviews WHERE status = 'APPROVED' GROUP BY product_id`
- [ ] Add `REFRESH MATERIALIZED VIEW CONCURRENTLY product_ratings` on schedule (every 15 min via a Go ticker in review-return-service, or via pg_cron)
- [ ] Add `GET /v1/products/{id}/rating` endpoint serving from materialized view
- [ ] product-catalog-service: include rating from this endpoint (or denormalize into products table via event) on product detail

## Done criteria
- Admin can approve/reject returns via API
- Approved return triggers reverse pickup (Plan 10)
- Seller debited on return completion
- `product_ratings` materialized view queryable and refreshed automatically
- Return request accepts multiple items and image URLs
