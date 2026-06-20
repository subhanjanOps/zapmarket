# Code Review: `product-catalog-service`

**Reviewer:** Principal Engineer
**Date:** 2026-06-21
**Branch:** `features/cluster-setup`
**Verdict:** CHANGES REQUIRED

---

## Executive Summary

Well-structured service with clean layer separation, consistent dependency injection, and a solid feature set (products, SKUs, categories with topological sort, product images). However: zero tests, seller authorization bypass on write operations, committed `.env`, a cache race condition, and a gRPC connection leak block production approval.

---

## Critical Findings

### C-1: Zero Test Coverage
No `*_test.go` files exist in the entire service. `//go:generate mockgen` directives are present but never executed. Complex logic (topological sort, cache invalidation) is completely untested.

### C-2: Seller Authorization Not Enforced on Write Operations
**Files:** `internal/handler/http/product_handler.go:255-344`, `internal/handler/http/sku_handler.go:198-268`
`UpdateProduct`, `DeleteProduct`, `UpdateSKU`, `DeleteSKU` all gate on role (`seller`/`admin`) but never verify the requesting seller owns the resource. Seller A can modify or delete Seller B's products.

**Fix:** After fetching the existing resource, compare `existingProduct.SellerID` against the authenticated user's ID; return HTTP 403 on mismatch.

### C-3: Committed `.env` File With Real-Looking Credentials
**File:** `services/product-catalog-service/.env`
Contains MinIO credentials (`minioadmin`/`minioadmin`), JWT secret, and DB credentials. Must be removed from git history and `.gitignore`d.

### C-4: Race Condition in `DeleteProduct` Cache Decorator
**File:** `internal/service/product_cache.go:119-131`
`GetProductByID` and `DeleteProduct` are two separate calls with no transaction. A concurrent slug update between them leaves a stale slug key in Redis indefinitely. Error from `GetProductByID` is also silently swallowed (`p, _ :=`).

---

## High Priority Findings

### H-1: `authctx` Depends on Proto-Generated Type — Dependency Inversion Violation
**File:** `internal/authctx/authctx.go:12-13`
Imports and re-exports `*authpb.User`. Should use a domain `AuthUser` struct; middleware translates at the boundary.

### H-2: `internal/errors/category.go` Is Entirely Dead Code
Defines 5 error constructors; none are called anywhere. All error construction happens inline via `pkgerrors`.

### H-3: `UpdateProduct` Handler Contains Business Logic (Fetch-and-Merge)
**File:** `internal/handler/http/product_handler.go:269-316`
Merge pattern (which fields are optional, how to handle partial updates) lives in the interface layer. Must move to a service-layer use case.

### H-4: Extra DB Round-Trip After Create/Update SKU
**File:** `internal/handler/http/sku_handler.go:91-97, 234-238`
After `CreateSKU`/`UpdateSKU` a `GetSKUByID` is issued unnecessarily. Repository should `RETURNING` the row.

### H-5: `GetSKUByID` Has No Nil UUID Guard
**File:** `internal/service/sku_service.go:72-76`
Unlike `GetProductByID` and `GetCategoryByID`, no `uuid.Nil` check. Inconsistent contract.

### H-6: `BulkCreateCategories` Has No Body Size Limit
**File:** `internal/handler/http/category_handler.go:100-124`
No `http.MaxBytesReader` before JSON parse. Client can send unbounded payload.

### H-7: gRPC Connection Leaked in Auth Middleware
**File:** `internal/middleware/auth.go:23-31`
`grpc.ClientConn` is never stored on `AuthMiddleware` — no way to close it during graceful shutdown.

---

## Medium Priority Findings

- **M-1:** Error code inconsistency — `"DATABASE_ERROR"` vs `"INTERNAL_SERVER_ERROR"` across repositories
- **M-2:** `sort_by` validated in both service AND repository — redundant fallback in repo silently masks bugs
- **M-3:** Cache key uses MD5 (`crypto/md5`) — triggers security scanners; `json.Marshal` error silently discarded causing all lists to share one cache slot on failure
- **M-4:** `GET /products/{id}` and `/slug/{slug}` don't filter DRAFT products — buyers can access drafts
- **M-5:** `BulkCreateCategories` has no atomicity across waves — partial write on wave failure with no rollback
- **M-6:** `DecodeJSON` has no body size limit — affects all JSON endpoints
- **M-7:** Layer naming deviates from `architecture-principles.md` (`service/` should be `application/`, etc.)
- **M-8:** `GetImageByProductID` and `GetImageBySKUID` omit `object_key` from SELECT — latent data integrity bug

---

## Low Priority Findings

- **L-1:** `GetLimitOffset` silently falls back on parse errors — should return 400
- **L-2:** `uniqueStrings` utility lives in `category_service.go` — wrong home
- **L-3:** `SKUId` field should be `SKUID` per Go convention
- **L-4:** Service discovery hostname hardcoded (`zapmarket-product-catalog-service`) in `main.go:211`
- **L-5:** `UploadProductImage` log says "uploading" but fires after the upload completes
- **L-6:** `CreateProductImage` auto-assigned `position` never returned — response always shows `position: 0`

---

## Recommended Refactoring Plan

**Sprint 1 — Blockers:**
1. Remove `.env` from git, rotate credentials
2. Add seller ownership check in all mutating handlers
3. Fix `DeleteProduct` cache race and swallowed error
4. Store and `Close()` gRPC connection in `AuthMiddleware`

**Sprint 2 — High Priority:**
5. Run `mockgen` and write unit tests (topological sort, cache, service validation paths)
6. Move fetch-and-merge logic from handler to service layer
7. Delete `internal/errors/category.go` or use it consistently
8. Apply `io.LimitReader` in `DecodeJSON`; add body limit to bulk endpoint

**Sprint 3 — Medium Priority:**
9. Add `object_key` to `GetImageByProductID`/`GetImageBySKUID`
10. Return `position` from `CreateProductImage` via `RETURNING`
11. Align layer naming with architecture principles
12. Standardize error codes across repositories
13. Replace MD5 with SHA-256; handle `json.Marshal` error in cache key
14. Filter DRAFT products on public endpoints

**Sprint 4 — Housekeeping:**
15. Fix `SKUId` → `SKUID`
16. Remove redundant sort validation from repositories
17. Externalize service discovery hostname

---

## Final Verdict: CHANGES REQUIRED
