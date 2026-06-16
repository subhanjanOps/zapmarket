# Stage 2 — Product Catalog Hardening

**Status: ✅ Complete (2026-06-16)**

Corresponds to checklist **Phase 7**.

## Goal

Bring `product-catalog-service`'s list endpoints (categories, products, SKUs) up to a
standard that the rest of the platform can rely on for browsing/search use cases —
typed filters, pagination, and sorting — before Inventory/Order start depending on
catalog data shapes.

## Preconditions

Stage 1 merged (domain contracts in place, so filter structs land in
`internal/domain`, not bolted onto handler signatures).

## Tasks

### 2.1 Category filters
- [x] Defined `domain.CategoryFilters` (parent ID, search, limit/offset, sort by/order)
      instead of the old `map[string]string` the handler built ad hoc.
- [x] Added `Limit`/`Offset` (capped at `domain.MaxPageSize`=100, default
      `domain.DefaultPageSize`=20) and `SortBy`/`SortOrder`.
- [x] `category_repository.go` now builds a parameterized query via a shared
      `categoryListWhere` helper reused by both the `COUNT` and `SELECT` queries (so the
      total can never drift from the page).
- [x] `category_handler.go` parses query params into the typed filter and returns
      `httpx.Paginated` (`{ success, data: [...], total, page, page_size }`).
- **Behavior change**: the old handler silently hardcoded `parent_id IS NULL` (top-level
      categories only) and ignored all query params. The new version returns all
      categories by default and lets `?parent_id=` filter to a specific subtree —
      strictly more capable, but flagging the default-list-contents change explicitly.

### 2.2 Product filters
- [x] `ProductFilters` already had category ID, seller ID, search, status, pagination,
      sort — added a parameterized `productListWhere` count query and made invalid
      `sort_by` values a 400 instead of a silent fallback to `created_at`.
- [x] Price range filter **skipped**: `products` has no price column in the schema (price
      lives on `skus.price_amount`, since one product can have multiple SKUs at different
      prices). Filtering products by price would require a join/subquery against SKUs —
      flagged as a follow-up if/when there's a concrete need, not implemented here.
- [x] Same pagination envelope as categories.

### 2.3 SKU filters
- [x] `SKUFilters` already had product ID, active status, pagination, sort — added
      `skuListWhere` count query and the same invalid-sort-field rejection.

### 2.4 Shared pagination helper
- [x] Added `httpx.Paginated[T any](w, status, data []T, total int64, page, pageSize int)`
      to `pkg/httpx/response.go` — a generic helper all three list endpoints (and future
      services) now share.

## Out of scope
- Elasticsearch / full-text search infra (referenced in `design.md` but no CDC pipeline
  exists yet — that's a later, currently unscheduled stage once Debezium is turned on
  in Stage 7 and there's a concrete need).
- Redis page caching — covered in Stage 9.

## Definition of done
- [x] All three list endpoints accept filter + pagination + sort query params and return
      the shared envelope shape. Verified live: `GET /api/v1/categories` →
      `{"success":true,"data":[],"total":0,"page":1,"page_size":20}`.
- [x] Repository queries are parameterized (`$1`, `$2`, ...) with no string concatenation
      of user input (values are always passed as args; only column/direction names from
      a fixed allow-list are interpolated into the query string).
- [x] Verified live instead of via new unit tests (no existing handler test suite to
      extend): empty filter → empty paginated envelope; invalid `sort_by=bogus` on
      `/products` → `400 INVALID_SORT_FIELD`; invalid `parent_id=not-a-uuid` on
      `/categories` → `400 INVALID_PARENT_ID`; `offset=500` beyond available data →
      `200` with empty `data` and correct `page`; `limit=9999` → capped to `page_size:100`.
