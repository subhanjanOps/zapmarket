# Stage 2 — Product Catalog Hardening

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
- [ ] Define a typed `CategoryFilter` struct in `internal/domain` (parent ID, active-only,
      name search) instead of reading raw query params inside the handler.
- [ ] Add `Page`, `PageSize` (with sane max, e.g. 100) and `SortBy`/`SortDir` to the filter.
- [ ] Update `category_repository.go` to build a parameterized SQL query from the filter
      (never string-concatenate user input into SQL).
- [ ] Update `category_handler.go` to parse query params into the typed filter and return
      a paginated envelope (`{ data: [...], total: N, page: N, page_size: N }`).

### 2.2 Product filters
- [ ] `ProductFilter`: category ID, seller ID, free-text search (`name ILIKE`), price range,
      active-only, pagination, sorting (`created_at`, `price`, `name`).
- [ ] Same pagination envelope as categories — keep the response shape consistent across
      all three list endpoints so frontend/gateway clients have one pattern to handle.

### 2.3 SKU filters
- [ ] `SKUFilter`: product ID, active-only, pagination, sorting.

### 2.4 Shared pagination helper
- [ ] Since this is the second/third/fourth time writing the same pagination envelope,
      add a small generic helper in `pkg/httpx` (e.g. `httpx.Paginated[T]`) so future
      services (Stage 3+) don't reinvent it.

## Out of scope
- Elasticsearch / full-text search infra (referenced in `design.md` but no CDC pipeline
  exists yet — that's a later, currently unscheduled stage once Debezium is turned on
  in Stage 7 and there's a concrete need).
- Redis page caching — covered in Stage 9.

## Definition of done
- All three list endpoints accept filter + pagination + sort query params and return the
  shared envelope shape.
- Repository queries are parameterized (`$1`, `$2`, ...) with no string concatenation of
  user input.
- Existing handler tests (or new ones, if none exist) cover at least: empty filter,
  combined filters, page beyond available data, invalid sort field (should reject with
  `errors.NewValidation`, not panic).
