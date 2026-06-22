# product-catalog-service Review

## Executive Summary

The service is well-structured and reads cleanly. Layering is mostly correct (handler → service → repository via interfaces), repositories use parameterized SQL, pagination is bounded, the cache-aside layer is composed by interface, and the image upload path sniffs content type rather than trusting the client. The team has clearly internalized most of the engineering standards.

However, there is one **critical, exploitable authorization gap that blocks production**: no seller ownership check exists anywhere. Any authenticated seller can update, delete, or attach images to *any other seller's* product, SKU, or image. The `seller_id` is captured on create but never enforced on subsequent mutations. This is the central question the review brief asks ("can seller A edit seller B's product?") and the answer today is **yes**.

Secondary concerns: zero test coverage (no `*_test.go` files exist despite mockgen directives being wired up), a multi-currency model that stores `currency` per SKU but never validates it, image upload mutations that don't verify the image belongs to the product in the URL path, and a cache layer that can serve stale data and silently swallows Redis errors.

Verdict: **CHANGES REQUIRED.**

---

## Critical Findings

### C1. No seller ownership enforcement — horizontal privilege escalation
**Files:** `internal/handler/http/product_handler.go`, `sku_handler.go`, `product_image_handler.go`, `internal/service/*`, `main.go`

`RequireRole("seller", "admin")` gates the *mutation* routes, but role is the only check. Nothing ties the acting seller to the resource being mutated:

- `UpdateProduct` / `DeleteProduct` take only the product `id` from the URL. The update handler fetches the existing product (to preserve `seller_id`) but never compares `existingProduct.SellerID` against the caller's `user.Id`. Seller A can `PUT /api/v1/products/{B's id}` and overwrite seller B's product. `DeleteProduct` doesn't even load the product first.
- `CreateSKU` / `UpdateSKU` / `DeleteSKU` never load the parent product to check its owner. Any seller can mutate any SKU.
- `CreateProductImage` / `UpdateImagePosition` / `DeleteProductImage` never check that the product (or image) belongs to the caller.

This violates the engineering-standards "Always validate: Authorization" rule and is a textbook IDOR / horizontal-privilege-escalation bug. `CreateProduct` correctly derives `seller_id` from the token (good — it ignores any client-supplied seller), which makes the *absence* of the same rigor on update/delete more glaring.

**Fix:** Enforce ownership in the **service layer** (it must not live only in the handler, or the gRPC path / future callers bypass it). Thread the acting user identity + role into the use case (an explicit `actorID`/`actorRole` argument or a small auth-context value object), load the owning product, and reject with `403 FORBIDDEN` when `actorRole != "admin"` and `product.SellerID != actorID`. For SKUs and images, resolve ownership through the parent product. Add a `seller_id`-scoped `WHERE` to the update/delete SQL as defense in depth.

### C2. Image mutations are not scoped to the product in the route
**File:** `internal/handler/http/product_image_handler.go`

`UpdateImagePosition` and `DeleteProductImage` are routed under `/products/{product_id}/images/{id}` but ignore `product_id` entirely — they operate on `{id}` alone. Even once C1 is fixed, an image from product X can be manipulated via product Y's URL. Resolve the image, confirm `image.ProductID == product_id`, and confirm ownership of that product.

---

## High Priority Findings

### H1. Zero automated test coverage
No `*_test.go` files exist in the service. `//go:generate mockgen` directives are present on every service and the repository contracts, so the seams for testing are deliberately built — but nothing uses them. The engineering standards list Testability as a top-five priority and "Optimization comes after correctness," yet correctness is entirely unverified. At minimum, table-driven tests are needed for: the ownership checks (C1, once added), `BulkCreateCategories` topological sort (including the unresolvable-parent/cycle path), `capPageSize` / `validateSortField`, the cache-aside hit/miss/invalidation paths, and the image content-type sniffing.

### H2. Multi-currency `currency` is stored but never validated
**Files:** `internal/service/sku_service.go`, `internal/domain/models.go`, migration `0001_init.up.sql`

`CreateSKU` defaults `Currency` to `"INR"` when empty but accepts *any* string otherwise. The column is `CHAR(3)` with no `CHECK` constraint and no ISO-4217 validation in the service. A client can persist `"XXX"`, `"us"`, or junk. Given the brief calls out multi-currency as a new, load-bearing concern, this is a data-integrity hole — downstream FX conversion (`currency-service` / `fx-rates`) will choke on garbage codes. Validate against an allow-list (or call currency-service) and normalize to upper-case. `UpdateSKU` doesn't default currency at all, so an update with an empty currency would attempt to write `""` into a `CHAR(3) NOT NULL` column and fail at the DB rather than with a clean validation error.

### H3. Cache layer swallows Redis errors and can serve stale data
**File:** `internal/service/product_cache.go`

- Every `rdb.Set(...).Err()` and `rdb.Del(...).Err()` result is discarded with `_ =`, violating the "Never ignore errors" standard. A failed invalidation on `UpdateProduct` is invisible — stale data is served for the full TTL with no signal. At least log at `Warn`.
- `GetProductByID` populates *both* the id key and the slug key on a miss, but `UpdateProduct`/`DeleteProduct` invalidate using the slug from the passed struct. On a **slug change**, the stale `product:slug:<oldSlug>` entry is never invalidated and serves a stale product for 5 minutes.
- `DeleteProduct` recovers the slug via `p, _ := c.inner.GetProductByID(...)` — the error is dropped, so a fetch failure silently skips slug-key cleanup.

### H4. gRPC and public reads expose non-ACTIVE products
**Files:** `internal/handler/grpc/product_catalog_grpc_handler.go`, `internal/handler/http/product_handler.go`

`GetProduct`/`GetSKU`/`GetSKUsByProduct` (gRPC) have no auth and no status filtering. The HTTP `GetProductByID`/`GetProductBySlug` are fully public and also return `DRAFT`/`INACTIVE` products. A competitor can enumerate a seller's unreleased catalog by ID/slug. Confirm intent; if drafts must stay private, filter to `ACTIVE` for unauthenticated/cross-service reads (or require ownership for non-active).

---

## Medium Priority Findings

### M1. Extra read-after-write round trip on SKU create/update
**File:** `internal/handler/http/sku_handler.go`

`CreateSKU`/`UpdateSKU` call `GetSKUByID` immediately after the write to return DB-defaulted timestamps — an extra round trip per mutation, duplicated across both endpoints. Prefer `INSERT ... RETURNING` / `UPDATE ... RETURNING` and populate the struct in place.

### M2. Optimistic locking is inconsistent between product and SKU
**Files:** `internal/repository/product_repository.go`, `sku_repository.go`

`UpdateProduct` uses `WHERE ... AND updated_at = $8` for optimistic concurrency and maps 0-rows to `409`. `UpdateSku` has no such guard — concurrent SKU updates are last-write-wins silently. Apply the same pattern to SKUs or document why not. The product "merge-from-existing" PUT also treats a field set to its zero value (e.g. clearing a description with `""`) as "not provided"; acceptable for PUT-as-PATCH but should be documented.

### M3. Pagination cap is split across two places and the echoed limit is wrong
**Files:** `internal/handler/http/base.go`, `internal/service/list_validation.go`

The handler computes `limit` via `GetLimitOffset` (no max), then the service re-caps via `capPageSize`. `httpx.Paginated` echoes the *handler's* uncapped limit while the DB used the capped one — `limit=10000` returns 100 rows but reports `limit=10000`. Single source of truth; surface the effective limit.

### M4. Image upload multi-reader reconstruction is subtle and untested
**Files:** `internal/handler/http/product_image_handler.go`, `internal/service/product_image_service.go`

The sniff-buffer + `io.MultiReader` reconstruction and `formFileSize` header read are correct but fragile enough to warrant a test (see H1). The service's orphan-cleanup-on-DB-failure is good. No magic-byte vs. extension cross-check beyond `DetectContentType` — acceptable.

### M5. `uniqueStrings` mutates its input slice via `ss[:0]`
**File:** `internal/service/category_service.go`

`out := ss[:0]` reuses the caller's backing array. Safe here (locally-built input) but a latent aliasing bug if reused on a caller-owned slice. Allocate fresh unless the in-place optimization is genuinely needed.

---

## Low Priority Findings

- **L1.** `domainProductToProto`/`domainSKUToProto` use `time.Time.String()` for timestamps — non-RFC3339, hard to parse. Use `time.RFC3339` or a proto `Timestamp`.
- **L2.** Many distinct validation failures share one code `"INVALID_DATA"`; clients can't distinguish "name required" from "category required." Consider field-specific codes.
- **L3.** `DecodeJSON` uses a bare decoder with no `DisallowUnknownFields` and no body-size limit on JSON endpoints (only the image endpoint caps body size).
- **L4.** Observability standard requires `trace_id`/`request_id`/`correlation_id` in logs; `chimiddleware.RequestID` sets it on context but the `slog` calls don't include it, so logs can't be correlated.
- **L5.** Every read logs at `Info` (`"fetching product by id"`, etc.) — noisy in production; prefer `Debug`.
- **L6.** `attributesToRawMessage` accepts arbitrary, unbounded JSON stored verbatim into the GIN-indexed JSONB column.
- **L7.** Naming inconsistency: `SkuRepository`/`CreateSku` vs. domain `SKU` / service `CreateSKU`. Standardize the `SKU` initialism casing.

---

## Recommended Refactoring Plan

1. **(Critical, first)** Introduce service-layer ownership enforcement. Add an actor abstraction (`actorID`, `role`) to `UpdateProduct`, `DeleteProduct`, and all SKU/image mutations. Load the owning product, reject non-owner non-admin with `403`. Add `seller_id` to update/delete `WHERE` clauses as defense in depth. Fix image routes to validate `product_id` matches the image. (C1, C2)
2. **(Critical)** Backfill tests for the new ownership rules before merging — they guard the regression you just fixed. (H1)
3. **(High)** Validate/normalize currency in `CreateSKU`/`UpdateSKU`, default it on update, add a DB `CHECK` or FK. (H2)
4. **(High)** Log Redis errors, fix slug-key invalidation on slug change, stop dropping the `DeleteProduct` lookup error. (H3)
5. **(High)** Decide and document non-`ACTIVE` product visibility for gRPC/public reads; filter if drafts must be private. (H4)
6. **(Medium)** Consolidate pagination capping; switch SKU create/update to `RETURNING`; add optimistic locking to SKU update. (M1–M3)
7. **(Low)** RFC3339 timestamps, field-specific error codes, JSON body limits, request-id in structured logs.

---

## Final Verdict

**CHANGES REQUIRED.**

The code is clean and the architecture is sound, but the missing seller ownership checks (C1/C2) are a directly exploitable horizontal-privilege-escalation vulnerability that fails the explicit authorization requirement in the engineering standards. Combined with zero test coverage and unvalidated multi-currency input, this is not yet production-ready. Address C1, C2, H1, and H2 at minimum before re-review.
